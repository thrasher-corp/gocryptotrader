package binance

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"

	gws "github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	binanceUFuturesWebsocketURL     = "wss://fstream.binance.com"
	binanceUFuturesAuthWebsocketURL = "wss://fstream-auth.binance.com"
)

var defaultSubscriptions = []string{
	depthChan,
	tickerAllChan,
	continuousKline,
}

// getKlineIntervalString returns a string representation of the kline interval.
func getKlineIntervalString(interval kline.Interval) string {
	klineMap := map[kline.Interval]string{
		kline.OneMin: "1m", kline.ThreeMin: "3m", kline.FiveMin: "5m", kline.FifteenMin: "15m", kline.ThirtyMin: "30m",
		kline.OneHour: "1h", kline.TwoHour: "2h", kline.FourHour: "4h", kline.SixHour: "6h", kline.EightHour: "8h", kline.TwelveHour: "12h",
		kline.OneDay: "1d", kline.ThreeDay: "3d", kline.OneWeek: "1w", kline.OneMonth: "1M",
	}
	intervalString, okay := klineMap[interval]
	if !okay {
		return ""
	}
	return intervalString
}

// WsUFuturesConnect initiates a websocket connection
func (e *Exchange) WsUFuturesConnect(ctx context.Context, conn websocket.Connection) error {
	err := e.CurrencyPairs.IsAssetEnabled(asset.USDTMarginedFutures)
	if err != nil {
		return err
	}

	dialer := gws.Dialer{
		HandshakeTimeout: e.Config.HTTPTimeout,
		Proxy:            http.ProxyFromEnvironment,
	}
	wsURL := binanceUFuturesWebsocketURL + "/stream"
	conn.SetURL(wsURL)

	if e.Websocket.CanUseAuthenticatedEndpoints() {
		listenKey, err = e.GetWsAuthStreamKey(context.TODO())
		switch {
		case err != nil:
			e.Websocket.SetCanUseAuthenticatedEndpoints(false)
			log.Errorf(log.ExchangeSys, "%v unable to connect to authenticated Websocket. Error: %s", e.Name, err)
		default:
			wsURL = binanceUFuturesAuthWebsocketURL + "?streams=" + listenKey
			conn.SetURL(wsURL)
		}
	}
	err = conn.Dial(ctx, &dialer, http.Header{})
	if err != nil {
		return fmt.Errorf("%v - Unable to connect to Websocket. Error: %s", e.Name, err)
	}
	conn.SetupPingHandler(request.UnAuth, websocket.PingHandler{
		UseGorillaHandler: true,
		MessageType:       gws.PongMessage,
		Delay:             pingDelay,
	})
	return nil
}

func (e *Exchange) wsHandleFuturesData(_ context.Context, respRaw []byte, assetType asset.Item) error {
	result := struct {
		Result json.RawMessage `json:"result"`
		ID     int64           `json:"id"`
		Stream string          `json:"stream"`
		Data   json.RawMessage `json:"data"`
	}{}
	err := json.Unmarshal(respRaw, &result)
	if err != nil {
		return err
	}
	if result.Stream == "" || (result.ID != 0 && result.Result != nil) {
		if !e.Websocket.Match.IncomingWithData(result.ID, respRaw) {
			return errors.New("Unhandled data: " + string(respRaw))
		}
		return nil
	}
	var stream string
	switch result.Stream {
	case assetIndexAllChan, forceOrderAllChan, bookTickerAllChan, tickerAllChan, miniTickerAllChan:
		stream = result.Stream
	default:
		stream = extractStreamInfo(result.Stream)
	}
	switch stream {
	case assetIndexAllChan, "assetIndex":
		return e.processMultiAssetModeAssetIndexes(result.Data, true)
	case contractInfoAllChan:
		return e.processContractInfoStream(result.Data)
	case forceOrderAllChan, "forceOrder":
		return e.processForceOrder(result.Data, assetType)
	case bookTickerAllChan, "bookTicker":
		return e.processBookTicker(result.Data, assetType)
	case tickerAllChan:
		return e.processMarketTicker(result.Data, true, assetType)
	case "ticker":
		return e.processMarketTicker(result.Data, false, assetType)
	case miniTickerAllChan:
		return e.processMiniTickers(result.Data, true, assetType)
	case "miniTicker":
		return e.processMiniTickers(result.Data, false, assetType)
	case "aggTrade":
		return e.processAggregateTrade(result.Data, assetType)
	case "markPrice":
		return e.processMarkPriceUpdate(result.Data, false)
	case "!markPrice@arr":
		return e.processMarkPriceUpdate(result.Data, true)
	case "depth":
		return e.processOrderbookDepthUpdate(result.Data, assetType)
	case "compositeIndex":
		return e.processCompositeIndex(result.Data)
	case continuousKline:
		return e.processContinuousKlineUpdate(result.Data, assetType)
	}
	return fmt.Errorf("unhandled stream data %s", string(respRaw))
}

func (e *Exchange) processContinuousKlineUpdate(respRaw []byte, assetType asset.Item) error {
	var resp FutureContinuousKline
	err := json.Unmarshal(respRaw, &resp)
	if err != nil {
		return err
	}
	cp, err := currency.NewPairFromString(resp.Pair)
	if err != nil {
		return err
	}
	e.Websocket.DataHandler <- websocket.KlineData{
		Timestamp:  resp.EventTime.Time(),
		Pair:       cp,
		AssetType:  assetType,
		Exchange:   e.Name,
		StartTime:  resp.KlineData.StartTime.Time(),
		CloseTime:  resp.KlineData.EndTime.Time(),
		Interval:   resp.KlineData.Interval,
		OpenPrice:  resp.KlineData.OpenPrice.Float64(),
		ClosePrice: resp.KlineData.ClosePrice.Float64(),
		HighPrice:  resp.KlineData.HighPrice.Float64(),
		LowPrice:   resp.KlineData.LowPrice.Float64(),
		Volume:     resp.KlineData.Volume.Float64(),
	}
	return nil
}

func (e *Exchange) processCompositeIndex(respRaw []byte) error {
	var resp UFutureCompositeIndex
	err := json.Unmarshal(respRaw, &resp)
	if err != nil {
		return err
	}
	e.Websocket.DataHandler <- resp
	return nil
}

// bookTickerSymbolsMap used to track symbols whose snapshot is recorded.
var (
	bookTickerSymbolsMap  = map[string]struct{}{}
	bookTickerSymbolsLock sync.Mutex
)

func (e *Exchange) processOrderbookDepthUpdate(respRaw []byte, assetType asset.Item) error {
	var resp FuturesDepthOrderbook
	err := json.Unmarshal(respRaw, &resp)
	if err != nil {
		return err
	}
	cp, err := currency.NewPairFromString(resp.Symbol)
	if err != nil {
		return err
	}
	asks := make(orderbook.Levels, len(resp.Asks))
	bids := make(orderbook.Levels, len(resp.Bids))
	for a := range resp.Asks {
		asks[a].Price, err = strconv.ParseFloat(resp.Asks[a][0], 64)
		if err != nil {
			return err
		}
		asks[a].Amount, err = strconv.ParseFloat(resp.Asks[a][1], 64)
		if err != nil {
			return err
		}
	}
	for b := range resp.Bids {
		bids[b].Price, err = strconv.ParseFloat(resp.Bids[b][0], 64)
		if err != nil {
			return err
		}
		bids[b].Amount, err = strconv.ParseFloat(resp.Bids[b][1], 64)
		if err != nil {
			return err
		}
	}
	bookTickerSymbolsLock.Lock()
	defer bookTickerSymbolsLock.Unlock()
	if _, okay := bookTickerSymbolsMap[resp.Symbol]; !okay {
		bookTickerSymbolsMap[strings.ToUpper(resp.Symbol)] = struct{}{}
		return e.Websocket.Orderbook.LoadSnapshot(&orderbook.Book{
			Bids:         bids,
			Asks:         asks,
			Exchange:     e.Name,
			Pair:         cp,
			Asset:        assetType,
			LastUpdated:  resp.TransactionTime.Time(),
			LastUpdateID: resp.LastUpdateID,
		})
	}
	return e.Websocket.Orderbook.Update(&orderbook.Update{
		UpdateID:   resp.LastUpdateID,
		UpdateTime: resp.TransactionTime.Time(),
		Asset:      asset.USDTMarginedFutures,
		Action:     orderbook.UpdateAction,
		Pair:       cp,
		Asks:       asks,
		Bids:       bids,
	})
}

func (e *Exchange) processAggregateTrade(respRaw []byte, assetType asset.Item) error {
	var resp FuturesAggTrade
	err := json.Unmarshal(respRaw, &resp)
	if err != nil {
		return err
	}
	cp, err := currency.NewPairFromString(resp.Symbol)
	if err != nil {
		return err
	}
	e.Websocket.DataHandler <- []trade.Data{
		{
			TID:          strconv.FormatInt(resp.AggregateTradeID, 10),
			Exchange:     e.Name,
			CurrencyPair: cp,
			AssetType:    assetType,
			Price:        resp.Price.Float64(),
			Amount:       resp.Quantity.Float64(),
			Timestamp:    resp.TradeTime.Time(),
		},
	}
	return nil
}

func extractStreamInfo(resultStream string) string {
	splitStream := strings.Split(resultStream, "@")
	if len(splitStream) < 2 {
		return resultStream
	}
	switch splitStream[1] {
	case "aggTrade", "markPrice", "ticker", "bookTicker", "forceOrder", "depth",
		"compositeIndex", "assetIndex", "miniTicker", "indexPrice":
		return splitStream[1]
	default:
		switch {
		case strings.HasPrefix(splitStream[1], "depth"):
			return "depth"
		case strings.HasPrefix(splitStream[1], "continuousKline"):
			return "continuousKline"
		case strings.HasPrefix(splitStream[0], "!markPrice"):
			return "!markPrice@arr"
		}
	}
	return resultStream
}

func (e *Exchange) processMiniTickers(respRaw []byte, array bool, assetType asset.Item) error {
	if array {
		var resp []FutureMiniTickerPrice
		err := json.Unmarshal(respRaw, &resp)
		if err != nil {
			return err
		}
		tickerPrices, err := e.getMiniTickers(resp, assetType)
		if err != nil {
			return err
		}
		e.Websocket.DataHandler <- tickerPrices
		return nil
	}
	var resp FutureMiniTickerPrice
	err := json.Unmarshal(respRaw, &resp)
	if err != nil {
		return err
	}
	cp, err := currency.NewPairFromString(resp.Symbol)
	if err != nil {
		return err
	}
	e.Websocket.DataHandler <- &ticker.Price{
		Pair:         cp,
		High:         resp.HighPrice.Float64(),
		Low:          resp.LowPrice.Float64(),
		Volume:       resp.Volume.Float64(),
		QuoteVolume:  resp.QuoteVolume.Float64(),
		Open:         resp.OpenPrice.Float64(),
		ExchangeName: e.Name,
		AssetType:    assetType,
		LastUpdated:  resp.EventTime.Time(),
	}
	return nil
}

func (e *Exchange) getMiniTickers(miniTickers []FutureMiniTickerPrice, assetType asset.Item) ([]ticker.Price, error) {
	tickerPrices := make([]ticker.Price, len(miniTickers))
	for i := range miniTickers {
		cp, err := currency.NewPairFromString(miniTickers[i].Symbol)
		if err != nil {
			return nil, err
		}
		tickerPrices[i] = ticker.Price{
			Pair:         cp,
			High:         miniTickers[i].HighPrice.Float64(),
			Low:          miniTickers[i].LowPrice.Float64(),
			Volume:       miniTickers[i].Volume.Float64(),
			QuoteVolume:  miniTickers[i].QuoteVolume.Float64(),
			Open:         miniTickers[i].OpenPrice.Float64(),
			ExchangeName: e.Name,
			AssetType:    assetType,
			LastUpdated:  miniTickers[i].EventTime.Time(),
		}
	}
	return tickerPrices, nil
}

func (e *Exchange) processMarketTicker(respRaw []byte, array bool, assetType asset.Item) error {
	if array {
		var resp []UFutureMarketTicker
		err := json.Unmarshal(respRaw, &resp)
		if err != nil {
			return err
		}
		tickerPrices, err := e.getTickerInfos(resp)
		if err != nil {
			return err
		}
		e.Websocket.DataHandler <- tickerPrices
		return nil
	}
	var resp UFutureMarketTicker
	err := json.Unmarshal(respRaw, &resp)
	if err != nil {
		return err
	}
	cp, err := currency.NewPairFromString(resp.Symbol)
	if err != nil {
		return err
	}
	e.Websocket.DataHandler <- &ticker.Price{
		Pair:         cp,
		Last:         resp.LastPrice.Float64(),
		High:         resp.HighPrice.Float64(),
		Low:          resp.LowPrice.Float64(),
		Volume:       resp.TotalTradeBaseVolume.Float64(),
		QuoteVolume:  resp.TotalQuoteAssetVolume.Float64(),
		Open:         resp.OpenPrice.Float64(),
		ExchangeName: e.Name,
		AssetType:    assetType,
		LastUpdated:  resp.EventTime.Time(),
	}
	return nil
}

func (e *Exchange) getTickerInfos(marketTickers []UFutureMarketTicker) ([]ticker.Price, error) {
	tickerPrices := make([]ticker.Price, len(marketTickers))
	for a := range marketTickers {
		cp, err := currency.NewPairFromString(marketTickers[a].Symbol)
		if err != nil {
			return nil, err
		}
		tickerPrices[a] = ticker.Price{
			Pair:         cp,
			Last:         marketTickers[a].LastPrice.Float64(),
			High:         marketTickers[a].HighPrice.Float64(),
			Low:          marketTickers[a].LowPrice.Float64(),
			Volume:       marketTickers[a].TotalTradeBaseVolume.Float64(),
			QuoteVolume:  marketTickers[a].TotalQuoteAssetVolume.Float64(),
			Open:         marketTickers[a].OpenPrice.Float64(),
			ExchangeName: e.Name,
			AssetType:    asset.USDTMarginedFutures,
			LastUpdated:  marketTickers[a].EventTime.Time(),
		}
	}
	return tickerPrices, nil
}

func (e *Exchange) processBookTicker(respRaw []byte, assetType asset.Item) error {
	var resp FuturesBookTicker
	err := json.Unmarshal(respRaw, &resp)
	if err != nil {
		return err
	}
	cp, err := currency.NewPairFromString(resp.Symbol)
	if err != nil {
		return err
	}
	bookTickerSymbolsLock.Lock()
	defer bookTickerSymbolsLock.Unlock()
	if _, okay := bookTickerSymbolsMap[resp.Symbol]; !okay {
		bookTickerSymbolsMap[strings.ToUpper(resp.Symbol)] = struct{}{}
		return e.Websocket.Orderbook.LoadSnapshot(&orderbook.Book{
			Bids: orderbook.Levels{{
				Amount: resp.BestBidQty.Float64(),
				Price:  resp.BestBidPrice.Float64(),
			}},
			Asks: []orderbook.Level{{
				Amount: resp.BestAskQty.Float64(),
				Price:  resp.BestAskPrice.Float64(),
			}},
			Pair:         cp,
			Exchange:     e.Name,
			Asset:        assetType,
			LastUpdated:  resp.TransactionTime.Time(),
			LastUpdateID: resp.OrderbookUpdateID,
		})
	}
	return e.Websocket.Orderbook.Update(&orderbook.Update{
		UpdateID:   resp.OrderbookUpdateID,
		UpdateTime: resp.TransactionTime.Time(),
		Asset:      assetType,
		Action:     orderbook.UpdateAction,
		Bids: []orderbook.Level{{
			Amount: resp.BestBidQty.Float64(),
			Price:  resp.BestBidPrice.Float64(),
		}},
		Asks: []orderbook.Level{{
			Amount: resp.BestAskQty.Float64(),
			Price:  resp.BestAskPrice.Float64(),
		}},
		Pair: cp,
	})
}

func (e *Exchange) processForceOrder(respRaw []byte, assetType asset.Item) error {
	var resp MarketLiquidationOrder
	err := json.Unmarshal(respRaw, &resp)
	if err != nil {
		return err
	}
	oType, err := order.StringToOrderType(resp.Order.OrderType)
	if err != nil {
		return err
	}
	oSide, err := order.StringToOrderSide(resp.Order.Side)
	if err != nil {
		return err
	}
	oStatus, err := order.StringToOrderStatus(resp.Order.OrderStatus)
	if err != nil {
		return err
	}
	cp, err := currency.NewPairFromString(resp.Order.Symbol)
	if err != nil {
		return err
	}
	e.Websocket.DataHandler <- order.Detail{
		Price:                resp.Order.Price.Float64(),
		Amount:               resp.Order.OriginalQuantity.Float64(),
		AverageExecutedPrice: resp.Order.AveragePrice.Float64(),
		ExecutedAmount:       resp.Order.OrderFilledAccumulatedQuantity.Float64(),
		RemainingAmount:      resp.Order.OriginalQuantity.Float64() - resp.Order.OrderFilledAccumulatedQuantity.Float64(),
		Exchange:             e.Name,
		Type:                 oType,
		Side:                 oSide,
		Status:               oStatus,
		AssetType:            assetType,
		LastUpdated:          resp.Order.OrderTradeTime.Time(),
		Pair:                 cp,
		TimeInForce:          resp.Order.TimeInForce,
	}
	return nil
}

func (e *Exchange) processContractInfoStream(respRaw []byte) error {
	var resp FuturesContractInfo
	err := json.Unmarshal(respRaw, &resp)
	if err != nil {
		return err
	}
	e.Websocket.DataHandler <- resp
	return nil
}

func (e *Exchange) processMultiAssetModeAssetIndexes(respRaw []byte, array bool) error {
	if array {
		var resp []UFuturesAssetIndexUpdate
		err := json.Unmarshal(respRaw, &resp)
		if err != nil {
			return err
		}
		e.Websocket.DataHandler <- &resp
	}
	return nil
}

func (e *Exchange) processMarkPriceUpdate(respRaw []byte, array bool) error {
	if array {
		var resp []FuturesMarkPrice
		err := json.Unmarshal(respRaw, &resp)
		if err != nil {
			return err
		}
		e.Websocket.DataHandler <- resp
	}
	var resp FuturesMarkPrice
	err := json.Unmarshal(respRaw, &resp)
	if err != nil {
		return err
	}
	e.Websocket.DataHandler <- resp
	return nil
}

// SubscribeFutures subscribes to a set of channels
func (e *Exchange) SubscribeFutures(ctx context.Context, conn websocket.Connection, channelsToSubscribe subscription.List) error {
	return e.handleSubscriptions(ctx, conn, "SUBSCRIBE", channelsToSubscribe)
}

// UnsubscribeFutures unsubscribes from a set of channels
func (e *Exchange) UnsubscribeFutures(ctx context.Context, conn websocket.Connection, channelsToUnsubscribe subscription.List) error {
	return e.handleSubscriptions(ctx, conn, "UNSUBSCRIBE", channelsToUnsubscribe)
}

func (e *Exchange) handleSubscriptions(ctx context.Context, conn websocket.Connection, operation string, subscriptionChannels subscription.List) error {
	payload := WsPayload{
		ID:     e.MessageID(),
		Method: operation,
	}
	for i := range subscriptionChannels {
		payload.Params = append(payload.Params, subscriptionChannels[i].Channel)
		if i%50 == 0 && i != 0 {
			err := conn.SendJSONMessage(ctx, request.UnAuth, payload)
			if err != nil {
				return err
			}
			payload.Params = []string{}
			payload.ID = e.MessageID()
		}
	}
	if len(payload.Params) > 0 {
		err := conn.SendJSONMessage(ctx, request.UnAuth, payload)
		if err != nil {
			return err
		}
	}
	if operation == "UNSUBSCRIBE" {
		err := e.Websocket.RemoveSubscriptions(conn, subscriptionChannels...)
		if err != nil {
			return err
		}
	}
	return e.Websocket.AddSuccessfulSubscriptions(conn, subscriptionChannels...)
}

// GenerateUFuturesDefaultSubscriptions generates the default subscription set
func (e *Exchange) GenerateUFuturesDefaultSubscriptions() (subscription.List, error) {
	var subscriptions subscription.List
	pairs, err := e.FetchTradablePairs(context.Background(), asset.USDTMarginedFutures)
	if err != nil {
		return nil, err
	}
	if len(pairs) > 4 {
		pairs = pairs[:3]
	}
	channels := defaultSubscriptions
	for z := range channels {
		var chSubscription *subscription.Subscription
		switch channels[z] {
		case assetIndexAllChan, contractInfoAllChan, forceOrderAllChan,
			bookTickerAllChan, tickerAllChan, miniTickerAllChan, markPriceAllChan:
			if channels[z] == markPriceAllChan {
				channels[z] += "@1s"
			}
			subscriptions = append(subscriptions, &subscription.Subscription{
				Channel: channels[z],
			})
		case aggTradeChan, depthChan, markPriceChan, tickerChan, klineChan,
			miniTickerChan, bookTickersChan, forceOrderChan, compositeIndexChan, assetIndexChan:
			for y := range pairs {
				lp := pairs[y].Lower()
				lp.Delimiter = ""
				chSubscription = &subscription.Subscription{
					Channel: lp.String() + channels[z],
				}
				switch channels[z] {
				case depthChan:
					chSubscription.Channel += "@100ms"
				case klineChan:
					chSubscription.Channel += "_" + getKlineIntervalString(kline.FiveMin)
				}
				subscriptions = append(subscriptions, chSubscription)
			}
		case continuousKline:
			for y := range pairs {
				lp := pairs[y].Lower()
				lp.Delimiter = ""
				chSubscription = &subscription.Subscription{
					// Contract types:"PERPETUAL", "CURRENT_MONTH", "NEXT_MONTH", "CURRENT_QUARTER", "NEXT_QUARTER"
					// by default we are subscribing to PERPETUAL contract types
					Channel: lp.String() + "_PERPETUAL@" + channels[z] + "_" + getKlineIntervalString(kline.FifteenMin),
				}
				subscriptions = append(subscriptions, chSubscription)
			}
		default:
			return nil, errors.New("unsupported subscription")
		}
	}
	return subscriptions, nil
}

// ListSubscriptions retrieves list of subscriptions
func (e *Exchange) ListSubscriptions(ctx context.Context, conn websocket.Connection) ([]string, error) {
	req := &WsPayload{
		ID:     e.MessageID(),
		Method: "LIST_SUBSCRIPTIONS",
	}
	var resp WebsocketActionResponse
	respRaw, err := conn.SendMessageReturnResponse(ctx, request.UnAuth, req.ID, &req)
	if err != nil {
		return nil, err
	}
	return resp.Result, json.Unmarshal(respRaw, &resp)
}

// SetProperty to set a property for the websocket connection you are using.
func (e *Exchange) SetProperty(ctx context.Context, conn websocket.Connection, property string, value any) error {
	// Currently, the only property can be set is to set whether "combined" stream payloads are enabled are not.
	req := &struct {
		ID     int64  `json:"method"`
		Method string `json:"params"`
		Params []any  `json:"id"`
	}{
		ID:     e.MessageSequence(),
		Method: "SET_PROPERTY",
		Params: []any{
			property,
			value,
		},
	}
	var resp WebsocketActionResponse
	respRaw, err := conn.SendMessageReturnResponse(ctx, request.UnAuth, req.ID, &req)
	if err != nil {
		return err
	}
	return json.Unmarshal(respRaw, &resp)
}
