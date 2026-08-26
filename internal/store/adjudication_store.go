package store

import (
	"database/sql"

	"task249-linagediag/internal/model"
)

// CreateAdjudication 记录一次断裂边的人工裁决。
func (s *Store) CreateAdjudication(batchID, edgeID int64, action, note string) (*AdjudicationRow, error) {
	now := nowUnix()
	res, err := s.DB.Exec(
		`INSERT INTO adjudications(batch_id, edge_id, action, note, created_at) VALUES(?,?,?,?,?)`,
		batchID, edgeID, action, note, now)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return s.GetAdjudication(id)
}

// GetAdjudication 按 ID 读取裁决记录。
func (s *Store) GetAdjudication(id int64) (*AdjudicationRow, error) {
	row := s.DB.QueryRow(
		`SELECT id,batch_id,edge_id,action,note,created_at FROM adjudications WHERE id=?`, id)
	a := &AdjudicationRow{}
	if err := row.Scan(&a.ID, &a.BatchID, &a.EdgeID, &a.Action, &a.Note, &a.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrEdgeNotFound
		}
		return nil, err
	}
	return a, nil
}

// ListAdjudications 列出某批次的全部裁决（按 ID 升序）。
func (s *Store) ListAdjudications(batchID int64) ([]*AdjudicationRow, error) {
	rows, err := s.DB.Query(
		`SELECT id,batch_id,edge_id,action,note,created_at FROM adjudications WHERE batch_id=? ORDER BY id`,
		batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*AdjudicationRow
	for rows.Next() {
		a := &AdjudicationRow{}
		if err := rows.Scan(&a.ID, &a.BatchID, &a.EdgeID, &a.Action, &a.Note, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// AdjudicationRow 是裁决表的行结构（避免与 model 包循环依赖，单独定义）。
type AdjudicationRow struct {
	ID        int64     `json:"id"`
	BatchID   int64     `json:"batch_id"`
	EdgeID    int64     `json:"edge_id"`
	Action    string    `json:"action"`
	Note      string `json:"note"`
	CreatedAt int64  `json:"created_at"`
}
