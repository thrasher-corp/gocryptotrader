package deposit

import "errors"

// ErrAddressNotFound is returned when the deposit address is not found
var ErrAddressNotFound = errors.New("deposit address not found")

// Address holds a deposit address
type Address struct {
	Address string
	Tag     string // Represents either a tag or memo
	Chain   string
}
