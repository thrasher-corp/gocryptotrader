package mexc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/buger/jsonparser"
	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/mexc/mexc_proto_types"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
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

// WsConnect initiates a websocket connection
func (me *MEXC) WsConnect() error {
	me.Websocket.Enable()
	if !me.Websocket.IsEnabled() || !me.IsEnabled() {
		return stream.ErrWebsocketNotEnabled
	}
	var dialer = websocket.Dialer{
		EnableCompression: true,
		ReadBufferSize:    8192,
		WriteBufferSize:   8192,
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
	me.Websocket.Conn.SetupPingHandler(request.Unset, stream.PingHandler{
		MessageType: websocket.TextMessage,
		Message:     []byte(`{"method": "PING"}`),
		Delay:       time.Second * 20,
	})
	if me.Websocket.CanUseAuthenticatedEndpoints() {
		// err = me.WsAuth(context.TODO())
		// if err != nil {
		// 	log.Errorf(log.ExchangeSys, "Error connecting auth socket: %s\n", err.Error())
		// 	me.Websocket.SetCanUseAuthenticatedEndpoints(false)
		// }
	}
	return me.Websocket.Conn.SendJSONMessage(context.Background(), request.UnAuth, &WsSubscriptionPayload{Method: "SUBSCRIPTION", Params: []string{"spot@public.aggre.depth.v3.api.pb@100ms@BTCUSDT"}})
}

// wsReadData sends msgs from public and auth websockets to data handler
func (me *MEXC) wsReadData(ws stream.Connection) {
	defer me.Websocket.Wg.Done()
	for {
		resp := ws.ReadMessage()
		if resp.Raw == nil {
			return
		}
		if err := me.WsHandleData(resp.Raw); err != nil {
			panic(err.Error())
			me.Websocket.DataHandler <- err
		}
	}
}

// generateSubscriptions returns a list of subscriptions from the configured subscriptions feature
func (me *MEXC) generateSubscriptions() (subscription.List, error) {
	return subscription.List{}, nil
	// return me.Features.Subscriptions.ExpandTemplates(me)
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
	print("respRaw[0]: ", respRaw[0])
	if strings.HasPrefix(string(respRaw), "{") {
		if id, err := jsonparser.GetInt(respRaw, "id"); err == nil {
			if !me.Websocket.Match.IncomingWithData(id, respRaw) {
				me.Websocket.DataHandler <- stream.UnhandledMessageWarning{
					Message: string(respRaw) + stream.UnhandledMessage,
				}
			}
			return nil
		}
		// Ignore json messages which doesn't have an ID.
		return nil
	}
	channelDetail := strings.Split(string(respRaw), "@")
	println("channelDetail: ", channelDetail)
	println(string(respRaw))
	switch channelDetail[1] {
	case chnlBookTiker:
		var result mexc_proto_types.PublicAggreBookTickerV3Api
		err := proto.Unmarshal(respRaw, &result)
		if err != nil {
			return err
		}
		cp, err := currency.NewPairFromString(channelDetail[2])
		if err != nil {
			return err
		}
		return me.Websocket.Orderbook.Update(&orderbook.Update{
			Pair:  cp,
			Asset: asset.Spot,
			Asks: []orderbook.Tranche{
				{
					Price:  result.AskPrice.Float64(),
					Amount: result.AskQuantity.Float64(),
				},
			},
			Bids: []orderbook.Tranche{
				{
					Price:  result.BidPrice.Float64(),
					Amount: result.BidQuantity.Float64(),
				},
			},
		})
	case chnlAggregateDepthV3:
		var result mexc_proto_types.PublicAggreDepthsV3Api
		err := proto.Unmarshal(respRaw, &result)
		if err != nil {
			return err
		}
		cp, err := currency.NewPairFromString(channelDetail[3])
		if err != nil {
			return err
		}
		asks := make(orderbook.Tranches, len(result.Asks))
		for a := range result.Asks {
			asks[a] = orderbook.Tranche{
				Price:  result.Asks[a].Price.Float64(),
				Amount: result.Asks[a].Quantity.Float64(),
			}
		}
		bids := make(orderbook.Tranches, len(result.Bids))
		for b := range result.Bids {
			asks[b] = orderbook.Tranche{
				Price:  result.Asks[b].Price.Float64(),
				Amount: result.Asks[b].Quantity.Float64(),
			}
		}
		return me.Websocket.Orderbook.LoadSnapshot(&orderbook.Base{
			Asset:       asset.Spot,
			Bids:        bids,
			Asks:        asks,
			Pair:        cp,
			LastUpdated: time.Now(),
		})
	case chnlDealsV3:
		var result mexc_proto_types.PublicDealsV3Api
		err := proto.Unmarshal(respRaw, &result)
		if err != nil {
			return err
		}
		cp, err := currency.NewPairFromString(channelDetail[2])
		if err != nil {
			return err
		}
		tradesDetail := make([]trade.Data, len(result.Deals))
		for t := range result.Deals {
			tradesDetail[t] = trade.Data{
				Exchange:     me.Name,
				CurrencyPair: cp,
				AssetType:    asset.Spot,
				Price:        result.Deals[t].Price.Float64(),
				Amount:       result.Deals[t].Quantity.Float64(),
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
		cp, err := currency.NewPairFromString(channelDetail[2])
		if err != nil {
			return err
		}
		asks := make(orderbook.Tranches, len(result.Asks))
		for a := range result.Asks {
			asks[a] = orderbook.Tranche{
				Price:  result.Asks[a].Price.Float64(),
				Amount: result.Asks[a].Quantity.Float64(),
			}
		}
		bids := make(orderbook.Tranches, len(result.Bids))
		for b := range result.Bids {
			asks[b] = orderbook.Tranche{
				Price:  result.Asks[b].Price.Float64(),
				Amount: result.Asks[b].Quantity.Float64(),
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
		cp, err := currency.NewPairFromString(channelDetail[2])
		if err != nil {
			return err
		}
		tradesDetail := make([]trade.Data, len(result.Deals))
		for t := range result.Deals {
			tradesDetail[t] = trade.Data{
				Exchange:     me.Name,
				CurrencyPair: cp,
				AssetType:    asset.Spot,
				Price:        result.Deals[t].Price.Float64(),
				Amount:       result.Deals[t].Quantity.Float64(),
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
	case chnlKlineV3:
		var result mexc_proto_types.PublicSpotKlineV3Api
		err := proto.Unmarshal(respRaw, &result)
		if err != nil {
			return err
		}
		cp, err := currency.NewPairFromString(channelDetail[2])
		if err != nil {
			return err
		}
		me.Websocket.DataHandler <- []stream.KlineData{{
			Pair:       cp,
			Exchange:   me.Name,
			AssetType:  asset.Spot,
			Interval:   result.Interval,
			CloseTime:  result.WindowEnd.Time(),
			Volume:     result.Amount.Float64(),
			StartTime:  result.WindowStart.Time(),
			LowPrice:   result.LowestPrice.Float64(),
			OpenPrice:  result.OpeningPrice.Float64(),
			ClosePrice: result.ClosingPrice.Float64(),
			HighPrice:  result.HighestPrice.Float64(),
		}}
		return nil
	case chnlIncreaseDepthBatchV3:
		var result mexc_proto_types.PublicIncreaseDepthsBatchV3Api
		err := proto.Unmarshal(respRaw, &result)
		if err != nil {
			return err
		}
		cp, err := currency.NewPairFromString(channelDetail[2])
		if err != nil {
			return err
		}
		for ob := range result.Items {
			asks := make(orderbook.Tranches, len(result.Items[ob].Asks))
			for a := range result.Items[ob].Asks {
				asks[a] = orderbook.Tranche{
					Price:  result.Items[ob].Asks[a].Price.Float64(),
					Amount: result.Items[ob].Asks[a].Quantity.Float64(),
				}
			}
			bids := make(orderbook.Tranches, len(result.Items[ob].Bids))
			for b := range result.Items[ob].Bids {
				bids[b] = orderbook.Tranche{
					Price:  result.Items[ob].Bids[b].Price.Float64(),
					Amount: result.Items[ob].Bids[b].Quantity.Float64(),
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
		cp, err := currency.NewPairFromString(channelDetail[2])
		if err != nil {
			return err
		}

		asks := make(orderbook.Tranches, len(result.Asks))
		for a := range result.Asks {
			asks[a] = orderbook.Tranche{
				Price:  result.Asks[a].Price.Float64(),
				Amount: result.Asks[a].Quantity.Float64(),
			}
		}
		bids := make(orderbook.Tranches, len(result.Bids))
		for b := range result.Bids {
			asks[b] = orderbook.Tranche{
				Price:  result.Asks[b].Price.Float64(),
				Amount: result.Asks[b].Quantity.Float64(),
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
		cp, err := currency.NewPairFromString(channelDetail[2])
		if err != nil {
			return err
		}
		tickersDetail := make([]ticker.Price, len(result.Items))
		for a := range result.Items {
			tickersDetail[a] = ticker.Price{
				Pair:         cp,
				ExchangeName: me.Name,
				AssetType:    asset.Spot,
				Bid:          result.Items[a].BidPrice.Float64(),
				Ask:          result.Items[a].AskPrice.Float64(),
				BidSize:      result.Items[a].BidQuantity.Float64(),
				AskSize:      result.Items[a].AskQuantity.Float64(),
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
		me.Websocket.DataHandler <- account.Change{
			Exchange: me.Name,
			Currency: currency.NewCode(result.VcoinName),
			Asset:    asset.Spot,
			Amount:   result.BalanceAmount.Float64(),
			Account:  result.Type,
		}
		return nil
	case chnlPrivateDealsV3:
		// var result mexc_proto_types.PushDataV3ApiWrapper
		// err := proto.Unmarshal(respRaw, &result)
		// if err != nil {
		// 	return err
		// }
		// cp, err := currency.NewPairFromString(*result.Symbol)
		// if err != nil {
		// 	return err
		// }
		// orderFills := make([]fill.Data, len(result.))
		// me.Websocket.DataHandler <- fill.Data{
		// 	ID:           result.TradeId,
		// 	Timestamp:    result.Time.Time(),
		// 	Exchange:     me.Name,
		// 	AssetType:    asset.Spot,
		// 	CurrencyPair: cp,
		// 	// Side
		// 	// OrderID
		// 	// ClientOrderID
		// 	// TradeID
		// 	// Price
		// 	// Amount
		// }
		// return nil
	case chnlPrivateOrdersAPI:
	default:
		me.Websocket.DataHandler <- stream.UnhandledMessageWarning{
			Message: string(respRaw) + stream.UnhandledMessage,
		}
		return nil
	}
	return nil
}
