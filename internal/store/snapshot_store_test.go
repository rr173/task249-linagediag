package store

import (
	"path/filepath"
	"testing"

	"task249-linagediag/internal/model"
)

// TestCreateSnapshotPreservesDrafts 验证：发布新诊断快照时，此前保存的草稿态快照
// 不应被标记为「替代」，草稿内容仍可继续查看。
func TestCreateSnapshotPreservesDrafts(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(filepath.Join(dir, "t.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	b, err := st.CreateBatch("draft-preserve")
	if err != nil {
		t.Fatalf("create batch: %v", err)
	}

	// 1) 保存一份草稿态快照。
	draft, err := st.CreateSnapshot(b.ID, 1, model.SnapDraft, "draft result")
	if err != nil {
		t.Fatalf("create draft: %v", err)
	}
	if draft.Status != model.SnapDraft {
		t.Fatalf("draft status: want %q got %q", model.SnapDraft, draft.Status)
	}

	// 2) 发布一份新快照（不应把草稿标记为替代）。
	if _, err := st.CreateSnapshot(b.ID, 2, model.SnapPublished, "published result"); err != nil {
		t.Fatalf("publish: %v", err)
	}

	// 3) 草稿仍应为 draft 态且可被读取查看。
	got, err := st.GetSnapshot(draft.ID)
	if err != nil {
		t.Fatalf("get draft after publish: %v", err)
	}
	if got.Status != model.SnapDraft {
		t.Fatalf("draft superseded by publish: want status %q got %q (note=%q)",
			model.SnapDraft, got.Status, got.Note)
	}
	if got.Note != "draft result" {
		t.Fatalf("draft content changed: want %q got %q", "draft result", got.Note)
	}
	if got.SupersededBy != nil {
		t.Fatalf("draft should not be superseded, got superseded_by=%d", *got.SupersededBy)
	}

	// 4) 再次发布：上一份发布快照被标记为替代，草稿仍保持 draft。
	second, err := st.CreateSnapshot(b.ID, 3, model.SnapPublished, "second publish")
	if err != nil {
		t.Fatalf("second publish: %v", err)
	}
	prev, err := st.GetSnapshot(2)
	if err != nil {
		t.Fatalf("get previous published: %v", err)
	}
	if prev.Status != model.SnapSuperseded {
		t.Fatalf("previous published not superseded: want %q got %q", model.SnapSuperseded, prev.Status)
	}
	if prev.SupersededBy == nil || *prev.SupersededBy != second.ID {
		t.Fatalf("previous published superseded_by: want %d got %v", second.ID, prev.SupersededBy)
	}
	gotDraft, err := st.GetSnapshot(draft.ID)
	if err != nil {
		t.Fatalf("get draft after second publish: %v", err)
	}
	if gotDraft.Status != model.SnapDraft {
		t.Fatalf("draft changed after second publish: want %q got %q", model.SnapDraft, gotDraft.Status)
	}
}
