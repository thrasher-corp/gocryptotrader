package gate

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// StakingCoinResponse holds an on-chain staking product
type StakingCoinResponse struct {
	ProductID           uint64                   `json:"pid"`
	ProductType         uint8                    `json:"productType"` // 0 voucher, 1 locked position, 2 US Treasury bond
	IsDeFi              types.Boolean            `json:"isDefi"`
	Currencies          currency.Currencies      `json:"currency"`
	EstimatedAPR        types.Number             `json:"estimateApr"`
	MinStakeAmount      types.Number             `json:"minStakeAmount"`
	MaxStakeAmount      types.Number             `json:"maxStakeAmount"`
	ProtocolName        string                   `json:"protocolName"`
	RedeemPeriodDays    types.Number             `json:"redeemPeriod"`
	ExchangeRate        types.Number             `json:"exchangeRate"`
	ExchangeRateReserve types.Number             `json:"exchangeRateReserve"`
	ExtraInterest       []*StakingExtraInterest  `json:"extraInterest"`
	CurrencyRewards     []*StakingCurrencyReward `json:"currencyRewards"`
}

// StakingExtraInterest holds an on-chain staking product's additional reward campaign
type StakingExtraInterest struct {
	StartTime       types.Time                `json:"start_time"`
	EndTime         types.Time                `json:"end_time"`
	RewardCoin      currency.Code             `json:"reward_coin"`
	SegmentInterest []*StakingSegmentInterest `json:"segment_interest"`
}

// StakingSegmentInterest holds a tier of an on-chain staking additional reward campaign
type StakingSegmentInterest struct {
	MoneyMin  types.Number `json:"money_min"`
	MoneyMax  types.Number `json:"money_max"`
	MoneyRate types.Number `json:"money_rate"`
}

// StakingCurrencyReward holds an on-chain staking product's reward currency
type StakingCurrencyReward struct {
	APR               types.Number  `json:"apr"`
	RewardCoin        currency.Code `json:"reward_coin"`
	RewardDelayDays   types.Number  `json:"reward_delay_days"` // -1 when dividends are only paid on redemption
	InterestDelayDays types.Number  `json:"interest_delay_days"`
}

// StakingSwapRequest holds an on-chain staking stake or redemption request
type StakingSwapRequest struct {
	Coin currency.Code `json:"coin"`
	// Side is 0 to stake and 1 to redeem; it is always sent as 0 is a valid side
	Side      uint8        `json:"side"`
	Amount    types.Number `json:"amount"`
	ProductID uint64       `json:"pid,omitempty"` // Identifies the DeFi mining protocol
}

// StakingSwapResponse holds an on-chain staking stake or redemption order
type StakingSwapResponse struct {
	ID             uint64        `json:"id"`
	ProductID      uint64        `json:"pid"`
	UserID         uint64        `json:"uid"`
	Coin           currency.Code `json:"coin"`
	Type           uint8         `json:"type"` // 0 staking, 1 redemption
	Subtype        string        `json:"subtype"`
	Amount         types.Number  `json:"amount"`
	ExchangeRate   types.Number  `json:"exchange_rate"`
	ExchangeAmount types.Number  `json:"exchange_amount"`
	UpdateTime     types.Time    `json:"updateStamp"`
	CreateTime     types.Time    `json:"createStamp"`
	Status         uint8         `json:"status"`
	ProtocolType   uint64        `json:"protocol_type"`
	ClientOrderID  string        `json:"client_order_id"`
	Source         string        `json:"source"`
}

// StakingOrderItem holds an on-chain staking order
type StakingOrderItem struct {
	ProductID      uint64        `json:"pid"`
	Coin           currency.Code `json:"coin"`
	Amount         types.Number  `json:"amount"`
	Type           uint8         `json:"type"`   // 0 staking, 1 redemption
	Status         uint8         `json:"status"` // 1 success, 3 delayed redemption in progress, 6 redemption cancelled
	RedeemTime     types.Time    `json:"redeem_stamp"`
	CreateTime     types.Time    `json:"createStamp"`
	ExchangeAmount types.Number  `json:"exchange_amount"`
	Fee            types.Number  `json:"fee"`
}

// StakingOrdersResponse holds a page of on-chain staking orders
type StakingOrdersResponse struct {
	Page       uint64              `json:"page"`
	PageSize   uint64              `json:"pageSize"`
	PageCount  uint64              `json:"pageCount"`
	TotalCount uint64              `json:"totalCount"`
	List       []*StakingOrderItem `json:"list"`
}

// StakingDividendRecord holds an on-chain staking dividend record
type StakingDividendRecord struct {
	ProductID          uint64         `json:"pid"`
	MortgageCoin       currency.Code  `json:"mortgage_coin"`
	Amount             types.Number   `json:"amount"`
	RewardCoin         currency.Code  `json:"reward_coin"`
	Interest           types.Number   `json:"interest"`
	Fee                types.Number   `json:"fee"`
	Status             uint8          `json:"status"` // 4 success
	BonusDate          types.DateTime `json:"bonus_date"`
	ScheduledBonusTime types.Time     `json:"should_bonus_stamp"`
}

// StakingDividendRecordsResponse holds a page of on-chain staking dividend records
type StakingDividendRecordsResponse struct {
	Page       uint64                   `json:"page"`
	PageSize   uint64                   `json:"pageSize"`
	PageCount  uint64                   `json:"pageCount"`
	TotalCount uint64                   `json:"totalCount"`
	List       []*StakingDividendRecord `json:"list"`
}

// StakingAssetItemResponse holds an on-chain staking position
type StakingAssetItemResponse struct {
	ProductID          uint64              `json:"pid"`
	MortgageCurrencies currency.Currencies `json:"mortgage_coin"`
	MortgageAmount     types.Number        `json:"mortgage_amount"`
	CreateTime         types.Time          `json:"createStamp"`
	ExtraIncome        types.Number        `json:"extra_income"`
	FreezeAmount       types.Number        `json:"freeze_amount"`
	MoveIncome         types.Number        `json:"move_income"`
	Type               uint8               `json:"type"`   // 0 voucher, 1 locked position, 2 US Treasury bond
	Status             uint8               `json:"status"` // 1 listed, 2 delisted
	IncomeTotal        types.Number        `json:"income_total"`
	// YesterdayIncomeMulti is kept raw because the API documents the array without describing its items
	YesterdayIncomeMulti []json.RawMessage    `json:"yesterday_income_multi"`
	RewardCoins          []*StakingRewardCoin `json:"reward_coins"`
	DeFiIncome           *StakingDeFiIncome   `json:"defi_income"`
}

// StakingRewardCoin holds an on-chain staking position's reward currency
type StakingRewardCoin struct {
	RewardCoin        currency.Code `json:"reward_coin"`
	InterestDelayDays uint64        `json:"interest_delay_days"`
	RewardDelayDays   int64         `json:"reward_delay_days"` // -1 when dividends are only paid on redemption
}

// StakingDeFiIncome holds an on-chain staking position's DeFi earnings
type StakingDeFiIncome struct {
	Total []*StakingCoinAmount `json:"total"`
}

// StakingCoinAmount holds an amount of a currency
type StakingCoinAmount struct {
	Coin   currency.Code `json:"coin"`
	Amount types.Number  `json:"amount"`
}
