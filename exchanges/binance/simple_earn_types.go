package binance

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// SimpleEarnCollateralRecordRequest holds the parameters of GetSimpleEarnCollateralRecord
type SimpleEarnCollateralRecordRequest struct {
	ProductID string
	StartTime time.Time
	EndTime   time.Time
	Current   uint64
	Size      uint64
}

// SimpleEarnCollateralRecordResponse holds a page of Simple Earn collateral records
type SimpleEarnCollateralRecordResponse struct {
	Rows  []SimpleEarnCollateralRecord `json:"rows"`
	Total types.Number                 `json:"total"`
}

// SimpleEarnCollateralRecord holds a flexible product amount used as loan collateral
type SimpleEarnCollateralRecord struct {
	Amount      types.Number  `json:"amount"`
	ProductID   string        `json:"productId"`
	Asset       currency.Code `json:"asset"`
	CreateTime  types.Time    `json:"createTime"`
	Type        string        `json:"type"`
	ProductName string        `json:"productName"`
	OrderID     uint64        `json:"orderId"`
}

// SimpleEarnPersonalLeftQuotaResponse holds the account's remaining subscription quota for a product
type SimpleEarnPersonalLeftQuotaResponse struct {
	LeftPersonalQuota types.Number `json:"leftPersonalQuota"`
}

// SimpleEarnFlexiblePositionRequest holds the parameters of GetFlexibleProductPosition
type SimpleEarnFlexiblePositionRequest struct {
	Asset     currency.Code
	ProductID string
	Current   uint64
	Size      uint64
}

// SimpleEarnFlexiblePositionResponse holds a page of flexible product positions
type SimpleEarnFlexiblePositionResponse struct {
	Rows  []SimpleEarnFlexiblePosition `json:"rows"`
	Total uint64                       `json:"total"`
}

// SimpleEarnFlexiblePosition holds a flexible product position
type SimpleEarnFlexiblePosition struct {
	TotalAmount                    types.Number       `json:"totalAmount"`
	TierAnnualPercentageRate       map[string]float64 `json:"tierAnnualPercentageRate"` // keyed by tier, such as 0-5BTC
	LatestAnnualPercentageRate     types.Number       `json:"latestAnnualPercentageRate"`
	YesterdayAirdropPercentageRate types.Number       `json:"yesterdayAirdropPercentageRate"`
	Asset                          currency.Code      `json:"asset"`
	AirDropAsset                   currency.Code      `json:"airDropAsset"`
	CanRedeem                      bool               `json:"canRedeem"`
	CollateralAmount               types.Number       `json:"collateralAmount"`
	ProductID                      string             `json:"productId"`
	YesterdayRealTimeRewards       types.Number       `json:"yesterdayRealTimeRewards"`
	CumulativeBonusRewards         types.Number       `json:"cumulativeBonusRewards"`
	CumulativeRealTimeRewards      types.Number       `json:"cumulativeRealTimeRewards"`
	CumulativeTotalRewards         types.Number       `json:"cumulativeTotalRewards"`
	AutoSubscribe                  bool               `json:"autoSubscribe"`
}

// SimpleEarnFlexibleRedemptionRecordRequest holds the parameters of GetFlexibleRedemptionRecord
type SimpleEarnFlexibleRedemptionRecordRequest struct {
	ProductID string
	RedeemID  uint64
	Asset     currency.Code
	StartTime time.Time
	EndTime   time.Time
	Current   uint64
	Size      uint64
}

// SimpleEarnFlexibleRedemptionRecordResponse holds a page of flexible product redemptions
type SimpleEarnFlexibleRedemptionRecordResponse struct {
	Rows  []SimpleEarnFlexibleRedemption `json:"rows"`
	Total uint64                         `json:"total"`
}

// SimpleEarnFlexibleRedemption holds a flexible product redemption
type SimpleEarnFlexibleRedemption struct {
	Amount             types.Number  `json:"amount"`
	Asset              currency.Code `json:"asset"`
	Time               types.Time    `json:"time"`
	ProjectID          string        `json:"projectId"`
	RedeemID           uint64        `json:"redeemId"`
	DestinationAccount string        `json:"destAccount"`
	Status             string        `json:"status"`
}

// SimpleEarnFlexibleRewardHistoryRequest holds the parameters of GetFlexibleRewardHistory
type SimpleEarnFlexibleRewardHistoryRequest struct {
	ProductID string
	Asset     currency.Code
	Type      string // BONUS, REALTIME, REWARDS or ALL; ALL when empty
	StartTime time.Time
	EndTime   time.Time
	Current   uint64
	Size      uint64
}

// SimpleEarnFlexibleRewardHistoryResponse holds a page of flexible product rewards
type SimpleEarnFlexibleRewardHistoryResponse struct {
	Rows  []SimpleEarnFlexibleReward `json:"rows"`
	Total uint64                     `json:"total"`
}

// SimpleEarnFlexibleReward holds a flexible product reward
type SimpleEarnFlexibleReward struct {
	Asset     currency.Code `json:"asset"`
	Rewards   types.Number  `json:"rewards"`
	ProjectID string        `json:"projectId"`
	Type      string        `json:"type"`
	Time      types.Time    `json:"time"`
}

// SimpleEarnFlexibleSubscriptionPreviewResponse holds the estimated rewards of a flexible product subscription
type SimpleEarnFlexibleSubscriptionPreviewResponse struct {
	TotalAmount                   types.Number  `json:"totalAmount"`
	RewardAsset                   currency.Code `json:"rewardAsset"`
	AirDropAsset                  currency.Code `json:"airDropAsset"`
	EstimatedDailyBonusRewards    types.Number  `json:"estDailyBonusRewards"`
	EstimatedDailyRealTimeRewards types.Number  `json:"estDailyRealTimeRewards"`
	EstimatedDailyAirdropRewards  types.Number  `json:"estDailyAirdropRewards"`
}

// SimpleEarnFlexibleSubscriptionRecordRequest holds the parameters of GetFlexibleSubscriptionRecord
type SimpleEarnFlexibleSubscriptionRecordRequest struct {
	ProductID  string
	PurchaseID uint64
	Asset      currency.Code
	StartTime  time.Time
	EndTime    time.Time
	Current    uint64
	Size       uint64
}

// SimpleEarnFlexibleSubscriptionRecordResponse holds a page of flexible product subscriptions
type SimpleEarnFlexibleSubscriptionRecordResponse struct {
	Rows  []SimpleEarnFlexibleSubscription `json:"rows"`
	Total uint64                           `json:"total"`
}

// SimpleEarnFlexibleSubscription holds a flexible product subscription
type SimpleEarnFlexibleSubscription struct {
	Amount            types.Number  `json:"amount"`
	Asset             currency.Code `json:"asset"`
	Time              types.Time    `json:"time"`
	PurchaseID        uint64        `json:"purchaseId"`
	ProductID         string        `json:"productId"`
	Type              string        `json:"type"`
	SourceAccount     string        `json:"sourceAccount"`
	AmountFromSpot    types.Number  `json:"amtFromSpot"`
	AmountFromFunding types.Number  `json:"amtFromFunding"`
	Status            string        `json:"status"`
}

// SimpleEarnLockedPositionRequest holds the parameters of GetLockedProductPosition
type SimpleEarnLockedPositionRequest struct {
	Asset      currency.Code
	PositionID uint64
	ProjectID  string
	Current    uint64
	Size       uint64
}

// SimpleEarnLockedPositionResponse holds a page of locked product positions
type SimpleEarnLockedPositionResponse struct {
	Rows  []SimpleEarnLockedPosition `json:"rows"`
	Total uint64                     `json:"total"`
}

// SimpleEarnLockedPosition holds a locked product position
type SimpleEarnLockedPosition struct {
	PositionID                 uint64        `json:"positionId"`
	ParentPositionID           uint64        `json:"parentPositionId"`
	ProjectID                  string        `json:"projectId"`
	Asset                      currency.Code `json:"asset"`
	Amount                     types.Number  `json:"amount"`
	PurchaseTime               types.Time    `json:"purchaseTime"`
	Duration                   types.Number  `json:"duration"`    // days
	AccrualDays                types.Number  `json:"accrualDays"` // days
	RewardAsset                currency.Code `json:"rewardAsset"`
	APY                        types.Number  `json:"APY"`
	RewardAmount               types.Number  `json:"rewardAmt"`
	ExtraRewardAsset           currency.Code `json:"extraRewardAsset"`
	ExtraRewardAPR             types.Number  `json:"extraRewardAPR"`
	EstimatedExtraRewardAmount types.Number  `json:"estExtraRewardAmt"`
	BoostRewardAsset           currency.Code `json:"boostRewardAsset"`
	BoostAPR                   types.Number  `json:"boostApr"`
	TotalBoostRewardAmount     types.Number  `json:"totalBoostRewardAmt"`
	NextPay                    types.Number  `json:"nextPay"`
	NextPayDate                types.Time    `json:"nextPayDate"`
	PayPeriod                  types.Number  `json:"payPeriod"` // days
	RedeemAmountEarly          types.Number  `json:"redeemAmountEarly"`
	RewardsEndDate             types.Time    `json:"rewardsEndDate"`
	DeliverDate                types.Time    `json:"deliverDate"`
	RedeemPeriod               types.Number  `json:"redeemPeriod"` // days
	RedeemingAmount            types.Number  `json:"redeemingAmt"`
	RedeemTo                   string        `json:"redeemTo"`
	PartialAmountDeliverDate   types.Time    `json:"partialAmtDeliverDate"`
	CanRedeemEarly             bool          `json:"canRedeemEarly"`
	CanFastRedemption          bool          `json:"canFastRedemption"`
	AutoSubscribe              bool          `json:"autoSubscribe"`
	Type                       string        `json:"type"`
	Status                     string        `json:"status"`
	CanReStake                 bool          `json:"canReStake"`
}

// SimpleEarnLockedRedemptionRecordRequest holds the parameters of GetLockedRedemptionRecord
type SimpleEarnLockedRedemptionRecordRequest struct {
	PositionID uint64
	RedeemID   uint64
	Asset      currency.Code
	StartTime  time.Time
	EndTime    time.Time
	Current    uint64
	Size       uint64
}

// SimpleEarnLockedRedemptionRecordResponse holds a page of locked product redemptions
type SimpleEarnLockedRedemptionRecordResponse struct {
	Rows  []SimpleEarnLockedRedemption `json:"rows"`
	Total uint64                       `json:"total"`
}

// SimpleEarnLockedRedemption holds a locked product redemption
type SimpleEarnLockedRedemption struct {
	PositionID                 uint64        `json:"positionId"`
	RedeemID                   uint64        `json:"redeemId"`
	Time                       types.Time    `json:"time"`
	Asset                      currency.Code `json:"asset"`
	LockPeriod                 types.Number  `json:"lockPeriod"` // days
	Amount                     types.Number  `json:"amount"`
	OriginalAmount             types.Number  `json:"originalAmount"`
	Type                       string        `json:"type"`
	DeliverDate                types.Time    `json:"deliverDate"`
	LossAmount                 types.Number  `json:"lossAmount"`
	IsComplete                 bool          `json:"isComplete"`
	RewardAsset                currency.Code `json:"rewardAsset"`
	RewardAmount               types.Number  `json:"rewardAmt"`
	ExtraRewardAsset           currency.Code `json:"extraRewardAsset"`
	EstimatedExtraRewardAmount types.Number  `json:"estExtraRewardAmt"`
	Status                     string        `json:"status"`
}

// SimpleEarnLockedRewardHistoryRequest holds the parameters of GetLockedRewardHistory
type SimpleEarnLockedRewardHistoryRequest struct {
	PositionID uint64
	Asset      currency.Code
	StartTime  time.Time
	EndTime    time.Time
	Current    uint64
	Size       uint64
}

// SimpleEarnLockedRewardHistoryResponse holds a page of locked product rewards
type SimpleEarnLockedRewardHistoryResponse struct {
	Rows  []SimpleEarnLockedReward `json:"rows"`
	Total uint64                   `json:"total"`
}

// SimpleEarnLockedReward holds a locked product reward
type SimpleEarnLockedReward struct {
	PositionID uint64        `json:"positionId"`
	Time       types.Time    `json:"time"`
	Asset      currency.Code `json:"asset"`
	LockPeriod types.Number  `json:"lockPeriod"` // days
	Amount     types.Number  `json:"amount"`
	Type       string        `json:"type"`
}

// SimpleEarnLockedSubscriptionPreview holds the estimated rewards and schedule of a locked product subscription
type SimpleEarnLockedSubscriptionPreview struct {
	RewardAsset                     currency.Code `json:"rewardAsset"`
	TotalRewardAmount               types.Number  `json:"totalRewardAmt"`
	ExtraRewardAsset                currency.Code `json:"extraRewardAsset"`
	EstimatedTotalExtraRewardAmount types.Number  `json:"estTotalExtraRewardAmt"`
	BoostRewardAsset                currency.Code `json:"boostRewardAsset"`
	EstimatedDailyRewardAmount      types.Number  `json:"estDailyRewardAmt"`
	NextPay                         types.Number  `json:"nextPay"`
	NextPayDate                     types.Time    `json:"nextPayDate"`
	ValueDate                       types.Time    `json:"valueDate"`
	RewardsEndDate                  types.Time    `json:"rewardsEndDate"`
	DeliverDate                     types.Time    `json:"deliverDate"`
	NextSubscriptionDate            types.Time    `json:"nextSubscriptionDate"`
}

// SimpleEarnLockedSubscriptionRecordRequest holds the parameters of GetLockedSubscriptionRecord
type SimpleEarnLockedSubscriptionRecordRequest struct {
	PurchaseID uint64
	Asset      currency.Code
	StartTime  time.Time
	EndTime    time.Time
	Current    uint64
	Size       uint64
}

// SimpleEarnLockedSubscriptionRecordResponse holds a page of locked product subscriptions
type SimpleEarnLockedSubscriptionRecordResponse struct {
	Rows  []SimpleEarnLockedSubscription `json:"rows"`
	Total uint64                         `json:"total"`
}

// SimpleEarnLockedSubscription holds a locked product subscription
type SimpleEarnLockedSubscription struct {
	PositionID        uint64        `json:"positionId"`
	PurchaseID        string        `json:"purchaseId"`
	ProjectID         string        `json:"projectId"`
	Time              types.Time    `json:"time"`
	Asset             currency.Code `json:"asset"`
	Amount            types.Number  `json:"amount"`
	LockPeriod        types.Number  `json:"lockPeriod"` // days
	Type              string        `json:"type"`
	SourceAccount     string        `json:"sourceAccount"`
	AmountFromSpot    types.Number  `json:"amtFromSpot"`
	AmountFromFunding types.Number  `json:"amtFromFunding"`
	Status            string        `json:"status"`
}

// SimpleEarnRateHistoryRequest holds the parameters of GetSimpleEarnRateHistory
type SimpleEarnRateHistoryRequest struct {
	ProductID string
	APRPeriod string // DAY or YEAR; DAY when empty
	StartTime time.Time
	EndTime   time.Time
	Current   uint64
	Size      uint64
}

// SimpleEarnRateHistoryResponse holds a page of flexible product annual percentage rates
type SimpleEarnRateHistoryResponse struct {
	Rows  []SimpleEarnRate `json:"rows"`
	Total types.Number     `json:"total"`
}

// SimpleEarnRate holds a flexible product's annual percentage rate at a point in time
type SimpleEarnRate struct {
	ProductID            string        `json:"productId"`
	Asset                currency.Code `json:"asset"`
	AnnualPercentageRate types.Number  `json:"annualPercentageRate"`
	Time                 types.Time    `json:"time"`
}

// SimpleEarnFlexibleProductListResponse holds a page of Simple Earn flexible products
type SimpleEarnFlexibleProductListResponse struct {
	Rows  []SimpleEarnFlexibleProduct `json:"rows"`
	Total uint64                      `json:"total"`
}

// SimpleEarnFlexibleProduct holds a Simple Earn flexible product
type SimpleEarnFlexibleProduct struct {
	Asset                      currency.Code      `json:"asset"`
	LatestAnnualPercentageRate types.Number       `json:"latestAnnualPercentageRate"`
	TierAnnualPercentageRate   map[string]float64 `json:"tierAnnualPercentageRate"` // keyed by tier, such as 0-5BTC
	AirDropPercentageRate      types.Number       `json:"airDropPercentageRate"`
	CanPurchase                bool               `json:"canPurchase"`
	CanRedeem                  bool               `json:"canRedeem"`
	IsSoldOut                  bool               `json:"isSoldOut"`
	Hot                        bool               `json:"hot"`
	MinimumPurchaseAmount      types.Number       `json:"minPurchaseAmount"`
	ProductID                  string             `json:"productId"`
	SubscriptionStartTime      types.Time         `json:"subscriptionStartTime"`
	Status                     string             `json:"status"`
}

// SimpleEarnLockedProductListResponse holds a page of Simple Earn locked products
type SimpleEarnLockedProductListResponse struct {
	Rows  []SimpleEarnLockedProduct `json:"rows"`
	Total uint64                    `json:"total"`
}

// SimpleEarnLockedProduct holds a Simple Earn locked product
type SimpleEarnLockedProduct struct {
	ProjectID string                        `json:"projectId"`
	Detail    SimpleEarnLockedProductDetail `json:"detail"`
	Quota     SimpleEarnLockedProductQuota  `json:"quota"`
}

// SimpleEarnLockedProductDetail holds the terms and rewards of a Simple Earn locked product
type SimpleEarnLockedProductDetail struct {
	Asset                 currency.Code `json:"asset"`
	RewardAsset           currency.Code `json:"rewardAsset"`
	Duration              uint64        `json:"duration"` // days
	Renewable             bool          `json:"renewable"`
	IsSoldOut             bool          `json:"isSoldOut"`
	APR                   types.Number  `json:"apr"`
	Status                string        `json:"status"`
	SubscriptionStartTime types.Time    `json:"subscriptionStartTime"`
	ExtraRewardAsset      currency.Code `json:"extraRewardAsset"`
	ExtraRewardAPR        types.Number  `json:"extraRewardAPR"`
	BoostRewardAsset      currency.Code `json:"boostRewardAsset"`
	BoostAPR              types.Number  `json:"boostApr"`
	BoostEndTime          types.Time    `json:"boostEndTime"`
}

// SimpleEarnLockedProductQuota holds the personal quota and minimum subscription of a Simple Earn locked product
type SimpleEarnLockedProductQuota struct {
	TotalPersonalQuota types.Number `json:"totalPersonalQuota"`
	Minimum            types.Number `json:"minimum"`
}

// RedeemFlexibleProductRequest holds the parameters of RedeemFlexibleProduct
type RedeemFlexibleProductRequest struct {
	ProductID          string
	RedeemAll          bool
	Amount             float64 // required unless RedeemAll is set
	DestinationAccount string  // SPOT or FUND; SPOT when empty
}

// SimpleEarnRedemptionResponse holds the outcome of a Simple Earn redemption
type SimpleEarnRedemptionResponse struct {
	RedeemID uint64 `json:"redeemId"`
	Success  bool   `json:"success"`
}

// SimpleAccountResponse holds the value held in Simple Earn products
type SimpleAccountResponse struct {
	TotalAmountInBTC          types.Number `json:"totalAmountInBTC"`
	TotalAmountInUSDT         types.Number `json:"totalAmountInUSDT"`
	TotalFlexibleAmountInBTC  types.Number `json:"totalFlexibleAmountInBTC"`
	TotalFlexibleAmountInUSDT types.Number `json:"totalFlexibleAmountInUSDT"`
	TotalLockedInBTC          types.Number `json:"totalLockedInBTC"`
	TotalLockedInUSDT         types.Number `json:"totalLockedInUSDT"`
}

// SubscribeFlexibleProductRequest holds the parameters of SubscribeToFlexibleProducts
type SubscribeFlexibleProductRequest struct {
	ProductID string
	Amount    float64
	// AutoSubscribe is sent only when set; the endpoint turns auto-subscription on when it is left out
	AutoSubscribe *bool
	SourceAccount string // SPOT, FUND or ALL; SPOT when empty
}

// SimpleEarnFlexibleSubscriptionResponse holds the outcome of a flexible product subscription
type SimpleEarnFlexibleSubscriptionResponse struct {
	PurchaseID uint64 `json:"purchaseId"`
	Success    bool   `json:"success"`
}

// SubscribeLockedProductRequest holds the parameters of SubscribeToLockedProducts
type SubscribeLockedProductRequest struct {
	ProjectID string
	Amount    float64
	// AutoSubscribe is sent only when set; the endpoint turns auto-subscription on when it is left out
	AutoSubscribe *bool
	SourceAccount string // SPOT, FUND or ALL; SPOT when empty
	RedeemTo      string // SPOT or FLEXIBLE; SPOT when empty
}

// SimpleEarnLockedSubscriptionResponse holds the outcome of a locked product subscription
type SimpleEarnLockedSubscriptionResponse struct {
	PurchaseID uint64 `json:"purchaseId"`
	PositionID string `json:"positionId"`
	Success    bool   `json:"success"`
}
