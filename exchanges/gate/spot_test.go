package gate

import (
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestListSpotCurrencies(t *testing.T) {
	t.Parallel()
	result, err := e.ListSpotCurrencies(t.Context())
	require.NoError(t, err, "ListSpotCurrencies must not error")
	if mockTests {
		require.Len(t, result, 5, "ListSpotCurrencies must return every recorded currency")
		exp := &CurrencyInfoResponse{
			Currency:        currency.NewCode("STPT"),
			Name:            "StandardTokenizationProtocol",
			Delisted:        true,
			DepositDisabled: true,
			TradeDisabled:   true,
			Chain:           "ETH",
			Chains:          []*SpotCurrencyChain{{Name: "ETH", Address: "0xDe7D85157d9714EADf595045CC12Ca4A5f3E2aDb", DepositDisabled: true}},
			Category:        []string{},
		}
		assert.Equal(t, exp, result[0], "ListSpotCurrencies should decode every field")
		return
	}
	assert.NotEmpty(t, result, "ListSpotCurrencies should return currencies")
}

func TestGetCurrencyDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetCurrencyDetail(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetCurrencyDetail must reject an empty currency")

	result, err := e.GetCurrencyDetail(t.Context(), currency.BTC)
	require.NoError(t, err, "GetCurrencyDetail must not error")
	if mockTests {
		exp := &CurrencyInfoResponse{
			Currency:    currency.BTC,
			Name:        "Bitcoin",
			Chain:       "BTC",
			Chains:      []*SpotCurrencyChain{{Name: "BTC"}, {Name: "BSC", Address: "0x7130d2a12b9bcbfae4f2634d864a1ee1ce3ead9c"}},
			TotalSupply: "21000000",
			MarketCap:   1353080333764.8,
			Category:    []string{},
		}
		assert.Equal(t, exp, result, "GetCurrencyDetail should decode every field")
		return
	}
	assert.NotNil(t, result, "GetCurrencyDetail should return a result")
}

func TestListSpotCurrencyPairs(t *testing.T) {
	t.Parallel()
	result, err := e.ListSpotCurrencyPairs(t.Context())
	require.NoError(t, err, "ListSpotCurrencyPairs must not error")
	if mockTests {
		require.Len(t, result, 6, "ListSpotCurrencyPairs must return every recorded currency pair")
		exp := &CurrencyPairDetailResponse{
			ID:                            getPair(t, asset.Spot),
			Base:                          currency.BTC,
			BaseName:                      "Aleph.im",
			Quote:                         currency.USDT,
			QuoteName:                     "Tether",
			Fee:                           0.2,
			MinBaseAmount:                 0.01,
			MinQuoteAmount:                3,
			MaxQuoteAmount:                5000000,
			AmountPrecision:               2,
			PricePrecision:                5,
			TradeStatus:                   "tradable",
			SellStart:                     types.Time(time.Unix(1608264000, 0)),
			BuyStart:                      types.Time(time.Unix(1608350400, 0)),
			Type:                          "normal",
			TradeURL:                      "https://www.gate.com/trade/ALEPH_USDT",
			MaximumQuoteRisePercentage:    0.4,
			MaximumQuoteDeclinePercentage: 0.4,
			Slippage:                      0.1,
			MarketOrderMaxStock:           7228044.81,
			MarketOrderMaxMoney:           200000,
		}
		assert.Equal(t, exp, result[0], "ListSpotCurrencyPairs should decode every field")
		return
	}
	assert.NotEmpty(t, result, "ListSpotCurrencyPairs should return currency pairs")
}

func TestGetCurrencyPairDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetCurrencyPairDetail(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetCurrencyPairDetail must reject an empty currency pair")

	unifiedMarket := currency.NewBTCUSD().Format(currency.PairFormat{Uppercase: true, Delimiter: currency.UnderscoreDelimiter})
	for _, tc := range []struct {
		pair currency.Pair
		exp  *CurrencyPairDetailResponse
	}{
		{
			pair: getPair(t, asset.Spot),
			exp: &CurrencyPairDetailResponse{
				ID:                            getPair(t, asset.Spot),
				Base:                          currency.BTC,
				BaseName:                      "Bitcoin",
				Quote:                         currency.USDT,
				QuoteName:                     "Tether",
				Fee:                           0.2,
				MinBaseAmount:                 0.000001,
				MinQuoteAmount:                3,
				MaxBaseAmount:                 100,
				MaxQuoteAmount:                5000000,
				AmountPrecision:               6,
				PricePrecision:                1,
				TradeStatus:                   "tradable",
				Type:                          "normal",
				TradeURL:                      "https://www.gate.com/trade/BTC_USDT",
				MaximumQuoteRisePercentage:    0.08,
				MaximumQuoteDeclinePercentage: 0.08,
				Slippage:                      0.03,
				MarketOrderMaxStock:           57.94,
				MarketOrderMaxMoney:           5000000,
			},
		},
		{
			// Unified markets list the quote currencies they accept in trade_quotes
			pair: unifiedMarket,
			exp: &CurrencyPairDetailResponse{
				ID:                            unifiedMarket,
				Base:                          currency.BTC,
				BaseName:                      "Bitcoin",
				Quote:                         currency.USD,
				QuoteName:                     "US Dollar",
				TradeQuotes:                   []currency.Code{currency.NewCode("RLUSD"), currency.USD, currency.NewCode("USD1"), currency.USDC},
				Fee:                           0.2,
				MinBaseAmount:                 0.000001,
				MinQuoteAmount:                1,
				MaxBaseAmount:                 100,
				MaxQuoteAmount:                5000000,
				AmountPrecision:               6,
				PricePrecision:                1,
				TradeStatus:                   "tradable",
				Type:                          "normal",
				TradeURL:                      "https://www.gate.com/trade/BTC_USD",
				MaximumQuoteRisePercentage:    0.25,
				MaximumQuoteDeclinePercentage: 0.25,
				Slippage:                      0.06,
				MarketOrderMaxStock:           1,
				MarketOrderMaxMoney:           100000,
			},
		},
	} {
		t.Run(tc.pair.String(), func(t *testing.T) {
			t.Parallel()
			result, err := e.GetCurrencyPairDetail(t.Context(), tc.pair)
			require.NoError(t, err, "GetCurrencyPairDetail must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetCurrencyPairDetail should decode every field")
				return
			}
			assert.Equal(t, tc.pair, result.ID, "GetCurrencyPairDetail should return the requested currency pair")
		})
	}
}

func TestGetTickers(t *testing.T) {
	t.Parallel()
	_, err := e.GetTickers(t.Context(), getPair(t, asset.Spot), "utc2")
	require.ErrorIs(t, err, errInvalidTimezone, "GetTickers must reject an unsupported timezone")

	for _, tc := range []struct {
		timezone string
		exp      *TickerResponse
	}{
		{
			exp: &TickerResponse{
				CurrencyPair:     getPair(t, asset.Spot),
				Last:             67502.7,
				LowestAsk:        67502.7,
				LowestSize:       9.03203,
				HighestBid:       67502.6,
				HighestSize:      12.1675,
				ChangePercentage: 0.37,
				BaseVolume:       4322.4404579762,
				QuoteVolume:      290264968.57010812722,
				High24Hour:       67842.2,
				Low24Hour:        66610,
			},
		},
		{
			// Only the all timezone returns both the UTC+0 and UTC+8 changes
			timezone: allTimeZones,
			exp: &TickerResponse{
				CurrencyPair:     getPair(t, asset.Spot),
				Last:             82177.8,
				LowestAsk:        82170.4,
				LowestSize:       2.799905,
				HighestBid:       82170.3,
				HighestSize:      4.246692,
				ChangePercentage: -1.89,
				ChangeUTC0:       0.4,
				ChangeUTC8:       -0.66,
				BaseVolume:       8949.156124,
				QuoteVolume:      742123076.3466114,
				High24Hour:       84352.2,
				Low24Hour:        81040.5,
			},
		},
	} {
		t.Run("timezone "+tc.timezone, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetTickers(t.Context(), getPair(t, asset.Spot), tc.timezone)
			require.NoError(t, err, "GetTickers must not error")
			require.Len(t, result, 1, "GetTickers must return one ticker for a currency pair")
			if mockTests {
				assert.Equal(t, tc.exp, result[0], "GetTickers should decode every field")
				return
			}
			assert.Equal(t, getPair(t, asset.Spot), result[0].CurrencyPair, "GetTickers should return the requested currency pair")
		})
	}

	result, err := e.GetTickers(t.Context(), currency.EMPTYPAIR, "")
	require.NoError(t, err, "GetTickers must not error for every currency pair")
	if mockTests {
		require.Len(t, result, 5, "GetTickers must return every recorded ticker")
		exp := &TickerResponse{
			CurrencyPair:     currency.NewPairWithDelimiter("SUSHI3L", "USDT", currency.UnderscoreDelimiter),
			Last:             0.00113,
			LowestAsk:        0.001127,
			HighestBid:       0.001124,
			ChangePercentage: -8.27,
			BaseVolume:       10218896.34,
			QuoteVolume:      11910.21673754,
			High24Hour:       0.001256,
			Low24Hour:        0.001104,
			ETFNetValue:      0.00113387,
			ETFPreNetValue:   0.00115756,
			ETFPreTimestamp:  types.Time(time.Unix(1775404800, 0)),
			ETFLeverage:      3.041778253883235,
		}
		assert.Equal(t, exp, result[2], "GetTickers should decode leveraged token fields")
		return
	}
	assert.NotEmpty(t, result, "GetTickers should return tickers")
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := e.GetTicker(t.Context(), currency.EMPTYPAIR, utc8TimeZone)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetTicker must reject an empty currency pair")

	_, err = newFixedResponseExchange(t, "/api/v4/spot/tickers", `[]`).GetTicker(t.Context(), currency.NewBTCUSDT(), "")
	require.ErrorIs(t, err, errNoTickerData, "GetTicker must reject an empty ticker list")

	result, err := e.GetTicker(t.Context(), getPair(t, asset.Spot), utc8TimeZone)
	require.NoError(t, err, "GetTicker must not error")
	if mockTests {
		exp := &TickerResponse{
			CurrencyPair:     getPair(t, asset.Spot),
			Last:             67197,
			LowestAsk:        67198.1,
			LowestSize:       0.316979,
			HighestBid:       67198,
			HighestSize:      12.696796,
			ChangePercentage: -0.64,
			BaseVolume:       10980.383815,
			QuoteVolume:      737010383.3594575,
			High24Hour:       68405.5,
			Low24Hour:        65996.1,
		}
		assert.Equal(t, exp, result, "GetTicker should decode every field")
		return
	}
	assert.Equal(t, getPair(t, asset.Spot), result.CurrencyPair, "GetTicker should return the requested currency pair")
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderbook(t.Context(), currency.EMPTYPAIR, "0.1", 10, false)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetOrderbook must reject an empty currency pair")

	result, err := e.GetOrderbook(t.Context(), getPair(t, asset.Spot), "0.1", 10, false)
	require.NoError(t, err, "GetOrderbook must not error")
	if mockTests {
		exp := &OrderbookResponse{
			ResponseTime: types.Time(time.UnixMilli(1774968317360)),
			UpdateTime:   types.Time(time.UnixMilli(1774968317359)),
			Asks: OrderbookLevels{
				{Price: 67197.1, Amount: 0.485259},
				{Price: 67197.9, Amount: 0.016632},
				{Price: 67198, Amount: 0.0148},
				{Price: 67199, Amount: 0.0155},
				{Price: 67199.1, Amount: 0.024881},
			},
			Bids: OrderbookLevels{
				{Price: 67197, Amount: 0.442669},
				{Price: 67196.9, Amount: 0.01},
				{Price: 67196.7, Amount: 0.001547},
				{Price: 67196.1, Amount: 0.012274},
				{Price: 67196, Amount: 0.025678},
			},
		}
		assert.Equal(t, exp, result, "GetOrderbook should decode every field")
		return
	}
	assert.NotEmpty(t, result.Asks, "GetOrderbook should return asks")
}

func TestGetMarketTrades(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetMarketTrades(t.Context(), getPair(t, asset.Spot), "", true, endTime, startTime, 1, 0)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetMarketTrades must reject a start time after the end time")

	_, err = e.GetMarketTrades(t.Context(), currency.EMPTYPAIR, "", true, startTime, endTime, 1, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetMarketTrades must reject an empty currency pair")

	for _, tc := range []struct {
		name     string
		from, to time.Time
		limit    uint64
	}{
		{name: "latest"},
		{name: "time range", from: startTime, to: endTime, limit: 100},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetMarketTrades(t.Context(), getPair(t, asset.Spot), "", true, tc.from, tc.to, 1, tc.limit)
			require.NoError(t, err, "GetMarketTrades must not error")
			if mockTests {
				require.Len(t, result, 5, "GetMarketTrades must return every recorded trade")
				exp := &TradeResponse{
					ID:                "192742010",
					CreateTimeSeconds: types.Time(time.Unix(1775423972, 0)),
					CreateTime:        types.Time(time.Unix(0, 1775423972061432000)),
					CurrencyPair:      "BTC_USDT",
					Side:              "buy",
					Amount:            0.029077,
					Price:             67502.7,
					SequenceID:        "192742010",
				}
				assert.Equal(t, exp, result[0], "GetMarketTrades should decode every field")
				return
			}
			assert.NotNil(t, result, "GetMarketTrades should return a result")
		})
	}
}

func TestCandlestickUnmarshalJSON(t *testing.T) {
	t.Parallel()
	data := []byte(`[["1738108800","229534412.73508700","103734.3","104779.9","101336.6","101343.8","2232.94510000","true"],["1738195200","178316032.62306100","104718.6","106467.1","103286.4","103734.4","1695.00787000","true"],["1738281600","231315376.16747100","102431","106042.7","101555.9","104718.6","2228.03609000","true"]]`)
	var targets []CandlestickResponse
	err := json.Unmarshal(data, &targets)
	require.NoError(t, err)
	require.Len(t, targets, 3)
	assert.Equal(t, CandlestickResponse{
		Timestamp:           types.Time(time.Unix(1738108800, 0)),
		QuoteCurrencyVolume: 229534412.73508700,
		ClosePrice:          103734.3,
		HighestPrice:        104779.9,
		LowestPrice:         101336.6,
		OpenPrice:           101343.8,
		BaseCurrencyAmount:  2232.94510000,
		WindowClosed:        true,
	}, targets[0])
}

func TestGetCandlesticks(t *testing.T) {
	t.Parallel()
	_, err := e.GetCandlesticks(t.Context(), currency.EMPTYPAIR, 0, time.Time{}, time.Time{}, kline.OneDay)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetCandlesticks must reject an empty currency pair")

	_, err = e.GetCandlesticks(t.Context(), getPair(t, asset.Spot), 0, time.Time{}, time.Time{}, kline.FiveDay)
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval, "GetCandlesticks must reject an unsupported interval")

	result, err := e.GetCandlesticks(t.Context(), getPair(t, asset.Spot), 0, time.Time{}, time.Time{}, kline.OneDay)
	require.NoError(t, err, "GetCandlesticks must not error")
	if mockTests {
		require.Len(t, result, 5, "GetCandlesticks must return every recorded candle")
		exp := &CandlestickResponse{
			Timestamp:           types.Time(time.Unix(1766361600, 0)),
			QuoteCurrencyVolume: 901846017.17068550,
			ClosePrice:          88622.1,
			HighestPrice:        90586.3,
			LowestPrice:         87899.6,
			OpenPrice:           88655.4,
			BaseCurrencyAmount:  10108.46374900,
			WindowClosed:        true,
		}
		assert.Equal(t, exp, result[0], "GetCandlesticks should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetCandlesticks should return candles")
}

func TestGetAccountBatchFeeRates(t *testing.T) {
	t.Parallel()
	_, err := e.GetAccountBatchFeeRates(t.Context(), nil)
	require.ErrorIs(t, err, currency.ErrCurrencyPairsEmpty, "GetAccountBatchFeeRates must reject empty currency pairs")

	_, err = e.GetAccountBatchFeeRates(t.Context(), make([]string, 51))
	require.ErrorIs(t, err, errTooManyCurrencyPairs, "GetAccountBatchFeeRates must reject more than 50 currency pairs")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAccountBatchFeeRates(t.Context(), []string{"BTC_USDT", "ETH_USDT"})
	require.NoError(t, err, "GetAccountBatchFeeRates must not error")
	if mockTests {
		exp := map[string]*SpotTradingFeeRateResponse{
			"BTC_USDT": {
				CurrencyPair:                   getPair(t, asset.Spot),
				UserID:                         10001,
				TakerFee:                       0.002,
				MakerFee:                       0.002,
				RetailPriceImprovementMakerFee: -0.00175,
				LoanFee:                        0.18,
				PointType:                      "1",
				DebitFee:                       3,
				RPIMMLevel:                     2,
			},
			"ETH_USDT": {
				CurrencyPair:                   currency.NewPairWithDelimiter("ETH", "USDT", currency.UnderscoreDelimiter),
				UserID:                         10001,
				TakerFee:                       0.001,
				MakerFee:                       0.0009,
				RetailPriceImprovementMakerFee: -0.00175,
				GTDiscount:                     true,
				GTTakerFee:                     0.0008,
				GTMakerFee:                     0.0007,
				LoanFee:                        0.18,
				PointType:                      "1",
				DebitFee:                       1,
			},
		}
		assert.Equal(t, exp, result, "GetAccountBatchFeeRates should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetAccountBatchFeeRates should return fee rates")
}

func TestGetSpotAccounts(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSpotAccounts(t.Context(), currency.BTC)
	require.NoError(t, err, "GetSpotAccounts must not error")
	if mockTests {
		exp := []*SpotAccountResponse{{Currency: currency.BTC, Available: 968.8, Locked: 0.5, UpdateID: 98}}
		assert.Equal(t, exp, result, "GetSpotAccounts should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSpotAccounts should return a result")
}

func TestGetSpotAccountBook(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpotAccountBook(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSpotAccountBook must reject a nil request")

	startTime, endTime := getTime()
	_, err = e.GetSpotAccountBook(t.Context(), &GetSpotAccountBookRequest{From: endTime.Add(-time.Hour * 24 * 31), To: endTime})
	require.ErrorIs(t, err, errTimeRangeTooLong, "GetSpotAccountBook must reject a range longer than 30 days")

	_, err = e.GetSpotAccountBook(t.Context(), &GetSpotAccountBookRequest{From: endTime, To: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetSpotAccountBook must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSpotAccountBook(t.Context(), &GetSpotAccountBookRequest{Currency: currency.BTC, From: startTime, To: endTime, Page: 1, Limit: 10})
	require.NoError(t, err, "GetSpotAccountBook must not error")
	if mockTests {
		exp := []*SpotAccountBookResponse{{
			ID:       "123456",
			Time:     types.Time(time.UnixMilli(1744150000123)),
			Currency: currency.BTC,
			Change:   1.03,
			Balance:  4.59316525194,
			Type:     "margin_in",
			Code:     "101",
			Text:     "3815099",
		}}
		assert.Equal(t, exp, result, "GetSpotAccountBook should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSpotAccountBook should return a result")
}

func TestCreateBatchOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CreateBatchOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrEmptyParams, "CreateBatchOrders must reject an empty batch")

	arg := CreateOrderRequest{
		Side:    order.Sell.Lower(),
		Amount:  0.001,
		Price:   12349,
		Account: asset.Spot,
		Type:    "limit",
	}
	_, err = e.CreateBatchOrders(t.Context(), []CreateOrderRequest{arg})
	require.ErrorIs(t, err, order.ErrClientOrderIDMustBeSet, "CreateBatchOrders must reject an order without text")

	arg.Text = "1234"
	_, err = e.CreateBatchOrders(t.Context(), []CreateOrderRequest{arg})
	require.ErrorIs(t, err, errInvalidTextPrefix, "CreateBatchOrders must reject text without the t- prefix")

	arg.Text = "t-1234"
	_, err = e.CreateBatchOrders(t.Context(), []CreateOrderRequest{arg})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CreateBatchOrders must reject an empty currency pair")

	arg.CurrencyPair = getPair(t, asset.Spot)
	_, err = e.CreateBatchOrders(t.Context(), []CreateOrderRequest{arg, {Text: "t-1235", CurrencyPair: arg.CurrencyPair, Side: arg.Side, Amount: 1, Price: 1, Account: asset.Margin}})
	require.ErrorIs(t, err, errDifferentAccount, "CreateBatchOrders must reject orders from different accounts")

	invalidSide := arg
	invalidSide.Side = "short"
	_, err = e.CreateBatchOrders(t.Context(), []CreateOrderRequest{invalidSide})
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "CreateBatchOrders must reject a side other than buy or sell")

	invalidAccount := arg
	invalidAccount.Account = asset.Options
	_, err = e.CreateBatchOrders(t.Context(), []CreateOrderRequest{invalidAccount})
	require.ErrorIs(t, err, asset.ErrInvalidAsset, "CreateBatchOrders must reject an account other than spot, margin or cross margin")

	zeroAmount := arg
	zeroAmount.Amount = 0
	_, err = e.CreateBatchOrders(t.Context(), []CreateOrderRequest{zeroAmount})
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "CreateBatchOrders must reject an order without an amount")

	limitWithoutPrice := arg
	limitWithoutPrice.Price = 0
	_, err = e.CreateBatchOrders(t.Context(), []CreateOrderRequest{limitWithoutPrice})
	require.ErrorIs(t, err, limits.ErrPriceBelowMin, "CreateBatchOrders must reject a limit order without a price")

	_, err = e.CreateBatchOrders(t.Context(), slices.Repeat([]CreateOrderRequest{arg}, 11))
	require.ErrorIs(t, err, errTooManyBatchOrders, "CreateBatchOrders must reject more than 10 orders on a currency pair")

	fivePairs := make([]CreateOrderRequest, 5)
	for i, base := range []currency.Code{currency.BTC, currency.ETH, currency.LTC, currency.XRP, currency.SOL} {
		fivePairs[i] = arg
		fivePairs[i].CurrencyPair = currency.NewPairWithDelimiter(base.String(), "USDT", currency.UnderscoreDelimiter)
	}
	_, err = e.CreateBatchOrders(t.Context(), fivePairs)
	require.ErrorIs(t, err, errTooManyBatchOrders, "CreateBatchOrders must reject more than 4 currency pairs")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CreateBatchOrders(t.Context(), []CreateOrderRequest{
		{
			Text:         "t-123123123",
			CurrencyPair: getPair(t, asset.Spot),
			Side:         order.Sell.Lower(),
			Amount:       0.001,
			Price:        12349,
			Account:      asset.Spot,
			Type:         "limit",
		},
		{
			Text:         "t-123123124",
			CurrencyPair: getPair(t, asset.Spot),
			Side:         order.Buy.Lower(),
			Amount:       1,
			Price:        1234567789,
			Account:      asset.Spot,
			Type:         "limit",
		},
	})
	require.NoError(t, err, "CreateBatchOrders must not error")
	if mockTests {
		exp := []*SpotOrderResponse{
			{
				OrderID:            "12332324",
				BatchOrderID:       "12332324",
				Text:               "t-123123123",
				AmendText:          "-",
				Succeeded:          true,
				CreateTimeSeconds:  types.Time(time.Unix(1744150000, 0)),
				UpdateTimeSeconds:  types.Time(time.Unix(1744150001, 0)),
				CreateTime:         types.Time(time.UnixMilli(1744150000123)),
				UpdateTime:         types.Time(time.UnixMilli(1744150001123)),
				CurrencyPair:       "BTC_USDT",
				Status:             statusOpen,
				Type:               "limit",
				Account:            "spot",
				Side:               "sell",
				Amount:             0.001,
				Price:              12349,
				TimeInForce:        gtcTIF,
				RemainingAmount:    0.001,
				FeeCurrency:        currency.USDT,
				RebatedFeeCurrency: currency.BTC,
				STPID:              10240,
				STPAction:          "cn",
				FinishAs:           "open",
			},
			{
				Text:       "t-123123124",
				ErrorLabel: "BALANCE_NOT_ENOUGH",
				Message:    "Not enough balance",
			},
		}
		assert.Equal(t, exp, result, "CreateBatchOrders should decode every field")
		return
	}
	assert.NotEmpty(t, result, "CreateBatchOrders should return order results")
}

func TestValidateSpotOrderPrice(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		orderType string
		price     float64
		err       error
	}{
		{orderType: "limit", price: 1},
		{orderType: "limit", err: limits.ErrPriceBelowMin},
		{err: limits.ErrPriceBelowMin}, // Gate defaults to a limit order
		{orderType: "market"},
		{orderType: "market", price: -1, err: limits.ErrPriceBelowMin},
	} {
		err := validateSpotOrderPrice(tc.orderType, tc.price)
		if tc.err != nil {
			assert.ErrorIsf(t, err, tc.err, "validateSpotOrderPrice should reject %q orders priced %v", tc.orderType, tc.price)
			continue
		}
		assert.NoErrorf(t, err, "validateSpotOrderPrice should accept %q orders priced %v", tc.orderType, tc.price)
	}
}

func TestGetSpotOpenOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSpotOpenOrders(t.Context(), 0, 0, "")
	require.NoError(t, err, "GetSpotOpenOrders must not error")
	if mockTests {
		exp := []*SpotOrdersDetailResponse{{
			CurrencyPair: getPair(t, asset.Spot),
			Total:        1,
			Orders: []*SpotOrderResponse{{
				OrderID:            "12332324",
				Text:               "t-123456",
				AmendText:          "-",
				CreateTimeSeconds:  types.Time(time.Unix(1744150000, 0)),
				UpdateTimeSeconds:  types.Time(time.Unix(1744150100, 0)),
				CreateTime:         types.Time(time.UnixMilli(1744150000123)),
				UpdateTime:         types.Time(time.UnixMilli(1744150100123)),
				CurrencyPair:       "BTC_USDT",
				Status:             statusOpen,
				Type:               "limit",
				Account:            "spot",
				Side:               "buy",
				Amount:             1,
				Price:              5.00032,
				TimeInForce:        gtcTIF,
				RemainingAmount:    0.5,
				FilledAmount:       0.5,
				AverageFillPrice:   5.00032,
				FeeDeducted:        0.005,
				FeeCurrency:        currency.BTC,
				FilledTotal:        2.50016,
				RebatedFeeCurrency: currency.USDT,
				STPAction:          "-",
				FinishAs:           "open",
			}},
		}}
		assert.Equal(t, exp, result, "GetSpotOpenOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSpotOpenOrders should return a result")
}

func TestSpotClosePositionWhenCrossCurrencyDisabled(t *testing.T) {
	t.Parallel()
	_, err := e.SpotClosePositionWhenCrossCurrencyDisabled(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SpotClosePositionWhenCrossCurrencyDisabled must reject a nil request")

	_, err = e.SpotClosePositionWhenCrossCurrencyDisabled(t.Context(), &ClosePositionRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "SpotClosePositionWhenCrossCurrencyDisabled must reject an empty currency pair")

	_, err = e.SpotClosePositionWhenCrossCurrencyDisabled(t.Context(), &ClosePositionRequest{CurrencyPair: getPair(t, asset.Spot)})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "SpotClosePositionWhenCrossCurrencyDisabled must reject an empty amount")

	_, err = e.SpotClosePositionWhenCrossCurrencyDisabled(t.Context(), &ClosePositionRequest{Amount: 0.1, CurrencyPair: getPair(t, asset.Spot)})
	require.ErrorIs(t, err, limits.ErrPriceBelowMin, "SpotClosePositionWhenCrossCurrencyDisabled must reject an empty price")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SpotClosePositionWhenCrossCurrencyDisabled(t.Context(), &ClosePositionRequest{
		Amount:       0.1,
		Price:        1234567384,
		CurrencyPair: getPair(t, asset.Spot),
	})
	require.NoError(t, err, "SpotClosePositionWhenCrossCurrencyDisabled must not error")
	if mockTests {
		exp := &SpotOrderResponse{
			OrderID:            "1852454420",
			Text:               "apiv4",
			AmendText:          "-",
			CreateTimeSeconds:  types.Time(time.Unix(1744150000, 0)),
			UpdateTimeSeconds:  types.Time(time.Unix(1744150000, 0)),
			CreateTime:         types.Time(time.UnixMilli(1744150000073)),
			UpdateTime:         types.Time(time.UnixMilli(1744150000074)),
			CurrencyPair:       "BTC_USDT",
			Status:             statusOpen,
			Type:               "limit",
			Account:            crossMarginAccount,
			Side:               "buy",
			Amount:             0.1,
			Price:              1234567384,
			TimeInForce:        gtcTIF,
			AutoRepay:          true,
			RemainingAmount:    0.1,
			FeeCurrency:        currency.BTC,
			RebatedFeeCurrency: currency.USDT,
			STPAction:          "-",
			FinishAs:           "open",
		}
		assert.Equal(t, exp, result, "SpotClosePositionWhenCrossCurrencyDisabled should decode every field")
		return
	}
	assert.NotNil(t, result, "SpotClosePositionWhenCrossCurrencyDisabled should return a result")
}

func TestPlaceSpotOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceSpotOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "PlaceSpotOrder must reject a nil request")

	arg := &CreateOrderRequest{Type: "limit"}
	_, err = e.PlaceSpotOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "PlaceSpotOrder must reject an empty currency pair")

	arg.CurrencyPair = getPair(t, asset.Spot)
	_, err = e.PlaceSpotOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "PlaceSpotOrder must reject an invalid side")

	arg.Side = order.Buy.Lower()
	_, err = e.PlaceSpotOrder(t.Context(), arg)
	require.ErrorIs(t, err, asset.ErrInvalidAsset, "PlaceSpotOrder must reject an unsupported account")

	arg.Account = asset.Spot
	_, err = e.PlaceSpotOrder(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "PlaceSpotOrder must reject an empty amount")

	arg.Amount = 1
	arg.Price = -1
	_, err = e.PlaceSpotOrder(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrPriceBelowMin, "PlaceSpotOrder must reject a negative price")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.PlaceSpotOrder(t.Context(), &CreateOrderRequest{
		CurrencyPair: getPair(t, asset.Spot),
		Side:         order.Buy.Lower(),
		Amount:       1,
		Price:        900000,
		Account:      asset.Spot,
		Type:         "limit",
	})
	require.NoError(t, err, "PlaceSpotOrder must not error")
	if mockTests {
		exp := &SpotOrderResponse{
			OrderID:            "1852454420",
			Text:               "apiv4",
			AmendText:          "-",
			CreateTimeSeconds:  types.Time(time.Unix(1744150000, 0)),
			UpdateTimeSeconds:  types.Time(time.Unix(1744150000, 0)),
			CreateTime:         types.Time(time.UnixMilli(1744150000073)),
			UpdateTime:         types.Time(time.UnixMilli(1744150000074)),
			CurrencyPair:       "BTC_USDT",
			Status:             statusOpen,
			Type:               "limit",
			Account:            "spot",
			Side:               "buy",
			Amount:             1,
			Price:              900000,
			TimeInForce:        gtcTIF,
			RemainingAmount:    1,
			FeeCurrency:        currency.BTC,
			RebatedFeeCurrency: currency.USDT,
			STPAction:          "-",
			FinishAs:           "open",
			Slippage:           0.05,
		}
		assert.Equal(t, exp, result, "PlaceSpotOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "PlaceSpotOrder should return a result")
}

func TestGetSpotOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpotOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSpotOrders must reject a nil request")

	_, err = e.GetSpotOrders(t.Context(), &GetSpotOrdersRequest{CurrencyPair: getPair(t, asset.Spot)})
	require.ErrorIs(t, err, errInvalidOrderStatus, "GetSpotOrders must reject an empty status")

	_, err = e.GetSpotOrders(t.Context(), &GetSpotOrdersRequest{Status: statusOpen})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetSpotOrders must reject open orders without a currency pair")

	_, err = e.GetSpotOrders(t.Context(), &GetSpotOrdersRequest{Status: statusFinished, Side: order.Long})
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "GetSpotOrders must reject an invalid side")

	startTime, endTime := getTime()
	_, err = e.GetSpotOrders(t.Context(), &GetSpotOrdersRequest{Status: statusFinished, From: endTime, To: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetSpotOrders must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSpotOrders(t.Context(), &GetSpotOrdersRequest{CurrencyPair: getPair(t, asset.Spot), Status: statusOpen, Page: 100, Limit: 100})
	require.NoError(t, err, "GetSpotOrders must not error")
	if mockTests {
		exp := []*SpotOrderResponse{{
			OrderID:            "1852454420",
			Text:               "t-abc123",
			AmendText:          "-",
			CreateTimeSeconds:  types.Time(time.Unix(1744150000, 0)),
			UpdateTimeSeconds:  types.Time(time.Unix(1744150000, 0)),
			CreateTime:         types.Time(time.UnixMilli(1744150000073)),
			UpdateTime:         types.Time(time.UnixMilli(1744150000074)),
			CurrencyPair:       "BTC_USDT",
			TradeQuote:         currency.USDT,
			Status:             statusOpen,
			Type:               "limit",
			Account:            "spot",
			Side:               "buy",
			Amount:             0.001,
			Price:              65000,
			TimeInForce:        gtcTIF,
			RemainingAmount:    0.001,
			FeeCurrency:        currency.BTC,
			RebatedFeeCurrency: currency.USDT,
			FinishAs:           "open",
			StopProfit:         &SpotOrderTPSL{TriggerPrice: 67000, OrderPrice: 67000},
			StopLoss:           &SpotOrderTPSL{TriggerPrice: 63000, OrderPrice: 63000},
		}}
		assert.Equal(t, exp, result, "GetSpotOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSpotOrders should return a result")
}

func TestCancelAllOpenOrdersSpecifiedCurrencyPair(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllOpenOrdersSpecifiedCurrencyPair(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CancelAllOpenOrdersSpecifiedCurrencyPair must reject a nil request")

	_, err = e.CancelAllOpenOrdersSpecifiedCurrencyPair(t.Context(), &CancelAllSpotOrdersRequest{CurrencyPair: getPair(t, asset.Spot), Side: order.Long, Account: "spot"})
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "CancelAllOpenOrdersSpecifiedCurrencyPair must reject an invalid side")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	cancelledBuy := []*SpotOrderResponse{{
		OrderID:            "1852454420",
		Text:               "t-abc123",
		AmendText:          "-",
		Succeeded:          true,
		CreateTimeSeconds:  types.Time(time.Unix(1744150000, 0)),
		UpdateTimeSeconds:  types.Time(time.Unix(1744150300, 0)),
		CreateTime:         types.Time(time.UnixMilli(1744150000073)),
		UpdateTime:         types.Time(time.UnixMilli(1744150300074)),
		CurrencyPair:       "BTC_USDT",
		Status:             "cancelled",
		Type:               "limit",
		Account:            "spot",
		Side:               "buy",
		Amount:             0.001,
		Price:              65000,
		TimeInForce:        gtcTIF,
		RemainingAmount:    0.001,
		FeeCurrency:        currency.BTC,
		RebatedFeeCurrency: currency.USDT,
		FinishAs:           "cancelled",
	}}
	unifiedMarket := currency.NewBTCUSD().Format(currency.PairFormat{Uppercase: true, Delimiter: currency.UnderscoreDelimiter})
	for _, tc := range []struct {
		name string
		arg  *CancelAllSpotOrdersRequest
		exp  []*SpotOrderResponse
	}{
		{name: "buy", arg: &CancelAllSpotOrdersRequest{CurrencyPair: getPair(t, asset.Spot), Side: order.Buy, Account: "spot"}, exp: cancelledBuy},
		{name: "sell", arg: &CancelAllSpotOrdersRequest{CurrencyPair: getPair(t, asset.Spot), Side: order.Sell, Account: "spot"}, exp: []*SpotOrderResponse{}},
		// UnknownSide and AnySide omit the optional side filter so Gate cancels both bids and asks
		{name: "unknown side", arg: &CancelAllSpotOrdersRequest{CurrencyPair: getPair(t, asset.Spot), Account: "spot"}, exp: []*SpotOrderResponse{}},
		{name: "any side", arg: &CancelAllSpotOrdersRequest{CurrencyPair: getPair(t, asset.Spot), Side: order.AnySide, Account: "spot"}, exp: []*SpotOrderResponse{}},
		{
			name: "unified market quote",
			arg:  &CancelAllSpotOrdersRequest{CurrencyPair: unifiedMarket, Account: "unified", TradeQuote: currency.USDC, ActionMode: "ACK"},
			exp:  []*SpotOrderResponse{{OrderID: "1852454421", Text: "t-abc124", AmendText: "-", Succeeded: true}},
		},
	} {
		result, err := e.CancelAllOpenOrdersSpecifiedCurrencyPair(t.Context(), tc.arg)
		require.NoErrorf(t, err, "CancelAllOpenOrdersSpecifiedCurrencyPair must not error for %s", tc.name)
		if mockTests {
			assert.Equalf(t, tc.exp, result, "CancelAllOpenOrdersSpecifiedCurrencyPair should decode every field for %s", tc.name)
		}
	}
}

func TestCancelBatchOrdersWithIDList(t *testing.T) {
	t.Parallel()
	_, err := e.CancelBatchOrdersWithIDList(t.Context(), []CancelOrderByIDRequest{})
	require.ErrorIs(t, err, errNoValidParameterPassed, "CancelBatchOrdersWithIDList must reject an empty batch")

	_, err = e.CancelBatchOrdersWithIDList(t.Context(), make([]CancelOrderByIDRequest, 21))
	require.ErrorIs(t, err, errInvalidOrderSize, "CancelBatchOrdersWithIDList must reject more than 20 orders")

	_, err = e.CancelBatchOrdersWithIDList(t.Context(), []CancelOrderByIDRequest{{ID: "1234567"}})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CancelBatchOrdersWithIDList must reject an empty currency pair")

	_, err = e.CancelBatchOrdersWithIDList(t.Context(), []CancelOrderByIDRequest{{CurrencyPair: getPair(t, asset.Spot)}})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelBatchOrdersWithIDList must reject an empty order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelBatchOrdersWithIDList(t.Context(), []CancelOrderByIDRequest{
		{
			CurrencyPair: getPair(t, asset.Spot),
			ID:           "1234567",
		},
		{
			CurrencyPair: getPair(t, asset.Spot),
			ID:           "123453",
		},
	})
	require.NoError(t, err, "CancelBatchOrdersWithIDList must not error")
	if mockTests {
		exp := []*CancelOrderByIDResponse{
			{CurrencyPair: getPair(t, asset.Spot), OrderID: "1234567", Text: "t-1234567", Succeeded: true},
			{CurrencyPair: getPair(t, asset.Spot), OrderID: "123453", Label: "ORDER_NOT_FOUND", Message: "Order not found"},
		}
		assert.Equal(t, exp, result, "CancelBatchOrdersWithIDList should decode every field")
		return
	}
	assert.NotEmpty(t, result, "CancelBatchOrdersWithIDList should return order results")
}

func TestGetSpotOrder(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpotOrder(t.Context(), "", getPair(t, asset.Spot), asset.Spot)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetSpotOrder must reject an empty order ID")

	_, err = e.GetSpotOrder(t.Context(), "1234", currency.EMPTYPAIR, asset.Spot)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetSpotOrder must reject an empty currency pair")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSpotOrder(t.Context(), "1234", getPair(t, asset.Spot), asset.Spot)
	require.NoError(t, err, "GetSpotOrder must not error")
	if mockTests {
		exp := &SpotOrderResponse{
			OrderID:            "1234",
			Text:               "t-abc123",
			AmendText:          "-",
			CreateTimeSeconds:  types.Time(time.Unix(1744150000, 0)),
			UpdateTimeSeconds:  types.Time(time.Unix(1744150000, 0)),
			CreateTime:         types.Time(time.UnixMilli(1744150000073)),
			UpdateTime:         types.Time(time.UnixMilli(1744150000074)),
			CurrencyPair:       "BTC_USDT",
			TradeQuote:         currency.USDT,
			Status:             "closed",
			Type:               "limit",
			Account:            "spot",
			Side:               "buy",
			Amount:             0.001,
			Price:              65000,
			TimeInForce:        gtcTIF,
			FilledAmount:       0.001,
			AverageFillPrice:   63469.3,
			FeeDeducted:        0.00000022,
			FeeCurrency:        currency.BTC,
			FillPrice:          63.4693,
			FilledTotal:        63.4693,
			RebatedFeeCurrency: currency.USDT,
			FinishAs:           "filled",
			StopProfit:         &SpotOrderTPSL{TriggerPrice: 67000, OrderPrice: 67000},
			StopLoss:           &SpotOrderTPSL{TriggerPrice: 63000, OrderPrice: 63000},
		}
		assert.Equal(t, exp, result, "GetSpotOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSpotOrder should return a result")
}

func TestAmendSpotOrder(t *testing.T) {
	t.Parallel()
	_, err := e.AmendSpotOrder(t.Context(), "", &AmendSpotOrderRequest{CurrencyPair: getPair(t, asset.Spot), Price: 1000})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "AmendSpotOrder must reject an empty order ID")

	_, err = e.AmendSpotOrder(t.Context(), "123", nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "AmendSpotOrder must reject a nil request")

	_, err = e.AmendSpotOrder(t.Context(), "123", &AmendSpotOrderRequest{Price: 1000})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "AmendSpotOrder must reject an empty currency pair")

	_, err = e.AmendSpotOrder(t.Context(), "123", &AmendSpotOrderRequest{CurrencyPair: getPair(t, asset.Spot)})
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "AmendSpotOrder must reject a request without amount or price")
	require.ErrorIs(t, err, limits.ErrPriceBelowMin, "AmendSpotOrder must reject a request without amount or price")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	// An empty stop loss cancels the existing one while a nil take profit or stop loss would leave it unchanged
	result, err := e.AmendSpotOrder(t.Context(), "123", &AmendSpotOrderRequest{
		CurrencyPair: getPair(t, asset.Spot),
		Price:        1000,
		StopProfit:   &SpotOrderTPSL{TriggerPrice: 67000, OrderPrice: 67000},
		StopLoss:     &SpotOrderTPSL{},
	})
	require.NoError(t, err, "AmendSpotOrder must not error")
	if mockTests {
		exp := &SpotOrderResponse{
			OrderID:            "123",
			Text:               "t-abc123",
			AmendText:          "-",
			CreateTimeSeconds:  types.Time(time.Unix(1744150000, 0)),
			UpdateTimeSeconds:  types.Time(time.Unix(1744150200, 0)),
			CreateTime:         types.Time(time.UnixMilli(1744150000073)),
			UpdateTime:         types.Time(time.UnixMilli(1744150200074)),
			CurrencyPair:       "BTC_USDT",
			Status:             statusOpen,
			Type:               "limit",
			Account:            "spot",
			Side:               "buy",
			Amount:             0.001,
			Price:              1000,
			TimeInForce:        gtcTIF,
			RemainingAmount:    0.001,
			FeeCurrency:        currency.BTC,
			RebatedFeeCurrency: currency.USDT,
			FinishAs:           "open",
			StopProfit:         &SpotOrderTPSL{TriggerPrice: 67000, OrderPrice: 67000},
		}
		assert.Equal(t, exp, result, "AmendSpotOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "AmendSpotOrder should return a result")
}

func TestAmendSpotOrderRequestValidate(t *testing.T) {
	t.Parallel()
	assert.ErrorIs(t, (*AmendSpotOrderRequest)(nil).validate(), common.ErrNilPointer, "validate should reject a nil request")
	assert.ErrorIs(t, (&AmendSpotOrderRequest{Price: 1000}).validate(), currency.ErrCurrencyPairEmpty, "validate should reject an empty currency pair")

	err := (&AmendSpotOrderRequest{CurrencyPair: currency.NewBTCUSDT()}).validate()
	assert.ErrorIs(t, err, order.ErrAmountIsInvalid, "validate should reject a request without an amount or price")
	assert.ErrorIs(t, err, limits.ErrPriceBelowMin, "validate should reject a request without an amount or price")

	assert.NoError(t, (&AmendSpotOrderRequest{CurrencyPair: currency.NewBTCUSDT(), Amount: 1}).validate(), "validate should accept an amount alone")
	assert.NoError(t, (&AmendSpotOrderRequest{CurrencyPair: currency.NewBTCUSDT(), Price: 1000}).validate(), "validate should accept a price alone")
}

func TestCancelSingleSpotOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelSingleSpotOrder(t.Context(), "", getPair(t, asset.Spot), false)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelSingleSpotOrder must reject an empty order ID")

	_, err = e.CancelSingleSpotOrder(t.Context(), "1234", currency.EMPTYPAIR, false)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CancelSingleSpotOrder must reject an empty currency pair")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelSingleSpotOrder(t.Context(), "1234", getPair(t, asset.Spot), false)
	require.NoError(t, err, "CancelSingleSpotOrder must not error")
	if mockTests {
		exp := &SpotOrderResponse{
			OrderID:            "1234",
			Text:               "t-abc123",
			AmendText:          "-",
			CreateTimeSeconds:  types.Time(time.Unix(1744150000, 0)),
			UpdateTimeSeconds:  types.Time(time.Unix(1744150300, 0)),
			CreateTime:         types.Time(time.UnixMilli(1744150000073)),
			UpdateTime:         types.Time(time.UnixMilli(1744150300074)),
			CurrencyPair:       "BTC_USDT",
			Status:             "cancelled",
			Type:               "limit",
			Account:            "spot",
			Side:               "buy",
			Amount:             0.001,
			Price:              65000,
			TimeInForce:        gtcTIF,
			RemainingAmount:    0.001,
			FeeCurrency:        currency.BTC,
			RebatedFeeCurrency: currency.USDT,
			FinishAs:           "cancelled",
		}
		assert.Equal(t, exp, result, "CancelSingleSpotOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "CancelSingleSpotOrder should return a result")
}

func TestGetMySpotTradingHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetMySpotTradingHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetMySpotTradingHistory must reject a nil request")

	_, err = e.GetMySpotTradingHistory(t.Context(), &GetMySpotTradingHistoryRequest{OrderID: "987345678234"})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetMySpotTradingHistory must reject an order ID without a currency pair")

	startTime, endTime := getTime()
	_, err = e.GetMySpotTradingHistory(t.Context(), &GetMySpotTradingHistoryRequest{From: endTime, To: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetMySpotTradingHistory must reject a start time after the end time")

	_, err = e.GetMySpotTradingHistory(t.Context(), &GetMySpotTradingHistoryRequest{From: endTime.Add(-time.Hour * 24 * 31), To: endTime})
	require.ErrorIs(t, err, errTimeRangeTooLong, "GetMySpotTradingHistory must reject a range longer than 30 days")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetMySpotTradingHistory(t.Context(), &GetMySpotTradingHistoryRequest{
		CurrencyPair: getPair(t, asset.Spot),
		OrderID:      "987345678234",
		Page:         1,
		Limit:        100,
		From:         startTime,
		To:           endTime,
	})
	require.NoError(t, err, "GetMySpotTradingHistory must not error")
	if mockTests {
		exp := []*SpotPersonalTradeHistoryResponse{{
			TradeID:           "1232893232",
			CreateTimeSeconds: types.Time(time.Unix(1744150000, 0)),
			CreateTime:        types.Time(time.UnixMicro(1744150000123456)),
			CurrencyPair:      getPair(t, asset.Spot),
			OrderID:           "987345678234",
			Side:              "buy",
			Role:              "maker",
			Amount:            0.15,
			Price:             65000,
			Fee:               0.0005,
			FeeCurrency:       currency.BTC,
			AmendText:         "-",
			SequenceID:        "588018",
			Text:              "t-test",
			Deal:              9750,
			TradeQuote:        currency.USDT,
		}}
		assert.Equal(t, exp, result, "GetMySpotTradingHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetMySpotTradingHistory should return a result")
}

func TestGetServerTime(t *testing.T) {
	t.Parallel()
	result, err := e.GetServerTime(t.Context(), asset.Spot)
	require.NoError(t, err, "GetServerTime must not error")
	if mockTests {
		assert.Equal(t, time.UnixMilli(1774386128160), result, "GetServerTime should decode the server time")
		return
	}
	assert.WithinDuration(t, time.Now(), result, time.Minute, "GetServerTime should return the current time")
}

func TestCountdownCancelSpotOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CountdownCancelSpotOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CountdownCancelSpotOrders must reject a nil request")

	for _, timeout := range []time.Duration{-time.Second, 4 * time.Second, 5500 * time.Millisecond} {
		_, err = e.CountdownCancelSpotOrders(t.Context(), &CountdownCancelOrderRequest{Countdown: types.DurationSeconds(timeout), CurrencyPair: getPair(t, asset.Spot)})
		require.ErrorIsf(t, err, errInvalidTimeout, "CountdownCancelSpotOrders must reject a countdown of %s", timeout)
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name string
		arg  *CountdownCancelOrderRequest
		exp  *TriggerTimeResponse
	}{
		{
			name: "start countdown",
			arg:  &CountdownCancelOrderRequest{Countdown: types.DurationSeconds(10 * time.Second), CurrencyPair: getPair(t, asset.Spot)},
			exp:  &TriggerTimeResponse{TriggerTime: types.Time(time.UnixMilli(1744190264000))},
		},
		{
			// A timeout of 0 stops the countdown, which Gate documents as valid
			name: "stop countdown",
			arg:  &CountdownCancelOrderRequest{},
			exp:  &TriggerTimeResponse{},
		},
	} {
		result, err := e.CountdownCancelSpotOrders(t.Context(), tc.arg)
		require.NoErrorf(t, err, "CountdownCancelSpotOrders must not error for %s", tc.name)
		if mockTests {
			assert.Equalf(t, tc.exp, result, "CountdownCancelSpotOrders should decode the trigger time for %s", tc.name)
		}
	}
}

func TestAmendBatchSpotOrders(t *testing.T) {
	t.Parallel()
	_, err := e.AmendBatchSpotOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrEmptyParams, "AmendBatchSpotOrders must reject an empty batch")

	valid := &BatchAmendSpotOrderRequest{OrderID: "121212", CurrencyPair: getPair(t, asset.Spot), Amount: 1, AmendText: "test"}
	_, err = e.AmendBatchSpotOrders(t.Context(), slices.Repeat([]*BatchAmendSpotOrderRequest{valid}, 6))
	require.ErrorIs(t, err, errTooManyBatchOrders, "AmendBatchSpotOrders must reject more than 5 orders")

	_, err = e.AmendBatchSpotOrders(t.Context(), []*BatchAmendSpotOrderRequest{nil})
	require.ErrorIs(t, err, common.ErrNilPointer, "AmendBatchSpotOrders must reject a nil order")

	_, err = e.AmendBatchSpotOrders(t.Context(), []*BatchAmendSpotOrderRequest{{AmendSpotOrderRequest: valid.AmendSpotOrderRequest}})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "AmendBatchSpotOrders must reject an empty order ID")

	_, err = e.AmendBatchSpotOrders(t.Context(), []*BatchAmendSpotOrderRequest{{OrderID: "121212", Amount: 1}})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "AmendBatchSpotOrders must reject an empty currency pair")

	_, err = e.AmendBatchSpotOrders(t.Context(), []*BatchAmendSpotOrderRequest{{OrderID: "121212", CurrencyPair: getPair(t, asset.Spot)}})
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "AmendBatchSpotOrders must reject an order without amount or price")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.AmendBatchSpotOrders(t.Context(), []*BatchAmendSpotOrderRequest{valid})
	require.NoError(t, err, "AmendBatchSpotOrders must not error")
	if mockTests {
		exp := []*SpotOrderResponse{{
			OrderID:            "121212",
			BatchOrderID:       "121212",
			Text:               "t-123456",
			AmendText:          "test",
			Succeeded:          true,
			CreateTimeSeconds:  types.Time(time.Unix(1744150000, 0)),
			UpdateTimeSeconds:  types.Time(time.Unix(1744150100, 0)),
			CreateTime:         types.Time(time.UnixMilli(1744150000123)),
			UpdateTime:         types.Time(time.UnixMilli(1744150100123)),
			CurrencyPair:       "BTC_USDT",
			Status:             statusOpen,
			Type:               "limit",
			Account:            "spot",
			Side:               "buy",
			Amount:             1,
			Price:              5.00032,
			TimeInForce:        gtcTIF,
			RemainingAmount:    1,
			FeeCurrency:        currency.BTC,
			RebatedFeeCurrency: currency.USDT,
			STPID:              10240,
			STPAction:          "cn",
			FinishAs:           "open",
		}}
		assert.Equal(t, exp, result, "AmendBatchSpotOrders should decode every field")
		return
	}
	assert.NotEmpty(t, result, "AmendBatchSpotOrders should return order results")
}

func TestGetSpotInsuranceHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpotInsuranceHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSpotInsuranceHistory must reject a nil request")

	startTime, endTime := getTime()
	_, err = e.GetSpotInsuranceHistory(t.Context(), &GetSpotInsuranceHistoryRequest{Currency: currency.USDT, From: startTime, To: endTime})
	require.ErrorIs(t, err, errBusinessTypeRequired, "GetSpotInsuranceHistory must reject an empty business type")

	_, err = e.GetSpotInsuranceHistory(t.Context(), &GetSpotInsuranceHistoryRequest{Business: "unified", From: startTime, To: endTime})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetSpotInsuranceHistory must reject an empty currency")

	_, err = e.GetSpotInsuranceHistory(t.Context(), &GetSpotInsuranceHistoryRequest{Business: "unified", Currency: currency.USDT, To: endTime})
	require.ErrorIs(t, err, common.ErrDateUnset, "GetSpotInsuranceHistory must reject an unset start time")

	_, err = e.GetSpotInsuranceHistory(t.Context(), &GetSpotInsuranceHistoryRequest{Business: "unified", Currency: currency.USDT, From: endTime, To: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetSpotInsuranceHistory must reject a start time after the end time")

	result, err := e.GetSpotInsuranceHistory(t.Context(), &GetSpotInsuranceHistoryRequest{
		Business: "unified",
		Currency: currency.USDT,
		From:     startTime,
		To:       endTime,
		Page:     1,
		Limit:    2,
	})
	require.NoError(t, err, "GetSpotInsuranceHistory must not error")
	if mockTests {
		exp := []*SpotInsuranceHistoryResponse{{Currency: currency.USDT, Balance: 3678340.316884272366, Time: types.Time(time.UnixMilli(1744156800000))}}
		assert.Equal(t, exp, result, "GetSpotInsuranceHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSpotInsuranceHistory should return a result")
}

func TestCreatePriceTriggeredOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CreatePriceTriggeredOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreatePriceTriggeredOrder must reject a nil request")

	_, err = e.CreatePriceTriggeredOrder(t.Context(), &PriceTriggeredOrderRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CreatePriceTriggeredOrder must reject an empty market")

	arg := &PriceTriggeredOrderRequest{Market: getPair(t, asset.Spot), Trigger: TriggerPriceInfo{Price: -1}}
	_, err = e.CreatePriceTriggeredOrder(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrPriceBelowMin, "CreatePriceTriggeredOrder must reject a non-positive trigger price")

	arg.Trigger.Price = 1
	_, err = e.CreatePriceTriggeredOrder(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidTriggerRule, "CreatePriceTriggeredOrder must reject an invalid trigger rule")

	arg.Trigger.Rule = ">="
	arg.Trigger.ExpireAfter = fractionalSeconds
	_, err = e.CreatePriceTriggeredOrder(t.Context(), arg)
	require.ErrorIs(t, err, errNotWholeSeconds, "CreatePriceTriggeredOrder must reject a trigger expiry with a fraction of a second")

	arg.Trigger.ExpireAfter = negativeSeconds
	_, err = e.CreatePriceTriggeredOrder(t.Context(), arg)
	require.ErrorIs(t, err, errNotWholeSeconds, "CreatePriceTriggeredOrder must reject a negative trigger expiry")

	arg.Trigger.ExpireAfter = types.DurationSeconds(time.Hour)
	_, err = e.CreatePriceTriggeredOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "CreatePriceTriggeredOrder must reject an invalid side")

	arg.Put.Side = "SELL"
	arg.Put.Price = -1
	_, err = e.CreatePriceTriggeredOrder(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrPriceBelowMin, "CreatePriceTriggeredOrder must reject a negative put price")

	arg.Put.Price = 1
	_, err = e.CreatePriceTriggeredOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "CreatePriceTriggeredOrder must reject an empty amount")

	arg.Put.Amount = 1
	arg.Put.TimeInForce = pocTIF
	_, err = e.CreatePriceTriggeredOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrInvalidTimeInForce, "CreatePriceTriggeredOrder must reject time in force other than gtc or ioc")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CreatePriceTriggeredOrder(t.Context(), &PriceTriggeredOrderRequest{
		Trigger: TriggerPriceInfo{
			Price:       123,
			Rule:        ">=",
			ExpireAfter: types.DurationSeconds(time.Hour),
		},
		Put: PutOrder{
			Type:        "limit",
			Side:        "sell",
			Price:       2312312,
			Amount:      30,
			TimeInForce: "GTC",
		},
		Market: getPair(t, asset.Spot),
	})
	require.NoError(t, err, "CreatePriceTriggeredOrder must not error")
	if mockTests {
		assert.Equal(t, &TriggerOrderIDResponse{ID: 1432329, IDString: "1432329"}, result, "CreatePriceTriggeredOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "CreatePriceTriggeredOrder should return a result")
}

func TestGetPriceTriggeredOrderList(t *testing.T) {
	t.Parallel()
	_, err := e.GetPriceTriggeredOrderList(t.Context(), "", currency.EMPTYPAIR, "", 0, 0)
	require.ErrorIs(t, err, errInvalidOrderStatus, "GetPriceTriggeredOrderList must reject an empty status")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPriceTriggeredOrderList(t.Context(), statusOpen, currency.EMPTYPAIR, "", 0, 0)
	require.NoError(t, err, "GetPriceTriggeredOrderList must not error")
	if mockTests {
		exp := []*SpotPriceTriggeredOrderResponse{{
			Trigger:      TriggerPriceInfo{Price: 100, Rule: ">=", ExpireAfter: types.DurationSeconds(time.Hour)},
			Put:          PutOrder{Type: "limit", Side: "buy", Price: 2.15, Amount: 2, Account: "normal", TimeInForce: gtcTIF, Text: "api"},
			AutoOrderID:  1283293,
			UserID:       1234,
			CreationTime: types.Time(time.Unix(1744150000, 0)),
			Status:       statusOpen,
			Market:       getPair(t, asset.Spot),
		}}
		assert.Equal(t, exp, result, "GetPriceTriggeredOrderList should decode every field")
		return
	}
	assert.NotNil(t, result, "GetPriceTriggeredOrderList should return a result")
}

func TestCancelMultipleSpotOpenOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelMultipleSpotOpenOrders(t.Context(), getPair(t, asset.Spot), "normal")
	require.NoError(t, err, "CancelMultipleSpotOpenOrders must not error")
	if mockTests {
		exp := []*SpotPriceTriggeredOrderResponse{{
			Trigger:      TriggerPriceInfo{Price: 100, Rule: "<=", ExpireAfter: types.DurationSeconds(time.Hour)},
			Put:          PutOrder{Type: "market", Side: "sell", Price: 99, Amount: 2, Account: "normal", TimeInForce: iocTIF, Text: "api"},
			AutoOrderID:  1283294,
			UserID:       1234,
			CreationTime: types.Time(time.Unix(1744150000, 0)),
			FireTime:     types.Time(time.Unix(1744150500, 0)),
			Status:       "cancelled",
			Reason:       "cancelled by user",
			Market:       getPair(t, asset.Spot),
		}}
		assert.Equal(t, exp, result, "CancelMultipleSpotOpenOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "CancelMultipleSpotOpenOrders should return a result")
}

func TestGetSinglePriceTriggeredOrder(t *testing.T) {
	t.Parallel()
	_, err := e.GetSinglePriceTriggeredOrder(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetSinglePriceTriggeredOrder must reject an empty order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSinglePriceTriggeredOrder(t.Context(), "1234")
	require.NoError(t, err, "GetSinglePriceTriggeredOrder must not error")
	if mockTests {
		exp := &SpotPriceTriggeredOrderResponse{
			Trigger:      TriggerPriceInfo{Price: 100, Rule: ">=", ExpireAfter: types.DurationSeconds(time.Hour)},
			Put:          PutOrder{Type: "limit", Side: "buy", Price: 2.15, Amount: 2, Account: "normal", TimeInForce: gtcTIF, Text: "api"},
			AutoOrderID:  1234,
			UserID:       1234,
			CreationTime: types.Time(time.Unix(1744150000, 0)),
			FireTime:     types.Time(time.Unix(1744150001, 0)),
			FiredOrderID: 18523,
			Status:       "finish",
			Market:       getPair(t, asset.Spot),
		}
		assert.Equal(t, exp, result, "GetSinglePriceTriggeredOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSinglePriceTriggeredOrder should return a result")
}

func TestCancelPriceTriggeredOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelPriceTriggeredOrder(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelPriceTriggeredOrder must reject an empty order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelPriceTriggeredOrder(t.Context(), "1234")
	require.NoError(t, err, "CancelPriceTriggeredOrder must not error")
	if mockTests {
		exp := &SpotPriceTriggeredOrderResponse{
			Trigger:      TriggerPriceInfo{Price: 100, Rule: ">=", ExpireAfter: types.DurationSeconds(time.Hour)},
			Put:          PutOrder{Type: "limit", Side: "buy", Price: 2.15, Amount: 2, Account: "normal", TimeInForce: gtcTIF, Text: "api"},
			AutoOrderID:  1234,
			UserID:       1234,
			CreationTime: types.Time(time.Unix(1744150000, 0)),
			FireTime:     types.Time(time.Unix(1744150500, 0)),
			Status:       "cancelled",
			Market:       getPair(t, asset.Spot),
		}
		assert.Equal(t, exp, result, "CancelPriceTriggeredOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "CancelPriceTriggeredOrder should return a result")
}

func TestCreateSpotPOVOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CreateSpotPOVOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreateSpotPOVOrder must reject a nil request")

	arg := &SpotPOVOrderRequest{}
	_, err = e.CreateSpotPOVOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CreateSpotPOVOrder must reject an empty currency pair")

	arg.CurrencyPair = getPair(t, asset.Spot)
	_, err = e.CreateSpotPOVOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "CreateSpotPOVOrder must reject an invalid side")

	arg.Side = order.Buy.Lower()
	_, err = e.CreateSpotPOVOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "CreateSpotPOVOrder must reject an empty amount")

	arg.Amount = 1
	_, err = e.CreateSpotPOVOrder(t.Context(), arg)
	require.ErrorIs(t, err, errParticipationRateRequired, "CreateSpotPOVOrder must reject an empty participation rate")

	arg.ParticipationRatePercentage = 5
	_, err = e.CreateSpotPOVOrder(t.Context(), arg)
	require.ErrorIs(t, err, errTimeToLiveRequired, "CreateSpotPOVOrder must reject an empty time to live")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CreateSpotPOVOrder(t.Context(), &SpotPOVOrderRequest{
		CurrencyPair:                getPair(t, asset.Spot),
		Side:                        order.Buy.Lower(),
		Amount:                      1,
		ParticipationRatePercentage: 5,
		TimeToLive:                  "1h",
		LimitPrice:                  63000,
		TriggerPrice:                63000,
		Text:                        "t-pov1",
	})
	require.NoError(t, err, "CreateSpotPOVOrder must not error")
	if mockTests {
		exp := &SpotPOVOrderResponse{
			ID:                          "1216",
			CurrencyPair:                getPair(t, asset.Spot),
			Side:                        "buy",
			Amount:                      1,
			ParticipationRatePercentage: 5,
			TimeToLive:                  "1h",
			LimitPrice:                  63000,
			TriggerPrice:                63000,
			Status:                      "CREATED",
			ExpireTime:                  types.Time(time.UnixMilli(1744153600074)),
			CreateTime:                  types.Time(time.UnixMilli(1744150000074)),
			UpdateTime:                  types.Time(time.UnixMilli(1744150000074)),
			Text:                        "t-pov1",
		}
		assert.Equal(t, exp, result, "CreateSpotPOVOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "CreateSpotPOVOrder should return a result")
}

func TestGetSpotPOVOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpotPOVOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSpotPOVOrders must reject a nil request")

	_, err = e.GetSpotPOVOrders(t.Context(), &GetSpotPOVOrdersRequest{})
	require.ErrorIs(t, err, errInvalidOrderStatus, "GetSpotPOVOrders must reject an empty status")

	_, err = e.GetSpotPOVOrders(t.Context(), &GetSpotPOVOrdersRequest{Status: statusOpen, Side: order.Short})
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "GetSpotPOVOrders must reject an invalid side")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSpotPOVOrders(t.Context(), &GetSpotPOVOrdersRequest{
		CurrencyPair: getPair(t, asset.Spot),
		Status:       statusOpen,
		Side:         order.Buy,
		Page:         1,
		Limit:        10,
	})
	require.NoError(t, err, "GetSpotPOVOrders must not error")
	if mockTests {
		exp := []*SpotPOVOrderResponse{{
			ID:                          "1216",
			CurrencyPair:                getPair(t, asset.Spot),
			Side:                        "buy",
			Amount:                      0.01,
			ParticipationRatePercentage: 10,
			TimeToLive:                  "1h",
			LimitPrice:                  63000,
			TriggerPrice:                63000,
			Status:                      "RUNNING",
			StartTime:                   types.Time(time.UnixMilli(1744150001074)),
			ExpireTime:                  types.Time(time.UnixMilli(1744153600074)),
			CreateTime:                  types.Time(time.UnixMilli(1744150000074)),
			UpdateTime:                  types.Time(time.UnixMilli(1744150001074)),
		}}
		assert.Equal(t, exp, result, "GetSpotPOVOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSpotPOVOrders should return a result")
}

func TestCancelSpotPOVOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelSpotPOVOrders(t.Context(), getPair(t, asset.Spot))
	require.NoError(t, err, "CancelSpotPOVOrders must not error")
	if mockTests {
		exp := []*SpotPOVOrderResponse{{
			ID:                          "1216",
			CurrencyPair:                getPair(t, asset.Spot),
			Side:                        "buy",
			Amount:                      0.01,
			ParticipationRatePercentage: 10,
			TimeToLive:                  "1h",
			LimitPrice:                  63000,
			TriggerPrice:                63000,
			Status:                      "CANCELING",
			StartTime:                   types.Time(time.UnixMilli(1744150001074)),
			ExpireTime:                  types.Time(time.UnixMilli(1744153600074)),
			CreateTime:                  types.Time(time.UnixMilli(1744150000074)),
			UpdateTime:                  types.Time(time.UnixMilli(1744150002074)),
		}}
		assert.Equal(t, exp, result, "CancelSpotPOVOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "CancelSpotPOVOrders should return a result")
}

func TestGetSpotPOVOrder(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpotPOVOrder(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetSpotPOVOrder must reject an empty order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSpotPOVOrder(t.Context(), "1216")
	require.NoError(t, err, "GetSpotPOVOrder must not error")
	if mockTests {
		exp := &SpotPOVOrderResponse{
			ID:                          "1216",
			CurrencyPair:                getPair(t, asset.Spot),
			Side:                        "sell",
			Amount:                      0.01,
			ParticipationRatePercentage: 20,
			TimeToLive:                  "1d",
			Status:                      "COMPLETED",
			StartTime:                   types.Time(time.UnixMilli(1744150001074)),
			EndTime:                     types.Time(time.UnixMilli(1744150901074)),
			ExpireTime:                  types.Time(time.UnixMilli(1744236400074)),
			CreateTime:                  types.Time(time.UnixMilli(1744150000074)),
			UpdateTime:                  types.Time(time.UnixMilli(1744150901074)),
			Text:                        "t-pov2",
		}
		assert.Equal(t, exp, result, "GetSpotPOVOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSpotPOVOrder should return a result")
}

func TestCancelSpotPOVOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelSpotPOVOrder(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelSpotPOVOrder must reject an empty order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelSpotPOVOrder(t.Context(), "1216")
	require.NoError(t, err, "CancelSpotPOVOrder must not error")
	if mockTests {
		exp := &SpotPOVOrderResponse{
			ID:                          "1216",
			CurrencyPair:                getPair(t, asset.Spot),
			Side:                        "buy",
			Amount:                      0.01,
			ParticipationRatePercentage: 40,
			TimeToLive:                  "6h",
			LimitPrice:                  63000,
			Status:                      "CANCELING",
			StartTime:                   types.Time(time.UnixMilli(1744150001074)),
			ExpireTime:                  types.Time(time.UnixMilli(1744171600074)),
			CreateTime:                  types.Time(time.UnixMilli(1744150000074)),
			UpdateTime:                  types.Time(time.UnixMilli(1744150002074)),
		}
		assert.Equal(t, exp, result, "CancelSpotPOVOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "CancelSpotPOVOrder should return a result")
}
