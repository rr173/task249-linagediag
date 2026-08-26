package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTask249Bug09ColumnsStayWithinBatch(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	first := createBatch(t, srv, "first")
	second := createBatch(t, srv, "second")
	foreignTable := ingestTableIDForColumnProbe(t, srv, second)
	code, body := do(t, http.MethodGet, srv.URL+"/api/batches/"+itoa(first)+"/columns?table_id="+itoa(foreignTable), "")
	if code != http.StatusNotFound {
		t.Fatalf("cross-batch column read: want 404 got %d body=%s", code, body)
	}
}

func ingestTableIDForColumnProbe(t *testing.T, srv *httptest.Server, batchID int64) int64 {
	t.Helper()
	code, body := do(t, http.MethodPost, srv.URL+"/api/batches/"+itoa(batchID)+"/tables",
		`{"qualified_name":"db.s.foreign","schema_version":"v1","columns":[{"name":"secret","data_type":"text"}]}`)
	if code != http.StatusCreated {
		t.Fatalf("ingest: want 201 got %d body=%s", code, body)
	}
	var result struct {
		TableID int64 `json:"TableID"`
	}
	if err := json.Unmarshal(body, &result); err != nil || result.TableID == 0 {
		t.Fatalf("missing table id: %s", body)
	}
	return result.TableID
}
