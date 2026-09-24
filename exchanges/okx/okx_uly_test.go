package okx

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

// TestDeprecatedUlyReplacedByInstFamily pins the uly to instFamily migration
// across the affected endpoints: OKX removed uly from their documented
// parameters in favour of instFamily, and silently drops uly if it is sent,
// so every case asserts instFamily carries the filter and uly stays absent.
func TestDeprecatedUlyReplacedByInstFamily(t *testing.T) {
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")

	var mu sync.Mutex
	var gotPath string
	var gotQuery url.Values

	srv := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotPath = r.URL.Path
		gotQuery = r.URL.Query()
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		data := `{"code":"0","msg":"","data":[]}`
		if r.URL.Path == "/public/insurance-fund" || r.URL.Path == "/public/liquidation-orders" {
			// Single-pointer responses need a one-item array
			data = `{"code":"0","msg":"","data":[{}]}`
		}
		_, _ = w.Write([]byte(data))
	}))

	b := e.GetBase()
	b.SkipAuthCheck = true
	require.NoError(t, e.SetHTTPClient(srv.Client()), "SetHTTPClient must not error")
	for k := range b.API.Endpoints.GetURLMap() {
		require.NoErrorf(t, b.API.Endpoints.SetRunningURL(k, srv.URL+"/"), "Setup must point endpoint %s at the mock server", k)
	}

	for _, tc := range []struct {
		name   string
		call   func() error
		path   string
		family string
	}{
		{
			name: "tickers",
			call: func() error {
				_, err := e.GetTickers(t.Context(), instTypeSwap, "BTC-USDT")
				return err
			},
			path:   "/market/tickers",
			family: "BTC-USDT",
		},
		{
			name: "block tickers",
			call: func() error {
				_, err := e.GetBlockTickers(t.Context(), instTypeSwap, "BTC-USDT")
				return err
			},
			path:   "/market/block-tickers",
			family: "BTC-USDT",
		},
		{
			name: "mark price",
			call: func() error {
				_, err := e.GetMarkPrice(t.Context(), instTypeSwap, "BTC-USDT", "")
				return err
			},
			path:   "/public/mark-price",
			family: "BTC-USDT",
		},
		{
			// The live endpoint rejects instFamily for most option families
			// while accepting the deprecated uly, so this filter must stay on
			// uly; pinned here so the migration does not sweep it up.
			name: "public instruments keeps uly",
			call: func() error {
				_, err := e.GetInstruments(t.Context(), &InstrumentsFetchParams{
					InstrumentType: instTypeSwap,
					Underlying:     "BTC-USDT",
				})
				return err
			},
			path:   "/public/instruments",
			family: "",
		},
		{
			name: "pending order list",
			call: func() error {
				_, err := e.GetOrderList(t.Context(), &OrderListRequestParams{
					InstrumentType: instTypeSpot,
					Underlying:     "BTC-USDT",
				})
				return err
			},
			path:   "/trade/orders-pending",
			family: "BTC-USDT",
		},
		{
			name: "account instruments",
			call: func() error {
				_, err := e.GetAccountInstruments(t.Context(), 2, "BTC-USDT", "")
				return err
			},
			path:   "/account/instruments",
			family: "BTC-USDT",
		},
		{
			name: "portfolio margin position tiers",
			call: func() error {
				_, err := e.GetPMPositionLimitation(t.Context(), instTypeSwap, "BTC-USDT")
				return err
			},
			path:   "/account/position-tiers",
			family: "BTC-USDT",
		},
		{
			name: "trade fee",
			call: func() error {
				_, err := e.GetTradeFee(t.Context(), instTypeSpot, "", "BTC-USDT", "")
				return err
			},
			path:   "/account/trade-fee",
			family: "BTC-USDT",
		},
		{
			name: "open interest swap family",
			call: func() error {
				_, err := e.GetOpenInterestData(t.Context(), instTypeSwap, "BTC-USDT", "")
				return err
			},
			path:   "/public/open-interest",
			family: "BTC-USDT",
		},
		{
			// The live endpoint rejects instFamily for option families while
			// accepting uly, so option underlyings must keep travelling as uly.
			name: "open interest option underlying keeps uly",
			call: func() error {
				_, err := e.GetOpenInterestData(t.Context(), instTypeOption, "SOL-USD", "")
				return err
			},
			path:   "/public/open-interest",
			family: "",
		},
		{
			name: "option market data",
			call: func() error {
				_, err := e.GetOptionMarketData(t.Context(), "BTC-USD", time.Time{})
				return err
			},
			path:   "/public/opt-summary",
			family: "BTC-USD",
		},
		{
			name: "delivery history",
			call: func() error {
				_, err := e.GetDeliveryHistory(t.Context(), instTypeFutures, "BTC-USD", time.Time{}, time.Time{}, 0)
				return err
			},
			path:   "/public/delivery-exercise-history",
			family: "BTC-USD",
		},
		{
			name: "position tiers",
			call: func() error {
				_, err := e.GetPositionTiers(t.Context(), instTypeSwap, TradeModeCross, "BTC-USDT", "BTC-USDT-SWAP", "", currency.EMPTYCODE)
				return err
			},
			path:   "/public/position-tiers",
			family: "BTC-USDT",
		},
		{
			name: "insurance fund",
			call: func() error {
				_, err := e.GetInsuranceFundInformation(t.Context(), &InsuranceFundInformationRequestParams{
					InstrumentType:   instTypeSwap,
					InstrumentFamily: "BTC-USDT",
				})
				return err
			},
			path:   "/public/insurance-fund",
			family: "BTC-USDT",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.NoError(t, tc.call(), "the migrated request should not error")
			mu.Lock()
			path, query := gotPath, gotQuery
			mu.Unlock()
			assert.Equal(t, tc.path, path, "the documented endpoint should be requested")
			switch tc.name {
			case "public instruments keeps uly":
				assert.Equal(t, "BTC-USDT", query.Get("uly"), "the underlying filter must keep travelling as uly on public/instruments")
				assert.NotContains(t, query, "instFamily", "the live endpoint rejects instFamily for most option families")
				return
			case "open interest option underlying keeps uly":
				assert.Equal(t, "SOL-USD", query.Get("uly"), "the option underlying filter must keep travelling as uly on public/open-interest")
				assert.NotContains(t, query, "instFamily", "the live endpoint rejects instFamily for option families")
				return
			}
			assert.Equalf(t, tc.family, query.Get("instFamily"), "the family filter should travel as instFamily")
			assert.NotContains(t, query, "uly", "OKX dropped the deprecated uly parameter, so it must not be sent")
		})
	}
}
