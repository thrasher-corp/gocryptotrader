package kucoin

import (
	"fmt"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// UnmarshalTo acts as interface to exchange API response
type UnmarshalTo interface {
	GetError() error
}

// Error defines all error information for each request
type Error struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
}

// GetError returns an error unless the response code reports success. KuCoin's success code is 200000, which newer
// endpoints send with a message such as "success", so the message alone does not make a response an error; a few
// endpoints answer 200
func (e Error) GetError() error {
	code, err := strconv.ParseInt(e.Code, 10, 64)
	if err != nil {
		return fmt.Errorf("error parsing response code %q: %w", e.Code, err)
	}
	if code == 200 || code == 200000 {
		return nil
	}
	return fmt.Errorf("code: %s message: %s", e.Code, e.Msg)
}

// AccountInfo represents account information
type AccountInfo struct {
	ID          string        `json:"id"`
	Currency    currency.Code `json:"currency"`
	AccountType string        `json:"type"` // Account type: main, trade, trade_hf, margin
	Balance     types.Number  `json:"balance"`
	Available   types.Number  `json:"available"`
	Holds       types.Number  `json:"holds"`
}

// Response represents response model and implements UnmarshalTo interface
type Response struct {
	Data any `json:"data"`
	Error
}
