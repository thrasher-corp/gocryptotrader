package gateio

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

func TestGetCrossExchangeSymbols(t *testing.T) {
	t.Parallel()
	_, err := e.GetCrossExchangeSymbols(t.Context(), nil)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	e.Verbose = true
	result, err := e.GetCrossExchangeSymbols(t.Context(), []string{"BINANCE_FUTURE_BTC_USDT"})
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestGetCrossExchangeRiskLimits(t *testing.T) {
	t.Parallel()
	_, err := e.GetCrossExchangeSymbols(t.Context(), nil)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	e.Verbose = true
	result, err := e.GetCrossExchangeRiskLimits(t.Context(), []string{"BINANCE_FUTURE_BTC_USDT", "BINANCE_FUTURE_ETH_USDT"})
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestGetCrossExchangeTransferCoins(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossExchangeTransferCoins(t.Context(), currency.EMPTYCODE)
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestGetCrossExchangeTransferHistory(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossExchangeTransferHistory(t.Context(), nil)
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestCrossExchangeFundTransfer(t *testing.T) {
	t.Parallel()
	_, err := e.CrossExchangeFundTransfer(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	_, err = e.CrossExchangeFundTransfer(t.Context(), &CrossExchangeTransferRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.CrossExchangeFundTransfer(t.Context(), &CrossExchangeTransferRequest{Coin: currency.BTC})
	require.ErrorIs(t, err, order.ErrAmountMustBeSet)

	_, err = e.CrossExchangeFundTransfer(t.Context(), &CrossExchangeTransferRequest{Coin: currency.BTC, Amount: 0.001})
	require.ErrorIs(t, err, errCrossExchangeFromAccountRequired)

	_, err = e.CrossExchangeFundTransfer(t.Context(), &CrossExchangeTransferRequest{Coin: currency.BTC, Amount: 0.001, From: "spot"})
	require.ErrorIs(t, err, errCrossExchangeToAccountRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CrossExchangeFundTransfer(t.Context(), &CrossExchangeTransferRequest{
		Coin: currency.BTC, Amount: 0.001, From: "spot", To: "crossex",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, result.TxID)
}

func TestCreateCrossExchangeOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CreateCrossExchangeOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	_, err = e.CreateCrossExchangeOrder(t.Context(), &CrossExchangeOrderCreateRequest{})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	_, err = e.CreateCrossExchangeOrder(t.Context(), &CrossExchangeOrderCreateRequest{Symbol: "BINANCE_FUTURE_BTC_USDT"})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CreateCrossExchangeOrder(t.Context(), &CrossExchangeOrderCreateRequest{
		Symbol:       "BINANCE_FUTURE_BTC_USDT",
		Side:         order.Buy,
		OrderType:    "GTC",
		Quantity:     1,
		Price:        65000,
		TimeInForce:  "GTC",
		ReduceOnly:   true,
		PositionSide: order.Short,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, result.OrderID)
}

func TestGetCrossExchangeOrderDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetCrossExchangeOrderDetails(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossExchangeOrderDetails(t.Context(), "20491522002333905922")
	require.NoError(t, err)
	assert.NotEmpty(t, result.OrderID)
}

func TestModifyCrossExchangeOrder(t *testing.T) {
	t.Parallel()
	_, err := e.ModifyCrossExchangeOrder(t.Context(), "", &CrossExchangeOrderUpdateRequest{})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	_, err = e.ModifyCrossExchangeOrder(t.Context(), "20491522002333905922", nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.ModifyCrossExchangeOrder(t.Context(), "20491522002333905922", &CrossExchangeOrderUpdateRequest{Price: 64000})
	require.NoError(t, err)
	assert.NotEmpty(t, result.OrderID)
}

func TestCancelCrossExchangeOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelCrossExchangeOrder(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelCrossExchangeOrder(t.Context(), "20491522002333905922")
	require.NoError(t, err)
	assert.NotEmpty(t, result.OrderID)
}

func TestGetCrossExchangeConvertQuote(t *testing.T) {
	t.Parallel()
	_, err := e.GetCrossExchangeConvertQuote(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	_, err = e.GetCrossExchangeConvertQuote(t.Context(), &CrossExchangeConvertQuoteRequest{})
	require.ErrorIs(t, err, errCrossExchangeExchangeTypeRequired)

	_, err = e.GetCrossExchangeConvertQuote(t.Context(), &CrossExchangeConvertQuoteRequest{ExchangeType: "BINANCE"})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.GetCrossExchangeConvertQuote(t.Context(), &CrossExchangeConvertQuoteRequest{ExchangeType: "BINANCE", FromCoin: currency.BTC})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.GetCrossExchangeConvertQuote(t.Context(), &CrossExchangeConvertQuoteRequest{ExchangeType: "BINANCE", FromCoin: currency.BTC, ToCoin: currency.USDT})
	require.ErrorIs(t, err, order.ErrAmountMustBeSet)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossExchangeConvertQuote(t.Context(), &CrossExchangeConvertQuoteRequest{
		ExchangeType: "BINANCE",
		FromCoin:     currency.BTC,
		ToCoin:       currency.USDT,
		FromAmount:   1,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, result.QuoteID)
}

func TestExecuteCrossExchangeConvertOrder(t *testing.T) {
	t.Parallel()
	_, err := e.ExecuteCrossExchangeConvertOrder(t.Context(), "")
	require.ErrorIs(t, err, errCrossExchangeQuoteIDRequired)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.ExecuteCrossExchangeConvertOrder(t.Context(), "CTH46487058372")
	require.NoError(t, err)
	assert.NotEmpty(t, result.OrderID)
}

func TestGetCrossExchangeAccountAssets(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossExchangeAccountAssets(t.Context(), "")
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestUpdateCrossExchangeAccount(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateCrossExchangeAccount(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.UpdateCrossExchangeAccount(t.Context(), &CrossExchangeAccountUpdateRequest{
		PositionMode: "SINGLE",
		AccountMode:  "CROSS_EXCHANGE",
		ExchangeType: "BINANCE",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, result.ExchangeType)
}

func TestGetCrossExchangeContractLeverage(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossExchangeContractLeverage(t.Context(), nil)
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestSetCrossExchangeContractLeverage(t *testing.T) {
	t.Parallel()
	_, err := e.SetCrossExchangeContractLeverage(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	_, err = e.SetCrossExchangeContractLeverage(t.Context(), &CrossExchangeLeverageRequest{})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	_, err = e.SetCrossExchangeContractLeverage(t.Context(), &CrossExchangeLeverageRequest{Symbol: "BINANCE_FUTURE_BTC_USDT"})
	require.ErrorIs(t, err, errCrossExchangeLeverageRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.SetCrossExchangeContractLeverage(t.Context(), &CrossExchangeLeverageRequest{
		Symbol:   "BINANCE_FUTURE_BTC_USDT",
		Leverage: 10,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, result.Symbol)
}

func TestGetCrossExchangeMarginLeverage(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossExchangeMarginLeverage(t.Context(), nil)
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestSetCrossExchangeMarginLeverage(t *testing.T) {
	t.Parallel()
	_, err := e.SetCrossExchangeMarginLeverage(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	_, err = e.SetCrossExchangeMarginLeverage(t.Context(), &CrossExchangeLeverageRequest{})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	_, err = e.SetCrossExchangeMarginLeverage(t.Context(), &CrossExchangeLeverageRequest{Symbol: "BINANCE_MARGIN_BTC_USDT"})
	require.ErrorIs(t, err, errCrossExchangeLeverageRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.SetCrossExchangeMarginLeverage(t.Context(), &CrossExchangeLeverageRequest{
		Symbol:   "BINANCE_MARGIN_BTC_USDT",
		Leverage: 5,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, result.Symbol)
}

func TestCloseCrossExchangePosition(t *testing.T) {
	t.Parallel()
	_, err := e.CloseCrossExchangePosition(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	_, err = e.CloseCrossExchangePosition(t.Context(), &CrossExchangeClosePositionRequest{})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CloseCrossExchangePosition(t.Context(), &CrossExchangeClosePositionRequest{
		Symbol:       "BINANCE_FUTURE_BTC_USDT",
		PositionSide: "LONG",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, result.OrderID)
}

func TestGetCrossExchangeInterestRates(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossExchangeInterestRates(t.Context(), currency.EMPTYCODE, "")
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestGetCrossExchangeUserFeeRates(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossExchangeUserFeeRates(t.Context())
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestGetCrossExchangeContractPositions(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossExchangeContractPositions(t.Context(), "", "")
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestGetCrossExchangeMarginPositions(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossExchangeMarginPositions(t.Context(), "", "")
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestGetCrossExchangeADLRank(t *testing.T) {
	t.Parallel()
	_, err := e.GetCrossExchangeADLRank(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossExchangeADLRank(t.Context(), "BINANCE_FUTURE_ADA_USDT")
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestGetCrossExchangeOpenOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossExchangeOpenOrders(t.Context(), nil)
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestGetCrossExchangeOrderHistory(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossExchangeOrderHistory(t.Context(), nil)
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestGetCrossExchangeContractPositionHistory(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossExchangeContractPositionHistory(t.Context(), nil)
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestGetCrossExchangeMarginPositionHistory(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossExchangeMarginPositionHistory(t.Context(), nil)
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestGetCrossExchangeMarginInterestHistory(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossExchangeMarginInterestHistory(t.Context(), nil)
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestGetCrossExchangeTradeHistory(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossExchangeTradeHistory(t.Context(), nil)
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestGetCrossExchangeAccountBook(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossExchangeAccountBook(t.Context(), nil)
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestGetCrossExchangeCoinDiscountRates(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossExchangeCoinDiscountRates(t.Context(), currency.ETH, "")
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}
