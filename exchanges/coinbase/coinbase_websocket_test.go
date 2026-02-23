package coinbase

import (
	"context"
	stderrors "errors"
	"log"
	"strconv"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	testsubs "github.com/thrasher-corp/gocryptotrader/internal/testing/subscriptions"
)

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
	t.Run("nil message", func(t *testing.T) {
		t.Parallel()
		err := e.wsHandleData(t.Context(), testexch.GetMockConn(t, e, ""), nil)
		var syntaxErr *gctjson.SyntaxError
		assert.True(t, stderrors.As(err, &syntaxErr) || strings.Contains(err.Error(), "Syntax error no sources available, the input json is empty"), errJSONUnmarshalUnexpected)
	})

	t.Run("error type message", func(t *testing.T) {
		t.Parallel()
		mockJSON := []byte(`{"type": "error"}`)
		err := e.wsHandleData(t.Context(), testexch.GetMockConn(t, e, ""), mockJSON)
		assert.Error(t, err)
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
		var unmarshalTypeErr *gctjson.UnmarshalTypeError
		mockJSON := []byte(`{"sequence_num": 0, "channel": "status", "events": [{"type": 1234}]}`)
		err := e.wsHandleData(t.Context(), testexch.GetMockConn(t, e, ""), mockJSON)
		assert.True(t, stderrors.As(err, &unmarshalTypeErr) || strings.Contains(err.Error(), "mismatched type with value"), errJSONUnmarshalUnexpected)
	})

	t.Run("ticker tickers unmarshal", func(t *testing.T) {
		t.Parallel()
		var unmarshalTypeErr *gctjson.UnmarshalTypeError
		mockJSON := []byte(`{"sequence_num": 0, "channel": "ticker", "events": [{"type": "moo", "tickers": false}]}`)
		err := e.wsHandleData(t.Context(), testexch.GetMockConn(t, e, ""), mockJSON)
		assert.True(t, stderrors.As(err, &unmarshalTypeErr) || strings.Contains(err.Error(), "mismatched type with value"), errJSONUnmarshalUnexpected)
	})

	t.Run("candles events type unmarshal", func(t *testing.T) {
		t.Parallel()
		var unmarshalTypeErr *gctjson.UnmarshalTypeError
		mockJSON := []byte(`{"sequence_num": 0, "channel": "candles", "events": [{"type": false}]}`)
		err := e.wsHandleData(t.Context(), testexch.GetMockConn(t, e, ""), mockJSON)
		assert.True(t, stderrors.As(err, &unmarshalTypeErr) || strings.Contains(err.Error(), "mismatched type with value"), errJSONUnmarshalUnexpected)
	})

	t.Run("market_trades events type unmarshal", func(t *testing.T) {
		t.Parallel()
		var unmarshalTypeErr *gctjson.UnmarshalTypeError
		mockJSON := []byte(`{"sequence_num": 0, "channel": "market_trades", "events": [{"type": false}]}`)
		err := e.wsHandleData(t.Context(), testexch.GetMockConn(t, e, ""), mockJSON)
		assert.True(t, stderrors.As(err, &unmarshalTypeErr) || strings.Contains(err.Error(), "mismatched type with value"), errJSONUnmarshalUnexpected)
	})

	t.Run("l2_data updates unmarshal", func(t *testing.T) {
		t.Parallel()
		var unmarshalTypeErr *gctjson.UnmarshalTypeError
		mockJSON := []byte(`{"sequence_num": 0, "channel": "l2_data", "events": [{"type": false, "updates": [{"price_level": "1.1"}]}]}`)
		err := e.wsHandleData(t.Context(), testexch.GetMockConn(t, e, ""), mockJSON)
		assert.True(t, stderrors.As(err, &unmarshalTypeErr) || strings.Contains(err.Error(), "mismatched type with value"), errJSONUnmarshalUnexpected)
	})

	t.Run("user events type unmarshal", func(t *testing.T) {
		t.Parallel()
		var unmarshalTypeErr *gctjson.UnmarshalTypeError
		mockJSON := []byte(`{"sequence_num": 0, "channel": "user", "events": [{"type": false}]}`)
		err := e.wsHandleData(t.Context(), testexch.GetMockConn(t, e, ""), mockJSON)
		assert.True(t, stderrors.As(err, &unmarshalTypeErr) || strings.Contains(err.Error(), "mismatched type with value"), errJSONUnmarshalUnexpected)
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

func TestWsHandleDataSequence(t *testing.T) {
	t.Parallel()
	connA := testexch.GetMockConn(t, e, "ws://coinbase-seq-a")
	connB := testexch.GetMockConn(t, e, "ws://coinbase-seq-b")
	buildSubMsg := func(seq uint64) []byte {
		return []byte(`{"sequence_num":` + strconv.FormatUint(seq, 10) + `,"channel":"subscriptions"}`)
	}

	assert.NoError(t, e.wsHandleData(t.Context(), connA, buildSubMsg(7)), "wsHandleData should not error for initial sequence")
	assert.NoError(t, e.wsHandleData(t.Context(), connA, buildSubMsg(8)), "wsHandleData should not error for in-order sequence")
	assert.ErrorIs(t, e.wsHandleData(t.Context(), connA, buildSubMsg(10)), errOutOfSequence, "wsHandleData should error for out-of-order sequence")
	assert.NoError(t, e.wsHandleData(t.Context(), connA, buildSubMsg(11)), "wsHandleData should not error after sequence state is resynced")
	assert.NoError(t, e.wsHandleData(t.Context(), connB, buildSubMsg(3)), "wsHandleData should not error for a different connection sequence state")
}

func TestProcessSnapshotUpdate(t *testing.T) {
	t.Parallel()
	req := WebsocketOrderbookDataHolder{Changes: []WebsocketOrderbookData{{Side: "fakeside", PriceLevel: 1.1, NewQuantity: 2.2}}, ProductID: currency.NewBTCUSD()}
	err := e.ProcessSnapshot(&req, time.Time{})
	assert.ErrorIs(t, err, order.ErrSideIsInvalid)
	err = e.ProcessUpdate(&req, time.Time{})
	assert.ErrorIs(t, err, order.ErrSideIsInvalid)
	req.Changes[0].Side = "offer"
	err = e.ProcessSnapshot(&req, time.Now())
	assert.NoError(t, err)
	err = e.ProcessUpdate(&req, time.Now())
	assert.NoError(t, err)
}

func TestGenerateSubscriptions(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	if err := testexch.Setup(e); err != nil {
		log.Fatal(err)
	}
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
	_, err = subscription.List{{Channel: "wibble"}}.ExpandTemplates(e)
	assert.ErrorContains(t, err, "subscription channel not supported: wibble")
}

func TestSubscribeUnsubscribe(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	req := subscription.List{{Channel: "heartbeat", Asset: asset.Spot, Pairs: currency.Pairs{currency.NewPairWithDelimiter(testCrypto.String(), testFiat.String(), "-")}}}
	err := subscribeForTest(t.Context(), e, req)
	assert.NoError(t, err)
	err = unsubscribeForTest(t.Context(), e, req)
	assert.NoError(t, err)
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

func TestCheckWSSequenceAdditionalCoverage(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex))
	assert.NoError(t, ex.checkWSSequence(nil, 1))
	conn := testexch.GetMockConn(t, ex, "ws://coinbase-seq")
	// first sequence seen sets expected+1
	assert.NoError(t, ex.checkWSSequence(conn, 7))
	// in-order
	assert.NoError(t, ex.checkWSSequence(conn, 8))
	// out-of-order resets expected and returns err
	err := ex.checkWSSequence(conn, 10)
	assert.ErrorIs(t, err, errOutOfSequence)
	// resumed should now accept 11
	assert.NoError(t, ex.checkWSSequence(conn, 11))
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

func TestManageSubsNilConn(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	err := ex.manageSubs(t.Context(), nil, "subscribe", subscription.List{})
	assert.ErrorIs(t, err, websocket.ErrNotConnected)
}

func TestSubscribeUnsubscribeForConnectionNilConn(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	err := ex.subscribeForConnection(t.Context(), nil, subscription.List{})
	assert.ErrorIs(t, err, websocket.ErrNotConnected)
	err = ex.unsubscribeForConnection(t.Context(), nil, subscription.List{})
	assert.ErrorIs(t, err, websocket.ErrNotConnected)
}

func TestGetWSJWTCacheAndRefresh(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	// cached token path
	ex.jwt.token = "cached"
	ex.jwt.expiresAt = time.Now().Add(time.Hour)
	tok, err := ex.GetWSJWT(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "cached", tok)

	// expired path uses GetJWT; without creds we just assert it returns an error
	ex.jwt.expiresAt = time.Now().Add(-time.Second)
	_, err = ex.GetWSJWT(t.Context())
	assert.Error(t, err)
}

func TestProcessBidAskArray(t *testing.T) {
	t.Parallel()
	snap := &WebsocketOrderbookDataHolder{Changes: []WebsocketOrderbookData{{Side: "bid", PriceLevel: 1.1, NewQuantity: 2.2}, {Side: "offer", PriceLevel: 1.2, NewQuantity: 3.3}}}
	bids, asks, err := processBidAskArray(snap, true)
	require.NoError(t, err)
	assert.Len(t, bids, 1)
	assert.Len(t, asks, 1)

	upd := &WebsocketOrderbookDataHolder{Changes: []WebsocketOrderbookData{{Side: "bid", PriceLevel: 1.1, NewQuantity: 2.2}}}
	bids, asks, err = processBidAskArray(upd, false)
	require.NoError(t, err)
	assert.Len(t, bids, 1)
	assert.Empty(t, asks)

	bad := &WebsocketOrderbookDataHolder{Changes: []WebsocketOrderbookData{{Side: "wat", PriceLevel: 1.1, NewQuantity: 2.2}}}
	_, _, err = processBidAskArray(bad, false)
	assert.ErrorIs(t, err, order.ErrSideIsInvalid)
}

func TestStatusToStandardStatusWebsocket(t *testing.T) {
	t.Parallel()
	st, err := statusToStandardStatus("PENDING")
	require.NoError(t, err)
	assert.Equal(t, order.New, st)
	_, err = statusToStandardStatus("unknown")
	assert.ErrorIs(t, err, order.ErrUnsupportedStatusType)
}

func TestStringToStandardTypeWebsocket(t *testing.T) {
	t.Parallel()
	tp, err := stringToStandardType("LIMIT_ORDER_TYPE")
	require.NoError(t, err)
	assert.Equal(t, order.Limit, tp)
	_, err = stringToStandardType("wat")
	assert.ErrorIs(t, err, order.ErrUnrecognisedOrderType)
}

func TestStringToStandardAssetWebsocket(t *testing.T) {
	t.Parallel()
	at, err := stringToStandardAsset("SPOT")
	require.NoError(t, err)
	assert.Equal(t, asset.Spot, at)
	_, err = stringToStandardAsset("wat")
	assert.ErrorIs(t, err, asset.ErrNotSupported)
}

func TestStrategyDecoderWebsocket(t *testing.T) {
	t.Parallel()
	tif, err := strategyDecoder("IMMEDIATE_OR_CANCEL")
	require.NoError(t, err)
	assert.True(t, tif.Is(order.ImmediateOrCancel))
	_, err = strategyDecoder("wat")
	assert.ErrorIs(t, err, errUnrecognisedStrategyType)
}

func TestChannelNameWebsocket(t *testing.T) {
	t.Parallel()
	name, err := channelName(&subscription.Subscription{Channel: subscription.HeartbeatChannel})
	require.NoError(t, err)
	assert.Equal(t, "heartbeats", name)
	_, err = channelName(&subscription.Subscription{Channel: "wat"})
	assert.ErrorIs(t, err, subscription.ErrNotSupported)
}

func TestProcessSnapshotUpdateSendsToOrderbook(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex))
	pair := currency.NewBTCUSD()
	require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Spot, currency.Pairs{pair}, true))
	ex.pairAliases.Load(map[currency.Pair]currency.Pairs{pair: {pair}})
	snap := WebsocketOrderbookDataHolder{ProductID: pair, Changes: []WebsocketOrderbookData{{Side: "bid", PriceLevel: 1.1, NewQuantity: 2.2}}}
	err := ex.ProcessSnapshot(&snap, time.Now())
	assert.NoError(t, err)
	upd := WebsocketOrderbookDataHolder{ProductID: pair, Changes: []WebsocketOrderbookData{{Side: "bid", PriceLevel: 1.2, NewQuantity: 1.1}}}
	err = ex.ProcessUpdate(&upd, time.Now())
	assert.NoError(t, err)
}

func receiveDataHandlerPayload(t *testing.T, ex *Exchange) any {
	t.Helper()
	select {
	case payload := <-ex.Websocket.DataHandler.C:
		return payload.Data
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for websocket data handler payload")
		return nil
	}
}

func TestWSProcessCandle(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex))

	resp := &StandardWebsocketResponse{
		Timestamp: time.Unix(1704067200, 0),
		Events: []byte(`[{
			"type":"update",
			"candles":[{
				"start":"1704067200",
				"low":"1",
				"high":"2",
				"open":"1.25",
				"close":"1.75",
				"volume":"3.5",
				"product_id":"BTC-USD"
			}]
		}]`),
	}
	require.NoError(t, ex.wsProcessCandle(t.Context(), resp))

	data := receiveDataHandlerPayload(t, ex)
	candles, ok := data.([]websocket.KlineData)
	require.True(t, ok)
	require.Len(t, candles, 1)
	assert.Equal(t, currency.NewPairWithDelimiter("BTC", "USD", "-"), candles[0].Pair)
	assert.Equal(t, asset.Spot, candles[0].AssetType)

	resp.Events = []byte(`[{"type":false}]`)
	assert.Error(t, ex.wsProcessCandle(t.Context(), resp))
}

func TestWSProcessMarketTrades(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex))

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
	require.NoError(t, ex.wsProcessMarketTrades(t.Context(), resp))

	data := receiveDataHandlerPayload(t, ex)
	trades, ok := data.([]trade.Data)
	require.True(t, ok)
	require.Len(t, trades, 1)
	assert.Equal(t, currency.NewPairWithDelimiter("BTC", "USD", "-"), trades[0].CurrencyPair)
	assert.Equal(t, order.Buy, trades[0].Side)

	resp.Events = []byte(`[{"type":false}]`)
	assert.Error(t, ex.wsProcessMarketTrades(t.Context(), resp))
}

func TestWSProcessL2(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex))

	exchangePair := currency.NewPairWithDelimiter("BTC", "USD", "-")
	aliasPair := currency.NewBTCUSD()
	require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Spot, currency.Pairs{aliasPair}, true))
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
	require.NoError(t, ex.wsProcessL2(resp))
	_, err := ex.Websocket.Orderbook.GetOrderbook(aliasPair, asset.Spot)
	assert.NoError(t, err)

	resp.Events = []byte(`[{"type":"wat","product_id":"BTC-USD","updates":[]}]`)
	assert.ErrorIs(t, ex.wsProcessL2(resp), errUnknownL2DataType)
}

func TestWSProcessUser(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex))

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
	require.NoError(t, ex.wsProcessUser(t.Context(), resp))

	data := receiveDataHandlerPayload(t, ex)
	orders, ok := data.([]order.Detail)
	require.True(t, ok)
	require.Len(t, orders, 3)
	assert.True(t, orders[0].TimeInForce.Is(order.GoodTillCancel))
	assert.True(t, orders[0].TimeInForce.Is(order.PostOnly))
	assert.Equal(t, asset.Futures, orders[1].AssetType)

	resp.Events = []byte(`[{"type":"snapshot","orders":[{"order_type":"WAT"}]}]`)
	assert.ErrorIs(t, ex.wsProcessUser(t.Context(), resp), order.ErrUnrecognisedOrderType)
}

func subscribeForTest(ctx context.Context, e *Exchange, subs subscription.List) error {
	wsRunningURL, err := e.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}
	conn, err := e.Websocket.GetConnection(wsRunningURL)
	if err != nil {
		conn, err = e.Websocket.GetConnection(coinbaseWebsocketURL)
		if err != nil {
			return err
		}
	}
	return e.subscribeForConnection(ctx, conn, subs)
}

func unsubscribeForTest(ctx context.Context, e *Exchange, subs subscription.List) error {
	wsRunningURL, err := e.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}
	conn, err := e.Websocket.GetConnection(wsRunningURL)
	if err != nil {
		conn, err = e.Websocket.GetConnection(coinbaseWebsocketURL)
		if err != nil {
			return err
		}
	}
	return e.unsubscribeForConnection(ctx, conn, subs)
}
