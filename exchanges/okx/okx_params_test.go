package okx

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

// TestDocsPinnedRequestParameters pins the wire parameter names and endpoint
// routes against the OKX v5 documentation. OKX silently ignores unknown query
// parameters, so a misnamed filter fails quietly: every case asserts the
// documented name is sent and the broken name this PR replaces stays absent.
func TestDocsPinnedRequestParameters(t *testing.T) {
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")

	var mu sync.Mutex
	var gotPath string
	var gotQuery url.Values

	srv := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("reading request body should not error: %v", err)
			return
		}
		_ = body
		mu.Lock()
		gotPath = r.URL.Path
		gotQuery = r.URL.Query()
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		data := `{"code":"0","msg":"","data":[]}`
		if r.URL.Path == "/tradingBot/recurring/orders-algo-details" {
			// GetRecurringOrderDetails decodes a single item from the array
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

	lastRequest := func() (string, url.Values) {
		mu.Lock()
		defer mu.Unlock()
		return gotPath, gotQuery
	}

	for _, tc := range []struct {
		name     string
		call     func() error
		path     string
		params   map[string]string
		absent   []string
		absentOk bool
	}{
		{
			name: "RFQs send clRfqId",
			call: func() error {
				_, err := e.GetRFQs(t.Context(), &RFQRequestParams{ClientRFQID: "rfq-client-1"})
				return err
			},
			path:   "/rfq/rfqs",
			params: map[string]string{"clRfqId": "rfq-client-1"},
			absent: []string{"clRFQId"},
		},
		{
			name: "Quotes send clRfqId",
			call: func() error {
				_, err := e.GetQuotes(t.Context(), &QuoteRequestParams{ClientRFQID: "rfq-client-1"})
				return err
			},
			path:   "/rfq/quotes",
			params: map[string]string{"clRfqId": "rfq-client-1"},
			absent: []string{"clRFQId"},
		},
		{
			name: "RFQ trades send clRfqId without state",
			call: func() error {
				_, err := e.GetRFQTrades(t.Context(), &RFQTradesRequestParams{ClientRFQID: "rfq-client-1"})
				return err
			},
			path:   "/rfq/trades",
			params: map[string]string{"clRfqId": "rfq-client-1"},
			absent: []string{"clRFQId", "state"},
		},
		{
			name: "Spread order books size uses sz",
			call: func() error {
				_, err := e.GetPublicSpreadOrderBooks(t.Context(), "BTC-USDT_BTC-USDT-SWAP", 50)
				return err
			},
			path:   "/sprd/books",
			params: map[string]string{"sz": "50"},
			absent: []string{"size"},
		},
		{
			name: "Subaccount bills filter uses subAcct",
			call: func() error {
				_, err := e.HistoryOfSubaccountTransfer(t.Context(), currency.BTC, "", "sub-one", time.Time{}, time.Time{}, 0)
				return err
			},
			path:   "/asset/subaccount/bills",
			params: map[string]string{"subAcct": "sub-one"},
			absent: []string{"subacct", "setAcct"},
		},
		{
			name: "Custody subaccount list filter uses subAcct",
			call: func() error {
				_, err := e.GetCustodyTradingSubaccountList(t.Context(), "sub-one")
				return err
			},
			path:   "/users/entrust-subaccount-list",
			params: map[string]string{"subAcct": "sub-one"},
			absent: []string{"setAcct"},
		},
		{
			name: "Position tiers filter uses singular tier",
			call: func() error {
				_, err := e.GetPositionTiers(t.Context(), instTypeSpot, "cross", "", "", "", "1", currency.EMPTYCODE)
				return err
			},
			path:   "/public/position-tiers",
			params: map[string]string{"tier": "1"},
			absent: []string{"tiers"},
		},
		{
			name: "Daily lead trader PNL hits the daily endpoint",
			call: func() error {
				_, err := e.GetDailyLeadTraderPNL(t.Context(), "SWAP", "trader-1", "2")
				return err
			},
			path:   "/copytrading/public-pnl",
			params: map[string]string{"uniqueCode": "trader-1", "lastDays": "2"},
		},
		{
			name: "Lead trader currency preferences drop lastDays",
			call: func() error {
				_, err := e.GetLeadTraderCurrencyPreferences(t.Context(), "SWAP", "trader-1")
				return err
			},
			path:   "/copytrading/public-preference-currency",
			params: map[string]string{"uniqueCode": "trader-1"},
			absent: []string{"lastDays"},
		},
		{
			name: "Trade fee drops the response-only ruleType",
			call: func() error {
				_, err := e.GetTradeFee(t.Context(), instTypeSpot, "BTC-USDT", "", "")
				return err
			},
			path:   "/account/trade-fee",
			params: map[string]string{"instType": "SPOT", "instId": "BTC-USDT"},
			absent: []string{"ruleType"},
		},
		{
			name: "Max buy or sell amount drops unSpotOffset",
			call: func() error {
				_, err := e.GetMaximumBuySellAmountOROpenAmount(t.Context(), currency.BTC, "BTC-USDT", "cash", "", 5)
				return err
			},
			path:   "/account/max-size",
			params: map[string]string{"instId": "BTC-USDT"},
			absent: []string{"unSpotOffset"},
		},
		{
			name: "Max available amount drops quickMgnType and upSpotOffset",
			call: func() error {
				_, err := e.GetMaximumAvailableTradableAmount(t.Context(), currency.BTC, "BTC-USDT", "cash", false, 5)
				return err
			},
			path:   "/account/max-avail-size",
			params: map[string]string{"instId": "BTC-USDT"},
			absent: []string{"quickMgnType", "upSpotOffset"},
		},
		{
			name: "Pending algo order list drops unsupported clOrdId",
			call: func() error {
				_, err := e.GetAlgoOrderList(t.Context(), "conditional", "", "", "", time.Time{}, time.Time{}, 1)
				return err
			},
			path:   "/trade/orders-algo-pending",
			params: map[string]string{"ordType": "conditional"},
			absent: []string{"clOrdId"},
		},
		{
			name: "Recurring buy order list drops state",
			call: func() error {
				_, err := e.GetRecurringBuyOrderList(t.Context(), "", time.Time{}, time.Time{}, 30)
				return err
			},
			path:   "/tradingBot/recurring/orders-algo-pending",
			params: map[string]string{"limit": "30"},
			absent: []string{"state"},
		},
		{
			name: "Recurring order details drop state",
			call: func() error {
				_, err := e.GetRecurringOrderDetails(t.Context(), "560473220642766848")
				return err
			},
			path:   "/tradingBot/recurring/orders-algo-details",
			params: map[string]string{"algoId": "560473220642766848"},
			absent: []string{"state"},
		},
		{
			name: "Taker volume drops instFamily",
			call: func() error {
				_, err := e.GetTakerVolume(t.Context(), currency.BTC, instTypeSpot, time.Time{}, time.Time{}, kline.OneDay)
				return err
			},
			path:   "/rubik/stat/taker-volume",
			params: map[string]string{"instType": "SPOT"},
			absent: []string{"instFamily"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.NoError(t, tc.call(), "the pinned request should not error")
			path, query := lastRequest()
			assert.Equal(t, tc.path, path, "the documented endpoint should be requested")
			for name, want := range tc.params {
				assert.Equalf(t, want, query.Get(name), "parameter %s should carry the documented value", name)
			}
			for _, name := range tc.absent {
				assert.NotContains(t, query, name, "OKX ignores an undocumented %s parameter, so it must not be sent", name)
			}
		})
	}
}
