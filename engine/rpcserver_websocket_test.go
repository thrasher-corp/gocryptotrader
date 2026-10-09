package engine

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/gctrpc"
	mockws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
)

func TestRPCPairChangesActivateIdleWebsocket(t *testing.T) {
	t.Parallel()
	for _, operation := range []string{"SetExchangePair", "SetExchangeAsset", "SetAllExchangePairs"} {
		for _, legacy := range []bool{false, true} {
			name := operation + "/idle"
			if legacy {
				name = operation + "/offline legacy"
			}
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				em := NewExchangeManager()
				exchangeName := "GateIO"
				if legacy {
					exchangeName = "Binance"
				}
				exch, err := em.NewExchangeByName(exchangeName)
				require.NoError(t, err, "exchange must be available")
				exch.SetDefaults()
				exch.SetEnabled(true)
				base := exch.GetBase()
				pair := currency.NewBTCUSDT()
				var enabled currency.Pairs
				assetEnabled := true
				if operation == "SetExchangeAsset" {
					enabled = currency.Pairs{pair}
					assetEnabled = false
				}
				stores := make([]*currency.PairsManager, 2)
				for i := range stores {
					stores[i] = &currency.PairsManager{Pairs: currency.FullStore{asset.Spot: {
						Available: currency.Pairs{pair}, Enabled: enabled, AssetEnabled: assetEnabled,
						ConfigFormat: &currency.PairFormat{Uppercase: true}, RequestFormat: &currency.PairFormat{Uppercase: true},
					}}}
				}
				base.CurrencyPairs.Pairs = stores[0].Pairs
				cfg := config.Exchange{
					Name: exchangeName, Enabled: true, CurrencyPairs: stores[1],
					Features:                &config.FeaturesConfig{Enabled: config.FeaturesEnabledConfig{Websocket: true}},
					WebsocketTrafficTimeout: time.Minute, ConnectionMonitorDelay: 10 * time.Millisecond,
				}
				base.Config = &cfg
				m := websocket.NewManager()
				base.Websocket = m
				srv, dialer := mockws.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					mockws.WsMockUpgrader(t, w, r, mockws.EchoHandler)
				}))
				url := "ws" + srv.URL[len("http"):]
				var legacyConnects int
				require.NoError(t, m.Setup(&websocket.ManagerSetup{
					ExchangeConfig: &cfg, UseMultiConnectionManagement: !legacy,
					Features:   &protocol.Features{Subscribe: true, Unsubscribe: true},
					DefaultURL: url, RunningURL: url,
					Connector:  func() error { legacyConnects++; return errExpectedTestError },
					Subscriber: func(subscription.List) error { return nil }, Unsubscriber: func(subscription.List) error { return nil },
					GenerateSubscriptions: func() (subscription.List, error) { return nil, nil },
				}), "manager setup must succeed")
				t.Cleanup(func() {
					require.NoError(t, m.Disable(), "manager must disable during cleanup")
					if m.IsConnected() {
						require.NoError(t, m.Shutdown(), "connected manager must shut down")
					} else {
						close(m.ShutdownC)
					}
					done := make(chan struct{})
					go func() { m.Wg.Wait(); close(done) }()
					select {
					case <-done:
					case <-time.After(time.Second):
						require.FailNow(t, "websocket readers and monitors must stop during cleanup")
					}
				})
				if !legacy {
					require.NoError(t, m.SetupNewConnection(&websocket.ConnectionSetup{
						URL:       url,
						Connector: func(ctx context.Context, conn websocket.Connection) error { return conn.Dial(ctx, dialer, nil, nil) },
						GenerateSubscriptions: func() (subscription.List, error) {
							enabledErr := base.CurrencyPairs.IsAssetEnabled(asset.Spot)
							if errors.Is(enabledErr, asset.ErrNotEnabled) {
								return nil, nil
							}
							if enabledErr != nil {
								return nil, enabledErr
							}
							pairs, err := base.CurrencyPairs.GetPairs(asset.Spot, true)
							if err != nil {
								return nil, err
							}
							if len(pairs) == 0 {
								return nil, nil
							}
							return subscription.List{{Channel: "ticker", Asset: asset.Spot, Pairs: pairs}}, nil
						},
						Subscriber: func(_ context.Context, conn websocket.Connection, subs subscription.List) error {
							return m.AddSuccessfulSubscriptions(conn, subs...)
						},
						Unsubscriber: func(_ context.Context, conn websocket.Connection, subs subscription.List) error {
							return m.RemoveSubscriptions(conn, subs...)
						},
						Handler: func(context.Context, websocket.Connection, []byte) error { return nil },
					}), "multi-connection setup must succeed")
				}
				err = m.Connect(t.Context())
				if legacy {
					require.ErrorIs(t, err, errExpectedTestError, "legacy startup must reproduce an offline venue")
					require.False(t, m.IsIdle(), "failed legacy connection must not be mistaken for idle")
				} else {
					require.NoError(t, err, "empty startup connect must succeed")
					require.True(t, m.IsIdle(), "empty multi-connection startup must be idle")
				}
				require.NoError(t, em.Add(exch), "exchange must register with the engine")
				s := RPCServer{Engine: &Engine{ExchangeManager: em, Config: &config.Config{Exchanges: []config.Exchange{cfg}}}}
				var response *gctrpc.GenericResponse
				switch operation {
				case "SetExchangePair":
					response, err = s.SetExchangePair(t.Context(), &gctrpc.SetExchangePairRequest{Exchange: exchangeName, AssetType: asset.Spot.String(), Enable: true, Pairs: []*gctrpc.CurrencyPair{{Base: "BTC", Quote: "USDT"}}})
				case "SetExchangeAsset":
					response, err = s.SetExchangeAsset(t.Context(), &gctrpc.SetExchangeAssetRequest{Exchange: exchangeName, Asset: asset.Spot.String(), Enable: true})
				case "SetAllExchangePairs":
					response, err = s.SetAllExchangePairs(t.Context(), &gctrpc.SetExchangeAllPairsRequest{Exchange: exchangeName, Enable: true})
				}
				require.NoError(t, err, "pair or asset RPC must succeed")
				require.NotNil(t, response, "RPC must return a response")
				assert.Equal(t, MsgStatusSuccess, response.Status, "RPC should report successful configuration change")
				pairs, err := base.CurrencyPairs.GetPairs(asset.Spot, true)
				require.NoError(t, err, "enabled pairs must be readable")
				assert.Equal(t, currency.Pairs{pair}, pairs, "RPC should enable the configured pair")
				if legacy {
					assert.Equal(t, 1, legacyConnects, "offline legacy RPC should not retry the failed startup connection")
					assert.False(t, m.IsConnected(), "offline legacy websocket should remain disconnected")
				} else {
					assert.True(t, m.IsConnected(), "RPC should activate idle websocket")
					assert.False(t, m.IsIdle(), "activated manager should clear idle state")
					assert.Len(t, m.GetSubscriptions(), 1, "RPC should register the newly enabled subscription")
				}
			})
		}
	}
}
