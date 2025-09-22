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
	"github.com/thrasher-corp/gocryptotrader/log"
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
	// TODO: Implement DCP (Disconnection Protect) subscription

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

var (
	errUnhandledStreamData = errors.New("unhandled stream data")
	errUnsupportedCategory = errors.New("unsupported category")
)

// WsConnect connects to a websocket feed
func (e *Exchange) WsConnect(ctx context.Context, conn websocket.Connection) error {
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

// WebsocketAuthenticatePrivateConnection sends an authentication message to the private websocket for inbound account
// data
func (e *Exchange) WebsocketAuthenticatePrivateConnection(ctx context.Context, conn websocket.Connection) error {
	req, err := e.GetAuthenticationPayload(ctx, strconv.FormatInt(conn.GenerateMessageID(false), 10))
	if err != nil {
		return err
	}
	resp, err := conn.SendMessageReturnResponse(ctx, wsSubscriptionEPL, req.RequestID, req)
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

// WebsocketAuthenticateTradeConnection sends an authentication message to the private trade websocket for outbound
// account data
func (e *Exchange) WebsocketAuthenticateTradeConnection(ctx context.Context, conn websocket.Connection) error {
	// request ID is not returned with the response, a workaround in the trade connection handler monitors the response
	// for the operation type "auth", which is then set in the response match key.
	req, err := e.GetAuthenticationPayload(ctx, "auth")
	if err != nil {
		return err
	}
	resp, err := conn.SendMessageReturnResponse(ctx, wsSubscriptionEPL, req.RequestID, req)
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
		c, ok := retCode[response.ReturnCode]
		if !ok {
			c = "unknown return error code"
		}
		return fmt.Errorf("%s failed - code:%d [%v] msg:%s", response.Operation, response.ReturnCode, c, response.ReturnMessage)
	}
	return nil
}

// GetAuthenticationPayload returns the authentication payload for the websocket connection to upgrade the connection.
func (e *Exchange) GetAuthenticationPayload(ctx context.Context, requestID string) (*Authenticate, error) {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return nil, err
	}
	expires := time.Now().Add(time.Hour * 6).UnixMilli()
	hmac, err := crypto.GetHMAC(crypto.HashSHA256, []byte("GET/realtime"+strconv.FormatInt(expires, 10)), []byte(creds.Secret))
	if err != nil {
		return nil, err
	}
	return &Authenticate{
		RequestID: requestID,
		Operation: "auth",
		Args:      []any{creds.Key, expires, hex.EncodeToString(hmac)},
	}, nil
}

func (e *Exchange) handleSubscriptions(conn websocket.Connection, operation string, subs subscription.List) (args []SubscriptionArgument, err error) {
	subs, err = subs.ExpandTemplates(e)
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

// generateSubscriptions generates default subscription
func (e *Exchange) generateSubscriptions() (subscription.List, error) {
	return e.Features.Subscriptions.ExpandTemplates(e)
}

// GetSubscriptionTemplate returns a subscription channel template
func (e *Exchange) GetSubscriptionTemplate(_ *subscription.Subscription) (*template.Template, error) {
	return template.New("master.tmpl").Funcs(template.FuncMap{
		"channelName":      channelName,
		"isSymbolChannel":  isSymbolChannel,
		"intervalToString": intervalToString,
		"getCategoryName":  getCategoryName,
	}).Parse(subTplText)
}

func (e *Exchange) wsHandleTradeData(conn websocket.Connection, respRaw []byte) error {
	var response struct {
		RequestID string `json:"reqId"`
		Operation string `json:"op"`
	}
	if err := json.Unmarshal(respRaw, &response); err != nil {
		return err
	}

	if response.RequestID != "" {
		return conn.RequireMatchWithData(response.RequestID, respRaw)
	}

	switch response.Operation {
	case "auth": // When authenticating the connection there is no request ID, so a static value is used.
		return conn.RequireMatchWithData(response.Operation, respRaw)
	case "pong":
		return nil
	default:
		return fmt.Errorf("%w for trade: %v", errUnhandledStreamData, string(respRaw))
	}
}

func (e *Exchange) wsHandleData(conn websocket.Connection, assetType asset.Item, respRaw []byte) error {
	var result WebsocketResponse
	if err := json.Unmarshal(respRaw, &result); err != nil {
		return err
	}
	if result.Topic == "" {
		return e.handleNoTopicWebsocketResponse(conn, &result, respRaw)
	}
	topicSplit := strings.Split(result.Topic, ".")
	switch topicSplit[0] {
	case chanOrderbook:
		return e.wsProcessOrderbook(assetType, &result)
	case chanPublicTrade:
		return e.wsProcessPublicTrade(assetType, &result)
	case chanPublicTicker:
		return e.wsProcessPublicTicker(assetType, &result)
	case chanKline:
		return e.wsProcessKline(assetType, &result, topicSplit)
	case chanLiquidation:
		return e.wsProcessLiquidation(&result)
	case chanLeverageTokenKline:
		return e.wsProcessLeverageTokenKline(assetType, &result, topicSplit)
	case chanLeverageTokenTicker:
		return e.wsProcessLeverageTokenTicker(assetType, &result)
	case chanLeverageTokenNav:
		return e.wsLeverageTokenNav(&result)
	}
	return fmt.Errorf("%w %s", errUnhandledStreamData, string(respRaw))
}

func (e *Exchange) wsHandleAuthenticatedData(ctx context.Context, conn websocket.Connection, respRaw []byte) error {
	var result WebsocketResponse
	if err := json.Unmarshal(respRaw, &result); err != nil {
		return err
	}
	if result.Topic == "" {
		return e.handleNoTopicWebsocketResponse(conn, &result, respRaw)
	}
	topicSplit := strings.Split(result.Topic, ".")
	switch topicSplit[0] {
	case chanPositions:
		return e.wsProcessPosition(&result)
	case chanExecution:
		return e.wsProcessExecution(&result)
	case chanOrder:
		// Use first order's orderLinkId to match with an entire batch of order change requests
		if id, err := jsonparser.GetString(respRaw, "data", "[0]", "orderLinkId"); err == nil {
			if conn.IncomingWithData(id, respRaw) {
				return nil // If the data has been routed, return
			}
		}
		return e.wsProcessOrder(&result)
	case chanWallet:
		return e.wsProcessWalletPushData(ctx, respRaw)
	case chanGreeks:
		return e.wsProcessGreeks(respRaw)
	}
	return fmt.Errorf("%w %s", errUnhandledStreamData, string(respRaw))
}

func (e *Exchange) handleNoTopicWebsocketResponse(conn websocket.Connection, result *WebsocketResponse, respRaw []byte) error {
	switch result.Operation {
	case "subscribe", "unsubscribe", "auth":
		if result.RequestID != "" {
			return conn.RequireMatchWithData(result.RequestID, respRaw)
		}
	case "ping", "pong":
	default:
		e.Websocket.DataHandler <- websocket.UnhandledMessageWarning{Message: string(respRaw)}
	}
	return nil
}

func (e *Exchange) wsProcessGreeks(resp []byte) error {
	var result GreeksResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return err
	}
	e.Websocket.DataHandler <- &result
	return nil
}

func (e *Exchange) wsProcessWalletPushData(ctx context.Context, resp []byte) error {
	var result WebsocketWallet
	if err := json.Unmarshal(resp, &result); err != nil {
		return err
	}
	creds, err := e.GetCredentials(ctx)
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
	e.Websocket.DataHandler <- changes
	return account.ProcessChange(e.Name, changes, creds)
}

// wsProcessOrder the order stream to see changes to your orders in real-time.
func (e *Exchange) wsProcessOrder(resp *WebsocketResponse) error {
	var result []WebsocketOrderDetails
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return err
	}
	execution := make([]order.Detail, len(result))
	for x := range result {
		cp, a, err := e.matchPairAssetFromResponse(result[x].Category, result[x].Symbol)
		if err != nil {
			return err
		}
		orderType, err := order.StringToOrderType(result[x].OrderType)
		if err != nil {
			return err
		}
		tif, err := order.StringToTimeInForce(result[x].TimeInForce)
		if err != nil {
			return err
		}
		execution[x] = order.Detail{
			TimeInForce:          tif,
			Amount:               result[x].Quantity.Float64(),
			Exchange:             e.Name,
			OrderID:              result[x].OrderID,
			ClientOrderID:        result[x].OrderLinkID,
			Side:                 result[x].Side,
			Type:                 orderType,
			Pair:                 cp,
			Cost:                 result[x].CumulativeExecutedQuantity.Float64() * result[x].AveragePrice.Float64(),
			Fee:                  result[x].CumulativeExecutedFee.Float64(),
			AssetType:            a,
			Status:               StringToOrderStatus(result[x].OrderStatus),
			Price:                result[x].Price.Float64(),
			ExecutedAmount:       result[x].CumulativeExecutedQuantity.Float64(),
			AverageExecutedPrice: result[x].AveragePrice.Float64(),
			Date:                 result[x].CreatedTime.Time(),
			LastUpdated:          result[x].UpdatedTime.Time(),
		}
	}
	e.Websocket.DataHandler <- execution
	return nil
}

func (e *Exchange) wsProcessExecution(resp *WebsocketResponse) error {
	var result WsExecutions
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return err
	}
	executions := make([]fill.Data, len(result))
	for x := range result {
		cp, a, err := e.matchPairAssetFromResponse(result[x].Category, result[x].Symbol)
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
			Exchange:      e.Name,
			AssetType:     a,
			CurrencyPair:  cp,
			Side:          side,
			OrderID:       result[x].OrderID,
			ClientOrderID: result[x].OrderLinkID,
			Price:         result[x].ExecPrice.Float64(),
			Amount:        result[x].ExecQty.Float64(),
		}
	}
	e.Websocket.DataHandler <- executions
	return nil
}

func (e *Exchange) wsProcessPosition(resp *WebsocketResponse) error {
	var result WsPositions
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return err
	}
	e.Websocket.DataHandler <- result
	return nil
}

func (e *Exchange) wsLeverageTokenNav(resp *WebsocketResponse) error {
	var result LTNav
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return err
	}
	e.Websocket.DataHandler <- result
	return nil
}

func (e *Exchange) wsProcessLeverageTokenTicker(assetType asset.Item, resp *WebsocketResponse) error {
	var result TickerWebsocket
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return err
	}
	cp, err := e.MatchSymbolWithAvailablePairs(result.Symbol, assetType, hasPotentialDelimiter(assetType))
	if err != nil {
		return err
	}
	e.Websocket.DataHandler <- &ticker.Price{
		Last:         result.LastPrice.Float64(),
		High:         result.HighPrice24H.Float64(),
		Low:          result.LowPrice24H.Float64(),
		Pair:         cp,
		ExchangeName: e.Name,
		AssetType:    assetType,
		LastUpdated:  resp.PushTimestamp.Time(),
	}
	return nil
}

func (e *Exchange) wsProcessLeverageTokenKline(assetType asset.Item, resp *WebsocketResponse, topicSplit []string) error {
	var result LTKlines
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return err
	}
	cp, err := e.MatchSymbolWithAvailablePairs(topicSplit[2], assetType, hasPotentialDelimiter(assetType))
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
			Exchange:   e.Name,
			StartTime:  result[x].Start.Time(),
			CloseTime:  result[x].End.Time(),
			Interval:   interval.String(),
			OpenPrice:  result[x].Open.Float64(),
			ClosePrice: result[x].Close.Float64(),
			HighPrice:  result[x].High.Float64(),
			LowPrice:   result[x].Low.Float64(),
		}
	}
	e.Websocket.DataHandler <- result
	return nil
}

func (e *Exchange) wsProcessLiquidation(resp *WebsocketResponse) error {
	var result WebsocketLiquidation
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return err
	}
	e.Websocket.DataHandler <- result
	return nil
}

func (e *Exchange) wsProcessKline(assetType asset.Item, resp *WebsocketResponse, topicSplit []string) error {
	var result WsKlines
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return err
	}
	cp, err := e.MatchSymbolWithAvailablePairs(topicSplit[2], assetType, hasPotentialDelimiter(assetType))
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
			Exchange:   e.Name,
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
	e.Websocket.DataHandler <- spotCandlesticks
	return nil
}

func (e *Exchange) wsProcessPublicTicker(assetType asset.Item, resp *WebsocketResponse) error {
	var tickResp TickerWebsocket
	if err := json.Unmarshal(resp.Data, &tickResp); err != nil {
		return err
	}

	p, err := e.MatchSymbolWithAvailablePairs(tickResp.Symbol, assetType, hasPotentialDelimiter(assetType))
	if err != nil {
		return err
	}

	tick := &ticker.Price{Pair: p, ExchangeName: e.Name, AssetType: assetType}
	if resp.Type != "snapshot" {
		// ticker updates may be partial, so we need to update the current ticker
		tick, err = e.GetCachedTicker(p, assetType)
		if err != nil {
			return err
		}
	}
	updateTicker(tick, &tickResp)
	tick.LastUpdated = resp.PushTimestamp.Time()
	if err := ticker.ProcessTicker(tick); err != nil {
		return err
	}
	e.Websocket.DataHandler <- tick
	return nil
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

func (e *Exchange) wsProcessPublicTrade(assetType asset.Item, resp *WebsocketResponse) error {
	var result WebsocketPublicTrades
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return err
	}
	tradeDatas := make([]trade.Data, len(result))
	for x := range result {
		cp, err := e.MatchSymbolWithAvailablePairs(result[x].Symbol, assetType, hasPotentialDelimiter(assetType))
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
			Exchange:     e.Name,
			Price:        result[x].Price.Float64(),
			Amount:       result[x].Size.Float64(),
			Side:         side,
			TID:          result[x].TradeID,
		}
	}
	return trade.AddTradesToBuffer(tradeDatas...)
}

func (e *Exchange) wsProcessOrderbook(assetType asset.Item, resp *WebsocketResponse) error {
	var result WsOrderbookDetail
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return err
	}

	cp, err := e.MatchSymbolWithAvailablePairs(result.Symbol, assetType, hasPotentialDelimiter(assetType))
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
		return e.Websocket.Orderbook.LoadSnapshot(&orderbook.Book{
			Pair:         cp,
			Exchange:     e.Name,
			Asset:        assetType,
			LastUpdated:  resp.OrderbookLastUpdated.Time(),
			LastUpdateID: result.UpdateID,
			LastPushed:   resp.PushTimestamp.Time(),
			Asks:         asks,
			Bids:         bids,
		})
	}
	return e.Websocket.Orderbook.Update(&orderbook.Update{
		Pair:       cp,
		Asks:       asks,
		Bids:       bids,
		Asset:      assetType,
		UpdateID:   result.UpdateID,
		UpdateTime: resp.OrderbookLastUpdated.Time(),
		LastPushed: resp.PushTimestamp.Time(),
		AllowEmpty: true,
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
	case chanPositions, chanExecution, chanOrder, chanWallet:
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
func (e *Exchange) submitDirectSubscription(ctx context.Context, conn websocket.Connection, a asset.Item, operation string, channelsToSubscribe subscription.List) error {
	payloads, err := e.directSubscriptionPayload(conn, a, operation, channelsToSubscribe)
	if err != nil {
		return err
	}

	op := e.Websocket.AddSubscriptions
	if operation == "unsubscribe" {
		op = e.Websocket.RemoveSubscriptions
	}

	for _, payload := range payloads {
		if a == asset.Options {
			// The options connection does not send the subscription request id back with the subscription notification payload
			// therefore the code doesn't wait for the response to check whether the subscription is successful or not.
			if err := conn.SendJSONMessage(ctx, wsSubscriptionEPL, payload); err != nil {
				return err
			}
		} else {
			response, err := conn.SendMessageReturnResponse(ctx, wsSubscriptionEPL, payload.RequestID, payload)
			if err != nil {
				return err
			}
			var resp SubscriptionResponse
			if err := json.Unmarshal(response, &resp); err != nil {
				return err
			}
			if !resp.Success {
				return fmt.Errorf("%s with request ID %s msg: %s", resp.Operation, resp.RequestID, resp.ReturnMessage)
			}
		}
		if err := op(conn, payload.associatedSubs...); err != nil {
			return err
		}
	}
	return nil
}

// TODO: Remove this function when template expansion is across all assets
func (e *Exchange) directSubscriptionPayload(conn websocket.Connection, assetType asset.Item, operation string, channelsToSubscribe subscription.List) ([]SubscriptionArgument, error) {
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
	pairFmt, err := e.GetPairFormat(assetType, true)
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
		case chanPositions, chanExecution, chanOrder, chanWallet, chanGreeks:
			if chanMap[s.Channel] {
				continue
			}
			authArg.Arguments = append(authArg.Arguments, s.Channel)
			// add channel name to map so we only subscribe to channel once
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
func (e *Exchange) generateAuthSubscriptions() (subscription.List, error) {
	if !e.Websocket.CanUseAuthenticatedEndpoints() {
		return nil, nil
	}

	for _, configSub := range e.Config.Features.Subscriptions.Enabled() {
		if configSub.Authenticated {
			log.Warnf(log.WebsocketMgr, "%s has an authenticated subscription %q in config which is not supported. Please remove.", e.Name, configSub.Channel)
			configSub.Enabled = false
		}
	}

	var subscriptions subscription.List
	// TODO: Implement DCP (Disconnection Protect) subscription
	for _, channel := range []string{chanPositions, chanExecution, chanOrder, chanWallet} {
		subscriptions = append(subscriptions, &subscription.Subscription{Channel: channel, Asset: asset.All})
	}
	return subscriptions, nil
}

func (e *Exchange) authSubscribe(ctx context.Context, conn websocket.Connection, channelSubscriptions subscription.List) error {
	return e.submitDirectSubscription(ctx, conn, asset.Spot, "subscribe", channelSubscriptions)
}

func (e *Exchange) authUnsubscribe(ctx context.Context, conn websocket.Connection, channelSubscriptions subscription.List) error {
	return e.submitDirectSubscription(ctx, conn, asset.Spot, "unsubscribe", channelSubscriptions)
}

// matchPairAssetFromResponse returns the currency pair and asset type based on the category and symbol. Used with a dedicated
// auth connection where multiple asset type changes are piped through a single connection.
func (e *Exchange) matchPairAssetFromResponse(category, symbol string) (currency.Pair, asset.Item, error) {
	assets := make([]asset.Item, 0, 2)
	switch category {
	case cSpot:
		assets = append(assets, asset.Spot)
	case cInverse:
		assets = append(assets, asset.CoinMarginedFutures)
	case cLinear:
		assets = append(assets, asset.USDTMarginedFutures, asset.USDCMarginedFutures)
	case cOption:
		assets = append(assets, asset.Options)
	default:
		return currency.EMPTYPAIR, 0, fmt.Errorf("incoming symbol %q %w: %q", symbol, errUnsupportedCategory, category)
	}
	for _, a := range assets {
		cp, err := e.MatchSymbolWithAvailablePairs(symbol, a, hasPotentialDelimiter(a))
		if err != nil {
			if !errors.Is(err, currency.ErrPairNotFound) {
				return currency.EMPTYPAIR, 0, fmt.Errorf("%w for symbol %q: %q", err, category, symbol)
			}
			continue
		}
		return cp, a, nil
	}
	return currency.EMPTYPAIR, 0, currency.ErrPairNotFound
}
