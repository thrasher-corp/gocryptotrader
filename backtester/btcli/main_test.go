package main

import (
	"errors"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func TestRejectPositionalArguments(t *testing.T) {
	errBefore := errors.New("before called")
	tests := []struct {
		name    string
		args    []string
		wantErr error
	}{
		{name: "named flag", args: []string{"btcli", "parent", "child", "--value", "set"}},
		{name: "global flag", args: []string{"btcli", "--host", "localhost", "parent", "child", "--value", "set"}},
		{name: "child before hook", args: []string{"btcli", "parent", "child", "--value", "before"}, wantErr: errBefore},
		{name: "parent positional", args: []string{"btcli", "parent", "unexpected"}, wantErr: errPositionalArgument},
		{name: "child positional", args: []string{"btcli", "parent", "child", "unexpected"}, wantErr: errPositionalArgument},
		{name: "trailing positional", args: []string{"btcli", "parent", "child", "--value", "set", "unexpected"}, wantErr: errPositionalArgument},
		{name: "empty positional", args: []string{"btcli", "parent", "child", ""}, wantErr: errPositionalArgument},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			command := &cli.Command{
				Name: "parent",
				Subcommands: []*cli.Command{{
					Name:  "child",
					Flags: []cli.Flag{&cli.StringFlag{Name: "value"}},
					Before: func(c *cli.Context) error {
						if c.String("value") == "before" {
							return errBefore
						}
						return nil
					},
				}},
			}
			rejectPositionalArguments([]*cli.Command{command})
			app := &cli.App{Flags: []cli.Flag{&cli.StringFlag{Name: "host"}}, Commands: []*cli.Command{command}}
			err := app.Run(tc.args)
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr, "command should return the expected error")
				return
			}
			assert.NoError(t, err, "valid flags should run without error")
		})
	}

	t.Run("original before", func(t *testing.T) {
		command := &cli.Command{Name: "test", Flags: []cli.Flag{&cli.StringFlag{Name: "value"}}, Before: func(*cli.Context) error { return errBefore }}
		rejectPositionalArguments([]*cli.Command{command})
		app := &cli.App{Commands: []*cli.Command{command}}
		assert.ErrorIs(t, app.Run([]string{"btcli", "test", "--value", "set"}), errBefore, "original before hook should be preserved")
	})
}

func TestMainRejectsPositionalArguments(t *testing.T) {
	t.Parallel()
	if os.Getenv("BTCLI_TEST_MAIN") == "1" {
		os.Args = []string{"btcli", "--cert", os.DevNull, "listalltasks", "unexpected"}
		main()
		return
	}
	cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestMainRejectsPositionalArguments$") //nolint:gosec // re-runs this test binary to exercise main
	cmd.Env = append(os.Environ(), "BTCLI_TEST_MAIN=1")
	out, err := cmd.CombinedOutput()
	require.Error(t, err, "main must exit with an error")
	assert.Contains(t, string(out), errPositionalArgument.Error(), "main should reject the positional argument")
}

func TestMainShowsRequiredFlags(t *testing.T) {
	t.Parallel()
	if os.Getenv("BTCLI_TEST_HELP") == "1" {
		os.Args = []string{"btcli", "starttask", "--help"}
		main()
		return
	}
	cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestMainShowsRequiredFlags$") //nolint:gosec // re-runs this test binary to exercise main
	cmd.Env = append(os.Environ(), "BTCLI_TEST_HELP=1")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "main must show command help")
	assert.Contains(t, string(out), "(required)", "command help should mark required flags")
}
