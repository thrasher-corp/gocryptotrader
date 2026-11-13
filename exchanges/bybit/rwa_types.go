package bybit

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// RealWorldAssetProductListResponse holds real world asset products
type RealWorldAssetProductListResponse struct {
	List []RealWorldAssetProduct `json:"list"`
}

// RealWorldAssetProduct is a real world asset product
type RealWorldAssetProduct struct {
	// ProductID is documented as an integer, but the API sends it quoted
	ProductID types.Number `json:"productId"`
	// Coin is the settlement coin
	Coin currency.Code `json:"coin"`
	// AssetSymbol is the underlying asset, which also names it
	AssetSymbol string `json:"assetSymbol"`
	Manager     string `json:"manager"`
	// BaseAPR and BonusAPR are decimals, 0.055 for 5.5%; BonusAPR is empty without an active bonus
	BaseAPR    types.Number `json:"baseApr"`
	BonusAPR   types.Number `json:"bonusApr"`
	SavingType string       `json:"savingType"`
	// DurationInDays is the lock-up, 0 for Flexible products
	DurationInDays uint64 `json:"duration"`
	// NAV is the latest net asset value per share
	NAV types.Number `json:"nav"`
	// MinStakeAmount and MaxStakeAmount are in the settlement coin; MaxStakeAmount, UserMaxAmount and UserQuota are
	// empty when unlimited, and UserQuota is also empty for unsigned requests
	MinStakeAmount types.Number `json:"minStakeAmount"`
	MaxStakeAmount types.Number `json:"maxStakeAmount"`
	UserMaxAmount  types.Number `json:"userMaxAmount"`
	UserQuota      types.Number `json:"userQuota"`
	MinRedeemShare types.Number `json:"minRedeemShare"`
	// RedeemFeeRate and SubscriptionFee are decimals, 0.001 for 0.1%
	RedeemFeeRate   types.Number `json:"redeemFeeRate"`
	SubscriptionFee types.Number `json:"subscriptionFee"`
	ExternalLink    string       `json:"extLink"`
	// AmountPrecision and SharePrecision are the decimal places of settlement coin amounts and shares
	AmountPrecision uint64 `json:"amountPrecision"`
	SharePrecision  uint64 `json:"sharePrecision"`
	// RedeemDurationMinutes is undocumented: the redemption settlement time in minutes
	RedeemDurationMinutes uint64 `json:"redeem_duration_mins"`
}

// PlaceRealWorldAssetOrderRequest is a real world asset stake or redeem order
type PlaceRealWorldAssetOrderRequest struct {
	ProductID uint64
	OrderType string
	// Coin is the settlement coin
	Coin currency.Code
	// OrderLinkID is unique per user; up to 36 letters, digits, hyphens and underscores
	OrderLinkID string
	// StakeAmount is required for Stake orders, in the settlement coin
	StakeAmount float64
	// RedeemShares is required for Redeem orders
	RedeemShares float64
	AccountType  string
}

// PlaceRealWorldAssetOrderResponse identifies an accepted real world asset order, which settles asynchronously
type PlaceRealWorldAssetOrderResponse struct {
	OrderID     string `json:"orderId"`
	OrderLinkID string `json:"orderLinkId"`
}

// RealWorldAssetPositionListResponse holds real world asset positions
type RealWorldAssetPositionListResponse struct {
	List []RealWorldAssetPosition `json:"list"`
}

// RealWorldAssetPosition is a real world asset position
type RealWorldAssetPosition struct {
	// ProductID is documented as an integer, but the product endpoints send it quoted
	ProductID   types.Number  `json:"productId"`
	Coin        currency.Code `json:"coin"`
	AssetSymbol string        `json:"assetSymbol"`
	// EffectiveShare is the redeemable share quantity
	EffectiveShare types.Number `json:"effectiveShare"`
	// ProcessingStakeAmount and ProcessingRedeemShare await settlement
	ProcessingStakeAmount types.Number `json:"processingStakeAmount"`
	ProcessingRedeemShare types.Number `json:"processingRedeemShare"`
	BonusEarned           types.Number `json:"bonusEarned"`
	NAV                   types.Number `json:"nav"`
	// HoldAmount is EffectiveShare times NAV, in the settlement coin
	HoldAmount types.Number `json:"holdAmount"`
	// DurationInDays is the lock-up, 0 for Flexible products
	DurationInDays uint64 `json:"duration"`
}

// RealWorldAssetOrderListRequest filters real world asset orders. An OrderID, or else an OrderLinkID, selects one order
// and ignores the other filters; the window defaults to the last 7 days and reaches back at most 180 days
type RealWorldAssetOrderListRequest struct {
	OrderID     string
	OrderLinkID string
	OrderType   string
	ProductID   uint64
	StartTime   time.Time
	EndTime     time.Time
	Limit       uint64
	Cursor      string
}

// RealWorldAssetOrderListResponse holds a page of real world asset orders
type RealWorldAssetOrderListResponse struct {
	List           []RealWorldAssetOrder `json:"list"`
	NextPageCursor string                `json:"nextPageCursor"`
}

// RealWorldAssetOrder is a real world asset stake or redeem order
type RealWorldAssetOrder struct {
	OrderID     string `json:"orderId"`
	OrderLinkID string `json:"orderLinkId"`
	OrderType   string `json:"orderType"`
	// ProductID is documented as an integer, but the product endpoints send it quoted
	ProductID types.Number  `json:"productId"`
	Coin      currency.Code `json:"coin"`
	// StakeAmount is only set for Stake orders and RedeemShares for Redeem orders
	StakeAmount  types.Number `json:"stakeAmount"`
	RedeemShares types.Number `json:"redeemShares"`
	Status       string       `json:"status"`
	AccountType  string       `json:"accountType"`
	CreatedTime  types.Time   `json:"createdTime"`
	UpdatedTime  types.Time   `json:"updatedTime"`
	// SettledShares is only set for successful Stake orders and SettledAmount for successful Redeem orders
	SettledShares types.Number `json:"settledShares"`
	SettledAmount types.Number `json:"settledAmount"`
}

// RealWorldAssetNAVChartRequest selects a real world asset product's NAV history. The window defaults to the 7 days
// before the end time, which defaults to now, and may span at most 180 days
type RealWorldAssetNAVChartRequest struct {
	ProductID uint64
	StartTime time.Time
	EndTime   time.Time
}

// RealWorldAssetNAVChartResponse holds a real world asset product's daily NAV, oldest first
type RealWorldAssetNAVChartResponse struct {
	// ProductID is documented as an integer, but the API sends it quoted
	ProductID types.Number        `json:"productId"`
	List      []RealWorldAssetNAV `json:"list"`
}

// RealWorldAssetNAV is a real world asset product's NAV on a day
type RealWorldAssetNAV struct {
	// Date is YYYY-MM-DD in UTC
	Date string       `json:"date"`
	NAV  types.Number `json:"nav"`
}
