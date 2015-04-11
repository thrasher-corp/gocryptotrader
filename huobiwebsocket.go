package main

import (
	"fmt"
	"log"
	"github.com/thrasher-/socketio"
)

const (
	HUOBI_SOCKETIO_ADDRESS = "https://hq.huobi.com:443"

	//Service API
	HUOBI_SOCKET_REQ_SYMBOL_LIST = "reqSymbolList"
	HUOBI_SOCKET_REQ_SYMBOL_DETAIL = "reqSymbolDetail"
	HUOBI_SOCKET_REQ_SUBSCRIBE = "reqMsgSubscribe"
	HUOBI_SOCKET_REQ_UNSUBSCRIBE = "reqMsgUnsubscribe"

	// Market data API
	HUOBI_SOCKET_MARKET_DETAIL = "marketDetail"
	HUOBI_SOCKET_TRADE_DETAIL = "tradeDetail"
	HUOBI_SOCKET_MARKET_DEPTH_TOP = "marketDepthTop"
	HUOBI_SOCKET_MARKET_DEPTH_TOP_SHORT = "marketDepthTopShort"
	HUOBI_SOCKET_MARKET_DEPTH = "marketDepth"
	HUOBI_SOCKET_MARKET_DEPTH_TOP_DIFF = "marketDepthTopDiff"
	HUOBI_SOCKET_MARKET_DEPTH_DIFF = "marketDepthDiff"
	HUOBI_SOCKET_MARKET_LAST_KLINE = "lastKLine"
	HUOBI_SOCKET_MARKET_LAST_TIMELINE = "lastTimeLine"
	HUOBI_SOCKET_MARKET_OVERVIEW = "marketOverview"
	HUOBI_SOCKET_MARKET_STATIC = "marketStatic"

	// History data API
	HUOBI_SOCKET_REQ_TIMELINE = "reqTimeLine"
	HUOBI_SOCKET_REQ_KLINE = "reqKLine"
	HUOBI_SOCKET_REQ_DEPTH_TOP = "reqMarketDepthTop"
	HUOBI_SOCKET_REQ_DEPTH = "reqMarketDepth"
	HUOBI_SOCKET_REQ_TRADE_DETAIL_TOP = "reqTradeDetailTop"
	HUOBI_SOCKET_REQ_MARKET_DETAIL = "reqMarketDetail"
)

var HuobiSocket *socketio.SocketIO

type HuobiWebsocketMarketOverview struct {
	SymbolID string `json:"symbolId"`
	Last float64 `json:"priceNew"`
	Open float64 `json:"priceOpen"`
	High float64 `json:"priceHigh"`
	Low float64 `json:"priceLow"`
	Ask float64 `json:"priceAsk"`
	Bid float64 `json:"priceBid"`
	Volume float64 `json:"totalVolume"`
	TotalAmount float64 `json:"totalAmount"`
}

type HuobiResponse struct {
	Version int `json:"version"`
	MsgType string `json:"msgType"`
	RequestIndex int64 `json:"requestIndex"`
	RetCode int64 `json:"retCode"`
	RetMessage string `json:"retMsg"`
	Payload map[string]interface{} `json:"payload"`
}

func (h *HUOBI) BuildHuobiWebsocketRequest(msgType string, requestIndex int64, symbolRequest []string) (map[string]interface{}) {
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

func (h *HUOBI) BuildHuobiWebsocketRequestExtra(msgType string, requestIndex int64, symbolIdList interface{}) (interface{}) {
	request := map[string]interface{}{}
	request["version"] = 1
	request["msgType"] = msgType

	if requestIndex != 0 {
		request["requestIndex"] = requestIndex
	}

	request["symbolList"] = symbolIdList
	return request
}

func (h *HUOBI) BuildHuobiWebsocketParamsList(objectName, currency, pushType, period, percentage string) (interface{}) {
	list := map[string]interface{}{}
	list["symbolId"] = currency
	list["pushType"] = pushType

	if period != "" {
		list["period"] = period
	}
	if percentage != "" {
		list["percentage"] = percentage
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

	msg := h.BuildHuobiWebsocketRequestExtra(HUOBI_SOCKET_REQ_SUBSCRIBE, 100, h.BuildHuobiWebsocketParamsList(HUOBI_SOCKET_MARKET_OVERVIEW, "btccny", "pushLong", "", ""))
	result, err := JSONEncode(msg)
	if err != nil {
		log.Println(err)
	}
	output <- socketio.CreateMessageEvent("request", string(result), nil)

	msg = h.BuildHuobiWebsocketRequestExtra(HUOBI_SOCKET_REQ_SUBSCRIBE, 100, h.BuildHuobiWebsocketParamsList(HUOBI_SOCKET_MARKET_OVERVIEW, "ltccny", "pushLong", "", ""))
	result, err = JSONEncode(msg)
	if err != nil {
		log.Println(err)
	}
	output <- socketio.CreateMessageEvent("request", string(result), nil)
}

func (h *HUOBI) OnDisconnect(output chan socketio.Message) {
	log.Println("Disconnected from websocket client.. Reconnecting")
	h.WebsocketClient()
}

func (h *HUOBI) OnMessage(message []byte, output chan socketio.Message) {
	log.Println(string(message))
}

func (h *HUOBI) OnRequest(message []byte, output chan socketio.Message) {
	response := HuobiResponse{}
	log.Println(string(message))
	err := JSONDecode(message, &response)
	if err != nil {
		log.Println(err)
	}
}

func (h *HUOBI) WebsocketClient() {
	events := make(map[string]func(message []byte, output chan socketio.Message))
	events["request"] = h.OnRequest
	events["message"] = h.OnMessage

	HuobiSocket = &socketio.SocketIO{
		OnConnect: h.OnConnect,
		OnDisconnect: h.OnDisconnect,
		OnEvent: events,
	}

  	err := socketio.ConnectToSocket(HUOBI_SOCKETIO_ADDRESS, HuobiSocket)
  	if err != nil {
    	fmt.Println(err)
    	return
  	}
  	
    log.Printf("%s Websocket client disconnected.", h.GetName())
}