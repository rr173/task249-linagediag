package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// tableIDFor 登记一张表版本并解析其返回的表 ID（供作业的输入/输出表引用）。
func tableIDFor(t *testing.T, srv *httptest.Server, batchID int64, qualifiedName, version, columns string) int64 {
	t.Helper()
	body := `{"qualified_name":"` + qualifiedName + `","schema_version":"` + version + `","columns":` + columns + `}`
	c, resp := do(t, http.MethodPost, srv.URL+"/api/batches/"+itoa(batchID)+"/tables", body)
	if c != http.StatusCreated {
		t.Fatalf("ingest %s: want 201 got %d body=%s", qualifiedName, c, resp)
	}
	var tbl struct {
		ID int64 `json:"TableID"`
	}
	if err := json.Unmarshal(resp, &tbl); err != nil || tbl.ID == 0 {
		t.Fatalf("ingest %s: bad table id in %s", qualifiedName, resp)
	}
	return tbl.ID
}

// TestHTTPInferPassthroughSkipsTypeIncompatible 验证：作业的输入表与输出表存在同名列时，
// 仅当类型兼容才自动推断直通边；同名但类型不兼容的列不应被连边。
func TestHTTPInferPassthroughSkipsTypeIncompatible(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	batchID := createBatch(t, srv, "passthrough-types")

	// 输入表：id(bigint) 与 note(string)。
	inID := tableIDFor(t, srv, batchID, "db.s.in", "v1",
		`[{"name":"id","data_type":"bigint","is_primary":true},{"name":"note","data_type":"string"}]`)
	// 输出表：id(bigint) 与 note(int) —— note 同名但跨类型不相容。
	outID := tableIDFor(t, srv, batchID, "db.s.out", "v1",
		`[{"name":"id","data_type":"bigint","is_primary":true},{"name":"note","data_type":"int"}]`)

	// 登记作业：以 in 为输入、out 为输出，触发同名直通推断。
	code, body := do(t, http.MethodPost, srv.URL+"/api/batches/"+itoa(batchID)+"/jobs",
		`{"output_table_id":`+itoa(outID)+`,"name":"copy","summary":"","input_table_ids":[`+itoa(inID)+`]}`)
	if code != http.StatusCreated {
		t.Fatalf("add job: want 201 got %d body=%s", code, body)
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
		SourceColumn string `json:"source_column"`
		TargetColumn string `json:"target_column"`
		Status       string `json:"status"`
	}
	if err := json.Unmarshal(body, &edges); err != nil {
		t.Fatalf("lineage decode: %v body=%s", err, body)
	}

	var idEdge, noteEdge bool
	for _, e := range edges {
		if e.SourceColumn == "id" && e.TargetColumn == "id" {
			idEdge = true
		}
		if e.SourceColumn == "note" && e.TargetColumn == "note" {
			noteEdge = true
		}
	}
	if !idEdge {
		t.Fatalf("expected inferred passthrough edge for compatible column 'id', got %s", body)
	}
	if noteEdge {
		t.Fatalf("type-incompatible column 'note' (string->int) must not get a passthrough edge, got %s", body)
	}
}
