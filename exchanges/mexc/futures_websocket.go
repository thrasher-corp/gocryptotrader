package mexc

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/database/repository/trade"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/log"
)

var futuresWebsocketURL = "wss://contract.mexc.com/edge"

const (

	// Public channels
	channelFTickers     = "tickers"
	channelFTicker      = "ticker"
	channelFDeal        = "deal"
	channelFDepthFull   = "depth.full"
	channelFKline       = "kline"
	channelFFundingRate = "funding.rate"
	channelFIndexPrice  = "index.price"
	channelFFairPrice   = "fair.price"

	// Private channels
	channelLogin              = "login"
	channelFPersonalPositions = "personal.position"
	channelFPersonalAssets    = "personal.asset"
	channelFPersonalOrder     = "personal.order"
	channelFPersonalADLLevel  = "personal.adl.level"
	channelFPersonalRiskLimit = "personal.risk.limit"
	channelFPositionMode      = "personal.position.mode"

	// Ping response channel
	channelFPong = "pong"

	futuresSubscribeMethod   = "sub"
	futuresUnsubscribeMethod = "unsub"

	// futuresPingDelay keeps the contract connection alive; the venue closes an idle one
	futuresPingDelay = time.Second * 15

	// defaultFuturesDepthLevels is used when a depth subscription carries no level count:
	// the venue rejects depth.full without a limit
	defaultFuturesDepthLevels = 20
)

var defaultFuturesSubscriptions = subscription.List{
	{Enabled: true, Asset: asset.Futures, Channel: subscription.TickerChannel},
	{Enabled: true, Asset: asset.Futures, Channel: subscription.AllTradesChannel},
	{Enabled: true, Asset: asset.Futures, Channel: subscription.OrderbookChannel},
	{Enabled: true, Asset: asset.Futures, Channel: subscription.CandlesChannel, Interval: kline.FifteenMin},

	// MEXC futures has no private trades channel: own fills arrive through personal.order.
	{Enabled: true, Asset: asset.Futures, Channel: subscription.MyOrdersChannel, Authenticated: true},
	{Enabled: true, Asset: asset.Futures, Channel: subscription.MyAccountChannel, Authenticated: true},
}

// WsFuturesConnect established a futures websocket connection
func (e *Exchange) WsFuturesConnect(ctx context.Context, conn websocket.Connection) error {
	if !e.Websocket.IsEnabled() || !e.IsEnabled() {
		return websocket.ErrWebsocketNotEnabled
	}
	dialer := gws.Dialer{
		EnableCompression: true,
		ReadBufferSize:    8192,
		WriteBufferSize:   8192,
	}
	if err := conn.Dial(ctx, &dialer, http.Header{}, nil); err != nil {
		return err
	}
	conn.SetupPingHandler(request.UnAuth, websocket.PingHandler{
		Message:     []byte(`{"method": "ping"}`),
		MessageType: gws.TextMessage,
		// The venue drops an idle contract connection after roughly a minute: measured as
		// close 1005 every ~70s with the previous 15 minute delay
		Delay: futuresPingDelay,
	})
	return nil
}

// wsAuth authenticates a futures websocket connection
func (e *Exchange) wsAuth(ctx context.Context, conn websocket.Connection) error {
	credentials, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}
	param := &FWebsocketReqParam{
		RequestTime: time.Now().UnixMilli(),
		APIKey:      credentials.Key,
	}
	hmac, err := crypto.GetHMAC(crypto.HashSHA256,
		[]byte(param.APIKey+strconv.FormatInt(param.RequestTime, 10)),
		[]byte(credentials.Secret))
	if err != nil {
		return err
	}
	param.Signature = base64.StdEncoding.EncodeToString(hmac)
	data, err := conn.SendMessageReturnResponse(ctx, request.Auth, "rs.login", &WsSubscriptionPayload{
		Param:  param,
		Method: channelLogin,
	})
	if err != nil {
		return err
	}
	var result *WsFuturesLoginResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return err
	}
	if result.Data != "success" {
		return fmt.Errorf("code: %d, msg: %s", result.Code, result.Message)
	}
	return nil
}

// generateFuturesSubscriptions generates a futures default subscription instances
func (e *Exchange) generateFuturesSubscriptions() (subscription.List, error) {
	return defaultFuturesSubscriptions.ExpandTemplates(e)
}

// SubscribeFutures subscribes to a futures websocket channel
func (e *Exchange) SubscribeFutures(ctx context.Context, conn websocket.Connection, subscriptions subscription.List) error {
	return e.handleSubscriptionFuturesPayload(ctx, conn, subscriptions, futuresSubscribeMethod)
}

// UnsubscribeFutures unsubscribes to a futures websocket channel
func (e *Exchange) UnsubscribeFutures(ctx context.Context, conn websocket.Connection, subscriptions subscription.List) error {
	return e.handleSubscriptionFuturesPayload(ctx, conn, subscriptions, futuresUnsubscribeMethod)
}

// isFuturesSymbolChannel reports whether a futures channel is subscribed per contract and
// therefore requires a symbol in the subscription parameters
func isFuturesSymbolChannel(channel string) bool {
	return slices.Contains([]string{
		channelFTicker, channelFDeal, channelFDepthFull, channelFKline,
		channelFFundingRate, channelFIndexPrice, channelFFairPrice,
	}, channel)
}

// futuresSubscriptionParam builds the per-contract subscription parameters for a channel
func (e *Exchange) futuresSubscriptionParam(s *subscription.Subscription, pair currency.Pair) (*FWebsocketReqParam, error) {
	fPair, err := e.FormatExchangeCurrency(pair, asset.Futures)
	if err != nil {
		return nil, err
	}
	param := &FWebsocketReqParam{Symbol: fPair.String()}
	switch s.QualifiedChannel {
	case channelFDepthFull:
		param.Limit = s.Levels
		if param.Limit <= 0 {
			param.Limit = defaultFuturesDepthLevels
		}
	case channelFKline:
		intervalString, err := ContractIntervalString(s.Interval)
		if err != nil {
			return nil, err
		}
		param.Interval = intervalString
	}
	return param, nil
}

// futuresSubscriptionPayloads builds the venue payloads for a subscription list. Per-contract
// channels yield one payload per pair, each carrying the symbol: a payload without a symbol is
// rejected by the venue and takes the connection down with it.
func (e *Exchange) futuresSubscriptionPayloads(subscriptionItems subscription.List, method string) ([]*WsSubscriptionPayload, error) {
	payloads := make([]*WsSubscriptionPayload, 0, len(subscriptionItems))
	for x := range subscriptionItems {
		s := subscriptionItems[x]
		if s == nil {
			continue
		}
		if !isFuturesSymbolChannel(s.QualifiedChannel) {
			payloads = append(payloads, &WsSubscriptionPayload{Method: method + "." + s.QualifiedChannel})
			continue
		}
		if len(s.Pairs) == 0 {
			return nil, fmt.Errorf("%w: %s %s", currency.ErrCurrencyPairsEmpty, asset.Futures, s.QualifiedChannel)
		}
		for p := range s.Pairs {
			param, err := e.futuresSubscriptionParam(s, s.Pairs[p])
			if err != nil {
				return nil, err
			}
			payloads = append(payloads, &WsSubscriptionPayload{Method: method + "." + s.QualifiedChannel, Param: param})
		}
	}
	return payloads, nil
}

func (e *Exchange) handleSubscriptionFuturesPayload(ctx context.Context, conn websocket.Connection, subscriptionItems subscription.List, method string) error {
	payloads, err := e.futuresSubscriptionPayloads(subscriptionItems, method)
	if err != nil {
		return err
	}
	for i := range payloads {
		if err := conn.SendJSONMessage(ctx, request.UnAuth, payloads[i]); err != nil {
			return err
		}
	}
	// The subscription store is not updated by the transport: without this the manager
	// reports ErrSubscriptionsNotAdded and tears the connection down.
	if method == futuresUnsubscribeMethod {
		return e.Websocket.RemoveSubscriptions(conn, subscriptionItems...)
	}
	return e.Websocket.AddSuccessfulSubscriptions(conn, subscriptionItems...)
}

// WsHandleFuturesData processed futures websocket data
func (e *Exchange) WsHandleFuturesData(ctx context.Context, conn websocket.Connection, respRaw []byte) error {
	var resp *WsFuturesData
	if err := json.Unmarshal(respRaw, &resp); err != nil {
		return err
	}
	if resp.Channel == "" {
		if resp.Message != "" {
			log.Debugln(log.ExchangeSys, resp.Message)
		}
		return nil
	}
	if resp.Channel == "rs.login" {
		if !conn.IncomingWithData(resp.Channel, respRaw) {
			return e.Websocket.DataHandler.Send(ctx, websocket.UnhandledMessageWarning{
				Message: string(respRaw) + websocket.UnhandledMessage,
			})
		}
	}
	if resp.Channel == channelFPong {
		// Answer to {"method":"ping"}; carries no payload and needs no routing.
		return nil
	}
	cnlSplits := strings.Split(resp.Channel, ".")
	switch strings.Join(cnlSplits[1:], ".") {
	case channelFTickers:
		return e.processFuturesTickers(ctx, resp.Data)
	case channelFTicker:
		return e.processFuturesTicker(ctx, resp.Data)
	case channelFDeal:
		return e.processFuturesFillData(ctx, resp.Data, resp.Symbol)
	case channelFDepthFull:
		return e.processOrderbookDepth(resp.Data, resp.Symbol)
	case channelFKline:
		return e.processFuturesKlineData(ctx, resp.Data, resp.Symbol)
	case channelFFundingRate:
		var data *FuturesWsFundingRate
		if err := json.Unmarshal(resp.Data, &data); err != nil {
			return err
		}
		return e.Websocket.DataHandler.Send(ctx, data)
	case channelFIndexPrice:
		return e.processIndexPrice(ctx, resp.Data)
	case channelFFairPrice:
		return e.processFairPrice(ctx, resp.Data)
	case channelFPersonalPositions:
		return e.processPersonalPosition(ctx, resp.Data)
	case channelFPersonalAssets:
		return e.processPersonalAsset(ctx, resp.Data)
	case channelFPersonalOrder:
		return e.processPersonalOrder(ctx, resp.Data)
	case channelFPersonalADLLevel:
		var data *FuturesADLLevel
		if err := json.Unmarshal(resp.Data, &data); err != nil {
			return err
		}
		return e.Websocket.DataHandler.Send(ctx, data)
	case channelFPersonalRiskLimit:
		var data *FuturesWebsocketRiskLimit
		if err := json.Unmarshal(resp.Data, &data); err != nil {
			return err
		}
		return e.Websocket.DataHandler.Send(ctx, data)
	case channelFPositionMode:
		var data *FuturesPositionMode
		if err := json.Unmarshal(resp.Data, &data); err != nil {
			return err
		}
		return e.Websocket.DataHandler.Send(ctx, data)
	}
	return nil
}

func (e *Exchange) processPersonalPosition(ctx context.Context, data []byte) error {
	var resp *FuturesWsPersonalPosition
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	cp, err := currency.NewPairFromString(resp.Symbol)
	if err != nil {
		return err
	}
	var marginType margin.Type
	switch resp.OpenType {
	case 1:
		marginType = margin.Isolated
	case 2:
		marginType = margin.Multi
	}
	var oState order.Status
	switch resp.State {
	case 1:
		oState = order.Holding
	case 2:
		oState = order.SystemHolding
	case 3:
		oState = order.Closed
	}
	var oSide order.Side
	switch resp.PositionType {
	case 1:
		oSide = order.Long
	case 2:
		oSide = order.Short
	}
	return e.Websocket.DataHandler.Send(ctx, order.Detail{
		TimeInForce: order.GoodTillCancel,
		Leverage:    resp.Leverage,
		Price:       resp.HoldAvgPrice,
		Amount:      resp.HoldAvgPrice,
		Fee:         resp.HoldFee,
		Exchange:    e.Name,
		OrderID:     strconv.FormatInt(resp.PositionID, 10),
		Side:        oSide,
		Status:      oState,
		AssetType:   asset.Futures,
		Pair:        cp,
		MarginType:  marginType,
	})
}

func (e *Exchange) processPersonalAsset(ctx context.Context, data []byte) error {
	var resp *FuturesPersonalAsset
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, accounts.Change{
		AssetType: asset.Futures,
		Balance: accounts.Balance{
			Currency: currency.NewCode(resp.Currency),
			Total:    resp.AvailableBalance,
			Hold:     resp.FrozenBalance,
			Free:     resp.AvailableBalance - resp.FrozenBalance,
		},
	})
}

func (e *Exchange) processPersonalOrder(ctx context.Context, data []byte) error {
	var resp *WsFuturesPersonalOrder
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	cp, err := currency.NewPairFromString(resp.Symbol)
	if err != nil {
		return err
	}
	var oType order.Type
	var tif order.TimeInForce
	switch resp.OrderType {
	case 1:
		oType = order.Limit
	case 2:
		oType = order.Limit
		tif = order.PostOnly
	case 3:
		tif = order.ImmediateOrCancel
		oType = order.Market
	case 4:
		tif = order.FillOrKill
		oType = order.Market
	case 5:
		oType = order.Market
	case 6:
		oType = order.Chase
	}
	var oSide order.Side
	switch resp.Side {
	case 1, 4:
		oSide = order.Long
	case 2, 3:
		oSide = order.Short
	}
	var oState order.Status
	switch resp.State {
	case 1:
		oState = order.AnyStatus
	case 2:
		oState = order.PartiallyFilled
	case 3:
		oState = order.Filled
	case 4:
		oState = order.Cancelled
	case 5:
		oState = order.Expired
	}
	var marginType margin.Type
	switch resp.OpenType {
	case 1:
		marginType = margin.Isolated
	case 2:
		marginType = margin.Multi
	}
	return e.Websocket.DataHandler.Send(ctx, &order.Detail{
		Exchange:             e.Name,
		Pair:                 cp,
		TimeInForce:          tif,
		Leverage:             resp.Leverage,
		Price:                resp.Price,
		Amount:               resp.Volume,
		AverageExecutedPrice: resp.DealAvgPrice,
		QuoteAmount:          resp.DealAvgPrice * resp.DealVol,
		ExecutedAmount:       resp.DealVol,
		RemainingAmount:      resp.Volume - resp.DealVol,
		FeeAsset:             currency.NewCode(resp.FeeCurrency),
		OrderID:              resp.OrderID,
		ClientOrderID:        resp.ExternalOid,
		Type:                 oType,
		Side:                 oSide,
		Status:               oState,
		AssetType:            asset.Futures,
		LastUpdated:          resp.UpdateTime.Time(),
		MarginType:           marginType,
	})
}

func (e *Exchange) processFairPrice(ctx context.Context, data []byte) error {
	var resp *PriceAndSymbol
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	cp, err := currency.NewPairFromString(resp.Symbol)
	if err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, ticker.Price{
		ExchangeName: e.Name,
		IndexPrice:   resp.Price,
		Pair:         cp,
	})
}

func (e *Exchange) processIndexPrice(ctx context.Context, data []byte) error {
	var resp *PriceAndSymbol
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	cp, err := currency.NewPairFromString(resp.Symbol)
	if err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, ticker.Price{
		ExchangeName: e.Name,
		IndexPrice:   resp.Price,
		Pair:         cp,
	})
}

func (e *Exchange) processFuturesKlineData(ctx context.Context, data []byte, symbol string) error {
	var resp *FuturesWebsocketKline
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	cp, err := currency.NewPairFromString(symbol)
	if err != nil {
		return err
	}
	interval, err := IntervalFromString(resp.Interval)
	if err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, kline.Item{
		Pair:     cp,
		Exchange: e.Name,
		Asset:    asset.Futures,
		Interval: interval,
		Candles: []kline.Candle{
			{
				Open:   resp.OpeningPrice,
				High:   resp.HighestPrice,
				Low:    resp.LowestPrice,
				Close:  resp.ClosePrice,
				Volume: resp.TotalTransactionVolume,
			},
		},
	})
}

func (e *Exchange) processOrderbookDepth(data []byte, symbol string) error {
	var resp *FuturesWsDepth
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	cp, err := currency.NewPairFromString(symbol)
	if err != nil {
		return err
	}
	return e.Websocket.Orderbook.LoadSnapshot(&orderbook.Book{
		Bids:        resp.Bids.Levels(),
		Asks:        resp.Asks.Levels(),
		Exchange:    e.Name,
		Pair:        cp,
		Asset:       asset.Futures,
		LastUpdated: time.Now(),
	})
}

func (e *Exchange) processFuturesFillData(ctx context.Context, data []byte, symbol string) error {
	var resp []FuturesTransactionFills
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	cp, err := currency.NewPairFromString(symbol)
	if err != nil {
		return err
	}
	for x := range resp {
		var oSide order.Side
		switch resp[x].TransactionDirection {
		case 1:
			oSide = order.Buy
		case 2:
			oSide = order.Sell
		}
		if err := e.Websocket.DataHandler.Send(ctx, &trade.Data{
			Timestamp: resp[x].TransationTime.Time(),
			Exchange:  e.Name,
			AssetType: asset.Futures.String(),
			Base:      cp.Base.String(),
			Quote:     cp.Quote.String(),
			Side:      oSide.String(),
			Price:     resp[x].Price,
			Amount:    resp[x].Volume,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (e *Exchange) processFuturesTicker(ctx context.Context, data []byte) error {
	var resp *FuturesPriceTickerDetail
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	cp, err := currency.NewPairFromString(resp.Symbol)
	if err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, &ticker.Price{
		Last: resp.LastPrice,
		High: resp.High24Price,
		Low:  resp.Lower24Price,
		// Ask1/Bid1 are the best offer and bid. MinAskPrice/MaxBidPrice are the venue's
		// price-limit band (e.g. last 14.022 -> band 11.222/16.833) and are not a BBO.
		Ask:          resp.Ask1,
		Bid:          resp.Bid1,
		Volume:       resp.Volume24,
		IndexPrice:   resp.IndexPrice,
		Pair:         cp,
		ExchangeName: e.Name,
		AssetType:    asset.Futures,
		LastUpdated:  resp.Timestamp.Time(),
	})
}

func (e *Exchange) processFuturesTickers(ctx context.Context, data []byte) error {
	var tickers []FuturesTickerItem
	if err := json.Unmarshal(data, &tickers); err != nil {
		return err
	}
	// The tickers channel is a venue-wide broadcast: every contract arrives regardless of
	// what is subscribed. Publishing all of them floods consumers with pairs the platform
	// does not track, so only enabled futures pairs are forwarded.
	enabledPairs, err := e.GetEnabledPairs(asset.Futures)
	if err != nil {
		return nil //nolint:nilerr // futures not enabled: nothing to publish, not a stream error
	}
	for t := range tickers {
		cp, err := currency.NewPairFromString(tickers[t].Symbol)
		if err != nil {
			return err
		}
		if !enabledPairs.Contains(cp, true) {
			continue
		}
		if err := e.Websocket.DataHandler.Send(ctx, &ticker.Price{
			ExchangeName: e.Name,
			Pair:         cp,
			AssetType:    asset.Futures,
			Last:         tickers[t].LastPrice,
			MarkPrice:    tickers[t].FairPrice,
			Volume:       tickers[t].Volume24,
		}); err != nil {
			return err
		}
	}
	return nil
}
