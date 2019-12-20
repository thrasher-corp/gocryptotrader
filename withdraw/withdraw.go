package withdraw

import (
	"errors"
	"time"

	"github.com/gofrs/uuid"
)

// WithdrawCryptocurrencyFundsByExchange withdraws the desired cryptocurrency and amount to a desired cryptocurrency address
func Submit(exchName string, req *Request) (*Response, error) {
	if req == nil {
		return nil, errors.New("crypto withdraw request param is nil")
	}

	// exch := engine.GetExchangeByName(exchName)
	// if exch == nil {
	// 	return nil, engine.ErrExchangeNotFound
	// }
	//
	// exchID, err := exch.WithdrawCryptocurrencyFunds(req)
	// if err != nil {
	// 	return nil, err
	// }
	id, _ := uuid.NewV4()

	resp := &Response{
		ID:             id,
		ExchangeID:     "",
		RequestDetails: req,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	return resp, nil
}

// RequestByID returns a withdrawal request by ID
func RequestByID(id uuid.UUID) (*Response, error) {
	return &Response{}, nil
}