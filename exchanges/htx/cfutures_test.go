package htx

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func TestFinancialRecordDataUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var arrayResp FinancialRecordData
	err := json.Unmarshal([]byte(`{"code":200,"msg":"","data":[{"query_id":12,"id":34,"symbol":"ETH","contract_code":"ETH-USD","type":3,"amount":1.25,"ts":1604312615051}],"ts":1604312615051}`), &arrayResp)
	require.NoError(t, err, "FinancialRecordData unmarshal must support v3 array data")
	require.Len(t, arrayResp.Data.FinancialRecord, 1, "financial records must decode from v3 array data")
	assert.Equal(t, int64(12), arrayResp.Data.FinancialRecord[0].QueryID, "query id should decode")

	var emptyResp FinancialRecordData
	err = json.Unmarshal([]byte(`{"code":200,"msg":"","data":"","ts":1604312615051}`), &emptyResp)
	require.NoError(t, err, "FinancialRecordData unmarshal must support empty string data")
	assert.Empty(t, emptyResp.Data.FinancialRecord, "financial records should be empty")

	var legacyResp FinancialRecordData
	err = json.Unmarshal([]byte(`{"data":{"financial_record":[{"query_id":12,"id":34}],"total_page":2},"ts":1604312615051}`), &legacyResp)
	require.NoError(t, err, "FinancialRecordData unmarshal must support legacy object data")
	assert.Equal(t, int64(2), legacyResp.Data.TotalPage, "legacy total page should decode")

	err = json.Unmarshal([]byte(`{`), &legacyResp)
	require.Error(t, err, "FinancialRecordData unmarshal must return malformed JSON errors")
	err = json.Unmarshal([]byte(`{"data":1}`), &legacyResp)
	require.Error(t, err, "FinancialRecordData unmarshal must return malformed data errors")
}

func TestSwapOrderHistoryUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var arrayResp SwapOrderHistory
	err := json.Unmarshal([]byte(`{"code":200,"msg":"","data":[{"query_id":12,"order_id":34,"order_id_str":"34","symbol":"ETH","contract_code":"ETH-USD","lever_rate":20,"direction":"buy","offset":"open","volume":1,"price":10,"create_date":1604312615051,"order_source":"api","order_price_type":"limit","margin_frozen":0,"profit":0,"trade_volume":0,"trade_turnover":0,"fee":0,"trade_avg_price":0,"status":6,"order_type":1,"fee_asset":"ETH","liquidation_type":"0"}],"ts":1604312615051}`), &arrayResp)
	require.NoError(t, err, "SwapOrderHistory unmarshal must support v3 array data")
	require.Len(t, arrayResp.Data.Orders, 1, "orders must decode from v3 array data")
	assert.Equal(t, int64(12), arrayResp.Data.Orders[0].QueryID, "query id should decode")

	var emptyResp SwapOrderHistory
	err = json.Unmarshal([]byte(`{"code":200,"msg":"","data":"","ts":1604312615051}`), &emptyResp)
	require.NoError(t, err, "SwapOrderHistory unmarshal must support empty string data")
	assert.Empty(t, emptyResp.Data.Orders, "orders should be empty")

	var legacyResp SwapOrderHistory
	err = json.Unmarshal([]byte(`{"data":{"orders":[{"query_id":12,"order_id":34}],"total_page":2},"ts":1604312615051}`), &legacyResp)
	require.NoError(t, err, "SwapOrderHistory unmarshal must support legacy object data")
	assert.Equal(t, int64(2), legacyResp.Data.TotalPage, "legacy total page should decode")

	err = json.Unmarshal([]byte(`{`), &legacyResp)
	require.Error(t, err, "SwapOrderHistory unmarshal must return malformed JSON errors")
	err = json.Unmarshal([]byte(`{"data":1}`), &legacyResp)
	require.Error(t, err, "SwapOrderHistory unmarshal must return malformed data errors")
}

func TestAccountTradeHistoryDataUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var arrayResp AccountTradeHistoryData
	err := json.Unmarshal([]byte(`{"code":200,"msg":"","data":[{"query_id":12,"id":"match","match_id":34,"order_id":56,"order_id_str":"56","symbol":"ETH","contract_code":"ETH-USD","direction":"buy","offset":"open","trade_volume":1,"trade_price":10,"trade_turnover":10,"trade_fee":0.1,"offset_profitloss":0,"create_date":"1604312615051","role":"Maker","order_source":"api","fee_asset":"ETH"}],"ts":1604312615051}`), &arrayResp)
	require.NoError(t, err, "AccountTradeHistoryData unmarshal must support v3 array data")
	require.Len(t, arrayResp.Data.Trades, 1, "trades must decode from v3 array data")
	assert.Equal(t, int64(12), arrayResp.Data.Trades[0].QueryID, "query id should decode")

	var emptyResp AccountTradeHistoryData
	err = json.Unmarshal([]byte(`{"code":200,"msg":"","data":"","ts":1604312615051}`), &emptyResp)
	require.NoError(t, err, "AccountTradeHistoryData unmarshal must support empty string data")
	assert.Empty(t, emptyResp.Data.Trades, "trades should be empty")

	var legacyResp AccountTradeHistoryData
	err = json.Unmarshal([]byte(`{"data":{"trades":[{"query_id":12,"id":"match"}],"total_page":2},"ts":1604312615051}`), &legacyResp)
	require.NoError(t, err, "AccountTradeHistoryData unmarshal must support legacy object data")
	assert.Equal(t, int64(2), legacyResp.Data.TotalPage, "legacy total page should decode")

	err = json.Unmarshal([]byte(`{`), &legacyResp)
	require.Error(t, err, "AccountTradeHistoryData unmarshal must return malformed JSON errors")
	err = json.Unmarshal([]byte(`{"data":1}`), &legacyResp)
	require.Error(t, err, "AccountTradeHistoryData unmarshal must return malformed data errors")
}

func TestQuerySwapIndexPriceInfo(t *testing.T) {
	t.Parallel()
	_, err := e.QuerySwapIndexPriceInfo(t.Context(), btcusdPair)
	require.NoError(t, err)
}

func TestSwapOpenInterestInformation(t *testing.T) {
	t.Parallel()
	_, err := e.SwapOpenInterestInformation(t.Context(), btcusdPair)
	require.NoError(t, err)
}

func TestGetSwapMarketDepth(t *testing.T) {
	t.Parallel()
	_, err := e.GetSwapMarketDepth(t.Context(), btcusdPair, "step0")
	require.NoError(t, err)
}

func TestGetSwapKlineData(t *testing.T) {
	t.Parallel()
	r, err := e.GetSwapKlineData(t.Context(), btcusdPair, "5min", 5, time.Now().Add(-time.Hour), time.Now())
	require.NoError(t, err)
	assert.NotEmpty(t, r.Data, "GetSwapKlineData should return some data")
}

func TestGetSwapMarketOverview(t *testing.T) {
	t.Parallel()
	_, err := e.GetSwapMarketOverview(t.Context(), btcusdPair)
	require.NoError(t, err)
}

func TestGetLastTrade(t *testing.T) {
	t.Parallel()
	_, err := e.GetLastTrade(t.Context(), btcusdPair)
	require.NoError(t, err)
}

func TestGetBatchTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetBatchTrades(t.Context(), btcusdPair, 5)
	require.NoError(t, err)
}

func TestGetTieredAjustmentFactorInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetTieredAjustmentFactorInfo(t.Context(), btcusdPair)
	require.NoError(t, err)
}

func TestGetOpenInterestInfo(t *testing.T) {
	t.Parallel()
	updatePairsOnce(t, e)
	_, err := e.GetOpenInterestInfo(t.Context(), btcusdPair, "5min", "cryptocurrency", 50)
	require.NoError(t, err)
}

func TestGetTraderSentimentIndexAccount(t *testing.T) {
	t.Parallel()
	_, err := e.GetTraderSentimentIndexAccount(t.Context(), btcusdPair, "5min")
	require.NoError(t, err)
}

func TestGetTraderSentimentIndexPosition(t *testing.T) {
	t.Parallel()
	_, err := e.GetTraderSentimentIndexPosition(t.Context(), btcusdPair, "5min")
	require.NoError(t, err)
}

func TestGetLiquidationOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetLiquidationOrders(t.Context(), btcusdPair, "closed", time.Now().AddDate(0, 0, -2), time.Now(), "", 0)
	assert.NoError(t, err, "GetLiquidationOrders should not error")
}

func TestGetHistoricalFundingRates(t *testing.T) {
	t.Parallel()
	_, err := e.GetHistoricalFundingRatesForPair(t.Context(), btcusdPair, 0, 0)
	require.NoError(t, err)
}

func TestGetPremiumIndexKlineData(t *testing.T) {
	t.Parallel()
	_, err := e.GetPremiumIndexKlineData(t.Context(), btcusdPair, "5min", 15)
	require.NoError(t, err)
}

func TestGetEstimatedFundingRates(t *testing.T) {
	t.Parallel()
	_, err := e.GetEstimatedFundingRates(t.Context(), btcusdPair, "5min", 15)
	require.NoError(t, err)

	_, err = e.GetEstimatedFundingRates(t.Context(), btcusdPair, "invalid", 15)
	require.Error(t, err, "GetEstimatedFundingRates must reject invalid period")
}

func TestGetBasisData(t *testing.T) {
	t.Parallel()
	_, err := e.GetBasisData(t.Context(), btcusdPair, "5min", "close", 5)
	require.NoError(t, err)
}

func TestGetSystemStatusInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetSystemStatusInfo(t.Context(), btcusdPair)
	require.NoError(t, err)
}

func TestGetSwapPriceLimits(t *testing.T) {
	t.Parallel()
	_, err := e.GetSwapPriceLimits(t.Context(), btcusdPair)
	require.NoError(t, err)
}

func TestGetSwapAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetSwapAccountInfo(t.Context(), ethusdPair)
	require.NoError(t, err)
}

func TestGetSwapPositionsInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetSwapPositionsInfo(t.Context(), ethusdPair)
	require.NoError(t, err)
}

func TestGetSwapAssetsAndPositions(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetSwapAssetsAndPositions(t.Context(), ethusdPair)
	require.NoError(t, err)
}

func TestGetSwapAllSubAccAssets(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetSwapAllSubAccAssets(t.Context(), ethusdPair)
	require.NoError(t, err)
}

func TestGetSubAccPositionInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetSubAccPositionInfo(t.Context(), ethusdPair, 0)
	require.NoError(t, err)
}

func TestSwapSingleSubAccAssets(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	h.API.AuthenticatedSupport = true
	_, err := h.SwapSingleSubAccAssets(t.Context(), ethusdPair, 123)
	require.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "SwapSingleSubAccAssets must return credentials error")
}

func TestGetAccountFinancialRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetAccountFinancialRecords(t.Context(), ethusdPair, "3,4", 15, 0, 0)
	require.NoError(t, err)
}

func TestGetSwapSettlementRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	r, err := e.GetSwapSettlementRecords(t.Context(), ethusdPair, time.Now().AddDate(0, -1, 0), time.Now(), 0, 0)
	require.NoError(t, err)
	assert.NotEmpty(t, r.Data, "GetSwapSettlementRecords should return some data")
}

func TestGetAvailableLeverage(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetAvailableLeverage(t.Context(), ethusdPair)
	require.NoError(t, err)
}

func TestGetSwapOrderLimitInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetSwapOrderLimitInfo(t.Context(), ethusdPair, "limit")
	require.NoError(t, err)
}

func TestGetSwapTradingFeeInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetSwapTradingFeeInfo(t.Context(), ethusdPair)
	require.NoError(t, err)
}

func TestGetSwapTransferLimitInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetSwapTransferLimitInfo(t.Context(), ethusdPair)
	require.NoError(t, err)
}

func TestGetSwapPositionLimitInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetSwapPositionLimitInfo(t.Context(), ethusdPair)
	require.NoError(t, err)
}

func TestAccountTransferData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.AccountTransferData(t.Context(), ethusdPair, "123", "master_to_sub", 15)
	require.NoError(t, err)
}

func TestAccountTransferRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.AccountTransferRecords(t.Context(), ethusdPair, "master_to_sub", 12, 0, 0)
	require.NoError(t, err)
}

func TestPlaceSwapOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.PlaceSwapOrders(t.Context(), ethusdPair, "", "buy", "open", "limit", 0.01, 1, 1)
	require.NoError(t, err)
}

func TestPlaceSwapBatchOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	var req BatchOrderRequestType
	order1 := batchOrderData{
		ContractCode:   "ETH-USD",
		ClientOrderID:  "",
		Price:          5,
		Volume:         1,
		Direction:      "buy",
		Offset:         "open",
		LeverageRate:   1,
		OrderPriceType: "limit",
	}
	order2 := batchOrderData{
		ContractCode:   "BTC-USD",
		ClientOrderID:  "",
		Price:          2.5,
		Volume:         1,
		Direction:      "buy",
		Offset:         "open",
		LeverageRate:   1,
		OrderPriceType: "limit",
	}
	req.Data = append(req.Data, order1, order2)

	_, err := e.PlaceSwapBatchOrders(t.Context(), req)
	require.NoError(t, err)
}

func TestCancelSwapOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.CancelSwapOrder(t.Context(), "test123", "", ethusdPair)
	require.NoError(t, err)
}

func TestCancelAllSwapOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.CancelAllSwapOrders(t.Context(), ethusdPair)
	require.NoError(t, err)
}

func TestPlaceLightningCloseOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.PlaceLightningCloseOrder(t.Context(), ethusdPair, "buy", "lightning", 5, 1)
	require.NoError(t, err)
}

func TestGetSwapOrderInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetSwapOrderInfo(t.Context(), ethusdPair, "123", "")
	require.NoError(t, err)
}

func TestGetSwapOrderDetails(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetSwapOrderDetails(t.Context(), ethusdPair, "123", "10", "cancelledOrder", 0, 0)
	require.NoError(t, err)
}

func TestGetSwapOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetSwapOpenOrders(t.Context(), ethusdPair, 0, 0)
	require.NoError(t, err)
}

func TestGetSwapOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetSwapOrderHistory(t.Context(), ethusdPair, "all", "all", []order.Status{order.PartiallyCancelled, order.Active}, 25, 0, 0)
	require.NoError(t, err)
}

func TestGetSwapTradeHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetSwapTradeHistory(t.Context(), ethusdPair, "liquidateShort", 10, 0, 0)
	require.NoError(t, err)
}

func TestPlaceSwapTriggerOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.PlaceSwapTriggerOrder(t.Context(), ethusdPair, "greaterOrEqual", "buy", "open", "optimal_5", 5, 3, 1, 1)
	require.NoError(t, err)
}

func TestCancelSwapTriggerOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.CancelSwapTriggerOrder(t.Context(), ethusdPair, "test123")
	require.NoError(t, err)
}

func TestCancelAllSwapTriggerOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.CancelAllSwapTriggerOrders(t.Context(), ethusdPair)
	require.NoError(t, err)
}

func TestGetSwapTriggerOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetSwapTriggerOrderHistory(t.Context(), ethusdPair, "open", "all", 15, 0, 0)
	require.NoError(t, err)
}

func TestGetSwapMarkets(t *testing.T) {
	t.Parallel()
	_, err := e.GetSwapMarkets(t.Context(), currency.EMPTYPAIR)
	require.NoError(t, err)
}

func TestGetSwapFundingRates(t *testing.T) {
	t.Parallel()
	_, err := e.GetSwapFundingRates(t.Context())
	require.NoError(t, err)
}

func TestGetBatchCoinMarginSwapContracts(t *testing.T) {
	t.Parallel()
	resp, err := e.GetBatchCoinMarginSwapContracts(t.Context())
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}
