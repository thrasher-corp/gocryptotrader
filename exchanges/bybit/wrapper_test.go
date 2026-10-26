package bybit

import (
	"context"
	"fmt"
	"io"
	"maps"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"uuid"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/deposit"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	testsubs "github.com/thrasher-corp/gocryptotrader/internal/testing/subscriptions"
	mockws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
	"github.com/thrasher-corp/gocryptotrader/types"
	"github.com/thrasher-corp/gocryptotrader/types/decimal"
)

func TestDeriveSubmitOrderArguments(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		submit *order.Submit
		exp    *PlaceOrderRequest
		err    error
	}{
		{err: order.ErrSubmissionIsNil},
		{
			submit: &order.Submit{
				Exchange:  e.GetName(),
				Pair:      currency.NewBTCUSDT(),
				AssetType: asset.Binary,
				Side:      order.Buy,
				Type:      order.Market,
				Amount:    1,
			},
			err: asset.ErrNotSupported,
		},
		{
			submit: &order.Submit{
				Exchange:  e.GetName(),
				Pair:      currency.NewBTCUSDT(),
				AssetType: asset.USDCMarginedFutures,
				Side:      order.Buy,
				Type:      order.Market,
				Amount:    1,
			},
			exp: &PlaceOrderRequest{
				Category:    getCategoryName(asset.USDCMarginedFutures),
				Symbol:      currency.NewBTCUSDT().Format(currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter}).String(),
				Side:        sideBuy,
				OrderType:   orderTypeToString(order.Market),
				Quantity:    1,
				TimeInForce: "IOC",
			},
		},
		{
			submit: &order.Submit{
				Exchange:  e.GetName(),
				Pair:      currency.NewBTCUSDT(),
				AssetType: asset.USDCMarginedFutures,
				Side:      order.Buy,
				Type:      order.Market,
				Amount:    1,
				RiskManagementModes: order.RiskManagementModes{
					TakeProfit: order.RiskManagement{Price: 100},
					StopLoss:   order.RiskManagement{Price: 200},
				},
			},
			exp: &PlaceOrderRequest{
				Category:    getCategoryName(asset.USDCMarginedFutures),
				Symbol:      currency.NewBTCUSDT().Format(currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter}).String(),
				Side:        sideBuy,
				OrderType:   orderTypeToString(order.Market),
				Quantity:    1,
				TimeInForce: "IOC",
				TakeProfit:  100,
				TPTriggerBy: "LastPrice",
				StopLoss:    200,
				SLTriggerBy: "LastPrice",
			},
		},
		{
			submit: &order.Submit{
				Exchange:     e.GetName(),
				Pair:         currency.NewBTCUSDT(),
				AssetType:    asset.USDCMarginedFutures,
				Side:         order.Sell,
				Type:         order.Limit,
				Amount:       1,
				Price:        5000,
				TriggerPrice: 150,
				TimeInForce:  order.FillOrKill,
			},
			exp: &PlaceOrderRequest{
				Category:         getCategoryName(asset.USDCMarginedFutures),
				Symbol:           currency.NewBTCUSDT().Format(currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter}).String(),
				Side:             sideSell,
				OrderType:        orderTypeToString(order.Limit),
				Quantity:         1,
				TimeInForce:      "FOK",
				TriggerPrice:     150,
				TriggerDirection: 2,
				Price:            5000,
				TriggerBy:        "LastPrice",
			},
		},
		{
			submit: &order.Submit{
				Exchange:     e.GetName(),
				Pair:         currency.NewBTCUSDT(),
				AssetType:    asset.USDCMarginedFutures,
				Side:         order.Sell,
				Type:         order.Limit,
				Amount:       1,
				Price:        5000,
				TriggerPrice: 150,
				TimeInForce:  order.PostOnly,
			},
			exp: &PlaceOrderRequest{
				Category:         getCategoryName(asset.USDCMarginedFutures),
				Symbol:           currency.NewBTCUSDT().Format(currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter}).String(),
				Side:             sideSell,
				OrderType:        orderTypeToString(order.Limit),
				Quantity:         1,
				TimeInForce:      "PostOnly",
				TriggerPrice:     150,
				TriggerDirection: 2,
				Price:            5000,
				TriggerBy:        "LastPrice",
			},
		},
		{
			submit: &order.Submit{
				Exchange:     e.GetName(),
				Pair:         currency.NewBTCUSDT(),
				AssetType:    asset.Spot,
				Side:         order.Sell,
				Type:         order.Limit,
				Amount:       1,
				Price:        5000,
				TriggerPrice: 150,
				TimeInForce:  order.ImmediateOrCancel,
			},
			exp: &PlaceOrderRequest{
				Category:     getCategoryName(asset.Spot),
				Symbol:       currency.NewBTCUSDT().Format(currency.PairFormat{Uppercase: true}).String(),
				Side:         sideSell,
				OrderType:    orderTypeToString(order.Limit),
				Quantity:     1,
				TimeInForce:  "IOC",
				TriggerPrice: 150,
				Price:        5000,
				OrderFilter:  "tpslOrder",
				TriggerBy:    "LastPrice",
			},
		},
		{
			// Spot market buys are sized in the quote currency, which Bybit takes by default
			submit: &order.Submit{
				Exchange:    e.GetName(),
				Pair:        currency.NewBTCUSDT(),
				AssetType:   asset.Spot,
				Side:        order.Buy,
				Type:        order.Market,
				Amount:      0.001,
				QuoteAmount: 25,
			},
			exp: &PlaceOrderRequest{
				Category:    cSpot,
				Symbol:      "BTCUSDT",
				Side:        sideBuy,
				OrderType:   "Market",
				Quantity:    25,
				TimeInForce: "IOC",
			},
		},
		{
			// A spot market buy with a trigger price is sized by value like any spot market buy, so it sends its quote
			// amount without a market unit, unlike a stop market buy
			submit: &order.Submit{
				Exchange:     e.GetName(),
				Pair:         currency.NewBTCUSDT(),
				AssetType:    asset.Spot,
				Side:         order.Buy,
				Type:         order.Market,
				Amount:       0.001,
				QuoteAmount:  25,
				TriggerPrice: 90000,
			},
			exp: &PlaceOrderRequest{
				Category:     cSpot,
				Symbol:       "BTCUSDT",
				Side:         sideBuy,
				OrderType:    "Market",
				Quantity:     25,
				TimeInForce:  "IOC",
				TriggerPrice: 90000,
				TriggerBy:    "LastPrice",
				OrderFilter:  "tpslOrder",
			},
		},
		{
			// A spot market sell is sized in the base currency whatever quote amount it carries
			submit: &order.Submit{
				Exchange:    e.GetName(),
				Pair:        currency.NewBTCUSDT(),
				AssetType:   asset.Spot,
				Side:        order.Sell,
				Type:        order.Market,
				Amount:      0.001,
				QuoteAmount: 25,
			},
			exp: &PlaceOrderRequest{
				Category:    cSpot,
				Symbol:      "BTCUSDT",
				Side:        sideSell,
				OrderType:   "Market",
				Quantity:    0.001,
				TimeInForce: "IOC",
			},
		},
		{
			submit: &order.Submit{
				Exchange:   e.GetName(),
				Pair:       currency.NewBTCUSDT(),
				AssetType:  asset.Spot,
				Side:       order.Sell,
				Type:       order.Limit,
				Amount:     0.5,
				Price:      90000,
				AutoBorrow: true,
			},
			exp: &PlaceOrderRequest{
				Category:    cSpot,
				Symbol:      "BTCUSDT",
				Side:        sideSell,
				OrderType:   "Limit",
				Quantity:    0.5,
				Price:       90000,
				TimeInForce: "GTC",
				IsLeverage:  true,
			},
		},
		{
			submit: &order.Submit{
				Exchange:  e.GetName(),
				Pair:      currency.NewBTCUSDT(),
				AssetType: asset.USDTMarginedFutures,
				Side:      order.Buy,
				Type:      order.Limit,
				Amount:    1,
				Price:     80000,
				RiskManagementModes: order.RiskManagementModes{
					TakeProfit: order.RiskManagement{Price: 90000, TriggerPriceType: order.MarkPrice, OrderType: order.Limit, LimitPrice: 89990},
					StopLoss:   order.RiskManagement{Price: 75000, TriggerPriceType: order.IndexPrice, OrderType: order.Market},
				},
			},
			exp: &PlaceOrderRequest{
				Category:     cLinear,
				Symbol:       "BTCUSDT",
				Side:         sideBuy,
				OrderType:    "Limit",
				Quantity:     1,
				Price:        80000,
				TimeInForce:  "GTC",
				TakeProfit:   90000,
				TPTriggerBy:  "MarkPrice",
				TPOrderType:  "Limit",
				TPLimitPrice: 89990,
				StopLoss:     75000,
				SLTriggerBy:  "IndexPrice",
				SLOrderType:  "Market",
				TPSLMode:     "Partial",
			},
		},
		{
			// Bybit places stop orders as market or limit orders with a trigger price
			submit: &order.Submit{
				Exchange:     e.GetName(),
				Pair:         currency.NewBTCUSDT(),
				AssetType:    asset.USDTMarginedFutures,
				Side:         order.Sell,
				Type:         order.StopLimit,
				Amount:       1,
				Price:        79000,
				TriggerPrice: 79500,
			},
			exp: &PlaceOrderRequest{
				Category:         cLinear,
				Symbol:           "BTCUSDT",
				Side:             sideSell,
				OrderType:        "Limit",
				Quantity:         1,
				Price:            79000,
				TimeInForce:      "GTC",
				TriggerPrice:     79500,
				TriggerDirection: 2,
				TriggerBy:        "LastPrice",
			},
		},
		{
			// A futures conditional order says which way the price must cross its trigger, whatever its side
			submit: &order.Submit{
				Exchange:      e.GetName(),
				Pair:          currency.NewBTCUSDT(),
				AssetType:     asset.USDTMarginedFutures,
				Side:          order.Sell,
				Type:          order.StopMarket,
				Amount:        1,
				TriggerPrice:  90000,
				StopDirection: order.StopUp,
			},
			exp: &PlaceOrderRequest{
				Category:         cLinear,
				Symbol:           "BTCUSDT",
				Side:             sideSell,
				OrderType:        "Market",
				Quantity:         1,
				TimeInForce:      "IOC",
				TriggerPrice:     90000,
				TriggerDirection: 1,
				TriggerBy:        "LastPrice",
			},
		},
		{
			submit: &order.Submit{
				Exchange:     e.GetName(),
				Pair:         currency.NewBTCUSDT(),
				AssetType:    asset.USDTMarginedFutures,
				Side:         order.Buy,
				Type:         order.StopMarket,
				Amount:       1,
				TriggerPrice: 70000,
			},
			exp: &PlaceOrderRequest{
				Category:         cLinear,
				Symbol:           "BTCUSDT",
				Side:             sideBuy,
				OrderType:        "Market",
				Quantity:         1,
				TimeInForce:      "IOC",
				TriggerPrice:     70000,
				TriggerDirection: 2,
				TriggerBy:        "LastPrice",
			},
		},
		{
			// A spot stop market buy is sized in the base currency, which Bybit takes only when the order says so
			submit: &order.Submit{
				Exchange:     e.GetName(),
				Pair:         currency.NewBTCUSDT(),
				AssetType:    asset.Spot,
				Side:         order.Buy,
				Type:         order.StopMarket,
				Amount:       0.01,
				TriggerPrice: 90000,
			},
			exp: &PlaceOrderRequest{
				Category:     getCategoryName(asset.Spot),
				Symbol:       "BTCUSDT",
				Side:         sideBuy,
				OrderType:    "Market",
				Quantity:     0.01,
				MarketUnit:   "baseCoin",
				TimeInForce:  "IOC",
				TriggerPrice: 90000,
				TriggerBy:    "LastPrice",
				OrderFilter:  "tpslOrder",
			},
		},
		{
			// A spot stop order is a stop market order, sized in the base currency too
			submit: &order.Submit{Exchange: e.GetName(), Pair: currency.NewBTCUSDT(), AssetType: asset.Spot, Side: order.Buy, Type: order.Stop, Amount: 0.01, TriggerPrice: 90000},
			exp: &PlaceOrderRequest{
				Category: getCategoryName(asset.Spot), Symbol: "BTCUSDT", Side: sideBuy, OrderType: "Market", Quantity: 0.01, MarketUnit: "baseCoin",
				TimeInForce: "IOC", TriggerPrice: 90000, TriggerBy: "LastPrice", OrderFilter: "tpslOrder",
			},
		},
		{
			submit: &order.Submit{Exchange: e.GetName(), Pair: currency.NewBTCUSDT(), AssetType: asset.Spot, Side: order.Sell, Type: order.StopMarket, Amount: 0.01, TriggerPrice: 80000},
			exp: &PlaceOrderRequest{
				Category: getCategoryName(asset.Spot), Symbol: "BTCUSDT", Side: sideSell, OrderType: "Market", Quantity: 0.01, MarketUnit: "baseCoin",
				TimeInForce: "IOC", TriggerPrice: 80000, TriggerBy: "LastPrice", OrderFilter: "tpslOrder",
			},
		},
		{
			// Coin margined conditional orders say which way the price must cross their trigger, as other futures do
			submit: &order.Submit{Exchange: e.GetName(), Pair: currency.NewPair(currency.ADA, currency.USD), AssetType: asset.CoinMarginedFutures, Side: order.Sell, Type: order.StopMarket, Amount: 100, TriggerPrice: 0.6, StopDirection: order.StopUp},
			exp: &PlaceOrderRequest{
				Category: cInverse, Symbol: "ADAUSD", Side: sideSell, OrderType: "Market", Quantity: 100, TimeInForce: "IOC",
				TriggerPrice: 0.6, TriggerDirection: 1, TriggerBy: "LastPrice",
			},
		},
		{
			// A short futures order sells, a reduce only one may only reduce the position, and only spot orders borrow
			submit: &order.Submit{Exchange: e.GetName(), Pair: currency.NewBTCUSDT(), AssetType: asset.USDTMarginedFutures, Side: order.Short, Type: order.Limit, Amount: 1, Price: 80000, ReduceOnly: true, AutoBorrow: true},
			exp:    &PlaceOrderRequest{Category: cLinear, Symbol: "BTCUSDT", Side: sideSell, OrderType: "Limit", Quantity: 1, Price: 80000, TimeInForce: "GTC", ReduceOnly: true},
		},
		{
			// A limit stop loss on its own needs partial mode as a limit take profit does
			submit: &order.Submit{
				Exchange: e.GetName(), Pair: currency.NewBTCUSDT(), AssetType: asset.USDTMarginedFutures, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 80000,
				RiskManagementModes: order.RiskManagementModes{StopLoss: order.RiskManagement{Price: 75000, OrderType: order.Limit, LimitPrice: 74990}},
			},
			exp: &PlaceOrderRequest{
				Category: cLinear, Symbol: "BTCUSDT", Side: sideBuy, OrderType: "Limit", Quantity: 1, Price: 80000, TimeInForce: "GTC",
				StopLoss: 75000, SLTriggerBy: "LastPrice", SLOrderType: "Limit", SLLimitPrice: 74990, TPSLMode: "Partial",
			},
		},
		{
			// Spot limit orders take limit take profits and stop losses without a TP/SL mode, which only futures have
			submit: &order.Submit{
				Exchange: e.GetName(), Pair: currency.NewBTCUSDT(), AssetType: asset.Spot, Side: order.Buy, Type: order.Limit, Amount: 0.01, Price: 28000,
				RiskManagementModes: order.RiskManagementModes{
					TakeProfit: order.RiskManagement{Price: 35000, OrderType: order.Limit, LimitPrice: 36000},
					StopLoss:   order.RiskManagement{Price: 27000, OrderType: order.Limit, LimitPrice: 27500},
				},
			},
			exp: &PlaceOrderRequest{
				Category: cSpot, Symbol: "BTCUSDT", Side: sideBuy, OrderType: "Limit", Quantity: 0.01, Price: 28000, TimeInForce: "GTC",
				TakeProfit: 35000, TPTriggerBy: "LastPrice", TPOrderType: "Limit", TPLimitPrice: 36000,
				StopLoss: 27000, SLTriggerBy: "LastPrice", SLOrderType: "Limit", SLLimitPrice: 27500,
			},
		},
		{
			submit: &order.Submit{Exchange: e.GetName(), Pair: currency.NewBTCUSDT(), AssetType: asset.USDTMarginedFutures, Side: order.Sell, Type: order.StopMarket, Amount: 1},
			err:    errTriggerPriceRequired,
		},
		{
			submit: &order.Submit{Exchange: e.GetName(), Pair: currency.NewBTCUSDT(), AssetType: asset.USDTMarginedFutures, Side: order.Sell, Type: order.TrailingStop, Amount: 1, TriggerPrice: 79500},
			err:    order.ErrUnsupportedOrderType,
		},
		{
			submit: &order.Submit{
				Exchange:  e.GetName(),
				Pair:      currency.NewBTCUSDT(),
				AssetType: asset.USDTMarginedFutures,
				Side:      order.Buy,
				Type:      order.Market,
				Amount:    1,
				RiskManagementModes: order.RiskManagementModes{
					TakeProfit: order.RiskManagement{Price: 90000, OrderType: order.StopLimit},
				},
			},
			err: order.ErrUnsupportedOrderType,
		},
		{
			// Bybit triggers a stop loss as a market or limit order only
			submit: &order.Submit{
				Exchange:  e.GetName(),
				Pair:      currency.NewBTCUSDT(),
				AssetType: asset.USDTMarginedFutures,
				Side:      order.Buy,
				Type:      order.Market,
				Amount:    1,
				RiskManagementModes: order.RiskManagementModes{
					StopLoss: order.RiskManagement{Price: 75000, OrderType: order.TrailingStop},
				},
			},
			err: order.ErrUnsupportedOrderType,
		},
	} {
		got, err := e.deriveSubmitOrderArguments(tc.submit)
		if tc.err != nil {
			require.ErrorIs(t, err, tc.err, "deriveSubmitOrderArguments must return the expected error")
			continue
		}
		require.NoError(t, err, "deriveSubmitOrderArguments must not error")
		assert.Equal(t, tc.exp, got, "deriveSubmitOrderArguments should derive the expected request")
	}

	// A conditional order takes the times in force of the order type Bybit places it as
	for _, tc := range []struct {
		orderType order.Type
		market    bool
	}{
		{orderType: order.Limit},
		{orderType: order.StopLimit},
		{orderType: order.Market, market: true},
		{orderType: order.Stop, market: true},
		{orderType: order.StopMarket, market: true},
	} {
		for _, tif := range timeInForceTestCases {
			s := &order.Submit{Exchange: e.GetName(), Pair: currency.NewBTCUSDT(), AssetType: asset.USDTMarginedFutures, Side: order.Buy, Type: tc.orderType, Amount: 1, Price: 80000, TimeInForce: tif.timeInForce}
			if tc.orderType.Is(order.Stop) {
				s.TriggerPrice = 79000
			}
			exp := tif.limit
			if tc.market {
				exp = tif.market
			}
			got, err := e.deriveSubmitOrderArguments(s)
			if exp == "" {
				assert.ErrorIsf(t, err, order.ErrUnsupportedTimeInForce, "deriveSubmitOrderArguments should refuse a %s order with time in force %q", tc.orderType, tif.timeInForce)
				continue
			}
			require.NoErrorf(t, err, "deriveSubmitOrderArguments must not error for a %s order with time in force %q", tc.orderType, tif.timeInForce)
			assert.Equalf(t, exp, got.TimeInForce, "deriveSubmitOrderArguments should send Bybit's time in force for a %s order with time in force %q", tc.orderType, tif.timeInForce)
		}
	}

	// Bybit requires a client order ID on every options order, so one is generated when none is given
	option := currency.NewPairWithDelimiter("BTC", "26NOV24-92000-C", "-")
	got, err := e.deriveSubmitOrderArguments(&order.Submit{Exchange: e.GetName(), Pair: option, AssetType: asset.Options, Side: order.Buy, Type: order.Limit, Amount: 0.01, Price: 500})
	require.NoError(t, err, "deriveSubmitOrderArguments must not error for an options order")
	_, err = uuid.Parse(got.OrderLinkID)
	assert.NoError(t, err, "deriveSubmitOrderArguments should generate a UUID client order ID for an options order")
	// Bybit requires each options order's client order ID to be unique
	again, err := e.deriveSubmitOrderArguments(&order.Submit{Exchange: e.GetName(), Pair: option, AssetType: asset.Options, Side: order.Buy, Type: order.Limit, Amount: 0.01, Price: 500})
	require.NoError(t, err, "deriveSubmitOrderArguments must not error for an options order")
	assert.NotEqual(t, got.OrderLinkID, again.OrderLinkID, "deriveSubmitOrderArguments should generate a new client order ID for each options order")
	got, err = e.deriveSubmitOrderArguments(&order.Submit{Exchange: e.GetName(), Pair: option, AssetType: asset.Options, Side: order.Buy, Type: order.Limit, Amount: 0.01, Price: 500, ClientOrderID: "option-buy-001"})
	require.NoError(t, err, "deriveSubmitOrderArguments must not error for an options order")
	assert.Equal(t, "option-buy-001", got.OrderLinkID, "deriveSubmitOrderArguments should keep a given client order ID")
}

func TestRiskOrderType(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		orderType order.Type
		exp       string
	}{
		// No order type sends none, which leaves Bybit's default of a market order
		{orderType: order.UnknownType},
		{orderType: order.Limit, exp: "Limit"},
		{orderType: order.Market, exp: "Market"},
	} {
		got, err := riskOrderType(tc.orderType)
		require.NoErrorf(t, err, "riskOrderType must not error for %s", tc.orderType)
		assert.Equalf(t, tc.exp, got, "riskOrderType should return Bybit's name for %s", tc.orderType)
	}
	// Bybit triggers a take profit or stop loss as a market or limit order only
	for _, orderType := range []order.Type{order.StopLimit, order.TrailingStop} {
		_, err := riskOrderType(orderType)
		assert.ErrorIsf(t, err, order.ErrUnsupportedOrderType, "riskOrderType should reject %s", orderType)
	}
}

func TestOrderTypeToString(t *testing.T) {
	t.Parallel()
	for orderType, exp := range map[order.Type]string{
		order.Limit:  "Limit",
		order.Market: "Market",
		// Types Bybit has no name for keep GCT's
		order.StopLimit: "STOP LIMIT",
	} {
		assert.Equalf(t, exp, orderTypeToString(orderType), "orderTypeToString should name %s", orderType)
	}
}

func TestBybitOrderType(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		orderType   order.Type
		exp         string
		conditional bool
	}{
		{orderType: order.Limit, exp: "Limit"},
		{orderType: order.Market, exp: "Market"},
		// Bybit places a stop order as a market or limit order with a trigger price
		{orderType: order.StopLimit, exp: "Limit", conditional: true},
		{orderType: order.Stop, exp: "Market", conditional: true},
		{orderType: order.StopMarket, exp: "Market", conditional: true},
	} {
		got, conditional, err := bybitOrderType(tc.orderType)
		require.NoErrorf(t, err, "bybitOrderType must not error for %s", tc.orderType)
		assert.Equalf(t, tc.exp, got, "bybitOrderType should return Bybit's order type for %s", tc.orderType)
		assert.Equalf(t, tc.conditional, conditional, "bybitOrderType should report whether %s is conditional", tc.orderType)
	}
	// An order needs a type, and Place Order takes no trailing stop, which Bybit sets on a position instead
	for _, orderType := range []order.Type{order.UnknownType, order.TrailingStop} {
		_, _, err := bybitOrderType(orderType)
		assert.ErrorIsf(t, err, order.ErrUnsupportedOrderType, "bybitOrderType should reject %s", orderType)
	}
}

// timeInForceTestCases holds each valid time in force with the name Bybit takes it by for a limit and for a market
// order, or none where Bybit cannot honour it: Bybit takes GTC, PostOnly, IOC or FOK for a limit order and executes
// every market order as IOC
var timeInForceTestCases = []struct {
	timeInForce   order.TimeInForce
	limit, market string
}{
	{timeInForce: order.UnknownTIF, limit: "GTC", market: "IOC"},
	{timeInForce: order.GoodTillCancel, limit: "GTC"},
	{timeInForce: order.PostOnly, limit: "PostOnly"},
	{timeInForce: order.GoodTillCancel | order.PostOnly, limit: "PostOnly"},
	{timeInForce: order.ImmediateOrCancel, limit: "IOC", market: "IOC"},
	{timeInForce: order.FillOrKill, limit: "FOK"},
	{timeInForce: order.GoodTillDay},
	{timeInForce: order.GoodTillTime},
	{timeInForce: order.GoodTillCrossing},
	{timeInForce: order.StopOrReduce},
	{timeInForce: order.GoodTillCancel | order.GoodTillDay},
	{timeInForce: order.GoodTillCancel | order.StopOrReduce},
	{timeInForce: order.GoodTillDay | order.PostOnly},
	{timeInForce: order.GoodTillTime | order.PostOnly},
	{timeInForce: order.GoodTillCrossing | order.PostOnly},
	{timeInForce: order.StopOrReduce | order.PostOnly},
	{timeInForce: order.GoodTillCancel | order.GoodTillCrossing | order.PostOnly},
}

func TestBybitTimeInForce(t *testing.T) {
	t.Parallel()
	for _, tc := range timeInForceTestCases {
		for _, o := range []struct {
			kind   string
			market bool
			exp    string
		}{{kind: "limit", exp: tc.limit}, {kind: "market", market: true, exp: tc.market}} {
			got, err := bybitTimeInForce(tc.timeInForce, o.market)
			if o.exp == "" {
				assert.ErrorIsf(t, err, order.ErrUnsupportedTimeInForce, "bybitTimeInForce should refuse a %s order with time in force %q", o.kind, tc.timeInForce)
				continue
			}
			require.NoErrorf(t, err, "bybitTimeInForce must not error for a %s order with time in force %q", o.kind, tc.timeInForce)
			assert.Equalf(t, o.exp, got, "bybitTimeInForce should return Bybit's time in force for a %s order with time in force %q", o.kind, tc.timeInForce)
		}
	}
	// Combinations that order.Submit's validation already refuses
	for _, tif := range []order.TimeInForce{order.ImmediateOrCancel | order.FillOrKill, order.ImmediateOrCancel | order.GoodTillCancel, order.FillOrKill | order.PostOnly} {
		for _, market := range []bool{false, true} {
			_, err := bybitTimeInForce(tif, market)
			assert.ErrorIsf(t, err, order.ErrUnsupportedTimeInForce, "bybitTimeInForce should refuse time in force %q, market %t", tif, market)
		}
	}
}

func TestDeriveAmendOrderArguments(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		action *order.Modify
		exp    *AmendOrderRequest
		err    error
	}{
		{err: order.ErrModifyOrderIsNil},
		{
			action: &order.Modify{
				OrderID:   "69420",
				Pair:      currency.NewBTCUSDT(),
				AssetType: asset.Binary,
			},
			err: asset.ErrNotSupported,
		},
		{
			action: &order.Modify{
				OrderID:   "69420",
				Pair:      currency.NewBTCUSDT(),
				AssetType: asset.USDCMarginedFutures,
			},
			// Amending neither trigger nor take profit nor stop loss leaves their trigger types alone
			exp: &AmendOrderRequest{
				Category: getCategoryName(asset.USDCMarginedFutures),
				Symbol:   currency.NewBTCUSDT().Format(currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter}).String(),
				OrderID:  "69420",
			},
		},
		{
			action: &order.Modify{
				ClientOrderID:    "linear-tp-001",
				Pair:             currency.NewBTCUSDT(),
				AssetType:        asset.USDTMarginedFutures,
				Amount:           2,
				Price:            81000,
				TriggerPrice:     80500,
				TriggerPriceType: order.IndexPrice,
				RiskManagementModes: order.RiskManagementModes{
					TakeProfit: order.RiskManagement{Price: 90000, TriggerPriceType: order.MarkPrice, OrderType: order.Limit, LimitPrice: 89990},
					StopLoss:   order.RiskManagement{Price: 75000, LimitPrice: 74990},
				},
			},
			exp: &AmendOrderRequest{
				Category:     cLinear,
				Symbol:       "BTCUSDT",
				OrderLinkID:  "linear-tp-001",
				Quantity:     2,
				Price:        81000,
				TriggerPrice: 80500,
				TriggerBy:    "IndexPrice",
				TakeProfit:   new(90000.0),
				TPTriggerBy:  "MarkPrice",
				TPLimitPrice: 89990,
				StopLoss:     new(75000.0),
				SLTriggerBy:  "LastPrice",
				SLLimitPrice: 74990,
			},
		},
		{
			action: &order.Modify{
				OrderID:             "69420",
				Pair:                currency.NewBTCUSDT(),
				AssetType:           asset.USDTMarginedFutures,
				RiskManagementModes: order.RiskManagementModes{TakeProfit: order.RiskManagement{LimitPrice: 89990}},
			},
			// A partial TP/SL order's limit prices are amended without their unchanged trigger prices or types, each on
			// its own
			exp: &AmendOrderRequest{Category: cLinear, Symbol: "BTCUSDT", OrderID: "69420", TPLimitPrice: 89990},
		},
		{
			action: &order.Modify{
				OrderID:             "69420",
				Pair:                currency.NewBTCUSDT(),
				AssetType:           asset.USDTMarginedFutures,
				RiskManagementModes: order.RiskManagementModes{StopLoss: order.RiskManagement{LimitPrice: 74990}},
			},
			exp: &AmendOrderRequest{Category: cLinear, Symbol: "BTCUSDT", OrderID: "69420", SLLimitPrice: 74990},
		},
	} {
		got, err := e.deriveAmendOrderArguments(tc.action)
		if tc.err != nil {
			require.ErrorIs(t, err, tc.err, "deriveAmendOrderArguments must return the expected error")
			continue
		}
		require.NoError(t, err, "deriveAmendOrderArguments must not error")
		assert.Equal(t, tc.exp, got, "deriveAmendOrderArguments should derive the expected request")
	}
}

func TestOrderIDRequired(t *testing.T) {
	t.Parallel()
	// Bybit names an order by its order ID or client order ID, and takes either or both
	for _, c := range []*order.Cancel{
		{OrderID: "1321003749386327552"},
		{ClientOrderID: "spot-test-01"},
		{OrderID: "1321003749386327552", ClientOrderID: "spot-test-01"},
	} {
		assert.NoErrorf(t, orderIDRequired(c).Check(), "orderIDRequired should accept order ID %q and client order ID %q", c.OrderID, c.ClientOrderID)
	}
	err := orderIDRequired(&order.Cancel{Pair: currency.NewBTCUSDT(), AssetType: asset.Spot}).Check()
	assert.ErrorIs(t, err, order.ErrOrderIDNotSet, "orderIDRequired should reject a cancellation that names no order")
}

func TestDeriveCancelOrderArguments(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		action *order.Cancel
		exp    *CancelOrderRequest
		err    error
	}{
		{err: order.ErrCancelOrderIsNil},
		{
			action: &order.Cancel{
				OrderID:   "69420",
				Pair:      currency.NewBTCUSDT(),
				AssetType: asset.Binary,
			},
			err: asset.ErrNotSupported,
		},
		{
			action: &order.Cancel{
				OrderID:   "69420",
				Pair:      currency.NewBTCUSDT(),
				AssetType: asset.USDCMarginedFutures,
			},
			exp: &CancelOrderRequest{
				Category: getCategoryName(asset.USDCMarginedFutures),
				Symbol:   currency.NewBTCUSDT().Format(currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter}).String(),
				OrderID:  "69420",
			},
		},
		{
			action: &order.Cancel{Pair: currency.NewBTCUSDT(), AssetType: asset.Spot},
			err:    order.ErrOrderIDNotSet,
		},
		{
			action: &order.Cancel{OrderID: "69420", AssetType: asset.Spot},
			err:    order.ErrPairIsEmpty,
		},
		{
			// SubmitOrder places a spot stop order as a TP/SL order, which Bybit cancels only when told to look there
			action: &order.Cancel{OrderID: "69420", Pair: currency.NewBTCUSDT(), AssetType: asset.Spot, Type: order.StopLimit},
			exp: &CancelOrderRequest{
				Category:    cSpot,
				Symbol:      "BTCUSDT",
				OrderID:     "69420",
				OrderFilter: "tpslOrder",
			},
		},
		{
			action: &order.Cancel{OrderID: "69420", Pair: currency.NewBTCUSDT(), AssetType: asset.Spot, Type: order.StopMarket},
			exp:    &CancelOrderRequest{Category: cSpot, Symbol: "BTCUSDT", OrderID: "69420", OrderFilter: "tpslOrder"},
		},
		{
			action: &order.Cancel{OrderID: "69420", Pair: currency.NewBTCUSDT(), AssetType: asset.Spot, Type: order.Stop},
			exp:    &CancelOrderRequest{Category: cSpot, Symbol: "BTCUSDT", OrderID: "69420", OrderFilter: "tpslOrder"},
		},
		{
			action: &order.Cancel{OrderID: "69420", Pair: currency.NewBTCUSDT(), AssetType: asset.USDTMarginedFutures, Type: order.StopMarket},
			exp: &CancelOrderRequest{
				Category: cLinear,
				Symbol:   "BTCUSDT",
				OrderID:  "69420",
			},
		},
		{
			// Bybit cancels by client order ID alone
			action: &order.Cancel{ClientOrderID: "spot-buy-001", Pair: currency.NewBTCUSDT(), AssetType: asset.Spot},
			exp: &CancelOrderRequest{
				Category:    cSpot,
				Symbol:      "BTCUSDT",
				OrderLinkID: "spot-buy-001",
			},
		},
	} {
		got, err := e.deriveCancelOrderArguments(tc.action)
		if tc.err != nil {
			require.ErrorIs(t, err, tc.err, "deriveCancelOrderArguments must return the expected error")
			continue
		}
		require.NoError(t, err, "deriveCancelOrderArguments must not error")
		assert.Equal(t, tc.exp, got, "deriveCancelOrderArguments should derive the expected request")
	}
}

func TestSetDefaultsKlineIntervals(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	ex.SetDefaults()
	// The intervals Get Kline serves; the request builder makes any other interval from one of them
	served := []kline.Interval{kline.OneMin, kline.ThreeMin, kline.FiveMin, kline.FifteenMin, kline.ThirtyMin, kline.OneHour, kline.TwoHour, kline.FourHour, kline.SixHour, kline.TwelveHour, kline.OneDay, kline.OneWeek, kline.OneMonth}
	for _, interval := range kline.SupportedIntervals {
		assert.Equalf(t, slices.Contains(served, interval), ex.Features.Enabled.Kline.Intervals.ExchangeSupported(interval), "SetDefaults should enable %s only if Bybit serves candles at it", interval)
	}
}

func TestSetDefaultsRetryPolicy(t *testing.T) {
	t.Parallel()
	// stalled returns an exchange with the requester SetDefaults builds and a short client timeout, whose server holds the
	// first request until the client gives up on it and answers the others, and the number of requests the server got
	stalled := func(t *testing.T) (*Exchange, *atomic.Int64) {
		t.Helper()
		hits := new(atomic.Int64)
		ex := newTransportTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
			// The server notices the client leave only once the request body is read
			_, err := io.Copy(io.Discard, r.Body)
			assert.NoError(t, err, "reading the request body should not error")
			if hits.Add(1) == 1 {
				select {
				case <-r.Context().Done():
				case <-time.After(10 * time.Second):
					assert.Fail(t, "the client should give up on the stalled request")
				}
				return
			}
			_, err = fmt.Fprint(w, `{"retCode":0,"retMsg":"OK","result":{},"retExtInfo":{},"time":1760000000000}`)
			assert.NoError(t, err, "writing the reply should not error")
		})
		require.NoError(t, ex.SetHTTPClientTimeout(250*time.Millisecond), "SetHTTPClientTimeout must not error")
		return ex, hits
	}

	t.Run("POST", func(t *testing.T) {
		t.Parallel()
		ex, hits := stalled(t)
		_, err := ex.PlaceOrder(t.Context(), &PlaceOrderRequest{Category: cSpot, Symbol: "BTCUSDT", Side: sideBuy, OrderType: "Limit", Quantity: 0.001, Price: 80000, TimeInForce: "GTC"})
		var urlErr *url.Error
		require.ErrorAs(t, err, &urlErr, "PlaceOrder must return the HTTP client's error")
		assert.True(t, urlErr.Timeout(), "PlaceOrder should return the client's timeout")
		assert.Equal(t, int64(1), hits.Load(), "a POST that timed out should be sent once, as Bybit may have applied it")
	})

	t.Run("GET", func(t *testing.T) {
		t.Parallel()
		ex, hits := stalled(t)
		_, err := ex.GetBybitServerTime(t.Context())
		require.NoError(t, err, "GetBybitServerTime must not error once a retry is answered")
		assert.Greater(t, hits.Load(), int64(1), "a GET that timed out should be sent again")
	})
}

func TestSetup(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	ex.SetDefaults()
	require.Error(t, ex.Setup(nil), "Setup must reject a missing configuration")
	cfg, err := ex.GetStandardConfig()
	require.NoError(t, err, "GetStandardConfig must not error")
	cfg.Enabled = false
	require.NoError(t, ex.Setup(cfg), "Setup must not error for a disabled exchange")
	assert.False(t, ex.IsEnabled(), "Setup should leave a disabled exchange disabled")

	withoutURL := func(missing exchange.URL) func(*testing.T, *Exchange, *config.Exchange) {
		return func(t *testing.T, ex *Exchange, _ *config.Exchange) {
			t.Helper()
			urls := ex.API.Endpoints.GetURLMap()
			ex.API.Endpoints = ex.NewEndpoints()
			for k, v := range urls {
				if k != missing.String() {
					require.NoError(t, ex.API.Endpoints.SetRunningURL(k, v), "SetRunningURL must not error")
				}
			}
		}
	}
	withURL := func(endpoint exchange.URL, url string) func(*testing.T, *Exchange, *config.Exchange) {
		return func(_ *testing.T, _ *Exchange, cfg *config.Exchange) {
			cfg.API.Endpoints = map[string]string{endpoint.String(): url}
		}
	}
	for _, tc := range []struct {
		name      string
		configure func(*testing.T, *Exchange, *config.Exchange)
		err       error
		// errText is matched in the error's message; the websocket manager does not export its errors
		errText string
		// registered are the streams of the connections Setup registers before the failing step; every URL resolves
		// before any connection is registered, so a missing URL registers none
		registered []string
	}{
		{name: "proxy address", configure: func(_ *testing.T, _ *Exchange, cfg *config.Exchange) { cfg.ProxyAddress = string(rune(0x7f)) }, err: exchange.ErrSettingProxyAddress},
		{name: "websocket traffic timeout", configure: func(_ *testing.T, _ *Exchange, cfg *config.Exchange) { cfg.WebsocketTrafficTimeout = time.Millisecond }, errText: "invalid traffic timeout"},
		{name: "no spot stream URL", configure: withoutURL(exchange.WebsocketSpot), err: exchange.ErrEndpointPathNotFound, errText: exchange.WebsocketSpot.String()},
		{name: "no options stream URL", configure: withoutURL(exchange.WebsocketOptions), err: exchange.ErrEndpointPathNotFound, errText: exchange.WebsocketOptions.String()},
		{name: "no USDT margined stream URL", configure: withoutURL(exchange.WebsocketUSDTMargined), err: exchange.ErrEndpointPathNotFound, errText: exchange.WebsocketUSDTMargined.String()},
		{name: "no USDC margined stream URL", configure: withoutURL(exchange.WebsocketUSDCMargined), err: exchange.ErrEndpointPathNotFound, errText: exchange.WebsocketUSDCMargined.String()},
		{name: "no coin margined stream URL", configure: withoutURL(exchange.WebsocketCoinMargined), err: exchange.ErrEndpointPathNotFound, errText: exchange.WebsocketCoinMargined.String()},
		{
			name: "no trade stream URL", configure: withoutURL(exchange.WebsocketTrade), err: exchange.ErrEndpointPathNotFound, errText: exchange.WebsocketTrade.String(),
		},
		{
			name: "no private stream URL", configure: withoutURL(exchange.WebsocketPrivate), err: exchange.ErrEndpointPathNotFound, errText: exchange.WebsocketPrivate.String(),
		},
		{
			// The spot connection is registered first, so it is refused only by a websocket manager that cannot take one
			name: "spot connection refused", configure: func(_ *testing.T, ex *Exchange, _ *config.Exchange) { ex.Websocket.TrafficAlert = nil }, errText: "traffic alert is nil",
		},
		{name: "options stream on the spot URL", configure: withURL(exchange.WebsocketOptions, spotPublic), errText: "duplicate connection setup", registered: []string{spotPublic}},
		{name: "coin margined stream on the spot URL", configure: withURL(exchange.WebsocketCoinMargined, spotPublic), errText: "duplicate connection setup", registered: []string{spotPublic, optionPublic, linearPublic}},
		{
			// The spread connection is one of those that subscribe only to the channels a user configures
			name: "coin margined stream on the spread URL", configure: withURL(exchange.WebsocketCoinMargined, mainnetStreamOrigin+spreadPublicPath), errText: "duplicate connection setup",
			registered: []string{spotPublic, optionPublic, linearPublic, mainnetStreamOrigin + spreadPublicPath, mainnetStreamOrigin + systemStatusPath, mainnetStreamOrigin + advancedEarnPath},
		},
		{
			// The streams without an endpoint of their own are placed on the spot stream's origin
			name: "spot stream URL without a host", configure: withURL(exchange.WebsocketSpot, "/v5/public/spot"), err: errNoStreamOrigin, errText: exchange.WebsocketSpot.String(),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			ex.SetDefaults()
			cfg, err := ex.GetStandardConfig()
			require.NoError(t, err, "GetStandardConfig must not error")
			tc.configure(t, ex, cfg)
			err = ex.Setup(cfg)
			require.Error(t, err, "Setup must fail")
			if tc.err != nil {
				assert.ErrorIs(t, err, tc.err, "Setup should return the failing step's error")
			}
			if tc.errText != "" {
				assert.ErrorContains(t, err, tc.errText, "Setup should return the failing step's error")
			}
			registered, err := ex.Websocket.GetConfiguredWebsocketURLs()
			require.NoError(t, err, "GetConfiguredWebsocketURLs must not error")
			assert.ElementsMatch(t, tc.registered, registered, "Setup should register only the connections before the failing step")
		})
	}

	t.Run("connections", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		ex.SetDefaults()
		cfg, err := ex.GetStandardConfig()
		require.NoError(t, err, "GetStandardConfig must not error")
		require.NoError(t, ex.Setup(cfg), "Setup must not error")
		registered, err := ex.Websocket.GetConfiguredWebsocketURLs()
		require.NoError(t, err, "GetConfiguredWebsocketURLs must not error")
		exp := []string{
			spotPublic, optionPublic, linearPublic, inversePublic, mainnetStreamOrigin + systemStatusPath, mainnetStreamOrigin + advancedEarnPath,
			mainnetStreamOrigin + spreadPublicPath, mainnetStreamOrigin + rfqPublicPath, mainnetStreamOrigin + eventPublicPath, websocketTrade, websocketPrivate,
		}
		assert.ElementsMatch(t, exp, registered, "Setup should register a connection to every stream")
		// USDT and USDC margined futures share the linear stream, so their connections are told apart by message filter
		for filter, url := range map[any]string{
			asset.USDTMarginedFutures:    linearPublic,
			asset.USDCMarginedFutures:    linearPublic,
			SystemStatusConnection:       mainnetStreamOrigin + systemStatusPath,
			AdvancedEarnOffersConnection: mainnetStreamOrigin + advancedEarnPath,
			OutboundTradeConnection:      websocketTrade,
			InboundPrivateConnection:     websocketPrivate,
		} {
			conn, err := ex.Websocket.CreateTestConnection(filter)
			require.NoErrorf(t, err, "CreateTestConnection must not error for %v", filter)
			assert.Equalf(t, url, conn.GetURL(), "Setup should connect %v to its stream", filter)
		}
	})

	// The streams without an endpoint of their own follow the configured spot stream to testnet or another host, but
	// not to a regional host, which does not serve them
	t.Run("spot stream origin", func(t *testing.T) {
		t.Parallel()
		for spotOrigin, origin := range map[string]string{
			"wss://stream-testnet.bybit.com": "wss://stream-testnet.bybit.com",
			"ws://127.0.0.1:8443":            "ws://127.0.0.1:8443",
			"wss://stream.bybit.tr":          mainnetStreamOrigin,
		} {
			ex := new(Exchange)
			ex.SetDefaults()
			cfg, err := ex.GetStandardConfig()
			require.NoError(t, err, "GetStandardConfig must not error")
			cfg.API.Endpoints = map[string]string{exchange.WebsocketSpot.String(): spotOrigin + "/v5/public/spot"}
			require.NoErrorf(t, ex.Setup(cfg), "Setup must not error for a spot stream on %s", spotOrigin)
			registered, err := ex.Websocket.GetConfiguredWebsocketURLs()
			require.NoError(t, err, "GetConfiguredWebsocketURLs must not error")
			exp := []string{
				spotOrigin + "/v5/public/spot", optionPublic, linearPublic, inversePublic, origin + systemStatusPath, origin + advancedEarnPath,
				origin + spreadPublicPath, origin + rfqPublicPath, origin + eventPublicPath, websocketTrade, websocketPrivate,
			}
			assert.ElementsMatchf(t, exp, registered, "Setup should register the streams without an endpoint of their own on %s for a spot stream on %s", origin, spotOrigin)
		}
	})

	// Bybit's docs give EU accounts a system status stream of their own on the EU host, which serves none of the others
	t.Run("EU system status stream", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		ex.SetDefaults()
		cfg, err := ex.GetStandardConfig()
		require.NoError(t, err, "GetStandardConfig must not error")
		cfg.API.Endpoints = map[string]string{exchange.WebsocketSpot.String(): "wss://stream.bybit.eu/v5/public/spot"}
		require.NoError(t, ex.Setup(cfg), "Setup must not error")
		for filter, url := range map[string]string{
			SystemStatusConnection:       "wss://stream.bybit.eu" + systemStatusPath,
			AdvancedEarnOffersConnection: mainnetStreamOrigin + advancedEarnPath,
		} {
			conn, err := ex.Websocket.CreateTestConnection(filter)
			require.NoErrorf(t, err, "CreateTestConnection must not error for %s", filter)
			assert.Equalf(t, url, conn.GetURL(), "Setup should connect %s to %s for a spot stream on the EU host", filter, url)
		}
	})

	// Each public connection subscribes to its asset's topics, and handles and unsubscribes them as that asset; the
	// pushes are live frames from each stream
	t.Run("streams", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		ex.SetDefaults()
		cfg, err := ex.GetStandardConfig()
		require.NoError(t, err, "GetStandardConfig must not error")
		cfg.Features = &config.FeaturesConfig{Enabled: config.FeaturesEnabledConfig{Websocket: true}}
		require.NoError(t, ex.Setup(cfg), "Setup must not error")
		ex.Name = t.Name()
		// Spot subscribes only to the channels configured; the other connections subscribe to their pairs' defaults
		ex.Features.Subscriptions = subscription.List{{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel}}
		pairs := map[asset.Item]currency.Pair{
			asset.Spot:                currency.NewPair(currency.ETH, currency.USDT),
			asset.USDTMarginedFutures: currency.NewBTCUSDT(),
			asset.USDCMarginedFutures: currency.NewPair(currency.BTC, currency.PERP),
			asset.CoinMarginedFutures: currency.NewBTCUSD(),
			asset.Options:             fixtureOptionPair,
		}
		for a, p := range pairs {
			require.NoErrorf(t, ex.CurrencyPairs.StorePairs(a, currency.Pairs{p}, false), "StorePairs must not error for %s available pairs", a)
			require.NoErrorf(t, ex.CurrencyPairs.StorePairs(a, currency.Pairs{p}, true), "StorePairs must not error for %s enabled pairs", a)
		}
		pushes := make(map[string][]byte)
		for _, frame := range fixtureFrames(t, "testdata/wsSetupTickers.json") {
			var push WebsocketTickerUpdate
			require.NoError(t, json.Unmarshal(frame, &push), "Unmarshal must not error")
			pushes[push.Topic] = frame
		}
		server := httptest.NewTestServer(t, mockws.CurryWsMockUpgrader(t, func(_ testing.TB, msg []byte, c *gws.Conn) error {
			var req SubscriptionRequest
			if err := json.Unmarshal(msg, &req); err != nil {
				return err
			}
			if req.Operation != subscribeOperation && req.Operation != unsubscribeOperation {
				return nil
			}
			reply := &SubscriptionResponse{Success: true, ConnectionID: "d9avrat34tmf5pih98i0-gsmne", RequestID: req.RequestID, Operation: req.Operation}
			if slices.Contains(req.Arguments, "tickers.BTC-24SEP27-170000-P-USDT") {
				// The option stream answers with the topics it applied rather than the request ID
				reply = &SubscriptionResponse{Success: true, ConnectionID: "aa01fbfffe80af37-00000001-000b37b9-7147f432704fd28c-00e1c172", Type: commandResponseType, Data: SubscriptionTopics{FailTopics: []string{}, SuccessTopics: req.Arguments}}
			}
			if err := c.WriteJSON(reply); err != nil {
				return err
			}
			for _, topic := range req.Arguments {
				if push, ok := pushes[topic]; ok && req.Operation == subscribeOperation {
					if err := c.WriteMessage(gws.TextMessage, push); err != nil {
						return err
					}
				}
			}
			return nil
		}))
		server.Start()
		require.NoError(t, ex.Websocket.SetAllConnectionURLs("ws"+strings.TrimPrefix(server.URL, "http")), "SetAllConnectionURLs must not error")
		require.NoError(t, ex.Websocket.Connect(context.WithoutCancel(t.Context())), "Connect must not error")
		t.Cleanup(func() { stopWebsocket(t, ex) })

		subscribed := func() map[asset.Item][]string {
			topics := make(map[asset.Item][]string)
			for _, s := range ex.Websocket.GetSubscriptions() {
				topics[s.Asset] = append(topics[s.Asset], s.QualifiedChannel)
			}
			for _, channels := range topics {
				slices.Sort(channels)
			}
			return topics
		}
		expTopics := map[asset.Item][]string{
			asset.Spot:                {"tickers.ETHUSDT"},
			asset.USDTMarginedFutures: {"orderbook.50.BTCUSDT", "publicTrade.BTCUSDT", "tickers.BTCUSDT"},
			asset.USDCMarginedFutures: {"orderbook.50.BTCPERP", "publicTrade.BTCPERP", "tickers.BTCPERP"},
			asset.CoinMarginedFutures: {"orderbook.50.BTCUSD", "publicTrade.BTCUSD", "tickers.BTCUSD"},
			// An option trade topic covers every contract of its base coin
			asset.Options: {"orderbook.25.BTC-24SEP27-170000-P-USDT", "publicTrade.BTC", "tickers.BTC-24SEP27-170000-P-USDT"},
		}
		assert.Equal(t, expTopics, subscribed(), "each connection should subscribe to its asset's topics")

		got := make(map[asset.Item]ticker.Price, len(pairs))
		timeout := time.After(time.Minute)
		for len(got) < len(pairs) {
			select {
			case payload := <-ex.Websocket.DataHandler.C:
				switch data := payload.Data.(type) {
				case *ticker.Price:
					got[data.AssetType] = ticker.Price{Pair: data.Pair, AssetType: data.AssetType, Last: data.Last, MarkPrice: data.MarkPrice}
				case error:
					require.NoError(t, data, "the streams must not relay errors")
				}
			case <-timeout:
				require.FailNow(t, "every stream's ticker must be relayed within a minute")
			}
		}
		// Spot tickers carry no mark price, and the option has not traded
		expTickers := map[asset.Item]ticker.Price{
			asset.Spot:                {Pair: currency.NewPair(currency.ETH, currency.USDT), AssetType: asset.Spot, Last: 2511.64},
			asset.USDTMarginedFutures: {Pair: currency.NewBTCUSDT(), AssetType: asset.USDTMarginedFutures, Last: 83596.8, MarkPrice: 83589.82},
			asset.USDCMarginedFutures: {Pair: currency.NewPair(currency.BTC, currency.PERP), AssetType: asset.USDCMarginedFutures, Last: 83480.6, MarkPrice: 83490.11},
			asset.CoinMarginedFutures: {Pair: currency.NewBTCUSD(), AssetType: asset.CoinMarginedFutures, Last: 83474.9, MarkPrice: 83476.81},
			asset.Options:             {Pair: fixtureOptionPair, AssetType: asset.Options, MarkPrice: 83345.2503722},
		}
		assert.Equal(t, expTickers, got, "each connection should handle its stream's tickers as its own asset")

		// USDT and USDC margined futures share the linear stream, so each must unsubscribe through its own connection
		for _, a := range []asset.Item{asset.USDTMarginedFutures, asset.USDCMarginedFutures} {
			conn, err := ex.Websocket.GetConnection(a)
			require.NoErrorf(t, err, "GetConnection must not error for %s", a)
			subs := slices.DeleteFunc(ex.Websocket.GetSubscriptions(), func(s *subscription.Subscription) bool { return s.Asset != a })
			require.NoErrorf(t, ex.Websocket.UnsubscribeChannels(t.Context(), conn, subs), "UnsubscribeChannels must not error for %s", a)
		}
		delete(expTopics, asset.USDTMarginedFutures)
		delete(expTopics, asset.USDCMarginedFutures)
		assert.Equal(t, expTopics, subscribed(), "UnsubscribeChannels should unsubscribe only the topics of the connection's asset")
	})
}

// marketErrorReply is the envelope Bybit answers a request it refuses with
const marketErrorReply = `{"retCode":10001,"retMsg":"params error","result":{},"retExtInfo":{},"time":1760000000000}`

// btcOptionBaseCoinsReply is Bybit's reply listing the base coins of crypto options, trimmed to BTC
const btcOptionBaseCoinsReply = `{"retCode":0,"retMsg":"success","result":{"list":[{"baseCoin":"BTC","quoteCoin":"USDT","settleCoin":"USDT","optionShowName":"BTC-Options","optionOnlineTime":1739952000000,"hasSymbol":1,"underlyingType":0}]},"retExtInfo":{},"time":1791899082770}`

// marketReplyInstance returns an exchange of its own, named for the test, whose REST requests are answered with the
// reply keyed by their path and query; any other request fails the test
func marketReplyInstance(t *testing.T, replies map[string]string) *Exchange {
	t.Helper()
	return newTransportTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		reply, ok := replies[r.URL.Path+"?"+r.URL.RawQuery]
		if !assert.Truef(t, ok, "%s?%s should be a request the test expects", r.URL.Path, r.URL.RawQuery) {
			reply = marketErrorReply
		}
		_, err := fmt.Fprint(w, reply)
		assert.NoError(t, err, "writing the reply should not error")
	})
}

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	_, err := e.FetchTradablePairs(t.Context(), asset.Futures)
	require.ErrorIs(t, err, asset.ErrNotSupported, "FetchTradablePairs must reject an asset Bybit does not trade")

	for _, tc := range []struct {
		asset asset.Item
		exp   currency.Pairs
	}{
		{asset: asset.Spot, exp: currency.Pairs{currency.NewPairWithDelimiter("BTC", "USDT", "_"), currency.NewPairWithDelimiter("ETH", "USDT", "_")}},
		// Linear instruments come in two pages; the USDT settled ones include a dated contract, whose quote holds its expiry
		{asset: asset.USDTMarginedFutures, exp: currency.Pairs{currency.NewPairWithDelimiter("BTC", "USDT", "_"), currency.NewPairWithDelimiter("10000LADYS", "USDT", "_"), currency.NewPairWithDelimiter("BTC", "USDT-25DEC26", "_")}},
		{asset: asset.USDCMarginedFutures, exp: currency.Pairs{currency.NewPairWithDelimiter("BTC", "PERP", "-"), currency.NewPairWithDelimiter("ETH", "PERP", "-")}},
		{asset: asset.CoinMarginedFutures, exp: currency.Pairs{currency.NewPairWithDelimiter("ADA", "USD", "_"), currency.NewPairWithDelimiter("BTC", "USD", "_"), currency.NewPairWithDelimiter("BTC", "USDH26", "_")}},
		// Options are listed by base coin, of the base coins Bybit reports having symbols, and BTC options come in two pages
		{asset: asset.Options, exp: currency.Pairs{currency.NewPairWithDelimiter("BTC", "26NOV24-92000-C", "-"), currency.NewPairWithDelimiter("BTC", "26NOV24-90000-P", "-"), currency.NewPairWithDelimiter("ETH", "26SEP25-14000-P", "-"), currency.NewPairWithDelimiter("SOL", "27DEC24-440-P", "-")}},
	} {
		t.Run(tc.asset.String(), func(t *testing.T) {
			t.Parallel()
			pairs, err := e.FetchTradablePairs(t.Context(), tc.asset)
			require.NoError(t, err, "FetchTradablePairs must not error")
			if mockTests {
				assert.Equal(t, tc.exp, pairs, "FetchTradablePairs should return the asset's trading instruments")
				return
			}
			assert.NotEmpty(t, pairs, "FetchTradablePairs should return the asset's pairs")
		})
	}

	// Neither an asset without a pair format nor one Bybit lists no instruments for makes a request
	ex := marketReplyInstance(t, nil)
	ps, err := ex.CurrencyPairs.Get(asset.Spot)
	require.NoError(t, err, "Get must not error")
	ps.ConfigFormat = nil
	require.NoError(t, ex.CurrencyPairs.Store(asset.Spot, ps), "Store must not error")
	_, err = ex.FetchTradablePairs(t.Context(), asset.Spot)
	require.ErrorIs(t, err, currency.ErrPairFormatIsNil, "FetchTradablePairs must report an asset without a pair format")
	require.NoError(t, ex.CurrencyPairs.Store(asset.Margin, &currency.PairStore{AssetEnabled: true, RequestFormat: &currency.PairFormat{Uppercase: true}, ConfigFormat: &currency.PairFormat{Uppercase: true, Delimiter: "_"}}), "Store must not error")
	_, err = ex.FetchTradablePairs(t.Context(), asset.Margin)
	require.ErrorIs(t, err, asset.ErrNotSupported, "FetchTradablePairs must reject an asset Bybit lists no instruments for")

	for _, tc := range []struct {
		name    string
		asset   asset.Item
		replies map[string]string
		err     error
	}{
		{name: "instruments refused", asset: asset.Spot, replies: map[string]string{"/v5/market/instruments-info?category=spot&limit=1000&status=Trading": marketErrorReply}, err: errAPIResponse},
		{name: "option base coins refused", asset: asset.Options, replies: map[string]string{"/v5/market/option-base-coins?underlyingType=0": marketErrorReply}, err: errAPIResponse},
		{
			name: "option instruments refused", asset: asset.Options, err: errAPIResponse,
			replies: map[string]string{"/v5/market/option-base-coins?underlyingType=0": btcOptionBaseCoinsReply, "/v5/market/instruments-info?baseCoin=BTC&category=option&limit=1000&status=Trading": marketErrorReply},
		},
		{
			name: "instrument without a symbol", asset: asset.Spot, err: currency.ErrCreatingPair,
			replies: map[string]string{"/v5/market/instruments-info?category=spot&limit=1000&status=Trading": `{"retCode":0,"retMsg":"OK","result":{"category":"spot","list":[{"symbol":"","baseCoin":"","quoteCoin":"USDT","status":"Trading"}]},"retExtInfo":{},"time":1791897731347}`},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := marketReplyInstance(t, tc.replies).FetchTradablePairs(t.Context(), tc.asset)
			assert.ErrorIs(t, err, tc.err, "FetchTradablePairs should return the error")
		})
	}
}

func TestFetchTradablePairsFiltersNonTradingOptions(t *testing.T) {
	t.Parallel()

	tradingPair := currency.NewPairWithDelimiter("BTC", "14MAR26-95000-C", "-")
	settlingPair := currency.NewPairWithDelimiter("BTC", "14MAR26-90000-C", "-")
	preLaunchPair := currency.NewPairWithDelimiter("BTC", "14MAR26-85000-C", "-")
	tradingInstrument := newOptionInstrumentInfo("BTC14MAR26-95000-C", tradingStatus)
	settlingInstrument := newOptionInstrumentInfo("BTC14MAR26-90000-C", "Settling")
	preLaunchInstrument := newOptionInstrumentInfo("BTC14MAR26-85000-C", "PreLaunch")

	var queries []url.Values
	ex := newInstrumentInfoTestExchange(t, "BybitOptionsFetchTradablePairsTest", cOption, map[string][]InstrumentInfo{
		"BTC": {tradingInstrument, settlingInstrument, preLaunchInstrument},
	}, &queries)

	pairs, err := ex.FetchTradablePairs(t.Context(), asset.Options)
	require.NoError(t, err, "FetchTradablePairs must not error")
	assert.Equal(t, currency.Pairs{tradingPair}, pairs, "FetchTradablePairs should only include trading option pairs")

	require.NoError(t, ex.UpdatePairs(pairs, asset.Options, false), "UpdatePairs must not error for available option pairs")
	available, err := ex.GetAvailablePairs(asset.Options)
	require.NoError(t, err, "GetAvailablePairs must not error")
	assert.Equal(t, currency.Pairs{tradingPair}, available, "Available option pairs should only include trading instruments")

	enabled, err := ex.GetEnabledPairs(asset.Options)
	require.NoError(t, err, "GetEnabledPairs must not error")
	assert.Equal(t, currency.Pairs{tradingPair}, enabled, "Enabled option pairs should only include trading instruments")

	err = ex.CurrencyPairs.EnablePair(asset.Options, settlingPair)
	require.ErrorIs(t, err, currency.ErrPairNotFound, "EnablePair must error for a settling option pair")

	err = ex.CurrencyPairs.EnablePair(asset.Options, preLaunchPair)
	require.ErrorIs(t, err, currency.ErrPairNotFound, "EnablePair must error for a pre-launch option pair")

	assertOptionsInstrumentQueries(t, queries, "FetchTradablePairs", "BTC")
}

func TestOptionBaseCoins(t *testing.T) {
	t.Parallel()
	// Only crypto underlyings are asked for; stock, commodity, forex and oil options need TradFi agreements
	_, err := marketReplyInstance(t, map[string]string{"/v5/market/option-base-coins?underlyingType=0": marketErrorReply}).optionBaseCoins(t.Context())
	require.ErrorIs(t, err, errAPIResponse, "optionBaseCoins must return the error of a refused request")

	coins, err := e.optionBaseCoins(t.Context())
	require.NoError(t, err, "optionBaseCoins must not error")
	if mockTests {
		// XAUT is recorded with no option contracts listed yet
		assert.Equal(t, []currency.Code{currency.BTC, currency.ETH, currency.SOL}, coins, "optionBaseCoins should return the base coins that have option contracts")
		return
	}
	assert.Contains(t, coins, currency.BTC, "optionBaseCoins should return BTC, which Bybit lists options of")
}

func TestUpdateTradablePairs(t *testing.T) {
	t.Parallel()
	ex := testInstance(t)
	require.NoError(t, ex.UpdateTradablePairs(t.Context()), "UpdateTradablePairs must not error")
	for _, tc := range []struct {
		asset              asset.Item
		available, enabled currency.Pairs
	}{
		{asset: asset.Spot, available: currency.Pairs{currency.NewPairWithDelimiter("BTC", "USDT", "_"), currency.NewPairWithDelimiter("ETH", "USDT", "_")}, enabled: currency.Pairs{currency.NewPairWithDelimiter("BTC", "USDT", "_")}},
		{asset: asset.USDTMarginedFutures, available: currency.Pairs{currency.NewPairWithDelimiter("BTC", "USDT", "_"), currency.NewPairWithDelimiter("10000LADYS", "USDT", "_"), currency.NewPairWithDelimiter("BTC", "USDT-25DEC26", "_")}, enabled: currency.Pairs{currency.NewPairWithDelimiter("BTC", "USDT", "_"), currency.NewPairWithDelimiter("10000LADYS", "USDT", "_")}},
		{asset: asset.USDCMarginedFutures, available: currency.Pairs{currency.NewPairWithDelimiter("BTC", "PERP", "-"), currency.NewPairWithDelimiter("ETH", "PERP", "-")}, enabled: currency.Pairs{currency.NewPairWithDelimiter("BTC", "PERP", "-"), currency.NewPairWithDelimiter("ETH", "PERP", "-")}},
		{asset: asset.CoinMarginedFutures, available: currency.Pairs{currency.NewPairWithDelimiter("ADA", "USD", "_"), currency.NewPairWithDelimiter("BTC", "USD", "_"), currency.NewPairWithDelimiter("BTC", "USDH26", "_")}, enabled: currency.Pairs{currency.NewPairWithDelimiter("BTC", "USD", "_"), currency.NewPairWithDelimiter("ADA", "USD", "_")}},
		// The configured SOL option is no longer listed, so it is disabled
		{asset: asset.Options, available: currency.Pairs{currency.NewPairWithDelimiter("BTC", "26NOV24-92000-C", "-"), currency.NewPairWithDelimiter("BTC", "26NOV24-90000-P", "-"), currency.NewPairWithDelimiter("ETH", "26SEP25-14000-P", "-"), currency.NewPairWithDelimiter("SOL", "27DEC24-440-P", "-")}, enabled: currency.Pairs{currency.NewPairWithDelimiter("BTC", "26NOV24-92000-C", "-")}},
	} {
		available, err := ex.GetAvailablePairs(tc.asset)
		require.NoErrorf(t, err, "GetAvailablePairs must not error for %s", tc.asset)
		enabled, err := ex.GetEnabledPairs(tc.asset)
		require.NoErrorf(t, err, "GetEnabledPairs must not error for %s", tc.asset)
		if mockTests {
			assert.Equalf(t, tc.available, available, "UpdateTradablePairs should store the %s pairs Bybit lists", tc.asset)
			assert.Equalf(t, tc.enabled, enabled, "UpdateTradablePairs should keep the enabled %s pairs Bybit still lists", tc.asset)
			continue
		}
		assert.NotEmptyf(t, available, "UpdateTradablePairs should store %s pairs", tc.asset)
		assert.NotEmptyf(t, enabled, "UpdateTradablePairs should leave a %s pair enabled", tc.asset)
	}

	// Spot alone is enabled, so its instruments are the only ones requested
	for _, tc := range []struct {
		name  string
		reply string
		err   error
	}{
		{name: "instruments refused", reply: marketErrorReply, err: errAPIResponse},
		// A category without a trading instrument would otherwise remove every pair
		{name: "no trading instruments", reply: `{"retCode":0,"retMsg":"OK","result":{"category":"spot","list":[]},"retExtInfo":{},"time":1791897731347}`, err: currency.ErrCurrencyPairsEmpty},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := marketReplyInstance(t, map[string]string{"/v5/market/instruments-info?category=spot&limit=1000&status=Trading": tc.reply})
			for _, a := range []asset.Item{asset.USDTMarginedFutures, asset.USDCMarginedFutures, asset.CoinMarginedFutures, asset.Options} {
				require.NoErrorf(t, ex.CurrencyPairs.SetAssetEnabled(a, false), "SetAssetEnabled must not error for %s", a)
			}
			before, err := ex.GetAvailablePairs(asset.Spot)
			require.NoError(t, err, "GetAvailablePairs must not error")
			err = ex.UpdateTradablePairs(t.Context())
			require.ErrorIs(t, err, tc.err, "UpdateTradablePairs must return the error")
			after, err := ex.GetAvailablePairs(asset.Spot)
			require.NoError(t, err, "GetAvailablePairs must not error")
			assert.Equal(t, before, after, "UpdateTradablePairs should keep the available pairs it could not update")
		})
	}
}

// TestUpdateTickerInverseVolumes covers the websocket path, which merges partial updates into the
// cached ticker and so has to map the pair the same way the REST path does
func TestUpdateTickerInverseVolumes(t *testing.T) {
	t.Parallel()
	wsTicker := func(volume24Hour, turnover24Hour types.Number) *WebsocketTicker {
		t := new(WebsocketTicker)
		t.Volume24Hour, t.Turnover24Hour = volume24Hour, turnover24Hour
		return t
	}
	volumes := map[string]bool{"volume24h": true, "turnover24h": true}

	tick := &ticker.Price{AssetType: asset.CoinMarginedFutures}
	updateTicker(tick, wsTicker(147122986, 1873.7925), volumes)
	assert.Equal(t, 1873.7925, tick.BaseVolume, "an inverse ticker's turnover24h should be recorded as the base volume")
	assert.Equal(t, 147122986.0, tick.QuoteVolume, "an inverse ticker's volume24h should be recorded as the quote volume")

	// a delta carrying only the volumes must not overwrite the base figure with the quote one
	updateTicker(tick, wsTicker(140241671, 1787.3508), volumes)
	assert.Equal(t, 1787.3508, tick.BaseVolume, "a later update should keep mapping turnover24h to the base volume")
	assert.Equal(t, 140241671.0, tick.QuoteVolume, "a later update should keep mapping volume24h to the quote volume")

	linear := &ticker.Price{AssetType: asset.USDTMarginedFutures}
	updateTicker(linear, wsTicker(52633.7910, 4129941840.8562), volumes)
	assert.Equal(t, 52633.7910, linear.BaseVolume, "a linear ticker's volume24h should be recorded as the base volume")
	assert.Equal(t, 4129941840.8562, linear.QuoteVolume, "a linear ticker's turnover24h should be recorded as the quote volume")
}

func TestBaseQuoteVolumes(t *testing.T) {
	t.Parallel()
	// The volume24h and turnover24h of a ticker of each category
	for _, tc := range []struct {
		asset            asset.Item
		volume, turnover types.Number
		exp              [2]float64
	}{
		// An inverse contract is worth one unit of its quote currency, so volume counts that currency and turnover the base
		{asset: asset.CoinMarginedFutures, volume: 147122986, turnover: 1873.7925, exp: [2]float64{1873.7925, 147122986}},
		{asset: asset.USDTMarginedFutures, volume: 52633.7910, turnover: 4129941840.8562, exp: [2]float64{52633.7910, 4129941840.8562}},
		{asset: asset.Spot, volume: 8125.606399, turnover: 638025011.4045, exp: [2]float64{8125.606399, 638025011.4045}},
		{asset: asset.Options, volume: 0.08, turnover: 6269.45554303, exp: [2]float64{0.08, 6269.45554303}},
	} {
		baseVolume, quoteVolume := baseQuoteVolumes(tc.volume, tc.turnover, tc.asset)
		assert.Equalf(t, tc.exp, [2]float64{baseVolume, quoteVolume}, "baseQuoteVolumes should return the base and quote volumes of %s", tc.asset)
	}
}

// tickerTestCase is an enabled pair whose ticker the fixtures record
type tickerTestCase struct {
	asset asset.Item
	pair  currency.Pair
	exp   *ticker.Price
}

// tickerTestCases returns the test pair of each asset and, in mock tests, its ticker; call ensureTradablePairs first
func tickerTestCases() []tickerTestCase {
	return []tickerTestCase{
		{
			asset: asset.Spot,
			pair:  spotTradablePair,
			exp:   &ticker.Price{Last: 34466.01, High: 34724, Low: 34029.55, Bid: 34466, BidSize: 0.879803, Ask: 34466.01, AskSize: 0.68816, BaseVolume: 6993.899145, QuoteVolume: 240396884.49893426, Pair: currency.NewPairWithDelimiter("BTC", "USDT", "_"), AssetType: asset.Spot},
		},
		{
			asset: asset.USDTMarginedFutures,
			pair:  usdtMarginedTradablePair,
			exp:   &ticker.Price{Last: 0.0007338, High: 0.0007971, Low: 0.0006665, Bid: 0.0007338, BidSize: 42500, Ask: 0.0007342, AskSize: 325700, BaseVolume: 24211728200, QuoteVolume: 17938033.6173, Pair: currency.NewPairWithDelimiter("10000LADYS", "USDT", "_"), AssetType: asset.USDTMarginedFutures},
		},
		{
			asset: asset.USDCMarginedFutures,
			pair:  usdcMarginedTradablePair,
			exp:   &ticker.Price{Last: 2508.57, High: 2541.73, Low: 2404.47, Bid: 2507.69, BidSize: 0.25, Ask: 2507.7, AskSize: 0.09, BaseVolume: 9443.02, QuoteVolume: 23600073.7027, Pair: currency.NewPairWithDelimiter("ETH", "PERP", "-"), AssetType: asset.USDCMarginedFutures},
		},
		{
			// An inverse contract is worth one unit of the quote currency, so turnover24h is the base volume
			asset: asset.CoinMarginedFutures,
			pair:  inverseTradablePair,
			exp:   &ticker.Price{Last: 0.2353, High: 0.2482, Low: 0.2237, Bid: 0.2351, BidSize: 304, Ask: 0.2352, AskSize: 301, BaseVolume: 18168059.7354, QuoteVolume: 4313313, Pair: currency.NewPairWithDelimiter("ADA", "USD", "_"), AssetType: asset.CoinMarginedFutures},
		},
		{
			// Option tickers are requested by the base coins of the enabled pairs
			asset: asset.Options,
			pair:  optionsTradablePair,
			exp:   &ticker.Price{Last: 10, High: 10, Low: 5, Ask: 30, AskSize: 0.1, BaseVolume: 0.09, QuoteVolume: 3383.89479203, Pair: currency.NewPairWithDelimiter("BTC", "26NOV24-92000-C", "-"), AssetType: asset.Options},
		},
	}
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	err := e.UpdateTickers(t.Context(), asset.Futures)
	require.ErrorIs(t, err, asset.ErrNotSupported, "UpdateTickers must reject an asset Bybit does not trade")

	ensureTradablePairs(t)
	for _, tc := range tickerTestCases() {
		t.Run(tc.asset.String(), func(t *testing.T) {
			t.Parallel()
			ex := storeTestInstance(t)
			require.NoError(t, ex.UpdateTickers(t.Context(), tc.asset), "UpdateTickers must not error")
			tick, err := ticker.GetTicker(ex.Name, tc.pair, tc.asset)
			require.NoError(t, err, "GetTicker must find the enabled pair's ticker")
			if !mockTests {
				assert.Truef(t, tick.Pair.Equal(tc.pair), "UpdateTickers should store the ticker of %s", tc.pair)
				return
			}
			tc.exp.ExchangeName = ex.Name
			tick.LastUpdated = time.Time{}
			assert.Equal(t, tc.exp, tick, "UpdateTickers should store the enabled pair's ticker")
		})
	}

	ex := storeTestInstance(t)
	require.NoError(t, ex.UpdateTickers(t.Context(), asset.Spot), "UpdateTickers must not error")
	_, err = ticker.GetTicker(ex.Name, currency.NewPairWithDelimiter("ETH", "USDT", "_"), asset.Spot)
	assert.ErrorIs(t, err, ticker.ErrTickerNotFound, "UpdateTickers should store only the enabled pairs' tickers")
	require.NoError(t, ex.CurrencyPairs.SetAssetEnabled(asset.Spot, false), "SetAssetEnabled must not error")
	err = ex.UpdateTickers(t.Context(), asset.Spot)
	assert.ErrorIs(t, err, asset.ErrNotEnabled, "UpdateTickers should reject a disabled asset")

	// An asset whose pairs Bybit has no tickers for is refused before any request
	ex = marketReplyInstance(t, nil)
	require.NoError(t, ex.CurrencyPairs.Store(asset.Margin, &currency.PairStore{AssetEnabled: true, Available: currency.Pairs{currency.NewBTCUSDT()}, Enabled: currency.Pairs{currency.NewBTCUSDT()}, RequestFormat: &currency.PairFormat{Uppercase: true}, ConfigFormat: &currency.PairFormat{Uppercase: true, Delimiter: "_"}}), "Store must not error")
	err = ex.UpdateTickers(t.Context(), asset.Margin)
	require.ErrorIs(t, err, asset.ErrNotSupported, "UpdateTickers must reject an asset Bybit has no tickers for")

	// Option tickers come for every contract of a base coin, and only those of the enabled contracts are stored
	call := currency.NewPairWithDelimiter("BTC", "24SEP27-170000-C-USDT", currency.DashDelimiter)
	ex = marketReplyInstance(t, map[string]string{"/v5/market/tickers?baseCoin=BTC&category=option": `{"retCode":0,"retMsg":"OK","result":{"category":"option","list":[{"symbol":"BTC-24SEP27-170000-C-USDT","bid1Price":"1060","bid1Size":"38","bid1Iv":"0.4182","ask1Price":"1565","ask1Size":"38","ask1Iv":"0.455","lastPrice":"1160","highPrice24h":"1160","lowPrice24h":"1160","markPrice":"1222.57618805","indexPrice":"83360.7469723","markIv":"0.431","underlyingPrice":"87578.5","openInterest":"0.25","turnover24h":"0","volume24h":"0","totalVolume":"1","totalTurnover":"25944","delta":"0.08507457","gamma":"0.00000424","vega":"132.72640173","theta":"-8.27084263","predictedDeliveryPrice":"0","change24h":"0"},{"symbol":"BTC-24SEP27-170000-P-USDT","bid1Price":"82440","bid1Size":"18.55","bid1Iv":"0.2395","ask1Price":"84715","ask1Size":"18.55","ask1Iv":"0.4989","lastPrice":"0","highPrice24h":"0","lowPrice24h":"0","markPrice":"83644.07618805","indexPrice":"83360.7469723","markIv":"0.431","underlyingPrice":"87578.5","openInterest":"0","turnover24h":"0","volume24h":"0","totalVolume":"0","totalTurnover":"0","delta":"-0.91492544","gamma":"0.00000424","vega":"132.72640173","theta":"-8.27084263","predictedDeliveryPrice":"0","change24h":"0"}]},"retExtInfo":{},"time":1791899228444}`})
	require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Options, currency.Pairs{call, fixtureOptionPair}, false), "StorePairs must not error for available pairs")
	require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Options, currency.Pairs{fixtureOptionPair}, true), "StorePairs must not error for enabled pairs")
	require.NoError(t, ex.UpdateTickers(t.Context(), asset.Options), "UpdateTickers must not error")
	tick, err := ticker.GetTicker(ex.Name, fixtureOptionPair, asset.Options)
	require.NoError(t, err, "GetTicker must find the enabled option's ticker")
	tick.LastUpdated = time.Time{}
	exp := &ticker.Price{Bid: 82440, BidSize: 18.55, Ask: 84715, AskSize: 18.55, Pair: fixtureOptionPair, ExchangeName: ex.Name, AssetType: asset.Options}
	assert.Equal(t, exp, tick, "UpdateTickers should store the enabled option's ticker")
	_, err = ticker.GetTicker(ex.Name, call, asset.Options)
	assert.ErrorIs(t, err, ticker.ErrTickerNotFound, "UpdateTickers should not store the ticker of an option that is not enabled")

	// Each locked market is Bybit's ticker with its best bid raised to its best ask
	for _, tc := range []struct {
		name    string
		asset   asset.Item
		replies map[string]string
		err     error
	}{
		{name: "tickers refused", asset: asset.Spot, replies: map[string]string{"/v5/market/tickers?category=spot": marketErrorReply}, err: errAPIResponse},
		{
			name: "locked market", asset: asset.Spot, err: ticker.ErrBidEqualsAsk,
			replies: map[string]string{"/v5/market/tickers?category=spot": `{"retCode":0,"retMsg":"OK","result":{"category":"spot","list":[{"symbol":"BTCUSDT","bid1Price":"83357.5","bid1Size":"1.441468","ask1Price":"83357.5","ask1Size":"0.570892","lastPrice":"83357.5","prevPrice24h":"82737.9","price24hPcnt":"0.0075","highPrice24h":"84172.6","lowPrice24h":"81031.2","turnover24h":"857094249.58053143","volume24h":"10348.591662","usdIndexPrice":"83253.998511"}]},"retExtInfo":{},"time":1791899227170}`},
		},
		{name: "option tickers refused", asset: asset.Options, replies: map[string]string{"/v5/market/tickers?baseCoin=BTC&category=option": marketErrorReply}, err: errAPIResponse},
		{
			name: "locked option market", asset: asset.Options, err: ticker.ErrBidEqualsAsk,
			replies: map[string]string{"/v5/market/tickers?baseCoin=BTC&category=option": `{"retCode":0,"retMsg":"OK","result":{"category":"option","list":[{"symbol":"BTC-24SEP27-170000-P-USDT","bid1Price":"84715","bid1Size":"18.55","bid1Iv":"0.2395","ask1Price":"84715","ask1Size":"18.55","ask1Iv":"0.4989","lastPrice":"0","highPrice24h":"0","lowPrice24h":"0","markPrice":"83644.07618805","indexPrice":"83360.7469723","markIv":"0.431","underlyingPrice":"87578.5","openInterest":"0","turnover24h":"0","volume24h":"0","totalVolume":"0","totalTurnover":"0","delta":"-0.91492544","gamma":"0.00000424","vega":"132.72640173","theta":"-8.27084263","predictedDeliveryPrice":"0","change24h":"0"}]},"retExtInfo":{},"time":1791899228444}`},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := marketReplyInstance(t, tc.replies)
			require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Options, currency.Pairs{fixtureOptionPair}, false), "StorePairs must not error for available pairs")
			require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Options, currency.Pairs{fixtureOptionPair}, true), "StorePairs must not error for enabled pairs")
			err := ex.UpdateTickers(t.Context(), tc.asset)
			assert.ErrorIs(t, err, tc.err, "UpdateTickers should return the error")
		})
	}
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateTicker(t.Context(), currency.NewBTCUSDT(), asset.Futures)
	require.ErrorIs(t, err, asset.ErrNotSupported, "UpdateTicker must reject an asset Bybit does not trade")

	ensureTradablePairs(t)
	for _, tc := range tickerTestCases() {
		t.Run(tc.asset.String(), func(t *testing.T) {
			t.Parallel()
			ex := storeTestInstance(t)
			tick, err := ex.UpdateTicker(t.Context(), tc.pair, tc.asset)
			require.NoError(t, err, "UpdateTicker must not error")
			if !mockTests {
				assert.Truef(t, tick.Pair.Equal(tc.pair), "UpdateTicker should return the ticker of %s", tc.pair)
				return
			}
			tc.exp.ExchangeName = ex.Name
			tick.LastUpdated = time.Time{}
			assert.Equal(t, tc.exp, tick, "UpdateTicker should return the pair's ticker")
		})
	}

	ex := storeTestInstance(t)
	_, err = ex.UpdateTicker(t.Context(), currency.NewPairWithDelimiter("ETH", "USDT", "_"), asset.Spot)
	assert.ErrorIs(t, err, ticker.ErrTickerNotFound, "UpdateTicker should not find the ticker of a pair that is not enabled")
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateOrderbook(t.Context(), currency.EMPTYPAIR, asset.Spot)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "UpdateOrderbook must reject an empty pair")
	_, err = e.UpdateOrderbook(t.Context(), currency.NewBTCUSDT(), asset.Futures)
	require.ErrorIs(t, err, asset.ErrNotSupported, "UpdateOrderbook must reject an asset Bybit does not trade")

	ensureTradablePairs(t)
	for _, tc := range []struct {
		asset asset.Item
		pair  currency.Pair
		exp   *orderbook.Book
	}{
		{
			// Bybit returns one level of a spot book unless asked for more, so UpdateOrderbook asks for 1000
			asset: asset.Spot,
			pair:  spotTradablePair,
			exp: &orderbook.Book{
				Bids:         orderbook.Levels{{Price: 83465.5, Amount: 0.462911}},
				Asks:         orderbook.Levels{{Price: 83465.6, Amount: 0.824568}},
				Pair:         currency.NewBTCUSDT(),
				Asset:        asset.Spot,
				LastUpdated:  time.UnixMilli(1791898229153),
				LastUpdateID: 60122406,
			},
		},
		{
			asset: asset.USDTMarginedFutures,
			pair:  usdtMarginedTradablePair,
			exp: &orderbook.Book{
				Bids:         orderbook.Levels{{Price: 0.000428, Amount: 1402800}, {Price: 0.0004275, Amount: 9126600}, {Price: 0.000427, Amount: 10330600}},
				Asks:         orderbook.Levels{{Price: 0.000429, Amount: 5450600}, {Price: 0.0004295, Amount: 5663600}, {Price: 0.00043, Amount: 12162400}},
				Pair:         currency.NewPairWithDelimiter("10000LADYS", "USDT", ""),
				Asset:        asset.USDTMarginedFutures,
				LastUpdated:  time.UnixMilli(1693087746322),
				LastUpdateID: 44717580,
			},
		},
		{
			asset: asset.USDCMarginedFutures,
			pair:  usdcMarginedTradablePair,
			exp: &orderbook.Book{
				Bids:         orderbook.Levels{{Price: 2507.59, Amount: 0.06}, {Price: 2507.41, Amount: 0.03}, {Price: 2507.12, Amount: 0.09}},
				Asks:         orderbook.Levels{{Price: 2507.6, Amount: 0.82}, {Price: 2507.61, Amount: 0.25}, {Price: 2507.68, Amount: 0.04}},
				Pair:         currency.NewPairWithDelimiter("ETH", "PERP", ""),
				Asset:        asset.USDCMarginedFutures,
				LastUpdated:  time.UnixMilli(1791898230472),
				LastUpdateID: 91531,
			},
		},
		{
			asset: asset.CoinMarginedFutures,
			pair:  inverseTradablePair,
			exp: &orderbook.Book{
				Bids:         orderbook.Levels{{Price: 0.235, Amount: 4158}, {Price: 0.2349, Amount: 11468}, {Price: 0.2348, Amount: 10523}},
				Asks:         orderbook.Levels{{Price: 0.2352, Amount: 3892}, {Price: 0.2353, Amount: 10061}, {Price: 0.2354, Amount: 13363}},
				Pair:         currency.NewPairWithDelimiter("ADA", "USD", ""),
				Asset:        asset.CoinMarginedFutures,
				LastUpdated:  time.UnixMilli(1791898231671),
				LastUpdateID: 61581,
			},
		},
		{
			// Option books are 25 levels deep at most
			asset: asset.Options,
			pair:  optionsTradablePair,
			exp: &orderbook.Book{
				Bids:         orderbook.Levels{{Price: 500, Amount: 0.03}},
				Asks:         orderbook.Levels{},
				Pair:         currency.NewPairWithDelimiter("BTC", "26NOV24-92000-C", "-"),
				Asset:        asset.Options,
				LastUpdated:  time.UnixMilli(1700502830097),
				LastUpdateID: 5426736984,
			},
		},
	} {
		t.Run(tc.asset.String(), func(t *testing.T) {
			t.Parallel()
			ex := storeTestInstance(t)
			book, err := ex.UpdateOrderbook(t.Context(), tc.pair, tc.asset)
			require.NoError(t, err, "UpdateOrderbook must not error")
			if !mockTests {
				// A live option's book can be empty on either side
				if tc.asset != asset.Options {
					assert.NotEmpty(t, book.Bids, "UpdateOrderbook should return bids")
					assert.NotEmpty(t, book.Asks, "UpdateOrderbook should return asks")
				}
				return
			}
			tc.exp.Exchange, tc.exp.ValidateOrderbook, tc.exp.RestSnapshot = ex.Name, true, true
			book.InsertedAt = time.Time{}
			assert.Equal(t, tc.exp, book, "UpdateOrderbook should return the pair's book")
		})
	}

	ex := storeTestInstance(t)
	require.NoError(t, ex.CurrencyPairs.SetAssetEnabled(asset.Spot, false), "SetAssetEnabled must not error")
	_, err = ex.UpdateOrderbook(t.Context(), spotTradablePair, asset.Spot)
	assert.ErrorIs(t, err, asset.ErrNotEnabled, "UpdateOrderbook should reject a disabled asset")

	// A USDC margined futures contract is asked for by its dashed symbol. Bybit lists none now, but still answers for
	// the delivered BTC-26DEC25 with an empty book and no timestamp, which is stamped when it is stored
	ex = storeTestInstance(t)
	usdcFutures := currency.NewPair(currency.BTC, currency.NewCode("26DEC25"))
	book, err := ex.UpdateOrderbook(t.Context(), usdcFutures, asset.USDCMarginedFutures)
	require.NoError(t, err, "UpdateOrderbook must not error for a USDC margined futures contract")
	assert.WithinDuration(t, time.Now(), book.LastUpdated, time.Minute, "UpdateOrderbook should stamp a book Bybit sends without a time")
	book.InsertedAt, book.LastUpdated = time.Time{}, time.Time{}
	exp := &orderbook.Book{Exchange: ex.Name, Pair: currency.NewPairWithDelimiter("BTC", "26DEC25", currency.DashDelimiter), Asset: asset.USDCMarginedFutures, Bids: orderbook.Levels{}, Asks: orderbook.Levels{}, ValidateOrderbook: true, RestSnapshot: true}
	assert.Equal(t, exp, book, "UpdateOrderbook should return the contract's book")

	// Neither an asset without a request format nor one Bybit has no books for makes a request
	ex = marketReplyInstance(t, nil)
	ps, err := ex.CurrencyPairs.Get(asset.Spot)
	require.NoError(t, err, "Get must not error")
	ps.RequestFormat = nil
	require.NoError(t, ex.CurrencyPairs.Store(asset.Spot, ps), "Store must not error")
	_, err = ex.UpdateOrderbook(t.Context(), currency.NewBTCUSDT(), asset.Spot)
	require.ErrorIs(t, err, currency.ErrPairFormatIsNil, "UpdateOrderbook must report an asset without a request format")
	require.NoError(t, ex.CurrencyPairs.Store(asset.Margin, &currency.PairStore{AssetEnabled: true, RequestFormat: &currency.PairFormat{Uppercase: true}, ConfigFormat: &currency.PairFormat{Uppercase: true, Delimiter: "_"}}), "Store must not error")
	_, err = ex.UpdateOrderbook(t.Context(), currency.NewBTCUSDT(), asset.Margin)
	require.ErrorIs(t, err, asset.ErrNotSupported, "UpdateOrderbook must reject an asset Bybit has no books for")

	ex = marketReplyInstance(t, map[string]string{"/v5/market/orderbook?category=spot&limit=1000&symbol=BTCUSDT": marketErrorReply})
	_, err = ex.UpdateOrderbook(t.Context(), currency.NewBTCUSDT(), asset.Spot)
	require.ErrorIs(t, err, errAPIResponse, "UpdateOrderbook must return the error of a refused request")

	// Bybit's book with its best bid priced at zero, which the orderbook store refuses
	ex = marketReplyInstance(t, map[string]string{"/v5/market/orderbook?category=spot&limit=1000&symbol=BTCUSDT": `{"retCode":0,"retMsg":"OK","result":{"s":"BTCUSDT","a":[["83465.6","0.824568"]],"b":[["0","0.462911"]],"ts":1791898229153,"u":60122406,"seq":115367826202,"cts":1791898229150},"retExtInfo":{},"time":1791898229225}`})
	book, err = ex.UpdateOrderbook(t.Context(), currency.NewBTCUSDT(), asset.Spot)
	require.ErrorIs(t, err, orderbook.ErrPriceZero, "UpdateOrderbook must return the store's refusal")
	exp = &orderbook.Book{Exchange: ex.Name, Pair: currency.NewBTCUSDT(), Asset: asset.Spot, Bids: orderbook.Levels{{Amount: 0.462911}}, Asks: orderbook.Levels{{Price: 83465.6, Amount: 0.824568}}, LastUpdated: time.UnixMilli(1791898229153), LastUpdateID: 60122406, ValidateOrderbook: true}
	assert.Equal(t, exp, book, "UpdateOrderbook should return the book the store refused")
}

func TestOrderbookDepth(t *testing.T) {
	t.Parallel()
	// Get Orderbook serves up to 1000 levels of spot, linear and inverse books, and 25 of option books
	for a, exp := range map[asset.Item]uint64{
		asset.Spot:                1000,
		asset.USDTMarginedFutures: 1000,
		asset.USDCMarginedFutures: 1000,
		asset.CoinMarginedFutures: 1000,
		asset.Options:             25,
	} {
		assert.Equalf(t, exp, orderbookDepth(a), "orderbookDepth should return the deepest %s book Bybit serves", a)
	}
}

func TestUpdateAccountBalances(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateAccountBalances(t.Context(), asset.Margin)
	require.ErrorIs(t, err, asset.ErrNotSupported, "UpdateAccountBalances must reject an asset Bybit does not trade")
	refused, err := restReplyTestInstance(t, nil).UpdateAccountBalances(t.Context(), asset.Spot)
	require.ErrorIs(t, err, errAPIResponse, "UpdateAccountBalances must return Bybit's refusal")
	assert.Nil(t, refused, "UpdateAccountBalances should return no balances when Bybit refuses to report them")
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	ex := testInstance(t)
	var last accounts.CurrencyBalances
	for _, a := range []asset.Item{asset.Spot, asset.USDTMarginedFutures, asset.USDCMarginedFutures, asset.CoinMarginedFutures, asset.Options} {
		subAccts, err := ex.UpdateAccountBalances(t.Context(), a)
		require.NoErrorf(t, err, "UpdateAccountBalances must not error for %s", a)
		require.Lenf(t, subAccts, 1, "UpdateAccountBalances must return the unified account for %s", a)
		assert.Equalf(t, asset.Spot, subAccts[0].AssetType, "UpdateAccountBalances should return the unified account as the spot sub account for %s", a)
		cached, err := ex.GetCachedCurrencyBalances(t.Context(), a)
		require.NoErrorf(t, err, "GetCachedCurrencyBalances must not error for %s", a)
		assert.Equalf(t, subAccts[0].Balances, cached, "UpdateAccountBalances should store the balances it returns for %s", a)
		last = maps.Clone(subAccts[0].Balances)
		if !mockTests {
			continue
		}
		// BTC's wallet balance includes its spot borrowing, its orders and positions hold margin, and USDC owes
		// derivatives losses
		btcWallet, btcLocked, btcOrderMargin, btcPositionMargin, btcSpotBorrow, btcBonus := 0.01792413, 0.002, 0.0001, 0.00025, 0.001, 0.0005
		btcHold := btcLocked + btcOrderMargin + btcPositionMargin
		exp := accounts.CurrencyBalances{
			currency.BTC:  {Currency: currency.BTC, Total: btcWallet, Hold: btcHold, Free: btcWallet - btcHold, Borrowed: 0.001, AvailableWithoutBorrow: btcWallet - btcSpotBorrow - btcHold - btcBonus},
			currency.USDT: {Currency: currency.USDT, Total: 0.80486234, Free: 0.80486234, AvailableWithoutBorrow: 0.80486234},
			currency.USDC: {Currency: currency.USDC, Total: -12.5, Borrowed: 12.5},
		}
		for code, balance := range subAccts[0].Balances {
			balance.UpdatedAt = time.Time{}
			subAccts[0].Balances[code] = balance
		}
		assert.Equalf(t, exp, subAccts[0].Balances, "UpdateAccountBalances should map every unified balance for %s", a)
	}
	// The unified account is stored once, so balances across every asset type count it once
	stored, err := ex.Accounts.SubAccounts(nil, asset.All)
	require.NoError(t, err, "SubAccounts must not error")
	require.Len(t, stored, 1, "UpdateAccountBalances must store the unified account once for every asset type asked for")
	all, err := ex.Accounts.CurrencyBalances(nil, asset.All)
	require.NoError(t, err, "CurrencyBalances must not error")
	assert.Equal(t, last, all, "balances across every asset type should be the unified account's")
}

func TestUpdateAccountBalancesKeepsNewerWalletPush(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	var push, coins string // The wallet push arriving during the next request, if any, and the coins its reply lists
	var ex *Exchange
	ex = newTransportTestExchange(t, func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		pushed, listed := push, coins
		mu.Unlock()
		if pushed != "" {
			assert.NoError(t, ex.wsProcessWalletPushData(t.Context(), []byte(`{"id":"wallet-push","topic":"wallet","creationTime":1787197420650,"data":[{"accountType":"UNIFIED","coin":[`+pushed+`]}]}`)), "a wallet push arriving during the request should not error")
		}
		_, err := fmt.Fprint(w, `{"retCode":0,"retMsg":"OK","result":{"list":[{"accountType":"UNIFIED","coin":[`+listed+`]}]},"retExtInfo":{},"time":1760000000000}`)
		assert.NoError(t, err, "writing the reply should not error")
	})
	withoutTimes := func(b accounts.CurrencyBalances) accounts.CurrencyBalances {
		for code, balance := range b {
			balance.UpdatedAt = time.Time{}
			b[code] = balance
		}
		return b
	}
	usdt := func(total float64) accounts.Balance {
		return accounts.Balance{Currency: currency.USDT, Total: total, Free: total, AvailableWithoutBorrow: total}
	}
	btc := accounts.Balance{Currency: currency.BTC, Total: 1, Free: 1, AvailableWithoutBorrow: 1}
	listed := `{"coin":"USDT","walletBalance":"50"},{"coin":"BTC","walletBalance":"1"}`
	for _, tc := range []struct {
		name  string
		push  string
		coins string
		exp   accounts.CurrencyBalances
	}{
		// The push arrived after the snapshot was requested, so its USDT and its ETH, which the snapshot leaves out, stay
		{
			name: "a push arriving during the request", push: `{"coin":"USDT","walletBalance":"100"},{"coin":"ETH","walletBalance":"2"}`, coins: listed,
			exp: accounts.CurrencyBalances{currency.USDT: usdt(100), currency.BTC: btc, currency.ETH: {Currency: currency.ETH, Total: 2, Free: 2, AvailableWithoutBorrow: 2}},
		},
		{name: "a snapshot requested after the push", coins: listed, exp: accounts.CurrencyBalances{currency.USDT: usdt(50), currency.BTC: btc}},
		// Bybit lists no coin that holds nothing, so an empty reply still empties the coins stored before the request
		{name: "an empty reply after a push arriving during the request", push: `{"coin":"USDT","walletBalance":"70"}`, exp: accounts.CurrencyBalances{currency.USDT: usdt(70)}},
	} {
		mu.Lock()
		push, coins = tc.push, tc.coins
		mu.Unlock()
		subAccts, err := ex.UpdateAccountBalances(t.Context(), asset.Spot)
		require.NoErrorf(t, err, "UpdateAccountBalances must not error for %s", tc.name)
		require.Lenf(t, subAccts, 1, "UpdateAccountBalances must return the unified account for %s", tc.name)
		cached, err := ex.GetCachedCurrencyBalances(t.Context(), asset.Spot)
		require.NoErrorf(t, err, "GetCachedCurrencyBalances must not error for %s", tc.name)
		assert.Equalf(t, tc.exp, withoutTimes(cached), "UpdateAccountBalances should store the newest balances for %s", tc.name)
		assert.Equalf(t, tc.exp, withoutTimes(subAccts[0].Balances), "UpdateAccountBalances should return the balances it stores for %s", tc.name)
	}
}

func TestGetCachedSubAccounts(t *testing.T) {
	t.Parallel()
	ex := withMockCredentials(websocketTestInstance(t))
	_, err := ex.GetCachedSubAccounts(t.Context(), asset.Margin)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetCachedSubAccounts must reject an asset Bybit does not trade")
	unified, futuresCopy := accounts.NewSubAccount(asset.Spot, ""), accounts.NewSubAccount(asset.USDTMarginedFutures, "")
	unified.Balances.Set(currency.USDT, accounts.Balance{Total: 100})
	futuresCopy.Balances.Set(currency.USDT, accounts.Balance{Total: 7})
	require.NoError(t, ex.Accounts.Save(t.Context(), accounts.SubAccounts{unified, futuresCopy}, true), "Accounts.Save must not error")
	for _, a := range []asset.Item{asset.All, asset.Spot, asset.USDTMarginedFutures, asset.USDCMarginedFutures, asset.CoinMarginedFutures, asset.Options} {
		got, err := ex.GetCachedSubAccounts(t.Context(), a)
		require.NoErrorf(t, err, "GetCachedSubAccounts must not error for %s", a)
		assert.Equalf(t, accounts.SubAccounts{unified}, got, "GetCachedSubAccounts should return the unified account for %s", a)
	}
}

func TestGetCachedCurrencyBalances(t *testing.T) {
	t.Parallel()
	ex := withMockCredentials(websocketTestInstance(t))
	_, err := ex.GetCachedCurrencyBalances(t.Context(), asset.Margin)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetCachedCurrencyBalances must reject an asset Bybit does not trade")
	unified, futuresCopy := accounts.NewSubAccount(asset.Spot, ""), accounts.NewSubAccount(asset.USDTMarginedFutures, "")
	unified.Balances.Set(currency.USDT, accounts.Balance{Total: 100})
	futuresCopy.Balances.Set(currency.USDT, accounts.Balance{Total: 7})
	require.NoError(t, ex.Accounts.Save(t.Context(), accounts.SubAccounts{unified, futuresCopy}, true), "Accounts.Save must not error")
	for _, a := range []asset.Item{asset.All, asset.Spot, asset.USDTMarginedFutures, asset.USDCMarginedFutures, asset.CoinMarginedFutures, asset.Options} {
		got, err := ex.GetCachedCurrencyBalances(t.Context(), a)
		require.NoErrorf(t, err, "GetCachedCurrencyBalances must not error for %s", a)
		assert.Equalf(t, unified.Balances, got, "GetCachedCurrencyBalances should return the unified account's balances for %s", a)
	}
}

func TestUpdateAccountBalancesDropsUnreportedCoins(t *testing.T) {
	t.Parallel()
	// Without a coin filter Get Wallet Balance lists only the coins holding assets or liabilities, so a coin left out
	// of a later reply has nothing left and must not keep its earlier balance
	var mu sync.Mutex
	var calls int
	ex := newTransportTestExchange(t, func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		calls++
		first := calls == 1
		mu.Unlock()
		coins := `{"coin":"USDT","walletBalance":"100"}`
		if first {
			coins = `{"coin":"BTC","walletBalance":"0.5"},` + coins
		}
		_, err := fmt.Fprint(w, `{"retCode":0,"retMsg":"OK","result":{"list":[{"accountType":"UNIFIED","coin":[`+coins+`]}]},"retExtInfo":{},"time":1760000000000}`)
		assert.NoError(t, err, "writing the reply should not error")
	})
	for range 2 {
		_, err := ex.UpdateAccountBalances(t.Context(), asset.Spot)
		require.NoError(t, err, "UpdateAccountBalances must not error")
	}
	cached, err := ex.GetCachedCurrencyBalances(t.Context(), asset.Spot)
	require.NoError(t, err, "GetCachedCurrencyBalances must not error")
	assert.NotContains(t, cached, currency.BTC, "UpdateAccountBalances should drop a coin Bybit no longer reports")
	assert.Contains(t, cached, currency.USDT, "UpdateAccountBalances should keep the coins Bybit still reports")
}

func TestUnifiedBalance(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		coin WalletBalanceCoin
		exp  accounts.Balance
	}{
		{
			name: "no borrowing", coin: WalletBalanceCoin{WalletBalance: 0.80486234},
			exp: accounts.Balance{Total: 0.80486234, Free: 0.80486234, AvailableWithoutBorrow: 0.80486234},
		},
		{
			// The wallet balance includes what spot margin trading borrowed, which cannot be used without borrowing
			name: "spot borrowing", coin: WalletBalanceCoin{WalletBalance: 2.5, Locked: 0.5, BorrowAmount: 1.25, SpotBorrow: 1.25},
			exp: accounts.Balance{Total: 2.5, Hold: 0.5, Free: 2, Borrowed: 1.25, AvailableWithoutBorrow: 0.75},
		},
		{
			// Derivatives liabilities count as borrowed, but only spot borrowing sits in the wallet balance
			name: "derivatives borrowing", coin: WalletBalanceCoin{WalletBalance: 10, BorrowAmount: 4, SpotBorrow: 1},
			exp: accounts.Balance{Total: 10, Free: 10, Borrowed: 4, AvailableWithoutBorrow: 9},
		},
		{
			// Spot borrowing beyond what is free leaves nothing to use without borrowing
			name: "more spot borrowing than is free", coin: WalletBalanceCoin{WalletBalance: 1, Locked: 0.25, BorrowAmount: 1, SpotBorrow: 1},
			exp: accounts.Balance{Total: 1, Hold: 0.25, Free: 0.75, Borrowed: 1},
		},
		{
			// Derivatives orders and positions hold their initial margin as spot orders hold their locks, and the bonus
			// cannot be used without borrowing
			name: "derivatives margin and a bonus", coin: WalletBalanceCoin{WalletBalance: 1000, Locked: 50, TotalOrderInitialMargin: 100, TotalPositionInitialMargin: 600, Bonus: 25},
			exp: accounts.Balance{Total: 1000, Hold: 750, Free: 250, AvailableWithoutBorrow: 225},
		},
		{
			// In cross margin a coin's margin can exceed its wallet balance, which leaves nothing free
			name: "more margin than the wallet holds", coin: WalletBalanceCoin{WalletBalance: 500, TotalPositionInitialMargin: 600},
			exp: accounts.Balance{Total: 500, Hold: 600},
		},
		{
			// A negative wallet balance is a derivatives loss the account owes, which leaves nothing free
			name: "derivatives losses owed", coin: WalletBalanceCoin{WalletBalance: -12.5, BorrowAmount: 12.5},
			exp: accounts.Balance{Total: -12.5, Borrowed: 12.5},
		},
	} {
		assert.Equalf(t, tc.exp, unifiedBalance(&tc.coin), "unifiedBalance should return the balance of a wallet with %s", tc.name)
	}
}

func TestGetAccountFundingHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetAccountFundingHistory(t.Context())
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "GetAccountFundingHistory should not be supported")
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetWithdrawalsHistory(t.Context(), currency.BTC, asset.Futures)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetWithdrawalsHistory must reject an asset Bybit does not trade")
	refused, err := restReplyTestInstance(t, nil).GetWithdrawalsHistory(t.Context(), currency.BTC, asset.Spot)
	require.ErrorIs(t, err, errAPIResponse, "GetWithdrawalsHistory must return Bybit's refusal")
	assert.Nil(t, refused, "GetWithdrawalsHistory should return no withdrawals when Bybit refuses to list them")

	// The history ends at a page without a cursor, a page repeating the cursor it was asked with, or a page without
	// rows, whatever cursor the last two hold
	for _, tc := range []struct {
		name  string
		pages map[string]string
		exp   []string
	}{
		{
			name: "a page repeating its cursor",
			pages: map[string]string{
				"":   `{"rows":[{"withdrawId":"1"}],"nextPageCursor":"c1"}`,
				"c1": `{"rows":[{"withdrawId":"2"}],"nextPageCursor":"c1"}`,
			},
			exp: []string{"1", "2"},
		},
		{
			name: "a page without rows",
			pages: map[string]string{
				"":   `{"rows":[{"withdrawId":"1"}],"nextPageCursor":"c1"}`,
				"c1": `{"rows":[],"nextPageCursor":"c2"}`,
			},
			exp: []string{"1"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			history, err := newPagingTestExchange(t, tc.pages).GetWithdrawalsHistory(t.Context(), currency.BTC, asset.Spot)
			require.NoError(t, err, "GetWithdrawalsHistory must stop at the last page")
			ids := make([]string, len(history))
			for i := range history {
				ids[i] = history[i].TransferID
			}
			assert.Equal(t, tc.exp, ids, "GetWithdrawalsHistory should return the withdrawals of each page up to the last")
		})
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	history, err := e.GetWithdrawalsHistory(t.Context(), currency.BTC, asset.Spot)
	require.NoError(t, err, "GetWithdrawalsHistory must not error")
	if !mockTests {
		return
	}
	// Both on-chain withdrawals and internal transfers to other Bybit accounts
	exp := []exchange.WithdrawalHistory{
		{Status: "success", TransferID: "131629076", Currency: "BTC", Amount: 0.0015, Fee: 0.0001, CryptoToAddress: "XXXXXX", CryptoTxID: "3d7bddb797f0e86420c982c0723653b8b728fd0ec9953b6b354445848d83a185", CryptoChain: "BTC", Timestamp: time.UnixMilli(1742738340000)},
		{Status: "success", TransferID: "131603458", Currency: "BTC", Amount: 0.002, CryptoToAddress: "100000001", Timestamp: time.UnixMilli(1742729437000)},
		{Status: "success", TransferID: "131500001", Currency: "BTC", Amount: 0.01, Fee: 0.0002, CryptoToAddress: "XXXXXX", CryptoTxID: "9e0a1d6c3b5f4e2a8d7c6b5a4f3e2d1c0b9a8f7e6d5c4b3a2f1e0d9c8b7a6f5e", CryptoChain: "BTC", Timestamp: time.UnixMilli(1742700100000)},
	}
	assert.Equal(t, exp, history, "GetWithdrawalsHistory should return the withdrawals of every page")
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetRecentTrades(t.Context(), currency.NewBTCUSDT(), asset.Futures)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetRecentTrades must reject an asset Bybit does not trade")

	ensureTradablePairs(t)
	for _, tc := range []struct {
		asset asset.Item
		pair  currency.Pair
		exp   []trade.Data
	}{
		{
			// Spot serves at most 60 recent trades
			asset: asset.Spot,
			pair:  spotTradablePair,
			exp: []trade.Data{
				{TID: "2290000001231863417", Exchange: e.Name, CurrencyPair: currency.NewBTCUSDT(), AssetType: asset.Spot, Side: order.Buy, Price: 83482.3, Amount: 0.014733, Timestamp: time.UnixMilli(1791898238736)},
				{TID: "2290000001231863432", Exchange: e.Name, CurrencyPair: currency.NewBTCUSDT(), AssetType: asset.Spot, Side: order.Buy, Price: 83482.3, Amount: 0.00006, Timestamp: time.UnixMilli(1791898238959)},
				{TID: "2290000001231863435", Exchange: e.Name, CurrencyPair: currency.NewBTCUSDT(), AssetType: asset.Spot, Side: order.Buy, Price: 83486.3, Amount: 0.00006, Timestamp: time.UnixMilli(1791898238975)},
			},
		},
		{
			asset: asset.USDTMarginedFutures,
			pair:  usdtMarginedTradablePair,
			exp: []trade.Data{
				{TID: "d9fe7a5c-d718-5fd4-b6d4-cc456e3db511", Exchange: e.Name, CurrencyPair: currency.NewPairWithDelimiter("10000LADYS", "USDT", ""), AssetType: asset.USDTMarginedFutures, Side: order.Buy, Price: 0.000415, Amount: 259200, Timestamp: time.UnixMilli(1693086704010)},
				{TID: "3d4c6367-b5b1-5dee-8039-f0cdd4116226", Exchange: e.Name, CurrencyPair: currency.NewPairWithDelimiter("10000LADYS", "USDT", ""), AssetType: asset.USDTMarginedFutures, Side: order.Buy, Price: 0.000415, Amount: 237600, Timestamp: time.UnixMilli(1693086704012)},
			},
		},
		{
			asset: asset.USDCMarginedFutures,
			pair:  usdcMarginedTradablePair,
			exp: []trade.Data{
				{TID: "20788e62-3f02-58d7-9ac1-a92095126e4b", Exchange: e.Name, CurrencyPair: currency.NewPairWithDelimiter("ETH", "PERP", ""), AssetType: asset.USDCMarginedFutures, Side: order.Sell, Price: 2507.71, Amount: 0.03, Timestamp: time.UnixMilli(1791898234331)},
				{TID: "1b31198f-785d-582c-b88c-2fc8551229c5", Exchange: e.Name, CurrencyPair: currency.NewPairWithDelimiter("ETH", "PERP", ""), AssetType: asset.USDCMarginedFutures, Side: order.Sell, Price: 2507.33, Amount: 0.01, Timestamp: time.UnixMilli(1791898239352)},
			},
		},
		{
			asset: asset.CoinMarginedFutures,
			pair:  inverseTradablePair,
			exp: []trade.Data{
				{TID: "f3cf0c33-130d-55ea-876a-aac956f2e388", Exchange: e.Name, CurrencyPair: currency.NewPairWithDelimiter("ADA", "USD", ""), AssetType: asset.CoinMarginedFutures, Side: order.Sell, Price: 0.2351, Amount: 100, Timestamp: time.UnixMilli(1791898227546)},
				{TID: "e2c1017a-7676-574b-9f52-de2d5195b401", Exchange: e.Name, CurrencyPair: currency.NewPairWithDelimiter("ADA", "USD", ""), AssetType: asset.CoinMarginedFutures, Side: order.Buy, Price: 0.2352, Amount: 8, Timestamp: time.UnixMilli(1791898234183)},
			},
		},
		{
			// Option trades are requested with the option's base coin
			asset: asset.Options,
			pair:  optionsTradablePair,
			exp: []trade.Data{
				{TID: "f41969b7-bb1e-5dfa-8d41-56af7fb36631", Exchange: e.Name, CurrencyPair: currency.NewPairWithDelimiter("BTC", "26NOV24-92000-C", "-"), AssetType: asset.Options, Side: order.Sell, Price: 40, Amount: 1.46, Timestamp: time.UnixMilli(1692899613925)},
				{TID: "7cb8afae-d5cc-5833-ad14-6d6dc08c3d32", Exchange: e.Name, CurrencyPair: currency.NewPairWithDelimiter("BTC", "26NOV24-92000-C", "-"), AssetType: asset.Options, Side: order.Buy, Price: 35, Amount: 0.02, Timestamp: time.UnixMilli(1692943393092)},
			},
		},
	} {
		t.Run(tc.asset.String(), func(t *testing.T) {
			t.Parallel()
			trades, err := e.GetRecentTrades(t.Context(), tc.pair, tc.asset)
			require.NoError(t, err, "GetRecentTrades must not error")
			if mockTests {
				assert.Equal(t, tc.exp, trades, "GetRecentTrades should return the pair's trades, oldest first")
				return
			}
			// A live option can go without trades
			if tc.asset != asset.Options {
				assert.NotEmpty(t, trades, "GetRecentTrades should return trades")
			}
		})
	}

	// A USDC margined futures contract is asked for by its dashed symbol; Bybit lists none now, but still serves the
	// trades of the delivered BTC-26DEC25
	t.Run("USDC margined futures contract", func(t *testing.T) {
		t.Parallel()
		usdcFutures := currency.NewPairWithDelimiter("BTC", "26DEC25", currency.DashDelimiter)
		trades, err := e.GetRecentTrades(t.Context(), currency.NewPair(currency.BTC, currency.NewCode("26DEC25")), asset.USDCMarginedFutures)
		require.NoError(t, err, "GetRecentTrades must not error")
		if !mockTests {
			require.NotEmpty(t, trades, "GetRecentTrades must return the contract's trades")
			assert.Equal(t, usdcFutures, trades[0].CurrencyPair, "GetRecentTrades should return the contract's trades")
			return
		}
		exp := []trade.Data{
			{TID: "3b24043f-00de-50cd-b131-1e34c19f7b4e", Exchange: e.Name, CurrencyPair: usdcFutures, AssetType: asset.USDCMarginedFutures, Side: order.Buy, Price: 88750.5, Amount: 0.011, Timestamp: time.UnixMilli(1766735821433)},
			{TID: "86d2fcd3-5506-5449-9bcd-2973b332b51c", Exchange: e.Name, CurrencyPair: usdcFutures, AssetType: asset.USDCMarginedFutures, Side: order.Buy, Price: 88750.5, Amount: 0.01, Timestamp: time.UnixMilli(1766735821642)},
		}
		assert.Equal(t, exp, trades, "GetRecentTrades should return the contract's trades, oldest first")
	})

	ex := marketReplyInstance(t, nil)
	require.NoError(t, ex.CurrencyPairs.Store(asset.Margin, &currency.PairStore{AssetEnabled: true, RequestFormat: &currency.PairFormat{Uppercase: true}, ConfigFormat: &currency.PairFormat{Uppercase: true, Delimiter: "_"}}), "Store must not error")
	_, err = ex.GetRecentTrades(t.Context(), currency.NewBTCUSDT(), asset.Margin)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetRecentTrades must reject an asset Bybit has no trades for")

	const spotTrades = "/v5/market/recent-trade?category=spot&limit=60&symbol=BTCUSDT"
	ex = marketReplyInstance(t, map[string]string{spotTrades: marketErrorReply})
	_, err = ex.GetRecentTrades(t.Context(), currency.NewBTCUSDT(), asset.Spot)
	require.ErrorIs(t, err, errAPIResponse, "GetRecentTrades must return the error of a refused request")

	// Bybit's trades with the side of one left out
	ex = marketReplyInstance(t, map[string]string{spotTrades: `{"retCode":0,"retMsg":"OK","result":{"category":"spot","list":[{"execId":"2290000001231863435","symbol":"BTCUSDT","price":"83486.3","size":"0.00006","side":"","time":"1791898238975","isBlockTrade":false,"isRPITrade":false,"seq":"115367841488"}]},"retExtInfo":{},"time":1791898239934}`})
	_, err = ex.GetRecentTrades(t.Context(), currency.NewBTCUSDT(), asset.Spot)
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "GetRecentTrades must reject a trade without a side")

	// Saved trades go to the trade store, which refuses Bybit's trade with its price zeroed
	ex = marketReplyInstance(t, map[string]string{spotTrades: `{"retCode":0,"retMsg":"OK","result":{"category":"spot","list":[{"execId":"2290000001231863435","symbol":"BTCUSDT","price":"0","size":"0.00006","side":"Buy","time":"1791898238975","isBlockTrade":false,"isRPITrade":false,"seq":"115367841488"}]},"retExtInfo":{},"time":1791898239934}`})
	ex.SetSaveTradeDataStatus(true)
	_, err = ex.GetRecentTrades(t.Context(), currency.NewBTCUSDT(), asset.Spot)
	assert.ErrorContains(t, err, "received invalid trade data", "GetRecentTrades should report a trade the trade store refuses")
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetHistoricTrades(t.Context(), currency.NewBTCUSDT(), asset.Spot, time.Time{}, time.Time{})
	require.ErrorIs(t, err, common.ErrDateUnset, "GetHistoricTrades must require a time range")
	_, err = e.GetHistoricTrades(t.Context(), currency.NewBTCUSDT(), asset.Spot, time.UnixMilli(1693087633000), time.UnixMilli(1693087632000))
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetHistoricTrades must reject a start after the end")
	// The range ends ahead of now so that live trades made while the test runs stay within it
	start, end := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), time.Now().Add(time.Hour)
	_, err = e.GetHistoricTrades(t.Context(), currency.NewBTCUSDT(), asset.Futures, start, end)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetHistoricTrades must reject an asset Bybit does not trade")

	ensureTradablePairs(t)
	for _, tc := range []struct {
		asset asset.Item
		pair  currency.Pair
		exp   []trade.Data
	}{
		{
			asset: asset.Spot,
			pair:  spotTradablePair,
			exp: []trade.Data{
				{TID: "2290000001231863435", Exchange: e.Name, CurrencyPair: currency.NewBTCUSDT(), AssetType: asset.Spot, Side: order.Buy, Price: 83486.3, Amount: 0.00006, Timestamp: time.UnixMilli(1791898238975)},
				{TID: "2290000001231863432", Exchange: e.Name, CurrencyPair: currency.NewBTCUSDT(), AssetType: asset.Spot, Side: order.Buy, Price: 83482.3, Amount: 0.00006, Timestamp: time.UnixMilli(1791898238959)},
				{TID: "2290000001231863417", Exchange: e.Name, CurrencyPair: currency.NewBTCUSDT(), AssetType: asset.Spot, Side: order.Buy, Price: 83482.3, Amount: 0.014733, Timestamp: time.UnixMilli(1791898238736)},
			},
		},
		{
			asset: asset.USDTMarginedFutures,
			pair:  usdtMarginedTradablePair,
			exp: []trade.Data{
				{TID: "7edbd23b-a693-5666-8411-14f34644ad1d", Exchange: e.Name, CurrencyPair: currency.NewPairWithDelimiter("10000LADYS", "USDT", ""), AssetType: asset.USDTMarginedFutures, Side: order.Buy, Price: 0.000437, Amount: 136200, Timestamp: time.UnixMilli(1693087638127)},
				{TID: "dbb948b1-f6cf-5171-b419-edf96bb0889d", Exchange: e.Name, CurrencyPair: currency.NewPairWithDelimiter("10000LADYS", "USDT", ""), AssetType: asset.USDTMarginedFutures, Side: order.Sell, Price: 0.000436, Amount: 31600, Timestamp: time.UnixMilli(1693087638098)},
			},
		},
		{
			asset: asset.USDCMarginedFutures,
			pair:  usdcMarginedTradablePair,
			exp: []trade.Data{
				{TID: "1b31198f-785d-582c-b88c-2fc8551229c5", Exchange: e.Name, CurrencyPair: currency.NewPairWithDelimiter("ETH", "PERP", ""), AssetType: asset.USDCMarginedFutures, Side: order.Sell, Price: 2507.33, Amount: 0.01, Timestamp: time.UnixMilli(1791898239352)},
				{TID: "20788e62-3f02-58d7-9ac1-a92095126e4b", Exchange: e.Name, CurrencyPair: currency.NewPairWithDelimiter("ETH", "PERP", ""), AssetType: asset.USDCMarginedFutures, Side: order.Sell, Price: 2507.71, Amount: 0.03, Timestamp: time.UnixMilli(1791898234331)},
			},
		},
		{
			asset: asset.CoinMarginedFutures,
			pair:  inverseTradablePair,
			exp: []trade.Data{
				{TID: "e2c1017a-7676-574b-9f52-de2d5195b401", Exchange: e.Name, CurrencyPair: currency.NewPairWithDelimiter("ADA", "USD", ""), AssetType: asset.CoinMarginedFutures, Side: order.Buy, Price: 0.2352, Amount: 8, Timestamp: time.UnixMilli(1791898234183)},
				{TID: "f3cf0c33-130d-55ea-876a-aac956f2e388", Exchange: e.Name, CurrencyPair: currency.NewPairWithDelimiter("ADA", "USD", ""), AssetType: asset.CoinMarginedFutures, Side: order.Sell, Price: 0.2351, Amount: 100, Timestamp: time.UnixMilli(1791898227546)},
			},
		},
		{
			asset: asset.Options,
			pair:  optionsTradablePair,
			exp: []trade.Data{
				{TID: "7cb8afae-d5cc-5833-ad14-6d6dc08c3d32", Exchange: e.Name, CurrencyPair: currency.NewPairWithDelimiter("BTC", "26NOV24-92000-C", "-"), AssetType: asset.Options, Side: order.Buy, Price: 35, Amount: 0.02, Timestamp: time.UnixMilli(1692943393092)},
				{TID: "f41969b7-bb1e-5dfa-8d41-56af7fb36631", Exchange: e.Name, CurrencyPair: currency.NewPairWithDelimiter("BTC", "26NOV24-92000-C", "-"), AssetType: asset.Options, Side: order.Sell, Price: 40, Amount: 1.46, Timestamp: time.UnixMilli(1692899613925)},
			},
		},
	} {
		t.Run(tc.asset.String(), func(t *testing.T) {
			t.Parallel()
			trades, err := e.GetHistoricTrades(t.Context(), tc.pair, tc.asset, start, end)
			require.NoError(t, err, "GetHistoricTrades must not error")
			if mockTests {
				assert.Equal(t, tc.exp, trades, "GetHistoricTrades should return the pair's trades in the time range")
				return
			}
			if tc.asset != asset.Options {
				assert.NotEmpty(t, trades, "GetHistoricTrades should return the recent trades")
			}
			for i := range trades {
				assert.Truef(t, trades[i].Timestamp.After(start) && trades[i].Timestamp.Before(end), "GetHistoricTrades should return only trades within the time range, not %s", trades[i].Timestamp)
			}
		})
	}

	// A USDC margined futures contract is asked for by its dashed symbol; Bybit lists none now, but still serves the
	// trades of the delivered BTC-26DEC25
	t.Run("USDC margined futures contract", func(t *testing.T) {
		t.Parallel()
		usdcFutures := currency.NewPairWithDelimiter("BTC", "26DEC25", currency.DashDelimiter)
		trades, err := e.GetHistoricTrades(t.Context(), currency.NewPair(currency.BTC, currency.NewCode("26DEC25")), asset.USDCMarginedFutures, start, end)
		require.NoError(t, err, "GetHistoricTrades must not error")
		if !mockTests {
			require.NotEmpty(t, trades, "GetHistoricTrades must return the contract's trades")
			assert.Equal(t, usdcFutures, trades[0].CurrencyPair, "GetHistoricTrades should return the contract's trades")
			return
		}
		exp := []trade.Data{
			{TID: "86d2fcd3-5506-5449-9bcd-2973b332b51c", Exchange: e.Name, CurrencyPair: usdcFutures, AssetType: asset.USDCMarginedFutures, Side: order.Buy, Price: 88750.5, Amount: 0.01, Timestamp: time.UnixMilli(1766735821642)},
			{TID: "3b24043f-00de-50cd-b131-1e34c19f7b4e", Exchange: e.Name, CurrencyPair: usdcFutures, AssetType: asset.USDCMarginedFutures, Side: order.Buy, Price: 88750.5, Amount: 0.011, Timestamp: time.UnixMilli(1766735821433)},
		}
		assert.Equal(t, exp, trades, "GetHistoricTrades should return the contract's trades in the time range")
	})

	ex := marketReplyInstance(t, nil)
	require.NoError(t, ex.CurrencyPairs.Store(asset.Margin, &currency.PairStore{AssetEnabled: true, RequestFormat: &currency.PairFormat{Uppercase: true}, ConfigFormat: &currency.PairFormat{Uppercase: true, Delimiter: "_"}}), "Store must not error")
	_, err = ex.GetHistoricTrades(t.Context(), currency.NewBTCUSDT(), asset.Margin, start, end)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetHistoricTrades must reject an asset Bybit has no trades for")

	for _, tc := range []struct {
		name    string
		asset   asset.Item
		pair    currency.Pair
		replies map[string]string
		err     error
	}{
		{name: "trades refused", asset: asset.Spot, pair: currency.NewBTCUSDT(), replies: map[string]string{"/v5/market/recent-trade?category=spot&limit=60&symbol=BTCUSDT": marketErrorReply}, err: errAPIResponse},
		{name: "option trades refused", asset: asset.Options, pair: fixtureOptionPair, replies: map[string]string{"/v5/market/recent-trade?baseCoin=BTC&category=option&limit=1000&symbol=BTC-24SEP27-170000-P-USDT": marketErrorReply}, err: errAPIResponse},
		{
			// Bybit's trades with the side of one left out
			name: "trade without a side", asset: asset.Spot, pair: currency.NewBTCUSDT(), err: order.ErrSideIsInvalid,
			replies: map[string]string{"/v5/market/recent-trade?category=spot&limit=60&symbol=BTCUSDT": `{"retCode":0,"retMsg":"OK","result":{"category":"spot","list":[{"execId":"2290000001231863435","symbol":"BTCUSDT","price":"83486.3","size":"0.00006","side":"","time":"1791898238975","isBlockTrade":false,"isRPITrade":false,"seq":"115367841488"}]},"retExtInfo":{},"time":1791898239934}`},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := marketReplyInstance(t, tc.replies).GetHistoricTrades(t.Context(), tc.pair, tc.asset, start, end)
			assert.ErrorIs(t, err, tc.err, "GetHistoricTrades should return the error")
		})
	}

	trades, err := e.GetHistoricTrades(t.Context(), spotTradablePair, asset.Spot, time.UnixMilli(1), time.UnixMilli(2))
	require.NoError(t, err, "GetHistoricTrades must not error")
	assert.Empty(t, trades, "GetHistoricTrades should leave out trades before the time range")
	if !mockTests {
		return
	}
	trades, err = e.GetHistoricTrades(t.Context(), spotTradablePair, asset.Spot, time.UnixMilli(1791898238950), time.UnixMilli(1791898238970))
	require.NoError(t, err, "GetHistoricTrades must not error")
	exp := []trade.Data{{TID: "2290000001231863432", Exchange: e.Name, CurrencyPair: currency.NewBTCUSDT(), AssetType: asset.Spot, Side: order.Buy, Price: 83482.3, Amount: 0.00006, Timestamp: time.UnixMilli(1791898238959)}}
	assert.Equal(t, exp, trades, "GetHistoricTrades should leave out the trades on either side of the time range")
}

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	_, err := e.SubmitOrder(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrSubmissionIsNil, "SubmitOrder must reject a nil order")
	_, err = e.SubmitOrder(t.Context(), &order.Submit{Exchange: e.Name, Pair: currency.NewBTCUSDT(), AssetType: asset.Spot, Side: order.Buy, Type: order.Market, Amount: 0.001})
	require.ErrorIs(t, err, order.ErrAmountMustBeSet, "SubmitOrder must require a quote amount for a spot market buy")
	_, err = e.SubmitOrder(t.Context(), &order.Submit{Exchange: e.Name, Pair: currency.NewBTCUSDT(), AssetType: asset.Binary, Side: order.Buy, Type: order.Limit, Price: 1, Amount: 1})
	require.ErrorIs(t, err, asset.ErrNotSupported, "SubmitOrder must reject an asset Bybit does not trade")
	_, err = e.SubmitOrder(t.Context(), &order.Submit{Exchange: e.Name, Pair: currency.NewBTCUSDT(), AssetType: asset.USDTMarginedFutures, Side: order.Buy, Type: order.Market, Amount: 1, TimeInForce: order.FillOrKill})
	require.ErrorIs(t, err, order.ErrUnsupportedTimeInForce, "SubmitOrder must refuse a time in force Bybit cannot honour before sending")
	noOrderID := restReplyTestInstance(t, map[string]string{"/v5/order/create": `{"retCode":0,"retMsg":"OK","result":{"orderId":"","orderLinkId":"spot-test-10"},"retExtInfo":{},"time":1760000000000}`})
	_, err = noOrderID.SubmitOrder(t.Context(), &order.Submit{Exchange: noOrderID.Name, Pair: currency.NewBTCUSDT(), AssetType: asset.Spot, Side: order.Buy, Type: order.Limit, Price: 1, Amount: 1, ClientOrderID: "spot-test-10"})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "SubmitOrder must not report an order placed without Bybit's order ID")
	// A spot limit order with a trigger price is placed as a TP/SL order, so it is stored as a stop limit order and its
	// cancellation names the TP/SL filter
	conditional := restReplyTestInstance(t, map[string]string{"/v5/order/create": `{"retCode":0,"retMsg":"OK","result":{"orderId":"1321003749386327599","orderLinkId":"spot-tpsl-01"},"retExtInfo":{},"time":1760000000000}`})
	submitted, err := conditional.SubmitOrder(t.Context(), &order.Submit{Exchange: conditional.Name, Pair: currency.NewBTCUSDT(), AssetType: asset.Spot, Side: order.Sell, Type: order.Limit, Price: 89000, Amount: 0.01, TriggerPrice: 90000, ClientOrderID: "spot-tpsl-01"})
	require.NoError(t, err, "SubmitOrder must not error for a limit order with a trigger price")
	assert.Equal(t, order.StopLimit, submitted.Type, "SubmitOrder should report a limit order with a trigger price as a stop limit order")
	stored, err := submitted.DeriveDetail(uuid.Nil())
	require.NoError(t, err, "DeriveDetail must not error")
	cancel, err := stored.DeriveCancel()
	require.NoError(t, err, "DeriveCancel must not error")
	cancelArg, err := conditional.deriveCancelOrderArguments(cancel)
	require.NoError(t, err, "deriveCancelOrderArguments must not error")
	assert.Equal(t, "tpslOrder", cancelArg.OrderFilter, "the cancellation of a stored spot conditional order should name the TP/SL filter")
	// Bybit requires a client order ID on every options order, so SubmitOrder returns the one it generated and sent
	var mu sync.Mutex
	var sentLinkID string
	generated := newTransportTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			OrderLinkID string `json:"orderLinkId"`
		}
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&body), "decoding the order should not error")
		mu.Lock()
		sentLinkID = body.OrderLinkID
		mu.Unlock()
		_, err := fmt.Fprint(w, `{"retCode":0,"retMsg":"OK","result":{"orderId":"9f1a3b5c-7d9e-4f1a-8b3c-5d7e9f1a3b5d","orderLinkId":""},"retExtInfo":{},"time":1760000000000}`)
		assert.NoError(t, err, "writing the reply should not error")
	})
	require.NoError(t, generated.DisableRateLimiter(), "DisableRateLimiter must not error")
	placed, err := generated.SubmitOrder(t.Context(), &order.Submit{Exchange: generated.Name, Pair: currency.NewPairWithDelimiter("BTC", "26NOV24-92000-C", "-"), AssetType: asset.Options, Side: order.Sell, Type: order.Limit, Price: 500, Amount: 0.01})
	require.NoError(t, err, "SubmitOrder must not error for an options order without a client order ID")
	mu.Lock()
	assert.NotEmpty(t, sentLinkID, "SubmitOrder should send a client order ID with an options order")
	assert.Equal(t, sentLinkID, placed.ClientOrderID, "SubmitOrder should return the client order ID it sent")
	mu.Unlock()

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		ensureTradablePairs(t)
		// A buy below the market rests on the book until it is cancelled; Bybit refuses prices too far from the market
		tick, err := e.UpdateTicker(t.Context(), spotTradablePair, asset.Spot)
		require.NoError(t, err, "UpdateTicker must not error")
		s := &order.Submit{Exchange: e.Name, Pair: spotTradablePair, AssetType: asset.Spot, Side: order.Buy, Type: order.Limit, Price: math.Floor(tick.Last * 0.97), Amount: 0.0001}
		resp, err := e.SubmitOrder(t.Context(), s)
		require.NoError(t, err, "SubmitOrder must not error")
		require.NotEmpty(t, resp.OrderID, "SubmitOrder must return the order ID")
		require.NoError(t, e.CancelOrder(t.Context(), &order.Cancel{OrderID: resp.OrderID, Pair: s.Pair, AssetType: s.AssetType}), "CancelOrder must not error")
		return
	}

	// Bybit refuses an order the account cannot pay for
	_, err = e.SubmitOrder(t.Context(), &order.Submit{Exchange: e.Name, Pair: spotTradablePair, AssetType: asset.Spot, Side: order.Sell, Type: order.Limit, Price: 90000, Amount: 10})
	require.ErrorIs(t, err, errAPIResponse, "SubmitOrder must return Bybit's refusal")
	assert.ErrorContains(t, err, "Balance insufficient", "SubmitOrder should return Bybit's reason")
	for _, tc := range []struct {
		name   string
		submit *order.Submit
		exp    *order.SubmitResponse
	}{
		{
			name:   "spot limit",
			submit: &order.Submit{Exchange: e.Name, Pair: spotTradablePair, AssetType: asset.Spot, Side: order.Buy, Type: order.Limit, Price: 1, Amount: 1, ClientOrderID: "1234"},
			exp:    &order.SubmitResponse{Exchange: e.Name, Pair: spotTradablePair, AssetType: asset.Spot, Side: order.Buy, Type: order.Limit, Price: 1, Amount: 1, ClientOrderID: "1234", Status: order.New, OrderID: "1321003749386327580"},
		},
		{
			// Bybit takes a spot market buy's size in the quote currency
			name:   "spot market buy",
			submit: &order.Submit{Exchange: e.Name, Pair: spotTradablePair, AssetType: asset.Spot, Side: order.Buy, Type: order.Market, QuoteAmount: 25, ClientOrderID: "spot-buy-001"},
			exp:    &order.SubmitResponse{Exchange: e.Name, Pair: spotTradablePair, AssetType: asset.Spot, Side: order.Buy, Type: order.Market, QuoteAmount: 25, ClientOrderID: "spot-buy-001", Status: order.New, OrderID: "1321003749386327581"},
		},
		{
			// A limit take profit makes the take profit and stop loss apply to the filled quantity, Bybit's partial mode
			name: "USDT margined futures limit with take profit and stop loss",
			submit: &order.Submit{
				Exchange: e.Name, Pair: currency.NewBTCUSDT(), AssetType: asset.USDTMarginedFutures, Side: order.Buy, Type: order.Limit, Price: 80000, Amount: 0.01, ClientOrderID: "linear-tp-001",
				RiskManagementModes: order.RiskManagementModes{
					TakeProfit: order.RiskManagement{Price: 90000, TriggerPriceType: order.MarkPrice, OrderType: order.Limit, LimitPrice: 89990},
					StopLoss:   order.RiskManagement{Price: 75000, TriggerPriceType: order.IndexPrice, OrderType: order.Market},
				},
			},
			exp: &order.SubmitResponse{Exchange: e.Name, Pair: currency.NewBTCUSDT(), AssetType: asset.USDTMarginedFutures, Side: order.Buy, Type: order.Limit, Price: 80000, Amount: 0.01, ClientOrderID: "linear-tp-001", Status: order.New, OrderID: "c1956690-b731-4191-97c0-94b00422231b"},
		},
		{
			name:   "options market",
			submit: &order.Submit{Exchange: e.Name, Pair: optionsTradablePair, AssetType: asset.Options, Side: order.Sell, Type: order.Market, Price: 1, Amount: 1, Leverage: 1234, ClientOrderID: "1234"},
			exp:    &order.SubmitResponse{Exchange: e.Name, Pair: optionsTradablePair, AssetType: asset.Options, Side: order.Sell, Type: order.Market, Price: 1, Amount: 1, Leverage: 1234, ClientOrderID: "1234", Status: order.New, OrderID: "9f1a3b5c-7d9e-4f1a-8b3c-5d7e9f1a3b5c"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			resp, err := e.SubmitOrder(t.Context(), tc.submit)
			require.NoError(t, err, "SubmitOrder must not error")
			assert.NotZero(t, resp.Date, "SubmitOrder should date the order")
			resp.Date, resp.LastUpdated = time.Time{}, time.Time{}
			assert.Equal(t, tc.exp, resp, "SubmitOrder should return Bybit's order ID with the order")
		})
	}
}

func TestWebsocketOrderEntrySupported(t *testing.T) {
	t.Parallel()
	// Bybit's websocket order entry trades spot, USDT and USDC contracts, inverse contracts and options
	for _, a := range []asset.Item{asset.Spot, asset.USDTMarginedFutures, asset.USDCMarginedFutures, asset.CoinMarginedFutures, asset.Options} {
		assert.NoErrorf(t, websocketOrderEntrySupported(a), "websocketOrderEntrySupported should accept %s", a)
	}
	// Bybit's websocket order entry does not take spread trading, nor assets the exchange does not trade
	for _, a := range []asset.Item{asset.Spread, asset.Margin, asset.Futures} {
		assert.ErrorIsf(t, websocketOrderEntrySupported(a), common.ErrFunctionNotSupported, "websocketOrderEntrySupported should refuse %s", a)
	}
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	_, err := e.ModifyOrder(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrModifyOrderIsNil, "ModifyOrder must reject a nil modification")
	_, err = e.ModifyOrder(t.Context(), &order.Modify{Pair: currency.NewBTCUSDT(), AssetType: asset.Spot, Price: 1})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "ModifyOrder must require an order ID or client order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		ensureTradablePairs(t)
		_, err = e.ModifyOrder(t.Context(), &order.Modify{OrderID: "1234", Type: order.Limit, Side: order.Buy, AssetType: asset.Spot, Pair: spotTradablePair, Price: 1234, Amount: 0.15})
		require.ErrorIs(t, err, order.ErrOrderNotFound, "ModifyOrder must report an unknown order as not found")
		return
	}

	// A filled order can no longer be amended
	_, err = e.ModifyOrder(t.Context(), &order.Modify{Exchange: e.Name, OrderID: "c1956690-b731-4191-97c0-94b00422231b", AssetType: asset.USDTMarginedFutures, Pair: currency.NewBTCUSDT(), Price: 82000})
	require.ErrorIs(t, err, order.ErrOrderNotFound, "ModifyOrder must report an order Bybit cannot find")
	for _, tc := range []struct {
		name   string
		modify *order.Modify
		exp    *order.ModifyResponse
	}{
		{
			name:   "options by order ID",
			modify: &order.Modify{Exchange: e.Name, OrderID: "8d3a3cbd-14ac-4f3e-a72b-3a5b1a4d1f9a", Type: order.Limit, Side: order.Buy, AssetType: asset.Options, Pair: optionsTradablePair, Price: 1234, Amount: 0.15},
			exp:    &order.ModifyResponse{Exchange: e.Name, OrderID: "8d3a3cbd-14ac-4f3e-a72b-3a5b1a4d1f9a", Type: order.Limit, Side: order.Buy, AssetType: asset.Options, Pair: optionsTradablePair, Price: 1234, Amount: 0.15},
		},
		{
			// Each trigger type is sent with the price it applies to; Bybit returns the order ID of a client order ID
			name: "USDT margined futures by client order ID",
			modify: &order.Modify{
				Exchange: e.Name, ClientOrderID: "linear-tp-001", AssetType: asset.USDTMarginedFutures, Pair: currency.NewBTCUSDT(), Price: 81000, Amount: 2,
				TriggerPrice: 80500, TriggerPriceType: order.IndexPrice,
				RiskManagementModes: order.RiskManagementModes{
					TakeProfit: order.RiskManagement{Price: 90000, TriggerPriceType: order.MarkPrice, OrderType: order.Limit, LimitPrice: 89990},
					StopLoss:   order.RiskManagement{Price: 75000, LimitPrice: 74990},
				},
			},
			exp: &order.ModifyResponse{Exchange: e.Name, OrderID: "c1956690-b731-4191-97c0-94b00422231b", ClientOrderID: "linear-tp-001", AssetType: asset.USDTMarginedFutures, Pair: currency.NewBTCUSDT(), Price: 81000, Amount: 2, TriggerPrice: 80500},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			resp, err := e.ModifyOrder(t.Context(), tc.modify)
			require.NoError(t, err, "ModifyOrder must not error")
			assert.Equal(t, tc.exp, resp, "ModifyOrder should return the amended order")
		})
	}
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	err := e.CancelOrder(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrCancelOrderIsNil, "CancelOrder must reject a nil cancellation")
	err = e.CancelOrder(t.Context(), &order.Cancel{Pair: currency.NewBTCUSDT(), AssetType: asset.Spot})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelOrder must require an order ID or client order ID")
	err = e.CancelOrder(t.Context(), &order.Cancel{OrderID: "1321003749386327552", AssetType: asset.Spot})
	require.ErrorIs(t, err, order.ErrPairIsEmpty, "CancelOrder must require a pair")
	// A spot conditional order placed with Bybit's StopOrder filter reads back as a stop order, as a TP/SL order does, so
	// a stop order Bybit cannot find among TP/SL orders is looked for among conditional orders
	for _, tc := range []struct {
		name    string
		cancel  *order.Cancel
		found   string
		filters []string
		err     error
	}{
		{name: "a TP/SL order", cancel: &order.Cancel{AssetType: asset.Spot, Pair: currency.NewBTCUSDT(), OrderID: "1", Type: order.StopLimit}, found: "tpslOrder", filters: []string{"tpslOrder"}},
		{name: "a conditional order", cancel: &order.Cancel{AssetType: asset.Spot, Pair: currency.NewBTCUSDT(), OrderID: "1", Type: order.StopMarket}, found: "StopOrder", filters: []string{"tpslOrder", "StopOrder"}},
		{name: "a stop order under neither filter", cancel: &order.Cancel{AssetType: asset.Spot, Pair: currency.NewBTCUSDT(), OrderID: "1", Type: order.StopLimit}, filters: []string{"tpslOrder", "StopOrder"}, err: order.ErrOrderNotFound},
		{name: "a plain spot order", cancel: &order.Cancel{AssetType: asset.Spot, Pair: currency.NewBTCUSDT(), OrderID: "1", Type: order.Limit}, filters: []string{""}, err: order.ErrOrderNotFound},
		{name: "a futures stop order", cancel: &order.Cancel{AssetType: asset.USDTMarginedFutures, Pair: currency.NewBTCUSDT(), OrderID: "1", Type: order.StopMarket}, filters: []string{""}, err: order.ErrOrderNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var mu sync.Mutex
			var filters []string
			ex := newTransportTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
				var body struct {
					OrderFilter string `json:"orderFilter"`
				}
				assert.NoError(t, json.NewDecoder(r.Body).Decode(&body), "decoding the cancellation should not error")
				mu.Lock()
				filters = append(filters, body.OrderFilter)
				mu.Unlock()
				reply := `{"retCode":110001,"retMsg":"Order does not exist","result":{},"retExtInfo":{},"time":1760000000000}`
				if tc.found != "" && body.OrderFilter == tc.found {
					reply = `{"retCode":0,"retMsg":"OK","result":{"orderId":"1","orderLinkId":""},"retExtInfo":{},"time":1760000000000}`
				}
				_, err := fmt.Fprint(w, reply)
				assert.NoError(t, err, "writing the reply should not error")
			})
			require.NoError(t, ex.DisableRateLimiter(), "DisableRateLimiter must not error")
			assert.ErrorIs(t, ex.CancelOrder(t.Context(), tc.cancel), tc.err, "CancelOrder should return the outcome of the cancellation Bybit found")
			mu.Lock()
			defer mu.Unlock()
			assert.Equal(t, tc.filters, filters, "CancelOrder should look for the order under each filter it can be placed with")
		})
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		ensureTradablePairs(t)
		err = e.CancelOrder(t.Context(), &order.Cancel{Exchange: e.Name, AssetType: asset.USDTMarginedFutures, Pair: usdtMarginedTradablePair, OrderID: "1234"})
		require.ErrorIs(t, err, order.ErrOrderNotFound, "CancelOrder must report an unknown order as not found")
		return
	}

	// A filled order can no longer be cancelled
	err = e.CancelOrder(t.Context(), &order.Cancel{Exchange: e.Name, AssetType: asset.USDTMarginedFutures, Pair: currency.NewBTCUSDT(), OrderID: "c1956690-b731-4191-97c0-94b00422231b"})
	require.ErrorIs(t, err, order.ErrOrderNotFound, "CancelOrder must report an order Bybit cannot find")
	for _, tc := range []struct {
		name   string
		cancel *order.Cancel
	}{
		{name: "spot", cancel: &order.Cancel{Exchange: e.Name, AssetType: asset.Spot, Pair: spotTradablePair, OrderID: "1321003749386327552"}},
		{name: "spot by client order ID", cancel: &order.Cancel{Exchange: e.Name, AssetType: asset.Spot, Pair: spotTradablePair, ClientOrderID: "spot-test-01"}},
		{name: "USDT margined futures", cancel: &order.Cancel{Exchange: e.Name, AssetType: asset.USDTMarginedFutures, Pair: usdtMarginedTradablePair, OrderID: "fd4300ae-7847-404e-b947-b46980a4d140"}},
		{name: "USDC margined futures", cancel: &order.Cancel{Exchange: e.Name, AssetType: asset.USDCMarginedFutures, Pair: usdcMarginedTradablePair, OrderID: "9a3c6f8e-5d21-4b7a-8c3e-1f2d4a6b8c90"}},
		{name: "coin margined futures", cancel: &order.Cancel{Exchange: e.Name, AssetType: asset.CoinMarginedFutures, Pair: inverseTradablePair, OrderID: "8e1a2b3c-4d5e-4f60-8172-93a4b5c6d7e8"}},
		{name: "options", cancel: &order.Cancel{Exchange: e.Name, AssetType: asset.Options, Pair: optionsTradablePair, OrderID: "8d3a3cbd-14ac-4f3e-a72b-3a5b1a4d1f9a"}},
	} {
		assert.NoErrorf(t, e.CancelOrder(t.Context(), tc.cancel), "CancelOrder should not error for %s", tc.name)
	}
}

func TestCancelWithStopOrderFallback(t *testing.T) {
	t.Parallel()
	notFound := &APIError{Code: 170213, Message: "Order does not exist."}
	for _, tc := range []struct {
		name    string
		filter  string
		replies map[string]error
		filters []string
		err     error
	}{
		{name: "a TP/SL order", filter: "tpslOrder", filters: []string{"tpslOrder"}},
		{name: "a conditional order", filter: "tpslOrder", replies: map[string]error{"tpslOrder": notFound}, filters: []string{"tpslOrder", "StopOrder"}},
		{name: "an order under neither filter", filter: "tpslOrder", replies: map[string]error{"tpslOrder": notFound, "StopOrder": notFound}, filters: []string{"tpslOrder", "StopOrder"}, err: order.ErrOrderNotFound},
		{name: "a conditional order cancellation failing", filter: "tpslOrder", replies: map[string]error{"tpslOrder": notFound, "StopOrder": context.DeadlineExceeded}, filters: []string{"tpslOrder", "StopOrder"}, err: context.DeadlineExceeded},
		{name: "a TP/SL order cancellation failing otherwise", filter: "tpslOrder", replies: map[string]error{"tpslOrder": context.DeadlineExceeded}, filters: []string{"tpslOrder"}, err: context.DeadlineExceeded},
		{name: "an order not found without the TP/SL filter", replies: map[string]error{"": notFound}, filters: []string{""}, err: order.ErrOrderNotFound},
		{name: "a conditional order not found", filter: "StopOrder", replies: map[string]error{"StopOrder": notFound}, filters: []string{"StopOrder"}, err: order.ErrOrderNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var filters []string
			send := func(_ context.Context, r *CancelOrderRequest) (*OrderResponse, error) {
				filters = append(filters, r.OrderFilter)
				return &OrderResponse{}, tc.replies[r.OrderFilter]
			}
			err := cancelWithStopOrderFallback(t.Context(), &CancelOrderRequest{OrderFilter: tc.filter}, send)
			assert.ErrorIs(t, err, tc.err, "cancelWithStopOrderFallback should return the last cancellation's outcome")
			assert.Equal(t, tc.filters, filters, "cancelWithStopOrderFallback should resend only a TP/SL cancellation Bybit finds no order for")
		})
	}
}

func TestCancelBatchOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelBatchOrders(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrCancelOrderIsNil, "CancelBatchOrders must reject no cancellations")
	for _, tc := range []struct {
		name   string
		cancel order.Cancel
		err    error
	}{
		{name: "an order without an ID", cancel: order.Cancel{Pair: currency.NewBTCUSDT(), AssetType: asset.Spot}, err: order.ErrOrderIDNotSet},
		{name: "an order without a pair", cancel: order.Cancel{OrderID: "1321003749386327552", AssetType: asset.Spot}, err: order.ErrPairIsEmpty},
		{name: "an order without an asset", cancel: order.Cancel{OrderID: "1321003749386327552", Pair: currency.NewBTCUSDT()}, err: order.ErrAssetNotSet},
		{name: "an asset Bybit does not trade", cancel: order.Cancel{OrderID: "1321003749386327552", Pair: currency.NewBTCUSDT(), AssetType: asset.Margin}, err: asset.ErrNotSupported},
	} {
		_, err = e.CancelBatchOrders(t.Context(), []order.Cancel{{OrderID: "1321003749386327552", Pair: currency.NewBTCUSDT(), AssetType: asset.Spot}, tc.cancel})
		require.ErrorIsf(t, err, tc.err, "CancelBatchOrders must reject %s before sending anything", tc.name)
	}
	option := currency.NewPairWithDelimiter("BTC", "26NOV24-92000-C", "-")
	_, err = noRequestFormatTestInstance(t, asset.Options).CancelBatchOrders(t.Context(), []order.Cancel{{OrderID: "3f8a2c1e-5b7d-4e9f-a1c3-5e7f9a1b3c01", Pair: option, AssetType: asset.Options}})
	require.ErrorIs(t, err, currency.ErrPairFormatIsNil, "CancelBatchOrders must not send an order whose symbol it cannot write")
	refused, err := restReplyTestInstance(t, nil).CancelBatchOrders(t.Context(), []order.Cancel{{OrderID: "3f8a2c1e-5b7d-4e9f-a1c3-5e7f9a1b3c01", Pair: option, AssetType: asset.Options}})
	require.ErrorIs(t, err, errAPIResponse, "CancelBatchOrders must return Bybit's refusal")
	require.NotNil(t, refused, "CancelBatchOrders must return the outcomes reported before Bybit refused")
	assert.Empty(t, refused.Status, "CancelBatchOrders should report no outcome for orders Bybit refused to cancel")
	// Bybit reports each order's outcome in request order, so a reply that leaves orders out cannot be matched to them
	short := restReplyTestInstance(t, map[string]string{"/v5/order/cancel-batch": `{"retCode":0,"retMsg":"OK","result":{"list":[{"category":"option","symbol":"BTC-26NOV24-92000-C","orderId":"3f8a2c1e-5b7d-4e9f-a1c3-5e7f9a1b3c01","orderLinkId":""}]},"retExtInfo":{"list":[{"code":0,"msg":"OK"}]},"time":1760000000000}`})
	_, err = short.CancelBatchOrders(t.Context(), []order.Cancel{
		{OrderID: "3f8a2c1e-5b7d-4e9f-a1c3-5e7f9a1b3c01", Pair: option, AssetType: asset.Options},
		{OrderID: "3f8a2c1e-5b7d-4e9f-a1c3-5e7f9a1b3c02", Pair: option, AssetType: asset.Options},
	})
	require.ErrorIs(t, err, errBatchOutcomeMismatch, "CancelBatchOrders must reject a reply that leaves out some of its orders")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		ensureTradablePairs(t)
		resp, err := e.CancelBatchOrders(t.Context(), []order.Cancel{{OrderID: "1", AccountID: "1", Pair: optionsTradablePair, AssetType: asset.Options}, {OrderID: "2", Pair: optionsTradablePair, AssetType: asset.Options}})
		require.NoError(t, err, "CancelBatchOrders must not error")
		require.Len(t, resp.Status, 2, "CancelBatchOrders must report each order's outcome")
		for id, status := range resp.Status {
			assert.NotEqualf(t, "success", status, "CancelBatchOrders should report that unknown order %s was not cancelled", id)
		}
		return
	}

	optionIDs := []string{"3f8a2c1e-5b7d-4e9f-a1c3-5e7f9a1b3c01", "3f8a2c1e-5b7d-4e9f-a1c3-5e7f9a1b3c02", "3f8a2c1e-5b7d-4e9f-a1c3-5e7f9a1b3c03", "3f8a2c1e-5b7d-4e9f-a1c3-5e7f9a1b3c04", "3f8a2c1e-5b7d-4e9f-a1c3-5e7f9a1b3c05", "3f8a2c1e-5b7d-4e9f-a1c3-5e7f9a1b3c06"}
	options := make([]order.Cancel, len(optionIDs))
	optionStatus := make(map[string]string, len(optionIDs))
	for i, id := range optionIDs {
		options[i] = order.Cancel{OrderID: id, Pair: optionsTradablePair, AssetType: asset.Options}
		optionStatus[id] = "success"
	}
	for _, tc := range []struct {
		name   string
		cancel []order.Cancel
		exp    *order.CancelBatchResponse
	}{
		{
			// Each category is cancelled in a request of its own, and Bybit reports each order's outcome
			name: "spot and USDT margined futures",
			cancel: []order.Cancel{
				{OrderID: "1321003749386327552", Pair: spotTradablePair, AssetType: asset.Spot},
				{OrderID: "fd4300ae-7847-404e-b947-b46980a4d140", Pair: usdtMarginedTradablePair, AssetType: asset.USDTMarginedFutures},
				{ClientOrderID: "spot-test-09", Pair: spotTradablePair, AssetType: asset.Spot},
			},
			exp: &order.CancelBatchResponse{Status: map[string]string{
				"1321003749386327552":                  "success",
				"fd4300ae-7847-404e-b947-b46980a4d140": "success",
				"spot-test-09":                         "Order does not exist.",
			}},
		},
		{
			// Bybit cancels at most five options a request
			name:   "six options",
			cancel: options,
			exp:    &order.CancelBatchResponse{Status: optionStatus},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			resp, err := e.CancelBatchOrders(t.Context(), tc.cancel)
			require.NoError(t, err, "CancelBatchOrders must not error")
			assert.Equal(t, tc.exp, resp, "CancelBatchOrders should report each order's outcome")
		})
	}

	t.Run("more outcomes than orders", func(t *testing.T) {
		t.Parallel()
		_, err := e.CancelBatchOrders(t.Context(), []order.Cancel{{OrderID: "8e1a2b3c-4d5e-4f60-8172-93a4b5c6d7e8", Pair: inverseTradablePair, AssetType: asset.CoinMarginedFutures}})
		assert.ErrorIs(t, err, errBatchOutcomeMismatch, "CancelBatchOrders should reject a reply whose outcomes do not match its orders")
	})
}

func TestBatchOrderSize(t *testing.T) {
	t.Parallel()
	for category, exp := range map[string]int{cOption: 5, cSpot: 10, cLinear: 10, cInverse: 10} {
		assert.Equalf(t, exp, batchOrderSize(category), "batchOrderSize should allow as many %s orders as Bybit takes in a batch and counts in a second", category)
	}
}

func TestCancelAllOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllOrders(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrCancelOrderIsNil, "CancelAllOrders must reject a nil cancellation")
	_, err = e.CancelAllOrders(t.Context(), &order.Cancel{Exchange: e.Name, AssetType: asset.Futures, Pair: currency.NewBTCUSDT()})
	require.ErrorIs(t, err, asset.ErrNotSupported, "CancelAllOrders must reject an asset Bybit does not trade")
	// Bybit cancels coin margined futures by symbol, base coin or settle coin, and inverse settle coins are many
	_, err = e.CancelAllOrders(t.Context(), &order.Cancel{Exchange: e.Name, AssetType: asset.CoinMarginedFutures})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CancelAllOrders must require a pair for coin margined futures")
	_, err = noRequestFormatTestInstance(t, asset.Spot).CancelAllOrders(t.Context(), &order.Cancel{Exchange: e.Name, AssetType: asset.Spot, Pair: currency.NewBTCUSDT()})
	require.ErrorIs(t, err, currency.ErrPairFormatIsNil, "CancelAllOrders must not send a pair whose symbol it cannot write")
	refused, err := restReplyTestInstance(t, nil).CancelAllOrders(t.Context(), &order.Cancel{Exchange: e.Name, AssetType: asset.USDTMarginedFutures})
	require.ErrorIs(t, err, errAPIResponse, "CancelAllOrders must return Bybit's refusal")
	assert.Empty(t, refused.Status, "CancelAllOrders should report no order cancelled when Bybit refuses")
	// Bybit cancels at most 500 futures orders a request, so a full futures reply is followed by another request while
	// each brings new orders, up to a bound
	page := func(request int) string {
		orders := make([]string, cancelAllLimit)
		for i := range orders {
			orders[i] = fmt.Sprintf(`{"orderId":"o%d-%d","orderLinkId":""}`, request, i)
		}
		return `{"retCode":0,"retMsg":"OK","result":{"list":[` + strings.Join(orders, ",") + `],"success":"1"},"retExtInfo":{},"time":1760000000000}`
	}
	rest := `{"retCode":0,"retMsg":"OK","result":{"list":[{"orderId":"r1","orderLinkId":""}],"success":"1"},"retExtInfo":{},"time":1760000000000}`
	for _, tc := range []struct {
		name     string
		asset    asset.Item
		pair     currency.Pair
		reply    func(request int) string
		requests int
		statuses int
		err      error
	}{
		{name: "full reply then the rest", asset: asset.USDTMarginedFutures, reply: func(request int) string {
			if request == 0 {
				return page(request)
			}
			return rest
		}, requests: 2, statuses: cancelAllLimit + 1},
		{name: "full replies without new orders", asset: asset.USDTMarginedFutures, reply: func(int) string { return page(0) }, requests: 2, statuses: cancelAllLimit, err: errCancelAllOrdersFailed},
		{name: "full replies up to the bound", asset: asset.USDTMarginedFutures, reply: page, requests: cancelAllMaxRequests, statuses: cancelAllLimit * cancelAllMaxRequests, err: errCancelAllOrdersFailed},
		{name: "inverse full reply then the rest", asset: asset.CoinMarginedFutures, pair: currency.NewPair(currency.ADA, currency.USD), reply: func(request int) string {
			if request == 0 {
				return page(request)
			}
			return rest
		}, requests: 2, statuses: cancelAllLimit + 1},
		// Option replies are never capped, so a reply of 500 orders is all there were
		{name: "options", asset: asset.Options, reply: page, requests: 1, statuses: cancelAllLimit},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var mu sync.Mutex
			var requests int
			ex := newTransportTestExchange(t, func(w http.ResponseWriter, _ *http.Request) {
				mu.Lock()
				body := tc.reply(requests)
				requests++
				mu.Unlock()
				_, err := fmt.Fprint(w, body)
				assert.NoError(t, err, "writing the reply should not error")
			})
			require.NoError(t, ex.DisableRateLimiter(), "DisableRateLimiter must not error")
			resp, err := ex.CancelAllOrders(t.Context(), &order.Cancel{Exchange: ex.Name, AssetType: tc.asset, Pair: tc.pair})
			assert.ErrorIs(t, err, tc.err, "CancelAllOrders should return the expected error")
			assert.Len(t, resp.Status, tc.statuses, "CancelAllOrders should report every order it cancelled")
			mu.Lock()
			defer mu.Unlock()
			assert.Equal(t, tc.requests, requests, "CancelAllOrders should repeat a full futures reply's request only while it brings new orders")
		})
	}
	// Bybit cancels spot orders one kind at a time, so each kind is asked for and its orders reported
	var kindsMu sync.Mutex
	var kinds []string
	spot := newTransportTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			OrderFilter string `json:"orderFilter"`
		}
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&body), "decoding the cancellation should not error")
		kindsMu.Lock()
		kinds = append(kinds, body.OrderFilter)
		kindsMu.Unlock()
		var id string
		switch body.OrderFilter {
		case "Order":
			id = "1321003749386327552"
		case "tpslOrder":
			id = "1321003749386327560"
		case "StopOrder":
			id = "1321003749386327570"
		case "OcoOrder":
			id = "1321003749386327591"
		case "BidirectionalTpslOrder":
			id = "1321003749386327595"
		}
		_, err := fmt.Fprint(w, `{"retCode":0,"retMsg":"OK","result":{"list":[{"orderId":"`+id+`","orderLinkId":""}],"success":"1"},"retExtInfo":{},"time":1760000000000}`)
		assert.NoError(t, err, "writing the reply should not error")
	})
	require.NoError(t, spot.DisableRateLimiter(), "DisableRateLimiter must not error")
	everyKind, err := spot.CancelAllOrders(t.Context(), &order.Cancel{Exchange: spot.Name, AssetType: asset.Spot})
	require.NoError(t, err, "CancelAllOrders must not error for spot orders")
	assert.Equal(t, order.CancelAllResponse{Status: map[string]string{
		"1321003749386327552": "success",
		"1321003749386327560": "success",
		"1321003749386327570": "success",
		"1321003749386327591": "success",
		"1321003749386327595": "success",
	}}, everyKind, "CancelAllOrders should report the cancelled orders of every kind")
	kindsMu.Lock()
	assert.ElementsMatch(t, []string{"Order", "tpslOrder", "StopOrder", "OcoOrder", "BidirectionalTpslOrder"}, kinds, "CancelAllOrders should cancel each kind of spot order once: orders, TP/SL, conditional, OCO and bidirectional TP/SL orders")
	kindsMu.Unlock()

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		ensureTradablePairs(t)
		for a, p := range map[asset.Item]currency.Pair{asset.Spot: spotTradablePair, asset.USDTMarginedFutures: usdtMarginedTradablePair, asset.CoinMarginedFutures: inverseTradablePair, asset.Options: optionsTradablePair} {
			_, err := e.CancelAllOrders(t.Context(), &order.Cancel{Exchange: e.Name, AssetType: a, Pair: p})
			assert.NoErrorf(t, err, "CancelAllOrders should not error for %s", a)
		}
		return
	}

	for _, tc := range []struct {
		name   string
		cancel *order.Cancel
		exp    order.CancelAllResponse
	}{
		{
			// Spot orders are cancelled one kind at a time: orders, TP/SL, conditional, OCO and bidirectional TP/SL orders
			name:   "spot",
			cancel: &order.Cancel{Exchange: e.Name, AssetType: asset.Spot, Pair: spotTradablePair},
			exp: order.CancelAllResponse{Status: map[string]string{
				"1321003749386327552": "success",
				"1321003749386327590": "success",
				"1321003749386327560": "success",
				"1321003749386327591": "success",
			}},
		},
		{
			name:   "USDT margined futures pair",
			cancel: &order.Cancel{Exchange: e.Name, AssetType: asset.USDTMarginedFutures, Pair: usdtMarginedTradablePair},
			exp:    order.CancelAllResponse{Status: map[string]string{"fd4300ae-7847-404e-b947-b46980a4d140": "success"}},
		},
		{
			// Without a pair, USDT and USDC margined orders are cancelled by settle coin
			name:   "USDT margined futures",
			cancel: &order.Cancel{Exchange: e.Name, AssetType: asset.USDTMarginedFutures},
			exp:    order.CancelAllResponse{Status: map[string]string{"fd4300ae-7847-404e-b947-b46980a4d140": "success", "5bf6aa3e-0c07-4a1f-a3a3-1cb5e3bf8f30": "success"}},
		},
		{
			name:   "USDC margined futures",
			cancel: &order.Cancel{Exchange: e.Name, AssetType: asset.USDCMarginedFutures},
			exp:    order.CancelAllResponse{Status: map[string]string{"9a3c6f8e-5d21-4b7a-8c3e-1f2d4a6b8c90": "success"}},
		},
		{
			name:   "coin margined futures pair",
			cancel: &order.Cancel{Exchange: e.Name, AssetType: asset.CoinMarginedFutures, Pair: inverseTradablePair},
			exp:    order.CancelAllResponse{Status: map[string]string{"8e1a2b3c-4d5e-4f60-8172-93a4b5c6d7e8": "success"}},
		},
		{
			name:   "options pair",
			cancel: &order.Cancel{Exchange: e.Name, AssetType: asset.Options, Pair: optionsTradablePair},
			exp:    order.CancelAllResponse{Status: map[string]string{"8d3a3cbd-14ac-4f3e-a72b-3a5b1a4d1f9a": "success"}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			resp, err := e.CancelAllOrders(t.Context(), tc.cancel)
			require.NoError(t, err, "CancelAllOrders must not error")
			assert.Equal(t, tc.exp, resp, "CancelAllOrders should report the cancelled orders")
		})
	}
	if mockTests {
		// Bybit reports a cancellation it could not carry out with success "0"
		_, err = e.CancelAllOrders(t.Context(), &order.Cancel{Exchange: e.Name, AssetType: asset.USDTMarginedFutures, Pair: currency.NewBTCUSDT()})
		assert.ErrorIs(t, err, errCancelAllOrdersFailed, "CancelAllOrders should report a cancellation Bybit failed")
	}
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderInfo(t.Context(), "", currency.NewBTCUSDT(), asset.Spot)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetOrderInfo must require an order ID")
	_, err = e.GetOrderInfo(t.Context(), "1321003749386327552", currency.EMPTYPAIR, asset.Spot)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetOrderInfo must require a pair")
	_, err = e.GetOrderInfo(t.Context(), "1321003749386327552", currency.NewBTCUSDT(), asset.Margin)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetOrderInfo must reject an asset Bybit does not trade")
	ex := testInstance(t)
	require.NoError(t, ex.CurrencyPairs.SetAssetEnabled(asset.Spot, false), "SetAssetEnabled must not error")
	_, err = ex.GetOrderInfo(t.Context(), "1321003749386327552", currency.NewBTCUSDT(), asset.Spot)
	require.ErrorIs(t, err, asset.ErrNotEnabled, "GetOrderInfo must reject a disabled asset")
	_, err = noRequestFormatTestInstance(t, asset.Spot).GetOrderInfo(t.Context(), "1321003749386327552", currency.NewBTCUSDT(), asset.Spot)
	require.ErrorIs(t, err, currency.ErrPairFormatIsNil, "GetOrderInfo must not request a pair whose symbol it cannot write")
	_, err = restReplyTestInstance(t, nil).GetOrderInfo(t.Context(), "1321003749386327552", currency.NewBTCUSDT(), asset.Spot)
	require.ErrorIs(t, err, errAPIResponse, "GetOrderInfo must return Bybit's refusal to list open orders")
	// An order that is not open is looked up in the order history
	notOpen := restReplyTestInstance(t, map[string]string{"/v5/order/realtime": `{"retCode":0,"retMsg":"OK","result":{"category":"spot","nextPageCursor":"","list":[]},"retExtInfo":{},"time":1760000000000}`})
	_, err = notOpen.GetOrderInfo(t.Context(), "1321003749386327552", currency.NewBTCUSDT(), asset.Spot)
	require.ErrorIs(t, err, errAPIResponse, "GetOrderInfo must return Bybit's refusal to list the order history")

	// An order whose contract has expired or been delisted since keeps the pair it was requested by, though its symbol is
	// no longer an available pair; one of another symbol does not
	emptyList := `{"retCode":0,"retMsg":"OK","result":{"category":"%s","nextPageCursor":"","list":[]},"retExtInfo":{},"time":1792700000000}`
	closedOrder := `{"retCode":0,"retMsg":"OK","result":{"category":"%s","nextPageCursor":"","list":[{"orderId":"d1c2b8f4-5a6e-4f3b-9c7d-2e1f0a9b8c7d","orderLinkId":"","symbol":"%s","side":"Buy","orderType":"Limit","timeInForce":"GTC","orderStatus":"Filled","qty":"0.1","price":"100","avgPrice":"100","cumExecQty":"0.1","cumExecValue":"10","leavesQty":"0","cumExecFee":"0.05","createdTime":"1792108800000","updatedTime":"1792108801000"}]},"retExtInfo":{},"time":1792700000000}`
	expiredOption := currency.NewPairWithDelimiter("BTC", "16OCT26-70000-C", "-")
	for _, tc := range []struct {
		name    string
		asset   asset.Item
		pair    currency.Pair
		replies map[string]string
		err     error
	}{
		{
			name: "expired option", asset: asset.Options, pair: expiredOption,
			replies: map[string]string{"/v5/order/realtime": fmt.Sprintf(closedOrder, "option", "BTC-16OCT26-70000-C")},
		},
		{
			name: "delisted spot pair from the order history", asset: asset.Spot, pair: currency.NewPairWithDelimiter("XYZ", "USDT", ""),
			replies: map[string]string{"/v5/order/realtime": fmt.Sprintf(emptyList, "spot"), "/v5/order/history": fmt.Sprintf(closedOrder, "spot", "XYZUSDT")},
		},
		{
			name: "expired option requested by another option's pair", asset: asset.Options, pair: currency.NewPairWithDelimiter("BTC", "30OCT26-70000-C", "-"),
			replies: map[string]string{"/v5/order/realtime": fmt.Sprintf(closedOrder, "option", "BTC-16OCT26-70000-C")},
			err:     currency.ErrPairNotFound,
		},
		{
			name: "expired option requested by another option's pair from the order history", asset: asset.Options, pair: currency.NewPairWithDelimiter("BTC", "30OCT26-70000-C", "-"),
			replies: map[string]string{"/v5/order/realtime": fmt.Sprintf(emptyList, "option"), "/v5/order/history": fmt.Sprintf(closedOrder, "option", "BTC-16OCT26-70000-C")},
			err:     currency.ErrPairNotFound,
		},
		{
			name: "expired USDT margined future", asset: asset.USDTMarginedFutures, pair: currency.NewPair(currency.BTC, currency.NewCode("USDT-16OCT26")),
			replies: map[string]string{"/v5/order/realtime": fmt.Sprintf(closedOrder, "linear", "BTCUSDT-16OCT26")},
		},
		{
			name: "expired USDC margined future", asset: asset.USDCMarginedFutures, pair: currency.NewPair(currency.BTC, currency.NewCode("16OCT26")),
			replies: map[string]string{"/v5/order/realtime": fmt.Sprintf(closedOrder, "linear", "BTC-16OCT26")},
		},
		{
			// The linear category returns USDT and USDC margined orders alike, so a pair of one asset is not taken for the
			// other's order
			name: "USDC perpetual requested as USDT margined", asset: asset.USDTMarginedFutures, pair: currency.NewPair(currency.BTC, currency.PERP),
			replies: map[string]string{"/v5/order/realtime": fmt.Sprintf(closedOrder, "linear", "BTCPERP")},
			err:     currency.ErrPairNotFound,
		},
		{
			name: "USDT margined future requested as USDC margined", asset: asset.USDCMarginedFutures, pair: currency.NewPair(currency.NewCode("BTCUSDT"), currency.NewCode("16OCT26")),
			replies: map[string]string{"/v5/order/realtime": fmt.Sprintf(closedOrder, "linear", "BTCUSDT-16OCT26")},
			err:     currency.ErrPairNotFound,
		},
	} {
		expired := restReplyTestInstance(t, tc.replies)
		require.NoError(t, expired.CurrencyPairs.StorePairs(asset.Options, currency.Pairs{currency.NewPairWithDelimiter("BTC", "30OCT26-70000-C", "-")}, false), "StorePairs must not error")
		got, err := expired.GetOrderInfo(t.Context(), "d1c2b8f4-5a6e-4f3b-9c7d-2e1f0a9b8c7d", tc.pair, tc.asset)
		require.ErrorIsf(t, err, tc.err, "GetOrderInfo must return the expected error for an %s order", tc.name)
		if tc.err != nil {
			continue
		}
		assert.Equalf(t, tc.pair, got.Pair, "GetOrderInfo should return the %s order with the pair it was requested by", tc.name)
		assert.Equalf(t, order.Filled, got.Status, "GetOrderInfo should return the %s order's status", tc.name)
		assert.Equalf(t, 0.1, got.ExecutedAmount, "GetOrderInfo should return the %s order's executed amount", tc.name)
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
		_, err = e.GetOrderInfo(t.Context(), "12234", spotTradablePair, asset.Spot)
		require.ErrorIs(t, err, order.ErrOrderNotFound, "GetOrderInfo must report an unknown order as not found")
		return
	}

	// The option order is neither open nor in the order history
	_, err = e.GetOrderInfo(t.Context(), "b551f227-7059-4fb5-a6a6-699c04dbd2f2", optionsTradablePair, asset.Options)
	require.ErrorIs(t, err, order.ErrOrderNotFound, "GetOrderInfo must report an order Bybit does not find")
	orders := newOrderFixtures(t)
	for _, tc := range []struct {
		name string
		pair currency.Pair
		exp  *order.Detail
	}{
		{name: "spot", pair: spotTradablePair, exp: &orders.spotPartial},
		// The filled order is no longer open, so it comes from the order history
		{name: "USDT margined futures", pair: usdtMarginedTradablePair, exp: &orders.linearInfoFilled},
		{name: "USDC margined futures", pair: usdcMarginedTradablePair, exp: &orders.usdcNew},
		{name: "coin margined futures", pair: inverseTradablePair, exp: &orders.inversePartial},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := e.GetOrderInfo(t.Context(), tc.exp.OrderID, tc.pair, tc.exp.AssetType)
			require.NoError(t, err, "GetOrderInfo must not error")
			assert.Equal(t, tc.exp, got, "GetOrderInfo should return the order")
		})
	}
}

func TestRequestedPair(t *testing.T) {
	t.Parallel()
	pair := currency.NewPairWithDelimiter("BTC", "16OCT26-70000-C", "-")
	assert.Equal(t, pair, requestedPair(&OpenOrder{Symbol: "BTC-16OCT26-70000-C"}, "BTC-16OCT26-70000-C", pair), "requestedPair should return the pair for an order of its symbol")
	assert.Equal(t, currency.EMPTYPAIR, requestedPair(&OpenOrder{Symbol: "BTC-30OCT26-70000-C"}, "BTC-16OCT26-70000-C", pair), "requestedPair should return no pair for an order of another symbol")
}

func TestOrderSymbol(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		asset asset.Item
		pair  currency.Pair
		exp   string
	}{
		{asset: asset.Spot, pair: currency.NewPairWithDelimiter("btc", "usdt", "_"), exp: "BTCUSDT"},
		{asset: asset.USDTMarginedFutures, pair: currency.NewBTCUSDT(), exp: "BTCUSDT"},
		// A dated USDT margined contract keeps its expiry in the quote of its pair
		{asset: asset.USDTMarginedFutures, pair: currency.NewPairWithDelimiter("BTC", "USDT-25DEC26", "_"), exp: "BTCUSDT-25DEC26"},
		{asset: asset.USDCMarginedFutures, pair: currency.NewPairWithDelimiter("ETH", "PERP", "-"), exp: "ETHPERP"},
		// Bybit names a dated USDC margined contract with a dash between its base coin and expiry
		{asset: asset.USDCMarginedFutures, pair: currency.NewPairWithDelimiter("BTC", "25DEC26", "-"), exp: "BTC-25DEC26"},
		{asset: asset.CoinMarginedFutures, pair: currency.NewPairWithDelimiter("BTC", "USDH26", "_"), exp: "BTCUSDH26"},
		{asset: asset.Options, pair: currency.NewPairWithDelimiter("BTC", "13MAR26-48000-C-USDT", "-"), exp: "BTC-13MAR26-48000-C-USDT"},
	} {
		symbol, err := e.orderSymbol(tc.pair, tc.asset)
		require.NoErrorf(t, err, "orderSymbol must not error for %s %s", tc.asset, tc.pair)
		assert.Equalf(t, tc.exp, symbol, "orderSymbol should return Bybit's symbol for %s %s", tc.asset, tc.pair)
	}
	_, err := e.orderSymbol(currency.EMPTYPAIR, asset.Spot)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "orderSymbol should require a pair")
}

func TestOrderSettleCoin(t *testing.T) {
	t.Parallel()
	for a, exp := range map[asset.Item]currency.Code{
		// USDT and USDC margined futures share the linear category, whose orders need a symbol, base coin or settle coin
		asset.USDTMarginedFutures: currency.USDT,
		asset.USDCMarginedFutures: currency.USDC,
		// Spot orders take no settle coin, and inverse and option orders are listed without one
		asset.Spot:                currency.EMPTYCODE,
		asset.CoinMarginedFutures: currency.EMPTYCODE,
		asset.Options:             currency.EMPTYCODE,
	} {
		assert.Equalf(t, exp, orderSettleCoin(a), "orderSettleCoin should return the settle coin that selects %s orders", a)
	}
}

func TestMatchSymbol(t *testing.T) {
	t.Parallel()
	ex := testInstance(t)
	usdcDated := currency.NewPairWithDelimiter("BTC", "25DEC26", "-")
	usdc, err := ex.GetAvailablePairs(asset.USDCMarginedFutures)
	require.NoError(t, err, "GetAvailablePairs must not error")
	require.NoError(t, ex.CurrencyPairs.StorePairs(asset.USDCMarginedFutures, append(usdc, usdcDated), false), "StorePairs must not error")
	for _, tc := range []struct {
		asset  asset.Item
		symbol string
		exp    currency.Pair
	}{
		{asset: asset.Spot, symbol: "BTCUSDT", exp: currency.NewPairWithDelimiter("BTC", "USDT", "_")},
		// Dated USDT margined contracts keep the dash between quote and expiry, which their pairs hold too
		{asset: asset.USDTMarginedFutures, symbol: "BTCUSDT-25DEC26", exp: currency.NewPairWithDelimiter("BTC", "USDT-25DEC26", "_")},
		{asset: asset.USDCMarginedFutures, symbol: "BTCPERP", exp: currency.NewPairWithDelimiter("BTC", "PERP", "-")},
		// Dated USDC margined contract and option symbols carry a dash between base and quote that the stored pairs'
		// format lacks
		{asset: asset.USDCMarginedFutures, symbol: "BTC-25DEC26", exp: usdcDated},
		{asset: asset.Options, symbol: "BTC-13MAR26-48000-C-USDT", exp: currency.NewPairWithDelimiter("BTC", "13MAR26-48000-C-USDT", "-")},
		{asset: asset.CoinMarginedFutures, symbol: "BTCUSDH26", exp: currency.NewPairWithDelimiter("BTC", "USDH26", "_")},
	} {
		pair, err := ex.matchSymbol(tc.symbol, tc.asset)
		require.NoErrorf(t, err, "matchSymbol must not error for %s %s", tc.asset, tc.symbol)
		assert.Equalf(t, tc.exp, pair, "matchSymbol should return the available %s pair that %s names", tc.asset, tc.symbol)
	}
	_, err = ex.matchSymbol("ZZZQUSDT", asset.Spot)
	assert.ErrorIs(t, err, currency.ErrPairNotFound, "matchSymbol should reject a symbol that names no available pair")
}

func TestOrderDetail(t *testing.T) {
	t.Parallel()
	ex := testInstance(t)
	spotPairs, err := ex.GetAvailablePairs(asset.Spot)
	require.NoError(t, err, "GetAvailablePairs must not error")
	require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Spot, append(spotPairs, currency.NewPairWithDelimiter("XRP", "USDT", "_")), false), "StorePairs must not error")
	for _, tc := range []struct {
		name  string
		asset asset.Item
		order OpenOrder
		exp   order.Detail
		fees  order.Fees
	}{
		{
			// Spot orders cost the quote currency, and report their fees per coin in cumFeeDetail
			name:  "spot limit order",
			asset: asset.Spot,
			order: OpenOrder{
				OrderID: "1321003749386327552", OrderLinkID: "spot-test-01", Symbol: "BTCUSDT", Price: 15600, Quantity: 0.1, Side: "Buy", OrderStatus: "PartiallyFilled",
				AveragePrice: 15600, LeavesQuantity: 0.06, CumulativeExecutedQuantity: 0.04, CumulativeExecutedValue: 624, CumulativeExecutedFee: 0.00004, TimeInForce: "GTC",
				OrderType: "Limit", CreatedTime: types.Time(time.UnixMilli(1684738540559)), UpdatedTime: types.Time(time.UnixMilli(1684738540561)),
				CumulativeFeeDetail: map[currency.Code]types.Number{currency.BTC: 0.00004, currency.MNT: 0.00242968},
			},
			exp: order.Detail{
				Exchange: ex.Name, OrderID: "1321003749386327552", ClientOrderID: "spot-test-01", Pair: currency.NewPairWithDelimiter("BTC", "USDT", "_"), AssetType: asset.Spot,
				Type: order.Limit, Side: order.Buy, Status: order.PartiallyFilled, TimeInForce: order.GoodTillCancel, Price: 15600, Amount: 0.1, AverageExecutedPrice: 15600,
				ExecutedAmount: 0.04, RemainingAmount: 0.06, Cost: 624, CostAsset: currency.USDT, Date: time.UnixMilli(1684738540559), LastUpdated: time.UnixMilli(1684738540561),
			},
			fees: mustFees(t, order.FeeAmount{Currency: currency.BTC, Amount: 0.00004}, order.FeeAmount{Currency: currency.MNT, Amount: 0.00242968}),
		},
		{
			// Bybit sizes a spot market buy by value unless told otherwise, and reports some such buys, like this leg of a
			// bidirectional TP/SL order, with an empty market unit
			name:  "spot market buy without a market unit",
			asset: asset.Spot,
			order: OpenOrder{
				OrderID: "2132879318752520961", OrderLinkID: "2132879318752520962", Symbol: "XRPUSDT", Quantity: 6.972626, Side: "Buy", OrderStatus: "Filled",
				AveragePrice: 1.9408, CumulativeExecutedQuantity: 3.59, CumulativeExecutedValue: 6.967472, CumulativeExecutedFee: 0.00359, TimeInForce: "IOC",
				OrderType: "Market", StopOrderType: "BidirectionalTpslOrder", TriggerPrice: 1.9407, CreatedTime: types.Time(time.UnixMilli(1768995028286)),
				UpdatedTime: types.Time(time.UnixMilli(1769005101477)), CumulativeFeeDetail: map[currency.Code]types.Number{currency.XRP: 0.00359},
			},
			exp: order.Detail{
				Exchange: ex.Name, OrderID: "2132879318752520961", ClientOrderID: "2132879318752520962", Pair: currency.NewPairWithDelimiter("XRP", "USDT", "_"),
				AssetType: asset.Spot, Type: order.OCO, Side: order.Buy, Status: order.Filled, TimeInForce: order.ImmediateOrCancel, Amount: 3.59, QuoteAmount: 6.972626,
				TriggerPrice: 1.9407, AverageExecutedPrice: 1.9408, ExecutedAmount: 3.59, Cost: 6.967472, CostAsset: currency.USDT,
				Date: time.UnixMilli(1768995028286), LastUpdated: time.UnixMilli(1769005101477),
			},
			fees: mustFees(t, order.FeeAmount{Currency: currency.XRP, Amount: 0.00359}),
		},
		{
			// Bybit documents UNKNOWN as an order type it only returns; such an order keeps no type rather than failing
			name:  "unknown order type",
			asset: asset.Spot,
			order: OpenOrder{
				OrderID: "1321003749386327571", OrderLinkID: "spot-test-06", Symbol: "BTCUSDT", Quantity: 0.1, Side: "Sell", OrderStatus: "Filled", TimeInForce: "GTC",
				OrderType: "UNKNOWN", CreatedTime: types.Time(time.UnixMilli(1693087635000)), UpdatedTime: types.Time(time.UnixMilli(1693087635012)),
			},
			exp: order.Detail{
				Exchange: ex.Name, OrderID: "1321003749386327571", ClientOrderID: "spot-test-06", Pair: currency.NewPairWithDelimiter("BTC", "USDT", "_"), AssetType: asset.Spot,
				Type: order.UnknownType, Side: order.Sell, Status: order.Filled, TimeInForce: order.GoodTillCancel, Amount: 0.1, CostAsset: currency.USDT,
				Date: time.UnixMilli(1693087635000), LastUpdated: time.UnixMilli(1693087635012),
			},
		},
		{
			// A spot market buy sized by value holds its quote amount in qty, and the base amount it bought in cumExecQty
			name:  "spot market buy by value",
			asset: asset.Spot,
			order: OpenOrder{
				OrderID: "1321003749386327570", OrderLinkID: "spot-test-05", Symbol: "BTCUSDT", Quantity: 25, Side: "Buy", OrderStatus: "Filled", AveragePrice: 26040.57,
				CumulativeExecutedQuantity: 0.00096, CumulativeExecutedValue: 24.9989472, TimeInForce: "IOC", OrderType: "Market", MarketUnit: "quoteCoin",
				CreatedTime: types.Time(time.UnixMilli(1693087635000)), UpdatedTime: types.Time(time.UnixMilli(1693087635012)),
				CumulativeFeeDetail: map[currency.Code]types.Number{currency.BTC: 0.00000096},
			},
			exp: order.Detail{
				Exchange: ex.Name, OrderID: "1321003749386327570", ClientOrderID: "spot-test-05", Pair: currency.NewPairWithDelimiter("BTC", "USDT", "_"), AssetType: asset.Spot,
				Type: order.Market, Side: order.Buy, Status: order.Filled, TimeInForce: order.ImmediateOrCancel, Amount: 0.00096, QuoteAmount: 25, AverageExecutedPrice: 26040.57,
				ExecutedAmount: 0.00096, Cost: 24.9989472, CostAsset: currency.USDT, Date: time.UnixMilli(1693087635000), LastUpdated: time.UnixMilli(1693087635012),
			},
			fees: mustFees(t, order.FeeAmount{Currency: currency.BTC, Amount: 0.00000096}),
		},
		{
			// An untriggered take profit has charged nothing, which an empty cumFeeDetail reports
			name:  "USDT margined dated contract take profit",
			asset: asset.USDTMarginedFutures,
			order: OpenOrder{
				OrderID: "5bf6aa3e-0c07-4a1f-a3a3-1cb5e3bf8f30", OrderLinkID: "linear-tp-002", Symbol: "BTCUSDT-25DEC26", Price: 90000, Quantity: 0.01, Side: "Sell",
				OrderStatus: "Untriggered", LeavesQuantity: 0.01, TimeInForce: "GTC", OrderType: "Limit", StopOrderType: "TakeProfit", TriggerPrice: 89900, ReduceOnly: true,
				CreatedTime: types.Time(time.UnixMilli(1791884000000)), UpdatedTime: types.Time(time.UnixMilli(1791884000000)), CumulativeFeeDetail: map[currency.Code]types.Number{},
			},
			exp: order.Detail{
				Exchange: ex.Name, OrderID: "5bf6aa3e-0c07-4a1f-a3a3-1cb5e3bf8f30", ClientOrderID: "linear-tp-002", Pair: currency.NewPairWithDelimiter("BTC", "USDT-25DEC26", "_"),
				AssetType: asset.USDTMarginedFutures, Type: order.TakeProfit, Side: order.Sell, Status: order.Pending, TimeInForce: order.GoodTillCancel, ReduceOnly: true, Price: 90000,
				Amount: 0.01, TriggerPrice: 89900, RemainingAmount: 0.01, CostAsset: currency.USDT, SettlementCurrency: currency.USDT,
				Date: time.UnixMilli(1791884000000), LastUpdated: time.UnixMilli(1791884000000),
			},
			fees: mustFees(t),
		},
		{
			// A conditional market order is reported as the market order it becomes, with its condition in stopOrderType
			name:  "USDT margined conditional market order",
			asset: asset.USDTMarginedFutures,
			order: OpenOrder{
				OrderID: "5bf6aa3e-0c07-4a1f-a3a3-1cb5e3bf8f31", OrderLinkID: "linear-stop-001", Symbol: "BTCUSDT", Quantity: 0.01, Side: "Buy",
				OrderStatus: "Untriggered", LeavesQuantity: 0.01, TimeInForce: "IOC", OrderType: "Market", StopOrderType: "Stop", TriggerPrice: 95000,
				CreatedTime: types.Time(time.UnixMilli(1791884000000)), UpdatedTime: types.Time(time.UnixMilli(1791884000000)), CumulativeFeeDetail: map[currency.Code]types.Number{},
			},
			exp: order.Detail{
				Exchange: ex.Name, OrderID: "5bf6aa3e-0c07-4a1f-a3a3-1cb5e3bf8f31", ClientOrderID: "linear-stop-001", Pair: currency.NewPairWithDelimiter("BTC", "USDT", "_"),
				AssetType: asset.USDTMarginedFutures, Type: order.StopMarket, Side: order.Buy, Status: order.Pending, TimeInForce: order.ImmediateOrCancel,
				Amount: 0.01, TriggerPrice: 95000, RemainingAmount: 0.01, CostAsset: currency.USDT, SettlementCurrency: currency.USDT,
				Date: time.UnixMilli(1791884000000), LastUpdated: time.UnixMilli(1791884000000),
			},
			fees: mustFees(t),
		},
		{
			// An order from before cumFeeDetail lacks it, so its fees are not reported; the deprecated cumExecFee is not used
			name:  "USDT margined order without cumFeeDetail",
			asset: asset.USDTMarginedFutures,
			order: OpenOrder{
				OrderID: "c1956690-b731-4191-97c0-94b00422231b", OrderLinkID: "linear-002", Symbol: "BTCUSDT", Price: 26000, Quantity: 0.02, Side: "Buy", OrderStatus: "Filled",
				AveragePrice: 26000, CumulativeExecutedQuantity: 0.02, CumulativeExecutedValue: 520, CumulativeExecutedFee: 0.286, TimeInForce: "GTC", OrderType: "Limit",
				CreatedTime: types.Time(time.UnixMilli(1693086704010)), UpdatedTime: types.Time(time.UnixMilli(1693086704012)),
			},
			exp: order.Detail{
				Exchange: ex.Name, OrderID: "c1956690-b731-4191-97c0-94b00422231b", ClientOrderID: "linear-002", Pair: currency.NewPairWithDelimiter("BTC", "USDT", "_"),
				AssetType: asset.USDTMarginedFutures, Type: order.Limit, Side: order.Buy, Status: order.Filled, TimeInForce: order.GoodTillCancel, Price: 26000, Amount: 0.02,
				AverageExecutedPrice: 26000, ExecutedAmount: 0.02, Cost: 520, CostAsset: currency.USDT, SettlementCurrency: currency.USDT,
				Date: time.UnixMilli(1693086704010), LastUpdated: time.UnixMilli(1693086704012),
			},
		},
		{
			// Bybit documents Block trade-Limit as an order type only unified account block trades return; they trade as limit
			// orders
			name:  "block trade limit order",
			asset: asset.USDTMarginedFutures,
			order: OpenOrder{
				OrderID: "f2a1b3c4-5d6e-4f70-8192-a3b4c5d6e7f8", OrderLinkID: "linear-block-001", Symbol: "BTCUSDT", Price: 26000, Quantity: 0.02, Side: "Buy",
				OrderStatus: "Filled", TimeInForce: "GTC", OrderType: "Block trade-Limit", CreatedTime: types.Time(time.UnixMilli(1693086704010)),
				UpdatedTime: types.Time(time.UnixMilli(1693086704012)),
			},
			exp: order.Detail{
				Exchange: ex.Name, OrderID: "f2a1b3c4-5d6e-4f70-8192-a3b4c5d6e7f8", ClientOrderID: "linear-block-001", Pair: currency.NewPairWithDelimiter("BTC", "USDT", "_"),
				AssetType: asset.USDTMarginedFutures, Type: order.Limit, Side: order.Buy, Status: order.Filled, TimeInForce: order.GoodTillCancel, Price: 26000,
				Amount: 0.02, CostAsset: currency.USDT, SettlementCurrency: currency.USDT, Date: time.UnixMilli(1693086704010), LastUpdated: time.UnixMilli(1693086704012),
			},
		},
		{
			// A block trade's roll back has no GoCryptoTrader counterpart, so it keeps no type rather than failing
			name:  "block trade roll back",
			asset: asset.USDTMarginedFutures,
			order: OpenOrder{
				OrderID: "f2a1b3c4-5d6e-4f70-8192-a3b4c5d6e7f9", OrderLinkID: "linear-block-002", Symbol: "BTCUSDT", Price: 26000, Quantity: 0.02, Side: "Sell",
				OrderStatus: "Filled", TimeInForce: "GTC", OrderType: "Block trade Roll Back", CreatedTime: types.Time(time.UnixMilli(1693086705010)),
				UpdatedTime: types.Time(time.UnixMilli(1693086705012)),
			},
			exp: order.Detail{
				Exchange: ex.Name, OrderID: "f2a1b3c4-5d6e-4f70-8192-a3b4c5d6e7f9", ClientOrderID: "linear-block-002", Pair: currency.NewPairWithDelimiter("BTC", "USDT", "_"),
				AssetType: asset.USDTMarginedFutures, Type: order.UnknownType, Side: order.Sell, Status: order.Filled, TimeInForce: order.GoodTillCancel, Price: 26000,
				Amount: 0.02, CostAsset: currency.USDT, SettlementCurrency: currency.USDT, Date: time.UnixMilli(1693086705010), LastUpdated: time.UnixMilli(1693086705012),
			},
		},
		{
			// An RPI order rests on the book as a post only order
			name:  "USDC margined RPI order",
			asset: asset.USDCMarginedFutures,
			order: OpenOrder{
				OrderID: "9a3c6f8e-5d21-4b7a-8c3e-1f2d4a6b8c90", OrderLinkID: "usdc-001", Symbol: "ETHPERP", Price: 2600, Quantity: 0.5, Side: "Buy", OrderStatus: "New",
				LeavesQuantity: 0.5, TimeInForce: "RPI", OrderType: "Limit", CreatedTime: types.Time(time.UnixMilli(1705017413000)),
				UpdatedTime: types.Time(time.UnixMilli(1705017413000)), CumulativeFeeDetail: map[currency.Code]types.Number{},
			},
			exp: order.Detail{
				Exchange: ex.Name, OrderID: "9a3c6f8e-5d21-4b7a-8c3e-1f2d4a6b8c90", ClientOrderID: "usdc-001", Pair: currency.NewPairWithDelimiter("ETH", "PERP", "-"),
				AssetType: asset.USDCMarginedFutures, Type: order.Limit, Side: order.Buy, Status: order.New, TimeInForce: order.PostOnly, Price: 2600, Amount: 0.5,
				RemainingAmount: 0.5, CostAsset: currency.USDC, SettlementCurrency: currency.USDC, Date: time.UnixMilli(1705017413000), LastUpdated: time.UnixMilli(1705017413000),
			},
			fees: mustFees(t),
		},
		{
			// Inverse orders settle in their base coin and report their fees in cumExecFee
			name:  "coin margined futures order",
			asset: asset.CoinMarginedFutures,
			order: OpenOrder{
				OrderID: "8e1a2b3c-4d5e-4f60-8172-93a4b5c6d7e8", OrderLinkID: "inverse-001", Symbol: "ADAUSD", Price: 0.5, Quantity: 100, Side: "Buy",
				OrderStatus: "PartiallyFilled", AveragePrice: 0.5, LeavesQuantity: 50, CumulativeExecutedQuantity: 50, CumulativeExecutedValue: 100,
				CumulativeExecutedFee: 0.055, TimeInForce: "GTC", OrderType: "Limit", CreatedTime: types.Time(time.UnixMilli(1704941124000)),
				UpdatedTime: types.Time(time.UnixMilli(1704941125000)),
			},
			exp: order.Detail{
				Exchange: ex.Name, OrderID: "8e1a2b3c-4d5e-4f60-8172-93a4b5c6d7e8", ClientOrderID: "inverse-001", Pair: currency.NewPairWithDelimiter("ADA", "USD", "_"),
				AssetType: asset.CoinMarginedFutures, Type: order.Limit, Side: order.Buy, Status: order.PartiallyFilled, TimeInForce: order.GoodTillCancel, Price: 0.5,
				Amount: 100, AverageExecutedPrice: 0.5, ExecutedAmount: 50, RemainingAmount: 50, Cost: 100, CostAsset: currency.ADA, SettlementCurrency: currency.ADA,
				Date: time.UnixMilli(1704941124000), LastUpdated: time.UnixMilli(1704941125000),
			},
			fees: mustFees(t, order.FeeAmount{Currency: currency.ADA, Amount: 0.055}),
		},
		{
			// An option named with a USDT suffix settles in USDT, and reports its fees in cumExecFee
			name:  "USDT settled option order",
			asset: asset.Options,
			order: OpenOrder{
				OrderID: "8d3a3cbd-14ac-4f3e-a72b-3a5b1a4d1f9a", OrderLinkID: "option-test-001", Symbol: "BTC-13MAR26-48000-C-USDT", Price: 3500, Quantity: 0.5,
				Side: "Sell", OrderStatus: "Cancelled", AveragePrice: 3500, CumulativeExecutedQuantity: 0.3, CumulativeExecutedValue: 1050, CumulativeExecutedFee: 0.6,
				TimeInForce: "GTC", OrderType: "Limit", CreatedTime: types.Time(time.UnixMilli(1772000000000)), UpdatedTime: types.Time(time.UnixMilli(1772000005000)),
			},
			exp: order.Detail{
				Exchange: ex.Name, OrderID: "8d3a3cbd-14ac-4f3e-a72b-3a5b1a4d1f9a", ClientOrderID: "option-test-001", Pair: currency.NewPairWithDelimiter("BTC", "13MAR26-48000-C-USDT", "-"),
				AssetType: asset.Options, Type: order.Limit, Side: order.Sell, Status: order.Cancelled, TimeInForce: order.GoodTillCancel, Price: 3500, Amount: 0.5,
				AverageExecutedPrice: 3500, ExecutedAmount: 0.3, Cost: 1050, CostAsset: currency.USDT, SettlementCurrency: currency.USDT,
				Date: time.UnixMilli(1772000000000), LastUpdated: time.UnixMilli(1772000005000),
			},
			fees: mustFees(t, order.FeeAmount{Currency: currency.USDT, Amount: 0.6}),
		},
	} {
		got, err := ex.orderDetail(&tc.order, tc.asset, currency.EMPTYPAIR)
		require.NoErrorf(t, err, "orderDetail must not error for a %s", tc.name)
		exp := tc.exp
		exp.SetFees(tc.fees)
		assert.Equalf(t, &exp, got, "orderDetail should convert a %s", tc.name)
	}

	// A spot market buy sized by value has bought nothing of its base amount until it fills, yet stays an open order
	untriggered, err := ex.orderDetail(&OpenOrder{
		OrderID: "2132879318752520963", Symbol: "BTCUSDT", Quantity: 100, Side: "Buy", OrderStatus: "Untriggered", TimeInForce: "IOC",
		OrderType: "Market", StopOrderType: "tpslOrder", TriggerPrice: 90000, MarketUnit: "quoteCoin",
	}, asset.Spot, currency.EMPTYPAIR)
	require.NoError(t, err, "orderDetail must not error for an untriggered spot market buy sized by value")
	assert.True(t, untriggered.IsActive(), "orderDetail should leave an untriggered spot market buy sized by value active")

	// The known pair of one linear asset is not taken for an order of the other's
	_, err = ex.orderDetail(&OpenOrder{Symbol: "BTCPERP", Side: "Buy", OrderType: "Limit", TimeInForce: "GTC"}, asset.USDTMarginedFutures, currency.NewPair(currency.BTC, currency.PERP))
	require.ErrorIs(t, err, currency.ErrPairNotFound, "orderDetail must not take a known pair for a USDC margined order asked for as USDT margined")
	_, err = ex.orderDetail(&OpenOrder{Symbol: "BTCUSDT-16OCT26", Side: "Buy", OrderType: "Limit", TimeInForce: "GTC"}, asset.USDCMarginedFutures, currency.NewPair(currency.NewCode("BTCUSDT"), currency.NewCode("16OCT26")))
	require.ErrorIs(t, err, currency.ErrPairNotFound, "orderDetail must not take a known pair for a USDT margined order asked for as USDC margined")
	expired, err := ex.orderDetail(&OpenOrder{Symbol: "BTCUSDT-16OCT26", Side: "Buy", OrderType: "Limit", TimeInForce: "GTC"}, asset.USDTMarginedFutures, currency.NewPair(currency.BTC, currency.NewCode("USDT-16OCT26")))
	require.NoError(t, err, "orderDetail must not error for an expired USDT margined future its caller knows")
	assert.Equal(t, currency.USDT, expired.SettlementCurrency, "orderDetail should settle an expired USDT margined future in USDT")

	// An order whose symbol is no longer an available pair takes the pair its caller knows, and keeps a listed one's own
	delisted := currency.NewPairWithDelimiter("XYZ", "USDT", "")
	got, err := ex.orderDetail(&OpenOrder{Symbol: "XYZUSDT", Side: "Buy", OrderType: "Limit", TimeInForce: "GTC"}, asset.Spot, delisted)
	require.NoError(t, err, "orderDetail must not error for an order of a pair its caller knows")
	assert.Equal(t, delisted, got.Pair, "orderDetail should take the known pair of an order whose symbol is no longer available")
	got, err = ex.orderDetail(&OpenOrder{Symbol: "BTCUSDT", Side: "Buy", OrderType: "Limit", TimeInForce: "GTC"}, asset.Spot, delisted)
	require.NoError(t, err, "orderDetail must not error for an order of an available pair")
	assert.True(t, currency.NewBTCUSDT().Equal(got.Pair), "orderDetail should take an available pair's own match over the known pair")

	// An asset Bybit does not trade has no cost currency, even when its pairs are stored
	require.NoError(t, ex.CurrencyPairs.Store(asset.Margin, &currency.PairStore{AssetEnabled: true, Available: currency.Pairs{currency.NewBTCUSDT()}}), "Store must not error")
	for _, tc := range []struct {
		name  string
		asset asset.Item
		order OpenOrder
		err   error
	}{
		{name: "symbol that names no available pair", asset: asset.Spot, order: OpenOrder{Symbol: "ZZZQUSDT", Side: "Buy", OrderType: "Limit", TimeInForce: "GTC"}, err: currency.ErrPairNotFound},
		{name: "unknown order type", asset: asset.Spot, order: OpenOrder{Symbol: "BTCUSDT", Side: "Buy", OrderType: "Iceberg", TimeInForce: "GTC"}, err: order.ErrUnrecognisedOrderType},
		{name: "unknown side", asset: asset.Spot, order: OpenOrder{Symbol: "BTCUSDT", Side: "Hold", OrderType: "Limit", TimeInForce: "GTC"}, err: order.ErrSideIsInvalid},
		{name: "unknown time in force", asset: asset.Spot, order: OpenOrder{Symbol: "BTCUSDT", Side: "Buy", OrderType: "Limit", TimeInForce: "Forever"}, err: order.ErrInvalidTimeInForce},
		{name: "asset Bybit does not trade", asset: asset.Margin, order: OpenOrder{Symbol: "BTCUSDT", Side: "Buy", OrderType: "Limit", TimeInForce: "GTC"}, err: asset.ErrNotSupported},
		{name: "fee without a coin", asset: asset.Spot, order: OpenOrder{Symbol: "BTCUSDT", Side: "Buy", OrderType: "Limit", TimeInForce: "GTC", CumulativeFeeDetail: map[currency.Code]types.Number{currency.EMPTYCODE: 0.1}}, err: currency.ErrCurrencyCodeEmpty},
	} {
		_, err := ex.orderDetail(&tc.order, tc.asset, currency.EMPTYPAIR)
		assert.ErrorIsf(t, err, tc.err, "orderDetail should reject an order with a %s", tc.name)
	}
}

func TestOrderCostCurrency(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		asset  asset.Item
		pair   currency.Pair
		symbol string
		exp    currency.Code
	}{
		// Spot orders cost their quote currency, and derivatives their settle coin
		{asset: asset.Spot, pair: currency.NewBTCUSDT(), symbol: "BTCUSDT", exp: currency.USDT},
		{asset: asset.Spot, pair: currency.NewPair(currency.ETH, currency.BTC), symbol: "ETHBTC", exp: currency.BTC},
		{asset: asset.USDTMarginedFutures, pair: currency.NewBTCUSDT(), symbol: "BTCUSDT", exp: currency.USDT},
		{asset: asset.USDCMarginedFutures, pair: currency.NewPair(currency.ETH, currency.PERP), symbol: "ETHPERP", exp: currency.USDC},
		// Inverse contracts settle in their base coin
		{asset: asset.CoinMarginedFutures, pair: currency.NewPair(currency.ADA, currency.USD), symbol: "ADAUSD", exp: currency.ADA},
		{asset: asset.CoinMarginedFutures, pair: currency.NewPairWithDelimiter("BTC", "USDH26", "_"), symbol: "BTCUSDH26", exp: currency.BTC},
		// Options settle in USDC, unless named with a USDT suffix
		{asset: asset.Options, pair: currency.NewPairWithDelimiter("BTC", "26NOV24-92000-C", "-"), symbol: "BTC-26NOV24-92000-C", exp: currency.USDC},
		{asset: asset.Options, pair: currency.NewPairWithDelimiter("BTC", "13MAR26-48000-C-USDT", "-"), symbol: "BTC-13MAR26-48000-C-USDT", exp: currency.USDT},
	} {
		got, err := orderCostCurrency(tc.asset, tc.pair, tc.symbol)
		require.NoErrorf(t, err, "orderCostCurrency must not error for %s %s", tc.asset, tc.symbol)
		assert.Equalf(t, tc.exp, got, "orderCostCurrency should return the currency %s %s orders cost", tc.asset, tc.symbol)
	}
	_, err := orderCostCurrency(asset.Margin, currency.NewBTCUSDT(), "BTCUSDT")
	assert.ErrorIs(t, err, asset.ErrNotSupported, "orderCostCurrency should reject an asset Bybit does not trade")
}

func TestOrderFees(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name       string
		asset      asset.Item
		settle     currency.Code
		detail     map[currency.Code]types.Number
		cumExecFee types.Number
		exp        order.Fees
	}{
		// Inverse and option orders report their fees in cumExecFee, in the settle coin
		{name: "coin margined futures", asset: asset.CoinMarginedFutures, settle: currency.ADA, cumExecFee: 0.055, exp: mustFees(t, order.FeeAmount{Currency: currency.ADA, Amount: 0.055})},
		{name: "options", asset: asset.Options, settle: currency.USDC, cumExecFee: 0.6, exp: mustFees(t, order.FeeAmount{Currency: currency.USDC, Amount: 0.6})},
		// Spot and linear orders report their fees per coin in cumFeeDetail, and cumExecFee is deprecated for them
		{
			name: "spot", asset: asset.Spot, settle: currency.USDT, cumExecFee: 0.00004,
			detail: map[currency.Code]types.Number{currency.BTC: 0.00004, currency.MNT: 0.00242968},
			exp:    mustFees(t, order.FeeAmount{Currency: currency.BTC, Amount: 0.00004}, order.FeeAmount{Currency: currency.MNT, Amount: 0.00242968}),
		},
		{name: "USDC margined futures", asset: asset.USDCMarginedFutures, settle: currency.USDC, detail: map[currency.Code]types.Number{currency.USDC: 0.61022}, exp: mustFees(t, order.FeeAmount{Currency: currency.USDC, Amount: 0.61022})},
		// An empty cumFeeDetail reports that nothing was charged
		{name: "USDT margined futures without fees", asset: asset.USDTMarginedFutures, settle: currency.USDT, detail: map[currency.Code]types.Number{}, exp: mustFees(t)},
		// Orders from before cumFeeDetail lack it, so their fees are not reported
		{name: "USDT margined futures before cumFeeDetail", asset: asset.USDTMarginedFutures, settle: currency.USDT, cumExecFee: 0.286},
	} {
		got, err := orderFees(tc.asset, tc.settle, tc.detail, tc.cumExecFee)
		require.NoErrorf(t, err, "orderFees must not error for %s", tc.name)
		assert.Equalf(t, tc.exp, got, "orderFees should return the fees of %s", tc.name)
	}
	_, err := orderFees(asset.Spot, currency.USDT, map[currency.Code]types.Number{currency.EMPTYCODE: 0.1}, 0)
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "orderFees should reject a fee without a coin")
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := e.GetDepositAddress(t.Context(), currency.EMPTYCODE, "", "ETH")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetDepositAddress must require a currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		addr, err := e.GetDepositAddress(t.Context(), currency.USDT, "", currency.ETH.String())
		require.NoError(t, err, "GetDepositAddress must not error")
		assert.NotEmpty(t, addr.Address, "GetDepositAddress should return an address")
		return
	}

	// A chain the account has no address on
	_, err = e.GetDepositAddress(t.Context(), currency.USDT, "", "SOL")
	require.ErrorIs(t, err, deposit.ErrAddressNotFound, "GetDepositAddress must report a chain without an address")
	for _, tc := range []struct {
		coin  currency.Code
		chain string
		exp   *deposit.Address
	}{
		{coin: currency.USDT, chain: "ETH", exp: &deposit.Address{Address: "XXXXXX", Chain: "ETH"}},
		{coin: currency.XRP, chain: "XRP", exp: &deposit.Address{Address: "XXXXXX", Tag: "100000123", Chain: "XRP"}},
	} {
		addr, err := e.GetDepositAddress(t.Context(), tc.coin, "", tc.chain)
		require.NoErrorf(t, err, "GetDepositAddress must not error for %s on %s", tc.coin, tc.chain)
		assert.Equalf(t, tc.exp, addr, "GetDepositAddress should return the %s address on %s", tc.coin, tc.chain)
	}
}

func TestGetAvailableTransferChains(t *testing.T) {
	t.Parallel()
	refused, err := restReplyTestInstance(t, nil).GetAvailableTransferChains(t.Context(), currency.USDT)
	require.ErrorIs(t, err, errAPIResponse, "GetAvailableTransferChains must return Bybit's refusal")
	assert.Nil(t, refused, "GetAvailableTransferChains should return no chains when Bybit refuses to list them")
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	chains, err := e.GetAvailableTransferChains(t.Context(), currency.USDT)
	require.NoError(t, err, "GetAvailableTransferChains must not error")
	if !mockTests {
		assert.NotEmpty(t, chains, "GetAvailableTransferChains should return USDT's chains")
		return
	}
	// The TRON chain takes deposits but not withdrawals, and the closed Solana chain is left out
	assert.Equal(t, []string{"ETH", "TRX"}, chains, "GetAvailableTransferChains should return the chains open to deposits or withdrawals")
}

func TestWithdrawCryptocurrencyFunds(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawCryptocurrencyFunds(t.Context(), nil)
	require.ErrorIs(t, err, withdraw.ErrRequestCannotBeNil, "WithdrawCryptocurrencyFunds must reject a nil request")
	req := &withdraw.Request{Exchange: e.Name, Amount: -0.1, Currency: currency.LTC, Crypto: withdraw.CryptoRequest{Chain: currency.LTC.String(), Address: "XXXXXX"}}
	_, err = e.WithdrawCryptocurrencyFunds(t.Context(), req)
	require.ErrorContains(t, err, withdraw.ErrStrAmountMustBeGreaterThanZero, "WithdrawCryptocurrencyFunds must reject a negative amount")
	req.Amount, req.Crypto.Chain = 0.1, ""
	_, err = e.WithdrawCryptocurrencyFunds(t.Context(), req)
	require.ErrorIs(t, err, errMissingChainInformation, "WithdrawCryptocurrencyFunds must require a chain")
	req.Crypto.Chain = currency.LTC.String()

	// A withdrawal to an address with a memo sends its tag, which Bybit needs to match the address book entry
	var sent map[string]any
	tagged := newTransportTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&sent), "decoding the withdrawal should not error")
		_, err := fmt.Fprint(w, `{"retCode":0,"retMsg":"success","result":{"id":"10196"},"retExtInfo":{},"time":1672196571239}`)
		assert.NoError(t, err, "writing the reply should not error")
	})
	taggedReq := &withdraw.Request{Exchange: tagged.Name, Amount: 25, Currency: currency.XRP, Crypto: withdraw.CryptoRequest{Chain: currency.XRP.String(), Address: "XXXXXX", AddressTag: "100000123"}}
	before := time.Now().Truncate(time.Millisecond)
	_, err = tagged.WithdrawCryptocurrencyFunds(t.Context(), taggedReq)
	require.NoError(t, err, "WithdrawCryptocurrencyFunds must not error for an address with a tag")
	assert.Equal(t, "100000123", sent["tag"], "WithdrawCryptocurrencyFunds should send the address's tag")
	// It also sends the current time in milliseconds, which Bybit uses against withdrawal replays
	sentAt, ok := sent["timestamp"].(float64)
	require.True(t, ok, "WithdrawCryptocurrencyFunds must send a numeric timestamp")
	assert.WithinRange(t, time.UnixMilli(int64(sentAt)), before, time.Now(), "WithdrawCryptocurrencyFunds should send the current time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		_, err = e.WithdrawCryptocurrencyFunds(t.Context(), req)
		require.ErrorIs(t, err, errAPIResponse, "WithdrawCryptocurrencyFunds must be refused for an address outside the address book")
		return
	}
	// Withdrawals are drawn from the funding wallet
	resp, err := e.WithdrawCryptocurrencyFunds(t.Context(), req)
	require.NoError(t, err, "WithdrawCryptocurrencyFunds must not error")
	assert.Equal(t, &withdraw.ExchangeResponse{ID: "10195"}, resp, "WithdrawCryptocurrencyFunds should return the withdrawal ID")
}

func TestWithdrawFiatFunds(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawFiatFunds(t.Context(), &withdraw.Request{})
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "WithdrawFiatFunds should not be supported")
}

func TestWithdrawFiatFundsToInternationalBank(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawFiatFundsToInternationalBank(t.Context(), &withdraw.Request{})
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "WithdrawFiatFundsToInternationalBank should not be supported")
}

// spotOrderPage returns a page of one open spot order with orderID, and the cursor of the next page
func spotOrderPage(orderID, nextPageCursor string) string {
	return `{"category":"spot","nextPageCursor":"` + nextPageCursor + `","list":[{"orderId":"` + orderID + `","orderLinkId":"","symbol":"BTCUSDT","price":"15600","qty":"0.1","side":"Buy","orderStatus":"New","leavesQty":"0.1","timeInForce":"GTC","orderType":"Limit","createdTime":"1684738540559","updatedTime":"1684738540559","cumFeeDetail":{}}]}`
}

// orderListingEnds holds listings that end at a page repeating the cursor it was asked with, or a page without orders,
// whatever cursor the latter holds, and the IDs of the orders listed up to there
var orderListingEnds = []struct {
	name  string
	pages map[string]string
	exp   []string
}{
	{name: "a page repeating its cursor", pages: map[string]string{"": spotOrderPage("1", "c1"), "c1": spotOrderPage("2", "c1")}, exp: []string{"1", "2"}},
	{name: "a page without orders", pages: map[string]string{"": spotOrderPage("1", "c1"), "c1": `{"category":"spot","nextPageCursor":"c2","list":[]}`}, exp: []string{"1"}},
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetActiveOrders(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrGetOrdersRequestIsNil, "GetActiveOrders must reject a nil request")
	_, err = e.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Type: order.AnyType})
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "GetActiveOrders must require a side")
	_, err = e.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Side: order.AnySide})
	require.ErrorIs(t, err, order.ErrUnrecognisedOrderType, "GetActiveOrders must require an order type")
	_, err = e.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Margin, Side: order.AnySide, Type: order.AnyType})
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetActiveOrders must reject an asset Bybit does not trade")
	_, err = restReplyTestInstance(t, nil).GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Side: order.AnySide, Type: order.AnyType})
	require.ErrorIs(t, err, errAPIResponse, "GetActiveOrders must return Bybit's refusal")
	// An order it cannot interpret fails the request rather than being reported wrongly
	unknownSide := restReplyTestInstance(t, map[string]string{"/v5/order/realtime": `{"retCode":0,"retMsg":"OK","result":{"category":"spot","nextPageCursor":"","list":[{"orderId":"1321003749386327599","orderLinkId":"spot-test-11","symbol":"BTCUSDT","price":"15600","qty":"0.1","side":"Hold","orderStatus":"New","leavesQty":"0.1","timeInForce":"GTC","orderType":"Limit","createdTime":"1684738540559","updatedTime":"1684738540559","cumFeeDetail":{}}]},"retExtInfo":{},"time":1760000000000}`})
	_, err = unknownSide.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Side: order.AnySide, Type: order.AnyType})
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "GetActiveOrders must reject an order of a side it does not know")
	for _, tc := range orderListingEnds {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := newPagingTestExchange(t, tc.pages).GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Side: order.AnySide, Type: order.AnyType})
			require.NoError(t, err, "GetActiveOrders must stop at the last page")
			ids := make([]string, len(got))
			for i := range got {
				ids[i] = got[i].OrderID
			}
			assert.Equal(t, tc.exp, ids, "GetActiveOrders should return the orders of each page up to the last")
		})
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		for _, a := range []asset.Item{asset.Spot, asset.USDTMarginedFutures, asset.USDCMarginedFutures, asset.CoinMarginedFutures, asset.Options} {
			_, err := e.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: a, Side: order.AnySide, Type: order.AnyType})
			assert.NoErrorf(t, err, "GetActiveOrders should not error for %s", a)
		}
		return
	}

	orders := newOrderFixtures(t)
	pairs, err := currency.NewPairsFromStrings([]string{"BTC_USDT", "BTC_ETH", "BTC_USDC"})
	require.NoError(t, err, "NewPairsFromStrings must not error")
	for _, tc := range []struct {
		name string
		req  *order.MultiOrderRequest
		exp  order.FilteredOrders
	}{
		{
			// Two pages; the order for a symbol Bybit no longer lists is left out
			name: "spot",
			req:  &order.MultiOrderRequest{AssetType: asset.Spot, Side: order.AnySide, Type: order.AnyType},
			exp:  order.FilteredOrders{orders.spotPartial, orders.spotNew, orders.spotTPSL},
		},
		{
			name: "spot pairs",
			req:  &order.MultiOrderRequest{Pairs: pairs, AssetType: asset.Spot, Side: order.AnySide, Type: order.AnyType},
			exp:  order.FilteredOrders{orders.spotPartial, orders.spotTPSL},
		},
		{
			// USDT and USDC margined orders are requested by settle coin
			name: "USDT margined futures",
			req:  &order.MultiOrderRequest{Pairs: currency.Pairs{usdtMarginedTradablePair, currency.NewPairWithDelimiter("BTC", "USDT-25DEC26", "_")}, AssetType: asset.USDTMarginedFutures, Side: order.AnySide, Type: order.AnyType},
			exp:  order.FilteredOrders{orders.linearTakeProfit, orders.linearDatedPartial},
		},
		{
			name: "USDC margined futures",
			req:  &order.MultiOrderRequest{AssetType: asset.USDCMarginedFutures, Side: order.AnySide, Type: order.AnyType},
			exp:  order.FilteredOrders{orders.usdcNew},
		},
		{
			name: "coin margined futures",
			req:  &order.MultiOrderRequest{AssetType: asset.CoinMarginedFutures, Side: order.AnySide, Type: order.AnyType},
			exp:  order.FilteredOrders{orders.inversePartial},
		},
		{
			name: "options",
			req:  &order.MultiOrderRequest{AssetType: asset.Options, Side: order.AnySide, Type: order.AnyType},
			exp:  order.FilteredOrders{orders.optionPartial},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := e.GetActiveOrders(t.Context(), tc.req)
			require.NoError(t, err, "GetActiveOrders must not error")
			assert.Equal(t, tc.exp, got, "GetActiveOrders should return the open orders")
		})
	}
}

func TestConstructOrderDetails(t *testing.T) {
	t.Parallel()
	ex := testInstance(t)
	btcOrder := OpenOrder{
		OrderID: "1321003749386327552", OrderLinkID: "spot-test-01", Symbol: "BTCUSDT", Price: 15600, Quantity: 0.1, Side: "Buy", OrderStatus: "PartiallyFilled",
		AveragePrice: 15600, LeavesQuantity: 0.06, CumulativeExecutedQuantity: 0.04, CumulativeExecutedValue: 624, TimeInForce: "GTC", OrderType: "Limit",
		CreatedTime: types.Time(time.UnixMilli(1684738540559)), UpdatedTime: types.Time(time.UnixMilli(1684738540561)),
		CumulativeFeeDetail: map[currency.Code]types.Number{currency.BTC: 0.00004},
	}
	ethOrder := OpenOrder{
		OrderID: "1321003749386327553", OrderLinkID: "spot-test-02", Symbol: "ETHUSDT", Price: 1900, Quantity: 0.5, Side: "Sell", OrderStatus: "New",
		LeavesQuantity: 0.5, TimeInForce: "PostOnly", OrderType: "Limit", CreatedTime: types.Time(time.UnixMilli(1684738600000)),
		UpdatedTime: types.Time(time.UnixMilli(1684738600000)), CumulativeFeeDetail: map[currency.Code]types.Number{},
	}
	// Orders for symbols that are not available pairs are skipped
	delistedOrder := OpenOrder{OrderID: "1321003749386327554", Symbol: "ZZZQUSDT", Side: "Buy", OrderType: "Limit", TimeInForce: "GTC"}
	btc := order.Detail{
		Exchange: ex.Name, OrderID: "1321003749386327552", ClientOrderID: "spot-test-01", Pair: currency.NewPairWithDelimiter("BTC", "USDT", "_"), AssetType: asset.Spot,
		Type: order.Limit, Side: order.Buy, Status: order.PartiallyFilled, TimeInForce: order.GoodTillCancel, Price: 15600, Amount: 0.1, AverageExecutedPrice: 15600,
		ExecutedAmount: 0.04, RemainingAmount: 0.06, Cost: 624, CostAsset: currency.USDT, Date: time.UnixMilli(1684738540559), LastUpdated: time.UnixMilli(1684738540561),
	}
	btc.SetFees(mustFees(t, order.FeeAmount{Currency: currency.BTC, Amount: 0.00004}))
	eth := order.Detail{
		Exchange: ex.Name, OrderID: "1321003749386327553", ClientOrderID: "spot-test-02", Pair: currency.NewPairWithDelimiter("ETH", "USDT", "_"), AssetType: asset.Spot,
		Type: order.Limit, Side: order.Sell, Status: order.New, TimeInForce: order.PostOnly, Price: 1900, Amount: 0.5, RemainingAmount: 0.5, CostAsset: currency.USDT,
		Date: time.UnixMilli(1684738600000), LastUpdated: time.UnixMilli(1684738600000),
	}
	eth.SetFees(mustFees(t))
	for _, tc := range []struct {
		name        string
		pair        currency.Pair
		filterPairs currency.Pairs
		exp         order.FilteredOrders
	}{
		{name: "pair", pair: currency.NewBTCUSDT(), exp: order.FilteredOrders{btc}},
		{name: "filter pairs", filterPairs: currency.Pairs{currency.NewPairWithDelimiter("ETH", "USDT", "_")}, exp: order.FilteredOrders{eth}},
		{name: "every pair", exp: order.FilteredOrders{btc, eth}},
	} {
		got, err := ex.ConstructOrderDetails([]OpenOrder{btcOrder, ethOrder, delistedOrder}, asset.Spot, tc.pair, tc.filterPairs)
		require.NoErrorf(t, err, "ConstructOrderDetails must not error for %s", tc.name)
		assert.Equalf(t, tc.exp, got, "ConstructOrderDetails should keep the orders of the %s", tc.name)
	}
	for _, tc := range []struct {
		field string
		order OpenOrder
		err   error
	}{
		{field: "side", order: OpenOrder{Symbol: "BTCUSDT", Side: "Hold", OrderType: "Limit", TimeInForce: "GTC"}, err: order.ErrSideIsInvalid},
		{field: "order type", order: OpenOrder{Symbol: "BTCUSDT", Side: "Buy", OrderType: "Iceberg", TimeInForce: "GTC"}, err: order.ErrUnrecognisedOrderType},
		{field: "time in force", order: OpenOrder{Symbol: "BTCUSDT", Side: "Buy", OrderType: "Limit", TimeInForce: "Forever"}, err: order.ErrInvalidTimeInForce},
	} {
		_, err := ex.ConstructOrderDetails([]OpenOrder{tc.order}, asset.Spot, currency.EMPTYPAIR, nil)
		assert.ErrorIsf(t, err, tc.err, "ConstructOrderDetails should reject an order with an unknown %s", tc.field)
	}
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderHistory(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrGetOrdersRequestIsNil, "GetOrderHistory must reject a nil request")
	_, err = e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Type: order.AnyType})
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "GetOrderHistory must require a side")
	_, err = e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Side: order.AnySide})
	require.ErrorIs(t, err, order.ErrUnrecognisedOrderType, "GetOrderHistory must require an order type")
	_, err = e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.Margin, Side: order.AnySide, Type: order.AnyType})
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetOrderHistory must reject an asset Bybit does not trade")
	_, err = restReplyTestInstance(t, nil).GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Side: order.AnySide, Type: order.AnyType})
	require.ErrorIs(t, err, errAPIResponse, "GetOrderHistory must return Bybit's refusal")
	// An order it cannot interpret fails the request rather than being reported wrongly
	unknownTimeInForce := restReplyTestInstance(t, map[string]string{"/v5/order/history": `{"retCode":0,"retMsg":"OK","result":{"category":"spot","nextPageCursor":"","list":[{"orderId":"1321003749386327598","orderLinkId":"spot-test-12","symbol":"BTCUSDT","price":"27000","qty":"0.01","side":"Sell","orderStatus":"Cancelled","timeInForce":"Forever","orderType":"Limit","createdTime":"1693087700000","updatedTime":"1693091300000","cumFeeDetail":{}}]},"retExtInfo":{},"time":1760000000000}`})
	_, err = unknownTimeInForce.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Side: order.AnySide, Type: order.AnyType})
	require.ErrorIs(t, err, order.ErrInvalidTimeInForce, "GetOrderHistory must reject an order whose time in force it does not know")
	for _, tc := range orderListingEnds {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := newPagingTestExchange(t, tc.pages).GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Side: order.AnySide, Type: order.AnyType})
			require.NoError(t, err, "GetOrderHistory must stop at the last page")
			ids := make([]string, len(got))
			for i := range got {
				ids[i] = got[i].OrderID
			}
			assert.Equal(t, tc.exp, ids, "GetOrderHistory should return the orders of each page up to the last")
		})
	}
	// Bybit returns only the week after a start time given alone, so the weeks up to now are requested one by one
	var mu sync.Mutex
	var queries []url.Values
	startOnly := newTransportTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		queries = append(queries, r.URL.Query())
		mu.Unlock()
		_, err := fmt.Fprint(w, `{"retCode":0,"retMsg":"OK","result":{"category":"linear","nextPageCursor":"","list":[]},"retExtInfo":{},"time":1760000000000}`)
		assert.NoError(t, err, "writing the reply should not error")
	})
	require.NoError(t, startOnly.DisableRateLimiter(), "DisableRateLimiter must not error")
	requested := time.Now()
	start := requested.Add(-20 * 24 * time.Hour)
	_, err = startOnly.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.USDTMarginedFutures, Side: order.AnySide, Type: order.AnyType, StartTime: start})
	require.NoError(t, err, "GetOrderHistory must not error for a start time alone")
	mu.Lock()
	require.Len(t, queries, 3, "GetOrderHistory must request each week from the start time to now")
	assert.Equal(t, strconv.FormatInt(start.UnixMilli(), 10), queries[0].Get("startTime"), "GetOrderHistory should start at the start time")
	lastEnd, err := strconv.ParseInt(queries[2].Get("endTime"), 10, 64)
	require.NoError(t, err, "ParseInt must not error for the last week's end time")
	assert.GreaterOrEqual(t, lastEnd, requested.UnixMilli(), "GetOrderHistory should fetch up to the time it was asked")
	for i, q := range queries {
		assert.NotEmptyf(t, q.Get("endTime"), "GetOrderHistory should bound week %d with an end time", i)
		if i > 0 {
			assert.Equalf(t, queries[i-1].Get("endTime"), q.Get("startTime"), "GetOrderHistory should start week %d where the one before it ended", i)
		}
	}
	// A range of exactly two weeks is two requests, the second ending at the range's end
	queries = nil
	mu.Unlock()
	end := start.Add(2 * orderHistoryWindow)
	_, err = startOnly.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.USDTMarginedFutures, Side: order.AnySide, Type: order.AnyType, StartTime: start, EndTime: end})
	require.NoError(t, err, "GetOrderHistory must not error for a two week range")
	mu.Lock()
	require.Len(t, queries, 2, "GetOrderHistory must request a two week range a week at a time")
	assert.Equal(t, strconv.FormatInt(end.UnixMilli(), 10), queries[1].Get("endTime"), "GetOrderHistory should end the last week at the range's end")
	// A range that ends in the future is fetched up to now, so no week starts after now
	queries = nil
	mu.Unlock()
	requested = time.Now()
	start = requested.Add(time.Hour - orderHistoryWindow)
	_, err = startOnly.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.USDTMarginedFutures, Side: order.AnySide, Type: order.AnyType, StartTime: start, EndTime: requested.Add(2 * time.Hour)})
	require.NoError(t, err, "GetOrderHistory must not error for a range ending in the future")
	mu.Lock()
	require.Len(t, queries, 1, "GetOrderHistory must request the week up to now alone")
	lastEnd, err = strconv.ParseInt(queries[0].Get("endTime"), 10, 64)
	require.NoError(t, err, "ParseInt must not error for the week's end time")
	assert.GreaterOrEqual(t, lastEnd, requested.UnixMilli(), "GetOrderHistory should fetch the week up to the time it was asked")
	assert.Less(t, lastEnd, requested.Add(time.Hour).UnixMilli(), "GetOrderHistory should end the week at the time it was asked")
	mu.Unlock()

	// An order of a requested pair whose contract has expired or been delisted since keeps that pair, as GetOrderInfo's
	// does, though its symbol is no longer an available pair, and an order of a listed pair keeps its own match
	historyOrder := `{"orderId":"%s","orderLinkId":"","symbol":"%s","side":"Buy","orderType":"Limit","timeInForce":"GTC","orderStatus":"Filled","qty":"0.1","price":"100","avgPrice":"100","cumExecQty":"0.1","cumExecValue":"10","leavesQty":"0","cumExecFee":"0.05","createdTime":"1792108800000","updatedTime":"1792108801000"}`
	historyOrders := make([]string, 0, 11)
	for _, o := range [][2]string{
		{"expired-order", "BTCUSDT-16OCT26"},
		{"listed-order", "BTCUSDT"},
		{"usdc-perpetual-order", "BTCPERP"},
		{"expired-usdc-order", "BTC-16OCT26"},
		{"expired-inverse-order", "BTCUSDU26"},
		{"expired-option-order", "BTC-16OCT26-70000-C"},
		{"expired-usdt-option-order", "BTC-16OCT26-70000-C-USDT"},
		{"delisted-spot-order", "XYZUSDT"},
		{"delisted-usdt-perpetual-order", "ETHUSDT"},
		{"delisted-usdc-perpetual-order", "ETHPERP"},
		{"delisted-inverse-perpetual-order", "ETHUSD"},
	} {
		historyOrders = append(historyOrders, fmt.Sprintf(historyOrder, o[0], o[1]))
	}
	expiredReply := `{"retCode":0,"retMsg":"OK","result":{"category":"linear","nextPageCursor":"","list":[` + strings.Join(historyOrders, ",") + `]},"retExtInfo":{},"time":1792700000000}`
	expiredContract := restReplyTestInstance(t, map[string]string{"/v5/order/history": expiredReply})
	// Only perpetuals and an option of another expiry remain available, so no dated symbol above is an available pair
	for a, available := range map[asset.Item]currency.Pairs{
		asset.USDTMarginedFutures: {currency.NewBTCUSDT()},
		asset.USDCMarginedFutures: {currency.NewPair(currency.BTC, currency.PERP)},
		asset.CoinMarginedFutures: {currency.NewBTCUSD()},
		asset.Options:             {currency.NewPairWithDelimiter("BTC", "30OCT26-70000-C", currency.DashDelimiter)},
	} {
		require.NoErrorf(t, expiredContract.CurrencyPairs.StorePairs(a, available, false), "StorePairs must not error for %s", a)
	}
	// A caller may name a pair with a delimiter, which Bybit's symbol does not carry
	expired := currency.NewPairWithDelimiter("BTC", "USDT-16OCT26", currency.DashDelimiter)
	expiredUSDC := currency.NewPair(currency.BTC, currency.NewCode("16OCT26"))
	expiredInverse := currency.NewPair(currency.BTC, currency.NewCode("USDU26"))
	expiredOption := currency.NewPairWithDelimiter("BTC", "16OCT26-70000-C", currency.DashDelimiter)
	expiredUSDTOption := currency.NewPairWithDelimiter("BTC", "16OCT26-70000-C-USDT", currency.DashDelimiter)
	delistedSpot := currency.NewPairWithDelimiter("XYZ", "USDT", "")
	delistedUSDT, delistedUSDC, delistedInverse := currency.NewPair(currency.ETH, currency.USDT), currency.NewPair(currency.ETH, currency.PERP), currency.NewPair(currency.ETH, currency.USD)
	type returned struct {
		id   string
		pair currency.Pair
	}
	for _, tc := range []struct {
		name  string
		asset asset.Item
		pairs currency.Pairs
		exp   []returned
	}{
		{name: "the expired contract requested", asset: asset.USDTMarginedFutures, pairs: currency.Pairs{expired, currency.NewBTCUSDT()}, exp: []returned{{"expired-order", expired}, {"listed-order", currency.NewBTCUSDT()}}},
		// Equal pairs, with and without a delimiter, name one symbol without making it ambiguous
		{name: "the expired contract requested twice", asset: asset.USDTMarginedFutures, pairs: currency.Pairs{expired, currency.NewPair(currency.BTC, currency.NewCode("USDT-16OCT26"))}, exp: []returned{{"expired-order", expired}}},
		{name: "no pair requested", asset: asset.USDTMarginedFutures, exp: []returned{{"listed-order", currency.NewBTCUSDT()}}},
		{name: "the empty pair, which asks for every pair", asset: asset.USDTMarginedFutures, pairs: currency.Pairs{currency.EMPTYPAIR}, exp: []returned{{"listed-order", currency.NewBTCUSDT()}}},
		// Neither of two unequal pairs naming one symbol is taken for its order, even when one is named again
		{name: "two unequal pairs of the expired contract's symbol", asset: asset.USDTMarginedFutures, pairs: currency.Pairs{expired, currency.NewPair(currency.NewCode("BTCUSDT-16"), currency.NewCode("OCT26")), expired}},
		{name: "an expired USDC margined future", asset: asset.USDCMarginedFutures, pairs: currency.Pairs{expiredUSDC}, exp: []returned{{"expired-usdc-order", expiredUSDC}}},
		{name: "an expired coin margined future", asset: asset.CoinMarginedFutures, pairs: currency.Pairs{expiredInverse}, exp: []returned{{"expired-inverse-order", expiredInverse}}},
		// A USDC settled option's symbol is a prefix of the USDT settled one's, and neither is taken for the other
		{name: "an expired option", asset: asset.Options, pairs: currency.Pairs{expiredOption}, exp: []returned{{"expired-option-order", expiredOption}}},
		{name: "an expired USDT settled option", asset: asset.Options, pairs: currency.Pairs{expiredUSDTOption}, exp: []returned{{"expired-usdt-option-order", expiredUSDTOption}}},
		{name: "a delisted spot pair", asset: asset.Spot, pairs: currency.Pairs{delistedSpot}, exp: []returned{{"delisted-spot-order", delistedSpot}}},
		{name: "a delisted USDT perpetual", asset: asset.USDTMarginedFutures, pairs: currency.Pairs{delistedUSDT}, exp: []returned{{"delisted-usdt-perpetual-order", delistedUSDT}}},
		{name: "a delisted USDC perpetual", asset: asset.USDCMarginedFutures, pairs: currency.Pairs{delistedUSDC}, exp: []returned{{"delisted-usdc-perpetual-order", delistedUSDC}}},
		{name: "a delisted inverse perpetual", asset: asset.CoinMarginedFutures, pairs: currency.Pairs{delistedInverse}, exp: []returned{{"delisted-inverse-perpetual-order", delistedInverse}}},
		// The linear category returns USDT and USDC margined orders alike, so a pair of one margin is not taken for an
		// order of the other's symbol
		{name: "a USDC margined pair of the USDT margined symbol", asset: asset.USDCMarginedFutures, pairs: currency.Pairs{currency.NewPair(currency.NewCode("BTCUSDT"), currency.NewCode("16OCT26"))}},
		{name: "a USDT margined pair of the USDC perpetual's symbol", asset: asset.USDTMarginedFutures, pairs: currency.Pairs{currency.NewPair(currency.BTC, currency.PERP)}},
	} {
		got, err := expiredContract.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: tc.asset, Side: order.AnySide, Type: order.AnyType, Pairs: tc.pairs})
		require.NoErrorf(t, err, "GetOrderHistory must not error for %s", tc.name)
		require.Lenf(t, got, len(tc.exp), "GetOrderHistory must return the expected orders for %s", tc.name)
		for _, o := range tc.exp {
			i := slices.IndexFunc(got, func(d order.Detail) bool { return d.OrderID == o.id })
			require.GreaterOrEqualf(t, i, 0, "GetOrderHistory must return %s for %s", o.id, tc.name)
			assert.Truef(t, o.pair.Equal(got[i].Pair), "GetOrderHistory should return %s with pair %s for %s", o.id, o.pair, tc.name)
		}
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		for _, a := range []asset.Item{asset.Spot, asset.USDTMarginedFutures, asset.USDCMarginedFutures, asset.CoinMarginedFutures, asset.Options} {
			_, err := e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: a, Side: order.AnySide, Type: order.AnyType})
			assert.NoErrorf(t, err, "GetOrderHistory should not error for %s", a)
		}
		return
	}

	orders := newOrderFixtures(t)
	for _, tc := range []struct {
		name string
		req  *order.MultiOrderRequest
		exp  order.FilteredOrders
	}{
		{
			// Two pages; the order for a symbol Bybit no longer lists is left out
			name: "spot",
			req:  &order.MultiOrderRequest{Pairs: currency.Pairs{spotTradablePair}, AssetType: asset.Spot, Side: order.AnySide, Type: order.AnyType},
			exp:  order.FilteredOrders{orders.spotFilledByValue, orders.spotCancelled},
		},
		{
			// Ten days are fetched a week at a time
			name: "USDT margined futures",
			req:  &order.MultiOrderRequest{AssetType: asset.USDTMarginedFutures, Side: order.AnySide, Type: order.AnyType, StartTime: time.UnixMilli(1788220800000), EndTime: time.UnixMilli(1789084800000)},
			exp:  order.FilteredOrders{orders.linearFilled, orders.linearCancelled},
		},
		{
			name: "USDC margined futures",
			req:  &order.MultiOrderRequest{Pairs: currency.Pairs{usdcMarginedTradablePair}, AssetType: asset.USDCMarginedFutures, Side: order.AnySide, Type: order.AnyType},
			exp:  order.FilteredOrders{orders.usdcFilled},
		},
		{
			name: "coin margined futures",
			req:  &order.MultiOrderRequest{Pairs: currency.Pairs{inverseTradablePair}, AssetType: asset.CoinMarginedFutures, Side: order.AnySide, Type: order.AnyType},
			exp:  order.FilteredOrders{orders.inverseFilled},
		},
		{
			name: "options",
			req:  &order.MultiOrderRequest{Pairs: currency.Pairs{optionsTradablePair}, AssetType: asset.Options, Side: order.AnySide, Type: order.AnyType},
			exp:  order.FilteredOrders{orders.optionFilled},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := e.GetOrderHistory(t.Context(), tc.req)
			require.NoError(t, err, "GetOrderHistory must not error")
			assert.Equal(t, tc.exp, got, "GetOrderHistory should return the closed orders")
		})
	}
}

func TestGetFeeByType(t *testing.T) {
	t.Parallel()
	_, err := e.GetFeeByType(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFeeByType must reject a nil fee builder")
	_, err = e.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyTradeFee, Amount: 1, PurchasePrice: 1})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetFeeByType must require a pair")
	_, err = e.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyWithdrawalFee, Pair: currency.NewBTCUSDT(), Amount: 1})
	require.ErrorIs(t, err, common.ErrFunctionNotSupported, "GetFeeByType must reject fee types other than trading fees")

	// Without credentials, or when told to skip them, the fee is a worst case estimate of 1%
	offline := testInstance(t)
	offline.SkipAuthCheck = true
	for _, feeType := range []exchange.FeeType{exchange.OfflineTradeFee, exchange.CryptocurrencyTradeFee} {
		fee, err := offline.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: feeType, Pair: currency.NewBTCUSDT(), Amount: 2, PurchasePrice: 30000})
		require.NoErrorf(t, err, "GetFeeByType must not error for fee type %v", feeType)
		assert.Equalf(t, 600.0, fee, "GetFeeByType should estimate the offline fee for fee type %v", feeType)
	}

	ex := testInstance(t)
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, ex)
	}
	ex.SkipAuthCheck = false
	_, err = ex.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyTradeFee, Pair: currency.NewPair(currency.XRP, currency.JPY), Amount: 1, PurchasePrice: 1})
	require.ErrorIs(t, err, currency.ErrPairNotFound, "GetFeeByType must reject a pair Bybit does not list")
	noFormat := noRequestFormatTestInstance(t, asset.Spot)
	noFormat.SkipAuthCheck = false
	_, err = noFormat.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyTradeFee, Pair: currency.NewBTCUSDT(), Amount: 1, PurchasePrice: 1})
	require.ErrorIs(t, err, currency.ErrPairFormatIsNil, "GetFeeByType must not request the fee rate of a pair whose symbol it cannot write")
	_, err = restReplyTestInstance(t, nil).GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyTradeFee, Pair: currency.NewBTCUSDT(), Amount: 1, PurchasePrice: 1})
	require.ErrorIs(t, err, errAPIResponse, "GetFeeByType must return Bybit's refusal to report the fee rate")
	if !mockTests {
		ensureTradablePairs(t)
		fee, err := ex.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyTradeFee, Pair: spotTradablePair, Amount: 1, PurchasePrice: 30000})
		require.NoError(t, err, "GetFeeByType must not error")
		assert.Positive(t, fee, "GetFeeByType should return the account's trading fee")
		return
	}
	// Without credentials, an inverse trade's value is its number of contracts, each worth one unit of the quote currency
	offlineInverse := testInstance(t)
	offlineInverse.SkipAuthCheck = true
	fee, err := offlineInverse.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.OfflineTradeFee, Pair: inverseTradablePair, Amount: 2, PurchasePrice: 30000})
	require.NoError(t, err, "GetFeeByType must not error for an offline coin margined fee")
	assert.Equal(t, 0.02, fee, "GetFeeByType should estimate an offline coin margined fee from the number of contracts")
	// The account has no fee rate for SOL options
	_, err = ex.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyTradeFee, Pair: currency.NewPairWithDelimiter("SOL", "4MAR26-85-C-USDT", "-"), Amount: 1, PurchasePrice: 1})
	require.ErrorIs(t, err, errFeeRateNotFound, "GetFeeByType must report a pair without a fee rate")
	for _, tc := range []struct {
		name  string
		pair  currency.Pair
		maker bool
		exp   float64
	}{
		// Spot lists BTC_USDT first, so its spot fee rate applies
		{name: "spot taker", pair: currency.NewBTCUSDT(), exp: 0.001 * 2 * 30000},
		{name: "spot maker", pair: currency.NewBTCUSDT(), maker: true, exp: 0.0008 * 2 * 30000},
		{name: "USDT margined futures", pair: usdtMarginedTradablePair, exp: 0.00055 * 2 * 30000},
		{name: "USDC margined futures", pair: usdcMarginedTradablePair, exp: 0.00055 * 2 * 30000},
		// An inverse contract is worth one US dollar, so its fee is the rate times the contracts
		{name: "coin margined futures maker", pair: inverseTradablePair, maker: true, exp: 0.0002 * 2},
		{name: "coin margined futures", pair: inverseTradablePair, exp: 0.00055 * 2},
		// Option fee rates are per base coin, and Bybit charges them on the underlying's index price
		{name: "options", pair: optionsTradablePair, exp: 0.0003 * 82551.02154501 * 2},
		{name: "options maker", pair: optionsTradablePair, maker: true, exp: 0.0002 * 82551.02154501 * 2},
	} {
		fee, err := ex.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyTradeFee, Pair: tc.pair, IsMaker: tc.maker, Amount: 2, PurchasePrice: 30000})
		require.NoErrorf(t, err, "GetFeeByType must not error for %s", tc.name)
		assert.InDeltaf(t, tc.exp, fee, 1e-9, "GetFeeByType should apply the account's %s fee rate", tc.name)
	}
	// Bybit charges an option trade at most 7% of the option's price
	capped, err := ex.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyTradeFee, Pair: optionsTradablePair, Amount: 2, PurchasePrice: 30})
	require.NoError(t, err, "GetFeeByType must not error for a cheap option")
	assert.InDelta(t, 0.07*30*2, capped, 1e-9, "GetFeeByType should cap an option's fee at 7 percent of its price")
	offlineOption, err := offlineInverse.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.OfflineTradeFee, Pair: optionsTradablePair, Amount: 2, PurchasePrice: 30})
	require.NoError(t, err, "GetFeeByType must not error for an offline option fee")
	assert.InDelta(t, 0.07*30*2, offlineOption, 1e-9, "GetFeeByType should estimate an offline option fee at Bybit's cap of 7 percent of its price")
	// An option's fee needs its underlying's index price, from the option's ticker
	optionFeeRate := `{"retCode":0,"retMsg":"OK","result":{"category":"option","list":[{"baseCoin":"BTC","takerFeeRate":"0.0003","makerFeeRate":"0.0002"}]},"retExtInfo":{},"time":1760000000000}`
	for _, tc := range []struct {
		name    string
		tickers string
		err     error
	}{
		{name: "a ticker Bybit refuses", tickers: `{"retCode":10001,"retMsg":"params error","result":{},"retExtInfo":{},"time":1760000000000}`, err: errAPIResponse},
		{name: "no ticker", tickers: `{"retCode":0,"retMsg":"OK","result":{"category":"option","list":[]},"retExtInfo":{},"time":1760000000000}`, err: ticker.ErrTickerNotFound},
	} {
		noTicker := restReplyTestInstance(t, map[string]string{"/v5/account/fee-rate": optionFeeRate, "/v5/market/tickers": tc.tickers})
		noTicker.SkipAuthCheck = false
		require.NoErrorf(t, noTicker.CurrencyPairs.StorePairs(asset.Options, currency.Pairs{optionsTradablePair}, false), "StorePairs must not error for %s", tc.name)
		_, err = noTicker.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyTradeFee, Pair: optionsTradablePair, Amount: 1, PurchasePrice: 30})
		assert.ErrorIsf(t, err, tc.err, "GetFeeByType should report %s", tc.name)
	}
	// Pro accounts' option fees are capped at a smaller share of the option's price, so the account's tier is read
	optionTicker := `{"retCode":0,"retMsg":"OK","result":{"category":"option","list":[{"symbol":"BTC-26NOV24-92000-C","indexPrice":"82551.02154501"}]},"retExtInfo":{},"time":1760000000000}`
	for _, tc := range []struct {
		name, vipLevel string
		price, exp     float64
	}{
		{name: "Pro 1", vipLevel: "PRO-1", price: 30, exp: 0.07 * 30 * 2},
		{name: "Pro 4", vipLevel: "PRO-4", price: 30, exp: 0.06 * 30 * 2},
		{name: "Pro 5", vipLevel: "PRO-5", price: 30, exp: 0.04 * 30 * 2},
		// The fee rate on the index price is 4.13% of this price: over Pro 5's cap but under 7%
		{name: "Pro 5", vipLevel: "PRO-5", price: 600, exp: 0.04 * 600 * 2},
	} {
		tiered := restReplyTestInstance(t, map[string]string{
			"/v5/account/fee-rate": optionFeeRate,
			"/v5/market/tickers":   optionTicker,
			"/v5/user/query-api":   `{"retCode":0,"retMsg":"","result":{"vipLevel":"` + tc.vipLevel + `"},"retExtInfo":{},"time":1760000000000}`,
		})
		tiered.SkipAuthCheck = false
		require.NoErrorf(t, tiered.CurrencyPairs.StorePairs(asset.Options, currency.Pairs{optionsTradablePair}, false), "StorePairs must not error for %s", tc.name)
		fee, err := tiered.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyTradeFee, Pair: optionsTradablePair, Amount: 2, PurchasePrice: tc.price})
		require.NoErrorf(t, err, "GetFeeByType must not error for a %s account at a price of %v", tc.name, tc.price)
		assert.InDeltaf(t, tc.exp, fee, 1e-9, "GetFeeByType should cap a %s account's option fee at its tier's share of the price %v", tc.name, tc.price)
	}
	noTier := restReplyTestInstance(t, map[string]string{
		"/v5/account/fee-rate": optionFeeRate,
		"/v5/market/tickers":   optionTicker,
		"/v5/user/query-api":   `{"retCode":10003,"retMsg":"API key is invalid.","result":{},"retExtInfo":{},"time":1760000000000}`,
	})
	noTier.SkipAuthCheck = false
	require.NoError(t, noTier.CurrencyPairs.StorePairs(asset.Options, currency.Pairs{optionsTradablePair}, false), "StorePairs must not error")
	_, err = noTier.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyTradeFee, Pair: optionsTradablePair, Amount: 1, PurchasePrice: 30})
	assert.ErrorIs(t, err, errAPIResponse, "GetFeeByType should report an account tier Bybit refuses")
	// A fee below every tier's cap does not depend on the tier, so a tier Bybit refuses does not stop its estimate
	uncapped, err := noTier.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyTradeFee, Pair: optionsTradablePair, Amount: 2, PurchasePrice: 30000})
	require.NoError(t, err, "GetFeeByType must not need the account's tier for a fee below every tier's cap")
	assert.InDelta(t, 0.0003*82551.02154501*2, uncapped, 1e-9, "GetFeeByType should charge the fee rate on the index price below every tier's cap")
	noOptionFormat := noRequestFormatTestInstance(t, asset.Options)
	noOptionFormat.SkipAuthCheck = false
	_, err = noOptionFormat.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyTradeFee, Pair: optionsTradablePair, Amount: 1, PurchasePrice: 30})
	assert.ErrorIs(t, err, currency.ErrPairFormatIsNil, "GetFeeByType should not request the ticker of an option whose symbol it cannot write")
}

func TestOptionFeeCap(t *testing.T) {
	t.Parallel()
	for vipLevel, exp := range map[string]float64{
		"No VIP": 0.07, "VIP-1": 0.07, "VIP-5": 0.07, "VIP-Supreme": 0.07, "PRO-1": 0.07, "": 0.07,
		"PRO-2": 0.06, "PRO-3": 0.06, "PRO-4": 0.06,
		"PRO-5": 0.04, "PRO-6": 0.04,
	} {
		assert.Equalf(t, exp, optionFeeCap(vipLevel), "optionFeeCap should return Bybit's option fee cap for %q", vipLevel)
	}
}

func TestGetOfflineTradeFee(t *testing.T) {
	t.Parallel()
	// The worst case is a fee of 1% of the trade's value
	assert.Equal(t, 125.0, getOfflineTradeFee(25000, 0.5), "getOfflineTradeFee should charge a hundredth of the trade's value")
	assert.Zero(t, getOfflineTradeFee(25000, 0), "getOfflineTradeFee should charge nothing for a trade of nothing")
}

func TestFeeAsset(t *testing.T) {
	t.Parallel()
	ex := testInstance(t)
	for _, tc := range []struct {
		pair currency.Pair
		exp  asset.Item
	}{
		// Spot lists BTC_USDT before USDT margined futures do
		{pair: currency.NewBTCUSDT(), exp: asset.Spot},
		{pair: currency.NewPairWithDelimiter("BTC", "USDT-25DEC26", "_"), exp: asset.USDTMarginedFutures},
		{pair: currency.NewPair(currency.BTC, currency.PERP), exp: asset.USDCMarginedFutures},
		{pair: currency.NewPair(currency.ADA, currency.USD), exp: asset.CoinMarginedFutures},
		{pair: currency.NewPairWithDelimiter("BTC", "13MAR26-48000-C-USDT", "-"), exp: asset.Options},
	} {
		a, err := ex.feeAsset(tc.pair)
		require.NoErrorf(t, err, "feeAsset must not error for %s", tc.pair)
		assert.Equalf(t, tc.exp, a, "feeAsset should return the first asset that lists %s", tc.pair)
	}
	_, err := ex.feeAsset(currency.NewPair(currency.XRP, currency.JPY))
	assert.ErrorIs(t, err, currency.ErrPairNotFound, "feeAsset should reject a pair no asset lists")

	// An asset without pairs is passed over
	ex.CurrencyPairs.Delete(asset.Spot)
	a, err := ex.feeAsset(currency.NewBTCUSDT())
	require.NoError(t, err, "feeAsset must not error without spot pairs")
	assert.Equal(t, asset.USDTMarginedFutures, a, "feeAsset should return the next asset that lists the pair")
}

func TestValidateAPICredentials(t *testing.T) {
	t.Parallel()
	err := e.ValidateAPICredentials(t.Context(), asset.Margin)
	require.ErrorIs(t, err, asset.ErrNotSupported, "ValidateAPICredentials must reject an asset Bybit does not trade")
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	// Credentials are valid when they can read the unified account's balances
	assert.NoError(t, testInstance(t).ValidateAPICredentials(t.Context(), asset.Spot), "ValidateAPICredentials should not error for valid credentials")
}

// candlePageTestCases holds a pair of each asset with candles and its first two hourly candles of 1 October 2026, which
// candlePageTestInstance requests one to a page
var candlePageTestCases = []struct {
	asset   asset.Item
	pair    currency.Pair
	candles []kline.Candle
}{
	{
		asset: asset.Spot,
		pair:  currency.NewBTCUSDT(),
		candles: []kline.Candle{
			{Time: time.UnixMilli(1790812800000), Open: 83616.2, High: 83656.8, Low: 83415.9, Close: 83509.1, Volume: 245.654482, QuoteVolume: 20513410.67309914},
			{Time: time.UnixMilli(1790816400000), Open: 83509.1, High: 83691, Low: 83471.8, Close: 83514.3, Volume: 268.805241, QuoteVolume: 22464396.7397634},
		},
	},
	{
		asset: asset.USDTMarginedFutures,
		pair:  currency.NewBTCUSDT(),
		candles: []kline.Candle{
			{Time: time.UnixMilli(1790812800000), Open: 83573.8, High: 83617.4, Low: 83374, Close: 83468.7, Volume: 976.028, QuoteVolume: 81466096.8998},
			{Time: time.UnixMilli(1790816400000), Open: 83468.7, High: 83656.1, Low: 83438.7, Close: 83469.7, Volume: 786.118, QuoteVolume: 65671255.4582},
		},
	},
	{
		asset: asset.USDCMarginedFutures,
		pair:  currency.NewPair(currency.BTC, currency.PERP),
		candles: []kline.Candle{
			{Time: time.UnixMilli(1790812800000), Open: 83527.5, High: 83568, Low: 83331.4, Close: 83419.2, Volume: 5.128, QuoteVolume: 427795.4665},
			{Time: time.UnixMilli(1790816400000), Open: 83419.2, High: 83604, Low: 83376.9, Close: 83425.1, Volume: 11.281, QuoteVolume: 941566.5783},
		},
	},
	{
		// An inverse candle's volume counts USD contracts and its turnover the base currency
		asset: asset.CoinMarginedFutures,
		pair:  currency.NewBTCUSD(),
		candles: []kline.Candle{
			{Time: time.UnixMilli(1790812800000), Open: 83523, High: 83568.2, Low: 83324.4, Close: 83425.2, Volume: 25.57477885, QuoteVolume: 2133426},
			{Time: time.UnixMilli(1790816400000), Open: 83425.2, High: 83600, Low: 83385.9, Close: 83413.8, Volume: 26.895959, QuoteVolume: 2245520},
		},
	},
}

// candlePageTestInstance returns an exchange of its own that requests one hourly candle at a time
func candlePageTestInstance(t *testing.T) *Exchange {
	t.Helper()
	ex := testInstance(t)
	ex.Features.Enabled.Kline.Intervals = kline.DeployExchangeIntervals(kline.IntervalCapacity{Interval: kline.OneHour, Capacity: 1})
	return ex
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	ensureTradablePairs(t)
	_, err := e.GetHistoricCandles(t.Context(), optionsTradablePair, asset.Options, kline.OneHour, time.Now().Add(-time.Hour*3), time.Now())
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetHistoricCandles must reject options, which have no candles")

	start, end := time.UnixMilli(1692748800000), time.UnixMilli(1693094400000)
	for _, tc := range []struct {
		asset    asset.Item
		pair     currency.Pair
		interval kline.Interval
		end      time.Time
		exp      *kline.Item
	}{
		{
			asset:    asset.Spot,
			pair:     spotTradablePair,
			interval: kline.OneDay,
			end:      end,
			exp: &kline.Item{
				Exchange: e.Name, Pair: currency.NewBTCUSDT(), Asset: asset.Spot, Interval: kline.OneDay,
				Candles: []kline.Candle{
					{Time: time.UnixMilli(1692748800000), Open: 26055.96, High: 26828.97, Low: 25815.02, Close: 26434.1, Volume: 9231.841678, QuoteVolume: 242606836.35109652},
					{Time: time.UnixMilli(1692835200000), Open: 26434.1, High: 26581.96, Low: 25861.66, Close: 26179.04, Volume: 5670.414738, QuoteVolume: 148770066.12487193},
					{Time: time.UnixMilli(1692921600000), Open: 26179.04, High: 26308, Low: 25780.01, Close: 26057.97, Volume: 7330.623157, QuoteVolume: 190802348.69902463},
					{Time: time.UnixMilli(1693008000000), Open: 26057.97, High: 26125, Low: 25986.86, Close: 26022.55, Volume: 1510.386762, QuoteVolume: 39329818.11324434},
				},
			},
		},
		{
			asset:    asset.USDTMarginedFutures,
			pair:     usdtMarginedTradablePair,
			interval: kline.OneDay,
			end:      end,
			exp: &kline.Item{
				Exchange: e.Name, Pair: currency.NewPairWithDelimiter("10000LADYS", "USDT", ""), Asset: asset.USDTMarginedFutures, Interval: kline.OneDay,
				Candles: []kline.Candle{
					{Time: time.UnixMilli(1692748800000), Open: 0.000252, High: 0.000274, Low: 0.0002485, Close: 0.0002675, Volume: 36510318200, QuoteVolume: 9621294.59825},
					{Time: time.UnixMilli(1692835200000), Open: 0.0002675, High: 0.000293, Low: 0.00026, Close: 0.0002875, Volume: 26041256900, QuoteVolume: 7131049.82345},
					{Time: time.UnixMilli(1692921600000), Open: 0.0002875, High: 0.000296, Low: 0.000272, Close: 0.0002805, Volume: 53354780600, QuoteVolume: 15188058.6729},
					{Time: time.UnixMilli(1693008000000), Open: 0.0002805, High: 0.00048, Low: 0.0002765, Close: 0.0003745, Volume: 302432632700, QuoteVolume: 114487750.11205},
				},
			},
		},
		{
			asset:    asset.USDCMarginedFutures,
			pair:     usdcMarginedTradablePair,
			interval: kline.OneDay,
			end:      end,
			exp: &kline.Item{
				Exchange: e.Name, Pair: currency.NewPairWithDelimiter("ETH", "PERP", ""), Asset: asset.USDCMarginedFutures, Interval: kline.OneDay,
				Candles: []kline.Candle{
					{Time: time.UnixMilli(1692748800000), Open: 1631.73, High: 1699.01, Low: 1627.54, Close: 1677.14, Volume: 8257.9, QuoteVolume: 13750594.9205},
					{Time: time.UnixMilli(1692835200000), Open: 1677.14, High: 1680.85, Low: 1633.17, Close: 1659.07, Volume: 4584.65, QuoteVolume: 7591676.2446},
					{Time: time.UnixMilli(1692921600000), Open: 1659.07, High: 1674.61, Low: 1632.87, Close: 1651.39, Volume: 4797.14, QuoteVolume: 7921436.5451},
					{Time: time.UnixMilli(1693008000000), Open: 1651.39, High: 1653.21, Low: 1641.98, Close: 1644.77, Volume: 720.1, QuoteVolume: 1186107.9245},
				},
			},
		},
		{
			// An inverse candle's volume counts USD contracts and its turnover the base currency
			asset:    asset.CoinMarginedFutures,
			pair:     inverseTradablePair,
			interval: kline.OneHour,
			end:      time.UnixMilli(1692759600000),
			exp: &kline.Item{
				Exchange: e.Name, Pair: currency.NewPairWithDelimiter("ADA", "USD", ""), Asset: asset.CoinMarginedFutures, Interval: kline.OneHour,
				Candles: []kline.Candle{
					{Time: time.UnixMilli(1692748800000), Open: 0.2578, High: 0.2593, Low: 0.2578, Close: 0.2583, Volume: 180681.48601755, QuoteVolume: 46763},
					{Time: time.UnixMilli(1692752400000), Open: 0.2583, High: 0.26, Low: 0.2581, Close: 0.2594, Volume: 245056.69708332, QuoteVolume: 63462},
					{Time: time.UnixMilli(1692756000000), Open: 0.2594, High: 0.26, Low: 0.2586, Close: 0.2586, Volume: 70399.64984162, QuoteVolume: 18254},
				},
			},
		},
	} {
		t.Run(tc.asset.String(), func(t *testing.T) {
			t.Parallel()
			if !mockTests {
				liveEnd := time.Now().Truncate(tc.interval.Duration())
				candles, err := e.GetHistoricCandles(t.Context(), tc.pair, tc.asset, tc.interval, liveEnd.Add(-tc.interval.Duration()*3), liveEnd)
				require.NoError(t, err, "GetHistoricCandles must not error")
				assert.Len(t, candles.Candles, 3, "GetHistoricCandles should return a candle per interval")
				return
			}
			candles, err := e.GetHistoricCandles(t.Context(), tc.pair, tc.asset, tc.interval, start, tc.end)
			require.NoError(t, err, "GetHistoricCandles must not error")
			assert.Equal(t, tc.exp, candles, "GetHistoricCandles should return the candles within the time range, oldest first")
		})
	}

	for _, tc := range candlePageTestCases {
		t.Run(tc.asset.String()+" full page", func(t *testing.T) {
			t.Parallel()
			ex := candlePageTestInstance(t)
			candles, err := ex.GetHistoricCandles(t.Context(), tc.pair, tc.asset, kline.OneHour, tc.candles[0].Time, tc.candles[1].Time)
			require.NoError(t, err, "GetHistoricCandles must not error")
			exp := &kline.Item{Exchange: ex.Name, Pair: tc.pair, Asset: tc.asset, Interval: kline.OneHour, Candles: tc.candles[:1]}
			assert.Equal(t, exp, candles, "GetHistoricCandles should return the first candle of a full page")
		})
	}

	// A USDC margined futures contract is asked for by its dashed symbol; Bybit lists none now, but still serves the
	// candles of the delivered BTC-26DEC25, which do not change
	t.Run("USDC margined futures contract", func(t *testing.T) {
		t.Parallel()
		ex := testInstance(t)
		usdcFutures := currency.NewPair(currency.BTC, currency.NewCode("26DEC25"))
		require.NoError(t, ex.CurrencyPairs.StorePairs(asset.USDCMarginedFutures, currency.Pairs{usdcFutures}, false), "StorePairs must not error for available pairs")
		require.NoError(t, ex.CurrencyPairs.StorePairs(asset.USDCMarginedFutures, currency.Pairs{usdcFutures}, true), "StorePairs must not error for enabled pairs")
		candles, err := ex.GetHistoricCandles(t.Context(), usdcFutures, asset.USDCMarginedFutures, kline.OneDay, time.UnixMilli(1766448000000), time.UnixMilli(1766707200000))
		require.NoError(t, err, "GetHistoricCandles must not error")
		exp := &kline.Item{
			Exchange: ex.Name, Pair: usdcFutures, Asset: asset.USDCMarginedFutures, Interval: kline.OneDay,
			Candles: []kline.Candle{
				{Time: time.UnixMilli(1766448000000), Open: 88500, High: 88898, Low: 86750, Close: 87416.5, Volume: 1.243, QuoteVolume: 109112.4735},
				{Time: time.UnixMilli(1766534400000), Open: 87416.5, High: 88000, Low: 86456, Close: 87750, Volume: 5.835, QuoteVolume: 509714.9335},
				{Time: time.UnixMilli(1766620800000), Open: 87750, High: 88571, Low: 86920, Close: 87245.5, Volume: 36.598, QuoteVolume: 3214955.859},
			},
		}
		assert.Equal(t, exp, candles, "GetHistoricCandles should return the contract's candles within the time range, oldest first")
	})

	// More candles than one request serves are refused before a request is made; GetHistoricCandlesExtended fetches them
	_, err = e.GetHistoricCandles(t.Context(), spotTradablePair, asset.Spot, kline.OneMin, start, start.Add(24*time.Hour))
	require.ErrorIs(t, err, kline.ErrRequestExceedsExchangeLimits, "GetHistoricCandles must refuse more candles than one request serves")

	ex := marketReplyInstance(t, map[string]string{"/v5/market/kline?category=spot&end=1693094399999&interval=D&limit=1000&start=1692748800000&symbol=BTCUSDT": marketErrorReply})
	_, err = ex.GetHistoricCandles(t.Context(), currency.NewBTCUSDT(), asset.Spot, kline.OneDay, start, end)
	require.ErrorIs(t, err, errAPIResponse, "GetHistoricCandles must return the error of a refused request")
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	ensureTradablePairs(t)
	_, err := e.GetHistoricCandlesExtended(t.Context(), optionsTradablePair, asset.Options, kline.FiveMin, time.Now().Add(-time.Hour), time.Now())
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetHistoricCandlesExtended must reject options, which have no candles")

	start, end := time.UnixMilli(1692889428738), time.UnixMilli(1693145028738)
	for _, tc := range []struct {
		asset    asset.Item
		pair     currency.Pair
		interval kline.Interval
		count    int
		exp      *kline.Item
	}{
		{
			// 4261 minutes take five requests; the fixtures hold the newest and oldest candle of each, the rest being padding
			asset:    asset.Spot,
			pair:     spotTradablePair,
			interval: kline.OneMin,
			count:    4261,
			exp: &kline.Item{
				Exchange: e.Name, Pair: currency.NewBTCUSDT(), Asset: asset.Spot, Interval: kline.OneMin,
				Candles: []kline.Candle{
					{Time: time.UnixMilli(1692889380000), Open: 26117.96, High: 26139.97, Low: 26117.96, Close: 26139.97, Volume: 9.783565, QuoteVolume: 255650.35615606},
					{Time: time.UnixMilli(1692949320000), Open: 26024.01, High: 26024.01, Low: 26017.44, Close: 26017.44, Volume: 0.569562, QuoteVolume: 14820.28897415},
					{Time: time.UnixMilli(1692949380000), Open: 26017.44, High: 26017.99, Low: 26010.19, Close: 26010.19, Volume: 2.176552, QuoteVolume: 56622.34355863},
					{Time: time.UnixMilli(1693009320000), Open: 26057.65, High: 26057.66, Low: 26057.65, Close: 26057.66, Volume: 0.002532, QuoteVolume: 65.97798786},
					{Time: time.UnixMilli(1693009380000), Open: 26057.66, High: 26062.64, Low: 26057.66, Close: 26062.64, Volume: 3.958518, QuoteVolume: 103156.08937572},
					{Time: time.UnixMilli(1693069320000), Open: 26037.85, High: 26037.85, Low: 26034.25, Close: 26034.25, Volume: 0.676922, QuoteVolume: 17624.51093096},
					{Time: time.UnixMilli(1693069380000), Open: 26034.25, High: 26034.25, Low: 26034, Close: 26034.01, Volume: 1.007656, QuoteVolume: 26233.37843275},
					{Time: time.UnixMilli(1693129320000), Open: 26056.09, High: 26056.1, Low: 26056.09, Close: 26056.09, Volume: 0.014182, QuoteVolume: 369.52756245},
					{Time: time.UnixMilli(1693129380000), Open: 26056.09, High: 26056.1, Low: 26056.09, Close: 26056.1, Volume: 0.044087, QuoteVolume: 1148.73522573},
					{Time: time.UnixMilli(1693144980000), Open: 26081.97, High: 26083.97, Low: 26081.96, Close: 26083.96, Volume: 0.87951, QuoteVolume: 22940.8606566},
				},
			},
		},
		{
			// An inverse candle's volume counts USD contracts and its turnover the base currency
			asset:    asset.CoinMarginedFutures,
			pair:     inverseTradablePair,
			interval: kline.OneHour,
			count:    72,
			exp: &kline.Item{
				Exchange: e.Name, Pair: currency.NewPairWithDelimiter("ADA", "USD", ""), Asset: asset.CoinMarginedFutures, Interval: kline.OneHour,
				Candles: []kline.Candle{
					{Time: time.UnixMilli(1692889200000), Open: 0.2628, High: 0.2635, Low: 0.2614, Close: 0.2617, Volume: 483242.19194522, QuoteVolume: 126770},
					{Time: time.UnixMilli(1693144800000), Open: 0.2625, High: 0.2651, Low: 0.2625, Close: 0.2646, Volume: 453092.36315104, QuoteVolume: 119853},
				},
			},
		},
		{
			asset:    asset.USDTMarginedFutures,
			pair:     usdtMarginedTradablePair,
			interval: kline.OneDay,
			count:    4,
			exp: &kline.Item{
				Exchange: e.Name, Pair: currency.NewPairWithDelimiter("10000LADYS", "USDT", ""), Asset: asset.USDTMarginedFutures, Interval: kline.OneDay,
				Candles: []kline.Candle{
					{Time: time.UnixMilli(1692835200000), Open: 0.0002675, High: 0.000293, Low: 0.00026, Close: 0.0002875, Volume: 26041256900, QuoteVolume: 7131049.82345},
					{Time: time.UnixMilli(1692921600000), Open: 0.0002875, High: 0.000296, Low: 0.000272, Close: 0.0002805, Volume: 53354780600, QuoteVolume: 15188058.6729},
					{Time: time.UnixMilli(1693008000000), Open: 0.0002805, High: 0.00048, Low: 0.0002765, Close: 0.0003745, Volume: 302432632700, QuoteVolume: 114487750.11205},
					{Time: time.UnixMilli(1693094400000), Open: 0.0003745, High: 0.0003885, Low: 0.0003015, Close: 0.0003155, Volume: 179703986200, QuoteVolume: 60495658.01195},
				},
			},
		},
	} {
		t.Run(tc.asset.String(), func(t *testing.T) {
			t.Parallel()
			if !mockTests {
				liveEnd := time.Now().Truncate(tc.interval.Duration())
				candles, err := e.GetHistoricCandlesExtended(t.Context(), tc.pair, tc.asset, tc.interval, liveEnd.Add(-tc.interval.Duration()*3), liveEnd)
				require.NoError(t, err, "GetHistoricCandlesExtended must not error")
				assert.Len(t, candles.Candles, 3, "GetHistoricCandlesExtended should return a candle per interval")
				return
			}
			candles, err := e.GetHistoricCandlesExtended(t.Context(), tc.pair, tc.asset, tc.interval, start, end)
			require.NoError(t, err, "GetHistoricCandlesExtended must not error")
			require.Len(t, candles.Candles, tc.count, "GetHistoricCandlesExtended must return a candle per interval")
			candles.Candles = slices.DeleteFunc(candles.Candles, func(c kline.Candle) bool { return c.Volume == 0 })
			assert.Equal(t, tc.exp, candles, "GetHistoricCandlesExtended should return the fetched candles within the time range, oldest first")
		})
	}

	// Each of the two requests fills its page
	for _, tc := range candlePageTestCases {
		t.Run(tc.asset.String()+" full pages", func(t *testing.T) {
			t.Parallel()
			ex := candlePageTestInstance(t)
			candles, err := ex.GetHistoricCandlesExtended(t.Context(), tc.pair, tc.asset, kline.OneHour, tc.candles[0].Time, tc.candles[1].Time.Add(time.Hour))
			require.NoError(t, err, "GetHistoricCandlesExtended must not error")
			exp := &kline.Item{Exchange: ex.Name, Pair: tc.pair, Asset: tc.asset, Interval: kline.OneHour, Candles: tc.candles}
			assert.Equal(t, exp, candles, "GetHistoricCandlesExtended should return the first candle of each full page")
		})
	}

	// A USDC margined futures contract is asked for by its dashed symbol; Bybit lists none now, but still serves the
	// candles of the delivered BTC-26DEC25, which do not change
	t.Run("USDC margined futures contract", func(t *testing.T) {
		t.Parallel()
		ex := testInstance(t)
		usdcFutures := currency.NewPair(currency.BTC, currency.NewCode("26DEC25"))
		require.NoError(t, ex.CurrencyPairs.StorePairs(asset.USDCMarginedFutures, currency.Pairs{usdcFutures}, false), "StorePairs must not error for available pairs")
		require.NoError(t, ex.CurrencyPairs.StorePairs(asset.USDCMarginedFutures, currency.Pairs{usdcFutures}, true), "StorePairs must not error for enabled pairs")
		candles, err := ex.GetHistoricCandlesExtended(t.Context(), usdcFutures, asset.USDCMarginedFutures, kline.OneDay, time.UnixMilli(1766448000000), time.UnixMilli(1766707200000))
		require.NoError(t, err, "GetHistoricCandlesExtended must not error")
		exp := &kline.Item{
			Exchange: ex.Name, Pair: usdcFutures, Asset: asset.USDCMarginedFutures, Interval: kline.OneDay,
			Candles: []kline.Candle{
				{Time: time.UnixMilli(1766448000000), Open: 88500, High: 88898, Low: 86750, Close: 87416.5, Volume: 1.243, QuoteVolume: 109112.4735},
				{Time: time.UnixMilli(1766534400000), Open: 87416.5, High: 88000, Low: 86456, Close: 87750, Volume: 5.835, QuoteVolume: 509714.9335},
				{Time: time.UnixMilli(1766620800000), Open: 87750, High: 88571, Low: 86920, Close: 87245.5, Volume: 36.598, QuoteVolume: 3214955.859},
			},
		}
		assert.Equal(t, exp, candles, "GetHistoricCandlesExtended should return the contract's candles within the time range, oldest first")
	})

	_, err = e.GetHistoricCandlesExtended(t.Context(), currency.NewPair(currency.NewCode("ZZZQ"), currency.USDT), asset.Spot, kline.OneDay, start, end)
	require.ErrorIs(t, err, currency.ErrPairNotEnabled, "GetHistoricCandlesExtended must refuse a pair that is not enabled")

	ex := marketReplyInstance(t, map[string]string{"/v5/market/kline?category=spot&end=1693094399999&interval=D&limit=1000&start=1692748800000&symbol=BTCUSDT": marketErrorReply})
	_, err = ex.GetHistoricCandlesExtended(t.Context(), currency.NewBTCUSDT(), asset.Spot, kline.OneDay, time.UnixMilli(1692748800000), time.UnixMilli(1693094400000))
	require.ErrorIs(t, err, errAPIResponse, "GetHistoricCandlesExtended must return the error of a refused request")
}

func TestKlineCandles(t *testing.T) {
	t.Parallel()
	assert.Empty(t, klineCandles(nil, asset.Spot), "klineCandles should return no candles for none")
	// Bybit's BTCUSDT spot and BTCUSD inverse candles of the first hour of 1 October 2026
	for _, tc := range []struct {
		asset asset.Item
		item  KlineItem
		exp   kline.Candle
	}{
		{
			asset: asset.Spot,
			item:  KlineItem{StartTime: types.Time(time.UnixMilli(1790812800000)), OpenPrice: 83616.2, HighPrice: 83656.8, LowPrice: 83415.9, ClosePrice: 83509.1, Volume: 245.654482, Turnover: 20513410.67309914},
			exp:   kline.Candle{Time: time.UnixMilli(1790812800000), Open: 83616.2, High: 83656.8, Low: 83415.9, Close: 83509.1, Volume: 245.654482, QuoteVolume: 20513410.67309914},
		},
		{
			// An inverse candle's volume counts USD contracts and its turnover the base currency
			asset: asset.CoinMarginedFutures,
			item:  KlineItem{StartTime: types.Time(time.UnixMilli(1790812800000)), OpenPrice: 83523, HighPrice: 83568.2, LowPrice: 83324.4, ClosePrice: 83425.2, Volume: 2133426, Turnover: 25.57477885},
			exp:   kline.Candle{Time: time.UnixMilli(1790812800000), Open: 83523, High: 83568.2, Low: 83324.4, Close: 83425.2, Volume: 25.57477885, QuoteVolume: 2133426},
		},
	} {
		assert.Equalf(t, []kline.Candle{tc.exp}, klineCandles([]KlineItem{tc.item}, tc.asset), "klineCandles should convert a %s candle", tc.asset)
	}
	items := []KlineItem{
		{StartTime: types.Time(time.UnixMilli(1790812800000)), OpenPrice: 83616.2, HighPrice: 83656.8, LowPrice: 83415.9, ClosePrice: 83509.1, Volume: 245.654482, Turnover: 20513410.67309914},
		{StartTime: types.Time(time.UnixMilli(1790816400000)), OpenPrice: 83509.1, HighPrice: 83620, LowPrice: 83480.5, ClosePrice: 83601.7, Volume: 198.36, Turnover: 16571326.5},
	}
	exp := []kline.Candle{
		{Time: time.UnixMilli(1790812800000), Open: 83616.2, High: 83656.8, Low: 83415.9, Close: 83509.1, Volume: 245.654482, QuoteVolume: 20513410.67309914},
		{Time: time.UnixMilli(1790816400000), Open: 83509.1, High: 83620, Low: 83480.5, Close: 83601.7, Volume: 198.36, QuoteVolume: 16571326.5},
	}
	assert.Equal(t, exp, klineCandles(items, asset.Spot), "klineCandles should convert every candle, in turn")
}

func TestGetServerTime(t *testing.T) {
	t.Parallel()
	_, err := marketReplyInstance(t, map[string]string{"/v5/market/time?": marketErrorReply}).GetServerTime(t.Context(), asset.Empty)
	require.ErrorIs(t, err, errAPIResponse, "GetServerTime must return the error of a refused request")

	serverTime, err := e.GetServerTime(t.Context(), asset.Empty)
	require.NoError(t, err, "GetServerTime must not error")
	if mockTests {
		assert.Equal(t, time.Unix(0, 1791534103963033914), serverTime, "GetServerTime should return the server time to the nanosecond")
		return
	}
	assert.WithinDuration(t, time.Now(), serverTime, time.Minute, "GetServerTime should return the current time")
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	err := e.UpdateOrderExecutionLimits(t.Context(), asset.Futures)
	require.ErrorIs(t, err, asset.ErrNotSupported, "UpdateOrderExecutionLimits must reject an asset Bybit does not trade")

	for _, tc := range []struct {
		name    string
		asset   asset.Item
		replies map[string]string
	}{
		{name: "instruments refused", asset: asset.Spot, replies: map[string]string{"/v5/market/instruments-info?category=spot&limit=1000": marketErrorReply}},
		{name: "option base coins refused", asset: asset.Options, replies: map[string]string{"/v5/market/option-base-coins?underlyingType=0": marketErrorReply}},
		{
			name: "option instruments refused", asset: asset.Options,
			replies: map[string]string{"/v5/market/option-base-coins?underlyingType=0": btcOptionBaseCoinsReply, "/v5/market/instruments-info?baseCoin=BTC&category=option&limit=1000&status=Trading": marketErrorReply},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := marketReplyInstance(t, tc.replies).UpdateOrderExecutionLimits(t.Context(), tc.asset)
			assert.ErrorIs(t, err, errAPIResponse, "UpdateOrderExecutionLimits should return the error of a refused request")
		})
	}

	// Bybit checks a spot order against maxLimitOrderQty or maxMarketOrderQty and minOrderAmt, not the deprecated
	// minOrderQty, maxOrderQty and maxOrderAmt, which its instruments set to the same values; each field of this BTCUSDT
	// instrument differs, so a limit read from the wrong field shows
	t.Run("spot order size fields", func(t *testing.T) {
		t.Parallel()
		ex := marketReplyInstance(t, map[string]string{"/v5/market/instruments-info?category=spot&limit=1000": `{"retCode":0,"retMsg":"OK","result":{"category":"spot","list":[{"symbolId":9,"symbol":"BTCUSDT","baseCoin":"BTC","quoteCoin":"USDT","innovation":"0","status":"Trading","marginTrading":"utaOnly","stTag":"0","lotSizeFilter":{"basePrecision":"0.000001","quotePrecision":"0.0000001","minOrderQty":"0.00002","maxOrderQty":"250","minOrderAmt":"5","maxOrderAmt":"8000000","maxLimitOrderQty":"230","maxMarketOrderQty":"120","postOnlyMaxLimitOrderSize":"1150"},"priceFilter":{"tickSize":"0.1"},"riskParameters":{"priceLimitRatioX":"0.005","priceLimitRatioY":"0.01"},"symbolType":"","xstockMultiplier":"1"}]},"retExtInfo":{},"time":1791897732915}`})
		require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Spot, currency.Pairs{currency.NewBTCUSDT()}, false), "StorePairs must not error")
		require.NoError(t, ex.UpdateOrderExecutionLimits(t.Context(), asset.Spot), "UpdateOrderExecutionLimits must not error")
		l, err := ex.GetOrderExecutionLimits(asset.Spot, currency.NewBTCUSDT())
		require.NoError(t, err, "GetOrderExecutionLimits must not error")
		l.UpdatedAt = time.Time{}
		exp := limits.MinMaxLevel{Key: key.NewExchangeAssetPair(ex.Name, asset.Spot, currency.NewBTCUSDT()), PriceStepIncrementSize: 0.1, PriceDivisor: 1, MultiplierDecimal: 1, MaximumBaseAmount: 230, MarketMaxQty: 120, MinimumQuoteAmount: 5, AmountStepIncrementSize: 0.000001, QuoteStepIncrementSize: 0.0000001}
		assert.Equal(t, exp, l, "UpdateOrderExecutionLimits should load the limits Bybit checks spot orders against")
		// The limits hold a market order to maxMarketOrderQty and a limit order to maxLimitOrderQty
		assert.ErrorIs(t, l.Validate(80000, 150, order.Market), limits.ErrMarketAmountExceedsMax, "Validate should refuse a market order above maxMarketOrderQty")
		assert.NoError(t, l.Validate(80000, 150, order.Limit), "Validate should accept a limit order within maxLimitOrderQty")
		assert.ErrorIs(t, l.Validate(80000, 240, order.Limit), limits.ErrAmountExceedsMax, "Validate should refuse a limit order above maxLimitOrderQty")
	})

	// Of Bybit's BTCUSDT and ETHUSDT spot instruments, only the pair the exchange lists gets limits; a delivery time,
	// which spot instruments do not have, is ignored
	for _, tc := range []struct {
		name  string
		reply string
	}{
		{name: "instrument of an unlisted pair", reply: `{"retCode":0,"retMsg":"OK","result":{"category":"spot","list":[{"symbolId":9,"symbol":"BTCUSDT","baseCoin":"BTC","quoteCoin":"USDT","innovation":"0","status":"Trading","marginTrading":"utaOnly","stTag":"0","lotSizeFilter":{"basePrecision":"0.000001","quotePrecision":"0.0000001","minOrderQty":"0.000001","maxOrderQty":"230","minOrderAmt":"5","maxOrderAmt":"8000000","maxLimitOrderQty":"230","maxMarketOrderQty":"120","postOnlyMaxLimitOrderSize":"1150"},"priceFilter":{"tickSize":"0.1"},"riskParameters":{"priceLimitRatioX":"0.005","priceLimitRatioY":"0.01"},"symbolType":"","xstockMultiplier":"1"},{"symbolId":10,"symbol":"ETHUSDT","baseCoin":"ETH","quoteCoin":"USDT","innovation":"0","status":"Trading","marginTrading":"utaOnly","stTag":"0","lotSizeFilter":{"basePrecision":"0.00001","quotePrecision":"0.0000001","minOrderQty":"0.00001","maxOrderQty":"8118","minOrderAmt":"5","maxOrderAmt":"7000000","maxLimitOrderQty":"8118","maxMarketOrderQty":"2706","postOnlyMaxLimitOrderSize":"40590"},"priceFilter":{"tickSize":"0.01"},"riskParameters":{"priceLimitRatioX":"0.005","priceLimitRatioY":"0.01"},"symbolType":"","xstockMultiplier":"1"}]},"retExtInfo":{},"time":1791897732915}`},
		{name: "spot instrument with a delivery time", reply: `{"retCode":0,"retMsg":"OK","result":{"category":"spot","list":[{"symbolId":9,"symbol":"BTCUSDT","baseCoin":"BTC","quoteCoin":"USDT","innovation":"0","status":"Trading","marginTrading":"utaOnly","stTag":"0","deliveryTime":"1798185600000","lotSizeFilter":{"basePrecision":"0.000001","quotePrecision":"0.0000001","minOrderQty":"0.000001","maxOrderQty":"230","minOrderAmt":"5","maxOrderAmt":"8000000","maxLimitOrderQty":"230","maxMarketOrderQty":"120","postOnlyMaxLimitOrderSize":"1150"},"priceFilter":{"tickSize":"0.1"},"riskParameters":{"priceLimitRatioX":"0.005","priceLimitRatioY":"0.01"},"symbolType":"","xstockMultiplier":"1"},{"symbolId":10,"symbol":"ETHUSDT","baseCoin":"ETH","quoteCoin":"USDT","innovation":"0","status":"Trading","marginTrading":"utaOnly","stTag":"0","lotSizeFilter":{"basePrecision":"0.00001","quotePrecision":"0.0000001","minOrderQty":"0.00001","maxOrderQty":"8118","minOrderAmt":"5","maxOrderAmt":"7000000","maxLimitOrderQty":"8118","maxMarketOrderQty":"2706","postOnlyMaxLimitOrderSize":"40590"},"priceFilter":{"tickSize":"0.01"},"riskParameters":{"priceLimitRatioX":"0.005","priceLimitRatioY":"0.01"},"symbolType":"","xstockMultiplier":"1"}]},"retExtInfo":{},"time":1791897732915}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := marketReplyInstance(t, map[string]string{"/v5/market/instruments-info?category=spot&limit=1000": tc.reply})
			require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Spot, currency.Pairs{currency.NewBTCUSDT()}, false), "StorePairs must not error")
			require.NoError(t, ex.UpdateOrderExecutionLimits(t.Context(), asset.Spot), "UpdateOrderExecutionLimits must not error")
			l, err := ex.GetOrderExecutionLimits(asset.Spot, currency.NewBTCUSDT())
			require.NoError(t, err, "GetOrderExecutionLimits must not error")
			l.UpdatedAt = time.Time{}
			exp := limits.MinMaxLevel{Key: key.NewExchangeAssetPair(ex.Name, asset.Spot, currency.NewBTCUSDT()), PriceStepIncrementSize: 0.1, PriceDivisor: 1, MultiplierDecimal: 1, MaximumBaseAmount: 230, MarketMaxQty: 120, MinimumQuoteAmount: 5, AmountStepIncrementSize: 0.000001, QuoteStepIncrementSize: 0.0000001}
			assert.Equal(t, exp, l, "UpdateOrderExecutionLimits should load the listed pair's limits")
			_, err = ex.GetOrderExecutionLimits(asset.Spot, currency.NewPair(currency.ETH, currency.USDT))
			assert.ErrorIs(t, err, limits.ErrOrderLimitNotFound, "UpdateOrderExecutionLimits should load no limits for a pair the exchange does not list")
		})
	}

	if !mockTests {
		ex := tradablePairsTestInstance(t)
		for _, a := range ex.GetAssetTypes(false) {
			require.NoErrorf(t, ex.UpdateOrderExecutionLimits(t.Context(), a), "UpdateOrderExecutionLimits must not error for %s", a)
			pairs, err := ex.GetEnabledPairs(a)
			require.NoErrorf(t, err, "GetEnabledPairs must not error for %s", a)
			for _, p := range pairs {
				l, err := ex.GetOrderExecutionLimits(a, p)
				require.NoErrorf(t, err, "GetOrderExecutionLimits must not error for %s %s", a, p)
				minimum := l.MinimumBaseAmount
				if a == asset.Spot {
					// Bybit limits a spot order's value rather than its quantity
					minimum = l.MinimumQuoteAmount
				}
				assert.Positivef(t, minimum, "the minimum order size should be positive for %s %s", a, p)
				assert.Positivef(t, l.PriceDivisor, "PriceDivisor should be positive for %s %s", a, p)
				if a == asset.USDTMarginedFutures && !p.Quote.Equal(currency.USDT) {
					assert.NotZerof(t, l.Expiry, "Expiry should be set for the dated contract %s", p)
				}
			}
		}
		return
	}

	for _, tc := range []struct {
		asset asset.Item
		pair  currency.Pair
		exp   limits.MinMaxLevel
	}{
		{
			// Spot amounts step by the base precision, and spot instruments have no price bounds
			asset: asset.Spot,
			pair:  currency.NewBTCUSDT(),
			exp:   limits.MinMaxLevel{PriceStepIncrementSize: 0.1, PriceDivisor: 1, MultiplierDecimal: 1, MaximumBaseAmount: 230, MarketMaxQty: 120, MinimumQuoteAmount: 5, AmountStepIncrementSize: 0.000001, QuoteStepIncrementSize: 0.0000001},
		},
		{
			// A symbol that prices 10000 coins as one has its prices divided by 10000. The perpetual contract is being
			// delisted, which Bybit settles at the index price averaged over the 30 minutes before its delivery time
			asset: asset.USDTMarginedFutures,
			pair:  currency.NewPair(currency.NewCode("10000LADYS"), currency.USDT),
			exp:   limits.MinMaxLevel{MinPrice: 0.0000001, MaxPrice: 0.1999998, PriceStepIncrementSize: 0.0000001, PriceDivisor: 10000, MultiplierDecimal: 1, MinimumBaseAmount: 100, MaximumBaseAmount: 67110000, MinimumQuoteAmount: 5, AmountStepIncrementSize: 100, MarketMaxQty: 6720000, Listed: time.UnixMilli(1683802102000), Delisting: time.UnixMilli(1735630200000), Delisted: time.UnixMilli(1735632000000)},
		},
		{
			asset: asset.USDTMarginedFutures,
			pair:  currency.NewPairWithDelimiter("BTC", "USDT-25DEC26", "_"),
			exp:   limits.MinMaxLevel{MinPrice: 0.1, MaxPrice: 1999999.8, PriceStepIncrementSize: 0.1, PriceDivisor: 1, MultiplierDecimal: 1, MinimumBaseAmount: 0.001, MaximumBaseAmount: 10, MinimumQuoteAmount: 5, AmountStepIncrementSize: 0.001, MarketMaxQty: 10, Listed: time.UnixMilli(1767945600000), Expiry: time.UnixMilli(1798185600000)},
		},
		{
			asset: asset.USDCMarginedFutures,
			pair:  currency.NewPair(currency.ETH, currency.PERP),
			exp:   limits.MinMaxLevel{MinPrice: 0.01, MaxPrice: 199999.98, PriceStepIncrementSize: 0.01, PriceDivisor: 1, MultiplierDecimal: 1, MinimumBaseAmount: 0.01, MaximumBaseAmount: 2500, MinimumQuoteAmount: 5, AmountStepIncrementSize: 0.01, MarketMaxQty: 500, Listed: time.UnixMilli(1659657600000)},
		},
		{
			asset: asset.CoinMarginedFutures,
			pair:  currency.NewPairWithDelimiter("BTC", "USDH26", "_"),
			exp:   limits.MinMaxLevel{MinPrice: 0.5, MaxPrice: 999999, PriceStepIncrementSize: 0.5, PriceDivisor: 1, MultiplierDecimal: 1, MinimumBaseAmount: 1, MaximumBaseAmount: 5000000, AmountStepIncrementSize: 1, MarketMaxQty: 1000000, Listed: time.UnixMilli(1757664000000), Expiry: time.UnixMilli(1774598400000)},
		},
		{
			asset: asset.Options,
			pair:  currency.NewPairWithDelimiter("BTC", "26NOV24-92000-C", "-"),
			exp:   limits.MinMaxLevel{MinPrice: 5, MaxPrice: 10000000, PriceStepIncrementSize: 5, PriceDivisor: 1, MultiplierDecimal: 1, MinimumBaseAmount: 0.01, MaximumBaseAmount: 500, AmountStepIncrementSize: 0.01, Listed: time.UnixMilli(1732348800000), Expiry: time.UnixMilli(1732608000000)},
		},
		{
			// From the second page of BTC options
			asset: asset.Options,
			pair:  currency.NewPairWithDelimiter("BTC", "26NOV24-90000-P", "-"),
			exp:   limits.MinMaxLevel{MinPrice: 5, MaxPrice: 10000000, PriceStepIncrementSize: 5, PriceDivisor: 1, MultiplierDecimal: 1, MinimumBaseAmount: 0.01, MaximumBaseAmount: 500, AmountStepIncrementSize: 0.01, Listed: time.UnixMilli(1732348800000), Expiry: time.UnixMilli(1732608000000)},
		},
	} {
		t.Run(tc.asset.String()+" "+tc.pair.String(), func(t *testing.T) {
			t.Parallel()
			ex := tradablePairsTestInstance(t)
			require.NoError(t, ex.UpdateOrderExecutionLimits(t.Context(), tc.asset), "UpdateOrderExecutionLimits must not error")
			l, err := ex.GetOrderExecutionLimits(tc.asset, tc.pair)
			require.NoError(t, err, "GetOrderExecutionLimits must not error")
			tc.exp.Key = key.NewExchangeAssetPair(ex.Name, tc.asset, tc.pair)
			l.UpdatedAt = time.Time{}
			assert.Equal(t, tc.exp, l, "UpdateOrderExecutionLimits should load the instrument's limits")
		})
	}
}

func TestUpdateOrderExecutionLimitsFiltersNonTradingOptions(t *testing.T) {
	t.Parallel()

	tradingPair := currency.NewPairWithDelimiter("BTC", "14MAR26-95000-C", "-")
	settlingPair := currency.NewPairWithDelimiter("BTC", "14MAR26-90000-C", "-")
	closedPair := currency.NewPairWithDelimiter("BTC", "14MAR26-85000-C", "-")
	tradingInstrument := newOptionInstrumentInfo("BTC14MAR26-95000-C", tradingStatus)
	settlingInstrument := newOptionInstrumentInfo("BTC14MAR26-90000-C", "Settling")
	closedInstrument := newOptionInstrumentInfo("BTC14MAR26-85000-C", "Closed")

	var queries []url.Values
	ex := newInstrumentInfoTestExchange(t, "BybitOptionsUpdateLimitsTest", cOption, map[string][]InstrumentInfo{
		"BTC": {tradingInstrument, settlingInstrument, closedInstrument},
	}, &queries)

	require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Options, currency.Pairs{tradingPair}, false), "StorePairs must not error for available option pairs")
	require.NoError(t, ex.UpdateOrderExecutionLimits(t.Context(), asset.Options), "UpdateOrderExecutionLimits must not error")

	loadedLimit, err := ex.GetOrderExecutionLimits(asset.Options, tradingPair)
	require.NoError(t, err, "GetOrderExecutionLimits must not error for a trading option pair")
	require.True(t, loadedLimit.Key.Pair().Equal(tradingPair), "Loaded limit pair must match the trading option pair")
	assert.Equal(t, tradingInstrument.LotSizeFilter.MinOrderQuantity.Float64(), loadedLimit.MinimumBaseAmount, "MinimumBaseAmount should match the trading option instrument")

	_, err = ex.GetOrderExecutionLimits(asset.Options, settlingPair)
	require.ErrorIs(t, err, limits.ErrOrderLimitNotFound, "GetOrderExecutionLimits must error for a settling option pair")

	_, err = ex.GetOrderExecutionLimits(asset.Options, closedPair)
	require.ErrorIs(t, err, limits.ErrOrderLimitNotFound, "GetOrderExecutionLimits must error for a closed option pair")

	assertOptionsInstrumentQueries(t, queries, "UpdateOrderExecutionLimits", "BTC")
}

func TestUpdateOrderExecutionLimitsLeavesNonOptionsStatusHandlingUnchanged(t *testing.T) {
	t.Parallel()

	spotPair := currency.NewPairWithDelimiter("BTC", "USDT", "_")
	closedInstrument := newOptionInstrumentInfo("BTCUSDT", "Closed")

	var queries []url.Values
	ex := newInstrumentInfoTestExchange(t, "BybitSpotUpdateLimitsTest", cSpot, map[string][]InstrumentInfo{
		"": {closedInstrument},
	}, &queries)

	require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Spot, currency.Pairs{spotPair}, false), "StorePairs must not error for available spot pairs")
	require.NoError(t, ex.UpdateOrderExecutionLimits(t.Context(), asset.Spot), "UpdateOrderExecutionLimits must not error for spot pairs")

	loadedLimit, err := ex.GetOrderExecutionLimits(asset.Spot, spotPair)
	require.NoError(t, err, "GetOrderExecutionLimits must not error for a closed spot pair")
	assert.Equal(t, closedInstrument.LotSizeFilter.BasePrecision.Float64(), loadedLimit.AmountStepIncrementSize, "AmountStepIncrementSize should match the closed spot instrument")

	require.Len(t, queries, 1, "UpdateOrderExecutionLimits must query spot instruments once")
	assert.Equal(t, cSpot, queries[0].Get("category"), "UpdateOrderExecutionLimits should request the spot category")
	assert.Empty(t, queries[0].Get("status"), "UpdateOrderExecutionLimits should not filter non-options instrument status")
	assert.Equal(t, "1000", queries[0].Get("limit"), "UpdateOrderExecutionLimits should request the expected page size")
}

func TestSetLeverage(t *testing.T) {
	t.Parallel()
	err := e.SetLeverage(t.Context(), asset.Spot, currency.NewBTCUSDT(), margin.Multi, 5, order.UnknownSide)
	require.ErrorIs(t, err, asset.ErrNotSupported, "SetLeverage must reject spot, which has no leverage to set")
	err = e.SetLeverage(t.Context(), asset.USDTMarginedFutures, currency.NewBTCUSDT(), margin.Isolated, 5, order.CouldNotBuy)
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "SetLeverage must reject a side that is not a position side")
	err = e.SetLeverage(t.Context(), asset.USDTMarginedFutures, currency.NewBTCUSDT(), margin.Isolated, 0, order.Buy)
	require.ErrorIs(t, err, errInvalidLeverage, "SetLeverage must reject leverage that is not positive")
	err = e.SetLeverage(t.Context(), asset.USDTMarginedFutures, currency.EMPTYPAIR, margin.Isolated, 5, order.Buy)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "SetLeverage must require a pair")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		ensureTradablePairs(t)
	}
	// Both sides take the leverage, which one-way mode and cross margin require
	for _, tc := range []struct {
		asset      asset.Item
		pair       currency.Pair
		marginType margin.Type
		side       order.Side
	}{
		{asset: asset.USDTMarginedFutures, pair: usdtMarginedTradablePair, marginType: margin.Multi, side: order.Buy},
		{asset: asset.USDCMarginedFutures, pair: usdcMarginedTradablePair, marginType: margin.Multi, side: order.Buy},
		{asset: asset.USDTMarginedFutures, pair: usdtMarginedTradablePair, marginType: margin.Isolated, side: order.Short},
		{asset: asset.CoinMarginedFutures, pair: inverseTradablePair, marginType: margin.Isolated, side: order.Sell},
		{asset: asset.CoinMarginedFutures, pair: inverseTradablePair, marginType: margin.Isolated, side: order.UnknownSide},
	} {
		err = e.SetLeverage(t.Context(), tc.asset, tc.pair, tc.marginType, 5, tc.side)
		if !mockTests && err != nil {
			// Bybit refuses to set the leverage a symbol already has
			assert.ErrorContainsf(t, err, "code 110043", "SetLeverage should only be refused when %s %s already has the leverage", tc.asset, tc.pair)
			continue
		}
		assert.NoErrorf(t, err, "SetLeverage should not error for %s %s %s", tc.asset, tc.marginType, tc.side)
	}
	if mockTests {
		// Bybit answers 110043 when the symbol already has the leverage
		err = e.SetLeverage(t.Context(), asset.USDTMarginedFutures, usdtMarginedTradablePair, margin.Multi, 10, order.Buy)
		assert.NoError(t, err, "SetLeverage should treat leverage the symbol already has as set")
		// Bybit names a dated USDC margined contract BTC-25DEC26, which the recorded request requires
		err = e.SetLeverage(t.Context(), asset.USDCMarginedFutures, currency.NewPairWithDelimiter("BTC", "25DEC26", "-"), margin.Multi, 5, order.Buy)
		assert.NoError(t, err, "SetLeverage should set a dated USDC margined contract's leverage by its Bybit symbol")
	}
}

func TestIsPerpetualFutureCurrency(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		asset asset.Item
		pair  currency.Pair
		exp   bool
	}{
		{asset: asset.Spot, pair: currency.NewBTCUSDT()},
		{asset: asset.Options, pair: currency.NewPairWithDelimiter("BTC", "26NOV24-92000-C", "-")},
		{asset: asset.CoinMarginedFutures, pair: currency.NewPair(currency.ADA, currency.USD), exp: true},
		{asset: asset.CoinMarginedFutures, pair: currency.NewPairWithDelimiter("BTC", "USDH26", "_")},
		{asset: asset.USDTMarginedFutures, pair: currency.NewBTCUSDT(), exp: true},
		{asset: asset.USDTMarginedFutures, pair: currency.NewPairWithDelimiter("BTC", "USDT-25DEC26", "_")},
		{asset: asset.USDCMarginedFutures, pair: currency.NewPair(currency.ETH, currency.PERP), exp: true},
	} {
		is, err := e.IsPerpetualFutureCurrency(tc.asset, tc.pair)
		require.NoErrorf(t, err, "IsPerpetualFutureCurrency must not error for %s %s", tc.asset, tc.pair)
		assert.Equalf(t, tc.exp, is, "IsPerpetualFutureCurrency should report whether %s %s is perpetual", tc.asset, tc.pair)
	}
}

func TestGetFuturesContractDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesContractDetails(t.Context(), asset.Spot)
	require.ErrorIs(t, err, futures.ErrNotFuturesAsset, "GetFuturesContractDetails must reject a spot asset")
	_, err = e.GetFuturesContractDetails(t.Context(), asset.Futures)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetFuturesContractDetails must reject a futures asset Bybit does not trade")

	for _, tc := range []struct {
		asset asset.Item
		exp   []futures.Contract
	}{
		{
			// USDT and USDC margined futures share the linear instruments, which come in two pages
			asset: asset.USDTMarginedFutures,
			exp: []futures.Contract{
				{Exchange: e.Name, Name: currency.NewPairWithDelimiter("BTC", "USDT", "_"), Underlying: currency.NewBTCUSDT(), Asset: asset.USDTMarginedFutures, StartDate: time.UnixMilli(1584230400000), IsActive: true, Status: tradingStatus, Type: futures.Perpetual, SettlementType: futures.Linear, SettlementCurrency: currency.USDT, MarginCurrency: currency.USDT, Multiplier: 1, MaxLeverage: 150, FundingRateFloor: decimal.MustFromFloat(-0.00333), FundingRateCeiling: decimal.MustFromFloat(0.00333)},
				// A perpetual contract ends when Bybit delists it
				{Exchange: e.Name, Name: currency.NewPairWithDelimiter("10000LADYS", "USDT", "_"), Underlying: currency.NewPair(currency.NewCode("10000LADYS"), currency.USDT), Asset: asset.USDTMarginedFutures, StartDate: time.UnixMilli(1683802102000), EndDate: time.UnixMilli(1735632000000), IsActive: true, Status: tradingStatus, Type: futures.Perpetual, SettlementType: futures.Linear, SettlementCurrency: currency.USDT, MarginCurrency: currency.USDT, Multiplier: 1, MaxLeverage: 25, FundingRateFloor: decimal.MustFromFloat(-0.015), FundingRateCeiling: decimal.MustFromFloat(0.015)},
				// A dated contract pays no funding
				{Exchange: e.Name, Name: currency.NewPairWithDelimiter("BTC", "USDT-25DEC26", "_"), Underlying: currency.NewBTCUSDT(), Asset: asset.USDTMarginedFutures, StartDate: time.UnixMilli(1767945600000), EndDate: time.UnixMilli(1798185600000), IsActive: true, Status: tradingStatus, Type: futures.Yearly, SettlementType: futures.Linear, SettlementCurrency: currency.USDT, MarginCurrency: currency.USDT, Multiplier: 1, MaxLeverage: 50, FundingRateFloor: decimal.MustFromFloat(0), FundingRateCeiling: decimal.MustFromFloat(0)},
			},
		},
		{
			asset: asset.USDCMarginedFutures,
			exp: []futures.Contract{
				{Exchange: e.Name, Name: currency.NewPairWithDelimiter("BTC", "PERP", "-"), Underlying: currency.NewPair(currency.BTC, currency.USDC), Asset: asset.USDCMarginedFutures, StartDate: time.UnixMilli(1610604231000), IsActive: true, Status: tradingStatus, Type: futures.Perpetual, SettlementType: futures.Linear, SettlementCurrency: currency.USDC, MarginCurrency: currency.USDC, Multiplier: 1, MaxLeverage: 125, FundingRateFloor: decimal.MustFromFloat(-0.005), FundingRateCeiling: decimal.MustFromFloat(0.005)},
				{Exchange: e.Name, Name: currency.NewPairWithDelimiter("ETH", "PERP", "-"), Underlying: currency.NewPair(currency.ETH, currency.USDC), Asset: asset.USDCMarginedFutures, StartDate: time.UnixMilli(1659657600000), IsActive: true, Status: tradingStatus, Type: futures.Perpetual, SettlementType: futures.Linear, SettlementCurrency: currency.USDC, MarginCurrency: currency.USDC, Multiplier: 1, MaxLeverage: 66.67, FundingRateFloor: decimal.MustFromFloat(-0.005), FundingRateCeiling: decimal.MustFromFloat(0.005)},
			},
		},
		{
			// Inverse contracts settle in their base coin
			asset: asset.CoinMarginedFutures,
			exp: []futures.Contract{
				{Exchange: e.Name, Name: currency.NewPairWithDelimiter("ADA", "USD", "_"), Underlying: currency.NewPair(currency.ADA, currency.USD), Asset: asset.CoinMarginedFutures, StartDate: time.UnixMilli(1648080000000), IsActive: true, Status: tradingStatus, Type: futures.Perpetual, SettlementType: futures.Inverse, SettlementCurrency: currency.ADA, MarginCurrency: currency.ADA, Multiplier: 1, MaxLeverage: 50, FundingRateFloor: decimal.MustFromFloat(-0.01), FundingRateCeiling: decimal.MustFromFloat(0.01)},
				{Exchange: e.Name, Name: currency.NewPairWithDelimiter("BTC", "USD", "_"), Underlying: currency.NewBTCUSD(), Asset: asset.CoinMarginedFutures, StartDate: time.UnixMilli(1542211200000), IsActive: true, Status: tradingStatus, Type: futures.Perpetual, SettlementType: futures.Inverse, SettlementCurrency: currency.BTC, MarginCurrency: currency.BTC, Multiplier: 1, MaxLeverage: 100, FundingRateFloor: decimal.MustFromFloat(-0.005), FundingRateCeiling: decimal.MustFromFloat(0.005)},
				{Exchange: e.Name, Name: currency.NewPairWithDelimiter("BTC", "USDH26", "_"), Underlying: currency.NewBTCUSD(), Asset: asset.CoinMarginedFutures, StartDate: time.UnixMilli(1757664000000), EndDate: time.UnixMilli(1774598400000), IsActive: true, Status: tradingStatus, Type: futures.HalfYearly, SettlementType: futures.Inverse, SettlementCurrency: currency.BTC, MarginCurrency: currency.BTC, Multiplier: 1, MaxLeverage: 100, FundingRateFloor: decimal.MustFromFloat(0), FundingRateCeiling: decimal.MustFromFloat(0)},
			},
		},
	} {
		t.Run(tc.asset.String(), func(t *testing.T) {
			t.Parallel()
			contracts, err := e.GetFuturesContractDetails(t.Context(), tc.asset)
			require.NoError(t, err, "GetFuturesContractDetails must not error")
			if mockTests {
				assert.Equal(t, tc.exp, contracts, "GetFuturesContractDetails should return the asset's contracts")
				return
			}
			require.NotEmpty(t, contracts, "GetFuturesContractDetails must return contracts")
			for i := range contracts {
				assert.Equalf(t, tc.asset, contracts[i].Asset, "GetFuturesContractDetails should return %s contracts", tc.asset)
				assert.Equalf(t, 1.0, contracts[i].Multiplier, "GetFuturesContractDetails should report a multiplier of 1 for %s", contracts[i].Name)
			}
		})
	}

	ex := marketReplyInstance(t, nil)
	ps, err := ex.CurrencyPairs.Get(asset.USDTMarginedFutures)
	require.NoError(t, err, "Get must not error")
	ps.ConfigFormat = nil
	require.NoError(t, ex.CurrencyPairs.Store(asset.USDTMarginedFutures, ps), "Store must not error")
	_, err = ex.GetFuturesContractDetails(t.Context(), asset.USDTMarginedFutures)
	require.ErrorIs(t, err, currency.ErrPairFormatIsNil, "GetFuturesContractDetails must report an asset without a pair format")

	const linearInstruments = "/v5/market/instruments-info?category=linear&limit=1000"
	for _, tc := range []struct {
		name  string
		reply string
		err   error
	}{
		{name: "instruments refused", reply: marketErrorReply, err: errAPIResponse},
		{name: "instrument without a symbol", reply: `{"retCode":0,"retMsg":"OK","result":{"category":"linear","list":[{"symbol":"","contractType":"LinearPerpetual","status":"Trading","baseCoin":"","quoteCoin":"USDT","settleCoin":"USDT"}],"nextPageCursor":""},"retExtInfo":{},"time":1791897742730}`, err: currency.ErrCreatingPair},
		{
			// Bybit's dated contract with its delivery time left out
			name: "dated contract without a delivery time", err: errInvalidContractLength,
			reply: `{"retCode":0,"retMsg":"OK","result":{"category":"linear","list":[{"symbol":"BTCUSDT-25DEC26","contractType":"LinearFutures","status":"Trading","baseCoin":"BTC","quoteCoin":"USDT","launchTime":"1767945600000","deliveryTime":"0","deliveryFeeRate":"0","priceScale":"2","leverageFilter":{"minLeverage":"1","maxLeverage":"50.00","leverageStep":"0.01"},"priceFilter":{"minPrice":"0.10","maxPrice":"1999999.80","tickSize":"0.10"},"lotSizeFilter":{"maxOrderQty":"10.000","minOrderQty":"0.001","qtyStep":"0.001","postOnlyMaxOrderQty":"10.000","maxMktOrderQty":"10.000","minNotionalValue":"5"},"unifiedMarginTrade":true,"fundingInterval":0,"settleCoin":"USDT","copyTrading":"none","upperFundingRate":"0","lowerFundingRate":"0","isPreListing":false,"preListingInfo":null,"riskParameters":{"priceLimitRatioX":"0.05","priceLimitRatioY":"0.1"},"displayName":"BTCUSDT-25DEC26","symbolType":"","forbidUplWithdrawal":false,"symbolId":1133,"fullName":"Bitcoin","marketRegion":"","underlyingTicker":"","tags":[]}],"nextPageCursor":""},"retExtInfo":{},"time":1791897742730}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := marketReplyInstance(t, map[string]string{linearInstruments: tc.reply}).GetFuturesContractDetails(t.Context(), asset.USDTMarginedFutures)
			assert.ErrorIs(t, err, tc.err, "GetFuturesContractDetails should return the error")
		})
	}

	// Bybit's BTCUSDT instrument with its contract type left out is returned as a contract of unknown type, and logged
	// when verbose
	ex = marketReplyInstance(t, map[string]string{linearInstruments: `{"retCode":0,"retMsg":"OK","result":{"category":"linear","list":[{"symbol":"BTCUSDT","contractType":"","status":"Trading","baseCoin":"BTC","quoteCoin":"USDT","launchTime":"1584230400000","deliveryTime":"0","deliveryFeeRate":"","priceScale":"2","leverageFilter":{"minLeverage":"1","maxLeverage":"150.00","leverageStep":"0.01"},"priceFilter":{"minPrice":"0.10","maxPrice":"1999999.80","tickSize":"0.10"},"lotSizeFilter":{"maxOrderQty":"1500.000","minOrderQty":"0.001","qtyStep":"0.001","postOnlyMaxOrderQty":"1500.000","maxMktOrderQty":"150.000","minNotionalValue":"5"},"unifiedMarginTrade":true,"fundingInterval":480,"settleCoin":"USDT","copyTrading":"both","upperFundingRate":"0.00333","lowerFundingRate":"-0.00333","isPreListing":false,"preListingInfo":null,"riskParameters":{"priceLimitRatioX":"0.01","priceLimitRatioY":"0.02"},"displayName":"BTCUSDT","symbolType":"","forbidUplWithdrawal":false,"symbolId":5,"fullName":"Bitcoin","marketRegion":"","underlyingTicker":"","tags":[]}],"nextPageCursor":""},"retExtInfo":{},"time":1791897742730}`})
	ex.Verbose = true
	contracts, err := ex.GetFuturesContractDetails(t.Context(), asset.USDTMarginedFutures)
	require.NoError(t, err, "GetFuturesContractDetails must not error")
	exp := []futures.Contract{{Exchange: ex.Name, Name: currency.NewPairWithDelimiter("BTC", "USDT", "_"), Underlying: currency.NewBTCUSDT(), Asset: asset.USDTMarginedFutures, StartDate: time.UnixMilli(1584230400000), IsActive: true, Status: tradingStatus, Type: futures.Unknown, SettlementType: futures.Linear, SettlementCurrency: currency.USDT, MarginCurrency: currency.USDT, Multiplier: 1, MaxLeverage: 150, FundingRateFloor: decimal.MustFromFloat(-0.00333), FundingRateCeiling: decimal.MustFromFloat(0.00333)}}
	assert.Equal(t, exp, contracts, "GetFuturesContractDetails should return a contract of unknown type")
}

func TestInstruments(t *testing.T) {
	t.Parallel()
	_, err := marketReplyInstance(t, map[string]string{"/v5/market/instruments-info?category=linear&limit=1000": marketErrorReply}).instruments(t.Context(), &InstrumentInfoRequest{Category: cLinear})
	require.ErrorIs(t, err, errAPIResponse, "instruments must return the error of a refused request")

	symbols := func(infos []InstrumentInfo) []string {
		s := make([]string, len(infos))
		for i := range infos {
			s[i] = infos[i].Symbol
		}
		return s
	}

	// Every page is asked for with the largest page size, whatever the request's limit, until Bybit sends no cursor
	req := &InstrumentInfoRequest{Category: cLinear, Limit: 5}
	infos, err := e.instruments(t.Context(), req)
	require.NoError(t, err, "instruments must not error")
	assert.Equal(t, &InstrumentInfoRequest{Category: cLinear, Limit: 5}, req, "instruments should leave the request unchanged")
	if mockTests {
		assert.Equal(t, []string{"BTCPERP", "BTCUSDT", "10000LADYSUSDT", "BTCUSDT-25DEC26", "ETHPERP"}, symbols(infos), "instruments should return the instruments of every page")
	} else {
		assert.NotEmpty(t, infos, "instruments should return the linear instruments")
	}

	// Paging also stops at a page without instruments, and at a cursor Bybit sends back unchanged
	const firstPage = `{"retCode":0,"retMsg":"OK","result":{"category":"linear","list":[{"symbol":"BTCPERP","contractType":"LinearPerpetual","status":"Trading"}],"nextPageCursor":"first%3DBTCPERP%26last%3DBTCPERP"},"retExtInfo":{},"time":1791897742730}`
	for _, tc := range []struct {
		name       string
		secondPage string
		exp        []string
	}{
		{name: "empty page", secondPage: `{"retCode":0,"retMsg":"OK","result":{"category":"linear","list":[],"nextPageCursor":"first%3DETHPERP%26last%3DETHPERP"},"retExtInfo":{},"time":1791897742730}`, exp: []string{"BTCPERP"}},
		{name: "repeated cursor", secondPage: `{"retCode":0,"retMsg":"OK","result":{"category":"linear","list":[{"symbol":"ETHPERP","contractType":"LinearPerpetual","status":"Trading"}],"nextPageCursor":"first%3DBTCPERP%26last%3DBTCPERP"},"retExtInfo":{},"time":1791897742730}`, exp: []string{"BTCPERP", "ETHPERP"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := marketReplyInstance(t, map[string]string{
				"/v5/market/instruments-info?category=linear&limit=1000":                                               firstPage,
				"/v5/market/instruments-info?category=linear&cursor=first%253DBTCPERP%2526last%253DBTCPERP&limit=1000": tc.secondPage,
			})
			infos, err := ex.instruments(t.Context(), &InstrumentInfoRequest{Category: cLinear})
			require.NoError(t, err, "instruments must not error")
			assert.Equal(t, tc.exp, symbols(infos), "instruments should stop paging")
		})
	}
}

func TestGetContractLength(t *testing.T) {
	t.Parallel()
	_, err := getContractLength(0)
	require.ErrorIs(t, err, errInvalidContractLength, "getContractLength must reject a contract without a length")
	_, err = getContractLength(-time.Hour)
	require.ErrorIs(t, err, errInvalidContractLength, "getContractLength must reject a contract that ends before it starts")
	for _, tc := range []struct {
		length time.Duration
		exp    futures.ContractType
	}{
		{length: kline.OneWeek.Duration(), exp: futures.Weekly},
		{length: kline.TwoWeek.Duration(), exp: futures.Fortnightly},
		{length: kline.ThreeWeek.Duration(), exp: futures.ThreeWeekly},
		{length: kline.ThreeMonth.Duration(), exp: futures.Quarterly},
		{length: 196 * kline.OneDay.Duration(), exp: futures.HalfYearly},
		{length: kline.NineMonth.Duration(), exp: futures.NineMonthly},
		{length: 350 * kline.OneDay.Duration(), exp: futures.Yearly},
		{length: 2 * kline.OneYear.Duration(), exp: futures.LongDated},
	} {
		ct, err := getContractLength(tc.length)
		require.NoErrorf(t, err, "getContractLength must not error for %s", tc.length)
		assert.Equalf(t, tc.exp, ct, "getContractLength should classify a %s contract", tc.length)
	}
}

func TestGetLatestFundingRates(t *testing.T) {
	t.Parallel()
	_, err := e.GetLatestFundingRates(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetLatestFundingRates must reject a nil request")
	_, err = e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.USDTMarginedFutures, IncludePredictedRate: true})
	require.ErrorIs(t, err, common.ErrFunctionNotSupported, "GetLatestFundingRates must reject a predicted rate request")
	for _, a := range []asset.Item{asset.Futures, asset.Spot, asset.Options} {
		_, err = e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: a})
		require.ErrorIsf(t, err, asset.ErrNotSupported, "GetLatestFundingRates must reject %s, which has no funding rates", a)
	}
	// A delisted contract has neither a ticker nor an instrument
	_, err = e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.CoinMarginedFutures, Pair: currency.NewPair(currency.EOS, currency.USD)})
	require.ErrorIs(t, err, futures.ErrNotPerpetualFuture, "GetLatestFundingRates must report a contract without a funding rate")

	ensureTradablePairs(t)
	for _, tc := range []struct {
		name string
		req  *fundingrate.LatestRateRequest
		exp  []fundingrate.LatestRateResponse
	}{
		{
			// Without a pair, every enabled pair of the asset; the funding interval dates the latest rate
			name: "USDT margined futures",
			req:  &fundingrate.LatestRateRequest{Asset: asset.USDTMarginedFutures},
			exp: []fundingrate.LatestRateResponse{
				{Exchange: e.Name, Asset: asset.USDTMarginedFutures, Pair: currency.NewPairWithDelimiter("BTC", "USDT", "_"), LatestRate: fundingrate.Rate{Time: time.UnixMilli(1791878400000), Rate: decimal.MustFromFloat(0.00008524)}, TimeOfNextRate: time.UnixMilli(1791907200000)},
				{Exchange: e.Name, Asset: asset.USDTMarginedFutures, Pair: currency.NewPair(currency.NewCode("10000LADYS"), currency.USDT), LatestRate: fundingrate.Rate{Time: time.UnixMilli(1704988800000), Rate: decimal.MustFromFloat(0.0001)}, TimeOfNextRate: time.UnixMilli(1705017600000)},
			},
		},
		{
			name: "USDC margined futures pair",
			req:  &fundingrate.LatestRateRequest{Asset: asset.USDCMarginedFutures, Pair: usdcMarginedTradablePair},
			exp: []fundingrate.LatestRateResponse{
				{Exchange: e.Name, Asset: asset.USDCMarginedFutures, Pair: currency.NewPairWithDelimiter("ETH", "PERP", "-"), LatestRate: fundingrate.Rate{Time: time.UnixMilli(1791878400000), Rate: decimal.MustFromFloat(0.00003911)}, TimeOfNextRate: time.UnixMilli(1791907200000)},
			},
		},
		{
			name: "coin margined futures pair",
			req:  &fundingrate.LatestRateRequest{Asset: asset.CoinMarginedFutures, Pair: inverseTradablePair},
			exp: []fundingrate.LatestRateResponse{
				{Exchange: e.Name, Asset: asset.CoinMarginedFutures, Pair: currency.NewPairWithDelimiter("ADA", "USD", "_"), LatestRate: fundingrate.Rate{Time: time.UnixMilli(1791878400000), Rate: decimal.MustFromFloat(0.00008907)}, TimeOfNextRate: time.UnixMilli(1791907200000)},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rates, err := e.GetLatestFundingRates(t.Context(), tc.req)
			require.NoError(t, err, "GetLatestFundingRates must not error")
			if !mockTests {
				require.NotEmpty(t, rates, "GetLatestFundingRates must return rates")
				for i := range rates {
					assert.NotZerof(t, rates[i].TimeOfNextRate, "GetLatestFundingRates should time the next rate of %s", rates[i].Pair)
				}
				return
			}
			for i := range rates {
				assert.WithinDuration(t, time.Now(), rates[i].TimeChecked, time.Minute, "GetLatestFundingRates should stamp when it checked the rate")
				rates[i].TimeChecked = time.Time{}
			}
			assert.Equal(t, tc.exp, rates, "GetLatestFundingRates should return the latest funding rates")
		})
	}

	ex := marketReplyInstance(t, nil)
	ps, err := ex.CurrencyPairs.Get(asset.USDTMarginedFutures)
	require.NoError(t, err, "Get must not error")
	ps.RequestFormat = nil
	require.NoError(t, ex.CurrencyPairs.Store(asset.USDTMarginedFutures, ps), "Store must not error")
	_, err = ex.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.USDTMarginedFutures, Pair: currency.NewBTCUSDT()})
	require.ErrorIs(t, err, currency.ErrPairFormatIsNil, "GetLatestFundingRates must report an asset without a request format")

	const linearTickers, linearInstruments = "/v5/market/tickers?category=linear", "/v5/market/instruments-info?category=linear&limit=1000"
	for _, tc := range []struct {
		name    string
		replies map[string]string
		err     error
	}{
		{name: "tickers refused", replies: map[string]string{linearTickers: marketErrorReply}, err: errAPIResponse},
		{
			name: "instruments refused", err: errAPIResponse,
			replies: map[string]string{
				linearTickers:     `{"retCode":0,"retMsg":"OK","result":{"category":"linear","list":[{"symbol":"BTCUSDT","lastPrice":"83451.40","indexPrice":"83486.85","markPrice":"83452.91","prevPrice24h":"82518.70","price24hPcnt":"0.011302","highPrice24h":"84138.60","lowPrice24h":"81000.00","prevPrice1h":"83368.10","openInterest":"56666.914","openInterestValue":"4729018874.02","singleOpenInterest":"28333.457","singleOpenInterestValue":"2364509437.01","turnover24h":"5945717007.8390","volume24h":"71815.9990","fundingRate":"0.00008524","nextFundingTime":"1791907200000","predictedDeliveryPrice":"","basisRate":"","deliveryFeeRate":"","deliveryTime":"0","ask1Size":"0.732","bid1Price":"83451.30","ask1Price":"83451.40","bid1Size":"3.856","basis":"","preOpenPrice":"","preQty":"","curPreListingPhase":"","fundingIntervalHour":"8","basisRateYear":"","fundingCap":"0.00333"}]},"retExtInfo":{},"time":1791898223522}`,
				linearInstruments: marketErrorReply,
			},
		},
		{
			// Bybit's BTCUSDT ticker with its symbol left out
			name: "ticker without a symbol", err: currency.ErrSymbolStringEmpty,
			replies: map[string]string{
				linearTickers:     `{"retCode":0,"retMsg":"OK","result":{"category":"linear","list":[{"symbol":"","lastPrice":"83451.40","indexPrice":"83486.85","markPrice":"83452.91","prevPrice24h":"82518.70","price24hPcnt":"0.011302","highPrice24h":"84138.60","lowPrice24h":"81000.00","prevPrice1h":"83368.10","openInterest":"56666.914","openInterestValue":"4729018874.02","singleOpenInterest":"28333.457","singleOpenInterestValue":"2364509437.01","turnover24h":"5945717007.8390","volume24h":"71815.9990","fundingRate":"0.00008524","nextFundingTime":"1791907200000","predictedDeliveryPrice":"","basisRate":"","deliveryFeeRate":"","deliveryTime":"0","ask1Size":"0.732","bid1Price":"83451.30","ask1Price":"83451.40","bid1Size":"3.856","basis":"","preOpenPrice":"","preQty":"","curPreListingPhase":"","fundingIntervalHour":"8","basisRateYear":"","fundingCap":"0.00333"}]},"retExtInfo":{},"time":1791898223522}`,
				linearInstruments: `{"retCode":0,"retMsg":"OK","result":{"category":"linear","list":[{"symbol":"BTCUSDT","contractType":"LinearPerpetual","status":"Trading","baseCoin":"BTC","quoteCoin":"USDT","launchTime":"1584230400000","deliveryTime":"0","deliveryFeeRate":"","priceScale":"2","leverageFilter":{"minLeverage":"1","maxLeverage":"150.00","leverageStep":"0.01"},"priceFilter":{"minPrice":"0.10","maxPrice":"1999999.80","tickSize":"0.10"},"lotSizeFilter":{"maxOrderQty":"1500.000","minOrderQty":"0.001","qtyStep":"0.001","postOnlyMaxOrderQty":"1500.000","maxMktOrderQty":"150.000","minNotionalValue":"5"},"unifiedMarginTrade":true,"fundingInterval":480,"settleCoin":"USDT","copyTrading":"both","upperFundingRate":"0.00333","lowerFundingRate":"-0.00333","isPreListing":false,"preListingInfo":null,"riskParameters":{"priceLimitRatioX":"0.01","priceLimitRatioY":"0.02"},"displayName":"BTCUSDT","symbolType":"","forbidUplWithdrawal":false,"symbolId":5,"fullName":"Bitcoin","marketRegion":"","underlyingTicker":"","tags":[]}],"nextPageCursor":""},"retExtInfo":{},"time":1791897742730}`,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := marketReplyInstance(t, tc.replies).GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.USDTMarginedFutures})
			assert.ErrorIs(t, err, tc.err, "GetLatestFundingRates should return the error")
		})
	}
}

func TestGetOpenInterest(t *testing.T) {
	t.Parallel()
	_, err := e.GetOpenInterest(t.Context(), key.PairAsset{Base: currency.ETH.Item, Quote: currency.USDT.Item, Asset: asset.Spot})
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetOpenInterest must reject spot, which has no open interest")
	// A delisted contract has no ticker
	_, err = e.GetOpenInterest(t.Context(), key.PairAsset{Base: currency.EOS.Item, Quote: currency.USD.Item, Asset: asset.CoinMarginedFutures})
	require.ErrorIs(t, err, currency.ErrPairNotFound, "GetOpenInterest must report a contract without a ticker")
	_, err = e.GetOpenInterest(t.Context(), key.PairAsset{Asset: asset.USDTMarginedFutures})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetOpenInterest must reject a key without a pair")
	for _, tc := range []struct {
		name    string
		keys    []key.PairAsset
		replies map[string]string
	}{
		{name: "ticker refused", keys: []key.PairAsset{{Base: currency.BTC.Item, Quote: currency.USDT.Item, Asset: asset.USDTMarginedFutures}}, replies: map[string]string{"/v5/market/tickers?category=linear&symbol=BTCUSDT": marketErrorReply}},
		// Without pairs, USDC margined futures' tickers are asked for first
		{name: "tickers refused", replies: map[string]string{"/v5/market/tickers?category=linear": marketErrorReply}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := marketReplyInstance(t, tc.replies).GetOpenInterest(t.Context(), tc.keys...)
			assert.ErrorIs(t, err, errAPIResponse, "GetOpenInterest should return the error of a refused request")
		})
	}

	ensureTradablePairs(t)
	usdc := key.PairAsset{Base: usdcMarginedTradablePair.Base.Item, Quote: usdcMarginedTradablePair.Quote.Item, Asset: asset.USDCMarginedFutures}
	usdt := key.PairAsset{Base: currency.BTC.Item, Quote: currency.USDT.Item, Asset: asset.USDTMarginedFutures}
	inverse := key.PairAsset{Base: inverseTradablePair.Base.Item, Quote: inverseTradablePair.Quote.Item, Asset: asset.CoinMarginedFutures}
	// Bybit's tickers count open interest on both sides of each contract and on one side, which is what GCT reports
	for _, tc := range []struct {
		name string
		keys []key.PairAsset
		exp  []futures.OpenInterest
	}{
		{name: "USDC margined futures pair", keys: []key.PairAsset{usdc}, exp: []futures.OpenInterest{{Key: key.NewExchangeAssetPair(e.Name, asset.USDCMarginedFutures, currency.NewPair(currency.ETH, currency.PERP)), OpenInterest: 6910.6}}},
		{name: "USDT margined futures pair", keys: []key.PairAsset{usdt}, exp: []futures.OpenInterest{{Key: key.NewExchangeAssetPair(e.Name, asset.USDTMarginedFutures, currency.NewBTCUSDT()), OpenInterest: 28999.236}}},
		{name: "coin margined futures pair", keys: []key.PairAsset{inverse}, exp: []futures.OpenInterest{{Key: key.NewExchangeAssetPair(e.Name, asset.CoinMarginedFutures, currency.NewPair(currency.ADA, currency.USD)), OpenInterest: 1742253}}},
		{
			// Several pairs are read from their categories' tickers
			name: "pairs of two assets",
			keys: []key.PairAsset{usdt, inverse},
			exp: []futures.OpenInterest{
				{Key: key.NewExchangeAssetPair(e.Name, asset.USDTMarginedFutures, currency.NewBTCUSDT()), OpenInterest: 28333.457},
				{Key: key.NewExchangeAssetPair(e.Name, asset.CoinMarginedFutures, currency.NewPair(currency.ADA, currency.USD)), OpenInterest: 1742253},
			},
		},
		{
			// Without pairs, every enabled futures pair; USDT and USDC margined futures share the linear tickers. The ticker
			// of the delisted 10000LADYSUSDT has no single side open interest
			name: "every enabled pair",
			exp: []futures.OpenInterest{
				{Key: key.NewExchangeAssetPair(e.Name, asset.USDCMarginedFutures, currency.NewPair(currency.BTC, currency.PERP)), OpenInterest: 668.26},
				{Key: key.NewExchangeAssetPair(e.Name, asset.USDCMarginedFutures, currency.NewPair(currency.ETH, currency.PERP)), OpenInterest: 6910.6},
				{Key: key.NewExchangeAssetPair(e.Name, asset.USDTMarginedFutures, currency.NewBTCUSDT()), OpenInterest: 28333.457},
				{Key: key.NewExchangeAssetPair(e.Name, asset.USDTMarginedFutures, currency.NewPair(currency.NewCode("10000LADYS"), currency.USDT))},
				{Key: key.NewExchangeAssetPair(e.Name, asset.CoinMarginedFutures, currency.NewPair(currency.ADA, currency.USD)), OpenInterest: 1742253},
				{Key: key.NewExchangeAssetPair(e.Name, asset.CoinMarginedFutures, currency.NewBTCUSD()), OpenInterest: 263402427},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			oi, err := e.GetOpenInterest(t.Context(), tc.keys...)
			require.NoError(t, err, "GetOpenInterest must not error")
			if mockTests {
				assert.Equal(t, tc.exp, oi, "GetOpenInterest should return the open interest of the pairs")
				return
			}
			require.NotEmpty(t, oi, "GetOpenInterest must return open interest")
			for i := range oi {
				assert.Positivef(t, oi[i].OpenInterest, "GetOpenInterest should return the open interest of %s", oi[i].Key)
			}
		})
	}
}

func TestGetCurrencyTradeURL(t *testing.T) {
	t.Parallel()
	_, err := e.GetCurrencyTradeURL(t.Context(), asset.Spot, currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetCurrencyTradeURL must reject an empty pair")
	_, err = e.GetCurrencyTradeURL(t.Context(), asset.Futures, currency.NewBTCUSDT())
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetCurrencyTradeURL must reject an asset Bybit does not trade")

	dated := currency.NewPairWithDelimiter("BTC", "USDZ26", "_")
	ex := marketReplyInstance(t, nil)
	require.NoError(t, ex.CurrencyPairs.Store(asset.Margin, &currency.PairStore{AssetEnabled: true, RequestFormat: &currency.PairFormat{Uppercase: true}, ConfigFormat: &currency.PairFormat{Uppercase: true, Delimiter: "_"}}), "Store must not error")
	_, err = ex.GetCurrencyTradeURL(t.Context(), asset.Margin, currency.NewBTCUSDT())
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetCurrencyTradeURL must reject an asset Bybit has no trade page for")
	ps, err := ex.CurrencyPairs.Get(asset.CoinMarginedFutures)
	require.NoError(t, err, "Get must not error")
	ps.RequestFormat = nil
	require.NoError(t, ex.CurrencyPairs.Store(asset.CoinMarginedFutures, ps), "Store must not error")
	_, err = ex.GetCurrencyTradeURL(t.Context(), asset.CoinMarginedFutures, dated)
	require.ErrorIs(t, err, currency.ErrPairFormatIsNil, "GetCurrencyTradeURL must report an asset without a request format")

	// A dated inverse contract's page is named for its length, which Bybit's instrument gives
	const datedInstrument = "/v5/market/instruments-info?category=inverse&limit=1000&symbol=BTCUSDZ26"
	for _, tc := range []struct {
		name  string
		reply string
		exp   string
		err   error
	}{
		{name: "instrument refused", reply: marketErrorReply, err: errAPIResponse},
		{
			// Bybit's instrument with its delivery time left out
			name: "contract without a delivery time", err: errInvalidContractLength,
			reply: `{"retCode":0,"retMsg":"OK","result":{"category":"inverse","list":[{"symbol":"BTCUSDZ26","contractType":"InverseFutures","status":"Trading","baseCoin":"BTC","quoteCoin":"USD","launchTime":"1781251200000","deliveryTime":"0","deliveryFeeRate":"0.0005","priceScale":"2","leverageFilter":{"minLeverage":"1","maxLeverage":"100.00","leverageStep":"0.01"},"priceFilter":{"minPrice":"0.50","maxPrice":"9999999.00","tickSize":"0.50"},"lotSizeFilter":{"maxOrderQty":"5000000","minOrderQty":"1","qtyStep":"1","postOnlyMaxOrderQty":"5000000","maxMktOrderQty":"1000000","minNotionalValue":"5"},"unifiedMarginTrade":true,"fundingInterval":0,"settleCoin":"BTC","copyTrading":"none","upperFundingRate":"0","lowerFundingRate":"0","isPreListing":false,"preListingInfo":null,"riskParameters":{"priceLimitRatioX":"0.05","priceLimitRatioY":"0.1"},"displayName":"BTCUSD1225","symbolType":"","forbidUplWithdrawal":false,"symbolId":2708,"fullName":"Bitcoin","marketRegion":"","underlyingTicker":"","tags":[]}],"nextPageCursor":""},"retExtInfo":{},"time":1791898186927}`,
		},
		{
			// Bybit's instrument with its launch moved to 91 days before its delivery; Bybit lists only 196 day contracts now
			name: "quarterly contract", exp: "https://www.bybit.com/trade/inverse/futures/BTCUSD_Q",
			reply: `{"retCode":0,"retMsg":"OK","result":{"category":"inverse","list":[{"symbol":"BTCUSDZ26","contractType":"InverseFutures","status":"Trading","baseCoin":"BTC","quoteCoin":"USD","launchTime":"1790323200000","deliveryTime":"1798185600000","deliveryFeeRate":"0.0005","priceScale":"2","leverageFilter":{"minLeverage":"1","maxLeverage":"100.00","leverageStep":"0.01"},"priceFilter":{"minPrice":"0.50","maxPrice":"9999999.00","tickSize":"0.50"},"lotSizeFilter":{"maxOrderQty":"5000000","minOrderQty":"1","qtyStep":"1","postOnlyMaxOrderQty":"5000000","maxMktOrderQty":"1000000","minNotionalValue":"5"},"unifiedMarginTrade":true,"fundingInterval":0,"settleCoin":"BTC","copyTrading":"none","upperFundingRate":"0","lowerFundingRate":"0","isPreListing":false,"preListingInfo":null,"riskParameters":{"priceLimitRatioX":"0.05","priceLimitRatioY":"0.1"},"displayName":"BTCUSD1225","symbolType":"","forbidUplWithdrawal":false,"symbolId":2708,"fullName":"Bitcoin","marketRegion":"","underlyingTicker":"","tags":[]}],"nextPageCursor":""},"retExtInfo":{},"time":1791898186927}`,
		},
		{
			name: "bi-quarterly contract", exp: "https://www.bybit.com/trade/inverse/futures/BTCUSD_BIQ",
			reply: `{"retCode":0,"retMsg":"OK","result":{"category":"inverse","list":[{"symbol":"BTCUSDZ26","contractType":"InverseFutures","status":"Trading","baseCoin":"BTC","quoteCoin":"USD","launchTime":"1781251200000","deliveryTime":"1798185600000","deliveryFeeRate":"0.0005","priceScale":"2","leverageFilter":{"minLeverage":"1","maxLeverage":"100.00","leverageStep":"0.01"},"priceFilter":{"minPrice":"0.50","maxPrice":"9999999.00","tickSize":"0.50"},"lotSizeFilter":{"maxOrderQty":"5000000","minOrderQty":"1","qtyStep":"1","postOnlyMaxOrderQty":"5000000","maxMktOrderQty":"1000000","minNotionalValue":"5"},"unifiedMarginTrade":true,"fundingInterval":0,"settleCoin":"BTC","copyTrading":"none","upperFundingRate":"0","lowerFundingRate":"0","isPreListing":false,"preListingInfo":null,"riskParameters":{"priceLimitRatioX":"0.05","priceLimitRatioY":"0.1"},"displayName":"BTCUSD1225","symbolType":"","forbidUplWithdrawal":false,"symbolId":2708,"fullName":"Bitcoin","marketRegion":"","underlyingTicker":"","tags":[]}],"nextPageCursor":""},"retExtInfo":{},"time":1791898186927}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			u, err := marketReplyInstance(t, map[string]string{datedInstrument: tc.reply}).GetCurrencyTradeURL(t.Context(), asset.CoinMarginedFutures, dated)
			require.ErrorIs(t, err, tc.err, "GetCurrencyTradeURL must return the expected error")
			assert.Equal(t, tc.exp, u, "GetCurrencyTradeURL should return the contract's trade page")
		})
	}

	if !mockTests {
		ex := tradablePairsTestInstance(t)
		for _, a := range ex.GetAssetTypes(false) {
			pairs, err := ex.GetEnabledPairs(a)
			require.NoErrorf(t, err, "GetEnabledPairs must not error for %s", a)
			require.NotEmptyf(t, pairs, "GetEnabledPairs must return pairs for %s", a)
			u, err := ex.GetCurrencyTradeURL(t.Context(), a, pairs[0])
			require.NoErrorf(t, err, "GetCurrencyTradeURL must not error for %s", a)
			assert.Truef(t, strings.HasPrefix(u, tradeBaseURL), "GetCurrencyTradeURL should return a Bybit URL for %s, not %q", a, u)
		}
		available, err := ex.GetAvailablePairs(asset.CoinMarginedFutures)
		require.NoError(t, err, "GetAvailablePairs must not error")
		i := slices.IndexFunc(available, func(p currency.Pair) bool { return !p.Quote.Equal(currency.USD) })
		require.NotEqual(t, -1, i, "Bybit must list a dated inverse contract")
		u, err := ex.GetCurrencyTradeURL(t.Context(), asset.CoinMarginedFutures, available[i])
		require.NoError(t, err, "GetCurrencyTradeURL must not error for a dated inverse contract")
		assert.Truef(t, strings.HasPrefix(u, tradeBaseURL+"trade/inverse/futures/"), "GetCurrencyTradeURL should return the dated inverse contract's page, not %q", u)
		return
	}

	// An inverse contract that has been delivered is no longer listed
	_, err = e.GetCurrencyTradeURL(t.Context(), asset.CoinMarginedFutures, currency.NewPairWithDelimiter("BTC", "USDM26", "_"))
	require.ErrorIs(t, err, currency.ErrCurrencyNotFound, "GetCurrencyTradeURL must report a dated inverse contract Bybit does not list")
	for _, tc := range []struct {
		asset asset.Item
		pair  currency.Pair
		exp   string
	}{
		{asset: asset.Spot, pair: spotTradablePair, exp: "https://www.bybit.com/en/trade/spot/BTC/USDT"},
		{asset: asset.CoinMarginedFutures, pair: inverseTradablePair, exp: "https://www.bybit.com/trade/inverse/ADAUSD"},
		// A dated inverse contract's page is named for its length: Q for a quarter, BIQ for longer
		{asset: asset.CoinMarginedFutures, pair: currency.NewPairWithDelimiter("BTC", "USDH26", "_"), exp: "https://www.bybit.com/trade/inverse/futures/BTCUSD_BIQ"},
		{asset: asset.USDTMarginedFutures, pair: usdtMarginedTradablePair, exp: "https://www.bybit.com/trade/usdt/10000LADYSUSDT"},
		{asset: asset.USDCMarginedFutures, pair: usdcMarginedTradablePair, exp: "https://www.bybit.com/trade/futures/usdc/ETH-PERP"},
		{asset: asset.Options, pair: optionsTradablePair, exp: "https://www.bybit.com/trade/option/usdc/BTC"},
	} {
		u, err := e.GetCurrencyTradeURL(t.Context(), tc.asset, tc.pair)
		require.NoErrorf(t, err, "GetCurrencyTradeURL must not error for %s %s", tc.asset, tc.pair)
		assert.Equalf(t, tc.exp, u, "GetCurrencyTradeURL should return the trade page of %s %s", tc.asset, tc.pair)
	}
}

// TestGenerateSubscriptions exercises generateSubscriptions
func TestGenerateSubscriptions(t *testing.T) {
	t.Parallel()

	intervalLabel := func(t *testing.T, i kline.Interval) string {
		t.Helper()

		switch i {
		case kline.OneHour:
			return "60"
		default:
			require.Failf(t, "unsupported interval", "Interval must be explicitly handled in this test, got %s", i)
			return ""
		}
	}

	expectedChannel := func(t *testing.T, channel string) string {
		t.Helper()

		switch channel {
		case subscription.TickerChannel:
			return chanPublicTicker
		case subscription.OrderbookChannel:
			return chanOrderbook
		case subscription.AllTradesChannel:
			return chanPublicTrade
		case subscription.CandlesChannel:
			return chanKline
		default:
			require.Failf(t, "unsupported channel", "Channel must be explicitly handled in this test, got %s", channel)
			return ""
		}
	}

	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")

	e.Websocket.SetCanUseAuthenticatedEndpoints(true)
	subs, err := e.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions must not error")
	require.NotEmpty(t, subs, "generateSubscriptions must return subscriptions")

	seenQualifiedChannels := make(map[string]struct{}, len(subs))
	for _, sub := range subs {
		assert.NotEmpty(t, sub.QualifiedChannel, "QualifiedChannel should not be empty")
		assert.Len(t, sub.Pairs, 1, "Pairs should contain a single symbol per generated subscription")
		_, found := seenQualifiedChannels[sub.QualifiedChannel]
		assert.Falsef(t, found, "QualifiedChannel should be unique, got duplicate %q", sub.QualifiedChannel)
		seenQualifiedChannels[sub.QualifiedChannel] = struct{}{}
	}

	expectedBaseSubscriptions := subscription.List{
		{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel},
		{Enabled: true, Asset: asset.Spot, Channel: subscription.OrderbookChannel, Levels: 50},
		{Enabled: true, Asset: asset.Spot, Channel: subscription.AllTradesChannel},
		{Enabled: true, Asset: asset.Spot, Channel: subscription.CandlesChannel, Interval: kline.OneHour},
	}

	exp := subscription.List{}
	for _, baseSub := range expectedBaseSubscriptions {
		for _, a := range e.GetAssetTypes(true) {
			if !e.IsAssetWebsocketSupported(a) {
				continue
			}
			if baseSub.Asset != asset.All && baseSub.Asset != a {
				continue
			}

			pairs, err := e.GetEnabledPairs(a)
			require.NoErrorf(t, err, "GetEnabledPairs must not error for asset %s", a)
			pairFmt, err := e.GetPairFormat(a, true)
			require.NoErrorf(t, err, "GetPairFormat must not error for asset %s", a)
			pairs = common.SortStrings(pairs.Format(pairFmt))

			s := baseSub.Clone()
			s.Asset = a

			for i, p := range pairs {
				sub := s.Clone()
				switch sub.Channel {
				case subscription.CandlesChannel:
					sub.QualifiedChannel = fmt.Sprintf("%s.%s.%s", expectedChannel(t, sub.Channel), intervalLabel(t, sub.Interval), p)
				case subscription.OrderbookChannel:
					sub.QualifiedChannel = fmt.Sprintf("%s.%d.%s", expectedChannel(t, sub.Channel), sub.Levels, p)
				default:
					sub.QualifiedChannel = expectedChannel(t, sub.Channel) + "." + p.String()
				}
				sub.Pairs = pairs[i : i+1]
				exp = append(exp, sub)
			}
		}
	}

	testsubs.EqualLists(t, exp, subs)
}

func TestAuthSubscribe(t *testing.T) {
	t.Parallel()

	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")
	require.NoError(t, e.authSubscribe(t.Context(), &FixtureConnection{}, subscription.List{}), "authSubscribe must not error without subscriptions")

	authsubs, err := e.generateAuthSubscriptions()
	require.NoError(t, err, "generateAuthSubscriptions must not error")
	require.Empty(t, authsubs, "generateAuthSubscriptions must not return subs")

	e.Websocket.SetCanUseAuthenticatedEndpoints(true)
	authsubs, err = e.generateAuthSubscriptions()
	require.NoError(t, err, "generateAuthSubscriptions must not error")
	require.NotEmpty(t, authsubs, "generateAuthSubscriptions must return subs")
	var positionSubscriptions subscription.List
	for _, sub := range authsubs {
		if sub.Channel == chanPositions {
			positionSubscriptions = append(positionSubscriptions, sub)
		}
	}
	require.Len(t, positionSubscriptions, 1, "position must have one all-in-one subscription")
	require.Empty(t, positionSubscriptions[0].Pairs, "all-in-one position subscription must not specify symbols")

	require.NoError(t, e.authSubscribe(t.Context(), &FixtureConnection{}, authsubs), "authSubscribe must not error")
	require.NoError(t, e.authUnsubscribe(t.Context(), &FixtureConnection{}, authsubs), "authUnsubscribe must not error")
}

// orderFixtures holds the details of the orders testdata/http.json records, as the shared exchange reports them
type orderFixtures struct {
	spotPartial, spotNew, spotTPSL, spotFilledByValue, spotCancelled                      order.Detail
	linearTakeProfit, linearDatedPartial, linearFilled, linearCancelled, linearInfoFilled order.Detail
	usdcNew, usdcFilled, inversePartial, inverseFilled, optionPartial, optionFilled       order.Detail
}

// newOrderFixtures returns the details of the orders testdata/http.json records. Linear and spot orders report their
// fees per coin in cumFeeDetail, which orders from before it lack; inverse and option orders report theirs in
// cumExecFee, in the coin they settle in
func newOrderFixtures(tb testing.TB) *orderFixtures {
	tb.Helper()
	ladys := currency.NewPair(currency.NewCode("10000LADYS"), currency.USDT)
	option := currency.NewPairWithDelimiter("BTC", "26NOV24-92000-C", "-")
	o := &orderFixtures{
		spotPartial: order.Detail{
			Exchange: e.Name, OrderID: "1321003749386327552", ClientOrderID: "spot-test-01", Pair: currency.NewPairWithDelimiter("BTC", "USDT", "_"), AssetType: asset.Spot,
			Type: order.Limit, Side: order.Buy, Status: order.PartiallyFilled, TimeInForce: order.GoodTillCancel, Price: 15600, Amount: 0.1,
			AverageExecutedPrice: 15600, ExecutedAmount: 0.04, RemainingAmount: 0.06, Cost: 624, CostAsset: currency.USDT,
			Date: time.UnixMilli(1684738540559), LastUpdated: time.UnixMilli(1684738540561),
		},
		spotNew: order.Detail{
			Exchange: e.Name, OrderID: "1321003749386327553", ClientOrderID: "spot-test-02", Pair: currency.NewPairWithDelimiter("ETH", "USDT", "_"), AssetType: asset.Spot,
			Type: order.Limit, Side: order.Sell, Status: order.New, TimeInForce: order.GoodTillCancel, Price: 1900, Amount: 0.5, RemainingAmount: 0.5,
			CostAsset: currency.USDT, Date: time.UnixMilli(1684738600000), LastUpdated: time.UnixMilli(1684738600000),
		},
		spotTPSL: order.Detail{
			Exchange: e.Name, OrderID: "1321003749386327560", ClientOrderID: "spot-test-04", Pair: currency.NewPairWithDelimiter("BTC", "USDT", "_"), AssetType: asset.Spot,
			Type: order.StopLimit, Side: order.Sell, Status: order.Pending, TimeInForce: order.GoodTillCancel, Price: 16500, Amount: 0.05, TriggerPrice: 16400,
			RemainingAmount: 0.05, CostAsset: currency.USDT, Date: time.UnixMilli(1684738700000), LastUpdated: time.UnixMilli(1684738700000),
		},
		// A spot market buy sized by value holds its quote amount in qty, and the base amount it bought in cumExecQty
		spotFilledByValue: order.Detail{
			Exchange: e.Name, OrderID: "1321003749386327570", ClientOrderID: "spot-test-05", Pair: currency.NewPairWithDelimiter("BTC", "USDT", "_"), AssetType: asset.Spot,
			Type: order.Market, Side: order.Buy, Status: order.Filled, TimeInForce: order.ImmediateOrCancel, Amount: 0.00096, QuoteAmount: 25,
			AverageExecutedPrice: 26040.57, ExecutedAmount: 0.00096, Cost: 24.9989472, CostAsset: currency.USDT,
			Date: time.UnixMilli(1693087635000), LastUpdated: time.UnixMilli(1693087635012),
		},
		spotCancelled: order.Detail{
			Exchange: e.Name, OrderID: "1321003749386327571", ClientOrderID: "spot-test-06", Pair: currency.NewPairWithDelimiter("BTC", "USDT", "_"), AssetType: asset.Spot,
			Type: order.Limit, Side: order.Sell, Status: order.Cancelled, TimeInForce: order.GoodTillCancel, Price: 27000, Amount: 0.01, CostAsset: currency.USDT,
			Date: time.UnixMilli(1693087700000), LastUpdated: time.UnixMilli(1693091300000),
		},
		linearTakeProfit: order.Detail{
			Exchange: e.Name, OrderID: "fd4300ae-7847-404e-b947-b46980a4d140", ClientOrderID: "test-000005", Pair: ladys, AssetType: asset.USDTMarginedFutures,
			Type: order.TakeProfit, Side: order.Sell, Status: order.Pending, TimeInForce: order.GoodTillCancel, ReduceOnly: true, Price: 0.0005, Amount: 100000,
			TriggerPrice: 0.00049, RemainingAmount: 100000, CostAsset: currency.USDT, SettlementCurrency: currency.USDT,
			Date: time.UnixMilli(1693086704500), LastUpdated: time.UnixMilli(1693086704500),
		},
		// A dated USDT margined contract keeps its expiry in the quote of its pair
		linearDatedPartial: order.Detail{
			Exchange: e.Name, OrderID: "5bf6aa3e-0c07-4a1f-a3a3-1cb5e3bf8f30", ClientOrderID: "test-000006", Pair: currency.NewPairWithDelimiter("BTC", "USDT-25DEC26", "_"),
			AssetType: asset.USDTMarginedFutures, Type: order.Limit, Side: order.Buy, Status: order.PartiallyFilled, TimeInForce: order.GoodTillCancel, Price: 84000,
			Amount: 0.01, AverageExecutedPrice: 84000, ExecutedAmount: 0.005, RemainingAmount: 0.005, Cost: 420, CostAsset: currency.USDT,
			SettlementCurrency: currency.USDT, Date: time.UnixMilli(1791884000000), LastUpdated: time.UnixMilli(1791884060000),
		},
		linearFilled: order.Detail{
			Exchange: e.Name, OrderID: "c1956690-b731-4191-97c0-94b00422231b", ClientOrderID: "linear-002", Pair: currency.NewPairWithDelimiter("BTC", "USDT", "_"),
			AssetType: asset.USDTMarginedFutures, Type: order.Limit, Side: order.Buy, Status: order.Filled, TimeInForce: order.GoodTillCancel, Price: 81000,
			Amount: 0.02, AverageExecutedPrice: 81000, ExecutedAmount: 0.02, Cost: 1620, CostAsset: currency.USDT, SettlementCurrency: currency.USDT,
			Date: time.UnixMilli(1788429600000), LastUpdated: time.UnixMilli(1788429601200),
		},
		linearCancelled: order.Detail{
			Exchange: e.Name, OrderID: "0b9f2a6e-34d1-4e8a-9f0c-6d2e1a7b5c43", ClientOrderID: "linear-003", Pair: ladys, AssetType: asset.USDTMarginedFutures,
			Type: order.Limit, Side: order.Buy, Status: order.Cancelled, TimeInForce: order.GoodTillCancel, Price: 0.0004, Amount: 200000,
			CostAsset: currency.USDT, SettlementCurrency: currency.USDT, Date: time.UnixMilli(1788912000000), LastUpdated: time.UnixMilli(1788915600000),
		},
		linearInfoFilled: order.Detail{
			Exchange: e.Name, OrderID: "e5bcd3f2-8a41-4c7d-b5e9-2f3a6c8d1b07", ClientOrderID: "linear-004", Pair: ladys, AssetType: asset.USDTMarginedFutures,
			Type: order.Market, Side: order.Buy, Status: order.Filled, TimeInForce: order.ImmediateOrCancel, Price: 0.000415, Amount: 237600,
			AverageExecutedPrice: 0.000415, ExecutedAmount: 237600, Cost: 98.604, CostAsset: currency.USDT, SettlementCurrency: currency.USDT,
			Date: time.UnixMilli(1693086704010), LastUpdated: time.UnixMilli(1693086704012),
		},
		usdcNew: order.Detail{
			Exchange: e.Name, OrderID: "9a3c6f8e-5d21-4b7a-8c3e-1f2d4a6b8c90", ClientOrderID: "usdc-001", Pair: currency.NewPairWithDelimiter("ETH", "PERP", "-"),
			AssetType: asset.USDCMarginedFutures, Type: order.Limit, Side: order.Buy, Status: order.New, TimeInForce: order.GoodTillCancel, Price: 2600,
			Amount: 0.5, RemainingAmount: 0.5, CostAsset: currency.USDC, SettlementCurrency: currency.USDC,
			Date: time.UnixMilli(1705017413000), LastUpdated: time.UnixMilli(1705017413000),
		},
		usdcFilled: order.Detail{
			Exchange: e.Name, OrderID: "3e7d9b1a-6c4f-4e2b-a8d5-7f9c1e3b5a24", ClientOrderID: "usdc-002", Pair: currency.NewPairWithDelimiter("ETH", "PERP", "-"),
			AssetType: asset.USDCMarginedFutures, Type: order.Limit, Side: order.Sell, Status: order.Filled, TimeInForce: order.GoodTillCancel, Price: 1585,
			Amount: 0.7, AverageExecutedPrice: 1585, ExecutedAmount: 0.7, Cost: 1109.5, CostAsset: currency.USDC, SettlementCurrency: currency.USDC,
			Date: time.UnixMilli(1697489879000), LastUpdated: time.UnixMilli(1697489879598),
		},
		inversePartial: order.Detail{
			Exchange: e.Name, OrderID: "8e1a2b3c-4d5e-4f60-8172-93a4b5c6d7e8", ClientOrderID: "inverse-001", Pair: currency.NewPairWithDelimiter("ADA", "USD", "_"),
			AssetType: asset.CoinMarginedFutures, Type: order.Limit, Side: order.Buy, Status: order.PartiallyFilled, TimeInForce: order.GoodTillCancel,
			Price: 0.5, Amount: 100, AverageExecutedPrice: 0.5, ExecutedAmount: 50, RemainingAmount: 50, Cost: 100, CostAsset: currency.ADA,
			SettlementCurrency: currency.ADA, Date: time.UnixMilli(1704941124000), LastUpdated: time.UnixMilli(1704941125000),
		},
		inverseFilled: order.Detail{
			Exchange: e.Name, OrderID: "2c4e6a8b-1d3f-4a5c-9e7b-0f1a2b3c4d5e", ClientOrderID: "inverse-002", Pair: currency.NewPairWithDelimiter("ADA", "USD", "_"),
			AssetType: asset.CoinMarginedFutures, Type: order.Market, Side: order.Buy, Status: order.Filled, TimeInForce: order.ImmediateOrCancel,
			Price: 0.2617, Amount: 234, AverageExecutedPrice: 0.2617, ExecutedAmount: 234, Cost: 894.1536, CostAsset: currency.ADA,
			SettlementCurrency: currency.ADA, Date: time.UnixMilli(1693086264000), LastUpdated: time.UnixMilli(1693086264683),
		},
		optionPartial: order.Detail{
			Exchange: e.Name, OrderID: "8d3a3cbd-14ac-4f3e-a72b-3a5b1a4d1f9a", ClientOrderID: "option-test-001", Pair: option, AssetType: asset.Options,
			Type: order.Limit, Side: order.Buy, Status: order.PartiallyFilled, TimeInForce: order.GoodTillCancel, Price: 3500, Amount: 0.5,
			AverageExecutedPrice: 3500, ExecutedAmount: 0.3, RemainingAmount: 0.2, Cost: 1050, CostAsset: currency.USDC, SettlementCurrency: currency.USDC,
			Date: time.UnixMilli(1731900000000), LastUpdated: time.UnixMilli(1731900005000),
		},
		optionFilled: order.Detail{
			Exchange: e.Name, OrderID: "7cb8afae-d5cc-5833-ad14-6d6dc08c3d32", ClientOrderID: "option-test-002", Pair: option, AssetType: asset.Options,
			Type: order.Limit, Side: order.Buy, Status: order.Filled, TimeInForce: order.ImmediateOrCancel, Price: 35, Amount: 0.02,
			AverageExecutedPrice: 35, ExecutedAmount: 0.02, Cost: 0.7, CostAsset: currency.USDC, SettlementCurrency: currency.USDC,
			Date: time.UnixMilli(1692943393000), LastUpdated: time.UnixMilli(1692943393092),
		},
	}
	o.spotPartial.SetFees(mustFees(tb, order.FeeAmount{Currency: currency.BTC, Amount: 0.00004}, order.FeeAmount{Currency: currency.MNT, Amount: 0.00242968}))
	o.spotNew.SetFees(mustFees(tb))
	o.spotFilledByValue.SetFees(mustFees(tb, order.FeeAmount{Currency: currency.BTC, Amount: 0.00000096}))
	o.spotCancelled.SetFees(mustFees(tb))
	o.linearTakeProfit.SetFees(mustFees(tb))
	o.linearDatedPartial.SetFees(mustFees(tb, order.FeeAmount{Currency: currency.USDT, Amount: 0.231}))
	o.linearFilled.SetFees(mustFees(tb, order.FeeAmount{Currency: currency.USDT, Amount: 0.324}))
	o.linearCancelled.SetFees(mustFees(tb))
	o.linearInfoFilled.SetFees(mustFees(tb, order.FeeAmount{Currency: currency.USDT, Amount: 0.0542322}))
	o.usdcNew.SetFees(mustFees(tb))
	o.usdcFilled.SetFees(mustFees(tb, order.FeeAmount{Currency: currency.USDC, Amount: 0.61022}))
	o.inversePartial.SetFees(mustFees(tb, order.FeeAmount{Currency: currency.ADA, Amount: 0.055}))
	o.inverseFilled.SetFees(mustFees(tb, order.FeeAmount{Currency: currency.ADA, Amount: 0.49177448}))
	o.optionPartial.SetFees(mustFees(tb, order.FeeAmount{Currency: currency.USDC, Amount: 0.6}))
	o.optionFilled.SetFees(mustFees(tb, order.FeeAmount{Currency: currency.USDC, Amount: 0.0003}))
	return o
}

// restReplyTestInstance returns an exchange of its own whose REST requests a local server answers, mock and live alike:
// a request to a path in replies gets that path's reply, and any other request Bybit's refusal for want of permission
func restReplyTestInstance(t *testing.T, replies map[string]string) *Exchange {
	t.Helper()
	ex := newTransportTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		reply, ok := replies[r.URL.Path]
		if !ok {
			reply = `{"retCode":10005,"retMsg":"Permission denied, please check your API key permissions.","result":{},"retExtInfo":{},"time":1760000000000}`
		}
		_, err := fmt.Fprint(w, reply)
		assert.NoError(t, err, "writing the reply should not error")
	})
	// The replies are local, so throttling protects nothing
	require.NoError(t, ex.DisableRateLimiter(), "DisableRateLimiter must not error")
	return ex
}

// noRequestFormatTestInstance returns an exchange of its own that holds asset a's pairs without the request format that
// Bybit symbols are written in
func noRequestFormatTestInstance(t *testing.T, a asset.Item) *Exchange {
	t.Helper()
	ex := testInstance(t)
	ps, err := ex.CurrencyPairs.Get(a)
	require.NoErrorf(t, err, "Get must not error for %s", a)
	ps.RequestFormat = nil
	require.NoErrorf(t, ex.CurrencyPairs.Store(a, ps), "Store must not error for %s", a)
	return ex
}

// storeTestInstance returns an exchange of its own, named for the test, for tests that read or write the global ticker,
// orderbook or order limit stores. Live, it holds the tradable pairs ensureTradablePairs loads
func storeTestInstance(t *testing.T) *Exchange {
	t.Helper()
	ensureTradablePairs(t)
	ex := testInstance(t)
	if !mockTests {
		testexch.UpdatePairsOnce(t, ex)
	}
	ex.Name = t.Name()
	return ex
}

// tradablePairsTestInstance returns an exchange of its own, named for the test, holding the tradable pairs Bybit lists:
// those the fixtures record, or the live ones
func tradablePairsTestInstance(t *testing.T) *Exchange {
	t.Helper()
	ex := testInstance(t)
	testexch.UpdatePairsOnce(t, ex)
	ex.Name = t.Name()
	return ex
}

func newInstrumentInfoTestExchange(t *testing.T, name, category string, responses map[string][]InstrumentInfo, queries *[]url.Values) *Exchange {
	t.Helper()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = name

	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method, "Request method should be GET")
		if r.URL.Path == "/v5/market/option-base-coins" {
			assert.Equal(t, "0", r.URL.Query().Get("underlyingType"), "Option base coins should be requested for crypto underlyings")
			var coins []string
			for baseCoin := range responses {
				if baseCoin != "" {
					coins = append(coins, `{"baseCoin":"`+baseCoin+`","quoteCoin":"USDT","settleCoin":"USDT","hasSymbol":1,"underlyingType":0}`)
				}
			}
			_, err := w.Write([]byte(`{"retCode":0,"retMsg":"OK","result":{"list":[` + strings.Join(coins, ",") + `]},"retExtInfo":{},"time":1760000000000}`))
			assert.NoError(t, err, "Writing the option base coins response should not error")
			return
		}
		assert.Equal(t, "/v5/market/instruments-info", r.URL.Path, "Request path should be the instruments info endpoint")

		query := r.URL.Query()
		*queries = append(*queries, query)

		payload := map[string]any{
			"retCode": 0,
			"retMsg":  "OK",
			"result": map[string]any{
				"category":       category,
				"list":           instrumentInfoResponseList(responses[query.Get("baseCoin")]),
				"nextPageCursor": "",
			},
			"retExtInfo": map[string]any{},
			"time":       time.Now().UnixMilli(),
		}

		w.WriteHeader(http.StatusOK)
		err := json.NewEncoder(w).Encode(payload)
		assert.NoError(t, err, "Encoding the instruments info response should not error")
	}))

	require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
	require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL), "SetRunningURL must not error")

	return ex
}

func newOptionInstrumentInfo(symbol, status string) InstrumentInfo {
	info := InstrumentInfo{
		Symbol:      symbol,
		BaseCoin:    currency.BTC,
		Status:      status,
		OptionsType: "Call",
	}
	info.PriceFilter.MinPrice = 1
	info.PriceFilter.MaxPrice = 1_000_000
	info.PriceFilter.TickSize = 0.1
	info.LotSizeFilter.MinOrderQuantity = 0.1
	info.LotSizeFilter.MaxOrderQuantity = 10
	info.LotSizeFilter.QuantityStep = 0.1
	info.LotSizeFilter.BasePrecision = 0.1
	info.LotSizeFilter.QuotePrecision = 0.1
	info.LotSizeFilter.MinOrderAmount = 1
	info.LotSizeFilter.MaxOrderAmount = 1_000_000
	info.LotSizeFilter.MinNotionalValue = 1
	return info
}

func instrumentInfoResponseList(infos []InstrumentInfo) []map[string]any {
	response := make([]map[string]any, 0, len(infos))
	for i := range infos {
		response = append(response, map[string]any{
			"symbol":      infos[i].Symbol,
			"optionsType": infos[i].OptionsType,
			"status":      infos[i].Status,
			"baseCoin":    infos[i].BaseCoin,
			"priceFilter": map[string]any{
				"minPrice": infos[i].PriceFilter.MinPrice.Float64(),
				"maxPrice": infos[i].PriceFilter.MaxPrice.Float64(),
				"tickSize": infos[i].PriceFilter.TickSize.Float64(),
			},
			"lotSizeFilter": map[string]any{
				"minOrderQty":      infos[i].LotSizeFilter.MinOrderQuantity.Float64(),
				"maxOrderQty":      infos[i].LotSizeFilter.MaxOrderQuantity.Float64(),
				"qtyStep":          infos[i].LotSizeFilter.QuantityStep.Float64(),
				"basePrecision":    infos[i].LotSizeFilter.BasePrecision.Float64(),
				"quotePrecision":   infos[i].LotSizeFilter.QuotePrecision.Float64(),
				"minOrderAmt":      infos[i].LotSizeFilter.MinOrderAmount.Float64(),
				"maxOrderAmt":      infos[i].LotSizeFilter.MaxOrderAmount.Float64(),
				"minNotionalValue": infos[i].LotSizeFilter.MinNotionalValue.Float64(),
			},
		})
	}
	return response
}

func assertOptionsInstrumentQueries(t *testing.T, queries []url.Values, caller string, baseCoins ...string) {
	t.Helper()

	require.Len(t, queries, len(baseCoins), caller+" must query every option base coin")

	requestedBaseCoins := make([]string, 0, len(queries))
	for _, query := range queries {
		requestedBaseCoins = append(requestedBaseCoins, query.Get("baseCoin"))
		assert.Equal(t, cOption, query.Get("category"), caller+" should request the option category")
		assert.Equal(t, tradingStatus, query.Get("status"), caller+" should request trading option instruments")
		assert.Equal(t, "1000", query.Get("limit"), caller+" should request the expected page size")
	}

	assert.ElementsMatch(t, baseCoins, requestedBaseCoins, caller+" should query every option base coin exactly once")
}

func TestQuoteSized(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		asset      asset.Item
		orderType  string
		side       string
		marketUnit string
		exp        bool
	}{
		// A spot market buy is sized by value unless its unit is the base coin, and a sell by quantity unless it is the
		// quote coin
		{asset: asset.Spot, orderType: "Market", side: sideBuy, exp: true},
		{asset: asset.Spot, orderType: "Market", side: sideBuy, marketUnit: "quoteCoin", exp: true},
		{asset: asset.Spot, orderType: "Market", side: sideBuy, marketUnit: "baseCoin"},
		{asset: asset.Spot, orderType: "Market", side: sideSell},
		{asset: asset.Spot, orderType: "Market", side: sideSell, marketUnit: "quoteCoin", exp: true},
		{asset: asset.Spot, orderType: "Market", side: sideSell, marketUnit: "baseCoin"},
		// Limit orders and derivatives are sized by quantity, whatever market unit they report
		{asset: asset.Spot, orderType: "Limit", side: sideBuy},
		{asset: asset.Spot, orderType: "Limit", side: sideSell},
		{asset: asset.Spot, orderType: "Limit", side: sideBuy, marketUnit: "quoteCoin"},
		{asset: asset.USDTMarginedFutures, orderType: "Market", side: sideSell, marketUnit: "quoteCoin"},
		{asset: asset.USDTMarginedFutures, orderType: "Market", side: sideBuy},
		{asset: asset.USDCMarginedFutures, orderType: "Market", side: sideBuy},
		{asset: asset.CoinMarginedFutures, orderType: "Market", side: sideBuy},
		{asset: asset.Options, orderType: "Market", side: sideBuy},
	} {
		assert.Equalf(t, tc.exp, quoteSized(tc.asset, tc.orderType, tc.side, tc.marketUnit), "quoteSized should report whether a %s %s %s order with market unit %q is sized by value", tc.asset, tc.orderType, tc.side, tc.marketUnit)
	}
}

func TestConditionalOrderType(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		orderType     order.Type
		stopOrderType string
		exp           order.Type
	}{
		{orderType: order.Limit, exp: order.Limit},
		{orderType: order.Market, stopOrderType: "UNKNOWN", exp: order.Market},
		{orderType: order.Market, stopOrderType: "Stop", exp: order.StopMarket},
		{orderType: order.Limit, stopOrderType: "Stop", exp: order.StopLimit},
		{orderType: order.Limit, stopOrderType: "tpslOrder", exp: order.StopLimit},
		{orderType: order.Market, stopOrderType: "tpslOrder", exp: order.StopMarket},
		{orderType: order.UnknownType, stopOrderType: "Stop", exp: order.Stop},
		{orderType: order.Market, stopOrderType: "StopLoss", exp: order.StopMarket},
		{orderType: order.Limit, stopOrderType: "PartialStopLoss", exp: order.StopLimit},
		{orderType: order.Market, stopOrderType: "TakeProfit", exp: order.TakeProfitMarket},
		{orderType: order.Limit, stopOrderType: "PartialTakeProfit", exp: order.TakeProfit},
		{orderType: order.Market, stopOrderType: "PartialTakeProfit", exp: order.TakeProfitMarket},
		{orderType: order.Market, stopOrderType: "TrailingStop", exp: order.TrailingStop},
		{orderType: order.Limit, stopOrderType: "OcoOrder", exp: order.OCO},
		{orderType: order.Market, stopOrderType: "OcoOrder", exp: order.OCO},
		{orderType: order.Market, stopOrderType: "BidirectionalTpslOrder", exp: order.OCO},
		{orderType: order.Limit, stopOrderType: "BidirectionalTpslOrder", exp: order.OCO},
		{orderType: order.Market, stopOrderType: "MmRateClose", exp: order.StopMarket},
	} {
		assert.Equalf(t, tc.exp, conditionalOrderType(tc.orderType, tc.stopOrderType), "conditionalOrderType should map a %s %q order", tc.orderType, tc.stopOrderType)
	}
}

func TestSubmittedOrderType(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		orderType    order.Type
		triggerPrice float64
		exp          order.Type
	}{
		{orderType: order.Limit, exp: order.Limit},
		{orderType: order.Market, exp: order.Market},
		// A limit or market order with a trigger price is placed as a conditional order
		{orderType: order.Limit, triggerPrice: 90000, exp: order.StopLimit},
		{orderType: order.Market, triggerPrice: 90000, exp: order.StopMarket},
		{orderType: order.StopLimit, triggerPrice: 90000, exp: order.StopLimit},
		{orderType: order.StopMarket, triggerPrice: 90000, exp: order.StopMarket},
		{orderType: order.Stop, triggerPrice: 90000, exp: order.StopMarket},
	} {
		got := submittedOrderType(&order.Submit{Type: tc.orderType, TriggerPrice: tc.triggerPrice})
		assert.Equalf(t, tc.exp, got, "submittedOrderType should return the type a %s order with trigger price %v reads back as", tc.orderType, tc.triggerPrice)
	}
}
