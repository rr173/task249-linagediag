// Package store 负责 SQLite 持久化：建表迁移与所有实体的 CRUD。
// 使用纯 Go 驱动 modernc.org/sqlite（CGO 无关，离线可构建）。
package store

import (
	"database/sql"
	_ "modernc.org/sqlite"
	"time"
)

const schemaDDL = `
CREATE TABLE IF NOT EXISTS batches (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	name        TEXT NOT NULL,
	status      TEXT NOT NULL,
	created_at  INTEGER NOT NULL,
	updated_at  INTEGER NOT NULL,
	sealed_at   INTEGER
);
CREATE TABLE IF NOT EXISTS tables (
	id             INTEGER PRIMARY KEY AUTOINCREMENT,
	batch_id       INTEGER NOT NULL,
	qualified_name TEXT NOT NULL,
	schema_version TEXT NOT NULL,
	note           TEXT,
	created_at     INTEGER NOT NULL,
	UNIQUE(batch_id, qualified_name, schema_version)
);
CREATE TABLE IF NOT EXISTS columns (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	table_id   INTEGER NOT NULL,
	name       TEXT NOT NULL,
	data_type  TEXT NOT NULL,
	status     TEXT NOT NULL,
	is_primary INTEGER NOT NULL DEFAULT 0,
	created_at INTEGER NOT NULL,
	UNIQUE(table_id, name)
);
CREATE TABLE IF NOT EXISTS jobs (
	id              INTEGER PRIMARY KEY AUTOINCREMENT,
	batch_id        INTEGER NOT NULL,
	name            TEXT NOT NULL,
	output_table_id INTEGER NOT NULL,
	summary         TEXT,
	run_at          INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS job_inputs (
	job_id   INTEGER NOT NULL,
	table_id INTEGER NOT NULL,
	PRIMARY KEY (job_id, table_id)
);
CREATE TABLE IF NOT EXISTS transforms (
	id             INTEGER PRIMARY KEY AUTOINCREMENT,
	batch_id       INTEGER NOT NULL,
	job_id         INTEGER NOT NULL,
	source_table   TEXT NOT NULL,
	source_column  TEXT NOT NULL,
	target_table   TEXT NOT NULL,
	target_column  TEXT NOT NULL,
	expression     TEXT,
	status         TEXT NOT NULL,
	created_at     INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS edges (
	id              INTEGER PRIMARY KEY AUTOINCREMENT,
	batch_id        INTEGER NOT NULL,
	source_column_id INTEGER NOT NULL,
	target_column_id INTEGER NOT NULL,
	transform_id     INTEGER NOT NULL,
	status          TEXT NOT NULL,
	reason          TEXT,
	created_at      INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS snapshots (
	id             INTEGER PRIMARY KEY AUTOINCREMENT,
	batch_id       INTEGER NOT NULL,
	version        INTEGER NOT NULL,
	status         TEXT NOT NULL,
	note           TEXT,
	created_at     INTEGER NOT NULL,
	superseded_by  INTEGER
);
CREATE TABLE IF NOT EXISTS adjudications (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	batch_id   INTEGER NOT NULL,
	edge_id    INTEGER NOT NULL,
	action     TEXT NOT NULL,
	note       TEXT,
	created_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_tables_batch     ON tables(batch_id);
CREATE INDEX IF NOT EXISTS idx_columns_table    ON columns(table_id);
CREATE INDEX IF NOT EXISTS idx_transforms_batch ON transforms(batch_id);
CREATE INDEX IF NOT EXISTS idx_edges_batch      ON edges(batch_id);
CREATE INDEX IF NOT EXISTS idx_edges_target     ON edges(target_column_id);
CREATE INDEX IF NOT EXISTS idx_snapshots_batch  ON snapshots(batch_id);
CREATE INDEX IF NOT EXISTS idx_jobs_batch       ON jobs(batch_id);
`

// Store 封装数据库连接与全部持久化操作。
type Store struct {
	DB *sql.DB
}

// Open 打开（必要时创建）SQLite 数据库并执行建表迁移。
func Open(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	// SQLite 单写者模型：限制连接数，避免并发写入 "database is locked"。
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(`PRAGMA busy_timeout=5000`); err != nil {
		_ = db.Close()
		return nil, err
	}
	if _, err := db.Exec(`PRAGMA journal_mode=WAL`); err != nil {
		_ = db.Close()
		return nil, err
	}
	if _, err := db.Exec(`PRAGMA foreign_keys=ON`); err != nil {
		_ = db.Close()
		return nil, err
	}
	if _, err := db.Exec(schemaDDL); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Store{DB: db}, nil
}

// Close 关闭数据库连接。
func (s *Store) Close() error {
	if s.DB == nil {
		return nil
	}
	return s.DB.Close()
}

// nowUnix 返回 UTC Unix 纳秒，作为统一存储时间格式（幂等、可排序）。
func nowUnix() int64 {
	return time.Now().UTC().UnixNano()
}
