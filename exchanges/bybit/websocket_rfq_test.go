package bybit

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// rfqReply returns a function building the RFQ public connection's reply to a request, which echoes its request ID;
// a request naming a topic containing FAIL fails as Bybit fails an unknown topic
func rfqReply(tb testing.TB) func(*SubscriptionRequest) []byte {
	tb.Helper()
	return func(req *SubscriptionRequest) []byte {
		reply := SubscriptionResponse{Success: true, ConnectionID: "dat23sjnufgc73fclss0-1m5nqb", RequestID: req.RequestID, Operation: req.Operation}
		for _, topic := range req.Arguments {
			if strings.Contains(topic, "FAIL") {
				reply.Success, reply.ReturnMessage = false, "Params Error"
			}
		}
		b, err := json.Marshal(reply)
		require.NoError(tb, err, "Marshal must not error")
		return b
	}
}

func TestGenerateRFQPublicSubscriptions(t *testing.T) {
	t.Parallel()
	ex := testInstance(t)
	ex.Features.Subscriptions = subscription.List{
		{Enabled: true, Channel: subscription.AllTradesChannel},
		{Enabled: true, Channel: ChannelRFQPublicTrades},
		{Enabled: true, Channel: ChannelRFQQuotes, Authenticated: true},
	}
	subs, err := ex.generateRFQPublicSubscriptions()
	require.NoError(t, err, "generateRFQPublicSubscriptions must not error")
	require.Len(t, subs, 1, "generateRFQPublicSubscriptions must generate the public trade subscription")
	assert.Equal(t, ChannelRFQPublicTrades, subs[0].QualifiedChannel, "generateRFQPublicSubscriptions should qualify the subscription by its topic")
	assert.IsType(t, topicKey{}, subs[0].Key, "generateRFQPublicSubscriptions should key the subscription by its topic")
}

func TestRFQPublicSubscribe(t *testing.T) {
	t.Parallel()
	ex := testInstance(t)
	ex.Features.Subscriptions = subscription.List{{Enabled: true, Channel: ChannelRFQPublicTrades}}
	subs, err := ex.generateRFQPublicSubscriptions()
	require.NoError(t, err, "generateRFQPublicSubscriptions must not error")
	conn := &topicFixtureConnection{match: websocket.NewMatch(), reply: rfqReply(t), handler: ex.wsHandleRFQPublicData}

	require.NoError(t, ex.rfqPublicSubscribe(t.Context(), conn, subs), "rfqPublicSubscribe must not error")
	require.Len(t, conn.sent, 1, "rfqPublicSubscribe must send one request")
	assert.Equal(t, "subscribe", conn.sent[0].Operation, "rfqPublicSubscribe should send a subscribe request")
	assert.Equal(t, []string{ChannelRFQPublicTrades}, conn.sent[0].Arguments, "rfqPublicSubscribe should request the public trade topic")
	stored := ex.Websocket.GetSubscription(subs[0])
	require.NotNil(t, stored, "rfqPublicSubscribe must store the subscription")
	assert.Equal(t, subscription.SubscribedState, stored.State(), "the subscription should be subscribed")

	require.NoError(t, ex.rfqPublicUnsubscribe(t.Context(), conn, subs), "rfqPublicUnsubscribe must not error")
	assert.Equal(t, "unsubscribe", conn.sent[1].Operation, "rfqPublicUnsubscribe should send an unsubscribe request")
	assert.Nil(t, ex.Websocket.GetSubscription(subs[0]), "rfqPublicUnsubscribe should remove the subscription")

	failing := subscription.List{topicSubscription(&subscription.Subscription{Channel: ChannelRFQPublicTrades}, "rfq.open.FAIL")}
	err = ex.rfqPublicSubscribe(t.Context(), conn, failing)
	require.ErrorIs(t, err, errTopicsRejected, "rfqPublicSubscribe must report a topic Bybit rejected")
	assert.ErrorContains(t, err, "rfq.open.FAIL", "rfqPublicSubscribe should name the rejected topic")
	assert.ErrorContains(t, err, "Params Error", "rfqPublicSubscribe should include Bybit's reason")
	assert.Nil(t, ex.Websocket.GetSubscription(failing[0]), "rfqPublicSubscribe should not store a rejected subscription")

	conn.sendErr = errFixtureSend
	assert.ErrorIs(t, ex.rfqPublicSubscribe(t.Context(), conn, subs), errFixtureSend, "rfqPublicSubscribe should return a send failure")

	unrouted := &topicFixtureConnection{match: websocket.NewMatch(), reply: func(*SubscriptionRequest) []byte { return []byte(`{"success":`) }}
	assert.Error(t, ex.rfqPublicSubscribe(t.Context(), unrouted, subs), "rfqPublicSubscribe should return a malformed reply's error")
}

func TestManageRFQSubscriptions(t *testing.T) {
	t.Parallel()
	ex := testInstance(t)
	trades := topicSubscription(&subscription.Subscription{Enabled: true, Channel: ChannelRFQPublicTrades}, ChannelRFQPublicTrades)
	// The handler passes each reply to its request by the request ID it echoes, so a request waiting on anything else
	// would not be answered
	conn := &topicFixtureConnection{match: websocket.NewMatch(), reply: rfqReply(t), handler: ex.wsHandleRFQPublicData}
	require.NoError(t, ex.manageRFQSubscriptions(t.Context(), conn, "subscribe", subscription.List{trades}), "manageRFQSubscriptions must not error")
	require.Len(t, conn.sent, 1, "manageRFQSubscriptions must send one request")
	assert.Equal(t, "subscribe", conn.sent[0].Operation, "manageRFQSubscriptions should send the operation")
	assert.Equal(t, []string{ChannelRFQPublicTrades}, conn.sent[0].Arguments, "manageRFQSubscriptions should send the topic")
	assert.NotEmpty(t, conn.sent[0].RequestID, "manageRFQSubscriptions should send a request ID for the reply to echo")
	stored := ex.Websocket.GetSubscription(trades)
	require.NotNil(t, stored, "manageRFQSubscriptions must store the subscription")
	assert.Equal(t, subscription.SubscribedState, stored.State(), "manageRFQSubscriptions should mark the subscription subscribed")

	// Bybit rejects a whole request for one bad topic, so a rejected batch fails all its topics; the batches after it
	// are still sent
	batches := make(subscription.List, 11)
	for i := range batches {
		topic := "rfq.open.public.trades." + strconv.Itoa(i)
		if i == 3 {
			topic = "rfq.open.FAIL"
		}
		batches[i] = topicSubscription(&subscription.Subscription{Enabled: true, Channel: ChannelRFQPublicTrades}, topic)
	}
	conn.sent = nil
	err := ex.manageRFQSubscriptions(t.Context(), conn, "subscribe", batches)
	require.ErrorIs(t, err, errTopicsRejected, "manageRFQSubscriptions must report the topics of a rejected request")
	assert.ErrorContains(t, err, "Params Error", "manageRFQSubscriptions should carry Bybit's reason")
	for _, s := range batches[:10] {
		assert.ErrorContainsf(t, err, s.QualifiedChannel, "manageRFQSubscriptions should name rejected topic %s", s.QualifiedChannel)
		assert.Nilf(t, ex.Websocket.GetSubscription(s), "manageRFQSubscriptions should not store rejected topic %s", s.QualifiedChannel)
	}
	require.Len(t, conn.sent, 2, "manageRFQSubscriptions must send ten topics a request")
	assert.Len(t, conn.sent[0].Arguments, 10, "manageRFQSubscriptions should send ten topics in the first request")
	assert.Equal(t, []string{"rfq.open.public.trades.10"}, conn.sent[1].Arguments, "manageRFQSubscriptions should send the rest after a rejected request")
	assert.NotNil(t, ex.Websocket.GetSubscription(batches[10]), "manageRFQSubscriptions should store the topic of the accepted request")

	conn.sent = nil
	require.NoError(t, ex.manageRFQSubscriptions(t.Context(), conn, "unsubscribe", subscription.List{trades}), "manageRFQSubscriptions must not error when unsubscribing")
	assert.Equal(t, "unsubscribe", conn.sent[0].Operation, "manageRFQSubscriptions should send the operation")
	assert.Nil(t, ex.Websocket.GetSubscription(trades), "manageRFQSubscriptions should remove an unsubscribed topic")

	conn.sendErr = errFixtureSend
	err = ex.manageRFQSubscriptions(t.Context(), conn, "subscribe", subscription.List{trades})
	require.ErrorIs(t, err, errFixtureSend, "manageRFQSubscriptions must return a failed send")
	assert.ErrorContains(t, err, "subscribe "+ChannelRFQPublicTrades, "manageRFQSubscriptions should name the topics it could not send")
	assert.Nil(t, ex.Websocket.GetSubscription(trades), "manageRFQSubscriptions should not store a topic it could not send")

	nullReply := &topicFixtureConnection{match: websocket.NewMatch(), reply: func(*SubscriptionRequest) []byte { return []byte(`null`) }}
	err = ex.manageRFQSubscriptions(t.Context(), nullReply, "subscribe", subscription.List{trades})
	require.ErrorIs(t, err, errTopicsRejected, "manageRFQSubscriptions must fail every topic of a null reply")
	assert.ErrorIs(t, err, common.ErrNoResponse, "manageRFQSubscriptions should report a null reply")
	assert.Nil(t, ex.Websocket.GetSubscription(trades), "manageRFQSubscriptions should not store a topic whose reply is null")
	unrouted := &topicFixtureConnection{match: websocket.NewMatch(), reply: func(*SubscriptionRequest) []byte { return []byte(`{"success":`) }}
	require.Error(t, ex.manageRFQSubscriptions(t.Context(), unrouted, "subscribe", subscription.List{trades}), "manageRFQSubscriptions must error for a reply that does not decode")
	assert.Nil(t, ex.Websocket.GetSubscription(trades), "manageRFQSubscriptions should not store a topic whose reply does not decode")
}

func TestWebsocketHandleRFQPublicData(t *testing.T) {
	t.Parallel()
	ex := testInstance(t)
	conn := &topicFixtureConnection{match: websocket.NewMatch()}
	handle := func(ctx context.Context, b []byte) error { return ex.wsHandleRFQPublicData(ctx, conn, b) }

	if !mockTests {
		replies, err := conn.match.Set("gct-live", 1)
		require.NoError(t, err, "match.Set must not error")
		for _, frame := range liveWindowFrames(t, mainnetStreamOrigin+rfqPublicPath, []string{ChannelRFQPublicTrades}, 5*time.Second) {
			require.NoErrorf(t, handle(t.Context(), frame), "wsHandleRFQPublicData must handle %s", frame)
		}
		require.Len(t, replies, 1, "wsHandleRFQPublicData must route the subscription reply")
		var reply SubscriptionResponse
		require.NoError(t, json.Unmarshal(<-replies, &reply), "Unmarshal must not error")
		assert.True(t, reply.Success, "the RFQ stream should accept the public trade topic")
		for _, frame := range relayedFrames(ex) {
			f, ok := frame.(*WebsocketRFQPublicTradeUpdate)
			require.Truef(t, ok, "wsHandleRFQPublicData must relay public trade frames, not %T", frame)
			assert.NotEmpty(t, f.Data, "public trade frames should hold trades")
		}
		return
	}

	testexch.FixtureToDataHandler(t, "testdata/wsRFQPublic.jsonl", handle)
	exp := []any{
		&WebsocketRFQPublicTradeUpdate{
			ID:           "rfq_public_trades_1791008130310",
			Topic:        "rfq.open.public.trades",
			CreationTime: types.Time(time.UnixMilli(1791008130310)),
			Data: []RFQPublicTrade{
				{
					RFQID:        "1791008100511241492396882271696580",
					StrategyType: "custom",
					CreatedAt:    types.Time(time.UnixMilli(1791008130177)),
					UpdatedAt:    types.Time(time.UnixMilli(1791008130301)),
					Legs: []RFQPublicTradeLeg{
						{
							Category:  "option",
							Symbol:    "BTC-25DEC26-100000-C",
							Side:      "Buy",
							Price:     1250.5,
							Quantity:  10,
							MarkPrice: 121480.2,
						},
						{
							Category:  "linear",
							Symbol:    "BTCUSDT",
							Side:      "Sell",
							Price:     121500,
							Quantity:  4.6,
							MarkPrice: 121475.33,
						},
					},
				},
			},
		},
	}
	assert.Equal(t, exp, relayedFrames(ex), "wsHandleRFQPublicData should relay every frame")

	ch, err := conn.match.Set("100", 1)
	require.NoError(t, err, "match.Set must not error")
	reply := []byte(`{"req_id":"100","success":true,"ret_msg":"","op":"subscribe","conn_id":"dat23sjnufgc73fclss0-1m5nqb"}`)
	require.NoError(t, handle(t.Context(), reply), "wsHandleRFQPublicData must route a subscription reply")
	require.Len(t, ch, 1, "wsHandleRFQPublicData must deliver the reply to its request")
	assert.Equal(t, reply, <-ch, "wsHandleRFQPublicData should deliver the reply to its request")
	assert.NoError(t, handle(t.Context(), []byte(`{"req_id":"ping1","op":"pong","args":["1791534244031"],"conn_id":"dat23sjnufgc73fclss0-1m5nqb"}`)), "wsHandleRFQPublicData should ignore a pong")
	assert.ErrorIs(t, handle(t.Context(), []byte(`{"req_id":"101","success":true,"op":"unsubscribe"}`)), websocket.ErrSignatureNotMatched, "wsHandleRFQPublicData should reject a reply no request awaits")
	assert.ErrorIs(t, handle(t.Context(), []byte(`{"op":"auth","success":true}`)), errUnhandledStreamData, "wsHandleRFQPublicData should reject an unknown operation")
	assert.ErrorIs(t, handle(t.Context(), []byte(`null`)), errUnhandledStreamData, "wsHandleRFQPublicData should reject a null frame")
	assert.ErrorIs(t, handle(t.Context(), []byte(`{"topic":"rfq.open.rfqs"}`)), errUnhandledStreamData, "wsHandleRFQPublicData should reject an unknown topic")
	assert.Error(t, handle(t.Context(), []byte(`{"topic":`)), "wsHandleRFQPublicData should reject malformed JSON")
	assert.Error(t, handle(t.Context(), []byte(`{"op":["subscribe"]}`)), "wsHandleRFQPublicData should reject a malformed reply")
}

func TestWebsocketHandleRFQPrivateData(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		reply := livePrivateSubscribe(t, []string{ChannelRFQQuotes, ChannelRFQTrades, ChannelRFQInquiries})
		assert.Truef(t, reply.Success, "the private stream should accept the RFQ topics: %s", reply.ReturnMessage)
		return
	}
	ex := testInstance(t)
	testexch.FixtureToDataHandler(t, "testdata/wsRFQPrivate.jsonl", ex.wsHandleRFQPrivateData)
	exp := []any{
		&WebsocketRFQQuoteUpdate{
			ID:           "rfq_quotes_1791008120140",
			Topic:        "rfq.open.quotes",
			CreationTime: types.Time(time.UnixMilli(1791008120140)),
			Data: []RFQQuoteUpdate{
				{
					RFQID:       "1791008100511241492396882271696580",
					QuoteID:     "1791008120130044334361923221559805",
					QuoteLinkID: "quote-gct-001",
					ExpiresAt:   types.Time(time.UnixMilli(1791008180126)),
					DeskCode:    "SPMAKERTEST",
					Status:      "Active",
					CreatedAt:   types.Time(time.UnixMilli(1791008120126)),
					UpdatedAt:   types.Time(time.UnixMilli(1791008120126)),
					QuoteBuyList: []RFQQuoteUpdateLeg{
						{Category: "option", Symbol: "BTC-25DEC26-100000-C", Price: 1250.5, Quantity: 10},
					},
					QuoteSellList: []RFQQuoteUpdateLeg{
						{Category: "option", Symbol: "BTC-25DEC26-100000-C", Price: 1260, Quantity: 10},
					},
				},
			},
		},
		&WebsocketRFQQuoteUpdate{
			ID:           "rfq_quotes_1791008130305",
			Topic:        "rfq.open.quotes",
			CreationTime: types.Time(time.UnixMilli(1791008130305)),
			Data: []RFQQuoteUpdate{
				{
					RFQID:             "1791008100511241492396882271696580",
					RFQLinkID:         "rfq-gct-001",
					QuoteID:           "1791008120130044334361923221559805",
					ExpiresAt:         types.Time(time.UnixMilli(1791008180126)),
					DeskCode:          "SPMAKERTEST",
					Status:            "Filled",
					ExecutedQuoteSide: "Buy",
					CreatedAt:         types.Time(time.UnixMilli(1791008120126)),
					UpdatedAt:         types.Time(time.UnixMilli(1791008130301)),
					QuoteBuyList: []RFQQuoteUpdateLeg{
						{Category: "option", Symbol: "BTC-25DEC26-100000-C", Price: 1250.5, Quantity: 10},
					},
					QuoteSellList: []RFQQuoteUpdateLeg{
						{Category: "option", Symbol: "BTC-25DEC26-100000-C", Price: 1260, Quantity: 10},
					},
				},
			},
		},
		&WebsocketRFQTradeUpdate{
			ID:           "rfq_trades_1791008130308",
			Topic:        "rfq.open.trades",
			CreationTime: types.Time(time.UnixMilli(1791008130308)),
			Data: []RFQTrade{
				{
					RFQID:         "1791008100511241492396882271696580",
					RFQLinkID:     "rfq-gct-001",
					QuoteID:       "1791008120130044334361923221559805",
					QuoteSide:     "Buy",
					StrategyType:  "custom",
					Status:        "Filled",
					RFQDeskCode:   "1nu9d1",
					QuoteDeskCode: "SPMAKERTEST",
					CreatedAt:     types.Time(time.UnixMilli(1791008130177)),
					UpdatedAt:     types.Time(time.UnixMilli(1791008130301)),
					Legs: []RFQTradeLeg{
						{
							Category:     "option",
							OrderID:      "64fe4108-555e-4361-ae2d-3a8d0c292859",
							Symbol:       "BTC-25DEC26-100000-C",
							Side:         "Buy",
							Price:        1250.5,
							Quantity:     10,
							MarkPrice:    121480.2,
							ExecutionFee: 7.5,
							ExecutionID:  "42b8be1e-36cf-4aba-bb75-4602cc11df37",
						},
					},
				},
				{
					RFQID:         "1791008090883736854420161904593980",
					QuoteID:       "1791008095586174663206671159484665",
					QuoteLinkID:   "quote-gct-000",
					QuoteSide:     "Sell",
					StrategyType:  "PerpBasis",
					Status:        "Failed",
					RFQDeskCode:   "1nu9d1",
					QuoteDeskCode: "SPMAKERTEST",
					CreatedAt:     types.Time(time.UnixMilli(1791008095588)),
					UpdatedAt:     types.Time(time.UnixMilli(1791008096046)),
					Legs: []RFQTradeLeg{
						{
							Category:      "linear",
							OrderID:       "69667acb-7048-48d7-90b9-ccbdfd423130",
							Symbol:        "BTCUSDT",
							Side:          "Buy",
							Price:         121038,
							Quantity:      1,
							ResultCode:    110007,
							ResultMessage: "Insufficient available balance",
							RejectParty:   "Taker",
						},
					},
				},
			},
		},
		&WebsocketRFQInquiryUpdate{
			ID:           "rfq_rfqs_1791008100520",
			Topic:        "rfq.open.rfqs",
			CreationTime: types.Time(time.UnixMilli(1791008100520)),
			Data: []RFQInquiryUpdate{
				{
					RFQID:     "1791008100511241492396882271696580",
					RFQLinkID: "rfq-gct-001",
					Counterparties: []string{
						"SPMAKERTEST",
					},
					ExpiresAt:              types.Time(time.UnixMilli(1791008700512)),
					StrategyType:           "custom",
					Status:                 "Active",
					AcceptOtherQuoteStatus: true,
					DeskCode:               "1nu9d1",
					CreatedAt:              types.Time(time.UnixMilli(1791008100512)),
					UpdatedAt:              types.Time(time.UnixMilli(1791008100512)),
					Legs: []RFQLeg{
						{Category: "option", Symbol: "BTC-25DEC26-100000-C", Side: "Buy", Quantity: 10},
					},
				},
			},
		},
	}
	assert.Equal(t, exp, relayedFrames(ex), "wsHandleRFQPrivateData should relay every frame")

	assert.ErrorIs(t, ex.wsHandleRFQPrivateData(t.Context(), []byte(`{"topic":"rfq.open.public.trades"}`)), errUnhandledStreamData, "wsHandleRFQPrivateData should reject an unknown topic")
	assert.Error(t, ex.wsHandleRFQPrivateData(t.Context(), []byte(`{"op":"pong"}`)), "wsHandleRFQPrivateData should reject a frame without a topic")
	assert.Error(t, ex.wsHandleRFQPrivateData(t.Context(), []byte(`{"topic":"rfq.open.quotes","data":{}}`)), "wsHandleRFQPrivateData should reject a malformed frame")
}
