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

// FuturesPublicTradesData stores recent public trades for futures
type FuturesPublicTradesData struct {
	ID           int64      `json:"id"`
	Price        float64    `json:"price,string"`
	Qty          float64    `json:"qty,string"`
	QuoteQty     float64    `json:"quoteQty,string"`
	Time         types.Time `json:"time"`
	IsBuyerMaker bool       `json:"isBuyerMaker"`
}

// CompressedTradesData stores futures trades data in a compressed format
type CompressedTradesData struct {
	TradeID      int64      `json:"a"`
	Price        float64    `json:"p"`
	Quantity     float64    `json:"q"`
	FirstTradeID int64      `json:"f"`
	LastTradeID  int64      `json:"l"`
	Timestamp    types.Time `json:"t"`
	BuyerMaker   bool       `json:"b"`
}

// MarkPriceData stores mark price data for futures
type MarkPriceData struct {
	Symbol          string     `json:"symbol"`
	MarkPrice       float64    `json:"markPrice"`
	LastFundingRate float64    `json:"lastFundingRate"`
	NextFundingTime types.Time `json:"nextFundingTime"`
	Time            types.Time `json:"time"`
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

// AllLiquidationOrders gets all liquidation orders
type AllLiquidationOrders struct {
	Symbol       string     `json:"symbol"`
	Price        float64    `json:"price,string"`
	OrigQty      float64    `json:"origQty,string"`
	ExecutedQty  float64    `json:"executedQty,string"`
	AveragePrice float64    `json:"averagePrice,string"`
	Status       string     `json:"status"`
	TimeInForce  string     `json:"timeInForce"`
	OrderType    string     `json:"type"`
	Side         string     `json:"side"`
	Time         types.Time `json:"time"`
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

// GlobalLongShortRatio stores ratio data of all longs vs shorts
type GlobalLongShortRatio struct {
	Symbol         string     `json:"symbol"`
	LongShortRatio float64    `json:"longShortRatio"`
	LongAccount    float64    `json:"longAccount"`
	ShortAccount   float64    `json:"shortAccount"`
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
	ClientOrderID string     `json:"clientOrderID"`
	CumQty        float64    `json:"cumQty,string"`
	CumBase       float64    `json:"cumBase,string"`
	ExecuteQty    float64    `json:"executeQty,string"`
	OrderID       int64      `json:"orderID,string"`
	AvgPrice      float64    `json:"avgPrice,string"`
	OrigQty       float64    `json:"origQty,string"`
	Price         float64    `json:"price,string"`
	ReduceOnly    bool       `json:"reduceOnly"`
	Side          string     `json:"side"`
	PositionSide  string     `json:"positionSide"`
	Status        string     `json:"status"`
	StopPrice     int64      `json:"stopPrice"`
	ClosePosition bool       `json:"closePosition"`
	Symbol        string     `json:"symbol"`
	Pair          string     `json:"pair"`
	TimeInForce   string     `json:"TimeInForce"`
	OrderType     string     `json:"type"`
	OrigType      string     `json:"origType"`
	ActivatePrice float64    `json:"activatePrice,string"`
	PriceRate     float64    `json:"priceRate,string"`
	UpdateTime    types.Time `json:"updateTime"`
	WorkingType   string     `json:"workingType"`
	PriceProtect  bool       `json:"priceProtect"`
	Code          int64      `json:"code"`
	Msg           string     `json:"msg"`
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
	ClientOrderID string     `json:"clientOrderId"`
	CumQty        float64    `json:"cumQty,string"`
	CumBase       float64    `json:"cumBase,string"`
	ExecuteQty    float64    `json:"executedQty,string"`
	OrderID       int64      `json:"orderId"`
	AvgPrice      float64    `json:"avgPrice,string"`
	OrigQty       float64    `json:"origQty,string"`
	Price         float64    `json:"price,string"`
	ReduceOnly    bool       `json:"reduceOnly"`
	Side          string     `json:"side"`
	PositionSide  string     `json:"positionSide"`
	Status        string     `json:"status"`
	StopPrice     float64    `json:"stopPrice,string"`
	ClosePosition bool       `json:"closePosition"`
	Symbol        string     `json:"symbol"`
	Pair          string     `json:"pair"`
	TimeInForce   string     `json:"TimeInForce"`
	OrderType     string     `json:"type"`
	OrigType      string     `json:"origType"`
	ActivatePrice float64    `json:"activatePrice,string"`
	PriceRate     float64    `json:"priceRate,string"`
	UpdateTime    types.Time `json:"updateTime"`
	WorkingType   string     `json:"workingType"`
	PriceProtect  bool       `json:"priceProtect"`
}

// FuturesOrderGetData stores futures order data for get requests
type FuturesOrderGetData struct {
	AveragePrice       float64    `json:"avgPrice,string"`
	ClientOrderID      string     `json:"clientOrderID"`
	CumulativeQuantity float64    `json:"cumQty,string"`
	CumulativeBase     float64    `json:"cumBase,string"`
	ExecutedQuantity   float64    `json:"executedQty,string"`
	OrderID            int64      `json:"orderId"`
	OriginalQuantity   float64    `json:"origQty,string"`
	OriginalType       string     `json:"origType"`
	Price              float64    `json:"price,string"`
	ReduceOnly         bool       `json:"reduceOnly"`
	Side               string     `json:"buy"`
	PositionSide       string     `json:"positionSide"`
	Status             string     `json:"status"`
	StopPrice          float64    `json:"stopPrice,string"`
	ClosePosition      bool       `json:"closePosition"`
	Symbol             string     `json:"symbol"`
	Pair               string     `json:"pair"`
	TimeInForce        string     `json:"timeInForce"`
	OrderType          string     `json:"type"`
	ActivatePrice      float64    `json:"activatePrice,string"`
	PriceRate          float64    `json:"priceRate,string"`
	Time               types.Time `json:"time"`
	UpdateTime         types.Time `json:"updateTime"`
	WorkingType        string     `json:"workingType"`
	PriceProtect       bool       `json:"priceProtect"`
}

// FuturesOrderData stores order data for futures
type FuturesOrderData struct {
	AvgPrice      float64    `json:"avgPrice,string"`
	ClientOrderID string     `json:"clientOrderId"`
	CumBase       string     `json:"cumBase"`
	ExecutedQty   float64    `json:"executedQty,string"`
	OrderID       int64      `json:"orderId"`
	OrigQty       float64    `json:"origQty,string"`
	OrigType      string     `json:"origType"`
	Price         float64    `json:"price,string"`
	ReduceOnly    bool       `json:"reduceOnly"`
	Side          string     `json:"side"`
	PositionSide  string     `json:"positionSide"`
	Status        string     `json:"status"`
	StopPrice     float64    `json:"stopPrice,string"`
	ClosePosition bool       `json:"closePosition"`
	Symbol        string     `json:"symbol"`
	Pair          string     `json:"pair"`
	Time          types.Time `json:"time"`
	TimeInForce   string     `json:"timeInForce"`
	OrderType     string     `json:"type"`
	ActivatePrice float64    `json:"activatePrice,string"`
	PriceRate     float64    `json:"priceRate,string"`
	UpdateTime    types.Time `json:"updateTime"`
	WorkingType   string     `json:"workingType"`
	PriceProtect  bool       `json:"priceProtect"`
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

// LevelDetail stores level detail data
type LevelDetail struct {
	Level         string  `json:"level"`
	MaxBorrowable float64 `json:"maxBorrowable,string"`
	InterestRate  float64 `json:"interestRate,string"`
}

// MarginInfoData stores margin info data
type MarginInfoData struct {
	Data []struct {
		MarginRatio string `json:"marginRatio"`
		Base        struct {
			AssetName    string        `json:"assetName"`
			LevelDetails []LevelDetail `json:"levelDetails"`
		} `json:"base"`
		Quote struct {
			AssetName    string        `json:"assetName"`
			LevelDetails []LevelDetail `json:"levelDetails"`
		} `json:"quote"`
	} `json:"data"`
}

// FuturesAccountBalanceData stores account balance data for futures
type FuturesAccountBalanceData struct {
	AccountAlias       string     `json:"accountAlias"`
	Asset              string     `json:"asset"`
	Balance            float64    `json:"balance,string"`
	WithdrawAvailable  float64    `json:"withdrawAvailable,string"`
	CrossWalletBalance float64    `json:"crossWalletBalance,string"`
	CrossUnPNL         float64    `json:"crossUnPNL,string"`
	AvailableBalance   float64    `json:"availableBalance,string"`
	UpdateTime         types.Time `json:"updateTime"`
}

// FuturesAccountInformationPosition  holds account position data
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

// ModifyIsolatedMarginData stores margin modification data
type ModifyIsolatedMarginData struct {
	Amount  float64 `json:"amount"`
	Code    int64   `json:"code"`
	Msg     string  `json:"msg"`
	ModType string  `json:"modType"`
}

// GetPositionMarginChangeHistoryData gets margin change history for positions
type GetPositionMarginChangeHistoryData struct {
	Amount           float64    `json:"amount"`
	Asset            string     `json:"asset"`
	Symbol           string     `json:"symbol"`
	Timestamp        types.Time `json:"time"`
	MarginChangeType int64      `json:"type"`
	PositionSide     string     `json:"positionSide"`
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
	Symbol          string     `json:"symbol"`
	ID              int64      `json:"id"`
	OrderID         int64      `json:"orderID"`
	Pair            string     `json:"pair"`
	Side            string     `json:"side"`
	Price           string     `json:"price"`
	Qty             float64    `json:"qty"`
	RealizedPNL     float64    `json:"realizedPNL"`
	MarginAsset     string     `json:"marginAsset"`
	BaseQty         float64    `json:"baseQty"`
	Commission      float64    `json:"commission"`
	CommissionAsset string     `json:"commissionAsset"`
	Timestamp       types.Time `json:"timestamp"`
	PositionSide    string     `json:"positionSide"`
	Buyer           bool       `json:"buyer"`
	Maker           bool       `json:"maker"`
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
	Pair     string `json:"pair"`
	Brackets []struct {
		Bracket          int64   `json:"bracket"`
		InitialLeverage  float64 `json:"initialLeverage"`
		QtyCap           float64 `json:"qtyCap"`
		QtylFloor        float64 `json:"qtyFloor"`
		MaintMarginRatio float64 `json:"maintMarginRatio"`
	}
}

// ForcedOrdersData stores forced orders data
type ForcedOrdersData struct {
	OrderID       int64      `json:"orderId"`
	Symbol        string     `json:"symbol"`
	Status        string     `json:"status"`
	ClientOrderID string     `json:"clientOrderId"`
	Price         float64    `json:"price,string"`
	AvgPrice      float64    `json:"avgPrice,string"`
	OrigQty       float64    `json:"origQty,string"`
	ExecutedQty   float64    `json:"executedQty,string"`
	CumQuote      float64    `json:"cumQuote,string"`
	TimeInForce   string     `json:"timeInForce"`
	OrderType     string     `json:"orderType"`
	ReduceOnly    bool       `json:"reduceOnly"`
	ClosePosition bool       `json:"closePosition"`
	Side          string     `json:"side"`
	PositionSide  string     `json:"positionSide"`
	StopPrice     float64    `json:"stopPrice,string"`
	WorkingType   string     `json:"workingType"`
	PriceProtect  float64    `json:"priceProtect,string"`
	OrigType      string     `json:"origType"`
	Time          types.Time `json:"time"`
	UpdateTime    types.Time `json:"updateTime"`
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

// InterestHistoryData gets interest history data
type InterestHistoryData struct {
	Asset       string     `json:"asset"`
	Interest    float64    `json:"interest"`
	LendingType string     `json:"lendingType"`
	ProductName string     `json:"productName"`
	Time        types.Time `json:"time"`
}

// FundingRateData stores funding rates data
type FundingRateData struct {
	Symbol      string     `json:"symbol"`
	FundingRate float64    `json:"fundingRate,string"`
	FundingTime types.Time `json:"fundingTime"`
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
	OrderTypes      []string `json:"OrderType"`
	TimeInForce     []string `json:"timeInForce"`
	LiquidationFee  float64  `json:"liquidationFee,string"`
	MarketTakeBound float64  `json:"marketTakeBound,string"`
}

// CExchangeInfo stores exchange info for cfutures
type CExchangeInfo struct {
	ExchangeFilters []any `json:"exchangeFilters"`
	RateLimits      []struct {
		Interval      string `json:"interval"`
		IntervalNum   int64  `json:"intervalNul"`
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
		OrderTypes            []string   `json:"orderType"`
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
