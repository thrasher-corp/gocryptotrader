package gateio

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// StakingCoin holds an on-chain staking coin product detail
type StakingCoin struct {
	Pid            int64        `json:"pid"`
	ProductType    int64        `json:"productType"`
	IsDeFi         int64        `json:"isDeFi"`
	Currency       string       `json:"currency"`
	EstimatedAPR   string       `json:"estimatedApr"`
	MinStakeAmount types.Number `json:"minStakeAmount"`
	MaxStakeAmount types.Number `json:"maxStakeAmount"`
	ProtocolName   string       `json:"protocolName"`
	RedeemPeriod   int64        `json:"redeemPeriod"`
	ExchangeRate   types.Number `json:"exchangeRate"`
}

// StakingSwapRequest holds an on-chain token swap request for earned coins
type StakingSwapRequest struct {
	Coin   string       `json:"coin"`
	Side   int64        `json:"side"`
	Amount types.Number `json:"amount"`
	Pid    int64        `json:"pid,omitempty"`
}

// StakingSwapResponse holds the response for an on-chain staking swap
type StakingSwapResponse struct {
	ID              int64        `json:"id"`
	Pid             int64        `json:"pid"`
	Coin            string       `json:"coin"`
	UID             int64        `json:"uid"`
	Type            int64        `json:"type"`
	Subtype         int64        `json:"subtype"`
	Amount          types.Number `json:"amount"`
	ExchangeRate    types.Number `json:"exchange_rate"`
	ExchangeAmount  types.Number `json:"exchange_amount"`
	UpdateTimestamp types.Time   `json:"updateTimestamp"`
}

// StakingOrderItem holds an on-chain staking order item
type StakingOrderItem struct {
	Pid    int64        `json:"pid"`
	Coin   string       `json:"coin"`
	Amount types.Number `json:"amount"`
	Type   int64        `json:"type"`
	Status int64        `json:"status"`
}

// StakingOrdersResponse holds the paginated response for on-chain staking orders
type StakingOrdersResponse struct {
	Page       int64               `json:"page"`
	PageSize   int64               `json:"pageSize"`
	PageCount  int64               `json:"pageCount"`
	TotalCount int64               `json:"totalCount"`
	List       []*StakingOrderItem `json:"list"`
}

// StakingDividendRecord holds an on-chain staking dividend record item
type StakingDividendRecord struct {
	PID          int64         `json:"pid"`
	MortgageCoin currency.Code `json:"mortgage_coin"`
	Amount       types.Number  `json:"amount"`
	RewardCoin   currency.Code `json:"reward_coin"`
	Interest     types.Number  `json:"interest"`
}

// StakingDividendRecordsResponse holds the paginated response for staking dividend records
type StakingDividendRecordsResponse struct {
	Page       int64                    `json:"page"`
	PageSize   int64                    `json:"pageSize"`
	PageCount  int64                    `json:"pageCount"`
	TotalCount int64                    `json:"totalCount"`
	List       []*StakingDividendRecord `json:"list"`
}

// StakingAssetItem holds an on-chain staking asset item
type StakingAssetItem struct {
	Pid            int64        `json:"pid"`
	MortgageCoin   string       `json:"mortgage_coin"`
	MortgageAmount types.Number `json:"mortgage_amount"`
	CreateStamp    types.Time   `json:"createStamp"`
	ExtraIncome    string       `json:"extra_income"`
	FreezeAmount   types.Number `json:"freeze_amount"`
	MoveIncome     types.Number `json:"move_income"`
	Type           int64        `json:"type"`
	Status         int64        `json:"status"`
	IncomeTotal    types.Number `json:"income_total"`
}
