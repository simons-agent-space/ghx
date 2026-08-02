package subcmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/simons-agent-space/ghx/internal/api"
)

// PRView fetches and prints a single pull request.
func PRView(ctx context.Context, c *api.Client, args []string) error {
	fs := flagSet("pr-view", "--repo OWNER/REPO NUMBER [--json]")
	repo := fs.String("repo", "", "owner/repo (required)")
	jsonOut := fs.Bool("json", false, "print raw JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *repo == "" {
		return fmt.Errorf("--repo is required (e.g. --repo simons-agent-space/agentctl)")
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("expected exactly one PR NUMBER")
	}
	n, err := strconv.Atoi(fs.Arg(0))
	if err != nil {
		return fmt.Errorf("invalid PR number %q: %w", fs.Arg(0), err)
	}

	var raw map[string]any
	if err := c.Get(ctx, fmt.Sprintf("/repos/%s/pulls/%d", *repo, n), &raw); err != nil {
		return err
	}
	if *jsonOut {
		return writeJSON(os.Stdout, raw)
	}
	printPRView(os.Stdout, raw, *repo)
	return nil
}

func printPRView(w io.Writer, raw map[string]any, repo string) {
	state := asString(raw["state"])
	if asBool(raw["draft"]) && state == "open" {
		state = "open (draft)"
	}
	author := ""
	if u, ok := raw["user"].(map[string]any); ok {
		author = asString(u["login"])
	}
	headRef, headSHA := "", ""
	if head, ok := raw["head"].(map[string]any); ok {
		headRef = asString(head["ref"])
		headSHA = asString(head["sha"])
	}
	baseRef := ""
	if base, ok := raw["base"].(map[string]any); ok {
		baseRef = asString(base["ref"])
	}
	fmt.Fprintf(w, "PR #%d: %s\n", asInt(raw["number"]), asString(raw["title"]))
	fmt.Fprintf(w, "Repo:      %s\n", repo)
	fmt.Fprintf(w, "State:     %s\n", state)
	fmt.Fprintf(w, "Author:    %s\n", author)
	fmt.Fprintf(w, "Branch:    %s -> %s\n", headRef, baseRef)
	fmt.Fprintf(w, "Head SHA:  %s\n", headSHA)
	fmt.Fprintf(w, "Mergeable: %s\n", boolWord(asBool(raw["mergeable"])))
	fmt.Fprintf(w, "Merged:    %s\n", boolWord(asBool(raw["merged"])))
	fmt.Fprintf(w, "URL:       %s\n", asString(raw["html_url"]))
	fmt.Fprintf(w, "Created:   %s\n", asString(raw["created_at"]))
	fmt.Fprintf(w, "Updated:   %s\n", asString(raw["updated_at"]))
	if body := asString(raw["body"]); body != "" {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Body:")
		fmt.Fprintln(w, "------")
		fmt.Fprintln(w, body)
	}
}
