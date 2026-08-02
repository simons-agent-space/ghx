package subcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/simons-agent-space/ghx/internal/api"
)

// TestPRCreateHitsPullsEndpoint verifies that pr-create POSTs to
// /repos/OWNER/REPO/pulls (the "create a pull request" endpoint).
func TestPRCreateHitsPullsEndpoint(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"number": 99, "html_url": "https://example/pr/99"}`))
	}))
	defer srv.Close()

	c := api.NewClientWithToken("test-token")
	c.BaseURL = srv.URL

	if err := PRCreate(context.Background(), c, []string{
		"--repo", "owner/repo",
		"--head", "feature/x",
		"--title", "feat: something",
	}); err != nil {
		t.Fatalf("PRCreate: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	wantPath := "/repos/owner/repo/pulls"
	if gotPath != wantPath {
		t.Errorf("path = %q, want %q", gotPath, wantPath)
	}
}

// TestPRCreatePayload verifies the JSON payload contains every
// required field with the correct values, and that --base defaults
// to "main" when omitted.
func TestPRCreatePayload(t *testing.T) {
	var gotRaw []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRaw, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"number": 1, "html_url": "x"}`))
	}))
	defer srv.Close()

	c := api.NewClientWithToken("test-token")
	c.BaseURL = srv.URL

	if err := PRCreate(context.Background(), c, []string{
		"--repo", "owner/repo",
		"--head", "feature/x",
		"--title", "feat: something",
	}); err != nil {
		t.Fatalf("PRCreate: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(gotRaw, &payload); err != nil {
		t.Fatalf("body is not JSON: %v\nraw: %s", err, gotRaw)
	}
	if payload["title"] != "feat: something" {
		t.Errorf("title = %v, want 'feat: something'", payload["title"])
	}
	if payload["head"] != "feature/x" {
		t.Errorf("head = %v, want 'feature/x'", payload["head"])
	}
	if payload["base"] != "main" {
		t.Errorf("base = %v, want 'main' (default)", payload["base"])
	}
	if body, _ := payload["body"].(string); body != "" {
		t.Errorf("body = %q, want empty when --body-file is unset", body)
	}
	if draft, _ := payload["draft"].(bool); draft {
		t.Errorf("draft = true, want false (default)")
	}
}

// TestPRCreateExplicitBase verifies --base overrides the default.
func TestPRCreateExplicitBase(t *testing.T) {
	var gotRaw []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRaw, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"number": 1, "html_url": "x"}`))
	}))
	defer srv.Close()

	c := api.NewClientWithToken("test-token")
	c.BaseURL = srv.URL

	if err := PRCreate(context.Background(), c, []string{
		"--repo", "owner/repo",
		"--head", "feature/x",
		"--title", "feat",
		"--base", "release/2.0",
	}); err != nil {
		t.Fatalf("PRCreate: %v", err)
	}
	var payload map[string]any
	json.Unmarshal(gotRaw, &payload)
	if payload["base"] != "release/2.0" {
		t.Errorf("base = %v, want 'release/2.0'", payload["base"])
	}
}

// TestPRCreateBodyFromFile verifies --body-file is read verbatim
// and sent as the "body" field.
func TestPRCreateBodyFromFile(t *testing.T) {
	var gotRaw []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRaw, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"number": 1, "html_url": "x"}`))
	}))
	defer srv.Close()

	c := api.NewClientWithToken("test-token")
	c.BaseURL = srv.URL

	want := "multi\nline\nbody with **markdown** and \"quotes\""
	bodyFile := filepath.Join(t.TempDir(), "body.md")
	if err := os.WriteFile(bodyFile, []byte(want), 0o644); err != nil {
		t.Fatalf("seed body: %v", err)
	}

	if err := PRCreate(context.Background(), c, []string{
		"--repo", "owner/repo",
		"--head", "feature/x",
		"--title", "feat",
		"--body-file", bodyFile,
	}); err != nil {
		t.Fatalf("PRCreate: %v", err)
	}
	var payload map[string]any
	json.Unmarshal(gotRaw, &payload)
	if body, _ := payload["body"].(string); body != want {
		t.Errorf("body mismatch:\n--- got ---\n%s\n--- want ---\n%s", body, want)
	}
}

// TestPRCreateDraftFlag verifies --draft=true is sent through.
func TestPRCreateDraftFlag(t *testing.T) {
	var gotRaw []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRaw, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"number": 1, "html_url": "x"}`))
	}))
	defer srv.Close()

	c := api.NewClientWithToken("test-token")
	c.BaseURL = srv.URL

	if err := PRCreate(context.Background(), c, []string{
		"--repo", "owner/repo",
		"--head", "feature/x",
		"--title", "WIP",
		"--draft",
	}); err != nil {
		t.Fatalf("PRCreate: %v", err)
	}
	var payload map[string]any
	json.Unmarshal(gotRaw, &payload)
	if draft, _ := payload["draft"].(bool); !draft {
		t.Errorf("draft = false, want true when --draft is set")
	}
}

// TestPRCreateMissingRepo verifies --repo is required.
func TestPRCreateMissingRepo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("server must not be called when --repo is missing: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(500)
	}))
	defer srv.Close()

	c := api.NewClientWithToken("test-token")
	c.BaseURL = srv.URL

	err := PRCreate(context.Background(), c, []string{
		"--head", "feature/x", "--title", "feat",
	})
	if err == nil {
		t.Fatal("expected error when --repo is missing")
	}
	if !strings.Contains(err.Error(), "--repo is required") {
		t.Errorf("error must mention --repo; got %v", err)
	}
}

// TestPRCreateMissingHead verifies --head is required.
func TestPRCreateMissingHead(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("server must not be called when --head is missing: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(500)
	}))
	defer srv.Close()

	c := api.NewClientWithToken("test-token")
	c.BaseURL = srv.URL

	err := PRCreate(context.Background(), c, []string{
		"--repo", "owner/repo", "--title", "feat",
	})
	if err == nil {
		t.Fatal("expected error when --head is missing")
	}
	if !strings.Contains(err.Error(), "--head is required") {
		t.Errorf("error must mention --head; got %v", err)
	}
}

// TestPRCreateMissingTitle verifies --title is required.
func TestPRCreateMissingTitle(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("server must not be called when --title is missing: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(500)
	}))
	defer srv.Close()

	c := api.NewClientWithToken("test-token")
	c.BaseURL = srv.URL

	err := PRCreate(context.Background(), c, []string{
		"--repo", "owner/repo", "--head", "feature/x",
	})
	if err == nil {
		t.Fatal("expected error when --title is missing")
	}
	if !strings.Contains(err.Error(), "--title is required") {
		t.Errorf("error must mention --title; got %v", err)
	}
}

// TestPRCreateRejectsPositionalArgs verifies that pr-create (which
// has no positional args — every input is a flag) rejects any
// leftover positional argument with a clear error before any HTTP
// call lands.
func TestPRCreateRejectsPositionalArgs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("server must not be called with unexpected positional args: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(500)
	}))
	defer srv.Close()

	c := api.NewClientWithToken("test-token")
	c.BaseURL = srv.URL

	err := PRCreate(context.Background(), c, []string{
		"--repo", "owner/repo",
		"--head", "feature/x",
		"--title", "feat",
		"stray-positional",
	})
	if err == nil {
		t.Fatal("expected error for unexpected positional argument")
	}
	if !strings.Contains(err.Error(), "unexpected positional") {
		t.Errorf("error must mention unexpected positional; got %v", err)
	}
}

// TestPRCreateSurfacesAPIErrors verifies HTTP 4xx/5xx responses
// from GitHub are returned as errors (not silently treated as
// success).
func TestPRCreateSurfacesAPIErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(`{"message": "Validation Failed", "errors": [{"resource": "PullRequest", "code": "custom", "message": "head sha cannot be reached"}]}`))
	}))
	defer srv.Close()

	c := api.NewClientWithToken("test-token")
	c.BaseURL = srv.URL

	err := PRCreate(context.Background(), c, []string{
		"--repo", "owner/repo", "--head", "feature/x", "--title", "feat",
	})
	if err == nil {
		t.Fatal("expected error on 422 response")
	}
}

// TestPRCreateJSONOutput verifies --json prints the raw API
// response object decoded to stdout (so callers can pipe into jq
// or similar). Catches the same "returns no error but prints
// nothing" regression the pr-comment-add test caught.
func TestPRCreateJSONOutput(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"number": 99, "html_url": "https://example/pr/99", "title": "feat: x"}`))
	}))
	defer srv.Close()

	c := api.NewClientWithToken("test-token")
	c.BaseURL = srv.URL

	oldStdout := os.Stdout
	rPipe, wPipe, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = wPipe

	runErr := PRCreate(context.Background(), c, []string{
		"--repo", "owner/repo", "--head", "feature/x", "--title", "feat: x",
		"--json",
	})

	wPipe.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, rPipe)

	if runErr != nil {
		t.Fatalf("PRCreate: %v", runErr)
	}
	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\noutput: %s", err, buf.String())
	}
	if num, _ := got["number"].(float64); num != 99 {
		t.Errorf("json number = %v, want 99; full output: %s", got["number"], buf.String())
	}
	if url, _ := got["html_url"].(string); url != "https://example/pr/99" {
		t.Errorf("json html_url = %q, want %q", url, "https://example/pr/99")
	}
}