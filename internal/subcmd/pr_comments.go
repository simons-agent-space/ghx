package subcmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/simons-agent-space/ghx/internal/api"
)

// PRComment is the unified shape used to print all PR comment types.
type PRComment struct {
	Kind    string // "issue", "review", "inline"
	Author  string
	Created string
	Body    string
	State   string // for "review"
	Path    string // for "inline"
	Line    any    // for "inline"
}

// PRComments fetches and prints all comments on a PR.
//
// Combines three sources:
//   - issue comments (general PR discussion)
//   - review-level comments (the approve / changes summary)
//   - inline review comments (specific code lines)
func PRComments(ctx context.Context, c *api.Client, args []string) error {
	fs := flagSet("pr-comments", "--repo OWNER/REPO NUMBER [--json]")
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

	var all []PRComment

	// Issue-level comments.
	var issueComments []struct {
		User      struct{ Login string `json:"login"` } `json:"user"`
		CreatedAt string                                 `json:"created_at"`
		Body      string                                 `json:"body"`
	}
	if err := c.Get(ctx, fmt.Sprintf("/repos/%s/issues/%d/comments", *repo, n), &issueComments); err != nil {
		return err
	}
	for _, cm := range issueComments {
		all = append(all, PRComment{
			Kind:    "issue",
			Author:  cm.User.Login,
			Created: cm.CreatedAt,
			Body:    cm.Body,
		})
	}

	// Review-level comments.
	var reviews []struct {
		User        struct{ Login string `json:"login"` } `json:"user"`
		SubmittedAt string                                 `json:"submitted_at"`
		State       string                                 `json:"state"`
		Body        string                                 `json:"body"`
	}
	if err := c.Get(ctx, fmt.Sprintf("/repos/%s/pulls/%d/reviews", *repo, n), &reviews); err != nil {
		return err
	}
	for _, r := range reviews {
		all = append(all, PRComment{
			Kind:    "review",
			Author:  r.User.Login,
			Created: r.SubmittedAt,
			Body:    r.Body,
			State:   r.State,
		})
	}

	// Inline review comments.
	var inline []struct {
		User      struct{ Login string `json:"login"` } `json:"user"`
		CreatedAt string                                 `json:"created_at"`
		Path      string                                 `json:"path"`
		Line      any                                    `json:"line"`
		Body      string                                 `json:"body"`
	}
	if err := c.Get(ctx, fmt.Sprintf("/repos/%s/pulls/%d/comments", *repo, n), &inline); err != nil {
		return err
	}
	for _, i := range inline {
		all = append(all, PRComment{
			Kind:    "inline",
			Author:  i.User.Login,
			Created: i.CreatedAt,
			Body:    i.Body,
			Path:    i.Path,
			Line:    i.Line,
		})
	}

	sort.Slice(all, func(i, j int) bool { return all[i].Created < all[j].Created })

	if *jsonOut {
		return writeJSON(os.Stdout, all)
	}
	if len(all) == 0 {
		fmt.Fprintln(os.Stdout, "(no comments)")
		return nil
	}
	printComments(os.Stdout, all)
	return nil
}

func printComments(w io.Writer, all []PRComment) {
	for _, cm := range all {
		header := fmt.Sprintf("[%s] %s @ %s", cm.Kind, cm.Author, cm.Created)
		if cm.State != "" {
			header += "  state=" + cm.State
		}
		if cm.Path != "" {
			header += "  " + cm.Path
			if cm.Line != nil {
				header += fmt.Sprintf(":L%v", cm.Line)
			}
		}
		fmt.Fprintln(w, header)
		fmt.Fprintln(w, strings.Repeat("-", len(header)))
		if cm.Body != "" {
			fmt.Fprintln(w, cm.Body)
		} else {
			fmt.Fprintln(w, "(no body)")
		}
		fmt.Fprintln(w)
	}
}
