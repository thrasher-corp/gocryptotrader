package bybit

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// SpotXProjectListRequest filters Launchpool, Puzzle or Token Splash projects
type SpotXProjectListRequest struct {
	// Status is required: 0 for upcoming, 1 for ongoing or 2 for ended projects
	Status uint64
	// ActivityCoin is case-insensitive; Launchpool matches it against the reward coin
	ActivityCoin currency.Code
	ProjectID    string
	Cursor       string
	// Limit is at most 10
	Limit uint64
}

// LaunchpoolProjectListResponse holds a page of Launchpool projects
type LaunchpoolProjectListResponse struct {
	List []LaunchpoolProject `json:"list"`
	// NextPageCursor is empty on the last page
	NextPageCursor string `json:"nextPageCursor"`
}

// LaunchpoolProject is a Launchpool project
type LaunchpoolProject struct {
	ProjectID    string        `json:"projectId"`
	ActivityCoin currency.Code `json:"activityCoin"`
	RewardCoin   currency.Code `json:"rewardCoin"`
	// Status is 0 upcoming, 1 ongoing or 2 ended
	Status      uint64       `json:"status"`
	TotalReward types.Number `json:"totalReward"`
	// StakeEndTime is when the stakes are redeemed automatically
	StakeStartTime types.Time `json:"stakeStartTime"`
	StakeEndTime   types.Time `json:"stakeEndTime"`
	// StakeStartTimeUTC0 and StakeEndTimeUTC0 are undocumented: the stake window as UTC date times
	StakeStartTimeUTC0    string           `json:"stakeStartTimeUtc0"`
	StakeEndTimeUTC0      string           `json:"stakeEndTimeUtc0"`
	TotalParticipantCount types.Number     `json:"totalParticipantCount"`
	Pools                 []LaunchpoolPool `json:"pools"`
}

// LaunchpoolPool is a staking pool of a Launchpool project
type LaunchpoolPool struct {
	PoolID    string        `json:"poolId"`
	StakeCoin currency.Code `json:"stakeCoin"`
	// APR is a percentage, 15.5 for 15.5%
	APR               types.Number `json:"apr"`
	TotalStakedAmount types.Number `json:"totalStakedAmount"`
	ParticipantCount  types.Number `json:"participantCount"`
}

// LaunchpoolUserActivityLogRequest filters the account's Launchpool records
type LaunchpoolUserActivityLogRequest struct {
	StakeCoin currency.Code
	// Type filters by operation, nil for all: 0 stake, 1 redeem, 2 reward, 3 automatic redemption, 4 leveraged stake, 5
	// leveraged redeem, 6 automatic leveraged redemption, 7 leveraged liquidation, 8 risk control redemption, 9
	// leveraged risk control redemption, 10 earn product reward
	Type *uint64
	// Status filters by record status, nil for all: 0 pending, 1 success or 2 failed
	Status *uint64
	// StartTime and EndTime must be set together
	StartTime time.Time
	EndTime   time.Time
	// PageSize is at most 10
	PageSize uint64
	// Current is the page number from 1 to 100
	Current uint64
}

// LaunchpoolUserActivityLogResponse holds a page of the account's Launchpool records
type LaunchpoolUserActivityLogResponse struct {
	List     []LaunchpoolActivity `json:"list"`
	Total    uint64               `json:"total"`
	PageSize uint64               `json:"pageSize"`
	Current  uint64               `json:"current"`
}

// LaunchpoolActivity is a Launchpool stake, redeem or reward record
type LaunchpoolActivity struct {
	StakeCoin  currency.Code `json:"stakeCoin"`
	RewardCoin currency.Code `json:"rewardCoin"`
	Amount     types.Number  `json:"amount"`
	// Type and Status take the values of the LaunchpoolUserActivityLogRequest filters
	Type        uint64     `json:"type"`
	Status      uint64     `json:"status"`
	CreatedTime types.Time `json:"createdTime"`
	OrderID     string     `json:"orderId"`
}

// LaunchpoolCurrentStakingResponse holds the account's active Launchpool positions and their USD totals
type LaunchpoolCurrentStakingResponse struct {
	TotalInvestmentUSD types.Number                `json:"totalInvestmentUsd"`
	TotalEarningsUSD   types.Number                `json:"totalEarningsUsd"`
	TodayEarningsUSD   types.Number                `json:"todayEarningsUsd"`
	List               []LaunchpoolStakingPosition `json:"list"`
}

// LaunchpoolStakingPosition is an active Launchpool position
type LaunchpoolStakingPosition struct {
	StakeCoin   currency.Code `json:"stakeCoin"`
	RewardCoin  currency.Code `json:"rewardCoin"`
	StakeAmount types.Number  `json:"stakeAmount"`
	TotalReward types.Number  `json:"totalReward"`
	// AutoRedeemDate is when the principal returns automatically; zero when unset
	AutoRedeemDate types.Time `json:"autoRedeemDate"`
}

// LaunchpoolUserHistoryRequest filters the account's ended Launchpool positions
type LaunchpoolUserHistoryRequest struct {
	StakeCoin  currency.Code
	RewardCoin currency.Code
	// StartTime and EndTime must be set together; they bound the positions' staking periods
	StartTime time.Time
	EndTime   time.Time
	// PageSize is at most 10
	PageSize uint64
	// Current is the page number from 1 to 100
	Current uint64
}

// LaunchpoolUserHistoryResponse holds a page of the account's ended Launchpool positions
type LaunchpoolUserHistoryResponse struct {
	List     []LaunchpoolHistoricalPosition `json:"list"`
	Total    uint64                         `json:"total"`
	PageSize uint64                         `json:"pageSize"`
	Current  uint64                         `json:"current"`
}

// LaunchpoolHistoricalPosition is an ended Launchpool position
type LaunchpoolHistoricalPosition struct {
	StakeCoin   currency.Code `json:"stakeCoin"`
	RewardCoin  currency.Code `json:"rewardCoin"`
	TotalReward types.Number  `json:"totalReward"`
	// StakeEndTime is when the stake was redeemed automatically
	StakeStartTime types.Time `json:"stakeStartTime"`
	StakeEndTime   types.Time `json:"stakeEndTime"`
}

// PuzzleProjectListResponse holds a page of Puzzle projects
type PuzzleProjectListResponse struct {
	List []PuzzleProject `json:"list"`
	// NextPageCursor is empty on the last page
	NextPageCursor string `json:"nextPageCursor"`
}

// PuzzleProject is a Puzzle project
type PuzzleProject struct {
	ProjectID    string        `json:"projectId"`
	ActivityCoin currency.Code `json:"activityCoin"`
	RewardCoin   currency.Code `json:"rewardCoin"`
	// Status is 0 upcoming, 1 ongoing or 2 ended
	Status    uint64     `json:"status"`
	StartTime types.Time `json:"startTime"`
	EndTime   types.Time `json:"endTime"`
	// StartTimeUTC0 and EndTimeUTC0 are undocumented: the project window as UTC date times
	StartTimeUTC0    string       `json:"startTimeUtc0"`
	EndTimeUTC0      string       `json:"endTimeUtc0"`
	TotalReward      types.Number `json:"totalReward"`
	ParticipantCount types.Number `json:"participantCount"`
}

// TokenSplashProjectListResponse holds a page of Token Splash projects
type TokenSplashProjectListResponse struct {
	List []TokenSplashProject `json:"list"`
	// NextPageCursor is empty on the last page
	NextPageCursor string `json:"nextPageCursor"`
}

// TokenSplashProject is a Token Splash project
type TokenSplashProject struct {
	// Status is 0 upcoming, 1 ongoing or 2 ended
	Status       uint64        `json:"status"`
	ProjectID    string        `json:"projectId"`
	ActivityCoin currency.Code `json:"activityCoin"`
	// RewardCoin is the new user pool's token, else the old user pool's, else the trade pool's
	RewardCoin currency.Code `json:"rewardCoin"`
	// TotalReward is the sum of the new user, old user and trade prize pools
	TotalReward      types.Number `json:"totalReward"`
	ParticipantCount types.Number `json:"participantCount"`
	// RegistrationStartTime is the earliest non-zero registration start and ActivityEndTime the latest announcement
	RegistrationStartTime types.Time `json:"registrationStartTime"`
	ActivityEndTime       types.Time `json:"activityEndTime"`
	// RegistrationStartTimeUTC0 and ActivityEndTimeUTC0 are undocumented: the same times as UTC date times
	RegistrationStartTimeUTC0 string `json:"registrationStartTimeUtc0"`
	ActivityEndTimeUTC0       string `json:"activityEndTimeUtc0"`
}

// TokenSplashUserActivityParamsResponse holds the account's trade task progress in Token Splash projects
type TokenSplashUserActivityParamsResponse struct {
	List []TokenSplashUserActivity `json:"list"`
}

// TokenSplashUserActivity is the account's participation in a Token Splash project
type TokenSplashUserActivity struct {
	ProjectID    string               `json:"projectId"`
	ActivityCoin currency.Code        `json:"activityCoin"`
	TradeTask    TokenSplashTradeTask `json:"tradeTask"`
}

// TokenSplashTradeTask is a Token Splash trade task and the account's progress in it
type TokenSplashTradeTask struct {
	TradeRequiredAmount types.Number  `json:"tradeRequiredAmount"`
	TradeUnit           currency.Code `json:"tradeUnit"`
	// TradeToken is the activity token ID for on-chain trade tasks and the configured trade token otherwise
	TradeToken      string       `json:"tradeToken"`
	TradedAmount    types.Number `json:"tradedAmount"`
	MaxRewardAmount types.Number `json:"maxRewardAmount"`
	// EstimatedRewardAmount is min(TradedAmount / TradeRequiredAmount, 1) * MaxRewardAmount, truncated to 4 decimals
	EstimatedRewardAmount types.Number  `json:"estimatedRewardAmount"`
	RewardCoin            currency.Code `json:"rewardCoin"`
}
