package kraken

import (
	"math"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

// futuresTestTicker is the PF_XBTUSD ticker the tickers fixtures hold
var futuresTestTicker = FuturesTicker{
	Symbol:                           "PF_XBTUSD",
	Last:                             81783,
	LastTime:                         time.Date(2026, 10, 9, 0, 34, 6, 429728331, time.UTC),
	LastSize:                         0.0099,
	Tag:                              "perpetual",
	Pair:                             "XBT:USD",
	MarkPrice:                        81780.434623335,
	Bid:                              81782,
	BidSize:                          0.0225,
	Ask:                              81784,
	AskSize:                          0.0873,
	Volume24Hour:                     8076.6192,
	QuoteVolume24Hour:                661565749.685,
	VolumeWeightedAveragePrice24Hour: 81911.2221714,
	OpenInterest:                     2357.1126,
	Open24Hour:                       83241,
	High24Hour:                       83487,
	Low24Hour:                        80320,
	FundingRate:                      1.4840707878906,
	FundingRatePrediction:            1.30091458365,
	RelativeFundingRate:              0.000018166270833333,
	RelativeFundingRatePrediction:    0.000015901875,
	IndexPrice:                       81770.17,
	Change24Hour:                     -1.75,
}

func TestGetFuturesTickers(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesTickers(t.Context(), &FuturesTickersRequest{Pairs: currency.Pairs{currency.EMPTYPAIR}})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetFuturesTickers must reject an empty pair")

	for _, tc := range []struct {
		name string
		req  *FuturesTickersRequest
		exp  *FuturesTickersResponse
	}{
		{
			name: "every futures market",
			exp: &FuturesTickersResponse{
				Tickers: []FuturesTicker{
					futuresTestTicker,
					{
						Symbol:                           "FF_XBTUSD_261225",
						Last:                             82429,
						LastTime:                         time.Date(2026, 10, 9, 0, 28, 28, 698106696, time.UTC),
						LastSize:                         0.0001,
						Tag:                              "quarter",
						Pair:                             "XBT:USD",
						MarkPrice:                        82457.182699103,
						Bid:                              82411,
						BidSize:                          0.0016,
						Ask:                              82623,
						AskSize:                          0.0019,
						Volume24Hour:                     48.2406,
						QuoteVolume24Hour:                3976515.2019,
						VolumeWeightedAveragePrice24Hour: 82430.88191067,
						OpenInterest:                     43.1221,
						Open24Hour:                       84375,
						High24Hour:                       84380,
						Low24Hour:                        81516,
						Suspended:                        true,
						IndexPrice:                       81735.84,
						Change24Hour:                     -2.31,
					},
				},
				ServerTime: time.Date(2026, 10, 9, 0, 34, 8, 878000000, time.UTC),
			},
		},
		{
			name: "options",
			req:  &FuturesTickersRequest{ContractTypes: []string{"options"}},
			exp: &FuturesTickersResponse{
				Tickers: []FuturesTicker{
					{
						Symbol:                           "OF_XBTUSD_261030_85000_P",
						Last:                             5074,
						LastTime:                         time.Date(2026, 10, 8, 17, 1, 12, 938077192, time.UTC),
						LastSize:                         1,
						Tag:                              "month",
						Pair:                             "XBT:USD",
						MarkPrice:                        4429.08265781467,
						Bid:                              4380,
						BidSize:                          2,
						Ask:                              4475,
						AskSize:                          3,
						Volume24Hour:                     4,
						QuoteVolume24Hour:                20296,
						VolumeWeightedAveragePrice24Hour: 5074,
						OpenInterest:                     12,
						Open24Hour:                       5020,
						High24Hour:                       5110,
						Low24Hour:                        4990,
						ExtrinsicValue:                   1412.97751988982,
						MarkImpliedVolatility:            0.32968840788,
						IndexPrice:                       81735.84,
						Change24Hour:                     1.08,
						Greeks: FuturesOptionGreeks{
							ImpliedVolatility: 0.329733331356438,
							Delta:             -0.660429663114145,
							Gamma:             0.0000560698741738512,
							Vega:              7254.74552254084,
							Theta:             -20487.1892918115,
							Rho:               -3419.59881938167,
						},
					},
				},
				ServerTime: time.Date(2026, 10, 9, 0, 34, 52, 412000000, time.UTC),
			},
		},
		{
			name: "contract types and pairs",
			req: &FuturesTickersRequest{
				ContractTypes: []string{"flexible_futures", "futures_inverse"},
				Pairs: currency.Pairs{
					currency.NewPairWithDelimiter("PF", "EURUSD", currency.UnderscoreDelimiter),
					currency.NewPairWithDelimiter("PI", "XBTUSD", currency.UnderscoreDelimiter),
				},
			},
			exp: &FuturesTickersResponse{
				Tickers: []FuturesTicker{
					{
						Symbol:                           "PF_EURUSD",
						Last:                             1.12332,
						LastTime:                         time.Date(2026, 10, 8, 16, 17, 57, 933075667, time.UTC),
						LastSize:                         350,
						Tag:                              "perpetual",
						Pair:                             "EUR:USD",
						MarkPrice:                        1.12250993857,
						Bid:                              1.1218,
						BidSize:                          336,
						Ask:                              1.12603,
						AskSize:                          24644,
						Volume24Hour:                     356,
						QuoteVolume24Hour:                399.90192,
						VolumeWeightedAveragePrice24Hour: 1.12332,
						OpenInterest:                     4330,
						Open24Hour:                       1.12301,
						High24Hour:                       1.12345,
						Low24Hour:                        1.12298,
						FundingRate:                      0.000187291640827084,
						FundingRatePrediction:            0.000187285572758334,
						RelativeFundingRate:              0.000167120229166667,
						RelativeFundingRatePrediction:    0.000167070091666667,
						IndexPrice:                       1.121,
						PostOnly:                         true,
						Change24Hour:                     0.03,
						IsUnderlyingMarketClosed:         true,
					},
					{
						Symbol:                           "PI_XBTUSD",
						Last:                             81641.5,
						LastTime:                         time.Date(2026, 10, 9, 0, 3, 26, 32848197, time.UTC),
						LastSize:                         1,
						Tag:                              "perpetual",
						Pair:                             "XBT:USD",
						MarkPrice:                        81807.156629089,
						Bid:                              81746,
						BidSize:                          81,
						Ask:                              81904.5,
						AskSize:                          95,
						Volume24Hour:                     596069,
						QuoteVolume24Hour:                596102,
						VolumeWeightedAveragePrice24Hour: 82633.52771491,
						OpenInterest:                     2162910,
						Open24Hour:                       83224,
						High24Hour:                       83324,
						Low24Hour:                        80576,
						FundingRate:                      1.318994364e-09,
						FundingRatePrediction:            1.343640021e-09,
						RelativeFundingRate:              0.000107753595833333,
						RelativeFundingRatePrediction:    0.00010987885,
						IndexPrice:                       81738.9,
						Change24Hour:                     -1.9,
					},
				},
				ServerTime: time.Date(2026, 10, 9, 0, 34, 59, 193000000, time.UTC),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetFuturesTickers(t.Context(), tc.req)
			require.NoError(t, err, "GetFuturesTickers must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetFuturesTickers should decode every field")
				return
			}
			assert.NotEmpty(t, result.Tickers, "GetFuturesTickers should return tickers")
		})
	}
}

func TestGetFuturesTicker(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesTicker(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetFuturesTicker must reject an empty pair")

	result, err := e.GetFuturesTicker(t.Context(), futuresTestPair)
	require.NoError(t, err, "GetFuturesTicker must not error")
	if mockTests {
		exp := &FuturesTickerResponse{Ticker: futuresTestTicker, ServerTime: time.Date(2026, 10, 9, 0, 34, 8, 878000000, time.UTC)}
		assert.Equal(t, exp, result, "GetFuturesTicker should decode every field")
		return
	}
	assert.Equal(t, "PF_XBTUSD", result.Ticker.Symbol, "GetFuturesTicker should return the requested market's ticker")
}

func TestGetFuturesOrderbook(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesOrderbook(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetFuturesOrderbook must reject an empty pair")

	result, err := e.GetFuturesOrderbook(t.Context(), futuresTestPair)
	require.NoError(t, err, "GetFuturesOrderbook must not error")
	if mockTests {
		exp := &FuturesOrderbookResponse{
			OrderBook: FuturesOrderbook{
				Asks: []FuturesOrderbookLevel{{Price: 81784, Size: 0.0001}, {Price: 81787, Size: 0.0267}, {Price: 81788, Size: 0.0079}},
				Bids: []FuturesOrderbookLevel{{Price: 81781, Size: 0.5012}, {Price: 81782, Size: 0.0413}, {Price: 81783, Size: 0.1323}},
			},
			ServerTime: time.Date(2026, 10, 9, 0, 34, 10, 410000000, time.UTC),
		}
		assert.Equal(t, exp, result, "GetFuturesOrderbook should decode every field")
		return
	}
	assert.NotEmpty(t, result.OrderBook.Bids, "GetFuturesOrderbook should return bids")
}

func TestGetFuturesTradeHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesTradeHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFuturesTradeHistory must reject a nil request")
	_, err = e.GetFuturesTradeHistory(t.Context(), &FuturesTradeHistoryRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetFuturesTradeHistory must reject an empty pair")

	result, err := e.GetFuturesTradeHistory(t.Context(), &FuturesTradeHistoryRequest{Pair: futuresTestPair})
	require.NoError(t, err, "GetFuturesTradeHistory must not error")
	if mockTests {
		exp := &FuturesTradeHistoryResponse{
			History: []FuturesTrade{
				{Price: 81830, Side: "buy", Size: 0.8988, Time: time.Date(2026, 10, 9, 0, 33, 31, 277047879, time.UTC), TradeID: 2, Type: "fill", UID: "2a615649-9ca2-4239-99fa-5c6f4a3186f2", SequenceID: 102223859},
				{Price: 81831, Side: "sell", Size: 0.0357, Time: time.Date(2026, 10, 9, 0, 33, 31, 283564147, time.UTC), TradeID: 1, Type: "liquidation", UID: "6469aaa4-09a7-43ab-bc48-29e0021df4d9", SequenceID: 102223860},
			},
			ServerTime: time.Date(2026, 10, 9, 0, 34, 11, 984000000, time.UTC),
		}
		assert.Equal(t, exp, result, "GetFuturesTradeHistory should decode every field")
	} else {
		assert.NotEmpty(t, result.History, "GetFuturesTradeHistory should return trades")
	}

	// A LastTime outside UTC proves it is sent in UTC
	lastTime := time.Date(2026, 10, 9, 10, 36, 10, 123456789, time.FixedZone("AEST", 10*60*60))
	result, err = e.GetFuturesTradeHistory(t.Context(), &FuturesTradeHistoryRequest{Pair: futuresTestPair, LastTime: lastTime, IncludeMTFData: true})
	require.NoError(t, err, "GetFuturesTradeHistory must not error with every parameter")
	if mockTests {
		exp := &FuturesTradeHistoryResponse{
			History: []FuturesTrade{
				{
					Price:                         81777,
					Side:                          "buy",
					Size:                          0.0001,
					Time:                          time.Date(2026, 10, 9, 0, 36, 8, 46564502, time.UTC),
					TradeID:                       2,
					Type:                          "fill",
					UID:                           "be1656cd-9c3d-4d7c-af86-6aa4a49d361f",
					SequenceID:                    102224271,
					InstrumentIdentificationType:  "ISIN",
					ISIN:                          "GB00BQ84JX13",
					ExecutionVenue:                "CRYP",
					PriceNotation:                 "MONE",
					PriceCurrency:                 currency.USD,
					NotionalAmount:                8.1777,
					NotionalCurrency:              currency.USD,
					PublicationTime:               time.Date(2026, 10, 9, 0, 36, 8, 46564502, time.UTC),
					PublicationVenue:              "CRYP",
					TransactionIdentificationCode: "BE1656CD9C3D4D7CAF866AA4A49D361F",
				},
				{
					Price:                         81782,
					Side:                          "sell",
					Size:                          0.0154,
					Time:                          time.Date(2026, 10, 9, 0, 36, 8, 55435815, time.UTC),
					TradeID:                       1,
					Type:                          "partial liquidation",
					UID:                           "ffdd2ae2-d2ef-4fd9-86c8-d32e3f63371b",
					SequenceID:                    102224285,
					InstrumentIdentificationType:  "ISIN",
					ISIN:                          "GB00BQ84JX13",
					ExecutionVenue:                "CRYP",
					PriceNotation:                 "MONE",
					PriceCurrency:                 currency.USD,
					NotionalAmount:                1259.4428,
					NotionalCurrency:              currency.USD,
					PublicationTime:               time.Date(2026, 10, 9, 0, 36, 8, 155435815, time.UTC),
					PublicationVenue:              "CRYP",
					TransactionIdentificationCode: "FFDD2AE2D2EF4FD986C8D32E3F63371B",
					ToBeCleared:                   true,
				},
			},
			ServerTime: time.Date(2026, 10, 9, 0, 36, 48, 31000000, time.UTC),
		}
		assert.Equal(t, exp, result, "GetFuturesTradeHistory should decode every MTF field")
		return
	}
	for i := range result.History {
		assert.NotEmptyf(t, result.History[i].ISIN, "GetFuturesTradeHistory should return the MTF data of trade %d", i)
		assert.Truef(t, result.History[i].Time.Before(lastTime), "GetFuturesTradeHistory should return trade %d from before LastTime", i)
	}
}

func TestFuturesOrderbookLevelUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var level FuturesOrderbookLevel
	require.NoError(t, level.UnmarshalJSON([]byte(`[81784,0.0001]`)), "UnmarshalJSON must not error for a whole level")
	assert.Equal(t, FuturesOrderbookLevel{Price: 81784, Size: 0.0001}, level, "UnmarshalJSON should decode the price and size")
	err := level.UnmarshalJSON([]byte(`[81784]`))
	assert.ErrorIs(t, err, errUnexpectedLength, "UnmarshalJSON should reject a short level")
}

func TestFuturesTradeUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var trade FuturesTrade
	require.NoError(t, json.Unmarshal([]byte(`{"sequence_id":"18446744073709551615","size":"0.5"}`), &trade), "Unmarshal must not error")
	assert.Equal(t, FuturesTrade{SequenceID: math.MaxUint64, Size: 0.5}, trade, "Unmarshal should keep a 64-bit sequence ID exact and accept the size as the string Kraken documents")
}

func TestFuturesMarketSymbol(t *testing.T) {
	t.Parallel()
	_, err := e.futuresMarketSymbol(currency.EMPTYPAIR)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "futuresMarketSymbol should reject an empty pair")
	for _, pair := range []currency.Pair{currency.NewPairWithDelimiter("PF", "", currency.UnderscoreDelimiter), currency.NewPairWithDelimiter("", "XBTUSD", currency.UnderscoreDelimiter)} {
		_, err := e.futuresMarketSymbol(pair)
		assert.ErrorIsf(t, err, currency.ErrCurrencyPairEmpty, "futuresMarketSymbol should reject %q, which lacks a base or a quote", pair)
	}
	for pair, exp := range map[currency.Pair]string{futuresTestPair: "PF_XBTUSD", piXBTUSDPair: "PI_XBTUSD"} {
		symbol, err := e.futuresMarketSymbol(pair)
		require.NoErrorf(t, err, "futuresMarketSymbol must not error for %s", pair)
		assert.Equalf(t, exp, symbol, "futuresMarketSymbol should format %s as its market symbol", pair)
	}
}

func TestFuturesContractTypeParams(t *testing.T) {
	t.Parallel()
	assert.Empty(t, futuresContractTypeParams(nil), "futuresContractTypeParams should set nothing without contract types")
	exp := url.Values{"contractType": {"futures_inverse", "flexible_futures"}}
	assert.Equal(t, exp, futuresContractTypeParams([]string{"futures_inverse", "flexible_futures"}), "futuresContractTypeParams should repeat the parameter for each type")
}
