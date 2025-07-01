package coinbasepro

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"text/template"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
)

const (
	coinbaseproWebsocketURL = "wss://advanced-trade-ws.coinbase.com"
)

var subscriptionNames = map[string]string{
	subscription.HeartbeatChannel: "heartbeats",
	subscription.TickerChannel:    "ticker",
	subscription.CandlesChannel:   "candles",
	subscription.AllTradesChannel: "market_trades",
	subscription.OrderbookChannel: "level2",
	subscription.MyAccountChannel: "user",
	"status":                      "status",
	"ticker_batch":                "ticker_batch",
	/* Not Implemented:
	"futures_balance_summary":                "futures_balance_summary",
	*/
}

var defaultSubscriptions = subscription.List{
	{Enabled: true, Channel: subscription.HeartbeatChannel},
	{Enabled: true, Asset: asset.All, Channel: "status"},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.CandlesChannel},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.AllTradesChannel},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.OrderbookChannel},
	{Enabled: true, Asset: asset.All, Channel: subscription.MyAccountChannel, Authenticated: true},
	{Enabled: true, Asset: asset.Spot, Channel: "ticker_batch"},
	/* Not Implemented:
	{Enabled: false, Asset: asset.Spot, Channel: "futures_balance_summary", Authenticated: true},
	*/
}

// WsConnect initiates a websocket connection
func (c *CoinbasePro) WsConnect() error {
	ctx := context.TODO()
	if !c.Websocket.IsEnabled() || !c.IsEnabled() {
		return websocket.ErrWebsocketNotEnabled
	}
	var dialer gws.Dialer
	err := c.Websocket.Conn.Dial(ctx, &dialer, http.Header{})
	if err != nil {
		return err
	}
	c.Websocket.Wg.Add(1)
	go c.wsReadData()
	return nil
}

// wsReadData receives and passes on websocket messages for processing
func (c *CoinbasePro) wsReadData() {
	defer c.Websocket.Wg.Done()
	var seqCount uint64
	for {
		resp := c.Websocket.Conn.ReadMessage()
		if resp.Raw == nil {
			return
		}
		sequence, err := c.wsHandleData(resp.Raw)
		if err != nil {
			c.Websocket.DataHandler <- err
		}
		if sequence != nil {
			if *sequence != seqCount {
				c.Websocket.DataHandler <- fmt.Sprintf(warnSequenceIssue, sequence, seqCount)
				seqCount = *sequence
			}
			seqCount++
		}
	}
}

// wsHandleData handles all the websocket data coming from the websocket connection
func (c *CoinbasePro) wsHandleData(respRaw []byte) (*uint64, error) {
	var inc StandardWebsocketResponse
	if err := json.Unmarshal(respRaw, &inc); err != nil {
		return nil, err
	}
	if inc.Error != "" {
		return &inc.Sequence, errors.New(inc.Error)
	}
	switch inc.Channel {
	case "subscriptions", "heartbeats":
		return &inc.Sequence, nil
	case "status":
		var wsStatus []WebsocketProductHolder
		if err := json.Unmarshal(inc.Events, &wsStatus); err != nil {
			return &inc.Sequence, err
		}
		c.Websocket.DataHandler <- wsStatus
	case "ticker", "ticker_batch":
		var wsTicker []WebsocketTickerHolder
		if err := json.Unmarshal(inc.Events, &wsTicker); err != nil {
			return &inc.Sequence, err
		}
		var sliToSend []ticker.Price
		aliases := c.pairAliases.GetAliases()
		for i := range wsTicker {
			for j := range wsTicker[i].Tickers {
				tickAlias := aliases[wsTicker[i].Tickers[j].ProductID]
				newTick := ticker.Price{
					LastUpdated:  inc.Timestamp,
					AssetType:    asset.Spot,
					ExchangeName: c.Name,
					High:         wsTicker[i].Tickers[j].High24H.Float64(),
					Low:          wsTicker[i].Tickers[j].Low24H.Float64(),
					Last:         wsTicker[i].Tickers[j].Price.Float64(),
					Volume:       wsTicker[i].Tickers[j].Volume24H.Float64(),
					Bid:          wsTicker[i].Tickers[j].BestBid.Float64(),
					BidSize:      wsTicker[i].Tickers[j].BestBidQuantity.Float64(),
					Ask:          wsTicker[i].Tickers[j].BestAsk.Float64(),
					AskSize:      wsTicker[i].Tickers[j].BestAskQuantity.Float64(),
				}
				var errs error
				for k := range tickAlias {
					isEnabled, err := c.CurrencyPairs.IsPairEnabled(tickAlias[k], asset.Spot)
					if err != nil {
						errs = common.AppendError(errs, err)
						continue
					}
					if isEnabled {
						newTick.Pair = tickAlias[k]
						sliToSend = append(sliToSend, newTick)
					}
				}
			}
		}
		c.Websocket.DataHandler <- sliToSend
	case "candles":
		var wsCandles []WebsocketCandleHolder
		if err := json.Unmarshal(inc.Events, &wsCandles); err != nil {
			return &inc.Sequence, err
		}
		var sliToSend []websocket.KlineData
		for i := range wsCandles {
			for j := range wsCandles[i].Candles {
				sliToSend = append(sliToSend, websocket.KlineData{
					Timestamp:  inc.Timestamp,
					Pair:       wsCandles[i].Candles[j].ProductID,
					AssetType:  asset.Spot,
					Exchange:   c.Name,
					StartTime:  wsCandles[i].Candles[j].Start.Time(),
					OpenPrice:  wsCandles[i].Candles[j].Open.Float64(),
					ClosePrice: wsCandles[i].Candles[j].Close.Float64(),
					HighPrice:  wsCandles[i].Candles[j].High.Float64(),
					LowPrice:   wsCandles[i].Candles[j].Low.Float64(),
					Volume:     wsCandles[i].Candles[j].Volume.Float64(),
				})
			}
		}
		c.Websocket.DataHandler <- sliToSend
	case "market_trades":
		var wsTrades []WebsocketMarketTradeHolder
		if err := json.Unmarshal(inc.Events, &wsTrades); err != nil {
			return &inc.Sequence, err
		}
		var sliToSend []trade.Data
		for i := range wsTrades {
			for j := range wsTrades[i].Trades {
				sliToSend = append(sliToSend, trade.Data{
					TID:          wsTrades[i].Trades[j].TradeID,
					Exchange:     c.Name,
					CurrencyPair: wsTrades[i].Trades[j].ProductID,
					AssetType:    asset.Spot,
					Side:         wsTrades[i].Trades[j].Side,
					Price:        wsTrades[i].Trades[j].Price.Float64(),
					Amount:       wsTrades[i].Trades[j].Size.Float64(),
					Timestamp:    wsTrades[i].Trades[j].Time,
				})
			}
		}
		c.Websocket.DataHandler <- sliToSend
	case "l2_data":
		var wsL2 []WebsocketOrderbookDataHolder
		err := json.Unmarshal(inc.Events, &wsL2)
		if err != nil {
			return &inc.Sequence, err
		}
		for i := range wsL2 {
			switch wsL2[i].Type {
			case "snapshot":
				err = c.ProcessSnapshot(&wsL2[i], inc.Timestamp)
			case "update":
				err = c.ProcessUpdate(&wsL2[i], inc.Timestamp)
			default:
				err = fmt.Errorf("%w %v", errUnknownL2DataType, wsL2[i].Type)
			}
			if err != nil {
				return &inc.Sequence, err
			}
		}
	case "user":
		var wsUser []WebsocketOrderDataHolder
		err := json.Unmarshal(inc.Events, &wsUser)
		if err != nil {
			return &inc.Sequence, err
		}
		var sliToSend []order.Detail
		for i := range wsUser {
			for j := range wsUser[i].Orders {
				var oType order.Type
				oType, err = stringToStandardType(wsUser[i].Orders[j].OrderType)
				if err != nil {
					c.Websocket.DataHandler <- order.ClassificationError{
						Exchange: c.Name,
						Err:      err,
					}
				}
				var oSide order.Side
				oSide, err = order.StringToOrderSide(wsUser[i].Orders[j].OrderSide)
				if err != nil {
					c.Websocket.DataHandler <- order.ClassificationError{
						Exchange: c.Name,
						Err:      err,
					}
				}
				var oStatus order.Status
				oStatus, err = statusToStandardStatus(wsUser[i].Orders[j].Status)
				if err != nil {
					c.Websocket.DataHandler <- order.ClassificationError{
						Exchange: c.Name,
						Err:      err,
					}
				}
				price := wsUser[i].Orders[j].AveragePrice
				if wsUser[i].Orders[j].LimitPrice != 0 {
					price = wsUser[i].Orders[j].LimitPrice
				}
				var asset asset.Item
				asset, err = stringToStandardAsset(wsUser[i].Orders[j].ProductType)
				if err != nil {
					c.Websocket.DataHandler <- order.ClassificationError{
						Exchange: c.Name,
						Err:      err,
					}
				}
				var tif order.TimeInForce
				tif, err = strategyDecoder(wsUser[i].Orders[j].TimeInForce)
				if err != nil {
					c.Websocket.DataHandler <- order.ClassificationError{
						Exchange: c.Name,
						Err:      err,
					}
				}
				if wsUser[i].Orders[j].PostOnly {
					tif |= order.PostOnly
				}
				sliToSend = append(sliToSend, order.Detail{
					Price:           price.Float64(),
					ClientOrderID:   wsUser[i].Orders[j].ClientOrderID,
					ExecutedAmount:  wsUser[i].Orders[j].CumulativeQuantity.Float64(),
					RemainingAmount: wsUser[i].Orders[j].LeavesQuantity.Float64(),
					Amount:          wsUser[i].Orders[j].CumulativeQuantity.Float64() + wsUser[i].Orders[j].LeavesQuantity.Float64(),
					OrderID:         wsUser[i].Orders[j].OrderID,
					Side:            oSide,
					Type:            oType,
					Pair:            wsUser[i].Orders[j].ProductID,
					AssetType:       asset,
					Status:          oStatus,
					TriggerPrice:    wsUser[i].Orders[j].StopPrice.Float64(),
					TimeInForce:     tif,
					Fee:             wsUser[i].Orders[j].TotalFees.Float64(),
					Date:            wsUser[i].Orders[j].CreationTime,
					CloseTime:       wsUser[i].Orders[j].EndTime,
					Exchange:        c.Name,
				})
			}
			for j := range wsUser[i].Positions.PerpetualFuturesPositions {
				var oSide order.Side
				oSide, err = order.StringToOrderSide(wsUser[i].Positions.PerpetualFuturesPositions[j].PositionSide)
				if err != nil {
					c.Websocket.DataHandler <- order.ClassificationError{
						Exchange: c.Name,
						Err:      err,
					}
				}
				var mType margin.Type
				mType, err = margin.StringToMarginType(wsUser[i].Positions.PerpetualFuturesPositions[j].MarginType)
				if err != nil {
					c.Websocket.DataHandler <- order.ClassificationError{
						Exchange: c.Name,
						Err:      err,
					}
				}
				sliToSend = append(sliToSend, order.Detail{
					Pair:       wsUser[i].Positions.PerpetualFuturesPositions[j].ProductID,
					Side:       oSide,
					MarginType: mType,
					Amount:     wsUser[i].Positions.PerpetualFuturesPositions[j].NetSize.Float64(),
					Leverage:   wsUser[i].Positions.PerpetualFuturesPositions[j].Leverage.Float64(),
					AssetType:  asset.Futures,
					Exchange:   c.Name,
				})
			}
			for j := range wsUser[i].Positions.ExpiringFuturesPositions {
				var oSide order.Side
				oSide, err = order.StringToOrderSide(wsUser[i].Positions.ExpiringFuturesPositions[j].Side)
				if err != nil {
					c.Websocket.DataHandler <- order.ClassificationError{
						Exchange: c.Name,
						Err:      err,
					}
				}
				sliToSend = append(sliToSend, order.Detail{
					Pair:           wsUser[i].Positions.ExpiringFuturesPositions[j].ProductID,
					Side:           oSide,
					ContractAmount: wsUser[i].Positions.ExpiringFuturesPositions[j].NumberOfContracts.Float64(),
					Price:          wsUser[i].Positions.ExpiringFuturesPositions[j].EntryPrice.Float64(),
				})
			}
		}
		c.Websocket.DataHandler <- sliToSend
	default:
		return &inc.Sequence, errChannelNameUnknown
	}
	return &inc.Sequence, nil
}

// ProcessSnapshot processes the initial orderbook snap shot
func (c *CoinbasePro) ProcessSnapshot(snapshot *WebsocketOrderbookDataHolder, timestamp time.Time) error {
	bids, asks, err := processBidAskArray(snapshot, true)
	if err != nil {
		return err
	}
	book := &orderbook.Book{
		Bids:              bids,
		Asks:              asks,
		Exchange:          c.Name,
		Pair:              snapshot.ProductID,
		Asset:             asset.Spot,
		LastUpdated:       timestamp,
		ValidateOrderbook: c.ValidateOrderbook,
	}
	for _, a := range c.pairAliases.GetAlias(snapshot.ProductID) {
		isEnabled, err := c.IsPairEnabled(a, asset.Spot)
		if err != nil {
			return err
		}
		if isEnabled {
			book.Pair = a
			err = c.Websocket.Orderbook.LoadSnapshot(book)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// ProcessUpdate updates the orderbook local cache
func (c *CoinbasePro) ProcessUpdate(update *WebsocketOrderbookDataHolder, timestamp time.Time) error {
	bids, asks, err := processBidAskArray(update, false)
	if err != nil {
		return err
	}
	obU := &orderbook.Update{
		Bids:       bids,
		Asks:       asks,
		Pair:       update.ProductID,
		UpdateTime: timestamp,
		Asset:      asset.Spot,
	}
	for _, a := range c.pairAliases.GetAlias(update.ProductID) {
		isEnabled, err := c.IsPairEnabled(a, asset.Spot)
		if err != nil {
			return err
		}
		if isEnabled {
			obU.Pair = a
			err = c.Websocket.Orderbook.Update(obU)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// GenerateSubscriptions adds default subscriptions to websocket to be handled by ManageSubscriptions()
func (c *CoinbasePro) generateSubscriptions() (subscription.List, error) {
	return c.Features.Subscriptions.ExpandTemplates(c)
}

// GetSubscriptionTemplate returns a subscription channel template
func (c *CoinbasePro) GetSubscriptionTemplate(_ *subscription.Subscription) (*template.Template, error) {
	return template.New("master.tmpl").Funcs(template.FuncMap{"channelName": channelName}).Parse(subTplText)
}

// Subscribe sends a websocket message to receive data from a list of channels
func (c *CoinbasePro) Subscribe(subs subscription.List) error {
	return c.ParallelChanOp(context.TODO(), subs, func(ctx context.Context, subs subscription.List) error { return c.manageSubs(ctx, "subscribe", subs) }, 1)
}

// Unsubscribe sends a websocket message to stop receiving data from a list of channels
func (c *CoinbasePro) Unsubscribe(subs subscription.List) error {
	return c.ParallelChanOp(context.TODO(), subs, func(ctx context.Context, subs subscription.List) error { return c.manageSubs(ctx, "unsubscribe", subs) }, 1)
}

// manageSubs subscribes or unsubscribes from a list of websocket channels
func (c *CoinbasePro) manageSubs(ctx context.Context, op string, subs subscription.List) error {
	var errs error
	subs, errs = subs.ExpandTemplates(c)
	for _, s := range subs {
		r := &WebsocketRequest{
			Type:       op,
			ProductIDs: s.Pairs,
			Channel:    s.QualifiedChannel,
			Timestamp:  strconv.FormatInt(time.Now().Unix(), 10),
		}
		var err error
		limitType := WSUnauthRate
		if s.Authenticated {
			limitType = WSAuthRate
			if r.JWT, err = c.GetWSJWT(ctx); err != nil {
				return err
			}
		}
		if err = c.Websocket.Conn.SendJSONMessage(ctx, limitType, r); err == nil {
			switch op {
			case "subscribe":
				err = c.Websocket.AddSuccessfulSubscriptions(c.Websocket.Conn, s)
			case "unsubscribe":
				err = c.Websocket.RemoveSubscriptions(c.Websocket.Conn, s)
			}
		}
		errs = common.AppendError(errs, err)
	}
	return errs
}

// GetWSJWT returns a JWT, using a stored one of it's provided, and generating a new one otherwise
func (c *CoinbasePro) GetWSJWT(ctx context.Context) (string, error) {
	c.jwtStruct.m.RLock()
	if c.jwtStruct.expiresAt.After(time.Now()) {
		retStr := c.jwtStruct.token
		c.jwtStruct.m.RUnlock()
		return retStr, nil
	}
	c.jwtStruct.m.RUnlock()
	c.jwtStruct.m.Lock()
	defer c.jwtStruct.m.Unlock()
	var err error
	c.jwtStruct.token, c.jwtStruct.expiresAt, err = c.GetJWT(ctx, "")
	return c.jwtStruct.token, err
}

// processBidAskArray is a helper function that turns WebsocketOrderbookDataHolder into arrays of bids and asks
func processBidAskArray(data *WebsocketOrderbookDataHolder, snapshot bool) (bids, asks orderbook.Levels, err error) {
	bids = make(orderbook.Levels, 0, len(data.Changes))
	asks = make(orderbook.Levels, 0, len(data.Changes))
	for i := range data.Changes {
		change := orderbook.Level{Price: data.Changes[i].PriceLevel.Float64(), Amount: data.Changes[i].NewQuantity.Float64()}
		switch data.Changes[i].Side {
		case "bid":
			bids = append(bids, change)
		case "offer":
			asks = append(asks, change)
		default:
			return nil, nil, fmt.Errorf("%w %v", order.ErrSideIsInvalid, data.Changes[i].Side)
		}
	}
	if snapshot {
		return slices.Clip(bids), slices.Clip(asks), nil
	}
	return bids, asks, nil
}

// statusToStandardStatus is a helper function that converts a Coinbase Pro status string to a standardised order.Status type
func statusToStandardStatus(stat string) (order.Status, error) {
	switch stat {
	case "PENDING":
		return order.New, nil
	case "OPEN":
		return order.Active, nil
	case "FILLED":
		return order.Filled, nil
	case "CANCELLED":
		return order.Cancelled, nil
	case "EXPIRED":
		return order.Expired, nil
	case "FAILED":
		return order.Rejected, nil
	default:
		return order.UnknownStatus, fmt.Errorf("%w %v", order.ErrUnsupportedStatusType, stat)
	}
}

// stringToStandardType is a helper function that converts a Coinbase Pro side string to a standardised order.Type type
func stringToStandardType(str string) (order.Type, error) {
	switch str {
	case "LIMIT_ORDER_TYPE":
		return order.Limit, nil
	case "MARKET_ORDER_TYPE":
		return order.Market, nil
	case "STOP_LIMIT_ORDER_TYPE":
		return order.StopLimit, nil
	default:
		return order.UnknownType, fmt.Errorf("%w %v", order.ErrUnrecognisedOrderType, str)
	}
}

// stringToStandardAsset is a helper function that converts a Coinbase Pro asset string to a standardised asset.Item type
func stringToStandardAsset(str string) (asset.Item, error) {
	switch str {
	case "SPOT":
		return asset.Spot, nil
	case "FUTURE":
		return asset.Futures, nil
	default:
		return asset.Empty, asset.ErrNotSupported
	}
}

// strategyDecoder is a helper function that converts a Coinbase Pro time in force string to a few standardised bools
func strategyDecoder(str string) (tif order.TimeInForce, err error) {
	switch str {
	case "IMMEDIATE_OR_CANCEL":
		return order.ImmediateOrCancel, nil
	case "FILL_OR_KILL":
		return order.FillOrKill, nil
	case "GOOD_UNTIL_CANCELLED":
		return order.GoodTillCancel, nil
	case "GOOD_UNTIL_DATE_TIME":
		return order.GoodTillDay | order.GoodTillTime, nil
	default:
		return order.UnknownTIF, fmt.Errorf("%w %v", errUnrecognisedStrategyType, str)
	}
}

// Base64URLEncode is a helper function that does some tweaks to standard Base64 encoding, in a way which JWT requires
func base64URLEncode(b []byte) string {
	s := base64.StdEncoding.EncodeToString(b)
	s = strings.Split(s, "=")[0]
	s = strings.ReplaceAll(s, "+", "-")
	s = strings.ReplaceAll(s, "/", "_")
	return s
}

// checkSubscriptions looks for incompatible subscriptions and if found replaces all with defaults
// This should be unnecessary and removable by mid-2025
func (c *CoinbasePro) checkSubscriptions() {
	for _, s := range c.Config.Features.Subscriptions {
		switch s.Channel {
		case "level2_batch", "matches":
			c.Config.Features.Subscriptions = defaultSubscriptions.Clone()
			c.Features.Subscriptions = c.Config.Features.Subscriptions.Enabled()
			return
		}
	}
}

func channelName(s *subscription.Subscription) string {
	if n, ok := subscriptionNames[s.Channel]; ok {
		return n
	}
	panic(fmt.Errorf("%w: %s", subscription.ErrNotSupported, s.Channel))
}

const subTplText = `
{{ range $asset, $pairs := $.AssetPairs }}
	{{- channelName $.S -}}
	{{- $.AssetSeparator }}
{{- end }}
`
