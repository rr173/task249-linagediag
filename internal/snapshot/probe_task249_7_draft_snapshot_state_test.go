package snapshot

import (
	"os"
	"path/filepath"
	"testing"

	"task249-linagediag/internal/store"
)

func TestTask249Bug07PublishingDoesNotSupersedeDraft(t *testing.T) {
	dir := t.TempDir()
	st, err := store.Open(filepath.Join(dir, "snapshots.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close(); _ = os.RemoveAll(dir) }()

	b, err := st.CreateBatch("snapshot-state")
	if err != nil {
		t.Fatal(err)
	}
	svc := NewService(st)
	draft, err := svc.SaveDraft(b.ID, "working diagnosis")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Publish(b.ID, "published diagnosis"); err != nil {
		t.Fatal(err)
	}
	snapshots, err := st.ListSnapshots(b.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, snap := range snapshots {
		if snap.ID == draft.ID && string(snap.Status) != "draft" {
			t.Fatalf("draft snapshot changed status after publish: %+v", snap)
		}
	}
}
