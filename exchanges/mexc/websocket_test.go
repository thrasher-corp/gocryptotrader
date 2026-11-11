package mexc

import (
	"cmp"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/mexc/mexc_proto_types"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"google.golang.org/protobuf/proto"
)

func TestAssetTypeToString(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "spot", assetTypeToString(asset.Spot))
	assert.Empty(t, assetTypeToString(asset.Margin))
}

func TestChannelName(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		asset    asset.Item
		channel  string
		expected string
	}{
		{asset.Spot, subscription.TickerChannel, channelBookTiker},
		{asset.Spot, subscription.OrderbookChannel, channelLimitDepthV3},
		{asset.Spot, subscription.AllTradesChannel, channelAggreDealsV3},
		{asset.Spot, subscription.CandlesChannel, channelKlineV3},
		{asset.Spot, subscription.MyTradesChannel, channelPrivateDealsV3},
		{asset.Spot, subscription.MyOrdersChannel, channelPrivateOrdersAPI},
		{asset.Spot, subscription.MyAccountChannel, channelAccountV3},
		{asset.Spot, subscription.HeartbeatChannel, subscription.HeartbeatChannel},
	} {
		assert.Equalf(t, tc.expected, channelName(&subscription.Subscription{Asset: tc.asset, Channel: tc.channel}), "channelName should return correct channel for %s %s", tc.asset, tc.channel)
	}
	assert.Equal(t, subscription.TickerChannel, channelName(&subscription.Subscription{Asset: asset.Margin, Channel: subscription.TickerChannel}),
		"an unsupported asset should fall through to the raw channel name")
}

// wsTestSymbol is the only pair the mock exchange enables, so every test frame carries it.
const wsTestSymbol = "BTCUSDT"

// wsPushFrame builds a protobuf push frame the way MEXC sends it: the qualified channel is the
// first field, which is also what WsHandleData routes on.
func wsPushFrame(tb testing.TB, channel string, sendTime int64, body isPushBody) []byte {
	tb.Helper()
	return wsPushFrameForSymbol(tb, wsTestSymbol, channel, sendTime, body)
}

// wsPushFrameForSymbol builds a push frame for an explicit symbol, for the cases where the symbol
// itself is the subject of the test.
func wsPushFrameForSymbol(tb testing.TB, symbol, channel string, sendTime int64, body isPushBody) []byte {
	tb.Helper()
	w := &mexc_proto_types.PushDataV3ApiWrapper{
		Channel:  channel,
		Symbol:   &symbol,
		SendTime: &sendTime,
	}
	switch b := body.(type) {
	case *mexc_proto_types.PublicAggreBookTickerV3Api:
		w.Body = &mexc_proto_types.PushDataV3ApiWrapper_PublicAggreBookTicker{PublicAggreBookTicker: b}
	case *mexc_proto_types.PublicMiniTickerV3Api:
		w.Body = &mexc_proto_types.PushDataV3ApiWrapper_PublicMiniTicker{PublicMiniTicker: b}
	case *mexc_proto_types.PublicMiniTickersV3Api:
		w.Body = &mexc_proto_types.PushDataV3ApiWrapper_PublicMiniTickers{PublicMiniTickers: b}
	case *mexc_proto_types.PublicAggreDealsV3Api:
		w.Body = &mexc_proto_types.PushDataV3ApiWrapper_PublicAggreDeals{PublicAggreDeals: b}
	case *mexc_proto_types.PublicSpotKlineV3Api:
		w.Body = &mexc_proto_types.PushDataV3ApiWrapper_PublicSpotKline{PublicSpotKline: b}
	case *mexc_proto_types.PublicAggreDepthsV3Api:
		w.Body = &mexc_proto_types.PushDataV3ApiWrapper_PublicAggreDepths{PublicAggreDepths: b}
	case *mexc_proto_types.PublicLimitDepthsV3Api:
		w.Body = &mexc_proto_types.PushDataV3ApiWrapper_PublicLimitDepths{PublicLimitDepths: b}
	case *mexc_proto_types.PublicIncreaseDepthsBatchV3Api:
		w.Body = &mexc_proto_types.PushDataV3ApiWrapper_PublicIncreaseDepthsBatch{PublicIncreaseDepthsBatch: b}
	case *mexc_proto_types.PublicBookTickerBatchV3Api:
		w.Body = &mexc_proto_types.PushDataV3ApiWrapper_PublicBookTickerBatch{PublicBookTickerBatch: b}
	case *mexc_proto_types.PrivateAccountV3Api:
		w.Body = &mexc_proto_types.PushDataV3ApiWrapper_PrivateAccount{PrivateAccount: b}
	case *mexc_proto_types.PrivateDealsV3Api:
		w.Body = &mexc_proto_types.PushDataV3ApiWrapper_PrivateDeals{PrivateDeals: b}
	case *mexc_proto_types.PrivateOrdersV3Api:
		w.Body = &mexc_proto_types.PushDataV3ApiWrapper_PrivateOrders{PrivateOrders: b}
	default:
		tb.Fatalf("unsupported body type %T", body)
	}
	raw, err := proto.Marshal(w)
	require.NoError(tb, err, "proto.Marshal must not error")
	return raw
}

type isPushBody any

// drainTickers returns every ticker.Price relayed to the data handler so far
func drainTickers(tb testing.TB) []*ticker.Price {
	tb.Helper()
	var out []*ticker.Price
	for {
		select {
		case p := <-e.Websocket.DataHandler.C:
			if tick, ok := p.Data.(*ticker.Price); ok {
				out = append(out, tick)
			}
		default:
			return out
		}
	}
}

// TestWsSpotTickerFromBookTicker asserts the spot best bid/offer reaches the ticker and not only
// the orderbook: the websocket ticker used to be silently empty (zero SPOT TICKER lines in 15 min).
func TestWsSpotTickerFromBookTicker(t *testing.T) {
	drainTickers(t)
	raw := wsPushFrame(t, "spot@"+channelBookTiker+"@100ms@BTCUSDT", 1736412092433,
		&mexc_proto_types.PublicAggreBookTickerV3Api{
			BidPrice: "93387.28", BidQuantity: "3.73485",
			AskPrice: "93387.29", AskQuantity: "7.669875",
		})
	require.NoError(t, e.WsHandleData(t.Context(), nil, raw), "WsHandleData must not error")

	ticks := drainTickers(t)
	require.Len(t, ticks, 1, "exactly one ticker must be published")
	got := ticks[0]
	assert.Equal(t, 93387.28, got.Bid, "Bid should be correct")
	assert.Equal(t, 3.73485, got.BidSize, "BidSize should be correct")
	assert.Equal(t, 93387.29, got.Ask, "Ask should be correct")
	assert.Equal(t, 7.669875, got.AskSize, "AskSize should be correct")
	assert.Equal(t, asset.Spot, got.AssetType, "AssetType should be correct")
	assert.Equal(t, e.Name, got.ExchangeName, "ExchangeName should be correct")
	assert.Equal(t, int64(1736412092433), got.LastUpdated.UnixMilli(), "LastUpdated should come from the exchange send time")
}

// TestWsSpotTickerFromMiniTicker asserts last/high/low/volume arrive over the websocket and that the
// two spot ticker channels merge instead of blanking each other's fields.
func TestWsSpotTickerFromMiniTicker(t *testing.T) {
	drainTickers(t)
	bookRaw := wsPushFrame(t, "spot@"+channelBookTiker+"@100ms@BTCUSDT", 1736412092433,
		&mexc_proto_types.PublicAggreBookTickerV3Api{
			BidPrice: "93387.28", BidQuantity: "3.73485",
			AskPrice: "93387.29", AskQuantity: "7.669875",
		})
	require.NoError(t, e.WsHandleData(t.Context(), nil, bookRaw), "WsHandleData must not error")

	miniRaw := wsPushFrame(t, "spot@"+channelMiniTickerV3+"@BTCUSDT@"+miniTickerTimezone, 1736412092500,
		&mexc_proto_types.PublicMiniTickerV3Api{
			Symbol: "BTCUSDT", Price: "93390.11", High: "94000.5", Low: "92000.25",
			Volume: "323169.867864", Quantity: "12058672.07",
		})
	require.NoError(t, e.WsHandleData(t.Context(), nil, miniRaw), "WsHandleData must not error")

	ticks := drainTickers(t)
	require.Len(t, ticks, 2, "both channels must publish a ticker")
	got := ticks[1]
	assert.Equal(t, 93390.11, got.Last, "Last should be correct")
	assert.Equal(t, 94000.5, got.High, "High should be correct")
	assert.Equal(t, 92000.25, got.Low, "Low should be correct")
	assert.Equal(t, 12058672.07, got.BaseVolume, "BaseVolume should be the base asset volume (miniTicker quantity)")
	assert.Equal(t, 323169.867864, got.QuoteVolume, "QuoteVolume should be the quote volume (miniTicker volume)")
	assert.Equal(t, 93387.28, got.Bid, "Bid from the bookTicker channel should survive a miniTicker update")
	assert.Equal(t, 93387.29, got.Ask, "Ask from the bookTicker channel should survive a miniTicker update")
	assert.Equal(t, int64(1736412092500), got.LastUpdated.UnixMilli(), "LastUpdated should come from the exchange send time")
}

// TestWsMiniTickerVolumePresence pins that an explicit zero volume is applied and an omitted one keeps
// the last known figure: idle symbols report "0", which setIfNonZero would discard.
func TestWsMiniTickerVolumePresence(t *testing.T) {
	drainTickers(t)
	for i, tc := range []struct {
		volume, quantity    string
		wantQuote, wantBase float64
	}{
		{"323169.867864", "12058672.07", 323169.867864, 12058672.07},
		{"0", "0", 0, 0},
		{"777", "888", 777, 888},
		{"", "", 777, 888},
	} {
		raw := wsPushFrame(t, "spot@"+channelMiniTickerV3+"@BTCUSDT@"+miniTickerTimezone, 1736412092500+int64(i),
			&mexc_proto_types.PublicMiniTickerV3Api{Symbol: "BTCUSDT", Price: "93390.11", High: "94000.5", Low: "92000.25", Volume: tc.volume, Quantity: tc.quantity})
		require.NoError(t, e.WsHandleData(t.Context(), nil, raw), "WsHandleData must not error")
		ticks := drainTickers(t)
		require.Len(t, ticks, 1, "exactly one ticker must be published")
		assert.Equalf(t, tc.wantBase, ticks[0].BaseVolume, "frame %d base volume should match", i)
		assert.Equalf(t, tc.wantQuote, ticks[0].QuoteVolume, "frame %d quote volume should match", i)
	}
}

// TestWsHandleDataUndecodableFrame asserts a binary frame that is not a valid push frame is
// reported rather than silently dropped. It used to be answered with an unhandled-message warning,
// because the handler routed on the raw bytes and merely failed to find a separator in them; a
// frame that cannot be decoded and a frame whose channel has no mapping are different faults, and
// reporting both as "unhandled" is what kept the unroutable private channels invisible.
func TestWsHandleDataUndecodableFrame(t *testing.T) {
	t.Parallel()
	assert.Error(t, e.WsHandleData(t.Context(), nil, []byte("no-separator-here")), "an undecodable frame should be reported as a decode error")
}

func TestChannelSuffix(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "@"+miniTickerTimezone, channelSuffix(channelMiniTickerV3), "miniTicker requires a timezone suffix")
	assert.Empty(t, channelSuffix(channelBookTiker), "other channels take no suffix")
}

func TestSubscriptionAccepted(t *testing.T) {
	t.Parallel()
	const ch = "spot@public.miniTicker.v3.api.pb@BTCUSDT@UTC+8"
	assert.True(t, subscriptionAccepted("SUBSCRIPTION", ch, ch), "an echoed channel means accepted")
	assert.False(t, subscriptionAccepted("SUBSCRIPTION", ch, "Not Subscribed successfully! ["+ch+"].  Reason: Blocked! "), "a rejection carrying code 0 should not count as accepted")
	assert.False(t, subscriptionAccepted("UNSUBSCRIPTION", ch, "no subscription"), "an unsubscribe reply that does not echo the channel is a rejection")
	assert.True(t, subscriptionAccepted("UNSUBSCRIPTION", ch, ch), "an echoed channel means the unsubscribe was accepted")
}

func TestIsSpotSubscription(t *testing.T) {
	t.Parallel()
	for _, a := range []asset.Item{asset.Spot, asset.All, asset.Empty} {
		t.Run(cmp.Or(a.String(), "empty"), func(t *testing.T) {
			t.Parallel()
			assert.Truef(t, isSpotSubscription(&subscription.Subscription{Asset: a}), "a subscription for asset %q should be expanded with the spot template", a)
		})
	}
	assert.False(t, isSpotSubscription(&subscription.Subscription{Asset: asset.Futures}), "a futures subscription should not be expanded with the spot template")
}

func TestGenerateSubscriptionsIncludesMiniTicker(t *testing.T) {
	t.Parallel()
	// A fresh instance: the live websocket tests' testexch.SetupWs empties the shared one's subscriptions.
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	require.NoError(t, ex.setEnabledPairs(spotTradablePair), "setEnabledPairs must not error")
	subs, err := ex.generateSubscriptions(nil)
	require.NoError(t, err, "generateSubscriptions must not error")
	var found bool
	for _, s := range subs {
		if s.Channel != channelMiniTickerV3 {
			continue
		}
		found = true
		assert.Equal(t, "spot@"+channelMiniTickerV3+"@BTCUSDT@"+miniTickerTimezone, s.QualifiedChannel, "miniTicker should carry the mandatory timezone suffix; MEXC blocks the subscription without it")
	}
	assert.True(t, found, "the spot miniTicker channel should be subscribed")
}

func TestIsSymbolChannel(t *testing.T) {
	t.Parallel()
	assert.True(t, isSymbolChannel(channelBookTiker))
	assert.False(t, isSymbolChannel(channelAccountV3))
	assert.False(t, isSymbolChannel(channelPrivateDealsV3))
	assert.False(t, isSymbolChannel(channelPrivateOrdersAPI))
}

func TestWsIntervalString(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "Min15", wsIntervalString(&subscription.Subscription{Interval: kline.FifteenMin}))
	assert.Empty(t, wsIntervalString(&subscription.Subscription{Interval: kline.SixMonth}))
}

// TestAccountTypeMatches covers the case-insensitive account type comparison without credentials.
// The live test that caught the original defect skips when no keys are set, so the regression it
// guards against would otherwise be invisible to the standard run.
func TestAccountTypeMatches(t *testing.T) {
	t.Parallel()
	assert.True(t, accountTypeMatches("SPOT", asset.Spot), "the exchange's upper-case SPOT should match asset.Spot")
	assert.True(t, accountTypeMatches("spot", asset.Spot), "an already lower-case value should still match")
	assert.True(t, accountTypeMatches("SPOT", asset.Empty), "an empty asset should match anything")
	assert.False(t, accountTypeMatches("SPOT", asset.Futures), "a different account type should not match")
}

// TestWsSpotTickerFromMiniTickers applies each item of the all-symbols miniTickers push the way the
// per-symbol miniTicker is applied, and skips a symbol that is not tracked instead of failing the frame.
func TestWsSpotTickerFromMiniTickers(t *testing.T) {
	drainTickers(t)
	raw := wsPushFrameForSymbol(t, "", "spot@"+channelMiniTickersV3+"@"+miniTickerTimezone, 1736412093000,
		&mexc_proto_types.PublicMiniTickersV3Api{Items: []*mexc_proto_types.PublicMiniTickerV3Api{
			{Symbol: "NOTTRACKEDUSDT", Price: "1", High: "1", Low: "1", Volume: "1", Quantity: "1"},
			{Symbol: wsTestSymbol, Price: "93391.5", High: "94001", Low: "92001", Volume: "0", Quantity: "0"},
		}})
	require.NoError(t, e.WsHandleData(t.Context(), nil, raw), "WsHandleData must not error on an untracked symbol")

	ticks := drainTickers(t)
	require.Len(t, ticks, 1, "only the tracked symbol must publish a ticker")
	got := ticks[0]
	assert.Equal(t, 93391.5, got.Last, "Last should be correct")
	assert.Equal(t, 94001.0, got.High, "High should be correct")
	assert.Equal(t, 92001.0, got.Low, "Low should be correct")
	assert.Zero(t, got.BaseVolume, "an explicit zero base volume should be kept")
	assert.Zero(t, got.QuoteVolume, "an explicit zero quote volume should be kept")
	assert.Equal(t, int64(1736412093000), got.LastUpdated.UnixMilli(), "LastUpdated should come from the exchange send time")
}

// TestMiniTickersSubscription qualifies the all-symbols channel without a symbol but with the
// mandatory timezone suffix, and keeps it out of the default subscriptions.
func TestMiniTickersSubscription(t *testing.T) {
	t.Parallel()
	subs, err := subscription.List{{Enabled: true, Asset: asset.Spot, Channel: channelMiniTickersV3}}.ExpandTemplates(e)
	require.NoError(t, err, "ExpandTemplates must not error")
	require.Len(t, subs, 1, "the all-symbols channel must expand to a single subscription")
	assert.Equal(t, "spot@"+channelMiniTickersV3+"@"+miniTickerTimezone, subs[0].QualifiedChannel, "the channel should carry no symbol and the timezone suffix")
	for _, s := range defaultSubscriptions {
		assert.NotEqual(t, channelMiniTickersV3, s.Channel, "the all-symbols channel should not be a default subscription")
	}
}

func TestWebsocketDial(t *testing.T) {
	t.Parallel()
	var listenKeyRequests atomic.Int64
	ex := newSignedTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		listenKeyRequests.Add(1)
		http.Error(w, `{"code":700003,"msg":"listen key limit reached"}`, http.StatusBadRequest)
	}))
	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
	for range 2 {
		conn := &listenKeyTestConn{url: "wss://private"}
		require.NoError(t, ex.wsDial(t.Context(), conn, true), "wsDial must dial the private connection without a listen key")
		assert.Equal(t, int64(1), conn.dials.Load(), "the private connection should be dialled")
		assert.Equal(t, "wss://private", conn.url, "the private connection should be dialled without a listen key")
	}
	assert.Equal(t, int64(1), listenKeyRequests.Load(), "a refused listen key should not be asked for again")
	assert.True(t, ex.spotListenKeyRefused.Load(), "the refusal should be recorded")
	assert.True(t, ex.Websocket.CanUseAuthenticatedEndpoints(), "the authenticated websocket should stay usable, as the private futures channels log in with the API key")
}

func TestSubscribe(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		refused bool
	}{
		{name: "private channels subscribed while they can be used"},
		{name: "private channels refused once a listen key is refused", refused: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newPrivateTestExchange(t, func(_ http.ResponseWriter, r *http.Request) {
				t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			})
			ex.Features.Subscriptions = defaultSubscriptions.Clone()
			ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
			public, err := ex.generatePublicSubscriptions()
			require.NoError(t, err, "generatePublicSubscriptions must not error")
			require.NotEmpty(t, public, "the public spot channels must be generated")
			private, err := ex.generatePrivateSubscriptions()
			require.NoError(t, err, "generatePrivateSubscriptions must not error")
			require.NotEmpty(t, private, "the private spot channels must be generated")
			replies := make(map[string]string)
			for _, s := range append(subscription.List{public[0]}, private...) {
				replies[s.QualifiedChannel] = `{"id":0,"code":0,"msg":"` + s.QualifiedChannel + `"}`
			}
			conn := &subscriptionTestConn{replies: replies}
			ex.spotListenKeyRefused.Store(tc.refused)
			require.NoError(t, ex.Subscribe(t.Context(), conn, public[:1]), "Subscribe must subscribe a public spot channel")
			err = ex.Subscribe(t.Context(), conn, private)
			want := append(subscription.List{public[0]}, private...)
			if tc.refused {
				assert.ErrorIs(t, err, errNoListenKey, "the private spot channels should be refused without a listen key")
				want = want[:1]
			} else {
				assert.NoError(t, err, "the private spot channels should be subscribed")
			}
			sent := make([]string, 0, len(want))
			for _, s := range want {
				sent = append(sent, s.QualifiedChannel)
			}
			assert.ElementsMatch(t, sent, conn.sent, "only the channels subscribed should be sent")
			assert.ElementsMatch(t, want, ex.Websocket.GetSubscriptions(), "only the channels subscribed should be stored")
		})
	}
}

func TestSpotPrivateChannelsUsable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name                   string
		authenticated, refused bool
		usable                 bool
	}{
		{name: "authenticated", authenticated: true, usable: true},
		{name: "authenticated, listen key refused", authenticated: true, refused: true},
		{name: "unauthenticated, listen key refused", refused: true},
		{name: "unauthenticated"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := &Exchange{Websocket: new(websocket.Manager)}
			ex.Websocket.SetCanUseAuthenticatedEndpoints(tc.authenticated)
			ex.spotListenKeyRefused.Store(tc.refused)
			assert.Equal(t, tc.usable, ex.spotPrivateChannelsUsable(), "the private spot channels should be usable only with the authenticated websocket and no listen key refused")
		})
	}
}

func TestGeneratePrivateSubscriptions(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name                   string
		authenticated, refused bool
		channels               []string
	}{
		{name: "authenticated", authenticated: true, channels: []string{channelAccountV3, channelPrivateDealsV3, channelPrivateOrdersAPI}},
		{name: "listen key refused", authenticated: true, refused: true},
		{name: "unauthenticated"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex), "Setup must not error")
			require.NoError(t, ex.setEnabledPairs(spotTradablePair), "setEnabledPairs must not error")
			ex.Features.Subscriptions = defaultSubscriptions.Clone()
			ex.Websocket.SetCanUseAuthenticatedEndpoints(tc.authenticated)
			ex.spotListenKeyRefused.Store(tc.refused)
			subs, err := ex.generatePrivateSubscriptions()
			require.NoError(t, err, "generatePrivateSubscriptions must not error")
			channels := make([]string, 0, len(subs))
			for _, s := range subs {
				assert.Equalf(t, asset.Spot, s.Asset, "%s should be a spot channel", s.QualifiedChannel)
				channels = append(channels, wsChannelName(s.QualifiedChannel))
			}
			assert.ElementsMatch(t, tc.channels, channels, "the private spot channels should be generated only while they can be used")
		})
	}
}
