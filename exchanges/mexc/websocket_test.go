package mexc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/mexc/mexc_proto_types"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
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
	assert.Equal(t, 12058672.07, got.Volume, "Volume should be the base asset volume (miniTicker quantity)")
	assert.Equal(t, 323169.867864, got.QuoteVolume, "QuoteVolume should be the quote volume (miniTicker volume)")
	assert.Equal(t, 93387.28, got.Bid, "Bid from the bookTicker channel should survive a miniTicker update")
	assert.Equal(t, 93387.29, got.Ask, "Ask from the bookTicker channel should survive a miniTicker update")
	assert.Equal(t, int64(1736412092500), got.LastUpdated.UnixMilli(), "LastUpdated should come from the exchange send time")
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
	assert.False(t, subscriptionAccepted("SUBSCRIPTION", ch, "Not Subscribed successfully! ["+ch+"].  Reason： Blocked! "), "a rejection carrying code 0 should not count as accepted")
	assert.True(t, subscriptionAccepted("UNSUBSCRIPTION", ch, "no subscription"), "unsubscribe responses are not channel echoes")
}

func TestGenerateSubscriptionsIncludesMiniTicker(t *testing.T) {
	t.Parallel()
	subs, err := e.generateSubscriptions()
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
	assert.True(t, accountTypeMatches("SPOT", asset.Spot), "the exchange's upper-case SPOT must match asset.Spot")
	assert.True(t, accountTypeMatches("spot", asset.Spot), "an already lower-case value must still match")
	assert.True(t, accountTypeMatches("SPOT", asset.Empty), "an empty asset must match anything")
	assert.False(t, accountTypeMatches("SPOT", asset.Futures), "a different account type must not match")
}
