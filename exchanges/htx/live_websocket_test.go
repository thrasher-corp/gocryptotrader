package htx

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	mockws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
)

// connectHTXLiveWebsocket keeps the connection context alive through test cleanup.
func connectHTXLiveWebsocket(t *testing.T, h *Exchange, subs subscription.List) {
	t.Helper()
	h.Features.Subscriptions = subs
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	t.Cleanup(func() {
		defer cancel()
		if err := h.Websocket.Disable(); err != nil {
			assert.ErrorIs(t, err, websocket.ErrAlreadyDisabled, "cleanup should disable reconnects")
		}
		if err := h.Websocket.Shutdown(); err != nil {
			assert.ErrorIs(t, err, websocket.ErrNotConnected, "cleanup should close live sockets")
		}
	})
	require.NoError(t, h.Websocket.Connect(ctx), "HTX websocket must connect")
}

func TestConnectHTXLiveWebsocket(t *testing.T) {
	t.Parallel()
	h := newHTXLiveExchange(t, &accounts.Credentials{Key: "test-key", Secret: "test-secret"}, htxLiveOptions{websocket: true})
	server := httptest.NewTestServer(t, mockws.CurryWsMockUpgrader(t, wsFixture))
	server.Start()
	require.NoError(t, h.Websocket.SetAllConnectionURLs("ws"+strings.TrimPrefix(server.URL, "http")), "websocket must use the local server")
	connectHTXLiveWebsocket(t, h, nil)
	_, err := h.Websocket.GetConnection(exchange.WebsocketTrade)
	require.NoError(t, err, "authenticated trade socket must connect")
}

// TestHTXLivePrivateWebsocket exercises the manager, login and subscriptions together.
func TestHTXLivePrivateWebsocket(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		asset    asset.Item
		endpoint exchange.URL
	}{
		{asset.Spot, exchange.WebsocketPrivate},
		{asset.Futures, exchange.WebsocketFuturesPrivate},
		{asset.CoinMarginedFutures, exchange.WebsocketCoinMarginedPrivate},
		{asset.USDTMarginedFutures, exchange.WebsocketUSDTMarginedPrivate},
	} {
		t.Run(tc.asset.String(), func(t *testing.T) {
			t.Parallel()
			if mockTests {
				t.Skip("Live testing requires the mock_test_off build tag")
			}
			h := newHTXLiveExchange(t, apiCredentials, htxLiveOptions{websocket: true})
			if tc.asset != asset.USDTMarginedFutures {
				require.NoError(t, h.CurrencyPairs.SetAssetEnabled(asset.USDTMarginedFutures, false), "test must isolate the requested private socket")
			}
			channels := []string{subscription.MyAccountChannel, subscription.MyOrdersChannel, subscription.MyTradesChannel}
			if tc.asset == asset.USDTMarginedFutures {
				channels = append(channels, wsTradeUpdatesChannel, wsExecutionDetailsChannel)
			}
			subs := make(subscription.List, 0, len(channels))
			for _, channel := range channels {
				subs = append(subs, &subscription.Subscription{Enabled: true, Authenticated: true, Asset: tc.asset, Channel: channel})
			}
			connectHTXLiveWebsocket(t, h, subs)
			conn, err := h.Websocket.GetConnection(tc.endpoint)
			require.NoError(t, err, "the requested private socket must authenticate")
			registered := conn.Subscriptions().List()
			require.Len(t, registered, len(channels), "the connection must own every acknowledged subscription")
			for _, sub := range registered {
				assert.Equal(t, tc.asset, sub.Asset, "subscription should belong to the requested asset")
				assert.Equal(t, subscription.SubscribedState, sub.State(), "subscription should be acknowledged")
			}
			t.Run("external account activity notifications", func(t *testing.T) {
				t.Parallel()
				deadline := time.NewTimer(10 * time.Second)
				defer deadline.Stop()
				observed := map[string]bool{"balance": false, "order": false, "execution": false}
			observe:
				for !observed["balance"] || !observed["order"] || !observed["execution"] {
					select {
					case <-deadline.C:
						break observe
					case <-t.Context().Done():
						require.FailNow(t, "notification observation context must remain active")
					case message, ok := <-h.Websocket.DataHandler.C:
						require.True(t, ok, "private stream must remain open")
						switch data := message.Data.(type) {
						case error:
							require.NoError(t, data, "private message handling must succeed")
						case *order.Detail:
							assert.Equal(t, tc.asset, data.AssetType, "order update should belong to the connected asset")
							assert.NotEmpty(t, data.OrderID, "order update should carry an exchange identifier")
							observed["order"] = true
							for _, trade := range data.Trades {
								observed["execution"] = observed["execution"] || (trade.Amount > 0 && trade.TID != "" && trade.TID != "0")
							}
						case []fill.Data:
							for _, execution := range data {
								assert.Equal(t, tc.asset, execution.AssetType, "execution should belong to the connected asset")
								assert.NotEmpty(t, execution.OrderID, "execution should carry its exchange order identifier")
								assert.NotEmpty(t, execution.TradeID, "execution should carry its exchange trade identifier")
								assert.Positive(t, execution.Amount, "execution should carry its traded amount")
								assert.Positive(t, execution.Price, "execution should carry its traded price")
							}
							observed["execution"] = observed["execution"] || len(data) != 0
						case []accounts.Change:
							for _, change := range data {
								assert.Equal(t, tc.asset, change.AssetType, "balance update should belong to the connected asset")
							}
							observed["balance"] = observed["balance"] || len(data) != 0
						case WebsocketAccountUpdate:
							assert.False(t, data.Currency.IsEmpty(), "balance update should identify its currency")
							observed["balance"] = true
						}
					}
				}
				for _, area := range []string{"balance", "order", "execution"} {
					t.Run(area, func(t *testing.T) {
						t.Parallel()
						if !observed[area] {
							t.Skip("The account had no matching external activity during the observation window")
						}
					})
				}
			})
		})
	}
}
