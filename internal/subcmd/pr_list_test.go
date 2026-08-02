package subcmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestPrintPRListEmpty(t *testing.T) {
	var buf bytes.Buffer
	printPRList(&buf, nil)
	if !strings.Contains(buf.String(), "(no pull requests)") {
		t.Errorf("expected empty marker, got: %s", buf.String())
	}
}

func TestPrintPRList(t *testing.T) {
	prs := []map[string]any{
		{
			"number":   float64(10),
			"title":    "feat: A",
			"state":    "open",
			"draft":    true,
			"html_url": "https://github.com/x/y/pull/10",
			"head":     map[string]any{"ref": "feature-a"},
			"base":     map[string]any{"ref": "main"},
		},
		{
			"number":   float64(11),
			"title":    "fix: B",
			"state":    "closed",
			"html_url": "https://github.com/x/y/pull/11",
			"head":     map[string]any{"ref": "feature-b"},
			"base":     map[string]any{"ref": "main"},
		},
	}
	var buf bytes.Buffer
	printPRList(&buf, prs)
	out := buf.String()
	for _, want := range []string{
		"#10",
		"open (draft)",
		"feature-a",
		"feat: A",
		"#11",
		"closed",
		"feature-b",
		"fix: B",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\ngot:\n%s", want, out)
		}
	}
}
