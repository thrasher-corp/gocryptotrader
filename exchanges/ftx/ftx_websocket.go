package ftx

import (
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"net/http"
	"strconv"
	"strings"
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
	wsMarkets         = "markets"
	wsFills           = "fills"
	wsOrders          = "orders"
	wsUpdate          = "update"
	wsPartial         = "partial"
	subscribe         = "subscribe"
	unsubscribe       = "unsubscribe"
)

var obSuccess = make(map[currency.Pair]bool)

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
	f.GenerateDefaultSubscriptions()
	go f.wsReadData()
	if f.GetAuthenticatedAPISupport(exchange.WebsocketAuthentication) {
		err := f.WsAuth()
		if err != nil {
			f.Websocket.DataHandler <- err
			f.Websocket.SetCanUseAuthenticatedEndpoints(false)
		}
		f.GenerateAuthSubscriptions()
	}
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
	case wsFills, wsOrders, wsMarkets:
		sub.Operation = subscribe
		sub.Channel = channelToSubscribe.Channel
	default:
		a, err := f.GetPairAssetType(channelToSubscribe.Currency)
		if err != nil {
			return err
		}
		sub.Operation = subscribe
		sub.Channel = channelToSubscribe.Channel
		sub.Market = f.FormatExchangeCurrency(channelToSubscribe.Currency, a).String()
	}
	return f.WebsocketConn.SendJSONMessage(sub)
}

// GenerateDefaultSubscriptions generates default subscription
func (f *FTX) GenerateDefaultSubscriptions() {
	var subscriptions []wshandler.WebsocketChannelSubscription
	subscriptions = append(subscriptions, wshandler.WebsocketChannelSubscription{
		Channel: wsMarkets,
	})
	var channels = []string{wsTicker, wsTrades, wsOrderbook}
	for a := range f.CurrencyPairs.AssetTypes {
		pairs := f.GetEnabledPairs(f.CurrencyPairs.AssetTypes[a])
		for z := range pairs {
			newPair := currency.NewPairWithDelimiter(pairs[z].Base.String(), pairs[z].Quote.String(), "-")
			for x := range channels {
				subscriptions = append(subscriptions, wshandler.WebsocketChannelSubscription{
					Channel:  channels[x],
					Currency: newPair,
				})
			}
		}
	}
	f.Websocket.SubscribeToChannels(subscriptions)
}

// GenerateAuthSubscriptions generates default subscription
func (f *FTX) GenerateAuthSubscriptions() {
	var subscriptions []wshandler.WebsocketChannelSubscription
	var channels = []string{wsOrders, wsFills}
	for x := range channels {
		subscriptions = append(subscriptions, wshandler.WebsocketChannelSubscription{
			Channel: channels[x],
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
	case wsUpdate:
		var p currency.Pair
		var a asset.Item
		market, ok := result["market"]
		if ok {
			p = currency.NewPairFromString(market.(string))
			a, err = f.GetPairAssetType(p)
			if err != nil {
				return err
			}
		}
		switch result["channel"] {
		case wsTicker:
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
		case wsOrderbook:
			var resultData WsOrderbookDataStore
			err = json.Unmarshal(respRaw, &resultData)
			if err != nil {
				return err
			}
			if len(resultData.OBData.Asks) == 0 && len(resultData.OBData.Bids) == 0 {
				return nil
			}
			err = f.WsProcessUpdateOB(&resultData.OBData, p, a)
			if err != nil {
				f.wsResubToOB(p)
				return err
			}
		case wsTrades:
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
		case wsOrders:
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
			var resp order.Detail
			resp.Side = oSide
			resp.Amount = resultData.OrderData.Size
			resp.AssetType = assetType
			resp.ClientOrderID = resultData.OrderData.ClientID
			resp.Exchange = f.Name
			resp.ExecutedAmount = resultData.OrderData.FilledSize
			resp.ID = strconv.FormatInt(resultData.OrderData.ID, 10)
			resp.Pair = pair
			resp.RemainingAmount = resultData.OrderData.Size - resultData.OrderData.FilledSize
			var orderVars OrderVars
			orderVars, err = f.compatibleOrderVars(resultData.OrderData.Side,
				resultData.OrderData.Status,
				resultData.OrderData.OrderType,
				resultData.OrderData.FilledSize,
				resultData.OrderData.Size,
				resultData.OrderData.AvgFillPrice)
			if err != nil {
				return err
			}
			resp.Status = orderVars.Status
			resp.Side = orderVars.Side
			resp.Type = orderVars.OrderType
			resp.Fee = orderVars.Fee
			f.Websocket.DataHandler <- resp
		case wsFills:
			var resultData WsFillsDataStore
			err = json.Unmarshal(respRaw, &resultData)
			if err != nil {
				return err
			}
			f.Websocket.DataHandler <- resultData.FillsData
		default:
			f.Websocket.DataHandler <- wshandler.UnhandledMessageWarning{Message: f.Name + wshandler.UnhandledMessage + string(respRaw)}
		}
	case wsPartial:
		switch result["channel"] {
		case "orderbook":
			var p currency.Pair
			var a asset.Item
			market, ok := result["market"]
			if ok {
				p = currency.NewPairFromString(market.(string))
				a, err = f.GetPairAssetType(p)
				if err != nil {
					return err
				}
			}
			var resultData WsOrderbookDataStore
			err = json.Unmarshal(respRaw, &resultData)
			if err != nil {
				return err
			}
			err = f.WsProcessPartialOB(&resultData.OBData, p, a)
			if err != nil {
				f.wsResubToOB(p)
				return err
			}
			// reset obchecksum failure blockage for pair
			delete(obSuccess, p)
		case wsMarkets:
			var resultData WSMarkets
			err = json.Unmarshal(respRaw, &resultData)
			if err != nil {
				return err
			}
			f.Websocket.DataHandler <- resultData.Data
		}
	case "error":
		f.Websocket.DataHandler <- wshandler.UnhandledMessageWarning{Message: f.Name + wshandler.UnhandledMessage + string(respRaw)}
	}
	return nil
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (f *FTX) Unsubscribe(channelToSubscribe wshandler.WebsocketChannelSubscription) error {
	var unSub WsSub
	a, err := f.GetPairAssetType(channelToSubscribe.Currency)
	if err != nil {
		return err
	}
	unSub.Operation = unsubscribe
	unSub.Channel = channelToSubscribe.Channel
	unSub.Market = f.FormatExchangeCurrency(channelToSubscribe.Currency, a).String()
	return f.WebsocketConn.SendJSONMessage(unSub)
}

// WsProcessUpdateOB processes an update on the orderbook
func (f *FTX) WsProcessUpdateOB(data *WsOrderbookData, p currency.Pair, a asset.Item) error {
	update := wsorderbook.WebsocketOrderbookUpdate{
		Asset:      a,
		Pair:       p,
		UpdateTime: timestampFromFloat64(data.Time),
	}

	var err error
	for x := range data.Bids {
		update.Bids = append(update.Bids, orderbook.Item{
			Price:  data.Bids[x][0],
			Amount: data.Bids[x][1],
		})
	}
	for x := range data.Asks {
		update.Asks = append(update.Asks, orderbook.Item{
			Price:  data.Asks[x][0],
			Amount: data.Asks[x][1],
		})
	}

	err = f.Websocket.Orderbook.Update(&update)
	if err != nil {
		return err
	}

	updatedOb := f.Websocket.Orderbook.GetOrderbook(p, a)
	checksum := f.CalcUpdateOBChecksum(updatedOb)

	if checksum != data.Checksum {
		log.Warnf(log.ExchangeSys, "%s checksum failure for item %s",
			f.Name,
			p)
		return errors.New("checksum failed")
	}
	f.Websocket.DataHandler <- wshandler.WebsocketOrderbookUpdate{
		Exchange: f.Name,
		Asset:    a,
		Pair:     p,
	}

	return nil
}

func (f *FTX) wsResubToOB(p currency.Pair) {
	if ok := obSuccess[p]; ok {
		return
	}

	obSuccess[p] = true

	channelToResubscribe := wshandler.WebsocketChannelSubscription{
		Channel:  wsOrderbook,
		Currency: p,
	}
	f.Websocket.ResubscribeToChannel(channelToResubscribe)
}

// WsProcessPartialOB creates an OB from websocket data
func (f *FTX) WsProcessPartialOB(data *WsOrderbookData, p currency.Pair, a asset.Item) error {
	signedChecksum := f.CalcPartialOBChecksum(data)
	if signedChecksum != data.Checksum {
		return fmt.Errorf("%s channel: %s. Orderbook partial for %v checksum invalid",
			f.Name,
			a,
			p)
	}
	var bids, asks []orderbook.Item
	for x := range data.Bids {
		bids = append(bids, orderbook.Item{
			Price:  data.Bids[x][0],
			Amount: data.Bids[x][1],
		})
	}
	for x := range data.Asks {
		asks = append(asks, orderbook.Item{
			Price:  data.Asks[x][0],
			Amount: data.Asks[x][1],
		})
	}

	newOrderBook := orderbook.Base{
		Asks:         asks,
		Bids:         bids,
		AssetType:    a,
		LastUpdated:  timestampFromFloat64(data.Time),
		Pair:         p,
		ExchangeName: f.Name,
	}

	if err := f.Websocket.Orderbook.LoadSnapshot(&newOrderBook); err != nil {
		return err
	}

	f.Websocket.DataHandler <- wshandler.WebsocketOrderbookUpdate{
		Exchange: f.Name,
		Asset:    a,
		Pair:     p,
	}
	return nil
}

// CalcPartialOBChecksum calculates checksum of partial OB data received from WS
func (f *FTX) CalcPartialOBChecksum(data *WsOrderbookData) int {
	var checksum strings.Builder
	var price, amount string
	for i := 0; i < 100; i++ {
		if len(data.Bids)-1 >= i {
			price = strconv.FormatFloat(data.Bids[i][0], 'f', -1, 64)
			if strings.IndexByte(price, '.') == -1 {
				price += ".0"
			}
			amount = strconv.FormatFloat(data.Bids[i][1], 'f', -1, 64)
			if strings.IndexByte(amount, '.') == -1 {
				amount += ".0"
			}
			checksum.WriteString(price + ":" + amount + ":")
		}
		if len(data.Asks)-1 >= i {
			price = strconv.FormatFloat(data.Asks[i][0], 'f', -1, 64)
			if strings.IndexByte(price, '.') == -1 {
				price += ".0"
			}
			amount = strconv.FormatFloat(data.Asks[i][1], 'f', -1, 64)
			if strings.IndexByte(amount, '.') == -1 {
				amount += ".0"
			}
			checksum.WriteString(price + ":" + amount + ":")
		}
	}
	checksumStr := strings.TrimSuffix(checksum.String(), ":")
	return int(crc32.ChecksumIEEE([]byte(checksumStr)))
}

// CalcUpdateOBChecksum calculates checksum of update OB data received from WS
func (f *FTX) CalcUpdateOBChecksum(data *orderbook.Base) int {
	var checksum strings.Builder
	var price, amount string
	for i := 0; i < 100; i++ {
		if len(data.Bids)-1 >= i {
			price = strconv.FormatFloat(data.Bids[i].Price, 'f', -1, 64)
			if strings.IndexByte(price, '.') == -1 {
				price += ".0"
			}
			amount = strconv.FormatFloat(data.Bids[i].Amount, 'f', -1, 64)
			if strings.IndexByte(amount, '.') == -1 {
				amount += ".0"
			}
			checksum.WriteString(price + ":" + amount + ":")
		}
		if len(data.Asks)-1 >= i {
			price = strconv.FormatFloat(data.Asks[i].Price, 'f', -1, 64)
			if strings.IndexByte(price, '.') == -1 {
				price += ".0"
			}
			amount = strconv.FormatFloat(data.Asks[i].Amount, 'f', -1, 64)
			if strings.IndexByte(amount, '.') == -1 {
				amount += ".0"
			}
			checksum.WriteString(price + ":" + amount + ":")
		}
	}
	checksumStr := strings.TrimSuffix(checksum.String(), ":")
	return int(crc32.ChecksumIEEE([]byte(checksumStr)))
}
