package store

import (
	"database/sql"

	"task249-linagediag/internal/model"
)

// CreateColumn 在某表版本下登记一个列节点。同一表版本内列名唯一。
func (s *Store) CreateColumn(tableID int64, name, dataType string, status model.ColumnStatus, isPrimary bool) (*model.ColumnMeta, error) {
	now := nowUnix()
	res, err := s.DB.Exec(
		`INSERT INTO columns(table_id, name, data_type, status, is_primary, created_at) VALUES(?,?,?,?,?,?)`,
		tableID, name, dataType, status, boolToInt(isPrimary), now)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return s.GetColumn(id)
}

// GetColumn 按列 ID 读取。
func (s *Store) GetColumn(id int64) (*model.ColumnMeta, error) {
	row := s.DB.QueryRow(
		`SELECT id,table_id,name,data_type,status,is_primary,created_at FROM columns WHERE id=?`, id)
	return scanColumn(row)
}

// FindColumn 按 (表版本, 列名) 定位列节点。
func (s *Store) FindColumn(tableID int64, name string) (*model.ColumnMeta, error) {
	row := s.DB.QueryRow(
		`SELECT id,table_id,name,data_type,status,is_primary,created_at FROM columns WHERE table_id=? AND name=?`,
		tableID, name)
	return scanColumn(row)
}

// ListColumns 列出某表版本的全部列。
func (s *Store) ListColumns(tableID int64) ([]*model.ColumnMeta, error) {
	rows, err := s.DB.Query(
		`SELECT id,table_id,name,data_type,status,is_primary,created_at FROM columns WHERE table_id=? ORDER BY id`,
		tableID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.ColumnMeta
	for rows.Next() {
		c, err := scanColumnRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// SetColumnStatus 更新列节点状态。
func (s *Store) SetColumnStatus(id int64, status model.ColumnStatus) error {
	_, err := s.DB.Exec(`UPDATE columns SET status=? WHERE id=?`, status, id)
	return err
}

// ColumnID 便捷查询：(限定名, 版本, 列名) → 列 ID。找不到返回 0。
func (s *Store) ColumnID(batchID int64, qualifiedName, schemaVersion, column string) (int64, error) {
	var colID int64
	err := s.DB.QueryRow(
		`SELECT c.id FROM columns c
		 JOIN tables t ON c.table_id = t.id
		 WHERE t.batch_id=? AND t.qualified_name=? AND t.schema_version=? AND c.name=?`,
		batchID, qualifiedName, schemaVersion, column).Scan(&colID)
	if err == sql.ErrNoRows {
		return 0, model.ErrColumnNotFound
	}
	if err != nil {
		return 0, err
	}
	return colID, nil
}

func scanColumn(row *sql.Row) (*model.ColumnMeta, error) {
	c := &model.ColumnMeta{}
	var isPrimary int
	if err := row.Scan(&c.ID, &c.TableID, &c.Name, &c.DataType, &c.Status, &isPrimary, &c.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrColumnNotFound
		}
		return nil, err
	}
	c.IsPrimary = isPrimary != 0
	return c, nil
}

func scanColumnRow(rows *sql.Rows) (*model.ColumnMeta, error) {
	c := &model.ColumnMeta{}
	var isPrimary int
	if err := rows.Scan(&c.ID, &c.TableID, &c.Name, &c.DataType, &c.Status, &isPrimary, &c.CreatedAt); err != nil {
		return nil, err
	}
	c.IsPrimary = isPrimary != 0
	return c, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
