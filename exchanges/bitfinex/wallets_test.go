package bitfinex

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

func TestGetWallets(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	result, err := e.GetWallets(t.Context())
	require.NoError(t, err, "GetWallets must not error")
	if mockTests {
		exp := []Wallet{{
			Type:             "exchange",
			Currency:         currency.UST,
			Balance:          19788.6529257,
			AvailableBalance: new(19788.6529257),
			LastChange:       "Exchange 2.0 UST for USD @ 11.696",
			LastChangeMetadata: &WalletChangeMetadata{
				Reason:          "TRADE",
				OrderID:         1189740779,
				OppositeOrderID: 1189785673,
				TradePrice:      11.696,
				TradeAmount:     -2,
				OrderClientID:   1598516362757,
				OrderGroupID:    1598516362629,
			},
		}, {
			Type:     "funding",
			Currency: currency.BTC,
			Balance:  0.5,
		}, {
			Type:             "margin",
			Currency:         currency.NewCode("USTF0"),
			Balance:          500,
			AvailableBalance: new(300.0),
		}, {
			Type:             "margin",
			Currency:         currency.NewCode("BTCF0"),
			Balance:          0.01,
			AvailableBalance: new(0.01),
		}}
		assert.Equal(t, exp, result, "GetWallets should decode every wallet")
	}

	_, err = newUnavailableServerExchange(t).GetWallets(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetWallets should return the request failure")
}
