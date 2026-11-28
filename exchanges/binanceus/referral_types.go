package binanceus

import "github.com/thrasher-corp/gocryptotrader/types"

// ReferralRewardHistoryRequest holds the parameters of a referral reward history request
type ReferralRewardHistoryRequest struct {
	UserBusinessType uint64 // 0 for the rewards earned as a referrer and 1 for those earned as a referee
	Page             uint64 // Left out when zero
	Rows             uint64 // Rewards a page holds, from 1 to 200
}

// ReferralRewardHistoryResponse holds a page of referral rewards and the number of rewards matching the request
type ReferralRewardHistoryResponse struct {
	Total uint64           `json:"total"`
	Rows  []ReferralReward `json:"rows"`
}

// ReferralReward holds a referral reward
type ReferralReward struct {
	UserID          uint64       `json:"userId"`
	RewardAmount    types.Number `json:"rewardAmount"`
	ReceiveDateTime types.Time   `json:"receiveDateTime"`
	RewardType      string       `json:"rewardType"` // Such as USD
}
