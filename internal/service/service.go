// Package service 是血缘诊断的编排层，组合 meta/transform/propagate/diagnosis/snapshot，
// 对外提供与业务闭环一致的高层方法。HTTP 层与 smoke 自测均通过本层访问能力。
package service

import (
	"task249-linagediag/internal/diagnosis"
	"task249-linagediag/internal/meta"
	"task249-linagediag/internal/model"
	"task249-linagediag/internal/propagate"
	"task249-linagediag/internal/snapshot"
	"task249-linagediag/internal/store"
	"task249-linagediag/internal/transform"
)

// Service 编排服务。
type Service struct {
	Store     *store.Store
	Meta      *meta.Service
	Transform *transform.Service
	Propagate *propagate.Service
	Diagnosis *diagnosis.Service
	Snapshot  *snapshot.Service
}

// New 构造编排服务并初始化各子服务。
func New(st *store.Store) *Service {
	return &Service{
		Store:     st,
		Meta:      meta.NewService(st),
		Transform: transform.NewService(st),
		Propagate: propagate.NewService(st),
		Diagnosis: diagnosis.NewService(st, transform.NewService(st)),
		Snapshot:  snapshot.NewService(st),
	}
}

// ImportBatch 新建血缘批次。
func (s *Service) ImportBatch(name string) (*model.LineageBatch, error) {
	return s.Store.CreateBatch(name)
}

// AddTransform 登记一条列级变换声明。
func (s *Service) AddTransform(batchID, jobID int64, srcTable, srcCol, tgtTable, tgtCol, expr string) (*model.Transform, error) {
	if err := s.Store.AssertBatchMutable(batchID); err != nil {
		return nil, err
	}
	return s.Store.CreateTransform(&model.Transform{
		BatchID:      batchID,
		JobID:        jobID,
		SourceTable:  srcTable,
		SourceColumn: srcCol,
		TargetTable:  tgtTable,
		TargetColumn: tgtCol,
		Expression:   expr,
		Status:       model.EdgeDeclared,
	})
}

// AddJob 登记作业运行（含输入表版本）。
func (s *Service) AddJob(batchID, outputTableID int64, name, summary string, inputTableIDs []int64) (*model.JobRun, error) {
	return s.Store.CreateJob(batchID, outputTableID, name, summary, inputTableIDs)
}

// BuildLineage 执行「构图中」：标记状态、重建血缘边、检测环路。
// 环路属于非法状态机迁移，直接返回 ErrCycleDetected。
func (s *Service) BuildLineage(batchID int64) error {
	if err := s.Store.AssertBatchMutable(batchID); err != nil {
		return err
	}
	if err := s.Store.SetBatchStatus(batchID, model.BatchGraphing); err != nil {
		return err
	}
	if _, err := s.Transform.BuildEdges(batchID); err != nil {
		return err
	}
	if err := s.Propagate.AssertAcyclic(batchID); err != nil {
		return err
	}
	return s.Store.SetBatchStatus(batchID, model.BatchDiagnosing)
}

// Diagnose 定位截断并进入「待诊断」扭转（已在 BuildLineage 后置 diagnosing，此处仅返回结论）。
func (s *Service) Diagnose(batchID int64) ([]model.DiagnosisFinding, error) {
	return s.Diagnosis.Locate(batchID)
}

// Impact 计算某列变更的影响面。
func (s *Service) Impact(batchID int64, table, column string) ([]model.ColumnRef, error) {
	return s.Propagate.Impact(batchID, table, column)
}

// Confirm / Exempt / Revise 透传诊断裁决。
func (s *Service) Confirm(batchID, edgeID int64, note string) error {
	return s.Diagnosis.Confirm(batchID, edgeID, note)
}
func (s *Service) Exempt(batchID, edgeID int64, note string) error {
	return s.Diagnosis.Exempt(batchID, edgeID, note)
}
func (s *Service) Revise(batchID, edgeID int64, newSrcTable, newSrcCol, note string) error {
	return s.Diagnosis.Revise(batchID, edgeID, newSrcTable, newSrcCol, note)
}

// PublishSnapshot 发布诊断快照。
func (s *Service) PublishSnapshot(batchID int64, note string) (*model.Snapshot, error) {
	return s.Snapshot.Publish(batchID, note)
}

// Seal 校验（已发布快照 + 无断裂边）后封存批次。
func (s *Service) Seal(batchID int64) error {
	broken, err := s.Diagnosis.BrokenCount(batchID)
	if err != nil {
		return err
	}
	return s.Snapshot.FreezeIfComplete(batchID, broken)
}

// ConfirmAndPublish 便捷流程：确认一条边 → 重新诊断（应为 0 断裂）→ 发布快照。
func (s *Service) ConfirmAndPublish(batchID, edgeID int64, note string) (*model.Snapshot, error) {
	if err := s.Confirm(batchID, edgeID, note); err != nil {
		return nil, err
	}
	if err := s.Store.SetBatchStatus(batchID, model.BatchConfirmed); err != nil {
		return nil, err
	}
	broken, err := s.Diagnosis.BrokenCount(batchID)
	if err != nil {
		return nil, err
	}
	if broken > 0 {
		return nil, model.ErrInvalidTransition
	}
	return s.PublishSnapshot(batchID, "all breaks resolved")
}
