package kraken

import (
	"context"
	"encoding/base64"
	"net/http"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/stream"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	testsubs "github.com/thrasher-corp/gocryptotrader/internal/testing/subscriptions"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// futuresTestChallenge is the challenge the mock futures server issues
const futuresTestChallenge = "226aee50-88fc-4618-a42a-34f7709570b2"

// mockFuturesServer answers futures requests as Kraken does: a challenge with futuresTestChallenge, and a subscription
// or unsubscription with one reply for each product, or one for a feed without products, in the order requested. It
// records each request
type mockFuturesServer struct {
	m        sync.Mutex
	requests []string
	// alerts maps a product, or a feed without products, to the alert its request is answered with
	alerts map[string]string
	// challengeReply replaces the reply to a challenge request
	challengeReply string
	// reply replaces each reply to a subscription or unsubscription
	reply string
}

func (s *mockFuturesServer) handle(_ testing.TB, msg []byte, w *gws.Conn) error {
	var req wsFuturesRequest
	if err := json.Unmarshal(msg, &req); err != nil {
		return err
	}
	s.m.Lock()
	s.requests = append(s.requests, string(msg))
	alerts, challengeReply, rawReply := s.alerts, s.challengeReply, s.reply
	s.m.Unlock()
	reply := func(v map[string]any) error {
		b, err := json.Marshal(v)
		if err != nil {
			return err
		}
		return w.WriteMessage(gws.TextMessage, b)
	}
	if req.Event == wsFuturesEventChallenge {
		if challengeReply != "" {
			return w.WriteMessage(gws.TextMessage, []byte(challengeReply))
		}
		return reply(map[string]any{"event": wsFuturesEventChallenge, "message": futuresTestChallenge})
	}
	targets := req.ProductIDs
	if len(targets) == 0 {
		targets = []string{req.Feed}
	}
	for _, target := range targets {
		if rawReply != "" {
			if err := w.WriteMessage(gws.TextMessage, []byte(rawReply)); err != nil {
				return err
			}
			continue
		}
		if alert, ok := alerts[target]; ok {
			if err := reply(map[string]any{"event": wsFuturesEventAlert, "message": alert}); err != nil {
				return err
			}
			continue
		}
		ack := map[string]any{"event": req.Event + "d", "feed": req.Feed}
		if len(req.ProductIDs) != 0 {
			ack["product_ids"] = []string{target}
		}
		if req.SignedChallenge != "" {
			ack["api_key"], ack["original_challenge"], ack["signed_challenge"] = req.APIKey, req.OriginalChallenge, req.SignedChallenge
		}
		if err := reply(ack); err != nil {
			return err
		}
	}
	return nil
}

// takeRequests returns and forgets the requests received
func (s *mockFuturesServer) takeRequests() []string {
	s.m.Lock()
	defer s.m.Unlock()
	r := s.requests
	s.requests = nil
	return r
}

// newFuturesTestExchange returns an exchange with test credentials whose futures connection, connected without
// subscriptions, is served by server
func newFuturesTestExchange(t *testing.T, server *mockFuturesServer) (*Exchange, websocket.Connection) {
	t.Helper()
	ex := newTestExchange(t)
	useTestCredentials(ex)
	ex.Features.Subscriptions = subscription.List{}
	useTestWebsocket(t, ex, newMockWebsocketURL(t, server.handle), wsFuturesConnection)
	ex.Websocket.SetSubscriptionsNotRequired()
	require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
	conn, err := ex.Websocket.GetConnection(wsFuturesConnection)
	require.NoError(t, err, "GetConnection must not error")
	return ex, conn
}

// futuresTestSignedChallenge returns futuresTestChallenge signed with the test credentials' secret
func futuresTestSignedChallenge(t *testing.T) string {
	t.Helper()
	signed, err := futuresChallengeSignature(decodedTestSecret(t), futuresTestChallenge)
	require.NoError(t, err, "futuresChallengeSignature must not error")
	return signed
}

// futuresSubscription returns a futures subscription to a feed, as template expansion gives it
func futuresSubscription(channel, feed string, pairs ...currency.Pair) *subscription.Subscription {
	return &subscription.Subscription{Enabled: true, Asset: asset.Futures, Channel: channel, QualifiedChannel: feed, Pairs: pairs, Authenticated: slices.Contains(futuresAccountFeeds, feed)}
}

func TestFuturesChallengeSignature(t *testing.T) {
	t.Parallel()
	// Kraken's documented example
	secret, err := base64.StdEncoding.DecodeString("7zxMEF5p/Z8l2p2U7Ghv6x14Af+Fx+92tPgUdVQ748FOIrEoT9bgT+bTRfXc5pz8na+hL/QdrCVG7bh9KpT0eMTm")
	require.NoError(t, err, "DecodeString must not error")
	signed, err := futuresChallengeSignature(string(secret), "c100b894-1729-464d-ace1-52dbce11db42")
	require.NoError(t, err, "futuresChallengeSignature must not error")
	assert.Equal(t, "4JEpF3ix66GA2B+ooK128Ift4XQVtc137N9yeg4Kqsn9PI0Kpzbysl9M1IeCEdjg0zl00wkVqcsnG4bmnlMb3A==", signed, "futuresChallengeSignature should match Kraken's documented example")
}

func TestGenerateFuturesSubscriptions(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	require.NoError(t, ex.CurrencyPairs.EnablePair(asset.Futures, piXBTUSDPair), "EnablePair must not error")
	ex.Features.Subscriptions = append(defaultSubscriptions.Clone(), &subscription.Subscription{Enabled: true, Asset: asset.Futures, Channel: subscription.HeartbeatChannel})
	public := subscription.List{
		futuresSubscription(subscription.TickerChannel, wsFuturesFeedTicker, futuresTestPair),
		futuresSubscription(subscription.TickerChannel, wsFuturesFeedTicker, piXBTUSDPair),
		futuresSubscription(subscription.AllTradesChannel, wsFuturesFeedTrade, futuresTestPair),
		futuresSubscription(subscription.AllTradesChannel, wsFuturesFeedTrade, piXBTUSDPair),
		futuresSubscription(subscription.OrderbookChannel, wsFuturesFeedBook, futuresTestPair),
		futuresSubscription(subscription.OrderbookChannel, wsFuturesFeedBook, piXBTUSDPair),
		// A feed without products is subscribed to once
		futuresSubscription(subscription.HeartbeatChannel, wsFuturesFeedHeartbeat),
	}
	ex.Websocket.SetCanUseAuthenticatedEndpoints(false)
	got, err := ex.generateFuturesSubscriptions()
	require.NoError(t, err, "generateFuturesSubscriptions must not error")
	testsubs.EqualLists(t, public, got)

	useTestCredentials(ex)
	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
	got, err = ex.generateFuturesSubscriptions()
	require.NoError(t, err, "generateFuturesSubscriptions must not error with credentials")
	exp := append(public.Clone(),
		futuresSubscription(subscription.MyOrdersChannel, wsFuturesFeedOpenOrders),
		futuresSubscription(subscription.MyTradesChannel, wsFuturesFeedFills),
	)
	testsubs.EqualLists(t, exp, got)
	for _, s := range ex.Features.Subscriptions {
		assert.Emptyf(t, s.Pairs, "generateFuturesSubscriptions should not change the configured %s subscription", s.Channel)
	}
}

func TestValidateFuturesSubscriptions(t *testing.T) {
	t.Parallel()
	valid := subscription.List{
		futuresSubscription(subscription.TickerChannel, wsFuturesFeedTicker, futuresTestPair),
		futuresSubscription(subscription.OrderbookChannel, wsFuturesFeedBook, futuresTestPair),
		futuresSubscription(subscription.AllTradesChannel, wsFuturesFeedTrade, futuresTestPair),
		futuresSubscription(subscription.MyOrdersChannel, wsFuturesFeedOpenOrders),
		futuresSubscription(subscription.MyTradesChannel, wsFuturesFeedFills),
		futuresSubscription(subscription.MyWalletChannel, wsFuturesFeedBalances),
		futuresSubscription(wsFuturesFeedOpenPositions, wsFuturesFeedOpenPositions),
		futuresSubscription(wsFuturesFeedNotifications, wsFuturesFeedNotifications),
		futuresSubscription(subscription.HeartbeatChannel, wsFuturesFeedHeartbeat),
	}
	assert.NoError(t, e.ValidateSubscriptions(valid), "ValidateSubscriptions should accept every feed Kraken streams")
	for _, s := range valid {
		assert.NoErrorf(t, validateFuturesSubscription(s), "validateFuturesSubscription should accept %s", s)
	}
	for _, tc := range []struct {
		s   *subscription.Subscription
		err error
	}{
		{futuresSubscription(subscription.CandlesChannel, subscription.CandlesChannel, futuresTestPair), errFuturesUnsupportedFeed},
		{futuresSubscription(subscription.AllOrdersChannel, subscription.AllOrdersChannel, futuresTestPair), errFuturesUnsupportedFeed},
		{futuresSubscription("ticker_lite", "ticker_lite", futuresTestPair), errFuturesUnsupportedFeed},
		{&subscription.Subscription{Asset: asset.Futures, Channel: subscription.OrderbookChannel, QualifiedChannel: wsFuturesFeedBook, Levels: 10, Pairs: currency.Pairs{futuresTestPair}}, errFuturesOrderbookLevels},
	} {
		assert.ErrorIsf(t, e.ValidateSubscriptions(subscription.List{tc.s}), tc.err, "ValidateSubscriptions should reject %s", tc.s)
		assert.ErrorIsf(t, validateFuturesSubscription(tc.s), tc.err, "validateFuturesSubscription should reject %s", tc.s)
	}
}

func TestFuturesSubscribe(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	server := new(mockFuturesServer)
	var url string
	if mockTests {
		url = newMockWebsocketURL(t, server.handle)
		useTestCredentials(ex)
	}
	useTestWebsocket(t, ex, url, wsFuturesConnection)
	// Feeds relay to the data handler, which is drained as the engine drains it, keeping what was not handled
	var errs []error
	var unhandled []websocket.UnhandledMessageWarning
	stop, drained := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(drained)
		for {
			select {
			case <-stop:
				return
			case p := <-ex.Websocket.DataHandler.C:
				switch data := p.Data.(type) {
				case error:
					errs = append(errs, data)
				case websocket.UnhandledMessageWarning:
					unhandled = append(unhandled, data)
				}
			}
		}
	}()
	stopDraining := sync.OnceFunc(func() {
		close(stop)
		<-drained
	})
	t.Cleanup(stopDraining)
	require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must subscribe without error")
	exp := subscription.List{
		futuresSubscription(subscription.TickerChannel, wsFuturesFeedTicker, futuresTestPair),
		futuresSubscription(subscription.AllTradesChannel, wsFuturesFeedTrade, futuresTestPair),
		futuresSubscription(subscription.OrderbookChannel, wsFuturesFeedBook, futuresTestPair),
	}
	// A live run subscribes to the account feeds only when the test config enables authenticated websocket use and holds
	// credentials
	if ex.Websocket.CanUseAuthenticatedEndpoints() && ex.AreCredentialsValid(t.Context()) {
		exp = append(exp, futuresSubscription(subscription.MyOrdersChannel, wsFuturesFeedOpenOrders), futuresSubscription(subscription.MyTradesChannel, wsFuturesFeedFills))
	}
	subs := ex.Websocket.GetSubscriptions()
	testsubs.EqualLists(t, exp, subs.Clone())
	for _, s := range subs {
		assert.Equalf(t, subscription.SubscribedState, s.State(), "%s should be subscribed", s)
	}
	if !mockTests {
		require.Eventually(t, func() bool {
			_, err := ticker.GetTicker(ex.Name, futuresTestPair, asset.Futures)
			return err == nil
		}, 30*time.Second, 100*time.Millisecond, "the ticker feed must store the futures ticker")
		require.Eventually(t, func() bool {
			_, err := ex.Websocket.Orderbook.GetOrderbook(futuresTestPair, asset.Futures)
			return err == nil
		}, 30*time.Second, 100*time.Millisecond, "the book feed must store the futures order book")
	}
	signed := futuresTestSignedChallenge(t)
	if mockTests {
		assertRequests(t, []string{
			`{"event":"subscribe","feed":"ticker","product_ids":["PF_XBTUSD"]}`,
			`{"event":"subscribe","feed":"trade","product_ids":["PF_XBTUSD"]}`,
			`{"event":"subscribe","feed":"book","product_ids":["PF_XBTUSD"]}`,
			`{"event":"challenge","api_key":"test-key"}`,
			`{"event":"subscribe","feed":"open_orders","api_key":"test-key","original_challenge":"` + futuresTestChallenge + `","signed_challenge":"` + signed + `"}`,
			`{"event":"subscribe","feed":"fills","api_key":"test-key","original_challenge":"` + futuresTestChallenge + `","signed_challenge":"` + signed + `"}`,
		}, server.takeRequests())
	}

	conn, err := ex.Websocket.GetConnection(wsFuturesConnection)
	require.NoError(t, err, "GetConnection must not error")
	require.NoError(t, ex.Websocket.UnsubscribeChannels(t.Context(), conn, subs), "UnsubscribeChannels must not error")
	assert.Empty(t, ex.Websocket.GetSubscriptions(), "UnsubscribeChannels should remove every subscription")
	stopDraining()
	assert.Empty(t, errs, "futures messages should be handled without error")
	assert.Empty(t, unhandled, "futures messages should all be handled")
	if mockTests {
		// The challenge issued on the connection is reused
		assertRequests(t, []string{
			`{"event":"unsubscribe","feed":"ticker","product_ids":["PF_XBTUSD"]}`,
			`{"event":"unsubscribe","feed":"trade","product_ids":["PF_XBTUSD"]}`,
			`{"event":"unsubscribe","feed":"book","product_ids":["PF_XBTUSD"]}`,
			`{"event":"unsubscribe","feed":"open_orders","api_key":"test-key","original_challenge":"` + futuresTestChallenge + `","signed_challenge":"` + signed + `"}`,
			`{"event":"unsubscribe","feed":"fills","api_key":"test-key","original_challenge":"` + futuresTestChallenge + `","signed_challenge":"` + signed + `"}`,
		}, server.takeRequests())
	}
}

func TestFuturesSubscribeAlerts(t *testing.T) {
	t.Parallel()
	piETHUSD := currency.NewPairWithDelimiter("PI", "ETHUSD", currency.UnderscoreDelimiter)
	server := &mockFuturesServer{alerts: map[string]string{"PI_XBTUSD": "Couldn't subscribe to invalid product `PI_XBTUSD`"}}
	ex, conn := newFuturesTestExchange(t, server)
	accepted := futuresSubscription(subscription.TickerChannel, wsFuturesFeedTicker, futuresTestPair)
	rejected := futuresSubscription(subscription.TickerChannel, wsFuturesFeedTicker, piXBTUSDPair)
	alsoAccepted := futuresSubscription(subscription.TickerChannel, wsFuturesFeedTicker, piETHUSD)
	err := ex.futuresSubscribe(t.Context(), conn, subscription.List{accepted, rejected, alsoAccepted})
	require.ErrorIs(t, err, errFuturesSubscriptionFailed, "a subscription Kraken alerts must error")
	assert.ErrorContains(t, err, "Couldn't subscribe to invalid product `PI_XBTUSD`", "the error should carry Kraken's alert")
	assertRequests(t, []string{`{"event":"subscribe","feed":"ticker","product_ids":["PF_XBTUSD","PI_XBTUSD","PI_ETHUSD"]}`}, server.takeRequests())
	assert.NotNil(t, ex.Websocket.GetSubscription(accepted), "the subscription acknowledged before the alert should be stored")
	assert.Nil(t, ex.Websocket.GetSubscription(rejected), "the alerted subscription should not be stored")
	assert.NotNil(t, ex.Websocket.GetSubscription(alsoAccepted), "the subscription acknowledged after the alert should be stored")

	// A request whose subscription is already as asked changes nothing, so it succeeds
	server.m.Lock()
	server.alerts = map[string]string{"PI_XBTUSD": wsFuturesAlreadySubscribed}
	server.m.Unlock()
	require.NoError(t, ex.futuresSubscribe(t.Context(), conn, subscription.List{rejected}), "futuresSubscribe must not error for a subscription Kraken already holds")
	assert.NotNil(t, ex.Websocket.GetSubscription(rejected), "a subscription Kraken already holds should be stored")
	server.m.Lock()
	server.alerts = map[string]string{"PI_XBTUSD": wsFuturesNotSubscribed}
	server.m.Unlock()
	require.NoError(t, ex.futuresUnsubscribe(t.Context(), conn, subscription.List{rejected}), "futuresUnsubscribe must not error for a subscription Kraken does not hold")
	assert.Nil(t, ex.Websocket.GetSubscription(rejected), "a subscription Kraken does not hold should be removed")

	server.m.Lock()
	server.alerts = map[string]string{"PF_XBTUSD": "Unsubscribe failed"}
	server.m.Unlock()
	err = ex.futuresUnsubscribe(t.Context(), conn, subscription.List{accepted})
	assert.ErrorIs(t, err, errFuturesSubscriptionFailed, "an unsubscription Kraken alerts should error")
	assert.NotNil(t, ex.Websocket.GetSubscription(accepted), "a subscription Kraken did not unsubscribe should be kept")
}

func TestFuturesSubscribeUnexpectedReply(t *testing.T) {
	t.Parallel()
	ex, conn := newFuturesTestExchange(t, &mockFuturesServer{reply: `{"event":"subscribed","feed":"ticker","product_ids":["PF_MOONUSD"]}`})
	s := futuresSubscription(subscription.TickerChannel, wsFuturesFeedTicker, futuresTestPair)
	err := ex.futuresSubscribe(t.Context(), conn, subscription.List{s})
	assert.ErrorIs(t, err, errFuturesUnexpectedReply, "an acknowledgement of a product not requested should error")
	assert.Nil(t, ex.Websocket.GetSubscription(s), "the subscription should not be stored")
}

func TestFuturesSubscribeChallenge(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, reply string
		err         error
	}{
		{"refused", `{"event":"error","message":"Json Error"}`, errFuturesSubscriptionFailed},
		{"empty", `{"event":"challenge","message":""}`, errFuturesChallengeEmpty},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			server := &mockFuturesServer{challengeReply: tc.reply}
			ex, conn := newFuturesTestExchange(t, server)
			s := futuresSubscription(subscription.MyOrdersChannel, wsFuturesFeedOpenOrders)
			err := ex.futuresSubscribe(t.Context(), conn, subscription.List{s})
			assert.ErrorIs(t, err, tc.err, "a challenge Kraken does not issue should fail the subscription")
			assertRequests(t, []string{`{"event":"challenge","api_key":"test-key"}`}, server.takeRequests())
			assert.Nil(t, ex.Websocket.GetSubscription(s), "the subscription should not be stored")
		})
	}
}

func TestWebsocketFuturesHandleData(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	seedTestAssetNames(ex)
	useTestCredentials(ex)
	ex.Features.Enabled.TradeFeed = true
	ex.Websocket.Trade.Setup(true, ex.Websocket.DataHandler)
	ex.Features.Enabled.FillsFeed = true
	ex.Websocket.Fills.Setup(true, ex.Websocket.DataHandler)
	conn, err := ex.Websocket.CreateTestConnection(wsFuturesConnection)
	require.NoError(t, err, "CreateTestConnection must not error")
	testexch.FixtureToDataHandler(t, "testdata/wsFutures.json", func(ctx context.Context, b []byte) error {
		return ex.wsFuturesHandleData(ctx, conn, b)
	})
	got := relayed(ex)
	require.Len(t, got, 24, "wsFuturesHandleData must relay every payload")

	expTicker := &ticker.Price{
		Last:                82382,
		Bid:                 82380,
		BidSize:             0.0003,
		Ask:                 82381,
		AskSize:             0.4671,
		BaseVolume:          7779.8361,
		QuoteVolume:         636625710.8256,
		Open:                82634,
		High:                83262,
		Low:                 80320,
		PercentChange24Hour: -0.3,
		OpenInterest:        2370.562,
		MarkPrice:           82380.4640336367,
		IndexPrice:          82367.95,
		Pair:                futuresTestPair,
		ExchangeName:        t.Name(),
		AssetType:           asset.Futures,
		LastUpdated:         time.UnixMilli(1791518715420),
	}
	assert.Equal(t, expTicker, got[0], "the ticker should be relayed")
	stored, err := ticker.GetTicker(t.Name(), futuresTestPair, asset.Futures)
	require.NoError(t, err, "GetTicker must not error")
	assert.Equal(t, expTicker, stored, "the ticker should be stored as it was relayed")

	for i := 1; i < 5; i++ {
		assert.IsTypef(t, &orderbook.Depth{}, got[i], "payload %d should be the order book", i)
	}
	book, err := ex.Websocket.Orderbook.GetOrderbook(futuresTestPair, asset.Futures)
	require.NoError(t, err, "GetOrderbook must not error")
	// The updates add bids below the snapshot's
	assert.Equal(t, orderbook.Levels{{Price: 82380, Amount: 0.0003}, {Price: 82379, Amount: 0.0012}, {Price: 82378, Amount: 0.0003}, {Price: 82343, Amount: 0.1228}, {Price: 81529, Amount: 1.5296}, {Price: 81235, Amount: 0.0031}}, book.Bids, "the bids should have the updates applied")
	assert.Equal(t, orderbook.Levels{{Price: 82381, Amount: 0.6051}, {Price: 82382, Amount: 0.4345}, {Price: 82383, Amount: 0.0243}}, book.Asks, "the asks should be the snapshot's")
	assert.Equal(t, time.UnixMilli(1791518718754), book.LastUpdated, "the book should be as of the last update")
	lastUpdateID, err := ex.Websocket.Orderbook.LastUpdateID(futuresTestPair, asset.Futures)
	require.NoError(t, err, "LastUpdateID must not error")
	assert.Equal(t, int64(173877394), lastUpdateID, "the book should follow the last update's sequence number")

	futuresTrade := func(uid string, price, amount float64, ms int64) trade.Data {
		return trade.Data{Exchange: t.Name(), CurrencyPair: futuresTestPair, AssetType: asset.Futures, TID: uid, Side: order.Sell, Price: price, Amount: amount, Timestamp: time.UnixMilli(ms)}
	}
	assert.Equal(t, []trade.Data{
		futuresTrade("8fc9d2b3-f541-4e2b-8e0c-11559e0cd62b", 82382, 0.0002, 1791518714883),
		futuresTrade("d4d20435-6841-4f2d-8df9-0cd6a4198172", 82382, 0.0001, 1791518714883),
	}, got[5], "the trade snapshot should be relayed")
	assert.Equal(t, []trade.Data{futuresTrade("4fba15f2-e9e2-4811-8016-2c8b5d1e4c6e", 82380, 0.0002, 1791518719257)}, got[6], "the trade should be relayed")

	snapshotTime := time.UnixMilli(1612275024153)
	expOrders := []any{
		&order.Detail{
			Exchange:        t.Name(),
			AssetType:       asset.Futures,
			OrderID:         "723ba95f-13b7-418b-8fcf-ab7ba6620555",
			Pair:            futuresTestPair,
			Side:            order.Sell,
			Type:            order.StopLimit,
			Status:          order.Open,
			Price:           34900,
			TriggerPrice:    13789,
			Amount:          1000,
			RemainingAmount: 1000,
			Date:            snapshotTime,
			LastUpdated:     snapshotTime,
		},
		&order.Detail{
			Exchange:        t.Name(),
			AssetType:       asset.Futures,
			OrderID:         "59302619-41d2-4f0b-941f-7e7914760ad3",
			Pair:            futuresTestPair,
			Side:            order.Sell,
			Type:            order.TrailingStop,
			Status:          order.Open,
			TriggerPrice:    3200.1,
			Amount:          12,
			RemainingAmount: 12,
			Date:            snapshotTime,
			LastUpdated:     time.UnixMilli(1612275178153),
		},
		&order.Detail{
			Exchange:        t.Name(),
			AssetType:       asset.Futures,
			OrderID:         "7a2f793e-26f3-4987-a938-56d296a11560",
			Pair:            futuresTestPair,
			Side:            order.Sell,
			Type:            order.Limit,
			Status:          order.Open,
			Price:           35058,
			Amount:          1000,
			RemainingAmount: 1000,
			Date:            time.UnixMilli(1612275209430),
			LastUpdated:     time.UnixMilli(1612275209430),
		},
		&order.Detail{
			Exchange:        t.Name(),
			AssetType:       asset.Futures,
			OrderID:         "59302619-41d2-4f0b-941f-7e7914760ad3",
			Pair:            futuresTestPair,
			Side:            order.Sell,
			Type:            order.Limit,
			Status:          order.Open,
			ReduceOnly:      true,
			Price:           10640,
			Amount:          304,
			RemainingAmount: 304,
			Date:            time.UnixMilli(1567702877410),
			LastUpdated:     time.UnixMilli(1567702877410),
		},
		// A cancelled order is named by its ID alone
		&order.Detail{Exchange: t.Name(), AssetType: asset.Futures, OrderID: "660c6b23-8007-48c1-a7c9-4893f4572e8c", Status: order.Cancelled},
	}
	for i := range expOrders {
		assert.Equalf(t, expOrders[i], got[7+i], "open orders payload %d should be relayed", i)
	}

	piETHUSD := currency.NewPairWithDelimiter("PI", "ETHUSD", currency.UnderscoreDelimiter)
	makerTime, takerTime := time.UnixMilli(1600256910739), time.UnixMilli(1600256945531)
	expFills := []any{
		&order.Detail{
			Exchange:    t.Name(),
			AssetType:   asset.Futures,
			OrderID:     "9e30258b-5a98-4002-968a-5b0e149bcfbf",
			Pair:        futuresTestPair,
			Side:        order.Buy,
			Type:        order.Limit,
			FeeAsset:    currency.XBT,
			LastUpdated: makerTime,
			Trades: []order.TradeHistory{{
				TID:       "cad76f07-814e-4dc6-8478-7867407b6bff",
				Price:     10937.5,
				Amount:    5000,
				Fee:       -0.00009142857,
				FeeAsset:  "XBT",
				Exchange:  t.Name(),
				Type:      order.Limit,
				Side:      order.Buy,
				Timestamp: makerTime,
				IsMaker:   true,
			}},
		},
		&order.Detail{
			Exchange:    t.Name(),
			AssetType:   asset.Futures,
			OrderID:     "7e60b6e8-e4c2-4ce8-bbd0-ef81e18b65bb",
			Pair:        piETHUSD,
			Side:        order.Buy,
			Type:        order.Limit,
			FeeAsset:    currency.ETH,
			LastUpdated: takerTime,
			Trades: []order.TradeHistory{{
				TID:       "b1aa44b2-4f2a-4031-999c-ae1175c91580",
				Price:     364.65,
				Amount:    5000,
				Fee:       0.00685588921,
				FeeAsset:  "ETH",
				Exchange:  t.Name(),
				Type:      order.Limit,
				Side:      order.Buy,
				Timestamp: takerTime,
			}},
		},
		[]fill.Data{
			{
				ID:           "cad76f07-814e-4dc6-8478-7867407b6bff",
				Timestamp:    makerTime,
				Exchange:     t.Name(),
				AssetType:    asset.Futures,
				CurrencyPair: futuresTestPair,
				Side:         order.Buy,
				OrderID:      "9e30258b-5a98-4002-968a-5b0e149bcfbf",
				TradeID:      "cad76f07-814e-4dc6-8478-7867407b6bff",
				Price:        10937.5,
				Amount:       5000,
			},
			{
				ID:           "b1aa44b2-4f2a-4031-999c-ae1175c91580",
				Timestamp:    takerTime,
				Exchange:     t.Name(),
				AssetType:    asset.Futures,
				CurrencyPair: piETHUSD,
				Side:         order.Buy,
				OrderID:      "7e60b6e8-e4c2-4ce8-bbd0-ef81e18b65bb",
				TradeID:      "b1aa44b2-4f2a-4031-999c-ae1175c91580",
				Price:        364.65,
				Amount:       5000,
			},
		},
	}
	for i := range expFills {
		assert.Equalf(t, expFills[i], got[12+i], "fills payload %d should be relayed", i)
	}

	// The balances snapshot stores and relays the cash and multi-collateral wallets, then relays the message
	cash := accounts.CurrencyBalances{}
	for code, amount := range map[string]float64{
		"USDT": 4997.5012493753, "XBT": 0.1285407184, "ETH": 1.8714395862, "LTC": 47.6462740614, "GBP": 3733.488646461,
		"USDC": 5001.00020004, "USD": 5000, "BCH": 16.8924625832, "EUR": 4459.070194683, "XRP": 7065.5399485629,
	} {
		c := currency.NewCode(code)
		cash[c] = accounts.Balance{Currency: c, Total: amount, Free: amount}
	}
	flex := accounts.CurrencyBalances{
		currency.USDT: {Currency: currency.USDT},
		currency.USD:  {Currency: currency.USD},
	}
	assert.Equal(t, accounts.SubAccounts{
		{ID: "cash", AssetType: asset.Futures, Balances: cash},
		{ID: "flex", AssetType: asset.Futures, Balances: flex},
	}, futuresTestSubAccounts(t, got[15]), "the balances snapshot's wallets should be relayed")
	assert.IsType(t, &WebsocketFuturesBalances{}, got[16], "the balances snapshot should be relayed")
	usd := accounts.CurrencyBalances{currency.USD: {Currency: currency.USD, Total: 5000, Free: 5000}}
	assert.Equal(t, accounts.SubAccounts{{ID: "cash", AssetType: asset.Futures, Balances: usd}}, futuresTestSubAccounts(t, got[17]), "the cash wallet update should be relayed")
	assert.IsType(t, &WebsocketFuturesBalances{}, got[18], "the cash wallet update should be relayed")
	assert.Equal(t, accounts.SubAccounts{{ID: "flex", AssetType: asset.Futures, Balances: usd}}, futuresTestSubAccounts(t, got[19]), "the multi-collateral wallet update should be relayed")
	assert.IsType(t, &WebsocketFuturesBalances{}, got[20], "the multi-collateral wallet update should be relayed")
	margin, ok := got[21].(*WebsocketFuturesBalances)
	require.True(t, ok, "a margin account update must be relayed as the message alone")
	assert.Equal(t, map[string]WebsocketFuturesMarginAccount{"F-XBT:USD": {
		Name:           "F-XBT:USD",
		Pair:           "XBT/USD",
		Unit:           "XBT",
		PortfolioValue: 0.1219368845,
		Balance:        0.1219368845,
		Available:      0.1219368845,
	}}, margin.Futures, "the margin account should be relayed")
	for _, tc := range []struct {
		subAccount string
		c          currency.Code
		exp        accounts.Balance
	}{
		{"cash", currency.USD, accounts.Balance{Currency: currency.USD, Total: 5000, Free: 5000}},
		{"cash", currency.XBT, accounts.Balance{Currency: currency.XBT, Total: 0.1285407184, Free: 0.1285407184}},
		{"flex", currency.USD, accounts.Balance{Currency: currency.USD, Total: 5000, Free: 5000}},
	} {
		b, err := ex.Accounts.GetBalance(tc.subAccount, ex.GetDefaultCredentials(), asset.Futures, tc.c)
		require.NoErrorf(t, err, "GetBalance must not error for %s %s", tc.subAccount, tc.c)
		b.UpdatedAt = time.Time{}
		assert.Equalf(t, tc.exp, b, "the %s %s balance should be stored", tc.subAccount, tc.c)
	}

	positions, ok := got[22].(*WebsocketFuturesPositions)
	require.True(t, ok, "the open positions must be relayed")
	assert.Len(t, positions.Positions, 3, "every open position should be relayed")
	notifications, ok := got[23].(*WebsocketFuturesNotifications)
	require.True(t, ok, "the notifications must be relayed")
	assert.Equal(t, []WebsocketFuturesNotification{{ID: 5, Type: "market", Priority: "low", Note: "A note describing the notification.", EffectiveTime: types.Time(time.UnixMilli(1520288300000))}}, notifications.Notifications, "the notification should be relayed")
}

// futuresTestSubAccounts asserts a payload is relayed sub-accounts whose balances are stamped, and clears the stamps so
// they compare with an expected value
func futuresTestSubAccounts(t *testing.T, payload any) accounts.SubAccounts {
	t.Helper()
	subAccts, ok := payload.(accounts.SubAccounts)
	require.True(t, ok, "the payload must be sub-accounts")
	for _, s := range subAccts {
		for c, b := range s.Balances {
			assert.NotZerof(t, b.UpdatedAt, "the %s %s balance should be stamped", s.ID, c)
			b.UpdatedAt = time.Time{}
			s.Balances[c] = b
		}
	}
	return subAccts
}

func TestWebsocketFuturesBookSequenceGap(t *testing.T) {
	t.Parallel()
	server := new(mockFuturesServer)
	ex, conn := newFuturesTestExchange(t, server)
	s := futuresSubscription(subscription.OrderbookChannel, wsFuturesFeedBook, futuresTestPair)
	require.NoError(t, ex.futuresSubscribe(t.Context(), conn, subscription.List{s}), "futuresSubscribe must not error")
	server.takeRequests()
	snapshot := `{"feed":"book_snapshot","product_id":"PF_XBTUSD","timestamp":1791518718655,"seq":173877391,"tickSize":null,"bids":[{"price":82380.0,"qty":0.0003}],"asks":[{"price":82381.0,"qty":0.6051}]}`
	require.NoError(t, ex.wsFuturesHandleData(t.Context(), conn, []byte(snapshot)), "wsFuturesHandleData must load the snapshot")
	gap := `{"feed":"book","product_id":"PF_XBTUSD","side":"buy","seq":173877393,"price":81529.0,"qty":1.5296,"timestamp":1791518718725}`
	require.ErrorIs(t, ex.wsFuturesHandleData(t.Context(), conn, []byte(gap)), errFuturesSequenceGap, "an update out of sequence must error")
	_, err := ex.Websocket.Orderbook.GetOrderbook(futuresTestPair, asset.Futures)
	assert.ErrorIs(t, err, orderbook.ErrOrderbookInvalid, "an update out of sequence should invalidate the book")
	next := `{"feed":"book","product_id":"PF_XBTUSD","side":"buy","seq":173877394,"price":82343.0,"qty":0.1228,"timestamp":1791518718754}`
	assert.NoError(t, ex.wsFuturesHandleData(t.Context(), conn, []byte(next)), "updates to an invalid book should be dropped until a new snapshot")

	exp := []string{
		`{"event":"unsubscribe","feed":"book","product_ids":["PF_XBTUSD"]}`,
		`{"event":"subscribe","feed":"book","product_ids":["PF_XBTUSD"]}`,
	}
	var got []string
	require.Eventually(t, func() bool {
		got = append(got, server.takeRequests()...)
		return len(got) >= len(exp)
	}, testWaitTimeout, 10*time.Millisecond, "an update out of sequence must resubscribe the book")
	assert.Equal(t, exp, got, "an update out of sequence should resubscribe the book, unsubscribing first")
	require.Eventually(t, func() bool {
		stored := ex.Websocket.GetSubscription(s)
		return stored != nil && stored.State() == subscription.SubscribedState
	}, testWaitTimeout, 10*time.Millisecond, "the book must be subscribed again")
	require.NoError(t, ex.wsFuturesHandleData(t.Context(), conn, []byte(snapshot)), "wsFuturesHandleData must load the new snapshot")
	_, err = ex.Websocket.Orderbook.GetOrderbook(futuresTestPair, asset.Futures)
	assert.NoError(t, err, "a new snapshot should restore the book")
}

func TestWebsocketFuturesResubscribeBookChannelNames(t *testing.T) {
	t.Parallel()
	snapshot := `{"feed":"book_snapshot","product_id":"PF_XBTUSD","timestamp":1791518718655,"seq":173877391,"tickSize":null,"bids":[{"price":82380.0,"qty":0.0003}],"asks":[{"price":82381.0,"qty":0.6051}]}`
	gap := `{"feed":"book","product_id":"PF_XBTUSD","side":"buy","seq":173877393,"price":81529.0,"qty":1.5296,"timestamp":1791518718725}`
	// The book cannot apply an update without a timestamp
	rejected := `{"feed":"book","product_id":"PF_XBTUSD","side":"buy","seq":173877392,"price":81529.0,"qty":1.5296}`
	for _, tc := range []struct {
		name    string
		channel string
		update  string
		err     error
	}{
		{"orderbook sequence gap", subscription.OrderbookChannel, gap, errFuturesSequenceGap},
		{"book sequence gap", wsFuturesFeedBook, gap, errFuturesSequenceGap},
		{"orderbook rejected update", subscription.OrderbookChannel, rejected, orderbook.ErrLastUpdatedNotSet},
		{"book rejected update", wsFuturesFeedBook, rejected, orderbook.ErrLastUpdatedNotSet},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			server := new(mockFuturesServer)
			ex, conn := newFuturesTestExchange(t, server)
			s := futuresSubscription(tc.channel, wsFuturesFeedBook, futuresTestPair)
			require.NoError(t, ex.futuresSubscribe(t.Context(), conn, subscription.List{s}), "futuresSubscribe must not error")
			server.takeRequests()
			require.NoError(t, ex.wsFuturesHandleData(t.Context(), conn, []byte(snapshot)), "wsFuturesHandleData must load the snapshot")
			require.ErrorIs(t, ex.wsFuturesHandleData(t.Context(), conn, []byte(tc.update)), tc.err, "wsFuturesHandleData must reject the update")
			exp := []string{
				`{"event":"unsubscribe","feed":"book","product_ids":["PF_XBTUSD"]}`,
				`{"event":"subscribe","feed":"book","product_ids":["PF_XBTUSD"]}`,
			}
			var got []string
			require.Eventually(t, func() bool {
				got = append(got, server.takeRequests()...)
				return len(got) >= len(exp)
			}, testWaitTimeout, 10*time.Millisecond, "the book must be resubscribed")
			assert.Equal(t, exp, got, "the book should be unsubscribed before being subscribed again")
			require.Eventually(t, func() bool {
				return s.State() == subscription.SubscribedState && ex.Websocket.GetSubscription(s) == s
			}, testWaitTimeout, 10*time.Millisecond, "the subscription must be subscribed again")
		})
	}
}

func TestWebsocketFuturesUpdateBookResubscribe(t *testing.T) {
	t.Parallel()
	snapshot := `{"feed":"book_snapshot","product_id":"PF_XBTUSD","timestamp":1791518718655,"seq":173877391,"tickSize":null,"bids":[{"price":82380.0,"qty":0.0003}],"asks":[{"price":82381.0,"qty":0.6051}]}`
	setup := func(t *testing.T) (*Exchange, websocket.Connection, *mockFuturesServer, *subscription.Subscription) {
		t.Helper()
		server := new(mockFuturesServer)
		ex, conn := newFuturesTestExchange(t, server)
		s := futuresSubscription(subscription.OrderbookChannel, wsFuturesFeedBook, futuresTestPair)
		require.NoError(t, ex.futuresSubscribe(t.Context(), conn, subscription.List{s}), "futuresSubscribe must not error")
		server.takeRequests()
		require.NoError(t, ex.wsFuturesHandleData(t.Context(), conn, []byte(snapshot)), "wsFuturesHandleData must load the snapshot")
		return ex, conn, server, s
	}

	t.Run("rejected update", func(t *testing.T) {
		t.Parallel()
		ex, conn, server, s := setup(t)
		// The book cannot apply an update without a timestamp
		rejected := `{"feed":"book","product_id":"PF_XBTUSD","side":"buy","seq":173877392,"price":81529.0,"qty":1.5296}`
		err := ex.wsFuturesUpdateBook(t.Context(), conn, []byte(rejected))
		require.ErrorIs(t, err, orderbook.ErrLastUpdatedNotSet, "an update the book rejects must error")
		exp := []string{
			`{"event":"unsubscribe","feed":"book","product_ids":["PF_XBTUSD"]}`,
			`{"event":"subscribe","feed":"book","product_ids":["PF_XBTUSD"]}`,
		}
		var got []string
		require.Eventually(t, func() bool {
			got = append(got, server.takeRequests()...)
			return len(got) >= len(exp)
		}, testWaitTimeout, 10*time.Millisecond, "an update the book rejects must resubscribe the book")
		assert.Equal(t, exp, got, "an update the book rejects should resubscribe the book, unsubscribing first")
		require.Eventually(t, func() bool {
			stored := ex.Websocket.GetSubscription(s)
			return stored != nil && stored.State() == subscription.SubscribedState
		}, testWaitTimeout, 10*time.Millisecond, "the book must be subscribed again")
	})

	t.Run("unrelayed update", func(t *testing.T) {
		t.Parallel()
		ex, conn, server, _ := setup(t)
		var err error
		for err == nil {
			err = ex.Websocket.DataHandler.Send(t.Context(), nil)
		}
		update := `{"feed":"book","product_id":"PF_XBTUSD","side":"buy","seq":173877392,"price":81529.0,"qty":1.5296,"timestamp":1791518718725}`
		err = ex.wsFuturesUpdateBook(t.Context(), conn, []byte(update))
		require.Error(t, err, "an update the data handler cannot relay must error")
		assert.NotErrorIs(t, err, orderbook.ErrOrderbookInvalid, "an update the data handler cannot relay should leave the book valid")
		id, err := ex.Websocket.Orderbook.LastUpdateID(futuresTestPair, asset.Futures)
		require.NoError(t, err, "LastUpdateID must not error for a valid book")
		assert.Equal(t, int64(173877392), id, "the update should be applied although it was not relayed")
		var got []string
		assert.Never(t, func() bool {
			got = append(got, server.takeRequests()...)
			return len(got) != 0
		}, 100*time.Millisecond, 5*time.Millisecond, "an update the data handler cannot relay should not resubscribe the book")
	})
}

func TestWebsocketFuturesHandleDataErrors(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	conn, err := ex.Websocket.CreateTestConnection(wsFuturesConnection)
	require.NoError(t, err, "CreateTestConnection must not error")
	for _, tc := range []struct {
		name string
		msg  string
		err  error
	}{
		{"unmatched reply", `{"event":"subscribed","feed":"ticker","product_ids":["PF_XBTUSD"]}`, errFuturesUnexpectedReply},
		{"unsolicited alert", `{"event":"alert","message":"Bad request"}`, errFuturesUnexpectedReply},
		{"neither event nor feed", `{"product_id":"PF_XBTUSD"}`, websocket.ErrSignatureNotMatched},
		{"ticker product", `{"feed":"ticker","product_id":"PF_MOONUSD"}`, currency.ErrPairNotFound},
		{"book snapshot product", `{"feed":"book_snapshot","product_id":"PF_MOONUSD","bids":[],"asks":[]}`, currency.ErrPairNotFound},
		{"book update product", `{"feed":"book","product_id":"PF_MOONUSD","side":"buy","seq":1}`, currency.ErrPairNotFound},
		{"book update without a snapshot", `{"feed":"book","product_id":"PF_ETHUSD","side":"buy","seq":1,"price":2500,"qty":1,"timestamp":1791518718707}`, orderbook.ErrDepthNotFound},
		{"trade product", `{"feed":"trade","product_id":"PF_MOONUSD","side":"buy"}`, currency.ErrPairNotFound},
		{"trade side", `{"feed":"trade","product_id":"PF_XBTUSD","side":"up"}`, order.ErrSideIsInvalid},
		{"open order instrument", `{"feed":"open_orders_snapshot","orders":[{"instrument":"PF_MOONUSD","type":"limit"}]}`, currency.ErrPairNotFound},
		{"open order type", `{"feed":"open_orders","order":{"instrument":"PF_XBTUSD","type":"moon"}}`, errUnknownOrderType},
		{"fill instrument", `{"feed":"fills","fills":[{"instrument":"PF_MOONUSD","order_type":"lmt"}]}`, currency.ErrPairNotFound},
		{"fill order type", `{"feed":"fills","fills":[{"instrument":"PF_XBTUSD","order_type":"moon"}]}`, errUnknownOrderType},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tcEx := newTestExchange(t)
			tcEx.Features.Enabled.TradeFeed = true
			assert.ErrorIs(t, tcEx.wsFuturesHandleData(t.Context(), conn, []byte(tc.msg)), tc.err, "wsFuturesHandleData should return the expected error")
		})
	}
	ex.Features.Enabled.TradeFeed = true
	for _, feed := range []string{
		wsFuturesFeedTicker, wsFuturesFeedBookSnapshot, wsFuturesFeedBook, wsFuturesFeedTradeSnapshot, wsFuturesFeedTrade,
		wsFuturesFeedOpenOrdersSnapshot, wsFuturesFeedOpenOrders, wsFuturesFeedFillsSnapshot, wsFuturesFeedFills,
		wsFuturesFeedBalancesSnapshot, wsFuturesFeedBalances, wsFuturesFeedOpenPositions, wsFuturesFeedNotifications,
	} {
		msg := `{"feed":"` + feed + `","product_id":7,"order":7,"orders":7,"fills":7,"holding":7,"positions":7,"notifications":7,"trades":7}`
		assert.Errorf(t, ex.wsFuturesHandleData(t.Context(), conn, []byte(msg)), "wsFuturesHandleData should reject a %s message it cannot decode", feed)
	}
}

func TestWebsocketFuturesHandleDataUnhandled(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	conn, err := ex.Websocket.CreateTestConnection(wsFuturesConnection)
	require.NoError(t, err, "CreateTestConnection must not error")
	for _, msg := range []string{`{"event":"info","version":1}`, `{"feed":"heartbeat","time":1791518718452}`} {
		require.NoErrorf(t, ex.wsFuturesHandleData(t.Context(), conn, []byte(msg)), "wsFuturesHandleData must not error for %s", msg)
	}
	assert.Empty(t, ex.Websocket.DataHandler.C, "the info event and heartbeats should not be relayed")
	require.NoError(t, ex.wsFuturesHandleData(t.Context(), conn, []byte(`{"feed":"account_log","logs":[]}`)), "wsFuturesHandleData must not error for an unhandled feed")
	require.Len(t, ex.Websocket.DataHandler.C, 1, "an unhandled feed's message must be relayed")
	warning, ok := (<-ex.Websocket.DataHandler.C).Data.(websocket.UnhandledMessageWarning)
	require.True(t, ok, "an unhandled feed's message must be relayed as a warning")
	assert.Contains(t, warning.Message, "account_log", "the warning should carry the message")
}

func TestWebsocketFuturesHandleDataTradesDisabled(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	ex.Features.Enabled.TradeFeed = false
	ex.Features.Enabled.SaveTradeData = false
	require.NoError(t, ex.wsFuturesHandleData(t.Context(), nil, []byte(`{"feed":"trade","product_id":7}`)), "wsFuturesHandleData must not decode trades nobody uses")
	assert.Empty(t, ex.Websocket.DataHandler.C, "trades should not be relayed when the trade feed is disabled")
}

func TestFuturesOrderUpdateStatus(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		reason   string
		isCancel bool
		filled   float64
		exp      order.Status
	}{
		{"new_placed_order_by_user", false, 0, order.Open},
		{"limit_order_from_stop", false, 0, order.Open},
		{"partial_fill", false, 0.25, order.PartiallyFilled},
		{"full_fill", true, 0, order.Filled},
		{"stop_order_triggered", true, 0, order.Closed},
		{"contract_expired", true, 0, order.Expired},
		{"cancelled_by_user", true, 0, order.Cancelled},
		{"cancelled_by_user", true, 0.25, order.PartiallyFilledCancelled},
		{"cancelled_by_admin", true, 0, order.Cancelled},
		{"dead_man_switch", true, 0, order.Cancelled},
		{"liquidation", true, 0, order.Cancelled},
		{"not_enough_margin", true, 0, order.Cancelled},
		{"market_inactive", true, 0, order.Cancelled},
		{"post_order_failed_because_it_would_filled", true, 0, order.Rejected},
		{"ioc_order_failed_because_it_would_not_be_executed", true, 0, order.Rejected},
		{"would_execute_self", true, 0, order.Rejected},
		{"would_not_reduce_position", true, 0, order.Rejected},
		{"order_for_edit_not_found", true, 0, order.Rejected},
	} {
		assert.Equalf(t, tc.exp, futuresOrderUpdateStatus(tc.reason, tc.isCancel, tc.filled), "futuresOrderUpdateStatus should convert %s with %v filled", tc.reason, tc.filled)
	}
}

func TestWebsocketFuturesConnect(t *testing.T) {
	t.Parallel()
	var restRequests atomic.Int32
	ex := newHTTPTestExchange(t, func(w http.ResponseWriter, _ *http.Request) {
		restRequests.Add(1)
		_, err := w.Write([]byte(`{"error":["EService:Unavailable"]}`))
		assert.NoError(t, err, "Write should not error")
	})
	useTestWebsocket(t, ex, newMockWebsocketURL(t, func(testing.TB, []byte, *gws.Conn) error { return nil }), wsFuturesConnection)
	conn, err := ex.Websocket.CreateTestConnection(wsFuturesConnection)
	require.NoError(t, err, "CreateTestConnection must not error")
	// Fee currencies are translated once the assets are seeded, so connecting goes on without them
	require.NoError(t, ex.wsFuturesConnect(t.Context(), conn), "wsFuturesConnect must not error when the assets cannot be seeded")
	t.Cleanup(func() { assert.NoError(t, conn.Shutdown(), "Shutdown should not error") })
	assert.NotZero(t, restRequests.Load(), "wsFuturesConnect should try to seed the assets")
	assert.False(t, ex.assetNames.seeded(), "wsFuturesConnect should leave the assets unseeded when seeding fails")

	seedTestAssetNames(ex)
	restRequests.Store(0)
	seeded, err := ex.Websocket.CreateTestConnection(wsFuturesConnection)
	require.NoError(t, err, "CreateTestConnection must not error")
	require.NoError(t, ex.wsFuturesConnect(t.Context(), seeded), "wsFuturesConnect must not error with the assets seeded")
	t.Cleanup(func() { assert.NoError(t, seeded.Shutdown(), "Shutdown should not error") })
	assert.Zero(t, restRequests.Load(), "wsFuturesConnect should not seed assets already seeded")

	useTestWebsocket(t, ex, "ws://127.0.0.1:1", wsFuturesConnection)
	unreachable, err := ex.Websocket.CreateTestConnection(wsFuturesConnection)
	require.NoError(t, err, "CreateTestConnection must not error")
	assert.Error(t, ex.wsFuturesConnect(t.Context(), unreachable), "wsFuturesConnect should return a failed dial's error")
}

func TestManageFuturesSubs(t *testing.T) {
	t.Parallel()
	server := &mockFuturesServer{alerts: map[string]string{"PI_XBTUSD": "Couldn't subscribe to invalid product `PI_XBTUSD`"}}
	ex, conn := newFuturesTestExchange(t, server)
	tickerPF := futuresSubscription(subscription.TickerChannel, wsFuturesFeedTicker, futuresTestPair)
	tradePF := futuresSubscription(subscription.AllTradesChannel, wsFuturesFeedTrade, futuresTestPair)
	tickerPI := futuresSubscription(subscription.TickerChannel, wsFuturesFeedTicker, piXBTUSDPair)
	err := ex.manageFuturesSubs(t.Context(), conn, wsFuturesEventSubscribe, subscription.List{tickerPF, tradePF, tickerPI})
	assert.ErrorIs(t, err, errFuturesSubscriptionFailed, "manageFuturesSubs should report a subscription Kraken alerts")
	assertRequests(t, []string{
		`{"event":"subscribe","feed":"ticker","product_ids":["PF_XBTUSD","PI_XBTUSD"]}`,
		`{"event":"subscribe","feed":"trade","product_ids":["PF_XBTUSD"]}`,
	}, server.takeRequests())
	assert.NotNil(t, ex.Websocket.GetSubscription(tickerPF), "manageFuturesSubs should record a subscription Kraken acknowledges")
	assert.NotNil(t, ex.Websocket.GetSubscription(tradePF), "manageFuturesSubs should go on to the next group after a failed one")
	assert.Nil(t, ex.Websocket.GetSubscription(tickerPI), "manageFuturesSubs should not record a subscription Kraken alerts")

	require.NoError(t, ex.manageFuturesSubs(t.Context(), conn, wsFuturesEventUnsubscribe, subscription.List{tickerPF, tradePF}), "manageFuturesSubs must not error unsubscribing")
	assertRequests(t, []string{
		`{"event":"unsubscribe","feed":"ticker","product_ids":["PF_XBTUSD"]}`,
		`{"event":"unsubscribe","feed":"trade","product_ids":["PF_XBTUSD"]}`,
	}, server.takeRequests())
	assert.Empty(t, ex.Websocket.GetSubscriptions(), "manageFuturesSubs should remove each subscription Kraken unsubscribes")
}

func TestManageFuturesSubGroup(t *testing.T) {
	t.Parallel()
	server := &mockFuturesServer{alerts: map[string]string{wsFuturesFeedHeartbeat: wsFuturesAlreadySubscribed}}
	ex, conn := newFuturesTestExchange(t, server)
	signed := futuresTestSignedChallenge(t)
	tickers := subscription.List{
		futuresSubscription(subscription.TickerChannel, wsFuturesFeedTicker, futuresTestPair),
		futuresSubscription(subscription.TickerChannel, wsFuturesFeedTicker, piXBTUSDPair),
	}
	for _, tc := range []struct {
		name  string
		group subscription.List
		exp   []string
	}{
		{name: "product feed", group: tickers, exp: []string{`{"event":"subscribe","feed":"ticker","product_ids":["PF_XBTUSD","PI_XBTUSD"]}`}},
		// Kraken alerts a request for a feed already subscribed to, which leaves the feed as asked
		{name: "feed without products", group: subscription.List{futuresSubscription(subscription.HeartbeatChannel, wsFuturesFeedHeartbeat)}, exp: []string{`{"event":"subscribe","feed":"heartbeat"}`}},
		{
			name:  "account feed",
			group: subscription.List{futuresSubscription(subscription.MyOrdersChannel, wsFuturesFeedOpenOrders)},
			exp: []string{
				`{"event":"challenge","api_key":"test-key"}`,
				`{"event":"subscribe","feed":"open_orders","api_key":"test-key","original_challenge":"` + futuresTestChallenge + `","signed_challenge":"` + signed + `"}`,
			},
		},
	} {
		require.NoErrorf(t, ex.manageFuturesSubGroup(t.Context(), conn, wsFuturesEventSubscribe, tc.group), "manageFuturesSubGroup must not error subscribing to the %s", tc.name)
		assertRequests(t, tc.exp, server.takeRequests())
		for _, s := range tc.group {
			assert.Equalf(t, subscription.SubscribedState, s.State(), "manageFuturesSubGroup should record the %s subscription %s", tc.name, s)
		}
	}
	require.NoError(t, ex.manageFuturesSubGroup(t.Context(), conn, wsFuturesEventUnsubscribe, tickers), "manageFuturesSubGroup must not error unsubscribing")
	assertRequests(t, []string{`{"event":"unsubscribe","feed":"ticker","product_ids":["PF_XBTUSD","PI_XBTUSD"]}`}, server.takeRequests())
	for _, s := range tickers {
		assert.Nilf(t, ex.Websocket.GetSubscription(s), "manageFuturesSubGroup should remove the unsubscribed %s", s)
	}

	unanswered := futuresSubscription(subscription.TickerChannel, wsFuturesFeedTicker)
	answered := futuresSubscription(subscription.TickerChannel, wsFuturesFeedTicker, futuresTestPair)
	err := ex.manageFuturesSubGroup(t.Context(), conn, wsFuturesEventSubscribe, subscription.List{answered, unanswered})
	assert.ErrorIs(t, err, errSubscriptionResponseMissing, "manageFuturesSubGroup should report a subscription no reply answers")
	assert.NotNil(t, ex.Websocket.GetSubscription(answered), "manageFuturesSubGroup should record the subscription a reply answers")
	assertRequests(t, []string{`{"event":"subscribe","feed":"ticker","product_ids":["PF_XBTUSD"]}`}, server.takeRequests())

	err = ex.manageFuturesSubGroup(t.Context(), conn, wsFuturesEventSubscribe, subscription.List{futuresSubscription(subscription.TickerChannel, wsFuturesFeedTicker, currency.EMPTYPAIR)})
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "manageFuturesSubGroup should reject a subscription it cannot name a product for")
	assert.Empty(t, server.takeRequests(), "manageFuturesSubGroup should send nothing for a subscription it rejects")

	disconnected, err := ex.Websocket.CreateTestConnection(wsFuturesConnection)
	require.NoError(t, err, "CreateTestConnection must not error")
	assert.Error(t, ex.manageFuturesSubGroup(t.Context(), disconnected, wsFuturesEventSubscribe, subscription.List{futuresSubscription(subscription.HeartbeatChannel, wsFuturesFeedHeartbeat)}), "manageFuturesSubGroup should return a failed request's error")

	tickerSub := futuresSubscription(subscription.TickerChannel, wsFuturesFeedTicker, futuresTestPair)
	heartbeatSub := futuresSubscription(subscription.HeartbeatChannel, wsFuturesFeedHeartbeat)
	for _, tc := range []struct {
		name, reply string
		s           *subscription.Subscription
		err         error
	}{
		{"error event", `{"event":"error","message":"Invalid product id"}`, tickerSub, errFuturesSubscriptionFailed},
		{"product not requested", `{"event":"subscribed","feed":"ticker","product_ids":["PF_ETHUSD"]}`, tickerSub, errFuturesUnexpectedReply},
		{"several products", `{"event":"subscribed","feed":"ticker","product_ids":["PF_XBTUSD","PF_ETHUSD"]}`, tickerSub, errFuturesUnexpectedReply},
		{"product for a feed without products", `{"event":"subscribed","feed":"heartbeat","product_ids":["PF_XBTUSD"]}`, heartbeatSub, errFuturesUnexpectedReply},
	} {
		server.m.Lock()
		server.reply = tc.reply
		server.m.Unlock()
		err := ex.manageFuturesSubGroup(t.Context(), conn, wsFuturesEventUnsubscribe, subscription.List{tc.s})
		assert.ErrorIsf(t, err, tc.err, "manageFuturesSubGroup should reject a reply with %s", tc.name)
	}
	server.m.Lock()
	server.reply = `{"event":"unsubscribed","feed":"ticker","product_ids":"PF_XBTUSD"}`
	server.m.Unlock()
	assert.Error(t, ex.manageFuturesSubGroup(t.Context(), conn, wsFuturesEventUnsubscribe, subscription.List{answered}), "manageFuturesSubGroup should reject a reply it cannot decode")
	assert.NotNil(t, ex.Websocket.GetSubscription(answered), "manageFuturesSubGroup should keep a subscription whose unsubscription fails")

	noCreds := newTestExchange(t)
	noCreds.SkipAuthCheck = false
	noCreds.API.AuthenticatedSupport = true
	noCreds.SetCredentials(&accounts.Credentials{})
	err = noCreds.manageFuturesSubGroup(t.Context(), conn, wsFuturesEventSubscribe, subscription.List{futuresSubscription(subscription.MyTradesChannel, wsFuturesFeedFills)})
	assert.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "manageFuturesSubGroup should not request an account feed it cannot sign")
}

func TestSignFuturesRequest(t *testing.T) {
	t.Parallel()
	server := new(mockFuturesServer)
	ex, conn := newFuturesTestExchange(t, server)
	signed := futuresTestSignedChallenge(t)
	req := &wsFuturesRequest{Event: wsFuturesEventSubscribe, Feed: wsFuturesFeedOpenOrders}
	require.NoError(t, ex.signFuturesRequest(t.Context(), conn, req), "signFuturesRequest must not error")
	exp := &wsFuturesRequest{Event: wsFuturesEventSubscribe, Feed: wsFuturesFeedOpenOrders, APIKey: "test-key", OriginalChallenge: futuresTestChallenge, SignedChallenge: signed}
	assert.Equal(t, exp, req, "signFuturesRequest should sign the request with the connection's challenge")
	assertRequests(t, []string{`{"event":"challenge","api_key":"test-key"}`}, server.takeRequests())

	req = &wsFuturesRequest{Event: wsFuturesEventSubscribe, Feed: wsFuturesFeedOpenOrders}
	require.NoError(t, ex.signFuturesRequest(t.Context(), conn, req), "signFuturesRequest must not error signing again")
	assert.Equal(t, exp, req, "signFuturesRequest should sign with the challenge it holds")
	assert.Empty(t, server.takeRequests(), "signFuturesRequest should not request a challenge the connection holds")

	// A challenge is issued on a connection, so another connection's challenge is not reused
	other, err := ex.Websocket.CreateTestConnection(wsFuturesConnection)
	require.NoError(t, err, "CreateTestConnection must not error")
	ex.futuresChallengeMu.Lock()
	ex.futuresChallenge = futuresChallenge{conn: other, original: "stale", signed: "stale"}
	ex.futuresChallengeMu.Unlock()
	req = &wsFuturesRequest{Event: wsFuturesEventSubscribe, Feed: wsFuturesFeedOpenOrders}
	require.NoError(t, ex.signFuturesRequest(t.Context(), conn, req), "signFuturesRequest must not error for a connection without a challenge")
	assert.Equal(t, exp, req, "signFuturesRequest should sign with the connection's own challenge")
	assertRequests(t, []string{`{"event":"challenge","api_key":"test-key"}`}, server.takeRequests())
	assert.Error(t, ex.signFuturesRequest(t.Context(), other, &wsFuturesRequest{}), "signFuturesRequest should return a failed challenge request's error")

	for _, tc := range []struct {
		name, reply string
		err         error
	}{
		{"refused", `{"event":"error","message":"Json Error"}`, errFuturesSubscriptionFailed},
		{"empty", `{"event":"challenge","message":""}`, errFuturesChallengeEmpty},
	} {
		tcEx, tcConn := newFuturesTestExchange(t, &mockFuturesServer{challengeReply: tc.reply})
		err := tcEx.signFuturesRequest(t.Context(), tcConn, &wsFuturesRequest{})
		assert.ErrorIsf(t, err, tc.err, "signFuturesRequest should reject a challenge reply that is %s", tc.name)
	}
	undecodableEx, undecodableConn := newFuturesTestExchange(t, &mockFuturesServer{challengeReply: `{"event":7}`})
	assert.Error(t, undecodableEx.signFuturesRequest(t.Context(), undecodableConn, &wsFuturesRequest{}), "signFuturesRequest should reject a challenge reply it cannot decode")

	noCreds := newTestExchange(t)
	noCreds.SkipAuthCheck = false
	noCreds.API.AuthenticatedSupport = true
	noCreds.SetCredentials(&accounts.Credentials{})
	assert.ErrorIs(t, noCreds.signFuturesRequest(t.Context(), conn, &wsFuturesRequest{}), exchange.ErrCredentialsAreEmpty, "signFuturesRequest should not sign without credentials")
}

func TestFuturesRequest(t *testing.T) {
	t.Parallel()
	server := new(mockFuturesServer)
	ex, conn := newFuturesTestExchange(t, server)
	resps, err := ex.futuresRequest(t.Context(), conn, &wsFuturesRequest{Event: wsFuturesEventSubscribe, Feed: wsFuturesFeedTicker, ProductIDs: []string{"PF_XBTUSD", "PI_XBTUSD"}}, 2)
	require.NoError(t, err, "futuresRequest must not error")
	require.Len(t, resps, 2, "futuresRequest must return each reply expected")
	assert.JSONEq(t, `{"event":"subscribed","feed":"ticker","product_ids":["PF_XBTUSD"]}`, string(resps[0]), "futuresRequest should return the replies in the order received")
	assert.JSONEq(t, `{"event":"subscribed","feed":"ticker","product_ids":["PI_XBTUSD"]}`, string(resps[1]), "futuresRequest should return the replies in the order received")
	assertRequests(t, []string{`{"event":"subscribe","feed":"ticker","product_ids":["PF_XBTUSD","PI_XBTUSD"]}`}, server.takeRequests())

	// Kraken's replies do not name the request they answer, so each request must wait for the one before to be answered
	var wg sync.WaitGroup
	for _, product := range []string{"PF_XBTUSD", "PF_ETHUSD", "PF_SOLUSD", "PF_XRPUSD", "PF_LTCUSD", "PF_DOTUSD", "PF_ADAUSD", "PF_BCHUSD"} {
		wg.Go(func() {
			resps, err := ex.futuresRequest(t.Context(), conn, &wsFuturesRequest{Event: wsFuturesEventSubscribe, Feed: wsFuturesFeedTicker, ProductIDs: []string{product}}, 1)
			if assert.NoErrorf(t, err, "futuresRequest should not error for %s", product) && assert.Lenf(t, resps, 1, "futuresRequest should return %s's reply", product) {
				assert.JSONEqf(t, `{"event":"subscribed","feed":"ticker","product_ids":["`+product+`"]}`, string(resps[0]), "futuresRequest should return the reply to %s's request", product)
			}
		})
	}
	wg.Wait()

	disconnected, err := ex.Websocket.CreateTestConnection(wsFuturesConnection)
	require.NoError(t, err, "CreateTestConnection must not error")
	_, err = ex.futuresRequest(t.Context(), disconnected, &wsFuturesRequest{Event: wsFuturesEventSubscribe, Feed: wsFuturesFeedHeartbeat}, 1)
	assert.Error(t, err, "futuresRequest should return a failed send's error")
}

func TestFuturesPair(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	pair, err := ex.futuresPair("PF_XBTUSD")
	require.NoError(t, err, "futuresPair must not error")
	assert.Equal(t, futuresTestPair, pair, "futuresPair should return the available pair the product ID names")
	_, err = ex.futuresPair("PF_MOONUSD")
	assert.ErrorIs(t, err, currency.ErrPairNotFound, "futuresPair should reject a product without an available pair")
}

func TestWebsocketFuturesProcessTicker(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	msg := `{"time":1791518715420,"product_id":"PI_XBTUSD","feed":"ticker","bid":82370.5,"ask":82371.0,"bid_size":1200,"ask_size":3400,"volume":596069,"volumeQuote":596102,"index":82367.95,"last":82371.0,"change":-0.25,"openInterest":21450021,"markPrice":82370.2,"open":82590.0,"high":83200.0,"low":80310.0}`
	require.NoError(t, ex.wsFuturesProcessTicker(t.Context(), []byte(msg)), "wsFuturesProcessTicker must not error")
	exp := &ticker.Price{
		Last:    82371,
		Bid:     82370.5,
		BidSize: 1200,
		Ask:     82371,
		AskSize: 3400,
		// An inverse contract's volume counts its quote currency
		QuoteVolume:         596102,
		Open:                82590,
		High:                83200,
		Low:                 80310,
		PercentChange24Hour: -0.25,
		OpenInterest:        21450021,
		MarkPrice:           82370.2,
		IndexPrice:          82367.95,
		Pair:                piXBTUSDPair,
		ExchangeName:        t.Name(),
		AssetType:           asset.Futures,
		LastUpdated:         time.UnixMilli(1791518715420),
	}
	assert.Equal(t, []any{exp}, relayed(ex), "wsFuturesProcessTicker should relay the ticker")
	stored, err := ticker.GetTicker(t.Name(), piXBTUSDPair, asset.Futures)
	require.NoError(t, err, "GetTicker must not error")
	assert.Equal(t, exp, stored, "wsFuturesProcessTicker should store the ticker it relays")

	err = ex.wsFuturesProcessTicker(t.Context(), []byte(`{"feed":"ticker","product_id":"PF_XBTUSD","bid":82380.0,"ask":82380.0}`))
	assert.ErrorIs(t, err, ticker.ErrBidEqualsAsk, "wsFuturesProcessTicker should return a ticker the store rejects")
	assert.Empty(t, relayed(ex), "wsFuturesProcessTicker should not relay a ticker the store rejects")
	assert.ErrorIs(t, ex.wsFuturesProcessTicker(t.Context(), []byte(`{"feed":"ticker","product_id":"PF_MOONUSD"}`)), currency.ErrPairNotFound, "wsFuturesProcessTicker should reject an unknown product")
	assert.Error(t, ex.wsFuturesProcessTicker(t.Context(), []byte(`{"feed":"ticker","product_id":7}`)), "wsFuturesProcessTicker should reject a ticker it cannot decode")
}

func TestWebsocketFuturesLoadBookSnapshot(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	require.NoError(t, ex.wsFuturesLoadBookSnapshot(t.Context(), fixtureLines(t, "testdata/wsFutures.json")[3]), "wsFuturesLoadBookSnapshot must not error")
	book, err := ex.Websocket.Orderbook.GetOrderbook(futuresTestPair, asset.Futures)
	require.NoError(t, err, "GetOrderbook must not error")
	assert.Equal(t, orderbook.Levels{{Price: 82380, Amount: 0.0003}, {Price: 82379, Amount: 0.0012}, {Price: 82378, Amount: 0.0003}}, book.Bids, "wsFuturesLoadBookSnapshot should load the bids")
	assert.Equal(t, orderbook.Levels{{Price: 82381, Amount: 0.6051}, {Price: 82382, Amount: 0.4345}, {Price: 82383, Amount: 0.0243}}, book.Asks, "wsFuturesLoadBookSnapshot should load the asks")
	assert.Equal(t, time.UnixMilli(1791518718655), book.LastUpdated, "wsFuturesLoadBookSnapshot should stamp the book with the snapshot's time")
	assert.Equal(t, int64(173877391), book.LastUpdateID, "wsFuturesLoadBookSnapshot should record the snapshot's sequence number")
	assert.ErrorIs(t, ex.wsFuturesLoadBookSnapshot(t.Context(), []byte(`{"feed":"book_snapshot","product_id":"PF_MOONUSD","bids":[],"asks":[]}`)), currency.ErrPairNotFound, "wsFuturesLoadBookSnapshot should reject an unknown product")
	assert.Error(t, ex.wsFuturesLoadBookSnapshot(t.Context(), []byte(`{"feed":"book_snapshot","bids":7}`)), "wsFuturesLoadBookSnapshot should reject a snapshot it cannot decode")
}

func TestWebsocketFuturesUpdateBook(t *testing.T) {
	t.Parallel()
	server := new(mockFuturesServer)
	ex, conn := newFuturesTestExchange(t, server)
	s := futuresSubscription(subscription.OrderbookChannel, wsFuturesFeedBook, futuresTestPair)
	require.NoError(t, ex.futuresSubscribe(t.Context(), conn, subscription.List{s}), "futuresSubscribe must not error")
	server.takeRequests()
	lines := fixtureLines(t, "testdata/wsFutures.json")
	assert.ErrorIs(t, ex.wsFuturesUpdateBook(t.Context(), conn, lines[4]), orderbook.ErrDepthNotFound, "wsFuturesUpdateBook should reject an update before a snapshot")
	require.NoError(t, ex.wsFuturesLoadBookSnapshot(t.Context(), lines[3]), "wsFuturesLoadBookSnapshot must not error")
	require.NoError(t, ex.wsFuturesUpdateBook(t.Context(), conn, lines[4]), "wsFuturesUpdateBook must not error for a bid update")
	ask := `{"feed":"book","product_id":"PF_XBTUSD","side":"sell","seq":173877393,"price":82382.0,"qty":0.0,"timestamp":1791518718725}`
	require.NoError(t, ex.wsFuturesUpdateBook(t.Context(), conn, []byte(ask)), "wsFuturesUpdateBook must not error for an ask update")
	book, err := ex.Websocket.Orderbook.GetOrderbook(futuresTestPair, asset.Futures)
	require.NoError(t, err, "GetOrderbook must not error")
	assert.Equal(t, orderbook.Levels{{Price: 82380, Amount: 0.0003}, {Price: 82379, Amount: 0.0012}, {Price: 82378, Amount: 0.0003}, {Price: 81235, Amount: 0.0031}}, book.Bids, "wsFuturesUpdateBook should apply a buy update to the bids")
	assert.Equal(t, orderbook.Levels{{Price: 82381, Amount: 0.6051}, {Price: 82383, Amount: 0.0243}}, book.Asks, "wsFuturesUpdateBook should remove the ask level a sell update zeroes")
	assert.Equal(t, int64(173877393), book.LastUpdateID, "wsFuturesUpdateBook should follow the updates' sequence numbers")
	assert.Equal(t, time.UnixMilli(1791518718725), book.LastUpdated, "wsFuturesUpdateBook should stamp the book with the last update's time")

	gap := `{"feed":"book","product_id":"PF_XBTUSD","side":"buy","seq":173877395,"price":82343.0,"qty":0.1228,"timestamp":1791518718754}`
	assert.ErrorIs(t, ex.wsFuturesUpdateBook(t.Context(), conn, []byte(gap)), errFuturesSequenceGap, "wsFuturesUpdateBook should reject an update out of sequence")
	_, err = ex.Websocket.Orderbook.GetOrderbook(futuresTestPair, asset.Futures)
	assert.ErrorIs(t, err, orderbook.ErrOrderbookInvalid, "wsFuturesUpdateBook should invalidate a book an update skips")
	assert.NoError(t, ex.wsFuturesUpdateBook(t.Context(), conn, []byte(gap)), "wsFuturesUpdateBook should drop updates to an invalid book")
	exp := []string{
		`{"event":"unsubscribe","feed":"book","product_ids":["PF_XBTUSD"]}`,
		`{"event":"subscribe","feed":"book","product_ids":["PF_XBTUSD"]}`,
	}
	var got []string
	require.Eventually(t, func() bool {
		got = append(got, server.takeRequests()...)
		return len(got) >= len(exp)
	}, testWaitTimeout, 10*time.Millisecond, "wsFuturesUpdateBook must resubscribe a book an update skips")
	assert.Equal(t, exp, got, "wsFuturesUpdateBook should resubscribe a book an update skips once, unsubscribing first")
	require.Eventually(t, func() bool {
		stored := ex.Websocket.GetSubscription(s)
		return stored != nil && stored.State() == subscription.SubscribedState
	}, testWaitTimeout, 10*time.Millisecond, "the book must be subscribed again")

	assert.ErrorIs(t, ex.wsFuturesUpdateBook(t.Context(), conn, []byte(`{"feed":"book","product_id":"PF_MOONUSD","side":"buy","seq":1}`)), currency.ErrPairNotFound, "wsFuturesUpdateBook should reject an unknown product")
	assert.Error(t, ex.wsFuturesUpdateBook(t.Context(), conn, []byte(`{"feed":"book","product_id":"PF_XBTUSD","seq":"1"}`)), "wsFuturesUpdateBook should reject an update it cannot decode")
}

func TestFuturesBookLevels(t *testing.T) {
	t.Parallel()
	got := futuresBookLevels([]WebsocketFuturesBookLevel{{Price: 82380, Quantity: 0.0003}, {Price: 82379, Quantity: 0.0012}})
	assert.Equal(t, orderbook.Levels{{Price: 82380, Amount: 0.0003}, {Price: 82379, Amount: 0.0012}}, got, "futuresBookLevels should convert each level in order")
	assert.Empty(t, futuresBookLevels(nil), "futuresBookLevels should return no levels without any")
}

func TestWebsocketFuturesProcessTrades(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	ex.Features.Enabled.TradeFeed = true
	ex.Websocket.Trade.Setup(true, ex.Websocket.DataHandler)
	snapshot := `{"feed":"trade_snapshot","product_id":"PF_XBTUSD","trades":[` +
		`{"product_id":"PF_XBTUSD","feed":"trade","uid":"8fc9d2b3-f541-4e2b-8e0c-11559e0cd62b","side":"sell","type":"fill","time":1791518714883,"qty":0.0002,"price":82382.0,"seq":842136},` +
		`{"product_id":"PF_XBTUSD","feed":"trade","uid":"d4d20435-6841-4f2d-8df9-0cd6a4198172","side":"buy","type":"fill","time":1791518714890,"qty":0.0001,"price":82383.0,"seq":842137}]}`
	require.NoError(t, ex.wsFuturesProcessTrades(wsFuturesFeedTradeSnapshot, []byte(snapshot)), "wsFuturesProcessTrades must not error for a snapshot")
	assert.Equal(t, []any{[]trade.Data{
		{Exchange: t.Name(), CurrencyPair: futuresTestPair, AssetType: asset.Futures, TID: "8fc9d2b3-f541-4e2b-8e0c-11559e0cd62b", Side: order.Sell, Price: 82382, Amount: 0.0002, Timestamp: time.UnixMilli(1791518714883)},
		{Exchange: t.Name(), CurrencyPair: futuresTestPair, AssetType: asset.Futures, TID: "d4d20435-6841-4f2d-8df9-0cd6a4198172", Side: order.Buy, Price: 82383, Amount: 0.0001, Timestamp: time.UnixMilli(1791518714890)},
	}}, relayed(ex), "wsFuturesProcessTrades should relay a snapshot's trades")
	single := `{"product_id":"PF_XBTUSD","feed":"trade","uid":"4fba15f2-e9e2-4811-8016-2c8b5d1e4c6e","side":"sell","type":"liquidation","time":1791518719257,"qty":0.0002,"price":82380.0,"seq":842138}`
	require.NoError(t, ex.wsFuturesProcessTrades(wsFuturesFeedTrade, []byte(single)), "wsFuturesProcessTrades must not error for a trade")
	assert.Equal(t, []any{[]trade.Data{
		{Exchange: t.Name(), CurrencyPair: futuresTestPair, AssetType: asset.Futures, TID: "4fba15f2-e9e2-4811-8016-2c8b5d1e4c6e", Side: order.Sell, Price: 82380, Amount: 0.0002, Timestamp: time.UnixMilli(1791518719257)},
	}}, relayed(ex), "wsFuturesProcessTrades should relay a trade")

	for _, tc := range []struct {
		name, feed, msg string
		err             error
	}{
		{"unknown product", wsFuturesFeedTrade, `{"feed":"trade","product_id":"PF_MOONUSD","side":"buy"}`, currency.ErrPairNotFound},
		{"unknown side", wsFuturesFeedTradeSnapshot, `{"feed":"trade_snapshot","trades":[{"product_id":"PF_XBTUSD","side":"up"}]}`, order.ErrSideIsInvalid},
	} {
		assert.ErrorIsf(t, ex.wsFuturesProcessTrades(tc.feed, []byte(tc.msg)), tc.err, "wsFuturesProcessTrades should reject a trade with an %s", tc.name)
	}
	assert.Error(t, ex.wsFuturesProcessTrades(wsFuturesFeedTradeSnapshot, []byte(`{"feed":"trade_snapshot","trades":7}`)), "wsFuturesProcessTrades should reject a snapshot it cannot decode")
	assert.Error(t, ex.wsFuturesProcessTrades(wsFuturesFeedTrade, []byte(`{"feed":"trade","qty":"1"}`)), "wsFuturesProcessTrades should reject a trade it cannot decode")
	assert.Empty(t, relayed(ex), "wsFuturesProcessTrades should relay no trades from a message it rejects")

	ex.Features.Enabled.TradeFeed = false
	ex.Features.Enabled.SaveTradeData = false
	assert.NoError(t, ex.wsFuturesProcessTrades(wsFuturesFeedTrade, []byte(`{"feed":"trade","qty":"1"}`)), "wsFuturesProcessTrades should not decode trades with the trade feed and trade saving disabled")
}

func TestWebsocketFuturesProcessOpenOrders(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	placed, updated := time.UnixMilli(1612275024153), time.UnixMilli(1612275178153)
	snapshot := `{"feed":"open_orders_snapshot","account":"e258dba9-4dd4-4da5-bfef-75beb91c098e","orders":[{"instrument":"PF_XBTUSD","time":1612275024153,"last_update_time":1612275178153,"qty":10,"filled":4,"limit_price":34900,"stop_price":0,"type":"limit","order_id":"723ba95f-13b7-418b-8fcf-ab7ba6620555","cli_ord_id":"my-order","direction":0,"reduce_only":false}]}`
	require.NoError(t, ex.wsFuturesProcessOpenOrders(t.Context(), nil, wsFuturesFeedOpenOrdersSnapshot, []byte(snapshot)), "wsFuturesProcessOpenOrders must not error for a snapshot")
	assert.Equal(t, []any{&order.Detail{
		Exchange:        t.Name(),
		AssetType:       asset.Futures,
		OrderID:         "723ba95f-13b7-418b-8fcf-ab7ba6620555",
		ClientOrderID:   "my-order",
		Pair:            futuresTestPair,
		Side:            order.Buy,
		Type:            order.Limit,
		Status:          order.PartiallyFilled,
		Price:           34900,
		Amount:          10,
		ExecutedAmount:  4,
		RemainingAmount: 6,
		Date:            placed,
		LastUpdated:     updated,
	}}, relayed(ex), "wsFuturesProcessOpenOrders should relay a snapshot's orders, a partly filled one as such")

	filled := `{"feed":"open_orders","order":{"instrument":"PF_XBTUSD","time":1612275024153,"last_update_time":1612275178153,"qty":10,"filled":10,"limit_price":34900,"stop_price":0,"type":"limit","order_id":"723ba95f-13b7-418b-8fcf-ab7ba6620555","cli_ord_id":"my-order","direction":0,"reduce_only":false},"is_cancel":true,"reason":"full_fill"}`
	require.NoError(t, ex.wsFuturesProcessOpenOrders(t.Context(), nil, wsFuturesFeedOpenOrders, []byte(filled)), "wsFuturesProcessOpenOrders must not error for an update")
	assert.Equal(t, []any{&order.Detail{
		Exchange:       t.Name(),
		AssetType:      asset.Futures,
		OrderID:        "723ba95f-13b7-418b-8fcf-ab7ba6620555",
		ClientOrderID:  "my-order",
		Pair:           futuresTestPair,
		Side:           order.Buy,
		Type:           order.Limit,
		Status:         order.Filled,
		Price:          34900,
		Amount:         10,
		ExecutedAmount: 10,
		Date:           placed,
		LastUpdated:    updated,
	}}, relayed(ex), "wsFuturesProcessOpenOrders should relay an order removed as filled")

	cancelled := `{"feed":"open_orders","order_id":"660c6b23-8007-48c1-a7c9-4893f4572e8c","is_cancel":true,"reason":"cancelled_by_user"}`
	require.NoError(t, ex.wsFuturesProcessOpenOrders(t.Context(), nil, wsFuturesFeedOpenOrders, []byte(cancelled)), "wsFuturesProcessOpenOrders must not error for a cancellation")
	assert.Equal(t, []any{&order.Detail{Exchange: t.Name(), AssetType: asset.Futures, OrderID: "660c6b23-8007-48c1-a7c9-4893f4572e8c", Status: order.Cancelled}}, relayed(ex), "wsFuturesProcessOpenOrders should relay a cancelled order by its ID alone")

	// Orders on a contract missing from the available pairs, such as an expired one, and of a type that cannot be decoded
	// come ahead of an order on a listed contract
	unrelayable := `{"feed":"open_orders_snapshot","account":"DemoUser","orders":[` + strings.Replace(futuresOpenOrder("unlisted", 5, 2), `"PF_XBTUSD"`, `"FF_XBTUSD_250926"`, 1) + `,` +
		strings.Replace(futuresOpenOrder("undecoded", 5, 0), `"type":"limit"`, `"type":"moon"`, 1) + `,` + futuresOpenOrder("listed", 10, 4) + `]}`
	err := ex.wsFuturesProcessOpenOrders(t.Context(), nil, wsFuturesFeedOpenOrdersSnapshot, []byte(unrelayable))
	assert.ErrorIs(t, err, currency.ErrPairNotFound, "wsFuturesProcessOpenOrders should return the error of an order on a contract missing from the available pairs")
	assert.ErrorIs(t, err, errUnknownOrderType, "wsFuturesProcessOpenOrders should return the error of an order of a type it cannot decode")
	assert.Equal(t, []any{&order.Detail{
		Exchange:        t.Name(),
		AssetType:       asset.Futures,
		OrderID:         "listed",
		Pair:            futuresTestPair,
		Side:            order.Buy,
		Type:            order.Limit,
		Status:          order.PartiallyFilled,
		Price:           20000,
		Amount:          10,
		ExecutedAmount:  4,
		RemainingAmount: 6,
		Date:            placed,
		LastUpdated:     updated,
	}}, relayed(ex), "wsFuturesProcessOpenOrders should relay a snapshot's order after ones it cannot relay")
	assert.Equal(t, 5.0, ex.futuresOrders.get("unlisted").quantity, "wsFuturesProcessOpenOrders should record an order it cannot relay")

	for _, tc := range []struct {
		name, feed, msg string
		err             error
	}{
		{"snapshot instrument", wsFuturesFeedOpenOrdersSnapshot, `{"feed":"open_orders_snapshot","orders":[{"instrument":"PF_MOONUSD","type":"limit"}]}`, currency.ErrPairNotFound},
		{"update order type", wsFuturesFeedOpenOrders, `{"feed":"open_orders","order":{"instrument":"PF_XBTUSD","type":"moon"}}`, errUnknownOrderType},
	} {
		assert.ErrorIsf(t, ex.wsFuturesProcessOpenOrders(t.Context(), nil, tc.feed, []byte(tc.msg)), tc.err, "wsFuturesProcessOpenOrders should reject a %s it cannot convert", tc.name)
	}
	assert.Error(t, ex.wsFuturesProcessOpenOrders(t.Context(), nil, wsFuturesFeedOpenOrdersSnapshot, []byte(`{"feed":"open_orders_snapshot","orders":7}`)), "wsFuturesProcessOpenOrders should reject a snapshot it cannot decode")
	assert.Error(t, ex.wsFuturesProcessOpenOrders(t.Context(), nil, wsFuturesFeedOpenOrders, []byte(`{"feed":"open_orders","order":7}`)), "wsFuturesProcessOpenOrders should reject an update it cannot decode")
	assert.Empty(t, relayed(ex), "wsFuturesProcessOpenOrders should relay no orders from a message it rejects")

	t.Run("relay full", func(t *testing.T) {
		t.Parallel()
		ex := newTestExchange(t)
		ex.Websocket.DataHandler = stream.NewRelay(1)
		require.NoError(t, ex.Websocket.DataHandler.Send(t.Context(), "earlier payload"), "Send must not error")
		snapshot := `{"feed":"open_orders_snapshot","account":"DemoUser","orders":[` + futuresOpenOrder("first", 10, 0) + `,` + futuresOpenOrder("second", 10, 0) + `]}`
		var errs interface{ Unwrap() []error }
		require.ErrorAs(t, ex.wsFuturesProcessOpenOrders(t.Context(), nil, wsFuturesFeedOpenOrdersSnapshot, []byte(snapshot)), &errs, "wsFuturesProcessOpenOrders must return the relay's errors")
		assert.Len(t, errs.Unwrap(), 2, "wsFuturesProcessOpenOrders should try to relay each order of a snapshot")
	})
}

func TestFuturesWebsocketOrderDetail(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	placed, updated := time.UnixMilli(1612275024153), time.UnixMilli(1612275178153)
	stop := &WebsocketFuturesOpenOrder{
		Instrument:     "PF_XBTUSD",
		Time:           types.Time(placed),
		LastUpdateTime: types.Time(updated),
		Quantity:       1000,
		Filled:         250,
		LimitPrice:     34900,
		StopPrice:      13789,
		Type:           "stop",
		OrderID:        "723ba95f-13b7-418b-8fcf-ab7ba6620555",
		ClientOrderID:  "my-stop",
		Direction:      1,
		ReduceOnly:     true,
	}
	d, err := ex.futuresWsOrderDetail(stop, order.PartiallyFilled)
	require.NoError(t, err, "futuresWsOrderDetail must not error")
	assert.Equal(t, &order.Detail{
		Exchange:        t.Name(),
		AssetType:       asset.Futures,
		OrderID:         "723ba95f-13b7-418b-8fcf-ab7ba6620555",
		ClientOrderID:   "my-stop",
		Pair:            futuresTestPair,
		Side:            order.Sell,
		Type:            order.StopLimit,
		Status:          order.PartiallyFilled,
		ReduceOnly:      true,
		Price:           34900,
		TriggerPrice:    13789,
		Amount:          1000,
		ExecutedAmount:  250,
		RemainingAmount: 750,
		Date:            placed,
		LastUpdated:     updated,
	}, d, "futuresWsOrderDetail should convert a sell stop with a limit price to a stop limit order")
	for _, tc := range []struct {
		orderType  string
		limitPrice float64
		expType    order.Type
		expTIF     order.TimeInForce
	}{
		{"stop", 0, order.Stop, order.UnknownTIF},
		{"take_profit", 35000, order.TakeProfit | order.Limit, order.UnknownTIF},
		{"take_profit", 0, order.TakeProfit, order.UnknownTIF},
		{"trailing_stop", 0, order.TrailingStop, order.UnknownTIF},
		{"limit", 34900, order.Limit, order.UnknownTIF},
		{"post", 34900, order.Limit, order.PostOnly},
	} {
		d, err := ex.futuresWsOrderDetail(&WebsocketFuturesOpenOrder{Instrument: "PF_XBTUSD", Type: tc.orderType, LimitPrice: tc.limitPrice}, order.Open)
		require.NoErrorf(t, err, "futuresWsOrderDetail must not error for a %s order", tc.orderType)
		assert.Equalf(t, tc.expType, d.Type, "futuresWsOrderDetail should convert the type of a %s order with a limit price of %v", tc.orderType, tc.limitPrice)
		assert.Equalf(t, tc.expTIF, d.TimeInForce, "futuresWsOrderDetail should convert the time in force of a %s order", tc.orderType)
		assert.Equalf(t, order.Buy, d.Side, "futuresWsOrderDetail should convert a %s order in direction 0 to a buy", tc.orderType)
	}
	_, err = ex.futuresWsOrderDetail(&WebsocketFuturesOpenOrder{Instrument: "PF_MOONUSD", Type: "limit"}, order.Open)
	assert.ErrorIs(t, err, currency.ErrPairNotFound, "futuresWsOrderDetail should reject an unknown instrument")
	_, err = ex.futuresWsOrderDetail(&WebsocketFuturesOpenOrder{Instrument: "PF_XBTUSD", Type: "moon"}, order.Open)
	assert.ErrorIs(t, err, errUnknownOrderType, "futuresWsOrderDetail should reject an unknown order type")
}

func TestWebsocketFuturesProcessFills(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	seedTestAssetNames(ex)
	ex.Features.Enabled.FillsFeed = false
	fillTime := time.UnixMilli(1600256966528)
	msg := []byte(`{"feed":"fills","account":"DemoUser","fills":[{"instrument":"PF_XBTUSD","time":1600256966528,"price":10935.5,"seq":38,"buy":false,"qty":2500.0,"remaining_order_qty":0.0,"order_id":"3696d19b-3226-46bd-993d-a9a7aacc8fbc","cli_ord_id":"my-stop","fill_id":"c14ee7cb-ad6b-4f25-b0c0-7e0b4e7a4e3b","fill_type":"taker","fee_paid":0.000456,"fee_currency":"BTC","taker_order_type":"stop","order_type":"stop"}]}`)
	// Without an open_orders snapshot, the fill could follow fills of the order it never saw, so it gives the trade
	expDetail := &order.Detail{
		Exchange:      t.Name(),
		AssetType:     asset.Futures,
		OrderID:       "3696d19b-3226-46bd-993d-a9a7aacc8fbc",
		ClientOrderID: "my-stop",
		Pair:          futuresTestPair,
		Side:          order.Sell,
		Type:          order.Stop,
		FeeAsset:      currency.XBT,
		LastUpdated:   fillTime,
		Trades: []order.TradeHistory{{
			TID:       "c14ee7cb-ad6b-4f25-b0c0-7e0b4e7a4e3b",
			Price:     10935.5,
			Amount:    2500,
			Fee:       0.000456,
			FeeAsset:  "XBT",
			Exchange:  t.Name(),
			Type:      order.Stop,
			Side:      order.Sell,
			Timestamp: fillTime,
		}},
	}
	require.NoError(t, ex.wsFuturesProcessFills(t.Context(), nil, wsFuturesFeedFills, msg), "wsFuturesProcessFills must not error with the fills feed disabled")
	assert.Equal(t, []any{expDetail}, relayed(ex), "wsFuturesProcessFills should relay a fill as its order's trade alone with the fills feed disabled")

	ex.Features.Enabled.FillsFeed = true
	ex.Websocket.Fills.Setup(true, ex.Websocket.DataHandler)
	require.NoError(t, ex.wsFuturesProcessFills(t.Context(), nil, wsFuturesFeedFills, msg), "wsFuturesProcessFills must not error with the fills feed enabled")
	expFill := []fill.Data{{
		ID:            "c14ee7cb-ad6b-4f25-b0c0-7e0b4e7a4e3b",
		Timestamp:     fillTime,
		Exchange:      t.Name(),
		AssetType:     asset.Futures,
		CurrencyPair:  futuresTestPair,
		Side:          order.Sell,
		OrderID:       "3696d19b-3226-46bd-993d-a9a7aacc8fbc",
		ClientOrderID: "my-stop",
		TradeID:       "c14ee7cb-ad6b-4f25-b0c0-7e0b4e7a4e3b",
		Price:         10935.5,
		Amount:        2500,
	}}
	assert.Equal(t, []any{expDetail, expFill}, relayed(ex), "wsFuturesProcessFills should relay a fill as a fill too with the fills feed enabled")

	for _, tc := range []struct {
		name, msg string
		err       error
	}{
		{"instrument", `{"feed":"fills","fills":[{"instrument":"PF_MOONUSD","order_type":"lmt"}]}`, currency.ErrPairNotFound},
		{"order type", `{"feed":"fills","fills":[{"instrument":"PF_XBTUSD","order_type":"moon"}]}`, errUnknownOrderType},
	} {
		assert.ErrorIsf(t, ex.wsFuturesProcessFills(t.Context(), nil, wsFuturesFeedFills, []byte(tc.msg)), tc.err, "wsFuturesProcessFills should reject a fill by its %s", tc.name)
	}
	assert.Error(t, ex.wsFuturesProcessFills(t.Context(), nil, wsFuturesFeedFills, []byte(`{"feed":"fills","fills":7}`)), "wsFuturesProcessFills should reject fills it cannot decode")
	assert.Empty(t, relayed(ex), "wsFuturesProcessFills should relay nothing from a message it rejects")

	t.Run("fills it cannot relay first", func(t *testing.T) {
		t.Parallel()
		ex := newTestExchange(t)
		seedTestAssetNames(ex)
		ex.Features.Enabled.FillsFeed = true
		ex.Websocket.Fills.Setup(true, ex.Websocket.DataHandler)
		// A snapshot lists the oldest fills first, which can be on a contract missing from the available pairs, such as an
		// expired one
		snapshot := `{"feed":"fills_snapshot","account":"DemoUser","fills":[` + futuresFill("unlisted", "f-1", "FF_XBTUSD_250926", 2, 3, 20000, 0.25, "USD") + `,` +
			strings.Replace(futuresFill("undecoded", "f-2", "PF_XBTUSD", 1, 0, 20000, 0.25, "USD"), `"order_type":"lmt"`, `"order_type":"moon"`, 1) + `,` +
			futuresFill("listed", "f-3", "PF_XBTUSD", 4, 6, 20000, 0.5, "USD") + `]}`
		err := ex.wsFuturesProcessFills(t.Context(), nil, wsFuturesFeedFillsSnapshot, []byte(snapshot))
		assert.ErrorIs(t, err, currency.ErrPairNotFound, "wsFuturesProcessFills should return the error of a fill on a contract missing from the available pairs")
		assert.ErrorIs(t, err, errUnknownOrderType, "wsFuturesProcessFills should return the error of a fill of an order type it cannot decode")
		fillTime := time.UnixMilli(1600256966528)
		exp := []any{
			&order.Detail{
				Exchange:    ex.Name,
				AssetType:   asset.Futures,
				OrderID:     "listed",
				Pair:        futuresTestPair,
				Side:        order.Buy,
				Type:        order.Limit,
				FeeAsset:    currency.USD,
				LastUpdated: fillTime,
				Trades:      []order.TradeHistory{{TID: "f-3", Price: 20000, Amount: 4, Fee: 0.5, FeeAsset: "USD", Exchange: ex.Name, Type: order.Limit, Side: order.Buy, Timestamp: fillTime}},
			},
			[]fill.Data{{ID: "f-3", Timestamp: fillTime, Exchange: ex.Name, AssetType: asset.Futures, CurrencyPair: futuresTestPair, Side: order.Buy, OrderID: "listed", TradeID: "f-3", Price: 20000, Amount: 4}},
		}
		assert.Equal(t, exp, relayed(ex), "wsFuturesProcessFills should relay a fill after ones it cannot relay, as its order's trade and as a fill")
		assert.Equal(t, 4.0, ex.futuresOrders.get("listed").fillQuantity, "wsFuturesProcessFills should count a fill after ones it cannot relay into its order's totals")
		assert.Equal(t, 2.0, ex.futuresOrders.get("unlisted").fillQuantity, "wsFuturesProcessFills should count a fill it cannot relay into its order's totals")
	})

	t.Run("relay full", func(t *testing.T) {
		t.Parallel()
		ex := newTestExchange(t)
		seedTestAssetNames(ex)
		ex.Websocket.DataHandler = stream.NewRelay(1)
		require.NoError(t, ex.Websocket.DataHandler.Send(t.Context(), "earlier payload"), "Send must not error")
		ex.Features.Enabled.FillsFeed = true
		ex.Websocket.Fills.Setup(true, ex.Websocket.DataHandler)
		msg := `{"feed":"fills","account":"DemoUser","fills":[` + futuresFill("first", "f-1", "PF_XBTUSD", 4, 6, 20000, 0.25, "USD") + `,` +
			futuresFill("second", "f-2", "PF_XBTUSD", 6, 0, 20000, 0.25, "USD") + `]}`
		var errs interface{ Unwrap() []error }
		require.ErrorAs(t, ex.wsFuturesProcessFills(t.Context(), nil, wsFuturesFeedFills, []byte(msg)), &errs, "wsFuturesProcessFills must return the relay's errors")
		assert.Len(t, errs.Unwrap(), 3, "wsFuturesProcessFills should try to relay each fill as its order's trade, and the fills to the fills feed")
		assert.Equal(t, 6.0, ex.futuresOrders.get("second").fillQuantity, "wsFuturesProcessFills should count a fill after one it could not relay into its order's totals")
	})

	t.Run("contract listed after a fill", func(t *testing.T) {
		t.Parallel()
		ex := newTestExchange(t)
		seedTestAssetNames(ex)
		ex.Features.Enabled.FillsFeed = false
		listedLater := currency.NewPairWithDelimiter("PF", "GCTUSD", currency.UnderscoreDelimiter)
		snapshot := `{"feed":"open_orders_snapshot","account":"DemoUser","orders":[` + strings.Replace(futuresOpenOrder("o-1", 5, 0), `"PF_XBTUSD"`, `"PF_GCTUSD"`, 1) + `]}`
		assert.ErrorIs(t, ex.wsFuturesProcessOpenOrders(t.Context(), nil, wsFuturesFeedOpenOrdersSnapshot, []byte(snapshot)), currency.ErrPairNotFound, "wsFuturesProcessOpenOrders should return the error of an order on a contract missing from the available pairs")
		earlier := futuresFillMessage(wsFuturesFeedFills, "o-1", "f-1", "PF_GCTUSD", 2, 3, 100, 0.25, "USD")
		assert.ErrorIs(t, ex.wsFuturesProcessFills(t.Context(), nil, wsFuturesFeedFills, []byte(earlier)), currency.ErrPairNotFound, "wsFuturesProcessFills should return the error of a fill on a contract missing from the available pairs")
		assert.Empty(t, relayed(ex), "nothing should be relayed for a contract missing from the available pairs")
		pairs, err := ex.CurrencyPairs.GetPairs(asset.Futures, false)
		require.NoError(t, err, "GetPairs must not error")
		require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Futures, append(pairs, listedLater), false), "StorePairs must not error")
		later := futuresFillMessage(wsFuturesFeedFills, "o-1", "f-2", "PF_GCTUSD", 3, 0, 200, 0.5, "USD")
		require.NoError(t, ex.wsFuturesProcessFills(t.Context(), nil, wsFuturesFeedFills, []byte(later)), "wsFuturesProcessFills must not error once the contract is listed")
		fillTime := time.UnixMilli(1600256966528)
		exp := &order.Detail{
			Exchange:             ex.Name,
			AssetType:            asset.Futures,
			OrderID:              "o-1",
			Pair:                 listedLater,
			Side:                 order.Buy,
			Type:                 order.Limit,
			Status:               order.Filled,
			Amount:               5,
			ExecutedAmount:       5,
			Cost:                 800,
			AverageExecutedPrice: 160,
			Fee:                  0.75,
			FeeAsset:             currency.USD,
			LastUpdated:          fillTime,
			Trades:               []order.TradeHistory{{TID: "f-2", Price: 200, Amount: 3, Fee: 0.5, FeeAsset: "USD", Exchange: ex.Name, Type: order.Limit, Side: order.Buy, Timestamp: fillTime}},
		}
		assert.Equal(t, []any{exp}, relayed(ex), "wsFuturesProcessFills should relay the order's totals with the fill it could not relay before its contract was listed")
	})

	t.Run("order type it cannot decode before a later fill", func(t *testing.T) {
		t.Parallel()
		ex := newTestExchange(t)
		seedTestAssetNames(ex)
		ex.Features.Enabled.FillsFeed = false
		snapshot := `{"feed":"open_orders_snapshot","account":"DemoUser","orders":[` + futuresOpenOrder("o-1", 5, 0) + `]}`
		require.NoError(t, ex.wsFuturesProcessOpenOrders(t.Context(), nil, wsFuturesFeedOpenOrdersSnapshot, []byte(snapshot)), "wsFuturesProcessOpenOrders must not error")
		relayed(ex)
		earlier := strings.Replace(futuresFillMessage(wsFuturesFeedFills, "o-1", "f-1", "PF_XBTUSD", 2, 3, 100, 0.25, "USD"), `"order_type":"lmt"`, `"order_type":"moon"`, 1)
		assert.ErrorIs(t, ex.wsFuturesProcessFills(t.Context(), nil, wsFuturesFeedFills, []byte(earlier)), errUnknownOrderType, "wsFuturesProcessFills should return the error of a fill of an order type it cannot decode")
		assert.Empty(t, relayed(ex), "nothing should be relayed for a fill of an order type it cannot decode")
		later := futuresFillMessage(wsFuturesFeedFills, "o-1", "f-2", "PF_XBTUSD", 3, 0, 200, 0.5, "USD")
		require.NoError(t, ex.wsFuturesProcessFills(t.Context(), nil, wsFuturesFeedFills, []byte(later)), "wsFuturesProcessFills must not error for a fill it can decode")
		fillTime := time.UnixMilli(1600256966528)
		exp := &order.Detail{
			Exchange:             ex.Name,
			AssetType:            asset.Futures,
			OrderID:              "o-1",
			Pair:                 futuresTestPair,
			Side:                 order.Buy,
			Type:                 order.Limit,
			Status:               order.Filled,
			Amount:               5,
			ExecutedAmount:       5,
			Cost:                 800,
			AverageExecutedPrice: 160,
			Fee:                  0.75,
			FeeAsset:             currency.USD,
			LastUpdated:          fillTime,
			Trades:               []order.TradeHistory{{TID: "f-2", Price: 200, Amount: 3, Fee: 0.5, FeeAsset: "USD", Exchange: ex.Name, Type: order.Limit, Side: order.Buy, Timestamp: fillTime}},
		}
		assert.Equal(t, []any{exp}, relayed(ex), "wsFuturesProcessFills should relay the order's totals with the fill of an order type it could not decode")
	})
}

func TestWebsocketFuturesProcessBalances(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	useTestCredentials(ex)
	msg := `{"feed":"balances","account":"7a641082-55c7-4411-a85f-930ec2e09617","holding":{"usd":5000.0},"futures":{},` +
		`"flex_futures":{"currencies":{"USD":{"quantity":5000.0,"available":4000.0},"XBT":{"quantity":0.5,"available":-0.1},"ETH":{"quantity":2.0,"available":2.5}}},"timestamp":1640995200000,"seq":84}`
	require.NoError(t, ex.wsFuturesProcessBalances(t.Context(), wsFuturesFeedBalances, []byte(msg)), "wsFuturesProcessBalances must not error")
	got := relayed(ex)
	require.Len(t, got, 2, "wsFuturesProcessBalances must relay the wallets and the message")
	assert.Equal(t, accounts.SubAccounts{
		{ID: "cash", AssetType: asset.Futures, Balances: accounts.CurrencyBalances{currency.USD: {Currency: currency.USD, Total: 5000, Free: 5000}}},
		{ID: "flex", AssetType: asset.Futures, Balances: accounts.CurrencyBalances{
			// Margin in use holds the rest of a currency, and unrealised profit cannot free more than the currency held
			currency.USD: {Currency: currency.USD, Total: 5000, Hold: 1000, Free: 4000},
			currency.XBT: {Currency: currency.XBT, Total: 0.5, Hold: 0.5},
			currency.ETH: {Currency: currency.ETH, Total: 2, Free: 2},
		}},
	}, futuresTestSubAccounts(t, got[0]), "wsFuturesProcessBalances should relay the cash and multi-collateral wallets")
	b, ok := got[1].(*WebsocketFuturesBalances)
	require.True(t, ok, "wsFuturesProcessBalances must relay the message")
	assert.Equal(t, uint64(84), b.Sequence, "wsFuturesProcessBalances should relay the message as sent")
	stored, err := ex.Accounts.GetBalance("flex", ex.GetDefaultCredentials(), asset.Futures, currency.XBT)
	require.NoError(t, err, "GetBalance must not error")
	stored.UpdatedAt = time.Time{}
	assert.Equal(t, accounts.Balance{Currency: currency.XBT, Total: 0.5, Hold: 0.5}, stored, "wsFuturesProcessBalances should store the wallets' balances")

	require.NoError(t, ex.wsFuturesProcessBalances(t.Context(), wsFuturesFeedBalances, fixtureLines(t, "testdata/wsFutures.json")[16]), "wsFuturesProcessBalances must not error for a margin account update")
	got = relayed(ex)
	require.Len(t, got, 1, "wsFuturesProcessBalances must relay a margin account update as the message alone")
	assert.IsType(t, &WebsocketFuturesBalances{}, got[0], "wsFuturesProcessBalances should relay a margin account update as the message")
	assert.Error(t, ex.wsFuturesProcessBalances(t.Context(), wsFuturesFeedBalances, []byte(`{"feed":"balances","holding":7}`)), "wsFuturesProcessBalances should reject balances it cannot decode")

	noCreds := newTestExchange(t)
	noCreds.SkipAuthCheck = false
	noCreds.API.AuthenticatedSupport = true
	noCreds.SetCredentials(&accounts.Credentials{})
	assert.ErrorIs(t, noCreds.wsFuturesProcessBalances(t.Context(), wsFuturesFeedBalances, []byte(`{"feed":"balances","holding":{"usd":1}}`)), exchange.ErrCredentialsAreEmpty, "wsFuturesProcessBalances should return the error storing the balances")
	assert.Empty(t, relayed(noCreds), "wsFuturesProcessBalances should relay nothing of balances it could not store")

	ex.Websocket.DataHandler = stream.NewRelay(1)
	require.NoError(t, ex.Websocket.DataHandler.Send(t.Context(), struct{}{}), "DataHandler must accept the saturation payload")
	err = ex.wsFuturesProcessBalances(t.Context(), wsFuturesFeedBalances, []byte(`{"feed":"balances","holding":{"usd":2}}`))
	assert.ErrorContains(t, err, "channel buffer is full: failed to relay <accounts.SubAccounts>", "wsFuturesProcessBalances should return the error relaying the wallets")
}
