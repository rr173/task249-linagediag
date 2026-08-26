package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"task249-linagediag/internal/service"
	"task249-linagediag/internal/store"
)

func newTestServer(t *testing.T) (*httptest.Server, func()) {
	t.Helper()
	dir, err := os.MkdirTemp("", "linagediag-httptest-")
	if err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(dir, "t.db")
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(New(service.New(st)))
	return srv, func() {
		srv.Close()
		st.Close()
		os.RemoveAll(dir)
	}
}

func do(t *testing.T, method, url, body string) (int, []byte) {
	t.Helper()
	var r io.Reader
	if body != "" {
		r = bytes.NewBufferString(body)
	}
	req, err := http.NewRequest(method, url, r)
	if err != nil {
		t.Fatal(err)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, b
}

func TestHTTPLineageFlow(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	// health
	code, _ := do(t, http.MethodGet, srv.URL+"/api/health", "")
	if code != http.StatusOK {
		t.Fatalf("health: want 200 got %d", code)
	}

	// create batch
	code, b := do(t, http.MethodPost, srv.URL+"/api/batches", `{"name":"http-smoke"}`)
	if code != http.StatusCreated {
		t.Fatalf("create batch: want 201 got %d body=%s", code, b)
	}
	var batch struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(b, &batch); err != nil || batch.ID == 0 {
		t.Fatalf("bad batch resp: %s", b)
	}
	bid := batch.ID

	// ingest tables
	ingest := func(qn, sv string, cols string) {
		body := `{"qualified_name":"` + qn + `","schema_version":"` + sv + `","columns":` + cols + `}`
		c, bb := do(t, http.MethodPost, srv.URL+"/api/batches/"+itoa(bid)+"/tables", body)
		if c != http.StatusCreated {
			t.Fatalf("ingest %s: want 201 got %d body=%s", qn, c, bb)
		}
	}
	ingest("db.s.src", "v1", `[{"name":"id","data_type":"bigint","is_primary":true},{"name":"amount","data_type":"decimal"}]`)
	ingest("db.s.src", "v2", `[{"name":"id","data_type":"bigint","is_primary":true},{"name":"amt","data_type":"decimal"}]`)
	ingest("db.s.tgt", "v1", `[{"name":"id","data_type":"bigint","is_primary":true},{"name":"amount","data_type":"decimal"}]`)

	// add transform referencing renamed-away column
	c, bb := do(t, http.MethodPost, srv.URL+"/api/batches/"+itoa(bid)+"/transforms",
		`{"source_table":"db.s.src","source_column":"amount","target_table":"db.s.tgt","target_column":"amount","expression":"tgt.amount=src.amount"}`)
	if c != http.StatusCreated {
		t.Fatalf("add transform: want 201 got %d body=%s", c, bb)
	}

	// build
	c, bb = do(t, http.MethodPost, srv.URL+"/api/batches/"+itoa(bid)+"/build", "")
	if c != http.StatusOK {
		t.Fatalf("build: want 200 got %d body=%s", c, bb)
	}

	// diagnose -> expect broken
	c, bb = do(t, http.MethodGet, srv.URL+"/api/batches/"+itoa(bid)+"/diagnose", "")
	if c != http.StatusOK {
		t.Fatalf("diagnose: want 200 got %d body=%s", c, bb)
	}
	var findings []struct {
		Broken bool `json:"broken"`
	}
	if err := json.Unmarshal(bb, &findings); err != nil {
		t.Fatalf("diagnose decode: %v", err)
	}
	if len(findings) == 0 || !findings[0].Broken {
		t.Fatalf("expected at least one broken finding, got %s", bb)
	}

	// scenario -> sealed
	c, bb = do(t, http.MethodPost, srv.URL+"/api/batches/"+itoa(bid)+"/scenario", "")
	if c != http.StatusOK {
		t.Fatalf("scenario: want 200 got %d body=%s", c, bb)
	}
	var scen struct {
		BrokenBefore int  `json:"broken_before"`
		BrokenAfter  int  `json:"broken_after"`
		Sealed       bool `json:"sealed"`
	}
	if err := json.Unmarshal(bb, &scen); err != nil {
		t.Fatalf("scenario decode: %v", err)
	}
	if scen.BrokenBefore == 0 || scen.BrokenAfter != 0 || !scen.Sealed {
		t.Fatalf("scenario unexpected: %+v", scen)
	}

	// get batch -> sealed_at present
	c, bb = do(t, http.MethodGet, srv.URL+"/api/batches/"+itoa(bid), "")
	if c != http.StatusOK {
		t.Fatalf("get batch: want 200 got %d", c)
	}
	var got struct {
		SealedAt *int64 `json:"sealed_at"`
	}
	if err := json.Unmarshal(bb, &got); err != nil {
		t.Fatalf("get batch decode: %v", err)
	}
	if got.SealedAt == nil {
		t.Fatalf("expected sealed_at after scenario, got %s", bb)
	}

	// list snapshots
	c, bb = do(t, http.MethodGet, srv.URL+"/api/batches/"+itoa(bid)+"/snapshots", "")
	if c != http.StatusOK {
		t.Fatalf("list snapshots: want 200 got %d body=%s", c, bb)
	}
}

func itoa(v int64) string {
	return strconv.FormatInt(v, 10)
}
