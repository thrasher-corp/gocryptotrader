package huobi

import (
	"log"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/socketio"
)

const (
	HUOBI_SOCKETIO_ADDRESS = "https://hq.huobi.com:443"

	//Service API
	HUOBI_SOCKET_REQ_SYMBOL_LIST   = "reqSymbolList"
	HUOBI_SOCKET_REQ_SYMBOL_DETAIL = "reqSymbolDetail"
	HUOBI_SOCKET_REQ_SUBSCRIBE     = "reqMsgSubscribe"
	HUOBI_SOCKET_REQ_UNSUBSCRIBE   = "reqMsgUnsubscribe"

	// Market data API
	HUOBI_SOCKET_MARKET_DETAIL          = "marketDetail"
	HUOBI_SOCKET_TRADE_DETAIL           = "tradeDetail"
	HUOBI_SOCKET_MARKET_DEPTH_TOP       = "marketDepthTop"
	HUOBI_SOCKET_MARKET_DEPTH_TOP_SHORT = "marketDepthTopShort"
	HUOBI_SOCKET_MARKET_DEPTH           = "marketDepth"
	HUOBI_SOCKET_MARKET_DEPTH_TOP_DIFF  = "marketDepthTopDiff"
	HUOBI_SOCKET_MARKET_DEPTH_DIFF      = "marketDepthDiff"
	HUOBI_SOCKET_MARKET_LAST_KLINE      = "lastKLine"
	HUOBI_SOCKET_MARKET_LAST_TIMELINE   = "lastTimeLine"
	HUOBI_SOCKET_MARKET_OVERVIEW        = "marketOverview"
	HUOBI_SOCKET_MARKET_STATIC          = "marketStatic"

	// History data API
	HUOBI_SOCKET_REQ_TIMELINE         = "reqTimeLine"
	HUOBI_SOCKET_REQ_KLINE            = "reqKLine"
	HUOBI_SOCKET_REQ_DEPTH_TOP        = "reqMarketDepthTop"
	HUOBI_SOCKET_REQ_DEPTH            = "reqMarketDepth"
	HUOBI_SOCKET_REQ_TRADE_DETAIL_TOP = "reqTradeDetailTop"
	HUOBI_SOCKET_REQ_MARKET_DETAIL    = "reqMarketDetail"
)

var HuobiSocket *socketio.SocketIO

type HuobiDepth struct {
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

type HuobiWebsocketTrade struct {
	Price      []float64 `json:"price"`
	Level      []float64 `json:"level"`
	Amount     []float64 `json:"amount"`
	AccuAmount []float64 `json:"accuAmount"`
}

type HuobiWebsocketTradeDetail struct {
	SymbolID string                `json:"symbolId"`
	TradeID  []int64               `json:"tradeId"`
	Price    []float64             `json:"price"`
	Time     []int64               `json:"time"`
	Amount   []float64             `json:"amount"`
	TopBids  []HuobiWebsocketTrade `json:"topBids"`
	TopAsks  []HuobiWebsocketTrade `json:"topAsks"`
}

type HuobiWebsocketMarketOverview struct {
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

type HuobiWebsocketLastTimeline struct {
	ID        int64   `json:"_id"`
	SymbolID  string  `json:"symbolId"`
	Time      int64   `json:"time"`
	LastPrice float64 `json:"priceLast"`
	Amount    float64 `json:"amount"`
	Volume    float64 `json:"volume"`
	Count     int64   `json:"count"`
}

type HuobiResponse struct {
	Version      int                    `json:"version"`
	MsgType      string                 `json:"msgType"`
	RequestIndex int64                  `json:"requestIndex"`
	RetCode      int64                  `json:"retCode"`
	RetMessage   string                 `json:"retMsg"`
	Payload      map[string]interface{} `json:"payload"`
}

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

func (h *HUOBI) BuildHuobiWebsocketRequestExtra(msgType string, requestIndex int64, symbolIdList interface{}) interface{} {
	request := map[string]interface{}{}
	request["version"] = 1
	request["msgType"] = msgType

	if requestIndex != 0 {
		request["requestIndex"] = requestIndex
	}

	request["symbolList"] = symbolIdList
	return request
}

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

func (h *HUOBI) OnConnect(output chan socketio.Message) {
	if h.Verbose {
		log.Printf("%s Connected to Websocket.", h.GetName())
	}

	for _, x := range h.EnabledPairs {
		currency := common.StringToLower(x)
		msg := h.BuildHuobiWebsocketRequestExtra(HUOBI_SOCKET_REQ_SUBSCRIBE, 100, h.BuildHuobiWebsocketParamsList(HUOBI_SOCKET_MARKET_OVERVIEW, currency, "pushLong", "", "", "", "", ""))
		result, err := common.JSONEncode(msg)
		if err != nil {
			log.Println(err)
		}
		output <- socketio.CreateMessageEvent("request", string(result), nil, HuobiSocket.Version)
	}
}

func (h *HUOBI) OnDisconnect(output chan socketio.Message) {
	log.Printf("%s Disconnected from websocket server.. Reconnecting.\n", h.GetName())
	h.WebsocketClient()
}

func (h *HUOBI) OnError() {
	log.Printf("%s Error with Websocket connection.. Reconnecting.\n", h.GetName())
	h.WebsocketClient()
}

func (h *HUOBI) OnMessage(message []byte, output chan socketio.Message) {
}

func (h *HUOBI) OnRequest(message []byte, output chan socketio.Message) {
	response := HuobiResponse{}
	err := common.JSONDecode(message, &response)
	if err != nil {
		log.Println(err)
	}
}

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
		err := socketio.ConnectToSocket(HUOBI_SOCKETIO_ADDRESS, HuobiSocket)
		if err != nil {
			log.Printf("%s Unable to connect to Websocket. Err: %s\n", h.GetName(), err)
			continue
		}
		log.Printf("%s Disconnected from Websocket.\n", h.GetName())
	}
}
