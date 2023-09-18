package bitstamp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	bitstampWSURL = "wss://ws.bitstamp.net" //nolint // gosec false positive
	hbInterval    = 8 * time.Second         // Connection monitor defaults to 10s inactivity
)

var hbMsg = []byte(`{"event":"bts:heartbeat"}`)

// WsConnect connects to a websocket feed
func (b *Bitstamp) WsConnect() error {
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
	b.Websocket.Conn.SetupPingHandler(stream.PingHandler{
		MessageType: websocket.TextMessage,
		Message:     hbMsg,
		Delay:       hbInterval,
	})
	err = b.seedOrderBook(context.TODO())
	if err != nil {
		b.Websocket.DataHandler <- err
	}

	b.Websocket.Wg.Add(1)
	go b.wsReadData()

	return nil
}

// wsReadData receives and passes on websocket messages for processing
func (b *Bitstamp) wsReadData() {
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

func (b *Bitstamp) wsHandleData(respRaw []byte) error {
	var wsResponse websocketResponse
	err := json.Unmarshal(respRaw, &wsResponse)
	if err != nil {
		return err
	}

	switch wsResponse.Event {
	case "bts:heartbeat":
		return nil
	case "bts:subscribe", "bts:subscription_succeeded":
		if b.Verbose {
			log.Debugf(log.ExchangeSys, "%v - Websocket subscription acknowledgement", b.Name)
		}
	case "bts:unsubscribe":
		if b.Verbose {
			log.Debugf(log.ExchangeSys, "%v - Websocket unsubscribe acknowledgement", b.Name)
		}
	case "bts:request_reconnect":
		if b.Verbose {
			log.Debugf(log.ExchangeSys, "%v - Websocket reconnection request received", b.Name)
		}
		go func() {
			err := b.Websocket.Shutdown()
			if err != nil {
				log.Errorf(log.WebsocketMgr, "%s failed to shutdown websocket: %v", b.Name, err)
			}
		}() // Connection monitor will reconnect
	case "data":
		wsOrderBookTemp := websocketOrderBookResponse{}
		err := json.Unmarshal(respRaw, &wsOrderBookTemp)
		if err != nil {
			return err
		}
		var currencyPair string
		splitter := strings.Split(wsResponse.Channel, currency.UnderscoreDelimiter)
		if len(splitter) == 3 {
			currencyPair = splitter[2]
		} else {
			return errWSPairParsingError
		}
		pFmt, err := b.GetPairFormat(asset.Spot, true)
		if err != nil {
			return err
		}

		enabledPairs, err := b.GetEnabledPairs(asset.Spot)
		if err != nil {
			return err
		}

		p, err := currency.NewPairFromFormattedPairs(currencyPair, enabledPairs, pFmt)
		if err != nil {
			return err
		}

		err = b.wsUpdateOrderbook(&wsOrderBookTemp.Data, p, asset.Spot)
		if err != nil {
			return err
		}
	case "trade":
		if !b.IsSaveTradeDataEnabled() {
			return nil
		}
		wsTradeTemp := websocketTradeResponse{}
		err := json.Unmarshal(respRaw, &wsTradeTemp)
		if err != nil {
			return err
		}

		var currencyPair string
		splitter := strings.Split(wsResponse.Channel, currency.UnderscoreDelimiter)
		if len(splitter) == 3 {
			currencyPair = splitter[2]
		} else {
			return errWSPairParsingError
		}
		pFmt, err := b.GetPairFormat(asset.Spot, true)
		if err != nil {
			return err
		}

		enabledPairs, err := b.GetEnabledPairs(asset.Spot)
		if err != nil {
			return err
		}

		p, err := currency.NewPairFromFormattedPairs(currencyPair, enabledPairs, pFmt)
		if err != nil {
			return err
		}

		side := order.Buy
		if wsTradeTemp.Data.Type == 1 {
			side = order.Sell
		}
		var a asset.Item
		a, err = b.GetPairAssetType(p)
		if err != nil {
			return err
		}
		return trade.AddTradesToBuffer(b.Name, trade.Data{
			Timestamp:    time.Unix(wsTradeTemp.Data.Timestamp, 0),
			CurrencyPair: p,
			AssetType:    a,
			Exchange:     b.Name,
			Price:        wsTradeTemp.Data.Price,
			Amount:       wsTradeTemp.Data.Amount,
			Side:         side,
			TID:          strconv.FormatInt(wsTradeTemp.Data.ID, 10),
		})
	case "order_created", "order_deleted", "order_changed":
		if b.Verbose {
			log.Debugf(log.ExchangeSys, "%v - Websocket order acknowledgement", b.Name)
		}
	default:
		b.Websocket.DataHandler <- stream.UnhandledMessageWarning{Message: b.Name + stream.UnhandledMessage + string(respRaw)}
	}
	return nil
}

func (b *Bitstamp) generateDefaultSubscriptions() ([]stream.ChannelSubscription, error) {
	var channels = []string{"live_trades_", "order_book_"}
	enabledCurrencies, err := b.GetEnabledPairs(asset.Spot)
	if err != nil {
		return nil, err
	}
	var subscriptions []stream.ChannelSubscription
	for i := range channels {
		for j := range enabledCurrencies {
			p, err := b.FormatExchangeCurrency(enabledCurrencies[j], asset.Spot)
			if err != nil {
				return nil, err
			}
			subscriptions = append(subscriptions, stream.ChannelSubscription{
				Channel: channels[i] + p.String(),
				Asset:   asset.Spot,
			})
		}
	}
	return subscriptions, nil
}

// Subscribe sends a websocket message to receive data from the channel
func (b *Bitstamp) Subscribe(channelsToSubscribe []stream.ChannelSubscription) error {
	var errs error
	for i := range channelsToSubscribe {
		req := websocketEventRequest{
			Event: "bts:subscribe",
			Data: websocketData{
				Channel: channelsToSubscribe[i].Channel,
			},
		}
		err := b.Websocket.Conn.SendJSONMessage(req)
		if err != nil {
			errs = common.AppendError(errs, err)
			continue
		}
		b.Websocket.AddSuccessfulSubscriptions(channelsToSubscribe[i])
	}
	return errs
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (b *Bitstamp) Unsubscribe(channelsToUnsubscribe []stream.ChannelSubscription) error {
	var errs error
	for i := range channelsToUnsubscribe {
		req := websocketEventRequest{
			Event: "bts:unsubscribe",
			Data: websocketData{
				Channel: channelsToUnsubscribe[i].Channel,
			},
		}
		err := b.Websocket.Conn.SendJSONMessage(req)
		if err != nil {
			errs = common.AppendError(errs, err)
			continue
		}
		b.Websocket.RemoveSuccessfulUnsubscriptions(channelsToUnsubscribe[i])
	}
	return errs
}

func (b *Bitstamp) wsUpdateOrderbook(update *websocketOrderBook, p currency.Pair, assetType asset.Item) error {
	if len(update.Asks) == 0 && len(update.Bids) == 0 {
		return errors.New("no orderbook data")
	}

	obUpdate := &orderbook.Base{
		Bids:            make(orderbook.Items, len(update.Bids)),
		Asks:            make(orderbook.Items, len(update.Asks)),
		Pair:            p,
		LastUpdated:     time.UnixMicro(update.Microtimestamp),
		Asset:           assetType,
		Exchange:        b.Name,
		VerifyOrderbook: b.CanVerifyOrderbook,
	}

	for i := range update.Asks {
		target, err := strconv.ParseFloat(update.Asks[i][0], 64)
		if err != nil {
			return err
		}
		amount, err := strconv.ParseFloat(update.Asks[i][1], 64)
		if err != nil {
			return err
		}
		obUpdate.Asks[i] = orderbook.Item{Price: target, Amount: amount}
	}
	for i := range update.Bids {
		target, err := strconv.ParseFloat(update.Bids[i][0], 64)
		if err != nil {
			return err
		}
		amount, err := strconv.ParseFloat(update.Bids[i][1], 64)
		if err != nil {
			return err
		}
		obUpdate.Bids[i] = orderbook.Item{Price: target, Amount: amount}
	}
	filterOrderbookZeroBidPrice(obUpdate)
	return b.Websocket.Orderbook.LoadSnapshot(obUpdate)
}

func (b *Bitstamp) seedOrderBook(ctx context.Context) error {
	p, err := b.GetEnabledPairs(asset.Spot)
	if err != nil {
		return err
	}

	for x := range p {
		pairFmt, err := b.FormatExchangeCurrency(p[x], asset.Spot)
		if err != nil {
			return err
		}
		orderbookSeed, err := b.GetOrderbook(ctx, pairFmt.String())
		if err != nil {
			return err
		}

		newOrderBook := &orderbook.Base{
			Pair:            p[x],
			Asset:           asset.Spot,
			Exchange:        b.Name,
			VerifyOrderbook: b.CanVerifyOrderbook,
			Bids:            make(orderbook.Items, len(orderbookSeed.Bids)),
			Asks:            make(orderbook.Items, len(orderbookSeed.Asks)),
			LastUpdated:     time.Unix(orderbookSeed.Timestamp, 0),
		}

		for i := range orderbookSeed.Asks {
			newOrderBook.Asks[i] = orderbook.Item{
				Price:  orderbookSeed.Asks[i].Price,
				Amount: orderbookSeed.Asks[i].Amount,
			}
		}
		for i := range orderbookSeed.Bids {
			newOrderBook.Bids[i] = orderbook.Item{
				Price:  orderbookSeed.Bids[i].Price,
				Amount: orderbookSeed.Bids[i].Amount,
			}
		}

		filterOrderbookZeroBidPrice(newOrderBook)

		err = b.Websocket.Orderbook.LoadSnapshot(newOrderBook)
		if err != nil {
			return err
		}
	}
	return nil
}
