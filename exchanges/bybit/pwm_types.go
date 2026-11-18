package bybit

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// PrivateWealthManagementFundTransferRequest transfers funds between two custodian sub-accounts of a fund
type PrivateWealthManagementFundTransferRequest struct {
	TransferID string
	FromUserID uint64
	ToUserID   uint64
	Amount     float64
	Coin       currency.Code
}

// PrivateWealthManagementFundTransferResponse is the status of a fund transfer between custodian sub-accounts
type PrivateWealthManagementFundTransferResponse struct {
	TransferID string `json:"transferId"`
	Status     string `json:"status"`
}

// PrivateWealthManagementFundTransferRecordsResponse is a fund transfer between custodian sub-accounts
type PrivateWealthManagementFundTransferRecordsResponse struct {
	TransferID string        `json:"transferId"`
	Status     string        `json:"status"`
	FromUserID uint64        `json:"fromUserId"`
	ToUserID   uint64        `json:"toUserId"`
	Amount     types.Number  `json:"amount"`
	Coin       currency.Code `json:"coin"`
}

// PrivateWealthManagementInvestmentPlansRequest filters the account's investment plans
type PrivateWealthManagementInvestmentPlansRequest struct {
	PlanID string
	Status string
	Limit  uint64
	Cursor string
}

// PrivateWealthManagementInvestmentPlansResponse holds a page of the account's investment plans
type PrivateWealthManagementInvestmentPlansResponse struct {
	List []PrivateWealthManagementInvestmentPlan `json:"list"`
	// NextPageCursor is empty on the last page
	NextPageCursor string `json:"nextPageCursor"`
}

// PrivateWealthManagementInvestmentPlan is an investment plan
type PrivateWealthManagementInvestmentPlan struct {
	PlanID   string `json:"planId"`
	PlanName string `json:"planName"`
	PlanType string `json:"planType"`
	Status   string `json:"status"`
	// Source is who created the plan: consultant, direct or institution
	Source string `json:"source"`
	// CurrentAssetUSD is the configured amount while the plan is PendingSubscription
	CurrentAssetUSD        types.Number                                    `json:"currentAssetUsd"`
	AccumulateYieldUSD     types.Number                                    `json:"accumulateYieldUsd"`
	InvestmentDistribution []PrivateWealthManagementInvestmentDistribution `json:"investmentDistribution"`
	CreatedTime            types.Time                                      `json:"createdTime"`
}

// PrivateWealthManagementInvestmentDistribution is the holding of a product in an investment plan
type PrivateWealthManagementInvestmentDistribution struct {
	Category string        `json:"category"`
	Coin     currency.Code `json:"coin"`
	// ProductID is the fund ID for fund products
	ProductID string `json:"productId"`
	// CurrentAmount is the configured amount while the plan is PendingSubscription
	CurrentAmount types.Number `json:"currentAmount"`
}

// PrivateWealthManagementInvestmentPlanDetailResponse is an Active or Closed investment plan's assets and positions
type PrivateWealthManagementInvestmentPlanDetailResponse struct {
	PlanID             string       `json:"planId"`
	PlanName           string       `json:"planName"`
	PlanType           string       `json:"planType"`
	Status             string       `json:"status"`
	CurrentAssetUSD    types.Number `json:"currentAssetUsd"`
	AccumulateYieldUSD types.Number `json:"accumulateYieldUsd"`
	// WeightedAverageAPR is a decimal, 0.086 for 8.6%
	WeightedAverageAPR types.Number                         `json:"weightedAvgApr"`
	CurrentAssets      []PrivateWealthManagementCoinAmount  `json:"currentAssets"`
	Positions          PrivateWealthManagementPlanPositions `json:"positions"`
	CreatedTime        types.Time                           `json:"createdTime"`
}

// PrivateWealthManagementCoinAmount is an amount of a coin held by an investment plan
type PrivateWealthManagementCoinAmount struct {
	Coin   currency.Code `json:"coin"`
	Amount types.Number  `json:"amount"`
}

// PrivateWealthManagementPlanPositions holds an investment plan's positions by product category
type PrivateWealthManagementPlanPositions struct {
	MultiCoinsEarning PrivateWealthManagementMultiCoinsEarningPositions `json:"multiCoinsEarning"`
	FixedYield        PrivateWealthManagementFixedYieldPositions        `json:"fixedYield"`
	EquityFunds       PrivateWealthManagementEquityFundPositions        `json:"equityFunds"`
	OnchainEarn       PrivateWealthManagementOnchainEarnPositions       `json:"onchainEarn"`
	// FundingAccount is the idle funds of the plan's sub-account
	FundingAccount []PrivateWealthManagementCoinAmount `json:"fundingAccount"`
}

// PrivateWealthManagementMultiCoinsEarningPositions holds an investment plan's flexible savings positions
type PrivateWealthManagementMultiCoinsEarningPositions struct {
	TotalInvestmentUSD types.Number                                       `json:"totalInvestmentUsd"`
	AccumulateYieldUSD types.Number                                       `json:"accumulateYieldUsd"`
	WeightedAverageAPR types.Number                                       `json:"weightedAvgApr"`
	Items              []PrivateWealthManagementMultiCoinsEarningPosition `json:"items"`
}

// PrivateWealthManagementMultiCoinsEarningPosition is a flexible savings position of an investment plan
type PrivateWealthManagementMultiCoinsEarningPosition struct {
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

// PrivateWealthManagementFixedYieldPositions holds an investment plan's fixed yield positions
type PrivateWealthManagementFixedYieldPositions struct {
	TotalInvestmentUSD types.Number                                `json:"totalInvestmentUsd"`
	AccumulateYieldUSD types.Number                                `json:"accumulateYieldUsd"`
	WeightedAverageAPR types.Number                                `json:"weightedAvgApr"`
	Items              []PrivateWealthManagementFixedYieldPosition `json:"items"`
}

// PrivateWealthManagementFixedYieldPosition is a fixed yield position of an investment plan
type PrivateWealthManagementFixedYieldPosition struct {
	Category        string        `json:"category"`
	ProductID       string        `json:"productId"`
	Coin            currency.Code `json:"coin"`
	CurrentAmount   types.Number  `json:"currentAmount"`
	AccumulateYield types.Number  `json:"accumulateYield"`
	APR             types.Number  `json:"apr"`
	// DurationInDays is the lock-up
	DurationInDays uint64     `json:"duration"`
	MaturityTime   types.Time `json:"maturityTime"`
	AutoReinvest   bool       `json:"autoReinvest"`
	PositionID     string     `json:"positionId"`
	// Status is 0 Processing, 1 Active, 2 Redeeming, 3 PendingSubscription or 4 Closed
	Status string `json:"status"`
}

// PrivateWealthManagementEquityFundPositions holds an investment plan's equity fund positions
type PrivateWealthManagementEquityFundPositions struct {
	TotalInvestmentUSD types.Number                                `json:"totalInvestmentUsd"`
	AccumulateYieldUSD types.Number                                `json:"accumulateYieldUsd"`
	WeightedAverageAPR types.Number                                `json:"weightedAvgApr"`
	Items              []PrivateWealthManagementEquityFundPosition `json:"items"`
}

// PrivateWealthManagementEquityFundPosition is an equity fund position of an investment plan
type PrivateWealthManagementEquityFundPosition struct {
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

// PrivateWealthManagementOnchainEarnPositions holds an investment plan's on-chain earn positions
type PrivateWealthManagementOnchainEarnPositions struct {
	TotalInvestmentUSD types.Number                                 `json:"totalInvestmentUsd"`
	AccumulateYieldUSD types.Number                                 `json:"accumulateYieldUsd"`
	Items              []PrivateWealthManagementOnchainEarnPosition `json:"items"`
}

// PrivateWealthManagementOnchainEarnPosition is an on-chain earn position of an investment plan
type PrivateWealthManagementOnchainEarnPosition struct {
	Category    string        `json:"category"`
	ProductID   string        `json:"productId"`
	Coin        currency.Code `json:"coin"`
	StakeAmount types.Number  `json:"stakeAmount"`
	APR         types.Number  `json:"apr"`
	PositionID  string        `json:"positionId"`
	// Status is 0 Processing, 1 Active, 2 Redeeming, 3 PendingSubscription or 4 Closed
	Status string `json:"status"`
}

// PrivateWealthManagementPendingInvestmentPlanDetailResponse is a PendingSubscription investment plan's configuration
type PrivateWealthManagementPendingInvestmentPlanDetailResponse struct {
	PlanID   string `json:"planId"`
	PlanName string `json:"planName"`
	PlanType string `json:"planType"`
	Status   string `json:"status"`
	// Source is who created the plan: consultant, direct or institution
	Source                   string                                     `json:"source"`
	TotalConfiguredAmountUSD types.Number                               `json:"totalConfiguredAmountUsd"`
	Products                 PrivateWealthManagementPendingPlanProducts `json:"products"`
	CreatedTime              types.Time                                 `json:"createdTime"`
}

// PrivateWealthManagementPendingPlanProducts holds a pending investment plan's configured products by category
type PrivateWealthManagementPendingPlanProducts struct {
	MultiCoinsEarning PrivateWealthManagementPendingProductGroup    `json:"multiCoinsEarning"`
	FixedYield        PrivateWealthManagementPendingFixedYieldGroup `json:"fixedYield"`
	EquityFunds       PrivateWealthManagementPendingEquityFundGroup `json:"equityFunds"`
	OnchainEarn       PrivateWealthManagementPendingProductGroup    `json:"onchainEarn"`
}

// PrivateWealthManagementPendingProductGroup holds a pending investment plan's configured flexible savings or on-chain
// earn products
type PrivateWealthManagementPendingProductGroup struct {
	ConfiguredAmountUSD types.Number                                  `json:"configuredAmountUsd"`
	Items               []PrivateWealthManagementPendingProductConfig `json:"items"`
}

// PrivateWealthManagementPendingProductConfig is a configured flexible savings or on-chain earn product of a pending
// investment plan
type PrivateWealthManagementPendingProductConfig struct {
	Category  string        `json:"category"`
	ProductID string        `json:"productId"`
	Coin      currency.Code `json:"coin"`
	// ConfiguredAmount is in the coin
	ConfiguredAmount types.Number `json:"configuredAmount"`
	// APR is a reference value
	APR types.Number `json:"apr"`
}

// PrivateWealthManagementPendingFixedYieldGroup holds a pending investment plan's configured fixed yield products
type PrivateWealthManagementPendingFixedYieldGroup struct {
	ConfiguredAmountUSD types.Number                                     `json:"configuredAmountUsd"`
	Items               []PrivateWealthManagementPendingFixedYieldConfig `json:"items"`
}

// PrivateWealthManagementPendingFixedYieldConfig is a configured fixed yield product of a pending investment plan
type PrivateWealthManagementPendingFixedYieldConfig struct {
	Category         string        `json:"category"`
	ProductID        string        `json:"productId"`
	Coin             currency.Code `json:"coin"`
	ConfiguredAmount types.Number  `json:"configuredAmount"`
	APR              types.Number  `json:"apr"`
	// DurationInDays is the lock-up
	DurationInDays uint64 `json:"duration"`
}

// PrivateWealthManagementPendingEquityFundGroup holds a pending investment plan's configured equity funds
type PrivateWealthManagementPendingEquityFundGroup struct {
	ConfiguredAmountUSD types.Number                                     `json:"configuredAmountUsd"`
	Items               []PrivateWealthManagementPendingEquityFundConfig `json:"items"`
}

// PrivateWealthManagementPendingEquityFundConfig is a configured equity fund of a pending investment plan
type PrivateWealthManagementPendingEquityFundConfig struct {
	Category string `json:"category"`
	// ProductID is the fund ID
	ProductID        string                                  `json:"productId"`
	FundName         string                                  `json:"fundName"`
	Coin             currency.Code                           `json:"coin"`
	ConfiguredAmount types.Number                            `json:"configuredAmount"`
	Tags             []string                                `json:"tags"`
	Introduction     PrivateWealthManagementFundIntroduction `json:"introduction"`
}

// PrivateWealthManagementFundIntroduction introduces an equity fund
type PrivateWealthManagementFundIntroduction struct {
	Description            string       `json:"description"`
	HistoricalYieldRateMax types.Number `json:"historicalYieldRateMax"`
	HistoricalYieldRateMin types.Number `json:"historicalYieldRateMin"`
	SharpRatio             types.Number `json:"sharpRatio"`
	// MaxDrawback is the maximum drawdown, negative
	MaxDrawback  types.Number `json:"maxDrawback"`
	LockupPeriod string       `json:"lockupPeriod"`
}

// PrivateWealthManagementClaimRequest claims the withdrawable funds of an Active investment plan
type PrivateWealthManagementClaimRequest struct {
	PlanID string
	// ToAccountType defaults to FUND
	ToAccountType string
	OrderLinkID   string
}

// PrivateWealthManagementClaimResponse is the outcome of a withdrawable funds claim
type PrivateWealthManagementClaimResponse struct {
	PlanID string `json:"planId"`
	// ToAccountType is the numeric code of the destination account type, although the request takes its name
	ToAccountType uint64     `json:"toAccountType"`
	Status        string     `json:"status"`
	CreatedTime   types.Time `json:"createdTime"`
}

// PrivateWealthManagementAssetTrendRequest selects an investment plan's asset values. The window defaults to the last 7
// days
type PrivateWealthManagementAssetTrendRequest struct {
	PlanID    string
	StartTime time.Time
	EndTime   time.Time
}

// PrivateWealthManagementAssetTrendResponse holds an investment plan's daily asset value, oldest first
type PrivateWealthManagementAssetTrendResponse struct {
	PlanID     string                                   `json:"planId"`
	DataPoints []PrivateWealthManagementAssetTrendPoint `json:"dataPoints"`
}

// PrivateWealthManagementAssetTrendPoint is an investment plan's asset value at a day's settlement
type PrivateWealthManagementAssetTrendPoint struct {
	// Date is YYYY-MM-DD
	Date          string       `json:"date"`
	AssetValueUSD types.Number `json:"assetValueUsd"`
}

// PrivateWealthManagementFundHistoricalNAVRequest selects a fund's NAV history. The window defaults to the last 7 days
// and may span at most 180 days
type PrivateWealthManagementFundHistoricalNAVRequest struct {
	FundID    string
	StartTime time.Time
	EndTime   time.Time
}

// PrivateWealthManagementFundHistoricalNAVResponse holds a fund's daily NAV, oldest first
type PrivateWealthManagementFundHistoricalNAVResponse struct {
	FundID   string        `json:"fundId"`
	FundName string        `json:"fundName"`
	Coin     currency.Code `json:"coin"`
	// CurrentNAV is the current share value divided by the initial share value
	CurrentNAV types.Number                          `json:"currentNav"`
	DataPoints []PrivateWealthManagementFundNAVPoint `json:"dataPoints"`
}

// PrivateWealthManagementFundNAVPoint is a fund's NAV at a day's settlement
type PrivateWealthManagementFundNAVPoint struct {
	// Date is YYYY-MM-DD
	Date string       `json:"date"`
	NAV  types.Number `json:"nav"`
}

// PrivateWealthManagementSubscribeRequest subscribes to a PendingSubscription investment plan
type PrivateWealthManagementSubscribeRequest struct {
	PlanID string
	// AccountType is the source account, FUND by default
	AccountType string
	OrderLinkID string
}

// PrivateWealthManagementSubscribeResponse is an investment plan after subscription; it becomes Active on the first
// successful subscription
type PrivateWealthManagementSubscribeResponse struct {
	PlanID      string `json:"planId"`
	Status      string `json:"status"`
	OrderLinkID string `json:"orderLinkId"`
}

// PrivateWealthManagementInvestMoreRequest adds to a product of an Active investment plan
type PrivateWealthManagementInvestMoreRequest struct {
	PlanID string
	// AccountType is the source account, FUND by default
	AccountType string
	Category    string
	ProductID   string
	// Amount is in the product's coin
	Amount      float64
	OrderLinkID string
}

// PrivateWealthManagementInvestMoreResponse is the outcome of an additional investment in an investment plan
type PrivateWealthManagementInvestMoreResponse struct {
	PlanID      string        `json:"planId"`
	Category    string        `json:"category"`
	ProductID   string        `json:"productId"`
	Coin        currency.Code `json:"coin"`
	Amount      types.Number  `json:"amount"`
	Status      string        `json:"status"`
	OrderLinkID string        `json:"orderLinkId"`
	OrderID     string        `json:"orderId"`
}

// PrivateWealthManagementRedeemRequest redeems from a product of an investment plan
type PrivateWealthManagementRedeemRequest struct {
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

// PrivateWealthManagementRedeemResponse is the outcome of an investment plan redemption; equity fund redemptions are
// Pending until approved, others succeed at once
type PrivateWealthManagementRedeemResponse struct {
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

// PrivateWealthManagementInvestmentPlanOrdersRequest filters investment plan orders
type PrivateWealthManagementInvestmentPlanOrdersRequest struct {
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

// PrivateWealthManagementInvestmentPlanOrdersResponse holds a page of investment plan orders, newest first
type PrivateWealthManagementInvestmentPlanOrdersResponse struct {
	List []PrivateWealthManagementInvestmentPlanOrder `json:"list"`
	// NextPageCursor is empty on the last page
	NextPageCursor string `json:"nextPageCursor"`
}

// PrivateWealthManagementInvestmentPlanOrder is a subscription, redemption or auto-reinvest order of an investment plan
type PrivateWealthManagementInvestmentPlanOrder struct {
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

// PrivateWealthManagementSubscribableProductInfoResponse holds the products a customised investment plan can hold, by
// category
type PrivateWealthManagementSubscribableProductInfoResponse struct {
	Products []PrivateWealthManagementProductGroup `json:"products"`
}

// PrivateWealthManagementProductGroup holds the subscribable products of a category
type PrivateWealthManagementProductGroup struct {
	Type  string                               `json:"type"`
	Cards []PrivateWealthManagementProductCard `json:"cards"`
}

// PrivateWealthManagementProductCard is a product a customised investment plan can hold. Funds set FundName, the APR
// range, Introduction, MaxDrawdown and SharpRatio; other products set ProductID and APR
type PrivateWealthManagementProductCard struct {
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
	// DurationInDays is the lock-up; 0 is flexible
	DurationInDays uint64       `json:"duration"`
	MaxDrawdown    types.Number `json:"maxDrawdown"`
	SharpRatio     types.Number `json:"sharpRatio"`
	EstimatedAPR   types.Number `json:"estAPR"`
}

// PrivateWealthManagementCustomisedPlanRequest creates and subscribes to an investment plan of subscribable products
type PrivateWealthManagementCustomisedPlanRequest struct {
	// AccountType is the source account, FUND by default
	AccountType string
	Products    []PrivateWealthManagementCustomisedPlanProduct
	// OrderLinkID is undocumented as a request parameter, but the response echoes it and Bybit's Python SDK requires
	// it, so it is sent when set
	OrderLinkID string
}

// PrivateWealthManagementCustomisedPlanProduct is a product of a customised investment plan; Category, ProductID and
// FundName are passed through from GetPrivateWealthManagementSubscribableProductInfo
type PrivateWealthManagementCustomisedPlanProduct struct {
	Category  string
	ProductID string
	FundName  string
	// Amount is in the product's coin
	Amount float64
}

// PrivateWealthManagementCustomisedPlanResponse is a created customised investment plan, Active once subscribed
type PrivateWealthManagementCustomisedPlanResponse struct {
	PlanID string `json:"planId"`
	// PlanName is generated as PWM-{planId}
	PlanName    string `json:"planName"`
	Status      string `json:"status"`
	OrderLinkID string `json:"orderLinkId"`
}

// PrivateWealthManagementFundsRequest filters the institution's funds
type PrivateWealthManagementFundsRequest struct {
	Coin   currency.Code
	FundID string
	Status string
	Limit  uint64
	// Cursor is a fund ID
	Cursor string
}

// PrivateWealthManagementFundsResponse holds a page of the institution's funds
type PrivateWealthManagementFundsResponse struct {
	List []PrivateWealthManagementFund `json:"list"`
	// NextPageCursor is empty on the last page
	NextPageCursor string `json:"nextPageCursor"`
}

// PrivateWealthManagementFund is a fund managed by the institution
type PrivateWealthManagementFund struct {
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

// PrivateWealthManagementSettleFundProfitResponse is the outcome of a fund profit settlement
type PrivateWealthManagementSettleFundProfitResponse struct {
	FundID string `json:"fundId"`
	Status string `json:"status"`
	// TotalProfitShared and InstitutionIncome are in the coin
	TotalProfitShared types.Number  `json:"totalProfitShared"`
	InstitutionIncome types.Number  `json:"instIncome"`
	Coin              currency.Code `json:"coin"`
	CreatedTime       types.Time    `json:"createdTime"`
}

// PrivateWealthManagementCreateFundRequest creates a fund pending subscription
type PrivateWealthManagementCreateFundRequest struct {
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

// PrivateWealthManagementCreateFundResponse is a created fund
type PrivateWealthManagementCreateFundResponse struct {
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

// PrivateWealthManagementCreateInvestmentPlanRequest creates an investment plan of the institution's funds for a user
type PrivateWealthManagementCreateInvestmentPlanRequest struct {
	AccountUID string
	// PlanName is at most 50 characters
	PlanName               string
	PlanType               string
	InvestmentDistribution []PrivateWealthManagementPlanFundAllocation
	RequestLinkID          string
}

// PrivateWealthManagementPlanFundAllocation is a fund configured into a new investment plan
type PrivateWealthManagementPlanFundAllocation struct {
	// ProductID is the fund ID
	ProductID string
	// Amount is in the fund's coin
	Amount float64
}

// PrivateWealthManagementCreateInvestmentPlanResponse is a created investment plan, PendingSubscription until the user
// subscribes
type PrivateWealthManagementCreateInvestmentPlanResponse struct {
	PlanID      string     `json:"planId"`
	PlanType    string     `json:"planType"`
	AccountUID  string     `json:"accountUid"`
	Status      string     `json:"status"`
	CreatedTime types.Time `json:"createdTime"`
}

// PrivateWealthManagementAssetManagerInvestmentPlansRequest filters the investment plans the institution created
type PrivateWealthManagementAssetManagerInvestmentPlansRequest struct {
	PlanID          string
	Status          string
	SubscriptionUID string
	Limit           uint64
	// Cursor is an investment plan ID
	Cursor string
}

// PrivateWealthManagementAssetManagerInvestmentPlansResponse holds a page of the investment plans the institution
// created
type PrivateWealthManagementAssetManagerInvestmentPlansResponse struct {
	List []PrivateWealthManagementAssetManagerInvestmentPlan `json:"list"`
	// NextPageCursor is empty on the last page
	NextPageCursor string `json:"nextPageCursor"`
}

// PrivateWealthManagementAssetManagerInvestmentPlan is an investment plan the institution created
type PrivateWealthManagementAssetManagerInvestmentPlan struct {
	PlanID   string `json:"planId"`
	PlanName string `json:"planName"`
	PlanType string `json:"planType"`
	// SubscriptionUID is the user who subscribed to the plan
	SubscriptionUID string `json:"subscriptionUid"`
	Status          string `json:"status"`
	// Source is who created the plan: consultant, direct or institution
	Source string `json:"source"`
	// CurrentAssetUSD is the configured amount while the plan is PendingSubscription
	CurrentAssetUSD        types.Number                                    `json:"currentAssetUsd"`
	AccumulateYieldUSD     types.Number                                    `json:"accumulateYieldUsd"`
	InvestmentDistribution []PrivateWealthManagementInvestmentDistribution `json:"investmentDistribution"`
	CreatedTime            types.Time                                      `json:"createdTime"`
}

// PrivateWealthManagementManageInvestmentPlanRequest closes or deletes an investment plan, or adds funds and updates
// their amounts
type PrivateWealthManagementManageInvestmentPlanRequest struct {
	PlanID string
	// UpdateStatus is Closed or Deleted; empty leaves the status unchanged
	UpdateStatus string
	// UpdateFunds updates the amount of a fund already in the plan, or adds it; at most 10 funds without duplicates
	UpdateFunds   []PrivateWealthManagementFundAllocation
	RequestLinkID string
}

// PrivateWealthManagementFundAllocation is a fund's configured amount in an investment plan update
type PrivateWealthManagementFundAllocation struct {
	FundID string
	// Amount is in the fund's coin
	Amount float64
}

// PrivateWealthManagementManageInvestmentPlanResponse is an investment plan after an update
type PrivateWealthManagementManageInvestmentPlanResponse struct {
	PlanID      string                               `json:"planId"`
	Status      string                               `json:"status"`
	UpdatedTime types.Time                           `json:"updatedTime"`
	UpdateFunds []PrivateWealthManagementUpdatedFund `json:"updateFunds"`
}

// PrivateWealthManagementUpdatedFund is a fund configured into an investment plan after an update
type PrivateWealthManagementUpdatedFund struct {
	FundID string `json:"fundId"`
	// Amount is in the fund's coin
	Amount types.Number `json:"amount"`
}

// PrivateWealthManagementFundOrdersRequest filters the orders of the institution's funds. Without a window the last 7
// days are returned; with one, at most the 7 days before the end time
type PrivateWealthManagementFundOrdersRequest struct {
	FundID    string
	OrderType string
	Status    string
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
	// Cursor is an order ID
	Cursor string
}

// PrivateWealthManagementFundOrdersResponse holds a page of fund orders
type PrivateWealthManagementFundOrdersResponse struct {
	List []PrivateWealthManagementFundOrder `json:"list"`
	// NextPageCursor is empty on the last page
	NextPageCursor string `json:"nextPageCursor"`
}

// PrivateWealthManagementFundOrder is a subscription or redemption order of a fund
type PrivateWealthManagementFundOrder struct {
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

// PrivateWealthManagementManageOrderRequest approves or rejects a fund order pending review
type PrivateWealthManagementManageOrderRequest struct {
	OrderID string
	// Action is approve or reject
	Action        string
	RequestLinkID string
}

// PrivateWealthManagementManageOrderResponse is a fund order after approval or rejection
type PrivateWealthManagementManageOrderResponse struct {
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

// PrivateWealthManagementCreateFundSubAccountResponse is the creation status of a fund sub-account
type PrivateWealthManagementCreateFundSubAccountResponse struct {
	FundID string `json:"fundId"`
	// SubAccountUID is "0" until the sub-account is created
	SubAccountUID string     `json:"subAccountUid"`
	CreatedTime   types.Time `json:"createdTime"`
	Status        string     `json:"status"`
}
