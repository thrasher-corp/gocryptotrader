package binance

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Response holds basic binance api response data
type Response struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

// FuturesPublicTradesData stores recent public trades for futures. Quantity counts contracts, so
// BaseQuantity is the traded base asset amount; coin margined futures sends no quote figure
type FuturesPublicTradesData struct {
	ID           int64        `json:"id"`
	Price        types.Number `json:"price"`
	Quantity     types.Number `json:"qty"`
	BaseQuantity types.Number `json:"baseQty"`
	Time         types.Time   `json:"time"`
	IsBuyerMaker bool         `json:"isBuyerMaker"`
}

// SymbolPriceTicker stores ticker price stats
type SymbolPriceTicker struct {
	Symbol string     `json:"symbol"`
	Price  float64    `json:"price,string"`
	Time   types.Time `json:"time"`
}

// SymbolOrderBookTicker stores orderbook ticker data
type SymbolOrderBookTicker struct {
	Symbol   string     `json:"symbol"`
	BidPrice float64    `json:"bidPrice,string"`
	AskPrice float64    `json:"askPrice,string"`
	BidQty   float64    `json:"bidQty,string"`
	AskQty   float64    `json:"askQty,string"`
	Time     types.Time `json:"time"`
}

// FuturesCandleStick holds kline data
type FuturesCandleStick struct {
	OpenTime                types.Time
	Open                    types.Number
	High                    types.Number
	Low                     types.Number
	Close                   types.Number
	Volume                  types.Number
	CloseTime               types.Time
	BaseAssetVolume         types.Number
	NumberOfTrades          int64
	TakerBuyVolume          types.Number
	TakerBuyBaseAssetVolume types.Number
}

// UnmarshalJSON unmarshals FuturesCandleStick data from JSON
func (f *FuturesCandleStick) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[11]any{&f.OpenTime, &f.Open, &f.High, &f.Low, &f.Close, &f.Volume, &f.CloseTime, &f.BaseAssetVolume, &f.NumberOfTrades, &f.TakerBuyVolume, &f.TakerBuyBaseAssetVolume})
}

// OpenInterestData stores open interest data
type OpenInterestData struct {
	Symbol       string     `json:"symbol"`
	Pair         string     `json:"pair"`
	OpenInterest float64    `json:"openInterest,string"`
	ContractType string     `json:"contractType"`
	Time         types.Time `json:"time"`
}

// OpenInterestStats stores stats for open interest data
type OpenInterestStats struct {
	Pair                 string     `json:"pair"`
	ContractType         string     `json:"contractType"`
	SumOpenInterest      float64    `json:"sumOpenInterest,string"`
	SumOpenInterestValue float64    `json:"sumOpenInterestValue,string"`
	Timestamp            types.Time `json:"timestamp"`
}

// TopTraderAccountRatio stores account ratio data for top traders
type TopTraderAccountRatio struct {
	Pair           string     `json:"pair"`
	LongShortRatio float64    `json:"longShortRatio,string"`
	LongAccount    float64    `json:"longAccount,string"`
	ShortAccount   float64    `json:"shortAccount,string"`
	Timestamp      types.Time `json:"timestamp"`
}

// TopTraderPositionRatio stores position ratio for top trader accounts
type TopTraderPositionRatio struct {
	Pair           string     `json:"pair"`
	LongShortRatio float64    `json:"longShortRatio,string"`
	LongPosition   float64    `json:"longPosition,string"`
	ShortPosition  float64    `json:"shortPosition,string"`
	Timestamp      types.Time `json:"timestamp"`
}

// TakerBuySellVolume stores taker buy sell volume
type TakerBuySellVolume struct {
	Pair           string     `json:"pair"`
	ContractType   string     `json:"contractType"`
	TakerBuyVolume float64    `json:"takerBuyVol,string"`
	BuySellRatio   float64    `json:"takerSellVol,string"`
	BuyVol         float64    `json:"takerBuyVolValue,string"`
	SellVol        float64    `json:"takerSellVolValue,string"`
	Timestamp      types.Time `json:"timestamp"`
}

// FuturesBasisData gets futures basis data
type FuturesBasisData struct {
	Pair         string     `json:"pair"`
	ContractType string     `json:"contractType"`
	FuturesPrice float64    `json:"futuresPrice,string"`
	IndexPrice   float64    `json:"indexPrice,string"`
	Basis        float64    `json:"basis,string"`
	BasisRate    float64    `json:"basisRate,string"`
	Timestamp    types.Time `json:"timestamp"`
}

// PlaceBatchOrderData stores batch order data for placing
type PlaceBatchOrderData struct {
	Symbol           string  `json:"symbol"`
	Side             string  `json:"side"`
	PositionSide     string  `json:"positionSide,omitempty"`
	OrderType        string  `json:"type"`
	TimeInForce      string  `json:"timeInForce,omitempty"`
	Quantity         float64 `json:"quantity"`
	ReduceOnly       string  `json:"reduceOnly,omitempty"`
	Price            float64 `json:"price"`
	NewClientOrderID string  `json:"newClientOrderId,omitempty"`
	StopPrice        float64 `json:"stopPrice,omitempty"`
	ActivationPrice  float64 `json:"activationPrice,omitempty"`
	CallbackRate     float64 `json:"callbackRate,omitempty"`
	WorkingType      string  `json:"workingType,omitempty"`
	PriceProtect     string  `json:"priceProtect,omitempty"`
	NewOrderRespType string  `json:"newOrderRespType,omitempty"`
}

// BatchCancelOrderData stores batch cancel order data
type BatchCancelOrderData struct {
	ClientOrderID           string       `json:"clientOrderId"`
	CumulativeQuantity      types.Number `json:"cumQty"`
	ExecutedQuantity        types.Number `json:"executedQty"`
	OrderID                 int64        `json:"orderId"`
	OriginalQuantity        types.Number `json:"origQty"`
	Price                   types.Number `json:"price"`
	ReduceOnly              bool         `json:"reduceOnly"`
	Side                    string       `json:"side"`
	PositionSide            string       `json:"positionSide"`
	Status                  string       `json:"status"`
	StopPrice               types.Number `json:"stopPrice"`
	ClosePosition           bool         `json:"closePosition"`
	Symbol                  string       `json:"symbol"`
	Pair                    string       `json:"pair"`
	TimeInForce             string       `json:"timeInForce"`
	OrderType               string       `json:"type"`
	OriginalType            string       `json:"origType"`
	ActivatePrice           types.Number `json:"activatePrice"`
	PriceRate               types.Number `json:"priceRate"`
	UpdateTime              types.Time   `json:"updateTime"`
	WorkingType             string       `json:"workingType"`
	PriceProtect            bool         `json:"priceProtect"`
	PriceMatch              string       `json:"priceMatch"`
	SelfTradePreventionMode string       `json:"selfTradePreventionMode"`
	Code                    int64        `json:"code"`
	Message                 string       `json:"msg"`
}

// FuturesNewOrderRequest stores all the data needed to submit a
// delivery/coin-margined-futures order.
type FuturesNewOrderRequest struct {
	Symbol           currency.Pair
	Side             string
	PositionSide     string
	OrderType        string
	TimeInForce      string
	NewClientOrderID string
	ClosePosition    string
	WorkingType      string
	NewOrderRespType string
	Quantity         float64
	Price            float64
	StopPrice        float64
	ActivationPrice  float64
	CallbackRate     float64
	ReduceOnly       bool
	PriceProtect     bool
}

// FuturesOrderPlaceData stores futures order data
type FuturesOrderPlaceData struct {
	ClientOrderID           string       `json:"clientOrderId"`
	CumulativeQuantity      types.Number `json:"cumQty"`
	ExecutedQuantity        types.Number `json:"executedQty"`
	OrderID                 int64        `json:"orderId"`
	OriginalQuantity        types.Number `json:"origQty"`
	Price                   types.Number `json:"price"`
	ReduceOnly              bool         `json:"reduceOnly"`
	Side                    string       `json:"side"`
	PositionSide            string       `json:"positionSide"`
	Status                  string       `json:"status"`
	StopPrice               types.Number `json:"stopPrice"`
	ClosePosition           bool         `json:"closePosition"`
	Symbol                  string       `json:"symbol"`
	Pair                    string       `json:"pair"`
	TimeInForce             string       `json:"timeInForce"`
	OrderType               string       `json:"type"`
	OriginalType            string       `json:"origType"`
	ActivatePrice           types.Number `json:"activatePrice"`
	PriceRate               types.Number `json:"priceRate"`
	UpdateTime              types.Time   `json:"updateTime"`
	WorkingType             string       `json:"workingType"`
	PriceProtect            bool         `json:"priceProtect"`
	PriceMatch              string       `json:"priceMatch"`
	SelfTradePreventionMode string       `json:"selfTradePreventionMode"`
	// Code and Message carry a per-item rejection in a batch placement response, where an entry
	// that failed is otherwise indistinguishable from a zero-valued success
	Code    int64  `json:"code"`
	Message string `json:"msg"`
}

// FuturesOrderGetData stores futures order data for get requests
type FuturesOrderGetData struct {
	AveragePrice            types.Number `json:"avgPrice"`
	ClientOrderID           string       `json:"clientOrderId"`
	CumulativeQuantity      types.Number `json:"cumQty"`
	CumulativeBase          types.Number `json:"cumBase"`
	ExecutedQuantity        types.Number `json:"executedQty"`
	OrderID                 int64        `json:"orderId"`
	OriginalQuantity        types.Number `json:"origQty"`
	OriginalType            string       `json:"origType"`
	Price                   types.Number `json:"price"`
	ReduceOnly              bool         `json:"reduceOnly"`
	Side                    string       `json:"side"`
	PositionSide            string       `json:"positionSide"`
	Status                  string       `json:"status"`
	StopPrice               types.Number `json:"stopPrice"`
	ClosePosition           bool         `json:"closePosition"`
	Symbol                  string       `json:"symbol"`
	Pair                    string       `json:"pair"`
	TimeInForce             string       `json:"timeInForce"`
	OrderType               string       `json:"type"`
	ActivatePrice           types.Number `json:"activatePrice"`
	PriceRate               types.Number `json:"priceRate"`
	Time                    types.Time   `json:"time"`
	UpdateTime              types.Time   `json:"updateTime"`
	WorkingType             string       `json:"workingType"`
	PriceProtect            bool         `json:"priceProtect"`
	PriceMatch              string       `json:"priceMatch"`
	SelfTradePreventionMode string       `json:"selfTradePreventionMode"`
}

// FuturesOrderData stores order data for futures. cumQuote and goodTillDate were added to
// allOrders on 2026-08-05; openOrders shares this type and does not send them, so they are zero there
type FuturesOrderData struct {
	AveragePrice            types.Number `json:"avgPrice"`
	ClientOrderID           string       `json:"clientOrderId"`
	CumulativeBase          types.Number `json:"cumBase"`
	CumulativeQuote         types.Number `json:"cumQuote"`
	ExecutedQuantity        types.Number `json:"executedQty"`
	OrderID                 int64        `json:"orderId"`
	OriginalQuantity        types.Number `json:"origQty"`
	OriginalType            string       `json:"origType"`
	Price                   types.Number `json:"price"`
	ReduceOnly              bool         `json:"reduceOnly"`
	Side                    string       `json:"side"`
	PositionSide            string       `json:"positionSide"`
	Status                  string       `json:"status"`
	StopPrice               types.Number `json:"stopPrice"`
	ClosePosition           bool         `json:"closePosition"`
	Symbol                  string       `json:"symbol"`
	Pair                    string       `json:"pair"`
	Time                    types.Time   `json:"time"`
	TimeInForce             string       `json:"timeInForce"`
	OrderType               string       `json:"type"`
	ActivatePrice           types.Number `json:"activatePrice"`
	PriceRate               types.Number `json:"priceRate"`
	UpdateTime              types.Time   `json:"updateTime"`
	WorkingType             string       `json:"workingType"`
	PriceProtect            bool         `json:"priceProtect"`
	PriceMatch              string       `json:"priceMatch"`
	SelfTradePreventionMode string       `json:"selfTradePreventionMode"`
	GoodTillDate            types.Time   `json:"goodTillDate"`
}

// OrderVars stores side, status and type for any order/trade
type OrderVars struct {
	Side      order.Side
	Status    order.Status
	OrderType order.Type
	Fee       float64
}

// AutoCancelAllOrdersData gives data of auto cancelling all open orders
type AutoCancelAllOrdersData struct {
	Symbol        string `json:"symbol"`
	CountdownTime int64  `json:"countdownTime,string"`
}

// FuturesAccountBalanceData stores account balance data for futures
type FuturesAccountBalanceData struct {
	AccountAlias       string     `json:"accountAlias"`
	Asset              string     `json:"asset"`
	Balance            float64    `json:"balance,string"`
	WithdrawAvailable  float64    `json:"withdrawAvailable,string"`
	CrossWalletBalance float64    `json:"crossWalletBalance,string"`
	CrossUnPNL         float64    `json:"crossUnPnl,string"`
	AvailableBalance   float64    `json:"availableBalance,string"`
	UpdateTime         types.Time `json:"updateTime"`
}

// FuturesAccountInformationPosition holds account position data
type FuturesAccountInformationPosition struct {
	Symbol                 string     `json:"symbol"`
	Amount                 float64    `json:"positionAmt,string"`
	InitialMargin          float64    `json:"initialMargin,string"`
	MaintenanceMargin      float64    `json:"maintMargin,string"`
	UnrealizedProfit       float64    `json:"unrealizedProfit,string"`
	PositionInitialMargin  float64    `json:"positionInitialMargin,string"`
	OpenOrderInitialMargin float64    `json:"openOrderInitialMargin,string"`
	Leverage               float64    `json:"leverage,string"`
	Isolated               bool       `json:"isolated"`
	PositionSide           string     `json:"positionSide"`
	EntryPrice             float64    `json:"entryPrice,string"`
	MaxQty                 float64    `json:"maxQty,string"`
	UpdateTime             types.Time `json:"updateTime"`
	NotionalValue          float64    `json:"notionalValue,string"`
	IsolatedWallet         float64    `json:"isolatedWallet,string"`
}

// FuturesAccountInformation stores account information for futures account
type FuturesAccountInformation struct {
	Assets      []FuturesAccountAsset               `json:"assets"`
	Positions   []FuturesAccountInformationPosition `json:"positions"`
	CanDeposit  bool                                `json:"canDeposit"`
	CanTrade    bool                                `json:"canTrade"`
	CanWithdraw bool                                `json:"canWithdraw"`
	FeeTier     int64                               `json:"feeTier"`
	UpdateTime  types.Time                          `json:"updateTime"`
}

// FuturesAccountAsset holds account asset information
type FuturesAccountAsset struct {
	Asset                  currency.Code `json:"asset"`
	WalletBalance          float64       `json:"walletBalance,string"`
	UnrealizedProfit       float64       `json:"unrealizedProfit,string"`
	MarginBalance          float64       `json:"marginBalance,string"`
	MaintenanceMargin      float64       `json:"maintMargin,string"`
	InitialMargin          float64       `json:"initialMargin,string"`
	PositionInitialMargin  float64       `json:"positionInitialMargin,string"`
	OpenOrderInitialMargin float64       `json:"openOrderInitialMargin,string"`
	MaxWithdrawAmount      float64       `json:"maxWithdrawAmount,string"`
	CrossWalletBalance     float64       `json:"crossWalletBalance,string"`
	CrossUnPNL             float64       `json:"crossUnPnl,string"`
	AvailableBalance       float64       `json:"availableBalance,string"`
}

// GenericAuthResponse is a general data response for a post auth request
type GenericAuthResponse struct {
	Code int64  `json:"code"`
	Msg  string `json:"msg"`
}

// FuturesMarginUpdatedResponse stores margin update response data
type FuturesMarginUpdatedResponse struct {
	Amount float64 `json:"amount"`
	Type   int     `json:"type"`
	GenericAuthResponse
}

// FuturesLeverageData stores leverage data for futures
type FuturesLeverageData struct {
	Leverage int64   `json:"leverage"`
	MaxQty   float64 `json:"maxQty,string"`
	Symbol   string  `json:"symbol"`
}

// GetPositionMarginChangeHistoryData gets margin change history for positions
type GetPositionMarginChangeHistoryData struct {
	Amount           types.Number  `json:"amount"`
	Asset            currency.Code `json:"asset"`
	Symbol           string        `json:"symbol"`
	Timestamp        types.Time    `json:"time"`
	MarginChangeType int64         `json:"type"`
	PositionSide     string        `json:"positionSide"`
}

// FuturesPositionInformation stores futures position info
type FuturesPositionInformation struct {
	Symbol           string     `json:"symbol"`
	PositionAmount   float64    `json:"positionAmt,string"`
	EntryPrice       float64    `json:"entryPrice,string"`
	MarkPrice        float64    `json:"markPrice,string"`
	UnRealizedProfit float64    `json:"unRealizedProfit,string"`
	LiquidationPrice float64    `json:"liquidationPrice,string"`
	Leverage         float64    `json:"leverage,string"`
	MaxQty           float64    `json:"maxQty,string"`
	MarginType       string     `json:"marginType"`
	IsolatedMargin   float64    `json:"isolatedMargin,string"`
	IsAutoAddMargin  bool       `json:"isAutoAddMargin,string"`
	PositionSide     string     `json:"positionSide"`
	NotionalValue    float64    `json:"notionalValue,string"`
	IsolatedWallet   float64    `json:"isolatedWallet,string"`
	UpdateTime       types.Time `json:"updateTime"`
}

// FuturesAccountTradeList stores account trade list data
type FuturesAccountTradeList struct {
	Symbol          string        `json:"symbol"`
	ID              int64         `json:"id"`
	OrderID         int64         `json:"orderId"`
	Pair            string        `json:"pair"`
	Side            string        `json:"side"`
	Price           types.Number  `json:"price"`
	Quantity        types.Number  `json:"qty"`
	RealizedPNL     types.Number  `json:"realizedPnl"`
	MarginAsset     currency.Code `json:"marginAsset"`
	BaseQuantity    types.Number  `json:"baseQty"`
	QuoteQuantity   types.Number  `json:"quoteQty"`
	Commission      types.Number  `json:"commission"`
	CommissionAsset currency.Code `json:"commissionAsset"`
	Timestamp       types.Time    `json:"time"`
	PositionSide    string        `json:"positionSide"`
	Buyer           bool          `json:"buyer"`
	Maker           bool          `json:"maker"`
}

// FuturesIncomeHistoryData stores futures income history data
type FuturesIncomeHistoryData struct {
	Symbol     string     `json:"symbol"`
	IncomeType string     `json:"incomeType"`
	Income     float64    `json:"income,string"`
	Asset      string     `json:"asset"`
	Info       string     `json:"info"`
	Timestamp  types.Time `json:"time"`
}

// NotionalBracketData stores notional bracket data
type NotionalBracketData struct {
	Pair     string            `json:"pair"`
	Brackets []NotionalBracket `json:"brackets"`
}

// NotionalBracket is a single leverage bracket
type NotionalBracket struct {
	Bracket          int64   `json:"bracket"`
	InitialLeverage  float64 `json:"initialLeverage"`
	QtyCap           float64 `json:"qtyCap"`
	QtylFloor        float64 `json:"qtylFloor"` // Binance's own spelling, typo included
	MaintMarginRatio float64 `json:"maintMarginRatio"`
	Cumulative       float64 `json:"cum"`
}

// ForcedOrdersData stores forced orders data
type ForcedOrdersData struct {
	OrderID          int64        `json:"orderId"`
	Symbol           string       `json:"symbol"`
	Pair             string       `json:"pair"`
	Status           string       `json:"status"`
	ClientOrderID    string       `json:"clientOrderId"`
	Price            types.Number `json:"price"`
	AveragePrice     types.Number `json:"avgPrice"`
	OriginalQuantity types.Number `json:"origQty"`
	ExecutedQuantity types.Number `json:"executedQty"`
	CumulativeBase   types.Number `json:"cumBase"`
	CumulativeQuote  types.Number `json:"cumQuote"`
	TimeInForce      string       `json:"timeInForce"`
	OrderType        string       `json:"type"`
	ReduceOnly       bool         `json:"reduceOnly"`
	ClosePosition    bool         `json:"closePosition"`
	Side             string       `json:"side"`
	PositionSide     string       `json:"positionSide"`
	StopPrice        types.Number `json:"stopPrice"`
	WorkingType      string       `json:"workingType"`
	PriceProtect     bool         `json:"priceProtect"`
	OriginalType     string       `json:"origType"`
	Time             types.Time   `json:"time"`
	UpdateTime       types.Time   `json:"updateTime"`
	GoodTillDate     types.Time   `json:"goodTillDate"`
}

// ADLEstimateData stores data for ADL estimates
type ADLEstimateData struct {
	Symbol      string `json:"symbol"`
	ADLQuantile struct {
		Long  float64 `json:"LONG"`
		Short float64 `json:"SHORT"`
		Hedge float64 `json:"HEDGE"`
	} `json:"adlQuantile"`
}

// SymbolsData stores perp futures' symbols
type SymbolsData struct {
	Symbol string `json:"symbol"`
}

// PerpsExchangeInfo stores data for perps
type PerpsExchangeInfo struct {
	Symbols []SymbolsData `json:"symbols"`
}

// UFuturesExchangeInfo stores exchange info for ufutures
type UFuturesExchangeInfo struct {
	RateLimits []struct {
		Interval      string `json:"interval"`
		IntervalNum   int64  `json:"intervalNum"`
		Limit         int64  `json:"limit"`
		RateLimitType string `json:"rateLimitType"`
	} `json:"rateLimits"`
	ServerTime types.Time           `json:"serverTime"`
	Symbols    []UFuturesSymbolInfo `json:"symbols"`
	Timezone   string               `json:"timezone"`
}

// UFuturesSymbolInfo contains details of a currency symbol
// for a usdt margined future contract
type UFuturesSymbolInfo struct {
	Symbol                   string     `json:"symbol"`
	Pair                     string     `json:"pair"`
	ContractType             string     `json:"contractType"`
	DeliveryDate             types.Time `json:"deliveryDate"`
	OnboardDate              types.Time `json:"onboardDate"`
	Status                   string     `json:"status"`
	MaintenanceMarginPercent float64    `json:"maintMarginPercent,string"`
	RequiredMarginPercent    float64    `json:"requiredMarginPercent,string"`
	BaseAsset                string     `json:"baseAsset"`
	QuoteAsset               string     `json:"quoteAsset"`
	MarginAsset              string     `json:"marginAsset"`
	PricePrecision           int64      `json:"pricePrecision"`
	QuantityPrecision        int64      `json:"quantityPrecision"`
	BaseAssetPrecision       int64      `json:"baseAssetPrecision"`
	QuotePrecision           int64      `json:"quotePrecision"`
	UnderlyingType           string     `json:"underlyingType"`
	UnderlyingSubType        []string   `json:"underlyingSubType"`
	SettlePlan               float64    `json:"settlePlan"`
	TriggerProtect           float64    `json:"triggerProtect,string"`
	Filters                  []struct {
		FilterType        string  `json:"filterType"`
		MinPrice          float64 `json:"minPrice,string"`
		MaxPrice          float64 `json:"maxPrice,string"`
		TickSize          float64 `json:"tickSize,string"`
		StepSize          float64 `json:"stepSize,string"`
		MaxQty            float64 `json:"maxQty,string"`
		MinQty            float64 `json:"minQty,string"`
		Limit             int64   `json:"limit"`
		MultiplierDown    float64 `json:"multiplierDown,string"`
		MultiplierUp      float64 `json:"multiplierUp,string"`
		MultiplierDecimal float64 `json:"multiplierDecimal,string"`
		Notional          float64 `json:"notional,string"`
	} `json:"filters"`
	OrderTypes      []string `json:"orderTypes"`
	TimeInForce     []string `json:"timeInForce"`
	LiquidationFee  float64  `json:"liquidationFee,string"`
	MarketTakeBound float64  `json:"marketTakeBound,string"`
}

// CExchangeInfo stores exchange info for cfutures
type CExchangeInfo struct {
	ExchangeFilters []any `json:"exchangeFilters"`
	RateLimits      []struct {
		Interval      string `json:"interval"`
		IntervalNum   int64  `json:"intervalNum"`
		Limit         int64  `json:"limit"`
		RateLimitType string `json:"rateLimitType"`
	} `json:"rateLimits"`
	ServerTime types.Time `json:"serverTime"`
	Symbols    []struct {
		Filters []struct {
			FilterType        string  `json:"filterType"`
			MinPrice          float64 `json:"minPrice,string"`
			MaxPrice          float64 `json:"maxPrice,string"`
			StepSize          float64 `json:"stepSize,string"`
			TickSize          float64 `json:"tickSize,string"`
			MaxQty            float64 `json:"maxQty,string"`
			MinQty            float64 `json:"minQty,string"`
			Limit             int64   `json:"limit"`
			MultiplierDown    float64 `json:"multiplierDown,string"`
			MultiplierUp      float64 `json:"multiplierUp,string"`
			MultiplierDecimal float64 `json:"multiplierDecimal,string"`
		} `json:"filters"`
		OrderTypes            []string   `json:"orderTypes"`
		TimeInForce           []string   `json:"timeInForce"`
		Symbol                string     `json:"symbol"`
		Pair                  string     `json:"pair"`
		ContractType          string     `json:"contractType"`
		DeliveryDate          types.Time `json:"deliveryDate"`
		OnboardDate           types.Time `json:"onboardDate"`
		ContractStatus        string     `json:"contractStatus"`
		ContractSize          int64      `json:"contractSize"`
		QuoteAsset            string     `json:"quoteAsset"`
		BaseAsset             string     `json:"baseAsset"`
		MarginAsset           string     `json:"marginAsset"`
		PricePrecision        int64      `json:"pricePrecision"`
		QuantityPrecision     int64      `json:"quantityPrecision"`
		BaseAssetPrecision    int64      `json:"baseAssetPrecision"`
		QuotePrecision        int64      `json:"quotePrecision"`
		MaintMarginPercent    float64    `json:"maintMarginPercent,string"`
		RequiredMarginPercent float64    `json:"requiredMarginPercent,string"`
	} `json:"symbols"`
	Timezone string `json:"timezone"`
}
