package mexc

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// The replies below are the venue's documented examples, as no credentials were available to record live ones. Some
// values are changed from the examples' zeros, so that more fields' decoding is checked

func TestSetFuturesTimeWindow(t *testing.T) {
	t.Parallel()
	start := time.UnixMilli(1761000000000)
	params := make(map[string][]string)
	require.NoError(t, setFuturesTimeWindow(params, start, start.Add(futuresHistoryWindow)), "a 90 day window must be accepted")
	assert.Equal(t, map[string][]string{"start_time": {"1761000000000"}, "end_time": {"1768776000000"}}, params, "the window should be sent in milliseconds")

	assert.ErrorIs(t, setFuturesTimeWindow(params, start, start.Add(futuresHistoryWindow+time.Millisecond)), errHistoryWindowTooWide)
	assert.ErrorIs(t, setFuturesTimeWindow(params, start, start.Add(-time.Second)), common.ErrStartAfterEnd)

	params = make(map[string][]string)
	require.NoError(t, setFuturesTimeWindow(params, time.Time{}, start), "an open start must be accepted")
	assert.Equal(t, map[string][]string{"end_time": {"1761000000000"}}, params, "only the set end should be sent")
}

func TestGetFuturesAccountAssets(t *testing.T) {
	t.Parallel()
	ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
		method: http.MethodGet,
		path:   "private/account/assets",
		data: `[{"currency":"USDT","positionMargin":1.5,"availableBalance":32.80984793658,"cashBalance":32.80984793,` +
			`"frozenBalance":2,"equity":36.30984793,"unrealized":0,"bonus":0,"availableCash":32.80984793658,` +
			`"availableOpen":32.80984793658,"debtAmount":0,"contributeMarginAmount":0,"vcoinId":"128f589271cb4951b03e71e6323eb7be"}]`,
	}), liveReadOnly)
	result, err := ex.GetFuturesAccountAssets(t.Context())
	require.NoError(t, err, "GetFuturesAccountAssets must not error")
	if !mockTests {
		return
	}
	exp := []*FuturesAccountAssetResponse{{
		Currency:         currency.USDT,
		PositionMargin:   1.5,
		FrozenBalance:    2,
		AvailableBalance: 32.80984793658,
		CashBalance:      32.80984793,
		Equity:           36.30984793,
		AvailableCash:    32.80984793658,
		AvailableOpen:    32.80984793658,
		CurrencyID:       "128f589271cb4951b03e71e6323eb7be",
	}}
	assert.Equal(t, exp, result, "GetFuturesAccountAssets should decode every field")
}

func TestGetFuturesAccountAsset(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesAccountAsset(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
		method: http.MethodGet,
		path:   "private/account/asset/MX",
		data: `{"currency":"MX","positionMargin":0,"availableBalance":0.9964714959772486,"cashBalance":0.99647149,"frozenBalance":0,` +
			`"equity":0.99647149,"unrealized":0,"bonus":0,"availableCash":0.9964714959772486,"availableOpen":0.9964714959772486,` +
			`"debtAmount":0,"contributeMarginAmount":0,"vcoinId":"8c1e92655f9a4064808ce03ec1a48d38"}`,
	}), liveReadOnly)
	result, err := ex.GetFuturesAccountAsset(t.Context(), currency.NewCode("mx"))
	require.NoError(t, err, "GetFuturesAccountAsset must not error")
	if !mockTests {
		return
	}
	exp := &FuturesAccountAssetResponse{
		Currency:         currency.MX,
		AvailableBalance: 0.9964714959772486,
		CashBalance:      0.99647149,
		Equity:           0.99647149,
		AvailableCash:    0.9964714959772486,
		AvailableOpen:    0.9964714959772486,
		CurrencyID:       "8c1e92655f9a4064808ce03ec1a48d38",
	}
	assert.Equal(t, exp, result, "GetFuturesAccountAsset should decode every field")
}

func TestGetFuturesAssetTransferRecords(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesAssetTransferRecords(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.GetFuturesAssetTransferRecords(t.Context(), &FuturesTransferRecordsRequest{PageSize: futuresDefaultPageSizeLimit + 1})
	require.ErrorIs(t, err, errInvalidLimit)

	ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
		method: http.MethodGet,
		path:   "private/account/transfer_record",
		query:  "currency=USDT&page_num=1&page_size=20&state=SUCCESS&type=IN",
		data: `{"pageSize":20,"totalCount":1,"totalPage":1,"currentPage":1,"resultList":[{"id":303381159,` +
			`"txid":"86f334f8850346e89ce4d0ba52b7aa87","currency":"USDT","amount":1,"type":"IN","state":"SUCCESS",` +
			`"createTime":1760537642000,"updateTime":1760537641000}]}`,
	}), liveReadOnly)
	result, err := ex.GetFuturesAssetTransferRecords(t.Context(), &FuturesTransferRecordsRequest{
		Currency: currency.USDT,
		State:    "SUCCESS",
		Type:     "IN",
		PageSize: 20,
	})
	require.NoError(t, err, "GetFuturesAssetTransferRecords must not error")
	if !mockTests {
		return
	}
	exp := &FuturesPageResponse[FuturesTransferRecordResponse]{
		PageSize:    20,
		TotalCount:  1,
		TotalPage:   1,
		CurrentPage: 1,
		ResultList: []FuturesTransferRecordResponse{{
			ID:            303381159,
			TransactionID: "86f334f8850346e89ce4d0ba52b7aa87",
			Currency:      currency.USDT,
			Amount:        1,
			Type:          "IN",
			State:         "SUCCESS",
			CreateTime:    types.Time(time.UnixMilli(1760537642000)),
			UpdateTime:    types.Time(time.UnixMilli(1760537641000)),
		}},
	}
	assert.Equal(t, exp, result, "GetFuturesAssetTransferRecords should decode the page")
}

func TestGetFuturesProfitRate(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesProfitRate(t.Context(), 3)
	require.ErrorIs(t, err, errInvalidProfitRateType)

	ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
		method: http.MethodGet,
		path:   "private/account/profit_rate/2",
		data:   `{"ranking":0,"profitRate":0.0125,"statisticTime":1666612800000}`,
	}), liveReadOnly)
	result, err := ex.GetFuturesProfitRate(t.Context(), FuturesProfitRateWeek)
	require.NoError(t, err, "GetFuturesProfitRate must not error")
	if !mockTests {
		return
	}
	assert.Equal(t, &FuturesProfitRateResponse{ProfitRate: 0.0125, StatisticTime: types.Time(time.UnixMilli(1666612800000))}, result,
		"GetFuturesProfitRate should decode every field")
}

func TestGetFuturesFeeDeductionConfigs(t *testing.T) {
	t.Parallel()
	ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
		method: http.MethodGet,
		path:   "private/account/feeDeductConfigs",
		data:   `[{"deductCoin":"MX","settleCoin":"USDT","discountRatio":0.8},{"deductCoin":"MXPOINT","settleCoin":"USDT","discountRatio":1}]`,
	}), liveReadOnly)
	result, err := ex.GetFuturesFeeDeductionConfigs(t.Context())
	require.NoError(t, err, "GetFuturesFeeDeductionConfigs must not error")
	if !mockTests {
		return
	}
	exp := []*FuturesFeeDeductionConfigResponse{
		{DeductCoin: currency.MX, SettleCoin: currency.USDT, DiscountRatio: 0.8},
		{DeductCoin: currency.NewCode("MXPOINT"), SettleCoin: currency.USDT, DiscountRatio: 1},
	}
	assert.Equal(t, exp, result, "GetFuturesFeeDeductionConfigs should decode every configuration")
}

func TestGetFuturesSpotDiscountConfigs(t *testing.T) {
	t.Parallel()
	ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
		method: http.MethodGet,
		path:   "private/account/config/contractFeeDiscountConfig",
		data: `[{"id":1,"currency":"MX","state":"ENABLED","discountConfig":"[{\"day\":1,\"discount\":0.5,\"level\":1,` +
			`\"volMax\":2147483647,\"volMin\":500}]","ineffectiveDiscountConfig":"","operator":"","createTime":1699775818000,` +
			`"updateTime":1732982217000,"enabled":true}]`,
	}), liveReadOnly)
	result, err := ex.GetFuturesSpotDiscountConfigs(t.Context())
	require.NoError(t, err, "GetFuturesSpotDiscountConfigs must not error")
	if !mockTests {
		return
	}
	// The discount bands are held in a string, which is empty where there are none
	exp := []*FuturesDiscountConfigResponse{{
		ID:             1,
		Currency:       currency.MX,
		State:          "ENABLED",
		DiscountConfig: FuturesDiscountTiers{{Level: 1, VolumeMin: 500, VolumeMax: 2147483647, Day: 1, Discount: 0.5}},
		CreateTime:     types.Time(time.UnixMilli(1699775818000)),
		UpdateTime:     types.Time(time.UnixMilli(1732982217000)),
		Enabled:        true,
	}}
	assert.Equal(t, exp, result, "GetFuturesSpotDiscountConfigs should decode every field")
}

func TestFuturesDiscountTiersUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for input, exp := range map[string]FuturesDiscountTiers{
		`"[{\"level\":2,\"discount\":0.7}]"`: {{Level: 2, Discount: 0.7}},
		`[{"level":3,"discount":0.6}]`:       {{Level: 3, Discount: 0.6}},
		`""`:                                 nil,
		`null`:                               nil,
	} {
		var tiers FuturesDiscountTiers
		require.NoErrorf(t, json.Unmarshal([]byte(input), &tiers), "Unmarshal must not error for %s", input)
		assert.Equalf(t, exp, tiers, "%s should decode exactly", input)
	}
	var tiers FuturesDiscountTiers
	assert.Error(t, json.Unmarshal([]byte(`"{"`), &tiers), "malformed bands within a string should be refused")
}

func TestGetFuturesFeeDeductionDetails(t *testing.T) {
	t.Parallel()
	start := time.UnixMilli(1761800000000)
	_, err := e.GetFuturesFeeDeductionDetails(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.GetFuturesFeeDeductionDetails(t.Context(), &FuturesFeeDeductionDetailsRequest{})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = e.GetFuturesFeeDeductionDetails(t.Context(), &FuturesFeeDeductionDetailsRequest{Symbol: futuresTradablePair, DealIDs: make([]string, futuresFeeDeductionDealsLimit+1)})
	require.ErrorIs(t, err, errTooManyOrders)
	_, err = e.GetFuturesFeeDeductionDetails(t.Context(), &FuturesFeeDeductionDetailsRequest{Symbol: futuresTradablePair, StartTime: start, EndTime: start.Add(futuresHistoryWindow + time.Hour)})
	require.ErrorIs(t, err, errHistoryWindowTooWide)

	ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
		method: http.MethodGet,
		path:   "private/order/fee_details",
		query:  "end_time=1761886400000&ids=54236193%2C52060733&page_num=1&start_time=1761800000000&symbol=BTC_USDT",
		data: `[{"id":54236193,"orderId":739132568196170240,"feeCurrency":"USDT","fee":0.0906990372,"deductType":2,` +
			`"deductFeeCurrency":"USDT","deductFee":0.0906990372,"timestamp":1761893337000},{"id":52060733,` +
			`"orderId":735908917455962624,"feeCurrency":"USDT","fee":0.297936756,"deductType":0,"timestamp":1761124758000}]`,
	}), liveUnavailable)
	result, err := ex.GetFuturesFeeDeductionDetails(t.Context(), &FuturesFeeDeductionDetailsRequest{
		Symbol:    futuresTradablePair,
		DealIDs:   []string{"54236193", "52060733"},
		StartTime: start,
		EndTime:   start.Add(24 * time.Hour),
	})
	require.NoError(t, err, "GetFuturesFeeDeductionDetails must not error")
	// The order IDs pass 2^53, so they must keep every digit
	exp := []*FuturesFeeDeductionDetailResponse{
		{
			ID:                "54236193",
			OrderID:           "739132568196170240",
			FeeCurrency:       currency.USDT,
			Fee:               0.0906990372,
			DeductType:        2,
			DeductFeeCurrency: currency.USDT,
			DeductFee:         0.0906990372,
			Timestamp:         types.Time(time.UnixMilli(1761893337000)),
		},
		{ID: "52060733", OrderID: "735908917455962624", FeeCurrency: currency.USDT, Fee: 0.297936756, Timestamp: types.Time(time.UnixMilli(1761124758000))},
	}
	assert.Equal(t, exp, result, "GetFuturesFeeDeductionDetails should decode every deal")
}

func TestGetFuturesDiscountType(t *testing.T) {
	t.Parallel()
	ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
		method: http.MethodGet,
		path:   "private/account/discountType",
		data:   `{"useFeeDiscount":true,"useFeeDeduct":false}`,
	}), liveReadOnly)
	result, err := ex.GetFuturesDiscountType(t.Context())
	require.NoError(t, err, "GetFuturesDiscountType must not error")
	if !mockTests {
		return
	}
	assert.Equal(t, &FuturesDiscountTypeResponse{UseFeeDiscount: true}, result, "GetFuturesDiscountType should decode both flags")
}

func TestGetFuturesFeeStatistics(t *testing.T) {
	t.Parallel()
	ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
		method: http.MethodGet,
		path:   "private/account/asset_book/order_deal_fee/total",
		data:   `{"totalFee":1.2345,"saveFee":0.1}`,
	}), liveReadOnly)
	result, err := ex.GetFuturesFeeStatistics(t.Context())
	require.NoError(t, err, "GetFuturesFeeStatistics must not error")
	if !mockTests {
		return
	}
	assert.Equal(t, &FuturesFeeStatisticsResponse{TotalFee: 1.2345, SaveFee: 0.1}, result, "GetFuturesFeeStatistics should decode every field")
}

func TestFuturesPositionsParams(t *testing.T) {
	t.Parallel()
	_, err := futuresPositionsParams(nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = futuresPositionsParams(&FuturesPositionsRequest{PositionType: 3})
	require.ErrorIs(t, err, errInvalidPositionType)
	_, err = futuresPositionsParams(&FuturesPositionsRequest{PageSize: futuresDefaultPageSizeLimit + 1})
	require.ErrorIs(t, err, errInvalidLimit)

	params, err := futuresPositionsParams(&FuturesPositionsRequest{
		Symbol:       futuresTradablePair,
		PositionID:   "1027177055",
		PositionType: FuturesPositionTypeShort,
		StartTime:    time.UnixMilli(1757500000000),
		EndTime:      time.UnixMilli(1757600000000),
		PageNumber:   2, PageSize: 50,
	})
	require.NoError(t, err, "futuresPositionsParams must not error")
	assert.Equal(t, "end_time=1757600000000&page_num=2&page_size=50&position_id=1027177055&position_type=2&start_time=1757500000000&symbol=BTC_USDT",
		params.Encode(), "every filter should be sent")
}

func TestGetFuturesHistoricalPositions(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesHistoricalPositions(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
		method: http.MethodGet,
		path:   "private/position/list/history_positions",
		query:  "page_num=1&symbol=BTC_USDT",
		data: `[{"positionId":1027177055,"symbol":"BTC_USDT","positionType":1,"openType":1,"state":3,"holdVol":0,"frozenVol":0,` +
			`"closeVol":2,"holdAvgPrice":112449.5,"holdAvgPriceFullyScale":"112449.5","openAvgPrice":112449.5,` +
			`"openAvgPriceFullyScale":"112449.5","closeAvgPrice":113572.3,"liquidatePrice":56314.7,"oim":0,"im":0,"holdFee":0,` +
			`"realised":0.2155,"leverage":2,"createTime":1757506663000,"updateTime":1757509217000,"autoAddIm":false,"version":3,` +
			`"profitRatio":0.0191,"newOpenAvgPrice":112449.5,"newCloseAvgPrice":113572.3,"closeProfitLoss":0.2245,"fee":-0.009,` +
			`"positionShowStatus":"CLOSED","deductFeeList":[],"totalFee":0.009,"zeroSaveTotalFeeBinance":0,"zeroTradeTotalFeeBinance":0.009}]`,
	}), liveReadOnly)
	result, err := ex.GetFuturesHistoricalPositions(t.Context(), &FuturesPositionsRequest{Symbol: futuresTradablePair})
	require.NoError(t, err, "GetFuturesHistoricalPositions must not error")
	if !mockTests {
		return
	}
	exp := []*FuturesPositionResponse{{
		PositionID:               "1027177055",
		Symbol:                   "BTC_USDT",
		PositionType:             FuturesPositionTypeLong,
		OpenType:                 FuturesOpenTypeIsolated,
		State:                    FuturesPositionStateClosed,
		CloseVolume:              2,
		HoldAveragePrice:         112449.5,
		HoldAveragePriceExact:    112449.5,
		CloseAveragePrice:        113572.3,
		OpenAveragePrice:         112449.5,
		OpenAveragePriceExact:    112449.5,
		NewOpenAveragePrice:      112449.5,
		NewCloseAveragePrice:     113572.3,
		LiquidatePrice:           56314.7,
		RealisedPNL:              0.2155,
		Leverage:                 2,
		ProfitRatio:              0.0191,
		CloseProfitLoss:          0.2245,
		Fee:                      -0.009,
		TotalFee:                 0.009,
		DeductFeeList:            []FuturesDeductFee{},
		PositionShowStatus:       "CLOSED",
		ZeroTradeTotalFeeBinance: 0.009,
		Version:                  3,
		CreateTime:               types.Time(time.UnixMilli(1757506663000)),
		UpdateTime:               types.Time(time.UnixMilli(1757509217000)),
	}}
	assert.Equal(t, exp, result, "GetFuturesHistoricalPositions should decode every field")
}

func TestGetFuturesOpenPositions(t *testing.T) {
	t.Parallel()
	ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
		method: http.MethodGet,
		path:   "private/position/open_positions",
		query:  "symbol=BTC_USDT",
		data: `[{"positionId":1109973831,"symbol":"BTC_USDT","positionType":1,"openType":1,"state":1,"holdVol":5,"frozenVol":0,` +
			`"closeVol":0,"holdAvgPrice":109777.5,"holdAvgPriceFullyScale":"109777.5","openAvgPrice":109777.5,` +
			`"openAvgPriceFullyScale":"109777.5","closeAvgPrice":0,"liquidatePrice":55020.5,"oim":27.444375,"im":27.444375,` +
			`"holdFee":0,"realised":0,"leverage":2,"marginRatio":0.0027,"createTime":1761887133854,"updateTime":1761887133854,` +
			`"autoAddIm":false,"version":1,"profitRatio":0,"newOpenAvgPrice":109777.5,"newCloseAvgPrice":0,"closeProfitLoss":0,` +
			`"fee":0,"deductFeeList":[],"totalFee":0,"zeroSaveTotalFeeBinance":0.0301,"zeroTradeTotalFeeBinance":0.0301,` +
			`"lowSaveTotalFee":0,"positionDrop":false,"feeRates":{"min":0,"max":0.0002},"unRealizedPnl":-0.0039}]`,
	}), liveReadOnly)
	result, err := ex.GetFuturesOpenPositions(t.Context(), futuresTradablePair, "")
	require.NoError(t, err, "GetFuturesOpenPositions must not error")
	if !mockTests {
		return
	}
	exp := []*FuturesPositionResponse{{
		PositionID:               "1109973831",
		Symbol:                   "BTC_USDT",
		HoldVolume:               5,
		PositionType:             FuturesPositionTypeLong,
		OpenType:                 FuturesOpenTypeIsolated,
		State:                    FuturesPositionStateHolding,
		HoldAveragePrice:         109777.5,
		HoldAveragePriceExact:    109777.5,
		OpenAveragePrice:         109777.5,
		OpenAveragePriceExact:    109777.5,
		NewOpenAveragePrice:      109777.5,
		LiquidatePrice:           55020.5,
		OriginalInitialMargin:    27.444375,
		InitialMargin:            27.444375,
		Leverage:                 2,
		MarginRatio:              0.0027,
		DeductFeeList:            []FuturesDeductFee{},
		ZeroSaveTotalFeeBinance:  0.0301,
		ZeroTradeTotalFeeBinance: 0.0301,
		FeeRates:                 &FuturesFeeRange{Max: 0.0002},
		UnrealisedPNL:            -0.0039,
		Version:                  1,
		CreateTime:               types.Time(time.UnixMilli(1761887133854)),
		UpdateTime:               types.Time(time.UnixMilli(1761887133854)),
	}}
	assert.Equal(t, exp, result, "GetFuturesOpenPositions should decode every field")
}

func TestFuturesPositionUnrealisedPNL(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 1.5, (&FuturesPositionResponse{UnrealisedPNL: 1.5, PushedUnrealisedPNL: 2}).unrealisedPNL(), "a REST reply's unrealised PNL should be preferred")
	assert.Equal(t, 2.0, (&FuturesPositionResponse{PushedUnrealisedPNL: 2}).unrealisedPNL(), "a push's unrealised PNL should be read without a REST one")
}

func TestGetFuturesFundingFees(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesFundingFees(t.Context(), &FuturesPositionsRequest{PositionType: 9})
	require.ErrorIs(t, err, errInvalidPositionType)

	// The venue's example for this endpoint shows a funding rate, so the record follows its field table
	ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
		method: http.MethodGet,
		path:   "private/position/funding_records",
		query:  "page_num=1&position_type=1&symbol=BTC_USDT",
		data: `{"pageSize":20,"totalCount":1,"totalPage":1,"currentPage":1,"resultList":[{"id":7123,"symbol":"BTC_USDT",` +
			`"positionId":1109973831,"positionType":1,"positionValue":548.8875,"funding":-0.0104,"rate":-0.000019,` +
			`"settleTime":1761897600000}]}`,
	}), liveReadOnly)
	result, err := ex.GetFuturesFundingFees(t.Context(), &FuturesPositionsRequest{Symbol: futuresTradablePair, PositionType: FuturesPositionTypeLong})
	require.NoError(t, err, "GetFuturesFundingFees must not error")
	if !mockTests {
		return
	}
	// A paid funding fee is negative
	exp := []*FuturesFundingFeeResponse{{
		ID:            7123,
		Symbol:        "BTC_USDT",
		PositionID:    "1109973831",
		PositionType:  FuturesPositionTypeLong,
		PositionValue: 548.8875,
		Funding:       -0.0104,
		Rate:          -0.000019,
		SettleTime:    types.Time(time.UnixMilli(1761897600000)),
	}}
	assert.Equal(t, exp, result, "GetFuturesFundingFees should decode every field")
}

func TestGetFuturesRiskLimits(t *testing.T) {
	t.Parallel()
	ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
		method: http.MethodGet,
		path:   "private/account/risk_limit",
		query:  "symbol=BTC_USDT",
		data: `{"BTC_USDT":[{"level":6,"maxVol":4500000,"mmr":0.05,"imr":0.1,"maxLeverage":500,"symbol":"BTC_USDT",` +
			`"positionType":2,"openType":1,"leverage":2,"limitBySys":false,"currentMmr":0.05},{"level":1,"maxVol":50000,` +
			`"mmr":0.001,"imr":0.002,"maxLeverage":500,"symbol":"BTC_USDT","positionType":1,"openType":1,"leverage":2,"limitBySys":false}]}`,
	}), liveReadOnly)
	result, err := ex.GetFuturesRiskLimits(t.Context(), futuresTradablePair)
	require.NoError(t, err, "GetFuturesRiskLimits must not error")
	if !mockTests {
		return
	}
	exp := map[string][]*FuturesRiskLimitResponse{"BTC_USDT": {
		{
			Symbol:                       "BTC_USDT",
			PositionType:                 FuturesPositionTypeShort,
			OpenType:                     FuturesOpenTypeIsolated,
			Level:                        6,
			MaxVolume:                    4500000,
			MaxLeverage:                  500,
			MaintenanceMarginRate:        0.05,
			InitialMarginRate:            0.1,
			Leverage:                     2,
			CurrentMaintenanceMarginRate: 0.05,
		},
		{
			Symbol:                "BTC_USDT",
			PositionType:          FuturesPositionTypeLong,
			OpenType:              FuturesOpenTypeIsolated,
			Level:                 1,
			MaxVolume:             50000,
			MaxLeverage:           500,
			MaintenanceMarginRate: 0.001,
			InitialMarginRate:     0.002,
			Leverage:              2,
		},
	}}
	assert.Equal(t, exp, result, "GetFuturesRiskLimits should decode both position sides")
}

func TestGetFuturesFeeDetails(t *testing.T) {
	t.Parallel()
	ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
		method: http.MethodGet,
		path:   "private/account/tiered_fee_rate/v2",
		query:  "symbol=BTC_USDT",
		data: `{"symbol":"BTC_USDT","joinDiscount":true,"enjoyDiscount":true,"discountRate":0.5,"joinDeduct":true,"enjoyDeduct":true,` +
			`"deductRate":0.8,"originalMakerFee":0.0001,"originalTakerFee":0.0004,"realMakerFee":0.00005,"realTakerFee":0.0002,` +
			`"dealAmount":1250.5,"walletBalance":35.10272884,"inviterKyc":"","level":9999,"feeType":1,"agentFee":false,` +
			`"feeRateMode":"TIERED","leverageFeeRates":[],"tieredFeeRates":[{"takerFeeRate":0,"makerFeeRate":0,` +
			`"minTieredDealAmount":0,"maxTieredDealAmount":10000000,"realTakerFee":0,"realMakerFee":0},{"takerFeeRate":0.0004,` +
			`"makerFeeRate":0.0001,"minTieredDealAmount":10000001,"realTakerFee":0.0002,"realMakerFee":0.00005}],` +
			`"tieredDealAmount":54,"tieredEffectiveDay":0,"tieredAppointContract":true,"tieredExcludeContractId":true,` +
			`"tieredContractIds":[10,77,1104],"tieredExcludeZeroFee":false,"statisticType":"FIXED","fixedStartTime":1760544000000,` +
			`"fixedEndTime":1763049600000,"minLeverage":1,"maxLeverage":50,"isMaxLeverage":true}`,
	}), liveReadOnly)
	result, err := ex.GetFuturesFeeDetails(t.Context(), futuresTradablePair)
	require.NoError(t, err, "GetFuturesFeeDetails must not error")
	if !mockTests {
		return
	}
	// The leverage fee rates are set only in a fee mode the venue gives no example of, so they are not decoded
	exp := &FuturesFeeDetailsResponse{
		Symbol:                  "BTC_USDT",
		OriginalMakerFee:        0.0001,
		OriginalTakerFee:        0.0004,
		JoinDiscount:            true,
		EnjoyDiscount:           true,
		JoinDeduct:              true,
		EnjoyDeduct:             true,
		RealMakerFee:            0.00005,
		RealTakerFee:            0.0002,
		DiscountRate:            0.5,
		DeductRate:              0.8,
		DealAmount:              1250.5,
		WalletBalance:           35.10272884,
		Level:                   9999,
		FeeType:                 1,
		FeeRateMode:             "TIERED",
		TieredDealAmount:        54,
		TieredAppointContract:   true,
		TieredExcludeContractID: true,
		FixedEndTime:            types.Time(time.UnixMilli(1763049600000)),
		IsMaxLeverage:           true,
		TieredFeeRates: []FuturesTieredFeeRate{
			{MaxTieredDealAmount: 10000000},
			{TakerFeeRate: 0.0004, MakerFeeRate: 0.0001, MinTieredDealAmount: 10000001, RealTakerFee: 0.0002, RealMakerFee: 0.00005},
		},
		TieredContractIDs: []uint64{10, 77, 1104},
		StatisticType:     "FIXED",
		FixedStartTime:    types.Time(time.UnixMilli(1760544000000)),
		MinLeverage:       1,
		MaxLeverage:       50,
	}
	assert.Equal(t, exp, result, "GetFuturesFeeDetails should decode every field")
}

func TestChangeFuturesPositionMargin(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, e.ChangeFuturesPositionMargin(t.Context(), nil), common.ErrNilPointer)
	require.ErrorIs(t, e.ChangeFuturesPositionMargin(t.Context(), &FuturesChangeMarginRequest{Amount: 1, Type: FuturesMarginAdd}), errPositionIDRequired)
	require.ErrorIs(t, e.ChangeFuturesPositionMargin(t.Context(), &FuturesChangeMarginRequest{PositionID: 1, Type: FuturesMarginAdd}), limits.ErrAmountBelowMin)
	require.ErrorIs(t, e.ChangeFuturesPositionMargin(t.Context(), &FuturesChangeMarginRequest{PositionID: 1, Amount: 1, Type: "add"}), errInvalidMarginChangeType)

	ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
		method: http.MethodPost,
		path:   "private/position/change_margin",
		body:   `{"positionId":1109973831,"amount":1.5,"type":"SUB"}`,
	}), liveUnavailable)
	require.NoError(t, ex.ChangeFuturesPositionMargin(t.Context(), &FuturesChangeMarginRequest{PositionID: 1109973831, Amount: 1.5, Type: FuturesMarginSubtract}),
		"ChangeFuturesPositionMargin must not error")
}

func TestSetFuturesAutoAddMargin(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, e.SetFuturesAutoAddMargin(t.Context(), 0, true), errPositionIDRequired)

	ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
		method: http.MethodPost,
		path:   "private/position/change_auto_add_im",
		body:   `{"positionId":1109973831,"isEnabled":true}`,
	}), liveUnavailable)
	require.NoError(t, ex.SetFuturesAutoAddMargin(t.Context(), 1109973831, true), "SetFuturesAutoAddMargin must not error")
}

func TestGetFuturesLeverage(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesLeverage(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
		method: http.MethodGet,
		path:   "private/position/leverage",
		query:  "symbol=BTC_USDT",
		data: `[{"level":6,"maxVol":4500000,"mmr":0.05,"imr":0.1,"positionType":1,"openType":1,"leverage":2,` +
			`"limitBySys":false,"currentMmr":0.001,"maxLeverageView":500},{"level":6,"maxVol":4500000,"mmr":0.05,` +
			`"imr":0.1,"positionType":2,"openType":1,"leverage":3,"limitBySys":false,"currentMmr":0.001,"maxLeverageView":500}]`,
	}), liveReadOnly)
	result, err := ex.GetFuturesLeverage(t.Context(), futuresTradablePair)
	require.NoError(t, err, "GetFuturesLeverage must not error")
	if !mockTests {
		return
	}
	exp := []*FuturesLeverageResponse{
		{
			PositionType:                 FuturesPositionTypeLong,
			OpenType:                     FuturesOpenTypeIsolated,
			Leverage:                     2,
			MaxLeverageView:              500,
			Level:                        6,
			MaxVolume:                    4500000,
			MaintenanceMarginRate:        0.05,
			InitialMarginRate:            0.1,
			CurrentMaintenanceMarginRate: 0.001,
		},
		{
			PositionType:                 FuturesPositionTypeShort,
			OpenType:                     FuturesOpenTypeIsolated,
			Leverage:                     3,
			MaxLeverageView:              500,
			Level:                        6,
			MaxVolume:                    4500000,
			MaintenanceMarginRate:        0.05,
			InitialMarginRate:            0.1,
			CurrentMaintenanceMarginRate: 0.001,
		},
	}
	assert.Equal(t, exp, result, "GetFuturesLeverage should decode both position sides")
}

func TestChangeFuturesLeverage(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, e.ChangeFuturesLeverage(t.Context(), nil), common.ErrNilPointer)
	require.ErrorIs(t, e.ChangeFuturesLeverage(t.Context(), &FuturesChangeLeverageRequest{PositionID: 1}), errInvalidLeverage)
	require.ErrorIs(t, e.ChangeFuturesLeverage(t.Context(), &FuturesChangeLeverageRequest{Leverage: 2, OpenType: FuturesOpenTypeIsolated, PositionType: FuturesPositionTypeLong}), currency.ErrSymbolStringEmpty)
	require.ErrorIs(t, e.ChangeFuturesLeverage(t.Context(), &FuturesChangeLeverageRequest{Leverage: 2, Symbol: "BTC_USDT", PositionType: FuturesPositionTypeLong}), errInvalidOpenType)
	require.ErrorIs(t, e.ChangeFuturesLeverage(t.Context(), &FuturesChangeLeverageRequest{Leverage: 2, Symbol: "BTC_USDT", OpenType: FuturesOpenTypeCross}), errInvalidPositionType)

	ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
		method: http.MethodPost,
		path:   "private/position/change_leverage",
		body:   `{"leverage":20,"openType":2,"symbol":"BTC_USDT","positionType":2}`,
	}), liveUnavailable)
	require.NoError(t, ex.ChangeFuturesLeverage(t.Context(), &FuturesChangeLeverageRequest{
		Leverage:     20,
		OpenType:     FuturesOpenTypeCross,
		Symbol:       "BTC_USDT",
		PositionType: FuturesPositionTypeShort,
	}), "ChangeFuturesLeverage must not error for a position side")

	ex = testExchangeFor(t, futuresHandler(t, futuresRoute{
		method: http.MethodPost,
		path:   "private/position/change_leverage",
		body:   `{"positionId":1109973831,"leverage":5}`,
	}), liveUnavailable)
	require.NoError(t, ex.ChangeFuturesLeverage(t.Context(), &FuturesChangeLeverageRequest{PositionID: 1109973831, Leverage: 5}),
		"ChangeFuturesLeverage must not error for an open position")
}

func TestGetFuturesPositionMode(t *testing.T) {
	t.Parallel()
	ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
		method: http.MethodGet,
		path:   "private/position/position_mode",
		data:   `2`,
	}), liveReadOnly)
	result, err := ex.GetFuturesPositionMode(t.Context())
	require.NoError(t, err, "GetFuturesPositionMode must not error")
	if !mockTests {
		assert.Contains(t, []uint8{FuturesPositionModeHedge, FuturesPositionModeOneWay}, result, "the position mode should be hedge or one-way")
		return
	}
	assert.EqualValues(t, FuturesPositionModeOneWay, result, "the position mode should be decoded")
}

func TestChangeFuturesPositionMode(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, e.ChangeFuturesPositionMode(t.Context(), 0), errInvalidPositionMode)

	ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
		method: http.MethodPost,
		path:   "private/position/change_position_mode",
		body:   `{"positionMode":1}`,
	}), liveUnavailable)
	require.NoError(t, ex.ChangeFuturesPositionMode(t.Context(), FuturesPositionModeHedge), "ChangeFuturesPositionMode must not error")
}

func TestChangeFuturesMultiAssetMode(t *testing.T) {
	t.Parallel()
	replay := futuresHandler(t, futuresRoute{method: http.MethodPost, path: "private/multiAssets/changeMultiAssetMode/true"})
	ex := testExchangeFor(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"), "the venue documents a form request")
		replay(w, r)
	}, liveUnavailable)
	require.NoError(t, ex.ChangeFuturesMultiAssetMode(t.Context(), true), "ChangeFuturesMultiAssetMode must not error")
}

func TestGetFuturesSelfTradePreventionGroups(t *testing.T) {
	t.Parallel()
	ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
		method: http.MethodGet,
		path:   "private/market_maker/self_trade/blacklist",
		query:  "configName=stpconfi001",
		data:   `[{"groupName":"stpconfi001","blackList":["45359845","72009483"],"createTime":1768804555000,"updateTime":1768804555000}]`,
	}), liveUnavailable)
	result, err := ex.GetFuturesSelfTradePreventionGroups(t.Context(), "stpconfi001")
	require.NoError(t, err, "GetFuturesSelfTradePreventionGroups must not error")
	exp := []*FuturesSelfTradePreventionGroupResponse{{
		GroupName:  "stpconfi001",
		BlackList:  []FuturesID{"45359845", "72009483"},
		CreateTime: types.Time(time.UnixMilli(1768804555000)),
		UpdateTime: types.Time(time.UnixMilli(1768804555000)),
	}}
	assert.Equal(t, exp, result, "GetFuturesSelfTradePreventionGroups should decode every field")
}

func TestGetFuturesSelfTradePreventionGroup(t *testing.T) {
	t.Parallel()
	ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
		method: http.MethodGet,
		path:   "private/market_maker/self_trade/blacklist/search",
		data:   `{"groupName":"stpconfi001","blackList":[45359845],"createTime":1768804555000}`,
	}), liveUnavailable)
	result, err := ex.GetFuturesSelfTradePreventionGroup(t.Context())
	require.NoError(t, err, "GetFuturesSelfTradePreventionGroup must not error")
	// The member is sent as a number
	exp := &FuturesSelfTradePreventionGroupResponse{GroupName: "stpconfi001", BlackList: []FuturesID{"45359845"}, CreateTime: types.Time(time.UnixMilli(1768804555000))}
	assert.Equal(t, exp, result, "GetFuturesSelfTradePreventionGroup should decode every field")
}

func TestCreateFuturesSelfTradePreventionGroup(t *testing.T) {
	t.Parallel()
	_, err := e.CreateFuturesSelfTradePreventionGroup(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.CreateFuturesSelfTradePreventionGroup(t.Context(), &FuturesSelfTradePreventionGroupRequest{Blacklist: []uint64{1, 2}})
	require.ErrorIs(t, err, errSelfTradePreventionGroupNameRequired)
	_, err = e.CreateFuturesSelfTradePreventionGroup(t.Context(), &FuturesSelfTradePreventionGroupRequest{ConfigName: "stp"})
	require.ErrorIs(t, err, errUIDRequired)
	_, err = e.CreateFuturesSelfTradePreventionGroup(t.Context(), &FuturesSelfTradePreventionGroupRequest{ConfigName: "stp", Blacklist: []uint64{1}})
	require.ErrorIs(t, err, errSelfTradePreventionUIDsShort)

	ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
		method: http.MethodPost,
		path:   "private/market_maker/self_trade/blacklist/create",
		body:   `{"configName":"stpconfi001","blacklist":[45359845,72009483]}`,
		data:   `1`,
	}), liveUnavailable)
	result, err := ex.CreateFuturesSelfTradePreventionGroup(t.Context(), &FuturesSelfTradePreventionGroupRequest{ConfigName: "stpconfi001", Blacklist: []uint64{45359845, 72009483}})
	require.NoError(t, err, "CreateFuturesSelfTradePreventionGroup must not error")
	assert.EqualValues(t, 1, result, "the group's ID should be returned")
}

func TestUpdateFuturesSelfTradePreventionGroup(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, e.UpdateFuturesSelfTradePreventionGroup(t.Context(), &FuturesSelfTradePreventionGroupRequest{ConfigName: "stp"}), errUIDRequired)

	ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
		method: http.MethodPost,
		path:   "private/market_maker/self_trade/blacklist/update",
		body:   `{"configName":"stpconfi001","blacklist":[45359845]}`,
	}), liveUnavailable)
	require.NoError(t, ex.UpdateFuturesSelfTradePreventionGroup(t.Context(), &FuturesSelfTradePreventionGroupRequest{ConfigName: "stpconfi001", Blacklist: []uint64{45359845}}),
		"UpdateFuturesSelfTradePreventionGroup must not error")
}

func TestDeleteFuturesSelfTradePreventionGroup(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, e.DeleteFuturesSelfTradePreventionGroup(t.Context(), ""), errSelfTradePreventionGroupNameRequired)

	ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
		method: http.MethodPost,
		path:   "private/market_maker/self_trade/blacklist/delete",
		body:   `{"configName":"stpconfi001"}`,
	}), liveUnavailable)
	require.NoError(t, ex.DeleteFuturesSelfTradePreventionGroup(t.Context(), "stpconfi001"), "DeleteFuturesSelfTradePreventionGroup must not error")
}
