package gateio

import (
	"net/url"
	"testing"
	"time"

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

const crossExchangeTestOrderID = "20491522002333905922"

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

func TestCrossExchangeSymbolIdentifierMarshalJSON(t *testing.T) {
	t.Parallel()
	got, err := newCrossExchangeSymbol("okx", asset.Futures, currency.NewBTCUSDT()).MarshalJSON()
	require.NoError(t, err, "MarshalJSON must not error")
	assert.Equal(t, `"OKX_FUTURE_BTC_USDT"`, string(got), "MarshalJSON should render the symbol as a JSON string")

	_, err = CrossExchangeSymbolIdentifier{Exchange: "OKX", Asset: asset.Futures}.MarshalJSON()
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "MarshalJSON should reject a symbol it cannot render")
}

func TestCrossExchangeSymbolIdentifierBusinessType(t *testing.T) {
	t.Parallel()
	for a, exp := range map[asset.Item]string{
		asset.Spot:    "SPOT",
		asset.Margin:  "MARGIN",
		asset.Futures: "FUTURE",
	} {
		assert.Equalf(t, exp, CrossExchangeSymbolIdentifier{Asset: a}.businessType(), "businessType should map %s", a)
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

func TestCrossExchangeHistoryParams(t *testing.T) {
	t.Parallel()
	_, err := crossExchangeHistoryParams(CrossExchangeSymbolIdentifier{Exchange: "BINANCE", Asset: asset.Futures}, time.Time{}, time.Time{}, 0, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "crossExchangeHistoryParams must reject a symbol it cannot format")
	startTime, endTime := time.UnixMilli(1744103854944), time.UnixMilli(1744190254944)
	_, err = crossExchangeHistoryParams(CrossExchangeSymbolIdentifier{}, endTime, startTime, 0, 0)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "crossExchangeHistoryParams must reject a start after the end")

	params, err := crossExchangeHistoryParams(CrossExchangeSymbolIdentifier{}, time.Time{}, time.Time{}, 0, 0)
	require.NoError(t, err, "crossExchangeHistoryParams must not error without filters")
	assert.Empty(t, params, "crossExchangeHistoryParams should send no filters when none are set")

	params, err = crossExchangeHistoryParams(newCrossExchangeSymbol("BINANCE", asset.Futures, currency.NewBTCUSDT()), startTime, endTime, 2, 50)
	require.NoError(t, err, "crossExchangeHistoryParams must not error")
	exp := url.Values{
		"symbol": {"BINANCE_FUTURE_BTC_USDT"},
		"from":   {"1744103854944"},
		"to":     {"1744190254944"},
		"page":   {"2"},
		"limit":  {"50"},
	}
	assert.Equal(t, exp, params, "crossExchangeHistoryParams should send the time range in milliseconds")
}

func TestGetCrossExchangeSymbols(t *testing.T) {
	t.Parallel()
	_, err := e.GetCrossExchangeSymbols(t.Context(), []CrossExchangeSymbolIdentifier{{Exchange: "BINANCE", Asset: asset.Spot}})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetCrossExchangeSymbols must reject a symbol it cannot format")

	result, err := e.GetCrossExchangeSymbols(t.Context(), []CrossExchangeSymbolIdentifier{
		newCrossExchangeSymbol("BINANCE", asset.Futures, currency.NewBTCUSDT()),
	})
	require.NoError(t, err, "GetCrossExchangeSymbols must not error")
	if mockTests {
		exp := []*CrossExchangeSymbol{{
			Symbol:          "BINANCE_FUTURE_BTC_USDT",
			ExchangeType:    "BINANCE",
			BusinessType:    "FUTURE",
			State:           "live",
			MinSize:         0.001,
			MinNotional:     50,
			LotSize:         0.001,
			TickSize:        0.1,
			MaxNumOrders:    100,
			MaxMarketSize:   120,
			MaxLimitSize:    1000,
			LiquidationFee:  0.0125,
			DefaultLeverage: 5,
			SupportCross:    true,
		}}
		assert.Equal(t, exp, result, "GetCrossExchangeSymbols should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetCrossExchangeSymbols should return symbols")
}

func TestGetCrossExchangeRiskLimits(t *testing.T) {
	t.Parallel()
	_, err := e.GetCrossExchangeRiskLimits(t.Context(), nil)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "GetCrossExchangeRiskLimits must require a symbol")
	_, err = e.GetCrossExchangeRiskLimits(t.Context(), []CrossExchangeSymbolIdentifier{
		newCrossExchangeSymbol("BINANCE", asset.Futures, currency.NewBTCUSDT()),
		{},
	})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "GetCrossExchangeRiskLimits must reject an empty symbol")

	result, err := e.GetCrossExchangeRiskLimits(t.Context(), []CrossExchangeSymbolIdentifier{
		newCrossExchangeSymbol("BINANCE", asset.Futures, currency.NewBTCUSDT()),
		newCrossExchangeSymbol("BINANCE", asset.Futures, currency.NewPair(currency.ETH, currency.USDT)),
	})
	require.NoError(t, err, "GetCrossExchangeRiskLimits must not error")
	if !mockTests {
		assert.NotEmpty(t, result, "GetCrossExchangeRiskLimits should return risk limits")
		return
	}
	require.Len(t, result, 2, "GetCrossExchangeRiskLimits must return both recorded symbols")
	exp := &CrossExchangeRiskLimit{
		Symbol: "BINANCE_FUTURE_BTC_USDT",
		Tiers: []*CrossExchangeRiskLimitTier{
			{MaxRiskLimitValue: 3000000, LeverageMax: 20, MaintenanceRate: 0.0065, Tier: "1"},
			{MinRiskLimitValue: 3000000, MaxRiskLimitValue: 12000000, QuickCalculationAmount: 10500, LeverageMax: 19, MaintenanceRate: 0.01, Tier: "2"},
			{MinRiskLimitValue: 12000000, MaxRiskLimitValue: 70000000, QuickCalculationAmount: 130500, LeverageMax: 18, MaintenanceRate: 0.02, Tier: "3"},
			{MinRiskLimitValue: 70000000, MaxRiskLimitValue: 100000000, QuickCalculationAmount: 480500, LeverageMax: 17, MaintenanceRate: 0.025, Tier: "4"},
		},
	}
	assert.Equal(t, exp, result[0], "GetCrossExchangeRiskLimits should decode every field")
}

func TestGetCrossExchangeTransferCoins(t *testing.T) {
	t.Parallel()
	result, err := e.GetCrossExchangeTransferCoins(t.Context(), currency.USDT)
	require.NoError(t, err, "GetCrossExchangeTransferCoins must not error")
	if mockTests {
		exp := []*CrossExchangeTransferCoin{{Coin: currency.USDT, MinTransAmount: 10, EstimatedFee: 1, Precision: 6}}
		assert.Equal(t, exp, result, "GetCrossExchangeTransferCoins should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetCrossExchangeTransferCoins should return currencies")
}

func TestCrossExchangeFundTransfer(t *testing.T) {
	t.Parallel()
	_, err := e.CrossExchangeFundTransfer(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CrossExchangeFundTransfer must reject a nil request")
	_, err = e.CrossExchangeFundTransfer(t.Context(), &CrossExchangeTransferRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "CrossExchangeFundTransfer must require a currency")
	_, err = e.CrossExchangeFundTransfer(t.Context(), &CrossExchangeTransferRequest{Coin: currency.USDT})
	require.ErrorIs(t, err, order.ErrAmountMustBeSet, "CrossExchangeFundTransfer must require an amount")
	_, err = e.CrossExchangeFundTransfer(t.Context(), &CrossExchangeTransferRequest{Coin: currency.USDT, Amount: 242.45})
	require.ErrorIs(t, err, errCrossExchangeFromAccountRequired, "CrossExchangeFundTransfer must require a source account")
	_, err = e.CrossExchangeFundTransfer(t.Context(), &CrossExchangeTransferRequest{Coin: currency.USDT, Amount: 242.45, From: "SPOT"})
	require.ErrorIs(t, err, errCrossExchangeToAccountRequired, "CrossExchangeFundTransfer must require a destination account")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CrossExchangeFundTransfer(t.Context(), &CrossExchangeTransferRequest{Coin: currency.USDT, Amount: 242.45, From: "SPOT", To: "CROSSEX"})
	require.NoError(t, err, "CrossExchangeFundTransfer must not error")
	if mockTests {
		assert.Equal(t, &CrossExchangeTransferResponse{TxID: "23453", Text: "23453"}, result, "CrossExchangeFundTransfer should decode every field")
		return
	}
	assert.NotEmpty(t, result.TxID, "CrossExchangeFundTransfer should return a transfer ID")
}

func TestGetCrossExchangeTransferHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetCrossExchangeTransferHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetCrossExchangeTransferHistory must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.GetCrossExchangeTransferHistory(t.Context(), &GetCrossExchangeTransferHistoryRequest{StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetCrossExchangeTransferHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossExchangeTransferHistory(t.Context(), &GetCrossExchangeTransferHistoryRequest{
		Coin:      currency.USDT,
		OrderID:   "23453",
		StartTime: startTime,
		EndTime:   endTime,
		Page:      1,
		Limit:     10,
	})
	require.NoError(t, err, "GetCrossExchangeTransferHistory must not error")
	if mockTests {
		exp := []*CrossExchangeTransferRecord{
			{
				ID:              "33829017692939266",
				Text:            "33829017692939266",
				FromAccountType: "CROSSEX_BINANCE",
				ToAccountType:   "CROSSEX_OKX",
				Coin:            currency.BTC,
				Amount:          1.1234567,
				ActualReceive:   1.123,
				Status:          "SUCCESS",
				CreateTime:      types.Time(time.UnixMilli(1750681141933)),
				UpdateTime:      types.Time(time.UnixMilli(1750681142933)),
			},
			{
				ID:              "38083797492939266",
				Text:            "38083797492939266",
				FromAccountType: "CROSSEX",
				ToAccountType:   "SPOT",
				Coin:            currency.USDT,
				Amount:          100,
				Status:          "FAIL",
				FailReason:      "Insufficient transferAvailable",
				CreateTime:      types.Time(time.UnixMilli(1750681141933)),
				UpdateTime:      types.Time(time.UnixMilli(1750681141933)),
			},
		}
		assert.Equal(t, exp, result, "GetCrossExchangeTransferHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetCrossExchangeTransferHistory should return transfer records")
}

func TestCreateCrossExchangeOrder(t *testing.T) {
	t.Parallel()
	futureSymbol := newCrossExchangeSymbol("BINANCE", asset.Futures, currency.NewBTCUSDT())
	spotSymbol := newCrossExchangeSymbol("BINANCE", asset.Spot, currency.NewPair(currency.ADA, currency.USDT))
	_, err := e.CreateCrossExchangeOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreateCrossExchangeOrder must reject a nil request")
	_, err = e.CreateCrossExchangeOrder(t.Context(), &CrossExchangeOrderCreateRequest{})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "CreateCrossExchangeOrder must require a symbol")
	_, err = e.CreateCrossExchangeOrder(t.Context(), &CrossExchangeOrderCreateRequest{Symbol: futureSymbol})
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "CreateCrossExchangeOrder must require a side")
	_, err = e.CreateCrossExchangeOrder(t.Context(), &CrossExchangeOrderCreateRequest{Symbol: futureSymbol, Side: order.Long})
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "CreateCrossExchangeOrder must only accept a buy or sell side")
	_, err = e.CreateCrossExchangeOrder(t.Context(), &CrossExchangeOrderCreateRequest{Symbol: futureSymbol, Side: order.Buy, OrderType: order.Stop})
	require.ErrorIs(t, err, order.ErrUnsupportedOrderType, "CreateCrossExchangeOrder must only accept limit and market orders")
	_, err = e.CreateCrossExchangeOrder(t.Context(), &CrossExchangeOrderCreateRequest{Symbol: futureSymbol, Side: order.Buy})
	require.ErrorIs(t, err, order.ErrAmountMustBeSet, "CreateCrossExchangeOrder must reject an order without a quantity")
	_, err = e.CreateCrossExchangeOrder(t.Context(), &CrossExchangeOrderCreateRequest{Symbol: futureSymbol, Side: order.Buy, OrderType: order.Market, QuoteQuantity: 10})
	require.ErrorIs(t, err, order.ErrAmountMustBeSet, "CreateCrossExchangeOrder must require a quantity for a futures market buy")
	_, err = e.CreateCrossExchangeOrder(t.Context(), &CrossExchangeOrderCreateRequest{Symbol: spotSymbol, Side: order.Buy, OrderType: order.Market, Quantity: 10})
	require.ErrorIs(t, err, order.ErrAmountMustBeSet, "CreateCrossExchangeOrder must require a quote quantity for a spot market buy")
	_, err = e.CreateCrossExchangeOrder(t.Context(), &CrossExchangeOrderCreateRequest{Symbol: futureSymbol, Side: order.Buy, Quantity: 1})
	require.ErrorIs(t, err, order.ErrPriceMustBeSetIfLimitOrder, "CreateCrossExchangeOrder must require a price for an order defaulting to limit")
	_, err = e.CreateCrossExchangeOrder(t.Context(), &CrossExchangeOrderCreateRequest{Symbol: futureSymbol, Side: order.Buy, Quantity: 1, Price: 65000, TimeInForce: order.ImmediateOrCancel, RetailPriceImprovement: true})
	require.ErrorIs(t, err, errCrossExchangeTimeInForceConflict, "CreateCrossExchangeOrder must reject RPI alongside another time in force")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CreateCrossExchangeOrder(t.Context(), &CrossExchangeOrderCreateRequest{
		Symbol:       futureSymbol,
		Side:         order.Buy,
		OrderType:    order.Limit,
		Quantity:     1,
		Price:        65000,
		TimeInForce:  order.GoodTillCancel,
		ReduceOnly:   true,
		PositionSide: order.Short,
	})
	require.NoError(t, err, "CreateCrossExchangeOrder must not error")
	if !mockTests {
		assert.NotEmpty(t, result.OrderID, "CreateCrossExchangeOrder should return an order ID")
		return
	}
	assert.Equal(t, &CrossExchangeOrderActionResponse{OrderID: "123456", Text: "cross-test-1"}, result, "CreateCrossExchangeOrder should decode every field")

	result, err = e.CreateCrossExchangeOrder(t.Context(), &CrossExchangeOrderCreateRequest{Symbol: spotSymbol, Side: order.Buy, OrderType: order.Market, QuoteQuantity: 10})
	require.NoError(t, err, "CreateCrossExchangeOrder must not error for a spot market buy")
	assert.Equal(t, &CrossExchangeOrderActionResponse{OrderID: "123457", Text: "123457"}, result, "CreateCrossExchangeOrder should decode every field")
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
			name:        "retail price improvement maps to RPI",
			req:         &CrossExchangeOrderCreateRequest{Symbol: symbol, RetailPriceImprovement: true},
			contains:    []string{`"time_in_force":"RPI"`},
			notContains: []string{`RetailPriceImprovement`},
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
			got, err := tc.req.MarshalJSON()
			if tc.expectedErr != nil {
				require.ErrorIs(t, err, tc.expectedErr, "MarshalJSON must return the expected error")
				return
			}
			require.NoError(t, err, "MarshalJSON must not error")
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
		exp := []*CrossExchangeBatchCancelOrderResponse{
			{OrderID: "123456", Accepted: true},
			{Text: "crossex-test-1", Label: "TRADE_ORDER_NOT_FOUND_ERROR", Message: "The order was not found"},
		}
		assert.Equal(t, exp, result, "CancelBatchCrossExchangeOrders should decode every field")
		return
	}
	assert.NotEmpty(t, result, "CancelBatchCrossExchangeOrders should return a result for every order")
}

func TestCancelCrossExchangeOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelCrossExchangeOrder(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelCrossExchangeOrder must require an order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelCrossExchangeOrder(t.Context(), crossExchangeTestOrderID)
	require.NoError(t, err, "CancelCrossExchangeOrder must not error")
	if mockTests {
		assert.Equal(t, &CrossExchangeOrderActionResponse{OrderID: crossExchangeTestOrderID, Text: "crossex-test-1"}, result, "CancelCrossExchangeOrder should decode every field")
		return
	}
	assert.NotEmpty(t, result.OrderID, "CancelCrossExchangeOrder should return the order ID")
}

func TestModifyCrossExchangeOrder(t *testing.T) {
	t.Parallel()
	_, err := e.ModifyCrossExchangeOrder(t.Context(), "", &CrossExchangeOrderUpdateRequest{})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "ModifyCrossExchangeOrder must require an order ID")
	_, err = e.ModifyCrossExchangeOrder(t.Context(), crossExchangeTestOrderID, nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ModifyCrossExchangeOrder must reject a nil request")
	_, err = e.ModifyCrossExchangeOrder(t.Context(), crossExchangeTestOrderID, &CrossExchangeOrderUpdateRequest{})
	require.ErrorIs(t, err, errNoValidParameterPassed, "ModifyCrossExchangeOrder must require a quantity or price")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.ModifyCrossExchangeOrder(t.Context(), crossExchangeTestOrderID, &CrossExchangeOrderUpdateRequest{Price: 64000})
	require.NoError(t, err, "ModifyCrossExchangeOrder must not error")
	if mockTests {
		assert.Equal(t, &CrossExchangeOrderActionResponse{OrderID: crossExchangeTestOrderID, Text: "crossex-test-1"}, result, "ModifyCrossExchangeOrder should decode every field")
		return
	}
	assert.NotEmpty(t, result.OrderID, "ModifyCrossExchangeOrder should return the order ID")
}

func TestGetCrossExchangeOrderDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetCrossExchangeOrderDetails(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetCrossExchangeOrderDetails must require an order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossExchangeOrderDetails(t.Context(), crossExchangeTestOrderID)
	require.NoError(t, err, "GetCrossExchangeOrderDetails must not error")
	if mockTests {
		exp := &CrossExchangeOrder{
			UserID:               "10001004",
			OrderID:              crossExchangeTestOrderID,
			Text:                 "crossex-test-1",
			State:                "FILLED",
			Symbol:               "BINANCE_SPOT_ADA_USDT",
			Side:                 "BUY",
			Type:                 "MARKET",
			Attribute:            "COMMON",
			ExchangeType:         "BINANCE",
			BusinessType:         "SPOT",
			QuoteQuantity:        7,
			TimeInForce:          "RPI",
			ExecutedQuantity:     12.9,
			ExecutedAmount:       6.96471,
			ExecutedAveragePrice: 0.5399,
			FeeCoin:              currency.ADA,
			Fee:                  0.0129,
			Leverage:             1,
			LastExecutedQuantity: 12.9,
			LastExecutedPrice:    0.5399,
			LastExecutedAmount:   6.96471,
			PositionSide:         "NONE",
			CreateTime:           types.Time(time.UnixMilli(1750681141933)),
			UpdateTime:           types.Time(time.UnixMilli(1750681142379)),
		}
		assert.Equal(t, exp, result, "GetCrossExchangeOrderDetails should decode every field")
		return
	}
	assert.NotEmpty(t, result.OrderID, "GetCrossExchangeOrderDetails should return the order")
}

func TestGetCrossExchangeConvertQuote(t *testing.T) {
	t.Parallel()
	_, err := e.GetCrossExchangeConvertQuote(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetCrossExchangeConvertQuote must reject a nil request")
	_, err = e.GetCrossExchangeConvertQuote(t.Context(), &CrossExchangeConvertQuoteRequest{})
	require.ErrorIs(t, err, errCrossExchangeExchangeTypeRequired, "GetCrossExchangeConvertQuote must require an exchange type")
	_, err = e.GetCrossExchangeConvertQuote(t.Context(), &CrossExchangeConvertQuoteRequest{ExchangeType: "BINANCE"})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetCrossExchangeConvertQuote must require a currency to sell")
	_, err = e.GetCrossExchangeConvertQuote(t.Context(), &CrossExchangeConvertQuoteRequest{ExchangeType: "BINANCE", FromCoin: currency.BTC})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetCrossExchangeConvertQuote must require a currency to buy")
	_, err = e.GetCrossExchangeConvertQuote(t.Context(), &CrossExchangeConvertQuoteRequest{ExchangeType: "BINANCE", FromCoin: currency.BTC, ToCoin: currency.USDT})
	require.ErrorIs(t, err, order.ErrAmountMustBeSet, "GetCrossExchangeConvertQuote must require an amount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossExchangeConvertQuote(t.Context(), &CrossExchangeConvertQuoteRequest{
		ExchangeType: "BINANCE",
		FromCoin:     currency.BTC,
		ToCoin:       currency.USDT,
		FromAmount:   1,
	})
	require.NoError(t, err, "GetCrossExchangeConvertQuote must not error")
	if mockTests {
		exp := &CrossExchangeConvertQuoteResponse{
			QuoteID:           "CTH46487058372",
			ValidMilliseconds: 5000,
			FromCoin:          currency.BTC,
			ToCoin:            currency.USDT,
			FromAmount:        1,
			ToAmount:          65000,
			Price:             65000,
		}
		assert.Equal(t, exp, result, "GetCrossExchangeConvertQuote should decode every field")
		return
	}
	assert.NotEmpty(t, result.QuoteID, "GetCrossExchangeConvertQuote should return a quote ID")
}

func TestExecuteCrossExchangeConvertOrder(t *testing.T) {
	t.Parallel()
	_, err := e.ExecuteCrossExchangeConvertOrder(t.Context(), "")
	require.ErrorIs(t, err, errQuoteIDRequired, "ExecuteCrossExchangeConvertOrder must require a quote ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.ExecuteCrossExchangeConvertOrder(t.Context(), "CTH46487058372")
	require.NoError(t, err, "ExecuteCrossExchangeConvertOrder must not error")
	if mockTests {
		assert.Equal(t, &CrossExchangeConvertOrderResponse{OrderID: "123456", Text: "123456"}, result, "ExecuteCrossExchangeConvertOrder should decode every field")
		return
	}
	assert.NotEmpty(t, result.OrderID, "ExecuteCrossExchangeConvertOrder should return an order ID")
}

func TestUpdateCrossExchangeAccount(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateCrossExchangeAccount(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "UpdateCrossExchangeAccount must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.UpdateCrossExchangeAccount(t.Context(), &CrossExchangeAccountUpdateRequest{
		PositionMode: "SINGLE",
		AccountMode:  "CROSS_EXCHANGE",
		ExchangeType: "BINANCE",
	})
	require.NoError(t, err, "UpdateCrossExchangeAccount must not error")
	if mockTests {
		exp := &CrossExchangeAccountUpdateResponse{PositionMode: "SINGLE", AccountMode: "CROSS_EXCHANGE", ExchangeType: "BINANCE"}
		assert.Equal(t, exp, result, "UpdateCrossExchangeAccount should decode every field")
		return
	}
	assert.NotNil(t, result, "UpdateCrossExchangeAccount should return the requested change")
}

func TestGetCrossExchangeAccountAssets(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossExchangeAccountAssets(t.Context(), "")
	require.NoError(t, err, "GetCrossExchangeAccountAssets must not error")
	if !mockTests {
		assert.NotNil(t, result, "GetCrossExchangeAccountAssets should return account data")
		return
	}
	exp := &CrossExchangeAccount{
		UserID:                "123456789",
		AvailableMargin:       1200,
		MarginBalance:         1200,
		InitialMargin:         500,
		MaintenanceMargin:     250,
		InitialMarginRate:     2.4,
		MaintenanceMarginRate: 4.8,
		PositionMode:          "SINGLE",
		AccountLimit:          5000,
		CreateTime:            types.Time(time.UnixMilli(1687573845000)),
		UpdateTime:            types.Time(time.UnixMilli(1687588938000)),
		AccountMode:           "CROSS_EXCHANGE",
		ExchangeType:          "CROSSEX",
		Assets: []*CrossExchangeAccountAsset{{
			UserID:                     "123456789",
			Coin:                       currency.USDT,
			ExchangeType:               "BINANCE",
			Balance:                    1000,
			UnrealisedPNL:              200,
			Equity:                     1200,
			FuturesInitialMargin:       400,
			FuturesMaintenanceMargin:   130,
			BorrowingInitialMargin:     100,
			BorrowingMaintenanceMargin: 120,
			AvailableBalance:           1000,
		}},
	}
	assert.Equal(t, exp, result, "GetCrossExchangeAccountAssets should decode every field")

	result, err = e.GetCrossExchangeAccountAssets(t.Context(), "BINANCE")
	require.NoError(t, err, "GetCrossExchangeAccountAssets must not error in isolated exchange mode")
	exp = &CrossExchangeAccount{
		UserID:                "123456789",
		AvailableMargin:       800,
		MarginBalance:         900,
		InitialMargin:         100,
		MaintenanceMargin:     50,
		InitialMarginRate:     9,
		MaintenanceMarginRate: 18,
		PositionMode:          "DUAL",
		AccountLimit:          5000,
		CreateTime:            types.Time(time.UnixMilli(1687573845000)),
		UpdateTime:            types.Time(time.UnixMilli(1687588938000)),
		AccountMode:           "ISOLATED_EXCHANGE",
		ExchangeType:          "BINANCE",
		Assets: []*CrossExchangeAccountAsset{{
			UserID:                   "123456789",
			Coin:                     currency.USDT,
			ExchangeType:             "BINANCE",
			Balance:                  900,
			Equity:                   900,
			FuturesInitialMargin:     100,
			FuturesMaintenanceMargin: 50,
			AvailableBalance:         800,
			Liability:                10,
		}},
	}
	assert.Equal(t, exp, result, "GetCrossExchangeAccountAssets should decode every isolated exchange field")
}

func TestSetCrossExchangeContractLeverage(t *testing.T) {
	t.Parallel()
	_, err := e.SetCrossExchangeContractLeverage(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SetCrossExchangeContractLeverage must reject a nil request")
	_, err = e.SetCrossExchangeContractLeverage(t.Context(), &CrossExchangeLeverageRequest{})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "SetCrossExchangeContractLeverage must require a symbol")
	_, err = e.SetCrossExchangeContractLeverage(t.Context(), &CrossExchangeLeverageRequest{
		Symbol: newCrossExchangeSymbol("BINANCE", asset.Futures, currency.NewBTCUSDT()),
	})
	require.ErrorIs(t, err, errCrossExchangeLeverageRequired, "SetCrossExchangeContractLeverage must require a leverage")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SetCrossExchangeContractLeverage(t.Context(), &CrossExchangeLeverageRequest{
		Symbol:   newCrossExchangeSymbol("BINANCE", asset.Futures, currency.NewBTCUSDT()),
		Leverage: 10,
	})
	require.NoError(t, err, "SetCrossExchangeContractLeverage must not error")
	if mockTests {
		assert.Equal(t, &CrossExchangeLeverageResponse{Symbol: "BINANCE_FUTURE_BTC_USDT", Leverage: 10}, result, "SetCrossExchangeContractLeverage should decode every field")
		return
	}
	assert.NotEmpty(t, result.Symbol, "SetCrossExchangeContractLeverage should return the symbol")
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
		exp     map[string]types.Number
	}{
		{exp: map[string]types.Number{"BINANCE_FUTURE_BTC_USDT": 10, "OKX_FUTURE_BTC_USDT": 10}},
		{symbols: []CrossExchangeSymbolIdentifier{newCrossExchangeSymbol("BINANCE", asset.Futures, currency.NewBTCUSDT())}, exp: map[string]types.Number{"BINANCE_FUTURE_BTC_USDT": 10}},
	} {
		result, err := e.GetCrossExchangeContractLeverage(t.Context(), tc.symbols)
		require.NoError(t, err, "GetCrossExchangeContractLeverage must not error")
		if mockTests {
			assert.Equal(t, tc.exp, result, "GetCrossExchangeContractLeverage should decode every symbol")
		}
	}
}

func TestSetCrossExchangeMarginLeverage(t *testing.T) {
	t.Parallel()
	_, err := e.SetCrossExchangeMarginLeverage(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SetCrossExchangeMarginLeverage must reject a nil request")
	_, err = e.SetCrossExchangeMarginLeverage(t.Context(), &CrossExchangeLeverageRequest{})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "SetCrossExchangeMarginLeverage must require a symbol")
	_, err = e.SetCrossExchangeMarginLeverage(t.Context(), &CrossExchangeLeverageRequest{
		Symbol: newCrossExchangeSymbol("BINANCE", asset.Margin, currency.NewBTCUSDT()),
	})
	require.ErrorIs(t, err, errCrossExchangeLeverageRequired, "SetCrossExchangeMarginLeverage must require a leverage")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SetCrossExchangeMarginLeverage(t.Context(), &CrossExchangeLeverageRequest{
		Symbol:   newCrossExchangeSymbol("BINANCE", asset.Margin, currency.NewBTCUSDT()),
		Leverage: 5,
	})
	require.NoError(t, err, "SetCrossExchangeMarginLeverage must not error")
	if mockTests {
		assert.Equal(t, &CrossExchangeLeverageResponse{Symbol: "BINANCE_MARGIN_BTC_USDT", Leverage: 5}, result, "SetCrossExchangeMarginLeverage should decode every field")
		return
	}
	assert.NotEmpty(t, result.Symbol, "SetCrossExchangeMarginLeverage should return the symbol")
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
		exp     map[string]types.Number
	}{
		{exp: map[string]types.Number{"BINANCE_MARGIN_BTC_USDT": 5, "OKX_MARGIN_BTC_USDT": 5}},
		{symbols: []CrossExchangeSymbolIdentifier{newCrossExchangeSymbol("BINANCE", asset.Margin, currency.NewBTCUSDT())}, exp: map[string]types.Number{"BINANCE_MARGIN_BTC_USDT": 5}},
	} {
		result, err := e.GetCrossExchangeMarginLeverage(t.Context(), tc.symbols)
		require.NoError(t, err, "GetCrossExchangeMarginLeverage must not error")
		if mockTests {
			assert.Equal(t, tc.exp, result, "GetCrossExchangeMarginLeverage should decode every symbol")
		}
	}
}

func TestCloseCrossExchangePosition(t *testing.T) {
	t.Parallel()
	_, err := e.CloseCrossExchangePosition(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CloseCrossExchangePosition must reject a nil request")
	_, err = e.CloseCrossExchangePosition(t.Context(), &CrossExchangeClosePositionRequest{})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "CloseCrossExchangePosition must require a symbol")
	_, err = e.CloseCrossExchangePosition(t.Context(), &CrossExchangeClosePositionRequest{Symbol: newCrossExchangeSymbol("BINANCE", asset.Margin, currency.NewBTCUSDT())})
	require.ErrorIs(t, err, errCrossExchangePositionSideRequired, "CloseCrossExchangePosition must require a position side for a margin position")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CloseCrossExchangePosition(t.Context(), &CrossExchangeClosePositionRequest{
		Symbol:       newCrossExchangeSymbol("BINANCE", asset.Futures, currency.NewBTCUSDT()),
		PositionSide: "LONG",
	})
	require.NoError(t, err, "CloseCrossExchangePosition must not error")
	if mockTests {
		assert.Equal(t, &CrossExchangeOrderActionResponse{OrderID: "123456", Text: "123456"}, result, "CloseCrossExchangePosition should decode every field")
		return
	}
	assert.NotEmpty(t, result.OrderID, "CloseCrossExchangePosition should return the closing order ID")
}

func TestSetCrossExchangePositionMarginMode(t *testing.T) {
	t.Parallel()
	symbol := newCrossExchangeSymbol("HYPERLIQUID", asset.Futures, currency.NewPair(currency.BTC, currency.USDC))
	_, err := e.SetCrossExchangePositionMarginMode(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SetCrossExchangePositionMarginMode must reject a nil request")
	_, err = e.SetCrossExchangePositionMarginMode(t.Context(), &CrossExchangeMarginModeRequest{})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "SetCrossExchangePositionMarginMode must require a symbol")
	_, err = e.SetCrossExchangePositionMarginMode(t.Context(), &CrossExchangeMarginModeRequest{Symbol: symbol})
	require.ErrorIs(t, err, errCrossExchangeMarginModeRequired, "SetCrossExchangePositionMarginMode must require a margin mode")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SetCrossExchangePositionMarginMode(t.Context(), &CrossExchangeMarginModeRequest{Symbol: symbol, MarginMode: "ISOLATED"})
	require.NoError(t, err, "SetCrossExchangePositionMarginMode must not error")
	if mockTests {
		assert.Equal(t, &CrossExchangeMarginModeResponse{Symbol: "HYPERLIQUID_FUTURE_BTC_USDC", MarginMode: "ISOLATED"}, result, "SetCrossExchangePositionMarginMode should decode every field")
		return
	}
	assert.NotNil(t, result, "SetCrossExchangePositionMarginMode should return the margin mode")
}

func TestGetCrossExchangePositionMarginMode(t *testing.T) {
	t.Parallel()
	_, err := e.GetCrossExchangePositionMarginMode(t.Context(), CrossExchangeSymbolIdentifier{})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "GetCrossExchangePositionMarginMode must require a symbol")
	_, err = e.GetCrossExchangePositionMarginMode(t.Context(), CrossExchangeSymbolIdentifier{Exchange: "HYPERLIQUID", Asset: asset.Futures})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetCrossExchangePositionMarginMode must reject a symbol it cannot format")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossExchangePositionMarginMode(t.Context(), newCrossExchangeSymbol("HYPERLIQUID", asset.Futures, currency.NewPair(currency.BTC, currency.USDC)))
	require.NoError(t, err, "GetCrossExchangePositionMarginMode must not error")
	if mockTests {
		assert.Equal(t, &CrossExchangeMarginModeResponse{Symbol: "HYPERLIQUID_FUTURE_BTC_USDC", MarginMode: "ISOLATED"}, result, "GetCrossExchangePositionMarginMode should decode every field")
		return
	}
	assert.NotNil(t, result, "GetCrossExchangePositionMarginMode should return the margin mode")
}

func TestUpdateCrossExchangeIsolatedMargin(t *testing.T) {
	t.Parallel()
	symbol := newCrossExchangeSymbol("HYPERLIQUID", asset.Futures, currency.NewPair(currency.BTC, currency.USDC))
	_, err := e.UpdateCrossExchangeIsolatedMargin(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "UpdateCrossExchangeIsolatedMargin must reject a nil request")
	_, err = e.UpdateCrossExchangeIsolatedMargin(t.Context(), &CrossExchangeIsolatedMarginRequest{})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "UpdateCrossExchangeIsolatedMargin must require a symbol")
	_, err = e.UpdateCrossExchangeIsolatedMargin(t.Context(), &CrossExchangeIsolatedMarginRequest{Symbol: symbol})
	require.ErrorIs(t, err, errCrossExchangeMarginRequired, "UpdateCrossExchangeIsolatedMargin must require a margin adjustment")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.UpdateCrossExchangeIsolatedMargin(t.Context(), &CrossExchangeIsolatedMarginRequest{Symbol: symbol, Margin: -30, PositionSide: "NONE"})
	require.NoError(t, err, "UpdateCrossExchangeIsolatedMargin must not error")
	if mockTests {
		exp := &CrossExchangeIsolatedMarginResponse{Symbol: "HYPERLIQUID_FUTURE_BTC_USDC", Margin: -30, PositionSide: "NONE"}
		assert.Equal(t, exp, result, "UpdateCrossExchangeIsolatedMargin should decode every field")
		return
	}
	assert.NotNil(t, result, "UpdateCrossExchangeIsolatedMargin should return the adjustment")
}

func TestGetCrossExchangeInterestRates(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	bch := &CrossExchangeInterestRate{Coin: currency.BCH, ExchangeType: "GATE", HourInterestRate: 0.00000465, Time: types.Time(time.UnixMilli(1763371200000))}
	for _, tc := range []struct {
		coin         currency.Code
		exchangeType string
		exp          []*CrossExchangeInterestRate
	}{
		{exp: []*CrossExchangeInterestRate{bch, {Coin: currency.ADA, ExchangeType: "BINANCE", HourInterestRate: 0.0000036558334, Time: types.Time(time.UnixMilli(1763371200000))}}},
		{coin: currency.BCH, exchangeType: "GATE", exp: []*CrossExchangeInterestRate{bch}},
	} {
		result, err := e.GetCrossExchangeInterestRates(t.Context(), tc.coin, tc.exchangeType)
		require.NoError(t, err, "GetCrossExchangeInterestRates must not error")
		if mockTests {
			assert.Equal(t, tc.exp, result, "GetCrossExchangeInterestRates should decode every field")
		}
	}
}

func TestGetCrossExchangeUserFeeRates(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossExchangeUserFeeRates(t.Context())
	require.NoError(t, err, "GetCrossExchangeUserFeeRates must not error")
	if mockTests {
		exp := []*CrossExchangeFee{
			{
				ExchangeType:   "BINANCE",
				SpotMakerFee:   0.0001,
				SpotTakerFee:   0.00025,
				FutureMakerFee: 0.00006,
				FutureTakerFee: 0.00022,
				SpecialFeeList: []*CrossExchangeSpecialFee{},
			},
			{
				ExchangeType:      "GATE",
				SpotMakerFee:      0.0001,
				SpotTakerFee:      0.00025,
				SpotRPIMakerFee:   0.00005,
				FutureMakerFee:    0.00006,
				FutureTakerFee:    0.00022,
				FutureRPIMakerFee: 0.00003,
				SpecialFeeList:    []*CrossExchangeSpecialFee{{Symbol: "GATE_SPOT_FLOW_USDT", TakerFeeRate: 0.0004, MakerFeeRate: 0.0001, RPIFeeRate: 0.00002}},
			},
		}
		assert.Equal(t, exp, result, "GetCrossExchangeUserFeeRates should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetCrossExchangeUserFeeRates should return fee rates")
}

func TestGetCrossExchangeContractPositions(t *testing.T) {
	t.Parallel()
	_, err := e.GetCrossExchangeContractPositions(t.Context(), CrossExchangeSymbolIdentifier{Exchange: "BINANCE", Asset: asset.Futures}, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetCrossExchangeContractPositions must reject a symbol it cannot format")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	exp := []*CrossExchangePosition{{
		UserID:            "10001004",
		PositionID:        "20062926505289216",
		Symbol:            "BINANCE_FUTURE_BTC_USDT",
		PositionSide:      "LONG",
		InitialMargin:     5.79934625,
		MarginMode:        "CROSS",
		MaintenanceMargin: 0.06229625,
		PositionQuantity:  10,
		PositionValue:     5.795,
		UnrealisedPNL:     0.369,
		UnrealisedPNLRate: 0.068,
		EntryPrice:        0.5426,
		MarkPrice:         0.5795,
		Leverage:          1,
		MaxLeverage:       18,
		RiskLimit:         1,
		Fee:               0.002713,
		FundingFee:        -0.01,
		FundingTime:       types.Time(time.UnixMilli(1750700000000)),
		CreateTime:        types.Time(time.UnixMilli(1750682334273)),
		UpdateTime:        types.Time(time.UnixMilli(1750730699867)),
		ClosedPNL:         12,
	}}
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
			assert.Equal(t, exp, result, "GetCrossExchangeContractPositions should decode every field")
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
	exp := []*CrossExchangeMarginPosition{{
		UserID:              "12345",
		PositionID:          "20126312530221056",
		Symbol:              "BINANCE_MARGIN_BTC_USDT",
		PositionSide:        "LONG",
		InitialMargin:       10,
		MaintenanceMargin:   5,
		AssetQuantity:       0.001,
		AssetCoin:           currency.BTC,
		PositionValue:       65,
		Liability:           0.0001708920658,
		LiabilityCoin:       currency.USDT,
		Interest:            0.0001708920658,
		MaxPositionQuantity: 0.002,
		EntryPrice:          64000,
		IndexPrice:          65000,
		UnrealisedPNL:       -0.0001708920658,
		UnrealisedPNLRate:   -3,
		Leverage:            3,
		MaxLeverage:         5,
		CreateTime:          types.Time(time.UnixMilli(1765794740152)),
		UpdateTime:          types.Time(time.UnixMilli(1766716075010)),
	}}
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
			assert.Equal(t, exp, result, "GetCrossExchangeMarginPositions should decode every field")
		}
	}
}

func TestGetCrossExchangeADLRank(t *testing.T) {
	t.Parallel()
	_, err := e.GetCrossExchangeADLRank(t.Context(), CrossExchangeSymbolIdentifier{})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "GetCrossExchangeADLRank must require a symbol")
	_, err = e.GetCrossExchangeADLRank(t.Context(), CrossExchangeSymbolIdentifier{Exchange: "BINANCE", Asset: asset.Futures})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetCrossExchangeADLRank must reject a symbol it cannot format")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossExchangeADLRank(t.Context(), newCrossExchangeSymbol("BINANCE", asset.Futures, currency.NewPair(currency.ADA, currency.USDT)))
	require.NoError(t, err, "GetCrossExchangeADLRank must not error")
	if mockTests {
		exp := []*CrossExchangeADLRank{{UserID: "111", Symbol: "BINANCE_FUTURE_ADA_USDT", CrossExchangeADLRank: 1, ExchangeADLRank: 1}}
		assert.Equal(t, exp, result, "GetCrossExchangeADLRank should decode every field")
		return
	}
	assert.NotNil(t, result, "GetCrossExchangeADLRank should return the ranking")
}

func TestGetCrossExchangeOpenOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetCrossExchangeOpenOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetCrossExchangeOpenOrders must reject a nil request")
	_, err = e.GetCrossExchangeOpenOrders(t.Context(), &GetCrossExchangeOpenOrdersRequest{Symbol: CrossExchangeSymbolIdentifier{Exchange: "BINANCE", Asset: asset.Futures}})
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
		exp := []*CrossExchangeOrder{{
			UserID:               "10001004",
			OrderID:              "2048529119934720",
			Text:                 "2048529119934720",
			State:                "PARTIALLY_FILLED",
			Symbol:               "BINANCE_FUTURE_BTC_USDT",
			Side:                 "BUY",
			Type:                 "LIMIT",
			Attribute:            "COMMON",
			ExchangeType:         "BINANCE",
			BusinessType:         "FUTURE",
			Quantity:             6,
			Price:                65000,
			TimeInForce:          "POC",
			ExecutedQuantity:     1,
			ExecutedAmount:       65000,
			ExecutedAveragePrice: 65000,
			FeeCoin:              currency.USDT,
			Fee:                  0.0110354,
			ReduceOnly:           true,
			Leverage:             10,
			LastExecutedQuantity: 1,
			LastExecutedPrice:    65000,
			LastExecutedAmount:   65000,
			PositionSide:         "LONG",
			CreateTime:           types.Time(time.UnixMilli(1750682602377)),
			UpdateTime:           types.Time(time.UnixMilli(1750682602413)),
		}}
		assert.Equal(t, exp, result, "GetCrossExchangeOpenOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "GetCrossExchangeOpenOrders should return orders")
}

func TestGetCrossExchangeOrderHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetCrossExchangeOrderHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetCrossExchangeOrderHistory must reject a nil request")
	_, err = e.GetCrossExchangeOrderHistory(t.Context(), &GetCrossExchangeOrderHistoryRequest{Symbol: CrossExchangeSymbolIdentifier{Exchange: "BINANCE", Asset: asset.Futures}})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetCrossExchangeOrderHistory must reject a symbol it cannot format")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	startTime, endTime := getTime()
	result, err := e.GetCrossExchangeOrderHistory(t.Context(), &GetCrossExchangeOrderHistoryRequest{
		Page:       1,
		Limit:      10,
		Symbol:     newCrossExchangeSymbol("BINANCE", asset.Futures, currency.NewBTCUSDT()),
		StartTime:  startTime,
		EndTime:    endTime,
		Attributes: []string{"COMMON", "LIQ"},
	})
	require.NoError(t, err, "GetCrossExchangeOrderHistory must not error")
	if mockTests {
		exp := []*CrossExchangeOrder{{
			UserID:               "10001004",
			OrderID:              "2048522992198912",
			Text:                 "2048522992198912",
			State:                "FILLED",
			Symbol:               "BINANCE_FUTURE_BTC_USDT",
			Side:                 "SELL",
			Type:                 "MARKET",
			Attribute:            "LIQ",
			ExchangeType:         "BINANCE",
			BusinessType:         "FUTURE",
			Quantity:             0.01,
			TimeInForce:          "IOC",
			ExecutedQuantity:     0.01,
			ExecutedAmount:       650,
			ExecutedAveragePrice: 65000,
			FeeCoin:              currency.USDT,
			Fee:                  0.325,
			ReduceOnly:           true,
			Leverage:             5,
			LastExecutedQuantity: 0.01,
			LastExecutedPrice:    65000,
			LastExecutedAmount:   650,
			PositionSide:         "NONE",
			CreateTime:           types.Time(time.UnixMilli(1750681141933)),
			UpdateTime:           types.Time(time.UnixMilli(1750681142379)),
		}}
		assert.Equal(t, exp, result, "GetCrossExchangeOrderHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetCrossExchangeOrderHistory should return orders")
}

func TestGetCrossExchangeContractPositionHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetCrossExchangeContractPositionHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetCrossExchangeContractPositionHistory must reject a nil request")
	_, err = e.GetCrossExchangeContractPositionHistory(t.Context(), &GetCrossExchangePositionHistoryRequest{Symbol: CrossExchangeSymbolIdentifier{Exchange: "BINANCE", Asset: asset.Futures}})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetCrossExchangeContractPositionHistory must reject a symbol it cannot format")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	startTime, endTime := getTime()
	result, err := e.GetCrossExchangeContractPositionHistory(t.Context(), &GetCrossExchangePositionHistoryRequest{
		Page:      1,
		Limit:     10,
		Symbol:    newCrossExchangeSymbol("BINANCE", asset.Futures, currency.NewBTCUSDT()),
		StartTime: startTime,
		EndTime:   endTime,
	})
	require.NoError(t, err, "GetCrossExchangeContractPositionHistory must not error")
	if mockTests {
		exp := []*CrossExchangeHistoricalPosition{{
			PositionID:          "20064013106942976",
			UserID:              "12345678",
			Symbol:              "BINANCE_FUTURE_BTC_USDT",
			ClosedType:          "COMPLETE_CLOSED",
			ClosedPNL:           -0.001,
			ClosedPNLRate:       -0.001,
			OpenAveragePrice:    0.5598,
			ClosedAveragePrice:  0.5597,
			MaxPositionQuantity: 10,
			ClosedQuantity:      10,
			ClosedValue:         5.597,
			Fee:                 0.0055975,
			FundingFee:          0.01,
			PositionSide:        "LONG",
			PositionMode:        "DUAL",
			Leverage:            1,
			MarginMode:          "CROSS",
			BusinessType:        "FUTURE",
			CreateTime:          types.Time(time.UnixMilli(1750941400632)),
			UpdateTime:          types.Time(time.UnixMilli(1750941402661)),
		}}
		assert.Equal(t, exp, result, "GetCrossExchangeContractPositionHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetCrossExchangeContractPositionHistory should return positions")
}

func TestGetCrossExchangeMarginPositionHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetCrossExchangeMarginPositionHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetCrossExchangeMarginPositionHistory must reject a nil request")
	_, err = e.GetCrossExchangeMarginPositionHistory(t.Context(), &GetCrossExchangePositionHistoryRequest{Symbol: CrossExchangeSymbolIdentifier{Exchange: "BINANCE", Asset: asset.Margin}})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetCrossExchangeMarginPositionHistory must reject a symbol it cannot format")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	startTime, endTime := getTime()
	result, err := e.GetCrossExchangeMarginPositionHistory(t.Context(), &GetCrossExchangePositionHistoryRequest{
		Page:      1,
		Limit:     10,
		Symbol:    newCrossExchangeSymbol("BINANCE", asset.Margin, currency.NewBTCUSDT()),
		StartTime: startTime,
		EndTime:   endTime,
	})
	require.NoError(t, err, "GetCrossExchangeMarginPositionHistory must not error")
	if mockTests {
		exp := []*CrossExchangeHistoricalMarginPosition{{
			PositionID:          "20064013106942977",
			UserID:              "12345678",
			Symbol:              "BINANCE_MARGIN_BTC_USDT",
			ClosedType:          "PARTIAL_CLOSED",
			ClosedPNL:           5,
			ClosedPNLRate:       0.1,
			OpenAveragePrice:    64000,
			ClosedAveragePrice:  65000,
			MaxPositionQuantity: 0.01,
			ClosedQuantity:      0.005,
			ClosedValue:         325,
			PositionSide:        "LONG",
			Leverage:            3,
			Interest:            0.2,
			BusinessType:        "MARGIN",
			CreateTime:          types.Time(time.UnixMilli(1750941400632)),
			UpdateTime:          types.Time(time.UnixMilli(1750941402661)),
		}}
		assert.Equal(t, exp, result, "GetCrossExchangeMarginPositionHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetCrossExchangeMarginPositionHistory should return positions")
}

func TestGetCrossExchangeMarginInterestHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetCrossExchangeMarginInterestHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetCrossExchangeMarginInterestHistory must reject a nil request")
	_, err = e.GetCrossExchangeMarginInterestHistory(t.Context(), &GetCrossExchangeMarginInterestHistoryRequest{Symbol: CrossExchangeSymbolIdentifier{Exchange: "BINANCE", Asset: asset.Margin}})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetCrossExchangeMarginInterestHistory must reject a symbol it cannot format")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	startTime, endTime := getTime()
	result, err := e.GetCrossExchangeMarginInterestHistory(t.Context(), &GetCrossExchangeMarginInterestHistoryRequest{
		Symbol:       newCrossExchangeSymbol("BINANCE", asset.Margin, currency.NewBTCUSDT()),
		StartTime:    startTime,
		EndTime:      endTime,
		Page:         1,
		Limit:        10,
		ExchangeType: "BINANCE",
	})
	require.NoError(t, err, "GetCrossExchangeMarginInterestHistory must not error")
	if mockTests {
		exp := []*CrossExchangeMarginInterestRecord{{
			UserID:        "2124575357",
			Symbol:        "BINANCE_MARGIN_BTC_USDT",
			InterestID:    "2115944013038336",
			LiabilityID:   "2115944013038080",
			Liability:     2,
			LiabilityCoin: currency.USDT,
			Interest:      0.00000732,
			InterestRate:  0.00000366,
			InterestType:  "IMMEDIATE_OPEN_ORDER",
			CreateTime:    types.Time(time.UnixMilli(1766755565807)),
			ExchangeType:  "BINANCE",
		}}
		assert.Equal(t, exp, result, "GetCrossExchangeMarginInterestHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetCrossExchangeMarginInterestHistory should return interest records")
}

func TestGetCrossExchangeTradeHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetCrossExchangeTradeHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetCrossExchangeTradeHistory must reject a nil request")
	_, err = e.GetCrossExchangeTradeHistory(t.Context(), &GetCrossExchangeTradeHistoryRequest{Symbol: CrossExchangeSymbolIdentifier{Exchange: "BINANCE", Asset: asset.Futures}})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetCrossExchangeTradeHistory must reject a symbol it cannot format")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	startTime, endTime := getTime()
	result, err := e.GetCrossExchangeTradeHistory(t.Context(), &GetCrossExchangeTradeHistoryRequest{
		Page:      1,
		Limit:     10,
		Symbol:    newCrossExchangeSymbol("BINANCE", asset.Futures, currency.NewBTCUSDT()),
		StartTime: startTime,
		EndTime:   endTime,
	})
	require.NoError(t, err, "GetCrossExchangeTradeHistory must not error")
	if mockTests {
		exp := []*CrossExchangeTrade{{
			UserID:        "3511316454450547",
			TransactionID: "2049614605858560",
			OrderID:       "2049614605857536",
			Text:          "2049614605857536",
			Symbol:        "BINANCE_FUTURE_BTC_USDT",
			ExchangeType:  "BINANCE",
			BusinessType:  "FUTURE",
			Side:          "SELL",
			Quantity:      10,
			Price:         0.5597,
			Fee:           0.0027985,
			FeeCoin:       currency.USDT,
			FeeRate:       0.0005,
			MatchRole:     "MAKER",
			RealisedPNL:   -0.001,
			PositionMode:  "DUAL",
			PositionSide:  "LONG",
			CreateTime:    types.Time(time.UnixMilli(1750941402661)),
		}}
		assert.Equal(t, exp, result, "GetCrossExchangeTradeHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetCrossExchangeTradeHistory should return trades")
}

func TestGetCrossExchangeAccountBook(t *testing.T) {
	t.Parallel()
	_, err := e.GetCrossExchangeAccountBook(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetCrossExchangeAccountBook must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.GetCrossExchangeAccountBook(t.Context(), &GetCrossExchangeAccountBookRequest{StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetCrossExchangeAccountBook must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossExchangeAccountBook(t.Context(), &GetCrossExchangeAccountBookRequest{
		Page:          1,
		Limit:         10,
		Coin:          currency.USDT,
		StatementType: "FUNDING_FEE",
		StartTime:     startTime,
		EndTime:       endTime,
	})
	require.NoError(t, err, "GetCrossExchangeAccountBook must not error")
	if mockTests {
		exp := []*CrossExchangeAccountBookRecord{{
			ID:            "121",
			UserID:        "12345678",
			BusinessID:    "20818182821",
			StatementType: "FUNDING_FEE",
			ExchangeType:  "BINANCE",
			Coin:          currency.USDT,
			Symbol:        "BINANCE_FUTURE_BTC_USDT",
			Change:        -0.002,
			Balance:       81,
			CreateTime:    types.Time(time.UnixMilli(1750941402661)),
		}}
		assert.Equal(t, exp, result, "GetCrossExchangeAccountBook should decode every field")
		return
	}
	assert.NotNil(t, result, "GetCrossExchangeAccountBook should return account book records")
}

func TestGetCrossExchangeCoinDiscountRates(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		exchangeType string
		exp          []*CrossExchangeCoinDiscountRate
	}{
		{
			exp: []*CrossExchangeCoinDiscountRate{
				{Coin: currency.SOL, ExchangeType: "GATE", Tier: "1", MaxValue: 10000, DiscountRate: 0.95},
				{Coin: currency.SOL, ExchangeType: "GATE", Tier: "2", MinValue: 10000, MaxValue: 20000, DiscountRate: 0.93},
			},
		},
		{
			exchangeType: "BINANCE",
			exp:          []*CrossExchangeCoinDiscountRate{{Coin: currency.ETH, ExchangeType: "BINANCE", Tier: "1", MaxValue: 10000, DiscountRate: 0.95}},
		},
	} {
		result, err := e.GetCrossExchangeCoinDiscountRates(t.Context(), currency.ETH, tc.exchangeType)
		require.NoError(t, err, "GetCrossExchangeCoinDiscountRates must not error")
		if mockTests {
			assert.Equal(t, tc.exp, result, "GetCrossExchangeCoinDiscountRates should decode every field")
		}
	}
}

func TestGetCrossExchangeMarketTickers(t *testing.T) {
	t.Parallel()
	_, err := e.GetCrossExchangeMarketTickers(t.Context(), []CrossExchangeSymbolIdentifier{newCrossExchangeSymbol("GATE", asset.Margin, currency.NewBTCUSDT())})
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetCrossExchangeMarketTickers must reject a margin symbol")
	_, err = e.GetCrossExchangeMarketTickers(t.Context(), []CrossExchangeSymbolIdentifier{{Exchange: "GATE", Asset: asset.Futures}})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetCrossExchangeMarketTickers must reject a symbol it cannot format")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossExchangeMarketTickers(t.Context(), []CrossExchangeSymbolIdentifier{
		newCrossExchangeSymbol("GATE", asset.Futures, currency.NewBTCUSDT()),
		newCrossExchangeSymbol("GATE", asset.Spot, currency.NewBTCUSDT()),
	})
	require.NoError(t, err, "GetCrossExchangeMarketTickers must not error")
	if mockTests {
		exp := []*CrossExchangeMarketTickerResponse{
			{
				Symbol:            "GATE_FUTURE_BTC_USDT",
				LastPrice:         64052.4,
				Open24Hour:        65144.7,
				Low24Hour:         64375,
				High24Hour:        65734.8,
				BaseVolume24Hour:  31705,
				QuoteVolume24Hour: 2063128626,
				MarkPrice:         65148.9,
				IndexPrice:        65174.38,
				OpenInterest:      65568.2144,
				OpenInterestQuote: 4271697043.12416,
				Timestamp:         types.Time(time.UnixMilli(1785168000000)),
			},
			{
				Symbol:            "GATE_SPOT_BTC_USDT",
				LastPrice:         65179.4,
				Low24Hour:         64410.9,
				High24Hour:        65744,
				BaseVolume24Hour:  3480.769758,
				QuoteVolume24Hour: 226794942.82361,
				Timestamp:         types.Time(time.UnixMilli(1785168000000)),
			},
		}
		assert.Equal(t, exp, result, "GetCrossExchangeMarketTickers should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetCrossExchangeMarketTickers should return tickers")
}

func TestGetCrossExchangeFundingInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetCrossExchangeFundingInfo(t.Context(), []CrossExchangeSymbolIdentifier{{Exchange: "BINANCE", Asset: asset.Futures}})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetCrossExchangeFundingInfo must reject a symbol it cannot format")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossExchangeFundingInfo(t.Context(), []CrossExchangeSymbolIdentifier{
		newCrossExchangeSymbol("BINANCE", asset.Futures, currency.NewBTCUSDT()),
		newCrossExchangeSymbol("KRAKEN", asset.Futures, currency.NewBTCUSD()),
	})
	require.NoError(t, err, "GetCrossExchangeFundingInfo must not error")
	if mockTests {
		exp := []*CrossExchangeFundingInfoResponse{
			{Symbol: "BINANCE_FUTURE_BTC_USDT", FundingRate: 0.00006537, FundingIntervalSeconds: 28800, NextFundingTime: types.Time(time.UnixMilli(1785168000000))},
			{Symbol: "KRAKEN_FUTURE_BTC_USD", FundingRate: 0.000011898754310345, FundingIntervalSeconds: 3600, NextFundingTime: types.Time(time.UnixMilli(1785139200000))},
		}
		assert.Equal(t, exp, result, "GetCrossExchangeFundingInfo should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetCrossExchangeFundingInfo should return funding information")
}
