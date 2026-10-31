package bybit

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	mockws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
	"github.com/thrasher-corp/gocryptotrader/types"
)

const advancedEarnOffersFixture = "testdata/wsAdvancedEarnOffers.jsonl"

var advancedEarnChannels = []string{
	AdvancedEarnDualAssetsOffersChannel, AdvancedEarnSmartLeverageOffersChannel, AdvancedEarnDiscountBuyOffersChannel, AdvancedEarnDoubleWinOffersChannel,
}

func TestAdvancedEarnConnectionSetup(t *testing.T) {
	t.Parallel()
	cfg := &config.Exchange{WebsocketResponseCheckTimeout: 30 * time.Millisecond, WebsocketResponseMaxLimit: 7 * time.Second}
	streamURL := "wss://stream-testnet.bybit.com" + advancedEarnPath
	setup := e.AdvancedEarnConnectionSetup(cfg, streamURL)
	require.NotNil(t, setup, "AdvancedEarnConnectionSetup must return a setup")
	assert.Equal(t, streamURL, setup.URL, "AdvancedEarnConnectionSetup should connect to the URL given")
	assert.Equal(t, cfg.WebsocketResponseCheckTimeout, setup.ResponseCheckTimeout, "AdvancedEarnConnectionSetup should use the configured response check timeout")
	assert.Equal(t, cfg.WebsocketResponseMaxLimit, setup.ResponseMaxLimit, "AdvancedEarnConnectionSetup should use the configured response limit")
	assert.Equal(t, AdvancedEarnOffersConnection, setup.MessageFilter, "AdvancedEarnConnectionSetup should set its message filter")
	assert.NotNil(t, setup.Connector, "AdvancedEarnConnectionSetup should set a connector")
	assert.NotNil(t, setup.GenerateSubscriptions, "AdvancedEarnConnectionSetup should set a subscription generator")
	assert.NotNil(t, setup.Subscriber, "AdvancedEarnConnectionSetup should set a subscriber")
	assert.NotNil(t, setup.Unsubscriber, "AdvancedEarnConnectionSetup should set an unsubscriber")
	assert.NotNil(t, setup.Handler, "AdvancedEarnConnectionSetup should set a handler")

	var dials atomic.Int32
	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		dials.Add(1)
		w.WriteHeader(http.StatusBadRequest)
	}))
	server.Start()
	ex := newAdvancedEarnStreamInstance(t, "ws"+strings.TrimPrefix(server.URL, "http"))
	require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
	assert.Zero(t, dials.Load(), "Connect should not dial the Advanced Earn stream when no Advanced Earn channel is configured")
}

func TestGenerateAdvancedEarnSubscriptions(t *testing.T) {
	t.Parallel()
	ex := newAdvancedEarnTestExchange(t)
	subs, err := ex.generateAdvancedEarnSubscriptions()
	require.NoError(t, err, "generateAdvancedEarnSubscriptions must not error")
	assert.Empty(t, subs, "generateAdvancedEarnSubscriptions should generate nothing by default")

	ex.Features.Subscriptions = subscription.List{
		{Enabled: true, Channel: AdvancedEarnDualAssetsOffersChannel},
		{Enabled: false, Channel: AdvancedEarnSmartLeverageOffersChannel},
		{Enabled: true, Channel: subscription.TickerChannel, Asset: asset.Spot},
		{Enabled: true, Channel: AdvancedEarnDoubleWinOffersChannel},
	}
	subs, err = ex.generateAdvancedEarnSubscriptions()
	require.NoError(t, err, "generateAdvancedEarnSubscriptions must not error")
	exp := subscription.List{
		{Enabled: true, Channel: AdvancedEarnDualAssetsOffersChannel},
		{Enabled: true, Channel: AdvancedEarnDoubleWinOffersChannel},
	}
	assert.Equal(t, exp, subs, "generateAdvancedEarnSubscriptions should generate the enabled Advanced Earn channels")
	require.Len(t, subs, 2, "generateAdvancedEarnSubscriptions must generate two subscriptions")
	assert.NotSame(t, ex.Features.Subscriptions[0], subs[0], "generateAdvancedEarnSubscriptions should clone the configured subscriptions")
}

func TestIsAdvancedEarnChannel(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		channel string
		exp     bool
	}{
		{channel: "earn.dualassets.offers", exp: true},
		{channel: "earn.smartleverage.offers", exp: true},
		{channel: "earn.discountbuy.offers", exp: true},
		{channel: "earn.doublewin.offers", exp: true},
		{channel: "earn.sharkfin.offers"},
		{channel: subscription.TickerChannel},
		{channel: ""},
	} {
		assert.Equalf(t, tc.exp, isAdvancedEarnChannel(tc.channel), "isAdvancedEarnChannel should report whether the fp stream serves %q", tc.channel)
	}
}

func TestWebsocketHandleAdvancedEarnData(t *testing.T) {
	t.Parallel()
	ex := newAdvancedEarnTestExchange(t)
	testexch.FixtureToDataHandler(t, advancedEarnOffersFixture, func(ctx context.Context, b []byte) error {
		return ex.wsHandleAdvancedEarnData(ctx, nil, b)
	})
	require.Len(t, ex.Websocket.DataHandler.C, 4, "wsHandleAdvancedEarnData must relay each topic's offers")
	relayed := make([]any, 0, 4)
	for range 4 {
		relayed = append(relayed, (<-ex.Websocket.DataHandler.C).Data)
	}
	assert.Equal(t, advancedEarnOffersExp(), relayed, "wsHandleAdvancedEarnData should relay every decoded field")

	conn := &advancedEarnMatchConnection{match: websocket.NewMatch()}
	replies, err := conn.match.Set("2f9d3c1e", 1)
	require.NoError(t, err, "match.Set must not error")
	reply := []byte(`{"req_id":"2f9d3c1e","success":true,"ret_msg":"","op":"subscribe","conn_id":"d7k2mock"}`)
	require.NoError(t, ex.wsHandleAdvancedEarnData(t.Context(), conn, reply), "wsHandleAdvancedEarnData must route a subscription reply")
	require.Len(t, replies, 1, "wsHandleAdvancedEarnData must deliver the reply to its request")
	assert.Equal(t, reply, <-replies, "wsHandleAdvancedEarnData should deliver the reply to its request")

	for _, tc := range []struct {
		name  string
		frame string
		err   error
	}{
		{name: "pong", frame: `{"op":"pong","args":["1791534543114"],"conn_id":"d7k2mock"}`},
		{name: "unrequested reply", frame: `{"req_id":"9a8b7c6d","success":true,"ret_msg":"","op":"unsubscribe","conn_id":"d7k2mock"}`, err: websocket.ErrSignatureNotMatched},
		{name: "unknown operation", frame: `{"op":"auth","success":true}`, err: errUnhandledAdvancedEarnMessage},
		{name: "unknown topic", frame: `{"topic":"earn.sharkfin.offers","data":[]}`, err: errUnhandledAdvancedEarnMessage},
		{name: "null", frame: `null`, err: errUnhandledAdvancedEarnMessage},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := ex.wsHandleAdvancedEarnData(t.Context(), &advancedEarnMatchConnection{match: websocket.NewMatch()}, []byte(tc.frame))
			assert.ErrorIs(t, err, tc.err, "wsHandleAdvancedEarnData should handle the frame")
		})
	}
	assert.Error(t, ex.wsHandleAdvancedEarnData(t.Context(), nil, []byte(`not json`)), "wsHandleAdvancedEarnData should error on a frame that is not JSON")
	for _, channel := range advancedEarnChannels {
		frame := []byte(`{"topic":"` + channel + `","data":{}}`)
		assert.Errorf(t, ex.wsHandleAdvancedEarnData(t.Context(), nil, frame), "wsHandleAdvancedEarnData should error on %s offers that do not decode", channel)
	}
}

func TestAdvancedEarnOffersStream(t *testing.T) {
	t.Parallel()
	url := mainnetStreamOrigin + advancedEarnPath
	if mockTests {
		server := httptest.NewTestServer(t, mockws.CurryWsMockUpgrader(t, advancedEarnMockStream))
		server.Start()
		url = "ws" + strings.TrimPrefix(server.URL, "http")
	}
	ex := newAdvancedEarnStreamInstance(t, url, advancedEarnChannels...)
	require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
	subs := ex.Websocket.GetSubscriptions()
	assert.ElementsMatch(t, advancedEarnChannels, advancedEarnSubscriptionChannels(subs), "Connect should subscribe to every configured Advanced Earn channel")
	for _, s := range subs {
		assert.Equalf(t, subscription.SubscribedState, s.State(), "Connect should mark %s subscribed", s.Channel)
	}

	offers := collectAdvancedEarnOffers(t, ex)
	if mockTests {
		assert.Equal(t, advancedEarnOffersExp(), offers, "the stream's offers should decode every field")
	} else {
		for i, o := range offers {
			assert.NotEmptyf(t, o, "the stream should relay %s offers", advancedEarnChannels[i])
		}
	}

	conn, err := ex.Websocket.GetConnection(AdvancedEarnOffersConnection)
	require.NoError(t, err, "GetConnection must not error")
	require.NoError(t, ex.AdvancedEarnUnsubscribe(t.Context(), conn, subs), "AdvancedEarnUnsubscribe must not error")
	assert.Empty(t, ex.Websocket.GetSubscriptions(), "AdvancedEarnUnsubscribe should remove every subscription")

	err = ex.AdvancedEarnSubscribe(t.Context(), conn, subscription.List{
		{Enabled: true, Channel: "earn.sharkfin.offers"},
		{Enabled: true, Channel: AdvancedEarnDoubleWinOffersChannel},
	})
	require.ErrorIs(t, err, errAdvancedEarnSubscriptionRejected, "AdvancedEarnSubscribe must report a rejected topic")
	assert.Equal(t, []string{AdvancedEarnDoubleWinOffersChannel}, advancedEarnSubscriptionChannels(ex.Websocket.GetSubscriptions()), "AdvancedEarnSubscribe should still subscribe to the topics after a rejected one")
}

func TestManageAdvancedEarnSubscriptions(t *testing.T) {
	t.Parallel()
	ex := newAdvancedEarnTestExchange(t)
	dualAssets := &subscription.Subscription{Enabled: true, Channel: AdvancedEarnDualAssetsOffersChannel}
	sharkFin := &subscription.Subscription{Enabled: true, Channel: "earn.sharkfin.offers"}
	discountBuy := &subscription.Subscription{Enabled: true, Channel: AdvancedEarnDiscountBuyOffersChannel}
	doubleWin := &subscription.Subscription{Enabled: true, Channel: AdvancedEarnDoubleWinOffersChannel}

	conn := &FixtureConnection{reply: advancedEarnReply}
	err := ex.manageAdvancedEarnSubscriptions(t.Context(), conn, "subscribe", subscription.List{dualAssets, sharkFin, doubleWin})
	require.ErrorIs(t, err, errAdvancedEarnSubscriptionRejected, "manageAdvancedEarnSubscriptions must report the rejected topic")
	assert.ErrorContains(t, err, "subscribe earn.sharkfin.offers", "manageAdvancedEarnSubscriptions should name the rejected topic")
	assert.Equal(t, [][]string{{AdvancedEarnDualAssetsOffersChannel}, {"earn.sharkfin.offers"}, {AdvancedEarnDoubleWinOffersChannel}}, advancedEarnRequestTopics(t, conn, "subscribe"), "manageAdvancedEarnSubscriptions should send each topic in a request of its own")
	assert.ElementsMatch(t, []string{AdvancedEarnDualAssetsOffersChannel, AdvancedEarnDoubleWinOffersChannel}, advancedEarnSubscriptionChannels(ex.Websocket.GetSubscriptions()), "manageAdvancedEarnSubscriptions should store the accepted topics, including those after a rejected one")
	assert.Equal(t, subscription.SubscribedState, dualAssets.State(), "manageAdvancedEarnSubscriptions should mark an accepted topic subscribed")

	conn = &FixtureConnection{reply: advancedEarnReply}
	require.NoError(t, ex.manageAdvancedEarnSubscriptions(t.Context(), conn, "unsubscribe", subscription.List{dualAssets}), "manageAdvancedEarnSubscriptions must not error when unsubscribing")
	assert.Equal(t, [][]string{{AdvancedEarnDualAssetsOffersChannel}}, advancedEarnRequestTopics(t, conn, "unsubscribe"), "manageAdvancedEarnSubscriptions should send an unsubscribe request")
	assert.Equal(t, []string{AdvancedEarnDoubleWinOffersChannel}, advancedEarnSubscriptionChannels(ex.Websocket.GetSubscriptions()), "manageAdvancedEarnSubscriptions should remove an unsubscribed topic")

	conn = &FixtureConnection{reply: replySequence("", `{"req_id":"9a8b7c6d","success":true,"ret_msg":"","op":"subscribe","conn_id":"d7k2mock"}`)}
	err = ex.manageAdvancedEarnSubscriptions(t.Context(), conn, "subscribe", subscription.List{discountBuy, dualAssets})
	require.ErrorIs(t, err, errFixtureSend, "manageAdvancedEarnSubscriptions must report a failed send")
	assert.ErrorContains(t, err, "subscribe earn.discountbuy.offers", "manageAdvancedEarnSubscriptions should name the topic it could not send")
	assert.ElementsMatch(t, []string{AdvancedEarnDualAssetsOffersChannel, AdvancedEarnDoubleWinOffersChannel}, advancedEarnSubscriptionChannels(ex.Websocket.GetSubscriptions()), "manageAdvancedEarnSubscriptions should carry on past a failed send")

	err = ex.manageAdvancedEarnSubscriptions(t.Context(), &FixtureConnection{reply: advancedEarnReply}, "subscribe", subscription.List{doubleWin})
	require.ErrorIs(t, err, subscription.ErrDuplicate, "manageAdvancedEarnSubscriptions must report a subscription it holds already")
}

func TestSendAdvancedEarnSubscription(t *testing.T) {
	t.Parallel()
	ex := newAdvancedEarnTestExchange(t)
	req := &AdvancedEarnSubscriptionRequest{RequestID: "2f9d3c1e", Operation: "subscribe", Arguments: []string{AdvancedEarnDualAssetsOffersChannel}}
	for _, tc := range []struct {
		name  string
		reply string // an empty reply fails the send
		err   error
	}{
		{name: "accepted", reply: `{"req_id":"2f9d3c1e","success":true,"ret_msg":"","op":"subscribe","conn_id":"d7k2mock"}`},
		{name: "rejected", reply: `{"req_id":"2f9d3c1e","success":false,"ret_msg":"Params Error","op":"subscribe","conn_id":"d7k2mock"}`, err: errAdvancedEarnSubscriptionRejected},
		{name: "null", reply: `null`, err: common.ErrNoResponse},
		{name: "failed send", err: errFixtureSend},
	} {
		conn := &signatureConnection{FixtureConnection: &FixtureConnection{reply: replySequence(tc.reply)}}
		err := ex.sendAdvancedEarnSubscription(t.Context(), conn, req)
		require.ErrorIsf(t, err, tc.err, "sendAdvancedEarnSubscription must return the expected error (%s)", tc.name)
		assert.Equalf(t, []any{req}, conn.sent, "sendAdvancedEarnSubscription should send the request (%s)", tc.name)
		assert.Equalf(t, []any{"2f9d3c1e"}, conn.signatures, "sendAdvancedEarnSubscription should wait on the request ID, which the reply echoes (%s)", tc.name)
	}
	err := ex.sendAdvancedEarnSubscription(t.Context(), &FixtureConnection{reply: replySequence(`{"req_id":"2f9d3c1e","success":false,"ret_msg":"Params Error","op":"subscribe","conn_id":"d7k2mock"}`)}, req)
	assert.ErrorContains(t, err, "Params Error", "sendAdvancedEarnSubscription should carry the stream's reason for a rejection")
	err = ex.sendAdvancedEarnSubscription(t.Context(), &FixtureConnection{reply: replySequence(`{"success":"maybe"}`)}, req)
	require.Error(t, err, "sendAdvancedEarnSubscription must error for a reply that does not decode")
}

// advancedEarnReply answers an Advanced Earn subscription request as the fp stream does: it echoes the request ID and
// rejects a topic it does not serve with Params Error
func advancedEarnReply(r any) ([]byte, error) {
	req, ok := r.(*AdvancedEarnSubscriptionRequest)
	if !ok {
		return nil, errUnexpectedRequest
	}
	known := len(req.Arguments) == 1 && slices.Contains(advancedEarnChannels, req.Arguments[0])
	retMsg := ""
	if !known {
		retMsg = "Params Error"
	}
	return fmt.Appendf(nil, `{"req_id":%q,"success":%t,"ret_msg":%q,"op":%q,"conn_id":"d7k2mock"}`, req.RequestID, known, retMsg, req.Operation), nil
}

// advancedEarnRequestTopics returns the topics of each Advanced Earn request sent on conn, which must all be for
// operation
func advancedEarnRequestTopics(t *testing.T, conn *FixtureConnection, operation string) [][]string {
	t.Helper()
	topics := make([][]string, len(conn.sent))
	for i, r := range conn.sent {
		req, ok := r.(*AdvancedEarnSubscriptionRequest)
		require.True(t, ok, "conn.sent must hold AdvancedEarnSubscriptionRequests")
		require.Equal(t, operation, req.Operation, "req.Operation must match the operation")
		require.NotEmpty(t, req.RequestID, "req.RequestID must be set for the reply to match")
		topics[i] = req.Arguments
	}
	return topics
}

// newAdvancedEarnTestExchange returns an exchange of its own, for tests that change its websocket or subscriptions
func newAdvancedEarnTestExchange(t *testing.T) *Exchange {
	t.Helper()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	return ex
}

// newAdvancedEarnStreamInstance returns an exchange whose websocket holds only the Advanced Earn offers connection,
// registered as Setup registers it but pointed at url, with channels configured
func newAdvancedEarnStreamInstance(t *testing.T, url string, channels ...string) *Exchange {
	t.Helper()
	ex := newAdvancedEarnTestExchange(t)
	ex.Websocket = websocket.NewManager()
	require.NoError(t, ex.Websocket.Setup(&websocket.ManagerSetup{
		ExchangeConfig:               ex.Config,
		Features:                     &ex.Features.Supports.WebsocketCapabilities,
		UseMultiConnectionManagement: true,
	}), "Websocket.Setup must not error")
	setup := ex.AdvancedEarnConnectionSetup(ex.Config, url)
	require.NoError(t, ex.Websocket.SetupNewConnection(setup), "SetupNewConnection must not error")
	ex.Features.Subscriptions = make(subscription.List, len(channels))
	for i, c := range channels {
		ex.Features.Subscriptions[i] = &subscription.Subscription{Enabled: true, Channel: c}
	}
	t.Cleanup(func() {
		if ex.Websocket.IsConnected() {
			assert.NoError(t, ex.Websocket.Shutdown(), "Websocket.Shutdown should not error")
		}
	})
	return ex
}

// advancedEarnMockStream answers like the fp stream: a topic it does not know is rejected with Params Error, a pong
// answers a ping, and each subscription is followed by the topic's recorded push
func advancedEarnMockStream(tb testing.TB, msg []byte, c *gws.Conn) error {
	tb.Helper()
	var req AdvancedEarnSubscriptionRequest
	if err := json.Unmarshal(msg, &req); err != nil {
		return err
	}
	if req.Operation == "ping" {
		return c.WriteMessage(gws.TextMessage, []byte(`{"op":"pong","args":["1791534543114"],"conn_id":"d7k2mock"}`))
	}
	if len(req.Arguments) != 1 {
		return fmt.Errorf("each request should carry one topic, not %q", req.Arguments)
	}
	push, known := advancedEarnRecordedPushes(tb)[req.Arguments[0]]
	retMsg := ""
	if !known {
		retMsg = "Params Error"
	}
	reply := fmt.Sprintf(`{"req_id":%q,"success":%t,"ret_msg":%q,"op":%q,"conn_id":"d7k2mock"}`, req.RequestID, known, retMsg, req.Operation)
	if err := c.WriteMessage(gws.TextMessage, []byte(reply)); err != nil {
		return err
	}
	if known && req.Operation == "subscribe" {
		return c.WriteMessage(gws.TextMessage, push)
	}
	return nil
}

// advancedEarnRecordedPushes returns the recorded push of each topic
func advancedEarnRecordedPushes(tb testing.TB) map[string][]byte {
	tb.Helper()
	f, err := os.Open(advancedEarnOffersFixture)
	require.NoError(tb, err, "Opening the offers fixture must not error")
	defer f.Close()
	pushes := make(map[string][]byte)
	s := bufio.NewScanner(f)
	for s.Scan() {
		var msg WebsocketAdvancedEarnResponse
		require.NoError(tb, json.Unmarshal(s.Bytes(), &msg), "Unmarshal must not error")
		pushes[msg.Topic] = slices.Clone(s.Bytes())
	}
	require.NoError(tb, s.Err(), "Scanning the offers fixture must not error")
	return pushes
}

// collectAdvancedEarnOffers returns the first offers relayed for each topic, in the order of advancedEarnChannels
func collectAdvancedEarnOffers(t *testing.T, ex *Exchange) []any {
	t.Helper()
	offers := make([]any, len(advancedEarnChannels))
	timeout := time.After(time.Minute)
	for slices.Contains(offers, nil) {
		select {
		case payload := <-ex.Websocket.DataHandler.C:
			var i int
			switch data := payload.Data.(type) {
			case []DualAssetsOffer:
				i = 0
			case []SmartLeverageOffer:
				i = 1
			case []DiscountBuyOffer:
				i = 2
			case []DoubleWinOffer:
				i = 3
			case error:
				require.NoError(t, data, "the stream must not relay errors")
			default:
				continue
			}
			if offers[i] == nil {
				offers[i] = payload.Data
			}
		case <-timeout:
			require.Fail(t, "the stream must relay offers for every topic within a minute")
		}
	}
	return offers
}

// advancedEarnSubscriptionChannels returns the channels of subs
func advancedEarnSubscriptionChannels(subs subscription.List) []string {
	channels := make([]string, len(subs))
	for i, s := range subs {
		channels[i] = s.Channel
	}
	return channels
}

// advancedEarnMatchConnection answers RequireMatchWithData from a match, standing in for a connection
type advancedEarnMatchConnection struct {
	websocket.Connection
	match *websocket.Match
}

func (c *advancedEarnMatchConnection) RequireMatchWithData(signature any, data []byte) error {
	return c.match.RequireMatchWithData(signature, data)
}

// advancedEarnOffersExp is what the recorded pushes in advancedEarnOffersFixture decode to
func advancedEarnOffersExp() []any {
	return []any{
		[]DualAssetsOffer{
			{
				ProductID:    "155494",
				CurrentPrice: 0.2392,
				BuyLowPrice: []DualAssetsOfferPrice{
					{
						SelectPrice:         0.235,
						APYE8:               222173550,
						MaxInvestmentAmount: 5000000,
						ExpiredAt:           types.Time(time.Unix(1791534555, 0)),
					},
					{
						SelectPrice:         0.23,
						APYE8:               164145150,
						MaxInvestmentAmount: 5000000,
						ExpiredAt:           types.Time(time.Unix(1791534555, 0)),
					},
				},
				SellHighPrice: []DualAssetsOfferPrice{
					{
						SelectPrice:         0.25,
						APYE8:               145787400,
						MaxInvestmentAmount: 20903010.0334,
						ExpiredAt:           types.Time(time.Unix(1791534555, 0)),
					},
				},
			},
			{
				ProductID:    "155477",
				CurrentPrice: 12.867,
				BuyLowPrice: []DualAssetsOfferPrice{
					{
						SelectPrice:         12.882,
						APYE8:               447660450,
						MaxInvestmentAmount: 5000000,
						ExpiredAt:           types.Time(time.Unix(1791534560, 0)),
					},
				},
				SellHighPrice: []DualAssetsOfferPrice{
					{
						SelectPrice:         12.883,
						APYE8:               448376850,
						MaxInvestmentAmount: 388078.2365,
						ExpiredAt:           types.Time(time.Unix(1791534560, 0)),
					},
				},
			},
		},
		[]SmartLeverageOffer{
			{
				ProductID:           "69219",
				BreakevenPrice:      2520.827409,
				ExpireAt:            types.Time(time.UnixMilli(1791534553126)),
				MaxInvestmentAmount: 9500,
				CurrentPrice:        2500.16,
			},
			{
				ProductID:           "69220",
				BreakevenPrice:      2479.741556,
				ExpireAt:            types.Time(time.UnixMilli(1791534562932)),
				MaxInvestmentAmount: 9500,
				CurrentPrice:        2500.16,
			},
		},
		[]DiscountBuyOffer{
			{
				ProductID:           "15927",
				CurrentPrice:        82605.07,
				PurchasePrice:       82425,
				KnockoutPrice:       83400,
				KnockoutCouponE8:    10000000,
				MaxInvestmentAmount: 950000,
				InstitutionUID:      "103489231",
				ExpiredAt:           types.Time(time.UnixMilli(1791534560467)),
			},
			{
				ProductID:           "15928",
				CurrentPrice:        2500.15,
				PurchasePrice:       2492,
				KnockoutPrice:       2530,
				KnockoutCouponE8:    10000000,
				MaxInvestmentAmount: 950000,
				InstitutionUID:      "103489231",
				ExpiredAt:           types.Time(time.UnixMilli(1791534563700)),
			},
		},
		[]DoubleWinOffer{
			{
				ProductID:           "45907",
				CurrentPrice:        110.264,
				Leverage:            49.99,
				MaxInvestmentAmount: 1900,
				ExpireTime:          types.Time(time.UnixMilli(1791534562846)),
			},
			{
				ProductID:           "45922",
				CurrentPrice:        1.07331,
				Leverage:            10.46,
				MaxInvestmentAmount: 2722.4231263,
				ExpireTime:          types.Time(time.UnixMilli(1791534553164)),
			},
		},
	}
}
