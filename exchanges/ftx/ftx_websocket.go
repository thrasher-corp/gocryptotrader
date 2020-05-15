package ftx

import (
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"net/http"
	"strconv"
	"strings"
	"sync"
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

var obMutex sync.Mutex

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
			obMutex.Lock()
			defer obMutex.Unlock()
			if len(resultData.OBData.Asks) == 0 && len(resultData.OBData.Bids) == 0 {
				return nil
			}
			err := f.WsProcessUpdateOB(&resultData.OBData, p, a)
			if err != nil {
				f.wsResubToOB(p)
				return err
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
			switch resultData.OrderData.Status {
			case strings.ToLower(order.New.String()):
				resp.Status = order.New
			case strings.ToLower(order.Open.String()):
				resp.Status = order.Open
			case closedStatus:
				if resultData.OrderData.FilledSize != 0 && resultData.OrderData.FilledSize != resultData.OrderData.Size {
					resp.Status = order.PartiallyCancelled
				}
				if resultData.OrderData.FilledSize == 0 {
					resp.Status = order.Cancelled
				}
				if resultData.OrderData.FilledSize == resultData.OrderData.Size {
					resp.Status = order.Filled
				}
			}
			var feeBuilder exchange.FeeBuilder
			feeBuilder.PurchasePrice = resultData.OrderData.AvgFillPrice
			feeBuilder.Amount = resultData.OrderData.Size
			resp.Type = order.Market
			if resultData.OrderData.OrderType == strings.ToLower(order.Limit.String()) {
				resp.Type = order.Limit
				feeBuilder.IsMaker = true
			}
			fee, err := f.GetFee(&feeBuilder)
			if err != nil {
				return err
			}
			resp.Fee = fee
			f.Websocket.DataHandler <- resp
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
	case "partial":
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
		var resultData WsOrderbookDataStore
		err = json.Unmarshal(respRaw, &resultData)
		if err != nil {
			return err
		}
		obMutex.Lock()
		obMutex.Unlock()
		f.WsProcessPartialOB(&resultData.OBData, p, a)
		if err != nil {
			f.wsResubToOB(p)
			return err
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
	unSub.Operation = "unsubscribe"
	unSub.Channel = channelToSubscribe.Channel
	unSub.Market = f.FormatExchangeCurrency(channelToSubscribe.Currency, a).String()
	return f.WebsocketConn.SendJSONMessage(unSub)
}

// CalcOBChecksum calculates checksum of our stored orderbook
func (f *FTX) CalcOBChecksum(orderbookData *orderbook.Base) int64 {
	var checksum strings.Builder
	for i := 0; i < 100; i++ {
		if i < len(orderbookData.Bids)-1 {
			price := strconv.FormatFloat(orderbookData.Bids[i].Price, 'f', -1, 64)
			amount := strconv.FormatFloat(orderbookData.Bids[i].Amount, 'f', -1, 64)
			checksum.WriteString(price + ":" + amount + ":")
		}
		if i < len(orderbookData.Asks)-1 {
			price := strconv.FormatFloat(orderbookData.Asks[i].Price, 'f', -1, 64)
			amount := strconv.FormatFloat(orderbookData.Asks[i].Amount, 'f', -1, 64)
			checksum.WriteString(price + ":" + amount + ":")
		}
	}
	checksumStr := strings.TrimSuffix(checksum.String(), ":")
	return int64(crc32.ChecksumIEEE([]byte(checksumStr)))
}

// AppendWsOrderbookItems adds websocket orderbook data bid/asks into an orderbook item array
func (f *FTX) AppendWsOrderbookItems(entries [][2]float64) ([]orderbook.Item, error) {
	var items []orderbook.Item
	for x := range entries {
		items = append(items, orderbook.Item{Amount: entries[x][1], Price: entries[x][0]})
	}
	return items, nil
}

// WsProcessUpdateOB processes an update on the orderbook
func (f *FTX) WsProcessUpdateOB(data *WsOrderbookData, p currency.Pair, a asset.Item) error {
	updateOB := wsorderbook.WebsocketOrderbookUpdate{
		Asset:      a,
		Pair:       p,
		UpdateTime: timestampFromFloat64(data.Time),
	}
	var err error
	updateOB.Asks, err = f.AppendWsOrderbookItems(data.Asks)
	if err != nil {
		return err
	}
	updateOB.Bids, err = f.AppendWsOrderbookItems(data.Bids)
	if err != nil {
		return err
	}
	err = f.Websocket.Orderbook.Update(&updateOB)
	if err != nil {
		return err
	}
	updatedOB := f.Websocket.Orderbook.GetOrderbook(p, a)
	checksum := f.CalcOBChecksum(updatedOB)
	if checksum != data.Checksum {
		log.Warnf(log.ExchangeSys, "%s checksum failure for item %s",
			f.Name,
			p.String())
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
	channelToResubscribe := wshandler.WebsocketChannelSubscription{
		Channel:  wsOrderbook,
		Currency: p,
	}
	f.Websocket.ResubscribeToChannel(channelToResubscribe)
}

// WsProcessPartialOB creates an OB from websocket data
func (f *FTX) WsProcessPartialOB(data *WsOrderbookData, p currency.Pair, a asset.Item) error {
	fmt.Printf("HELOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOO\n\n\n\n\n\n\n")
	signedChecksum := f.CalcPartialOBChecksum(data)
	if signedChecksum != data.Checksum {
		return fmt.Errorf("%s channel: %s. Orderbook partial for %v checksum invalid",
			f.Name,
			a,
			p)
	}
	if f.Verbose {
		log.Debugf(log.ExchangeSys,
			"%s passed checksum for market %s",
			f.Name,
			p)
	}

	asks, err := f.AppendWsOrderbookItems(data.Asks)
	if err != nil {
		return err
	}

	bids, err := f.AppendWsOrderbookItems(data.Bids)
	if err != nil {
		return err
	}

	newOrderBook := orderbook.Base{
		Asks:         asks,
		Bids:         bids,
		AssetType:    a,
		LastUpdated:  timestampFromFloat64(data.Time),
		Pair:         p,
		ExchangeName: f.Name,
	}

	err = f.Websocket.Orderbook.LoadSnapshot(&newOrderBook)
	if err != nil {
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
func (f *FTX) CalcPartialOBChecksum(data *WsOrderbookData) int64 {
	var checksum strings.Builder
	var price, amount string
	for i := 0; i < 100; i++ {
		if i < len(data.Bids)-1 {
			price = strconv.FormatFloat(data.Bids[i][0], 'f', -1, 64)
			amount = strconv.FormatFloat(data.Bids[i][1], 'f', -1, 64)
			checksum.WriteString(price + ":" + amount + ":")
		}
		if i < len(data.Asks)-1 {
			price = strconv.FormatFloat(data.Asks[i][0], 'f', -1, 64)
			amount = strconv.FormatFloat(data.Asks[i][1], 'f', -1, 64)
			checksum.WriteString(price + ":" + amount + ":")
		}
	}
	checksumStr := strings.TrimSuffix(checksum.String(), ":")
	return int64(crc32.ChecksumIEEE([]byte(checksumStr)))
}
