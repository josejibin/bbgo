package cmd

import (
	"flag"
	"testing"

	"github.com/urfave/cli/v2"
)

func testCLIContext(args []string) *cli.Context {
	app := cli.NewApp()
	set := flag.NewFlagSet("test", flag.ContinueOnError)
	_ = set.Parse(args)
	return cli.NewContext(app, set, nil)
}

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
			ctx := testCLIContext(tt.args)
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
			ctx := testCLIContext(tt.args)
			got := boolFlagFromArgs(ctx, tt.names...)
			if got != tt.want {
				t.Errorf("boolFlagFromArgs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetOptionalInt(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    int
		set     bool
		wantErr bool
	}{
		{"not present", []string{"42"}, 0, false, false},
		{"value after flag", []string{"42", "--line", "12"}, 12, true, false},
		{"value with equals", []string{"42", "--line=12"}, 12, true, false},
		{"invalid", []string{"42", "--line", "12x"}, 0, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, set, err := getOptionalInt(testCLIContext(tt.args), "line")
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want || set != tt.set {
				t.Fatalf("getOptionalInt() = (%d, %v), want (%d, %v)", got, set, tt.want, tt.set)
			}
		})
	}
}
