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

// responseFuturesStream a channel thought which the data coming from the two websocket connection will go through.
var responseFuturesStream = make(chan stream.Response)

// WsFuturesConnect initiates a websocket connection for futures account
func (g *Gateio) WsFuturesConnect() error {
	if !g.Websocket.IsEnabled() || !g.IsEnabled() {
		return errors.New(stream.WebsocketNotEnabled)
	}
	err := g.CurrencyPairs.IsAssetEnabled(asset.Futures)
	if err != nil {
		return err
	}
	var dialer websocket.Dialer
	err = g.Websocket.SetWebsocketURL(futuresWebsocketUsdtURL, false, true)
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
	go g.wsReadFuturesData()
	go g.wsFunnelFuturesConnectionData(g.Websocket.Conn)
	go g.wsFunnelFuturesConnectionData(g.Websocket.AuthConn)
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
	return nil
}

// GenerateFuturesDefaultSubscriptions returns default subscriptions information.
func (g *Gateio) GenerateFuturesDefaultSubscriptions() ([]stream.ChannelSubscription, error) {
	channelsToSubscribe := defaultFuturesSubscriptions
	if g.Websocket.CanUseAuthenticatedEndpoints() {
		channelsToSubscribe = append(channelsToSubscribe,
			futuresOrdersChannel,
			futuresUserTradesChannel,
			futuresBalancesChannel,
		)
	}
	pairs, err := g.GetEnabledPairs(asset.Futures)
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
			fpair, err := g.FormatExchangeCurrency(pairs[j], asset.Futures)
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

// wsReadFuturesData read coming messages thought the websocket connection and pass the data to wsHandleData for further process.
func (g *Gateio) wsReadFuturesData() {
	defer g.Websocket.Wg.Done()
	for {
		select {
		case <-g.Websocket.ShutdownC:
			select {
			case resp := <-responseFuturesStream:
				err := g.wsHandleFuturesData(resp.Raw, asset.Futures)
				if err != nil {
					select {
					case g.Websocket.DataHandler <- err:
					default:
						log.Errorf(log.WebsocketMgr, "%s websocket handle data error: %v", g.Name, err)
					}
				}
			default:
			}
			return
		case resp := <-responseFuturesStream:
			err := g.wsHandleFuturesData(resp.Raw, asset.Futures)
			if err != nil {
				g.Websocket.DataHandler <- err
			}
		}
	}
}

// wsFunnelFuturesConnectionData receives data from multiple connection and pass the data
// to wsRead through a channel responseStream
func (g *Gateio) wsFunnelFuturesConnectionData(ws stream.Connection) {
	defer g.Websocket.Wg.Done()
	for {
		resp := ws.ReadMessage()
		if resp.Raw == nil {
			return
		}
		responseFuturesStream <- stream.Response{Raw: resp.Raw}
	}
}

func (g *Gateio) wsHandleFuturesData(respRaw []byte, assetType asset.Item) error {
	var result WsResponse
	var eventResponse WsEventResponse
	err := json.Unmarshal(respRaw, &eventResponse)
	if err == nil &&
		(eventResponse.Result != nil || eventResponse.Error != nil) &&
		(eventResponse.Event == "subscribe" || eventResponse.Event == "unsubscribe") {
		if !g.Websocket.Match.IncomingWithData(eventResponse.ID, respRaw) {
			return fmt.Errorf("couldn't match subscription message with ID: %d", eventResponse.ID)
		}
		return nil
	}
	err = json.Unmarshal(respRaw, &result)
	if err != nil {
		return err
	}
	switch result.Channel {
	// Futures push datas.
	case futuresTickersChannel:
		return g.processFuturesTickers(respRaw, assetType)
	case futuresTradesChannel:
		return g.processFuturesTrades(respRaw, assetType)
	case futuresOrderbookChannel:
		return g.processFuturesOrderbookSnapshot(result.Event, respRaw, assetType)
	case futuresOrderbookTickerChannel:
		return g.processFuturesOrderbookTicker(respRaw)
	case futuresOrderbookUpdateChannel:
		return g.processFuturesAndOptionsOrderbookUpdate(respRaw, assetType)
	case futuresCandlesticksChannel:
		return g.processFuturesCandlesticks(respRaw, assetType)
	case futuresOrdersChannel:
		return g.processFuturesOrdersPushData(respRaw, assetType)
	case futuresUserTradesChannel:
		return g.procesFuturesUserTrades(respRaw, assetType)
	case futuresLiquidatesChannel:
		return g.processFuturesLiquidatesNotification(respRaw)
	case futuresAutoDeleveragesChannel:
		return g.processFuturesAutoDeleveragesNotification(respRaw)
	case futuresAutoPositionCloseChannel:
		return g.processPositionCloseData(respRaw)
	case futuresBalancesChannel:
		return g.processBalancePushData(respRaw, assetType)
	case futuresReduceRiskLimitsChannel:
		return g.processFuturesReduceRiskLimitNotification(respRaw)
	case futuresPositionsChannel:
		return g.processFuturesPositionsNotification(respRaw)
	case futuresAutoOrdersChannel:
		return g.processFuturesAutoOrderPushData(respRaw)
	default:
		g.Websocket.DataHandler <- stream.UnhandledMessageWarning{
			Message: g.Name + stream.UnhandledMessage + string(respRaw),
		}
		return errors.New(stream.UnhandledMessage)
	}
}

// handleFuturesSubscription sends a websocket message to receive data from the channel
func (g *Gateio) handleFuturesSubscription(event string, channelsToSubscribe []stream.ChannelSubscription) error {
	payloads, err := g.generateFuturesPayload(event, channelsToSubscribe)
	if err != nil {
		return err
	}
	var errs error
	var respByte []byte
	// con represents the websocket connection. 0 - for usdt settle and 1 - for btc settle connections.
	for con, val := range payloads {
		for k := range val {
			if con == 0 {
				respByte, err = g.Websocket.Conn.SendMessageReturnResponse(val[k].ID, val[k])
			} else {
				respByte, err = g.Websocket.AuthConn.SendMessageReturnResponse(val[k].ID, val[k])
			}
			if err != nil {
				errs = common.AppendError(errs, err)
				continue
			}
			var resp WsEventResponse
			if err = json.Unmarshal(respByte, &resp); err != nil {
				errs = common.AppendError(errs, err)
			} else {
				if resp.Error != nil && resp.Error.Code != 0 {
					errs = common.AppendError(errs, fmt.Errorf("error while %s to channel %s error code: %d message: %s", val[k].Event, val[k].Channel, resp.Error.Code, resp.Error.Message))
					continue
				}
				g.Websocket.AddSuccessfulSubscriptions(channelsToSubscribe[k])
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
		if g.Websocket.CanUseAuthenticatedEndpoints() {
			switch channelsToSubscribe[i].Channel {
			case futuresOrdersChannel, futuresUserTradesChannel,
				futuresLiquidatesChannel, futuresAutoDeleveragesChannel,
				futuresAutoPositionCloseChannel, futuresBalancesChannel,
				futuresReduceRiskLimitsChannel, futuresPositionsChannel,
				futuresAutoOrdersChannel:
				value, ok := channelsToSubscribe[i].Params["user"].(string)
				if ok {
					params = append(
						[]string{value},
						params...)
				}
				var sigTemp string
				sigTemp, err = g.generateWsSignature(creds.Secret, event, channelsToSubscribe[i].Channel, timestamp)
				if err != nil {
					return [2][]WsInput{}, err
				}
				auth = &WsAuthInput{
					Method: "api_key",
					Key:    creds.Key,
					Sign:   sigTemp,
				}
			}
		}
		frequency, okay := channelsToSubscribe[i].Params["frequency"].(kline.Interval)
		if okay {
			var frequencyString string
			frequencyString, err = g.GetIntervalString(frequency)
			if err != nil {
				return payloads, err
			}
			params = append(params, frequencyString)
		}
		levelString, okay := channelsToSubscribe[i].Params["level"].(string)
		if okay {
			params = append(params, levelString)
		}
		limit, okay := channelsToSubscribe[i].Params["limit"].(int)
		if okay {
			params = append(params, strconv.Itoa(limit))
		}
		accuracy, okay := channelsToSubscribe[i].Params["accuracy"].(string)
		if okay {
			params = append(params, accuracy)
		}
		switch channelsToSubscribe[i].Channel {
		case futuresCandlesticksChannel:
			interval, okay := channelsToSubscribe[i].Params["interval"].(kline.Interval)
			if okay {
				var intervalString string
				intervalString, err = g.GetIntervalString(interval)
				if err != nil {
					return payloads, err
				}
				params = append([]string{intervalString}, params...)
			}
		case futuresOrderbookChannel:
			intervalString, okay := channelsToSubscribe[i].Params["interval"].(string)
			if okay {
				params = append(params, intervalString)
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

func (g *Gateio) processFuturesTickers(data []byte, assetType asset.Item) error {
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
			AssetType:    assetType,
			Pair:         currencyPair,
			LastUpdated:  time.Unix(resp.Time, 0),
		}
	}
	g.Websocket.DataHandler <- tickerPriceDatas
	return nil
}

func (g *Gateio) processFuturesTrades(data []byte, assetType asset.Item) error {
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
			Timestamp:    resp.Result[x].CreateTimeMs.Time(),
			CurrencyPair: currencyPair,
			AssetType:    assetType,
			Exchange:     g.Name,
			Price:        resp.Result[x].Price,
			Amount:       resp.Result[x].Size,
			TID:          strconv.FormatInt(resp.Result[x].ID, 10),
		}
	}
	return trade.AddTradesToBuffer(g.Name, trades...)
}

func (g *Gateio) processFuturesCandlesticks(data []byte, assetType asset.Item) error {
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
			AssetType:  assetType,
			Exchange:   g.Name,
			StartTime:  resp.Result[x].Timestamp.Time(),
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

func (g *Gateio) processFuturesAndOptionsOrderbookUpdate(data []byte, assetType asset.Item) error {
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
	if (assetType == asset.Options && !fetchedOptionsCurrencyPairSnapshotOrderbook[update.ContractName]) ||
		(assetType != asset.Options && !fetchedFuturesCurrencyPairSnapshotOrderbook[update.ContractName]) {
		orderbooks, err := g.FetchOrderbook(context.Background(), pair, assetType)
		if err != nil {
			return err
		}
		if orderbooks.LastUpdateID < update.FirstUpdatedID || orderbooks.LastUpdateID > update.LastUpdatedID {
			return nil
		}
		err = g.Websocket.Orderbook.LoadSnapshot(orderbooks)
		if err != nil {
			return err
		}
		if assetType == asset.Options {
			fetchedOptionsCurrencyPairSnapshotOrderbook[update.ContractName] = true
		} else {
			fetchedFuturesCurrencyPairSnapshotOrderbook[update.ContractName] = true
		}
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

func (g *Gateio) processFuturesOrderbookSnapshot(event string, data []byte, assetType asset.Item) error {
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
			Asset:           assetType,
			Exchange:        g.Name,
			Pair:            pair,
			LastUpdated:     snapshot.TimestampInMs.Time(),
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
			Asset:           assetType,
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

func (g *Gateio) processFuturesOrdersPushData(data []byte, assetType asset.Item) error {
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
			LastUpdated:    resp.Result[x].FinishTimeMs.Time(),
			Date:           resp.Result[x].CreateTimeMs.Time(),
			ExecutedAmount: resp.Result[x].Size - resp.Result[x].Left,
			Price:          resp.Result[x].Price,
			AssetType:      assetType,
			AccountID:      resp.Result[x].User,
			CloseTime:      resp.Result[x].FinishTimeMs.Time(),
		}
	}
	g.Websocket.DataHandler <- orderDetails
	return nil
}

func (g *Gateio) procesFuturesUserTrades(data []byte, assetType asset.Item) error {
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
			Timestamp:    resp.Result[x].CreateTimeMs.Time(),
			Exchange:     g.Name,
			CurrencyPair: currencyPair,
			OrderID:      resp.Result[x].OrderID,
			TradeID:      resp.Result[x].ID,
			Price:        resp.Result[x].Price,
			Amount:       resp.Result[x].Size,
			AssetType:    assetType,
		}
	}
	return g.Websocket.Fills.Update(fills...)
}

func (g *Gateio) processFuturesLiquidatesNotification(data []byte) error {
	resp := struct {
		Time    int64                              `json:"time"`
		Channel string                             `json:"channel"`
		Event   string                             `json:"event"`
		Result  []WsFuturesLiquidationNotification `json:"result"`
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

func (g *Gateio) processBalancePushData(data []byte, assetType asset.Item) error {
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
	accountChange := make([]account.Change, len(resp.Result))
	for x := range resp.Result {
		info := strings.Split(resp.Result[x].Text, currency.UnderscoreDelimiter)
		if len(info) != 2 {
			return errors.New("malformed text")
		}
		code := currency.NewCode(info[0])
		accountChange[x] = account.Change{
			Exchange: g.Name,
			Currency: code,
			Asset:    assetType,
			Amount:   resp.Result[x].Balance,
			Account:  resp.Result[x].User,
		}
	}
	g.Websocket.DataHandler <- accountChange
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
