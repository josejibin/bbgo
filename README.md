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
bbgo comment list <PR_ID> [--inline-only]
bbgo comment post <PR_ID> --body "..." [--file PATH --line N] [--tag TAG]
bbgo comment post <PR_ID> --body - [--file PATH --line N] [--tag TAG]   # read body from stdin
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

## Security

- Token is stored in the OS keychain (falls back to an encrypted file at `~/.bbgo/token`)
- `BBGO_TOKEN` is supported for CI/headless environments as an alternative to secure local storage
- Token is never written to config files or logs
- All output is passed through a redaction filter
