package gateio

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
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
	if g.Websocket.CanUseAuthenticatedEndpoints() {
		g.wsServerSignIn(context.Background())
	}
	go g.WsChannelsMultiplexer.Run()
	return nil
}

// GenerateDefaultFuturesSubscriptions returns default subscriptions informations.
func (g *Gateio) GenerateDefaultFuturesSubscriptions() ([]stream.ChannelSubscription, error) {
	if g.Websocket.CanUseAuthenticatedEndpoints() {
		defaultSubscriptions = append(defaultFuturesSubscriptions,
			futuresOrdersChannel,
			futuresUserTradesChannel,
			futuresBalancesChannel,
		)
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
		// Interval_Currency-Pair
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

func (g *Gateio) procesFuturesAndOptionsOrderbookUpdate(data []byte) error {
	var response WsResponse
	update := &WsFuturesAndOptionsOrderbookUpdate{}
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
		return errors.New("malformed orderbook data")
	}
	var assetType asset.Item
	if response.Channel == optionsOrderbookUpdateChannel {
		assetType = asset.Options
	} else {
		assetType = asset.Futures
	}
	return g.Websocket.Orderbook.Update(&orderbook.Update{
		UpdateTime: time.UnixMilli(update.TimestampInMs),
		Asks:       asks,
		Bids:       bids,
		Pair:       pair,
		Asset:      assetType,
		MaxDepth:   int(update.LastUpdatedID - update.FirstUpdatedID),
	})
}

func (g *Gateio) processFuturesOrderbookSnapshot(event string, data []byte) error {
	if event == "all" {
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
	type response struct {
		Time    int64                           `json:"time"`
		Channel string                          `json:"channel"`
		Event   string                          `json:"event"`
		Result  []WsFuturesOrderbookUpdateEvent `json:"result"`
	}
	var resp response
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	dataMap := map[string][2][]orderbook.Item{}
	for x := range resp.Result {
		ab, ok := dataMap[resp.Result[x].CurrencyPair]
		if !ok {
			ab = [2][]orderbook.Item{}
		}
		if resp.Result[x].Amount > 0 {
			ab[1] = append(ab[1], orderbook.Item{
				Price:  resp.Result[x].Price,
				Amount: resp.Result[x].Amount,
			})
		} else {
			ab[0] = append(ab[0], orderbook.Item{
				Price:  resp.Result[x].Price,
				Amount: -resp.Result[x].Amount,
			})
		}
		if !ok {
			dataMap[resp.Result[x].CurrencyPair] = ab
		}
	}
	if len(dataMap) == 0 {
		return errors.New("missing orderbook ask and bid data")
	}
	for key, ab := range dataMap {
		currencyPair, err := currency.NewPairFromString(key)
		if err != nil {
			return err
		}
		err = g.Websocket.Orderbook.LoadSnapshot(&orderbook.Base{
			Asks:            ab[0],
			Bids:            ab[1],
			Asset:           asset.Futures,
			Exchange:        g.Name,
			Pair:            currencyPair,
			LastUpdated:     time.Unix(resp.Time, 0),
			VerifyOrderbook: g.CanVerifyOrderbook,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (g *Gateio) processFuturesOrdersPushData(data []byte) error {
	type response struct {
		Time    int64            `json:"time"`
		Channel string           `json:"channel"`
		Event   string           `json:"event"`
		Result  []WsFuturesOrder `json:"result"`
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
		status, err := order.StringToOrderStatus(func() string {
			if resp.Result[x].Status == "finished" {
				return "cancelled"
			}
			return resp.Result[x].Status
		}())
		if err != nil {
			return err
		}
		g.Websocket.DataHandler <- &order.Detail{
			Amount:         resp.Result[x].Size,
			Exchange:       g.Name,
			OrderID:        strconv.FormatInt(resp.Result[x].ID, 10),
			Status:         status,
			Pair:           currencyPair,
			LastUpdated:    time.UnixMilli(resp.Result[x].FinishTimeMs),
			Date:           time.UnixMilli(resp.Result[x].CreateTimeMs),
			ExecutedAmount: resp.Result[x].Size - resp.Result[x].Left,
			Price:          resp.Result[x].Price,
			AssetType:      asset.Futures,
			AccountID:      resp.Result[x].User,
			CloseTime:      time.UnixMilli(resp.Result[x].FinishTimeMs),
		}
	}
	return nil
}

func (g *Gateio) procesFuturesUserTrades(data []byte) error {
	type response struct {
		Time    int64                `json:"time"`
		Channel string               `json:"channel"`
		Event   string               `json:"event"`
		Result  []WsFuturesUserTrade `json:"result"`
	}
	var resp response
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	fills := make([]fill.Data, len(resp.Result))
	for x := range resp.Result {
		currencyPair, err := currency.NewPairFromString(resp.Result[x].Contract)
		if err != nil {
			return err
		}
		fills[x] = fill.Data{
			Timestamp:    time.UnixMilli(resp.Result[x].CreateTimeMs),
			Exchange:     g.Name,
			CurrencyPair: currencyPair,
			OrderID:      resp.Result[x].OrderID,
			TradeID:      resp.Result[x].ID,
			Price:        resp.Result[x].Price,
			Amount:       resp.Result[x].Size,
		}
	}
	return g.Websocket.Fills.Update(fills...)
}

func (g *Gateio) processFuturesLiquidatesNotification(data []byte) error {
	type response struct {
		Time    int64                               `json:"time"`
		Channel string                              `json:"channel"`
		Event   string                              `json:"event"`
		Result  []WsFuturesLiquidiationNotification `json:"result"`
	}
	var resp response
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	g.Websocket.DataHandler <- &resp
	return nil
}

func (g *Gateio) processFuturesAutoDeleveragesNotification(data []byte) error {
	type response struct {
		Time    int64                                  `json:"time"`
		Channel string                                 `json:"channel"`
		Event   string                                 `json:"event"`
		Result  []WsFuturesAutoDeleveragesNotification `json:"result"`
	}
	var resp response
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	g.Websocket.DataHandler <- &resp
	return nil
}

func (g *Gateio) processPositionCloseData(data []byte) error {
	type response struct {
		Time    int64             `json:"time"`
		Channel string            `json:"channel"`
		Event   string            `json:"event"`
		Result  []WsPositionClose `json:"result"`
	}
	var resp response
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	g.Websocket.DataHandler <- &resp
	return nil
}

func (g *Gateio) processBalancePushData(data []byte) error {
	type response struct {
		Time    int64       `json:"time"`
		Channel string      `json:"channel"`
		Event   string      `json:"event"`
		Result  []WsBalance `json:"result"`
	}
	var resp response
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	for x := range resp.Result {
		info := strings.Split(resp.Result[x].Text, currency.UnderscoreDelimiter)
		if len(info) != 2 {
			return errors.New("malformed text")
		}
		code := currency.NewCode(info[0])
		g.Websocket.DataHandler <- account.Change{
			Exchange: g.Name,
			Currency: code,
			Asset:    asset.Futures,
			Amount:   resp.Result[x].Balance,
		}
	}
	return nil
}

func (g *Gateio) processFuturesReduceRiskLimitNotification(data []byte) error {
	type response struct {
		Time    int64                                  `json:"time"`
		Channel string                                 `json:"channel"`
		Event   string                                 `json:"event"`
		Result  []WsFuturesReduceRiskLimitNotification `json:"result"`
	}
	var resp response
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	g.Websocket.DataHandler <- &resp
	return nil
}

func (g *Gateio) processFuturesPositionsNotification(data []byte) error {
	type response struct {
		Time    int64               `json:"time"`
		Channel string              `json:"channel"`
		Event   string              `json:"event"`
		Result  []WsFuturesPosition `json:"result"`
	}
	var resp response
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	g.Websocket.DataHandler <- &resp
	return nil
}

func (g *Gateio) processFuturesAutoOrderPushData(data []byte) error {
	type response struct {
		Time    int64                `json:"time"`
		Channel string               `json:"channel"`
		Event   string               `json:"event"`
		Result  []WsFuturesAutoOrder `json:"result"`
	}
	var resp response
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	g.Websocket.DataHandler <- &resp
	return nil
}
