package simulator

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/bitstamp"
	"github.com/thrasher-corp/gocryptotrader/internal/testing/livetest"
)

func TestSimulate(t *testing.T) {
	// Not parallel: UpdateOrderbook stores the book in the orderbook package's shared store
	b := bitstamp.Exchange{}
	b.SetDefaults()
	b.Verbose = false
	b.CurrencyPairs = currency.PairsManager{
		UseGlobalFormat: true,
		RequestFormat: &currency.PairFormat{
			Uppercase: true,
		},
		Pairs: map[asset.Item]*currency.PairStore{
			asset.Spot: {
				AssetEnabled: true,
			},
		},
	}
	if livetest.Mock {
		book, err := os.ReadFile("testdata/order_book_btcusd.json")
		require.NoError(t, err, "ReadFile must not error")
		var server *httptest.Server
		server = httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet || r.TLS != nil || "http://"+r.Host != server.URL || r.URL.RequestURI() != "/api/v2/order_book/btcusd/?group=1" {
				http.NotFound(w, r)
				return
			}
			_, _ = w.Write(book)
		}))
		require.NoError(t, b.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
		require.NoError(t, b.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL+"/api"), "SetRunningURL must not error")
	}
	o, err := b.UpdateOrderbook(t.Context(),
		currency.NewBTCUSD(), asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	buy, err := o.SimulateOrder(10000000, true)
	if err != nil {
		t.Fatal(err)
	}
	sale, err := o.SimulateOrder(2171, false)
	if err != nil {
		t.Fatal(err)
	}
	if livetest.Mock {
		require.NotNil(t, buy, "SimulateOrder must return the buy's result")
		require.NotNil(t, sale, "SimulateOrder must return the sale's result")
		assert.InDelta(t, 1.90674027, buy.Amount, 1e-10, "the buy should take every recorded ask's BTC")
		assert.Contains(t, buy.Status, "Buying using 160648.291867 USD worth of BTC will send the price", "the buy should report spending the recorded asks' USD value")
		assert.InDelta(t, 114003.3011647639, sale.Amount, 1e-8, "the sale should return the recorded bids' USD value")
		assert.Contains(t, sale.Status, "Selling using 1.353175 BTC worth of USD will send the price", "the sale should report selling the recorded bids' BTC")
	}
}
