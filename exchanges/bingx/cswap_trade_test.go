package bingx

import (
	"io"
	"math"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// coinMarginedSignedQuery returns a signed request's parameters without its timestamp and signature, checking the
// signature covers them
func coinMarginedSignedQuery(tb testing.TB, r *http.Request) url.Values {
	tb.Helper()
	verifyQuerySignature(tb, r.URL.RawQuery)
	q := r.URL.Query()
	assert.NotEmpty(tb, q.Get("timestamp"), "timestamp should be sent")
	q.Del("timestamp")
	q.Del("signature")
	return q
}

func TestPlaceCoinMarginedOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceCoinMarginedOrder(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer, "PlaceCoinMarginedOrder should reject a nil request")
	for _, tc := range []struct {
		name string
		req  *PlaceCoinMarginedOrderRequest
		err  error
	}{
		{"empty symbol", &PlaceCoinMarginedOrderRequest{Type: OrderTypeLimit, Side: SideBuy, PositionSide: PositionSideLong}, currency.ErrCurrencyPairEmpty},
		{"empty type", &PlaceCoinMarginedOrderRequest{Symbol: currency.NewBTCUSD(), Side: SideBuy, PositionSide: PositionSideLong}, order.ErrTypeIsInvalid},
		{"empty side", &PlaceCoinMarginedOrderRequest{Symbol: currency.NewBTCUSD(), Type: OrderTypeLimit, PositionSide: PositionSideLong}, order.ErrSideIsInvalid},
		{"empty position side", &PlaceCoinMarginedOrderRequest{Symbol: currency.NewBTCUSD(), Type: OrderTypeLimit, Side: SideBuy}, errPositionSideRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.PlaceCoinMarginedOrder(t.Context(), tc.req)
			assert.ErrorIs(t, err, tc.err, "PlaceCoinMarginedOrder should reject the order")
		})
	}
	valid := PlaceCoinMarginedOrderRequest{Symbol: currency.NewBTCUSD(), Type: OrderTypeLimit, Side: SideBuy, PositionSide: PositionSideLong}
	_, err = new(Exchange).PlaceCoinMarginedOrder(t.Context(), &valid)
	assert.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "PlaceCoinMarginedOrder should return the symbol formatting error")
	unencodable := valid
	unencodable.StopLoss = &CoinMarginedTakeProfitStopLossRequest{Type: OrderTypeStopMarket, StopPrice: math.NaN()}
	_, err = e.PlaceCoinMarginedOrder(t.Context(), &unencodable)
	assert.ErrorContains(t, err, "stop loss", "PlaceCoinMarginedOrder should return a stop loss encoding error")
	unencodable = valid
	unencodable.TakeProfit = &CoinMarginedTakeProfitStopLossRequest{Type: OrderTypeTakeProfitMarket, StopPrice: math.Inf(1)}
	_, err = e.PlaceCoinMarginedOrder(t.Context(), &unencodable)
	assert.ErrorContains(t, err, "take profit", "PlaceCoinMarginedOrder should return a take profit encoding error")

	t.Run("sends every parameter", func(t *testing.T) {
		t.Parallel()
		ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method, "method should be POST")
			assert.Equal(t, "/openApi/cswap/v2/trade/order", r.URL.Path, "path should be the order path")
			exp := url.Values{
				"symbol":        {"ETH-USD"},
				"type":          {"STOP"},
				"side":          {"SELL"},
				"positionSide":  {"SHORT"},
				"price":         {"3777"},
				"quantity":      {"20"},
				"stopPrice":     {"3700.5"},
				"workingType":   {"MARK_PRICE"},
				"stopLoss":      {`{"type":"STOP","stopPrice":3999,"price":2888,"workingType":"CONTRACT_PRICE"}`},
				"takeProfit":    {`{"type":"TAKE_PROFIT_MARKET","stopPrice":3666}`},
				"clientOrderId": {"gct-order-1"},
				"timeInForce":   {"GTC"},
				"recvWindow":    {"5000"},
			}
			assert.Equal(t, exp, coinMarginedSignedQuery(t, r), "query should hold every parameter")
			_, _ = io.WriteString(w, `{"code":0,"msg":"","data":{"order":{"orderId":"1802706634360750000","symbol":"ETH-USD"}}}`)
		})
		result, err := ex.PlaceCoinMarginedOrder(t.Context(), &PlaceCoinMarginedOrderRequest{
			Symbol:        currency.NewPair(currency.ETH, currency.USD),
			Type:          OrderTypeStop,
			Side:          SideSell,
			PositionSide:  PositionSideShort,
			Price:         3777,
			Quantity:      20,
			StopPrice:     3700.5,
			WorkingType:   WorkingTypeMarkPrice,
			StopLoss:      &CoinMarginedTakeProfitStopLossRequest{Type: OrderTypeStop, StopPrice: 3999, Price: 2888, WorkingType: WorkingTypeContractPrice},
			TakeProfit:    &CoinMarginedTakeProfitStopLossRequest{Type: OrderTypeTakeProfitMarket, StopPrice: 3666},
			ClientOrderID: "gct-order-1",
			TimeInForce:   TimeInForceGoodTillCancel,
		})
		require.NoError(t, err, "PlaceCoinMarginedOrder must not error")
		assert.Equal(t, &PlaceCoinMarginedOrderResponse{OrderID: 1802706634360750000, Symbol: "ETH-USD"}, result, "PlaceCoinMarginedOrder should return the placed order")
	})

	for _, tc := range []struct {
		name, body string
		err        error
	}{
		{"reply without an order", `{"code":0,"msg":"","data":{}}`, common.ErrNoResponse},
		{"reply without an order ID", `{"code":0,"msg":"","data":{"order":{"symbol":"ETH-USD"}}}`, errCoinMarginedOrderIDEmpty},
		{"API error", `{"code":109400,"msg":"Invalid parameters"}`, errAPIResponse},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newTestServerExchange(t, func(w http.ResponseWriter, _ *http.Request) {
				_, _ = io.WriteString(w, tc.body)
			})
			_, err := ex.PlaceCoinMarginedOrder(t.Context(), &valid)
			assert.ErrorIs(t, err, tc.err, "PlaceCoinMarginedOrder should reject the reply")
		})
	}

	t.Run("place order", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		if !mockTests {
			sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)
		}
		result, err := e.PlaceCoinMarginedOrder(t.Context(), &PlaceCoinMarginedOrderRequest{
			Symbol:       currency.NewBTCUSD(),
			Type:         OrderTypeLimit,
			Side:         SideBuy,
			PositionSide: PositionSideLong,
			Price:        10000,
			Quantity:     1,
			TimeInForce:  TimeInForcePostOnly,
		})
		require.NoError(t, err, "PlaceCoinMarginedOrder must not error")
		if mockTests {
			exp := &PlaceCoinMarginedOrderResponse{
				OrderID:      1802706634360750000,
				Symbol:       "ETH-USD",
				PositionSide: PositionSideShort,
				Side:         SideSell,
				Type:         OrderTypeLimit,
				Price:        3777,
				Quantity:     20,
				TimeInForce:  TimeInForceGoodTillCancel,
			}
			assert.Equal(t, exp, result, "PlaceCoinMarginedOrder should decode every field")
			return
		}
		assert.NotZero(t, result.OrderID, "OrderID should be set")
	})
}

func TestGetCoinMarginedTradeCommissionRate(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetCoinMarginedTradeCommissionRate(t.Context())
	require.NoError(t, err, "GetCoinMarginedTradeCommissionRate must not error")
	if mockTests {
		exp := &CoinMarginedTradeCommissionRateResponse{TakerCommissionRate: 0.0004, MakerCommissionRate: 0.00013999}
		assert.Equal(t, exp, result, "GetCoinMarginedTradeCommissionRate should decode every field")
		return
	}
	assert.NotNil(t, result, "GetCoinMarginedTradeCommissionRate should return the rates")
}

func TestGetCoinMarginedLeverage(t *testing.T) {
	t.Parallel()
	_, err := e.GetCoinMarginedLeverage(t.Context(), currency.EMPTYPAIR)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetCoinMarginedLeverage should reject an empty symbol")
	_, err = new(Exchange).GetCoinMarginedLeverage(t.Context(), currency.NewBTCUSD())
	assert.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "GetCoinMarginedLeverage should return the symbol formatting error")

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetCoinMarginedLeverage(t.Context(), currency.NewBTCUSD())
	require.NoError(t, err, "GetCoinMarginedLeverage must not error")
	require.NotNil(t, result, "GetCoinMarginedLeverage must return the leverage")
	if mockTests {
		exp := &CoinMarginedLeverageResponse{
			Symbol:               "BTC-USD",
			LongLeverage:         4,
			ShortLeverage:        8,
			MaxLongLeverage:      150,
			MaxShortLeverage:     150,
			AvailableLongVolume:  15000000,
			AvailableShortVolume: 15000000,
		}
		assert.Equal(t, exp, result, "GetCoinMarginedLeverage should decode every field")
		return
	}
	assert.Positive(t, result.MaxLongLeverage, "MaxLongLeverage should be positive")
}

func TestSetCoinMarginedLeverage(t *testing.T) {
	t.Parallel()
	_, err := e.SetCoinMarginedLeverage(t.Context(), currency.EMPTYPAIR, PositionSideLong, 4)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "SetCoinMarginedLeverage should reject an empty symbol")
	_, err = e.SetCoinMarginedLeverage(t.Context(), currency.NewBTCUSD(), "", 4)
	assert.ErrorIs(t, err, errPositionSideRequired, "SetCoinMarginedLeverage should reject an empty side")
	_, err = e.SetCoinMarginedLeverage(t.Context(), currency.NewBTCUSD(), PositionSideLong, 0)
	assert.ErrorIs(t, err, errCoinMarginedLeverageEmpty, "SetCoinMarginedLeverage should reject an empty leverage")
	_, err = new(Exchange).SetCoinMarginedLeverage(t.Context(), currency.NewBTCUSD(), PositionSideLong, 4)
	assert.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "SetCoinMarginedLeverage should return the symbol formatting error")

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	if !mockTests {
		sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)
	}
	result, err := e.SetCoinMarginedLeverage(t.Context(), currency.NewBTCUSD(), PositionSideLong, 4)
	require.NoError(t, err, "SetCoinMarginedLeverage must not error")
	require.NotNil(t, result, "SetCoinMarginedLeverage must return the leverage")
	if mockTests {
		exp := &CoinMarginedLeverageResponse{
			Symbol:               "BTC-USD",
			LongLeverage:         4,
			ShortLeverage:        8,
			MaxLongLeverage:      150,
			MaxShortLeverage:     150,
			AvailableLongVolume:  15000000,
			AvailableShortVolume: 15000000,
		}
		assert.Equal(t, exp, result, "SetCoinMarginedLeverage should decode every field")
		return
	}
	assert.Equal(t, uint64(4), result.LongLeverage, "LongLeverage should be the leverage set")
}

func TestCancelAllCoinMarginedOrders(t *testing.T) {
	t.Parallel()
	_, err := new(Exchange).CancelAllCoinMarginedOrders(t.Context(), currency.NewBTCUSD())
	assert.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "CancelAllCoinMarginedOrders should return the symbol formatting error")

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	if !mockTests {
		sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelAllCoinMarginedOrders(t.Context(), currency.NewBTCUSD())
	require.NoError(t, err, "CancelAllCoinMarginedOrders must not error")
	require.NotNil(t, result, "CancelAllCoinMarginedOrders must return the outcome")
	if mockTests {
		exp := &CancelAllCoinMarginedOrdersResponse{
			Success: []CoinMarginedOrderResponse{
				{
					Symbol:       "BTC-USD",
					OrderID:      1801610628516806656,
					Side:         SideBuy,
					PositionSide: "LONG",
					Type:         OrderTypeLimit,
					Quantity:     2,
					Price:        27173,
					Status:       "CANCELLED",
					Time:         types.Time(time.UnixMilli(1718372420802)),
					UpdateTime:   types.Time(time.UnixMilli(1718372420820)),
				},
			},
			Failed: []CoinMarginedCancelOrderFailure{
				{OrderID: 1801610628516806656, Code: 123},
				{OrderID: 1801610628516806656, Code: 123},
			},
		}
		assert.Equal(t, exp, result, "CancelAllCoinMarginedOrders should decode every field")
	}
}

func TestCloseAllCoinMarginedPositions(t *testing.T) {
	t.Parallel()
	_, err := new(Exchange).CloseAllCoinMarginedPositions(t.Context(), currency.NewBTCUSD())
	assert.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "CloseAllCoinMarginedPositions should return the symbol formatting error")

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	if !mockTests {
		sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)
	}
	result, err := e.CloseAllCoinMarginedPositions(t.Context(), currency.NewBTCUSD())
	require.NoError(t, err, "CloseAllCoinMarginedPositions must not error")
	require.NotNil(t, result, "CloseAllCoinMarginedPositions must return the outcome")
	if mockTests {
		exp := &CloseAllCoinMarginedPositionsResponse{
			Success: []NumericID{1801870087554072576},
			Failed:  []CoinMarginedClosePositionFailure{{PositionID: 12345678910111234, ErrorCode: 123, ErrorMessage: "balabala"}},
		}
		assert.Equal(t, exp, result, "CloseAllCoinMarginedPositions should decode every field")
	}
}

func TestGetCoinMarginedPositions(t *testing.T) {
	t.Parallel()
	_, err := new(Exchange).GetCoinMarginedPositions(t.Context(), currency.NewBTCUSD())
	assert.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "GetCoinMarginedPositions should return the symbol formatting error")

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetCoinMarginedPositions(t.Context(), currency.NewBTCUSD())
	require.NoError(t, err, "GetCoinMarginedPositions must not error")
	if mockTests {
		exp := []CoinMarginedPositionResponse{
			{
				Symbol:           "BTC-USD",
				PositionID:       1801165371278884900,
				PositionSide:     PositionSideLong,
				PositionAmount:   3,
				AvailableAmount:  3,
				UnrealisedProfit: -0.00010485,
				InitialMargin:    0.00110845,
				LiquidationPrice: 2024.7812708419876,
				AveragePrice:     67662,
				Leverage:         4,
				MarkPrice:        66098.9,
				RiskRate:         0.00013841,
				UpdateTime:       types.Time(time.UnixMilli(1718409600901)),
			},
			{
				Symbol:           "ETH-USD",
				PositionID:       1796163366063964200,
				PositionSide:     PositionSideLong,
				PositionAmount:   376,
				AvailableAmount:  376,
				UnrealisedProfit: -0.08051938,
				InitialMargin:    0.19994044,
				LiquidationPrice: 630.4805786791729,
				AveragePrice:     3761.12,
				Leverage:         5,
				MarkPrice:        3480.77,
				RiskRate:         0.00096807,
				UpdateTime:       types.Time(time.UnixMilli(1718409600705)),
			},
		}
		assert.Equal(t, exp, result, "GetCoinMarginedPositions should decode every field")
	}
}

func TestGetCoinMarginedAccountAssets(t *testing.T) {
	t.Parallel()
	_, err := new(Exchange).GetCoinMarginedAccountAssets(t.Context(), currency.NewBTCUSD())
	assert.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "GetCoinMarginedAccountAssets should return the symbol formatting error")

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetCoinMarginedAccountAssets(t.Context(), currency.NewBTCUSD())
	require.NoError(t, err, "GetCoinMarginedAccountAssets must not error")
	if mockTests {
		exp := []CoinMarginedAccountAssetResponse{
			{
				Asset:            currency.BTC,
				Balance:          0.14438227,
				Equity:           0.14428116,
				UnrealisedProfit: -0.0001011,
				AvailableMargin:  0.14317271,
				UsedMargin:       0.00110845,
				ShortUID:         "12345678",
			},
		}
		assert.Equal(t, exp, result, "GetCoinMarginedAccountAssets should decode every field")
	}
}

func TestGetCoinMarginedForceOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetCoinMarginedForceOrders(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer, "GetCoinMarginedForceOrders should reject a nil request")
	_, err = e.GetCoinMarginedForceOrders(t.Context(), &CoinMarginedForceOrdersRequest{StartTime: time.UnixMilli(2000), EndTime: time.UnixMilli(1000)})
	assert.ErrorIs(t, err, common.ErrStartAfterEnd, "GetCoinMarginedForceOrders should reject a start time after the end time")
	_, err = new(Exchange).GetCoinMarginedForceOrders(t.Context(), &CoinMarginedForceOrdersRequest{Symbol: currency.NewBTCUSD()})
	assert.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "GetCoinMarginedForceOrders should return the symbol formatting error")

	t.Run("sends every parameter and reads the orders", func(t *testing.T) {
		t.Parallel()
		ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method, "method should be GET")
			assert.Equal(t, "/openApi/cswap/v1/trade/forceOrders", r.URL.Path, "path should be the force orders path")
			exp := url.Values{
				"symbol":        {"BTC-USD"},
				"autoCloseType": {"ADL"},
				"startTime":     {"1702688795000"},
				"endTime":       {"1702731995000"},
				"limit":         {"100"},
				"recvWindow":    {"5000"},
			}
			assert.Equal(t, exp, coinMarginedSignedQuery(t, r), "query should hold every parameter")
			// BingX documents the reply as a list of orders with no example, so the list holds User's History Orders' example
			_, _ = io.WriteString(w, `{"code":0,"msg":"","data":[{"symbol":"PYTH-USD","orderId":1736007506620112100,"side":"SELL","positionSide":"SHORT","type":"LIMIT","origQty":"33","price":"0.3916","executedQty":"33","avgPrice":"0.3916","cumQuote":"13","stopPrice":"","profit":"0.0000","commission":"-0.002585","status":"FILLED","time":1702731418000,"updateTime":1702731470000,"clientOrderId":"","leverage":"15X","takeProfit":{"type":"TAKE_PROFIT","quantity":0,"stopPrice":0,"price":0,"workingType":""},"stopLoss":{"type":"STOP","quantity":0,"stopPrice":0,"price":0,"workingType":""},"advanceAttr":0,"positionID":0,"takeProfitEntrustPrice":0,"stopLossEntrustPrice":0,"orderType":"","workingType":"MARK_PRICE"}]}`)
		})
		result, err := ex.GetCoinMarginedForceOrders(t.Context(), &CoinMarginedForceOrdersRequest{
			Symbol:        currency.NewBTCUSD(),
			AutoCloseType: AutoCloseTypeAutoDeleveraging,
			StartTime:     time.UnixMilli(1702688795000),
			EndTime:       time.UnixMilli(1702731995000),
			Limit:         100,
		})
		require.NoError(t, err, "GetCoinMarginedForceOrders must not error")
		exp := []CoinMarginedOrderResponse{
			{
				Symbol:           "PYTH-USD",
				OrderID:          1736007506620112100,
				Side:             SideSell,
				PositionSide:     "SHORT",
				Type:             OrderTypeLimit,
				OriginalQuantity: 33,
				Price:            0.3916,
				ExecutedQuantity: 33,
				AveragePrice:     0.3916,
				CumulativeQuote:  13,
				Commission:       -0.002585,
				Status:           "FILLED",
				Time:             types.Time(time.UnixMilli(1702731418000)),
				UpdateTime:       types.Time(time.UnixMilli(1702731470000)),
				Leverage:         "15X",
				TakeProfit:       CoinMarginedTakeProfitStopLoss{Type: OrderTypeTakeProfit},
				StopLoss:         CoinMarginedTakeProfitStopLoss{Type: OrderTypeStop},
				WorkingType:      WorkingTypeMarkPrice,
			},
		}
		assert.Equal(t, exp, result, "GetCoinMarginedForceOrders should decode every field")
	})

	t.Run("force orders", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("BingX documents no example reply to record; the test server case covers decoding")
		}
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		result, err := e.GetCoinMarginedForceOrders(t.Context(), &CoinMarginedForceOrdersRequest{Symbol: currency.NewBTCUSD(), AutoCloseType: AutoCloseTypeLiquidation, Limit: 5})
		require.NoError(t, err, "GetCoinMarginedForceOrders must not error")
		assert.LessOrEqual(t, len(result), 5, "GetCoinMarginedForceOrders should return at most the limit")
	})
}

func TestGetCoinMarginedOrderTradeDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetCoinMarginedOrderTradeDetail(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer, "GetCoinMarginedOrderTradeDetail should reject a nil request")
	_, err = e.GetCoinMarginedOrderTradeDetail(t.Context(), &CoinMarginedOrderTradeDetailRequest{PageIndex: 1})
	assert.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetCoinMarginedOrderTradeDetail should reject an empty order ID")

	t.Run("sends every parameter", func(t *testing.T) {
		t.Parallel()
		ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method, "method should be GET")
			assert.Equal(t, "/openApi/cswap/v1/trade/allFillOrders", r.URL.Path, "path should be the fill orders path")
			exp := url.Values{"orderId": {"1796163365782945792"}, "pageIndex": {"2"}, "pageSize": {"50"}, "recvWindow": {"5000"}}
			assert.Equal(t, exp, coinMarginedSignedQuery(t, r), "query should hold every parameter")
			_, _ = io.WriteString(w, `{"code":0,"msg":"","data":[]}`)
		})
		result, err := ex.GetCoinMarginedOrderTradeDetail(t.Context(), &CoinMarginedOrderTradeDetailRequest{OrderID: 1796163365782945792, PageIndex: 2, PageSize: 50})
		require.NoError(t, err, "GetCoinMarginedOrderTradeDetail must not error")
		assert.Empty(t, result, "GetCoinMarginedOrderTradeDetail should return no trades")
	})

	t.Run("order the account does not hold", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("a live account's answer for an order it does not hold; the test server case covers decoding")
		}
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		result, err := e.GetCoinMarginedOrderTradeDetail(t.Context(), &CoinMarginedOrderTradeDetailRequest{OrderID: 1})
		if err != nil {
			assert.ErrorIs(t, err, order.ErrOrderNotFound, "GetCoinMarginedOrderTradeDetail should report an order the account does not hold")
			return
		}
		assert.Empty(t, result, "GetCoinMarginedOrderTradeDetail should return no trades for an order the account does not hold")
	})

	t.Run("order trades", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		if !mockTests {
			t.Skip("querying an order's trades live needs an order the account filled")
		}
		result, err := e.GetCoinMarginedOrderTradeDetail(t.Context(), &CoinMarginedOrderTradeDetailRequest{OrderID: 1796163365782945792})
		require.NoError(t, err, "GetCoinMarginedOrderTradeDetail must not error")
		exp := []CoinMarginedOrderTradeDetailResponse{
			{
				OrderID:      1796163365782945792,
				Symbol:       "ETH-USD",
				Type:         OrderTypeMarket,
				Side:         SideBuy,
				PositionSide: PositionSideLong,
				TradeID:      20789331,
				Volume:       376,
				TradePrice:   3761.12,
				Amount:       3760,
				Commission:   -0.00039988,
				Currency:     currency.ETH,
				Buyer:        true,
				TradeTime:    types.Time(time.UnixMilli(1717073692000)),
			},
		}
		assert.Equal(t, exp, result, "GetCoinMarginedOrderTradeDetail should decode every field")
	})
}

func TestCancelCoinMarginedOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelCoinMarginedOrder(t.Context(), currency.NewBTCUSD(), 0, "")
	assert.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelCoinMarginedOrder should reject an order without an ID")

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	if !mockTests {
		t.Skip("cancelling an order live needs an order the account placed")
	}
	result, err := e.CancelCoinMarginedOrder(t.Context(), currency.NewBTCUSD(), 1736011869418901234, "")
	require.NoError(t, err, "CancelCoinMarginedOrder must not error")
	exp := &CoinMarginedOrderResponse{
		Symbol:           "BTC-USDT",
		OrderID:          1736011869418901200,
		Side:             SideBuy,
		PositionSide:     "LONG",
		Type:             OrderTypeLimit,
		OriginalQuantity: 3,
		Price:            4.5081,
		Status:           "CANCELLED",
		Time:             types.Time(time.UnixMilli(1702732457867)),
		UpdateTime:       types.Time(time.UnixMilli(1702732457888)),
	}
	assert.Equal(t, exp, result, "CancelCoinMarginedOrder should decode every field")
}

func TestCoinMarginedOrderIdentityParams(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name          string
		symbol        currency.Pair
		orderID       uint64
		clientOrderID string
		exp           url.Values
		err           error
	}{
		{name: "empty symbol", orderID: 1, err: currency.ErrCurrencyPairEmpty},
		{name: "no order ID", symbol: currency.NewBTCUSD(), err: order.ErrOrderIDNotSet},
		{name: "order ID", symbol: currency.NewBTCUSD(), orderID: 1736011869418901234, exp: url.Values{"symbol": {"BTC-USD"}, "orderId": {"1736011869418901234"}}},
		{name: "client order ID", symbol: currency.NewBTCUSD(), clientOrderID: "gct-1", exp: url.Values{"symbol": {"BTC-USD"}, "clientOrderId": {"gct-1"}}},
		{name: "both IDs", symbol: currency.NewBTCUSD(), orderID: 7, clientOrderID: "gct-1", exp: url.Values{"symbol": {"BTC-USD"}, "orderId": {"7"}, "clientOrderId": {"gct-1"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			params, err := e.coinMarginedOrderIdentityParams(tc.symbol, tc.orderID, tc.clientOrderID)
			if tc.err != nil {
				assert.ErrorIs(t, err, tc.err, "coinMarginedOrderIdentityParams should return the expected error")
				return
			}
			require.NoError(t, err, "coinMarginedOrderIdentityParams must not error")
			assert.Equal(t, tc.exp, params, "coinMarginedOrderIdentityParams should return the order's identity")
		})
	}
	t.Run("unformattable symbol", func(t *testing.T) {
		t.Parallel()
		_, err := new(Exchange).coinMarginedOrderIdentityParams(currency.NewBTCUSD(), 1, "")
		assert.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "coinMarginedOrderIdentityParams should return the formatting error")
	})
}

func TestCoinMarginedOrder(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, body string
		exp        *CoinMarginedOrderResponse
		err        error
	}{
		{name: "order", body: `{"code":0,"msg":"","data":{"order":{"symbol":"BTC-USD","orderId":"1816342420721254400","positionSide":"Long","status":"Pending"}}}`, exp: &CoinMarginedOrderResponse{Symbol: "BTC-USD", OrderID: 1816342420721254400, PositionSide: "Long", Status: "Pending"}},
		{name: "no order", body: `{"code":0,"msg":"","data":{}}`, err: common.ErrNoResponse},
		{name: "API error", body: `{"code":109421,"msg":"order not exist"}`, err: order.ErrOrderNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodDelete, r.Method, "method should be the requested method")
				assert.Equal(t, "/openApi/test", r.URL.Path, "path should be the requested path")
				assert.Equal(t, url.Values{"symbol": {"BTC-USD"}, "recvWindow": {"5000"}}, coinMarginedSignedQuery(t, r), "query should hold the parameters given")
				_, _ = io.WriteString(w, tc.body)
			})
			result, err := ex.coinMarginedOrder(t.Context(), http.MethodDelete, "/openApi/test", url.Values{"symbol": {"BTC-USD"}}, coinMarginedCancelOrderRate)
			if tc.err != nil {
				assert.ErrorIs(t, err, tc.err, "coinMarginedOrder should return the expected error")
				return
			}
			require.NoError(t, err, "coinMarginedOrder must not error")
			assert.Equal(t, tc.exp, result, "coinMarginedOrder should return the order")
		})
	}
}

func TestGetCoinMarginedOpenOrders(t *testing.T) {
	t.Parallel()
	_, err := new(Exchange).GetCoinMarginedOpenOrders(t.Context(), currency.NewBTCUSD())
	assert.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "GetCoinMarginedOpenOrders should return the symbol formatting error")

	t.Run("API error", func(t *testing.T) {
		t.Parallel()
		ex := newTestServerExchange(t, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, `{"code":109400,"msg":"Invalid parameters"}`)
		})
		_, err := ex.GetCoinMarginedOpenOrders(t.Context(), currency.NewBTCUSD())
		assert.ErrorIs(t, err, errAPIResponse, "GetCoinMarginedOpenOrders should return the API error")
	})

	t.Run("open orders", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		result, err := e.GetCoinMarginedOpenOrders(t.Context(), currency.NewBTCUSD())
		require.NoError(t, err, "GetCoinMarginedOpenOrders must not error")
		if mockTests {
			exp := []CoinMarginedOrderResponse{
				{
					Symbol:           "BTC-USD",
					OrderID:          1733405587011123500,
					Side:             SideSell,
					PositionSide:     "LONG",
					Type:             OrderTypeLimit,
					OriginalQuantity: 0.003,
					Price:            44459.6,
					Status:           "PENDING",
					Time:             types.Time(time.UnixMilli(1702256915574)),
					UpdateTime:       types.Time(time.UnixMilli(1702256915610)),
					WorkingType:      WorkingTypeMarkPrice,
				},
				{
					Symbol:           "BTC-USDT",
					OrderID:          1733405587011123500,
					Side:             SideSell,
					PositionSide:     "LONG",
					Type:             OrderTypeLimit,
					OriginalQuantity: 0.003,
					Price:            44454.6,
					Status:           "PENDING",
					Time:             types.Time(time.UnixMilli(1702111071719)),
					UpdateTime:       types.Time(time.UnixMilli(1702111071735)),
					WorkingType:      WorkingTypeMarkPrice,
				},
			}
			assert.Equal(t, exp, result, "GetCoinMarginedOpenOrders should decode every field")
		}
	})
}

func TestGetCoinMarginedOrderDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetCoinMarginedOrderDetail(t.Context(), currency.NewBTCUSD(), 0, "")
	assert.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetCoinMarginedOrderDetail should reject an order without an ID")

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	if !mockTests {
		t.Skip("querying an order live needs an order the account placed")
	}
	result, err := e.GetCoinMarginedOrderDetail(t.Context(), currency.NewPair(currency.OP, currency.USD), 1736012449498123456, "")
	require.NoError(t, err, "GetCoinMarginedOrderDetail must not error")
	exp := &CoinMarginedOrderResponse{
		Symbol:           "OP-USD",
		OrderID:          1736012449498123500,
		Side:             SideSell,
		PositionSide:     "LONG",
		Type:             OrderTypeLimit,
		OriginalQuantity: 1,
		Price:            2.171,
		Status:           "PENDING",
		Time:             types.Time(time.UnixMilli(1702732596168)),
		UpdateTime:       types.Time(time.UnixMilli(1702732596188)),
		WorkingType:      WorkingTypeMarkPrice,
	}
	assert.Equal(t, exp, result, "GetCoinMarginedOrderDetail should decode every field")
}

func TestGetCoinMarginedOrderHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetCoinMarginedOrderHistory(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer, "GetCoinMarginedOrderHistory should reject a nil request")
	_, err = e.GetCoinMarginedOrderHistory(t.Context(), &CoinMarginedOrderHistoryRequest{Symbol: currency.NewBTCUSD()})
	assert.ErrorIs(t, err, errCoinMarginedLimitEmpty, "GetCoinMarginedOrderHistory should reject an empty limit")
	_, err = e.GetCoinMarginedOrderHistory(t.Context(), &CoinMarginedOrderHistoryRequest{StartTime: time.UnixMilli(2000), EndTime: time.UnixMilli(1000), Limit: 10})
	assert.ErrorIs(t, err, common.ErrStartAfterEnd, "GetCoinMarginedOrderHistory should reject a start time after the end time")
	_, err = new(Exchange).GetCoinMarginedOrderHistory(t.Context(), &CoinMarginedOrderHistoryRequest{Symbol: currency.NewBTCUSD(), Limit: 10})
	assert.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "GetCoinMarginedOrderHistory should return the symbol formatting error")

	t.Run("sends every parameter", func(t *testing.T) {
		t.Parallel()
		ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method, "method should be GET")
			assert.Equal(t, "/openApi/cswap/v1/trade/orderHistory", r.URL.Path, "path should be the order history path")
			exp := url.Values{
				"symbol":     {"PYTH-USD"},
				"orderId":    {"1736007506620112100"},
				"startTime":  {"1702688795000"},
				"endTime":    {"1702731995000"},
				"limit":      {"500"},
				"recvWindow": {"5000"},
			}
			assert.Equal(t, exp, coinMarginedSignedQuery(t, r), "query should hold every parameter")
			_, _ = io.WriteString(w, `{"code":0,"msg":"","data":{"orders":[]}}`)
		})
		result, err := ex.GetCoinMarginedOrderHistory(t.Context(), &CoinMarginedOrderHistoryRequest{
			Symbol:    currency.NewPair(currency.NewCode("PYTH"), currency.USD),
			OrderID:   1736007506620112100,
			StartTime: time.UnixMilli(1702688795000),
			EndTime:   time.UnixMilli(1702731995000),
			Limit:     500,
		})
		require.NoError(t, err, "GetCoinMarginedOrderHistory must not error")
		assert.Empty(t, result, "GetCoinMarginedOrderHistory should return no orders")
	})

	t.Run("API error", func(t *testing.T) {
		t.Parallel()
		ex := newTestServerExchange(t, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, `{"code":109400,"msg":"Invalid parameters"}`)
		})
		_, err := ex.GetCoinMarginedOrderHistory(t.Context(), &CoinMarginedOrderHistoryRequest{Limit: 500})
		assert.ErrorIs(t, err, errAPIResponse, "GetCoinMarginedOrderHistory should return the API error")
	})

	t.Run("order history", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		result, err := e.GetCoinMarginedOrderHistory(t.Context(), &CoinMarginedOrderHistoryRequest{Symbol: currency.NewBTCUSD(), Limit: 500})
		require.NoError(t, err, "GetCoinMarginedOrderHistory must not error")
		if mockTests {
			exp := []CoinMarginedOrderResponse{
				{
					Symbol:           "PYTH-USD",
					OrderID:          1736007506620112100,
					Side:             SideSell,
					PositionSide:     "SHORT",
					Type:             OrderTypeLimit,
					OriginalQuantity: 33,
					Price:            0.3916,
					ExecutedQuantity: 33,
					AveragePrice:     0.3916,
					CumulativeQuote:  13,
					Commission:       -0.002585,
					Status:           "FILLED",
					Time:             types.Time(time.UnixMilli(1702731418000)),
					UpdateTime:       types.Time(time.UnixMilli(1702731470000)),
					Leverage:         "15X",
					TakeProfit:       CoinMarginedTakeProfitStopLoss{Type: OrderTypeTakeProfit},
					StopLoss:         CoinMarginedTakeProfitStopLoss{Type: OrderTypeStop},
					WorkingType:      WorkingTypeMarkPrice,
				},
			}
			assert.Equal(t, exp, result, "GetCoinMarginedOrderHistory should decode every field")
			return
		}
		assert.LessOrEqual(t, len(result), 500, "GetCoinMarginedOrderHistory should return at most the limit")
	})
}

func TestGetCoinMarginedMarginType(t *testing.T) {
	t.Parallel()
	_, err := e.GetCoinMarginedMarginType(t.Context(), currency.EMPTYPAIR)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetCoinMarginedMarginType should reject an empty symbol")
	_, err = new(Exchange).GetCoinMarginedMarginType(t.Context(), currency.NewBTCUSD())
	assert.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "GetCoinMarginedMarginType should return the symbol formatting error")

	for _, tc := range []struct {
		name, body string
		exp        []CoinMarginedMarginTypeResponse
		err        error
	}{
		{name: "single object reply", body: `{"code":0,"msg":"","data":{"symbol":"BTC-USD","marginType":"CROSSED"}}`, exp: []CoinMarginedMarginTypeResponse{{Symbol: "BTC-USD", MarginType: MarginTypeCrossed}}},
		{name: "empty reply", body: `{"code":0,"msg":"","data":[]}`, err: common.ErrNoResponse},
		{name: "API error", body: `{"code":109400,"msg":"Invalid parameters"}`, err: errAPIResponse},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newTestServerExchange(t, func(w http.ResponseWriter, _ *http.Request) {
				_, _ = io.WriteString(w, tc.body)
			})
			result, err := ex.GetCoinMarginedMarginType(t.Context(), currency.NewBTCUSD())
			if tc.err != nil {
				assert.ErrorIs(t, err, tc.err, "GetCoinMarginedMarginType should return the expected error")
				return
			}
			require.NoError(t, err, "GetCoinMarginedMarginType must not error")
			assert.Equal(t, tc.exp, result, "GetCoinMarginedMarginType should read the reply")
		})
	}

	t.Run("margin type", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		result, err := e.GetCoinMarginedMarginType(t.Context(), currency.NewBTCUSD())
		require.NoError(t, err, "GetCoinMarginedMarginType must not error")
		if mockTests {
			exp := []CoinMarginedMarginTypeResponse{{Symbol: "BTC-USD", MarginType: MarginTypeIsolated}}
			assert.Equal(t, exp, result, "GetCoinMarginedMarginType should decode every field")
			return
		}
		assert.NotEmpty(t, result[0].MarginType, "MarginType should be set")
	})
}

func TestSetCoinMarginedMarginType(t *testing.T) {
	t.Parallel()
	err := e.SetCoinMarginedMarginType(t.Context(), currency.EMPTYPAIR, MarginTypeIsolated)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "SetCoinMarginedMarginType should reject an empty symbol")
	err = e.SetCoinMarginedMarginType(t.Context(), currency.NewBTCUSD(), "")
	assert.ErrorIs(t, err, margin.ErrInvalidMarginType, "SetCoinMarginedMarginType should reject an empty margin type")
	err = new(Exchange).SetCoinMarginedMarginType(t.Context(), currency.NewBTCUSD(), MarginTypeIsolated)
	assert.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "SetCoinMarginedMarginType should return the symbol formatting error")

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	if !mockTests {
		sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)
	}
	assert.NoError(t, e.SetCoinMarginedMarginType(t.Context(), currency.NewBTCUSD(), MarginTypeIsolated), "SetCoinMarginedMarginType should not error")
}

func TestAdjustCoinMarginedIsolatedMargin(t *testing.T) {
	t.Parallel()
	assert.ErrorIs(t, e.AdjustCoinMarginedIsolatedMargin(t.Context(), nil), common.ErrNilPointer, "AdjustCoinMarginedIsolatedMargin should reject a nil request")
	for _, tc := range []struct {
		name string
		req  *AdjustCoinMarginedIsolatedMarginRequest
		err  error
	}{
		{"empty symbol", &AdjustCoinMarginedIsolatedMarginRequest{Amount: 0.01, Type: CoinMarginedMarginAdjustmentTypeIncrease, PositionSide: PositionSideLong}, currency.ErrCurrencyPairEmpty},
		{"empty amount", &AdjustCoinMarginedIsolatedMarginRequest{Symbol: currency.NewBTCUSD(), Type: CoinMarginedMarginAdjustmentTypeIncrease, PositionSide: PositionSideLong}, order.ErrAmountIsInvalid},
		{"negative amount", &AdjustCoinMarginedIsolatedMarginRequest{Symbol: currency.NewBTCUSD(), Amount: -0.01, Type: CoinMarginedMarginAdjustmentTypeDecrease, PositionSide: PositionSideLong}, order.ErrAmountIsInvalid},
		{"empty type", &AdjustCoinMarginedIsolatedMarginRequest{Symbol: currency.NewBTCUSD(), Amount: 0.01, PositionSide: PositionSideLong}, errCoinMarginedMarginAdjustmentTypeEmpty},
		{"empty position side", &AdjustCoinMarginedIsolatedMarginRequest{Symbol: currency.NewBTCUSD(), Amount: 0.01, Type: CoinMarginedMarginAdjustmentTypeIncrease}, errPositionSideRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIs(t, e.AdjustCoinMarginedIsolatedMargin(t.Context(), tc.req), tc.err, "AdjustCoinMarginedIsolatedMargin should reject the request")
		})
	}
	req := &AdjustCoinMarginedIsolatedMarginRequest{Symbol: currency.NewBTCUSD(), Amount: 0.01, Type: CoinMarginedMarginAdjustmentTypeIncrease, PositionSide: PositionSideLong}
	assert.ErrorIs(t, new(Exchange).AdjustCoinMarginedIsolatedMargin(t.Context(), req), currency.ErrPairManagerNotInitialised, "AdjustCoinMarginedIsolatedMargin should return the symbol formatting error")

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	if mockTests {
		assert.NoError(t, e.AdjustCoinMarginedIsolatedMargin(t.Context(), req), "AdjustCoinMarginedIsolatedMargin should not error")
		return
	}
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)
	ensureTradablePairs(t)
	err := e.AdjustCoinMarginedIsolatedMargin(t.Context(), &AdjustCoinMarginedIsolatedMarginRequest{Symbol: coinMarginedTradablePair, Amount: 0.0001, Type: CoinMarginedMarginAdjustmentTypeIncrease, PositionSide: PositionSideLong})
	if err != nil {
		requireAPIErrorCode(t, err, livePositionStateCodes...)
	}
}
