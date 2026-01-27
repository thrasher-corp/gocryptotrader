package coinbase

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"text/template"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	coinbaseWebsocketURL = "wss://advanced-trade-ws.coinbase.com"
)

var subscriptionNames = map[string]string{
	subscription.HeartbeatChannel: "heartbeats",
	subscription.TickerChannel:    "ticker",
	subscription.CandlesChannel:   "candles",
	subscription.AllTradesChannel: "market_trades",
	subscription.OrderbookChannel: "level2",
	subscription.MyAccountChannel: "user",
	"status":                      "status",
	"ticker_batch":                "ticker_batch",
	/* Not Implemented:
	"futures_balance_summary":                "futures_balance_summary",
	*/
}

var defaultSubscriptions = subscription.List{
	{Enabled: true, Channel: subscription.HeartbeatChannel},
	{Enabled: true, Asset: asset.All, Channel: "status"},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.CandlesChannel},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.AllTradesChannel},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.OrderbookChannel},
	{Enabled: true, Asset: asset.All, Channel: subscription.MyAccountChannel, Authenticated: true},
	{Enabled: true, Asset: asset.Spot, Channel: "ticker_batch"},
	/* Not Implemented:
	{Enabled: false, Asset: asset.Spot, Channel: "futures_balance_summary", Authenticated: true},
	*/
}

// WsConnect initiates a websocket connection
func (e *Exchange) WsConnect() error {
	ctx := context.TODO()
	if !e.Websocket.IsEnabled() || !e.IsEnabled() {
		return websocket.ErrWebsocketNotEnabled
	}
	var dialer gws.Dialer
	if err := e.Websocket.Conn.Dial(ctx, &dialer, http.Header{}); err != nil {
		return err
	}
	e.Websocket.Wg.Add(1)
	go e.wsReadData(ctx)
	return nil
}

// wsReadData receives and passes on websocket messages for processing
func (e *Exchange) wsReadData(ctx context.Context) {
	defer e.Websocket.Wg.Done()
	var seqCount uint64
	for {
		resp := e.Websocket.Conn.ReadMessage()
		if resp.Raw == nil {
			return
		}
		sequence, err := e.wsHandleData(ctx, resp.Raw)
		if err != nil {
			if errSend := e.Websocket.DataHandler.Send(ctx, err); errSend != nil {
				log.Errorf(log.WebsocketMgr, "%s %s: %s %s", e.Name, e.Websocket.Conn.GetURL(), errSend, err)
			}
		}
		if sequence != nil {
			if *sequence != seqCount {
				err := fmt.Errorf("%w: received %v, expected %v", errOutOfSequence, sequence, seqCount)
				if errSend := e.Websocket.DataHandler.Send(ctx, err); errSend != nil {
					log.Errorf(log.WebsocketMgr, "%s %s: %s %s", e.Name, e.Websocket.Conn.GetURL(), errSend, err)
				}
				seqCount = *sequence
			}
			seqCount++
		}
	}
}

// wsProcessTicker handles ticker data from the websocket
func (e *Exchange) wsProcessTicker(ctx context.Context, resp *StandardWebsocketResponse) error {
	var wsTickers []WebsocketTickerHolder
	if err := json.Unmarshal(resp.Events, &wsTickers); err != nil {
		return err
	}
	var allTickers []ticker.Price
	aliases := e.pairAliases.GetAliases()
	for i := range wsTickers {
		for j := range wsTickers[i].Tickers {
			symbolAliases := aliases[wsTickers[i].Tickers[j].ProductID]
			t := ticker.Price{
				LastUpdated:  resp.Timestamp,
				AssetType:    asset.Spot,
				ExchangeName: e.Name,
				High:         wsTickers[i].Tickers[j].High24H.Float64(),
				Low:          wsTickers[i].Tickers[j].Low24H.Float64(),
				Last:         wsTickers[i].Tickers[j].Price.Float64(),
				Volume:       wsTickers[i].Tickers[j].Volume24H.Float64(),
				Bid:          wsTickers[i].Tickers[j].BestBid.Float64(),
				BidSize:      wsTickers[i].Tickers[j].BestBidQuantity.Float64(),
				Ask:          wsTickers[i].Tickers[j].BestAsk.Float64(),
				AskSize:      wsTickers[i].Tickers[j].BestAskQuantity.Float64(),
			}
			var errs error
			for k := range symbolAliases {
				if isEnabled, err := e.CurrencyPairs.IsPairEnabled(symbolAliases[k], asset.Spot); err != nil {
					errs = common.AppendError(errs, err)
					continue
				} else if isEnabled {
					t.Pair = symbolAliases[k]
					allTickers = append(allTickers, t)
				}
			}
		}
	}
	return e.Websocket.DataHandler.Send(ctx, allTickers)
}

// wsProcessCandle handles candle data from the websocket
func (e *Exchange) wsProcessCandle(ctx context.Context, resp *StandardWebsocketResponse) error {
	var wsCandles []WebsocketCandleHolder
	if err := json.Unmarshal(resp.Events, &wsCandles); err != nil {
		return err
	}
	var allCandles []websocket.KlineData
	for i := range wsCandles {
		for j := range wsCandles[i].Candles {
			allCandles = append(allCandles, websocket.KlineData{
				Timestamp:  resp.Timestamp,
				Pair:       wsCandles[i].Candles[j].ProductID,
				AssetType:  asset.Spot,
				Exchange:   e.Name,
				StartTime:  wsCandles[i].Candles[j].Start.Time(),
				OpenPrice:  wsCandles[i].Candles[j].Open.Float64(),
				ClosePrice: wsCandles[i].Candles[j].Close.Float64(),
				HighPrice:  wsCandles[i].Candles[j].High.Float64(),
				LowPrice:   wsCandles[i].Candles[j].Low.Float64(),
				Volume:     wsCandles[i].Candles[j].Volume.Float64(),
			})
		}
	}
	return e.Websocket.DataHandler.Send(ctx, allCandles)
}

// wsProcessMarketTrades handles market trades data from the websocket
func (e *Exchange) wsProcessMarketTrades(ctx context.Context, resp *StandardWebsocketResponse) error {
	var wsTrades []WebsocketMarketTradeHolder
	if err := json.Unmarshal(resp.Events, &wsTrades); err != nil {
		return err
	}
	var allTrades []trade.Data
	for i := range wsTrades {
		for j := range wsTrades[i].Trades {
			allTrades = append(allTrades, trade.Data{
				TID:          wsTrades[i].Trades[j].TradeID,
				Exchange:     e.Name,
				CurrencyPair: wsTrades[i].Trades[j].ProductID,
				AssetType:    asset.Spot,
				Side:         wsTrades[i].Trades[j].Side,
				Price:        wsTrades[i].Trades[j].Price.Float64(),
				Amount:       wsTrades[i].Trades[j].Size.Float64(),
				Timestamp:    wsTrades[i].Trades[j].Time,
			})
		}
	}
	return e.Websocket.DataHandler.Send(ctx, allTrades)
}

// wsProcessL2 handles l2 orderbook data from the websocket
func (e *Exchange) wsProcessL2(resp *StandardWebsocketResponse) error {
	var wsL2 []WebsocketOrderbookDataHolder
	err := json.Unmarshal(resp.Events, &wsL2)
	if err != nil {
		return err
	}
	for i := range wsL2 {
		switch wsL2[i].Type {
		case "snapshot":
			err = e.ProcessSnapshot(&wsL2[i], resp.Timestamp)
		case "update":
			err = e.ProcessUpdate(&wsL2[i], resp.Timestamp)
		default:
			err = fmt.Errorf("%w %v", errUnknownL2DataType, wsL2[i].Type)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// wsProcessUser handles user data from the websocket
func (e *Exchange) wsProcessUser(ctx context.Context, resp *StandardWebsocketResponse) error {
	var wsUser []WebsocketOrderDataHolder
	err := json.Unmarshal(resp.Events, &wsUser)
	if err != nil {
		return err
	}
	var allOrders []order.Detail
	for i := range wsUser {
		for j := range wsUser[i].Orders {
			var oType order.Type
			if oType, err = stringToStandardType(wsUser[i].Orders[j].OrderType); err != nil {
				return err
			}
			var oSide order.Side
			if oSide, err = order.StringToOrderSide(wsUser[i].Orders[j].OrderSide); err != nil {
				return err
			}
			var oStatus order.Status
			if oStatus, err = statusToStandardStatus(wsUser[i].Orders[j].Status); err != nil {
				return err
			}
			price := wsUser[i].Orders[j].AveragePrice
			if wsUser[i].Orders[j].LimitPrice != 0 {
				price = wsUser[i].Orders[j].LimitPrice
			}
			var assetType asset.Item
			if assetType, err = stringToStandardAsset(wsUser[i].Orders[j].ProductType); err != nil {
				return err
			}
			var tif order.TimeInForce
			if tif, err = strategyDecoder(wsUser[i].Orders[j].TimeInForce); err != nil {
				return err
			}
			if wsUser[i].Orders[j].PostOnly {
				tif |= order.PostOnly
			}
			allOrders = append(allOrders, order.Detail{
				Price:           price.Float64(),
				ClientOrderID:   wsUser[i].Orders[j].ClientOrderID,
				ExecutedAmount:  wsUser[i].Orders[j].CumulativeQuantity.Float64(),
				RemainingAmount: wsUser[i].Orders[j].LeavesQuantity.Float64(),
				Amount:          wsUser[i].Orders[j].CumulativeQuantity.Float64() + wsUser[i].Orders[j].LeavesQuantity.Float64(),
				OrderID:         wsUser[i].Orders[j].OrderID,
				Side:            oSide,
				Type:            oType,
				Pair:            wsUser[i].Orders[j].ProductID,
				AssetType:       assetType,
				Status:          oStatus,
				TriggerPrice:    wsUser[i].Orders[j].StopPrice.Float64(),
				TimeInForce:     tif,
				Fee:             wsUser[i].Orders[j].TotalFees.Float64(),
				Date:            wsUser[i].Orders[j].CreationTime,
				CloseTime:       wsUser[i].Orders[j].EndTime,
				Exchange:        e.Name,
			})
		}
		for j := range wsUser[i].Positions.PerpetualFuturesPositions {
			var oSide order.Side
			if oSide, err = order.StringToOrderSide(wsUser[i].Positions.PerpetualFuturesPositions[j].PositionSide); err != nil {
				return err
			}
			var mType margin.Type
			if mType, err = margin.StringToMarginType(wsUser[i].Positions.PerpetualFuturesPositions[j].MarginType); err != nil {
				return err
			}
			allOrders = append(allOrders, order.Detail{
				Pair:       wsUser[i].Positions.PerpetualFuturesPositions[j].ProductID,
				Side:       oSide,
				MarginType: mType,
				Amount:     wsUser[i].Positions.PerpetualFuturesPositions[j].NetSize.Float64(),
				Leverage:   wsUser[i].Positions.PerpetualFuturesPositions[j].Leverage.Float64(),
				AssetType:  asset.Futures,
				Exchange:   e.Name,
			})
		}
		for j := range wsUser[i].Positions.ExpiringFuturesPositions {
			var oSide order.Side
			if oSide, err = order.StringToOrderSide(wsUser[i].Positions.ExpiringFuturesPositions[j].Side); err != nil {
				return err
			}
			allOrders = append(allOrders, order.Detail{
				Pair:           wsUser[i].Positions.ExpiringFuturesPositions[j].ProductID,
				Side:           oSide,
				ContractAmount: wsUser[i].Positions.ExpiringFuturesPositions[j].NumberOfContracts.Float64(),
				Price:          wsUser[i].Positions.ExpiringFuturesPositions[j].EntryPrice.Float64(),
			})
		}
	}
	return e.Websocket.DataHandler.Send(ctx, allOrders)
}

// wsHandleData handles all the websocket data coming from the websocket connection
func (e *Exchange) wsHandleData(ctx context.Context, respRaw []byte) (*uint64, error) {
	var resp StandardWebsocketResponse
	if err := json.Unmarshal(respRaw, &resp); err != nil {
		return nil, err
	}
	if resp.Error != "" {
		return &resp.Sequence, errors.New(resp.Error)
	}
	switch resp.Channel {
	case "subscriptions", "heartbeats":
		return &resp.Sequence, nil
	case "status":
		var wsStatus []WebsocketProductHolder
		if err := json.Unmarshal(resp.Events, &wsStatus); err != nil {
			return &resp.Sequence, err
		}
		return &resp.Sequence, e.Websocket.DataHandler.Send(ctx, wsStatus)
	case "ticker", "ticker_batch":
		if err := e.wsProcessTicker(ctx, &resp); err != nil {
			return &resp.Sequence, err
		}
	case "candles":
		if err := e.wsProcessCandle(ctx, &resp); err != nil {
			return &resp.Sequence, err
		}
	case "market_trades":
		if err := e.wsProcessMarketTrades(ctx, &resp); err != nil {
			return &resp.Sequence, err
		}
	case "l2_data":
		if err := e.wsProcessL2(&resp); err != nil {
			return &resp.Sequence, err
		}
	case "user":
		if err := e.wsProcessUser(ctx, &resp); err != nil {
			return &resp.Sequence, err
		}
	default:
		return &resp.Sequence, errChannelNameUnknown
	}
	return &resp.Sequence, nil
}

// ProcessSnapshot processes the initial orderbook snap shot
func (e *Exchange) ProcessSnapshot(snapshot *WebsocketOrderbookDataHolder, timestamp time.Time) error {
	bids, asks, err := processBidAskArray(snapshot, true)
	if err != nil {
		return err
	}
	book := &orderbook.Book{
		Bids:              bids,
		Asks:              asks,
		Exchange:          e.Name,
		Pair:              snapshot.ProductID,
		Asset:             asset.Spot,
		LastUpdated:       timestamp,
		ValidateOrderbook: e.ValidateOrderbook,
	}
	for _, a := range e.pairAliases.GetAlias(snapshot.ProductID) {
		isEnabled, err := e.IsPairEnabled(a, asset.Spot)
		if err != nil {
			return err
		}
		if isEnabled {
			book.Pair = a
			if err := e.Websocket.Orderbook.LoadSnapshot(book); err != nil {
				return err
			}
		}
	}
	return nil
}

// ProcessUpdate updates the orderbook local cache
func (e *Exchange) ProcessUpdate(update *WebsocketOrderbookDataHolder, timestamp time.Time) error {
	bids, asks, err := processBidAskArray(update, false)
	if err != nil {
		return err
	}
	obU := &orderbook.Update{
		Bids:       bids,
		Asks:       asks,
		Pair:       update.ProductID,
		UpdateTime: timestamp,
		Asset:      asset.Spot,
	}
	for _, a := range e.pairAliases.GetAlias(update.ProductID) {
		isEnabled, err := e.IsPairEnabled(a, asset.Spot)
		if err != nil {
			return err
		}
		if isEnabled {
			obU.Pair = a
			if err := e.Websocket.Orderbook.Update(obU); err != nil {
				return err
			}
		}
	}
	return nil
}

// generateSubscriptions adds default subscriptions to websocket to be handled by ManageSubscriptions()
func (e *Exchange) generateSubscriptions() (subscription.List, error) {
	return e.Features.Subscriptions.ExpandTemplates(e)
}

// GetSubscriptionTemplate returns a subscription channel template
func (e *Exchange) GetSubscriptionTemplate(_ *subscription.Subscription) (*template.Template, error) {
	return template.New("master.tmpl").Funcs(template.FuncMap{"channelName": channelName}).Parse(subTplText)
}

// Subscribe sends a websocket message to receive data from a list of channels
func (e *Exchange) Subscribe(subs subscription.List) error {
	return e.ParallelChanOp(context.TODO(), subs, func(ctx context.Context, subs subscription.List) error { return e.manageSubs(ctx, "subscribe", subs) }, 1)
}

// Unsubscribe sends a websocket message to stop receiving data from a list of channels
func (e *Exchange) Unsubscribe(subs subscription.List) error {
	return e.ParallelChanOp(context.TODO(), subs, func(ctx context.Context, subs subscription.List) error { return e.manageSubs(ctx, "unsubscribe", subs) }, 1)
}

// manageSubs subscribes or unsubscribes from a list of websocket channels
func (e *Exchange) manageSubs(ctx context.Context, op string, subs subscription.List) error {
	var errs error
	subs, errs = subs.ExpandTemplates(e)
	for _, s := range subs {
		r := &WebsocketRequest{
			Type:       op,
			ProductIDs: s.Pairs,
			Channel:    s.QualifiedChannel,
			Timestamp:  strconv.FormatInt(time.Now().Unix(), 10),
		}
		var err error
		limitType := WSUnauthRate
		if s.Authenticated {
			limitType = WSAuthRate
			if r.JWT, err = e.GetWSJWT(ctx); err != nil {
				return err
			}
		}
		if err = e.Websocket.Conn.SendJSONMessage(ctx, limitType, r); err == nil {
			switch op {
			case "subscribe":
				err = e.Websocket.AddSuccessfulSubscriptions(e.Websocket.Conn, s)
			case "unsubscribe":
				err = e.Websocket.RemoveSubscriptions(e.Websocket.Conn, s)
			}
		}
		errs = common.AppendError(errs, err)
	}
	return errs
}

// GetWSJWT returns a JWT, using a stored one of it's provided, and generating a new one otherwise
func (e *Exchange) GetWSJWT(ctx context.Context) (string, error) {
	e.jwt.m.RLock()
	if e.jwt.expiresAt.After(time.Now()) {
		retStr := e.jwt.token
		e.jwt.m.RUnlock()
		return retStr, nil
	}
	e.jwt.m.RUnlock()
	e.jwt.m.Lock()
	defer e.jwt.m.Unlock()
	var err error
	e.jwt.token, e.jwt.expiresAt, err = e.GetJWT(ctx, "")
	return e.jwt.token, err
}

// processBidAskArray is a helper function that turns WebsocketOrderbookDataHolder into arrays of bids and asks
func processBidAskArray(data *WebsocketOrderbookDataHolder, snapshot bool) (bids, asks orderbook.Levels, err error) {
	bids = make(orderbook.Levels, 0, len(data.Changes))
	asks = make(orderbook.Levels, 0, len(data.Changes))
	for i := range data.Changes {
		change := orderbook.Level{Price: data.Changes[i].PriceLevel.Float64(), Amount: data.Changes[i].NewQuantity.Float64()}
		switch data.Changes[i].Side {
		case "bid":
			bids = append(bids, change)
		case "offer":
			asks = append(asks, change)
		default:
			return nil, nil, fmt.Errorf("%w %v", order.ErrSideIsInvalid, data.Changes[i].Side)
		}
	}
	if snapshot {
		return slices.Clip(bids), slices.Clip(asks), nil
	}
	return bids, asks, nil
}

// statusToStandardStatus is a helper function that converts a Coinbase Pro status string to a standardised order.Status type
func statusToStandardStatus(stat string) (order.Status, error) {
	switch stat {
	case "PENDING":
		return order.New, nil
	case "OPEN":
		return order.Active, nil
	case "FILLED":
		return order.Filled, nil
	case "CANCELLED":
		return order.Cancelled, nil
	case "EXPIRED":
		return order.Expired, nil
	case "FAILED":
		return order.Rejected, nil
	default:
		return order.UnknownStatus, fmt.Errorf("%w %v", order.ErrUnsupportedStatusType, stat)
	}
}

// stringToStandardType is a helper function that converts a Coinbase Pro side string to a standardised order.Type type
func stringToStandardType(str string) (order.Type, error) {
	switch str {
	case "LIMIT_ORDER_TYPE":
		return order.Limit, nil
	case "MARKET_ORDER_TYPE":
		return order.Market, nil
	case "STOP_LIMIT_ORDER_TYPE":
		return order.StopLimit, nil
	default:
		return order.UnknownType, fmt.Errorf("%w %v", order.ErrUnrecognisedOrderType, str)
	}
}

// stringToStandardAsset is a helper function that converts a Coinbase Pro asset string to a standardised asset.Item type
func stringToStandardAsset(str string) (asset.Item, error) {
	switch str {
	case "SPOT":
		return asset.Spot, nil
	case "FUTURE":
		return asset.Futures, nil
	default:
		return asset.Empty, asset.ErrNotSupported
	}
}

// strategyDecoder is a helper function that converts a Coinbase Pro time in force string to a few standardised bools
func strategyDecoder(str string) (tif order.TimeInForce, err error) {
	switch str {
	case "IMMEDIATE_OR_CANCEL":
		return order.ImmediateOrCancel, nil
	case "FILL_OR_KILL":
		return order.FillOrKill, nil
	case "GOOD_UNTIL_CANCELLED":
		return order.GoodTillCancel, nil
	case "GOOD_UNTIL_DATE_TIME":
		return order.GoodTillDay | order.GoodTillTime, nil
	default:
		return order.UnknownTIF, fmt.Errorf("%w %v", errUnrecognisedStrategyType, str)
	}
}

// checkSubscriptions looks for incompatible subscriptions and if found replaces all with defaults
// This should be unnecessary and removable by mid-2025
func (e *Exchange) checkSubscriptions() {
	for _, s := range e.Config.Features.Subscriptions {
		switch s.Channel {
		case "level2_batch", "matches":
			e.Config.Features.Subscriptions = defaultSubscriptions.Clone()
			e.Features.Subscriptions = e.Config.Features.Subscriptions.Enabled()
			return
		}
	}
}

func channelName(s *subscription.Subscription) (string, error) {
	if n, ok := subscriptionNames[s.Channel]; ok {
		return n, nil
	}
	return "", fmt.Errorf("%w: %s", subscription.ErrNotSupported, s.Channel)
}

const subTplText = `
{{ range $asset, $pairs := $.AssetPairs }}
	{{- channelName $.S -}}
	{{- $.AssetSeparator }}
{{- end }}
`
