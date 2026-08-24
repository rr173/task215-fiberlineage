package httpapi

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"task215-fiberlineage/internal/service"
	"task215-fiberlineage/internal/store"
	"testing"
)

func TestHandlerServesResearchPageAndHealthAPI(t *testing.T) {
	st, err := store.Open(t.TempDir() + "/web.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if err := st.Migrate(); err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(NewServer(service.New(st)).Handler())
	t.Cleanup(ts.Close)
	page, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer page.Body.Close()
	if page.StatusCode != http.StatusOK {
		t.Fatalf("page status = %d", page.StatusCode)
	}
	body, err := io.ReadAll(page.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "古籍纸张纤维谱系比对台") {
		t.Fatal("research page title missing")
	}
	health, err := http.Get(ts.URL + "/api/health")
	if err != nil {
		t.Fatal(err)
	}
	defer health.Body.Close()
	if health.StatusCode != http.StatusOK {
		t.Fatalf("health status = %d", health.StatusCode)
	}
}
