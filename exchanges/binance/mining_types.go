package binance

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// MiningResponse is the envelope of every mining endpoint. The mining methods return only Data: a successful response
// always carries code 0 and an empty message, and an error response a negative code, which the transport reports
type MiningResponse[T any] struct {
	Code    int64  `json:"code"` // Signed because Binance error codes are negative
	Message string `json:"msg"`
	Data    T      `json:"data"`
}

// MiningAccountHashrate is a mining account's hashrate history in one bucket type
type MiningAccountHashrate struct {
	// Type is the hashrate bucket, such as H_hashrate (hourly) or D_hashrate (daily)
	Type     string           `json:"type"`
	UserName string           `json:"userName"`
	List     []MiningHashrate `json:"list"`
}

// MiningHashrate is a mining account's hashrate and rejection rate at a time
type MiningHashrate struct {
	Time     types.Time   `json:"time"`
	Hashrate types.Number `json:"hashrate"`
	Reject   types.Number `json:"reject"`
}

// MiningAlgorithm is a mining algorithm
type MiningAlgorithm struct {
	AlgorithmName string `json:"algoName"`
	AlgorithmID   uint64 `json:"algoId"`
	PoolIndex     uint64 `json:"poolIndex"`
	Unit          string `json:"unit"`
}

// MiningCoin is a mineable coin and its algorithm
type MiningCoin struct {
	CoinName      currency.Code `json:"coinName"`
	CoinID        uint64        `json:"coinId"`
	PoolIndex     uint64        `json:"poolIndex"`
	AlgorithmID   uint64        `json:"algoId"`
	AlgorithmName string        `json:"algoName"`
}

// MiningPaymentRequest holds the parameters of Earnings List and Extra Bonus List
type MiningPaymentRequest struct {
	Algorithm string
	UserName  string
	Coin      currency.Code
	StartDate time.Time
	EndDate   time.Time
	PageIndex uint64
	// PageSize is between 10 and 200
	PageSize uint64
}

// MiningEarningsResponse is a page of a mining account's earnings
type MiningEarningsResponse struct {
	AccountProfits []MiningEarning `json:"accountProfits"`
	TotalNumber    uint64          `json:"totalNum"`
	PageSize       uint64          `json:"pageSize"`
}

// MiningEarning is a mining account's earning for a day
type MiningEarning struct {
	Time types.Time `json:"time"`
	// Type is 0 mining wallet, 5 mining address, 7 pool savings, 8 transferred, 31 income transfer, 32 hashrate resale
	// to mining wallet or 33 hashrate resale to pool savings
	Type           uint64        `json:"type"`
	HashTransfer   float64       `json:"hashTransfer"`
	TransferAmount float64       `json:"transferAmount"`
	DayHashRate    float64       `json:"dayHashRate"`
	ProfitAmount   float64       `json:"profitAmount"`
	CoinName       currency.Code `json:"coinName"`
	// Status is 0 unpaid, 1 paying or 2 paid
	Status uint64 `json:"status"`
}

// MiningExtraBonusResponse is a page of a mining account's extra bonuses
type MiningExtraBonusResponse struct {
	OtherProfits []MiningExtraBonus `json:"otherProfits"`
	TotalNumber  uint64             `json:"totalNum"`
	PageSize     uint64             `json:"pageSize"`
}

// MiningExtraBonus is a mining account's extra bonus for a day
type MiningExtraBonus struct {
	Time     types.Time    `json:"time"`
	CoinName currency.Code `json:"coinName"`
	// Type is 1 merged mining, 2 activity bonus, 3 rebate, 4 smart pool, 6 income transfer or 7 pool savings
	Type         uint64  `json:"type"`
	ProfitAmount float64 `json:"profitAmount"`
	// Status is 0 unpaid, 1 paying or 2 paid
	Status uint64 `json:"status"`
}

// HashrateResaleDetailResponse is a page of the income a hashrate resale configuration transferred
type HashrateResaleDetailResponse struct {
	ProfitTransferDetails []HashrateResaleProfitTransfer `json:"profitTransferDetails"`
	TotalNumber           uint64                         `json:"totalNum"`
	PageSize              uint64                         `json:"pageSize"`
}

// HashrateResaleProfitTransfer is the income a hashrate resale transferred on a day
type HashrateResaleProfitTransfer struct {
	PoolUsername   string        `json:"poolUsername"`
	ToPoolUsername string        `json:"toPoolUsername"`
	AlgorithmName  string        `json:"algoName"`
	HashRate       float64       `json:"hashRate"`
	Day            types.Time    `json:"day"` // Sent as a yyyymmdd date
	Amount         float64       `json:"amount"`
	CoinName       currency.Code `json:"coinName"`
}

// HashrateResaleListResponse is a page of hashrate resale configurations
type HashrateResaleListResponse struct {
	ConfigDetails []HashrateResaleConfig `json:"configDetails"`
	TotalNumber   uint64                 `json:"totalNum"`
	PageSize      uint64                 `json:"pageSize"`
}

// HashrateResaleConfig is a hashrate resale configuration
type HashrateResaleConfig struct {
	ConfigID       uint64     `json:"configId"`
	PoolUsername   string     `json:"poolUsername"`
	ToPoolUsername string     `json:"toPoolUsername"`
	AlgorithmName  string     `json:"algoName"`
	HashRate       float64    `json:"hashRate"`
	StartDay       types.Time `json:"startDay"` // Sent as a yyyymmdd date
	EndDay         types.Time `json:"endDay"`   // Sent as a yyyymmdd date
	// Status is 0 processing, 1 cancelled or 2 terminated
	Status uint64 `json:"status"`
	// Type is 0 for a transferred and 1 for a received hashrate
	Type uint64 `json:"type"`
}

// HashrateResaleRequest holds the parameters of a hashrate resale
type HashrateResaleRequest struct {
	UserName   string
	Algorithm  string
	StartDate  time.Time
	EndDate    time.Time
	ToPoolUser string
	// HashRate is in h/s; it must exceed 500000000000 for BTC and 500000 for ETH
	HashRate uint64
}

// MiningAccountEarningRequest holds the parameters of Mining Account Earning
type MiningAccountEarningRequest struct {
	Algorithm string
	StartDate time.Time
	EndDate   time.Time
	PageIndex uint64
	// PageSize is between 10 and 200
	PageSize uint64
}

// MiningAccountEarningResponse is a page of a mining account's referral, refund and rebate earnings
type MiningAccountEarningResponse struct {
	AccountProfits []MiningAccountProfit `json:"accountProfits"`
	TotalNumber    uint64                `json:"totalNum"`
	PageSize       uint64                `json:"pageSize"`
}

// MiningAccountProfit is a mining account's referral, refund or rebate earning
type MiningAccountProfit struct {
	Time     types.Time    `json:"time"`
	CoinName currency.Code `json:"coinName"`
	// Type is 0 referral, 1 refund or 2 rebate
	Type    uint64  `json:"type"`
	PUID    uint64  `json:"puid"`    // Mining sub-account ID
	SubName string  `json:"subName"` // Mining account
	Amount  float64 `json:"amount"`
}

// MinerDetail is a miner's hashrate history in one bucket type
type MinerDetail struct {
	WorkerName string `json:"workerName"`
	// Type is the hashrate bucket, such as H_hashrate (hourly) or D_hashrate (daily)
	Type          string          `json:"type"`
	HashrateDatas []MinerHashrate `json:"hashrateDatas"`
}

// MinerHashrate is a miner's hashrate and rejection rate at a time
type MinerHashrate struct {
	Time     types.Time   `json:"time"`
	Hashrate types.Number `json:"hashrate"`
	Reject   float64      `json:"reject"`
}

// MinerListRequest holds the parameters of Request for Miner List
type MinerListRequest struct {
	Algorithm string
	UserName  string
	PageIndex uint64
	// Descending sorts in descending order instead of the default ascending order
	Descending bool
	// SortColumn is 1 miner name (default), 2 real-time hashrate, 3 daily average hashrate, 4 real-time rejection rate
	// or 5 last submission time
	SortColumn uint64
	// WorkerStatus is 0 all (default), 1 valid, 2 invalid or 3 failure
	WorkerStatus uint64
}

// MinerListResponse is a page of a mining account's miners
type MinerListResponse struct {
	WorkerDatas []MinerWorker `json:"workerDatas"`
	TotalNumber uint64        `json:"totalNum"`
	PageSize    uint64        `json:"pageSize"`
}

// MinerWorker is a miner's status and hashrate
type MinerWorker struct {
	WorkerID   string `json:"workerId"`
	WorkerName string `json:"workerName"`
	// Status is 1 valid, 2 invalid or 3 no longer valid
	Status        uint64     `json:"status"`
	HashRate      float64    `json:"hashRate"`
	DayHashRate   float64    `json:"dayHashRate"`
	RejectRate    float64    `json:"rejectRate"`
	LastShareTime types.Time `json:"lastShareTime"`
}

// MiningStatisticsResponse is a mining account's hashrate and earnings summary
type MiningStatisticsResponse struct {
	FifteenMinuteHashRate types.Number                   `json:"fifteenMinHashRate"`
	DayHashRate           types.Number                   `json:"dayHashRate"`
	ValidNumber           uint64                         `json:"validNum"`
	InvalidNumber         uint64                         `json:"invalidNum"`
	ProfitToday           map[currency.Code]types.Number `json:"profitToday"`
	ProfitYesterday       map[currency.Code]types.Number `json:"profitYesterday"`
	UserName              string                         `json:"userName"`
	Unit                  string                         `json:"unit"`
	Algorithm             string                         `json:"algo"`
}
