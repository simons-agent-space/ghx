package subcmd

import (
	"context"
	"fmt"
	"os"

	"github.com/simons-agent-space/ghx/internal/api"
)

// PREdit updates a PR's body from a file.
func PREdit(ctx context.Context, c *api.Client, args []string) error {
	fs := flagSet("pr-edit", "--repo OWNER/REPO NUMBER --body-file PATH")
	repo := fs.String("repo", "", "owner/repo (required)")
	bodyFile := fs.String("body-file", "", "path to file containing new PR body (required)")
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

	body, err := os.ReadFile(*bodyFile)
	if err != nil {
		return fmt.Errorf("read body file %s: %w", *bodyFile, err)
	}

	var out map[string]any
	if err := c.Patch(ctx, fmt.Sprintf("/repos/%s/pulls/%s", *repo, fs.Arg(0)), map[string]any{"body": string(body)}, &out); err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "PR %s updated: %s\n", fs.Arg(0), asString(out["html_url"]))
	return nil
}
