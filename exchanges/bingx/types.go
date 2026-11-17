package bingx

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

var (
	errInvalidNumericID      = errors.New("invalid numeric ID")
	errInvalidStopGuaranteed = errors.New("invalid guaranteed stop value")
	errInvalidPercentage     = errors.New("invalid percentage")
)

// Side is the side parameter of an order
type Side string

// Order sides
const (
	SideBuy  Side = "BUY"
	SideSell Side = "SELL"
)

// PositionSide is the positionSide parameter of a perpetual futures order or position request
type PositionSide string

// Position sides: BOTH in one-way position mode, LONG or SHORT in hedge mode
const (
	PositionSideBoth  PositionSide = "BOTH"
	PositionSideLong  PositionSide = "LONG"
	PositionSideShort PositionSide = "SHORT"
)

// OrderType is the type parameter of an order
type OrderType string

// Order types. Spot orders take MARKET, LIMIT, TAKE_STOP_LIMIT, TAKE_STOP_MARKET, TRIGGER_LIMIT and TRIGGER_MARKET.
// USDT-M perpetual orders take LIMIT, MARKET, STOP_MARKET, TAKE_PROFIT_MARKET, STOP, TAKE_PROFIT, TRIGGER_LIMIT,
// TRIGGER_MARKET, TRAILING_STOP_MARKET and TRAILING_TP_SL; coin-M perpetual orders take LIMIT, MARKET, STOP_MARKET,
// TAKE_PROFIT_MARKET, STOP and TAKE_PROFIT. Order events also report LIQUIDATION
const (
	OrderTypeLimit                        OrderType = "LIMIT"
	OrderTypeMarket                       OrderType = "MARKET"
	OrderTypeStopMarket                   OrderType = "STOP_MARKET"
	OrderTypeTakeProfitMarket             OrderType = "TAKE_PROFIT_MARKET"
	OrderTypeStop                         OrderType = "STOP"
	OrderTypeTakeProfit                   OrderType = "TAKE_PROFIT"
	OrderTypeTriggerLimit                 OrderType = "TRIGGER_LIMIT"
	OrderTypeTriggerMarket                OrderType = "TRIGGER_MARKET"
	OrderTypeTrailingStopMarket           OrderType = "TRAILING_STOP_MARKET"
	OrderTypeTrailingTakeProfitOrStopLoss OrderType = "TRAILING_TP_SL"
	OrderTypeTakeStopLimit                OrderType = "TAKE_STOP_LIMIT"
	OrderTypeTakeStopMarket               OrderType = "TAKE_STOP_MARKET"
	OrderTypeLiquidation                  OrderType = "LIQUIDATION"
)

// TimeInForce is how long an order stays on the order book, the timeInForce parameter of an order
type TimeInForce string

// Time in force values, taken by spot and perpetual orders
const (
	TimeInForceGoodTillCancel    TimeInForce = "GTC"
	TimeInForceImmediateOrCancel TimeInForce = "IOC"
	TimeInForceFillOrKill        TimeInForce = "FOK"
	TimeInForcePostOnly          TimeInForce = "PostOnly"
)

// WorkingType is the price that triggers a conditional perpetual order, its workingType parameter
type WorkingType string

// Working types; MARK_PRICE is the default, and a guaranteed stop loss takes CONTRACT_PRICE only
const (
	WorkingTypeMarkPrice     WorkingType = "MARK_PRICE"
	WorkingTypeContractPrice WorkingType = "CONTRACT_PRICE"
	WorkingTypeIndexPrice    WorkingType = "INDEX_PRICE"
)

// MarginType is the marginType parameter of a perpetual margin mode request
type MarginType string

// Margin types. Coin-M perpetuals take ISOLATED and CROSSED; USDT-M perpetuals also take SEPARATE_ISOLATED, which
// keeps each position's margin apart
const (
	MarginTypeIsolated         MarginType = "ISOLATED"
	MarginTypeCrossed          MarginType = "CROSSED"
	MarginTypeSeparateIsolated MarginType = "SEPARATE_ISOLATED"
)

// SwapContractStatus is the status of a USDT-M perpetual contract
type SwapContractStatus uint64

// Perpetual contract statuses; a contract forbidden to open positions can still close them
const (
	SwapContractStatusOffline                  SwapContractStatus = 0
	SwapContractStatusOnline                   SwapContractStatus = 1
	SwapContractStatusPreOnline                SwapContractStatus = 5
	SwapContractStatusForbiddenToOpenPositions SwapContractStatus = 25
)

// IncomeType is the incomeType parameter of a USDT-M perpetual fund flow query, and the type its records report
type IncomeType string

// Income types of the USDT-M perpetual fund flow
const (
	IncomeTypeTransfer         IncomeType = "TRANSFER"
	IncomeTypeRealisedPNL      IncomeType = "REALIZED_PNL"
	IncomeTypeFundingFee       IncomeType = "FUNDING_FEE"
	IncomeTypeTradingFee       IncomeType = "TRADING_FEE"
	IncomeTypeInsuranceClear   IncomeType = "INSURANCE_CLEAR" // liquidation
	IncomeTypeTrialFund        IncomeType = "TRIAL_FUND"
	IncomeTypeAutoDeleveraging IncomeType = "ADL"
	IncomeTypeSystemDeduction  IncomeType = "SYSTEM_DEDUCTION"
	IncomeTypeGuaranteedPrice  IncomeType = "GTD_PRICE"
)

// NonCryptoContractType is the type of a non-crypto (TradFi) perpetual contract
type NonCryptoContractType string

// Non-crypto perpetual contract types
const (
	NonCryptoContractTypeForex      NonCryptoContractType = "nonCryptoForex"
	NonCryptoContractTypeCommodity  NonCryptoContractType = "nonCryptoCommodity"
	NonCryptoContractTypeStock      NonCryptoContractType = "nonCryptoStock"
	NonCryptoContractTypeStockIndex NonCryptoContractType = "nonCryptoStockIndex"
)

// MarketStatus is the current market status of a non-crypto perpetual contract
type MarketStatus string

// Market statuses of a non-crypto perpetual contract; one in a rebase halt, pre-open or recovery phase is closed
const (
	MarketStatusOpen  MarketStatus = "open"
	MarketStatusClose MarketStatus = "close"
	MarketStatusPause MarketStatus = "pause"
)

// TradingSessionType is the type of a non-crypto perpetual contract's scheduled interval
type TradingSessionType string

// Trading session types. A regular session does not guarantee trading is open; the contract's market status tells
const (
	TradingSessionTypeRegular   TradingSessionType = "REGULAR"
	TradingSessionTypeNoTrading TradingSessionType = "NO_TRADING"
)

// StopGuaranteed is the stopGuaranteed parameter of a USDT-M perpetual conditional order, which turns the guaranteed
// stop loss and take profit on or off
type StopGuaranteed string

// Guaranteed stop values, taken by STOP_MARKET, TAKE_PROFIT_MARKET, STOP, TAKE_PROFIT, TRIGGER_LIMIT and
// TRIGGER_MARKET orders. cutfee turns the guarantee on with the VIP fee reduction, which applies to stop loss orders
// only. Orders report the value as a string or as a JSON boolean
const (
	StopGuaranteedTrue   StopGuaranteed = "true"
	StopGuaranteedFalse  StopGuaranteed = "false"
	StopGuaranteedCutFee StopGuaranteed = "cutfee"
)

// UnmarshalJSON decodes a guaranteed stop value from a JSON string or boolean; null leaves the value unchanged
func (s *StopGuaranteed) UnmarshalJSON(data []byte) error {
	switch string(data) {
	case jsonNull:
		return nil
	case "true":
		*s = StopGuaranteedTrue
		return nil
	case "false":
		*s = StopGuaranteedFalse
		return nil
	}
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf("%w: %s", errInvalidStopGuaranteed, data)
	}
	*s = StopGuaranteed(value)
	return nil
}

// AssetMode is the assetMode parameter of the USDT-M perpetual multi-assets mode requests
type AssetMode string

// Asset modes: single asset mode margins each contract with its settlement currency, and multi-assets mode lets the
// assets the multi-assets rules list margin any contract
const (
	AssetModeSingleAsset AssetMode = "singleAssetMode"
	AssetModeMultiAssets AssetMode = "multiAssetsMode"
)

// CancelReplaceMode is the cancelReplaceMode parameter of a cancel-replace request
type CancelReplaceMode string

// Cancel-replace modes: STOP_ON_FAILURE places the new order only when the cancellation succeeds, and ALLOW_FAILURE
// places it regardless. Spot and USDT-M perpetual cancel-replace requests take the same values
const (
	CancelReplaceModeStopOnFailure CancelReplaceMode = "STOP_ON_FAILURE"
	CancelReplaceModeAllowFailure  CancelReplaceMode = "ALLOW_FAILURE"
)

// CancelRestrictions is the cancelRestrictions parameter of a cancel-replace request, which cancels the order only in
// the given status
type CancelRestrictions string

// Cancel restrictions of USDT-M perpetual cancel-replace requests, which cancel the order only while it is new,
// pending or partially filled
const (
	CancelRestrictionsOnlyNew             CancelRestrictions = "ONLY_NEW"
	CancelRestrictionsOnlyPending         CancelRestrictions = "ONLY_PENDING"
	CancelRestrictionsOnlyPartiallyFilled CancelRestrictions = "ONLY_PARTIALLY_FILLED"
)

// CancelAllAfterType is the type parameter of a cancel all after request
type CancelAllAfterType string

// Cancel all after request types: ACTIVATE starts or extends the countdown and CLOSE stops it. Spot and USDT-M
// perpetual requests take the same values
const (
	CancelAllAfterTypeActivate CancelAllAfterType = "ACTIVATE"
	CancelAllAfterTypeClose    CancelAllAfterType = "CLOSE"
)

// AutoCloseType is the autoCloseType parameter of the USDT-M and coin-M perpetual force order queries
type AutoCloseType string

// Auto close types: liquidation orders, or orders that auto-deleveraging (ADL) placed
const (
	AutoCloseTypeLiquidation      AutoCloseType = "LIQUIDATION"
	AutoCloseTypeAutoDeleveraging AutoCloseType = "ADL"
)

// TradingUnit is the tradingUnit parameter of the USDT-M perpetual historical transaction orders query
type TradingUnit string

// Trading units: quantities in coins, such as BTC, or in contracts
const (
	TradingUnitCoin     TradingUnit = "COIN"
	TradingUnitContract TradingUnit = "CONT"
)

// SwapMarginAdjustmentType is the type parameter of a USDT-M perpetual isolated position margin change, and the type
// its reply reports
type SwapMarginAdjustmentType uint64

// USDT-M perpetual isolated margin adjustment types, which add margin to or remove margin from a position
const (
	SwapMarginAdjustmentTypeIncrease SwapMarginAdjustmentType = 1
	SwapMarginAdjustmentTypeDecrease SwapMarginAdjustmentType = 2
)

// VSTAdjustType is the adjustType parameter of a VST application on the demo trading host
type VSTAdjustType string

// VST adjustment types, which increase or decrease the demo account's VST
const (
	VSTAdjustTypeIncrease VSTAdjustType = "0"
	VSTAdjustTypeDecrease VSTAdjustType = "1"
)

// TWAPPriceType is the priceType parameter of a TWAP order, how its price variance limits the prices of its child
// orders
type TWAPPriceType string

// TWAP price types: constant takes the price variance as a price difference in USDT, and percentage as a slippage
// percentage
const (
	TWAPPriceTypeConstant   TWAPPriceType = "constant"
	TWAPPriceTypePercentage TWAPPriceType = "percentage"
)

// ReverseType is the type parameter of a one-click position reversal
type ReverseType string

// Reverse types: Reverse reverses the position at once, and TriggerReverse when its trigger price is reached
const (
	ReverseTypeReverse        ReverseType = "Reverse"
	ReverseTypeTriggerReverse ReverseType = "TriggerReverse"
)

// SpotSymbolStatus is the status of a spot trading symbol
type SpotSymbolStatus uint64

// Spot symbol statuses, as BingX names them
const (
	SpotSymbolStatusOffline     SpotSymbolStatus = 0
	SpotSymbolStatusOnline      SpotSymbolStatus = 1
	SpotSymbolStatusPreOpen     SpotSymbolStatus = 5
	SpotSymbolStatusAccessed    SpotSymbolStatus = 10
	SpotSymbolStatusSuspended   SpotSymbolStatus = 25
	SpotSymbolStatusPreDelisted SpotSymbolStatus = 29
	SpotSymbolStatusDelisted    SpotSymbolStatus = 30
)

// SpotTradeSide is the taker's side of a historical spot trade
type SpotTradeSide uint64

// Spot trade sides
const (
	SpotTradeSideBuy  SpotTradeSide = 1
	SpotTradeSideSell SpotTradeSide = 2
)

// SpotKlineTimeZone is the timeZone parameter of a spot candle request, the UTC offset candle periods start from
type SpotKlineTimeZone uint64

// Spot candle time zones. BingX applies UTC+8 when the parameter is absent, so a request always sends it
const (
	SpotKlineTimeZoneUTC      SpotKlineTimeZone = 0
	SpotKlineTimeZoneUTCPlus8 SpotKlineTimeZone = 8
)

// OrderBookAggregation is the type parameter of a spot aggregated order book request, the price precision its levels
// are merged to
type OrderBookAggregation string

// Order book aggregations: step0 keeps the symbol's price precision, and step1 to step5 merge levels to 10 to 100000
// times it
const (
	OrderBookAggregationStep0 OrderBookAggregation = "step0"
	OrderBookAggregationStep1 OrderBookAggregation = "step1"
	OrderBookAggregationStep2 OrderBookAggregation = "step2"
	OrderBookAggregationStep3 OrderBookAggregation = "step3"
	OrderBookAggregationStep4 OrderBookAggregation = "step4"
	OrderBookAggregationStep5 OrderBookAggregation = "step5"
)

// SpotOrderStatus is the status of a spot order, the status parameter of a spot order history request, and the
// cancelRestrictions parameter of a spot cancellation, which cancels an order only while it has that status
type SpotOrderStatus string

// Spot order statuses. Order history filters by FILLED, CANCELED or FAILED, and cancel restrictions take NEW, PENDING
// or PARTIALLY_FILLED
const (
	SpotOrderStatusNew             SpotOrderStatus = "NEW"
	SpotOrderStatusPending         SpotOrderStatus = "PENDING"
	SpotOrderStatusPartiallyFilled SpotOrderStatus = "PARTIALLY_FILLED"
	SpotOrderStatusFilled          SpotOrderStatus = "FILLED"
	SpotOrderStatusCancelled       SpotOrderStatus = "CANCELED"
	SpotOrderStatusFailed          SpotOrderStatus = "FAILED"
)

// CancelAllAfterStatus is the status a cancel all after request reports
type CancelAllAfterStatus string

// Cancel all after statuses
const (
	CancelAllAfterStatusActivated CancelAllAfterStatus = "ACTIVATED"
	CancelAllAfterStatusClosed    CancelAllAfterStatus = "CLOSED"
	CancelAllAfterStatusFailed    CancelAllAfterStatus = "FAILED"
)

// OCOOrderType is the type of an order in a spot one-cancels-the-other (OCO) order list
type OCOOrderType string

// OCO order types: the list's limit order and its stop-limit order
const (
	OCOOrderTypeLimit     OCOOrderType = "ocoLimit"
	OCOOrderTypeStopLimit OCOOrderType = "ocoTps"
)

// CoinMarginedMarginAdjustmentType is the type parameter of a coin-M isolated margin adjustment
type CoinMarginedMarginAdjustmentType string

// Coin-M isolated margin adjustment types, which add margin to (Increase) or remove margin from (Decrease) an isolated
// position. USDT-M adjustments take 1 and 2 instead
const (
	CoinMarginedMarginAdjustmentTypeIncrease CoinMarginedMarginAdjustmentType = "Increase"
	CoinMarginedMarginAdjustmentTypeDecrease CoinMarginedMarginAdjustmentType = "Decrease"
)

// WalletType is the walletType parameter of the internal transfer, withdrawal and deposit address endpoints, and the
// fromAccountType and toAccountType parameter of the sub-mother account transfers
type WalletType uint64

// Wallet types. The English text of some pages gives 4 for the spot account, while their Chinese text, the other pages
// and BingX's changelog give 15
const (
	WalletTypeFund             WalletType = 1
	WalletTypeStandardFutures  WalletType = 2
	WalletTypePerpetualFutures WalletType = 3
	WalletTypeSpot             WalletType = 15
)

// UserAccountType is the userAccountType parameter of an internal transfer, the kind of identifier userAccount holds
type UserAccountType uint64

// User account types; a phone number also needs its calling code
const (
	UserAccountTypeUID         UserAccountType = 1
	UserAccountTypePhoneNumber UserAccountType = 2
	UserAccountTypeEmail       UserAccountType = 3
)

// TransferAccount is the fromAccount and toAccount parameter of a transfer between the account's own accounts
type TransferAccount string

// Transfer accounts
const (
	TransferAccountFund                  TransferAccount = "fund"
	TransferAccountSpot                  TransferAccount = "spot"
	TransferAccountStandardFutures       TransferAccount = "stdFutures"
	TransferAccountCoinMarginedPerpetual TransferAccount = "coinMPerp"
	TransferAccountUSDTMarginedPerpetual TransferAccount = "USDTMPerp"
)

// AssetTransferType is the type parameter of the asset transfer records, the accounts a transfer moved between
type AssetTransferType string

// Asset transfer types. Strategy trading is the grid account. The current documentation lists none; these are the
// values BingX's earlier documentation lists
const (
	AssetTransferTypeFundToStandardFutures             AssetTransferType = "FUND_SFUTURES"
	AssetTransferTypeStandardFuturesToFund             AssetTransferType = "SFUTURES_FUND"
	AssetTransferTypeFundToPerpetualFutures            AssetTransferType = "FUND_PFUTURES"
	AssetTransferTypePerpetualFuturesToFund            AssetTransferType = "PFUTURES_FUND"
	AssetTransferTypeStandardFuturesToPerpetualFutures AssetTransferType = "SFUTURES_PFUTURES"
	AssetTransferTypePerpetualFuturesToStandardFutures AssetTransferType = "PFUTURES_SFUTURES"
	AssetTransferTypeFundToStrategyTrading             AssetTransferType = "FUND_STRADING"
	AssetTransferTypeStrategyTradingToFund             AssetTransferType = "STRADING_FUND"
	AssetTransferTypeFundToCopyTrading                 AssetTransferType = "FUND_CTRADING"
	AssetTransferTypeStandardFuturesToCopyTrading      AssetTransferType = "SFUTURES_CTRADING"
	AssetTransferTypePerpetualFuturesToCopyTrading     AssetTransferType = "PFUTURES_CTRADING"
	AssetTransferTypeCopyTradingToFund                 AssetTransferType = "CTRADING_FUND"
	AssetTransferTypeCopyTradingToStandardFutures      AssetTransferType = "CTRADING_SFUTURES"
	AssetTransferTypeCopyTradingToPerpetualFutures     AssetTransferType = "CTRADING_PFUTURES"
)

// AccountType is the accountType parameter of the asset overview requests, and the account each overview balance is
// for
type AccountType string

// Account types. AccountTypeSpot covers the spot and fund accounts; the spot and wealth values keep BingX's spellings,
// sopt and eran
const (
	AccountTypeSpot                  AccountType = "sopt"
	AccountTypeStandardFutures       AccountType = "stdFutures"
	AccountTypeCoinMarginedPerpetual AccountType = "coinMPerp"
	AccountTypeUSDTMarginedPerpetual AccountType = "USDTMPerp"
	AccountTypeCopyTrading           AccountType = "copyTrading"
	AccountTypeGrid                  AccountType = "grid"
	AccountTypeWealth                AccountType = "eran"
	AccountTypeC2C                   AccountType = "c2c"
)

// InternalTransferRecordType is the recordType of an internal transfer record, whether the account sent or received it
type InternalTransferRecordType string

// Internal transfer record types
const (
	InternalTransferRecordTypeOut InternalTransferRecordType = "out"
	InternalTransferRecordTypeIn  InternalTransferRecordType = "in"
)

// DepositStatus is the status of a deposit, and the status filter of the deposit record queries
type DepositStatus uint64

// Deposit statuses
const (
	DepositStatusInProgress    DepositStatus = 0
	DepositStatusCompleted     DepositStatus = 1
	DepositStatusChainUploaded DepositStatus = 6
)

// WithdrawStatus is the status of a withdrawal or an internal transfer, and the status filter of the withdrawal records
type WithdrawStatus uint64

// Withdraw statuses
const (
	WithdrawStatusUnderReview WithdrawStatus = 4
	WithdrawStatusFailed      WithdrawStatus = 5
	WithdrawStatusCompleted   WithdrawStatus = 6
)

// TransferType is the transferType of a deposit or withdrawal record, whether it moved on chain or within BingX
type TransferType uint64

// Transfer types: deposit records report TransferTypeDeposit, and withdrawal records TransferTypeWithdrawal or
// TransferTypeInternalTransfer
const (
	TransferTypeDeposit          TransferType = 0
	TransferTypeWithdrawal       TransferType = 1
	TransferTypeInternalTransfer TransferType = 2
)

// VASPEntityID is the vaspEntityId parameter of a withdrawal by a Korean individual user, the virtual asset service
// provider it is sent to
type VASPEntityID string

// VASP entity IDs; VASPEntityIDOthers is for a provider not listed, and needs no recipient details
const (
	VASPEntityIDBithumb VASPEntityID = "Bithumb"
	VASPEntityIDCoinone VASPEntityID = "Coinone"
	VASPEntityIDHexlant VASPEntityID = "Hexlant"
	VASPEntityIDKorbit  VASPEntityID = "Korbit"
	VASPEntityIDUpbit   VASPEntityID = "Upbit"
	VASPEntityIDOthers  VASPEntityID = "Others"
)

// DepositAddressStatus is the status of a deposit address
type DepositAddressStatus uint64

// Deposit address statuses
const (
	DepositAddressStatusActivated  DepositAddressStatus = 0
	DepositAddressStatusApplied    DepositAddressStatus = 1
	DepositAddressStatusNotApplied DepositAddressStatus = 2
)

// AccountStatus is the status of an account
type AccountStatus uint64

// Account statuses
const (
	AccountStatusNormal       AccountStatus = 0
	AccountStatusDeregistered AccountStatus = 1
	AccountStatusLocked       AccountStatus = 2
)

// APIKeyPermission is a permission of an API key
type APIKeyPermission uint64

// API key permissions. Professional contract trading is perpetual futures trading
const (
	APIKeyPermissionSpotTrading                 APIKeyPermission = 1
	APIKeyPermissionRead                        APIKeyPermission = 2
	APIKeyPermissionProfessionalContractTrading APIKeyPermission = 3
	APIKeyPermissionUniversalTransfer           APIKeyPermission = 4
	APIKeyPermissionWithdraw                    APIKeyPermission = 5
	APIKeyPermissionSubAccountInternalTransfer  APIKeyPermission = 7
)

// SubAccountType is the fromType and toType parameter of a sub-mother account asset transfer, whether an account is the
// parent account or a sub-account
type SubAccountType uint64

// Sub-account types
const (
	SubAccountTypeParentAccount SubAccountType = 1
	SubAccountTypeSubAccount    SubAccountType = 2
)

// SubMotherAccountTransferType is the type of a transfer between the parent account and a sub-account, and the type
// filter of the sub-mother account transfer history. Capital is the fund account, contract the standard futures
// account and swap the USDT-M perpetual futures account. The current documentation lists none; these are the values
// BingX's earlier documentation lists
type SubMotherAccountTransferType string

// Sub-mother account transfer types
const (
	SubMotherAccountTransferTypeMainCapitalToSubCapital   SubMotherAccountTransferType = "MAIN_CAPITAL_TO_SUB_CAPITAL"
	SubMotherAccountTransferTypeMainCapitalToSubContract  SubMotherAccountTransferType = "MAIN_CAPITAL_TO_SUB_CONTRACT"
	SubMotherAccountTransferTypeMainCapitalToSubSwap      SubMotherAccountTransferType = "MAIN_CAPITAL_TO_SUB_SWAP"
	SubMotherAccountTransferTypeMainContractToSubCapital  SubMotherAccountTransferType = "MAIN_CONTRACT_TO_SUB_CAPITAL"
	SubMotherAccountTransferTypeMainContractToSubContract SubMotherAccountTransferType = "MAIN_CONTRACT_TO_SUB_CONTRACT"
	SubMotherAccountTransferTypeMainContractToSubSwap     SubMotherAccountTransferType = "MAIN_CONTRACT_TO_SUB_SWAP"
	SubMotherAccountTransferTypeMainSwapToSubCapital      SubMotherAccountTransferType = "MAIN_SWAP_TO_SUB_CAPITAL"
	SubMotherAccountTransferTypeMainSwapToSubContract     SubMotherAccountTransferType = "MAIN_SWAP_TO_SUB_CONTRACT"
	SubMotherAccountTransferTypeMainSwapToSubSwap         SubMotherAccountTransferType = "MAIN_SWAP_TO_SUB_SWAP"
	SubMotherAccountTransferTypeSubCapitalToMainCapital   SubMotherAccountTransferType = "SUB_CAPITAL_TO_MAIN_CAPITAL"
	SubMotherAccountTransferTypeSubCapitalToMainContract  SubMotherAccountTransferType = "SUB_CAPITAL_TO_MAIN_CONTRACT"
	SubMotherAccountTransferTypeSubCapitalToMainSwap      SubMotherAccountTransferType = "SUB_CAPITAL_TO_MAIN_SWAP"
	SubMotherAccountTransferTypeSubContractToMainCapital  SubMotherAccountTransferType = "SUB_CONTRACT_TO_MAIN_CAPITAL"
	SubMotherAccountTransferTypeSubContractToMainContract SubMotherAccountTransferType = "SUB_CONTRACT_TO_MAIN_CONTRACT"
	SubMotherAccountTransferTypeSubContractToMainSwap     SubMotherAccountTransferType = "SUB_CONTRACT_TO_MAIN_SWAP"
	SubMotherAccountTransferTypeSubSwapToMainCapital      SubMotherAccountTransferType = "SUB_SWAP_TO_MAIN_CAPITAL"
	SubMotherAccountTransferTypeSubSwapToMainContract     SubMotherAccountTransferType = "SUB_SWAP_TO_MAIN_CONTRACT"
	SubMotherAccountTransferTypeSubSwapToMainSwap         SubMotherAccountTransferType = "SUB_SWAP_TO_MAIN_SWAP"
)

// AgentBusinessType is the businessType parameter of the agent commission queries
type AgentBusinessType string

// Agent business types. Daily commission details take every value, and channel commission details take
// perpetualFutures, standardFutures and spot. perpetualFutures includes the TradFi perpetuals, which tradFi selects
// alone
const (
	AgentBusinessTypePerpetualFutures   AgentBusinessType = "perpetualFutures"
	AgentBusinessTypeStandardFutures    AgentBusinessType = "standardFutures"
	AgentBusinessTypeSpot               AgentBusinessType = "spot"
	AgentBusinessTypeCopyTradePro       AgentBusinessType = "copyTradePro"
	AgentBusinessTypeTraditionalFinance AgentBusinessType = "tradFi"
)

// AgentInvitationType is how an agent's user was invited: the distanceType parameter of daily commission details, and
// the referralType partner information reports
type AgentInvitationType uint64

// Agent invitation types
const (
	AgentInvitationTypeDirect   AgentInvitationType = 1
	AgentInvitationTypeIndirect AgentInvitationType = 2
)

// AgentDepositBusinessType is the bizType parameter of the invited users' deposit details, and the business type each
// deposit reports
type AgentDepositBusinessType uint64

// AgentDepositBusinessTypeCryptoDeposit is the only documented deposit business type, a crypto deposit
const AgentDepositBusinessTypeCryptoDeposit AgentDepositBusinessType = 1

// AgentCommissionBusinessType is the commissionBizType parameter of the API transaction commission query, and the
// business type each commission reports
type AgentCommissionBusinessType uint64

// Agent commission business types, the commission on API trades of perpetual futures or of spot
const (
	AgentCommissionBusinessTypePerpetualFutures AgentCommissionBusinessType = 81
	AgentCommissionBusinessTypeSpot             AgentCommissionBusinessType = 82
)

// AgentAssetType is the assetType parameter of channel commission details
type AgentAssetType uint64

// Agent asset types, crypto or traditional finance assets
const (
	AgentAssetTypeCrypto             AgentAssetType = 1
	AgentAssetTypeTraditionalFinance AgentAssetType = 2
)

// AgentBenefit is the benefit an invited user currently receives, which the invited user queries report
type AgentBenefit uint64

// Agent benefits: none, a cashback of trading fees or a discount on perpetual futures fees
const (
	AgentBenefitNone                 AgentBenefit = 0
	AgentBenefitFeeCashback          AgentBenefit = 1
	AgentBenefitPerpetualFeeDiscount AgentBenefit = 2
)

// CopyTradingContractType is the contractType parameter of the copy trading pairs query, and the contract type the
// perpetual copy trading replies report
type CopyTradingContractType string

// Copy trading contract types, standard futures or USDT-M perpetual futures
const (
	CopyTradingContractTypeStandardFutures  CopyTradingContractType = "SFUTURES"
	CopyTradingContractTypePerpetualFutures CopyTradingContractType = "PFUTURES"
)

// CopyTradingAccountLevel is a trader's copy trading level, which the perpetual trading overview reports
type CopyTradingAccountLevel string

// Copy trading account levels
const (
	CopyTradingAccountLevelBronze  CopyTradingAccountLevel = "Bronze"
	CopyTradingAccountLevelSilver  CopyTradingAccountLevel = "Silver"
	CopyTradingAccountLevelGold    CopyTradingAccountLevel = "Gold"
	CopyTradingAccountLevelDiamond CopyTradingAccountLevel = "Diamond"
)

// CopyTradingOpenTrader is whether a trader leads copy trading in a pair, which the copy trading pairs report
type CopyTradingOpenTrader string

// Copy trading open trader values
const (
	CopyTradingOpenTraderYes CopyTradingOpenTrader = "yes"
	CopyTradingOpenTraderNo  CopyTradingOpenTrader = "no"
)

// CopyTradingSpotOrderStatus is the status of the order that sells a trader's spot assets
type CopyTradingSpotOrderStatus uint64

// Copy trading spot order statuses
const (
	CopyTradingSpotOrderStatusNotTriggered CopyTradingSpotOrderStatus = 5
	CopyTradingSpotOrderStatusProcessing   CopyTradingSpotOrderStatus = 10
	// CopyTradingSpotOrderStatusInCommission is an order placed and awaiting execution
	CopyTradingSpotOrderStatusInCommission CopyTradingSpotOrderStatus = 11
	CopyTradingSpotOrderStatusCancelling   CopyTradingSpotOrderStatus = 20
	CopyTradingSpotOrderStatusSuccess      CopyTradingSpotOrderStatus = 30
	CopyTradingSpotOrderStatusCancelled    CopyTradingSpotOrderStatus = 31
	CopyTradingSpotOrderStatusFailed       CopyTradingSpotOrderStatus = 40
)

// DualCurrencyDirection is the direction parameter of the dual-currency product list, and the direction dual-currency
// products, quotes, orders and positions report
type DualCurrencyDirection string

// Dual-currency directions: CALL sells high and PUT buys low
const (
	DualCurrencyDirectionCall DualCurrencyDirection = "CALL"
	DualCurrencyDirectionPut  DualCurrencyDirection = "PUT"
)

// DualCurrencyAccount is an account a dual-currency order is paid from, a value of its selectAccounts parameter, or
// settled to, its settleTargetAccount parameter
type DualCurrencyAccount string

// Dual-currency accounts, the fund account or the spot account
const (
	DualCurrencyAccountFund DualCurrencyAccount = "FUND"
	DualCurrencyAccountSpot DualCurrencyAccount = "SPOT"
)

// DualCurrencyOrderStatus is the status of a dual-currency order, which positions and order records report
type DualCurrencyOrderStatus uint64

// Dual-currency order statuses
const (
	DualCurrencyOrderStatusProcessing       DualCurrencyOrderStatus = 2
	DualCurrencyOrderStatusCancelled        DualCurrencyOrderStatus = 4
	DualCurrencyOrderStatusConfirmingShares DualCurrencyOrderStatus = 5
	DualCurrencyOrderStatusSettling         DualCurrencyOrderStatus = 6
	DualCurrencyOrderStatusSettled          DualCurrencyOrderStatus = 7
	DualCurrencyOrderStatusRefunded         DualCurrencyOrderStatus = 9
)

// DualCurrencyBusinessCode is whether BingX handled a dual-currency quote or order normally, its bizCode
type DualCurrencyBusinessCode uint64

// Dual-currency business codes
const (
	DualCurrencyBusinessCodeNormal   DualCurrencyBusinessCode = 0
	DualCurrencyBusinessCodeAbnormal DualCurrencyBusinessCode = 1
)

// SignalSide is the side parameter of a signal
type SignalSide string

// Signal sides
const (
	SignalSideLong  SignalSide = "long"
	SignalSideShort SignalSide = "short"
)

// ActivityStatus is the status parameter of the activity list queries
type ActivityStatus uint64

// Activity statuses; the activity lists return ongoing activities when no status is given
const (
	ActivityStatusStartingSoon ActivityStatus = 1
	ActivityStatusOngoing      ActivityStatus = 2
)

// WebsocketRequestType is the reqType of a websocket subscription request
type WebsocketRequestType string

// Websocket request types
const (
	WebsocketRequestTypeSubscribe   WebsocketRequestType = "sub"
	WebsocketRequestTypeUnsubscribe WebsocketRequestType = "unsub"
)

// ExecutionType is what happened to an order in an order update event, its x field
type ExecutionType string

// Execution types. Perpetual order events report CALCULATED for an auto-deleveraging or liquidation fill
const (
	ExecutionTypeNew        ExecutionType = "NEW"
	ExecutionTypeCancelled  ExecutionType = "CANCELED"
	ExecutionTypeCalculated ExecutionType = "CALCULATED"
	ExecutionTypeExpired    ExecutionType = "EXPIRED"
	ExecutionTypeTrade      ExecutionType = "TRADE"
)

// AccountUpdateReason is why an account balance event was pushed, its m field
type AccountUpdateReason string

// Account update reasons. Spot events document every reason; perpetual events document DEPOSIT, WITHDRAW, ORDER and
// FUNDING_FEE
const (
	AccountUpdateReasonInitial             AccountUpdateReason = "INIT"
	AccountUpdateReasonDeposit             AccountUpdateReason = "DEPOSIT"
	AccountUpdateReasonWithdraw            AccountUpdateReason = "WITHDRAW"
	AccountUpdateReasonOrder               AccountUpdateReason = "ORDER"
	AccountUpdateReasonFundingFee          AccountUpdateReason = "FUNDING_FEE"
	AccountUpdateReasonWithdrawReject      AccountUpdateReason = "WITHDRAW_REJECT"
	AccountUpdateReasonAdjustment          AccountUpdateReason = "ADJUSTMENT"
	AccountUpdateReasonInsuranceClear      AccountUpdateReason = "INSURANCE_CLEAR"
	AccountUpdateReasonAdminDeposit        AccountUpdateReason = "ADMIN_DEPOSIT"
	AccountUpdateReasonAdminWithdraw       AccountUpdateReason = "ADMIN_WITHDRAW"
	AccountUpdateReasonMarginTransfer      AccountUpdateReason = "MARGIN_TRANSFER"
	AccountUpdateReasonMarginTypeChange    AccountUpdateReason = "MARGIN_TYPE_CHANGE"
	AccountUpdateReasonAssetTransfer       AccountUpdateReason = "ASSET_TRANSFER"
	AccountUpdateReasonOptionsPremiumFee   AccountUpdateReason = "OPTIONS_PREMIUM_FEE"
	AccountUpdateReasonOptionsSettleProfit AccountUpdateReason = "OPTIONS_SETTLE_PROFIT"
	AccountUpdateReasonAutoExchange        AccountUpdateReason = "AUTO_EXCHANGE"
)

// Percentage is a percentage BingX sends as a string with or without a percent sign, such as "-1.05%", or as a number,
// holding the percentage shown: -1.05
type Percentage float64

// UnmarshalJSON decodes a percentage from a string, with or without a trailing percent sign, or from a number. An empty
// string or null decodes to zero
func (p *Percentage) UnmarshalJSON(data []byte) error {
	s := string(data)
	if s == jsonNull {
		*p = 0
		return nil
	}
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}
	if s == "" {
		*p = 0
		return nil
	}
	v, err := strconv.ParseFloat(strings.TrimSuffix(s, "%"), 64)
	if err != nil {
		return fmt.Errorf("%w: %s", errInvalidPercentage, data)
	}
	*p = Percentage(v)
	return nil
}

// Float64 returns the percentage as a float64
func (p Percentage) Float64() float64 {
	return float64(p)
}

// SystemTimeResponse holds the server time
type SystemTimeResponse struct {
	ServerTime types.Time `json:"serverTime"`
}

// NumericID is an identifier BingX sends as a JSON number on some endpoints and as a string on others. Integer JSON
// numbers decode exactly, so identifiers above 2^53 keep every digit
type NumericID uint64

// UnmarshalJSON decodes a JSON number or a string holding one; an empty string or null decodes to zero
func (n *NumericID) UnmarshalJSON(data []byte) error {
	s := string(data)
	if s == jsonNull || s == `""` {
		*n = 0
		return nil
	}
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return fmt.Errorf("%w: %q: %w", errInvalidNumericID, data, err)
	}
	*n = NumericID(v)
	return nil
}

// String returns the identifier in decimal
func (n NumericID) String() string {
	return strconv.FormatUint(uint64(n), 10)
}

// objectOrArray decodes a response that holds one object when a symbol is requested and an array of them otherwise
type objectOrArray[T any] []T

// UnmarshalJSON decodes a JSON object as a one-element list and a JSON array as a list
func (o *objectOrArray[T]) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) > 0 && trimmed[0] == '{' {
		var single T
		if err := json.Unmarshal(trimmed, &single); err != nil {
			return err
		}
		*o = objectOrArray[T]{single}
		return nil
	}
	var list []T
	if err := json.Unmarshal(trimmed, &list); err != nil {
		return err
	}
	*o = list
	return nil
}
