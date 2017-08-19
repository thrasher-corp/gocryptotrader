package bitfinex

import (
	"log"
	"net/http"
	"reflect"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-/gocryptotrader/common"
)

const (
	bitfinexWebsocket                   = "wss://api.bitfinex.com/ws"
	bitfinexWebsocketVersion            = "1.1"
	bitfinexWebsocketPositionSnapshot   = "ps"
	bitfinexWebsocketPositionNew        = "pn"
	bitfinexWebsocketPositionUpdate     = "pu"
	bitfinexWebsocketPositionClose      = "pc"
	bitfinexWebsocketWalletSnapshot     = "ws"
	bitfinexWebsocketWalletUpdate       = "wu"
	bitfinexWebsocketOrderSnapshot      = "os"
	bitfinexWebsocketOrderNew           = "on"
	bitfinexWebsocketOrderUpdate        = "ou"
	bitfinexWebsocketOrderCancel        = "oc"
	bitfinexWebsocketTradeExecuted      = "te"
	bitfinexWebsocketHeartbeat          = "hb"
	bitfinexWebsocketAlertRestarting    = "20051"
	bitfinexWebsocketAlertRefreshing    = "20060"
	bitfinexWebsocketAlertResume        = "20061"
	bitfinexWebsocketUnknownEvent       = "10000"
	bitfinexWebsocketUnknownPair        = "10001"
	bitfinexWebsocketSubscriptionFailed = "10300"
	bitfinexWebsocketAlreadySubscribed  = "10301"
	bitfinexWebsocketUnknownChannel     = "10302"
)

// WebsocketPingHandler sends a ping request to the websocket server
func (b *Bitfinex) WebsocketPingHandler() error {
	request := make(map[string]string)
	request["event"] = "ping"

	return b.WebsocketSend(request)
}

// WebsocketSend sends data to the websocket server
func (b *Bitfinex) WebsocketSend(data interface{}) error {
	json, err := common.JSONEncode(data)
	if err != nil {
		return err
	}

	return b.WebsocketConn.WriteMessage(websocket.TextMessage, json)
}

// WebsocketSubscribe subscribes to the websocket channel
func (b *Bitfinex) WebsocketSubscribe(channel string, params map[string]string) error {
	request := make(map[string]string)
	request["event"] = "subscribe"
	request["channel"] = channel

	if len(params) > 0 {
		for k, v := range params {
			request[k] = v
		}
	}
	return b.WebsocketSend(request)
}

// WebsocketSendAuth sends a autheticated event payload
func (b *Bitfinex) WebsocketSendAuth() error {
	request := make(map[string]interface{})
	payload := "AUTH" + strconv.FormatInt(time.Now().UnixNano(), 10)[:13]
	request["event"] = "auth"
	request["apiKey"] = b.APIKey
	request["authSig"] = common.HexEncodeToString(common.GetHMAC(common.HashSHA512_384, []byte(payload), []byte(b.APISecret)))
	request["authPayload"] = payload

	return b.WebsocketSend(request)
}

// WebsocketSendUnauth sends an unauthenticated payload
func (b *Bitfinex) WebsocketSendUnauth() error {
	request := make(map[string]string)
	request["event"] = "unauth"

	return b.WebsocketSend(request)
}

// WebsocketAddSubscriptionChannel adds a new subscription channel to the
// WebsocketSubdChannels map in bitfinex.go (Bitfinex struct)
func (b *Bitfinex) WebsocketAddSubscriptionChannel(chanID int, channel, pair string) {
	chanInfo := WebsocketChanInfo{Pair: pair, Channel: channel}
	b.WebsocketSubdChannels[chanID] = chanInfo

	if b.Verbose {
		log.Printf("%s Subscribed to Channel: %s Pair: %s ChannelID: %d\n", b.GetName(), channel, pair, chanID)
	}
}

// WebsocketClient makes a connection with the websocket server
func (b *Bitfinex) WebsocketClient() {
	channels := []string{"book", "trades", "ticker"}
	for b.Enabled && b.Websocket {
		var Dialer websocket.Dialer
		var err error
		b.WebsocketConn, _, err = Dialer.Dial(bitfinexWebsocket, http.Header{})

		if err != nil {
			log.Printf("%s Unable to connect to Websocket. Error: %s\n", b.GetName(), err)
			continue
		}

		msgType, resp, err := b.WebsocketConn.ReadMessage()
		if err != nil {
			log.Printf("%s Unable to read from Websocket. Error: %s\n", b.GetName(), err)
			continue
		}
		if msgType != websocket.TextMessage {
			continue
		}

		type WebsocketHandshake struct {
			Event   string  `json:"event"`
			Code    int64   `json:"code"`
			Version float64 `json:"version"`
		}

		hs := WebsocketHandshake{}
		err = common.JSONDecode(resp, &hs)
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
				err := common.JSONDecode(resp, &result)
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
						log.Printf("Unable to locate chanID: %d\n", chanID)
					} else {
						if len(chanData) == 2 {
							if reflect.TypeOf(chanData[1]).String() == "string" {
								if chanData[1].(string) == bitfinexWebsocketHeartbeat {
									continue
								}
							}
						}
						switch chanInfo.Channel {
						case "book":
							orderbook := []WebsocketBook{}
							switch len(chanData) {
							case 2:
								data := chanData[1].([]interface{})
								for _, x := range data {
									y := x.([]interface{})
									orderbook = append(orderbook, WebsocketBook{Price: y[0].(float64), Count: int(y[1].(float64)), Amount: y[2].(float64)})
								}
							case 4:
								orderbook = append(orderbook, WebsocketBook{Price: chanData[1].(float64), Count: int(chanData[2].(float64)), Amount: chanData[3].(float64)})
							}
							log.Println(orderbook)
						case "ticker":
							ticker := WebsocketTicker{Bid: chanData[1].(float64), BidSize: chanData[2].(float64), Ask: chanData[3].(float64), AskSize: chanData[4].(float64),
								DailyChange: chanData[5].(float64), DialyChangePerc: chanData[6].(float64), LastPrice: chanData[7].(float64), Volume: chanData[8].(float64)}

							log.Printf("Bitfinex %s Websocket Last %f Volume %f\n", chanInfo.Pair, ticker.LastPrice, ticker.Volume)
						case "account":
							switch chanData[1].(string) {
							case bitfinexWebsocketPositionSnapshot:
								positionSnapshot := []WebsocketPosition{}
								data := chanData[2].([]interface{})
								for _, x := range data {
									y := x.([]interface{})
									positionSnapshot = append(positionSnapshot, WebsocketPosition{Pair: y[0].(string), Status: y[1].(string), Amount: y[2].(float64), Price: y[3].(float64),
										MarginFunding: y[4].(float64), MarginFundingType: int(y[5].(float64))})
								}
								log.Println(positionSnapshot)
							case bitfinexWebsocketPositionNew, bitfinexWebsocketPositionUpdate, bitfinexWebsocketPositionClose:
								data := chanData[2].([]interface{})
								position := WebsocketPosition{Pair: data[0].(string), Status: data[1].(string), Amount: data[2].(float64), Price: data[3].(float64),
									MarginFunding: data[4].(float64), MarginFundingType: int(data[5].(float64))}
								log.Println(position)
							case bitfinexWebsocketWalletSnapshot:
								data := chanData[2].([]interface{})
								walletSnapshot := []WebsocketWallet{}
								for _, x := range data {
									y := x.([]interface{})
									walletSnapshot = append(walletSnapshot, WebsocketWallet{Name: y[0].(string), Currency: y[1].(string), Balance: y[2].(float64), UnsettledInterest: y[3].(float64)})
								}
								log.Println(walletSnapshot)
							case bitfinexWebsocketWalletUpdate:
								data := chanData[2].([]interface{})
								wallet := WebsocketWallet{Name: data[0].(string), Currency: data[1].(string), Balance: data[2].(float64), UnsettledInterest: data[3].(float64)}
								log.Println(wallet)
							case bitfinexWebsocketOrderSnapshot:
								orderSnapshot := []WebsocketOrder{}
								data := chanData[2].([]interface{})
								for _, x := range data {
									y := x.([]interface{})
									orderSnapshot = append(orderSnapshot, WebsocketOrder{OrderID: int64(y[0].(float64)), Pair: y[1].(string), Amount: y[2].(float64), OrigAmount: y[3].(float64),
										OrderType: y[4].(string), Status: y[5].(string), Price: y[6].(float64), PriceAvg: y[7].(float64), Timestamp: y[8].(string)})
								}
								log.Println(orderSnapshot)
							case bitfinexWebsocketOrderNew, bitfinexWebsocketOrderUpdate, bitfinexWebsocketOrderCancel:
								data := chanData[2].([]interface{})
								order := WebsocketOrder{OrderID: int64(data[0].(float64)), Pair: data[1].(string), Amount: data[2].(float64), OrigAmount: data[3].(float64),
									OrderType: data[4].(string), Status: data[5].(string), Price: data[6].(float64), PriceAvg: data[7].(float64), Timestamp: data[8].(string), Notify: int(data[9].(float64))}
								log.Println(order)
							case bitfinexWebsocketTradeExecuted:
								data := chanData[2].([]interface{})
								trade := WebsocketTradeExecuted{TradeID: int64(data[0].(float64)), Pair: data[1].(string), Timestamp: int64(data[2].(float64)), OrderID: int64(data[3].(float64)),
									AmountExecuted: data[4].(float64), PriceExecuted: data[5].(float64)}
								log.Println(trade)
							}
						case "trades":
							trades := []WebsocketTrade{}
							switch len(chanData) {
							case 2:
								data := chanData[1].([]interface{})
								for _, x := range data {
									y := x.([]interface{})
									trades = append(trades, WebsocketTrade{ID: int64(y[0].(float64)), Timestamp: int64(y[1].(float64)), Price: y[2].(float64), Amount: y[3].(float64)})
								}
							case 5:
								trade := WebsocketTrade{ID: int64(chanData[1].(float64)), Timestamp: int64(chanData[2].(float64)), Price: chanData[3].(float64), Amount: chanData[4].(float64)}
								trades = append(trades, trade)

								if b.Verbose {
									log.Printf("Bitfinex %s Websocket Trade ID %d Timestamp %d Price %f Amount %f\n", chanInfo.Pair, trade.ID, trade.Timestamp, trade.Price, trade.Amount)
								}
							}
							log.Println(trades)
						}
					}
				}
			}
		}
		b.WebsocketConn.Close()
		log.Printf("%s Websocket client disconnected.\n", b.GetName())
	}
}
