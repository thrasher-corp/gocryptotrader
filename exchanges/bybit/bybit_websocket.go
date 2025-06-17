package bybit

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/buger/jsonparser"
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
	// Authenticated channels are currently being managed by the `generateAuthSubscriptions` method for the private connection
	// TODO: expand subscription template generation to handle authenticated subscriptions across all assets
}

var subscriptionNames = map[string]string{
	subscription.TickerChannel:    chanPublicTicker,
	subscription.OrderbookChannel: chanOrderbook,
	subscription.AllTradesChannel: chanPublicTrade,
	subscription.MyOrdersChannel:  chanOrder,
	subscription.MyWalletChannel:  chanWallet,
	subscription.MyTradesChannel:  chanExecution,
	subscription.CandlesChannel:   chanKline,
}

// WsConnect connects to a websocket feed
func (by *Bybit) WsConnect(ctx context.Context, conn websocket.Connection) error {
	if err := conn.Dial(ctx, &gws.Dialer{}, http.Header{}); err != nil {
		return err
	}
	conn.SetupPingHandler(request.Unset, websocket.PingHandler{
		MessageType: gws.TextMessage,
		Message:     []byte(`{"op": "ping"}`),
		Delay:       bybitWebsocketTimer,
	})
	return nil
}

// WebsocketAuthenticateConnection sends an authentication message to receive auth data
func (by *Bybit) WebsocketAuthenticateConnection(ctx context.Context, conn websocket.Connection) error {
	creds, err := by.GetCredentials(ctx)
	if err != nil {
		return err
	}
	intNonce := time.Now().Add(time.Hour * 6).UnixMilli()
	strNonce := strconv.FormatInt(intNonce, 10)
	hmac, err := crypto.GetHMAC(crypto.HashSHA256, []byte("GET/realtime"+strNonce), []byte(creds.Secret))
	if err != nil {
		return err
	}
	req := Authenticate{
		RequestID: strconv.FormatInt(conn.GenerateMessageID(false), 10),
		Operation: "auth",
		Args:      []any{creds.Key, intNonce, hex.EncodeToString(hmac)},
	}
	resp, err := conn.SendMessageReturnResponse(ctx, request.Unset, req.RequestID, req)
	if err != nil {
		return err
	}
	var response SubscriptionResponse
	if err := json.Unmarshal(resp, &response); err != nil {
		return err
	}
	if !response.Success {
		return fmt.Errorf("%s with request ID %s msg: %s", response.Operation, response.RequestID, response.RetMsg)
	}
	return nil
}

// Subscribe sends a websocket message to receive data from the channel
func (by *Bybit) Subscribe(ctx context.Context, conn websocket.Connection, channelsToSubscribe subscription.List) error {
	return by.handleSpotSubscription(ctx, conn, "subscribe", channelsToSubscribe)
}

func (by *Bybit) handleSubscriptions(conn websocket.Connection, operation string, subs subscription.List) (args []SubscriptionArgument, err error) {
	subs, err = subs.ExpandTemplates(by)
	if err != nil {
		return
	}

	for _, list := range []subscription.List{subs.Public(), subs.Private()} {
		for _, b := range common.Batch(list, 10) {
			args = append(args, SubscriptionArgument{
				auth:           b[0].Authenticated,
				Operation:      operation,
				RequestID:      strconv.FormatInt(conn.GenerateMessageID(false), 10),
				Arguments:      b.QualifiedChannels(),
				associatedSubs: b,
			})
		}
	}
	return
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (by *Bybit) Unsubscribe(ctx context.Context, conn websocket.Connection, channelsToUnsubscribe subscription.List) error {
	return by.handleSpotSubscription(ctx, conn, "unsubscribe", channelsToUnsubscribe)
}

func (by *Bybit) handleSpotSubscription(ctx context.Context, conn websocket.Connection, operation string, channelsToSubscribe subscription.List) error {
	payloads, err := by.handleSubscriptions(conn, operation, channelsToSubscribe)
	if err != nil {
		return err
	}
	for _, payload := range payloads {
		response, err := conn.SendMessageReturnResponse(ctx, request.Unset, payload.RequestID, payload)
		if err != nil {
			return err
		}
		var resp SubscriptionResponse
		if err := json.Unmarshal(response, &resp); err != nil {
			return err
		}
		if !resp.Success {
			return fmt.Errorf("%s with request ID %s msg: %s", resp.Operation, resp.RequestID, resp.RetMsg)
		}
		if operation == "unsubscribe" {
			err = by.Websocket.RemoveSubscriptions(conn, payload.associatedSubs...)
		} else {
			err = by.Websocket.AddSubscriptions(conn, payload.associatedSubs...)
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

func (by *Bybit) wsHandleData(_ context.Context, assetType asset.Item, respRaw []byte) error {
	var result WebsocketResponse
	if err := json.Unmarshal(respRaw, &result); err != nil {
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
			by.Websocket.DataHandler <- websocket.UnhandledMessageWarning{Message: string(respRaw)}
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
	}
	return fmt.Errorf("unhandled stream data %s", string(respRaw))
}

func (by *Bybit) wsHandleAuthenticatedData(ctx context.Context, respRaw []byte) error {
	var result WebsocketResponse
	if err := json.Unmarshal(respRaw, &result); err != nil {
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
			by.Websocket.DataHandler <- websocket.UnhandledMessageWarning{Message: string(respRaw)}
			return nil
		}
		return nil
	}
	topicSplit := strings.Split(result.Topic, ".")
	if len(topicSplit) == 0 {
		return errInvalidPushData
	}

	switch topicSplit[0] {
	case chanPositions:
		return by.wsProcessPosition(&result)
	case chanExecution:
		return by.wsProcessExecution(&result)
	case chanOrder:
		// Below provides a way of matching an order change to a websocket request. There is no batch support for this
		// so the first element will be used to match the order link ID.
		if id, err := jsonparser.GetString(respRaw, "data", "[0]", "orderLinkId"); err == nil {
			if by.Websocket.Match.IncomingWithData(id, respRaw) {
				return nil // If the data has been routed, return
			}
		}
		return by.wsProcessOrder(&result)
	case chanWallet:
		return by.wsProcessWalletPushData(ctx, respRaw)
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

func (by *Bybit) wsProcessWalletPushData(ctx context.Context, resp []byte) error {
	var result WebsocketWallet
	if err := json.Unmarshal(resp, &result); err != nil {
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
				AssetType: asset.Spot,
				Balance: &account.Balance{
					Currency:  result.Data[x].Coin[y].Coin,
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
func (by *Bybit) wsProcessOrder(resp *WebsocketResponse) error {
	var result WsOrders
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return err
	}
	execution := make([]order.Detail, len(result))
	for x := range result {
		cp, a, err := by.getPairFromCategory(result[x].Category, result[x].Symbol)
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
		tif, err := order.StringToTimeInForce(result[x].TimeInForce)
		if err != nil {
			return err
		}
		execution[x] = order.Detail{
			TimeInForce:          tif,
			Amount:               result[x].Qty.Float64(),
			Exchange:             by.Name,
			OrderID:              result[x].OrderID,
			ClientOrderID:        result[x].OrderLinkID,
			Side:                 side,
			Type:                 orderType,
			Pair:                 cp,
			Cost:                 result[x].CumExecQty.Float64() * result[x].AvgPrice.Float64(),
			Fee:                  result[x].CumExecFee.Float64(),
			AssetType:            a,
			Status:               StringToOrderStatus(result[x].OrderStatus),
			Price:                result[x].Price.Float64(),
			ExecutedAmount:       result[x].CumExecQty.Float64(),
			AverageExecutedPrice: result[x].AvgPrice.Float64(),
			Date:                 result[x].CreatedTime.Time(),
			LastUpdated:          result[x].UpdatedTime.Time(),
		}
	}
	by.Websocket.DataHandler <- execution
	return nil
}

func (by *Bybit) wsProcessExecution(resp *WebsocketResponse) error {
	var result WsExecutions
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return err
	}
	executions := make([]fill.Data, len(result))
	for x := range result {
		cp, a, err := by.getPairFromCategory(result[x].Category, result[x].Symbol)
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
			AssetType:     a,
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
	var result TickerWebsocket
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
	var tickResp TickerWebsocket
	if err := json.Unmarshal(resp.Data, &tickResp); err != nil {
		return err
	}

	p, err := by.MatchSymbolWithAvailablePairs(tickResp.Symbol, assetType, hasPotentialDelimiter(assetType))
	if err != nil {
		return err
	}

	tick := &ticker.Price{Pair: p, ExchangeName: by.Name, AssetType: assetType}
	if resp.Type != "snapshot" {
		snapshot, err := by.GetCachedTicker(p, assetType)
		if err != nil {
			return err
		}
		// ticker updates may be partial, so we need to update the current ticker
		tick = snapshot
	}
	updateTicker(tick, &tickResp)
	tick.LastUpdated = resp.PushTimestamp.Time()
	if err := ticker.ProcessTicker(tick); err != nil {
		return err
	}
	by.Websocket.DataHandler <- tick
	return err
}

func updateTicker(tick *ticker.Price, resp *TickerWebsocket) {
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

// TODO: Remove this function when template expansion is across all assets
func (by *Bybit) submitDirectSubscription(ctx context.Context, conn websocket.Connection, a asset.Item, operation string, channelsToSubscribe subscription.List) error {
	payloads, err := by.directSubscriptionPayload(conn, a, operation, channelsToSubscribe)
	if err != nil {
		return err
	}

	op := by.Websocket.AddSubscriptions
	if operation == "unsubscribe" {
		op = by.Websocket.RemoveSubscriptions
	}

	for _, payload := range payloads {
		if a == asset.Options {
			// The options connection does not send the subscription request id back with the subscription notification payload
			// therefore the code doesn't wait for the response to check whether the subscription is successful or not.
			if err := conn.SendJSONMessage(ctx, request.Unset, payload); err != nil {
				return err
			}
		} else {
			response, err := conn.SendMessageReturnResponse(ctx, request.Unset, payload.RequestID, payload)
			if err != nil {
				return err
			}
			var resp SubscriptionResponse
			if err := json.Unmarshal(response, &resp); err != nil {
				return err
			}
			if !resp.Success {
				return fmt.Errorf("%s with request ID %s msg: %s", resp.Operation, resp.RequestID, resp.RetMsg)
			}
		}
		if err := op(conn, payload.associatedSubs...); err != nil {
			return err
		}
	}
	return nil
}

// TODO: Remove this function when template expansion is across all assets
func (by *Bybit) directSubscriptionPayload(conn websocket.Connection, assetType asset.Item, operation string, channelsToSubscribe subscription.List) ([]SubscriptionArgument, error) {
	var args []SubscriptionArgument
	arg := SubscriptionArgument{
		Operation: operation,
		RequestID: strconv.FormatInt(conn.GenerateMessageID(false), 10),
		Arguments: []string{},
	}
	authArg := SubscriptionArgument{
		auth:      true,
		Operation: operation,
		RequestID: strconv.FormatInt(conn.GenerateMessageID(false), 10),
		Arguments: []string{},
	}

	chanMap := map[string]bool{}
	pairFmt, err := by.GetPairFormat(assetType, true)
	if err != nil {
		return nil, err
	}
	for _, s := range channelsToSubscribe {
		var pair currency.Pair
		if len(s.Pairs) > 1 {
			return nil, subscription.ErrNotSinglePair
		}
		if len(s.Pairs) == 1 {
			pair = s.Pairs[0]
		}
		switch s.Channel {
		case chanOrderbook:
			arg.Arguments = append(arg.Arguments, fmt.Sprintf("%s.%d.%s", s.Channel, 50, pairFmt.Format(pair)))
			arg.associatedSubs = append(arg.associatedSubs, s)
		case chanPublicTrade, chanPublicTicker, chanLiquidation, chanLeverageTokenTicker, chanLeverageTokenNav:
			arg.Arguments = append(arg.Arguments, s.Channel+"."+pairFmt.Format(pair))
			arg.associatedSubs = append(arg.associatedSubs, s)
		case chanKline, chanLeverageTokenKline:
			interval, err := intervalToString(kline.FiveMin)
			if err != nil {
				return nil, err
			}
			arg.Arguments = append(arg.Arguments, s.Channel+"."+interval+"."+pairFmt.Format(pair))
			arg.associatedSubs = append(arg.associatedSubs, s)
		case chanPositions, chanExecution, chanOrder, chanWallet, chanGreeks, chanDCP:
			if chanMap[s.Channel] {
				continue
			}
			authArg.Arguments = append(authArg.Arguments, s.Channel)
			// adding the channel to selected channels so that we will not visit it again.
			chanMap[s.Channel] = true
			authArg.associatedSubs = append(authArg.associatedSubs, s)
		}

		if len(arg.Arguments) >= 10 {
			args = append(args, arg)
			arg = SubscriptionArgument{
				Operation: operation,
				RequestID: strconv.FormatInt(conn.GenerateMessageID(false), 10),
				Arguments: []string{},
			}
		}
	}
	if len(arg.Arguments) != 0 {
		args = append(args, arg)
	}
	if len(authArg.Arguments) != 0 {
		args = append(args, authArg)
	}
	return args, nil
}

// generateAuthSubscriptions generates default subscription for the dedicated auth websocket connection. These are
// agnostic to the asset type and pair as all account level data will be routed through this connection.
// TODO: Remove this function when template expansion is across all assets
func (by *Bybit) generateAuthSubscriptions() (subscription.List, error) {
	if !by.Websocket.CanUseAuthenticatedEndpoints() {
		return nil, nil
	}
	var subscriptions subscription.List
	for _, channel := range []string{chanPositions, chanExecution, chanOrder, chanWallet} {
		subscriptions = append(subscriptions, &subscription.Subscription{Channel: channel, Asset: asset.All})
	}
	return subscriptions, nil
}

// LinearSubscribe sends a subscription message to linear public channels.
func (by *Bybit) authSubscribe(ctx context.Context, conn websocket.Connection, channelSubscriptions subscription.List) error {
	return by.submitDirectSubscription(ctx, conn, asset.Spot, "subscribe", channelSubscriptions)
}

// LinearUnsubscribe sends an unsubscription messages through linear public channels.
func (by *Bybit) authUnsubscribe(ctx context.Context, conn websocket.Connection, channelSubscriptions subscription.List) error {
	return by.submitDirectSubscription(ctx, conn, asset.Spot, "unsubscribe", channelSubscriptions)
}

// getPairFromCategory returns the currency pair and asset type based on the category and symbol. Used with a dedicated
// auth connection where multiple asset type changes are piped through a single connection.
func (by *Bybit) getPairFromCategory(category, symbol string) (currency.Pair, asset.Item, error) {
	assets := make([]asset.Item, 0, 2)
	switch category {
	case "spot":
		assets = append(assets, asset.Spot)
	case "inverse":
		assets = append(assets, asset.CoinMarginedFutures)
	case "linear":
		assets = append(assets, asset.USDTMarginedFutures, asset.USDCMarginedFutures)
	case "option":
		assets = append(assets, asset.Options)
	default:
		return currency.EMPTYPAIR, 0, fmt.Errorf("category %q not supported for incoming symbol %q", category, symbol)
	}
	for _, a := range assets {
		cp, err := by.MatchSymbolWithAvailablePairs(symbol, a, hasPotentialDelimiter(a))
		if err != nil {
			if !errors.Is(err, currency.ErrPairNotFound) {
				return currency.EMPTYPAIR, 0, fmt.Errorf("could not match symbol %s with asset %s: %w", symbol, a, err)
			}
			continue
		}
		return cp, a, nil
	}
	return currency.EMPTYPAIR, 0, currency.ErrPairNotFound
}
