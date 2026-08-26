package httpapi

import (
	"net/http"
	"testing"
)

func TestHTTPBatchLifecycleAndInvalidIdentifiers(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	code, body := do(t, http.MethodGet, srv.URL+"/api/batches", "")
	if code != http.StatusOK {
		t.Fatalf("list batches: want 200 got %d body=%s", code, body)
	}
	code, body = do(t, http.MethodPost, srv.URL+"/api/batches", `{"name":"batch-api"}`)
	if code != http.StatusCreated {
		t.Fatalf("create batch: want 201 got %d body=%s", code, body)
	}
	var created struct {
		ID int64 `json:"id"`
	}
	if err := decodeJSON(body, &created); err != nil || created.ID == 0 {
		t.Fatalf("create response missing id: %s", body)
	}

	code, body = do(t, http.MethodGet, srv.URL+"/api/batches/"+itoa(created.ID), "")
	if code != http.StatusOK {
		t.Fatalf("get batch: want 200 got %d body=%s", code, body)
	}
	code, _ = do(t, http.MethodGet, srv.URL+"/api/batches/not-an-id", "")
	if code != http.StatusBadRequest {
		t.Fatalf("invalid batch id: want 400 got %d", code)
	}
	code, _ = do(t, http.MethodGet, srv.URL+"/api/batches/999999", "")
	if code != http.StatusNotFound {
		t.Fatalf("missing batch: want 404 got %d", code)
	}
}
