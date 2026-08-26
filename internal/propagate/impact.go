package propagate

import (
	"fmt"

	"task249-linagediag/internal/model"
)

// Impact 返回某列（table.column）发生变更后受影响的全部下游列。
// 用于「上游列重命名/类型变更 → 下游为何仍用旧值」的影响面评估。
func (svc *Service) Impact(batchID int64, table, column string) ([]model.ColumnRef, error) {
	g, err := svc.BuildGraph(batchID)
	if err != nil {
		return nil, err
	}
	start := model.ColumnRef{Table: table, Column: column}
	if !g.HasNode(start) {
		return nil, fmt.Errorf("propagate: column %s.%s not present in lineage graph", table, column)
	}
	return g.Downstream(start), nil
}
