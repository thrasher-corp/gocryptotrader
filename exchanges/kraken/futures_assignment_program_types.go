package kraken

import (
	"errors"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
)

var (
	errFuturesContractTypeEmpty        = errors.New("contract type is empty")
	errFuturesAssignmentTimeFrameEmpty = errors.New("assignment time frame is empty")
	errFuturesAssignmentScopeConflict  = errors.New("an assignment preference cannot be scoped to both a contract and a currency pair")
	errFuturesAssignmentIDEmpty        = errors.New("assignment preference ID is empty")
)

// FuturesAssignmentProgramsResponse holds the account's active assignment preferences
type FuturesAssignmentProgramsResponse struct {
	Participants []FuturesAssignmentPreference `json:"participants"`
	ServerTime   time.Time                     `json:"serverTime"`
}

// FuturesAssignmentPreference is an assignment preference with the ID that deletes it
type FuturesAssignmentPreference struct {
	ID          uint64                       `json:"id"`
	Participant FuturesAssignmentParticipant `json:"participant"`
}

// FuturesAssignmentPreferenceResponse holds the assignment preference a request added or deleted
type FuturesAssignmentPreferenceResponse struct {
	FuturesAssignmentPreference
	ServerTime time.Time `json:"serverTime"`
}

// FuturesAssignmentParticipant holds the terms on which an account takes over the positions of liquidated accounts
type FuturesAssignmentParticipant struct {
	// ContractType is the wallet the preference applies to, such as flex or fi_xbtusd
	ContractType string `json:"contractType"`
	// Contract is set when the preference covers one contract, such as PF_XBTUSD
	Contract string `json:"contract"`
	// MaxSize caps a single assignment and MaxPosition the whole position, each nil when uncapped. They are quote currency
	// notional on the flex wallet and contracts on single-collateral wallets, and a MaxPosition of 0 opts the scope out
	MaxSize     *float64 `json:"maxSize"`
	MaxPosition *float64 `json:"maxPosition"`
	AcceptLong  bool     `json:"acceptLong"`
	AcceptShort bool     `json:"acceptShort"`
	// TimeFrame is WEEKDAYS, WEEKEND or ALL, evaluated in UTC
	TimeFrame string `json:"timeFrame"`
	// Enabled is false for a preference that opts its scope out of the assignments coarser preferences allow
	Enabled bool `json:"enabled"`
	// CashAccountOpenPositionMaxNotional caps the wallet's open position notional, and is nil when uncapped
	CashAccountOpenPositionMaxNotional *float64 `json:"cashAccountOpenPositionMaxNotional"`
	// BaseCurrency and QuoteCurrency are set when the preference covers every contract on a currency pair
	BaseCurrency  currency.Code `json:"baseCurrency"`
	QuoteCurrency currency.Code `json:"quoteCurrency"`
	// MinimumProfitabilityLongBasisPoints and MinimumProfitabilityShortBasisPoints are the minimum edge an assignment of
	// a long or short position targets, each nil when the platform default applies
	MinimumProfitabilityLongBasisPoints  *uint64 `json:"minimumProfitabilityPerAssignmentLongBps"`
	MinimumProfitabilityShortBasisPoints *uint64 `json:"minimumProfitabilityPerAssignmentShortBps"`
}

// FuturesAddAssignmentPreferenceRequest holds the parameters of Add assignment preference. A preference covers a whole
// wallet, every contract on a currency pair or one contract, and finer preferences override coarser ones field by field
type FuturesAddAssignmentPreferenceRequest struct {
	// ContractType is the wallet the preference applies to, such as flex or fi_xbtusd
	ContractType string
	// Contract limits a flex wallet preference to one contract, and cannot be set with Pair
	Contract currency.Pair
	// Pair limits a flex wallet preference to every contract on a currency pair, such as XBT/USD, and cannot be set with
	// Contract. Its quote must be USD
	Pair currency.Pair
	// MaxSize caps a single assignment, and is not sent when it is 0. It is quote currency notional on the flex wallet
	// and contracts on single-collateral wallets, and Kraken drops it from a flex wallet-wide preference
	MaxSize float64
	// MaxPosition caps the position in a contract, holdings included, in MaxSize's units. nil sends no cap, and 0 opts
	// the scope out of assignments
	MaxPosition *float64
	AcceptLong  bool
	AcceptShort bool
	// TimeFrame is WEEKDAYS, WEEKEND or ALL, evaluated in UTC
	TimeFrame string
	// Enabled false adds a preference that opts its scope out of the assignments coarser preferences allow
	Enabled bool
	// CashAccountOpenPositionMaxNotional caps the wallet's open position notional; nil sends no cap
	CashAccountOpenPositionMaxNotional *float64
	// MinimumProfitabilityLongBasisPoints and MinimumProfitabilityShortBasisPoints set the minimum edge an assignment of
	// a long or short position targets, and nil leaves the platform default. The lowest minimums are offered positions
	// first, and Kraken rejects a value equal to the platform default
	MinimumProfitabilityLongBasisPoints  *uint64
	MinimumProfitabilityShortBasisPoints *uint64
}

// FuturesAssignmentPreferencesHistoryResponse holds every change to the account's assignment preferences
type FuturesAssignmentPreferencesHistoryResponse struct {
	Participants []FuturesAssignmentPreferenceChange `json:"participants"`
	ServerTime   time.Time                           `json:"serverTime"`
}

// FuturesAssignmentPreferenceChange is an assignment preference as a change left it
type FuturesAssignmentPreferenceChange struct {
	Deleted     bool                         `json:"deleted"`
	Participant FuturesAssignmentParticipant `json:"participant"`
	Timestamp   time.Time                    `json:"timestamp"`
}
