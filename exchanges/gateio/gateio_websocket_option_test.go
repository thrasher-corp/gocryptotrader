package gateio

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func TestProcessOptionsContractTickers(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		payload string
		wantErr bool
	}{
		{
			name:    "mocked mark and index prices",
			payload: `{"name":"BTC_USDT-20261225-100000-C","last_price":"118.4","mark_price":"118.35","index_price":"100000.25"}`,
		},
		{
			name:    "mocked malformed response",
			payload: `{`,
			wantErr: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex), "Setup must not error")
			err := ex.processOptionsContractTickers(t.Context(), []byte(tc.payload))
			if tc.wantErr {
				require.Error(t, err, "malformed response must fail")
				return
			}
			require.NoError(t, err, "options ticker processing must succeed")
			select {
			case msg := <-ex.Websocket.DataHandler.C:
				got, ok := msg.Data.(*ticker.Price)
				require.True(t, ok, "message must contain a ticker price")
				assert.Equal(t, 118.35, got.MarkPrice, "mark price should match")
				assert.Equal(t, 100000.25, got.IndexPrice, "index price should match")
				assert.Equal(t, asset.Options, got.AssetType, "asset should be options")
				assert.Equal(t, currency.NewPairWithDelimiter("BTC", "USDT-20261225-100000-C", currency.UnderscoreDelimiter), got.Pair, "contract should match")
			default:
				require.Fail(t, "options ticker must be emitted")
			}
		})
	}
}
