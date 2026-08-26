package store

import (
	"database/sql"

	"task249-linagediag/internal/model"
)

// CreateBatch 新建血缘批次，初始状态为「接收中」。同一批次幂等由调用方保证。
func (s *Store) CreateBatch(name string) (*model.LineageBatch, error) {
	now := nowUnix()
	res, err := s.DB.Exec(
		`INSERT INTO batches(name, status, created_at, updated_at) VALUES(?,?,?,?)`,
		name, model.BatchReceiving, now, now)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return s.GetBatch(id)
}

// GetBatch 按 ID 读取批次。
func (s *Store) GetBatch(id int64) (*model.LineageBatch, error) {
	row := s.DB.QueryRow(
		`SELECT id,name,status,created_at,updated_at,sealed_at FROM batches WHERE id=?`, id)
	b := &model.LineageBatch{}
	var sealed sql.NullInt64
	if err := row.Scan(&b.ID, &b.Name, &b.Status, &b.CreatedAt, &b.UpdatedAt, &sealed); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrBatchNotFound
		}
		return nil, err
	}
	if sealed.Valid {
		v := sealed.Int64
		b.SealedAt = &v
	}
	return b, nil
}

// AssertBatchMutable verifies that a batch exists and has not been sealed.
// All service-level write operations use this guard before changing metadata.
func (s *Store) AssertBatchMutable(id int64) error {
	b, err := s.GetBatch(id)
	if err != nil {
		return err
	}
	if b.SealedAt != nil {
		return model.ErrSealedMutation
	}
	return nil
}

// SetBatchStatus 更新批次状态（同时刷新 updated_at）。
func (s *Store) SetBatchStatus(id int64, st model.BatchStatus) error {
	_, err := s.DB.Exec(`UPDATE batches SET status=?, updated_at=? WHERE id=?`, st, nowUnix(), id)
	return err
}

// SealBatch 将批次封存，记录 sealed_at。封存后任何写操作都应被拒绝。
func (s *Store) SealBatch(id int64) error {
	now := nowUnix()
	_, err := s.DB.Exec(
		`UPDATE batches SET status=?, sealed_at=?, updated_at=? WHERE id=?`,
		model.BatchSealed, now, now, id)
	return err
}

// ListBatches 列出全部批次（按 ID 升序）。
func (s *Store) ListBatches() ([]*model.LineageBatch, error) {
	rows, err := s.DB.Query(
		`SELECT id,name,status,created_at,updated_at,sealed_at FROM batches ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.LineageBatch
	for rows.Next() {
		b := &model.LineageBatch{}
		var sealed sql.NullInt64
		if err := rows.Scan(&b.ID, &b.Name, &b.Status, &b.CreatedAt, &b.UpdatedAt, &sealed); err != nil {
			return nil, err
		}
		if sealed.Valid {
			v := sealed.Int64
			b.SealedAt = &v
		}
		out = append(out, b)
	}
	return out, rows.Err()
}
