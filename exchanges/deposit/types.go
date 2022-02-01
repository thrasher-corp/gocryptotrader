package deposit

import "errors"

var (
	// ErrUnsupportedOnExchange defines an error when there is no deposit address
	// support on an exchange for currency i.e. a tokenized stock.
	ErrUnsupportedOnExchange = errors.New("unsupported currency no deposit address")

	// ErrAddressBeingCreated is for when an exchange creates deposit addresses
	// ad-hoc per request and another request might need to be made.
	ErrAddressBeingCreated = errors.New("deposit address is being created")
)

// Address holds a deposit address
type Address struct {
	Address string
	Tag     string // Represents either a tag or memo
	Chain   string
}
