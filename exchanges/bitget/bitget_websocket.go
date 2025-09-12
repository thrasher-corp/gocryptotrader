package bitget

import (
	"context"
	"encoding/base64"
	"fmt"
	"hash/crc32"
	"net/http"
	"strconv"
	"strings"
	"sync"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	bitgetPublicWSURL  = "wss://ws.bitget.com/v2/ws/public"
	bitgetPrivateWSURL = "wss://ws.bitget.com/v2/ws/private"
)

var subscriptionNames = map[asset.Item]map[string]string{
	asset.Spot: {
		subscription.TickerChannel:    bitgetTicker,
		subscription.CandlesChannel:   bitgetCandleDailyChannel,
		subscription.AllOrdersChannel: bitgetTrade,
		subscription.OrderbookChannel: bitgetBookFullChannel,
		subscription.MyTradesChannel:  bitgetFillChannel,
		subscription.MyOrdersChannel:  bitgetOrdersChannel,
		"myTriggerOrders":             bitgetOrdersAlgoChannel,
		"account":                     bitgetAccount,
	},
	asset.Futures: {
		subscription.TickerChannel:    bitgetTicker,
		subscription.CandlesChannel:   bitgetCandleDailyChannel,
		subscription.AllOrdersChannel: bitgetTrade,
		subscription.OrderbookChannel: bitgetBookFullChannel,
		subscription.MyTradesChannel:  bitgetFillChannel,
		subscription.MyOrdersChannel:  bitgetOrdersChannel,
		"myTriggerOrders":             bitgetOrdersAlgoChannel,
		"account":                     bitgetAccount,
		"positions":                   bitgetPositionsChannel,
		"positionsHistory":            bitgetPositionsHistoryChannel,
	},
	asset.Margin: {
		"indexPrice":                 bitgetIndexPriceChannel,
		subscription.MyOrdersChannel: bitgetOrdersIsolatedChannel,
		"account":                    bitgetAccountIsolatedChannel,
	},
	asset.CrossMargin: {
		"indexPrice":                 bitgetIndexPriceChannel,
		subscription.MyOrdersChannel: bitgetOrdersCrossedChannel,
		"account":                    bitgetAccountCrossedChannel,
	},
}

var defaultSubscriptions = subscription.List{
	{Enabled: false, Channel: subscription.TickerChannel, Asset: asset.Spot},
	{Enabled: false, Channel: subscription.TickerChannel, Asset: asset.Futures},
	{Enabled: false, Channel: subscription.CandlesChannel, Asset: asset.Spot},
	{Enabled: false, Channel: subscription.CandlesChannel, Asset: asset.Futures},
	{Enabled: false, Channel: subscription.AllOrdersChannel, Asset: asset.Spot},
	{Enabled: false, Channel: subscription.AllOrdersChannel, Asset: asset.Futures},
	{Enabled: true, Channel: subscription.OrderbookChannel, Asset: asset.Spot},
	{Enabled: true, Channel: subscription.OrderbookChannel, Asset: asset.Futures},
	{Enabled: false, Channel: subscription.MyTradesChannel, Authenticated: true, Asset: asset.Spot},
	{Enabled: false, Channel: subscription.MyTradesChannel, Authenticated: true, Asset: asset.Futures},
	{Enabled: false, Channel: subscription.MyOrdersChannel, Authenticated: true, Asset: asset.Spot},
	{Enabled: false, Channel: subscription.MyOrdersChannel, Authenticated: true, Asset: asset.Futures},
	{Enabled: false, Channel: subscription.MyOrdersChannel, Authenticated: true, Asset: asset.Margin},
	{Enabled: false, Channel: subscription.MyOrdersChannel, Authenticated: true, Asset: asset.CrossMargin},
	{Enabled: false, Channel: "myTriggerOrders", Authenticated: true, Asset: asset.Spot},
	{Enabled: false, Channel: "myTriggerOrders", Authenticated: true, Asset: asset.Futures},
	{Enabled: false, Channel: "account", Authenticated: true, Asset: asset.Spot},
	{Enabled: false, Channel: "account", Authenticated: true, Asset: asset.Futures},
	{Enabled: false, Channel: "account", Authenticated: true, Asset: asset.Margin},
	{Enabled: false, Channel: "account", Authenticated: true, Asset: asset.CrossMargin},
	{Enabled: false, Channel: "positions", Authenticated: true, Asset: asset.Futures},
	{Enabled: false, Channel: "positionsHistory", Authenticated: true, Asset: asset.Futures},
	{Enabled: false, Channel: "indexPrice", Asset: asset.Margin},
}

// WsConnect connects to a websocket feed
func (e *Exchange) WsConnect() error {
	if !e.Websocket.IsEnabled() || !e.IsEnabled() {
		return websocket.ErrWebsocketNotEnabled
	}
	var dialer gws.Dialer
	err := e.Websocket.Conn.Dial(context.TODO(), &dialer, http.Header{})
	if err != nil {
		return err
	}
	if e.Verbose {
		log.Debugf(log.ExchangeSys, "%s connected to Websocket.\n", e.Name)
	}
	e.Websocket.Wg.Add(1)
	go e.wsReadData(e.Websocket.Conn)
	e.Websocket.Conn.SetupPingHandler(request.Unset, websocket.PingHandler{
		Websocket:   true,
		Message:     []byte(`ping`),
		MessageType: gws.TextMessage,
		Delay:       time.Second * 25,
	})
	if e.IsWebsocketAuthenticationSupported() {
		var authDialer gws.Dialer
		err = e.WsAuth(context.TODO(), &authDialer)
		if err != nil {
			log.Errorf(log.ExchangeSys, "Error connecting auth socket: %s\n", err.Error())
			e.Websocket.SetCanUseAuthenticatedEndpoints(false)
		}
	}
	return nil
}

// WsAuth sends an authentication message to the websocket
func (e *Exchange) WsAuth(ctx context.Context, dialer *gws.Dialer) error {
	if !e.Websocket.CanUseAuthenticatedEndpoints() {
		return fmt.Errorf("%v %w", e.Name, errAuthenticatedWebsocketDisabled)
	}
	err := e.Websocket.AuthConn.Dial(context.TODO(), dialer, http.Header{})
	if err != nil {
		return err
	}
	e.Websocket.Wg.Add(1)
	go e.wsReadData(e.Websocket.AuthConn)
	e.Websocket.AuthConn.SetupPingHandler(request.Unset, websocket.PingHandler{
		Websocket:   true,
		Message:     []byte(`ping`),
		MessageType: gws.TextMessage,
		Delay:       time.Second * 25,
	})
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	message := timestamp + "GET" + "/user/verify"
	hmac, err := crypto.GetHMAC(crypto.HashSHA256, []byte(message), []byte(creds.Secret))
	if err != nil {
		return err
	}
	base64Sign := base64.StdEncoding.EncodeToString(hmac)
	payload := WsLogin{
		Operation: "login",
		Arguments: []WsLoginArgument{
			{
				APIKey:     creds.Key,
				Signature:  base64Sign,
				Timestamp:  timestamp,
				Passphrase: creds.ClientID,
			},
		},
	}
	err = e.Websocket.AuthConn.SendJSONMessage(ctx, request.Unset, payload)
	if err != nil {
		return err
	}
	// Without this, the exchange will sometimes process a subscription message before it finishes processing the login message. Might be able to reduce the duration
	time.Sleep(time.Second / 2)
	return nil
}

// wsReadData receives and passes on websocket messages for processing
func (e *Exchange) wsReadData(ws websocket.Connection) {
	defer e.Websocket.Wg.Done()
	for {
		resp := ws.ReadMessage()
		if resp.Raw == nil {
			return
		}
		err := e.wsHandleData(resp.Raw)
		if err != nil {
			e.Websocket.DataHandler <- err
		}
	}
}

// wsHandleData handles data from the websocket connection
func (e *Exchange) wsHandleData(respRaw []byte) error {
	var wsResponse WsResponse
	if respRaw != nil && string(respRaw[:4]) == "pong" {
		if e.Verbose {
			log.Debugf(log.ExchangeSys, "%v - Websocket pong received\n", e.Name)
		}
		return nil
	}
	err := json.Unmarshal(respRaw, &wsResponse)
	if err != nil {
		return err
	}
	// Under the assumption that the exchange only ever sends one of these. If both can be sent, this will need to be made more complicated
	toCheck := wsResponse.Event + wsResponse.Action
	switch toCheck {
	case "subscribe":
		if e.Verbose {
			log.Debugf(log.ExchangeSys, "%v - Websocket %v succeeded for %v\n", e.Name, wsResponse.Event, wsResponse.Arg)
		}
	case "error":
		return fmt.Errorf(errWebsocketGeneric, e.Name, wsResponse.Code, wsResponse.Message)
	case "login":
		if wsResponse.Code != 0 {
			return fmt.Errorf(errWebsocketLoginFailed, e.Name, wsResponse.Message)
		}
		if e.Verbose {
			log.Debugf(log.ExchangeSys, "%v - Websocket login succeeded\n", e.Name)
		}
	case "snapshot":
		switch wsResponse.Arg.Channel {
		case bitgetTicker:
			err = e.tickerDataHandler(&wsResponse, respRaw)
		case bitgetCandleDailyChannel:
			err = e.candleDataHandler(&wsResponse)
		case bitgetTrade:
			err = e.tradeDataHandler(&wsResponse)
		case bitgetBookFullChannel:
			err = e.orderbookDataHandler(&wsResponse)
		case bitgetAccount:
			err = e.accountSnapshotDataHandler(&wsResponse, respRaw)
		case bitgetFillChannel:
			err = e.fillDataHandler(&wsResponse, respRaw)
		case bitgetOrdersChannel:
			err = e.genOrderDataHandler(&wsResponse, respRaw)
		case bitgetOrdersAlgoChannel:
			err = e.triggerOrderDataHandler(&wsResponse, respRaw)
		case bitgetPositionsChannel:
			err = e.positionsDataHandler(&wsResponse)
		case bitgetPositionsHistoryChannel:
			err = e.positionsHistoryDataHandler(&wsResponse)
		case bitgetIndexPriceChannel:
			err = e.indexPriceDataHandler(&wsResponse)
		case bitgetAccountCrossedChannel:
			err = e.crossAccountDataHandler(&wsResponse)
		case bitgetOrdersCrossedChannel, bitgetOrdersIsolatedChannel:
			err = e.marginOrderDataHandler(&wsResponse)
		case bitgetAccountIsolatedChannel:
			err = e.isolatedAccountDataHandler(&wsResponse)
		default:
			e.Websocket.DataHandler <- websocket.UnhandledMessageWarning{Message: e.Name + websocket.UnhandledMessage + string(respRaw)}
		}
	case "update":
		switch wsResponse.Arg.Channel {
		case bitgetCandleDailyChannel:
			err = e.candleDataHandler(&wsResponse)
		case bitgetTrade:
			err = e.tradeDataHandler(&wsResponse)
		case bitgetBookFullChannel:
			err = e.orderbookDataHandler(&wsResponse)
		case bitgetAccount:
			err = e.accountUpdateDataHandler(&wsResponse, respRaw)
		default:
			e.Websocket.DataHandler <- websocket.UnhandledMessageWarning{Message: e.Name + websocket.UnhandledMessage + string(respRaw)}
		}
	default:
		e.Websocket.DataHandler <- websocket.UnhandledMessageWarning{Message: e.Name + websocket.UnhandledMessage + string(respRaw)}
	}
	return err
}

// TickerDataHandler handles incoming ticker data for websockets
func (e *Exchange) tickerDataHandler(wsResponse *WsResponse, respRaw []byte) error {
	respAsset := itemDecoder(wsResponse.Arg.InstrumentType)
	switch respAsset {
	case asset.Spot:
		var ticks []WsTickerSnapshotSpot
		err := json.Unmarshal(wsResponse.Data, &ticks)
		if err != nil {
			return err
		}
		for i := range ticks {
			pair, err := pairFromStringHelper(ticks[i].InstrumentID)
			if err != nil {
				return err
			}
			e.Websocket.DataHandler <- &ticker.Price{
				Last:         ticks[i].LastPrice.Float64(),
				High:         ticks[i].High24H.Float64(),
				Low:          ticks[i].Low24H.Float64(),
				Bid:          ticks[i].BidPrice.Float64(),
				Ask:          ticks[i].AskPrice.Float64(),
				Volume:       ticks[i].BaseVolume.Float64(),
				QuoteVolume:  ticks[i].QuoteVolume.Float64(),
				Open:         ticks[i].Open24H.Float64(),
				Pair:         pair,
				ExchangeName: e.Name,
				AssetType:    itemDecoder(wsResponse.Arg.InstrumentType),
				LastUpdated:  ticks[i].Timestamp.Time(),
			}
		}
	case asset.Futures:
		var ticks []WsTickerSnapshotFutures
		err := json.Unmarshal(wsResponse.Data, &ticks)
		if err != nil {
			return err
		}
		for i := range ticks {
			pair, err := pairFromStringHelper(ticks[i].InstrumentID)
			if err != nil {
				return err
			}
			e.Websocket.DataHandler <- &ticker.Price{
				Last:         ticks[i].LastPrice.Float64(),
				High:         ticks[i].High24H.Float64(),
				Low:          ticks[i].Low24H.Float64(),
				Bid:          ticks[i].BidPrice.Float64(),
				Ask:          ticks[i].AskPrice.Float64(),
				Volume:       ticks[i].BaseVolume.Float64(),
				QuoteVolume:  ticks[i].QuoteVolume.Float64(),
				Open:         ticks[i].Open24H.Float64(),
				MarkPrice:    ticks[i].MarkPrice.Float64(),
				IndexPrice:   ticks[i].IndexPrice.Float64(),
				Pair:         pair,
				ExchangeName: e.Name,
				AssetType:    itemDecoder(wsResponse.Arg.InstrumentType),
				LastUpdated:  ticks[i].Timestamp.Time(),
			}
		}
	default:
		e.Websocket.DataHandler <- websocket.UnhandledMessageWarning{Message: e.Name + websocket.UnhandledMessage + string(respRaw)}
	}
	return nil
}

// CandleDataHandler handles candle data, as functionality is shared between updates and snapshots
func (e *Exchange) candleDataHandler(wsResponse *WsResponse) error {
	var candles [][8]string
	err := json.Unmarshal(wsResponse.Data, &candles)
	if err != nil {
		return err
	}
	pair, err := pairFromStringHelper(wsResponse.Arg.InstrumentID)
	if err != nil {
		return err
	}
	resp := make([]websocket.KlineData, len(candles))
	for i := range candles {
		ts, err := strconv.ParseInt(candles[i][0], 10, 64)
		if err != nil {
			return err
		}
		open, err := strconv.ParseFloat(candles[i][1], 64)
		if err != nil {
			return err
		}
		closePrice, err := strconv.ParseFloat(candles[i][4], 64)
		if err != nil {
			return err
		}
		high, err := strconv.ParseFloat(candles[i][2], 64)
		if err != nil {
			return err
		}
		low, err := strconv.ParseFloat(candles[i][3], 64)
		if err != nil {
			return err
		}
		volume, err := strconv.ParseFloat(candles[i][5], 64)
		if err != nil {
			return err
		}
		resp[i] = websocket.KlineData{
			Timestamp:  wsResponse.Timestamp.Time(),
			Pair:       pair,
			AssetType:  itemDecoder(wsResponse.Arg.InstrumentType),
			Exchange:   e.Name,
			StartTime:  time.UnixMilli(ts),
			CloseTime:  time.UnixMilli(ts).Add(time.Hour * 24),
			Interval:   "1d",
			OpenPrice:  open,
			ClosePrice: closePrice,
			HighPrice:  high,
			LowPrice:   low,
			Volume:     volume,
		}
	}
	e.Websocket.DataHandler <- resp
	return nil
}

// TradeDataHandler handles trade data, as functionality is shared between updates and snapshots
func (e *Exchange) tradeDataHandler(wsResponse *WsResponse) error {
	var trades []WsTradeResponse
	pair, err := pairFromStringHelper(wsResponse.Arg.InstrumentID)
	if err != nil {
		return err
	}
	err = json.Unmarshal(wsResponse.Data, &trades)
	if err != nil {
		return err
	}
	resp := make([]trade.Data, len(trades))
	for i := range trades {
		resp[i] = trade.Data{
			Timestamp:    trades[i].Timestamp.Time(),
			CurrencyPair: pair,
			AssetType:    itemDecoder(wsResponse.Arg.InstrumentType),
			Exchange:     e.Name,
			Price:        trades[i].Price.Float64(),
			Amount:       trades[i].Size.Float64(),
			Side:         sideDecoder(trades[i].Side),
			TID:          strconv.FormatInt(trades[i].TradeID, 10),
		}
	}
	e.Websocket.DataHandler <- resp
	return nil
}

// OrderbookDataHandler handles orderbook data, as functionality is shared between updates and snapshots
func (e *Exchange) orderbookDataHandler(wsResponse *WsResponse) error {
	var ob []WsOrderBookResponse
	pair, err := pairFromStringHelper(wsResponse.Arg.InstrumentID)
	if err != nil {
		return err
	}
	err = json.Unmarshal(wsResponse.Data, &ob)
	if err != nil {
		return err
	}
	if len(ob) == 0 {
		return errReturnEmpty
	}
	bids, err := levelConstructor(ob[0].Bids)
	if err != nil {
		return err
	}
	asks, err := levelConstructor(ob[0].Asks)
	if err != nil {
		return err
	}
	if wsResponse.Action[0] == 's' {
		orderbook := orderbook.Book{
			Pair:                   pair,
			Asset:                  itemDecoder(wsResponse.Arg.InstrumentType),
			Bids:                   bids,
			Asks:                   asks,
			LastUpdated:            wsResponse.Timestamp.Time(),
			Exchange:               e.Name,
			ValidateOrderbook:      e.ValidateOrderbook,
			ChecksumStringRequired: true,
		}
		err = e.Websocket.Orderbook.LoadSnapshot(&orderbook)
		if err != nil {
			return err
		}
	} else {
		update := orderbook.Update{
			Bids:             bids,
			Asks:             asks,
			Pair:             pair,
			UpdateTime:       wsResponse.Timestamp.Time(),
			Asset:            itemDecoder(wsResponse.Arg.InstrumentType),
			GenerateChecksum: e.CalculateUpdateOrderbookChecksum,
			ExpectedChecksum: uint32(ob[0].Checksum), //nolint:gosec // The exchange sends it as ints expecting overflows to be handled as Go does by default
		}
		// Sometimes the exchange returns updates with no new asks or bids, just a checksum and timestamp
		if len(update.Bids) != 0 || len(update.Asks) != 0 {
			err = e.Websocket.Orderbook.Update(&update)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// AccountSnapshotDataHandler handles account snapshot data
func (e *Exchange) accountSnapshotDataHandler(wsResponse *WsResponse, respRaw []byte) error {
	var hold account.Holdings
	hold.Exchange = e.Name
	var sub account.SubAccount
	hold.Accounts = append(hold.Accounts, sub)
	respAsset := itemDecoder(wsResponse.Arg.InstrumentType)
	sub.AssetType = respAsset
	switch respAsset {
	case asset.Spot:
		var acc []WsAccountSpotResponse
		err := json.Unmarshal(wsResponse.Data, &acc)
		if err != nil {
			return err
		}
		sub.Currencies = make([]account.Balance, len(acc))
		for i := range acc {
			sub.Currencies[i] = account.Balance{
				Currency: acc[i].Coin,
				Hold:     acc[i].Frozen.Float64() + acc[i].Locked.Float64(),
				Free:     acc[i].Available.Float64(),
				Total:    sub.Currencies[i].Hold + sub.Currencies[i].Free,
			}
		}
	case asset.Futures:
		var acc []WsAccountFuturesResponse
		err := json.Unmarshal(wsResponse.Data, &acc)
		if err != nil {
			return err
		}
		sub.Currencies = make([]account.Balance, len(acc))
		for i := range acc {
			sub.Currencies[i] = account.Balance{
				Currency: acc[i].MarginCoin,
				Hold:     acc[i].Frozen.Float64(),
				Free:     acc[i].Available.Float64(),
				Total:    acc[i].Available.Float64() + acc[i].Frozen.Float64(),
			}
		}
	default:
		e.Websocket.DataHandler <- websocket.UnhandledMessageWarning{Message: e.Name + websocket.UnhandledMessage + string(respRaw)}
	}
	// Plan to add handling of account.Holdings on websocketDataHandler side in a later PR
	e.Websocket.DataHandler <- hold
	return nil
}

func (e *Exchange) fillDataHandler(wsResponse *WsResponse, respRaw []byte) error {
	respAsset := itemDecoder(wsResponse.Arg.InstrumentType)
	switch respAsset {
	case asset.Spot:
		var fil []WsFillSpotResponse
		err := json.Unmarshal(wsResponse.Data, &fil)
		if err != nil {
			return err
		}
		resp := make([]fill.Data, len(fil))
		for i := range fil {
			pair, err := pairFromStringHelper(fil[i].Symbol)
			if err != nil {
				return err
			}
			resp[i] = fill.Data{
				ID:           strconv.FormatInt(fil[i].TradeID, 10),
				Timestamp:    fil[i].CreationTime.Time(),
				Exchange:     e.Name,
				AssetType:    asset.Spot,
				CurrencyPair: pair,
				Side:         sideDecoder(fil[i].Side),
				OrderID:      strconv.FormatInt(fil[i].OrderID, 10),
				TradeID:      strconv.FormatInt(fil[i].TradeID, 10),
				Price:        fil[i].PriceAverage.Float64(),
				Amount:       fil[i].Size.Float64(),
			}
		}
		e.Websocket.DataHandler <- resp
	case asset.Futures:
		var fil []WsFillFuturesResponse
		err := json.Unmarshal(wsResponse.Data, &fil)
		if err != nil {
			return err
		}
		resp := make([]fill.Data, len(fil))
		for i := range fil {
			pair, err := pairFromStringHelper(fil[i].Symbol)
			if err != nil {
				return err
			}
			resp[i] = fill.Data{
				Exchange:     e.Name,
				CurrencyPair: pair,
				OrderID:      strconv.FormatInt(fil[i].OrderID, 10),
				TradeID:      strconv.FormatInt(fil[i].TradeID, 10),
				Side:         sideDecoder(fil[i].Side),
				Price:        fil[i].Price.Float64(),
				Amount:       fil[i].BaseVolume.Float64(),
				Timestamp:    fil[i].CreationTime.Time(),
			}
		}
		e.Websocket.DataHandler <- resp
	default:
		e.Websocket.DataHandler <- websocket.UnhandledMessageWarning{Message: e.Name + websocket.UnhandledMessage + string(respRaw)}
	}
	return nil
}

// genOrderDataHandler handles generic order data
func (e *Exchange) genOrderDataHandler(wsResponse *WsResponse, respRaw []byte) error {
	respAsset := itemDecoder(wsResponse.Arg.InstrumentType)
	switch respAsset {
	case asset.Spot:
		var orders []WsOrderSpotResponse
		err := json.Unmarshal(wsResponse.Data, &orders)
		if err != nil {
			return err
		}
		resp := make([]order.Detail, len(orders))
		for i := range orders {
			pair, err := pairFromStringHelper(orders[i].InstrumentID)
			if err != nil {
				return err
			}
			var baseAmount, quoteAmount float64
			side := sideDecoder(orders[i].Side)
			if side == order.Buy {
				quoteAmount = orders[i].Size.Float64()
			}
			if side == order.Sell {
				baseAmount = orders[i].Size.Float64()
			}
			orderType := typeDecoder(orders[i].OrderType)
			if orderType == order.Limit {
				baseAmount = orders[i].NewSize.Float64()
			}
			resp[i] = order.Detail{
				Exchange:             e.Name,
				AssetType:            asset.Spot,
				Pair:                 pair,
				OrderID:              strconv.FormatInt(orders[i].OrderID, 10),
				ClientOrderID:        orders[i].ClientOrderID,
				Price:                orders[i].PriceAverage.Float64(),
				Amount:               baseAmount,
				QuoteAmount:          quoteAmount,
				Type:                 orderType,
				TimeInForce:          strategyDecoder(orders[i].Force),
				Side:                 side,
				AverageExecutedPrice: orders[i].PriceAverage.Float64(),
				Status:               statusDecoder(orders[i].Status),
				Date:                 orders[i].CreationTime.Time(),
				LastUpdated:          orders[i].UpdateTime.Time(),
			}
			for x := range orders[i].FeeDetail {
				resp[i].Fee += orders[i].FeeDetail[x].TotalFee.Float64()
				resp[i].FeeAsset = orders[i].FeeDetail[x].FeeCoin
			}
		}
		e.Websocket.DataHandler <- resp
	case asset.Futures:
		var orders []WsOrderFuturesResponse
		err := json.Unmarshal(wsResponse.Data, &orders)
		if err != nil {
			return err
		}
		resp := make([]order.Detail, len(orders))
		for i := range orders {
			pair, err := pairFromStringHelper(orders[i].InstrumentID)
			if err != nil {
				return err
			}
			var baseAmount, quoteAmount float64
			side := sideDecoder(orders[i].Side)
			if side == order.Buy {
				quoteAmount = orders[i].Size.Float64()
			}
			if side == order.Sell {
				baseAmount = orders[i].Size.Float64()
			}
			orderType := typeDecoder(orders[i].OrderType)
			if orderType == order.Limit {
				baseAmount = orders[i].BaseVolume.Float64()
			}
			resp[i] = order.Detail{
				Exchange:             e.Name,
				AssetType:            asset.Futures,
				Pair:                 pair,
				Amount:               baseAmount,
				QuoteAmount:          quoteAmount,
				Type:                 orderType,
				TimeInForce:          strategyDecoder(orders[i].Force),
				Side:                 side,
				ExecutedAmount:       orders[i].FilledQuantity.Float64(),
				Date:                 orders[i].CreationTime.Time(),
				ClientOrderID:        orders[i].ClientOrderID,
				Leverage:             orders[i].Leverage.Float64(),
				MarginType:           marginDecoder(orders[i].MarginMode),
				OrderID:              strconv.FormatInt(orders[i].OrderID, 10),
				Price:                orders[i].Price.Float64(),
				AverageExecutedPrice: orders[i].PriceAverage.Float64(),
				ReduceOnly:           bool(orders[i].ReduceOnly),
				Status:               statusDecoder(orders[i].Status),
				LimitPriceLower:      orders[i].PresetStopSurplusPrice.Float64(),
				LimitPriceUpper:      orders[i].PresetStopLossPrice.Float64(),
				LastUpdated:          orders[i].UpdateTime.Time(),
			}
			for x := range orders[i].FeeDetail {
				resp[i].Fee += orders[i].FeeDetail[x].Fee.Float64()
				resp[i].FeeAsset = orders[i].FeeDetail[x].FeeCoin
			}
		}
		e.Websocket.DataHandler <- resp
	default:
		e.Websocket.DataHandler <- websocket.UnhandledMessageWarning{Message: e.Name + websocket.UnhandledMessage + string(respRaw)}
	}
	return nil
}

// TriggerOrderDataHandler handles trigger order data
func (e *Exchange) triggerOrderDataHandler(wsResponse *WsResponse, respRaw []byte) error {
	respAsset := itemDecoder(wsResponse.Arg.InstrumentType)
	switch respAsset {
	case asset.Spot:
		var orders []WsTriggerOrderSpotResponse
		err := json.Unmarshal(wsResponse.Data, &orders)
		if err != nil {
			return err
		}
		resp := make([]order.Detail, len(orders))
		for i := range orders {
			pair, err := pairFromStringHelper(orders[i].InstrumentID)
			if err != nil {
				return err
			}
			resp[i] = order.Detail{
				Exchange:      e.Name,
				AssetType:     asset.Spot,
				Pair:          pair,
				OrderID:       strconv.FormatInt(orders[i].OrderID, 10),
				ClientOrderID: orders[i].ClientOrderID,
				TriggerPrice:  orders[i].TriggerPrice.Float64(),
				Price:         orders[i].Price.Float64(),
				Amount:        orders[i].Size.Float64(),
				Type:          typeDecoder(orders[i].OrderType),
				Side:          sideDecoder(orders[i].Side),
				Status:        statusDecoder(orders[i].Status),
				Date:          orders[i].CreationTime.Time(),
				LastUpdated:   orders[i].UpdateTime.Time(),
			}
		}
		e.Websocket.DataHandler <- resp
	case asset.Futures:
		var orders []WsTriggerOrderFuturesResponse
		err := json.Unmarshal(wsResponse.Data, &orders)
		if err != nil {
			return err
		}
		resp := make([]order.Detail, len(orders))
		for i := range orders {
			pair, err := pairFromStringHelper(orders[i].InstrumentID)
			if err != nil {
				return err
			}
			resp[i] = order.Detail{
				Exchange:             e.Name,
				AssetType:            asset.Futures,
				Pair:                 pair,
				OrderID:              strconv.FormatInt(orders[i].OrderID, 10),
				ClientOrderID:        orders[i].ClientOrderID,
				TriggerPrice:         orders[i].TriggerPrice.Float64(),
				Price:                orders[i].Price.Float64(),
				AverageExecutedPrice: orders[i].ExecutePrice.Float64(),
				Amount:               orders[i].Size.Float64(),
				Type:                 typeDecoder(orders[i].OrderType),
				Side:                 sideDecoder(orders[i].Side),
				Status:               statusDecoder(orders[i].Status),
				Date:                 orders[i].CreationTime.Time(),
				LastUpdated:          orders[i].UpdateTime.Time(),
			}
		}
		e.Websocket.DataHandler <- resp
	default:
		e.Websocket.DataHandler <- websocket.UnhandledMessageWarning{Message: e.Name + websocket.UnhandledMessage + string(respRaw)}
	}
	return nil
}

// PositionsDataHandler handles data on futures positions
func (e *Exchange) positionsDataHandler(wsResponse *WsResponse) error {
	var positions []WsPositionResponse
	err := json.Unmarshal(wsResponse.Data, &positions)
	if err != nil {
		return err
	}
	resp := make([]order.Detail, len(positions))
	for i := range positions {
		pair, err := pairFromStringHelper(positions[i].InstrumentID)
		if err != nil {
			return err
		}
		resp[i] = order.Detail{
			Exchange:             e.Name,
			AssetType:            asset.Futures,
			Pair:                 pair,
			OrderID:              strconv.FormatInt(positions[i].PositionID, 10),
			MarginType:           marginDecoder(positions[i].MarginMode),
			Side:                 sideDecoder(positions[i].HoldSide),
			Amount:               positions[i].Total.Float64(),
			AverageExecutedPrice: positions[i].OpenPriceAverage.Float64(),
			Leverage:             positions[i].Leverage.Float64(),
			Date:                 positions[i].CreationTime.Time(),
			Fee:                  positions[i].TotalFee.Float64(),
			LastUpdated:          positions[i].UpdateTime.Time(),
		}
	}
	e.Websocket.DataHandler <- resp
	return nil
}

// PositionsHistoryDataHandler handles data on futures positions history
func (e *Exchange) positionsHistoryDataHandler(wsResponse *WsResponse) error {
	var positions []WsPositionHistoryResponse
	err := json.Unmarshal(wsResponse.Data, &positions)
	if err != nil {
		return err
	}
	resp := make([]futures.PositionHistory, len(positions))
	for i := range positions {
		pair, err := pairFromStringHelper(positions[i].InstrumentID)
		if err != nil {
			return err
		}
		resp[i] = futures.PositionHistory{
			Exchange:          e.Name,
			PositionID:        strconv.FormatInt(positions[i].PositionID, 10),
			Pair:              pair,
			MarginCoin:        positions[i].MarginCoin,
			MarginType:        marginDecoder(positions[i].MarginMode),
			Side:              sideDecoder(positions[i].HoldSide),
			PositionMode:      positionModeDecoder(positions[i].PositionMode),
			OpenAveragePrice:  positions[i].OpenPriceAverage.Float64(),
			CloseAveragePrice: positions[i].ClosePriceAverage.Float64(),
			OpenSize:          positions[i].OpenSize.Float64(),
			CloseSize:         positions[i].CloseSize.Float64(),
			RealisedPnl:       positions[i].AchievedProfits.Float64(),
			SettlementFee:     positions[i].SettleFee.Float64(),
			OpenFee:           positions[i].OpenFee.Float64(),
			CloseFee:          positions[i].CloseFee.Float64(),
			StartDate:         positions[i].CreationTime.Time(),
			LastUpdated:       positions[i].UpdateTime.Time(),
		}
	}
	// Implement a better handler for this once work on account.Holdings begins
	e.Websocket.DataHandler <- resp
	return nil
}

// IndexPriceDataHandler handles index price data
func (e *Exchange) indexPriceDataHandler(wsResponse *WsResponse) error {
	var indexPrice []WsIndexPriceResponse
	err := json.Unmarshal(wsResponse.Data, &indexPrice)
	if err != nil {
		return err
	}
	resp := make([]ticker.Price, len(indexPrice))
	var cur int
	for i := range indexPrice {
		as := itemDecoder(wsResponse.Arg.InstrumentType)
		pair, enabled, err := e.MatchSymbolCheckEnabled(indexPrice[i].Symbol, as, false)
		// The exchange sometimes returns unavailable pairs such as "USDT/USDT" which should be ignored
		if !enabled || err != nil {
			continue
		}
		resp[cur] = ticker.Price{
			ExchangeName: e.Name,
			AssetType:    as,
			Pair:         pair,
			Last:         indexPrice[i].IndexPrice.Float64(),
			LastUpdated:  indexPrice[i].Timestamp.Time(),
		}
	}
	resp = resp[:cur]
	e.Websocket.DataHandler <- resp
	return nil
}

// CrossAccountDataHandler handles cross margin account data
func (e *Exchange) crossAccountDataHandler(wsResponse *WsResponse) error {
	var acc []WsAccountCrossMarginResponse
	err := json.Unmarshal(wsResponse.Data, &acc)
	if err != nil {
		return err
	}
	var hold account.Holdings
	hold.Exchange = e.Name
	var sub account.SubAccount
	hold.Accounts = append(hold.Accounts, sub)
	sub.AssetType = asset.CrossMargin
	sub.Currencies = make([]account.Balance, len(acc))
	for i := range acc {
		sub.Currencies[i] = account.Balance{
			Currency:               acc[i].Coin,
			Hold:                   acc[i].Frozen.Float64(),
			Free:                   acc[i].Available.Float64(),
			Borrowed:               acc[i].Borrow.Float64(),
			AvailableWithoutBorrow: acc[i].Available.Float64(),                                                                                                           // Need to check if Bitget actually calculates values this way
			Total:                  acc[i].Available.Float64() + acc[i].Frozen.Float64() + acc[i].Borrow.Float64() + acc[i].Interest.Float64() + acc[i].Coupon.Float64(), // Here too
		}
	}
	e.Websocket.DataHandler <- hold
	return nil
}

// MarginOrderDataHandler handles margin order data
func (e *Exchange) marginOrderDataHandler(wsResponse *WsResponse) error {
	var orders []WsOrderMarginResponse
	err := json.Unmarshal(wsResponse.Data, &orders)
	if err != nil {
		return err
	}
	resp := make([]order.Detail, len(orders))
	pair, err := pairFromStringHelper(wsResponse.Arg.InstrumentID)
	if err != nil {
		return err
	}
	for i := range orders {
		resp[i] = order.Detail{
			Exchange:             e.Name,
			Pair:                 pair,
			OrderID:              strconv.FormatInt(orders[i].OrderID, 10),
			ClientOrderID:        orders[i].ClientOrderID,
			AverageExecutedPrice: orders[i].FillPrice.Float64(),
			Price:                orders[i].Price.Float64(),
			Amount:               orders[i].BaseSize.Float64(),
			QuoteAmount:          orders[i].QuoteSize.Float64(),
			Type:                 typeDecoder(orders[i].OrderType),
			TimeInForce:          strategyDecoder(orders[i].Force),
			Side:                 sideDecoder(orders[i].Side),
			Status:               statusDecoder(orders[i].Status),
			Date:                 orders[i].CreationTime.Time(),
		}
		for x := range orders[i].FeeDetail {
			resp[i].Fee += orders[i].FeeDetail[x].TotalFee.Float64()
			resp[i].FeeAsset = orders[i].FeeDetail[x].FeeCoin
		}
		if wsResponse.Arg.Channel == bitgetOrdersIsolatedChannel {
			resp[i].AssetType = asset.Margin
		} else {
			resp[i].AssetType = asset.CrossMargin
		}
	}
	e.Websocket.DataHandler <- resp
	return nil
}

// IsolatedAccountDataHandler handles isolated margin account data
func (e *Exchange) isolatedAccountDataHandler(wsResponse *WsResponse) error {
	var acc []WsAccountIsolatedMarginResponse
	err := json.Unmarshal(wsResponse.Data, &acc)
	if err != nil {
		return err
	}
	var hold account.Holdings
	hold.Exchange = e.Name
	var sub account.SubAccount
	hold.Accounts = append(hold.Accounts, sub)
	sub.AssetType = asset.Margin
	sub.Currencies = make([]account.Balance, len(acc))
	for i := range acc {
		sub.Currencies[i] = account.Balance{
			Currency:               acc[i].Coin,
			Hold:                   acc[i].Frozen.Float64(),
			Free:                   acc[i].Available.Float64(),
			Borrowed:               acc[i].Borrow.Float64(),
			AvailableWithoutBorrow: acc[i].Available.Float64(),                                                                                                           // Need to check if Bitget actually calculates values this way
			Total:                  acc[i].Available.Float64() + acc[i].Frozen.Float64() + acc[i].Borrow.Float64() + acc[i].Interest.Float64() + acc[i].Coupon.Float64(), // Here too
		}
	}
	e.Websocket.DataHandler <- hold
	return nil
}

// AccountUpdateDataHandler
func (e *Exchange) accountUpdateDataHandler(wsResponse *WsResponse, respRaw []byte) error {
	creds, err := e.GetCredentials(context.TODO())
	if err != nil {
		return err
	}
	var resp []account.Change
	switch itemDecoder(wsResponse.Arg.InstrumentType) {
	case asset.Spot:
		var acc []WsAccountSpotResponse
		err := json.Unmarshal(wsResponse.Data, &acc)
		if err != nil {
			return err
		}
		resp = make([]account.Change, len(acc))
		for i := range acc {
			resp[i] = account.Change{
				AssetType: asset.Spot,
				Balance: &account.Balance{
					Currency:  acc[i].Coin,
					Hold:      acc[i].Frozen.Float64() + acc[i].Locked.Float64(),
					Free:      acc[i].Available.Float64(),
					Total:     acc[i].Available.Float64() + acc[i].Frozen.Float64() + acc[i].Locked.Float64(),
					UpdatedAt: acc[i].UpdateTime.Time(),
				},
			}
		}
		e.Websocket.DataHandler <- resp
	case asset.Futures:
		var acc []WsAccountFuturesResponse
		err := json.Unmarshal(wsResponse.Data, &acc)
		if err != nil {
			return err
		}
		resp = make([]account.Change, len(acc))
		for i := range acc {
			resp[i] = account.Change{
				AssetType: asset.Futures,
				Balance: &account.Balance{
					Currency:  acc[i].MarginCoin,
					Hold:      acc[i].Frozen.Float64(),
					Free:      acc[i].Available.Float64(),
					Total:     acc[i].Available.Float64() + acc[i].Frozen.Float64(),
					UpdatedAt: time.Now(),
				},
			}
		}
		e.Websocket.DataHandler <- resp
	default:
		e.Websocket.DataHandler <- websocket.UnhandledMessageWarning{Message: e.Name + websocket.UnhandledMessage + string(respRaw)}
	}
	return account.ProcessChange(e.Name, resp, creds)
}

// LevelConstructor turns the exchange's orderbook data into a standardised format for the engine
func levelConstructor(data [][2]string) ([]orderbook.Level, error) {
	resp := make([]orderbook.Level, len(data))
	var err error
	for i := range data {
		resp[i] = orderbook.Level{
			StrPrice:  data[i][0],
			StrAmount: data[i][1],
		}
		resp[i].Price, err = strconv.ParseFloat(data[i][0], 64)
		if err != nil {
			return nil, err
		}
		resp[i].Amount, err = strconv.ParseFloat(data[i][1], 64)
		if err != nil {
			return nil, err
		}
	}
	return resp, nil
}

// CalculateUpdateOrderbookChecksum calculates the checksum of the orderbook data
func (e *Exchange) CalculateUpdateOrderbookChecksum(orderbookData *orderbook.Book) uint32 {
	var builder strings.Builder
	for i := range 25 {
		if len(orderbookData.Bids) > i {
			builder.WriteString(orderbookData.Bids[i].StrPrice + ":" + orderbookData.Bids[i].StrAmount + ":")
		}
		if len(orderbookData.Asks) > i {
			builder.WriteString(orderbookData.Asks[i].StrPrice + ":" + orderbookData.Asks[i].StrAmount + ":")
		}
	}
	check := builder.String()
	if check != "" {
		check = check[:len(check)-1]
	}
	return crc32.ChecksumIEEE([]byte(check))
}

// GenerateDefaultSubscriptions generates default subscriptions
func (e *Exchange) generateDefaultSubscriptions() (subscription.List, error) {
	at := e.GetAssetTypes(false)
	assetPairs := make(map[asset.Item]currency.Pairs)
	for i := range at {
		pairs, err := e.GetEnabledPairs(at[i])
		if err != nil {
			return nil, err
		}
		assetPairs[at[i]] = pairs
	}
	subs := make(subscription.List, 0, len(defaultSubscriptions))
	for _, sub := range defaultSubscriptions {
		if sub.Enabled {
			subs = append(subs, sub.Clone()) // Slow, consider this a placeholder until templating support is finished
		}
	}
	subs = subs[:len(subs):len(subs)]
	for i := range subs {
		subs[i].Channel = subscriptionNames[subs[i].Asset][subs[i].Channel]
		switch subs[i].Channel {
		case bitgetAccount, bitgetFillChannel, bitgetPositionsChannel, bitgetPositionsHistoryChannel, bitgetIndexPriceChannel, bitgetAccountCrossedChannel, bitgetAccountIsolatedChannel:
		default:
			subs[i].Pairs = assetPairs[subs[i].Asset]
		}
	}
	return subs, nil
}

// Subscribe sends a websocket message to receive data from the channel
func (e *Exchange) Subscribe(subs subscription.List) error {
	return e.manageSubs("subscribe", subs)
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (e *Exchange) Unsubscribe(subs subscription.List) error {
	return e.manageSubs("unsubscribe", subs)
}

// ReqSplitter splits a request into multiple requests to avoid going over the byte limit
func reqSplitter(req *WsRequest) []WsRequest {
	capacity := (len(req.Arguments) / 47) + 1
	reqs := make([]WsRequest, capacity)
	for i := range capacity {
		reqs[i].Operation = req.Operation
		if i == capacity-1 {
			reqs[i].Arguments = req.Arguments[i*47:]
			break
		}
		reqs[i].Arguments = req.Arguments[i*47 : (i+1)*47]
	}
	return reqs
}

// ReqBuilder builds a request in the manner the exchange expects
func (e *Exchange) reqBuilder(req *WsRequest, sub *subscription.Subscription) {
	for i := range sub.Pairs {
		form, _ := e.GetPairFormat(asset.Spot, true)
		sub.Pairs[i] = sub.Pairs[i].Format(form)
		req.Arguments = append(req.Arguments, WsArgument{
			Channel:        sub.Channel,
			InstrumentType: itemEncoder(sub.Asset, sub.Pairs[i]),
			InstrumentID:   sub.Pairs[i].String(),
		})
	}
	if len(sub.Pairs) == 0 {
		req.Arguments = append(req.Arguments, WsArgument{
			Channel:        sub.Channel,
			InstrumentType: itemEncoder(sub.Asset, currency.Pair{}),
			Coin:           currency.NewCode("default"),
			InstrumentID:   "default",
		})
		if sub.Asset == asset.Futures {
			req.Arguments = append(req.Arguments, WsArgument{
				Channel:        sub.Channel,
				InstrumentType: "USDT-FUTURES",
				Coin:           currency.NewCode("default"),
				InstrumentID:   "default",
			}, WsArgument{
				Channel:        sub.Channel,
				InstrumentType: "USDC-FUTURES",
				Coin:           currency.NewCode("default"),
				InstrumentID:   "default",
			})
		}
	}
}

// manageSubs subscribes or unsubscribes from a list of websocket channels
func (e *Exchange) manageSubs(op string, subs subscription.List) error {
	unauthBase := &WsRequest{
		Operation: op,
	}
	authBase := &WsRequest{
		Operation: op,
	}
	for _, s := range subs {
		if s.Authenticated {
			e.reqBuilder(authBase, s)
		} else {
			e.reqBuilder(unauthBase, s)
		}
	}
	unauthReq := reqSplitter(unauthBase)
	authReq := reqSplitter(authBase)
	wg := sync.WaitGroup{}
	errC := make(chan error, len(unauthReq)+len(authReq))
	for i := range unauthReq {
		if len(unauthReq[i].Arguments) != 0 {
			wg.Add(1)
			go func(req WsRequest) {
				defer wg.Done()
				err := e.Websocket.Conn.SendJSONMessage(context.TODO(), RateSubscription, req)
				if err != nil {
					errC <- err
				}
			}(unauthReq[i])
		}
	}
	for i := range authReq {
		if len(authReq[i].Arguments) != 0 {
			wg.Add(1)
			go func(req WsRequest) {
				defer wg.Done()
				err := e.Websocket.AuthConn.SendJSONMessage(context.TODO(), RateSubscription, req)
				if err != nil {
					errC <- err
				}
			}(authReq[i])
		}
	}
	wg.Wait()
	close(errC)
	var errs error
	for err := range errC {
		errs = common.AppendError(errs, err)
	}
	return errs
}

// GetSubscriptionTemplate returns a subscription channel template
func (e *Exchange) GetSubscriptionTemplate(_ *subscription.Subscription) (*template.Template, error) {
	return template.New("master.tmpl").Funcs(template.FuncMap{"channelName": channelName}).Parse(subTplText)
}

func channelName(s *subscription.Subscription) string {
	if n, ok := subscriptionNames[s.Asset][s.Channel]; ok {
		return n
	}
	panic(fmt.Errorf("%w: %s", subscription.ErrNotSupported, s.Channel))
}

const subTplText = `
{{ range $asset, $pairs := $.AssetPairs }}
	{{- channelName $.S -}}
	{{- $.AssetSeparator }}
{{- end }}
`
