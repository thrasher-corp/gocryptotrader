package htx

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	mockws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
)

func TestWSConnect(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	h.SetEnabled(false)
	err := h.wsConnect(t.Context(), nil)
	require.ErrorIs(t, err, websocket.ErrWebsocketNotEnabled, "wsConnect must reject a disabled exchange")
}

func TestGenerateSubscriptionsForAsset(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	h.Features.Subscriptions = defaultFuturesSubscriptions.Clone()
	require.NoError(t, h.SetPairs(currency.Pairs{btcusdPair}, asset.CoinMarginedFutures, true), "enabled pairs must be set")

	subs, err := h.generateSubscriptionsForAsset(asset.CoinMarginedFutures, false)
	require.NoError(t, err, "generateSubscriptionsForAsset must not error")
	require.Len(t, subs, 5, "coin-margined subscriptions must include each public market channel")
	for _, sub := range subs {
		assert.Equal(t, asset.CoinMarginedFutures, sub.Asset, "subscription asset should match")
		assert.False(t, sub.Authenticated, "public futures subscriptions should not require authentication")
		assert.NotEmpty(t, sub.QualifiedChannel, "subscription channel should be expanded")
	}

	h.Features.Subscriptions = defaultFuturesSubscriptions.Clone()
	h.futureContractCodes = map[string]currency.Code{"CW": currency.NewCode("250829")}
	deliveryPairs := currency.Pairs{currency.NewPairWithDelimiter("BTC", "250829", "-")}
	require.NoError(t, h.SetPairs(deliveryPairs, asset.Futures, false), "available delivery pairs must be set")
	require.NoError(t, h.SetPairs(deliveryPairs, asset.Futures, true), "enabled delivery pairs must be set")
	subs, err = h.generateSubscriptionsForAsset(asset.Futures, false)
	require.NoError(t, err, "delivery generateSubscriptionsForAsset must not error")
	require.Len(t, subs, 4, "delivery subscriptions must include each public market channel")
	for _, sub := range subs {
		assert.Contains(t, sub.QualifiedChannel, "BTC_CW", "delivery channel should use HTX's expiry shorthand")
	}
}

func TestGetWebsocketConnection(t *testing.T) {
	t.Parallel()
	h := testexch.MockWsInstance[Exchange](t, mockws.CurryWsMockUpgrader(t, wsFixture))
	conn, err := h.getWebsocketConnection(&subscription.Subscription{Asset: asset.CoinMarginedFutures})
	require.NoError(t, err, "getWebsocketConnection must find the coin-margined connection")
	assert.NotNil(t, conn, "coin-margined connection should be returned")

	_, err = h.getWebsocketConnection(&subscription.Subscription{Asset: asset.Binary})
	require.ErrorIs(t, err, asset.ErrNotSupported, "getWebsocketConnection must reject unsupported assets")
}

func TestSubscribeConnection(t *testing.T) {
	t.Parallel()
	h := testexch.MockWsInstance[Exchange](t, mockws.CurryWsMockUpgrader(t, wsFixture))
	conn, err := h.Websocket.GetConnection(exchange.WebsocketSpot)
	require.NoError(t, err, "spot websocket connection must be available")
	sub := &subscription.Subscription{
		Asset:            asset.CoinMarginedFutures,
		Channel:          subscription.TickerChannel,
		Pairs:            currency.Pairs{btcusdPair},
		QualifiedChannel: "market.BTC-USD.detail",
	}
	err = h.subscribeConnection(t.Context(), conn, subscription.List{sub})
	require.NoError(t, err, "subscribeConnection must not error")
	assert.Equal(t, subscription.SubscribedState, sub.State(), "subscription should be active")
}

func TestUnsubscribeConnection(t *testing.T) {
	t.Parallel()
	h := testexch.MockWsInstance[Exchange](t, mockws.CurryWsMockUpgrader(t, wsFixture))
	conn, err := h.Websocket.GetConnection(exchange.WebsocketSpot)
	require.NoError(t, err, "spot websocket connection must be available")
	sub := &subscription.Subscription{
		Asset:            asset.CoinMarginedFutures,
		Channel:          subscription.TickerChannel,
		Pairs:            currency.Pairs{btcusdPair},
		QualifiedChannel: "market.BTC-USD.detail",
	}
	require.NoError(t, h.subscribeConnection(t.Context(), conn, subscription.List{sub}), "subscribeConnection must not error")
	require.NoError(t, h.unsubscribeConnection(t.Context(), conn, subscription.List{sub}), "unsubscribeConnection must not error")
	assert.Nil(t, conn.Subscriptions().Get(sub), "subscription should be removed")
}

func TestWSAuthenticateConnection(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	err := h.wsAuthenticateConnection(t.Context(), h.Websocket.AuthConn)
	require.ErrorIs(t, err, exchange.ErrAuthenticationSupportNotEnabled, "wsAuthenticateConnection must require credentials")
}

func TestWSHandleFundingRateMsg(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	sub := &subscription.Subscription{
		Asset: asset.CoinMarginedFutures,
		Pairs: currency.Pairs{btcusdPair},
	}
	err := h.wsHandleFundingRateMsg(t.Context(), sub, []byte(`{
		"ch":"public.BTC-USD.funding_rate",
		"ts":1604312615051,
		"tick":{
			"estimated_rate":"0.0002",
			"funding_rate":"0.0001",
			"contract_code":"BTC-USD",
			"funding_time":"1604312615051",
			"next_funding_time":"1604341415051"
		}
	}`))
	require.NoError(t, err, "wsHandleFundingRateMsg must not error")
	message := <-h.Websocket.DataHandler.C
	rate, ok := message.Data.(*WsFundingRate)
	require.True(t, ok, "websocket funding update must have the expected type")
	assert.Equal(t, asset.CoinMarginedFutures, rate.Asset, "asset should match")
	assert.Equal(t, btcusdPair, rate.Pair, "pair should match")
	assert.Equal(t, 0.0001, rate.Tick.FundingRate.Float64(), "funding rate should match")
	assert.Equal(t, time.UnixMilli(1604312615051), rate.Tick.FundingTime.Time(), "funding time should match")
}
