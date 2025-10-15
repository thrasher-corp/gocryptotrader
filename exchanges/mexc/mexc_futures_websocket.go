package mexc

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/database/repository/trade"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
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

var futuresWsURL = "wss://contract.mexc.com/edge"

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
)

var defaultFuturesSubscriptions = []string{
	channelFTickers,
	channelFDeal,
	channelFDepthFull,
	channelFKline,
}

// WsFuturesConnect established a futures websocket connection
func (e *Exchange) WsFuturesConnect() error {
	if !e.Websocket.IsEnabled() || !e.IsEnabled() {
		return websocket.ErrWebsocketNotEnabled
	}
	dialer := gws.Dialer{
		EnableCompression: true,
		ReadBufferSize:    8192,
		WriteBufferSize:   8192,
	}
	if err := e.Websocket.SetWebsocketURL(futuresWsURL, false, true); err != nil {
		return err
	}
	if err := e.Websocket.Conn.Dial(context.Background(), &dialer, http.Header{}); err != nil {
		return err
	}
	e.Websocket.Wg.Add(1)
	go e.wsFuturesReadData(e.Websocket.Conn)
	if e.Websocket.CanUseAuthenticatedEndpoints() {
		if err := e.wsAuth(); err != nil {
			log.Warnf(log.ExchangeSys, "authentication error: %v", err)
			e.Websocket.SetCanUseAuthenticatedEndpoints(false)
		}
	}
	if e.Verbose {
		log.Debugf(log.ExchangeSys, "Successful connection to %v\n", e.Websocket.GetWebsocketURL())
	}
	return nil
}

// wsAuth authenticates a futures websocket connection
func (e *Exchange) wsAuth() error {
	credentials, err := e.GetCredentials(context.Background())
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
	data, err := e.Websocket.Conn.SendMessageReturnResponse(context.Background(), request.Auth, "rs.login", &WsSubscriptionPayload{
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

// GenerateDefaultFuturesSubscriptions generates a futures default subscription instances
func (e *Exchange) GenerateDefaultFuturesSubscriptions() (subscription.List, error) {
	channels := defaultFuturesSubscriptions
	if e.Websocket.CanUseAuthenticatedEndpoints() {
		channels = append(channels, channelFPersonalPositions, channelFPersonalAssets, channelFPersonalOrder, channelFPersonalADLLevel, channelFPersonalRiskLimit, channelFPositionMode)
	}
	enabledPairs, err := e.GetEnabledPairs(asset.Futures)
	if err != nil {
		return nil, err
	}
	subscriptionsList := make(subscription.List, len(channels))
	for c := range channels {
		switch channels[c] {
		case channelFTicker, channelFDeal, channelFDepthFull, channelFFundingRate, channelFIndexPrice, channelFFairPrice:
			subscriptionsList[c] = &subscription.Subscription{
				Channel: channels[c],
				Pairs:   enabledPairs,
			}
		case channelFKline:
			subscriptionsList[c] = &subscription.Subscription{
				Channel:  channels[c],
				Pairs:    enabledPairs,
				Interval: kline.FifteenMin,
			}
		case channelFTickers, channelFPersonalPositions, channelFPersonalAssets, channelFPersonalOrder,
			channelFPersonalADLLevel, channelFPersonalRiskLimit, channelFPositionMode:
			subscriptionsList[c] = &subscription.Subscription{
				Channel: channels[c],
			}
		}
	}
	return subscriptionsList, nil
}

// SubscribeFutures subscribes to a futures websocket channel
func (e *Exchange) SubscribeFutures(subscriptions subscription.List) error {
	return e.handleSubscriptionFuturesPayload(subscriptions, "sub")
}

// UnsubscribeFutures unsubscribes to a futures websocket channel
func (e *Exchange) UnsubscribeFutures(subscriptions subscription.List) error {
	return e.handleSubscriptionFuturesPayload(subscriptions, "unsub")
}

func (e *Exchange) handleSubscriptionFuturesPayload(subscriptionItems subscription.List, method string) error {
	for x := range subscriptionItems {
		switch subscriptionItems[x].Channel {
		case channelFDeal, channelFTicker, channelFDepthFull, channelFKline, channelFFundingRate, channelFIndexPrice, channelFFairPrice:
			params := make([]FWebsocketReqParam, len(subscriptionItems[x].Pairs))
			for p := range subscriptionItems[x].Pairs {
				params[p].Symbol = subscriptionItems[x].Pairs[p].String()
				switch subscriptionItems[x].Channel {
				case channelFDeal:
					params[p].Compress = true
					params[p].Limit = subscriptionItems[x].Levels
				case channelFKline:
					intervalString, err := ContractIntervalString(subscriptionItems[x].Interval)
					if err != nil {
						return err
					}
					params[p].Interval = intervalString
				}
				if err := e.Websocket.Conn.SendJSONMessage(context.Background(), request.UnAuth, &WsSubscriptionPayload{
					Method: method + "." + subscriptionItems[x].Channel,
					Param:  &params[p],
				}); err != nil {
					return err
				}
			}
		default:
			if err := e.Websocket.Conn.SendJSONMessage(context.Background(), request.UnAuth, &WsSubscriptionPayload{
				Method: method + "." + subscriptionItems[x].Channel,
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

// wsFuturesReadData sends futures assets related msgs from public and auth websockets to data handler
func (e *Exchange) wsFuturesReadData(ws websocket.Connection) {
	defer e.Websocket.Wg.Done()
	for {
		resp := ws.ReadMessage()
		if len(resp.Raw) == 0 {
			return
		}
		if err := e.WsHandleFuturesData(resp.Raw); err != nil {
			e.Websocket.DataHandler <- err
		}
	}
}

// WsHandleFuturesData processed futures websocket data
func (e *Exchange) WsHandleFuturesData(respRaw []byte) error {
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
		if !e.Websocket.Match.IncomingWithData(resp.Channel, respRaw) {
			e.Websocket.DataHandler <- websocket.UnhandledMessageWarning{
				Message: string(respRaw) + websocket.UnhandledMessage,
			}
		}
	}
	cnlSplits := strings.Split(resp.Channel, ".")
	switch strings.Join(cnlSplits[1:], ".") {
	case channelFTickers:
		return e.processFuturesTickers(resp.Data)
	case channelFTicker:
		return e.processFuturesTicker(resp.Data)
	case channelFDeal:
		return e.processFuturesFillData(resp.Data, resp.Symbol)
	case channelFDepthFull:
		return e.processOrderbookDepth(resp.Data, resp.Symbol)
	case channelFKline:
		return e.processFuturesKlineData(resp.Data, resp.Symbol)
	case channelFFundingRate:
		var data *FuturesWsFundingRate
		if err := json.Unmarshal(resp.Data, &data); err != nil {
			return err
		}
		e.Websocket.DataHandler <- data
		return nil
	case channelFIndexPrice:
		return e.processIndexPrice(resp.Data)
	case channelFFairPrice:
		return e.processFairPrice(resp.Data)
	case channelFPersonalPositions:
		return e.processPersonalPosition(resp.Data)
	case channelFPersonalAssets:
		return e.processPersonalAsset(resp.Data)
	case channelFPersonalOrder:
		return e.processPersonalOrder(resp.Data)
	case channelFPersonalADLLevel:
		var data *FuturesADLLevel
		if err := json.Unmarshal(resp.Data, &data); err != nil {
			return err
		}
		e.Websocket.DataHandler <- data
		return nil
	case channelFPersonalRiskLimit:
		var data *FuturesWebsocketRiskLimit
		if err := json.Unmarshal(resp.Data, &data); err != nil {
			return err
		}
		e.Websocket.DataHandler <- data
		return nil
	case channelFPositionMode:
		var data *FuturesPositionMode
		if err := json.Unmarshal(resp.Data, &data); err != nil {
			return err
		}
		e.Websocket.DataHandler <- data
		return nil
	}
	return nil
}

func (e *Exchange) processPersonalPosition(data []byte) error {
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
	e.Websocket.DataHandler <- order.Detail{
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
	}
	return nil
}

func (e *Exchange) processPersonalAsset(data []byte) error {
	var resp *FuturesPersonalAsset
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	e.Websocket.DataHandler <- account.Change{
		AssetType: asset.Futures,
		Balance: &account.Balance{
			Currency: currency.NewCode(resp.Currency),
			Total:    resp.AvailableBalance,
			Hold:     resp.FrozenBalance,
			Free:     resp.AvailableBalance - resp.FrozenBalance,
		},
	}
	return nil
}

func (e *Exchange) processPersonalOrder(data []byte) error {
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
	e.Websocket.DataHandler <- &order.Detail{
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
		Exchange:             e.Name,
		OrderID:              resp.OrderID,
		ClientOrderID:        resp.ExternalOid,
		Type:                 oType,
		Side:                 oSide,
		Status:               oState,
		AssetType:            asset.Futures,
		LastUpdated:          resp.UpdateTime.Time(),
		MarginType:           marginType,
	}
	return nil
}

func (e *Exchange) processFairPrice(data []byte) error {
	var resp *PriceAndSymbol
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	cp, err := currency.NewPairFromString(resp.Symbol)
	if err != nil {
		return err
	}
	e.Websocket.DataHandler <- ticker.Price{
		IndexPrice: resp.Price,
		Pair:       cp,
	}
	return nil
}

func (e *Exchange) processIndexPrice(data []byte) error {
	var resp *PriceAndSymbol
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	cp, err := currency.NewPairFromString(resp.Symbol)
	if err != nil {
		return err
	}
	e.Websocket.DataHandler <- ticker.Price{
		IndexPrice: resp.Price,
		Pair:       cp,
	}
	return nil
}

func (e *Exchange) processFuturesKlineData(data []byte, symbol string) error {
	var resp *FuturesWebsocketKline
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	cp, err := currency.NewPairFromString(symbol)
	if err != nil {
		return err
	}
	e.Websocket.DataHandler <- websocket.KlineData{
		Pair:       cp,
		Exchange:   e.Name,
		AssetType:  asset.Spot,
		Interval:   resp.Interval,
		OpenPrice:  resp.OpeningPrice,
		Timestamp:  resp.TradeTime.Time(),
		HighPrice:  resp.HighestPrice,
		LowPrice:   resp.LowestPrice,
		ClosePrice: resp.ClosePrice,
		Volume:     resp.TotalTransactionVolume,
	}
	return nil
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

func (e *Exchange) processFuturesFillData(data []byte, symbol string) error {
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
		e.Websocket.DataHandler <- &trade.Data{
			Timestamp: resp[x].TransationTime.Time(),
			Exchange:  e.Name,
			AssetType: asset.Futures.String(),
			Base:      cp.Base.String(),
			Quote:     cp.Quote.String(),
			Side:      oSide.String(),
			Price:     resp[x].Price,
			Amount:    resp[x].Volume,
		}
	}
	return nil
}

func (e *Exchange) processFuturesTicker(data []byte) error {
	var resp *FuturesPriceTickerDetail
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	cp, err := currency.NewPairFromString(resp.Symbol)
	if err != nil {
		return err
	}
	e.Websocket.DataHandler <- &ticker.Price{
		Last:         resp.LastPrice,
		High:         resp.High24Price,
		Low:          resp.Lower24Price,
		Ask:          resp.MinAskPrice,
		Bid:          resp.MaxBidPrice,
		Volume:       resp.Volume24,
		IndexPrice:   resp.IndexPrice,
		Pair:         cp,
		ExchangeName: e.Name,
		AssetType:    asset.Futures,
		LastUpdated:  resp.Timestamp.Time(),
	}
	return nil
}

func (e *Exchange) processFuturesTickers(data []byte) error {
	var tickers []FuturesTickerItem
	if err := json.Unmarshal(data, &tickers); err != nil {
		return err
	}
	priceTickers := make([]ticker.Price, len(tickers))
	for t := range tickers {
		cp, err := currency.NewPairFromString(tickers[t].Symbol)
		if err != nil {
			return err
		}
		priceTickers[t] = ticker.Price{
			Pair:      cp,
			Last:      tickers[t].LastPrice,
			MarkPrice: tickers[t].FairPrice,
			Volume:    tickers[t].Volume24,
		}
	}
	e.Websocket.DataHandler <- priceTickers
	return nil
}
