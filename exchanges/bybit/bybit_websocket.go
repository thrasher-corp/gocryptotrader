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
	wsSpotPrivate       = "spot/ws"
	bybitWebsocketTimer = 30 * time.Second
	wsOrderbook         = "depth"
	wsTicker            = "bookTicker"
	wsTrades            = "trade"
	wsMarkets           = "kline"

	wsAccountInfoStr = "outboundAccountInfo"
	wsOrderStr       = "executionReport"
	wsOrderFilledStr = "ticketInfo"

	sub    = "sub"    // event for subscribe
	cancel = "cancel" // event for unsubscribe
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

	return nil
}

func readAuthThings(wow stream.Connection) {
	for {
		resp := wow.ReadMessage()
		fmt.Println("ZOOM: ", string(resp.Raw))
	}
}

// WsAuth sends an authentication message to receive auth data
func (by *Bybit) WsAuth() error {
	var dialer websocket.Dialer
	err := by.Websocket.AuthConn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}
	go readAuthThings(by.Websocket.AuthConn)

	intNonce := (time.Now().Unix() + 1) * 1000
	strNonce := strconv.FormatInt(intNonce, 10)
	hmac, err := crypto.GetHMAC(
		crypto.HashSHA256,
		[]byte("GET/realtime"+strNonce),
		[]byte(by.API.Credentials.Secret),
	)
	if err != nil {
		return err
	}
	sign := crypto.HexEncodeToString(hmac)
	req := Authenticate{
		Operation: "auth",
		Args:      []interface{}{by.API.Credentials.Key, intNonce, sign},
	}
	return by.Websocket.AuthConn.SendJSONMessage(req)
}

// Subscribe sends a websocket message to receive data from the channel
func (by *Bybit) Subscribe(channelsToSubscribe []stream.ChannelSubscription) error {
	var errs common.Errors
	for i := range channelsToSubscribe {
		var subReq WsReq
		subReq.Topic = channelsToSubscribe[i].Channel
		subReq.Event = sub

		formattedPair, err := by.FormatExchangeCurrency(channelsToSubscribe[i].Currency, asset.Spot)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if channelsToSubscribe[i].Channel == wsMarkets {
			subReq.Parameters = WsParams{
				Symbol:    formattedPair.String(),
				IsBinary:  true,
				KlineType: "1m",
			}
		} else {
			subReq.Parameters = WsParams{
				Symbol:   formattedPair.String(),
				IsBinary: true,
			}
		}
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

		formattedPair, err := by.FormatExchangeCurrency(channelsToUnsubscribe[i].Currency, asset.Spot)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		unSub.Parameters = WsParams{
			Symbol: formattedPair.String(),
		}
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
	pairs, err := by.GetEnabledPairs(asset.Spot)
	if err != nil {
		return nil, err
	}
	for z := range pairs {
		for x := range channels {
			subscriptions = append(subscriptions,
				stream.ChannelSubscription{
					Channel:  channels[x],
					Currency: pairs[z],
					Asset:    asset.Spot,
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

			p, err := by.extractCurrencyPair(data.Symbol, asset.Spot)
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
				AssetType:       asset.Spot,
				Date:            data.OrderCreationTime.Time(),
				Pair:            p,
				ClientOrderID:   data.ClientOrderID,
			}
			return nil
		case wsOrderFilledStr:
			// already handled in wsOrderStr case
			return nil
		}
	}

	t, ok := multiStreamData["topic"].(string)
	if !ok {
		log.Errorf(log.ExchangeSys, "%s Received unhandle message on websocket: %s\n", by.Name, multiStreamData)
		return nil
	}

	switch t {
	case wsOrderbook:
		var data WsOrderbook
		err := json.Unmarshal(respRaw, &data)
		if err != nil {
			return err
		}
		p, err := by.extractCurrencyPair(data.OBData.Symbol, asset.Spot)
		if err != nil {
			return err
		}

		err = by.wsUpdateOrderbook(&data.OBData, p, asset.Spot)
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

		p, err := by.extractCurrencyPair(data.Parameters.Symbol, asset.Spot)
		if err != nil {
			return err
		}

		side := order.Sell
		if data.TradeData.Side {
			side = order.Buy
		}

		return trade.AddTradesToBuffer(by.Name, trade.Data{
			Timestamp:    data.TradeData.Time.Time(),
			CurrencyPair: p,
			AssetType:    asset.Spot,
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

		p, err := by.extractCurrencyPair(data.Ticker.Symbol, asset.Spot)
		if err != nil {
			return err
		}

		by.Websocket.DataHandler <- &ticker.Price{
			ExchangeName: by.Name,
			Bid:          data.Ticker.Bid,
			Ask:          data.Ticker.Ask,
			LastUpdated:  data.Ticker.Time.Time(),
			AssetType:    asset.Spot,
			Pair:         p,
		}

	case wsMarkets:
		var data KlineStream
		err := json.Unmarshal(respRaw, &data)
		if err != nil {
			return err
		}

		p, err := by.extractCurrencyPair(data.Kline.Symbol, asset.Spot)
		if err != nil {
			return err
		}

		by.Websocket.DataHandler <- stream.KlineData{
			Pair:       p,
			AssetType:  asset.Spot,
			Exchange:   by.Name,
			StartTime:  data.Kline.StartTime.Time(),
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

	return nil
}

func (by *Bybit) wsUpdateOrderbook(update *WsOrderbookData, p currency.Pair, assetType asset.Item) error {
	if update == nil || (len(update.Asks) == 0 && len(update.Bids) == 0) {
		return errors.New("no orderbook data")
	}
	asks := make([]orderbook.Item, len(update.Asks))
	for i := range update.Asks {
		target, err := strconv.ParseFloat(update.Asks[i][0], 64)
		if err != nil {
			return err
		}
		amount, err := strconv.ParseFloat(update.Asks[i][1], 64)
		if err != nil {
			return err
		}
		asks[i] = orderbook.Item{Price: target, Amount: amount}
	}
	bids := make([]orderbook.Item, len(update.Bids))
	for i := range update.Bids {
		target, err := strconv.ParseFloat(update.Bids[i][0], 64)
		if err != nil {
			return err
		}
		amount, err := strconv.ParseFloat(update.Bids[i][1], 64)
		if err != nil {
			return err
		}
		bids[i] = orderbook.Item{Price: target, Amount: amount}
	}
	return by.Websocket.Orderbook.LoadSnapshot(&orderbook.Base{
		Bids:            bids,
		Asks:            asks,
		Pair:            p,
		LastUpdated:     update.Time.Time(),
		Asset:           assetType,
		Exchange:        by.Name,
		VerifyOrderbook: by.CanVerifyOrderbook,
	})
}
