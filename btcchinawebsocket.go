package main

import (
	"log"
	"fmt"
	"github.com/thrasher-/socketio"
)

const (
	BTCCHINA_SOCKETIO_ADDRESS = "https://websocket.btcchina.com"
)

type BTCChinaWebsocketOrder struct {
	Price float64 `json:"price"`
	TotalAmount float64 `json:"totalamount"`
	Type string `json:"type"`
}

type BTCChinaWebsocketGroupOrder struct {
	Asks []BTCChinaWebsocketOrder `json:"ask"`
	Bids []BTCChinaWebsocketOrder `json:"bid"`
	Market string `json:"market"`
}

type BTCChinaWebsocketTrade struct {
	Amount float64 `json:"amount,string"`
	Date float64 `json:"date"`
	Market string `json:"market"`
	Price float64 `json:"price,string"`
	TradeID float64 `json:"trade_id"`
	Type string `json:"type"`
}

type BTCChinaWebsocketTicker struct {
	Buy float64 `json:"buy"`
	Date float64 `json:"date"`
	High float64 `json:"high"`
	Last float64 `json:"last"`
	Low float64 `json:"low"`
	Market string`json:"market"`
	Open float64 `json:"open"`
	PrevClose float64 `json:"prev_close"`
	Sell float64 `json:"sell"`
	Volume float64 `json:"vol"`
	Vwap float64 `json:"vwap"`
}

var BTCChinaSocket *socketio.SocketIO

func (b *BTCChina) OnConnect(output chan socketio.Message) {
	if b.Verbose {
		log.Printf("%s Connected to Websocket.", b.GetName())
	}
	currencies := []string{"cnybtc", "cnyltc", "btcltc"}
	endpoints := []string{"marketdata", "grouporder"}

	for _, x := range endpoints {
		for _, y := range currencies {
			channel := fmt.Sprintf(`"%s_%s"`, x, y)
			if b.Verbose {
				log.Printf("%s Websocket subscribing to channel: %s.", b.GetName(), channel)
			}
			output <- socketio.CreateMessageEvent("subscribe", channel, b.OnMessage, BTCChinaSocket.Version)
		}
	}
}

func (b *BTCChina) OnDisconnect(output chan socketio.Message) {
	log.Printf("%s Disconnected from websocket server.. Reconnecting.\n", b.GetName())
	b.WebsocketClient()
}

func (b *BTCChina) OnError() {
	log.Printf("%s Error with Websocket connection.. Reconnecting.\n", b.GetName())
	b.WebsocketClient()
}

func (b *BTCChina) OnMessage(message []byte, output chan socketio.Message) {
	if b.Verbose {
		log.Println("onmsg")
		log.Println(string(message))
	}
}

func (b *BTCChina) OnTicker(message []byte, output chan socketio.Message) {
	type Response struct {
		Ticker BTCChinaWebsocketTicker `json:"ticker"`
	}
	var resp Response
	err := JSONDecode(message, &resp)

	if err != nil {
		log.Println(err)
		return
	}
}

func (b *BTCChina) OnGroupOrder(message []byte, output chan socketio.Message) {
	type Response struct {
		GroupOrder BTCChinaWebsocketGroupOrder `json:"grouporder"`
	}
	var resp Response
	err := JSONDecode(message, &resp)

	if err != nil {
		log.Println(err)
		return
	}
}

func (b *BTCChina) OnTrade(message []byte, output chan socketio.Message) {
	trade := BTCChinaWebsocketTrade{}
	err := JSONDecode(message, &trade)

	if err != nil {
		log.Println(err)
		return
	}
}

func (b *BTCChina) WebsocketClient() {
	events := make(map[string]func(message []byte, output chan socketio.Message))
	events["grouporder"] = b.OnGroupOrder
	events["ticker"] = b.OnTicker
	events["trade"] = b.OnTrade

	BTCChinaSocket = &socketio.SocketIO{
		Version: 1,
		OnConnect: b.OnConnect,
		OnEvent: events,
		OnError: b.OnError,
		OnMessage: b.OnMessage,
		OnDisconnect: b.OnDisconnect,
	}

  	err := socketio.ConnectToSocket(BTCCHINA_SOCKETIO_ADDRESS, BTCChinaSocket)
  	if err != nil {
    	log.Println(err)
  	}
  	
    log.Printf("%s Websocket client disconnected.. Reconnecting.", b.GetName())
    b.WebsocketClient()
}