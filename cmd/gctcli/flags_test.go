package main

import (
	"io"
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
		args     []string
		defaults map[string]any
		values   map[string]any
	}{
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

			var contexts []*cli.Context
			for _, args := range [][]string{tc.args, nil} {
				command := &cli.Command{Name: tc.command.Name, HideHelp: true, Flags: flags, Action: func(c *cli.Context) error {
					contexts = append(contexts, c)
					return nil
				}}
				app := &cli.App{HideHelp: true, Writer: io.Discard, ErrWriter: io.Discard, Commands: []*cli.Command{command}}
				baseArgs := []string{"gctcli", command.Name, "--exchange=Binance", "--pair=BTC-USDT", "--asset=spot"}
				if command.Name == "convertcandles" {
					baseArgs = append(baseArgs, "--conversion_interval=300")
				}
				if tc.command == dataHistoryJobCommands.Command(command.Name) {
					baseArgs = append(baseArgs, "--interval=60")
				}
				require.NoError(t, app.Run(append(baseArgs, args...)), "command flags must parse")
			}

			require.Len(t, contexts, 2, "both invocations must reach the action")
			for name, want := range tc.values {
				assert.Equal(t, want, contexts[0].Value(name), "earlier command values should survive a subsequent parse")
			}
			for name, want := range tc.defaults {
				assert.Equal(t, want, contexts[1].Value(name), "omitted flags should retain their command defaults")
			}
		})
	}
}
