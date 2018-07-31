package binance

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-/gocryptotrader/common"
)

const (
	binanceDefaultWebsocketURL = "wss://stream.binance.com:9443"
	binancePingPeriod          = 20 * time.Second
)

// WebsocketClient starts and handles the websocket client connection
func (b *Binance) WebsocketClient() {
	for b.Enabled && b.Websocket {
		var Dialer websocket.Dialer
		var err error
		// myenabledPairs := strings.ToLower(strings.Replace(strings.Join(b.EnabledPairs, "@ticker/"), "-", "", -1)) + "@trade"

		myenabledPairsTicker := strings.ToLower(strings.Replace(strings.Join(b.EnabledPairs, "@ticker/"), "-", "", -1)) + "@ticker"
		myenabledPairsTrade := strings.ToLower(strings.Replace(strings.Join(b.EnabledPairs, "@trade/"), "-", "", -1)) + "@trade"
		myenabledPairsKline := strings.ToLower(strings.Replace(strings.Join(b.EnabledPairs, "@kline_1m/"), "-", "", -1)) + "@kline_1m"
		wsurl := b.WebsocketURL + "/stream?streams=" + myenabledPairsTicker + "/" + myenabledPairsTrade + "/" + myenabledPairsKline

		// b.WebsocketConn, _, err = Dialer.Dial(binanceDefaultWebsocketURL+myenabledPairs, http.Header{})
		b.WebsocketConn, _, err = Dialer.Dial(wsurl, http.Header{})

		if err != nil {
			log.Printf("%s Unable to connect to Websocket. Error: %s\n", b.Name, err)
			continue
		}

		if b.Verbose {
			log.Printf("%s Connected to Websocket.\n", b.Name)
		}

		for b.Enabled && b.Websocket {
			msgType, resp, err := b.WebsocketConn.ReadMessage()
			if err != nil {
				log.Println(err)
				break
			}

			switch msgType {
			case websocket.TextMessage:
				multiStreamData := MultiStreamData{}

				err := common.JSONDecode(resp, &multiStreamData)

				if err != nil {
					log.Println("Could not load multi stream data.", string(resp))
					continue
				}

				if strings.Contains(multiStreamData.Stream, "trade") {
					trade := TradeStream{}
					err := common.JSONDecode(multiStreamData.Data, &trade)

					if err != nil {
						log.Println("Could not convert to a TradeStream structure")
						continue
					}
					log.Println("Trade received", trade.Symbol, trade.TimeStamp, trade.TradeID, trade.Price, trade.Quantity)
				} else if strings.Contains(multiStreamData.Stream, "ticker") {
					ticker := TickerStream{}

					err := common.JSONDecode(multiStreamData.Data, &ticker)
					if err != nil {
						log.Println("Could not convert to a TickerStream structure")
						continue
					}

					log.Println("Ticker received", ticker.Symbol, ticker.EventTime, ticker.TotalTradedVolume, ticker.LastTradeID)
				} else if strings.Contains(multiStreamData.Stream, "kline") {
					kline := KlineStream{}

					err := common.JSONDecode(multiStreamData.Data, &kline)
					if err != nil {
						log.Println("Could not convert to a KlineStream structure")
						continue
					}

					log.Println("Kline received", kline.Symbol, kline.EventTime, kline.Kline.HighPrice, kline.Kline.LowPrice)
				}
				type MsgType struct {
					MessageType string `json:"messageType"`
				}
			}
		}
		b.WebsocketConn.Close()
		log.Printf("%s Websocket client disconnected.", b.Name)
	}
}
