package main 

import (
	"log"
	"net/http"
	"time"
	"fmt"
	"strings"
	"github.com/gorilla/websocket"
)

type OKCoinWebsocketTicker struct {
	Timestamp int64 `json:"timestamp,string"`
	Vol string `json:"vol"`
	Buy float64 `json:"buy,string"`
	High float64 `json:"high,string"`
	Last float64 `json:"last,string"`
	Low float64 `json:"low,string"`
	Sell float64 `json:"sell,string"`
}

type OKCoinWebsocketOrderbook struct {
	Asks [][]float64 `json:"asks"`
	Bids [][]float64 `json:"bids"`
	Timestamp int64 `json:"timestamp,string"`
}

type OKCoinWebsocketEvent struct {
	Event string `json:"event"`
	Channel string `json:"channel"`
}

type OKCoinWebsocketResponse struct {
	Channel string `json:"channel"`
	Data interface{} `json:"data"`
}

type OKCoinWebsocketParams struct {
	Partner string `json:"partner"`
}

type OKCoinWebsocketAuthParams struct {
	Partner string `json:"partner"`
	SecretKey string `json:"secretkey"`
}

type OKCoinWebsocketEventAuth struct {
	Event string `json:"event"`
	Channel string `json:"channel"`
	Parameters OKCoinWebsocketAuthParams `json:"parameters"`
}

type OKCoinWebsocketEventAuthRemove struct {
	Event string `json:"event"`
	Channel string `json:"channel"`
	Parameters OKCoinWebsocketParams `json:"parameters"`
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

func (o *OKCoin) AddChannelAuthenticated(conn *websocket.Conn, channel string) {
	event := OKCoinWebsocketEventAuth{"addChannel", channel, OKCoinWebsocketAuthParams{o.PartnerID, o.SecretKey}}
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

func (o *OKCoin) RemoveChannelAuthenticated(conn *websocket.Conn, channel string) {
	event := OKCoinWebsocketEventAuthRemove{"removeChannel", channel, OKCoinWebsocketParams{o.PartnerID}}
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

	currencyChan := ""
	if o.WebsocketURL == OKCOIN_WEBSOCKET_URL_CHINA {
		currencyChan = "ok_cny_realtrades"
	} else {
		currencyChan = "ok_usd_realtrades"
	}

	o.AddChannelAuthenticated(okConn, currencyChan)
	klineValues := []string{"1min", "3min", "5min", "15min", "30min", "1hour", "2hour", "4hour", "6hour", "12hour", "day", "3day", "week"}
	for _, x := range currencies {
		o.AddChannel(okConn, fmt.Sprintf("ok_%s_ticker", x))
		o.AddChannel(okConn, fmt.Sprintf("ok_%s_depth60", x))
		o.AddChannel(okConn, fmt.Sprintf("ok_%s_trades", x))

		for _, y := range klineValues {
			o.AddChannel(okConn, fmt.Sprintf("ok_%s_kline_%s", x, y))
		}
	}

	for {
		msgType, resp, err := okConn.ReadMessage()
		if err != nil {
			log.Println(err)
			break
		}
		switch msgType {
		case websocket.TextMessage:
			response := []interface{}{}
			err = JSONDecode(resp, &response)

			if err != nil {
				log.Println(err)
				break
			}

			for _, y := range response {
				z := y.(map[string]interface{})
				channel := z["channel"]
				data := z["data"]
				channelStr, ok := channel.(string)
				
				if !ok {
					log.Println("Unable to convert channel to string")
					continue
				}

				dataJSON, err := JSONEncode(data)

				if err != nil {
					log.Println(err)
					continue
				}

				switch true {
				case strings.Contains(channelStr, "ticker"): 
					ticker := OKCoinWebsocketTicker{}
					err = JSONDecode(dataJSON, &ticker)

					if err != nil {
						log.Println(err)
						continue
					}
				case strings.Contains(channelStr, "depth60"): 
					orderbook := OKCoinWebsocketOrderbook{}
					err = JSONDecode(dataJSON, &orderbook)

					if err != nil {
						log.Println(err)
						continue
					}
				case strings.Contains(channelStr, "trades"): 
					type TradeResponse struct {
						Data [][]string
					}

					trades := TradeResponse{}
					err = JSONDecode(dataJSON, &trades.Data)

					if err != nil {
						log.Println(err)
						continue
					}
					// to-do: convert from string array to trade struct
				case strings.Contains(channelStr, "kline"): 
					// to-do
				}
			}
		}
	}
	okConn.Close()
	log.Printf("%s Websocket client disconnected.", o.GetName())
}