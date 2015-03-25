package main 

import (
	"log"
	"net/http"
	"time"
	"fmt"
	"github.com/gorilla/websocket"
)

type OKCoinWebsocketEvent struct {
	Event string `json:"event"`
	Channel string `json:"channel"`
}

type OKCoinWebsocketResponse struct {
	Channel string `json:"channel"`
	Data interface{} `json:"data"`
}

var okConn websocket.Conn

func (o *OKCoin) PingHandler(message string) (error) {
	err := okConn.WriteControl(websocket.PingMessage, []byte("{'event':'ping'}"), time.Now().Add(time.Second))

	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}	

func (o *OKCoin) AddChannel(conn *websocket.Conn, channel string) {
	event := OKCoinWebsocketEvent{"addChannel", channel}
	json, err := JSONEncode(event)
	if err != nil {
		log.Println(err)
		return
	}
	err = conn.WriteMessage(websocket.TextMessage, json)

	if err != nil {
		log.Println(err)
		return
	}
}

func (o* OKCoin) RemoveChannel(conn *websocket.Conn, channel string) {
	event := OKCoinWebsocketEvent{"removeChannel", channel}
	json, err := JSONEncode(event)
	if err != nil {
		log.Println(err)
		return
	}
	err = conn.WriteMessage(websocket.TextMessage, json)

	if err != nil {
		log.Println(err)
		return
	}
}

func (o *OKCoin) WebsocketClient(currencies []string) {
	if len(currencies) == 0 {
		log.Println("No currencies for Websocket client specified.")
		return
	}

	var Dialer websocket.Dialer
	okConn, resp, err := Dialer.Dial(o.WebsocketURL, http.Header{})

	if err != nil {
		log.Println(err)
		return
	}

	
	if o.Verbose {
		log.Printf("%s Connected to Websocket.", o.GetName())
		log.Println(resp)
	}

	okConn.SetPingHandler(o.PingHandler)

	for _, x := range currencies {
		o.AddChannel(okConn, fmt.Sprintf("ok_%s_ticker", x))
		o.AddChannel(okConn, fmt.Sprintf("ok_%s_depth", x))
		o.AddChannel(okConn, fmt.Sprintf("ok_%s_trades", x))
	}

	for {
		msgType, resp, err := okConn.ReadMessage()
		if err != nil {
			log.Println(err)
			break
		}
		switch msgType {
		case websocket.TextMessage:
			log.Println("\n" + string(resp))
		}
	}
	okConn.Close()
	log.Printf("%s Websocket client disconnected.", o.GetName())
}