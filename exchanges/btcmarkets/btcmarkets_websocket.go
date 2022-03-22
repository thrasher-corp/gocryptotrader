package btcmarkets

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"net/http"
	"strconv"
	"strings"
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

var (
	errTypeAssertionFailure = errors.New("type assertion failure")
	errChecksumFailure      = errors.New("crc32 checksum failure")
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

	b.Websocket.Wg.Add(1)
	go b.wsReadData()
	return nil
}

// wsReadData receives and passes on websocket messages for processing
func (b *BTCMarkets) wsReadData() {
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

// UnmarshalJSON implements the unmarshaler interface.
func (w *WebsocketOrderbook) UnmarshalJSON(data []byte) error {
	resp := make([][3]interface{}, len(data))
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}

	*w = WebsocketOrderbook(make(orderbook.Items, len(resp)))
	for x := range resp {
		sPrice, ok := resp[x][0].(string)
		if !ok {
			return fmt.Errorf("price string %w", errTypeAssertionFailure)
		}
		var price float64
		price, err = strconv.ParseFloat(sPrice, 64)
		if err != nil {
			return err
		}

		sAmount, ok := resp[x][1].(string)
		if !ok {
			return fmt.Errorf("amount string %w", errTypeAssertionFailure)
		}

		var amount float64
		amount, err = strconv.ParseFloat(sAmount, 64)
		if err != nil {
			return err
		}

		count, ok := resp[x][2].(float64)
		if !ok {
			return fmt.Errorf("count float64 %w", errTypeAssertionFailure)
		}

		(*w)[x] = orderbook.Item{
			Amount:     amount,
			Price:      price,
			OrderCount: int64(count),
		}
	}
	return nil
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

		if ob.Snapshot {
			err = b.Websocket.Orderbook.LoadSnapshot(&orderbook.Base{
				Pair:            ob.Currency,
				Bids:            orderbook.Items(ob.Bids),
				Asks:            orderbook.Items(ob.Asks),
				LastUpdated:     ob.Timestamp,
				LastUpdateID:    ob.SnapshotID,
				Asset:           asset.Spot,
				Exchange:        b.Name,
				VerifyOrderbook: b.CanVerifyOrderbook,
			})
		} else {
			err = b.Websocket.Orderbook.Update(&buffer.Update{
				UpdateTime: ob.Timestamp,
				UpdateID:   ob.SnapshotID,
				Asset:      asset.Spot,
				Bids:       orderbook.Items(ob.Bids),
				Asks:       orderbook.Items(ob.Asks),
				Pair:       ob.Currency,
				Checksum:   ob.Checksum,
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

		creds, err := b.GetCredentials(context.TODO())
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
			ClientID:        creds.ClientID,
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
	creds, err := b.GetCredentials(context.TODO())
	if err != nil {
		return err
	}
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
		signTime := strconv.FormatInt(time.Now().UnixMilli(), 10)
		strToSign := "/users/self/subscribe" + "\n" + signTime
		var tempSign []byte
		tempSign, err = crypto.GetHMAC(crypto.HashSHA512,
			[]byte(strToSign),
			[]byte(creds.Secret))
		if err != nil {
			return err
		}
		sign := crypto.Base64Encode(tempSign)
		payload.Key = creds.Key
		payload.Signature = sign
		payload.Timestamp = signTime
		break
	}

	err = b.Websocket.Conn.SendJSONMessage(payload)
	if err != nil {
		return err
	}
	b.Websocket.AddSuccessfulSubscriptions(channelsToSubscribe...)
	return nil
}

// checksum provides assurance on current in memory liquidity
func checksum(ob *orderbook.Base, checksum uint32) error {
	check := crc32.ChecksumIEEE([]byte(concat(ob.Bids) + concat(ob.Asks)))
	if check != checksum {
		return fmt.Errorf("%s %s %s ID: %v expected: %v but received: %v %w",
			ob.Exchange,
			ob.Pair,
			ob.Asset,
			ob.LastUpdateID,
			checksum,
			check,
			errChecksumFailure)
	}
	return nil
}

// concat concatenates price and amounts together for checksum processing
func concat(liquidity orderbook.Items) string {
	length := 10
	if len(liquidity) < 10 {
		length = len(liquidity)
	}
	var c string
	for x := 0; x < length; x++ {
		c += trim(liquidity[x].Price) + trim(liquidity[x].Amount)
	}
	return c
}

// trim turns value into string, removes the decimal point and all the leading
// zeros.
func trim(value float64) string {
	valstr := strconv.FormatFloat(value, 'f', -1, 64)
	valstr = strings.ReplaceAll(valstr, ".", "")
	valstr = strings.TrimLeft(valstr, "0")
	return valstr
}
