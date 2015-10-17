package main

import (
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"reflect"
	"strconv"
	"time"
)

const (
	BITFINEX_WEBSOCKET         = "wss://api2.bitfinex.com:3000/ws"
	BITFINEX_WEBSOCKET_VERSION = "1.0"
)

type BitfinexWebsocketChanInfo struct {
	Channel string
	Pair    string
}

type BitfinexWebsocketBook struct {
	Price  float64
	Count  int
	Amount float64
}

type BitfinexWebsocketTrade struct {
	ID        int64
	Timestamp int64
	Price     float64
	Amount    float64
}

type BitfinexWebsocketTicker struct {
	Bid             float64
	BidSize         float64
	Ask             float64
	AskSize         float64
	DailyChange     float64
	DialyChangePerc float64
	LastPrice       float64
	Volume          float64
}

func (b *Bitfinex) WebsocketPingHandler() error {
	request := make(map[string]string)
	request["event"] = "ping"
	return b.WebsocketSend(request)
}

func (b *Bitfinex) WebsocketSend(data interface{}) error {
	json, err := JSONEncode(data)
	if err != nil {
		return err
	}

	err = b.WebsocketConn.WriteMessage(websocket.TextMessage, json)

	if err != nil {
		return err
	}
	return nil
}

func (b *Bitfinex) WebsocketSubscribe(channel string, params map[string]string) {
	request := make(map[string]string)
	request["event"] = "subscribe"
	request["channel"] = channel

	if len(params) > 0 {
		for k, v := range params {
			request[k] = v
		}
	}

	b.WebsocketSend(request)
}

func (b *Bitfinex) WebsocketSendAuth() error {
	request := make(map[string]interface{})
	payload := "AUTH" + strconv.FormatInt(time.Now().UnixNano(), 10)[:13]
	request["event"] = "auth"
	request["apiKey"] = b.APIKey
	request["authSig"] = HexEncodeToString(GetHMAC(HASH_SHA512_384, []byte(payload), []byte(b.APISecret)))
	request["authPayload"] = payload
	return b.WebsocketSend(request)
}

func (b *Bitfinex) WebsocketSendUnauth() error {
	request := make(map[string]string)
	request["event"] = "unauth"
	return b.WebsocketSend(request)
}

func (b *Bitfinex) WebsocketAddSubscriptionChannel(chanID int, channel, pair string) {
	chanInfo := BitfinexWebsocketChanInfo{Pair: pair, Channel: channel}
	b.WebsocketSubdChannels[chanID] = chanInfo

	if b.Verbose {
		log.Printf("%s Subscribed to Channel: %s Pair: %s ChannelID: %d\n", b.GetName(), channel, pair, chanID)
	}
}

func (b *Bitfinex) WebsocketClient() {
	channels := []string{"book", "trades", "ticker"}
	for b.Enabled && b.Websocket {
		var Dialer websocket.Dialer
		var err error
		b.WebsocketConn, _, err = Dialer.Dial(BITFINEX_WEBSOCKET, http.Header{})

		if err != nil {
			log.Printf("%s Unable to connect to Websocket. Error: %s\n", b.GetName(), err)
			continue
		}

		msgType, resp, err := b.WebsocketConn.ReadMessage()
		if msgType != websocket.TextMessage {
			continue
		}

		type WebsocketHandshake struct {
			Event   string `json:"event"`
			Code    int64  `json:"code"`
			Version int    `json:"version"`
		}

		hs := WebsocketHandshake{}
		err = JSONDecode(resp, &hs)
		if err != nil {
			log.Println(err)
			continue
		}

		if hs.Event == "info" {
			if b.Verbose {
				log.Printf("%s Connected to Websocket.\n", b.GetName())
			}
		}

		for _, x := range channels {
			for _, y := range b.EnabledPairs {
				params := make(map[string]string)
				if x == "book" {
					params["prec"] = "P0"
				}
				params["pair"] = y
				b.WebsocketSubscribe(x, params)
			}
		}

		if b.AuthenticatedAPISupport {
			err = b.WebsocketSendAuth()
			if err != nil {
				log.Println(err)
			}
		}

		for b.Enabled && b.Websocket {
			msgType, resp, err := b.WebsocketConn.ReadMessage()
			if err != nil {
				log.Println(err)
				break
			}

			switch msgType {
			case websocket.TextMessage:
				var result interface{}
				err := JSONDecode(resp, &result)
				if err != nil {
					log.Println(err)
					continue
				}

				switch reflect.TypeOf(result).String() {
				case "map[string]interface {}":
					eventData := result.(map[string]interface{})
					event := eventData["event"]

					switch event {
					case "subscribed":
						b.WebsocketAddSubscriptionChannel(int(eventData["chanId"].(float64)), eventData["channel"].(string), eventData["pair"].(string))
					case "auth":
						status := eventData["status"].(string)

						if status == "OK" {
							b.WebsocketAddSubscriptionChannel(0, "account", "N/A")
						} else if status == "fail" {
							log.Printf("%s Websocket unable to AUTH. Error code: %s\n", b.GetName(), eventData["code"].(string))
							b.AuthenticatedAPISupport = false
						}
					}
				case "[]interface {}":
					chanData := result.([]interface{})
					chanID := int(chanData[0].(float64))
					chanInfo, ok := b.WebsocketSubdChannels[chanID]

					if !ok {
						log.Println("Unable to locate chanID: %d", chanID)
					} else {
						switch chanInfo.Channel {
						case "book":
							orderbook := []BitfinexWebsocketBook{}
							switch len(chanData) {
							case 2:
								data := chanData[1].([]interface{})
								for _, x := range data {
									y := x.([]interface{})
									orderbook = append(orderbook, BitfinexWebsocketBook{Price: y[0].(float64), Count: int(y[1].(float64)), Amount: y[2].(float64)})
								}
							case 4:
								orderbook = append(orderbook, BitfinexWebsocketBook{Price: chanData[1].(float64), Count: int(chanData[2].(float64)), Amount: chanData[3].(float64)})
							}
						case "ticker":
							ticker := BitfinexWebsocketTicker{Bid: chanData[1].(float64), BidSize: chanData[2].(float64), Ask: chanData[3].(float64), AskSize: chanData[4].(float64),
								DailyChange: chanData[5].(float64), DialyChangePerc: chanData[6].(float64), LastPrice: chanData[7].(float64), Volume: chanData[8].(float64)}

							log.Printf("Bitfinex %s Websocket Last %f Volume %f\n", chanInfo.Pair, ticker.LastPrice, ticker.Volume)
						case "account":
							//to-do
						case "trades":
							trades := []BitfinexWebsocketTrade{}
							switch len(chanData) {
							case 2:
								data := chanData[1].([]interface{})
								for _, x := range data {
									y := x.([]interface{})
									trades = append(trades, BitfinexWebsocketTrade{ID: int64(y[0].(float64)), Timestamp: int64(y[1].(float64)), Price: y[2].(float64), Amount: y[3].(float64)})
								}
							case 5:
								trade := BitfinexWebsocketTrade{ID: int64(chanData[1].(float64)), Timestamp: int64(chanData[2].(float64)), Price: chanData[3].(float64), Amount: chanData[4].(float64)}
								trades = append(trades, trade)

								if b.Verbose {
									log.Printf("Bitfinex %s Websocket Trade ID %d Timestamp %d Price %f Amount %f\n", chanInfo.Pair, trade.ID, trade.Timestamp, trade.Price, trade.Amount)
								}
							}
						}
					}
				}
			}
		}
		b.WebsocketConn.Close()
		log.Printf("%s Websocket client disconnected.\n", b.GetName())
	}
}
