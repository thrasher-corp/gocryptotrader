package htx

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

// TestBookLevel covers a tick whose book side is short or absent. HTX serves null for the side
// when nothing rests there, so neither element can be indexed unconditionally
func TestBookLevel(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name                string
		level               []float64
		wantPrice, wantSize float64
	}{
		{name: "price and size", level: []float64{78553, 12}, wantPrice: 78553, wantSize: 12},
		{name: "price only", level: []float64{78553}, wantPrice: 78553},
		{name: "empty side", level: []float64{}},
		{name: "absent side"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			price, size := bookLevel(tc.level)
			assert.Equal(t, tc.wantPrice, price, "bookLevel should return the resting price")
			assert.Equal(t, tc.wantSize, size, "bookLevel should return the resting size")
		})
	}
}

// TestUpdateTickerVolumeUnits runs UpdateTicker end to end for spot and for a coin margined swap.
// vol is the quote currency on the spot endpoint but counts contracts on the swap one, so only spot
// has a quote volume to record, and the spot merged response carries the bid and ask the summary
// endpoint omits. A book side that is null or carries only its price is recorded rather than refused
func TestUpdateTickerVolumeUnits(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name                       string
		a                          asset.Item
		pair                       currency.Pair
		endpoint                   exchange.URL
		body                       string
		wantBase, wantQuote        float64
		wantBid, wantBidSize       float64
		wantLastUpdatedFromPayload bool
	}{
		{
			// trimmed from GET /market/detail/merged?symbol=btcusdt
			name: "spot records the quote turnover and the book", a: asset.Spot,
			pair: currency.NewBTCUSDT(), endpoint: exchange.RestSpot,
			body:     `{"status":"ok","ts":1788926189000,"tick":{"amount":6454.5661,"vol":506778563.47,"high":79000,"low":78000,"open":78500,"close":78616.33,"bid":[78615,1.5],"ask":[78617,2.5]}}`,
			wantBase: 6454.5661, wantQuote: 506778563.47, wantBid: 78615, wantBidSize: 1.5,
			wantLastUpdatedFromPayload: true,
		},
		{
			// trimmed from GET /swap-ex/market/detail/merged?contract_code=BTC-USD, where vol is
			// 65744 contracts of 100 USD against amount 83.81 BTC
			name: "coin margined records no quote volume", a: asset.CoinMarginedFutures,
			pair: currency.NewBTCUSD(), endpoint: exchange.RestFutures,
			body:     `{"status":"ok","ts":1788926189000,"tick":{"amount":"83.8102","vol":"65744","high":"79434.2","low":"77567.2","open":"79212.4","close":"78741.1","bid":[78740,3600],"ask":[78742,1200]}}`,
			wantBase: 83.8102, wantQuote: 0, wantBid: 78740, wantBidSize: 3600,
		},
		{
			// trimmed from GET /swap-ex/market/detail/merged?contract_code=OAS-USD, one of the
			// coin margined swaps with nothing resting on either side, which the batch endpoint
			// records with a zero book
			name: "coin margined records an empty book", a: asset.CoinMarginedFutures,
			pair: currency.NewPair(currency.NewCode("OAS"), currency.USD), endpoint: exchange.RestFutures,
			body: `{"ch":"market.OAS-USD.detail.merged","status":"ok","tick":{"amount":"0","ask":null,"bid":null,"close":"0.02","count":0,"high":"0.02","id":1789024695,"low":"0.02","open":"0.02","ts":1789024695555,"vol":"0"},"ts":1789024695555}`,
		},
		{
			// A side carrying only its price is recorded with a zero size, as on the futures paths
			name: "spot records a side carrying only its price", a: asset.Spot,
			pair: currency.NewPair(currency.ETH, currency.USDT), endpoint: exchange.RestSpot,
			body:     `{"status":"ok","ts":1788926189000,"tick":{"amount":1.5,"vol":3712.5,"high":2500,"low":2400,"open":2450,"close":2475,"bid":[2474.5],"ask":[2475.5,2]}}`,
			wantBase: 1.5, wantQuote: 3712.5, wantBid: 2474.5,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, err := fmt.Fprint(w, tc.body)
				assert.NoError(t, err, "writing the ticker response should not error")
			}))
			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex), "Setup must not error")
			ex.Name = t.Name()
			require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
			require.NoError(t, ex.API.Endpoints.SetRunningURL(tc.endpoint.String(), server.URL), "SetRunningURL must not error")

			got, err := ex.UpdateTicker(t.Context(), tc.pair, tc.a)
			require.NoError(t, err, "UpdateTicker must not error")
			assert.Equal(t, tc.wantBase, got.BaseVolume, "amount should be recorded as the base volume")
			assert.Equal(t, tc.wantQuote, got.QuoteVolume, "only an endpoint serving the quote currency should set a quote volume")
			assert.Equal(t, got.Close, got.Last, "Last should carry the latest price, as Close does")
			assert.Equal(t, tc.wantBid, got.Bid, "the resting bid should reach the store")
			assert.Equal(t, tc.wantBidSize, got.BidSize, "the resting bid size should reach the store")
			if tc.wantLastUpdatedFromPayload {
				assert.Equal(t, int64(1788926189000), got.LastUpdated.UnixMilli(), "the response timestamp should reach the store")
			}
		})
	}
}

func TestFMarketOverviewDataUnmarshal(t *testing.T) {
	t.Parallel()
	var o FMarketOverviewData
	err := json.Unmarshal([]byte(`{"ch":"market.BTC_CQ.detail.merged","tick":{"amount":"7.63","ask":[70020.48,2],"bid":[68941.77,3],"close":"69892.45","count":1768,"high":"70288.9","id":1787211582,"low":"67657.1","open":"68860.25","ts":1787211582961,"vol":"5188"},"ts":1787211582961}`), &o)
	require.NoError(t, err, "Unmarshal must not error")
	assert.Equal(t, []float64{70020.48, 2}, o.Tick.Ask, "Ask should decode")
	assert.Equal(t, []float64{68941.77, 3}, o.Tick.Bid, "Bid should decode")
	assert.Equal(t, int64(1787211582), o.Tick.ID, "ID should decode")
	assert.Equal(t, 69892.45, o.Tick.Close.Float64(), "Close should decode")
}

func TestWSTickerDerivativeRecordsNoQuoteVolume(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Setup Instance must not error")
	sub := &subscription.Subscription{Key: "market.BTC-USD.detail", Asset: asset.CoinMarginedFutures, Pairs: currency.Pairs{currency.NewBTCUSD()}, Channel: subscription.TickerChannel}
	require.NoError(t, e.Websocket.AddSubscriptions(e.Websocket.Conn, sub), "AddSubscriptions must not error")

	// Trimmed from the coin margined swap detail channel, where vol is a count of contracts
	payload := []byte(`{"ch":"market.BTC-USD.detail","ts":1630998026649,"tick":{"low":51000,"high":52924.14,"open":51823.62,"close":52379.99,"vol":727676,"amount":13991.028076056185}}`)
	require.NoError(t, e.wsHandleTickerMsg(t.Context(), sub, payload), "wsHandleData must not error")

	e.Websocket.DataHandler.Close()
	require.Len(t, e.Websocket.DataHandler.C, 1, "Must see correct number of records")
	tickAny := <-e.Websocket.DataHandler.C
	tick, ok := tickAny.Data.(*ticker.Price)
	require.True(t, ok, "Must get the correct type from DataHandler")
	assert.Equal(t, 13991.028076056185, tick.BaseVolume, "amount should be recorded as the base volume")
	assert.Zero(t, tick.QuoteVolume, "vol counts contracts here, so no quote volume should be recorded")
}
