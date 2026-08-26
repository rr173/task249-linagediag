package store

import (
	"database/sql"

	"task249-linagediag/internal/model"
)

// CreateEdge 写入一条计算得到的血缘边。
func (s *Store) CreateEdge(e *model.Edge) (*model.Edge, error) {
	now := nowUnix()
	res, err := s.DB.Exec(
		`INSERT INTO edges(batch_id, source_column_id, target_column_id, transform_id, status, reason, created_at)
		 VALUES(?,?,?,?,?,?,?)`,
		e.BatchID, e.SourceColID, e.TargetColID, e.TransformID, e.Status, e.Reason, now)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return s.GetEdge(id)
}

// GetEdge 按 ID 读取血缘边。
func (s *Store) GetEdge(id int64) (*model.Edge, error) {
	row := s.DB.QueryRow(
		`SELECT id,batch_id,source_column_id,target_column_id,transform_id,status,reason,created_at FROM edges WHERE id=?`, id)
	return scanEdge(row)
}

// ListEdges 列出某批次的全部血缘边。
func (s *Store) ListEdges(batchID int64) ([]*model.Edge, error) {
	rows, err := s.DB.Query(
		`SELECT id,batch_id,source_column_id,target_column_id,transform_id,status,reason,created_at FROM edges WHERE batch_id=? ORDER BY id`,
		batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.Edge
	for rows.Next() {
		e, err := scanEdgeRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// EdgesForTarget 返回指向某目标列的全部血缘边。
func (s *Store) EdgesForTarget(targetColID int64) ([]*model.Edge, error) {
	rows, err := s.DB.Query(
		`SELECT id,batch_id,source_column_id,target_column_id,transform_id,status,reason,created_at FROM edges WHERE target_column_id=? ORDER BY id`,
		targetColID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.Edge
	for rows.Next() {
		e, err := scanEdgeRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// SetEdgeStatus 更新血缘边状态。
func (s *Store) SetEdgeStatus(id int64, status model.EdgeStatus, reason string) error {
	_, err := s.DB.Exec(`UPDATE edges SET status=?, reason=? WHERE id=?`, status, reason, id)
	return err
}

// DeleteEdgesForBatch 删除某批次的全部血缘边（重建前清理）。
func (s *Store) DeleteEdgesForBatch(batchID int64) error {
	_, err := s.DB.Exec(`DELETE FROM edges WHERE batch_id=?`, batchID)
	return err
}

func scanEdge(row *sql.Row) (*model.Edge, error) {
	e := &model.Edge{}
	var reason sql.NullString
	if err := row.Scan(&e.ID, &e.BatchID, &e.SourceColID, &e.TargetColID, &e.TransformID, &e.Status, &reason, &e.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrEdgeNotFound
		}
		return nil, err
	}
	e.Reason = reason.String
	return e, nil
}

func scanEdgeRow(rows *sql.Rows) (*model.Edge, error) {
	e := &model.Edge{}
	var reason sql.NullString
	if err := rows.Scan(&e.ID, &e.BatchID, &e.SourceColID, &e.TargetColID, &e.TransformID, &e.Status, &reason, &e.CreatedAt); err != nil {
		return nil, err
	}
	e.Reason = reason.String
	return e, nil
}
