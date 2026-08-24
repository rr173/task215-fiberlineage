package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"task215-fiberlineage/internal/service"
	"task215-fiberlineage/internal/store"
)

func TestBug08_MissingSampleMapsToNotFound(t *testing.T) {
	st, err := store.Open(t.TempDir() + "/http.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	ts := httptest.NewServer(NewServer(service.New(st)).Handler())
	t.Cleanup(ts.Close)
	resp, err := http.Get(ts.URL + "/api/samples/999")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("missing sample status = %d, want 404", resp.StatusCode)
	}
}
