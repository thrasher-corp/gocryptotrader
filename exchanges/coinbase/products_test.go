package coinbase

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// assertRequestJWT asserts a request carries a JWT signed with the placeholder key for its method, host and path, as
// Coinbase requires of every authenticated endpoint's request; it reports with assert, as it runs in a handler
func assertRequestJWT(t *testing.T, r *http.Request) {
	t.Helper()
	token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !assert.True(t, ok, "the request should carry a bearer JWT") {
		return
	}
	jwt, err := parseTestJWT(token)
	if assert.NoError(t, err, "the request's JWT should verify") {
		assert.Equal(t, r.Method+" "+r.Host+r.URL.Path, jwt.Claims["uri"], "the request's JWT should be signed for its method, host and path")
	}
}

// newJWTCheckingServerExchange returns an exchange whose REST requests all receive one fixed response body, asserting
// each carries a JWT signed for it, which the mock recording does not check
func newJWTCheckingServerExchange(t *testing.T, body string) *Exchange {
	t.Helper()
	return newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertRequestJWT(t, r)
		_, err := w.Write([]byte(body))
		assert.NoError(t, err, "Writing the response should not error")
	}))
}

func TestGetBestBidAsk(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	resp, err := e.GetBestBidAsk(t.Context(), []string{testSpotProductID, "ETH-USD"})
	require.NoError(t, err, "GetBestBidAsk must not error")
	if mockTests {
		exp := []PriceBook{
			{
				ProductID: "BTC-USD",
				Bids:      []BookLevel{{Price: 82253.5, Size: 0.030142}},
				Asks:      []BookLevel{{Price: 82253.51, Size: 0.17994759}},
				Time:      time.Date(2026, 10, 19, 0, 54, 45, 737644000, time.UTC),
			},
			{
				ProductID: "ETH-USD",
				Bids:      []BookLevel{{Price: 2411.91, Size: 0.00089087}},
				Asks:      []BookLevel{{Price: 2411.92, Size: 2.542}},
				Time:      time.Date(2026, 10, 19, 0, 54, 46, 265221000, time.UTC),
			},
		}
		assert.Equal(t, exp, resp, "GetBestBidAsk should decode every field")
	} else {
		assert.Len(t, resp, 2, "GetBestBidAsk should return each product's book")
	}

	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertRequestJWT(t, r)
		assert.Equal(t, []string{testSpotProductID, "ETH-USD"}, r.URL.Query()["product_ids"], "GetBestBidAsk should send each product ID as its own product_ids param")
		_, err := w.Write([]byte(`{"pricebooks":[]}`))
		assert.NoError(t, err, "Writing the response should not error")
	}))
	resp, err = ex.GetBestBidAsk(t.Context(), []string{testSpotProductID, "ETH-USD"})
	require.NoError(t, err, "GetBestBidAsk must not error for an empty response")
	assert.Empty(t, resp, "GetBestBidAsk should return no books for an empty response")
	_, err = newUnavailableServerExchange(t).GetBestBidAsk(t.Context(), nil)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetBestBidAsk should return the request failure")
}

func TestGetMarketTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarketTrades(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetMarketTrades must reject a nil request")
	_, err = e.GetMarketTrades(t.Context(), &MarketTradesRequest{Limit: 5})
	require.ErrorIs(t, err, errProductIDRequired, "GetMarketTrades must require a product ID")
	_, err = e.GetMarketTrades(t.Context(), &MarketTradesRequest{ProductID: testSpotProductID})
	require.ErrorIs(t, err, errInvalidLimit, "GetMarketTrades must require a limit")
	_, err = e.GetMarketTrades(t.Context(), &MarketTradesRequest{ProductID: testSpotProductID, Limit: 5, Start: time.Unix(2, 0), End: time.Unix(1, 0)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetMarketTrades must refuse a reversed window")

	skipLiveWithoutCredentials(t)
	resp, err := e.GetMarketTrades(t.Context(), &MarketTradesRequest{ProductID: testSpotProductID, Limit: 5})
	require.NoError(t, err, "GetMarketTrades must not error")
	if mockTests {
		exp := &MarketTradesResponse{
			Trades: []MarketTrade{
				{
					TradeID:   "1107786249",
					ProductID: "BTC-USD",
					Price:     82246.58,
					Size:      0.00000006,
					Time:      time.Date(2026, 10, 19, 0, 54, 48, 302241000, time.UTC),
					Side:      "BUY",
					Exchange:  "coinbase",
				},
				{
					TradeID:   "1107786248",
					ProductID: "BTC-USD",
					Price:     82253.37,
					Size:      0.00000012,
					Time:      time.Date(2026, 10, 19, 0, 54, 47, 479784000, time.UTC),
					Side:      "BUY",
					Exchange:  "coinbase",
				},
				{
					TradeID:   "1107786247",
					ProductID: "BTC-USD",
					Price:     82253.37,
					Size:      0.00113775,
					Time:      time.Date(2026, 10, 19, 0, 54, 47, 479784000, time.UTC),
					Side:      "BUY",
					Exchange:  "coinbase",
				},
				{
					TradeID:   "1107786246",
					ProductID: "BTC-USD",
					Price:     82253.48,
					Size:      0.03,
					Time:      time.Date(2026, 10, 19, 0, 54, 47, 479784000, time.UTC),
					Side:      "BUY",
					Exchange:  "coinbase",
				},
				{
					TradeID:   "1107786245",
					ProductID: "BTC-USD",
					Price:     82253.5,
					Size:      0.00004364,
					Time:      time.Date(2026, 10, 19, 0, 54, 47, 479784000, time.UTC),
					Side:      "BUY",
					Exchange:  "coinbase",
				},
			},
			BestBid: 82246.58,
			BestAsk: 82246.59,
		}
		assert.Equal(t, exp, resp, "GetMarketTrades should decode every field")
	} else {
		assert.NotEmpty(t, resp.Trades, "GetMarketTrades should return trades")
		assert.Positive(t, resp.BestAsk.Float64(), "GetMarketTrades should return the best ask")
	}

	_, err = newJWTCheckingServerExchange(t, `null`).GetMarketTrades(t.Context(), &MarketTradesRequest{ProductID: testSpotProductID, Limit: 5})
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetMarketTrades should error for a null response")
	_, err = newUnavailableServerExchange(t).GetMarketTrades(t.Context(), &MarketTradesRequest{ProductID: testSpotProductID, Limit: 5})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetMarketTrades should return the request failure")
}

func TestGetProduct(t *testing.T) {
	t.Parallel()
	_, err := e.GetProduct(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetProduct must reject a nil request")
	_, err = e.GetProduct(t.Context(), &ProductRequest{GetTradabilityStatus: true})
	require.ErrorIs(t, err, errProductIDRequired, "GetProduct must require a product ID")

	skipLiveWithoutCredentials(t)
	resp, err := e.GetProduct(t.Context(), &ProductRequest{ProductID: testSpotProductID, GetTradabilityStatus: true})
	require.NoError(t, err, "GetProduct must not error")
	if mockTests {
		exp := &Product{
			ProductID:                    "BTC-USD",
			Price:                        82253.5,
			PricePercentageChange24Hour:  0.9857012787131,
			Volume24Hour:                 3030.47178149,
			VolumePercentageChange24Hour: 55.75999394976477,
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
			AliasTo:                      []string{"BTC-USDC"},
			BaseDisplaySymbol:            "BTC",
			QuoteDisplaySymbol:           "USD",
			PriceIncrement:               0.01,
			DisplayName:                  "BTC-USD",
			ProductVenue:                 "CBE",
			ApproximateQuote24HourVolume: 249266910.68,
			NewAt:                        time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			High24Hour:                   82454.5,
			Low24Hour:                    80564.57,
		}
		assert.Equal(t, exp, resp, "GetProduct should decode every field")
	} else {
		assert.Equal(t, testSpotProductID, resp.ProductID, "GetProduct should return the product")
		assert.Positive(t, resp.Price.Float64(), "GetProduct should return the product's price")
	}

	_, err = newJWTCheckingServerExchange(t, `null`).GetProduct(t.Context(), &ProductRequest{ProductID: testSpotProductID})
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetProduct should error for a null response")
	_, err = newUnavailableServerExchange(t).GetProduct(t.Context(), &ProductRequest{ProductID: testSpotProductID})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetProduct should return the request failure")
}

func TestGetProductBook(t *testing.T) {
	t.Parallel()
	_, err := e.GetProductBook(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetProductBook must reject a nil request")
	_, err = e.GetProductBook(t.Context(), &ProductBookRequest{Limit: 5})
	require.ErrorIs(t, err, errProductIDRequired, "GetProductBook must require a product ID")
	_, err = e.GetProductBook(t.Context(), &ProductBookRequest{ProductID: testSpotProductID, AggregationPriceIncrement: -1})
	require.ErrorIs(t, err, errInvalidPrice, "GetProductBook must refuse a negative aggregation increment")

	skipLiveWithoutCredentials(t)
	resp, err := e.GetProductBook(t.Context(), &ProductBookRequest{ProductID: testSpotProductID, Limit: 5})
	require.NoError(t, err, "GetProductBook must not error")
	if mockTests {
		exp := &ProductBookResponse{
			PriceBook: PriceBook{
				ProductID: "BTC-USD",
				Bids: []BookLevel{
					{Price: 82245.65, Size: 0.0182378},
					{Price: 82243.73, Size: 0.1837},
					{Price: 82242.42, Size: 0.0099},
					{Price: 82241.54, Size: 0.05233363},
					{Price: 82241.53, Size: 0.07278638},
				},
				Asks: []BookLevel{
					{Price: 82246.59, Size: 0.00340262},
					{Price: 82248.21, Size: 0.00083341},
					{Price: 82250.36, Size: 0.024},
					{Price: 82250.99, Size: 0.07278638},
					{Price: 82251.85, Size: 0.08843131},
				},
				Time: time.Date(2026, 10, 19, 0, 54, 50, 835785000, time.UTC),
			},
			Last:           82246.12,
			MidMarket:      82246.12,
			SpreadBPS:      0.11429045265,
			SpreadAbsolute: 0.94,
		}
		assert.Equal(t, exp, resp, "GetProductBook should decode every field")
	} else {
		assert.Len(t, resp.PriceBook.Bids, 5, "GetProductBook should return the limited bids")
		assert.Len(t, resp.PriceBook.Asks, 5, "GetProductBook should return the limited asks")
	}

	_, err = newJWTCheckingServerExchange(t, `null`).GetProductBook(t.Context(), &ProductBookRequest{ProductID: testSpotProductID})
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetProductBook should error for a null response")
	_, err = newUnavailableServerExchange(t).GetProductBook(t.Context(), &ProductBookRequest{ProductID: testSpotProductID})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetProductBook should return the request failure")
}

func TestGetProductCandles(t *testing.T) {
	t.Parallel()
	_, err := e.GetProductCandles(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetProductCandles must reject a nil request")
	_, err = e.GetProductCandles(t.Context(), &ProductCandlesRequest{Start: testCandlesStart, End: testCandlesStart.Add(time.Hour), Granularity: GranularityOneHour})
	require.ErrorIs(t, err, errProductIDRequired, "GetProductCandles must require a product ID")
	_, err = e.GetProductCandles(t.Context(), &ProductCandlesRequest{ProductID: testSpotProductID, Start: testCandlesStart, Granularity: GranularityOneHour})
	require.ErrorIs(t, err, errCandleWindowUnset, "GetProductCandles must require a window end")
	_, err = e.GetProductCandles(t.Context(), &ProductCandlesRequest{ProductID: testSpotProductID, Start: testCandlesStart.Add(time.Hour), End: testCandlesStart, Granularity: GranularityOneHour})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetProductCandles must refuse a reversed window")
	_, err = e.GetProductCandles(t.Context(), &ProductCandlesRequest{ProductID: testSpotProductID, Start: testCandlesStart, End: testCandlesStart.Add(time.Hour), Granularity: "ONE_WEEK"})
	require.ErrorIs(t, err, errInvalidGranularity, "GetProductCandles must refuse an unknown granularity")
	_, err = e.GetProductCandles(t.Context(), &ProductCandlesRequest{ProductID: testSpotProductID, Start: testCandlesStart, End: testCandlesStart.Add(351 * time.Minute), Granularity: GranularityOneMinute})
	require.ErrorIs(t, err, errCandleWindowTooLong, "GetProductCandles must refuse a window spanning more than 350 candles")
	_, err = e.GetProductCandles(t.Context(), &ProductCandlesRequest{ProductID: testSpotProductID, Start: testCandlesStart, End: testCandlesStart.Add(time.Hour), Granularity: GranularityOneHour, Limit: maximumCandles + 1})
	require.ErrorIs(t, err, errInvalidLimit, "GetProductCandles must refuse a limit above the maximum")

	skipLiveWithoutCredentials(t)
	resp, err := e.GetProductCandles(t.Context(), &ProductCandlesRequest{ProductID: testSpotProductID, Start: testCandlesStart, End: testCandlesStart.Add(6 * time.Hour), Granularity: GranularityOneHour, Limit: 3})
	require.NoError(t, err, "GetProductCandles must not error")
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
		}
		assert.Equal(t, exp, resp, "GetProductCandles should decode every field")
	} else {
		require.Len(t, resp, 3, "GetProductCandles must return the limited candles")
		assert.WithinRange(t, resp[0].Start.Time(), testCandlesStart, testCandlesStart.Add(6*time.Hour), "GetProductCandles should return candles within the window")
	}

	resp, err = newJWTCheckingServerExchange(t, `{"candles":[]}`).GetProductCandles(t.Context(), &ProductCandlesRequest{ProductID: testSpotProductID, Start: testCandlesStart, End: testCandlesStart.Add(time.Hour), Granularity: GranularityOneHour})
	require.NoError(t, err, "GetProductCandles must not error for an empty response")
	assert.Empty(t, resp, "GetProductCandles should return no candles for an empty response")
	_, err = newUnavailableServerExchange(t).GetProductCandles(t.Context(), &ProductCandlesRequest{ProductID: testSpotProductID, Start: testCandlesStart, End: testCandlesStart.Add(time.Hour), Granularity: GranularityOneHour})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetProductCandles should return the request failure")
}

func TestListProducts(t *testing.T) {
	t.Parallel()
	_, err := e.ListProducts(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ListProducts must reject a nil request")

	skipLiveWithoutCredentials(t)
	resp, err := e.ListProducts(t.Context(), &ListProductsRequest{Limit: 2, GetTradabilityStatus: true})
	require.NoError(t, err, "ListProducts must not error")
	if mockTests {
		exp := &ProductsResponse{
			Products: []Product{
				{
					ProductID:                    "BTC-USD",
					Price:                        82245.65,
					PricePercentageChange24Hour:  0.97606353983222,
					Volume24Hour:                 3030.47178149,
					VolumePercentageChange24Hour: 55.75999394976477,
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
					AliasTo:                      []string{"BTC-USDC"},
					BaseDisplaySymbol:            "BTC",
					QuoteDisplaySymbol:           "USD",
					PriceIncrement:               0.01,
					DisplayName:                  "BTC-USD",
					ProductVenue:                 "CBE",
					ApproximateQuote24HourVolume: 249243121.48,
					NewAt:                        time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
					High24Hour:                   82454.5,
					Low24Hour:                    80564.57,
				},
				{
					ProductID:                    "BTC-USDC",
					Price:                        82245.65,
					PricePercentageChange24Hour:  0.97606353983222,
					Volume24Hour:                 3030.47178149,
					VolumePercentageChange24Hour: 55.75999394976477,
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
					ApproximateQuote24HourVolume: 249243121.48,
					NewAt:                        time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
					High24Hour:                   82454.5,
					Low24Hour:                    80564.57,
				},
			},
			NumberOfProducts: 2,
			Pagination: PaginationMetadata{
				NextCursor: "cHJvZHVjdHNfY3Vyc29yOkJUQy1VU0RD",
				HasNext:    true,
			},
		}
		assert.Equal(t, exp, resp, "ListProducts should decode every field")
	} else {
		assert.Len(t, resp.Products, 2, "ListProducts should return the limited products")
	}

	_, err = newJWTCheckingServerExchange(t, `null`).ListProducts(t.Context(), &ListProductsRequest{})
	assert.ErrorIs(t, err, common.ErrNoResponse, "ListProducts should error for a null response")
	_, err = newUnavailableServerExchange(t).ListProducts(t.Context(), &ListProductsRequest{})
	assert.ErrorIs(t, err, request.ErrBadStatus, "ListProducts should return the request failure")
}
