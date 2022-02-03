package bybit

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	bybitWSBaseURL      = "wss://stream.bybit.com/"
	wsSpotPublicTopicV2 = "spot/quote/ws/v2"
	bybitWebsocketTimer = 30 * time.Second
	wsOrderbook         = "depth"
	wsTicker            = "bookTicker"
	wsTrades            = "trade"
	wsMarkets           = "kline"

	wsAccountInfoStr = "outboundAccountInfo"
	wsOrderStr       = "executionReport"
	wsOrderFilledStr = "ticketInfo"

	wsUpdate  = "update"
	wsPartial = "partial"
	sub       = "sub"    // event for subscribe
	cancel    = "cancel" // event for unsubscribe
)

// WsConnect connects to a websocket feed
func (by *Bybit) WsConnect() error {
	if !by.Websocket.IsEnabled() || !by.IsEnabled() {
		return errors.New(stream.WebsocketNotEnabled)
	}
	var dialer websocket.Dialer
	err := by.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}
	by.Websocket.Conn.SetupPingHandler(stream.PingHandler{
		MessageType: websocket.PingMessage,
		Delay:       bybitWebsocketTimer,
	})
	if by.Verbose {
		log.Debugf(log.ExchangeSys, "%s Connected to Websocket.\n", by.Name)
	}

	go by.wsReadData()
	if by.GetAuthenticatedAPISupport(exchange.WebsocketAuthentication) {
		err = by.WsAuth()
		if err != nil {
			by.Websocket.DataHandler <- err
			by.Websocket.SetCanUseAuthenticatedEndpoints(false)
		}
	}

	subs, err := by.GenerateDefaultSubscriptions()
	if err != nil {
		return err
	}
	return by.Websocket.SubscribeToChannels(subs)
}

// WsAuth sends an authentication message to receive auth data
func (by *Bybit) WsAuth() error {
	intNonce := (time.Now().Unix() + 1) * 1000
	strNonce := strconv.FormatInt(intNonce, 10)
	hmac := crypto.GetHMAC(
		crypto.HashSHA256,
		[]byte("GET/realtime"+strNonce),
		[]byte(by.API.Credentials.Secret),
	)
	sign := crypto.HexEncodeToString(hmac)
	req := Authenticate{
		Operation: "auth",
		Args:      []string{by.API.Credentials.Key, strNonce, sign},
	}
	return by.Websocket.Conn.SendJSONMessage(req)
}

// Subscribe sends a websocket message to receive data from the channel
func (by *Bybit) Subscribe(channelsToSubscribe []stream.ChannelSubscription) error {
	var errs common.Errors
	for i := range channelsToSubscribe {
		var subReq WsReq
		subReq.Topic = channelsToSubscribe[i].Channel
		subReq.Event = sub

		a, err := by.GetPairAssetType(channelsToSubscribe[i].Currency)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		formattedPair, err := by.FormatExchangeCurrency(channelsToSubscribe[i].Currency, a)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		subReq.Symbol = formattedPair.String()
		err = by.Websocket.Conn.SendJSONMessage(subReq)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		by.Websocket.AddSuccessfulSubscriptions(channelsToSubscribe[i])
	}
	if errs != nil {
		return errs
	}
	return nil
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (by *Bybit) Unsubscribe(channelsToUnsubscribe []stream.ChannelSubscription) error {
	var errs common.Errors

	for i := range channelsToUnsubscribe {
		var unSub WsReq
		unSub.Event = cancel
		unSub.Topic = channelsToUnsubscribe[i].Channel

		a, err := by.GetPairAssetType(channelsToUnsubscribe[i].Currency)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		formattedPair, err := by.FormatExchangeCurrency(channelsToUnsubscribe[i].Currency, a)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		unSub.Symbol = formattedPair.String()
		err = by.Websocket.Conn.SendJSONMessage(unSub)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		by.Websocket.RemoveSuccessfulUnsubscriptions(channelsToUnsubscribe[i])
	}
	if errs != nil {
		return errs
	}
	return nil
}

// wsReadData gets and passes on websocket messages for processing
func (by *Bybit) wsReadData() {
	by.Websocket.Wg.Add(1)
	defer by.Websocket.Wg.Done()

	for {
		select {
		case <-by.Websocket.ShutdownC:
			return
		default:
			resp := by.Websocket.Conn.ReadMessage()
			if resp.Raw == nil {
				return
			}

			err := by.wsHandleData(resp.Raw)
			if err != nil {
				by.Websocket.DataHandler <- err
			}
		}
	}
}

// GenerateDefaultSubscriptions generates default subscription
func (by *Bybit) GenerateDefaultSubscriptions() ([]stream.ChannelSubscription, error) {
	var subscriptions []stream.ChannelSubscription
	var channels = []string{wsTicker, wsTrades, wsOrderbook}
	assets := by.GetAssetTypes(true)
	for a := range assets {
		pairs, err := by.GetEnabledPairs(assets[a])
		if err != nil {
			return nil, err
		}
		for z := range pairs {
			for x := range channels {
				subscriptions = append(subscriptions,
					stream.ChannelSubscription{
						Channel:  channels[x],
						Currency: pairs[z],
						Asset:    assets[a],
					})
			}
		}
	}
	if by.GetAuthenticatedAPISupport(exchange.WebsocketAuthentication) {
		var authchan = []string{wsAccountInfoStr, wsOrderStr, wsOrderFilledStr}
		for x := range authchan {
			subscriptions = append(subscriptions, stream.ChannelSubscription{
				Channel: authchan[x],
			})
		}
	}
	return subscriptions, nil
}

func stringToOrderStatus(status string) (order.Status, error) {
	switch status {
	case "NEW":
		return order.New, nil
	case "CANCELED":
		return order.Cancelled, nil
	case "REJECTED":
		return order.Rejected, nil
	case "TRADE":
		return order.PartiallyFilled, nil
	case "EXPIRED":
		return order.Expired, nil
	default:
		return order.UnknownStatus, errors.New(status + " not recognised as order status")
	}
}

func (by *Bybit) wsHandleData(respRaw []byte) error {
	var multiStreamData map[string]interface{}
	err := json.Unmarshal(respRaw, &multiStreamData)
	if err != nil {
		return err
	}

	if method, ok := multiStreamData["event"].(string); ok {
		if strings.EqualFold(method, sub) {
			return nil
		}
		if strings.EqualFold(method, cancel) {
			return nil
		}
	}

	if e, ok := multiStreamData["e"].(string); ok {
		switch e {
		case wsAccountInfoStr:
			var data wsAccountInfo
			err := json.Unmarshal(respRaw, &data)
			if err != nil {
				return fmt.Errorf("%v - Could not convert to outboundAccountInfo structure %s",
					by.Name,
					err)
			}
			by.Websocket.DataHandler <- data
			return nil
		case wsOrderStr:
			var data wsOrderUpdate
			err := json.Unmarshal(respRaw, &data)
			if err != nil {
				return fmt.Errorf("%v - Could not convert to executionReport structure %s",
					by.Name,
					err)
			}
			var orderID = strconv.FormatInt(data.OrderID, 10)
			oType, err := order.StringToOrderType(data.OrderType)
			if err != nil {
				by.Websocket.DataHandler <- order.ClassificationError{
					Exchange: by.Name,
					OrderID:  orderID,
					Err:      err,
				}
			}
			var oSide order.Side
			oSide, err = order.StringToOrderSide(data.Side)
			if err != nil {
				by.Websocket.DataHandler <- order.ClassificationError{
					Exchange: by.Name,
					OrderID:  orderID,
					Err:      err,
				}
			}
			var oStatus order.Status
			oStatus, err = stringToOrderStatus(data.OrderStatus)
			if err != nil {
				by.Websocket.DataHandler <- order.ClassificationError{
					Exchange: by.Name,
					OrderID:  orderID,
					Err:      err,
				}
			}
			var p currency.Pair
			var a asset.Item
			p, a, err = by.GetRequestFormattedPairAndAssetType(data.Symbol)
			if err != nil {
				return err
			}

			by.Websocket.DataHandler <- &order.Detail{
				Price:           data.Price,
				Amount:          data.Quantity,
				ExecutedAmount:  data.CumulativeFilledQuantity,
				RemainingAmount: data.Quantity - data.CumulativeFilledQuantity,
				Exchange:        by.Name,
				ID:              orderID,
				Type:            oType,
				Side:            oSide,
				Status:          oStatus,
				AssetType:       a,
				Date:            data.OrderCreationTime,
				Pair:            p,
				ClientOrderID:   data.ClientOrderID,
			}
			return nil
		case wsOrderFilledStr:
			// already handled in wsOrderStr case
			return nil
		}
	}

	if t, ok := multiStreamData["topic"].(string); ok {
		switch t {
		case wsOrderbook:
			var data WsOrderbook
			err := json.Unmarshal(respRaw, &data)
			if err != nil {
				return err
			}
			p, err := currency.NewPairFromString(data.OBData.Symbol)
			if err != nil {
				return err
			}

			a, err := by.GetPairAssetType(p)
			if err != nil {
				return err
			}

			err = by.wsUpdateOrderbook(data.OBData, p, a)
			if err != nil {
				return err
			}

		case wsTrades:
			if !by.IsSaveTradeDataEnabled() {
				return nil
			}
			var data WsTrade
			err := json.Unmarshal(respRaw, &data)
			if err != nil {
				return err
			}

			p, err := currency.NewPairFromString(data.Parameters.Symbol)
			if err != nil {
				return err
			}

			side := order.Sell
			if data.TradeData.Side {
				side = order.Buy
			}
			var a asset.Item
			a, err = by.GetPairAssetType(p)
			if err != nil {
				return err
			}
			return trade.AddTradesToBuffer(by.Name, trade.Data{
				Timestamp:    time.Unix(data.TradeData.Time, 0),
				CurrencyPair: p,
				AssetType:    a,
				Exchange:     by.Name,
				Price:        data.TradeData.Price,
				Amount:       data.TradeData.Size,
				Side:         side,
				TID:          data.TradeData.ID,
			})

		case wsTicker:
			var data WsSpotTicker
			err := json.Unmarshal(respRaw, &data)
			if err != nil {
				return err
			}

			p, err := currency.NewPairFromString(data.Ticker.Symbol)
			if err != nil {
				return err
			}

			by.Websocket.DataHandler <- &ticker.Price{
				ExchangeName: by.Name,
				Bid:          data.Ticker.Bid,
				Ask:          data.Ticker.Ask,
				LastUpdated:  time.Unix(data.Ticker.Time, 0),
				AssetType:    asset.Spot,
				Pair:         p,
			}

		case wsMarkets:
			var data KlineStream
			err := json.Unmarshal(respRaw, &data)
			if err != nil {
				return err
			}

			p, err := currency.NewPairFromString(data.Kline.Symbol)
			if err != nil {
				return err
			}

			a, err := by.GetPairAssetType(p)
			if err != nil {
				return err
			}
			by.Websocket.DataHandler <- stream.KlineData{
				Pair:       p,
				AssetType:  a,
				Exchange:   by.Name,
				StartTime:  data.Kline.StartTime,
				Interval:   data.Parameters.KlineType,
				OpenPrice:  data.Kline.OpenPrice,
				ClosePrice: data.Kline.ClosePrice,
				HighPrice:  data.Kline.HighPrice,
				LowPrice:   data.Kline.LowPrice,
				Volume:     data.Kline.Volume,
			}

		default:
			by.Websocket.DataHandler <- stream.UnhandledMessageWarning{Message: by.Name + stream.UnhandledMessage + string(respRaw)}
		}
	}

	return nil
}

func (by *Bybit) wsUpdateOrderbook(update WsOrderbookData, p currency.Pair, assetType asset.Item) error {
	if len(update.Asks) == 0 && len(update.Bids) == 0 {
		return errors.New("no orderbook data")
	}
	var asks, bids []orderbook.Item
	for i := range update.Asks {
		target, err := strconv.ParseFloat(update.Asks[i][0], 64)
		if err != nil {
			by.Websocket.DataHandler <- err
			continue
		}
		amount, err := strconv.ParseFloat(update.Asks[i][1], 64)
		if err != nil {
			by.Websocket.DataHandler <- err
			continue
		}
		asks = append(asks, orderbook.Item{Price: target, Amount: amount})
	}
	for i := range update.Bids {
		target, err := strconv.ParseFloat(update.Bids[i][0], 64)
		if err != nil {
			by.Websocket.DataHandler <- err
			continue
		}
		amount, err := strconv.ParseFloat(update.Bids[i][1], 64)
		if err != nil {
			by.Websocket.DataHandler <- err
			continue
		}

		bids = append(bids, orderbook.Item{Price: target, Amount: amount})
	}
	return by.Websocket.Orderbook.LoadSnapshot(&orderbook.Base{
		Bids:            bids,
		Asks:            asks,
		Pair:            p,
		LastUpdated:     time.Unix(update.Time, 0),
		Asset:           assetType,
		Exchange:        by.Name,
		VerifyOrderbook: by.CanVerifyOrderbook,
	})
}
