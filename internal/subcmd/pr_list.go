package subcmd

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"

	"github.com/simons-agent-space/ghx/internal/api"
)

// PRList lists pull requests in a repo.
func PRList(ctx context.Context, c *api.Client, args []string) error {
	fs := flagSet("pr-list", "--repo OWNER/REPO [--state open|closed|all] [--head REF] [--author USER] [--json]")
	repo := fs.String("repo", "", "owner/repo (required)")
	state := fs.String("state", "open", "filter by state (open, closed, all)")
	head := fs.String("head", "", "filter by head branch (owner:branch)")
	author := fs.String("author", "", "filter by author username")
	jsonOut := fs.Bool("json", false, "print raw JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *repo == "" {
		return fmt.Errorf("--repo is required")
	}
	q := url.Values{}
	q.Set("state", *state)
	q.Set("per_page", "30")
	if *head != "" {
		q.Set("head", *head)
	}
	if *author != "" {
		q.Set("author", *author)
	}
	var arr []map[string]any
	if err := c.Get(ctx, "/repos/"+*repo+"/pulls?"+q.Encode(), &arr); err != nil {
		return err
	}
	if *jsonOut {
		return writeJSON(os.Stdout, arr)
	}
	printPRList(os.Stdout, arr)
	return nil
}

func printPRList(w io.Writer, prs []map[string]any) {
	if len(prs) == 0 {
		fmt.Fprintln(w, "(no pull requests)")
		return
	}
	for _, pr := range prs {
		n := asInt(pr["number"])
		state := asString(pr["state"])
		if asBool(pr["draft"]) && state == "open" {
			state = "open (draft)"
		}
		headRef, baseRef := "", ""
		if head, ok := pr["head"].(map[string]any); ok {
			headRef = asString(head["ref"])
		}
		if base, ok := pr["base"].(map[string]any); ok {
			baseRef = asString(base["ref"])
		}
		fmt.Fprintf(w, "#%-5d %-12s %s -> %s\n", n, state, headRef, baseRef)
		fmt.Fprintf(w, "  %s\n", asString(pr["title"]))
		fmt.Fprintf(w, "  %s\n\n", asString(pr["html_url"]))
	}
}
