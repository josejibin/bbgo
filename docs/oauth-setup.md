# OAuth Client Setup (Workspace Admin Guide)

This guide is for **Bitbucket workspace admins**. It covers the one-time setup that lets team
members run `bbgo config login` and authenticate as themselves — so pull requests, comments, and
approvals are attributed to the real user, not a bot or service account.

You only do this once per workspace. After that, every team member uses the same client
credentials to log in.

## Why OAuth instead of tokens?

| Method | Attribution | Availability |
|---|---|---|
| Workspace/repo access token | Bot user | Admin-issued |
| Personal API token | Real user | Often blocked by org policy |
| **OAuth login (this guide)** | **Real user** | Works even when API tokens are blocked |

bbgo's login flow is the same pattern as `gcloud auth login`: the CLI opens a browser, the user
signs in (corporate SSO included), and the CLI receives short-lived tokens that act on the user's
behalf. Tokens auto-refresh; users never copy/paste anything.

## Prerequisites

- You are an **admin of the Bitbucket workspace** (you can see *Workspace settings*).
- You know which port the team will use for the login callback. bbgo's default is **8976** —
  keep it unless you have a reason not to.

## Step 1 — Open the OAuth clients page

1. Go to [bitbucket.org](https://bitbucket.org) and open your workspace.
2. Click the workspace avatar → **Workspace settings**.
3. In the left sidebar under **Apps and features**, click **OAuth clients**.
4. Click **Create OAuth client**.

![OAuth clients page with the Create OAuth client button](images/oauth-clients-page.png)

## Step 2 — Fill in the dialog

The **Create OAuth client** dialog has three tabs: **Details**, **Authorization**, and **Scopes**.

### Details tab

| Field | Value |
|---|---|
| **Name** | `bbgo CLI` (anything recognizable) |
| **Description** | `Team CLI for PRs — users authenticate as themselves` |
| **Learn more URL / EULA URL / Privacy policy URL** | Optional and purely informational (shown as links on the consent screen; Bitbucket never fetches them). Leave blank or point at your bbgo repo / internal wiki. |

![Details tab of the Create OAuth client dialog](images/add-client-form.png)

### Authorization tab

| Field | Value |
|---|---|
| **Supported grant types** | ✅ **Authorization code** only (**Refresh token** is included automatically) |
| **Callback URL** | `http://localhost:8976/callback` |

> **Do not enable the Client credentials grant.** It lets anyone holding the client secret mint
> tokens that act as the client's *creator* (an admin) with no user consent — recreating the
> bot-attribution problem this setup exists to avoid, and needlessly raising the stakes if the
> secret leaks. bbgo only needs Authorization code (+ refresh).

> **The callback URL must match exactly.** Bitbucket validates it on every login, including the
> port. If you register a different port here, users must pass `--port N` to `bbgo config login`.
> Bitbucket does not support random loopback ports the way Google's OAuth does.

![Authorization tab: grant types and callback URL](images/authorization-grant-types.png)

### Scopes tab

Check:

| Scope | Level | Used by |
|---|---|---|
| Account | Read | `config verify`, identity check after login |
| Repositories | Write (Read is implied) | `file get`, repo resolution |
| Pull requests | Write (Read is implied) | `pr create`, `comment`, `review approve` / `request-changes` |

Leave everything else unchecked — bbgo does not need Wikis, Snippets, Issues, Webhooks, Pipelines,
or admin scopes. (Grayed-out checkmarks are implied by your selections and are fine.)

![Scopes tab with Account read, Repositories write, Pull requests write](images/client-permissions.png)

## Step 3 — Save and share the credentials

On save, an **"OAuth client created"** dialog shows the **Client ID** and the **Secret** (masked,
with copy buttons). You can retrieve both later from the client's row in the list.

- **Client ID** — not secret; fine to put in a team wiki
- **Secret** — treat like a password: share via your password manager or secrets tool,
  not chat or email

![OAuth client created dialog showing Client ID and masked Secret](images/client-key-secret.png)

## Step 4 — Tell the team

**Option A — zero-config team binary (recommended):** build bbgo with the credentials embedded
(`make build-team CLIENT_ID=<key> CLIENT_SECRET=<secret>`, ideally in CI with the secret as a
secured variable) and distribute the binary. Team members just run `bbgo config login` — same
model as `gcloud auth login`. See "Team builds" in the README for the security reasoning (short
version: the embedded pair is extractable from the binary but can only *start* a consent-requiring
browser login, because Client credentials is disabled — which is why Step 2 insists on that).

**Option B — share the credentials:** each member runs, once:

```bash
bbgo config login --client-id <KEY> --client-secret <SECRET>
```

On shared/multi-user machines, prefer environment variables so the secret never appears in `ps`
output:

```bash
export BBGO_OAUTH_CLIENT_ID=<KEY> BBGO_OAUTH_CLIENT_SECRET=<SECRET>
bbgo config login
```

A browser opens; they sign in to Bitbucket (SSO works) and click **Accept** on the consent
screen:

![Consent screen shown to each user on first login](images/consent-screen.png)

> Every scope granted to the client is listed here — another reason to keep scopes minimal.

bbgo then confirms with:

```
Logged in as Jane Doe (janedoe). PRs and comments will be attributed to this user.
```

The credentials are stored in the OS keychain (encrypted-file fallback), and later re-logins are
just `bbgo config login` — the client ID/secret are remembered. Access tokens expire after ~2 hours
and refresh automatically.

## Troubleshooting

**"cannot listen on port 8976"** — something on the user's machine is using the port. Free it, or
register a second client with a different callback port and use `--port`.

**"redirect_uri mismatch" in the browser** — the callback URL on the client doesn't exactly match
`http://localhost:<port>/callback`. Check scheme (`http`, not `https`), host (`localhost`), port,
and path.

**"OAuth session expired and refresh failed"** — the refresh token was revoked (e.g., the client
was deleted/recreated, or the user revoked access under *Personal settings → Authorized
applications*). Re-run `bbgo config login`.

**User sees the consent screen for the wrong workspace** — OAuth clients belong to one workspace.
If the team spans multiple workspaces, create one client per workspace.

**Headless machines (SSH, containers, CI)** — the browser flow needs a local browser. For CI, use
the `BBGO_TOKEN` environment variable with whatever credential your pipeline is allowed to use;
attribution to a human user is generally not wanted in CI anyway.

## Security notes

- The client secret lets someone *start* a login flow — it cannot read repositories by itself.
  A user still has to sign in and grant access. Rotate it from the same settings page if it leaks.
- Atlassian's [OAuth 2.0 enforcement changes (May 2026)](https://community.developer.atlassian.com/t/oauth-2-0-and-api-authentication-changes-for-bitbucket-cloud/99003)
  require exact redirect URI matching and rotate refresh tokens on every refresh. bbgo handles both.
- Tokens and the client secret never appear in bbgo output — they are registered with the
  output redaction layer.
