package httpapi

import (
	"net/http"
	"testing"
)

func TestHTTPMetadataRegistrationAndColumnListing(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	batchID := createBatch(t, srv, "metadata-api")
	body := `{"qualified_name":"db.s.source","schema_version":"v1","columns":[{"name":"id","data_type":"bigint","is_primary":true},{"name":"amount","data_type":"decimal"}]}`
	code, response := do(t, http.MethodPost, srv.URL+"/api/batches/"+itoa(batchID)+"/tables", body)
	if code != http.StatusCreated {
		t.Fatalf("ingest table: want 201 got %d body=%s", code, response)
	}
	var table struct {
		ID int64 `json:"TableID"`
	}
	if err := decodeJSON(response, &table); err != nil || table.ID == 0 {
		t.Fatalf("ingest response missing table id: %s", response)
	}

	code, response = do(t, http.MethodGet, srv.URL+"/api/batches/"+itoa(batchID)+"/tables", "")
	if code != http.StatusOK || len(response) == 0 {
		t.Fatalf("list tables: want 200 with body, got %d body=%s", code, response)
	}
	code, response = do(t, http.MethodGet, srv.URL+"/api/batches/"+itoa(batchID)+"/columns?table_id="+itoa(table.ID), "")
	if code != http.StatusOK || len(response) == 0 {
		t.Fatalf("list columns: want 200 with body, got %d body=%s", code, response)
	}
	code, _ = do(t, http.MethodGet, srv.URL+"/api/batches/"+itoa(batchID)+"/columns", "")
	if code != http.StatusBadRequest {
		t.Fatalf("missing table_id: want 400 got %d", code)
	}
}
