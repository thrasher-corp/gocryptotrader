package bybit

import (
	"sync"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/types"
)

var validCategory = []string{"spot", "linear", "inverse", "option"}

// supportedOptionsTypes Bybit does not offer a way to retrieve option denominations via its API
var supportedOptionsTypes = []string{"BTC", "ETH", "SOL"}

type orderbookResponse struct {
	Symbol    string            `json:"s"`
	Asks      [][2]types.Number `json:"a"`
	Bids      [][2]types.Number `json:"b"`
	Timestamp types.Time        `json:"ts"`
	UpdateID  int64             `json:"u"`
}

// Authenticate stores authentication variables required
type Authenticate struct {
	RequestID string `json:"req_id"`
	Args      []any  `json:"args"`
	Operation string `json:"op"`
}

// SubscriptionArgument represents a subscription arguments.
type SubscriptionArgument struct {
	auth           bool              `json:"-"`
	RequestID      string            `json:"req_id"`
	Operation      string            `json:"op"`
	Arguments      []string          `json:"args"`
	associatedSubs subscription.List `json:"-"`
}

// Fee holds fee information
type Fee struct {
	BaseCoin string       `json:"baseCoin"`
	Symbol   string       `json:"symbol"`
	Taker    types.Number `json:"takerFeeRate"`
	Maker    types.Number `json:"makerFeeRate"`
}

// AccountFee holds account fee information
type AccountFee struct {
	Category string `json:"category"`
	List     []Fee  `json:"list"`
}

// InstrumentsInfo represents a category, page indicator, and list of instrument information.
type InstrumentsInfo struct {
	Category       string           `json:"category"`
	List           []InstrumentInfo `json:"list"`
	NextPageCursor string           `json:"nextPageCursor"`
}

// InstrumentInfo holds all instrument info across
// spot, linear, option types
type InstrumentInfo struct {
	Symbol          string       `json:"symbol"`
	ContractType    string       `json:"contractType"`
	Innovation      string       `json:"innovation"`
	MarginTrading   string       `json:"marginTrading"`
	OptionsType     string       `json:"optionsType"`
	Status          string       `json:"status"`
	BaseCoin        string       `json:"baseCoin"`
	QuoteCoin       string       `json:"quoteCoin"`
	LaunchTime      types.Time   `json:"launchTime"`
	DeliveryTime    types.Time   `json:"deliveryTime"`
	DeliveryFeeRate types.Number `json:"deliveryFeeRate"`
	PriceScale      types.Number `json:"priceScale"`
	LeverageFilter  struct {
		MinLeverage  types.Number `json:"minLeverage"`
		MaxLeverage  types.Number `json:"maxLeverage"`
		LeverageStep types.Number `json:"leverageStep"`
	} `json:"leverageFilter"`
	PriceFilter struct {
		MinPrice types.Number `json:"minPrice"`
		MaxPrice types.Number `json:"maxPrice"`
		TickSize types.Number `json:"tickSize"`
	} `json:"priceFilter"`
	LotSizeFilter struct {
		MaxOrderQty         types.Number `json:"maxOrderQty"`
		MinOrderQty         types.Number `json:"minOrderQty"`
		QtyStep             types.Number `json:"qtyStep"`
		PostOnlyMaxOrderQty types.Number `json:"postOnlyMaxOrderQty"`
		BasePrecision       types.Number `json:"basePrecision"`
		QuotePrecision      types.Number `json:"quotePrecision"`
		MinOrderAmt         types.Number `json:"minOrderAmt"`
		MaxOrderAmt         types.Number `json:"maxOrderAmt"`
		MinNotionalValue    types.Number `json:"minNotionalValue"`
	} `json:"lotSizeFilter"`
	UnifiedMarginTrade bool   `json:"unifiedMarginTrade"`
	FundingInterval    int64  `json:"fundingInterval"`
	SettleCoin         string `json:"settleCoin"`
}

// RestResponse represents a REST response instance.
type RestResponse struct {
	RetCode    int64  `json:"retCode"`
	RetMsg     string `json:"retMsg"`
	Result     any    `json:"result"`
	RetExtInfo struct {
		List []ErrorMessage `json:"list"`
	} `json:"retExtInfo"`
	Time types.Time `json:"time"`
}

// KlineResponse represents a kline item list instance as an array of string.
type KlineResponse struct {
	Symbol   string      `json:"symbol"`
	Category string      `json:"category"`
	List     []KlineItem `json:"list"`
}

// KlineItem stores an individual kline data item
type KlineItem struct {
	StartTime types.Time
	Open      types.Number
	High      types.Number
	Low       types.Number
	Close     types.Number

	// not available for mark and index price kline data
	TradeVolume types.Number
	Turnover    types.Number
}

// UnmarshalJSON implements the json.Unmarshaler interface for KlineItem
func (k *KlineItem) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[7]any{&k.StartTime, &k.Open, &k.High, &k.Low, &k.Close, &k.TradeVolume, &k.Turnover})
}

// MarkPriceKlineResponse represents a kline data item.
type MarkPriceKlineResponse struct {
	Symbol   string      `json:"symbol"`
	Category string      `json:"category"`
	List     []KlineItem `json:"list"`
}

func constructOrderbook(o *orderbookResponse) (*Orderbook, error) {
	s := Orderbook{
		Symbol:         o.Symbol,
		UpdateID:       o.UpdateID,
		GenerationTime: o.Timestamp.Time(),
	}
	s.Bids = processOB(o.Bids)
	s.Asks = processOB(o.Asks)
	return &s, nil
}

// TickerData represents a list of ticker detailed information.
type TickerData struct {
	Category string       `json:"category"`
	List     []TickerItem `json:"list"`
}

// TickerItem represents a ticker item detail
type TickerItem struct {
	Symbol                 string       `json:"symbol"`
	TickDirection          string       `json:"tickDirection"`
	LastPrice              types.Number `json:"lastPrice"`
	IndexPrice             types.Number `json:"indexPrice"`
	MarkPrice              types.Number `json:"markPrice"`
	PrevPrice24H           types.Number `json:"prevPrice24h"`
	Price24HPcnt           types.Number `json:"price24hPcnt"`
	HighPrice24H           types.Number `json:"highPrice24h"`
	LowPrice24H            types.Number `json:"lowPrice24h"`
	PrevPrice1H            types.Number `json:"prevPrice1h"`
	OpenInterest           types.Number `json:"openInterest"`
	OpenInterestValue      types.Number `json:"openInterestValue"`
	Turnover24H            types.Number `json:"turnover24h"`
	Volume24H              types.Number `json:"volume24h"`
	FundingRate            types.Number `json:"fundingRate"`
	NextFundingTime        types.Time   `json:"nextFundingTime"`
	PredictedDeliveryPrice types.Number `json:"predictedDeliveryPrice"`
	BasisRate              types.Number `json:"basisRate"`
	DeliveryFeeRate        types.Number `json:"deliveryFeeRate"`
	DeliveryTime           types.Time   `json:"deliveryTime"`
	Ask1Size               types.Number `json:"ask1Size"`
	Bid1Price              types.Number `json:"bid1Price"`
	Ask1Price              types.Number `json:"ask1Price"`
	Bid1Size               types.Number `json:"bid1Size"`
	Basis                  types.Number `json:"basis"`
	Bid1Iv                 types.Number `json:"bid1Iv"`
	Ask1Iv                 types.Number `json:"ask1Iv"`
	MarkIv                 types.Number `json:"markIv"`
	MarkPriceIv            types.Number `json:"markPriceIv"`
	UnderlyingPrice        types.Number `json:"underlyingPrice"`
	TotalVolume            types.Number `json:"totalVolume"`
	TotalTurnover          types.Number `json:"totalTurnover"`
	Delta                  types.Number `json:"delta"`
	Gamma                  types.Number `json:"gamma"`
	Vega                   types.Number `json:"vega"`
	Theta                  types.Number `json:"theta"`
	Change24Hour           types.Number `json:"change24h"`
	UsdIndexPrice          types.Number `json:"usdIndexPrice"`
	BidPrice               types.Number `json:"bidPrice"`
	BidSize                types.Number `json:"bidSize"`
	BidIv                  types.Number `json:"bidIv"`
	AskPrice               types.Number `json:"askPrice"`
	AskSize                types.Number `json:"askSize"`
	AskIv                  types.Number `json:"askIv"`
}

// FundingRateHistory represents a funding rate history for a category.
type FundingRateHistory struct {
	Category string        `json:"category"`
	List     []FundingRate `json:"list"`
}

// FundingRate represents a funding rate instance.
type FundingRate struct {
	Symbol               string       `json:"symbol"`
	FundingRate          types.Number `json:"fundingRate"`
	FundingRateTimestamp types.Time   `json:"fundingRateTimestamp"`
}

// TradingHistory represents a trading history list.
type TradingHistory struct {
	Category string               `json:"category"`
	List     []TradingHistoryItem `json:"list"`
}

// TradingHistoryItem represents a trading history item instance.
type TradingHistoryItem struct {
	ExecutionID  string       `json:"execId"`
	Symbol       string       `json:"symbol"`
	Side         string       `json:"side"`
	Price        types.Number `json:"price"`
	Size         types.Number `json:"size"`
	TradeTime    types.Time   `json:"time"`
	IsBlockTrade bool         `json:"isBlockTrade"`
}

// OpenInterest represents open interest of each symbol.
type OpenInterest struct {
	Symbol   string `json:"symbol"`
	Category string `json:"category"`
	List     []struct {
		OpenInterest types.Number `json:"openInterest"`
		Timestamp    types.Time   `json:"timestamp"`
	} `json:"list"`
	NextPageCursor string `json:"nextPageCursor"`
}

// HistoricVolatility represents option historical volatility
type HistoricVolatility struct {
	Period int64        `json:"period"`
	Value  types.Number `json:"value"`
	Time   types.Time   `json:"time"`
}

// InsuranceHistory represents an insurance list.
type InsuranceHistory struct {
	UpdatedTime types.Time `json:"updatedTime"`
	List        []struct {
		Coin    string       `json:"coin"`
		Balance types.Number `json:"balance"`
		Value   types.Number `json:"value"`
	} `json:"list"`
}

// RiskLimitHistory represents risk limit history of a category.
type RiskLimitHistory struct {
	Category string `json:"category"`
	List     []struct {
		ID                int64        `json:"id"`
		IsLowestRisk      int64        `json:"isLowestRisk"`
		Symbol            string       `json:"symbol"`
		RiskLimitValue    types.Number `json:"riskLimitValue"`
		MaintenanceMargin types.Number `json:"maintenanceMargin"`
		InitialMargin     types.Number `json:"initialMargin"`
		MaxLeverage       types.Number `json:"maxLeverage"`
	} `json:"list"`
}

// DeliveryPrice represents the delivery price information.
type DeliveryPrice struct {
	Category       string `json:"category"`
	NextPageCursor string `json:"nextPageCursor"`
	List           []struct {
		Symbol        string       `json:"symbol"`
		DeliveryPrice types.Number `json:"deliveryPrice"`
		DeliveryTime  types.Time   `json:"deliveryTime"`
	} `json:"list"`
}

// PlaceOrderParams represents
type PlaceOrderParams struct {
	Category               string        `json:"category"`   // Required
	Symbol                 currency.Pair `json:"symbol"`     // Required
	Side                   string        `json:"side"`       // Required
	OrderType              string        `json:"orderType"`  // Required // Market, Limit
	OrderQuantity          float64       `json:"qty,string"` // Required // Order quantity. For Spot Market Buy order, please note that qty should be quote currency amount
	Price                  float64       `json:"price,string,omitempty"`
	TimeInForce            string        `json:"timeInForce,omitempty"`      // IOC and GTC
	OrderLinkID            string        `json:"orderLinkId,omitempty"`      // User customised order ID. A max of 36 characters. Combinations of numbers, letters (upper and lower cases), dashes, and underscores are supported. future orderLinkId rules:
	WhetherToBorrow        bool          `json:"-"`                          // '0' for default spot, '1' for Margin trading.
	IsLeverage             int64         `json:"isLeverage,omitempty"`       // Required   // '0' for default spot, '1' for Margin trading.
	OrderFilter            string        `json:"orderFilter,omitempty"`      // Valid for spot only. Order,tpslOrder. If not passed, Order by default
	TriggerDirection       int64         `json:"triggerDirection,omitempty"` // Required // Conditional order param. Used to identify the expected direction of the conditional order. '1': triggered when market price rises to triggerPrice '2': triggered when market price falls to triggerPrice
	TriggerPrice           float64       `json:"triggerPrice,omitempty,string"`
	TriggerPriceType       string        `json:"triggerBy,omitempty"` // Conditional order param. Trigger price type. 'LastPrice', 'IndexPrice', 'MarkPrice'
	OrderImpliedVolatility string        `json:"orderIv,omitempty"`
	PositionIdx            int64         `json:"positionIdx,omitempty"` // Under hedge-mode, this param is required '0': one-way mode '1': hedge-mode Buy side '2': hedge-mode Sell side
	TakeProfitPrice        float64       `json:"takeProfit,omitempty,string"`
	TakeProfitTriggerBy    string        `json:"tpTriggerBy,omitempty"` // The price type to trigger take profit. 'MarkPrice', 'IndexPrice', default: 'LastPrice'
	StopLossTriggerBy      string        `json:"slTriggerBy,omitempty"` // The price type to trigger stop loss. MarkPrice, IndexPrice, default: LastPrice
	StopLossPrice          float64       `json:"stopLoss,omitempty,string"`
	SMPExecutionType       string        `json:"smpType,omitempty"` // default: 'None', 'CancelMaker', 'CancelTaker', 'CancelBoth'
	ReduceOnly             bool          `json:"reduceOnly,omitempty"`
	CloseOnTrigger         bool          `json:"closeOnTrigger,omitempty"`
	MarketMakerProtection  bool          `json:"mmp,omitempty"` // option only. true means set the order as a market maker protection order.

	// TP/SL mode
	// "Full": entire position for TP/SL. Then, tpOrderType or slOrderType must be Market
	// 'Partial': partial position tp/sl. Limit TP/SL order are supported. Note: When create limit tp/sl, tpslMode is required and it must be Partial
	TpslMode     string  `json:"tpslMode,omitempty"`
	TpOrderType  string  `json:"tpOrderType,omitempty"`
	SlOrderType  string  `json:"slOrderType,omitempty"`
	TpLimitPrice float64 `json:"tpLimitPrice,omitempty,string"`
	SlLimitPrice float64 `json:"slLimitPrice,omitempty,string"`
}

// OrderResponse holds newly placed order information.
type OrderResponse struct {
	OrderID     string `json:"orderId"`
	OrderLinkID string `json:"orderLinkId"`
}

// AmendOrderParams represents a parameter for amending order.
type AmendOrderParams struct {
	Category               string        `json:"category,omitempty"`
	Symbol                 currency.Pair `json:"symbol,omitzero"`
	OrderID                string        `json:"orderId,omitempty"`
	OrderLinkID            string        `json:"orderLinkId,omitempty"` // User customised order ID. A max of 36 characters. Combinations of numbers, letters (upper and lower cases), dashes, and underscores are supported. future orderLinkId rules:
	OrderImpliedVolatility string        `json:"orderIv,omitempty"`
	TriggerPrice           float64       `json:"triggerPrice,omitempty,string"`
	OrderQuantity          float64       `json:"qty,omitempty,string"` // Order quantity. For Spot Market Buy order, please note that qty should be quote currency amount
	Price                  float64       `json:"price,string,omitempty"`

	TakeProfitPrice float64 `json:"takeProfit,omitempty,string"`
	StopLossPrice   float64 `json:"stopLoss,omitempty,string"`

	TakeProfitTriggerBy string `json:"tpTriggerBy,omitempty"` // The price type to trigger take profit. 'MarkPrice', 'IndexPrice', default: 'LastPrice'
	StopLossTriggerBy   string `json:"slTriggerBy,omitempty"` // The price type to trigger stop loss. MarkPrice, IndexPrice, default: LastPrice
	TriggerPriceType    string `json:"triggerBy,omitempty"`   // Conditional order param. Trigger price type. 'LastPrice', 'IndexPrice', 'MarkPrice'

	TakeProfitLimitPrice float64 `json:"tpLimitPrice,omitempty,string"`
	StopLossLimitPrice   float64 `json:"slLimitPrice,omitempty,string"`

	// TP/SL mode
	// Full: entire position for TP/SL. Then, tpOrderType or slOrderType must be Market
	// Partial: partial position tp/sl. Limit TP/SL order are supported. Note: When create limit tp/sl,
	// 'tpslMode' is required and it must be Partial
	// Valid for 'linear' & 'inverse'
	TPSLMode string `json:"tpslMode,omitempty"`
}

// CancelOrderParams represents a cancel order parameters.
type CancelOrderParams struct {
	Category    string        `json:"category,omitempty"`
	Symbol      currency.Pair `json:"symbol,omitzero"`
	OrderID     string        `json:"orderId,omitempty"`
	OrderLinkID string        `json:"orderLinkId,omitempty"` // User customised order ID. A max of 36 characters. Combinations of numbers, letters (upper and lower cases), dashes, and underscores are supported. future orderLinkId rules:

	OrderFilter string `json:"orderFilter,omitempty"` // Valid for spot only. Order,tpslOrder. If not passed, Order by default
}

// TradeOrders represents category and list of trade orders of the category.
type TradeOrders struct {
	List           []TradeOrder `json:"list"`
	NextPageCursor string       `json:"nextPageCursor"`
	Category       string       `json:"category"`
}

// TradeOrder represents a trade order details.
type TradeOrder struct {
	OrderID                string       `json:"orderId"`
	OrderLinkID            string       `json:"orderLinkId"`
	BlockTradeID           string       `json:"blockTradeId"`
	Symbol                 string       `json:"symbol"`
	Price                  types.Number `json:"price"`
	OrderQuantity          types.Number `json:"qty"`
	Side                   string       `json:"side"`
	IsLeverage             string       `json:"isLeverage"`
	PositionIdx            int64        `json:"positionIdx"`
	OrderStatus            string       `json:"orderStatus"`
	CancelType             string       `json:"cancelType"`
	RejectReason           string       `json:"rejectReason"`
	AveragePrice           types.Number `json:"avgPrice"`
	LeavesQuantity         types.Number `json:"leavesQty"`
	LeavesValue            string       `json:"leavesValue"`
	CumulativeExecQuantity types.Number `json:"cumExecQty"`
	CumulativeExecValue    types.Number `json:"cumExecValue"`
	CumulativeExecFee      types.Number `json:"cumExecFee"`
	TimeInForce            string       `json:"timeInForce"`
	OrderType              string       `json:"orderType"`
	StopOrderType          string       `json:"stopOrderType"`
	OrderIv                string       `json:"orderIv"`
	TriggerPrice           types.Number `json:"triggerPrice"`
	TakeProfitPrice        types.Number `json:"takeProfit"`
	StopLossPrice          types.Number `json:"stopLoss"`
	TpTriggerBy            string       `json:"tpTriggerBy"`
	SlTriggerBy            string       `json:"slTriggerBy"`
	TriggerDirection       int64        `json:"triggerDirection"`
	TriggerBy              string       `json:"triggerBy"`
	LastPriceOnCreated     string       `json:"lastPriceOnCreated"`
	ReduceOnly             bool         `json:"reduceOnly"`
	CloseOnTrigger         bool         `json:"closeOnTrigger"`
	SmpType                string       `json:"smpType"`
	SmpGroup               int64        `json:"smpGroup"`
	SmpOrderID             string       `json:"smpOrderId"`
	TpslMode               string       `json:"tpslMode"`
	TpLimitPrice           types.Number `json:"tpLimitPrice"`
	SlLimitPrice           types.Number `json:"slLimitPrice"`
	PlaceType              string       `json:"placeType"`
	CreatedTime            types.Time   `json:"createdTime"`
	UpdatedTime            types.Time   `json:"updatedTime"`

	// UTA Spot: add new response field 'ocoTriggerBy',
	// and the value can be 'OcoTriggerByUnknown', 'OcoTriggerByTp', 'OcoTriggerBySl'
	OCOTriggerType string `json:"ocoTriggerType"`
	OCOTriggerBy   string `json:"ocoTriggerBy"`
}

// CancelAllResponse represents a cancel all trade orders response.
type CancelAllResponse struct {
	List []OrderResponse `json:"list"`
}

// CancelAllOrdersParam request parameters for cancel all orders.
type CancelAllOrdersParam struct {
	Category    string        `json:"category"`
	Symbol      currency.Pair `json:"symbol"`
	BaseCoin    string        `json:"baseCoin,omitempty"`
	SettleCoin  string        `json:"settleCoin,omitempty"`
	OrderFilter string        `json:"orderFilter,omitempty"` // Valid for spot only. Order,tpslOrder. If not passed, Order by default

	// Possible value: Stop. Only used for category=linear or inverse and orderFilter=StopOrder,
	// you can cancel conditional orders except TP/SL order and Trailing stop orders with this param
	StopOrderType string `json:"stopOrderType,omitempty"`
}

// PlaceBatchOrderParam represents a parameter for placing batch orders
type PlaceBatchOrderParam struct {
	Category string                `json:"category"`
	Request  []BatchOrderItemParam `json:"request"`
}

// BatchOrderItemParam represents a batch order place parameter.
type BatchOrderItemParam struct {
	Category         string        `json:"category,omitempty"`
	Symbol           currency.Pair `json:"symbol,omitzero"`
	OrderType        string        `json:"orderType,omitempty"`
	Side             string        `json:"side,omitempty"`
	OrderQuantity    float64       `json:"qty,string,omitempty"`
	Price            float64       `json:"price,string,omitempty"`
	TriggerDirection int64         `json:"triggerDirection,omitempty"`
	TriggerPrice     int64         `json:"triggerPrice,omitempty"`
	OrderIv          int64         `json:"orderIv,omitempty,string"`
	TriggerBy        string        `json:"triggerBy,omitempty"` // Possible values:  LastPrice, IndexPrice, and MarkPrice
	TimeInForce      string        `json:"timeInForce,omitempty"`

	// PositionIndex Used to identify positions in different position modes. Under hedge-mode,
	// this param is required (USDT perps have hedge mode)
	// 0: one-way mode 1: hedge-mode Buy side 2: hedge-mode Sell side
	PositionIndex         int64  `json:"positionIdx,omitempty"`
	OrderLinkID           string `json:"orderLinkId,omitempty"`
	TakeProfit            string `json:"takeProfit,omitempty"`  // Take profit price, valid for linear
	StopLoss              string `json:"stopLoss,omitempty"`    // Stop loss price, valid for linear
	TakeProfitTriggerBy   string `json:"tpTriggerBy,omitempty"` // MarkPrice, IndexPrice, default: LastPrice. Valid for linear
	StopLossTriggerBy     string `json:"slTriggerBy,omitempty"` // MarkPrice, IndexPrice, default: LastPrice
	SMPType               string `json:"smpType,omitempty"`
	ReduceOnly            bool   `json:"reduceOnly,omitempty"`
	CloseOnTrigger        bool   `json:"closeOnTrigger,omitempty"`
	MarketMakerProtection bool   `json:"mmp,omitempty"`
	TPSLMode              string `json:"tpslMode,omitempty"`
	TakeProfitLimitPrice  string `json:"tpLimitPrice,omitempty"`
	StopLossLimitPrice    string `json:"slLimitPrice,omitempty"`
	TakeProfitOrderType   string `json:"tpOrderType,omitempty"`
	StopLossOrderType     string `json:"slOrderType,omitempty"`
}

// BatchOrdersList represents a list trade orders.
type BatchOrdersList struct {
	List []BatchOrderResponse `json:"list"`
}

// BatchOrderResponse represents a batch trade order item response.
type BatchOrderResponse struct {
	Category    string     `json:"category"`
	Symbol      string     `json:"symbol"`
	OrderID     string     `json:"orderId"`
	OrderLinkID string     `json:"orderLinkId"`
	CreateAt    types.Time `json:"createAt"`
}

// ErrorMessage represents an error message item
type ErrorMessage struct {
	Code    int64  `json:"code"`
	Message string `json:"msg"`
}

// BatchAmendOrderParams request parameter for batch amend order.
type BatchAmendOrderParams struct {
	Category string                     `json:"category"`
	Request  []BatchAmendOrderParamItem `json:"request"`
}

// BatchAmendOrderParamItem represents a single order amend item in batch amend order
type BatchAmendOrderParamItem struct {
	Symbol                 currency.Pair `json:"symbol"`
	OrderID                string        `json:"orderId,omitempty"`
	OrderLinkID            string        `json:"orderLinkId,omitempty"` // User customised order ID. A max of 36 characters. Combinations of numbers, letters (upper and lower cases), dashes, and underscores are supported. future orderLinkId rules:
	OrderImpliedVolatility string        `json:"orderIv,omitempty"`
	OrderQuantity          float64       `json:"qty,omitempty,string"` // Order quantity. For Spot Market Buy order, please note that qty should be quote currency amount
	Price                  float64       `json:"price,string,omitempty"`

	// TP/SL mode
	// Full: entire position for TP/SL. Then, tpOrderType or slOrderType must be Market
	// Partial: partial position tp/sl. Limit TP/SL order are supported. Note: When create limit tp/sl,
	// 'tpslMode' is required and it must be Partial
	// Valid for 'linear' & 'inverse'
	TPSLMode string `json:"tpslMode,omitempty"`
}

// CancelBatchOrder represents a batch cancel request parameters.
type CancelBatchOrder struct {
	Category string              `json:"category"`
	Request  []CancelOrderParams `json:"request"`
}

// CancelBatchResponseItem represents a batch cancel response item.
type CancelBatchResponseItem struct {
	Category    string `json:"category,omitempty"`
	Symbol      string `json:"symbol,omitempty"`
	OrderID     string `json:"orderId,omitempty"`
	OrderLinkID string `json:"orderLinkId,omitempty"`
}

type cancelBatchResponse struct {
	List []CancelBatchResponseItem `json:"list"`
}

// BorrowQuota represents
type BorrowQuota struct {
	Symbol               string       `json:"symbol"`
	MaxTradeQty          string       `json:"maxTradeQty"`
	Side                 string       `json:"side"`
	MaxTradeAmount       string       `json:"maxTradeAmount"`
	BorrowCoin           string       `json:"borrowCoin"`
	SpotMaxTradeQuantity types.Number `json:"spotMaxTradeQty"`
	SpotMaxTradeAmount   types.Number `json:"spotMaxTradeAmount"`
}

// SetDCPParams represents the set disconnect cancel all parameters.
type SetDCPParams struct {
	TimeWindow int64 `json:"timeWindow"`
}

// PositionInfoList represents a list of positions infos.
type PositionInfoList struct {
	Category       string         `json:"category"`
	NextPageCursor string         `json:"nextPageCursor"`
	List           []PositionInfo `json:"list"`
}

// PositionInfo represents a position info item.
type PositionInfo struct {
	Symbol           string       `json:"symbol"`
	Side             string       `json:"side"`
	Size             types.Number `json:"size"`
	AveragePrice     types.Number `json:"avgPrice"`
	PositionValue    types.Number `json:"positionValue"`
	TradeMode        int64        `json:"tradeMode"`
	PositionStatus   string       `json:"positionStatus"`
	AutoAddMargin    int64        `json:"autoAddMargin"`
	ADLRankIndicator int64        `json:"adlRankIndicator"`
	Leverage         types.Number `json:"leverage"`
	PositionBalance  types.Number `json:"positionBalance"`
	MarkPrice        types.Number `json:"markPrice"`
	LiqPrice         types.Number `json:"liqPrice"`
	BustPrice        types.Number `json:"bustPrice"`
	PositionMM       types.Number `json:"positionMM"`
	PositionIM       types.Number `json:"positionIM"`
	TpslMode         string       `json:"tpslMode"`
	TakeProfit       types.Number `json:"takeProfit"`
	StopLoss         types.Number `json:"stopLoss"`
	TrailingStop     types.Number `json:"trailingStop"`
	UnrealisedPnl    types.Number `json:"unrealisedPnl"`
	PositionIndex    int64        `json:"positionIdx"`
	RiskID           int64        `json:"riskId"`
	RiskLimitValue   string       `json:"riskLimitValue"`

	// Futures & Perp: it is the all time cumulative realised P&L
	// Option: it is the realised P&L when you hold that position
	CumRealisedPnl types.Number `json:"cumRealisedPnl"`
	CreatedTime    types.Time   `json:"createdTime"`
	UpdatedTime    types.Time   `json:"updatedTime"`

	IsReduceOnly           bool       `json:"isReduceOnly"`
	MMRSysUpdatedTime      types.Time `json:"mmrSysUpdatedTime"`
	LeverageSysUpdatedTime types.Time `json:"leverageSysUpdatedTime"`

	// Cross sequence, used to associate each fill and each position update
	// Different symbols may have the same seq, please use seq + symbol to check unique
	// Returns "-1" if the symbol has never been traded
	// Returns the seq updated by the last transaction when there are settings like leverage, risk limit
	Sequence int64 `json:"seq"`
}

// SetLeverageParams parameters to set the leverage.
type SetLeverageParams struct {
	Category     string  `json:"category"`
	Symbol       string  `json:"symbol"`
	BuyLeverage  float64 `json:"buyLeverage,string"`  // [0, max leverage of corresponding risk limit]. Note: Under one-way mode, buyLeverage must be the same as sellLeverage
	SellLeverage float64 `json:"sellLeverage,string"` // [0, max leverage of corresponding risk limit]. Note: Under one-way mode, buyLeverage must be the same as sellLeverage
}

// SwitchTradeModeParams parameters to switch between cross margin and isolated margin trade mode.
type SwitchTradeModeParams struct {
	Category     string  `json:"category"`
	Symbol       string  `json:"symbol"`
	BuyLeverage  float64 `json:"buyLeverage,string"`
	SellLeverage float64 `json:"sellLeverage,string"`
	TradeMode    int64   `json:"tradeMode"` // 0: cross margin. 1: isolated margin
}

// TPSLModeParams parameters for settle Take Profit(TP) or Stop Loss(SL) mode.
type TPSLModeParams struct {
	Category string `json:"category"`
	Symbol   string `json:"symbol"`
	TpslMode string `json:"tpSlMode"` // TP/SL mode. Full,Partial
}

// TPSLModeResponse represents response for the take profit and stop loss mode change.
type TPSLModeResponse struct {
	TPSLMode string `json:"tpSlMode"`
}

// SwitchPositionModeParams represents a position switch mode parameters.
type SwitchPositionModeParams struct {
	Category     string        `json:"category"`
	Symbol       currency.Pair `json:"symbol"` // Symbol name. Either symbol or coin is required. symbol has a higher priority
	Coin         currency.Code `json:"coin"`
	PositionMode int64         `json:"mode"` // Position mode. 0: Merged Single. 3: Both Sides
}

// SetRiskLimitParam represents a risk limit set parameter.
type SetRiskLimitParam struct {
	Category     string        `json:"category"`
	Symbol       currency.Pair `json:"symbol"` // Symbol name. Either symbol or coin is required. symbol has a higher priority
	RiskID       int64         `json:"riskId"`
	PositionMode int64         `json:"positionIdx"` // Used to identify positions in different position modes. For hedge mode, it is required '0': one-way mode '1': hedge-mode Buy side '2': hedge-mode Sell side
}

// RiskLimitResponse represents a risk limit response.
type RiskLimitResponse struct {
	RiskID         int64  `json:"riskId"`
	RiskLimitValue string `json:"riskLimitValue"`
	Category       string `json:"category"`
}

// TradingStopParams take profit, stop loss or trailing stop for the position.
type TradingStopParams struct {
	Category                 string        `json:"category"`
	Symbol                   currency.Pair `json:"symbol"`                 // Symbol name. Either symbol or coin is required. symbol has a higher priority
	TakeProfit               string        `json:"takeProfit,omitempty"`   // Cannot be less than 0, 0 means cancel TP
	StopLoss                 string        `json:"stopLoss,omitempty"`     // Cannot be less than 0, 0 means cancel SL
	TrailingStop             string        `json:"trailingStop,omitempty"` // Trailing stop by price distance. Cannot be less than 0, 0 means cancel TS
	TakeProfitTriggerType    string        `json:"tpTriggerBy,omitempty"`
	StopLossTriggerType      string        `json:"slTriggerBy,omitempty"`
	ActivePrice              float64       `json:"activePrice,omitempty,string"`
	TakeProfitOrStopLossMode string        `json:"tpslMode,omitempty"`
	TakeProfitOrderType      string        `json:"tpOrderType,omitempty"`
	StopLossOrderType        string        `json:"slOrderType,omitempty"`
	TakeProfitSize           float64       `json:"tpSize,string,omitempty"`
	StopLossSize             float64       `json:"slSize,string,omitempty"`
	TakeProfitLimitPrice     float64       `json:"tpLimitPrice,string,omitempty"`
	StopLossLimitPrice       float64       `json:"slLimitPrice,string,omitempty"`
	PositionIndex            int64         `json:"positionIdx,omitempty"`
}

// AutoAddMarginParam represents parameters for auto add margin
type AutoAddMarginParam struct {
	Category      string        `json:"category"`
	Symbol        currency.Pair `json:"symbol"`
	AutoAddmargin int64         `json:"autoAddMargin,string"` // Turn on/off. 0: off. 1: on

	// Positions in different position modes.
	// 0: one-way mode, 1: hedge-mode Buy side, 2: hedge-mode Sell side
	PositionIndex int64 `json:"positionIdx,omitempty,string"`
}

// AddOrReduceMarginParam holds manually add or reduce margin for isolated margin position parameters.
type AddOrReduceMarginParam struct {
	Category      string        `json:"category"`
	Symbol        currency.Pair `json:"symbol"`
	Margin        int64         `json:"margin,string"` // Add or reduce. To add, then 10; To reduce, then -10. Support up to 4 decimal
	PositionIndex int64         `json:"positionIdx"`   // Same as PositionIndex value in AutoAddMarginParam
}

// AddOrReduceMargin represents a add or reduce margin response.
type AddOrReduceMargin struct {
	Category                 string       `json:"category"`
	Symbol                   string       `json:"symbol"`
	PositionIndex            int64        `json:"positionIdx"` // position mode index
	RiskID                   int64        `json:"riskId"`
	RiskLimitValue           string       `json:"riskLimitValue"`
	Size                     types.Number `json:"size"`
	PositionValue            string       `json:"positionValue"`
	AveragePrice             types.Number `json:"avgPrice"`
	LiquidationPrice         types.Number `json:"liqPrice"`
	BustPrice                types.Number `json:"bustPrice"`
	MarkPrice                types.Number `json:"markPrice"`
	Leverage                 string       `json:"leverage"`
	AutoAddMargin            int64        `json:"autoAddMargin"`
	PositionStatus           string       `json:"positionStatus"`
	PositionIM               types.Number `json:"positionIM"`
	PositionMM               types.Number `json:"positionMM"`
	UnrealisedProfitAndLoss  types.Number `json:"unrealisedPnl"`
	CumRealisedProfitAndLoss types.Number `json:"cumRealisedPnl"`
	StopLoss                 types.Number `json:"stopLoss"`
	TakeProfit               types.Number `json:"takeProfit"`
	TrailingStop             types.Number `json:"trailingStop"`
	CreatedTime              types.Time   `json:"createdTime"`
	UpdatedTime              types.Time   `json:"updatedTime"`
}

// ExecutionResponse represents users order execution response
type ExecutionResponse struct {
	NextPageCursor string      `json:"nextPageCursor"`
	Category       string      `json:"category"`
	List           []Execution `json:"list"`
}

// Execution represents execution record
type Execution struct {
	Symbol                 string       `json:"symbol"`
	OrderType              string       `json:"orderType"`
	UnderlyingPrice        types.Number `json:"underlyingPrice"`
	IndexPrice             types.Number `json:"indexPrice"`
	OrderLinkID            string       `json:"orderLinkId"`
	Side                   string       `json:"side"`
	OrderID                string       `json:"orderId"`
	StopOrderType          string       `json:"stopOrderType"`
	LeavesQuantity         types.Number `json:"leavesQty"`
	ExecTime               types.Time   `json:"execTime"`
	IsMaker                bool         `json:"isMaker"`
	ExecFee                types.Number `json:"execFee"`
	FeeRate                types.Number `json:"feeRate"`
	ExecID                 string       `json:"execId"`
	TradeImpliedVolatility string       `json:"tradeIv"`
	BlockTradeID           string       `json:"blockTradeId"`
	MarkPrice              types.Number `json:"markPrice"`
	ExecPrice              types.Number `json:"execPrice"`
	MarkIv                 string       `json:"markIv"`
	OrderQuantity          types.Number `json:"orderQty"`
	ExecValue              string       `json:"execValue"`
	ExecType               string       `json:"execType"`
	OrderPrice             types.Number `json:"orderPrice"`
	ExecQuantity           types.Number `json:"execQty"`
	ClosedSize             types.Number `json:"closedSize"`
}

// ClosedProfitAndLossResponse represents list of closed profit and loss records
type ClosedProfitAndLossResponse struct {
	NextPageCursor string `json:"nextPageCursor"`
	Category       string `json:"category"`
	List           []struct {
		Symbol              string       `json:"symbol"`
		OrderID             string       `json:"orderId"`
		Side                string       `json:"side"`
		Quantity            types.Number `json:"qty"`
		OrderPrice          types.Number `json:"orderPrice"`
		OrderType           string       `json:"orderType"`
		ExecutionType       string       `json:"execType"`
		ClosedSize          types.Number `json:"closedSize"`
		CumulatedEntryValue string       `json:"cumEntryValue"`
		AvgEntryPrice       types.Number `json:"avgEntryPrice"`
		CumulatedExitValue  string       `json:"cumExitValue"`
		AvgExitPrice        types.Number `json:"avgExitPrice"`
		ClosedPnl           string       `json:"closedPnl"`
		FillCount           types.Number `json:"fillCount"`
		Leverage            string       `json:"leverage"`
		CreatedTime         types.Time   `json:"createdTime"`
		UpdatedTime         types.Time   `json:"updatedTime"`
	} `json:"list"`
}

type paramsConfig struct {
	Spot            bool
	Option          bool
	Linear          bool
	Inverse         bool
	MandatorySymbol bool

	OptionalCategory bool
	OptionalBaseCoin bool
}

// TransactionLog represents a transaction log history.
type TransactionLog struct {
	NextPageCursor string               `json:"nextPageCursor"`
	List           []TransactionLogItem `json:"list"`
}

// TransactionLogItem represents a transaction log item information.
type TransactionLogItem struct {
	Symbol          string       `json:"symbol"`
	Side            string       `json:"side"`
	Funding         string       `json:"funding"`
	OrderLinkID     string       `json:"orderLinkId"`
	OrderID         string       `json:"orderId"`
	Fee             types.Number `json:"fee"`
	Change          string       `json:"change"`
	CashFlow        string       `json:"cashFlow"`
	TransactionTime types.Time   `json:"transactionTime"`
	Type            string       `json:"type"`
	FeeRate         types.Number `json:"feeRate"`
	BonusChange     types.Number `json:"bonusChange"`
	Size            types.Number `json:"size"`
	Qty             types.Number `json:"qty"`
	CashBalance     types.Number `json:"cashBalance"`
	Currency        string       `json:"currency"`
	Category        string       `json:"category"`
	TradePrice      types.Number `json:"tradePrice"`
	TradeID         string       `json:"tradeId"`
}

// PreUpdateOptionDeliveryRecord represents delivery records of Option
type PreUpdateOptionDeliveryRecord struct {
	NextPageCursor string `json:"nextPageCursor"`
	Category       string `json:"category"`
	List           []struct {
		Symbol        string       `json:"symbol"`
		Side          string       `json:"side"`
		DeliveryTime  types.Time   `json:"deliveryTime"`
		ExercisePrice types.Number `json:"strike"`
		Fee           types.Number `json:"fee"`
		Position      string       `json:"position"`
		DeliveryPrice types.Number `json:"deliveryPrice"`
		DeliveryRpl   string       `json:"deliveryRpl"` // Realized PnL of the delivery
	} `json:"list"`
}

// SettlementSession represents a USDC settlement session.
type SettlementSession struct {
	NextPageCursor string `json:"nextPageCursor"`
	Category       string `json:"category"`
	List           []struct {
		RealisedProfitAndLoss string       `json:"realisedPnl"`
		Symbol                string       `json:"symbol"`
		Side                  string       `json:"side"`
		MarkPrice             types.Number `json:"markPrice"`
		Size                  types.Number `json:"size"`
		CreatedTime           types.Time   `json:"createdTime"`
		SessionAveragePrice   types.Number `json:"sessionAvgPrice"`
	} `json:"list"`
}

// WalletBalance represents wallet balance
type WalletBalance struct {
	List []struct {
		TotalEquity            types.Number `json:"totalEquity"`
		AccountIMRate          types.Number `json:"accountIMRate"`
		TotalMarginBalance     types.Number `json:"totalMarginBalance"`
		TotalInitialMargin     types.Number `json:"totalInitialMargin"`
		AccountType            string       `json:"accountType"`
		TotalAvailableBalance  types.Number `json:"totalAvailableBalance"`
		AccountMMRate          types.Number `json:"accountMMRate"`
		TotalPerpUPL           types.Number `json:"totalPerpUPL"`
		TotalWalletBalance     types.Number `json:"totalWalletBalance"`
		AccountLTV             types.Number `json:"accountLTV"` // Account LTV: account total borrowed size / (account total equity + account total borrowed size).
		TotalMaintenanceMargin types.Number `json:"totalMaintenanceMargin"`
		Coin                   []struct {
			AvailableToBorrow       types.Number  `json:"availableToBorrow"`
			Bonus                   types.Number  `json:"bonus"`
			AccruedInterest         types.Number  `json:"accruedInterest"`
			AvailableToWithdraw     types.Number  `json:"availableToWithdraw"`
			AvailableBalanceForSpot types.Number  `json:"free"`
			TotalOrderIM            types.Number  `json:"totalOrderIM"`
			Equity                  types.Number  `json:"equity"`
			Locked                  types.Number  `json:"locked"`
			MarginCollateral        bool          `json:"marginCollateral"`
			SpotHedgingQuantity     types.Number  `json:"spotHedgingQty"`
			TotalPositionMM         types.Number  `json:"totalPositionMM"`
			USDValue                types.Number  `json:"usdValue"`
			UnrealisedPNL           types.Number  `json:"unrealisedPnl"`
			BorrowAmount            types.Number  `json:"borrowAmount"`
			TotalPositionIM         types.Number  `json:"totalPositionIM"`
			WalletBalance           types.Number  `json:"walletBalance"`
			CumulativeRealisedPNL   types.Number  `json:"cumRealisedPnl"`
			Coin                    currency.Code `json:"coin"`
			CollateralSwitch        bool          `json:"collateralSwitch"`
		} `json:"coin"`
	} `json:"list"`
}

// UnifiedAccountUpgradeResponse represents a response parameter for update to unified account.
type UnifiedAccountUpgradeResponse struct {
	UnifiedUpdateStatus string `json:"unifiedUpdateStatus"`
	UnifiedUpdateMsg    struct {
		Messages []string `json:"msg"`
	} `json:"unifiedUpdateMsg"`
}

// BorrowHistory represents interest records.
type BorrowHistory struct {
	NextPageCursor string `json:"nextPageCursor"`
	List           []struct {
		Currency                  string       `json:"currency"`
		CreatedTime               types.Time   `json:"createdTime"`
		BorrowCost                types.Number `json:"borrowCost"`
		HourlyBorrowRate          types.Number `json:"hourlyBorrowRate"`
		InterestBearingBorrowSize types.Number `json:"InterestBearingBorrowSize"`
		CostExemption             string       `json:"costExemption"`
		BorrowAmount              types.Number `json:"borrowAmount"`
		UnrealisedLoss            types.Number `json:"unrealisedLoss"`
		FreeBorrowedAmount        types.Number `json:"freeBorrowedAmount"`
	} `json:"list"`
}

// CollateralInfo represents collateral information of the current unified margin account.
type CollateralInfo struct {
	List []struct {
		Currency            string       `json:"currency"`
		HourlyBorrowRate    types.Number `json:"hourlyBorrowRate"`
		MaxBorrowingAmount  types.Number `json:"maxBorrowingAmount"`
		FreeBorrowingAmount types.Number `json:"freeBorrowingAmount"`
		FreeBorrowingLimit  types.Number `json:"freeBorrowingLimit"`
		FreeBorrowAmount    types.Number `json:"freeBorrowAmount"` // The amount of borrowing within your total borrowing amount that is exempt from interest charges
		BorrowAmount        types.Number `json:"borrowAmount"`
		AvailableToBorrow   types.Number `json:"availableToBorrow"`
		Borrowable          bool         `json:"borrowable"`
		BorrowUsageRate     types.Number `json:"borrowUsageRate"`
		MarginCollateral    bool         `json:"marginCollateral"`
		CollateralSwitch    bool         `json:"collateralSwitch"`
		CollateralRatio     types.Number `json:"collateralRatio"` // Collateral ratio
	} `json:"list"`
}

// CoinGreeks represents current account greeks information.
type CoinGreeks struct {
	List []struct {
		BaseCoin   string       `json:"baseCoin"`
		TotalDelta types.Number `json:"totalDelta"`
		TotalGamma types.Number `json:"totalGamma"`
		TotalVega  types.Number `json:"totalVega"`
		TotalTheta types.Number `json:"totalTheta"`
	} `json:"list"`
}

// FeeRate represents maker and taker fee rate information for a symbol.
type FeeRate struct {
	Symbol       string       `json:"symbol"`
	TakerFeeRate types.Number `json:"takerFeeRate"`
	MakerFeeRate types.Number `json:"makerFeeRate"`
}

// AccountInfo represents margin mode account information.
type AccountInfo struct {
	UnifiedMarginStatus int64      `json:"unifiedMarginStatus"`
	MarginMode          string     `json:"marginMode"` // ISOLATED_MARGIN, REGULAR_MARGIN, PORTFOLIO_MARGIN
	DcpStatus           string     `json:"dcpStatus"`  // Disconnected-CancelAll-Prevention status: ON, OFF
	TimeWindow          int64      `json:"timeWindow"`
	SmpGroup            int64      `json:"smpGroup"`
	IsMasterTrader      bool       `json:"isMasterTrader"`
	SpotHedgingStatus   string     `json:"spotHedgingStatus"`
	UpdatedTime         types.Time `json:"updatedTime"`
}

// SetMarginModeResponse represents a response for setting margin mode.
type SetMarginModeResponse struct {
	Reasons []struct {
		ReasonCode string `json:"reasonCode"`
		ReasonMsg  string `json:"reasonMsg"`
	} `json:"reasons"`
}

// MMPRequestParam represents an MMP request parameter.
type MMPRequestParam struct {
	BaseCoin           string       `json:"baseCoin"`
	TimeWindowMS       int64        `json:"window,string"`
	FrozenPeriod       int64        `json:"frozenPeriod,string"`
	TradeQuantityLimit types.Number `json:"qtyLimit"`
	DeltaLimit         types.Number `json:"deltaLimit"`
}

// MMPStates represents an MMP states.
type MMPStates struct {
	Result []struct {
		BaseCoin           string       `json:"baseCoin"`
		MmpEnabled         bool         `json:"mmpEnabled"`
		Window             string       `json:"window"`
		FrozenPeriod       string       `json:"frozenPeriod"`
		TradeQuantityLimit types.Number `json:"qtyLimit"`
		DeltaLimit         types.Number `json:"deltaLimit"`
		MmpFrozenUntil     types.Number `json:"mmpFrozenUntil"`
		MmpFrozen          bool         `json:"mmpFrozen"`
	} `json:"result"`
}

// CoinExchangeRecords represents a coin exchange records.
type CoinExchangeRecords struct {
	OrderBody []struct {
		FromCoin              string       `json:"fromCoin"`
		FromAmount            types.Number `json:"fromAmount"`
		ToCoin                string       `json:"toCoin"`
		ToAmount              types.Number `json:"toAmount"`
		ExchangeRate          types.Number `json:"exchangeRate"`
		CreatedTime           types.Time   `json:"createdTime"`
		ExchangeTransactionID string       `json:"exchangeTxId"`
	} `json:"orderBody"`
	NextPageCursor string `json:"nextPageCursor"`
}

// DeliveryRecord represents delivery records of USDC futures and Options.
type DeliveryRecord struct {
	NextPageCursor string `json:"nextPageCursor"`
	Category       string `json:"category"`
	List           []struct {
		Symbol                        string       `json:"symbol"`
		Side                          string       `json:"side"`
		DeliveryTime                  types.Time   `json:"deliveryTime"`
		ExercisePrice                 types.Number `json:"strike"`
		Fee                           types.Number `json:"fee"`
		Position                      types.Number `json:"position"`
		DeliveryPrice                 types.Number `json:"deliveryPrice"`
		DeliveryRealizedProfitAndLoss types.Number `json:"deliveryRpl"`
	} `json:"list"`
}

// AccountInfos represents account type and account information
type AccountInfos map[string]struct {
	Status string `json:"status"`
	Assets []struct {
		Coin     string `json:"coin"`
		Frozen   string `json:"frozen"`
		Free     string `json:"free"`
		Withdraw string `json:"withdraw"`
	} `json:"assets"`
}

// CoinBalances represents coin balances for a specific asset type.
type CoinBalances struct {
	AccountType string `json:"accountType"`
	BizType     int64  `json:"bizType"`
	AccountID   string `json:"accountId"`
	MemberID    string `json:"memberId"`
	Balance     []struct {
		Coin               string       `json:"coin"`
		WalletBalance      types.Number `json:"walletBalance"`
		TransferBalance    types.Number `json:"transferBalance"`
		Bonus              types.Number `json:"bonus"`
		TransferSafeAmount string       `json:"transferSafeAmount"`
	} `json:"balance"`
}

// CoinBalance represents coin balance for a specific asset type.
type CoinBalance struct {
	AccountType string `json:"accountType"`
	BizType     int64  `json:"bizType"`
	AccountID   string `json:"accountId"`
	MemberID    string `json:"memberId"`
	Balance     struct {
		Coin               string       `json:"coin"`
		WalletBalance      types.Number `json:"walletBalance"`
		TransferBalance    types.Number `json:"transferBalance"`
		Bonus              types.Number `json:"bonus"`
		TransferSafeAmount string       `json:"transferSafeAmount"`
	} `json:"balance"`
}

// TransferableCoins represents list of transferable coins.
type TransferableCoins struct {
	List []string `json:"list"`
}

// TransferParams represents parameters from internal coin transfer.
type TransferParams struct {
	TransferID      uuid.UUID     `json:"transferId"`
	Coin            currency.Code `json:"coin"`
	Amount          types.Number  `json:"amount,string"`
	FromAccountType string        `json:"fromAccountType"`
	ToAccountType   string        `json:"toAccountType"`

	// Added for universal transfers
	FromMemberID int64 `json:"fromMemberId"`
	ToMemberID   int64 `json:"toMemberId"`
}

// TransferResponse represents a transfer response
type TransferResponse struct {
	List []struct {
		TransferID      string       `json:"transferId"`
		Coin            string       `json:"coin"`
		Amount          types.Number `json:"amount"`
		FromAccountType string       `json:"fromAccountType"`
		ToAccountType   string       `json:"toAccountType"`
		Timestamp       types.Time   `json:"timestamp"`
		Status          string       `json:"status"`

		// Returned with universal transfer IDs.
		FromMemberID string `json:"fromMemberId"`
		ToMemberID   string `json:"toMemberId"`
	} `json:"list"`
	NextPageCursor string `json:"nextPageCursor"`
}

// SubUID represents a sub-users ID
type SubUID struct {
	SubMemberIDs             []string `json:"subMemberIds"`
	TransferableSubMemberIDs []string `json:"transferableSubMemberIds"`
}

// AllowedDepositCoinInfo represents coin deposit information.
type AllowedDepositCoinInfo struct {
	ConfigList []struct {
		Coin               string `json:"coin"`
		Chain              string `json:"chain"`
		CoinShowName       string `json:"coinShowName"`
		ChainType          string `json:"chainType"`
		BlockConfirmNumber int64  `json:"blockConfirmNumber"`
		MinDepositAmount   string `json:"minDepositAmount"`
	} `json:"configList"`
	NextPageCursor string `json:"nextPageCursor"`
}

// StatusResponse represents account information
type StatusResponse struct {
	Status int64 `json:"status"` // 1: SUCCESS 0: FAIL
}

// DepositRecords represents deposit records
type DepositRecords struct {
	Rows []struct {
		Coin          string `json:"coin"`
		Chain         string `json:"chain"`
		Amount        string `json:"amount"`
		TxID          string `json:"txID"`
		Status        int64  `json:"status"`
		ToAddress     string `json:"toAddress"`
		Tag           string `json:"tag"`
		DepositFee    string `json:"depositFee"`
		SuccessAt     string `json:"successAt"`
		Confirmations string `json:"confirmations"`
		TxIndex       string `json:"txIndex"`
		BlockHash     string `json:"blockHash"`
	} `json:"rows"`
	NextPageCursor string `json:"nextPageCursor"`
}

// InternalDepositRecords represents internal deposit records response instances
type InternalDepositRecords struct {
	Rows []struct {
		ID          string       `json:"id"`
		Amount      types.Number `json:"amount"`
		Type        int64        `json:"type"`
		Coin        string       `json:"coin"`
		Address     string       `json:"address"`
		Status      int64        `json:"status"`
		CreatedTime types.Time   `json:"createdTime"`
	} `json:"rows"`
	NextPageCursor string `json:"nextPageCursor"`
}

// DepositAddresses represents deposit address information.
type DepositAddresses struct {
	Coin   string `json:"coin"`
	Chains []struct {
		ChainType      string `json:"chainType"`
		AddressDeposit string `json:"addressDeposit"`
		TagDeposit     string `json:"tagDeposit"`
		Chain          string `json:"chain"`
	} `json:"chains"`
}

// CoinInfo represents coin info information.
type CoinInfo struct {
	Rows []struct {
		Name         string       `json:"name"`
		Coin         string       `json:"coin"`
		RemainAmount types.Number `json:"remainAmount"`
		Chains       []struct {
			ChainType             string       `json:"chainType"`
			Confirmation          string       `json:"confirmation"`
			WithdrawFee           types.Number `json:"withdrawFee"`
			DepositMin            types.Number `json:"depositMin"`
			WithdrawMin           types.Number `json:"withdrawMin"`
			Chain                 string       `json:"chain"`
			ChainDeposit          types.Number `json:"chainDeposit"`
			ChainWithdraw         string       `json:"chainWithdraw"`
			MinAccuracy           types.Number `json:"minAccuracy"`
			WithdrawPercentageFee types.Number `json:"withdrawPercentageFee"`
		} `json:"chains"`
	} `json:"rows"`
}

// WithdrawalRecords represents a list of withdrawal records.
type WithdrawalRecords struct {
	Rows []struct {
		Coin          string       `json:"coin"`
		Chain         string       `json:"chain"`
		Amount        types.Number `json:"amount"`
		TransactionID string       `json:"txID"`
		Status        string       `json:"status"`
		ToAddress     string       `json:"toAddress"`
		Tag           string       `json:"tag"`
		WithdrawFee   types.Number `json:"withdrawFee"`
		CreateTime    types.Time   `json:"createTime"`
		UpdateTime    types.Time   `json:"updateTime"`
		WithdrawID    string       `json:"withdrawId"`
		WithdrawType  int64        `json:"withdrawType"`
	} `json:"rows"`
	NextPageCursor string `json:"nextPageCursor"`
}

// WithdrawableAmount represents withdrawable amount information for each currency code
type WithdrawableAmount struct {
	LimitAmountUsd     string `json:"limitAmountUsd"`
	WithdrawableAmount map[string]struct {
		Coin               string       `json:"coin"`
		WithdrawableAmount types.Number `json:"withdrawableAmount"`
		AvailableBalance   types.Number `json:"availableBalance"`
	} `json:"withdrawableAmount"`
}

// WithdrawalParam represents asset withdrawal request parameter.
type WithdrawalParam struct {
	Coin        currency.Code `json:"coin,omitzero"`
	Chain       string        `json:"chain,omitempty"`
	Address     string        `json:"address,omitempty"`
	Tag         string        `json:"tag,omitempty"`
	Amount      float64       `json:"amount,omitempty,string"`
	Timestamp   int64         `json:"timestamp,omitempty"`
	ForceChain  int64         `json:"forceChain,omitempty"` // Whether or not to force an on-chain withdrawal '0'(default): If the address is parsed out to be an internal address, then internal transfer '1': Force the withdrawal to occur on-chain '2': Use UID to withdraw
	AccountType string        `json:"accountType,omitempty"`
}

// CreateSubUserParams parameter to create a new sub user id. Use master user's api key only.
type CreateSubUserParams struct {
	Username   string `json:"username,omitempty"`   // Give a username of the new sub user id.
	Password   string `json:"password,omitempty"`   // Set the password for the new sub user id. 8-30 characters, must include numbers, capital and little letters.
	MemberType int64  `json:"memberType,omitempty"` // '1': normal sub account, '6': custodial sub account
	Switch     int64  `json:"switch,omitempty"`     // '0': turn off quick login (default) '1': turn on quick login
	IsUTC      bool   `json:"isUta,omitempty"`
	Note       string `json:"note,omitempty"`
}

// SubUserItem represents a sub user response instance.
type SubUserItem struct {
	UID        string `json:"uid"`
	Username   string `json:"username"`
	MemberType int64  `json:"memberType"`
	Status     int64  `json:"status"`
	Remark     string `json:"remark"`
}

// SubUIDAPIKeyParam represents a sub-user ID API key creation parameter.
type SubUIDAPIKeyParam struct {
	Subuid int64  `json:"subuid"`
	Note   string `json:"note"`

	ReadOnly int64 `json:"readOnly"`

	// Set the IP bind. example: ["192.168.0.1,192.168.0.2"]note:
	// don't pass ips or pass with ["*"] means no bind
	// No ip bound api key will be invalid after 90 days
	// api key will be invalid after 7 days once the account password is changed
	IPs []string `json:"ips"`

	// Tick the types of permission. one of below types must be passed, otherwise the error is thrown
	Permissions map[string][]string `json:"permissions,omitempty"`
}

// SubUIDAPIResponse represents sub UID API key response.
type SubUIDAPIResponse struct {
	ID          string              `json:"id"`
	Note        string              `json:"note"`
	APIKey      string              `json:"apiKey"`
	ReadOnly    int64               `json:"readOnly"`
	Secret      string              `json:"secret"`
	Permissions map[string][]string `json:"permissions"`

	IPS                   []string  `json:"ips"`
	Type                  int64     `json:"type"`
	DeadlineDay           int64     `json:"deadlineDay"`
	ExpiredAt             time.Time `json:"expiredAt"`
	CreatedAt             time.Time `json:"createdAt"`
	IsMarginUnified       int64     `json:"unified"` // Whether the account to which the account upgrade to unified margin account.
	IsUnifiedTradeAccount uint8     `json:"uta"`     // Whether the account to which the account upgrade to unified trade account.
	UserID                int64     `json:"userID"`
	InviterID             int64     `json:"inviterID"`
	VipLevel              string    `json:"vipLevel"`
	MktMakerLevel         string    `json:"mktMakerLevel"`
	AffiliateID           int64     `json:"affiliateID"`
	RsaPublicKey          string    `json:"rsaPublicKey"`
	IsMaster              bool      `json:"isMaster"`

	// Personal account kyc level. LEVEL_DEFAULT, LEVEL_1， LEVEL_2
	KycLevel  string `json:"kycLevel"`
	KycRegion string `json:"kycRegion"`
}

// SubAccountAPIKeys holds list of sub-account API Keys
type SubAccountAPIKeys struct {
	Result []struct {
		ID          string    `json:"id"`
		Ips         []string  `json:"ips"`
		APIKey      string    `json:"apiKey"`
		Note        string    `json:"note"`
		Status      int64     `json:"status"`
		ExpiredAt   time.Time `json:"expiredAt"`
		CreatedAt   time.Time `json:"createdAt"`
		Type        int64     `json:"type"`
		Permissions struct {
			ContractTrade []string `json:"ContractTrade"`
			Spot          []string `json:"Spot"`
			Wallet        []string `json:"Wallet"`
			Options       []string `json:"Options"`
			Derivatives   []string `json:"Derivatives"`
			CopyTrading   []string `json:"CopyTrading"`
			BlockTrade    []string `json:"BlockTrade"`
			Exchange      []string `json:"Exchange"`
			Nft           []string `json:"NFT"`
			Affiliate     []string `json:"Affiliate"`
		} `json:"permissions"`
		Secret      string `json:"secret"`
		ReadOnly    bool   `json:"readOnly"`
		DeadlineDay int64  `json:"deadlineDay"`
		Flag        string `json:"flag"`
	} `json:"result"`
	NextPageCursor string `json:"nextPageCursor"`
}

// WalletType represents available wallet types for the master account or sub account
type WalletType struct {
	Accounts []struct {
		UID         string   `json:"uid"`
		AccountType []string `json:"accountType"`
	} `json:"accounts"`
}

// SubUIDAPIKeyUpdateParam represents a sub-user ID API key update parameter.
type SubUIDAPIKeyUpdateParam struct {
	APIKey   string `json:"apikey"`
	ReadOnly int64  `json:"readOnly,omitempty"`
	// Set the IP bind. example: ["192.168.0.1,192.168.0.2"]note:
	// don't pass ips or pass with ["*"] means no bind
	// No ip bound api key will be invalid after 90 days
	// api key will be invalid after 7 days once the account password is changed
	IPs string `json:"ips"`

	// You can provide the IP addresses as a list of strings.
	IPAddresses []string `json:"-"`

	// Tick the types of permission. one of below types must be passed, otherwise the error is thrown
	Permissions PermissionsList `json:"permissions"`
}

// PermissionsList represents list of sub api permissions.
type PermissionsList struct {
	ContractTrade []string `json:"ContractTrade,omitempty"`
	Spot          []string `json:"Spot,omitempty"`
	Wallet        []string `json:"Wallet,omitempty"`
	Options       []string `json:"Options,omitempty"`
	Exchange      []string `json:"Exchange,omitempty"`
	CopyTrading   []string `json:"CopyTrading,omitempty"`
	BlockTrade    []string `json:"BlockTrade,omitempty"`
	NFT           []string `json:"NFT,omitempty"`
}

// AffiliateCustomerInfo represents user information
type AffiliateCustomerInfo struct {
	UID                 string       `json:"uid"`
	TakerVol30Day       types.Number `json:"takerVol30Day"`
	MakerVol30Day       types.Number `json:"makerVol30Day"`
	TradeVol30Day       types.Number `json:"tradeVol30Day"`
	DepositAmount30Day  types.Number `json:"depositAmount30Day"`
	TakerVol365Day      types.Number `json:"takerVol365Day"`
	MakerVol365Day      types.Number `json:"makerVol365Day"`
	TradeVol365Day      types.Number `json:"tradeVol365Day"`
	DepositAmount365Day types.Number `json:"depositAmount365Day"`
	TotalWalletBalance  types.Number `json:"totalWalletBalance"`
	DepositUpdateTime   time.Time    `json:"depositUpdateTime"`
	VipLevel            string       `json:"vipLevel"`
	VolUpdateTime       time.Time    `json:"volUpdateTime"`
}

// LeverageTokenInfo represents leverage token information.
type LeverageTokenInfo struct {
	FundFee          types.Number `json:"fundFee"`
	FundFeeTime      types.Time   `json:"fundFeeTime"`
	LtCoin           string       `json:"ltCoin"`
	LtName           string       `json:"ltName"`
	LtStatus         string       `json:"ltStatus"`
	ManageFeeRate    types.Number `json:"manageFeeRate"`
	ManageFeeTime    types.Time   `json:"manageFeeTime"`
	MaxPurchase      string       `json:"maxPurchase"`
	MaxPurchaseDaily string       `json:"maxPurchaseDaily"`
	MaxRedeem        string       `json:"maxRedeem"`
	MaxRedeemDaily   string       `json:"maxRedeemDaily"`
	MinPurchase      string       `json:"minPurchase"`
	MinRedeem        string       `json:"minRedeem"`
	NetValue         types.Number `json:"netValue"`
	PurchaseFeeRate  types.Number `json:"purchaseFeeRate"`
	RedeemFeeRate    types.Number `json:"redeemFeeRate"`
	Total            types.Number `json:"total"`
	Value            types.Number `json:"value"`
}

// LeveragedTokenMarket represents leverage token market details.
type LeveragedTokenMarket struct {
	Basket      types.Number `json:"basket"`
	Circulation types.Number `json:"circulation"`
	Leverage    types.Number `json:"leverage"` // Real leverage calculated by last traded price
	LTCoin      string       `json:"ltCoin"`
	NetValue    types.Number `json:"nav"`
	NavTime     types.Time   `json:"navTime"` // Update time for net asset value (in milliseconds and UTC time zone)
}

// LeverageToken represents a response instance when purchasing a leverage token.
type LeverageToken struct {
	Amount        types.Number `json:"amount"`
	ExecAmt       types.Number `json:"execAmt"`
	ExecQty       types.Number `json:"execQty"`
	LtCoin        string       `json:"ltCoin"`
	LtOrderStatus string       `json:"ltOrderStatus"`
	PurchaseID    string       `json:"purchaseId"`
	SerialNo      string       `json:"serialNo"`
	ValueCoin     string       `json:"valueCoin"`
}

// RedeemToken represents leverage redeem token
type RedeemToken struct {
	ExecAmt       types.Number `json:"execAmt"`
	ExecQty       types.Number `json:"execQty"`
	LtCoin        string       `json:"ltCoin"`
	LtOrderStatus string       `json:"ltOrderStatus"`
	Quantity      types.Number `json:"quantity"`
	RedeemID      string       `json:"redeemId"`
	SerialNo      string       `json:"serialNo"`
	ValueCoin     string       `json:"valueCoin"`
}

// RedeemPurchaseRecord represents a purchase and redeem record instance.
type RedeemPurchaseRecord struct {
	Amount        types.Number `json:"amount"`
	Fee           types.Number `json:"fee"`
	LtCoin        string       `json:"ltCoin"`
	LtOrderStatus string       `json:"ltOrderStatus"`
	LtOrderType   string       `json:"ltOrderType"`
	OrderID       string       `json:"orderId"`
	OrderTime     types.Time   `json:"orderTime"`
	SerialNo      string       `json:"serialNo"`
	UpdateTime    types.Time   `json:"updateTime"`
	Value         types.Number `json:"value"`
	ValueCoin     string       `json:"valueCoin"`
}

// SpotMarginMode represents data about whether spot margin trade is on / off
type SpotMarginMode struct {
	SpotMarginMode string `json:"spotMarginMode"`
}

// VIPMarginData represents VIP margin data.
type VIPMarginData struct {
	VipCoinList []struct {
		List []struct {
			Borrowable         bool         `json:"borrowable"`
			CollateralRatio    types.Number `json:"collateralRatio"`
			Currency           string       `json:"currency"`
			HourlyBorrowRate   types.Number `json:"hourlyBorrowRate"`
			LiquidationOrder   string       `json:"liquidationOrder"`
			MarginCollateral   bool         `json:"marginCollateral"`
			MaxBorrowingAmount types.Number `json:"maxBorrowingAmount"`
		} `json:"list"`
		VipLevel string `json:"vipLevel"`
	} `json:"vipCoinList"`
}

// MarginCoinInfo represents margin coin information.
type MarginCoinInfo struct {
	Coin             string       `json:"coin"`
	ConversionRate   types.Number `json:"conversionRate"`
	LiquidationOrder int64        `json:"liquidationOrder"`
}

// BorrowableCoinInfo represents borrowable coin information.
type BorrowableCoinInfo struct {
	Coin               string `json:"coin"`
	BorrowingPrecision int64  `json:"borrowingPrecision"`
	RepaymentPrecision int64  `json:"repaymentPrecision"`
}

// InterestAndQuota represents interest and quota information.
type InterestAndQuota struct {
	Coin           string       `json:"coin"`
	InterestRate   string       `json:"interestRate"`
	LoanAbleAmount types.Number `json:"loanAbleAmount"`
	MaxLoanAmount  types.Number `json:"maxLoanAmount"`
}

// AccountLoanInfo covers: Margin trade (Normal Account)
type AccountLoanInfo struct {
	AcctBalanceSum  types.Number `json:"acctBalanceSum"`
	DebtBalanceSum  types.Number `json:"debtBalanceSum"`
	LoanAccountList []struct {
		Interest     string       `json:"interest"`
		Loan         string       `json:"loan"`
		Locked       string       `json:"locked"`
		TokenID      string       `json:"tokenId"`
		Free         types.Number `json:"free"`
		RemainAmount types.Number `json:"remainAmount"`
		Total        types.Number `json:"total"`
	} `json:"loanAccountList"`
	RiskRate     types.Number `json:"riskRate"`
	Status       int64        `json:"status"`
	SwitchStatus int64        `json:"switchStatus"`
}

// BorrowResponse represents borrow response transaction id.
type BorrowResponse struct {
	TransactID string `json:"transactId"`
}

// LendArgument represents currency borrow and repay parameter.
type LendArgument struct {
	Coin           currency.Code `json:"coin"`
	AmountToBorrow float64       `json:"qty,string"`
}

// RepayResponse represents a repay id
type RepayResponse struct {
	RepayID string `json:"repayId"`
}

// BorrowOrderDetail represents a borrow order detail info.
type BorrowOrderDetail struct {
	ID              string       `json:"id"`
	AccountID       string       `json:"accountId"`
	Coin            string       `json:"coin"`
	CreatedTime     types.Time   `json:"createdTime"`
	InterestAmount  types.Number `json:"interestAmount"`
	InterestBalance types.Number `json:"interestBalance"`
	LoanAmount      types.Number `json:"loanAmount"`
	LoanBalance     types.Number `json:"loanBalance"`
	RemainAmount    types.Number `json:"remainAmount"`
	Status          int64        `json:"status"`
	Type            int64        `json:"type"`
}

// CoinRepaymentResponse represents a coin repayment detail.
type CoinRepaymentResponse struct {
	AccountID          string       `json:"accountId"`
	Coin               string       `json:"coin"`
	RepaidAmount       types.Number `json:"repaidAmount"`
	RepayID            string       `json:"repayId"`
	RepayMarginOrderID string       `json:"repayMarginOrderId"`
	RepayTime          types.Time   `json:"repayTime"`
	TransactIDs        []struct {
		RepaidAmount       types.Number `json:"repaidAmount"`
		RepaidInterest     types.Number `json:"repaidInterest"`
		RepaidPrincipal    types.Number `json:"repaidPrincipal"`
		RepaidSerialNumber types.Number `json:"repaidSerialNumber"`
		TransactID         string       `json:"transactId"`
	} `json:"transactIds"`
}

// InstitutionalProductInfo represents institutional product info.
type InstitutionalProductInfo struct {
	MarginProductInfo []struct {
		ProductID           string   `json:"productId"`
		Leverage            string   `json:"leverage"`
		SupportSpot         int64    `json:"supportSpot"`
		SupportContract     int64    `json:"supportContract"`
		WithdrawLine        string   `json:"withdrawLine"`
		TransferLine        string   `json:"transferLine"`
		SpotBuyLine         string   `json:"spotBuyLine"`
		SpotSellLine        string   `json:"spotSellLine"`
		ContractOpenLine    string   `json:"contractOpenLine"`
		LiquidationLine     string   `json:"liquidationLine"`
		StopLiquidationLine string   `json:"stopLiquidationLine"`
		ContractLeverage    string   `json:"contractLeverage"`
		TransferRatio       string   `json:"transferRatio"`
		SpotSymbols         []string `json:"spotSymbols"`
		ContractSymbols     []string `json:"contractSymbols"`
	} `json:"marginProductInfo"`
}

// InstitutionalMarginCoinInfo represents margin coin info for institutional lending
// token and tokens convert information.
type InstitutionalMarginCoinInfo struct {
	MarginToken []struct {
		ProductID string `json:"productId"`
		TokenInfo []struct {
			Token            string `json:"token"`
			ConvertRatioList []struct {
				Ladder       string       `json:"ladder"`
				ConvertRatio types.Number `json:"convertRatio"`
			} `json:"convertRatioList"`
		} `json:"tokenInfo"`
	} `json:"marginToken"`
}

// LoanOrderDetails retrieves institutional loan order detail item.
type LoanOrderDetails struct {
	OrderID             string       `json:"orderId"`
	OrderProductID      string       `json:"orderProductId"`
	ParentUID           string       `json:"parentUid"`
	LoanTime            types.Time   `json:"loanTime"`
	LoanCoin            string       `json:"loanCoin"`
	LoanAmount          types.Number `json:"loanAmount"`
	UnpaidAmount        types.Number `json:"unpaidAmount"`
	UnpaidInterest      types.Number `json:"unpaidInterest"`
	RepaidAmount        types.Number `json:"repaidAmount"`
	RepaidInterest      types.Number `json:"repaidInterest"`
	InterestRate        types.Number `json:"interestRate"`
	Status              int64        `json:"status"`
	Leverage            string       `json:"leverage"`
	SupportSpot         int64        `json:"supportSpot"`
	SupportContract     int64        `json:"supportContract"`
	WithdrawLine        types.Number `json:"withdrawLine"`
	TransferLine        types.Number `json:"transferLine"`
	SpotBuyLine         types.Number `json:"spotBuyLine"`
	SpotSellLine        types.Number `json:"spotSellLine"`
	ContractOpenLine    types.Number `json:"contractOpenLine"`
	LiquidationLine     types.Number `json:"liquidationLine"`
	StopLiquidationLine types.Number `json:"stopLiquidationLine"`
	ContractLeverage    types.Number `json:"contractLeverage"`
	TransferRatio       types.Number `json:"transferRatio"`
	SpotSymbols         []string     `json:"spotSymbols"`
	ContractSymbols     []string     `json:"contractSymbols"`
}

// OrderRepayInfo represents repaid information information.
type OrderRepayInfo struct {
	RepayOrderID string     `json:"repayOrderId"`
	RepaidTime   types.Time `json:"repaidTime"`
	Token        string     `json:"token"`
	Quantity     types.Time `json:"quantity"`
	Interest     types.Time `json:"interest"`
	BusinessType string     `json:"businessType"`
	Status       string     `json:"status"`
}

// LTVInfo represents institutional lending Loan-to-value(LTV)
type LTVInfo struct {
	LtvInfo []struct {
		LoanToValue    string       `json:"ltv"`
		ParentUID      string       `json:"parentUid"`
		SubAccountUids []string     `json:"subAccountUids"`
		UnpaidAmount   types.Number `json:"unpaidAmount"`
		UnpaidInfo     []struct {
			Token          string       `json:"token"`
			UnpaidQuantity types.Number `json:"unpaidQty"`
			UnpaidInterest types.Number `json:"unpaidInterest"`
		} `json:"unpaidInfo"`
		Balance     string `json:"balance"`
		BalanceInfo []struct {
			Token           string       `json:"token"`
			Price           types.Number `json:"price"`
			Qty             types.Number `json:"qty"`
			ConvertedAmount types.Number `json:"convertedAmount"`
		} `json:"balanceInfo"`
	} `json:"ltvInfo"`
}

// BindOrUnbindUIDResponse holds uid information after binding/unbinding.
type BindOrUnbindUIDResponse struct {
	UID     string `json:"uid"`
	Operate string `json:"operate"`
}

// C2CLendingCoinInfo represent contract-to-contract lending coin information.
type C2CLendingCoinInfo struct {
	Coin            string       `json:"coin"`
	LoanToPoolRatio types.Number `json:"loanToPoolRatio"`
	MaxRedeemQty    types.Number `json:"maxRedeemQty"`
	MinPurchaseQty  types.Number `json:"minPurchaseQty"`
	Precision       types.Number `json:"precision"`
	Rate            types.Number `json:"rate"`
}

// C2CLendingFundsParams represents deposit funds parameter
type C2CLendingFundsParams struct {
	Coin         currency.Code `json:"coin"`
	Quantity     float64       `json:"quantity,string"`
	SerialNumber string        `json:"serialNO"`
}

// C2CLendingFundResponse represents contract-to-contract deposit funds item.
type C2CLendingFundResponse struct {
	Coin         string       `json:"coin"`
	OrderID      string       `json:"orderId"`
	SerialNumber string       `json:"serialNo"`
	Status       string       `json:"status"`
	CreatedTime  types.Time   `json:"createdTime"`
	Quantity     types.Number `json:"quantity"`
	UpdatedTime  types.Time   `json:"updatedTime"`

	// added for redeem funds
	PrincipalQty string `json:"principalQty"`

	// added for to distinguish between redeem funds and deposit funds
	OrderType string `json:"orderType"`
}

// LendingAccountInfo represents contract-to-contract lending account info item.
type LendingAccountInfo struct {
	Coin              string       `json:"coin"`
	PrincipalInterest string       `json:"principalInterest"`
	PrincipalQty      types.Number `json:"principalQty"`
	PrincipalTotal    types.Number `json:"principalTotal"`
	Quantity          types.Number `json:"quantity"`
}

// BrokerEarningItem represents contract-to-contract broker earning item.
type BrokerEarningItem struct {
	UserID   string     `json:"userId"`
	BizType  string     `json:"bizType"`
	Symbol   string     `json:"symbol"`
	Coin     string     `json:"coin"`
	Earning  string     `json:"earning"`
	OrderID  string     `json:"orderId"`
	ExecTime types.Time `json:"execTime"`
}

// ServerTime represents server time
type ServerTime struct {
	TimeSecond types.Time `json:"timeSecond"`
	TimeNano   types.Time `json:"timeNano"`
}

// Orderbook stores the orderbook data
type Orderbook struct {
	UpdateID       int64
	Bids           []orderbook.Level
	Asks           []orderbook.Level
	Symbol         string
	GenerationTime time.Time
}

// WsOrderbookDetail represents an orderbook detail information.
type WsOrderbookDetail struct {
	Symbol   string            `json:"s"`
	Bids     [][2]types.Number `json:"b"`
	Asks     [][2]types.Number `json:"a"`
	UpdateID int64             `json:"u"`
	Sequence int64             `json:"seq"`
}

// SubscriptionResponse represents a subscription response.
type SubscriptionResponse struct {
	Success   bool   `json:"success"`
	RetMsg    string `json:"ret_msg"`
	ConnID    string `json:"conn_id"`
	RequestID string `json:"req_id"`
	Operation string `json:"op"`
}

// WebsocketResponse represents push data response struct.
type WebsocketResponse struct {
	Topic         string          `json:"topic"`
	Type          string          `json:"type"`
	PushTimestamp types.Time      `json:"ts"` // The timestamp (ms) that the system generates the data
	Data          json.RawMessage `json:"data"`
	CrossSequence int64           `json:"cs"`

	// for ping messages
	Operation string `json:"op"`

	// for subscription response checks.
	RequestID string `json:"req_id"`

	// The timestamp from the match engine when orderbook data is produced. It can be correlated with T from public trade channel
	OrderbookLastUpdated types.Time `json:"cts"`
}

// WebsocketPublicTrades represents
type WebsocketPublicTrades []struct {
	OrderFillTimestamp   types.Time   `json:"T"`
	Symbol               string       `json:"s"`
	Side                 string       `json:"S"`
	Size                 types.Number `json:"v"`
	Price                types.Number `json:"p"`
	PriceChangeDirection string       `json:"L"`
	TradeID              string       `json:"i"`
	BlockTrade           bool         `json:"BT"`
}

// WsKlines represents a list of Kline data.
type WsKlines []struct {
	Confirm   bool         `json:"confirm"`
	Start     types.Time   `json:"start"`
	End       types.Time   `json:"end"`
	Open      types.Number `json:"open"`
	Close     types.Number `json:"close"`
	High      types.Number `json:"high"`
	Low       types.Number `json:"low"`
	Volume    types.Number `json:"volume"`
	Turnover  string       `json:"turnover"`
	Interval  string       `json:"interval"`
	Timestamp types.Time   `json:"timestamp"`
}

// WebsocketLiquidation represents liquidation stream push data.
type WebsocketLiquidation struct {
	Symbol      string       `json:"symbol"`
	Side        string       `json:"side"`
	Price       types.Number `json:"price"`
	Size        types.Number `json:"size"`
	UpdatedTime types.Time   `json:"updatedTime"`
}

// LTKlines represents a leverage token kline.
type LTKlines []struct {
	Confirm   bool         `json:"confirm"`
	Interval  string       `json:"interval"`
	Start     types.Time   `json:"start"`
	End       types.Time   `json:"end"`
	Open      types.Number `json:"open"`
	Close     types.Number `json:"close"`
	High      types.Number `json:"high"`
	Low       types.Number `json:"low"`
	Timestamp types.Time   `json:"timestamp"`
}

// LTNav represents leveraged token nav stream.
type LTNav struct {
	Symbol         string       `json:"symbol"`
	Time           types.Time   `json:"time"`
	Nav            types.Number `json:"nav"`
	BasketPosition types.Number `json:"basketPosition"`
	Leverage       types.Number `json:"leverage"`
	BasketLoan     types.Number `json:"basketLoan"`
	Circulation    types.Number `json:"circulation"`
	Basket         types.Number `json:"basket"`
}

// WsPositions represents a position information.
type WsPositions []struct {
	PositionIdx      int64        `json:"positionIdx"`
	TradeMode        int64        `json:"tradeMode"`
	RiskID           int64        `json:"riskId"`
	RiskLimitValue   types.Number `json:"riskLimitValue"`
	Symbol           string       `json:"symbol"`
	Side             string       `json:"side"`
	Size             types.Number `json:"size"`
	EntryPrice       types.Number `json:"entryPrice"`
	Leverage         types.Number `json:"leverage"`
	PositionValue    types.Number `json:"positionValue"`
	PositionBalance  types.Number `json:"positionBalance"`
	MarkPrice        types.Number `json:"markPrice"`
	PositionIM       types.Number `json:"positionIM"`
	PositionMM       types.Number `json:"positionMM"`
	TakeProfit       types.Number `json:"takeProfit"`
	StopLoss         types.Number `json:"stopLoss"`
	TrailingStop     types.Number `json:"trailingStop"`
	UnrealisedPnl    types.Number `json:"unrealisedPnl"`
	CumRealisedPnl   types.Number `json:"cumRealisedPnl"`
	CreatedTime      types.Time   `json:"createdTime"`
	UpdatedTime      types.Time   `json:"updatedTime"`
	TpslMode         string       `json:"tpslMode"`
	LiqPrice         types.Number `json:"liqPrice"`
	BustPrice        types.Number `json:"bustPrice"`
	Category         string       `json:"category"`
	PositionStatus   string       `json:"positionStatus"`
	AdlRankIndicator int64        `json:"adlRankIndicator"`
}

// WsExecutions represents execution stream to see your executions in real-time.
type WsExecutions []struct {
	Category        string       `json:"category"`
	Symbol          string       `json:"symbol"`
	ExecFee         types.Number `json:"execFee"`
	ExecID          string       `json:"execId"`
	ExecPrice       types.Number `json:"execPrice"`
	ExecQty         types.Number `json:"execQty"`
	ExecType        string       `json:"execType"`
	ExecValue       types.Number `json:"execValue"`
	IsMaker         bool         `json:"isMaker"`
	FeeRate         string       `json:"feeRate"`
	TradeIv         string       `json:"tradeIv"`
	MarkIv          string       `json:"markIv"`
	BlockTradeID    string       `json:"blockTradeId"`
	MarkPrice       types.Number `json:"markPrice"`
	IndexPrice      types.Number `json:"indexPrice"`
	UnderlyingPrice types.Number `json:"underlyingPrice"`
	LeavesQty       types.Number `json:"leavesQty"`
	OrderID         string       `json:"orderId"`
	OrderLinkID     string       `json:"orderLinkId"`
	OrderPrice      types.Number `json:"orderPrice"`
	OrderQty        types.Number `json:"orderQty"`
	OrderType       string       `json:"orderType"`
	StopOrderType   string       `json:"stopOrderType"`
	Side            string       `json:"side"`
	ExecTime        types.Time   `json:"execTime"`
	IsLeverage      types.Number `json:"isLeverage"`
	ClosedSize      types.Number `json:"closedSize"`
}

// WsOrders represents private order
type WsOrders []struct {
	Symbol             string       `json:"symbol"`
	OrderID            string       `json:"orderId"`
	Side               string       `json:"side"`
	OrderType          string       `json:"orderType"`
	CancelType         string       `json:"cancelType"`
	Price              types.Number `json:"price"`
	Qty                types.Number `json:"qty"`
	OrderIv            string       `json:"orderIv"`
	TimeInForce        string       `json:"timeInForce"`
	OrderStatus        string       `json:"orderStatus"`
	OrderLinkID        string       `json:"orderLinkId"`
	LastPriceOnCreated string       `json:"lastPriceOnCreated"`
	ReduceOnly         bool         `json:"reduceOnly"`
	LeavesQty          types.Number `json:"leavesQty"`
	LeavesValue        types.Number `json:"leavesValue"`
	CumExecQty         types.Number `json:"cumExecQty"`
	CumExecValue       types.Number `json:"cumExecValue"`
	AvgPrice           types.Number `json:"avgPrice"`
	BlockTradeID       string       `json:"blockTradeId"`
	PositionIdx        int64        `json:"positionIdx"`
	CumExecFee         types.Number `json:"cumExecFee"`
	CreatedTime        types.Time   `json:"createdTime"`
	UpdatedTime        types.Time   `json:"updatedTime"`
	RejectReason       string       `json:"rejectReason"`
	StopOrderType      string       `json:"stopOrderType"`
	TpslMode           string       `json:"tpslMode"`
	TriggerPrice       types.Number `json:"triggerPrice"`
	TakeProfit         types.Number `json:"takeProfit"`
	StopLoss           types.Number `json:"stopLoss"`
	TpTriggerBy        types.Number `json:"tpTriggerBy"`
	SlTriggerBy        types.Number `json:"slTriggerBy"`
	TpLimitPrice       types.Number `json:"tpLimitPrice"`
	SlLimitPrice       types.Number `json:"slLimitPrice"`
	TriggerDirection   int64        `json:"triggerDirection"`
	TriggerBy          string       `json:"triggerBy"`
	CloseOnTrigger     bool         `json:"closeOnTrigger"`
	Category           string       `json:"category"`
	PlaceType          string       `json:"placeType"`
	SmpType            string       `json:"smpType"` // SMP execution type
	SmpGroup           int64        `json:"smpGroup"`
	SmpOrderID         string       `json:"smpOrderId"`

	// UTA Spot: add new response field ocoTriggerBy, and the value can be
	// OcoTriggerByUnknown, OcoTriggerByTp, OcoTriggerBySl
	OCOTriggerBy string `json:"ocoTriggerBy"`
}

// WebsocketWallet represents a wallet stream to see changes to your wallet in real-time.
type WebsocketWallet struct {
	ID           string     `json:"id"`
	Topic        string     `json:"topic"`
	CreationTime types.Time `json:"creationTime"`
	Data         []struct {
		AccountIMRate          types.Number `json:"accountIMRate"`
		AccountMMRate          types.Number `json:"accountMMRate"`
		TotalEquity            types.Number `json:"totalEquity"`
		TotalWalletBalance     types.Number `json:"totalWalletBalance"`
		TotalMarginBalance     types.Number `json:"totalMarginBalance"`
		TotalAvailableBalance  types.Number `json:"totalAvailableBalance"`
		TotalPerpUPL           types.Number `json:"totalPerpUPL"`
		TotalInitialMargin     types.Number `json:"totalInitialMargin"`
		TotalMaintenanceMargin types.Number `json:"totalMaintenanceMargin"`
		Coin                   []struct {
			Coin                string       `json:"coin"`
			Equity              types.Number `json:"equity"`
			UsdValue            types.Number `json:"usdValue"`
			WalletBalance       types.Number `json:"walletBalance"`
			AvailableToWithdraw types.Number `json:"availableToWithdraw"`
			AvailableToBorrow   types.Number `json:"availableToBorrow"`
			BorrowAmount        types.Number `json:"borrowAmount"`
			AccruedInterest     types.Number `json:"accruedInterest"`
			TotalOrderIM        types.Number `json:"totalOrderIM"`
			TotalPositionIM     types.Number `json:"totalPositionIM"`
			TotalPositionMM     types.Number `json:"totalPositionMM"`
			UnrealisedPnl       types.Number `json:"unrealisedPnl"`
			CumRealisedPnl      types.Number `json:"cumRealisedPnl"`
			Bonus               types.Number `json:"bonus"`
			SpotHedgingQuantity types.Number `json:"spotHedgingQty"`
		} `json:"coin"`
		AccountType string `json:"accountType"`
		AccountLTV  string `json:"accountLTV"`
	} `json:"data"`
}

// GreeksResponse represents changes to your greeks data
type GreeksResponse struct {
	ID           string     `json:"id"`
	Topic        string     `json:"topic"`
	CreationTime types.Time `json:"creationTime"`
	Data         []struct {
		BaseCoin   string       `json:"baseCoin"`
		TotalDelta types.Number `json:"totalDelta"`
		TotalGamma types.Number `json:"totalGamma"`
		TotalVega  types.Number `json:"totalVega"`
		TotalTheta types.Number `json:"totalTheta"`
	} `json:"data"`
}

// PingMessage represents a ping message.
type PingMessage struct {
	Operation string `json:"op"`
	RequestID string `json:"req_id"`
}

// InstrumentInfoItem represents an instrument long short ratio information.
type InstrumentInfoItem struct {
	Symbol    string       `json:"symbol"`
	BuyRatio  types.Number `json:"buyRatio"`
	SellRatio types.Number `json:"sellRatio"`
	Timestamp types.Time   `json:"timestamp"`
}

// Error defines all error information for each request
type Error struct {
	ReturnCode      int64  `json:"ret_code"`
	ReturnMsg       string `json:"ret_msg"`
	ReturnCodeV5    int64  `json:"retCode"`
	ReturnMessageV5 string `json:"retMsg"`
	ExtCode         string `json:"ext_code"`
	ExtMsg          string `json:"ext_info"`
}

// accountTypeHolder holds the account type associated with the loaded API key.
type accountTypeHolder struct {
	accountType AccountType
	m           sync.Mutex
}

// AccountType constants
type AccountType uint8

// String returns the account type as a string
func (a AccountType) String() string {
	switch a {
	case 0:
		return "unset"
	case accountTypeNormal:
		return "normal"
	case accountTypeUnified:
		return "unified"
	default:
		return "unknown"
	}
}
