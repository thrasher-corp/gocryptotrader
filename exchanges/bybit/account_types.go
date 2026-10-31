package bybit

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// WalletBalanceRequest holds the parameters of GetWalletBalance
type WalletBalanceRequest struct {
	// AccountType is required; UNIFIED is the only account type since Bybit retired classic accounts
	AccountType string
	// Coins limits the result to these coins; otherwise only coins with assets or liabilities are returned
	Coins []currency.Code
}

// WalletBalanceResponse holds the wallet balance of each requested account type
type WalletBalanceResponse struct {
	List []WalletBalance `json:"list"`
}

// WalletBalance holds an account's wallet balance and risk rates. The account wide fields do not apply to isolated
// margin
type WalletBalance struct {
	AccountType                  string       `json:"accountType"`
	AccountInitialMarginRate     types.Number `json:"accountIMRate"`
	AccountMaintenanceMarginRate types.Number `json:"accountMMRate"`
	// TotalEquity is the USD value of every coin's equity
	TotalEquity types.Number `json:"totalEquity"`
	// TotalWalletBalance is the USD value of every coin's wallet balance
	TotalWalletBalance types.Number `json:"totalWalletBalance"`
	// TotalMarginBalance is TotalWalletBalance plus TotalPerpetualUnrealisedPNL, in USD
	TotalMarginBalance types.Number `json:"totalMarginBalance"`
	// TotalAvailableBalance is in USD: TotalMarginBalance (cross margin) or TotalEquity (portfolio margin), less the
	// haircut and TotalInitialMargin
	TotalAvailableBalance types.Number `json:"totalAvailableBalance"`
	// TotalPerpetualUnrealisedPNL is the unrealised P&L of perpetuals and futures, in USD
	TotalPerpetualUnrealisedPNL types.Number `json:"totalPerpUPL"`
	// TotalInitialMargin and TotalMaintenanceMargin are in USD
	TotalInitialMargin     types.Number `json:"totalInitialMargin"`
	TotalMaintenanceMargin types.Number `json:"totalMaintenanceMargin"`
	// The ByMarkPrice fields repeat the fields above, which Bybit now also calculates by mark price
	AccountInitialMarginRateByMarkPrice     types.Number `json:"accountIMRateByMp"`
	AccountMaintenanceMarginRateByMarkPrice types.Number `json:"accountMMRateByMp"`
	TotalInitialMarginByMarkPrice           types.Number `json:"totalInitialMarginByMp"`
	TotalMaintenanceMarginByMarkPrice       types.Number `json:"totalMaintenanceMarginByMp"`
	// Deprecated: Bybit deprecated AccountLTV
	AccountLTV types.Number        `json:"accountLTV"`
	Coin       []WalletBalanceCoin `json:"coin"`
}

// WalletBalanceCoin holds the balance of one coin in an account
type WalletBalanceCoin struct {
	Coin currency.Code `json:"coin"`
	// Equity is WalletBalance - SpotBorrow + UnrealisedPNL + the coin's option value
	Equity        types.Number `json:"equity"`
	USDValue      types.Number `json:"usdValue"`
	WalletBalance types.Number `json:"walletBalance"`
	// Locked is held by spot open orders
	Locked types.Number `json:"locked"`
	// SpotHedgingQuantity is the spot quantity hedging portfolio margin positions
	SpotHedgingQuantity types.Number `json:"spotHedgingQty"`
	// BorrowAmount is the coin's spot and derivatives liabilities
	BorrowAmount    types.Number `json:"borrowAmount"`
	AccruedInterest types.Number `json:"accruedInterest"`
	// TotalOrderInitialMargin, TotalPositionInitialMargin and TotalPositionMaintenanceMargin are "" in portfolio
	// margin mode
	TotalOrderInitialMargin        types.Number `json:"totalOrderIM"`
	TotalPositionInitialMargin     types.Number `json:"totalPositionIM"`
	TotalPositionMaintenanceMargin types.Number `json:"totalPositionMM"`
	UnrealisedPNL                  types.Number `json:"unrealisedPnl"`
	CumulativeRealisedPNL          types.Number `json:"cumRealisedPnl"`
	Bonus                          types.Number `json:"bonus"`
	// MarginCollateral is whether the platform allows the coin as collateral; CollateralSwitch, whether the user
	// turned it on, only applies when it is true
	MarginCollateral bool `json:"marginCollateral"`
	CollateralSwitch bool `json:"collateralSwitch"`
	// CollateralRestriction is the platform collateral restriction status: -1 unknown, 0 not enabled, 1 not enabled
	// but close to the platform's collateral limit, 2 enabled (adding collateral, switching collateral on and
	// changing margin mode are rejected)
	CollateralRestriction string `json:"colRes"`
	// SpotBorrow is borrowed by spot margin trading and manual borrowing, excluding active spot margin orders
	SpotBorrow types.Number `json:"spotBorrow"`
	// Deprecated: Free applied to the retired spot wallet
	Free types.Number `json:"free"`
	// Deprecated: AvailableToWithdraw is always "" for UNIFIED accounts; GetTransferableAmount returns the
	// transferable amount instead
	AvailableToWithdraw types.Number `json:"availableToWithdraw"`
	// Deprecated: AvailableToBorrow is always ""; GetCollateralInfo returns the amount available to borrow instead
	AvailableToBorrow types.Number `json:"availableToBorrow"`
}

// TransferableAmountResponse holds the amounts that can be transferred out of the unified account
type TransferableAmountResponse struct {
	// AvailableWithdrawal is the transferable amount of the first requested coin
	AvailableWithdrawal    types.Number                   `json:"availableWithdrawal"`
	AvailableWithdrawalMap map[currency.Code]types.Number `json:"availableWithdrawalMap"`
}

// TransactionLogRequest holds the parameters of GetTransactionLog; every field is optional
type TransactionLogRequest struct {
	AccountType string
	Category    string
	Currency    currency.Code
	BaseCoin    currency.Code
	Type        string
	// TransactionSubType movePosition limits the result to logs generated by moving positions
	TransactionSubType string
	// StartTime and EndTime default to the last 24 hours; with only one set, the range is the 24 hours from or to it
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
	Cursor    string
}

// TransactionLogResponse holds a page of transaction logs
type TransactionLogResponse struct {
	List           []TransactionLogItem `json:"list"`
	NextPageCursor string               `json:"nextPageCursor"`
}

// TransactionLogItem holds one transaction log
type TransactionLogItem struct {
	ID              string     `json:"id"`
	Symbol          string     `json:"symbol"`
	Category        string     `json:"category"`
	Side            string     `json:"side"`
	TransactionTime types.Time `json:"transactionTime"`
	Type            string     `json:"type"`
	// TransactionSubType is movePosition for logs generated by moving positions, otherwise ""
	TransactionSubType string `json:"transSubType"`
	// Quantity is signed for spot, where a negative value decreases the coin; for derivatives it is the unsigned
	// quantity of the trade
	Quantity types.Number `json:"qty"`
	// Size is the signed position size after the trade
	Size       types.Number  `json:"size"`
	Currency   currency.Code `json:"currency"`
	TradePrice types.Number  `json:"tradePrice"`
	// Funding is positive when funding is received, the opposite sign of execFee from Get Trade History
	Funding types.Number `json:"funding"`
	// Fee is positive for an expense and negative for a rebate
	Fee      types.Number `json:"fee"`
	CashFlow types.Number `json:"cashFlow"`
	// Change is CashFlow + Funding - Fee
	Change      types.Number `json:"change"`
	CashBalance types.Number `json:"cashBalance"`
	// FeeRate is the trading fee rate for TRADE logs and the funding rate for SETTLEMENT logs
	FeeRate     types.Number `json:"feeRate"`
	BonusChange types.Number `json:"bonusChange"`
	TradeID     string       `json:"tradeId"`
	OrderID     string       `json:"orderId"`
	OrderLinkID string       `json:"orderLinkId"`
	// ExtraFees is a JSON encoded list of extra fees for spot orders on the Indonesian site and spot fiat orders on
	// the EU site, otherwise ""
	ExtraFees   string `json:"extraFees"`
	DisplayType string `json:"displayType"`
}

// AccountInfoResponse holds the margin mode and status of the account
type AccountInfoResponse struct {
	// UnifiedMarginStatus is 1 classic account, 3 UTA 1.0, 4 UTA 1.0 Pro, 5 UTA 2.0 and 6 UTA 2.0 Pro
	UnifiedMarginStatus uint64     `json:"unifiedMarginStatus"`
	MarginMode          string     `json:"marginMode"`
	IsMasterTrader      bool       `json:"isMasterTrader"`
	SpotHedgingStatus   string     `json:"spotHedgingStatus"`
	UpdatedTime         types.Time `json:"updatedTime"`
	// Deprecated: DCPStatus is always OFF; GetDCPInfo returns the DCP configuration instead
	DCPStatus string `json:"dcpStatus"`
	// Deprecated: TimeWindow is always 0; GetDCPInfo returns the DCP configuration instead
	TimeWindow uint64 `json:"timeWindow"`
	// Deprecated: SMPGroup is always 0; GetSMPGroupID returns the SMP group instead. Bybit sends smpGroup as a number
	// here and as a string on its order endpoints, so either is accepted
	SMPGroup types.Number `json:"smpGroup"`
}

// AccountInstrumentsInfoRequest holds the parameters of GetAccountInstrumentsInfo
type AccountInstrumentsInfoRequest struct {
	// Category is required: spot, linear or inverse
	Category string
	Symbol   string
	// Limit and Cursor do not apply to spot
	Limit  uint64
	Cursor string
}

// AccountInstrumentsInfoResponse holds a page of the instruments the account can trade
type AccountInstrumentsInfoResponse struct {
	Category string              `json:"category"`
	List     []AccountInstrument `json:"list"`
	// NextPageCursor is not returned for spot
	NextPageCursor string `json:"nextPageCursor"`
}

// AccountInstrument holds an instrument's specification. Spot and derivatives instruments return different fields;
// each field not shared by both says which it belongs to
type AccountInstrument struct {
	Symbol   string `json:"symbol"`
	SymbolID uint64 `json:"symbolId"`
	// ContractType is for linear and inverse
	ContractType string        `json:"contractType"`
	Status       string        `json:"status"`
	BaseCoin     currency.Code `json:"baseCoin"`
	QuoteCoin    currency.Code `json:"quoteCoin"`
	SymbolType   string        `json:"symbolType"`
	// LaunchTime, DeliveryTime, DeliveryFeeRate, PriceScale and LeverageFilter are for linear and inverse;
	// DeliveryTime is the delisting time of a perpetual
	LaunchTime      types.Time                      `json:"launchTime"`
	DeliveryTime    types.Time                      `json:"deliveryTime"`
	DeliveryFeeRate types.Number                    `json:"deliveryFeeRate"`
	PriceScale      types.Number                    `json:"priceScale"`
	LeverageFilter  AccountInstrumentLeverageFilter `json:"leverageFilter"`
	PriceFilter     AccountInstrumentPriceFilter    `json:"priceFilter"`
	LotSizeFilter   AccountInstrumentLotSizeFilter  `json:"lotSizeFilter"`
	// UnifiedMarginTrade to UnderlyingTicker are for linear and inverse; FundingInterval is in minutes, and
	// MarketRegion and UnderlyingTicker apply to TradFi perpetuals
	UnifiedMarginTrade bool                            `json:"unifiedMarginTrade"`
	FundingInterval    uint64                          `json:"fundingInterval"`
	SettleCoin         currency.Code                   `json:"settleCoin"`
	CopyTrading        string                          `json:"copyTrading"`
	UpperFundingRate   types.Number                    `json:"upperFundingRate"`
	LowerFundingRate   types.Number                    `json:"lowerFundingRate"`
	DisplayName        string                          `json:"displayName"`
	FullName           string                          `json:"fullName"`
	MarketRegion       string                          `json:"marketRegion"`
	UnderlyingTicker   string                          `json:"underlyingTicker"`
	RiskParameters     AccountInstrumentRiskParameters `json:"riskParameters"`
	// IsPreListing and PreListingInfo are for linear and inverse; PreListingInfo is nil unless IsPreListing
	IsPreListing   bool                             `json:"isPreListing"`
	PreListingInfo *AccountInstrumentPreListingInfo `json:"preListingInfo"`
	// IsPublicRPI is whether every user can submit RPI orders for the symbol, and may lag by up to an hour
	IsPublicRPI bool `json:"isPublicRpi"`
	// Deprecated: MyRPIPermission always equals IsPublicRPI
	MyRPIPermission bool `json:"myRpiPermission"`
	// Deprecated: Innovation is for spot; SymbolType replaces it
	Innovation types.Boolean `json:"innovation"`
	// XStockMultiplier is for spot xstocks symbols: stock price = token price / multiplier
	XStockMultiplier types.Number `json:"xstockMultiplier"`
	// MarginTrading and SpecialTreatmentTag are for spot; SpecialTreatmentTag is whether the symbol has a special treatment label
	MarginTrading       string        `json:"marginTrading"`
	SpecialTreatmentTag types.Boolean `json:"stTag"`
}

// AccountInstrumentLeverageFilter holds a derivatives instrument's leverage limits
type AccountInstrumentLeverageFilter struct {
	MinLeverage  types.Number `json:"minLeverage"`
	MaxLeverage  types.Number `json:"maxLeverage"`
	LeverageStep types.Number `json:"leverageStep"`
}

// AccountInstrumentPriceFilter holds an instrument's price limits; MinPrice and MaxPrice are for linear and inverse
type AccountInstrumentPriceFilter struct {
	MinPrice types.Number `json:"minPrice"`
	MaxPrice types.Number `json:"maxPrice"`
	TickSize types.Number `json:"tickSize"`
}

// AccountInstrumentLotSizeFilter holds an instrument's order size limits. Spot and derivatives return different
// fields; each field not shared by both says which it belongs to
type AccountInstrumentLotSizeFilter struct {
	// MinOrderQuantity is deprecated for spot, which checks MinOrderAmount instead
	MinOrderQuantity types.Number `json:"minOrderQty"`
	// MaxOrderQuantity is the limit and post-only maximum for linear and inverse, and deprecated for spot in favour of
	// MaxLimitOrderQuantity and MaxMarketOrderQuantity
	MaxOrderQuantity types.Number `json:"maxOrderQty"`
	// QuantityStep, MinNotionalValue and MaxMktOrderQuantity are for linear and inverse
	QuantityStep     types.Number `json:"qtyStep"`
	MinNotionalValue types.Number `json:"minNotionalValue"`
	// MaxMktOrderQuantity is the derivatives market order maximum; its spot counterpart, MaxMarketOrderQuantity, is
	// sent under a different key
	MaxMktOrderQuantity types.Number `json:"maxMktOrderQty"`
	// Deprecated: PostOnlyMaxOrderQuantity is for linear and inverse; MaxOrderQuantity replaces it
	PostOnlyMaxOrderQuantity types.Number `json:"postOnlyMaxOrderQty"`
	// BasePrecision to PostOnlyMaxLimitOrderSize are for spot; MaxOrderAmount is deprecated in favour of
	// MaxLimitOrderQuantity and MaxMarketOrderQuantity
	BasePrecision             types.Number `json:"basePrecision"`
	QuotePrecision            types.Number `json:"quotePrecision"`
	MinOrderAmount            types.Number `json:"minOrderAmt"`
	MaxOrderAmount            types.Number `json:"maxOrderAmt"`
	MaxLimitOrderQuantity     types.Number `json:"maxLimitOrderQty"`
	MaxMarketOrderQuantity    types.Number `json:"maxMarketOrderQty"`
	PostOnlyMaxLimitOrderSize types.Number `json:"postOnlyMaxLimitOrderSize"`
}

// AccountInstrumentRiskParameters holds the ratios that bound an instrument's limit order prices
type AccountInstrumentRiskParameters struct {
	PriceLimitRatioX types.Number `json:"priceLimitRatioX"`
	PriceLimitRatioY types.Number `json:"priceLimitRatioY"`
}

// AccountInstrumentPreListingInfo holds the auction details of a pre-market contract
type AccountInstrumentPreListingInfo struct {
	CurrentAuctionPhase string                             `json:"curAuctionPhase"`
	Phases              []AccountInstrumentPreListingPhase `json:"phases"`
	AuctionFeeInfo      AccountInstrumentAuctionFeeInfo    `json:"auctionFeeInfo"`
	SkipCallAuction     bool                               `json:"skipCallAuction"`
}

// AccountInstrumentPreListingPhase holds the times of one pre-market trading phase
type AccountInstrumentPreListingPhase struct {
	Phase     string              `json:"phase"`
	StartTime PreListingPhaseTime `json:"startTime"`
	EndTime   PreListingPhaseTime `json:"endTime"`
}

// AccountInstrumentAuctionFeeInfo holds a pre-market contract's fee rates
type AccountInstrumentAuctionFeeInfo struct {
	// AuctionFeeRate applies during the auction phase; Bybit's docs note no trading fee is charged until continuous
	// trading starts
	AuctionFeeRate types.Number `json:"auctionFeeRate"`
	// TakerFeeRate and MakerFeeRate apply during continuous trading
	TakerFeeRate types.Number `json:"takerFeeRate"`
	MakerFeeRate types.Number `json:"makerFeeRate"`
}

// ManualBorrowResponse holds the coin and amount borrowed
type ManualBorrowResponse struct {
	Coin   currency.Code `json:"coin"`
	Amount types.Number  `json:"amount"`
}

// ManualRepayRequest holds the parameters of ManualRepay and ManualRepayWithoutAssetConversion
type ManualRepayRequest struct {
	// Coin is required by ManualRepayWithoutAssetConversion; ManualRepay repays every coin without it, which needs
	// RepaymentType ALL
	Coin currency.Code
	// Amount can only be set with Coin; without it the coin's liabilities of RepaymentType are repaid in full
	Amount float64
	// RepaymentType is ALL, FIXED or FLEXIBLE (the default); ALL repays floating rate liabilities first
	RepaymentType string
}

// ManualRepayResponse holds the status of a repayment: P processing, SU success or FA failed
type ManualRepayResponse struct {
	ResultStatus string `json:"resultStatus"`
}

// FeeRateRequest holds the parameters of GetFeeRate; every field is optional
type FeeRateRequest struct {
	// Category linear and inverse return the same rates, for every contract
	Category string
	// Symbol applies to spot, linear and inverse
	Symbol string
	// BaseCoin applies to option
	BaseCoin currency.Code
}

// FeeRateResponse holds the account's trading fee rates
type FeeRateResponse struct {
	// Category is only returned for spot and option
	Category string    `json:"category"`
	List     []FeeRate `json:"list"`
}

// FeeRate holds the fee rates of a symbol, or of an option base coin
type FeeRate struct {
	// Symbol is "" for options
	Symbol string `json:"symbol"`
	// BaseCoin is only returned for options
	BaseCoin        currency.Code `json:"baseCoin"`
	TakerFeeRate    types.Number  `json:"takerFeeRate"`
	MakerFeeRate    types.Number  `json:"makerFeeRate"`
	RPIMakerFeeRate types.Number  `json:"rpiMakerFeeRate"`
}

// CollateralInfoResponse holds the collateral details of each coin
type CollateralInfoResponse struct {
	List []CollateralInfo `json:"list"`
}

// CollateralInfo holds the borrowing and collateral details of a coin in the unified account
type CollateralInfo struct {
	Currency         currency.Code `json:"currency"`
	HourlyBorrowRate types.Number  `json:"hourlyBorrowRate"`
	// MaxBorrowingAmount and AvailableToBorrow are shared by the master account and its subaccounts
	MaxBorrowingAmount types.Number `json:"maxBorrowingAmount"`
	// FreeBorrowingLimit is the interest free borrowing limit, which only applies to borrowing caused by contracts'
	// unrealised losses
	FreeBorrowingLimit types.Number `json:"freeBorrowingLimit"`
	// FreeBorrowAmount is the part of BorrowAmount exempt from interest
	FreeBorrowAmount types.Number `json:"freeBorrowAmount"`
	BorrowAmount     types.Number `json:"borrowAmount"`
	// OtherBorrowAmount is borrowed by the other accounts under the same master account
	OtherBorrowAmount types.Number `json:"otherBorrowAmount"`
	AvailableToBorrow types.Number `json:"availableToBorrow"`
	Borrowable        bool         `json:"borrowable"`
	// BorrowUsageRate is the master and subaccounts' borrowing over MaxBorrowingAmount; 0.5 is 50%
	BorrowUsageRate types.Number `json:"borrowUsageRate"`
	// MarginCollateral is whether the platform allows the coin as collateral; CollateralSwitch, whether the user
	// turned it on, only applies when it is true
	MarginCollateral bool `json:"marginCollateral"`
	CollateralSwitch bool `json:"collateralSwitch"`
	// Deprecated: CollateralRatio is inaccurate since tiered collateral ratios began on 19 February 2025
	CollateralRatio types.Number `json:"collateralRatio"`
	// Deprecated: FreeBorrowingAmount is always ""; FreeBorrowingLimit replaces it
	FreeBorrowingAmount types.Number `json:"freeBorrowingAmount"`
}

// DCPInfoResponse holds the Disconnected-CancelAll-Prevention configuration of each product
type DCPInfoResponse struct {
	DCPInfos []DCPInfo `json:"dcpInfos"`
}

// DCPInfo holds a product's Disconnected-CancelAll-Prevention configuration
type DCPInfo struct {
	// Product is SPOT, DERIVATIVES or OPTIONS
	Product   string `json:"product"`
	DCPStatus string `json:"dcpStatus"`
	// TimeWindow is the trigger window in seconds, between 3 and 300
	TimeWindow types.Number `json:"timeWindow"`
}

// SetMarginModeResponse holds the reasons a margin mode change failed, empty when it succeeded
type SetMarginModeResponse struct {
	Reasons []SetMarginModeReason `json:"reasons"`
}

// SetMarginModeReason holds one reason a margin mode change failed
type SetMarginModeReason struct {
	ReasonCode    string `json:"reasonCode"`
	ReasonMessage string `json:"reasonMsg"`
}

// BorrowHistoryRequest holds the parameters of GetBorrowHistory; every field is optional
type BorrowHistoryRequest struct {
	Currency currency.Code
	// StartTime and EndTime default to the last 30 days; with only one set, the range is the 30 days from or to it
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
	Cursor    string
}

// BorrowHistoryResponse holds a page of interest records
type BorrowHistoryResponse struct {
	List           []BorrowHistoryRecord `json:"list"`
	NextPageCursor string                `json:"nextPageCursor"`
}

// BorrowHistoryRecord holds one interest record
type BorrowHistoryRecord struct {
	Currency    currency.Code `json:"currency"`
	CreatedTime types.Time    `json:"createdTime"`
	// BorrowCost is the interest charged
	BorrowCost       types.Number `json:"borrowCost"`
	HourlyBorrowRate types.Number `json:"hourlyBorrowRate"`
	// InterestBearingBorrowSize keeps Bybit's capitalised key
	InterestBearingBorrowSize types.Number `json:"InterestBearingBorrowSize"`
	CostExemption             types.Number `json:"costExemption"`
	BorrowAmount              types.Number `json:"borrowAmount"`
	UnrealisedLoss            types.Number `json:"unrealisedLoss"`
	// FreeBorrowedAmount is the interest free part of BorrowAmount
	FreeBorrowedAmount types.Number `json:"freeBorrowedAmount"`
}

// SetCollateralCoinRequest holds one coin's collateral switch for BatchSetCollateralCoin; USDT and USDC cannot be set
type SetCollateralCoinRequest struct {
	Coin             currency.Code
	CollateralSwitch bool
}

// BatchSetCollateralCoinResponse holds the collateral switch of each coin set
type BatchSetCollateralCoinResponse struct {
	List []CollateralCoinSwitch `json:"list"`
}

// CollateralCoinSwitch holds a coin's collateral switch
type CollateralCoinSwitch struct {
	Coin currency.Code `json:"coin"`
	// CollateralSwitch is ON or OFF
	CollateralSwitch string `json:"collateralSwitch"`
}

// CoinGreeksResponse holds the account's option greeks per base coin
type CoinGreeksResponse struct {
	List []CoinGreeks `json:"list"`
}

// CoinGreeks holds the account's option greeks of a base coin
type CoinGreeks struct {
	BaseCoin   currency.Code `json:"baseCoin"`
	TotalDelta types.Number  `json:"totalDelta"`
	TotalGamma types.Number  `json:"totalGamma"`
	TotalVega  types.Number  `json:"totalVega"`
	TotalTheta types.Number  `json:"totalTheta"`
}

// MMPStateResponse holds the Market Maker Protection state of a base coin
type MMPStateResponse struct {
	Result []MMPState `json:"result"`
}

// MMPState holds a base coin's Market Maker Protection configuration and state
type MMPState struct {
	BaseCoin   currency.Code `json:"baseCoin"`
	MMPEnabled bool          `json:"mmpEnabled"`
	// Window and FrozenPeriod are in milliseconds
	Window        types.Number `json:"window"`
	FrozenPeriod  types.Number `json:"frozenPeriod"`
	QuantityLimit types.Number `json:"qtyLimit"`
	DeltaLimit    types.Number `json:"deltaLimit"`
	VegaLimit     types.Number `json:"vegaLimit"`
	// MMPFrozenUntil only applies while MMPFrozen
	MMPFrozenUntil types.Time `json:"mmpFrozenUntil"`
	MMPFrozen      bool       `json:"mmpFrozen"`
}

// SetMMPRequest holds the parameters of SetMMP; Market Maker Protection applies to options only
type SetMMPRequest struct {
	BaseCoin currency.Code
	// Window is sent in milliseconds
	Window time.Duration
	// FrozenPeriod is sent in milliseconds; 0 keeps trading frozen until ResetMMP
	FrozenPeriod time.Duration
	// QuantityLimit, DeltaLimit and VegaLimit are positive with up to two decimal places
	QuantityLimit float64
	DeltaLimit    float64
	// VegaLimit is optional and defaults to 500000
	VegaLimit float64
}

// SMPGroupIDResponse holds the account's Self-Match Prevention group
type SMPGroupIDResponse struct {
	// SMPGroup is 0 when the account has no group. Bybit sends smpGroup as a number here and as a string on its order
	// endpoints, so either is accepted
	SMPGroup types.Number `json:"smpGroup"`
}

// TradeBehaviourConfigResponse holds the account's trade behaviour settings
type TradeBehaviourConfigResponse struct {
	// LimitPriceActionSpot true adjusts a spot limit order priced beyond the price limit to the nearest allowed price,
	// false rejects it
	LimitPriceActionSpot bool `json:"lpaSpot"`
	// LimitPriceActionPerpetual true rejects a futures limit order priced beyond the price limit, false adjusts it to
	// the nearest allowed price
	LimitPriceActionPerpetual     bool `json:"lpaPerp"`
	DeltaEnable                   bool `json:"deltaEnable"`
	SpotMNTFeeDeductionEnabled    bool `json:"smsef"`
	FuturesMNTFeeDeductionEnabled bool `json:"fmsef"`
	// SMPType is 0 no Self-Match Prevention, 1 cancel taker, 2 cancel maker or 3 cancel both
	SMPType uint64 `json:"smpType"`
}

// SetDeltaNeutralModeResponse holds the outcome of a delta neutral mode change
type SetDeltaNeutralModeResponse struct {
	// ResultStatus is success or failed. Bybit's docs type it as an integer but list string values, and their example
	// returns an empty result
	ResultStatus string `json:"resultStatus"`
}

// RepayLiabilityResponse holds the coins used to repay liabilities
type RepayLiabilityResponse struct {
	List []LiabilityRepayment `json:"list"`
}

// LiabilityRepayment holds the quantity of a coin used to repay liabilities, taken in the liquidation order
type LiabilityRepayment struct {
	Coin              currency.Code `json:"coin"`
	RepaymentQuantity types.Number  `json:"repaymentQty"`
}

// UnifiedAccountUpgradeResponse holds the status of an upgrade to UTA 2.0 Pro
type UnifiedAccountUpgradeResponse struct {
	// UnifiedUpdateStatus is FAIL, PROCESS or SUCCESS
	UnifiedUpdateStatus string `json:"unifiedUpdateStatus"`
	// UnifiedUpdateMessage is only returned when the upgrade failed
	UnifiedUpdateMessage UnifiedUpdateMessage `json:"unifiedUpdateMsg"`
}

// UnifiedUpdateMessage holds the reasons an upgrade failed
type UnifiedUpdateMessage struct {
	Message []string `json:"msg"`
}

// OptionAssetInfoResponse holds the option asset details of each coin
type OptionAssetInfoResponse struct {
	Result []OptionAssetInfo `json:"result"`
}

// OptionAssetInfo holds a coin's option profit and loss, delta and margin
type OptionAssetInfo struct {
	Coin currency.Code `json:"coin"`
	// TotalDelta only includes the delta of option positions
	TotalDelta         types.Number `json:"totalDelta"`
	TotalRealisedPNL   types.Number `json:"totalRPL"`
	TotalUnrealisedPNL types.Number `json:"totalUPL"`
	// AssetInitialMargin and AssetMaintenanceMargin include the margin of option open orders
	AssetInitialMargin     types.Number `json:"assetIM"`
	AssetMaintenanceMargin types.Number `json:"assetMM"`
	SendTime               types.Time   `json:"sendTime"`
}

// PayInfoResponse holds the collateral available to repay liabilities and the borrowing of the requested coin
type PayInfoResponse struct {
	CollateralInfo PayCollateralInfo `json:"collateralInfo"`
	BorrowInfo     PayBorrowInfo     `json:"borrowInfo"`
}

// PayCollateralInfo holds the collateral available to repay liabilities
type PayCollateralInfo struct {
	CollateralList []PayCollateral `json:"collateralList"`
}

// PayCollateral holds a coin available to repay liabilities
type PayCollateral struct {
	Coin          currency.Code `json:"coin"`
	AvailableSize types.Number  `json:"availableSize"`
	// AvailableValue is in USD
	AvailableValue types.Number `json:"availableValue"`
	// CoinScale is the coin's precision
	CoinScale       uint64       `json:"coinScale"`
	BorrowSize      types.Number `json:"borrowSize"`
	SpotHedgeAmount types.Number `json:"spotHedgeAmount"`
	AssetFrozen     types.Number `json:"assetFrozen"`
}

// PayBorrowInfo holds the borrowing of the requested coin, or of every coin
type PayBorrowInfo struct {
	// Coin is only returned when a coin is requested
	Coin       currency.Code `json:"coin"`
	BorrowSize types.Number  `json:"borrowSize"`
	// BorrowValue is in USD
	BorrowValue      types.Number `json:"borrowValue"`
	AssetFrozen      types.Number `json:"assetFrozen"`
	AvailableBalance types.Number `json:"availableBalance"`
}

// TradeInfoForAnalysisRequest holds the parameters of GetTradeInfoForAnalysis
type TradeInfoForAnalysisRequest struct {
	// Symbol is required
	Symbol    string
	StartTime time.Time
	EndTime   time.Time
}

// TradeInfoForAnalysisResponse holds aggregated spot trade statistics for a symbol
type TradeInfoForAnalysisResponse struct {
	SymbolRealisedPNL        types.Number        `json:"symbolRnl"`
	NetExecutedQuantity      types.Number        `json:"netExecQty"`
	SumExecutedValue         types.Number        `json:"sumExecValue"`
	SumExecutedQuantity      types.Number        `json:"sumExecQty"`
	AverageBuyExecutedPrice  types.Number        `json:"avgBuyExecPrice"`
	SumBuyExecutedValue      types.Number        `json:"sumBuyExecValue"`
	SumBuyExecutedQuantity   types.Number        `json:"sumBuyExecQty"`
	SumBuyExecutedFee        types.Number        `json:"sumBuyExecFee"`
	SumBuyOrderQuantity      types.Number        `json:"sumBuyOrderQty"`
	AverageSellExecutedPrice types.Number        `json:"avgSellExecPrice"`
	SumSellExecutedValue     types.Number        `json:"sumSellExecValue"`
	SumSellExecutedQuantity  types.Number        `json:"sumSellExecQty"`
	SumSellExecutedFee       types.Number        `json:"sumSellExecFee"`
	SumSellOrderQuantity     types.Number        `json:"sumSellOrderQty"`
	MaxMarginVersion         uint64              `json:"maxMarginVersion"`
	BaseCoin                 currency.Code       `json:"baseCoin"`
	SettleCoin               currency.Code       `json:"settleCoin"`
	SumPriceList             []TradeInfoDailySum `json:"sumPriceList"`
}

// TradeInfoDailySum holds a day's aggregated execution values
type TradeInfoDailySum struct {
	// Day is the date as Bybit formats it; the docs do not give the format
	Day                  string       `json:"day"`
	SumBuyExecutedValue  types.Number `json:"sumBuyExecValue"`
	SumSellExecutedValue types.Number `json:"sumSellExecValue"`
	SumExecutedValue     types.Number `json:"sumExecValue"`
}
