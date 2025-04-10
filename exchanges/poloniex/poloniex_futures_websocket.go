package poloniex

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/internal/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	futuresWebsocketPrivateURL = "wss://ws.poloniex.com/ws/v3/private"
	futuresWebsocketPublicURL  = "wss://ws.poloniex.com/ws/v3/public"
)

const (
	cnlFuturesSymbol        = "symbol"
	cnlFuturesOrderbookLvl2 = "book_lv2"
	cnlFuturesOrderbook     = "book"
	cnlFuturesTickers       = "tickers"
	cnlFuturesTrades        = "trades"
	cnlFuturesIndexPrice    = "index_price"
	cnlFuturesMarkPrice     = "mark_price"
	cnlFuturesFundingRate   = "funding_rate"

	cnlFuturesPrivatePositions = "positions"
	cnlFuturesPrivateOrders    = "orders"
	cnlFuturesPrivateTrades    = "trade"
	cnlFuturesAccount          = "account"
)

const (
	candles1Min, candles5Min, candles10Min, candles15Min, candles30Min, candles1Hr, candles2Hr,
	candles4Hr, candles6Hr, candles12Hr, candles1Day, candles3Day, candles1Week, candles1Month = "candles_minute_1", "candles_minute_5", "candles_minute_10", "candles_minute_15", "candles_minute_30", "candles_hour_1",
		"candles_hour_2", "candles_hour_4", "candles_hour_6",
		"candles_hour_12", "candles_day_1", "candles_day_3", "candles_week_1", "candles_month_1"

	markCandles1Min, markCandles5Min, markCandles10Min, markCandles15Min,
	markCandles30Min, markCandles1Hr, markCandles2Hr, markCandles4Hr, markCandles12Hr, markCandles1Day, markCandles3Day, markCandles1Week = "mark_price_candles_minute_1", "mark_price_candles_minute_5", "mark_price_candles_minute_10", "mark_price_candles_minute_15",
		"mark_candles_minute_30", "mark_candles_hour_1", "mark_candles_hour_2", "mark_candles_hour_4", "mark_candles_hour_12",
		"mark_candles_day_1", "mark_candles_day_3", "mark_candles_week_1"

	indexCandles1Min, indexCandles5Min, indexCandles10Min, indexCandles15Min, indexCandles30Min, indexCandles1Hr, indexCandles2Hr, indexCandles4Hr, indexCandles12Hr, indexCandles1Day, indexCandles3Day, indexCandles1Week = "index_candles_minute_1",
		"index_candles_minute_5", "index_candles_minute_10", "index_candles_minute_15", "index_candles_minute_30", "index_candles_hour_1", "index_candles_hour_2", "index_candles_hour_4",
		"index_candles_hour_12", "index_candles_day_1", "index_candles_day_3", "index_candles_week_1"
)

var defaultFuturesChannels = []string{
	cnlFuturesTickers,
	cnlFuturesOrderbook,
	candles15Min,
}

var onceFuturesOrderbook map[string]bool

// WsFuturesConnect establishes a websocket connection to the futures websocket server.
func (p *Poloniex) WsFuturesConnect() error {
	if !p.Websocket.IsEnabled() || !p.IsEnabled() {
		return websocket.ErrWebsocketNotEnabled
	}
	var dialer gws.Dialer
	onceFuturesOrderbook = make(map[string]bool)
	err := p.Websocket.SetWebsocketURL(futuresWebsocketPublicURL, false, false)
	if err != nil {
		return err
	}
	err = p.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}
	p.Websocket.Conn.SetupPingHandler(request.Unset, websocket.PingHandler{
		Delay:       time.Second * 15,
		Message:     []byte(`{"type":"ping"}`),
		MessageType: gws.TextMessage,
	})
	if p.Websocket.CanUseAuthenticatedEndpoints() {
		err = p.AuthConnect()
		if err != nil {
			p.Websocket.SetCanUseAuthenticatedEndpoints(false)
			log.Errorf(log.ExchangeSys, "%v - authentication failed: %v\n", p.Name, err)
		}
	}
	p.Websocket.Wg.Add(1)
	go p.wsFuturesReadData(p.Websocket.Conn)
	return nil
}

// AuthConnect establishes a websocket and authenticates to futures private websocket
func (p *Poloniex) AuthConnect() error {
	creds, err := p.GetCredentials(context.Background())
	if err != nil {
		return err
	}
	var dialer gws.Dialer
	err = p.Websocket.SetWebsocketURL(futuresWebsocketPrivateURL, false, false)
	if err != nil {
		return err
	}
	err = p.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}
	p.Websocket.AuthConn.SetupPingHandler(request.Unset, websocket.PingHandler{
		Delay:       time.Second * 15,
		Message:     []byte(`{"type":"ping"}`),
		MessageType: gws.TextMessage,
	})
	timestamp := time.Now().UnixMilli()
	signatureStrings := "GET\n/ws\nsignTimestamp=" + strconv.FormatInt(timestamp, 10)

	var hmac []byte
	hmac, err = crypto.GetHMAC(crypto.HashSHA256,
		[]byte(signatureStrings),
		[]byte(creds.Secret))
	if err != nil {
		return err
	}
	data, err := p.Websocket.AuthConn.SendMessageReturnResponse(context.Background(), request.UnAuth, "auth", &SubscriptionPayload{
		Event:   "subscribe",
		Channel: []string{"auth"},
		Params: map[string]any{
			"key":           creds.Key,
			"signTimestamp": timestamp,
			"signature":     crypto.Base64Encode(hmac),
		},
	})
	if err != nil {
		return err
	}
	var resp *AuthenticationResponse
	err = json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	if !resp.Data.Success {
		return fmt.Errorf("authentication failed with status code: %s", resp.Data.Message)
	}
	p.Websocket.Wg.Add(1)
	go p.wsFuturesReadData(p.Websocket.AuthConn)
	return nil
}

// wsFuturesReadData handles data from the websocket connection for futures instruments subscriptions.
func (p *Poloniex) wsFuturesReadData(conn websocket.Connection) {
	defer p.Websocket.Wg.Done()
	for {
		resp := conn.ReadMessage()
		if resp.Raw == nil {
			return
		}
		err := p.wsFuturesHandleData(resp.Raw)
		if err != nil {
			p.Websocket.DataHandler <- fmt.Errorf("%s: %w", p.Name, err)
		}
	}
}

func (p *Poloniex) wsFuturesHandleData(respRaw []byte) error {
	var result *FuturesSubscriptionResp
	err := json.Unmarshal(respRaw, &result)
	if err != nil {
		return err
	}
	switch result.Channel {
	case "auth":
		if !p.Websocket.Match.IncomingWithData("auth", respRaw) {
			return fmt.Errorf("could not match data with %s %s", "auth", respRaw)
		}
		return nil
	case cnlFuturesSymbol:
		var resp []ProductInfo
		return p.processData(result.Data, &resp)
	case cnlFuturesOrderbookLvl2,
		cnlFuturesOrderbook:
		return p.processFuturesOrderbook(result.Data, result.Action)
	case candles1Min, candles5Min, candles10Min, candles15Min, candles30Min, candles1Hr, candles2Hr, candles4Hr,
		candles6Hr, candles12Hr, candles1Day, candles3Day, candles1Week, candles1Month:
		interval, err := stringToInterval(strings.Join(strings.Split(result.Channel, "_")[1:], "_"))
		if err != nil {
			return err
		}
		return p.processFuturesCandlesticks(result.Data, interval)
	case cnlFuturesTickers:
		return p.processFuturesTickers(result.Data)
	case cnlFuturesTrades:
		return p.processFuturesTrades(result.Data)
	case cnlFuturesIndexPrice:
		var resp []InstrumentIndexPrice
		return p.processData(result.Data, &resp)
	case cnlFuturesMarkPrice:
		var resp []V3FuturesMarkPrice
		return p.processData(result.Data, &resp)
	case markCandles1Min, markCandles5Min, markCandles10Min, markCandles15Min,
		markCandles30Min, markCandles1Hr, markCandles2Hr, markCandles4Hr, markCandles12Hr, markCandles1Day, markCandles3Day, markCandles1Week,
		// Index Candlestick channels
		indexCandles1Min, indexCandles5Min, indexCandles10Min, indexCandles15Min, indexCandles30Min,
		indexCandles1Hr, indexCandles2Hr, indexCandles4Hr, indexCandles12Hr, indexCandles1Day, indexCandles3Day, indexCandles1Week:
		var interval kline.Interval
		if strings.HasPrefix(result.Channel, "mark_price") {
			interval, err = stringToInterval(strings.Join(strings.Split(result.Channel, "_")[3:], "_"))
		} else {
			interval, err = stringToInterval(strings.Join(strings.Split(result.Channel, "_")[2:], "_"))
		}
		if err != nil {
			return err
		}
		return p.processFuturesMarkAndIndexPriceCandlesticks(result.Data, interval)
	case cnlFuturesFundingRate:
		return p.processFuturesFundingRate(result.Data)
	case cnlFuturesPrivatePositions:
		var resp []V3FuturesPosition
		return p.processData(result.Data, &resp)
	case cnlFuturesPrivateOrders:
		return p.processFuturesOrders(result.Data)
	case cnlFuturesPrivateTrades:
		return p.processFuturesTradeFills(result.Data)
	case cnlFuturesAccount:
		return p.processFuturesAccountData(result.Data)
	default:
		p.Websocket.DataHandler <- websocket.UnhandledMessageWarning{Message: p.Name + websocket.UnhandledMessage + string(respRaw)}
		return fmt.Errorf("%s unhandled message: %s", p.Name, string(respRaw))
	}
}

func (p *Poloniex) processFuturesAccountData(data []byte) error {
	var resp []FuturesAccountBalance
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	accChanges := []account.Change{}
	for a := range resp {
		for b := range resp[a].Details {
			accChanges = append(accChanges, account.Change{
				Exchange: p.Name,
				Currency: currency.NewCode(resp[a].Details[b].Currency),
				Asset:    asset.Futures,
				Amount:   resp[a].Details[b].Available.Float64(),
			})
		}
	}
	p.Websocket.DataHandler <- accChanges
	return nil
}

func (p *Poloniex) processFuturesTradeFills(data []byte) error {
	var resp []FuturesTradeFill
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	tfills := make([]fill.Data, len(resp))
	for a := range resp {
		pair, err := currency.NewPairFromString(resp[a].Symbol)
		if err != nil {
			return err
		}
		oSide, err := order.StringToOrderSide(resp[a].Side)
		if err != nil {
			return err
		}
		tfills[a] = fill.Data{
			CurrencyPair:  pair,
			Side:          oSide,
			Exchange:      p.Name,
			AssetType:     asset.Futures,
			OrderID:       resp[a].OrderID,
			ID:            resp[a].TradeID,
			TradeID:       resp[a].TradeID,
			ClientOrderID: resp[a].ClientOrderID,
			Timestamp:     resp[a].UpdateTime.Time(),
			Price:         resp[a].FillPrice.Float64(),
			Amount:        resp[a].FillQuantity.Float64(),
		}
	}
	p.Websocket.DataHandler <- tfills
	return nil
}

func (p *Poloniex) processFuturesOrders(data []byte) error {
	var resp []FuturesV3OrderDetail
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	orders := make([]order.Detail, len(resp))
	for o := range resp {
		oType, err := order.StringToOrderType(resp[o].OrderType)
		if err != nil {
			return err
		}
		oSide, err := order.StringToOrderSide(resp[o].Side)
		if err != nil {
			return err
		}
		oStatus, err := order.StringToOrderStatus(resp[o].State)
		if err != nil {
			return err
		}
		pair, err := currency.NewPairFromString(resp[o].Symbol)
		if err != nil {
			return err
		}
		orders[o] = order.Detail{
			ReduceOnly:           resp[o].ReduceOnly,
			Leverage:             resp[o].Leverage.Float64(),
			Price:                resp[o].Price.Float64(),
			Amount:               resp[o].Size.Float64(),
			TriggerPrice:         resp[o].TakeProfitTriggerPrice.Float64(),
			AverageExecutedPrice: resp[o].AveragePrice.Float64(),
			ExecutedAmount:       resp[o].ExecQuantity.Float64(),
			RemainingAmount:      resp[o].Size.Float64() - resp[o].ExecQuantity.Float64(),
			Fee:                  resp[o].FeeAmount.Float64(),
			FeeAsset:             currency.NewCode(resp[o].FeeCurrency),
			Exchange:             p.Name,
			OrderID:              resp[o].OrderID,
			ClientOrderID:        resp[o].ClientOrderID,
			Type:                 oType,
			Side:                 oSide,
			Status:               oStatus,
			AssetType:            asset.Futures,
			Date:                 resp[o].CreationTime.Time(),
			Pair:                 pair,
		}
	}
	p.Websocket.DataHandler <- orders
	return nil
}

func (p *Poloniex) processFuturesFundingRate(data []byte) error {
	var resp []V3FuturesFundingRate
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}

	for a := range resp {
		pair, err := currency.NewPairFromString(resp[a].Symbol)
		if err != nil {
			return err
		}
		p.Websocket.DataHandler <- websocket.FundingData{
			Timestamp:    resp[a].Timestamp.Time(),
			CurrencyPair: pair,
			AssetType:    asset.Futures,
			Exchange:     p.Name,
			Rate:         resp[a].FundingRate.Float64(),
		}
	}
	return nil
}

func (p *Poloniex) processFuturesMarkAndIndexPriceCandlesticks(data []byte, interval kline.Interval) error {
	var resp []V3WsFuturesMarkAndIndexPriceCandle
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}

	candles := make([]websocket.KlineData, len(resp))
	for a := range resp {
		pair, err := currency.NewPairFromString(resp[a].Symbol)
		if err != nil {
			return err
		}
		candles[a] = websocket.KlineData{
			Timestamp:  resp[a].PushTimestamp.Time(),
			Pair:       pair,
			AssetType:  asset.Futures,
			Exchange:   p.Name,
			StartTime:  resp[a].StartTime.Time(),
			CloseTime:  resp[a].EndTime.Time(),
			Interval:   interval.String(),
			OpenPrice:  resp[a].OpeningPrice.Float64(),
			ClosePrice: resp[a].ClosingPrice.Float64(),
			HighPrice:  resp[a].HighestPrice.Float64(),
			LowPrice:   resp[a].LowestPrice.Float64(),
		}
	}
	p.Websocket.DataHandler <- candles
	return nil
}

func (p *Poloniex) processData(data []byte, respStruct interface{}) error {
	err := json.Unmarshal(data, &respStruct)
	if err != nil {
		return err
	}
	p.Websocket.DataHandler <- respStruct
	return nil
}

func (p *Poloniex) processFuturesOrderbook(data []byte, action string) error {
	var resp []FuturesV3Orderbook
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	for x := range resp {
		pair, err := currency.NewPairFromString(resp[x].Symbol)
		if err != nil {
			return err
		}
		asks := make([]orderbook.Tranche, len(resp[x].Asks))
		for a := range resp[x].Asks {
			asks[a].Price = resp[x].Asks[a][0].Float64()
			asks[a].Amount = resp[x].Asks[a][1].Float64()
		}
		bids := make([]orderbook.Tranche, len(resp[x].Bids))
		for a := range resp[x].Bids {
			bids[a].Price = resp[x].Bids[a][0].Float64()
			bids[a].Amount = resp[x].Bids[a][1].Float64()
		}
		_, okay := onceFuturesOrderbook[resp[x].Symbol]
		if !okay || action == "snapshot" {
			if onceFuturesOrderbook == nil {
				onceFuturesOrderbook = make(map[string]bool)
			}
			onceFuturesOrderbook[resp[x].Symbol] = true
			err = p.Websocket.Orderbook.LoadSnapshot(&orderbook.Base{
				Bids:         bids,
				Asks:         asks,
				Exchange:     p.Name,
				Pair:         pair,
				Asset:        asset.Futures,
				LastUpdated:  resp[x].CreationTime.Time(),
				LastUpdateID: resp[x].ID.Int64(),
			})
			if err != nil {
				return err
			}
			continue
		}
		err = p.Websocket.Orderbook.Update(&orderbook.Update{
			UpdateID:       resp[x].ID.Int64(),
			UpdateTime:     resp[x].CreationTime.Time(),
			UpdatePushedAt: resp[x].Timestamp.Time(),
			Asset:          asset.Futures,
			Action:         orderbook.UpdateInsert,
			Bids:           bids,
			Asks:           asks,
			Pair:           pair,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *Poloniex) processFuturesTickers(data []byte) error {
	var resp []V3FuturesTickerDetail
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	tickerPrices := make([]ticker.Price, len(resp))
	for a := range resp {
		pair, err := currency.NewPairFromString(resp[a].Symbol)
		if err != nil {
			return err
		}
		tickerPrices[a] = ticker.Price{
			High:        resp[a].HighPrice.Float64(),
			Low:         resp[a].LowPrice.Float64(),
			Bid:         resp[a].BestBidPrice.Float64(),
			BidSize:     resp[a].BestBidSize.Float64(),
			Ask:         resp[a].BestAskPrice.Float64(),
			AskSize:     resp[a].BestAskSize.Float64(),
			Volume:      resp[a].Quantity.Float64(),
			QuoteVolume: resp[a].Amount.Float64(),
			// PriceATH
			Open:         resp[a].OpeningPrice.Float64(),
			Close:        resp[a].ClosingPrice.Float64(),
			MarkPrice:    resp[a].MarkPrice.Float64(),
			Pair:         pair,
			ExchangeName: p.Name,
			AssetType:    asset.Futures,
			LastUpdated:  resp[a].Timestamp.Time(),
		}
	}
	p.Websocket.DataHandler <- tickerPrices
	return nil
}

// processFuturesTrades handles latest trading data for this product, including the latest price, trading volume, trading direction, etc.
func (p *Poloniex) processFuturesTrades(data []byte) error {
	var resp []FuturesTrades
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	trades := make([]trade.Data, len(resp))
	for t := range resp {
		pair, err := currency.NewPairFromString(resp[t].Symbol)
		if err != nil {
			return err
		}
		oSide, err := order.StringToOrderSide(resp[t].Side)
		if err != nil {
			return err
		}
		trades[t] = trade.Data{
			TID:          trades[t].TID,
			Exchange:     p.Name,
			CurrencyPair: pair,
			AssetType:    asset.Futures,
			Side:         oSide,
			Price:        resp[t].Price.Float64(),
			Amount:       resp[t].Amount.Float64(),
			Timestamp:    resp[t].Timestamp.Time(),
		}
	}
	p.Websocket.DataHandler <- trades
	return nil
}

func (p *Poloniex) processFuturesCandlesticks(data []byte, interval kline.Interval) error {
	var resp []WsFuturesCandlesctick
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}

	candles := make([]websocket.KlineData, len(resp))
	for a := range resp {
		pair, err := currency.NewPairFromString(resp[a].Symbol)
		if err != nil {
			return err
		}
		candles[a] = websocket.KlineData{
			Timestamp:  resp[a].PushTime.Time(),
			Pair:       pair,
			AssetType:  asset.Futures,
			Exchange:   p.Name,
			StartTime:  resp[a].StartTime.Time(),
			CloseTime:  resp[a].EndTime.Time(),
			Interval:   interval.String(),
			OpenPrice:  resp[a].OpenPrice.Float64(),
			ClosePrice: resp[a].ClosePrice.Float64(),
			HighPrice:  resp[a].HighestPrice.Float64(),
			LowPrice:   resp[a].LowestPrice.Float64(),
			Volume:     resp[a].Amount.Float64(),
		}
	}
	p.Websocket.DataHandler <- candles
	return nil
}

// ------------------------------------------------------------------------------------------------

// GenerateFuturesDefaultSubscriptions adds default subscriptions to futures websockets.
func (p *Poloniex) GenerateFuturesDefaultSubscriptions() (subscription.List, error) {
	enabledPairs, err := p.GetEnabledPairs(asset.Futures)
	if err != nil {
		return nil, err
	}
	channels := defaultFuturesChannels
	subscriptions := subscription.List{}
	for i := range channels {
		switch channels[i] {
		case cnlFuturesPrivatePositions,
			cnlFuturesPrivateOrders,
			cnlFuturesPrivateTrades,
			cnlFuturesAccount:
			subscriptions = append(subscriptions, &subscription.Subscription{
				Channel:       channels[i],
				Asset:         asset.Futures,
				Authenticated: true,
			})
		case cnlFuturesSymbol,
			cnlFuturesOrderbookLvl2,
			cnlFuturesOrderbook,
			cnlFuturesTickers,
			cnlFuturesTrades,
			cnlFuturesIndexPrice,
			cnlFuturesMarkPrice,
			indexCandles1Min, indexCandles5Min, indexCandles10Min, indexCandles15Min, indexCandles30Min, indexCandles1Hr, indexCandles2Hr, indexCandles4Hr, indexCandles12Hr, indexCandles1Day, indexCandles3Day, indexCandles1Week,
			cnlFuturesFundingRate:
			subscriptions = append(subscriptions, &subscription.Subscription{
				Channel: channels[i],
				Asset:   asset.Futures,
				Pairs:   enabledPairs,
			})
		}
	}
	return subscriptions, nil
}

func (p *Poloniex) handleFuturesSubscriptions(operation string, subscs subscription.List) []FuturesSubscriptionInput {
	payloads := []FuturesSubscriptionInput{}
	for x := range subscs {
		if len(subscs[x].Pairs) == 0 {
			input := FuturesSubscriptionInput{
				ID:    strconv.FormatInt(p.Websocket.Conn.GenerateMessageID(false), 10),
				Type:  operation,
				Topic: subscs[x].Channel,
			}
			payloads = append(payloads, input)
		} else {
			for i := range subscs[x].Pairs {
				input := FuturesSubscriptionInput{
					ID:    strconv.FormatInt(p.Websocket.Conn.GenerateMessageID(false), 10),
					Type:  operation,
					Topic: subscs[x].Channel,
				}
				if !subscs[x].Pairs[x].IsEmpty() {
					input.Topic += ":" + subscs[x].Pairs[i].String()
				}
				payloads = append(payloads, input)
			}
		}
	}
	return payloads
}

// SubscribeFutures sends a websocket message to receive data from the channel
func (p *Poloniex) SubscribeFutures(subs subscription.List) error {
	payloads := p.handleFuturesSubscriptions("subscribe", subs)
	var err error
	for i := range payloads {
		err = p.Websocket.Conn.SendJSONMessage(context.Background(), request.UnAuth, payloads[i])
		if err != nil {
			return err
		}
	}
	return p.Websocket.AddSuccessfulSubscriptions(p.Websocket.Conn, subs...)
}

// UnsubscribeFutures sends a websocket message to stop receiving data from the channel
func (p *Poloniex) UnsubscribeFutures(unsub subscription.List) error {
	payloads := p.handleFuturesSubscriptions("unsubscribe", unsub)
	var err error
	for i := range payloads {
		err = p.Websocket.Conn.SendJSONMessage(context.Background(), request.UnAuth, payloads[i])
		if err != nil {
			return err
		}
	}
	return p.Websocket.RemoveSubscriptions(p.Websocket.Conn, unsub...)
}

// ----------------------------------------------------------------
// Configuration update based on Gks

// generateSubscriptions returns a list of subscriptions from the configured subscriptions feature
func (p *Poloniex) generateSubscriptions() (subscription.List, error) {
	return p.Features.Subscriptions.ExpandTemplates(p)
}
