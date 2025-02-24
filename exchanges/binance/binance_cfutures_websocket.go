package binance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	binanceCFuturesWebsocketURL = "wss://dstream.binance.com"
)

var defaultCFuturesSubscriptions = []string{
	depthChan,
	tickerAllChan,
	continuousKline,
	bookTickerAllChan,
}

// WsCFutureConnect initiates a websocket connection to coin margined futures websocket
func (b *Binance) WsCFutureConnect() error {
	if !b.Websocket.IsEnabled() || !b.IsEnabled() {
		return stream.ErrWebsocketNotEnabled
	}
	var err error
	var dialer websocket.Dialer
	dialer.HandshakeTimeout = b.Config.HTTPTimeout
	dialer.Proxy = http.ProxyFromEnvironment
	wsURL := binanceCFuturesWebsocketURL + "/stream"
	err = b.Websocket.SetWebsocketURL(wsURL, false, false)
	if err != nil {
		b.Websocket.SetCanUseAuthenticatedEndpoints(false)
		log.Errorf(log.ExchangeSys,
			"%v unable to connect to authenticated Websocket. Error: %s",
			b.Name,
			err)
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
	go b.wsCFuturesReadData()
	return nil
}

// GenerateDefaultCFuturesSubscriptions generates a list of subscription instances.
func (b *Binance) GenerateDefaultCFuturesSubscriptions() (subscription.List, error) {
	var channels = defaultCFuturesSubscriptions
	var subscriptions subscription.List
	pairs, err := b.FetchTradablePairs(context.Background(), asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	if len(pairs) > 4 {
		pairs = pairs[:3]
	}
	for z := range channels {
		var chSubscription *subscription.Subscription
		switch channels[z] {
		case contractInfoAllChan, forceOrderAllChan,
			bookTickerAllChan, tickerAllChan, miniTickerAllChan:
			subscriptions = append(subscriptions, &subscription.Subscription{
				Channel: channels[z],
			})
		case aggTradeChan, depthChan, markPriceChan, tickerChan,
			klineChan, miniTickerChan, forceOrderChan,
			indexPriceCFuturesChan, bookTickerCFuturesChan:
			for y := range pairs {
				lp := pairs[y].Lower()
				lp.Delimiter = ""
				chSubscription = &subscription.Subscription{
					Channel: lp.String() + channels[z],
				}
				switch channels[z] {
				case depthChan:
					chSubscription.Channel += "@100ms"
				case klineChan, indexPriceKlineCFuturesChan, markPriceKlineCFuturesChan:
					chSubscription.Channel += "_" + getKlineIntervalString(kline.FiveMin)
				}
				subscriptions = append(subscriptions, chSubscription)
			}
		case continuousKline:
			for y := range pairs {
				lp := pairs[y].Lower()
				lp.Delimiter = ""
				chSubscription = &subscription.Subscription{
					// Contract types:""perpetual", "current_quarter", "next_quarter""
					Channel: lp.String() + "_PERPETUAL@" + channels[z] + "_" + getKlineIntervalString(kline.FiveMin),
				}
				subscriptions = append(subscriptions, chSubscription)
			}
		default:
			return nil, subscription.ErrNotSupported
		}
	}
	return subscriptions, nil
}

// wsCFuturesReadData receives and passes on websocket messages for processing
// for Coin margined instruments.
func (b *Binance) wsCFuturesReadData() {
	defer b.Websocket.Wg.Done()
	for {
		resp := b.Websocket.Conn.ReadMessage()
		if resp.Raw == nil {
			return
		}
		err := b.wsHandleCFuturesData(resp.Raw)
		if err != nil {
			b.Websocket.DataHandler <- err
		}
	}
}

func (b *Binance) wsHandleCFuturesData(respRaw []byte) error {
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
	case contractInfoAllChan:
		return b.processContractInfoStream(result.Data)
	case forceOrderAllChan, "forceOrder":
		return b.processCFuturesForceOrder(result.Data)
	case bookTickerAllChan, "bookTicker":
		return b.processBookTicker(result.Data, asset.CoinMarginedFutures)
	case tickerAllChan:
		return b.processCFuturesMarketTicker(result.Data, true)
	case "ticker":
		return b.processCFuturesMarketTicker(result.Data, false)
	case miniTickerAllChan:
		return b.processMiniTickers(result.Data, true, asset.CoinMarginedFutures)
	case "miniTicker":
		return b.processMiniTickers(result.Data, false, asset.CoinMarginedFutures)
	case "aggTrade":
		return b.processAggregateTrade(result.Data, asset.CoinMarginedFutures)
	case "markPrice":
		return b.processMarkPriceUpdate(result.Data, false)
	case cnlDepth:
		return b.processOrderbookDepthUpdate(result.Data, asset.CoinMarginedFutures)
	case continuousKline:
		return b.processContinuousKlineUpdate(result.Data, asset.CoinMarginedFutures)
	case klineChan:
		return b.processKlineData(result.Data)
	case indexPriceCFuturesChan:
		return b.processIndexPrice(result.Data)
	case indexPriceKlineCFuturesChan,
		markPriceKlineCFuturesChan:
		return b.processMarkPriceKline(result.Data)
	}
	return fmt.Errorf("unhandled stream data %s", string(respRaw))
}

func (b *Binance) processCFuturesMarketTicker(respRaw []byte, array bool) error {
	if array {
		var resp []CFuturesMarketTicker
		err := json.Unmarshal(respRaw, &resp)
		if err != nil {
			return err
		}
		tickerPrices, err := b.getCFuturesTickerInfos(resp)
		if err != nil {
			return err
		}
		b.Websocket.DataHandler <- tickerPrices
		return nil
	}
	var resp CFuturesMarketTicker
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
		Volume:       resp.TotalTradedVolume.Float64(),
		QuoteVolume:  resp.TotalQuoteAssetVolume.Float64(),
		Open:         resp.OpenPrice.Float64(),
		ExchangeName: b.Name,
		AssetType:    asset.CoinMarginedFutures,
		LastUpdated:  resp.EventTime.Time(),
	}
	return nil
}

func (b *Binance) getCFuturesTickerInfos(marketTickers []CFuturesMarketTicker) ([]ticker.Price, error) {
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
			AssetType:    asset.CoinMarginedFutures,
			LastUpdated:  marketTickers[a].EventTime.Time(),
		}
	}
	return tickerPrices, nil
}

func (b *Binance) processKlineData(respRaw []byte) error {
	var resp CFutureKlineData
	err := json.Unmarshal(respRaw, &resp)
	if err != nil {
		return err
	}
	cp, err := currency.NewPairFromString(resp.Symbol)
	if err != nil {
		return err
	}
	b.Websocket.DataHandler <- &stream.KlineData{
		Pair:       cp,
		Exchange:   b.Name,
		Interval:   resp.KlineData.Interval,
		AssetType:  asset.CoinMarginedFutures,
		StartTime:  resp.KlineData.StartTime.Time(),
		CloseTime:  resp.KlineData.CloseTime.Time(),
		OpenPrice:  resp.KlineData.OpenPrice.Float64(),
		ClosePrice: resp.KlineData.ClosePrice.Float64(),
		HighPrice:  resp.KlineData.HighPrice.Float64(),
		LowPrice:   resp.KlineData.LowPrice.Float64(),
		Volume:     resp.KlineData.Volume.Float64(),
	}
	return nil
}

func (b *Binance) processIndexPrice(respRaw []byte) error {
	var resp CFutureIndexPriceStream
	err := json.Unmarshal(respRaw, &resp)
	if err != nil {
		return err
	}
	cp, err := currency.NewPairFromString(resp.Pair)
	if err != nil {
		return err
	}
	b.Websocket.DataHandler <- &ticker.Price{
		Pair:        cp,
		Last:        resp.IndexPrice.Float64(),
		LastUpdated: resp.EventTime.Time(),
	}
	return nil
}

func (b *Binance) processCFuturesForceOrder(respRaw []byte) error {
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
		AssetType:            asset.CoinMarginedFutures,
		LastUpdated:          resp.Order.OrderTradeTime.Time(),
		Pair:                 cp,
	}
	return nil
}

func (b *Binance) processMarkPriceKline(respRaw []byte) error {
	var resp CFutureMarkOrIndexPriceKline
	err := json.Unmarshal(respRaw, &resp)
	if err != nil {
		return err
	}
	cp, err := currency.NewPairFromString(resp.Pair)
	if err != nil {
		return err
	}
	b.Websocket.DataHandler <- &stream.KlineData{
		Pair:       cp,
		AssetType:  asset.CoinMarginedFutures,
		Interval:   resp.Kline.Interval,
		StartTime:  resp.Kline.StartTime.Time(),
		CloseTime:  resp.Kline.CloseTime.Time(),
		OpenPrice:  resp.Kline.OpenPrice.Float64(),
		ClosePrice: resp.Kline.ClosePrice.Float64(),
		HighPrice:  resp.Kline.HighPrice.Float64(),
		LowPrice:   resp.Kline.LowPrice.Float64(),
	}
	return nil
}
