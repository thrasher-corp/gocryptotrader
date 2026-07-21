package coinbase

import (
	"context"
	"errors"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	gctjson "github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	testsubs "github.com/thrasher-corp/gocryptotrader/internal/testing/subscriptions"
)

type subscriptionTestConnection struct {
	websocket.Connection
	sendErr error
	sent    []any
}

func (c *subscriptionTestConnection) SendJSONMessage(_ context.Context, _ request.EndpointLimit, payload any) error {
	c.sent = append(c.sent, payload)
	return c.sendErr
}

func TestWsConnect(t *testing.T) {
	t.Parallel()
	exch := &Exchange{}
	exch.Websocket = sharedtestvalues.NewTestWebsocket()
	err := exch.Websocket.Connect(t.Context())
	assert.ErrorIs(t, err, websocket.ErrWebsocketNotEnabled)
	err = exchangeBaseHelper(exch)
	require.NoError(t, err)
	err = exch.Websocket.Enable(t.Context())
	assert.NoError(t, err)
}

func TestWsHandleData(t *testing.T) {
	t.Parallel()
	assertUnmarshalTypeError := func(t *testing.T, err error) {
		t.Helper()
		var unmarshalTypeErr *gctjson.UnmarshalTypeError
		assert.True(t, errors.As(err, &unmarshalTypeErr) || strings.Contains(err.Error(), "mismatched type with value"), errJSONUnmarshalUnexpected)
	}

	t.Run("nil message", func(t *testing.T) {
		t.Parallel()
		err := e.wsHandleData(t.Context(), testexch.GetMockConn(t, e, ""), nil)
		var syntaxErr *gctjson.SyntaxError
		assert.True(t, errors.As(err, &syntaxErr) || strings.Contains(err.Error(), "Syntax error no sources available, the input json is empty"), errJSONUnmarshalUnexpected)
	})

	t.Run("error type message", func(t *testing.T) {
		t.Parallel()
		mockJSON := []byte(`{"type": "error"}`)
		err := e.wsHandleData(t.Context(), testexch.GetMockConn(t, e, ""), mockJSON)
		assert.EqualError(t, err, "error", "wsHandleData should return the websocket error type")
	})

	t.Run("subscriptions channel", func(t *testing.T) {
		t.Parallel()

		mockJSON := []byte(`{"sequence_num": 0, "channel": "subscriptions"}`)
		err := e.wsHandleData(t.Context(), testexch.GetMockConn(t, e, ""), mockJSON)
		assert.NoError(t, err)
	})

	t.Run("heartbeats channel", func(t *testing.T) {
		t.Parallel()
		mockJSON := []byte(`{"sequence_num": 0, "channel": "heartbeats"}`)
		err := e.wsHandleData(t.Context(), testexch.GetMockConn(t, e, ""), mockJSON)
		assert.NoError(t, err)
	})

	t.Run("status channel success", func(t *testing.T) {
		t.Parallel()
		mockJSON := []byte(`{"sequence_num": 0, "channel": "status", "events": [{"type": "status", "products": []}]}`)
		err := e.wsHandleData(t.Context(), testexch.GetMockConn(t, e, ""), mockJSON)
		assert.NoError(t, err)
	})

	t.Run("status events type unmarshal", func(t *testing.T) {
		t.Parallel()
		mockJSON := []byte(`{"sequence_num": 0, "channel": "status", "events": [{"type": 1234}]}`)
		err := e.wsHandleData(t.Context(), testexch.GetMockConn(t, e, ""), mockJSON)
		assertUnmarshalTypeError(t, err)
	})

	t.Run("ticker tickers unmarshal", func(t *testing.T) {
		t.Parallel()
		mockJSON := []byte(`{"sequence_num": 0, "channel": "ticker", "events": [{"type": "moo", "tickers": false}]}`)
		err := e.wsHandleData(t.Context(), testexch.GetMockConn(t, e, ""), mockJSON)
		assertUnmarshalTypeError(t, err)
	})

	t.Run("candles events type unmarshal", func(t *testing.T) {
		t.Parallel()
		mockJSON := []byte(`{"sequence_num": 0, "channel": "candles", "events": [{"type": false}]}`)
		err := e.wsHandleData(t.Context(), testexch.GetMockConn(t, e, ""), mockJSON)
		assertUnmarshalTypeError(t, err)
	})

	t.Run("market_trades events type unmarshal", func(t *testing.T) {
		t.Parallel()
		mockJSON := []byte(`{"sequence_num": 0, "channel": "market_trades", "events": [{"type": false}]}`)
		err := e.wsHandleData(t.Context(), testexch.GetMockConn(t, e, ""), mockJSON)
		assertUnmarshalTypeError(t, err)
	})

	t.Run("l2_data updates unmarshal", func(t *testing.T) {
		t.Parallel()
		mockJSON := []byte(`{"sequence_num": 0, "channel": "l2_data", "events": [{"type": false, "updates": [{"price_level": "1.1"}]}]}`)
		err := e.wsHandleData(t.Context(), testexch.GetMockConn(t, e, ""), mockJSON)
		assertUnmarshalTypeError(t, err)
	})

	t.Run("user events type unmarshal", func(t *testing.T) {
		t.Parallel()
		mockJSON := []byte(`{"sequence_num": 0, "channel": "user", "events": [{"type": false}]}`)
		err := e.wsHandleData(t.Context(), testexch.GetMockConn(t, e, ""), mockJSON)
		assertUnmarshalTypeError(t, err)
	})

	t.Run("unknown channel", func(t *testing.T) {
		t.Parallel()
		mockJSON := []byte(`{"sequence_num": 0, "channel": "fakechan", "events": [{"type": ""}]}`)
		err := e.wsHandleData(t.Context(), testexch.GetMockConn(t, e, ""), mockJSON)
		assert.ErrorIs(t, err, errChannelNameUnknown)
	})

	t.Run("sequence validation before payload error", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex))
		conn := testexch.GetMockConn(t, ex, "ws://coinbase-wshandledata-seq")
		assert.NoError(t, ex.wsHandleData(t.Context(), conn, []byte(`{"sequence_num": 1, "channel": "subscriptions"}`)))
		err := ex.wsHandleData(t.Context(), conn, []byte(`{"sequence_num": 3, "channel": "subscriptions", "type": "error"}`))
		assert.ErrorIs(t, err, errOutOfSequence)
	})

	t.Run("ticker with alias loaded", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex))
		p, err := ex.FormatExchangeCurrency(currency.NewBTCUSD(), asset.Spot)
		require.NoError(t, err)
		ex.pairAliases.Load(map[currency.Pair]currency.Pairs{p: {p}})
		mockJSON := []byte(`{"sequence_num": 0, "channel": "ticker", "events": [{"type": "moo", "tickers": [{"product_id": "BTC-USD", "price": "1.1"}]}]}`)
		err = ex.wsHandleData(t.Context(), testexch.GetMockConn(t, ex, ""), mockJSON)
		assert.NoError(t, err)
	})
}

func TestCheckWSSequence(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	connA := testexch.GetMockConn(t, ex, "ws://coinbase-seq-a")
	connB := testexch.GetMockConn(t, ex, "ws://coinbase-seq-b")

	assert.NoError(t, ex.checkWSSequence(connA, 7), "checkWSSequence should accept an initial sequence")
	assert.NoError(t, ex.checkWSSequence(connA, 8), "checkWSSequence should accept an in-order sequence")
	assert.ErrorIs(t, ex.checkWSSequence(connA, 10), errOutOfSequence, "checkWSSequence should reject an out-of-order sequence")
	assert.NoError(t, ex.checkWSSequence(connA, 11), "checkWSSequence should accept the resynchronised sequence")
	assert.NoError(t, ex.checkWSSequence(connB, 3), "checkWSSequence should maintain independent connection state")
}

func TestProcessSnapshot(t *testing.T) {
	t.Parallel()

	t.Run("invalid side", func(t *testing.T) {
		t.Parallel()
		req := WebsocketOrderbookDataHolder{Changes: []WebsocketOrderbookData{{Side: "fakeside", PriceLevel: 1.1, NewQuantity: 2.2}}, ProductID: currency.NewBTCUSD()}
		err := e.ProcessSnapshot(&req, time.Time{})
		assert.ErrorIs(t, err, order.ErrSideIsInvalid)
	})

	t.Run("valid offer", func(t *testing.T) {
		t.Parallel()
		req := WebsocketOrderbookDataHolder{Changes: []WebsocketOrderbookData{{Side: "offer", PriceLevel: 1.1, NewQuantity: 2.2}}, ProductID: currency.NewBTCUSD()}
		err := e.ProcessSnapshot(&req, time.Now())
		assert.NoError(t, err)
	})
}

func TestProcessUpdate(t *testing.T) {
	t.Parallel()

	t.Run("invalid side", func(t *testing.T) {
		t.Parallel()
		req := WebsocketOrderbookDataHolder{Changes: []WebsocketOrderbookData{{Side: "fakeside", PriceLevel: 1.1, NewQuantity: 2.2}}, ProductID: currency.NewBTCUSD()}
		err := e.ProcessUpdate(&req, time.Time{})
		assert.ErrorIs(t, err, order.ErrSideIsInvalid)
	})

	t.Run("valid offer", func(t *testing.T) {
		t.Parallel()
		req := WebsocketOrderbookDataHolder{Changes: []WebsocketOrderbookData{{Side: "offer", PriceLevel: 1.1, NewQuantity: 2.2}}, ProductID: currency.NewBTCUSD()}
		err := e.ProcessUpdate(&req, time.Now())
		assert.NoError(t, err)
	})
}

func TestGenerateSubscriptions(t *testing.T) {
	t.Parallel()

	t.Run("enabled subscriptions", func(t *testing.T) {
		t.Parallel()
		e := new(Exchange)
		require.NoError(t, testexch.Setup(e), "Setup must not error")
		e.Websocket.SetCanUseAuthenticatedEndpoints(true)
		p1, err := e.GetEnabledPairs(asset.Spot)
		require.NoError(t, err)
		p2, err := e.GetEnabledPairs(asset.Futures)
		require.NoError(t, err)
		exp := subscription.List{}
		for _, baseSub := range defaultSubscriptions.Enabled() {
			s := baseSub.Clone()
			s.QualifiedChannel = subscriptionNames[s.Channel]
			switch s.Asset {
			case asset.Spot:
				s.Pairs = p1
			case asset.Futures:
				s.Pairs = p2
			case asset.All:
				s2 := s.Clone()
				s2.Asset = asset.Futures
				s2.Pairs = p2
				exp = append(exp, s2)
				s.Asset = asset.Spot
				s.Pairs = p1
			}
			exp = append(exp, s)
		}
		subs, err := e.generateSubscriptions()
		require.NoError(t, err)
		testsubs.EqualLists(t, exp, subs)
	})

	t.Run("unsupported channel", func(t *testing.T) {
		t.Parallel()
		e := new(Exchange)
		require.NoError(t, testexch.Setup(e), "Setup must not error")
		_, err := subscription.List{{Channel: "wibble"}}.ExpandTemplates(e)
		assert.ErrorIs(t, err, subscription.ErrNotSupported)
	})
}

func TestSubscribeForConnection(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	conn := &subscriptionTestConnection{Connection: testexch.GetMockConn(t, ex, "wss://coinbase-subscribe.test")}
	sub := &subscription.Subscription{
		Channel:          subscription.TickerChannel,
		Asset:            asset.Spot,
		Pairs:            currency.Pairs{currency.NewBTCUSD()},
		QualifiedChannel: "ticker",
	}

	require.NoError(t, ex.subscribeForConnection(t.Context(), conn, subscription.List{sub}), "subscribeForConnection must not error")
	assert.Len(t, conn.sent, 1, "subscribeForConnection should send one request")
	assert.Same(t, sub, ex.Websocket.GetSubscription(sub), "subscribeForConnection should store the subscription")
	assert.Equal(t, subscription.SubscribedState, sub.State(), "subscribeForConnection should mark the subscription subscribed")
}

func TestUnsubscribeForConnection(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	conn := &subscriptionTestConnection{Connection: testexch.GetMockConn(t, ex, "wss://coinbase-unsubscribe.test")}
	sub := &subscription.Subscription{
		Channel:          subscription.TickerChannel,
		Asset:            asset.Spot,
		Pairs:            currency.Pairs{currency.NewBTCUSD()},
		QualifiedChannel: "ticker",
	}
	require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(conn, sub), "AddSuccessfulSubscriptions must not error")

	require.NoError(t, ex.unsubscribeForConnection(t.Context(), conn, subscription.List{sub}), "unsubscribeForConnection must not error")
	assert.Len(t, conn.sent, 1, "unsubscribeForConnection should send one request")
	assert.Nil(t, ex.Websocket.GetSubscription(sub), "unsubscribeForConnection should remove the subscription")
	assert.Equal(t, subscription.UnsubscribedState, sub.State(), "unsubscribeForConnection should mark the subscription unsubscribed")
}

func TestManageSubs(t *testing.T) {
	t.Parallel()

	t.Run("send failure", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		errSend := errors.New("send failure")
		conn := &subscriptionTestConnection{Connection: testexch.GetMockConn(t, ex, "wss://coinbase-manage.test"), sendErr: errSend}
		sub := &subscription.Subscription{QualifiedChannel: "ticker"}

		err := ex.manageSubs(t.Context(), conn, "subscribe", subscription.List{sub})
		assert.ErrorIs(t, err, errSend, "manageSubs should return the send failure")
		assert.Nil(t, ex.Websocket.GetSubscription(sub), "manageSubs should not store the subscription after a send failure")
	})
}

func TestCheckSubscriptions(t *testing.T) {
	t.Parallel()
	e := &Exchange{
		Base: exchange.Base{
			Config: &config.Exchange{
				Features: &config.FeaturesConfig{
					Subscriptions: subscription.List{
						{Enabled: true, Channel: "matches"},
					},
				},
			},
			Features: exchange.Features{},
		},
	}
	e.checkSubscriptions()
	testsubs.EqualLists(t, defaultSubscriptions.Enabled(), e.Features.Subscriptions)
	testsubs.EqualLists(t, defaultSubscriptions, e.Config.Features.Subscriptions)
}

func TestGetSubscriptionTemplate(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	tpl, err := ex.GetSubscriptionTemplate(nil)
	require.NoError(t, err)
	require.NotNil(t, tpl)
	_, err = template.Must(tpl, nil).Parse("{{ channelName . }}")
	assert.NoError(t, err)
}

func TestGetWSJWT(t *testing.T) {
	t.Parallel()

	t.Run("cached token", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		ex.jwt.token = "cached"
		ex.jwt.expiresAt = time.Now().Add(time.Hour)
		tok, err := ex.GetWSJWT(t.Context())
		require.NoError(t, err)
		assert.Equal(t, "cached", tok)
	})

	t.Run("expired token refresh error", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		ex.jwt.expiresAt = time.Now().Add(-time.Second)
		_, err := ex.GetWSJWT(t.Context())
		assert.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty)
	})
}

func TestProcessBidAskArray(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		data       *WebsocketOrderbookDataHolder
		snapshot   bool
		bidsLength int
		asksLength int
		err        error
	}{
		{
			name:       "snapshot bid and offer",
			data:       &WebsocketOrderbookDataHolder{Changes: []WebsocketOrderbookData{{Side: "bid", PriceLevel: 1.1, NewQuantity: 2.2}, {Side: "offer", PriceLevel: 1.2, NewQuantity: 3.3}}},
			snapshot:   true,
			bidsLength: 1,
			asksLength: 1,
		},
		{
			name:       "update bid only",
			data:       &WebsocketOrderbookDataHolder{Changes: []WebsocketOrderbookData{{Side: "bid", PriceLevel: 1.1, NewQuantity: 2.2}}},
			bidsLength: 1,
		},
		{
			name: "invalid side",
			data: &WebsocketOrderbookDataHolder{Changes: []WebsocketOrderbookData{{Side: "wat", PriceLevel: 1.1, NewQuantity: 2.2}}},
			err:  order.ErrSideIsInvalid,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			bids, asks, err := processBidAskArray(tc.data, tc.snapshot)
			if tc.err != nil {
				assert.ErrorIs(t, err, tc.err)
				return
			}
			require.NoError(t, err)
			assert.Len(t, bids, tc.bidsLength)
			assert.Len(t, asks, tc.asksLength)
		})
	}
}

func receiveDataHandlerPayload(t *testing.T, ex *Exchange, timeout time.Duration) any {
	t.Helper()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case payload := <-ex.Websocket.DataHandler.C:
		return payload.Data
	case <-timer.C:
		t.Fatal("timed out waiting for websocket data handler payload")
		return nil
	}
}

func TestWsProcessCandle(t *testing.T) {
	t.Parallel()

	t.Run("update", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		resp := &StandardWebsocketResponse{
			Timestamp: time.Unix(1704067200, 0),
			Events: []byte(`[{
				"type":"update",
				"candles":[
				{
					"start":"1704067200",
					"low":"1",
					"high":"2",
					"open":"1.25",
					"close":"1.75",
					"volume":"3.5",
					"product_id":"BTC-USD"
				},
				{
					"start":"1704067500",
					"low":"1.5",
					"high":"2.5",
					"open":"1.75",
					"close":"2.25",
					"volume":"1.1",
					"product_id":"BTC-USD"
				}
			]
			}]`),
		}
		require.NoError(t, ex.wsProcessCandle(t.Context(), resp), "wsProcessCandle must not error")

		data := receiveDataHandlerPayload(t, ex, time.Second)
		candles, ok := data.([]kline.Item)
		require.True(t, ok, "payload must be kline items")
		require.Len(t, candles, 2, "candles must include both updates")
		assert.Equal(t, currency.NewPairWithDelimiter("BTC", "USD", "-"), candles[0].Pair, "first candle should use exchange pair")
		assert.Equal(t, asset.Spot, candles[0].Asset, "first candle should use spot asset")
		assert.Len(t, candles[0].Candles, 1, "first candle should contain one item")
		assert.Len(t, candles[1].Candles, 1, "second candle should contain one item")
		assert.Equal(t, kline.FiveMin, candles[0].Interval, "first candle should use five minute interval")
		assert.Equal(t, kline.FiveMin, candles[1].Interval, "second candle should use five minute interval")
	})

	t.Run("invalid event", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		resp := &StandardWebsocketResponse{Events: []byte(`[{"type":false}]`)}
		assert.ErrorIs(t, ex.wsProcessCandle(t.Context(), resp), errCandleDataUnmarshal, "wsProcessCandle should return candle unmarshal error")
	})
}

func TestWsProcessMarketTrades(t *testing.T) {
	t.Parallel()

	t.Run("update", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		resp := &StandardWebsocketResponse{
			Events: []byte(`[{
				"type":"update",
				"trades":[{
					"trade_id":"123",
					"product_id":"BTC-USD",
					"price":"101.2",
					"size":"0.5",
					"side":"BUY",
					"time":"2024-01-01T00:00:00Z"
				}]
			}]`),
		}
		require.NoError(t, ex.wsProcessMarketTrades(t.Context(), resp), "wsProcessMarketTrades must not error")

		data := receiveDataHandlerPayload(t, ex, time.Second)
		trades, ok := data.([]trade.Data)
		require.True(t, ok, "payload must be trade data")
		require.Len(t, trades, 1, "payload must include one trade")
		assert.Equal(t, currency.NewPairWithDelimiter("BTC", "USD", "-"), trades[0].CurrencyPair, "trade should use exchange pair")
		assert.Equal(t, order.Buy, trades[0].Side, "trade should use buy side")
	})

	t.Run("invalid event", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		resp := &StandardWebsocketResponse{Events: []byte(`[{"type":false}]`)}
		assert.ErrorIs(t, ex.wsProcessMarketTrades(t.Context(), resp), errMarketTradeDataUnmarshal, "wsProcessMarketTrades should return market trade unmarshal error")
	})
}

func TestWsProcessL2(t *testing.T) {
	t.Parallel()

	t.Run("snapshot update", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		exchangePair := currency.NewPairWithDelimiter("BTC", "USD", "-")
		aliasPair := currency.NewBTCUSD()
		require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Spot, currency.Pairs{aliasPair}, true), "StorePairs must not error")
		ex.pairAliases.Load(map[currency.Pair]currency.Pairs{exchangePair: {aliasPair}})

		resp := &StandardWebsocketResponse{
			Timestamp: time.Now(),
			Events: []byte(`[{
				"type":"snapshot",
				"product_id":"BTC-USD",
				"updates":[
					{"side":"bid","price_level":"1.1","new_quantity":"2.2"},
					{"side":"offer","price_level":"1.2","new_quantity":"2.3"}
				]
			},{
				"type":"update",
				"product_id":"BTC-USD",
				"updates":[
					{"side":"bid","price_level":"1.15","new_quantity":"1.9"}
				]
			}]`),
		}
		require.NoError(t, ex.wsProcessL2(resp), "wsProcessL2 must not error")
		_, err := ex.Websocket.Orderbook.GetOrderbook(aliasPair, asset.Spot)
		assert.NoError(t, err, "orderbook should be available")
	})

	t.Run("unknown type", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		resp := &StandardWebsocketResponse{Events: []byte(`[{"type":"wat","product_id":"BTC-USD","updates":[]}]`)}
		assert.ErrorIs(t, ex.wsProcessL2(resp), errUnknownL2DataType, "wsProcessL2 should return unknown type error")
	})
}

func TestWsProcessUser(t *testing.T) {
	t.Parallel()

	t.Run("snapshot", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		resp := &StandardWebsocketResponse{
			Events: []byte(`[{
				"type":"snapshot",
				"orders":[{
					"order_type":"LIMIT_ORDER_TYPE",
					"order_side":"BUY",
					"status":"OPEN",
					"avg_price":"100",
					"limit_price":"101",
					"client_order_id":"cid",
					"cumulative_quantity":"0.25",
					"leaves_quantity":"0.75",
					"order_id":"oid",
					"product_id":"BTC-USD",
					"product_type":"SPOT",
					"stop_price":"0",
					"time_in_force":"GOOD_UNTIL_CANCELLED",
					"total_fees":"0.1",
					"creation_time":"2024-01-01T00:00:00Z",
					"end_time":"2024-01-01T01:00:00Z",
					"post_only":true
				}],
				"positions":{
					"perpetual_futures_positions":[{
						"product_id":"BTC-USD",
						"position_side":"LONG",
						"margin_type":"cross",
						"net_size":"1",
						"leverage":"2"
					}],
					"expiring_futures_positions":[{
						"product_id":"BTC-USD",
						"side":"SHORT",
						"number_of_contracts":"3",
						"entry_price":"99"
					}]
				}
			}]`),
		}
		require.NoError(t, ex.wsProcessUser(t.Context(), resp), "wsProcessUser must not error")

		data := receiveDataHandlerPayload(t, ex, time.Second)
		orders, ok := data.([]order.Detail)
		require.True(t, ok, "payload must be order details")
		require.Len(t, orders, 3, "payload must include order and positions")
		assert.True(t, orders[0].TimeInForce.Is(order.GoodTillCancel), "order should be good until cancel")
		assert.True(t, orders[0].TimeInForce.Is(order.PostOnly), "order should be post only")
		assert.Equal(t, asset.Futures, orders[1].AssetType, "position should be futures asset")
	})

	t.Run("unknown order type", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		resp := &StandardWebsocketResponse{Events: []byte(`[{"type":"snapshot","orders":[{"order_type":"WAT"}]}]`)}
		assert.ErrorIs(t, ex.wsProcessUser(t.Context(), resp), order.ErrUnrecognisedOrderType, "wsProcessUser should return unrecognised order type")
	})
}
