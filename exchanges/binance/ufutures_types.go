package binance

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

var (
	validFuturesIntervals = []string{
		"1m", "3m", "5m", "15m", "30m",
		"1h", "2h", "4h", "6h", "8h",
		"12h", "1d", "3d", "1w", "1M",
	}

	validContractType = []string{
		"ALL", "CURRENT_QUARTER", "NEXT_QUARTER",
	}

	validNewOrderRespType = []string{"ACK", "RESULT"}

	validWorkingType = []string{"MARK_PRICE", "CONTRACT_TYPE"}

	validPositionSide = []string{"BOTH", "LONG", "SHORT"}

	validMarginType = []string{"ISOLATED", "CROSSED"}

	validIncomeType = []string{"TRANSFER", "WELCOME_BONUS", "REALIZED_PNL", "FUNDING_FEE", "COMMISSION", "INSURANCE_CLEAR"}

	validAutoCloseTypes = []string{"LIQUIDATION", "ADL"}

	validMarginChange = map[string]int64{
		"add":    1,
		"reduce": 2,
	}

	uValidOBLimits = []string{"5", "10", "20", "50", "100", "500", "1000"}

	uValidPeriods = []string{"5m", "15m", "30m", "1h", "2h", "4h", "6h", "12h", "1d"}
)

// UPublicTradesData stores trade data
type UPublicTradesData struct {
	ID           int64      `json:"id"`
	Price        float64    `json:"price,string"`
	Qty          float64    `json:"qty,string"`
	QuoteQty     float64    `json:"quoteQty,string"`
	Time         types.Time `json:"time"`
	IsBuyerMaker bool       `json:"isBuyerMaker"`
}

// UCompressedTradeData stores compressed trade data
type UCompressedTradeData struct {
	AggregateTradeID int64      `json:"a"`
	Price            float64    `json:"p,string"`
	Quantity         float64    `json:"q,string"`
	FirstTradeID     int64      `json:"f"`
	LastTradeID      int64      `json:"l"`
	Timestamp        types.Time `json:"t"`
	IsBuyerMaker     bool       `json:"m"`
}

// UMarkPrice stores mark price data
type UMarkPrice struct {
	Symbol               string     `json:"symbol"`
	MarkPrice            float64    `json:"markPrice,string"`
	IndexPrice           float64    `json:"indexPrice,string"`
	LastFundingRate      float64    `json:"lastFundingRate,string"`
	EstimatedSettlePrice float64    `json:"estimatedSettlePrice,string"`
	NextFundingTime      types.Time `json:"nextFundingTime"`
	Time                 types.Time `json:"time"`
}

// FundingRateInfoResponse stores funding rate info
type FundingRateInfoResponse struct {
	Symbol                   string       `json:"symbol"`
	AdjustedFundingRateCap   types.Number `json:"adjustedFundingRateCap"`
	AdjustedFundingRateFloor types.Number `json:"adjustedFundingRateFloor"`
	FundingIntervalHours     int64        `json:"fundingIntervalHours"`
	Disclaimer               bool         `json:"disclaimer"`
}

// FundingRateHistory stores funding rate history
type FundingRateHistory struct {
	Symbol      string     `json:"symbol"`
	FundingRate float64    `json:"fundingRate,string"`
	FundingTime types.Time `json:"fundingTime"`
}

// U24HrPriceChangeStats stores price change stats data
type U24HrPriceChangeStats struct {
	Symbol             string     `json:"symbol"`
	PriceChange        float64    `json:"priceChange,string"`
	PriceChangePercent float64    `json:"priceChangePercent,string"`
	WeightedAvgPrice   float64    `json:"weightedAvgPrice,string"`
	PrevClosePrice     float64    `json:"prevClosePrice,string"`
	LastPrice          float64    `json:"lastPrice,string"`
	LastQty            float64    `json:"lastQty,string"`
	OpenPrice          float64    `json:"openPrice,string"`
	HighPrice          float64    `json:"highPrice,string"`
	LowPrice           float64    `json:"lowPrice,string"`
	Volume             float64    `json:"volume,string"`
	QuoteVolume        float64    `json:"quoteVolume,string"`
	OpenTime           types.Time `json:"openTime"`
	CloseTime          types.Time `json:"closeTime"`
	FirstID            int64      `json:"firstId"`
	LastID             int64      `json:"lastId"`
	Count              int64      `json:"count"`
}

// USymbolPriceTicker stores symbol price ticker data
type USymbolPriceTicker struct {
	Symbol string     `json:"symbol"`
	Price  float64    `json:"price,string"`
	Time   types.Time `json:"time"`
}

// USymbolOrderbookTicker stores symbol orderbook ticker data
type USymbolOrderbookTicker struct {
	Symbol   string     `json:"symbol"`
	BidPrice float64    `json:"bidPrice,string"`
	BidQty   float64    `json:"bidQty,string"`
	AskPrice float64    `json:"askPrice,string"`
	AskQty   float64    `json:"askQty,string"`
	Time     types.Time `json:"time"`
}

// ULiquidationOrdersData stores liquidation orders data
type ULiquidationOrdersData struct {
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

// UOpenInterestData stores open interest data
type UOpenInterestData struct {
	OpenInterest float64    `json:"openInterest,string"`
	Symbol       string     `json:"symbol"`
	Time         types.Time `json:"time"`
}

// UOpenInterestStats stores open interest stats data
type UOpenInterestStats struct {
	Symbol               string     `json:"symbol"`
	SumOpenInterest      float64    `json:"sumOpenInterest,string"`
	SumOpenInterestValue float64    `json:"sumOpenInterestValue,string"`
	Timestamp            types.Time `json:"timestamp"`
}

// ULongShortRatio stores top trader accounts' or positions' or global long/short ratio data
type ULongShortRatio struct {
	Symbol         string     `json:"symbol"`
	LongShortRatio float64    `json:"longShortRatio,string"`
	LongAccount    float64    `json:"longAccount,string"`
	ShortAccount   float64    `json:"shortAccount,string"`
	Timestamp      types.Time `json:"timestamp"`
}

// UTakerVolumeData stores volume data on buy/sell side from takers
type UTakerVolumeData struct {
	BuySellRatio float64    `json:"buySellRatio,string"`
	BuyVol       float64    `json:"buyVol,string"`
	SellVol      float64    `json:"sellVol,string"`
	Timestamp    types.Time `json:"timestamp"`
}

// UCompositeIndexInfoData stores composite index data for usdt margined futures
type UCompositeIndexInfoData struct {
	Symbol        string     `json:"symbol"`
	Time          types.Time `json:"time"`
	BaseAssetList []struct {
		BaseAsset          currency.Code `json:"baseAsset"`
		QuoteAsset         currency.Code `json:"quoteAsset"`
		WeightInQuantity   float64       `json:"weightInQuantity,string"`
		WeightInPercentage float64       `json:"weightInPercentage,string"`
	} `json:"baseAssetList"`
}

// UOrderData stores order data
type UOrderData struct {
	ClientOrderID      string     `json:"clientOrderId"`
	Time               types.Time `json:"time"`
	CumulativeQuantity float64    `json:"cumQty,string"`
	CumulativeQuote    float64    `json:"cumQuote,string"`
	ExecutedQuantity   float64    `json:"executedQty,string"`
	OrderID            int64      `json:"orderId"`
	AveragePrice       float64    `json:"avgPrice,string"`
	OriginalQuantity   float64    `json:"origQty,string"`
	Price              float64    `json:"price,string"`
	ReduceOnly         bool       `json:"reduceOnly"`
	Side               string     `json:"side"`
	PositionSide       string     `json:"positionSide"`
	Status             string     `json:"status"`
	StopPrice          float64    `json:"stopPrice,string"`
	ClosePosition      bool       `json:"closePosition"`
	Symbol             string     `json:"symbol"`
	TimeInForce        string     `json:"timeInForce"`
	OrderType          string     `json:"type"`
	OriginalType       string     `json:"origType"`
	ActivatePrice      float64    `json:"activatePrice,string"`
	PriceRate          float64    `json:"priceRate,string"`
	UpdateTime         types.Time `json:"updateTime"`
	WorkingType        string     `json:"workingType"`
	Code               int64      `json:"code"`
	Message            string     `json:"msg"`
}

// UFuturesOrderData stores order data for ufutures
type UFuturesOrderData struct {
	AvgPrice      float64    `json:"avgPrice,string"`
	ClientOrderID string     `json:"clientOrderId"`
	CumQuote      string     `json:"cumQuote"`
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
	Time          types.Time `json:"time"`
	TimeInForce   string     `json:"timeInForce"`
	OrderType     string     `json:"type"`
	ActivatePrice float64    `json:"activatePrice,string"`
	PriceRate     float64    `json:"priceRate,string"`
	UpdateTime    types.Time `json:"updateTime"`
	WorkingType   string     `json:"workingType"`
}

// UAccountBalanceV2Data stores account balance data for ufutures
type UAccountBalanceV2Data struct {
	AccountAlias       string        `json:"accountAlias"`
	Asset              currency.Code `json:"asset"`
	Balance            float64       `json:"balance,string"`
	CrossWalletBalance float64       `json:"crossWalletBalance,string"`
	CrossUnrealizedPNL float64       `json:"crossUnPnl,string"`
	AvailableBalance   float64       `json:"availableBalance,string"`
	MaxWithdrawAmount  float64       `json:"maxWithdrawAmount,string"`
}

// UAccountInformationV2Data stores account info for ufutures
type UAccountInformationV2Data struct {
	FeeTier                     int64       `json:"feeTier"`
	CanTrade                    bool        `json:"canTrade"`
	CanDeposit                  bool        `json:"canDeposit"`
	CanWithdraw                 bool        `json:"canWithdraw"`
	UpdateTime                  types.Time  `json:"updateTime"`
	MultiAssetsMargin           bool        `json:"multiAssetsMargin"`
	TotalInitialMargin          float64     `json:"totalInitialMargin,string"`
	TotalMaintenanceMargin      float64     `json:"totalMaintMargin,string"`
	TotalWalletBalance          float64     `json:"totalWalletBalance,string"`
	TotalUnrealizedProfit       float64     `json:"totalUnrealizedProfit,string"`
	TotalMarginBalance          float64     `json:"totalMarginBalance,string"`
	TotalPositionInitialMargin  float64     `json:"totalPositionInitialMargin,string"`
	TotalOpenOrderInitialMargin float64     `json:"totalOpenOrderInitialMargin,string"`
	TotalCrossWalletBalance     float64     `json:"totalCrossWalletBalance,string"`
	TotalCrossUnrealizedPNL     float64     `json:"totalCrossUnPnl,string"`
	AvailableBalance            float64     `json:"availableBalance,string"`
	MaxWithdrawAmount           float64     `json:"maxWithdrawAmount,string"`
	Assets                      []UAsset    `json:"assets"`
	Positions                   []UPosition `json:"positions"`
}

// UAsset holds account asset information
type UAsset struct {
	Asset                  string  `json:"asset"`
	WalletBalance          float64 `json:"walletBalance,string"`
	UnrealizedProfit       float64 `json:"unrealizedProfit,string"`
	MarginBalance          float64 `json:"marginBalance,string"`
	MaintenanceMargin      float64 `json:"maintMargin,string"`
	InitialMargin          float64 `json:"initialMargin,string"`
	PositionInitialMargin  float64 `json:"positionInitialMargin,string"`
	OpenOrderInitialMargin float64 `json:"openOrderInitialMargin,string"`
	CrossWalletBalance     float64 `json:"crossWalletBalance,string"`
	CrossUnPnl             float64 `json:"crossUnPnl,string"`
	AvailableBalance       float64 `json:"availableBalance,string"`
	MaxWithdrawAmount      float64 `json:"maxWithdrawAmount,string"`
}

// UPosition holds account position information
type UPosition struct {
	Symbol                 string     `json:"symbol"`
	InitialMargin          float64    `json:"initialMargin,string"`
	MaintenanceMargin      float64    `json:"maintMargin,string"`
	UnrealisedProfit       float64    `json:"unrealizedProfit,string"`
	PositionInitialMargin  float64    `json:"positionInitialMargin,string"`
	OpenOrderInitialMargin float64    `json:"openOrderInitialMargin,string"`
	Leverage               float64    `json:"leverage,string"`
	Isolated               bool       `json:"isolated"`
	IsolatedWallet         float64    `json:"isolatedWallet,string"`
	EntryPrice             float64    `json:"entryPrice,string"`
	MaxNotional            float64    `json:"maxNotional,string"`
	BidNotional            float64    `json:"bidNotional,string"`
	AskNotional            float64    `json:"askNotional,string"`
	PositionSide           string     `json:"positionSide"`
	PositionAmount         float64    `json:"positionAmt,string"`
	UpdateTime             types.Time `json:"updateTime"`
}

// UChangeInitialLeverage stores leverage change data
type UChangeInitialLeverage struct {
	Leverage         int64   `json:"leverage"`
	MaxNotionalValue float64 `json:"maxNotionalValue,string"`
	Symbol           string  `json:"symbol"`
}

// UModifyIsolatedPosMargin stores modified isolated margin positions' data
type UModifyIsolatedPosMargin struct {
	Amount     float64 `json:"amount,string"`
	MarginType int64   `json:"type"`
}

// UPositionMarginChangeHistoryData gets position margin change history data
type UPositionMarginChangeHistoryData struct {
	Amount       float64    `json:"amount,string"`
	Asset        string     `json:"asset"`
	Symbol       string     `json:"symbol"`
	Time         types.Time `json:"time"`
	MarginType   int64      `json:"type"`
	PositionSide string     `json:"positionSide"`
}

// UPositionInformationV2 stores positions data
type UPositionInformationV2 struct {
	Symbol           string     `json:"symbol"`
	PositionAmount   float64    `json:"positionAmt,string"`
	EntryPrice       float64    `json:"entryPrice,string"`
	MarkPrice        float64    `json:"markPrice,string"`
	UnrealizedProfit float64    `json:"unrealizedProfit,string"`
	LiquidationPrice float64    `json:"liquidationPrice,string"`
	Leverage         float64    `json:"leverage,string"`
	MaxNotionalValue float64    `json:"maxNotionalValue,string"`
	MarginType       string     `json:"marginType"`
	IsAutoAddMargin  bool       `json:"isAutoAddMargin,string"`
	PositionSide     string     `json:"positionSide"`
	Notional         float64    `json:"notional,string"`
	IsolatedWallet   float64    `json:"isolatedWallet,string"`
	IsolatedMargin   float64    `json:"isolatedMargin,string"`
	UpdateTime       types.Time `json:"updateTime"`
}

// UAccountTradeHistory stores trade data for the users account
type UAccountTradeHistory struct {
	Buyer           bool       `json:"buyer"`
	Commission      float64    `json:"commission,string"`
	CommissionAsset string     `json:"commissionAsset"`
	ID              int64      `json:"id"`
	Maker           bool       `json:"maker"`
	OrderID         int64      `json:"orderId"`
	Price           float64    `json:"price,string"`
	Qty             float64    `json:"qty,string"`
	QuoteQty        float64    `json:"quoteQty"`
	RealizedPNL     float64    `json:"realizedPnl,string"`
	Side            string     `json:"side"`
	PositionSide    string     `json:"positionSide"`
	Symbol          string     `json:"symbol"`
	Time            types.Time `json:"time"`
}

// UAccountIncomeHistory stores income history data
type UAccountIncomeHistory struct {
	Symbol     string     `json:"symbol"`
	IncomeType string     `json:"incomeType"`
	Income     float64    `json:"income,string"`
	Asset      string     `json:"asset"`
	Info       string     `json:"info"`
	Time       types.Time `json:"time"`
	TranID     int64      `json:"tranId"`
	TradeID    string     `json:"tradeId"`
}

// UNotionalLeverageAndBrakcetsData stores notional and leverage brackets data for the account
type UNotionalLeverageAndBrakcetsData struct {
	Symbol   string `json:"symbol"`
	Brackets []struct {
		Bracket                int64   `json:"bracket"`
		InitialLeverage        float64 `json:"initialLeverage"`
		NotionalCap            float64 `json:"notionalCap"`
		NotionalFloor          float64 `json:"notionalFloor"`
		MaintenanceMarginRatio float64 `json:"maintMarginRatio"`
		Cumulative             float64 `json:"cum"`
	} `json:"brackets"`
}

// UPositionADLEstimationData stores ADL estimation data for a position
type UPositionADLEstimationData struct {
	Symbol      string `json:"symbol"`
	ADLQuantile struct {
		Long  int64 `json:"LONG"`
		Short int64 `json:"SHORT"`
		Hedge int64 `json:"HEDGE"`
	} `json:"adlQuantile"`
}

// UForceOrdersData stores liquidation orders data for the account
type UForceOrdersData struct {
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
	OrderType     string     `json:"type"`
	ReduceOnly    bool       `json:"reduceOnly"`
	ClosePosition bool       `json:"closePosition"`
	Side          string     `json:"side"`
	PositionSide  string     `json:"positionSide"`
	StopPrice     float64    `json:"stopPrice,string"`
	WorkingType   string     `json:"workingType"`
	PriceProtect  bool       `json:"priceProtect,string"`
	OrigType      string     `json:"origType"`
	Time          types.Time `json:"time"`
	UpdateTime    types.Time `json:"updateTime"`
}

// UFuturesNewOrderRequest stores order data for placing
type UFuturesNewOrderRequest struct {
	Symbol           currency.Pair `json:"symbol"`
	Side             string        `json:"side"`
	PositionSide     string        `json:"position_side"`
	OrderType        string        `json:"order_type"`
	TimeInForce      string        `json:"time_in_force"`
	NewClientOrderID string        `json:"new_client_order_id"`
	ClosePosition    string        `json:"close_position"`
	WorkingType      string        `json:"working_type"`
	NewOrderRespType string        `json:"new_order_resp_type"`
	Quantity         float64       `json:"quantity"`
	Price            float64       `json:"price"`
	StopPrice        float64       `json:"stop_price"`
	ActivationPrice  float64       `json:"activation_price"`
	CallbackRate     float64       `json:"callback_rate"`
	ReduceOnly       bool          `json:"reduce_only"`
}
