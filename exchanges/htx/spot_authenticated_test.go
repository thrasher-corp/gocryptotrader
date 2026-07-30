package htx

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

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
