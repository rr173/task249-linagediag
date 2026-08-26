package store

import (
	"database/sql"

	"task249-linagediag/internal/model"
)

// CreateTransform 登记一条列级变换声明。
func (s *Store) CreateTransform(t *model.Transform) (*model.Transform, error) {
	now := nowUnix()
	res, err := s.DB.Exec(
		`INSERT INTO transforms(batch_id, job_id, source_table, source_column, target_table, target_column, expression, status, created_at)
		 VALUES(?,?,?,?,?,?,?,?,?)`,
		t.BatchID, t.JobID, t.SourceTable, t.SourceColumn, t.TargetTable, t.TargetColumn, t.Expression, t.Status, now)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return s.GetTransform(id)
}

// GetTransform 按 ID 读取变换声明。
func (s *Store) GetTransform(id int64) (*model.Transform, error) {
	row := s.DB.QueryRow(
		`SELECT id,batch_id,job_id,source_table,source_column,target_table,target_column,expression,status,created_at
		 FROM transforms WHERE id=?`, id)
	return scanTransform(row)
}

// ListTransforms 列出某批次的全部变换声明（按 ID 升序）。
func (s *Store) ListTransforms(batchID int64) ([]*model.Transform, error) {
	rows, err := s.DB.Query(
		`SELECT id,batch_id,job_id,source_table,source_column,target_table,target_column,expression,status,created_at
		 FROM transforms WHERE batch_id=? ORDER BY id`, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.Transform
	for rows.Next() {
		t, err := scanTransformRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// SetTransformStatus 更新变换声明状态。
func (s *Store) SetTransformStatus(id int64, status model.EdgeStatus) error {
	_, err := s.DB.Exec(`UPDATE transforms SET status=? WHERE id=?`, status, id)
	return err
}

// UpdateTransform 整体更新一条变换声明（用于「修订」裁决补全上游列）。
func (s *Store) UpdateTransform(id int64, sourceTable, sourceColumn, targetTable, targetColumn, expression string, status model.EdgeStatus) error {
	_, err := s.DB.Exec(
		`UPDATE transforms SET source_table=?, source_column=?, target_table=?, target_column=?, expression=?, status=? WHERE id=?`,
		sourceTable, sourceColumn, targetTable, targetColumn, expression, status, id)
	return err
}

func scanTransform(row *sql.Row) (*model.Transform, error) {
	t := &model.Transform{}
	var expr sql.NullString
	if err := row.Scan(&t.ID, &t.BatchID, &t.JobID, &t.SourceTable, &t.SourceColumn, &t.TargetTable, &t.TargetColumn, &expr, &t.Status, &t.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrTransformNotFound
		}
		return nil, err
	}
	t.Expression = expr.String
	return t, nil
}

func scanTransformRow(rows *sql.Rows) (*model.Transform, error) {
	t := &model.Transform{}
	var expr sql.NullString
	if err := rows.Scan(&t.ID, &t.BatchID, &t.JobID, &t.SourceTable, &t.SourceColumn, &t.TargetTable, &t.TargetColumn, &expr, &t.Status, &t.CreatedAt); err != nil {
		return nil, err
	}
	t.Expression = expr.String
	return t, nil
}
