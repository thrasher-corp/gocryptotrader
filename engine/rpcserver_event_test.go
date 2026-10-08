package engine

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/gctrpc"
)

func TestAddEvent(t *testing.T) {
	t.Parallel()
	for _, item := range []string{ItemPrice, ItemOrderbook} {
		for _, tc := range []struct {
			name         string
			assetEnabled bool
			pairEnabled  bool
			wantErr      error
		}{
			{name: "enabled scope", assetEnabled: true, pairEnabled: true},
			{name: "available-only pair", assetEnabled: true, wantErr: errCurrencyNotEnabled},
			{name: "disabled asset", pairEnabled: true, wantErr: asset.ErrNotEnabled},
		} {
			t.Run(item+" "+tc.name, func(t *testing.T) {
				t.Parallel()
				em := NewExchangeManager()
				ex, err := em.NewExchangeByName("Binance")
				require.NoError(t, err, "exchange must be created")
				ex.SetDefaults()
				ex.SetEnabled(true)
				pair := currency.NewBTCUSDT()
				store := &currency.PairStore{Available: currency.Pairs{pair}, AssetEnabled: tc.assetEnabled, ConfigFormat: &currency.PairFormat{Uppercase: true}, RequestFormat: &currency.PairFormat{Uppercase: true}}
				if tc.pairEnabled {
					store.Enabled = currency.Pairs{pair}
				}
				require.NoError(t, ex.GetBase().CurrencyPairs.Store(asset.Spot, store), "pair scope must be configured")
				require.NoError(t, em.Add(ex), "exchange must be registered")
				events := &eventManager{exchangeManager: em}
				events.started.Store(true)
				s := RPCServer{Engine: &Engine{ExchangeManager: em, eventManager: events}}
				response, err := s.AddEvent(t.Context(), &gctrpc.AddEventRequest{
					Exchange: "Binance", AssetType: "spot", Item: item, Action: ActionTest,
					Pair:            &gctrpc.CurrencyPair{Base: "BTC", Quote: "USDT", Delimiter: "-"},
					ConditionParams: &gctrpc.ConditionParams{Condition: ConditionGreaterThan, Price: 1, OrderbookAmount: 1},
				})
				require.ErrorIs(t, err, tc.wantErr, "event creation must enforce the expected market data scope")
				if tc.wantErr == nil {
					require.NotNil(t, response, "enabled event must return its ID")
					require.Len(t, events.events, 1, "enabled event must be registered")
					assert.True(t, pair.Equal(events.events[0].Pair), "event should retain its enabled pair")
				} else {
					assert.Empty(t, events.events, "unsynchronised scopes should not register events")
				}
			})
		}
	}
}
