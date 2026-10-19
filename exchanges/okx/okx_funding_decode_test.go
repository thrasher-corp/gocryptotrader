package okx

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

func TestFundingRateResponseUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, payload string
		present       bool
		want          float64
		invalid       bool
	}{
		{name: "zero string", payload: `{"realizedRate":"0"}`, present: true},
		{name: "zero number", payload: `{"realizedRate":0}`, present: true},
		{name: "positive", payload: `{"realizedRate":"0.001"}`, present: true, want: 0.001},
		{name: "negative", payload: `{"realizedRate":"-0.001"}`, present: true, want: -0.001},
		{name: "missing", payload: `{}`},
		{name: "empty", payload: `{"realizedRate":""}`},
		{name: "null", payload: `{"realizedRate":null}`},
		{name: "invalid", payload: `{"realizedRate":"unknown"}`, invalid: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := FundingRateResponse{RealisedRatePresent: true, RealisedRate: 1}
			err := json.Unmarshal([]byte(tc.payload), &r)
			if tc.invalid {
				require.Error(t, err, "invalid rate must fail decoding")
				return
			}
			require.NoError(t, err, "valid rate must decode")
			assert.Equal(t, tc.present, r.RealisedRatePresent, "presence should match settled evidence")
			assert.Equal(t, tc.want, r.RealisedRate.Float64(), "rate should preserve the supplied value")
		})
	}
}
