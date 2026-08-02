package subcmd

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/simons-agent-space/ghx/internal/api"
)

// TestPRListQueryIncludesAuthor verifies that the --author flag is
// translated into the GitHub REST API's `author` query parameter.
func TestPRListQueryIncludesAuthor(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	c := api.NewClientWithToken("test-token")
	c.BaseURL = srv.URL

	if err := PRList(context.Background(), c, []string{
		"--repo", "owner/repo",
		"--author", "alice",
		"--state", "all",
	}); err != nil {
		t.Fatalf("PRList: %v", err)
	}
	if !strings.Contains(gotQuery, "author=alice") {
		t.Errorf("query missing author=alice; got: %q", gotQuery)
	}
	if !strings.Contains(gotQuery, "state=all") {
		t.Errorf("query missing state=all; got: %q", gotQuery)
	}
}

// TestPRListQueryOmitsAuthorWhenUnset verifies that empty --author does
// not add an empty `author=` query parameter.
func TestPRListQueryOmitsAuthorWhenUnset(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	c := api.NewClientWithToken("test-token")
	c.BaseURL = srv.URL

	if err := PRList(context.Background(), c, []string{"--repo", "owner/repo"}); err != nil {
		t.Fatalf("PRList: %v", err)
	}
	if strings.Contains(gotQuery, "author=") {
		t.Errorf("query should not contain author when flag is unset; got: %q", gotQuery)
	}
}
