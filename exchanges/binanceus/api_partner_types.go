package binanceus

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// UserEligibilityResponse holds whether the user is eligible for an API partner's rebates
type UserEligibilityResponse struct {
	IsEligible bool   `json:"isEligible"`
	IsNewUser  bool   `json:"ifNewUser"`
	VIPLevel   uint64 `json:"vipLevel"`
	ReferrerID uint64 `json:"referrerId"`
}

// RebateHistoryRequest holds the parameters of a rebate history query
type RebateHistoryRequest struct {
	PartnerID string    // The API partner's 8 character ID
	StartTime time.Time // 7 days before EndTime when zero
	EndTime   time.Time // Now when zero
	Limit     uint64    // 100 when zero, at most 1000
	Page      uint64
}

// RebateRecordResponse holds a rebate paid to the user
type RebateRecordResponse struct {
	Quantity types.Number  `json:"qty"`
	Asset    currency.Code `json:"asset"`
	Date     types.Time    `json:"date"`
}
