package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTask249Bug02PassthroughRequiresCompatibleType(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	batchID := createBatch(t, srv, "passthrough-types")
	sourceID := ingestTableID(t, srv, batchID, "db.s.source", "v1", `[{"name":"id","data_type":"text"}]`)
	targetID := ingestTableID(t, srv, batchID, "db.s.target", "v1", `[{"name":"id","data_type":"bigint"}]`)
	code, body := do(t, http.MethodPost, srv.URL+"/api/batches/"+itoa(batchID)+"/jobs",
		`{"output_table_id":`+itoa(targetID)+`,"name":"copy","summary":"copy columns","input_table_ids":[`+itoa(sourceID)+`]}`)
	if code != http.StatusCreated {
		t.Fatalf("create job: want 201 got %d body=%s", code, body)
	}
	code, body = do(t, http.MethodPost, srv.URL+"/api/batches/"+itoa(batchID)+"/build", "")
	if code != http.StatusOK {
		t.Fatalf("build: want 200 got %d body=%s", code, body)
	}
	code, body = do(t, http.MethodGet, srv.URL+"/api/batches/"+itoa(batchID)+"/lineage", "")
	if code != http.StatusOK {
		t.Fatalf("lineage: want 200 got %d body=%s", code, body)
	}
	var edges []struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(body, &edges); err != nil {
		t.Fatal(err)
	}
	for _, edge := range edges {
		if edge.Status == "inferred" {
			t.Fatalf("incompatible same-name columns must not create inferred edge: %s", body)
		}
	}
}

func ingestTableID(t *testing.T, srv *httptest.Server, batchID int64, qualifiedName, version, columns string) int64 {
	t.Helper()
	body := `{"qualified_name":"` + qualifiedName + `","schema_version":"` + version + `","columns":` + columns + `}`
	code, response := do(t, http.MethodPost, srv.URL+"/api/batches/"+itoa(batchID)+"/tables", body)
	if code != http.StatusCreated {
		t.Fatalf("ingest table: want 201 got %d body=%s", code, response)
	}
	var result struct {
		TableID int64 `json:"TableID"`
	}
	if err := json.Unmarshal(response, &result); err != nil || result.TableID == 0 {
		t.Fatalf("missing table id: %s", response)
	}
	return result.TableID
}
