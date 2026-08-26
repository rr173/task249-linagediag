package service

import (
	"task249-linagediag/internal/model"
)

// LineageEdge 面向 HTTP / 自测的血缘边视图（带上表限定名与列名）。
type LineageEdge struct {
	EdgeID       int64  `json:"edge_id"`
	SourceTable  string `json:"source_table"`
	SourceColumn string `json:"source_column"`
	TargetTable  string `json:"target_table"`
	TargetColumn string `json:"target_column"`
	Status       string `json:"status"`
	Reason       string `json:"reason"`
}

// LineageView 返回某批次的血缘视图：非断裂边用列 ID 解析名称；
// 断裂边用其底层 transform 记录的源/目标表列名。
func (s *Service) LineageView(batchID int64) ([]LineageEdge, error) {
	edges, err := s.Store.ListEdges(batchID)
	if err != nil {
		return nil, err
	}
	out := make([]LineageEdge, 0, len(edges))
	for _, e := range edges {
		le := LineageEdge{
			EdgeID: e.ID,
			Status: string(e.Status),
			Reason: e.Reason,
		}
		if e.SourceColID != 0 && e.TargetColID != 0 {
			src, err := s.Store.GetColumn(e.SourceColID)
			if err != nil {
				return nil, err
			}
			dst, err := s.Store.GetColumn(e.TargetColID)
			if err != nil {
				return nil, err
			}
			sTbl, err := s.Store.GetTable(src.TableID)
			if err != nil {
				return nil, err
			}
			dTbl, err := s.Store.GetTable(dst.TableID)
			if err != nil {
				return nil, err
			}
			le.SourceTable, le.SourceColumn = sTbl.QualifiedName, src.Name
			le.TargetTable, le.TargetColumn = dTbl.QualifiedName, dst.Name
		} else {
			t, err := s.Store.GetTransform(e.TransformID)
			if err != nil {
				return nil, err
			}
			le.SourceTable, le.SourceColumn = t.SourceTable, t.SourceColumn
			le.TargetTable, le.TargetColumn = t.TargetTable, t.TargetColumn
		}
		out = append(out, le)
	}
	return out, nil
}

// Scenario 端到端场景：在已有批次上重放「上游列重命名 → 下游派生列未更新 → 定位 → 修订 → 发布」闭环。
// 供 smoke 自测与 HTTP 自检复用，返回关键中间量用于断言。
type ScenarioResult struct {
	BatchID      int64                  `json:"batch_id"`
	BrokenBefore int                    `json:"broken_before"`
	Findings     []model.DiagnosisFinding `json:"findings"`
	BrokenAfter  int                    `json:"broken_after"`
	SnapshotVer  int                    `json:"snapshot_version"`
	Sealed       bool                   `json:"sealed"`
}

// RunScenario 在给定批次上执行修复闭环（假设批次已导入元数据、变换、作业并 BuildLineage）。
func (s *Service) RunScenario(batchID int64) (*ScenarioResult, error) {
	before, err := s.Diagnosis.BrokenCount(batchID)
	if err != nil {
		return nil, err
	}
	findings, err := s.Diagnose(batchID)
	if err != nil {
		return nil, err
	}
	// 对每条断裂边，按推断候选修订上游列。
	edges, err := s.Store.ListEdges(batchID)
	if err != nil {
		return nil, err
	}
	for _, e := range edges {
		if e.Status != model.EdgeBroken {
			continue
		}
		t, err := s.Store.GetTransform(e.TransformID)
		if err != nil {
			return nil, err
		}
		cands, cerr := s.Transform.InferRenameCandidates(batchID, t.SourceTable, t.SourceColumn)
		if cerr == nil && len(cands) > 0 {
			if err := s.Revise(batchID, e.ID, t.SourceTable, cands[0].NewName, "auto-revise by rename inference"); err != nil {
				return nil, err
			}
		} else {
			if err := s.Exempt(batchID, e.ID, "no rename candidate, exempt"); err != nil {
				return nil, err
			}
		}
	}
	after, err := s.Diagnosis.BrokenCount(batchID)
	if err != nil {
		return nil, err
	}
	snap, err := s.PublishSnapshot(batchID, "scenario publish")
	if err != nil {
		return nil, err
	}
	sealed := false
	if after == 0 {
		if err := s.Seal(batchID); err == nil {
			sealed = true
		}
	}
	return &ScenarioResult{
		BatchID:      batchID,
		BrokenBefore: before,
		Findings:     findings,
		BrokenAfter:  after,
		SnapshotVer:  snap.Version,
		Sealed:       sealed,
	}, nil
}
