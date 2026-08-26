// Package meta 负责元数据管理：保存表版本与列节点，并校验类型相容性。
// 这是血缘图谱的「上游数据源」登记环节。
package meta

import (
	"fmt"

	"task249-linagediag/internal/model"
	"task249-linagediag/internal/store"
)

// ColumnSpec 一个列的登记规格（HTTP 入参使用 snake_case 键）。
type ColumnSpec struct {
	Name      string             `json:"name"`
	DataType  string             `json:"data_type"`
	IsPrimary bool               `json:"is_primary"`
	Status    model.ColumnStatus `json:"status,omitempty"`
}

// TableSpec 一个表版本的登记规格（HTTP 入参使用 snake_case 键）。
type TableSpec struct {
	QualifiedName string       `json:"qualified_name"`
	SchemaVersion string       `json:"schema_version"`
	Note          string       `json:"note"`
	Columns       []ColumnSpec `json:"columns"`
}

// IngestResult 登记结果摘要。
type IngestResult struct {
	TableID int64
	Columns int
}

// IngestTable 把一张表版本及其列登记进 store。
// 校验：列名非空、类型非空、同一表版本内列名唯一；主键至多一个。
func (svc *Service) IngestTable(batchID int64, spec TableSpec) (*IngestResult, error) {
	if err := svc.store.AssertBatchMutable(batchID); err != nil {
		return nil, err
	}
	if spec.QualifiedName == "" {
		return nil, fmt.Errorf("meta: qualified_name is empty")
	}
	if spec.SchemaVersion == "" {
		return nil, fmt.Errorf("meta: schema_version is empty")
	}
	seen := map[string]bool{}
	primaryCount := 0
	for _, c := range spec.Columns {
		if c.Name == "" {
			return nil, fmt.Errorf("meta: column name is empty in table %s", spec.QualifiedName)
		}
		if c.DataType == "" {
			return nil, fmt.Errorf("meta: column %s has empty data_type", c.Name)
		}
		if seen[c.Name] {
			return nil, fmt.Errorf("meta: duplicate column %s in table %s", c.Name, spec.QualifiedName)
		}
		seen[c.Name] = true
		if c.IsPrimary {
			primaryCount++
		}
	}
	if primaryCount > 1 {
		return nil, fmt.Errorf("meta: table %s has more than one primary key", spec.QualifiedName)
	}

	tbl, err := svc.store.CreateTable(batchID, spec.QualifiedName, spec.SchemaVersion, spec.Note)
	if err != nil {
		return nil, err
	}
	for _, c := range spec.Columns {
		st := c.Status
		if st == "" {
			if c.IsPrimary {
				st = model.ColOriginal
			} else {
				st = model.ColDerived
			}
		}
		if _, err := svc.store.CreateColumn(tbl.ID, c.Name, c.DataType, st, c.IsPrimary); err != nil {
			return nil, err
		}
	}
	return &IngestResult{TableID: tbl.ID, Columns: len(spec.Columns)}, nil
}

// Service 元数据服务，持有 store 依赖。
type Service struct {
	store *store.Store
}

// NewService 构造元数据服务。
func NewService(st *store.Store) *Service {
	return &Service{store: st}
}
