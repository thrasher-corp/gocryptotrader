package binance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
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
func (b *Binance) WsUFuturesConnect() error {
	if !b.Websocket.IsEnabled() || !b.IsEnabled() {
		return stream.ErrWebsocketNotEnabled
	}
	var err error
	var dialer websocket.Dialer
	dialer.HandshakeTimeout = b.Config.HTTPTimeout
	dialer.Proxy = http.ProxyFromEnvironment
	wsURL := binanceUFuturesWebsocketURL + "/stream"
	err = b.Websocket.SetWebsocketURL(wsURL, false, false)
	if err != nil {
		return err
	}
	if b.Websocket.CanUseAuthenticatedEndpoints() {
		listenKey, err = b.GetWsAuthStreamKey(context.TODO())
		switch {
		case err != nil:
			b.Websocket.SetCanUseAuthenticatedEndpoints(false)
			log.Errorf(log.ExchangeSys,
				"%v unable to connect to authenticated Websocket. Error: %s",
				b.Name,
				err)
		default:
			wsURL = binanceUFuturesAuthWebsocketURL + "?streams=" + listenKey
			err = b.Websocket.SetWebsocketURL(wsURL, false, false)
			if err != nil {
				return err
			}
		}
	}
	err = b.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		return fmt.Errorf("%v - Unable to connect to Websocket. Error: %s", b.Name, err)
	}
	b.Websocket.Conn.SetupPingHandler(request.UnAuth, stream.PingHandler{
		UseGorillaHandler: true,
		MessageType:       websocket.PongMessage,
		Delay:             pingDelay,
	})
	b.Websocket.Wg.Add(1)
	go b.wsUFuturesReadData(asset.USDTMarginedFutures)
	return nil
}

// wsUFuturesReadData receives and passes on websocket messages for processing
// for USDT margined instruments.
func (b *Binance) wsUFuturesReadData(assetType asset.Item) {
	defer b.Websocket.Wg.Done()
	for {
		resp := b.Websocket.Conn.ReadMessage()
		if resp.Raw == nil {
			return
		}
		err := b.wsHandleFuturesData(resp.Raw, assetType)
		if err != nil {
			b.Websocket.DataHandler <- err
		}
	}
}

func (b *Binance) wsHandleFuturesData(respRaw []byte, assetType asset.Item) error {
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
		if !b.Websocket.Match.IncomingWithData(result.ID, respRaw) {
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
		return b.processMultiAssetModeAssetIndexes(result.Data, true)
	case contractInfoAllChan:
		return b.processContractInfoStream(result.Data)
	case forceOrderAllChan, "forceOrder":
		return b.processForceOrder(result.Data, assetType)
	case bookTickerAllChan, "bookTicker":
		return b.processBookTicker(result.Data, assetType)
	case tickerAllChan:
		return b.processMarketTicker(result.Data, true, assetType)
	case "ticker":
		return b.processMarketTicker(result.Data, false, assetType)
	case miniTickerAllChan:
		return b.processMiniTickers(result.Data, true, assetType)
	case "miniTicker":
		return b.processMiniTickers(result.Data, false, assetType)
	case "aggTrade":
		return b.processAggregateTrade(result.Data, assetType)
	case "markPrice":
		return b.processMarkPriceUpdate(result.Data, false)
	case "!markPrice@arr":
		return b.processMarkPriceUpdate(result.Data, true)
	case "depth":
		return b.processOrderbookDepthUpdate(result.Data, assetType)
	case "compositeIndex":
		return b.processCompositeIndex(result.Data)
	case continuousKline:
		return b.processContinuousKlineUpdate(result.Data, assetType)
	}
	return fmt.Errorf("unhandled stream data %s", string(respRaw))
}

func (b *Binance) processContinuousKlineUpdate(respRaw []byte, assetType asset.Item) error {
	var resp FutureContinuousKline
	err := json.Unmarshal(respRaw, &resp)
	if err != nil {
		return err
	}
	cp, err := currency.NewPairFromString(resp.Pair)
	if err != nil {
		return err
	}
	b.Websocket.DataHandler <- stream.KlineData{
		Timestamp:  resp.EventTime.Time(),
		Pair:       cp,
		AssetType:  assetType,
		Exchange:   b.Name,
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

func (b *Binance) processCompositeIndex(respRaw []byte) error {
	var resp UFutureCompositeIndex
	err := json.Unmarshal(respRaw, &resp)
	if err != nil {
		return err
	}
	b.Websocket.DataHandler <- resp
	return nil
}

// bookTickerSymbolsMap used to track symbols whose snapshot is recorded.
var (
	bookTickerSymbolsMap  = map[string]struct{}{}
	bookTickerSymbolsLock sync.Mutex
)

func (b *Binance) processOrderbookDepthUpdate(respRaw []byte, assetType asset.Item) error {
	var resp FuturesDepthOrderbook
	err := json.Unmarshal(respRaw, &resp)
	if err != nil {
		return err
	}
	cp, err := currency.NewPairFromString(resp.Symbol)
	if err != nil {
		return err
	}
	asks := make(orderbook.Tranches, len(resp.Asks))
	bids := make(orderbook.Tranches, len(resp.Bids))
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
		return b.Websocket.Orderbook.LoadSnapshot(&orderbook.Base{
			Bids:         bids,
			Asks:         asks,
			Exchange:     b.Name,
			Pair:         cp,
			Asset:        assetType,
			LastUpdated:  resp.TransactionTime.Time(),
			LastUpdateID: resp.LastUpdateID,
		})
	}
	return b.Websocket.Orderbook.Update(&orderbook.Update{
		UpdateID:   resp.LastUpdateID,
		UpdateTime: resp.TransactionTime.Time(),
		Asset:      asset.USDTMarginedFutures,
		Action:     orderbook.Amend,
		Pair:       cp,
		Asks:       asks,
		Bids:       bids,
	})
}

func (b *Binance) processAggregateTrade(respRaw []byte, assetType asset.Item) error {
	var resp FuturesAggTrade
	err := json.Unmarshal(respRaw, &resp)
	if err != nil {
		return err
	}
	cp, err := currency.NewPairFromString(resp.Symbol)
	if err != nil {
		return err
	}
	b.Websocket.DataHandler <- []trade.Data{
		{
			TID:          strconv.FormatInt(resp.AggregateTradeID, 10),
			Exchange:     b.Name,
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

func (b *Binance) processMiniTickers(respRaw []byte, array bool, assetType asset.Item) error {
	if array {
		var resp []FutureMiniTickerPrice
		err := json.Unmarshal(respRaw, &resp)
		if err != nil {
			return err
		}
		tickerPrices, err := b.getMiniTickers(resp, assetType)
		if err != nil {
			return err
		}
		b.Websocket.DataHandler <- tickerPrices
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
	b.Websocket.DataHandler <- &ticker.Price{
		Pair:         cp,
		High:         resp.HighPrice.Float64(),
		Low:          resp.LowPrice.Float64(),
		Volume:       resp.Volume.Float64(),
		QuoteVolume:  resp.QuoteVolume.Float64(),
		Open:         resp.OpenPrice.Float64(),
		ExchangeName: b.Name,
		AssetType:    assetType,
		LastUpdated:  resp.EventTime.Time(),
	}
	return nil
}

func (b *Binance) getMiniTickers(miniTickers []FutureMiniTickerPrice, assetType asset.Item) ([]ticker.Price, error) {
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
			ExchangeName: b.Name,
			AssetType:    assetType,
			LastUpdated:  miniTickers[i].EventTime.Time(),
		}
	}
	return tickerPrices, nil
}

func (b *Binance) processMarketTicker(respRaw []byte, array bool, assetType asset.Item) error {
	if array {
		var resp []UFutureMarketTicker
		err := json.Unmarshal(respRaw, &resp)
		if err != nil {
			return err
		}
		tickerPrices, err := b.getTickerInfos(resp)
		if err != nil {
			return err
		}
		b.Websocket.DataHandler <- tickerPrices
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
	b.Websocket.DataHandler <- &ticker.Price{
		Pair:         cp,
		Last:         resp.LastPrice.Float64(),
		High:         resp.HighPrice.Float64(),
		Low:          resp.LowPrice.Float64(),
		Volume:       resp.TotalTradeBaseVolume.Float64(),
		QuoteVolume:  resp.TotalQuoteAssetVolume.Float64(),
		Open:         resp.OpenPrice.Float64(),
		ExchangeName: b.Name,
		AssetType:    assetType,
		LastUpdated:  resp.EventTime.Time(),
	}
	return nil
}

func (b *Binance) getTickerInfos(marketTickers []UFutureMarketTicker) ([]ticker.Price, error) {
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
			ExchangeName: b.Name,
			AssetType:    asset.USDTMarginedFutures,
			LastUpdated:  marketTickers[a].EventTime.Time(),
		}
	}
	return tickerPrices, nil
}

func (b *Binance) processBookTicker(respRaw []byte, assetType asset.Item) error {
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
		return b.Websocket.Orderbook.LoadSnapshot(&orderbook.Base{
			Bids: orderbook.Tranches{{
				Amount: resp.BestBidQty.Float64(),
				Price:  resp.BestBidPrice.Float64(),
			}},
			Asks: []orderbook.Tranche{{
				Amount: resp.BestAskQty.Float64(),
				Price:  resp.BestAskPrice.Float64(),
			}},
			Pair:         cp,
			Exchange:     b.Name,
			Asset:        assetType,
			LastUpdated:  resp.TransactionTime.Time(),
			LastUpdateID: resp.OrderbookUpdateID,
		})
	}
	return b.Websocket.Orderbook.Update(&orderbook.Update{
		UpdateID:   resp.OrderbookUpdateID,
		UpdateTime: resp.TransactionTime.Time(),
		Asset:      assetType,
		Action:     orderbook.Amend,
		Bids: []orderbook.Tranche{{
			Amount: resp.BestBidQty.Float64(),
			Price:  resp.BestBidPrice.Float64(),
		}},
		Asks: []orderbook.Tranche{{
			Amount: resp.BestAskQty.Float64(),
			Price:  resp.BestAskPrice.Float64(),
		}},
		Pair: cp,
	})
}

func (b *Binance) processForceOrder(respRaw []byte, assetType asset.Item) error {
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
	b.Websocket.DataHandler <- order.Detail{
		Price:                resp.Order.Price.Float64(),
		Amount:               resp.Order.OriginalQuantity.Float64(),
		AverageExecutedPrice: resp.Order.AveragePrice.Float64(),
		ExecutedAmount:       resp.Order.OrderFilledAccumulatedQuantity.Float64(),
		RemainingAmount:      resp.Order.OriginalQuantity.Float64() - resp.Order.OrderFilledAccumulatedQuantity.Float64(),
		Exchange:             b.Name,
		Type:                 oType,
		Side:                 oSide,
		Status:               oStatus,
		AssetType:            assetType,
		LastUpdated:          resp.Order.OrderTradeTime.Time(),
		Pair:                 cp,
	}
	return nil
}

func (b *Binance) processContractInfoStream(respRaw []byte) error {
	var resp FuturesContractInfo
	err := json.Unmarshal(respRaw, &resp)
	if err != nil {
		return err
	}
	b.Websocket.DataHandler <- resp
	return nil
}

func (b *Binance) processMultiAssetModeAssetIndexes(respRaw []byte, array bool) error {
	if array {
		var resp []UFuturesAssetIndexUpdate
		err := json.Unmarshal(respRaw, &resp)
		if err != nil {
			return err
		}
		b.Websocket.DataHandler <- &resp
	}
	return nil
}

func (b *Binance) processMarkPriceUpdate(respRaw []byte, array bool) error {
	if array {
		var resp []FuturesMarkPrice
		err := json.Unmarshal(respRaw, &resp)
		if err != nil {
			return err
		}
		b.Websocket.DataHandler <- resp
	}
	var resp FuturesMarkPrice
	err := json.Unmarshal(respRaw, &resp)
	if err != nil {
		return err
	}
	b.Websocket.DataHandler <- resp
	return nil
}

func (b *Binance) processKline(respRaw []byte) error {
	var resp UFuturesKline
	err := json.Unmarshal(respRaw, &resp)
	if err != nil {
		return err
	}
	splittedEvents := strings.Split(resp.EventType, "@")
	var cp currency.Pair
	cp, err = currency.NewPairFromString(splittedEvents[0])
	if err != nil {
		return err
	}
	b.Websocket.DataHandler <- stream.KlineData{
		Pair:       cp,
		StartTime:  resp.KlineData.StartTime.Time(),
		CloseTime:  resp.KlineData.CloseTime.Time(),
		Timestamp:  resp.EventTime.Time(),
		OpenPrice:  resp.KlineData.OpenPrice.Float64(),
		HighPrice:  resp.KlineData.HighPrice.Float64(),
		LowPrice:   resp.KlineData.LowPrice.Float64(),
		ClosePrice: resp.KlineData.ClosePrice.Float64(),
		Volume:     resp.KlineData.BaseVolume.Float64(),
		AssetType:  asset.USDTMarginedFutures,
		Exchange:   b.Name,
		Interval:   resp.KlineData.Interval,
	}
	return nil
}

func (b *Binance) processDepthUpdate(respRaw []byte) error {
	var resp UFuturesOrderbook
	err := json.Unmarshal(respRaw, &resp)
	if err != nil {
		return err
	}
	cp, err := currency.NewPairFromString(resp.Data.Symbol)
	if err != nil {
		return err
	}
	oUpdate := &orderbook.Update{
		UpdateID:   resp.Data.FinalUpdateIDLastStream,
		UpdateTime: resp.Data.TransactionTime.Time(),
		Asset:      asset.USDTMarginedFutures,
		Action:     orderbook.Amend,
		Bids:       make([]orderbook.Tranche, len(resp.Data.Bids)),
		Asks:       make([]orderbook.Tranche, len(resp.Data.Asks)),
		Pair:       cp,
	}
	for a := range resp.Data.Asks {
		oUpdate.Asks[a].Price = resp.Data.Asks[a][0].Float64()
		oUpdate.Asks[a].Amount = resp.Data.Asks[a][1].Float64()
	}
	for b := range resp.Data.Bids {
		oUpdate.Bids[b].Price = resp.Data.Bids[b][0].Float64()
		oUpdate.Bids[b].Amount = resp.Data.Bids[b][1].Float64()
	}
	return b.Websocket.Orderbook.Update(oUpdate)
}

func (b *Binance) processTrades(respRaw []byte) error {
	var resp UFuturesAggregatedTrade
	err := json.Unmarshal(respRaw, &resp)
	if err != nil {
		return err
	}
	cp, err := currency.NewPairFromString(resp.Symbol)
	if err != nil {
		return err
	}
	return trade.AddTradesToBuffer(b.Name, trade.Data{
		TID:          strconv.FormatInt(resp.LastTradeID, 10),
		Exchange:     b.Name,
		CurrencyPair: cp,
		AssetType:    asset.USDTMarginedFutures,
		Price:        resp.Price.Float64(),
		Amount:       resp.Quantity.Float64(),
		Timestamp:    resp.EventTime.Time(),
	})
}

// SubscribeFutures subscribes to a set of channels
func (b *Binance) SubscribeFutures(channelsToSubscribe subscription.List) error {
	return b.handleSubscriptions("SUBSCRIBE", channelsToSubscribe)
}

// UnsubscribeFutures unsubscribes from a set of channels
func (b *Binance) UnsubscribeFutures(channelsToUnsubscribe subscription.List) error {
	return b.handleSubscriptions("UNSUBSCRIBE", channelsToUnsubscribe)
}

func (b *Binance) handleSubscriptions(operation string, subscriptionChannels subscription.List) error {
	payload := WsPayload{
		ID:     b.Websocket.Conn.GenerateMessageID(false),
		Method: operation,
	}
	for i := range subscriptionChannels {
		payload.Params = append(payload.Params, subscriptionChannels[i].Channel)
		if i%50 == 0 && i != 0 {
			err := b.Websocket.Conn.SendJSONMessage(context.Background(), request.UnAuth, payload)
			if err != nil {
				return err
			}
			payload.Params = []string{}
			payload.ID = b.Websocket.Conn.GenerateMessageID(false)
		}
	}
	if len(payload.Params) > 0 {
		err := b.Websocket.Conn.SendJSONMessage(context.Background(), request.UnAuth, payload)
		if err != nil {
			return err
		}
	}
	if operation == "UNSUBSCRIBE" {
		err := b.Websocket.RemoveSubscriptions(b.Websocket.Conn, subscriptionChannels...)
		if err != nil {
			return err
		}
	}
	return b.Websocket.AddSuccessfulSubscriptions(b.Websocket.Conn, subscriptionChannels...)
}

// GenerateUFuturesDefaultSubscriptions generates the default subscription set
func (b *Binance) GenerateUFuturesDefaultSubscriptions() (subscription.List, error) {
	var channels = defaultSubscriptions
	var subscriptions subscription.List
	pairs, err := b.FetchTradablePairs(context.Background(), asset.USDTMarginedFutures)
	if err != nil {
		return nil, err
	}
	if len(pairs) > 4 {
		pairs = pairs[:3]
	}
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
func (b *Binance) ListSubscriptions() ([]string, error) {
	req := &WsPayload{
		ID:     b.Websocket.Conn.GenerateMessageID(false),
		Method: "LIST_SUBSCRIPTIONS",
	}
	var resp WebsocketActionResponse
	respRaw, err := b.Websocket.Conn.SendMessageReturnResponse(context.Background(), request.UnAuth, req.ID, &req)
	if err != nil {
		return nil, err
	}
	return resp.Result, json.Unmarshal(respRaw, &resp)
}

// SetProperty to set a property for the websocket connection you are using.
func (b *Binance) SetProperty(property string, value interface{}) error {
	// Currently, the only property can be set is to set whether "combined" stream payloads are enabled are not.
	req := &struct {
		ID     int64         `json:"method"`
		Method string        `json:"params"`
		Params []interface{} `json:"id"`
	}{
		ID:     b.Websocket.Conn.GenerateMessageID(false),
		Method: "SET_PROPERTY",
		Params: []interface{}{
			property,
			value,
		},
	}
	var resp WebsocketActionResponse
	respRaw, err := b.Websocket.Conn.SendMessageReturnResponse(context.Background(), request.UnAuth, req.ID, &req)
	if err != nil {
		return err
	}
	return json.Unmarshal(respRaw, &resp)
}
