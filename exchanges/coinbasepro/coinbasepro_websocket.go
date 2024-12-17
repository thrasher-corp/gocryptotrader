package coinbasepro

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/buger/jsonparser"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
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
	// Subscriptions to status return an "authentication failure" error, despite the endpoint not being authenticated and other authenticated channels working fine.
	{Enabled: false, Channel: "status"},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.CandlesChannel},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.AllTradesChannel},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.OrderbookChannel},
	{Enabled: true, Asset: asset.All, Channel: subscription.MyAccountChannel, Authenticated: true},
	{Enabled: false, Asset: asset.Spot, Channel: "ticker_batch"},
	/* Not Implemented:
	{Enabled: false, Asset: asset.Spot, Channel: "futures_balance_summary", Authenticated: true},
	*/
}

// WsConnect initiates a websocket connection
func (c *CoinbasePro) WsConnect() error {
	if !c.Websocket.IsEnabled() || !c.IsEnabled() {
		return stream.ErrWebsocketNotEnabled
	}
	var dialer websocket.Dialer
	err := c.Websocket.Conn.Dial(&dialer, http.Header{})
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
		warn, err := c.wsHandleData(resp.Raw, seqCount)
		if err != nil {
			c.Websocket.DataHandler <- err
		}
		if warn != "" {
			c.Websocket.DataHandler <- warn
			tempStr := strings.SplitN(warn, "Out of order sequence number. Received ", 2)[1]
			tempStr = strings.SplitN(tempStr, ", expected ", 2)[0]
			tempNum, err := strconv.ParseUint(tempStr, 10, 64)
			if err != nil {
				c.Websocket.DataHandler <- err
			} else {
				seqCount = tempNum
			}
		}
		seqCount++
	}
}

// wsHandleData handles all the websocket data coming from the websocket connection
func (c *CoinbasePro) wsHandleData(respRaw []byte, seqCount uint64) (string, error) {
	var warnString string
	ertype, _, _, err := jsonparser.Get(respRaw, "type")
	if err == nil && string(ertype) == "error" {
		return warnString, errors.New(string(respRaw))
	}
	seqData, _, _, err := jsonparser.Get(respRaw, "sequence_num")
	if err != nil {
		return warnString, err
	}
	seqNum, err := strconv.ParseUint(string(seqData), 10, 64)
	if err != nil {
		return warnString, err
	}
	if seqNum != seqCount {
		warnString = fmt.Sprintf(warnSequenceIssue, seqNum, seqCount)
	}
	channelRaw, _, _, err := jsonparser.Get(respRaw, "channel")
	if err != nil {
		return warnString, err
	}
	channel := string(channelRaw)
	if channel == "subscriptions" || channel == "heartbeats" {
		return warnString, nil
	}
	data, _, _, err := jsonparser.Get(respRaw, "events")
	if err != nil {
		return warnString, err
	}
	switch channel {
	case "status":
		wsStatus := []WebsocketProductHolder{}
		err = json.Unmarshal(data, &wsStatus)
		if err != nil {
			return warnString, err
		}
		c.Websocket.DataHandler <- wsStatus
	case "ticker", "ticker_batch":
		wsTicker := []WebsocketTickerHolder{}
		err = json.Unmarshal(data, &wsTicker)
		if err != nil {
			return warnString, err
		}
		sliToSend := []ticker.Price{}
		var timestamp time.Time
		timestamp, err = getTimestamp(respRaw)
		if err != nil {
			return warnString, err
		}
		for i := range wsTicker {
			for j := range wsTicker[i].Tickers {
				sliToSend = append(sliToSend, ticker.Price{
					LastUpdated:  timestamp,
					Pair:         wsTicker[i].Tickers[j].ProductID,
					AssetType:    asset.Spot,
					ExchangeName: c.Name,
					High:         wsTicker[i].Tickers[j].High24H,
					Low:          wsTicker[i].Tickers[j].Low24H,
					Last:         wsTicker[i].Tickers[j].Price,
					Volume:       wsTicker[i].Tickers[j].Volume24H,
				})
			}
		}
		c.Websocket.DataHandler <- sliToSend
	case "candles":
		wsCandles := []WebsocketCandleHolder{}
		err = json.Unmarshal(data, &wsCandles)
		if err != nil {
			return warnString, err
		}
		sliToSend := []stream.KlineData{}
		var timestamp time.Time
		timestamp, err = getTimestamp(respRaw)
		if err != nil {
			return warnString, err
		}
		for i := range wsCandles {
			for j := range wsCandles[i].Candles {
				sliToSend = append(sliToSend, stream.KlineData{
					Timestamp:  timestamp,
					Pair:       wsCandles[i].Candles[j].ProductID,
					AssetType:  asset.Spot,
					Exchange:   c.Name,
					StartTime:  wsCandles[i].Candles[j].Start.Time(),
					OpenPrice:  wsCandles[i].Candles[j].Open,
					ClosePrice: wsCandles[i].Candles[j].Close,
					HighPrice:  wsCandles[i].Candles[j].High,
					LowPrice:   wsCandles[i].Candles[j].Low,
					Volume:     wsCandles[i].Candles[j].Volume,
				})
			}
		}
		c.Websocket.DataHandler <- sliToSend
	case "market_trades":
		wsTrades := []WebsocketMarketTradeHolder{}
		err = json.Unmarshal(data, &wsTrades)
		if err != nil {
			return warnString, err
		}
		sliToSend := []trade.Data{}
		for i := range wsTrades {
			for j := range wsTrades[i].Trades {
				sliToSend = append(sliToSend, trade.Data{
					TID:          wsTrades[i].Trades[j].TradeID,
					Exchange:     c.Name,
					CurrencyPair: wsTrades[i].Trades[j].ProductID,
					AssetType:    asset.Spot,
					Side:         wsTrades[i].Trades[j].Side,
					Price:        wsTrades[i].Trades[j].Price,
					Amount:       wsTrades[i].Trades[j].Size,
					Timestamp:    wsTrades[i].Trades[j].Time,
				})
			}
		}
		c.Websocket.DataHandler <- sliToSend
	case "l2_data":
		var wsL2 []WebsocketOrderbookDataHolder
		err := json.Unmarshal(data, &wsL2)
		if err != nil {
			return warnString, err
		}
		timestamp, err := getTimestamp(respRaw)
		if err != nil {
			return warnString, err
		}
		for i := range wsL2 {
			switch wsL2[i].Type {
			case "snapshot":
				err = c.ProcessSnapshot(&wsL2[i], timestamp)
			case "update":
				err = c.ProcessUpdate(&wsL2[i], timestamp)
			default:
				err = fmt.Errorf("%w %v", errUnknownL2DataType, wsL2[i].Type)
			}
			if err != nil {
				return warnString, err
			}
		}
	case "user":
		var wsUser []WebsocketOrderDataHolder
		err := json.Unmarshal(data, &wsUser)
		if err != nil {
			return warnString, err
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
				var ioc, fok bool
				ioc, fok, err = strategyDecoder(wsUser[i].Orders[j].TimeInForce)
				if err != nil {
					c.Websocket.DataHandler <- order.ClassificationError{
						Exchange: c.Name,
						Err:      err,
					}
				}
				sliToSend = append(sliToSend, order.Detail{
					Price:             price,
					ClientOrderID:     wsUser[i].Orders[j].ClientOrderID,
					ExecutedAmount:    wsUser[i].Orders[j].CumulativeQuantity,
					RemainingAmount:   wsUser[i].Orders[j].LeavesQuantity,
					Amount:            wsUser[i].Orders[j].CumulativeQuantity + wsUser[i].Orders[j].LeavesQuantity,
					OrderID:           wsUser[i].Orders[j].OrderID,
					Side:              oSide,
					Type:              oType,
					PostOnly:          wsUser[i].Orders[j].PostOnly,
					Pair:              wsUser[i].Orders[j].ProductID,
					AssetType:         asset,
					Status:            oStatus,
					TriggerPrice:      wsUser[i].Orders[j].StopPrice,
					ImmediateOrCancel: ioc,
					FillOrKill:        fok,
					Fee:               wsUser[i].Orders[j].TotalFees,
					Date:              wsUser[i].Orders[j].CreationTime,
					CloseTime:         wsUser[i].Orders[j].EndTime,
					Exchange:          c.Name,
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
					Amount:     wsUser[i].Positions.PerpetualFuturesPositions[j].NetSize,
					Leverage:   wsUser[i].Positions.PerpetualFuturesPositions[j].Leverage,
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
					ContractAmount: wsUser[i].Positions.ExpiringFuturesPositions[j].NumberOfContracts,
					Price:          wsUser[i].Positions.ExpiringFuturesPositions[j].EntryPrice,
				})
			}
		}
		c.Websocket.DataHandler <- sliToSend
	default:
		return warnString, errChannelNameUnknown
	}
	return warnString, nil
}

// ProcessSnapshot processes the initial orderbook snap shot
func (c *CoinbasePro) ProcessSnapshot(snapshot *WebsocketOrderbookDataHolder, timestamp time.Time) error {
	bids, asks, err := processBidAskArray(snapshot)
	if err != nil {
		return err
	}
	return c.Websocket.Orderbook.LoadSnapshot(&orderbook.Base{
		Bids:            bids,
		Asks:            asks,
		Exchange:        c.Name,
		Pair:            snapshot.ProductID,
		Asset:           asset.Spot,
		LastUpdated:     timestamp,
		VerifyOrderbook: c.CanVerifyOrderbook,
	})
}

// ProcessUpdate updates the orderbook local cache
func (c *CoinbasePro) ProcessUpdate(update *WebsocketOrderbookDataHolder, timestamp time.Time) error {
	bids, asks, err := processBidAskArray(update)
	if err != nil {
		return err
	}
	obU := orderbook.Update{
		Bids:       bids,
		Asks:       asks,
		Pair:       update.ProductID,
		UpdateTime: timestamp,
		Asset:      asset.Spot,
	}
	return c.Websocket.Orderbook.Update(&obU)
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
	return c.ParallelChanOp(subs, func(subs subscription.List) error { return c.manageSubs("subscribe", subs) }, 1)
}

// Unsubscribe sends a websocket message to stop receiving data from a list of channels
func (c *CoinbasePro) Unsubscribe(subs subscription.List) error {
	return c.ParallelChanOp(subs, func(subs subscription.List) error { return c.manageSubs("unsubscribe", subs) }, 1)
}

// manageSub subscribes or unsubscribes from a list of websocket channels
func (c *CoinbasePro) manageSubs(op string, subs subscription.List) error {
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
			err = c.signWsRequest(r)
			if err != nil {
				errs = common.AppendError(errs, err)
				continue
			}
		}
		if err = c.Websocket.Conn.SendJSONMessage(context.TODO(), limitType, r); err == nil {
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

func (c *CoinbasePro) signWsRequest(r *WebsocketRequest) error {
	jwt, err := c.GetWSJWT()
	if err != nil {
		return err
	}
	r.JWT = jwt
	return nil
}

// GetWSJWT returns a JWT, using a stored one of it's provided, and generating a new one otherwise
func (c *CoinbasePro) GetWSJWT() (string, error) {
	c.mut.RLock()
	if c.jwtExpire.After(time.Now()) {
		retStr := c.jwt
		c.mut.RUnlock()
		return retStr, nil
	}
	go c.mut.RUnlock()
	c.mut.Lock()
	defer c.mut.Unlock()
	var err error
	c.jwt, c.jwtExpire, err = c.GetJWT(context.Background(), "")
	return c.jwt, err
}

// getTimestamp is a helper function which pulls a RFC3339-formatted timestamp from a byte slice of JSON data
func getTimestamp(rawData []byte) (time.Time, error) {
	data, _, _, err := jsonparser.Get(rawData, "timestamp")
	if err != nil {
		return time.Time{}, err
	}
	return time.Parse(time.RFC3339, string(data))
}

// processBidAskArray is a helper function that turns WebsocketOrderbookDataHolder into arrays of bids and asks
func processBidAskArray(data *WebsocketOrderbookDataHolder) (bids, asks orderbook.Tranches, err error) {
	bids = make(orderbook.Tranches, 0, len(data.Changes))
	asks = make(orderbook.Tranches, 0, len(data.Changes))
	for i := range data.Changes {
		change := orderbook.Tranche{Price: data.Changes[i].PriceLevel, Amount: data.Changes[i].NewQuantity}
		switch data.Changes[i].Side {
		case "bid":
			bids = append(bids, change)
		case "offer":
			asks = append(asks, change)
		default:
			return nil, nil, fmt.Errorf("%w %v", order.ErrSideIsInvalid, data.Changes[i].Side)
		}
	}
	bids.SortBids()
	asks.SortAsks()
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
		return order.UnknownStatus, fmt.Errorf("%w %v", errUnrecognisedStatusType, stat)
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
		return order.UnknownType, fmt.Errorf("%w %v", errUnrecognisedOrderType, str)
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
		return asset.Empty, fmt.Errorf("%w %v", errUnrecognisedAssetType, str)
	}
}

// strategyDecoder is a helper function that converts a Coinbase Pro time in force string to a few standardised bools
func strategyDecoder(str string) (ioc, fok bool, err error) {
	switch str {
	case "IMMEDIATE_OR_CANCEL":
		return true, false, nil
	case "FILL_OR_KILL":
		return false, true, nil
	case "GOOD_UNTIL_CANCELLED", "GOOD_UNTIL_DATE_TIME":
		return false, false, nil
	default:
		return false, false, fmt.Errorf("%w %v", errUnrecognisedStrategyType, str)
	}
}

// Base64URLEncode is a helper function that does some tweaks to standard Base64 encoding, in a way which JWT requires
func base64URLEncode(b []byte) string {
	s := crypto.Base64Encode(b)
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
