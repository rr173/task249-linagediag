package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestTask249Bug03RenameCandidateUsesHighestConfidence(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	batchID := createBatch(t, srv, "rename-candidates")
	ingestTable(t, srv, batchID, "db.s.src", "v1", `[{"name":"amount","data_type":"decimal"}]`)
	ingestTable(t, srv, batchID, "db.s.src", "v2", `[{"name":"amount_new","data_type":"decimal"},{"name":"amt","data_type":"decimal"}]`)
	ingestTable(t, srv, batchID, "db.s.tgt", "v1", `[{"name":"amount","data_type":"decimal"}]`)
	code, body := do(t, http.MethodPost, srv.URL+"/api/batches/"+itoa(batchID)+"/transforms",
		`{"source_table":"db.s.src","source_column":"amount","target_table":"db.s.tgt","target_column":"amount","expression":"tgt.amount=src.amount"}`)
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
		SuggestedFix string `json:"suggested_fix"`
	}
	if err := json.Unmarshal(body, &findings); err != nil || len(findings) != 1 {
		t.Fatalf("unexpected diagnose response: %s", body)
	}
	if !strings.Contains(findings[0].SuggestedFix, "amount_new") {
		t.Fatalf("highest-confidence rename must be suggested first: %s", body)
	}
}
