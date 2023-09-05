package bitstamp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
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

var (
	hbMsg = []byte(`{"event":"bts:heartbeat"}`)

	defaultSubChannels = []string{
		bitstampAPIWSTrades,
		bitstampAPIWSOrderbook,
	}

	defaultAuthSubChannels = []string{
		bitstampAPIWSMyOrders,
		bitstampAPIWSMyTrades,
	}
)

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
		if err := b.wsHandleData(resp.Raw); err != nil {
			b.Websocket.DataHandler <- err
		}
	}
}

func (b *Bitstamp) wsHandleData(respRaw []byte) error {
	wsResponse := &websocketResponse{}
	if err := json.Unmarshal(respRaw, wsResponse); err != nil {
		return err
	}

	if err := b.parseChannelName(wsResponse); err != nil {
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
		if err := b.handleWSOrderbook(context.TODO(), wsResponse, respRaw); err != nil {
			return err
		}
	case "trade":
		if err := b.handleWSTrade(context.TODO(), wsResponse, respRaw); err != nil {
			return err
		}
	case "order_created", "order_deleted", "order_changed":
		if err := b.handleWSOrder(context.TODO(), wsResponse, respRaw); err != nil {
			return err
		}
	default:
		b.Websocket.DataHandler <- stream.UnhandledMessageWarning{Message: b.Name + stream.UnhandledMessage + string(respRaw)}
	}
	return nil
}

func (b *Bitstamp) handleWSOrderbook(_ context.Context, wsResp *websocketResponse, msg []byte) error {
	if wsResp.pair == currency.EMPTYPAIR {
		return errWSPairParsingError
	}

	wsOrderBookTemp := websocketOrderBookResponse{}
	err := json.Unmarshal(msg, &wsOrderBookTemp)
	if err != nil {
		return err
	}

	return b.wsUpdateOrderbook(&wsOrderBookTemp.Data, wsResp.pair, asset.Spot)
}

func (b *Bitstamp) handleWSTrade(_ context.Context, wsResp *websocketResponse, msg []byte) error {
	if !b.IsSaveTradeDataEnabled() {
		return nil
	}

	if wsResp.pair == currency.EMPTYPAIR {
		return errWSPairParsingError
	}

	wsTradeTemp := websocketTradeResponse{}
	if err := json.Unmarshal(msg, &wsTradeTemp); err != nil {
		return err
	}

	side := order.Buy
	if wsTradeTemp.Data.Type == 1 {
		side = order.Sell
	}
	return trade.AddTradesToBuffer(b.Name, trade.Data{
		Timestamp:    time.Unix(wsTradeTemp.Data.Timestamp, 0),
		CurrencyPair: wsResp.pair,
		AssetType:    asset.Spot,
		Exchange:     b.Name,
		Price:        wsTradeTemp.Data.Price,
		Amount:       wsTradeTemp.Data.Amount,
		Side:         side,
		TID:          strconv.FormatInt(wsTradeTemp.Data.ID, 10),
	})
}

func (b *Bitstamp) handleWSOrder(_ context.Context, wsResp *websocketResponse, msg []byte) error {
	if wsResp.channelType != bitstampAPIWSMyOrders {
		return nil
	}

	r := &websocketOrderResponse{}
	if err := json.Unmarshal(msg, &r); err != nil {
		return err
	}

	o := r.Order
	if o.ID == 0 {
		return fmt.Errorf("unable to parse an order id from order msg: %s", msg)
	}

	var status order.Status
	switch wsResp.Event {
	case "order_created":
		status = order.New
	case "order_changed":
		if o.ExecutedAmount > 0 {
			status = order.PartiallyFilled
		}
	case "order_deleted":
		if o.RemainingAmount == 0 && o.Amount > 0 {
			status = order.Filled
		} else {
			status = order.Cancelled
		}
	}

	// o.ExecutedAmount is an atomic partial fill amount; We want total
	executedAmount := o.Amount - o.RemainingAmount

	d := &order.Detail{
		Price:           o.Price,
		Amount:          o.Amount,
		RemainingAmount: o.RemainingAmount,
		ExecutedAmount:  executedAmount,
		Exchange:        b.Name,
		OrderID:         o.IDStr,
		ClientOrderID:   o.ClientOrderID,
		Side:            o.Side,
		Status:          status,
		AssetType:       asset.Spot,
		Date:            o.Microtimestamp,
		Pair:            wsResp.pair,
	}

	b.Websocket.DataHandler <- d

	return nil
}

func (b *Bitstamp) generateDefaultSubscriptions() ([]stream.ChannelSubscription, error) {
	enabledCurrencies, err := b.GetEnabledPairs(asset.Spot)
	if err != nil {
		return nil, err
	}
	var subscriptions []stream.ChannelSubscription
	for i := range enabledCurrencies {
		p, err := b.FormatExchangeCurrency(enabledCurrencies[i], asset.Spot)
		if err != nil {
			return nil, err
		}
		for j := range defaultSubChannels {
			subscriptions = append(subscriptions, stream.ChannelSubscription{
				Channel:  defaultSubChannels[j] + "_" + p.String(),
				Asset:    asset.Spot,
				Currency: p,
			})
		}
		if b.Websocket.CanUseAuthenticatedEndpoints() {
			for j := range defaultAuthSubChannels {
				subscriptions = append(subscriptions, stream.ChannelSubscription{
					Channel:  defaultAuthSubChannels[j] + "_" + p.String(),
					Asset:    asset.Spot,
					Currency: p,
					Params: map[string]interface{}{
						"auth": struct{}{},
					},
				})
			}
		}
	}
	return subscriptions, nil
}

// Subscribe sends a websocket message to receive data from the channel
func (b *Bitstamp) Subscribe(channelsToSubscribe []stream.ChannelSubscription) error {
	var errs error
	var auth *websocketAuthResponse

	for i := range channelsToSubscribe {
		if _, ok := channelsToSubscribe[i].Params["auth"]; ok {
			var err error
			auth, err = b.fetchWSAuth(context.TODO())
			if err != nil {
				errs = common.AppendError(errs, err)
			}
			break
		}
	}

	for i := range channelsToSubscribe {
		req := websocketEventRequest{
			Event: "bts:subscribe",
			Data: websocketData{
				Channel: channelsToSubscribe[i].Channel,
			},
		}
		if _, ok := channelsToSubscribe[i].Params["auth"]; ok && auth != nil {
			req.Data.Channel = "private-" + req.Data.Channel + "-" + string(auth.UserID)
			req.Data.Auth = auth.Token
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

// fetchWSAuth Retrieves a userID and auth-token from REST for subscribing to a websocket channel
// The token life-expectancy is only about 60s; use it immediately and do not store it
func (b *Bitstamp) fetchWSAuth(ctx context.Context) (*websocketAuthResponse, error) {
	resp := &websocketAuthResponse{}
	err := b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, bitstampAPIWSAuthToken, true, nil, resp)
	if err != nil {
		return nil, fmt.Errorf("error fetching auth token: %w", err)
	}
	return resp, nil
}

// parseChannel splits the ws response channel and sets the channel type and pair
func (b *Bitstamp) parseChannelName(r *websocketResponse) error {
	if r.Channel == "" {
		return nil
	}

	chanName := r.Channel
	authParts := strings.Split(r.Channel, "-")
	switch len(authParts) {
	case 1:
		// Not an auth channel
	case 3:
		chanName = authParts[1]
	default:
		return fmt.Errorf("channel name does not contain exactly 0 or 2 hyphens: %v", r.Channel)
	}

	parts := strings.Split(chanName, currency.UnderscoreDelimiter)
	if len(parts) != 3 {
		return fmt.Errorf("%v: channel name does not contain exactly 2 underscores: %v", errWSPairParsingError, r.Channel)
	}

	symbol := parts[2]
	pFmt, err := b.GetPairFormat(asset.Spot, true)
	if err != nil {
		return err
	}

	enabledPairs, err := b.GetEnabledPairs(asset.Spot)
	if err != nil {
		return err
	}

	p, err := currency.NewPairFromFormattedPairs(symbol, enabledPairs, pFmt)
	if err != nil {
		return err
	}

	r.pair = p
	r.channelType = parts[0] + "_" + parts[1]

	return nil
}
