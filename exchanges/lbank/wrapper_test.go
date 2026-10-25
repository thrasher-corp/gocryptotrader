package lbank

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/deposit"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// newTestExchangeConfig returns a copy of LBank's test configuration
func newTestExchangeConfig(t *testing.T) *config.Exchange {
	t.Helper()
	raw, err := testExchangeConfig()
	require.NoError(t, err, "testExchangeConfig must not error")
	exchCfg := new(config.Exchange)
	require.NoError(t, json.Unmarshal(raw, exchCfg), "Unmarshal must decode the test configuration")
	return exchCfg
}

func TestSetup(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		secret string
		exp    bool
	}{
		{name: "HmacSHA256 secret", secret: "093F44F700FC48F17DDB67390C895CE5", exp: true},
		{name: "unparsable RSA secret", secret: "MII!!not a key", exp: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			exch := newTestExchangeConfig(t)
			exch.API.AuthenticatedSupport = true
			exch.API.AuthenticatedWebsocketSupport = true
			exch.API.Credentials.Key = testAPIKey
			exch.API.Credentials.Secret = tc.secret
			ex := new(Exchange)
			ex.SetDefaults()
			ex.Websocket = sharedtestvalues.NewTestWebsocket()
			require.NoError(t, ex.Setup(exch), "Setup must not error")
			assert.Equal(t, tc.exp, ex.API.AuthenticatedSupport, "Setup should keep authentication only for a secret it can sign with")
			assert.Equal(t, tc.exp, ex.API.AuthenticatedWebsocketSupport, "Setup should keep websocket authentication only for a secret it can sign with")
		})
	}

	exch := newTestExchangeConfig(t)
	exch.Enabled = false
	ex := new(Exchange)
	ex.SetDefaults()
	require.NoError(t, ex.Setup(exch), "Setup must not error for a disabled exchange")
	assert.False(t, ex.IsEnabled(), "Setup should disable the exchange")
}

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	_, err := e.FetchTradablePairs(t.Context(), asset.Margin)
	require.ErrorIs(t, err, asset.ErrNotSupported, "FetchTradablePairs must reject an asset LBank doesn't trade")

	spot, err := e.FetchTradablePairs(t.Context(), asset.Spot)
	require.NoError(t, err, "FetchTradablePairs must not error for spot")
	contracts, err := e.FetchTradablePairs(t.Context(), asset.USDTMarginedFutures)
	require.NoError(t, err, "FetchTradablePairs must not error for contracts")
	if mockTests {
		exp := currency.Pairs{
			currency.NewPairWithDelimiter("btc", "usdt", currency.UnderscoreDelimiter),
			currency.NewPairWithDelimiter("eth", "usdt", currency.UnderscoreDelimiter),
			currency.NewPairWithDelimiter("btc3l", "usdt", currency.UnderscoreDelimiter),
		}
		assert.Equal(t, exp, spot, "FetchTradablePairs should return every spot pair")
		exp = currency.Pairs{currency.NewPair(currency.NewCode("BTC"), currency.NewCode("USDT")), currency.NewPair(currency.NewCode("ETH"), currency.NewCode("USDT"))}
		assert.Equal(t, exp, contracts, "FetchTradablePairs should return every contract's pair")
		return
	}
	assert.True(t, spot.Contains(currency.NewBTCUSDT(), false), "FetchTradablePairs should return BTC-USDT spot")
	assert.True(t, contracts.Contains(currency.NewBTCUSDT(), false), "FetchTradablePairs should return the BTC-USDT contract")
}

func TestUpdateTradablePairs(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	require.NoError(t, ex.UpdateTradablePairs(t.Context()), "UpdateTradablePairs must not error")
	spot, err := ex.GetAvailablePairs(asset.Spot)
	require.NoError(t, err, "GetAvailablePairs must not error for spot")
	contracts, err := ex.GetAvailablePairs(asset.USDTMarginedFutures)
	require.NoError(t, err, "GetAvailablePairs must not error for contracts")
	if mockTests {
		assert.Len(t, spot, 3, "UpdateTradablePairs should store every spot pair")
		assert.Len(t, contracts, 2, "UpdateTradablePairs should store every contract's pair")
	}
	assert.True(t, spot.Contains(spotTestPair, true), "UpdateTradablePairs should store BTC-USDT spot")
	assert.True(t, contracts.Contains(futuresTestPair, true), "UpdateTradablePairs should store the BTC-USDT contract")
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	require.ErrorIs(t, ex.UpdateTickers(t.Context(), asset.Margin), asset.ErrNotSupported, "UpdateTickers must reject an asset LBank doesn't trade")

	for _, a := range ex.GetAssetTypes(true) {
		require.NoErrorf(t, ex.UpdateTickers(t.Context(), a), "UpdateTickers must not error for %s", a)
	}
	spot, err := ticker.GetTicker(ex.Name, spotTestPair, asset.Spot)
	require.NoError(t, err, "UpdateTickers must store the spot ticker")
	contract, err := ticker.GetTicker(ex.Name, futuresTestPair, asset.USDTMarginedFutures)
	require.NoError(t, err, "UpdateTickers must store the contract ticker")
	if !mockTests {
		assert.Positive(t, spot.Last, "UpdateTickers should store a spot price")
		assert.Positive(t, contract.Last, "UpdateTickers should store a contract price")
		return
	}
	exp := &ticker.Price{
		ExchangeName: ex.Name, Pair: spot.Pair, AssetType: asset.Spot,
		Last: 80796.78, High: 81092.52, Low: 79270.99, BaseVolume: 2863.0684, QuoteVolume: 230249026.95, PercentChange24Hour: 0.04,
		LastUpdated: time.UnixMilli(1792890784417),
	}
	assert.Equal(t, exp, spot, "UpdateTickers should store the spot ticker")
	assert.True(t, spot.Pair.Equal(spotTestPair), "UpdateTickers should store the enabled spot pair's ticker")
	exp = &ticker.Price{
		ExchangeName: ex.Name, Pair: contract.Pair, AssetType: asset.USDTMarginedFutures,
		Last: 80766, High: 81073.2, Low: 79195.9, Open24Hour: 80710, BaseVolume: 3718.8446, QuoteVolume: 298925536.6276579,
		MarkPrice: 80766, IndexPrice: 80796.6, LastUpdated: time.Unix(1792890789, 0),
	}
	assert.Equal(t, exp, contract, "UpdateTickers should store the contract ticker")
	assert.True(t, contract.Pair.Equal(futuresTestPair), "UpdateTickers should store the enabled contract's ticker")
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	_, err := ex.UpdateTicker(t.Context(), currency.EMPTYPAIR, asset.Spot)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "UpdateTicker must require a pair")
	_, err = ex.UpdateTicker(t.Context(), spotTestPair, asset.Margin)
	require.ErrorIs(t, err, asset.ErrNotSupported, "UpdateTicker must reject an asset LBank doesn't trade")

	result, err := ex.UpdateTicker(t.Context(), spotTestPair, asset.Spot)
	require.NoError(t, err, "UpdateTicker must not error for spot")
	if mockTests {
		exp := &ticker.Price{
			ExchangeName: ex.Name, Pair: spotTestPair, AssetType: asset.Spot,
			Last: 80796.78, High: 81092.52, Low: 79270.99, BaseVolume: 2863.0685, QuoteVolume: 230249030.18, PercentChange24Hour: 0.04,
			LastUpdated: time.UnixMilli(1792890786820),
		}
		assert.Equal(t, exp, result, "UpdateTicker should return the spot ticker")
	} else {
		assert.Positive(t, result.Last, "UpdateTicker should return a spot price")
	}

	// The standard ticker endpoint refuses a leveraged token, which UpdateTicker then asks its own endpoint for
	leveraged := currency.NewPairWithDelimiter("btc3l", "usdt", currency.UnderscoreDelimiter)
	result, err = ex.UpdateTicker(t.Context(), leveraged, asset.Spot)
	require.NoError(t, err, "UpdateTicker must not error for a leveraged token")
	if mockTests {
		exp := &ticker.Price{
			ExchangeName: ex.Name, Pair: leveraged, AssetType: asset.Spot,
			Last: 2.678543, High: 2.721474, Low: 2.536929, BaseVolume: 112529.32, QuoteVolume: 300017.7961, PercentChange24Hour: 0.03,
			LastUpdated: time.UnixMilli(1792890787880),
		}
		assert.Equal(t, exp, result, "UpdateTicker should return the leveraged token's ticker")
	} else {
		assert.Positive(t, result.Last, "UpdateTicker should return the leveraged token's price")
	}

	result, err = ex.UpdateTicker(t.Context(), futuresTestPair, asset.USDTMarginedFutures)
	require.NoError(t, err, "UpdateTicker must not error for a contract")
	if mockTests {
		exp := &ticker.Price{
			ExchangeName: ex.Name, Pair: futuresTestPair, AssetType: asset.USDTMarginedFutures,
			Last: 80766, High: 81073.2, Low: 79195.9, Open24Hour: 80710, BaseVolume: 3718.8446, QuoteVolume: 298925536.6276579,
			MarkPrice: 80766, IndexPrice: 80796.6, LastUpdated: time.Unix(1792890789, 0),
		}
		assert.Equal(t, exp, result, "UpdateTicker should return the contract ticker")
	} else {
		assert.Positive(t, result.Last, "UpdateTicker should return a contract price")
	}

	_, err = ex.UpdateTicker(t.Context(), currency.NewPair(currency.NewCode("NOPE"), currency.USDT), asset.USDTMarginedFutures)
	assert.ErrorIs(t, err, ticker.ErrTickerNotFound, "UpdateTicker should report a contract LBank doesn't list")
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	_, err := ex.UpdateOrderbook(t.Context(), currency.EMPTYPAIR, asset.Spot)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "UpdateOrderbook must require a pair")
	_, err = ex.UpdateOrderbook(t.Context(), spotTestPair, asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported, "UpdateOrderbook must reject an asset LBank doesn't trade")

	result, err := ex.UpdateOrderbook(t.Context(), spotTestPair, asset.Spot)
	require.NoError(t, err, "UpdateOrderbook must not error for spot")
	if mockTests {
		require.Len(t, result.Asks, orderbookDepth, "UpdateOrderbook must store every ask level fetched")
		require.Len(t, result.Bids, orderbookDepth, "UpdateOrderbook must store every bid level fetched")
		require.NotZero(t, result.InsertedAt, "UpdateOrderbook must record when the book was stored")
		result.Asks, result.Bids, result.InsertedAt = result.Asks[:1], result.Bids[:1], time.Time{}
		exp := &orderbook.Book{
			Exchange: ex.Name, Pair: spotTestPair, Asset: asset.Spot,
			Asks:              orderbook.Levels{{Price: 80966.61, Amount: 2.50266}},
			Bids:              orderbook.Levels{{Price: 80966.6, Amount: 6.19313}},
			LastUpdated:       time.UnixMilli(1792897789848),
			ValidateOrderbook: true,
			RestSnapshot:      true,
		}
		assert.Equal(t, exp, result, "UpdateOrderbook should store the spot book")
	} else {
		assert.NotEmpty(t, result.Asks, "UpdateOrderbook should store asks")
	}

	result, err = ex.UpdateOrderbook(t.Context(), futuresTestPair, asset.USDTMarginedFutures)
	require.NoError(t, err, "UpdateOrderbook must not error for a contract")
	require.NotZero(t, result.LastUpdated, "UpdateOrderbook must date the contract book")
	require.NotZero(t, result.InsertedAt, "UpdateOrderbook must record when the book was stored")
	if !mockTests {
		assert.NotEmpty(t, result.Asks, "UpdateOrderbook should store asks")
		return
	}
	require.Len(t, result.Asks, orderbookDepth, "UpdateOrderbook must store every contract ask level fetched")
	require.Len(t, result.Bids, orderbookDepth, "UpdateOrderbook must store every contract bid level fetched")
	result.Asks, result.Bids, result.LastUpdated, result.InsertedAt = result.Asks[:1], result.Bids[:1], time.Time{}, time.Time{}
	exp := &orderbook.Book{
		Exchange: ex.Name, Pair: futuresTestPair, Asset: asset.USDTMarginedFutures,
		Asks:              orderbook.Levels{{Price: 80940, Amount: 10.3864, OrderCount: 1}},
		Bids:              orderbook.Levels{{Price: 80939.9, Amount: 14.7768, OrderCount: 2}},
		ValidateOrderbook: true,
		RestSnapshot:      true,
	}
	assert.Equal(t, exp, result, "UpdateOrderbook should store the contract book")
}

func TestUpdateAccountBalances(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateAccountBalances(t.Context(), asset.USDTMarginedFutures)
	require.ErrorIs(t, err, asset.ErrNotSupported, "UpdateAccountBalances must reject an asset LBank has no balances for")

	skipLiveWithoutCredentials(t)
	result, err := e.UpdateAccountBalances(t.Context(), asset.Spot)
	require.NoError(t, err, "UpdateAccountBalances must not error")
	if !mockTests {
		assert.NotEmpty(t, result, "UpdateAccountBalances should return the account's balances")
		return
	}
	require.Len(t, result, 1, "UpdateAccountBalances must return the spot account")
	for c, b := range result[0].Balances {
		require.NotZerof(t, b.UpdatedAt, "UpdateAccountBalances must date the %s balance", c)
		b.UpdatedAt = time.Time{}
		result[0].Balances[c] = b
	}
	usdt, btc, eth := currency.NewCode("usdt"), currency.NewCode("btc"), currency.NewCode("eth")
	exp := accounts.SubAccounts{{
		AssetType: asset.Spot,
		Balances: accounts.CurrencyBalances{
			usdt: {Currency: usdt, Total: 1000, Free: 1000},
			btc:  {Currency: btc, Total: 0.012, Hold: 0.002, Free: 0.01},
			eth:  {Currency: eth, Total: 0.1, Free: 0.1},
		},
	}}
	assert.Equal(t, exp, result, "UpdateAccountBalances should return every balance")
}

func TestGetAccountFundingHistory(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	result, err := e.GetAccountFundingHistory(t.Context())
	require.NoError(t, err, "GetAccountFundingHistory must not error")
	if !mockTests {
		assert.NotNil(t, result, "GetAccountFundingHistory should return the account's history")
		return
	}
	exp := []exchange.FundingHistory{
		{
			ExchangeName: e.Name, Status: "successful", TransferID: "081e4e9351dd0274922168da5f2d14ea6c495b1c3b440244f4a6dd9fe196bf2b",
			Timestamp: time.UnixMilli(1649012310000), Currency: "usdt", Amount: 9, TransferType: "deposit",
			CryptoToAddress: "TYASr5UV6HEcXatwdFQfmLVUqQQQMUxHLS", CryptoTxID: "081e4e9351dd0274922168da5f2d14ea6c495b1c3b440244f4a6dd9fe196bf2b", CryptoChain: "trc20",
		},
		{
			ExchangeName: e.Name, Status: "completed", TransferID: "1902194", Timestamp: time.UnixMilli(1649014002000), Currency: "usdt",
			Amount: 2, Fee: 1, TransferType: "withdrawal", CryptoToAddress: "TBjrW5JHDyPZjFc5nrRMhRWUDaJmhGhmD6",
			CryptoTxID: "47eeee2763ad49b8817524dacfa7d092fb58f8b0ab7e5d25473314df1a793c3d", CryptoChain: "trc20",
		},
	}
	assert.Equal(t, exp, result, "GetAccountFundingHistory should return the deposits and withdrawals")
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetWithdrawalsHistory(t.Context(), currency.USDT, asset.USDTMarginedFutures)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetWithdrawalsHistory must reject an asset other than spot")

	skipLiveWithoutCredentials(t)
	result, err := e.GetWithdrawalsHistory(t.Context(), currency.USDT, asset.Spot)
	require.NoError(t, err, "GetWithdrawalsHistory must not error")
	if !mockTests {
		assert.NotNil(t, result, "GetWithdrawalsHistory should return the account's withdrawals")
		return
	}
	exp := []exchange.WithdrawalHistory{{
		Status: "completed", TransferID: "1902194", Timestamp: time.UnixMilli(1649014002000), Currency: "usdt", Amount: 2, Fee: 1,
		TransferType: "withdrawal", CryptoToAddress: "TBjrW5JHDyPZjFc5nrRMhRWUDaJmhGhmD6",
		CryptoTxID: "47eeee2763ad49b8817524dacfa7d092fb58f8b0ab7e5d25473314df1a793c3d", CryptoChain: "trc20",
	}}
	assert.Equal(t, exp, result, "GetWithdrawalsHistory should return the coin's withdrawals")
}

func TestFundingStatuses(t *testing.T) {
	t.Parallel()
	for status, exp := range map[string]string{"1": "applying", "2": "successful", "3": "failed", "4": "cancelled", "5": "transfer", "9": "9"} {
		assert.Equalf(t, exp, depositStatus(status), "depositStatus should describe status %s", status)
	}
	for status, exp := range map[string]string{"1": "applying", "2": "cancelled", "3": "failed", "4": "completed", "5": "5"} {
		assert.Equalf(t, exp, withdrawalStatus(status), "withdrawalStatus should describe status %s", status)
	}
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetRecentTrades(t.Context(), futuresTestPair, asset.USDTMarginedFutures)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetRecentTrades must reject contracts, whose trades LBank doesn't serve")

	result, err := e.GetRecentTrades(t.Context(), spotTestPair, asset.Spot)
	require.NoError(t, err, "GetRecentTrades must not error")
	if !mockTests {
		assert.NotEmpty(t, result, "GetRecentTrades should return trades")
		return
	}
	require.Len(t, result, maxTradesSize, "GetRecentTrades must return every trade fetched")
	assert.True(t, slices.IsSortedFunc(result, func(a, b trade.Data) int { return a.Timestamp.Compare(b.Timestamp) }), "GetRecentTrades should return the trades oldest first")
	exp := trade.Data{
		Exchange: e.Name, TID: "49d0ef5069d3470490167f6dd00e7799", CurrencyPair: spotTestPair, AssetType: asset.Spot, Side: order.Buy, Price: 80940.84,
		Amount: 0.00002, Timestamp: time.UnixMilli(1792897604196),
	}
	assert.Equal(t, exp, result[0], "GetRecentTrades should return the oldest trade first")
}

func TestTradeData(t *testing.T) {
	t.Parallel()
	// LBank marks a trade at the ask, where the buyer took liquidity, with isBuyerMaker
	for _, tc := range []struct {
		in  TradeResponse
		exp trade.Data
	}{
		{
			in: TradeResponse{ID: "4f9ff15a834f4fec8b9e77393e01b9c0", Price: 80966.9, Quantity: 0.00328, Time: types.Time(time.UnixMilli(1792897777298)), IsBuyerMaker: true},
			exp: trade.Data{
				Exchange: e.Name, TID: "4f9ff15a834f4fec8b9e77393e01b9c0", CurrencyPair: spotTestPair, AssetType: asset.Spot, Side: order.Buy, Price: 80966.9,
				Amount: 0.00328, Timestamp: time.UnixMilli(1792897777298),
			},
		},
		{
			in: TradeResponse{ID: "1356e49b40fa4c0993da1ac2794721d0", Price: 80966.61, Quantity: 0.00021, Time: types.Time(time.UnixMilli(1792897776502))},
			exp: trade.Data{
				Exchange: e.Name, TID: "1356e49b40fa4c0993da1ac2794721d0", CurrencyPair: spotTestPair, AssetType: asset.Spot, Side: order.Sell, Price: 80966.61,
				Amount: 0.00021, Timestamp: time.UnixMilli(1792897776502),
			},
		},
	} {
		assert.Equalf(t, tc.exp, e.tradeData(&tc.in, spotTestPair), "tradeData should map trade %s", tc.in.ID)
	}
}

// historicTradesVenue serves trades the way LBank's trades endpoint does: up to size trades strictly after the time
// given, oldest first. It refuses a time it was already asked for, so a pager that stops advancing fails rather than
// hangs
type historicTradesVenue struct {
	mu     sync.Mutex
	trades []TradeResponse
	asked  map[string]bool
	times  []int64
}

func (v *historicTradesVenue) handler(t *testing.T) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		after, err := strconv.ParseInt(r.URL.Query().Get("time"), 10, 64)
		if !assert.NoError(t, err, "the pager should ask for trades after a time in milliseconds") {
			http.Error(w, "bad time", http.StatusBadRequest)
			return
		}
		size, err := strconv.Atoi(r.URL.Query().Get("size"))
		if !assert.NoError(t, err, "the pager should send a size") {
			http.Error(w, "bad size", http.StatusBadRequest)
			return
		}
		v.mu.Lock()
		defer v.mu.Unlock()
		if v.asked == nil {
			v.asked = make(map[string]bool)
		}
		if v.asked[r.URL.Query().Get("time")] {
			assert.Failf(t, "repeated page", "the pager asked for the trades after %d again", after)
			http.Error(w, "repeated page", http.StatusBadRequest)
			return
		}
		v.asked[r.URL.Query().Get("time")] = true
		v.times = append(v.times, after)
		page := make([]string, 0, size)
		for i := range v.trades {
			if v.trades[i].Time.Time().UnixMilli() <= after {
				continue
			}
			if len(page) == size {
				break
			}
			page = append(page, fmt.Sprintf(`{"quoteQty":%v,"price":%v,"qty":%v,"id":%q,"time":%d,"isBuyerMaker":%v}`,
				v.trades[i].QuoteQuantity, v.trades[i].Price, v.trades[i].Quantity, v.trades[i].ID, v.trades[i].Time.Time().UnixMilli(), v.trades[i].IsBuyerMaker))
		}
		writeResponse(t, w, `{"result":"true","data":[`+strings.Join(page, ",")+`],"error_code":0}`)
	}
}

// historicTrades returns a trade a millisecond from the start for each count, the trades at one millisecond sharing it
func historicTrades(start time.Time, counts ...int) []TradeResponse {
	var trades []TradeResponse
	for ms, count := range counts {
		for range count {
			price := float64(len(trades) + 1)
			trades = append(trades, TradeResponse{
				ID:            strconv.Itoa(len(trades)),
				Price:         price,
				Quantity:      0.5,
				QuoteQuantity: price / 2,
				Time:          types.Time(start.Add(time.Duration(ms) * time.Millisecond)),
				IsBuyerMaker:  len(trades)%2 == 0,
			})
		}
	}
	return trades
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	start := time.UnixMilli(1792884094000)
	_, err := e.GetHistoricTrades(t.Context(), spotTestPair, asset.Spot, start, start.Add(-time.Minute))
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetHistoricTrades must reject a start after the end")
	_, err = e.GetHistoricTrades(t.Context(), futuresTestPair, asset.USDTMarginedFutures, start, start.Add(time.Minute))
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetHistoricTrades must reject contracts, whose trades LBank doesn't serve")

	t.Run("pages", func(t *testing.T) {
		t.Parallel()
		// A trade before the start, 499 trades, 3 sharing the millisecond the first full page ends on, 600 more, and a
		// trade after the end
		counts := append([]int{1}, make([]int, 1102)...)
		for i := 1; i < len(counts); i++ {
			counts[i] = 1
		}
		counts[500] = 3
		venue := &historicTradesVenue{trades: historicTrades(start.Add(-time.Millisecond), counts...)}
		ex := newTestServerExchange(t, venue.handler(t))
		end := start.Add(time.Duration(len(counts)-3) * time.Millisecond)
		result, err := ex.GetHistoricTrades(t.Context(), spotTestPair, asset.Spot, start, end)
		require.NoError(t, err, "GetHistoricTrades must not error")
		// Trade IDs are compared, since a diff of a thousand trades takes testify minutes to print; tradeData's mapping is
		// covered by TestGetRecentTrades
		var exp []string
		for i := range venue.trades {
			tm := venue.trades[i].Time.Time()
			if tm.Before(start) || tm.After(end) {
				continue
			}
			exp = append(exp, venue.trades[i].ID)
		}
		got := make([]string, len(result))
		for i := range result {
			got[i] = result[i].TID
		}
		assert.Equal(t, exp, got, "GetHistoricTrades should return every trade in the range once, oldest first")
		assert.Equal(t, ex.tradeData(&venue.trades[1], spotTestPair), result[0], "GetHistoricTrades should return the first trade in the range first")
		assert.Equal(t, start.Add(-time.Millisecond).UnixMilli(), venue.times[0], "GetHistoricTrades should ask from a millisecond before the start")
		assert.Len(t, venue.times, 3, "GetHistoricTrades should stop once a page passes the end")
	})

	t.Run("crowded millisecond", func(t *testing.T) {
		t.Parallel()
		venue := &historicTradesVenue{trades: historicTrades(start, maxTradesSize+1)}
		ex := newTestServerExchange(t, venue.handler(t))
		_, err := ex.GetHistoricTrades(t.Context(), spotTestPair, asset.Spot, start, start.Add(time.Minute))
		assert.ErrorIs(t, err, errHistoryPageFull, "GetHistoricTrades should report a millisecond holding more trades than a page")
	})

	if !mockTests {
		result, err := e.GetHistoricTrades(t.Context(), spotTestPair, asset.Spot, time.Now().Add(-time.Minute*5), time.Now())
		require.NoError(t, err, "GetHistoricTrades must not error")
		assert.NotNil(t, result, "GetHistoricTrades should return trades")
	}
}

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	_, err := e.SubmitOrder(t.Context(), &order.Submit{Exchange: e.Name, Pair: futuresTestPair, AssetType: asset.USDTMarginedFutures, Side: order.Buy, Type: order.Limit, Price: 1, Amount: 1})
	require.ErrorIs(t, err, asset.ErrNotSupported, "SubmitOrder must reject contracts, which LBank's API doesn't trade")
	_, err = e.SubmitOrder(t.Context(), &order.Submit{Exchange: e.Name, Pair: spotTestPair, AssetType: asset.Spot, Side: order.Buy, Type: order.Limit, Price: 1, Amount: 1, TimeInForce: order.GoodTillTime})
	require.ErrorIs(t, err, order.ErrUnsupportedTimeInForce, "SubmitOrder must reject a time in force LBank doesn't offer")
	_, err = e.SubmitOrder(t.Context(), &order.Submit{Exchange: e.Name, Pair: spotTestPair, AssetType: asset.Spot, Side: order.Buy, Type: order.Market, Amount: 1})
	require.ErrorIs(t, err, order.ErrAmountMustBeSet, "SubmitOrder must require a market buy's quote amount")

	skipLiveUnlessCanManipulate(t)
	if !mockTests {
		req := liveTestOrderRequest(t)
		result, err := e.SubmitOrder(t.Context(), &order.Submit{Exchange: e.Name, Pair: spotTestPair, AssetType: asset.Spot, Side: order.Buy, Type: order.Limit, Price: req.Price, Amount: req.Amount})
		require.NoError(t, err, "SubmitOrder must not error")
		require.NotEmpty(t, result.OrderID, "SubmitOrder must return the order's ID")
		cancelLiveTestOrderOnCleanup(t, result.OrderID)
		assert.Equal(t, order.New, result.Status, "SubmitOrder should return the order's status")
		return
	}
	for _, tc := range []struct {
		name    string
		submit  *order.Submit
		orderID string
	}{
		{name: "limit buy", orderID: "0cf8a3de-4597-4296-af45-be7abaa06b01", submit: &order.Submit{Side: order.Buy, Type: order.Limit, Price: 50000, Amount: 0.001, ClientOrderID: "gct-limit"}},
		{name: "post only sell", orderID: "0cf8a3de-4597-4296-af45-be7abaa06b02", submit: &order.Submit{Side: order.Sell, Type: order.Limit, TimeInForce: order.PostOnly, Price: 90000, Amount: 0.002}},
		{name: "immediate or cancel buy", orderID: "0cf8a3de-4597-4296-af45-be7abaa06b03", submit: &order.Submit{Side: order.Buy, Type: order.Limit, TimeInForce: order.ImmediateOrCancel, Price: 80000, Amount: 0.001}},
		{name: "fill or kill sell", orderID: "0cf8a3de-4597-4296-af45-be7abaa06b04", submit: &order.Submit{Side: order.Sell, Type: order.Limit, TimeInForce: order.FillOrKill, Price: 81000, Amount: 0.001}},
		{name: "market buy", orderID: "0cf8a3de-4597-4296-af45-be7abaa06b05", submit: &order.Submit{Side: order.Buy, Type: order.Market, QuoteAmount: 100}},
		{name: "market sell", orderID: "0cf8a3de-4597-4296-af45-be7abaa06b06", submit: &order.Submit{Side: order.Sell, Type: order.Market, Amount: 0.001}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.submit.Exchange, tc.submit.Pair, tc.submit.AssetType = e.Name, spotTestPair, asset.Spot
			result, err := e.SubmitOrder(t.Context(), tc.submit)
			require.NoError(t, err, "SubmitOrder must not error")
			require.NotZero(t, result.Date, "SubmitOrder must date the order")
			require.NotZero(t, result.LastUpdated, "SubmitOrder must date the order")
			result.Date, result.LastUpdated = time.Time{}, time.Time{}
			status := order.New
			if tc.submit.Type == order.Market {
				status = order.Filled
			}
			exp := &order.SubmitResponse{
				Exchange: e.Name, Type: tc.submit.Type, Side: tc.submit.Side, Pair: spotTestPair, AssetType: asset.Spot,
				TimeInForce: tc.submit.TimeInForce, Price: tc.submit.Price, Amount: tc.submit.Amount, QuoteAmount: tc.submit.QuoteAmount,
				ClientOrderID: tc.submit.ClientOrderID, Status: status, OrderID: tc.orderID,
			}
			assert.Equal(t, exp, result, "SubmitOrder should return the placed order")
		})
	}
}

func TestOrderTypeString(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		side      order.Side
		orderType order.Type
		tif       order.TimeInForce
		exp       string
		err       error
	}{
		{side: order.Buy, orderType: order.Limit, exp: "buy"},
		{side: order.Sell, orderType: order.Limit, tif: order.GoodTillCancel, exp: "sell"},
		{side: order.Bid, orderType: order.Limit, tif: order.PostOnly, exp: "buy_maker"},
		{side: order.Ask, orderType: order.Limit, tif: order.GoodTillCancel | order.PostOnly, exp: "sell_maker"},
		{side: order.Buy, orderType: order.Limit, tif: order.ImmediateOrCancel, exp: "buy_ioc"},
		{side: order.Sell, orderType: order.Limit, tif: order.FillOrKill, exp: "sell_fok"},
		{side: order.Buy, orderType: order.Market, exp: "buy_market"},
		{side: order.Sell, orderType: order.Market, tif: order.ImmediateOrCancel, exp: "sell_market"},
		{side: order.Buy, orderType: order.Market, tif: order.FillOrKill, err: order.ErrUnsupportedTimeInForce},
		{side: order.Buy, orderType: order.Limit, tif: order.ImmediateOrCancel | order.PostOnly, err: order.ErrUnsupportedTimeInForce},
		{side: order.Buy, orderType: order.Limit, tif: order.GoodTillDay, err: order.ErrUnsupportedTimeInForce},
		{side: order.Buy, orderType: order.StopLimit, err: order.ErrUnsupportedOrderType},
		{side: order.AnySide, orderType: order.Limit, err: order.ErrSideIsInvalid},
	} {
		got, err := orderTypeString(tc.side, tc.orderType, tc.tif)
		if tc.err != nil {
			assert.ErrorIsf(t, err, tc.err, "orderTypeString should reject a %s %s %s order", tc.side, tc.orderType, tc.tif)
			continue
		}
		require.NoErrorf(t, err, "orderTypeString must not error for a %s %s %s order", tc.side, tc.orderType, tc.tif)
		assert.Equalf(t, tc.exp, got, "orderTypeString should map a %s %s %s order", tc.side, tc.orderType, tc.tif)
	}
}

func TestParseOrderType(t *testing.T) {
	t.Parallel()
	type parsed struct {
		side      order.Side
		orderType order.Type
		tif       order.TimeInForce
	}
	for in, exp := range map[string]parsed{
		"buy":         {side: order.Buy, orderType: order.Limit, tif: order.GoodTillCancel},
		"sell":        {side: order.Sell, orderType: order.Limit, tif: order.GoodTillCancel},
		"buy_market":  {side: order.Buy, orderType: order.Market},
		"sell_market": {side: order.Sell, orderType: order.Market},
		"buy_maker":   {side: order.Buy, orderType: order.Limit, tif: order.PostOnly},
		"sell_maker":  {side: order.Sell, orderType: order.Limit, tif: order.PostOnly},
		"buy_ioc":     {side: order.Buy, orderType: order.Limit, tif: order.ImmediateOrCancel},
		"sell_fok":    {side: order.Sell, orderType: order.Limit, tif: order.FillOrKill},
	} {
		side, orderType, tif, err := parseOrderType(in)
		require.NoErrorf(t, err, "parseOrderType must not error for %q", in)
		assert.Equalf(t, exp, parsed{side: side, orderType: orderType, tif: tif}, "parseOrderType should parse %q", in)
	}
	for in, exp := range map[string]error{
		"":          order.ErrSideIsInvalid,
		"hold":      order.ErrSideIsInvalid,
		"long_ioc":  order.ErrSideIsInvalid,
		"BUY":       order.ErrSideIsInvalid,
		"buy_stop":  order.ErrUnsupportedOrderType,
		"sell_post": order.ErrUnsupportedOrderType,
	} {
		_, _, _, err := parseOrderType(in)
		assert.ErrorIsf(t, err, exp, "parseOrderType should reject %q", in)
	}
}

func TestOrderStatus(t *testing.T) {
	t.Parallel()
	for in, exp := range map[int64]order.Status{
		-1: order.Cancelled,
		0:  order.Active,
		1:  order.PartiallyFilled,
		2:  order.Filled,
		3:  order.PartiallyFilledCancelled,
		4:  order.Cancelling,
	} {
		got, err := orderStatus(in)
		require.NoErrorf(t, err, "orderStatus must not error for %d", in)
		assert.Equalf(t, exp, got, "orderStatus should map %d", in)
	}
	_, err := orderStatus(5)
	assert.ErrorIs(t, err, order.ErrUnsupportedStatusType, "orderStatus should reject a status LBank doesn't document")
}

func TestOrderDetail(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		in   OrderResponse
		exp  *order.Detail
	}{
		{
			name: "partly filled limit order",
			in: OrderResponse{
				OrderID: "1", ClientOrderID: "c1", Type: "sell_ioc", Price: 100, OriginalQuantity: 2, OriginalQuoteOrderQuantity: 200,
				ExecutedQuantity: 0.5, CumulativeQuoteQuantity: 50.5, Status: 1, CreateTime: types.Time(time.UnixMilli(1)), UpdateTime: types.Time(time.UnixMilli(2)),
			},
			exp: &order.Detail{
				Exchange: e.Name, OrderID: "1", ClientOrderID: "c1", Pair: spotTestPair, AssetType: asset.Spot, Side: order.Sell, Type: order.Limit,
				TimeInForce: order.ImmediateOrCancel, Status: order.PartiallyFilled, Price: 100, Amount: 2, QuoteAmount: 200, ExecutedAmount: 0.5,
				RemainingAmount: 1.5, Cost: 50.5, AverageExecutedPrice: 101, Date: time.UnixMilli(1), LastUpdated: time.UnixMilli(2),
			},
		},
		{
			name: "filled market buy of a quote amount",
			in:   OrderResponse{OrderID: "2", Type: "buy_market", OriginalQuoteOrderQuantity: 100, ExecutedQuantity: 0.00125, CumulativeQuoteQuantity: 100, Status: 2},
			exp: &order.Detail{
				Exchange: e.Name, OrderID: "2", Pair: spotTestPair, AssetType: asset.Spot, Side: order.Buy, Type: order.Market, Status: order.Filled,
				QuoteAmount: 100, ExecutedAmount: 0.00125, Cost: 100, AverageExecutedPrice: 80000,
			},
		},
		{
			name: "unfilled order",
			in:   OrderResponse{OrderID: "3", Type: "buy", Price: 10, OriginalQuantity: 1, Status: 0},
			exp: &order.Detail{
				Exchange: e.Name, OrderID: "3", Pair: spotTestPair, AssetType: asset.Spot, Side: order.Buy, Type: order.Limit,
				TimeInForce: order.GoodTillCancel, Status: order.Active, Price: 10, Amount: 1, RemainingAmount: 1,
			},
		},
	} {
		got, err := e.orderDetail(&tc.in, spotTestPair)
		require.NoErrorf(t, err, "orderDetail must not error for a %s", tc.name)
		assert.Equalf(t, tc.exp, got, "orderDetail should map a %s", tc.name)
	}
	_, err := e.orderDetail(&OrderResponse{Type: "hold"}, spotTestPair)
	assert.ErrorIs(t, err, order.ErrSideIsInvalid, "orderDetail should reject an order type it can't map")
	_, err = e.orderDetail(&OrderResponse{Type: "buy", Status: 7}, spotTestPair)
	assert.ErrorIs(t, err, order.ErrUnsupportedStatusType, "orderDetail should reject a status it can't map")
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	_, err := e.ModifyOrder(t.Context(), &order.Modify{})
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "ModifyOrder should report LBank can't modify orders")
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, e.CancelOrder(t.Context(), nil), order.ErrCancelOrderIsNil, "CancelOrder must reject a nil request")
	require.ErrorIs(t, e.CancelOrder(t.Context(), &order.Cancel{AssetType: asset.Spot, OrderID: "1"}), order.ErrPairIsEmpty, "CancelOrder must require a pair")
	require.ErrorIs(t, e.CancelOrder(t.Context(), &order.Cancel{Pair: futuresTestPair, AssetType: asset.USDTMarginedFutures, OrderID: "1"}), asset.ErrNotSupported, "CancelOrder must reject contracts")
	require.ErrorIs(t, e.CancelOrder(t.Context(), &order.Cancel{Pair: spotTestPair, AssetType: asset.Spot}), order.ErrOrderIDNotSet, "CancelOrder must require an order ID or client order ID")

	skipLiveUnlessCanManipulate(t)
	if !mockTests {
		placed := placeLiveTestOrder(t)
		assert.NoError(t, e.CancelOrder(t.Context(), &order.Cancel{Pair: spotTestPair, AssetType: asset.Spot, OrderID: placed.OrderID}), "CancelOrder should cancel by order ID")
		return
	}
	assert.NoError(t, e.CancelOrder(t.Context(), &order.Cancel{Pair: spotTestPair, AssetType: asset.Spot, OrderID: "fc2e51b9-2e6e-4949-a5f3-35a4f10c0554"}), "CancelOrder should cancel by order ID")
	assert.NoError(t, e.CancelOrder(t.Context(), &order.Cancel{Pair: spotTestPair, AssetType: asset.Spot, ClientOrderID: "4b11c4ac-d54a-4bb5-86f9-06322174a036"}), "CancelOrder should cancel by client order ID")
}

func TestCancelBatchOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelBatchOrders(t.Context(), []order.Cancel{{Pair: spotTestPair, AssetType: asset.Spot, OrderID: "1"}})
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "CancelBatchOrders should report LBank can't cancel orders by ID in a batch")
}

func TestCancelAllOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Spot})
	require.ErrorIs(t, err, order.ErrPairIsEmpty, "CancelAllOrders must require a pair")
	_, err = e.CancelAllOrders(t.Context(), &order.Cancel{Pair: futuresTestPair, AssetType: asset.USDTMarginedFutures})
	require.ErrorIs(t, err, asset.ErrNotSupported, "CancelAllOrders must reject contracts")

	skipLiveUnlessCanManipulate(t)
	if !mockTests {
		t.Skip("CancelAllOrders is not tested live, as it would cancel every open order on the pair, not only the test's")
	}
	result, err := e.CancelAllOrders(t.Context(), &order.Cancel{Pair: spotTestPair, AssetType: asset.Spot})
	require.NoError(t, err, "CancelAllOrders must not error")
	exp := order.CancelAllResponse{Status: map[string]string{
		"fc2e51b9-2e6e-4949-a5f3-35a4f10c0554": order.Cancelled.String(),
		"2539cee7-a0de-4caa-bbbe-993d39bc8efb": order.Cancelled.String(),
	}}
	assert.Equal(t, exp, result, "CancelAllOrders should report each order LBank cancelled")
}

func TestGetServerTime(t *testing.T) {
	t.Parallel()
	_, err := e.GetServerTime(t.Context(), asset.Margin)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetServerTime must reject an asset LBank doesn't trade")
	for a, exp := range map[asset.Item]time.Time{asset.Spot: time.UnixMilli(1792890784108), asset.USDTMarginedFutures: time.UnixMilli(1792890789640)} {
		result, err := e.GetServerTime(t.Context(), a)
		require.NoErrorf(t, err, "GetServerTime must not error for %s", a)
		if mockTests {
			assert.Equalf(t, exp, result, "GetServerTime should return the %s server time", a)
			continue
		}
		assert.WithinDurationf(t, time.Now(), result, time.Minute, "GetServerTime should return the current %s server time", a)
	}
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderInfo(t.Context(), "1", futuresTestPair, asset.USDTMarginedFutures)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetOrderInfo must reject contracts")
	_, err = e.GetOrderInfo(t.Context(), "", spotTestPair, asset.Spot)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetOrderInfo must require an order ID")
	_, err = e.GetOrderInfo(t.Context(), "1", currency.EMPTYPAIR, asset.Spot)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetOrderInfo must require the order's pair")

	skipLiveWithoutCredentials(t)
	if !mockTests {
		_, err = e.GetOrderInfo(t.Context(), "fc2e51b9-2e6e-4949-a5f3-35a4f10c0554", spotTestPair, asset.Spot)
		assert.Error(t, err, "GetOrderInfo should report an order the account doesn't hold")
		return
	}
	result, err := e.GetOrderInfo(t.Context(), "fc2e51b9-2e6e-4949-a5f3-35a4f10c0554", spotTestPair, asset.Spot)
	require.NoError(t, err, "GetOrderInfo must not error")
	exp := &order.Detail{
		Exchange: e.Name, OrderID: "fc2e51b9-2e6e-4949-a5f3-35a4f10c0554", ClientOrderID: "4b11c4ac-d54a-4bb5-86f9-06322174a036", Pair: spotTestPair,
		AssetType: asset.Spot, Side: order.Buy, Type: order.Limit, TimeInForce: order.PostOnly, Status: order.PartiallyFilledCancelled,
		Price: 51000, Amount: 0.001, QuoteAmount: 51, ExecutedAmount: 0.0004, Cost: 20.4, Date: time.UnixMilli(1772335196123),
		LastUpdated: time.UnixMilli(1772335197000),
	}
	require.InDelta(t, 0.0006, result.RemainingAmount, 1e-12, "GetOrderInfo must return the amount left")
	require.InDelta(t, 51000, result.AverageExecutedPrice, 1e-9, "GetOrderInfo must return the average price")
	result.RemainingAmount, result.AverageExecutedPrice = 0, 0
	assert.Equal(t, exp, result, "GetOrderInfo should return the order in any state")
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := e.GetDepositAddress(t.Context(), currency.EMPTYCODE, "", "trc20")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetDepositAddress must require a coin")

	skipLiveWithoutCredentials(t)
	result, err := e.GetDepositAddress(t.Context(), currency.USDT, "", "trc20")
	require.NoError(t, err, "GetDepositAddress must not error")
	if !mockTests {
		assert.NotEmpty(t, result.Address, "GetDepositAddress should return an address")
		return
	}
	assert.Equal(t, &deposit.Address{Address: "TDxtabCC8iQwaxUUrPcE4WL2jArGAfvQ5A", Chain: "trc20"}, result, "GetDepositAddress should return the chain's address")

	result, err = e.GetDepositAddress(t.Context(), currency.XRP, "", "")
	require.NoError(t, err, "GetDepositAddress must not error on the default chain")
	assert.Equal(t, &deposit.Address{Address: "rLbKJjgXNjTGLFrEXbF5yBuWR3XWHF5nMa", Tag: "107824"}, result, "GetDepositAddress should return the address and its memo as the tag")

	empty := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeResponse(t, w, `{"result":true,"data":{"address":"","memo":null,"coin":"usdt"},"error_code":0}`)
	}))
	_, err = empty.GetDepositAddress(t.Context(), currency.USDT, "", "")
	assert.ErrorIs(t, err, deposit.ErrAddressNotFound, "GetDepositAddress should report a coin without an address")
}

func TestWithdrawCryptocurrencyFunds(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawCryptocurrencyFunds(t.Context(), &withdraw.Request{Exchange: e.Name, Type: withdraw.Crypto, Currency: currency.USDT, Amount: 10})
	require.Error(t, err, "WithdrawCryptocurrencyFunds must validate the request")

	skipLiveUnlessCanManipulate(t)
	if !mockTests {
		t.Skip("WithdrawCryptocurrencyFunds moves funds off LBank, which a live test cannot undo")
	}
	req := &withdraw.Request{
		Exchange:    e.Name,
		Type:        withdraw.Crypto,
		Currency:    currency.USDT,
		Amount:      10,
		Description: "gct",
		Crypto:      withdraw.CryptoRequest{Address: "TDxtabCC8iQwaxUUrPcE4WL2jArGAfvQ5A", Chain: "trc20"},
	}
	// Without a fee the chain's fee is looked up, as LBank requires it
	result, err := e.WithdrawCryptocurrencyFunds(t.Context(), req)
	require.NoError(t, err, "WithdrawCryptocurrencyFunds must not error")
	assert.Equal(t, &withdraw.ExchangeResponse{ID: "1900378"}, result, "WithdrawCryptocurrencyFunds should return the withdrawal's ID")

	req.Crypto.FeeAmount, req.ClientOrderID = 1.5, "gct-withdrawal-1"
	result, err = e.WithdrawCryptocurrencyFunds(t.Context(), req)
	require.NoError(t, err, "WithdrawCryptocurrencyFunds must not error with a fee")
	assert.Equal(t, &withdraw.ExchangeResponse{ID: "1900377"}, result, "WithdrawCryptocurrencyFunds should send the fee and client ID given")

	req.Crypto.FeeAmount, req.Crypto.Chain = 0, "bep20"
	_, err = e.WithdrawCryptocurrencyFunds(t.Context(), req)
	assert.ErrorIs(t, err, currency.ErrCurrencyNotFound, "WithdrawCryptocurrencyFunds should report a chain it finds no fee for")
}

func TestNetworkWithdrawalFee(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	for _, tc := range []struct {
		coin  currency.Code
		chain string
		exp   float64
	}{
		{coin: currency.USDT, exp: 1.5},
		{coin: currency.USDT, chain: "TRC20", exp: 1.5},
		{coin: currency.USDT, chain: "erc20", exp: 1.1},
		{coin: currency.DOGE, exp: 9.7059},
	} {
		got, err := e.networkWithdrawalFee(t.Context(), tc.coin, tc.chain, 100)
		require.NoErrorf(t, err, "networkWithdrawalFee must not error for %s on %q", tc.coin, tc.chain)
		if mockTests {
			assert.InDeltaf(t, tc.exp, got, 1e-12, "networkWithdrawalFee should return the fee of %s on %q", tc.coin, tc.chain)
		}
	}
	_, err := e.networkWithdrawalFee(t.Context(), currency.BTC, "", 1)
	assert.ErrorIs(t, err, currency.ErrCurrencyNotFound, "networkWithdrawalFee should report a coin the account has no chains for")
}

func TestWithdrawalFee(t *testing.T) {
	t.Parallel()
	for feeType, exp := range map[uint64]float64{0: 2, 1: 2, 2: 0.5, 3: 2.5} {
		assert.InDeltaf(t, exp, withdrawalFee(feeType, 2, 0.005, 100), 1e-12, "withdrawalFee should charge fee type %d", feeType)
	}
}

func TestWithdrawFiatFunds(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawFiatFunds(t.Context(), &withdraw.Request{})
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "WithdrawFiatFunds should report LBank can't withdraw fiat")
	_, err = e.WithdrawFiatFundsToInternationalBank(t.Context(), &withdraw.Request{})
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "WithdrawFiatFundsToInternationalBank should report LBank can't withdraw fiat")
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.USDTMarginedFutures, Side: order.AnySide, Type: order.AnyType})
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetActiveOrders must reject contracts")

	skipLiveWithoutCredentials(t)
	// Without pairs the enabled pairs are listed, BTC-USDT alone in the test config
	result, err := e.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Side: order.AnySide, Type: order.AnyType})
	require.NoError(t, err, "GetActiveOrders must not error")
	if !mockTests {
		assert.NotNil(t, result, "GetActiveOrders should return the open orders")
		return
	}
	exp := order.FilteredOrders{
		{
			Exchange: e.Name, OrderID: "11f0e0a7-eac3-436b-b2d3-779c4f910e31", ClientOrderID: "8e145780-033c-4f59-8cdc-c2e75856db65", Pair: spotTestPair,
			AssetType: asset.Spot, Side: order.Buy, Type: order.Limit, TimeInForce: order.GoodTillCancel, Status: order.Active, Price: 50000, Amount: 0.01,
			QuoteAmount: 500, RemainingAmount: 0.01, Date: time.UnixMilli(1772335195111), LastUpdated: time.UnixMilli(1772335196000),
		},
		{
			Exchange: e.Name, OrderID: "46e9dbe6-688b-4337-b7b8-e91cfb3e2cd5", Pair: spotTestPair, AssetType: asset.Spot, Side: order.Sell, Type: order.Limit,
			TimeInForce: order.FillOrKill, Status: order.PartiallyFilled, Price: 100000, Amount: 0.01, QuoteAmount: 1000, ExecutedAmount: 0.003,
			RemainingAmount: 0.007, Cost: 300, AverageExecutedPrice: 100000, Date: time.UnixMilli(1772335195222), LastUpdated: time.UnixMilli(1772335199000),
		},
	}
	assert.Equal(t, exp, result, "GetActiveOrders should return the open orders")
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.USDTMarginedFutures, Side: order.AnySide, Type: order.AnyType})
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetOrderHistory must reject contracts")

	skipLiveWithoutCredentials(t)
	result, err := e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Side: order.AnySide, Type: order.AnyType, Pairs: currency.Pairs{spotTestPair}})
	require.NoError(t, err, "GetOrderHistory must not error")
	if !mockTests {
		assert.NotNil(t, result, "GetOrderHistory should return the order history")
		return
	}
	exp := order.FilteredOrders{
		{
			Exchange: e.Name, OrderID: "2539cee7-a0de-4caa-bbbe-993d39bc8efb", Pair: spotTestPair, AssetType: asset.Spot, Side: order.Buy, Type: order.Limit,
			TimeInForce: order.GoodTillCancel, Status: order.Cancelled, Price: 51000, Amount: 0.01, QuoteAmount: 510, RemainingAmount: 0.01,
			Date: time.UnixMilli(1772335195333), LastUpdated: time.UnixMilli(1772335196000),
		},
		{
			Exchange: e.Name, OrderID: "5ca490f9-eede-45c7-82fe-89b6beb3fd7b", Pair: spotTestPair, AssetType: asset.Spot, Side: order.Buy, Type: order.Market,
			Status: order.Filled, QuoteAmount: 100, ExecutedAmount: 0.00125, Cost: 100, AverageExecutedPrice: 80000,
			Date: time.UnixMilli(1772335195444), LastUpdated: time.UnixMilli(1772335197000),
		},
	}
	assert.Equal(t, exp, result, "GetOrderHistory should return the cancelled and filled orders")
}

func TestListOrders(t *testing.T) {
	t.Parallel()
	// orders returns a page of open limit buys whose IDs carry the page's prefix
	orders := func(prefix string, count int) []OrderResponse {
		resp := make([]OrderResponse, count)
		for i := range resp {
			resp[i] = OrderResponse{OrderID: prefix + strconv.Itoa(i), Type: "buy", Price: 1, OriginalQuantity: 1}
		}
		return resp
	}
	ethPair := currency.NewPairWithDelimiter("eth", "usdt", currency.UnderscoreDelimiter)
	for _, tc := range []struct {
		name string
		page func(n uint64) *OrdersPageResponse
		// pages is the number of pages each pair should be asked for, and orders the number of orders each should return
		pages  int
		orders int
		err    error
	}{
		{
			name: "short page",
			page: func(n uint64) *OrdersPageResponse {
				if n == 1 {
					return &OrdersPageResponse{Orders: orders("a", maxOrdersPageLength)}
				}
				return &OrdersPageResponse{Orders: orders("b", 3)}
			},
			pages: 2, orders: maxOrdersPageLength + 3,
		},
		{
			name: "last page by total",
			page: func(uint64) *OrdersPageResponse {
				return &OrdersPageResponse{Total: 1, Orders: orders("a", maxOrdersPageLength)}
			},
			pages: 1, orders: maxOrdersPageLength,
		},
		{
			name: "empty page",
			page: func(n uint64) *OrdersPageResponse {
				if n == 1 {
					return &OrdersPageResponse{Orders: orders("a", maxOrdersPageLength)}
				}
				return &OrdersPageResponse{}
			},
			pages: 2, orders: maxOrdersPageLength,
		},
		{
			name:  "repeated page",
			page:  func(uint64) *OrdersPageResponse { return &OrdersPageResponse{Orders: orders("a", maxOrdersPageLength)} },
			pages: 2, orders: maxOrdersPageLength,
		},
		{
			name: "endless pages",
			page: func(n uint64) *OrdersPageResponse {
				return &OrdersPageResponse{Orders: orders(strconv.FormatUint(n, 10)+"-", maxOrdersPageLength)}
			},
			pages: maxOrderPages, err: errHistoryPageLimit,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			asked := make(map[string][]uint64)
			list := func(_ context.Context, req *OrdersPageRequest) (*OrdersPageResponse, error) {
				assert.Equal(t, uint64(maxOrdersPageLength), req.PageLength, "listOrders should ask for full pages")
				asked[req.Pair.String()] = append(asked[req.Pair.String()], req.CurrentPage)
				return tc.page(req.CurrentPage), nil
			}
			pairs := currency.Pairs{spotTestPair, ethPair}
			if tc.err != nil {
				pairs = pairs[:1]
			}
			result, err := e.listOrders(t.Context(), pairs, list)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err, "listOrders must stop paging at the page limit")
				assert.Len(t, asked[spotTestPair.String()], tc.pages, "listOrders should ask for pages up to the limit")
				return
			}
			require.NoError(t, err, "listOrders must not error")
			expPages := make([]uint64, tc.pages)
			for i := range expPages {
				expPages[i] = uint64(i) + 1
			}
			exp := map[string][]uint64{spotTestPair.String(): expPages, ethPair.String(): expPages}
			assert.Equal(t, exp, asked, "listOrders should page each pair from the first page")
			require.Len(t, result, 2*tc.orders, "listOrders must return each pair's orders once")
			assert.Equal(t, spotTestPair, result[0].Pair, "listOrders should return the first pair's orders first")
			assert.Equal(t, ethPair, result[tc.orders].Pair, "listOrders should return the second pair's orders after")
		})
	}

	listErr := func(context.Context, *OrdersPageRequest) (*OrdersPageResponse, error) { return nil, errRequestFailed }
	_, err := e.listOrders(t.Context(), currency.Pairs{spotTestPair}, listErr)
	assert.ErrorIs(t, err, errRequestFailed, "listOrders should return a listing's error")
	badOrder := func(context.Context, *OrdersPageRequest) (*OrdersPageResponse, error) {
		return &OrdersPageResponse{Orders: []OrderResponse{{OrderID: "1", Type: "hold"}}}, nil
	}
	_, err = e.listOrders(t.Context(), currency.Pairs{spotTestPair}, badOrder)
	assert.ErrorIs(t, err, order.ErrSideIsInvalid, "listOrders should return an order it can't map as an error")
}

func TestGetFeeByType(t *testing.T) {
	t.Parallel()
	_, err := e.GetFeeByType(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFeeByType must reject a nil fee builder")

	for _, tc := range []struct {
		name    string
		builder *exchange.FeeBuilder
		exp     float64
	}{
		{name: "offline trade", builder: &exchange.FeeBuilder{FeeType: exchange.OfflineTradeFee, Pair: spotTestPair, PurchasePrice: 50000, Amount: 0.01}, exp: 0.5},
		{name: "taker trade", builder: &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyTradeFee, Pair: spotTestPair, PurchasePrice: 50000, Amount: 0.01}, exp: 0.5},
		{name: "maker trade", builder: &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyTradeFee, Pair: spotTestPair, IsMaker: true, PurchasePrice: 50000, Amount: 0.01}, exp: 0.4},
		{name: "withdrawal", builder: &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyWithdrawalFee, Pair: spotTestPair, Amount: 1}, exp: 0.00005},
		{name: "bank deposit", builder: &exchange.FeeBuilder{FeeType: exchange.InternationalBankDepositFee, Amount: 1}},
	} {
		got, err := e.GetFeeByType(t.Context(), tc.builder)
		require.NoErrorf(t, err, "GetFeeByType must not error for a %s", tc.name)
		if mockTests {
			assert.InDeltaf(t, tc.exp, got, 1e-12, "GetFeeByType should return the fee of a %s", tc.name)
		} else {
			assert.GreaterOrEqualf(t, got, 0.0, "GetFeeByType should return the fee of a %s", tc.name)
		}
	}

	// Without credentials a trade's fee is the standard rate
	ex := new(Exchange)
	require.NoError(t, setupTestExchange(ex), "setupTestExchange must not error")
	got, err := ex.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyTradeFee, Pair: spotTestPair, IsMaker: true, PurchasePrice: 50000, Amount: 0.01})
	require.NoError(t, err, "GetFeeByType must not error without credentials")
	assert.InDelta(t, 0.5, got, 1e-12, "GetFeeByType should charge the standard rate without credentials")
}

func TestGetAvailableTransferChains(t *testing.T) {
	t.Parallel()
	_, err := e.GetAvailableTransferChains(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetAvailableTransferChains must require a coin")

	result, err := e.GetAvailableTransferChains(t.Context(), currency.USDT)
	require.NoError(t, err, "GetAvailableTransferChains must not error")
	if mockTests {
		assert.Equal(t, []string{"trc20", "erc20", "omni"}, result, "GetAvailableTransferChains should return every chain LBank lists for the coin")
		return
	}
	assert.Contains(t, result, "trc20", "GetAvailableTransferChains should return the coin's chains")
}

func TestValidateAPICredentials(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	assert.NoError(t, e.ValidateAPICredentials(t.Context(), asset.Spot), "ValidateAPICredentials should accept working credentials")
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	start := time.Unix(1792800000, 0)
	_, err := e.GetHistoricCandles(t.Context(), futuresTestPair, asset.USDTMarginedFutures, kline.OneHour, start, start.Add(6*time.Hour))
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetHistoricCandles must reject contracts, whose candles LBank doesn't serve")

	// These candles are long settled, so the live venue returns the same ones
	result, err := e.GetHistoricCandles(t.Context(), spotTestPair, asset.Spot, kline.OneHour, start, start.Add(6*time.Hour))
	require.NoError(t, err, "GetHistoricCandles must not error")
	exp := &kline.Item{
		Exchange: e.Name, Pair: spotTestPair, Asset: asset.Spot, Interval: kline.OneHour,
		Candles: []kline.Candle{
			{Time: start, Open: 81189.8, High: 81211.94, Low: 80608.01, Close: 80622.55, Volume: 195.01898},
			{Time: start.Add(time.Hour), Open: 80622.55, High: 80962.64, Low: 80620.45, Close: 80932.06, Volume: 80.85023},
			{Time: start.Add(2 * time.Hour), Open: 80932.06, High: 80992.09, Low: 80845.76, Close: 80990.93, Volume: 42.19741},
			{Time: start.Add(3 * time.Hour), Open: 80990.93, High: 81090.86, Low: 80848.43, Close: 80971.68, Volume: 90.46431},
			{Time: start.Add(4 * time.Hour), Open: 80971.68, High: 81092.52, Low: 80841.71, Close: 80880.94, Volume: 90.22035},
			{Time: start.Add(5 * time.Hour), Open: 80880.94, High: 80889.36, Low: 80726.48, Close: 80770.51, Volume: 205.79895},
		},
	}
	assert.Equal(t, exp, result, "GetHistoricCandles should return the window's candles")

	// LBank's day candles start at midnight UTC+8, so UTC day candles are built from 4 hour candles
	dayStart := time.Unix(1792627200, 0)
	result, err = e.GetHistoricCandles(t.Context(), spotTestPair, asset.Spot, kline.OneDay, dayStart, dayStart.Add(48*time.Hour))
	require.NoError(t, err, "GetHistoricCandles must not error for day candles")
	assert.Equal(t, expDayCandles(dayStart), result, "GetHistoricCandles should build UTC day candles")
}

// expDayCandles returns the UTC day candles of BTC-USDT built from the 4 hour candles the fixtures record from the day
// given
func expDayCandles(start time.Time) *kline.Item {
	return &kline.Item{
		Exchange: e.Name, Pair: spotTestPair, Asset: asset.Spot, Interval: kline.OneDay,
		Candles: []kline.Candle{
			{Time: start, Open: 84096.12, High: 84375.88, Low: 80941.53, Close: 81416.8, Volume: 913.88683 + 784.74756 + 766.37354 + 1131.02148 + 767.52199 + 271.31405},
			{Time: start.Add(24 * time.Hour), Open: 81416.8, High: 82052.3, Low: 80520.54, Close: 81189.8, Volume: 708.54744 + 347.2973 + 422.73095 + 1323.01356 + 501.55758 + 621.80753},
		},
	}
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	start := time.Unix(1792627200, 0)
	_, err := e.GetHistoricCandlesExtended(t.Context(), futuresTestPair, asset.USDTMarginedFutures, kline.OneDay, start, start.Add(48*time.Hour))
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetHistoricCandlesExtended must reject contracts, whose candles LBank doesn't serve")

	result, err := e.GetHistoricCandlesExtended(t.Context(), spotTestPair, asset.Spot, kline.OneDay, start, start.Add(48*time.Hour))
	require.NoError(t, err, "GetHistoricCandlesExtended must not error")
	assert.Equal(t, expDayCandles(start), result, "GetHistoricCandlesExtended should build UTC day candles")

	t.Run("ranges", func(t *testing.T) {
		t.Parallel()
		var mu sync.Mutex
		var asked []string
		ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			q := r.URL.Query()
			mu.Lock()
			asked = append(asked, q.Get("time")+"/"+q.Get("size"))
			mu.Unlock()
			from, err := strconv.ParseInt(q.Get("time"), 10, 64)
			if !assert.NoError(t, err, "GetHistoricCandlesExtended should send a start in seconds") {
				http.Error(w, "bad time", http.StatusBadRequest)
				return
			}
			size, err := strconv.Atoi(q.Get("size"))
			if !assert.NoError(t, err, "GetHistoricCandlesExtended should send a size") {
				http.Error(w, "bad size", http.StatusBadRequest)
				return
			}
			candles := make([]string, size)
			for i := range candles {
				candles[i] = fmt.Sprintf("[%d,1,2,0.5,1.5,%d]", from+int64(i)*3600, i+1)
			}
			writeResponse(t, w, `{"result":"true","data":[`+strings.Join(candles, ",")+`],"error_code":0}`)
		}))
		ex.Features.Enabled.Kline.GlobalResultLimit = 3
		hourStart := time.Unix(1792800000, 0)
		result, err := ex.GetHistoricCandlesExtended(t.Context(), spotTestPair, asset.Spot, kline.OneHour, hourStart, hourStart.Add(7*time.Hour))
		require.NoError(t, err, "GetHistoricCandlesExtended must not error")
		assert.Equal(t, []string{"1792800000/3", "1792810800/3", "1792821600/1"}, asked, "GetHistoricCandlesExtended should ask for each range's candles alone")
		exp := &kline.Item{Exchange: ex.Name, Pair: spotTestPair, Asset: asset.Spot, Interval: kline.OneHour}
		for i, volume := range []float64{1, 2, 3, 1, 2, 3, 1} {
			exp.Candles = append(exp.Candles, kline.Candle{Time: hourStart.Add(time.Duration(i) * time.Hour), Open: 1, High: 2, Low: 0.5, Close: 1.5, Volume: volume})
		}
		assert.Equal(t, exp, result, "GetHistoricCandlesExtended should join the ranges' candles")
	})
}

func TestGetFuturesContractDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesContractDetails(t.Context(), asset.Spot)
	require.ErrorIs(t, err, futures.ErrNotFuturesAsset, "GetFuturesContractDetails must reject spot")
	_, err = e.GetFuturesContractDetails(t.Context(), asset.CoinMarginedFutures)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetFuturesContractDetails must reject coin margined futures, which LBank doesn't list")

	result, err := e.GetFuturesContractDetails(t.Context(), asset.USDTMarginedFutures)
	require.NoError(t, err, "GetFuturesContractDetails must not error")
	if !mockTests {
		assert.NotEmpty(t, result, "GetFuturesContractDetails should return contracts")
		return
	}
	usdt := currency.NewCode("USDT")
	btc := currency.NewPair(currency.NewCode("BTC"), usdt)
	eth := currency.NewPair(currency.NewCode("ETH"), usdt)
	exp := []futures.Contract{
		{
			Exchange: e.Name, Name: btc, Underlying: btc, Asset: asset.USDTMarginedFutures, IsActive: true, Type: futures.Perpetual,
			SettlementType: futures.Linear, SettlementCurrency: usdt, MarginCurrency: usdt, Multiplier: 1, MaxLeverage: 200,
		},
		{
			Exchange: e.Name, Name: eth, Underlying: eth, Asset: asset.USDTMarginedFutures, IsActive: true, Type: futures.Perpetual,
			SettlementType: futures.Linear, SettlementCurrency: usdt, MarginCurrency: usdt, Multiplier: 1, MaxLeverage: 200,
		},
	}
	assert.Equal(t, exp, result, "GetFuturesContractDetails should return every perpetual contract")
}

func TestGetLatestFundingRates(t *testing.T) {
	t.Parallel()
	_, err := e.GetLatestFundingRates(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetLatestFundingRates must reject a nil request")
	_, err = e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.Spot})
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetLatestFundingRates must reject spot")
	_, err = e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.USDTMarginedFutures, Pair: currency.NewPair(currency.NewCode("NOPE"), currency.USDT)})
	require.ErrorIs(t, err, currency.ErrCurrencyNotFound, "GetLatestFundingRates must report a contract LBank doesn't list")

	// Without a pair the enabled contracts' rates are returned, BTC-USDT alone in the test config
	for _, pair := range []currency.Pair{futuresTestPair, currency.EMPTYPAIR} {
		result, err := e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.USDTMarginedFutures, Pair: pair})
		require.NoErrorf(t, err, "GetLatestFundingRates must not error for %q", pair)
		require.Lenf(t, result, 1, "GetLatestFundingRates must return a rate for %q", pair)
		require.NotZerof(t, result[0].TimeChecked, "GetLatestFundingRates must record when it checked the rate for %q", pair)
		if !mockTests {
			assert.Truef(t, result[0].TimeOfNextRate.After(result[0].LatestRate.Time), "GetLatestFundingRates should return a rate before the next for %q", pair)
			continue
		}
		result[0].TimeChecked = time.Time{}
		exp := []fundingrate.LatestRateResponse{{
			Exchange: e.Name, Asset: asset.USDTMarginedFutures, Pair: futuresTestPair,
			LatestRate:     fundingrate.Rate{Time: time.UnixMilli(1792886400000), Rate: mustPreciseNumber(t, "0.00009783").Decimal()},
			TimeOfNextRate: time.UnixMilli(1792915200000),
		}}
		assert.Equalf(t, exp, result, "GetLatestFundingRates should return the rate set an interval before the next for %q", pair)
	}
}

func TestIsPerpetualFutureCurrency(t *testing.T) {
	t.Parallel()
	for a, exp := range map[asset.Item]bool{asset.USDTMarginedFutures: true, asset.Spot: false, asset.CoinMarginedFutures: false} {
		got, err := e.IsPerpetualFutureCurrency(a, futuresTestPair)
		require.NoErrorf(t, err, "IsPerpetualFutureCurrency must not error for %s", a)
		assert.Equalf(t, exp, got, "IsPerpetualFutureCurrency should report whether %s is perpetual", a)
	}
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	require.ErrorIs(t, ex.UpdateOrderExecutionLimits(t.Context(), asset.Margin), asset.ErrNotSupported, "UpdateOrderExecutionLimits must reject an asset LBank doesn't trade")

	for a, exp := range map[asset.Item]limits.MinMaxLevel{
		asset.Spot: {
			Key: key.NewExchangeAssetPair(ex.Name, asset.Spot, spotTestPair), PriceStepIncrementSize: 0.01, AmountStepIncrementSize: 0.00001,
			MinimumBaseAmount: 0.00001, MinNotional: 1,
		},
		asset.USDTMarginedFutures: {
			Key: key.NewExchangeAssetPair(ex.Name, asset.USDTMarginedFutures, futuresTestPair), PriceStepIncrementSize: 0.1, AmountStepIncrementSize: 0.0001,
			MinimumBaseAmount: 0.0001, MaximumBaseAmount: 150, MinNotional: 1,
		},
	} {
		require.NoErrorf(t, ex.UpdateOrderExecutionLimits(t.Context(), a), "UpdateOrderExecutionLimits must not error for %s", a)
		got, err := limits.GetOrderExecutionLimits(exp.Key)
		require.NoErrorf(t, err, "UpdateOrderExecutionLimits must store the %s limits", a)
		if !mockTests {
			assert.Positivef(t, got.PriceStepIncrementSize, "UpdateOrderExecutionLimits should store the %s price step", a)
			continue
		}
		require.NotZerof(t, got.UpdatedAt, "UpdateOrderExecutionLimits must date the %s limits", a)
		got.UpdatedAt = time.Time{}
		assert.Equalf(t, exp, got, "UpdateOrderExecutionLimits should store the %s limits", a)
	}
}

func TestGetCurrencyTradeURL(t *testing.T) {
	t.Parallel()
	_, err := e.GetCurrencyTradeURL(t.Context(), asset.Spot, currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetCurrencyTradeURL must require a pair")
	_, err = e.GetCurrencyTradeURL(t.Context(), asset.Margin, spotTestPair)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetCurrencyTradeURL must reject an asset LBank doesn't trade")
	for a, exp := range map[asset.Item]string{asset.Spot: "https://www.lbank.com/trade/btc_usdt", asset.USDTMarginedFutures: "https://www.lbank.com/futures/btcusdt"} {
		pair := spotTestPair
		if a == asset.USDTMarginedFutures {
			pair = futuresTestPair
		}
		got, err := e.GetCurrencyTradeURL(t.Context(), a, pair)
		require.NoErrorf(t, err, "GetCurrencyTradeURL must not error for %s", a)
		assert.Equalf(t, exp, got, "GetCurrencyTradeURL should return the %s trade page", a)
	}
}

func TestCheckSpot(t *testing.T) {
	t.Parallel()
	assert.NoError(t, checkSpot(asset.Spot), "checkSpot should accept spot")
	assert.ErrorIs(t, checkSpot(asset.USDTMarginedFutures), asset.ErrNotSupported, "checkSpot should reject contracts")
}
