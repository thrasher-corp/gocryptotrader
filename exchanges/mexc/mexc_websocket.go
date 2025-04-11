package mexc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/buger/jsonparser"
	gws "github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/currency"
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

	chnlBookTiker        = "public.aggre.bookTicker.v3.api.pb"
	chnlAggregateDepthV3 = "public.aggre.depth.v3.api.pb"
	chnlDealsV3          = "public.deals.v3.api.pb"
	chnlIncreaseDepthV3  = "public.increase.depth.v3.api.pb"
	chnlAggreDealsV3     = "public.aggre.deals.v3.api.pb"
	chnlKlineV3          = "public.kline.v3.api.pb"
	chnlLimitDepthV3     = "public.limit.depth.v3.api.pb"
	chnlBookTickerBatch  = "public.bookTicker.batch.v3.api.pb"
	chnlAccountV3        = "private.account.v3.api.pb"
	chnlPrivateDealsV3   = "private.deals.v3.api.pb"
	chnlPrivateOrdersAPI = "private.orders.v3.api.pb"

	chnlIncreaseDepthBatchV3 = "public.increase.depth.batch.v3.api.pb"
)

var defacultChannels = []string{chnlBookTiker, chnlAggregateDepthV3, chnlDealsV3, chnlIncreaseDepthV3}

// WsConnect initiates a websocket connection
func (me *MEXC) WsConnect() error {
	me.Websocket.Enable()
	if !me.Websocket.IsEnabled() || !me.IsEnabled() {
		return websocket.ErrWebsocketNotEnabled
	}
	var dialer = gws.Dialer{
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
	return me.Websocket.Conn.SendJSONMessage(context.Background(), request.UnAuth, &WsSubscriptionPayload{Method: "SUBSCRIPTION", Params: []string{"spot@public.aggre.depth.v3.api.pb@100ms@BTCUSDT"}})
}

// wsReadData sends msgs from public and auth websockets to data handler
func (me *MEXC) wsReadData(ws websocket.Connection) {
	defer me.Websocket.Wg.Done()
	for {
		resp := ws.ReadMessage()
		if resp.Raw == nil {
			return
		}
		if err := me.WsHandleData(resp.Raw); err != nil {
			me.Websocket.DataHandler <- err
		}
	}
}

// generateSubscriptions returns a list of subscriptions from the configured subscriptions feature
func (me *MEXC) generateSubscriptions() (subscription.List, error) {
	subscriptions := subscription.List{}
	assets := []asset.Item{asset.Spot, asset.Futures}
	for _, a := range assets {
		enabledPair, err := me.GetEnabledPairs(a)
		if err != nil {
			return nil, err
		}
		for c := range defacultChannels {
			item := &subscription.Subscription{
				Channel: defacultChannels[c],
				Pairs:   enabledPair,
				Asset:   a,
			}
			switch defacultChannels[c] {
			case chnlBookTiker,
				chnlAggregateDepthV3,
				chnlAggreDealsV3:
				item.Interval = kline.HundredMilliseconds
			case chnlKlineV3:
				item.Interval = kline.FifteenMin
			case chnlLimitDepthV3:
				item.Levels = 5
			case chnlAccountV3,
				chnlPrivateDealsV3,
				chnlPrivateOrdersAPI:
				item.Pairs = []currency.Pair{}
			}
			subscriptions = append(subscriptions, item)
		}
	}
	return subscriptions, nil
}

func (me *MEXC) Subscribe(channelsToSubscribe subscription.List) error {
	return me.handleSubscription("SUBSCRIPTION", channelsToSubscribe)
}

func (me *MEXC) Unsubscribe(channelsToSubscribe subscription.List) error {
	return me.handleSubscription("UNSUBSCRIPTION", channelsToSubscribe)
}

func assetTypeToString(assetType asset.Item) (string, error) {
	switch assetType {
	case asset.Spot,
		asset.Futures:
		return strings.ToLower(assetType.String()), nil
	default:
		return "", fmt.Errorf("%w: asset type: %v", asset.ErrNotSupported, assetType)
	}
}

func (me *MEXC) handleSubscription(method string, subs subscription.List) error {
	payloads := make([]WsSubscriptionPayload, len(subs))
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
			}
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
		case chnlIncreaseDepthV3, chnlDealsV3, chnlIncreaseDepthBatchV3, chnlBookTickerBatch:
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
	return nil
}

// WsHandleData will read websocket raw data and pass to appropriate handler
func (me *MEXC) WsHandleData(respRaw []byte) error {
	println(string(respRaw))
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
	println("dataSplit[1]: ", dataSplit[1])
	switch dataSplit[1] {
	case chnlBookTiker:
		var result mexc_proto_types.PublicAggreBookTickerV3Api
		err := proto.Unmarshal(respRaw, &result)
		if err != nil {
			return err
		}
		cp, err := currency.NewPairFromString(dataSplit[2])
		if err != nil {
			return err
		}
		ask := orderbook.Tranche{}
		ask.Price, err = strconv.ParseFloat(result.AskPrice, 64)
		if err != nil {
			return err
		}
		ask.Amount, err = strconv.ParseFloat(result.AskQuantity, 64)
		if err != nil {
			return err
		}
		bid := orderbook.Tranche{}
		bid.Price, err = strconv.ParseFloat(result.BidPrice, 64)
		if err != nil {
			return err
		}
		bid.Amount, err = strconv.ParseFloat(result.BidQuantity, 64)
		if err != nil {
			return err
		}
		return me.Websocket.Orderbook.Update(&orderbook.Update{
			Pair:  cp,
			Asset: asset.Spot,
			Asks:  []orderbook.Tranche{ask},
			Bids:  []orderbook.Tranche{bid},
		})
	case chnlAggregateDepthV3:
		println("Here: ...")
		var result mexc_proto_types.PublicAggreDepthsV3Api
		err := proto.Unmarshal(respRaw, &result)
		if err != nil {
			println(err.Error())
			return err
		}
		// valu, err := json.Marshal(&result)
		// if err != nil {
		// 	return err
		// }
		// println("string(valu): ", string(valu))
		os.Exit(0)
		cp, err := currency.NewPairFromString(dataSplit[3])
		if err != nil {
			return err
		}
		asks := make(orderbook.Tranches, len(result.Asks))
		for a := range result.Asks {
			asks[a].Price, err = strconv.ParseFloat(result.Asks[a].Price, 64)
			if err != nil {
				return err
			}
			asks[a].Amount, err = strconv.ParseFloat(result.Asks[a].Quantity, 64)
			if err != nil {
				return err
			}
		}
		bids := make(orderbook.Tranches, len(result.Bids))
		for b := range result.Bids {
			bids[b].Price, err = strconv.ParseFloat(result.Bids[b].Price, 64)
			if err != nil {
				return err
			}
			bids[b].Amount, err = strconv.ParseFloat(result.Bids[b].Quantity, 64)
			if err != nil {
				return err
			}
		}
		return me.Websocket.Orderbook.LoadSnapshot(&orderbook.Base{
			Asset:       asset.Spot,
			Asks:        asks,
			Bids:        bids,
			Pair:        cp,
			LastUpdated: time.Now(),
		})
	case chnlDealsV3:
		var result mexc_proto_types.PublicDealsV3Api
		err := proto.Unmarshal(respRaw, &result)
		if err != nil {
			return err
		}
		cp, err := currency.NewPairFromString(dataSplit[2])
		if err != nil {
			return err
		}
		tradesDetail := make([]trade.Data, len(result.Deals))
		for t := range result.Deals {
			price, err := strconv.ParseFloat(result.Deals[t].Price, 64)
			if err != nil {
				return err
			}
			quantity, err := strconv.ParseFloat(result.Deals[t].Quantity, 64)
			if err != nil {
				return err
			}
			tradesDetail[t] = trade.Data{
				Exchange:     me.Name,
				CurrencyPair: cp,
				AssetType:    asset.Spot,
				Price:        price,
				Amount:       quantity,
				Timestamp:    result.Deals[t].Time.Time(),
				Side: func() order.Side {
					if result.Deals[t].TradeType == 1 {
						return order.Buy
					}
					return order.Sell
				}(),
			}
		}
		me.Websocket.DataHandler <- tradesDetail
		return nil
	case chnlIncreaseDepthV3:
		var result mexc_proto_types.PublicIncreaseDepthsV3Api
		err := proto.Unmarshal(respRaw, &result)
		if err != nil {
			return err
		}
		cp, err := currency.NewPairFromString(dataSplit[2])
		if err != nil {
			return err
		}
		asks := make(orderbook.Tranches, len(result.Asks))
		for a := range result.Asks {
			asks[a].Price, err = strconv.ParseFloat(result.Asks[a].Price, 64)
			if err != nil {
				return err
			}
			asks[a].Amount, err = strconv.ParseFloat(result.Asks[a].Quantity, 64)
			if err != nil {
				return err
			}
		}
		bids := make(orderbook.Tranches, len(result.Bids))
		for b := range result.Bids {
			bids[b].Price, err = strconv.ParseFloat(result.Bids[b].Price, 64)
			if err != nil {
				return err
			}
			bids[b].Amount, err = strconv.ParseFloat(result.Bids[b].Quantity, 64)
			if err != nil {
				return err
			}
		}
		return me.Websocket.Orderbook.Update(&orderbook.Update{
			Asset: asset.Spot,
			Bids:  bids,
			Asks:  asks,
			Pair:  cp,

			UpdateTime: time.Now(),
		})
	case chnlAggreDealsV3:
		var result mexc_proto_types.PublicAggreDealsV3Api
		err := proto.Unmarshal(respRaw, &result)
		if err != nil {
			return err
		}
		cp, err := currency.NewPairFromString(dataSplit[2])
		if err != nil {
			return err
		}
		tradesDetail := make([]trade.Data, len(result.Deals))
		for t := range result.Deals {
			price, err := strconv.ParseFloat(result.Deals[t].Price, 64)
			if err != nil {
				return err
			}
			amount, err := strconv.ParseFloat(result.Deals[t].Quantity, 64)
			if err != nil {
				return err
			}
			dealTime, err := strconv.ParseInt(result.Deals[t].Time, 10, 64)
			if err != nil {
				return err
			}
			tradesDetail[t] = trade.Data{
				Exchange:     me.Name,
				CurrencyPair: cp,
				AssetType:    asset.Spot,
				Price:        price,
				Amount:       amount,
				Timestamp:    time.UnixMilli(dealTime),
				Side: func() order.Side {
					if result.Deals[t].TradeType == 1 {
						return order.Buy
					}
					return order.Sell
				}(),
			}
		}
		me.Websocket.DataHandler <- tradesDetail
		return nil
	case chnlKlineV3:
		var result mexc_proto_types.PublicSpotKlineV3Api
		err := proto.Unmarshal(respRaw, &result)
		if err != nil {
			return err
		}
		cp, err := currency.NewPairFromString(dataSplit[2])
		if err != nil {
			return err
		}
		klineData := websocket.KlineData{
			Pair:      cp,
			Exchange:  me.Name,
			AssetType: asset.Spot,
			Interval:  result.Interval,
		}
		windowEndUnixMilli, err := strconv.ParseInt(result.WindowEnd, 10, 64)
		if err != nil {
			return err
		}
		klineData.CloseTime = time.UnixMilli(windowEndUnixMilli)
		if klineData.Volume, err = strconv.ParseFloat(result.Amount, 64); err != nil {
			return err
		}
		klineStartTimeUnixMilli, err := strconv.ParseInt(result.WindowStart, 10, 64)
		if err != nil {
			return err
		}
		klineData.StartTime = time.UnixMilli(klineStartTimeUnixMilli)
		klineData.LowPrice, err = strconv.ParseFloat(result.LowestPrice, 64)
		if err != nil {
			return err
		}
		klineData.HighPrice, err = strconv.ParseFloat(result.HighestPrice, 64)
		if err != nil {
			return err
		}
		klineData.LowPrice, err = strconv.ParseFloat(result.LowestPrice, 64)
		if err != nil {
			return err
		}
		klineData.OpenPrice, err = strconv.ParseFloat(result.OpeningPrice, 64)
		if err != nil {
			return err
		}
		klineData.ClosePrice, err = strconv.ParseFloat(result.ClosingPrice, 64)
		if err != nil {
			return err
		}
		me.Websocket.DataHandler <- []websocket.KlineData{klineData}
		return nil
	case chnlIncreaseDepthBatchV3:
		var result mexc_proto_types.PublicIncreaseDepthsBatchV3Api
		err := proto.Unmarshal(respRaw, &result)
		if err != nil {
			return err
		}
		cp, err := currency.NewPairFromString(dataSplit[2])
		if err != nil {
			return err
		}
		for ob := range result.Items {
			asks := make(orderbook.Tranches, len(result.Items[ob].Asks))
			for a := range result.Items[ob].Asks {
				asks[a].Price, err = strconv.ParseFloat(result.Items[ob].Asks[a].Price, 64)
				if err != nil {
					return err
				}
				asks[a].Amount, err = strconv.ParseFloat(result.Items[ob].Asks[a].Quantity, 64)
				if err != nil {
					return err
				}
			}
			bids := make(orderbook.Tranches, len(result.Items[ob].Bids))
			for b := range result.Items[ob].Bids {
				bids[b].Price, err = strconv.ParseFloat(result.Items[ob].Bids[b].Price, 64)
				if err != nil {
					return err
				}
				bids[b].Amount, err = strconv.ParseFloat(result.Items[ob].Bids[b].Quantity, 64)
				if err != nil {
					return err
				}
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
		var result mexc_proto_types.PublicLimitDepthsV3Api
		err := proto.Unmarshal(respRaw, &result)
		if err != nil {
			return err
		}
		cp, err := currency.NewPairFromString(dataSplit[2])
		if err != nil {
			return err
		}

		asks := make(orderbook.Tranches, len(result.Asks))
		for a := range result.Asks {
			asks[a].Price, err = strconv.ParseFloat(result.Asks[a].Price, 64)
			if err != nil {
				return err
			}
			asks[a].Amount, err = strconv.ParseFloat(result.Asks[a].Quantity, 64)
			if err != nil {
				return err
			}
		}
		bids := make(orderbook.Tranches, len(result.Bids))
		for b := range result.Bids {
			bids[b].Price, err = strconv.ParseFloat(result.Bids[b].Price, 64)
			if err != nil {
				return err
			}
			bids[b].Amount, err = strconv.ParseFloat(result.Bids[b].Quantity, 64)
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
		var result mexc_proto_types.PublicBookTickerBatchV3Api
		err := proto.Unmarshal(respRaw, &result)
		if err != nil {
			return err
		}
		cp, err := currency.NewPairFromString(dataSplit[2])
		if err != nil {
			return err
		}
		tickersDetail := make([]ticker.Price, len(result.Items))
		for a := range result.Items {
			tickersDetail[a] = ticker.Price{
				Pair:         cp,
				ExchangeName: me.Name,
				AssetType:    asset.Spot,
			}
			tickersDetail[a].Bid, err = strconv.ParseFloat(result.Items[a].BidPrice, 64)
			if err != nil {
				return err
			}
			tickersDetail[a].Ask, err = strconv.ParseFloat(result.Items[a].AskPrice, 64)
			if err != nil {
				return err
			}
			tickersDetail[a].BidSize, err = strconv.ParseFloat(result.Items[a].BidQuantity, 64)
			if err != nil {
				return err
			}
			tickersDetail[a].AskSize, err = strconv.ParseFloat(result.Items[a].AskQuantity, 64)
			if err != nil {
				return err
			}
		}
		me.Websocket.DataHandler <- tickersDetail
		return nil
	case chnlAccountV3:
		var result mexc_proto_types.PrivateAccountV3Api
		err := proto.Unmarshal(respRaw, &result)
		if err != nil {
			return err
		}
		balanceAmount, err := strconv.ParseFloat(result.BalanceAmount, 64)
		if err != nil {
			return err
		}
		me.Websocket.DataHandler <- account.Change{
			Exchange: me.Name,
			Currency: currency.NewCode(result.VcoinName),
			Asset:    asset.Spot,
			Amount:   balanceAmount,
			Account:  result.Type,
		}
		return nil
	case chnlPrivateDealsV3:
		body := &mexc_proto_types.PushDataV3ApiWrapper_PrivateDeals{}
		result := mexc_proto_types.PushDataV3ApiWrapper{
			Body: body,
		}
		err := proto.Unmarshal(respRaw, &result)
		if err != nil {
			return err
		}
		cp, err := currency.NewPairFromString(*result.Symbol)
		if err != nil {
			return err
		}
		price, err := strconv.ParseFloat(body.PrivateDeals.Price, 64)
		if err != nil {
			return err
		}
		amount, err := strconv.ParseFloat(body.PrivateDeals.Amount, 64)
		if err != nil {
			return err
		}
		dealTimeMilli, err := strconv.ParseInt(body.PrivateDeals.Time, 10, 64)
		if err != nil {
			return err
		}
		me.Websocket.DataHandler <- []trade.Data{
			{
				TID:          body.PrivateDeals.OrderId,
				Exchange:     me.Name,
				CurrencyPair: cp,
				AssetType:    asset.Spot,
				Price:        price,
				Amount:       amount,
				Timestamp:    time.UnixMilli(dealTimeMilli),
				Side: func() order.Side {
					if body.PrivateDeals.TradeType == 1 {
						return order.Buy
					}
					return order.Sell
				}(),
			},
		}
		return nil
	case chnlPrivateOrdersAPI:
		body := &mexc_proto_types.PushDataV3ApiWrapper_PrivateOrders{}
		result := mexc_proto_types.PushDataV3ApiWrapper{
			Body: body,
		}
		err := proto.Unmarshal(respRaw, &result)
		if err != nil {
			return err
		}
		var oType order.Type
		switch body.PrivateOrders.OrderType {
		case 1:
			oType = order.Limit
		case 2:
			oType = order.PostOnly
		case 3:
			oType = order.ImmediateOrCancel
		case 4:
			oType = order.FillOrKill
		case 5:
			oType = order.Market
		case 100:
			oType = order.OCO
		}
		var oStatus order.Status
		switch body.PrivateOrders.Status {
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
		cp, err := currency.NewPairFromString(*result.Symbol)
		if err != nil {
			return err
		}
		me.Websocket.DataHandler <- &order.Detail{
			PostOnly:             body.PrivateOrders.OrderType == 2,
			Price:                body.PrivateOrders.Price.Float64(),
			Amount:               body.PrivateOrders.Amount.Float64(),
			ContractAmount:       body.PrivateOrders.Quantity.Float64(),
			AverageExecutedPrice: body.PrivateOrders.AvgPrice.Float64(),
			QuoteAmount:          body.PrivateOrders.Amount.Float64(),
			ExecutedAmount:       body.PrivateOrders.CumulativeAmount.Float64() - body.PrivateOrders.RemainAmount.Float64(),
			RemainingAmount:      body.PrivateOrders.RemainAmount.Float64(),
			Exchange:             me.Name,
			OrderID:              body.PrivateOrders.Id,
			ClientID:             body.PrivateOrders.ClientId,
			Type:                 oType,
			Side: func() order.Side {
				if body.PrivateOrders.TradeType == 1 {
					return order.Buy
				}
				return order.Sell
			}(),
			Status:      oStatus,
			AssetType:   asset.Spot,
			LastUpdated: body.PrivateOrders.CreateTime.Time(),
			Pair:        cp,
		}
	default:
		me.Websocket.DataHandler <- websocket.UnhandledMessageWarning{
			Message: string(respRaw) + websocket.UnhandledMessage,
		}
	}
	return nil
}
