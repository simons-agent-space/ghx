package subcmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestPrintChecks(t *testing.T) {
	const sha = "1256d5c6a5bfefdba175244f772d65724557ffae"
	const branch = "feature-x"
	checkRuns := []struct {
		Name       string `json:"name"`
		Status     string `json:"status"`
		Conclusion string `json:"conclusion"`
		HTMLURL    string `json:"html_url"`
		StartedAt  string `json:"started_at"`
	}{
		{Name: "ci", Status: "completed", Conclusion: "success", HTMLURL: "https://example/run/1"},
	}
	workflowRuns := []struct {
		Name       string `json:"name"`
		Status     string `json:"status"`
		Conclusion string `json:"conclusion"`
		HeadSHA    string `json:"head_sha"`
		Event      string `json:"event"`
		HTMLURL    string `json:"html_url"`
		RunNumber  int    `json:"run_number"`
	}{
		{Name: "build", Status: "completed", Conclusion: "success", HeadSHA: sha, Event: "push", HTMLURL: "https://example/run/2", RunNumber: 42},
	}
	var buf bytes.Buffer
	printChecks(&buf, sha, branch, checkRuns, workflowRuns)
	out := buf.String()
	for _, want := range []string{
		sha,
		branch,
		"ci",
		"completed/success",
		"https://example/run/1",
		"build #42",
		"1256d5c6a5",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\ngot:\n%s", want, out)
		}
	}
}

func TestPrintChecksEmpty(t *testing.T) {
	var checkRuns []struct {
		Name       string `json:"name"`
		Status     string `json:"status"`
		Conclusion string `json:"conclusion"`
		HTMLURL    string `json:"html_url"`
		StartedAt  string `json:"started_at"`
	}
	var workflowRuns []struct {
		Name       string `json:"name"`
		Status     string `json:"status"`
		Conclusion string `json:"conclusion"`
		HeadSHA    string `json:"head_sha"`
		Event      string `json:"event"`
		HTMLURL    string `json:"html_url"`
		RunNumber  int    `json:"run_number"`
	}
	var buf bytes.Buffer
	printChecks(&buf, "sha", "branch", checkRuns, workflowRuns)
	out := buf.String()
	for _, want := range []string{
		"(none)", // appears twice — once for check runs, once for workflow runs
		"PR head SHA: sha",
		"PR head ref: branch",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\ngot:\n%s", want, out)
		}
	}
}
