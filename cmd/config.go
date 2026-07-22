package cmd

import (
	"fmt"

	"github.com/josejibin/bbgo/internal/bitbucket"
	"github.com/josejibin/bbgo/internal/config"
	"github.com/josejibin/bbgo/internal/secrets"
	"github.com/urfave/cli/v2"
)

func ConfigCommands() *cli.Command {
	return &cli.Command{
		Name:  "config",
		Usage: "Manage bbgo configuration",
		Subcommands: []*cli.Command{
			{
				Name:  "set",
				Usage: "Set configuration values",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "token", Usage: "Bitbucket API token"},
					&cli.StringFlag{Name: "workspace", Usage: "Bitbucket workspace"},
					&cli.StringFlag{Name: "repo", Usage: "Default repo slug (workspace/repo)"},
				},
				Action: configSet,
			},
			{
				Name:  "login",
				Usage: "Log in via OAuth in the browser — actions are attributed to your own user",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "client-id", Usage: "OAuth client key (required on first login)", EnvVars: []string{"BBGO_OAUTH_CLIENT_ID"}},
					&cli.StringFlag{Name: "client-secret", Usage: "OAuth client secret (required on first login; prefer the env var on shared machines — flags are visible in `ps`)", EnvVars: []string{"BBGO_OAUTH_CLIENT_SECRET"}},
					&cli.IntFlag{Name: "port", Value: 8976, Usage: "localhost callback port (must match the client callback URL)"},
				},
				Action: configLogin,
			},
			{
				Name:   "logout",
				Usage:  "Remove the stored OAuth session",
				Action: configLogout,
			},
			{
				Name:   "show",
				Usage:  "Show current configuration",
				Action: configShow,
			},
			{
				Name:   "clear-token",
				Usage:  "Remove token from keychain",
				Action: configClearToken,
			},
			{
				Name:   "verify",
				Usage:  "Verify API credentials",
				Action: configVerify,
			},
		},
	}
}

func configSet(c *cli.Context) error {
	cfgPath := c.String("config")
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return err
	}

	changed := false

	if token := c.String("token"); token != "" {
		if err := secrets.StoreToken(token); err != nil {
			return fmt.Errorf("storing token: %w", err)
		}
		fmt.Println("Token stored securely.")
		changed = true
	}

	if ws := c.String("workspace"); ws != "" {
		cfg.Workspace = ws
		changed = true
	}

	if repo := c.String("repo"); repo != "" {
		cfg.DefaultRepo = repo
		changed = true
	}

	if changed {
		if err := config.Save(cfgPath, cfg); err != nil {
			return err
		}
		fmt.Println("Configuration saved.")
	} else {
		fmt.Println("No values specified. Use --token, --workspace, or --repo.")
	}

	return nil
}

func configLogin(c *cli.Context) error {
	clientID := c.String("client-id")
	clientSecret := c.String("client-secret")

	// Reuse client credentials from a previous login only when neither flag
	// is given — mixing a new client ID with an old stored secret (or vice
	// versa) would fail confusingly at the token exchange.
	if clientID == "" && clientSecret == "" {
		if prev, err := secrets.LoadOAuth(); err == nil && prev != nil {
			clientID = prev.ClientID
			clientSecret = prev.ClientSecret
		}
	}
	if clientID == "" || clientSecret == "" {
		return fmt.Errorf(`OAuth client credentials required for first login.

Ask a workspace admin to create an OAuth client:
  Bitbucket → Workspace settings → OAuth clients → Add client
  Callback URL: http://localhost:%d/callback
  Permissions: Account (read), Repositories (write), Pull requests (write)
  Check "This is a private consumer"

Then run: bbgo config login --client-id <key> --client-secret <secret>`, c.Int("port"))
	}

	app := bitbucket.NewOAuthApp(clientID, clientSecret)
	ts, err := app.BrowserLogin(c.Int("port"), openBrowser, c.App.Writer)
	if err != nil {
		return err
	}

	creds := &secrets.OAuthCredentials{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		AccessToken:  ts.AccessToken,
		RefreshToken: ts.RefreshToken,
		ExpiresAt:    ts.ExpiresAt,
	}
	if err := secrets.StoreOAuth(creds); err != nil {
		return fmt.Errorf("storing OAuth session: %w", err)
	}

	// Confirm identity so the user sees who PRs will be attributed to.
	client := bitbucket.NewClient(ts.AccessToken, getBool(c, "verbose"))
	var user struct {
		DisplayName string `json:"display_name"`
		Nickname    string `json:"nickname"`
	}
	if err := client.GetJSON("/2.0/user", &user); err != nil {
		fmt.Println("Logged in (could not fetch user profile).")
		return nil
	}
	fmt.Printf("Logged in as %s (%s). PRs and comments will be attributed to this user.\n", user.DisplayName, user.Nickname)
	return nil
}

func configLogout(c *cli.Context) error {
	if err := secrets.ClearOAuth(); err != nil {
		return err
	}
	fmt.Println("OAuth session removed.")
	return nil
}

func configShow(c *cli.Context) error {
	cfgPath := c.String("config")
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return err
	}

	fmt.Printf("workspace: %s\n", valueOrEmpty(cfg.Workspace))
	fmt.Printf("repo:      %s\n", valueOrEmpty(cfg.DefaultRepo))

	if creds, oerr := secrets.LoadOAuth(); oerr == nil && creds != nil {
		fmt.Println("auth:      OAuth [logged in]")
	} else if token, terr := secrets.LoadToken(); terr == nil && token != "" {
		fmt.Println("auth:      API token [stored in keychain]")
	} else {
		fmt.Println("auth:      [not set]")
	}

	return nil
}

func configClearToken(c *cli.Context) error {
	if err := secrets.ClearToken(); err != nil {
		return err
	}
	fmt.Println("Token removed.")
	return nil
}

func configVerify(c *cli.Context) error {
	client, err := newClient(c)
	if err != nil {
		return err
	}

	var result struct {
		Values []struct {
			Slug string `json:"slug"`
			Name string `json:"name"`
		} `json:"values"`
	}
	if err := client.GetJSON("/2.0/workspaces", &result); err != nil {
		return err
	}

	fmt.Println("Credentials verified. Accessible workspaces:")
	for _, ws := range result.Values {
		fmt.Printf("  %s (%s)\n", ws.Slug, ws.Name)
	}
	return nil
}

func valueOrEmpty(s string) string {
	if s == "" {
		return "(not set)"
	}
	return s
}
