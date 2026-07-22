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

`make build-team CLIENT_ID=... CLIENT_SECRET=...` additionally injects `cmd.DefaultOAuthClientID`/`cmd.DefaultOAuthClientSecret` for zero-config `bbgo config login` (used only when no flags/env/stored credentials exist; requires the OAuth client's Client credentials grant to be disabled).

## Project Layout

```
main.go                          # cli.App, global flags, Before hook, RedactWriter wiring
docs/
  oauth-setup.md                 # workspace-admin guide for creating the OAuth client
  images/                        # screenshot slots referenced by oauth-setup.md
cmd/
  helpers.go                     # resolveRepo(), newClient(), exitWithError(), getOutputFormat()
  config.go                      # config set/show/clear-token/verify/login/logout
  browser.go                     # openBrowser() per-OS default browser launcher
  pr.go                          # pr list/show/diff/files/create
  comment.go                     # comment add/submit/pending/discard/list/post/delete
  review.go                      # review list/approve/request-changes/unapprove
  file.go                        # file get
internal/
  bitbucket/client.go            # HTTP client, Basic Auth, retry on 429, typed errors
  bitbucket/types.go             # API response structs (PullRequest, Comment, etc.)
  bitbucket/pr.go                # PR API operations + CreatePRRequest type
  bitbucket/oauth.go             # OAuth 2.0 flow: BrowserLogin (loopback), ExchangeCode, Refresh
  bitbucket/comment.go           # Comment API + tag embedding via HTML comments
  bitbucket/review.go            # Review API (approve, unapprove, request-changes)
  bitbucket/file.go              # File content retrieval from /src/{ref}/{path}
  config/store.go                # YAML config at ~/.bbgo/config.yaml (0600 perms)
  git/remote.go                  # Parse Bitbucket SSH/HTTPS remote URLs, detect branch
  pending/store.go               # Local pending comments store (~/.bbgo/pending_comments.json)
  secrets/keychain.go            # OS keychain (zalando/go-keyring) + AES-GCM file fallback
  secrets/oauth.go               # OAuthCredentials persistence (keychain "bitbucket-oauth" / ~/.bbgo/oauth)
  secrets/redact.go              # RedactSecrets() + RedactWriter for stdout/stderr
  output/formatter.go            # PrintJSON() + Table (tabwriter)
```

## Key Architecture Decisions

- **CLI framework:** `urfave/cli/v2` (not Cobra). Commands are `*cli.Command` structs returned by exported functions (`PRCommands()`, `ConfigCommands()`, etc.) and registered in `main.go`.
- **Auth:** Bearer token (`Authorization: Bearer <token>`). Two ways to get one: OAuth 2.0 browser login (`bbgo config login`, authorization-code flow with localhost callback on port 8976 — preferred, attributes actions to the real user) or a static API token. Resolution order in `cmd/helpers.go:newClient()`: `BBGO_TOKEN` env → OAuth session (auto-refresh via `refreshOAuth()`, refresh tokens rotate) → static token. The OAuth client must be registered by a workspace admin with callback `http://localhost:8976/callback`; Bitbucket does not support gcloud-style random loopback ports.
- **Token storage:** OS keychain first → encrypted file fallback (`~/.bbgo/token`, AES-256-GCM). Token is loaded into unexported `loadedToken` var at startup, accessed via `secrets.Token()`. OAuth sessions are stored the same way as JSON under account `bitbucket-oauth` (fallback `~/.bbgo/oauth`); load/store via `secrets.LoadOAuth()`/`secrets.StoreOAuth()` — `LoadOAuth` returns `(nil, nil)` when not logged in.
- **Security:** `RedactWriter` wraps `app.Writer` and `app.ErrWriter` in `main.go`. All output passes through `RedactSecrets()` which replaces the token and regex-matched secret patterns.
- **Repo resolution chain:** `--repo` flag → `config.default_repo` → `git remote origin` auto-detect. Implemented in `cmd/helpers.go:resolveRepo()`.
- **Output format resolution:** Subcommand `--output` flag → walks `c.Lineage()` for global `--output` flag → defaults to `"text"`. Implemented in `cmd/pr.go:getOutputFormat()`.

## Error Handling

Exit codes are mapped from error types in `cmd/helpers.go:ExitCodeForError()`:
- `*bitbucket.AuthError` → exit 2 (401)
- `*bitbucket.NotFoundError` → exit 3 (404)
- `*bitbucket.ForbiddenError` → exit 4 (403 — permission/scope problem)
- Git detection failures → exit 5 (string match on error message)
- Everything else → exit 1

404/403 errors include the request method+path (from the client) and are enriched by `cmd/helpers.go:DecorateError()` (called in main's ExitErrHandler) with the resolved repo, where it came from (flag/config/git — tracked by `rememberRepo()`), and actionable hints. `--verbose` also logs the repo resolution and the auth source.

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

Tests exist for: config load/save, secret redaction, git URL parsing, comment tagging, client error types, OAuth (authorize URL, code exchange, refresh rotation, full loopback login flow with a fake browser, encrypted credential storage). No tests for cmd handlers (would need integration-level setup). Run `go test ./...` — all should pass.

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
