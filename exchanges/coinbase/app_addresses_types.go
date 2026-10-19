package coinbase

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
)

// AppCreateAddressRequest creates a receive address for a wallet account
type AppCreateAddressRequest struct {
	AccountID string
	// Name labels the address
	Name string
	// Network is the blockchain the address is on, such as ethereum, solana, bitcoin, base or polygon; the account's
	// currency's default network when unset
	Network string
}

// appCreateAddressBody is Create Address's request body, whose fields are all optional
type appCreateAddressBody struct {
	Name    string `json:"name,omitempty"`
	Network string `json:"network,omitempty"`
}

// AppListAddressesRequest selects a page of an account's addresses
type AppListAddressesRequest struct {
	AccountID string
	// Page selects the page; nil selects the first
	Page *AppPageRequest
}

// AppGetAddressRequest selects one of an account's addresses
type AppGetAddressRequest struct {
	AccountID string
	AddressID string
}

// AppListAddressTransactionsRequest selects a page of the transactions sent to one of an account's addresses
type AppListAddressTransactionsRequest struct {
	AccountID string
	AddressID string
	// Page selects the page; nil selects the first
	Page *AppPageRequest
}

// AppAddress is a receive address of an account; an address belongs to one account only. Currency and
// DestinationTag are undocumented
type AppAddress struct {
	ID      string `json:"id"`
	Address string `json:"address"`
	// Name is the address's label
	Name string `json:"name"`
	// Network is the blockchain the address is on
	Network      string    `json:"network"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Resource     string    `json:"resource"`
	ResourcePath string    `json:"resource_path"`
	// CallbackURL is the URL the address's notifications went to, which Show Address sends; Coinbase stopped sending
	// notifications on 17 February 2025
	CallbackURL    string        `json:"callback_url"`
	Currency       currency.Code `json:"currency"`
	DestinationTag string        `json:"destination_tag"`
}
