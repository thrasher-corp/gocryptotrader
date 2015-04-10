package main

import (
	"fmt"
	"log"
	"github.com/thrasher-/socketio"
)

const (
	HUOBI_SOCKETIO_ADDRESS = "https://hq.huobi.com:443"
	HUOBI_SOCKET_SYMBOL_LIST = "reqSymbolList"
	HUOBI_SOCKET_SYMBOL_DETAIL = "reqSymbolDetail"
	HUOBI_SOCKET_SUBSCRIBE = "reqMsgSubscribe"
	HUOBI_SOCKET_UNSUBSCRIBE = "reqMsgUnsubscribe"
	HUOBI_SOCKET_TIMELINE = "reqTimeLine"
	HUOBI_SOCKET_KLINE = "reqKLine"
	HUOBI_SOCKET_DEPTH = "reqMarketDepth"
	HUOBI_SOCKET_DEPTH_TOP = "reqMarketDepthTop"
	HUOBI_SOCKET_TRADE_DETAIL_TOP = "reqTradeDetailTop"
	HUOBI_SOCKET_MARKET_DETAIL = "reqMarketDetail"
)

var HuobiSocket *socketio.SocketIO

type HuobiRequest struct {
	Version int `json:"version"`
	MsgType string `json:"msgType"`
}

type HuobiResponse struct {
	Version int `json:"version"`
	MsgType string `json:"msgType"`
	RequestIndex int64 `json:"requestIndex"`
	RetCode int64 `json:"retCode"`
	RetMessage string `json:"retMsg"`
	Payload map[string]interface{} `json:"payload"`
}

func (h *HUOBI) OnConnect(output chan socketio.Message) {
	if h.Verbose {
		log.Printf("%s Connected to Websocket.", h.GetName())
	}
	msg := HuobiRequest{1, HUOBI_SOCKET_SYMBOL_LIST}
	result, err := JSONEncode(msg)
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
	err := JSONDecode(message, &response)
	if err != nil {
		log.Println(err)
	}
	log.Println(response)
}

func (h *HUOBI) WebsocketClient() {
	events := make(map[string]func(message []byte, output chan socketio.Message))
	events["request"] = h.OnRequest

	HuobiSocket = &socketio.SocketIO{
		OnConnect: h.OnConnect,
		OnDisconnect: h.OnDisconnect,
		OnMessage: h.OnMessage,
		OnEvent: events,
	}

  	err := socketio.ConnectToSocket(HUOBI_SOCKETIO_ADDRESS, HuobiSocket)
  	if err != nil {
    	fmt.Println(err)
    	return
  	}
  	
    log.Printf("%s Websocket client disconnected.", h.GetName())
}