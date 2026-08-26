package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func createBatch(t *testing.T, srv *httptest.Server, name string) int64 {
	t.Helper()
	code, body := do(t, http.MethodPost, srv.URL+"/api/batches", `{"name":"`+name+`"}`)
	if code != http.StatusCreated {
		t.Fatalf("create batch: want 201 got %d body=%s", code, body)
	}
	var batch struct {
		ID int64 `json:"id"`
	}
	if err := decodeJSON(body, &batch); err != nil || batch.ID == 0 {
		t.Fatalf("invalid batch response: %s", body)
	}
	return batch.ID
}

func ingestTable(t *testing.T, srv *httptest.Server, batchID int64, qualifiedName, version, columns string) {
	t.Helper()
	body := `{"qualified_name":"` + qualifiedName + `","schema_version":"` + version + `","columns":` + columns + `}`
	code, response := do(t, http.MethodPost, srv.URL+"/api/batches/"+itoa(batchID)+"/tables", body)
	if code != http.StatusCreated {
		t.Fatalf("ingest %s %s: want 201 got %d body=%s", qualifiedName, version, code, response)
	}
}

func createBrokenBatch(t *testing.T, srv *httptest.Server) int64 {
	t.Helper()
	batchID := createBatch(t, srv, "broken-edge")
	ingestTable(t, srv, batchID, "db.s.src", "v1", `[{"name":"id","data_type":"bigint"},{"name":"amount","data_type":"decimal"}]`)
	ingestTable(t, srv, batchID, "db.s.src", "v2", `[{"name":"id","data_type":"bigint"},{"name":"amt","data_type":"decimal"}]`)
	ingestTable(t, srv, batchID, "db.s.tgt", "v1", `[{"name":"id","data_type":"bigint"},{"name":"amount","data_type":"decimal"}]`)
	code, body := do(t, http.MethodPost, srv.URL+"/api/batches/"+itoa(batchID)+"/transforms",
		`{"source_table":"db.s.src","source_column":"amount","target_table":"db.s.tgt","target_column":"amount","expression":"tgt.amount=src.amount"}`)
	if code != http.StatusCreated {
		t.Fatalf("add broken transform: want 201 got %d body=%s", code, body)
	}
	code, body = do(t, http.MethodPost, srv.URL+"/api/batches/"+itoa(batchID)+"/build", "")
	if code != http.StatusOK {
		t.Fatalf("build broken batch: want 200 got %d body=%s", code, body)
	}
	return batchID
}

func decodeJSON(body []byte, dst interface{}) error {
	return json.Unmarshal(body, dst)
}

func contains(body []byte, fragment string) bool {
	return bytes.Contains(body, []byte(fragment))
}
