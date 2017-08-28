package btcc

import (
	"fmt"
	"log"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/socketio"
)

const (
	BTCC_SOCKETIO_ADDRESS = "https://websocket.btcc.com"
)

var BTCCSocket *socketio.SocketIO

func (b *BTCC) OnConnect(output chan socketio.Message) {
	if b.Verbose {
		log.Printf("%s Connected to Websocket.", b.GetName())
	}

	currencies := []string{}
	for _, x := range b.EnabledPairs {
		currency := common.StringToLower(x[3:] + x[0:3])
		currencies = append(currencies, currency)
	}
	endpoints := []string{"marketdata", "grouporder"}

	for _, x := range endpoints {
		for _, y := range currencies {
			channel := fmt.Sprintf(`"%s_%s"`, x, y)
			if b.Verbose {
				log.Printf("%s Websocket subscribing to channel: %s.", b.GetName(), channel)
			}
			output <- socketio.CreateMessageEvent("subscribe", channel, b.OnMessage, BTCCSocket.Version)
		}
	}
}

func (b *BTCC) OnDisconnect(output chan socketio.Message) {
	log.Printf("%s Disconnected from websocket server.. Reconnecting.\n", b.GetName())
	b.WebsocketClient()
}

func (b *BTCC) OnError() {
	log.Printf("%s Error with Websocket connection.. Reconnecting.\n", b.GetName())
	b.WebsocketClient()
}

func (b *BTCC) OnMessage(message []byte, output chan socketio.Message) {
	if b.Verbose {
		log.Printf("%s Websocket message received which isn't handled by default.\n", b.GetName())
		log.Println(string(message))
	}
}

func (b *BTCC) OnTicker(message []byte, output chan socketio.Message) {
	type Response struct {
		Ticker WebsocketTicker `json:"ticker"`
	}
	var resp Response
	err := common.JSONDecode(message, &resp)

	if err != nil {
		log.Println(err)
		return
	}
}

func (b *BTCC) OnGroupOrder(message []byte, output chan socketio.Message) {
	type Response struct {
		GroupOrder WebsocketGroupOrder `json:"grouporder"`
	}
	var resp Response
	err := common.JSONDecode(message, &resp)

	if err != nil {
		log.Println(err)
		return
	}
}

func (b *BTCC) OnTrade(message []byte, output chan socketio.Message) {
	trade := WebsocketTrade{}
	err := common.JSONDecode(message, &trade)

	if err != nil {
		log.Println(err)
		return
	}
}

func (b *BTCC) WebsocketClient() {
	events := make(map[string]func(message []byte, output chan socketio.Message))
	events["grouporder"] = b.OnGroupOrder
	events["ticker"] = b.OnTicker
	events["trade"] = b.OnTrade

	BTCCSocket = &socketio.SocketIO{
		Version:      1,
		OnConnect:    b.OnConnect,
		OnEvent:      events,
		OnError:      b.OnError,
		OnMessage:    b.OnMessage,
		OnDisconnect: b.OnDisconnect,
	}

	for b.Enabled && b.Websocket {
		err := socketio.ConnectToSocket(BTCC_SOCKETIO_ADDRESS, BTCCSocket)
		if err != nil {
			log.Printf("%s Unable to connect to Websocket. Err: %s\n", b.GetName(), err)
			continue
		}
		log.Printf("%s Disconnected from Websocket.\n", b.GetName())
	}
}
