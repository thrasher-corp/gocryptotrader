package poloniex

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	poloniexWebsocketAddress = "wss://api2.poloniex.com"
	wsAccountNotificationID  = 1000
	wsTickerDataID           = 1002
	ws24HourExchangeVolumeID = 1003
	wsHeartbeat              = 1010

	accountNotificationBalanceUpdate     = "b"
	accountNotificationOrderUpdate       = "o"
	accountNotificationPendingOrder      = "p"
	accountNotificationOrderLimitCreated = "n"
	accountNotificationTrades            = "t"
	accountNotificationKilledOrder       = "k"
	accountNotificationMarginPosition    = "m"

	orderbookInitial = "i"
	orderbookUpdate  = "o"
	tradeUpdate      = "t"
)

var (
	errNotEnoughData        = errors.New("element length not adequate to process")
	errTypeAssertionFailure = errors.New("type assertion failure")
	errIDNotFoundInPairMap  = errors.New("id not associated with currency pair map")
	errIDNotFoundInCodeMap  = errors.New("id not associated with currency code map")
)

// WsConnect initiates a websocket connection
func (p *Poloniex) WsConnect() error {
	ctx := context.TODO()
	if !p.Websocket.IsEnabled() || !p.IsEnabled() {
		return websocket.ErrWebsocketNotEnabled
	}
	var dialer gws.Dialer
	err := p.Websocket.Conn.Dial(ctx, &dialer, http.Header{})
	if err != nil {
		return err
	}

	err = p.loadCurrencyDetails(ctx)
	if err != nil {
		return err
	}

	p.Websocket.Wg.Add(1)
	go p.wsReadData()

	return nil
}

// TODO: Create routine to refresh list every day/week(?) for production
func (p *Poloniex) loadCurrencyDetails(ctx context.Context) error {
	if p.details.isInitial() {
		ticks, err := p.GetTicker(ctx)
		if err != nil {
			return err
		}
		err = p.details.loadPairs(ticks)
		if err != nil {
			return err
		}

		currs, err := p.GetCurrencies(ctx)
		if err != nil {
			return err
		}

		err = p.details.loadCodes(currs)
		if err != nil {
			return err
		}
	}
	return nil
}

// wsReadData handles data from the websocket connection
func (p *Poloniex) wsReadData() {
	defer p.Websocket.Wg.Done()
	for {
		resp := p.Websocket.Conn.ReadMessage()
		if resp.Raw == nil {
			return
		}
		err := p.wsHandleData(resp.Raw)
		if err != nil {
			p.Websocket.DataHandler <- fmt.Errorf("%s: %w", p.Name, err)
		}
	}
}

func (p *Poloniex) wsHandleData(respRaw []byte) error {
	var result any
	err := json.Unmarshal(respRaw, &result)
	if err != nil {
		return err
	}

	data, ok := result.([]any)
	if !ok {
		return fmt.Errorf("%w data is not []any",
			errTypeAssertionFailure)
	}

	if len(data) == 0 {
		return nil
	}
	if len(data) == 2 {
		// subscription acknowledgement
		// TODO: Add in subscriber ack
		return nil
	}

	channelID, ok := data[0].(float64)
	if !ok {
		return fmt.Errorf("%w channel id is not of type float64",
			errTypeAssertionFailure)
	}
	switch channelID {
	case ws24HourExchangeVolumeID, wsHeartbeat:
		return nil
	case wsAccountNotificationID:
		var notificationsArray []any
		notificationsArray, ok = data[2].([]any)
		if !ok {
			return fmt.Errorf("%w account notification is not a []any",
				errTypeAssertionFailure)
		}
		for i := range notificationsArray {
			var notification []any
			notification, ok = (notificationsArray[i]).([]any)
			if !ok {
				return fmt.Errorf("%w notification array element is not a []any",
					errTypeAssertionFailure)
			}
			var updateType string
			updateType, ok = notification[0].(string)
			if !ok {
				return fmt.Errorf("%w update type is not a string",
					errTypeAssertionFailure)
			}

			switch updateType {
			case accountNotificationPendingOrder:
				err = p.processAccountPendingOrder(notification)
				if err != nil {
					return fmt.Errorf("account notification pending order: %w", err)
				}
			case accountNotificationOrderUpdate:
				err = p.processAccountOrderUpdate(notification)
				if err != nil {
					return fmt.Errorf("account notification order update: %w", err)
				}
			case accountNotificationOrderLimitCreated:
				err = p.processAccountOrderLimit(notification)
				if err != nil {
					return fmt.Errorf("account notification limit order creation: %w", err)
				}
			case accountNotificationBalanceUpdate:
				err = p.processAccountBalanceUpdate(notification)
				if err != nil {
					return fmt.Errorf("account notification balance update: %w", err)
				}
			case accountNotificationTrades:
				err = p.processAccountTrades(notification)
				if err != nil {
					return fmt.Errorf("account notification trades: %w", err)
				}
			case accountNotificationKilledOrder:
				err = p.processAccountKilledOrder(notification)
				if err != nil {
					return fmt.Errorf("account notification killed order: %w", err)
				}
			case accountNotificationMarginPosition:
				err = p.processAccountMarginPosition(notification)
				if err != nil {
					return fmt.Errorf("account notification margin position: %w", err)
				}
			default:
				return fmt.Errorf("unhandled account update: %s", string(respRaw))
			}
		}
		return nil
	case wsTickerDataID:
		err = p.wsHandleTickerData(data)
		if err != nil {
			return fmt.Errorf("websocket ticker process: %w", err)
		}
		return nil
	}

	priceAggBook, ok := data[2].([]any)
	if !ok {
		return fmt.Errorf("%w price aggregated book not []any",
			errTypeAssertionFailure)
	}

	for x := range priceAggBook {
		subData, ok := priceAggBook[x].([]any)
		if !ok {
			return fmt.Errorf("%w price aggregated book element not []any",
				errTypeAssertionFailure)
		}

		updateIdent, ok := subData[0].(string)
		if !ok {
			return fmt.Errorf("%w update identifier not a string",
				errTypeAssertionFailure)
		}

		switch updateIdent {
		case orderbookInitial:
			err = p.WsProcessOrderbookSnapshot(subData)
			if err != nil {
				return fmt.Errorf("websocket process orderbook snapshot: %w", err)
			}
		case orderbookUpdate:
			var pair currency.Pair
			pair, err = p.details.GetPair(channelID)
			if err != nil {
				return err
			}
			var seqNo float64
			seqNo, ok = data[1].(float64)
			if !ok {
				return fmt.Errorf("%w sequence number is not a float64",
					errTypeAssertionFailure)
			}
			err = p.WsProcessOrderbookUpdate(seqNo, subData, pair)
			if err != nil {
				return fmt.Errorf("websocket process orderbook update: %w", err)
			}
		case tradeUpdate:
			err = p.processTrades(channelID, subData)
			if err != nil {
				return fmt.Errorf("websocket process trades update: %w", err)
			}
		default:
			p.Websocket.DataHandler <- websocket.UnhandledMessageWarning{
				Message: p.Name + websocket.UnhandledMessage + string(respRaw),
			}
		}
	}
	return nil
}

func (p *Poloniex) wsHandleTickerData(data []any) error {
	tickerData, ok := data[2].([]any)
	if !ok {
		return fmt.Errorf("%w ticker data is not []any",
			errTypeAssertionFailure)
	}

	currencyID, ok := tickerData[0].(float64)
	if !ok {
		return fmt.Errorf("%w currency ID not float64", errTypeAssertionFailure)
	}

	pair, err := p.details.GetPair(currencyID)
	if err != nil {
		return err
	}

	enabled, err := p.GetEnabledPairs(asset.Spot)
	if err != nil {
		return err
	}

	if !enabled.Contains(pair, true) {
		return nil
	}

	tlp, ok := tickerData[1].(string)
	if !ok {
		return fmt.Errorf("%w last price not string", errTypeAssertionFailure)
	}

	lastPrice, err := strconv.ParseFloat(tlp, 64)
	if err != nil {
		return err
	}

	la, ok := tickerData[2].(string)
	if !ok {
		return fmt.Errorf("%w lowest ask price not string",
			errTypeAssertionFailure)
	}

	lowestAsk, err := strconv.ParseFloat(la, 64)
	if err != nil {
		return err
	}

	hb, ok := tickerData[3].(string)
	if !ok {
		return fmt.Errorf("%w highest bid price not string",
			errTypeAssertionFailure)
	}

	highestBid, err := strconv.ParseFloat(hb, 64)
	if err != nil {
		return err
	}

	bcv, ok := tickerData[5].(string)
	if !ok {
		return fmt.Errorf("%w base currency volume not string",
			errTypeAssertionFailure)
	}

	baseCurrencyVolume24H, err := strconv.ParseFloat(bcv, 64)
	if err != nil {
		return err
	}

	qcv, ok := tickerData[6].(string)
	if !ok {
		return fmt.Errorf("%w quote currency volume not string",
			errTypeAssertionFailure)
	}

	quoteCurrencyVolume24H, err := strconv.ParseFloat(qcv, 64)
	if err != nil {
		return err
	}

	// Unused variables below, can add later if needed:
	// percentageChange, ok := tickerData[4].(string)
	// Not integrating isFrozen with currency details as this will slow down
	// the sync RW mutex (can use REST calls for now).
	// isFrozen, ok := tickerData[7].(float64) // == 1 means it is frozen
	// highestTradeIn24Hm, ok := tickerData[8].(string)
	// lowestTradePrice24H, ok := tickerData[9].(string)

	p.Websocket.DataHandler <- &ticker.Price{
		ExchangeName: p.Name,
		Volume:       baseCurrencyVolume24H,
		QuoteVolume:  quoteCurrencyVolume24H,
		High:         highestBid,
		Low:          lowestAsk,
		Bid:          highestBid,
		Ask:          lowestAsk,
		Last:         lastPrice,
		AssetType:    asset.Spot,
		Pair:         pair,
	}
	return nil
}

// WsProcessOrderbookSnapshot processes a new orderbook snapshot into a local
// of orderbooks
func (p *Poloniex) WsProcessOrderbookSnapshot(data []any) error {
	subDataMap, ok := data[1].(map[string]any)
	if !ok {
		return fmt.Errorf("%w subData element is not map[string]any",
			errTypeAssertionFailure)
	}

	pMap, ok := subDataMap["currencyPair"]
	if !ok {
		return errors.New("could not find currency pair in map")
	}

	pair, ok := pMap.(string)
	if !ok {
		return fmt.Errorf("%w subData element is not map[string]any",
			errTypeAssertionFailure)
	}

	if len(data) < 3 {
		return fmt.Errorf("%w for pair %v", errNotEnoughData, pair)
	}

	ts, ok := data[2].(string)
	if !ok {
		return common.GetTypeAssertError("string", data[2], "timestamp string")
	}

	tsMilli, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return err
	}

	oMap, ok := subDataMap["orderBook"]
	if !ok {
		return errors.New("could not find orderbook data in map")
	}

	ob, ok := oMap.([]any)
	if !ok {
		return fmt.Errorf("%w orderbook data is not []any",
			errTypeAssertionFailure)
	}

	if len(ob) != 2 {
		return errNotEnoughData
	}

	askData, ok := ob[0].(map[string]any)
	if !ok {
		return fmt.Errorf("%w ask data is not map[string]any",
			errTypeAssertionFailure)
	}

	bidData, ok := ob[1].(map[string]any)
	if !ok {
		return fmt.Errorf("%w bid data is not map[string]any",
			errTypeAssertionFailure)
	}

	var book orderbook.Book
	book.Asks = make(orderbook.Levels, 0, len(askData))
	for price, volume := range askData {
		var p float64
		p, err = strconv.ParseFloat(price, 64)
		if err != nil {
			return err
		}
		v, ok := volume.(string)
		if !ok {
			return fmt.Errorf("%w ask volume data not string",
				errTypeAssertionFailure)
		}
		var a float64
		a, err = strconv.ParseFloat(v, 64)
		if err != nil {
			return err
		}
		book.Asks = append(book.Asks, orderbook.Level{Price: p, Amount: a})
	}

	book.Bids = make(orderbook.Levels, 0, len(bidData))
	for price, volume := range bidData {
		var p float64
		p, err = strconv.ParseFloat(price, 64)
		if err != nil {
			return err
		}
		v, ok := volume.(string)
		if !ok {
			return fmt.Errorf("%w bid volume data not string",
				errTypeAssertionFailure)
		}
		var a float64
		a, err = strconv.ParseFloat(v, 64)
		if err != nil {
			return err
		}
		book.Bids = append(book.Bids, orderbook.Level{Price: p, Amount: a})
	}

	// Both sides are completely out of order - sort needs to be used
	book.Asks.SortAsks()
	book.Bids.SortBids()
	book.Asset = asset.Spot
	book.ValidateOrderbook = p.ValidateOrderbook
	book.LastUpdated = time.UnixMilli(tsMilli)
	book.Pair, err = currency.NewPairFromString(pair)
	if err != nil {
		return err
	}
	book.Exchange = p.Name

	return p.Websocket.Orderbook.LoadSnapshot(&book)
}

// WsProcessOrderbookUpdate processes new orderbook updates
func (p *Poloniex) WsProcessOrderbookUpdate(sequenceNumber float64, data []any, pair currency.Pair) error {
	if len(data) < 5 {
		return errNotEnoughData
	}

	ps, ok := data[2].(string)
	if !ok {
		return fmt.Errorf("%w price not string", errTypeAssertionFailure)
	}
	price, err := strconv.ParseFloat(ps, 64)
	if err != nil {
		return err
	}
	vs, ok := data[3].(string)
	if !ok {
		return fmt.Errorf("%w volume not string", errTypeAssertionFailure)
	}
	volume, err := strconv.ParseFloat(vs, 64)
	if err != nil {
		return err
	}
	bs, ok := data[1].(float64)
	if !ok {
		return fmt.Errorf("%w buysell not float64", errTypeAssertionFailure)
	}

	ts, ok := data[4].(string)
	if !ok {
		return common.GetTypeAssertError("string", data[2], "timestamp string")
	}

	tsMilli, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return err
	}

	update := &orderbook.Update{
		Pair:       pair,
		Asset:      asset.Spot,
		UpdateID:   int64(sequenceNumber),
		UpdateTime: time.UnixMilli(tsMilli),
	}
	if bs == 1 {
		update.Bids = []orderbook.Level{{Price: price, Amount: volume}}
	} else {
		update.Asks = []orderbook.Level{{Price: price, Amount: volume}}
	}
	return p.Websocket.Orderbook.Update(update)
}

// GenerateDefaultSubscriptions Adds default subscriptions to websocket to be handled by ManageSubscriptions()
func (p *Poloniex) GenerateDefaultSubscriptions() (subscription.List, error) {
	enabledPairs, err := p.GetEnabledPairs(asset.Spot)
	if err != nil {
		return nil, err
	}

	subscriptions := make(subscription.List, 0, len(enabledPairs))
	subscriptions = append(subscriptions, &subscription.Subscription{
		Channel: strconv.FormatInt(wsTickerDataID, 10),
	})

	if p.IsWebsocketAuthenticationSupported() {
		subscriptions = append(subscriptions, &subscription.Subscription{
			Channel: strconv.FormatInt(wsAccountNotificationID, 10),
		})
	}

	for j := range enabledPairs {
		enabledPairs[j].Delimiter = currency.UnderscoreDelimiter
		subscriptions = append(subscriptions, &subscription.Subscription{
			Channel: "orderbook",
			Pairs:   currency.Pairs{enabledPairs[j]},
			Asset:   asset.Spot,
		})
	}
	return subscriptions, nil
}

// Subscribe sends a websocket message to receive data from the channel
func (p *Poloniex) Subscribe(subs subscription.List) error {
	ctx := context.TODO()
	return p.manageSubs(ctx, subs, wsSubscribeOp)
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (p *Poloniex) Unsubscribe(subs subscription.List) error {
	ctx := context.TODO()
	return p.manageSubs(ctx, subs, wsUnsubscribeOp)
}

func (p *Poloniex) manageSubs(ctx context.Context, subs subscription.List, op wsOp) error {
	var creds *account.Credentials
	if p.IsWebsocketAuthenticationSupported() {
		var err error
		creds, err = p.GetCredentials(ctx)
		if err != nil {
			return err
		}
	}

	var errs error
	for _, s := range subs {
		var err error
		if creds != nil && strings.EqualFold(strconv.FormatInt(wsAccountNotificationID, 10), s.Channel) {
			err = p.wsSendAuthorisedCommand(ctx, creds.Secret, creds.Key, op)
		} else {
			req := wsCommand{Command: op}
			if strings.EqualFold(strconv.FormatInt(wsTickerDataID, 10), s.Channel) {
				req.Channel = wsTickerDataID
			} else {
				if len(s.Pairs) != 1 {
					return subscription.ErrNotSinglePair
				}
				req.Channel = s.Pairs[0].String()
			}
			err = p.Websocket.Conn.SendJSONMessage(ctx, request.Unset, req)
		}
		if err == nil {
			if op == wsSubscribeOp {
				err = p.Websocket.AddSuccessfulSubscriptions(p.Websocket.Conn, s)
			} else {
				err = p.Websocket.RemoveSubscriptions(p.Websocket.Conn, s)
			}
		}
		if err != nil {
			errs = common.AppendError(errs, err)
		}
	}
	return errs
}

func (p *Poloniex) wsSendAuthorisedCommand(ctx context.Context, secret, key string, op wsOp) error {
	nonce := fmt.Sprintf("nonce=%v", time.Now().UnixNano())
	hmac, err := crypto.GetHMAC(crypto.HashSHA512, []byte(nonce), []byte(secret))
	if err != nil {
		return err
	}
	req := wsAuthorisationRequest{
		Command: op,
		Channel: 1000,
		Sign:    hex.EncodeToString(hmac),
		Key:     key,
		Payload: nonce,
	}
	return p.Websocket.Conn.SendJSONMessage(ctx, request.Unset, req)
}

func (p *Poloniex) processAccountMarginPosition(notification []any) error {
	if len(notification) < 5 {
		return errNotEnoughData
	}

	orderID, ok := notification[1].(float64)
	if !ok {
		return fmt.Errorf("%w order id not float64", errTypeAssertionFailure)
	}

	currencyID, ok := notification[2].(float64)
	if !ok {
		return fmt.Errorf("%w currency id not float64", errTypeAssertionFailure)
	}
	code, err := p.details.GetCode(currencyID)
	if err != nil {
		return err
	}

	a, ok := notification[3].(string)
	if !ok {
		return fmt.Errorf("%w amount not string", errTypeAssertionFailure)
	}

	amount, err := strconv.ParseFloat(a, 64)
	if err != nil {
		return err
	}

	// null returned so ok check is not needed
	clientOrderID, _ := notification[4].(string)

	// Temp struct for margin position changes
	p.Websocket.DataHandler <- struct {
		OrderID       string
		Code          currency.Code
		Amount        float64
		ClientOrderID string
	}{
		OrderID:       strconv.FormatFloat(orderID, 'f', -1, 64),
		Code:          code,
		Amount:        amount,
		ClientOrderID: clientOrderID,
	}

	return nil
}

func (p *Poloniex) processAccountPendingOrder(notification []any) error {
	if len(notification) < 7 {
		return errNotEnoughData
	}

	orderID, ok := notification[1].(float64)
	if !ok {
		return fmt.Errorf("%w order id not float64", errTypeAssertionFailure)
	}

	currencyID, ok := notification[2].(float64)
	if !ok {
		return fmt.Errorf("%w currency id not float64", errTypeAssertionFailure)
	}
	pair, err := p.details.GetPair(currencyID)
	if err != nil {
		if !errors.Is(err, errIDNotFoundInPairMap) {
			return err
		}
		log.Errorf(log.WebsocketMgr,
			"%s - Unknown currency pair ID. Currency will appear as the pair ID: '%v'",
			p.Name,
			currencyID)
	}

	price, ok := notification[3].(string)
	if !ok {
		return fmt.Errorf("%w price not string", errTypeAssertionFailure)
	}
	orderPrice, err := strconv.ParseFloat(price, 64)
	if err != nil {
		return err
	}
	amount, ok := notification[4].(string)
	if !ok {
		return fmt.Errorf("%w amount not string", errTypeAssertionFailure)
	}
	orderAmount, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		return err
	}
	side, ok := notification[5].(string)
	if !ok {
		return fmt.Errorf("%w order type not string", errTypeAssertionFailure)
	}
	orderSide := order.Buy
	if side == "0" {
		orderSide = order.Sell
	}

	// null returned so ok check is not needed
	clientOrderID, _ := notification[6].(string)

	p.Websocket.DataHandler <- &order.Detail{
		Exchange:        p.Name,
		OrderID:         strconv.FormatFloat(orderID, 'f', -1, 64),
		Pair:            pair,
		AssetType:       asset.Spot,
		Side:            orderSide,
		Price:           orderPrice,
		Amount:          orderAmount,
		RemainingAmount: orderAmount,
		ClientOrderID:   clientOrderID,
		Status:          order.Pending,
	}
	return nil
}

func (p *Poloniex) processAccountOrderUpdate(notification []any) error {
	if len(notification) < 5 {
		return errNotEnoughData
	}

	orderID, ok := notification[1].(float64)
	if !ok {
		return fmt.Errorf("%w order id not float64", errTypeAssertionFailure)
	}

	a, ok := notification[2].(string)
	if !ok {
		return fmt.Errorf("%w amount not string", errTypeAssertionFailure)
	}
	amount, err := strconv.ParseFloat(a, 64)
	if err != nil {
		return err
	}

	oType, ok := notification[3].(string)
	if !ok {
		return fmt.Errorf("%w order type not string", errTypeAssertionFailure)
	}

	var oStatus order.Status
	var cancelledAmount float64
	if oType == "c" {
		if len(notification) < 6 {
			return errNotEnoughData
		}
		cancel, ok := notification[5].(string)
		if !ok {
			return fmt.Errorf("%w cancel amount not string", errTypeAssertionFailure)
		}

		cancelledAmount, err = strconv.ParseFloat(cancel, 64)
		if err != nil {
			return err
		}

		if amount > 0 {
			oStatus = order.PartiallyCancelled
		} else {
			oStatus = order.Cancelled
		}
	} else {
		if amount > 0 {
			oStatus = order.PartiallyFilled
		} else {
			oStatus = order.Filled
		}
	}

	// null returned so ok check is not needed
	clientOrderID, _ := notification[4].(string)

	p.Websocket.DataHandler <- &order.Detail{
		Exchange:        p.Name,
		RemainingAmount: cancelledAmount,
		Amount:          amount + cancelledAmount,
		ExecutedAmount:  amount,
		OrderID:         strconv.FormatFloat(orderID, 'f', -1, 64),
		Type:            order.Limit,
		Status:          oStatus,
		AssetType:       asset.Spot,
		ClientOrderID:   clientOrderID,
	}
	return nil
}

func (p *Poloniex) processAccountOrderLimit(notification []any) error {
	if len(notification) != 9 {
		return errNotEnoughData
	}

	currencyID, ok := notification[1].(float64)
	if !ok {
		return fmt.Errorf("%w currency ID not string", errTypeAssertionFailure)
	}
	pair, err := p.details.GetPair(currencyID)
	if err != nil {
		if !errors.Is(err, errIDNotFoundInPairMap) {
			return err
		}
		log.Errorf(log.WebsocketMgr,
			"%s - Unknown currency pair ID. Currency will appear as the pair ID: '%v'",
			p.Name,
			currencyID)
	}

	orderID, ok := notification[2].(float64)
	if !ok {
		return fmt.Errorf("%w order ID not float64", errTypeAssertionFailure)
	}

	side, ok := notification[3].(string)
	if !ok {
		return fmt.Errorf("%w order type not string", errTypeAssertionFailure)
	}
	orderSide := order.Buy
	if side == "0" {
		orderSide = order.Sell
	}

	rate, ok := notification[4].(string)
	if !ok {
		return fmt.Errorf("%w rate not string", errTypeAssertionFailure)
	}
	orderPrice, err := strconv.ParseFloat(rate, 64)
	if err != nil {
		return err
	}
	amount, ok := notification[5].(string)
	if !ok {
		return fmt.Errorf("%w amount not string", errTypeAssertionFailure)
	}
	orderAmount, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		return err
	}

	ts, ok := notification[6].(string)
	if !ok {
		return fmt.Errorf("%w time not string", errTypeAssertionFailure)
	}

	var timeParse time.Time
	timeParse, err = time.Parse(time.DateTime, ts)
	if err != nil {
		return err
	}

	origAmount, ok := notification[7].(string)
	if !ok {
		return fmt.Errorf("%w original amount not string", errTypeAssertionFailure)
	}
	origOrderAmount, err := strconv.ParseFloat(origAmount, 64)
	if err != nil {
		return err
	}

	// null returned so ok check is not needed
	clientOrderID, _ := notification[8].(string)
	p.Websocket.DataHandler <- &order.Detail{
		Exchange:        p.Name,
		Price:           orderPrice,
		RemainingAmount: orderAmount,
		ExecutedAmount:  origOrderAmount - orderAmount,
		Amount:          origOrderAmount,
		OrderID:         strconv.FormatFloat(orderID, 'f', -1, 64),
		Type:            order.Limit,
		Side:            orderSide,
		Status:          order.New,
		AssetType:       asset.Spot,
		Date:            timeParse,
		Pair:            pair,
		ClientOrderID:   clientOrderID,
	}
	return nil
}

func (p *Poloniex) processAccountBalanceUpdate(notification []any) error {
	if len(notification) < 4 {
		return errNotEnoughData
	}

	currencyID, ok := notification[1].(float64)
	if !ok {
		return fmt.Errorf("%w currency ID not float64", errTypeAssertionFailure)
	}
	code, err := p.details.GetCode(currencyID)
	if err != nil {
		return err
	}

	walletType, ok := notification[2].(string)
	if !ok {
		return fmt.Errorf("%w wallet addr not string", errTypeAssertionFailure)
	}

	a, ok := notification[3].(string)
	if !ok {
		return fmt.Errorf("%w amount not string", errTypeAssertionFailure)
	}
	amount, err := strconv.ParseFloat(a, 64)
	if err != nil {
		return err
	}

	// TODO: Integrate with exchange account system
	// NOTES: This will affect free amount, a rest call might be needed to get
	// locked and total amounts periodically.
	p.Websocket.DataHandler <- account.Change{
		Account:   deriveWalletType(walletType),
		AssetType: asset.Spot,
		Balance: &account.Balance{
			Currency: code,
			Total:    amount,
			Free:     amount,
		},
	}
	return nil
}

func deriveWalletType(s string) string {
	switch s {
	case "e":
		return "exchange"
	case "m":
		return "margin"
	case "l":
		return "lending"
	default:
		return "unknown"
	}
}

func (p *Poloniex) processAccountTrades(notification []any) error {
	if len(notification) < 11 {
		return errNotEnoughData
	}

	tradeID, ok := notification[1].(float64)
	if !ok {
		return fmt.Errorf("%w tradeID not float64", errTypeAssertionFailure)
	}

	r, ok := notification[2].(string)
	if !ok {
		return fmt.Errorf("%w rate not string", errTypeAssertionFailure)
	}
	rate, err := strconv.ParseFloat(r, 64)
	if err != nil {
		return err
	}

	a, ok := notification[3].(string)
	if !ok {
		return fmt.Errorf("%w amount not string", errTypeAssertionFailure)
	}
	amount, err := strconv.ParseFloat(a, 64)
	if err != nil {
		return err
	}

	// notification[4].(string) is the fee multiplier
	// notification[5].(string) is the funding type 0 (exchange wallet),
	// 1 (borrowed funds), 2 (margin funds), or 3 (lending funds)

	orderID, ok := notification[6].(float64)
	if !ok {
		return fmt.Errorf("%w orderID not float64", errTypeAssertionFailure)
	}

	fee, ok := notification[7].(string)
	if !ok {
		return fmt.Errorf("%w fee not string", errTypeAssertionFailure)
	}
	totalFee, err := strconv.ParseFloat(fee, 64)
	if err != nil {
		return err
	}

	t, ok := notification[8].(string)
	if !ok {
		return fmt.Errorf("%w time not string", errTypeAssertionFailure)
	}
	timeParse, err := time.Parse(time.DateTime, t)
	if err != nil {
		return err
	}

	// null returned so ok check is not needed
	clientOrderID, _ := notification[9].(string)

	tt, ok := notification[10].(string)
	if !ok {
		return fmt.Errorf("%w time not string", errTypeAssertionFailure)
	}
	tradeTotal, err := strconv.ParseFloat(tt, 64)
	if err != nil {
		return err
	}

	p.Websocket.DataHandler <- &order.Detail{
		Exchange: p.Name,
		OrderID:  strconv.FormatFloat(orderID, 'f', -1, 64),
		Fee:      totalFee,
		Trades: []order.TradeHistory{{
			Price:     rate,
			Amount:    amount,
			Fee:       totalFee,
			Exchange:  p.Name,
			TID:       strconv.FormatFloat(tradeID, 'f', -1, 64),
			Timestamp: timeParse,
			Total:     tradeTotal,
		}},
		AssetType:     asset.Spot,
		ClientOrderID: clientOrderID,
	}
	return nil
}

func (p *Poloniex) processAccountKilledOrder(notification []any) error {
	if len(notification) < 3 {
		return errNotEnoughData
	}

	orderID, ok := notification[1].(float64)
	if !ok {
		return fmt.Errorf("%w order ID not float64", errTypeAssertionFailure)
	}

	// null returned so ok check is not needed
	clientOrderID, _ := notification[2].(string)

	p.Websocket.DataHandler <- &order.Detail{
		Exchange:      p.Name,
		OrderID:       strconv.FormatFloat(orderID, 'f', -1, 64),
		Status:        order.Cancelled,
		AssetType:     asset.Spot,
		ClientOrderID: clientOrderID,
	}
	return nil
}

func (p *Poloniex) processTrades(currencyID float64, subData []any) error {
	if !p.IsSaveTradeDataEnabled() {
		return nil
	}
	pair, err := p.details.GetPair(currencyID)
	if err != nil {
		return err
	}

	if len(subData) != 6 {
		return errNotEnoughData
	}

	var tradeID string
	switch tradeIDData := subData[1].(type) { // tradeID type intermittently changes
	case string:
		tradeID = tradeIDData
	case float64:
		tradeID = strconv.FormatFloat(tradeIDData, 'f', -1, 64)
	default:
		return fmt.Errorf("unhandled type for websocket trade update: %v",
			tradeIDData)
	}

	orderSide, ok := subData[2].(float64)
	if !ok {
		return fmt.Errorf("%w order side not float64",
			errTypeAssertionFailure)
	}

	side := order.Buy
	if orderSide != 1 {
		side = order.Sell
	}

	v, ok := subData[3].(string)
	if !ok {
		return fmt.Errorf("%w volume not string",
			errTypeAssertionFailure)
	}
	volume, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return err
	}
	rate, ok := subData[4].(string)
	if !ok {
		return fmt.Errorf("%w rate not string", errTypeAssertionFailure)
	}
	price, err := strconv.ParseFloat(rate, 64)
	if err != nil {
		return err
	}
	timestamp, ok := subData[5].(float64)
	if !ok {
		return fmt.Errorf("%w time not float64", errTypeAssertionFailure)
	}

	return p.AddTradesToBuffer(trade.Data{
		TID:          tradeID,
		Exchange:     p.Name,
		CurrencyPair: pair,
		AssetType:    asset.Spot,
		Side:         side,
		Price:        price,
		Amount:       volume,
		Timestamp:    time.Unix(int64(timestamp), 0),
	})
}
