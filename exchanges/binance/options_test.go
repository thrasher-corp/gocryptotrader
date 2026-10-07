package binance

import (
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetAccountFundingFlow(t *testing.T) {
	t.Parallel()
	_, err := e.GetAccountFundingFlow(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAccountFundingFlow must reject a nil request")
	_, err = e.GetAccountFundingFlow(t.Context(), &OptionsAccountFundingFlowRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetAccountFundingFlow must reject an empty currency")

	startTime, endTime := getTime()
	_, err = e.GetAccountFundingFlow(t.Context(), &OptionsAccountFundingFlowRequest{Currency: currency.USDT, StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetAccountFundingFlow must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAccountFundingFlow(t.Context(), &OptionsAccountFundingFlowRequest{Currency: currency.USDT, StartTime: startTime, EndTime: endTime})
	require.NoError(t, err, "GetAccountFundingFlow must not error")
	if mockTests {
		exp := []OptionsAccountFunding{
			{
				ID:         1125899906842624000,
				Asset:      currency.USDT,
				Amount:     -0.552,
				Type:       "FEE",
				CreateDate: types.Time(time.Unix(1744120800, 0)),
			},
			{
				ID:         1125899906842624001,
				Asset:      currency.USDT,
				Amount:     1000,
				Type:       "TRANSFER",
				CreateDate: types.Time(time.Unix(1744124400, 0)),
			},
		}
		assert.Equal(t, exp, result, "GetAccountFundingFlow should decode every field")
		return
	}
	assert.NotNil(t, result, "GetAccountFundingFlow should return funding flows")
}

func TestGetOptionMarginAccountInformation(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetOptionMarginAccountInformation(t.Context())
	require.NoError(t, err, "GetOptionMarginAccountInformation must not error")
	if mockTests {
		exp := &OptionsMarginAccountInformationResponse{
			Asset: []OptionsMarginAccountAsset{
				{
					Asset:             currency.USDT,
					MarginBalance:     10099.448,
					Equity:            10094.44662,
					Available:         8725.92524,
					InitialMargin:     1084.52138,
					MaintenanceMargin: 151.00138,
					UnrealizedPNL:     -5.00138,
					AdjustedEquity:    34.13282285,
				},
			},
			Greek: []OptionsGreek{
				{
					Underlying: "BTCUSDT",
					Delta:      -0.05,
					Gamma:      -0.002,
					Theta:      -0.05,
					Vega:       -0.002,
				},
			},
			Time:         types.Time(time.UnixMilli(1762843368098)),
			CanTrade:     true,
			CanDeposit:   true,
			CanWithdraw:  true,
			ReduceOnly:   true,
			TradeGroupID: -1,
		}
		assert.Equal(t, exp, result, "GetOptionMarginAccountInformation should decode every field")
		return
	}
	assert.NotNil(t, result, "GetOptionMarginAccountInformation should return account information")
}

func TestGetDownloadIDForOptionTransactionHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetDownloadIDForOptionTransactionHistory(t.Context(), endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetDownloadIDForOptionTransactionHistory must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetDownloadIDForOptionTransactionHistory(t.Context(), startTime, endTime)
	require.NoError(t, err, "GetDownloadIDForOptionTransactionHistory must not error")
	if mockTests {
		exp := &OptionsTransactionHistoryDownloadIDResponse{
			AverageCostTimestampOfLast30Days: 7241837,
			DownloadID:                       "546975389218332672",
		}
		assert.Equal(t, exp, result, "GetDownloadIDForOptionTransactionHistory should decode every field")
		return
	}
	assert.NotEmpty(t, result.DownloadID, "GetDownloadIDForOptionTransactionHistory should return a download ID")
}

func TestGetOptionTransactionHistoryDownloadLinkByID(t *testing.T) {
	t.Parallel()
	_, err := e.GetOptionTransactionHistoryDownloadLinkByID(t.Context(), "")
	require.ErrorIs(t, err, errDownloadIDRequired, "GetOptionTransactionHistoryDownloadLinkByID must reject an empty download ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetOptionTransactionHistoryDownloadLinkByID(t.Context(), "545923594199212032")
	require.NoError(t, err, "GetOptionTransactionHistoryDownloadLinkByID must not error")
	if mockTests {
		exp := &OptionsTransactionHistoryDownloadLinkResponse{
			DownloadID:          "545923594199212032",
			Status:              "completed",
			URL:                 "https://example.com/545923594199212032.zip",
			Notified:            true,
			ExpirationTimestamp: 1645009771000,
			IsExpired:           true,
		}
		assert.Equal(t, exp, result, "GetOptionTransactionHistoryDownloadLinkByID should decode every field")
		return
	}
	assert.NotNil(t, result, "GetOptionTransactionHistoryDownloadLinkByID should return the download link")
}

func TestCheckEOptionsServerTime(t *testing.T) {
	t.Parallel()
	result, err := e.CheckEOptionsServerTime(t.Context())
	require.NoError(t, err, "CheckEOptionsServerTime must not error")
	if mockTests {
		exp := &OptionsServerTimeResponse{
			ServerTime: types.Time(time.UnixMilli(1791285187287)),
		}
		assert.Equal(t, exp, result, "CheckEOptionsServerTime should decode every field")
		return
	}
	assert.NotZero(t, result.ServerTime, "CheckEOptionsServerTime should return the server time")
}

func TestGetOptionsExchangeInformation(t *testing.T) {
	t.Parallel()
	result, err := e.GetOptionsExchangeInformation(t.Context())
	require.NoError(t, err, "GetOptionsExchangeInformation must not error")
	if mockTests {
		exp := &OptionsExchangeInformationResponse{
			Timezone:   "UTC",
			ServerTime: types.Time(time.UnixMilli(1791282968917)),
			OptionContracts: []OptionsContract{
				{
					BaseAsset:   currency.NewCode("XAG"),
					QuoteAsset:  currency.USDT,
					Underlying:  "XAGUSDT",
					SettleAsset: currency.USDT,
				},
				{
					BaseAsset:   currency.BTC,
					QuoteAsset:  currency.USDT,
					Underlying:  "BTCUSDT",
					SettleAsset: currency.USDT,
					NakedSell:   true,
				},
			},
			OptionAssets: []OptionsAsset{
				{
					Name: currency.USDT,
				},
			},
			OptionSymbols: []OptionsSymbolDetail{
				{
					ExpiryDate: types.Time(time.Unix(1775808000, 0)),
					Filters: []OptionsSymbolFilter{
						{
							FilterType: "PRICE_FILTER",
							MinPrice:   965,
							MaxPrice:   8675,
							TickSize:   5,
						},
						{
							FilterType:  "LOT_SIZE",
							MinQuantity: 0.01,
							MaxQuantity: 200,
							StepSize:    0.01,
						},
					},
					Symbol:               "BTC-260410-72000-P",
					Side:                 "PUT",
					StrikePrice:          72000,
					Underlying:           "BTCUSDT",
					Unit:                 1,
					LiquidationFeeRate:   0.0019,
					MinQuantity:          0.01,
					MaxQuantity:          200,
					InitialMargin:        0.15,
					MaintenanceMargin:    0.075,
					MinInitialMargin:     0.1,
					MinMaintenanceMargin: 0.05,
					PriceScale:           3,
					QuantityScale:        2,
					QuoteAsset:           currency.USDT,
					ContractType:         "CRYPTO_OPTIONS",
					UnderlyingType:       "CRYPTO",
					Status:               "TRADING",
				},
				{
					ExpiryDate: types.Time(time.Unix(1798185600, 0)),
					Filters: []OptionsSymbolFilter{
						{
							FilterType: "PRICE_FILTER",
							MinPrice:   1375,
							MaxPrice:   12360,
							TickSize:   5,
						},
						{
							FilterType:  "LOT_SIZE",
							MinQuantity: 0.01,
							MaxQuantity: 200,
							StepSize:    0.01,
						},
					},
					Symbol:               "BTC-261225-85000-C",
					Side:                 "CALL",
					StrikePrice:          85000,
					Underlying:           "BTCUSDT",
					Unit:                 1,
					LiquidationFeeRate:   0.0019,
					MinQuantity:          0.01,
					MaxQuantity:          200,
					InitialMargin:        0.15,
					MaintenanceMargin:    0.075,
					MinInitialMargin:     0.1,
					MinMaintenanceMargin: 0.05,
					PriceScale:           3,
					QuantityScale:        2,
					QuoteAsset:           currency.USDT,
					ContractType:         "CRYPTO_OPTIONS",
					UnderlyingType:       "CRYPTO",
					NakedSell:            true,
					Status:               "TRADING",
				},
			},
			RateLimits: []RateLimitInfo{
				{
					RateLimitType: "REQUEST_WEIGHT",
					Interval:      "MINUTE",
					IntervalNum:   1,
					Limit:         400,
				},
				{
					RateLimitType: "ORDERS",
					Interval:      "MINUTE",
					IntervalNum:   1,
					Limit:         100,
				},
				{
					RateLimitType: "ORDERS",
					Interval:      "SECOND",
					IntervalNum:   10,
					Limit:         30,
				},
			},
		}
		assert.Equal(t, exp, result, "GetOptionsExchangeInformation should decode every field")
		return
	}
	assert.NotEmpty(t, result.OptionSymbols, "GetOptionsExchangeInformation should return option symbols")
}

func TestGetEOptionsHistoricalExerciseRecords(t *testing.T) {
	t.Parallel()
	_, err := e.GetEOptionsHistoricalExerciseRecords(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetEOptionsHistoricalExerciseRecords must reject a nil request")

	startTime, endTime := getTime()
	_, err = e.GetEOptionsHistoricalExerciseRecords(t.Context(), &OptionsExerciseHistoryRequest{Underlying: "BTCUSDT", StartTime: endTime, EndTime: startTime, Limit: 10})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetEOptionsHistoricalExerciseRecords must reject a start time after the end time")

	result, err := e.GetEOptionsHistoricalExerciseRecords(t.Context(), &OptionsExerciseHistoryRequest{Underlying: "BTCUSDT", StartTime: startTime, EndTime: endTime, Limit: 10})
	require.NoError(t, err, "GetEOptionsHistoricalExerciseRecords must not error")
	if mockTests {
		exp := []ExerciseHistoryItem{
			{
				Symbol:          "BTC-261006-90000-C",
				StrikePrice:     90000,
				RealStrikePrice: 85507.724,
				ExpiryDate:      types.Time(time.Unix(1791273600, 0)),
				StrikeResult:    "EXTRINSIC_VALUE_EXPIRED",
			},
			{
				Symbol:          "BTC-261006-90000-P",
				StrikePrice:     90000,
				RealStrikePrice: 85507.724,
				ExpiryDate:      types.Time(time.Unix(1791273600, 0)),
				StrikeResult:    "REALISTIC_VALUE_STRICKEN",
			},
		}
		assert.Equal(t, exp, result, "GetEOptionsHistoricalExerciseRecords should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetEOptionsHistoricalExerciseRecords should return exercise records")
}

func TestGetOptionsIndexPrice(t *testing.T) {
	t.Parallel()
	_, err := e.GetOptionsIndexPrice(t.Context(), "")
	require.ErrorIs(t, err, errUnderlyingIsRequired, "GetOptionsIndexPrice must reject an empty underlying")

	result, err := e.GetOptionsIndexPrice(t.Context(), "BTCUSDT")
	require.NoError(t, err, "GetOptionsIndexPrice must not error")
	if mockTests {
		exp := &OptionsIndexPriceResponse{
			Time:       types.Time(time.Unix(1791285189, 0)),
			IndexPrice: 86162.79847826,
		}
		assert.Equal(t, exp, result, "GetOptionsIndexPrice should decode every field")
		return
	}
	assert.Positive(t, result.IndexPrice.Float64(), "GetOptionsIndexPrice should return the index price")
}

func TestGetEOptionsCandlesticks(t *testing.T) {
	t.Parallel()
	_, err := e.GetEOptionsCandlesticks(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetEOptionsCandlesticks must reject a nil request")
	_, err = e.GetEOptionsCandlesticks(t.Context(), &OptionsKlineRequest{Interval: kline.OneDay})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetEOptionsCandlesticks must reject an empty symbol")

	ensureTradablePairs(t)
	_, err = e.GetEOptionsCandlesticks(t.Context(), &OptionsKlineRequest{Symbol: optionsTradablePair})
	require.ErrorIs(t, err, kline.ErrInvalidInterval, "GetEOptionsCandlesticks must reject an empty interval")

	startTime, endTime := getTime()
	_, err = e.GetEOptionsCandlesticks(t.Context(), &OptionsKlineRequest{Symbol: optionsTradablePair, Interval: kline.OneDay, StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetEOptionsCandlesticks must reject a start time after the end time")

	result, err := e.GetEOptionsCandlesticks(t.Context(), &OptionsKlineRequest{Symbol: optionsTradablePair, Interval: kline.OneDay, StartTime: startTime, EndTime: endTime, Limit: 1000})
	require.NoError(t, err, "GetEOptionsCandlesticks must not error")
	if mockTests {
		exp := []EOptionsCandlestick{
			{
				OpenTime:                 types.Time(time.Unix(1791072000, 0)),
				Open:                     380,
				High:                     940,
				Low:                      380,
				Close:                    885,
				Volume:                   8.27,
				CloseTime:                types.Time(time.UnixMilli(1791158399999)),
				QuoteAssetVolume:         5683.05,
				NumberOfTrades:           31,
				TakerBuyBaseAssetVolume:  6.05,
				TakerBuyQuoteAssetVolume: 3822.5,
			},
			{
				OpenTime:                 types.Time(time.Unix(1791158400, 0)),
				Open:                     810,
				High:                     980,
				Low:                      190,
				Close:                    305,
				Volume:                   43.19,
				CloseTime:                types.Time(time.UnixMilli(1791244799999)),
				QuoteAssetVolume:         18514.35,
				NumberOfTrades:           137,
				TakerBuyBaseAssetVolume:  22.49,
				TakerBuyQuoteAssetVolume: 10575.2,
			},
		}
		assert.Equal(t, exp, result, "GetEOptionsCandlesticks should decode every field")
		return
	}
	assert.NotNil(t, result, "GetEOptionsCandlesticks should return klines")
}

func TestEOptionsCandlestickUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var c EOptionsCandlestick
	require.NoError(t, json.Unmarshal([]byte(`[1791158400000,"810.000","980.000","190.000","305.000","43.19",1791244799999,"18514.35000",137,"22.49","10575.20000","0"]`), &c), "Unmarshal must not error")
	exp := EOptionsCandlestick{
		OpenTime:                 types.Time(time.UnixMilli(1791158400000)),
		Open:                     810,
		High:                     980,
		Low:                      190,
		Close:                    305,
		Volume:                   43.19,
		CloseTime:                types.Time(time.UnixMilli(1791244799999)),
		QuoteAssetVolume:         18514.35,
		NumberOfTrades:           137,
		TakerBuyBaseAssetVolume:  22.49,
		TakerBuyQuoteAssetVolume: 10575.2,
	}
	assert.Equal(t, exp, c, "EOptionsCandlestick should decode every element")
	assert.Error(t, json.Unmarshal([]byte(`{"open":"810.000"}`), &c), "Unmarshal should error on an object")
	assert.Error(t, json.Unmarshal([]byte(`[1791158400000,"810.000","980.000","190.000","305.000","43.19",1791244799999,"18514.35000","many"]`), &c), "Unmarshal should error on a malformed trade count")
}

func TestGetEOptionsOpenInterests(t *testing.T) {
	t.Parallel()
	_, err := e.GetEOptionsOpenInterests(t.Context(), currency.EMPTYCODE, time.Now())
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetEOptionsOpenInterests must reject an empty underlying asset")
	_, err = e.GetEOptionsOpenInterests(t.Context(), currency.BTC, time.Time{})
	require.ErrorIs(t, err, errExpirationTimeRequired, "GetEOptionsOpenInterests must reject an empty expiration")

	ensureTradablePairs(t)
	// Option symbols carry their expiry as YYMMDD, so a listed option names an expiry Binance holds open interest for
	expiration, err := time.Parse("060102", optionsTradablePair.Quote.String()[:6])
	require.NoError(t, err, "the option expiry must parse")
	result, err := e.GetEOptionsOpenInterests(t.Context(), optionsTradablePair.Base, expiration)
	require.NoError(t, err, "GetEOptionsOpenInterests must not error")
	if mockTests {
		exp := []OptionsOpenInterest{
			{
				Symbol:             "BTC-260410-87000-C",
				SumOpenInterest:    134.21,
				SumOpenInterestUSD: 11563950.497106813,
				Timestamp:          types.Time(time.Unix(1791285180, 0)),
			},
			{
				Symbol:             "BTC-260410-82000-C",
				SumOpenInterest:    0.01,
				SumOpenInterestUSD: 861.6310630435,
				Timestamp:          types.Time(time.Unix(1791285180, 0)),
			},
		}
		assert.Equal(t, exp, result, "GetEOptionsOpenInterests should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetEOptionsOpenInterests should return open interest")
}

func TestGetOptionMarkPrice(t *testing.T) {
	t.Parallel()
	ensureTradablePairs(t)
	for _, tc := range []struct {
		name   string
		symbol currency.Pair
		exp    []OptionMarkPrice
	}{
		{"every symbol", currency.EMPTYPAIR, []OptionMarkPrice{
			{
				Symbol:           "BTC-261225-85000-C",
				MarkPrice:        6935.765,
				BidIV:            0.36595106,
				AskIV:            0.36914587,
				MarkIV:           0.366,
				Delta:            0.59185483,
				Theta:            -36.01368586,
				Gamma:            0.00002621,
				Vega:             156.51072692,
				HighPriceLimit:   12480,
				LowPriceLimit:    1390,
				RiskFreeInterest: 0.0527,
			},
			{
				Symbol:           "BTC-261225-85000-P",
				MarkPrice:        4798.435,
				BidIV:            0.36961377,
				AskIV:            0.37025273,
				MarkIV:           0.366,
				Delta:            -0.40814517,
				Theta:            -36.24735514,
				Gamma:            0.00002604,
				Vega:             156.51072692,
				HighPriceLimit:   8635,
				LowPriceLimit:    960,
				RiskFreeInterest: 0.0527,
			},
		}},
		{"symbol", optionsTradablePair, []OptionMarkPrice{
			{
				Symbol:           "BTC-260410-72000-P",
				MarkPrice:        4798.435,
				BidIV:            0.36961377,
				AskIV:            0.37025273,
				MarkIV:           0.366,
				Delta:            -0.40814517,
				Theta:            -36.24735514,
				Gamma:            0.00002604,
				Vega:             156.51072692,
				HighPriceLimit:   8635,
				LowPriceLimit:    960,
				RiskFreeInterest: 0.0527,
			},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetOptionMarkPrice(t.Context(), tc.symbol)
			require.NoError(t, err, "GetOptionMarkPrice must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetOptionMarkPrice should decode every field")
				return
			}
			assert.NotEmpty(t, result, "GetOptionMarkPrice should return mark prices")
		})
	}
}

func TestGetEOptionsOrderbook(t *testing.T) {
	t.Parallel()
	_, err := e.GetEOptionsOrderbook(t.Context(), currency.EMPTYPAIR, 10)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetEOptionsOrderbook must reject an empty symbol")

	ensureTradablePairs(t)
	result, err := e.GetEOptionsOrderbook(t.Context(), optionsTradablePair, 10)
	require.NoError(t, err, "GetEOptionsOrderbook must not error")
	if mockTests {
		exp := &OptionsOrderBookResponse{
			Bids: []orderbook.Level{
				{
					Amount: 7.78,
					Price:  295,
				},
				{
					Amount: 49.9,
					Price:  290,
				},
				{
					Amount: 12.59,
					Price:  285,
				},
			},
			Asks: []orderbook.Level{
				{
					Amount: 18.98,
					Price:  300,
				},
				{
					Amount: 17.29,
					Price:  305,
				},
				{
					Amount: 46.92,
					Price:  310,
				},
			},
			TransactionTime: types.Time(time.UnixMilli(1791285203088)),
			LastUpdateID:    42791009448,
		}
		assert.Equal(t, exp, result, "GetEOptionsOrderbook should decode every field")
		return
	}
	assert.NotNil(t, result, "GetEOptionsOrderbook should return the order book")
}

func TestGetEOptionsRecentTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetEOptionsRecentTrades(t.Context(), currency.EMPTYPAIR, 10)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetEOptionsRecentTrades must reject an empty symbol")

	ensureTradablePairs(t)
	result, err := e.GetEOptionsRecentTrades(t.Context(), optionsTradablePair, 10)
	require.NoError(t, err, "GetEOptionsRecentTrades must not error")
	if mockTests {
		exp := []OptionsTrade{
			{
				ID:            2332864780648238564,
				TradeID:       466,
				Symbol:        "BTC-260410-72000-P",
				Price:         275,
				Quantity:      1,
				QuoteQuantity: 275,
				Side:          1,
				Time:          types.Time(time.UnixMilli(1791283421518)),
			},
			{
				ID:            2323857542484164932,
				TradeID:       469,
				Symbol:        "BTC-260410-72000-P",
				Price:         285,
				Quantity:      0.8,
				QuoteQuantity: 228,
				Side:          -1,
				Time:          types.Time(time.UnixMilli(1791283521651)),
			},
		}
		assert.Equal(t, exp, result, "GetEOptionsRecentTrades should decode every field")
		return
	}
	assert.NotNil(t, result, "GetEOptionsRecentTrades should return trades")
}

func TestGetEOptions24hrTickerPriceChangeStatistics(t *testing.T) {
	t.Parallel()
	ensureTradablePairs(t)
	for _, tc := range []struct {
		name   string
		symbol currency.Pair
		exp    []EOptionTicker
	}{
		{"every symbol", currency.EMPTYPAIR, []EOptionTicker{
			{
				Symbol:             "BTC-261225-85000-C",
				PriceChange:        255,
				PriceChangePercent: 0.041,
				LastPrice:          6480,
				LastQuantity:       0.02,
				Open:               6225,
				High:               6480,
				Low:                6225,
				Volume:             0.02,
				Amount:             129.6,
				BidPrice:           6935,
				AskPrice:           6985,
				OpenTime:           types.Time(time.Unix(1791017220, 0)),
				CloseTime:          types.Time(time.UnixMilli(1791128468463)),
				FirstTradeID:       214,
				TradeCount:         1,
				StrikePrice:        85000,
				ExercisePrice:      86162.79847826,
			},
			{
				Symbol:             "BTC-261225-85000-P",
				PriceChange:        345,
				PriceChangePercent: 0.071,
				LastPrice:          5210,
				LastQuantity:       0.01,
				Open:               4865,
				High:               6730,
				Low:                4865,
				Volume:             0.02,
				Amount:             101,
				BidPrice:           4855,
				AskPrice:           4865,
				OpenTime:           types.Time(time.Unix(1790949720, 0)),
				CloseTime:          types.Time(time.UnixMilli(1791177609822)),
				FirstTradeID:       70,
				TradeCount:         2,
				StrikePrice:        85000,
				ExercisePrice:      86162.79847826,
			},
		}},
		{"symbol", optionsTradablePair, []EOptionTicker{
			{
				Symbol:             "BTC-260410-72000-P",
				PriceChange:        345,
				PriceChangePercent: 0.071,
				LastPrice:          5210,
				LastQuantity:       0.01,
				Open:               4865,
				High:               6730,
				Low:                4865,
				Volume:             0.02,
				Amount:             101,
				BidPrice:           4855,
				AskPrice:           4865,
				OpenTime:           types.Time(time.Unix(1790949720, 0)),
				CloseTime:          types.Time(time.UnixMilli(1791177609822)),
				FirstTradeID:       70,
				TradeCount:         2,
				StrikePrice:        72000,
				ExercisePrice:      86162.79847826,
			},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetEOptions24hrTickerPriceChangeStatistics(t.Context(), tc.symbol)
			require.NoError(t, err, "GetEOptions24hrTickerPriceChangeStatistics must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetEOptions24hrTickerPriceChangeStatistics should decode every field")
				return
			}
			assert.NotEmpty(t, result, "GetEOptions24hrTickerPriceChangeStatistics should return tickers")
		})
	}
}

func TestSendOptionsAutoCancelAllOpenOrdersHeartbeat(t *testing.T) {
	t.Parallel()
	_, err := e.SendOptionsAutoCancelAllOpenOrdersHeartbeat(t.Context(), nil)
	require.ErrorIs(t, err, errUnderlyingIsRequired, "SendOptionsAutoCancelAllOpenOrdersHeartbeat must reject an empty underlying list")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SendOptionsAutoCancelAllOpenOrdersHeartbeat(t.Context(), []string{"BTCUSDT", "ETHUSDT"})
	require.NoError(t, err, "SendOptionsAutoCancelAllOpenOrdersHeartbeat must not error")
	if mockTests {
		exp := &OptionsAutoCancelAllOpenOrdersHeartbeatResponse{
			Underlyings: []string{
				"BTCUSDT",
				"ETHUSDT",
			},
		}
		assert.Equal(t, exp, result, "SendOptionsAutoCancelAllOpenOrdersHeartbeat should decode every field")
		return
	}
	assert.NotNil(t, result, "SendOptionsAutoCancelAllOpenOrdersHeartbeat should return the reset underlyings")
}

func TestGetAutoCancelAllOpenOrdersConfig(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAutoCancelAllOpenOrdersConfig(t.Context(), "BTCUSDT")
	require.NoError(t, err, "GetAutoCancelAllOpenOrdersConfig must not error")
	if mockTests {
		exp := &OptionsAutoCancelAllOpenOrdersConfigResponse{
			Underlying:    "BTCUSDT",
			CountdownTime: 100000,
		}
		assert.Equal(t, exp, result, "GetAutoCancelAllOpenOrdersConfig should decode every field")
		return
	}
	assert.NotNil(t, result, "GetAutoCancelAllOpenOrdersConfig should return the config")
}

func TestSetOptionsAutoCancelAllOpenOrders(t *testing.T) {
	t.Parallel()
	_, err := e.SetOptionsAutoCancelAllOpenOrders(t.Context(), "", 30*time.Second)
	require.ErrorIs(t, err, errUnderlyingIsRequired, "SetOptionsAutoCancelAllOpenOrders must reject an empty underlying")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SetOptionsAutoCancelAllOpenOrders(t.Context(), "BTCUSDT", 30*time.Second)
	require.NoError(t, err, "SetOptionsAutoCancelAllOpenOrders must not error")
	if mockTests {
		exp := &OptionsAutoCancelAllOpenOrdersConfigResponse{
			Underlying:    "BTCUSDT",
			CountdownTime: 30000,
		}
		assert.Equal(t, exp, result, "SetOptionsAutoCancelAllOpenOrders should decode every field")
	} else {
		assert.NotNil(t, result, "SetOptionsAutoCancelAllOpenOrders should return the config")
	}

	// A zero countdown turns the auto-cancel off, so it must be sent rather than rejected
	result, err = e.SetOptionsAutoCancelAllOpenOrders(t.Context(), "BTCUSDT", 0)
	require.NoError(t, err, "SetOptionsAutoCancelAllOpenOrders must not error turning the countdown off")
	if mockTests {
		assert.Equal(t, &OptionsAutoCancelAllOpenOrdersConfigResponse{Underlying: "BTCUSDT"}, result, "SetOptionsAutoCancelAllOpenOrders should return a zero countdown")
	}
}

func TestGetOptionsMarketMakerProtection(t *testing.T) {
	t.Parallel()
	_, err := e.GetOptionsMarketMakerProtection(t.Context(), "")
	require.ErrorIs(t, err, errUnderlyingIsRequired, "GetOptionsMarketMakerProtection must reject an empty underlying")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetOptionsMarketMakerProtection(t.Context(), "BTCUSDT")
	require.NoError(t, err, "GetOptionsMarketMakerProtection must not error")
	if mockTests {
		exp := &OptionsMarketMakerProtectionResponse{
			UnderlyingID:             2,
			Underlying:               "BTCUSDT",
			WindowTimeInMilliseconds: 3000,
			FrozenTimeInMilliseconds: 300000,
			QuantityLimit:            2,
			DeltaLimit:               2.3,
			LastTriggerTime:          types.Time(time.Unix(1744120800, 0)),
		}
		assert.Equal(t, exp, result, "GetOptionsMarketMakerProtection should decode every field")
		return
	}
	assert.NotNil(t, result, "GetOptionsMarketMakerProtection should return the config")
}

func TestResetOptionsMarketMakerProtection(t *testing.T) {
	t.Parallel()
	_, err := e.ResetOptionsMarketMakerProtection(t.Context(), "")
	require.ErrorIs(t, err, errUnderlyingIsRequired, "ResetOptionsMarketMakerProtection must reject an empty underlying")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.ResetOptionsMarketMakerProtection(t.Context(), "BTCUSDT")
	require.NoError(t, err, "ResetOptionsMarketMakerProtection must not error")
	if mockTests {
		exp := &OptionsMarketMakerProtectionResponse{
			UnderlyingID:             2,
			Underlying:               "BTCUSDT",
			WindowTimeInMilliseconds: 3000,
			FrozenTimeInMilliseconds: 300000,
			QuantityLimit:            2,
			DeltaLimit:               2.3,
			LastTriggerTime:          types.Time(time.Unix(1744120800, 0)),
		}
		assert.Equal(t, exp, result, "ResetOptionsMarketMakerProtection should decode every field")
		return
	}
	assert.NotNil(t, result, "ResetOptionsMarketMakerProtection should return the config")
}

func TestSetOptionsMarketMakerProtectionConfig(t *testing.T) {
	t.Parallel()
	_, err := e.SetOptionsMarketMakerProtectionConfig(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SetOptionsMarketMakerProtectionConfig must reject a nil request")
	_, err = e.SetOptionsMarketMakerProtectionConfig(t.Context(), &OptionsMarketMakerProtectionConfigRequest{WindowTime: 3 * time.Second, FrozenTime: 5 * time.Minute, QuantityLimit: 1.5, DeltaLimit: 1.5})
	require.ErrorIs(t, err, errUnderlyingIsRequired, "SetOptionsMarketMakerProtectionConfig must reject an empty underlying")
	_, err = e.SetOptionsMarketMakerProtectionConfig(t.Context(), &OptionsMarketMakerProtectionConfigRequest{Underlying: "BTCUSDT", WindowTime: 3 * time.Second, FrozenTime: 5 * time.Minute, DeltaLimit: 1.5})
	require.ErrorIs(t, err, errQuantityLimitRequired, "SetOptionsMarketMakerProtectionConfig must reject an empty quantity limit")
	_, err = e.SetOptionsMarketMakerProtectionConfig(t.Context(), &OptionsMarketMakerProtectionConfigRequest{Underlying: "BTCUSDT", WindowTime: 3 * time.Second, FrozenTime: 5 * time.Minute, QuantityLimit: 1.5})
	require.ErrorIs(t, err, errDeltaLimitRequired, "SetOptionsMarketMakerProtectionConfig must reject an empty delta limit")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SetOptionsMarketMakerProtectionConfig(t.Context(), &OptionsMarketMakerProtectionConfigRequest{
		Underlying:    "BTCUSDT",
		WindowTime:    3 * time.Second,
		FrozenTime:    5 * time.Minute,
		QuantityLimit: 1.5,
		DeltaLimit:    1.5,
	})
	require.NoError(t, err, "SetOptionsMarketMakerProtectionConfig must not error")
	if mockTests {
		exp := &OptionsMarketMakerProtectionResponse{
			UnderlyingID:             2,
			Underlying:               "BTCUSDT",
			WindowTimeInMilliseconds: 3000,
			FrozenTimeInMilliseconds: 300000,
			QuantityLimit:            1.5,
			DeltaLimit:               1.5,
			LastTriggerTime:          types.Time(time.Unix(1744120800, 0)),
		}
		assert.Equal(t, exp, result, "SetOptionsMarketMakerProtectionConfig should decode every field")
		return
	}
	assert.NotNil(t, result, "SetOptionsMarketMakerProtectionConfig should return the config")
}

func TestGetEOptionsAccountTradeList(t *testing.T) {
	t.Parallel()
	_, err := e.GetEOptionsAccountTradeList(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetEOptionsAccountTradeList must reject a nil request")
	_, err = e.GetEOptionsAccountTradeList(t.Context(), &OptionsAccountTradeListRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetEOptionsAccountTradeList must reject an empty symbol")

	ensureTradablePairs(t)
	startTime, endTime := getTime()
	_, err = e.GetEOptionsAccountTradeList(t.Context(), &OptionsAccountTradeListRequest{Symbol: optionsTradablePair, StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetEOptionsAccountTradeList must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetEOptionsAccountTradeList(t.Context(), &OptionsAccountTradeListRequest{Symbol: optionsTradablePair, StartTime: startTime, EndTime: endTime})
	require.NoError(t, err, "GetEOptionsAccountTradeList must not error")
	if mockTests {
		exp := []OptionsAccountTradeItem{
			{
				ID:             4611875134427365000,
				TradeID:        239,
				OrderID:        4611875134427365377,
				Symbol:         "BTC-260410-72000-P",
				Price:          1300,
				Quantity:       0.01,
				Fee:            -0.00624,
				RealizedProfit: 1.25,
				Side:           "SELL",
				Type:           "LIMIT",
				Liquidity:      "MAKER",
				Time:           types.Time(time.Unix(1744120800, 0)),
				PriceScale:     3,
				QuantityScale:  2,
				OptionSide:     "PUT",
				QuoteAsset:     currency.USDT,
			},
		}
		assert.Equal(t, exp, result, "GetEOptionsAccountTradeList should decode every field")
		return
	}
	assert.NotNil(t, result, "GetEOptionsAccountTradeList should return trades")
}

func TestCancelAllOptionsOrdersByUnderlying(t *testing.T) {
	t.Parallel()
	err := e.CancelAllOptionsOrdersByUnderlying(t.Context(), "")
	require.ErrorIs(t, err, errUnderlyingIsRequired, "CancelAllOptionsOrdersByUnderlying must reject an empty underlying")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	assert.NoError(t, e.CancelAllOptionsOrdersByUnderlying(t.Context(), "BTCUSDT"), "CancelAllOptionsOrdersByUnderlying should not error")
}

func TestCancelAllOptionOrdersOnSpecificSymbol(t *testing.T) {
	t.Parallel()
	err := e.CancelAllOptionOrdersOnSpecificSymbol(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CancelAllOptionOrdersOnSpecificSymbol must reject an empty symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	ensureTradablePairs(t)
	assert.NoError(t, e.CancelAllOptionOrdersOnSpecificSymbol(t.Context(), optionsTradablePair), "CancelAllOptionOrdersOnSpecificSymbol should not error")
}

func TestOptionsOrderParams(t *testing.T) {
	t.Parallel()
	_, err := e.optionsOrderParams(nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "optionsOrderParams must reject a nil request")
	_, err = e.optionsOrderParams(&OptionsOrderRequest{Side: "SELL", OrderType: "LIMIT", Quantity: 1, Price: 1})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "optionsOrderParams must reject an empty symbol")
	symbol := currency.Pair{Base: currency.BTC, Quote: currency.NewCode("260410-72000-P"), Delimiter: currency.DashDelimiter}
	_, err = e.optionsOrderParams(&OptionsOrderRequest{Symbol: symbol, OrderType: "LIMIT", Quantity: 1, Price: 1})
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "optionsOrderParams must reject an empty side")
	_, err = e.optionsOrderParams(&OptionsOrderRequest{Symbol: symbol, Side: "SELL", Quantity: 1, Price: 1})
	require.ErrorIs(t, err, order.ErrTypeIsInvalid, "optionsOrderParams must reject an empty order type")
	_, err = e.optionsOrderParams(&OptionsOrderRequest{Symbol: symbol, Side: "SELL", OrderType: "LIMIT", Price: 1})
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "optionsOrderParams must reject an empty quantity")
	_, err = e.optionsOrderParams(&OptionsOrderRequest{Symbol: symbol, Side: "SELL", OrderType: "LIMIT", Quantity: 1})
	require.ErrorIs(t, err, order.ErrPriceMustBeSetIfLimitOrder, "optionsOrderParams must reject a limit order without a price")

	params, err := e.optionsOrderParams(&OptionsOrderRequest{
		Symbol:                  symbol,
		Side:                    "SELL",
		OrderType:               "LIMIT",
		Quantity:                1000000,
		Price:                   0.00001,
		TimeInForce:             "GTX",
		ReduceOnly:              true,
		PostOnly:                true,
		NewOrderResponseType:    "RESULT",
		ClientOrderID:           "the-client-order-id",
		IsMarketMakerProtection: true,
		SelfTradePreventionMode: "EXPIRE_BOTH",
	})
	require.NoError(t, err, "optionsOrderParams must not error")
	exp := url.Values{
		"symbol":                  {"BTC-260410-72000-P"},
		"side":                    {"SELL"},
		"type":                    {"LIMIT"},
		"quantity":                {"1000000"},
		"price":                   {"0.00001"},
		"timeInForce":             {"GTX"},
		"reduceOnly":              {"true"},
		"postOnly":                {"true"},
		"newOrderRespType":        {"RESULT"},
		"clientOrderId":           {"the-client-order-id"},
		"isMmp":                   {"true"},
		"selfTradePreventionMode": {"EXPIRE_BOTH"},
	}
	assert.Equal(t, exp, params, "optionsOrderParams should send every parameter in plain decimal notation")
}

func TestPlaceBatchEOptionsOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceBatchEOptionsOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrEmptyParams, "PlaceBatchEOptionsOrder must reject an empty batch")
	_, err = e.PlaceBatchEOptionsOrder(t.Context(), []OptionsOrderRequest{{Side: "SELL", OrderType: "LIMIT", Quantity: 1, Price: 1}})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "PlaceBatchEOptionsOrder must reject an order without a symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	ensureTradablePairs(t)
	result, err := e.PlaceBatchEOptionsOrder(t.Context(), []OptionsOrderRequest{
		{Symbol: optionsTradablePair, Side: "SELL", OrderType: "LIMIT", Quantity: 0.01, Price: 1300, PostOnly: true, ClientOrderID: "the-client-order-id"},
		{Symbol: optionsTradablePair, Side: "BUY", OrderType: "LIMIT", Quantity: 0.01, Price: 1000, TimeInForce: "IOC", ClientOrderID: "the-client-order-id-2"},
	})
	require.NoError(t, err, "PlaceBatchEOptionsOrder must not error")
	if mockTests {
		exp := []OptionsOrderResponse{
			{
				OrderID:                 4611875134427365377,
				Symbol:                  "BTC-260410-72000-P",
				Price:                   1300,
				Quantity:                0.01,
				Side:                    "SELL",
				Type:                    "LIMIT",
				TimeInForce:             "GTC",
				UpdateTime:              types.Time(time.Unix(1744120801, 0)),
				Status:                  "NEW",
				ClientOrderID:           "the-client-order-id",
				PriceScale:              3,
				QuantityScale:           2,
				OptionSide:              "PUT",
				QuoteAsset:              currency.USDT,
				PostOnly:                true,
				CreateTime:              types.Time(time.Unix(1744120800, 0)),
				Source:                  "API",
				SelfTradePreventionMode: "EXPIRE_MAKER",
			},
			{
				OrderID:                 4611875134427365378,
				Symbol:                  "BTC-260410-72000-P",
				Price:                   1000,
				Quantity:                0.01,
				ExecutedQuantity:        0.01,
				Side:                    "BUY",
				Type:                    "LIMIT",
				TimeInForce:             "IOC",
				ReduceOnly:              true,
				UpdateTime:              types.Time(time.Unix(1744120801, 0)),
				Status:                  "FILLED",
				AveragePrice:            1300,
				ClientOrderID:           "the-client-order-id-2",
				PriceScale:              3,
				QuantityScale:           2,
				OptionSide:              "PUT",
				QuoteAsset:              currency.USDT,
				MarketMakerProtection:   true,
				Fee:                     0.0624,
				CreateTime:              types.Time(time.Unix(1744120800, 0)),
				Source:                  "API",
				SelfTradePreventionMode: "EXPIRE_TAKER",
			},
		}
		assert.Equal(t, exp, result, "PlaceBatchEOptionsOrder should decode every field")
		return
	}
	assert.NotEmpty(t, result, "PlaceBatchEOptionsOrder should return an entry per order")
}

func TestCancelBatchOptionsOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelBatchOptionsOrders(t.Context(), currency.EMPTYPAIR, []uint64{4611875134427365377}, nil)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CancelBatchOptionsOrders must reject an empty symbol")

	ensureTradablePairs(t)
	_, err = e.CancelBatchOptionsOrders(t.Context(), optionsTradablePair, nil, nil)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelBatchOptionsOrders must require an order ID or a client order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelBatchOptionsOrders(t.Context(), optionsTradablePair, []uint64{4611875134427365377}, []string{"the-client-order-id-2"})
	require.NoError(t, err, "CancelBatchOptionsOrders must not error")
	if mockTests {
		exp := []OptionsCancelledOrder{
			{
				OrderID:                 4611875134427365377,
				Symbol:                  "BTC-260410-72000-P",
				Price:                   1300,
				Quantity:                0.01,
				ExecutedQuantity:        0.004,
				Side:                    "SELL",
				Type:                    "LIMIT",
				TimeInForce:             "GTC",
				ReduceOnly:              true,
				UpdateTime:              types.Time(time.Unix(1744120801, 0)),
				Status:                  "CANCELLED",
				AveragePrice:            1300,
				ClientOrderID:           "the-client-order-id",
				PriceScale:              3,
				QuantityScale:           2,
				OptionSide:              "PUT",
				QuoteAsset:              currency.USDT,
				MarketMakerProtection:   true,
				Fee:                     0.0624,
				CreateTime:              types.Time(time.Unix(1744120800, 0)),
				Source:                  "API",
				SelfTradePreventionMode: "EXPIRE_MAKER",
			},
			{
				OrderID:                 4611875134427365378,
				Symbol:                  "BTC-260410-72000-P",
				Price:                   1300,
				Quantity:                0.01,
				Side:                    "SELL",
				Type:                    "LIMIT",
				TimeInForce:             "GTC",
				UpdateTime:              types.Time(time.Unix(1744120801, 0)),
				Status:                  "CANCELLED",
				ClientOrderID:           "the-client-order-id-2",
				PriceScale:              3,
				QuantityScale:           2,
				OptionSide:              "PUT",
				QuoteAsset:              currency.USDT,
				CreateTime:              types.Time(time.Unix(1744120800, 0)),
				Source:                  "API",
				SelfTradePreventionMode: "EXPIRE_MAKER",
			},
		}
		assert.Equal(t, exp, result, "CancelBatchOptionsOrders should decode every field")
		return
	}
	assert.NotEmpty(t, result, "CancelBatchOptionsOrders should return the cancelled orders")
}

func TestGetSingleEOptionsOrder(t *testing.T) {
	t.Parallel()
	_, err := e.GetSingleEOptionsOrder(t.Context(), currency.EMPTYPAIR, "", 4611875134427365377)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetSingleEOptionsOrder must reject an empty symbol")

	ensureTradablePairs(t)
	_, err = e.GetSingleEOptionsOrder(t.Context(), optionsTradablePair, "", 0)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetSingleEOptionsOrder must require an order ID or a client order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSingleEOptionsOrder(t.Context(), optionsTradablePair, "", 4611875134427365377)
	require.NoError(t, err, "GetSingleEOptionsOrder must not error")
	if mockTests {
		exp := &OptionsOrderStatusResponse{
			OrderID:                 4611875134427365377,
			Symbol:                  "BTC-260410-72000-P",
			Price:                   1300,
			Quantity:                0.01,
			ExecutedQuantity:        0.01,
			Side:                    "SELL",
			Type:                    "LIMIT",
			TimeInForce:             "GTC",
			ReduceOnly:              true,
			UpdateTime:              types.Time(time.Unix(1744120801, 0)),
			Status:                  "FILLED",
			AveragePrice:            1300,
			ClientOrderID:           "the-client-order-id",
			PriceScale:              3,
			QuantityScale:           2,
			OptionSide:              "PUT",
			QuoteAsset:              currency.USDT,
			MarketMakerProtection:   true,
			PostOnly:                true,
			CreateTime:              types.Time(time.Unix(1744120800, 0)),
			SelfTradePreventionMode: "EXPIRE_MAKER",
		}
		assert.Equal(t, exp, result, "GetSingleEOptionsOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSingleEOptionsOrder should return the order")
}

func TestNewOptionsOrder(t *testing.T) {
	t.Parallel()
	_, err := e.NewOptionsOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "NewOptionsOrder must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	ensureTradablePairs(t)
	result, err := e.NewOptionsOrder(t.Context(), &OptionsOrderRequest{
		Symbol:                  optionsTradablePair,
		Side:                    "SELL",
		OrderType:               "LIMIT",
		Quantity:                0.01,
		Price:                   1300,
		PostOnly:                true,
		NewOrderResponseType:    "RESULT",
		ClientOrderID:           "the-client-order-id",
		IsMarketMakerProtection: true,
	})
	require.NoError(t, err, "NewOptionsOrder must not error")
	if mockTests {
		exp := &OptionsOrderResponse{
			OrderID:                 4611875134427365377,
			Symbol:                  "BTC-260410-72000-P",
			Price:                   1300,
			Quantity:                0.01,
			ExecutedQuantity:        0.004,
			Side:                    "SELL",
			Type:                    "LIMIT",
			TimeInForce:             "GTC",
			ReduceOnly:              true,
			UpdateTime:              types.Time(time.Unix(1744120801, 0)),
			Status:                  "PARTIALLY_FILLED",
			AveragePrice:            1300,
			ClientOrderID:           "the-client-order-id",
			PriceScale:              3,
			QuantityScale:           2,
			OptionSide:              "PUT",
			QuoteAsset:              currency.USDT,
			MarketMakerProtection:   true,
			Fee:                     0.0249,
			PostOnly:                true,
			CreateTime:              types.Time(time.Unix(1744120800, 0)),
			Source:                  "API",
			SelfTradePreventionMode: "EXPIRE_MAKER",
		}
		assert.Equal(t, exp, result, "NewOptionsOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "NewOptionsOrder should return the order")
}

func TestCancelOptionsOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelOptionsOrder(t.Context(), currency.EMPTYPAIR, "213123", 4611875134427365377)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CancelOptionsOrder must reject an empty symbol")

	ensureTradablePairs(t)
	_, err = e.CancelOptionsOrder(t.Context(), optionsTradablePair, "", 0)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelOptionsOrder must require an order ID or a client order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelOptionsOrder(t.Context(), optionsTradablePair, "213123", 4611875134427365377)
	require.NoError(t, err, "CancelOptionsOrder must not error")
	if mockTests {
		exp := &OptionsCancelOrderResponse{
			OrderID:                 4611875134427365377,
			Symbol:                  "BTC-260410-72000-P",
			Price:                   1300,
			Quantity:                0.01,
			ExecutedQuantity:        0.004,
			Side:                    "SELL",
			Type:                    "LIMIT",
			TimeInForce:             "GTC",
			ReduceOnly:              true,
			UpdateTime:              types.Time(time.Unix(1744120801, 0)),
			Status:                  "CANCELLED",
			AveragePrice:            1300,
			ClientOrderID:           "213123",
			PriceScale:              3,
			QuantityScale:           2,
			OptionSide:              "PUT",
			QuoteAsset:              currency.USDT,
			MarketMakerProtection:   true,
			CreateDate:              types.Time(time.Unix(1744120800, 0)),
			Source:                  "API",
			SelfTradePreventionMode: "EXPIRE_MAKER",
		}
		assert.Equal(t, exp, result, "CancelOptionsOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "CancelOptionsOrder should return the order")
}

func TestGetOptionPositionInformation(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	ensureTradablePairs(t)
	result, err := e.GetOptionPositionInformation(t.Context(), optionsTradablePair)
	require.NoError(t, err, "GetOptionPositionInformation must not error")
	if mockTests {
		exp := []OptionPosition{
			{
				EntryPrice:    1000,
				Symbol:        "BTC-260410-72000-P",
				Side:          "SHORT",
				Quantity:      -0.1,
				MarkValue:     105.00138,
				UnrealizedPNL: -5.00138,
				MarkPrice:     1050.0138,
				StrikePrice:   72000,
				ExpiryDate:    types.Time(time.Unix(1775808000, 0)),
				PriceScale:    2,
				QuantityScale: 2,
				OptionSide:    "PUT",
				QuoteAsset:    currency.USDT,
				Time:          types.Time(time.UnixMilli(1762872654561)),
				BidQuantity:   0.02,
				AskQuantity:   0.01,
			},
		}
		assert.Equal(t, exp, result, "GetOptionPositionInformation should decode every field")
		return
	}
	assert.NotNil(t, result, "GetOptionPositionInformation should return positions")
}

func TestGetCurrentOpenOptionsOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetCurrentOpenOptionsOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetCurrentOpenOptionsOrders must reject a nil request")

	ensureTradablePairs(t)
	startTime, endTime := getTime()
	_, err = e.GetCurrentOpenOptionsOrders(t.Context(), &OptionsOpenOrdersRequest{Symbol: optionsTradablePair, StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetCurrentOpenOptionsOrders must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCurrentOpenOptionsOrders(t.Context(), &OptionsOpenOrdersRequest{Symbol: optionsTradablePair, OrderID: 4611875134427365377, StartTime: startTime, EndTime: endTime})
	require.NoError(t, err, "GetCurrentOpenOptionsOrders must not error")
	if mockTests {
		exp := []OptionsOpenOrder{
			{
				OrderID:                 4611875134427365377,
				Symbol:                  "BTC-260410-72000-P",
				Price:                   1300,
				Quantity:                0.01,
				ExecutedQuantity:        0.004,
				Side:                    "SELL",
				Type:                    "LIMIT",
				TimeInForce:             "GTC",
				ReduceOnly:              true,
				UpdateTime:              types.Time(time.Unix(1744120801, 0)),
				Status:                  "PARTIALLY_FILLED",
				AveragePrice:            1300,
				ClientOrderID:           "the-client-order-id",
				PriceScale:              3,
				QuantityScale:           2,
				OptionSide:              "PUT",
				QuoteAsset:              currency.USDT,
				MarketMakerProtection:   true,
				CreateTime:              types.Time(time.Unix(1744120800, 0)),
				SelfTradePreventionMode: "EXPIRE_MAKER",
			},
		}
		assert.Equal(t, exp, result, "GetCurrentOpenOptionsOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "GetCurrentOpenOptionsOrders should return orders")
}

func TestGetOptionsOrdersHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetOptionsOrdersHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetOptionsOrdersHistory must reject a nil request")
	_, err = e.GetOptionsOrdersHistory(t.Context(), &OptionsOrderHistoryRequest{Limit: 10})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetOptionsOrdersHistory must reject an empty symbol")

	ensureTradablePairs(t)
	startTime, endTime := getTime()
	_, err = e.GetOptionsOrdersHistory(t.Context(), &OptionsOrderHistoryRequest{Symbol: optionsTradablePair, StartTime: endTime, EndTime: startTime, Limit: 10})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetOptionsOrdersHistory must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetOptionsOrdersHistory(t.Context(), &OptionsOrderHistoryRequest{Symbol: optionsTradablePair, OrderID: 4611875134427365377, StartTime: startTime, EndTime: endTime, Limit: 10})
	require.NoError(t, err, "GetOptionsOrdersHistory must not error")
	if mockTests {
		exp := []OptionsOrderHistoryItem{
			{
				OrderID:               4611875134427365377,
				Symbol:                "BTC-260410-72000-P",
				Price:                 1300,
				Quantity:              0.01,
				ExecutedQuantity:      0.01,
				Side:                  "SELL",
				Type:                  "LIMIT",
				TimeInForce:           "GTC",
				ReduceOnly:            true,
				UpdateTime:            types.Time(time.Unix(1744120801, 0)),
				Status:                "FILLED",
				AveragePrice:          1300,
				ClientOrderID:         "the-client-order-id",
				PriceScale:            3,
				QuantityScale:         2,
				OptionSide:            "PUT",
				QuoteAsset:            currency.USDT,
				MarketMakerProtection: true,
				CreateTime:            types.Time(time.Unix(1744120800, 0)),
			},
		}
		assert.Equal(t, exp, result, "GetOptionsOrdersHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetOptionsOrdersHistory should return orders")
}

func TestGetUserOptionsExerciseRecord(t *testing.T) {
	t.Parallel()
	_, err := e.GetUserOptionsExerciseRecord(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUserOptionsExerciseRecord must reject a nil request")

	ensureTradablePairs(t)
	startTime, endTime := getTime()
	_, err = e.GetUserOptionsExerciseRecord(t.Context(), &OptionsUserExerciseRecordRequest{Symbol: optionsTradablePair, StartTime: endTime, EndTime: startTime, Limit: 10})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetUserOptionsExerciseRecord must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUserOptionsExerciseRecord(t.Context(), &OptionsUserExerciseRecordRequest{Symbol: optionsTradablePair, StartTime: startTime, EndTime: endTime, Limit: 10})
	require.NoError(t, err, "GetUserOptionsExerciseRecord must not error")
	if mockTests {
		exp := []UserOptionsExerciseRecord{
			{
				ID:            "1125899906842624042",
				Currency:      currency.USDT,
				Symbol:        "BTC-260410-72000-P",
				ExercisePrice: 72000,
				Quantity:      1,
				Amount:        1250.5,
				Fee:           1.25,
				CreateDate:    types.Time(time.Unix(1775808000, 0)),
				PriceScale:    2,
				QuantityScale: 2,
				OptionSide:    "PUT",
				PositionSide:  "LONG",
				QuoteAsset:    currency.USDT,
			},
		}
		assert.Equal(t, exp, result, "GetUserOptionsExerciseRecord should decode every field")
		return
	}
	assert.NotNil(t, result, "GetUserOptionsExerciseRecord should return exercise records")
}

func TestKeepaliveOptionsUserDataStream(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		_, err := e.StartOptionsUserDataStream(t.Context())
		require.NoError(t, err, "StartOptionsUserDataStream must not error")
	}
	assert.NoError(t, e.KeepaliveOptionsUserDataStream(t.Context()), "KeepaliveOptionsUserDataStream should not error")
}

func TestStartOptionsUserDataStream(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.StartOptionsUserDataStream(t.Context())
	require.NoError(t, err, "StartOptionsUserDataStream must not error")
	if mockTests {
		exp := &OptionsListenKeyResponse{
			ListenKey:  "pqia91ma19a5s61cv6a81va65sdf19v8a65a1a5s61cv6a81va65sdf19v8a65a1",
			Expiration: types.Time(time.UnixMilli(1762855900452)),
		}
		assert.Equal(t, exp, result, "StartOptionsUserDataStream should decode every field")
		return
	}
	assert.NotEmpty(t, result.ListenKey, "StartOptionsUserDataStream should return a listen key")
}

func TestCloseOptionsUserDataStream(t *testing.T) {
	t.Parallel()
	if !mockTests {
		// Closing invalidates the listen key every client of the account shares
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	assert.NoError(t, e.CloseOptionsUserDataStream(t.Context()), "CloseOptionsUserDataStream should not error")
}
