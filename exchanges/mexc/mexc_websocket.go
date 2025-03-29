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
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/mexc/mexc_proto_types"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
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
		switch subs[s].Channel {
		case chnlBookTiker,
			chnlAggregateDepthV3,
			chnlAggreDealsV3,
			chnlKlineV3:
			assetTypeString, err := assetTypeToString(subs[s].Asset)
			if err != nil {
				return err
			}
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
			assetTypeString, err := assetTypeToString(subs[s].Asset)
			if err != nil {
				return err
			}
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
			assetTypeString, err := assetTypeToString(subs[s].Asset)
			if err != nil {
				return err
			}
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
			assetTypeString, err := assetTypeToString(subs[s].Asset)
			if err != nil {
				return err
			}
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

	for p := range payloads {
		data, err := me.Websocket.Conn.SendMessageReturnResponse(context.Background(), request.UnAuth, payloads[p].ID, &payloads[p])
		if err != nil {
			return err
		}
		println(string(data))
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
	channelDetail := strings.Split(string(respRaw), "@")[1]
	println("channelDetail: ", channelDetail)
	println(string(respRaw))
	switch channelDetail {
	case chnlBookTiker:
	case chnlAggregateDepthV3:
	case chnlDealsV3:
	case chnlIncreaseDepthV3:
	case chnlAggreDealsV3:
	case chnlKlineV3:
	case chnlIncreaseDepthBatchV3:
	case chnlLimitDepthV3:
	case chnlBookTickerBatch:
	case chnlAccountV3:
	case chnlPrivateDealsV3:
	case chnlPrivateOrdersAPI:
		println(string(respRaw))
	case "":
	}
	var depthData mexc_proto_types.PublicAggreDepthsV3Api
	err := proto.Unmarshal(respRaw, &depthData)
	if err != nil {
		println("\n\nERROR " + err.Error() + "\n\n")
		return err
		// panic(err)
	}
	return nil
}

func processPublicAggregatedDepth(data []byte, in interface{}) error {
	return nil
}
