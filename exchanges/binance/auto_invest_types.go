package binance

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// AutoInvestTargetAssetListResponse holds the Auto-Invest target assets and their simulated returns
type AutoInvestTargetAssetListResponse struct {
	TargetAssets        []currency.Code         `json:"targetAssets"`
	AutoInvestAssetList []AutoInvestTargetAsset `json:"autoInvestAssetList"`
}

// AutoInvestTargetAsset holds a target asset's simulated returns over several periods
type AutoInvestTargetAsset struct {
	TargetAsset             currency.Code            `json:"targetAsset"`
	ROIAndDimensionTypeList []AutoInvestROIDimension `json:"roiAndDimensionTypeList"`
}

// AutoInvestROIDimension holds a simulated return over a period such as 3 year or 7 day
type AutoInvestROIDimension struct {
	SimulateROI    types.Number `json:"simulateRoi"`
	DimensionValue types.Number `json:"dimensionValue"`
	DimensionUnit  string       `json:"dimensionUnit"`
}

// AutoInvestTargetAssetROI holds a target asset's simulated return accumulated up to a date
type AutoInvestTargetAssetROI struct {
	Date        types.Time   `json:"date"`
	SimulateROI types.Number `json:"simulateRoi"`
}

// AutoInvestAllAssetsResponse holds every Auto-Invest source asset and target asset
type AutoInvestAllAssetsResponse struct {
	TargetAssets []currency.Code `json:"targetAssets"`
	SourceAssets []currency.Code `json:"sourceAssets"`
}

// AutoInvestSourceAssetListRequest holds the parameters of GetSourceAssetList
type AutoInvestSourceAssetListRequest struct {
	TargetAsset          currency.Code
	IndexID              uint64
	UsageType            string // RECURRING or ONE_TIME
	FlexibleAllowedToUse bool
	SourceType           string // MAIN_SITE for Binance or TR for Binance Turkey
}

// AutoInvestSourceAssetListResponse holds the Auto-Invest source assets and the fee and tax rates
type AutoInvestSourceAssetListResponse struct {
	FeeRate      types.Number            `json:"feeRate"`
	TaxRate      types.Number            `json:"taxRate"`
	SourceAssets []AutoInvestSourceAsset `json:"sourceAssets"`
}

// AutoInvestSourceAsset holds the subscription limits of an Auto-Invest source asset
type AutoInvestSourceAsset struct {
	SourceAsset        currency.Code `json:"sourceAsset"`
	AssetMinimumAmount types.Number  `json:"assetMinAmount"`
	AssetMaximumAmount types.Number  `json:"assetMaxAmount"`
	Scale              types.Number  `json:"scale"` // decimal places
	FlexibleAmount     types.Number  `json:"flexibleAmount"`
}

// InvestmentPlanSchedule holds the subscription schedule and portfolio of an Auto-Invest plan
type InvestmentPlanSchedule struct {
	SubscriptionAmount       float64
	SubscriptionCycle        string // such as H1, H4, DAILY, WEEKLY, BI_WEEKLY or MONTHLY
	SubscriptionStartDay     uint64 // day of the month in UTC; required for MONTHLY
	SubscriptionStartWeekday string // MON to SUN in UTC; required for WEEKLY and BI_WEEKLY
	SubscriptionStartTime    uint64 // hour of the day in UTC, 0 to 23
	SourceAsset              currency.Code
	FlexibleAllowedToUse     bool
	Details                  []PortfolioDetail // percentages must sum to 100
}

// PortfolioDetail holds a target asset's percentage of an Auto-Invest portfolio
type PortfolioDetail struct {
	TargetAsset currency.Code
	Percentage  uint64
}

// InvestmentPlanCreationRequest holds the parameters of InvestmentPlanCreation
type InvestmentPlanCreationRequest struct {
	SourceType string // MAIN_SITE for Binance or TR for Binance Turkey
	RequestID  string // the source type followed by a unique string, such as TR12354859
	PlanType   string // SINGLE, PORTFOLIO or INDEX
	IndexID    uint64 // INDEX plans only
	InvestmentPlanSchedule
}

// AutoInvestPlanResponse holds a created or adjusted Auto-Invest plan
type AutoInvestPlanResponse struct {
	PlanID                uint64     `json:"planId"`
	NextExecutionDateTime types.Time `json:"nextExecutionDateTime"`
}

// InvestmentPlanAdjustmentRequest holds the parameters of InvestmentPlanAdjustment
type InvestmentPlanAdjustmentRequest struct {
	PlanID uint64
	InvestmentPlanSchedule
}

// AutoInvestPlanStatusResponse holds an Auto-Invest plan's status after a change
type AutoInvestPlanStatusResponse struct {
	PlanID                uint64     `json:"planId"`
	NextExecutionDateTime types.Time `json:"nextExecutionDateTime"`
	Status                string     `json:"status"`
}

// AutoInvestPlanListResponse holds the account's Auto-Invest plans of a plan type and their total value
type AutoInvestPlanListResponse struct {
	PlanValueInUSD types.Number     `json:"planValueInUSD"`
	PlanValueInBTC types.Number     `json:"planValueInBTC"`
	PNLInUSD       types.Number     `json:"pnlInUSD"`
	ROI            types.Number     `json:"roi"`
	Plans          []AutoInvestPlan `json:"plans"`
}

// AutoInvestPlan holds an Auto-Invest plan
type AutoInvestPlan struct {
	PlanID                   uint64        `json:"planId"`
	PlanType                 string        `json:"planType"`
	EditAllowed              types.Boolean `json:"editAllowed"`
	CreationDateTime         types.Time    `json:"creationDateTime"`
	FirstExecutionDateTime   types.Time    `json:"firstExecutionDateTime"`
	NextExecutionDateTime    types.Time    `json:"nextExecutionDateTime"`
	Status                   string        `json:"status"`
	LastUpdatedDateTime      types.Time    `json:"lastUpdatedDateTime"`
	TargetAsset              currency.Code `json:"targetAsset"`
	TotalTargetAmount        types.Number  `json:"totalTargetAmount"`
	SourceAsset              currency.Code `json:"sourceAsset"`
	TotalInvestedInUSD       types.Number  `json:"totalInvestedInUSD"`
	SubscriptionAmount       types.Number  `json:"subscriptionAmount"`
	SubscriptionCycle        string        `json:"subscriptionCycle"`
	SubscriptionStartDay     types.Number  `json:"subscriptionStartDay"`     // null unless the cycle is monthly
	SubscriptionStartWeekday string        `json:"subscriptionStartWeekday"` // null unless the cycle is weekly
	SubscriptionStartTime    types.Number  `json:"subscriptionStartTime"`    // hour of the day in UTC
	SourceWallet             string        `json:"sourceWallet"`
	FlexibleAllowedToUse     types.Boolean `json:"flexibleAllowedToUse"`
	PlanValueInUSD           types.Number  `json:"planValueInUSD"`
	PNLInUSD                 types.Number  `json:"pnlInUSD"`
	ROI                      types.Number  `json:"roi"`
}

// AutoInvestPlanHoldingResponse holds an Auto-Invest plan's holdings
type AutoInvestPlanHoldingResponse struct {
	PlanID                 uint64                        `json:"planId"`
	PlanType               string                        `json:"planType"`
	EditAllowed            types.Boolean                 `json:"editAllowed"`
	FlexibleAllowedToUse   types.Boolean                 `json:"flexibleAllowedToUse"`
	CreationDateTime       types.Time                    `json:"creationDateTime"`
	FirstExecutionDateTime types.Time                    `json:"firstExecutionDateTime"`
	NextExecutionDateTime  types.Time                    `json:"nextExecutionDateTime"`
	Status                 string                        `json:"status"`
	TargetAsset            currency.Code                 `json:"targetAsset"`
	SourceAsset            currency.Code                 `json:"sourceAsset"`
	PlanValueInUSD         types.Number                  `json:"planValueInUSD"`
	PNLInUSD               types.Number                  `json:"pnlInUSD"`
	ROI                    types.Number                  `json:"roi"`
	TotalInvestedInUSD     types.Number                  `json:"totalInvestedInUSD"`
	Details                []AutoInvestPlanHoldingDetail `json:"details"`
}

// AutoInvestPlanHoldingDetail holds an Auto-Invest plan's holding of a target asset; the available and redeemed
// amounts and the asset value are for INDEX plans only
type AutoInvestPlanHoldingDetail struct {
	TargetAsset         currency.Code `json:"targetAsset"`
	AveragePriceInUSD   types.Number  `json:"averagePriceInUSD"`
	TotalInvestedInUSD  types.Number  `json:"totalInvestedInUSD"`
	PurchasedAmount     types.Number  `json:"purchasedAmount"`
	PurchasedAmountUnit currency.Code `json:"purchasedAmountUnit"`
	PNLInUSD            types.Number  `json:"pnlInUSD"`
	ROI                 types.Number  `json:"roi"`
	Percentage          types.Number  `json:"percentage"`
	AssetStatus         string        `json:"assetStatus"`
	AvailableAmount     types.Number  `json:"availableAmount"`
	AvailableAmountUnit currency.Code `json:"availableAmountUnit"`
	RedeemedAmout       types.Number  `json:"redeemedAmout"`
	RedeemedAmoutUnit   currency.Code `json:"redeemedAmoutUnit"`
	AssetValueInUSD     types.Number  `json:"assetValueInUSD"`
}

// AutoInvestSubscriptionHistoryRequest holds the parameters of GetSubscriptionsTransactionHistory
type AutoInvestSubscriptionHistoryRequest struct {
	PlanID      uint64
	StartTime   time.Time
	EndTime     time.Time
	TargetAsset currency.Code
	PlanType    string // SINGLE, PORTFOLIO, INDEX or ALL
	Size        uint64
	Current     uint64
}

// AutoInvestSubscriptionTransaction holds an Auto-Invest subscription transaction
type AutoInvestSubscriptionTransaction struct {
	ID                  uint64        `json:"id"`
	TargetAsset         currency.Code `json:"targetAsset"`
	PlanType            string        `json:"planType"`
	PlanName            string        `json:"planName"`
	PlanID              uint64        `json:"planId"`
	TransactionDateTime types.Time    `json:"transactionDateTime"`
	TransactionStatus   string        `json:"transactionStatus"`
	FailedType          string        `json:"failedType"`
	SourceAsset         currency.Code `json:"sourceAsset"`
	SourceAssetAmount   types.Number  `json:"sourceAssetAmount"`
	TargetAssetAmount   types.Number  `json:"targetAssetAmount"`
	SourceWallet        string        `json:"sourceWallet"`
	FlexibleUsed        types.Boolean `json:"flexibleUsed"`
	TransactionFee      types.Number  `json:"transactionFee"`
	TransactionFeeUnit  currency.Code `json:"transactionFeeUnit"`
	ExecutionPrice      types.Number  `json:"executionPrice"` // source asset per target asset
	ExecutionType       string        `json:"executionType"`
	SubscriptionCycle   string        `json:"subscriptionCycle"`
}

// AutoInvestIndexDetailResponse holds an Auto-Invest index and its asset allocation
type AutoInvestIndexDetailResponse struct {
	IndexID         uint64                      `json:"indexId"`
	IndexName       string                      `json:"indexName"`
	Status          string                      `json:"status"`
	AssetAllocation []AutoInvestAssetAllocation `json:"assetAllocation"`
}

// AutoInvestAssetAllocation holds a target asset's percentage of an index
type AutoInvestAssetAllocation struct {
	TargetAsset currency.Code `json:"targetAsset"`
	Allocation  types.Number  `json:"allocation"`
}

// IndexLinkedPlanPositionResponse holds the account's position in an index-linked plan
type IndexLinkedPlanPositionResponse struct {
	IndexID              uint64                      `json:"indexId"`
	TotalInvestedInUSD   types.Number                `json:"totalInvestedInUSD"`
	CurrentInvestedInUSD types.Number                `json:"currentInvestedInUSD"`
	PNLInUSD             types.Number                `json:"pnlInUSD"`
	ROI                  types.Number                `json:"roi"`
	AssetAllocation      []AutoInvestAssetAllocation `json:"assetAllocation"`
	Details              []IndexLinkedPlanAsset      `json:"details"`
}

// IndexLinkedPlanAsset holds an index-linked plan's holding of a target asset
type IndexLinkedPlanAsset struct {
	TargetAsset          currency.Code `json:"targetAsset"`
	AveragePriceInUSD    types.Number  `json:"averagePriceInUSD"`
	TotalInvestedInUSD   types.Number  `json:"totalInvestedInUSD"`
	CurrentInvestedInUSD types.Number  `json:"currentInvestedInUSD"`
	PurchasedAmount      types.Number  `json:"purchasedAmount"`
	PNLInUSD             types.Number  `json:"pnlInUSD"`
	ROI                  types.Number  `json:"roi"`
	Percentage           types.Number  `json:"percentage"`
	AvailableAmount      types.Number  `json:"availableAmount"`
	RedeemedAmount       types.Number  `json:"redeemedAmount"`
	AssetValueInUSD      types.Number  `json:"assetValueInUSD"`
}

// OneTimeTransactionRequest holds the parameters of OneTimeTransaction; one of PlanID, IndexID and Details is required
type OneTimeTransactionRequest struct {
	SourceType           string // MAIN_SITE for Binance or TR for Binance Turkey
	RequestID            string // the source type followed by a unique string, such as TR12354859
	SubscriptionAmount   float64
	SourceAsset          currency.Code
	FlexibleAllowedToUse bool
	PlanID               uint64 // a PORTFOLIO plan
	IndexID              uint64
	Details              []PortfolioDetail // percentages must sum to 100
}

// AutoInvestOneTimeTransactionResponse holds a one-time Auto-Invest transaction and how long to wait before checking
// its status
type AutoInvestOneTimeTransactionResponse struct {
	TransactionID uint64 `json:"transactionId"`
	WaitSecond    uint64 `json:"waitSecond"`
}

// AutoInvestOneTimeTransactionStatusResponse holds the status of a one-time Auto-Invest transaction
type AutoInvestOneTimeTransactionStatusResponse struct {
	TransactionID uint64 `json:"transactionId"`
	Status        string `json:"status"`
}

// AutoInvestRedemptionResponse holds the identifier of an index-linked plan redemption
type AutoInvestRedemptionResponse struct {
	RedemptionID uint64 `json:"redemptionId"`
}

// IndexLinkedPlanRedemptionHistoryRequest holds the parameters of GetIndexLinkedPlanRedemption
type IndexLinkedPlanRedemptionHistoryRequest struct {
	RequestID string
	StartTime time.Time
	EndTime   time.Time
	Current   uint64
	Asset     currency.Code
	Size      uint64
}

// IndexLinkedPlanRedemption holds an index-linked plan redemption
type IndexLinkedPlanRedemption struct {
	IndexID            uint64        `json:"indexId"`
	IndexName          string        `json:"indexName"`
	RedemptionID       uint64        `json:"redemptionId"`
	Status             string        `json:"status"`
	Asset              currency.Code `json:"asset"`
	Amount             types.Number  `json:"amount"`
	RedemptionDateTime types.Time    `json:"redemptionDateTime"`
	TransactionFee     types.Number  `json:"transactionFee"`
	TransactionFeeUnit currency.Code `json:"transactionFeeUnit"`
}

// IndexLinkedPlanRebalance holds an index-linked plan rebalance
type IndexLinkedPlanRebalance struct {
	IndexID            uint64                                `json:"indexId"`
	IndexName          string                                `json:"indexName"`
	RebalanceID        uint64                                `json:"rebalanceId"`
	Status             string                                `json:"status"`
	RebalanceFee       types.Number                          `json:"rebalanceFee"`
	RebalanceFeeUnit   currency.Code                         `json:"rebalanceFeeUnit"`
	TransactionDetails []IndexLinkedPlanRebalanceTransaction `json:"transactionDetails"`
}

// IndexLinkedPlanRebalanceTransaction holds the trade of one asset in an index-linked plan rebalance
type IndexLinkedPlanRebalanceTransaction struct {
	Asset               currency.Code `json:"asset"`
	TransactionDateTime types.Time    `json:"transactionDateTime"`
	RebalanceDirection  string        `json:"rebalanceDirection"`
	RebalanceAmount     types.Number  `json:"rebalanceAmount"`
}
