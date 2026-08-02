package subcmd

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/simons-agent-space/ghx/internal/api"
)

// PRCommentAdd posts a new issue-level comment on a pull request.
//
// PRs are issues in GitHub's data model, so the endpoint is the
// issue-comments collection. This is the general PR-discussion
// comment path; inline review comments on diff lines use a
// different endpoint and are out of scope here.
//
// Body source is a file (--body-file) to keep parity with pr-edit
// and to avoid quoting hazards on the command line.
func PRCommentAdd(ctx context.Context, c *api.Client, args []string) error {
	fs := flagSet("pr-comment-add", "--repo OWNER/REPO NUMBER --body-file PATH [--json]")
	repo := fs.String("repo", "", "owner/repo (required)")
	bodyFile := fs.String("body-file", "", "path to file containing the comment body (required)")
	jsonOut := fs.Bool("json", false, "print raw JSON of the created comment")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *repo == "" {
		return fmt.Errorf("--repo is required")
	}
	if *bodyFile == "" {
		return fmt.Errorf("--body-file is required")
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("expected exactly one PR NUMBER")
	}
	n, err := strconv.Atoi(fs.Arg(0))
	if err != nil {
		return fmt.Errorf("invalid PR number %q: %w", fs.Arg(0), err)
	}

	body, err := os.ReadFile(*bodyFile)
	if err != nil {
		return fmt.Errorf("read body file %s: %w", *bodyFile, err)
	}

	var out map[string]any
	if err := c.Post(ctx, fmt.Sprintf("/repos/%s/issues/%d/comments", *repo, n), map[string]any{"body": string(body)}, &out); err != nil {
		return err
	}
	if *jsonOut {
		return writeJSON(os.Stdout, out)
	}
	fmt.Fprintf(os.Stdout, "Comment %d added on PR %d: %s\n", asInt(out["id"]), n, asString(out["html_url"]))
	return nil
}
