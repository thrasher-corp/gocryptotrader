package bitstamp

import (
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetCurrencies(t *testing.T) {
	t.Parallel()
	currencies, err := e.GetCurrencies(t.Context())
	require.NoError(t, err, "GetCurrencies must not error")
	i := slices.IndexFunc(currencies, func(c CurrencyResponse) bool { return c.Currency.Equal(currency.BTC) })
	require.NotEqual(t, -1, i, "GetCurrencies must list BTC")
	btc := currencies[i]
	if mockTests {
		assert.Equal(t, CurrencyResponse{
			Name:              "Bitcoin",
			Currency:          currency.BTC,
			Type:              "crypto",
			Symbol:            "₿",
			Decimals:          8,
			Logo:              "https://assets.bitstamp.net/assets/logos/btc.svg",
			AvailableSupply:   19934406,
			DepositEnabled:    true,
			WithdrawalEnabled: true,
			Networks: []CurrencyNetwork{
				{Network: "bitcoin", WithdrawalMinimumAmount: 0.00006, WithdrawalDecimals: 8, DepositEnabled: true, WithdrawalEnabled: true},
				{Network: "xrpl", WithdrawalDecimals: 8},
			},
		}, btc, "GetCurrencies should decode every BTC field")
		return
	}
	assert.Equal(t, "crypto", btc.Type, "Type should be correct")
	assert.Equal(t, uint8(8), btc.Decimals, "Decimals should be correct")
	assert.Positive(t, btc.AvailableSupply.Float64(), "AvailableSupply should be positive")
	assert.True(t, slices.ContainsFunc(btc.Networks, func(n CurrencyNetwork) bool { return n.Network == "bitcoin" }), "BTC should be listed on the bitcoin network")
}

func TestGetTickers(t *testing.T) {
	t.Parallel()
	tickers, err := e.GetTickers(t.Context())
	require.NoError(t, err, "GetTickers must not error")
	if mockTests {
		require.Len(t, tickers, 10, "GetTickers must return every market's ticker")
		exp := TickerResponse{
			Last:                       84246.67,
			High:                       84900,
			Low:                        82873.04,
			VolumeWeightedAveragePrice: 84052.46,
			Volume:                     1588.01930719,
			BestBid:                    84250.32,
			BestAsk:                    84250.33,
			Timestamp:                  types.Time(time.Unix(1790317267, 0)),
			Open:                       84383.03,
			Open24Hour:                 84110.88,
			PercentChange24Hour:        0.16,
			Side:                       OrderSide(order.Sell),
			MarketType:                 MarketTypeSpot,
			Market:                     "BTC/USD",
		}
		assert.Equal(t, exp, tickers[0], "GetTickers should decode a spot ticker")
		exp = TickerResponse{
			Last:                       84055,
			High:                       84922,
			Low:                        82895,
			VolumeWeightedAveragePrice: 84071,
			Volume:                     64.26666,
			BestBid:                    84260,
			BestAsk:                    84261,
			Timestamp:                  types.Time(time.Unix(1790317267, 0)),
			Open:                       84431,
			Open24Hour:                 84221,
			PercentChange24Hour:        -0.2,
			Side:                       OrderSide(order.Buy),
			MarketType:                 MarketTypePerpetual,
			MarkPrice:                  84250.99780466,
			IndexPrice:                 84241.79066666667,
			OpenInterest:               48.09714,
			OpenInterestValue:          4052232.0365504245,
			Market:                     "BTC/USD-PERP",
		}
		assert.Equal(t, exp, tickers[8], "GetTickers should decode a perpetual ticker")
	}
	marketTypes := make(map[string]bool)
	for i := range tickers {
		assert.NotEmpty(t, tickers[i].Market, "Market should be set")
		assert.Positive(t, tickers[i].Last.Float64(), "Last should be positive")
		marketTypes[tickers[i].MarketType] = true
	}
	assert.True(t, marketTypes[MarketTypeSpot], "GetTickers should return spot markets")
	assert.True(t, marketTypes[MarketTypePerpetual], "GetTickers should return perpetual markets")
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := e.GetTicker(t.Context(), currency.EMPTYPAIR)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetTicker should error on an empty pair")
	_, err = e.GetHourlyTicker(t.Context(), currency.EMPTYPAIR)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetHourlyTicker should error on an empty pair")

	for _, tc := range []struct {
		pair       currency.Pair
		marketType string
		exp        map[string]*TickerResponse
	}{
		{
			pair:       spotPair,
			marketType: MarketTypeSpot,
			exp: map[string]*TickerResponse{
				"GetTicker": {
					Last:                       84246.67,
					High:                       84900,
					Low:                        82873.04,
					VolumeWeightedAveragePrice: 84052.46,
					Volume:                     1588.01336151,
					BestBid:                    84250.32,
					BestAsk:                    84250.33,
					Timestamp:                  types.Time(time.Unix(1790317268, 0)),
					Open:                       84383.03,
					Open24Hour:                 84118.33,
					PercentChange24Hour:        0.15,
					Side:                       OrderSide(order.Sell),
					MarketType:                 MarketTypeSpot,
				},
				"GetHourlyTicker": {
					Last:                       84246.67,
					High:                       84263.32,
					Low:                        83966.76,
					VolumeWeightedAveragePrice: 84056.79,
					Volume:                     14.90884959,
					BestBid:                    84250.32,
					BestAsk:                    84250.33,
					Timestamp:                  types.Time(time.Unix(1790317270, 0)),
					Open:                       84175.39,
					Side:                       OrderSide(order.Sell),
					MarketType:                 MarketTypeSpot,
				},
			},
		},
		{
			pair:       perpetualPair,
			marketType: MarketTypePerpetual,
			exp: map[string]*TickerResponse{
				"GetTicker": {
					Last:                       84055,
					High:                       84922,
					Low:                        82895,
					VolumeWeightedAveragePrice: 84071,
					Volume:                     64.26666,
					BestBid:                    84260,
					BestAsk:                    84261,
					Timestamp:                  types.Time(time.Unix(1790317270, 0)),
					Open:                       84431,
					Open24Hour:                 84221,
					PercentChange24Hour:        -0.2,
					Side:                       OrderSide(order.Buy),
					MarketType:                 MarketTypePerpetual,
					MarkPrice:                  84249.73980309,
					IndexPrice:                 84240.324,
					OpenInterest:               48.09714,
					OpenInterestValue:          4052171.530272792,
				},
				"GetHourlyTicker": {
					Last:                       84055,
					High:                       84114,
					Low:                        84055,
					VolumeWeightedAveragePrice: 84107,
					Volume:                     0.54543,
					BestBid:                    84260,
					BestAsk:                    84261,
					Timestamp:                  types.Time(time.Unix(1790317270, 0)),
					Open:                       84114,
					Side:                       OrderSide(order.Buy),
					MarketType:                 MarketTypePerpetual,
					MarkPrice:                  84249.73980309,
					IndexPrice:                 84240.324,
					OpenInterest:               48.09714,
					OpenInterestValue:          4052171.530272792,
				},
			},
		},
	} {
		for name, get := range map[string]func() (*TickerResponse, error){
			"GetTicker":       func() (*TickerResponse, error) { return e.GetTicker(t.Context(), tc.pair) },
			"GetHourlyTicker": func() (*TickerResponse, error) { return e.GetHourlyTicker(t.Context(), tc.pair) },
		} {
			tick, err := get()
			require.NoErrorf(t, err, "%s must not error for %s", name, tc.pair)
			require.NotNilf(t, tick, "%s must return a ticker for %s", name, tc.pair)
			if mockTests {
				assert.Equalf(t, tc.exp[name], tick, "%s should decode the ticker for %s", name, tc.pair)
				continue
			}
			assert.Positivef(t, tick.Last.Float64(), "%s Last should be positive for %s", name, tc.pair)
			assert.Positivef(t, tick.BestBid.Float64(), "%s BestBid should be positive for %s", name, tc.pair)
			assert.Positivef(t, tick.BestAsk.Float64(), "%s BestAsk should be positive for %s", name, tc.pair)
			assert.NotZerof(t, tick.Timestamp.Time(), "%s Timestamp should be set for %s", name, tc.pair)
			assert.Containsf(t, []order.Side{order.Buy, order.Sell}, tick.Side.Side(), "%s Side should be valid for %s", name, tc.pair)
			assert.Equalf(t, tc.marketType, tick.MarketType, "%s MarketType should be correct for %s", name, tc.pair)
			if tc.marketType == MarketTypePerpetual {
				assert.Positivef(t, tick.MarkPrice.Float64(), "%s MarkPrice should be positive for %s", name, tc.pair)
				assert.Positivef(t, tick.IndexPrice.Float64(), "%s IndexPrice should be positive for %s", name, tc.pair)
			}
		}
	}
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderbook(t.Context(), currency.EMPTYPAIR, OrderbookGroupedByPrice)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetOrderbook should error on an empty pair")
	_, err = e.GetOrderbook(t.Context(), spotPair, OrderbookUngroupedWithOrderIDs+1)
	assert.ErrorIs(t, err, errInvalidOrderbookGrouping, "GetOrderbook should error on an invalid grouping")

	for _, tc := range []struct {
		pair     currency.Pair
		grouping OrderbookGrouping
		exp      *OrderbookResponse // The mock book's timestamps and best levels
	}{
		{
			pair:     spotPair,
			grouping: OrderbookUngrouped,
			exp: &OrderbookResponse{
				Timestamp:      types.Time(time.Unix(1790317271, 0)),
				Microtimestamp: types.Time(time.UnixMicro(1790317271348059)),
				Bids:           []OrderbookLevel{{Price: 84250.32, Amount: 0.125}},
				Asks:           []OrderbookLevel{{Price: 84250.33, Amount: 0.23}},
			},
		},
		{
			pair:     spotPair,
			grouping: OrderbookGroupedByPrice,
			exp: &OrderbookResponse{
				Timestamp:      types.Time(time.Unix(1790317272, 0)),
				Microtimestamp: types.Time(time.UnixMicro(1790317272953412)),
				Bids:           []OrderbookLevel{{Price: 84250.32, Amount: 0.46625964}},
				Asks:           []OrderbookLevel{{Price: 84250.33, Amount: 0.57086879}},
			},
		},
		{
			pair:     spotPair,
			grouping: OrderbookUngroupedWithOrderIDs,
			exp: &OrderbookResponse{
				Timestamp:      types.Time(time.Unix(1790317274, 0)),
				Microtimestamp: types.Time(time.UnixMicro(1790317274758218)),
				Bids:           []OrderbookLevel{{Price: 84250.32, Amount: 0.125, OrderID: 2054071468261386}},
				Asks:           []OrderbookLevel{{Price: 84250.33, Amount: 0.23, OrderID: 2054071468728320}},
			},
		},
		{
			pair:     perpetualPair,
			grouping: OrderbookGroupedByPrice,
			exp: &OrderbookResponse{
				Timestamp:      types.Time(time.Unix(1790317277, 0)),
				Microtimestamp: types.Time(time.UnixMicro(1790317277763643)),
				Bids:           []OrderbookLevel{{Price: 84258, Amount: 0.33625}},
				Asks:           []OrderbookLevel{{Price: 84259, Amount: 0.45819}},
			},
		},
	} {
		ob, err := e.GetOrderbook(t.Context(), tc.pair, tc.grouping)
		require.NoErrorf(t, err, "GetOrderbook must not error for %s grouping %d", tc.pair, tc.grouping)
		if mockTests {
			require.Lenf(t, ob.Bids, 10, "GetOrderbook must return every bid for %s grouping %d", tc.pair, tc.grouping)
			require.Lenf(t, ob.Asks, 10, "GetOrderbook must return every ask for %s grouping %d", tc.pair, tc.grouping)
			top := &OrderbookResponse{Timestamp: ob.Timestamp, Microtimestamp: ob.Microtimestamp, Bids: ob.Bids[:1], Asks: ob.Asks[:1]}
			assert.Equalf(t, tc.exp, top, "GetOrderbook should decode the book for %s grouping %d", tc.pair, tc.grouping)
		}
		assert.NotZerof(t, ob.Microtimestamp.Time(), "Microtimestamp should be set for %s grouping %d", tc.pair, tc.grouping)
		for _, levels := range [][]OrderbookLevel{ob.Bids, ob.Asks} {
			require.NotEmptyf(t, levels, "levels must not be empty for %s grouping %d", tc.pair, tc.grouping)
			assert.Positivef(t, levels[0].Price, "Price should be positive for %s grouping %d", tc.pair, tc.grouping)
			assert.Positivef(t, levels[0].Amount, "Amount should be positive for %s grouping %d", tc.pair, tc.grouping)
			if tc.grouping == OrderbookUngroupedWithOrderIDs {
				assert.NotZerof(t, levels[0].OrderID, "OrderID should be set for %s grouping %d", tc.pair, tc.grouping)
			} else {
				assert.Zerof(t, levels[0].OrderID, "OrderID should not be set for %s grouping %d", tc.pair, tc.grouping)
			}
		}
	}
}

func TestGetTransactions(t *testing.T) {
	t.Parallel()
	_, err := e.GetTransactions(t.Context(), currency.EMPTYPAIR, "")
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetTransactions should error on an empty pair")
	_, err = e.GetTransactions(t.Context(), spotPair, "week")
	assert.ErrorIs(t, err, errInvalidTransactionPeriod, "GetTransactions should error on an invalid time period")

	for _, period := range []string{"", TransactionPeriodHour} {
		transactions, err := e.GetTransactions(t.Context(), spotPair, period)
		require.NoErrorf(t, err, "GetTransactions must not error for period %q", period)
		require.NotEmptyf(t, transactions, "GetTransactions must return transactions for period %q", period)
		if mockTests {
			exp := TransactionResponse{
				Date:    types.Time(time.Unix(1790317257, 0)),
				TradeID: 642950935,
				Price:   84246.67,
				Side:    OrderSide(order.Sell),
				Amount:  0.05934952,
			}
			assert.Equalf(t, exp, transactions[0], "GetTransactions should decode the latest transaction for period %q", period)
		}
		for _, tr := range transactions {
			assert.NotZero(t, tr.Date.Time(), "Date should be set")
			assert.NotZero(t, tr.TradeID, "TradeID should be set")
			assert.Positive(t, tr.Price.Float64(), "Price should be positive")
			assert.Positive(t, tr.Amount.Float64(), "Amount should be positive")
			assert.Contains(t, []order.Side{order.Buy, order.Sell}, tr.Side.Side(), "Side should be valid")
		}
	}
}

func TestGetMarkets(t *testing.T) {
	t.Parallel()
	for _, entity := range []string{"", "EUROPE_SA"} {
		markets, err := e.GetMarkets(t.Context(), entity)
		require.NoErrorf(t, err, "GetMarkets must not error for entity %q", entity)
		if mockTests {
			i := slices.IndexFunc(markets, func(m MarketResponse) bool { return m.Name == "BTC/USD" })
			require.NotEqualf(t, -1, i, "GetMarkets must list BTC/USD for entity %q", entity)
			exp := MarketResponse{
				Name:                          "BTC/USD",
				MarketSymbol:                  "btcusd",
				BaseCurrency:                  currency.BTC,
				BaseDecimals:                  8,
				CounterCurrency:               currency.USD,
				CounterDecimals:               2,
				MinimumOrderValue:             10,
				TradingEnabled:                true,
				InstantOrderCounterDecimals:   2,
				InstantAndMarketOrdersEnabled: true,
				Description:                   "Bitcoin / U.S. dollar",
				MarketType:                    MarketTypeSpot,
			}
			assert.Equalf(t, exp, markets[i], "GetMarkets should decode a spot market for entity %q", entity)
		}
		marketTypes := make(map[string]bool)
		for i := range markets {
			m := &markets[i]
			marketTypes[m.MarketType] = true
			assert.NotEmpty(t, m.Name, "Name should be set")
			assert.NotEmpty(t, m.MarketSymbol, "MarketSymbol should be set")
			assert.False(t, m.BaseCurrency.IsEmpty(), "BaseCurrency should be set")
			assert.False(t, m.CounterCurrency.IsEmpty(), "CounterCurrency should be set")
			assert.Positive(t, m.MinimumOrderValue.Float64(), "MinimumOrderValue should be positive")
			if m.MarketType == MarketTypePerpetual {
				assert.Positive(t, m.MaxLeverage.Float64(), "MaxLeverage should be positive")
				assert.Positive(t, m.ContractSize.Float64(), "ContractSize should be positive")
				assert.Equal(t, "Linear", m.PayoffType, "PayoffType should be correct")
			}
		}
		assert.Truef(t, marketTypes[MarketTypeSpot], "GetMarkets should return spot markets for entity %q", entity)
		if entity != "" {
			continue
		}
		assert.True(t, marketTypes[MarketTypePerpetual], "GetMarkets should return perpetual markets")
		if mockTests {
			i := slices.IndexFunc(markets, func(m MarketResponse) bool { return m.Name == "BTC/USD-PERP" })
			require.NotEqual(t, -1, i, "GetMarkets must list BTC/USD-PERP")
			exp := MarketResponse{
				Name:                          "BTC/USD-PERP",
				MarketSymbol:                  "btcusd-perp",
				BaseCurrency:                  currency.BTC,
				BaseDecimals:                  5,
				CounterCurrency:               currency.USD,
				MinimumOrderValue:             10,
				MaximumOrderValue:             1000000,
				MinimumOrderAmount:            0.00001,
				MaximumOrderAmount:            20,
				TradingEnabled:                true,
				InstantOrderCounterDecimals:   2,
				InstantAndMarketOrdersEnabled: true,
				Description:                   "Bitcoin / U.S. dollar Perpetual",
				MarketType:                    MarketTypePerpetual,
				UnderlyingAsset:               "Kaiko BTC Benchmark Reference Rate",
				PayoffType:                    "Linear",
				ContractSize:                  1,
				MaxLeverage:                   10,
				ISIN:                          "EZHKD4DNKHY3",
				AssetClass:                    AssetClassCrypto,
			}
			assert.Equal(t, exp, markets[i], "GetMarkets should decode a perpetual market")
		}
	}
}

func TestGetOHLC(t *testing.T) {
	t.Parallel()
	start := time.Unix(1735689600, 0)
	end := time.Unix(1735693200, 0)
	for _, tc := range []struct {
		req *OHLCRequest
		err error
	}{
		{req: nil, err: common.ErrNilPointer},
		{req: &OHLCRequest{}, err: currency.ErrCurrencyPairEmpty},
		{req: &OHLCRequest{Pair: spotPair}, err: errInvalidInterval},
		{req: &OHLCRequest{Pair: spotPair, Step: kline.OneMin}, err: errInvalidLimit},
		{req: &OHLCRequest{Pair: spotPair, Step: kline.OneMin, Limit: 1001}, err: errInvalidLimit},
		{req: &OHLCRequest{Pair: spotPair, Step: kline.OneMin, Limit: 10, Start: end, End: start}, err: common.ErrStartAfterEnd},
	} {
		_, err := e.GetOHLC(t.Context(), tc.req)
		assert.ErrorIs(t, err, tc.err, "GetOHLC should return the correct error")
	}

	for _, exclude := range []bool{false, true} {
		resp, err := e.GetOHLC(t.Context(), &OHLCRequest{Pair: spotPair, Step: kline.OneMin, Limit: 10, Start: start, End: end, ExcludeCurrentCandle: exclude})
		require.NoError(t, err, "GetOHLC must not error")
		assert.Equal(t, "BTC/USD", resp.Data.Pair, "Pair should be correct")
		require.Len(t, resp.Data.OHLC, 10, "GetOHLC must return the requested candles")
		if mockTests {
			exp := Candle{
				Timestamp: types.Time(time.Unix(1735692660, 0)),
				Open:      93888,
				High:      93917,
				Low:       93885,
				Close:     93917,
				Volume:    0.98096975,
			}
			assert.Equal(t, exp, resp.Data.OHLC[0], "GetOHLC should decode the first candle")
		}
		for _, c := range resp.Data.OHLC {
			assert.False(t, c.Timestamp.Time().Before(start), "Timestamp should not be before the start")
			assert.Positive(t, c.Open.Float64(), "Open should be positive")
			assert.Positive(t, c.High.Float64(), "High should be positive")
			assert.Positive(t, c.Low.Float64(), "Low should be positive")
			assert.Positive(t, c.Close.Float64(), "Close should be positive")
		}
	}
}

func TestGetEURUSDConversionRate(t *testing.T) {
	t.Parallel()
	rate, err := e.GetEURUSDConversionRate(t.Context())
	require.NoError(t, err, "GetEURUSDConversionRate must not error")
	if mockTests {
		assert.Equal(t, &EURUSDConversionRateResponse{Buy: 1.17311222, Sell: 1.099995}, rate, "GetEURUSDConversionRate should decode the rates")
		return
	}
	assert.Positive(t, rate.Buy.Float64(), "Buy should be positive")
	assert.Positive(t, rate.Sell.Float64(), "Sell should be positive")
}

func TestGetFundingRate(t *testing.T) {
	t.Parallel()
	_, err := e.GetFundingRate(t.Context(), currency.EMPTYPAIR)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetFundingRate should error on an empty pair")

	rate, err := e.GetFundingRate(t.Context(), perpetualPair)
	require.NoError(t, err, "GetFundingRate must not error")
	if mockTests {
		exp := &FundingRateResponse{
			FundingRate:     0.000002,
			Timestamp:       types.Time(time.Unix(1790317287, 0)),
			Market:          "BTC/USD-PERP",
			NextFundingTime: types.Time(time.Unix(1790323200, 0)),
		}
		assert.Equal(t, exp, rate, "GetFundingRate should decode the funding rate")
		return
	}
	assert.Equal(t, "BTC/USD-PERP", rate.Market, "Market should be correct")
	assert.NotZero(t, rate.Timestamp.Time(), "Timestamp should be set")
	assert.True(t, rate.NextFundingTime.Time().After(rate.Timestamp.Time()), "NextFundingTime should be after Timestamp")
}

func TestGetFundingRateHistory(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		req *FundingRateHistoryRequest
		err error
	}{
		{req: nil, err: common.ErrNilPointer},
		{req: &FundingRateHistoryRequest{}, err: currency.ErrCurrencyPairEmpty},
		{req: &FundingRateHistoryRequest{Pair: perpetualPair, Limit: 101}, err: errInvalidLimit},
		{req: &FundingRateHistoryRequest{Pair: perpetualPair, Since: time.Unix(2, 0), Until: time.Unix(1, 0)}, err: common.ErrStartAfterEnd},
	} {
		_, err := e.GetFundingRateHistory(t.Context(), tc.req)
		assert.ErrorIs(t, err, tc.err, "GetFundingRateHistory should return the correct error")
	}

	history, err := e.GetFundingRateHistory(t.Context(), &FundingRateHistoryRequest{Pair: perpetualPair, Limit: 3})
	require.NoError(t, err, "GetFundingRateHistory must not error")
	if mockTests {
		exp := &FundingRateHistoryResponse{
			Market: "BTC/USD-PERP",
			FundingRateHistory: []FundingRate{
				{FundingRate: 0.000019, Timestamp: types.Time(time.Unix(1790236800, 0))},
				{Timestamp: types.Time(time.Unix(1790265600, 0))},
				{Timestamp: types.Time(time.Unix(1790294400, 0))},
			},
		}
		assert.Equal(t, exp, history, "GetFundingRateHistory should decode the rates")
		return
	}
	assert.Equal(t, "BTC/USD-PERP", history.Market, "Market should be correct")
	require.Len(t, history.FundingRateHistory, 3, "GetFundingRateHistory must return the requested rates")
	for i := 1; i < len(history.FundingRateHistory); i++ {
		assert.True(t, history.FundingRateHistory[i].Timestamp.Time().After(history.FundingRateHistory[i-1].Timestamp.Time()), "rates should be in ascending order")
	}
}

func TestGetVirtualAssetServiceProviders(t *testing.T) {
	t.Parallel()
	_, err := e.GetVirtualAssetServiceProviders(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer, "GetVirtualAssetServiceProviders should error on a nil request")
	providers, err := e.GetVirtualAssetServiceProviders(t.Context(), &PaginationRequest{Page: 1, PerPage: 3})
	require.NoError(t, err, "GetVirtualAssetServiceProviders must not error")
	if mockTests {
		exp := &VirtualAssetServiceProvidersResponse{
			Data: []VirtualAssetServiceProvider{
				{UUID: "48feffe5-a5d0-44fe-bf4c-734e7d721249", Name: "Payco"},
				{UUID: "3b21f525-d47c-4514-b575-c3c0eb0846a4", Name: "V-Bank AG (DE)"},
				{UUID: "0a0ab6c6-3e85-4cb5-b182-ddc7adc3bc19", Name: "Bitpace"},
			},
			Pagination: Pagination{Page: 1, Size: 3, Count: 1204},
		}
		assert.Equal(t, exp, providers, "GetVirtualAssetServiceProviders should decode the page of providers")
		return
	}
	require.NotEmpty(t, providers.Data, "GetVirtualAssetServiceProviders must return providers")
	assert.NotEmpty(t, providers.Data[0].UUID, "UUID should be set")
	assert.NotEmpty(t, providers.Data[0].Name, "Name should be set")
}
