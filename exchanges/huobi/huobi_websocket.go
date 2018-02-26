package huobi

import (
	"log"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/socketio"
)

const (
	huobiSocketIOAddress = "https://hq.huobi.com:443"

	//Service API
	huobiSocketReqSymbolList   = "reqSymbolList"
	huobiSocketReqSymbolDetail = "reqSymbolDetail"
	huobiSocketReqSubscribe    = "reqMsgSubscribe"
	huobiSocketReqUnsubscribe  = "reqMsgUnsubscribe"

	// Market data API
	huobiSocketMarketDetail        = "marketDetail"
	huobiSocketTradeDetail         = "tradeDetail"
	huobiSocketMarketDepthTop      = "marketDepthTop"
	huobiSocketMarketDepthTopShort = "marketDepthTopShort"
	huobiSocketMarketDepth         = "marketDepth"
	huobiSocketMarketDepthTopDiff  = "marketDepthTopDiff"
	huobiSocketMarketDepthDiff     = "marketDepthDiff"
	huobiSocketMarketLastKline     = "lastKLine"
	huobiSocketMarketLastTimeline  = "lastTimeLine"
	huobiSocketMarketOverview      = "marketOverview"
	huobiSocketMarketStatic        = "marketStatic"

	// History data API
	huobiSocketReqTimeline       = "reqTimeLine"
	huobiSocketReqKline          = "reqKLine"
	huobiSocketReqDepthTop       = "reqMarketDepthTop"
	huobiSocketReqDepth          = "reqMarketDepth"
	huobiSocketReqTradeDetailTop = "reqTradeDetailTop"
	huobiSocketReqMarketDetail   = "reqMarketDetail"
)

// HuobiSocket is a pointer to a IO Socket
var HuobiSocket *socketio.SocketIO

// Depth holds depth information
type Depth struct {
	SymbolID  string    `json:"symbolId"`
	Time      float64   `json:"time"`
	Version   float64   `json:"version"`
	BidName   string    `json:"bidName"`
	BidPrice  []float64 `json:"bidPrice"`
	BidTotal  []float64 `json:"bidTotal"`
	BidAmount []float64 `json:"bidAmount"`
	AskName   string    `json:"askName"`
	AskPrice  []float64 `json:"askPrice"`
	AskTotal  []float64 `json:"askTotal"`
	AskAmount []float64 `json:"askAmount"`
}

// WebsocketTrade holds full trade data
type WebsocketTrade struct {
	Price      []float64 `json:"price"`
	Level      []float64 `json:"level"`
	Amount     []float64 `json:"amount"`
	AccuAmount []float64 `json:"accuAmount"`
}

// WebsocketTradeDetail holds specific trade details
type WebsocketTradeDetail struct {
	SymbolID string           `json:"symbolId"`
	TradeID  []int64          `json:"tradeId"`
	Price    []float64        `json:"price"`
	Time     []int64          `json:"time"`
	Amount   []float64        `json:"amount"`
	TopBids  []WebsocketTrade `json:"topBids"`
	TopAsks  []WebsocketTrade `json:"topAsks"`
}

// WebsocketMarketOverview holds market overview data
type WebsocketMarketOverview struct {
	SymbolID    string  `json:"symbolId"`
	Last        float64 `json:"priceNew"`
	Open        float64 `json:"priceOpen"`
	High        float64 `json:"priceHigh"`
	Low         float64 `json:"priceLow"`
	Ask         float64 `json:"priceAsk"`
	Bid         float64 `json:"priceBid"`
	Volume      float64 `json:"totalVolume"`
	TotalAmount float64 `json:"totalAmount"`
}

// WebsocketLastTimeline holds timeline data
type WebsocketLastTimeline struct {
	ID        int64   `json:"_id"`
	SymbolID  string  `json:"symbolId"`
	Time      int64   `json:"time"`
	LastPrice float64 `json:"priceLast"`
	Amount    float64 `json:"amount"`
	Volume    float64 `json:"volume"`
	Count     int64   `json:"count"`
}

// WebsocketResponse is a general response type for websocket
type WebsocketResponse struct {
	Version      int                    `json:"version"`
	MsgType      string                 `json:"msgType"`
	RequestIndex int64                  `json:"requestIndex"`
	RetCode      int64                  `json:"retCode"`
	RetMessage   string                 `json:"retMsg"`
	Payload      map[string]interface{} `json:"payload"`
}

// BuildHuobiWebsocketRequest packages a new request
func (h *HUOBI) BuildHuobiWebsocketRequest(msgType string, requestIndex int64, symbolRequest []string) map[string]interface{} {
	request := map[string]interface{}{}
	request["version"] = 1
	request["msgType"] = msgType

	if requestIndex != 0 {
		request["requestIndex"] = requestIndex
	}

	if len(symbolRequest) != 0 {
		request["symbolIdList"] = symbolRequest
	}

	return request
}

// BuildHuobiWebsocketRequestExtra packages an extra request
func (h *HUOBI) BuildHuobiWebsocketRequestExtra(msgType string, requestIndex int64, symbolIDList interface{}) interface{} {
	request := map[string]interface{}{}
	request["version"] = 1
	request["msgType"] = msgType

	if requestIndex != 0 {
		request["requestIndex"] = requestIndex
	}

	request["symbolList"] = symbolIDList
	return request
}

// BuildHuobiWebsocketParamsList packages a parameter list
func (h *HUOBI) BuildHuobiWebsocketParamsList(objectName, currency, pushType, period, count, from, to, percentage string) interface{} {
	list := map[string]interface{}{}
	list["symbolId"] = currency
	list["pushType"] = pushType

	if period != "" {
		list["period"] = period
	}
	if percentage != "" {
		list["percent"] = percentage
	}
	if count != "" {
		list["count"] = count
	}
	if from != "" {
		list["from"] = from
	}
	if to != "" {
		list["to"] = to
	}

	listArray := []map[string]interface{}{}
	listArray = append(listArray, list)

	listCompleted := make(map[string][]map[string]interface{})
	listCompleted[objectName] = listArray
	return listCompleted
}

// OnConnect handles connection establishment
func (h *HUOBI) OnConnect(output chan socketio.Message) {
	if h.Verbose {
		log.Printf("%s Connected to Websocket.", h.GetName())
	}

	for _, x := range h.EnabledPairs {
		currency := common.StringToLower(x)
		msg := h.BuildHuobiWebsocketRequestExtra(huobiSocketReqSubscribe, 100, h.BuildHuobiWebsocketParamsList(huobiSocketMarketOverview, currency, "pushLong", "", "", "", "", ""))
		result, err := common.JSONEncode(msg)
		if err != nil {
			log.Println(err)
		}
		output <- socketio.CreateMessageEvent("request", string(result), nil, HuobiSocket.Version)
	}
}

// OnDisconnect handles disconnection
func (h *HUOBI) OnDisconnect(output chan socketio.Message) {
	log.Printf("%s Disconnected from websocket server.. Reconnecting.\n", h.GetName())
	h.WebsocketClient()
}

// OnError handles error issues
func (h *HUOBI) OnError() {
	log.Printf("%s Error with Websocket connection.. Reconnecting.\n", h.GetName())
	h.WebsocketClient()
}

// OnMessage handles messages from the exchange
func (h *HUOBI) OnMessage(message []byte, output chan socketio.Message) {
}

// OnRequest handles requests
func (h *HUOBI) OnRequest(message []byte, output chan socketio.Message) {
	response := WebsocketResponse{}
	err := common.JSONDecode(message, &response)
	if err != nil {
		log.Println(err)
	}
}

// WebsocketClient creates a new websocket client
func (h *HUOBI) WebsocketClient() {
	events := make(map[string]func(message []byte, output chan socketio.Message))
	events["request"] = h.OnRequest
	events["message"] = h.OnMessage

	HuobiSocket = &socketio.SocketIO{
		Version:      0.9,
		OnConnect:    h.OnConnect,
		OnEvent:      events,
		OnError:      h.OnError,
		OnDisconnect: h.OnDisconnect,
	}

	for h.Enabled && h.Websocket {
		err := socketio.ConnectToSocket(huobiSocketIOAddress, HuobiSocket)
		if err != nil {
			log.Printf("%s Unable to connect to Websocket. Err: %s\n", h.GetName(), err)
			continue
		}
		log.Printf("%s Disconnected from Websocket.\n", h.GetName())
	}
}
