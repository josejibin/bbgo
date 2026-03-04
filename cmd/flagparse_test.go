package cmd

import (
	"flag"
	"testing"

	"github.com/urfave/cli/v2"
)

func TestStringFlagFromArgs(t *testing.T) {
	tests := []struct {
		name  string
		args  []string
		names []string
		want  string
	}{
		{"--flag value", []string{"42", "--output", "json"}, []string{"output", "o"}, "json"},
		{"--flag=value", []string{"42", "--output=json"}, []string{"output", "o"}, "json"},
		{"-o value", []string{"42", "-o", "json"}, []string{"output", "o"}, "json"},
		{"-o=value", []string{"42", "-o=json"}, []string{"output", "o"}, "json"},
		{"not present", []string{"42"}, []string{"output", "o"}, ""},
		{"flag without value", []string{"42", "--output"}, []string{"output", "o"}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := cli.NewApp()
			ctx := cli.NewContext(app, &flag.FlagSet{}, nil)
			ctx.Args()
			// Build a context with args
			set := &flag.FlagSet{}
			set.Parse(tt.args)
			ctx = cli.NewContext(app, set, nil)

			got := stringFlagFromArgs(ctx, tt.names...)
			if got != tt.want {
				t.Errorf("stringFlagFromArgs() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBoolFlagFromArgs(t *testing.T) {
	tests := []struct {
		name  string
		args  []string
		names []string
		want  bool
	}{
		{"present", []string{"42", "--stat"}, []string{"stat"}, true},
		{"not present", []string{"42"}, []string{"stat"}, false},
		{"different flag", []string{"42", "--other"}, []string{"stat"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := cli.NewApp()
			set := &flag.FlagSet{}
			set.Parse(tt.args)
			ctx := cli.NewContext(app, set, nil)

			got := boolFlagFromArgs(ctx, tt.names...)
			if got != tt.want {
				t.Errorf("boolFlagFromArgs() = %v, want %v", got, tt.want)
			}
		})
	}
}
