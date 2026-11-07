package bybit

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// PWMFundTransferRequest transfers funds between two custodian sub-accounts of a fund
type PWMFundTransferRequest struct {
	TransferID string
	FromUserID uint64
	ToUserID   uint64
	Amount     float64
	Coin       currency.Code
}

// PWMFundTransferResponse is the status of a fund transfer between custodian sub-accounts
type PWMFundTransferResponse struct {
	TransferID string `json:"transferId"`
	Status     string `json:"status"`
}

// PWMFundTransferRecordsResponse is a fund transfer between custodian sub-accounts
type PWMFundTransferRecordsResponse struct {
	TransferID string        `json:"transferId"`
	Status     string        `json:"status"`
	FromUserID uint64        `json:"fromUserId"`
	ToUserID   uint64        `json:"toUserId"`
	Amount     types.Number  `json:"amount"`
	Coin       currency.Code `json:"coin"`
}

// PWMInvestmentPlansRequest filters the account's investment plans
type PWMInvestmentPlansRequest struct {
	PlanID string
	Status string
	Limit  uint64
	Cursor string
}

// PWMInvestmentPlansResponse holds a page of the account's investment plans
type PWMInvestmentPlansResponse struct {
	List []PWMInvestmentPlan `json:"list"`
	// NextPageCursor is empty on the last page
	NextPageCursor string `json:"nextPageCursor"`
}

// PWMInvestmentPlan is an investment plan
type PWMInvestmentPlan struct {
	PlanID   string `json:"planId"`
	PlanName string `json:"planName"`
	PlanType string `json:"planType"`
	Status   string `json:"status"`
	// Source is who created the plan: consultant, direct or institution
	Source string `json:"source"`
	// CurrentAssetUSD is the configured amount while the plan is PendingSubscription
	CurrentAssetUSD        types.Number                `json:"currentAssetUsd"`
	AccumulateYieldUSD     types.Number                `json:"accumulateYieldUsd"`
	InvestmentDistribution []PWMInvestmentDistribution `json:"investmentDistribution"`
	CreatedTime            types.Time                  `json:"createdTime"`
}

// PWMInvestmentDistribution is the holding of a product in an investment plan
type PWMInvestmentDistribution struct {
	Category string        `json:"category"`
	Coin     currency.Code `json:"coin"`
	// ProductID is the fund ID for fund products
	ProductID string `json:"productId"`
	// CurrentAmount is the configured amount while the plan is PendingSubscription
	CurrentAmount types.Number `json:"currentAmount"`
}

// PWMInvestmentPlanDetailResponse is an Active or Closed investment plan's assets and positions
type PWMInvestmentPlanDetailResponse struct {
	PlanID             string       `json:"planId"`
	PlanName           string       `json:"planName"`
	PlanType           string       `json:"planType"`
	Status             string       `json:"status"`
	CurrentAssetUSD    types.Number `json:"currentAssetUsd"`
	AccumulateYieldUSD types.Number `json:"accumulateYieldUsd"`
	// WeightedAverageAPR is a decimal, 0.086 for 8.6%
	WeightedAverageAPR types.Number     `json:"weightedAvgApr"`
	CurrentAssets      []PWMCoinAmount  `json:"currentAssets"`
	Positions          PWMPlanPositions `json:"positions"`
	CreatedTime        types.Time       `json:"createdTime"`
}

// PWMCoinAmount is an amount of a coin held by an investment plan
type PWMCoinAmount struct {
	Coin   currency.Code `json:"coin"`
	Amount types.Number  `json:"amount"`
}

// PWMPlanPositions holds an investment plan's positions by product category
type PWMPlanPositions struct {
	MultiCoinsEarning PWMMultiCoinsEarningPositions `json:"multiCoinsEarning"`
	FixedYield        PWMFixedYieldPositions        `json:"fixedYield"`
	EquityFunds       PWMEquityFundPositions        `json:"equityFunds"`
	OnchainEarn       PWMOnchainEarnPositions       `json:"onchainEarn"`
	// FundingAccount is the idle funds of the plan's sub-account
	FundingAccount []PWMCoinAmount `json:"fundingAccount"`
}

// PWMMultiCoinsEarningPositions holds an investment plan's flexible savings positions
type PWMMultiCoinsEarningPositions struct {
	TotalInvestmentUSD types.Number                   `json:"totalInvestmentUsd"`
	AccumulateYieldUSD types.Number                   `json:"accumulateYieldUsd"`
	WeightedAverageAPR types.Number                   `json:"weightedAvgApr"`
	Items              []PWMMultiCoinsEarningPosition `json:"items"`
}

// PWMMultiCoinsEarningPosition is a flexible savings position of an investment plan
type PWMMultiCoinsEarningPosition struct {
	Category      string        `json:"category"`
	ProductID     string        `json:"productId"`
	Coin          currency.Code `json:"coin"`
	CurrentAmount types.Number  `json:"currentAmount"`
	// AccumulateYield is in the coin
	AccumulateYield types.Number `json:"accumulateYield"`
	APR             types.Number `json:"apr"`
	PositionID      string       `json:"positionId"`
	// Status is 0 Processing, 1 Active, 2 Redeeming, 3 PendingSubscription or 4 Closed
	Status string `json:"status"`
}

// PWMFixedYieldPositions holds an investment plan's fixed yield positions
type PWMFixedYieldPositions struct {
	TotalInvestmentUSD types.Number            `json:"totalInvestmentUsd"`
	AccumulateYieldUSD types.Number            `json:"accumulateYieldUsd"`
	WeightedAverageAPR types.Number            `json:"weightedAvgApr"`
	Items              []PWMFixedYieldPosition `json:"items"`
}

// PWMFixedYieldPosition is a fixed yield position of an investment plan
type PWMFixedYieldPosition struct {
	Category        string        `json:"category"`
	ProductID       string        `json:"productId"`
	Coin            currency.Code `json:"coin"`
	CurrentAmount   types.Number  `json:"currentAmount"`
	AccumulateYield types.Number  `json:"accumulateYield"`
	APR             types.Number  `json:"apr"`
	// Duration is the lock-up in days
	Duration     uint64     `json:"duration"`
	MaturityTime types.Time `json:"maturityTime"`
	AutoReinvest bool       `json:"autoReinvest"`
	PositionID   string     `json:"positionId"`
	// Status is 0 Processing, 1 Active, 2 Redeeming, 3 PendingSubscription or 4 Closed
	Status string `json:"status"`
}

// PWMEquityFundPositions holds an investment plan's equity fund positions
type PWMEquityFundPositions struct {
	TotalInvestmentUSD types.Number            `json:"totalInvestmentUsd"`
	AccumulateYieldUSD types.Number            `json:"accumulateYieldUsd"`
	WeightedAverageAPR types.Number            `json:"weightedAvgApr"`
	Items              []PWMEquityFundPosition `json:"items"`
}

// PWMEquityFundPosition is an equity fund position of an investment plan
type PWMEquityFundPosition struct {
	Category string `json:"category"`
	// ProductID is the fund ID
	ProductID       string        `json:"productId"`
	FundName        string        `json:"fundName"`
	Coin            currency.Code `json:"coin"`
	Tags            []string      `json:"tags"`
	NAV             types.Number  `json:"nav"`
	UserShares      types.Number  `json:"userShares"`
	ShareValue      types.Number  `json:"shareValue"`
	HoldingValue    types.Number  `json:"holdingValue"`
	AccumulateYield types.Number  `json:"accumulateYield"`
	APR30Day        types.Number  `json:"apr30d"`
	// APRTotal is the annualised return since inception
	APRTotal   types.Number `json:"aprTotal"`
	SharpRatio types.Number `json:"sharpRatio"`
	// MaxDrawdown is negative
	MaxDrawdown types.Number `json:"maxDrawdown"`
	// CreatedTime is the fund's inception
	CreatedTime types.Time `json:"createdTime"`
	RunningDays uint64     `json:"runningDays"`
	PositionID  string     `json:"positionId"`
	// Status is 0 Processing, 1 Active, 2 Redeeming, 3 PendingSubscription or 4 Closed
	Status string `json:"status"`
}

// PWMOnchainEarnPositions holds an investment plan's on-chain earn positions
type PWMOnchainEarnPositions struct {
	TotalInvestmentUSD types.Number             `json:"totalInvestmentUsd"`
	AccumulateYieldUSD types.Number             `json:"accumulateYieldUsd"`
	Items              []PWMOnchainEarnPosition `json:"items"`
}

// PWMOnchainEarnPosition is an on-chain earn position of an investment plan
type PWMOnchainEarnPosition struct {
	Category    string        `json:"category"`
	ProductID   string        `json:"productId"`
	Coin        currency.Code `json:"coin"`
	StakeAmount types.Number  `json:"stakeAmount"`
	APR         types.Number  `json:"apr"`
	PositionID  string        `json:"positionId"`
	// Status is 0 Processing, 1 Active, 2 Redeeming, 3 PendingSubscription or 4 Closed
	Status string `json:"status"`
}

// PWMPendingInvestmentPlanDetailResponse is a PendingSubscription investment plan's configuration
type PWMPendingInvestmentPlanDetailResponse struct {
	PlanID   string `json:"planId"`
	PlanName string `json:"planName"`
	PlanType string `json:"planType"`
	Status   string `json:"status"`
	// Source is who created the plan: consultant, direct or institution
	Source                   string                 `json:"source"`
	TotalConfiguredAmountUSD types.Number           `json:"totalConfiguredAmountUsd"`
	Products                 PWMPendingPlanProducts `json:"products"`
	CreatedTime              types.Time             `json:"createdTime"`
}

// PWMPendingPlanProducts holds a pending investment plan's configured products by category
type PWMPendingPlanProducts struct {
	MultiCoinsEarning PWMPendingProductGroup    `json:"multiCoinsEarning"`
	FixedYield        PWMPendingFixedYieldGroup `json:"fixedYield"`
	EquityFunds       PWMPendingEquityFundGroup `json:"equityFunds"`
	OnchainEarn       PWMPendingProductGroup    `json:"onchainEarn"`
}

// PWMPendingProductGroup holds a pending investment plan's configured flexible savings or on-chain earn products
type PWMPendingProductGroup struct {
	ConfiguredAmountUSD types.Number              `json:"configuredAmountUsd"`
	Items               []PWMPendingProductConfig `json:"items"`
}

// PWMPendingProductConfig is a configured flexible savings or on-chain earn product of a pending investment plan
type PWMPendingProductConfig struct {
	Category  string        `json:"category"`
	ProductID string        `json:"productId"`
	Coin      currency.Code `json:"coin"`
	// ConfiguredAmount is in the coin
	ConfiguredAmount types.Number `json:"configuredAmount"`
	// APR is a reference value
	APR types.Number `json:"apr"`
}

// PWMPendingFixedYieldGroup holds a pending investment plan's configured fixed yield products
type PWMPendingFixedYieldGroup struct {
	ConfiguredAmountUSD types.Number                 `json:"configuredAmountUsd"`
	Items               []PWMPendingFixedYieldConfig `json:"items"`
}

// PWMPendingFixedYieldConfig is a configured fixed yield product of a pending investment plan
type PWMPendingFixedYieldConfig struct {
	Category         string        `json:"category"`
	ProductID        string        `json:"productId"`
	Coin             currency.Code `json:"coin"`
	ConfiguredAmount types.Number  `json:"configuredAmount"`
	APR              types.Number  `json:"apr"`
	// Duration is the lock-up in days
	Duration uint64 `json:"duration"`
}

// PWMPendingEquityFundGroup holds a pending investment plan's configured equity funds
type PWMPendingEquityFundGroup struct {
	ConfiguredAmountUSD types.Number                 `json:"configuredAmountUsd"`
	Items               []PWMPendingEquityFundConfig `json:"items"`
}

// PWMPendingEquityFundConfig is a configured equity fund of a pending investment plan
type PWMPendingEquityFundConfig struct {
	Category string `json:"category"`
	// ProductID is the fund ID
	ProductID        string              `json:"productId"`
	FundName         string              `json:"fundName"`
	Coin             currency.Code       `json:"coin"`
	ConfiguredAmount types.Number        `json:"configuredAmount"`
	Tags             []string            `json:"tags"`
	Introduction     PWMFundIntroduction `json:"introduction"`
}

// PWMFundIntroduction introduces an equity fund
type PWMFundIntroduction struct {
	Description            string       `json:"description"`
	HistoricalYieldRateMax types.Number `json:"historicalYieldRateMax"`
	HistoricalYieldRateMin types.Number `json:"historicalYieldRateMin"`
	SharpRatio             types.Number `json:"sharpRatio"`
	// MaxDrawback is the maximum drawdown, negative
	MaxDrawback  types.Number `json:"maxDrawback"`
	LockupPeriod string       `json:"lockupPeriod"`
}

// PWMClaimRequest claims the withdrawable funds of an Active investment plan
type PWMClaimRequest struct {
	PlanID string
	// ToAccountType defaults to FUND
	ToAccountType string
	OrderLinkID   string
}

// PWMClaimResponse is the outcome of a withdrawable funds claim
type PWMClaimResponse struct {
	PlanID string `json:"planId"`
	// ToAccountType is the numeric code of the destination account type, although the request takes its name
	ToAccountType uint64     `json:"toAccountType"`
	Status        string     `json:"status"`
	CreatedTime   types.Time `json:"createdTime"`
}

// PWMAssetTrendRequest selects an investment plan's asset values. The window defaults to the last 7 days
type PWMAssetTrendRequest struct {
	PlanID    string
	StartTime time.Time
	EndTime   time.Time
}

// PWMAssetTrendResponse holds an investment plan's daily asset value, oldest first
type PWMAssetTrendResponse struct {
	PlanID     string               `json:"planId"`
	DataPoints []PWMAssetTrendPoint `json:"dataPoints"`
}

// PWMAssetTrendPoint is an investment plan's asset value at a day's settlement
type PWMAssetTrendPoint struct {
	// Date is YYYY-MM-DD
	Date          string       `json:"date"`
	AssetValueUSD types.Number `json:"assetValueUsd"`
}

// PWMFundHistoricalNAVRequest selects a fund's NAV history. The window defaults to the last 7 days and may span at most
// 180 days
type PWMFundHistoricalNAVRequest struct {
	FundID    string
	StartTime time.Time
	EndTime   time.Time
}

// PWMFundHistoricalNAVResponse holds a fund's daily NAV, oldest first
type PWMFundHistoricalNAVResponse struct {
	FundID   string        `json:"fundId"`
	FundName string        `json:"fundName"`
	Coin     currency.Code `json:"coin"`
	// CurrentNAV is the current share value divided by the initial share value
	CurrentNAV types.Number      `json:"currentNav"`
	DataPoints []PWMFundNAVPoint `json:"dataPoints"`
}

// PWMFundNAVPoint is a fund's NAV at a day's settlement
type PWMFundNAVPoint struct {
	// Date is YYYY-MM-DD
	Date string       `json:"date"`
	NAV  types.Number `json:"nav"`
}

// PWMSubscribeRequest subscribes to a PendingSubscription investment plan
type PWMSubscribeRequest struct {
	PlanID string
	// AccountType is the source account, FUND by default
	AccountType string
	OrderLinkID string
}

// PWMSubscribeResponse is an investment plan after subscription; it becomes Active on the first successful subscription
type PWMSubscribeResponse struct {
	PlanID      string `json:"planId"`
	Status      string `json:"status"`
	OrderLinkID string `json:"orderLinkId"`
}

// PWMInvestMoreRequest adds to a product of an Active investment plan
type PWMInvestMoreRequest struct {
	PlanID string
	// AccountType is the source account, FUND by default
	AccountType string
	Category    string
	ProductID   string
	// Amount is in the product's coin
	Amount      float64
	OrderLinkID string
}

// PWMInvestMoreResponse is the outcome of an additional investment in an investment plan
type PWMInvestMoreResponse struct {
	PlanID      string        `json:"planId"`
	Category    string        `json:"category"`
	ProductID   string        `json:"productId"`
	Coin        currency.Code `json:"coin"`
	Amount      types.Number  `json:"amount"`
	Status      string        `json:"status"`
	OrderLinkID string        `json:"orderLinkId"`
	OrderID     string        `json:"orderId"`
}

// PWMRedeemRequest redeems from a product of an investment plan
type PWMRedeemRequest struct {
	PlanID   string
	Category string
	// ProductID is the fund ID for fund products
	ProductID string
	// Shares is required for fund products
	Shares float64
	// Amount is required for other products
	Amount      float64
	OrderLinkID string
	// PositionID is required for FundPool and on-chain earn products
	PositionID string
}

// PWMRedeemResponse is the outcome of an investment plan redemption; equity fund redemptions are Pending until
// approved, others succeed at once
type PWMRedeemResponse struct {
	OrderID   string `json:"orderId"`
	PlanID    string `json:"planId"`
	Category  string `json:"category"`
	ProductID string `json:"productId"`
	// Shares is set for fund products and Amount for other products
	Shares types.Number `json:"shares"`
	Amount types.Number `json:"amount"`
	// EstimatedAmount is at the current share value; the settled amount may differ
	EstimatedAmount types.Number  `json:"estimatedAmount"`
	Coin            currency.Code `json:"coin"`
	Status          string        `json:"status"`
	OrderLinkID     string        `json:"orderLinkId"`
}

// PWMInvestmentPlanOrdersRequest filters investment plan orders
type PWMInvestmentPlanOrdersRequest struct {
	PlanID    string
	Category  string
	Type      string
	Status    string
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
	Cursor    string
	// OrderLinkID is at most 36 characters
	OrderLinkID string
}

// PWMInvestmentPlanOrdersResponse holds a page of investment plan orders, newest first
type PWMInvestmentPlanOrdersResponse struct {
	List []PWMInvestmentPlanOrder `json:"list"`
	// NextPageCursor is empty on the last page
	NextPageCursor string `json:"nextPageCursor"`
}

// PWMInvestmentPlanOrder is a subscription, redemption or auto-reinvest order of an investment plan
type PWMInvestmentPlanOrder struct {
	OrderID     string        `json:"orderId"`
	PlanID      string        `json:"planId"`
	Type        string        `json:"type"`
	AccountType string        `json:"accountType"`
	Coin        currency.Code `json:"coin"`
	Amount      types.Number  `json:"amount"`
	Category    string        `json:"category"`
	ProductID   string        `json:"productId"`
	Status      string        `json:"status"`
	OrderTime   types.Time    `json:"orderTime"`
}

// PWMSubscribableProductInfoResponse holds the products a customised investment plan can hold, by category
type PWMSubscribableProductInfoResponse struct {
	Products []PWMProductGroup `json:"products"`
}

// PWMProductGroup holds the subscribable products of a category
type PWMProductGroup struct {
	Type  string           `json:"type"`
	Cards []PWMProductCard `json:"cards"`
}

// PWMProductCard is a product a customised investment plan can hold. Funds set FundName, the APR range, Introduction,
// MaxDrawdown and SharpRatio; other products set ProductID and APR
type PWMProductCard struct {
	Category  string        `json:"category"`
	ProductID string        `json:"productId"`
	FundName  string        `json:"fundName"`
	Coin      currency.Code `json:"coin"`
	// APR, APRRangeLow, APRRangeHigh and EstimatedAPR are decimals, 0.05 for 5%
	APR          types.Number `json:"apr"`
	APRRangeLow  types.Number `json:"aprRangeLow"`
	APRRangeHigh types.Number `json:"aprRangeHigh"`
	Tags         []string     `json:"tags"`
	Introduction string       `json:"introduction"`
	// AUM is the assets under management in the coin
	AUM                 types.Number `json:"aum"`
	MinInvestmentAmount types.Number `json:"minInvestmentAmount"`
	MaxInvestmentAmount types.Number `json:"maxInvestmentAmount"`
	// Duration is the lock-up in days; 0 is flexible
	Duration     uint64       `json:"duration"`
	MaxDrawdown  types.Number `json:"maxDrawdown"`
	SharpRatio   types.Number `json:"sharpRatio"`
	EstimatedAPR types.Number `json:"estAPR"`
}

// PWMCustomisedPlanRequest creates and subscribes to an investment plan of subscribable products
type PWMCustomisedPlanRequest struct {
	// AccountType is the source account, FUND by default
	AccountType string
	Products    []PWMCustomisedPlanProduct
	// OrderLinkID is undocumented as a request parameter, but the response echoes it and Bybit's Python SDK requires
	// it, so it is sent when set
	OrderLinkID string
}

// PWMCustomisedPlanProduct is a product of a customised investment plan; Category, ProductID and FundName are passed
// through from GetPWMSubscribableProductInfo
type PWMCustomisedPlanProduct struct {
	Category  string
	ProductID string
	FundName  string
	// Amount is in the product's coin
	Amount float64
}

// PWMCustomisedPlanResponse is a created customised investment plan, Active once subscribed
type PWMCustomisedPlanResponse struct {
	PlanID string `json:"planId"`
	// PlanName is generated as PWM-{planId}
	PlanName    string `json:"planName"`
	Status      string `json:"status"`
	OrderLinkID string `json:"orderLinkId"`
}

// PWMFundsRequest filters the institution's funds
type PWMFundsRequest struct {
	Coin   currency.Code
	FundID string
	Status string
	Limit  uint64
	// Cursor is a fund ID
	Cursor string
}

// PWMFundsResponse holds a page of the institution's funds
type PWMFundsResponse struct {
	List []PWMFund `json:"list"`
	// NextPageCursor is empty on the last page
	NextPageCursor string `json:"nextPageCursor"`
}

// PWMFund is a fund managed by the institution
type PWMFund struct {
	FundID   string        `json:"fundId"`
	FundName string        `json:"fundName"`
	Coin     currency.Code `json:"coin"`
	Status   string        `json:"status"`
	// TotalEquity is in the coin
	TotalEquity types.Number `json:"totalEquity"`
	TotalShares types.Number `json:"totalShares"`
	// CurrentNAV is the share value divided by the initial share value
	CurrentNAV types.Number `json:"currentNav"`
	// CurrentAPR is the 30-day APR
	CurrentAPR types.Number `json:"currentAPR"`
	// AccountUID is the fund's main sub-account
	AccountUID     string   `json:"accountUid"`
	SubAccountList []string `json:"subAccountList"`
	// ProfitShareRate and ManagementFeeRate are percentages, 20 for 20%
	ProfitShareRate   types.Number `json:"profitShareRate"`
	ManagementFeeRate types.Number `json:"managementFeeRate"`
	// UncollectedProfit, CollectedProfit and TotalLoan are in the coin
	UncollectedProfit types.Number `json:"uncollectedProfit"`
	CollectedProfit   types.Number `json:"collectedProfit"`
	TotalLoan         types.Number `json:"totalLoan"`
	CreatedTime       types.Time   `json:"createdTime"`
}

// PWMSettleFundProfitResponse is the outcome of a fund profit settlement
type PWMSettleFundProfitResponse struct {
	FundID string `json:"fundId"`
	Status string `json:"status"`
	// TotalProfitShared and InstitutionIncome are in the coin
	TotalProfitShared types.Number  `json:"totalProfitShared"`
	InstitutionIncome types.Number  `json:"instIncome"`
	Coin              currency.Code `json:"coin"`
	CreatedTime       types.Time    `json:"createdTime"`
}

// PWMCreateFundRequest creates a fund pending subscription
type PWMCreateFundRequest struct {
	// FundName is 1 to 50 characters
	FundName string
	// Coin is the fund's base coin: BTC, ETH, USDT, USDC, SOL, MNT or XRP
	Coin currency.Code
	// ProfitShareRate is the high-watermark profit share and ManagementFeeRate the annual management fee, as
	// percentages from 0 to 100
	ProfitShareRate   float64
	ManagementFeeRate float64
	// FundIntroduction is an introduction ID from the institution's allowed list
	FundIntroduction string
	RequestLinkID    string
}

// PWMCreateFundResponse is a created fund
type PWMCreateFundResponse struct {
	FundID            string        `json:"fundId"`
	FundName          string        `json:"fundName"`
	Coin              currency.Code `json:"coin"`
	Status            string        `json:"status"`
	ProfitShareRate   types.Number  `json:"profitShareRate"`
	ManagementFeeRate types.Number  `json:"managementFeeRate"`
	// AccountUID is the fund's main sub-account, created by the system
	AccountUID  string     `json:"accountUid"`
	CreatedTime types.Time `json:"createdTime"`
}

// PWMCreateInvestmentPlanRequest creates an investment plan of the institution's funds for a user
type PWMCreateInvestmentPlanRequest struct {
	AccountUID string
	// PlanName is at most 50 characters
	PlanName               string
	PlanType               string
	InvestmentDistribution []PWMPlanFundAllocation
	RequestLinkID          string
}

// PWMPlanFundAllocation is a fund configured into a new investment plan
type PWMPlanFundAllocation struct {
	// ProductID is the fund ID
	ProductID string
	// Amount is in the fund's coin
	Amount float64
}

// PWMCreateInvestmentPlanResponse is a created investment plan, PendingSubscription until the user subscribes
type PWMCreateInvestmentPlanResponse struct {
	PlanID      string     `json:"planId"`
	PlanType    string     `json:"planType"`
	AccountUID  string     `json:"accountUid"`
	Status      string     `json:"status"`
	CreatedTime types.Time `json:"createdTime"`
}

// PWMAssetManagerInvestmentPlansRequest filters the investment plans the institution created
type PWMAssetManagerInvestmentPlansRequest struct {
	PlanID          string
	Status          string
	SubscriptionUID string
	Limit           uint64
	// Cursor is an investment plan ID
	Cursor string
}

// PWMAssetManagerInvestmentPlansResponse holds a page of the investment plans the institution created
type PWMAssetManagerInvestmentPlansResponse struct {
	List []PWMAssetManagerInvestmentPlan `json:"list"`
	// NextPageCursor is empty on the last page
	NextPageCursor string `json:"nextPageCursor"`
}

// PWMAssetManagerInvestmentPlan is an investment plan the institution created
type PWMAssetManagerInvestmentPlan struct {
	PlanID   string `json:"planId"`
	PlanName string `json:"planName"`
	PlanType string `json:"planType"`
	// SubscriptionUID is the user who subscribed to the plan
	SubscriptionUID string `json:"subscriptionUid"`
	Status          string `json:"status"`
	// Source is who created the plan: consultant, direct or institution
	Source string `json:"source"`
	// CurrentAssetUSD is the configured amount while the plan is PendingSubscription
	CurrentAssetUSD        types.Number                `json:"currentAssetUsd"`
	AccumulateYieldUSD     types.Number                `json:"accumulateYieldUsd"`
	InvestmentDistribution []PWMInvestmentDistribution `json:"investmentDistribution"`
	CreatedTime            types.Time                  `json:"createdTime"`
}

// PWMManageInvestmentPlanRequest closes or deletes an investment plan, or adds funds and updates their amounts
type PWMManageInvestmentPlanRequest struct {
	PlanID string
	// UpdateStatus is Closed or Deleted; empty leaves the status unchanged
	UpdateStatus string
	// UpdateFunds updates the amount of a fund already in the plan, or adds it; at most 10 funds without duplicates
	UpdateFunds   []PWMFundAllocation
	RequestLinkID string
}

// PWMFundAllocation is a fund's configured amount in an investment plan update
type PWMFundAllocation struct {
	FundID string
	// Amount is in the fund's coin
	Amount float64
}

// PWMManageInvestmentPlanResponse is an investment plan after an update
type PWMManageInvestmentPlanResponse struct {
	PlanID      string           `json:"planId"`
	Status      string           `json:"status"`
	UpdatedTime types.Time       `json:"updatedTime"`
	UpdateFunds []PWMUpdatedFund `json:"updateFunds"`
}

// PWMUpdatedFund is a fund configured into an investment plan after an update
type PWMUpdatedFund struct {
	FundID string `json:"fundId"`
	// Amount is in the fund's coin
	Amount types.Number `json:"amount"`
}

// PWMFundOrdersRequest filters the orders of the institution's funds. Without a window the last 7 days are returned;
// with one, at most the 7 days before the end time
type PWMFundOrdersRequest struct {
	FundID    string
	OrderType string
	Status    string
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
	// Cursor is an order ID
	Cursor string
}

// PWMFundOrdersResponse holds a page of fund orders
type PWMFundOrdersResponse struct {
	List []PWMFundOrder `json:"list"`
	// NextPageCursor is empty on the last page
	NextPageCursor string `json:"nextPageCursor"`
}

// PWMFundOrder is a subscription or redemption order of a fund
type PWMFundOrder struct {
	OrderID  string `json:"orderId"`
	FundID   string `json:"fundId"`
	FundName string `json:"fundName"`
	// AccountUID is the fund's main sub-account
	AccountUID string        `json:"accountUid"`
	OrderType  string        `json:"orderType"`
	Coin       currency.Code `json:"coin"`
	// Amount is only set for subscriptions and Shares for redemptions
	Amount      types.Number `json:"amount"`
	Shares      types.Number `json:"shares"`
	Status      string       `json:"status"`
	CreatedTime types.Time   `json:"createdTime"`
}

// PWMManageOrderRequest approves or rejects a fund order pending review
type PWMManageOrderRequest struct {
	OrderID string
	// Action is approve or reject
	Action        string
	RequestLinkID string
}

// PWMManageOrderResponse is a fund order after approval or rejection
type PWMManageOrderResponse struct {
	OrderID string `json:"orderId"`
	FundID  string `json:"fundId"`
	// AccountUID is the fund's main account
	AccountUID  string        `json:"accountUid"`
	OrderStatus string        `json:"orderStatus"`
	OrderType   string        `json:"orderType"`
	Action      string        `json:"action"`
	Coin        currency.Code `json:"coin"`
	// Amount is only set for subscriptions and Shares for redemptions
	Amount      types.Number `json:"amount"`
	Shares      types.Number `json:"shares"`
	UpdatedTime types.Time   `json:"updatedTime"`
}

// PWMCreateFundSubAccountResponse is the creation status of a fund sub-account
type PWMCreateFundSubAccountResponse struct {
	FundID string `json:"fundId"`
	// SubAccountUID is "0" until the sub-account is created
	SubAccountUID string     `json:"subAccountUid"`
	CreatedTime   types.Time `json:"createdTime"`
	Status        string     `json:"status"`
}
