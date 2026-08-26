package diagnosis

import (
	"fmt"

	"task249-linagediag/internal/model"
)

// Confirm 确认一条边（工程师确认映射正确）：置为 confirmed 并记录裁决。
func (svc *Service) Confirm(batchID, edgeID int64, note string) error {
	if err := svc.assertBatchMutable(batchID); err != nil {
		return err
	}
	if err := svc.assertEdgeInBatch(batchID, edgeID); err != nil {
		return err
	}
	if err := svc.store.SetEdgeStatus(edgeID, model.EdgeConfirmed, "confirmed by engineer"); err != nil {
		return err
	}
	_, err := svc.store.CreateAdjudication(batchID, edgeID, "confirm", note)
	return err
}

// Exempt 豁免一条断裂边（确认无需映射）：置为 exempt 并记录裁决。
func (svc *Service) Exempt(batchID, edgeID int64, note string) error {
	if err := svc.assertBatchMutable(batchID); err != nil {
		return err
	}
	if err := svc.assertEdgeInBatch(batchID, edgeID); err != nil {
		return err
	}
	if err := svc.store.SetEdgeStatus(edgeID, model.EdgeExempt, "exempted by engineer"); err != nil {
		return err
	}
	_, err := svc.store.CreateAdjudication(batchID, edgeID, "exempt", note)
	return err
}

// Revise 修订：为断裂边补全上游列（指向候选重命名列），重建血缘并确认。
// newSourceTable/newSourceColumn 为补全后的正确上游引用。
func (svc *Service) Revise(batchID, edgeID int64, newSourceTable, newSourceColumn, note string) error {
	if err := svc.assertBatchMutable(batchID); err != nil {
		return err
	}
	edge, err := svc.store.GetEdge(edgeID)
	if err != nil {
		return err
	}
	if edge.BatchID != batchID {
		return model.ErrEdgeNotFound
	}
	t, err := svc.store.GetTransform(edge.TransformID)
	if err != nil {
		return err
	}
	// 更新底层变换的源为候选列，并重建批次全部血缘边。
	if err := svc.store.UpdateTransform(t.ID, newSourceTable, newSourceColumn,
		t.TargetTable, t.TargetColumn, t.Expression, model.EdgeDeclared); err != nil {
		return err
	}
	if _, err := svc.transform.BuildEdges(batchID); err != nil {
		return err
	}
	// 重新定位：找到 (新源 → 原目标) 的血缘边并确认。
	edges, err := svc.store.ListEdges(batchID)
	if err != nil {
		return err
	}
	for _, e := range edges {
		if e.SourceColID == 0 || e.TargetColID == 0 {
			continue
		}
		srcCol, err := svc.store.GetColumn(e.SourceColID)
		if err != nil {
			return err
		}
		dstCol, err := svc.store.GetColumn(e.TargetColID)
		if err != nil {
			return err
		}
		if srcCol.Name == newSourceColumn && dstCol.Name == t.TargetColumn {
			srcTbl, _ := svc.store.GetTable(srcCol.TableID)
			dstTbl, _ := svc.store.GetTable(dstCol.TableID)
			if srcTbl.QualifiedName == newSourceTable && dstTbl.QualifiedName == t.TargetTable {
				if err := svc.store.SetEdgeStatus(e.ID, model.EdgeConfirmed, "revised mapping"); err != nil {
					return err
				}
				_, err := svc.store.CreateAdjudication(batchID, e.ID, "revise", note)
				return err
			}
		}
	}
	return fmt.Errorf("diagnosis: revised edge not found after rebuild for %s.%s -> %s.%s",
		newSourceTable, newSourceColumn, t.TargetTable, t.TargetColumn)
}

// assertBatchMutable 拒绝在已封存批次上做裁决。
func (svc *Service) assertBatchMutable(batchID int64) error {
	b, err := svc.store.GetBatch(batchID)
	if err != nil {
		return err
	}
	if b.SealedAt != nil {
		return model.ErrSealedMutation
	}
	return nil
}

func (svc *Service) assertEdgeInBatch(_ int64, edgeID int64) error {
	_, err := svc.store.GetEdge(edgeID)
	if err != nil {
		return err
	}
	return nil
}
