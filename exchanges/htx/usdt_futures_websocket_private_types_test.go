package htx

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestV5WsRateLimitUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		raw     string
		empty   bool
		wantErr bool
	}{
		{name: "object", raw: `{"limit":"24","interval":"3000","remaining":"19","reset":"1772779491561"}`},
		{name: "encoded string", raw: `"{\"limit\":\"24\",\"interval\":\"3000\",\"remaining\":\"19\",\"reset\":\"1772779491561\"}"`},
		{name: "null", raw: `null`, empty: true},
		{name: "empty string", raw: `""`, wantErr: true},
		{name: "invalid string", raw: `"not JSON"`, wantErr: true},
		{name: "array", raw: `[]`, wantErr: true},
		{name: "malformed", raw: `{`, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			state := V5WsRateLimit{Limit: 99}
			err := state.UnmarshalJSON([]byte(tc.raw))
			if tc.wantErr {
				require.Error(t, err, "invalid state must return a decoding error")
				assert.Equal(t, types.Number(99), state.Limit, "failed decode should preserve existing state")
				return
			}
			require.NoError(t, err, "rate limit must decode")
			if tc.empty {
				assert.Equal(t, V5WsRateLimit{}, state, "null should clear state")
				return
			}
			assert.Equal(t, types.Number(24), state.Limit)
			assert.Equal(t, types.Number(3000), state.Interval)
			assert.Equal(t, types.Number(19), state.Remaining)
			assert.Equal(t, time.UnixMilli(1772779491561), state.Reset.Time())
			var response V5WsOrderResponse
			require.NoError(t, json.Unmarshal([]byte(`{"rate_limit":`+tc.raw+`}`), &response), "embedded trade response must decode")
			assert.Equal(t, state, response.RateLimit, "embedded rate limit should match standalone state")
		})
	}
}
