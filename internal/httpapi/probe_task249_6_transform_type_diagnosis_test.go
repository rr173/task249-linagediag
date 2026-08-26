package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestTask249Bug06IncompatibleTransformIsDiagnosed(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	batchID := createBatch(t, srv, "transform-types")
	ingestTable(t, srv, batchID, "db.s.source", "v1", `[{"name":"value","data_type":"bigint"}]`)
	ingestTable(t, srv, batchID, "db.s.target", "v1", `[{"name":"value","data_type":"text"}]`)
	code, body := do(t, http.MethodPost, srv.URL+"/api/batches/"+itoa(batchID)+"/transforms",
		`{"source_table":"db.s.source","source_column":"value","target_table":"db.s.target","target_column":"value","expression":"target.value=source.value"}`)
	if code != http.StatusCreated {
		t.Fatalf("add transform: want 201 got %d body=%s", code, body)
	}
	code, body = do(t, http.MethodPost, srv.URL+"/api/batches/"+itoa(batchID)+"/build", "")
	if code != http.StatusOK {
		t.Fatalf("build: want 200 got %d body=%s", code, body)
	}
	code, body = do(t, http.MethodGet, srv.URL+"/api/batches/"+itoa(batchID)+"/diagnose", "")
	if code != http.StatusOK {
		t.Fatalf("diagnose: want 200 got %d body=%s", code, body)
	}
	var findings []struct {
		Broken bool   `json:"broken"`
		Reason string `json:"reason"`
	}
	if err := json.Unmarshal(body, &findings); err != nil || len(findings) != 1 {
		t.Fatalf("unexpected diagnose response: %s", body)
	}
	if !findings[0].Broken || !strings.Contains(findings[0].Reason, "incompatible") {
		t.Fatalf("incompatible transform must be diagnosed: %s", body)
	}
}
