package gateio

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func newCrossExchangeSymbol(exchangeName string, assetType asset.Item, pair currency.Pair) CrossExchangeSymbolIdentifier {
	return CrossExchangeSymbolIdentifier{Exchange: exchangeName, Asset: assetType, Pair: pair}
}

func TestCrossExchangeSymbolIdentifier(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		symbol   CrossExchangeSymbolIdentifier
		expected string
	}{
		{
			name:     "spot",
			symbol:   newCrossExchangeSymbol("binance", asset.Spot, currency.NewPairWithDelimiter("btc", "usdt", currency.DashDelimiter)),
			expected: "BINANCE_SPOT_BTC_USDT",
		},
		{
			name:     "margin",
			symbol:   newCrossExchangeSymbol("GATE", asset.Margin, currency.NewPair(currency.ADA, currency.USDT)),
			expected: "GATE_MARGIN_ADA_USDT",
		},
		{
			name:     "future",
			symbol:   newCrossExchangeSymbol("OKX", asset.Futures, currency.NewPair(currency.ETH, currency.USDT)),
			expected: "OKX_FUTURE_ETH_USDT",
		},
		{
			name:     "venue listed after this was written",
			symbol:   newCrossExchangeSymbol("newvenue", asset.Spot, currency.NewBTCUSDT()),
			expected: "NEWVENUE_SPOT_BTC_USDT",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.NoError(t, tc.symbol.Validate(), "Validate must not error")
			formatted, err := tc.symbol.Format()
			require.NoError(t, err, "Format must not error")
			assert.Equal(t, tc.expected, formatted, "formatted symbol should match")
			assert.Equal(t, tc.expected, tc.symbol.String(), "String should match")
			encoded, err := json.Marshal(tc.symbol)
			require.NoError(t, err, "Marshal must not error")
			assert.Equal(t, `"`+tc.expected+`"`, string(encoded), "JSON symbol should match")
		})
	}

	for _, tc := range []struct {
		name        string
		symbol      CrossExchangeSymbolIdentifier
		rendered    string
		expectedErr error
	}{
		{name: "missing exchange", symbol: CrossExchangeSymbolIdentifier{Asset: asset.Spot, Pair: currency.NewBTCUSDT()}, rendered: "_SPOT_BTC_USDT", expectedErr: errCrossExchangeExchangeTypeRequired},
		{name: "missing pair", symbol: CrossExchangeSymbolIdentifier{Exchange: "BINANCE", Asset: asset.Spot}, rendered: "BINANCE_SPOT__", expectedErr: currency.ErrCurrencyPairEmpty},
		{name: "unsupported asset", symbol: newCrossExchangeSymbol("BINANCE", asset.Options, currency.NewBTCUSDT()), rendered: "BINANCE_OPTIONS_BTC_USDT", expectedErr: asset.ErrNotSupported},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, tc.symbol.Validate(), tc.expectedErr, "Validate must return the expected error")
			_, err := tc.symbol.Format()
			require.ErrorIs(t, err, tc.expectedErr, "Format must return the expected error")
			_, err = json.Marshal(tc.symbol)
			require.ErrorIs(t, err, tc.expectedErr, "Marshal must return the expected error")
			assert.Equal(t, tc.rendered, tc.symbol.String(), "String should render the fields that are set rather than hide them")
		})
	}
}

func TestCrossExchangeSymbolIdentifierIsEmpty(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		symbol   CrossExchangeSymbolIdentifier
		expected bool
	}{
		{name: "zero value", symbol: CrossExchangeSymbolIdentifier{}, expected: true},
		{name: "whitespace exchange", symbol: CrossExchangeSymbolIdentifier{Exchange: "   "}, expected: true},
		{name: "exchange only", symbol: CrossExchangeSymbolIdentifier{Exchange: "BINANCE"}},
		{name: "asset only", symbol: CrossExchangeSymbolIdentifier{Asset: asset.Spot}},
		{name: "pair only", symbol: CrossExchangeSymbolIdentifier{Pair: currency.NewBTCUSDT()}},
		// A half-populated pair must count as present so it reaches Format and is rejected there, rather than being skipped as absent.
		{name: "pair base only", symbol: CrossExchangeSymbolIdentifier{Pair: currency.Pair{Base: currency.BTC}}},
		{name: "pair quote only", symbol: CrossExchangeSymbolIdentifier{Pair: currency.Pair{Quote: currency.USDT}}},
		{name: "fully populated", symbol: newCrossExchangeSymbol("BINANCE", asset.Spot, currency.NewBTCUSDT())},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, tc.symbol.IsEmpty(), "IsEmpty should return the expected result")
		})
	}
}

func TestFormatCrossExchangeSymbols(t *testing.T) {
	t.Parallel()
	futureSymbol := newCrossExchangeSymbol("BINANCE", asset.Futures, currency.NewBTCUSDT())

	formatted, err := formatCrossExchangeSymbols(nil)
	require.NoError(t, err, "formatCrossExchangeSymbols must not error without symbols")
	assert.Empty(t, formatted, "formatted symbols should be empty")

	_, err = formatCrossExchangeSymbols([]CrossExchangeSymbolIdentifier{futureSymbol, {}})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "formatCrossExchangeSymbols must reject an empty symbol")
	assert.ErrorContains(t, err, "index 1", "error should identify the offending symbol")

	_, err = formatCrossExchangeSymbols([]CrossExchangeSymbolIdentifier{futureSymbol, {Exchange: "BINANCE", Asset: asset.Spot}})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "formatCrossExchangeSymbols must reject a symbol it cannot format")
	assert.ErrorContains(t, err, "index 1", "error should identify the offending symbol")

	formatted, err = formatCrossExchangeSymbols([]CrossExchangeSymbolIdentifier{futureSymbol, newCrossExchangeSymbol("okx", asset.Spot, currency.NewPair(currency.ETH, currency.USDT))})
	require.NoError(t, err, "formatCrossExchangeSymbols must not error")
	assert.Equal(t, "BINANCE_FUTURE_BTC_USDT,OKX_SPOT_ETH_USDT", formatted, "formatted symbols should match")
}

func TestGetCrossExchangeSymbols(t *testing.T) {
	t.Parallel()
	_, err := e.GetCrossExchangeSymbols(t.Context(), []CrossExchangeSymbolIdentifier{{Exchange: "BINANCE", Asset: asset.Spot}})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetCrossExchangeSymbols must reject a symbol it cannot format")

	result, err := e.GetCrossExchangeSymbols(t.Context(), []CrossExchangeSymbolIdentifier{
		newCrossExchangeSymbol("BINANCE", asset.Futures, currency.NewBTCUSDT()),
	})
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestGetCrossExchangeRiskLimits(t *testing.T) {
	t.Parallel()
	_, err := e.GetCrossExchangeRiskLimits(t.Context(), nil)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	_, err = e.GetCrossExchangeRiskLimits(t.Context(), []CrossExchangeSymbolIdentifier{
		newCrossExchangeSymbol("BINANCE", asset.Futures, currency.NewBTCUSDT()),
		{},
	})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetCrossExchangeRiskLimits(t.Context(), []CrossExchangeSymbolIdentifier{
		newCrossExchangeSymbol("BINANCE", asset.Futures, currency.NewBTCUSDT()),
		newCrossExchangeSymbol("BINANCE", asset.Futures, currency.NewPair(currency.ETH, currency.USDT)),
	})
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestGetCrossExchangeTransferCoins(t *testing.T) {
	t.Parallel()
	result, err := e.GetCrossExchangeTransferCoins(t.Context(), currency.USDT)
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestGetCrossExchangeTransferHistory(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossExchangeTransferHistory(t.Context(), &GetCrossExchangeTransferHistoryRequest{
		Coin:       currency.USDT,
		OrderID:    "transfer-001",
		From:       1744103854,
		To:         1744190254,
		PageNumber: 1,
		Limit:      10,
	})
	require.NoError(t, err, "GetCrossExchangeTransferHistory must not error")
	if mockTests {
		assert.NotEmpty(t, result, "GetCrossExchangeTransferHistory should return transfer records")
	}
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

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
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

	_, err = e.CreateCrossExchangeOrder(t.Context(), &CrossExchangeOrderCreateRequest{
		Symbol: newCrossExchangeSymbol("BINANCE", asset.Futures, currency.NewBTCUSDT()),
	})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	_, err = e.CreateCrossExchangeOrder(t.Context(), &CrossExchangeOrderCreateRequest{
		Symbol: newCrossExchangeSymbol("BINANCE", asset.Futures, currency.NewBTCUSDT()),
		Side:   order.Buy,
	})
	require.ErrorIs(t, err, order.ErrAmountMustBeSet, "CreateCrossExchangeOrder must reject an order without a quantity")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	validRequest := &CrossExchangeOrderCreateRequest{
		Symbol:       newCrossExchangeSymbol("BINANCE", asset.Futures, currency.NewBTCUSDT()),
		Side:         order.Buy,
		OrderType:    order.Limit,
		Quantity:     1,
		Price:        65000,
		TimeInForce:  order.GoodTillCancel,
		ReduceOnly:   true,
		PositionSide: order.Short,
	}
	result, err := e.CreateCrossExchangeOrder(t.Context(), validRequest)
	require.NoError(t, err)
	if mockTests {
		assert.NotEmpty(t, result.OrderID)
	}
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

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.ModifyCrossExchangeOrder(t.Context(), "20491522002333905922", &CrossExchangeOrderUpdateRequest{Price: 64000})
	require.NoError(t, err)
	assert.NotEmpty(t, result.OrderID)
}

func TestCancelCrossExchangeOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelCrossExchangeOrder(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelCrossExchangeOrder(t.Context(), "20491522002333905922")
	require.NoError(t, err)
	assert.NotEmpty(t, result.OrderID)
}

func TestCancelBatchCrossExchangeOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelBatchCrossExchangeOrders(t.Context(), nil)
	require.ErrorIs(t, err, errNoValidParameterPassed, "CancelBatchCrossExchangeOrders must reject an empty batch")

	_, err = e.CancelBatchCrossExchangeOrders(t.Context(), []*CrossExchangeBatchCancelOrderRequest{nil})
	require.ErrorIs(t, err, common.ErrNilPointer, "CancelBatchCrossExchangeOrders must reject a nil request item")

	_, err = e.CancelBatchCrossExchangeOrders(t.Context(), []*CrossExchangeBatchCancelOrderRequest{{}})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelBatchCrossExchangeOrders must require an order ID or text ID for every item")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelBatchCrossExchangeOrders(t.Context(), []*CrossExchangeBatchCancelOrderRequest{
		{OrderID: "123456"},
		{Text: "crossex-test-1"},
	})
	require.NoError(t, err, "CancelBatchCrossExchangeOrders must not error")
	if mockTests {
		require.Len(t, result, 2, "CancelBatchCrossExchangeOrders must return a result for every mocked item")
		assert.Equal(t, "123456", result[0].OrderID, "accepted result order ID should match")
		assert.True(t, result[0].Accepted.Bool(), "first cancellation should be accepted")
		assert.Empty(t, result[0].Label, "accepted cancellation label should be empty")
		assert.Empty(t, result[0].Message, "accepted cancellation message should be empty")
		assert.Equal(t, "crossex-test-1", result[1].Text, "rejected result text ID should match")
		assert.False(t, result[1].Accepted.Bool(), "second cancellation should be rejected")
		assert.Equal(t, "TRADE_ORDER_NOT_FOUND_ERROR", result[1].Label, "rejected cancellation label should match")
		assert.Equal(t, "The order was not found", result[1].Message, "rejected cancellation message should match")
	}
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
	require.ErrorIs(t, err, errQuoteIDRequired)

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
	for _, exchangeType := range []string{"", "BINANCE"} {
		result, err := e.GetCrossExchangeAccountAssets(t.Context(), exchangeType)
		require.NoError(t, err, "GetCrossExchangeAccountAssets must not error")
		require.NotNil(t, result, "GetCrossExchangeAccountAssets must return account data")
		if mockTests {
			require.NotEmpty(t, result.Assets, "GetCrossExchangeAccountAssets must return account assets")
			assert.Equal(t, currency.USDT, result.Assets[0].Coin, "asset coin should match")
			assert.Equal(t, types.Number(100), result.Assets[0].UnrealizedPNL, "asset unrealized PNL should match")
			if exchangeType != "" {
				assert.Equal(t, exchangeType, result.Assets[0].ExchangeType, "filtered asset exchange type should match")
			}
		}
	}
}

func TestUpdateCrossExchangeAccount(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateCrossExchangeAccount(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
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
	_, err := e.GetCrossExchangeContractLeverage(t.Context(), []CrossExchangeSymbolIdentifier{{Exchange: "BINANCE", Asset: asset.Futures}})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetCrossExchangeContractLeverage must reject a symbol it cannot format")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		symbols []CrossExchangeSymbolIdentifier
	}{
		{},
		{symbols: []CrossExchangeSymbolIdentifier{newCrossExchangeSymbol("BINANCE", asset.Futures, currency.NewBTCUSDT())}},
	} {
		result, err := e.GetCrossExchangeContractLeverage(t.Context(), tc.symbols)
		require.NoError(t, err, "GetCrossExchangeContractLeverage must not error")
		if mockTests {
			assert.NotEmpty(t, result, "GetCrossExchangeContractLeverage should return leverage data")
		}
	}
}

func TestSetCrossExchangeContractLeverage(t *testing.T) {
	t.Parallel()
	_, err := e.SetCrossExchangeContractLeverage(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	_, err = e.SetCrossExchangeContractLeverage(t.Context(), &CrossExchangeLeverageRequest{})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	_, err = e.SetCrossExchangeContractLeverage(t.Context(), &CrossExchangeLeverageRequest{
		Symbol: newCrossExchangeSymbol("BINANCE", asset.Futures, currency.NewBTCUSDT()),
	})
	require.ErrorIs(t, err, errCrossExchangeLeverageRequired)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SetCrossExchangeContractLeverage(t.Context(), &CrossExchangeLeverageRequest{
		Symbol:   newCrossExchangeSymbol("BINANCE", asset.Futures, currency.NewBTCUSDT()),
		Leverage: 10,
	})
	require.NoError(t, err)
	if mockTests {
		assert.NotEmpty(t, result.Symbol)
	}
}

func TestGetCrossExchangeMarginLeverage(t *testing.T) {
	t.Parallel()
	_, err := e.GetCrossExchangeMarginLeverage(t.Context(), []CrossExchangeSymbolIdentifier{{Exchange: "BINANCE", Asset: asset.Margin}})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetCrossExchangeMarginLeverage must reject a symbol it cannot format")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		symbols []CrossExchangeSymbolIdentifier
	}{
		{},
		{symbols: []CrossExchangeSymbolIdentifier{newCrossExchangeSymbol("BINANCE", asset.Margin, currency.NewBTCUSDT())}},
	} {
		result, err := e.GetCrossExchangeMarginLeverage(t.Context(), tc.symbols)
		require.NoError(t, err, "GetCrossExchangeMarginLeverage must not error")
		if mockTests {
			assert.NotEmpty(t, result, "GetCrossExchangeMarginLeverage should return leverage data")
		}
	}
}

func TestSetCrossExchangeMarginLeverage(t *testing.T) {
	t.Parallel()
	_, err := e.SetCrossExchangeMarginLeverage(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	_, err = e.SetCrossExchangeMarginLeverage(t.Context(), &CrossExchangeLeverageRequest{})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	_, err = e.SetCrossExchangeMarginLeverage(t.Context(), &CrossExchangeLeverageRequest{
		Symbol: newCrossExchangeSymbol("BINANCE", asset.Margin, currency.NewBTCUSDT()),
	})
	require.ErrorIs(t, err, errCrossExchangeLeverageRequired)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SetCrossExchangeMarginLeverage(t.Context(), &CrossExchangeLeverageRequest{
		Symbol:   newCrossExchangeSymbol("BINANCE", asset.Margin, currency.NewBTCUSDT()),
		Leverage: 5,
	})
	require.NoError(t, err)
	if mockTests {
		assert.NotEmpty(t, result.Symbol)
	}
}

func TestCloseCrossExchangePosition(t *testing.T) {
	t.Parallel()
	_, err := e.CloseCrossExchangePosition(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	_, err = e.CloseCrossExchangePosition(t.Context(), &CrossExchangeClosePositionRequest{})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CloseCrossExchangePosition(t.Context(), &CrossExchangeClosePositionRequest{
		Symbol:       newCrossExchangeSymbol("BINANCE", asset.Futures, currency.NewBTCUSDT()),
		PositionSide: "LONG",
	})
	require.NoError(t, err)
	if mockTests {
		assert.NotEmpty(t, result.OrderID)
	}
}

func TestGetCrossExchangeInterestRates(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		coin         currency.Code
		exchangeType string
	}{
		{},
		{coin: currency.BCH, exchangeType: "GATE"},
	} {
		result, err := e.GetCrossExchangeInterestRates(t.Context(), tc.coin, tc.exchangeType)
		require.NoError(t, err, "GetCrossExchangeInterestRates must not error")
		if mockTests {
			assert.NotEmpty(t, result, "GetCrossExchangeInterestRates should return interest rates")
		}
	}
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
	_, err := e.GetCrossExchangeContractPositions(t.Context(), CrossExchangeSymbolIdentifier{Exchange: "BINANCE", Asset: asset.Futures}, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetCrossExchangeContractPositions must reject a symbol it cannot format")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		symbol       CrossExchangeSymbolIdentifier
		exchangeType string
	}{
		{},
		{symbol: newCrossExchangeSymbol("BINANCE", asset.Futures, currency.NewBTCUSDT()), exchangeType: "BINANCE"},
	} {
		result, err := e.GetCrossExchangeContractPositions(t.Context(), tc.symbol, tc.exchangeType)
		require.NoError(t, err, "GetCrossExchangeContractPositions must not error")
		if mockTests {
			assert.NotEmpty(t, result, "GetCrossExchangeContractPositions should return positions")
		}
	}
}

func TestGetCrossExchangeMarginPositions(t *testing.T) {
	t.Parallel()
	_, err := e.GetCrossExchangeMarginPositions(t.Context(), CrossExchangeSymbolIdentifier{Exchange: "BINANCE", Asset: asset.Margin}, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetCrossExchangeMarginPositions must reject a symbol it cannot format")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		symbol       CrossExchangeSymbolIdentifier
		exchangeType string
	}{
		{},
		{symbol: newCrossExchangeSymbol("BINANCE", asset.Margin, currency.NewBTCUSDT()), exchangeType: "BINANCE"},
	} {
		result, err := e.GetCrossExchangeMarginPositions(t.Context(), tc.symbol, tc.exchangeType)
		require.NoError(t, err, "GetCrossExchangeMarginPositions must not error")
		if mockTests {
			assert.NotEmpty(t, result, "GetCrossExchangeMarginPositions should return positions")
		}
	}
}

func TestGetCrossExchangeADLRank(t *testing.T) {
	t.Parallel()
	_, err := e.GetCrossExchangeADLRank(t.Context(), CrossExchangeSymbolIdentifier{})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	_, err = e.GetCrossExchangeADLRank(t.Context(), CrossExchangeSymbolIdentifier{Exchange: "BINANCE", Asset: asset.Futures})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetCrossExchangeADLRank must reject a symbol it cannot format")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossExchangeADLRank(t.Context(), newCrossExchangeSymbol("BINANCE", asset.Futures, currency.NewPair(currency.ADA, currency.USDT)))
	require.NoError(t, err)
	if mockTests {
		assert.NotEmpty(t, result)
	}
}

func TestGetCrossExchangeOpenOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetCrossExchangeOpenOrders(t.Context(), &GetCrossExchangeOpenOrdersRequest{Symbol: CrossExchangeSymbolIdentifier{Exchange: "BINANCE", Asset: asset.Futures}})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetCrossExchangeOpenOrders must reject a symbol it cannot format")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossExchangeOpenOrders(t.Context(), &GetCrossExchangeOpenOrdersRequest{
		Symbol:       newCrossExchangeSymbol("BINANCE", asset.Futures, currency.NewBTCUSDT()),
		ExchangeType: "BINANCE",
		BusinessType: "FUTURE",
	})
	require.NoError(t, err, "GetCrossExchangeOpenOrders must not error")
	if mockTests {
		assert.NotEmpty(t, result, "GetCrossExchangeOpenOrders should return orders")
	}
}

func TestGetCrossExchangeOrderHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetCrossExchangeOrderHistory(t.Context(), &GetCrossExchangeOrderHistoryRequest{Symbol: CrossExchangeSymbolIdentifier{Exchange: "BINANCE", Asset: asset.Futures}})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetCrossExchangeOrderHistory must reject a symbol it cannot format")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossExchangeOrderHistory(t.Context(), &GetCrossExchangeOrderHistoryRequest{
		Page:      1,
		Limit:     10,
		Symbol:    newCrossExchangeSymbol("BINANCE", asset.Futures, currency.NewBTCUSDT()),
		From:      1744103854,
		To:        1744190254,
		Attribute: "COMMON",
	})
	require.NoError(t, err, "GetCrossExchangeOrderHistory must not error")
	if mockTests {
		assert.NotEmpty(t, result, "GetCrossExchangeOrderHistory should return orders")
	}
}

func TestGetCrossExchangeContractPositionHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetCrossExchangeContractPositionHistory(t.Context(), &GetCrossExchangePositionHistoryRequest{Symbol: CrossExchangeSymbolIdentifier{Exchange: "BINANCE", Asset: asset.Futures}})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetCrossExchangeContractPositionHistory must reject a symbol it cannot format")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossExchangeContractPositionHistory(t.Context(), &GetCrossExchangePositionHistoryRequest{
		Page:   1,
		Limit:  10,
		Symbol: newCrossExchangeSymbol("BINANCE", asset.Futures, currency.NewBTCUSDT()),
		From:   1744103854,
		To:     1744190254,
	})
	require.NoError(t, err, "GetCrossExchangeContractPositionHistory must not error")
	if mockTests {
		assert.NotEmpty(t, result, "GetCrossExchangeContractPositionHistory should return positions")
	}
}

func TestGetCrossExchangeMarginPositionHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetCrossExchangeMarginPositionHistory(t.Context(), &GetCrossExchangePositionHistoryRequest{Symbol: CrossExchangeSymbolIdentifier{Exchange: "BINANCE", Asset: asset.Margin}})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetCrossExchangeMarginPositionHistory must reject a symbol it cannot format")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossExchangeMarginPositionHistory(t.Context(), &GetCrossExchangePositionHistoryRequest{
		Page:   1,
		Limit:  10,
		Symbol: newCrossExchangeSymbol("BINANCE", asset.Margin, currency.NewBTCUSDT()),
		From:   1744103854,
		To:     1744190254,
	})
	require.NoError(t, err, "GetCrossExchangeMarginPositionHistory must not error")
	if mockTests {
		assert.NotEmpty(t, result, "GetCrossExchangeMarginPositionHistory should return positions")
	}
}

func TestGetCrossExchangeMarginInterestHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetCrossExchangeMarginInterestHistory(t.Context(), &GetCrossExchangeMarginInterestHistoryRequest{Symbol: CrossExchangeSymbolIdentifier{Exchange: "BINANCE", Asset: asset.Margin}})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetCrossExchangeMarginInterestHistory must reject a symbol it cannot format")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossExchangeMarginInterestHistory(t.Context(), &GetCrossExchangeMarginInterestHistoryRequest{
		Symbol:       newCrossExchangeSymbol("BINANCE", asset.Margin, currency.NewBTCUSDT()),
		From:         1744103854,
		To:           1744190254,
		Page:         1,
		Limit:        10,
		ExchangeType: "BINANCE",
	})
	require.NoError(t, err, "GetCrossExchangeMarginInterestHistory must not error")
	if mockTests {
		assert.NotEmpty(t, result, "GetCrossExchangeMarginInterestHistory should return interest records")
	}
}

func TestGetCrossExchangeTradeHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetCrossExchangeTradeHistory(t.Context(), &GetCrossExchangeTradeHistoryRequest{Symbol: CrossExchangeSymbolIdentifier{Exchange: "BINANCE", Asset: asset.Futures}})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetCrossExchangeTradeHistory must reject a symbol it cannot format")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossExchangeTradeHistory(t.Context(), &GetCrossExchangeTradeHistoryRequest{
		Page:   1,
		Limit:  10,
		Symbol: newCrossExchangeSymbol("BINANCE", asset.Futures, currency.NewBTCUSDT()),
		From:   1744103854,
		To:     1744190254,
	})
	require.NoError(t, err, "GetCrossExchangeTradeHistory must not error")
	if mockTests {
		assert.NotEmpty(t, result, "GetCrossExchangeTradeHistory should return trades")
	}
}

func TestGetCrossExchangeAccountBook(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossExchangeAccountBook(t.Context(), &GetCrossExchangeAccountBookRequest{
		Page:          1,
		Limit:         10,
		Coin:          currency.USDT,
		StatementType: "TRADE_FEE",
		From:          1744103854,
		To:            1744190254,
	})
	require.NoError(t, err, "GetCrossExchangeAccountBook must not error")
	if mockTests {
		assert.NotEmpty(t, result, "GetCrossExchangeAccountBook should return account book records")
	}
}

func TestGetCrossExchangeCoinDiscountRates(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, exchangeType := range []string{"", "BINANCE"} {
		result, err := e.GetCrossExchangeCoinDiscountRates(t.Context(), currency.ETH, exchangeType)
		require.NoError(t, err, "GetCrossExchangeCoinDiscountRates must not error")
		if mockTests {
			assert.NotEmpty(t, result, "GetCrossExchangeCoinDiscountRates should return discount rates")
		}
	}
}

func TestCrossExchangeOrderCreateRequestMarshalJSON(t *testing.T) {
	t.Parallel()
	symbol := newCrossExchangeSymbol("BINANCE", asset.Futures, currency.NewBTCUSDT())
	for _, tc := range []struct {
		name        string
		req         *CrossExchangeOrderCreateRequest
		expected    string
		contains    []string
		notContains []string
		expectedErr error
	}{
		{
			// Pins the exact bytes sent to Gate; the crossex/orders fixture records this same payload.
			name: "full payload",
			req: &CrossExchangeOrderCreateRequest{
				Symbol:       symbol,
				Side:         order.Buy,
				OrderType:    order.Limit,
				TimeInForce:  order.GoodTillCancel,
				Quantity:     1,
				Price:        65000,
				ReduceOnly:   true,
				PositionSide: order.Short,
			},
			expected: `{"type":"LIMIT","time_in_force":"GTC","symbol":"BINANCE_FUTURE_BTC_USDT","side":"BUY","qty":"1","price":"65000","reduce_only":"true","position_side":"SHORT"}`,
		},
		{
			name:     "limit GTC",
			req:      &CrossExchangeOrderCreateRequest{Symbol: symbol, Side: order.Buy, OrderType: order.Limit, TimeInForce: order.GoodTillCancel},
			contains: []string{`"symbol":"BINANCE_FUTURE_BTC_USDT"`, `"type":"LIMIT"`, `"time_in_force":"GTC"`, `"side":"BUY"`},
		},
		{
			name:     "post only maps to POC",
			req:      &CrossExchangeOrderCreateRequest{Symbol: symbol, TimeInForce: order.PostOnly},
			contains: []string{`"time_in_force":"POC"`},
		},
		{
			name:     "position side",
			req:      &CrossExchangeOrderCreateRequest{Symbol: symbol, PositionSide: order.Long},
			contains: []string{`"position_side":"LONG"`},
		},
		{
			name:     "reduce only",
			req:      &CrossExchangeOrderCreateRequest{Symbol: symbol, ReduceOnly: true},
			contains: []string{`"reduce_only":"true"`},
		},
		{
			name:        "reduce only false is omitted",
			req:         &CrossExchangeOrderCreateRequest{Symbol: symbol, ReduceOnly: false},
			notContains: []string{`"reduce_only"`},
		},
		{
			name:        "unset optional values are omitted",
			req:         &CrossExchangeOrderCreateRequest{Symbol: symbol},
			notContains: []string{`"type"`, `"time_in_force"`, `"position_side"`, `"reduce_only"`},
		},
		{
			name:        "unformattable symbol is surfaced",
			req:         &CrossExchangeOrderCreateRequest{Symbol: CrossExchangeSymbolIdentifier{Exchange: "BINANCE", Asset: asset.Futures}},
			expectedErr: currency.ErrCurrencyPairEmpty,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := json.Marshal(tc.req)
			if tc.expectedErr != nil {
				require.ErrorIs(t, err, tc.expectedErr, "Marshal must return the expected error")
				return
			}
			require.NoError(t, err, "Marshal must not error")
			if tc.expected != "" {
				assert.Equal(t, tc.expected, string(got), "payload should match exactly")
			}
			for _, exp := range tc.contains {
				assert.Containsf(t, string(got), exp, "payload should contain %s", exp)
			}
			for _, unexp := range tc.notContains {
				assert.NotContainsf(t, string(got), unexp, "payload should not contain %s", unexp)
			}
		})
	}
}
