package store

import (
	"database/sql"

	"task249-linagediag/internal/model"
)

// CreateJob 登记作业运行，并写入其输入表版本关联。
func (s *Store) CreateJob(batchID, outputTableID int64, name, summary string, inputTableIDs []int64) (*model.JobRun, error) {
	now := nowUnix()
	tx, err := s.DB.Begin()
	if err != nil {
		return nil, err
	}
	res, err := tx.Exec(
		`INSERT INTO jobs(batch_id, name, output_table_id, summary, run_at) VALUES(?,?,?,?,?)`,
		batchID, name, outputTableID, summary, now)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	jobID, err := res.LastInsertId()
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	for _, in := range inputTableIDs {
		if _, err := tx.Exec(`INSERT OR IGNORE INTO job_inputs(job_id, table_id) VALUES(?,?)`, jobID, in); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.GetJob(jobID)
}

// GetJob 按 ID 读取作业运行。
func (s *Store) GetJob(id int64) (*model.JobRun, error) {
	row := s.DB.QueryRow(
		`SELECT id,batch_id,name,output_table_id,summary,run_at FROM jobs WHERE id=?`, id)
	j := &model.JobRun{}
	if err := row.Scan(&j.ID, &j.BatchID, &j.Name, &j.OutputTableID, &j.Summary, &j.RunAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrTransformNotFound
		}
		return nil, err
	}
	return j, nil
}

// ListJobs 列出某批次的全部作业运行。
func (s *Store) ListJobs(batchID int64) ([]*model.JobRun, error) {
	rows, err := s.DB.Query(
		`SELECT id,batch_id,name,output_table_id,summary,run_at FROM jobs WHERE batch_id=? ORDER BY id`,
		batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.JobRun
	for rows.Next() {
		j := &model.JobRun{}
		if err := rows.Scan(&j.ID, &j.BatchID, &j.Name, &j.OutputTableID, &j.Summary, &j.RunAt); err != nil {
			return nil, err
		}
		out = append(out, j)
	}
	return out, rows.Err()
}

// JobInputs 返回某作业的全部输入表版本 ID。
func (s *Store) JobInputs(jobID int64) ([]int64, error) {
	rows, err := s.DB.Query(`SELECT table_id FROM job_inputs WHERE job_id=? ORDER BY table_id`, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}
