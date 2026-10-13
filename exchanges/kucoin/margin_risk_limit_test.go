package kucoin

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetMarginRiskLimit(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarginRiskLimit(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetMarginRiskLimit must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *MarginRiskLimitRequest
		exp  []MarginRiskLimit
	}{
		{
			name: "cross",
			req:  &MarginRiskLimitRequest{Currency: currency.BTC},
			exp: []MarginRiskLimit{
				{
					Timestamp:         types.Time(time.UnixMilli(1729678659275)),
					Currency:          currency.BTC,
					BorrowMaxAmount:   75.15,
					BuyMaxAmount:      217.12,
					HoldMaxAmount:     217.12,
					MarginCoefficient: 1,
					Precision:         8,
					BorrowMinAmount:   0.001,
					BorrowMinUnit:     0.0001,
					BorrowEnabled:     true,
					BorrowCoefficient: 1,
				},
			},
		},
		{
			name: "isolated",
			req:  &MarginRiskLimitRequest{IsIsolated: true, Symbol: "BTC-USDT"},
			exp: []MarginRiskLimit{
				{
					Timestamp:              types.Time(time.UnixMilli(1729685173576)),
					Symbol:                 "BTC-USDT",
					BaseMaxBorrowAmount:    75.4,
					QuoteMaxBorrowAmount:   6000000,
					BaseMaxBuyAmount:       217.84,
					QuoteMaxBuyAmount:      8000000,
					BaseMaxHoldAmount:      217.84,
					QuoteMaxHoldAmount:     8000000,
					BasePrecision:          8,
					QuotePrecision:         8,
					BaseBorrowMinAmount:    0.001,
					QuoteBorrowMinAmount:   10,
					BaseBorrowMinUnit:      0.0001,
					QuoteBorrowMinUnit:     1,
					BaseBorrowEnabled:      true,
					QuoteBorrowEnabled:     true,
					BaseBorrowCoefficient:  1,
					QuoteBorrowCoefficient: 1,
					BaseMarginCoefficient:  1,
					QuoteMarginCoefficient: 1,
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetMarginRiskLimit(t.Context(), tc.req)
			require.NoError(t, err, "GetMarginRiskLimit must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetMarginRiskLimit should decode every field")
				return
			}
			assert.NotEmpty(t, result, "GetMarginRiskLimit should return risk limits")
		})
	}
}
