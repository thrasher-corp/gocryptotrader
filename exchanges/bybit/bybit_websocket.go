package bybit

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/buger/jsonparser"
	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
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
	websocketTrade   = "wss://stream.bybit.com/v5/trade"
)

var defaultSubscriptions = subscription.List{
	{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.OrderbookChannel, Levels: 50},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.AllTradesChannel},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.CandlesChannel, Interval: kline.OneHour},
}

var subscriptionNames = map[string]string{
	subscription.TickerChannel:    chanPublicTicker,
	subscription.OrderbookChannel: chanOrderbook,
	subscription.AllTradesChannel: chanPublicTrade,
	subscription.CandlesChannel:   chanKline,
}

// WsConnect connects to a websocket feed
func (by *Bybit) WsConnect(ctx context.Context, conn stream.Connection) error {
	if err := conn.DialContext(ctx, &websocket.Dialer{}, http.Header{}); err != nil {
		return err
	}
	conn.SetupPingHandler(request.Unset, stream.PingHandler{
		MessageType: websocket.TextMessage,
		Message:     []byte(`{"op": "ping"}`),
		Delay:       bybitWebsocketTimer,
	})
	return nil
}

// WebsocketAuthenticatePrivateConnection sends an authentication message to the private websocket for inbound account
// data
func (by *Bybit) WebsocketAuthenticatePrivateConnection(ctx context.Context, conn stream.Connection) error {
	req, err := by.GetAuthenticationPayload(ctx, strconv.FormatInt(conn.GenerateMessageID(false), 10))
	if err != nil {
		return err
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
		return fmt.Errorf("%s with request ID %s msg: %s", response.Operation, response.RequestID, response.ReturnMessage)
	}
	return nil
}

// WebsocketAuthenticateTradeConnection sends an authentication message to the private websocket for outbound account
// data
func (by *Bybit) WebsocketAuthenticateTradeConnection(ctx context.Context, conn stream.Connection) error {
	// request ID is not returned with the response, a workaround in the trade connection handler monitors the response
	// for the operation type "auth", which is then set in the response match key.
	req, err := by.GetAuthenticationPayload(ctx, "auth")
	if err != nil {
		return err
	}
	resp, err := conn.SendMessageReturnResponse(ctx, request.Unset, req.RequestID, req)
	if err != nil {
		return err
	}
	var response struct {
		ReturnCode    int64  `json:"retCode"`
		ReturnMessage string `json:"retMsg"`
		Operation     string `json:"op"`
		ConnectionID  string `json:"connId"`
	}
	if err := json.Unmarshal(resp, &response); err != nil {
		return err
	}
	if response.ReturnCode != 0 {
		return fmt.Errorf("%s failed - code:%d [%v] msg:%s", response.Operation, response.ReturnCode, retCode[response.ReturnCode], response.ReturnMessage)
	}
	return nil
}

// GetAuthenticationPayload returns the authentication payload for the websocket connection to upgrade the connection.
func (by *Bybit) GetAuthenticationPayload(ctx context.Context, requestID string) (*Authenticate, error) {
	creds, err := by.GetCredentials(ctx)
	if err != nil {
		return nil, err
	}
	intNonce := time.Now().Add(time.Hour * 6).UnixMilli()
	strNonce := strconv.FormatInt(intNonce, 10)
	hmac, err := crypto.GetHMAC(crypto.HashSHA256, []byte("GET/realtime"+strNonce), []byte(creds.Secret))
	if err != nil {
		return nil, err
	}
	return &Authenticate{
		RequestID: requestID,
		Operation: "auth",
		Args:      []any{creds.Key, intNonce, crypto.HexEncodeToString(hmac)},
	}, nil
}

// Subscribe sends a websocket message to receive data from the channel
func (by *Bybit) Subscribe(ctx context.Context, conn stream.Connection, channelsToSubscribe subscription.List) error {
	return by.handleSpotSubscription(ctx, conn, "subscribe", channelsToSubscribe)
}

func (by *Bybit) handleSubscriptions(conn stream.Connection, operation string, subs subscription.List) (args []SubscriptionArgument, err error) {
	subs, err = subs.ExpandTemplates(by)
	if err != nil {
		return
	}
	chans := []string{}
	authChans := []string{}
	for _, s := range subs {
		if s.Authenticated {
			authChans = append(authChans, s.QualifiedChannel)
		} else {
			chans = append(chans, s.QualifiedChannel)
		}
	}
	for _, b := range common.Batch(chans, 10) {
		args = append(args, SubscriptionArgument{
			Operation: operation,
			RequestID: strconv.FormatInt(conn.GenerateMessageID(false), 10),
			Arguments: b,
		})
	}
	if len(authChans) != 0 {
		args = append(args, SubscriptionArgument{
			auth:      true,
			Operation: operation,
			RequestID: strconv.FormatInt(conn.GenerateMessageID(false), 10),
			Arguments: authChans,
		})
	}
	return
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (by *Bybit) Unsubscribe(ctx context.Context, conn stream.Connection, channelsToUnsubscribe subscription.List) error {
	return by.handleSpotSubscription(ctx, conn, "unsubscribe", channelsToUnsubscribe)
}

func (by *Bybit) handleSpotSubscription(ctx context.Context, conn stream.Connection, operation string, channelsToSubscribe subscription.List) error {
	payloads, err := by.handleSubscriptions(conn, operation, channelsToSubscribe)
	if err != nil {
		return err
	}
	for _, payload := range payloads {
		var response []byte
		response, err = conn.SendMessageReturnResponse(ctx, request.Unset, payload.RequestID, payload)
		if err != nil {
			return err
		}
		var resp SubscriptionResponse
		err = json.Unmarshal(response, &resp)
		if err != nil {
			return err
		}
		if !resp.Success {
			return fmt.Errorf("%s with request ID %s msg: %s", resp.Operation, resp.RequestID, resp.ReturnMessage)
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
		"channelName":      channelName,
		"isSymbolChannel":  isSymbolChannel,
		"intervalToString": intervalToString,
	}).Parse(subTplText)
}

func (by *Bybit) wsHandleTradeData(_ context.Context, respRaw []byte) error {
	var response struct {
		RequestID string `json:"reqId"`
		Operation string `json:"op"`
	}
	if err := json.Unmarshal(respRaw, &response); err != nil {
		return err
	}

	if response.RequestID != "" {
		if !by.Websocket.Match.IncomingWithData(response.RequestID, respRaw) {
			return fmt.Errorf("could not match subscription with id %s data %s", response.RequestID, respRaw)
		}
		return nil
	}

	switch response.Operation {
	case "auth": // When authenticating the connection there is no request ID, so a static value is used.
		if !by.Websocket.Match.IncomingWithData(response.Operation, respRaw) {
			return fmt.Errorf("could not match subscription with id %s data %s", response.Operation, respRaw)
		}
		return nil
	case "pong":
		return nil
	default:
		return fmt.Errorf("cannot route trade data %v to correct handler", string(respRaw))
	}
}

func (by *Bybit) wsHandleData(_ context.Context, respRaw []byte, assetType asset.Item) error {
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
			by.Websocket.DataHandler <- stream.UnhandledMessageWarning{Message: string(respRaw)}
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

func (by *Bybit) wsHandleAuthenticated(_ context.Context, respRaw []byte) error {
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
			by.Websocket.DataHandler <- stream.UnhandledMessageWarning{Message: string(respRaw)}
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
		// so the first element will be used to match the order ID.
		if id, err := jsonparser.GetString(respRaw, "data", "[0]", "orderId"); err == nil {
			if by.Websocket.Match.IncomingWithData(id, respRaw) {
				return nil // If the data has been routed, return
			}
		}
		return by.wsProcessOrder(&result)
	case chanWallet:
		return by.wsProcessWalletPushData(respRaw)
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

func (by *Bybit) wsProcessWalletPushData(resp []byte) error {
	var result WebsocketWallet
	if err := json.Unmarshal(resp, &result); err != nil {
		return err
	}
	accounts := []account.Change{}
	for x := range result.Data {
		for y := range result.Data[x].Coin {
			accounts = append(accounts, account.Change{
				Exchange: by.Name,
				Currency: currency.NewCode(result.Data[x].Coin[y].Coin),
				Asset:    asset.Spot,
				Amount:   result.Data[x].Coin[y].WalletBalance.Float64(),
			})
		}
	}
	by.Websocket.DataHandler <- accounts
	return nil
}

// wsProcessOrder the order stream to see changes to your orders in real-time.
func (by *Bybit) wsProcessOrder(resp *WebsocketResponse) error {
	var result []WebsocketOrderDetails
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
		execution[x] = order.Detail{
			Amount:               result[x].Qty.Float64(),
			Exchange:             by.Name,
			OrderID:              result[x].OrderID,
			ClientOrderID:        result[x].OrderLinkID,
			Side:                 result[x].Side,
			Type:                 orderType,
			Pair:                 cp,
			Cost:                 result[x].CumExecQty.Float64() * result[x].AvgPrice.Float64(),
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
	ltKline := make([]stream.KlineData, len(result))
	for x := range result {
		interval, err := stringToInterval(result[x].Interval)
		if err != nil {
			return err
		}
		ltKline[x] = stream.KlineData{
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
	spotCandlesticks := make([]stream.KlineData, len(result))
	for x := range result {
		interval, err := stringToInterval(result[x].Interval)
		if err != nil {
			return err
		}
		spotCandlesticks[x] = stream.KlineData{
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
	snapshot, err := ticker.GetTicker(by.Name, p, assetType)
	if err == nil && resp.Type != "snapshot" {
		// ticker updates may be partial, so we need to update the current ticker
		tick = snapshot
	}
	updateTicker(tick, &tickResp)
	tick.LastUpdated = resp.PushTimestamp.Time()

	if err = ticker.ProcessTicker(tick); err == nil {
		by.Websocket.DataHandler <- tick
	}
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
	return trade.AddTradesToBuffer(by.Name, tradeDatas...)
}

func (by *Bybit) wsProcessOrderbook(assetType asset.Item, resp *WebsocketResponse) error {
	var result WsOrderbookDetail
	err := json.Unmarshal(resp.Data, &result)
	if err != nil {
		return err
	}
	cp, err := by.MatchSymbolWithAvailablePairs(result.Symbol, assetType, hasPotentialDelimiter(assetType))
	if err != nil {
		return err
	}
	asks := make([]orderbook.Tranche, len(result.Asks))
	for i := range result.Asks {
		asks[i].Price, err = strconv.ParseFloat(result.Asks[i][0], 64)
		if err != nil {
			return err
		}
		asks[i].Amount, err = strconv.ParseFloat(result.Asks[i][1], 64)
		if err != nil {
			return err
		}
	}
	bids := make([]orderbook.Tranche, len(result.Bids))
	for i := range result.Bids {
		bids[i].Price, err = strconv.ParseFloat(result.Bids[i][0], 64)
		if err != nil {
			return err
		}
		bids[i].Amount, err = strconv.ParseFloat(result.Bids[i][1], 64)
		if err != nil {
			return err
		}
	}
	if len(asks) == 0 && len(bids) == 0 {
		return nil
	}
	if resp.Type == "snapshot" {
		return by.Websocket.Orderbook.LoadSnapshot(&orderbook.Base{
			Pair:           cp,
			Exchange:       by.Name,
			Asset:          assetType,
			LastUpdated:    resp.OrderbookLastUpdated.Time(),
			LastUpdateID:   result.UpdateID,
			UpdatePushedAt: resp.PushTimestamp.Time(),
			Asks:           asks,
			Bids:           bids,
		})
	}
	return by.Websocket.Orderbook.Update(&orderbook.Update{
		Pair:           cp,
		Asks:           asks,
		Bids:           bids,
		Asset:          assetType,
		UpdateID:       result.UpdateID,
		UpdateTime:     resp.OrderbookLastUpdated.Time(),
		UpdatePushedAt: resp.PushTimestamp.Time(),
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
func (by *Bybit) handleSubscriptionNonTemplate(ctx context.Context, conn stream.Connection, a asset.Item, operation string, channelsToSubscribe subscription.List) error {
	payloads, err := by.handleSubscriptionsNonTemplate(conn, a, operation, channelsToSubscribe)
	if err != nil {
		return err
	}
	for _, payload := range payloads {
		if a == asset.Options {
			// The options connection does not send the subscription request id back with the subscription notification payload
			// therefore the code doesn't wait for the response to check whether the subscription is successful or not.
			err = conn.SendJSONMessage(ctx, request.Unset, payload)
			if err != nil {
				return err
			}
			continue
		}
		var response []byte
		response, err = conn.SendMessageReturnResponse(ctx, request.Unset, payload.RequestID, payload)
		if err != nil {
			return err
		}
		var resp SubscriptionResponse
		err = json.Unmarshal(response, &resp)
		if err != nil {
			return err
		}
		if !resp.Success {
			return fmt.Errorf("%s with request ID %s msg: %s", resp.Operation, resp.RequestID, resp.ReturnMessage)
		}
	}
	return nil
}

// TODO: Remove this function when template expansion is across all assets
func (by *Bybit) handleSubscriptionsNonTemplate(conn stream.Connection, assetType asset.Item, operation string, channelsToSubscribe subscription.List) ([]SubscriptionArgument, error) {
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
	pairFormat, err := by.GetPairFormat(assetType, true)
	if err != nil {
		return nil, err
	}
	for i := range channelsToSubscribe {
		if len(channelsToSubscribe[i].Pairs) != 1 {
			return nil, subscription.ErrNotSinglePair
		}
		pair := channelsToSubscribe[i].Pairs[0]
		switch channelsToSubscribe[i].Channel {
		case chanOrderbook:
			arg.Arguments = append(arg.Arguments, fmt.Sprintf("%s.%d.%s", channelsToSubscribe[i].Channel, 50, pair.Format(pairFormat).String()))
		case chanPublicTrade, chanPublicTicker, chanLiquidation, chanLeverageTokenTicker, chanLeverageTokenNav:
			arg.Arguments = append(arg.Arguments, channelsToSubscribe[i].Channel+"."+pair.Format(pairFormat).String())
		case chanKline, chanLeverageTokenKline:
			interval, err := intervalToString(kline.FiveMin)
			if err != nil {
				return nil, err
			}
			arg.Arguments = append(arg.Arguments, channelsToSubscribe[i].Channel+"."+interval+"."+pair.Format(pairFormat).String())
		case chanPositions, chanExecution, chanOrder, chanWallet, chanGreeks, chanDCP:
			if chanMap[channelsToSubscribe[i].Channel] {
				continue
			}
			authArg.Arguments = append(authArg.Arguments, channelsToSubscribe[i].Channel)
			// adding the channel to selected channels so that we will not visit it again.
			chanMap[channelsToSubscribe[i].Channel] = true
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
		subscriptions = append(subscriptions, &subscription.Subscription{
			Channel: channel,
			Pairs:   currency.Pairs{currency.EMPTYPAIR}, // This is a placeholder, the actual pair is not required for these channels
			Asset:   asset.All,
		})
	}
	return subscriptions, nil
}

// LinearSubscribe sends a subscription message to linear public channels.
func (by *Bybit) authSubscribe(ctx context.Context, conn stream.Connection, channelSubscriptions subscription.List) error {
	return by.handleSubscriptionNonTemplate(ctx, conn, asset.Spot, "subscribe", channelSubscriptions)
}

// LinearUnsubscribe sends an unsubscription messages through linear public channels.
func (by *Bybit) authUnsubscribe(ctx context.Context, conn stream.Connection, channelSubscriptions subscription.List) error {
	return by.handleSubscriptionNonTemplate(ctx, conn, asset.Spot, "unsubscribe", channelSubscriptions)
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
		return currency.EMPTYPAIR, 0, fmt.Errorf("category '%s' not supported for incoming symbol '%s'", category, symbol)
	}
	var err error
	for _, a := range assets {
		var cp currency.Pair
		if cp, err = by.MatchSymbolWithAvailablePairs(symbol, a, hasPotentialDelimiter(a)); err == nil {
			return cp, a, nil
		}
	}
	return currency.EMPTYPAIR, 0, err
}
