package mexc

import (
	"bytes"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Futures position types
const (
	FuturesPositionTypeLong  = 1
	FuturesPositionTypeShort = 2
)

// Futures open types, which are the margin modes
const (
	FuturesOpenTypeIsolated = 1
	FuturesOpenTypeCross    = 2
)

// Futures position modes
const (
	FuturesPositionModeHedge  = 1
	FuturesPositionModeOneWay = 2
)

// Futures position states
const (
	FuturesPositionStateHolding    = 1
	FuturesPositionStateSystemHeld = 2
	FuturesPositionStateClosed     = 3
)

// Futures position margin changes
const (
	FuturesMarginAdd      = "ADD"
	FuturesMarginSubtract = "SUB"
)

// Futures profit rate periods
const (
	FuturesProfitRateDay  = 1
	FuturesProfitRateWeek = 2
)

// FuturesAccountAssetResponse holds a currency's futures account balances
type FuturesAccountAssetResponse struct {
	Currency         currency.Code `json:"currency"`
	PositionMargin   float64       `json:"positionMargin"`
	FrozenBalance    float64       `json:"frozenBalance"`
	AvailableBalance float64       `json:"availableBalance"`
	// CashBalance is the withdrawable balance
	CashBalance     float64    `json:"cashBalance"`
	Equity          float64    `json:"equity"`
	Unrealised      float64    `json:"unrealized"`
	Bonus           float64    `json:"bonus"`
	BonusExpireTime types.Time `json:"bonusExpireTime"`
	// AvailableCash is the transferable amount and AvailableOpen the amount usable to open positions
	AvailableCash          float64 `json:"availableCash"`
	AvailableOpen          float64 `json:"availableOpen"`
	DebtAmount             float64 `json:"debtAmount"`
	ContributeMarginAmount float64 `json:"contributeMarginAmount"`
	CurrencyID             string  `json:"vcoinId"`
}

// FuturesTransferRecordsRequest holds the parameters of a futures transfer records request
type FuturesTransferRecordsRequest struct {
	Currency currency.Code
	// State is WAIT, SUCCESS or FAILED
	State string
	// Type is IN or OUT
	Type string
	FuturesPageRequest
}

// FuturesTransferRecordResponse is a transfer to or from the futures account
type FuturesTransferRecordResponse struct {
	ID            uint64        `json:"id"`
	TransactionID string        `json:"txid"`
	Currency      currency.Code `json:"currency"`
	Amount        float64       `json:"amount"`
	Type          string        `json:"type"`
	State         string        `json:"state"`
	CreateTime    types.Time    `json:"createTime"`
	UpdateTime    types.Time    `json:"updateTime"`
}

// FuturesProfitRateResponse holds the account's profit rate over a day or a week
type FuturesProfitRateResponse struct {
	Ranking       uint64     `json:"ranking"`
	ProfitRate    float64    `json:"profitRate"`
	StatisticTime types.Time `json:"statisticTime"`
}

// FuturesFeeDeductionConfigResponse is a fee deduction configuration
type FuturesFeeDeductionConfigResponse struct {
	DeductCoin    currency.Code `json:"deductCoin"`
	SettleCoin    currency.Code `json:"settleCoin"`
	DiscountRatio float64       `json:"discountRatio"`
}

// FuturesDiscountConfigResponse is a spot discount configuration for futures fees
type FuturesDiscountConfigResponse struct {
	ID                        uint64               `json:"id"`
	Currency                  currency.Code        `json:"currency"`
	State                     string               `json:"state"`
	DiscountConfig            FuturesDiscountTiers `json:"discountConfig"`
	IneffectiveDiscountConfig FuturesDiscountTiers `json:"ineffectiveDiscountConfig"`
	Operator                  string               `json:"operator"`
	CreateTime                types.Time           `json:"createTime"`
	UpdateTime                types.Time           `json:"updateTime"`
	Enabled                   bool                 `json:"enabled"`
	BlackContractIDSet        []uint64             `json:"blackContractIdSet"`
}

// FuturesDiscountTier is a band of a fee discount configuration
type FuturesDiscountTier struct {
	Level     uint64  `json:"level"`
	VolumeMin float64 `json:"volMin"`
	VolumeMax float64 `json:"volMax"`
	Day       uint64  `json:"day"`
	Discount  float64 `json:"discount"`
}

// FuturesDiscountTiers holds discount bands, which the venue sends as JSON within a JSON string
type FuturesDiscountTiers []FuturesDiscountTier

// UnmarshalJSON decodes discount bands from a JSON string holding them, or from a JSON list
func (f *FuturesDiscountTiers) UnmarshalJSON(data []byte) error {
	if len(data) > 0 && data[0] == '"' {
		var encoded string
		if err := json.Unmarshal(data, &encoded); err != nil {
			return err
		}
		data = []byte(encoded)
	}
	if data = bytes.TrimSpace(data); len(data) == 0 || bytes.Equal(data, []byte("null")) {
		*f = nil
		return nil
	}
	var tiers []FuturesDiscountTier
	if err := json.Unmarshal(data, &tiers); err != nil {
		return err
	}
	*f = tiers
	return nil
}

// FuturesFeeDeductionDetailsRequest holds the parameters of a futures fee deduction details request
type FuturesFeeDeductionDetailsRequest struct {
	Symbol currency.Pair
	// DealIDs selects up to 20 deals
	DealIDs []string
	// StartTime defaults to 7 days before now; the window is at most 90 days
	StartTime time.Time
	EndTime   time.Time
	FuturesPageRequest
}

// FuturesFeeDeductionDetailResponse holds a deal's fee and its deduction
type FuturesFeeDeductionDetailResponse struct {
	ID          FuturesID     `json:"id"`
	Symbol      string        `json:"symbol"`
	OrderID     FuturesID     `json:"orderId"`
	FeeCurrency currency.Code `json:"feeCurrency"`
	Fee         float64       `json:"fee"`
	// DeductType is 0 for no deduction, 1 for an MXPOINT deduction and 2 for a trial bonus deduction; it is
	// documented as a string and sent as a number
	DeductType        types.Number  `json:"deductType"`
	DeductFeeCurrency currency.Code `json:"deductFeeCurrency"`
	DeductFee         float64       `json:"deductFee"`
	Timestamp         types.Time    `json:"timestamp"`
}

// FuturesDiscountTypeResponse reports which fee discounts the account uses
type FuturesDiscountTypeResponse struct {
	UseFeeDiscount bool `json:"useFeeDiscount"`
	UseFeeDeduct   bool `json:"useFeeDeduct"`
}

// FuturesFeeStatisticsResponse holds the fees paid and saved over the last 30 days
type FuturesFeeStatisticsResponse struct {
	TotalFee float64 `json:"totalFee"`
	SaveFee  float64 `json:"saveFee"`
}

// FuturesPositionsRequest holds the parameters of a historical positions or funding fees request
type FuturesPositionsRequest struct {
	Symbol       currency.Pair
	PositionID   string
	PositionType uint8
	StartTime    time.Time
	EndTime      time.Time
	FuturesPageRequest
}

// FuturesPositionResponse holds a futures position. Volumes are in contracts
type FuturesPositionResponse struct {
	PositionID            FuturesID          `json:"positionId"`
	Symbol                string             `json:"symbol"`
	HoldVolume            float64            `json:"holdVol"`
	PositionType          uint8              `json:"positionType"`
	OpenType              uint8              `json:"openType"`
	State                 uint8              `json:"state"`
	FrozenVolume          float64            `json:"frozenVol"`
	CloseVolume           float64            `json:"closeVol"`
	HoldAveragePrice      float64            `json:"holdAvgPrice"`
	HoldAveragePriceExact types.Number       `json:"holdAvgPriceFullyScale"`
	CloseAveragePrice     float64            `json:"closeAvgPrice"`
	OpenAveragePrice      float64            `json:"openAvgPrice"`
	OpenAveragePriceExact types.Number       `json:"openAvgPriceFullyScale"`
	LiquidatePrice        float64            `json:"liquidatePrice"`
	OriginalInitialMargin float64            `json:"oim"`
	InitialMargin         float64            `json:"im"`
	AutoDeleverageLevel   uint8              `json:"adlLevel"`
	HoldFee               float64            `json:"holdFee"`
	RealisedPNL           float64            `json:"realised"`
	Leverage              uint64             `json:"leverage"`
	MarginRatio           float64            `json:"marginRatio"`
	AutoAddInitialMargin  bool               `json:"autoAddIm"`
	ProfitRatio           float64            `json:"profitRatio"`
	NewOpenAveragePrice   float64            `json:"newOpenAvgPrice"`
	NewCloseAveragePrice  float64            `json:"newCloseAvgPrice"`
	CloseProfitLoss       float64            `json:"closeProfitLoss"`
	Fee                   float64            `json:"fee"`
	TotalFee              float64            `json:"totalFee"`
	DeductFeeList         []FuturesDeductFee `json:"deductFeeList"`
	PositionShowStatus    string             `json:"positionShowStatus"`
	PositionDrop          bool               `json:"positionDrop"`
	LowSaveTotalFee       types.Number       `json:"lowSaveTotalFee"`
	FeeRates              *FuturesFeeRange   `json:"feeRates"`
	UnrealisedPNL         float64            `json:"unRealizedPnl"`
	// PushedUnrealisedPNL is the unrealised PNL a websocket push carries in place of UnrealisedPNL
	PushedUnrealisedPNL      float64    `json:"pnl"`
	ZeroSaveTotalFeeBinance  float64    `json:"zeroSaveTotalFeeBinance"`
	ZeroTradeTotalFeeBinance float64    `json:"zeroTradeTotalFeeBinance"`
	Version                  uint64     `json:"version"`
	CreateTime               types.Time `json:"createTime"`
	UpdateTime               types.Time `json:"updateTime"`
}

// FuturesDeductFee is a fee deducted in another currency
type FuturesDeductFee struct {
	Currency         currency.Code `json:"currency"`
	DeductFee        float64       `json:"deductFee"`
	ConvertSettleFee float64       `json:"convertSettleFee"`
}

// FuturesFeeRange holds the range of fee rates a position was charged
type FuturesFeeRange struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}

// FuturesFundingFeeResponse is a funding fee paid or received by a position
type FuturesFundingFeeResponse struct {
	ID            uint64    `json:"id"`
	Symbol        string    `json:"symbol"`
	PositionID    FuturesID `json:"positionId"`
	PositionType  uint8     `json:"positionType"`
	PositionValue float64   `json:"positionValue"`
	// Funding is positive when received and negative when paid
	Funding    float64    `json:"funding"`
	Rate       float64    `json:"rate"`
	SettleTime types.Time `json:"settleTime"`
}

// FuturesRiskLimitResponse holds a position side's risk limit
type FuturesRiskLimitResponse struct {
	Symbol                       string  `json:"symbol"`
	PositionType                 uint8   `json:"positionType"`
	OpenType                     uint8   `json:"openType"`
	Level                        uint64  `json:"level"`
	MaxVolume                    float64 `json:"maxVol"`
	MaxLeverage                  uint64  `json:"maxLeverage"`
	MaintenanceMarginRate        float64 `json:"mmr"`
	InitialMarginRate            float64 `json:"imr"`
	Leverage                     uint64  `json:"leverage"`
	LimitBySystem                bool    `json:"limitBySys"`
	CurrentMaintenanceMarginRate float64 `json:"currentMmr"`
}

// FuturesFeeDetailsResponse holds the account's fee rates
type FuturesFeeDetailsResponse struct {
	Symbol                  string                 `json:"symbol"`
	OriginalMakerFee        float64                `json:"originalMakerFee"`
	OriginalTakerFee        float64                `json:"originalTakerFee"`
	JoinDiscount            bool                   `json:"joinDiscount"`
	EnjoyDiscount           bool                   `json:"enjoyDiscount"`
	JoinDeduct              bool                   `json:"joinDeduct"`
	EnjoyDeduct             bool                   `json:"enjoyDeduct"`
	RealMakerFee            float64                `json:"realMakerFee"`
	RealTakerFee            float64                `json:"realTakerFee"`
	DiscountRate            float64                `json:"discountRate"`
	DeductRate              float64                `json:"deductRate"`
	DealAmount              float64                `json:"dealAmount"`
	WalletBalance           float64                `json:"walletBalance"`
	InviterKYC              string                 `json:"inviterKyc"`
	Level                   uint64                 `json:"level"`
	FeeType                 uint8                  `json:"feeType"`
	AgentFee                bool                   `json:"agentFee"`
	FeeRateMode             string                 `json:"feeRateMode"`
	TieredFeeRates          []FuturesTieredFeeRate `json:"tieredFeeRates"`
	TieredDealAmount        float64                `json:"tieredDealAmount"`
	TieredEffectiveDay      uint64                 `json:"tieredEffectiveDay"`
	TieredAppointContract   bool                   `json:"tieredAppointContract"`
	TieredExcludeContractID bool                   `json:"tieredExcludeContractId"`
	TieredContractIDs       []uint64               `json:"tieredContractIds"`
	TieredExcludeZeroFee    bool                   `json:"tieredExcludeZeroFee"`
	StatisticType           string                 `json:"statisticType"`
	FixedStartTime          types.Time             `json:"fixedStartTime"`
	FixedEndTime            types.Time             `json:"fixedEndTime"`
	MinLeverage             uint64                 `json:"minLeverage"`
	MaxLeverage             uint64                 `json:"maxLeverage"`
	IsMaxLeverage           bool                   `json:"isMaxLeverage"`
}

// FuturesChangeMarginRequest holds the parameters of a position margin change
type FuturesChangeMarginRequest struct {
	PositionID uint64  `json:"positionId"`
	Amount     float64 `json:"amount"`
	// Type is FuturesMarginAdd or FuturesMarginSubtract
	Type string `json:"type"`
}

// FuturesAutoAddMarginRequest turns the automatic addition of margin to an isolated position on or off
type FuturesAutoAddMarginRequest struct {
	PositionID uint64 `json:"positionId"`
	IsEnabled  bool   `json:"isEnabled"`
}

// FuturesLeverageResponse holds a position side's leverage
type FuturesLeverageResponse struct {
	Symbol                       string  `json:"symbol"`
	PositionType                 uint8   `json:"positionType"`
	OpenType                     uint8   `json:"openType"`
	Leverage                     uint64  `json:"leverage"`
	MaxLeverageView              uint64  `json:"maxLeverageView"`
	Level                        uint64  `json:"level"`
	MaxVolume                    float64 `json:"maxVol"`
	MaintenanceMarginRate        float64 `json:"mmr"`
	InitialMarginRate            float64 `json:"imr"`
	LimitBySystem                bool    `json:"limitBySys"`
	CurrentMaintenanceMarginRate float64 `json:"currentMmr"`
}

// FuturesChangeLeverageRequest holds the parameters of a leverage change. PositionID selects an open position;
// without one, OpenType, Symbol and PositionType select the leverage to set for new positions
type FuturesChangeLeverageRequest struct {
	PositionID   uint64 `json:"positionId,omitempty"`
	Leverage     uint64 `json:"leverage"`
	OpenType     uint8  `json:"openType,omitempty"`
	Symbol       string `json:"symbol,omitempty"`
	PositionType uint8  `json:"positionType,omitempty"`
	// LeverageMode is 1 for the advanced mode and 2 for the simple one
	LeverageMode     uint8 `json:"leverageMode,omitempty"`
	MarginSelected   *bool `json:"marginSelected,omitempty"`
	LeverageSelected *bool `json:"leverageSelected,omitempty"`
}

// FuturesPositionModeRequest sets the account's position mode
type FuturesPositionModeRequest struct {
	PositionMode uint8 `json:"positionMode"`
}

// FuturesSelfTradePreventionGroupResponse is a self-trade prevention group of sub-accounts
type FuturesSelfTradePreventionGroupResponse struct {
	GroupName  string      `json:"groupName"`
	BlackList  []FuturesID `json:"blackList"`
	CreateTime types.Time  `json:"createTime"`
	UpdateTime types.Time  `json:"updateTime"`
}

// FuturesSelfTradePreventionGroupRequest holds a self-trade prevention group's name and members
type FuturesSelfTradePreventionGroupRequest struct {
	ConfigName string   `json:"configName"`
	Blacklist  []uint64 `json:"blacklist,omitempty"`
}

// unrealisedPNL returns the position's unrealised PNL, which REST replies and websocket pushes carry in different
// fields
func (p *FuturesPositionResponse) unrealisedPNL() float64 {
	if p.UnrealisedPNL != 0 {
		return p.UnrealisedPNL
	}
	return p.PushedUnrealisedPNL
}
