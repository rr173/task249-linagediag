package model

import "errors"

// 领域错误。业务包在边界处返回这些错误，由 service/httpapi 映射到 HTTP 状态码。
var (
	ErrBatchNotFound     = errors.New("lineage: batch not found")
	ErrBatchSealed       = errors.New("lineage: batch is sealed, mutation rejected")
	ErrTableNotFound     = errors.New("lineage: table version not found")
	ErrColumnNotFound    = errors.New("lineage: column not found")
	ErrTransformNotFound = errors.New("lineage: transform not found")
	ErrEdgeNotFound      = errors.New("lineage: edge not found")
	ErrSnapshotNotFound  = errors.New("lineage: snapshot not found")
	ErrCycleDetected     = errors.New("lineage: circular column derivation detected")
	ErrTypeIncompatible  = errors.New("lineage: column type incompatible between source and target")
	ErrVersionMissing    = errors.New("lineage: referenced schema version missing")
	ErrSealedMutation    = errors.New("lineage: cannot mutate a sealed batch")
	ErrDuplicateColumn   = errors.New("lineage: duplicate column within the same table version")
	ErrInvalidTransition = errors.New("lineage: invalid state transition")
)

// IsCycle 判断错误是否为循环派生。
func IsCycle(err error) bool {
	return errors.Is(err, ErrCycleDetected)
}
