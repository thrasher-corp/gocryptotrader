package htx

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetMarginRates(t.Context(), btcusdtPair)
	require.NoError(t, err)
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.GetAccounts(t.Context())
	require.NoError(t, err)
}

func TestGetAccountBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.GetAccounts(t.Context())
	require.NoError(t, err, "GetAccounts must not error")

	userID := strconv.FormatInt(result[0].ID, 10)
	_, err = e.GetAccountBalance(t.Context(), userID)
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetAggregatedBalance(t.Context())
	require.NoError(t, err)
}

func TestSpotNewOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	arg := SpotNewOrderRequestParams{
		Symbol:    btcusdtPair,
		AccountID: 1997024,
		Amount:    0.01,
		Price:     10.1,
		Type:      SpotNewOrderRequestTypeBuyLimit,
	}

	_, err := e.SpotNewOrder(t.Context(), &arg)
	require.NoError(t, err)
}

func TestCancelExistingOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.CancelExistingOrder(t.Context(), 1337)
	assert.Error(t, err)
}

func TestGetOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.GetOrder(t.Context(), 1337)
	require.NoError(t, err)
}

func TestGetOrderMatchResults(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	h.API.AuthenticatedSupport = true
	_, err := h.GetOrderMatchResults(t.Context(), 1337)
	require.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "GetOrderMatchResults must return credentials error")
}

func TestGetOrders(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	h.API.AuthenticatedSupport = true
	_, err := h.GetOrders(t.Context(), btcusdtPair, "buy-limit", "2019-03-10", "2019-03-19", "submitted", "5", "prev", "100")
	require.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "GetOrders must return credentials error")
}

func TestGetOpenOrders(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	h.API.AuthenticatedSupport = true
	_, err := h.GetOpenOrders(t.Context(), btcusdtPair, "100009", "buy", 10)
	require.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "GetOpenOrders must return credentials error")
}

func TestGetOrdersMatch(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	h.API.AuthenticatedSupport = true
	_, err := h.GetOrdersMatch(t.Context(), btcusdtPair, "buy-limit", "2019-03-10", "2019-03-19", "5", "prev", "100")
	require.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "GetOrdersMatch must return credentials error")
}

func TestGetMarginLoanOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetMarginLoanOrders(t.Context(), btcusdtPair, "", "", "", "", "", "", "")
	require.NoError(t, err)
}

func TestGetMarginAccountBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetMarginAccountBalance(t.Context(), btcusdtPair)
	require.NoError(t, err)
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
			h := new(Exchange)
			require.NoError(t, testexch.Setup(h), "HTX setup must not error")
			h.API.AuthenticatedSupport = true
			_, err := h.MarginTransfer(t.Context(), btcusdtPair, "usdt", 1.25, tt.in)
			require.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "MarginTransfer must return credentials error")
		})
	}
}

func TestMarginOrder(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	h.API.AuthenticatedSupport = true
	_, err := h.MarginOrder(t.Context(), btcusdtPair, "usdt", 1.25)
	require.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "MarginOrder must return credentials error")
}

func TestMarginRepayment(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	h.API.AuthenticatedSupport = true
	_, err := h.MarginRepayment(t.Context(), 1, 1.25)
	require.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "MarginRepayment must return credentials error")
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
			h := new(Exchange)
			require.NoError(t, testexch.Setup(h), "HTX setup must not error")
			h.API.AuthenticatedSupport = true
			_, err := h.Withdraw(t.Context(), tt.code, tt.address, "", "trc20usdt", tt.amount, 0.1)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr, "Withdraw must return expected error")
				return
			}
			require.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "Withdraw must return credentials error")
		})
	}
}

func TestCancelWithdraw(t *testing.T) {
	t.Parallel()
	t.Run("credentials", func(t *testing.T) {
		t.Parallel()
		h := new(Exchange)
		require.NoError(t, testexch.Setup(h), "HTX setup must not error")
		h.API.AuthenticatedSupport = true
		_, err := h.CancelWithdraw(t.Context(), 1337)
		require.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "CancelWithdraw must return credentials error")
	})
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		_, err := e.CancelWithdraw(t.Context(), 1337)
		require.Error(t, err)
	})
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
	_, err := e.QueryDepositAddress(t.Context(), currency.USDT)
	if sharedtestvalues.AreAPICredentialsSet(e) {
		require.NoError(t, err)
	} else {
		require.ErrorIs(t, err, exchange.ErrAuthenticationSupportNotEnabled)
	}
}

func TestQueryWithdrawQuotas(t *testing.T) {
	t.Parallel()
	_, err := e.QueryWithdrawQuotas(t.Context(), currency.BTC.Lower().String())
	if sharedtestvalues.AreAPICredentialsSet(e) {
		require.NoError(t, err)
	} else {
		require.ErrorIs(t, err, exchange.ErrAuthenticationSupportNotEnabled)
	}
}

func TestSearchForExistedWithdrawsAndDeposits(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.SearchForExistedWithdrawsAndDeposits(t.Context(), currency.BTC, "deposit", "", 0, 100)
	require.NoError(t, err)
}

func TestCancelOrderBatch(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.CancelOrderBatch(t.Context(), []string{"1234"}, nil)
	require.NoError(t, err)
}

func TestCancelOpenOrdersBatch(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	_, err := h.CancelOpenOrdersBatch(t.Context(), "1", btcusdtPair)
	require.ErrorIs(t, err, exchange.ErrAuthenticationSupportNotEnabled, "CancelOpenOrdersBatch must require credentials")
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

func TestSpotAuthenticatedEndpoints(t *testing.T) {
	t.Parallel()
	pair := currency.NewBTCUSDT()
	for _, tc := range []struct {
		name     string
		method   string
		path     string
		response string
		call     func(*Exchange) error
	}{
		{
			name:   "GetMarginRates",
			method: http.MethodGet,
			path:   "/v1" + htxMarginRates,
			call: func(h *Exchange) error {
				_, err := h.GetMarginRates(t.Context(), pair)
				return err
			},
		},
		{
			name:   "GetAccounts",
			method: http.MethodGet,
			path:   "/v1" + htxAccounts,
			call: func(h *Exchange) error {
				_, err := h.GetAccounts(t.Context())
				return err
			},
		},
		{
			name:   "GetAccountBalance",
			method: http.MethodGet,
			path:   "/v1" + fmt.Sprintf(htxAccountBalance, "123"),
			call: func(h *Exchange) error {
				_, err := h.GetAccountBalance(t.Context(), "123")
				return err
			},
		},
		{
			name:   "GetAggregatedBalance",
			method: http.MethodGet,
			path:   "/v1" + htxAggregatedBalance,
			call: func(h *Exchange) error {
				_, err := h.GetAggregatedBalance(t.Context())
				return err
			},
		},
		{
			name:     "SpotNewOrder",
			method:   http.MethodPost,
			path:     "/v1" + htxOrderPlace,
			response: `{"status":"ok","data":"123"}`,
			call: func(h *Exchange) error {
				_, err := h.SpotNewOrder(t.Context(), &SpotNewOrderRequestParams{
					Symbol:    pair,
					AccountID: 123,
					Amount:    1,
					Price:     1,
					Type:      SpotNewOrderRequestTypeBuyLimit,
				})
				return err
			},
		},
		{
			name:     "CancelExistingOrder",
			method:   http.MethodPost,
			path:     "/v1" + fmt.Sprintf(htxOrderCancel, "123"),
			response: `{"status":"ok","data":"123"}`,
			call: func(h *Exchange) error {
				_, err := h.CancelExistingOrder(t.Context(), 123)
				return err
			},
		},
		{
			name:     "CancelOrderBatch",
			method:   http.MethodPost,
			path:     "/v1" + htxOrderCancelBatch,
			response: `{"status":"ok","data":{"success":["123"],"failed":[]}}`,
			call: func(h *Exchange) error {
				_, err := h.CancelOrderBatch(t.Context(), []string{"123"}, nil)
				return err
			},
		},
		{
			name:     "CancelOpenOrdersBatch",
			method:   http.MethodPost,
			path:     "/v1" + htxBatchCancelOpenOrders,
			response: `{"status":"ok","data":{"success-count":1,"failed-count":0,"next-id":0}}`,
			call: func(h *Exchange) error {
				_, err := h.CancelOpenOrdersBatch(t.Context(), "123", pair)
				return err
			},
		},
		{
			name:   "GetOrder",
			method: http.MethodGet,
			path:   "/v1" + htxGetOrder,
			call: func(h *Exchange) error {
				_, err := h.GetOrder(t.Context(), 123)
				return err
			},
		},
		{
			name:   "GetOrderMatchResults",
			method: http.MethodGet,
			path:   "/v1" + fmt.Sprintf(htxGetOrderMatch, "123"),
			call: func(h *Exchange) error {
				_, err := h.GetOrderMatchResults(t.Context(), 123)
				return err
			},
		},
		{
			name:   "GetOrders",
			method: http.MethodGet,
			path:   "/v1" + htxGetOrders,
			call: func(h *Exchange) error {
				_, err := h.GetOrders(t.Context(), pair, "buy-limit", "", "", "submitted", "", "", "10")
				return err
			},
		},
		{
			name:   "GetOpenOrders",
			method: http.MethodGet,
			path:   "/v1" + htxGetOpenOrders,
			call: func(h *Exchange) error {
				_, err := h.GetOpenOrders(t.Context(), pair, "123", "buy", 10)
				return err
			},
		},
		{
			name:   "GetOrdersMatch",
			method: http.MethodGet,
			path:   "/v1" + htxGetOrdersMatch,
			call: func(h *Exchange) error {
				_, err := h.GetOrdersMatch(t.Context(), pair, "buy-limit", "", "", "", "", "10")
				return err
			},
		},
		{
			name:     "MarginTransfer/in",
			method:   http.MethodPost,
			path:     "/v1" + htxMarginTransferIn,
			response: `{"status":"ok","data":123}`,
			call: func(h *Exchange) error {
				_, err := h.MarginTransfer(t.Context(), pair, "usdt", 1, true)
				return err
			},
		},
		{
			name:     "MarginTransfer/out",
			method:   http.MethodPost,
			path:     "/v1" + htxMarginTransferOut,
			response: `{"status":"ok","data":123}`,
			call: func(h *Exchange) error {
				_, err := h.MarginTransfer(t.Context(), pair, "usdt", 1, false)
				return err
			},
		},
		{
			name:     "MarginOrder",
			method:   http.MethodPost,
			path:     "/v1" + htxMarginOrders,
			response: `{"status":"ok","data":123}`,
			call: func(h *Exchange) error {
				_, err := h.MarginOrder(t.Context(), pair, "usdt", 1)
				return err
			},
		},
		{
			name:     "MarginRepayment",
			method:   http.MethodPost,
			path:     "/v1" + fmt.Sprintf(htxMarginRepay, "123"),
			response: `{"status":"ok","data":123}`,
			call: func(h *Exchange) error {
				_, err := h.MarginRepayment(t.Context(), 123, 1)
				return err
			},
		},
		{
			name:   "GetMarginLoanOrders",
			method: http.MethodGet,
			path:   "/v1" + htxMarginLoanOrders,
			call: func(h *Exchange) error {
				_, err := h.GetMarginLoanOrders(t.Context(), pair, "usdt", "", "", "", "", "", "10")
				return err
			},
		},
		{
			name:   "GetMarginAccountBalance",
			method: http.MethodGet,
			path:   "/v1" + htxMarginAccountBalance,
			call: func(h *Exchange) error {
				_, err := h.GetMarginAccountBalance(t.Context(), pair)
				return err
			},
		},
		{
			name:     "Withdraw",
			method:   http.MethodPost,
			path:     "/v1" + htxWithdrawCreate,
			response: `{"status":"ok","data":123}`,
			call: func(h *Exchange) error {
				_, err := h.Withdraw(t.Context(), currency.USDT, "address", "", "trc20usdt", 1, 0.1)
				return err
			},
		},
		{
			name:     "CancelWithdraw",
			method:   http.MethodPost,
			path:     "/v1" + fmt.Sprintf(htxWithdrawCancel, "123"),
			response: `{"status":"ok","data":123}`,
			call: func(h *Exchange) error {
				_, err := h.CancelWithdraw(t.Context(), 123)
				return err
			},
		},
		{
			name:     "QueryDepositAddress",
			method:   http.MethodGet,
			path:     "/v2" + htxAccountDepositAddress,
			response: `{"code":200,"data":[{"currency":"usdt","address":"address","addressTag":"","chain":"trc20usdt"}]}`,
			call: func(h *Exchange) error {
				_, err := h.QueryDepositAddress(t.Context(), currency.USDT)
				return err
			},
		},
		{
			name:   "QueryWithdrawQuotas",
			method: http.MethodGet,
			path:   "/v2" + htxAccountWithdrawQuota,
			call: func(h *Exchange) error {
				_, err := h.QueryWithdrawQuotas(t.Context(), "usdt")
				return err
			},
		},
		{
			name:   "SearchForExistedWithdrawsAndDeposits",
			method: http.MethodGet,
			path:   "/v1" + htxWithdrawHistory,
			call: func(h *Exchange) error {
				_, err := h.SearchForExistedWithdrawsAndDeposits(t.Context(), currency.USDT, "deposit", "next", 1, 10)
				return err
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, tc.method, r.Method, "authenticated spot method should match")
				assert.Equal(t, tc.path, r.URL.Path, "authenticated spot path should match HTX documentation")
				if tc.method == http.MethodGet {
					assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"), "authenticated spot GET content type should match")
				} else {
					assert.Equal(t, "application/json", r.Header.Get("Content-Type"), "authenticated spot POST content type should match")
				}
				w.Header().Set("Content-Type", "application/json")
				response := tc.response
				if response == "" {
					response = `{"status":"ok","code":200,"data":null}`
				}
				_, _ = w.Write([]byte(response))
			}))
			t.Cleanup(server.Close)

			h := new(Exchange)
			require.NoError(t, testexch.Setup(h), "HTX setup must not error")
			h.API.AuthenticatedSupport = true
			h.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
			require.NoError(t, h.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL), "spot endpoint must be set")
			require.NoError(t, tc.call(h), "authenticated spot endpoint must not error")
		})
	}
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
