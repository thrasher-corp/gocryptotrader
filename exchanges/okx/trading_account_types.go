package okx

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Position modes for placing perpetual swap orders
const positionModeNet = "net_mode"

const positionModeLongShort = "long_short_mode"

// AccountInstrumentsRequest holds the parameters for fetching the instruments available to the account
type AccountInstrumentsRequest struct {
	InstrumentType   string // Required: SPOT, MARGIN, SWAP, FUTURES, OPTION or EVENTS
	SeriesID         string // Required for EVENTS, e.g. BTC-ABOVE-DAILY
	InstrumentFamily string // Required for OPTION; only applicable to FUTURES, SWAP and OPTION
	InstrumentID     string
}

// AccountInstrumentResponse represents an instrument available to the account
type AccountInstrumentResponse struct {
	InstrumentType                   string                     `json:"instType"`
	SeriesID                         string                     `json:"seriesId"`
	InstrumentID                     string                     `json:"instId"`
	Underlying                       string                     `json:"uly"`
	TradingFeeGroupID                string                     `json:"groupId"`
	InstrumentFamily                 string                     `json:"instFamily"`
	BaseCurrency                     currency.Code              `json:"baseCcy"`
	QuoteCurrency                    currency.Code              `json:"quoteCcy"`
	SettlementCurrency               currency.Code              `json:"settleCcy"`
	ContractValue                    types.Number               `json:"ctVal"`
	ContractMultiplier               types.Number               `json:"ctMult"`
	ContractValueCurrency            currency.Code              `json:"ctValCcy"`
	OptionType                       string                     `json:"optType"`
	StrikePrice                      types.Number               `json:"stk"`
	ListTime                         types.Time                 `json:"listTime"`
	AuctionEndTime                   types.Time                 `json:"auctionEndTime"`
	ContinuousTradingSwitchTime      types.Time                 `json:"contTdSwTime"`
	PreMarketSwitchTime              types.Time                 `json:"preMktSwTime"`
	OpenType                         string                     `json:"openType"`
	RPIPermission                    string                     `json:"rpi"`
	ExpiryTime                       types.Time                 `json:"expTime"`
	MaxLeverage                      types.Number               `json:"lever"`
	TickSize                         types.Number               `json:"tickSz"`
	LotSize                          types.Number               `json:"lotSz"`
	MinimumOrderSize                 types.Number               `json:"minSz"`
	ContractType                     string                     `json:"ctType"`
	State                            string                     `json:"state"`
	RuleType                         string                     `json:"ruleType"`
	PositionLimitAmount              types.Number               `json:"posLmtAmt"`
	PositionLimitPercent             types.Number               `json:"posLmtPct"`
	MaxPlatformOpenInterestLimit     types.Number               `json:"maxPlatOILmt"`
	MaxPlatformOpenInterestCoinLimit types.Number               `json:"maxPlatOICoinLmt"`
	LongPositionRemainingQuota       types.Number               `json:"longPosRemainingQuota"`
	ShortPositionRemainingQuota      types.Number               `json:"shortPosRemainingQuota"`
	MaxLimitOrderSize                types.Number               `json:"maxLmtSz"`
	MaxMarketOrderSize               types.Number               `json:"maxMktSz"`
	MaxLimitOrderAmountUSD           types.Number               `json:"maxLmtAmt"`
	MaxMarketOrderAmountUSD          types.Number               `json:"maxMktAmt"`
	MaxTWAPOrderSize                 types.Number               `json:"maxTwapSz"`
	MaxIcebergOrderSize              types.Number               `json:"maxIcebergSz"`
	MaxTriggerOrderSize              types.Number               `json:"maxTriggerSz"`
	MaxStopOrderSize                 types.Number               `json:"maxStopSz"`
	FutureSettlement                 bool                       `json:"futureSettlement"`
	TradeQuoteCurrencies             []currency.Code            `json:"tradeQuoteCcyList"`
	InstrumentIDCode                 uint64                     `json:"instIdCode"`
	InstrumentCategory               string                     `json:"instCategory"`
	InitialPriceLimitPercent         types.Number               `json:"initPxLmtPct"`
	FloatingPriceLimitPercent        types.Number               `json:"floatPxLmtPct"`
	MaxPriceLimitPercent             types.Number               `json:"maxPxLmtPct"`
	UpcomingChanges                  []InstrumentUpcomingChange `json:"upcChg"`
}

// AccountBalanceRequest holds the parameters for fetching the trading account balance
type AccountBalanceRequest struct {
	Currencies []currency.Code // At most 20; all currencies with a balance when empty
}

// PositionsRequest holds the parameters for fetching positions
type PositionsRequest struct {
	InstrumentType string   // MARGIN, SWAP, FUTURES, OPTION or EVENTS
	InstrumentIDs  []string // At most 10
	PositionIDs    []string // At most 20
}

// AccountPositionResponse represents a position in the trading account
type AccountPositionResponse struct {
	InstrumentType                        string                   `json:"instType"`
	MarginMode                            string                   `json:"mgnMode"`
	PositionID                            string                   `json:"posId"`
	PositionSide                          string                   `json:"posSide"`
	QuantityOfPosition                    types.Number             `json:"pos"`
	HedgedPosition                        types.Number             `json:"hedgedPos"`
	BaseBalance                           types.Number             `json:"baseBal"`
	QuoteBalance                          types.Number             `json:"quoteBal"`
	BaseBorrowed                          types.Number             `json:"baseBorrowed"`
	BaseInterest                          types.Number             `json:"baseInterest"`
	QuoteBorrowed                         types.Number             `json:"quoteBorrowed"`
	QuoteInterest                         types.Number             `json:"quoteInterest"`
	PositionCurrency                      currency.Code            `json:"posCcy"`
	AvailablePosition                     types.Number             `json:"availPos"`
	AveragePrice                          types.Number             `json:"avgPx"`
	NonSettlementAveragePrice             types.Number             `json:"nonSettleAvgPx"`
	MarkPrice                             types.Number             `json:"markPx"`
	UnrealisedProfitAndLoss               types.Number             `json:"upl"`
	UnrealisedProfitAndLossRatio          types.Number             `json:"uplRatio"`
	UnrealisedProfitAndLossLastPrice      types.Number             `json:"uplLastPx"`
	UnrealisedProfitAndLossRatioLastPrice types.Number             `json:"uplRatioLastPx"`
	InstrumentID                          string                   `json:"instId"`
	Leverage                              types.Number             `json:"lever"`
	LiquidationPrice                      types.Number             `json:"liqPx"`
	InitialMarginRequirement              types.Number             `json:"imr"`
	Margin                                types.Number             `json:"margin"`
	MarginRatio                           types.Number             `json:"mgnRatio"`
	MaintenanceMarginRequirement          types.Number             `json:"mmr"`
	Liabilities                           types.Number             `json:"liab"`
	LiabilitiesCurrency                   currency.Code            `json:"liabCcy"`
	Interest                              types.Number             `json:"interest"`
	TradeID                               string                   `json:"tradeId"`
	OptionValue                           types.Number             `json:"optVal"`
	PendingCloseOrderLiability            types.Number             `json:"pendingCloseOrdLiabVal"`
	NotionalUSD                           types.Number             `json:"notionalUsd"`
	AutoDeleveraging                      string                   `json:"adl"`
	Currency                              currency.Code            `json:"ccy"`
	LastTradePrice                        types.Number             `json:"last"`
	IndexPrice                            types.Number             `json:"idxPx"`
	USDPrice                              types.Number             `json:"usdPx"`
	BreakEvenPrice                        types.Number             `json:"bePx"`
	DeltaBS                               types.Number             `json:"deltaBS"`
	DeltaPA                               types.Number             `json:"deltaPA"`
	GammaBS                               types.Number             `json:"gammaBS"`
	GammaPA                               types.Number             `json:"gammaPA"`
	ThetaBS                               types.Number             `json:"thetaBS"`
	ThetaPA                               types.Number             `json:"thetaPA"`
	VegaBS                                types.Number             `json:"vegaBS"`
	VegaPA                                types.Number             `json:"vegaPA"`
	SpotInUseAmount                       types.Number             `json:"spotInUseAmt"`
	SpotInUseCurrency                     currency.Code            `json:"spotInUseCcy"`
	ClientSpotInUseAmount                 types.Number             `json:"clSpotInUseAmt"`
	MaxSpotInUseAmount                    types.Number             `json:"maxSpotInUseAmt"`
	BusinessReferenceID                   string                   `json:"bizRefId"`
	BusinessReferenceType                 string                   `json:"bizRefType"`
	RealisedProfitAndLoss                 types.Number             `json:"realizedPnl"`
	SettledProfitAndLoss                  types.Number             `json:"settledPnl"`
	ProfitAndLoss                         types.Number             `json:"pnl"`
	Fee                                   types.Number             `json:"fee"`
	FundingFee                            types.Number             `json:"fundingFee"`
	LiquidationPenalty                    types.Number             `json:"liqPenalty"`
	CloseOrderAlgo                        []PositionCloseOrderAlgo `json:"closeOrderAlgo"`
	CreationTime                          types.Time               `json:"cTime"`
	UpdateTime                            types.Time               `json:"uTime"`
	PushTime                              types.Time               `json:"pTime"` // Sent by the positions websocket channel only
}

// PositionCloseOrderAlgo represents a close position algo order attached to a position
type PositionCloseOrderAlgo struct {
	AlgoID                     string       `json:"algoId"`
	StopLossTriggerPrice       types.Number `json:"slTriggerPx"`
	StopLossTriggerPriceType   string       `json:"slTriggerPxType"`
	TakeProfitTriggerPrice     types.Number `json:"tpTriggerPx"`
	TakeProfitTriggerPriceType string       `json:"tpTriggerPxType"`
	CloseFraction              types.Number `json:"closeFraction"`
}

// PositionsHistoryRequest holds the parameters for fetching position history
type PositionsHistoryRequest struct {
	InstrumentType string // MARGIN, SWAP, FUTURES, OPTION or EVENTS
	InstrumentID   string
	MarginMode     string // cross or isolated
	CloseType      string // Latest close position type: 1 partial, 2 all, 3 liquidation, 4 partial liquidation, 5 ADL not fully closed, 6 ADL fully closed
	PositionID     string
	After          time.Time // Returns records updated earlier than this time
	Before         time.Time // Returns records updated later than this time
	Limit          uint64    // At most 100
}

// PositionHistoryResponse represents a historical position
type PositionHistoryResponse struct {
	InstrumentType            string        `json:"instType"`
	InstrumentID              string        `json:"instId"`
	MarginMode                string        `json:"mgnMode"`
	CloseType                 string        `json:"type"`
	CreationTime              types.Time    `json:"cTime"`
	UpdateTime                types.Time    `json:"uTime"`
	OpenAveragePrice          types.Number  `json:"openAvgPx"`
	NonSettlementAveragePrice types.Number  `json:"nonSettleAvgPx"`
	CloseAveragePrice         types.Number  `json:"closeAvgPx"`
	PositionID                string        `json:"posId"`
	OpenMaxPosition           types.Number  `json:"openMaxPos"`
	CloseTotalPosition        types.Number  `json:"closeTotalPos"`
	RealisedProfitAndLoss     types.Number  `json:"realizedPnl"`
	SettledProfitAndLoss      types.Number  `json:"settledPnl"`
	ProfitAndLossRatio        types.Number  `json:"pnlRatio"`
	Fee                       types.Number  `json:"fee"`
	FundingFee                types.Number  `json:"fundingFee"`
	LiquidationPenalty        types.Number  `json:"liqPenalty"`
	ProfitAndLoss             types.Number  `json:"pnl"`
	PositionSide              string        `json:"posSide"`
	Leverage                  types.Number  `json:"lever"`
	Direction                 string        `json:"direction"`
	TriggerPrice              types.Number  `json:"triggerPx"`
	Underlying                string        `json:"uly"`
	Currency                  currency.Code `json:"ccy"`
}

// AccountPositionRiskRequest holds the parameters for fetching account and position risk
type AccountPositionRiskRequest struct {
	InstrumentType string // MARGIN, SWAP, FUTURES or OPTION
}

// AccountPositionRiskResponse represents account and position risk taken at the same time
type AccountPositionRiskResponse struct {
	Timestamp      types.Time             `json:"ts"`
	AdjustedEquity types.Number           `json:"adjEq"`
	Balances       []PositionRiskBalance  `json:"balData"`
	Positions      []PositionRiskPosition `json:"posData"`
}

// PositionRiskBalance represents a currency balance in an account and position risk snapshot
type PositionRiskBalance struct {
	Currency       currency.Code `json:"ccy"`
	Equity         types.Number  `json:"eq"`
	DiscountEquity types.Number  `json:"disEq"`
}

// PositionRiskPosition represents a position in an account and position risk snapshot
type PositionRiskPosition struct {
	InstrumentType   string        `json:"instType"`
	MarginMode       string        `json:"mgnMode"`
	PositionID       string        `json:"posId"`
	InstrumentID     string        `json:"instId"`
	Position         types.Number  `json:"pos"`
	BaseBalance      types.Number  `json:"baseBal"`
	QuoteBalance     types.Number  `json:"quoteBal"`
	PositionSide     string        `json:"posSide"`
	PositionCurrency currency.Code `json:"posCcy"`
	Currency         currency.Code `json:"ccy"`
	NotionalCoin     types.Number  `json:"notionalCcy"`
	NotionalUSD      types.Number  `json:"notionalUsd"`
}

// BillsDetailRequest holds the parameters for fetching account bills
type BillsDetailRequest struct {
	InstrumentType string // SPOT, MARGIN, SWAP, FUTURES, OPTION or EVENTS
	InstrumentID   string
	Currency       currency.Code
	MarginMode     string // isolated or cross
	ContractType   string // linear or inverse, only applicable to FUTURES and SWAP
	BillType       uint64 // See GetBillTypes
	BillSubType    uint64 // See GetBillTypes
	After          string // Bill ID; returns records earlier than this bill
	Before         string // Bill ID; returns records newer than this bill
	BeginTime      time.Time
	EndTime        time.Time
	Limit          uint64 // At most 100
}

// BillsDetailResponse represents an account bill
type BillsDetailResponse struct {
	InstrumentType             string        `json:"instType"`
	BillID                     string        `json:"billId"`
	Type                       string        `json:"type"`
	SubType                    string        `json:"subType"`
	Timestamp                  types.Time    `json:"ts"`
	BalanceChange              types.Number  `json:"balChg"`
	PositionLevelBalanceChange types.Number  `json:"posBalChg"`
	Balance                    types.Number  `json:"bal"`
	PositionLevelBalance       types.Number  `json:"posBal"`
	Size                       types.Number  `json:"sz"`
	Price                      types.Number  `json:"px"`
	Currency                   currency.Code `json:"ccy"`
	ProfitAndLoss              types.Number  `json:"pnl"`
	Fee                        types.Number  `json:"fee"`
	EarnAmount                 types.Number  `json:"earnAmt"`
	EarnAPR                    types.Number  `json:"earnApr"`
	MarginMode                 string        `json:"mgnMode"`
	InstrumentID               string        `json:"instId"`
	OrderID                    string        `json:"ordId"`
	ExecutionType              string        `json:"execType"`
	From                       string        `json:"from"`
	To                         string        `json:"to"`
	Notes                      string        `json:"notes"`
	Interest                   types.Number  `json:"interest"`
	Tag                        string        `json:"tag"`
	FillTime                   types.Time    `json:"fillTime"`
	TradeID                    string        `json:"tradeId"`
	ClientOrderID              string        `json:"clOrdId"`
	FillIndexPrice             types.Number  `json:"fillIdxPx"`
	FillMarkPrice              types.Number  `json:"fillMarkPx"`
	FillPriceVolatility        types.Number  `json:"fillPxVol"`
	FillPriceUSD               types.Number  `json:"fillPxUsd"`
	FillMarkVolatility         types.Number  `json:"fillMarkVol"`
	FillForwardPrice           types.Number  `json:"fillFwdPx"`
}

// BillsHistoryArchiveRequest holds the parameters for applying for, and fetching, the bills archive of a quarter
type BillsHistoryArchiveRequest struct {
	Year      uint64 `json:"year,string"`
	Quarter   string `json:"quarter"`        // Q1, Q2, Q3 or Q4
	BillTypes string `json:"type,omitempty"` // Comma separated bill types, e.g. 1,2,3; all types when empty
}

// ApplyBillDetailsResponse represents the result of applying for a bills archive
type ApplyBillDetailsResponse struct {
	Result    types.Boolean `json:"result"` // Whether a download link already exists
	Timestamp types.Time    `json:"ts"`
}

// BillsHistoryArchiveResponse represents a bills archive download link
type BillsHistoryArchiveResponse struct {
	FileHref  string     `json:"fileHref"`
	Timestamp types.Time `json:"ts"`
	State     string     `json:"state"`
}

// BillTypesRequest holds the parameters for fetching bill types
type BillTypesRequest struct {
	BillTypes string // Comma separated bill types, e.g. 1,2,3; all types when empty
}

// BillTypeResponse represents a bill type and its sub-types
type BillTypeResponse struct {
	BillType            string              `json:"type"`
	BillTypeDescription string              `json:"typeDesc"`
	SubTypeDetails      []BillSubTypeDetail `json:"subTypeDetails"`
}

// BillSubTypeDetail represents a bill sub-type
type BillSubTypeDetail struct {
	SubType            string `json:"subType"`
	SubTypeDescription string `json:"subTypeDesc"`
}

// AccountConfigurationResponse represents the account configuration
type AccountConfigurationResponse struct {
	UserID                         string          `json:"uid"`
	MainUserID                     string          `json:"mainUid"`
	AccountLevel                   types.Number    `json:"acctLv"` // 1: Spot mode, 2: Futures mode, 3: Multi-currency margin, 4: Portfolio margin
	AccountSelfTradePreventionMode string          `json:"acctStpMode"`
	PositionMode                   string          `json:"posMode"`
	AutoLoan                       bool            `json:"autoLoan"`
	GreeksType                     string          `json:"greeksType"`
	FeeType                        string          `json:"feeType"`
	Level                          string          `json:"level"`
	LevelTemporary                 string          `json:"levelTmp"`
	ContractIsolatedMode           string          `json:"ctIsoMode"`
	MarginIsolatedMode             string          `json:"mgnIsoMode"`
	SpotOffsetType                 string          `json:"spotOffsetType"`
	StrategyType                   string          `json:"stgyType"`
	RoleType                       types.Number    `json:"roleType"`
	TraderInstruments              []string        `json:"traderInsts"`
	SpotRoleType                   types.Number    `json:"spotRoleType"`
	SpotTraderInstruments          []string        `json:"spotTraderInsts"`
	OptionTradingAuthorisation     types.Number    `json:"opAuth"` // 0: not activated, 1: activated
	KYCLevel                       types.Number    `json:"kycLv"`
	Label                          string          `json:"label"`
	IPAddresses                    string          `json:"ip"`
	Permissions                    string          `json:"perm"`
	LiquidationGear                types.Number    `json:"liquidationGear"`
	SpotBorrowEnabled              bool            `json:"enableSpotBorrow"`
	SpotBorrowAutoRepay            bool            `json:"spotBorrowAutoRepay"`
	AccountType                    types.Number    `json:"type"`
	SettlementCurrency             currency.Code   `json:"settleCcy"`
	SettlementCurrencies           []currency.Code `json:"settleCcyList"`
}

// SetPositionModeRequest is the request body for setting the position mode
type SetPositionModeRequest struct {
	PositionMode string `json:"posMode"`
}

// PositionModeResponse represents the position mode set
type PositionModeResponse struct {
	PositionMode string `json:"posMode"`
}

// SetLeverageRequest holds the parameters for setting leverage
type SetLeverageRequest struct {
	InstrumentID string        `json:"instId,omitempty"`
	Currency     currency.Code `json:"ccy,omitzero"`
	Leverage     types.Number  `json:"lever"`
	MarginMode   string        `json:"mgnMode"`           // cross or isolated
	PositionSide string        `json:"posSide,omitempty"` // long or short; only required for isolated FUTURES and SWAP in long/short mode
}

// SetLeverageResponse represents the leverage set
type SetLeverageResponse struct {
	Leverage     types.Number `json:"lever"`
	MarginMode   string       `json:"mgnMode"`
	InstrumentID string       `json:"instId"`
	PositionSide string       `json:"posSide"`
}

// MaximumOrderQuantityRequest holds the parameters for fetching the maximum order quantity
type MaximumOrderQuantityRequest struct {
	InstrumentIDs      []string // Required; at most 5 of one instrument type
	TradeMode          string   // Required: cross, isolated, cash or spot_isolated
	Currency           currency.Code
	Price              float64 // Ignored by OKX when several instruments are requested
	Leverage           float64
	TradeQuoteCurrency currency.Code // Only applicable to SPOT
	Outcome            string        // yes or no, only applicable to EVENTS
}

// MaximumOrderQuantityResponse represents the maximum quantity to buy or sell
type MaximumOrderQuantityResponse struct {
	InstrumentID string        `json:"instId"`
	Currency     currency.Code `json:"ccy"`
	MaximumBuy   types.Number  `json:"maxBuy"`
	MaximumSell  types.Number  `json:"maxSell"`
}

// MaximumAvailableBalanceRequest holds the parameters for fetching the maximum available balance or equity
type MaximumAvailableBalanceRequest struct {
	InstrumentIDs      []string // Required; at most 5
	Currency           currency.Code
	TradeMode          string // Required: cross, isolated, cash or spot_isolated
	ReduceOnly         bool   // Only applicable to MARGIN
	Price              float64
	TradeQuoteCurrency currency.Code // Only applicable to SPOT
}

// MaximumAvailableBalanceResponse represents the maximum available balance or equity
type MaximumAvailableBalanceResponse struct {
	InstrumentID  string       `json:"instId"`
	AvailableBuy  types.Number `json:"availBuy"`
	AvailableSell types.Number `json:"availSell"`
}

// IncreaseDecreaseMarginRequest holds the parameters for adjusting the margin of an isolated position
type IncreaseDecreaseMarginRequest struct {
	InstrumentID      string        `json:"instId"`
	PositionSide      string        `json:"posSide"` // long, short or net
	MarginBalanceType string        `json:"type"`    // add or reduce
	Amount            types.Number  `json:"amt"`
	Currency          currency.Code `json:"ccy,omitzero"` // Only applicable to isolated MARGIN
}

// IncreaseDecreaseMarginResponse represents the margin adjustment
type IncreaseDecreaseMarginResponse struct {
	InstrumentID string        `json:"instId"`
	PositionSide string        `json:"posSide"`
	Amount       types.Number  `json:"amt"`
	Type         string        `json:"type"`
	Leverage     types.Number  `json:"leverage"`
	Currency     currency.Code `json:"ccy"`
}

// LeverageInfoRequest holds the parameters for fetching leverage
type LeverageInfoRequest struct {
	InstrumentIDs []string        // At most 20; either instrument IDs or currencies are required
	Currencies    []currency.Code // At most 20; leverage at the currency level for cross MARGIN
	MarginMode    string          // Required: cross or isolated
}

// LeverageResponse represents the leverage of an instrument or currency
type LeverageResponse struct {
	InstrumentID string        `json:"instId"`
	Currency     currency.Code `json:"ccy"`
	MarginMode   string        `json:"mgnMode"`
	PositionSide string        `json:"posSide"`
	Leverage     types.Number  `json:"lever"`
}

// LeverageEstimatedInfoRequest holds the parameters for fetching leverage estimated information
type LeverageEstimatedInfoRequest struct {
	InstrumentType string  // Required: MARGIN, SWAP or FUTURES
	MarginMode     string  // Required: isolated or cross
	Leverage       float64 // Required
	InstrumentID   string  // Required for SWAP and FUTURES
	Currency       currency.Code
	PositionSide   string // net, long or short
}

// LeverageEstimatedInfoResponse represents estimated information for a leverage
type LeverageEstimatedInfoResponse struct {
	EstimatedAvailableQuoteTransfer types.Number `json:"estAvailQuoteTrans"`
	EstimatedAvailableTransfer      types.Number `json:"estAvailTrans"`
	EstimatedLiquidationPrice       types.Number `json:"estLiqPx"`
	EstimatedMargin                 types.Number `json:"estMgn"`
	EstimatedQuoteMargin            types.Number `json:"estQuoteMgn"`
	EstimatedMaxAmount              types.Number `json:"estMaxAmt"`
	EstimatedQuoteMaxAmount         types.Number `json:"estQuoteMaxAmt"`
	ExistingOrders                  bool         `json:"existOrd"`
	MaxLeverage                     types.Number `json:"maxLever"`
	MinLeverage                     types.Number `json:"minLever"`
}

// MaximumLoanRequest holds the parameters for fetching the maximum loan of an instrument or currency
type MaximumLoanRequest struct {
	MarginMode         string        // Required: isolated or cross
	InstrumentIDs      []string      // At most 5; either instrument IDs or a currency are required
	Currency           currency.Code // Maximum manual borrow in Spot mode
	MarginCurrency     currency.Code
	TradeQuoteCurrency currency.Code // Only applicable to SPOT
}

// MaximumLoanResponse represents the maximum loan of an instrument or currency
type MaximumLoanResponse struct {
	InstrumentID   string        `json:"instId"`
	MarginMode     string        `json:"mgnMode"`
	MarginCurrency currency.Code `json:"mgnCcy"`
	MaxLoan        types.Number  `json:"maxLoan"`
	Currency       currency.Code `json:"ccy"`
	Side           string        `json:"side"`
}

// TradeFeeRequest holds the parameters for fetching fee rates
type TradeFeeRequest struct {
	InstrumentType   string // Required: SPOT, MARGIN, SWAP, FUTURES, OPTION or EVENTS
	InstrumentID     string // Applicable to SPOT and MARGIN
	InstrumentFamily string // Applicable to FUTURES, SWAP and OPTION
	GroupID          string // Trading fee group ID; cannot be combined with an instrument ID or family
}

// InterestAccruedRequest holds the parameters for fetching accrued interest
type InterestAccruedRequest struct {
	LoanType     string // 2: market loans, the default
	Currency     currency.Code
	InstrumentID string
	MarginMode   string    // cross or isolated
	After        time.Time // Returns records earlier than this time
	Before       time.Time // Returns records newer than this time
	Limit        uint64    // At most 100
}

// InterestAccruedResponse represents accrued interest
type InterestAccruedResponse struct {
	LoanType              string        `json:"type"`
	Currency              currency.Code `json:"ccy"`
	InstrumentID          string        `json:"instId"`
	MarginMode            string        `json:"mgnMode"`
	Interest              types.Number  `json:"interest"`
	InterestRate          types.Number  `json:"interestRate"` // Hourly borrowing interest rate
	Liability             types.Number  `json:"liab"`
	TotalLiability        types.Number  `json:"totalLiab"`
	InterestFreeLiability types.Number  `json:"interestFreeLiab"`
	Timestamp             types.Time    `json:"ts"`
}

// InterestRateRequest holds the parameters for fetching borrowing interest rates
type InterestRateRequest struct {
	Currency currency.Code
}

// InterestRateResponse represents a currency's hourly borrowing interest rate
type InterestRateResponse struct {
	InterestRate types.Number  `json:"interestRate"`
	Currency     currency.Code `json:"ccy"`
}

// SetFeeTypeRequest is the request body for setting the fee type
type SetFeeTypeRequest struct {
	FeeType string `json:"feeType"`
}

// FeeTypeResponse represents the fee type set
type FeeTypeResponse struct {
	FeeType string `json:"feeType"` // 0: charged in the currency received, 1: always charged in the quote currency
}

// SetGreeksRequest is the request body for setting the greeks display type
type SetGreeksRequest struct {
	GreeksType string `json:"greeksType"`
}

// GreeksTypeResponse represents the greeks display type set
type GreeksTypeResponse struct {
	GreeksType string `json:"greeksType"` // PA: greeks in coins, BS: Black-Scholes greeks in dollars
}

// IsolatedMarginTradingSettingsRequest holds the parameters for setting the isolated margin trading mode
type IsolatedMarginTradingSettingsRequest struct {
	IsolatedMode   string `json:"isoMode"` // auto_transfers_ccy (MARGIN only) or automatic
	InstrumentType string `json:"type"`    // MARGIN or CONTRACTS
}

// IsolatedMarginTradingSettingsResponse represents the isolated margin trading mode set
type IsolatedMarginTradingSettingsResponse struct {
	IsolatedMode string `json:"isoMode"`
}

// MaximumWithdrawalsRequest holds the parameters for fetching maximum withdrawals
type MaximumWithdrawalsRequest struct {
	Currencies []currency.Code // At most 20; every owned currency when empty
}

// MaximumWithdrawalResponse represents the maximum transferable amount from the trading account to the funding account
type MaximumWithdrawalResponse struct {
	Currency                                 currency.Code `json:"ccy"`
	MaxWithdrawal                            types.Number  `json:"maxWd"`
	MaxWithdrawalIncludingBorrowed           types.Number  `json:"maxWdEx"`
	SpotOffsetMaxWithdrawal                  types.Number  `json:"spotOffsetMaxWd"`
	SpotOffsetMaxWithdrawalIncludingBorrowed types.Number  `json:"spotOffsetMaxWdEx"`
}

// AccountRiskStateResponse represents the account risk state of a Portfolio margin account
type AccountRiskStateResponse struct {
	AtRisk                 bool       `json:"atRisk"`
	AtRiskDerivativesUnits []string   `json:"atRiskIdx"`
	AtRiskMarginUnits      []string   `json:"atRiskMgn"`
	Timestamp              types.Time `json:"ts"`
}

// BorrowInterestAndLimitRequest holds the parameters for fetching borrow interest and limits
type BorrowInterestAndLimitRequest struct {
	LoanType string // 2: market loans, the default
	Currency currency.Code
}

// BorrowInterestAndLimitResponse represents borrow interest and limits
type BorrowInterestAndLimitResponse struct {
	Debt             types.Number                   `json:"debt"`
	Interest         types.Number                   `json:"interest"`
	NextDiscountTime types.Time                     `json:"nextDiscountTime"`
	NextInterestTime types.Time                     `json:"nextInterestTime"`
	LoanAllocation   types.Number                   `json:"loanAlloc"`
	Records          []BorrowInterestAndLimitRecord `json:"records"`
}

// BorrowInterestAndLimitRecord represents the borrow interest and limit of a currency
type BorrowInterestAndLimitRecord struct {
	Currency                 currency.Code      `json:"ccy"`
	Rate                     types.Number       `json:"rate"`
	LoanQuota                types.Number       `json:"loanQuota"`
	SurplusLimit             types.Number       `json:"surplusLmt"`
	UsedLimit                types.Number       `json:"usedLmt"`
	Interest                 types.Number       `json:"interest"`
	InterestFreeLiability    types.Number       `json:"interestFreeLiab"`
	PotentialBorrowingAmount types.Number       `json:"potentialBorrowingAmt"`
	SurplusLimitDetails      SurplusLimitDetail `json:"surplusLmtDetails"`
	PositionLoan             types.Number       `json:"posLoan"`
	AvailableLoan            types.Number       `json:"availLoan"`
	UsedLoan                 types.Number       `json:"usedLoan"`
	AverageRate              types.Number       `json:"avgRate"`
}

// SurplusLimitDetail represents the available amount across all sub-accounts
type SurplusLimitDetail struct {
	AllAccountsRemainingQuota    types.Number `json:"allAcctRemainingQuota"`
	CurrentAccountRemainingQuota types.Number `json:"curAcctRemainingQuota"`
	PlatformRemainingQuota       types.Number `json:"platRemainingQuota"`
}

// ManualBorrowRepayRequest holds the parameters for manually borrowing or repaying in Spot mode
type ManualBorrowRepayRequest struct {
	Currency currency.Code `json:"ccy"`
	Side     string        `json:"side"` // borrow or repay
	Amount   types.Number  `json:"amt"`
}

// ManualBorrowRepayResponse represents a manual borrow or repayment
type ManualBorrowRepayResponse struct {
	Currency currency.Code `json:"ccy"`
	Side     string        `json:"side"`
	Amount   types.Number  `json:"amt"`
}

// SetAutoRepayRequest is the request body for setting auto repay
type SetAutoRepayRequest struct {
	AutoRepay bool `json:"autoRepay"`
}

// AutoRepayResponse represents the auto repay setting
type AutoRepayResponse struct {
	AutoRepay bool `json:"autoRepay"`
}

// BorrowRepayHistoryRequest holds the parameters for fetching the borrow and repay history in Spot mode
type BorrowRepayHistoryRequest struct {
	Currency  currency.Code
	EventType string    // auto_borrow, auto_repay, manual_borrow or manual_repay
	After     time.Time // Returns records earlier than this time, inclusive
	Before    time.Time // Returns records newer than this time, inclusive
	Limit     uint64    // At most 100
}

// BorrowRepayHistoryResponse represents a borrow or repay event
type BorrowRepayHistoryResponse struct {
	Currency            currency.Code `json:"ccy"`
	EventType           string        `json:"type"`
	Amount              types.Number  `json:"amt"`
	AccumulatedBorrowed types.Number  `json:"accBorrowed"`
	Timestamp           types.Time    `json:"ts"`
}

// PositionBuilderRequest holds the parameters for calculating margin information for simulated and current positions
type PositionBuilderRequest struct {
	AccountLevel                  string              `json:"acctLv,omitempty"` // 3: Multi-currency margin, 4: Portfolio margin; OKX defaults to 4
	IncludeRealPositionsAndEquity bool                `json:"inclRealPosAndEq"`
	Leverage                      types.Number        `json:"lever,omitempty"` // Cross margin leverage, only applicable to Multi-currency margin
	SimulatedPositions            []SimulatedPosition `json:"simPos,omitempty"`
	SimulatedAssets               []SimulatedAsset    `json:"simAsset,omitempty"`
	GreeksType                    string              `json:"greeksType,omitempty"` // BS, PA or CASH
	IndexVolatility               types.Number        `json:"idxVol,omitempty"`     // Price change ratio from -0.99 to 1; zero is omitted, which leaves the before volatility fields empty
}

// SimulatedPosition represents a simulated position
type SimulatedPosition struct {
	InstrumentID string       `json:"instId"`
	Position     types.Number `json:"pos"`
	AveragePrice types.Number `json:"avgPx"`
	Leverage     types.Number `json:"lever,omitempty"` // Only applicable to Multi-currency margin
}

// SimulatedAsset represents a simulated asset
type SimulatedAsset struct {
	Currency currency.Code `json:"ccy"`
	Amount   types.Number  `json:"amt"`
}

// PositionBuilderResponse represents the margin information of simulated and current positions
type PositionBuilderResponse struct {
	Equity                                  types.Number              `json:"eq"`
	TotalMaintenanceMarginRequirement       types.Number              `json:"totalMmr"`
	TotalInitialMarginRequirement           types.Number              `json:"totalImr"`
	BorrowMaintenanceMarginRequirement      types.Number              `json:"borrowMmr"`
	DerivativesMaintenanceMarginRequirement types.Number              `json:"derivMmr"`
	MarginRatio                             types.Number              `json:"marginRatio"`
	UnrealisedProfitAndLoss                 types.Number              `json:"upl"`
	AccountLeverage                         types.Number              `json:"acctLever"`
	Timestamp                               types.Time                `json:"ts"`
	Assets                                  []PositionBuilderAsset    `json:"assets"`
	RiskUnitData                            []PositionBuilderRiskUnit `json:"riskUnitData"`
	Positions                               []PositionBuilderPosition `json:"positions"` // Only applicable to Multi-currency margin
}

// PositionBuilderAsset represents an asset of a position builder result
type PositionBuilderAsset struct {
	Currency                           currency.Code `json:"ccy"`
	AvailableEquity                    types.Number  `json:"availEq"`
	SpotInUse                          types.Number  `json:"spotInUse"`
	BorrowMaintenanceMarginRequirement types.Number  `json:"borrowMmr"`
	BorrowInitialMarginRequirement     types.Number  `json:"borrowImr"`
}

// PositionBuilderRiskUnit represents a risk unit of a position builder result
type PositionBuilderRiskUnit struct {
	RiskUnit                           string                        `json:"riskUnit"`
	MaintenanceMarginRequirementBefore types.Number                  `json:"mmrBf"`
	MaintenanceMarginRequirement       types.Number                  `json:"mmr"`
	InitialMarginRequirementBefore     types.Number                  `json:"imrBf"`
	InitialMarginRequirement           types.Number                  `json:"imr"`
	UnrealisedProfitAndLoss            types.Number                  `json:"upl"`
	MR1                                types.Number                  `json:"mr1"`
	MR2                                types.Number                  `json:"mr2"`
	MR3                                types.Number                  `json:"mr3"`
	MR4                                types.Number                  `json:"mr4"`
	MR5                                types.Number                  `json:"mr5"`
	MR6                                types.Number                  `json:"mr6"`
	MR7                                types.Number                  `json:"mr7"`
	MR8                                types.Number                  `json:"mr8"`
	MR9                                types.Number                  `json:"mr9"`
	MR1Scenarios                       PositionBuilderMR1Scenarios   `json:"mr1Scenarios"`
	MR1FinalResult                     PositionBuilderMR1FinalResult `json:"mr1FinalResult"`
	MR6FinalResult                     PositionBuilderMR6FinalResult `json:"mr6FinalResult"`
	Delta                              types.Number                  `json:"delta"`
	Gamma                              types.Number                  `json:"gamma"`
	Theta                              types.Number                  `json:"theta"`
	Vega                               types.Number                  `json:"vega"`
	Portfolios                         []PositionBuilderPortfolio    `json:"portfolios"` // Only applicable to Portfolio margin
}

// PositionBuilderMR1Scenarios represents the MR1 stress test profit and loss by price change ratio for each volatility scenario
type PositionBuilderMR1Scenarios struct {
	VolatilityShockDown map[string]types.Number `json:"volShockDown"`
	VolatilitySame      map[string]types.Number `json:"volSame"`
	VolatilityShockUp   map[string]types.Number `json:"volShockUp"`
}

// PositionBuilderMR1FinalResult represents the MR1 worst case scenario of a risk unit
type PositionBuilderMR1FinalResult struct {
	ProfitAndLoss   types.Number `json:"pnl"`
	SpotShock       types.Number `json:"spotShock"`
	VolatilityShock string       `json:"volShock"`
}

// PositionBuilderMR6FinalResult represents the MR6 worst case scenario of a risk unit
type PositionBuilderMR6FinalResult struct {
	ProfitAndLoss types.Number `json:"pnl"`
	SpotShock     types.Number `json:"spotShock"`
}

// PositionBuilderPortfolio represents a portfolio of a position builder risk unit
type PositionBuilderPortfolio struct {
	InstrumentID       string       `json:"instId"`
	InstrumentType     string       `json:"instType"`
	Amount             types.Number `json:"amt"`
	PositionSide       string       `json:"posSide"`
	AveragePrice       types.Number `json:"avgPx"`
	MarkPriceBefore    types.Number `json:"markPxBf"`
	MarkPrice          types.Number `json:"markPx"`
	FloatProfitAndLoss types.Number `json:"floatPnl"`
	NotionalUSD        types.Number `json:"notionalUsd"`
	Delta              types.Number `json:"delta"`
	Gamma              types.Number `json:"gamma"`
	Theta              types.Number `json:"theta"`
	Vega               types.Number `json:"vega"`
	IsRealPosition     bool         `json:"isRealPos"`
}

// PositionBuilderPosition represents a position of a Multi-currency margin position builder result
type PositionBuilderPosition struct {
	InstrumentID                   string       `json:"instId"`
	InstrumentType                 string       `json:"instType"`
	Amount                         types.Number `json:"amt"`
	PositionSide                   string       `json:"posSide"`
	AveragePrice                   types.Number `json:"avgPx"`
	MarkPriceBefore                types.Number `json:"markPxBf"`
	MarkPrice                      types.Number `json:"markPx"`
	FloatProfitAndLoss             types.Number `json:"floatPnl"`
	InitialMarginRequirementBefore types.Number `json:"imrBf"`
	InitialMarginRequirement       types.Number `json:"imr"`
	MarginRatio                    types.Number `json:"mgnRatio"`
	Leverage                       types.Number `json:"lever"`
	NotionalUSD                    types.Number `json:"notionalUsd"`
	IsRealPosition                 bool         `json:"isRealPos"`
}

// PositionBuilderTrendGraphRequest holds the parameters for calculating a position builder trend graph
type PositionBuilderTrendGraphRequest struct {
	IncludeRealPositionsAndEquity bool                     `json:"inclRealPosAndEq"`
	SimulatedPositions            []SimulatedPosition      `json:"simPos,omitempty"`
	SimulatedAssets               []SimulatedAsset         `json:"simAsset,omitempty"`
	GraphType                     string                   `json:"type"` // mmr
	MMRConfig                     PositionBuilderMMRConfig `json:"mmrConfig"`
}

// PositionBuilderMMRConfig represents the maintenance margin requirement configuration of a trend graph
type PositionBuilderMMRConfig struct {
	AccountLevel string       `json:"acctLv,omitempty"` // 3: Multi-currency margin, 4: Portfolio margin
	Leverage     types.Number `json:"lever,omitempty"`  // Cross margin leverage, only applicable to Multi-currency margin
}

// PositionBuilderTrendGraphResponse represents a position builder trend graph
type PositionBuilderTrendGraphResponse struct {
	GraphType string                   `json:"type"`
	MMRData   []PositionBuilderMMRData `json:"mmrData"` // In ascending shock factor order
}

// PositionBuilderMMRData represents the maintenance margin requirement at a price change ratio
type PositionBuilderMMRData struct {
	ShockFactor                  types.Number `json:"shockFactor"`
	MaintenanceMarginRequirement types.Number `json:"mmr"`
	MaintenanceMarginRatio       types.Number `json:"mmrRatio"`
}

// SetRiskOffsetAmountRequest holds the parameters for setting the spot risk offset amount
type SetRiskOffsetAmountRequest struct {
	Currency              currency.Code `json:"ccy"`
	ClientSpotInUseAmount types.Number  `json:"clSpotInUseAmt"`
}

// RiskOffsetAmountResponse represents the spot risk offset amount set
type RiskOffsetAmountResponse struct {
	Currency              currency.Code `json:"ccy"`
	ClientSpotInUseAmount types.Number  `json:"clSpotInUseAmt"`
}

// GreeksRequest holds the parameters for fetching account greeks
type GreeksRequest struct {
	Currency currency.Code
}

// GreeksResponse represents the greeks of an asset in the account
type GreeksResponse struct {
	DeltaBS   types.Number  `json:"deltaBS"`
	DeltaPA   types.Number  `json:"deltaPA"`
	GammaBS   types.Number  `json:"gammaBS"`
	GammaPA   types.Number  `json:"gammaPA"`
	ThetaBS   types.Number  `json:"thetaBS"`
	ThetaPA   types.Number  `json:"thetaPA"`
	VegaBS    types.Number  `json:"vegaBS"`
	VegaPA    types.Number  `json:"vegaPA"`
	Currency  currency.Code `json:"ccy"`
	Timestamp types.Time    `json:"ts"`
}

// PMPositionLimitationRequest holds the parameters for fetching Portfolio margin position limitations
type PMPositionLimitationRequest struct {
	InstrumentType     string   // Required: SWAP, FUTURES or OPTION
	InstrumentFamilies []string // Required; at most 5
}

// PMLimitationResponse represents the Portfolio margin position limitation of an instrument family
type PMLimitationResponse struct {
	Underlying       string       `json:"uly"`
	InstrumentFamily string       `json:"instFamily"`
	MaxSize          types.Number `json:"maxSz"`
	PositionType     string       `json:"posType"`
}

// ActivateOptionResponse represents the option trading activation
type ActivateOptionResponse struct {
	Timestamp types.Time `json:"ts"`
}

// ActivateFeatureRequest is the request body for activating a feature
type ActivateFeatureRequest struct {
	Feature string `json:"feature"` // 1: USDC order book trading
}

// SetAutoLoanRequest is the request body for setting auto loan
type SetAutoLoanRequest struct {
	AutoLoan bool `json:"autoLoan"`
}

// AutoLoanResponse represents the auto loan setting
type AutoLoanResponse struct {
	AutoLoan bool `json:"autoLoan"`
}

// PresetAccountModeSwitchRequest holds the parameters for presetting an account mode switch
type PresetAccountModeSwitchRequest struct {
	AccountLevel   string       `json:"acctLv"`                   // 2: Futures mode, 3: Multi-currency margin, 4: Portfolio margin
	Leverage       types.Number `json:"lever,omitempty"`          // Required when leaving Portfolio margin with cross margin positions
	RiskOffsetType string       `json:"riskOffsetType,omitempty"` // Deprecated
}

// PresetAccountModeSwitchResponse represents an account mode switch preset
type PresetAccountModeSwitchResponse struct {
	CurrentAccountLevel string       `json:"curAcctLv"`
	AccountLevel        string       `json:"acctLv"`
	Leverage            types.Number `json:"lever"`
	RiskOffsetType      string       `json:"riskOffsetType"` // Deprecated
}

// PrecheckAccountModeSwitchResponse represents the precheck information for an account mode switch
type PrecheckAccountModeSwitchResponse struct {
	StatusCode          string                               `json:"sCode"` // 0: passed, 1: unmatched information, 3: leverage not set, 4: position tier or margin check failed
	CurrentAccountLevel string                               `json:"curAcctLv"`
	AccountLevel        string                               `json:"acctLv"`
	RiskOffsetType      string                               `json:"riskOffsetType"`
	UnmatchedInfoCheck  []AccountModeSwitchUnmatchedInfo     `json:"unmatchedInfoCheck"`
	Positions           []AccountModeSwitchPosition          `json:"posList"`
	PositionTierCheck   []AccountModeSwitchPositionTierCheck `json:"posTierCheck"`
	MarginBefore        *AccountModeSwitchMargin             `json:"mgnBf"`
	MarginAfter         *AccountModeSwitchMargin             `json:"mgnAft"`
}

// AccountModeSwitchUnmatchedInfo represents information that blocks an account mode switch
type AccountModeSwitchUnmatchedInfo struct {
	Type        string       `json:"type"`
	TotalAsset  types.Number `json:"totalAsset"`
	PositionIDs []string     `json:"posList"`
}

// AccountModeSwitchPosition represents the leverage of a cross margin contract position after an account mode switch
type AccountModeSwitchPosition struct {
	PositionID string       `json:"posId"`
	Leverage   types.Number `json:"lever"`
}

// AccountModeSwitchPositionTierCheck represents a cross margin contract position that fails the position tier check
type AccountModeSwitchPositionTierCheck struct {
	InstrumentFamily string       `json:"instFamily"`
	InstrumentType   string       `json:"instType"`
	Position         types.Number `json:"pos"`
	Leverage         types.Number `json:"lever"`
	MaxSize          types.Number `json:"maxSz"`
}

// AccountModeSwitchMargin represents margin information before or after an account mode switch
type AccountModeSwitchMargin struct {
	AccountAvailableEquity types.Number                    `json:"acctAvailEq"`
	MarginRatio            types.Number                    `json:"mgnRatio"`
	Details                []AccountModeSwitchMarginDetail `json:"details"`
}

// AccountModeSwitchMarginDetail represents the margin information of a currency for an account mode switch
type AccountModeSwitchMarginDetail struct {
	Currency        currency.Code `json:"ccy"`
	AvailableEquity types.Number  `json:"availEq"`
	MarginRatio     types.Number  `json:"mgnRatio"`
}

// SetAccountModeRequest is the request body for setting the account mode
type SetAccountModeRequest struct {
	AccountLevel string `json:"acctLv"`
}

// AccountModeResponse represents the account mode set
type AccountModeResponse struct {
	AccountLevel string `json:"acctLv"`
}

// SetCollateralAssetsRequest holds the parameters for setting collateral assets
type SetCollateralAssetsRequest struct {
	Type              string          `json:"type,omitempty"` // all or custom
	CollateralEnabled bool            `json:"collateralEnabled"`
	Currencies        []currency.Code `json:"ccyList,omitempty"` // Required when Type is custom
}

// CollateralAssetsResponse represents the collateral assets set
type CollateralAssetsResponse struct {
	Type              string          `json:"type"`
	CollateralEnabled bool            `json:"collateralEnabled"`
	Currencies        []currency.Code `json:"ccyList"`
}

// CollateralAssetsRequest holds the parameters for fetching collateral assets
type CollateralAssetsRequest struct {
	Currencies        []currency.Code // At most 20
	CollateralEnabled *bool           // Both collateral and non-collateral assets when nil
}

// CollateralAssetResponse represents whether a currency is a collateral asset
type CollateralAssetResponse struct {
	Currency          currency.Code `json:"ccy"`
	CollateralEnabled bool          `json:"collateralEnabled"`
}

// ResetMMPStatusRequest holds the parameters for resetting the MMP status
type ResetMMPStatusRequest struct {
	InstrumentType   string `json:"instType,omitempty"` // OPTION, the default
	InstrumentFamily string `json:"instFamily"`
}

// MMPStatusResponse represents the result of resetting the MMP status
type MMPStatusResponse struct {
	Result bool `json:"result"`
}

// SetMMPRequest holds the parameters for setting the MMP configuration
type SetMMPRequest struct {
	InstrumentFamily string       `json:"instFamily"`
	TimeInterval     uint64       `json:"timeInterval,string"`   // Milliseconds; 0 disables MMP
	FrozenInterval   uint64       `json:"frozenInterval,string"` // Milliseconds; 0 freezes trading until the MMP status is reset
	QuantityLimit    types.Number `json:"qtyLimit"`              // Contracts; must be greater than 0
}

// SetMMPResponse represents the MMP configuration set
type SetMMPResponse struct {
	InstrumentFamily string       `json:"instFamily"`
	TimeInterval     types.Number `json:"timeInterval"`
	FrozenInterval   types.Number `json:"frozenInterval"`
	QuantityLimit    types.Number `json:"qtyLimit"`
}

// MMPConfigRequest holds the parameters for fetching the MMP configuration
type MMPConfigRequest struct {
	InstrumentFamily string
}

// MMPConfigResponse represents the MMP configuration of an instrument family
type MMPConfigResponse struct {
	InstrumentFamily string       `json:"instFamily"`
	MMPFrozen        bool         `json:"mmpFrozen"`
	MMPFrozenUntil   types.Number `json:"mmpFrozenUntil"`
	TimeInterval     types.Number `json:"timeInterval"`
	FrozenInterval   types.Number `json:"frozenInterval"`
	QuantityLimit    types.Number `json:"qtyLimit"`
}

// MovePositionsRequest holds the parameters for moving positions between accounts under one master account
type MovePositionsRequest struct {
	FromAccount string            `json:"fromAcct"` // 0 for the master account
	ToAccount   string            `json:"toAcct"`   // 0 for the master account
	Legs        []MovePositionLeg `json:"legs"`
	ClientID    string            `json:"clientId"`
}

// MovePositionLeg represents a position to move
type MovePositionLeg struct {
	From MovePositionLegSource      `json:"from"`
	To   MovePositionLegDestination `json:"to"`
}

// MovePositionLegSource represents the position to move in the source account
type MovePositionLegSource struct {
	PositionID string       `json:"posId"`
	Size       types.Number `json:"sz"`
	Side       string       `json:"side"` // buy or sell, from the source account's perspective
}

// MovePositionLegDestination represents the configuration of the moved position in the destination account
type MovePositionLegDestination struct {
	TradeMode    string        `json:"tdMode,omitempty"`  // cross or isolated
	PositionSide string        `json:"posSide,omitempty"` // net, long or short
	Currency     currency.Code `json:"ccy,omitzero"`      // Only applicable to cross margin positions in Futures mode
}

// MovePositionsResponse represents the result of moving positions
type MovePositionsResponse struct {
	ClientID     string                  `json:"clientId"`
	BlockTradeID string                  `json:"blockTdId"`
	State        string                  `json:"state"`
	Timestamp    types.Time              `json:"ts"`
	FromAccount  string                  `json:"fromAcct"`
	ToAccount    string                  `json:"toAcct"`
	Legs         []MovePositionLegResult `json:"legs"`
}

// MovePositionLegResult represents the result of moving a position
type MovePositionLegResult struct {
	From MovePositionLegSourceResult      `json:"from"`
	To   MovePositionLegDestinationResult `json:"to"`
}

// MovePositionLegSourceResult represents the result of a moved position in the source account
type MovePositionLegSourceResult struct {
	PositionID    string       `json:"posId"`
	InstrumentID  string       `json:"instId"`
	Price         types.Number `json:"px"`
	Side          string       `json:"side"`
	Size          types.Number `json:"sz"`
	StatusCode    string       `json:"sCode"`
	StatusMessage string       `json:"sMsg"`
}

// MovePositionLegDestinationResult represents the result of a moved position in the destination account
type MovePositionLegDestinationResult struct {
	InstrumentID  string        `json:"instId"`
	Price         types.Number  `json:"px"`
	Side          string        `json:"side"`
	Size          types.Number  `json:"sz"`
	TradeMode     string        `json:"tdMode"`
	PositionSide  string        `json:"posSide"`
	Currency      currency.Code `json:"ccy"`
	StatusCode    string        `json:"sCode"`
	StatusMessage string        `json:"sMsg"`
}

// MovePositionsHistoryRequest holds the parameters for fetching moved positions from the last 3 days
type MovePositionsHistoryRequest struct {
	BlockTradeID string
	ClientID     string
	BeginTime    time.Time
	EndTime      time.Time
	Limit        uint64 // At most 100
	State        string // filled or pending
}

// MovePositionsHistoryResponse represents moved positions
type MovePositionsHistoryResponse struct {
	ClientID     string                   `json:"clientId"`
	BlockTradeID string                   `json:"blockTdId"`
	State        string                   `json:"state"`
	Timestamp    types.Time               `json:"ts"`
	FromAccount  string                   `json:"fromAcct"`
	ToAccount    string                   `json:"toAcct"`
	Legs         []MovePositionHistoryLeg `json:"legs"`
}

// MovePositionHistoryLeg represents a moved position
type MovePositionHistoryLeg struct {
	From MovePositionHistorySource      `json:"from"`
	To   MovePositionHistoryDestination `json:"to"`
}

// MovePositionHistorySource represents a moved position in the source account
type MovePositionHistorySource struct {
	PositionID   string       `json:"posId"`
	InstrumentID string       `json:"instId"`
	Price        types.Number `json:"px"`
	Side         string       `json:"side"`
	Size         types.Number `json:"sz"`
}

// MovePositionHistoryDestination represents a moved position in the destination account
type MovePositionHistoryDestination struct {
	InstrumentID string        `json:"instId"`
	Price        types.Number  `json:"px"`
	Side         string        `json:"side"`
	Size         types.Number  `json:"sz"`
	TradeMode    string        `json:"tdMode"`
	PositionSide string        `json:"posSide"`
	Currency     currency.Code `json:"ccy"`
}

// SetAutoEarnRequest holds the parameters for turning auto earn on or off
type SetAutoEarnRequest struct {
	EarnType string        `json:"earnType,omitempty"` // 0: auto lend and staking, the default; 1: USDG earn
	Currency currency.Code `json:"ccy"`
	Action   string        `json:"action"`        // turn_on or turn_off
	APR      types.Number  `json:"apr,omitempty"` // Deprecated minimum lending APR, e.g. 0.01 for 1%
}

// AutoEarnResponse represents the auto earn setting
type AutoEarnResponse struct {
	EarnType string        `json:"earnType"`
	Currency currency.Code `json:"ccy"`
	Action   string        `json:"action"`
	APR      types.Number  `json:"apr"` // Deprecated
}

// SetSettleCurrencyRequest is the request body for setting the USD-margined contract settlement currency
type SetSettleCurrencyRequest struct {
	SettlementCurrency currency.Code `json:"settleCcy"`
}

// SettleCurrencyResponse represents the USD-margined contract settlement currency set
type SettleCurrencyResponse struct {
	SettlementCurrency currency.Code `json:"settleCcy"`
}

// SetTradingConfigRequest holds the parameters for setting a trading configuration
type SetTradingConfigRequest struct {
	Type         string `json:"type"`               // stgyType
	StrategyType string `json:"stgyType,omitempty"` // 0: general strategy, 1: delta neutral strategy; only applicable when Type is stgyType
}

// TradingConfigResponse represents the trading configuration set
type TradingConfigResponse struct {
	Type         string `json:"type"`
	StrategyType string `json:"stgyType"`
}

// PrecheckSetDeltaNeutralResponse represents the precheck information for switching the strategy type
type PrecheckSetDeltaNeutralResponse struct {
	UnmatchedInfoCheck []DeltaNeutralUnmatchedInfo `json:"unmatchedInfoCheck"`
}

// DeltaNeutralUnmatchedInfo represents information that blocks a strategy type switch
type DeltaNeutralUnmatchedInfo struct {
	Type          string       `json:"type"`
	DeltaLeverage types.Number `json:"deltaLever"`
	OrderIDs      []string     `json:"ordList"`
	PositionIDs   []string     `json:"posList"`
}

// AdjustDemoAccountBalanceRequest holds the parameters for adjusting demo trading balances
type AdjustDemoAccountBalanceRequest struct {
	Type        string                  `json:"type"` // increase or reduce
	Adjustments []DemoBalanceAdjustment `json:"adjustments"`
}

// DemoBalanceAdjustment represents a demo trading balance adjustment of a currency
type DemoBalanceAdjustment struct {
	Currency currency.Code `json:"ccy"` // BTC, ETH, USDT or OKB
	Amount   types.Number  `json:"amt"`
}

// AdjustDemoAccountBalanceResponse represents the result of adjusting demo trading balances
type AdjustDemoAccountBalanceResponse struct {
	RemainingCount types.Number                  `json:"remainCnt"` // Remaining daily increase quota
	TotalCount     types.Number                  `json:"totalCnt"`  // Total daily increase quota
	Details        []DemoBalanceAdjustmentResult `json:"details"`
}

// DemoBalanceAdjustmentResult represents the result of a demo trading balance adjustment of a currency
type DemoBalanceAdjustmentResult struct {
	Currency currency.Code `json:"ccy"`
	Amount   types.Number  `json:"amt"`
	Balance  types.Number  `json:"bal"`
}

// GLPTodayPerformanceResponse represents the current day and month to date Global Liquidity Program performance
type GLPTodayPerformanceResponse struct {
	DataReady bool                    `json:"dataReady"`
	DataDate  string                  `json:"dataDate"` // yyyy-MM-dd in UTC+8
	Account   GLPAccount              `json:"account"`
	Programs  []GLPProgramPerformance `json:"programs"`
}

// GLPAccount represents the account identity of a Global Liquidity Program performance snapshot
type GLPAccount struct {
	MasterAccountID    string   `json:"masterAccountId"`
	CombinedAccountIDs []string `json:"combinedAccountIds"`
}

// GLPProgramPerformance represents the performance of an enrolled Global Liquidity Program
type GLPProgramPerformance struct {
	Program               string                    `json:"program"` //nolint:misspell // SPOT, PERP or FUT_NTO, OKX's Expiry and Nitro program
	MarketMakerBusinessID string                    `json:"marketMakerBusinessId"`
	EnrollmentStatus      string                    `json:"enrollmentStatus"`
	MarketMakerLevelID    string                    `json:"marketMakerLevelId"`
	EnrolledTierDisplay   string                    `json:"enrolledTierDisplay"`
	QualifyingPool        string                    `json:"qualifyingPool"`
	QualifyingRows        []string                  `json:"qualifyingRows"`
	Daily                 GLPPerformance            `json:"daily"`
	MonthToDate           GLPMonthToDatePerformance `json:"mtd"`
}

// GLPPerformance represents Global Liquidity Program volumes and market shares by pool
type GLPPerformance struct {
	Volume GLPPoolBreakdown `json:"volume"`
	Share  GLPPoolBreakdown `json:"share"`
}

// GLPMonthToDatePerformance represents month to date Global Liquidity Program performance
type GLPMonthToDatePerformance struct {
	Volume            GLPPoolBreakdown `json:"volume"`
	Share             GLPPoolBreakdown `json:"share"`
	MonthToDateStatus string           `json:"mtdStatus"`
	QualifyingShare   GLPMakerTaker    `json:"qualifyingShare"`
}

// GLPPoolBreakdown represents Global Liquidity Program values by pool type; a pool OKX does not report stays zero
type GLPPoolBreakdown struct {
	TypeA         GLPMakerTaker `json:"typeA"`
	TypeBTotal    GLPMakerTaker `json:"typeBTotal"`
	TypeBAdjusted GLPMakerTaker `json:"typeBAdj"`
	TradFiDoubled GLPMakerTaker `json:"tradfiX2"`
	Total         GLPMakerTaker `json:"total"`
}

// GLPMakerTaker represents a Global Liquidity Program maker and taker value pair
type GLPMakerTaker struct {
	Maker types.Number `json:"maker"`
	Taker types.Number `json:"taker"`
}

// GLPHistoricalPerformanceRequest holds the parameters for fetching daily Global Liquidity Program performance history
type GLPHistoricalPerformanceRequest struct {
	Program   string    //nolint:misspell // Required: SPOT, PERP or FUT_NTO, OKX's Expiry and Nitro program
	BeginTime time.Time // OKX defaults to the first day of the current month in UTC+8
	EndTime   time.Time // OKX defaults to today in UTC+8
	Limit     uint64    // At most 100
}

// GLPHistoricalPerformanceResponse represents a day of Global Liquidity Program performance
type GLPHistoricalPerformanceResponse struct {
	Date   string           `json:"date"` // yyyy-MM-dd in UTC+8
	Volume GLPPoolBreakdown `json:"volume"`
	Share  GLPPoolBreakdown `json:"share"`
}

// GLPOverviewResponse represents the active Global Liquidity Program enrollments
type GLPOverviewResponse struct {
	Programs []GLPOverviewProgram `json:"programs"`
}

// GLPOverviewProgram represents an active Global Liquidity Program enrollment
type GLPOverviewProgram struct {
	BusinessID          string                   `json:"bizId"`
	Program             string                   `json:"program"`
	Side                string                   `json:"side"`
	Instrument          string                   `json:"instrument"`
	CurrentTier         string                   `json:"currentTier"`
	EstimatedNextTier   string                   `json:"estNextTier"`
	EstimatedNextStatus string                   `json:"estNextStatus"`
	GracePeriodEndTime  types.Time               `json:"gracePeriodEndTime"`
	Requirements        []GLPTierRequirement     `json:"requirement"`
	Performance         []GLPOverviewPerformance `json:"performance"`
}

// GLPTierRequirement represents the market share a Global Liquidity Program tier requires
type GLPTierRequirement struct {
	Tier  string       `json:"tier"`
	Share types.Number `json:"share"`
}

// GLPOverviewPerformance represents week to date Global Liquidity Program performance on a trading date
type GLPOverviewPerformance struct {
	Date             types.Time   `json:"date"`
	AssessmentPeriod string       `json:"assessPeriod"`
	Share            types.Number `json:"share"`
	Volume           types.Number `json:"vol"`
}

// GLPAssessmentDetailsResponse represents Global Liquidity Program performance within the current assessment period
type GLPAssessmentDetailsResponse struct {
	AssessmentStartTime types.Time                 `json:"assessStartTime"`
	AssessmentEndTime   types.Time                 `json:"assessEndTime"`
	Performance         []GLPAssessmentPerformance `json:"performance"`
}

// GLPAssessmentPerformance represents daily and week to date Global Liquidity Program performance
type GLPAssessmentPerformance struct {
	Date                      types.Time   `json:"date"`
	OriginalVolume            types.Number `json:"origVol"`
	OriginalShare             types.Number `json:"origShare"`
	AdjustedVolume            types.Number `json:"adjVol"`
	AdjustedShare             types.Number `json:"adjShare"`
	AccumulatedOriginalVolume types.Number `json:"accOrigVol"`
	AccumulatedOriginalShare  types.Number `json:"accOrigShare"`
	AccumulatedAdjustedVolume types.Number `json:"accAdjVol"`
	AccumulatedAdjustedShare  types.Number `json:"accAdjShare"`
}

// GLPPerformanceHistoryResponse represents weekly Global Liquidity Program assessment results
type GLPPerformanceHistoryResponse struct {
	Performance []GLPAssessmentResult `json:"performance"`
}

// GLPAssessmentResult represents a weekly Global Liquidity Program assessment result
type GLPAssessmentResult struct {
	AssessmentStartTime       types.Time   `json:"assessStartTime"`
	AssessmentEndTime         types.Time   `json:"assessEndTime"`
	Tier                      string       `json:"tier"`
	NextTier                  string       `json:"nextTier"`
	AccumulatedOriginalVolume types.Number `json:"accOrigVol"`
	AccumulatedOriginalShare  types.Number `json:"accOrigShare"`
	AccumulatedAdjustedVolume types.Number `json:"accAdjVol"`
	AccumulatedAdjustedShare  types.Number `json:"accAdjShare"`
	AssessmentResult          string       `json:"assessResult"`
}
