package mexc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// TestUnsupportedOperations asserts the operations MEXC spot does not offer report it, rather than
// returning a zero value a caller could mistake for success.
func TestUnsupportedOperations(t *testing.T) {
	t.Parallel()

	t.Run("ModifyOrder", func(t *testing.T) {
		t.Parallel()
		_, err := e.ModifyOrder(t.Context(), &order.Modify{})
		assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "ModifyOrder should report it is unsupported")
	})
	t.Run("CancelBatchOrders", func(t *testing.T) {
		t.Parallel()
		_, err := e.CancelBatchOrders(t.Context(), nil)
		assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "CancelBatchOrders should report it is unsupported")
	})
	t.Run("WithdrawCryptocurrencyFunds", func(t *testing.T) {
		t.Parallel()
		_, err := e.WithdrawCryptocurrencyFunds(t.Context(), &withdraw.Request{})
		assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "WithdrawCryptocurrencyFunds should report it is unsupported")
	})
	t.Run("WithdrawFiatFunds", func(t *testing.T) {
		t.Parallel()
		_, err := e.WithdrawFiatFunds(t.Context(), &withdraw.Request{})
		assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "WithdrawFiatFunds should report it is unsupported")
	})
	t.Run("WithdrawFiatFundsToInternationalBank", func(t *testing.T) {
		t.Parallel()
		_, err := e.WithdrawFiatFundsToInternationalBank(t.Context(), &withdraw.Request{})
		assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "WithdrawFiatFundsToInternationalBank should report it is unsupported")
	})
	t.Run("GetFuturesContractDetails", func(t *testing.T) {
		t.Parallel()
		_, err := e.GetFuturesContractDetails(t.Context(), asset.Futures)
		assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "GetFuturesContractDetails should report it is unsupported")
	})
	t.Run("GetLatestFundingRates", func(t *testing.T) {
		t.Parallel()
		_, err := e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{})
		assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "GetLatestFundingRates should report it is unsupported")
	})
}

// TestGetFeeByTypeOffline asserts the offline fee estimate, which is the path taken when no
// credentials are available to ask the exchange for the account's own schedule.
func TestGetFeeByTypeOffline(t *testing.T) {
	t.Parallel()
	maker, err := e.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.OfflineTradeFee, IsMaker: true})
	require.NoError(t, err, "the offline maker fee must not error")
	assert.Zero(t, maker, "MEXC spot charges no maker fee")

	taker, err := e.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.OfflineTradeFee})
	require.NoError(t, err, "the offline taker fee must not error")
	assert.Equal(t, 0.0005, taker, "the offline taker fee should be 5 bps")
}
