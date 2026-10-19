package coinbase

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Coinbase Exchange currency types
const (
	ExchangeCurrencyTypeCrypto = "crypto"
	ExchangeCurrencyTypeFiat   = "fiat"
)

// ExchangeCurrency is a currency Coinbase Exchange knows of, with the networks it can be deposited and withdrawn on;
// not every currency trades
type ExchangeCurrency struct {
	ID   currency.Code `json:"id"`
	Name string        `json:"name"`
	// MinimumSize is the smallest amount of the currency Coinbase Exchange moves
	MinimumSize types.Number `json:"min_size"`
	// Status is online or delisted
	Status string `json:"status"`
	// Message explains the currency's status, such as deposits opening before trading does
	Message string `json:"message"`
	// MaximumPrecision is the smallest increment of the currency's amounts
	MaximumPrecision types.Number `json:"max_precision"`
	// ConvertibleTo are the currencies Coinbase converts the currency to one for one, such as USD to USDC
	ConvertibleTo  []currency.Code         `json:"convertible_to"`
	Details        ExchangeCurrencyDetails `json:"details"`
	DefaultNetwork string                  `json:"default_network"`
	// SupportedNetworks are the networks the currency moves on, which a fiat currency has none of
	SupportedNetworks []ExchangeCurrencyNetwork `json:"supported_networks"`
	DisplayName       string                    `json:"display_name"`
}

// ExchangeCurrencyDetails describe a currency on its default network. Fields Coinbase sends null for are zero
type ExchangeCurrencyDetails struct {
	// Type is crypto or fiat
	Type   string `json:"type"`
	Symbol string `json:"symbol"`
	// NetworkConfirmations is how many confirmations a deposit needs
	NetworkConfirmations uint64 `json:"network_confirmations"`
	SortOrder            uint64 `json:"sort_order"`
	// CryptoAddressLink and CryptoTransactionLink are block explorer URL templates, holding {{address}} and {{txId}}
	CryptoAddressLink     string   `json:"crypto_address_link"`
	CryptoTransactionLink string   `json:"crypto_transaction_link"`
	PushPaymentMethods    []string `json:"push_payment_methods"`
	GroupTypes            []string `json:"group_types"`
	DisplayName           string   `json:"display_name"`
	// ProcessingTimeSeconds is how long a withdrawal takes to process, in seconds
	ProcessingTimeSeconds   float64 `json:"processing_time_seconds"`
	MinimumWithdrawalAmount float64 `json:"min_withdrawal_amount"`
	MaximumWithdrawalAmount float64 `json:"max_withdrawal_amount"`
}

// ExchangeCurrencyNetwork is a network a currency is deposited and withdrawn on. Fields Coinbase sends null for are
// zero
type ExchangeCurrencyNetwork struct {
	// ID names the network in requests, such as ethereum or solana
	ID   string `json:"id"`
	Name string `json:"name"`
	// Status is online or delisted
	Status string `json:"status"`
	// ContractAddress is the currency's token contract, or asset ID, on the network
	ContractAddress string `json:"contract_address"`
	// CryptoAddressLink and CryptoTransactionLink are block explorer URL templates, holding {{address}} and {{txId}}
	CryptoAddressLink       string  `json:"crypto_address_link"`
	CryptoTransactionLink   string  `json:"crypto_transaction_link"`
	MinimumWithdrawalAmount float64 `json:"min_withdrawal_amount"`
	MaximumWithdrawalAmount float64 `json:"max_withdrawal_amount"`
	// NetworkConfirmations is how many confirmations a deposit on the network needs
	NetworkConfirmations uint64 `json:"network_confirmations"`
	// ProcessingTimeSeconds is how long a withdrawal on the network takes to process, in seconds
	ProcessingTimeSeconds uint64 `json:"processing_time_seconds"`
	// DestinationTagRegex matches the destination tag or memo the network's withdrawals take, when they take one
	DestinationTagRegex string `json:"destination_tag_regex"`
	IsEVMNetwork        bool   `json:"is_evm_network"`
}
