// Package propagate 负责列级血缘图的构建、变更影响传播与环路检测。
// 这是「变更传播」环节的核心算法层。
package propagate

import (
	"task249-linagediag/internal/model"
	"task249-linagediag/internal/store"
)

// Service 变更传播服务。
type Service struct {
	store *store.Store
}

// NewService 构造变更传播服务。
func NewService(st *store.Store) *Service {
	return &Service{store: st}
}

// columnRef 由列 ID 解析出 ColumnRef（表限定名 + 列名）。
func (svc *Service) columnRef(colID int64) (model.ColumnRef, error) {
	col, err := svc.store.GetColumn(colID)
	if err != nil {
		return model.ColumnRef{}, err
	}
	tbl, err := svc.store.GetTable(col.TableID)
	if err != nil {
		return model.ColumnRef{}, err
	}
	return model.ColumnRef{Table: tbl.QualifiedName, Column: col.Name}, nil
}

// BuildGraph 依据某批次全部血缘边构建列级依赖有向图。
// 断裂边（源/目标列 ID 为 0）不进入可达图，避免噪声。
func (svc *Service) BuildGraph(batchID int64) (*model.DependencyGraph, error) {
	edges, err := svc.store.ListEdges(batchID)
	if err != nil {
		return nil, err
	}
	g := model.NewDependencyGraph()
	for _, e := range edges {
		if e.SourceColID == 0 || e.TargetColID == 0 {
			continue
		}
		src, err := svc.columnRef(e.SourceColID)
		if err != nil {
			return nil, err
		}
		dst, err := svc.columnRef(e.TargetColID)
		if err != nil {
			return nil, err
		}
		g.AddEdge(src, dst)
	}
	return g, nil
}
