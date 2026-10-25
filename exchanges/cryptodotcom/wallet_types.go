package cryptodotcom

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Deposit address statuses
const (
	DepositAddressStatusInactive = "0"
	DepositAddressStatusActive   = "1"
)

// Deposit statuses, which get-deposit-history reports and filters by
const (
	DepositStatusNotArrived = "0"
	DepositStatusArrived    = "1"
	DepositStatusFailed     = "2"
	DepositStatusPending    = "3"
)

// Withdrawal statuses, which get-withdrawal-history reports and filters by
const (
	WithdrawalStatusPending           = "0"
	WithdrawalStatusProcessing        = "1"
	WithdrawalStatusRejected          = "2"
	WithdrawalStatusPaymentInProgress = "3"
	WithdrawalStatusPaymentFailed     = "4"
	WithdrawalStatusCompleted         = "5"
	WithdrawalStatusCancelled         = "6"
)

// WithdrawalRequest withdraws a currency to a whitelisted address; the client withdrawal ID, address tag and network
// ID are optional
type WithdrawalRequest struct {
	ClientWithdrawalID string        `json:"client_wid,omitempty"`
	Currency           currency.Code `json:"currency"`
	Amount             types.Number  `json:"amount"`
	Address            string        `json:"address"`
	// AddressTag is the memo or destination tag currencies such as XRP and XLM need
	AddressTag string `json:"address_tag,omitempty"`
	// NetworkID is a network from GetCurrencyNetworks, the currency's default network when empty
	NetworkID string `json:"network_id,omitempty"`
}

// WithdrawalResponse is a crypto withdrawal. A created withdrawal reports only its ID, client withdrawal ID, currency,
// amount, fee, address, network and creation time
type WithdrawalResponse struct {
	ID                 string        `json:"id"`
	ClientWithdrawalID string        `json:"client_wid"`
	Currency           currency.Code `json:"currency"`
	// Symbol is the currency as create-withdrawal's documented example names it, where its schema names it currency
	Symbol currency.Code       `json:"symbol"`
	Amount types.PreciseNumber `json:"amount"`
	Fee    types.PreciseNumber `json:"fee"`
	// Address carries the address tag, if any, after a question mark
	Address    string     `json:"address"`
	CreateTime types.Time `json:"create_time"`
	UpdateTime types.Time `json:"update_time"`
	// Status is a WithdrawalStatus constant
	Status        string `json:"status"`
	TransactionID string `json:"txid"`
	// NetworkID is only set for currencies withdrawn on more than one network
	NetworkID string `json:"network_id"`
}

// CurrencyNetworksResponse is the networks each currency can be deposited and withdrawn on
type CurrencyNetworksResponse struct {
	UpdateTime types.Time                       `json:"update_time"`
	Currencies map[currency.Code]WalletCurrency `json:"currency_map"`
}

// WalletCurrency is a currency's networks
type WalletCurrency struct {
	FullName string `json:"full_name"`
	// DefaultNetwork is the network a withdrawal naming none uses when there is more than one
	DefaultNetwork string            `json:"default_network"`
	Networks       []CurrencyNetwork `json:"network_list"`
}

// CurrencyNetwork is a network a currency can be deposited and withdrawn on
type CurrencyNetwork struct {
	NetworkID               string              `json:"network_id"`
	WithdrawalFee           types.PreciseNumber `json:"withdrawal_fee"`
	WithdrawalEnabled       bool                `json:"withdraw_enabled"`
	MinimumWithdrawalAmount types.PreciseNumber `json:"min_withdrawal_amount"`
	DepositEnabled          bool                `json:"deposit_enabled"`
	ConfirmationsRequired   uint64              `json:"confirmation_required,string"`
}

// depositAddressParams is the params of a get-deposit-address request
type depositAddressParams struct {
	Currency currency.Code `json:"currency"`
}

// depositAddressesResponse is the result of a get-deposit-address request
type depositAddressesResponse struct {
	DepositAddresses []DepositAddressResponse `json:"deposit_address_list"`
}

// DepositAddressResponse is an address a currency can be deposited to on one network
type DepositAddressResponse struct {
	ID       string        `json:"id"`
	Currency currency.Code `json:"currency"`
	// Network is the network ID, which tells apart a currency's addresses: CRO's are CRO on its main net and ETH as an
	// ERC20 token
	Network string `json:"network"`
	// Address carries the address tag, if any, after a question mark
	Address    string     `json:"address"`
	CreateTime types.Time `json:"create_time"`
	// Status is a DepositAddressStatus constant
	Status string `json:"status"`
}

// WalletHistoryRequest filters a deposit or withdrawal history; every field is optional
type WalletHistoryRequest struct {
	Currency currency.Code
	// StartTime is 90 days before the current time when zero
	StartTime time.Time
	// EndTime is the current time when zero
	EndTime time.Time
	// PageSize is at most 200, and 20 when zero
	PageSize uint64
	// Page counts from 0
	Page uint64
	// Status is a DepositStatus constant for deposits and a WithdrawalStatus constant for withdrawals
	Status string
}

// walletHistoryParams is the params of a get-deposit-history or get-withdrawal-history request
type walletHistoryParams struct {
	Currency  string `json:"currency,omitempty"`
	StartTime uint64 `json:"start_ts,omitempty,string"`
	EndTime   uint64 `json:"end_ts,omitempty,string"`
	PageSize  uint64 `json:"page_size,omitempty,string"`
	Page      uint64 `json:"page,omitempty,string"`
	Status    string `json:"status,omitempty"`
}

// depositHistoryResponse is the result of a get-deposit-history request
type depositHistoryResponse struct {
	Deposits []DepositResponse `json:"deposit_list"`
}

// DepositResponse is a crypto deposit
type DepositResponse struct {
	ID       string              `json:"id"`
	Currency currency.Code       `json:"currency"`
	Amount   types.PreciseNumber `json:"amount"`
	Fee      types.PreciseNumber `json:"fee"`
	// Address carries the address tag, if any, after a question mark
	Address    string     `json:"address"`
	CreateTime types.Time `json:"create_time"`
	UpdateTime types.Time `json:"update_time"`
	// Status is a DepositStatus constant
	Status string `json:"status"`
	// TransactionID is left out of the docs, but ccxt records Crypto.com sending it
	TransactionID string `json:"txid"`
}

// withdrawalHistoryResponse is the result of a get-withdrawal-history request
type withdrawalHistoryResponse struct {
	Withdrawals []WithdrawalResponse `json:"withdrawal_list"`
}
