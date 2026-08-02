package subcmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestPrintCommentsEmpty(t *testing.T) {
	var buf bytes.Buffer
	printComments(&buf, nil)
	if buf.Len() != 0 {
		t.Errorf("expected empty output for no comments, got: %s", buf.String())
	}
}

func TestPrintCommentsAllKinds(t *testing.T) {
	all := []PRComment{
		{Kind: "issue", Author: "alice", Created: "2026-01-01T00:00:00Z", Body: "Looks good."},
		{Kind: "review", Author: "bob", Created: "2026-01-02T00:00:00Z", Body: "Approved.", State: "APPROVED"},
		{Kind: "inline", Author: "carol", Created: "2026-01-03T00:00:00Z", Body: "Rename this var.", Path: "foo.go", Line: float64(42)},
	}
	var buf bytes.Buffer
	printComments(&buf, all)
	out := buf.String()
	for _, want := range []string{
		"[issue] alice",
		"Looks good.",
		"[review] bob",
		"state=APPROVED",
		"Approved.",
		"[inline] carol",
		"foo.go:L42",
		"Rename this var.",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\ngot:\n%s", want, out)
		}
	}
}

func TestPrintCommentsEmptyBody(t *testing.T) {
	all := []PRComment{{Kind: "review", Author: "x", Created: "2026-01-01T00:00:00Z", Body: ""}}
	var buf bytes.Buffer
	printComments(&buf, all)
	if !strings.Contains(buf.String(), "(no body)") {
		t.Errorf("expected (no body) marker, got: %s", buf.String())
	}
}
