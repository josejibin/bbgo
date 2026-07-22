# bbgo

A CLI for Bitbucket Cloud. Manages pull requests, comments, reviews, and file content.

## Install

```bash
go install github.com/josejibin/bbgo@latest
```

Or build from source:

```bash
git clone https://github.com/josejibin/bbgo.git
cd bbgo
make build
```

## Quick Start

```bash
# Configure credentials — OAuth browser login (recommended; PRs attributed to you)
bbgo config set --workspace your-workspace
bbgo config login --client-id <KEY> --client-secret <SECRET>

# Or use an API token (if your org allows them)
bbgo config set --token YOUR_API_TOKEN
# Or use BBGO_TOKEN for CI/headless usage

# Verify setup
bbgo config verify

# List open PRs
bbgo pr list

# Show PR details
bbgo pr show 42

# Create a PR (auto-detects branch and repo from git)
bbgo pr create --title "Fix login bug"
```

## Commands

### Config

```bash
bbgo config login [--client-id K --client-secret S] [--port N]   # OAuth browser login
bbgo config logout                          # Remove OAuth session
bbgo config set --token <API_TOKEN>         # Store token in OS keychain
bbgo config set --workspace <WORKSPACE>
bbgo config set --repo <WORKSPACE/REPO>     # Set default repo
bbgo config show                            # Show current config
bbgo config verify                          # Test API credentials
bbgo config clear-token                     # Remove token from keychain
```

#### OAuth login

`bbgo config login` runs a browser-based OAuth 2.0 flow (same pattern as `gcloud auth login`): it starts a temporary listener on `localhost:8976`, opens the Bitbucket authorization page, and exchanges the returned code for tokens. Everything you do is attributed to **your own Bitbucket user** — unlike workspace access tokens, which show up as a bot.

One-time setup (workspace admin — see the full [OAuth setup guide](docs/oauth-setup.md)): **Workspace settings → OAuth clients → Create OAuth client** with:

- Grant types: Authorization code only (do not enable Client credentials)
- Callback URL: `http://localhost:8976/callback`
- Scopes: Account (read), Repositories (write), Pull requests (write)

Each team member then runs `bbgo config login --client-id <key> --client-secret <secret>` once; the client credentials are remembered, so later re-logins are just `bbgo config login`. Access tokens expire after ~2 hours and are refreshed automatically (Bitbucket rotates refresh tokens; bbgo persists the new one on every refresh). If port 8976 is taken, pass `--port N` — but the client's callback URL must be registered with that same port.

#### Team builds: zero-config login (optional)

`gcloud auth login` needs no client ID because Google ships its own pre-registered OAuth client inside the gcloud binary. bbgo can do the same for your team:

```bash
make build-team CLIENT_ID=<key> CLIENT_SECRET=<secret>
```

Distribute that binary and everyone logs in with a plain `bbgo config login` — no flags, no credential sharing. The credentials never appear in the source tree or git history: they are injected at build time via `-ldflags` and exist only inside the produced binary. Build it in CI with the secret held as a secured pipeline variable; if building locally, export `CLIENT_SECRET` from your password manager rather than typing it inline (command lines land in shell history).

**Security note:** anyone holding the binary can extract the embedded secret (`strings bbgo`), so treat it as *distributable*, not secret. This is the same model gcloud and gh use, and RFC 8252 (OAuth for native apps) explicitly accepts it — the security comes from user consent, not the secret. It is safe **only** because the OAuth client has the **Client credentials grant disabled**: the ID+secret pair cannot mint tokens by itself; all it can do is start a browser login that still requires a human to sign in and click *Grant access*. Never embed credentials of a client with Client credentials enabled. If the secret ever needs revoking, rotate it in Bitbucket and rebuild.

Credential precedence is unchanged — `--client-id`/`--client-secret` flags or `BBGO_OAUTH_CLIENT_*` env vars → credentials stored from a previous login → embedded defaults — so a plain `make build` binary (nothing embedded) works exactly as before.

### Pull Requests

```bash
bbgo pr list [--state open|merged|declined|all] [--author USER] [--source BRANCH] [--dest BRANCH]
bbgo pr show <ID>
bbgo pr diff <ID> [--stat]
bbgo pr files <ID>
bbgo pr create --title "..." [--description "..."] [--source BRANCH] [--dest BRANCH] \
               [--reviewers user1,user2] [--close-source-branch] [--draft]
```

### Comments

```bash
# Batch workflow (default — avoids per-comment notifications)
bbgo comment add <PR_ID> --body "..." [--file PATH --line N] [--tag TAG]   # queue locally
bbgo comment add <PR_ID> --body "..." --now                                # bypass queue, post immediately
bbgo comment submit <PR_ID>                                                # post all pending comments
bbgo comment pending <PR_ID>                                               # list pending comments
bbgo comment discard <PR_ID>                                               # discard pending comments

# Direct commands
bbgo comment list <PR_ID> [--inline-only]
bbgo comment post <PR_ID> --body "..." [--file PATH --line N] [--tag TAG]  # post immediately
bbgo comment post <PR_ID> --body - [--file PATH --line N] [--tag TAG]      # read body from stdin
bbgo comment delete <PR_ID> <COMMENT_ID>
bbgo comment delete <PR_ID> --tag TAG [--dry-run]
```

### Reviews

```bash
bbgo review list <PR_ID>
bbgo review approve <PR_ID>
bbgo review request-changes <PR_ID>
bbgo review unapprove <PR_ID>
```

### File Content

```bash
bbgo file get <PATH> [--commit HASH] [--branch BRANCH] [--repo W/R]
```

## Global Flags

```
--repo, -r     workspace/repo override (or BBGO_REPO)
--verbose      print HTTP request details
--no-color     disable ANSI color output
--output, -o   json|text (default: text)
--config       override config file path
```

## Output

Default output is human-readable text. Use `--output json` for machine-readable JSON.

## Security & Token Handling

### Credential resolution order

When bbgo needs credentials, it checks these sources in order — the first one that succeeds wins:

1. **`BBGO_TOKEN` environment variable** — best for CI/headless environments where no keyring is available.
2. **OAuth session** (from `bbgo config login`) — auto-refreshed when the access token expires.
3. **API token** — OS keyring first, then the encrypted fallback file (`~/.bbgo/token`).

The OAuth session (access token, refresh token, client credentials) is stored the same way as the API token: OS keyring under service `bbgo`, account `bitbucket-oauth`, with an AES-256-GCM encrypted fallback at `~/.bbgo/oauth`. OAuth access and refresh tokens and the client secret are all covered by output redaction.

### Storing a token

```bash
bbgo config set --token YOUR_API_TOKEN
```

This attempts to save the token to the OS keyring first. If the keyring is unavailable (e.g., no desktop session, running in a container), the token is saved to the encrypted fallback file instead.

### What is the OS keyring?

The OS keyring (also called keychain) is a credential store provided by your operating system:

| OS    | Backend                  | How it works                                                                 |
|-------|--------------------------|------------------------------------------------------------------------------|
| macOS | Keychain (`security`)    | Credentials stored in the login keychain, unlocked when you log in           |
| Linux | Secret Service (GNOME Keyring / KWallet) | Uses the D-Bus Secret Service API; requires a running desktop session |
| Windows | Windows Credential Manager | Credentials tied to your Windows user profile                         |

bbgo uses the [go-keyring](https://github.com/zalando/go-keyring) library to access these backends. The token is stored under service `bbgo`, account `bitbucket-token`.

**When the keyring is unavailable** (SSH sessions, containers, headless CI), bbgo falls back to an encrypted file.

### Encrypted file fallback

When the OS keyring isn't available, the token is stored at `~/.bbgo/token`:

- Encrypted with **AES-256-GCM**
- Encryption key is derived (SHA-256) from `hostname + username` — this ties the token to the machine and user
- File permissions are set to `0600` (owner read/write only)
- The `~/.bbgo/` directory is created with `0700` permissions

If the keyring becomes available later (e.g., you switch from SSH to a desktop session), running `bbgo config set --token` again will migrate the token to the keyring and delete the fallback file.

### Clearing the token

```bash
bbgo config clear-token
```

This removes the token from both the OS keyring and the fallback file.

### Output redaction

All output (stdout and stderr) passes through a `RedactWriter` that:

- Replaces the loaded token with `[REDACTED]`
- Matches common secret patterns (`token=...`, `password=...`, `secret=...`, `api_key=...`) and redacts their values

The token is never written to the config file (`~/.bbgo/config.yaml`) or logs.
