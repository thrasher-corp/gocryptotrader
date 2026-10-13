package bybit

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// RWAProductListResponse holds real world asset products
type RWAProductListResponse struct {
	List []RWAProduct `json:"list"`
}

// RWAProduct is a real world asset product
type RWAProduct struct {
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
	// Duration is the lock-up in days, 0 for Flexible products
	Duration uint64 `json:"duration"`
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

// PlaceRWAOrderRequest is a real world asset stake or redeem order
type PlaceRWAOrderRequest struct {
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

// PlaceRWAOrderResponse identifies an accepted real world asset order, which settles asynchronously
type PlaceRWAOrderResponse struct {
	OrderID     string `json:"orderId"`
	OrderLinkID string `json:"orderLinkId"`
}

// RWAPositionListResponse holds real world asset positions
type RWAPositionListResponse struct {
	List []RWAPosition `json:"list"`
}

// RWAPosition is a real world asset position
type RWAPosition struct {
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
	// Duration is the lock-up in days, 0 for Flexible products
	Duration uint64 `json:"duration"`
}

// RWAOrderListRequest filters real world asset orders. An OrderID, or else an OrderLinkID, selects one order and
// ignores the other filters; the window defaults to the last 7 days and reaches back at most 180 days
type RWAOrderListRequest struct {
	OrderID     string
	OrderLinkID string
	OrderType   string
	ProductID   uint64
	StartTime   time.Time
	EndTime     time.Time
	Limit       uint64
	Cursor      string
}

// RWAOrderListResponse holds a page of real world asset orders
type RWAOrderListResponse struct {
	List           []RWAOrder `json:"list"`
	NextPageCursor string     `json:"nextPageCursor"`
}

// RWAOrder is a real world asset stake or redeem order
type RWAOrder struct {
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

// RWANAVChartRequest selects a real world asset product's NAV history. The window defaults to the 7 days before the
// end time, which defaults to now, and may span at most 180 days
type RWANAVChartRequest struct {
	ProductID uint64
	StartTime time.Time
	EndTime   time.Time
}

// RWANAVChartResponse holds a real world asset product's daily NAV, oldest first
type RWANAVChartResponse struct {
	// ProductID is documented as an integer, but the API sends it quoted
	ProductID types.Number `json:"productId"`
	List      []RWANAV     `json:"list"`
}

// RWANAV is a real world asset product's NAV on a day
type RWANAV struct {
	// Date is YYYY-MM-DD in UTC
	Date string       `json:"date"`
	NAV  types.Number `json:"nav"`
}
