// Package model 定义数据仓库血缘列级截断诊断服务的核心实体、状态机与错误。
package model

// 血缘批次状态机：接收中 → 构图中 → 待诊断 → 已确认 → 封存。
const (
	BatchReceiving  BatchStatus = "receiving"
	BatchGraphing   BatchStatus = "graphing"
	BatchDiagnosing BatchStatus = "diagnosing"
	BatchConfirmed  BatchStatus = "confirmed"
	BatchSealed     BatchStatus = "sealed"
)

// 列节点状态：原始 → 派生 → 变更 → 缺失。
const (
	ColOriginal ColumnStatus = "original"
	ColDerived  ColumnStatus = "derived"
	ColChanged  ColumnStatus = "changed"
	ColMissing  ColumnStatus = "missing"
)

// 变换边/血缘边状态：声明 → 推断 → 断裂 → 确认 → 豁免。
const (
	EdgeDeclared  EdgeStatus = "declared"
	EdgeInferred  EdgeStatus = "inferred"
	EdgeBroken    EdgeStatus = "broken"
	EdgeConfirmed EdgeStatus = "confirmed"
	EdgeExempt    EdgeStatus = "exempt"
)

// 诊断快照状态：草稿 → 发布 → 替代。
const (
	SnapDraft      SnapshotStatus = "draft"
	SnapPublished  SnapshotStatus = "published"
	SnapSuperseded SnapshotStatus = "superseded"
)

// BatchStatus 血缘批次状态。
type BatchStatus string

// ColumnStatus 列节点状态。
type ColumnStatus string

// EdgeStatus 变换边/血缘边状态。
type EdgeStatus string

// SnapshotStatus 诊断快照状态。
type SnapshotStatus string

// LineageBatch 一次血缘诊断的批次（一批相关的表版本、变换与作业运行）。
type LineageBatch struct {
	ID        int64       `json:"id"`
	Name      string      `json:"name"`
	Status    BatchStatus `json:"status"`
	CreatedAt int64   `json:"created_at"`
	UpdatedAt int64   `json:"updated_at"`
	SealedAt  *int64  `json:"sealed_at,omitempty"`
}

// TableMeta 表版本（qualified_name 形如 db.schema.table，schema_version 区分同一表的不同版本）。
type TableMeta struct {
	ID            int64     `json:"id"`
	BatchID       int64     `json:"batch_id"`
	QualifiedName string    `json:"qualified_name"`
	SchemaVersion string    `json:"schema_version"`
	Note          string    `json:"note"`
	CreatedAt     int64 `json:"created_at"`
}

// ColumnMeta 列节点，依附于某个表版本。
type ColumnMeta struct {
	ID        int64        `json:"id"`
	TableID   int64        `json:"table_id"`
	Name      string       `json:"name"`
	DataType  string       `json:"data_type"`
	Status    ColumnStatus `json:"status"`
	IsPrimary bool         `json:"is_primary"`
	CreatedAt int64    `json:"created_at"`
}

// JobRun 作业运行摘要：一个作业把若干输入表版本产出一个输出表版本。
type JobRun struct {
	ID            int64     `json:"id"`
	BatchID       int64     `json:"batch_id"`
	Name          string    `json:"name"`
	OutputTableID int64     `json:"output_table_id"`
	Summary       string    `json:"summary"`
	RunAt         int64 `json:"run_at"`
}

// Transform 列级变换声明：从 source 表某列映射到 target 表某列，附带表达式。
type Transform struct {
	ID           int64      `json:"id"`
	BatchID      int64      `json:"batch_id"`
	JobID        int64      `json:"job_id"`
	SourceTable  string     `json:"source_table"`
	SourceColumn string     `json:"source_column"`
	TargetTable  string     `json:"target_table"`
	TargetColumn string     `json:"target_column"`
	Expression   string     `json:"expression"`
	Status       EdgeStatus `json:"status"`
	CreatedAt    int64  `json:"created_at"`
}

// Edge 计算得到的列级血缘边（由 transform 解析或自动推断而来）。
type Edge struct {
	ID            int64      `json:"id"`
	BatchID       int64      `json:"batch_id"`
	SourceColID   int64      `json:"source_column_id"`
	TargetColID   int64      `json:"target_column_id"`
	TransformID   int64      `json:"transform_id"`
	Status        EdgeStatus `json:"status"`
	Reason        string     `json:"reason"`
	CreatedAt     int64  `json:"created_at"`
}

// Snapshot 诊断快照：冻结某一时刻的元数据输入与血缘结论。
type Snapshot struct {
	ID           int64          `json:"id"`
	BatchID      int64          `json:"batch_id"`
	Version      int            `json:"version"`
	Status       SnapshotStatus `json:"status"`
	Note         string         `json:"note"`
	CreatedAt    int64      `json:"created_at"`
	SupersededBy *int64         `json:"superseded_by,omitempty"`
}

// Adjudication 断裂边的人工裁决记录。
type Adjudication struct {
	ID        int64     `json:"id"`
	BatchID   int64     `json:"batch_id"`
	EdgeID    int64     `json:"edge_id"`
	Action    string    `json:"action"` // confirm | exempt | revise
	Note      string    `json:"note"`
	CreatedAt int64 `json:"created_at"`
}

// DiagnosisFinding 诊断结论：某下游列为何仍使用变更前的上游值。
type DiagnosisFinding struct {
	EdgeID         int64  `json:"edge_id"`
	TargetTable    string `json:"target_table"`
	TargetColumn   string `json:"target_column"`
	SourceTable    string `json:"source_table"`
	SourceColumn   string `json:"source_column"`
	Broken         bool   `json:"broken"`
	Reason         string `json:"reason"`
	SuggestedFix   string `json:"suggested_fix"`
}
