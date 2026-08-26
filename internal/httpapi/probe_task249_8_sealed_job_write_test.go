package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTask249Bug08SealedBatchRejectsJobWrite(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	batchID := createBatch(t, srv, "sealed-job")
	tableID := ingestTableIDForProbe(t, srv, batchID, "db.s.output", "v1", `[{"name":"value","data_type":"text"}]`)
	code, body := do(t, http.MethodPost, srv.URL+"/api/batches/"+itoa(batchID)+"/snapshots", `{"note":"sealed result"}`)
	if code != http.StatusCreated {
		t.Fatalf("publish: want 201 got %d body=%s", code, body)
	}
	code, body = do(t, http.MethodPost, srv.URL+"/api/batches/"+itoa(batchID)+"/seal", "")
	if code != http.StatusOK {
		t.Fatalf("seal: want 200 got %d body=%s", code, body)
	}
	code, body = do(t, http.MethodPost, srv.URL+"/api/batches/"+itoa(batchID)+"/jobs",
		`{"output_table_id":`+itoa(tableID)+`,"name":"late-job","summary":"must reject","input_table_ids":[`+itoa(tableID)+`]}`)
	if code != http.StatusConflict {
		t.Fatalf("sealed job write: want 409 got %d body=%s", code, body)
	}
}

func ingestTableIDForProbe(t *testing.T, srv *httptest.Server, batchID int64, qualifiedName, version, columns string) int64 {
	t.Helper()
	code, body := do(t, http.MethodPost, srv.URL+"/api/batches/"+itoa(batchID)+"/tables",
		`{"qualified_name":"`+qualifiedName+`","schema_version":"`+version+`","columns":`+columns+`}`)
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
