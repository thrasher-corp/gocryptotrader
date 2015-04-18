package main

import (
	"log"
	"github.com/thrasher-/socketio"
)

const (
	BTCCHINA_SOCKETIO_ADDRESS = "https://websocket.btcchina.com"
)

func (b *BTCChina) OnConnect(output chan socketio.Message) {
	if b.Verbose {
		log.Printf("%s Connected to Websocket.", b.GetName())
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
	log.Println(string(message))
}

func (b *BTCChina) WebsocketClient() {
	events := make(map[string]func(message []byte, output chan socketio.Message))
	events["message"] = b.OnMessage

	HuobiSocket = &socketio.SocketIO{
		Version: 1,
		OnConnect: b.OnConnect,
		OnEvent: events,
		OnError: b.OnError,
		OnDisconnect: b.OnDisconnect,
	}

  	err := socketio.ConnectToSocket(BTCCHINA_SOCKETIO_ADDRESS, HuobiSocket)
  	if err != nil {
    	log.Println(err)
  	}
  	
    log.Printf("%s Websocket client disconnected.. Reconnecting.", b.GetName())
    b.WebsocketClient()
}