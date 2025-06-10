package bitget

import (
	"context"
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
func (bi *Bitget) WsConnect() error {
	if !bi.Websocket.IsEnabled() || !bi.IsEnabled() {
		return websocket.ErrWebsocketNotEnabled
	}
	var dialer gws.Dialer
	err := bi.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}
	if bi.Verbose {
		log.Debugf(log.ExchangeSys, "%s connected to Websocket.\n", bi.Name)
	}
	bi.Websocket.Wg.Add(1)
	go bi.wsReadData(bi.Websocket.Conn)
	bi.Websocket.Conn.SetupPingHandler(request.Unset, websocket.PingHandler{
		Websocket:   true,
		Message:     []byte(`ping`),
		MessageType: gws.TextMessage,
		Delay:       time.Second * 25,
	})
	if bi.IsWebsocketAuthenticationSupported() {
		var authDialer gws.Dialer
		err = bi.WsAuth(context.TODO(), &authDialer)
		if err != nil {
			log.Errorf(log.ExchangeSys, "Error connecting auth socket: %s\n", err.Error())
			bi.Websocket.SetCanUseAuthenticatedEndpoints(false)
		}
	}
	return nil
}

// WsAuth sends an authentication message to the websocket
func (bi *Bitget) WsAuth(ctx context.Context, dialer *gws.Dialer) error {
	if !bi.Websocket.CanUseAuthenticatedEndpoints() {
		return fmt.Errorf("%v %w", bi.Name, errAuthenticatedWebsocketDisabled)
	}
	err := bi.Websocket.AuthConn.Dial(dialer, http.Header{})
	if err != nil {
		return err
	}
	bi.Websocket.Wg.Add(1)
	go bi.wsReadData(bi.Websocket.AuthConn)
	bi.Websocket.AuthConn.SetupPingHandler(request.Unset, websocket.PingHandler{
		Websocket:   true,
		Message:     []byte(`ping`),
		MessageType: gws.TextMessage,
		Delay:       time.Second * 25,
	})
	creds, err := bi.GetCredentials(ctx)
	if err != nil {
		return err
	}
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	message := timestamp + "GET" + "/user/verify"
	hmac, err := crypto.GetHMAC(crypto.HashSHA256, []byte(message), []byte(creds.Secret))
	if err != nil {
		return err
	}
	base64Sign := crypto.Base64Encode(hmac)
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
	err = bi.Websocket.AuthConn.SendJSONMessage(ctx, request.Unset, payload)
	if err != nil {
		return err
	}
	// Without this, the exchange will sometimes process a subscription message before it finishes processing the login message. Might be able to reduce the duration
	time.Sleep(time.Second / 2)
	return nil
}

// wsReadData receives and passes on websocket messages for processing
func (bi *Bitget) wsReadData(ws websocket.Connection) {
	defer bi.Websocket.Wg.Done()
	for {
		resp := ws.ReadMessage()
		if resp.Raw == nil {
			return
		}
		err := bi.wsHandleData(resp.Raw)
		if err != nil {
			bi.Websocket.DataHandler <- err
		}
	}
}

// wsHandleData handles data from the websocket connection
func (bi *Bitget) wsHandleData(respRaw []byte) error {
	var wsResponse WsResponse
	if respRaw != nil && string(respRaw[:4]) == "pong" {
		if bi.Verbose {
			log.Debugf(log.ExchangeSys, "%v - Websocket pong received\n", bi.Name)
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
		if bi.Verbose {
			log.Debugf(log.ExchangeSys, "%v - Websocket %v succeeded for %v\n", bi.Name, wsResponse.Event, wsResponse.Arg)
		}
	case "error":
		return fmt.Errorf(errWebsocketGeneric, bi.Name, wsResponse.Code, wsResponse.Message)
	case "login":
		if wsResponse.Code != 0 {
			return fmt.Errorf(errWebsocketLoginFailed, bi.Name, wsResponse.Message)
		}
		if bi.Verbose {
			log.Debugf(log.ExchangeSys, "%v - Websocket login succeeded\n", bi.Name)
		}
	case "snapshot":
		switch wsResponse.Arg.Channel {
		case bitgetTicker:
			err = bi.tickerDataHandler(&wsResponse, respRaw)
		case bitgetCandleDailyChannel:
			err = bi.candleDataHandler(&wsResponse)
		case bitgetTrade:
			err = bi.tradeDataHandler(&wsResponse)
		case bitgetBookFullChannel:
			err = bi.orderbookDataHandler(&wsResponse)
		case bitgetAccount:
			err = bi.accountSnapshotDataHandler(&wsResponse, respRaw)
		case bitgetFillChannel:
			err = bi.fillDataHandler(&wsResponse, respRaw)
		case bitgetOrdersChannel:
			err = bi.genOrderDataHandler(&wsResponse, respRaw)
		case bitgetOrdersAlgoChannel:
			err = bi.triggerOrderDataHandler(&wsResponse, respRaw)
		case bitgetPositionsChannel:
			err = bi.positionsDataHandler(&wsResponse)
		case bitgetPositionsHistoryChannel:
			err = bi.positionsHistoryDataHandler(&wsResponse)
		case bitgetIndexPriceChannel:
			err = bi.indexPriceDataHandler(&wsResponse)
		case bitgetAccountCrossedChannel:
			err = bi.crossAccountDataHandler(&wsResponse)
		case bitgetOrdersCrossedChannel, bitgetOrdersIsolatedChannel:
			err = bi.marginOrderDataHandler(&wsResponse)
		case bitgetAccountIsolatedChannel:
			err = bi.isolatedAccountDataHandler(&wsResponse)
		default:
			bi.Websocket.DataHandler <- websocket.UnhandledMessageWarning{Message: bi.Name + websocket.UnhandledMessage + string(respRaw)}
		}
	case "update":
		switch wsResponse.Arg.Channel {
		case bitgetCandleDailyChannel:
			err = bi.candleDataHandler(&wsResponse)
		case bitgetTrade:
			err = bi.tradeDataHandler(&wsResponse)
		case bitgetBookFullChannel:
			err = bi.orderbookDataHandler(&wsResponse)
		case bitgetAccount:
			err = bi.accountUpdateDataHandler(&wsResponse, respRaw)
		default:
			bi.Websocket.DataHandler <- websocket.UnhandledMessageWarning{Message: bi.Name + websocket.UnhandledMessage + string(respRaw)}
		}
	default:
		bi.Websocket.DataHandler <- websocket.UnhandledMessageWarning{Message: bi.Name + websocket.UnhandledMessage + string(respRaw)}
	}
	return err
}

// TickerDataHandler handles incoming ticker data for websockets
func (bi *Bitget) tickerDataHandler(wsResponse *WsResponse, respRaw []byte) error {
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
			bi.Websocket.DataHandler <- &ticker.Price{
				Last:         ticks[i].LastPrice,
				High:         ticks[i].High24H,
				Low:          ticks[i].Low24H,
				Bid:          ticks[i].BidPrice,
				Ask:          ticks[i].AskPrice,
				Volume:       ticks[i].BaseVolume,
				QuoteVolume:  ticks[i].QuoteVolume,
				Open:         ticks[i].Open24H,
				Pair:         pair,
				ExchangeName: bi.Name,
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
			bi.Websocket.DataHandler <- &ticker.Price{
				Last:         ticks[i].LastPrice,
				High:         ticks[i].High24H,
				Low:          ticks[i].Low24H,
				Bid:          ticks[i].BidPrice,
				Ask:          ticks[i].AskPrice,
				Volume:       ticks[i].BaseVolume,
				QuoteVolume:  ticks[i].QuoteVolume,
				Open:         ticks[i].Open24H,
				MarkPrice:    ticks[i].MarkPrice,
				IndexPrice:   ticks[i].IndexPrice,
				Pair:         pair,
				ExchangeName: bi.Name,
				AssetType:    itemDecoder(wsResponse.Arg.InstrumentType),
				LastUpdated:  ticks[i].Timestamp.Time(),
			}
		}
	default:
		bi.Websocket.DataHandler <- websocket.UnhandledMessageWarning{Message: bi.Name + websocket.UnhandledMessage + string(respRaw)}
	}
	return nil
}

// CandleDataHandler handles candle data, as functionality is shared between updates and snapshots
func (bi *Bitget) candleDataHandler(wsResponse *WsResponse) error {
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
			Exchange:   bi.Name,
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
	bi.Websocket.DataHandler <- resp
	return nil
}

// TradeDataHandler handles trade data, as functionality is shared between updates and snapshots
func (bi *Bitget) tradeDataHandler(wsResponse *WsResponse) error {
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
			Exchange:     bi.Name,
			Price:        trades[i].Price,
			Amount:       trades[i].Size,
			Side:         sideDecoder(trades[i].Side),
			TID:          strconv.FormatInt(trades[i].TradeID, 10),
		}
	}
	bi.Websocket.DataHandler <- resp
	return nil
}

// OrderbookDataHandler handles orderbook data, as functionality is shared between updates and snapshots
func (bi *Bitget) orderbookDataHandler(wsResponse *WsResponse) error {
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
	bids, err := trancheConstructor(ob[0].Bids)
	if err != nil {
		return err
	}
	asks, err := trancheConstructor(ob[0].Asks)
	if err != nil {
		return err
	}
	if wsResponse.Action[0] == 's' {
		orderbook := orderbook.Base{
			Pair:                   pair,
			Asset:                  itemDecoder(wsResponse.Arg.InstrumentType),
			Bids:                   bids,
			Asks:                   asks,
			LastUpdated:            wsResponse.Timestamp.Time(),
			Exchange:               bi.Name,
			VerifyOrderbook:        bi.CanVerifyOrderbook,
			ChecksumStringRequired: true,
		}
		err = bi.Websocket.Orderbook.LoadSnapshot(&orderbook)
		if err != nil {
			return err
		}
	} else {
		update := orderbook.Update{
			Bids:       bids,
			Asks:       asks,
			Pair:       pair,
			UpdateTime: wsResponse.Timestamp.Time(),
			Asset:      itemDecoder(wsResponse.Arg.InstrumentType),
			Checksum:   uint32(ob[0].Checksum), //nolint:gosec // The exchange sends it as ints expecting overflows to be handled as Go does by default
		}
		// Sometimes the exchange returns updates with no new asks or bids, just a checksum and timestamp
		if len(update.Bids) != 0 || len(update.Asks) != 0 {
			err = bi.Websocket.Orderbook.Update(&update)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// AccountSnapshotDataHandler handles account snapshot data
func (bi *Bitget) accountSnapshotDataHandler(wsResponse *WsResponse, respRaw []byte) error {
	var hold account.Holdings
	hold.Exchange = bi.Name
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
				Hold:     acc[i].Frozen + acc[i].Locked,
				Free:     acc[i].Available,
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
				Hold:     acc[i].Frozen,
				Free:     acc[i].Available,
				Total:    acc[i].Available + acc[i].Frozen,
			}
		}
	default:
		bi.Websocket.DataHandler <- websocket.UnhandledMessageWarning{Message: bi.Name + websocket.UnhandledMessage + string(respRaw)}
	}
	// Plan to add handling of account.Holdings on websocketDataHandler side in a later PR
	bi.Websocket.DataHandler <- hold
	return nil
}

func (bi *Bitget) fillDataHandler(wsResponse *WsResponse, respRaw []byte) error {
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
				Exchange:     bi.Name,
				AssetType:    asset.Spot,
				CurrencyPair: pair,
				Side:         sideDecoder(fil[i].Side),
				OrderID:      strconv.FormatInt(fil[i].OrderID, 10),
				TradeID:      strconv.FormatInt(fil[i].TradeID, 10),
				Price:        fil[i].PriceAverage,
				Amount:       fil[i].Size,
			}
		}
		bi.Websocket.DataHandler <- resp
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
				Exchange:     bi.Name,
				CurrencyPair: pair,
				OrderID:      strconv.FormatInt(fil[i].OrderID, 10),
				TradeID:      strconv.FormatInt(fil[i].TradeID, 10),
				Side:         sideDecoder(fil[i].Side),
				Price:        fil[i].Price,
				Amount:       fil[i].BaseVolume,
				Timestamp:    fil[i].CreationTime.Time(),
			}
		}
		bi.Websocket.DataHandler <- resp
	default:
		bi.Websocket.DataHandler <- websocket.UnhandledMessageWarning{Message: bi.Name + websocket.UnhandledMessage + string(respRaw)}
	}
	return nil
}

// genOrderDataHandler handles generic order data
func (bi *Bitget) genOrderDataHandler(wsResponse *WsResponse, respRaw []byte) error {
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
				quoteAmount = orders[i].Size
			}
			if side == order.Sell {
				baseAmount = orders[i].Size
			}
			orderType := typeDecoder(orders[i].OrderType)
			if orderType == order.Limit {
				baseAmount = orders[i].NewSize
			}
			resp[i] = order.Detail{
				Exchange:             bi.Name,
				AssetType:            asset.Spot,
				Pair:                 pair,
				OrderID:              strconv.FormatInt(orders[i].OrderID, 10),
				ClientOrderID:        orders[i].ClientOrderID,
				Price:                orders[i].PriceAverage,
				Amount:               baseAmount,
				QuoteAmount:          quoteAmount,
				Type:                 orderType,
				TimeInForce:          strategyDecoder(orders[i].Force),
				Side:                 side,
				AverageExecutedPrice: orders[i].PriceAverage,
				Status:               statusDecoder(orders[i].Status),
				Date:                 orders[i].CreationTime.Time(),
				LastUpdated:          orders[i].UpdateTime.Time(),
			}
			for x := range orders[i].FeeDetail {
				resp[i].Fee += orders[i].FeeDetail[x].TotalFee
				resp[i].FeeAsset = orders[i].FeeDetail[x].FeeCoin
			}
		}
		bi.Websocket.DataHandler <- resp
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
				quoteAmount = orders[i].Size
			}
			if side == order.Sell {
				baseAmount = orders[i].Size
			}
			orderType := typeDecoder(orders[i].OrderType)
			if orderType == order.Limit {
				baseAmount = orders[i].BaseVolume
			}
			resp[i] = order.Detail{
				Exchange:             bi.Name,
				AssetType:            asset.Futures,
				Pair:                 pair,
				Amount:               baseAmount,
				QuoteAmount:          quoteAmount,
				Type:                 orderType,
				TimeInForce:          strategyDecoder(orders[i].Force),
				Side:                 side,
				ExecutedAmount:       orders[i].FilledQuantity,
				Date:                 orders[i].CreationTime.Time(),
				ClientOrderID:        orders[i].ClientOrderID,
				Leverage:             orders[i].Leverage,
				MarginType:           marginDecoder(orders[i].MarginMode),
				OrderID:              strconv.FormatInt(orders[i].OrderID, 10),
				Price:                orders[i].Price,
				AverageExecutedPrice: orders[i].PriceAverage,
				ReduceOnly:           bool(orders[i].ReduceOnly),
				Status:               statusDecoder(orders[i].Status),
				LimitPriceLower:      orders[i].PresetStopSurplusPrice,
				LimitPriceUpper:      orders[i].PresetStopLossPrice,
				LastUpdated:          orders[i].UpdateTime.Time(),
			}
			for x := range orders[i].FeeDetail {
				resp[i].Fee += orders[i].FeeDetail[x].Fee
				resp[i].FeeAsset = orders[i].FeeDetail[x].FeeCoin
			}
		}
		bi.Websocket.DataHandler <- resp
	default:
		bi.Websocket.DataHandler <- websocket.UnhandledMessageWarning{Message: bi.Name + websocket.UnhandledMessage + string(respRaw)}
	}
	return nil
}

// TriggerOrderDataHandler handles trigger order data
func (bi *Bitget) triggerOrderDataHandler(wsResponse *WsResponse, respRaw []byte) error {
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
				Exchange:      bi.Name,
				AssetType:     asset.Spot,
				Pair:          pair,
				OrderID:       strconv.FormatInt(orders[i].OrderID, 10),
				ClientOrderID: orders[i].ClientOrderID,
				TriggerPrice:  orders[i].TriggerPrice,
				Price:         orders[i].Price,
				Amount:        orders[i].Size,
				Type:          typeDecoder(orders[i].OrderType),
				Side:          sideDecoder(orders[i].Side),
				Status:        statusDecoder(orders[i].Status),
				Date:          orders[i].CreationTime.Time(),
				LastUpdated:   orders[i].UpdateTime.Time(),
			}
		}
		bi.Websocket.DataHandler <- resp
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
				Exchange:             bi.Name,
				AssetType:            asset.Futures,
				Pair:                 pair,
				OrderID:              strconv.FormatInt(orders[i].OrderID, 10),
				ClientOrderID:        orders[i].ClientOrderID,
				TriggerPrice:         orders[i].TriggerPrice,
				Price:                orders[i].Price,
				AverageExecutedPrice: orders[i].ExecutePrice,
				Amount:               orders[i].Size,
				Type:                 typeDecoder(orders[i].OrderType),
				Side:                 sideDecoder(orders[i].Side),
				Status:               statusDecoder(orders[i].Status),
				Date:                 orders[i].CreationTime.Time(),
				LastUpdated:          orders[i].UpdateTime.Time(),
			}
		}
		bi.Websocket.DataHandler <- resp
	default:
		bi.Websocket.DataHandler <- websocket.UnhandledMessageWarning{Message: bi.Name + websocket.UnhandledMessage + string(respRaw)}
	}
	return nil
}

// PositionsDataHandler handles data on futures positions
func (bi *Bitget) positionsDataHandler(wsResponse *WsResponse) error {
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
			Exchange:             bi.Name,
			AssetType:            asset.Futures,
			Pair:                 pair,
			OrderID:              strconv.FormatInt(positions[i].PositionID, 10),
			MarginType:           marginDecoder(positions[i].MarginMode),
			Side:                 sideDecoder(positions[i].HoldSide),
			Amount:               positions[i].Total,
			AverageExecutedPrice: positions[i].OpenPriceAverage,
			Leverage:             positions[i].Leverage,
			Date:                 positions[i].CreationTime.Time(),
			Fee:                  positions[i].TotalFee,
			LastUpdated:          positions[i].UpdateTime.Time(),
		}
	}
	bi.Websocket.DataHandler <- resp
	return nil
}

// PositionsHistoryDataHandler handles data on futures positions history
func (bi *Bitget) positionsHistoryDataHandler(wsResponse *WsResponse) error {
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
			Exchange:          bi.Name,
			PositionID:        strconv.FormatInt(positions[i].PositionID, 10),
			Pair:              pair,
			MarginCoin:        positions[i].MarginCoin,
			MarginType:        marginDecoder(positions[i].MarginMode),
			Side:              sideDecoder(positions[i].HoldSide),
			PositionMode:      positionModeDecoder(positions[i].PositionMode),
			OpenAveragePrice:  positions[i].OpenPriceAverage,
			CloseAveragePrice: positions[i].ClosePriceAverage,
			OpenSize:          positions[i].OpenSize,
			CloseSize:         positions[i].CloseSize,
			RealisedPnl:       positions[i].AchievedProfits,
			SettlementFee:     positions[i].SettleFee,
			OpenFee:           positions[i].OpenFee,
			CloseFee:          positions[i].CloseFee,
			StartDate:         positions[i].CreationTime.Time(),
			LastUpdated:       positions[i].UpdateTime.Time(),
		}
	}
	// Implement a better handler for this once work on account.Holdings begins
	bi.Websocket.DataHandler <- resp
	return nil
}

// IndexPriceDataHandler handles index price data
func (bi *Bitget) indexPriceDataHandler(wsResponse *WsResponse) error {
	var indexPrice []WsIndexPriceResponse
	err := json.Unmarshal(wsResponse.Data, &indexPrice)
	if err != nil {
		return err
	}
	resp := make([]ticker.Price, len(indexPrice))
	var cur int
	for i := range indexPrice {
		as := itemDecoder(wsResponse.Arg.InstrumentType)
		pair, enabled, err := bi.MatchSymbolCheckEnabled(indexPrice[i].Symbol, as, false)
		// The exchange sometimes returns unavailable pairs such as "USDT/USDT" which should be ignored
		if !enabled || err != nil {
			continue
		}
		resp[cur] = ticker.Price{
			ExchangeName: bi.Name,
			AssetType:    as,
			Pair:         pair,
			Last:         indexPrice[i].IndexPrice,
			LastUpdated:  indexPrice[i].Timestamp.Time(),
		}
	}
	resp = resp[:cur]
	bi.Websocket.DataHandler <- resp
	return nil
}

// CrossAccountDataHandler handles cross margin account data
func (bi *Bitget) crossAccountDataHandler(wsResponse *WsResponse) error {
	var acc []WsAccountCrossMarginResponse
	err := json.Unmarshal(wsResponse.Data, &acc)
	if err != nil {
		return err
	}
	var hold account.Holdings
	hold.Exchange = bi.Name
	var sub account.SubAccount
	hold.Accounts = append(hold.Accounts, sub)
	sub.AssetType = asset.CrossMargin
	sub.Currencies = make([]account.Balance, len(acc))
	for i := range acc {
		sub.Currencies[i] = account.Balance{
			Currency:               acc[i].Coin,
			Hold:                   acc[i].Frozen,
			Free:                   acc[i].Available,
			Borrowed:               acc[i].Borrow,
			AvailableWithoutBorrow: acc[i].Available,                                                                   // Need to check if Bitget actually calculates values this way
			Total:                  acc[i].Available + acc[i].Frozen + acc[i].Borrow + acc[i].Interest + acc[i].Coupon, // Here too
		}
	}
	bi.Websocket.DataHandler <- hold
	return nil
}

// MarginOrderDataHandler handles margin order data
func (bi *Bitget) marginOrderDataHandler(wsResponse *WsResponse) error {
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
			Exchange:             bi.Name,
			Pair:                 pair,
			OrderID:              strconv.FormatInt(orders[i].OrderID, 10),
			ClientOrderID:        orders[i].ClientOrderID,
			AverageExecutedPrice: orders[i].FillPrice,
			Price:                orders[i].Price,
			Amount:               orders[i].BaseSize,
			QuoteAmount:          orders[i].QuoteSize,
			Type:                 typeDecoder(orders[i].OrderType),
			TimeInForce:          strategyDecoder(orders[i].Force),
			Side:                 sideDecoder(orders[i].Side),
			Status:               statusDecoder(orders[i].Status),
			Date:                 orders[i].CreationTime.Time(),
		}
		for x := range orders[i].FeeDetail {
			resp[i].Fee += orders[i].FeeDetail[x].TotalFee
			resp[i].FeeAsset = orders[i].FeeDetail[x].FeeCoin
		}
		if wsResponse.Arg.Channel == bitgetOrdersIsolatedChannel {
			resp[i].AssetType = asset.Margin
		} else {
			resp[i].AssetType = asset.CrossMargin
		}
	}
	bi.Websocket.DataHandler <- resp
	return nil
}

// IsolatedAccountDataHandler handles isolated margin account data
func (bi *Bitget) isolatedAccountDataHandler(wsResponse *WsResponse) error {
	var acc []WsAccountIsolatedMarginResponse
	err := json.Unmarshal(wsResponse.Data, &acc)
	if err != nil {
		return err
	}
	var hold account.Holdings
	hold.Exchange = bi.Name
	var sub account.SubAccount
	hold.Accounts = append(hold.Accounts, sub)
	sub.AssetType = asset.Margin
	sub.Currencies = make([]account.Balance, len(acc))
	for i := range acc {
		sub.Currencies[i] = account.Balance{
			Currency:               acc[i].Coin,
			Hold:                   acc[i].Frozen,
			Free:                   acc[i].Available,
			Borrowed:               acc[i].Borrow,
			AvailableWithoutBorrow: acc[i].Available,                                                                   // Need to check if Bitget actually calculates values this way
			Total:                  acc[i].Available + acc[i].Frozen + acc[i].Borrow + acc[i].Interest + acc[i].Coupon, // Here too
		}
	}
	bi.Websocket.DataHandler <- hold
	return nil
}

// AccountUpdateDataHandler
func (bi *Bitget) accountUpdateDataHandler(wsResponse *WsResponse, respRaw []byte) error {
	creds, err := bi.GetCredentials(context.TODO())
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
					Hold:      acc[i].Frozen + acc[i].Locked,
					Free:      acc[i].Available,
					Total:     acc[i].Available + acc[i].Frozen + acc[i].Locked,
					UpdatedAt: acc[i].UpdateTime.Time(),
				},
			}
		}
		bi.Websocket.DataHandler <- resp
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
					Currency: acc[i].MarginCoin,
					Hold:     acc[i].Frozen,
					Free:     acc[i].Available,
					Total:    acc[i].Available + acc[i].Frozen,
				},
			}
		}
		bi.Websocket.DataHandler <- resp
	default:
		bi.Websocket.DataHandler <- websocket.UnhandledMessageWarning{Message: bi.Name + websocket.UnhandledMessage + string(respRaw)}
	}
	return account.ProcessChange(bi.Name, resp, creds)
}

// TrancheConstructor turns the exchange's orderbook data into a standardised format for the engine
func trancheConstructor(data [][2]string) ([]orderbook.Tranche, error) {
	resp := make([]orderbook.Tranche, len(data))
	var err error
	for i := range data {
		resp[i] = orderbook.Tranche{
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
func (bi *Bitget) CalculateUpdateOrderbookChecksum(orderbookData *orderbook.Base, checksumVal uint32) error {
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
	if crc32.ChecksumIEEE([]byte(check)) != checksumVal {
		return errInvalidChecksum
	}
	return nil
}

// GenerateDefaultSubscriptions generates default subscriptions
func (bi *Bitget) generateDefaultSubscriptions() (subscription.List, error) {
	at := bi.GetAssetTypes(false)
	assetPairs := make(map[asset.Item]currency.Pairs)
	for i := range at {
		pairs, err := bi.GetEnabledPairs(at[i])
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
func (bi *Bitget) Subscribe(subs subscription.List) error {
	return bi.manageSubs("subscribe", subs)
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (bi *Bitget) Unsubscribe(subs subscription.List) error {
	return bi.manageSubs("unsubscribe", subs)
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
func (bi *Bitget) reqBuilder(req *WsRequest, sub *subscription.Subscription) {
	for i := range sub.Pairs {
		form, _ := bi.GetPairFormat(asset.Spot, true)
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
func (bi *Bitget) manageSubs(op string, subs subscription.List) error {
	unauthBase := &WsRequest{
		Operation: op,
	}
	authBase := &WsRequest{
		Operation: op,
	}
	for _, s := range subs {
		if s.Authenticated {
			bi.reqBuilder(authBase, s)
		} else {
			bi.reqBuilder(unauthBase, s)
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
				err := bi.Websocket.Conn.SendJSONMessage(context.TODO(), RateSubscription, req)
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
				err := bi.Websocket.AuthConn.SendJSONMessage(context.TODO(), RateSubscription, req)
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
func (bi *Bitget) GetSubscriptionTemplate(_ *subscription.Subscription) (*template.Template, error) {
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
