// Package snapshot 负责诊断快照的发布与封存：冻结某一时刻的元数据输入与血缘结论。
package snapshot

import (
	"fmt"

	"task249-linagediag/internal/model"
	"task249-linagediag/internal/store"
)

// Service 快照服务。
type Service struct {
	store *store.Store
}

// NewService 构造快照服务。
func NewService(st *store.Store) *Service {
	return &Service{store: st}
}

// SaveDraft 保存一个草稿态快照（不替代其它发布快照）。
func (svc *Service) SaveDraft(batchID int64, note string) (*model.Snapshot, error) {
	b, err := svc.store.GetBatch(batchID)
	if err != nil {
		return nil, err
	}
	if b.SealedAt != nil {
		return nil, model.ErrSealedMutation
	}
	ver, err := svc.store.NextSnapshotVersion(batchID)
	if err != nil {
		return nil, err
	}
	return svc.store.CreateSnapshot(batchID, ver, model.SnapDraft, note)
}

// Publish 发布一个诊断快照：落库为 published，并把同批次旧发布快照标记为替代。
func (svc *Service) Publish(batchID int64, note string) (*model.Snapshot, error) {
	b, err := svc.store.GetBatch(batchID)
	if err != nil {
		return nil, err
	}
	if b.SealedAt != nil {
		return nil, model.ErrSealedMutation
	}
	ver, err := svc.store.NextSnapshotVersion(batchID)
	if err != nil {
		return nil, err
	}
	return svc.store.CreateSnapshot(batchID, ver, model.SnapPublished, note)
}

// LatestPublished 返回最新发布的快照。
func (svc *Service) LatestPublished(batchID int64) (*model.Snapshot, error) {
	return svc.store.LatestPublishedSnapshot(batchID)
}

// Freeze 封存批次（应在发布快照后调用）。无发布快照则拒绝封存。
func (svc *Service) Freeze(batchID int64) error {
	if _, err := svc.store.LatestPublishedSnapshot(batchID); err != nil {
		return fmt.Errorf("snapshot: cannot seal batch %d without a published snapshot", batchID)
	}
	return svc.store.SealBatch(batchID)
}
