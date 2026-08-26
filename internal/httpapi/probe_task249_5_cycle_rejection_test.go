package httpapi

import (
	"net/http"
	"testing"
)

func TestTask249Bug05BuildRejectsColumnCycle(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	batchID := createBatch(t, srv, "cycle")
	ingestTable(t, srv, batchID, "db.s.left", "v1", `[{"name":"value","data_type":"text"}]`)
	ingestTable(t, srv, batchID, "db.s.right", "v1", `[{"name":"value","data_type":"text"}]`)
	for _, transform := range []string{
		`{"source_table":"db.s.left","source_column":"value","target_table":"db.s.right","target_column":"value","expression":"right.value=left.value"}`,
		`{"source_table":"db.s.right","source_column":"value","target_table":"db.s.left","target_column":"value","expression":"left.value=right.value"}`,
	} {
		code, body := do(t, http.MethodPost, srv.URL+"/api/batches/"+itoa(batchID)+"/transforms", transform)
		if code != http.StatusCreated {
			t.Fatalf("add transform: want 201 got %d body=%s", code, body)
		}
	}
	code, body := do(t, http.MethodPost, srv.URL+"/api/batches/"+itoa(batchID)+"/build", "")
	if code != http.StatusConflict {
		t.Fatalf("cyclic build: want 409 got %d body=%s", code, body)
	}
}
