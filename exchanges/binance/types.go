package binance

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/types"
	"github.com/thrasher-corp/gocryptotrader/types/decimal"
)

// Side is the side parameter of an order
type Side string

// Order sides, the side parameter of the order endpoints; the portfolio margin requests that take an order.Side take
// order.Buy or order.Sell instead
const (
	SideBuy  Side = "BUY"
	SideSell Side = "SELL"
)

// OrderType is the type parameter of an order, and the strategyType parameter of a portfolio margin COIN-M
// conditional order
type OrderType string

// Order types, the type parameter of the order endpoints. Spot and margin orders, portfolio margin's included, take
// LIMIT, MARKET, STOP_LOSS, STOP_LOSS_LIMIT, TAKE_PROFIT, TAKE_PROFIT_LIMIT and LIMIT_MAKER. USDⓈ-M and COIN-M
// orders take LIMIT, MARKET, STOP, STOP_MARKET, TAKE_PROFIT, TAKE_PROFIT_MARKET and TRAILING_STOP_MARKET, the
// conditional ones as algo orders; portfolio margin futures orders take LIMIT and MARKET, and its algo and conditional
// orders the conditional ones. Options orders take LIMIT only. TAKE_PROFIT differs by product: a spot or margin
// TAKE_PROFIT order places a market order when it triggers, while a futures one places a limit order
const (
	OrderTypeLimit              OrderType = "LIMIT"
	OrderTypeMarket             OrderType = "MARKET"
	OrderTypeStopLoss           OrderType = "STOP_LOSS"
	OrderTypeStopLossLimit      OrderType = "STOP_LOSS_LIMIT"
	OrderTypeTakeProfit         OrderType = "TAKE_PROFIT"
	OrderTypeTakeProfitLimit    OrderType = "TAKE_PROFIT_LIMIT"
	OrderTypeLimitMaker         OrderType = "LIMIT_MAKER"
	OrderTypeStop               OrderType = "STOP"
	OrderTypeStopMarket         OrderType = "STOP_MARKET"
	OrderTypeTakeProfitMarket   OrderType = "TAKE_PROFIT_MARKET"
	OrderTypeTrailingStopMarket OrderType = "TRAILING_STOP_MARKET"
)

// TimeInForce is how long an order stays on the order book, the timeInForce parameter of an order
type TimeInForce string

// Time in force values. Every product takes GTC, IOC and FOK; GTX (post only) applies to USDⓈ-M, COIN-M, options and
// portfolio margin futures orders, GTD to USDⓈ-M and portfolio margin USDⓈ-M orders, and RPI (retail price
// improvement) to USDⓈ-M orders
const (
	TimeInForceGoodTillCancel         TimeInForce = "GTC"
	TimeInForceImmediateOrCancel      TimeInForce = "IOC"
	TimeInForceFillOrKill             TimeInForce = "FOK"
	TimeInForceGoodTillCrossing       TimeInForce = "GTX"
	TimeInForceGoodTillDate           TimeInForce = "GTD"
	TimeInForceRetailPriceImprovement TimeInForce = "RPI"
)

// SelfTradePreventionMode is the selfTradePreventionMode parameter of an order
type SelfTradePreventionMode string

// Self-trade prevention modes. Not every endpoint takes every mode: DECREMENT and TRANSFER are spot only, and some
// futures endpoints do not take NONE
const (
	SelfTradePreventionModeNone        SelfTradePreventionMode = "NONE"
	SelfTradePreventionModeExpireTaker SelfTradePreventionMode = "EXPIRE_TAKER"
	SelfTradePreventionModeExpireMaker SelfTradePreventionMode = "EXPIRE_MAKER"
	SelfTradePreventionModeExpireBoth  SelfTradePreventionMode = "EXPIRE_BOTH"
	SelfTradePreventionModeDecrement   SelfTradePreventionMode = "DECREMENT"
	SelfTradePreventionModeTransfer    SelfTradePreventionMode = "TRANSFER"
)

// NewOrderResponseType is the newOrderRespType parameter of an order
type NewOrderResponseType string

// New order response types. A spot or margin ACK response carries the order's identifiers, RESULT adds its state and
// FULL, which only spot and margin orders take, adds its fills. Futures and options orders take ACK and RESULT: a
// futures ACK response carries the order as accepted, and RESULT returns a MARKET order, or a LIMIT order that cannot
// rest, in its final status
const (
	NewOrderResponseTypeAcknowledgement NewOrderResponseType = "ACK"
	NewOrderResponseTypeResult          NewOrderResponseType = "RESULT"
	NewOrderResponseTypeFull            NewOrderResponseType = "FULL"
)

// PegPriceType is the pegPriceType parameter of a spot pegged order
type PegPriceType string

// Peg price types of spot pegged orders, which follow the best price on the order's own side (PRIMARY_PEG) or on the
// other side (MARKET_PEG)
const (
	PegPriceTypePrimaryPeg PegPriceType = "PRIMARY_PEG"
	PegPriceTypeMarketPeg  PegPriceType = "MARKET_PEG"
)

// PegOffsetType is the pegOffsetType parameter of a spot pegged order
type PegOffsetType string

// PegOffsetTypePriceLevel is the only peg offset type of spot pegged orders, which offsets the pegged price by order
// book levels
const PegOffsetTypePriceLevel PegOffsetType = "PRICE_LEVEL"

// CancelReplaceMode is the cancelReplaceMode parameter of a spot cancel replace order
type CancelReplaceMode string

// Cancel replace modes: STOP_ON_FAILURE does not attempt the new order when the cancel fails, and ALLOW_FAILURE
// attempts it regardless
const (
	CancelReplaceModeStopOnFailure CancelReplaceMode = "STOP_ON_FAILURE"
	CancelReplaceModeAllowFailure  CancelReplaceMode = "ALLOW_FAILURE"
)

// CancelRestrictions is the cancelRestrictions parameter of a spot cancel or cancel replace order
type CancelRestrictions string

// Cancel restrictions, which cancel a spot order only while it is new or only while it is partially filled
const (
	CancelRestrictionsOnlyNew             CancelRestrictions = "ONLY_NEW"
	CancelRestrictionsOnlyPartiallyFilled CancelRestrictions = "ONLY_PARTIALLY_FILLED"
)

// OrderRateLimitExceededMode is the orderRateLimitExceededMode parameter of a spot cancel replace order
type OrderRateLimitExceededMode string

// Order rate limit exceeded modes of cancel replace orders: DO_NOTHING, the default, attempts the cancel only while the
// account is within its unfilled order count limit, and CANCEL_ONLY attempts it regardless
const (
	OrderRateLimitExceededModeDoNothing  OrderRateLimitExceededMode = "DO_NOTHING"
	OrderRateLimitExceededModeCancelOnly OrderRateLimitExceededMode = "CANCEL_ONLY"
)

// SideEffectType is the sideEffectType parameter of a margin order
type SideEffectType string

// Side effect types of margin orders, which borrow for the order (MARGIN_BUY), repay debt with its proceeds
// (AUTO_REPAY) or both (AUTO_BORROW_REPAY)
const (
	SideEffectTypeNoSideEffect    SideEffectType = "NO_SIDE_EFFECT"
	SideEffectTypeMarginBuy       SideEffectType = "MARGIN_BUY"
	SideEffectTypeAutoRepay       SideEffectType = "AUTO_REPAY"
	SideEffectTypeAutoBorrowRepay SideEffectType = "AUTO_BORROW_REPAY"
)

// SymbolStatus is the symbolStatus parameter of spot market data requests
type SymbolStatus string

// Spot symbol trading statuses, which filter the symbols a market data request returns
const (
	SymbolStatusTrading    SymbolStatus = "TRADING"
	SymbolStatusHalt       SymbolStatus = "HALT"
	SymbolStatusBreak      SymbolStatus = "BREAK"
	SymbolStatusCancelOnly SymbolStatus = "CANCEL_ONLY"
)

// TickerType is the type parameter of the spot ticker requests
type TickerType string

// Ticker types of the 24hr, rolling window and trading day tickers, over REST and the WebSocket API: FULL, the
// default, or MINI, which leaves out the bid, ask and weighted average price fields
const (
	TickerTypeFull TickerType = "FULL"
	TickerTypeMini TickerType = "MINI"
)

// Permission is a value of the permissions parameter of exchange information requests, and an account or symbol
// permission the replies report
type Permission string

// Account and symbol permissions. Exchange information lists the symbols with SPOT, MARGIN or LEVERAGED permission when
// no permission is requested
const (
	PermissionSpot            Permission = "SPOT"
	PermissionMargin          Permission = "MARGIN"
	PermissionLeveraged       Permission = "LEVERAGED"
	PermissionTradingGroup002 Permission = "TRD_GRP_002"
	PermissionTradingGroup003 Permission = "TRD_GRP_003"
	PermissionTradingGroup004 Permission = "TRD_GRP_004"
	PermissionTradingGroup005 Permission = "TRD_GRP_005"
	PermissionTradingGroup006 Permission = "TRD_GRP_006"
	PermissionTradingGroup007 Permission = "TRD_GRP_007"
	PermissionTradingGroup008 Permission = "TRD_GRP_008"
	PermissionTradingGroup009 Permission = "TRD_GRP_009"
	PermissionTradingGroup010 Permission = "TRD_GRP_010"
	PermissionTradingGroup011 Permission = "TRD_GRP_011"
	PermissionTradingGroup012 Permission = "TRD_GRP_012"
	PermissionTradingGroup013 Permission = "TRD_GRP_013"
	PermissionTradingGroup014 Permission = "TRD_GRP_014"
	PermissionTradingGroup015 Permission = "TRD_GRP_015"
	PermissionTradingGroup016 Permission = "TRD_GRP_016"
	PermissionTradingGroup017 Permission = "TRD_GRP_017"
	PermissionTradingGroup018 Permission = "TRD_GRP_018"
	PermissionTradingGroup019 Permission = "TRD_GRP_019"
	PermissionTradingGroup020 Permission = "TRD_GRP_020"
	PermissionTradingGroup021 Permission = "TRD_GRP_021"
	PermissionTradingGroup022 Permission = "TRD_GRP_022"
	PermissionTradingGroup023 Permission = "TRD_GRP_023"
	PermissionTradingGroup024 Permission = "TRD_GRP_024"
	PermissionTradingGroup025 Permission = "TRD_GRP_025"
)

// MarginAccountType is the type parameter of the margin available inventory and manual liquidation requests
type MarginAccountType string

// Margin account types: MARGIN for the cross margin account or ISOLATED for isolated margin accounts
const (
	MarginAccountTypeMargin   MarginAccountType = "MARGIN"
	MarginAccountTypeIsolated MarginAccountType = "ISOLATED"
)

// BorrowRepayType is the type parameter of margin borrow and repay requests and of their history queries
type BorrowRepayType string

// Margin borrow and repay types
const (
	BorrowRepayTypeBorrow BorrowRepayType = "BORROW"
	BorrowRepayTypeRepay  BorrowRepayType = "REPAY"
)

// CrossMarginTransferType is the type parameter of the cross margin transfer history, and the direction its records
// report
type CrossMarginTransferType string

// Cross margin transfer types: into (ROLL_IN) or out of (ROLL_OUT) the cross margin account
const (
	CrossMarginTransferTypeRollIn  CrossMarginTransferType = "ROLL_IN"
	CrossMarginTransferTypeRollOut CrossMarginTransferType = "ROLL_OUT"
)

// MarginCapitalFlowType is the type parameter of the margin capital flow, and the type its records report
type MarginCapitalFlowType string

// Margin capital flow types
const (
	MarginCapitalFlowTypeTransfer            MarginCapitalFlowType = "TRANSFER"
	MarginCapitalFlowTypeBorrow              MarginCapitalFlowType = "BORROW"
	MarginCapitalFlowTypeRepay               MarginCapitalFlowType = "REPAY"
	MarginCapitalFlowTypeBuyIncome           MarginCapitalFlowType = "BUY_INCOME"
	MarginCapitalFlowTypeBuyExpense          MarginCapitalFlowType = "BUY_EXPENSE"
	MarginCapitalFlowTypeSellIncome          MarginCapitalFlowType = "SELL_INCOME"
	MarginCapitalFlowTypeSellExpense         MarginCapitalFlowType = "SELL_EXPENSE"
	MarginCapitalFlowTypeTradingCommission   MarginCapitalFlowType = "TRADING_COMMISSION"
	MarginCapitalFlowTypeBuyLiquidation      MarginCapitalFlowType = "BUY_LIQUIDATION"
	MarginCapitalFlowTypeSellLiquidation     MarginCapitalFlowType = "SELL_LIQUIDATION"
	MarginCapitalFlowTypeRepayLiquidation    MarginCapitalFlowType = "REPAY_LIQUIDATION"
	MarginCapitalFlowTypeOtherLiquidation    MarginCapitalFlowType = "OTHER_LIQUIDATION"
	MarginCapitalFlowTypeLiquidationFee      MarginCapitalFlowType = "LIQUIDATION_FEE"
	MarginCapitalFlowTypeSmallBalanceConvert MarginCapitalFlowType = "SMALL_BALANCE_CONVERT"
	MarginCapitalFlowTypeCommissionReturn    MarginCapitalFlowType = "COMMISSION_RETURN"
	MarginCapitalFlowTypeSmallConvert        MarginCapitalFlowType = "SMALL_CONVERT"
)

// The values below are for wallet, sub-account and broker endpoints

// AccountType is an account type parameter of the wallet, sub-account and broker endpoints
type AccountType string

// Account types. The dust endpoints take SPOT or MARGIN; managed sub-account transfer logs take SPOT, MARGIN,
// ISOLATED_MARGIN, USDT_FUTURE or COIN_FUTURE, and sub-account universal transfers those or ALPHA, which moves funds
// between Alpha accounts only; managed sub-account futures asset details take USDT_FUTURE or COIN_FUTURE, and margin
// asset details MARGIN or ISOLATED_MARGIN; broker universal transfers take SPOT, USDT_FUTURE or COIN_FUTURE
const (
	AccountTypeSpot           AccountType = "SPOT"
	AccountTypeMargin         AccountType = "MARGIN"
	AccountTypeIsolatedMargin AccountType = "ISOLATED_MARGIN"
	AccountTypeUSDTFuture     AccountType = "USDT_FUTURE"
	AccountTypeCoinFuture     AccountType = "COIN_FUTURE"
	AccountTypeAlpha          AccountType = "ALPHA"
)

// AccountSnapshotType is the type parameter of the daily account snapshot request; snapshots report it in lowercase
type AccountSnapshotType string

// Account snapshot types
const (
	AccountSnapshotTypeSpot    AccountSnapshotType = "SPOT"
	AccountSnapshotTypeMargin  AccountSnapshotType = "MARGIN"
	AccountSnapshotTypeFutures AccountSnapshotType = "FUTURES"
)

// UniversalTransferType is the type parameter of user universal transfers and their history, naming the account
// transferred from and the account transferred to
type UniversalTransferType string

// Universal transfer types. MAIN is the spot account, UMFUTURE and CMFUTURE the USDⓈ-M and COIN-M futures accounts,
// ISOLATEDMARGIN an isolated margin account, and OPTION the options account
const (
	UniversalTransferTypeMainUSDTMarginedFutures      UniversalTransferType = "MAIN_UMFUTURE"
	UniversalTransferTypeMainCoinMarginedFutures      UniversalTransferType = "MAIN_CMFUTURE"
	UniversalTransferTypeMainMargin                   UniversalTransferType = "MAIN_MARGIN"
	UniversalTransferTypeUSDTMarginedFuturesMain      UniversalTransferType = "UMFUTURE_MAIN"
	UniversalTransferTypeUSDTMarginedFuturesMargin    UniversalTransferType = "UMFUTURE_MARGIN"
	UniversalTransferTypeCoinMarginedFuturesMain      UniversalTransferType = "CMFUTURE_MAIN"
	UniversalTransferTypeCoinMarginedFuturesMargin    UniversalTransferType = "CMFUTURE_MARGIN"
	UniversalTransferTypeMarginMain                   UniversalTransferType = "MARGIN_MAIN"
	UniversalTransferTypeMarginUSDTMarginedFutures    UniversalTransferType = "MARGIN_UMFUTURE"
	UniversalTransferTypeMarginCoinMarginedFutures    UniversalTransferType = "MARGIN_CMFUTURE"
	UniversalTransferTypeIsolatedMarginMargin         UniversalTransferType = "ISOLATEDMARGIN_MARGIN"
	UniversalTransferTypeMarginIsolatedMargin         UniversalTransferType = "MARGIN_ISOLATEDMARGIN"
	UniversalTransferTypeIsolatedMarginIsolatedMargin UniversalTransferType = "ISOLATEDMARGIN_ISOLATEDMARGIN"
	UniversalTransferTypeMainFunding                  UniversalTransferType = "MAIN_FUNDING"
	UniversalTransferTypeFundingMain                  UniversalTransferType = "FUNDING_MAIN"
	UniversalTransferTypeFundingUSDTMarginedFutures   UniversalTransferType = "FUNDING_UMFUTURE"
	UniversalTransferTypeUSDTMarginedFuturesFunding   UniversalTransferType = "UMFUTURE_FUNDING"
	UniversalTransferTypeMarginFunding                UniversalTransferType = "MARGIN_FUNDING"
	UniversalTransferTypeFundingMargin                UniversalTransferType = "FUNDING_MARGIN"
	UniversalTransferTypeFundingCoinMarginedFutures   UniversalTransferType = "FUNDING_CMFUTURE"
	UniversalTransferTypeCoinMarginedFuturesFunding   UniversalTransferType = "CMFUTURE_FUNDING"
	UniversalTransferTypeMainOption                   UniversalTransferType = "MAIN_OPTION"
	UniversalTransferTypeOptionMain                   UniversalTransferType = "OPTION_MAIN"
	UniversalTransferTypeUSDTMarginedFuturesOption    UniversalTransferType = "UMFUTURE_OPTION"
	UniversalTransferTypeOptionUSDTMarginedFutures    UniversalTransferType = "OPTION_UMFUTURE"
	UniversalTransferTypeMarginOption                 UniversalTransferType = "MARGIN_OPTION"
	UniversalTransferTypeOptionMargin                 UniversalTransferType = "OPTION_MARGIN"
	UniversalTransferTypeFundingOption                UniversalTransferType = "FUNDING_OPTION"
	UniversalTransferTypeOptionFunding                UniversalTransferType = "OPTION_FUNDING"
	UniversalTransferTypeMainPortfolioMargin          UniversalTransferType = "MAIN_PORTFOLIO_MARGIN"
	UniversalTransferTypePortfolioMarginMain          UniversalTransferType = "PORTFOLIO_MARGIN_MAIN"
)

// TransferDirection is the transfers parameter of the managed sub-account transfer logs
type TransferDirection string

// Transfer directions: transfers out of (FROM) or into (TO) the managed sub-account. The investor and trading parent
// logs document them in upper case, and the trading team sub-account's own log in lower case, from or to
const (
	TransferDirectionFrom TransferDirection = "FROM"
	TransferDirectionTo   TransferDirection = "TO"
)

// DelegationType is the type parameter of the user delegation history
type DelegationType string

// Delegation types
const (
	DelegationTypeDelegate   DelegationType = "DELEGATE"
	DelegationTypeUndelegate DelegationType = "UNDELEGATE"
)

// HistoricalDataType is the dataType parameter of the futures historical order book data download link request
type HistoricalDataType string

// Historical data types: tick level order book data (T_DEPTH) or order book snapshots (S_DEPTH)
const (
	HistoricalDataTypeTickDepth     HistoricalDataType = "T_DEPTH"
	HistoricalDataTypeSnapshotDepth HistoricalDataType = "S_DEPTH"
)

// The values below are for earn, loan, convert, auto-invest and algo trading endpoints

// SimpleEarnAccount is the sourceAccount parameter of Simple Earn subscriptions and the destAccount parameter of
// flexible product redemptions
type SimpleEarnAccount string

// Simple Earn accounts: subscriptions take SPOT (the default), FUND or ALL, and redemptions SPOT (the default) or FUND
const (
	SimpleEarnAccountSpot SimpleEarnAccount = "SPOT"
	SimpleEarnAccountFund SimpleEarnAccount = "FUND"
	SimpleEarnAccountAll  SimpleEarnAccount = "ALL"
)

// RedeemTo is the redeemTo parameter of locked Simple Earn subscriptions and redemption options
type RedeemTo string

// Where a locked product is redeemed to when it ends: the spot account or a flexible product
const (
	RedeemToSpot     RedeemTo = "SPOT"
	RedeemToFlexible RedeemTo = "FLEXIBLE"
)

// APRPeriod is the aprPeriod parameter of the Simple Earn rate history
type APRPeriod string

// Annual percentage rate periods
const (
	APRPeriodDay  APRPeriod = "DAY"
	APRPeriodYear APRPeriod = "YEAR"
)

// FlexibleRewardType is the type parameter of the flexible Simple Earn rewards history
type FlexibleRewardType string

// Flexible reward types
const (
	FlexibleRewardTypeBonus    FlexibleRewardType = "BONUS"
	FlexibleRewardTypeRealtime FlexibleRewardType = "REALTIME"
	FlexibleRewardTypeRewards  FlexibleRewardType = "REWARDS"
	FlexibleRewardTypeAll      FlexibleRewardType = "ALL"
)

// BoostRewardsType is the type parameter of the SOL staking boost rewards history
type BoostRewardsType string

// Boost rewards types
const (
	BoostRewardsTypeClaim      BoostRewardsType = "CLAIM"
	BoostRewardsTypeDistribute BoostRewardsType = "DISTRIBUTE"
)

// LTVAdjustmentDirection is the direction parameter of a flexible loan LTV adjustment, and the direction its result
// reports
type LTVAdjustmentDirection string

// LTV adjustment directions: adding collateral (ADDITIONAL) or removing it (REDUCED)
const (
	LTVAdjustmentDirectionAdditional LTVAdjustmentDirection = "ADDITIONAL"
	LTVAdjustmentDirectionReduced    LTVAdjustmentDirection = "REDUCED"
)

// CryptoLoanIncomeType is the type parameter of the crypto loan income history, and the type its records report
type CryptoLoanIncomeType string

// Crypto loan income types, in Binance's camel case
const (
	CryptoLoanIncomeTypeBorrowIn                         CryptoLoanIncomeType = "borrowIn"
	CryptoLoanIncomeTypeCollateralSpent                  CryptoLoanIncomeType = "collateralSpent"
	CryptoLoanIncomeTypeRepayAmount                      CryptoLoanIncomeType = "repayAmount"
	CryptoLoanIncomeTypeCollateralReturn                 CryptoLoanIncomeType = "collateralReturn"
	CryptoLoanIncomeTypeAddCollateral                    CryptoLoanIncomeType = "addCollateral"
	CryptoLoanIncomeTypeRemoveCollateral                 CryptoLoanIncomeType = "removeCollateral"
	CryptoLoanIncomeTypeCollateralReturnAfterLiquidation CryptoLoanIncomeType = "collateralReturnAfterLiquidation"
)

// ConvertWalletType is the walletType parameter of convert quotes and limit orders, the wallets paid from
type ConvertWalletType string

// Convert wallet types: SPOT, the default, FUNDING, EARN, or a combination of them
const (
	ConvertWalletTypeSpot            ConvertWalletType = "SPOT"
	ConvertWalletTypeFunding         ConvertWalletType = "FUNDING"
	ConvertWalletTypeEarn            ConvertWalletType = "EARN"
	ConvertWalletTypeSpotFunding     ConvertWalletType = "SPOT_FUNDING"
	ConvertWalletTypeFundingEarn     ConvertWalletType = "FUNDING_EARN"
	ConvertWalletTypeSpotFundingEarn ConvertWalletType = "SPOT_FUNDING_EARN"
	ConvertWalletTypeSpotEarn        ConvertWalletType = "SPOT_EARN"
)

// ExpiredType is the expiredType parameter of a convert limit order, how long it lasts
type ExpiredType string

// Convert limit order expiries
const (
	ExpiredType1Day   ExpiredType = "1_D"
	ExpiredType3Days  ExpiredType = "3_D"
	ExpiredType7Days  ExpiredType = "7_D"
	ExpiredType30Days ExpiredType = "30_D"
)

// QuoteValidTime is the validTime parameter of a convert quote request, how long the quote is valid
type QuoteValidTime string

// Convert quote validity periods; 10s is the default
const (
	QuoteValidTime10Seconds QuoteValidTime = "10s"
	QuoteValidTime30Seconds QuoteValidTime = "30s"
	QuoteValidTime1Minute   QuoteValidTime = "1m"
)

// OptionType is the optionType parameter of the dual investment product list, and the option type dual investment
// products and positions, options exchange information and the options new symbol stream report
type OptionType string

// Option types
const (
	OptionTypeCall OptionType = "CALL"
	OptionTypePut  OptionType = "PUT"
)

// DualInvestmentPositionStatus is the status parameter of the dual investment positions request, and the status
// positions report
type DualInvestmentPositionStatus string

// Dual investment position statuses
const (
	DualInvestmentPositionStatusPending         DualInvestmentPositionStatus = "PENDING"
	DualInvestmentPositionStatusPurchaseSuccess DualInvestmentPositionStatus = "PURCHASE_SUCCESS"
	DualInvestmentPositionStatusSettled         DualInvestmentPositionStatus = "SETTLED"
	DualInvestmentPositionStatusPurchaseFail    DualInvestmentPositionStatus = "PURCHASE_FAIL"
	DualInvestmentPositionStatusRefunding       DualInvestmentPositionStatus = "REFUNDING"
	DualInvestmentPositionStatusRefundSuccess   DualInvestmentPositionStatus = "REFUND_SUCCESS"
	DualInvestmentPositionStatusSettling        DualInvestmentPositionStatus = "SETTLING"
)

// AutoInvestPlanType is the planType parameter of the auto-invest endpoints, and the plan type plans report
type AutoInvestPlanType string

// Auto-invest plan types; ALL is for the subscription transaction history only
const (
	AutoInvestPlanTypeSingle    AutoInvestPlanType = "SINGLE"
	AutoInvestPlanTypePortfolio AutoInvestPlanType = "PORTFOLIO"
	AutoInvestPlanTypeIndex     AutoInvestPlanType = "INDEX"
	AutoInvestPlanTypeAll       AutoInvestPlanType = "ALL"
)

// AutoInvestPlanStatus is the status parameter of an auto-invest plan status change, and the status plans report
type AutoInvestPlanStatus string

// Auto-invest plan statuses
const (
	AutoInvestPlanStatusOngoing AutoInvestPlanStatus = "ONGOING"
	AutoInvestPlanStatusPaused  AutoInvestPlanStatus = "PAUSED"
	AutoInvestPlanStatusRemoved AutoInvestPlanStatus = "REMOVED"
)

// AutoInvestSourceType is the sourceType parameter of the auto-invest endpoints, the site a plan's funds come from
type AutoInvestSourceType string

// Auto-invest source types: Binance (MAIN_SITE) or Binance Turkey (TR)
const (
	AutoInvestSourceTypeMainSite AutoInvestSourceType = "MAIN_SITE"
	AutoInvestSourceTypeTurkey   AutoInvestSourceType = "TR"
)

// SourceAssetUsageType is the usageType parameter of the auto-invest source asset list
type SourceAssetUsageType string

// Source asset usage types
const (
	SourceAssetUsageTypeRecurring SourceAssetUsageType = "RECURRING"
	SourceAssetUsageTypeOneTime   SourceAssetUsageType = "ONE_TIME"
)

// SubscriptionCycle is the subscriptionCycle parameter of an auto-invest plan, and the cycle plans report
type SubscriptionCycle string

// Auto-invest subscription cycles
const (
	SubscriptionCycle1Hour    SubscriptionCycle = "H1"
	SubscriptionCycle4Hours   SubscriptionCycle = "H4"
	SubscriptionCycle8Hours   SubscriptionCycle = "H8"
	SubscriptionCycle12Hours  SubscriptionCycle = "H12"
	SubscriptionCycleDaily    SubscriptionCycle = "DAILY"
	SubscriptionCycleWeekly   SubscriptionCycle = "WEEKLY"
	SubscriptionCycleBiWeekly SubscriptionCycle = "BI_WEEKLY"
	SubscriptionCycleMonthly  SubscriptionCycle = "MONTHLY"
)

// Weekday is the subscriptionStartWeekday parameter of a weekly or fortnightly auto-invest plan, and the weekday
// plans report
type Weekday string

// Weekdays, in UTC
const (
	WeekdayMonday    Weekday = "MON"
	WeekdayTuesday   Weekday = "TUE"
	WeekdayWednesday Weekday = "WED"
	WeekdayThursday  Weekday = "THU"
	WeekdayFriday    Weekday = "FRI"
	WeekdaySaturday  Weekday = "SAT"
	WeekdaySunday    Weekday = "SUN"
)

// HistoricalReturnOnInvestmentType is the hisRoiType parameter of the auto-invest target asset return on investment
// data, the period it covers
type HistoricalReturnOnInvestmentType string

// Historical return on investment periods
const (
	HistoricalReturnOnInvestmentTypeFiveYear   HistoricalReturnOnInvestmentType = "FIVE_YEAR"
	HistoricalReturnOnInvestmentTypeThreeYear  HistoricalReturnOnInvestmentType = "THREE_YEAR"
	HistoricalReturnOnInvestmentTypeOneYear    HistoricalReturnOnInvestmentType = "ONE_YEAR"
	HistoricalReturnOnInvestmentTypeSixMonth   HistoricalReturnOnInvestmentType = "SIX_MONTH"
	HistoricalReturnOnInvestmentTypeThreeMonth HistoricalReturnOnInvestmentType = "THREE_MONTH"
	HistoricalReturnOnInvestmentTypeSevenDay   HistoricalReturnOnInvestmentType = "SEVEN_DAY"
)

// Urgency is the urgency parameter of a volume participation algo order, the relative speed of execution
type Urgency string

// Volume participation urgencies
const (
	UrgencyLow    Urgency = "LOW"
	UrgencyMedium Urgency = "MEDIUM"
	UrgencyHigh   Urgency = "HIGH"
)

// The values below are for USDⓈ-M, COIN-M and portfolio margin futures endpoints

// WorkingType is the workingType parameter of a conditional futures order
type WorkingType string

// Working types, the price that triggers a USDⓈ-M, COIN-M or portfolio margin conditional order
const (
	WorkingTypeContractPrice WorkingType = "CONTRACT_PRICE" // Binance's default
	WorkingTypeMarkPrice     WorkingType = "MARK_PRICE"
)

// PositionSide is the positionSide parameter of a futures order
type PositionSide string

// Position sides of USDⓈ-M, COIN-M and portfolio margin futures orders: BOTH in one-way mode, LONG or SHORT in hedge
// mode
const (
	PositionSideBoth  PositionSide = "BOTH"
	PositionSideLong  PositionSide = "LONG"
	PositionSideShort PositionSide = "SHORT"
)

// PriceMatch is the priceMatch parameter of a futures order
type PriceMatch string

// Price match modes of USDⓈ-M, COIN-M and portfolio margin futures orders, which price an order at the best, 5th, 10th
// or 20th best price on the opposite side of the order book (OPPONENT) or on its own side (QUEUE) instead of at a given
// price. NONE is what responses report for an order without one; the order endpoints do not list it
const (
	PriceMatchNone       PriceMatch = "NONE"
	PriceMatchOpponent   PriceMatch = "OPPONENT"
	PriceMatchOpponent5  PriceMatch = "OPPONENT_5"
	PriceMatchOpponent10 PriceMatch = "OPPONENT_10"
	PriceMatchOpponent20 PriceMatch = "OPPONENT_20"
	PriceMatchQueue      PriceMatch = "QUEUE"
	PriceMatchQueue5     PriceMatch = "QUEUE_5"
	PriceMatchQueue10    PriceMatch = "QUEUE_10"
	PriceMatchQueue20    PriceMatch = "QUEUE_20"
)

// AlgoType is the algoType parameter of a futures algo order
type AlgoType string

// AlgoTypeConditional is the only algo type of USDⓈ-M, COIN-M and portfolio margin algo orders
const AlgoTypeConditional AlgoType = "CONDITIONAL"

// MarginType is the marginType parameter of a futures margin type change
type MarginType string

// Margin types of USDⓈ-M and COIN-M positions
const (
	MarginTypeIsolated MarginType = "ISOLATED"
	MarginTypeCrossed  MarginType = "CROSSED"
)

// ContractType is the contractType parameter of futures market data
type ContractType string

// Contract types of USDⓈ-M and COIN-M continuous klines and futures data. TRADIFI_PERPETUAL is USDⓈ-M only, and ALL
// is for COIN-M taker buy and sell volume only
const (
	ContractTypeAll                         ContractType = "ALL"
	ContractTypePerpetual                   ContractType = "PERPETUAL"
	ContractTypeCurrentQuarter              ContractType = "CURRENT_QUARTER"
	ContractTypeNextQuarter                 ContractType = "NEXT_QUARTER"
	ContractTypeTraditionalFinancePerpetual ContractType = "TRADIFI_PERPETUAL"
)

// AutoCloseType is the autoCloseType parameter of futures forced order queries
type AutoCloseType string

// Auto close types of USDⓈ-M, COIN-M and portfolio margin forced orders: LIQUIDATION for liquidations and ADL for
// auto-deleveraging
const (
	AutoCloseTypeLiquidation      AutoCloseType = "LIQUIDATION"
	AutoCloseTypeAutoDeleveraging AutoCloseType = "ADL"
)

// IncomeType is the incomeType parameter of futures income history queries
type IncomeType string

// Income types of the USDⓈ-M, COIN-M and portfolio margin income history. COIN-M accounts, portfolio margin's
// included, take the first seven; USDⓈ-M accounts take more
const (
	IncomeTypeTransfer                            IncomeType = "TRANSFER"
	IncomeTypeWelcomeBonus                        IncomeType = "WELCOME_BONUS"
	IncomeTypeRealisedPNL                         IncomeType = "REALIZED_PNL"
	IncomeTypeFundingFee                          IncomeType = "FUNDING_FEE"
	IncomeTypeCommission                          IncomeType = "COMMISSION"
	IncomeTypeInsuranceClear                      IncomeType = "INSURANCE_CLEAR"
	IncomeTypeDeliveredSettlement                 IncomeType = "DELIVERED_SETTELMENT" //nolint:misspell // Binance's spelling of the wire value
	IncomeTypeReferralKickback                    IncomeType = "REFERRAL_KICKBACK"
	IncomeTypeCommissionRebate                    IncomeType = "COMMISSION_REBATE"
	IncomeTypeAPIRebate                           IncomeType = "API_REBATE"
	IncomeTypeContestReward                       IncomeType = "CONTEST_REWARD"
	IncomeTypeCrossCollateralTransfer             IncomeType = "CROSS_COLLATERAL_TRANSFER"
	IncomeTypeOptionsPremiumFee                   IncomeType = "OPTIONS_PREMIUM_FEE"
	IncomeTypeOptionsSettleProfit                 IncomeType = "OPTIONS_SETTLE_PROFIT"
	IncomeTypeInternalTransfer                    IncomeType = "INTERNAL_TRANSFER"
	IncomeTypeAutoExchange                        IncomeType = "AUTO_EXCHANGE"
	IncomeTypeCoinSwapDeposit                     IncomeType = "COIN_SWAP_DEPOSIT"
	IncomeTypeCoinSwapWithdraw                    IncomeType = "COIN_SWAP_WITHDRAW" //nolint:gosec // An income type, not a credential
	IncomeTypePositionLimitIncreaseFee            IncomeType = "POSITION_LIMIT_INCREASE_FEE"
	IncomeTypeStrategyUSDTMarginedFuturesTransfer IncomeType = "STRATEGY_UMFUTURES_TRANSFER"
	IncomeTypeFeeReturn                           IncomeType = "FEE_RETURN"
	IncomeTypeBFUSDReward                         IncomeType = "BFUSD_REWARD"
	IncomeTypeSpecialFundingFee                   IncomeType = "SPECIAL_FUNDING_FEE"
)

// BNBTransferSide is the transferSide parameter of the portfolio margin and portfolio margin pro BNB transfers
type BNBTransferSide string

// BNB transfer sides: into (TO_UM) or out of (FROM_UM) the USDⓈ-M futures account
const (
	BNBTransferSideToUSDTMargined   BNBTransferSide = "TO_UM"
	BNBTransferSideFromUSDTMargined BNBTransferSide = "FROM_UM"
)

var (
	errAPIResponse                 = errors.New("API error response")
	errAlgoTypeRequired            = errors.New("algo type is required")
	errLeverageRequired            = errors.New("leverage is required")
	errPriceMatchWithPrice         = errors.New("priceMatch cannot be sent with price")
	errLoanTermMustBeSet           = errors.New("loan term must be set")
	errValidEmailRequired          = errors.New("valid email address is required")
	errPageNumberRequired          = errors.New("page number is required")
	errLimitNumberRequired         = errors.New("invalid limit")
	errEmptySubAccountAPIKey       = errors.New("invalid sub-account API key")
	errInvalidFuturesType          = errors.New("invalid futures types")
	errInvalidAccountType          = errors.New("invalid account type specified")
	errProductIDRequired           = errors.New("product ID is required")
	errProjectIDRequired           = errors.New("project ID is required")
	errPlanIDRequired              = errors.New("plan ID  is required")
	errIndexIDIsRequired           = errors.New("index ID is required")
	errTransferAlgorithmRequired   = errors.New("transfer algorithm is required")
	errUsernameRequired            = errors.New("user name is required")
	errTransferTypeRequired        = errors.New("transfer type is required")
	errNameRequired                = errors.New("name is required")
	errPositionIDRequired          = errors.New("position ID is required")
	errPositionDirectionRequired   = errors.New("position direction is required")
	errRedemptionAccountRequired   = errors.New("redemption account not specified")
	errOptionTypeRequired          = errors.New("optionType is required")
	errPageSizeRequired            = errors.New("page size is required")
	errPortfolioDetailRequired     = errors.New("portfolio detail is required")
	errPlanTypeRequired            = errors.New("planType is required")
	errTransactionIDRequired       = errors.New("transaction ID is required")
	errQuestionnaireRequired       = errors.New("questionnaire payload missing")
	errRequestIDRequired           = errors.New("request ID is required")
	errStartTimeRequired           = errors.New("start time is required")
	errStrategyTypeRequired        = errors.New("strategy type is required")
	errReferenceNumberRequired     = errors.New("reference number is required")
	errExpiredTypeRequired         = errors.New("expiredType is required")
	errQuoteIDRequired             = errors.New("quote ID is required")
	errAccountIDRequired           = errors.New("account ID is required")
	errAccountRequired             = errors.New("account information required")
	errConfigIDRequired            = errors.New("config ID is required")
	errLendingTypeRequired         = errors.New("lending type is required")
	errEmptyCurrencyCodes          = errors.New("assetNames are required")
	errSourceTypeRequired          = errors.New("source type required")
	errInvalidPercentageAmount     = errors.New("invalid percentage amount")
	errHashRateRequired            = errors.New("hash rate is required")
	errCodeRequired                = errors.New("code is required")
	errAddressRequired             = errors.New("address is required")
	errInvalidSubscriptionCycle    = errors.New("invalid subscription cycle")
	errDurationRequired            = errors.New("duration is required")
	errInvalidTransactionType      = errors.New("invalid transaction type")
	errPossibleValuesRequired      = errors.New("urgency field is required")
	errPlanStatusRequired          = errors.New("plan status is required")
	errUsageTypeRequired           = errors.New("usage type is required")
	errDownloadIDRequired          = errors.New("download id is required")
	errMarginChangeTypeInvalid     = errors.New("invalid margin changeType")
	errCancelReplaceModeRequired   = errors.New("cancelReplaceMode is required")
	errExpirationTimeRequired      = errors.New("expiration time is required")
	errSubAccountIDMissing         = errors.New("sub-account id is missing")
	errSubAccountStatusMissing     = errors.New("sub-account status missing")
	errCommissionValueRequired     = errors.New("commission value is required")
	errRewardTypeMissing           = errors.New("reward type is required")
	errInvalidBrokerID             = errors.New("missing brokerID")
	errBatchCancelRequiresSamePair = errors.New("batch cancellation requires a single asset type and pair")
	errInvalidListElement          = errors.New("invalid comma separated list element")
	errTooManyItems                = errors.New("list holds more items than Binance takes in one request")
)

// objectOrArray decodes a response that is a single object when one symbol is requested and an array otherwise
type objectOrArray[T any] []T

// UnmarshalJSON decodes a JSON object into a one-element slice and a JSON array into a slice
func (o *objectOrArray[T]) UnmarshalJSON(data []byte) error {
	if trimmed := bytes.TrimLeft(data, " \t\r\n"); len(trimmed) > 0 && trimmed[0] == '{' {
		var single T
		if err := json.Unmarshal(trimmed, &single); err != nil {
			return err
		}
		*o = objectOrArray[T]{single}
		return nil
	}
	return json.Unmarshal(data, (*[]T)(o))
}

// commaSeparatedElement is a type the elements of a CommaSeparated list decode into
type commaSeparatedElement interface {
	uint64 | currency.Code | string
}

// CommaSeparated holds a list Binance sends as one comma separated string, such as "BNB,BTC,ETH" or
// "12345678,12345679"
type CommaSeparated[T commaSeparatedElement] []T

// UnmarshalJSON decodes a comma separated string into its elements, and an empty string into an empty list
func (c *CommaSeparated[T]) UnmarshalJSON(data []byte) error {
	var list string
	if err := json.Unmarshal(data, &list); err != nil {
		return err
	}
	if list == "" {
		*c = nil
		return nil
	}
	items := strings.Split(list, ",")
	elements := make(CommaSeparated[T], len(items))
	for i, item := range items {
		switch element := any(&elements[i]).(type) {
		case *uint64:
			id, err := strconv.ParseUint(item, 10, 64)
			if err != nil {
				return fmt.Errorf("%w %q: %w", errInvalidListElement, item, err)
			}
			*element = id
		case *currency.Code:
			*element = currency.NewCode(item)
		case *string:
			*element = item
		}
	}
	*c = elements
	return nil
}

// ExchangeInfoRequest holds the optional filters of an exchange information request. Symbols cannot be combined with
// Permissions or SymbolStatus
type ExchangeInfoRequest struct {
	Symbols currency.Pairs
	// Permissions defaults to SPOT, MARGIN and LEVERAGED; symbols with any other permission are listed only when it is
	// requested
	Permissions []Permission
	// ShowPermissionSets false leaves each symbol's permission sets empty; the endpoint defaults it to true
	ShowPermissionSets *bool
	SymbolStatus       SymbolStatus
}

// ExchangeInfoResponse holds the exchange trading rules and symbol information
type ExchangeInfoResponse struct {
	Timezone        string                  `json:"timezone"`
	ServerTime      types.Time              `json:"serverTime"`
	RateLimits      []RateLimitItemResponse `json:"rateLimits"`
	ExchangeFilters []ExchangeFilter        `json:"exchangeFilters"`
	Symbols         []SymbolInfo            `json:"symbols"`
	// SORs is sent only when smart order routing (SOR) is available
	SORs []SORInfo `json:"sors"`
}

// RateLimitItemResponse holds a rate limit. Count, the usage counted against it, is sent by the unfilled order count
// query and the WebSocket API but not by the exchange information
type RateLimitItemResponse struct {
	RateLimitType  string `json:"rateLimitType"`
	Interval       string `json:"interval"`
	IntervalNumber uint64 `json:"intervalNum"`
	Limit          uint64 `json:"limit"`
	Count          uint64 `json:"count"`
}

// ExchangeFilter holds an exchange filter, whose type determines which limit it sets
type ExchangeFilter struct {
	FilterType             string `json:"filterType"`
	MaxNumberOrders        uint64 `json:"maxNumOrders"`
	MaxNumberAlgoOrders    uint64 `json:"maxNumAlgoOrders"`
	MaxNumberIcebergOrders uint64 `json:"maxNumIcebergOrders"`
	MaxNumberOrderLists    uint64 `json:"maxNumOrderLists"`
}

// SymbolInfo holds a symbol's trading rules and filters
type SymbolInfo struct {
	Symbol             string        `json:"symbol"`
	Status             string        `json:"status"` // TRADING, END_OF_DAY, HALT, BREAK or CANCEL_ONLY; Binance rejects END_OF_DAY as a SymbolStatus filter
	BaseAsset          currency.Code `json:"baseAsset"`
	BaseAssetPrecision uint64        `json:"baseAssetPrecision"`
	QuoteAsset         currency.Code `json:"quoteAsset"`
	// QuotePrecision is to be removed in a future API version in favour of QuoteAssetPrecision
	QuotePrecision                  uint64                    `json:"quotePrecision"`
	QuoteAssetPrecision             uint64                    `json:"quoteAssetPrecision"`
	BaseCommissionPrecision         uint64                    `json:"baseCommissionPrecision"`
	QuoteCommissionPrecision        uint64                    `json:"quoteCommissionPrecision"`
	OrderTypes                      []OrderType               `json:"orderTypes"`
	IcebergAllowed                  bool                      `json:"icebergAllowed"`
	OneCancelsTheOtherAllowed       bool                      `json:"ocoAllowed"`
	OneTriggersTheOtherAllowed      bool                      `json:"otoAllowed"`
	OPOAllowed                      bool                      `json:"opoAllowed"`
	QuoteOrderQuantityMarketAllowed bool                      `json:"quoteOrderQtyMarketAllowed"`
	AllowTrailingStop               bool                      `json:"allowTrailingStop"`
	CancelReplaceAllowed            bool                      `json:"cancelReplaceAllowed"`
	AmendAllowed                    bool                      `json:"amendAllowed"`
	PegInstructionsAllowed          bool                      `json:"pegInstructionsAllowed"`
	IsSpotTradingAllowed            bool                      `json:"isSpotTradingAllowed"`
	IsMarginTradingAllowed          bool                      `json:"isMarginTradingAllowed"`
	Filters                         []SymbolFilter            `json:"filters"`
	Permissions                     []Permission              `json:"permissions"`
	PermissionSets                  [][]Permission            `json:"permissionSets"`
	DefaultSelfTradePreventionMode  SelfTradePreventionMode   `json:"defaultSelfTradePreventionMode"`
	AllowedSelfTradePreventionModes []SelfTradePreventionMode `json:"allowedSelfTradePreventionModes"`
}

// SymbolFilter holds a symbol filter, whose type determines which of the fields it sets
type SymbolFilter struct {
	FilterType string `json:"filterType"`
	// PRICE_FILTER
	MinPrice types.Number `json:"minPrice"`
	MaxPrice types.Number `json:"maxPrice"`
	TickSize types.Number `json:"tickSize"`
	// PERCENT_PRICE
	MultiplierUp   types.Number `json:"multiplierUp"`
	MultiplierDown types.Number `json:"multiplierDown"`
	// AveragePriceMinutes is the window of the average price the percent price and notional filters use; zero means the
	// last price
	AveragePriceMinutes uint64 `json:"avgPriceMins"`
	// PERCENT_PRICE_BY_SIDE
	BidMultiplierUp   types.Number `json:"bidMultiplierUp"`
	BidMultiplierDown types.Number `json:"bidMultiplierDown"`
	AskMultiplierUp   types.Number `json:"askMultiplierUp"`
	AskMultiplierDown types.Number `json:"askMultiplierDown"`
	// LOT_SIZE and MARKET_LOT_SIZE
	MinQuantity types.Number `json:"minQty"`
	MaxQuantity types.Number `json:"maxQty"`
	StepSize    types.Number `json:"stepSize"`
	// MIN_NOTIONAL and NOTIONAL
	MinNotional      types.Number `json:"minNotional"`
	ApplyToMarket    bool         `json:"applyToMarket"`
	ApplyMinToMarket bool         `json:"applyMinToMarket"`
	MaxNotional      types.Number `json:"maxNotional"`
	ApplyMaxToMarket bool         `json:"applyMaxToMarket"`
	// ICEBERG_PARTS
	Limit uint64 `json:"limit"`
	// MAX_NUM_ORDERS, MAX_NUM_ALGO_ORDERS, MAX_NUM_ICEBERG_ORDERS, MAX_NUM_ORDER_LISTS and MAX_NUM_ORDER_AMENDS
	MaxNumberOrders        uint64 `json:"maxNumOrders"`
	MaxNumberAlgoOrders    uint64 `json:"maxNumAlgoOrders"`
	MaxNumberIcebergOrders uint64 `json:"maxNumIcebergOrders"`
	MaxNumberOrderLists    uint64 `json:"maxNumOrderLists"`
	MaxNumberOrderAmends   uint64 `json:"maxNumOrderAmends"`
	// MAX_POSITION
	MaxPosition types.Number `json:"maxPosition"`
	// TRAILING_DELTA, in basis points
	MinTrailingAboveDelta uint64 `json:"minTrailingAboveDelta"`
	MaxTrailingAboveDelta uint64 `json:"maxTrailingAboveDelta"`
	MinTrailingBelowDelta uint64 `json:"minTrailingBelowDelta"`
	MaxTrailingBelowDelta uint64 `json:"maxTrailingBelowDelta"`
	// T_PLUS_SELL
	EndTime types.Time `json:"endTime"`
}

// SORInfo holds the symbols smart order routing (SOR) routes a base asset's orders between
type SORInfo struct {
	BaseAsset currency.Code `json:"baseAsset"`
	Symbols   []string      `json:"symbols"`
}

// ServerTimeResponse holds the server time
type ServerTimeResponse struct {
	ServerTime types.Time `json:"serverTime"`
}

// AggregatedTradeRequest holds the parameters of an aggregate trades request. FromID cannot be combined with
// StartTime or EndTime, and a Limit above 1000 is collected over several requests, which needs FromID or StartTime
type AggregatedTradeRequest struct {
	Symbol    currency.Pair
	FromID    uint64
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
}

// AggregatedTradeResponse holds an aggregate trade: the fills of one taker order at one price
type AggregatedTradeResponse struct {
	AggregateTradeID uint64       `json:"a"`
	Price            types.Number `json:"p"`
	Quantity         types.Number `json:"q"`
	FirstTradeID     uint64       `json:"f"`
	LastTradeID      uint64       `json:"l"`
	Timestamp        types.Time   `json:"T"`
	IsBuyerMaker     bool         `json:"m"`
	IsBestMatch      bool         `json:"M"`
}

// AveragePriceResponse holds a symbol's current average price
type AveragePriceResponse struct {
	IntervalInMinutes uint64       `json:"mins"`
	Price             types.Number `json:"price"`
	CloseTime         types.Time   `json:"closeTime"` // Time of the last trade
}

// OrderBookRequest holds the parameters of an order book request
type OrderBookRequest struct {
	Symbol currency.Pair
	// Limit is the number of levels on each side, 100 when zero and at most 5000
	Limit uint64
	// SymbolStatus rejects the request when the symbol's trading status differs
	SymbolStatus SymbolStatus
}

// OrderBookResponse holds an order book
type OrderBookResponse struct {
	LastUpdateID int64                            `json:"lastUpdateId"` // int64 to match the order book API it feeds
	Bids         orderbook.LevelsArrayPriceAmount `json:"bids"`
	Asks         orderbook.LevelsArrayPriceAmount `json:"asks"`
}

// RecentTradeRequest holds the parameters of a recent trades request
type RecentTradeRequest struct {
	Symbol currency.Pair
	Limit  uint64 // 500 when zero, at most 1000
}

// RecentTradeResponse holds a trade of the recent or historical trades lists
type RecentTradeResponse struct {
	ID            uint64       `json:"id"`
	Price         types.Number `json:"price"`
	Quantity      types.Number `json:"qty"`
	QuoteQuantity types.Number `json:"quoteQty"`
	Time          types.Time   `json:"time"`
	IsBuyerMaker  bool         `json:"isBuyerMaker"`
	IsBestMatch   bool         `json:"isBestMatch"`
}

// KlinesRequest holds the parameters of a klines or UIKlines request
type KlinesRequest struct {
	Symbol   currency.Pair
	Interval kline.Interval
	Limit    uint64 // 500 when zero, at most 1000
	// StartTime and EndTime are always in UTC
	StartTime time.Time
	EndTime   time.Time
	// TimeZone sets the time zone the intervals are interpreted in, as hours and minutes or hours alone from -12:00 to
	// +14:00; UTC when empty
	TimeZone string
}

// CandleStickResponse holds a kline, a candlestick bar
type CandleStickResponse struct {
	OpenTime                 types.Time
	OpenPrice                types.Number
	HighPrice                types.Number
	LowPrice                 types.Number
	ClosePrice               types.Number
	Volume                   types.Number
	CloseTime                types.Time
	QuoteAssetVolume         types.Number
	NumberOfTrades           uint64
	TakerBuyBaseAssetVolume  types.Number
	TakerBuyQuoteAssetVolume types.Number
}

// UnmarshalJSON decodes a kline from its array form, whose twelfth element is unused
func (c *CandleStickResponse) UnmarshalJSON(data []byte) error {
	target := [12]any{&c.OpenTime, &c.OpenPrice, &c.HighPrice, &c.LowPrice, &c.ClosePrice, &c.Volume, &c.CloseTime, &c.QuoteAssetVolume, &c.NumberOfTrades, &c.TakerBuyBaseAssetVolume, &c.TakerBuyQuoteAssetVolume, nil}
	return json.Unmarshal(data, &target)
}

// RollingWindowTickerRequest holds the parameters of a rolling window price change statistics request
type RollingWindowTickerRequest struct {
	Symbols currency.Pairs // At most 100
	// WindowSize is sent as whole minutes from 1 to 59, hours from 1 to 23 or days from 1 to 7; one day when zero
	WindowSize   time.Duration
	Type         TickerType // FULL when empty, or MINI
	SymbolStatus SymbolStatus
}

// TickerStatisticsResponse holds a symbol's price change statistics over a trading day or a rolling window. The MINI
// type leaves out the price change fields
type TickerStatisticsResponse struct {
	Symbol               string       `json:"symbol"`
	PriceChange          types.Number `json:"priceChange"`
	PriceChangePercent   types.Number `json:"priceChangePercent"`
	WeightedAveragePrice types.Number `json:"weightedAvgPrice"`
	OpenPrice            types.Number `json:"openPrice"`
	HighPrice            types.Number `json:"highPrice"`
	LowPrice             types.Number `json:"lowPrice"`
	LastPrice            types.Number `json:"lastPrice"`
	Volume               types.Number `json:"volume"`
	QuoteVolume          types.Number `json:"quoteVolume"`
	OpenTime             types.Time   `json:"openTime"`
	CloseTime            types.Time   `json:"closeTime"`
	FirstID              int64        `json:"firstId"` // -1 when there were no trades
	LastID               int64        `json:"lastId"`  // -1 when there were no trades
	Count                uint64       `json:"count"`
}

// PriceChangeStatsRequest holds the parameters of a 24hr ticker price change statistics request
type PriceChangeStatsRequest struct {
	Symbols      currency.Pairs // Every symbol when empty
	Type         TickerType     // FULL when empty, or MINI
	SymbolStatus SymbolStatus
}

// PriceChangeStatsResponse holds 24hr ticker price change statistics; the MINI type leaves out the price change,
// previous close, last quantity, bid and ask fields
type PriceChangeStatsResponse struct {
	Symbol               string       `json:"symbol"`
	PriceChange          types.Number `json:"priceChange"`
	PriceChangePercent   types.Number `json:"priceChangePercent"`
	WeightedAveragePrice types.Number `json:"weightedAvgPrice"`
	PreviousClosePrice   types.Number `json:"prevClosePrice"`
	LastPrice            types.Number `json:"lastPrice"`
	LastQuantity         types.Number `json:"lastQty"`
	BidPrice             types.Number `json:"bidPrice"`
	BidQuantity          types.Number `json:"bidQty"`
	AskPrice             types.Number `json:"askPrice"`
	AskQuantity          types.Number `json:"askQty"`
	OpenPrice            types.Number `json:"openPrice"`
	HighPrice            types.Number `json:"highPrice"`
	LowPrice             types.Number `json:"lowPrice"`
	Volume               types.Number `json:"volume"`
	QuoteVolume          types.Number `json:"quoteVolume"`
	OpenTime             types.Time   `json:"openTime"`
	CloseTime            types.Time   `json:"closeTime"`
	FirstID              int64        `json:"firstId"` // -1 when there were no trades
	LastID               int64        `json:"lastId"`  // -1 when there were no trades
	Count                uint64       `json:"count"`
}

// BookTickerResponse holds a symbol's best bid and ask price and quantity
type BookTickerResponse struct {
	Symbol      string       `json:"symbol"`
	BidPrice    types.Number `json:"bidPrice"`
	BidQuantity types.Number `json:"bidQty"`
	AskPrice    types.Number `json:"askPrice"`
	AskQuantity types.Number `json:"askQty"`
}

// SymbolPriceResponse holds a symbol's latest price
type SymbolPriceResponse struct {
	Symbol string       `json:"symbol"`
	Price  types.Number `json:"price"`
}

// TradingDayTickerRequest holds the parameters of a trading day ticker request
type TradingDayTickerRequest struct {
	Symbols currency.Pairs // At most 100
	// TimeZone sets the time zone the trading day starts in, as hours and minutes or hours alone; UTC when empty
	TimeZone     string
	Type         TickerType // FULL when empty, or MINI
	SymbolStatus SymbolStatus
}

// OrderConditionalFields holds the order response fields that appear only when an order uses the feature behind them,
// such as an iceberg quantity, self-trade prevention, a stop price, a strategy, a trailing stop, smart order routing
// (SOR), a price peg or an expiry
type OrderConditionalFields struct {
	IcebergQuantity            types.Number `json:"icebergQty"`
	PreventedMatchID           uint64       `json:"preventedMatchId"`
	PreventedQuantity          types.Number `json:"preventedQuantity"`
	StopPrice                  types.Number `json:"stopPrice"`
	StrategyID                 uint64       `json:"strategyId"`
	StrategyType               uint64       `json:"strategyType"`
	TrailingDeltaInBasisPoints uint64       `json:"trailingDelta"`
	// TrailingTime is the Unix time in milliseconds the trailing stop started tracking the price, -1 until then
	TrailingTime   int64         `json:"trailingTime"`
	UsedSOR        bool          `json:"usedSor"`
	WorkingFloor   string        `json:"workingFloor"`
	PegPriceType   PegPriceType  `json:"pegPriceType"`
	PegOffsetType  PegOffsetType `json:"pegOffsetType"`
	PegOffsetValue uint64        `json:"pegOffsetValue"`
	PeggedPrice    types.Number  `json:"peggedPrice"`
	ExpiryReason   string        `json:"expiryReason"`
}

// CancelOrderResponse holds a cancelled order
type CancelOrderResponse struct {
	Symbol                     string                  `json:"symbol"`
	OriginalClientOrderID      string                  `json:"origClientOrderId"`
	OrderID                    uint64                  `json:"orderId"`
	OrderListID                int64                   `json:"orderListId"` // -1 unless the order is part of an order list
	ClientOrderID              string                  `json:"clientOrderId"`
	TransactTime               types.Time              `json:"transactTime"`
	Price                      types.Number            `json:"price"`
	OriginalQuantity           types.Number            `json:"origQty"`
	ExecutedQuantity           types.Number            `json:"executedQty"`
	OriginalQuoteOrderQuantity types.Number            `json:"origQuoteOrderQty"`
	CumulativeQuoteQuantity    types.Number            `json:"cummulativeQuoteQty"`
	Status                     string                  `json:"status"`
	TimeInForce                TimeInForce             `json:"timeInForce"`
	Type                       OrderType               `json:"type"`
	Side                       Side                    `json:"side"`
	SelfTradePreventionMode    SelfTradePreventionMode `json:"selfTradePreventionMode"`
	OrderConditionalFields
}

// CancelledOpenOrderResponse holds an entry of a cancel all open orders response: a cancelled order, or a cancelled
// order list with its list status, orders and order reports
type CancelledOpenOrderResponse struct {
	CancelOrderResponse
	ContingencyType   string           `json:"contingencyType"`
	ListStatusType    string           `json:"listStatusType"`
	ListOrderStatus   string           `json:"listOrderStatus"`
	ListClientOrderID string           `json:"listClientOrderId"`
	TransactionTime   types.Time       `json:"transactionTime"`
	Orders            []OrderListOrder `json:"orders"`
	OrderReports      []OrderResponse  `json:"orderReports"`
}

// NewOrderRequest holds the parameters of a new order. Which quantity, price, stop and trailing parameters are
// mandatory depends on Type
type NewOrderRequest struct {
	Symbol                     currency.Pair
	Side                       Side // BUY or SELL
	Type                       OrderType
	TimeInForce                TimeInForce
	Quantity                   float64
	QuoteOrderQuantity         float64 // Quote asset amount a MARKET order spends or receives, instead of Quantity
	Price                      float64
	NewClientOrderID           string
	StrategyID                 uint64
	StrategyType               uint64 // At least 1000000; smaller values are reserved
	StopPrice                  float64
	TrailingDeltaInBasisPoints uint64
	IcebergQuantity            float64
	// NewOrderResponseType selects the ACK, RESULT or FULL response; MARKET and LIMIT orders default to FULL and the
	// others to ACK
	NewOrderResponseType    NewOrderResponseType
	SelfTradePreventionMode SelfTradePreventionMode
	PegPriceType            PegPriceType // PRIMARY_PEG or MARKET_PEG, which makes Price optional
	PegOffsetValue          uint64       // Price level to peg to, at most 100
	PegOffsetType           PegOffsetType
}

// NewOrderResponse holds a new order in the ACK, RESULT or FULL format: ACK sets only the identifiers and transaction
// time, and FULL adds the fills to RESULT
type NewOrderResponse struct {
	Symbol                     string       `json:"symbol"`
	OrderID                    uint64       `json:"orderId"`
	OrderListID                int64        `json:"orderListId"` // -1 unless the order is part of an order list
	ClientOrderID              string       `json:"clientOrderId"`
	TransactTime               types.Time   `json:"transactTime"`
	Price                      types.Number `json:"price"`
	OriginalQuantity           types.Number `json:"origQty"`
	ExecutedQuantity           types.Number `json:"executedQty"`
	OriginalQuoteOrderQuantity types.Number `json:"origQuoteOrderQty"`
	CumulativeQuoteQuantity    types.Number `json:"cummulativeQuoteQty"`
	Status                     string       `json:"status"`
	TimeInForce                TimeInForce  `json:"timeInForce"`
	Type                       OrderType    `json:"type"`
	Side                       Side         `json:"side"`
	// WorkingTime is the Unix time in milliseconds the order started working on the order book, -1 until then
	WorkingTime             int64                   `json:"workingTime"`
	SelfTradePreventionMode SelfTradePreventionMode `json:"selfTradePreventionMode"`
	Fills                   []OrderFill             `json:"fills"`
	OrderConditionalFields
}

// OrderFill holds a fill of a new order
type OrderFill struct {
	Price           types.Number  `json:"price"`
	Quantity        types.Number  `json:"qty"`
	Commission      types.Number  `json:"commission"`
	CommissionAsset currency.Code `json:"commissionAsset"`
	TradeID         uint64        `json:"tradeId"`
}

// CancelOrderRequest holds the parameters of an order cancellation; OrderID or OriginalClientOrderID is required
type CancelOrderRequest struct {
	Symbol                currency.Pair
	OrderID               uint64
	OriginalClientOrderID string
	NewClientOrderID      string             // Identifies the cancellation; generated when empty
	CancelRestrictions    CancelRestrictions // ONLY_NEW or ONLY_PARTIALLY_FILLED cancels only orders with that status
}

// CancelOrderListRequest holds the parameters of an order list cancellation; OrderListID or ListClientOrderID is
// required
type CancelOrderListRequest struct {
	Symbol            currency.Pair
	OrderListID       uint64
	ListClientOrderID string
	NewClientOrderID  string // Identifies the cancellation; generated when empty
}

// OneCancelsTheOtherOrderResponse holds an order list; placements and cancellations add the order reports
type OneCancelsTheOtherOrderResponse struct {
	OrderListID       uint64           `json:"orderListId"`
	ContingencyType   string           `json:"contingencyType"`
	ListStatusType    string           `json:"listStatusType"`
	ListOrderStatus   string           `json:"listOrderStatus"`
	ListClientOrderID string           `json:"listClientOrderId"`
	TransactionTime   types.Time       `json:"transactionTime"`
	Symbol            string           `json:"symbol"`
	Orders            []OrderListOrder `json:"orders"`
	OrderReports      []OrderResponse  `json:"orderReports"`
}

// OrderListOrder identifies an order of an order list
type OrderListOrder struct {
	Symbol        string `json:"symbol"`
	OrderID       uint64 `json:"orderId"`
	ClientOrderID string `json:"clientOrderId"`
}

// OrderResponse holds an order report of an order list placement, in the placement's ACK, RESULT or FULL format, or of
// an order list cancellation, which alone sends OriginalClientOrderID
type OrderResponse struct {
	Symbol                     string            `json:"symbol"`
	OriginalClientOrderID      string            `json:"origClientOrderId"`
	OrderID                    uint64            `json:"orderId"`
	OrderListID                int64             `json:"orderListId"` // -1 unless the order is part of an order list
	ClientOrderID              string            `json:"clientOrderId"`
	TransactTime               types.Time        `json:"transactTime"`
	Price                      types.Number      `json:"price"`
	OriginalQuantity           types.Number      `json:"origQty"`
	ExecutedQuantity           types.Number      `json:"executedQty"`
	OriginalQuoteOrderQuantity types.Number      `json:"origQuoteOrderQty"`
	CumulativeQuoteQuantity    types.Number      `json:"cummulativeQuoteQty"`
	Status                     string            `json:"status"`
	TimeInForce                order.TimeInForce `json:"timeInForce"`
	Type                       OrderType         `json:"type"`
	Side                       Side              `json:"side"`
	// WorkingTime is the Unix time in milliseconds the order started working on the order book, -1 until then
	WorkingTime             int64                   `json:"workingTime"`
	SelfTradePreventionMode SelfTradePreventionMode `json:"selfTradePreventionMode"`
	Fills                   []OrderFill             `json:"fills"`
	OrderConditionalFields
}

// CancelReplaceOrderRequest holds the parameters of a cancel-replace: the order to cancel, by CancelOrderID or
// CancelOriginalClientOrderID, and the new order, whose mandatory parameters depend on Type
type CancelReplaceOrderRequest struct {
	Symbol currency.Pair
	Side   Side // BUY or SELL
	Type   OrderType
	// CancelReplaceMode STOP_ON_FAILURE places the new order only if the cancellation succeeds, and ALLOW_FAILURE places
	// it either way
	CancelReplaceMode           CancelReplaceMode
	TimeInForce                 TimeInForce
	Quantity                    float64
	QuoteOrderQuantity          float64
	Price                       float64
	CancelNewClientOrderID      string // Identifies the cancellation; generated when empty
	CancelOriginalClientOrderID string
	CancelOrderID               uint64
	NewClientOrderID            string
	StrategyID                  uint64
	StrategyType                uint64 // At least 1000000; smaller values are reserved
	StopPrice                   float64
	TrailingDeltaInBasisPoints  uint64
	IcebergQuantity             float64
	NewOrderResponseType        NewOrderResponseType // ACK, RESULT or FULL
	SelfTradePreventionMode     SelfTradePreventionMode
	CancelRestrictions          CancelRestrictions // ONLY_NEW or ONLY_PARTIALLY_FILLED cancels only orders with that status
	// OrderRateLimitExceededMode DO_NOTHING, the default, cancels only within the unfilled order count limit, and
	// CANCEL_ONLY always cancels
	OrderRateLimitExceededMode OrderRateLimitExceededMode
	PegPriceType               PegPriceType
	PegOffsetValue             uint64
	PegOffsetType              PegOffsetType
}

// CancelAndReplaceResponse holds the outcome of a cancel-replace. Binance replies with it when both halves succeed and
// puts it in the error's data when either fails, so a failed half holds an error code and message instead of an order
type CancelAndReplaceResponse struct {
	CancelResult     string                        `json:"cancelResult"`   // SUCCESS or FAILURE
	NewOrderResult   string                        `json:"newOrderResult"` // SUCCESS, FAILURE or NOT_ATTEMPTED
	CancelResponse   CancelReplaceCancelResponse   `json:"cancelResponse"`
	NewOrderResponse CancelReplaceNewOrderResponse `json:"newOrderResponse"`
}

// CancelReplaceCancelResponse holds the cancellation half of a cancel-replace: the cancelled order, or the error code
// and message of a failed cancellation
type CancelReplaceCancelResponse struct {
	CancelOrderResponse
	Code    int64  `json:"code"` // negative when the cancellation failed
	Message string `json:"msg"`
}

// CancelReplaceNewOrderResponse holds the new order half of a cancel-replace: the placed order, or the error code and
// message of a failed placement
type CancelReplaceNewOrderResponse struct {
	NewOrderResponse
	Code    int64  `json:"code"` // negative when the placement failed
	Message string `json:"msg"`
}

// OneCancelsTheOtherOrderListRequest holds the parameters of a one-cancels-the-other (OCO) order list
type OneCancelsTheOtherOrderListRequest struct {
	Symbol                  currency.Pair
	ListClientOrderID       string
	Side                    Side    // BUY or SELL
	Quantity                float64 // Quantity of both orders
	Above                   OneCancelsTheOtherOrderListLeg
	Below                   OneCancelsTheOtherOrderListLeg
	NewOrderResponseType    NewOrderResponseType // ACK, RESULT or FULL
	SelfTradePreventionMode SelfTradePreventionMode
}

// OneCancelsTheOtherOrderListLeg holds the parameters of the above or below order of an OCO order list
type OneCancelsTheOtherOrderListLeg struct {
	Type                       OrderType
	ClientOrderID              string
	IcebergQuantity            uint64  // A whole number, usable only with the GTC time in force
	Price                      float64 // Limit price of a LIMIT_MAKER, STOP_LOSS_LIMIT or TAKE_PROFIT_LIMIT order
	StopPrice                  float64
	TrailingDeltaInBasisPoints uint64
	TimeInForce                TimeInForce // Required by STOP_LOSS_LIMIT and TAKE_PROFIT_LIMIT orders
	StrategyID                 uint64
	StrategyType               uint64 // At least 1000000; smaller values are reserved
	PegPriceType               PegPriceType
	PegOffsetType              PegOffsetType
	PegOffsetValue             uint64
}

// OrderTestResponse holds the commission rates a test order would pay, which are sent only when they are requested
type OrderTestResponse struct {
	StandardCommissionForOrder OrderCommissionRates `json:"standardCommissionForOrder"`
	SpecialCommissionForOrder  OrderCommissionRates `json:"specialCommissionForOrder"`
	TaxCommissionForOrder      OrderCommissionRates `json:"taxCommissionForOrder"`
	Discount                   CommissionDiscount   `json:"discount"`
}

// OrderCommissionRates holds the maker and taker commission rates of an order
type OrderCommissionRates struct {
	Maker types.Number `json:"maker"`
	Taker types.Number `json:"taker"`
}

// CommissionDiscount holds the discount on standard commissions paid in the discount asset
type CommissionDiscount struct {
	EnabledForAccount bool          `json:"enabledForAccount"`
	EnabledForSymbol  bool          `json:"enabledForSymbol"`
	DiscountAsset     currency.Code `json:"discountAsset"`
	Discount          types.Number  `json:"discount"` // Rate the standard commission is reduced by
}

// SOROrderRequest holds the parameters of an order using smart order routing (SOR), which supports only LIMIT and
// MARKET orders without a quote order quantity
type SOROrderRequest struct {
	Symbol                  currency.Pair
	Side                    Side      // BUY or SELL
	Type                    OrderType // LIMIT or MARKET
	TimeInForce             TimeInForce
	Quantity                float64
	Price                   float64
	NewClientOrderID        string
	StrategyID              uint64
	StrategyType            uint64 // At least 1000000; smaller values are reserved
	IcebergQuantity         float64
	NewOrderResponseType    NewOrderResponseType // ACK, RESULT or FULL; FULL when empty
	SelfTradePreventionMode SelfTradePreventionMode
}

// SOROrderResponse holds an order placed using smart order routing (SOR)
type SOROrderResponse struct {
	Symbol                     string       `json:"symbol"`
	OrderID                    uint64       `json:"orderId"`
	OrderListID                int64        `json:"orderListId"` // always -1
	ClientOrderID              string       `json:"clientOrderId"`
	TransactTime               types.Time   `json:"transactTime"`
	Price                      types.Number `json:"price"`
	OriginalQuantity           types.Number `json:"origQty"`
	ExecutedQuantity           types.Number `json:"executedQty"`
	OriginalQuoteOrderQuantity types.Number `json:"origQuoteOrderQty"`
	CumulativeQuoteQuantity    types.Number `json:"cummulativeQuoteQty"`
	Status                     string       `json:"status"`
	TimeInForce                TimeInForce  `json:"timeInForce"`
	Type                       OrderType    `json:"type"`
	Side                       Side         `json:"side"`
	// WorkingTime is the Unix time in milliseconds the order started working on the order book, -1 until then
	WorkingTime             int64                   `json:"workingTime"`
	Fills                   []SORFill               `json:"fills"`
	SelfTradePreventionMode SelfTradePreventionMode `json:"selfTradePreventionMode"`
	OrderConditionalFields
}

// SORFill holds a fill of an order placed using smart order routing (SOR)
type SORFill struct {
	MatchType       string        `json:"matchType"`
	Price           types.Number  `json:"price"`
	Quantity        types.Number  `json:"qty"`
	Commission      types.Number  `json:"commission"`
	CommissionAsset currency.Code `json:"commissionAsset"`
	TradeID         int64         `json:"tradeId"` // -1 for a fill matched through a trade report
	AllocationID    uint64        `json:"allocId"`
}

// AccountCommissionResponse holds the account's commission rates on a symbol
type AccountCommissionResponse struct {
	Symbol             string             `json:"symbol"`
	StandardCommission CommissionRates    `json:"standardCommission"`
	SpecialCommission  CommissionRates    `json:"specialCommission"`
	TaxCommission      CommissionRates    `json:"taxCommission"`
	Discount           CommissionDiscount `json:"discount"`
}

// CommissionRates holds maker, taker, buyer and seller commission rates
type CommissionRates struct {
	Maker  types.Number `json:"maker"`
	Taker  types.Number `json:"taker"`
	Buyer  types.Number `json:"buyer"`
	Seller types.Number `json:"seller"`
}

// AllOrderListsRequest holds the optional filters of an all order lists request. FromID cannot be combined with
// StartTime or EndTime, which can be at most 24 hours apart
type AllOrderListsRequest struct {
	FromID    uint64
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64 // 500 when zero, at most 1000
}

// AllOrdersRequest holds the parameters of an all orders request. OrderID returns orders from that ID on, and
// StartTime and EndTime can be at most 24 hours apart
type AllOrdersRequest struct {
	Symbol    currency.Pair
	OrderID   uint64
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64 // 500 when zero, at most 1000
}

// TradeOrderResponse holds an order as queried
type TradeOrderResponse struct {
	Symbol                     string            `json:"symbol"`
	OrderID                    uint64            `json:"orderId"`
	OrderListID                int64             `json:"orderListId"` // -1 unless the order is part of an order list
	ClientOrderID              string            `json:"clientOrderId"`
	Price                      types.Number      `json:"price"`
	OriginalQuantity           types.Number      `json:"origQty"`
	ExecutedQuantity           types.Number      `json:"executedQty"`
	OriginalQuoteOrderQuantity types.Number      `json:"origQuoteOrderQty"`
	CumulativeQuoteQuantity    types.Number      `json:"cummulativeQuoteQty"` // Negative when unavailable for some historical orders
	Status                     string            `json:"status"`
	TimeInForce                order.TimeInForce `json:"timeInForce"`
	Type                       OrderType         `json:"type"`
	Side                       Side              `json:"side"`
	Time                       types.Time        `json:"time"`
	UpdateTime                 types.Time        `json:"updateTime"`
	IsWorking                  bool              `json:"isWorking"`
	// WorkingTime is the Unix time in milliseconds the order started working on the order book, -1 until then
	WorkingTime             int64                   `json:"workingTime"`
	SelfTradePreventionMode SelfTradePreventionMode `json:"selfTradePreventionMode"`
	OrderConditionalFields
}

// AccountResponse holds the account information. The commissions are signed, since a liquidity
// provider's maker rebate is reported as a negative commission
type AccountResponse struct {
	MakerCommissionInBasisPoints  int64           `json:"makerCommission"`
	TakerCommissionInBasisPoints  int64           `json:"takerCommission"`
	BuyerCommissionInBasisPoints  int64           `json:"buyerCommission"`
	SellerCommissionInBasisPoints int64           `json:"sellerCommission"`
	CommissionRates               CommissionRates `json:"commissionRates"`
	CanTrade                      bool            `json:"canTrade"`
	CanWithdraw                   bool            `json:"canWithdraw"`
	CanDeposit                    bool            `json:"canDeposit"`
	Brokered                      bool            `json:"brokered"`
	RequireSelfTradePrevention    bool            `json:"requireSelfTradePrevention"`
	PreventSOR                    bool            `json:"preventSor"`
	UpdateTime                    types.Time      `json:"updateTime"`
	AccountType                   string          `json:"accountType"`
	Balances                      []Balance       `json:"balances"`
	Permissions                   []Permission    `json:"permissions"`
	UID                           uint64          `json:"uid"`
}

// Balance holds an asset's free and locked balance
type Balance struct {
	Asset  currency.Code   `json:"asset"`
	Free   decimal.Decimal `json:"free"`
	Locked decimal.Decimal `json:"locked"`
}

// AllocationsRequest holds the parameters of an allocations request; StartTime and EndTime can be at most 24 hours
// apart
type AllocationsRequest struct {
	Symbol           currency.Pair
	StartTime        time.Time
	EndTime          time.Time
	FromAllocationID uint64
	Limit            uint64 // 500 when zero, at most 1000
	OrderID          uint64
}

// AllocationResponse holds an allocation resulting from an order placed using smart order routing (SOR)
type AllocationResponse struct {
	Symbol          string        `json:"symbol"`
	AllocationID    uint64        `json:"allocationId"`
	AllocationType  string        `json:"allocationType"`
	OrderID         uint64        `json:"orderId"`
	OrderListID     int64         `json:"orderListId"` // -1 unless the order is part of an order list
	Price           types.Number  `json:"price"`
	Quantity        types.Number  `json:"qty"`
	QuoteQuantity   types.Number  `json:"quoteQty"`
	Commission      types.Number  `json:"commission"`
	CommissionAsset currency.Code `json:"commissionAsset"`
	Time            types.Time    `json:"time"`
	IsBuyer         bool          `json:"isBuyer"`
	IsMaker         bool          `json:"isMaker"`
	IsAllocator     bool          `json:"isAllocator"`
}

// PreventedMatchesRequest holds the parameters of a prevented matches request, by PreventedMatchID or by OrderID,
// which FromPreventedMatchID and Limit page through
type PreventedMatchesRequest struct {
	Symbol               currency.Pair
	PreventedMatchID     uint64
	OrderID              uint64
	FromPreventedMatchID uint64
	Limit                uint64 // 500 when zero, at most 1000
}

// PreventedMatchResponse holds a match that self-trade prevention (STP) prevented, expiring the orders involved
type PreventedMatchResponse struct {
	Symbol                  string                  `json:"symbol"`
	PreventedMatchID        uint64                  `json:"preventedMatchId"`
	TakerOrderID            uint64                  `json:"takerOrderId"`
	MakerSymbol             string                  `json:"makerSymbol"`
	MakerOrderID            uint64                  `json:"makerOrderId"`
	TradeGroupID            uint64                  `json:"tradeGroupId"`
	SelfTradePreventionMode SelfTradePreventionMode `json:"selfTradePreventionMode"`
	Price                   types.Number            `json:"price"`
	MakerPreventedQuantity  types.Number            `json:"makerPreventedQuantity"`
	TransactTime            types.Time              `json:"transactTime"`
}

// AccountTradeListRequest holds the parameters of an account trade list request; StartTime and EndTime can be at most
// 24 hours apart
type AccountTradeListRequest struct {
	Symbol    currency.Pair
	OrderID   uint64
	StartTime time.Time
	EndTime   time.Time
	FromID    uint64 // Trades from this trade ID on
	Limit     uint64 // 500 when zero, at most 1000
}

// AccountTradeResponse holds a trade of the account
type AccountTradeResponse struct {
	Symbol          string        `json:"symbol"`
	ID              uint64        `json:"id"`
	OrderID         uint64        `json:"orderId"`
	OrderListID     int64         `json:"orderListId"` // -1 unless the order is part of an order list
	Price           types.Number  `json:"price"`
	Quantity        types.Number  `json:"qty"`
	QuoteQuantity   types.Number  `json:"quoteQty"`
	Commission      types.Number  `json:"commission"`
	CommissionAsset currency.Code `json:"commissionAsset"`
	Time            types.Time    `json:"time"`
	IsBuyer         bool          `json:"isBuyer"`
	IsMaker         bool          `json:"isMaker"`
	IsBestMatch     bool          `json:"isBestMatch"`
}

// BNBBurnStatusResponse represents a response of spot trade and margin interest
type BNBBurnStatusResponse struct {
	SpotBNBBurn     bool `json:"spotBNBBurn"`
	InterestBNBBurn bool `json:"interestBNBBurn"`
}

// ErrResponse holds error response information
type ErrResponse struct {
	Code    types.Number    `json:"code"`
	Message string          `json:"msg"`
	Data    json.RawMessage `json:"data"` // Only some endpoints add details, such as the outcome of a cancel-replace
}
