# ghx

A small, stable GitHub REST CLI for the agent's workspace.

`ghx` exists because `gh` is not installed in the agent sandbox,
and rolling a one-off `curl | jq` invocation per API call is
fragile (unquoted tokens, missing `PATH`, no test coverage).

## Install

```sh
go build -o /workspace/bin/ghx ./cmd/ghx
```

## Auth

`ghx` reads a short-lived installation token from
`/tmp/gitbridge-credentials`, which is populated by
`/workspace/bin/gitbridge-auth OWNER/REPO`:

```sh
/workspace/bin/gitbridge-auth simons-agent-space/agentctl
ghx pr-view 10 --repo simons-agent-space/agentctl
```

If the credentials file is missing, `ghx` exits with a clear hint
to run `gitbridge-auth` first.

## Usage

```
ghx <subcommand> [args]

Subcommands:
  pr-view N --repo OWNER/REPO [--json]
      PR metadata + body + mergeable state.

  pr-list --repo OWNER/REPO [--state open|closed|all] [--head REF] [--json]
      List PRs in the repo.

  pr-comments N --repo OWNER/REPO [--json]
      Combined issue, review-level, and inline review comments
      for PR N, sorted chronologically.

  pr-checks N --repo OWNER/REPO [--json]
      Check runs + workflow runs for the PR head SHA.

  pr-edit N --repo OWNER/REPO --body-file PATH [--json]
      Update PR N's body from PATH.

  pr-comment-add N --repo OWNER/REPO --body-file PATH [--json]
      Post a new issue-level comment on PR N.

  api METHOD PATH [-d BODY]
      Raw passthrough to the GitHub REST API.

  version
      Print the version.

All subcommands accept `--json` to print raw JSON instead of the
human-readable summary.
```

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | success |
| 1 | usage / argument error |
| 2 | credentials missing or malformed |
| 3 | GitHub API authentication failure (401) |
| 4 | GitHub API resource not found (404) |
| 5 | other GitHub API error |
| 6 | network / I/O error |

## Design constraints

- Stdlib only — no external Go dependencies.
- Token never leaves the process; never echoed, never logged.
- Errors are typed so callers can `errors.Is` against
  `api.ErrCredentialsMissing`, `api.ErrAuthFailed`,
  `api.ErrNotFound`, `api.ErrAPIError`.
- Each subcommand is a single file under `internal/subcmd/`.
- Output is human-readable by default; `--json` for machine
  consumption.

## Tests

```sh
go vet ./...
go test -race ./...
```

Tests cover:
- Token extraction from the credentials URL.
- HTTP request construction (method, headers, body).
- Typed error mapping for non-2xx responses.

A fake HTTP server (`httptest`) is used so tests are hermetic.

## Development

Source of truth: <https://github.com/simons-agent-space/ghx>.
Test PRs use `test/*` branch naming.
