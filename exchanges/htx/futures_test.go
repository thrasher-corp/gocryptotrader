package htx

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func TestFuturesHistoryEndpointPaths(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "/api/v3/contract_financial_record", fFinancialRecords, "delivery futures financial records endpoint should match HTX docs")
	assert.Equal(t, "/api/v3/contract_hisorders", fOrderHistory, "delivery futures order history endpoint should match HTX docs")
	assert.Equal(t, "/api/v3/contract_matchresults", fMatchResult, "delivery futures trade history endpoint should match HTX docs")
	assert.Equal(t, "/swap-api/v3/swap_financial_record", htxSwapFinancialRecords, "coin-margined financial records endpoint should match HTX docs")
	assert.Equal(t, "/swap-api/v3/swap_hisorders", htxSwapOrderHistory, "coin-margined order history endpoint should match HTX docs")
	assert.Equal(t, "/swap-api/v3/swap_matchresults", htxSwapTradeHistory, "coin-margined trade history endpoint should match HTX docs")
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

func TestAddV3HistoryTimeRange(t *testing.T) {
	t.Parallel()
	req := make(map[string]any)
	addV3HistoryTimeRange(req, 10)
	startTime, ok := req["start_time"].(int64)
	require.True(t, ok, "start time must be set")
	endTime, ok := req["end_time"].(int64)
	require.True(t, ok, "end time must be set")
	assert.Greater(t, endTime, startTime, "end time should be after start time")
	assert.InDelta(t, int64(48*time.Hour/time.Millisecond), endTime-startTime, float64(time.Minute/time.Millisecond), "lookback should be capped at 48 hours")

	emptyReq := make(map[string]any)
	addV3HistoryTimeRange(emptyReq, 0)
	assert.Empty(t, emptyReq, "zero lookback should not set a time range")
}

func TestFGetContractInfo(t *testing.T) {
	t.Parallel()
	_, err := e.FGetContractInfo(t.Context(), "", "", currency.EMPTYPAIR)
	require.NoError(t, err)
}

func TestFIndexPriceInfo(t *testing.T) {
	t.Parallel()
	_, err := e.FIndexPriceInfo(t.Context(), currency.BTC)
	require.NoError(t, err)
}

func TestFContractPriceLimitations(t *testing.T) {
	t.Parallel()
	_, err := e.FContractPriceLimitations(t.Context(),
		"BTC", "this_week", currency.EMPTYPAIR)
	require.NoError(t, err)
}

func TestFContractOpenInterest(t *testing.T) {
	t.Parallel()
	_, err := e.FContractOpenInterest(t.Context(), "BTC", "this_week", currency.EMPTYPAIR)
	require.NoError(t, err)
}

func TestFGetEstimatedDeliveryPrice(t *testing.T) {
	t.Parallel()
	_, err := e.FGetEstimatedDeliveryPrice(t.Context(), currency.BTC)
	require.NoError(t, err)
}

func TestFGetMarketDepth(t *testing.T) {
	t.Parallel()
	_, err := e.FGetMarketDepth(t.Context(), btccwPair, "step5")
	require.NoError(t, err)
}

func TestFGetKlineData(t *testing.T) {
	t.Parallel()
	_, err := e.FGetKlineData(t.Context(), btccwPair, "5min", 5, time.Now().Add(-time.Minute*5), time.Now())
	require.NoError(t, err)
}

func TestFGetMarketOverviewData(t *testing.T) {
	t.Parallel()
	_, err := e.FGetMarketOverviewData(t.Context(), btccwPair)
	require.NoError(t, err)
}

func TestFLastTradeData(t *testing.T) {
	t.Parallel()
	_, err := e.FLastTradeData(t.Context(), btccwPair)
	require.NoError(t, err)
}

func TestFRequestPublicBatchTrades(t *testing.T) {
	t.Parallel()
	_, err := e.FRequestPublicBatchTrades(t.Context(), btccwPair, 50)
	require.NoError(t, err)
}

func TestFQueryTieredAdjustmentFactor(t *testing.T) {
	t.Parallel()
	_, err := e.FQueryTieredAdjustmentFactor(t.Context(), currency.BTC)
	require.NoError(t, err)
}

func TestFQueryHisOpenInterest(t *testing.T) {
	t.Parallel()
	_, err := e.FQueryHisOpenInterest(t.Context(), "BTC", "this_week", "60min", "cont", 3)
	require.NoError(t, err)
}

func TestFQuerySystemStatus(t *testing.T) {
	t.Parallel()
	_, err := e.FQuerySystemStatus(t.Context(), currency.BTC)
	require.NoError(t, err)
}

func TestFQueryTopAccountsRatio(t *testing.T) {
	t.Parallel()
	_, err := e.FQueryTopAccountsRatio(t.Context(), "BTC", "5min")
	require.NoError(t, err)
}

func TestFQueryTopPositionsRatio(t *testing.T) {
	t.Parallel()
	_, err := e.FQueryTopPositionsRatio(t.Context(), "BTC", "5min")
	require.NoError(t, err)
}

func TestFLiquidationOrders(t *testing.T) {
	t.Parallel()
	if _, err := e.FLiquidationOrders(t.Context(), currency.BTC, "filled", 0, 0, "", 0); err != nil {
		t.Error(err)
	}
}

func TestFIndexKline(t *testing.T) {
	t.Parallel()
	_, err := e.FIndexKline(t.Context(), btccwPair, "5min", 5)
	require.NoError(t, err)
}

func TestFGetBasisData(t *testing.T) {
	t.Parallel()
	_, err := e.FGetBasisData(t.Context(), btccwPair, "5min", "open", 3)
	require.NoError(t, err)
}

func TestFGetAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.FGetAccountInfo(t.Context(), currency.EMPTYCODE)
	require.NoError(t, err)
}

func TestFGetPositionsInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.FGetPositionsInfo(t.Context(), currency.EMPTYCODE)
	require.NoError(t, err)
}

func TestFGetAllSubAccountAssets(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.FGetAllSubAccountAssets(t.Context(), currency.EMPTYCODE)
	require.NoError(t, err)
}

func TestFGetSingleSubAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.FGetSingleSubAccountInfo(t.Context(), "", "154263566")
	require.NoError(t, err)
}

func TestFGetSingleSubPositions(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.FGetSingleSubPositions(t.Context(), "", "154263566")
	require.NoError(t, err)
}

func TestFGetFinancialRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.FGetFinancialRecords(t.Context(),
		"BTC", "closeLong", 2, 0, 0)
	require.NoError(t, err)
}

func TestFGetSettlementRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.FGetSettlementRecords(t.Context(),
		currency.BTC, 0, 0, time.Now().Add(-48*time.Hour), time.Now())
	require.NoError(t, err)
}

func TestFGetOrderLimits(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")

	_, err := h.FGetOrderLimits(t.Context(), "BTC", "not-real")
	require.Error(t, err, "FGetOrderLimits must reject invalid order price type")

	h.API.AuthenticatedSupport = true
	_, err = h.FGetOrderLimits(t.Context(), "BTC", "limit")
	require.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "FGetOrderLimits must return credentials error")
}

func TestFContractTradingFee(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.FContractTradingFee(t.Context(), currency.EMPTYCODE)
	require.NoError(t, err)
}

func TestFGetTransferLimits(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.FGetTransferLimits(t.Context(), currency.EMPTYCODE)
	require.NoError(t, err)
}

func TestFGetPositionLimits(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.FGetPositionLimits(t.Context(), currency.EMPTYCODE)
	require.NoError(t, err)
}

func TestFGetAssetsAndPositions(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.FGetAssetsAndPositions(t.Context(), currency.HT)
	require.NoError(t, err)
}

func TestFTransfer(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.FTransfer(t.Context(), "154263566", "HT", "sub_to_master", 5)
	require.NoError(t, err)
}

func TestFGetTransferRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.FGetTransferRecords(t.Context(), "HT", "master_to_sub", 90, 0, 0)
	require.NoError(t, err)
}

func TestFGetAvailableLeverage(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.FGetAvailableLeverage(t.Context(), currency.BTC)
	require.NoError(t, err)
}

func TestFOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.FOrder(t.Context(), currency.EMPTYPAIR, "BTC", "quarter", "123", "BUY", "open", "limit", 1, 1, 1)
	require.NoError(t, err)
}

func TestFPlaceBatchOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.FPlaceBatchOrder(t.Context(), []fBatchOrderData{
		{
			Symbol:         "btc",
			ContractType:   "quarter",
			Price:          5,
			Volume:         1,
			Direction:      "buy",
			Offset:         "open",
			LeverageRate:   1,
			OrderPriceType: "limit",
		},
		{
			Symbol:         "xrp",
			ContractType:   "this_week",
			Price:          10000,
			Volume:         1,
			Direction:      "sell",
			Offset:         "open",
			LeverageRate:   1,
			OrderPriceType: "limit",
		},
	})
	require.NoError(t, err)
}

func TestFCancelOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.FCancelOrder(t.Context(), currency.BTC, "123", "")
	require.NoError(t, err)
}

func TestFCancelAllOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	updatePairsOnce(t, e)
	_, err := e.FCancelAllOrders(t.Context(), btcFutureDatedPair, "", "")
	require.NoError(t, err)
}

func TestFFlashCloseOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.FFlashCloseOrder(t.Context(),
		currency.EMPTYPAIR, "BTC", "quarter", "BUY", "lightning", "", 1)
	require.NoError(t, err)
}

func TestFGetOrderInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.FGetOrderInfo(t.Context(), "BTC", "", "123")
	require.NoError(t, err)
}

func TestFOrderDetails(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.FOrderDetails(t.Context(), "BTC", "123", "quotation", time.Now().Add(-1*time.Hour), 0, 0)
	require.NoError(t, err)
}

func TestFGetOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.FGetOpenOrders(t.Context(), currency.BTC, 1, 2)
	require.NoError(t, err)
}

func TestFGetOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.FGetOrderHistory(t.Context(),
		currency.EMPTYPAIR, "BTC",
		"all", "all", "limit",
		[]order.Status{},
		5, 0, 0)
	require.NoError(t, err)
}

func TestFTradeHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.FTradeHistory(t.Context(), currency.EMPTYPAIR, "BTC", "all", 10, 0, 0)
	require.NoError(t, err)
}

func TestFPlaceTriggerOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.FPlaceTriggerOrder(t.Context(), currency.EMPTYPAIR, "EOS", "quarter", "greaterOrEqual", "limit", "buy", "close", 1.1, 1.05, 5, 2)
	require.NoError(t, err)
}

func TestFCancelTriggerOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.FCancelTriggerOrder(t.Context(), "ETH", "123")
	require.NoError(t, err)
}

func TestFCancelAllTriggerOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.FCancelAllTriggerOrders(t.Context(), currency.EMPTYPAIR, "BTC", "this_week")
	require.NoError(t, err)
}

func TestFQueryTriggerOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.FQueryTriggerOpenOrders(t.Context(), currency.EMPTYPAIR, "BTC", 0, 0)
	require.NoError(t, err)
}

func TestFQueryTriggerOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.FQueryTriggerOrderHistory(t.Context(), currency.EMPTYPAIR, "EOS", "all", "all", 10, 0, 0)
	require.NoError(t, err)
}

func TestFormatFuturesPair(t *testing.T) {
	t.Parallel()
	updatePairsOnce(t, e)

	r, err := e.formatFuturesPair(btccwPair, false)
	require.NoError(t, err)
	assert.Equal(t, "BTC_CW", r)

	// pair in the format of BTC210827 but make it lower case to test correct formatting
	r, err = e.formatFuturesPair(btcFutureDatedPair.Lower(), false)
	require.NoError(t, err)
	assert.Len(t, r, 9, "Should be an 9 character string")
	assert.Equal(t, "BTC2", r[0:4], "Should start with btc and a date this millennium")

	r, err = e.formatFuturesPair(btccwPair, true)
	require.NoError(t, err)
	assert.Len(t, r, 9, "Should be an 9 character string")
	assert.Equal(t, "BTC2", r[0:4], "Should start with btc and a date this millennium")

	r, err = e.formatFuturesPair(currency.NewBTCUSDT(), false)
	require.NoError(t, err)
	assert.Equal(t, "BTC-USDT", r)
}

var expiryWindows = map[string]uint{
	"CW": 14,
	"NW": 21,
	"CQ": 190,
	"NQ": 282,
}

// TestPairFromContractExpiryCode ensures at least some contract codes are available and loaded with sane dates
// Expectations are relaxed because dates are unpredictable and codes disappear intermittently
func TestPairFromContractExpiryCode(t *testing.T) {
	t.Parallel()

	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test Instance Setup must not fail")

	_, err := e.FetchTradablePairs(t.Context(), asset.Futures)
	require.NoError(t, err)

	tz, err := time.LoadLocation("Asia/Singapore") // HTX HQ and apparent local time for when codes become effective
	require.NoError(t, err, "LoadLocation must not error")

	today := time.Now()
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, tz) // Do not use Truncate; https://github.com/golang/go/issues/55921

	require.NotEmpty(t, e.futureContractCodes, "At least one contract code must be loaded")

	for cType, cachedContract := range e.futureContractCodes {
		t.Run(cType, func(t *testing.T) {
			t.Parallel()
			p, err := e.pairFromContractExpiryCode(currency.Pair{
				Base:  currency.BTC,
				Quote: currency.NewCode(cType),
			})
			require.NoError(t, err)
			assert.Equal(t, currency.BTC, p.Base, "pair Base should be BTC")
			assert.Equal(t, cachedContract, p.Quote, "pair Quote should match futureContractCodes value")
			exp, err := time.ParseInLocation("060102", p.Quote.String(), tz)
			require.NoError(t, err, "currency code must be a parsable date")
			require.Falsef(t, exp.Before(today), "expiry must be today or after; Got: %q", exp)
			diff := uint(exp.Sub(today).Hours() / 24)
			require.LessOrEqualf(t, diff, expiryWindows[cType], "expiry must be within expected update window; Today: %q, Expiry: %q",
				today.Format(time.DateOnly),
				exp.Format(time.DateOnly),
			)
		})
	}
}
