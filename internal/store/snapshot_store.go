package store

import (
	"database/sql"

	"task249-linagediag/internal/model"
)

// NextSnapshotVersion 返回某批次下一个快照版本号（当前最大版本 + 1）。
func (s *Store) NextSnapshotVersion(batchID int64) (int, error) {
	var max sql.NullInt64
	err := s.DB.QueryRow(`SELECT MAX(version) FROM snapshots WHERE batch_id=?`, batchID).Scan(&max)
	if err != nil {
		return 0, err
	}
	if !max.Valid {
		return 1, nil
	}
	return int(max.Int64) + 1, nil
}

// CreateSnapshot 登记诊断快照并落库；发布态时会把同批次旧发布快照标记为「替代」。
func (s *Store) CreateSnapshot(batchID int64, version int, status model.SnapshotStatus, note string) (*model.Snapshot, error) {
	now := nowUnix()
	tx, err := s.DB.Begin()
	if err != nil {
		return nil, err
	}
	res, err := tx.Exec(
		`INSERT INTO snapshots(batch_id, version, status, note, created_at) VALUES(?,?,?,?,?)`,
		batchID, version, status, note, now)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	snapID, err := res.LastInsertId()
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if status == model.SnapPublished {
		// 将同一批次此前发布的快照标记为替代。
		if _, err := tx.Exec(
			`UPDATE snapshots SET status=?, superseded_by=? WHERE batch_id=? AND status=? AND id<>?`,
			model.SnapSuperseded, snapID, batchID, model.SnapPublished, snapID); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.GetSnapshot(snapID)
}

// GetSnapshot 按 ID 读取快照。
func (s *Store) GetSnapshot(id int64) (*model.Snapshot, error) {
	row := s.DB.QueryRow(
		`SELECT id,batch_id,version,status,note,created_at,superseded_by FROM snapshots WHERE id=?`, id)
	return scanSnapshot(row)
}

// ListSnapshots 列出某批次的全部快照（按版本升序）。
func (s *Store) ListSnapshots(batchID int64) ([]*model.Snapshot, error) {
	rows, err := s.DB.Query(
		`SELECT id,batch_id,version,status,note,created_at,superseded_by FROM snapshots WHERE batch_id=? ORDER BY version`,
		batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.Snapshot
	for rows.Next() {
		snap, err := scanSnapshotRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, snap)
	}
	return out, rows.Err()
}

// LatestPublishedSnapshot 返回某批次最新发布的快照（版本最大）。
func (s *Store) LatestPublishedSnapshot(batchID int64) (*model.Snapshot, error) {
	row := s.DB.QueryRow(
		`SELECT id,batch_id,version,status,note,created_at,superseded_by FROM snapshots
		 WHERE batch_id=? AND status=? ORDER BY version DESC LIMIT 1`,
		batchID, model.SnapPublished)
	return scanSnapshot(row)
}

func scanSnapshot(row *sql.Row) (*model.Snapshot, error) {
	snap := &model.Snapshot{}
	var sup sql.NullInt64
	if err := row.Scan(&snap.ID, &snap.BatchID, &snap.Version, &snap.Status, &snap.Note, &snap.CreatedAt, &sup); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrSnapshotNotFound
		}
		return nil, err
	}
	if sup.Valid {
		v := sup.Int64
		snap.SupersededBy = &v
	}
	return snap, nil
}

func scanSnapshotRow(rows *sql.Rows) (*model.Snapshot, error) {
	snap := &model.Snapshot{}
	var sup sql.NullInt64
	if err := rows.Scan(&snap.ID, &snap.BatchID, &snap.Version, &snap.Status, &snap.Note, &snap.CreatedAt, &sup); err != nil {
		return nil, err
	}
	if sup.Valid {
		v := sup.Int64
		snap.SupersededBy = &v
	}
	return snap, nil
}
