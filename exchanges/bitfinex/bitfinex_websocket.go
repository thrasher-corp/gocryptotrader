package bitfinex

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
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

// WebsocketHandshake defines the communication between the websocket API for
// initial connection
type WebsocketHandshake struct {
	Event   string  `json:"event"`
	Code    int64   `json:"code"`
	Version float64 `json:"version"`
}

var pongReceive chan struct{}

// WsPingHandler sends a ping request to the websocket server
func (b *Bitfinex) WsPingHandler() error {
	request := make(map[string]string)
	request["event"] = "ping"

	return b.WsSend(request)
}

// WsSend sends data to the websocket server
func (b *Bitfinex) WsSend(data interface{}) error {
	json, err := common.JSONEncode(data)
	if err != nil {
		return err
	}
	return b.WebsocketConn.WriteMessage(websocket.TextMessage, json)
}

// WsSubscribe subscribes to the websocket channel
func (b *Bitfinex) WsSubscribe(channel string, params map[string]string) error {
	request := make(map[string]string)
	request["event"] = "subscribe"
	request["channel"] = channel

	if len(params) > 0 {
		for k, v := range params {
			request[k] = v
		}
	}
	return b.WsSend(request)
}

// WsSendAuth sends a autheticated event payload
func (b *Bitfinex) WsSendAuth() error {
	request := make(map[string]interface{})
	payload := "AUTH" + strconv.FormatInt(time.Now().UnixNano(), 10)[:13]
	request["event"] = "auth"
	request["apiKey"] = b.APIKey

	request["authSig"] = common.HexEncodeToString(
		common.GetHMAC(
			common.HashSHA512_384,
			[]byte(payload),
			[]byte(b.APISecret)))

	request["authPayload"] = payload

	return b.WsSend(request)
}

// WsSendUnauth sends an unauthenticated payload
func (b *Bitfinex) WsSendUnauth() error {
	request := make(map[string]string)
	request["event"] = "unauth"

	return b.WsSend(request)
}

// WsAddSubscriptionChannel adds a new subscription channel to the
// WebsocketSubdChannels map in bitfinex.go (Bitfinex struct)
func (b *Bitfinex) WsAddSubscriptionChannel(chanID int, channel, pair string) {
	chanInfo := WebsocketChanInfo{Pair: pair, Channel: channel}
	b.WebsocketSubdChannels[chanID] = chanInfo

	if b.Verbose {
		log.Printf("%s Subscribed to Channel: %s Pair: %s ChannelID: %d\n",
			b.GetName(),
			channel,
			pair,
			chanID)
	}
}

// WsConnect starts a new websocket connection
func (b *Bitfinex) WsConnect() error {
	if !b.Websocket.IsEnabled() || !b.IsEnabled() {
		return errors.New(exchange.WebsocketNotEnabled)
	}

	var channels = []string{"book", "trades", "ticker"}
	var Dialer websocket.Dialer
	var err error

	if b.Websocket.GetProxyAddress() != "" {
		proxy, err := url.Parse(b.Websocket.GetProxyAddress())
		if err != nil {
			return err
		}
		Dialer.Proxy = http.ProxyURL(proxy)
	}

	b.WebsocketConn, _, err = Dialer.Dial(b.Websocket.GetWebsocketURL(), http.Header{})
	if err != nil {
		return fmt.Errorf("Unable to connect to Websocket. Error: %s", err)
	}

	_, resp, err := b.WebsocketConn.ReadMessage()
	if err != nil {
		return fmt.Errorf("Unable to read from Websocket. Error: %s", err)
	}

	var hs WebsocketHandshake
	err = common.JSONDecode(resp, &hs)
	if err != nil {
		return err
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
			err := b.WsSubscribe(x, params)
			if err != nil {
				return err
			}
		}
	}

	if b.AuthenticatedAPISupport {
		err = b.WsSendAuth()
		if err != nil {
			return err
		}
	}

	pongReceive = make(chan struct{}, 1)
	comms := make(chan wsTraffic, 1)

	go b.WsReadData(comms)
	go b.WsDataHandler(comms)

	return nil
}

// wsTraffic defines websocket stream event
type wsTraffic struct {
	MsgType int
	Resp    []byte
}

// WsReadData reads and handles websocket stream data
func (b *Bitfinex) WsReadData(comms chan wsTraffic) {
	b.Websocket.Wg.Add(1)
	defer func() {
		err := b.WebsocketConn.Close()
		if err != nil {
			b.Websocket.DataHandler <- fmt.Errorf("bitfinex_websocket.go - closing websocket connection error %s",
				err)
		}
		b.Websocket.Wg.Done()
	}()

	for {
		select {
		case <-b.Websocket.ShutdownC:
			return
		default:
			msgType, resp, err := b.WebsocketConn.ReadMessage()
			if err != nil {
				b.Websocket.DataHandler <- err
				return
			}

			b.Websocket.TrafficTimer.Reset(exchange.WebsocketTrafficLimitTime)

			comms <- wsTraffic{
				MsgType: msgType,
				Resp:    resp,
			}
		}
	}
}

// WsDataHandler handles data from WsReadData
func (b *Bitfinex) WsDataHandler(comms chan wsTraffic) {
	b.Websocket.Wg.Add(1)
	defer b.Websocket.Wg.Done()

	for {
		select {
		case <-b.Websocket.ShutdownC:
			return

		case stream := <-comms:

			switch stream.MsgType {
			case websocket.TextMessage:
				var result interface{}
				common.JSONDecode(stream.Resp, &result)

				switch reflect.TypeOf(result).String() {
				case "map[string]interface {}":
					eventData := result.(map[string]interface{})
					event := eventData["event"]

					switch event {
					case "subscribed":
						b.WsAddSubscriptionChannel(int(eventData["chanId"].(float64)),
							eventData["channel"].(string),
							eventData["pair"].(string))

					case "auth":
						status := eventData["status"].(string)

						if status == "OK" {
							b.WsAddSubscriptionChannel(0, "account", "N/A")

						} else if status == "fail" {
							b.Websocket.DataHandler <- fmt.Errorf("bitfinex.go error - Websocket unable to AUTH. Error code: %s",
								eventData["code"].(string))

							b.AuthenticatedAPISupport = false
						}
					}

				case "[]interface {}":
					chanData := result.([]interface{})
					chanID := int(chanData[0].(float64))

					chanInfo, ok := b.WebsocketSubdChannels[chanID]
					if !ok {
						b.Websocket.DataHandler <- fmt.Errorf("bitfinex.go error - Unable to locate chanID: %d",
							chanID)
						continue
					} else {
						if len(chanData) == 2 {
							if reflect.TypeOf(chanData[1]).String() == "string" {
								if chanData[1].(string) == bitfinexWebsocketHeartbeat {
									continue
								} else if chanData[1].(string) == "pong" {
									pongReceive <- struct{}{}
									continue
								}
							}
						}

						switch chanInfo.Channel {
						case "book":
							newOrderbook := []WebsocketBook{}
							switch len(chanData) {
							case 2:
								data := chanData[1].([]interface{})
								for _, x := range data {
									y := x.([]interface{})
									newOrderbook = append(newOrderbook, WebsocketBook{
										Price:  y[0].(float64),
										Count:  int(y[1].(float64)),
										Amount: y[2].(float64)})
								}

							case 4:
								newOrderbook = append(newOrderbook, WebsocketBook{
									Price:  chanData[1].(float64),
									Count:  int(chanData[2].(float64)),
									Amount: chanData[3].(float64)})
							}

							if len(newOrderbook) > 1 {
								ob, err := localOrderBook.InsertInitialStore(pair.NewCurrencyPairFromString(chanInfo.Pair), "SPOT", newOrderbook)
								if err != nil {
									b.Websocket.DataHandler <- err
									continue
								}

								orderbook.ProcessOrderbook(b.GetName(), ob.Pair, ob, ob.AssetType)

								b.Websocket.DataHandler <- exchange.WebsocketOrderbookUpdate{
									Exchange: b.GetName(),
									Asset:    ob.AssetType,
									Pair:     ob.Pair,
								}
								continue
							}

							ob, err := localOrderBook.Update(pair.NewCurrencyPairFromString(chanInfo.Pair), "SPOT", newOrderbook[0])
							if err != nil {
								b.Websocket.DataHandler <- err
								continue
							}

							orderbook.ProcessOrderbook(b.GetName(), ob.Pair, ob, ob.AssetType)

							b.Websocket.DataHandler <- exchange.WebsocketOrderbookUpdate{
								Exchange: b.GetName(),
								Asset:    ob.AssetType,
								Pair:     ob.Pair,
							}

						case "ticker":
							b.Websocket.DataHandler <- exchange.TickerData{
								Quantity:   chanData[8].(float64),
								ClosePrice: chanData[7].(float64),
								HighPrice:  chanData[9].(float64),
								LowPrice:   chanData[10].(float64),
								Pair:       pair.NewCurrencyPairFromString(chanInfo.Pair),
								Exchange:   b.GetName(),
								AssetType:  "SPOT",
							}

						case "account":
							switch chanData[1].(string) {
							case bitfinexWebsocketPositionSnapshot:
								positionSnapshot := []WebsocketPosition{}
								data := chanData[2].([]interface{})
								for _, x := range data {
									y := x.([]interface{})
									positionSnapshot = append(positionSnapshot,
										WebsocketPosition{
											Pair:              y[0].(string),
											Status:            y[1].(string),
											Amount:            y[2].(float64),
											Price:             y[3].(float64),
											MarginFunding:     y[4].(float64),
											MarginFundingType: int(y[5].(float64))})
								}
								log.Println("Position Snapshot:", positionSnapshot)

							case bitfinexWebsocketPositionNew, bitfinexWebsocketPositionUpdate, bitfinexWebsocketPositionClose:
								data := chanData[2].([]interface{})
								position := WebsocketPosition{
									Pair:              data[0].(string),
									Status:            data[1].(string),
									Amount:            data[2].(float64),
									Price:             data[3].(float64),
									MarginFunding:     data[4].(float64),
									MarginFundingType: int(data[5].(float64))}
								log.Println("Current Position:", position)

							case bitfinexWebsocketWalletSnapshot:
								data := chanData[2].([]interface{})
								walletSnapshot := []WebsocketWallet{}
								for _, x := range data {
									y := x.([]interface{})
									walletSnapshot = append(walletSnapshot,
										WebsocketWallet{
											Name:              y[0].(string),
											Currency:          y[1].(string),
											Balance:           y[2].(float64),
											UnsettledInterest: y[3].(float64)})
								}
								log.Println("Current Wallet Snaptshot:", walletSnapshot)

							case bitfinexWebsocketWalletUpdate:
								data := chanData[2].([]interface{})
								wallet := WebsocketWallet{
									Name:              data[0].(string),
									Currency:          data[1].(string),
									Balance:           data[2].(float64),
									UnsettledInterest: data[3].(float64)}
								log.Println("Update Wallet:", wallet)

							case bitfinexWebsocketOrderSnapshot:
								orderSnapshot := []WebsocketOrder{}
								data := chanData[2].([]interface{})
								for _, x := range data {
									y := x.([]interface{})
									orderSnapshot = append(orderSnapshot,
										WebsocketOrder{
											OrderID:    int64(y[0].(float64)),
											Pair:       y[1].(string),
											Amount:     y[2].(float64),
											OrigAmount: y[3].(float64),
											OrderType:  y[4].(string),
											Status:     y[5].(string),
											Price:      y[6].(float64),
											PriceAvg:   y[7].(float64),
											Timestamp:  y[8].(string)})
								}
								log.Println("Orders Snapshot:", orderSnapshot)

							case bitfinexWebsocketOrderNew, bitfinexWebsocketOrderUpdate, bitfinexWebsocketOrderCancel:
								data := chanData[2].([]interface{})
								order := WebsocketOrder{
									OrderID:    int64(data[0].(float64)),
									Pair:       data[1].(string),
									Amount:     data[2].(float64),
									OrigAmount: data[3].(float64),
									OrderType:  data[4].(string),
									Status:     data[5].(string),
									Price:      data[6].(float64),
									PriceAvg:   data[7].(float64),
									Timestamp:  data[8].(string),
									Notify:     int(data[9].(float64))}
								log.Println("Current Orders:", order)

							case bitfinexWebsocketTradeExecuted:
								data := chanData[2].([]interface{})
								trade := WebsocketTradeExecuted{
									TradeID:        int64(data[0].(float64)),
									Pair:           data[1].(string),
									Timestamp:      int64(data[2].(float64)),
									OrderID:        int64(data[3].(float64)),
									AmountExecuted: data[4].(float64),
									PriceExecuted:  data[5].(float64)}
								log.Println("Current Trades:", trade)
							}

						case "trades":
							trades := []WebsocketTrade{}
							switch len(chanData) {
							case 2:
								data := chanData[1].([]interface{})
								for _, x := range data {
									y := x.([]interface{})
									if _, ok := y[0].(string); ok {
										continue
									}
									trades = append(trades,
										WebsocketTrade{
											//ID:        int64(y[0].(float64)), NULL is coming through
											Timestamp: int64(y[1].(float64)),
											Price:     y[2].(float64),
											Amount:    y[3].(float64)})
								}

							case 7:
								trade := WebsocketTrade{
									ID:        int64(chanData[3].(float64)),
									Timestamp: int64(chanData[4].(float64)),
									Price:     chanData[5].(float64),
									Amount:    chanData[6].(float64)}
								trades = append(trades, trade)
							}

							if len(trades) > 0 {
								side := "BUY"
								newAmount := trades[0].Amount
								if newAmount < 0 {
									side = "SELL"
									newAmount = newAmount * -1
								}

								b.Websocket.DataHandler <- exchange.TradeData{
									CurrencyPair: pair.NewCurrencyPairFromString(chanInfo.Pair),
									Timestamp:    time.Unix(trades[0].Timestamp, 0),
									Price:        trades[0].Price,
									Amount:       newAmount,
									Exchange:     b.GetName(),
									AssetType:    "SPOT",
									Side:         side,
								}
							}
						}
					}
				}
			}
		}
	}
}

var localOrderBook LocalStore

// LocalStore defines the storage of a local cache of orderbooks
type LocalStore struct {
	ob []orderbook.Base
	sync.Mutex
}

// InsertInitialStore add the initial orderbook snapshot when subscribed to a
// channel
func (l *LocalStore) InsertInitialStore(p pair.CurrencyPair, assetType string, books []WebsocketBook) (orderbook.Base, error) {
	for _, ob := range l.ob {
		if ob.Pair == p && ob.AssetType == assetType {
			return orderbook.Base{}, errors.New("bitfinex.go error - Currency pair asset type already set for orderbook")
		}
	}

	var bid, ask []orderbook.Item
	for _, book := range books {
		if book.Amount >= 0 {
			bid = append(bid, orderbook.Item{Amount: book.Amount, Price: book.Price})
		} else {
			ask = append(ask, orderbook.Item{Amount: book.Amount * -1, Price: book.Price})
		}
	}

	if len(bid) == 0 && len(ask) == 0 {
		return orderbook.Base{}, errors.New("bitfinex.go error - orderbooks not set correctly")
	}

	l.ob = append(l.ob,
		orderbook.Base{
			Pair:         p,
			CurrencyPair: p.Pair().String(),
			Bids:         bid,
			Asks:         ask,
			LastUpdated:  time.Now(),
			AssetType:    assetType})

	return l.Get(p, assetType)
}

// Update updates the orderbook list, removing and adding to the orderbook sides
func (l *LocalStore) Update(p pair.CurrencyPair, assetType string, book WebsocketBook) (orderbook.Base, error) {
	if book.Amount >= 0 {
		if book.Count == 0 {
			l.Remove(p, book, true)

			return l.Get(p, assetType)
		}

		l.Add(p, assetType, book.Amount, book.Price, true)
		return l.Get(p, assetType)
	}

	if book.Count == 0 {
		l.Remove(p, book, false)
		return l.Get(p, assetType)
	}

	l.Add(p, assetType, book.Amount*-1, book.Price, false)
	return l.Get(p, assetType)
}

// Get returns the full orderbook for a currency pair/asset
func (l *LocalStore) Get(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	l.Lock()
	defer l.Unlock()

	for _, data := range l.ob {
		if data.Pair == p && data.AssetType == assetType {
			return data, nil
		}
	}
	return orderbook.Base{}, errors.New("bitfinex.go error - could not find orderbook")
}

// Remove removes prices from an orderbook from either a bid or ask side
func (l *LocalStore) Remove(p pair.CurrencyPair, book WebsocketBook, bid bool) {
	l.Lock()
	defer l.Unlock()
	for x := range l.ob {
		if l.ob[x].Pair == p {
			if bid {
				for y := range l.ob[x].Bids {
					if l.ob[x].Bids[y].Price == book.Price {
						l.ob[x].Bids = append(l.ob[x].Bids[:y], l.ob[x].Bids[y+1:]...)
						l.ob[x].LastUpdated = time.Now()
						return
					}
				}
			} else {
				for y := range l.ob[x].Asks {
					if l.ob[x].Asks[y].Price == book.Price*-1 {
						l.ob[x].Asks = append(l.ob[x].Asks[:y], l.ob[x].Asks[y+1:]...)
						l.ob[x].LastUpdated = time.Now()
						return
					}
				}
			}
		}
	}
}

// Add adds a new orderbook entry on either a bid or ask side
func (l *LocalStore) Add(p pair.CurrencyPair, assetType string, amount, price float64, bid bool) {
	l.Lock()
	defer l.Unlock()

	if bid {
		for i := range l.ob {
			if l.ob[i].Pair == p && l.ob[i].AssetType == assetType {
				l.ob[i].Bids = append(l.ob[i].Bids, orderbook.Item{Amount: amount, Price: price})
				l.ob[i].LastUpdated = time.Now()
				return
			}
		}
	} else {
		for i := range l.ob {
			if l.ob[i].Pair == p && l.ob[i].AssetType == assetType {
				l.ob[i].Asks = append(l.ob[i].Asks, orderbook.Item{Amount: amount, Price: price})
				l.ob[i].LastUpdated = time.Now()
				return
			}
		}
	}
	log.Fatalf("bitfinex.go error - Could not find orderbook for Pair:%s Asset:%s Amount:%f Price:%f, BID: %t",
		p.Pair().String(),
		assetType,
		amount,
		price,
		bid)
}
