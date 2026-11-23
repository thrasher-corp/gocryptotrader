package main

import (
	"flag"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func TestGetFuturePositionOrders(t *testing.T) {
	// pairDelimiter is a process-wide CLI setting, so these cases run sequentially.
	originalDelimiter := pairDelimiter
	pairDelimiter = "-"
	t.Cleanup(func() { pairDelimiter = originalDelimiter })

	for _, tc := range []struct {
		name       string
		start      string
		end        string
		parseValue string
		message    string
	}{
		{name: "invalid start", start: "invalid-start", end: "2020-01-02 00:00:00", parseValue: "invalid-start", message: "invalid time format for start:"},
		{name: "invalid end", start: "2020-01-01 00:00:00", end: "invalid-end", parseValue: "invalid-end", message: "invalid time format for end:"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			flags := flag.NewFlagSet("getfuturepositionorders", flag.ContinueOnError)
			flags.String(assetFlagName, "futures", "")
			flags.String(pairFlagName, "BTC-USD", "")
			flags.String(startFlagName, "", "")
			flags.String(endFlagName, "", "")
			require.NoError(t, flags.Parse([]string{"--start=" + tc.start, "--end=" + tc.end}), "date flags must parse as strings")

			err := getFuturePositionOrders(cli.NewContext(nil, flags, nil))
			var parseErr *time.ParseError
			require.ErrorAs(t, err, &parseErr, "invalid dates must retain the underlying parse error")
			assert.Equal(t, tc.parseValue, parseErr.Value, "parse error should identify the invalid date")
			assert.ErrorContains(t, err, tc.message, "error should identify the date flag that failed")
		})
	}
}
