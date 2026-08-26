package httpapi

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestTask249Bug10AdjudicationStaysWithinBatch(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	first := createBrokenBatch(t, srv)
	second := createBrokenBatch(t, srv)
	code, body := do(t, http.MethodGet, srv.URL+"/api/batches/"+itoa(second)+"/diagnose", "")
	if code != http.StatusOK {
		t.Fatalf("second diagnose: want 200 got %d body=%s", code, body)
	}
	var findings []struct {
		EdgeID int64 `json:"edge_id"`
	}
	if err := json.Unmarshal(body, &findings); err != nil || len(findings) != 1 {
		t.Fatalf("unexpected second diagnose response: %s", body)
	}
	code, body = do(t, http.MethodPost, srv.URL+"/api/batches/"+itoa(first)+"/edges/"+itoa(findings[0].EdgeID)+"/confirm", `{"note":"wrong batch"}`)
	if code == http.StatusOK {
		t.Fatalf("cross-batch adjudication must be rejected: %s", body)
	}
	code, body = do(t, http.MethodGet, srv.URL+"/api/batches/"+itoa(second)+"/edges", "")
	if code != http.StatusOK {
		t.Fatalf("second edges: want 200 got %d body=%s", code, body)
	}
	var edges []struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(body, &edges); err != nil || len(edges) == 0 || edges[0].Status != "broken" {
		t.Fatalf("foreign edge must remain broken: %s", body)
	}
}
