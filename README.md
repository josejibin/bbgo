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
# Configure credentials
bbgo config set --workspace your-workspace
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
bbgo config set --token <API_TOKEN>         # Store token in OS keychain
bbgo config set --workspace <WORKSPACE>
bbgo config set --repo <WORKSPACE/REPO>     # Set default repo
bbgo config show                            # Show current config
bbgo config verify                          # Test API credentials
bbgo config clear-token                     # Remove token from keychain
```

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

### Token resolution order

When bbgo needs your API token, it checks these sources in order — the first one that succeeds wins:

1. **`BBGO_TOKEN` environment variable** — best for CI/headless environments where no keyring is available.
2. **OS keyring** — the default and most secure option for interactive use.
3. **Encrypted fallback file** (`~/.bbgo/token`) — used automatically when the OS keyring is unavailable.

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
