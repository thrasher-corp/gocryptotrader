package gate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

func TestGetFee(t *testing.T) {
	t.Parallel()
	_, err := e.GetFee(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFee must reject a nil fee builder")

	result, err := e.GetFee(t.Context(), &exchange.FeeBuilder{FeeType: exchange.OfflineTradeFee, PurchasePrice: 1000, Amount: 2})
	require.NoError(t, err, "GetFee must not error for the offline trade fee")
	assert.Equal(t, 4.0, result, "GetFee should return the worst case trade fee")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err = e.GetFee(t.Context(), &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyTradeFee, Pair: currency.NewBTCUSDT(), PurchasePrice: 1000, Amount: 2, IsMaker: true})
	require.NoError(t, err, "GetFee must not error")
	if mockTests {
		assert.Equal(t, 4.0, result, "GetFee should return the account's maker fee")
	}
}
