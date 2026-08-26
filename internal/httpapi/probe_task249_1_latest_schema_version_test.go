package httpapi

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestTask249Bug01LatestSchemaVersionDetectsRename(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	batchID := createBrokenBatch(t, srv)
	code, body := do(t, http.MethodGet, srv.URL+"/api/batches/"+itoa(batchID)+"/diagnose", "")
	if code != http.StatusOK {
		t.Fatalf("diagnose: want 200 got %d body=%s", code, body)
	}
	var findings []struct {
		Broken bool `json:"broken"`
	}
	if err := json.Unmarshal(body, &findings); err != nil {
		t.Fatal(err)
	}
	if len(findings) == 0 || !findings[0].Broken {
		t.Fatalf("renamed source column must be reported as broken: %s", body)
	}
}
