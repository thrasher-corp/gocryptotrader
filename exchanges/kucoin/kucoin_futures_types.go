package kucoin

import (
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

var validGranularity = []string{
	"1", "5", "15", "30", "60", "120", "240", "480", "720", "1440", "10080",
}

// Contract store contract details
type Contract struct {
	Symbol                  string       `json:"symbol"`
	RootSymbol              string       `json:"rootSymbol"`
	ContractType            string       `json:"type"`
	FirstOpenDate           types.Time   `json:"firstOpenDate"`
	ExpireDate              types.Time   `json:"expireDate"`
	SettleDate              types.Time   `json:"settleDate"`
	BaseCurrency            string       `json:"baseCurrency"`
	QuoteCurrency           string       `json:"quoteCurrency"`
	SettleCurrency          string       `json:"settleCurrency"`
	MaxOrderQty             float64      `json:"maxOrderQty"`
	MaxPrice                float64      `json:"maxPrice"`
	LotSize                 float64      `json:"lotSize"`
	TickSize                float64      `json:"tickSize"`
	IndexPriceTickSize      float64      `json:"indexPriceTickSize"`
	Multiplier              float64      `json:"multiplier"`
	InitialMargin           float64      `json:"initialMargin"`
	MaintainMargin          float64      `json:"maintainMargin"`
	MaxRiskLimit            float64      `json:"maxRiskLimit"`
	MinRiskLimit            float64      `json:"minRiskLimit"`
	RiskStep                float64      `json:"riskStep"`
	MakerFeeRate            float64      `json:"makerFeeRate"`
	TakerFeeRate            float64      `json:"takerFeeRate"`
	TakerFixFee             float64      `json:"takerFixFee"`
	MakerFixFee             float64      `json:"makerFixFee"`
	SettlementFee           float64      `json:"settlementFee"`
	IsDeleverage            bool         `json:"isDeleverage"`
	IsQuanto                bool         `json:"isQuanto"`
	IsInverse               bool         `json:"isInverse"`
	MarkMethod              string       `json:"markMethod"`
	FairMethod              string       `json:"fairMethod"`
	FundingBaseSymbol       string       `json:"fundingBaseSymbol"`
	FundingQuoteSymbol      string       `json:"fundingQuoteSymbol"`
	FundingRateSymbol       string       `json:"fundingRateSymbol"`
	IndexSymbol             string       `json:"indexSymbol"`
	SettlementSymbol        string       `json:"settlementSymbol"`
	Status                  string       `json:"status"`
	FundingFeeRate          float64      `json:"fundingFeeRate"`
	PredictedFundingFeeRate float64      `json:"predictedFundingFeeRate"`
	OpenInterest            types.Number `json:"openInterest"`
	TurnoverOf24h           float64      `json:"turnoverOf24h"`
	VolumeOf24h             float64      `json:"volumeOf24h"`
	MarkPrice               float64      `json:"markPrice"`
	IndexPrice              float64      `json:"indexPrice"`
	LastTradePrice          float64      `json:"lastTradePrice"`
	NextFundingRateTime     int64        `json:"nextFundingRateTime"`
	MaxLeverage             float64      `json:"maxLeverage"`
	SourceExchanges         []string     `json:"sourceExchanges"`
	PremiumsSymbol1M        string       `json:"premiumsSymbol1M"`
	PremiumsSymbol8H        string       `json:"premiumsSymbol8H"`
	FundingBaseSymbol1M     string       `json:"fundingBaseSymbol1M"`
	FundingQuoteSymbol1M    string       `json:"fundingQuoteSymbol1M"`
	LowPrice                float64      `json:"lowPrice"`
	HighPrice               float64      `json:"highPrice"`
	PriceChgPct             float64      `json:"priceChgPct"`
	PriceChg                float64      `json:"priceChg"`
}

// FuturesTicker stores ticker data
type FuturesTicker struct {
	Sequence     int64        `json:"sequence"`
	Symbol       string       `json:"symbol"`
	Side         order.Side   `json:"side"`
	Size         float64      `json:"size"`
	Price        types.Number `json:"price"`
	BestBidSize  float64      `json:"bestBidSize"`
	BestBidPrice types.Number `json:"bestBidPrice"`
	BestAskSize  float64      `json:"bestAskSize"`
	BestAskPrice types.Number `json:"bestAskPrice"`
	TradeID      string       `json:"tradeId"`
	FilledTime   types.Time   `json:"ts"`
}

type futuresOrderbookResponse struct {
	Asks     [][2]float64 `json:"asks"`
	Bids     [][2]float64 `json:"bids"`
	Time     types.Time   `json:"ts"`
	Sequence int64        `json:"sequence"`
	Symbol   string       `json:"symbol"`
}

// FuturesTrade stores trade data
type FuturesTrade struct {
	Sequence     int64      `json:"sequence"`
	TradeID      string     `json:"tradeId"`
	TakerOrderID string     `json:"takerOrderId"`
	MakerOrderID string     `json:"makerOrderId"`
	Price        float64    `json:"price,string"`
	Size         float64    `json:"size"`
	Side         string     `json:"side"`
	FilledTime   types.Time `json:"ts"`
}

// FuturesInterestRate stores interest rate data
type FuturesInterestRate struct {
	Symbol      string     `json:"symbol"`
	TimePoint   types.Time `json:"timePoint"`
	Value       float64    `json:"value"`
	Granularity int64      `json:"granularity"`
}

// Decomposition stores decomposition data
type Decomposition struct {
	Exchange string  `json:"exchange"`
	Price    float64 `json:"price"`
	Weight   float64 `json:"weight"`
}

// FuturesIndex stores index data
type FuturesIndex struct {
	FuturesInterestRate
	DecompositionList []Decomposition `json:"decompositionList"`
}

// FuturesMarkPrice stores mark price data
type FuturesMarkPrice struct {
	FuturesInterestRate
	IndexPrice float64 `json:"indexPrice"`
}

// FuturesFundingRate stores funding rate data
type FuturesFundingRate struct {
	FuturesInterestRate
	PredictedValue float64 `json:"predictedValue"`
}

// FundingHistoryItem represents funding history item
type FundingHistoryItem struct {
	Symbol      string     `json:"symbol"`
	FundingRate float64    `json:"fundingRate"`
	Timepoint   types.Time `json:"timepoint"`
}

// FuturesKline stores kline data
type FuturesKline struct {
	StartTime types.Time
	Open      float64
	Close     float64
	High      float64
	Low       float64
	Volume    float64
}

// UnmarshalJSON parses kline data from a JSON array into FuturesKline fields.
func (f *FuturesKline) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[6]any{&f.StartTime, &f.Open, &f.High, &f.Low, &f.Close, &f.Volume})
}

// FutureOrdersResponse represents a future order response list detail
type FutureOrdersResponse struct {
	CurrentPage int64          `json:"currentPage"`
	PageSize    int64          `json:"pageSize"`
	TotalNum    int64          `json:"totalNum"`
	TotalPage   int64          `json:"totalPage"`
	Items       []FuturesOrder `json:"items"`
}

// FuturesOrder represents futures order information
type FuturesOrder struct {
	ID             string     `json:"id"`
	Symbol         string     `json:"symbol"`
	OrderType      string     `json:"type"`
	Side           string     `json:"side"`
	Price          float64    `json:"price,string"`
	Size           float64    `json:"size"`
	Value          float64    `json:"value,string"`
	DealValue      float64    `json:"dealValue,string"`
	DealSize       float64    `json:"dealSize"`
	Stp            string     `json:"stp"`
	Stop           string     `json:"stop"`
	StopPriceType  string     `json:"stopPriceType"`
	StopTriggered  bool       `json:"stopTriggered"`
	StopPrice      float64    `json:"stopPrice,string"`
	TimeInForce    string     `json:"timeInForce"`
	PostOnly       bool       `json:"postOnly"`
	Hidden         bool       `json:"hidden"`
	Iceberg        bool       `json:"iceberg"`
	Leverage       float64    `json:"leverage,string"`
	ForceHold      bool       `json:"forceHold"`
	CloseOrder     bool       `json:"closeOrder"`
	VisibleSize    float64    `json:"visibleSize"`
	ClientOid      string     `json:"clientOid"`
	Remark         string     `json:"remark"`
	Tags           string     `json:"tags"`
	IsActive       bool       `json:"isActive"`
	CancelExist    bool       `json:"cancelExist"`
	CreatedAt      types.Time `json:"createdAt"`
	UpdatedAt      types.Time `json:"updatedAt"`
	EndAt          types.Time `json:"endAt"`
	OrderTime      types.Time `json:"orderTime"`
	SettleCurrency string     `json:"settleCurrency"`
	Status         string     `json:"status"`
	FilledValue    float64    `json:"filledValue,string"`
	FilledSize     float64    `json:"filledSize"`
	ReduceOnly     bool       `json:"reduceOnly"`
}

// FutureFillsResponse represents a future fills list response detail
type FutureFillsResponse struct {
	CurrentPage int64         `json:"currentPage"`
	PageSize    int64         `json:"pageSize"`
	TotalNum    int64         `json:"totalNum"`
	TotalPage   int64         `json:"totalPage"`
	Items       []FuturesFill `json:"items"`
}

// FuturesFill represents list of recent fills for futures orders
type FuturesFill struct {
	Symbol         string     `json:"symbol"`
	TradeID        string     `json:"tradeId"`
	OrderID        string     `json:"orderId"`
	Side           string     `json:"side"`
	Liquidity      string     `json:"liquidity"`
	ForceTaker     bool       `json:"forceTaker"`
	Price          float64    `json:"price,string"`
	Size           float64    `json:"size,string"`
	Value          float64    `json:"value,string"`
	FeeRate        float64    `json:"feeRate,string"`
	FixFee         float64    `json:"fixFee,string"`
	FeeCurrency    string     `json:"feeCurrency"`
	Stop           string     `json:"stop"`
	Fee            float64    `json:"fee,string"`
	OrderType      string     `json:"orderType"`
	TradeType      string     `json:"tradeType"`
	CreatedAt      types.Time `json:"createdAt"`
	SettleCurrency string     `json:"settleCurrency"`
	TradeTime      types.Time `json:"tradeTime"`
}

// FuturesOpenOrderStats represents futures open order summary stats information
type FuturesOpenOrderStats struct {
	OpenOrderBuySize  int64   `json:"openOrderBuySize"`
	OpenOrderSellSize int64   `json:"openOrderSellSize"`
	OpenOrderBuyCost  float64 `json:"openOrderBuyCost,string"`
	OpenOrderSellCost float64 `json:"openOrderSellCost,string"`
	SettleCurrency    string  `json:"settleCurrency"`
}

// FuturesPosition represents futures position detailed information
type FuturesPosition struct {
	ID                   string     `json:"id"`
	Symbol               string     `json:"symbol"`
	AutoDeposit          bool       `json:"autoDeposit"`
	MaintMarginReq       float64    `json:"maintMarginReq"`
	RiskLimit            int64      `json:"riskLimit"`
	RealLeverage         float64    `json:"realLeverage"`
	CrossMode            bool       `json:"crossMode"`
	ADLRankingPercentile float64    `json:"delevPercentage"`
	OpeningTimestamp     types.Time `json:"openingTimestamp"`
	CurrentTimestamp     types.Time `json:"currentTimestamp"`
	CurrentQty           float64    `json:"currentQty"`
	CurrentCost          float64    `json:"currentCost"` // Current position value
	CurrentComm          float64    `json:"currentComm"` // Current commission
	UnrealisedCost       float64    `json:"unrealisedCost"`
	RealisedGrossCost    float64    `json:"realisedGrossCost"`
	RealisedCost         float64    `json:"realisedCost"`
	IsOpen               bool       `json:"isOpen"`
	MarkPrice            float64    `json:"markPrice"`
	MarkValue            float64    `json:"markValue"`
	PosCost              float64    `json:"posCost"`   // Position value
	PosCross             float64    `json:"posCross"`  // Added margin
	PosInit              float64    `json:"posInit"`   // Leverage margin
	PosComm              float64    `json:"posComm"`   // Bankruptcy cost
	PosLoss              float64    `json:"posLoss"`   // Funding fees paid out
	PosMargin            float64    `json:"posMargin"` // Position margin
	PosMaint             float64    `json:"posMaint"`  // Maintenance margin
	MaintMargin          float64    `json:"maintMargin"`
	RealisedGrossPnl     float64    `json:"realisedGrossPnl"`
	RealisedPnl          float64    `json:"realisedPnl"`
	UnrealisedPnl        float64    `json:"unrealisedPnl"`
	UnrealisedPnlPcnt    float64    `json:"unrealisedPnlPcnt"`
	UnrealisedRoePcnt    float64    `json:"unrealisedRoePcnt"`
	AvgEntryPrice        float64    `json:"avgEntryPrice"`
	LiquidationPrice     float64    `json:"liquidationPrice"`
	BankruptPrice        float64    `json:"bankruptPrice"`
	SettleCurrency       string     `json:"settleCurrency"`
	MaintainMargin       float64    `json:"maintainMargin"`
	RiskLimitLevel       int64      `json:"riskLimitLevel"`
}

// WithdrawMarginResponse represents a response data after withdrawing a margin
type WithdrawMarginResponse struct {
	Symbol         string  `json:"symbol"`
	WithdrawAmount float64 `json:"withdrawAmount"`
}

// MarginRemovingResponse represents a response data for margin response
type MarginRemovingResponse struct {
	Symbol         string  `json:"symbol"`
	WithdrawAmount float64 `json:"withdrawAmount"`
}

// FuturesRiskLimitLevel represents futures risk limit level information
type FuturesRiskLimitLevel struct {
	Symbol         string  `json:"symbol"`
	Level          int64   `json:"level"`
	MaxRiskLimit   float64 `json:"maxRiskLimit"`
	MinRiskLimit   float64 `json:"minRiskLimit"`
	MaxLeverage    float64 `json:"maxLeverage"`
	InitialMargin  float64 `json:"initialMargin"`
	MaintainMargin float64 `json:"maintainMargin"`
}

// FuturesFundingHistory represents futures funding information
type FuturesFundingHistory struct {
	ID             string     `json:"id"`
	Symbol         string     `json:"symbol"`
	Time           types.Time `json:"timePoint"`
	FundingRate    float64    `json:"fundingRate"`
	MarkPrice      float64    `json:"markPrice"`
	PositionQty    float64    `json:"positionQty"`
	PositionCost   float64    `json:"positionCost"`
	Funding        float64    `json:"funding"`
	SettleCurrency string     `json:"settleCurrency"`
}

// FuturesAccount holds futures account detail information
type FuturesAccount struct {
	AccountEquity    float64 `json:"accountEquity"` // marginBalance + Unrealised PNL
	UnrealisedPNL    float64 `json:"unrealisedPNL"` // unrealised profit and loss
	MarginBalance    float64 `json:"marginBalance"` // positionMargin + orderMargin + frozenFunds + availableBalance - unrealisedPNL
	PositionMargin   float64 `json:"positionMargin"`
	OrderMargin      float64 `json:"orderMargin"`
	FrozenFunds      float64 `json:"frozenFunds"` // frozen funds for withdrawal and out-transfer
	AvailableBalance float64 `json:"availableBalance"`
	Currency         string  `json:"currency"`
}

// FuturesTransactionHistory represents a transaction history
type FuturesTransactionHistory struct {
	Time          types.Time `json:"time"`
	Type          string     `json:"type"`
	Amount        float64    `json:"amount"`
	Fee           float64    `json:"fee"`
	AccountEquity float64    `json:"accountEquity"`
	Status        string     `json:"status"`
	Remark        string     `json:"remark"`
	Offset        int64      `json:"offset"`
	Currency      string     `json:"currency"`
}

// APIKeyDetail represents the API key detail
type APIKeyDetail struct {
	SubName     string     `json:"subName"`
	Remark      string     `json:"remark"`
	APIKey      string     `json:"apiKey"`
	APISecret   string     `json:"apiSecret"`
	Passphrase  string     `json:"passphrase"`
	Permission  string     `json:"permission"`
	IPWhitelist string     `json:"ipWhitelist"`
	CreateAt    types.Time `json:"createdAt"`
}

// FuturesDepositDetailsResponse represents a futures deposits list detail response
type FuturesDepositDetailsResponse struct {
	CurrentPage int64                  `json:"currentPage"`
	PageSize    int64                  `json:"pageSize"`
	TotalNum    int64                  `json:"totalNum"`
	TotalPage   int64                  `json:"totalPage"`
	Items       []FuturesDepositDetail `json:"items"`
}

// FuturesDepositDetail represents futures deposit detail information
type FuturesDepositDetail struct {
	Currency   string     `json:"currency"`
	Status     string     `json:"status"`
	Address    string     `json:"address"`
	IsInner    bool       `json:"isInner"`
	Amount     float64    `json:"amount"`
	Fee        float64    `json:"fee"`
	WalletTxID string     `json:"walletTxId"`
	CreatedAt  types.Time `json:"createdAt"`
}

// FuturesWithdrawalLimit represents withdrawal limit information
type FuturesWithdrawalLimit struct {
	Currency            string  `json:"currency"`
	ChainID             string  `json:"chainId"`
	LimitAmount         float64 `json:"limitAmount"`
	UsedAmount          float64 `json:"usedAmount"`
	RemainAmount        float64 `json:"remainAmount"`
	AvailableAmount     float64 `json:"availableAmount"`
	WithdrawMinFee      float64 `json:"withdrawMinFee"`
	InnerWithdrawMinFee float64 `json:"innerWithdrawMinFee"`
	WithdrawMinSize     float64 `json:"withdrawMinSize"`
	IsWithdrawEnabled   bool    `json:"isWithdrawEnabled"`
	Precision           float64 `json:"precision"`
}

// FuturesWithdrawalsListResponse represents a list of futures Withdrawal history instance
type FuturesWithdrawalsListResponse struct {
	CurrentPage int64                      `json:"currentPage"`
	PageSize    int64                      `json:"pageSize"`
	TotalNum    int64                      `json:"totalNum"`
	TotalPage   int64                      `json:"totalPage"`
	Items       []FuturesWithdrawalHistory `json:"items"`
}

// FuturesWithdrawalHistory represents a list of Futures withdrawal history
type FuturesWithdrawalHistory struct {
	WithdrawalID string     `json:"withdrawalId"`
	Currency     string     `json:"currency"`
	Status       string     `json:"status"`
	Address      string     `json:"address"`
	Memo         string     `json:"memo"`
	IsInner      bool       `json:"isInner"`
	Amount       float64    `json:"amount"`
	Fee          float64    `json:"fee"`
	WalletTxID   string     `json:"walletTxId"`
	CreatedAt    types.Time `json:"createdAt"`
	Remark       string     `json:"remark"`
	Reason       string     `json:"reason"`
}

// TransferBase represents transfer base information
type TransferBase struct {
	ApplyID   string     `json:"applyId"`
	Currency  string     `json:"currency"`
	RecRemark string     `json:"recRemark"`
	RecSystem string     `json:"recSystem"`
	Status    string     `json:"status"`
	Amount    float64    `json:"amount,string"`
	Reason    string     `json:"reason"`
	CreatedAt types.Time `json:"createdAt"`
	Remark    string     `json:"remark"`
}

// TransferRes represents a transfer response
type TransferRes struct {
	TransferBase
	BizNo          string     `json:"bizNo"`
	PayAccountType string     `json:"payAccountType"`
	PayTag         string     `json:"payTag"`
	RecAccountType string     `json:"recAccountType"`
	RecTag         string     `json:"recTag"`
	Fee            float64    `json:"fee,string"`
	Serial         int64      `json:"sn"`
	UpdatedAt      types.Time `json:"updatedAt"`
}

// TransferListsResponse represents a transfer lists detail
type TransferListsResponse struct {
	CurrentPage int64      `json:"currentPage"`
	PageSize    int64      `json:"pageSize"`
	TotalNum    int64      `json:"totalNum"`
	TotalPage   int64      `json:"totalPage"`
	Items       []Transfer `json:"items"`
}

// Transfer represents a transfer detail
type Transfer struct {
	TransferBase
	Offset int64 `json:"offset"`
}

// FuturesServiceStatus represents service status
type FuturesServiceStatus struct {
	Status  string `json:"status"`
	Message string `json:"msg"`
}
