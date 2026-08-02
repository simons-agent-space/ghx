// Command ghx is a small GitHub REST CLI for the agent's workspace.
//
// Usage: see README.md.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/simons-agent-space/ghx/internal/api"
	"github.com/simons-agent-space/ghx/internal/subcmd"
)

const version = "ghx 0.1.0"

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "ghx:", err)
		os.Exit(exitCode(err))
	}
}

func run() error {
	if len(os.Args) < 2 {
		usage(os.Stderr)
		return fmt.Errorf("no subcommand given")
	}
	cmd := os.Args[1]
	args := os.Args[2:]

	// version does not need auth.
	if cmd == "version" {
		fmt.Println(version)
		return nil
	}

	ctx := context.Background()
	c, err := api.NewClient()
	if err != nil {
		return err
	}

	switch cmd {
	case "pr-view":
		return subcmd.PRView(ctx, c, args)
	case "pr-list":
		return subcmd.PRList(ctx, c, args)
	case "pr-comments":
		return subcmd.PRComments(ctx, c, args)
	case "pr-checks":
		return subcmd.PRChecks(ctx, c, args)
	case "pr-edit":
		return subcmd.PREdit(ctx, c, args)
	case "api":
		return subcmd.API(ctx, c, args)
	default:
		usage(os.Stderr)
		return fmt.Errorf("unknown subcommand: %s", cmd)
	}
}

func usage(w *os.File) {
	fmt.Fprintln(w, `ghx - small GitHub REST CLI

Usage:
  ghx <subcommand> [args]

Subcommands:
  pr-view N --repo OWNER/REPO [--json]
      PR metadata + body + mergeable state.
  pr-list --repo OWNER/REPO [--state open|closed|all] [--head REF] [--json]
      List PRs in the repo.
  pr-comments N --repo OWNER/REPO [--json]
      Combined issue, review-level, and inline review comments for PR N.
  pr-checks N --repo OWNER/REPO [--json]
      Check runs + workflow runs for the PR head SHA.
  pr-edit N --repo OWNER/REPO --body-file PATH
      Update PR N's body from PATH.
  api METHOD PATH [-d BODY]
      Raw passthrough to the GitHub REST API.
  version
      Print the version.

Auth:
  ghx reads the token from /tmp/gitbridge-credentials, populated by
  /workspace/bin/gitbridge-auth OWNER/REPO.`)
}

// exitCode maps errors to exit codes per the README contract.
func exitCode(err error) int {
	switch {
	case err == nil:
		return 0
	case errors.Is(err, api.ErrCredentialsMissing), errors.Is(err, api.ErrCredentialsMalformed):
		return 2
	case errors.Is(err, api.ErrAuthFailed):
		return 3
	case errors.Is(err, api.ErrNotFound):
		return 4
	case errors.Is(err, api.ErrAPIError):
		return 5
	}
	return 1
}
