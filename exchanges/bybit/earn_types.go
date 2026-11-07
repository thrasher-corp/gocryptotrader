package bybit

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// EarnProductInfoResponse holds Flexible Savings and On-Chain Earn products
type EarnProductInfoResponse struct {
	List []EarnProduct `json:"list"`
}

// EarnProduct is a Flexible Savings or On-Chain Earn product
type EarnProduct struct {
	Category string `json:"category"`
	// EstimateAPR is formatted for display, such as "4.25%"
	EstimateAPR    string           `json:"estimateApr"`
	Coin           currency.Code    `json:"coin"`
	MinStakeAmount types.Number     `json:"minStakeAmount"`
	MaxStakeAmount types.Number     `json:"maxStakeAmount"`
	Precision      types.Number     `json:"precision"`
	ProductID      string           `json:"productId"`
	Status         string           `json:"status"`
	BonusEvents    []EarnBonusEvent `json:"bonusEvents"`
	// MinRedeemAmount, MaxRedeemAmount, SwapCoin, SwapCoinPrecision, StakeExchangeRate and RedeemExchangeRate are only
	// set for On-Chain liquid staking token products, where the coin differs from the swap coin
	MinRedeemAmount types.Number `json:"minRedeemAmount"`
	MaxRedeemAmount types.Number `json:"maxRedeemAmount"`
	// Duration is Fixed or Flexible for On-Chain products
	Duration string `json:"duration"`
	// Term is the lock-up in days of On-Chain Fixed products
	Term                   uint64        `json:"term"`
	SwapCoin               currency.Code `json:"swapCoin"`
	SwapCoinPrecision      types.Number  `json:"swapCoinPrecision"`
	StakeExchangeRate      types.Number  `json:"stakeExchangeRate"`
	RedeemExchangeRate     types.Number  `json:"redeemExchangeRate"`
	RewardDistributionType string        `json:"rewardDistributionType"`
	// RewardIntervalMinute is the reward distribution interval in minutes
	RewardIntervalMinute uint64 `json:"rewardIntervalMinute"`
	// RedeemProcessingMinute is the estimated redemption time in minutes; documented as a string, sent as a number
	RedeemProcessingMinute  uint64     `json:"redeemProcessingMinute"`
	StakeTime               types.Time `json:"stakeTime"`
	InterestCalculationTime types.Time `json:"interestCalculationTime"`
	// HasTieredAPR, TierAPRDetails and RemainingPoolAmount are undocumented; only Flexible Savings products fill them
	HasTieredAPR        bool                `json:"hasTieredApr"`
	TierAPRDetails      []EarnTierAPRDetail `json:"tierAprDetails"`
	RemainingPoolAmount types.Number        `json:"remainingPoolAmount"`
}

// EarnBonusEvent is a bonus reward of an Earn product
type EarnBonusEvent struct {
	// APR is yesterday's reward APR
	APR          string        `json:"apr"`
	Coin         currency.Code `json:"coin"`
	Announcement string        `json:"announcement"`
}

// EarnTierAPRDetail is the estimated APR of an amount tier of a Flexible Savings product; undocumented
type EarnTierAPRDetail struct {
	Min types.Number `json:"min"`
	// Max is -1 for the tier without an upper bound
	Max types.Number `json:"max"`
	// EstimateAPR is formatted for display, such as "0.4%"
	EstimateAPR string `json:"estimateApr"`
}

// PlaceEarnOrderRequest is a Flexible Savings or On-Chain Earn stake or redeem order
type PlaceEarnOrderRequest struct {
	Category    string
	OrderType   string
	AccountType string
	Amount      float64
	Coin        currency.Code
	ProductID   string
	// OrderLinkID may not be reused within 30 minutes
	OrderLinkID string
	// RedeemPositionID is required to redeem On-Chain non-liquid staking token positions
	RedeemPositionID string
	ToAccountType    string
	// InterestCard applies an interest bonus card to a FlexibleSaving stake order
	InterestCard *EarnInterestCard
}

// EarnInterestCard identifies an interest bonus card from GetEarnCouponList
type EarnInterestCard struct {
	AwardID  uint64
	SpecCode string
}

// PlaceEarnOrderResponse identifies a placed Earn stake or redeem order
type PlaceEarnOrderResponse struct {
	OrderID     string `json:"orderId"`
	OrderLinkID string `json:"orderLinkId"`
}

// EarnOrderHistoryRequest filters Flexible Savings and On-Chain Earn orders. On-Chain orders need an OrderID or an
// OrderLinkID; a window wider than 7 days is rejected and none returns the last 7 days
type EarnOrderHistoryRequest struct {
	Category    string
	OrderID     string
	OrderLinkID string
	ProductID   string
	StartTime   time.Time
	EndTime     time.Time
	Limit       uint64
	Cursor      string
}

// EarnOrderHistoryResponse holds a page of Earn stake and redeem orders
type EarnOrderHistoryResponse struct {
	NextPageCursor string      `json:"nextPageCursor"`
	List           []EarnOrder `json:"list"`
}

// EarnOrder is a Flexible Savings or On-Chain Earn stake or redeem order
type EarnOrder struct {
	Coin        currency.Code `json:"coin"`
	OrderValue  types.Number  `json:"orderValue"`
	OrderType   string        `json:"orderType"`
	OrderID     string        `json:"orderId"`
	OrderLinkID string        `json:"orderLinkId"`
	Status      string        `json:"status"`
	CreatedAt   types.Time    `json:"createdAt"`
	ProductID   string        `json:"productId"`
	UpdatedAt   types.Time    `json:"updatedAt"`
	// SwapOrderValue is only set for On-Chain liquid staking token orders
	SwapOrderValue types.Number `json:"swapOrderValue"`
	// EstimateRedeemTime and EstimateStakeTime are only set for On-Chain orders
	EstimateRedeemTime types.Time `json:"estimateRedeemTime"`
	EstimateStakeTime  types.Time `json:"estimateStakeTime"`
}

// EarnStakedPositionRequest filters Flexible Savings and On-Chain Earn positions
type EarnStakedPositionRequest struct {
	Category  string
	ProductID string
	Coin      currency.Code
}

// EarnStakedPositionResponse holds Flexible Savings and On-Chain Earn positions. Fully redeemed Flexible Savings
// positions are included; On-Chain positions only while active
type EarnStakedPositionResponse struct {
	List []EarnStakedPosition `json:"list"`
}

// EarnStakedPosition is a Flexible Savings or On-Chain Earn position
type EarnStakedPosition struct {
	Coin      currency.Code `json:"coin"`
	ProductID string        `json:"productId"`
	Amount    types.Number  `json:"amount"`
	// TotalPNL is only set for On-Chain non-liquid staking token positions
	TotalPNL types.Number `json:"totalPnl"`
	// ClaimableYield accrues hourly and is distributed daily at 00:30 UTC; not set for On-Chain positions
	ClaimableYield types.Number `json:"claimableYield"`
	// ID, Status, OrderID, EstimateRedeemTime, EstimateStakeTime and EstimateInterestCalculationTime are only set for
	// On-Chain positions
	ID                              string     `json:"id"`
	Status                          string     `json:"status"`
	OrderID                         string     `json:"orderId"`
	EstimateRedeemTime              types.Time `json:"estimateRedeemTime"`
	EstimateStakeTime               types.Time `json:"estimateStakeTime"`
	EstimateInterestCalculationTime types.Time `json:"estimateInterestCalculationTime"`
	// SettlementTime is only set for On-Chain Fixed positions
	SettlementTime  types.Time         `json:"settlementTime"`
	AutoReinvest    string             `json:"autoReinvest"`
	AvailableAmount types.Number       `json:"availableAmount"`
	FreezeDetails   []EarnFreezeDetail `json:"freezeDetails"`
}

// EarnFreezeDetail is a frozen part of an Earn position
type EarnFreezeDetail struct {
	Amount      types.Number `json:"amount"`
	Description string       `json:"description"`
}

// ModifyEarnPositionRequest sets the auto-reinvestment of an On-Chain Earn position whose product duration is Fixed
type ModifyEarnPositionRequest struct {
	Category     string
	ProductID    uint64
	PositionID   uint64
	AutoReinvest bool
}

// EarnYieldHistoryRequest filters Flexible Savings and On-Chain Earn yields. ProductID is rejected for On-Chain; a
// window wider than 7 days is rejected and none returns the last 7 days
type EarnYieldHistoryRequest struct {
	Category  string
	ProductID string
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
	Cursor    string
}

// EarnYieldHistoryResponse holds a page of Earn yield distributions. The docs' table names the list "list", but their
// response example names it "yield", so either is decoded
type EarnYieldHistoryResponse struct {
	NextPageCursor string      `json:"nextPageCursor"`
	Yield          []EarnYield `json:"yield"`
}

// UnmarshalJSON decodes the yields from yield, or from list when the response has no yield
func (r *EarnYieldHistoryResponse) UnmarshalJSON(data []byte) error {
	type alias EarnYieldHistoryResponse
	aux := struct {
		*alias
		Yield []EarnYield `json:"yield"`
		List  []EarnYield `json:"list"`
	}{alias: (*alias)(r)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	switch {
	case aux.Yield != nil:
		r.Yield = aux.Yield
	case aux.List != nil:
		r.Yield = aux.List
	}
	return nil
}

// EarnYield is a Flexible Savings or On-Chain Earn yield distribution
type EarnYield struct {
	ProductID string        `json:"productId"`
	Coin      currency.Code `json:"coin"`
	// ID is only unique per user
	ID               string       `json:"id"`
	Amount           types.Number `json:"amount"`
	YieldType        string       `json:"yieldType"`
	DistributionMode string       `json:"distributionMode"`
	// EffectiveStakingAmount is the principal the yield was earned on
	EffectiveStakingAmount types.Number `json:"effectiveStakingAmount"`
	// OrderID is the redemption order, only set for Flexible Savings when DistributionMode is Manual
	OrderID   string     `json:"orderId"`
	Status    string     `json:"status"`
	CreatedAt types.Time `json:"createdAt"`
}

// EarnHourlyYieldHistoryResponse holds a page of Flexible Savings hourly yields
type EarnHourlyYieldHistoryResponse struct {
	NextPageCursor string            `json:"nextPageCursor"`
	List           []EarnHourlyYield `json:"list"`
}

// EarnHourlyYield is a Flexible Savings hourly yield
type EarnHourlyYield struct {
	ProductID string        `json:"productId"`
	Coin      currency.Code `json:"coin"`
	// ID is only unique per user
	ID                     string       `json:"id"`
	Amount                 types.Number `json:"amount"`
	EffectiveStakingAmount types.Number `json:"effectiveStakingAmount"`
	Status                 string       `json:"status"`
	HourlyDate             types.Time   `json:"hourlyDate"`
	CreatedAt              types.Time   `json:"createdAt"`
}

// EarnAPRHistoryRequest selects a Flexible Savings or On-Chain Earn product's APR history. A window wider than 182 days
// is rejected and none returns the last 7 days
type EarnAPRHistoryRequest struct {
	Category  string
	ProductID string
	StartTime time.Time
	EndTime   time.Time
}

// EarnAPRHistoryResponse holds a product's APR history, newest first
type EarnAPRHistoryResponse struct {
	List []EarnAPR `json:"list"`
}

// EarnAPR is a product's APR at a time: daily for On-Chain products, hourly for Flexible Savings
type EarnAPR struct {
	Timestamp types.Time `json:"timestamp"`
	// APR is formatted for display, such as "0.1%", although the docs describe a decimal
	APR string `json:"apr"`
}

// EarnCouponListResponse holds the interest-rate coupons and Dual Assets reward cards of a product category, in every
// state
type EarnCouponListResponse struct {
	InterestCards []EarnCouponInterestCard `json:"interestCards"`
	AwardCards    []EarnCouponAwardCard    `json:"awardCards"`
}

// EarnCouponInterestCard is an interest-rate coupon; pass its AwardID and SpecCode as a stake order's EarnInterestCard
type EarnCouponInterestCard struct {
	AwardID  uint64        `json:"awardId"`
	SpecCode string        `json:"specCode"`
	Coin     currency.Code `json:"coin"`
	// APY is the bonus APY as a decimal, 0.03 for 3%
	APY types.Number `json:"apy"`
	// Duration is the coupon validity in days
	Duration  uint64     `json:"duration"`
	ClaimedAt types.Time `json:"claimedAt"`
	ExpireAt  types.Time `json:"expireAt"`
	// UsedAt is zero while the coupon is unused
	UsedAt                  types.Time   `json:"usedAt"`
	Status                  string       `json:"status"`
	CurrentPNL              types.Number `json:"currentPnl"`
	LimitPNL                types.Number `json:"limitPnl"`
	PositionEffectiveAmount types.Number `json:"positionEffectiveAmount"`
	// ProductID is set once the coupon is in use or used
	ProductID uint64 `json:"productId"`
	Category  string `json:"category"`
}

// EarnCouponAwardCard is a Dual Assets reward card, such as a trial fund or zero-cost voucher
type EarnCouponAwardCard struct {
	AwardID   uint64     `json:"awardId"`
	SpecCode  string     `json:"specCode"`
	ClaimedAt types.Time `json:"claimedAt"`
	// UsedAt is zero while the card is unused
	UsedAt   types.Time   `json:"usedAt"`
	ExpireAt types.Time   `json:"expireAt"`
	Status   string       `json:"status"`
	Amount   types.Number `json:"amount"`
	// LimitPNLPercentage is the PnL cap as a decimal, 0.23 for 23%
	LimitPNLPercentage types.Number  `json:"limitPnlPercentage"`
	BaseCoin           currency.Code `json:"baseCoin"`
	QuoteCoin          currency.Code `json:"quoteCoin"`
	// Direction is 1 for BuyLow and 2 for SellHigh
	Direction uint64 `json:"direction"`
	Category  string `json:"category"`
}

// BYUSDTProductInfoResponse is the BYUSDT product. UserHolding, LeftQuota, CanMint and SavingsBalance are only set for
// signed requests
type BYUSDTProductInfoResponse struct {
	ProductID string        `json:"productId"`
	Coin      currency.Code `json:"coin"`
	// MintFeeRateE8, RedeemFeeRateE8, APRE8 and BonusAPRE8 are rates multiplied by 1e8
	MintFeeRateE8   types.Number `json:"mintFeeRateE8"`
	RedeemFeeRateE8 types.Number `json:"redeemFeeRateE8"`
	// MinInvestment is in USDT
	MinInvestment  types.Number `json:"minInvestment"`
	UserHolding    types.Number `json:"userHolding"`
	LeftQuota      types.Number `json:"leftQuota"`
	CanMint        bool         `json:"canMint"`
	SavingsBalance types.Number `json:"savingsBalance"`
	APRE8          types.Number `json:"aprE8"`
	BonusAPRE8     types.Number `json:"bonusAprE8"`
	// BonusMaxAmount is the most principal that earns the bonus APR
	BonusMaxAmount types.Number `json:"bonusMaxAmount"`
	// BaseCoinPrecision and TokenPrecision are the decimal places of USDT and BYUSDT amounts
	BaseCoinPrecision uint64 `json:"baseCoinPrecision"`
	TokenPrecision    uint64 `json:"tokenPrecision"`
}

// PlaceBYUSDTOrderRequest is a BYUSDT mint or redeem order. Mint takes USDT from the FlexibleSaving account; Redeem
// returns USDT to the UNIFIED account
type PlaceBYUSDTOrderRequest struct {
	Coin        currency.Code
	OrderLinkID string
	OrderType   string
	// Amount is in USDT to mint and in BYUSDT to redeem
	Amount      float64
	AccountType string
}

// PlaceBYUSDTOrderResponse identifies an accepted BYUSDT order, which settles asynchronously
type PlaceBYUSDTOrderResponse struct {
	OrderID     string `json:"orderId"`
	OrderLinkID string `json:"orderLinkId"`
}

// BYUSDTOrderListRequest filters BYUSDT orders; an OrderID or OrderLinkID alone selects one order
type BYUSDTOrderListRequest struct {
	Coin        currency.Code
	OrderLinkID string
	OrderID     string
	OrderType   string
	StartTime   time.Time
	EndTime     time.Time
	Cursor      string
	Limit       uint64
}

// BYUSDTOrderListResponse holds a page of BYUSDT orders
type BYUSDTOrderListResponse struct {
	List           []BYUSDTOrder `json:"list"`
	NextPageCursor string        `json:"nextPageCursor"`
}

// BYUSDTOrder is a BYUSDT mint or redeem order. TradFi balance changes are listed as Redeem orders with an
// ExternalEventType
type BYUSDTOrder struct {
	OrderID     string        `json:"orderId"`
	OrderLinkID string        `json:"orderLinkId"`
	OrderType   string        `json:"orderType"`
	FromCoin    currency.Code `json:"fromCoin"`
	ToCoin      currency.Code `json:"toCoin"`
	FromAmount  types.Number  `json:"fromAmount"`
	// ToAmount is the amount credited after fees
	ToAmount          types.Number `json:"toAmount"`
	ServiceFee        types.Number `json:"serviceFee"`
	FromAccount       string       `json:"fromAccount"`
	ToAccount         string       `json:"toAccount"`
	ExternalEventType string       `json:"externalEventType"`
	Status            string       `json:"status"`
	CreatedTime       types.Time   `json:"createdTime"`
}

// BYUSDTPositionInfoResponse is the BYUSDT holding and its yield
type BYUSDTPositionInfoResponse struct {
	TotalAmount    types.Number `json:"totalAmount"`
	TotalYield     types.Number `json:"totalYield"`
	YesterdayYield types.Number `json:"yesterdayYield"`
	// APRE8 and BonusAPRE8 are rates multiplied by 1e8; documented as integers, sent quoted
	APRE8          types.Number `json:"aprE8"`
	BonusAPRE8     types.Number `json:"bonusAprE8"`
	BonusMaxAmount types.Number `json:"bonusMaxAmount"`
	HasQuota       bool         `json:"hasQuota"`
}

// BYUSDTYieldRecordsRequest filters BYUSDT daily or hourly yields
type BYUSDTYieldRecordsRequest struct {
	Coin      currency.Code
	StartTime time.Time
	EndTime   time.Time
	Cursor    string
	Limit     uint64
}

// BYUSDTDailyYieldRecordsResponse holds a page of BYUSDT daily yields
type BYUSDTDailyYieldRecordsResponse struct {
	List           []BYUSDTDailyYield `json:"list"`
	NextPageCursor string             `json:"nextPageCursor"`
}

// BYUSDTDailyYield is a BYUSDT daily yield
type BYUSDTDailyYield struct {
	Yield       types.Number `json:"yield"`
	BonusYield  types.Number `json:"bonusYield"`
	Status      string       `json:"status"`
	CreatedTime types.Time   `json:"createdTime"`
}

// BYUSDTHourlyYieldRecordsResponse holds a page of BYUSDT hourly yields
type BYUSDTHourlyYieldRecordsResponse struct {
	List           []BYUSDTHourlyYield `json:"list"`
	NextPageCursor string              `json:"nextPageCursor"`
}

// BYUSDTHourlyYield is a BYUSDT hourly yield
type BYUSDTHourlyYield struct {
	// EffectiveAmount is the principal the yield was earned on
	EffectiveAmount types.Number `json:"effectiveAmount"`
	Yield           types.Number `json:"yield"`
	// RewardType is 0 for the base yield and 1 for the bonus yield
	RewardType uint64 `json:"rewardType"`
	// APRE8 is the applied APR multiplied by 1e8
	APRE8       types.Number `json:"aprE8"`
	HourlyDate  types.Time   `json:"hourlyDate"`
	CreatedTime types.Time   `json:"createdTime"`
}

// BYUSDTHistoryAPRResponse holds the BYUSDT daily APR, oldest first
type BYUSDTHistoryAPRResponse struct {
	List []BYUSDTHistoryAPR `json:"list"`
}

// BYUSDTHistoryAPR is the BYUSDT APR of a day
type BYUSDTHistoryAPR struct {
	Timestamp types.Time `json:"timestamp"`
	// APRE8 is the APR multiplied by 1e8
	APRE8 types.Number `json:"aprE8"`
}

// FixedSavingProductInfoResponse holds Fixed Saving products
type FixedSavingProductInfoResponse struct {
	List []FixedSavingProduct `json:"list"`
}

// FixedSavingProduct is a Fixed Saving product
type FixedSavingProduct struct {
	ProductID string        `json:"productId"`
	Category  string        `json:"category"`
	Coin      currency.Code `json:"coin"`
	// Duration is the term with its unit, such as "1d", "8h" or "2m"
	Duration       string                 `json:"duration"`
	Status         string                 `json:"status"`
	TieredAPYList  []FixedSavingTieredAPY `json:"tieredApyList"`
	MinStakeAmount types.Number           `json:"minStakeAmount"`
	MaxStakeAmount types.Number           `json:"maxStakeAmount"`
	// Precision is the decimal places of the coin
	Precision            uint64     `json:"precision"`
	SubscribeStartAt     types.Time `json:"subscribeStartAt"`
	SubscribeEndAt       types.Time `json:"subscribeEndAt"`
	AllowEarlyRedemption bool       `json:"allowEarlyRedemption"`
	// EarlyRedemptionAPY is formatted for display, such as "0.30%"
	EarlyRedemptionAPY string `json:"earlyRedemptionApy"`
	// RedemptionLimitDuration is the minimum holding time before early redemption, with its unit, such as "1d"
	RedemptionLimitDuration  string                       `json:"redemptionLimitDuration"`
	AllowAutoReinvest        bool                         `json:"allowAutoReinvest"`
	InterestCoinAPYList      []FixedSavingInterestCoinAPY `json:"interestCoinApyList"`
	IsVIP                    bool                         `json:"isVip"`
	CreditTime               types.Time                   `json:"creditTime"`
	SpecialUserGroupRequired bool                         `json:"specialUserGroupRequired"`
	SpecialUserGroupInfo     string                       `json:"specialUserGroupInfo"`
}

// FixedSavingTieredAPY is the APY of an amount tier of a Fixed Saving product
type FixedSavingTieredAPY struct {
	Min types.Number `json:"min"`
	// Max is -1 for the tier without an upper bound
	Max types.Number `json:"max"`
	APY string       `json:"apy"`
}

// FixedSavingInterestCoinAPY is the reward of a coin paid by a Fixed Saving product
type FixedSavingInterestCoinAPY struct {
	Coin currency.Code `json:"coin"`
	// APY is formatted for display, such as "2.50%"
	APY string `json:"apy"`
	// ExpectUnitEarning is the expected reward per unit invested, which moves with the reward coin's price
	ExpectUnitEarning types.Number `json:"expectUnitEarning"`
	CurrentPrice      types.Number `json:"currentPrice"`
}

// PlaceFixedSavingOrderRequest subscribes to a Fixed Saving product
type PlaceFixedSavingOrderRequest struct {
	ProductID   string
	Category    string
	Coin        currency.Code
	Amount      float64
	AccountType string
	OrderLinkID string
	// AutoInvest only applies to FundPool products
	AutoInvest bool
}

// PlaceFixedSavingOrderResponse identifies an accepted Fixed Saving order, which settles asynchronously
type PlaceFixedSavingOrderResponse struct {
	OrderID     string `json:"orderId"`
	OrderLinkID string `json:"orderLinkId"`
}

// RedeemFixedSavingRequest identifies a FundPool position to redeem early; the product must allow early redemption
type RedeemFixedSavingRequest struct {
	ProductID  string
	Category   string
	PositionID string
}

// RedeemFixedSavingResponse is the estimated outcome of an early Fixed Saving redemption, which is paid to the FUND
// account
type RedeemFixedSavingResponse struct {
	RedeemAmount types.Number `json:"redeemAmount"`
	// EstimatedEarnings are at the early redemption APY
	EstimatedEarnings types.Number `json:"estEarnings"`
}

// FixedSavingPositionInfoRequest filters active Fixed Saving positions
type FixedSavingPositionInfoRequest struct {
	ProductID string
	Category  string
	Coin      currency.Code
}

// FixedSavingPositionInfoResponse holds active Fixed Saving positions
type FixedSavingPositionInfoResponse struct {
	List []FixedSavingPosition `json:"list"`
}

// FixedSavingPosition is an active Fixed Saving position
type FixedSavingPosition struct {
	PositionID string        `json:"positionId"`
	ProductID  string        `json:"productId"`
	Category   string        `json:"category"`
	Coin       currency.Code `json:"coin"`
	Amount     types.Number  `json:"amount"`
	// EffectiveAmount earns interest from the next day
	EffectiveAmount types.Number `json:"effectiveAmount"`
	// Duration is the term with its unit, such as "1d", "8h" or "2m"
	Duration            string                               `json:"duration"`
	Status              string                               `json:"status"`
	SettlementTime      types.Time                           `json:"settlementTime"`
	CreatedAt           types.Time                           `json:"createdAt"`
	OrderID             string                               `json:"orderId"`
	EarlyRedeemInfo     *FixedSavingEarlyRedeemInfo          `json:"earlyRedeemInfo"`
	AllowAutoReinvest   bool                                 `json:"allowAutoReinvest"`
	AutoReinvest        string                               `json:"autoReinvest"`
	InterestCoinAPYList []FixedSavingPositionInterestCoinAPY `json:"interestCoinApyList"`
}

// FixedSavingEarlyRedeemInfo describes the early redemption of a Fixed Saving position
type FixedSavingEarlyRedeemInfo struct {
	AllowEarlyRedeem   bool          `json:"allowEarlyRedeem"`
	EarlyRedeemEarning types.Number  `json:"earlyRedeemEarning"`
	ReturnCoin         currency.Code `json:"returnCoin"`
	// RedemptionLimitDuration is the minimum holding time before early redemption, with its unit, such as "1d"
	RedemptionLimitDuration string `json:"redemptionLimitDuration"`
}

// FixedSavingPositionInterestCoinAPY is the reward of a coin paid on a Fixed Saving position
type FixedSavingPositionInterestCoinAPY struct {
	Coin currency.Code `json:"coin"`
	// APY is formatted for display, such as "2.50%"
	APY                 string       `json:"apy"`
	ExpectReturnEarning types.Number `json:"expectReturnEarning"`
	// Price is the locked price of the reward coin
	Price types.Number `json:"price"`
}

// FixedSavingOrderListRequest filters Fixed Saving orders; an OrderID alone selects one order. A ProductID requires a
// Category. The window applies to the creation time of Stake orders and the settlement time of Redeem orders
type FixedSavingOrderListRequest struct {
	OrderType string
	ProductID string
	Category  string
	OrderID   string
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
	Cursor    string
}

// FixedSavingOrderListResponse holds a page of Fixed Saving orders
type FixedSavingOrderListResponse struct {
	List           []FixedSavingOrder `json:"list"`
	NextPageCursor string             `json:"nextPageCursor"`
}

// FixedSavingOrder is a Fixed Saving stake, redeem or reinvest order
type FixedSavingOrder struct {
	OrderID     string        `json:"orderId"`
	OrderLinkID string        `json:"orderLinkId"`
	OrderType   string        `json:"orderType"`
	Status      string        `json:"status"`
	ProductID   string        `json:"productId"`
	Category    string        `json:"category"`
	Coin        currency.Code `json:"coin"`
	Amount      types.Number  `json:"amount"`
	// Duration is the term with its unit, such as "1d", "8h" or "2m"
	Duration string `json:"duration"`
	// AccountType is always FUND for Redeem orders
	AccountType    string     `json:"accountType"`
	SettlementTime types.Time `json:"settlementTime"`
	CreatedAt      types.Time `json:"createdAt"`
	// YieldInfoList is filled after settlement
	YieldInfoList []FixedSavingYieldInfo `json:"yieldInfoList"`
}

// FixedSavingYieldInfo is a yield paid on a Fixed Saving order
type FixedSavingYieldInfo struct {
	Coin      currency.Code `json:"coin"`
	Amount    types.Number  `json:"amount"`
	Status    string        `json:"status"`
	CreatedAt types.Time    `json:"createdAt"`
	// APY is formatted for display, such as "2.33%"
	APY string `json:"apy"`
}

// SetFixedSavingAutoInvestRequest enables or disables auto-reinvestment of an active Fixed Saving position whose
// product allows it
type SetFixedSavingAutoInvestRequest struct {
	ProductID  string
	Category   string
	PositionID string
	// Status is Enable or Disable
	Status string
}

// FlexibleSavingAutoSavingsResponse holds the Flexible Savings auto savings settings
type FlexibleSavingAutoSavingsResponse struct {
	SelectedAll bool                            `json:"selectedAll"`
	Coins       []FlexibleSavingAutoSavingsCoin `json:"coins"`
}

// FlexibleSavingAutoSavingsCoin is the Flexible Savings auto savings setting of a coin
type FlexibleSavingAutoSavingsCoin struct {
	Coin       currency.Code `json:"coin"`
	IsSelected bool          `json:"isSelected"`
	// MaxStakingAmount is the largest auto savings position, in the coin
	MaxStakingAmount types.Number `json:"maxStakingAmount"`
	// APR is a decimal, 0.05 for 5%
	APR types.Number `json:"apr"`
}

// SetFlexibleSavingAutoSavingsRequest enables or disables Flexible Savings auto savings
type SetFlexibleSavingAutoSavingsRequest struct {
	// Coin is the coin to update; empty updates the global setting
	Coin       currency.Code
	IsSelected bool
	// PurchaseImmediately subscribes the available Funding account balance when enabling
	PurchaseImmediately bool
}

// AirdropProductsResponse holds the Hold To Earn airdrop products, newest first
type AirdropProductsResponse struct {
	Products []AirdropProduct `json:"products"`
}

// AirdropProduct is a Hold To Earn airdrop product; a yield coin that differs from the product coin is a cross-coin
// airdrop
type AirdropProduct struct {
	CoinName currency.Code  `json:"coinName"`
	Yields   []AirdropYield `json:"yields"`
	Status   string         `json:"status"`
	// APY is yesterday's average APR across the yield coins and PersonalAPY the sum of the personal APRs, formatted for
	// display, such as "3.5%"
	APY             string `json:"apy"`
	PersonalAPY     string `json:"personalApy"`
	AnnouncementURL string `json:"announcementUrl"`
}

// AirdropYield is the yield of a coin paid by a Hold To Earn airdrop product
type AirdropYield struct {
	CoinName currency.Code `json:"coinName"`
	// APY is yesterday's APR and PersonalAPY the APR after the personal multiplier, formatted for display, such as "10%"
	APY         string `json:"apy"`
	PersonalAPY string `json:"personalApy"`
	// Multiplier is the personal coefficient applied to the APR; 1 without a position
	Multiplier types.Number `json:"multiplier"`
}

// AirdropDailyPNLRecordsRequest pages through Hold To Earn airdrop yields. TimeStart may be at most three months ago;
// no window returns the last three months
type AirdropDailyPNLRecordsRequest struct {
	TimeStart time.Time
	TimeEnd   time.Time
	// Limit is required, from 1 to 49
	Limit  uint64
	Cursor string
}

// AirdropDailyPNLRecordsResponse holds a page of Hold To Earn airdrop yields, newest first
type AirdropDailyPNLRecordsResponse struct {
	// NextCursor is empty on the last page
	NextCursor       string            `json:"nextCursor"`
	AirdropDailyPNLs []AirdropDailyPNL `json:"airdropDailyPnls"`
}

// AirdropDailyPNL is a Hold To Earn airdrop yield of a day
type AirdropDailyPNL struct {
	CoinName        currency.Code `json:"coinName"`
	YieldCoinName   currency.Code `json:"yieldCoinName"`
	EffectiveAmount types.Number  `json:"effectiveAmount"`
	PNL             types.Number  `json:"pnl"`
	// APY is formatted for display, such as "10%"
	APY       string     `json:"apy"`
	CreatedAt types.Time `json:"createdAt"`
}
