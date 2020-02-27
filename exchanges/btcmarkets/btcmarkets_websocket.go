package btcmarkets

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
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
	btcMarketsWSURL = "wss://socket.btcmarkets.net/v2"
)

// WsConnect connects to a websocket feed
func (b *BTCMarkets) WsConnect() error {
	if !b.Websocket.IsEnabled() || !b.IsEnabled() {
		return errors.New(wshandler.WebsocketNotEnabled)
	}
	var dialer websocket.Dialer
	err := b.WebsocketConn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}
	if b.Verbose {
		log.Debugf(log.ExchangeSys, "%s Connected to Websocket.\n", b.Name)
	}
	go b.wsReadData()
	if b.GetAuthenticatedAPISupport(exchange.WebsocketAuthentication) {
		b.createChannels()
	}
	b.generateDefaultSubscriptions()
	return nil
}

// wsReadData receives and passes on websocket messages for processing
func (b *BTCMarkets) wsReadData() {
	b.Websocket.Wg.Add(1)
	defer func() {
		b.Websocket.Wg.Done()
	}()

	for {
		select {
		case <-b.Websocket.ShutdownC:
			return
		default:
			resp, err := b.WebsocketConn.ReadMessage()
			if err != nil {
				b.Websocket.ReadMessageErrors <- err
				return
			}
			b.Websocket.TrafficAlert <- struct{}{}
			err = b.wsHandleData(resp.Raw)
			if err != nil {
				b.Websocket.DataHandler <- err
			}
		}
	}
}

func (b *BTCMarkets) wsHandleData(respRaw []byte) error {
	var wsResponse WsMessageType
	err := json.Unmarshal(respRaw, &wsResponse)
	if err != nil {
		return err
	}
	switch wsResponse.MessageType {
	case heartbeat:
		if b.Verbose {
			log.Debugf(log.ExchangeSys, "%v - Websocket heartbeat received %s", b.Name, respRaw)
		}
	case wsOB:
		var ob WsOrderbook
		err := json.Unmarshal(respRaw, &ob)
		if err != nil {
			return err
		}

		p := currency.NewPairFromString(ob.Currency)
		var bids, asks []orderbook.Item
		for x := range ob.Bids {
			var price, amount float64
			price, err = strconv.ParseFloat(ob.Bids[x][0].(string), 64)
			if err != nil {
				return err
			}
			amount, err = strconv.ParseFloat(ob.Bids[x][1].(string), 64)
			if err != nil {
				return err
			}
			bids = append(bids, orderbook.Item{
				Amount:     amount,
				Price:      price,
				OrderCount: int64(ob.Bids[x][2].(float64)),
			})
		}
		for x := range ob.Asks {
			var price, amount float64
			price, err = strconv.ParseFloat(ob.Asks[x][0].(string), 64)
			if err != nil {
				return err
			}
			amount, err = strconv.ParseFloat(ob.Asks[x][1].(string), 64)
			if err != nil {
				return err
			}
			asks = append(asks, orderbook.Item{
				Amount:     amount,
				Price:      price,
				OrderCount: int64(ob.Asks[x][2].(float64)),
			})
		}
		if ob.Snapshot {
			err = b.Websocket.Orderbook.LoadSnapshot(&orderbook.Base{
				Pair:         p,
				Bids:         bids,
				Asks:         asks,
				LastUpdated:  ob.Timestamp,
				AssetType:    asset.Spot,
				ExchangeName: b.Name,
			})
		} else {
			err = b.Websocket.Orderbook.Update(&wsorderbook.WebsocketOrderbookUpdate{
				UpdateTime: ob.Timestamp,
				Asset:      asset.Spot,
				Bids:       bids,
				Asks:       asks,
				Pair:       p,
			})
		}

		if err != nil {
			return err
		}
		b.Websocket.DataHandler <- wshandler.WebsocketOrderbookUpdate{
			Pair:     p,
			Asset:    asset.Spot,
			Exchange: b.Name,
		}
	case trade:
		var trade WsTrade
		err := json.Unmarshal(respRaw, &trade)
		if err != nil {
			return err
		}
		p := currency.NewPairFromString(trade.Currency)
		b.Websocket.DataHandler <- wshandler.TradeData{
			Timestamp:    trade.Timestamp,
			CurrencyPair: p,
			AssetType:    asset.Spot,
			Exchange:     b.Name,
			Price:        trade.Price,
			Amount:       trade.Volume,
			Side:         order.UnknownSide,
			EventType:    order.UnknownType,
		}
	case tick:
		var tick WsTick
		err := json.Unmarshal(respRaw, &tick)
		if err != nil {
			return err
		}

		p := currency.NewPairFromString(tick.Currency)

		b.Websocket.DataHandler <- &ticker.Price{
			ExchangeName: b.Name,
			Volume:       tick.Volume,
			High:         tick.High24,
			Low:          tick.Low24h,
			Bid:          tick.Bid,
			Ask:          tick.Ask,
			Last:         tick.Last,
			LastUpdated:  tick.Timestamp,
			AssetType:    asset.Spot,
			Pair:         p,
		}
	case fundChange:
		var transferData WsFundTransfer
		err := json.Unmarshal(respRaw, &transferData)
		if err != nil {
			return err
		}
		b.Websocket.DataHandler <- transferData
	case orderChange:
		var orderData WsOrderChange
		err := json.Unmarshal(respRaw, &orderData)
		if err != nil {
			return err
		}
		originalAmount := orderData.OpenVolume
		var price float64
		var trades []order.TradeHistory
		var orderID = strconv.FormatInt(orderData.OrderID, 10)
		for x := range orderData.Trades {
			var isMaker bool
			if orderData.Trades[x].LiquidityType == "Maker" {
				isMaker = true
			}
			trades = append(trades, order.TradeHistory{
				Price:    orderData.Trades[x].Price,
				Amount:   orderData.Trades[x].Volume,
				Fee:      orderData.Trades[x].Fee,
				Exchange: b.Name,
				TID:      strconv.FormatInt(orderData.Trades[x].TradeID, 10),
				IsMaker:  isMaker,
			})
			price = orderData.Trades[x].Price
			originalAmount += orderData.Trades[x].Volume
		}
		oType, err := order.StringToOrderType(orderData.OrderType)
		if err != nil {
			b.Websocket.DataHandler <- order.ClassificationError{
				Exchange: b.Name,
				OrderID:  orderID,
				Err:      err,
			}
		}
		oSide, err := order.StringToOrderSide(orderData.Side)
		if err != nil {
			b.Websocket.DataHandler <- order.ClassificationError{
				Exchange: b.Name,
				OrderID:  orderID,
				Err:      err,
			}
		}
		oStatus, err := order.StringToOrderStatus(orderData.Status)
		if err != nil {
			b.Websocket.DataHandler <- order.ClassificationError{
				Exchange: b.Name,
				OrderID:  orderID,
				Err:      err,
			}
		}
		p := currency.NewPairFromString(orderData.MarketID)
		var a asset.Item
		a, err = b.GetPairAssetType(p)
		if err != nil {
			return err
		}
		b.Websocket.DataHandler <- &order.Detail{
			Price:           price,
			Amount:          originalAmount,
			RemainingAmount: orderData.OpenVolume,
			Exchange:        b.Name,
			ID:              orderID,
			ClientID:        b.API.Credentials.ClientID,
			Type:            oType,
			Side:            oSide,
			Status:          oStatus,
			AssetType:       a,
			Date:            orderData.Timestamp,
			Trades:          trades,
			Pair:            p,
		}
	case "error":
		var wsErr WsError
		err := json.Unmarshal(respRaw, &wsErr)
		if err != nil {
			return err
		}
		return fmt.Errorf("%v websocket error. Code: %v Message: %v", b.Name, wsErr.Code, wsErr.Message)
	default:
		b.Websocket.DataHandler <- wshandler.UnhandledMessageWarning{Message: b.Name + wshandler.UnhandledMessage + string(respRaw)}
		return nil
	}
	return nil
}

func (b *BTCMarkets) generateDefaultSubscriptions() {
	var channels = []string{tick, trade, wsOB}
	enabledCurrencies := b.GetEnabledPairs(asset.Spot)
	var subscriptions []wshandler.WebsocketChannelSubscription
	for i := range channels {
		for j := range enabledCurrencies {
			subscriptions = append(subscriptions, wshandler.WebsocketChannelSubscription{
				Channel:  channels[i],
				Currency: enabledCurrencies[j],
			})
		}
	}
	b.Websocket.SubscribeToChannels(subscriptions)
}

// Subscribe sends a websocket message to receive data from the channel
func (b *BTCMarkets) Subscribe(channelToSubscribe wshandler.WebsocketChannelSubscription) error {
	unauthChannels := []string{tick, trade, wsOB}
	authChannels := []string{fundChange, heartbeat, orderChange}
	switch {
	case common.StringDataCompare(unauthChannels, channelToSubscribe.Channel):
		req := WsSubscribe{
			MarketIDs:   []string{b.FormatExchangeCurrency(channelToSubscribe.Currency, asset.Spot).String()},
			Channels:    []string{channelToSubscribe.Channel},
			MessageType: subscribe,
		}
		err := b.WebsocketConn.SendJSONMessage(req)
		if err != nil {
			return err
		}
	case common.StringDataCompare(authChannels, channelToSubscribe.Channel):
		message, ok := channelToSubscribe.Params["AuthSub"].(WsAuthSubscribe)
		if !ok {
			return errors.New("invalid params data")
		}
		tempAuthData := b.generateAuthSubscriptions()
		message.Channels = append(message.Channels, channelToSubscribe.Channel, heartbeat)
		message.Key = tempAuthData.Key
		message.Signature = tempAuthData.Signature
		message.Timestamp = tempAuthData.Timestamp
		err := b.WebsocketConn.SendJSONMessage(message)
		if err != nil {
			return err
		}
	}
	return nil
}

// Login logs in allowing private ws events
func (b *BTCMarkets) generateAuthSubscriptions() WsAuthSubscribe {
	var authSubInfo WsAuthSubscribe
	signTime := strconv.FormatInt(time.Now().UTC().UnixNano()/1000000, 10)
	strToSign := "/users/self/subscribe" + "\n" + signTime
	tempSign := crypto.GetHMAC(crypto.HashSHA512,
		[]byte(strToSign),
		[]byte(b.API.Credentials.Secret))
	sign := crypto.Base64Encode(tempSign)
	authSubInfo.Key = b.API.Credentials.Key
	authSubInfo.Signature = sign
	authSubInfo.Timestamp = signTime
	return authSubInfo
}

// createChannels creates channels that need to be
func (b *BTCMarkets) createChannels() {
	tempChannels := []string{orderChange, fundChange}
	var channels []wshandler.WebsocketChannelSubscription
	pairArray := b.GetEnabledPairs(asset.Spot)
	for y := range tempChannels {
		for x := range pairArray {
			var authSub WsAuthSubscribe
			var channel wshandler.WebsocketChannelSubscription
			channel.Params = make(map[string]interface{})
			channel.Channel = tempChannels[y]
			authSub.MarketIDs = append(authSub.MarketIDs, b.FormatExchangeCurrency(pairArray[x], asset.Spot).String())
			authSub.MessageType = subscribe
			channel.Params["AuthSub"] = authSub
			channels = append(channels, channel)
		}
	}
	b.Websocket.SubscribeToChannels(channels)
}
