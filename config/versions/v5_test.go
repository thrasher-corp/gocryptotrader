package versions

import (
	"bytes"
	"context"
	"encoding/json" //nolint:depguard // Direct use of golang json for Compact func
	"strings"
	"testing"

	"github.com/buger/jsonparser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVersion5Upgrade(t *testing.T) {
	t.Parallel()

	expDef := `{"orderManager":{"enabled":true,"verbose":false,"activelyTrackFuturesPositions":true,"futuresTrackingSeekDuration":31536000000000000,"cancelOrdersOnShutdown":false,"respectOrderHistoryLimits":true}}`
	expUser1 := `{"orderManager":{"enabled":false,"verbose":true,"activelyTrackFuturesPositions":false,"futuresTrackingSeekDuration":47000,"cancelOrdersOnShutdown":true,"respectOrderHistoryLimits":true}}`
	expUser2 := strings.Replace(expUser1, `mits":true`, `mits":false`, 1)

	tests := []struct {
		name string
		in   string
		out  string
		err  error
	}{
		{name: "Bad input should error", err: jsonparser.KeyPathNotFoundError},
		{name: "Missing orderManager should use the defaults", in: "{}", out: expDef},
		{name: "Enabled null should use defaults", in: strings.Replace(expDef, "true", "null", 1), out: expDef},
		{name: "RespectOrderHistoryLimits should be added if missing", in: strings.Replace(expUser1, `,"respectOrderHistoryLimits":true`, "", 1), out: expUser1},
		{name: "RespectOrderHistoryLimits null should default true", in: strings.Replace(expUser1, `mits":true`, `mits":null`, 1), out: expUser1},
		{name: "FutureTracking should be reversed", in: strings.Replace(expUser1, "47", "-47", 1), out: expUser1},
		{name: "Configured orderManager should be left alone", in: expUser2, out: expUser2},
	}

	for _, tt := range tests {
		_ = t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			out, err := new(Version5).UpgradeConfig(context.Background(), []byte(tt.in))
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
			b := new(bytes.Buffer)
			require.NoError(t, json.Compact(b, out), "json.Compact must not error")
			require.Equal(t, tt.out, b.String())
		})
	}
}

func TestVersion5Downgrade(t *testing.T) {
	t.Parallel()

	in := `{"orderManager":{"enabled":false,"verbose":true,"activelyTrackFuturesPositions":false,"futuresTrackingSeekDuration":-47000,"cancelOrdersOnShutdown":true,"respectOrderHistoryLimits":true}}`
	exp := `{"orderManager":{"enabled":false,"verbose":true,"activelyTrackFuturesPositions":false,"futuresTrackingSeekDuration":-47000,"cancelOrdersOnShutdown":true,"respectOrderHistoryLimits":true}}`
	out, err := new(Version5).DowngradeConfig(context.Background(), []byte(in))
	require.NoError(t, err)
	assert.Equal(t, exp, string(out), "DowngradeConfig should just reverse the futuresTrackingSeekDuration")

	out, err = new(Version5).DowngradeConfig(context.Background(), []byte(exp))
	require.NoError(t, err)
	assert.Equal(t, exp, string(out), "DowngradeConfig should leave an already negative futuresTrackingSeekDuration alone")
}
