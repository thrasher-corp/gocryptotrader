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
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream/buffer"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	btcMarketsWSURL = "wss://socket.btcmarkets.net/v2"
)

// WsConnect connects to a websocket feed
func (b *BTCMarkets) WsConnect() error {
	if !b.Websocket.IsEnabled() || !b.IsEnabled() {
		return errors.New(stream.WebsocketNotEnabled)
	}
	var dialer websocket.Dialer
	err := b.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}
	if b.Verbose {
		log.Debugf(log.ExchangeSys, "%s Connected to Websocket.\n", b.Name)
	}
	go b.wsReadData()
	return nil
}

// wsReadData receives and passes on websocket messages for processing
func (b *BTCMarkets) wsReadData() {
	b.Websocket.Wg.Add(1)
	defer b.Websocket.Wg.Done()

	for {
		resp := b.Websocket.Conn.ReadMessage()
		if resp.Raw == nil {
			return
		}
		err := b.wsHandleData(resp.Raw)
		if err != nil {
			b.Websocket.DataHandler <- err
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

		p, err := currency.NewPairFromString(ob.Currency)
		if err != nil {
			return err
		}

		var bids, asks orderbook.Items
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
			bids.SortBids() // Alignment completely out, sort is needed.
			err = b.Websocket.Orderbook.LoadSnapshot(&orderbook.Base{
				Pair:            p,
				Bids:            bids,
				Asks:            asks,
				LastUpdated:     ob.Timestamp,
				Asset:           asset.Spot,
				Exchange:        b.Name,
				VerifyOrderbook: b.CanVerifyOrderbook,
			})
		} else {
			err = b.Websocket.Orderbook.Update(&buffer.Update{
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
	case tradeEndPoint:
		if !b.IsSaveTradeDataEnabled() {
			return nil
		}
		var t WsTrade
		err := json.Unmarshal(respRaw, &t)
		if err != nil {
			return err
		}

		p, err := currency.NewPairFromString(t.Currency)
		if err != nil {
			return err
		}

		side := order.Buy
		if t.Side == "Ask" {
			side = order.Sell
		}

		return trade.AddTradesToBuffer(b.Name, trade.Data{
			Timestamp:    t.Timestamp,
			CurrencyPair: p,
			AssetType:    asset.Spot,
			Exchange:     b.Name,
			Price:        t.Price,
			Amount:       t.Volume,
			Side:         side,
			TID:          strconv.FormatInt(t.TradeID, 10),
		})
	case tick:
		var tick WsTick
		err := json.Unmarshal(respRaw, &tick)
		if err != nil {
			return err
		}

		p, err := currency.NewPairFromString(tick.Currency)
		if err != nil {
			return err
		}

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

		p, err := currency.NewPairFromString(orderData.MarketID)
		if err != nil {
			b.Websocket.DataHandler <- order.ClassificationError{
				Exchange: b.Name,
				OrderID:  orderID,
				Err:      err,
			}
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
			AssetType:       asset.Spot,
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
		b.Websocket.DataHandler <- stream.UnhandledMessageWarning{Message: b.Name + stream.UnhandledMessage + string(respRaw)}
		return nil
	}
	return nil
}

func (b *BTCMarkets) generateDefaultSubscriptions() ([]stream.ChannelSubscription, error) {
	var channels = []string{wsOB, tick, tradeEndPoint}
	enabledCurrencies, err := b.GetEnabledPairs(asset.Spot)
	if err != nil {
		return nil, err
	}
	var subscriptions []stream.ChannelSubscription
	for i := range channels {
		for j := range enabledCurrencies {
			subscriptions = append(subscriptions, stream.ChannelSubscription{
				Channel:  channels[i],
				Currency: enabledCurrencies[j],
				Asset:    asset.Spot,
			})
		}
	}

	var authChannels = []string{fundChange, heartbeat, orderChange}
	if b.Websocket.CanUseAuthenticatedEndpoints() {
		for i := range authChannels {
			subscriptions = append(subscriptions, stream.ChannelSubscription{
				Channel: authChannels[i],
			})
		}
	}
	return subscriptions, nil
}

// Subscribe sends a websocket message to receive data from the channel
func (b *BTCMarkets) Subscribe(channelsToSubscribe []stream.ChannelSubscription) error {
	var authChannels = []string{fundChange, heartbeat, orderChange}

	var payload WsSubscribe
	payload.MessageType = subscribe

	for i := range channelsToSubscribe {
		payload.Channels = append(payload.Channels,
			channelsToSubscribe[i].Channel)

		if channelsToSubscribe[i].Currency.String() != "" {
			if !common.StringDataCompare(payload.MarketIDs,
				channelsToSubscribe[i].Currency.String()) {
				payload.MarketIDs = append(payload.MarketIDs,
					channelsToSubscribe[i].Currency.String())
			}
		}
	}

	for i := range authChannels {
		if !common.StringDataCompare(payload.Channels, authChannels[i]) {
			continue
		}
		signTime := strconv.FormatInt(time.Now().UTC().UnixNano()/1000000, 10)
		strToSign := "/users/self/subscribe" + "\n" + signTime
		tempSign := crypto.GetHMAC(crypto.HashSHA512,
			[]byte(strToSign),
			[]byte(b.API.Credentials.Secret))
		sign := crypto.Base64Encode(tempSign)
		payload.Key = b.API.Credentials.Key
		payload.Signature = sign
		payload.Timestamp = signTime
		break
	}

	err := b.Websocket.Conn.SendJSONMessage(payload)
	if err != nil {
		return err
	}
	b.Websocket.AddSuccessfulSubscriptions(channelsToSubscribe...)
	return nil
}
