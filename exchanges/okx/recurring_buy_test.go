package okx

import (
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

func TestPlaceRecurringBuyOrder(t *testing.T) {
	t.Parallel()
	monthly := func() *PlaceRecurringBuyOrderRequest {
		return &PlaceRecurringBuyOrderRequest{
			StrategyName: "BTC|ETH recurring buy monthly", RecurringList: []RecurringListItem{{Currency: currency.BTC, Ratio: 0.2}, {Currency: currency.ETH, Ratio: 0.8}},
			Period: "monthly", RecurringDay: 1, RecurringTime: 2, TimeZone: 3, Amount: 100, InvestmentCurrency: currency.USDT, TradeMode: TradeModeCross,
		}
	}
	for _, tc := range []struct {
		name string
		arg  func() *PlaceRecurringBuyOrderRequest
		err  error
	}{
		{"nil request", func() *PlaceRecurringBuyOrderRequest { return nil }, common.ErrNilPointer},
		{"missing strategy name", func() *PlaceRecurringBuyOrderRequest { return &PlaceRecurringBuyOrderRequest{TimeZone: 3} }, errStrategyNameRequired},
		{"missing recurring list", func() *PlaceRecurringBuyOrderRequest { a := monthly(); a.RecurringList = nil; return a }, errRecurringListRequired},
		{"recurring currency without code", func() *PlaceRecurringBuyOrderRequest {
			a := monthly()
			a.RecurringList = []RecurringListItem{{Ratio: 1}}
			return a
		}, currency.ErrCurrencyCodeEmpty},
		{"recurring currency without ratio", func() *PlaceRecurringBuyOrderRequest {
			a := monthly()
			a.RecurringList = []RecurringListItem{{Currency: currency.BTC}}
			return a
		}, errRecurringRatioRequired},
		{"missing period", func() *PlaceRecurringBuyOrderRequest { a := monthly(); a.Period = ""; return a }, errRecurringPeriodRequired},
		{"monthly without recurring day", func() *PlaceRecurringBuyOrderRequest { a := monthly(); a.RecurringDay = 0; return a }, errRecurringDayRequired},
		{"weekly without recurring day", func() *PlaceRecurringBuyOrderRequest {
			a := monthly()
			a.Period = "weekly"
			a.RecurringDay = 0
			return a
		}, errRecurringDayRequired},
		{"hourly without recurring hour", func() *PlaceRecurringBuyOrderRequest { a := monthly(); a.Period = "hourly"; return a }, errRecurringHourRequired},
		{"recurring time above 23", func() *PlaceRecurringBuyOrderRequest { a := monthly(); a.RecurringTime = 24; return a }, errRecurringBuyTimeRequired},
		{"time zone below -12", func() *PlaceRecurringBuyOrderRequest { a := monthly(); a.TimeZone = -13; return a }, errInvalidTimeZone},
		{"time zone above 14", func() *PlaceRecurringBuyOrderRequest { a := monthly(); a.TimeZone = 15; return a }, errInvalidTimeZone},
		{"missing amount", func() *PlaceRecurringBuyOrderRequest { a := monthly(); a.Amount = 0; return a }, order.ErrAmountMustBeSet},
		{"missing investment currency", func() *PlaceRecurringBuyOrderRequest {
			a := monthly()
			a.InvestmentCurrency = currency.EMPTYCODE
			return a
		}, currency.ErrCurrencyCodeEmpty},
		{"missing trade mode", func() *PlaceRecurringBuyOrderRequest { a := monthly(); a.TradeMode = ""; return a }, errInvalidTradeModeValue},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.PlaceRecurringBuyOrder(t.Context(), tc.arg())
			assert.ErrorIs(t, err, tc.err, "PlaceRecurringBuyOrder should return the expected error")
		})
	}

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			arg  *PlaceRecurringBuyOrderRequest
			body string
		}{
			{
				name: "every parameter",
				arg: &PlaceRecurringBuyOrderRequest{
					StrategyName: "hourly BTC", RecurringList: []RecurringListItem{{Currency: currency.BTC, Ratio: 1, MinimumPrice: 30000, MaximumPrice: 50000}},
					Period: "hourly", RecurringHour: 4, RecurringTime: 23, TimeZone: -5, Amount: 20, InvestmentCurrency: currency.USDC, TradeMode: TradeModeCash,
					AlgoClientOrderID: "recurringClient1", Tag: "recurringTag1", TradeQuoteCurrency: currency.USDT, Source: []string{"1", "2"}, RecurringTimeType: "2",
				},
				body: `{"stgyName":"hourly BTC","recurringList":[{"ccy":"BTC","ratio":"1","minPx":"30000","maxPx":"50000"}],"period":"hourly","recurringHour":"4","recurringTime":"23","timeZone":"-5","amt":"20","investmentCcy":"USDC","tdMode":"cash","algoClOrdId":"recurringClient1","tag":"recurringTag1","tradeQuoteCcy":"USDT","source":["1","2"],"recurringTimeType":"2"}`,
			},
			{
				name: "daily with required parameters only",
				arg: &PlaceRecurringBuyOrderRequest{
					StrategyName: "daily BTC", RecurringList: []RecurringListItem{{Currency: currency.BTC, Ratio: 1}}, Period: "daily", Amount: 20,
					InvestmentCurrency: currency.USDT, TradeMode: TradeModeCash,
				},
				body: `{"stgyName":"daily BTC","recurringList":[{"ccy":"BTC","ratio":"1"}],"period":"daily","recurringTime":"0","timeZone":"0","amt":"20","investmentCcy":"USDT","tdMode":"cash"}`,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				_, err := newRequestPinExchange(t, "/tradingBot/recurring/order-algo", tc.body, tradingBotAlgoIDData).PlaceRecurringBuyOrder(t.Context(), tc.arg)
				require.NoError(t, err, "PlaceRecurringBuyOrder must not error")
			})
		}
	})

	t.Run("place order", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		}
		result, err := e.PlaceRecurringBuyOrder(contextGenerate(), &PlaceRecurringBuyOrderRequest{
			StrategyName:       "BTC|ETH recurring buy monthly",
			Amount:             100,
			RecurringList:      []RecurringListItem{{Currency: currency.BTC, Ratio: 0.2}, {Currency: currency.ETH, Ratio: 0.8}},
			Period:             "monthly",
			RecurringDay:       1,
			RecurringTime:      0,
			TimeZone:           8,
			TradeMode:          TradeModeCross,
			InvestmentCurrency: currency.USDT,
		})
		require.NoError(t, err, "PlaceRecurringBuyOrder must not error")
		if mockTests {
			assert.Equal(t, &TradingBotOrderResponse{AlgoID: "560472804207104000"}, result, "PlaceRecurringBuyOrder should decode every field")
		} else {
			assert.NotEmpty(t, result.AlgoID, "PlaceRecurringBuyOrder should return an algo ID")
		}
	})
}

func TestValidateRecurringBuySchedule(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name                        string
		period                      string
		recurringDay, recurringHour uint64
		err                         error
	}{
		{name: "missing period", recurringDay: 1, recurringHour: 4, err: errRecurringPeriodRequired},
		{name: "monthly with a day", period: recurringBuyPeriodMonthly, recurringDay: 1},
		{name: "monthly without a day", period: recurringBuyPeriodMonthly, recurringHour: 4, err: errRecurringDayRequired},
		{name: "weekly with a day", period: recurringBuyPeriodWeekly, recurringDay: 3},
		{name: "weekly without a day", period: recurringBuyPeriodWeekly, recurringHour: 4, err: errRecurringDayRequired},
		{name: "hourly with an hour", period: recurringBuyPeriodHourly, recurringHour: 4},
		{name: "hourly without an hour", period: recurringBuyPeriodHourly, recurringDay: 1, err: errRecurringHourRequired},
		{name: "daily needs neither", period: "daily"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIs(t, validateRecurringBuySchedule(tc.period, tc.recurringDay, tc.recurringHour), tc.err, "validateRecurringBuySchedule should return the expected error")
		})
	}
}

func TestAmendRecurringBuyOrder(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *AmendRecurringBuyOrderRequest
		err  error
	}{
		{"nil request", nil, common.ErrNilPointer},
		{"empty request", &AmendRecurringBuyOrderRequest{}, errAlgoIDRequired},
		{"missing algo ID", &AmendRecurringBuyOrderRequest{StrategyName: "stg1"}, errAlgoIDRequired},
		{"missing strategy name", &AmendRecurringBuyOrderRequest{AlgoID: "448965992920907776"}, errStrategyNameRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.AmendRecurringBuyOrder(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "AmendRecurringBuyOrder should return the expected error")
		})
	}

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		ex := newRequestPinExchange(t, "/tradingBot/recurring/amend-order-algo", `{"algoId":"448965992920907776","stgyName":"stg2"}`, tradingBotAlgoIDData)
		_, err := ex.AmendRecurringBuyOrder(t.Context(), &AmendRecurringBuyOrderRequest{AlgoID: "448965992920907776", StrategyName: "stg2"})
		require.NoError(t, err, "AmendRecurringBuyOrder must not error")
	})

	t.Run("amend order", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		}
		result, err := e.AmendRecurringBuyOrder(contextGenerate(), &AmendRecurringBuyOrderRequest{AlgoID: "448965992920907776", StrategyName: "stg1"})
		require.NoError(t, err, "AmendRecurringBuyOrder must not error")
		if mockTests {
			assert.Equal(t, &TradingBotOrderResponse{AlgoID: "448965992920907776"}, result, "AmendRecurringBuyOrder should decode every field")
		} else {
			assert.NotEmpty(t, result.AlgoID, "AmendRecurringBuyOrder should return an algo ID")
		}
	})
}

func TestStopRecurringBuyOrder(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  []string
		err  error
	}{
		{"no algo IDs", []string{}, common.ErrEmptyParams},
		{"more than 10 algo IDs", make([]string, 11), errExceedLimit},
		{"empty algo ID", []string{""}, errAlgoIDRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.StopRecurringBuyOrder(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "StopRecurringBuyOrder should return the expected error")
		})
	}

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		ex := newRequestPinExchange(t, "/tradingBot/recurring/stop-order-algo", `[{"algoId":"1232323434234"},{"algoId":"560472804207104000"}]`, tradingBotAlgoIDData)
		_, err := ex.StopRecurringBuyOrder(t.Context(), []string{"1232323434234", "560472804207104000"})
		require.NoError(t, err, "StopRecurringBuyOrder must not error")
	})

	t.Run("stop order", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		}
		result, err := e.StopRecurringBuyOrder(contextGenerate(), []string{"1232323434234"})
		require.NoError(t, err, "StopRecurringBuyOrder must not error")
		if mockTests {
			assert.Equal(t, []TradingBotOrderResponse{{AlgoID: "1232323434234"}}, result, "StopRecurringBuyOrder should decode every field")
		} else {
			assert.NotEmpty(t, result, "StopRecurringBuyOrder should return the stopped orders")
		}
	})
}

func TestGetRecurringBuyOrderList(t *testing.T) {
	t.Parallel()
	_, err := e.GetRecurringBuyOrderList(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetRecurringBuyOrderList(contextGenerate(), &RecurringBuyOrderListRequest{Limit: 30})
	require.NoError(t, err, "GetRecurringBuyOrderList must not error")
	if mockTests {
		exp := []RecurringBuyOrderResponse{
			{
				AlgoID:              "644497312047435776",
				AlgoClientOrderID:   "recurringClient1",
				InstrumentType:      "SPOT",
				CreationTime:        types.Time(time.UnixMilli(1699932133373)),
				UpdateTime:          types.Time(time.UnixMilli(1699952473152)),
				AlgoOrderType:       "recurring",
				State:               "running",
				StrategyName:        "stg1",
				RecurringList:       []RecurringListItem{{Currency: currency.BTC, Ratio: 0.2, MinimumPrice: 30000, MaximumPrice: 50000}, {Currency: currency.ETH, Ratio: 0.8}},
				Period:              "hourly",
				RecurringHour:       1,
				RecurringTime:       12,
				TimeZone:            8,
				Amount:              100,
				InvestmentAmount:    600,
				InvestmentCurrency:  currency.USDC,
				TotalPNL:            12.5,
				TotalAnnualisedRate: 0.85,
				PNLRatio:            0.0208,
				MarketCap:           612.5,
				Cycles:              6,
				Tag:                 "recurringTag1",
				TradeQuoteCurrency:  currency.USDT,
				Source:              []string{"1"},
				RecurringTimeType:   "1",
			},
		}
		assert.Equal(t, exp, result, "GetRecurringBuyOrderList should decode every field")
	} else {
		assert.NotNil(t, result, "GetRecurringBuyOrderList should return a list")
	}
}

func TestGetRecurringBuyOrderHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetRecurringBuyOrderHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetRecurringBuyOrderHistory(contextGenerate(), &RecurringBuyOrderListRequest{Limit: 30})
	require.NoError(t, err, "GetRecurringBuyOrderHistory must not error")
	if mockTests {
		exp := []RecurringBuyOrderResponse{
			{
				AlgoID:               "644496098429767680",
				InstrumentType:       "SPOT",
				CreationTime:         types.Time(time.UnixMilli(1699931844050)),
				UpdateTime:           types.Time(time.UnixMilli(1699932177659)),
				AlgoOrderType:        "recurring",
				State:                "stopped",
				StrategyName:         "stg1",
				RecurringList:        []RecurringListItem{{Currency: currency.BTC, Ratio: 0.2}, {Currency: currency.ETH, Ratio: 0.8}},
				Period:               "monthly",
				RecurringDay:         15,
				TimeZone:             -5,
				Amount:               100,
				InvestmentAmount:     200,
				InvestmentCurrency:   currency.USDC,
				TotalPNL:             -4.6,
				TotalAnnualisedRate:  -0.12,
				PNLRatio:             -0.023,
				MarketCap:            195.4,
				Cycles:               2,
				TradeQuoteCurrency:   currency.USDT,
				Source:               []string{"1", "2"},
				RecurringTimeType:    "1",
				RecurringTimeMinutes: 30,
			},
		}
		assert.Equal(t, exp, result, "GetRecurringBuyOrderHistory should decode every field")
	} else {
		assert.NotNil(t, result, "GetRecurringBuyOrderHistory should return a list")
	}
}

func TestRecurringBuyOrderListParams(t *testing.T) {
	t.Parallel()
	_, err := recurringBuyOrderListParams(nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	params, err := recurringBuyOrderListParams(&RecurringBuyOrderListRequest{})
	require.NoError(t, err, "recurringBuyOrderListParams must not error")
	assert.Empty(t, params, "recurringBuyOrderListParams should set no parameters for an empty request")

	params, err = recurringBuyOrderListParams(&RecurringBuyOrderListRequest{AlgoID: "644497312047435776", After: "644497312047435776", Before: "644497312047435700", Limit: 100})
	require.NoError(t, err, "recurringBuyOrderListParams must not error")
	exp := url.Values{"algoId": {"644497312047435776"}, "after": {"644497312047435776"}, "before": {"644497312047435700"}, "limit": {"100"}}
	assert.Equal(t, exp, params, "recurringBuyOrderListParams should set every parameter")
}

func TestGetRecurringBuyOrderDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetRecurringBuyOrderDetails(t.Context(), "")
	require.ErrorIs(t, err, errAlgoIDRequired)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetRecurringBuyOrderDetails(contextGenerate(), "560473220642766848")
	require.NoError(t, err, "GetRecurringBuyOrderDetails must not error")
	if mockTests {
		exp := &RecurringOrderDetailResponse{
			AlgoID:              "560473220642766848",
			InstrumentType:      "SPOT",
			CreationTime:        types.Time(time.UnixMilli(1699932133373)),
			UpdateTime:          types.Time(time.UnixMilli(1699952485451)),
			AlgoOrderType:       "recurring",
			State:               "running",
			StrategyName:        "stg1",
			Period:              "hourly",
			RecurringHour:       1,
			RecurringTime:       12,
			TimeZone:            8,
			Amount:              100,
			InvestmentAmount:    600,
			InvestmentCurrency:  currency.USDC,
			TotalPNL:            12.5,
			TotalAnnualisedRate: 0.85,
			PNLRatio:            0.0208,
			MarketCap:           612.5,
			Cycles:              6,
			TradeQuoteCurrency:  currency.USDT,
			Source:              []string{"1"},
			RecurringTimeType:   "1",
			RecurringList: []RecurringListItemDetailed{
				{AveragePrice: 36500.1, Currency: currency.BTC, MinimumPrice: 30000, MaximumPrice: 50000, Profit: 0.6, Price: 36683.2, Ratio: 0.2, TotalAmount: 0.0033},
				{AveragePrice: 2050.4, Currency: currency.ETH, Profit: 1.8, Price: 2058.36, Ratio: 0.8, TotalAmount: 0.2341},
			},
			NextInvestTime: types.Time(time.UnixMilli(1699956005500)),
		}
		assert.Equal(t, exp, result, "GetRecurringBuyOrderDetails should decode every field")
	} else {
		assert.NotEmpty(t, result.AlgoID, "GetRecurringBuyOrderDetails should return an algo ID")
	}
}

func TestGetRecurringBuySubOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetRecurringBuySubOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.GetRecurringBuySubOrders(t.Context(), &RecurringBuySubOrdersRequest{OrderID: "123422"})
	require.ErrorIs(t, err, errAlgoIDRequired)

	t.Run("query with every parameter", func(t *testing.T) {
		t.Parallel()
		exp := url.Values{"algoId": {"560473220642766848"}, "ordId": {"123422"}, "after": {"560523524230717440"}, "before": {"560523524184580096"}, "limit": {"10"}}
		ex := newQueryPinExchange(t, "/tradingBot/recurring/sub-orders", exp, tradingBotAlgoIDData)
		_, err := ex.GetRecurringBuySubOrders(t.Context(), &RecurringBuySubOrdersRequest{AlgoID: "560473220642766848", OrderID: "123422", After: "560523524230717440", Before: "560523524184580096", Limit: 10})
		require.NoError(t, err, "GetRecurringBuySubOrders must not error")
	})

	t.Run("sub orders", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		}
		result, err := e.GetRecurringBuySubOrders(contextGenerate(), &RecurringBuySubOrdersRequest{AlgoID: "560473220642766848", OrderID: "123422"})
		require.NoError(t, err, "GetRecurringBuySubOrders must not error")
		if mockTests {
			exp := []RecurringBuySubOrderResponse{
				{
					AlgoID:              "560473220642766848",
					InstrumentType:      "SPOT",
					InstrumentID:        "ETH-USDC",
					AlgoOrderType:       "recurring",
					OrderID:             "123422",
					CreationTime:        types.Time(time.UnixMilli(1679911222200)),
					UpdateTime:          types.Time(time.UnixMilli(1679911222207)),
					TradeMode:           TradeModeCash,
					OrderType:           orderMarket,
					Size:                80,
					State:               "filled",
					Side:                "buy",
					Price:               -1,
					Fee:                 -0.0000317205,
					FeeCurrency:         currency.ETH,
					AveragePrice:        1765.4,
					AccumulatedFillSize: 0.045315,
					Tag:                 "recurringTag1",
					AlgoClientOrderID:   "recurringClient1",
				},
			}
			assert.Equal(t, exp, result, "GetRecurringBuySubOrders should decode every field")
		} else {
			assert.NotNil(t, result, "GetRecurringBuySubOrders should return a list")
		}
	})
}

func TestAmendRecurringBuyTime(t *testing.T) {
	t.Parallel()
	custom := func() *AmendRecurringBuyTimeRequest {
		return &AmendRecurringBuyTimeRequest{AlgoID: "2837428373700509696", RecurringTimeType: "1", TimeZone: 8, Period: "monthly", RecurringDay: 1, RecurringTime: new(uint64(11))}
	}
	for _, tc := range []struct {
		name string
		arg  func() *AmendRecurringBuyTimeRequest
		err  error
	}{
		{"nil request", func() *AmendRecurringBuyTimeRequest { return nil }, common.ErrNilPointer},
		{"missing algo ID", func() *AmendRecurringBuyTimeRequest { a := custom(); a.AlgoID = ""; return a }, errAlgoIDRequired},
		{"missing recurring time type", func() *AmendRecurringBuyTimeRequest { a := custom(); a.RecurringTimeType = ""; return a }, errRecurringTimeTypeRequired},
		{"time zone below -12", func() *AmendRecurringBuyTimeRequest { a := custom(); a.TimeZone = -13; return a }, errInvalidTimeZone},
		{"time zone above 14", func() *AmendRecurringBuyTimeRequest { a := custom(); a.TimeZone = 15; return a }, errInvalidTimeZone},
		{"missing period", func() *AmendRecurringBuyTimeRequest { a := custom(); a.Period = ""; return a }, errRecurringPeriodRequired},
		{"hourly without recurring hour", func() *AmendRecurringBuyTimeRequest { a := custom(); a.Period = "hourly"; return a }, errRecurringHourRequired},
		{"custom monthly without recurring day", func() *AmendRecurringBuyTimeRequest { a := custom(); a.RecurringDay = 0; return a }, errRecurringDayRequired},
		{"custom weekly without recurring day", func() *AmendRecurringBuyTimeRequest { a := custom(); a.Period = "weekly"; a.RecurringDay = 0; return a }, errRecurringDayRequired},
		{"custom without recurring time", func() *AmendRecurringBuyTimeRequest { a := custom(); a.RecurringTime = nil; return a }, errRecurringBuyTimeRequired},
		{"recurring time above 23", func() *AmendRecurringBuyTimeRequest { a := custom(); a.RecurringTime = new(uint64(24)); return a }, errRecurringBuyTimeRequired},
		{"immediate trigger with recurring time above 23", func() *AmendRecurringBuyTimeRequest {
			a := custom()
			a.RecurringTimeType = "2"
			a.RecurringTime = new(uint64(24))
			return a
		}, errRecurringBuyTimeRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.AmendRecurringBuyTime(t.Context(), tc.arg())
			assert.ErrorIs(t, err, tc.err, "AmendRecurringBuyTime should return the expected error")
		})
	}

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			arg  *AmendRecurringBuyTimeRequest
			body string
		}{
			{
				name: "custom time with every parameter",
				arg:  &AmendRecurringBuyTimeRequest{AlgoID: "2837428373700509696", RecurringTimeType: "1", TimeZone: -5, Period: "hourly", RecurringHour: 8, RecurringDay: 1, RecurringTime: new(uint64(0))},
				body: `{"algoId":"2837428373700509696","recurringTimeType":"1","timeZone":"-5","period":"hourly","recurringHour":"8","recurringDay":"1","recurringTime":"0"}`,
			},
			{
				name: "immediate trigger with required parameters only",
				arg:  &AmendRecurringBuyTimeRequest{AlgoID: "2837428373700509696", RecurringTimeType: "2", Period: "daily"},
				body: `{"algoId":"2837428373700509696","recurringTimeType":"2","timeZone":"0","period":"daily"}`,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				_, err := newRequestPinExchange(t, "/tradingBot/recurring/amend-recurring-time", tc.body, tradingBotAlgoIDData).AmendRecurringBuyTime(t.Context(), tc.arg)
				require.NoError(t, err, "AmendRecurringBuyTime must not error")
			})
		}
	})

	t.Run("amend time", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		}
		result, err := e.AmendRecurringBuyTime(contextGenerate(), &AmendRecurringBuyTimeRequest{
			AlgoID: "2837428373700509696", RecurringTimeType: "1", Period: "hourly", RecurringHour: 8, RecurringDay: 1, RecurringTime: new(uint64(11)), TimeZone: 8,
		})
		require.NoError(t, err, "AmendRecurringBuyTime must not error")
		if mockTests {
			assert.Equal(t, &TradingBotOrderResponse{AlgoID: "2837428373700509696"}, result, "AmendRecurringBuyTime should decode every field")
		} else {
			assert.NotNil(t, result, "AmendRecurringBuyTime should return a result")
		}
	})
}

func TestAmendRecurringBuyAmount(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *RecurringBuyAmountRequest
		err  error
	}{
		{"nil request", nil, common.ErrNilPointer},
		{"missing algo ID", &RecurringBuyAmountRequest{Amount: 20}, errAlgoIDRequired},
		{"missing amount", &RecurringBuyAmountRequest{AlgoID: "2837428373700509696"}, errRecurringBuyAmountRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.AmendRecurringBuyAmount(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "AmendRecurringBuyAmount should return the expected error")
		})
	}

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		ex := newRequestPinExchange(t, "/tradingBot/recurring/amend-recurring-amount", `{"algoId":"2837428373700509696","amount":"20.5"}`, tradingBotAlgoIDData)
		_, err := ex.AmendRecurringBuyAmount(t.Context(), &RecurringBuyAmountRequest{AlgoID: "2837428373700509696", Amount: 20.5})
		require.NoError(t, err, "AmendRecurringBuyAmount must not error")
	})

	t.Run("amend amount", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		}
		result, err := e.AmendRecurringBuyAmount(contextGenerate(), &RecurringBuyAmountRequest{AlgoID: "2837428373700509696", Amount: 20})
		require.NoError(t, err, "AmendRecurringBuyAmount must not error")
		if mockTests {
			assert.Equal(t, &TradingBotOrderResponse{AlgoID: "2837428373700509696"}, result, "AmendRecurringBuyAmount should decode every field")
		} else {
			assert.NotEmpty(t, result.AlgoID, "AmendRecurringBuyAmount should return an algo ID")
		}
	})
}

func TestAddRecurringBuyInvestment(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *RecurringBuyAmountRequest
		err  error
	}{
		{"nil request", nil, common.ErrNilPointer},
		{"missing algo ID", &RecurringBuyAmountRequest{Amount: 20}, errAlgoIDRequired},
		{"missing amount", &RecurringBuyAmountRequest{AlgoID: "2837428373700509696"}, errRecurringBuyAmountRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.AddRecurringBuyInvestment(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "AddRecurringBuyInvestment should return the expected error")
		})
	}

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		ex := newRequestPinExchange(t, "/tradingBot/recurring/add-investment", `{"algoId":"2837428373700509696","amount":"20.5"}`, tradingBotAlgoIDData)
		_, err := ex.AddRecurringBuyInvestment(t.Context(), &RecurringBuyAmountRequest{AlgoID: "2837428373700509696", Amount: 20.5})
		require.NoError(t, err, "AddRecurringBuyInvestment must not error")
	})

	t.Run("add investment", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		}
		result, err := e.AddRecurringBuyInvestment(contextGenerate(), &RecurringBuyAmountRequest{AlgoID: "2837428373700509696", Amount: 20})
		require.NoError(t, err, "AddRecurringBuyInvestment must not error")
		if mockTests {
			assert.Equal(t, &TradingBotOrderResponse{AlgoID: "2837428373700509696"}, result, "AddRecurringBuyInvestment should decode every field")
		} else {
			assert.NotEmpty(t, result.AlgoID, "AddRecurringBuyInvestment should return an algo ID")
		}
	})
}

func TestValidateRecurringBuyAmountRequest(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *RecurringBuyAmountRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "missing algo ID", arg: &RecurringBuyAmountRequest{Amount: 100}, err: errAlgoIDRequired},
		{name: "missing amount", arg: &RecurringBuyAmountRequest{AlgoID: "560472804207104000"}, err: errRecurringBuyAmountRequired},
		{name: "negative amount", arg: &RecurringBuyAmountRequest{AlgoID: "560472804207104000", Amount: -100}, err: errRecurringBuyAmountRequired},
		{name: "valid", arg: &RecurringBuyAmountRequest{AlgoID: "560472804207104000", Amount: 100}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIs(t, validateRecurringBuyAmountRequest(tc.arg), tc.err, "validateRecurringBuyAmountRequest should return the expected error")
		})
	}
}

func TestPauseRecurringBuy(t *testing.T) {
	t.Parallel()
	_, err := e.PauseRecurringBuy(t.Context(), "")
	require.ErrorIs(t, err, errAlgoIDRequired)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.PauseRecurringBuy(contextGenerate(), "2837428373700509696")
	require.NoError(t, err, "PauseRecurringBuy must not error")
	if mockTests {
		assert.Equal(t, &TradingBotOrderResponse{AlgoID: "2837428373700509696"}, result, "PauseRecurringBuy should decode every field")
	} else {
		assert.NotEmpty(t, result.AlgoID, "PauseRecurringBuy should return an algo ID")
	}
}

func TestRestartRecurringBuy(t *testing.T) {
	t.Parallel()
	_, err := e.RestartRecurringBuy(t.Context(), "")
	require.ErrorIs(t, err, errAlgoIDRequired)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.RestartRecurringBuy(contextGenerate(), "2837428373700509696")
	require.NoError(t, err, "RestartRecurringBuy must not error")
	if mockTests {
		assert.Equal(t, &TradingBotOrderResponse{AlgoID: "2837428373700509696"}, result, "RestartRecurringBuy should decode every field")
	} else {
		assert.NotEmpty(t, result.AlgoID, "RestartRecurringBuy should return an algo ID")
	}
}

func TestAmendRecurringBuyPriceRange(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *AmendRecurringBuyPriceRangeRequest
		err  error
	}{
		{"nil request", nil, common.ErrNilPointer},
		{"missing algo ID", &AmendRecurringBuyPriceRangeRequest{RecurringList: []RecurringBuyPriceRange{{Currency: currency.BTC}}}, errAlgoIDRequired},
		{"missing recurring list", &AmendRecurringBuyPriceRangeRequest{AlgoID: "2837428373700509696"}, errRecurringListRequired},
		{"price range without currency", &AmendRecurringBuyPriceRangeRequest{AlgoID: "2837428373700509696", RecurringList: []RecurringBuyPriceRange{{MinimumPrice: 80000}}}, currency.ErrCurrencyCodeEmpty},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.AmendRecurringBuyPriceRange(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "AmendRecurringBuyPriceRange should return the expected error")
		})
	}

	t.Run("request body sends no limit as an empty price", func(t *testing.T) {
		t.Parallel()
		ex := newRequestPinExchange(t, "/tradingBot/recurring/amend-price-range", `{"algoId":"2837428373700509696","recurringList":[{"ccy":"BTC","minPx":"80000","maxPx":""},{"ccy":"ETH","minPx":"","maxPx":"4000"}]}`, tradingBotAlgoIDData)
		_, err := ex.AmendRecurringBuyPriceRange(t.Context(), &AmendRecurringBuyPriceRangeRequest{AlgoID: "2837428373700509696", RecurringList: []RecurringBuyPriceRange{{Currency: currency.BTC, MinimumPrice: 80000}, {Currency: currency.ETH, MaximumPrice: 4000}}})
		require.NoError(t, err, "AmendRecurringBuyPriceRange must not error")
	})

	t.Run("amend price range", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		}
		result, err := e.AmendRecurringBuyPriceRange(contextGenerate(), &AmendRecurringBuyPriceRangeRequest{AlgoID: "2837428373700509696", RecurringList: []RecurringBuyPriceRange{{Currency: currency.BTC, MinimumPrice: 80000, MaximumPrice: 120000}}})
		require.NoError(t, err, "AmendRecurringBuyPriceRange must not error")
		if mockTests {
			assert.Equal(t, &TradingBotOrderResponse{AlgoID: "2837428373700509696"}, result, "AmendRecurringBuyPriceRange should decode every field")
		} else {
			assert.NotEmpty(t, result.AlgoID, "AmendRecurringBuyPriceRange should return an algo ID")
		}
	})
}
