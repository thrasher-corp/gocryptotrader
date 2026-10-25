package coinbase

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
)

var testETHUSDCPerpetual = currency.NewPairWithDelimiter("ETH", "USDC-PERPETUAL", currency.UnderscoreDelimiter)

// replayDerivativesFixture handles every notification of a gateway fixture on a connection, returning the errors
// handling them
func replayDerivativesFixture(t *testing.T, ex *Exchange, conn websocket.Connection, name string) []error {
	t.Helper()
	var errs []error
	for _, message := range websocketFixture(t, name) {
		if err := ex.wsHandleDerivatives(t.Context(), conn, message); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

func TestWebsocketHandleDerivativesPublic(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	conn, err := ex.Websocket.CreateTestConnection(derivativesStreamsConnection)
	require.NoError(t, err, "CreateTestConnection must not error")
	require.Empty(t, replayDerivativesFixture(t, ex, conn, "wsDerivativesPublic.json"), "replaying the gateway's public fixture must not error")

	tickers := map[currency.Pair]*ticker.Price{
		testUSDCPerpetual: {
			Bid: 82378.7, BidSize: 2.014, Ask: 82378.8, AskSize: 2.7001, Last: 82378.7, Close: 82378.7, High: 82481.3, Low: 80583.1,
			BaseVolume: 36012.0853, QuoteVolume: 2934908089.6423597, PercentChange24Hour: 0.8687, MarkPrice: 82378.92, IndexPrice: 82344.6,
			OpenInterest: 1791.2812, Pair: testUSDCPerpetual, AssetType: asset.USDCMarginedFutures, LastUpdated: time.UnixMilli(1792374088702),
		},
		// The incremental ticker's changes are merged onto its snapshot
		testCoinPerpetual: {
			Bid: 82369.5, BidSize: 3000, Ask: 82370, AskSize: 62300, Last: 82371, Close: 82371, High: 82542, Low: 80600,
			BaseVolume: 12474.68536635, QuoteVolume: 1016357790, PercentChange24Hour: 0.8627, MarkPrice: 82369.45, IndexPrice: 82344.6,
			OpenInterest: 851292200, Pair: testCoinPerpetual, AssetType: asset.CoinMarginedFutures, LastUpdated: time.UnixMilli(1792374089174),
		},
	}
	for p, exp := range tickers {
		exp.ExchangeName = ex.Name
		got, err := ticker.GetTicker(ex.Name, p, exp.AssetType)
		require.NoErrorf(t, err, "GetTicker must not error for %s", p)
		assert.Equalf(t, exp, got, "the stored %s ticker should be the last streamed", p)
	}

	book, err := ex.Websocket.Orderbook.GetOrderbook(testUSDCPerpetual, asset.USDCMarginedFutures)
	require.NoError(t, err, "GetOrderbook must not error for the USDC perpetual")
	assert.Equal(t, orderbook.Levels{{Price: 82378.7, Amount: 2.4603}, {Price: 82377.3, Amount: 0.1938}, {Price: 82376.9, Amount: 0.2424}, {Price: 82369.6, Amount: 0.0426}, {Price: 82348, Amount: 0.1502}}, book.Bids, "the USDC perpetual's bids should be its snapshot's with its changes applied")
	assert.Equal(t, orderbook.Levels{
		{Price: 82378.8, Amount: 2.4621},
		{Price: 82378.9, Amount: 1.5009},
		{Price: 82379.5, Amount: 0.1863},
		{Price: 82379.7, Amount: 0.2053},
		{Price: 82380.3, Amount: 0.0002},
		{Price: 82380.4, Amount: 0.2583},
		{Price: 82380.8, Amount: 0.0019},
		{Price: 82381.1, Amount: 1.2141},
		{Price: 82381.3, Amount: 2.2102},
		{Price: 82382.6, Amount: 4.2367},
		{Price: 82382.9, Amount: 0.0047},
		{Price: 82383, Amount: 0.0737},
		{Price: 82383.1, Amount: 0.9392},
		{Price: 82383.2, Amount: 1.2158},
		{Price: 82387.3, Amount: 0.0431},
	}, book.Asks, "the USDC perpetual's asks should be its snapshot's with its changes applied")
	assert.Equal(t, int64(208589070917), book.LastUpdateID, "the USDC perpetual's book should hold its last change's ID")
	assert.Equal(t, time.UnixMilli(1792374089505), book.LastUpdated, "the USDC perpetual's book should be updated as its last change was sent")

	grouped, err := ex.Websocket.Orderbook.GetOrderbook(testCoinPerpetual, asset.CoinMarginedFutures)
	require.NoError(t, err, "GetOrderbook must not error for the coin margined perpetual")
	assert.Equal(t, orderbook.Levels{{Price: 82369.5, Amount: 6440}, {Price: 82368.5, Amount: 10}, {Price: 82368, Amount: 41650}, {Price: 82367.5, Amount: 69210}, {Price: 82367, Amount: 91510}}, grouped.Bids, "a grouped book's bids should be loaded whole")
	assert.Equal(t, orderbook.Levels{{Price: 82370, Amount: 61800}, {Price: 82370.5, Amount: 98190}, {Price: 82371, Amount: 97690}, {Price: 82371.5, Amount: 91310}, {Price: 82372, Amount: 39740}}, grouped.Asks, "a grouped book's asks should be loaded whole")

	var gotTrades [][]trade.Data
	var gotCandles []kline.Item
	relays := map[string]any{}
	for {
		var p any
		select {
		case payload := <-ex.Websocket.DataHandler.C:
			p = payload.Data
		default:
		}
		if p == nil {
			break
		}
		switch v := p.(type) {
		case []trade.Data:
			gotTrades = append(gotTrades, v)
		case kline.Item:
			gotCandles = append(gotCandles, v)
		case *DerivativesWebsocketQuote:
			relays["quote"] = v
		case *DerivativesWebsocketPerpetualInterest:
			relays["perpetual"] = v
		case *DerivativesWebsocketPublicMethodsState:
			relays["public methods state"] = v
		case *DerivativesWebsocketVolatilityIndex:
			relays["volatility index"] = v
		case *DerivativesWebsocketPriceStatistics:
			relays["price statistics"] = v
		case *[]DerivativesWebsocketPriceRanking:
			relays["price ranking"] = v
		case *DerivativesWebsocketPriceIndex:
			relays["price index"] = v
		case *[]DerivativesWebsocketMarkPriceOption:
			relays["mark price options"] = v
		case *DerivativesWebsocketEstimatedExpirationPrice:
			relays["estimated expiration price"] = v
		case *DerivativesWebsocketPlatformState:
			relays["platform state "+v.PriceIndex] = v
		}
	}
	sell := func(id string, p currency.Pair, a asset.Item, price, amount float64, timestamp int64) trade.Data {
		return trade.Data{TID: id, Exchange: ex.Name, CurrencyPair: p, AssetType: a, Side: order.Sell, Price: price, Amount: amount, Timestamp: time.UnixMilli(timestamp)}
	}
	expTrades := [][]trade.Data{
		{
			sell("453636740", testCoinPerpetual, asset.CoinMarginedFutures, 82369.5, 500, 1792374090927),
			sell("453636741", testCoinPerpetual, asset.CoinMarginedFutures, 82368.5, 10, 1792374090927),
		},
		{
			sell("USDC-80697112", testETHUSDCPerpetual, asset.USDCMarginedFutures, 2422.84, 0.3368, 1792374090937),
			sell("USDC-80697113", testETHUSDCPerpetual, asset.USDCMarginedFutures, 2422.84, 0.3368, 1792374090937),
			sell("USDC-80697127", testETHUSDCPerpetual, asset.USDCMarginedFutures, 2422.77, 1.465, 1792374091033),
		},
		{
			sell("453636742", testCoinPerpetual, asset.CoinMarginedFutures, 82365.5, 2940, 1792374094652),
			sell("453636743", testCoinPerpetual, asset.CoinMarginedFutures, 82365.5, 40, 1792374094652),
			sell("453636744", testCoinPerpetual, asset.CoinMarginedFutures, 82365.5, 70, 1792374094652),
		},
	}
	assert.Equal(t, expTrades, gotTrades, "the trades should be relayed with their taker's direction")
	assert.Equal(t, []kline.Item{{
		Exchange: ex.Name, Pair: testUSDCPerpetual, Asset: asset.USDCMarginedFutures, Interval: kline.OneMin,
		Candles: []kline.Candle{{Time: time.UnixMilli(1792374060000), Open: 82368.4, High: 82382.5, Low: 82368.4, Close: 82382.5, Volume: 5.6025}},
	}}, gotCandles, "the candle should be relayed with its channel's resolution")

	expRelays := map[string]any{
		"quote": &DerivativesWebsocketQuote{
			Timestamp: testMarketDataMilli(1792374088702), InstrumentName: "BTC_USDC-PERPETUAL", BestBidPrice: 82378.7, BestBidAmount: 2.014,
			BestAskPrice: 82378.8, BestAskAmount: 2.7001,
		},
		"perpetual":            &DerivativesWebsocketPerpetualInterest{Timestamp: testMarketDataMilli(1792374088707), IndexPrice: 82344.6, Interest: 5.1780566060084896e-05},
		"public methods state": &DerivativesWebsocketPublicMethodsState{AllowUnauthenticatedPublicRequests: true},
		"volatility index":     &DerivativesWebsocketVolatilityIndex{Timestamp: testMarketDataMilli(1792374087938), IndexName: "btc_usd", Volatility: 38.71},
		"price statistics":     &DerivativesWebsocketPriceStatistics{IndexName: "btc_usd", Change24Hour: 668.98, High24Hour: 82437.57, Low24Hour: 80602.44},
		"price ranking": &[]DerivativesWebsocketPriceRanking{
			{Identifier: "binance", Timestamp: testMarketDataMilli(1792374087597), OriginalPrice: 82336.05, Price: 82336.05},
			{Identifier: "bitfinex", Timestamp: testMarketDataMilli(1792374087683), OriginalPrice: 82440, Price: 82440},
			{Identifier: "coinbase", Enabled: true, Timestamp: testMarketDataMilli(1792374087000), OriginalPrice: 82350.39, Price: 82350.39, Weight: 20},
		},
		"price index": &DerivativesWebsocketPriceIndex{Timestamp: testMarketDataMilli(1792374087741), IndexName: "btc_usd", Price: 82344.6},
		"mark price options": &[]DerivativesWebsocketMarkPriceOption{
			{Timestamp: testMarketDataMilli(1792374088719), InstrumentName: "BTC-27NOV26-105000-P", MarkPrice: 0.2689, ImpliedVolatility: 0.3906},
			{Timestamp: testMarketDataMilli(1792374088719), InstrumentName: "BTC-26MAR27-75000-C", MarkPrice: 0.1607, ImpliedVolatility: 0.3876},
			{Timestamp: testMarketDataMilli(1792374088719), InstrumentName: "BTC-27NOV26-66000-C", MarkPrice: 0.2086, ImpliedVolatility: 0.4926},
		},
		"estimated expiration price":   &DerivativesWebsocketEstimatedExpirationPrice{Timestamp: testMarketDataMilli(1792374092012), Seconds: 22708, Price: 82343.41},
		"platform state 1000mog_usdc":  &DerivativesWebsocketPlatformState{PriceIndex: "1000mog_usdc", Reasons: json.RawMessage(`[]`)},
		"platform state 1000pepe_usdc": &DerivativesWebsocketPlatformState{PriceIndex: "1000pepe_usdc", Reasons: json.RawMessage(`[]`)},
	}
	assert.Equal(t, expRelays, relays, "every other notification should be relayed decoded")
}

func TestWebsocketHandleDerivativesOrderbookErrors(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	conn, err := ex.Websocket.CreateTestConnection(derivativesStreamsConnection)
	require.NoError(t, err, "CreateTestConnection must not error")
	snapshot := []byte(`{"jsonrpc":"2.0","method":"subscription","params":{"channel":"book.BTC-PERPETUAL.100ms","data":{"type":"snapshot","change_id":10,"instrument_name":"BTC-PERPETUAL","timestamp":1792374088507,"bids":[["new",100,1]],"asks":[["new",101,1]]}}}`)
	require.NoError(t, ex.wsHandleDerivatives(t.Context(), conn, snapshot), "a snapshot must load")

	// A change that does not follow the last one applied invalidates the book until resubscribing reloads it
	gap := []byte(`{"jsonrpc":"2.0","method":"subscription","params":{"channel":"book.BTC-PERPETUAL.100ms","data":{"type":"change","change_id":12,"prev_change_id":11,"instrument_name":"BTC-PERPETUAL","timestamp":1792374088600,"bids":[["change",100,2]],"asks":[]}}}`)
	assert.ErrorIs(t, ex.wsHandleDerivatives(t.Context(), conn, gap), errDerivativesOrderbookOutOfSequence, "a change out of sequence should error")
	_, err = ex.Websocket.Orderbook.GetOrderbook(testCoinPerpetual, asset.CoinMarginedFutures)
	assert.ErrorIs(t, err, orderbook.ErrOrderbookInvalid, "a book that missed a change should be invalidated")

	for name, tc := range map[string]struct {
		message string
		err     error
	}{
		"type":               {`{"method":"subscription","params":{"channel":"book.BTC-PERPETUAL.100ms","data":{"type":"replace","instrument_name":"BTC-PERPETUAL"}}}`, errDerivativesMalformedNotification},
		"instrument":         {`{"method":"subscription","params":{"channel":"book.BTC-26DEC26.100ms","data":{"type":"snapshot","instrument_name":"BTC-26DEC26"}}}`, errDerivativesUnknownInstrument},
		"grouped instrument": {`{"method":"subscription","params":{"channel":"book.BTC-26DEC26.none.10.100ms","data":{"instrument_name":"BTC-26DEC26"}}}`, errDerivativesUnknownInstrument},
		"data":               {`{"method":"subscription","params":{"channel":"book.BTC-PERPETUAL.100ms","data":[]}}`, errDerivativesMalformedNotification},
		"grouped data":       {`{"method":"subscription","params":{"channel":"book.BTC-PERPETUAL.none.10.100ms","data":[]}}`, errDerivativesMalformedNotification},
	} {
		assert.ErrorIsf(t, ex.wsHandleDerivatives(t.Context(), conn, []byte(tc.message)), tc.err, "wsHandleDerivatives should error for a book notification's %s", name)
	}
}

func TestWebsocketDerivativesResubscribe(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("the mock gateway websocket records the requests sent; live tests have none")
	}
	ex, venue := newDerivativesWebsocketTestExchange(t)
	conn, err := ex.Websocket.GetConnection(derivativesStreamsConnection)
	require.NoError(t, err, "GetConnection must not error")
	s := &subscription.Subscription{Channel: subscription.OrderbookChannel, QualifiedChannel: "book.BTC-PERPETUAL.100ms", Asset: asset.CoinMarginedFutures, Pairs: currency.Pairs{testCoinPerpetual}}
	require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(conn, s), "AddSuccessfulSubscriptions must not error")
	require.NoError(t, conn.Subscriptions().Add(s), "Add must not error")
	ex.wsDerivativesResubscribe(t.Context(), conn, "book.ETH-PERPETUAL.100ms")
	ex.wsDerivativesResubscribe(t.Context(), conn, s.QualifiedChannel)
	assert.Eventually(t, func() bool {
		return len(venue.requestsFor("public/unsubscribe")) == 1 && len(venue.requestsFor("public/subscribe")) == 1
	}, 30*time.Second, time.Millisecond, "wsDerivativesResubscribe should unsubscribe and subscribe the channel again")
}

func TestWebsocketHandleDerivativesIncrementalTickerErrors(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	conn, err := ex.Websocket.CreateTestConnection(derivativesStreamsConnection)
	require.NoError(t, err, "CreateTestConnection must not error")
	change := []byte(`{"method":"subscription","params":{"channel":"incremental_ticker.BTC-PERPETUAL","data":{"type":"change","instrument_name":"BTC-PERPETUAL","best_ask_amount":1}}}`)
	assert.ErrorIs(t, ex.wsHandleDerivatives(t.Context(), conn, change), errDerivativesNoTickerSnapshot, "a change before its snapshot should error")
	unknown := []byte(`{"method":"subscription","params":{"channel":"incremental_ticker.BTC-26DEC26","data":{"type":"snapshot"}}}`)
	assert.ErrorIs(t, ex.wsHandleDerivatives(t.Context(), conn, unknown), errDerivativesUnknownInstrument, "an unknown instrument's ticker should error")
	malformed := []byte(`{"method":"subscription","params":{"channel":"incremental_ticker.BTC-PERPETUAL","data":[]}}`)
	assert.ErrorIs(t, ex.wsHandleDerivatives(t.Context(), conn, malformed), errDerivativesMalformedNotification, "a malformed ticker should error")
	require.NoError(t, ex.wsHandleDerivatives(t.Context(), conn, []byte(`{"method":"subscription","params":{"channel":"incremental_ticker.BTC-PERPETUAL","data":{"type":"snapshot","instrument_name":"BTC-PERPETUAL","timestamp":1792374088707,"best_ask_amount":1}}}`)), "a snapshot must load")
	malformedChange := []byte(`{"method":"subscription","params":{"channel":"incremental_ticker.BTC-PERPETUAL","data":{"type":"change","best_ask_amount":"x"}}}`)
	assert.ErrorIs(t, ex.wsHandleDerivatives(t.Context(), conn, malformedChange), errDerivativesMalformedNotification, "a malformed change should error")
}

func TestWebsocketHandleDerivativesPublicErrors(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	conn, err := ex.Websocket.CreateTestConnection(derivativesStreamsConnection)
	require.NoError(t, err, "CreateTestConnection must not error")
	for name, tc := range map[string]struct {
		message string
		err     error
	}{
		"ticker data":       {`{"method":"subscription","params":{"channel":"ticker.BTC-PERPETUAL.100ms","data":[]}}`, errDerivativesMalformedNotification},
		"ticker instrument": {`{"method":"subscription","params":{"channel":"ticker.BTC-26DEC26.100ms","data":{"instrument_name":"BTC-26DEC26"}}}`, errDerivativesUnknownInstrument},
		"trades data":       {`{"method":"subscription","params":{"channel":"trades.BTC-PERPETUAL.100ms","data":{}}}`, errDerivativesMalformedNotification},
		"trades instrument": {`{"method":"subscription","params":{"channel":"trades.BTC-26DEC26.100ms","data":[{"instrument_name":"BTC-26DEC26","direction":"buy"}]}}`, errDerivativesUnknownInstrument},
		"trades direction":  {`{"method":"subscription","params":{"channel":"trades.BTC-PERPETUAL.100ms","data":[{"instrument_name":"BTC-PERPETUAL","direction":"hold"}]}}`, order.ErrSideIsInvalid},
		"candle channel":    {`{"method":"subscription","params":{"channel":"chart.trades.BTC-PERPETUAL","data":{}}}`, errDerivativesMalformedNotification},
		"candle resolution": {`{"method":"subscription","params":{"channel":"chart.trades.BTC-PERPETUAL.7","data":{}}}`, kline.ErrUnsupportedInterval},
		"candle instrument": {`{"method":"subscription","params":{"channel":"chart.trades.BTC-26DEC26.1","data":{}}}`, errDerivativesUnknownInstrument},
		"candle data":       {`{"method":"subscription","params":{"channel":"chart.trades.BTC-PERPETUAL.1","data":[]}}`, errDerivativesMalformedNotification},
		"relayed data":      {`{"method":"subscription","params":{"channel":"quote.BTC-PERPETUAL","data":[]}}`, errDerivativesMalformedNotification},
		"envelope":          {`[`, nil},
	} {
		err := ex.wsHandleDerivatives(t.Context(), conn, []byte(tc.message))
		if tc.err == nil {
			assert.Errorf(t, err, "wsHandleDerivatives should error for malformed %s", name)
			continue
		}
		assert.ErrorIsf(t, err, tc.err, "wsHandleDerivatives should error for a bad %s", name)
	}

	ex.Features.Enabled.TradeFeed = false
	assert.NoError(t, ex.wsHandleDerivatives(t.Context(), conn, []byte(`{"method":"subscription","params":{"channel":"trades.BTC-PERPETUAL.100ms","data":{}}}`)), "trades should be ignored when neither the trade feed nor trade saving is on")

	require.NoError(t, ex.wsHandleDerivatives(t.Context(), conn, []byte(`{"method":"subscription","params":{"channel":"nonsense","data":{}}}`)), "an unknown channel's notification must be relayed")
	require.NoError(t, ex.wsHandleDerivatives(t.Context(), conn, []byte(`{"method":"nonsense"}`)), "an unknown message must be relayed")
	assert.Len(t, relayed[websocket.UnhandledMessageWarning](ex), 2, "unknown messages should be relayed as unhandled")
}

func TestDerivativesQuoteVolume(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 2.5, derivativesQuoteVolume(1, 2.5), "derivativesQuoteVolume should prefer the notional volume")
	assert.Equal(t, 1.0, derivativesQuoteVolume(1, 0), "derivativesQuoteVolume should fall back to the USD volume")
}

func TestDerivativesBookLevels(t *testing.T) {
	t.Parallel()
	assert.Equal(t, orderbook.Levels{{Price: 1, Amount: 2}}, derivativesBookLevels([]DerivativesOrderBookLevel{{Price: 1, Amount: 2}, {Amount: 3}}), "derivativesBookLevels should leave out a level at a zero price")
	assert.Equal(t, orderbook.Levels{{Price: 1, Amount: 2}, {Price: 3}}, derivativesBookChangeLevels([]DerivativesWebsocketOrderbookChange{{Action: "new", Price: 1, Amount: 2}, {Action: derivativesBookDelete, Price: 3, Amount: 4}}), "derivativesBookChangeLevels should zero a deleted level's amount")
}

func TestDerivativesWebsocketOrderbookChangeUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var change DerivativesWebsocketOrderbookChange
	require.NoError(t, json.Unmarshal([]byte(`["change",82378.7,1.914]`), &change), "a book change must decode")
	assert.Equal(t, DerivativesWebsocketOrderbookChange{Action: "change", Price: 82378.7, Amount: 1.914}, change, "a book change should decode from its tuple")
	assert.ErrorIs(t, json.Unmarshal([]byte(`["change",1]`), &change), errInvalidTuple, "a book change with too few elements should error")
}

func TestDerivativesWebsocketInstrumentCreationUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var creation DerivativesWebsocketInstrumentCreation
	require.NoError(t, json.Unmarshal([]byte(`{"instrument_name":"BTC-27NOV26-60000-C","kind":"option","timestamp":1792374088702}`), &creation), "an instrument creation must decode")
	assert.Equal(t, "BTC-27NOV26-60000-C", creation.InstrumentName, "the instrument should decode")
	assert.Equal(t, testMarketDataMilli(1792374088702), creation.Timestamp, "the notification's timestamp should decode")
	assert.Error(t, json.Unmarshal([]byte(`[]`), &creation), "a malformed instrument should error")
	assert.Error(t, json.Unmarshal([]byte(`{"instrument_name":"x","timestamp":"y"}`), &creation), "a malformed timestamp should error")
}
