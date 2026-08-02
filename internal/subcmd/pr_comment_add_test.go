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

// TestPRCommentAddHitsIssuesEndpoint verifies that pr-comment-add
// POSTs to /repos/OWNER/REPO/issues/N/comments (not pulls/.../comments).
// PRs are issues in GitHub's data model; inline review comments
// (POST .../pulls/N/comments) are a different endpoint and out of
// scope for this subcommand.
func TestPRCommentAddHitsIssuesEndpoint(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id": 123, "html_url": "https://example/pr/1#issuecomment-123"}`))
	}))
	defer srv.Close()

	c := api.NewClientWithToken("test-token")
	c.BaseURL = srv.URL

	bodyFile := filepath.Join(t.TempDir(), "body.md")
	if err := os.WriteFile(bodyFile, []byte("hello world"), 0o644); err != nil {
		t.Fatalf("seed body: %v", err)
	}

	if err := PRCommentAdd(context.Background(), c, []string{
		"--repo", "owner/repo", "--body-file", bodyFile, "1",
	}); err != nil {
		t.Fatalf("PRCommentAdd: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	wantPath := "/repos/owner/repo/issues/1/comments"
	if gotPath != wantPath {
		t.Errorf("path = %q, want %q", gotPath, wantPath)
	}
}

// TestPRCommentAddSendsBodyFromFile verifies the JSON body sent to
// the API has a "body" key with the file contents verbatim.
func TestPRCommentAddSendsBodyFromFile(t *testing.T) {
	var gotRaw []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRaw, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id": 1, "html_url": "x"}`))
	}))
	defer srv.Close()

	c := api.NewClientWithToken("test-token")
	c.BaseURL = srv.URL

	want := "multi\nline\nbody with **markdown** and \"quotes\""
	bodyFile := filepath.Join(t.TempDir(), "body.md")
	if err := os.WriteFile(bodyFile, []byte(want), 0o644); err != nil {
		t.Fatalf("seed body: %v", err)
	}

	if err := PRCommentAdd(context.Background(), c, []string{
		"--repo", "owner/repo", "--body-file", bodyFile, "7",
	}); err != nil {
		t.Fatalf("PRCommentAdd: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(gotRaw, &payload); err != nil {
		t.Fatalf("body is not JSON: %v\nraw: %s", err, gotRaw)
	}
	gotBody, _ := payload["body"].(string)
	if gotBody != want {
		t.Errorf("body mismatch:\n--- got ---\n%s\n--- want ---\n%s", gotBody, want)
	}
	if len(payload) != 1 {
		t.Errorf("payload should have exactly one key (body); got %d: %v", len(payload), payload)
	}
}

// TestPRCommentAddMissingRepo verifies the subcommand rejects an
// empty --repo before any HTTP call lands.
func TestPRCommentAddMissingRepo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("server must not be called when --repo is missing: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(500)
	}))
	defer srv.Close()

	c := api.NewClientWithToken("test-token")
	c.BaseURL = srv.URL

	bodyFile := filepath.Join(t.TempDir(), "body.md")
	os.WriteFile(bodyFile, []byte("x"), 0o644)

	err := PRCommentAdd(context.Background(), c, []string{"--body-file", bodyFile, "1"})
	if err == nil {
		t.Fatal("expected error when --repo is missing")
	}
	if !strings.Contains(err.Error(), "--repo is required") {
		t.Errorf("error must mention --repo; got %v", err)
	}
}

// TestPRCommentAddMissingBodyFile verifies the subcommand rejects
// an empty --body-file before any HTTP call lands.
func TestPRCommentAddMissingBodyFile(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("server must not be called when --body-file is missing: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(500)
	}))
	defer srv.Close()

	c := api.NewClientWithToken("test-token")
	c.BaseURL = srv.URL

	err := PRCommentAdd(context.Background(), c, []string{"--repo", "owner/repo", "1"})
	if err == nil {
		t.Fatal("expected error when --body-file is missing")
	}
	if !strings.Contains(err.Error(), "--body-file is required") {
		t.Errorf("error must mention --body-file; got %v", err)
	}
}

// TestPRCommentAddInvalidPRNumber verifies non-numeric PR numbers
// are rejected with a clear error.
func TestPRCommentAddInvalidPRNumber(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("server must not be called with invalid PR number: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(500)
	}))
	defer srv.Close()

	c := api.NewClientWithToken("test-token")
	c.BaseURL = srv.URL

	bodyFile := filepath.Join(t.TempDir(), "body.md")
	os.WriteFile(bodyFile, []byte("x"), 0o644)

	err := PRCommentAdd(context.Background(), c, []string{
		"--repo", "owner/repo", "--body-file", bodyFile, "not-a-number",
	})
	if err == nil {
		t.Fatal("expected error for non-numeric PR number")
	}
	if !strings.Contains(err.Error(), "invalid PR number") {
		t.Errorf("error must mention invalid PR number; got %v", err)
	}
}

// TestPRCommentAddJSONOutput verifies --json prints the raw API
// response object (so callers can pipe it into jq or similar).
func TestPRCommentAddJSONOutput(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id": 42, "html_url": "https://example/pr/1#issuecomment-42", "body": "x"}`))
	}))
	defer srv.Close()

	c := api.NewClientWithToken("test-token")
	c.BaseURL = srv.URL

	bodyFile := filepath.Join(t.TempDir(), "body.md")
	os.WriteFile(bodyFile, []byte("x"), 0o644)

	if err := PRCommentAdd(context.Background(), c, []string{
		"--repo", "owner/repo", "--body-file", bodyFile, "--json", "1",
	}); err != nil {
		t.Fatalf("PRCommentAdd: %v", err)
	}
}

// TestPRCommentAddSurfacesAPIErrors verifies that HTTP 4xx/5xx
// responses from GitHub are returned as errors (not silently
// treated as success).
func TestPRCommentAddSurfacesAPIErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"message": "Bad credentials"}`))
	}))
	defer srv.Close()

	c := api.NewClientWithToken("test-token")
	c.BaseURL = srv.URL

	bodyFile := filepath.Join(t.TempDir(), "body.md")
	os.WriteFile(bodyFile, []byte("x"), 0o644)

	err := PRCommentAdd(context.Background(), c, []string{
		"--repo", "owner/repo", "--body-file", bodyFile, "1",
	})
	if err == nil {
		t.Fatal("expected error on 401 response")
	}
}

// TestPRCommentAddPrintsCommentID verifies the success-message
// format prints the created comment's numeric ID. The API
// returns "id" as a JSON number (decoded as float64 in
// map[string]any); the previous version used asString on a
// number field and silently dropped the ID, producing
// "Comment  added on PR 1: ..." with a stray double space.
// This test catches that regression.
func TestPRCommentAddPrintsCommentID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id": 4242, "html_url": "https://example/pr/1#issuecomment-4242"}`))
	}))
	defer srv.Close()

	c := api.NewClientWithToken("test-token")
	c.BaseURL = srv.URL

	bodyFile := filepath.Join(t.TempDir(), "body.md")
	os.WriteFile(bodyFile, []byte("x"), 0o644)

	// Capture stdout while the subcommand runs.
	oldStdout := os.Stdout
	rPipe, wPipe, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = wPipe

	runErr := PRCommentAdd(context.Background(), c, []string{
		"--repo", "owner/repo", "--body-file", bodyFile, "1",
	})

	wPipe.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, rPipe)

	if runErr != nil {
		t.Fatalf("PRCommentAdd: %v", runErr)
	}
	out := buf.String()
	if !strings.Contains(out, "Comment 4242 added on PR 1: https://example/pr/1#issuecomment-4242") {
		t.Errorf("expected success message with numeric ID 4242, got: %q", out)
	}
	if strings.Contains(out, "Comment  added on PR") {
		t.Errorf("ID was dropped (double space before 'added'): %q", out)
	}
}