package okx

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// LeadPositionsRequest holds the filters for a lead trader's own open or closed lead positions
type LeadPositionsRequest struct {
	InstrumentType string
	InstrumentID   string
	// After returns records earlier than this lead position ID
	After string
	// Before returns records newer than this lead position ID
	Before string
	Limit  uint64
}

// LeadPositionResponse holds an open lead position; the public lead trader endpoint omits the order and stop fields
type LeadPositionResponse struct {
	InstrumentID                 string        `json:"instId"`
	SubPositionID                string        `json:"subPosId"`
	PositionSide                 string        `json:"posSide"`
	MarginMode                   string        `json:"mgnMode"`
	Leverage                     types.Number  `json:"lever"`
	OpenOrderID                  string        `json:"openOrdId"`
	OpenAveragePrice             types.Number  `json:"openAvgPx"`
	OpenTime                     types.Time    `json:"openTime"`
	SubPosition                  types.Number  `json:"subPos"`
	TakeProfitTriggerPrice       types.Number  `json:"tpTriggerPx"`
	StopLossTriggerPrice         types.Number  `json:"slTriggerPx"`
	AlgoID                       string        `json:"algoId"`
	InstrumentType               string        `json:"instType"`
	TakeProfitOrderPrice         types.Number  `json:"tpOrdPx"`
	StopLossOrderPrice           types.Number  `json:"slOrdPx"`
	Margin                       types.Number  `json:"margin"`
	UnrealisedProfitAndLoss      types.Number  `json:"upl"`
	UnrealisedProfitAndLossRatio types.Number  `json:"uplRatio"`
	MarkPrice                    types.Number  `json:"markPx"`
	UniqueCode                   string        `json:"uniqueCode"`
	MarginCurrency               currency.Code `json:"ccy"`
	AvailableSubPosition         types.Number  `json:"availSubPos"`
}

// LeadPositionHistoryResponse holds a closed lead position; the public lead trader endpoint omits the order and
// closing fields
type LeadPositionHistoryResponse struct {
	InstrumentID       string        `json:"instId"`
	SubPositionID      string        `json:"subPosId"`
	PositionSide       string        `json:"posSide"`
	MarginMode         string        `json:"mgnMode"`
	Leverage           types.Number  `json:"lever"`
	OpenOrderID        string        `json:"openOrdId"`
	OpenAveragePrice   types.Number  `json:"openAvgPx"`
	OpenTime           types.Time    `json:"openTime"`
	SubPosition        types.Number  `json:"subPos"`
	CloseTime          types.Time    `json:"closeTime"`
	CloseAveragePrice  types.Number  `json:"closeAvgPx"`
	ProfitAndLoss      types.Number  `json:"pnl"`
	ProfitAndLossRatio types.Number  `json:"pnlRatio"`
	InstrumentType     string        `json:"instType"`
	Margin             types.Number  `json:"margin"`
	Currency           currency.Code `json:"ccy"`
	MarkPrice          types.Number  `json:"markPx"`
	UniqueCode         string        `json:"uniqueCode"`
	// ProfitSharingAmount is deprecated by OKX
	ProfitSharingAmount types.Number `json:"profitSharingAmt"`
	ClosedSubPosition   types.Number `json:"closeSubPos"`
	CloseType           string       `json:"type"`
}

// PlaceLeadStopOrderRequest holds the take-profit and stop-loss settings for an open lead position
type PlaceLeadStopOrderRequest struct {
	InstrumentType         string  `json:"instType,omitempty"`
	SubPositionID          string  `json:"subPosId"`
	TakeProfitTriggerPrice float64 `json:"tpTriggerPx,omitempty,string"`
	StopLossTriggerPrice   float64 `json:"slTriggerPx,omitempty,string"`
	// TakeProfitOrderPrice is -1 for the market price
	TakeProfitOrderPrice float64 `json:"tpOrdPx,omitempty,string"`
	// StopLossOrderPrice is -1 for the market price
	StopLossOrderPrice         float64 `json:"slOrdPx,omitempty,string"`
	TakeProfitTriggerPriceType string  `json:"tpTriggerPxType,omitempty"`
	StopLossTriggerPriceType   string  `json:"slTriggerPxType,omitempty"`
	Tag                        string  `json:"tag,omitempty"`
}

// LeadPositionActionResponse holds the lead position a stop order or close request acted on
type LeadPositionActionResponse struct {
	SubPositionID string `json:"subPosId"`
	Tag           string `json:"tag"`
}

// CloseLeadPositionRequest holds the parameters to close a lead position
type CloseLeadPositionRequest struct {
	InstrumentType string  `json:"instType,omitempty"`
	SubPositionID  string  `json:"subPosId"`
	OrderType      string  `json:"ordType,omitempty"`
	Price          float64 `json:"px,omitempty,string"`
	Tag            string  `json:"tag,omitempty"`
}

// LeadingInstrumentResponse holds an instrument and whether the lead trader leads it
type LeadingInstrumentResponse struct {
	InstrumentID string `json:"instId"`
	Enabled      bool   `json:"enabled"`
}

// AmendLeadingInstrumentsRequest holds the instruments a lead trader leads
type AmendLeadingInstrumentsRequest struct {
	InstrumentType string `json:"instType,omitempty"`
	// InstrumentID takes several instruments separated by commas
	InstrumentID string `json:"instId"`
}

// ProfitSharingDetailsRequest holds the filters for a lead trader's profit sharing details
type ProfitSharingDetailsRequest struct {
	InstrumentType string
	// After returns records earlier than this profit sharing ID
	After string
	// Before returns records newer than this profit sharing ID
	Before string
	Limit  uint64
}

// ProfitSharingDetailResponse holds profit a copy trader shared with the lead trader
type ProfitSharingDetailResponse struct {
	Currency            currency.Code `json:"ccy"`
	ProfitSharingAmount types.Number  `json:"profitSharingAmt"`
	NickName            string        `json:"nickName"`
	ProfitSharingID     string        `json:"profitSharingId"`
	InstrumentType      string        `json:"instType"`
	PortraitLink        string        `json:"portLink"`
	Timestamp           types.Time    `json:"ts"`
}

// TotalProfitSharingResponse holds the total profit shared with a lead trader since joining the platform
type TotalProfitSharingResponse struct {
	Currency                 currency.Code `json:"ccy"`
	TotalProfitSharingAmount types.Number  `json:"totalProfitSharingAmt"`
	InstrumentType           string        `json:"instType"`
}

// UnrealisedProfitSharingDetailResponse holds profit a copy trader is expected to share at the next settlement
type UnrealisedProfitSharingDetailResponse struct {
	Currency                      currency.Code `json:"ccy"`
	UnrealisedProfitSharingAmount types.Number  `json:"unrealizedProfitSharingAmt"`
	NickName                      string        `json:"nickName"`
	InstrumentType                string        `json:"instType"`
	PortraitLink                  string        `json:"portLink"`
	Timestamp                     types.Time    `json:"ts"`
}

// TotalUnrealisedProfitSharingResponse holds the total profit expected to be shared at the next settlement
type TotalUnrealisedProfitSharingResponse struct {
	ProfitSharingTimestamp             types.Time   `json:"profitSharingTs"`
	TotalUnrealisedProfitSharingAmount types.Number `json:"totalUnrealizedProfitSharingAmt"`
}

// AmendProfitSharingRatioRequest holds a lead trader's new profit sharing ratio
type AmendProfitSharingRatioRequest struct {
	InstrumentType string `json:"instType,omitempty"`
	// ProfitSharingRatio is always sent since zero is a valid ratio; 0.1 is 10%
	ProfitSharingRatio float64 `json:"profitSharingRatio,string"`
}

// CopyTradingResultResponse holds whether a copy trading setting was applied
type CopyTradingResultResponse struct {
	Result bool `json:"result"`
}

// CopyTradingAccountConfigurationResponse holds the account's copy and lead trading configuration
type CopyTradingAccountConfigurationResponse struct {
	UniqueCode   string            `json:"uniqueCode"`
	NickName     string            `json:"nickName"`
	PortraitLink string            `json:"portLink"`
	Details      []CopyTradingRole `json:"details"`
}

// CopyTradingRole holds the account's copy trading role for an instrument type
type CopyTradingRole struct {
	InstrumentType      string       `json:"instType"`
	RoleType            string       `json:"roleType"`
	ProfitSharingRatio  types.Number `json:"profitSharingRatio"`
	MaxCopyTraderNumber types.Number `json:"maxCopyTraderNum"`
	CopyTraderNumber    types.Number `json:"copyTraderNum"`
}

// CopySettingsRequest holds the settings for copying a lead trader
type CopySettingsRequest struct {
	InstrumentType       string `json:"instType,omitempty"`
	UniqueCode           string `json:"uniqueCode"`
	CopyMarginMode       string `json:"copyMgnMode"`
	CopyInstrumentIDType string `json:"copyInstIdType"`
	// InstrumentID takes several instruments separated by commas
	InstrumentID         string  `json:"instId,omitempty"`
	CopyMode             string  `json:"copyMode,omitempty"`
	CopyTotalAmount      float64 `json:"copyTotalAmt,string"`
	CopyAmount           float64 `json:"copyAmt,omitempty,string"`
	CopyRatio            float64 `json:"copyRatio,omitempty,string"`
	TakeProfitRatio      float64 `json:"tpRatio,omitempty,string"`
	StopLossRatio        float64 `json:"slRatio,omitempty,string"`
	StopLossTotalAmount  float64 `json:"slTotalAmt,omitempty,string"`
	SubPositionCloseType string  `json:"subPosCloseType"`
	Tag                  string  `json:"tag,omitempty"`
}

// StopCopyingRequest holds the lead trader to stop copying
type StopCopyingRequest struct {
	InstrumentType       string `json:"instType,omitempty"`
	UniqueCode           string `json:"uniqueCode"`
	SubPositionCloseType string `json:"subPosCloseType"`
}

// LeadTraderRequest identifies a lead trader
type LeadTraderRequest struct {
	InstrumentType string
	UniqueCode     string
}

// CopySettingsResponse holds the account's copy settings for a lead trader
type CopySettingsResponse struct {
	CopyMode             string                   `json:"copyMode"`
	CopyAmount           types.Number             `json:"copyAmt"`
	CopyRatio            types.Number             `json:"copyRatio"`
	CopyTotalAmount      types.Number             `json:"copyTotalAmt"`
	TakeProfitRatio      types.Number             `json:"tpRatio"`
	StopLossRatio        types.Number             `json:"slRatio"`
	CopyInstrumentIDType string                   `json:"copyInstIdType"`
	InstrumentIDs        []CopySettingsInstrument `json:"instIds"`
	StopLossTotalAmount  types.Number             `json:"slTotalAmt"`
	SubPositionCloseType string                   `json:"subPosCloseType"`
	CopyMarginMode       string                   `json:"copyMgnMode"`
	Currency             currency.Code            `json:"ccy"`
	CopyState            string                   `json:"copyState"`
	Tag                  string                   `json:"tag"`
}

// CopySettingsInstrument holds whether an instrument of the lead trader is copied
type CopySettingsInstrument struct {
	InstrumentID string        `json:"instId"`
	Enabled      types.Boolean `json:"enabled"`
}

// MyLeadTraderResponse holds a lead trader the account copies
type MyLeadTraderResponse struct {
	PortraitLink            string        `json:"portLink"`
	NickName                string        `json:"nickName"`
	Margin                  types.Number  `json:"margin"`
	CopyTotalAmount         types.Number  `json:"copyTotalAmt"`
	CopyTotalProfitAndLoss  types.Number  `json:"copyTotalPnl"`
	UniqueCode              string        `json:"uniqueCode"`
	Currency                currency.Code `json:"ccy"`
	ProfitSharingRatio      types.Number  `json:"profitSharingRatio"`
	BeginCopyTime           types.Time    `json:"beginCopyTime"`
	UnrealisedProfitAndLoss types.Number  `json:"upl"`
	TodayProfitAndLoss      types.Number  `json:"todayPnl"`
	LeadMode                string        `json:"leadMode"`
}

// CopyTradingConfigurationResponse holds the limits that apply to copy settings
type CopyTradingConfigurationResponse struct {
	MaxCopyAmount      types.Number `json:"maxCopyAmt"`
	MinCopyAmount      types.Number `json:"minCopyAmt"`
	MaxCopyTotalAmount types.Number `json:"maxCopyTotalAmt"`
	MinCopyRatio       types.Number `json:"minCopyRatio"`
	MaxCopyRatio       types.Number `json:"maxCopyRatio"`
	MaxTakeProfitRatio types.Number `json:"maxTpRatio"`
	MaxStopLossRatio   types.Number `json:"maxSlRatio"`
}

// LeadTraderRanksRequest holds the filters for the lead trader ranking
type LeadTraderRanksRequest struct {
	InstrumentType string
	SortType       string
	// HasVacancy limits the ranking to lead traders with copy trader vacancies
	HasVacancy bool
	// MinLeadDays is 1 for 7 days, 2 for 30 days, 3 for 90 days and 4 for 180 days
	MinLeadDays              uint64
	MinAssets                float64
	MaxAssets                float64
	MinAssetsUnderManagement float64
	MaxAssetsUnderManagement float64
	// DataVersion pages through a ranking snapshot; OKX keeps the last five, taken every 10 minutes
	DataVersion string
	Page        uint64
	Limit       uint64
}

// LeadTraderRanksResponse holds a page of the lead trader ranking
type LeadTraderRanksResponse struct {
	DataVersion string           `json:"dataVer"`
	TotalPage   types.Number     `json:"totalPage"`
	Ranks       []LeadTraderRank `json:"ranks"`
}

// LeadTraderRank holds a lead trader's ranking statistics
type LeadTraderRank struct {
	AssetsUnderManagement       types.Number                   `json:"aum"`
	CopyState                   string                         `json:"copyState"`
	MaxCopyTraderNumber         types.Number                   `json:"maxCopyTraderNum"`
	CopyTraderNumber            types.Number                   `json:"copyTraderNum"`
	AccumulatedCopyTraderNumber types.Number                   `json:"accCopyTraderNum"`
	PortraitLink                string                         `json:"portLink"`
	NickName                    string                         `json:"nickName"`
	Currency                    currency.Code                  `json:"ccy"`
	UniqueCode                  string                         `json:"uniqueCode"`
	WinRatio                    types.Number                   `json:"winRatio"`
	LeadDays                    types.Number                   `json:"leadDays"`
	TraderInstruments           []string                       `json:"traderInsts"`
	ProfitAndLoss               types.Number                   `json:"pnl"`
	ProfitAndLossRatio          types.Number                   `json:"pnlRatio"`
	ProfitAndLossRatios         []LeadTraderProfitAndLossRatio `json:"pnlRatios"`
	TraditionalFinanceTrader    bool                           `json:"tradFiTrader"`
}

// LeadTraderProfitAndLossRatio holds a lead trader's profit and loss ratio on a day
type LeadTraderProfitAndLossRatio struct {
	BeginTimestamp     types.Time   `json:"beginTs"`
	ProfitAndLossRatio types.Number `json:"pnlRatio"`
}

// LeadTraderProfitAndLossResponse holds a lead trader's profit and loss for a week, or accumulated to a day
type LeadTraderProfitAndLossResponse struct {
	BeginTimestamp     types.Time   `json:"beginTs"`
	ProfitAndLoss      types.Number `json:"pnl"`
	ProfitAndLossRatio types.Number `json:"pnlRatio"`
}

// LeadTraderPerformanceRequest identifies a lead trader and the period of their performance
type LeadTraderPerformanceRequest struct {
	InstrumentType string
	UniqueCode     string
	// LastDays is 1 for 7 days, 2 for 30 days, 3 for 90 days and 4 for 365 days
	LastDays string
}

// LeadTraderStatsResponse holds key data of a lead trader's performance
type LeadTraderStatsResponse struct {
	WinRatio                       types.Number  `json:"winRatio"`
	ProfitDays                     types.Number  `json:"profitDays"`
	LossDays                       types.Number  `json:"lossDays"`
	CurrentCopyTraderProfitAndLoss types.Number  `json:"curCopyTraderPnl"`
	AverageSubPositionNotional     types.Number  `json:"avgSubPosNotional"`
	InvestmentAmount               types.Number  `json:"investAmt"`
	Currency                       currency.Code `json:"ccy"`
}

// LeadTraderCurrencyPreferenceResponse holds the share of a lead trader's trading in a currency
type LeadTraderCurrencyPreferenceResponse struct {
	// Currency is Other for the remaining currencies combined
	Currency currency.Code `json:"ccy"`
	Ratio    types.Number  `json:"ratio"`
}

// LeadTraderPositionsRequest holds the filters for a lead trader's current or closed lead positions
type LeadTraderPositionsRequest struct {
	InstrumentType string
	UniqueCode     string
	// After returns records earlier than this lead position ID
	After string
	// Before returns records newer than this lead position ID
	Before string
	Limit  uint64
}

// CopyTradersRequest holds the filters for a lead trader's copy traders
type CopyTradersRequest struct {
	InstrumentType string
	UniqueCode     string
	Limit          uint64
}

// CopyTradersResponse holds a lead trader's copy traders, highest profit and loss first
type CopyTradersResponse struct {
	CopyTotalProfitAndLoss types.Number  `json:"copyTotalPnl"`
	Currency               currency.Code `json:"ccy"`
	// CopyTraderNumberChange is the change over the last 7 days, negative when copy traders left
	CopyTraderNumberChange types.Number `json:"copyTraderNumChg"`
	// CopyTraderNumberChangeRatio is the change ratio over the last 7 days
	CopyTraderNumberChangeRatio types.Number `json:"copyTraderNumChgRatio"`
	CopyTraders                 []CopyTrader `json:"copyTraders"`
}

// CopyTrader holds a copy trader of a lead trader
type CopyTrader struct {
	BeginCopyTime types.Time   `json:"beginCopyTime"`
	NickName      string       `json:"nickName"`
	PortraitLink  string       `json:"portLink"`
	ProfitAndLoss types.Number `json:"pnl"`
}
