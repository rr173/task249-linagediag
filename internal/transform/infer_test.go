package transform

import (
	"path/filepath"
	"testing"

	"task249-linagediag/internal/model"
	"task249-linagediag/internal/store"
)

// newStoreWithSource 打开临时库并登记 db.s.src 的 v1/v2 两版，返回 svc 与 v2 表内列名集合的构造帮助。
func newStoreWithSource(t *testing.T, v2Cols []model.ColumnMeta) (*Service, *store.Store, func()) {
	t.Helper()
	dir := t.TempDir()
	st, err := store.Open(filepath.Join(dir, "t.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	batch, err := st.CreateBatch("rename-test")
	if err != nil {
		t.Fatalf("create batch: %v", err)
	}
	tbl, err := st.CreateTable(batch.ID, "db.s.src", "v1", "")
	if err != nil {
		t.Fatalf("create v1 table: %v", err)
	}
	if _, err := st.CreateColumn(tbl.ID, "id", "bigint", model.ColOriginal, true); err != nil {
		t.Fatalf("create v1 id: %v", err)
	}
	if _, err := st.CreateColumn(tbl.ID, "amount", "decimal", model.ColDerived, false); err != nil {
		t.Fatalf("create v1 amount: %v", err)
	}
	tbl2, err := st.CreateTable(batch.ID, "db.s.src", "v2", "")
	if err != nil {
		t.Fatalf("create v2 table: %v", err)
	}
	for _, c := range v2Cols {
		if _, err := st.CreateColumn(tbl2.ID, c.Name, c.DataType, c.Status, c.IsPrimary); err != nil {
			t.Fatalf("create v2 %s: %v", c.Name, err)
		}
	}
	return NewService(st), st, func() { st.Close() }
}

// TestInferRenameCandidatesPicksMostSimilar 验证：上游列 amount 同时可能被改成两个
// 相似名称（amount_v2 与 amt）时，候选排序必须把最相近的那个排在 [0]，
// 供 diagnosis/locate 与 service/lineage 的自动修订取用。
func TestInferRenameCandidatesPicksMostSimilar(t *testing.T) {
	v2 := []model.ColumnMeta{
		{Name: "id", DataType: "bigint", Status: model.ColOriginal, IsPrimary: true},
		{Name: "amount_v2", DataType: "decimal", Status: model.ColChanged}, // 显式后缀 → 0.95
		{Name: "amt", DataType: "decimal", Status: model.ColChanged},       // 前缀重叠 → 较低分
	}
	svc, _, cleanup := newStoreWithSource(t, v2)
	defer cleanup()

	cands, err := svc.InferRenameCandidates(1, "db.s.src", "amount")
	if err != nil {
		t.Fatalf("infer: %v", err)
	}
	if len(cands) < 2 {
		t.Fatalf("expected at least 2 candidates, got %d: %+v", len(cands), cands)
	}
	// 最高置信度必须排在首位。
	if cands[0].NewName != "amount_v2" {
		t.Fatalf("expected best candidate amount_v2, got %s (full=%+v)", cands[0].NewName, cands)
	}
	// 整体应为置信度降序。
	for i := 1; i < len(cands); i++ {
		if cands[i-1].Confidence < cands[i].Confidence {
			t.Fatalf("candidates not in descending confidence: %+v", cands)
		}
	}
}

// TestInferRenameCandidatesTieBreakStable 验证：两个候选置信度相同时，
// 按 NewName 字典序稳定 tie-break，给出确定的优先候选。
func TestInferRenameCandidatesTieBreakStable(t *testing.T) {
	v2 := []model.ColumnMeta{
		{Name: "id", DataType: "bigint", Status: model.ColOriginal, IsPrimary: true},
		{Name: "amount_v2", DataType: "decimal", Status: model.ColChanged}, // 0.95
		{Name: "amount_new", DataType: "decimal", Status: model.ColChanged}, // 0.95
	}
	svc, _, cleanup := newStoreWithSource(t, v2)
	defer cleanup()

	cands, err := svc.InferRenameCandidates(1, "db.s.src", "amount")
	if err != nil {
		t.Fatalf("infer: %v", err)
	}
	if len(cands) < 2 {
		t.Fatalf("expected at least 2 candidates, got %d", len(cands))
	}
	// 两者同分（0.95），字典序 amount_new < amount_v2。
	if cands[0].NewName != "amount_new" || cands[1].NewName != "amount_v2" {
		t.Fatalf("expected [amount_new, amount_v2] under stable tie-break, got %+v", cands)
	}
	if cands[0].Confidence != cands[1].Confidence {
		t.Fatalf("expected equal confidence, got %+v", cands)
	}
}
