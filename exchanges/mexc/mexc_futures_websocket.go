package mexc

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
	cnlFTickers     = "tickers"
	cnlFTicker      = "ticker"
	cnlFDeal        = "deal"
	cnlFDepthFull   = "depth.full"
	cnlFKline       = "kline"
	cnlFFundingRate = "funding.rate"
	cnlFIndexPrice  = "index.price"
	cnlFFairPrice   = "fair.price"

	// Private channels
	cnlLogin              = "login"
	cnlFPersonalPositions = "personal.position"
	cnlFPersonalAssets    = "personal.asset"
	cnlFPersonalOrder     = "personal.order"
	cnlFPersonalADLLevel  = "personal.adl.level"
	cnlFPersonalRiskLimit = "personal.risk.limit"
	cnlFPositionMode      = "personal.position.mode"
)

var defaultFuturesSubscriptions = []string{
	cnlFTickers,
	cnlFDeal,
	cnlFDepthFull,
	cnlFKline,
}

// WsFuturesConnect established a futures websocket connection
func (me *MEXC) WsFuturesConnect() error {
	if !me.Websocket.IsEnabled() || !me.IsEnabled() {
		return websocket.ErrWebsocketNotEnabled
	}
	dialer := gws.Dialer{
		EnableCompression: true,
		ReadBufferSize:    8192,
		WriteBufferSize:   8192,
	}
	err := me.Websocket.SetWebsocketURL(futuresWsURL, false, true)
	if err != nil {
		return err
	}
	err = me.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}
	me.Websocket.Wg.Add(1)
	go me.wsFuturesReadData(me.Websocket.Conn)
	if me.Websocket.CanUseAuthenticatedEndpoints() {
		err := me.wsAuth()
		if err != nil {
			log.Warnf(log.ExchangeSys, "authentication error: %v", err)
			me.Websocket.SetCanUseAuthenticatedEndpoints(false)
		}
	}
	if me.Verbose {
		log.Debugf(log.ExchangeSys, "Successful connection to %v\n", me.Websocket.GetWebsocketURL())
	}
	return nil
}

// wsAuth authenticates a futures websocket connection
func (me *MEXC) wsAuth() error {
	credentials, err := me.GetCredentials(context.Background())
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
	param.Signature = crypto.HexEncodeToString(hmac)
	data, err := me.Websocket.Conn.SendMessageReturnResponse(context.Background(), request.Auth, "rs.login", &WsSubscriptionPayload{
		Param:  param,
		Method: cnlLogin,
	})
	if err != nil {
		return err
	}
	var result *WsFuturesLoginResponse
	err = json.Unmarshal(data, &result)
	if err != nil {
		return err
	}
	if result.Data != "success" {
		return fmt.Errorf("code: %d, msg: %s", result.Code, result.Message)
	}
	return nil
}

// GenerateDefaultFuturesSubscriptions generates a futures default subscription instances
func (me *MEXC) GenerateDefaultFuturesSubscriptions() (subscription.List, error) {
	channels := defaultFuturesSubscriptions
	if me.Websocket.CanUseAuthenticatedEndpoints() {
		channels = append(channels, cnlFPersonalPositions, cnlFPersonalAssets, cnlFPersonalOrder, cnlFPersonalADLLevel, cnlFPersonalRiskLimit, cnlFPositionMode)
	}
	enabledPairs, err := me.GetEnabledPairs(asset.Futures)
	if err != nil {
		return nil, err
	}
	subscriptionsList := make(subscription.List, len(channels))
	for c := range channels {
		switch channels[c] {
		case cnlFTicker, cnlFDeal, cnlFDepthFull, cnlFFundingRate, cnlFIndexPrice, cnlFFairPrice:
			subscriptionsList[c] = &subscription.Subscription{
				Channel: channels[c],
				Pairs:   enabledPairs,
			}
		case cnlFKline:
			subscriptionsList[c] = &subscription.Subscription{
				Channel:  channels[c],
				Pairs:    enabledPairs,
				Interval: kline.FifteenMin,
			}
		case cnlFTickers, cnlFPersonalPositions, cnlFPersonalAssets, cnlFPersonalOrder,
			cnlFPersonalADLLevel, cnlFPersonalRiskLimit, cnlFPositionMode:
			subscriptionsList[c] = &subscription.Subscription{
				Channel: channels[c],
			}
		}
	}
	return subscriptionsList, nil
}

// SubscribeFutures subscribes to a futures websocket channel
func (me *MEXC) SubscribeFutures(subscriptions subscription.List) error {
	return me.handleSubscriptionFuturesPayload(subscriptions, "sub")
}

// UnsubscribeFutures unsubscribes to a futures websocket channel
func (me *MEXC) UnsubscribeFutures(subscriptions subscription.List) error {
	return me.handleSubscriptionFuturesPayload(subscriptions, "unsub")
}

func (me *MEXC) handleSubscriptionFuturesPayload(subscriptionItems subscription.List, method string) error {
	for x := range subscriptionItems {
		switch subscriptionItems[x].Channel {
		case cnlFDeal, cnlFTicker, cnlFDepthFull, cnlFKline, cnlFFundingRate, cnlFIndexPrice, cnlFFairPrice:
			params := make([]FWebsocketReqParam, len(subscriptionItems[x].Pairs))
			for p := range subscriptionItems[x].Pairs {
				params[p].Symbol = subscriptionItems[x].Pairs[p].String()
				switch subscriptionItems[x].Channel {
				case cnlFDeal:
					params[p].Compress = true
					params[p].Limit = subscriptionItems[x].Levels
				case cnlFKline:
					intervalString, err := ContractIntervalString(subscriptionItems[x].Interval)
					if err != nil {
						return err
					}
					params[p].Interval = intervalString
				}
				err := me.Websocket.Conn.SendJSONMessage(context.Background(), request.UnAuth, &WsSubscriptionPayload{
					Method: method + "." + subscriptionItems[x].Channel,
					Param:  &params[p],
				})
				if err != nil {
					return err
				}
			}
		default:
			err := me.Websocket.Conn.SendJSONMessage(context.Background(), request.UnAuth, &WsSubscriptionPayload{
				Method: method + "." + subscriptionItems[x].Channel,
			})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// wsFuturesReadData sends futures assets related msgs from public and auth websockets to data handler
func (me *MEXC) wsFuturesReadData(ws websocket.Connection) {
	defer me.Websocket.Wg.Done()
	for {
		resp := ws.ReadMessage()
		if len(resp.Raw) == 0 {
			return
		}
		if err := me.WsHandleFuturesData(resp.Raw); err != nil {
			me.Websocket.DataHandler <- err
		}
	}
}

// WsHandleFuturesData processed futures websocket data
func (me *MEXC) WsHandleFuturesData(respRaw []byte) error {
	var resp *WsFuturesData
	err := json.Unmarshal(respRaw, &resp)
	if err != nil {
		return err
	}
	if resp.Channel == "" {
		if resp.Message != "" {
			log.Debugln(log.ExchangeSys, resp.Message)
		}
		return nil
	}
	if resp.Channel == "rs.login" {
		if !me.Websocket.Match.IncomingWithData(resp.Channel, respRaw) {
			me.Websocket.DataHandler <- websocket.UnhandledMessageWarning{
				Message: string(respRaw) + websocket.UnhandledMessage,
			}
		}
	}
	cnlSplits := strings.Split(resp.Channel, ".")
	switch strings.Join(cnlSplits[1:], ".") {
	case cnlFTickers:
		return me.processFuturesTickers(resp.Data)
	case cnlFTicker:
		return me.processFuturesTicker(resp.Data)
	case cnlFDeal:
		return me.processFuturesFillData(resp.Data, resp.Symbol)
	case cnlFDepthFull:
		return me.processOrderbookDepth(resp.Data, resp.Symbol)
	case cnlFKline:
		return me.processFuturesKlineData(resp.Data, resp.Symbol)
	case cnlFFundingRate:
		return me.processFuturesFundingRate(resp.Data)
	case cnlFIndexPrice:
		return me.processIndexPrice(resp.Data)
	case cnlFFairPrice:
		return me.processFairPrice(resp.Data)
	case cnlFPersonalPositions:
		return me.processPersonalPosition(resp.Data)
	case cnlFPersonalAssets:
		return me.processPersonalAsset(resp.Data)
	case cnlFPersonalOrder:
		return me.processPersonalOrder(resp.Data)
	case cnlFPersonalADLLevel:
		return me.processPersonalADLLevel(resp.Data)
	case cnlFPersonalRiskLimit:
		return me.processPersonalRiskLimit(resp.Data)
	case cnlFPositionMode:
		return me.processPersonalPositionMode(resp.Data)
	}
	return nil
}

func (me *MEXC) processPersonalADLLevel(data []byte) error {
	var resp *FuturesADLLevel
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	me.Websocket.DataHandler <- resp
	return nil
}

func (me *MEXC) processPersonalPositionMode(data []byte) error {
	var resp *FuturesPositionMode
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	me.Websocket.DataHandler <- resp
	return nil
}

func (me *MEXC) processPersonalRiskLimit(data []byte) error {
	var resp *FuturesWebsocketRiskLimit
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	me.Websocket.DataHandler <- resp
	return nil
}

func (me *MEXC) processPersonalPosition(data []byte) error {
	var resp *FuturesWsPersonalPosition
	err := json.Unmarshal(data, &resp)
	if err != nil {
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
	me.Websocket.DataHandler <- order.Detail{
		TimeInForce: order.GoodTillCancel,
		Leverage:    resp.Leverage,
		Price:       resp.HoldAvgPrice,
		Amount:      resp.HoldAvgPrice,
		Fee:         resp.HoldFee,
		Exchange:    me.Name,
		OrderID:     strconv.FormatInt(resp.PositionID, 10),
		Side:        oSide,
		Status:      oState,
		AssetType:   asset.Futures,
		Pair:        cp,
		MarginType:  marginType,
	}
	return nil
}

func (me *MEXC) processPersonalAsset(data []byte) error {
	var resp *FuturesPersonalAsset
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	me.Websocket.DataHandler <- account.Change{
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

func (me *MEXC) processPersonalOrder(data []byte) error {
	var resp *WsFuturesPersonalOrder
	err := json.Unmarshal(data, &resp)
	if err != nil {
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
	me.Websocket.DataHandler <- &order.Detail{
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
		Exchange:             me.Name,
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

func (me *MEXC) processFairPrice(data []byte) error {
	var resp *PriceAndSymbol
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	cp, err := currency.NewPairFromString(resp.Symbol)
	if err != nil {
		return err
	}
	me.Websocket.DataHandler <- ticker.Price{
		IndexPrice: resp.Price,
		Pair:       cp,
	}
	return nil
}

func (me *MEXC) processIndexPrice(data []byte) error {
	var resp *PriceAndSymbol
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	cp, err := currency.NewPairFromString(resp.Symbol)
	if err != nil {
		return err
	}
	me.Websocket.DataHandler <- ticker.Price{
		IndexPrice: resp.Price,
		Pair:       cp,
	}
	return nil
}

func (me *MEXC) processFuturesFundingRate(data []byte) error {
	var resp *FuturesWsFundingRate
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	me.Websocket.DataHandler <- resp
	return nil
}

func (me *MEXC) processFuturesKlineData(data []byte, symbol string) error {
	var resp *FuturesWebsocketKline
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	cp, err := currency.NewPairFromString(symbol)
	if err != nil {
		return err
	}
	me.Websocket.DataHandler <- websocket.KlineData{
		Pair:       cp,
		Exchange:   me.Name,
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

func (me *MEXC) processOrderbookDepth(data []byte, symbol string) error {
	var resp *FuturesWsDepth
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	cp, err := currency.NewPairFromString(symbol)
	if err != nil {
		return err
	}
	asks := make(orderbook.Tranches, len(resp.Asks))
	for a := range resp.Asks {
		asks[a] = orderbook.Tranche{
			Price:      resp.Asks[a][0],
			Amount:     resp.Asks[a][1],
			OrderCount: int64(resp.Asks[a][2]),
		}
	}
	bids := make(orderbook.Tranches, len(resp.Bids))
	for b := range resp.Bids {
		bids[b] = orderbook.Tranche{
			Price:      resp.Bids[b][0],
			Amount:     resp.Bids[b][1],
			OrderCount: int64(resp.Bids[b][2]),
		}
	}
	return me.Websocket.Orderbook.LoadSnapshot(&orderbook.Base{
		Bids:        bids,
		Asks:        asks,
		Exchange:    me.Name,
		Pair:        cp,
		Asset:       asset.Futures,
		LastUpdated: time.Now(),
	})
}

func (me *MEXC) processFuturesFillData(data []byte, symbol string) error {
	var resp []FuturesTransactionFills
	err := json.Unmarshal(data, &resp)
	if err != nil {
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
		me.Websocket.DataHandler <- &trade.Data{
			Timestamp: resp[x].TransationTime.Time(),
			Exchange:  me.Name,
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

func (me *MEXC) processFuturesTicker(data []byte) error {
	var resp *FuturesPriceTickerDetail
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	cp, err := currency.NewPairFromString(resp.Symbol)
	if err != nil {
		return err
	}
	me.Websocket.DataHandler <- &ticker.Price{
		Last:         resp.LastPrice,
		High:         resp.High24Price,
		Low:          resp.Lower24Price,
		Ask:          resp.MinAskPrice,
		Bid:          resp.MaxBidPrice,
		Volume:       resp.Volume24,
		IndexPrice:   resp.IndexPrice,
		Pair:         cp,
		ExchangeName: me.Name,
		AssetType:    asset.Futures,
		LastUpdated:  resp.Timestamp.Time(),
	}
	return nil
}

func (me *MEXC) processFuturesTickers(data []byte) error {
	var tickers []FuturesTickerItem
	err := json.Unmarshal(data, &tickers)
	if err != nil {
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
	me.Websocket.DataHandler <- priceTickers
	return nil
}
