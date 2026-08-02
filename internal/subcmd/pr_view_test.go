package subcmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestPrintPRView(t *testing.T) {
	raw := map[string]any{
		"number":     float64(10),
		"title":      "feat: something",
		"state":      "open",
		"draft":      true,
		"mergeable":  true,
		"merged":     false,
		"html_url":   "https://github.com/owner/repo/pull/10",
		"body":       "This is the body.",
		"created_at": "2026-01-01T00:00:00Z",
		"updated_at": "2026-01-02T00:00:00Z",
		"user":       map[string]any{"login": "alice"},
		"head":       map[string]any{"ref": "feature", "sha": "abc123def"},
		"base":       map[string]any{"ref": "main"},
	}
	var buf bytes.Buffer
	printPRView(&buf, raw, "owner/repo")
	out := buf.String()
	for _, want := range []string{
		"PR #10",
		"feat: something",
		"open (draft)",
		"alice",
		"feature -> main",
		"abc123def",
		"Mergeable: yes",
		"Merged:",
		"owner/repo",
		"This is the body.",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\ngot:\n%s", want, out)
		}
	}
}

func TestPrintPRViewMissingFields(t *testing.T) {
	// Empty map should not panic; every field defaults to zero value.
	raw := map[string]any{}
	var buf bytes.Buffer
	printPRView(&buf, raw, "owner/repo")
	out := buf.String()
	for _, want := range []string{
		"PR #0",
		"Mergeable: no",
		"Merged:",
		" -> ", // empty head -> base
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\ngot:\n%s", want, out)
		}
	}
}
