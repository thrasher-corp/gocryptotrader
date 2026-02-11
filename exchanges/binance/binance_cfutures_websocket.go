package binance

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	gws "github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
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
func (e *Exchange) WsCFutureConnect(ctx context.Context, conn websocket.Connection) error {
	if err := e.CurrencyPairs.IsAssetEnabled(asset.CoinMarginedFutures); err != nil {
		return err
	}

	dialer := gws.Dialer{
		HandshakeTimeout: e.Config.HTTPTimeout,
		Proxy:            http.ProxyFromEnvironment,
	}
	wsURL := binanceCFuturesWebsocketURL + "/stream"
	if err := e.Websocket.SetWebsocketURL(wsURL, false, false); err != nil {
		e.Websocket.SetCanUseAuthenticatedEndpoints(false)
		log.Errorf(log.ExchangeSys,
			"%v unable to connect to authenticated Websocket. Error: %s",
			e.Name,
			err)
	}
	if err := conn.Dial(ctx, &dialer, http.Header{}); err != nil {
		return fmt.Errorf("%v - Unable to connect to Websocket. Error: %s", e.Name, err)
	}
	conn.SetupPingHandler(request.UnAuth, websocket.PingHandler{
		UseGorillaHandler: true,
		MessageType:       gws.PongMessage,
		Delay:             pingDelay,
	})
	return nil
}

// GenerateDefaultCFuturesSubscriptions generates a list of subscription instances.
func (e *Exchange) GenerateDefaultCFuturesSubscriptions() (subscription.List, error) {
	var subscriptions subscription.List
	pairs, err := e.FetchTradablePairs(context.Background(), asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	if len(pairs) > 4 {
		pairs = pairs[:3]
	}
	channels := defaultCFuturesSubscriptions
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

func (e *Exchange) wsHandleCFuturesData(ctx context.Context, respRaw []byte) error {
	result := struct {
		Result json.RawMessage `json:"result"`
		ID     int64           `json:"id"`
		Stream string          `json:"stream"`
		Data   json.RawMessage `json:"data"`
	}{}
	if err := json.Unmarshal(respRaw, &result); err != nil {
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
	case contractInfoAllChan:
		return e.processContractInfoStream(ctx, result.Data)
	case forceOrderAllChan, "forceOrder":
		return e.processCFuturesForceOrder(ctx, result.Data)
	case bookTickerAllChan, "bookTicker":
		return e.processBookTicker(result.Data, asset.CoinMarginedFutures)
	case tickerAllChan:
		return e.processCFuturesMarketTicker(ctx, result.Data, true)
	case "ticker":
		return e.processCFuturesMarketTicker(ctx, result.Data, false)
	case miniTickerAllChan:
		return e.processMiniTickers(ctx, result.Data, true, asset.CoinMarginedFutures)
	case "miniTicker":
		return e.processMiniTickers(ctx, result.Data, false, asset.CoinMarginedFutures)
	case "aggTrade":
		return e.processAggregateTrade(ctx, result.Data, asset.CoinMarginedFutures)
	case "markPrice":
		return e.processMarkPriceUpdate(ctx, result.Data, false)
	case cnlDepth:
		return e.processOrderbookDepthUpdate(result.Data, asset.CoinMarginedFutures)
	case continuousKline:
		return e.processContinuousKlineUpdate(ctx, result.Data, asset.CoinMarginedFutures)
	case klineChan:
		return e.processKlineData(ctx, result.Data)
	case indexPriceCFuturesChan:
		return e.processIndexPrice(ctx, result.Data)
	case indexPriceKlineCFuturesChan,
		markPriceKlineCFuturesChan:
		return e.processMarkPriceKline(ctx, result.Data)
	}
	return fmt.Errorf("unhandled stream data %s", string(respRaw))
}

func (e *Exchange) processCFuturesMarketTicker(ctx context.Context, respRaw []byte, array bool) error {
	if array {
		var resp []CFuturesMarketTicker
		if err := json.Unmarshal(respRaw, &resp); err != nil {
			return err
		}
		tickerPrices, err := e.getCFuturesTickerInfos(resp)
		if err != nil {
			return err
		}
		return e.Websocket.DataHandler.Send(ctx, tickerPrices)
	}
	var resp CFuturesMarketTicker
	if err := json.Unmarshal(respRaw, &resp); err != nil {
		return err
	}
	cp, err := currency.NewPairFromString(resp.Symbol)
	if err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, &ticker.Price{
		Pair:         cp,
		Last:         resp.LastPrice.Float64(),
		High:         resp.HighPrice.Float64(),
		Low:          resp.LowPrice.Float64(),
		Volume:       resp.TotalTradedVolume.Float64(),
		QuoteVolume:  resp.TotalQuoteAssetVolume.Float64(),
		Open:         resp.OpenPrice.Float64(),
		ExchangeName: e.Name,
		AssetType:    asset.CoinMarginedFutures,
		LastUpdated:  resp.EventTime.Time(),
	})
}

func (e *Exchange) getCFuturesTickerInfos(marketTickers []CFuturesMarketTicker) ([]ticker.Price, error) {
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
			AssetType:    asset.CoinMarginedFutures,
			LastUpdated:  marketTickers[a].EventTime.Time(),
		}
	}
	return tickerPrices, nil
}

func (e *Exchange) processKlineData(ctx context.Context, respRaw []byte) error {
	var resp CFutureKlineData
	if err := json.Unmarshal(respRaw, &resp); err != nil {
		return err
	}
	cp, err := currency.NewPairFromString(resp.Symbol)
	if err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, &websocket.KlineData{
		Pair:       cp,
		Exchange:   e.Name,
		Interval:   resp.KlineData.Interval,
		AssetType:  asset.CoinMarginedFutures,
		StartTime:  resp.KlineData.StartTime.Time(),
		CloseTime:  resp.KlineData.CloseTime.Time(),
		OpenPrice:  resp.KlineData.OpenPrice.Float64(),
		ClosePrice: resp.KlineData.ClosePrice.Float64(),
		HighPrice:  resp.KlineData.HighPrice.Float64(),
		LowPrice:   resp.KlineData.LowPrice.Float64(),
		Volume:     resp.KlineData.Volume.Float64(),
	})
}

func (e *Exchange) processIndexPrice(ctx context.Context, respRaw []byte) error {
	var resp CFutureIndexPriceStream
	if err := json.Unmarshal(respRaw, &resp); err != nil {
		return err
	}
	cp, err := currency.NewPairFromString(resp.Pair)
	if err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, &ticker.Price{
		Pair:        cp,
		Last:        resp.IndexPrice.Float64(),
		LastUpdated: resp.EventTime.Time(),
	})
}

func (e *Exchange) processCFuturesForceOrder(ctx context.Context, respRaw []byte) error {
	var resp MarketLiquidationOrder
	if err := json.Unmarshal(respRaw, &resp); err != nil {
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
	return e.Websocket.DataHandler.Send(ctx, &order.Detail{
		Price:                resp.Order.Price.Float64(),
		Amount:               resp.Order.OriginalQuantity.Float64(),
		AverageExecutedPrice: resp.Order.AveragePrice.Float64(),
		ExecutedAmount:       resp.Order.OrderFilledAccumulatedQuantity.Float64(),
		RemainingAmount:      resp.Order.OriginalQuantity.Float64() - resp.Order.OrderFilledAccumulatedQuantity.Float64(),
		Exchange:             e.Name,
		Type:                 oType,
		Side:                 oSide,
		Status:               oStatus,
		AssetType:            asset.CoinMarginedFutures,
		LastUpdated:          resp.Order.OrderTradeTime.Time(),
		TimeInForce:          resp.Order.TimeInForce,
		Pair:                 cp,
	})
}

func (e *Exchange) processMarkPriceKline(ctx context.Context, respRaw []byte) error {
	var resp CFutureMarkOrIndexPriceKline
	if err := json.Unmarshal(respRaw, &resp); err != nil {
		return err
	}
	cp, err := currency.NewPairFromString(resp.Pair)
	if err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, &websocket.KlineData{
		Pair:       cp,
		AssetType:  asset.CoinMarginedFutures,
		Interval:   resp.Kline.Interval,
		StartTime:  resp.Kline.StartTime.Time(),
		CloseTime:  resp.Kline.CloseTime.Time(),
		OpenPrice:  resp.Kline.OpenPrice.Float64(),
		ClosePrice: resp.Kline.ClosePrice.Float64(),
		HighPrice:  resp.Kline.HighPrice.Float64(),
		LowPrice:   resp.Kline.LowPrice.Float64(),
	})
}
