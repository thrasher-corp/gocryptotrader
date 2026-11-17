package bingx

import (
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestSetCopyTradingPerpetualTakeProfitStopLoss(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *CopyTradingTakeProfitStopLossRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "missing position ID", arg: &CopyTradingTakeProfitStopLossRequest{StopLossMarkPrice: 105.38}, err: errCopyTradingPositionIDRequired},
		{name: "missing mark prices", arg: &CopyTradingTakeProfitStopLossRequest{PositionID: 1253517936071234567}, err: errCopyTradingTakeProfitStopLossRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.SetCopyTradingPerpetualTakeProfitStopLoss(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "SetCopyTradingPerpetualTakeProfitStopLoss should reject the request")
		})
	}
	t.Run("take profit only", func(t *testing.T) {
		t.Parallel()
		ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method, "method should be POST")
			verifyQuerySignature(t, r.URL.RawQuery)
			q := r.URL.Query()
			q.Del("timestamp")
			q.Del("signature")
			exp := url.Values{"positionId": {"1253517936071234567"}, "takeProfitMarkPrice": {"120.5"}, "recvWindow": {"5000"}}
			assert.Equal(t, exp, q, "only the take profit mark price should be sent")
			_, _ = io.WriteString(w, `{"code":0,"data":{"positionId":1253517936071234567,"takeProfitMarkPrice":120.5}}`)
		})
		result, err := ex.SetCopyTradingPerpetualTakeProfitStopLoss(t.Context(), &CopyTradingTakeProfitStopLossRequest{PositionID: 1253517936071234567, TakeProfitMarkPrice: 120.5})
		require.NoError(t, err, "SetCopyTradingPerpetualTakeProfitStopLoss must not error")
		exp := &CopyTradingTakeProfitStopLossResponse{PositionID: 1253517936071234567, TakeProfitMarkPrice: 120.5}
		assert.Equal(t, exp, result, "SetCopyTradingPerpetualTakeProfitStopLoss should decode the reply")
	})
	t.Run("valid request", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		if !mockTests {
			sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)
		}
		result, err := e.SetCopyTradingPerpetualTakeProfitStopLoss(t.Context(), &CopyTradingTakeProfitStopLossRequest{
			PositionID:        1253517936071234567,
			StopLossMarkPrice: 105.38,
		})
		require.NoError(t, err, "SetCopyTradingPerpetualTakeProfitStopLoss must not error")
		require.NotNil(t, result, "SetCopyTradingPerpetualTakeProfitStopLoss must return a result")
		if !mockTests {
			return
		}
		exp := &CopyTradingTakeProfitStopLossResponse{PositionID: 1253517936071234600, StopLossMarkPrice: 105.38}
		assert.Equal(t, exp, result, "SetCopyTradingPerpetualTakeProfitStopLoss should decode the documented example")
	})
}

func TestGetCopyTradingPerpetualTraderDetail(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetCopyTradingPerpetualTraderDetail(t.Context(), 7)
	require.NoError(t, err, "GetCopyTradingPerpetualTraderDetail must not error")
	require.NotNil(t, result, "GetCopyTradingPerpetualTraderDetail must return a result")
	if !mockTests {
		return
	}
	exp := &CopyTradingPerpetualTraderDetailResponse{
		ContractType:   CopyTradingContractTypePerpetualFutures,
		TraderID:       "123456",
		Status:         true,
		AccountLevel:   CopyTradingAccountLevelGold,
		Ratio:          5.32,
		AccountAssets:  10000.25,
		CopierProfit:   125.67,
		CopiersNumber:  154,
		RiskLevel:      3,
		WinRatio:       68.2,
		TradeNumber:    325,
		WinNumber:      221,
		LoseNumber:     104,
		AverageProfit:  75.25,
		AverageLosses:  45.12,
		PNLRatio:       1.67,
		AverageTime:    3.8,
		MarginMedian:   250,
		LeverageMedian: 10,
		TradeFrequency: 4.2,
		TradeDays:      85,
		LastTrade:      types.Time(time.UnixMilli(1711619555065)),
	}
	assert.Equal(t, exp, result, "GetCopyTradingPerpetualTraderDetail should decode the documented example")
}

func TestGetCopyTradingPerpetualProfitSummary(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetCopyTradingPerpetualProfitSummary(t.Context())
	require.NoError(t, err, "GetCopyTradingPerpetualProfitSummary must not error")
	require.NotNil(t, result, "GetCopyTradingPerpetualProfitSummary must return a result")
	if !mockTests {
		return
	}
	exp := &CopyTradingPerpetualProfitSummaryResponse{
		ContractType: CopyTradingContractTypePerpetualFutures,
		Earn:         158.32,
		Settled:      10253.55,
		ProfitRatio:  12.5,
	}
	assert.Equal(t, exp, result, "GetCopyTradingPerpetualProfitSummary should decode the documented example")
}

func TestGetCopyTradingPerpetualProfitDetail(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *CopyTradingProfitDetailRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "missing page index", arg: &CopyTradingProfitDetailRequest{PageSize: 20}, err: errPageIndexRequired},
		{name: "missing page size", arg: &CopyTradingProfitDetailRequest{PageIndex: 1}, err: errPageSizeRequired},
		{name: "reversed time range", arg: &CopyTradingProfitDetailRequest{StartTime: time.UnixMilli(2000), EndTime: time.UnixMilli(1000), PageIndex: 1, PageSize: 20}, err: common.ErrStartAfterEnd},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetCopyTradingPerpetualProfitDetail(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "GetCopyTradingPerpetualProfitDetail should reject the request")
		})
	}
	t.Run("valid request", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		result, err := e.GetCopyTradingPerpetualProfitDetail(t.Context(), &CopyTradingProfitDetailRequest{
			StartTime: time.UnixMilli(1711000000000),
			EndTime:   time.UnixMilli(1711619555000),
			PageIndex: 1,
			PageSize:  20,
		})
		require.NoError(t, err, "GetCopyTradingPerpetualProfitDetail must not error")
		if !mockTests {
			return
		}
		exp := []CopyTradingPerpetualProfitDetailResponse{
			{
				ContractType:   CopyTradingContractTypePerpetualFutures,
				CommissionTime: types.Time(time.Unix(1711600000, 0)),
				CopiersUID:     987654321,
				ProfitShare:    12.35,
			},
			{
				ContractType:   CopyTradingContractTypePerpetualFutures,
				CommissionTime: types.Time(time.Unix(1711500000, 0)),
				CopiersUID:     123456789,
				ProfitShare:    8.77,
			},
		}
		assert.Equal(t, exp, result, "GetCopyTradingPerpetualProfitDetail should decode the documented example")
	})
}

func TestSetCopyTradingPerpetualCommissionRate(t *testing.T) {
	t.Parallel()
	t.Run("missing commission rate", func(t *testing.T) {
		t.Parallel()
		_, err := e.SetCopyTradingPerpetualCommissionRate(t.Context(), 0)
		assert.ErrorIs(t, err, errCopyTradingCommissionRateRequired, "SetCopyTradingPerpetualCommissionRate should reject a missing rate")
	})
	t.Run("valid request", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		if !mockTests {
			sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)
		}
		result, err := e.SetCopyTradingPerpetualCommissionRate(t.Context(), 16)
		require.NoError(t, err, "SetCopyTradingPerpetualCommissionRate must not error")
		require.NotNil(t, result, "SetCopyTradingPerpetualCommissionRate must return a result")
		if !mockTests {
			return
		}
		exp := &CopyTradingPerpetualCommissionRateResponse{NewCommission: "16%"}
		assert.Equal(t, exp, result, "SetCopyTradingPerpetualCommissionRate should decode the documented example")
	})
}

func TestGetCopyTradingPerpetualTradingPairs(t *testing.T) {
	t.Parallel()
	t.Run("missing contract type", func(t *testing.T) {
		t.Parallel()
		_, err := e.GetCopyTradingPerpetualTradingPairs(t.Context(), "")
		assert.ErrorIs(t, err, errCopyTradingContractTypeRequired, "GetCopyTradingPerpetualTradingPairs should reject a missing contract type")
	})
	t.Run("valid request", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		result, err := e.GetCopyTradingPerpetualTradingPairs(t.Context(), CopyTradingContractTypePerpetualFutures)
		require.NoError(t, err, "GetCopyTradingPerpetualTradingPairs must not error")
		if !mockTests {
			return
		}
		exp := []CopyTradingPerpetualTradingPairResponse{{
			ContractType:     CopyTradingContractTypePerpetualFutures,
			Symbol:           "BTCUSDT",
			OpenTrader:       CopyTradingOpenTraderYes,
			MinOpenCount:     1,
			MaxLeverage:      125,
			StopSurplusRatio: 50,
			StopLossRatio:    30,
		}}
		assert.Equal(t, exp, result, "GetCopyTradingPerpetualTradingPairs should decode the documented example")
	})
}

func TestSellCopyTradingSpotOrder(t *testing.T) {
	t.Parallel()
	t.Run("missing order ID", func(t *testing.T) {
		t.Parallel()
		_, err := e.SellCopyTradingSpotOrder(t.Context(), 0)
		assert.ErrorIs(t, err, order.ErrOrderIDNotSet, "SellCopyTradingSpotOrder should reject a missing order ID")
	})
	t.Run("valid request", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		if !mockTests {
			sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)
		}
		result, err := e.SellCopyTradingSpotOrder(t.Context(), 1253517936071234567)
		require.NoError(t, err, "SellCopyTradingSpotOrder must not error")
		require.NotNil(t, result, "SellCopyTradingSpotOrder must return a result")
		if !mockTests {
			return
		}
		exp := &CopyTradingSpotSellOrderResponse{
			OrderNumber:       1773285851363541000,
			CoinName:          currency.BTC,
			ValuationCoinName: currency.USDT,
			Side:              2,
			Price:             70850,
			DealPrice:         70612.7438655,
			DelegatePrice:     70613.45,
			Status:            CopyTradingSpotOrderStatusSuccess,
		}
		assert.Equal(t, exp, result, "SellCopyTradingSpotOrder should decode the documented example")
	})
}

func TestGetCopyTradingSpotTraderDetail(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetCopyTradingSpotTraderDetail(t.Context())
	require.NoError(t, err, "GetCopyTradingSpotTraderDetail must not error")
	require.NotNil(t, result, "GetCopyTradingSpotTraderDetail must return a result")
	if !mockTests {
		return
	}
	exp := &CopyTradingSpotTraderDetailResponse{
		TraderID:       "123456",
		Status:         true,
		Ratio:          12.5,
		AccountAssets:  10500,
		PNL:            500,
		CopierProfit:   150,
		CopiersNumber:  10,
		WinRatio:       0.65,
		TradeNumber:    20,
		TradeFrequency: 4,
		TradeDays:      5,
		TradingPairs:   []string{"BTCUSDT", "ETHUSDT"},
		LastTrade:      types.Time(time.UnixMilli(1711619555065)),
	}
	assert.Equal(t, exp, result, "GetCopyTradingSpotTraderDetail should decode the documented example")
}

func TestGetCopyTradingSpotProfitSummary(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetCopyTradingSpotProfitSummary(t.Context())
	require.NoError(t, err, "GetCopyTradingSpotProfitSummary must not error")
	require.NotNil(t, result, "GetCopyTradingSpotProfitSummary must return a result")
	if !mockTests {
		return
	}
	assert.Equal(t, &CopyTradingSpotProfitSummaryResponse{Earn: 100, Settled: 500, ProfitRatio: 0.2}, result, "GetCopyTradingSpotProfitSummary should decode the documented example")
}

func TestGetCopyTradingSpotProfitDetail(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *CopyTradingProfitDetailRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "missing page index", arg: &CopyTradingProfitDetailRequest{PageSize: 20}, err: errPageIndexRequired},
		{name: "missing page size", arg: &CopyTradingProfitDetailRequest{PageIndex: 1}, err: errPageSizeRequired},
		{name: "reversed time range", arg: &CopyTradingProfitDetailRequest{StartTime: time.UnixMilli(2000), EndTime: time.UnixMilli(1000), PageIndex: 1, PageSize: 20}, err: common.ErrStartAfterEnd},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetCopyTradingSpotProfitDetail(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "GetCopyTradingSpotProfitDetail should reject the request")
		})
	}
	t.Run("valid request", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		result, err := e.GetCopyTradingSpotProfitDetail(t.Context(), &CopyTradingProfitDetailRequest{PageIndex: 1, PageSize: 20})
		require.NoError(t, err, "GetCopyTradingSpotProfitDetail must not error")
		if !mockTests {
			return
		}
		exp := []CopyTradingSpotProfitDetailResponse{{
			CommissionTime: types.Time(time.Unix(1711619555, 0)),
			CopiersUID:     123456,
			ProfitShare:    10,
		}}
		assert.Equal(t, exp, result, "GetCopyTradingSpotProfitDetail should decode the documented example")
	})
}

func TestGetCopyTradingSpotOrderHistory(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *CopyTradingSpotOrderHistoryRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "missing page index", arg: &CopyTradingSpotOrderHistoryRequest{PageSize: 50}, err: errPageIndexRequired},
		{name: "missing page size", arg: &CopyTradingSpotOrderHistoryRequest{PageIndex: 1}, err: errPageSizeRequired},
		{name: "reversed time range", arg: &CopyTradingSpotOrderHistoryRequest{StartTime: time.UnixMilli(2000), EndTime: time.UnixMilli(1000), PageIndex: 1, PageSize: 50}, err: common.ErrStartAfterEnd},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetCopyTradingSpotOrderHistory(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "GetCopyTradingSpotOrderHistory should reject the request")
		})
	}
	t.Run("symbol formatting error", func(t *testing.T) {
		t.Parallel()
		_, err := new(Exchange).GetCopyTradingSpotOrderHistory(t.Context(), &CopyTradingSpotOrderHistoryRequest{Symbol: currency.NewBTCUSDT(), PageIndex: 1, PageSize: 50})
		assert.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "GetCopyTradingSpotOrderHistory should return the symbol formatting error")
	})
	t.Run("valid request", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		result, err := e.GetCopyTradingSpotOrderHistory(t.Context(), &CopyTradingSpotOrderHistoryRequest{
			Symbol:    currency.NewBTCUSDT(),
			PageIndex: 1,
			PageSize:  50,
		})
		require.NoError(t, err, "GetCopyTradingSpotOrderHistory must not error")
		if !mockTests {
			return
		}
		exp := []CopyTradingSpotOrderHistoryResponse{{
			OpenOrderID:  123456,
			CloseOrderID: 123457,
			Symbol:       "BTCUSDT",
			Quantity:     0.5,
			Amount:       35000,
			Profit:       500,
			OpenPrice:    70000,
			CloseTime:    types.Time(time.UnixMilli(1711619555065)),
			ClosePrice:   71000,
		}}
		assert.Equal(t, exp, result, "GetCopyTradingSpotOrderHistory should decode the documented example")
	})
}

func TestCopyTradingPageParams(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name                string
		start, end          time.Time
		pageIndex, pageSize uint64
		err                 error
		exp                 url.Values
	}{
		{name: "missing page index", pageSize: 20, err: errPageIndexRequired},
		{name: "missing page size", pageIndex: 1, err: errPageSizeRequired},
		{name: "reversed time range", start: time.UnixMilli(2000), end: time.UnixMilli(1000), pageIndex: 1, pageSize: 20, err: common.ErrStartAfterEnd},
		{name: "pages only", pageIndex: 1, pageSize: 20, exp: url.Values{"pageIndex": {"1"}, "pageSize": {"20"}}},
		{name: "time range and pages", start: time.UnixMilli(1000), end: time.UnixMilli(2000), pageIndex: 3, pageSize: 100, exp: url.Values{"startTime": {"1000"}, "endTime": {"2000"}, "pageIndex": {"3"}, "pageSize": {"100"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			params, err := copyTradingPageParams(tc.start, tc.end, tc.pageIndex, tc.pageSize)
			if tc.err != nil {
				assert.ErrorIs(t, err, tc.err, "copyTradingPageParams should return the expected error")
				return
			}
			require.NoError(t, err, "copyTradingPageParams must not error")
			assert.Equal(t, tc.exp, params, "copyTradingPageParams should return the parameters")
		})
	}
}
