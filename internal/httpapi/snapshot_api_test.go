package httpapi

import (
	"net/http"
	"testing"
)

func TestHTTPSnapshotPublicationAndSealedMutation(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	batchID := createBatch(t, srv, "snapshot-api")
	code, body := do(t, http.MethodPost, srv.URL+"/api/batches/"+itoa(batchID)+"/snapshots", `{"note":"draft result"}`)
	if code != http.StatusCreated {
		t.Fatalf("publish snapshot: want 201 got %d body=%s", code, body)
	}
	code, body = do(t, http.MethodGet, srv.URL+"/api/batches/"+itoa(batchID)+"/snapshots", "")
	if code != http.StatusOK || !contains(body, `"published"`) {
		t.Fatalf("list snapshots: want published snapshot, got %d body=%s", code, body)
	}
	code, body = do(t, http.MethodPost, srv.URL+"/api/batches/"+itoa(batchID)+"/seal", "")
	if code != http.StatusOK {
		t.Fatalf("seal batch: want 200 got %d body=%s", code, body)
	}
	code, _ = do(t, http.MethodPost, srv.URL+"/api/batches/"+itoa(batchID)+"/tables", `{"qualified_name":"db.s.after","schema_version":"v1","columns":[]}`)
	if code != http.StatusConflict {
		t.Fatalf("sealed mutation: want 409 got %d", code)
	}
	// 作业登记同样受封存只读约束：封存后新增作业会导致封存结果与元数据不一致。
	code, _ = do(t, http.MethodPost, srv.URL+"/api/batches/"+itoa(batchID)+"/jobs", `{"output_table_id":0,"name":"late","summary":"","input_table_ids":[]}`)
	if code != http.StatusConflict {
		t.Fatalf("sealed mutation on jobs: want 409 got %d", code)
	}
}
