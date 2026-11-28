package binanceus

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestWebsocketAPIGetMostRecentTrades(t *testing.T) {
	t.Parallel()
	ex := newWebsocketAPITestExchange(t, false)
	result, err := ex.WebsocketAPI().GetMostRecentTrades(t.Context(), &RecentTradeRequest{Symbol: currency.NewBTCUSDT(), Limit: 3})
	require.NoError(t, err, "GetMostRecentTrades must not error")
	if !mockTests {
		assert.Len(t, result, 3, "GetMostRecentTrades should return the limit of trades")
		return
	}
	exp := []RecentTradeResponse{
		{ID: 32184163, Price: 73441.84, Quantity: 0.00019, QuoteQuantity: 13.9539496, Time: types.Time(time.UnixMilli(1794478307154)), IsBuyerMaker: true, IsBestMatch: true},
		{ID: 32184164, Price: 73431.17, Quantity: 0.0002, QuoteQuantity: 14.686234, Time: types.Time(time.UnixMilli(1794478346726)), IsBestMatch: true},
		{ID: 32184165, Price: 73438.24, Quantity: 0.00019, QuoteQuantity: 13.9532656, Time: types.Time(time.UnixMilli(1794478351810)), IsBuyerMaker: true, IsBestMatch: true},
	}
	assert.Equal(t, exp, result, "GetMostRecentTrades should decode every field")
}

func TestWebsocketAPIGetHistoricalTrades(t *testing.T) {
	t.Parallel()
	ex := newWebsocketAPITestExchange(t, false)
	result, err := ex.WebsocketAPI().GetHistoricalTrades(t.Context(), currency.NewBTCUSDT(), 3, 32183707)
	require.NoError(t, err, "GetHistoricalTrades must not error")
	exp := []RecentTradeResponse{
		{ID: 32183707, Price: 73687.07, Quantity: 0.00042, QuoteQuantity: 30.9485694, Time: types.Time(time.UnixMilli(1794464066712)), IsBuyerMaker: true, IsBestMatch: true},
		{ID: 32183708, Price: 73683.77, Quantity: 0.00098, QuoteQuantity: 72.2100946, Time: types.Time(time.UnixMilli(1794464409788)), IsBuyerMaker: true, IsBestMatch: true},
		{ID: 32183709, Price: 73716.55, Quantity: 0.00198, QuoteQuantity: 145.958769, Time: types.Time(time.UnixMilli(1794464635319)), IsBestMatch: true},
	}
	assert.Equal(t, exp, result, "GetHistoricalTrades should return the trades from the starting trade ID")
}

func TestWebsocketAPIGetAggregatedTrades(t *testing.T) {
	t.Parallel()
	ex := newWebsocketAPITestExchange(t, false)
	result, err := ex.WebsocketAPI().GetAggregatedTrades(t.Context(), &AggregatedTradeRequest{Symbol: currency.NewBTCUSDT(), FromID: 30522835, Limit: 3})
	require.NoError(t, err, "GetAggregatedTrades must not error")
	exp := []AggregatedTradeResponse{
		{AggregateTradeID: 30522835, Price: 73687.07, Quantity: 0.00075, FirstTradeID: 32183706, LastTradeID: 32183707, Timestamp: types.Time(time.UnixMilli(1794464066712)), IsBuyerMaker: true, IsBestMatch: true},
		{AggregateTradeID: 30522836, Price: 73683.77, Quantity: 0.00098, FirstTradeID: 32183708, LastTradeID: 32183708, Timestamp: types.Time(time.UnixMilli(1794464409788)), IsBuyerMaker: true, IsBestMatch: true},
		{AggregateTradeID: 30522837, Price: 73716.55, Quantity: 0.00198, FirstTradeID: 32183709, LastTradeID: 32183709, Timestamp: types.Time(time.UnixMilli(1794464635319)), IsBestMatch: true},
	}
	assert.Equal(t, exp, result, "GetAggregatedTrades should return the aggregate trades from the starting ID")
}

func TestWebsocketAPIGetOrderBook(t *testing.T) {
	t.Parallel()
	ex := newWebsocketAPITestExchange(t, false)
	result, err := ex.WebsocketAPI().GetOrderBook(t.Context(), &OrderBookRequest{Symbol: currency.NewBTCUSDT(), Limit: 5})
	require.NoError(t, err, "GetOrderBook must not error")
	require.NotNil(t, result, "GetOrderBook must return a result")
	if !mockTests {
		assert.Len(t, result.Bids, 5, "GetOrderBook should return the limit of bids")
		assert.Len(t, result.Asks, 5, "GetOrderBook should return the limit of asks")
		return
	}
	exp := &OrderBookResponse{
		LastUpdateID: 3418063583,
		Bids:         orderbook.LevelsArrayPriceAmount{{Price: 73415.8, Amount: 0.00034}, {Price: 73415.72, Amount: 0.00395}, {Price: 73415.69, Amount: 0.00198}, {Price: 73412.3, Amount: 0.02726}, {Price: 73409.32, Amount: 0.01432}},
		Asks:         orderbook.LevelsArrayPriceAmount{{Price: 73416, Amount: 0.00068}, {Price: 73416.01, Amount: 0.00593}, {Price: 73416.55, Amount: 0.00034}, {Price: 73417.81, Amount: 0.05814}, {Price: 73417.85, Amount: 0.08977}},
	}
	assert.Equal(t, exp, result, "GetOrderBook should decode every field")
}

func TestWebsocketAPIGetSpotKline(t *testing.T) {
	t.Parallel()
	t.Run("draws on the weight Binance.US charges", func(t *testing.T) {
		t.Parallel()
		ex, _, _ := newTwoServerTestExchange(t, websocketAPIMock, false)
		conn := newFakeConnection()
		conn.response = []byte(`{"status":200,"result":[]}`)
		ex.wsAPI.set(conn, time.Second)
		_, err := ex.WebsocketAPI().GetSpotKline(t.Context(), &KlinesRequest{Symbol: currency.NewBTCUSDT(), Interval: kline.OneHour})
		require.NoError(t, err, "GetSpotKline must not error")
		require.Len(t, conn.limits, 1, "GetSpotKline must send one request")
		assert.Equal(t, request.Weight(2), endpointCosts[conn.limits[0]].weight, "GetSpotKline should draw on the weight Binance.US charges for klines")
	})

	ex := newWebsocketAPITestExchange(t, false)
	result, err := ex.WebsocketAPI().GetSpotKline(t.Context(), &KlinesRequest{Symbol: currency.NewBTCUSDT(), Interval: kline.OneHour, StartTime: time.UnixMilli(1794459600000), EndTime: time.UnixMilli(1794466799999), Limit: 2})
	require.NoError(t, err, "GetSpotKline must not error")
	exp := []CandleStickResponse{
		{OpenTime: types.Time(time.UnixMilli(1794459600000)), OpenPrice: 73729.46, HighPrice: 73729.46, LowPrice: 73538.15, ClosePrice: 73583.11, Volume: 0.20483, CloseTime: types.Time(time.UnixMilli(1794463199999)), QuoteAssetVolume: 15076.9394201, NumberOfTrades: 132, TakerBuyBaseAssetVolume: 0.10514, TakerBuyQuoteAssetVolume: 7737.6648209},
		{OpenTime: types.Time(time.UnixMilli(1794463200000)), OpenPrice: 73598.16, HighPrice: 73716.55, LowPrice: 73437.5, ClosePrice: 73515.52, Volume: 0.27589, CloseTime: types.Time(time.UnixMilli(1794466799999)), QuoteAssetVolume: 20308.0265795, NumberOfTrades: 94, TakerBuyBaseAssetVolume: 0.05999, TakerBuyQuoteAssetVolume: 4414.2071048},
	}
	assert.Equal(t, exp, result, "GetSpotKline should return the klines of the past window")
}

func TestWebsocketAPIGetLatestSpotPrice(t *testing.T) {
	t.Parallel()
	ex := newWebsocketAPITestExchange(t, false)
	for _, tc := range []struct {
		name    string
		symbols currency.Pairs
		exp     []SymbolPriceResponse
	}{
		{name: "one symbol", symbols: currency.Pairs{currency.NewBTCUSDT()}, exp: []SymbolPriceResponse{{Symbol: "BTCUSDT", Price: 73438.24}}},
		{name: "several symbols", symbols: currency.Pairs{currency.NewBTCUSDT(), currency.NewPair(currency.ETH, currency.USDT)}, exp: []SymbolPriceResponse{{Symbol: "BTCUSDT", Price: 73438.24}, {Symbol: "ETHUSDT", Price: 2043}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := ex.WebsocketAPI().GetLatestSpotPrice(t.Context(), tc.symbols, "")
			require.NoError(t, err, "GetLatestSpotPrice must not error")
			if !mockTests {
				assert.Len(t, result, len(tc.symbols), "GetLatestSpotPrice should return each symbol's price")
				return
			}
			assert.Equal(t, tc.exp, result, "GetLatestSpotPrice should decode every symbol's price")
		})
	}
}

func TestWebsocketAPIGetAveragePrice(t *testing.T) {
	t.Parallel()
	t.Run("draws on the weight Binance.US charges", func(t *testing.T) {
		t.Parallel()
		ex, _, _ := newTwoServerTestExchange(t, websocketAPIMock, false)
		conn := newFakeConnection()
		conn.response = []byte(`{"status":200,"result":{"mins":5,"price":"1","closeTime":1794478351810}}`)
		ex.wsAPI.set(conn, time.Second)
		_, err := ex.WebsocketAPI().GetAveragePrice(t.Context(), currency.NewBTCUSDT())
		require.NoError(t, err, "GetAveragePrice must not error")
		require.Len(t, conn.limits, 1, "GetAveragePrice must send one request")
		assert.Equal(t, request.Weight(2), endpointCosts[conn.limits[0]].weight, "GetAveragePrice should draw on the weight Binance.US charges for the average price")
	})

	ex := newWebsocketAPITestExchange(t, false)
	result, err := ex.WebsocketAPI().GetAveragePrice(t.Context(), currency.NewBTCUSDT())
	require.NoError(t, err, "GetAveragePrice must not error")
	require.NotNil(t, result, "GetAveragePrice must return a result")
	if !mockTests {
		assert.Positive(t, result.Price.Float64(), "GetAveragePrice should return a price")
		return
	}
	assert.Equal(t, &AveragePriceResponse{IntervalInMinutes: 5, Price: 73404.64708945, CloseTime: types.Time(time.UnixMilli(1794478351810))}, result, "GetAveragePrice should decode every field")
}

func TestWebsocketAPIGetBestPrice(t *testing.T) {
	t.Parallel()
	ex := newWebsocketAPITestExchange(t, false)
	result, err := ex.WebsocketAPI().GetBestPrice(t.Context(), currency.Pairs{currency.NewBTCUSDT()}, "")
	require.NoError(t, err, "GetBestPrice must not error")
	if !mockTests {
		assert.Len(t, result, 1, "GetBestPrice should return the symbol's book ticker")
		return
	}
	assert.Equal(t, []BookTickerResponse{{Symbol: "BTCUSDT", BidPrice: 73397.8, BidQuantity: 0.00034, AskPrice: 73398.01, AskQuantity: 0.00592}}, result, "GetBestPrice should decode every field")
}

func TestWebsocketAPIGetPriceChangeStatistics(t *testing.T) {
	t.Parallel()
	ex := newWebsocketAPITestExchange(t, false)
	result, err := ex.WebsocketAPI().GetPriceChangeStatistics(t.Context(), &PriceChangeStatisticsRequest{Symbols: currency.Pairs{currency.NewBTCUSDT()}})
	require.NoError(t, err, "GetPriceChangeStatistics must not error")
	if !mockTests {
		assert.Len(t, result, 1, "GetPriceChangeStatistics should return the symbol's statistics")
		return
	}
	exp := []PriceChangeStatisticsResponse{{
		Symbol:               "BTCUSDT",
		PriceChange:          -280.54,
		PriceChangePercent:   -0.381,
		WeightedAveragePrice: 73980.8105783,
		PreviousClosePrice:   73726,
		LastPrice:            73438.24,
		LastQuantity:         0.00019,
		BidPrice:             73397.8,
		BidQuantity:          0.00034,
		AskPrice:             73398.01,
		AskQuantity:          0.00592,
		OpenPrice:            73718.78,
		HighPrice:            74797.33,
		LowPrice:             73266.01,
		Volume:               50.80216,
		QuoteVolume:          3758384.9759283,
		OpenTime:             types.Time(time.UnixMilli(1794391968291)),
		CloseTime:            types.Time(time.UnixMilli(1794478368291)),
		FirstID:              32176075,
		LastID:               32184165,
		Count:                8091,
	}}
	assert.Equal(t, exp, result, "GetPriceChangeStatistics should decode every field")
}

func TestWebsocketAPIGetTickerData(t *testing.T) {
	t.Parallel()
	ex := newWebsocketAPITestExchange(t, false)
	result, err := ex.WebsocketAPI().GetTickerData(t.Context(), &RollingWindowTickerRequest{Symbols: currency.Pairs{currency.NewBTCUSDT()}, WindowSize: time.Hour})
	require.NoError(t, err, "GetTickerData must not error")
	if !mockTests {
		assert.Len(t, result, 1, "GetTickerData should return the symbol's statistics")
		return
	}
	exp := []TickerStatisticsResponse{{
		Symbol:               "BTCUSDT",
		PriceChange:          9.19,
		PriceChangePercent:   0.013,
		WeightedAveragePrice: 73408.01213856,
		OpenPrice:            73429.05,
		HighPrice:            73528.06,
		LowPrice:             73266.01,
		LastPrice:            73438.24,
		Volume:               0.306,
		QuoteVolume:          22462.8517144,
		OpenTime:             types.Time(time.UnixMilli(1794474720000)),
		CloseTime:            types.Time(time.UnixMilli(1794478370948)),
		FirstID:              32183924,
		LastID:               32184165,
		Count:                242,
	}}
	assert.Equal(t, exp, result, "GetTickerData should decode every field")
}
