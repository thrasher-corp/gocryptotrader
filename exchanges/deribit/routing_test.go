package deribit

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	t.Run("routing lifecycle", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "exchange setup must succeed")
		var responseMtx sync.RWMutex
		body := `{"result":[]}`
		expectedKind := "spot"
		server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/v2/public/get_instruments", r.URL.Path, "request should discover instruments")
			responseMtx.RLock()
			defer responseMtx.RUnlock()
			assert.Equal(t, expectedKind, r.URL.Query().Get("kind"), "discovery should request the selected asset kind")
			_, err := fmt.Fprint(w, body)
			assert.NoError(t, err, "response should write")
		}))
		require.NoError(t, ex.SetHTTPClient(server.Client()), "client must configure")
		require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestFutures.String(), server.URL), "endpoint must configure")
		pair := currency.NewPairWithDelimiter("sol", "eth", "-")
		trades := &subscription.Subscription{Channel: subscription.AllTradesChannel}
		assert.True(t, ex.supportsChannel(trades, asset.Spot, pair), "unknown instruments should retain public trades")
		for _, tc := range []struct {
			name, body string
			asset      asset.Item
			routed     bool
			count      int
			invalid    bool
		}{
			{"native USDC", `{"result":[{"instrument_name":"SOL_ETH","is_active":true},{"instrument_name":"ETH_USDC","is_active":true}]}`, asset.Spot, false, 2, false},
			{"routed non-USDC", `{"result":[{"instrument_name":"SOL_ETH","is_active":true,"is_cbe_routed":true}]}`, asset.Spot, true, 1, false},
			{"derivative discovery preserves spot", `{"result":[{"instrument_name":"BTC-PERPETUAL","is_active":true}]}`, asset.Futures, true, 1, false},
			{"failed refresh preserves spot", `{"error":{"code":10001,"message":"failed"}}`, asset.Spot, true, 0, true},
			{"null result preserves spot", `{"result":null}`, asset.Spot, true, 0, true},
			{"missing result preserves spot", `{}`, asset.Spot, true, 0, true},
			{"malformed response preserves spot", `{`, asset.Spot, true, 0, true},
			{"partial parse preserves spot", `{"result":[{"instrument_name":"SOL_ETH","is_active":true},{"instrument_name":"x","is_active":true}]}`, asset.Spot, true, 0, true},
			{"null row preserves spot", `{"result":[null]}`, asset.Spot, true, 0, true},
			{"refresh to native", `{"result":[{"instrument_name":"SOL_ETH","is_active":true}]}`, asset.Spot, false, 1, false},
			{"routing alias", `{"result":[{"instrument_name":"SOL_ETH","is_active":true,"is_csr":true}]}`, asset.Spot, true, 1, false},
			{"inactive prunes", `{"result":[{"instrument_name":"SOL_ETH","is_active":false,"is_cbe_routed":true}]}`, asset.Spot, false, 0, false},
			{"routed again", `{"result":[{"instrument_name":"SOL_ETH","is_active":true,"is_cbe_routed":true}]}`, asset.Spot, true, 1, false},
			{"empty refresh prunes", `{"result":[]}`, asset.Spot, false, 0, false},
		} {
			t.Run(tc.name, func(t *testing.T) { // Sequential subtests exercise one instance's successive discoveries.
				responseMtx.Lock()
				body = tc.body
				expectedKind = "spot"
				if tc.asset == asset.Futures {
					expectedKind = "future"
				}
				responseMtx.Unlock()
				pairs, err := ex.FetchTradablePairs(t.Context(), tc.asset)
				if tc.invalid {
					require.Error(t, err, "invalid discovery must fail")
					assert.Nil(t, pairs, "failed discovery should not return partial pairs")
					if tc.name == "null row preserves spot" || tc.name == "null result preserves spot" || tc.name == "missing result preserves spot" || tc.name == "failed refresh preserves spot" {
						assert.ErrorIs(t, err, common.ErrInvalidResponse, "null row should identify invalid response")
					}
				} else {
					require.NoError(t, err, "discovery must succeed")
					assert.Len(t, pairs, tc.count, "discovery should contain active instruments")
				}
				assert.Equal(t, !tc.routed, ex.supportsChannel(trades, asset.Spot, pair), "routing should follow the last successful spot discovery independent of pair format")
				assert.True(t, ex.supportsChannel(trades, asset.Spot, currency.NewPair(currency.ETH, currency.USDC)), "native or unknown USDC should retain public trades")
				for _, a := range []asset.Item{asset.Futures, asset.Options, asset.FutureCombo, asset.OptionCombo} {
					assert.True(t, ex.supportsChannel(trades, a, pair), "derivative trades should remain supported")
				}
			})
		}
		t.Run("unsupported asset", func(t *testing.T) {
			_, err := ex.FetchTradablePairs(t.Context(), asset.Margin)
			assert.ErrorIs(t, err, asset.ErrNotSupported, "unsupported asset should fail before discovery")
		})
		t.Run("concurrent refresh and subscription reads", func(t *testing.T) {
			responseMtx.Lock()
			body = `{"result":[{"instrument_name":"SOL_ETH","is_active":true,"is_cbe_routed":true}]}`
			responseMtx.Unlock()
			var wg sync.WaitGroup
			stop := make(chan struct{})
			t.Cleanup(func() { close(stop); wg.Wait() })
			wg.Go(func() {
				for {
					select {
					case <-stop:
						return
					default:
						ex.supportsChannel(trades, asset.Spot, pair)
					}
				}
			})
			for range 5 {
				_, err := ex.FetchTradablePairs(t.Context(), asset.Spot)
				require.NoError(t, err, "concurrent refresh must succeed")
			}
			assert.False(t, ex.supportsChannel(trades, asset.Spot, pair), "final complete routing snapshot should remain visible")
		})
	})
}
