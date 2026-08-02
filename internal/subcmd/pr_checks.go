package subcmd

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"strconv"

	"github.com/simons-agent-space/ghx/internal/api"
)

// PRChecks fetches check runs and workflow runs for a PR's head SHA.
func PRChecks(ctx context.Context, c *api.Client, args []string) error {
	fs := flagSet("pr-checks", "--repo OWNER/REPO NUMBER [--json]")
	repo := fs.String("repo", "", "owner/repo (required)")
	jsonOut := fs.Bool("json", false, "print raw JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *repo == "" {
		return fmt.Errorf("--repo is required")
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("expected exactly one PR NUMBER")
	}
	n, err := strconv.Atoi(fs.Arg(0))
	if err != nil {
		return fmt.Errorf("invalid PR number %q: %w", fs.Arg(0), err)
	}

	// First, fetch the PR to discover the head SHA + branch.
	var pr map[string]any
	if err := c.Get(ctx, fmt.Sprintf("/repos/%s/pulls/%d", *repo, n), &pr); err != nil {
		return err
	}
	headSHA, headBranch := "", ""
	if head, ok := pr["head"].(map[string]any); ok {
		headSHA = asString(head["sha"])
		headBranch = asString(head["ref"])
	}
	if headSHA == "" {
		return fmt.Errorf("PR #%d has no head SHA", n)
	}

	var checkRuns struct {
		CheckRuns []struct {
			Name       string `json:"name"`
			Status     string `json:"status"`
			Conclusion string `json:"conclusion"`
			HTMLURL    string `json:"html_url"`
			StartedAt  string `json:"started_at"`
		} `json:"check_runs"`
	}
	if err := c.Get(ctx, fmt.Sprintf("/repos/%s/commits/%s/check-runs", *repo, headSHA), &checkRuns); err != nil {
		return err
	}

	q := url.Values{}
	q.Set("branch", headBranch)
	q.Set("per_page", "5")
	var workflowRuns struct {
		WorkflowRuns []struct {
			Name       string `json:"name"`
			Status     string `json:"status"`
			Conclusion string `json:"conclusion"`
			HeadSHA    string `json:"head_sha"`
			Event      string `json:"event"`
			HTMLURL    string `json:"html_url"`
			RunNumber  int    `json:"run_number"`
		} `json:"workflow_runs"`
	}
	if err := c.Get(ctx, "/repos/"+*repo+"/actions/runs?"+q.Encode(), &workflowRuns); err != nil {
		return err
	}

	if *jsonOut {
		return writeJSON(os.Stdout, map[string]any{
			"head_sha":      headSHA,
			"head_branch":   headBranch,
			"check_runs":    checkRuns.CheckRuns,
			"workflow_runs": workflowRuns.WorkflowRuns,
		})
	}
	printChecks(os.Stdout, headSHA, headBranch, checkRuns.CheckRuns, workflowRuns.WorkflowRuns)
	return nil
}

func printChecks(w io.Writer, sha, branch string, checkRuns []struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
	HTMLURL    string `json:"html_url"`
	StartedAt  string `json:"started_at"`
}, workflowRuns []struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
	HeadSHA    string `json:"head_sha"`
	Event      string `json:"event"`
	HTMLURL    string `json:"html_url"`
	RunNumber  int    `json:"run_number"`
}) {
	fmt.Fprintf(w, "PR head SHA: %s\n", sha)
	fmt.Fprintf(w, "PR head ref: %s\n\n", branch)

	fmt.Fprintln(w, "Check runs:")
	if len(checkRuns) == 0 {
		fmt.Fprintln(w, "  (none)")
	}
	for _, r := range checkRuns {
		fmt.Fprintf(w, "  - %s  status=%s  conclusion=%s\n", r.Name, r.Status, r.Conclusion)
		if r.HTMLURL != "" {
			fmt.Fprintf(w, "    %s\n", r.HTMLURL)
		}
	}
	fmt.Fprintln(w)

	fmt.Fprintln(w, "Workflow runs:")
	if len(workflowRuns) == 0 {
		fmt.Fprintln(w, "  (none)")
	}
	for _, r := range workflowRuns {
		short := r.HeadSHA
		if len(short) > 10 {
			short = short[:10]
		}
		fmt.Fprintf(w, "  - %s #%d  %s/%s  head=%s  event=%s\n", r.Name, r.RunNumber, r.Status, r.Conclusion, short, r.Event)
		if r.HTMLURL != "" {
			fmt.Fprintf(w, "    %s\n", r.HTMLURL)
		}
	}
}
