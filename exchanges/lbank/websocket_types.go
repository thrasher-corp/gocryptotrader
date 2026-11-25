package lbank

import (
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// websocketTimeLayout is the layout of the times LBank's websocket sends, which carry no zone and are in UTC+8
const websocketTimeLayout = "2006-01-02T15:04:05.999999999"

// WebsocketTime is a time LBank's websocket sends at UTC+8 without a zone, such as "2019-06-28T17:49:22.722"
type WebsocketTime time.Time

// UnmarshalJSON decodes a websocket time
func (t *WebsocketTime) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	if s == "" {
		*t = WebsocketTime{}
		return nil
	}
	parsed, err := time.ParseInLocation(websocketTimeLayout, s, beijingTime)
	if err != nil {
		return fmt.Errorf("%w: %w", errMalformedWebsocketTime, err)
	}
	*t = WebsocketTime(parsed)
	return nil
}

// Time returns the time
func (t WebsocketTime) Time() time.Time {
	return time.Time(t)
}

// websocketMessage holds the fields of any frame LBank's websocket sends: a ping, an error, or a pushed update under
// the key its type names
type websocketMessage struct {
	Action  string        `json:"action"`
	Ping    string        `json:"ping"`
	Type    string        `json:"type"`
	Pair    string        `json:"pair"`
	Status  string        `json:"status"`
	Message string        `json:"message"`
	Time    WebsocketTime `json:"TS"`
	// DepthTime is the time of an order book snapshot in Unix milliseconds
	DepthTime   types.Time            `json:"ds"`
	Ticker      *WebsocketTicker      `json:"tick"`
	Depth       *WebsocketDepth       `json:"depth"`
	Trade       *WebsocketTrade       `json:"trade"`
	Kline       *WebsocketKline       `json:"kbar"`
	OrderUpdate *WebsocketOrderUpdate `json:"orderUpdate"`
	AssetUpdate *WebsocketAssetUpdate `json:"data"`
}

// websocketSubscribeRequest subscribes to or unsubscribes from a channel
type websocketSubscribeRequest struct {
	Action       string `json:"action"`
	Subscribe    string `json:"subscribe"`
	Pair         string `json:"pair,omitempty"`
	Depth        string `json:"depth,omitempty"`
	Kline        string `json:"kbar,omitempty"`
	SubscribeKey string `json:"subscribeKey,omitempty"`
}

// websocketHeartbeatRequest is a ping, or the pong answering one of LBank's pings with its ID
type websocketHeartbeatRequest struct {
	Action string `json:"action"`
	Ping   string `json:"ping,omitempty"`
	Pong   string `json:"pong,omitempty"`
}

// WebsocketTicker holds a pair's latest price and 24 hour statistics
type WebsocketTicker struct {
	LastPrice              float64 `json:"latest"`
	High24Hour             float64 `json:"high"`
	Low24Hour              float64 `json:"low"`
	Volume24Hour           float64 `json:"vol"`
	Turnover24Hour         float64 `json:"turnover"`
	ChangePercentage24Hour float64 `json:"change"`
	// Direction is the side of the latest trade
	Direction string `json:"dir"`
	// BaseUSDPrice and BaseCNYPrice are the base currency's price in USD and CNY, QuoteUSDPrice and QuoteCNYPrice the
	// quote currency's
	BaseUSDPrice  float64 `json:"usd"`
	BaseCNYPrice  float64 `json:"cny"`
	QuoteUSDPrice float64 `json:"to_usd"`
	QuoteCNYPrice float64 `json:"to_cny"`
}

// WebsocketDepth holds a pair's order book snapshot, to the depth subscribed to
type WebsocketDepth struct {
	Asks orderbook.LevelsArrayPriceAmount `json:"asks"`
	Bids orderbook.LevelsArrayPriceAmount `json:"bids"`
}

// WebsocketTrade holds a public trade
type WebsocketTrade struct {
	Price         float64 `json:"price"`
	Quantity      float64 `json:"volume"`
	QuoteQuantity float64 `json:"amount"`
	// Direction is the side that took liquidity
	Direction string        `json:"direction"`
	Time      WebsocketTime `json:"TS"`
}

// WebsocketKline holds a candle
type WebsocketKline struct {
	// Slot is the candle's interval, such as 1min
	Slot       string        `json:"slot"`
	Time       WebsocketTime `json:"t"`
	Open       float64       `json:"o"`
	High       float64       `json:"h"`
	Low        float64       `json:"l"`
	Close      float64       `json:"c"`
	Volume     float64       `json:"v"`
	Turnover   float64       `json:"a"`
	TradeCount uint64        `json:"n"`
}

// WebsocketOrderUpdate holds an order's update, pushed when it is placed, trades or changes status
type WebsocketOrderUpdate struct {
	OrderID       string `json:"uuid"`
	CustomerID    string `json:"customerID"`
	Symbol        string `json:"symbol"`
	TransactionID string `json:"txUuid"`
	// Type is the order's side, buy or sell, as LBank documents it, or its side and type, such as buy_market
	Type        string       `json:"type"`
	OrderPrice  types.Number `json:"orderPrice"`
	OrderAmount types.Number `json:"orderAmt"`
	// Price is the trade's price when the order trades, and the order's price otherwise
	Price types.Number `json:"price"`
	// Amount and VolumePrice are the trade's base quantity and quote value
	Amount            types.Number `json:"amount"`
	VolumePrice       types.Number `json:"volumePrice"`
	Role              string       `json:"role"`
	AveragePrice      types.Number `json:"avgPrice"`
	AccumulatedAmount types.Number `json:"accAmt"`
	// RemainAmount is the base amount a sell order has left, and the quote value a buy order has left
	RemainAmount types.Number `json:"remainAmt"`
	// OrderStatus is -1 cancelled, 0 open, 1 partially filled, 2 filled, 3 partially filled and cancelled or 4
	// cancelling
	OrderStatus int64      `json:"orderStatus"`
	UpdateTime  types.Time `json:"updateTime"`
}

// WebsocketAssetUpdate holds a balance change
type WebsocketAssetUpdate struct {
	AssetCode currency.Code `json:"assetCode"`
	Total     types.Number  `json:"asset"`
	Free      types.Number  `json:"free"`
	Frozen    types.Number  `json:"freeze"`
	// UpdateType is DEPOSIT, WITHDRAW, ORDER_CREATE, ORDER_DEAL or ORDER_CANCEL
	UpdateType string     `json:"type"`
	Time       types.Time `json:"time"`
}
