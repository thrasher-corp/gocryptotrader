package bitfinex

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wsorderbook"
	log "github.com/thrasher-corp/gocryptotrader/logger"
)

const (
	bitfinexWebsocket                     = "wss://api.bitfinex.com/ws"
	bitfinexWebsocketVersion              = "1.1"
	bitfinexWebsocketPositionSnapshot     = "ps"
	bitfinexWebsocketPositionNew          = "pn"
	bitfinexWebsocketPositionUpdate       = "pu"
	bitfinexWebsocketPositionClose        = "pc"
	bitfinexWebsocketWalletSnapshot       = "ws"
	bitfinexWebsocketWalletUpdate         = "wu"
	bitfinexWebsocketOrderSnapshot        = "os"
	bitfinexWebsocketOrderNew             = "on"
	bitfinexWebsocketOrderUpdate          = "ou"
	bitfinexWebsocketOrderCancel          = "oc"
	bitfinexWebsocketTradeExecuted        = "te"
	bitfinexWebsocketTradeExecutionUpdate = "tu"
	bitfinexWebsocketTradeSnapshots       = "ts"
	bitfinexWebsocketHeartbeat            = "hb"
	bitfinexWebsocketAlertRestarting      = "20051"
	bitfinexWebsocketAlertRefreshing      = "20060"
	bitfinexWebsocketAlertResume          = "20061"
	bitfinexWebsocketUnknownEvent         = "10000"
	bitfinexWebsocketUnknownPair          = "10001"
	bitfinexWebsocketSubscriptionFailed   = "10300"
	bitfinexWebsocketAlreadySubscribed    = "10301"
	bitfinexWebsocketUnknownChannel       = "10302"
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
	req := make(map[string]string)
	req["event"] = "ping"
	return b.WebsocketConn.SendMessage(req)
}

// WsSendAuth sends a autheticated event payload
func (b *Bitfinex) WsSendAuth() error {
	if !b.GetAuthenticatedAPISupport(exchange.WebsocketAuthentication) {
		return fmt.Errorf("%v AuthenticatedWebsocketAPISupport not enabled", b.Name)
	}
	req := make(map[string]interface{})
	payload := "AUTH" + strconv.FormatInt(time.Now().UnixNano(), 10)[:13]
	req["event"] = "auth"
	req["apiKey"] = b.API.Credentials.Key

	req["authSig"] = crypto.HexEncodeToString(
		crypto.GetHMAC(
			crypto.HashSHA512_384,
			[]byte(payload),
			[]byte(b.API.Credentials.Secret)))

	req["authPayload"] = payload

	err := b.WebsocketConn.SendMessage(req)
	if err != nil {
		b.Websocket.SetCanUseAuthenticatedEndpoints(false)
		return err
	}
	return nil
}

// WsSendUnauth sends an unauthenticated payload
func (b *Bitfinex) WsSendUnauth() error {
	req := make(map[string]string)
	req["event"] = "unauth"

	return b.WebsocketConn.SendMessage(req)
}

// WsAddSubscriptionChannel adds a new subscription channel to the
// WebsocketSubdChannels map in bitfinex.go (Bitfinex struct)
func (b *Bitfinex) WsAddSubscriptionChannel(chanID int, channel, pair string) {
	chanInfo := WebsocketChanInfo{Pair: pair, Channel: channel}
	b.WebsocketSubdChannels[chanID] = chanInfo

	if b.Verbose {
		log.Debugf(log.ExchangeSys, "%s Subscribed to Channel: %s Pair: %s ChannelID: %d\n",
			b.GetName(),
			channel,
			pair,
			chanID)
	}
}

// WsConnect starts a new websocket connection
func (b *Bitfinex) WsConnect() error {
	if !b.Websocket.IsEnabled() || !b.IsEnabled() {
		return errors.New(wshandler.WebsocketNotEnabled)
	}

	var dialer websocket.Dialer
	b.WebsocketConn = &wshandler.WebsocketConnection{
		ExchangeName: b.Name,
		URL:          b.Websocket.GetWebsocketURL(),
		ProxyURL:     b.Websocket.GetProxyAddress(),
		Verbose:      b.Verbose,
	}
	err := b.WebsocketConn.Dial(&dialer, http.Header{})
	if err != nil {
		return fmt.Errorf("%v unable to connect to Websocket. Error: %s", b.Name, err)
	}

	resp, err := b.WebsocketConn.ReadMessage()
	if err != nil {
		b.Websocket.ReadMessageErrors <- err
		return fmt.Errorf("%v unable to read from Websocket. Error: %s", b.Name, err)
	}
	b.Websocket.TrafficAlert <- struct{}{}
	var hs WebsocketHandshake
	err = common.JSONDecode(resp.Raw, &hs)
	if err != nil {
		return err
	}

	err = b.WsSendAuth()
	if err != nil {
		log.Errorf(log.ExchangeSys, "%v - authentication failed: %v\n", b.Name, err)
	}

	b.GenerateDefaultSubscriptions()
	if hs.Event == "info" {
		if b.Verbose {
			log.Debugf(log.ExchangeSys, "%s Connected to Websocket.\n", b.GetName())
		}
	}

	pongReceive = make(chan struct{}, 1)

	go b.WsDataHandler()

	return nil
}

// WsDataHandler handles data from WsReadData
func (b *Bitfinex) WsDataHandler() {
	b.Websocket.Wg.Add(1)

	defer b.Websocket.Wg.Done()

	for {
		select {
		case <-b.Websocket.ShutdownC:
			return

		default:
			stream, err := b.WebsocketConn.ReadMessage()
			if err != nil {
				b.Websocket.ReadMessageErrors <- err
				return
			}
			b.Websocket.TrafficAlert <- struct{}{}

			if stream.Type == websocket.TextMessage {
				var result interface{}
				err = common.JSONDecode(stream.Raw, &result)
				if err != nil {
					b.Websocket.DataHandler <- err
					return
				}
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
							b.Websocket.DataHandler <- eventData
							b.WsAddSubscriptionChannel(0, "account", "N/A")
						} else if status == "fail" {
							b.Websocket.DataHandler <- fmt.Errorf("bitfinex.go error - Websocket unable to AUTH. Error code: %s",
								eventData["code"].(string))
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
					}
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
						var newOrderbook []WebsocketBook
						curr := currency.NewPairFromString(chanInfo.Pair)
						switch len(chanData) {
						case 2:
							data := chanData[1].([]interface{})
							for i := range data {
								y := data[i].([]interface{})
								newOrderbook = append(newOrderbook, WebsocketBook{
									Price:  y[0].(float64),
									Count:  int(y[1].(float64)),
									Amount: y[2].(float64)})
							}

							err := b.WsInsertSnapshot(curr,
								asset.Spot,
								newOrderbook)

							if err != nil {
								b.Websocket.DataHandler <- fmt.Errorf("bitfinex_websocket.go inserting snapshot error: %s",
									err)
							}
						case 4:
							newOrderbook = append(newOrderbook, WebsocketBook{
								Price:  chanData[1].(float64),
								Count:  int(chanData[2].(float64)),
								Amount: chanData[3].(float64)})
							err := b.WsUpdateOrderbook(curr,
								asset.Spot,
								newOrderbook)

							if err != nil {
								b.Websocket.DataHandler <- fmt.Errorf("bitfinex_websocket.go updating orderbook error: %s",
									err)
							}
						}
					case "ticker":
						b.Websocket.DataHandler <- wshandler.TickerData{
							Exchange:  b.Name,
							Volume:    chanData[8].(float64),
							High:      chanData[9].(float64),
							Low:       chanData[10].(float64),
							Bid:       chanData[1].(float64),
							Ask:       chanData[3].(float64),
							Last:      chanData[7].(float64),
							AssetType: asset.Spot,
							Pair:      currency.NewPairFromString(chanInfo.Pair),
						}

					case "account":
						switch chanData[1].(string) {
						case bitfinexWebsocketPositionSnapshot:
							var positionSnapshot []WebsocketPosition
							data := chanData[2].([]interface{})
							for i := range data {
								y := data[i].([]interface{})
								positionSnapshot = append(positionSnapshot,
									WebsocketPosition{
										Pair:              y[0].(string),
										Status:            y[1].(string),
										Amount:            y[2].(float64),
										Price:             y[3].(float64),
										MarginFunding:     y[4].(float64),
										MarginFundingType: int(y[5].(float64))})
							}

							if len(positionSnapshot) == 0 {
								continue
							}

							b.Websocket.DataHandler <- positionSnapshot

						case bitfinexWebsocketPositionNew, bitfinexWebsocketPositionUpdate, bitfinexWebsocketPositionClose:
							data := chanData[2].([]interface{})
							position := WebsocketPosition{
								Pair:              data[0].(string),
								Status:            data[1].(string),
								Amount:            data[2].(float64),
								Price:             data[3].(float64),
								MarginFunding:     data[4].(float64),
								MarginFundingType: int(data[5].(float64))}

							b.Websocket.DataHandler <- position

						case bitfinexWebsocketWalletSnapshot:
							data := chanData[2].([]interface{})
							var walletSnapshot []WebsocketWallet
							for i := range data {
								y := data[i].([]interface{})
								walletSnapshot = append(walletSnapshot,
									WebsocketWallet{
										Name:              y[0].(string),
										Currency:          y[1].(string),
										Balance:           y[2].(float64),
										UnsettledInterest: y[3].(float64)})
							}

							b.Websocket.DataHandler <- walletSnapshot

						case bitfinexWebsocketWalletUpdate:
							data := chanData[2].([]interface{})
							wallet := WebsocketWallet{
								Name:              data[0].(string),
								Currency:          data[1].(string),
								Balance:           data[2].(float64),
								UnsettledInterest: data[3].(float64)}

							b.Websocket.DataHandler <- wallet

						case bitfinexWebsocketOrderSnapshot:
							var orderSnapshot []WebsocketOrder
							data := chanData[2].([]interface{})
							for i := range data {
								y := data[i].([]interface{})
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

							b.Websocket.DataHandler <- orderSnapshot

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

							b.Websocket.DataHandler <- order

						case bitfinexWebsocketTradeExecuted:
							data := chanData[2].([]interface{})
							trade := WebsocketTradeExecuted{
								TradeID:        int64(data[0].(float64)),
								Pair:           data[1].(string),
								Timestamp:      int64(data[2].(float64)),
								OrderID:        int64(data[3].(float64)),
								AmountExecuted: data[4].(float64),
								PriceExecuted:  data[5].(float64)}

							b.Websocket.DataHandler <- trade
						case bitfinexWebsocketTradeSnapshots, bitfinexWebsocketTradeExecutionUpdate:
							data := chanData[2].([]interface{})
							trade := WebsocketTradeData{
								TradeID:        int64(data[0].(float64)),
								Pair:           data[1].(string),
								Timestamp:      int64(data[2].(float64)),
								OrderID:        int64(data[3].(float64)),
								AmountExecuted: data[4].(float64),
								PriceExecuted:  data[5].(float64),
								Fee:            data[6].(float64),
								FeeCurrency:    data[7].(string)}

							b.Websocket.DataHandler <- trade
						}

					case "trades":
						var trades []WebsocketTrade
						switch len(chanData) {
						case 2:
							data := chanData[1].([]interface{})
							for i := range data {
								y := data[i].([]interface{})
								if _, ok := y[0].(string); ok {
									continue
								}

								id, _ := y[0].(float64)

								trades = append(trades,
									WebsocketTrade{
										ID:        int64(id),
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
								newAmount *= -1
							}

							b.Websocket.DataHandler <- wshandler.TradeData{
								CurrencyPair: currency.NewPairFromString(chanInfo.Pair),
								Timestamp:    time.Unix(trades[0].Timestamp, 0),
								Price:        trades[0].Price,
								Amount:       newAmount,
								Exchange:     b.GetName(),
								AssetType:    asset.Spot,
								Side:         side,
							}
						}
					}
				}
			}
		}
	}
}

// WsInsertSnapshot add the initial orderbook snapshot when subscribed to a
// channel
func (b *Bitfinex) WsInsertSnapshot(p currency.Pair, assetType asset.Item, books []WebsocketBook) error {
	if len(books) == 0 {
		return errors.New("bitfinex.go error - no orderbooks submitted")
	}
	var bid, ask []orderbook.Item
	for i := range books {
		if books[i].Amount >= 0 {
			bid = append(bid, orderbook.Item{Amount: books[i].Amount, Price: books[i].Price})
		} else {
			ask = append(ask, orderbook.Item{Amount: books[i].Amount * -1, Price: books[i].Price})
		}
	}
	if len(bid) == 0 && len(ask) == 0 {
		return errors.New("bitfinex.go error - no orderbooks in item lists")
	}
	var newOrderBook orderbook.Base
	newOrderBook.Asks = ask
	newOrderBook.AssetType = assetType
	newOrderBook.Bids = bid
	newOrderBook.Pair = p
	newOrderBook.ExchangeName = b.GetName()

	err := b.Websocket.Orderbook.LoadSnapshot(&newOrderBook)
	if err != nil {
		return fmt.Errorf("bitfinex.go error - %s", err)
	}
	b.Websocket.DataHandler <- wshandler.WebsocketOrderbookUpdate{Pair: p,
		Asset:    assetType,
		Exchange: b.GetName()}
	return nil
}

// WsUpdateOrderbook updates the orderbook list, removing and adding to the
// orderbook sides
func (b *Bitfinex) WsUpdateOrderbook(p currency.Pair, assetType asset.Item, book []WebsocketBook) error {
	orderbookUpdate := wsorderbook.WebsocketOrderbookUpdate{
		Asks:  []orderbook.Item{},
		Bids:  []orderbook.Item{},
		Asset: assetType,
		Pair:  p,
	}

	for i := 0; i < len(book); i++ {
		switch {
		case book[i].Count > 0:
			if book[i].Amount > 0 {
				// update bid
				orderbookUpdate.Bids = append(orderbookUpdate.Bids, orderbook.Item{Amount: book[i].Amount, Price: book[i].Price})
			} else if book[i].Amount < 0 {
				// update ask
				orderbookUpdate.Asks = append(orderbookUpdate.Asks, orderbook.Item{Amount: book[i].Amount * -1, Price: book[i].Price})
			}
		case book[i].Count == 0:
			if book[i].Amount == 1 {
				// delete bid
				orderbookUpdate.Bids = append(orderbookUpdate.Bids, orderbook.Item{Amount: 0, Price: book[i].Price})
			} else if book[i].Amount == -1 {
				// delete ask
				orderbookUpdate.Asks = append(orderbookUpdate.Asks, orderbook.Item{Amount: 0, Price: book[i].Price})
			}
		}
	}
	orderbookUpdate.UpdateTime = time.Now()
	err := b.Websocket.Orderbook.Update(&orderbookUpdate)
	if err != nil {
		return err
	}

	b.Websocket.DataHandler <- wshandler.WebsocketOrderbookUpdate{Pair: p,
		Asset:    assetType,
		Exchange: b.GetName()}

	return nil
}

// GenerateDefaultSubscriptions Adds default subscriptions to websocket to be handled by ManageSubscriptions()
func (b *Bitfinex) GenerateDefaultSubscriptions() {
	var channels = []string{"book", "trades", "ticker"}
	var subscriptions []wshandler.WebsocketChannelSubscription
	for i := range channels {
		enabledPairs := b.GetEnabledPairs(asset.Spot)
		for j := range enabledPairs {
			b.appendOptionalDelimiter(&enabledPairs[j])
			params := make(map[string]interface{})
			if channels[i] == "book" {
				params["prec"] = "P0"
				params["len"] = "100"
			}
			subscriptions = append(subscriptions, wshandler.WebsocketChannelSubscription{
				Channel:  channels[i],
				Currency: enabledPairs[j],
				Params:   params,
			})
		}
	}
	b.Websocket.SubscribeToChannels(subscriptions)
}

// Subscribe sends a websocket message to receive data from the channel
func (b *Bitfinex) Subscribe(channelToSubscribe wshandler.WebsocketChannelSubscription) error {
	req := make(map[string]interface{})
	req["event"] = "subscribe"
	req["channel"] = channelToSubscribe.Channel
	if channelToSubscribe.Currency.String() != "" {
		req["pair"] = b.FormatExchangeCurrency(channelToSubscribe.Currency,
			asset.Spot).String()
	}
	if len(channelToSubscribe.Params) > 0 {
		for k, v := range channelToSubscribe.Params {
			req[k] = v
		}
	}
	return b.WebsocketConn.SendMessage(req)
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (b *Bitfinex) Unsubscribe(channelToSubscribe wshandler.WebsocketChannelSubscription) error {
	req := make(map[string]interface{})
	req["event"] = "unsubscribe"
	req["channel"] = channelToSubscribe.Channel

	if len(channelToSubscribe.Params) > 0 {
		for k, v := range channelToSubscribe.Params {
			req[k] = v
		}
	}
	return b.WebsocketConn.SendMessage(req)
}
