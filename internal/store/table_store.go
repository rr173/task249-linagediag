package store

import (
	"database/sql"

	"task249-linagediag/internal/model"
)

// CreateTable 登记一个表版本。同一批次内 (qualified_name, schema_version) 唯一。
func (s *Store) CreateTable(batchID int64, qualifiedName, schemaVersion, note string) (*model.TableMeta, error) {
	now := nowUnix()
	res, err := s.DB.Exec(
		`INSERT INTO tables(batch_id, qualified_name, schema_version, note, created_at) VALUES(?,?,?,?,?)`,
		batchID, qualifiedName, schemaVersion, note, now)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return s.GetTable(id)
}

// GetTable 按表版本 ID 读取。
func (s *Store) GetTable(id int64) (*model.TableMeta, error) {
	row := s.DB.QueryRow(
		`SELECT id,batch_id,qualified_name,schema_version,note,created_at FROM tables WHERE id=?`, id)
	t := &model.TableMeta{}
	if err := row.Scan(&t.ID, &t.BatchID, &t.QualifiedName, &t.SchemaVersion, &t.Note, &t.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrTableNotFound
		}
		return nil, err
	}
	return t, nil
}

// FindTable 按 (批次, 限定名, 版本) 定位表版本。
func (s *Store) FindTable(batchID int64, qualifiedName, schemaVersion string) (*model.TableMeta, error) {
	row := s.DB.QueryRow(
		`SELECT id,batch_id,qualified_name,schema_version,note,created_at FROM tables WHERE batch_id=? AND qualified_name=? AND schema_version=?`,
		batchID, qualifiedName, schemaVersion)
	t := &model.TableMeta{}
	if err := row.Scan(&t.ID, &t.BatchID, &t.QualifiedName, &t.SchemaVersion, &t.Note, &t.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrTableNotFound
		}
		return nil, err
	}
	return t, nil
}

// ListTables 列出某批次的全部表版本。
func (s *Store) ListTables(batchID int64) ([]*model.TableMeta, error) {
	rows, err := s.DB.Query(
		`SELECT id,batch_id,qualified_name,schema_version,note,created_at FROM tables WHERE batch_id=? ORDER BY id`,
		batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.TableMeta
	for rows.Next() {
		t := &model.TableMeta{}
		if err := rows.Scan(&t.ID, &t.BatchID, &t.QualifiedName, &t.SchemaVersion, &t.Note, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// LatestTableVersion 返回某批次内某限定名的最新登记版本（按 ID 降序，即登记顺序）。
func (s *Store) LatestTableVersion(batchID int64, qualifiedName string) (*model.TableMeta, error) {
	row := s.DB.QueryRow(
		`SELECT id,batch_id,qualified_name,schema_version,note,created_at FROM tables WHERE batch_id=? AND qualified_name=? ORDER BY id ASC LIMIT 1`,
		batchID, qualifiedName)
	t := &model.TableMeta{}
	if err := row.Scan(&t.ID, &t.BatchID, &t.QualifiedName, &t.SchemaVersion, &t.Note, &t.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrTableNotFound
		}
		return nil, err
	}
	return t, nil
}
