package gateio

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/types"
)

const (
	btcFuturesWebsocketURL  = "wss://fx-ws.gateio.ws/v4/ws/btc"
	usdtFuturesWebsocketURL = "wss://fx-ws.gateio.ws/v4/ws/usdt"

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

	futuresOrderbookUpdateLimit uint64 = 20
)

var defaultFuturesSubscriptions = []string{
	futuresTickersChannel,
	futuresTradesChannel,
	futuresOrderbookUpdateChannel,
	futuresCandlesticksChannel,
}

// WsFuturesConnect initiates a websocket connection for futures account
func (e *Exchange) WsFuturesConnect(ctx context.Context, conn websocket.Connection) error {
	a := asset.USDTMarginedFutures
	if conn.GetURL() == btcFuturesWebsocketURL {
		a = asset.CoinMarginedFutures
	}
	if err := e.CurrencyPairs.IsAssetEnabled(a); err != nil {
		return err
	}
	if err := conn.Dial(ctx, &gws.Dialer{}, http.Header{}); err != nil {
		return err
	}
	pingHandler, err := getWSPingHandler(futuresPingChannel)
	if err != nil {
		return err
	}
	conn.SetupPingHandler(websocketRateLimitNotNeededEPL, pingHandler)
	return nil
}

// GenerateFuturesDefaultSubscriptions returns default subscriptions information.
// TODO: Update to use the new subscription template system
func (e *Exchange) GenerateFuturesDefaultSubscriptions(a asset.Item) (subscription.List, error) {
	channelsToSubscribe := defaultFuturesSubscriptions
	if e.Websocket.CanUseAuthenticatedEndpoints() {
		channelsToSubscribe = append(channelsToSubscribe, futuresOrdersChannel, futuresUserTradesChannel, futuresBalancesChannel)
	}

	pairs, err := e.GetEnabledPairs(a)
	if err != nil {
		if errors.Is(err, asset.ErrNotEnabled) {
			return nil, nil // no enabled pairs, subscriptions require an associated pair.
		}
		return nil, err
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
				// This is the fastest frequency available for futures orderbook updates 20 levels every 20ms
				params["frequency"] = kline.TwentyMilliseconds
				params["level"] = strconv.FormatUint(futuresOrderbookUpdateLimit, 10)
			}
			fPair, err := e.FormatExchangeCurrency(pairs[j], a)
			if err != nil {
				return nil, err
			}
			subscriptions = append(subscriptions, &subscription.Subscription{
				Channel: channelsToSubscribe[i],
				Pairs:   currency.Pairs{fPair.Upper()},
				Params:  params,
				Asset:   a,
			})
		}
	}
	return subscriptions, nil
}

// FuturesSubscribe sends a websocket message to stop receiving data from the channel
func (e *Exchange) FuturesSubscribe(ctx context.Context, conn websocket.Connection, channelsToUnsubscribe subscription.List) error {
	return e.handleSubscription(ctx, conn, subscribeEvent, channelsToUnsubscribe, e.generateFuturesPayload)
}

// FuturesUnsubscribe sends a websocket message to stop receiving data from the channel
func (e *Exchange) FuturesUnsubscribe(ctx context.Context, conn websocket.Connection, channelsToUnsubscribe subscription.List) error {
	return e.handleSubscription(ctx, conn, unsubscribeEvent, channelsToUnsubscribe, e.generateFuturesPayload)
}

// WsHandleFuturesData handles futures websocket data
func (e *Exchange) WsHandleFuturesData(ctx context.Context, conn websocket.Connection, respRaw []byte, a asset.Item) error {
	push, err := parseWSHeader(respRaw)
	if err != nil {
		return err
	}

	if push.RequestID != "" {
		return conn.RequireMatchWithData(push.RequestID, respRaw)
	}

	if push.Event == subscribeEvent || push.Event == unsubscribeEvent {
		return conn.RequireMatchWithData(push.ID, respRaw)
	}

	switch push.Channel {
	case futuresTickersChannel:
		return e.processFuturesTickers(ctx, respRaw, a)
	case futuresTradesChannel:
		return e.processFuturesTrades(respRaw, a)
	case futuresOrderbookChannel:
		return e.processFuturesOrderbookSnapshot(push.Event, push.Result, a, push.Time)
	case futuresOrderbookTickerChannel:
		return e.processFuturesOrderbookTicker(ctx, push.Result)
	case futuresOrderbookUpdateChannel:
		return e.processFuturesOrderbookUpdate(ctx, push.Result, a, push.Time)
	case futuresCandlesticksChannel:
		return e.processFuturesCandlesticks(ctx, respRaw, a)
	case futuresOrdersChannel:
		processed, err := e.processFuturesOrdersPushData(respRaw, a)
		if err != nil {
			return err
		}
		return e.Websocket.DataHandler.Send(ctx, processed)
	case futuresUserTradesChannel:
		return e.procesFuturesUserTrades(respRaw, a)
	case futuresLiquidatesChannel:
		return e.processFuturesLiquidatesNotification(ctx, respRaw)
	case futuresAutoDeleveragesChannel:
		return e.processFuturesAutoDeleveragesNotification(ctx, respRaw)
	case futuresAutoPositionCloseChannel:
		return e.processPositionCloseData(ctx, respRaw)
	case futuresBalancesChannel:
		return e.processBalancePushData(ctx, push.Result, a)
	case futuresReduceRiskLimitsChannel:
		return e.processFuturesReduceRiskLimitNotification(ctx, respRaw)
	case futuresPositionsChannel:
		return e.processFuturesPositionsNotification(ctx, respRaw)
	case futuresAutoOrdersChannel:
		return e.processFuturesAutoOrderPushData(ctx, respRaw)
	case "futures.pong":
		return nil
	default:
		return e.Websocket.DataHandler.Send(ctx, websocket.UnhandledMessageWarning{
			Message: e.Name + websocket.UnhandledMessage + string(respRaw),
		})
	}
}

func (e *Exchange) generateFuturesPayload(ctx context.Context, event string, channelsToSubscribe subscription.List) ([]WsInput, error) {
	if len(channelsToSubscribe) == 0 {
		return nil, errors.New("cannot generate payload, no channels supplied")
	}
	var creds *accounts.Credentials
	var err error
	if e.Websocket.CanUseAuthenticatedEndpoints() {
		creds, err = e.GetCredentials(ctx)
		if err != nil {
			e.Websocket.SetCanUseAuthenticatedEndpoints(false)
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
		if e.Websocket.CanUseAuthenticatedEndpoints() {
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
				sigTemp, err = e.generateWsSignature(creds.Secret, event, channelsToSubscribe[i].Channel, timestamp.Unix())
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
			frequencyString, err = getIntervalString(frequency)
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
				intervalString, err = getIntervalString(interval)
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
			ID:      e.MessageSequence(),
			Event:   event,
			Channel: channelsToSubscribe[i].Channel,
			Payload: params,
			Auth:    auth,
			Time:    timestamp.Unix(),
		})
	}
	return outbound, nil
}

func (e *Exchange) processFuturesTickers(ctx context.Context, data []byte, assetType asset.Item) error {
	resp := struct {
		Time    types.Time       `json:"time"`
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
			ExchangeName: e.Name,
			Volume:       resp.Result[x].Volume24HBase.Float64(),
			QuoteVolume:  resp.Result[x].Volume24HQuote.Float64(),
			High:         resp.Result[x].High24H.Float64(),
			Low:          resp.Result[x].Low24H.Float64(),
			Last:         resp.Result[x].Last.Float64(),
			AssetType:    assetType,
			Pair:         resp.Result[x].Contract,
			LastUpdated:  resp.Time.Time(),
		}
	}
	return e.Websocket.DataHandler.Send(ctx, tickerPriceDatas)
}

func (e *Exchange) processFuturesTrades(data []byte, assetType asset.Item) error {
	saveTradeData := e.IsSaveTradeDataEnabled()
	if !saveTradeData && !e.IsTradeFeedEnabled() {
		return nil
	}

	resp := struct {
		Time    types.Time        `json:"time"`
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
			Timestamp:    resp.Result[x].CreateTime.Time(),
			CurrencyPair: resp.Result[x].Contract,
			AssetType:    assetType,
			Exchange:     e.Name,
			Price:        resp.Result[x].Price.Float64(),
			Amount:       resp.Result[x].Size,
			TID:          strconv.FormatInt(resp.Result[x].ID, 10),
		}
	}
	return e.Websocket.Trade.Update(saveTradeData, trades...)
}

func (e *Exchange) processFuturesCandlesticks(ctx context.Context, data []byte, assetType asset.Item) error {
	resp := struct {
		Time    types.Time           `json:"time"`
		Channel string               `json:"channel"`
		Event   string               `json:"event"`
		Result  []FuturesCandlestick `json:"result"`
	}{}
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	klineDatas := make([]websocket.KlineData, len(resp.Result))
	for x := range resp.Result {
		icp := strings.Split(resp.Result[x].Name, currency.UnderscoreDelimiter)
		if len(icp) < 3 {
			return fmt.Errorf("%w: futures candlestick websocket", common.ErrMalformedData)
		}
		currencyPair, err := currency.NewPairFromString(strings.Join(icp[1:], currency.UnderscoreDelimiter))
		if err != nil {
			return err
		}
		klineDatas[x] = websocket.KlineData{
			Pair:       currencyPair,
			AssetType:  assetType,
			Exchange:   e.Name,
			StartTime:  resp.Result[x].Timestamp.Time(),
			Interval:   icp[0],
			OpenPrice:  resp.Result[x].OpenPrice.Float64(),
			ClosePrice: resp.Result[x].ClosePrice.Float64(),
			HighPrice:  resp.Result[x].HighestPrice.Float64(),
			LowPrice:   resp.Result[x].LowestPrice.Float64(),
			Volume:     resp.Result[x].Volume,
		}
	}
	return e.Websocket.DataHandler.Send(ctx, klineDatas)
}

func (e *Exchange) processFuturesOrderbookTicker(ctx context.Context, incoming []byte) error {
	var data WsFuturesOrderbookTicker
	err := json.Unmarshal(incoming, &data)
	if err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, data)
}

func (e *Exchange) processFuturesOrderbookUpdate(ctx context.Context, incoming []byte, a asset.Item, pushTime time.Time) error {
	var data WsFuturesAndOptionsOrderbookUpdate
	if err := json.Unmarshal(incoming, &data); err != nil {
		return err
	}
	asks := make([]orderbook.Level, len(data.Asks))
	for x := range data.Asks {
		asks[x].Price = data.Asks[x].Price.Float64()
		asks[x].Amount = data.Asks[x].Size
	}
	bids := make([]orderbook.Level, len(data.Bids))
	for x := range data.Bids {
		bids[x].Price = data.Bids[x].Price.Float64()
		bids[x].Amount = data.Bids[x].Size
	}

	return e.wsOBUpdateMgr.ProcessOrderbookUpdate(ctx, e, data.FirstUpdatedID, &orderbook.Update{
		UpdateID:   data.LastUpdatedID,
		UpdateTime: data.Timestamp.Time(),
		LastPushed: pushTime,
		Pair:       data.ContractName,
		Asset:      a,
		Asks:       asks,
		Bids:       bids,
		AllowEmpty: true,
	})
}

func (e *Exchange) processFuturesOrderbookSnapshot(event string, incoming []byte, assetType asset.Item, lastPushed time.Time) error {
	if event == "all" {
		var data WsFuturesOrderbookSnapshot
		err := json.Unmarshal(incoming, &data)
		if err != nil {
			return err
		}
		base := orderbook.Book{
			Asset:             assetType,
			Exchange:          e.Name,
			Pair:              data.Contract,
			LastUpdated:       data.Timestamp.Time(),
			LastPushed:        lastPushed,
			ValidateOrderbook: e.ValidateOrderbook,
		}
		base.Asks = make([]orderbook.Level, len(data.Asks))
		for x := range data.Asks {
			base.Asks[x].Amount = data.Asks[x].Size
			base.Asks[x].Price = data.Asks[x].Price.Float64()
		}
		base.Bids = make([]orderbook.Level, len(data.Bids))
		for x := range data.Bids {
			base.Bids[x].Amount = data.Bids[x].Size
			base.Bids[x].Price = data.Bids[x].Price.Float64()
		}
		return e.Websocket.Orderbook.LoadSnapshot(&base)
	}
	var data []WsFuturesOrderbookUpdateEvent
	err := json.Unmarshal(incoming, &data)
	if err != nil {
		return err
	}
	dataMap := map[string][2][]orderbook.Level{}
	for x := range data {
		ab, ok := dataMap[data[x].CurrencyPair]
		if !ok {
			ab = [2][]orderbook.Level{}
		}
		if data[x].Amount > 0 {
			ab[1] = append(ab[1], orderbook.Level{
				Price:  data[x].Price.Float64(),
				Amount: data[x].Amount,
			})
		} else {
			ab[0] = append(ab[0], orderbook.Level{
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
		err = e.Websocket.Orderbook.LoadSnapshot(&orderbook.Book{
			Asks:              ab[0],
			Bids:              ab[1],
			Asset:             assetType,
			Exchange:          e.Name,
			Pair:              currencyPair,
			LastUpdated:       lastPushed,
			LastPushed:        lastPushed,
			ValidateOrderbook: e.ValidateOrderbook,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (e *Exchange) processFuturesOrdersPushData(data []byte, assetType asset.Item) ([]order.Detail, error) {
	resp := struct {
		Time    types.Time       `json:"time"`
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
			return nil, err
		}

		orderDetails[x] = order.Detail{
			Amount:         resp.Result[x].Size,
			Exchange:       e.Name,
			OrderID:        strconv.FormatInt(resp.Result[x].ID, 10),
			Status:         status,
			Pair:           resp.Result[x].Contract,
			LastUpdated:    resp.Result[x].FinishTime.Time(),
			Date:           resp.Result[x].CreateTime.Time(),
			ExecutedAmount: resp.Result[x].Size - resp.Result[x].Left,
			Price:          resp.Result[x].Price,
			AssetType:      assetType,
			AccountID:      resp.Result[x].User,
			CloseTime:      resp.Result[x].FinishTime.Time(),
		}
	}
	return orderDetails, nil
}

func (e *Exchange) procesFuturesUserTrades(data []byte, assetType asset.Item) error {
	if !e.IsFillsFeedEnabled() {
		return nil
	}

	resp := struct {
		Time    types.Time           `json:"time"`
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
			Timestamp:    resp.Result[x].CreateTime.Time(),
			Exchange:     e.Name,
			CurrencyPair: resp.Result[x].Contract,
			OrderID:      resp.Result[x].OrderID,
			TradeID:      resp.Result[x].ID,
			Price:        resp.Result[x].Price.Float64(),
			Amount:       resp.Result[x].Size,
			AssetType:    assetType,
		}
	}
	return e.Websocket.Fills.Update(fills...)
}

func (e *Exchange) processFuturesLiquidatesNotification(ctx context.Context, data []byte) error {
	resp := struct {
		Time    types.Time                         `json:"time"`
		Channel string                             `json:"channel"`
		Event   string                             `json:"event"`
		Result  []WsFuturesLiquidationNotification `json:"result"`
	}{}
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, &resp)
}

func (e *Exchange) processFuturesAutoDeleveragesNotification(ctx context.Context, data []byte) error {
	resp := struct {
		Time    types.Time                             `json:"time"`
		Channel string                                 `json:"channel"`
		Event   string                                 `json:"event"`
		Result  []WsFuturesAutoDeleveragesNotification `json:"result"`
	}{}
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, &resp)
}

func (e *Exchange) processPositionCloseData(ctx context.Context, data []byte) error {
	resp := struct {
		Time    types.Time        `json:"time"`
		Channel string            `json:"channel"`
		Event   string            `json:"event"`
		Result  []WsPositionClose `json:"result"`
	}{}
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, &resp)
}

func (e *Exchange) processBalancePushData(ctx context.Context, data []byte, assetType asset.Item) error {
	var resp []*WsBalance
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}

	subAccts := accounts.SubAccounts{}
	for _, bal := range resp {
		c := bal.Currency
		if assetType == asset.Options && c.IsEmpty() {
			c = currency.USDT // Settlement currency is USDT
		}
		a := accounts.NewSubAccount(assetType, bal.User)
		a.Balances.Set(c, accounts.Balance{
			Total:                  bal.Balance,
			Free:                   bal.Balance,
			AvailableWithoutBorrow: bal.Balance,
			UpdatedAt:              bal.Time.Time(),
		})
		subAccts = subAccts.Merge(a)
	}
	if err := e.Accounts.Save(ctx, subAccts, false); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, subAccts)
}

func (e *Exchange) processFuturesReduceRiskLimitNotification(ctx context.Context, data []byte) error {
	resp := struct {
		Time    types.Time                             `json:"time"`
		Channel string                                 `json:"channel"`
		Event   string                                 `json:"event"`
		Result  []WsFuturesReduceRiskLimitNotification `json:"result"`
	}{}
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, &resp)
}

func (e *Exchange) processFuturesPositionsNotification(ctx context.Context, data []byte) error {
	resp := struct {
		Time    types.Time          `json:"time"`
		Channel string              `json:"channel"`
		Event   string              `json:"event"`
		Result  []WsFuturesPosition `json:"result"`
	}{}
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, &resp)
}

func (e *Exchange) processFuturesAutoOrderPushData(ctx context.Context, data []byte) error {
	resp := struct {
		Time    types.Time           `json:"time"`
		Channel string               `json:"channel"`
		Event   string               `json:"event"`
		Result  []WsFuturesAutoOrder `json:"result"`
	}{}
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, &resp)
}
