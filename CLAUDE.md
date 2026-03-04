# CLAUDE.md — bbgo project guide

## What is bbgo?

A Go CLI for Bitbucket Cloud. It manages pull requests, comments, reviews, and file content via the Bitbucket 2.0 REST API. Designed to be called by Claude Code as a subprocess — all output is machine-parseable with `--output json`.

## Build & Test

```bash
make build          # builds ./bbgo with version from git tags
make test           # go test ./...
make lint           # golangci-lint run ./...
go build -o bbgo .  # quick build without version injection
```

Version is injected via ldflags: `-X main.version=$(VERSION)`

## Project Layout

```
main.go                          # cli.App, global flags, Before hook, RedactWriter wiring
cmd/
  helpers.go                     # resolveRepo(), newClient(), exitWithError(), getOutputFormat()
  config.go                      # config set/show/clear-token/verify
  pr.go                          # pr list/show/diff/files/create
  comment.go                     # comment list/post/delete (with tag support)
  review.go                      # review list/approve/request-changes/unapprove
  file.go                        # file get
internal/
  bitbucket/client.go            # HTTP client, Basic Auth, retry on 429, typed errors
  bitbucket/types.go             # API response structs (PullRequest, Comment, etc.)
  bitbucket/pr.go                # PR API operations + CreatePRRequest type
  bitbucket/comment.go           # Comment API + tag embedding via HTML comments
  bitbucket/review.go            # Review API (approve, unapprove, request-changes)
  bitbucket/file.go              # File content retrieval from /src/{ref}/{path}
  config/store.go                # YAML config at ~/.bbgo/config.yaml (0600 perms)
  git/remote.go                  # Parse Bitbucket SSH/HTTPS remote URLs, detect branch
  secrets/keychain.go            # OS keychain (zalando/go-keyring) + AES-GCM file fallback
  secrets/redact.go              # RedactSecrets() + RedactWriter for stdout/stderr
  output/formatter.go            # PrintJSON() + Table (tabwriter)
```

## Key Architecture Decisions

- **CLI framework:** `urfave/cli/v2` (not Cobra). Commands are `*cli.Command` structs returned by exported functions (`PRCommands()`, `ConfigCommands()`, etc.) and registered in `main.go`.
- **Auth:** HTTP Basic Auth (`username:app_password`), not Bearer token.
- **Token storage:** OS keychain first → encrypted file fallback (`~/.bbgo/token`, AES-256-GCM). Token is loaded into unexported `loadedToken` var at startup, accessed via `secrets.Token()`.
- **Security:** `RedactWriter` wraps `app.Writer` and `app.ErrWriter` in `main.go`. All output passes through `RedactSecrets()` which replaces the token and regex-matched secret patterns.
- **Repo resolution chain:** `--repo` flag → `config.default_repo` → `git remote origin` auto-detect. Implemented in `cmd/helpers.go:resolveRepo()`.
- **Output format resolution:** Subcommand `--output` flag → walks `c.Lineage()` for global `--output` flag → defaults to `"text"`. Implemented in `cmd/pr.go:getOutputFormat()`.

## Error Handling

Exit codes are mapped from error types in `cmd/helpers.go:exitCodeForError()`:
- `*bitbucket.AuthError` → exit 2 ("Auth failed — run `bbgo config verify`")
- `*bitbucket.NotFoundError` → exit 3 ("Not found — check repo slug and PR ID")
- Git detection failures → exit 5 (string match on error message)
- Everything else → exit 1

HTTP client retries 429 responses up to 3 times with exponential backoff (1s, 2s, 4s).

## Dependencies

Only 3 direct dependencies:
- `github.com/urfave/cli/v2` — CLI framework
- `github.com/zalando/go-keyring` — OS keychain
- `gopkg.in/yaml.v3` — config file

## Comment Tag System

Tags are embedded as HTML comments in comment body: `<!-- bbgo:tag:TAG_NAME -->`. This enables bulk operations like `bbgo comment delete <PR_ID> --tag ai-review`. The `HasTag()` function does a simple `strings.Contains` check.

## Conventions

- All command constructors are in `cmd/` and return `*cli.Command`.
- All Bitbucket API methods are methods on `*bitbucket.Client`.
- API struct types live in `internal/bitbucket/types.go`.
- Config file is `~/.bbgo/config.yaml`, token file is `~/.bbgo/token`.
- Use `requireIntArg(c, "PR_ID")` to parse required int arguments.
- Use `exitWithError(err)` (not `return err`) for API/resolution errors — it sets the correct exit code.
- Tests use `t.TempDir()` for file-based tests and `httptest.NewServer` for HTTP tests.

## Common Pitfalls

- **`-v` alias conflict:** urfave/cli reserves `-v` for `--version`. Don't alias `--verbose` to `-v`.
- **`--no-color` flag:** Defined globally but not yet wired to any formatting logic. Text output currently has no ANSI codes, so this is a no-op for now.
- **`RedactWriter.Write()` returns original byte count** (not redacted length) to prevent callers from seeing a mismatch.
- **`config.Load()` returns empty config (not error)** when the file doesn't exist. This is intentional — first-run UX.
- **`resolveRepo()` has a hybrid path:** when workspace is in config but default_repo isn't, it combines config workspace with git-detected repo slug.

## Testing

Tests exist for: config load/save, secret redaction, git URL parsing, comment tagging, client error types. No tests for cmd handlers (would need integration-level setup). Run `go test ./...` — all should pass.

## Bitbucket API Endpoints Used

| Operation         | Method | Path                                                  |
|-------------------|--------|-------------------------------------------------------|
| List PRs          | GET    | `/2.0/repositories/{w}/{r}/pullrequests`              |
| Get PR            | GET    | `/2.0/repositories/{w}/{r}/pullrequests/{id}`         |
| Create PR         | POST   | `/2.0/repositories/{w}/{r}/pullrequests`              |
| Get diff          | GET    | `/2.0/repositories/{w}/{r}/pullrequests/{id}/diff`    |
| Get diffstat      | GET    | `/2.0/repositories/{w}/{r}/pullrequests/{id}/diffstat`|
| List comments     | GET    | `/2.0/repositories/{w}/{r}/pullrequests/{id}/comments`|
| Post comment      | POST   | `/2.0/repositories/{w}/{r}/pullrequests/{id}/comments`|
| Delete comment    | DELETE | `/2.0/repositories/{w}/{r}/pullrequests/{id}/comments/{cid}` |
| Approve PR        | POST   | `/2.0/repositories/{w}/{r}/pullrequests/{id}/approve` |
| Unapprove PR      | DELETE | `/2.0/repositories/{w}/{r}/pullrequests/{id}/approve` |
| Request changes   | POST   | `/2.0/repositories/{w}/{r}/pullrequests/{id}/request-changes` |
| Get file content  | GET    | `/2.0/repositories/{w}/{r}/src/{ref}/{path}`          |
| Get repo info     | GET    | `/2.0/repositories/{w}/{r}`                           |
| List workspaces   | GET    | `/2.0/workspaces`                                     |

## Git Author

Use `jj <jj@josejibin.com>` for commits in this repo.
