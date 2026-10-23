package coinbase

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/types"
)

const testSpotProductID = "BTC-USD"

// testCandlesStart starts the candle window the mock fixtures were recorded for
var testCandlesStart = time.Date(2026, 10, 18, 0, 0, 0, 0, time.UTC)

func TestGetPublicMarketTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetPublicMarketTrades(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetPublicMarketTrades must reject a nil request")
	_, err = e.GetPublicMarketTrades(t.Context(), &MarketTradesRequest{Limit: 5})
	require.ErrorIs(t, err, errProductIDRequired, "GetPublicMarketTrades must require a product ID")
	_, err = e.GetPublicMarketTrades(t.Context(), &MarketTradesRequest{ProductID: testSpotProductID})
	require.ErrorIs(t, err, errInvalidLimit, "GetPublicMarketTrades must require a limit")
	_, err = e.GetPublicMarketTrades(t.Context(), &MarketTradesRequest{ProductID: testSpotProductID, Limit: maximumMarketTradesLimit + 1})
	require.ErrorIs(t, err, errInvalidLimit, "GetPublicMarketTrades must refuse a limit above the maximum")
	_, err = e.GetPublicMarketTrades(t.Context(), &MarketTradesRequest{ProductID: testSpotProductID, Limit: 5, Start: time.Unix(2, 0), End: time.Unix(1, 0)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetPublicMarketTrades must refuse a reversed window")

	resp, err := e.GetPublicMarketTrades(t.Context(), &MarketTradesRequest{ProductID: testSpotProductID, Limit: 5})
	require.NoError(t, err, "GetPublicMarketTrades must not error")
	if mockTests {
		exp := &MarketTradesResponse{
			Trades: []MarketTrade{
				{
					TradeID:   "1107776072",
					ProductID: "BTC-USD",
					Price:     82064.62,
					Size:      0.00000003,
					Time:      time.Date(2026, 10, 19, 0, 22, 37, 256118000, time.UTC),
					Side:      "BUY",
					Exchange:  "coinbase",
				},
				{
					TradeID:   "1107776071",
					ProductID: "BTC-USD",
					Price:     82064.62,
					Size:      0.00443456,
					Time:      time.Date(2026, 10, 19, 0, 22, 37, 187460000, time.UTC),
					Side:      "BUY",
					Exchange:  "coinbase",
				},
				{
					TradeID:   "1107776070",
					ProductID: "BTC-USD",
					Price:     82064.62,
					Size:      0.00000006,
					Time:      time.Date(2026, 10, 19, 0, 22, 36, 858633000, time.UTC),
					Side:      "BUY",
					Exchange:  "coinbase",
				},
				{
					TradeID:   "1107776069",
					ProductID: "BTC-USD",
					Price:     82064.62,
					Size:      0.00000003,
					Time:      time.Date(2026, 10, 19, 0, 22, 36, 713004000, time.UTC),
					Side:      "BUY",
					Exchange:  "coinbase",
				},
				{
					TradeID:   "1107776068",
					ProductID: "BTC-USD",
					Price:     82064.63,
					Size:      0.00024133,
					Time:      time.Date(2026, 10, 19, 0, 22, 36, 119702000, time.UTC),
					Side:      "SELL",
					Exchange:  "coinbase",
				},
			},
			BestBid: 82064.62,
			BestAsk: 82064.63,
		}
		assert.Equal(t, exp, resp, "GetPublicMarketTrades should decode every field")
	} else {
		assert.NotEmpty(t, resp.Trades, "GetPublicMarketTrades should return trades")
		assert.Positive(t, resp.BestAsk.Float64(), "GetPublicMarketTrades should return the best ask")
	}

	_, err = newResponseServerExchange(t, `null`).GetPublicMarketTrades(t.Context(), &MarketTradesRequest{ProductID: testSpotProductID, Limit: 5})
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetPublicMarketTrades should error for a null response")
	_, err = newUnavailableServerExchange(t).GetPublicMarketTrades(t.Context(), &MarketTradesRequest{ProductID: testSpotProductID, Limit: 5})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetPublicMarketTrades should return the request failure")
}

func TestMarketTradesParams(t *testing.T) {
	t.Parallel()
	params, err := marketTradesParams(&MarketTradesRequest{ProductID: testSpotProductID, Limit: 10, Start: time.Unix(100, 0), End: time.Unix(200, 0)})
	require.NoError(t, err, "marketTradesParams must not error")
	assert.Equal(t, "end=200&limit=10&start=100", params.Encode(), "marketTradesParams should send the window as Unix seconds")
}

func TestGetPublicProduct(t *testing.T) {
	t.Parallel()
	_, err := e.GetPublicProduct(t.Context(), "")
	require.ErrorIs(t, err, errProductIDRequired, "GetPublicProduct must require a product ID")

	resp, err := e.GetPublicProduct(t.Context(), testSpotProductID)
	require.NoError(t, err, "GetPublicProduct must not error")
	if mockTests {
		exp := &Product{
			ProductID:                    "BTC-USD",
			Price:                        82064.62,
			PricePercentageChange24Hour:  0.83251563577797,
			Volume24Hour:                 3001.99676484,
			VolumePercentageChange24Hour: 58.21332857711652,
			BaseIncrement:                0.00000001,
			QuoteIncrement:               0.01,
			QuoteMinimumSize:             1,
			QuoteMaximumSize:             150000000,
			BaseMinimumSize:              0.00000001,
			BaseMaximumSize:              3400,
			BaseName:                     "Bitcoin",
			QuoteName:                    "US Dollar",
			Status:                       "online",
			ProductType:                  "SPOT",
			QuoteCurrencyID:              "USD",
			BaseCurrencyID:               "BTC",
			AliasTo: []string{
				"BTC-USDC",
			},
			BaseDisplaySymbol:            "BTC",
			QuoteDisplaySymbol:           "USD",
			PriceIncrement:               0.01,
			DisplayName:                  "BTC-USD",
			ProductVenue:                 "CBE",
			ApproximateQuote24HourVolume: 246357723.75,
			NewAt:                        time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			High24Hour:                   82183.56,
			Low24Hour:                    80564.57,
		}
		assert.Equal(t, exp, resp, "GetPublicProduct should decode every field")
	} else {
		assert.Equal(t, testSpotProductID, resp.ProductID, "GetPublicProduct should return the product")
		assert.Positive(t, resp.Price.Float64(), "GetPublicProduct should return the product's price")
	}

	_, err = newResponseServerExchange(t, `null`).GetPublicProduct(t.Context(), testSpotProductID)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetPublicProduct should error for a null response")
	_, err = newUnavailableServerExchange(t).GetPublicProduct(t.Context(), testSpotProductID)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetPublicProduct should return the request failure")
}

func TestGetPublicProductBook(t *testing.T) {
	t.Parallel()
	_, err := e.GetPublicProductBook(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetPublicProductBook must reject a nil request")
	_, err = e.GetPublicProductBook(t.Context(), &ProductBookRequest{})
	require.ErrorIs(t, err, errProductIDRequired, "GetPublicProductBook must require a product ID")
	_, err = e.GetPublicProductBook(t.Context(), &ProductBookRequest{ProductID: testSpotProductID, AggregationPriceIncrement: -1})
	require.ErrorIs(t, err, errInvalidPrice, "GetPublicProductBook must refuse a negative aggregation increment")

	resp, err := e.GetPublicProductBook(t.Context(), &ProductBookRequest{ProductID: testSpotProductID, Limit: 5})
	require.NoError(t, err, "GetPublicProductBook must not error")
	if mockTests {
		exp := &ProductBookResponse{
			PriceBook: PriceBook{
				ProductID: "BTC-USD",
				Bids: []BookLevel{
					{
						Price: 82064.55,
						Size:  0.27404577,
					},
					{
						Price: 82062.9,
						Size:  0.02437154,
					},
					{
						Price: 82062.83,
						Size:  0.00599617,
					},
					{
						Price: 82061.56,
						Size:  0.07315264,
					},
					{
						Price: 82058.79,
						Size:  0.05271914,
					},
				},
				Asks: []BookLevel{
					{
						Price: 82064.63,
						Size:  0.18540826,
					},
					{
						Price: 82066.21,
						Size:  0.02437056,
					},
					{
						Price: 82068.48,
						Size:  0.0000362,
					},
					{
						Price: 82070.92,
						Size:  0.024,
					},
					{
						Price: 82074.98,
						Size:  0.07315264,
					},
				},
				Time: time.Date(2026, 10, 19, 0, 22, 38, 950675000, time.UTC),
			},
			Last:           82064.59,
			MidMarket:      82064.59,
			SpreadBPS:      0.009748414146,
			SpreadAbsolute: 0.08,
		}
		assert.Equal(t, exp, resp, "GetPublicProductBook should decode every field")
	} else {
		assert.Len(t, resp.PriceBook.Bids, 5, "GetPublicProductBook should return the limited bids")
		assert.Len(t, resp.PriceBook.Asks, 5, "GetPublicProductBook should return the limited asks")
	}

	_, err = newResponseServerExchange(t, `null`).GetPublicProductBook(t.Context(), &ProductBookRequest{ProductID: testSpotProductID})
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetPublicProductBook should error for a null response")
	_, err = newUnavailableServerExchange(t).GetPublicProductBook(t.Context(), &ProductBookRequest{ProductID: testSpotProductID})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetPublicProductBook should return the request failure")
}

func TestProductBookParams(t *testing.T) {
	t.Parallel()
	params, err := productBookParams(&ProductBookRequest{ProductID: testSpotProductID, Limit: 3, AggregationPriceIncrement: 0.00001})
	require.NoError(t, err, "productBookParams must not error")
	assert.Equal(t, "aggregation_price_increment=0.00001&limit=3&product_id=BTC-USD", params.Encode(), "productBookParams should send every field, the increment without exponent notation")
}

func TestGetPublicProductCandles(t *testing.T) {
	t.Parallel()
	_, err := e.GetPublicProductCandles(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetPublicProductCandles must reject a nil request")
	_, err = e.GetPublicProductCandles(t.Context(), &ProductCandlesRequest{})
	require.ErrorIs(t, err, errProductIDRequired, "GetPublicProductCandles must require a product ID")
	_, err = e.GetPublicProductCandles(t.Context(), &ProductCandlesRequest{ProductID: testSpotProductID, End: testCandlesStart, Granularity: GranularityOneHour})
	require.ErrorIs(t, err, errCandleWindowUnset, "GetPublicProductCandles must require a window start")
	_, err = e.GetPublicProductCandles(t.Context(), &ProductCandlesRequest{ProductID: testSpotProductID, Start: testCandlesStart.Add(time.Hour), End: testCandlesStart, Granularity: GranularityOneHour})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetPublicProductCandles must refuse a reversed window")
	_, err = e.GetPublicProductCandles(t.Context(), &ProductCandlesRequest{ProductID: testSpotProductID, Start: testCandlesStart, End: testCandlesStart.Add(time.Hour), Granularity: "ONE_WEEK"})
	require.ErrorIs(t, err, errInvalidGranularity, "GetPublicProductCandles must refuse an unknown granularity")
	_, err = e.GetPublicProductCandles(t.Context(), &ProductCandlesRequest{ProductID: testSpotProductID, Start: testCandlesStart, End: testCandlesStart.Add(351 * time.Minute), Granularity: GranularityOneMinute})
	require.ErrorIs(t, err, errCandleWindowTooLong, "GetPublicProductCandles must refuse a window spanning more than 350 candles")
	_, err = e.GetPublicProductCandles(t.Context(), &ProductCandlesRequest{ProductID: testSpotProductID, Start: testCandlesStart, End: testCandlesStart.Add(time.Hour), Granularity: GranularityOneHour, Limit: maximumCandles + 1})
	require.ErrorIs(t, err, errInvalidLimit, "GetPublicProductCandles must refuse a limit above the maximum")

	resp, err := e.GetPublicProductCandles(t.Context(), &ProductCandlesRequest{ProductID: testSpotProductID, Start: testCandlesStart, End: testCandlesStart.Add(6 * time.Hour), Granularity: GranularityOneHour})
	require.NoError(t, err, "GetPublicProductCandles must not error")
	if mockTests {
		exp := []Candle{
			{
				Start:  types.Time(time.Unix(1792303200, 0)),
				Low:    81561.76,
				High:   81684.9,
				Open:   81625.21,
				Close:  81613,
				Volume: 80.78403441,
			},
			{
				Start:  types.Time(time.Unix(1792299600, 0)),
				Low:    81548.75,
				High:   81767.06,
				Open:   81694.49,
				Close:  81625.21,
				Volume: 107.87421262,
			},
			{
				Start:  types.Time(time.Unix(1792296000, 0)),
				Low:    81674.7,
				High:   81856,
				Open:   81842.72,
				Close:  81694.49,
				Volume: 43.05822171,
			},
			{
				Start:  types.Time(time.Unix(1792292400, 0)),
				Low:    81667.57,
				High:   81969.84,
				Open:   81720.46,
				Close:  81842.71,
				Volume: 113.82555896,
			},
			{
				Start:  types.Time(time.Unix(1792288800, 0)),
				Low:    81473.18,
				High:   81730.39,
				Open:   81558.84,
				Close:  81720.47,
				Volume: 128.52979853,
			},
		}
		assert.Equal(t, exp, resp, "GetPublicProductCandles should decode every field")
	} else {
		require.NotEmpty(t, resp, "GetPublicProductCandles must return candles")
		assert.WithinRange(t, resp[0].Start.Time(), testCandlesStart, testCandlesStart.Add(6*time.Hour), "GetPublicProductCandles should return candles within the window")
	}

	_, err = newUnavailableServerExchange(t).GetPublicProductCandles(t.Context(), &ProductCandlesRequest{ProductID: testSpotProductID, Start: testCandlesStart, End: testCandlesStart.Add(time.Hour), Granularity: GranularityOneHour})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetPublicProductCandles should return the request failure")
}

func TestProductCandlesParams(t *testing.T) {
	t.Parallel()
	params, err := productCandlesParams(&ProductCandlesRequest{ProductID: testSpotProductID, Start: time.Unix(600, 0), End: time.Unix(600+350*60, 0), Granularity: GranularityOneMinute, Limit: 350})
	require.NoError(t, err, "productCandlesParams must accept a window of exactly 350 candles")
	assert.Equal(t, "end=21600&granularity=ONE_MINUTE&limit=350&start=600", params.Encode(), "productCandlesParams should send the window as Unix seconds")
}

func TestCandleDuration(t *testing.T) {
	t.Parallel()
	for granularity, exp := range map[string]time.Duration{
		GranularityOneMinute:     time.Minute,
		GranularityFiveMinute:    5 * time.Minute,
		GranularityFifteenMinute: 15 * time.Minute,
		GranularityThirtyMinute:  30 * time.Minute,
		GranularityOneHour:       time.Hour,
		GranularityTwoHour:       2 * time.Hour,
		GranularityFourHour:      4 * time.Hour,
		GranularitySixHour:       6 * time.Hour,
		GranularityOneDay:        24 * time.Hour,
	} {
		d, err := candleDuration(granularity)
		require.NoErrorf(t, err, "candleDuration must not error for %s", granularity)
		assert.Equalf(t, exp, d, "candleDuration should return %s's interval", granularity)
	}
	_, err := candleDuration("UNKNOWN_GRANULARITY")
	assert.ErrorIs(t, err, errInvalidGranularity, "candleDuration should refuse an unknown granularity")
}

func TestGetPublicServerTime(t *testing.T) {
	t.Parallel()
	resp, err := e.GetPublicServerTime(t.Context())
	require.NoError(t, err, "GetPublicServerTime must not error")
	if mockTests {
		exp := &ServerTimeResponse{
			ISO:          time.Date(2026, 10, 19, 0, 22, 36, 0, time.UTC),
			EpochSeconds: types.Time(time.Unix(1792369356, 0)),
			EpochMillis:  types.Time(time.UnixMilli(1792369356892)),
		}
		assert.Equal(t, exp, resp, "GetPublicServerTime should decode every field")
	} else {
		assert.WithinDuration(t, time.Now(), resp.EpochMillis.Time(), time.Minute, "GetPublicServerTime should return the current time")
	}

	_, err = newResponseServerExchange(t, `null`).GetPublicServerTime(t.Context())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetPublicServerTime should error for a null response")
	_, err = newUnavailableServerExchange(t).GetPublicServerTime(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetPublicServerTime should return the request failure")
}

func TestListPublicProducts(t *testing.T) {
	t.Parallel()
	_, err := e.ListPublicProducts(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ListPublicProducts must reject a nil request")

	resp, err := e.ListPublicProducts(t.Context(), &ListProductsRequest{Limit: 2})
	require.NoError(t, err, "ListPublicProducts must not error")
	if mockTests {
		exp := &ProductsResponse{
			Products: []Product{
				{
					ProductID:                    "BTC-USD",
					Price:                        82064.62,
					PricePercentageChange24Hour:  0.83251563577797,
					Volume24Hour:                 3001.99676484,
					VolumePercentageChange24Hour: 58.21332857711652,
					BaseIncrement:                0.00000001,
					QuoteIncrement:               0.01,
					QuoteMinimumSize:             1,
					QuoteMaximumSize:             150000000,
					BaseMinimumSize:              0.00000001,
					BaseMaximumSize:              3400,
					BaseName:                     "Bitcoin",
					QuoteName:                    "US Dollar",
					Status:                       "online",
					ProductType:                  "SPOT",
					QuoteCurrencyID:              "USD",
					BaseCurrencyID:               "BTC",
					AliasTo: []string{
						"BTC-USDC",
					},
					BaseDisplaySymbol:            "BTC",
					QuoteDisplaySymbol:           "USD",
					PriceIncrement:               0.01,
					DisplayName:                  "BTC-USD",
					ProductVenue:                 "CBE",
					ApproximateQuote24HourVolume: 246357723.75,
					NewAt:                        time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
					High24Hour:                   82183.56,
					Low24Hour:                    80564.57,
				},
				{
					ProductID:                    "BTC-USDC",
					Price:                        82064.62,
					PricePercentageChange24Hour:  0.83251563577797,
					Volume24Hour:                 3001.99676484,
					VolumePercentageChange24Hour: 58.21332857711652,
					BaseIncrement:                0.00000001,
					QuoteIncrement:               0.01,
					QuoteMinimumSize:             1,
					QuoteMaximumSize:             150000000,
					BaseMinimumSize:              0.00000001,
					BaseMaximumSize:              3400,
					BaseName:                     "Bitcoin",
					QuoteName:                    "USDC",
					Status:                       "online",
					ProductType:                  "SPOT",
					QuoteCurrencyID:              "USDC",
					BaseCurrencyID:               "BTC",
					Alias:                        "BTC-USD",
					AliasTo:                      []string{},
					BaseDisplaySymbol:            "BTC",
					QuoteDisplaySymbol:           "USD",
					PriceIncrement:               0.01,
					DisplayName:                  "BTC-USDC",
					ProductVenue:                 "CBE",
					ApproximateQuote24HourVolume: 246357723.75,
					NewAt:                        time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
					High24Hour:                   82183.56,
					Low24Hour:                    80564.57,
				},
			},
			NumberOfProducts: 2,
			Pagination: PaginationMetadata{
				NextCursor: "cHJvZHVjdHNfY3Vyc29yOkJUQy1VU0RD",
				HasNext:    true,
			},
		}
		assert.Equal(t, exp, resp, "ListPublicProducts should decode every spot field")
	} else {
		assert.Len(t, resp.Products, 2, "ListPublicProducts should return the limited products")
	}

	resp, err = e.ListPublicProducts(t.Context(), &ListProductsRequest{Limit: 2, ProductType: ProductTypeFuture, ContractExpiryType: ContractExpiryTypeExpiring})
	require.NoError(t, err, "ListPublicProducts must not error for futures")
	if mockTests {
		exp := &ProductsResponse{
			Products: []Product{
				{
					ProductID:                    "BIP-20DEC30-CDE",
					Price:                        82070,
					PricePercentageChange24Hour:  0.7859511236645,
					Volume24Hour:                 124337,
					VolumePercentageChange24Hour: 155.7690329747187,
					BaseIncrement:                1,
					QuoteIncrement:               0.01,
					QuoteMaximumSize:             100000000,
					BaseMinimumSize:              1,
					BaseMaximumSize:              5000,
					QuoteName:                    "US Dollar",
					ProductType:                  "FUTURE",
					QuoteCurrencyID:              "USD",
					FCMTradingSessionDetails: &FCMTradingSessionDetails{
						IsSessionOpen: true,
						OpenTime:      time.Date(2026, 10, 16, 22, 0, 0, 0, time.UTC),
						CloseTime:     time.Date(2026, 10, 19, 21, 0, 0, 0, time.UTC),
						SessionState:  "FCM_TRADING_SESSION_STATE_OPEN",
						ClosedReason:  "FCM_TRADING_SESSION_CLOSED_REASON_UNDEFINED",
						Maintenance: &FCMScheduledMaintenance{
							StartTime: time.Date(2026, 10, 24, 10, 0, 0, 0, time.UTC),
							EndTime:   time.Date(2026, 10, 24, 19, 0, 0, 0, time.UTC),
						},
					},
					MidMarketPrice:               82095,
					AliasTo:                      []string{},
					QuoteDisplaySymbol:           "USD",
					PriceIncrement:               5,
					DisplayName:                  "BTC PERP",
					ProductVenue:                 "FCM",
					ApproximateQuote24HourVolume: 102043375.9,
					NewAt:                        time.Date(2025, 7, 18, 13, 0, 0, 0, time.UTC),
					High24Hour:                   82220,
					Low24Hour:                    80620,
					FutureProductDetails: &FutureProductDetails{
						Venue:                  "cde",
						ContractCode:           "BIP",
						ContractExpiry:         time.Date(2089, 12, 30, 16, 0, 0, 0, time.UTC),
						ContractSize:           0.01,
						ContractRootUnit:       "BTC",
						GroupDescription:       "nano BTC Perp Futures",
						ContractExpiryTimezone: "Europe/London",
						GroupShortDescription:  "nano BTC Perp",
						RiskManagedBy:          "MANAGED_BY_FCM",
						ContractExpiryType:     "EXPIRING",
						PerpetualDetails: &PerpetualProductDetails{
							OpenInterest: 173743,
						},
						ContractDisplayName:      "BTC PERP",
						TimeToExpiryMilliseconds: 1994427438773,
						ContractExpiryName:       "2030-12",
						TwentyFourBySeven:        true,
						FundingInterval:          Duration(3600 * time.Second),
						OpenInterest:             173743,
						FundingRate:              0.000017,
						FundingTime:              time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC),
						DisplayName:              "Bitcoin Perpetual",
						RegionEnabled:            map[string]bool{},
						IntradayMarginRate: &MarginRate{
							LongMarginRate:  0.100079,
							ShortMarginRate: 0.0999811,
						},
						OvernightMarginRate: &MarginRate{
							LongMarginRate:  0.21025,
							ShortMarginRate: 0.256625,
						},
						SettlementPrice:  81520,
						FuturesAssetType: "FUTURES_ASSET_TYPE_CRYPTO",
						IndexPrice:       82009.434163,
						FuturesAssetTypes: []string{
							"FUTURES_ASSET_TYPE_CRYPTO",
						},
						TradingHoursType: "TRADING_HOURS_TYPE_UNSPECIFIED",
					},
				},
				{
					ProductID:                    "BIT-30OCT26-CDE",
					Price:                        82110,
					PricePercentageChange24Hour:  0.6990434142752,
					Volume24Hour:                 30545,
					VolumePercentageChange24Hour: 509.9241214057508,
					BaseIncrement:                1,
					QuoteIncrement:               0.01,
					QuoteMaximumSize:             100000000,
					BaseMinimumSize:              1,
					BaseMaximumSize:              5000,
					QuoteName:                    "US Dollar",
					ProductType:                  "FUTURE",
					QuoteCurrencyID:              "USD",
					FCMTradingSessionDetails: &FCMTradingSessionDetails{
						IsSessionOpen: true,
						OpenTime:      time.Date(2026, 10, 16, 22, 0, 0, 0, time.UTC),
						CloseTime:     time.Date(2026, 10, 19, 21, 0, 0, 0, time.UTC),
						SessionState:  "FCM_TRADING_SESSION_STATE_OPEN",
						ClosedReason:  "FCM_TRADING_SESSION_CLOSED_REASON_UNDEFINED",
						Maintenance: &FCMScheduledMaintenance{
							StartTime: time.Date(2026, 10, 24, 10, 0, 0, 0, time.UTC),
							EndTime:   time.Date(2026, 10, 24, 19, 0, 0, 0, time.UTC),
						},
					},
					MidMarketPrice:               82222.5,
					AliasTo:                      []string{},
					QuoteDisplaySymbol:           "USD",
					PriceIncrement:               5,
					DisplayName:                  "BTC 30 OCT 26",
					ProductVenue:                 "FCM",
					ApproximateQuote24HourVolume: 25080499.5,
					High24Hour:                   82310,
					Low24Hour:                    80725,
					FutureProductDetails: &FutureProductDetails{
						Venue:                  "cde",
						ContractCode:           "BIT",
						ContractExpiry:         time.Date(2026, 10, 30, 16, 0, 0, 0, time.UTC),
						ContractSize:           0.01,
						ContractRootUnit:       "BTC",
						GroupDescription:       "nano Bitcoin Futures",
						ContractExpiryTimezone: "Europe/London",
						GroupShortDescription:  "nano BTC",
						RiskManagedBy:          "MANAGED_BY_FCM",
						ContractExpiryType:     "EXPIRING",
						PerpetualDetails: &PerpetualProductDetails{
							OpenInterest: 120887,
						},
						ContractDisplayName:      "BTC 30 OCT 26",
						TimeToExpiryMilliseconds: 1006638773,
						ContractExpiryName:       "2026-10",
						TwentyFourBySeven:        true,
						OpenInterest:             120887,
						DisplayName:              "Bitcoin Futures (Oct 2026)",
						RegionEnabled:            map[string]bool{},
						IntradayMarginRate: &MarginRate{
							LongMarginRate:  0.1000828,
							ShortMarginRate: 0.0999804,
						},
						OvernightMarginRate: &MarginRate{
							LongMarginRate:  0.207125,
							ShortMarginRate: 0.2535,
						},
						SettlementPrice:  81700,
						FuturesAssetType: "FUTURES_ASSET_TYPE_CRYPTO",
						FuturesAssetTypes: []string{
							"FUTURES_ASSET_TYPE_CRYPTO",
						},
						TradingHoursType: "TRADING_HOURS_TYPE_UNSPECIFIED",
					},
				},
			},
			NumberOfProducts: 2,
			Pagination: PaginationMetadata{
				NextCursor: "cHJvZHVjdHNfY3Vyc29yOkJJVC0zME9DVDI2LUNERQ==",
				HasNext:    true,
			},
		}
		assert.Equal(t, exp, resp, "ListPublicProducts should decode every future field")
	} else {
		require.NotEmpty(t, resp.Products, "ListPublicProducts must return futures")
		assert.NotNil(t, resp.Products[0].FutureProductDetails, "ListPublicProducts should return each future's details")
	}

	_, err = newResponseServerExchange(t, `null`).ListPublicProducts(t.Context(), &ListProductsRequest{})
	assert.ErrorIs(t, err, common.ErrNoResponse, "ListPublicProducts should error for a null response")
	_, err = newUnavailableServerExchange(t).ListPublicProducts(t.Context(), &ListProductsRequest{})
	assert.ErrorIs(t, err, request.ErrBadStatus, "ListPublicProducts should return the request failure")
}

func TestProductsParams(t *testing.T) {
	t.Parallel()
	req := &ListProductsRequest{
		Limit:                  1,
		Offset:                 2,
		ProductType:            ProductTypeFuture,
		ProductIDs:             []string{"BIT-30OCT26-CDE", "ET-30OCT26-CDE"},
		ContractExpiryType:     ContractExpiryTypeExpiring,
		ExpiringContractStatus: ExpiringContractStatusAll,
		GetAllProducts:         true,
		ProductsSortOrder:      ProductsSortOrderListTimeDescending,
		Cursor:                 "abc",
		FuturesUnderlyingType:  "FUTURES_UNDERLYING_TYPE_SPOT",
		UserCountryCode:        "AU",
		ProductVenue:           ProductVenueDeribit,
		Expired:                true,
		GetTradabilityStatus:   true,
	}
	assert.Equal(t, "contract_expiry_type=EXPIRING&cursor=abc&expired=true&expiring_contract_status=STATUS_ALL&futures_underlying_type=FUTURES_UNDERLYING_TYPE_SPOT&get_all_products=true&limit=1&offset=2&product_ids=BIT-30OCT26-CDE&product_ids=ET-30OCT26-CDE&product_type=FUTURE&product_venue=DERIBIT&products_sort_order=PRODUCTS_SORT_ORDER_LIST_TIME_DESCENDING&user_country_code=AU",
		productsParams(req, false).Encode(), "productsParams should send every field but the tradability flag for the public list")
	assert.Equal(t, "true", productsParams(req, true).Get("get_tradability_status"), "productsParams should send the tradability flag for the authenticated list")
	assert.Empty(t, productsParams(&ListProductsRequest{}, true).Encode(), "productsParams should send nothing for an empty request")
}

func TestDurationUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var d Duration
	require.NoError(t, d.UnmarshalJSON([]byte(`"3600s"`)), "UnmarshalJSON must not error")
	assert.Equal(t, time.Hour, d.Duration(), "UnmarshalJSON should decode seconds")
	require.NoError(t, d.UnmarshalJSON([]byte(`null`)), "UnmarshalJSON must not error for null")
	assert.Zero(t, d, "UnmarshalJSON should leave null zero")
	d = Duration(time.Hour)
	require.NoError(t, d.UnmarshalJSON([]byte(`""`)), "UnmarshalJSON must not error for an empty string")
	assert.Zero(t, d, "UnmarshalJSON should leave an empty string zero")
	assert.ErrorIs(t, d.UnmarshalJSON([]byte(`"an hour"`)), errInvalidDuration, "UnmarshalJSON should refuse a malformed duration")
	assert.ErrorIs(t, d.UnmarshalJSON([]byte(`3600`)), errInvalidDuration, "UnmarshalJSON should refuse a number")
}
