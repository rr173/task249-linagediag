package httpapi

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestTask249Bug04ImpactIncludesTransitiveDownstream(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	batchID := createBatch(t, srv, "transitive-impact")
	for _, table := range []string{"db.s.source", "db.s.middle", "db.s.target"} {
		ingestTable(t, srv, batchID, table, "v1", `[{"name":"amount","data_type":"decimal"}]`)
	}
	addTransform := func(source, target string) {
		code, body := do(t, http.MethodPost, srv.URL+"/api/batches/"+itoa(batchID)+"/transforms",
			`{"source_table":"`+source+`","source_column":"amount","target_table":"`+target+`","target_column":"amount","expression":"`+target+`.amount=`+source+`.amount"}`)
		if code != http.StatusCreated {
			t.Fatalf("add transform: want 201 got %d body=%s", code, body)
		}
	}
	addTransform("db.s.source", "db.s.middle")
	addTransform("db.s.middle", "db.s.target")
	code, body := do(t, http.MethodPost, srv.URL+"/api/batches/"+itoa(batchID)+"/build", "")
	if code != http.StatusOK {
		t.Fatalf("build: want 200 got %d body=%s", code, body)
	}
	code, body = do(t, http.MethodGet, srv.URL+"/api/batches/"+itoa(batchID)+"/impact?table=db.s.source&column=amount", "")
	if code != http.StatusOK {
		t.Fatalf("impact: want 200 got %d body=%s", code, body)
	}
	var refs []struct {
		Table string `json:"Table"`
	}
	if err := json.Unmarshal(body, &refs); err != nil {
		t.Fatal(err)
	}
	if len(refs) != 2 {
		t.Fatalf("impact must include both direct and transitive downstream columns: %s", body)
	}
}
