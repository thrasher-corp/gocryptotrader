package htx

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

func TestSwapFinancialRecordsResponseUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var arrayResp SwapFinancialRecordsResponse
	err := json.Unmarshal([]byte(`{"code":200,"msg":"","data":[{"query_id":12,"id":34,"symbol":"ETH","contract_code":"ETH-USD","type":3,"amount":1.25,"ts":1604312615051}],"ts":1604312615051}`), &arrayResp)
	require.NoError(t, err, "SwapFinancialRecordsResponse unmarshal must support v3 array data")
	require.Len(t, arrayResp.Data.FinancialRecord, 1, "financial records must decode from v3 array data")
	assert.Equal(t, uint64(12), arrayResp.Data.FinancialRecord[0].QueryID, "query id should decode")
	assert.Equal(t, int64(1604312615051), arrayResp.Timestamp.Time().UnixMilli(), "response timestamp should decode")

	var emptyResp SwapFinancialRecordsResponse
	err = json.Unmarshal([]byte(`{"code":200,"msg":"","data":"","ts":1604312615051}`), &emptyResp)
	require.NoError(t, err, "SwapFinancialRecordsResponse unmarshal must support empty string data")
	assert.Empty(t, emptyResp.Data.FinancialRecord, "financial records should be empty")

	var legacyResp SwapFinancialRecordsResponse
	err = json.Unmarshal([]byte(`{"data":{"financial_record":[{"query_id":12,"id":34}],"total_page":2},"ts":1604312615051}`), &legacyResp)
	require.NoError(t, err, "SwapFinancialRecordsResponse unmarshal must support legacy object data")
	assert.Equal(t, uint64(2), legacyResp.Data.TotalPage, "legacy total page should decode")

	err = json.Unmarshal([]byte(`{`), &legacyResp)
	require.Error(t, err, "SwapFinancialRecordsResponse unmarshal must return malformed JSON errors")
	err = json.Unmarshal([]byte(`{"data":1}`), &legacyResp)
	require.Error(t, err, "SwapFinancialRecordsResponse unmarshal must return malformed data errors")
}

func TestSwapOrderHistoryResponseUnmarshalJSON(t *testing.T) {
	t.Parallel()
	t.Run("numeric order price", func(t *testing.T) {
		t.Parallel()
		var response SwapOrderHistoryResponse
		require.NoError(t, json.Unmarshal([]byte(`{"data":[{"order_price_type":6}]}`), &response), "numeric V3 order price type must decode")
		require.Len(t, response.Data.Orders, 1, "numeric V3 history must retain its order")
		assert.Equal(t, LegacyOrderPriceType("post_only"), response.Data.Orders[0].OrderPriceType, "numeric type should retain post-only semantics")
	})
	var arrayResp SwapOrderHistoryResponse
	err := json.Unmarshal([]byte(`{"code":200,"msg":"","data":[{"query_id":12,"order_id":34,"order_id_str":"34","symbol":"ETH","contract_code":"ETH-USD","lever_rate":20,"direction":"buy","offset":"open","volume":1,"price":10,"create_date":1604312615051,"update_time":1604312615052,"order_source":"api","order_price_type":"limit","margin_frozen":0,"profit":0,"trade_volume":0,"trade_turnover":0,"fee":0,"trade_avg_price":0,"status":6,"order_type":1,"fee_asset":"ETH","liquidation_type":"0"}],"ts":1604312615051}`), &arrayResp)
	require.NoError(t, err, "SwapOrderHistoryResponse unmarshal must support v3 array data")
	require.Len(t, arrayResp.Data.Orders, 1, "orders must decode from v3 array data")
	assert.Equal(t, uint64(12), arrayResp.Data.Orders[0].QueryID, "query id should decode")
	assert.Equal(t, time.UnixMilli(1604312615051), arrayResp.Data.Orders[0].CreateDate.Time(), "order creation time should retain milliseconds")
	assert.Equal(t, time.UnixMilli(1604312615052), arrayResp.Data.Orders[0].UpdateTime.Time(), "order update time should retain milliseconds")

	var emptyResp SwapOrderHistoryResponse
	err = json.Unmarshal([]byte(`{"code":200,"msg":"","data":"","ts":1604312615051}`), &emptyResp)
	require.NoError(t, err, "SwapOrderHistoryResponse unmarshal must support empty string data")
	assert.Empty(t, emptyResp.Data.Orders, "orders should be empty")

	var legacyResp SwapOrderHistoryResponse
	err = json.Unmarshal([]byte(`{"data":{"orders":[{"query_id":12,"order_id":34}],"total_page":2},"ts":1604312615051}`), &legacyResp)
	require.NoError(t, err, "SwapOrderHistoryResponse unmarshal must support legacy object data")
	assert.Equal(t, uint64(2), legacyResp.Data.TotalPage, "legacy total page should decode")

	err = json.Unmarshal([]byte(`{`), &legacyResp)
	require.Error(t, err, "SwapOrderHistoryResponse unmarshal must return malformed JSON errors")
	err = json.Unmarshal([]byte(`{"data":1}`), &legacyResp)
	require.Error(t, err, "SwapOrderHistoryResponse unmarshal must return malformed data errors")
}

func TestSwapTradeHistoryResponseUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var arrayResp SwapTradeHistoryResponse
	err := json.Unmarshal([]byte(`{"code":200,"msg":"","data":[{"query_id":12,"id":"match","match_id":34,"order_id":56,"order_id_str":"56","symbol":"ETH","contract_code":"ETH-USD","direction":"buy","offset":"open","trade_volume":1,"trade_price":10,"trade_turnover":10,"trade_fee":0.1,"offset_profitloss":0,"create_date":"1604312615051","role":"Maker","order_source":"api","fee_asset":"ETH"}],"ts":1604312615051}`), &arrayResp)
	require.NoError(t, err, "SwapTradeHistoryResponse unmarshal must support v3 array data")
	require.Len(t, arrayResp.Data.Trades, 1, "trades must decode from v3 array data")
	assert.Equal(t, uint64(12), arrayResp.Data.Trades[0].QueryID, "query id should decode")
	assert.Equal(t, time.UnixMilli(1604312615051), arrayResp.Data.Trades[0].CreateDate.Time(), "trade creation time should decode a quoted timestamp")

	var emptyResp SwapTradeHistoryResponse
	err = json.Unmarshal([]byte(`{"code":200,"msg":"","data":"","ts":1604312615051}`), &emptyResp)
	require.NoError(t, err, "SwapTradeHistoryResponse unmarshal must support empty string data")
	assert.Empty(t, emptyResp.Data.Trades, "trades should be empty")

	var legacyResp SwapTradeHistoryResponse
	err = json.Unmarshal([]byte(`{"data":{"trades":[{"query_id":12,"id":"match"}],"total_page":2},"ts":1604312615051}`), &legacyResp)
	require.NoError(t, err, "SwapTradeHistoryResponse unmarshal must support legacy object data")
	assert.Equal(t, uint64(2), legacyResp.Data.TotalPage, "legacy total page should decode")

	err = json.Unmarshal([]byte(`{`), &legacyResp)
	require.Error(t, err, "SwapTradeHistoryResponse unmarshal must return malformed JSON errors")
	err = json.Unmarshal([]byte(`{"data":1}`), &legacyResp)
	require.Error(t, err, "SwapTradeHistoryResponse unmarshal must return malformed data errors")
}

func TestTriggerOrderUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, triggered, inserted   string
		wantTriggered, wantInserted time.Time
	}{
		{name: "executed", triggered: `1603874307537`, inserted: `1603874307538`, wantTriggered: time.UnixMilli(1603874307537), wantInserted: time.UnixMilli(1603874307538)},
		{name: "not executed", triggered: `null`, inserted: `0`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			raw := `{"status":"ok","ts":1603874865911,"data":{"orders":[{"order_id":9007199254740993,"created_at":1603874307536,"canceled_at":1603874307539,"order_id_str":"9007199254740993","triggered_at":` + tc.triggered + `,"order_insert_at":` + tc.inserted + `}],"total_page":1,"current_page":1,"total_size":1}}`
			var response SwapTriggerOrderHistoryResponse
			require.NoError(t, json.Unmarshal([]byte(raw), &response), "trigger history envelope must decode")
			require.Len(t, response.Data.Orders, 1, "history must retain its trigger order")
			entry := response.Data.Orders[0]
			assert.Equal(t, uint64(9007199254740993), entry.OrderID, "order ID should retain all digits")
			assert.Equal(t, "9007199254740993", entry.OrderIDString, "documented order ID string should decode")
			assert.Equal(t, time.UnixMilli(1603874307539), entry.CancelledAt.Time(), "documented cancellation key should decode")
			assert.Equal(t, time.UnixMilli(1603874307536), entry.CreatedAt.Time(), "creation time should retain milliseconds")
			assert.Equal(t, tc.wantTriggered, entry.TriggeredAt.Time(), "trigger time should reflect execution state")
			assert.Equal(t, tc.wantInserted, entry.OrderInsertAt.Time(), "order insertion time should reflect execution state")
		})
	}
}
