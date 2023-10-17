package bybit

import (
	"encoding/json"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
)

var validCategory = []string{"spot", "linear", "inverse", "option"}

type orderbookResponse struct {
	Symbol    string               `json:"s"`
	Asks      [][2]string          `json:"a"`
	Bids      [][2]string          `json:"b"`
	Timestamp convert.ExchangeTime `json:"ts"`
	UpdateID  int64                `json:"u"`
}

// Authenticate stores authentication variables required
type Authenticate struct {
	RequestID string        `json:"req_id"`
	Args      []interface{} `json:"args"`
	Operation string        `json:"op"`
}

// SubscriptionArgument represents a subscription arguments.
type SubscriptionArgument struct {
	auth      bool     `json:"-"`
	RequestID string   `json:"req_id"`
	Operation string   `json:"op"`
	Arguments []string `json:"args"`
}

// Fee holds fee information
type Fee struct {
	BaseCoin string                  `json:"baseCoin"`
	Symbol   string                  `json:"symbol"`
	Taker    convert.StringToFloat64 `json:"takerFeeRate"`
	Maker    convert.StringToFloat64 `json:"makerFeeRate"`
}

// AccountFee holds account fee information
type AccountFee struct {
	Category string `json:"category"`
	List     []Fee  `json:"list"`
}

// InstrumentsInfo representa a category, page indicator, and list of instrument information.
type InstrumentsInfo struct {
	Category       string           `json:"category"`
	List           []InstrumentInfo `json:"list"`
	NextPageCursor string           `json:"nextPageCursor"`
}

// InstrumentInfo holds all instrument info across
// spot, linear, option types
type InstrumentInfo struct {
	Symbol          string                  `json:"symbol"`
	ContractType    string                  `json:"contractType"`
	Innovation      string                  `json:"innovation"`
	MarginTrading   string                  `json:"marginTrading"`
	OptionsType     string                  `json:"optionsType"`
	Status          string                  `json:"status"`
	BaseCoin        string                  `json:"baseCoin"`
	QuoteCoin       string                  `json:"quoteCoin"`
	LaunchTime      convert.ExchangeTime    `json:"launchTime"`
	DeliveryTime    convert.ExchangeTime    `json:"deliveryTime"`
	DeliveryFeeRate convert.StringToFloat64 `json:"deliveryFeeRate"`
	PriceScale      convert.StringToFloat64 `json:"priceScale"`
	LeverageFilter  struct {
		MinLeverage  convert.StringToFloat64 `json:"minLeverage"`
		MaxLeverage  convert.StringToFloat64 `json:"maxLeverage"`
		LeverageStep convert.StringToFloat64 `json:"leverageStep"`
	} `json:"leverageFilter"`
	PriceFilter struct {
		MinPrice convert.StringToFloat64 `json:"minPrice"`
		MaxPrice convert.StringToFloat64 `json:"maxPrice"`
		TickSize convert.StringToFloat64 `json:"tickSize"`
	} `json:"priceFilter"`
	LotSizeFilter struct {
		MaxOrderQty         convert.StringToFloat64 `json:"maxOrderQty"`
		MinOrderQty         convert.StringToFloat64 `json:"minOrderQty"`
		QtyStep             convert.StringToFloat64 `json:"qtyStep"`
		PostOnlyMaxOrderQty convert.StringToFloat64 `json:"postOnlyMaxOrderQty"`
		BasePrecision       convert.StringToFloat64 `json:"basePrecision"`
		QuotePrecision      convert.StringToFloat64 `json:"quotePrecision"`
		MinOrderAmt         convert.StringToFloat64 `json:"minOrderAmt"`
		MaxOrderAmt         convert.StringToFloat64 `json:"maxOrderAmt"`
	} `json:"lotSizeFilter"`
	UnifiedMarginTrade bool   `json:"unifiedMarginTrade"`
	FundingInterval    int64  `json:"fundingInterval"`
	SettleCoin         string `json:"settleCoin"`
}

// RestResponse represents a REST response instance.
type RestResponse struct {
	RetCode    int64       `json:"retCode"`
	RetMsg     string      `json:"retMsg"`
	Result     interface{} `json:"result"`
	RetExtInfo struct {
		List []ErrorMessage `json:"list"`
	} `json:"retExtInfo"`
	Time convert.ExchangeTime `json:"time"`
}

// KlineResponse represents a kline item list instance as an array of string.
type KlineResponse struct {
	Symbol   string     `json:"symbol"`
	Category string     `json:"category"`
	List     [][]string `json:"list"`
}

// KlineItem stores an individual kline data item
type KlineItem struct {
	StartTime time.Time
	Open      float64
	High      float64
	Low       float64
	Close     float64

	// not available for mark and index price kline data
	TradeVolume float64
	Turnover    float64
}

// MarkPriceKlineResponse represents a kline data item.
type MarkPriceKlineResponse struct {
	Symbol   string     `json:"symbol"`
	Category string     `json:"category"`
	List     [][]string `json:"list"`
}

func constructOrderbook(o *orderbookResponse) (*Orderbook, error) {
	var (
		s = Orderbook{
			Symbol:         o.Symbol,
			UpdateID:       o.UpdateID,
			GenerationTime: o.Timestamp.Time(),
		}
		err error
	)
	s.Bids, err = processOB(o.Bids)
	if err != nil {
		return nil, err
	}
	s.Asks, err = processOB(o.Asks)
	if err != nil {
		return nil, err
	}
	return &s, err
}

// TickerData represents a list of ticker detailed information.
type TickerData struct {
	Category string       `json:"category"`
	List     []TickerItem `json:"list"`
}

// TickerItem represents a ticker item detail.
type TickerItem struct {
	Symbol                 string                  `json:"symbol"`
	LastPrice              convert.StringToFloat64 `json:"lastPrice"`
	IndexPrice             convert.StringToFloat64 `json:"indexPrice"`
	MarkPrice              convert.StringToFloat64 `json:"markPrice"`
	PrevPrice24H           convert.StringToFloat64 `json:"prevPrice24h"`
	Price24HPcnt           convert.StringToFloat64 `json:"price24hPcnt"`
	HighPrice24H           convert.StringToFloat64 `json:"highPrice24h"`
	LowPrice24H            convert.StringToFloat64 `json:"lowPrice24h"`
	PrevPrice1H            convert.StringToFloat64 `json:"prevPrice1h"`
	OpenInterest           convert.StringToFloat64 `json:"openInterest"`
	OpenInterestValue      convert.StringToFloat64 `json:"openInterestValue"`
	Turnover24H            convert.StringToFloat64 `json:"turnover24h"`
	Volume24H              convert.StringToFloat64 `json:"volume24h"`
	FundingRate            convert.StringToFloat64 `json:"fundingRate"`
	NextFundingTime        convert.StringToFloat64 `json:"nextFundingTime"`
	PredictedDeliveryPrice convert.StringToFloat64 `json:"predictedDeliveryPrice"`
	BasisRate              convert.StringToFloat64 `json:"basisRate"`
	DeliveryFeeRate        convert.StringToFloat64 `json:"deliveryFeeRate"`
	DeliveryTime           convert.ExchangeTime    `json:"deliveryTime"`
	Ask1Size               convert.StringToFloat64 `json:"ask1Size"`
	Bid1Price              convert.StringToFloat64 `json:"bid1Price"`
	Ask1Price              convert.StringToFloat64 `json:"ask1Price"`
	Bid1Size               convert.StringToFloat64 `json:"bid1Size"`
	Basis                  convert.StringToFloat64 `json:"basis"`
}

// FundingRateHistory represents a funding rate history for a category.
type FundingRateHistory struct {
	Category string        `json:"category"`
	List     []FundingRate `json:"list"`
}

// FundingRate represents a funding rate instance.
type FundingRate struct {
	Symbol               string                  `json:"symbol"`
	FundingRate          convert.StringToFloat64 `json:"fundingRate"`
	FundingRateTimestamp convert.ExchangeTime    `json:"fundingRateTimestamp"`
}

// TradingHistory represents a trading history list.
type TradingHistory struct {
	Category string               `json:"category"`
	List     []TradingHistoryItem `json:"list"`
}

// TradingHistoryItem represents a trading history item instance.
type TradingHistoryItem struct {
	ExecutionID  string                  `json:"execId"`
	Symbol       string                  `json:"symbol"`
	Price        convert.StringToFloat64 `json:"price"`
	Size         convert.StringToFloat64 `json:"size"`
	Side         string                  `json:"side"`
	TradeTime    convert.ExchangeTime    `json:"time"`
	IsBlockTrade bool                    `json:"isBlockTrade"`
}

// OpenInterest represents open interest of each symbol.
type OpenInterest struct {
	Symbol   string `json:"symbol"`
	Category string `json:"category"`
	List     []struct {
		OpenInterest convert.StringToFloat64 `json:"openInterest"`
		Timestamp    convert.ExchangeTime    `json:"timestamp"`
	} `json:"list"`
	NextPageCursor string `json:"nextPageCursor"`
}

// HistoricVolatility represents option historical volatility
type HistoricVolatility struct {
	Period int64                   `json:"period"`
	Value  convert.StringToFloat64 `json:"value"`
	Time   convert.ExchangeTime    `json:"time"`
}

// InsuranceHistory represents an insurance list.
type InsuranceHistory struct {
	UpdatedTime convert.ExchangeTime `json:"updatedTime"`
	List        []struct {
		Coin    string                  `json:"coin"`
		Balance convert.StringToFloat64 `json:"balance"`
		Value   convert.StringToFloat64 `json:"value"`
	} `json:"list"`
}

// RiskLimitHistory represents risk limit history of a category.
type RiskLimitHistory struct {
	Category string `json:"category"`
	List     []struct {
		ID                int64                   `json:"id"`
		Symbol            string                  `json:"symbol"`
		RiskLimitValue    convert.StringToFloat64 `json:"riskLimitValue"`
		MaintenanceMargin convert.StringToFloat64 `json:"maintenanceMargin"`
		InitialMargin     convert.StringToFloat64 `json:"initialMargin"`
		IsLowestRisk      int64                   `json:"isLowestRisk"`
		MaxLeverage       convert.StringToFloat64 `json:"maxLeverage"`
	} `json:"list"`
}

// DeliveryPrice represents the delivery price information.
type DeliveryPrice struct {
	Category       string `json:"category"`
	NextPageCursor string `json:"nextPageCursor"`
	List           []struct {
		Symbol        string                  `json:"symbol"`
		DeliveryPrice convert.StringToFloat64 `json:"deliveryPrice"`
		DeliveryTime  convert.ExchangeTime    `json:"deliveryTime"`
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
	ReduceOnly             bool          `json:"reduceOnly,omitempty"`
	TakeProfitPrice        float64       `json:"takeProfit,omitempty,string"`
	TakeProfitTriggerBy    string        `json:"tpTriggerBy,omitempty"` // The price type to trigger take profit. 'MarkPrice', 'IndexPrice', default: 'LastPrice'
	StopLossTriggerBy      string        `json:"slTriggerBy,omitempty"` // The price type to trigger stop loss. MarkPrice, IndexPrice, default: LastPrice
	StopLossPrice          float64       `json:"stopLoss,omitempty,string"`
	CloseOnTrigger         bool          `json:"closeOnTrigger,omitempty"`
	SMPExecutionType       string        `json:"smpType,omitempty"` // default: 'None', 'CancelMaker', 'CancelTaker', 'CancelBoth'
	MarketMakerProtection  bool          `json:"mmp,omitempty"`     // option only. true means set the order as a market maker protection order.

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
	Category               string        `json:"category"`
	Symbol                 currency.Pair `json:"symbol"`
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
}

// CancelOrderParams represents a cancel order parameters.
type CancelOrderParams struct {
	Category    string        `json:"category,omitempty"`
	Symbol      currency.Pair `json:"symbol,omitempty"`
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
	OrderID                string                  `json:"orderId"`
	OrderLinkID            string                  `json:"orderLinkId"`
	BlockTradeID           string                  `json:"blockTradeId"`
	Symbol                 string                  `json:"symbol"`
	Price                  convert.StringToFloat64 `json:"price"`
	OrderQuantity          convert.StringToFloat64 `json:"qty"`
	Side                   string                  `json:"side"`
	IsLeverage             string                  `json:"isLeverage"`
	PositionIdx            int                     `json:"positionIdx"`
	OrderStatus            string                  `json:"orderStatus"`
	CancelType             string                  `json:"cancelType"`
	RejectReason           string                  `json:"rejectReason"`
	AveragePrice           convert.StringToFloat64 `json:"avgPrice"`
	LeavesQuantity         convert.StringToFloat64 `json:"leavesQty"`
	LeavesValue            string                  `json:"leavesValue"`
	CumulativeExecQuantity convert.StringToFloat64 `json:"cumExecQty"`
	CumulativeExecValue    convert.StringToFloat64 `json:"cumExecValue"`
	CumulativeExecFee      convert.StringToFloat64 `json:"cumExecFee"`
	TimeInForce            string                  `json:"timeInForce"`
	OrderType              string                  `json:"orderType"`
	StopOrderType          string                  `json:"stopOrderType"`
	OrderIv                string                  `json:"orderIv"`
	TriggerPrice           convert.StringToFloat64 `json:"triggerPrice"`
	TakeProfitPrice        convert.StringToFloat64 `json:"takeProfit"`
	StopLossPrice          convert.StringToFloat64 `json:"stopLoss"`
	TpTriggerBy            string                  `json:"tpTriggerBy"`
	SlTriggerBy            string                  `json:"slTriggerBy"`
	TriggerDirection       int                     `json:"triggerDirection"`
	TriggerBy              string                  `json:"triggerBy"`
	LastPriceOnCreated     string                  `json:"lastPriceOnCreated"`
	ReduceOnly             bool                    `json:"reduceOnly"`
	CloseOnTrigger         bool                    `json:"closeOnTrigger"`
	SmpType                string                  `json:"smpType"`
	SmpGroup               int                     `json:"smpGroup"`
	SmpOrderID             string                  `json:"smpOrderId"`
	TpslMode               string                  `json:"tpslMode"`
	TpLimitPrice           convert.StringToFloat64 `json:"tpLimitPrice"`
	SlLimitPrice           convert.StringToFloat64 `json:"slLimitPrice"`
	PlaceType              string                  `json:"placeType"`
	CreatedTime            convert.ExchangeTime    `json:"createdTime"`
	UpdatedTime            convert.ExchangeTime    `json:"updatedTime"`
}

// CancelAllResponse represents a cancel all trade orders response.
type CancelAllResponse struct {
	List []OrderResponse `json:"list"`
}

// CancelAllOrdersParam request parameters for cancel all orders.
type CancelAllOrdersParam struct {
	Category    string        `json:"category"`
	Symbol      currency.Pair `json:"symbol"`
	OrderFilter string        `json:"orderFilter,omitempty"` // Valid for spot only. Order,tpslOrder. If not passed, Order by default
	BaseCoin    string        `json:"baseCoin,omitempty"`
	SettleCoin  string        `json:"settleCoin,omitempty"`
}

// PlaceBatchOrderParam represents a parameter for placing batch orders
type PlaceBatchOrderParam struct {
	Category string                `json:"category"`
	Request  []BatchOrderItemParam `json:"request"`
}

// BatchOrderItemParam represents a batch order place parameter.
type BatchOrderItemParam struct {
	Category         string        `json:"category,omitempty"`
	Symbol           currency.Pair `json:"symbol,omitempty"`
	OrderType        string        `json:"orderType,omitempty"`
	Side             string        `json:"side,omitempty"`
	OrderQuantity    float64       `json:"qty,string,omitempty"`
	Price            float64       `json:"price,string,omitempty"`
	TriggerDirection int64         `json:"triggerDirection,omitempty"`
	TriggerPrice     int64         `json:"triggerPrice,omitempty"`
	TriggerBy        string        `json:"triggerBy,omitempty"` // Possible values:  LastPrice, IndexPrice, and MarkPrice
	OrderIv          int64         `json:"orderIv,omitempty,string"`
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
	ReduceOnly            bool   `json:"reduceOnly,omitempty"`
	CloseOnTrigger        bool   `json:"closeOnTrigger,omitempty"`
	SMPType               string `json:"smpType,omitempty"`
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
	Category    string               `json:"category"`
	Symbol      string               `json:"symbol"`
	OrderID     string               `json:"orderId"`
	OrderLinkID string               `json:"orderLinkId"`
	CreateAt    convert.ExchangeTime `json:"createAt"`
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
	Symbol         string `json:"symbol"`
	MaxTradeQty    string `json:"maxTradeQty"`
	Side           string `json:"side"`
	MaxTradeAmount string `json:"maxTradeAmount"`
	BorrowCoin     string `json:"borrowCoin"`
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
	PositionIndex    int64                   `json:"positionIdx"`
	RiskID           int64                   `json:"riskId"`
	RiskLimitValue   string                  `json:"riskLimitValue"`
	Symbol           string                  `json:"symbol"`
	Side             string                  `json:"side"`
	Size             convert.StringToFloat64 `json:"size"`
	AveragePrice     convert.StringToFloat64 `json:"avgPrice"`
	PositionValue    convert.StringToFloat64 `json:"positionValue"`
	TradeMode        int64                   `json:"tradeMode"`
	PositionStatus   string                  `json:"positionStatus"`
	AutoAddMargin    int64                   `json:"autoAddMargin"`
	AdlRankIndicator int64                   `json:"adlRankIndicator"`
	Leverage         convert.StringToFloat64 `json:"leverage"`
	PositionBalance  convert.StringToFloat64 `json:"positionBalance"`
	MarkPrice        convert.StringToFloat64 `json:"markPrice"`
	LiqPrice         convert.StringToFloat64 `json:"liqPrice"`
	BustPrice        convert.StringToFloat64 `json:"bustPrice"`
	PositionMM       convert.StringToFloat64 `json:"positionMM"`
	PositionIM       convert.StringToFloat64 `json:"positionIM"`
	TpslMode         string                  `json:"tpslMode"`
	TakeProfit       convert.StringToFloat64 `json:"takeProfit"`
	StopLoss         convert.StringToFloat64 `json:"stopLoss"`
	TrailingStop     convert.StringToFloat64 `json:"trailingStop"`
	UnrealisedPnl    convert.StringToFloat64 `json:"unrealisedPnl"`
	CumRealisedPnl   convert.StringToFloat64 `json:"cumRealisedPnl"`
	CreatedTime      convert.ExchangeTime    `json:"createdTime"`
	UpdatedTime      convert.ExchangeTime    `json:"updatedTime"`
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
	Symbol                   currency.Pair `json:"symbol"` // Symbol name. Either symbol or coin is required. symbol has a higher priority
	TakeProfit               string        `json:"takeProfit,omitempty"`
	StopLoss                 string        `json:"stopLoss,omitempty"`
	TrailingStop             string        `json:"trailingStop,omitempty"`
	TakeProfitTriggerType    string        `json:"tpTriggerBy,omitempty"`
	StopLossTriggerType      string        `json:"slTriggerBy,omitempty"`
	ActivePrice              string        `json:"activePrice,omitempty"`
	TakeProfitOrStopLossMode string        `json:"tpslMode,omitempty"`
	TakeProfitOrderType      string        `json:"tpOrderType,omitempty"`
	StopLossOrderType        string        `json:"slOrderType,omitempty"`
	TakeProfitSize           float64       `json:"tpSize,string,omitempty"`
	StopLossSize             float64       `json:"slSize,string,omitempty"`
	TakeProfitLimitPrice     float64       `json:"tpLimitPrice,string,omitempty"`
	StopLossLimitPrice       float64       `json:"slLimitPrice,string,omitempty"`
	PositionIndex            int64         `json:"positionIdx,omitempty"`
}

// AddRemoveMarginParams represents parameters for auto add margin
type AddRemoveMarginParams struct {
	Category      string        `json:"category,omitempty"`
	Symbol        currency.Pair `json:"symbol,omitempty"`
	AutoAddmargin int64         `json:"autoAddmargin,string,omitempty"`
	PositionMode  int64         `json:"positionIdx,string,omitempty"`
}

// AddOrReduceMargin represents a add or reduce margin response.
type AddOrReduceMargin struct {
	Category                 string                  `json:"category"`
	Symbol                   string                  `json:"symbol"`
	PositionIndex            int64                   `json:"positionIdx"` // position mode index
	RiskID                   int64                   `json:"riskId"`
	RiskLimitValue           string                  `json:"riskLimitValue"`
	Size                     convert.StringToFloat64 `json:"size"`
	PositionValue            string                  `json:"positionValue"`
	AveragePrice             convert.StringToFloat64 `json:"avgPrice"`
	LiquidationPrice         convert.StringToFloat64 `json:"liqPrice"`
	BustPrice                convert.StringToFloat64 `json:"bustPrice"`
	MarkPrice                convert.StringToFloat64 `json:"markPrice"`
	Leverage                 string                  `json:"leverage"`
	AutoAddMargin            int64                   `json:"autoAddMargin"`
	PositionStatus           string                  `json:"positionStatus"`
	PositionIM               convert.StringToFloat64 `json:"positionIM"`
	PositionMM               convert.StringToFloat64 `json:"positionMM"`
	UnrealisedProfitAndLoss  convert.StringToFloat64 `json:"unrealisedPnl"`
	CumRealisedProfitAndLoss convert.StringToFloat64 `json:"cumRealisedPnl"`
	StopLoss                 convert.StringToFloat64 `json:"stopLoss"`
	TakeProfit               convert.StringToFloat64 `json:"takeProfit"`
	TrailingStop             convert.StringToFloat64 `json:"trailingStop"`
	CreatedTime              convert.ExchangeTime    `json:"createdTime"`
	UpdatedTime              convert.ExchangeTime    `json:"updatedTime"`
}

// ExecutionResponse represents users order execution response
type ExecutionResponse struct {
	NextPageCursor string      `json:"nextPageCursor"`
	Category       string      `json:"category"`
	List           []Execution `json:"list"`
}

// Execution represents execution record
type Execution struct {
	Symbol                 string                  `json:"symbol"`
	OrderType              string                  `json:"orderType"`
	UnderlyingPrice        convert.StringToFloat64 `json:"underlyingPrice"`
	IndexPrice             convert.StringToFloat64 `json:"indexPrice"`
	OrderLinkID            string                  `json:"orderLinkId"`
	Side                   string                  `json:"side"`
	OrderID                string                  `json:"orderId"`
	StopOrderType          string                  `json:"stopOrderType"`
	LeavesQuantity         string                  `json:"leavesQty"`
	ExecTime               string                  `json:"execTime"`
	IsMaker                bool                    `json:"isMaker"`
	ExecFee                convert.StringToFloat64 `json:"execFee"`
	FeeRate                convert.StringToFloat64 `json:"feeRate"`
	ExecID                 string                  `json:"execId"`
	TradeImpliedVolatility string                  `json:"tradeIv"`
	BlockTradeID           string                  `json:"blockTradeId"`
	MarkPrice              convert.StringToFloat64 `json:"markPrice"`
	ExecPrice              convert.StringToFloat64 `json:"execPrice"`
	MarkIv                 string                  `json:"markIv"`
	OrderQuantity          string                  `json:"orderQty"`
	ExecValue              string                  `json:"execValue"`
	ExecType               string                  `json:"execType"`
	OrderPrice             convert.StringToFloat64 `json:"orderPrice"`
	ExecQuantity           convert.StringToFloat64 `json:"execQty"`
	ClosedSize             convert.StringToFloat64 `json:"closedSize"`
}

// ClosedProfitAndLossResponse represents list of closed profit and loss records
type ClosedProfitAndLossResponse struct {
	NextPageCursor string `json:"nextPageCursor"`
	Category       string `json:"category"`
	List           []struct {
		Symbol              string                  `json:"symbol"`
		OrderType           string                  `json:"orderType"`
		Leverage            string                  `json:"leverage"`
		UpdatedTime         convert.ExchangeTime    `json:"updatedTime"`
		Side                string                  `json:"side"`
		OrderID             string                  `json:"orderId"`
		ClosedPnl           string                  `json:"closedPnl"`
		AvgEntryPrice       convert.StringToFloat64 `json:"avgEntryPrice"`
		Quantity            convert.StringToFloat64 `json:"qty"`
		CumulatedEntryValue string                  `json:"cumEntryValue"`
		CreatedTime         convert.ExchangeTime    `json:"createdTime"`
		OrderPrice          convert.StringToFloat64 `json:"orderPrice"`
		ClosedSize          convert.StringToFloat64 `json:"closedSize"`
		AvgExitPrice        convert.StringToFloat64 `json:"avgExitPrice"`
		ExecutionType       string                  `json:"execType"`
		FillCount           convert.StringToFloat64 `json:"fillCount"`
		CumulatedExitValue  string                  `json:"cumExitValue"`
	} `json:"list"`
}

type paramsConfig struct {
	Spot            bool
	Option          bool
	Linear          bool
	Inverse         bool
	MendatorySymbol bool

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
	Symbol          string                  `json:"symbol"`
	Side            string                  `json:"side"`
	Funding         string                  `json:"funding"`
	OrderLinkID     string                  `json:"orderLinkId"`
	OrderID         string                  `json:"orderId"`
	Fee             convert.StringToFloat64 `json:"fee"`
	Change          string                  `json:"change"`
	CashFlow        string                  `json:"cashFlow"`
	TransactionTime convert.ExchangeTime    `json:"transactionTime"`
	Type            string                  `json:"type"`
	FeeRate         convert.StringToFloat64 `json:"feeRate"`
	BonusChange     convert.StringToFloat64 `json:"bonusChange"`
	Size            convert.StringToFloat64 `json:"size"`
	Qty             convert.StringToFloat64 `json:"qty"`
	CashBalance     convert.StringToFloat64 `json:"cashBalance"`
	Currency        string                  `json:"currency"`
	Category        string                  `json:"category"`
	TradePrice      convert.StringToFloat64 `json:"tradePrice"`
	TradeID         string                  `json:"tradeId"`
}

// PreUpdateOptionDeliveryRecord represents delivery records of Option
type PreUpdateOptionDeliveryRecord struct {
	NextPageCursor string `json:"nextPageCursor"`
	Category       string `json:"category"`
	List           []struct {
		Symbol        string                  `json:"symbol"`
		Side          string                  `json:"side"`
		DeliveryTime  convert.ExchangeTime    `json:"deliveryTime"`
		ExercisePrice convert.StringToFloat64 `json:"strike"`
		Fee           convert.StringToFloat64 `json:"fee"`
		Position      string                  `json:"position"`
		DeliveryPrice convert.StringToFloat64 `json:"deliveryPrice"`
		DeliveryRpl   string                  `json:"deliveryRpl"` // Realized PnL of the delivery
	} `json:"list"`
}

// SettlementSession represents a USDC settlement session.
type SettlementSession struct {
	NextPageCursor string `json:"nextPageCursor"`
	Category       string `json:"category"`
	List           []struct {
		RealisedProfitAndLoss string                  `json:"realisedPnl"`
		Symbol                string                  `json:"symbol"`
		Side                  string                  `json:"side"`
		MarkPrice             convert.StringToFloat64 `json:"markPrice"`
		Size                  convert.StringToFloat64 `json:"size"`
		CreatedTime           convert.ExchangeTime    `json:"createdTime"`
		SessionAveragePrice   convert.StringToFloat64 `json:"sessionAvgPrice"`
	} `json:"list"`
}

// WalletBalance represents wallet balance
type WalletBalance struct {
	List []struct {
		TotalEquity            convert.StringToFloat64 `json:"totalEquity"`
		AccountIMRate          convert.StringToFloat64 `json:"accountIMRate"`
		TotalMarginBalance     convert.StringToFloat64 `json:"totalMarginBalance"`
		TotalInitialMargin     convert.StringToFloat64 `json:"totalInitialMargin"`
		AccountType            string                  `json:"accountType"`
		TotalAvailableBalance  convert.StringToFloat64 `json:"totalAvailableBalance"`
		AccountMMRate          convert.StringToFloat64 `json:"accountMMRate"`
		TotalPerpUPL           string                  `json:"totalPerpUPL"`
		TotalWalletBalance     convert.StringToFloat64 `json:"totalWalletBalance"`
		AccountLTV             string                  `json:"accountLTV"` // Account LTV: account total borrowed size / (account total equity + account total borrowed size).
		TotalMaintenanceMargin convert.StringToFloat64 `json:"totalMaintenanceMargin"`
		Coin                   []struct {
			AvailableToBorrow       convert.StringToFloat64 `json:"availableToBorrow"`
			Bonus                   convert.StringToFloat64 `json:"bonus"`
			AccruedInterest         string                  `json:"accruedInterest"`
			AvailableToWithdraw     convert.StringToFloat64 `json:"availableToWithdraw"`
			AvailableBalanceForSpot convert.StringToFloat64 `json:"free"`
			TotalOrderIM            string                  `json:"totalOrderIM"`
			Equity                  convert.StringToFloat64 `json:"equity"`
			TotalPositionMM         string                  `json:"totalPositionMM"`
			USDValue                convert.StringToFloat64 `json:"usdValue"`
			UnrealisedPnl           convert.StringToFloat64 `json:"unrealisedPnl"`
			BorrowAmount            convert.StringToFloat64 `json:"borrowAmount"`
			TotalPositionIM         string                  `json:"totalPositionIM"`
			WalletBalance           convert.StringToFloat64 `json:"walletBalance"`
			CummulativeRealisedPnl  convert.StringToFloat64 `json:"cumRealisedPnl"`
			Coin                    string                  `json:"coin"`
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
		CreatedTime               convert.ExchangeTime    `json:"createdTime"`
		CostExemption             string                  `json:"costExemption"`
		InterestBearingBorrowSize string                  `json:"InterestBearingBorrowSize"`
		Currency                  string                  `json:"currency"`
		HourlyBorrowRate          convert.StringToFloat64 `json:"hourlyBorrowRate"`
		BorrowCost                convert.StringToFloat64 `json:"borrowCost"`
	} `json:"list"`
}

// CollateralInfo represents collateral information of the current unified margin account.
type CollateralInfo struct {
	List []struct {
		BorrowAmount        convert.StringToFloat64 `json:"borrowAmount"`
		AvailableToBorrow   string                  `json:"availableToBorrow"`
		FreeBorrowingAmount convert.StringToFloat64 `json:"freeBorrowingAmount"`
		Borrowable          bool                    `json:"borrowable"`
		Currency            string                  `json:"currency"`
		MaxBorrowingAmount  convert.StringToFloat64 `json:"maxBorrowingAmount"`
		HourlyBorrowRate    convert.StringToFloat64 `json:"hourlyBorrowRate"`
		MarginCollateral    bool                    `json:"marginCollateral"`
		CollateralRatio     convert.StringToFloat64 `json:"collateralRatio"`
	} `json:"list"`
}

// CoinGreeks represents current account greeks information.
type CoinGreeks struct {
	List []struct {
		BaseCoin   string                  `json:"baseCoin"`
		TotalDelta convert.StringToFloat64 `json:"totalDelta"`
		TotalGamma convert.StringToFloat64 `json:"totalGamma"`
		TotalVega  convert.StringToFloat64 `json:"totalVega"`
		TotalTheta convert.StringToFloat64 `json:"totalTheta"`
	} `json:"list"`
}

// FeeRate represents maker and taker fee rate information for a symbol.
type FeeRate struct {
	Symbol       string                  `json:"symbol"`
	TakerFeeRate convert.StringToFloat64 `json:"takerFeeRate"`
	MakerFeeRate convert.StringToFloat64 `json:"makerFeeRate"`
}

// AccountInfo represents margin mode account information.
type AccountInfo struct {
	MarginMode          string               `json:"marginMode"`
	UpdatedTime         convert.ExchangeTime `json:"updatedTime"`
	UnifiedMarginStatus int64                `json:"unifiedMarginStatus"`
	DcpStatus           string               `json:"dcpStatus"`
	TimeWindow          int64                `json:"timeWindow"`
	SmpGroup            int64                `json:"smpGroup"`
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
	BaseCoin           string                  `json:"baseCoin"`
	TimeWindowMS       int64                   `json:"window,string"`
	FrozenPeriod       int64                   `json:"frozenPeriod,string"`
	TradeQuantityLimit convert.StringToFloat64 `json:"qtyLimit"`
	DeltaLimit         convert.StringToFloat64 `json:"deltaLimit"`
}

// MMPStates represents an MMP states.
type MMPStates struct {
	Result []struct {
		BaseCoin           string                  `json:"baseCoin"`
		MmpEnabled         bool                    `json:"mmpEnabled"`
		Window             string                  `json:"window"`
		FrozenPeriod       string                  `json:"frozenPeriod"`
		TradeQuantityLimit convert.StringToFloat64 `json:"qtyLimit"`
		DeltaLimit         convert.StringToFloat64 `json:"deltaLimit"`
		MmpFrozenUntil     convert.StringToFloat64 `json:"mmpFrozenUntil"`
		MmpFrozen          bool                    `json:"mmpFrozen"`
	} `json:"result"`
}

// CoinExchangeRecords represents a coin exchange records.
type CoinExchangeRecords struct {
	OrderBody []struct {
		FromCoin              string                  `json:"fromCoin"`
		FromAmount            convert.StringToFloat64 `json:"fromAmount"`
		ToCoin                string                  `json:"toCoin"`
		ToAmount              convert.StringToFloat64 `json:"toAmount"`
		ExchangeRate          convert.StringToFloat64 `json:"exchangeRate"`
		CreatedTime           convert.ExchangeTime    `json:"createdTime"`
		ExchangeTransactionID string                  `json:"exchangeTxId"`
	} `json:"orderBody"`
	NextPageCursor string `json:"nextPageCursor"`
}

// DeliveryRecord represents delivery records of USDC futures and Options.
type DeliveryRecord struct {
	NextPageCursor string `json:"nextPageCursor"`
	Category       string `json:"category"`
	List           []struct {
		Symbol                        string                  `json:"symbol"`
		Side                          string                  `json:"side"`
		DeliveryTime                  convert.ExchangeTime    `json:"deliveryTime"`
		ExercisePrice                 convert.StringToFloat64 `json:"strike"`
		Fee                           convert.StringToFloat64 `json:"fee"`
		Position                      convert.StringToFloat64 `json:"position"`
		DeliveryPrice                 convert.StringToFloat64 `json:"deliveryPrice"`
		DeliveryRealizedProfitAndLoss convert.StringToFloat64 `json:"deliveryRpl"`
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
	BizType     int    `json:"bizType"`
	AccountID   string `json:"accountId"`
	MemberID    string `json:"memberId"`
	Balance     struct {
		Coin               string                  `json:"coin"`
		WalletBalance      convert.StringToFloat64 `json:"walletBalance"`
		TransferBalance    convert.StringToFloat64 `json:"transferBalance"`
		Bonus              convert.StringToFloat64 `json:"bonus"`
		TransferSafeAmount string                  `json:"transferSafeAmount"`
	} `json:"balance"`
}

// TransferableCoins represents list of transferable coins.
type TransferableCoins struct {
	List []string `json:"list"`
}

// TransferParams represents parameters from internal coin transfer.
type TransferParams struct {
	TransferID      uuid.UUID               `json:"transferId"`
	Coin            currency.Code           `json:"coin"`
	Amount          convert.StringToFloat64 `json:"amount,string"`
	FromAccountType string                  `json:"fromAccountType"`
	ToAccountType   string                  `json:"toAccountType"`

	// Added for universal transfers
	FromMemberID int64 `json:"fromMemberId"`
	ToMemberID   int64 `json:"toMemberId"`
}

// TransferResponse represents a transfer response
type TransferResponse struct {
	List []struct {
		TransferID      string                  `json:"transferId"`
		Coin            string                  `json:"coin"`
		Amount          convert.StringToFloat64 `json:"amount"`
		FromAccountType string                  `json:"fromAccountType"`
		ToAccountType   string                  `json:"toAccountType"`
		Timestamp       convert.ExchangeTime    `json:"timestamp"`
		Status          string                  `json:"status"`

		// Returned with universal transfer IDs.
		FromMemberID string `json:"fromMemberId"`
		ToMemberID   string `json:"toMemberId"`
	} `json:"list"`
	NextPageCursor string `json:"nextPageCursor"`
}

// SubUID represents a sub-users ID
type SubUID struct {
	SubMemberIds             []string `json:"subMemberIds"`
	TransferableSubMemberIds []string `json:"transferableSubMemberIds"`
}

// AllowedDepositCoinInfo represents coin deposit information.
type AllowedDepositCoinInfo struct {
	ConfigList []struct {
		Coin               string `json:"coin"`
		Chain              string `json:"chain"`
		CoinShowName       string `json:"coinShowName"`
		ChainType          string `json:"chainType"`
		BlockConfirmNumber int    `json:"blockConfirmNumber"`
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
		Status        int    `json:"status"`
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
		ID          string                  `json:"id"`
		Amount      convert.StringToFloat64 `json:"amount"`
		Type        int64                   `json:"type"`
		Coin        string                  `json:"coin"`
		Address     string                  `json:"address"`
		Status      int64                   `json:"status"`
		CreatedTime convert.ExchangeTime    `json:"createdTime"`
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
		Name         string                  `json:"name"`
		Coin         string                  `json:"coin"`
		RemainAmount convert.StringToFloat64 `json:"remainAmount"`
		Chains       []struct {
			ChainType             string                  `json:"chainType"`
			Confirmation          string                  `json:"confirmation"`
			WithdrawFee           convert.StringToFloat64 `json:"withdrawFee"`
			DepositMin            convert.StringToFloat64 `json:"depositMin"`
			WithdrawMin           convert.StringToFloat64 `json:"withdrawMin"`
			Chain                 string                  `json:"chain"`
			ChainDeposit          convert.StringToFloat64 `json:"chainDeposit"`
			ChainWithdraw         string                  `json:"chainWithdraw"`
			MinAccuracy           convert.StringToFloat64 `json:"minAccuracy"`
			WithdrawPercentageFee convert.StringToFloat64 `json:"withdrawPercentageFee"`
		} `json:"chains"`
	} `json:"rows"`
}

// WithdrawalRecords represents a list of withdrawal records.
type WithdrawalRecords struct {
	Rows []struct {
		Coin          string                  `json:"coin"`
		Chain         string                  `json:"chain"`
		Amount        convert.StringToFloat64 `json:"amount"`
		TransactionID string                  `json:"txID"`
		Status        string                  `json:"status"`
		ToAddress     string                  `json:"toAddress"`
		Tag           string                  `json:"tag"`
		WithdrawFee   convert.StringToFloat64 `json:"withdrawFee"`
		CreateTime    convert.ExchangeTime    `json:"createTime"`
		UpdateTime    convert.ExchangeTime    `json:"updateTime"`
		WithdrawID    string                  `json:"withdrawId"`
		WithdrawType  int                     `json:"withdrawType"`
	} `json:"rows"`
	NextPageCursor string `json:"nextPageCursor"`
}

// WithdrawableAmount represents withdrawable amount information for each currency code
type WithdrawableAmount struct {
	LimitAmountUsd     string `json:"limitAmountUsd"`
	WithdrawableAmount map[string]struct {
		Coin               string                  `json:"coin"`
		WithdrawableAmount convert.StringToFloat64 `json:"withdrawableAmount"`
		AvailableBalance   convert.StringToFloat64 `json:"availableBalance"`
	} `json:"withdrawableAmount"`
}

// WithdrawalParam represents asset withdrawal request parameter.
type WithdrawalParam struct {
	Coin        currency.Code `json:"coin,omitempty"`
	Chain       string        `json:"chain,omitempty"`
	Address     string        `json:"address,omitempty"`
	Tag         string        `json:"tag,omitempty"`
	Amount      float64       `json:"amount,omitempty,string"`
	Timestamp   int64         `json:"timestamp,omitempty"`
	ForceChain  int64         `json:"forceChain,omitempty"`
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

	IPS           []string  `json:"ips"`
	Type          int64     `json:"type"`
	DeadlineDay   int64     `json:"deadlineDay"`
	ExpiredAt     time.Time `json:"expiredAt"`
	CreatedAt     time.Time `json:"createdAt"`
	Unified       int64     `json:"unified"`
	Uta           int64     `json:"uta"`
	UserID        int64     `json:"userID"`
	InviterID     int64     `json:"inviterID"`
	VipLevel      string    `json:"vipLevel"`
	MktMakerLevel string    `json:"mktMakerLevel"`
	AffiliateID   int64     `json:"affiliateID"`
	RsaPublicKey  string    `json:"rsaPublicKey"`
	IsMaster      bool      `json:"isMaster"`
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
	ReadOnly int64 `json:"readOnly"`
	// Set the IP bind. example: ["192.168.0.1,192.168.0.2"]note:
	// don't pass ips or pass with ["*"] means no bind
	// No ip bound api key will be invalid after 90 days
	// api key will be invalid after 7 days once the account password is changed
	IPs []string `json:"ips"`
	// Tick the types of permission. one of below types must be passed, otherwise the error is thrown
	Permissions map[string][]string `json:"permissions,omitempty"`
}

// AffiliateCustomerInfo represents user information
type AffiliateCustomerInfo struct {
	UID                 string                  `json:"uid"`
	TakerVol30Day       convert.StringToFloat64 `json:"takerVol30Day"`
	MakerVol30Day       convert.StringToFloat64 `json:"makerVol30Day"`
	TradeVol30Day       convert.StringToFloat64 `json:"tradeVol30Day"`
	DepositAmount30Day  convert.StringToFloat64 `json:"depositAmount30Day"`
	TakerVol365Day      convert.StringToFloat64 `json:"takerVol365Day"`
	MakerVol365Day      convert.StringToFloat64 `json:"makerVol365Day"`
	TradeVol365Day      convert.StringToFloat64 `json:"tradeVol365Day"`
	DepositAmount365Day convert.StringToFloat64 `json:"depositAmount365Day"`
	TotalWalletBalance  convert.StringToFloat64 `json:"totalWalletBalance"`
	DepositUpdateTime   time.Time               `json:"depositUpdateTime"`
	VipLevel            string                  `json:"vipLevel"`
	VolUpdateTime       time.Time               `json:"volUpdateTime"`
}

// LeverageTokenInfo represents leverage token information.
type LeverageTokenInfo struct {
	FundFee          convert.StringToFloat64 `json:"fundFee"`
	FundFeeTime      convert.ExchangeTime    `json:"fundFeeTime"`
	LtCoin           string                  `json:"ltCoin"`
	LtName           string                  `json:"ltName"`
	LtStatus         string                  `json:"ltStatus"`
	ManageFeeRate    convert.StringToFloat64 `json:"manageFeeRate"`
	ManageFeeTime    convert.ExchangeTime    `json:"manageFeeTime"`
	MaxPurchase      string                  `json:"maxPurchase"`
	MaxPurchaseDaily string                  `json:"maxPurchaseDaily"`
	MaxRedeem        string                  `json:"maxRedeem"`
	MaxRedeemDaily   string                  `json:"maxRedeemDaily"`
	MinPurchase      string                  `json:"minPurchase"`
	MinRedeem        string                  `json:"minRedeem"`
	NetValue         convert.StringToFloat64 `json:"netValue"`
	PurchaseFeeRate  convert.StringToFloat64 `json:"purchaseFeeRate"`
	RedeemFeeRate    convert.StringToFloat64 `json:"redeemFeeRate"`
	Total            convert.StringToFloat64 `json:"total"`
	Value            convert.StringToFloat64 `json:"value"`
}

// LeveragedTokenMarket represents leverage token market details.
type LeveragedTokenMarket struct {
	Basket      convert.StringToFloat64 `json:"basket"`
	Circulation convert.StringToFloat64 `json:"circulation"`
	Leverage    convert.StringToFloat64 `json:"leverage"` // Real leverage calculated by last traded price
	LTCoin      string                  `json:"ltCoin"`
	NetValue    convert.StringToFloat64 `json:"nav"`
	NavTime     convert.ExchangeTime    `json:"navTime"` // Update time for net asset value (in milliseconds and UTC time zone)
}

// LeverageToken represents a response instance when purchasing a leverage token.
type LeverageToken struct {
	Amount        convert.StringToFloat64 `json:"amount"`
	ExecAmt       convert.StringToFloat64 `json:"execAmt"`
	ExecQty       convert.StringToFloat64 `json:"execQty"`
	LtCoin        string                  `json:"ltCoin"`
	LtOrderStatus string                  `json:"ltOrderStatus"`
	PurchaseID    string                  `json:"purchaseId"`
	SerialNo      string                  `json:"serialNo"`
	ValueCoin     string                  `json:"valueCoin"`
}

// RedeemToken represents leverage redeem token
type RedeemToken struct {
	ExecAmt       convert.StringToFloat64 `json:"execAmt"`
	ExecQty       convert.StringToFloat64 `json:"execQty"`
	LtCoin        string                  `json:"ltCoin"`
	LtOrderStatus string                  `json:"ltOrderStatus"`
	Quantity      convert.StringToFloat64 `json:"quantity"`
	RedeemID      string                  `json:"redeemId"`
	SerialNo      string                  `json:"serialNo"`
	ValueCoin     string                  `json:"valueCoin"`
}

// RedeemPurchaseRecord represents a purchase and redeem record instance.
type RedeemPurchaseRecord struct {
	Amount        convert.StringToFloat64 `json:"amount"`
	Fee           convert.StringToFloat64 `json:"fee"`
	LtCoin        string                  `json:"ltCoin"`
	LtOrderStatus string                  `json:"ltOrderStatus"`
	LtOrderType   string                  `json:"ltOrderType"`
	OrderID       string                  `json:"orderId"`
	OrderTime     convert.ExchangeTime    `json:"orderTime"`
	SerialNo      string                  `json:"serialNo"`
	UpdateTime    convert.ExchangeTime    `json:"updateTime"`
	Value         convert.StringToFloat64 `json:"value"`
	ValueCoin     string                  `json:"valueCoin"`
}

// SpotMarginMode represents data about whether spot margin trade is on / off
type SpotMarginMode struct {
	SpotMarginMode string `json:"spotMarginMode"`
}

// VIPMarginData represents VIP margin data.
type VIPMarginData struct {
	VipCoinList []struct {
		List []struct {
			Borrowable         bool   `json:"borrowable"`
			CollateralRatio    string `json:"collateralRatio"`
			Currency           string `json:"currency"`
			HourlyBorrowRate   string `json:"hourlyBorrowRate"`
			LiquidationOrder   string `json:"liquidationOrder"`
			MarginCollateral   bool   `json:"marginCollateral"`
			MaxBorrowingAmount string `json:"maxBorrowingAmount"`
		} `json:"list"`
		VipLevel string `json:"vipLevel"`
	} `json:"vipCoinList"`
}

// MarginCoinInfo represents margin coin information.
type MarginCoinInfo struct {
	Coin             string `json:"coin"`
	ConversionRate   string `json:"conversionRate"`
	LiquidationOrder int    `json:"liquidationOrder"`
}

// BorrowableCoinInfo represents borrowable coin information.
type BorrowableCoinInfo struct {
	BorrowingPrecision int64  `json:"borrowingPrecision"`
	Coin               string `json:"coin"`
	RepaymentPrecision int64  `json:"repaymentPrecision"`
}

// InterestAndQuota represents interest and quota information.
type InterestAndQuota struct {
	Coin           string                  `json:"coin"`
	InterestRate   string                  `json:"interestRate"`
	LoanAbleAmount convert.StringToFloat64 `json:"loanAbleAmount"`
	MaxLoanAmount  convert.StringToFloat64 `json:"maxLoanAmount"`
}

// AccountLoanInfo covers: Margin trade (Normal Account)
type AccountLoanInfo struct {
	AcctBalanceSum  convert.StringToFloat64 `json:"acctBalanceSum"`
	DebtBalanceSum  convert.StringToFloat64 `json:"debtBalanceSum"`
	LoanAccountList []struct {
		Interest     string                  `json:"interest"`
		Loan         string                  `json:"loan"`
		Locked       string                  `json:"locked"`
		TokenID      string                  `json:"tokenId"`
		Free         convert.StringToFloat64 `json:"free"`
		RemainAmount convert.StringToFloat64 `json:"remainAmount"`
		Total        convert.StringToFloat64 `json:"total"`
	} `json:"loanAccountList"`
	RiskRate     convert.StringToFloat64 `json:"riskRate"`
	Status       int64                   `json:"status"`
	SwitchStatus int64                   `json:"switchStatus"`
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
	ID              string                  `json:"id"`
	AccountID       string                  `json:"accountId"`
	Coin            string                  `json:"coin"`
	CreatedTime     convert.ExchangeTime    `json:"createdTime"`
	InterestAmount  convert.StringToFloat64 `json:"interestAmount"`
	InterestBalance convert.StringToFloat64 `json:"interestBalance"`
	LoanAmount      convert.StringToFloat64 `json:"loanAmount"`
	LoanBalance     convert.StringToFloat64 `json:"loanBalance"`
	RemainAmount    convert.StringToFloat64 `json:"remainAmount"`
	Status          int64                   `json:"status"`
	Type            int64                   `json:"type"`
}

// CoinRepaymentResponse represents a coin repayment detail.
type CoinRepaymentResponse struct {
	AccountID          string                  `json:"accountId"`
	Coin               string                  `json:"coin"`
	RepaidAmount       convert.StringToFloat64 `json:"repaidAmount"`
	RepayID            string                  `json:"repayId"`
	RepayMarginOrderID string                  `json:"repayMarginOrderId"`
	RepayTime          convert.ExchangeTime    `json:"repayTime"`
	TransactIds        []struct {
		RepaidAmount       convert.StringToFloat64 `json:"repaidAmount"`
		RepaidInterest     convert.StringToFloat64 `json:"repaidInterest"`
		RepaidPrincipal    convert.StringToFloat64 `json:"repaidPrincipal"`
		RepaidSerialNumber convert.StringToFloat64 `json:"repaidSerialNumber"`
		TransactID         string                  `json:"transactId"`
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
				Ladder       string                  `json:"ladder"`
				ConvertRatio convert.StringToFloat64 `json:"convertRatio"`
			} `json:"convertRatioList"`
		} `json:"tokenInfo"`
	} `json:"marginToken"`
}

// LoanOrderDetails retrieves institutional loan order detail item.
type LoanOrderDetails struct {
	OrderID             string                  `json:"orderId"`
	OrderProductID      string                  `json:"orderProductId"`
	ParentUID           string                  `json:"parentUid"`
	LoanTime            convert.ExchangeTime    `json:"loanTime"`
	LoanCoin            string                  `json:"loanCoin"`
	LoanAmount          convert.StringToFloat64 `json:"loanAmount"`
	UnpaidAmount        convert.StringToFloat64 `json:"unpaidAmount"`
	UnpaidInterest      convert.StringToFloat64 `json:"unpaidInterest"`
	RepaidAmount        convert.StringToFloat64 `json:"repaidAmount"`
	RepaidInterest      convert.StringToFloat64 `json:"repaidInterest"`
	InterestRate        convert.StringToFloat64 `json:"interestRate"`
	Status              int64                   `json:"status"`
	Leverage            string                  `json:"leverage"`
	SupportSpot         int64                   `json:"supportSpot"`
	SupportContract     int64                   `json:"supportContract"`
	WithdrawLine        convert.StringToFloat64 `json:"withdrawLine"`
	TransferLine        convert.StringToFloat64 `json:"transferLine"`
	SpotBuyLine         convert.StringToFloat64 `json:"spotBuyLine"`
	SpotSellLine        convert.StringToFloat64 `json:"spotSellLine"`
	ContractOpenLine    convert.StringToFloat64 `json:"contractOpenLine"`
	LiquidationLine     convert.StringToFloat64 `json:"liquidationLine"`
	StopLiquidationLine convert.StringToFloat64 `json:"stopLiquidationLine"`
	ContractLeverage    convert.StringToFloat64 `json:"contractLeverage"`
	TransferRatio       convert.StringToFloat64 `json:"transferRatio"`
	SpotSymbols         []string                `json:"spotSymbols"`
	ContractSymbols     []string                `json:"contractSymbols"`
}

// OrderRepayInfo represents repaid information information.
type OrderRepayInfo struct {
	RepayOrderID string               `json:"repayOrderId"`
	RepaidTime   convert.ExchangeTime `json:"repaidTime"`
	Token        string               `json:"token"`
	Quantity     convert.ExchangeTime `json:"quantity"`
	Interest     convert.ExchangeTime `json:"interest"`
	BusinessType string               `json:"businessType"`
	Status       string               `json:"status"`
}

// LTVInfo represents institutional lending Loan-to-value(LTV)
type LTVInfo struct {
	LtvInfo []struct {
		LoanToValue    string                  `json:"ltv"`
		ParentUID      string                  `json:"parentUid"`
		SubAccountUids []string                `json:"subAccountUids"`
		UnpaidAmount   convert.StringToFloat64 `json:"unpaidAmount"`
		UnpaidInfo     []struct {
			Token          string                  `json:"token"`
			UnpaidQty      convert.StringToFloat64 `json:"unpaidQty"`
			UnpaidInterest convert.StringToFloat64 `json:"unpaidInterest"`
		} `json:"unpaidInfo"`
		Balance     string `json:"balance"`
		BalanceInfo []struct {
			Token           string                  `json:"token"`
			Price           convert.StringToFloat64 `json:"price"`
			Qty             convert.StringToFloat64 `json:"qty"`
			ConvertedAmount convert.StringToFloat64 `json:"convertedAmount"`
		} `json:"balanceInfo"`
	} `json:"ltvInfo"`
}

// C2CLendingCoinInfo represent contract-to-contract lending coin information.
type C2CLendingCoinInfo struct {
	Coin            string                  `json:"coin"`
	LoanToPoolRatio convert.StringToFloat64 `json:"loanToPoolRatio"`
	MaxRedeemQty    convert.StringToFloat64 `json:"maxRedeemQty"`
	MinPurchaseQty  convert.StringToFloat64 `json:"minPurchaseQty"`
	Precision       convert.StringToFloat64 `json:"precision"`
	Rate            convert.StringToFloat64 `json:"rate"`
}

// C2CLendingFundsParams represents deposit funds parameter
type C2CLendingFundsParams struct {
	Coin         currency.Code `json:"coin"`
	Quantity     float64       `json:"quantity,string"`
	SerialNumber string        `json:"serialNO"`
}

// C2CLendingFundResponse represents contract-to-contract deposit funds item.
type C2CLendingFundResponse struct {
	Coin         string                  `json:"coin"`
	OrderID      string                  `json:"orderId"`
	SerialNumber string                  `json:"serialNo"`
	Status       string                  `json:"status"`
	CreatedTime  convert.ExchangeTime    `json:"createdTime"`
	Quantity     convert.StringToFloat64 `json:"quantity"`
	UpdatedTime  convert.ExchangeTime    `json:"updatedTime"`

	// added for redeem funds
	PrincipalQty string `json:"principalQty"`

	// added for to distinguish between redeem funds and deposit funds
	OrderType string `json:"orderType"`
}

// LendingAccountInfo represents contract-to-contract lending account info item.
type LendingAccountInfo struct {
	Coin              string                  `json:"coin"`
	PrincipalInterest string                  `json:"principalInterest"`
	PrincipalQty      convert.StringToFloat64 `json:"principalQty"`
	PrincipalTotal    convert.StringToFloat64 `json:"principalTotal"`
	Quantity          convert.StringToFloat64 `json:"quantity"`
}

// BrokerEarningItem represents contract-to-contract broker earning item.
type BrokerEarningItem struct {
	UserID   string               `json:"userId"`
	BizType  string               `json:"bizType"`
	Symbol   string               `json:"symbol"`
	Coin     string               `json:"coin"`
	Earning  string               `json:"earning"`
	OrderID  string               `json:"orderId"`
	ExecTime convert.ExchangeTime `json:"execTime"`
}

// ServerTime represents server time
type ServerTime struct {
	TimeSecond convert.ExchangeTime `json:"timeSecond"`
	TimeNano   convert.ExchangeTime `json:"timeNano"`
}

// Orderbook stores the orderbook data
type Orderbook struct {
	UpdateID       int64
	Bids           []orderbook.Item
	Asks           []orderbook.Item
	Symbol         string
	GenerationTime time.Time
}

// WsOrderbookDetail represents an orderbook detail information.
type WsOrderbookDetail struct {
	Symbol     string               `json:"s"`
	Bids       [][]string           `json:"b"`
	Asks       [][]string           `json:"a"`
	UpdateTime convert.ExchangeTime `json:"u"`
	Sequence   int64                `json:"seq"`
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
	Topic         string               `json:"topic"`
	Type          string               `json:"type"`
	Timestamp     convert.ExchangeTime `json:"ts"`
	Data          json.RawMessage      `json:"data"`
	CrossSequence int64                `json:"cs"`

	// for ping messages
	Operation string `json:"op"`

	// for subscription response checks.
	RequestID string `json:"req_id"`
}

// WebsocketPublicTrades represents
type WebsocketPublicTrades []struct {
	OrderFillTimestamp   convert.ExchangeTime    `json:"T"`
	Symbol               string                  `json:"s"`
	Side                 string                  `json:"S"`
	Size                 convert.StringToFloat64 `json:"v"`
	Price                convert.StringToFloat64 `json:"p"`
	PriceChangeDirection string                  `json:"L"`
	TradeID              string                  `json:"i"`
	BlockTrade           bool                    `json:"BT"`
}

// WsLinearTicker represents a linear ticker information.
type WsLinearTicker struct {
	Symbol            string                  `json:"symbol"`
	TickDirection     string                  `json:"tickDirection"`
	Price24HPcnt      convert.StringToFloat64 `json:"price24hPcnt"`
	LastPrice         convert.StringToFloat64 `json:"lastPrice"`
	PrevPrice24H      convert.StringToFloat64 `json:"prevPrice24h"`
	HighPrice24H      convert.StringToFloat64 `json:"highPrice24h"`
	LowPrice24H       convert.StringToFloat64 `json:"lowPrice24h"`
	PrevPrice1H       convert.StringToFloat64 `json:"prevPrice1h"`
	MarkPrice         convert.StringToFloat64 `json:"markPrice"`
	IndexPrice        convert.StringToFloat64 `json:"indexPrice"`
	OpenInterest      convert.StringToFloat64 `json:"openInterest"`
	OpenInterestValue convert.StringToFloat64 `json:"openInterestValue"`
	Turnover24H       convert.StringToFloat64 `json:"turnover24h"`
	Volume24H         convert.StringToFloat64 `json:"volume24h"`
	NextFundingTime   convert.ExchangeTime    `json:"nextFundingTime"`
	FundingRate       convert.StringToFloat64 `json:"fundingRate"`
	Bid1Price         convert.StringToFloat64 `json:"bid1Price"`
	Bid1Size          convert.StringToFloat64 `json:"bid1Size"`
	Ask1Price         convert.StringToFloat64 `json:"ask1Price"`
	Ask1Size          convert.StringToFloat64 `json:"ask1Size"`
}

// WsOptionTicker represents options public ticker data.
type WsOptionTicker struct {
	Symbol                 string                  `json:"symbol"`
	BidPrice               convert.StringToFloat64 `json:"bidPrice"`
	BidSize                convert.StringToFloat64 `json:"bidSize"`
	BidIv                  convert.StringToFloat64 `json:"bidIv"`
	AskPrice               convert.StringToFloat64 `json:"askPrice"`
	AskSize                convert.StringToFloat64 `json:"askSize"`
	AskIv                  convert.StringToFloat64 `json:"askIv"`
	LastPrice              convert.StringToFloat64 `json:"lastPrice"`
	HighPrice24H           convert.StringToFloat64 `json:"highPrice24h"`
	LowPrice24H            convert.StringToFloat64 `json:"lowPrice24h"`
	MarkPrice              convert.StringToFloat64 `json:"markPrice"`
	IndexPrice             convert.StringToFloat64 `json:"indexPrice"`
	MarkPriceIv            convert.StringToFloat64 `json:"markPriceIv"`
	UnderlyingPrice        convert.StringToFloat64 `json:"underlyingPrice"`
	OpenInterest           convert.StringToFloat64 `json:"openInterest"`
	Turnover24H            convert.StringToFloat64 `json:"turnover24h"`
	Volume24H              convert.StringToFloat64 `json:"volume24h"`
	TotalVolume            convert.StringToFloat64 `json:"totalVolume"`
	TotalTurnover          convert.StringToFloat64 `json:"totalTurnover"`
	Delta                  convert.StringToFloat64 `json:"delta"`
	Gamma                  convert.StringToFloat64 `json:"gamma"`
	Vega                   convert.StringToFloat64 `json:"vega"`
	Theta                  convert.StringToFloat64 `json:"theta"`
	PredictedDeliveryPrice convert.StringToFloat64 `json:"predictedDeliveryPrice"`
	Change24H              convert.StringToFloat64 `json:"change24h"`
}

// WsSpotTicker represents a spot public ticker information.
type WsSpotTicker struct {
	Symbol        string                  `json:"symbol"`
	LastPrice     convert.StringToFloat64 `json:"lastPrice"`
	HighPrice24H  convert.StringToFloat64 `json:"highPrice24h"`
	LowPrice24H   convert.StringToFloat64 `json:"lowPrice24h"`
	PrevPrice24H  convert.StringToFloat64 `json:"prevPrice24h"`
	Volume24H     convert.StringToFloat64 `json:"volume24h"`
	Turnover24H   convert.StringToFloat64 `json:"turnover24h"`
	Price24HPcnt  convert.StringToFloat64 `json:"price24hPcnt"`
	UsdIndexPrice convert.StringToFloat64 `json:"usdIndexPrice"`
}

// WsKlines represents a list of Kline data.
type WsKlines []struct {
	Start     convert.ExchangeTime    `json:"start"`
	End       convert.ExchangeTime    `json:"end"`
	Interval  string                  `json:"interval"`
	Open      convert.StringToFloat64 `json:"open"`
	Close     convert.StringToFloat64 `json:"close"`
	High      convert.StringToFloat64 `json:"high"`
	Low       convert.StringToFloat64 `json:"low"`
	Volume    convert.StringToFloat64 `json:"volume"`
	Turnover  string                  `json:"turnover"`
	Confirm   bool                    `json:"confirm"`
	Timestamp convert.ExchangeTime    `json:"timestamp"`
}

// WebsocketLiquidiation represents liquidation stream push data.
type WebsocketLiquidiation struct {
	Symbol      string                  `json:"symbol"`
	Side        string                  `json:"side"`
	Price       convert.StringToFloat64 `json:"price"`
	Size        convert.StringToFloat64 `json:"size"`
	UpdatedTime convert.ExchangeTime    `json:"updatedTime"`
}

// LTKlines represents a leverage token kline.
type LTKlines []struct {
	Start     convert.ExchangeTime    `json:"start"`
	End       convert.ExchangeTime    `json:"end"`
	Interval  string                  `json:"interval"`
	Open      convert.StringToFloat64 `json:"open"`
	Close     convert.StringToFloat64 `json:"close"`
	High      convert.StringToFloat64 `json:"high"`
	Low       convert.StringToFloat64 `json:"low"`
	Confirm   bool                    `json:"confirm"`
	Timestamp convert.ExchangeTime    `json:"timestamp"`
}

// LTTicker represents a leverage token ticker.
type LTTicker struct {
	Symbol             string                  `json:"symbol"`
	LastPrice          convert.StringToFloat64 `json:"lastPrice"`
	HighPrice24H       convert.StringToFloat64 `json:"highPrice24h"`
	LowPrice24H        convert.StringToFloat64 `json:"lowPrice24h"`
	PrevPrice24H       convert.StringToFloat64 `json:"prevPrice24h"`
	Price24HPercentage convert.StringToFloat64 `json:"price24hPcnt"`
}

// LTNav represents leveraged token nav stream.
type LTNav struct {
	Symbol         string                  `json:"symbol"`
	Time           convert.ExchangeTime    `json:"time"`
	Nav            convert.StringToFloat64 `json:"nav"`
	BasketPosition convert.StringToFloat64 `json:"basketPosition"`
	Leverage       convert.StringToFloat64 `json:"leverage"`
	BasketLoan     convert.StringToFloat64 `json:"basketLoan"`
	Circulation    convert.StringToFloat64 `json:"circulation"`
	Basket         convert.StringToFloat64 `json:"basket"`
}

// WsPositions represents a position information.
type WsPositions []struct {
	PositionIdx      int                     `json:"positionIdx"`
	TradeMode        int                     `json:"tradeMode"`
	RiskID           int                     `json:"riskId"`
	RiskLimitValue   convert.StringToFloat64 `json:"riskLimitValue"`
	Symbol           string                  `json:"symbol"`
	Side             string                  `json:"side"`
	Size             convert.StringToFloat64 `json:"size"`
	EntryPrice       convert.StringToFloat64 `json:"entryPrice"`
	Leverage         convert.StringToFloat64 `json:"leverage"`
	PositionValue    convert.StringToFloat64 `json:"positionValue"`
	PositionBalance  convert.StringToFloat64 `json:"positionBalance"`
	MarkPrice        convert.StringToFloat64 `json:"markPrice"`
	PositionIM       convert.StringToFloat64 `json:"positionIM"`
	PositionMM       convert.StringToFloat64 `json:"positionMM"`
	TakeProfit       convert.StringToFloat64 `json:"takeProfit"`
	StopLoss         convert.StringToFloat64 `json:"stopLoss"`
	TrailingStop     convert.StringToFloat64 `json:"trailingStop"`
	UnrealisedPnl    convert.StringToFloat64 `json:"unrealisedPnl"`
	CumRealisedPnl   convert.StringToFloat64 `json:"cumRealisedPnl"`
	CreatedTime      convert.ExchangeTime    `json:"createdTime"`
	UpdatedTime      convert.ExchangeTime    `json:"updatedTime"`
	TpslMode         string                  `json:"tpslMode"`
	LiqPrice         convert.StringToFloat64 `json:"liqPrice"`
	BustPrice        convert.StringToFloat64 `json:"bustPrice"`
	Category         string                  `json:"category"`
	PositionStatus   string                  `json:"positionStatus"`
	AdlRankIndicator int                     `json:"adlRankIndicator"`
}

// WsExecutions represents execution stream to see your executions in real-time.
type WsExecutions []struct {
	Category        string                  `json:"category"`
	Symbol          string                  `json:"symbol"`
	ExecFee         convert.StringToFloat64 `json:"execFee"`
	ExecID          string                  `json:"execId"`
	ExecPrice       convert.StringToFloat64 `json:"execPrice"`
	ExecQty         convert.StringToFloat64 `json:"execQty"`
	ExecType        string                  `json:"execType"`
	ExecValue       convert.StringToFloat64 `json:"execValue"`
	IsMaker         bool                    `json:"isMaker"`
	FeeRate         string                  `json:"feeRate"`
	TradeIv         string                  `json:"tradeIv"`
	MarkIv          string                  `json:"markIv"`
	BlockTradeID    string                  `json:"blockTradeId"`
	MarkPrice       convert.StringToFloat64 `json:"markPrice"`
	IndexPrice      convert.StringToFloat64 `json:"indexPrice"`
	UnderlyingPrice convert.StringToFloat64 `json:"underlyingPrice"`
	LeavesQty       convert.StringToFloat64 `json:"leavesQty"`
	OrderID         string                  `json:"orderId"`
	OrderLinkID     string                  `json:"orderLinkId"`
	OrderPrice      convert.StringToFloat64 `json:"orderPrice"`
	OrderQty        convert.StringToFloat64 `json:"orderQty"`
	OrderType       string                  `json:"orderType"`
	StopOrderType   string                  `json:"stopOrderType"`
	Side            string                  `json:"side"`
	ExecTime        convert.ExchangeTime    `json:"execTime"`
	IsLeverage      convert.StringToFloat64 `json:"isLeverage"`
	ClosedSize      convert.StringToFloat64 `json:"closedSize"`
}

// WsOrders represents private order
type WsOrders []struct {
	Symbol             string                  `json:"symbol"`
	OrderID            string                  `json:"orderId"`
	Side               string                  `json:"side"`
	OrderType          string                  `json:"orderType"`
	CancelType         string                  `json:"cancelType"`
	Price              convert.StringToFloat64 `json:"price"`
	Qty                convert.StringToFloat64 `json:"qty"`
	OrderIv            string                  `json:"orderIv"`
	TimeInForce        string                  `json:"timeInForce"`
	OrderStatus        string                  `json:"orderStatus"`
	OrderLinkID        string                  `json:"orderLinkId"`
	LastPriceOnCreated string                  `json:"lastPriceOnCreated"`
	ReduceOnly         bool                    `json:"reduceOnly"`
	LeavesQty          convert.StringToFloat64 `json:"leavesQty"`
	LeavesValue        convert.StringToFloat64 `json:"leavesValue"`
	CumExecQty         convert.StringToFloat64 `json:"cumExecQty"`
	CumExecValue       convert.StringToFloat64 `json:"cumExecValue"`
	AvgPrice           convert.StringToFloat64 `json:"avgPrice"`
	BlockTradeID       string                  `json:"blockTradeId"`
	PositionIdx        int                     `json:"positionIdx"`
	CumExecFee         convert.StringToFloat64 `json:"cumExecFee"`
	CreatedTime        convert.ExchangeTime    `json:"createdTime"`
	UpdatedTime        convert.ExchangeTime    `json:"updatedTime"`
	RejectReason       string                  `json:"rejectReason"`
	StopOrderType      string                  `json:"stopOrderType"`
	TpslMode           string                  `json:"tpslMode"`
	TriggerPrice       convert.StringToFloat64 `json:"triggerPrice"`
	TakeProfit         convert.StringToFloat64 `json:"takeProfit"`
	StopLoss           convert.StringToFloat64 `json:"stopLoss"`
	TpTriggerBy        convert.StringToFloat64 `json:"tpTriggerBy"`
	SlTriggerBy        convert.StringToFloat64 `json:"slTriggerBy"`
	TpLimitPrice       convert.StringToFloat64 `json:"tpLimitPrice"`
	SlLimitPrice       convert.StringToFloat64 `json:"slLimitPrice"`
	TriggerDirection   int                     `json:"triggerDirection"`
	TriggerBy          string                  `json:"triggerBy"`
	CloseOnTrigger     bool                    `json:"closeOnTrigger"`
	Category           string                  `json:"category"`
	PlaceType          string                  `json:"placeType"`
	SmpType            string                  `json:"smpType"` // SMP execution type
	SmpGroup           int                     `json:"smpGroup"`
	SmpOrderID         string                  `json:"smpOrderId"`
}

// WebsocketWallet represents a wallet stream to see changes to your wallet in real-time.
type WebsocketWallet struct {
	ID           string               `json:"id"`
	Topic        string               `json:"topic"`
	CreationTime convert.ExchangeTime `json:"creationTime"`
	Data         []struct {
		AccountIMRate          convert.StringToFloat64 `json:"accountIMRate"`
		AccountMMRate          convert.StringToFloat64 `json:"accountMMRate"`
		TotalEquity            convert.StringToFloat64 `json:"totalEquity"`
		TotalWalletBalance     convert.StringToFloat64 `json:"totalWalletBalance"`
		TotalMarginBalance     convert.StringToFloat64 `json:"totalMarginBalance"`
		TotalAvailableBalance  convert.StringToFloat64 `json:"totalAvailableBalance"`
		TotalPerpUPL           convert.StringToFloat64 `json:"totalPerpUPL"`
		TotalInitialMargin     convert.StringToFloat64 `json:"totalInitialMargin"`
		TotalMaintenanceMargin convert.StringToFloat64 `json:"totalMaintenanceMargin"`
		Coin                   []struct {
			Coin                string                  `json:"coin"`
			Equity              convert.StringToFloat64 `json:"equity"`
			UsdValue            convert.StringToFloat64 `json:"usdValue"`
			WalletBalance       convert.StringToFloat64 `json:"walletBalance"`
			AvailableToWithdraw convert.StringToFloat64 `json:"availableToWithdraw"`
			AvailableToBorrow   convert.StringToFloat64 `json:"availableToBorrow"`
			BorrowAmount        convert.StringToFloat64 `json:"borrowAmount"`
			AccruedInterest     convert.StringToFloat64 `json:"accruedInterest"`
			TotalOrderIM        convert.StringToFloat64 `json:"totalOrderIM"`
			TotalPositionIM     convert.StringToFloat64 `json:"totalPositionIM"`
			TotalPositionMM     convert.StringToFloat64 `json:"totalPositionMM"`
			UnrealisedPnl       convert.StringToFloat64 `json:"unrealisedPnl"`
			CumRealisedPnl      convert.StringToFloat64 `json:"cumRealisedPnl"`
			Bonus               convert.StringToFloat64 `json:"bonus"`
		} `json:"coin"`
		AccountType string `json:"accountType"`
		AccountLTV  string `json:"accountLTV"`
	} `json:"data"`
}

// GreeksResponse represents changes to your greeks data
type GreeksResponse struct {
	ID           string               `json:"id"`
	Topic        string               `json:"topic"`
	CreationTime convert.ExchangeTime `json:"creationTime"`
	Data         []struct {
		BaseCoin   string                  `json:"baseCoin"`
		TotalDelta convert.StringToFloat64 `json:"totalDelta"`
		TotalGamma convert.StringToFloat64 `json:"totalGamma"`
		TotalVega  convert.StringToFloat64 `json:"totalVega"`
		TotalTheta convert.StringToFloat64 `json:"totalTheta"`
	} `json:"data"`
}

// PingMessage represents a ping message.
type PingMessage struct {
	Operation string `json:"op"`
	RequestID string `json:"req_id"`
}

// InstrumentInfoItem represents an instrument long short ratio information.
type InstrumentInfoItem struct {
	Symbol    string                  `json:"symbol"`
	BuyRatio  convert.StringToFloat64 `json:"buyRatio"`
	SellRatio convert.StringToFloat64 `json:"sellRatio"`
	Timestamp convert.ExchangeTime    `json:"timestamp"`
}
