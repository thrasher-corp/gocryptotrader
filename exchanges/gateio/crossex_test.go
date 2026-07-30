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
		expectedErr error
	}{
		{name: "missing exchange", symbol: CrossExchangeSymbolIdentifier{Asset: asset.Spot, Pair: currency.NewBTCUSDT()}, expectedErr: errCrossExchangeExchangeTypeRequired},
		{name: "unsupported exchange", symbol: newCrossExchangeSymbol("UNSUPPORTED", asset.Spot, currency.NewBTCUSDT()), expectedErr: errCrossExchangeExchangeTypeInvalid},
		{name: "missing pair", symbol: CrossExchangeSymbolIdentifier{Exchange: "BINANCE", Asset: asset.Spot}, expectedErr: currency.ErrCurrencyPairEmpty},
		{name: "unsupported asset", symbol: newCrossExchangeSymbol("BINANCE", asset.Options, currency.NewBTCUSDT()), expectedErr: asset.ErrNotSupported},
		{name: "Bybit margin", symbol: newCrossExchangeSymbol("BYBIT", asset.Margin, currency.NewBTCUSDT()), expectedErr: asset.ErrNotSupported},
		{name: "Kraken spot", symbol: newCrossExchangeSymbol("KRAKEN", asset.Spot, currency.NewBTCUSDT()), expectedErr: asset.ErrNotSupported},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, tc.symbol.Validate(), tc.expectedErr, "Validate must return the expected error")
			_, err := tc.symbol.Format()
			require.ErrorIs(t, err, tc.expectedErr, "Format must return the expected error")
		})
	}
}

func TestFormatCrossExchangeSymbol(t *testing.T) {
	t.Parallel()
	futureSymbol := newCrossExchangeSymbol("BINANCE", asset.Futures, currency.NewBTCUSDT())

	formatted, err := formatCrossExchangeSymbol(CrossExchangeSymbolIdentifier{}, "OKX", "SPOT")
	require.NoError(t, err, "formatCrossExchangeSymbol must allow independent filters without a symbol")
	assert.Empty(t, formatted, "formatted symbol should be empty")

	_, err = formatCrossExchangeSymbol(futureSymbol, "OKX", "FUTURE")
	require.ErrorIs(t, err, errCrossExchangeExchangeTypeMismatch, "formatCrossExchangeSymbol must reject a conflicting exchange filter")

	_, err = formatCrossExchangeSymbol(futureSymbol, "BINANCE", "SPOT")
	require.ErrorIs(t, err, errCrossExchangeBusinessTypeMismatch, "formatCrossExchangeSymbol must reject a conflicting business filter")

	_, err = formatCrossExchangeSymbol(futureSymbol, "BINANCE", "FUTURE", asset.Margin)
	require.ErrorIs(t, err, asset.ErrNotSupported, "formatCrossExchangeSymbol must reject an unsupported endpoint asset")

	formatted, err = formatCrossExchangeSymbol(futureSymbol, "binance", "future", asset.Futures)
	require.NoError(t, err, "formatCrossExchangeSymbol must accept matching case-insensitive filters")
	assert.Equal(t, "BINANCE_FUTURE_BTC_USDT", formatted, "formatted symbol should match")
}

func TestGetCrossExchangeSymbols(t *testing.T) {
	t.Parallel()
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
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, arg := range []*GetCrossExchangeOpenOrdersRequest{
		nil,
		{
			Symbol:       newCrossExchangeSymbol("BINANCE", asset.Futures, currency.NewBTCUSDT()),
			ExchangeType: "BINANCE",
			BusinessType: "FUTURE",
		},
	} {
		result, err := e.GetCrossExchangeOpenOrders(t.Context(), arg)
		require.NoError(t, err, "GetCrossExchangeOpenOrders must not error")
		if mockTests {
			assert.NotEmpty(t, result, "GetCrossExchangeOpenOrders should return orders")
		}
	}
}

func TestGetCrossExchangeOrderHistory(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, arg := range []*GetCrossExchangeOrderHistoryRequest{
		nil,
		{Symbol: newCrossExchangeSymbol("BINANCE", asset.Futures, currency.NewBTCUSDT())},
	} {
		result, err := e.GetCrossExchangeOrderHistory(t.Context(), arg)
		require.NoError(t, err, "GetCrossExchangeOrderHistory must not error")
		if mockTests {
			assert.NotEmpty(t, result, "GetCrossExchangeOrderHistory should return orders")
		}
	}
}

func TestGetCrossExchangeContractPositionHistory(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, arg := range []*GetCrossExchangePositionHistoryRequest{
		nil,
		{Symbol: newCrossExchangeSymbol("BINANCE", asset.Futures, currency.NewBTCUSDT())},
	} {
		result, err := e.GetCrossExchangeContractPositionHistory(t.Context(), arg)
		require.NoError(t, err, "GetCrossExchangeContractPositionHistory must not error")
		if mockTests {
			assert.NotEmpty(t, result, "GetCrossExchangeContractPositionHistory should return positions")
		}
	}
}

func TestGetCrossExchangeMarginPositionHistory(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, arg := range []*GetCrossExchangePositionHistoryRequest{
		nil,
		{Symbol: newCrossExchangeSymbol("BINANCE", asset.Margin, currency.NewBTCUSDT())},
	} {
		result, err := e.GetCrossExchangeMarginPositionHistory(t.Context(), arg)
		require.NoError(t, err, "GetCrossExchangeMarginPositionHistory must not error")
		if mockTests {
			assert.NotEmpty(t, result, "GetCrossExchangeMarginPositionHistory should return positions")
		}
	}
}

func TestGetCrossExchangeMarginInterestHistory(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, arg := range []*GetCrossExchangeMarginInterestHistoryRequest{
		nil,
		{
			Symbol:       newCrossExchangeSymbol("BINANCE", asset.Margin, currency.NewBTCUSDT()),
			ExchangeType: "BINANCE",
		},
	} {
		result, err := e.GetCrossExchangeMarginInterestHistory(t.Context(), arg)
		require.NoError(t, err, "GetCrossExchangeMarginInterestHistory must not error")
		if mockTests {
			assert.NotEmpty(t, result, "GetCrossExchangeMarginInterestHistory should return interest records")
		}
	}
}

func TestGetCrossExchangeTradeHistory(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, arg := range []*GetCrossExchangeTradeHistoryRequest{
		nil,
		{Symbol: newCrossExchangeSymbol("BINANCE", asset.Futures, currency.NewBTCUSDT())},
	} {
		result, err := e.GetCrossExchangeTradeHistory(t.Context(), arg)
		require.NoError(t, err, "GetCrossExchangeTradeHistory must not error")
		if mockTests {
			assert.NotEmpty(t, result, "GetCrossExchangeTradeHistory should return trades")
		}
	}
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

func TestCrossExchangeOrderCreateRequestMarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name        string
		req         *CrossExchangeOrderCreateRequest
		contains    []string
		notContains []string
	}{
		{
			name:     "limit GTC",
			req:      &CrossExchangeOrderCreateRequest{Side: order.Buy, OrderType: order.Limit, TimeInForce: order.GoodTillCancel},
			contains: []string{`"type":"LIMIT"`, `"time_in_force":"GTC"`, `"side":"BUY"`},
		},
		{
			name:     "post only maps to POC",
			req:      &CrossExchangeOrderCreateRequest{TimeInForce: order.PostOnly},
			contains: []string{`"time_in_force":"POC"`},
		},
		{
			name:     "position side",
			req:      &CrossExchangeOrderCreateRequest{PositionSide: order.Long},
			contains: []string{`"position_side":"LONG"`},
		},
		{
			name:     "reduce only",
			req:      &CrossExchangeOrderCreateRequest{ReduceOnly: true},
			contains: []string{`"reduce_only":"true"`},
		},
		{
			name:        "reduce only false is omitted",
			req:         &CrossExchangeOrderCreateRequest{ReduceOnly: false},
			notContains: []string{`"reduce_only"`},
		},
		{
			name:        "unset optional values are omitted",
			req:         &CrossExchangeOrderCreateRequest{},
			notContains: []string{`"type"`, `"time_in_force"`, `"position_side"`, `"reduce_only"`},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := json.Marshal(tc.req)
			require.NoError(t, err, "Marshal must not error")
			for _, exp := range tc.contains {
				assert.Containsf(t, string(got), exp, "payload should contain %s", exp)
			}
			for _, unexp := range tc.notContains {
				assert.NotContainsf(t, string(got), unexp, "payload should not contain %s", unexp)
			}
		})
	}
}
