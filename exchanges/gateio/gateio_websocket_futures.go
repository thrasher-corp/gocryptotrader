package gateio

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
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

// WsFuturesConnect initiates a websocket connection for futures account
func (g *Gateio) WsFuturesConnect(ctx context.Context, conn stream.Connection) error {
	err := g.CurrencyPairs.IsAssetEnabled(asset.Futures)
	if err != nil {
		return err
	}
	err = conn.DialContext(ctx, &websocket.Dialer{}, http.Header{})
	if err != nil {
		return err
	}
	pingMessage, err := json.Marshal(WsInput{
		ID:      conn.GenerateMessageID(false),
		Time:    time.Now().Unix(), // TODO: Func for dynamic time as this will be the same time for every ping message.
		Channel: futuresPingChannel,
	})
	if err != nil {
		return err
	}
	conn.SetupPingHandler(websocketRateLimitNotNeededEPL, stream.PingHandler{
		Websocket:   true,
		MessageType: websocket.PingMessage,
		Delay:       time.Second * 15,
		Message:     pingMessage,
	})
	return nil
}

// GenerateFuturesDefaultSubscriptions returns default subscriptions information.
func (g *Gateio) GenerateFuturesDefaultSubscriptions(settlement currency.Code) (subscription.List, error) {
	channelsToSubscribe := defaultFuturesSubscriptions
	if g.Websocket.CanUseAuthenticatedEndpoints() {
		channelsToSubscribe = append(channelsToSubscribe, futuresOrdersChannel, futuresUserTradesChannel, futuresBalancesChannel)
	}

	pairs, err := g.GetEnabledPairs(asset.Futures)
	if err != nil {
		if errors.Is(err, asset.ErrNotEnabled) {
			return nil, nil // no enabled pairs, subscriptions require an associated pair.
		}
		return nil, err
	}

	switch {
	case settlement.Equal(currency.USDT):
		pairs = slices.DeleteFunc(pairs, func(p currency.Pair) bool { return !p.Quote.Equal(currency.USDT) })
	case settlement.Equal(currency.BTC):
		pairs = slices.DeleteFunc(pairs, func(p currency.Pair) bool { return p.Quote.Equal(currency.USDT) })
	default:
		return nil, fmt.Errorf("settlement currency %s not supported", settlement)
	}

	var subscriptions subscription.List
	for i := range channelsToSubscribe {
		for j := range pairs {
			params := make(map[string]any)
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
			fPair, err := g.FormatExchangeCurrency(pairs[j], asset.Futures)
			if err != nil {
				return nil, err
			}
			subscriptions = append(subscriptions, &subscription.Subscription{
				Channel: channelsToSubscribe[i],
				Pairs:   currency.Pairs{fPair.Upper()},
				Params:  params,
				Asset:   asset.Futures,
			})
		}
	}
	return subscriptions, nil
}

// FuturesSubscribe sends a websocket message to stop receiving data from the channel
func (g *Gateio) FuturesSubscribe(ctx context.Context, conn stream.Connection, channelsToUnsubscribe subscription.List) error {
	return g.handleSubscription(ctx, conn, subscribeEvent, channelsToUnsubscribe, g.generateFuturesPayload)
}

// FuturesUnsubscribe sends a websocket message to stop receiving data from the channel
func (g *Gateio) FuturesUnsubscribe(ctx context.Context, conn stream.Connection, channelsToUnsubscribe subscription.List) error {
	return g.handleSubscription(ctx, conn, unsubscribeEvent, channelsToUnsubscribe, g.generateFuturesPayload)
}

// WsHandleFuturesData handles futures websocket data
func (g *Gateio) WsHandleFuturesData(_ context.Context, respRaw []byte, a asset.Item) error {
	var push WsResponse
	err := json.Unmarshal(respRaw, &push)
	if err != nil {
		return err
	}

	if push.Event == subscribeEvent || push.Event == unsubscribeEvent {
		if !g.Websocket.Match.IncomingWithData(push.ID, respRaw) {
			return fmt.Errorf("couldn't match subscription message with ID: %d", push.ID)
		}
		return nil
	}

	switch push.Channel {
	case futuresTickersChannel:
		return g.processFuturesTickers(respRaw, a)
	case futuresTradesChannel:
		return g.processFuturesTrades(respRaw, a)
	case futuresOrderbookChannel:
		return g.processFuturesOrderbookSnapshot(push.Event, push.Result, a, push.TimeMs.Time())
	case futuresOrderbookTickerChannel:
		return g.processFuturesOrderbookTicker(push.Result)
	case futuresOrderbookUpdateChannel:
		return g.processFuturesAndOptionsOrderbookUpdate(push.Result, a)
	case futuresCandlesticksChannel:
		return g.processFuturesCandlesticks(respRaw, a)
	case futuresOrdersChannel:
		var processed []order.Detail
		processed, err = g.processFuturesOrdersPushData(respRaw, a)
		if err != nil {
			return err
		}
		g.Websocket.DataHandler <- processed
		return nil
	case futuresUserTradesChannel:
		return g.procesFuturesUserTrades(respRaw, a)
	case futuresLiquidatesChannel:
		return g.processFuturesLiquidatesNotification(respRaw)
	case futuresAutoDeleveragesChannel:
		return g.processFuturesAutoDeleveragesNotification(respRaw)
	case futuresAutoPositionCloseChannel:
		return g.processPositionCloseData(respRaw)
	case futuresBalancesChannel:
		return g.processBalancePushData(respRaw, a)
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

func (g *Gateio) generateFuturesPayload(ctx context.Context, conn stream.Connection, event string, channelsToSubscribe subscription.List) ([]WsInput, error) {
	if len(channelsToSubscribe) == 0 {
		return nil, errors.New("cannot generate payload, no channels supplied")
	}
	var creds *account.Credentials
	var err error
	if g.Websocket.CanUseAuthenticatedEndpoints() {
		creds, err = g.GetCredentials(ctx)
		if err != nil {
			g.Websocket.SetCanUseAuthenticatedEndpoints(false)
		}
	}

	outbound := make([]WsInput, 0, len(channelsToSubscribe))
	for i := range channelsToSubscribe {
		if len(channelsToSubscribe[i].Pairs) != 1 {
			return nil, subscription.ErrNotSinglePair
		}
		var auth *WsAuthInput
		timestamp := time.Now()
		var params []string
		params = []string{channelsToSubscribe[i].Pairs[0].String()}
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
				sigTemp, err = g.generateWsSignature(creds.Secret, event, channelsToSubscribe[i].Channel, timestamp.Unix())
				if err != nil {
					return nil, err
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
				return nil, err
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
					return nil, err
				}
				params = append([]string{intervalString}, params...)
			}
		case futuresOrderbookChannel:
			intervalString, okay := channelsToSubscribe[i].Params["interval"].(string)
			if okay {
				params = append(params, intervalString)
			}
		}
		outbound = append(outbound, WsInput{
			ID:      conn.GenerateMessageID(false),
			Event:   event,
			Channel: channelsToSubscribe[i].Channel,
			Payload: params,
			Auth:    auth,
			Time:    timestamp.Unix(),
		})
	}
	return outbound, nil
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
		tickerPriceDatas[x] = ticker.Price{
			ExchangeName: g.Name,
			Volume:       resp.Result[x].Volume24HBase.Float64(),
			QuoteVolume:  resp.Result[x].Volume24HQuote.Float64(),
			High:         resp.Result[x].High24H.Float64(),
			Low:          resp.Result[x].Low24H.Float64(),
			Last:         resp.Result[x].Last.Float64(),
			AssetType:    assetType,
			Pair:         resp.Result[x].Contract,
			LastUpdated:  time.Unix(resp.Time, 0),
		}
	}
	g.Websocket.DataHandler <- tickerPriceDatas
	return nil
}

func (g *Gateio) processFuturesTrades(data []byte, assetType asset.Item) error {
	saveTradeData := g.IsSaveTradeDataEnabled()
	if !saveTradeData && !g.IsTradeFeedEnabled() {
		return nil
	}

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
		trades[x] = trade.Data{
			Timestamp:    resp.Result[x].CreateTimeMs.Time(),
			CurrencyPair: resp.Result[x].Contract,
			AssetType:    assetType,
			Exchange:     g.Name,
			Price:        resp.Result[x].Price.Float64(),
			Amount:       resp.Result[x].Size,
			TID:          strconv.FormatInt(resp.Result[x].ID, 10),
		}
	}
	return g.Websocket.Trade.Update(saveTradeData, trades...)
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
			OpenPrice:  resp.Result[x].OpenPrice.Float64(),
			ClosePrice: resp.Result[x].ClosePrice.Float64(),
			HighPrice:  resp.Result[x].HighestPrice.Float64(),
			LowPrice:   resp.Result[x].LowestPrice.Float64(),
			Volume:     resp.Result[x].Volume,
		}
	}
	g.Websocket.DataHandler <- klineDatas
	return nil
}

func (g *Gateio) processFuturesOrderbookTicker(incoming []byte) error {
	var data WsFuturesOrderbookTicker
	err := json.Unmarshal(incoming, &data)
	if err != nil {
		return err
	}
	g.Websocket.DataHandler <- data
	return nil
}

func (g *Gateio) processFuturesAndOptionsOrderbookUpdate(incoming []byte, assetType asset.Item) error {
	var data WsFuturesAndOptionsOrderbookUpdate
	err := json.Unmarshal(incoming, &data)
	if err != nil {
		return err
	}
	if (assetType == asset.Options && !fetchedOptionsCurrencyPairSnapshotOrderbook[data.ContractName.String()]) ||
		(assetType != asset.Options && !fetchedFuturesCurrencyPairSnapshotOrderbook[data.ContractName.String()]) {
		orderbooks, err := g.FetchOrderbook(context.Background(), data.ContractName, assetType)
		if err != nil {
			return err
		}
		if orderbooks.LastUpdateID < data.FirstUpdatedID || orderbooks.LastUpdateID > data.LastUpdatedID {
			return nil
		}
		err = g.Websocket.Orderbook.LoadSnapshot(orderbooks)
		if err != nil {
			return err
		}
		if assetType == asset.Options {
			fetchedOptionsCurrencyPairSnapshotOrderbook[data.ContractName.String()] = true
		} else {
			fetchedFuturesCurrencyPairSnapshotOrderbook[data.ContractName.String()] = true
		}
	}
	updates := orderbook.Update{
		UpdateTime: data.TimestampInMs.Time(),
		Pair:       data.ContractName,
		Asset:      assetType,
	}
	updates.Asks = make([]orderbook.Tranche, len(data.Asks))
	for x := range data.Asks {
		updates.Asks[x].Amount = data.Asks[x].Size
		updates.Asks[x].Price = data.Asks[x].Price.Float64()
	}
	updates.Bids = make([]orderbook.Tranche, len(data.Bids))
	for x := range data.Bids {
		updates.Bids[x].Amount = data.Bids[x].Size
		updates.Bids[x].Price = data.Bids[x].Price.Float64()
	}
	if len(updates.Asks) == 0 && len(updates.Bids) == 0 {
		return errors.New("malformed orderbook data")
	}
	return g.Websocket.Orderbook.Update(&updates)
}

func (g *Gateio) processFuturesOrderbookSnapshot(event string, incoming []byte, assetType asset.Item, updatePushedAt time.Time) error {
	if event == "all" {
		var data WsFuturesOrderbookSnapshot
		err := json.Unmarshal(incoming, &data)
		if err != nil {
			return err
		}
		base := orderbook.Base{
			Asset:           assetType,
			Exchange:        g.Name,
			Pair:            data.Contract,
			LastUpdated:     data.TimestampInMs.Time(),
			UpdatePushedAt:  updatePushedAt,
			VerifyOrderbook: g.CanVerifyOrderbook,
		}
		base.Asks = make([]orderbook.Tranche, len(data.Asks))
		for x := range data.Asks {
			base.Asks[x].Amount = data.Asks[x].Size
			base.Asks[x].Price = data.Asks[x].Price.Float64()
		}
		base.Bids = make([]orderbook.Tranche, len(data.Bids))
		for x := range data.Bids {
			base.Bids[x].Amount = data.Bids[x].Size
			base.Bids[x].Price = data.Bids[x].Price.Float64()
		}
		return g.Websocket.Orderbook.LoadSnapshot(&base)
	}
	var data []WsFuturesOrderbookUpdateEvent
	err := json.Unmarshal(incoming, &data)
	if err != nil {
		return err
	}
	dataMap := map[string][2][]orderbook.Tranche{}
	for x := range data {
		ab, ok := dataMap[data[x].CurrencyPair]
		if !ok {
			ab = [2][]orderbook.Tranche{}
		}
		if data[x].Amount > 0 {
			ab[1] = append(ab[1], orderbook.Tranche{
				Price:  data[x].Price.Float64(),
				Amount: data[x].Amount,
			})
		} else {
			ab[0] = append(ab[0], orderbook.Tranche{
				Price:  data[x].Price.Float64(),
				Amount: -data[x].Amount,
			})
		}
		if !ok {
			dataMap[data[x].CurrencyPair] = ab
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
			LastUpdated:     updatePushedAt,
			UpdatePushedAt:  updatePushedAt,
			VerifyOrderbook: g.CanVerifyOrderbook,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (g *Gateio) processFuturesOrdersPushData(data []byte, assetType asset.Item) ([]order.Detail, error) {
	resp := struct {
		Time    int64            `json:"time"`
		Channel string           `json:"channel"`
		Event   string           `json:"event"`
		Result  []WsFuturesOrder `json:"result"`
	}{}
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return nil, err
	}
	orderDetails := make([]order.Detail, len(resp.Result))
	for x := range resp.Result {
		var status order.Status
		if resp.Result[x].Status == "finished" {
			if resp.Result[x].FinishAs == "ioc" || resp.Result[x].FinishAs == "reduce_only" {
				status = order.Cancelled
			} else {
				status, err = order.StringToOrderStatus(resp.Result[x].FinishAs)
			}
		} else {
			status, err = order.StringToOrderStatus(resp.Result[x].Status)
		}
		if err != nil {
			g.Websocket.DataHandler <- order.ClassificationError{
				Exchange: g.Name,
				OrderID:  strconv.FormatInt(resp.Result[x].ID, 10),
				Err:      err,
			}
		}

		orderDetails[x] = order.Detail{
			Amount:         resp.Result[x].Size,
			Exchange:       g.Name,
			OrderID:        strconv.FormatInt(resp.Result[x].ID, 10),
			Status:         status,
			Pair:           resp.Result[x].Contract,
			LastUpdated:    resp.Result[x].FinishTimeMs.Time(),
			Date:           resp.Result[x].CreateTimeMs.Time(),
			ExecutedAmount: resp.Result[x].Size - resp.Result[x].Left,
			Price:          resp.Result[x].Price,
			AssetType:      assetType,
			AccountID:      resp.Result[x].User,
			CloseTime:      resp.Result[x].FinishTimeMs.Time(),
		}
	}
	return orderDetails, nil
}

func (g *Gateio) procesFuturesUserTrades(data []byte, assetType asset.Item) error {
	if !g.IsFillsFeedEnabled() {
		return nil
	}

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
		fills[x] = fill.Data{
			Timestamp:    resp.Result[x].CreateTimeMs.Time(),
			Exchange:     g.Name,
			CurrencyPair: resp.Result[x].Contract,
			OrderID:      resp.Result[x].OrderID,
			TradeID:      resp.Result[x].ID,
			Price:        resp.Result[x].Price.Float64(),
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
