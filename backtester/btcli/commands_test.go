package main

import (
	"io"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/backtester/btrpc"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/urfave/cli/v2"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestExecuteStrategyFromFile(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name       string
		args       []string
		wantErr    error
		parseValue string
	}{
		{name: "empty path", args: []string{"--path="}, wantErr: errStrategyPathRequired},
		{name: "omitted overrides", wantErr: os.ErrNotExist},
		{name: "empty overrides", args: []string{"--starttimeoverride=", "--endtimeoverride="}, wantErr: os.ErrNotExist},
		{name: "start only", args: []string{"--starttimeoverride=2020-01-01 00:00:00"}, wantErr: os.ErrNotExist},
		{name: "end only", args: []string{"--endtimeoverride=2020-01-02 00:00:00"}, wantErr: os.ErrNotExist},
		{name: "valid range aliases", args: []string{"-s=2020-01-01 00:00:00", "-e=2020-01-02 00:00:00"}, wantErr: os.ErrNotExist},
		{name: "reversed range", args: []string{"--starttimeoverride=2020-01-02 00:00:00", "--endtimeoverride=2020-01-01 00:00:00"}, wantErr: common.ErrStartAfterEnd},
		{name: "equal dates", args: []string{"--starttimeoverride=2020-01-01 00:00:00", "--endtimeoverride=2020-01-01 00:00:00"}, wantErr: common.ErrStartEqualsEnd},
		{name: "invalid start", args: []string{"--starttimeoverride=invalid-start"}, parseValue: "invalid-start"},
		{name: "invalid end", args: []string{"--endtimeoverride=invalid-end"}, parseValue: "invalid-end"},
		{name: "negative interval", args: []string{"--intervaloverride=-1s"}, wantErr: errNegativeIntervalOverride},
		{name: "negative interval alias", args: []string{"-i=-1s"}, wantErr: errNegativeIntervalOverride},
		{name: "zero interval", args: []string{"--intervaloverride=0s"}, wantErr: os.ErrNotExist},
		{name: "positive interval", args: []string{"--intervaloverride=1m"}, wantErr: os.ErrNotExist},
		{name: "true boolean aliases", args: []string{"--dnr=true", "--dns=true"}, wantErr: os.ErrNotExist},
		{name: "explicit false booleans", args: []string{"--donotrunimmediately=false", "--donotstore=false"}, wantErr: os.ErrNotExist},
		{name: "do not run alias", args: []string{"--dnr"}, wantErr: os.ErrNotExist},
		{name: "do not store alias", args: []string{"--dns"}, wantErr: os.ErrNotExist},
		{name: "future range", args: []string{"--starttimeoverride=2999-01-01 00:00:00", "--endtimeoverride=3000-01-01 00:00:00"}, wantErr: common.ErrStartAfterTimeNow},
		{name: "smallest negative interval", args: []string{"--intervaloverride=-1ns"}, wantErr: errNegativeIntervalOverride},
		{name: "fractional minute interval alias", args: []string{"-i=90s"}, wantErr: os.ErrNotExist},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			flags := strategyFlags(t)
			command := &cli.Command{Name: executeStrategyFromFileCommand.Name, HideHelp: true, Flags: flags, Action: executeStrategyFromFile}
			app := &cli.App{HideHelp: true, Writer: io.Discard, ErrWriter: io.Discard, Commands: []*cli.Command{command}}
			args := append([]string{"btcli", command.Name, "--path=strategy.json"}, tc.args...)
			err := app.Run(args)
			if tc.parseValue != "" {
				var parseErr *time.ParseError
				require.ErrorAs(t, err, &parseErr, "date errors must preserve the underlying parse error")
				assert.Equal(t, tc.parseValue, parseErr.Value, "parse error should identify the invalid date")
				return
			}
			assert.ErrorIs(t, err, tc.wantErr, "command should validate overrides before attempting client setup")
		})
	}
}

func TestExecuteStrategyFromFileRequest(t *testing.T) {
	t.Parallel()

	start := time.Date(2020, 1, 1, 0, 0, 0, 0, time.Local)
	end := time.Date(2020, 1, 2, 0, 0, 0, 0, time.Local)
	for _, tc := range []struct {
		name       string
		args       []string
		wantErr    error
		parseValue string
		wantPath   string
		doNotRun   bool
		doNotStore bool
		start      time.Time
		end        time.Time
		interval   time.Duration
	}{
		{name: "empty path", args: []string{"--path="}, wantErr: errStrategyPathRequired},
		{name: "custom path", args: []string{"--path=another-strategy.json"}, wantPath: "another-strategy.json"},
		{name: "omitted overrides"},
		{name: "empty overrides", args: []string{"--starttimeoverride=", "--endtimeoverride="}},
		{name: "start only", args: []string{"--starttimeoverride=2020-01-01 00:00:00"}, start: start},
		{name: "end only", args: []string{"--endtimeoverride=2020-01-02 00:00:00"}, end: end},
		{name: "valid range aliases", args: []string{"-s=2020-01-01 00:00:00", "-e=2020-01-02 00:00:00"}, start: start, end: end},
		{name: "reversed range", args: []string{"--starttimeoverride=2020-01-02 00:00:00", "--endtimeoverride=2020-01-01 00:00:00"}, wantErr: common.ErrStartAfterEnd},
		{name: "equal dates", args: []string{"--starttimeoverride=2020-01-01 00:00:00", "--endtimeoverride=2020-01-01 00:00:00"}, wantErr: common.ErrStartEqualsEnd},
		{name: "invalid start", args: []string{"--starttimeoverride=invalid-start"}, parseValue: "invalid-start"},
		{name: "invalid end", args: []string{"--endtimeoverride=invalid-end"}, parseValue: "invalid-end"},
		{name: "negative interval", args: []string{"--intervaloverride=-1s"}, wantErr: errNegativeIntervalOverride},
		{name: "negative interval alias", args: []string{"-i=-1s"}, wantErr: errNegativeIntervalOverride},
		{name: "zero interval", args: []string{"--intervaloverride=0s"}},
		{name: "positive interval", args: []string{"--intervaloverride=1m"}, interval: time.Minute},
		{name: "true boolean aliases", args: []string{"--dnr=true", "--dns=true"}, doNotRun: true, doNotStore: true},
		{name: "explicit false booleans", args: []string{"--donotrunimmediately=false", "--donotstore=false"}},
		{name: "do not run alias", args: []string{"--dnr"}, doNotRun: true},
		{name: "do not store alias", args: []string{"--dns"}, doNotStore: true},
		{name: "future range", args: []string{"--starttimeoverride=2999-01-01 00:00:00", "--endtimeoverride=3000-01-01 00:00:00"}, wantErr: common.ErrStartAfterTimeNow},
		{name: "smallest negative interval", args: []string{"--intervaloverride=-1ns"}, wantErr: errNegativeIntervalOverride},
		{name: "fractional minute interval alias", args: []string{"-i=90s"}, interval: 90 * time.Second},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			flags := strategyFlags(t)
			var got *btrpc.ExecuteStrategyFromFileRequest
			command := &cli.Command{Name: executeStrategyFromFileCommand.Name, HideHelp: true, Flags: flags, Action: func(c *cli.Context) error {
				var err error
				got, err = executeStrategyFromFileRequest(c)
				return err
			}}
			app := &cli.App{HideHelp: true, Writer: io.Discard, ErrWriter: io.Discard, Commands: []*cli.Command{command}}
			err := app.Run(append([]string{"btcli", command.Name, "--path=strategy.json"}, tc.args...))
			if tc.parseValue != "" {
				var parseErr *time.ParseError
				require.ErrorAs(t, err, &parseErr, "date errors must preserve the underlying parse error")
				assert.Equal(t, tc.parseValue, parseErr.Value, "parse error should identify the invalid date")
				assert.Nil(t, got, "invalid dates should not produce a request")
				return
			}
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr, "invalid overrides must return the expected error")
				assert.Nil(t, got, "invalid overrides should not produce a request")
				return
			}
			require.NoError(t, err, "valid overrides must build a request")
			path := tc.wantPath
			if path == "" {
				path = "strategy.json"
			}
			want := &btrpc.ExecuteStrategyFromFileRequest{
				StrategyFilePath:    path,
				DoNotRunImmediately: tc.doNotRun,
				DoNotStore:          tc.doNotStore,
				StartTimeOverride:   timestamppb.New(tc.start),
				EndTimeOverride:     timestamppb.New(tc.end),
				IntervalOverride:    durationpb.New(tc.interval),
			}
			assert.Truef(t, proto.Equal(want, got), "request should preserve all flag values: got %s, want %s", got, want)
		})
	}
}

// strategyFlags copies the command's flags so parallel cases do not share cli's mutable metadata.
func strategyFlags(t *testing.T) []cli.Flag {
	t.Helper()
	flags := make([]cli.Flag, 0, len(executeStrategyFromFileCommand.Flags))
	for _, f := range executeStrategyFromFileCommand.Flags {
		switch f := f.(type) {
		case *cli.StringFlag:
			copyFlag := *f
			flags = append(flags, &copyFlag)
		case *cli.BoolFlag:
			copyFlag := *f
			flags = append(flags, &copyFlag)
		case *cli.DurationFlag:
			copyFlag := *f
			flags = append(flags, &copyFlag)
		default:
			require.FailNowf(t, "flag type must be supported", "unsupported flag type %T", f)
		}
	}
	return flags
}

func TestStrategyFlags(t *testing.T) {
	t.Parallel()

	first := strategyFlags(t)
	second := strategyFlags(t)
	require.Len(t, first, len(executeStrategyFromFileCommand.Flags), "first copy must contain every strategy flag")
	require.Len(t, second, len(executeStrategyFromFileCommand.Flags), "second copy must contain every strategy flag")
	for i, original := range executeStrategyFromFileCommand.Flags {
		t.Run(original.Names()[0], func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, original, first[i], "first copy should preserve the flag metadata")
			assert.Equal(t, original, second[i], "second copy should preserve the flag metadata")
			assert.NotSame(t, original, first[i], "first copy should own its flag metadata")
			assert.NotSame(t, original, second[i], "second copy should own its flag metadata")
			assert.NotSame(t, first[i], second[i], "separate copies should not share flag metadata")
		})
	}
}

func TestStartTask(t *testing.T) {
	app := &cli.App{Commands: []*cli.Command{startTaskCommand}}
	require.ErrorIs(t, app.Run([]string{"btcli", "starttask", "--id", ""}), errTaskIDRequired)
}

func TestTaskIDFlag(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		command *cli.Command
	}{
		{command: startTaskCommand},
		{command: stopTaskCommand},
		{command: clearTaskCommand},
	} {
		t.Run(tc.command.Name, func(t *testing.T) {
			t.Parallel()

			require.NotEmpty(t, tc.command.Flags, "task command must have its ID flag")
			flag, ok := tc.command.Flags[0].(*cli.StringFlag)
			require.True(t, ok, "task ID must be a string flag")
			// cli mutates flag metadata during parsing, so each parallel case owns a copy.
			copyFlag := *flag
			var contexts []*cli.Context
			command := &cli.Command{
				Name:     tc.command.Name,
				HideHelp: true,
				Flags:    []cli.Flag{&copyFlag},
				Action: func(c *cli.Context) error {
					contexts = append(contexts, c)
					return nil
				},
			}
			app := &cli.App{HideHelp: true, Writer: io.Discard, ErrWriter: io.Discard, Commands: []*cli.Command{command}}
			require.NoError(t, app.Run([]string{"btcli", command.Name, "--id=first-task"}), "explicit task ID must parse")
			assert.ErrorContains(t, app.Run([]string{"btcli", command.Name}), `Required flag "id" not set`, "subsequent invocation should still require its own task ID")
			require.NoError(t, app.Run([]string{"btcli", command.Name, "--id=second-task"}), "subsequent explicit task ID must parse")
			require.Len(t, contexts, 2, "only invocations with an ID must reach the action")
			assert.Equal(t, "first-task", contexts[0].String("id"), "earlier task ID should survive a subsequent parse")
			assert.Equal(t, "second-task", contexts[1].String("id"), "subsequent invocation should use its own task ID")
		})
	}
}

func TestStopTask(t *testing.T) {
	app := &cli.App{Commands: []*cli.Command{stopTaskCommand}}
	require.ErrorIs(t, app.Run([]string{"btcli", "stoptask", "--id", ""}), errTaskIDRequired)
}

func TestClearTask(t *testing.T) {
	app := &cli.App{Commands: []*cli.Command{clearTaskCommand}}
	require.ErrorIs(t, app.Run([]string{"btcli", "cleartask", "--id", ""}), errTaskIDRequired)
}
