package mexc

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/buger/jsonparser"
	gws "github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/mexc/mexc_proto_types"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
	"google.golang.org/protobuf/proto"
)

const (
	wsURL = "wss://wbs-api.mexc.com/ws"

	chnlBookTiker            = "public.aggre.bookTicker.v3.api.pb"
	chnlAggregateDepthV3     = "public.aggre.depth.v3.api.pb"
	chnlAggreDealsV3         = "public.aggre.deals.v3.api.pb"
	chnlKlineV3              = "public.kline.v3.api.pb"
	chnlLimitDepthV3         = "public.limit.depth.v3.api.pb"
	chnlBookTickerBatch      = "public.bookTicker.batch.v3.api.pb"
	chnlAccountV3            = "private.account.v3.api.pb"
	chnlPrivateDealsV3       = "private.deals.v3.api.pb"
	chnlPrivateOrdersAPI     = "private.orders.v3.api.pb"
	chnlIncreaseDepthBatchV3 = "public.increase.depth.batch.v3.api.pb"
)

var defacultChannels = []string{
	chnlBookTiker,
	chnlKlineV3,
	chnlAggreDealsV3,
	chnlAggregateDepthV3,
	chnlIncreaseDepthBatchV3,
}

// orderbookSnapshotLoadedPairs and syncOrderbookPairsLock holds list of symbols and if these instruments snapshot orderbook detail is loaded, and corresponding lock
var (
	orderbookSnapshotLoadedPairs = map[string]bool{}
	syncOrderbookPairsLock       sync.Mutex
)

// WsConnect initiates a websocket connection
func (me *MEXC) WsConnect() error {
	if !me.Websocket.IsEnabled() || !me.IsEnabled() {
		return websocket.ErrWebsocketNotEnabled
	}
	dialer := gws.Dialer{
		EnableCompression: true,
		ReadBufferSize:    8192,
		WriteBufferSize:   8192,
	}
	if me.Websocket.CanUseAuthenticatedEndpoints() {
		listenKey, err := me.GenerateListenKey(context.Background())
		if err != nil {
			return err
		}
		me.Websocket.Conn.SetURL(me.Websocket.Conn.GetURL() + "?listenKey=" + listenKey)
	}
	err := me.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}
	me.Websocket.Wg.Add(1)
	go me.wsReadData(me.Websocket.Conn)
	if me.Verbose {
		log.Debugf(log.ExchangeSys, "Successful connection to %v\n",
			me.Websocket.GetWebsocketURL())
	}
	me.Websocket.Conn.SetupPingHandler(request.Unset, websocket.PingHandler{
		MessageType: gws.TextMessage,
		Message:     []byte(`{"method": "PING"}`),
		Delay:       time.Second * 20,
	})
	return nil
}

// wsReadData sends msgs from public and auth websockets to data handler
func (me *MEXC) wsReadData(ws websocket.Connection) {
	defer me.Websocket.Wg.Done()
	for {
		resp := ws.ReadMessage()
		if len(resp.Raw) == 0 {
			return
		}
		if err := me.WsHandleData(resp.Raw); err != nil {
			me.Websocket.DataHandler <- err
		}
	}
}

// generateSubscriptions returns a list of subscriptions from the configured subscriptions feature
func (me *MEXC) generateSubscriptions() (subscription.List, error) {
	enabledPairs, err := me.GetEnabledPairs(asset.Spot)
	if err != nil {
		return nil, err
	}
	formatter, err := me.GetPairFormat(asset.Spot, true)
	if err != nil {
		return nil, err
	}
	subscriptions := make(subscription.List, len(defacultChannels))
	for c := range defacultChannels {
		subscriptions[c] = &subscription.Subscription{
			Channel: defacultChannels[c],
			Pairs:   enabledPairs.Format(formatter),
			Asset:   asset.Spot,
		}
		switch defacultChannels[c] {
		case chnlBookTiker,
			chnlAggregateDepthV3,
			chnlAggreDealsV3:
			subscriptions[c].Interval = kline.HundredMilliseconds
		case chnlKlineV3:
			subscriptions[c].Interval = kline.FifteenMin
		case chnlLimitDepthV3:
			subscriptions[c].Levels = 5
		case chnlAccountV3,
			chnlPrivateDealsV3,
			chnlPrivateOrdersAPI:
			subscriptions[c].Pairs = []currency.Pair{}
		}
	}
	return subscriptions, nil
}

// Subscribe subscribes to a channel
func (me *MEXC) Subscribe(channelsToSubscribe subscription.List) error {
	return me.handleSubscription("SUBSCRIPTION", channelsToSubscribe)
}

// Unsubscribe unsubscribes to a channel
func (me *MEXC) Unsubscribe(channelsToSubscribe subscription.List) error {
	return me.handleSubscription("UNSUBSCRIPTION", channelsToSubscribe)
}

func assetTypeToString(assetType asset.Item) (string, error) {
	switch assetType {
	case asset.Spot, asset.Futures:
		return strings.ToLower(assetType.String()), nil
	default:
		return "", fmt.Errorf("%w: asset type: %v", asset.ErrNotSupported, assetType)
	}
}

func (me *MEXC) handleSubscription(method string, subs subscription.List) error {
	payloads := make([]WsSubscriptionPayload, len(subs))
	successfulSubscriptions := subscription.List{}
	failedSubscriptions := subscription.List{}
	for s := range subs {
		assetTypeString, err := assetTypeToString(subs[s].Asset)
		if err != nil {
			return err
		}
		switch subs[s].Channel {
		case chnlBookTiker,
			chnlAggregateDepthV3,
			chnlAggreDealsV3,
			chnlKlineV3:
			intervalString, err := intervalToString(subs[s].Interval, true)
			if err != nil {
				return err
			}
			payloads[s].ID = me.Websocket.Conn.GenerateMessageID(false)
			payloads[s].Method = method
			payloads[s].Params = make([]string, len(subs[s].Pairs))
			for p := range subs[s].Pairs {
				if subs[s].Channel == chnlKlineV3 {
					payloads[s].Params[p] = assetTypeString + "@" + subs[s].Channel + "@" + subs[s].Pairs[p].String() + "@" + intervalString
				} else {
					payloads[s].Params[p] = assetTypeString + "@" + subs[s].Channel + "@" + intervalString + "@" + subs[s].Pairs[p].String()
				}
			}
			data, err := me.Websocket.Conn.SendMessageReturnResponse(context.Background(), request.UnAuth, payloads[s].ID, payloads[s])
			if err != nil {
				return err
			}
			var resp *WsSubscriptionResponse
			err = json.Unmarshal(data, &resp)
			if err != nil {
				return err
			} else if resp.Code != 0 {
				failedSubscriptions = append(failedSubscriptions, subs[s])
			}
			successfulSubscriptions = append(successfulSubscriptions, subs[s])
		case chnlLimitDepthV3:
			payloads[s].ID = me.Websocket.Conn.GenerateMessageID(false)
			payloads[s].Method = method
			payloads[s].Params = make([]string, len(subs[s].Pairs))
			for p := range subs[s].Pairs {
				payloads[s].Params[p] = assetTypeString + "@" + chnlLimitDepthV3 + "@" + subs[s].Pairs[p].String() + "@" + strconv.Itoa(subs[s].Levels)
			}
			data, err := me.Websocket.Conn.SendMessageReturnResponse(context.Background(), request.UnAuth, payloads[s].ID, payloads[s])
			if err != nil {
				return err
			}
			var resp *WsSubscriptionResponse
			err = json.Unmarshal(data, &resp)
			if err != nil {
				return err
			}
		case chnlAccountV3, chnlPrivateDealsV3, chnlPrivateOrdersAPI:
			payloads[s].ID = me.Websocket.Conn.GenerateMessageID(false)
			payloads[s].Method = method
			payloads[s].Params = []string{assetTypeString + "@" + subs[s].Channel}
			data, err := me.Websocket.Conn.SendMessageReturnResponse(context.Background(), request.UnAuth, payloads[s].ID, payloads[s])
			if err != nil {
				return err
			}
			var resp *WsSubscriptionResponse
			err = json.Unmarshal(data, &resp)
			if err != nil {
				return err
			}
		case chnlIncreaseDepthBatchV3, chnlBookTickerBatch:
			payloads[s].ID = me.Websocket.Conn.GenerateMessageID(false)
			payloads[s].Method = method
			payloads[s].Params = make([]string, len(subs[s].Pairs))
			for p := range subs[s].Pairs {
				payloads[s].Params[p] = assetTypeString + "@" + subs[s].Channel + "@" + subs[s].Pairs[p].String()
			}
			data, err := me.Websocket.Conn.SendMessageReturnResponse(context.Background(), request.UnAuth, payloads[s].ID, payloads[s])
			if err != nil {
				return err
			}
			var resp *WsSubscriptionResponse
			err = json.Unmarshal(data, &resp)
			if err != nil {
				return err
			}
		}
	}
	err := me.Websocket.RemoveSubscriptions(me.Websocket.Conn, failedSubscriptions...)
	if err != nil {
		return err
	}
	return me.Websocket.AddSuccessfulSubscriptions(me.Websocket.Conn, successfulSubscriptions...)
}

// WsHandleData will read websocket raw data and pass to appropriate handler
func (me *MEXC) WsHandleData(respRaw []byte) error {
	if strings.HasPrefix(string(respRaw), "{") {
		if id, err := jsonparser.GetInt(respRaw, "id"); err == nil {
			if !me.Websocket.Match.IncomingWithData(id, respRaw) {
				me.Websocket.DataHandler <- websocket.UnhandledMessageWarning{
					Message: string(respRaw) + websocket.UnhandledMessage,
				}
			}
		}
		// Ignore json messages which doesn't have an ID.
		return nil
	}
	dataSplit := strings.Split(string(respRaw), "@")
	switch dataSplit[1] {
	case chnlBookTiker:
		result := &mexc_proto_types.PushDataV3ApiWrapper{
			Body: &mexc_proto_types.PushDataV3ApiWrapper_PublicAggreBookTicker{},
		}
		err := proto.Unmarshal(respRaw, result)
		if err != nil {
			return err
		}
		body := result.GetPublicAggreBookTicker()
		ask := orderbook.Tranche{}
		ask.Price, err = strconv.ParseFloat(body.AskPrice, 64)
		if err != nil {
			return err
		}
		ask.Amount, err = strconv.ParseFloat(body.AskQuantity, 64)
		if err != nil {
			return err
		}
		bid := orderbook.Tranche{}
		bid.Price, err = strconv.ParseFloat(body.BidPrice, 64)
		if err != nil {
			return err
		}
		bid.Amount, err = strconv.ParseFloat(body.BidQuantity, 64)
		if err != nil {
			return err
		}
		cp, err := me.MatchSymbolWithAvailablePairs(*result.Symbol, asset.Spot, false)
		if err != nil {
			return err
		}
		if ok := orderbookSnapshotLoadedPairs[dataSplit[2]]; !ok {
			err = me.Websocket.Orderbook.LoadSnapshot(&orderbook.Base{
				Exchange:    me.Name,
				Asset:       asset.Spot,
				Asks:        []orderbook.Tranche{ask},
				Bids:        []orderbook.Tranche{bid},
				Pair:        cp,
				LastUpdated: time.Now(),
			})
			if err != nil {
				return err
			}
			syncOrderbookPairsLock.Lock()
			orderbookSnapshotLoadedPairs[dataSplit[2]] = true
			syncOrderbookPairsLock.Unlock()
			return nil
		}
		return me.Websocket.Orderbook.Update(&orderbook.Update{
			Pair:       cp,
			Asset:      asset.Spot,
			Asks:       []orderbook.Tranche{ask},
			Bids:       []orderbook.Tranche{bid},
			UpdateTime: time.Now(),
		})
	case chnlAggregateDepthV3:
		result := mexc_proto_types.PushDataV3ApiWrapper{
			Body: &mexc_proto_types.PushDataV3ApiWrapper_PublicAggreDepths{},
		}
		err := proto.Unmarshal(respRaw, &result)
		if err != nil {
			return err
		}
		depths := result.GetPublicAggreDepths()
		cp, err := me.MatchSymbolWithAvailablePairs(*result.Symbol, asset.Spot, false)
		if err != nil {
			return err
		}
		format, err := me.GetPairFormat(asset.Spot, false)
		if err != nil {
			return err
		}
		asks := make(orderbook.Tranches, len(depths.Asks))
		for a := range depths.Asks {
			asks[a].Price, err = strconv.ParseFloat(depths.Asks[a].Price, 64)
			if err != nil {
				return err
			}
			asks[a].Amount, err = strconv.ParseFloat(depths.Asks[a].Quantity, 64)
			if err != nil {
				return err
			}
		}
		bids := make(orderbook.Tranches, len(depths.Bids))
		for b := range depths.Bids {
			bids[b].Price, err = strconv.ParseFloat(depths.Bids[b].Price, 64)
			if err != nil {
				return err
			}
			bids[b].Amount, err = strconv.ParseFloat(depths.Bids[b].Quantity, 64)
			if err != nil {
				return err
			}
		}

		if !orderbookSnapshotLoadedPairs[*result.Symbol] {
			err = me.Websocket.Orderbook.LoadSnapshot(&orderbook.Base{
				Exchange:    me.Name,
				Asset:       asset.Spot,
				Asks:        asks,
				Bids:        bids,
				Pair:        cp.Format(format),
				LastUpdated: time.Now(),
			})
			if err != nil {
				return err
			}
			syncOrderbookPairsLock.Lock()
			orderbookSnapshotLoadedPairs[*result.Symbol] = true
			syncOrderbookPairsLock.Unlock()
		}
		return me.Websocket.Orderbook.Update(&orderbook.Update{
			Asset:      asset.Spot,
			Asks:       asks,
			Bids:       bids,
			Pair:       cp.Format(format),
			UpdateTime: time.Now(),
		})
	case chnlAggreDealsV3:
		result := mexc_proto_types.PushDataV3ApiWrapper{
			Body: &mexc_proto_types.PushDataV3ApiWrapper_PublicAggreDeals{},
		}
		err := proto.Unmarshal(respRaw, &result)
		if err != nil {
			return err
		}
		cp, err := me.MatchSymbolWithAvailablePairs(*result.Symbol, asset.Spot, false)
		if err != nil {
			return err
		}
		body := result.GetPublicAggreDeals()
		tradesDetail := make([]trade.Data, len(body.Deals))
		for t := range body.Deals {
			price, err := strconv.ParseFloat(body.Deals[t].Price, 64)
			if err != nil {
				return err
			}
			amount, err := strconv.ParseFloat(body.Deals[t].Quantity, 64)
			if err != nil {
				return err
			}
			tradesDetail[t] = trade.Data{
				Exchange:     me.Name,
				CurrencyPair: cp,
				AssetType:    asset.Spot,
				Price:        price,
				Amount:       amount,
				Timestamp:    time.UnixMilli(body.Deals[t].Time),
				Side: func() order.Side {
					if body.Deals[t].TradeType == 1 {
						return order.Buy
					}
					return order.Sell
				}(),
			}
		}
		me.Websocket.DataHandler <- tradesDetail
		return nil
	case chnlKlineV3:
		result := &mexc_proto_types.PushDataV3ApiWrapper{
			Body: &mexc_proto_types.PushDataV3ApiWrapper_PublicSpotKline{},
		}
		err := proto.Unmarshal(respRaw, result)
		if err != nil {
			return err
		}
		body := result.GetPublicSpotKline()
		cp, err := me.MatchSymbolWithAvailablePairs(*result.Symbol, asset.Spot, false)
		if err != nil {
			return err
		}
		klineData := websocket.KlineData{
			Pair:      cp,
			Exchange:  me.Name,
			AssetType: asset.Spot,
			Interval:  body.Interval,
		}
		klineData.CloseTime = time.UnixMilli(body.WindowEnd)
		if klineData.Volume, err = strconv.ParseFloat(body.Amount, 64); err != nil {
			return err
		}
		klineData.StartTime = time.UnixMilli(body.WindowStart)
		klineData.LowPrice, err = strconv.ParseFloat(body.LowestPrice, 64)
		if err != nil {
			return err
		}
		klineData.HighPrice, err = strconv.ParseFloat(body.HighestPrice, 64)
		if err != nil {
			return err
		}
		klineData.LowPrice, err = strconv.ParseFloat(body.LowestPrice, 64)
		if err != nil {
			return err
		}
		klineData.OpenPrice, err = strconv.ParseFloat(body.OpeningPrice, 64)
		if err != nil {
			return err
		}
		klineData.ClosePrice, err = strconv.ParseFloat(body.ClosingPrice, 64)
		if err != nil {
			return err
		}
		me.Websocket.DataHandler <- []websocket.KlineData{klineData}
		return nil
	case chnlIncreaseDepthBatchV3:
		result := &mexc_proto_types.PushDataV3ApiWrapper{
			Body: &mexc_proto_types.PushDataV3ApiWrapper_PublicIncreaseDepthsBatch{},
		}
		err := proto.Unmarshal(respRaw, result)
		if err != nil {
			return err
		}
		cp, err := me.MatchSymbolWithAvailablePairs(*result.Symbol, asset.Spot, true)
		if err != nil {
			return err
		}
		body := result.GetPublicIncreaseDepthsBatch()
		for ob := range body.Items {
			asks := make(orderbook.Tranches, len(body.Items[ob].Asks))
			for a := range body.Items[ob].Asks {
				asks[a].Price, err = strconv.ParseFloat(body.Items[ob].Asks[a].Price, 64)
				if err != nil {
					return err
				}
				asks[a].Amount, err = strconv.ParseFloat(body.Items[ob].Asks[a].Quantity, 64)
				if err != nil {
					return err
				}
			}
			bids := make(orderbook.Tranches, len(body.Items[ob].Bids))
			for b := range body.Items[ob].Bids {
				bids[b].Price, err = strconv.ParseFloat(body.Items[ob].Bids[b].Price, 64)
				if err != nil {
					return err
				}
				bids[b].Amount, err = strconv.ParseFloat(body.Items[ob].Bids[b].Quantity, 64)
				if err != nil {
					return err
				}
			}
			if ok := orderbookSnapshotLoadedPairs[dataSplit[2]]; !ok {
				err = me.Websocket.Orderbook.LoadSnapshot(&orderbook.Base{
					Exchange:    me.Name,
					Pair:        cp,
					Asks:        asks,
					Bids:        bids,
					Asset:       asset.Spot,
					LastUpdated: time.Now(),
				})
				if err != nil {
					return err
				}
				syncOrderbookPairsLock.Lock()
				orderbookSnapshotLoadedPairs[dataSplit[2]] = true
				syncOrderbookPairsLock.Unlock()
			}
			err = me.Websocket.Orderbook.Update(&orderbook.Update{
				Pair:       cp,
				Asks:       asks,
				Bids:       bids,
				UpdateTime: time.Now(),
				Asset:      asset.Spot,
			})
			if err != nil {
				return err
			}
		}
	case chnlLimitDepthV3:
		result := &mexc_proto_types.PushDataV3ApiWrapper{
			Body: &mexc_proto_types.PushDataV3ApiWrapper_PublicLimitDepths{},
		}
		err := proto.Unmarshal(respRaw, result)
		if err != nil {
			return err
		}
		cp, err := me.MatchSymbolWithAvailablePairs(*result.Symbol, asset.Spot, false)
		if err != nil {
			return err
		}
		body := result.GetPublicLimitDepths()
		asks := make(orderbook.Tranches, len(body.Asks))
		for a := range body.Asks {
			asks[a].Price, err = strconv.ParseFloat(body.Asks[a].Price, 64)
			if err != nil {
				return err
			}
			asks[a].Amount, err = strconv.ParseFloat(body.Asks[a].Quantity, 64)
			if err != nil {
				return err
			}
		}
		bids := make(orderbook.Tranches, len(body.Bids))
		for b := range body.Bids {
			bids[b].Price, err = strconv.ParseFloat(body.Bids[b].Price, 64)
			if err != nil {
				return err
			}
			bids[b].Amount, err = strconv.ParseFloat(body.Bids[b].Quantity, 64)
			if err != nil {
				return err
			}
		}
		return me.Websocket.Orderbook.LoadSnapshot(&orderbook.Base{
			Asset:       asset.Spot,
			Bids:        bids,
			Asks:        asks,
			Pair:        cp,
			LastUpdated: time.Now(),
		})
	case chnlBookTickerBatch:
		result := &mexc_proto_types.PushDataV3ApiWrapper{
			Body: &mexc_proto_types.PushDataV3ApiWrapper_PublicBookTickerBatch{},
		}
		err := proto.Unmarshal(respRaw, result)
		if err != nil {
			return err
		}
		cp, err := me.MatchSymbolWithAvailablePairs(*result.Symbol, asset.Spot, true)
		if err != nil {
			return err
		}
		body := result.GetPublicBookTickerBatch()
		tickersDetail := make([]ticker.Price, len(body.Items))
		for a := range body.Items {
			tickersDetail[a] = ticker.Price{
				Pair:         cp,
				ExchangeName: me.Name,
				AssetType:    asset.Spot,
			}
			tickersDetail[a].Bid, err = strconv.ParseFloat(body.Items[a].BidPrice, 64)
			if err != nil {
				return err
			}
			tickersDetail[a].Ask, err = strconv.ParseFloat(body.Items[a].AskPrice, 64)
			if err != nil {
				return err
			}
			tickersDetail[a].BidSize, err = strconv.ParseFloat(body.Items[a].BidQuantity, 64)
			if err != nil {
				return err
			}
			tickersDetail[a].AskSize, err = strconv.ParseFloat(body.Items[a].AskQuantity, 64)
			if err != nil {
				return err
			}
		}
		me.Websocket.DataHandler <- tickersDetail
		return nil
	case chnlAccountV3:
		result := &mexc_proto_types.PushDataV3ApiWrapper{
			Body: &mexc_proto_types.PushDataV3ApiWrapper_PrivateAccount{},
		}
		err := proto.Unmarshal(respRaw, result)
		if err != nil {
			return err
		}
		body := result.GetPrivateAccount()
		balanceAmount, err := strconv.ParseFloat(body.BalanceAmount, 64)
		if err != nil {
			return err
		}
		frozenAmount, err := strconv.ParseFloat(body.FrozenAmount, 64)
		if err != nil {
			return err
		}
		me.Websocket.DataHandler <- account.Change{
			AssetType: asset.Spot,
			Balance: &account.Balance{
				Currency: currency.NewCode(body.VcoinName),
				Total:    balanceAmount,
				Hold:     frozenAmount,
				Free:     balanceAmount - frozenAmount,
			},
		}
		return nil
	case chnlPrivateDealsV3:
		result := &mexc_proto_types.PushDataV3ApiWrapper{
			Body: &mexc_proto_types.PushDataV3ApiWrapper_PrivateDeals{},
		}
		err := proto.Unmarshal(respRaw, result)
		if err != nil {
			return err
		}
		cp, err := me.MatchSymbolWithAvailablePairs(*result.Symbol, asset.Spot, false)
		if err != nil {
			return err
		}
		body := result.GetPrivateDeals()
		price, err := strconv.ParseFloat(body.Price, 64)
		if err != nil {
			return err
		}
		amount, err := strconv.ParseFloat(body.Amount, 64)
		if err != nil {
			return err
		}
		dealTimeMilli, err := strconv.ParseInt(body.Time, 10, 64)
		if err != nil {
			return err
		}
		me.Websocket.DataHandler <- []trade.Data{
			{
				TID:          body.OrderId,
				Exchange:     me.Name,
				CurrencyPair: cp,
				AssetType:    asset.Spot,
				Price:        price,
				Amount:       amount,
				Timestamp:    time.UnixMilli(dealTimeMilli),
				Side: func() order.Side {
					if body.TradeType == 1 {
						return order.Buy
					}
					return order.Sell
				}(),
			},
		}
		return nil
	case chnlPrivateOrdersAPI:
		result := mexc_proto_types.PushDataV3ApiWrapper{
			Body: &mexc_proto_types.PushDataV3ApiWrapper_PrivateOrders{},
		}
		err := proto.Unmarshal(respRaw, &result)
		if err != nil {
			return err
		}
		var oType order.Type
		var tif order.TimeInForce
		body := result.GetPrivateOrders()
		switch body.OrderType {
		case 1:
			tif = order.GoodTillCancel
			oType = order.Limit
		case 2:
			tif = order.PostOnly
			oType = order.Market
		case 3:
			tif = order.ImmediateOrCancel
			oType = order.Market
		case 4:
			oType = order.Market
			tif = order.FillOrKill
		case 5:
			oType = order.Market
		case 100:
			oType = order.OCO
		}
		var oStatus order.Status
		switch body.Status {
		case 1:
			oStatus = order.New
		case 2:
			oStatus = order.Filled
		case 3:
			oStatus = order.PartiallyFilled
		case 4:
			oStatus = order.Cancelled
		case 5:
			oStatus = order.PartiallyCancelled
		}
		cp, err := me.MatchSymbolWithAvailablePairs(*result.Symbol, asset.Spot, false)
		if err != nil {
			return err
		}
		me.Websocket.DataHandler <- &order.Detail{
			Price:                body.Price.Float64(),
			Amount:               body.Amount.Float64(),
			ContractAmount:       body.Quantity.Float64(),
			AverageExecutedPrice: body.AvgPrice.Float64(),
			QuoteAmount:          body.Amount.Float64(),
			ExecutedAmount:       body.CumulativeAmount.Float64() - body.RemainAmount.Float64(),
			RemainingAmount:      body.RemainAmount.Float64(),
			Exchange:             me.Name,
			OrderID:              body.Id,
			ClientID:             body.ClientId,
			Type:                 oType,
			Side: func() order.Side {
				if body.TradeType == 1 {
					return order.Buy
				}
				return order.Sell
			}(),
			Status:      oStatus,
			AssetType:   asset.Spot,
			LastUpdated: body.CreateTime.Time(),
			Pair:        cp,
			TimeInForce: tif,
		}
	default:
		me.Websocket.DataHandler <- websocket.UnhandledMessageWarning{
			Message: string(respRaw) + websocket.UnhandledMessage,
		}
	}
	return nil
}
