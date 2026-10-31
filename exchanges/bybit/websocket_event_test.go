package bybit

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGenerateEventPublicSubscriptions(t *testing.T) {
	t.Parallel()
	ex := testInstance(t)
	symbol := map[string]any{"symbol": eventTestTargetSymbol}
	ex.Features.Subscriptions = subscription.List{
		{Enabled: true, Channel: subscription.OrderbookChannel},
		{Enabled: true, Channel: ChannelEventOrderbook, Params: symbol},
		{Enabled: true, Channel: ChannelEventPublicTrade, Params: symbol},
		{Enabled: false, Channel: ChannelEventPublicTrade, Params: map[string]any{"symbol": eventTestRangeSymbol}},
		{Enabled: true, Channel: ChannelSpreadTicker, Params: symbol},
	}
	subs, err := ex.generateEventPublicSubscriptions()
	require.NoError(t, err, "generateEventPublicSubscriptions must not error")
	assert.Equal(t, []string{"orderbook.25." + eventTestTargetSymbol, "publicTrade." + eventTestTargetSymbol}, subs.QualifiedChannels(), "generateEventPublicSubscriptions should qualify each enabled event subscription by its topic")
	for _, s := range subs {
		assert.IsTypef(t, topicKey{}, s.Key, "subscription %s should be keyed by its topic", s.QualifiedChannel)
	}

	unusable := subscription.List{
		{Enabled: true, Channel: ChannelEventOrderbook, Params: map[string]any{"symbol": 5001801}},
		{Enabled: true, Channel: ChannelEventPublicTrade},
		{Enabled: true, Channel: ChannelEventOrderbook, Params: map[string]any{"symbol": ""}},
	}
	ex.Features.Subscriptions = append(subscription.List{{Enabled: true, Channel: ChannelEventPublicTrade, Params: symbol}}, unusable...)
	subs, err = ex.generateEventPublicSubscriptions()
	assert.NoError(t, err, "generateEventPublicSubscriptions should not error for subscriptions without a usable symbol")
	assert.Equal(t, []string{"publicTrade." + eventTestTargetSymbol}, subs.QualifiedChannels(), "generateEventPublicSubscriptions should return only the subscriptions with a symbol")
	for _, s := range unusable {
		assert.Truef(t, s.Enabled, "generateEventPublicSubscriptions should leave %s with params %v as configured", s.Channel, s.Params)
	}
	subs, err = ex.generateEventPublicSubscriptions()
	assert.NoError(t, err, "generateEventPublicSubscriptions should not error when generating again")
	assert.Equal(t, []string{"publicTrade." + eventTestTargetSymbol}, subs.QualifiedChannels(), "generateEventPublicSubscriptions should generate the same subscriptions again")
}

func TestEventPublicSubscribe(t *testing.T) {
	t.Parallel()
	ex := testInstance(t)
	ex.Features.Subscriptions = subscription.List{
		{Enabled: true, Channel: ChannelEventOrderbook, Params: map[string]any{"symbol": eventTestTargetSymbol}},
		{Enabled: true, Channel: ChannelEventPublicTrade, Params: map[string]any{"symbol": eventTestTargetSymbol}},
	}
	subs, err := ex.generateEventPublicSubscriptions()
	require.NoError(t, err, "generateEventPublicSubscriptions must not error")
	conn := &topicFixtureConnection{match: websocket.NewMatch(), reply: commandReply(t), handler: ex.wsHandleEventPublicData}

	require.NoError(t, ex.eventPublicSubscribe(t.Context(), conn, subs), "eventPublicSubscribe must not error")
	require.Len(t, conn.sent, 1, "eventPublicSubscribe must send one request for the batch")
	assert.Equal(t, "subscribe", conn.sent[0].Operation, "eventPublicSubscribe should send a subscribe request")
	assert.Equal(t, subs.QualifiedChannels(), conn.sent[0].Arguments, "eventPublicSubscribe should request every topic")
	for _, s := range subs {
		stored := ex.Websocket.GetSubscription(s)
		require.NotNilf(t, stored, "eventPublicSubscribe must store subscription %s", s.QualifiedChannel)
		assert.Equalf(t, subscription.SubscribedState, stored.State(), "subscription %s should be subscribed", s.QualifiedChannel)
	}

	require.NoError(t, ex.eventPublicUnsubscribe(t.Context(), conn, subs), "eventPublicUnsubscribe must not error")
	assert.Equal(t, "unsubscribe", conn.sent[1].Operation, "eventPublicUnsubscribe should send an unsubscribe request")
	for _, s := range subs {
		assert.Nilf(t, ex.Websocket.GetSubscription(s), "eventPublicUnsubscribe should remove subscription %s", s.QualifiedChannel)
	}
}

func TestWebsocketHandleEventPublicData(t *testing.T) {
	t.Parallel()
	ex := testInstance(t)
	conn := &topicFixtureConnection{match: websocket.NewMatch()}
	handle := func(ctx context.Context, b []byte) error { return ex.wsHandleEventPublicData(ctx, conn, b) }

	if !mockTests {
		symbol := eventTestSymbol(t)
		topics := []string{"orderbook.25." + symbol, "publicTrade." + symbol}
		replies, err := conn.match.Set(commandSignature(topics), 1)
		require.NoError(t, err, "match.Set must not error")
		for _, frame := range liveWindowFrames(t, mainnetStreamOrigin+eventPublicPath, topics, 5*time.Second) {
			require.NoErrorf(t, handle(t.Context(), frame), "wsHandleEventPublicData must handle %s", frame)
		}
		require.Len(t, replies, 1, "wsHandleEventPublicData must route the subscription reply")
		var reply CommandResponse
		require.NoError(t, json.Unmarshal(<-replies, &reply), "Unmarshal must not error")
		assert.ElementsMatch(t, topics, reply.Data.SuccessTopics, "the event stream should accept every topic")
		got := relayedFrames(ex)
		require.NotEmpty(t, got, "wsHandleEventPublicData must relay the orderbook snapshot")
		for _, frame := range got {
			switch f := frame.(type) {
			case *WebsocketEventOrderbookUpdate:
				assert.Equal(t, symbol, f.Data.Symbol, "orderbook frames should be for the subscribed symbol")
			case *WebsocketEventPublicTradeUpdate:
				assert.NotEmpty(t, f.Data, "trade frames should hold trades")
			default:
				assert.Failf(t, "unexpected relayed frame", "the event stream should not relay %T", frame)
			}
		}
		return
	}

	testexch.FixtureToDataHandler(t, "testdata/wsEventPublic.jsonl", handle)
	exp := []any{
		&WebsocketEventOrderbookUpdate{
			Topic:     "orderbook.25.ETHUSDT-6NOV26-2200-BELOW",
			Type:      "snapshot",
			Timestamp: types.Time(time.UnixMilli(1791534287389)),
			Data: EventOrderbookData{
				Symbol: "ETHUSDT-6NOV26-2200-BELOW",
				Entries: []EventOrderbookEntry{
					{PayoutRatio: 3.5211, OrderValue: 500},
					{PayoutRatio: 3.3021, OrderValue: 1000},
				},
				UpdateID: 11240,
				Sequence: 101130842,
			},
			CreationTimestamp: types.Time(time.UnixMilli(1791534284744)),
		},
		&WebsocketEventOrderbookUpdate{
			Topic:     "orderbook.25.ETHUSDT-6NOV26-2200-BELOW",
			Type:      "delta",
			Timestamp: types.Time(time.UnixMilli(1791534292049)),
			Data: EventOrderbookData{
				Symbol: "ETHUSDT-6NOV26-2200-BELOW",
				Entries: []EventOrderbookEntry{
					{PayoutRatio: 3.3021},
				},
				UpdateID: 11241,
				Sequence: 101131105,
			},
			CreationTimestamp: types.Time(time.UnixMilli(1791534292029)),
		},
		&WebsocketEventPublicTradeUpdate{
			Topic:     "publicTrade.ETHUSDT-6NOV26-2200-BELOW",
			Type:      "snapshot",
			Timestamp: types.Time(time.UnixMilli(1791008160380)),
			Data: []EventWebsocketTrade{
				{
					OrderFillTimestamp: types.Time(time.UnixMilli(1791008160374)),
					Symbol:             "ETHUSDT-6NOV26-2200-BELOW",
					PayoutRatio:        3.4907,
					Amount:             500,
					TradeID:            "ea89cadf-6471-5fd4-8248-20b470d8478b",
					Sequence:           101174421,
				},
			},
		},
		&WebsocketEventOrderbookUpdate{
			Topic:     "orderbook.25.ETHUSDT-30OCT26-2100-ABOVE",
			Type:      "snapshot",
			Timestamp: types.Time(time.UnixMilli(1793347198806)),
			Data: EventOrderbookData{
				Symbol:   "ETHUSDT-30OCT26-2100-ABOVE",
				Entries:  []EventOrderbookEntry{},
				UpdateID: 1,
				Sequence: -1,
			},
			CreationTimestamp: types.Time(time.UnixMilli(1793326244226)),
		},
	}
	assert.Equal(t, exp, relayedFrames(ex), "wsHandleEventPublicData should relay every frame")

	assert.NoError(t, handle(t.Context(), []byte(`{"args":["1791534288825"],"op":"pong"}`)), "wsHandleEventPublicData should ignore a pong")
	assert.ErrorIs(t, handle(t.Context(), []byte(`{"success":true,"conn_id":"x","data":{"failTopics":[],"successTopics":["publicTrade.X"]},"type":"COMMAND_RESP"}`)), websocket.ErrSignatureNotMatched, "wsHandleEventPublicData should reject a reply no request awaits")
	assert.ErrorIs(t, handle(t.Context(), []byte(`{"topic":"tickers.ETHUSDT-6NOV26-2200-BELOW"}`)), errUnhandledStreamData, "wsHandleEventPublicData should reject an unknown topic")
	assert.Error(t, handle(t.Context(), []byte(`{"topic":`)), "wsHandleEventPublicData should reject malformed JSON")
}

func TestWebsocketHandleEventPrivateData(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		reply := livePrivateSubscribe(t, []string{ChannelEventPosition, ChannelEventExecution, ChannelEventOrder})
		assert.Truef(t, reply.Success, "the private stream should accept the event topics: %s", reply.ReturnMessage)
		return
	}
	ex := testInstance(t)
	testexch.FixtureToDataHandler(t, "testdata/wsEventPrivate.jsonl", ex.wsHandleEventPrivateData)
	exp := []any{
		&WebsocketEventPositionUpdate{
			ID:           "592324d2bce751-ad38-48eb-8f42-4671d1fb4d4e",
			Topic:        "position.event",
			CreationTime: types.Time(time.UnixMilli(1791619200020)),
			Data: []EventPosition{
				{
					Symbol:                  "ETHUSDT-10OCT26-2580-2640-OUT",
					BaseCoin:                currency.ETH,
					SettleCoin:              currency.USDT,
					Side:                    "Sell",
					EventContractType:       "Range",
					EventContractDirection:  "RANGE_OUT",
					AveragePayoutRatio:      1.18,
					PositionValue:           200,
					EventContractSettleTime: types.Time(time.Unix(1791619200, 0)),
					CumulativeClosedPNL:     -36,
					CreatedTime:             types.Time(time.UnixMilli(1791008195002)),
					UpdatedTime:             types.Time(time.UnixMilli(1791619200015)),
					Sequence:                101180113,
				},
			},
		},
		&WebsocketEventExecutionUpdate{
			ID:           "1509583_BTCUSDT-15MIN-DOWN_1311",
			Topic:        "execution",
			CreationTime: types.Time(time.UnixMilli(1791008150640)),
			Data: []EventExecutionUpdate{
				{
					Category:                    "event",
					TransactionID:               "0176bba4-66fa-542b-af33-cea000a83dc0",
					ExecutionID:                 "8b9c0d1e-2f3a-5b4c-8d6e-7f8091a2b3c4",
					OrderID:                     "0f2b7c55-6e1d-4b8a-9d3f-2a4c6e8b0d12",
					OrderLinkID:                 "event-gct-003",
					Symbol:                      "BTCUSDT-15MIN-DOWN",
					SymbolID:                    500017,
					BaseCoin:                    currency.BTC,
					QuoteCoin:                   currency.USDT,
					SettleCoin:                  currency.USDT,
					Side:                        "Sell",
					FeeCoin:                     currency.USDT,
					ExecutionType:               "Trade",
					ExecutionPrice:              1.95,
					ExecutionValue:              50,
					ExecutionFee:                2.25,
					ExecutionFeeRate:            0.045,
					OrderPrice:                  1.95,
					EventContractType:           1,
					EventContractDirection:      2,
					EventContractDurationWindow: 900,
					EventContractOrderValue:     50,
					EventContractIndexPrice:     121480.2,
					EventContractPayout:         97.5,
					EntryPrice:                  1.95,
					SettlePrice:                 121512.4,
					GrossPayoutRatio:            1.95,
					SessionRealisedPNL:          45.25,
					TransactionTime:             types.Time(time.UnixMilli(1791008150631)),
					SettleTime:                  types.Time(time.Unix(1791009000, 0)),
					CrossSequence:               1311,
					TimeInForce:                 "GoodTillCancel",
					ExtraFees:                   []EventExtraFee{},
				},
			},
		},
		&WebsocketEventExecutionUpdate{
			ID:           "1509583_ETHUSDT-6NOV26-2200-BELOW_101174421",
			Topic:        "execution.event",
			CreationTime: types.Time(time.UnixMilli(1791008160380)),
			Data: []EventExecutionUpdate{
				{
					Category:                 "event",
					TransactionID:            "5b1e9d3a-2c4f-5a6b-8d7e-1f0a2b3c4d5e",
					ExecutionID:              "ea89cadf-6471-5fd4-8248-20b470d8478b",
					OrderID:                  "746a6c87-a6dd-4008-8889-37e694046681",
					OrderLinkID:              "event-gct-001",
					Symbol:                   "ETHUSDT-6NOV26-2200-BELOW",
					SymbolID:                 5001801,
					BaseCoin:                 currency.ETH,
					QuoteCoin:                currency.USDT,
					SettleCoin:               currency.USDT,
					Side:                     "Sell",
					FeeCoin:                  currency.USDT,
					ExecutionType:            "Trade",
					ExecutionPrice:           3.4907,
					ExecutionValue:           500,
					ExecutionFee:             22.5,
					ExecutionFeeRate:         0.045,
					OrderPrice:               3.4907,
					EventContractType:        2,
					EventContractDirection:   4,
					EventContractTargetPrice: 2200,
					EventContractOrderValue:  500,
					EventContractIndexPrice:  2391.27,
					EventContractPayout:      1745.35,
					EntryPrice:               3.4907,
					GrossPayoutRatio:         3.4907,
					TransactionTime:          types.Time(time.UnixMilli(1791008160374)),
					SettleTime:               types.Time(time.Unix(1793952000, 0)),
					CrossSequence:            101174421,
					TimeInForce:              "GoodTillCancel",
					ExtraFees: []EventExtraFee{
						{
							FeeCoin:    currency.USDT,
							FeeType:    "GST",
							SubFeeType: "IND_GST",
							FeeRate:    0.0000675,
							Fee:        0.003375,
						},
					},
				},
				{
					Category:                "event",
					TransactionID:           "3c4d5e6f-7a8b-5c9d-8e0f-1a2b3c4d5e6f",
					ExecutionID:             "3f6a1c2e-9d8b-5e7f-a1b2-c3d4e5f60718",
					OrderID:                 "9c1d6f10-3b9a-4d7e-a0c2-5e8f7b2d4a61",
					OrderLinkID:             "event-gct-002",
					Symbol:                  "ETHUSDT-10OCT26-2580-2640-OUT",
					SymbolID:                5001709,
					BaseCoin:                currency.ETH,
					QuoteCoin:               currency.USDT,
					SettleCoin:              currency.USDT,
					Side:                    "Sell",
					FeeCoin:                 currency.USDT,
					ExecutionType:           "Trade",
					ExecutionPrice:          1.18,
					ExecutionValue:          200,
					ExecutionFee:            9,
					ExecutionFeeRate:        0.045,
					OrderPrice:              1.18,
					EventContractType:       3,
					EventContractDirection:  6,
					EventContractLowerBound: 2580,
					EventContractUpperBound: 2640,
					EventContractOrderValue: 1000,
					EventContractIndexPrice: 2391.27,
					EventContractPayout:     236,
					EntryPrice:              1.18,
					GrossPayoutRatio:        1.18,
					TransactionTime:         types.Time(time.UnixMilli(1791008195002)),
					SettleTime:              types.Time(time.Unix(1791619200, 0)),
					CrossSequence:           101174788,
					TimeInForce:             "GoodTillCancel",
					ExtraFees:               []EventExtraFee{},
				},
			},
		},
		&WebsocketEventOrderUpdate{
			ID:           "1509583_BTCUSDT-15MIN-DOWN_1311",
			Topic:        "order",
			CreationTime: types.Time(time.UnixMilli(1791008150640)),
			Data: []EventOrderUpdate{
				{
					Category:                    "event",
					OrderID:                     "0f2b7c55-6e1d-4b8a-9d3f-2a4c6e8b0d12",
					OrderLinkID:                 "event-gct-003",
					Symbol:                      "BTCUSDT-15MIN-DOWN",
					SymbolID:                    500017,
					Side:                        "Sell",
					BaseCoin:                    currency.BTC,
					QuoteCoin:                   currency.USDT,
					SettleCoin:                  currency.USDT,
					OrderStatus:                 "PartiallyFilled",
					EventContractType:           "UpDown",
					EventContractDirection:      "DOWN",
					EventContractDurationWindow: 900,
					EventContractOrderValue:     5000,
					EventContractIndexPrice:     121480.2,
					EventContractSettleTime:     types.Time(time.Unix(1791009000, 0)),
					PayoutRatio:                 1.95,
					OrderAveragePayoutRatio:     1.95,
					CumulativeExecutedValue:     50,
					CumulativeExecutedFee:       2.25,
					LeavesValue:                 4950,
					CumulativePayout:            97.5,
					CreateType:                  1,
					CancelType:                  "UNKNOWN",
					RejectReason:                "EC_NoError",
					ExtraFees:                   []EventExtraFee{},
					TransactionTime:             types.Time(time.UnixMilli(1791008150631)),
					CreatedTime:                 types.Time(time.UnixMilli(1791008100118)),
					UpdatedTime:                 types.Time(time.UnixMilli(1791008150631)),
				},
			},
		},
		&WebsocketEventOrderUpdate{
			ID:           "1509583_ETHUSDT-10OCT26-2580-2640-OUT_101174788",
			Topic:        "order.event",
			CreationTime: types.Time(time.UnixMilli(1791008195010)),
			Data: []EventOrderUpdate{
				{
					Category:                "event",
					OrderID:                 "9c1d6f10-3b9a-4d7e-a0c2-5e8f7b2d4a61",
					OrderLinkID:             "event-gct-002",
					Symbol:                  "ETHUSDT-10OCT26-2580-2640-OUT",
					SymbolID:                5001709,
					Side:                    "Sell",
					BaseCoin:                currency.ETH,
					QuoteCoin:               currency.USDT,
					SettleCoin:              currency.USDT,
					OrderStatus:             "PartiallyFilled",
					EventContractType:       "Range",
					EventContractDirection:  "RANGE_OUT",
					EventContractLowerBound: 2580,
					EventContractUpperBound: 2640,
					EventContractOrderValue: 1000,
					EventContractIndexPrice: 2391.27,
					EventContractSettleTime: types.Time(time.Unix(1791619200, 0)),
					PayoutRatio:             1.18,
					OrderAveragePayoutRatio: 1.18,
					CumulativeExecutedValue: 200,
					CumulativeExecutedFee:   9,
					LeavesValue:             800,
					CumulativePayout:        236,
					CreateType:              1,
					CancelType:              "UNKNOWN",
					RejectReason:            "EC_NoError",
					ExtraFees: []EventExtraFee{
						{
							FeeCoin:    currency.USDT,
							FeeType:    "GST",
							SubFeeType: "IND_GST",
							FeeRate:    0.0000675,
							Fee:        0.0135,
						},
					},
					TransactionTime: types.Time(time.UnixMilli(1791008195002)),
					CreatedTime:     types.Time(time.UnixMilli(1791008190511)),
					UpdatedTime:     types.Time(time.UnixMilli(1791008195002)),
				},
				{
					Category:                 "event",
					OrderID:                  "746a6c87-a6dd-4008-8889-37e694046681",
					OrderLinkID:              "event-gct-001",
					Symbol:                   "ETHUSDT-6NOV26-2200-BELOW",
					SymbolID:                 5001801,
					Side:                     "Sell",
					BaseCoin:                 currency.ETH,
					QuoteCoin:                currency.USDT,
					SettleCoin:               currency.USDT,
					OrderStatus:              "Filled",
					EventContractType:        "Target",
					EventContractDirection:   "BELOW",
					EventContractTargetPrice: 2200,
					EventContractOrderValue:  500,
					EventContractIndexPrice:  2391.27,
					EventContractSettleTime:  types.Time(time.Unix(1793952000, 0)),
					PayoutRatio:              3.4907,
					OrderAveragePayoutRatio:  3.4907,
					CumulativeExecutedValue:  500,
					CumulativeExecutedFee:    22.5,
					CumulativePayout:         1745.35,
					CreateType:               1,
					CancelType:               "UNKNOWN",
					RejectReason:             "EC_NoError",
					ExtraFees:                []EventExtraFee{},
					TransactionTime:          types.Time(time.UnixMilli(1791008160374)),
					CreatedTime:              types.Time(time.UnixMilli(1791008100917)),
					UpdatedTime:              types.Time(time.UnixMilli(1791008160374)),
				},
			},
		},
	}
	assert.Equal(t, exp, relayedFrames(ex), "wsHandleEventPrivateData should relay every frame")

	assert.ErrorIs(t, ex.wsHandleEventPrivateData(t.Context(), []byte(`{"topic":"wallet"}`)), errUnhandledStreamData, "wsHandleEventPrivateData should reject an unknown topic")
	assert.Error(t, ex.wsHandleEventPrivateData(t.Context(), []byte(`{"op":"pong"}`)), "wsHandleEventPrivateData should reject a frame without a topic")
	assert.Error(t, ex.wsHandleEventPrivateData(t.Context(), []byte(`{"topic":"order.event","data":{}}`)), "wsHandleEventPrivateData should reject a malformed frame")
}

func TestIsEventContractFrame(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		frame string
		exp   bool
	}{
		{frame: `{"topic":"execution","data":[{"category":"event","symbol":"BTCUSDT-15MIN-DOWN"}]}`, exp: true},
		{frame: `{"topic":"order","data":[{"category":"linear","symbol":"BTCUSDT"}]}`},
		{frame: `{"topic":"order","data":[]}`},
		{frame: `{"topic":"order"`},
	} {
		assert.Equalf(t, tc.exp, isEventContractFrame([]byte(tc.frame)), "isEventContractFrame should return %t for %s", tc.exp, tc.frame)
	}
}
