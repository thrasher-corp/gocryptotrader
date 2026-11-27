package gate

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func TestWebsocketDeliveryFuturesConnect(t *testing.T) {
	t.Parallel()

	t.Run("enabled asset", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		conn := &dialRecorderConnection{url: deliveryRealUSDTTradingURL}
		require.NoError(t, ex.WebsocketDeliveryFuturesConnect(t.Context(), conn, asset.DeliveryFutures), "WebsocketDeliveryFuturesConnect must dial an enabled asset")
		assert.True(t, conn.dialled, "WebsocketDeliveryFuturesConnect should dial")
		require.True(t, conn.pinged, "WebsocketDeliveryFuturesConnect must set up the ping handler")
		exp, err := getWSPingHandler(futuresPingChannel)
		require.NoError(t, err, "getWSPingHandler must not error")
		assert.Equal(t, exp, conn.pingHandler, "WebsocketDeliveryFuturesConnect should ping on the futures ping channel")
	})

	t.Run("disabled asset", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		require.NoError(t, ex.CurrencyPairs.SetAssetEnabled(asset.DeliveryFutures, false), "SetAssetEnabled must not error")
		conn := &dialRecorderConnection{url: deliveryRealUSDTTradingURL}
		assert.ErrorIs(t, ex.WebsocketDeliveryFuturesConnect(t.Context(), conn, asset.DeliveryFutures), asset.ErrNotEnabled, "WebsocketDeliveryFuturesConnect should reject a disabled asset")
		assert.False(t, conn.dialled, "WebsocketDeliveryFuturesConnect should not dial for a disabled asset")
	})

	t.Run("failed dial", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		errDial := errors.New("dial failed")
		conn := &dialRecorderConnection{url: deliveryRealUSDTTradingURL, dialErr: errDial}
		assert.ErrorIs(t, ex.WebsocketDeliveryFuturesConnect(t.Context(), conn, asset.DeliveryFutures), errDial, "WebsocketDeliveryFuturesConnect should return a dial error")
		assert.False(t, conn.pinged, "WebsocketDeliveryFuturesConnect should not set up the ping handler after a failed dial")
	})

	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mock build serves no Gate websocket")
		}
		ex, _ := newLiveWebsocketConnection(t, asset.DeliveryFutures)
		assert.True(t, ex.Websocket.IsConnected(), "WebsocketDeliveryFuturesConnect should connect")
	})
}

func TestGenerateDeliveryFuturesDefaultSubscriptions(t *testing.T) {
	t.Parallel()

	t.Run("unsupported asset", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		ex.SetDefaults()

		got, err := ex.GenerateDeliveryFuturesDefaultSubscriptions(t.Context(), asset.Futures)
		require.NoError(t, err, "GenerateDeliveryFuturesDefaultSubscriptions must not error for unsupported delivery asset")
		require.Empty(t, got, "unsupported delivery asset must not generate subscriptions")
	})

	t.Run("context credentials", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		ex.SetDefaults()
		ex.Websocket.SetCanUseAuthenticatedEndpoints(true)

		_, err := ex.GenerateDeliveryFuturesDefaultSubscriptions(t.Context(), asset.DeliveryFutures)
		require.NoError(t, err, "GenerateDeliveryFuturesDefaultSubscriptions must not error without credentials")
		require.False(t, ex.Websocket.CanUseAuthenticatedEndpoints(), "GenerateDeliveryFuturesDefaultSubscriptions must disable authenticated endpoints without credentials")

		ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
		ctx := accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: "test", Secret: "test"})
		_, err = ex.GenerateDeliveryFuturesDefaultSubscriptions(ctx, asset.DeliveryFutures)
		require.NoError(t, err, "GenerateDeliveryFuturesDefaultSubscriptions must not error with context credentials")
		assert.True(t, ex.Websocket.CanUseAuthenticatedEndpoints(), "GenerateDeliveryFuturesDefaultSubscriptions should keep authenticated endpoints when its context carries credentials")
	})

	_, err := e.GenerateDeliveryFuturesDefaultSubscriptions(t.Context(), asset.DeliveryFutures)
	assert.NoError(t, err)
}

func TestGenerateDeliveryFuturesPayload(t *testing.T) {
	t.Parallel()
	contract := currency.NewPairWithDelimiter("BTC", "USDT_20260626", currency.UnderscoreDelimiter)

	t.Run("no channels", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		ex.SetDefaults()
		_, err := ex.generateDeliveryFuturesPayload(t.Context(), subscribeEvent, nil)
		assert.ErrorIs(t, err, errNoChannelsSupplied, "generateDeliveryFuturesPayload should reject an empty list")
	})

	t.Run("several pairs", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		ex.SetDefaults()
		_, err := ex.generateDeliveryFuturesPayload(t.Context(), subscribeEvent, subscription.List{{Channel: futuresTickersChannel, Pairs: currency.Pairs{contract, contract}}})
		assert.ErrorIs(t, err, subscription.ErrNotSinglePair, "generateDeliveryFuturesPayload should reject a subscription to several contracts")
	})

	t.Run("unsupported frequency", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		ex.SetDefaults()
		_, err := ex.generateDeliveryFuturesPayload(t.Context(), subscribeEvent, subscription.List{{Channel: futuresOrderbookUpdateChannel, Pairs: currency.Pairs{contract}, Params: map[string]any{"frequency": kline.ThreeMin}}})
		assert.ErrorIs(t, err, kline.ErrUnsupportedInterval, "generateDeliveryFuturesPayload should reject an unsupported frequency")
	})

	t.Run("unsupported candle interval", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		ex.SetDefaults()
		_, err := ex.generateDeliveryFuturesPayload(t.Context(), subscribeEvent, subscription.List{{Channel: futuresCandlesticksChannel, Pairs: currency.Pairs{contract}, Params: map[string]any{"interval": kline.ThreeMin}}})
		assert.ErrorIs(t, err, kline.ErrUnsupportedInterval, "generateDeliveryFuturesPayload should reject an unsupported candle interval")
	})

	t.Run("public channels", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		ex.SetDefaults()
		ex.Websocket.SetCanUseAuthenticatedEndpoints(false)
		got, err := ex.generateDeliveryFuturesPayload(t.Context(), unsubscribeEvent, subscription.List{
			{Channel: futuresTickersChannel, Pairs: currency.Pairs{contract}},
			{Channel: futuresOrderbookUpdateChannel, Pairs: currency.Pairs{contract}, Params: map[string]any{"frequency": kline.HundredMilliseconds, "level": "100", "limit": 20, "accuracy": "0"}},
			{Channel: futuresCandlesticksChannel, Pairs: currency.Pairs{contract}, Params: map[string]any{"interval": kline.FiveMin}},
			{Channel: futuresOrderbookChannel, Pairs: currency.Pairs{contract}, Params: map[string]any{"limit": 20, "interval": "0"}},
			{Channel: futuresOrdersChannel, Pairs: currency.Pairs{contract}, Params: map[string]any{"user": "20011"}},
		})
		require.NoError(t, err, "generateDeliveryFuturesPayload must not error")
		exp := []struct {
			channel string
			payload []string
		}{
			{channel: futuresTickersChannel, payload: []string{"BTC_USDT_20260626"}},
			{channel: futuresOrderbookUpdateChannel, payload: []string{"BTC_USDT_20260626", "100ms", "100", "20", "0"}},
			{channel: futuresCandlesticksChannel, payload: []string{"5m", "BTC_USDT_20260626"}},
			{channel: futuresOrderbookChannel, payload: []string{"BTC_USDT_20260626", "20", "0"}},
			{channel: futuresOrdersChannel, payload: []string{"BTC_USDT_20260626"}},
		}
		require.Len(t, got, len(exp), "generateDeliveryFuturesPayload must return a request for each subscription")
		for i := range exp {
			assert.NotZerof(t, got[i].ID, "generateDeliveryFuturesPayload should number the %s request", exp[i].channel)
			assert.Equalf(t, unsubscribeEvent, got[i].Event, "generateDeliveryFuturesPayload should set the event of the %s request", exp[i].channel)
			assert.Equalf(t, exp[i].channel, got[i].Channel, "generateDeliveryFuturesPayload should name request %d's channel", i)
			assert.Equalf(t, exp[i].payload, got[i].Payload, "generateDeliveryFuturesPayload should build the %s payload", exp[i].channel)
			assert.NotZerof(t, got[i].Time, "generateDeliveryFuturesPayload should stamp the %s request", exp[i].channel)
			assert.Nilf(t, got[i].Auth, "generateDeliveryFuturesPayload should not sign the %s request without authenticated endpoints", exp[i].channel)
		}
	})

	t.Run("authenticated channel", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		ex.SetDefaults()
		ex.API.AuthenticatedWebsocketSupport = true
		ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
		ex.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
		got, err := ex.generateDeliveryFuturesPayload(t.Context(), subscribeEvent, subscription.List{
			{Channel: futuresOrdersChannel, Pairs: currency.Pairs{contract}, Params: map[string]any{"user": "20011"}},
			{Channel: futuresTickersChannel, Pairs: currency.Pairs{contract}},
		})
		require.NoError(t, err, "generateDeliveryFuturesPayload must not error")
		require.Len(t, got, 2, "generateDeliveryFuturesPayload must return a request for each subscription")
		assert.Equal(t, []string{"20011", "BTC_USDT_20260626"}, got[0].Payload, "generateDeliveryFuturesPayload should put the user ID before the contract")
		sig, err := ex.generateWebsocketSignature("secret", subscribeEvent, futuresOrdersChannel, got[0].Time)
		require.NoError(t, err, "generateWebsocketSignature must not error")
		assert.Equal(t, &WebsocketAuthInput{Method: "api_key", Key: "key", Sign: sig}, got[0].Auth, "generateDeliveryFuturesPayload should sign a user channel with the API key")
		assert.Equal(t, []string{"BTC_USDT_20260626"}, got[1].Payload, "generateDeliveryFuturesPayload should send a public channel's contract")
		assert.Nil(t, got[1].Auth, "generateDeliveryFuturesPayload should not sign a public channel")
	})

	t.Run("authenticated channel without credentials", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		ex.SetDefaults()
		ex.API.AuthenticatedWebsocketSupport = true
		ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
		got, err := ex.generateDeliveryFuturesPayload(t.Context(), subscribeEvent, subscription.List{{Channel: futuresOrdersChannel, Pairs: currency.Pairs{contract}, Params: map[string]any{"user": "20011"}}})
		require.NoError(t, err, "generateDeliveryFuturesPayload must not error")
		require.Len(t, got, 1, "generateDeliveryFuturesPayload must return a request for the subscription")
		assert.Equal(t, []string{"BTC_USDT_20260626"}, got[0].Payload, "generateDeliveryFuturesPayload should send the contract alone without credentials")
		assert.Nil(t, got[0].Auth, "generateDeliveryFuturesPayload should not sign without credentials")
		assert.False(t, ex.Websocket.CanUseAuthenticatedEndpoints(), "generateDeliveryFuturesPayload should disable authenticated endpoints without credentials")
	})
}

func TestDeliveryFuturesSubscribe(t *testing.T) {
	t.Parallel()
	t.Run("live channel subscribed again", func(t *testing.T) {
		t.Parallel()
		liveSubscriptionCase(t, asset.DeliveryFutures, (*Exchange).DeliveryFuturesUnsubscribe, (*Exchange).DeliveryFuturesSubscribe)
	})
}

func TestDeliveryFuturesUnsubscribe(t *testing.T) {
	t.Parallel()
	t.Run("live channel", func(t *testing.T) {
		t.Parallel()
		liveSubscriptionCase(t, asset.DeliveryFutures, (*Exchange).DeliveryFuturesUnsubscribe, nil)
	})
}
