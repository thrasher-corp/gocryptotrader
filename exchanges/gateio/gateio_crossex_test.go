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

func TestGetCrossexSymbols(t *testing.T) {
	t.Parallel()
	result, err := e.GetCrossexSymbols(t.Context(), nil)
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestGetCrossexRiskLimits(t *testing.T) {
	t.Parallel()
	result, err := e.GetCrossexRiskLimits(t.Context(), nil)
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestGetCrossexTransferCoins(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossexTransferCoins(t.Context(), currency.EMPTYCODE)
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestGetCrossexTransferHistory(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossexTransferHistory(t.Context(), nil)
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestCrossexFundTransfer(t *testing.T) {
	t.Parallel()
	_, err := e.CrossexFundTransfer(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	_, err = e.CrossexFundTransfer(t.Context(), &CrossexTransferRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.CrossexFundTransfer(t.Context(), &CrossexTransferRequest{Coin: currency.BTC})
	require.ErrorIs(t, err, order.ErrAmountMustBeSet)

	_, err = e.CrossexFundTransfer(t.Context(), &CrossexTransferRequest{Coin: currency.BTC, Amount: 0.001})
	require.ErrorIs(t, err, errCrossexFromAccountRequired)

	_, err = e.CrossexFundTransfer(t.Context(), &CrossexTransferRequest{Coin: currency.BTC, Amount: 0.001, From: "spot"})
	require.ErrorIs(t, err, errCrossexToAccountRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CrossexFundTransfer(t.Context(), &CrossexTransferRequest{
		Coin: currency.BTC, Amount: 0.001, From: "spot", To: "crossex",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, result.TxID)
}

func TestCreateCrossexOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CreateCrossexOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	_, err = e.CreateCrossexOrder(t.Context(), &CrossexOrderCreateRequest{})
	require.ErrorIs(t, err, errCrossexExchangeTypeRequired)

	_, err = e.CreateCrossexOrder(t.Context(), &CrossexOrderCreateRequest{ExchangeType: "BINANCE"})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	_, err = e.CreateCrossexOrder(t.Context(), &CrossexOrderCreateRequest{ExchangeType: "BINANCE", Symbol: "BINANCE_FUTURE_BTC_USDT"})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	_, err = e.CreateCrossexOrder(t.Context(), &CrossexOrderCreateRequest{ExchangeType: "BINANCE", Symbol: "BINANCE_FUTURE_BTC_USDT", Side: order.Buy})
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CreateCrossexOrder(t.Context(), &CrossexOrderCreateRequest{
		ExchangeType: "BINANCE",
		Symbol:       "BINANCE_FUTURE_BTC_USDT",
		Side:         order.Buy,
		Type:         "GTC",
		Quantity:     1,
		Price:        65000,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, result.OrderID)
}

func TestGetCrossexOrderDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetCrossexOrderDetails(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossexOrderDetails(t.Context(), "20491522002333905922")
	require.NoError(t, err)
	assert.NotEmpty(t, result.OrderID)
}

func TestModifyCrossexOrder(t *testing.T) {
	t.Parallel()
	_, err := e.ModifyCrossexOrder(t.Context(), "", &CrossexOrderUpdateRequest{})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.ModifyCrossexOrder(t.Context(), "20491522002333905922", &CrossexOrderUpdateRequest{Price: 64000})
	require.NoError(t, err)
	assert.NotEmpty(t, result.OrderID)
}

func TestCancelCrossexOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelCrossexOrder(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelCrossexOrder(t.Context(), "20491522002333905922")
	require.NoError(t, err)
	assert.NotEmpty(t, result.OrderID)
}

func TestGetCrossexConvertQuote(t *testing.T) {
	t.Parallel()
	_, err := e.GetCrossexConvertQuote(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	_, err = e.GetCrossexConvertQuote(t.Context(), &CrossexConvertQuoteRequest{})
	require.ErrorIs(t, err, errCrossexExchangeTypeRequired)

	_, err = e.GetCrossexConvertQuote(t.Context(), &CrossexConvertQuoteRequest{ExchangeType: "BINANCE"})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.GetCrossexConvertQuote(t.Context(), &CrossexConvertQuoteRequest{ExchangeType: "BINANCE", FromCoin: currency.BTC})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.GetCrossexConvertQuote(t.Context(), &CrossexConvertQuoteRequest{ExchangeType: "BINANCE", FromCoin: currency.BTC, ToCoin: currency.USDT})
	require.ErrorIs(t, err, order.ErrAmountMustBeSet)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossexConvertQuote(t.Context(), &CrossexConvertQuoteRequest{
		ExchangeType: "BINANCE",
		FromCoin:     currency.BTC,
		ToCoin:       currency.USDT,
		FromAmount:   1,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, result.QuoteID)
}

func TestExecuteCrossexConvertOrder(t *testing.T) {
	t.Parallel()
	_, err := e.ExecuteCrossexConvertOrder(t.Context(), "")
	require.ErrorIs(t, err, errCrossexQuoteIDRequired)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.ExecuteCrossexConvertOrder(t.Context(), "CTH46487058372")
	require.NoError(t, err)
	assert.NotEmpty(t, result.OrderID)
}

func TestGetCrossexAccountAssets(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossexAccountAssets(t.Context(), "")
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestUpdateCrossexAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.UpdateCrossexAccount(t.Context(), &CrossexAccountUpdateRequest{
		PositionMode: "SINGLE",
		AccountMode:  "CROSS_EXCHANGE",
		ExchangeType: "BINANCE",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, result.ExchangeType)
}

func TestGetCrossexContractLeverage(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossexContractLeverage(t.Context(), nil)
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestSetCrossexContractLeverage(t *testing.T) {
	t.Parallel()
	_, err := e.SetCrossexContractLeverage(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	_, err = e.SetCrossexContractLeverage(t.Context(), &CrossexLeverageRequest{})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	_, err = e.SetCrossexContractLeverage(t.Context(), &CrossexLeverageRequest{Symbol: "BINANCE_FUTURE_BTC_USDT"})
	require.ErrorIs(t, err, errCrossexLeverageRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.SetCrossexContractLeverage(t.Context(), &CrossexLeverageRequest{
		Symbol:   "BINANCE_FUTURE_BTC_USDT",
		Leverage: 10,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, result.Symbol)
}

func TestGetCrossexMarginLeverage(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossexMarginLeverage(t.Context(), nil)
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestSetCrossexMarginLeverage(t *testing.T) {
	t.Parallel()
	_, err := e.SetCrossexMarginLeverage(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	_, err = e.SetCrossexMarginLeverage(t.Context(), &CrossexLeverageRequest{})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	_, err = e.SetCrossexMarginLeverage(t.Context(), &CrossexLeverageRequest{Symbol: "BINANCE_MARGIN_BTC_USDT"})
	require.ErrorIs(t, err, errCrossexLeverageRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.SetCrossexMarginLeverage(t.Context(), &CrossexLeverageRequest{
		Symbol:   "BINANCE_MARGIN_BTC_USDT",
		Leverage: 5,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, result.Symbol)
}

func TestCloseCrossexPosition(t *testing.T) {
	t.Parallel()
	_, err := e.CloseCrossexPosition(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	_, err = e.CloseCrossexPosition(t.Context(), &CrossexClosePositionRequest{})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CloseCrossexPosition(t.Context(), &CrossexClosePositionRequest{
		Symbol:       "BINANCE_FUTURE_BTC_USDT",
		PositionSide: "LONG",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, result.OrderID)
}

func TestGetCrossexInterestRates(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossexInterestRates(t.Context(), currency.EMPTYCODE, "")
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestGetCrossexUserFeeRates(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossexUserFeeRates(t.Context())
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestGetCrossexContractPositions(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossexContractPositions(t.Context(), "", "")
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestGetCrossexMarginPositions(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossexMarginPositions(t.Context(), "", "")
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestGetCrossexADLRank(t *testing.T) {
	t.Parallel()
	_, err := e.GetCrossexADLRank(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossexADLRank(t.Context(), "BINANCE_FUTURE_ADA_USDT")
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestGetCrossexOpenOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossexOpenOrders(t.Context(), nil)
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestGetCrossexOrderHistory(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossexOrderHistory(t.Context(), nil)
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestGetCrossexContractPositionHistory(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossexContractPositionHistory(t.Context(), nil)
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestGetCrossexMarginPositionHistory(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossexMarginPositionHistory(t.Context(), nil)
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestGetCrossexMarginInterestHistory(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossexMarginInterestHistory(t.Context(), nil)
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestGetCrossexTradeHistory(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossexTradeHistory(t.Context(), nil)
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestGetCrossexAccountBook(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossexAccountBook(t.Context(), nil)
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestGetCrossexCoinDiscountRates(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossexCoinDiscountRates(t.Context(), "", "")
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}
