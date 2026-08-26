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

// TestHTTPCrossBatchEdgeRejected 验证批次边界校验：在批次 A 上传入批次 B 的边编号，
// confirm/exempt/revise 均应被拒绝，且批次 B 的诊断结果保持不变。
func TestHTTPCrossBatchEdgeRejected(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	// 两个相互独立的批次，各含一条断裂边。
	batchA := createBrokenBatch(t, srv)
	batchB := createBrokenBatch(t, srv)

	// 取批次 A 的断裂边编号。
	code, body := do(t, http.MethodGet, srv.URL+"/api/batches/"+itoa(batchA)+"/diagnose", "")
	if code != http.StatusOK {
		t.Fatalf("diagnose A: want 200 got %d body=%s", code, body)
	}
	var findings []struct {
		EdgeID int64 `json:"edge_id"`
		Broken bool  `json:"broken"`
	}
	if err := json.Unmarshal(body, &findings); err != nil || len(findings) != 1 || !findings[0].Broken {
		t.Fatalf("expected one broken finding in A: %s", body)
	}
	crossEdge := findings[0].EdgeID

	// confirm: 在批次 B 上对 A 的边做裁决 -> 必须被拒绝。
	code, _ = do(t, http.MethodPost, srv.URL+"/api/batches/"+itoa(batchB)+"/edges/"+itoa(crossEdge)+"/confirm", `{"note":"cross"}`)
	if code < 400 {
		t.Fatalf("cross-batch confirm should be rejected: code=%d", code)
	}
	// exempt: 同样被拒绝。
	code, _ = do(t, http.MethodPost, srv.URL+"/api/batches/"+itoa(batchB)+"/edges/"+itoa(crossEdge)+"/exempt", `{"note":"cross"}`)
	if code < 400 {
		t.Fatalf("cross-batch exempt should be rejected: code=%d", code)
	}
	// revise: 越过参数校验后仍因边界校验被拒。
	code, _ = do(t, http.MethodPost, srv.URL+"/api/batches/"+itoa(batchB)+"/edges/"+itoa(crossEdge)+"/revise",
		`{"new_source_table":"db.s.src","new_source_column":"amt","note":"cross"}`)
	if code < 400 {
		t.Fatalf("cross-batch revise should be rejected: code=%d", code)
	}

	// 批次 B 的诊断结果应保持原状：仍有一条断裂边，未被跨批次裁决改动。
	code, body = do(t, http.MethodGet, srv.URL+"/api/batches/"+itoa(batchB)+"/diagnose", "")
	if code != http.StatusOK {
		t.Fatalf("re-diagnose B: want 200 got %d body=%s", code, body)
	}
	if err := json.Unmarshal(body, &findings); err != nil || len(findings) != 1 || !findings[0].Broken {
		t.Fatalf("batch B diagnosis should remain broken and unchanged: %s", body)
	}
	// 批次 B 不应残留任何裁决记录（confirm/exempt 均被拒）。
	code, body = do(t, http.MethodGet, srv.URL+"/api/batches/"+itoa(batchB)+"/adjudications", "")
	if code != http.StatusOK {
		t.Fatalf("list B adjudications: want 200 got %d body=%s", code, body)
	}
	if contains(body, `"action"`) {
		t.Fatalf("batch B should have no adjudications: %s", body)
	}
}
