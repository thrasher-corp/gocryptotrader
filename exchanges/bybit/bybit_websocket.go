package bybit

import (
	"context"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"text/template"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
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
)

const (
	bybitWebsocketTimer = 20 * time.Second

	// Public v5 channels
	chanOrderbook           = "orderbook"
	chanPublicTrade         = "publicTrade"
	chanPublicTicker        = "tickers"
	chanKline               = "kline"
	chanLiquidation         = "liquidation"
	chanLeverageTokenKline  = "kline_lt"
	chanLeverageTokenTicker = "tickers_lt"
	chanLeverageTokenNav    = "lt"

	// Private v5 channels
	chanPositions = "position"
	chanExecution = "execution"
	chanOrder     = "order"
	chanWallet    = "wallet"
	chanGreeks    = "greeks"
	chanDCP       = "dcp"

	spotPublic    = "wss://stream.bybit.com/v5/public/spot"
	linearPublic  = "wss://stream.bybit.com/v5/public/linear"  // USDT, USDC perpetual & USDC Futures
	inversePublic = "wss://stream.bybit.com/v5/public/inverse" // Inverse contract
	optionPublic  = "wss://stream.bybit.com/v5/public/option"  // USDC Option

	// Main-net private
	websocketPrivate = "wss://stream.bybit.com/v5/private"
)

var defaultSubscriptions = subscription.List{
	{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.OrderbookChannel, Levels: 50},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.AllTradesChannel},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.CandlesChannel, Interval: kline.OneHour},
	{Enabled: true, Asset: asset.Spot, Authenticated: true, Channel: subscription.MyOrdersChannel},
	{Enabled: true, Asset: asset.Spot, Authenticated: true, Channel: subscription.MyWalletChannel},
	{Enabled: true, Asset: asset.Spot, Authenticated: true, Channel: subscription.MyTradesChannel},
}

var subscriptionNames = map[string]string{
	subscription.TickerChannel:    chanPublicTicker,
	subscription.OrderbookChannel: chanOrderbook,
	subscription.AllTradesChannel: chanPublicTrade,
	subscription.MyOrdersChannel:  chanOrder,
	subscription.MyTradesChannel:  chanExecution,
	subscription.MyWalletChannel:  chanWallet,
	subscription.CandlesChannel:   chanKline,
}

// WsConnect connects to a websocket feed
func (by *Bybit) WsConnect() error {
	ctx := context.TODO()
	if !by.Websocket.IsEnabled() || !by.IsEnabled() || !by.IsAssetWebsocketSupported(asset.Spot) {
		return websocket.ErrWebsocketNotEnabled
	}
	var dialer gws.Dialer
	err := by.Websocket.Conn.Dial(ctx, &dialer, http.Header{})
	if err != nil {
		return err
	}
	by.Websocket.Conn.SetupPingHandler(request.Unset, websocket.PingHandler{
		MessageType: gws.TextMessage,
		Message:     []byte(`{"op": "ping"}`),
		Delay:       bybitWebsocketTimer,
	})

	by.Websocket.Wg.Add(1)
	go by.wsReadData(ctx, asset.Spot, by.Websocket.Conn)
	if by.Websocket.CanUseAuthenticatedEndpoints() {
		err = by.WsAuth(ctx)
		if err != nil {
			by.Websocket.DataHandler <- err
			by.Websocket.SetCanUseAuthenticatedEndpoints(false)
		}
	}
	return nil
}

// WsAuth sends an authentication message to receive auth data
func (by *Bybit) WsAuth(ctx context.Context) error {
	creds, err := by.GetCredentials(ctx)
	if err != nil {
		return err
	}

	var dialer gws.Dialer
	if err := by.Websocket.AuthConn.Dial(ctx, &dialer, http.Header{}); err != nil {
		return err
	}

	by.Websocket.AuthConn.SetupPingHandler(request.Unset, websocket.PingHandler{
		MessageType: gws.TextMessage,
		Message:     []byte(`{"op":"ping"}`),
		Delay:       bybitWebsocketTimer,
	})

	by.Websocket.Wg.Add(1)
	go by.wsReadData(ctx, asset.Spot, by.Websocket.AuthConn)

	intNonce := time.Now().Add(time.Hour * 6).UnixMilli()
	strNonce := strconv.FormatInt(intNonce, 10)
	hmac, err := crypto.GetHMAC(
		crypto.HashSHA256,
		[]byte("GET/realtime"+strNonce),
		[]byte(creds.Secret),
	)
	if err != nil {
		return err
	}
	sign := hex.EncodeToString(hmac)
	req := Authenticate{
		RequestID: strconv.FormatInt(by.Websocket.AuthConn.GenerateMessageID(false), 10),
		Operation: "auth",
		Args:      []any{creds.Key, intNonce, sign},
	}
	resp, err := by.Websocket.AuthConn.SendMessageReturnResponse(ctx, request.Unset, req.RequestID, req)
	if err != nil {
		return err
	}
	var response SubscriptionResponse
	err = json.Unmarshal(resp, &response)
	if err != nil {
		return err
	}
	if !response.Success {
		return fmt.Errorf("%s with request ID %s msg: %s", response.Operation, response.RequestID, response.RetMsg)
	}
	return nil
}

// Subscribe sends a websocket message to receive data from the channel
func (by *Bybit) Subscribe(channelsToSubscribe subscription.List) error {
	ctx := context.TODO()
	return by.handleSpotSubscription(ctx, "subscribe", channelsToSubscribe)
}

func (by *Bybit) handleSubscriptions(operation string, subs subscription.List) (args []SubscriptionArgument, err error) {
	subs, err = subs.ExpandTemplates(by)
	if err != nil {
		return
	}

	for _, list := range []subscription.List{subs.Public(), subs.Private()} {
		for _, b := range common.Batch(list, 10) {
			args = append(args, SubscriptionArgument{
				auth:           b[0].Authenticated,
				Operation:      operation,
				RequestID:      strconv.FormatInt(by.Websocket.Conn.GenerateMessageID(false), 10),
				Arguments:      b.QualifiedChannels(),
				associatedSubs: b,
			})
		}
	}

	return
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (by *Bybit) Unsubscribe(channelsToUnsubscribe subscription.List) error {
	ctx := context.TODO()
	return by.handleSpotSubscription(ctx, "unsubscribe", channelsToUnsubscribe)
}

func (by *Bybit) handleSpotSubscription(ctx context.Context, operation string, channelsToSubscribe subscription.List) error {
	payloads, err := by.handleSubscriptions(operation, channelsToSubscribe)
	if err != nil {
		return err
	}
	for a := range payloads {
		var response []byte
		if payloads[a].auth {
			response, err = by.Websocket.AuthConn.SendMessageReturnResponse(ctx, request.Unset, payloads[a].RequestID, payloads[a])
			if err != nil {
				return err
			}
		} else {
			response, err = by.Websocket.Conn.SendMessageReturnResponse(ctx, request.Unset, payloads[a].RequestID, payloads[a])
			if err != nil {
				return err
			}
		}
		var resp SubscriptionResponse
		err = json.Unmarshal(response, &resp)
		if err != nil {
			return err
		}
		if !resp.Success {
			return fmt.Errorf("%s with request ID %s msg: %s", resp.Operation, resp.RequestID, resp.RetMsg)
		}

		var conn websocket.Connection
		if payloads[a].auth {
			conn = by.Websocket.AuthConn
		} else {
			conn = by.Websocket.Conn
		}

		if operation == "unsubscribe" {
			err = by.Websocket.RemoveSubscriptions(conn, payloads[a].associatedSubs...)
		} else {
			err = by.Websocket.AddSubscriptions(conn, payloads[a].associatedSubs...)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// generateSubscriptions generates default subscription
func (by *Bybit) generateSubscriptions() (subscription.List, error) {
	return by.Features.Subscriptions.ExpandTemplates(by)
}

// GetSubscriptionTemplate returns a subscription channel template
func (by *Bybit) GetSubscriptionTemplate(_ *subscription.Subscription) (*template.Template, error) {
	return template.New("master.tmpl").Funcs(template.FuncMap{
		"channelName":          channelName,
		"isSymbolChannel":      isSymbolChannel,
		"intervalToString":     intervalToString,
		"getCategoryName":      getCategoryName,
		"isCategorisedChannel": isCategorisedChannel,
	}).Parse(subTplText)
}

// wsReadData receives and passes on websocket messages for processing
func (by *Bybit) wsReadData(ctx context.Context, assetType asset.Item, ws websocket.Connection) {
	defer by.Websocket.Wg.Done()
	for {
		select {
		case <-by.Websocket.ShutdownC:
			return
		default:
			resp := ws.ReadMessage()
			if resp.Raw == nil {
				return
			}
			err := by.wsHandleData(ctx, assetType, resp.Raw)
			if err != nil {
				by.Websocket.DataHandler <- err
			}
		}
	}
}

func (by *Bybit) wsHandleData(ctx context.Context, assetType asset.Item, respRaw []byte) error {
	var result WebsocketResponse
	err := json.Unmarshal(respRaw, &result)
	if err != nil {
		return err
	}
	if result.Topic == "" {
		switch result.Operation {
		case "subscribe", "unsubscribe", "auth":
			if result.RequestID != "" {
				if !by.Websocket.Match.IncomingWithData(result.RequestID, respRaw) {
					return fmt.Errorf("could not match subscription with id %s data %s", result.RequestID, respRaw)
				}
			}
		case "ping", "pong":
		default:
			by.Websocket.DataHandler <- websocket.UnhandledMessageWarning{
				Message: string(respRaw),
			}
			return nil
		}
		return nil
	}
	topicSplit := strings.Split(result.Topic, ".")
	if len(topicSplit) == 0 {
		return errInvalidPushData
	}
	switch topicSplit[0] {
	case chanOrderbook:
		return by.wsProcessOrderbook(assetType, &result)
	case chanPublicTrade:
		return by.wsProcessPublicTrade(assetType, &result)
	case chanPublicTicker:
		return by.wsProcessPublicTicker(assetType, &result)
	case chanKline:
		return by.wsProcessKline(assetType, &result, topicSplit)
	case chanLiquidation:
		return by.wsProcessLiquidation(&result)
	case chanLeverageTokenKline:
		return by.wsProcessLeverageTokenKline(assetType, &result, topicSplit)
	case chanLeverageTokenTicker:
		return by.wsProcessLeverageTokenTicker(assetType, &result)
	case chanLeverageTokenNav:
		return by.wsLeverageTokenNav(&result)
	case chanPositions:
		return by.wsProcessPosition(&result)
	case chanExecution:
		return by.wsProcessExecution(asset.Spot, &result)
	case chanOrder:
		return by.wsProcessOrder(asset.Spot, &result)
	case chanWallet:
		return by.wsProcessWalletPushData(ctx, asset.Spot, respRaw)
	case chanGreeks:
		return by.wsProcessGreeks(respRaw)
	case chanDCP:
		return nil
	}
	return fmt.Errorf("unhandled stream data %s", string(respRaw))
}

func (by *Bybit) wsProcessGreeks(resp []byte) error {
	var result GreeksResponse
	err := json.Unmarshal(resp, &result)
	if err != nil {
		return err
	}
	by.Websocket.DataHandler <- &result
	return nil
}

func (by *Bybit) wsProcessWalletPushData(ctx context.Context, assetType asset.Item, resp []byte) error {
	var result WebsocketWallet
	err := json.Unmarshal(resp, &result)
	if err != nil {
		return err
	}
	creds, err := by.GetCredentials(ctx)
	if err != nil {
		return err
	}
	var changes []account.Change
	for x := range result.Data {
		for y := range result.Data[x].Coin {
			changes = append(changes, account.Change{
				AssetType: assetType,
				Balance: &account.Balance{
					Currency:  currency.NewCode(result.Data[x].Coin[y].Coin),
					Total:     result.Data[x].Coin[y].WalletBalance.Float64(),
					Free:      result.Data[x].Coin[y].WalletBalance.Float64(),
					UpdatedAt: result.CreationTime.Time(),
				},
			})
		}
	}
	by.Websocket.DataHandler <- changes
	return account.ProcessChange(by.Name, changes, creds)
}

// wsProcessOrder the order stream to see changes to your orders in real-time.
func (by *Bybit) wsProcessOrder(assetType asset.Item, resp *WebsocketResponse) error {
	var result WsOrders
	err := json.Unmarshal(resp.Data, &result)
	if err != nil {
		return err
	}
	execution := make([]order.Detail, len(result))
	for x := range result {
		cp, err := by.MatchSymbolWithAvailablePairs(result[x].Symbol, assetType, hasPotentialDelimiter(assetType))
		if err != nil {
			return err
		}
		orderType, err := order.StringToOrderType(result[x].OrderType)
		if err != nil {
			return err
		}
		side, err := order.StringToOrderSide(result[x].Side)
		if err != nil {
			return err
		}
		execution[x] = order.Detail{
			Amount:         result[x].Qty.Float64(),
			Exchange:       by.Name,
			OrderID:        result[x].OrderID,
			ClientOrderID:  result[x].OrderLinkID,
			Side:           side,
			Type:           orderType,
			Pair:           cp,
			Cost:           result[x].CumExecQty.Float64() * result[x].AvgPrice.Float64(),
			AssetType:      assetType,
			Status:         StringToOrderStatus(result[x].OrderStatus),
			Price:          result[x].Price.Float64(),
			ExecutedAmount: result[x].CumExecQty.Float64(),
			Date:           result[x].CreatedTime.Time(),
			LastUpdated:    result[x].UpdatedTime.Time(),
		}
	}
	by.Websocket.DataHandler <- execution
	return nil
}

func (by *Bybit) wsProcessExecution(assetType asset.Item, resp *WebsocketResponse) error {
	var result WsExecutions
	err := json.Unmarshal(resp.Data, &result)
	if err != nil {
		return err
	}
	executions := make([]fill.Data, len(result))
	for x := range result {
		cp, err := by.MatchSymbolWithAvailablePairs(result[x].Symbol, assetType, hasPotentialDelimiter(assetType))
		if err != nil {
			return err
		}
		side, err := order.StringToOrderSide(result[x].Side)
		if err != nil {
			return err
		}
		executions[x] = fill.Data{
			ID:            result[x].ExecID,
			Timestamp:     result[x].ExecTime.Time(),
			Exchange:      by.Name,
			AssetType:     assetType,
			CurrencyPair:  cp,
			Side:          side,
			OrderID:       result[x].OrderID,
			ClientOrderID: result[x].OrderLinkID,
			Price:         result[x].ExecPrice.Float64(),
			Amount:        result[x].ExecQty.Float64(),
		}
	}
	by.Websocket.DataHandler <- executions
	return nil
}

func (by *Bybit) wsProcessPosition(resp *WebsocketResponse) error {
	var result WsPositions
	err := json.Unmarshal(resp.Data, &result)
	if err != nil {
		return err
	}
	by.Websocket.DataHandler <- result
	return nil
}

func (by *Bybit) wsLeverageTokenNav(resp *WebsocketResponse) error {
	var result LTNav
	err := json.Unmarshal(resp.Data, &result)
	if err != nil {
		return err
	}
	by.Websocket.DataHandler <- result
	return nil
}

func (by *Bybit) wsProcessLeverageTokenTicker(assetType asset.Item, resp *WebsocketResponse) error {
	var result TickerItem
	err := json.Unmarshal(resp.Data, &result)
	if err != nil {
		return err
	}
	cp, err := by.MatchSymbolWithAvailablePairs(result.Symbol, assetType, hasPotentialDelimiter(assetType))
	if err != nil {
		return err
	}
	by.Websocket.DataHandler <- &ticker.Price{
		Last:         result.LastPrice.Float64(),
		High:         result.HighPrice24H.Float64(),
		Low:          result.LowPrice24H.Float64(),
		Pair:         cp,
		ExchangeName: by.Name,
		AssetType:    assetType,
		LastUpdated:  resp.PushTimestamp.Time(),
	}
	return nil
}

func (by *Bybit) wsProcessLeverageTokenKline(assetType asset.Item, resp *WebsocketResponse, topicSplit []string) error {
	var result LTKlines
	err := json.Unmarshal(resp.Data, &result)
	if err != nil {
		return err
	}
	cp, err := by.MatchSymbolWithAvailablePairs(topicSplit[2], assetType, hasPotentialDelimiter(assetType))
	if err != nil {
		return err
	}
	ltKline := make([]websocket.KlineData, len(result))
	for x := range result {
		interval, err := stringToInterval(result[x].Interval)
		if err != nil {
			return err
		}
		ltKline[x] = websocket.KlineData{
			Timestamp:  result[x].Timestamp.Time(),
			Pair:       cp,
			AssetType:  assetType,
			Exchange:   by.Name,
			StartTime:  result[x].Start.Time(),
			CloseTime:  result[x].End.Time(),
			Interval:   interval.String(),
			OpenPrice:  result[x].Open.Float64(),
			ClosePrice: result[x].Close.Float64(),
			HighPrice:  result[x].High.Float64(),
			LowPrice:   result[x].Low.Float64(),
		}
	}
	by.Websocket.DataHandler <- result
	return nil
}

func (by *Bybit) wsProcessLiquidation(resp *WebsocketResponse) error {
	var result WebsocketLiquidation
	err := json.Unmarshal(resp.Data, &result)
	if err != nil {
		return err
	}
	by.Websocket.DataHandler <- result
	return nil
}

func (by *Bybit) wsProcessKline(assetType asset.Item, resp *WebsocketResponse, topicSplit []string) error {
	var result WsKlines
	err := json.Unmarshal(resp.Data, &result)
	if err != nil {
		return err
	}
	cp, err := by.MatchSymbolWithAvailablePairs(topicSplit[2], assetType, hasPotentialDelimiter(assetType))
	if err != nil {
		return err
	}
	spotCandlesticks := make([]websocket.KlineData, len(result))
	for x := range result {
		interval, err := stringToInterval(result[x].Interval)
		if err != nil {
			return err
		}
		spotCandlesticks[x] = websocket.KlineData{
			Timestamp:  result[x].Timestamp.Time(),
			Pair:       cp,
			AssetType:  assetType,
			Exchange:   by.Name,
			StartTime:  result[x].Start.Time(),
			CloseTime:  result[x].End.Time(),
			Interval:   interval.String(),
			OpenPrice:  result[x].Open.Float64(),
			ClosePrice: result[x].Close.Float64(),
			HighPrice:  result[x].High.Float64(),
			LowPrice:   result[x].Low.Float64(),
			Volume:     result[x].Volume.Float64(),
		}
	}
	by.Websocket.DataHandler <- spotCandlesticks
	return nil
}

func (by *Bybit) wsProcessPublicTicker(assetType asset.Item, resp *WebsocketResponse) error {
	tickResp := new(TickerItem)
	if err := json.Unmarshal(resp.Data, tickResp); err != nil {
		return err
	}

	p, err := by.MatchSymbolWithAvailablePairs(tickResp.Symbol, assetType, hasPotentialDelimiter(assetType))
	if err != nil {
		return err
	}
	pFmt, err := by.GetPairFormat(assetType, false)
	if err != nil {
		return err
	}
	p = p.Format(pFmt)

	var tick *ticker.Price
	if resp.Type == "snapshot" {
		tick = &ticker.Price{
			Pair:         p,
			ExchangeName: by.Name,
			AssetType:    assetType,
		}
	} else {
		// ticker updates may be partial, so we need to update the current ticker
		tick, err = ticker.GetTicker(by.Name, p, assetType)
		if err != nil {
			return err
		}
	}

	updateTicker(tick, tickResp)
	tick.LastUpdated = resp.PushTimestamp.Time()

	if err = ticker.ProcessTicker(tick); err == nil {
		by.Websocket.DataHandler <- tick
	}

	return err
}

func updateTicker(tick *ticker.Price, resp *TickerItem) {
	if resp.LastPrice.Float64() != 0 {
		tick.Last = resp.LastPrice.Float64()
	}
	if resp.HighPrice24H.Float64() != 0 {
		tick.High = resp.HighPrice24H.Float64()
	}
	if resp.LowPrice24H.Float64() != 0 {
		tick.Low = resp.LowPrice24H.Float64()
	}
	if resp.Volume24H.Float64() != 0 {
		tick.Volume = resp.Volume24H.Float64()
	}

	if tick.AssetType == asset.Spot {
		return
	}

	if resp.MarkPrice.Float64() != 0 {
		tick.MarkPrice = resp.MarkPrice.Float64()
	}
	if resp.IndexPrice.Float64() != 0 {
		tick.IndexPrice = resp.IndexPrice.Float64()
	}
	if resp.OpenInterest.Float64() != 0 {
		tick.OpenInterest = resp.OpenInterest.Float64()
	}

	switch tick.AssetType {
	case asset.Options:
		if resp.BidPrice.Float64() != 0 {
			tick.Bid = resp.BidPrice.Float64()
		}
		if resp.BidSize.Float64() != 0 {
			tick.BidSize = resp.BidSize.Float64()
		}
		if resp.AskPrice.Float64() != 0 {
			tick.Ask = resp.AskPrice.Float64()
		}
		if resp.AskSize.Float64() != 0 {
			tick.AskSize = resp.AskSize.Float64()
		}
	case asset.USDCMarginedFutures, asset.USDTMarginedFutures, asset.CoinMarginedFutures:
		if resp.Bid1Price.Float64() != 0 {
			tick.Bid = resp.Bid1Price.Float64()
		}
		if resp.Bid1Size.Float64() != 0 {
			tick.BidSize = resp.Bid1Size.Float64()
		}
		if resp.Ask1Price.Float64() != 0 {
			tick.Ask = resp.Ask1Price.Float64()
		}
		if resp.Ask1Size.Float64() != 0 {
			tick.AskSize = resp.Ask1Size.Float64()
		}
	}
}

func (by *Bybit) wsProcessPublicTrade(assetType asset.Item, resp *WebsocketResponse) error {
	var result WebsocketPublicTrades
	err := json.Unmarshal(resp.Data, &result)
	if err != nil {
		return err
	}
	tradeDatas := make([]trade.Data, len(result))
	for x := range result {
		cp, err := by.MatchSymbolWithAvailablePairs(result[x].Symbol, assetType, hasPotentialDelimiter(assetType))
		if err != nil {
			return err
		}
		side, err := order.StringToOrderSide(result[x].Side)
		if err != nil {
			return err
		}
		tradeDatas[x] = trade.Data{
			Timestamp:    result[x].OrderFillTimestamp.Time(),
			CurrencyPair: cp,
			AssetType:    assetType,
			Exchange:     by.Name,
			Price:        result[x].Price.Float64(),
			Amount:       result[x].Size.Float64(),
			Side:         side,
			TID:          result[x].TradeID,
		}
	}
	return trade.AddTradesToBuffer(tradeDatas...)
}

func (by *Bybit) wsProcessOrderbook(assetType asset.Item, resp *WebsocketResponse) error {
	var result WsOrderbookDetail
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return err
	}
	if len(result.Bids) == 0 && len(result.Asks) == 0 {
		return nil
	}

	cp, err := by.MatchSymbolWithAvailablePairs(result.Symbol, assetType, hasPotentialDelimiter(assetType))
	if err != nil {
		return err
	}
	asks := make([]orderbook.Level, len(result.Asks))
	for i := range result.Asks {
		asks[i].Price = result.Asks[i][0].Float64()
		asks[i].Amount = result.Asks[i][1].Float64()
	}
	bids := make([]orderbook.Level, len(result.Bids))
	for i := range result.Bids {
		bids[i].Price = result.Bids[i][0].Float64()
		bids[i].Amount = result.Bids[i][1].Float64()
	}

	if resp.Type == "snapshot" {
		return by.Websocket.Orderbook.LoadSnapshot(&orderbook.Book{
			Pair:         cp,
			Exchange:     by.Name,
			Asset:        assetType,
			LastUpdated:  resp.OrderbookLastUpdated.Time(),
			LastUpdateID: result.UpdateID,
			LastPushed:   resp.PushTimestamp.Time(),
			Asks:         asks,
			Bids:         bids,
		})
	}
	return by.Websocket.Orderbook.Update(&orderbook.Update{
		Pair:       cp,
		Asks:       asks,
		Bids:       bids,
		Asset:      assetType,
		UpdateID:   result.UpdateID,
		UpdateTime: resp.OrderbookLastUpdated.Time(),
		LastPushed: resp.PushTimestamp.Time(),
	})
}

// channelName converts global channel names to exchange specific names
func channelName(s *subscription.Subscription) string {
	if name, ok := subscriptionNames[s.Channel]; ok {
		return name
	}
	return s.Channel
}

// isSymbolChannel returns whether the channel accepts a symbol parameter
func isSymbolChannel(name string) bool {
	switch name {
	case chanPositions, chanExecution, chanOrder, chanDCP, chanWallet:
		return false
	}
	return true
}

func isCategorisedChannel(name string) bool {
	switch name {
	case chanPositions, chanExecution, chanOrder:
		return true
	}
	return false
}

const subTplText = `
{{ with $name := channelName $.S }}
	{{- range $asset, $pairs := $.AssetPairs }}
		{{- if isSymbolChannel $name }}
			{{- range $p := $pairs }}
				{{- $name -}} .
				{{- if eq $name "orderbook" -}} {{- $.S.Levels -}} . {{- end }}
				{{- if eq $name "kline" -}} {{- intervalToString $.S.Interval -}} . {{- end }}
				{{- $p }}
				{{- $.PairSeparator }}
			{{- end }}
		{{- else }}
			{{- $name }}
			{{- if and (isCategorisedChannel $name) ($categoryName := getCategoryName $asset) -}} . {{- $categoryName -}} {{- end }}
		{{- end }}
	{{- end }}
	{{- $.AssetSeparator }}
{{- end }}
`

// hasPotentialDelimiter returns if the asset has a potential delimiter on the pairs being returned.
func hasPotentialDelimiter(a asset.Item) bool {
	return a == asset.Options || a == asset.USDCMarginedFutures
}
