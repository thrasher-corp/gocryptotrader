package htx

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestUnmarshalV3FuturesResponse(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name           string
		payload        string
		expectedArray  int
		expectedLegacy int64
		expectedErr    bool
	}{
		{name: "array", payload: `{"data":[{"id":1}],"ts":1604312615051}`, expectedArray: 1},
		{name: "legacy", payload: `{"data":{"items":[{"id":1}],"total_page":2},"ts":1604312615051}`, expectedLegacy: 2},
		{name: "empty string", payload: `{"data":"","ts":1604312615051}`},
		{name: "null", payload: `{"data":null,"ts":1604312615051}`},
		{name: "malformed document", payload: `{`, expectedErr: true},
		{name: "malformed data", payload: `{"data":1}`, expectedErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var timestamp types.Time
			var array []struct {
				ID int64 `json:"id"`
			}
			var legacy struct {
				Items []struct {
					ID int64 `json:"id"`
				} `json:"items"`
				TotalPage int64 `json:"total_page"`
			}
			err := unmarshalV3FuturesResponse([]byte(tc.payload), &timestamp, &array, &legacy)
			if tc.expectedErr {
				require.Error(t, err, "unmarshalV3FuturesResponse must return the expected error")
				return
			}
			require.NoError(t, err, "unmarshalV3FuturesResponse must not error")
			assert.Len(t, array, tc.expectedArray, "array response should decode")
			assert.Equal(t, tc.expectedLegacy, legacy.TotalPage, "legacy pagination should decode")
			assert.Equal(t, time.UnixMilli(1604312615051), timestamp.Time(), "timestamp should decode")
		})
	}
}

func TestFFinancialRecordsUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var arrayResp FFinancialRecords
	err := json.Unmarshal([]byte(`{"code":200,"msg":"","data":[{"query_id":12,"id":34,"symbol":"BTC","contract_code":"BTC-USD","type":3,"amount":1.25,"ts":1604312615051}],"ts":1604312615051}`), &arrayResp)
	require.NoError(t, err, "FFinancialRecords unmarshal must support v3 array data")
	require.Len(t, arrayResp.Data.FinancialRecord, 1, "financial records must decode from v3 array data")
	assert.Equal(t, int64(12), arrayResp.Data.FinancialRecord[0].QueryID, "query id should decode")

	var emptyResp FFinancialRecords
	err = json.Unmarshal([]byte(`{"code":200,"msg":"","data":"","ts":1604312615051}`), &emptyResp)
	require.NoError(t, err, "FFinancialRecords unmarshal must support empty string data")
	assert.Empty(t, emptyResp.Data.FinancialRecord, "financial records should be empty")

	var legacyResp FFinancialRecords
	err = json.Unmarshal([]byte(`{"data":{"financial_record":[{"id":34,"symbol":"BTC","type":3,"amount":1.25,"ts":1604312615051}],"total_page":2,"current_page":1,"total_size":3},"ts":1604312615051}`), &legacyResp)
	require.NoError(t, err, "FFinancialRecords unmarshal must support legacy object data")
	assert.Equal(t, int64(2), legacyResp.Data.TotalPage, "legacy total page should decode")

	err = json.Unmarshal([]byte(`{`), &legacyResp)
	require.Error(t, err, "FFinancialRecords unmarshal must return malformed JSON errors")
	err = json.Unmarshal([]byte(`{"data":1}`), &legacyResp)
	require.Error(t, err, "FFinancialRecords unmarshal must return malformed data errors")
}

func TestFOrderHistoryDataUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var arrayResp FOrderHistoryData
	err := json.Unmarshal([]byte(`{"code":200,"msg":"","data":[{"query_id":12,"order_id":34,"order_id_str":"34","symbol":"BTC","contract_code":"BTC-USD","contract_type":"quarter","lever_rate":20,"direction":"buy","offset":"open","volume":1,"price":10,"create_date":1604312615051,"order_source":"api","order_price_type":"limit","margin_frozen":0,"profit":0,"trade_volume":0,"trade_turnover":0,"fee":0,"trade_avg_price":0,"status":6,"order_type":1,"fee_asset":"BTC","liquidation_type":"0"}],"ts":1604312615051}`), &arrayResp)
	require.NoError(t, err, "FOrderHistoryData unmarshal must support v3 array data")
	require.Len(t, arrayResp.Data.Orders, 1, "orders must decode from v3 array data")
	assert.Equal(t, int64(12), arrayResp.Data.Orders[0].QueryID, "query id should decode")

	var emptyResp FOrderHistoryData
	err = json.Unmarshal([]byte(`{"code":200,"msg":"","data":"","ts":1604312615051}`), &emptyResp)
	require.NoError(t, err, "FOrderHistoryData unmarshal must support empty string data")
	assert.Empty(t, emptyResp.Data.Orders, "orders should be empty")

	var legacyResp FOrderHistoryData
	err = json.Unmarshal([]byte(`{"data":{"orders":[{"query_id":12,"order_id":34,"symbol":"BTC","liquidation_type":0}],"total_page":2},"ts":1604312615051}`), &legacyResp)
	require.NoError(t, err, "FOrderHistoryData unmarshal must support legacy object data")
	assert.Equal(t, int64(2), legacyResp.Data.TotalPage, "legacy total page should decode")

	err = json.Unmarshal([]byte(`{`), &legacyResp)
	require.Error(t, err, "FOrderHistoryData unmarshal must return malformed JSON errors")
	err = json.Unmarshal([]byte(`{"data":1}`), &legacyResp)
	require.Error(t, err, "FOrderHistoryData unmarshal must return malformed data errors")
}

func TestFTradeHistoryDataUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var arrayResp FTradeHistoryData
	err := json.Unmarshal([]byte(`{"code":200,"msg":"","data":[{"query_id":12,"id":"match","match_id":34,"order_id":56,"order_id_str":"56","symbol":"BTC","contract_code":"BTC-USD","contract_type":"quarter","direction":"buy","offset":"open","trade_volume":1,"trade_price":10,"trade_turnover":10,"trade_fee":0.1,"offset_profitloss":0,"create_date":1604312615051,"role":"Maker","order_source":"api","fee_asset":"BTC"}],"ts":1604312615051}`), &arrayResp)
	require.NoError(t, err, "FTradeHistoryData unmarshal must support v3 array data")
	require.Len(t, arrayResp.Data.Trades, 1, "trades must decode from v3 array data")
	assert.Equal(t, int64(12), arrayResp.Data.Trades[0].QueryID, "query id should decode")

	var emptyResp FTradeHistoryData
	err = json.Unmarshal([]byte(`{"code":200,"msg":"","data":"","ts":1604312615051}`), &emptyResp)
	require.NoError(t, err, "FTradeHistoryData unmarshal must support empty string data")
	assert.Empty(t, emptyResp.Data.Trades, "trades should be empty")

	var legacyResp FTradeHistoryData
	err = json.Unmarshal([]byte(`{"data":{"trades":[{"query_id":12,"id":"match"}],"total_page":2},"ts":1604312615051}`), &legacyResp)
	require.NoError(t, err, "FTradeHistoryData unmarshal must support legacy object data")
	assert.Equal(t, int64(2), legacyResp.Data.TotalPage, "legacy total page should decode")

	err = json.Unmarshal([]byte(`{`), &legacyResp)
	require.Error(t, err, "FTradeHistoryData unmarshal must return malformed JSON errors")
	err = json.Unmarshal([]byte(`{"data":1}`), &legacyResp)
	require.Error(t, err, "FTradeHistoryData unmarshal must return malformed data errors")
}
