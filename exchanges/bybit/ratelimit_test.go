package bybit

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

func TestGetWSRateLimitEPLByCategory(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		category string
		expected request.EndpointLimit
		err      error
	}{
		{"", 0, errUnknownCategory},
		{cSpot, wsOrderSpotEPL, nil},
		{cInverse, wsOrderInverseEPL, nil},
		{cLinear, wsOrderLinearEPL, nil},
		{cOption, wsOrderOptionsEPL, nil},
	} {
		t.Run(tc.category, func(t *testing.T) {
			t.Parallel()
			actual, err := getWSRateLimitEPLByCategory(tc.category)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expected, actual)
		})
	}
}
