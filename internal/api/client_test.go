package api

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestExtractToken(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"valid", "https://x-access-token:ghs_abc123@github.com/owner/repo.git", "ghs_abc123", false},
		{"missing prefix", "https://user:pass@github.com/owner/repo.git", "", true},
		{"missing @", "https://x-access-token:ghs_abc123", "", true},
		{"empty token", "https://x-access-token:@github.com/owner/repo.git", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ExtractToken(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got token %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestClientRequestHeaders(t *testing.T) {
	var (
		gotMethod, gotPath, gotAuth, gotUA, gotCT, gotAPIVer string
		gotBody                                              []byte
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		gotUA = r.Header.Get("User-Agent")
		gotCT = r.Header.Get("Content-Type")
		gotAPIVer = r.Header.Get("X-GitHub-Api-Version")
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(200)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := NewClientWithToken("ghs_test_token")
	c.BaseURL = srv.URL
	body := map[string]string{"hello": "world"}
	var out map[string]any
	if err := c.Post(context.Background(), "/test", body, &out); err != nil {
		t.Fatalf("post: %v", err)
	}
	if gotMethod != "POST" {
		t.Errorf("method = %s, want POST", gotMethod)
	}
	if gotPath != "/test" {
		t.Errorf("path = %s, want /test", gotPath)
	}
	if gotAuth != "token ghs_test_token" {
		t.Errorf("auth = %q, want token ghs_test_token", gotAuth)
	}
	if gotUA != UserAgent {
		t.Errorf("ua = %q, want %q", gotUA, UserAgent)
	}
	if gotCT != "application/json" {
		t.Errorf("content-type = %q, want application/json", gotCT)
	}
	if gotAPIVer == "" {
		t.Error("X-GitHub-Api-Version header missing")
	}
	if !strings.Contains(string(gotBody), `"hello":"world"`) {
		t.Errorf("body = %q, missing hello:world", string(gotBody))
	}
	if out["ok"] != true {
		t.Errorf("out = %v, want ok:true", out)
	}
}

func TestClientErrorMapping(t *testing.T) {
	cases := []struct {
		name       string
		statusCode int
		wantErr    error
	}{
		{"401 -> ErrAuthFailed", 401, ErrAuthFailed},
		{"404 -> ErrNotFound", 404, ErrNotFound},
		{"500 -> ErrAPIError", 500, ErrAPIError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
				w.Write([]byte(`{"message":"oops"}`))
			}))
			defer srv.Close()
			c := NewClientWithToken("x")
			c.BaseURL = srv.URL
			err := c.Get(context.Background(), "/x", nil)
			if err == nil {
				t.Fatal("expected error")
			}
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("err = %v, want errors.Is(_, %v)", err, tc.wantErr)
			}
		})
	}
}

func TestClientAbsoluteURLBypassesBase(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(200)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()
	c := NewClientWithToken("x")
	c.BaseURL = "https://api.github.com"
	var out map[string]any
	if err := c.Get(context.Background(), srv.URL+"/abs", &out); err != nil {
		t.Fatalf("get: %v", err)
	}
	if gotPath != "/abs" {
		t.Errorf("path = %s, want /abs", gotPath)
	}
}

func TestAPIErrorMessage(t *testing.T) {
	e := &APIError{StatusCode: 500, Method: "GET", Path: "/x", Body: "boom"}
	msg := e.Error()
	if !strings.Contains(msg, "500") || !strings.Contains(msg, "/x") {
		t.Errorf("message missing details: %q", msg)
	}
	if !errors.Is(e, ErrAPIError) {
		t.Error("APIError should unwrap to ErrAPIError")
	}
}
