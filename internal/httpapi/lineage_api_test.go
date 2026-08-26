package httpapi

import (
	"net/http"
	"testing"
)

func TestHTTPLineageBuildAndImpactViews(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	batchID := createBatch(t, srv, "lineage-api")
	ingestTable(t, srv, batchID, "db.s.source", "v1", `[{"name":"id","data_type":"bigint"},{"name":"amount","data_type":"decimal"}]`)
	ingestTable(t, srv, batchID, "db.s.target", "v1", `[{"name":"id","data_type":"bigint"},{"name":"amount","data_type":"decimal"}]`)

	code, body := do(t, http.MethodPost, srv.URL+"/api/batches/"+itoa(batchID)+"/transforms",
		`{"source_table":"db.s.source","source_column":"amount","target_table":"db.s.target","target_column":"amount","expression":"target.amount=source.amount"}`)
	if code != http.StatusCreated {
		t.Fatalf("add transform: want 201 got %d body=%s", code, body)
	}
	code, body = do(t, http.MethodPost, srv.URL+"/api/batches/"+itoa(batchID)+"/build", "")
	if code != http.StatusOK {
		t.Fatalf("build lineage: want 200 got %d body=%s", code, body)
	}
	for _, path := range []string{
		"/lineage",
		"/impact?table=db.s.source&column=amount",
	} {
		code, body = do(t, http.MethodGet, srv.URL+"/api/batches/"+itoa(batchID)+path, "")
		if code != http.StatusOK || len(body) == 0 {
			t.Fatalf("GET %s: want 200 with body, got %d body=%s", path, code, body)
		}
	}
}
