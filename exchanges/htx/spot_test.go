package htx

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

// Please supply your own test keys here for due diligence testing.
const canManipulateRealOrders = false

var apiCredentials = &accounts.Credentials{
	Key:    "",
	Secret: "",
}

func init() {
	if os.Getenv("GCT_HTX_RUN_LIVE_TESTS") != "true" {
		apiCredentials = new(accounts.Credentials)
	}
}

var (
	_                  exchange.IBotExchange = (*Exchange)(nil)
	e                  *Exchange
	btcFutureDatedPair currency.Pair
	btccwPair          = currency.NewPair(currency.BTC, currency.NewCode("CW"))
	btcusdPair         = currency.NewPairWithDelimiter("BTC", "USD", "-")
	btcusdtPair        = currency.NewPairWithDelimiter("BTC", "USDT", "-")
	ethusdPair         = currency.NewPairWithDelimiter("ETH", "USD", "-")
)

func TestMain(m *testing.M) {
	e = new(Exchange)
	if err := testexch.Setup(e); err != nil {
		log.Fatalf("HTX Setup error: %s", err)
	}

	if apiCredentials.Key != "" && apiCredentials.Secret != "" {
		e.API.AuthenticatedSupport = true
		e.API.AuthenticatedWebsocketSupport = true
		e.SetCredentials(apiCredentials)
	}

	os.Exit(m.Run())
}

func TestGetSignatureHost(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name    string
		in      string
		want    string
		wantErr error
	}{
		{
			name: "spot",
			in:   "https://api.huobi.pro",
			want: "api.huobi.pro",
		},
		{
			name: "futures with path",
			in:   "https://api.hbdm.com/swap-api/v1",
			want: "api.hbdm.com",
		},
		{
			name: "custom host with port",
			in:   "https://localhost:8443",
			want: "localhost:8443",
		},
		{
			name:    "missing host",
			in:      "/v1/order/orders",
			wantErr: errInvalidEndpoint,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := getSignatureHost(tt.in)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr, "getSignatureHost must return expected error")
				return
			}
			require.NoError(t, err, "getSignatureHost must not error")
			assert.Equal(t, tt.want, got, "signature host should match")
		})
	}
}

func TestSendHTTPRequest(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name       string
		statusCode int
		body       string
		nilResult  bool
		expected   error
		expectedID uint64
	}{
		{
			name:       "JSON response",
			statusCode: http.StatusOK,
			body:       `{"status":"ok","data":1}`,
			expectedID: 1,
		},
		{
			name:       "no content without result",
			statusCode: http.StatusNoContent,
			nilResult:  true,
		},
		{
			name:       "no content with result",
			statusCode: http.StatusNoContent,
			expected:   errExpectedResponseBody,
		},
		{
			name:       "unexpected response without result",
			statusCode: http.StatusOK,
			body:       `{"status":"ok"}`,
			nilResult:  true,
			expected:   errUnexpectedResponseBody,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/public", r.URL.Path, "request path should match")
				if tc.body != "" {
					w.Header().Set("Content-Type", "application/json")
				}
				w.WriteHeader(tc.statusCode)
				_, _ = w.Write([]byte(tc.body))
			}))
			t.Cleanup(server.Close)

			h := new(Exchange)
			require.NoError(t, testexch.Setup(h), "HTX setup must not error")
			require.NoError(t, h.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL), "spot endpoint must be set")
			var response struct {
				Response
				Data uint64 `json:"data"`
			}
			var result any = &response
			if tc.nilResult {
				result = nil
			}
			err := h.SendHTTPRequest(t.Context(), exchange.RestSpot, "/public", result)
			if tc.expected != nil {
				require.ErrorIs(t, err, tc.expected, "SendHTTPRequest must return the expected error")
				return
			}
			require.NoError(t, err, "SendHTTPRequest must not error")
			assert.Equal(t, tc.expectedID, response.Data, "response data should match")
		})
	}
}

func TestSendAuthenticatedHTTPRequest(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name         string
		statusCode   int
		body         string
		authenticate bool
		nilResult    bool
		expected     []error
		expectedID   uint64
	}{
		{
			name:       "authentication required",
			statusCode: http.StatusOK,
			expected:   []error{exchange.ErrAuthenticationSupportNotEnabled},
		},
		{
			name:         "JSON response",
			statusCode:   http.StatusOK,
			body:         `{"status":"ok","data":1}`,
			authenticate: true,
			expectedID:   1,
		},
		{
			name:         "no content without result",
			statusCode:   http.StatusNoContent,
			authenticate: true,
			nilResult:    true,
		},
		{
			name:         "no content with result",
			statusCode:   http.StatusNoContent,
			authenticate: true,
			expected:     []error{errExpectedResponseBody, request.ErrAuthRequestFailed},
		},
		{
			name:         "unexpected response without result",
			statusCode:   http.StatusOK,
			body:         `{"status":"ok"}`,
			authenticate: true,
			nilResult:    true,
			expected:     []error{errUnexpectedResponseBody, request.ErrAuthRequestFailed},
		},
		{
			name:         "API error without result",
			statusCode:   http.StatusOK,
			body:         `{"status":"error","err-code":"bad-request","err-msg":"invalid request"}`,
			authenticate: true,
			nilResult:    true,
			expected:     []error{request.ErrAuthRequestFailed},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/v1/private", r.URL.Path, "request path should match")
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"), "request content type should match")
				if tc.body != "" {
					w.Header().Set("Content-Type", "application/json")
				}
				w.WriteHeader(tc.statusCode)
				_, _ = w.Write([]byte(tc.body))
			}))
			t.Cleanup(server.Close)

			h := new(Exchange)
			require.NoError(t, testexch.Setup(h), "HTX setup must not error")
			if tc.authenticate {
				h.API.AuthenticatedSupport = true
				h.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
			}
			require.NoError(t, h.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL), "spot endpoint must be set")
			var response struct {
				Response
				Data uint64 `json:"data"`
			}
			var result any = &response
			if tc.nilResult {
				result = nil
			}
			err := h.SendAuthenticatedHTTPRequest(t.Context(), exchange.RestSpot, http.MethodPost, "/private", nil, nil, result, false)
			for _, expected := range tc.expected {
				assert.ErrorIs(t, err, expected, "SendAuthenticatedHTTPRequest should return the expected error")
			}
			if len(tc.expected) != 0 {
				require.Error(t, err, "SendAuthenticatedHTTPRequest must return an error")
				return
			}
			require.NoError(t, err, "SendAuthenticatedHTTPRequest must not error")
			assert.Equal(t, tc.expectedID, response.Data, "response data should match")
		})
	}
}

func TestUnmarshalResponse(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name      string
		response  string
		nilResult bool
		expected  error
		malformed bool
	}{
		{name: "empty response without result", nilResult: true},
		{name: "whitespace response without result", response: " \n\t", nilResult: true},
		{name: "empty response with result", expected: errExpectedResponseBody},
		{name: "unexpected response without result", response: `{}`, nilResult: true, expected: errUnexpectedResponseBody},
		{name: "malformed response", response: `{`, malformed: true},
		{name: "empty data", response: `{"code":200,"msg":"","data":"","ts":1604312615051}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			response := new(FFinancialRecords)
			var result any = response
			if tc.nilResult {
				result = nil
			}
			err := unmarshalResponse(json.RawMessage(tc.response), result)
			if tc.expected != nil {
				require.ErrorIs(t, err, tc.expected, "unmarshalResponse must return the expected error")
				return
			}
			if tc.malformed {
				require.Error(t, err, "unmarshalResponse must return malformed JSON errors")
				return
			}
			require.NoError(t, err, "unmarshalResponse must not error")
			if result != nil {
				assert.Empty(t, response.Data.FinancialRecord, "empty data should produce an empty result")
			}
		})
	}
}

func TestSpotMatchResultsEndpoint(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "/order/matchresults", htxGetOrdersMatch, "spot match results endpoint should match HTX docs")
}

func TestGetCurrenciesIncludingChains(t *testing.T) {
	t.Parallel()
	r, err := e.GetCurrenciesIncludingChains(t.Context(), currency.EMPTYCODE)
	require.NoError(t, err)
	assert.Greater(t, len(r), 1, "should get more than one currency back")
	r, err = e.GetCurrenciesIncludingChains(t.Context(), currency.USDT)
	require.NoError(t, err)
	assert.Equal(t, 1, len(r), "Should only get one currency back")
}

func TestGetMarginRates(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestSpot, http.MethodGet, "/v1"+htxMarginRates, emptySuccessResponse, nil)
	_, err := h.GetMarginRates(t.Context(), btcusdtPair)
	require.NoError(t, err, "GetMarginRates must not error")
}

func TestGetSpotKline(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpotKline(t.Context(), KlinesRequestParams{Symbol: btcusdtPair, Period: "1min"})
	require.NoError(t, err)
}

func TestGetMarketDetailMerged(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarketDetailMerged(t.Context(), btcusdtPair)
	require.NoError(t, err)
}

func TestGetDepth(t *testing.T) {
	t.Parallel()
	_, err := e.GetDepth(t.Context(),
		&OrderBookDataRequestParams{
			Symbol: btcusdtPair,
			Type:   OrderBookDataRequestParamsTypeStep1,
		})
	require.NoError(t, err)
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetTrades(t.Context(), btcusdtPair)
	require.NoError(t, err)
}

func TestGetLatestSpotPrice(t *testing.T) {
	t.Parallel()
	_, err := e.GetLatestSpotPrice(t.Context(), btcusdtPair)
	require.NoError(t, err)
}

func TestGetTradeHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetTradeHistory(t.Context(), btcusdtPair, 50)
	require.NoError(t, err)
}

func TestGetMarketDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarketDetail(t.Context(), btcusdtPair)
	require.NoError(t, err)
}

func TestGetSymbols(t *testing.T) {
	t.Parallel()
	_, err := e.GetSymbols(t.Context())
	require.NoError(t, err)
}

func TestGetCurrencies(t *testing.T) {
	t.Parallel()
	_, err := e.GetCurrencies(t.Context())
	require.NoError(t, err)
}

func TestGet24HrMarketSummary(t *testing.T) {
	t.Parallel()
	_, err := e.Get24HrMarketSummary(t.Context(), btcusdtPair)
	require.NoError(t, err)
}

func TestGetAccounts(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestSpot, http.MethodGet, "/v1"+htxAccounts, emptySuccessResponse, nil)
	_, err := h.GetAccounts(t.Context())
	require.NoError(t, err, "GetAccounts must not error")
}

func TestGetAccountBalance(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestSpot, http.MethodGet, "/v1"+fmt.Sprintf(htxAccountBalance, "123"), emptySuccessResponse, nil)
	_, err := h.GetAccountBalance(t.Context(), "123")
	require.NoError(t, err, "GetAccountBalance must not error")
}

func TestGetAccountBalanceCredentialsError(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	h.API.AuthenticatedSupport = true
	_, err := h.GetAccountBalance(t.Context(), "1")
	require.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "GetAccountBalance must return credentials error")
}

func TestGetAggregatedBalance(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestSpot, http.MethodGet, "/v1"+htxAggregatedBalance, emptySuccessResponse, nil)
	_, err := h.GetAggregatedBalance(t.Context())
	require.NoError(t, err, "GetAggregatedBalance must not error")
}

func TestSpotNewOrder(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestSpot, http.MethodPost, "/v1"+htxOrderPlace, `{"status":"ok","data":"123"}`, nil)
	arg := SpotNewOrderRequestParams{
		Symbol:    btcusdtPair,
		AccountID: 123,
		Amount:    0.01,
		Price:     10.1,
		Type:      SpotNewOrderRequestTypeBuyLimit,
	}

	_, err := h.SpotNewOrder(t.Context(), &arg)
	require.NoError(t, err, "SpotNewOrder must not error")
}

func TestCancelExistingOrder(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestSpot, http.MethodPost, "/v1"+fmt.Sprintf(htxOrderCancel, "123"), `{"status":"ok","data":"123"}`, nil)
	_, err := h.CancelExistingOrder(t.Context(), 123)
	require.NoError(t, err, "CancelExistingOrder must not error")
}

func TestGetOrder(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestSpot, http.MethodGet, "/v1"+htxGetOrder, emptySuccessResponse, nil)
	_, err := h.GetOrder(t.Context(), 1337)
	require.NoError(t, err, "GetOrder must not error")
}

func TestGetOrderMatchResults(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestSpot, http.MethodGet, "/v1"+fmt.Sprintf(htxGetOrderMatch, "1337"), emptySuccessResponse, nil)
	_, err := h.GetOrderMatchResults(t.Context(), 1337)
	require.NoError(t, err, "GetOrderMatchResults must not error")
}

func TestGetOrders(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestSpot, http.MethodGet, "/v1"+htxGetOrders, emptySuccessResponse, nil)
	_, err := h.GetOrders(t.Context(), btcusdtPair, "buy-limit", "", "", "submitted", "", "", "10")
	require.NoError(t, err, "GetOrders must not error")
}

func TestGetOpenOrders(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestSpot, http.MethodGet, "/v1"+htxGetOpenOrders, emptySuccessResponse, nil)
	_, err := h.GetOpenOrders(t.Context(), btcusdtPair, "100009", "buy", 10)
	require.NoError(t, err, "GetOpenOrders must not error")
}

func TestGetOrdersMatch(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestSpot, http.MethodGet, "/v1"+htxGetOrdersMatch, emptySuccessResponse, nil)
	_, err := h.GetOrdersMatch(t.Context(), btcusdtPair, "buy-limit", "", "", "", "", "10")
	require.NoError(t, err, "GetOrdersMatch must not error")
}

func TestGetMarginLoanOrders(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestSpot, http.MethodGet, "/v1"+htxMarginLoanOrders, emptySuccessResponse, nil)
	_, err := h.GetMarginLoanOrders(t.Context(), btcusdtPair, "usdt", "", "", "", "", "", "10")
	require.NoError(t, err, "GetMarginLoanOrders must not error")
}

func TestGetMarginAccountBalance(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestSpot, http.MethodGet, "/v1"+htxMarginAccountBalance, emptySuccessResponse, nil)
	_, err := h.GetMarginAccountBalance(t.Context(), btcusdtPair)
	require.NoError(t, err, "GetMarginAccountBalance must not error")
}

func TestMarginTransfer(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name string
		in   bool
	}{
		{name: "in", in: true},
		{name: "out"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			path := "/v1" + htxMarginTransferOut
			if tt.in {
				path = "/v1" + htxMarginTransferIn
			}
			h := newHTTPTestExchange(t, exchange.RestSpot, http.MethodPost, path, `{"status":"ok","data":123}`, nil)
			_, err := h.MarginTransfer(t.Context(), btcusdtPair, "usdt", 1.25, tt.in)
			require.NoError(t, err, "MarginTransfer must not error")
		})
	}
}

func TestMarginOrder(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestSpot, http.MethodPost, "/v1"+htxMarginOrders, `{"status":"ok","data":123}`, nil)
	_, err := h.MarginOrder(t.Context(), btcusdtPair, "usdt", 1.25)
	require.NoError(t, err, "MarginOrder must not error")
}

func TestMarginRepayment(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestSpot, http.MethodPost, "/v1"+fmt.Sprintf(htxMarginRepay, "1"), `{"status":"ok","data":123}`, nil)
	_, err := h.MarginRepayment(t.Context(), 1, 1.25)
	require.NoError(t, err, "MarginRepayment must not error")
}

func TestWithdraw(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name    string
		code    currency.Code
		address string
		amount  float64
		wantErr error
	}{
		{name: "invalid", wantErr: errWithdrawDetailsUnset},
		{name: "credentials", code: currency.USDT, address: "address", amount: 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.wantErr != nil {
				h := new(Exchange)
				require.NoError(t, testexch.Setup(h), "HTX setup must not error")
				_, err := h.Withdraw(t.Context(), tt.code, tt.address, "", "trc20usdt", tt.amount, 0.1)
				require.ErrorIs(t, err, tt.wantErr, "Withdraw must return expected error")
				return
			}
			h := newHTTPTestExchange(t, exchange.RestSpot, http.MethodPost, "/v1"+htxWithdrawCreate, `{"status":"ok","data":123}`, nil)
			_, err := h.Withdraw(t.Context(), tt.code, tt.address, "", "trc20usdt", tt.amount, 0.1)
			require.NoError(t, err, "Withdraw must not error")
		})
	}
}

func TestCancelWithdraw(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestSpot, http.MethodPost, "/v1"+fmt.Sprintf(htxWithdrawCancel, "1337"), `{"status":"ok","data":123}`, nil)
	_, err := h.CancelWithdraw(t.Context(), 1337)
	require.NoError(t, err, "CancelWithdraw must not error")
}

func setFeeBuilder() *exchange.FeeBuilder {
	return &exchange.FeeBuilder{
		Amount:  1,
		FeeType: exchange.CryptocurrencyTradeFee,
		Pair: currency.NewPairWithDelimiter(currency.BTC.String(),
			currency.LTC.String(),
			"_"),
		PurchasePrice:       1,
		FiatCurrency:        currency.USD,
		BankTransactionType: exchange.WireTransfer,
	}
}

func TestQueryDepositAddress(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestSpot, http.MethodGet, "/v2"+htxAccountDepositAddress, `{"code":200,"data":[{"currency":"usdt","address":"address","addressTag":"","chain":"trc20usdt"}]}`, nil)
	_, err := h.QueryDepositAddress(t.Context(), currency.USDT)
	require.NoError(t, err, "QueryDepositAddress must not error")
}

func TestQueryWithdrawQuotas(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestSpot, http.MethodGet, "/v2"+htxAccountWithdrawQuota, emptySuccessResponse, nil)
	_, err := h.QueryWithdrawQuotas(t.Context(), currency.BTC.Lower().String())
	require.NoError(t, err, "QueryWithdrawQuotas must not error")
}

func TestSearchForExistedWithdrawsAndDeposits(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestSpot, http.MethodGet, "/v1"+htxWithdrawHistory, emptySuccessResponse, nil)
	_, err := h.SearchForExistedWithdrawsAndDeposits(t.Context(), currency.BTC, "deposit", "", 0, 100)
	require.NoError(t, err, "SearchForExistedWithdrawsAndDeposits must not error")
}

func TestCancelOrderBatch(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestSpot, http.MethodPost, "/v1"+htxOrderCancelBatch, `{"status":"ok","data":{"success":["1234"],"failed":[]}}`, nil)
	_, err := h.CancelOrderBatch(t.Context(), []string{"1234"}, nil)
	require.NoError(t, err, "CancelOrderBatch must not error")
}

func TestCancelOpenOrdersBatch(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestSpot, http.MethodPost, "/v1"+htxBatchCancelOpenOrders, `{"status":"ok","data":{"success-count":1,"failed-count":0,"next-id":0}}`, nil)
	_, err := h.CancelOpenOrdersBatch(t.Context(), "1", btcusdtPair)
	require.NoError(t, err, "CancelOpenOrdersBatch must not error")
}

func TestGetBatchLinearSwapContracts(t *testing.T) {
	t.Parallel()
	resp, err := e.GetBatchLinearSwapContracts(t.Context())
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetBatchFuturesContracts(t *testing.T) {
	t.Parallel()
	resp, err := e.GetBatchFuturesContracts(t.Context())
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func updatePairsOnce(tb testing.TB, h *Exchange) {
	tb.Helper()

	updatePairsMutex.Lock()
	defer updatePairsMutex.Unlock()

	testexch.UpdatePairsOnce(tb, h)

	h.futureContractCodesMutex.Lock()
	if len(h.futureContractCodes) == 0 {
		// Restored pairs from cache, so haven't populated futureContract Codes
		require.NotEmpty(tb, futureContractCodesCache, "futureContractCodesCache must not be empty")
		h.futureContractCodes = futureContractCodesCache
	} else {
		futureContractCodesCache = h.futureContractCodes
	}
	h.futureContractCodesMutex.Unlock()

	if btcFutureDatedPair.Equal(currency.EMPTYPAIR) {
		p, err := h.pairFromContractExpiryCode(btccwPair)
		require.NoError(tb, err, "pairFromContractCode must not error")
		btcFutureDatedPair = p
	}

	err := h.CurrencyPairs.EnablePair(asset.Futures, btcFutureDatedPair) // Must enable every time we refresh the CurrencyPairs from cache
	require.NoError(tb, common.ExcludeError(err, currency.ErrPairAlreadyEnabled))
}

func TestGetTickers(t *testing.T) {
	t.Parallel()
	_, err := e.GetTickers(t.Context())
	require.NoError(t, err)
}

func TestGetCurrentServerTime(t *testing.T) {
	t.Parallel()
	st, err := e.GetCurrentServerTime(t.Context())
	require.NoError(t, err)
	assert.NotEmpty(t, st, "GetCurrentServerTime should return a time")
}

func TestGetFee(t *testing.T) {
	t.Parallel()
	feeBuilder := setFeeBuilder()
	// CryptocurrencyTradeFee Basic
	_, err := e.GetFee(feeBuilder)
	require.NoError(t, err)

	// CryptocurrencyTradeFee High quantity
	feeBuilder = setFeeBuilder()
	feeBuilder.Amount = 1000
	feeBuilder.PurchasePrice = 1000
	_, err = e.GetFee(feeBuilder)
	require.NoError(t, err)

	// CryptocurrencyTradeFee IsMaker
	feeBuilder = setFeeBuilder()
	feeBuilder.IsMaker = true
	_, err = e.GetFee(feeBuilder)
	require.NoError(t, err)

	// CryptocurrencyTradeFee Negative purchase price
	feeBuilder = setFeeBuilder()
	feeBuilder.PurchasePrice = -1000
	_, err = e.GetFee(feeBuilder)
	require.NoError(t, err)

	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	_, err = e.GetFee(feeBuilder)
	require.NoError(t, err)

	// CryptocurrencyWithdrawalFee Invalid currency
	feeBuilder = setFeeBuilder()
	feeBuilder.Pair.Base = currency.NewCode("hello")
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	_, err = e.GetFee(feeBuilder)
	require.NoError(t, err)

	// CryptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyDepositFee
	_, err = e.GetFee(feeBuilder)
	require.NoError(t, err)

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	_, err = e.GetFee(feeBuilder)
	require.NoError(t, err)

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.USD
	_, err = e.GetFee(feeBuilder)
	require.NoError(t, err)
}

func TestCalculateTradingFee(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		pair     currency.Pair
		expected float64
	}{
		{name: "crypto fiat", pair: currency.NewBTCUSD(), expected: 0.1},
		{name: "non-fiat quote", pair: currency.NewPair(currency.BTC, currency.ETH), expected: 0.2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, calculateTradingFee(tc.pair, 100, 1), "trading fee should use the documented rate")
		})
	}
}

func TestGetBatchCoinMarginSwapContracts(t *testing.T) {
	t.Parallel()
	resp, err := e.GetBatchCoinMarginSwapContracts(t.Context())
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}
