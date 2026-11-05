package gate

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
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestSetStockPNLParams(t *testing.T) {
	t.Parallel()
	params := url.Values{}
	setStockPNLParams(params, 0, 0)
	assert.Empty(t, params, "setStockPNLParams should not set unset calculation types")

	setStockPNLParams(params, 2, 1)
	assert.Equal(t, url.Values{"pnl_calc_type": {"2"}, "pnl_calc_price": {"1"}}, params, "setStockPNLParams should set both calculation types")
}

func TestGetStockUserAssets(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetStockUserAssets(t.Context(), &GetStockUserAssetsRequest{PNLCalculationCostType: 2, PNLCalculationPriceType: 1})
	require.NoError(t, err, "GetStockUserAssets must not error")
	if mockTests {
		exp := &StockUserAssetsResponse{
			Equity:                    10000.12,
			Balance:                   8000,
			Available:                 6500.5,
			PositionMarketValue:       1500.25,
			PositionPNL:               12.5,
			TodayPNL:                  3.2,
			OptionPositionMarketValue: 499.37,
			OptionPositionPNL:         -1.5,
			OptionTodayPNL:            0.25,
			UserExists:                true,
		}
		assert.Equal(t, exp, result, "GetStockUserAssets should decode every field")
		return
	}
	assert.NotNil(t, result, "GetStockUserAssets should return assets")
}

func TestGetStockSymbols(t *testing.T) {
	t.Parallel()
	result, err := e.GetStockSymbols(t.Context(), &GetStockSymbolsRequest{
		Symbols:                      []string{"AAPL", "TSLA"},
		Exchange:                     "us",
		WithMultilingualDescriptions: true,
		Page:                         1,
		PageSize:                     2,
	})
	require.NoError(t, err, "GetStockSymbols must not error")
	if mockTests {
		exp := &StockSymbolsResponse{
			Total:      2,
			TotalPages: 1,
			List: []*StockSymbol{
				{
					Symbol:                 "AAPL",
					Exchange:               "us",
					ExchangeDescription:    "US Stocks",
					QuoteCurrency:          currency.USD,
					QuoteCurrencyPrecision: 2,
					QuoteCurrencyUSDRate:   1,
					SymbolDescription:      "Apple Inc.",
					Category:               "CS",
					AssetType:              asset.Stock,
					TradeStatus:            "gt_lp",
					TradeMode:              4,
					OrderFillTiming:        1,
					IconLink:               "https://gimg.staticimgs.com/image/aapl_20260530_225346_4c48886d2def6cbe9aa8c887c38c29b5.png",
					QuoteCurrencySymbol:    "$",
					PricePrecision:         2,
					VolumePrecision:        2,
					SellPriceProtection:    0.05,
					BuyPriceProtection:     0.05,
					SymbolDescriptions:     []*StockSymbolDescription{{Language: "en", Value: "Apple Inc."}, {Language: "zh", Value: "苹果公司"}, {Language: "tw", Value: "蘋果"}},
				},
				{
					Symbol:                 "TSLA",
					Exchange:               "us",
					ExchangeDescription:    "US Stocks",
					QuoteCurrency:          currency.USD,
					QuoteCurrencyPrecision: 2,
					QuoteCurrencyUSDRate:   1,
					SymbolDescription:      "Tesla, Inc.",
					Category:               "CS",
					AssetType:              asset.Stock,
					TradeStatus:            "gt_lp",
					TradeMode:              4,
					OrderFillTiming:        1,
					IconLink:               "https://gimg.staticimgs.com/image/tsla_20260530_225346_416ba795cd897d97eeb80525aca1e745.png",
					QuoteCurrencySymbol:    "$",
					PricePrecision:         2,
					VolumePrecision:        2,
					SellPriceProtection:    0.05,
					BuyPriceProtection:     0.05,
					SymbolDescriptions:     []*StockSymbolDescription{{Language: "en", Value: "Tesla, Inc."}, {Language: "zh", Value: "特斯拉"}, {Language: "tw", Value: "特斯拉–"}},
				},
			},
		}
		assert.Equal(t, exp, result, "GetStockSymbols should decode every field")
		return
	}
	require.NotNil(t, result, "GetStockSymbols must return a result")
	assert.NotEmpty(t, result.List, "GetStockSymbols should return symbols")
}

func TestGetStockSymbolDetails(t *testing.T) {
	t.Parallel()
	result, err := e.GetStockSymbolDetails(t.Context(), &GetStockSymbolDetailsRequest{Symbols: []string{"AAPL"}, Exchange: "us", Page: 1, PageSize: 1})
	require.NoError(t, err, "GetStockSymbolDetails must not error")
	if mockTests {
		exp := &StockSymbolDetailsResponse{
			Total:      1,
			TotalPages: 1,
			List: []*StockSymbolDetail{{
				Symbol:                 "AAPL",
				Exchange:               "us",
				ExchangeDescription:    "US Stocks",
				QuoteCurrency:          currency.USD,
				QuoteCurrencyPrecision: 2,
				QuoteCurrencyUSDRate:   1,
				SymbolDescription:      "Apple Inc.",
				Category:               "CS",
				AssetType:              asset.Stock,
				SettlementCurrency:     currency.USD,
				MaxOrderVolume:         1621,
				StepOrderVolume:        0.01,
				MinOrderVolume:         0.01,
				PricePrecision:         2,
				VolumePrecision:        2,
				PriceProtection:        0.01,
				SellPriceProtection:    0.05,
				BuyPriceProtection:     0.05,
				SlippageRate:           0.004,
				CommissionRate:         0.0005,
				TradeStatus:            "gt_lp",
				TradeMode:              4,
				OrderFillTiming:        1,
				SymbolDescriptions:     []*StockSymbolDescription{{Language: "en", Value: "Apple Inc."}, {Language: "zh", Value: "苹果公司"}, {Language: "tw", Value: "蘋果"}},
				IconLink:               "https://gimg.staticimgs.com/image/aapl_20260530_225346_4c48886d2def6cbe9aa8c887c38c29b5.png",
			}},
		}
		assert.Equal(t, exp, result, "GetStockSymbolDetails should decode every field")
		return
	}
	require.NotNil(t, result, "GetStockSymbolDetails must return a result")
	assert.NotEmpty(t, result.List, "GetStockSymbolDetails should return symbol details")
}

func TestGetStockOrderbook(t *testing.T) {
	t.Parallel()
	_, err := e.GetStockOrderbook(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "GetStockOrderbook must require a symbol")

	result, err := e.GetStockOrderbook(t.Context(), "AAPL")
	require.NoError(t, err, "GetStockOrderbook must not error")
	if mockTests {
		exp := &StockOrderbookResponse{
			Symbol: "AAPL",
			Bids:   []*StockOrderbookLevel{{Price: 342.05}},
			Asks:   []*StockOrderbookLevel{{Price: 342.76}, {Price: 344}, {Price: 352}},
		}
		assert.Equal(t, exp, result, "GetStockOrderbook should decode every field")
		return
	}
	assert.NotNil(t, result, "GetStockOrderbook should return an orderbook")
}

func TestStockOrderRequestMarshalJSON(t *testing.T) {
	t.Parallel()
	payload, err := json.Marshal(&StockOrderRequest{Symbol: "AAPL", Side: order.Sell, PriceType: order.Market, Volume: 2})
	require.NoError(t, err, "Marshal must not error for a market order")
	assert.JSONEq(t, `{"symbol":"AAPL","side":1,"price_type":"market","trading_session":"regular","time_in_force":"day","volume":"2"}`, string(payload), "MarshalJSON should fill the regular session and day time in force for a market order")

	payload, err = json.Marshal(&StockOrderRequest{Symbol: "AAPL", Side: order.Buy, PriceType: order.Limit, Volume: 2, Price: 200, TimeInForce: order.GoodTillDay})
	require.NoError(t, err, "Marshal must not error for a limit order")
	assert.JSONEq(t, `{"symbol":"AAPL","side":2,"price_type":"limit","trading_session":"all","time_in_force":"day","volume":"2","price":"200"}`, string(payload), "MarshalJSON should fill the all session for a limit order")

	payload, err = json.Marshal(&StockOrderRequest{Symbol: "AAPL", Side: order.Buy, PriceType: order.Limit, Volume: 2, Price: 200, TradingSession: "regular"})
	require.NoError(t, err, "Marshal must not error with a trading session")
	assert.JSONEq(t, `{"symbol":"AAPL","side":2,"price_type":"limit","trading_session":"regular","time_in_force":"day","volume":"2","price":"200"}`, string(payload), "MarshalJSON should keep an explicit trading session")

	_, err = json.Marshal(&StockOrderRequest{Symbol: "AAPL", PriceType: order.Market, Volume: 2})
	assert.ErrorIs(t, err, order.ErrSideIsInvalid, "MarshalJSON should reject an unset side")
}

func TestCreateStockOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CreateStockOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreateStockOrder must reject a nil request")

	_, err = e.CreateStockOrder(t.Context(), &StockOrderRequest{})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "CreateStockOrder must require a symbol")

	_, err = e.CreateStockOrder(t.Context(), &StockOrderRequest{Symbol: "AAPL"})
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "CreateStockOrder must require a buy or sell side")

	_, err = e.CreateStockOrder(t.Context(), &StockOrderRequest{Symbol: "AAPL", Side: order.Buy, PriceType: order.Trigger})
	require.ErrorIs(t, err, order.ErrUnsupportedOrderType, "CreateStockOrder must require a market or limit price type")

	_, err = e.CreateStockOrder(t.Context(), &StockOrderRequest{Symbol: "AAPL", Side: order.Buy, PriceType: order.Limit})
	require.ErrorIs(t, err, order.ErrPriceMustBeSetIfLimitOrder, "CreateStockOrder must require a price for a limit order")

	_, err = e.CreateStockOrder(t.Context(), &StockOrderRequest{Symbol: "AAPL", Side: order.Buy, PriceType: order.Market})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "CreateStockOrder must require a volume")

	_, err = e.CreateStockOrder(t.Context(), &StockOrderRequest{Symbol: "AAPL", Side: order.Buy, PriceType: order.Market, Volume: 1, TimeInForce: order.GoodTillCancel})
	require.ErrorIs(t, err, order.ErrUnsupportedTimeInForce, "CreateStockOrder must reject a time in force other than a day order")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CreateStockOrder(t.Context(), &StockOrderRequest{
		Symbol:        "AAPL",
		Side:          order.Buy,
		PriceType:     order.Limit,
		Volume:        10,
		Price:         200.12,
		ClientOrderID: "client-202607070001",
	})
	require.NoError(t, err, "CreateStockOrder must not error")
	if mockTests {
		assert.Equal(t, &StockCreateOrderResponse{OrderID: "123456"}, result, "CreateStockOrder should decode every field")
		return
	}
	assert.NotEmpty(t, result.OrderID, "CreateStockOrder should return an order ID")
}

func TestGetStockOpenOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetStockOpenOrders(t.Context(), "AAPL")
	require.NoError(t, err, "GetStockOpenOrders must not error")
	if mockTests {
		exp := []*StockOpenOrderResponse{{
			OrderID:              "123456",
			Symbol:               "AAPL",
			Exchange:             "us",
			QuoteCurrency:        currency.USD,
			QuoteCurrencyUSDRate: 1,
			SymbolDescription:    "Apple Inc.",
			TradeStatus:          "open",
			TradeMode:            4,
			PriceType:            "limit",
			Side:                 2,
			Status:               2,
			Volume:               10,
			FillVolume:           2,
			Price:                200.12,
			TimeSetup:            types.Time(time.Unix(1769378400, 0)),
			TimeUpdate:           types.Time(time.Unix(1769378500, 0)),
			MaxOrderVolume:       10000,
			StepOrderVolume:      1,
			MinOrderVolume:       1,
			PricePrecision:       2,
			PriceProtection:      0.1,
			SellPriceProtection:  0.1,
			BuyPriceProtection:   0.1,
			CommissionRate:       0.001,
			SlippageRate:         0.001,
		}}
		assert.Equal(t, exp, result, "GetStockOpenOrders should decode every field")
	}
}

func TestCancelAllStockOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	require.NoError(t, e.CancelAllStockOrders(t.Context()), "CancelAllStockOrders must not error")
}

func TestGetStockOrderHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetStockOrderHistory(t.Context(), &GetStockOrderHistoryRequest{BeginTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetStockOrderHistory must reject a begin time after the end time")

	_, err = e.GetStockOrderHistory(t.Context(), &GetStockOrderHistoryRequest{Side: order.Short})
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "GetStockOrderHistory must reject a side other than buy or sell")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetStockOrderHistory(t.Context(), &GetStockOrderHistoryRequest{
		Symbol:    "AAPL",
		OrderIDs:  []string{"123456", "123457"},
		BeginTime: startTime,
		EndTime:   endTime,
		Side:      order.Buy,
		Page:      1,
		PageSize:  10,
	})
	require.NoError(t, err, "GetStockOrderHistory must not error")
	if mockTests {
		exp := &StockOrderHistoryResponse{
			Total:      1,
			TotalPages: 1,
			List: []*StockHistoricalOrder{{
				OrderID:              "123456",
				Symbol:               "AAPL",
				Exchange:             "us",
				QuoteCurrency:        currency.USD,
				QuoteCurrencyUSDRate: 1,
				SymbolDescription:    "Apple Inc.",
				PriceType:            "limit",
				Status:               3,
				StatusDescription:    "filled",
				StatusDetail:         &StockOrderStatusDetail{Title: "Filled", Message: "Order filled"},
				Side:                 2,
				TimeInForce:          "day",
				Volume:               10,
				FillVolume:           10,
				Price:                200.12,
				AverageFillPrice:     200.10,
				Commission:           2.001,
				TimeSetup:            types.Time(time.Unix(1744110000, 0)),
				TimeDone:             types.Time(time.Unix(1744110100, 0)),
			}},
		}
		assert.Equal(t, exp, result, "GetStockOrderHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetStockOrderHistory should return a result")
}

func TestUpdateStockOrder(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateStockOrder(t.Context(), "", &StockOrderUpdateRequest{Volume: 8, Price: 201.23})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "UpdateStockOrder must require an order ID")

	_, err = e.UpdateStockOrder(t.Context(), "123456", nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "UpdateStockOrder must reject a nil request")

	_, err = e.UpdateStockOrder(t.Context(), "123456", &StockOrderUpdateRequest{})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "UpdateStockOrder must require a volume")

	_, err = e.UpdateStockOrder(t.Context(), "123456", &StockOrderUpdateRequest{Volume: 8})
	require.ErrorIs(t, err, limits.ErrPriceBelowMin, "UpdateStockOrder must require a price")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.UpdateStockOrder(t.Context(), "123456", &StockOrderUpdateRequest{Volume: 8, Price: 201.23})
	require.NoError(t, err, "UpdateStockOrder must not error")
	if mockTests {
		assert.Equal(t, &StockOrderIDResponse{OrderID: 123456}, result, "UpdateStockOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "UpdateStockOrder should return the order ID")
}

func TestCancelStockOrder(t *testing.T) {
	t.Parallel()
	err := e.CancelStockOrder(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelStockOrder must require an order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	require.NoError(t, e.CancelStockOrder(t.Context(), "123456"), "CancelStockOrder must not error")
}

func TestGetStockPositions(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetStockPositions(t.Context(), &GetStockPositionsRequest{PNLCalculationCostType: 1, PNLCalculationPriceType: 2, Symbol: "AAPL", Exchange: "us"})
	require.NoError(t, err, "GetStockPositions must not error")
	if mockTests {
		exp := []*StockPositionResponse{{
			Symbol:                 "AAPL",
			Exchange:               "us",
			QuoteCurrency:          currency.USD,
			QuoteCurrencyPrecision: 2,
			QuoteCurrencyUSDRate:   1,
			TradeStatus:            "open",
			SymbolDescription:      "Apple Inc.",
			PositionPNL:            12.5,
			TodayPNL:               3.2,
			PNLRate:                0.012,
			TodayBuyAmount:         2001.2,
			TodayBuyVolume:         10,
			YesterdayVolume:        8,
			Volume:                 10,
			Available:              8,
			AverageCostPrice:       198.11,
			DilutedCostPrice:       198.11,
			LastPrice:              200.12,
			ExtendedLastPrice:      200.2,
			MaxOrderVolume:         10000,
			StepOrderVolume:        1,
			MinOrderVolume:         1,
			PricePrecision:         2,
			PriceProtection:        0.1,
			SellPriceProtection:    0.1,
			BuyPriceProtection:     0.1,
			CommissionRate:         0.001,
			SlippageRate:           0.001,
		}}
		assert.Equal(t, exp, result, "GetStockPositions should decode every field")
	}
}

func TestCloseStockPosition(t *testing.T) {
	t.Parallel()
	_, err := e.CloseStockPosition(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CloseStockPosition must reject a nil request")

	_, err = e.CloseStockPosition(t.Context(), &StockClosePositionRequest{})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "CloseStockPosition must require a symbol")

	_, err = e.CloseStockPosition(t.Context(), &StockClosePositionRequest{Symbol: "AAPL"})
	require.ErrorIs(t, err, errInvalidTradFiCloseType, "CloseStockPosition must require a close type")

	_, err = e.CloseStockPosition(t.Context(), &StockClosePositionRequest{Symbol: "AAPL", CloseType: 1})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "CloseStockPosition must require a volume for a partial close")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CloseStockPosition(t.Context(), &StockClosePositionRequest{Symbol: "AAPL", CloseType: 1, CloseVolume: 2})
	require.NoError(t, err, "CloseStockPosition must not error")
	if mockTests {
		assert.Equal(t, &StockOrderIDResponse{OrderID: 123457}, result, "CloseStockPosition should decode every field")
		return
	}
	assert.NotNil(t, result, "CloseStockPosition should return the closing order ID")
}

func TestCreateStockTransaction(t *testing.T) {
	t.Parallel()
	err := e.CreateStockTransaction(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreateStockTransaction must reject a nil request")

	err = e.CreateStockTransaction(t.Context(), &StockTransactionRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "CreateStockTransaction must require an asset")

	err = e.CreateStockTransaction(t.Context(), &StockTransactionRequest{Asset: currency.USDT})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "CreateStockTransaction must require a positive amount")

	err = e.CreateStockTransaction(t.Context(), &StockTransactionRequest{Asset: currency.USDT, Change: 100})
	require.ErrorIs(t, err, errTradFiTransactionTypeRequired, "CreateStockTransaction must require a transaction type")

	err = e.CreateStockTransaction(t.Context(), &StockTransactionRequest{Asset: currency.USDT, Change: 100, Type: "deposit"})
	require.ErrorIs(t, err, errStockReferenceIDRequired, "CreateStockTransaction must require a reference ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	err = e.CreateStockTransaction(t.Context(), &StockTransactionRequest{Asset: currency.USDT, Change: 100, Type: "deposit", ReferenceID: "transfer-202607070001"})
	require.NoError(t, err, "CreateStockTransaction must not error")
}

func TestGetStockTransactions(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetStockTransactions(t.Context(), &GetStockTransactionsRequest{BeginTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetStockTransactions must reject a begin time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	exp := &StockTransactionsResponse{
		Total:      1,
		TotalPages: 1,
		List: []*StockTransaction{{
			Asset:             currency.USDT,
			Symbol:            "AAPL",
			SymbolDisplayName: "Apple Inc.",
			Type:              "deposit",
			TypeDescription:   "Deposit",
			Change:            100,
			Balance:           1000,
			ReferenceID:       "transfer-202607070001",
			Time:              types.Time(time.Unix(1744150000, 0)),
			UnitText:          "USDT",
			Detail:            map[string]any{"source": "api"},
		}},
	}
	result, err := e.GetStockTransactions(t.Context(), &GetStockTransactionsRequest{BeginTime: startTime, EndTime: endTime, Type: "deposit", Page: 1, PageSize: 10})
	require.NoError(t, err, "GetStockTransactions must not error")
	if mockTests {
		assert.Equal(t, exp, result, "GetStockTransactions should decode every field")
	} else {
		assert.NotNil(t, result, "GetStockTransactions should return a result")
	}

	result, err = e.GetStockTransactions(t.Context(), &GetStockTransactionsRequest{ReferenceID: "transfer-202607070001"})
	require.NoError(t, err, "GetStockTransactions must not error with a reference ID")
	if mockTests {
		assert.Equal(t, exp, result, "GetStockTransactions should decode every field with a reference ID")
		return
	}
	assert.NotNil(t, result, "GetStockTransactions should return a result with a reference ID")
}

func TestGetStockExchanges(t *testing.T) {
	t.Parallel()
	result, err := e.GetStockExchanges(t.Context())
	require.NoError(t, err, "GetStockExchanges must not error")
	if mockTests {
		exp := []*StockExchangeResponse{
			{Exchange: "us", ExchangeDescription: "US Stocks", IconLink: "https://gimg.staticimgs.com/image/us_20260625_112846_9544f214f56243347961335c9a589a03.png", SupportsStockTransfer: true},
			{Exchange: "hk", ExchangeDescription: "HK Stocks", IconLink: "https://gimg.staticimgs.com/image/hk_20260625_112846_a95ac38da6606a88c4dbc74053724860.png", SupportsStockTransfer: true},
			{Exchange: "kr", ExchangeDescription: "KR Stocks"},
			{Exchange: "jp", ExchangeDescription: "JP Stocks"},
		}
		assert.Equal(t, exp, result, "GetStockExchanges should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetStockExchanges should return exchanges")
}

func TestGetStockFeeRates(t *testing.T) {
	t.Parallel()
	result, err := e.GetStockFeeRates(t.Context())
	require.NoError(t, err, "GetStockFeeRates must not error")
	if mockTests {
		require.Len(t, result, 17, "GetStockFeeRates must return every recorded VIP level")
		assert.Equal(t, &StockFeeRateResponse{VIPLevel: 4, MakerFee: 0.00095, TakerFee: 0.00096}, result[4], "GetStockFeeRates should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetStockFeeRates should return fee rates")
}
