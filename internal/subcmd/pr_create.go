package subcmd

import (
	"context"
	"fmt"
	"os"

	"github.com/simons-agent-space/ghx/internal/api"
)

// PRCreate opens a new pull request on GitHub.
//
// Mirrors the ergonomics of `gh pr create`:
//   --repo OWNER/REPO (required)
//   --head BRANCH     (required; GitHub also accepts OWNER:BRANCH)
//   --title "..."     (required; inline string since titles are short)
//   --base BRANCH     (defaults to "main" when omitted)
//   --body-file PATH  (optional; reads the PR body from PATH)
//   --draft           (create as draft; defaults to false)
//   --json            (print the raw API response)
//
// Endpoint: POST /repos/OWNER/REPO/pulls with
//
//	{ "title": ..., "head": ..., "base": ..., "body": ..., "draft": ... }
//
// `body` is sent as "" when --body-file is not supplied (some
// workflows leave the body intentionally blank).
func PRCreate(ctx context.Context, c *api.Client, args []string) error {
	fs := flagSet("pr-create", "--repo OWNER/REPO --head BRANCH --title TITLE [--base BRANCH] [--body-file PATH] [--draft] [--json]")
	repo := fs.String("repo", "", "owner/repo (required)")
	head := fs.String("head", "", "head branch (required; OWNER:BRANCH also accepted)")
	title := fs.String("title", "", "PR title (required)")
	base := fs.String("base", "main", "base branch (defaults to main)")
	bodyFile := fs.String("body-file", "", "path to file with the PR body (optional)")
	draft := fs.Bool("draft", false, "create as draft")
	jsonOut := fs.Bool("json", false, "print raw JSON of the created PR")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *repo == "" {
		return fmt.Errorf("--repo is required")
	}
	if *head == "" {
		return fmt.Errorf("--head is required")
	}
	if *title == "" {
		return fmt.Errorf("--title is required")
	}
	// fs.NArg() must be 0: every argument is a flag.
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments: %v (all values go through flags)", fs.Args())
	}

	body := ""
	if *bodyFile != "" {
		b, err := os.ReadFile(*bodyFile)
		if err != nil {
			return fmt.Errorf("read body file %s: %w", *bodyFile, err)
		}
		body = string(b)
	}

	payload := map[string]any{
		"title": *title,
		"head":  *head,
		"base":  *base,
		"body":  body,
		"draft": *draft,
	}

	var out map[string]any
	if err := c.Post(ctx, fmt.Sprintf("/repos/%s/pulls", *repo), payload, &out); err != nil {
		return err
	}
	if *jsonOut {
		return writeJSON(os.Stdout, out)
	}
	fmt.Fprintf(os.Stdout, "PR %s opened: %s\n", asString(out["number"]), asString(out["html_url"]))
	return nil
}