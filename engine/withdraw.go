package engine

import (
	"errors"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	withdrawal "github.com/thrasher-corp/gocryptotrader/database/repository/withdraw"
	"github.com/thrasher-corp/gocryptotrader/withdraw"
)

func SubmitWithdrawal(exchName string, req *withdraw.Request) (*withdraw.Response, error) {
	if req == nil {
		return nil, errors.New("crypto withdraw request param is nil")
	}

	err := withdraw.Valid(req)
	if err != nil {
		return nil, err
	}
	exch := GetExchangeByName(exchName)
	if exch == nil {
		return nil, ErrExchangeNotFound
	}

	var exchID string
	if req.Type == withdraw.Crypto {
		exchID, err = exch.WithdrawCryptocurrencyFunds(req)
		if err != nil {
			return nil, err
		}
	}

	if req.Type == withdraw.Fiat {
		exchID, err = exch.WithdrawFiatFunds(req)
		if err != nil {
			return nil, err
		}
	} else if req.Type == withdraw.Crypto {
		exchID, err = exch.WithdrawCryptocurrencyFunds(req)
		if err != nil {
			return nil, err
		}
	}

	id, _ := uuid.NewV4()
	resp := &withdraw.Response{
		ID:             id,
		Exchange: &withdraw.ExchangeResponse{
			Name: exchName,
			ID: exchID,
		},
		RequestDetails: req,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	withdraw.Cache.Add(id.String(), resp)
	withdrawal.Event(resp)
	return resp, nil
}

// RequestByID returns a withdrawal request by ID
func WithdrawRequestByID(id string) (*withdraw.Response, error) {
	v := withdraw.Cache.Get(id)
	if v != nil {
		fmt.Printf("\nCache hit:")
		return v.(*withdraw.Response), nil
	}
	l, err := withdrawal.EventByUUID(id)
	if err != nil {
		return nil, errors.New("not found")
	}
	withdraw.Cache.Add(id, l)
	return l, nil
}

func WithdrawRequestsByExchange(exchange string, limit int) ([]withdraw.Response, error) {
	return nil, nil
}

func WithdrawRequestsByDate(start, end time.Time) ([]withdraw.Response, error) {
	return nil, nil
}
