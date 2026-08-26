package propagate

import (
	"task249-linagediag/internal/model"
)

// DetectCycle 判断某批次的列级血缘图是否存在环（循环派生属于非法状态机迁移）。
func (svc *Service) DetectCycle(batchID int64) (bool, error) {
	g, err := svc.BuildGraph(batchID)
	if err != nil {
		return false, err
	}
	return g.HasCycle(), nil
}

// AssertAcyclic 在构图后断言无环，否则返回 ErrCycleDetected。
func (svc *Service) AssertAcyclic(batchID int64) error {
	cyclic, err := svc.DetectCycle(batchID)
	if err != nil {
		return err
	}
	if cyclic {
		return model.ErrCycleDetected
	}
	return nil
}
