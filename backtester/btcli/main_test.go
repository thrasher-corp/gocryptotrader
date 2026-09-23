package main

import (
	"errors"
	"testing"

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
				require.ErrorIs(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}

	t.Run("original before", func(t *testing.T) {
		command := &cli.Command{Name: "test", Flags: []cli.Flag{&cli.StringFlag{Name: "value"}}, Before: func(*cli.Context) error { return errBefore }}
		rejectPositionalArguments([]*cli.Command{command})
		app := &cli.App{Commands: []*cli.Command{command}}
		require.ErrorIs(t, app.Run([]string{"btcli", "test", "--value", "set"}), errBefore)
	})
}
