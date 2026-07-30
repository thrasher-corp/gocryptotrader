package htx

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func TestCoinMarginedAuthenticatedEndpoints(t *testing.T) {
	t.Parallel()
	contractCode := currency.NewPairWithDelimiter("ETH", "USD", "-")
	startTime := time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name          string
		path          string
		expectedField string
		expectedValue any
		call          func(*Exchange) error
	}{
		{
			name: "GetSwapAccountInfo",
			path: "/swap-api/v1/swap_account_info",
			call: func(h *Exchange) error {
				_, err := h.GetSwapAccountInfo(t.Context(), contractCode)
				return err
			},
		},
		{
			name: "GetSwapPositionsInfo",
			path: "/swap-api/v1/swap_position_info",
			call: func(h *Exchange) error {
				_, err := h.GetSwapPositionsInfo(t.Context(), contractCode)
				return err
			},
		},
		{
			name: "GetSwapAssetsAndPositions",
			path: "/swap-api/v1/swap_account_position_info",
			call: func(h *Exchange) error {
				_, err := h.GetSwapAssetsAndPositions(t.Context(), contractCode)
				return err
			},
		},
		{
			name: "GetSwapAllSubAccAssets",
			path: "/swap-api/v1/swap_sub_account_list",
			call: func(h *Exchange) error {
				_, err := h.GetSwapAllSubAccAssets(t.Context(), contractCode)
				return err
			},
		},
		{
			name: "SwapSingleSubAccAssets",
			path: "/swap-api/v1/swap_sub_account_info",
			call: func(h *Exchange) error {
				_, err := h.SwapSingleSubAccAssets(t.Context(), contractCode, 123)
				return err
			},
		},
		{
			name: "GetSubAccPositionInfo",
			path: "/swap-api/v1/swap_sub_position_info",
			call: func(h *Exchange) error {
				_, err := h.GetSubAccPositionInfo(t.Context(), contractCode, 123)
				return err
			},
		},
		{
			name: "GetAccountFinancialRecords",
			path: "/swap-api/v3/swap_financial_record",
			call: func(h *Exchange) error {
				_, err := h.GetAccountFinancialRecords(t.Context(), contractCode, "3,4", 2, 1, 20)
				return err
			},
		},
		{
			name: "GetSwapSettlementRecords",
			path: "/swap-api/v1/swap_user_settlement_records",
			call: func(h *Exchange) error {
				_, err := h.GetSwapSettlementRecords(t.Context(), contractCode, startTime, startTime.Add(time.Hour), 1, 20)
				return err
			},
		},
		{
			name: "GetAvailableLeverage",
			path: "/swap-api/v1/swap_available_level_rate",
			call: func(h *Exchange) error {
				_, err := h.GetAvailableLeverage(t.Context(), contractCode)
				return err
			},
		},
		{
			name: "SwitchCoinMarginedLeverage",
			path: "/swap-api/v1/swap_switch_lever_rate",
			call: func(h *Exchange) error {
				return h.SwitchCoinMarginedLeverage(t.Context(), contractCode, 5)
			},
		},
		{
			name: "GetSwapOrderLimitInfo",
			path: "/swap-api/v1/swap_order_limit",
			call: func(h *Exchange) error {
				_, err := h.GetSwapOrderLimitInfo(t.Context(), contractCode, "limit")
				return err
			},
		},
		{
			name: "GetSwapTradingFeeInfo",
			path: "/swap-api/v1/swap_fee",
			call: func(h *Exchange) error {
				_, err := h.GetSwapTradingFeeInfo(t.Context(), contractCode)
				return err
			},
		},
		{
			name: "GetSwapTransferLimitInfo",
			path: "/swap-api/v1/swap_transfer_limit",
			call: func(h *Exchange) error {
				_, err := h.GetSwapTransferLimitInfo(t.Context(), contractCode)
				return err
			},
		},
		{
			name: "GetSwapPositionLimitInfo",
			path: "/swap-api/v1/swap_position_limit",
			call: func(h *Exchange) error {
				_, err := h.GetSwapPositionLimitInfo(t.Context(), contractCode)
				return err
			},
		},
		{
			name: "AccountTransferData",
			path: "/swap-api/v1/swap_master_sub_transfer",
			call: func(h *Exchange) error {
				_, err := h.AccountTransferData(t.Context(), contractCode, "123", "master_to_sub", 1)
				return err
			},
		},
		{
			name: "AccountTransferRecords",
			path: "/swap-api/v1/swap_master_sub_transfer_record",
			call: func(h *Exchange) error {
				_, err := h.AccountTransferRecords(t.Context(), contractCode, "master_to_sub", 2, 1, 20)
				return err
			},
		},
		{
			name: "PlaceSwapOrders",
			path: "/swap-api/v1/swap_order",
			call: func(h *Exchange) error {
				_, err := h.PlaceSwapOrders(t.Context(), contractCode, "", "buy", "open", "limit", 1, 1, 1)
				return err
			},
		},
		{
			name: "PlaceSwapBatchOrders",
			path: "/swap-api/v1/swap_batchorder",
			call: func(h *Exchange) error {
				_, err := h.PlaceSwapBatchOrders(t.Context(), BatchOrderRequestType{Data: []batchOrderData{{
					ContractCode:   "ETH-USD",
					Price:          1,
					Volume:         1,
					Direction:      "buy",
					Offset:         "open",
					LeverageRate:   1,
					OrderPriceType: "limit",
				}}})
				return err
			},
		},
		{
			name: "CancelSwapOrder",
			path: "/swap-api/v1/swap_cancel",
			call: func(h *Exchange) error {
				_, err := h.CancelSwapOrder(t.Context(), "123", "", contractCode)
				return err
			},
		},
		{
			name: "CancelAllSwapOrders",
			path: "/swap-api/v1/swap_cancelall",
			call: func(h *Exchange) error {
				_, err := h.CancelAllSwapOrders(t.Context(), contractCode)
				return err
			},
		},
		{
			name: "PlaceLightningCloseOrder",
			path: "/swap-api/v1/swap_lightning_close_position",
			call: func(h *Exchange) error {
				_, err := h.PlaceLightningCloseOrder(t.Context(), contractCode, "buy", "lightning", 1, 123)
				return err
			},
		},
		{
			name: "GetSwapOrderDetails",
			path: "/swap-api/v1/swap_order_detail",
			call: func(h *Exchange) error {
				_, err := h.GetSwapOrderDetails(t.Context(), contractCode, "123", "10", "cancelledOrder", 1, 20)
				return err
			},
		},
		{
			name: "GetSwapOrderInfo",
			path: "/swap-api/v1/swap_order_info",
			call: func(h *Exchange) error {
				_, err := h.GetSwapOrderInfo(t.Context(), contractCode, "123", "")
				return err
			},
		},
		{
			name: "GetSwapOpenOrders",
			path: "/swap-api/v1/swap_openorders",
			call: func(h *Exchange) error {
				_, err := h.GetSwapOpenOrders(t.Context(), contractCode, 1, 20)
				return err
			},
		},
		{
			name: "GetSwapTradeHistory",
			path: "/swap-api/v3/swap_matchresults",
			call: func(h *Exchange) error {
				_, err := h.GetSwapTradeHistory(t.Context(), contractCode, "liquidateShort", 2, 1, 20)
				return err
			},
		},
		{
			name: "PlaceSwapTriggerOrder",
			path: "/swap-api/v1/swap_trigger_order",
			call: func(h *Exchange) error {
				_, err := h.PlaceSwapTriggerOrder(t.Context(), contractCode, "greaterOrEqual", "buy", "open", "optimal_5", 2, 1, 1, 1)
				return err
			},
		},
		{
			name:          "CancelSwapTriggerOrder",
			path:          "/swap-api/v1/swap_trigger_cancel",
			expectedField: "contract_code",
			expectedValue: "ETH-USD",
			call: func(h *Exchange) error {
				_, err := h.CancelSwapTriggerOrder(t.Context(), contractCode, "123")
				return err
			},
		},
		{
			name: "CancelAllSwapTriggerOrders",
			path: "/swap-api/v1/swap_trigger_cancelall",
			call: func(h *Exchange) error {
				_, err := h.CancelAllSwapTriggerOrders(t.Context(), contractCode)
				return err
			},
		},
		{
			name: "GetSwapTriggerOrderHistory",
			path: "/swap-api/v1/swap_trigger_hisorders",
			call: func(h *Exchange) error {
				_, err := h.GetSwapTriggerOrderHistory(t.Context(), contractCode, "open", "all", 2, 1, 20)
				return err
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method, "authenticated coin-margined method should match")
				assert.Equal(t, tc.path, r.URL.Path, "authenticated coin-margined path should match HTX documentation")
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"), "authenticated coin-margined content type should match")
				if tc.expectedField != "" {
					payload, err := io.ReadAll(r.Body)
					if !assert.NoError(t, err, "request body should be readable") {
						return
					}
					var body map[string]any
					if !assert.NoError(t, json.Unmarshal(payload, &body), "request body should decode") {
						return
					}
					assert.Equal(t, tc.expectedValue, body[tc.expectedField], "request body should contain the documented value")
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","code":200,"msg":"","data":null}`))
			}))
			t.Cleanup(server.Close)

			h := new(Exchange)
			require.NoError(t, testexch.Setup(h), "HTX setup must not error")
			h.API.AuthenticatedSupport = true
			h.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
			require.NoError(t, h.API.Endpoints.SetRunningURL(exchange.RestFutures.String(), server.URL), "coin-margined endpoint must be set")
			require.NoError(t, tc.call(h), "authenticated coin-margined endpoint must not error")
		})
	}
}
