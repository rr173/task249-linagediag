package httpapi

import (
	"net/http"
	"testing"
)

// TestHTTPBuildRejectsCircularColumnDependency 锁定修复：
// 两张表的列变换形成循环依赖时，BuildLineage 必须拒绝构建（返回 409 Conflict），
// 而非误判成功导致后续诊断结论不可信。
//
// 修复前：HasCycle 的三色 DFS 因死代码无法深入 white 节点，
// 两节点及以上的环一律漏检 → build 返回 200。
func TestHTTPBuildRejectsCircularColumnDependency(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	batchID := createBatch(t, srv, "circular-dep")
	ingestTable(t, srv, batchID, "db.s.t1", "v1", `[{"name":"id","data_type":"bigint"},{"name":"col_a","data_type":"decimal"}]`)
	ingestTable(t, srv, batchID, "db.s.t2", "v1", `[{"name":"id","data_type":"bigint"},{"name":"col_b","data_type":"decimal"}]`)

	// t1.col_a 派生自 t2.col_b，t2.col_b 又派生自 t1.col_a → 形成环。
	add := func(srcTable, srcCol, tgtTable, tgtCol string) {
		code, body := do(t, http.MethodPost, srv.URL+"/api/batches/"+itoa(batchID)+"/transforms",
			`{"source_table":"`+srcTable+`","source_column":"`+srcCol+`","target_table":"`+tgtTable+`","target_column":"`+tgtCol+`","expression":"derived"}`)
		if code != http.StatusCreated {
			t.Fatalf("add transform %s.%s->%s.%s: want 201 got %d body=%s", srcTable, srcCol, tgtTable, tgtCol, code, body)
		}
	}
	add("db.s.t2", "col_b", "db.s.t1", "col_a")
	add("db.s.t1", "col_a", "db.s.t2", "col_b")

	code, body := do(t, http.MethodPost, srv.URL+"/api/batches/"+itoa(batchID)+"/build", "")
	if code != http.StatusConflict {
		t.Fatalf("build circular lineage: want 409 Conflict got %d body=%s", code, body)
	}
	if !contains(body, "circular") {
		t.Fatalf("expected error body to mention circular derivation, got %s", body)
	}
}
