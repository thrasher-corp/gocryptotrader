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
		{name: "named flag", args: []string{"gctcli", "parent", "child", "--value", "set"}},
		{name: "global flag", args: []string{"gctcli", "--host", "localhost", "parent", "child", "--value", "set"}},
		{name: "parent positional", args: []string{"gctcli", "parent", "unexpected"}, wantErr: errPositionalArgument},
		{name: "child positional", args: []string{"gctcli", "parent", "child", "unexpected"}, wantErr: errPositionalArgument},
		{name: "trailing positional", args: []string{"gctcli", "parent", "child", "--value", "set", "unexpected"}, wantErr: errPositionalArgument},
		{name: "empty positional", args: []string{"gctcli", "parent", "child", ""}, wantErr: errPositionalArgument},
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
		require.ErrorIs(t, app.Run([]string{"gctcli", "test", "--value", "set"}), errBefore)
	})
}

func TestMainRejectsPositionalArguments(t *testing.T) {
	t.Parallel()
	if os.Getenv("GCTCLI_TEST_MAIN") == "1" {
		os.Args = []string{"gctcli", "--cert", os.DevNull, "getexchanges", "unexpected"}
		main()
		return
	}
	cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestMainRejectsPositionalArguments$") //nolint:gosec // re-runs this test binary to exercise main
	cmd.Env = append(os.Environ(), "GCTCLI_TEST_MAIN=1")
	out, err := cmd.CombinedOutput()
	require.Error(t, err, "main must exit with an error")
	assert.Contains(t, string(out), errPositionalArgument.Error(), "main should reject the positional argument")
}

func TestRequiredCommandFlags(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		command  *cli.Command
		flagName string
		required bool
	}{
		{name: "ticker exchange", command: getTickerCommand, flagName: exchangeFlag, required: true},
		{name: "ticker pair", command: getTickerCommand, flagName: pairFlag, required: true},
		{name: "ticker asset", command: getTickerCommand, flagName: assetFlag, required: true},
		{name: "submit side", command: submitOrderCommand, flagName: sideFlag, required: true},
		{name: "submit amount", command: submitOrderCommand, flagName: amountFlag, required: true},
		{name: "submit margin optional", command: submitOrderCommand, flagName: "margintype"},
		{name: "cancel pair optional", command: cancelOrderCommand, flagName: pairFlag},
		{name: "open interest pair optional", command: futuresCommands.Command("getopeninterest"), flagName: pairFlag},
		{name: "set leverage margin type optional", command: futuresCommands.Command("setleverage"), flagName: "margintype"},
		{name: "get leverage margin type optional", command: futuresCommands.Command("getleverage"), flagName: "margintype"},
		{name: "change position margin type optional", command: futuresCommands.Command("changepositionmargin"), flagName: "margintype"},
		{name: "set margin type required", command: futuresCommands.Command("setmargintype"), flagName: "margintype", required: true},
		{name: "set proxy optional", command: websocketManagerCommand.Command("setproxy"), flagName: "proxy"},
		{name: "set URL optional", command: websocketManagerCommand.Command("seturl"), flagName: "url"},
		{name: "margin rates currency", command: getMarginRatesHistoryCommand, flagName: "currency", required: true},
		{name: "job id alternative", command: dataHistoryCommands.Command("getajob"), flagName: "id"},
		{name: "job nickname alternative", command: dataHistoryCommands.Command("getajob"), flagName: "nickname"},
		{name: "logger level", command: setLoggerDetailsCommand, flagName: "level", required: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			for _, flag := range tc.command.Flags {
				if flag.Names()[0] != tc.flagName {
					continue
				}
				required, ok := flag.(cli.RequiredFlag)
				require.True(t, ok)
				require.Equal(t, tc.required, required.IsRequired())
				return
			}
			t.Fatalf("flag %q missing from %s", tc.flagName, tc.command.Name)
		})
	}
}

func TestWithdrawalRequestByExchangeID(t *testing.T) {
	for _, limit := range []string{"2147483648", "-2147483649"} {
		t.Run(limit, func(t *testing.T) {
			app := &cli.App{Commands: []*cli.Command{withdrawalRequestCommand}}
			err := app.Run([]string{"gctcli", "withdrawalrequesthistory", "byexchange", "--exchange", "Binance", "--asset", "spot", "--limit", limit})
			assert.ErrorIs(t, err, errWithdrawalLimitOutOfRange, "out-of-range limit should be rejected before RPC setup")
		})
	}
}

func TestWithdrawalRequestByDate(t *testing.T) {
	for _, limit := range []string{"2147483648", "-2147483649"} {
		t.Run(limit, func(t *testing.T) {
			app := &cli.App{Commands: []*cli.Command{withdrawalRequestCommand}}
			err := app.Run([]string{"gctcli", "withdrawalrequesthistory", "bydate", "--exchange", "Binance", "--limit", limit})
			assert.ErrorIs(t, err, errWithdrawalLimitOutOfRange, "out-of-range limit should be rejected before RPC setup")
		})
	}
}
