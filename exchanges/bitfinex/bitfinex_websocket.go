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
	BITFINEX_WEBSOCKET                     = "wss://api.bitfinex.com/ws"
	BITFINEX_WEBSOCKET_VERSION             = "1.1"
	BITFINEX_WEBSOCKET_POSITION_SNAPSHOT   = "ps"
	BITFINEX_WEBSOCKET_POSITION_NEW        = "pn"
	BITFINEX_WEBSOCKET_POSITION_UPDATE     = "pu"
	BITFINEX_WEBSOCKET_POSITION_CLOSE      = "pc"
	BITFINEX_WEBSOCKET_WALLET_SNAPSHOT     = "ws"
	BITFINEX_WEBSOCKET_WALLET_UPDATE       = "wu"
	BITFINEX_WEBSOCKET_ORDER_SNAPSHOT      = "os"
	BITFINEX_WEBSOCKET_ORDER_NEW           = "on"
	BITFINEX_WEBSOCKET_ORDER_UPDATE        = "ou"
	BITFINEX_WEBSOCKET_ORDER_CANCEL        = "oc"
	BITFINEX_WEBSOCKET_TRADE_EXECUTED      = "te"
	BITFINEX_WEBSOCKET_HEARTBEAT           = "hb"
	BITFINEX_WEBSOCKET_ALERT_RESTARTING    = "20051"
	BITFINEX_WEBSOCKET_ALERT_REFRESHING    = "20060"
	BITFINEX_WEBSOCKET_ALERT_RESUME        = "20061"
	BITFINEX_WEBSOCKET_UNKNOWN_EVENT       = "10000"
	BITFINEX_WEBSOCKET_UNKNOWN_PAIR        = "10001"
	BITFINEX_WEBSOCKET_SUBSCRIPTION_FAILED = "10300"
	BITFINEX_WEBSOCKET_ALREADY_SUBSCRIBED  = "10301"
	BITFINEX_WEBSOCKET_UNKNOWN_CHANNEL     = "10302"
)

func (b *Bitfinex) WebsocketPingHandler() error {
	request := make(map[string]string)
	request["event"] = "ping"
	return b.WebsocketSend(request)
}

func (b *Bitfinex) WebsocketSend(data interface{}) error {
	json, err := common.JSONEncode(data)
	if err != nil {
		return err
	}

	err = b.WebsocketConn.WriteMessage(websocket.TextMessage, json)

	if err != nil {
		return err
	}
	return nil
}

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

func (b *Bitfinex) WebsocketSendAuth() error {
	request := make(map[string]interface{})
	payload := "AUTH" + strconv.FormatInt(time.Now().UnixNano(), 10)[:13]
	request["event"] = "auth"
	request["apiKey"] = b.APIKey
	request["authSig"] = common.HexEncodeToString(common.GetHMAC(common.HASH_SHA512_384, []byte(payload), []byte(b.APISecret)))
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
								if chanData[1].(string) == BITFINEX_WEBSOCKET_HEARTBEAT {
									continue
								}
							}
						}
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
							switch chanData[1].(string) {
							case BITFINEX_WEBSOCKET_POSITION_SNAPSHOT:
								positionSnapshot := []BitfinexWebsocketPosition{}
								data := chanData[2].([]interface{})
								for _, x := range data {
									y := x.([]interface{})
									positionSnapshot = append(positionSnapshot, BitfinexWebsocketPosition{Pair: y[0].(string), Status: y[1].(string), Amount: y[2].(float64), Price: y[3].(float64),
										MarginFunding: y[4].(float64), MarginFundingType: int(y[5].(float64))})
								}
								log.Println(positionSnapshot)
							case BITFINEX_WEBSOCKET_POSITION_NEW, BITFINEX_WEBSOCKET_POSITION_UPDATE, BITFINEX_WEBSOCKET_POSITION_CLOSE:
								data := chanData[2].([]interface{})
								position := BitfinexWebsocketPosition{Pair: data[0].(string), Status: data[1].(string), Amount: data[2].(float64), Price: data[3].(float64),
									MarginFunding: data[4].(float64), MarginFundingType: int(data[5].(float64))}
								log.Println(position)
							case BITFINEX_WEBSOCKET_WALLET_SNAPSHOT:
								data := chanData[2].([]interface{})
								walletSnapshot := []BitfinexWebsocketWallet{}
								for _, x := range data {
									y := x.([]interface{})
									walletSnapshot = append(walletSnapshot, BitfinexWebsocketWallet{Name: y[0].(string), Currency: y[1].(string), Balance: y[2].(float64), UnsettledInterest: y[3].(float64)})
								}
								log.Println(walletSnapshot)
							case BITFINEX_WEBSOCKET_WALLET_UPDATE:
								data := chanData[2].([]interface{})
								wallet := BitfinexWebsocketWallet{Name: data[0].(string), Currency: data[1].(string), Balance: data[2].(float64), UnsettledInterest: data[3].(float64)}
								log.Println(wallet)
							case BITFINEX_WEBSOCKET_ORDER_SNAPSHOT:
								orderSnapshot := []BitfinexWebsocketOrder{}
								data := chanData[2].([]interface{})
								for _, x := range data {
									y := x.([]interface{})
									orderSnapshot = append(orderSnapshot, BitfinexWebsocketOrder{OrderID: int64(y[0].(float64)), Pair: y[1].(string), Amount: y[2].(float64), OrigAmount: y[3].(float64),
										OrderType: y[4].(string), Status: y[5].(string), Price: y[6].(float64), PriceAvg: y[7].(float64), Timestamp: y[8].(string)})
								}
								log.Println(orderSnapshot)
							case BITFINEX_WEBSOCKET_ORDER_NEW, BITFINEX_WEBSOCKET_ORDER_UPDATE, BITFINEX_WEBSOCKET_ORDER_CANCEL:
								data := chanData[2].([]interface{})
								order := BitfinexWebsocketOrder{OrderID: int64(data[0].(float64)), Pair: data[1].(string), Amount: data[2].(float64), OrigAmount: data[3].(float64),
									OrderType: data[4].(string), Status: data[5].(string), Price: data[6].(float64), PriceAvg: data[7].(float64), Timestamp: data[8].(string), Notify: int(data[9].(float64))}
								log.Println(order)
							case BITFINEX_WEBSOCKET_TRADE_EXECUTED:
								data := chanData[2].([]interface{})
								trade := BitfinexWebsocketTradeExecuted{TradeID: int64(data[0].(float64)), Pair: data[1].(string), Timestamp: int64(data[2].(float64)), OrderID: int64(data[3].(float64)),
									AmountExecuted: data[4].(float64), PriceExecuted: data[5].(float64)}
								log.Println(trade)
							}
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
