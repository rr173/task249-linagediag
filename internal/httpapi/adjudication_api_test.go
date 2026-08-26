package httpapi

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestHTTPBrokenEdgeRevisionAndAdjudication(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	batchID := createBrokenBatch(t, srv)
	code, body := do(t, http.MethodGet, srv.URL+"/api/batches/"+itoa(batchID)+"/diagnose", "")
	if code != http.StatusOK {
		t.Fatalf("diagnose: want 200 got %d body=%s", code, body)
	}
	var findings []struct {
		EdgeID int64 `json:"edge_id"`
		Broken bool  `json:"broken"`
	}
	if err := json.Unmarshal(body, &findings); err != nil || len(findings) != 1 || !findings[0].Broken {
		t.Fatalf("expected one broken finding: %s", body)
	}

	path := srv.URL + "/api/batches/" + itoa(batchID) + "/edges/" + itoa(findings[0].EdgeID) + "/revise"
	code, body = do(t, http.MethodPost, path, `{"new_source_table":"db.s.src","new_source_column":"amt","note":"rename confirmed"}`)
	if code != http.StatusOK {
		t.Fatalf("revise edge: want 200 got %d body=%s", code, body)
	}
	code, body = do(t, http.MethodGet, srv.URL+"/api/batches/"+itoa(batchID)+"/adjudications", "")
	if code != http.StatusOK || len(body) == 0 || !contains(body, `"action":"revise"`) {
		t.Fatalf("list adjudications missing revise record: code=%d body=%s", code, body)
	}
}
