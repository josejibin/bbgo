package cmd

import (
	"fmt"

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

func configShow(c *cli.Context) error {
	cfgPath := c.String("config")
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return err
	}

	fmt.Printf("workspace: %s\n", valueOrEmpty(cfg.Workspace))
	fmt.Printf("repo:      %s\n", valueOrEmpty(cfg.DefaultRepo))

	// Check if token exists
	token, err := secrets.LoadToken()
	if err == nil && token != "" {
		fmt.Println("token:     [stored in keychain]")
	} else {
		fmt.Println("token:     [not set]")
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
