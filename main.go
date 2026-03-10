package main

import (
	"fmt"
	"os"

	"github.com/josejibin/bbgo/cmd"
	"github.com/josejibin/bbgo/internal/secrets"
	"github.com/urfave/cli/v2"
)

var version = "dev"

func main() {
	errWriter := secrets.NewRedactWriter(os.Stderr)

	app := &cli.App{
		Name:    "bbgo",
		Usage:   "Bitbucket Cloud CLI",
		Version: version,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "repo",
				Aliases: []string{"r"},
				Usage:   "workspace/repo override",
				EnvVars: []string{"BBGO_REPO"},
			},
			&cli.BoolFlag{
				Name:  "verbose",
				Usage: "print HTTP request details",
			},
			&cli.BoolFlag{
				Name:  "no-color",
				Usage: "disable ANSI color output",
			},
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Value:   "text",
				Usage:   "output format: text or json",
			},
			&cli.StringFlag{
				Name:  "config",
				Usage: "override config file path",
			},
		},
		Writer:    secrets.NewRedactWriter(os.Stdout),
		ErrWriter: errWriter,
		Before: func(c *cli.Context) error {
			// Try to load token (ignore errors — token may not be set yet)
			_, _ = secrets.LoadToken()
			return nil
		},
		ExitErrHandler: func(c *cli.Context, err error) {
			if err == nil {
				return
			}
			_, _ = fmt.Fprintf(errWriter, "Error: %v\n", err)
			os.Exit(cmd.ExitCodeForError(err))
		},
		Commands: []*cli.Command{
			cmd.ConfigCommands(),
			cmd.PRCommands(),
			cmd.CommentCommands(),
			cmd.ReviewCommands(),
			cmd.FileCommands(),
		},
	}

	if err := app.Run(os.Args); err != nil {
		// ExitErrHandler handles action errors; this catches setup errors.
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
