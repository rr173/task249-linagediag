package snapshot

import (
	"task249-linagediag/internal/model"
)

// FreezeIfComplete 在满足「已发布快照 + 无断裂边」时自动封存批次；否则返回原因。
// 由 service 层在发布后调用，作为质量门禁。
func (svc *Service) FreezeIfComplete(batchID int64, brokenCount int) error {
	snap, err := svc.store.LatestPublishedSnapshot(batchID)
	if err != nil {
		return err
	}
	if snap == nil {
		return model.ErrSnapshotNotFound
	}
	if brokenCount > 0 {
		return model.ErrInvalidTransition
	}
	return svc.store.SealBatch(batchID)
}
