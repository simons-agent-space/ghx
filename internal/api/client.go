// Package api implements a minimal GitHub REST client for ghx.
//
// The client reads a short-lived installation token from
// /tmp/gitbridge-credentials, populated by
// /workspace/bin/gitbridge-auth OWNER/REPO.
//
// Only stdlib net/http + encoding/json are used. No external deps.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// Sentinel errors. Callers can errors.Is against these.
var (
	// ErrCredentialsMissing is returned when the credentials file is
	// missing or unreadable. Callers should run
	// /workspace/bin/gitbridge-auth OWNER/REPO to refresh it.
	ErrCredentialsMissing = errors.New("credentials missing; run /workspace/bin/gitbridge-auth OWNER/REPO first")

	// ErrCredentialsMalformed is returned when the credentials file
	// does not contain a usable x-access-token entry.
	ErrCredentialsMalformed = errors.New("credentials file malformed; expected https://x-access-token:TOKEN@github.com/owner/repo.git")

	// ErrAuthFailed is returned for HTTP 401.
	ErrAuthFailed = errors.New("github API authentication failed")

	// ErrNotFound is returned for HTTP 404.
	ErrNotFound = errors.New("github API resource not found")

	// ErrAPIError is returned for any other non-2xx response.
	// The underlying *APIError carries the status code + body.
	ErrAPIError = errors.New("github API error")
)

// CredentialsPath is the well-known gitbridge credentials file.
const CredentialsPath = "/tmp/gitbridge-credentials"

// UserAgent is sent on every request.
const UserAgent = "ghx/0.1"

// Client is a GitHub REST client. Create one with NewClient.
type Client struct {
	BaseURL string
	Token   string
	HTTP    *http.Client
}

// NewClient reads the credentials file and returns a Client.
func NewClient() (*Client, error) {
	token, err := readToken()
	if err != nil {
		return nil, err
	}
	return NewClientWithToken(token), nil
}

// NewClientWithToken returns a Client using the supplied token.
// Tests use this with a fake BaseURL.
func NewClientWithToken(token string) *Client {
	return &Client{
		BaseURL: "https://api.github.com",
		Token:   token,
		HTTP:    &http.Client{Timeout: 30 * time.Second},
	}
}

// readToken reads the token from the credentials file.
func readToken() (string, error) {
	data, err := os.ReadFile(CredentialsPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("%w: %s not found", ErrCredentialsMissing, CredentialsPath)
		}
		return "", fmt.Errorf("%w: %v", ErrCredentialsMissing, err)
	}
	tok, err := ExtractToken(string(data))
	if err != nil {
		return "", err
	}
	return tok, nil
}

// ExtractToken parses a token from a gitbridge credentials URL.
//
// Format: https://x-access-token:TOKEN@github.com/owner/repo.git
//
// Exported so tests cover it without touching the filesystem.
func ExtractToken(s string) (string, error) {
	const prefix = "x-access-token:"
	i := strings.Index(s, prefix)
	if i < 0 {
		return "", ErrCredentialsMalformed
	}
	rest := s[i+len(prefix):]
	j := strings.Index(rest, "@")
	if j < 0 {
		return "", ErrCredentialsMalformed
	}
	tok := rest[:j]
	if tok == "" {
		return "", ErrCredentialsMalformed
	}
	return tok, nil
}

// Get performs a GET. If out is non-nil, the body is JSON-decoded into it.
func (c *Client) Get(ctx context.Context, path string, out any) error {
	return c.do(ctx, http.MethodGet, path, nil, out)
}

// Post performs a POST with a JSON body.
func (c *Client) Post(ctx context.Context, path string, body, out any) error {
	return c.do(ctx, http.MethodPost, path, body, out)
}

// Patch performs a PATCH with a JSON body.
func (c *Client) Patch(ctx context.Context, path string, body, out any) error {
	return c.do(ctx, http.MethodPatch, path, body, out)
}

// Put performs a PUT with a JSON body.
func (c *Client) Put(ctx context.Context, path string, body, out any) error {
	return c.do(ctx, http.MethodPut, path, body, out)
}

// Delete performs a DELETE. body may be nil.
func (c *Client) Delete(ctx context.Context, path string, body, out any) error {
	return c.do(ctx, http.MethodDelete, path, body, out)
}

// APIError is the typed error returned for non-2xx responses other
// than 401/404. It unwraps to ErrAPIError so errors.Is works.
type APIError struct {
	StatusCode int
	Method     string
	Path       string
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("%s %s: status %d: %s", e.Method, e.Path, e.StatusCode, truncate(e.Body, 500))
}

func (e *APIError) Unwrap() error { return ErrAPIError }

func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	url := path
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = c.BaseURL + path
	}
	// Defense in depth: only attach the Authorization header when the
	// resolved URL is under BaseURL. The api subcommand validates paths
	// before calling, but this prevents token exfiltration even if a
	// caller (or future code) bypasses that validation.
	attachAuth := !strings.HasPrefix(path, "http://") && !strings.HasPrefix(path, "https://")
	if !attachAuth {
		base := strings.TrimSuffix(c.BaseURL, "/")
		attachAuth = strings.HasPrefix(url, base)
	}
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal body: %w", err)
		}
		bodyReader = strings.NewReader(string(b))
	}
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if attachAuth {
		req.Header.Set("Authorization", "token "+c.Token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode == 401 {
		return fmt.Errorf("%w: status 401: %s", ErrAuthFailed, truncate(string(respBody), 500))
	}
	if resp.StatusCode == 404 {
		return fmt.Errorf("%w: status 404: %s", ErrNotFound, truncate(string(respBody), 500))
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &APIError{
			StatusCode: resp.StatusCode,
			Method:     method,
			Path:       path,
			Body:       string(respBody),
		}
	}
	if out != nil {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("unmarshal response: %w", err)
		}
	}
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
