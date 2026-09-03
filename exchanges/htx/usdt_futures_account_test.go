package htx

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetV5AssetMode(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestUSDTMargined, http.MethodGet, "/v5/account/asset_mode", `{"code":200,"data":{"asset_mode":1}}`, nil)
	resp, err := h.GetV5AssetMode(t.Context())
	require.NoError(t, err, "GetV5AssetMode must not error")
	require.NotNil(t, resp, "asset mode response must not be nil")
	assert.Equal(t, uint64(1), resp.Data.AssetMode, "asset mode should decode")
}

func TestSetV5AssetMode(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestUSDTMargined, http.MethodPost, "/v5/account/asset_mode", `{"code":200,"data":{"asset_mode":2}}`, nil)
	resp, err := h.SetV5AssetMode(t.Context(), 2)
	require.NoError(t, err, "SetV5AssetMode must not error")
	require.NotNil(t, resp, "asset mode response must not be nil")
	assert.Equal(t, uint64(2), resp.Data.AssetMode, "asset mode should decode")
}

func TestGetV5AccountBills(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestUSDTMargined, http.MethodGet, "/v5/account/bills", `{"code":200,"data":[{"id":"1","amount":"2"}]}`, func(r *http.Request) {
		assert.Equal(t, "BTC-USDT", r.URL.Query().Get("contract_code"), "contract code should be sent")
		assert.Equal(t, "10", r.URL.Query().Get("limit"), "limit should be sent")
	})
	_, err := h.GetV5AccountBills(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetV5AccountBills must reject nil request")
	resp, err := h.GetV5AccountBills(t.Context(), &V5AccountBillsRequest{
		ContractCode: "BTC-USDT",
		MarginMode:   "cross",
		StartTime:    time.UnixMilli(1),
		Type:         "trade",
		EndTime:      time.UnixMilli(2),
		From:         1,
		Limit:        10,
		Direction:    "next",
	})
	require.NoError(t, err, "GetV5AccountBills must not error")
	require.Len(t, resp.Data, 1, "one bill must decode")
	assert.Equal(t, types.Number(2), resp.Data[0].Amount, "amount should decode")
}

func TestGetV5FeeDeductionCurrency(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestUSDTMargined, http.MethodGet, "/v5/account/fee_deduction_currency", `{"code":200,"data":{"fee_option":1,"deduction_currency":"HTX"}}`, nil)
	resp, err := h.GetV5FeeDeductionCurrency(t.Context())
	require.NoError(t, err, "GetV5FeeDeductionCurrency must not error")
	require.NotNil(t, resp, "fee deduction response must not be nil")
	assert.Equal(t, "HTX", resp.Data.DeductionCurrency, "deduction currency should decode")
}

func TestSetV5FeeDeductionCurrency(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestUSDTMargined, http.MethodPost, "/v5/account/fee_deduction_currency", `{"code":200,"data":{"fee_option":1,"deduction_currency":"HTX"}}`, nil)
	_, err := h.SetV5FeeDeductionCurrency(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SetV5FeeDeductionCurrency must reject nil request")
	resp, err := h.SetV5FeeDeductionCurrency(t.Context(), &V5SetFeeDeductionCurrencyRequest{FeeOption: 1, DeductionCurrency: "HTX"})
	require.NoError(t, err, "SetV5FeeDeductionCurrency must not error")
	require.NotNil(t, resp, "fee deduction response must not be nil")
	assert.Equal(t, "HTX", resp.Data.DeductionCurrency, "deduction currency should decode")
}

func TestV5UniversalTransfer(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestSpot, http.MethodPost, "/v5/account/universal_transfer", `{"code":200,"data":{"transfer_id":123}}`, nil)
	_, err := h.V5UniversalTransfer(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "V5UniversalTransfer must reject nil request")
	resp, err := h.V5UniversalTransfer(t.Context(), &V5UniversalTransferRequest{
		Amount:          10,
		Currency:        "USDT",
		FromAccountType: "spot",
		ToAccountType:   "linear-swap",
	})
	require.NoError(t, err, "V5UniversalTransfer must not error")
	require.NotNil(t, resp, "transfer response must not be nil")
	assert.Equal(t, uint64(123), resp.Data.TransferID, "transfer ID should decode")
}

func TestGetV5UniversalTransferRecords(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestSpot, http.MethodGet, "/v5/account/universal_transfer_records", `{"code":200,"data":[{"id":1,"transfer_id":"123","amount":"10"}]}`, func(r *http.Request) {
		assert.Equal(t, "123", r.URL.Query().Get("transfer_id"), "transfer ID should be sent")
		assert.Equal(t, "USDT", r.URL.Query().Get("currency"), "currency should be sent")
	})
	_, err := h.GetV5UniversalTransferRecords(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetV5UniversalTransferRecords must reject nil request")
	resp, err := h.GetV5UniversalTransferRecords(t.Context(), &V5UniversalTransferRecordsRequest{
		TransferID: 123,
		Currency:   "USDT",
		Status:     "success",
		StartTime:  time.UnixMilli(1),
		EndTime:    time.UnixMilli(2),
		From:       1,
		Limit:      10,
		Direction:  "next",
	})
	require.NoError(t, err, "GetV5UniversalTransferRecords must not error")
	require.Len(t, resp.Data, 1, "one transfer record must decode")
	assert.Equal(t, types.Number(123), resp.Data[0].TransferID, "transfer ID should decode")
}

func TestGetV5AccountBalance(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestUSDTMargined, http.MethodGet, "/v5/account/balance", `{"code":200,"data":{"state":"working"}}`, nil)
	resp, err := h.GetV5AccountBalance(t.Context())
	require.NoError(t, err, "GetV5AccountBalance must not error")
	require.NotNil(t, resp, "decoded balance must be returned")
	assert.Equal(t, "working", resp.Data.State, "account state should decode")
}
