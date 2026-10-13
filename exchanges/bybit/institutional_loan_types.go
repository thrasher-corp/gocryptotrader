package bybit

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// InstitutionalLoanProductInfoResponse holds institutional loan products
type InstitutionalLoanProductInfoResponse struct {
	MarginProductInfo []InstitutionalLoanProduct `json:"marginProductInfo"`
}

// InstitutionalLoanProduct holds an institutional loan product's leverage, restriction lines and symbol whitelists.
// An empty symbol whitelist allows every symbol of a supported product type
type InstitutionalLoanProduct struct {
	ProductID               string                            `json:"productId"`
	Leverage                types.Number                      `json:"leverage"`
	SupportSpot             types.Boolean                     `json:"supportSpot"`
	SupportContract         types.Boolean                     `json:"supportContract"` // USDT perpetuals
	SupportMarginTrading    types.Boolean                     `json:"supportMarginTrading"`
	DeferredLiquidationLine types.Number                      `json:"deferredLiquidationLine"`
	DeferredLiquidationTime string                            `json:"deferredLiquidationTime"` // Format undocumented; empty in every sample
	WithdrawLine            types.Number                      `json:"withdrawLine"`
	TransferLine            types.Number                      `json:"transferLine"`
	SpotBuyLine             types.Number                      `json:"spotBuyLine"`
	SpotSellLine            types.Number                      `json:"spotSellLine"`
	ContractOpenLine        types.Number                      `json:"contractOpenLine"`
	LiquidationLine         types.Number                      `json:"liquidationLine"`
	StopLiquidationLine     types.Number                      `json:"stopLiquidationLine"`
	ContractLeverage        types.Number                      `json:"contractLeverage"`
	TransferRatio           types.Number                      `json:"transferRatio"`
	SpotSymbols             []string                          `json:"spotSymbols"`
	ContractSymbols         []string                          `json:"contractSymbols"`
	SupportUSDCContract     types.Boolean                     `json:"supportUSDCContract"`
	SupportUSDCOptions      types.Boolean                     `json:"supportUSDCOptions"`
	USDTPerpetualOpenLine   types.Number                      `json:"USDTPerpetualOpenLine"`
	USDCContractOpenLine    types.Number                      `json:"USDCContractOpenLine"`
	USDCOptionsOpenLine     types.Number                      `json:"USDCOptionsOpenLine"`
	USDTPerpetualCloseLine  types.Number                      `json:"USDTPerpetualCloseLine"`
	USDCContractCloseLine   types.Number                      `json:"USDCContractCloseLine"`
	USDCOptionsCloseLine    types.Number                      `json:"USDCOptionsCloseLine"`
	USDCContractSymbols     []string                          `json:"USDCContractSymbols"`
	USDCOptionsSymbols      []string                          `json:"USDCOptionsSymbols"`
	MarginLeverage          types.Number                      `json:"marginLeverage"`
	USDTPerpetualLeverage   []InstitutionalLoanSymbolLeverage `json:"USDTPerpetualLeverage"`
	USDCContractLeverage    []InstitutionalLoanSymbolLeverage `json:"USDCContractLeverage"`
	// ProductType is documented as 0, 1 or 2 but sent as its name: Default, CTA or Hedge
	ProductType string `json:"productType"`
}

// InstitutionalLoanSymbolLeverage holds the maximum leverage of a symbol under an institutional loan
type InstitutionalLoanSymbolLeverage struct {
	Symbol   string       `json:"symbol"`
	Leverage types.Number `json:"leverage"`
}

// InstitutionalLoanMarginCoinInfoResponse holds the margin coins of institutional loan products
type InstitutionalLoanMarginCoinInfoResponse struct {
	MarginToken []InstitutionalLoanMarginToken `json:"marginToken"`
}

// InstitutionalLoanMarginToken holds the margin coins of an institutional loan product
type InstitutionalLoanMarginToken struct {
	ProductID string                             `json:"productId"`
	TokenInfo []InstitutionalLoanMarginTokenInfo `json:"tokenInfo"`
}

// InstitutionalLoanMarginTokenInfo holds a margin coin's tiered conversion ratios
type InstitutionalLoanMarginTokenInfo struct {
	Token            currency.Code                   `json:"token"`
	ConvertRatioList []InstitutionalLoanConvertRatio `json:"convertRatioList"`
}

// InstitutionalLoanConvertRatio holds the conversion ratio of a margin coin amount tier
type InstitutionalLoanConvertRatio struct {
	Ladder       string       `json:"ladder"` // Amount range of the tier, such as "0-500"
	ConvertRatio types.Number `json:"convertRatio"`
}

// InstitutionalLoanOrdersRequest holds the filters of GetInstitutionalLoanOrders; every field is optional
type InstitutionalLoanOrdersRequest struct {
	OrderID   string
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
}

// InstitutionalLoanOrdersResponse holds institutional loan orders
type InstitutionalLoanOrdersResponse struct {
	LoanInfo []InstitutionalLoanOrder `json:"loanInfo"`
}

// InstitutionalLoanOrder holds an institutional loan order and the product terms it was granted under
type InstitutionalLoanOrder struct {
	OrderID                 string                            `json:"orderId"`
	OrderProductID          string                            `json:"orderProductId"`
	ParentUID               string                            `json:"parentUid"` // UID bound to the loan
	LoanTime                types.Time                        `json:"loanTime"`
	LoanCoin                currency.Code                     `json:"loanCoin"`
	LoanAmount              types.Number                      `json:"loanAmount"`
	UnpaidAmount            types.Number                      `json:"unpaidAmount"` // Unpaid principal
	UnpaidInterest          types.Number                      `json:"unpaidInterest"`
	RepaidAmount            types.Number                      `json:"repaidAmount"` // Repaid principal
	RepaidInterest          types.Number                      `json:"repaidInterest"`
	InterestRate            types.Number                      `json:"interestRate"` // Daily rate
	Status                  string                            `json:"status"`       // 1: outstanding, 2: paid off
	Leverage                types.Number                      `json:"leverage"`
	SupportSpot             types.Boolean                     `json:"supportSpot"`
	SupportContract         types.Boolean                     `json:"supportContract"`
	WithdrawLine            types.Number                      `json:"withdrawLine"`
	TransferLine            types.Number                      `json:"transferLine"`
	SpotBuyLine             types.Number                      `json:"spotBuyLine"`
	SpotSellLine            types.Number                      `json:"spotSellLine"`
	ContractOpenLine        types.Number                      `json:"contractOpenLine"`
	DeferredLiquidationLine types.Number                      `json:"deferredLiquidationLine"`
	DeferredLiquidationTime string                            `json:"deferredLiquidationTime"` // Format undocumented; empty in every sample
	ReserveToken            currency.Code                     `json:"reserveToken"`
	ReserveQuantity         types.Number                      `json:"reserveQuantity"`
	LiquidationLine         types.Number                      `json:"liquidationLine"`
	StopLiquidationLine     types.Number                      `json:"stopLiquidationLine"`
	ContractLeverage        types.Number                      `json:"contractLeverage"`
	TransferRatio           types.Number                      `json:"transferRatio"`
	SpotSymbols             []string                          `json:"spotSymbols"`
	ContractSymbols         []string                          `json:"contractSymbols"`
	SupportUSDCContract     types.Boolean                     `json:"supportUSDCContract"`
	SupportUSDCOptions      types.Boolean                     `json:"supportUSDCOptions"`
	SupportMarginTrading    types.Boolean                     `json:"supportMarginTrading"`
	USDTPerpetualOpenLine   types.Number                      `json:"USDTPerpetualOpenLine"`
	USDCContractOpenLine    types.Number                      `json:"USDCContractOpenLine"`
	USDCOptionsOpenLine     types.Number                      `json:"USDCOptionsOpenLine"`
	USDTPerpetualCloseLine  types.Number                      `json:"USDTPerpetualCloseLine"`
	USDCContractCloseLine   types.Number                      `json:"USDCContractCloseLine"`
	USDCOptionsCloseLine    types.Number                      `json:"USDCOptionsCloseLine"`
	USDCContractSymbols     []string                          `json:"USDCContractSymbols"`
	USDCOptionsSymbols      []string                          `json:"USDCOptionsSymbols"`
	MarginLeverage          types.Number                      `json:"marginLeverage"`
	USDTPerpetualLeverage   []InstitutionalLoanSymbolLeverage `json:"USDTPerpetualLeverage"`
	USDCContractLeverage    []InstitutionalLoanSymbolLeverage `json:"USDCContractLeverage"`
}

// InstitutionalLoanRepaymentOrdersRequest holds the filters of GetInstitutionalLoanRepaymentOrders; every field is
// optional
type InstitutionalLoanRepaymentOrdersRequest struct {
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
}

// InstitutionalLoanRepaymentOrdersResponse holds institutional loan repayment orders
type InstitutionalLoanRepaymentOrdersResponse struct {
	RepayInfo []InstitutionalLoanRepaymentOrder `json:"repayInfo"`
}

// InstitutionalLoanRepaymentOrder holds an institutional loan repayment order
type InstitutionalLoanRepaymentOrder struct {
	RepayOrderID string        `json:"repayOrderId"`
	RepaidTime   types.Time    `json:"repaidTime"`
	Token        currency.Code `json:"token"`
	Quantity     types.Number  `json:"quantity"` // Repaid principal
	Interest     types.Number  `json:"interest"`
	BusinessType string        `json:"businessType"` // 1: normal repayment, 2: repaid by liquidation
	Status       string        `json:"status"`       // 1: success, 2: fail
}

// InstitutionalLoanLTVResponse holds the loan-to-value ratio of each risk unit
type InstitutionalLoanLTVResponse struct {
	LTVInfo []InstitutionalLoanLTV `json:"ltvInfo"`
	// LiquidationStatus is 0 normal, 1 under liquidation, 2 manual repayment in progress or 3 transfer in progress
	LiquidationStatus uint64 `json:"liqStatus"`
}

// InstitutionalLoanLTV holds a risk unit's loan-to-value ratio, debts and margin assets. While LiquidationStatus is not
// 0, Bybit sends LTV, UnpaidAmount and Balance empty, which decode as 0, and UnpaidInfo and BalanceInfo empty
type InstitutionalLoanLTV struct {
	LTV types.Number `json:"ltv"`
	// RemainingLiquidationTime is documented only as "remaining liquidation time (UTC time in seconds)", which may be a
	// countdown or a Unix time, so it is kept as a number; it is empty until liquidation is triggered
	RemainingLiquidationTime types.Number               `json:"rst"`
	ParentUID                string                     `json:"parentUid"` // Designated risk unit UID bound to the loan
	SubAccountUIDs           []string                   `json:"subAccountUids"`
	UnpaidAmount             types.Number               `json:"unpaidAmount"` // Total debt in USD
	UnpaidInfo               []InstitutionalLoanUnpaid  `json:"unpaidInfo"`
	Balance                  types.Number               `json:"balance"` // Total margin assets converted to USDT
	BalanceInfo              []InstitutionalLoanBalance `json:"balanceInfo"`
	LiquidationStatus        uint64                     `json:"liqStatus"` // As InstitutionalLoanLTVResponse.LiquidationStatus
}

// InstitutionalLoanUnpaid holds a coin's institutional loan debt
type InstitutionalLoanUnpaid struct {
	Token          currency.Code `json:"token"`
	UnpaidQuantity types.Number  `json:"unpaidQty"`      // Unpaid principal
	UnpaidInterest types.Number  `json:"unpaidInterest"` // Bybit documents this field as unused
}

// InstitutionalLoanBalance holds a margin coin's balance and its converted value
type InstitutionalLoanBalance struct {
	Token           currency.Code `json:"token"`
	Price           types.Number  `json:"price"`
	Quantity        types.Number  `json:"qty"`
	ConvertedAmount types.Number  `json:"convertedAmount"`
}

// BindOrUnbindUIDResponse holds the UID bound to or unbound from an institutional loan risk unit
type BindOrUnbindUIDResponse struct {
	UID     string `json:"uid"`
	Operate string `json:"operate"` // 0: bind, 1: unbind
}

// RepayInstitutionalLoanResponse holds the status of an institutional loan repayment
type RepayInstitutionalLoanResponse struct {
	RepayOrderStatus string `json:"repayOrderStatus"` // P: processing
}

// InstitutionalLoanCoinDeltaAmountResponse holds the delta limits of an institutional loan hedge product's risk unit
// and coins
type InstitutionalLoanCoinDeltaAmountResponse struct {
	RiskUnitDeltaAmount          types.Number                     `json:"riskUnitDeltaAmount"`          // Total limit in USD
	RiskUnitDeltaAvailableAmount types.Number                     `json:"riskUnitDeltaAvailableAmount"` // In USD
	RiskUnitDelta                types.Number                     `json:"riskUnitDelta"`
	List                         []InstitutionalLoanCoinDeltaInfo `json:"list"`
}

// InstitutionalLoanCoinDeltaInfo holds a coin's delta and delta limits
type InstitutionalLoanCoinDeltaInfo struct {
	Coin                     currency.Code `json:"coin"`
	CoinDeltaSize            types.Number  `json:"coinDeltaSize"`            // In coin
	CoinDeltaAvailableAmount types.Number  `json:"coinDeltaAvailableAmount"` // In USD
	CoinDeltaAmount          types.Number  `json:"coinDeltaAmount"`          // Total limit in USD
}

// InstitutionalLoanDelayLiquidationStatusResponse holds an institutional loan account's LTV and liquidation state
type InstitutionalLoanDelayLiquidationStatusResponse struct {
	LTV types.Number `json:"ltv"`
	// LiquidationStatus is 0 normal, 1 passive liquidation, 2 callback liquidation or 3 delay liquidation in progress
	LiquidationStatus uint64 `json:"liquidationStatus"`
	// DelayLiquidationStartTime and DelayLiquidationRemainingSeconds are set only while LiquidationStatus is 3
	DelayLiquidationStartTime        types.Time   `json:"delayLiqStartTime"`
	DelayLiquidationRemainingSeconds types.Number `json:"delayLiqRemainingSec"`
}
