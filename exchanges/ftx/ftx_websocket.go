package ftx

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	ftxWSURL          = "wss://ftx.com/ws/"
	ftxWebsocketTimer = 13
	wsTicker          = "ticker"
	wsTrades          = "trades"
	wsOrderbook       = "orderbook"
	wsFills           = "fills"
	wsOrders          = "orders"
)

// WsConnect connects to a websocket feed
func (f *FTX) WsConnect() error {
	if !f.Websocket.IsEnabled() || !f.IsEnabled() {
		return errors.New(wshandler.WebsocketNotEnabled)
	}
	var dialer websocket.Dialer
	err := f.WebsocketConn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}
	f.WebsocketConn.SetupPingHandler(wshandler.WebsocketPingHandler{
		MessageType: websocket.PingMessage,
		Delay:       ftxWebsocketTimer * time.Second,
	})
	if f.Verbose {
		log.Debugf(log.ExchangeSys, "%s Connected to Websocket.\n", f.Name)
	}
	go f.wsReadData()
	if f.GetAuthenticatedAPISupport(exchange.WebsocketAuthentication) {
		err := f.WsAuth()
		if err != nil {
			f.Websocket.DataHandler <- err
			f.Websocket.SetCanUseAuthenticatedEndpoints(false)
		}
	}
	f.GenerateDefaultSubscriptions()
	return nil
}

// WsAuth sends an authentication message to receive auth data
func (f *FTX) WsAuth() error {
	intNonce := int64(time.Now().UnixNano() / 1000000)
	strNonce := strconv.FormatInt(intNonce, 10)
	hmac := crypto.GetHMAC(
		crypto.HashSHA256,
		[]byte(strNonce+"websocket_login"),
		[]byte(f.API.Credentials.Secret),
	)
	sign := crypto.HexEncodeToString(hmac)
	req := Authenticate{Operation: "login",
		Args: AuthenticationData{Key: f.API.Credentials.Key,
			Sign: sign,
			Time: intNonce,
		},
	}
	return f.WebsocketConn.SendJSONMessage(req)
}

// Subscribe sends a websocket message to receive data from the channel
func (f *FTX) Subscribe(channelToSubscribe wshandler.WebsocketChannelSubscription) error {
	var sub WsSub
	switch channelToSubscribe.Channel {
	case wsFills, wsOrders:
		sub.Operation = "subscribe"
		sub.Channel = channelToSubscribe.Channel
	default:
		sub.Operation = "subscribe"
		sub.Channel = channelToSubscribe.Channel
		sub.Market = f.FormatExchangeCurrency(channelToSubscribe.Currency, asset.Futures).String()
	}
	return f.WebsocketConn.SendJSONMessage(sub)
}

// GenerateDefaultSubscriptions generates default subscription
func (f *FTX) GenerateDefaultSubscriptions() {
	var channels = []string{wsTicker, wsTrades, wsOrderbook, wsFills, wsOrders}
	pairs := f.GetEnabledPairs(asset.Futures)
	newPair := currency.NewPairWithDelimiter(pairs[0].Base.String(), pairs[0].Quote.String(), "-")
	fmt.Println(newPair)
	var subscriptions []wshandler.WebsocketChannelSubscription
	for x := range channels {
		subscriptions = append(subscriptions, wshandler.WebsocketChannelSubscription{
			Channel:  channels[x],
			Currency: newPair,
		})
	}
	f.Websocket.SubscribeToChannels(subscriptions)
}

// wsReadData gets and passes on websocket messages for processing
func (f *FTX) wsReadData() {
	f.Websocket.Wg.Add(1)

	defer func() {
		f.Websocket.Wg.Done()
	}()

	for {
		select {
		case <-f.Websocket.ShutdownC:
			return

		default:
			resp, err := f.WebsocketConn.ReadMessage()
			if err != nil {
				f.Websocket.ReadMessageErrors <- err
				return
			}
			f.Websocket.TrafficAlert <- struct{}{}
			err = f.wsHandleData(resp.Raw)
			if err != nil {
				f.Websocket.DataHandler <- err
			}
		}
	}
}

func (f *FTX) wsHandleData(respRaw []byte) error {
	type Result map[string]interface{}
	var result Result
	err := json.Unmarshal(respRaw, &result)
	if err != nil {
		return err
	}
	switch result["type"] {
	case "subscribed":
		switch {
		case result["channel"] == "ticker":
			resultData := result["data"].(WsTickerData)
			f.Websocket.DataHandler <- &ticker.Price{
				ExchangeName: f.Name,
				Bid:          resultData.Bid,
				Ask:          resultData.Ask,
				Last:         resultData.Last,
				LastUpdated:  resultData.Time,
				Pair:         currency.NewPairFromString(result["market"].(string)),
			}
		case result["channel"] == "orderbook":
			resultData := result["data"].(WsOrderbookData)
		case result["channel"] == "trades":
		default:
			f.Websocket.DataHandler <- wshandler.UnhandledMessageWarning{Message: f.Name + wshandler.UnhandledMessage + string(respRaw)}
			return nil
		}
	case "error":
		f.Websocket.DataHandler <- wshandler.UnhandledMessageWarning{Message: f.Name + wshandler.UnhandledMessage + string(respRaw)}
	}
	return nil
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (f *FTX) Unsubscribe(channelToSubscribe wshandler.WebsocketChannelSubscription) error {
	var unSub WsSub
	unSub.Operation = "unsubscribe"
	unSub.Channel = channelToSubscribe.Channel
	return f.WebsocketConn.SendJSONMessage(unSub)
}
