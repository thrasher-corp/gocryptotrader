package main

import (
	"io"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/urfave/cli/v2"
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
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Each parser owns its flag metadata so the cases can run concurrently.
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

func TestStartTask(t *testing.T) {
	app := &cli.App{Commands: []*cli.Command{startTaskCommand}}
	require.ErrorIs(t, app.Run([]string{"btcli", "starttask", "--id", ""}), errTaskIDRequired)
}

func TestStopTask(t *testing.T) {
	app := &cli.App{Commands: []*cli.Command{stopTaskCommand}}
	require.ErrorIs(t, app.Run([]string{"btcli", "stoptask", "--id", ""}), errTaskIDRequired)
}

func TestClearTask(t *testing.T) {
	app := &cli.App{Commands: []*cli.Command{clearTaskCommand}}
	require.ErrorIs(t, app.Run([]string{"btcli", "cleartask", "--id", ""}), errTaskIDRequired)
}
