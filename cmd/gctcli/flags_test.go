package main

import (
	"io"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func TestCommandFlagValues(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name     string
		command  *cli.Command
		baseArgs []string
		args     []string
		defaults map[string]any
		values   map[string]any
	}{
		{
			name: "ticker required flags", command: getTickerCommand,
			values: map[string]any{"exchange": "Binance", "pair": "BTC-USDT", "asset": "spot"},
		},
		{
			name: "candle identity aliases", command: getHistoricCandlesExtendedCommand,
			baseArgs: []string{"-e=Binance", "-p=BTC-USDT", "-a=spot"},
			values:   map[string]any{"exchange": "Binance", "pair": "BTC-USDT", "asset": "spot"},
		},
		{
			name: "futures identity aliases", command: futuresCommands.Command("getleverage"),
			baseArgs: []string{"-e=Binance", "-p=BTC-USDT", "-a=futures"},
			values:   map[string]any{"exchange": "Binance", "pair": "BTC-USDT", "asset": "futures"},
		},
		{
			name: "cancel optional pair and side", command: cancelOrderCommand,
			baseArgs: []string{"--exchange=Binance", "--asset=spot", "--order_id=123"},
			args:     []string{"--pair=BTC-USDT", "--side=buy"},
			defaults: map[string]any{"pair": "", "side": ""},
			values:   map[string]any{"pair": "BTC-USDT", "side": "buy"},
		},
		{
			name: "open interest optional asset and pair", command: futuresCommands.Command("getopeninterest"),
			baseArgs: []string{"-e=Binance"},
			args:     []string{"-a=futures", "-p=BTC-USDT"},
			defaults: map[string]any{"asset": "", "pair": ""},
			values:   map[string]any{"asset": "futures", "pair": "BTC-USDT"},
		},
		{
			name: "modify optional amount", command: modifyOrderCommand,
			baseArgs: []string{"--exchange=Binance", "--asset=spot", "--pair=BTC-USDT", "--order_id=123"},
			args:     []string{"--amount=2.5"},
			defaults: map[string]any{"amount": float64(0)},
			values:   map[string]any{"amount": 2.5},
		},
		{
			name: "submit required side and amount", command: submitOrderCommand,
			baseArgs: []string{"--exchange=Binance", "--asset=spot", "--pair=BTC-USDT", "--side=buy", "--type=market", "--amount=2.5"},
			values:   map[string]any{"side": "buy", "amount": 2.5},
		},
		{
			name: "Bollinger bands", command: technicalAnalysisCommand.Command("bbands"),
			args:     []string{"--stddevdown=0", "--stddevup=2.5", "--period=0", "--movingaveragetype=ema", "-g=60"},
			defaults: map[string]any{"stddevdown": 1.5, "stddevup": 1.5, "period": int64(9), "movingaveragetype": "sma", "granularity": int64(86400)},
			values:   map[string]any{"stddevdown": float64(0), "stddevup": 2.5, "period": int64(0), "movingaveragetype": "ema", "granularity": int64(60)},
		},
		{
			name: "MACD", command: technicalAnalysisCommand.Command("macd"),
			args:     []string{"--fastperiod=0", "--slowperiod=30"},
			defaults: map[string]any{"fastperiod": int64(12), "slowperiod": int64(26), "period": int64(9)},
			values:   map[string]any{"fastperiod": int64(0), "slowperiod": int64(30), "period": int64(9)},
		},
		{
			name: "historic candles fill alias", command: getHistoricCandlesCommand,
			args:     []string{"--fill", "--rangesize=0", "-g=60"},
			defaults: map[string]any{"fillmissingdatawithtrades": false, "rangesize": int64(10), "granularity": int64(86400)},
			values:   map[string]any{"fillmissingdatawithtrades": true, "rangesize": int64(0), "granularity": int64(60)},
		},
		{
			name: "historic candles explicit false", command: getHistoricCandlesCommand,
			args:     []string{"--fillmissingdatawithtrades=false"},
			defaults: map[string]any{"fillmissingdatawithtrades": false},
			values:   map[string]any{"fillmissingdatawithtrades": false},
		},
		{
			name: "extended candles", command: getHistoricCandlesExtendedCommand,
			args:     []string{"--fill=false", "--sync=false", "--db=true", "-i=60", "--start=2020-01-01 00:00:00"},
			defaults: map[string]any{"fillmissingdatawithtrades": false, "sync": false, "db": false, "interval": int64(86400)},
			values:   map[string]any{"fillmissingdatawithtrades": false, "sync": false, "db": true, "interval": int64(60), "start": "2020-01-01 00:00:00"},
		},
		{
			name: "save candles", command: dataHistoryJobCommands.Command("savecandles"),
			args:     []string{"--request_size_limit=0", "--max_retry_attempts=0", "--batch_size=0", "--upsert=false", "--overwrite_existing_data=true"},
			defaults: map[string]any{"request_size_limit": uint64(500), "max_retry_attempts": uint64(3), "batch_size": uint64(3), "upsert": false, "overwrite_existing_data": false},
			values:   map[string]any{"request_size_limit": uint64(0), "max_retry_attempts": uint64(0), "batch_size": uint64(0), "upsert": false, "overwrite_existing_data": true},
		},
		{
			name: "save trades", command: dataHistoryJobCommands.Command("savetrades"),
			args:     []string{"--request_size_limit=20"},
			defaults: map[string]any{"request_size_limit": uint64(10)},
			values:   map[string]any{"request_size_limit": uint64(20)},
		},
		{
			name: "convert candles", command: dataHistoryJobCommands.Command("convertcandles"),
			args:     []string{"--conversion_interval=300", "--overwrite_existing_data=false"},
			defaults: map[string]any{"request_size_limit": uint64(500), "overwrite_existing_data": false},
			values:   map[string]any{"conversion_interval": uint64(300), "overwrite_existing_data": false},
		},
		{
			name: "validate candles replace", command: dataHistoryJobCommands.Command("validatecandles"),
			args:     []string{"--replace_on_issue=true", "--comparison_decimal_places=0"},
			defaults: map[string]any{"replace_on_issue": false, "request_size_limit": uint64(50), "comparison_decimal_places": uint64(3)},
			values:   map[string]any{"replace_on_issue": true, "comparison_decimal_places": uint64(0)},
		},
		{
			name: "validate candles explicit false", command: dataHistoryJobCommands.Command("validatecandles"),
			args:     []string{"--replace_on_issue=false"},
			defaults: map[string]any{"replace_on_issue": false},
			values:   map[string]any{"replace_on_issue": false},
		},
		{
			name: "validate candles bare boolean", command: dataHistoryJobCommands.Command("validatecandles"),
			args:     []string{"--replace_on_issue"},
			defaults: map[string]any{"replace_on_issue": false},
			values:   map[string]any{"replace_on_issue": true},
		},
		{
			name: "secondary validation", command: dataHistoryJobCommands.Command("secondaryvalidatecandles"),
			args:     []string{"--comparison_decimal_places=6"},
			defaults: map[string]any{"comparison_decimal_places": uint64(3), "request_size_limit": uint64(50)},
			values:   map[string]any{"comparison_decimal_places": uint64(6)},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// cli mutates flag metadata during parsing; each subtest owns its definitions.
			flags := make([]cli.Flag, 0, len(tc.command.Flags))
			for _, f := range tc.command.Flags {
				switch f := f.(type) {
				case *cli.StringFlag:
					copyFlag := *f
					flags = append(flags, &copyFlag)
				case *cli.Int64Flag:
					copyFlag := *f
					flags = append(flags, &copyFlag)
				case *cli.Uint64Flag:
					copyFlag := *f
					flags = append(flags, &copyFlag)
				case *cli.Float64Flag:
					copyFlag := *f
					flags = append(flags, &copyFlag)
				case *cli.BoolFlag:
					copyFlag := *f
					flags = append(flags, &copyFlag)
				default:
					require.FailNowf(t, "flag type must be supported", "unsupported flag type %T", f)
				}
			}

			baseArgs := []string{"gctcli", tc.command.Name}
			if tc.baseArgs != nil {
				baseArgs = append(baseArgs, tc.baseArgs...)
			} else {
				baseArgs = append(baseArgs, "--exchange=Binance", "--pair=BTC-USDT", "--asset=spot")
			}
			if tc.command.Name == "convertcandles" {
				baseArgs = append(baseArgs, "--conversion_interval=300")
			}
			if tc.command == dataHistoryJobCommands.Command(tc.command.Name) {
				baseArgs = append(baseArgs, "--interval=60")
			}

			var contexts []*cli.Context
			for _, args := range [][]string{tc.args, nil} {
				command := &cli.Command{Name: tc.command.Name, HideHelp: true, Flags: flags, Action: func(c *cli.Context) error {
					contexts = append(contexts, c)
					return nil
				}}
				app := &cli.App{HideHelp: true, Writer: io.Discard, ErrWriter: io.Discard, Commands: []*cli.Command{command}}
				require.NoError(t, app.Run(append(baseArgs, args...)), "command flags must parse")
			}

			require.Len(t, contexts, 2, "both invocations must reach the action")
			for name, want := range tc.values {
				assert.Equal(t, want, contexts[0].Value(name), "earlier command values should survive a subsequent parse")
			}
			for name, want := range tc.defaults {
				assert.Equal(t, want, contexts[1].Value(name), "omitted flags should retain their command defaults")
			}

			for omitted, arg := range baseArgs[2:] {
				missingName, _, _ := strings.Cut(strings.TrimLeft(arg, "-"), "=")
				for _, f := range flags {
					if slices.Contains(f.Names(), missingName) {
						missingName = f.Names()[0]
						break
					}
				}
				args := append([]string{"gctcli", tc.command.Name}, baseArgs[2:omitted+2]...)
				args = append(args, baseArgs[omitted+3:]...)
				called := false
				command := &cli.Command{Name: tc.command.Name, HideHelp: true, Flags: flags, Action: func(*cli.Context) error {
					called = true
					return nil
				}}
				app := &cli.App{HideHelp: true, Writer: io.Discard, ErrWriter: io.Discard, Commands: []*cli.Command{command}}
				err := app.Run(args)
				require.Error(t, err, "omitting a required flag must fail after successful parses")
				assert.Contains(t, err.Error(), missingName, "error should identify the missing required flag")
				assert.False(t, called, "missing required flags should prevent the action")
			}
		})
	}
}
