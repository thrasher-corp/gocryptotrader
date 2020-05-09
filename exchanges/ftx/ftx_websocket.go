package ftx

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wsorderbook"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	ftxWSURL          = "wss://ftx.com/ws/"
	ftxWebsocketTimer = 13 * time.Second
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
		Delay:       ftxWebsocketTimer,
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
	intNonce := time.Now().UnixNano() / 1000000
	strNonce := strconv.FormatInt(intNonce, 10)
	hmac := crypto.GetHMAC(
		crypto.HashSHA256,
		[]byte(strNonce+"websocket_login"),
		[]byte(f.API.Credentials.Secret),
	)
	sign := crypto.HexEncodeToString(hmac)
	req := Authenticate{Operation: "login",
		Args: AuthenticationData{
			Key:  f.API.Credentials.Key,
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

	defer f.Websocket.Wg.Done()

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

func timestampFromFloat64(ts float64) time.Time {
	secs := int64(ts)
	nsecs := int64((ts - float64(secs)) * 1e9)
	return time.Unix(secs, nsecs)
}

func (f *FTX) wsHandleData(respRaw []byte) error {
	var result map[string]interface{}
	err := json.Unmarshal(respRaw, &result)
	if err != nil {
		return err
	}
	switch result["type"] {
	case "update":
		var p currency.Pair
		var a asset.Item
		_, ok := result["market"]
		if ok {
			p = currency.NewPairFromString(result["market"].(string))
			a, err = f.GetPairAssetType(p)
			if err != nil {
				return err
			}
		}
		switch result["channel"] {
		case "ticker":
			var resultData WsTickerDataStore
			err = json.Unmarshal(respRaw, &resultData)
			if err != nil {
				return err
			}
			f.Websocket.DataHandler <- &ticker.Price{
				ExchangeName: f.Name,
				Bid:          resultData.Ticker.Bid,
				Ask:          resultData.Ticker.Ask,
				Last:         resultData.Ticker.Last,
				LastUpdated:  timestampFromFloat64(resultData.Ticker.Time),
				Pair:         p,
				AssetType:    a,
			}
		case "orderbook":
			var resultData WsOrderbookDataStore
			err = json.Unmarshal(respRaw, &resultData)
			if err != nil {
				return err
			}
			var newOB orderbook.Base
			for x := range resultData.OBData.Asks {
				newOB.Asks = append(newOB.Asks, orderbook.Item{Price: resultData.OBData.Asks[x][0],
					Amount: resultData.OBData.Asks[x][1],
				})
			}
			for y := range resultData.OBData.Bids {
				newOB.Bids = append(newOB.Bids, orderbook.Item{Price: resultData.OBData.Bids[y][0],
					Amount: resultData.OBData.Bids[y][1],
				})
			}
			newOB.Pair = p
			newOB.AssetType = a
			newOB.ExchangeName = f.Name
			err = f.Websocket.Orderbook.Update(&wsorderbook.WebsocketOrderbookUpdate{
				Asset: newOB.AssetType,
				Bids:  newOB.Bids,
				Asks:  newOB.Asks,
				Pair:  p,
			})
			if err != nil {
				return err
			}
			f.Websocket.DataHandler <- wshandler.WebsocketOrderbookUpdate{
				Pair:     p,
				Asset:    a,
				Exchange: f.Name,
			}
		case "trades":
			var resultData WsTradeDataStore
			err = json.Unmarshal(respRaw, &resultData)
			if err != nil {
				return err
			}
			for z := range resultData.TradeData {
				var oSide order.Side
				oSide, err = order.StringToOrderSide(resultData.TradeData[z].Side)
				if err != nil {
					f.Websocket.DataHandler <- order.ClassificationError{
						Exchange: f.Name,
						Err:      err,
					}
				}
				f.Websocket.DataHandler <- wshandler.TradeData{
					Timestamp:    resultData.TradeData[z].Time,
					CurrencyPair: p,
					AssetType:    a,
					Exchange:     f.Name,
					Price:        resultData.TradeData[z].Price,
					Amount:       resultData.TradeData[z].Size,
					Side:         oSide,
				}
			}
		case "orders":
			var resultData WsOrderDataStore
			err = json.Unmarshal(respRaw, &resultData)
			if err != nil {
				return err
			}
			pair := currency.NewPairFromString(resultData.OrderData.Market)
			var assetType asset.Item
			assetType, err = f.GetPairAssetType(pair)
			if err != nil {
				return err
			}
			var oSide order.Side
			oSide, err = order.StringToOrderSide(resultData.OrderData.Side)
			if err != nil {
				f.Websocket.DataHandler <- order.ClassificationError{
					Exchange: f.Name,
					Err:      err,
				}
			}
			f.Websocket.DataHandler <- wshandler.TradeData{CurrencyPair: pair,
				AssetType: assetType,
				Exchange:  f.Name,
				Price:     resultData.OrderData.Price,
				Amount:    resultData.OrderData.Size,
				Side:      oSide,
			}
		case "fills":
			var resultData WsFillsDataStore
			err = json.Unmarshal(respRaw, &resultData)
			if err != nil {
				return err
			}
			f.Websocket.DataHandler <- resultData.FillsData
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
