package gateio

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
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
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	futuresWebsocketBtcURL  = "wss://fx-ws.gateio.ws/v4/ws/btc"
	futuresWebsocketUsdtURL = "wss://fx-ws.gateio.ws/v4/ws/usdt"

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

// responseStream a channel thought which the data coming from the two websocket connection will go through.
var responseStream = make(chan stream.Response)

var futuresAssetType = asset.Futures

// WsFuturesConnect initiates a websocket connection for futures account
func (g *Gateio) WsFuturesConnect() error {
	if !g.Websocket.IsEnabled() || !g.IsEnabled() {
		return errors.New(stream.WebsocketNotEnabled)
	}
	var dialer websocket.Dialer
	err := g.Websocket.SetWebsocketURL(futuresWebsocketUsdtURL, false, true)
	if err != nil {
		return err
	}
	err = g.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}

	err = g.Websocket.SetupNewConnection(stream.ConnectionSetup{
		URL:                  futuresWebsocketBtcURL,
		RateLimit:            gateioWebsocketRateLimit,
		ResponseCheckTimeout: g.Config.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     g.Config.WebsocketResponseMaxLimit,
		Authenticated:        true,
	})
	if err != nil {
		return err
	}
	err = g.Websocket.AuthConn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}
	g.Websocket.Wg.Add(3)
	go g.wsFunnelConnectionData(g.Websocket.Conn)
	go g.wsFunnelConnectionData(g.Websocket.AuthConn)
	go g.wsReadData()
	if g.Verbose {
		log.Debugf(log.ExchangeSys, "Successful connection to %v\n",
			g.Websocket.GetWebsocketURL())
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
		Websocket:   true,
		MessageType: websocket.PingMessage,
		Delay:       time.Second * 15,
		Message:     pingMessage,
	})
	g.Websocket.Wg.Add(1)
	go g.wsReadData()
	go g.WsChannelsMultiplexer.Run()
	return nil
}

// GenerateFuturesDefaultSubscriptions returns default subscriptions informations.
func (g *Gateio) GenerateFuturesDefaultSubscriptions() ([]stream.ChannelSubscription, error) {
	channelsToSubscribe := defaultFuturesSubscriptions
	if g.Websocket.CanUseAuthenticatedEndpoints() {
		channelsToSubscribe = append(channelsToSubscribe,
			futuresOrdersChannel,
			futuresUserTradesChannel,
			futuresBalancesChannel,
		)
	}
	pairs, err := g.GetEnabledPairs(futuresAssetType)
	if err != nil {
		return nil, err
	}
	subscriptions := make([]stream.ChannelSubscription, len(channelsToSubscribe)*len(pairs))
	count := 0
	for i := range channelsToSubscribe {
		for j := range pairs {
			params := make(map[string]interface{})
			switch channelsToSubscribe[i] {
			case futuresOrderbookChannel:
				params["limit"] = 100
				params["interval"] = "0"
			case futuresCandlesticksChannel:
				params["interval"] = kline.FiveMin
			case futuresOrderbookUpdateChannel:
				params["frequency"] = kline.ThousandMilliseconds
				params["level"] = "100"
			}
			fpair, err := g.FormatExchangeCurrency(pairs[j], futuresAssetType)
			if err != nil {
				return nil, err
			}
			subscriptions[count] = stream.ChannelSubscription{
				Channel:  channelsToSubscribe[i],
				Currency: fpair.Upper(),
				Params:   params,
			}
			count++
		}
	}
	return subscriptions, nil
}

// FuturesSubscribe sends a websocket message to stop receiving data from the channel
func (g *Gateio) FuturesSubscribe(channelsToUnsubscribe []stream.ChannelSubscription) error {
	return g.handleFuturesSubscription("subscribe", channelsToUnsubscribe)
}

// FuturesUnsubscribe sends a websocket message to stop receiving data from the channel
func (g *Gateio) FuturesUnsubscribe(channelsToUnsubscribe []stream.ChannelSubscription) error {
	return g.handleFuturesSubscription("unsubscribe", channelsToUnsubscribe)
}

// handleFuturesSubscription sends a websocket message to receive data from the channel
func (g *Gateio) handleFuturesSubscription(event string, channelsToSubscribe []stream.ChannelSubscription) error {
	payloads, err := g.generateFuturesPayload(event, channelsToSubscribe)
	if err != nil {
		return err
	}
	var errs common.Errors
	for con := range payloads {
		for k := range payloads[con] {
			if con == 0 {
				err = g.Websocket.Conn.SendJSONMessage(payloads[con][k])
			} else {
				err = g.Websocket.AuthConn.SendJSONMessage(payloads[con][k])
			}
			if err != nil {
				errs = append(errs, err)
				continue
			}
			channel := make(chan *WsEventResponse)
			g.WsChannelsMultiplexer.Register <- &wsChanReg{
				ID:   strconv.FormatInt(payloads[con][k].ID, 10),
				Chan: channel,
			}
			ticker := time.NewTicker(time.Second * 3)
		receive:
			for {
				select {
				case resp := <-channel:
					if resp.Result != nil && resp.Result.Status != "success" {
						errs = append(errs, fmt.Errorf("%s websocket connection: timeout waiting for response with and subscription: %v", g.Name, payloads[con][k].Channel))
						break receive
					} else if resp.Error != nil && resp.Error.Code != 0 {
						errs = append(errs, fmt.Errorf("error while %s to channel %s error code: %d message: %s", payloads[con][k].Event, payloads[con][k].Channel, resp.Error.Code, resp.Error.Message))
						break receive
					}
					g.Websocket.AddSuccessfulSubscriptions(channelsToSubscribe[k])
					g.WsChannelsMultiplexer.Unregister <- strconv.FormatInt(payloads[con][k].ID, 10)
					break receive
				case <-ticker.C:
					ticker.Stop()
					errs = append(errs, fmt.Errorf("%s websocket connection: timeout waiting for response with and subscription: %v",
						g.Name, payloads[con][k].Channel))
					g.WsChannelsMultiplexer.Unregister <- strconv.FormatInt(payloads[con][k].ID, 10)
				}
			}
		}
	}
	if errs != nil {
		return errs
	}
	return nil
}

func (g *Gateio) generateFuturesPayload(event string, channelsToSubscribe []stream.ChannelSubscription) ([2][]WsInput, error) {
	if len(channelsToSubscribe) == 0 {
		return [2][]WsInput{}, errors.New("cannot generate payload, no channels supplied")
	}
	var creds *account.Credentials
	var err error
	if g.Websocket.CanUseAuthenticatedEndpoints() {
		creds, err = g.GetCredentials(context.TODO())
		if err != nil {
			g.Websocket.SetCanUseAuthenticatedEndpoints(false)
		}
	}
	payloads := [2][]WsInput{}
	for i := range channelsToSubscribe {
		var auth *WsAuthInput
		timestamp := time.Now()
		var params []string
		params = []string{channelsToSubscribe[i].Currency.String()}
		if g.Websocket.CanUseAuthenticatedEndpoints() && (channelsToSubscribe[i].Channel == futuresOrdersChannel ||
			channelsToSubscribe[i].Channel == futuresUserTradesChannel ||
			channelsToSubscribe[i].Channel == futuresLiquidatesChannel ||
			channelsToSubscribe[i].Channel == futuresAutoDeleveragesChannel ||
			channelsToSubscribe[i].Channel == futuresAutoPositionCloseChannel ||
			channelsToSubscribe[i].Channel == futuresBalancesChannel ||
			channelsToSubscribe[i].Channel == futuresReduceRiskLimitsChannel ||
			channelsToSubscribe[i].Channel == futuresPositionsChannel ||
			channelsToSubscribe[i].Channel == futuresAutoOrdersChannel) {
			value, ok := channelsToSubscribe[i].Params["user"].(string)
			if ok {
				params = append(
					[]string{value},
					params...)
			}
			sigTemp, err := g.generateWsSignature(creds.Secret, event, channelsToSubscribe[i].Channel, timestamp)
			if err != nil {
				return [2][]WsInput{}, err
			}
			auth = &WsAuthInput{
				Method: "api_key",
				Key:    creds.Key,
				Sign:   sigTemp,
			}
		}
		if strings.HasPrefix(channelsToSubscribe[i].Currency.Quote.Upper().String(), "USDT") {
			payloads[0] = append(payloads[0], WsInput{
				ID:      g.Websocket.Conn.GenerateMessageID(false),
				Event:   event,
				Channel: channelsToSubscribe[i].Channel,
				Payload: params,
				Auth:    auth,
				Time:    timestamp.Unix(),
			})
		} else {
			payloads[1] = append(payloads[1], WsInput{
				ID:      g.Websocket.Conn.GenerateMessageID(false),
				Event:   event,
				Channel: channelsToSubscribe[i].Channel,
				Payload: params,
				Auth:    auth,
				Time:    timestamp.Unix(),
			})
		}
	}
	return payloads, nil
}

func (g *Gateio) processFuturesTickers(data []byte) error {
	resp := struct {
		Time    int64            `json:"time"`
		Channel string           `json:"channel"`
		Event   string           `json:"event"`
		Result  []WsFutureTicker `json:"result"`
	}{}
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	tickerPriceDatas := make([]ticker.Price, len(resp.Result))
	for x := range resp.Result {
		currencyPair, err := currency.NewPairFromString(resp.Result[x].Contract)
		if err != nil {
			return err
		}
		tickerPriceDatas[x] = ticker.Price{
			ExchangeName: g.Name,
			Volume:       resp.Result[x].Volume24HBase,
			QuoteVolume:  resp.Result[x].Volume24HQuote,
			High:         resp.Result[x].High24H,
			Low:          resp.Result[x].Low24H,
			Last:         resp.Result[x].Last,
			AssetType:    futuresAssetType,
			Pair:         currencyPair,
			LastUpdated:  time.Unix(resp.Time, 0),
		}
	}
	g.Websocket.DataHandler <- tickerPriceDatas
	return nil
}

func (g *Gateio) processFuturesTrades(data []byte) error {
	resp := struct {
		Time    int64             `json:"time"`
		Channel string            `json:"channel"`
		Event   string            `json:"event"`
		Result  []WsFuturesTrades `json:"result"`
	}{}
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
			AssetType:    futuresAssetType,
			Exchange:     g.Name,
			Price:        resp.Result[x].Price,
			Amount:       resp.Result[x].Size,
			TID:          strconv.FormatInt(resp.Result[x].ID, 10),
		}
	}
	return trade.AddTradesToBuffer(g.Name, trades...)
}

func (g *Gateio) processFuturesCandlesticks(data []byte) error {
	resp := struct {
		Time    int64                `json:"time"`
		Channel string               `json:"channel"`
		Event   string               `json:"event"`
		Result  []FuturesCandlestick `json:"result"`
	}{}
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	klineDatas := make([]stream.KlineData, len(resp.Result))
	for x := range resp.Result {
		icp := strings.Split(resp.Result[x].Name, currency.UnderscoreDelimiter)
		if len(icp) < 3 {
			return errors.New("malformed futures candlestick websocket push data")
		}
		currencyPair, err := currency.NewPairFromString(strings.Join(icp[1:], currency.UnderscoreDelimiter))
		if err != nil {
			return err
		}
		klineDatas[x] = stream.KlineData{
			Pair:       currencyPair,
			AssetType:  futuresAssetType,
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
	g.Websocket.DataHandler <- klineDatas
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
	var assetType asset.Item
	if response.Channel == optionsOrderbookUpdateChannel {
		assetType = asset.Options
	} else {
		assetType = futuresAssetType
	}
	updates := orderbook.Update{
		UpdateTime: time.UnixMilli(update.TimestampInMs),
		Pair:       pair,
		Asset:      assetType,
	}
	updates.Bids = make([]orderbook.Item, len(update.Bids))
	updates.Asks = make([]orderbook.Item, len(update.Asks))
	for x := range updates.Asks {
		updates.Asks[x] = orderbook.Item{
			Amount: update.Asks[x].Size,
			Price:  update.Asks[x].Price,
		}
	}
	for x := range updates.Bids {
		updates.Bids[x] = orderbook.Item{
			Amount: update.Bids[x].Size,
			Price:  update.Bids[x].Price,
		}
	}
	if len(updates.Asks) == 0 && len(updates.Bids) == 0 {
		return errors.New("malformed orderbook data")
	}
	return g.Websocket.Orderbook.Update(&updates)
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
		base := orderbook.Base{
			Asset:           futuresAssetType,
			Exchange:        g.Name,
			Pair:            pair,
			LastUpdated:     time.UnixMilli(snapshot.TimestampInMs),
			VerifyOrderbook: g.CanVerifyOrderbook,
		}
		base.Bids = make([]orderbook.Item, len(snapshot.Bids))
		base.Asks = make([]orderbook.Item, len(snapshot.Asks))
		for x := range base.Asks {
			base.Asks[x] = orderbook.Item{
				Amount: snapshot.Asks[x].Size,
				Price:  snapshot.Asks[x].Price,
			}
		}
		for x := range base.Bids {
			base.Bids[x] = orderbook.Item{
				Amount: snapshot.Bids[x].Size,
				Price:  snapshot.Bids[x].Price,
			}
		}
		return g.Websocket.Orderbook.LoadSnapshot(&base)
	}
	resp := struct {
		Time    int64                           `json:"time"`
		Channel string                          `json:"channel"`
		Event   string                          `json:"event"`
		Result  []WsFuturesOrderbookUpdateEvent `json:"result"`
	}{}
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
			Asset:           futuresAssetType,
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
	resp := struct {
		Time    int64            `json:"time"`
		Channel string           `json:"channel"`
		Event   string           `json:"event"`
		Result  []WsFuturesOrder `json:"result"`
	}{}
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	orderDetails := make([]order.Detail, len(resp.Result))
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
		orderDetails[x] = order.Detail{
			Amount:         resp.Result[x].Size,
			Exchange:       g.Name,
			OrderID:        strconv.FormatInt(resp.Result[x].ID, 10),
			Status:         status,
			Pair:           currencyPair,
			LastUpdated:    time.UnixMilli(resp.Result[x].FinishTimeMs),
			Date:           time.UnixMilli(resp.Result[x].CreateTimeMs),
			ExecutedAmount: resp.Result[x].Size - resp.Result[x].Left,
			Price:          resp.Result[x].Price,
			AssetType:      futuresAssetType,
			AccountID:      resp.Result[x].User,
			CloseTime:      time.UnixMilli(resp.Result[x].FinishTimeMs),
		}
	}
	g.Websocket.DataHandler <- orderDetails
	return nil
}

func (g *Gateio) procesFuturesUserTrades(data []byte) error {
	resp := struct {
		Time    int64                `json:"time"`
		Channel string               `json:"channel"`
		Event   string               `json:"event"`
		Result  []WsFuturesUserTrade `json:"result"`
	}{}
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
	resp := struct {
		Time    int64                               `json:"time"`
		Channel string                              `json:"channel"`
		Event   string                              `json:"event"`
		Result  []WsFuturesLiquidiationNotification `json:"result"`
	}{}
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	g.Websocket.DataHandler <- &resp
	return nil
}

func (g *Gateio) processFuturesAutoDeleveragesNotification(data []byte) error {
	resp := struct {
		Time    int64                                  `json:"time"`
		Channel string                                 `json:"channel"`
		Event   string                                 `json:"event"`
		Result  []WsFuturesAutoDeleveragesNotification `json:"result"`
	}{}
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	g.Websocket.DataHandler <- &resp
	return nil
}

func (g *Gateio) processPositionCloseData(data []byte) error {
	resp := struct {
		Time    int64             `json:"time"`
		Channel string            `json:"channel"`
		Event   string            `json:"event"`
		Result  []WsPositionClose `json:"result"`
	}{}
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	g.Websocket.DataHandler <- &resp
	return nil
}

func (g *Gateio) processBalancePushData(data []byte) error {
	resp := struct {
		Time    int64       `json:"time"`
		Channel string      `json:"channel"`
		Event   string      `json:"event"`
		Result  []WsBalance `json:"result"`
	}{}
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
		g.Websocket.DataHandler <- []account.Change{
			{
				Exchange: g.Name,
				Currency: code,
				Asset:    futuresAssetType,
				Amount:   resp.Result[x].Balance,
				Account:  resp.Result[x].User,
			},
		}
	}
	return nil
}

func (g *Gateio) processFuturesReduceRiskLimitNotification(data []byte) error {
	resp := struct {
		Time    int64                                  `json:"time"`
		Channel string                                 `json:"channel"`
		Event   string                                 `json:"event"`
		Result  []WsFuturesReduceRiskLimitNotification `json:"result"`
	}{}
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	g.Websocket.DataHandler <- &resp
	return nil
}

func (g *Gateio) processFuturesPositionsNotification(data []byte) error {
	resp := struct {
		Time    int64               `json:"time"`
		Channel string              `json:"channel"`
		Event   string              `json:"event"`
		Result  []WsFuturesPosition `json:"result"`
	}{}
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	g.Websocket.DataHandler <- &resp
	return nil
}

func (g *Gateio) processFuturesAutoOrderPushData(data []byte) error {
	resp := struct {
		Time    int64                `json:"time"`
		Channel string               `json:"channel"`
		Event   string               `json:"event"`
		Result  []WsFuturesAutoOrder `json:"result"`
	}{}
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	g.Websocket.DataHandler <- &resp
	return nil
}
