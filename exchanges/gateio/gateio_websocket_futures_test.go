package gateio

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

func TestGenerateFuturesPayload(t *testing.T) {
	t.Parallel()

	t.Run("empty channels", func(t *testing.T) {
		t.Parallel()

		_, err := e.generateFuturesPayload(t.Context(), subscribeEvent, nil)
		require.Error(t, err)
		require.Contains(t, fmt.Sprint(err), "cannot generate payload, no channels supplied")
	})

	t.Run("not single pair", func(t *testing.T) {
		t.Parallel()

		_, err := e.generateFuturesPayload(t.Context(), subscribeEvent, subscription.List{
			&subscription.Subscription{Channel: futuresTickersChannel, Pairs: nil},
		})
		require.ErrorIs(t, err, subscription.ErrNotSinglePair)
	})

	t.Run("frequency invalid interval", func(t *testing.T) {
		t.Parallel()

		_, err := e.generateFuturesPayload(t.Context(), subscribeEvent, subscription.List{
			&subscription.Subscription{
				Channel: futuresOrderbookUpdateChannel,
				Pairs:   currency.Pairs{BTCUSDT},
				Params:  map[string]any{"frequency": kline.Interval(time.Duration(-1))},
			},
		})
		require.ErrorIs(t, err, kline.ErrUnsupportedInterval)
	})

	t.Run("candlestick interval invalid", func(t *testing.T) {
		t.Parallel()

		_, err := e.generateFuturesPayload(t.Context(), subscribeEvent, subscription.List{
			&subscription.Subscription{
				Channel: futuresCandlesticksChannel,
				Pairs:   currency.Pairs{BTCUSDT},
				Params:  map[string]any{"interval": kline.Interval(time.Duration(-1))},
			},
		})
		require.Error(t, err, kline.ErrUnsupportedInterval)
	})

	t.Run("orderbook update with snapshot missing level", func(t *testing.T) {
		t.Parallel()

		_, err := e.generateFuturesPayload(t.Context(), subscribeEvent, subscription.List{
			&subscription.Subscription{Channel: futuresOrderbookUpdateWithSnapshotChannel, Pairs: currency.Pairs{BTCUSDT}, Params: map[string]any{}},
		})
		require.ErrorIs(t, err, errParameterRequired)
	})

	t.Run("orderbook update with snapshot bad level type", func(t *testing.T) {
		t.Parallel()

		_, err := e.generateFuturesPayload(t.Context(), subscribeEvent, subscription.List{
			&subscription.Subscription{Channel: futuresOrderbookUpdateWithSnapshotChannel, Pairs: currency.Pairs{BTCUSDT}, Params: map[string]any{"level": 50}},
		})
		require.ErrorIs(t, err, common.ErrTypeAssertFailure)
	})

	t.Run("orderbook update with snapshot empty pair", func(t *testing.T) {
		t.Parallel()

		_, err := e.generateFuturesPayload(t.Context(), subscribeEvent, subscription.List{
			&subscription.Subscription{Channel: futuresOrderbookUpdateWithSnapshotChannel, Pairs: currency.Pairs{currency.EMPTYPAIR}, Params: map[string]any{"level": uint64(50)}},
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "params must be populated with only the currency pair")
	})

	t.Run("happy path unauthenticated - params", func(t *testing.T) {
		t.Parallel()

		ex := new(Exchange)
		ex.SetDefaults()
		ex.Name = "generateFuturesPayloadTest"
		ex.Websocket = e.Websocket
		ex.Websocket.SetCanUseAuthenticatedEndpoints(false)

		got, err := ex.generateFuturesPayload(context.Background(), subscribeEvent, subscription.List{
			&subscription.Subscription{
				Channel: futuresOrderbookUpdateChannel,
				Pairs:   currency.Pairs{BTCUSDT},
				Params: map[string]any{
					"frequency": kline.TwentyMilliseconds,
					"level":     "20",
					"limit":     100,
					"accuracy":  "0",
				},
			},
			&subscription.Subscription{
				Channel: futuresCandlesticksChannel,
				Pairs:   currency.Pairs{BTCUSDT},
				Params:  map[string]any{"interval": kline.FiveMin},
			},
			&subscription.Subscription{
				Channel: futuresOrderbookChannel,
				Pairs:   currency.Pairs{BTCUSDT},
				Params:  map[string]any{"interval": "0", "limit": 100},
			},
			&subscription.Subscription{
				Channel: futuresOrderbookUpdateWithSnapshotChannel,
				Pairs:   currency.Pairs{BTCUSDT},
				Params:  map[string]any{"level": uint64(50)},
			},
		})
		require.NoError(t, err, "generateFuturesPayload must not error")
		require.Len(t, got, 4)

		for i := range got {
			require.NotZero(t, got[i].ID)
			require.Equal(t, subscribeEvent, got[i].Event)
			require.NotEmpty(t, got[i].Channel)
			require.NotZero(t, got[i].Time)
			require.Nil(t, got[i].Auth, "Auth must be nil when unauthenticated")
			require.NotEmpty(t, got[i].Payload, "Payload must not be empty")
		}

		require.Equal(t, []string{BTCUSDT.String(), "20ms", "20", "100", "0"}, got[0].Payload)
		require.Equal(t, []string{"5m", BTCUSDT.String()}, got[1].Payload)
		require.Equal(t, []string{BTCUSDT.String(), "100", "0"}, got[2].Payload)
		require.Equal(t, []string{"ob." + BTCUSDT.String() + ".50"}, got[3].Payload)
	})

	t.Run("authenticated channel - missing creds disables auth", func(t *testing.T) {
		t.Parallel()

		ex := new(Exchange)
		ex.SetDefaults()
		ex.Name = "generateFuturesPayloadAuthDisableTest"

		// Force path into GetCredentials() by allowing authenticated endpoints.
		ex.API.AuthenticatedWebsocketSupport = true
		ex.Websocket = e.Websocket
		ex.Websocket.SetCanUseAuthenticatedEndpoints(true)

		got, err := ex.generateFuturesPayload(t.Context(), subscribeEvent, subscription.List{
			&subscription.Subscription{
				Channel: futuresBalancesChannel,
				Pairs:   currency.Pairs{BTCUSDT},
			},
		})
		require.NoError(t, err, "generateFuturesPayload must not error")
		require.Len(t, got, 1)
		require.Nil(t, got[0].Auth, "Auth should be nil when GetCredentials fails")
		require.False(t, ex.Websocket.CanUseAuthenticatedEndpoints(), "authenticated endpoints must be disabled on GetCredentials error")
	})

	t.Run("authenticated channel - user param inserted + signature", func(t *testing.T) {
		t.Parallel()

		ex := new(Exchange)
		ex.SetDefaults()
		ex.Name = "generateFuturesPayloadAuthTest"
		ex.API.AuthenticatedWebsocketSupport = true
		ex.Websocket = e.Websocket
		ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
		ex.SetCredentials("key", "secret", "", "", "", "")

		got, err := ex.generateFuturesPayload(t.Context(), subscribeEvent, subscription.List{
			&subscription.Subscription{
				Channel: futuresBalancesChannel,
				Pairs:   currency.Pairs{BTCUSDT},
				Params:  map[string]any{"user": "user123"},
			},
		})
		require.NoError(t, err, "generateFuturesPayload must not error")
		require.Len(t, got, 1)

		require.NotNil(t, got[0].Auth, "Auth must not be nil when authenticated")
		require.Equal(t, "api_key", got[0].Auth.Method)
		require.Equal(t, "key", got[0].Auth.Key)
		require.NotEmpty(t, got[0].Auth.Sign)

		require.Equal(t, []string{"user123", BTCUSDT.String()}, got[0].Payload)

		sig, err := ex.generateWsSignature("secret", subscribeEvent, futuresBalancesChannel, got[0].Time)
		require.NoError(t, err)
		require.Equal(t, sig, got[0].Auth.Sign)
	})
}
