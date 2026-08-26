// Package diagnosis 负责截断诊断：定位断裂边并给出修复建议，
// 以及处理工程师对断裂边的人工裁决（确认/豁免/修订）。
package diagnosis

import (
	"fmt"

	"task249-linagediag/internal/model"
	"task249-linagediag/internal/store"
	"task249-linagediag/internal/transform"
)

// Service 诊断服务，组合 store 与 transform 推断能力。
type Service struct {
	store     *store.Store
	transform *transform.Service
}

// NewService 构造诊断服务。
func NewService(st *store.Store, ts *transform.Service) *Service {
	return &Service{store: st, transform: ts}
}

// Locate 对批次做截断诊断：遍历血缘边，找出断裂边，并对每条断裂边基于重命名推断给出修复建议。
func (svc *Service) Locate(batchID int64) ([]model.DiagnosisFinding, error) {
	edges, err := svc.store.ListEdges(batchID)
	if err != nil {
		return nil, err
	}
	var findings []model.DiagnosisFinding
	for _, e := range edges {
		if e.Status != model.EdgeBroken {
			continue
		}
		t, err := svc.store.GetTransform(e.TransformID)
		if err != nil {
			return nil, err
		}
		f := model.DiagnosisFinding{
			EdgeID:       e.ID,
			TargetTable:  t.TargetTable,
			TargetColumn: t.TargetColumn,
			SourceTable:  t.SourceTable,
			SourceColumn: t.SourceColumn,
			Broken:       true,
			Reason:       e.Reason,
		}
		cands, cerr := svc.transform.InferRenameCandidates(batchID, t.SourceTable, t.SourceColumn)
		if cerr == nil && len(cands) > 0 {
			best := cands[0]
			f.SuggestedFix = fmt.Sprintf(
				"上游列 %s.%s 可能已重命名为 %s.%s（置信度 %.2f），请补充映射变换后重建血缘",
				t.SourceTable, t.SourceColumn, best.Table, best.NewName, best.Confidence)
		} else {
			f.SuggestedFix = fmt.Sprintf(
				"上游列 %s.%s 在当前表版本中不存在且未找到重命名候选，请确认是否缺失映射或已被删除",
				t.SourceTable, t.SourceColumn)
		}
		findings = append(findings, f)
	}
	return findings, nil
}

// BrokenCount 返回批次中断裂边的数量。
func (svc *Service) BrokenCount(batchID int64) (int, error) {
	edges, err := svc.store.ListEdges(batchID)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, e := range edges {
		if e.Status == model.EdgeBroken {
			n++
		}
	}
	return n, nil
}
