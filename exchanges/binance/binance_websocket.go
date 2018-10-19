package binance

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/idoall/TokenExchangeCommon/commonutils"
	"github.com/idoall/gocryptotrader/common"
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

		fmt.Println(wsurl)
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

// WebsocketKline 获取 k 线
func (b *Binance) WebsocketKline(ch chan *KlineStream, timeIntervals []TimeInterval, symbolList []string, done <-chan struct{}) {

	for b.Enabled && b.Websocket {
		select {
		case <-done:
			return
		default:
			var Dialer websocket.Dialer
			var err error

			streamsArray := []string{}
			for _, tv := range timeIntervals {
				for _, sv := range symbolList {
					streamsArray = append(streamsArray, fmt.Sprintf("%s@kline_%s", strings.ToLower(sv), tv))
				}
			}

			streams := commonutils.JoinStrings(streamsArray, "/")

			wsurl := b.WebsocketURL + "/stream?streams=" + streams

			// b.WebsocketConn, _, err = Dialer.Dial(binanceDefaultWebsocketURL+myenabledPairs, http.Header{})
			b.WebsocketConn, _, err = Dialer.Dial(wsurl, http.Header{})
			if err != nil {
				log.Printf("%s Unable to connect to Websocket. Error: %s\n", b.Name, err)
				continue
			}

			if b.Verbose {
				log.Printf("%s Connected to Websocket.\n", b.Name)
				log.Printf("wsurl:%s\n", streams)
			}

			for b.Enabled && b.Websocket {
				select {
				case <-done:
					return
				default:
					_, resp, err := b.WebsocketConn.ReadMessage()
					if err != nil {
						log.Println(err)
						break
					}

					multiStreamData := MultiStreamData{}
					if err = common.JSONDecode(resp, &multiStreamData); err != nil {
						log.Println("Could not load multi stream data.", string(resp))
						continue
					}

					kline := KlineStream{}
					if err = commonutils.JSONDecode(multiStreamData.Data, &kline); err != nil {
						log.Println("Could not convert to a KlineStream structure")
						continue
					}

					ch <- &kline
				}
			}
			b.WebsocketConn.Close()
			log.Printf("%s Websocket client disconnected.", b.Name)
		}

	}
}

// WebsocketLastPrice 获取 最新价格
func (b *Binance) WebsocketLastPrice(chLastPrice chan float64, timeIntervals []TimeInterval, symbolList []string, done <-chan struct{}) {

	for b.Enabled && b.Websocket {
		select {
		case <-done:
			return
		default:
			var Dialer websocket.Dialer
			var err error

			streamsArray := []string{}
			for _, tv := range timeIntervals {
				for _, sv := range symbolList {
					streamsArray = append(streamsArray, fmt.Sprintf("%s@kline_%s", strings.ToLower(sv), tv))
				}
			}

			streams := commonutils.JoinStrings(streamsArray, "/")

			wsurl := b.WebsocketURL + "/stream?streams=" + streams

			// b.WebsocketConn, _, err = Dialer.Dial(binanceDefaultWebsocketURL+myenabledPairs, http.Header{})
			b.WebsocketConn, _, err = Dialer.Dial(wsurl, http.Header{})
			if err != nil {
				log.Printf("%s Unable to connect to Websocket. Error: %s\n", b.Name, err)
				continue
			}

			if b.Verbose {
				log.Printf("%s Connected to Websocket.\n", b.Name)
				log.Printf("wsurl:%s\n", streams)
			}

			for b.Enabled && b.Websocket {
				select {
				case <-done:
					return
				default:
					_, resp, err := b.WebsocketConn.ReadMessage()
					if err != nil {
						log.Println(err)
						break
					}

					multiStreamData := MultiStreamData{}
					if err = common.JSONDecode(resp, &multiStreamData); err != nil {
						log.Println("Could not load multi stream data.", string(resp))
						continue
					}

					kline := KlineStream{}
					if err = commonutils.JSONDecode(multiStreamData.Data, &kline); err != nil {
						log.Println("Could not convert to a KlineStream structure")
						continue
					}

					close, err := common.FloatFromString(kline.Kline.ClosePrice)
					if err != nil {
						log.Println("Could not convert to a ClosePrice float64")
						continue
					}
					chLastPrice <- close
				}
			}
			b.WebsocketConn.Close()
			log.Printf("%s Websocket client disconnected.", b.Name)
		}

	}
}
