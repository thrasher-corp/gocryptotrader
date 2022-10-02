package gateio

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
)

const (
	futuresWebsocket_btc_url  = "wss://fx-ws.gateio.ws/v4/ws/btc"
	futuresWebsocket_usdt_url = "wss://fx-ws.gateio.ws/v4/ws/usdt"

	futuresPingChannel            = "futures.ping"
	futuresTickersChannel         = "futures.tickers"
	futuresTradesChannel          = "futures.trades"
	futuresOrderbookChannel       = "futures.order_book"
	futuresOrderbookTickerChannel = "futures.book_ticker"
	futuresOrderbookUpdateChannel = "futures.order_book_update"
	futuresCandlesticksChannel    = "futures.candlesticks"
	futuresOrdersChannel          = "futures.orders"

	//  authenticated channels
	futuresUserTradesChannel        = "futures.usertrades"
	futuresLiquidatesChannel        = "futures.liquidates"
	futuresAutoDeleveragesChannel   = "futures.auto_deleverages"
	futuresAutoPositionCloseChannel = "futures.position_closes"
	futuresBalancesChannel          = "futures.balances"
	futuresReduceRiskLimitsChannel  = "futures.reduce_risk_limits"
	futuresPositionsChannel         = "futures.positions"
	futuresAutoOrdersChannel        = "futures.autoorders"
)

var defaultFuturesSubscriptions = []string{
	futuresTickersChannel,
	futuresTradesChannel,
	futuresOrderbookChannel,
	futuresOrderbookUpdateChannel,
	futuresCandlesticksChannel,
	futuresOrdersChannel,
}

// WsFuturesConnect initiates a websocket connection
func (g *Gateio) WsFuturesConnect() error {
	if !g.Websocket.IsEnabled() || !g.IsEnabled() {
		return errors.New(stream.WebsocketNotEnabled)
	}
	var dialer websocket.Dialer
	err := g.Websocket.SetWebsocketURL(futuresWebsocket_btc_url, false, true)
	if err != nil {
		return err
	}
	err = g.Websocket.SetWebsocketURL(futuresWebsocket_usdt_url, true, true)
	if err != nil {
		return err
	}
	err = g.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}
	pingMessage, err := json.Marshal(WsInput{
		ID: g.Websocket.Conn.GenerateMessageID(false),
		Time: func() int64 {
			return time.Now().Unix()
		}(),
		Channel: futuresPingChannel,
	})
	if err != nil {
		return err
	}
	g.Websocket.Conn.SetupPingHandler(stream.PingHandler{
		Websocket: true,
		Delay:     time.Second * 15,
		Message:   pingMessage,
	})
	g.Websocket.Wg.Add(1)
	go g.wsReadData()
	go g.WsChannelsMultiplexer.Run()
	return nil
}

// GenerateDefaultFuturesSubscriptions returns default subscriptions informations.
func (g *Gateio) GenerateDefaultFuturesSubscriptions() ([]stream.ChannelSubscription, error) {
	if g.Websocket.CanUseAuthenticatedEndpoints() {
		defaultSubscriptions = append(defaultFuturesSubscriptions,
			futuresUserTradesChannel,
			futuresBalancesChannel)
	}
	var subscriptions []stream.ChannelSubscription
	var pairs []currency.Pair
	var err error
	pairs, err = g.GetEnabledPairs(asset.Futures)
	if err != nil {
		return nil, err
	}
	for i := range defaultFuturesSubscriptions {
		for j := range pairs {
			params := make(map[string]interface{})
			if strings.EqualFold(defaultFuturesSubscriptions[i], futuresOrderbookChannel) {
				params["limit"] = 100
				params["interval"] = "0"
			} else if strings.EqualFold(defaultFuturesSubscriptions[i], futuresCandlesticksChannel) {
				params["interval"] = kline.FiveMin
			} else if strings.EqualFold(defaultFuturesSubscriptions[i], futuresOrderbookUpdateChannel) {
				params["frequency"] = kline.ThousandMilliseconds
				params["level"] = "100"
			}
			fpair, err := g.FormatExchangeCurrency(pairs[j], asset.Futures)
			if err != nil {
				return nil, err
			}
			subscriptions = append(subscriptions, stream.ChannelSubscription{
				Channel:  defaultFuturesSubscriptions[i],
				Currency: fpair.Upper(),
				Params:   params,
			})
		}
	}
	return subscriptions, nil
}

func (g *Gateio) processFuturesTickers(data []byte) error {
	type response struct {
		Time    int64            `json:"time"`
		Channel string           `json:"channel"`
		Event   string           `json:"event"`
		Result  []WsFutureTicker `json:"result"`
	}
	var resp response
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	for x := range resp.Result {
		currencyPair, err := currency.NewPairFromString(resp.Result[x].Contract)
		if err != nil {
			return err
		}
		g.Websocket.DataHandler <- &ticker.Price{
			ExchangeName: g.Name,
			Volume:       resp.Result[x].Volume24HBase,
			QuoteVolume:  resp.Result[x].Volume24HQuote,
			High:         resp.Result[x].High24H,
			Low:          resp.Result[x].Low24H,
			Last:         resp.Result[x].Last,
			AssetType:    asset.Futures,
			Pair:         currencyPair,
			LastUpdated:  time.Unix(resp.Time, 0),
		}
	}
	return nil
}

func (g *Gateio) processFuturesTrades(data []byte) error {
	type response struct {
		Time    int64             `json:"time"`
		Channel string            `json:"channel"`
		Event   string            `json:"event"`
		Result  []WsFuturesTrades `json:"result"`
	}
	var resp response
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	trades := make([]trade.Data, len(resp.Result))
	for x := range resp.Result {
		currencyPair, err := currency.NewPairFromString(resp.Result[x].Contract)
		if err != nil {
			return err
		}
		trades[x] = trade.Data{
			Timestamp:    time.UnixMilli(int64(resp.Result[x].CreateTimeMs)),
			CurrencyPair: currencyPair,
			AssetType:    asset.Futures,
			Exchange:     g.Name,
			Price:        resp.Result[x].Price,
			Amount:       resp.Result[x].Size,
			TID:          strconv.FormatInt(resp.Result[x].ID, 10),
		}
	}
	return trade.AddTradesToBuffer(g.Name, trades...)
}

func (g *Gateio) processFuturesCandlesticks(data []byte) error {
	type response struct {
		Time    int64                `json:"time"`
		Channel string               `json:"channel"`
		Event   string               `json:"event"`
		Result  []FuturesCandlestick `json:"result"`
	}
	var resp response
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	for x := range resp.Result {
		icp := strings.Split(resp.Result[x].Name, currency.UnderscoreDelimiter)
		if len(icp) < 3 {
			return errors.New("malformed futures candlestick websocket push data")
		}
		currencyPair, err := currency.NewPairFromString(strings.Join(icp[1:], currency.UnderscoreDelimiter))
		if err != nil {
			return err
		}
		g.Websocket.DataHandler <- stream.KlineData{
			Pair:       currencyPair,
			AssetType:  asset.Futures,
			Exchange:   g.Name,
			StartTime:  resp.Result[x].Timestamp,
			Interval:   icp[0],
			OpenPrice:  resp.Result[x].OpenPrice,
			ClosePrice: resp.Result[x].ClosePrice,
			HighPrice:  resp.Result[x].HighestPrice,
			LowPrice:   resp.Result[x].LowestPrice,
			Volume:     resp.Result[x].Volume,
		}
	}
	return nil
}

func (g *Gateio) processFuturesOrderbookTicker(data []byte) error {
	var response WsResponse
	orderbookTicker := &WsFuturesOrderbookTicker{}
	response.Result = orderbookTicker
	err := json.Unmarshal(data, &response)
	if err != nil {
		return err
	}
	g.Websocket.DataHandler <- response
	return nil
}

func (g *Gateio) procesFuturesOrderbookUpdate(data []byte) error {
	var response WsResponse
	update := &WsFuturesOrderbookUpdate{}
	response.Result = update
	err := json.Unmarshal(data, &response)
	if err != nil {
		return err
	}
	pair, err := currency.NewPairFromString(update.ContractName)
	if err != nil {
		return err
	}
	bids := make([]orderbook.Item, len(update.Bids))
	asks := make([]orderbook.Item, len(update.Asks))
	for x := range asks {
		asks[x] = orderbook.Item{
			Amount: update.Asks[x].Size,
			Price:  update.Asks[x].Price,
		}
	}
	for x := range bids {
		bids[x] = orderbook.Item{
			Amount: update.Bids[x].Size,
			Price:  update.Bids[x].Price,
		}
	}
	if len(asks) == 0 && len(bids) == 0 {
		return nil
	}
	return g.Websocket.Orderbook.Update(&orderbook.Update{
		UpdateTime: time.UnixMilli(update.TimestampInMs),
		Asks:       asks,
		Bids:       bids,
		Pair:       pair,
		Asset:      asset.Futures,
		MaxDepth:   int(update.LastUpdatedID - update.FirstUpdatedID),
	})
}

func (g *Gateio) processFuturesOrderbookSnapshot(data []byte) error {
	var response WsResponse
	snapshot := &WsFuturesOrderbookSnapshot{}
	response.Result = snapshot
	err := json.Unmarshal(data, &response)
	if err != nil {
		return err
	}
	pair, err := currency.NewPairFromString(snapshot.Contract)
	if err != nil {
		return err
	}
	bids := make([]orderbook.Item, len(snapshot.Bids))
	asks := make([]orderbook.Item, len(snapshot.Asks))
	for x := range asks {
		asks[x] = orderbook.Item{
			Amount: snapshot.Asks[x].Size,
			Price:  snapshot.Asks[x].Price,
		}
	}
	for x := range bids {
		bids[x] = orderbook.Item{
			Amount: snapshot.Bids[x].Size,
			Price:  snapshot.Bids[x].Price,
		}
	}
	return g.Websocket.Orderbook.LoadSnapshot(&orderbook.Base{
		Asks:            asks,
		Bids:            bids,
		Asset:           asset.Futures,
		Exchange:        g.Name,
		Pair:            pair,
		LastUpdated:     time.UnixMilli(snapshot.TimestampInMs),
		VerifyOrderbook: g.CanVerifyOrderbook,
	})
}
