package htx

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestUnmarshalV3FuturesResponse(t *testing.T) {
	t.Parallel()
	type testCase struct {
		name           string
		payload        string
		expectedArray  int
		expectedLegacy int64
		expectedErr    bool
	}
	for _, tc := range []testCase{
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
			type record struct {
				ID int64 `json:"id"`
			}
			type page struct {
				Items     []record `json:"items"`
				TotalPage int64    `json:"total_page"`
			}
			var array []record
			var legacy page
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

func TestFuturesV3RecordsUnmarshalJSON(t *testing.T) {
	t.Parallel()
	type testCase struct {
		name, payload string
		wantID        uint64
		wantPage      uint64
		wantErr       bool
	}
	for _, tc := range []testCase{
		{name: "array", payload: `[{"query_id":18446744073709551615}]`, wantID: 18446744073709551615},
		{name: "page", payload: `{"financial_record":[{"query_id":12}],"total_page":2}`, wantID: 12, wantPage: 2},
		{name: "empty string", payload: `""`},
		{name: "null", payload: `null`},
		{name: "missing result"},
		{name: "empty array", payload: `[]`},
		{name: "empty page", payload: `{}`},
		{name: "malformed array", payload: `[{`, wantErr: true},
		{name: "wrong shape", payload: `true`, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var records []FinancialRecord
			var page SwapFinancialRecordPage
			response := futuresV3Records[FinancialRecord, SwapFinancialRecordPage]{Array: &records, Page: &page}
			err := response.UnmarshalJSON([]byte(tc.payload))
			if tc.wantErr {
				require.Error(t, err, "invalid V3 record shapes must fail")
				return
			}
			require.NoError(t, err, "supported V3 record shapes must decode")
			assert.Equal(t, tc.wantPage, page.TotalPage, "legacy pagination should remain available")
			if tc.wantPage != 0 {
				records = page.Records
			}
			if tc.wantID == 0 {
				assert.Empty(t, records, "empty responses should contain no records")
				return
			}
			require.Len(t, records, 1, "response must contain one record")
			assert.Equal(t, tc.wantID, records[0].QueryID, "record identifier should retain its full range")
		})
	}
}

func TestFuturesFinancialRecordsResponseUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var arrayResp FuturesFinancialRecordsResponse
	err := json.Unmarshal([]byte(`{"code":200,"msg":"","data":[{"query_id":12,"id":34,"symbol":"BTC","contract_code":"BTC-USD","type":3,"amount":1.25,"ts":1604312615051}],"ts":1604312615051}`), &arrayResp)
	require.NoError(t, err, "FuturesFinancialRecordsResponse unmarshal must support v3 array data")
	require.Len(t, arrayResp.FinancialRecords.Records, 1, "financial records must decode from v3 array data")
	assert.Equal(t, uint64(12), arrayResp.FinancialRecords.Records[0].QueryID, "query id should decode")

	var emptyResp FuturesFinancialRecordsResponse
	err = json.Unmarshal([]byte(`{"code":200,"msg":"","data":"","ts":1604312615051}`), &emptyResp)
	require.NoError(t, err, "FuturesFinancialRecordsResponse unmarshal must support empty string data")
	assert.Empty(t, emptyResp.FinancialRecords.Records, "financial records should be empty")

	var legacyResp FuturesFinancialRecordsResponse
	err = json.Unmarshal([]byte(`{"data":{"financial_record":[{"id":34,"symbol":"BTC","type":3,"amount":1.25,"ts":1604312615051}],"total_page":2,"current_page":1,"total_size":3},"ts":1604312615051}`), &legacyResp)
	require.NoError(t, err, "FuturesFinancialRecordsResponse unmarshal must support legacy object data")
	assert.Equal(t, uint64(2), legacyResp.FinancialRecords.TotalPage, "legacy total page should decode")

	err = json.Unmarshal([]byte(`{`), &legacyResp)
	require.Error(t, err, "FuturesFinancialRecordsResponse unmarshal must return malformed JSON errors")
	err = json.Unmarshal([]byte(`{"data":1}`), &legacyResp)
	require.Error(t, err, "FuturesFinancialRecordsResponse unmarshal must return malformed data errors")
}

func TestFuturesOrderHistoryResponseUnmarshalJSON(t *testing.T) {
	t.Parallel()
	t.Run("numeric order price", func(t *testing.T) {
		t.Parallel()
		var response FuturesOrderHistoryResponse
		require.NoError(t, json.Unmarshal([]byte(`{"data":[{"order_price_type":6}]}`), &response), "numeric V3 order price type must decode")
		require.Len(t, response.OrderHistory.Orders, 1, "numeric V3 history must retain its order")
		assert.Equal(t, LegacyOrderPriceType("post_only"), response.OrderHistory.Orders[0].OrderPriceType, "numeric type should retain post-only semantics")
	})
	var arrayResp FuturesOrderHistoryResponse
	err := json.Unmarshal([]byte(`{"code":200,"msg":"","data":[{"query_id":12,"order_id":34,"order_id_str":"34","symbol":"BTC","contract_code":"BTC-USD","contract_type":"quarter","lever_rate":20,"direction":"buy","offset":"open","volume":1,"price":10,"create_date":1604312615051,"order_source":"api","order_price_type":"limit","margin_frozen":0,"profit":0,"trade_volume":0,"trade_turnover":0,"fee":0,"trade_avg_price":0,"status":6,"order_type":1,"fee_asset":"BTC","liquidation_type":"0"}],"ts":1604312615051}`), &arrayResp)
	require.NoError(t, err, "FuturesOrderHistoryResponse unmarshal must support v3 array data")
	require.Len(t, arrayResp.OrderHistory.Orders, 1, "orders must decode from v3 array data")
	assert.Equal(t, uint64(12), arrayResp.OrderHistory.Orders[0].QueryID, "query id should decode")

	var emptyResp FuturesOrderHistoryResponse
	err = json.Unmarshal([]byte(`{"code":200,"msg":"","data":"","ts":1604312615051}`), &emptyResp)
	require.NoError(t, err, "FuturesOrderHistoryResponse unmarshal must support empty string data")
	assert.Empty(t, emptyResp.OrderHistory.Orders, "orders should be empty")

	var legacyResp FuturesOrderHistoryResponse
	err = json.Unmarshal([]byte(`{"data":{"orders":[{"query_id":12,"order_id":34,"symbol":"BTC","liquidation_type":0}],"total_page":2},"ts":1604312615051}`), &legacyResp)
	require.NoError(t, err, "FuturesOrderHistoryResponse unmarshal must support legacy object data")
	assert.Equal(t, uint64(2), legacyResp.OrderHistory.TotalPage, "legacy total page should decode")

	err = json.Unmarshal([]byte(`{`), &legacyResp)
	require.Error(t, err, "FuturesOrderHistoryResponse unmarshal must return malformed JSON errors")
	err = json.Unmarshal([]byte(`{"data":1}`), &legacyResp)
	require.Error(t, err, "FuturesOrderHistoryResponse unmarshal must return malformed data errors")
}

func TestFuturesTradeHistoryResponseUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var arrayResp FuturesTradeHistoryResponse
	err := json.Unmarshal([]byte(`{"code":200,"msg":"","data":[{"query_id":12,"id":"match","match_id":34,"order_id":56,"order_id_str":"56","symbol":"BTC","contract_code":"BTC-USD","contract_type":"quarter","direction":"buy","offset":"open","trade_volume":1,"trade_price":10,"trade_turnover":10,"trade_fee":0.1,"offset_profitloss":0,"create_date":1604312615051,"role":"Maker","order_source":"api","fee_asset":"BTC"}],"ts":1604312615051}`), &arrayResp)
	require.NoError(t, err, "FuturesTradeHistoryResponse unmarshal must support v3 array data")
	require.Len(t, arrayResp.TradeHistory.Trades, 1, "trades must decode from v3 array data")
	assert.Equal(t, uint64(12), arrayResp.TradeHistory.Trades[0].QueryID, "query id should decode")

	var emptyResp FuturesTradeHistoryResponse
	err = json.Unmarshal([]byte(`{"code":200,"msg":"","data":"","ts":1604312615051}`), &emptyResp)
	require.NoError(t, err, "FuturesTradeHistoryResponse unmarshal must support empty string data")
	assert.Empty(t, emptyResp.TradeHistory.Trades, "trades should be empty")

	var legacyResp FuturesTradeHistoryResponse
	err = json.Unmarshal([]byte(`{"data":{"trades":[{"query_id":12,"id":"match"}],"total_page":2},"ts":1604312615051}`), &legacyResp)
	require.NoError(t, err, "FuturesTradeHistoryResponse unmarshal must support legacy object data")
	assert.Equal(t, uint64(2), legacyResp.TradeHistory.TotalPage, "legacy total page should decode")

	err = json.Unmarshal([]byte(`{`), &legacyResp)
	require.Error(t, err, "FuturesTradeHistoryResponse unmarshal must return malformed JSON errors")
	err = json.Unmarshal([]byte(`{"data":1}`), &legacyResp)
	require.Error(t, err, "FuturesTradeHistoryResponse unmarshal must return malformed data errors")
}

func TestFuturesMarketOverviewResponseUnmarshal(t *testing.T) {
	t.Parallel()
	var o FuturesMarketOverviewResponse
	err := json.Unmarshal([]byte(`{"ch":"market.BTC_CQ.detail.merged","tick":{"amount":"7.63","ask":[70020.48,2],"bid":[68941.77,3],"close":"69892.45","count":1768,"high":"70288.9","id":1787211582,"low":"67657.1","open":"68860.25","ts":1787211582961,"vol":"5188"},"ts":1787211582961}`), &o)
	require.NoError(t, err, "Unmarshal must not error")
	assert.Equal(t, []float64{70020.48, 2}, o.Tick.Ask, "Ask should decode")
	assert.Equal(t, []float64{68941.77, 3}, o.Tick.Bid, "Bid should decode")
	assert.Equal(t, uint64(1787211582), o.Tick.ID, "ID should decode")
	assert.Equal(t, 69892.45, o.Tick.Close.Float64(), "Close should decode")
}

func TestFuturesContractInfoUnmarshalJSON(t *testing.T) {
	t.Parallel()
	type testCase struct {
		name          string
		date          string
		expected      time.Time
		expectedError bool
	}
	for _, tc := range []testCase{
		{name: "delivery date", date: "20210813", expected: time.Date(2021, time.August, 13, 0, 0, 0, 0, time.UTC)},
		{name: "empty date"},
		{name: "invalid date", date: "20211313", expectedError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			payload := []byte(`{"contract_code":"BTC210813","delivery_date":"` + tc.date + `","contract_status":1}`)
			var response FuturesContractInfo
			err := json.Unmarshal(payload, &response)
			if tc.expectedError {
				require.ErrorIs(t, err, types.ErrInvalidTimestampFormat, "invalid delivery dates must fail decoding")
				return
			}
			require.NoError(t, err, "contract information must decode")
			assert.Equal(t, tc.expected, response.DeliveryDate.Time(), "delivery date should use the documented calendar format")
			assert.Equal(t, uint64(1), response.ContractStatus, "contract status should decode")
		})
	}
}

func TestFuturesOrderDetailsUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var response FuturesOrderDetails
	require.NoError(t, json.Unmarshal([]byte(`{"order_id":9007199254740993,"client_order_id":9007199254740994,"created_at":1604368087894,"canceled_at":0,"lever_rate":20,"trade_turnover":10.25,"adjust_value":0.125,"fee_asset":"BTC","trades":[{"trade_id":9007199254740995,"created_at":1604368087895}]}`), &response), "documented order details must decode")
	assert.Equal(t, uint64(9007199254740993), response.OrderID, "order ID should retain precision")
	assert.Equal(t, uint64(9007199254740994), response.ClientOrderID, "client order ID should retain precision")
	assert.Equal(t, time.UnixMilli(1604368087894), response.CreatedAt.Time(), "creation time should retain milliseconds")
	assert.True(t, response.CanceledAt.Time().IsZero(), "zero cancellation time should remain unset")
	assert.Equal(t, uint64(20), response.LeverageRate, "order leverage should decode as an integer")
	assert.InDelta(t, 10.25, response.TradeTurnover, 0, "turnover should retain its decimal part")
	assert.InDelta(t, 0.125, response.AdjustValue, 0, "adjustment should retain its decimal part")
	assert.Equal(t, currency.BTC, response.FeeAsset, "fee currency should decode")
	require.Len(t, response.Trades, 1, "order fill must decode")
	assert.Equal(t, uint64(9007199254740995), response.Trades[0].TradeID, "trade ID should retain precision")
	assert.Equal(t, time.UnixMilli(1604368087895), response.Trades[0].CreatedAt.Time(), "fill creation time should retain milliseconds")
}

func TestFuturesSettlementRecordUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var response FuturesSettlementRecord
	require.NoError(t, json.Unmarshal([]byte(`{"margin_balance":1.25,"fee_asset":"BTC"}`), &response), "documented settlement record must decode")
	assert.InDelta(t, 1.25, response.MarginBalance, 0, "margin balance should retain its decimal part")
	assert.Equal(t, currency.BTC, response.FeeAsset, "settlement fee currency should decode")
}

func TestFuturesTriggerOrderHistoryEntryUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var response FuturesTriggerOrderHistoryEntry
	require.NoError(t, json.Unmarshal([]byte(`{"order_id":9007199254740993,"trigger_price":0.0895,"created_at":1604390110565,"triggered_at":0,"order_insert_at":1604390110567,"canceled_at":1604390110566,"fail_code":1001}`), &response), "documented trigger order must decode")
	assert.Equal(t, uint64(9007199254740993), response.OrderID, "trigger order ID should retain precision")
	assert.InDelta(t, 0.0895, response.TriggerPrice, 0, "trigger price should retain its decimal part")
	assert.Equal(t, time.UnixMilli(1604390110565), response.CreatedAt.Time(), "trigger creation time should retain milliseconds")
	assert.Equal(t, time.UnixMilli(1604390110566), response.CancelledAt.Time(), "trigger cancellation time should retain milliseconds")
	assert.True(t, response.TriggeredAt.Time().IsZero(), "zero trigger time should remain unset")
	assert.Equal(t, time.UnixMilli(1604390110567), response.OrderInsertAt.Time(), "order insertion time should retain milliseconds")
	assert.Equal(t, uint64(1001), response.FailCode, "non-negative failure codes should decode")
}

func TestFuturesErrorCodes(t *testing.T) {
	t.Parallel()
	type testCase struct {
		name, key string
		response  any
	}
	for _, tc := range []testCase{
		{name: "FuturesCancelOrderError", key: "err_code", response: new(FuturesCancelOrderError)},
		{name: "FuturesTriggerOrderHistoryEntry", key: "fail_code", response: new(FuturesTriggerOrderHistoryEntry)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.NoError(t, json.Unmarshal([]byte(`{"`+tc.key+`":1001}`), tc.response), "documented positive error code must decode")
			require.Error(t, json.Unmarshal([]byte(`{"`+tc.key+`":-1}`), tc.response), "negative error code must not decode into an unsigned field")
		})
	}
}
