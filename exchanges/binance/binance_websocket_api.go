package binance

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/types"
)

const (
	binanceWebsocketAPIURL = "wss://ws-api.binance.com:443/ws-api/v3"
)

// websocket request status codes
var websocketStatusCodes = map[int64]string{
	400: "request failed",
	403: "request blocked",
	409: "request partially failed but also partially succeeded",
	418: "auto-banned for repeated violation of rate limits",
	419: "exceeded API request rate limit",
}

// WsConnectAPI creates a new websocket connection to API server
func (b *Binance) WsConnectAPI() error {
	if !b.Websocket.IsEnabled() || !b.IsEnabled() {
		return errors.New(stream.WebsocketNotEnabled)
	}

	var err error
	var dialer websocket.Dialer
	dialer.HandshakeTimeout = b.Config.HTTPTimeout
	dialer.Proxy = http.ProxyFromEnvironment

	b.Websocket.AuthConn.SetURL(binanceWebsocketAPIURL)
	err = b.Websocket.AuthConn.Dial(&dialer, http.Header{})
	if err != nil {
		return fmt.Errorf("%v - Unable to connect to Websocket. Error: %s", b.Name, err)
	}

	b.Websocket.AuthConn.SetupPingHandler(stream.PingHandler{
		UseGorillaHandler: true,
		MessageType:       websocket.PongMessage,
		Delay:             pingDelay,
	})

	b.Websocket.Wg.Add(1)
	go b.wsAPIReadData()
	return nil
}

// IsAPIStreamConnected checks if the API stream connection is established
func (b *Binance) IsAPIStreamConnected() bool {
	b.isAPIStreamConnectionLock.Lock()
	defer b.isAPIStreamConnectionLock.Unlock()
	return b.isAPIStreamConnected
}

// SetIsAPIStreamConnected sets a value of whether the API stream connection is established
func (b *Binance) SetIsAPIStreamConnected(isAPIStreamConnected bool) {
	b.isAPIStreamConnectionLock.Lock()
	defer b.isAPIStreamConnectionLock.Unlock()
	b.isAPIStreamConnected = isAPIStreamConnected
}

// wsAPIReadData receives and passes on websocket api messages for processing
func (b *Binance) wsAPIReadData() {
	defer b.Websocket.Wg.Done()

	for {
		resp := b.Websocket.AuthConn.ReadMessage()
		if resp.Raw == nil {
			return
		}
		err := b.wsHandleSpotAPIData(resp.Raw)
		if err != nil {
			b.Websocket.DataHandler <- err
		}
	}
}

// wsHandleSpotAPIData routes API response data.
func (b *Binance) wsHandleSpotAPIData(respRaw []byte) error {
	result := struct {
		Result json.RawMessage `json:"result"`
		ID     string          `json:"id"`
		Data   json.RawMessage `json:"data"`
	}{}
	err := json.Unmarshal(respRaw, &result)
	if err != nil {
		return err
	}
	if result.ID != "" {
		if !b.Websocket.Match.IncomingWithData(result.ID, respRaw) {
			return errors.New("Unhandled data: " + string(respRaw))
		}
		return nil
	}
	return fmt.Errorf("unhandled stream data %s", string(respRaw))
}

// GetWsOrderbook returns full orderbook information
//
// OrderBookDataRequestParams contains the following members
// symbol: string of currency pair
// limit: returned limit amount
func (b *Binance) GetWsOrderbook(obd *OrderBookDataRequestParams) (*OrderBook, error) {
	if obd == nil || *obd == (OrderBookDataRequestParams{}) {
		return nil, errNilArgument
	}
	if err := b.CheckLimit(obd.Limit); err != nil {
		return nil, err
	}
	var resp OrderBookData
	if err := b.SendWsRequest("depth", obd, &resp); err != nil {
		return nil, err
	}
	orderbook := OrderBook{
		Bids:         make([]OrderbookItem, len(resp.Bids)),
		Asks:         make([]OrderbookItem, len(resp.Asks)),
		LastUpdateID: resp.LastUpdateID,
	}
	for x := range resp.Bids {
		orderbook.Bids[x] = OrderbookItem{Price: resp.Bids[x][0].Float64(), Quantity: resp.Bids[x][1].Float64()}
	}
	for x := range resp.Asks {
		orderbook.Asks[x] = OrderbookItem{Price: resp.Asks[x][0].Float64(), Quantity: resp.Asks[x][1].Float64()}
	}
	return &orderbook, nil
}

// GetWsMostRecentTrades returns recent trade activity through the websocket connection
// limit: Up to 500 results returned
func (b *Binance) GetWsMostRecentTrades(rtr *RecentTradeRequestParams) ([]RecentTrade, error) {
	if rtr == nil || *rtr == (RecentTradeRequestParams{}) {
		return nil, errNilArgument
	}
	if rtr.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	var err error
	rtr.Symbol, err = b.FormatExchangeCurrency(rtr.Symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	var resp []RecentTrade
	return resp, b.SendWsRequest("trades.recent", rtr, &resp)
}

// GetWsAggregatedTrades retrieves aggregated trade activity.
func (b *Binance) GetWsAggregatedTrades(arg *WsAggregateTradeRequestParams) ([]AggregatedTrade, error) {
	if arg == nil || *arg == (WsAggregateTradeRequestParams{}) {
		return nil, errNilArgument
	}
	var resp []AggregatedTrade
	return resp, b.SendWsRequest("trades.aggregate", arg, &resp)
}

// GetWsCandlestick retrieves spot kline data through the websocket connection.
func (b *Binance) GetWsCandlestick(arg *KlinesRequestParams) ([]CandleStick, error) {
	return b.getWsKlines("klines", arg)
}

// GetWsOptimizedCandlestick retrieves spot candlestick bars through the websocket connection.
func (b *Binance) GetWsOptimizedCandlestick(arg *KlinesRequestParams) ([]CandleStick, error) {
	return b.getWsKlines("uiKlines", arg)
}

// getWsKlines retrieves spot kline data through the websocket connection.
func (b *Binance) getWsKlines(method string, arg *KlinesRequestParams) ([]CandleStick, error) {
	if arg == nil || *arg == (KlinesRequestParams{}) {
		return nil, nil
	}
	if arg.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if arg.Interval == "" {
		return nil, kline.ErrInvalidInterval
	}
	if !arg.StartTime.IsZero() {
		arg.StartTimestamp = arg.StartTime.UnixMilli()
	}
	if !arg.EndTime.IsZero() {
		arg.EndTimestamp = arg.EndTime.UnixMilli()
	}
	var resp [][]types.Number
	err := b.SendWsRequest(method, arg, &resp)
	if err != nil {
		return nil, err
	}

	klineData := make([]CandleStick, len(resp))
	for x := range resp {
		if len(resp[x]) != 12 {
			return nil, errors.New("unexpected kline data length")
		}
		klineData[x] = CandleStick{
			OpenTime:                 time.UnixMilli(resp[x][0].Int64()),
			Open:                     resp[x][1].Float64(),
			High:                     resp[x][2].Float64(),
			Low:                      resp[x][3].Float64(),
			Close:                    resp[x][4].Float64(),
			Volume:                   resp[x][5].Float64(),
			CloseTime:                time.UnixMilli(resp[x][6].Int64()),
			QuoteAssetVolume:         resp[x][7].Float64(),
			TradeCount:               resp[x][8].Float64(),
			TakerBuyAssetVolume:      resp[x][9].Float64(),
			TakerBuyQuoteAssetVolume: resp[x][10].Float64(),
		}
	}
	return klineData, nil
}

// GetWsCurrenctAveragePrice retrieves current average price for a symbol.
func (b *Binance) GetWsCurrenctAveragePrice(symbol currency.Pair) (*SymbolAveragePrice, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	arg := &struct {
		Symbol currency.Pair `json:"symbol"`
	}{
		Symbol: symbol,
	}
	var resp SymbolAveragePrice
	return &resp, b.SendWsRequest("avgPrice", arg, &resp)
}

// GetWs24HourPriceChange 24-hour rolling window price change statistics through the websocket stream.
func (b *Binance) GetWs24HourPriceChange(symbol currency.Pair, tickerType string) (*WsTickerPriceChange, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	arg := &struct {
		Symbol     currency.Pair `json:"symbol"`
		TickerType string        `json:"type,omitempty"`
	}{
		Symbol:     symbol,
		TickerType: tickerType,
	}
	var resp WsTickerPriceChange
	return &resp, b.SendWsRequest("ticker.24hr", arg, &resp)
}

// GetWs24HourPriceChanges 24-hour rolling window price changes statistics through the websocket stream.
// 'type': 'FULL' (default) or 'MINI'
// 'timeZone' Default: 0 (UTC)
func (b *Binance) GetWs24HourPriceChanges(symbols currency.Pairs, tickerType string) ([]WsTickerPriceChange, error) {
	if len(symbols) == 0 {
		return nil, currency.ErrCurrencyPairsEmpty
	}
	arg := &struct {
		Symbols    []string `json:"symbols"`
		TickerType string   `json:"type,omitempty"`
	}{
		Symbols:    symbols.Strings(),
		TickerType: tickerType,
	}
	var resp []WsTickerPriceChange
	return resp, b.SendWsRequest("ticker.24hr", arg, &resp)
}

// GetWsTradingDayTicker price change statistics for a trading day.
// 'type': 'FULL' (default) or 'MINI'
// 'timeZone' Default: 0 (UTC)
func (b *Binance) GetWsTradingDayTicker(symbol currency.Pair, timezone, tickerType string) (*WsTickerPriceChange, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	arg := &struct {
		Symbol     string `json:"symbol"`
		Timezone   string `json:"timeZone,omitempty"`
		TickerType string `json:"type,omitempty"`
	}{
		Symbol:     symbol.String(),
		Timezone:   timezone,
		TickerType: tickerType,
	}
	var resp WsTickerPriceChange
	return &resp, b.SendWsRequest("ticker.tradingDay", arg, &resp)
}

// GetWsTradingDayTickers price change statistics for a trading day.
// 'type': 'FULL' (default) or 'MINI'
// 'timeZone' Default: 0 (UTC)
func (b *Binance) GetWsTradingDayTickers(symbols currency.Pairs, timezone, tickerType string) ([]WsTickerPriceChange, error) {
	if len(symbols) == 0 {
		return nil, currency.ErrCurrencyPairsEmpty
	}
	arg := &struct {
		Symbols    []string `json:"symbols"`
		Timezone   string   `json:"timeZone,omitempty"`
		TickerType string   `json:"type,omitempty"`
	}{
		Symbols:    symbols.Strings(),
		Timezone:   timezone,
		TickerType: tickerType,
	}
	var resp []WsTickerPriceChange
	return resp, b.SendWsRequest("ticker.tradingDay", arg, &resp)
}

// GetRollingWindowPriceChangeStatistics retrieves rolling window price change statistics with a custom window.
// this request is similar to ticker.24hr, but statistics are computed on demand using the arbitrary window you specify
// func (b *Binance) GetRollingWindowPriceChangeStatistics()

// SendWsRequest sends websocket endpoint request through the websocket connection
func (b *Binance) SendWsRequest(method string, param, result interface{}) error {
	input := &struct {
		ID     string      `json:"id"`
		Method string      `json:"method"`
		Params interface{} `json:"params"`
	}{
		ID:     strconv.FormatInt(b.Websocket.AuthConn.GenerateMessageID(false), 10),
		Method: method,
		Params: param,
	}
	respRaw, err := b.Websocket.AuthConn.SendMessageReturnResponse(input.ID, input)
	if err != nil {
		return err
	}
	resp := &struct {
		ID     string      `json:"id"`
		Status int64       `json:"status"`
		Result interface{} `json:"result"`
	}{
		Result: result,
	}
	err = json.Unmarshal(respRaw, &resp)
	if err != nil {
		return err
	}
	switch resp.Status {
	case 200:
		return nil
	case 400, 403, 409, 418, 419:
		return errors.New(websocketStatusCodes[resp.Status])
	default:
		if resp.Status >= 500 {
			return errors.New("internal server error")
		}
		return errors.New("request failed")
	}
}
