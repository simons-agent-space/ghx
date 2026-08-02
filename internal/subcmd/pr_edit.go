package subcmd

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/simons-agent-space/ghx/internal/api"
)

// PREdit updates a PR's body from a file.
func PREdit(ctx context.Context, c *api.Client, args []string) error {
	fs := flagSet("pr-edit", "--repo OWNER/REPO NUMBER --body-file PATH [--json]")
	repo := fs.String("repo", "", "owner/repo (required)")
	bodyFile := fs.String("body-file", "", "path to file containing new PR body (required)")
	jsonOut := fs.Bool("json", false, "print raw JSON of the updated PR")
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
	if err := c.Patch(ctx, fmt.Sprintf("/repos/%s/pulls/%d", *repo, n), map[string]any{"body": string(body)}, &out); err != nil {
		return err
	}
	if *jsonOut {
		return writeJSON(os.Stdout, out)
	}
	fmt.Fprintf(os.Stdout, "PR %d updated: %s\n", n, asString(out["html_url"]))
	return nil
}
