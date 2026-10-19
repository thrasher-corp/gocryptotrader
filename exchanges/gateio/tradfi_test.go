package gateio

import (
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestTradFiResponseAsError(t *testing.T) {
	t.Parallel()
	var resp tradFiResponse[*TradFiTicker]
	require.NoError(t, json.Unmarshal([]byte(`{"label":"","data":{"last_price":"1.1"}}`), &resp), "Unmarshal must not error")
	assert.NoError(t, resp.AsError(), "AsError should not error without a label")

	require.NoError(t, json.Unmarshal([]byte(`{"label":"INVALID_ARGUMENT","message":"invalid argument","data":null}`), &resp), "Unmarshal must not error")
	err := resp.AsError()
	require.Error(t, err, "AsError must error with a label")
	assert.Contains(t, err.Error(), "INVALID_ARGUMENT", "AsError should carry the label")
	assert.Contains(t, err.Error(), "invalid argument", "AsError should carry the message")
	assert.ErrorContains(t, responseError(&resp), "INVALID_ARGUMENT", "responseError should surface the label")
}

func TestKlineIntervalToTypeString(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		interval kline.Interval
		exp      string
	}{
		{kline.OneMin, "1m"},
		{kline.FifteenMin, "15m"},
		{kline.OneHour, "1h"},
		{kline.FourHour, "4h"},
		{kline.OneDay, "1d"},
		{kline.OneWeek, "7d"},
		{kline.OneMonth, "30d"},
	} {
		klineType, err := klineIntervalToTypeString(tc.interval)
		require.NoErrorf(t, err, "klineIntervalToTypeString must not error for %s", tc.interval)
		assert.Equalf(t, tc.exp, klineType, "klineIntervalToTypeString should convert %s", tc.interval)
	}
	_, err := klineIntervalToTypeString(kline.FiveMin)
	assert.ErrorIs(t, err, kline.ErrUnsupportedInterval, "klineIntervalToTypeString should reject an unsupported interval")
}

func TestTradFiSideCode(t *testing.T) {
	t.Parallel()
	side, err := tradFiSideCode(order.Sell)
	require.NoError(t, err, "tradFiSideCode must not error for a sell")
	assert.Equal(t, uint8(1), side, "tradFiSideCode should convert a sell to 1")

	side, err = tradFiSideCode(order.Buy)
	require.NoError(t, err, "tradFiSideCode must not error for a buy")
	assert.Equal(t, uint8(2), side, "tradFiSideCode should convert a buy to 2")

	for _, s := range []order.Side{order.UnknownSide, order.Long, order.Short, order.Bid} {
		_, err = tradFiSideCode(s)
		assert.ErrorIsf(t, err, order.ErrSideIsInvalid, "tradFiSideCode should reject %s", s)
	}
}

func TestValidateTradFiCloseType(t *testing.T) {
	t.Parallel()
	assert.NoError(t, validateTradFiCloseType(1, 0.5), "validateTradFiCloseType should accept a partial close with a volume")
	assert.NoError(t, validateTradFiCloseType(2, 0), "validateTradFiCloseType should accept a full close without a volume")
	assert.ErrorIs(t, validateTradFiCloseType(1, 0), limits.ErrAmountBelowMin, "validateTradFiCloseType should require a partial close volume")
	assert.ErrorIs(t, validateTradFiCloseType(0, 1), errInvalidTradFiCloseType, "validateTradFiCloseType should reject an unset close type")
	assert.ErrorIs(t, validateTradFiCloseType(3, 1), errInvalidTradFiCloseType, "validateTradFiCloseType should reject an unknown close type")
}

func TestSetTradFiTimeRangeParams(t *testing.T) {
	t.Parallel()
	begin := time.Unix(1744103854, 0)
	end := time.Unix(1744190254, 0)

	params := url.Values{}
	require.NoError(t, setTradFiTimeRangeParams(params, begin, end), "setTradFiTimeRangeParams must not error")
	assert.Equal(t, url.Values{"begin_time": {"1744103854"}, "end_time": {"1744190254"}}, params, "setTradFiTimeRangeParams should set both times in seconds")

	params = url.Values{}
	require.NoError(t, setTradFiTimeRangeParams(params, begin, time.Time{}), "setTradFiTimeRangeParams must not error without an end time")
	assert.Equal(t, url.Values{"begin_time": {"1744103854"}}, params, "setTradFiTimeRangeParams should set only the begin time")

	params = url.Values{}
	require.NoError(t, setTradFiTimeRangeParams(params, time.Time{}, end), "setTradFiTimeRangeParams must not error without a begin time")
	assert.Equal(t, url.Values{"end_time": {"1744190254"}}, params, "setTradFiTimeRangeParams should set only the end time")

	params = url.Values{}
	assert.ErrorIs(t, setTradFiTimeRangeParams(params, end, begin), common.ErrStartAfterEnd, "setTradFiTimeRangeParams should reject a begin time after the end time")
	assert.Empty(t, params, "setTradFiTimeRangeParams should not set parameters for an invalid range")
}

func TestSetTradFiPageParams(t *testing.T) {
	t.Parallel()
	params := url.Values{}
	setTradFiPageParams(params, 0, 0)
	assert.Empty(t, params, "setTradFiPageParams should not set unset page parameters")

	setTradFiPageParams(params, 2, 50)
	assert.Equal(t, url.Values{"page": {"2"}, "page_size": {"50"}}, params, "setTradFiPageParams should set the page and page size")
}

func TestGetTradFiMT5Account(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetTradFiMT5Account(t.Context())
	require.NoError(t, err, "GetTradFiMT5Account must not error")
	if mockTests {
		exp := &TradFiMT5Account{Leverage: 100, StopOutLevel: "50%", Status: 3}
		assert.Equal(t, exp, result, "GetTradFiMT5Account should decode every field")
		return
	}
	assert.NotNil(t, result, "GetTradFiMT5Account should return a result")
}

func TestGetTradFiSymbolCategories(t *testing.T) {
	t.Parallel()
	result, err := e.GetTradFiSymbolCategories(t.Context())
	require.NoError(t, err, "GetTradFiSymbolCategories must not error")
	if mockTests {
		exp := []*TradFiCategory{
			{CategoryID: 0, CategoryName: "Favorites", IsFavourite: true},
			{CategoryID: 1, CategoryName: "Metals"},
			{CategoryID: 2, CategoryName: "Stocks"},
			{CategoryID: 3, CategoryName: "Indices"},
			{CategoryID: 4, CategoryName: "Forex"},
		}
		assert.Equal(t, exp, result, "GetTradFiSymbolCategories should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetTradFiSymbolCategories should return categories")
}

func TestGetTradFiSymbolCommissions(t *testing.T) {
	t.Parallel()
	_, err := e.GetTradFiSymbolCommissions(t.Context(), nil, nil)
	require.ErrorIs(t, err, errTradFiSymbolOrCategoryRequired, "GetTradFiSymbolCommissions must require a symbol or category")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetTradFiSymbolCommissions(t.Context(), []string{"AUDUSD", "XAUUSD"}, []string{"forex", "metal"})
	require.NoError(t, err, "GetTradFiSymbolCommissions must not error")
	if mockTests {
		exp := []*TradFiSymbolCommission{
			{CategoryCode: "forex", Symbol: "AUDUSD", FeePerLot: 6},
			{CategoryCode: "metal", Symbol: "XAUUSD", FeePerLot: 6.5},
		}
		assert.Equal(t, exp, result, "GetTradFiSymbolCommissions should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetTradFiSymbolCommissions should return commissions")
}

func TestGetTradFiSymbols(t *testing.T) {
	t.Parallel()
	result, err := e.GetTradFiSymbols(t.Context())
	require.NoError(t, err, "GetTradFiSymbols must not error")
	if mockTests {
		require.Len(t, result, 5, "GetTradFiSymbols must return every recorded symbol")
		exp := &TradFiSymbol{
			Symbol:                   "XAUUSD",
			SymbolDescription:        "Gold",
			CategoryID:               1,
			Status:                   "open",
			TradeMode:                "4",
			IconLink:                 "https://gimg.staticimgs.com/image/xauusd_20260115_162831_b7ddbdf6ff6ba89d1756a6d7fa9de9d8.png",
			CloseTime:                types.Time(time.Unix(1781729940, 0)),
			OpenTime:                 types.Time(time.Unix(1781647260, 0)),
			SettlementCurrency:       currency.USD,
			SettlementCurrencySymbol: "$",
			PricePrecision:           2,
			IsBase:                   true,
			Leverages:                []types.Number{20, 50, 100, 200, 500},
		}
		assert.Equal(t, exp, result[0], "GetTradFiSymbols should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetTradFiSymbols should return symbols")
}

func TestGetTradFiSymbolDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetTradFiSymbolDetails(t.Context(), nil)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "GetTradFiSymbolDetails must require a symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetTradFiSymbolDetails(t.Context(), []string{"EURUSD"})
	require.NoError(t, err, "GetTradFiSymbolDetails must not error")
	if mockTests {
		exp := []*TradFiSymbolDetail{{
			Symbol:             "EURUSD",
			SymbolDescription:  "Euro vs United States Dollar",
			CategoryName:       "Forex",
			ContractVolume:     100000,
			SettlementCurrency: currency.USD,
			MaxOrderVolume:     100,
			MinOrderVolume:     0.01,
			Leverage:           500,
			PricePrecision:     5,
			PriceStopLossLevel: 100,
			SwapCostType:       "1",
			BuySwapCostRate:    -7.937347,
			SellSwapCostRate:   2.856426,
			SwapCost3Day:       "3",
			TradeTimezone:      "GMT+2",
			TradeMode:          "4",
			IconLink:           "https://gimg.staticimgs.com/image/eurusd_20260123_120701_3982593a8387f849b5eb60a05bbadd3c.png",
		}}
		assert.Equal(t, exp, result, "GetTradFiSymbolDetails should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetTradFiSymbolDetails should return symbol details")
}

func TestGetTradFiKlines(t *testing.T) {
	t.Parallel()
	_, err := e.GetTradFiKlines(t.Context(), "", &GetTradFiKlinesRequest{Interval: kline.OneMin})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "GetTradFiKlines must require a symbol")

	_, err = e.GetTradFiKlines(t.Context(), "EURUSD", nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetTradFiKlines must reject a nil request")

	_, err = e.GetTradFiKlines(t.Context(), "EURUSD", &GetTradFiKlinesRequest{})
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval, "GetTradFiKlines must require a supported interval")

	startTime, endTime := getTime()
	_, err = e.GetTradFiKlines(t.Context(), "EURUSD", &GetTradFiKlinesRequest{Interval: kline.OneHour, BeginTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetTradFiKlines must reject a begin time after the end time")

	result, err := e.GetTradFiKlines(t.Context(), "EURUSD", &GetTradFiKlinesRequest{Interval: kline.OneMin})
	require.NoError(t, err, "GetTradFiKlines must not error")
	if mockTests {
		require.Len(t, result, 5, "GetTradFiKlines must return every recorded kline")
		exp := &TradFiKline{Open: 1.16068, Close: 1.16068, Low: 1.16066, High: 1.16102, Timestamp: types.Time(time.Unix(1781644140, 0))}
		assert.Equal(t, exp, result[0], "GetTradFiKlines should decode every field")
	} else {
		assert.NotEmpty(t, result, "GetTradFiKlines should return klines")
	}

	result, err = e.GetTradFiKlines(t.Context(), "EURUSD", &GetTradFiKlinesRequest{Interval: kline.OneHour, BeginTime: startTime, EndTime: endTime})
	require.NoError(t, err, "GetTradFiKlines must not error with a time range")
	if mockTests {
		exp := []*TradFiKline{
			{Open: 1.17198, Close: 1.17213, Low: 1.1718, High: 1.17268, Timestamp: types.Time(time.Unix(1744106400, 0))},
			{Open: 1.17208, Close: 1.16092, Low: 1.16023, High: 1.17263, Timestamp: types.Time(time.Unix(1744110000, 0))},
		}
		assert.Equal(t, exp, result, "GetTradFiKlines should decode every field with a time range")
	}

	result, err = e.GetTradFiKlines(t.Context(), "EURUSD", &GetTradFiKlinesRequest{Interval: kline.OneHour, EndTime: endTime, Limit: 2})
	require.NoError(t, err, "GetTradFiKlines must not error with an end time and limit")
	if mockTests {
		exp := []*TradFiKline{
			{Open: 1.17221, Close: 1.17246, Low: 1.17202, High: 1.17289, Timestamp: types.Time(time.Unix(1744182000, 0))},
			{Open: 1.17246, Close: 1.17191, Low: 1.17175, High: 1.17258, Timestamp: types.Time(time.Unix(1744185600, 0))},
		}
		assert.Equal(t, exp, result, "GetTradFiKlines should decode every field with an end time and limit")
		return
	}
	assert.Len(t, result, 2, "GetTradFiKlines should return the limited number of klines")
}

func TestGetTradFiSymbolTicker(t *testing.T) {
	t.Parallel()
	_, err := e.GetTradFiSymbolTicker(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "GetTradFiSymbolTicker must require a symbol")

	result, err := e.GetTradFiSymbolTicker(t.Context(), "EURUSD")
	require.NoError(t, err, "GetTradFiSymbolTicker must not error")
	if mockTests {
		exp := &TradFiTicker{
			HighestPrice:          1.16129,
			LowestPrice:           1.16046,
			PriceChangePercentage: 0.03,
			PriceChangeAmount:     0.00043,
			TodayOpenPrice:        1.16083,
			PreviousClosePrice:    1.16083,
			LastPrice:             1.16126,
			BidPrice:              1.16126,
			AskPrice:              1.16128,
			Status:                "open",
			CloseTime:             types.Time(time.Unix(1781729940, 0)),
			OpenTime:              types.Time(time.Unix(1781643660, 0)),
			TradeMode:             "4",
			CategoryName:          "Forex",
		}
		assert.Equal(t, exp, result, "GetTradFiSymbolTicker should decode every field")
		return
	}
	assert.NotNil(t, result, "GetTradFiSymbolTicker should return a ticker")
}

func TestActivateTradFiUser(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.ActivateTradFiUser(t.Context())
	require.NoError(t, err, "ActivateTradFiUser must not error")
	if mockTests {
		assert.Equal(t, &TradFiUserInfo{Status: 3, Leverage: 1}, result, "ActivateTradFiUser should decode every field")
		return
	}
	assert.NotNil(t, result, "ActivateTradFiUser should return the account state")
}

func TestGetTradFiUserAssets(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetTradFiUserAssets(t.Context())
	require.NoError(t, err, "GetTradFiUserAssets must not error")
	if mockTests {
		exp := &TradFiUserAssets{Equity: 1012.55, MarginLevel: 4299.58, Balance: 1000, Margin: 23.55, MarginFree: 989, UnrealisedPNL: 12.55}
		assert.Equal(t, exp, result, "GetTradFiUserAssets should decode every field")
		return
	}
	assert.NotNil(t, result, "GetTradFiUserAssets should return assets")
}

func TestCreateTradFiTransaction(t *testing.T) {
	t.Parallel()
	err := e.CreateTradFiTransaction(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreateTradFiTransaction must reject a nil request")

	err = e.CreateTradFiTransaction(t.Context(), &TradFiTransactionRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "CreateTradFiTransaction must require an asset")

	err = e.CreateTradFiTransaction(t.Context(), &TradFiTransactionRequest{Asset: currency.USDT})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "CreateTradFiTransaction must require a positive amount")

	err = e.CreateTradFiTransaction(t.Context(), &TradFiTransactionRequest{Asset: currency.USDT, Change: 100})
	require.ErrorIs(t, err, errTradFiTransactionTypeRequired, "CreateTradFiTransaction must require a transaction type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	err = e.CreateTradFiTransaction(t.Context(), &TradFiTransactionRequest{Asset: currency.USDT, Change: 100.00, Type: "deposit"})
	require.NoError(t, err, "CreateTradFiTransaction must not error")
}

func TestGetTradFiTransactions(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetTradFiTransactions(t.Context(), &GetTradFiTransactionsRequest{BeginTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetTradFiTransactions must reject a begin time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetTradFiTransactions(t.Context(), nil)
	require.NoError(t, err, "GetTradFiTransactions must not error")
	if mockTests {
		exp := &TradFiTransactionListData{
			Total:      2,
			TotalPages: 1,
			List: []*TradFiTransaction{
				{Asset: currency.USDT, Type: "dividend", TypeDescription: "Dividend Adjustment", Change: 1, Balance: 1, Time: types.Time(time.Unix(1769329389, 0))},
				{Asset: currency.USDT, Type: "fill_negative", TypeDescription: "Cover Negative Balance", Change: 0.5, Balance: 0.5, Time: types.Time(time.Unix(1769238545, 0))},
			},
		}
		assert.Equal(t, exp, result, "GetTradFiTransactions should decode every field")
	} else {
		assert.NotNil(t, result, "GetTradFiTransactions should return a result")
	}

	result, err = e.GetTradFiTransactions(t.Context(), &GetTradFiTransactionsRequest{
		BeginTime: startTime,
		EndTime:   endTime,
		Type:      "deposit",
		Page:      1,
		PageSize:  10,
	})
	require.NoError(t, err, "GetTradFiTransactions must not error with filters")
	if mockTests {
		exp := &TradFiTransactionListData{
			Total:      1,
			TotalPages: 1,
			List: []*TradFiTransaction{
				{Asset: currency.USDT, Type: "deposit", TypeDescription: "Transfer In", Change: 100, Balance: 100, Time: types.Time(time.Unix(1744150000, 0))},
			},
		}
		assert.Equal(t, exp, result, "GetTradFiTransactions should decode every field with filters")
		return
	}
	assert.NotNil(t, result, "GetTradFiTransactions should return a result with filters")
}

func TestTradFiOrderRequestMarshalJSON(t *testing.T) {
	t.Parallel()
	payload, err := json.Marshal(&TradFiOrderRequest{
		Symbol:    "EURUSD",
		Side:      order.Sell,
		PriceType: order.Market,
		Price:     1.09,
		Volume:    0.1,
	})
	require.NoError(t, err, "Marshal must not error")
	assert.JSONEq(t, `{"symbol":"EURUSD","side":1,"price_type":"market","price":"1.09","volume":"0.1"}`, string(payload), "MarshalJSON should convert the side and price type and omit unset options")

	_, err = json.Marshal(&TradFiOrderRequest{Symbol: "EURUSD", Side: order.Long, PriceType: order.Market})
	assert.ErrorIs(t, err, order.ErrSideIsInvalid, "MarshalJSON should reject a side other than buy or sell")
}

func TestCreateTradFiOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CreateTradFiOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreateTradFiOrder must reject a nil request")

	_, err = e.CreateTradFiOrder(t.Context(), &TradFiOrderRequest{})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "CreateTradFiOrder must require a symbol")

	_, err = e.CreateTradFiOrder(t.Context(), &TradFiOrderRequest{Symbol: "EURUSD"})
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "CreateTradFiOrder must require a buy or sell side")

	_, err = e.CreateTradFiOrder(t.Context(), &TradFiOrderRequest{Symbol: "EURUSD", Side: order.Buy, PriceType: order.Limit})
	require.ErrorIs(t, err, order.ErrUnsupportedOrderType, "CreateTradFiOrder must require a market or trigger price type")

	_, err = e.CreateTradFiOrder(t.Context(), &TradFiOrderRequest{Symbol: "EURUSD", Side: order.Buy, PriceType: order.Trigger})
	require.ErrorIs(t, err, limits.ErrPriceBelowMin, "CreateTradFiOrder must require a price")

	_, err = e.CreateTradFiOrder(t.Context(), &TradFiOrderRequest{Symbol: "EURUSD", Side: order.Buy, PriceType: order.Trigger, Price: 1.09})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "CreateTradFiOrder must require a volume")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CreateTradFiOrder(t.Context(), &TradFiOrderRequest{
		Symbol:          "EURUSD",
		Side:            order.Buy,
		PriceType:       order.Trigger,
		Price:           1.09,
		Volume:          0.1,
		PriceTakeProfit: 1.1,
		PriceStopLoss:   1.08,
		Leverage:        100,
	})
	require.NoError(t, err, "CreateTradFiOrder must not error")
	if mockTests {
		assert.Equal(t, &TradFiCreateOrderResult{ID: "117"}, result, "CreateTradFiOrder should decode every field")
		return
	}
	assert.NotEmpty(t, result.ID, "CreateTradFiOrder should return a queue task ID")
}

func TestGetTradFiActiveOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetTradFiActiveOrders(t.Context())
	require.NoError(t, err, "GetTradFiActiveOrders must not error")
	if mockTests {
		exp := []*TradFiOrder{
			{
				OrderID:           2630591,
				Symbol:            "USDCHF",
				SymbolDescription: "US Dollar vs Swiss Franc",
				PriceType:         "trigger",
				State:             1,
				StateDescription:  "Placed",
				Side:              2,
				Volume:            1.6,
				Price:             5,
				PriceTakeProfit:   5.2,
				PriceStopLoss:     4.8,
				TimeSetup:         types.Time(time.Unix(1768741530, 0)),
			},
			{
				OrderID:           2630590,
				Symbol:            "USDCHF",
				SymbolDescription: "US Dollar vs Swiss Franc",
				PriceType:         "trigger",
				State:             1,
				StateDescription:  "Placed",
				Side:              2,
				Volume:            1.6,
				Price:             5,
				PriceTakeProfit:   5.1,
				PriceStopLoss:     4.9,
				TimeSetup:         types.Time(time.Unix(1768741526, 0)),
			},
		}
		assert.Equal(t, exp, result, "GetTradFiActiveOrders should decode every field")
	}
}

func TestUpdateTradFiOrder(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateTradFiOrder(t.Context(), 0, &TradFiOrderUpdateRequest{Price: 1.091})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "UpdateTradFiOrder must require an order ID")

	_, err = e.UpdateTradFiOrder(t.Context(), 1223, nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "UpdateTradFiOrder must reject a nil request")

	_, err = e.UpdateTradFiOrder(t.Context(), 1223, &TradFiOrderUpdateRequest{})
	require.ErrorIs(t, err, limits.ErrPriceBelowMin, "UpdateTradFiOrder must require a price")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.UpdateTradFiOrder(t.Context(), 1223, &TradFiOrderUpdateRequest{Price: 1.091, PriceTakeProfit: 1.1, PriceStopLoss: 1.08})
	require.NoError(t, err, "UpdateTradFiOrder must not error")
	if mockTests {
		exp := &TradFiUpdatedOrder{OrderID: 1223, Symbol: "EURUSD", State: "1", Volume: 0.1, Price: 1.091, PriceTakeProfit: 1.1, PriceStopLoss: 1.08}
		assert.Equal(t, exp, result, "UpdateTradFiOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "UpdateTradFiOrder should return the updated order")
}

func TestCancelTradFiOrder(t *testing.T) {
	t.Parallel()
	err := e.CancelTradFiOrder(t.Context(), 0)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelTradFiOrder must require an order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	err = e.CancelTradFiOrder(t.Context(), 1223)
	require.NoError(t, err, "CancelTradFiOrder must not error")
}

func TestGetTradFiOrderHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetTradFiOrderHistory(t.Context(), &GetTradFiOrderHistoryRequest{BeginTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetTradFiOrderHistory must reject a begin time after the end time")

	_, err = e.GetTradFiOrderHistory(t.Context(), &GetTradFiOrderHistoryRequest{Side: order.Long})
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "GetTradFiOrderHistory must reject a side other than buy or sell")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetTradFiOrderHistory(t.Context(), nil)
	require.NoError(t, err, "GetTradFiOrderHistory must not error")
	if mockTests {
		exp := []*TradFiHistoricalOrder{{
			OrderID:           2648991,
			Symbol:            "USDCAD",
			SymbolDescription: "USD VS CAD;",
			PriceType:         "market",
			OrderOptType:      4,
			State:             4,
			StateDescription:  "Filled",
			Side:              2,
			Volume:            0.05,
			FillVolume:        0.05,
			ClosePNL:          -1.49755,
			Price:             1.3689,
			TriggerPrice:      1.3689,
			TimeSetup:         types.Time(time.Unix(1769397512, 0)),
			TimeDone:          types.Time(time.Unix(1769397512, 0)),
		}}
		assert.Equal(t, exp, result, "GetTradFiOrderHistory should decode every field")
	}

	result, err = e.GetTradFiOrderHistory(t.Context(), &GetTradFiOrderHistoryRequest{
		BeginTime: startTime,
		EndTime:   endTime,
		Symbol:    "EURUSD",
		Side:      order.Buy,
	})
	require.NoError(t, err, "GetTradFiOrderHistory must not error with filters")
	if mockTests {
		exp := []*TradFiHistoricalOrder{{
			OrderID:           2648980,
			Symbol:            "EURUSD",
			SymbolDescription: "Euro vs United States Dollar",
			PriceType:         "trigger",
			OrderOptType:      2,
			State:             4,
			StateDescription:  "Filled",
			Side:              2,
			Volume:            0.1,
			FillVolume:        0.1,
			Price:             1.09,
			TriggerPrice:      1.09,
			PriceTakeProfit:   1.1,
			PriceStopLoss:     1.08,
			TimeSetup:         types.Time(time.Unix(1744110000, 0)),
			TimeDone:          types.Time(time.Unix(1744113600, 0)),
		}}
		assert.Equal(t, exp, result, "GetTradFiOrderHistory should decode every field with filters")
	}
}

func TestGetTradFiOrderLog(t *testing.T) {
	t.Parallel()
	_, err := e.GetTradFiOrderLog(t.Context(), 0)
	require.ErrorIs(t, err, errTradFiOrderLogIDRequired, "GetTradFiOrderLog must require a log ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetTradFiOrderLog(t.Context(), 1214290)
	require.NoError(t, err, "GetTradFiOrderLog must not error")
	if mockTests {
		exp := &TradFiOrderLogResponse{
			OrderID:   13950884,
			LogID:     1214290,
			Symbol:    "AUDUSD",
			PriceType: "market",
			State:     4,
			Side:      2,
			Volume:    0.01,
			Price:     0.72221,
		}
		assert.Equal(t, exp, result, "GetTradFiOrderLog should decode every field")
		return
	}
	assert.NotNil(t, result, "GetTradFiOrderLog should return the order")
}

func TestGetTradFiActivePositions(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetTradFiActivePositions(t.Context())
	require.NoError(t, err, "GetTradFiActivePositions must not error")
	if mockTests {
		exp := []*TradFiPosition{{
			PositionID:        2648925,
			Symbol:            "EURUSD",
			SymbolDescription: "Euro vs United States Dollar",
			Margin:            2.354440320000000142,
			UnrealisedPNL:     8.0697,
			UnrealisedPNLRate: 0.0068548775107622,
			Volume:            0.01,
			PriceOpen:         1.17722,
			PositionDirection: "Long",
			PriceTakeProfit:   1.19,
			PriceStopLoss:     1.17,
			CounterpartyPrice: 1.18515986,
			TimeCreate:        types.Time(time.Unix(1769319320, 0)),
		}}
		assert.Equal(t, exp, result, "GetTradFiActivePositions should decode every field")
	}
}

func TestUpdateTradFiPosition(t *testing.T) {
	t.Parallel()
	err := e.UpdateTradFiPosition(t.Context(), 0, &TradFiPositionUpdateRequest{})
	require.ErrorIs(t, err, errTradFiPositionIDRequired, "UpdateTradFiPosition must require a position ID")

	err = e.UpdateTradFiPosition(t.Context(), 1223, nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "UpdateTradFiPosition must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	err = e.UpdateTradFiPosition(t.Context(), 1223, &TradFiPositionUpdateRequest{PriceTakeProfit: 1.10000, PriceStopLoss: 1.08000})
	require.NoError(t, err, "UpdateTradFiPosition must not error")
}

func TestCloseTradFiPosition(t *testing.T) {
	t.Parallel()
	err := e.CloseTradFiPosition(t.Context(), 0, &TradFiClosePositionRequest{CloseType: 2})
	require.ErrorIs(t, err, errTradFiPositionIDRequired, "CloseTradFiPosition must require a position ID")

	err = e.CloseTradFiPosition(t.Context(), 1223, nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CloseTradFiPosition must reject a nil request")

	err = e.CloseTradFiPosition(t.Context(), 1223, &TradFiClosePositionRequest{})
	require.ErrorIs(t, err, errInvalidTradFiCloseType, "CloseTradFiPosition must require a close type")

	err = e.CloseTradFiPosition(t.Context(), 1223, &TradFiClosePositionRequest{CloseType: 1})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "CloseTradFiPosition must require a volume for a partial close")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	err = e.CloseTradFiPosition(t.Context(), 1223, &TradFiClosePositionRequest{CloseType: 1, CloseVolume: 0.01})
	require.NoError(t, err, "CloseTradFiPosition must not error")
}

func TestGetTradFiPositionHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetTradFiPositionHistory(t.Context(), &GetTradFiPositionHistoryRequest{BeginTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetTradFiPositionHistory must reject a begin time after the end time")

	_, err = e.GetTradFiPositionHistory(t.Context(), &GetTradFiPositionHistoryRequest{PositionDirection: order.Buy})
	require.ErrorIs(t, err, errInvalidTradFiPositionFilter, "GetTradFiPositionHistory must reject a direction other than long or short")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetTradFiPositionHistory(t.Context(), nil)
	require.NoError(t, err, "GetTradFiPositionHistory must not error")
	if mockTests {
		exp := &TradFiHistoricalPositionListData{
			Total:      2,
			TotalPages: 1,
			List: []*TradFiHistoricalPosition{
				{
					PositionID:        2648854,
					Symbol:            "EURUSD",
					RealisedPNL:       -2.4,
					RealisedPNLRate:   -0.0339783598282142,
					Volume:            0.3,
					VolumeClosed:      0.3,
					PriceOpen:         1.17722016,
					PositionDirection: "Long",
					ClosePrice:        1.1772198600000001,
					TimeCreate:        types.Time(time.Unix(1769223566, 0)),
					TimeClose:         types.Time(time.Unix(1769223573, 0)),
					PositionStatus:    "1",
					RealisedPNLDetail: TradFiRealisedPNLDetail{Fee: -2.4},
				},
				{
					PositionID:        2648850,
					Symbol:            "EURUSD",
					RealisedPNL:       -1.6,
					RealisedPNLRate:   -0.0339783598282142,
					Volume:            0.2,
					VolumeClosed:      0.2,
					PriceOpen:         1.17722016,
					PositionDirection: "Long",
					ClosePrice:        1.1772198600000001,
					TimeCreate:        types.Time(time.Unix(1769223287, 0)),
					TimeClose:         types.Time(time.Unix(1769223391, 0)),
					PositionStatus:    "1",
					RealisedPNLDetail: TradFiRealisedPNLDetail{Fee: -1.6},
				},
			},
		}
		assert.Equal(t, exp, result, "GetTradFiPositionHistory should decode every field")
	} else {
		assert.NotNil(t, result, "GetTradFiPositionHistory should return a result")
	}

	result, err = e.GetTradFiPositionHistory(t.Context(), &GetTradFiPositionHistoryRequest{
		Page:              1,
		PageSize:          10,
		Symbol:            "EURUSD",
		PositionDirection: order.Long,
	})
	require.NoError(t, err, "GetTradFiPositionHistory must not error with filters")
	if mockTests {
		exp := &TradFiHistoricalPositionListData{
			Total:      1,
			TotalPages: 1,
			List: []*TradFiHistoricalPosition{{
				PositionID:        2648861,
				Symbol:            "EURUSD",
				RealisedPNL:       -48.2,
				RealisedPNLRate:   -0.97,
				Volume:            0.5,
				VolumeClosed:      0.5,
				PriceOpen:         1.1772,
				PositionDirection: "Long",
				PriceTakeProfit:   1.2,
				CounterpartyPrice: 1.1676,
				ClosePrice:        1.1675,
				TimeCreate:        types.Time(time.Unix(1769223566, 0)),
				TimeClose:         types.Time(time.Unix(1769224573, 0)),
				PositionStatus:    "2",
				CloseDetail:       &TradFiLiquidationDetail{MarginLevel: 49.5, Margin: 11.77, Equity: 5.83, StopOutLevel: 50},
				RealisedPNLDetail: TradFiRealisedPNLDetail{ClosedPNL: -48.5, Swap: -0.3, Fee: 0.6},
			}},
		}
		assert.Equal(t, exp, result, "GetTradFiPositionHistory should decode every field with filters")
		return
	}
	assert.NotNil(t, result, "GetTradFiPositionHistory should return a result with filters")
}
