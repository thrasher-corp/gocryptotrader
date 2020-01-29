package engine

import (
	"errors"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	withdrawal "github.com/thrasher-corp/gocryptotrader/database/repository/withdraw"
	"github.com/thrasher-corp/gocryptotrader/management/withdraw"
)

const (
	ErrWithdrawRequestNotFound = "%v not found"
	ErrRequestCannotbeNill = "request cannot be nil"
)

func SubmitWithdrawal(exchName string, req *withdraw.Request) (*withdraw.Response, error) {
	if req == nil {
		return nil, errors.New(ErrRequestCannotbeNill)
	}
	if req.Exchange == "" {
		req.Exchange = exchName
	}
	err := withdraw.Valid(req)
	if err != nil {
		return nil, err
	}
	exch := GetExchangeByName(exchName)
	if exch == nil {
		return nil, ErrExchangeNotFound
	}

	id, _ := uuid.NewV4()
	resp := &withdraw.Response{
		ID: id,
		Exchange: &withdraw.ExchangeResponse{
			Name: exchName,
		},
		RequestDetails: req,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	fmt.Printf("\n%+v\n", req)
	fmt.Printf("\n%+v\n", req.Fiat.Bank)
	// if req.Type == withdraw.Fiat {
	// 	v, errFiat := exch.WithdrawFiatFunds(req)
	// 	if errFiat != nil {
	// 		return nil, errFiat
	// 	}
	// 	resp.Exchange.Status = v.Status
	// 	resp.Exchange.ID = v.ID
	// } else if req.Type == withdraw.Crypto {
	// 	v, err := exch.WithdrawCryptocurrencyFunds(req)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	resp.Exchange.Status = v.Status
	// 	resp.Exchange.ID = v.ID
	// }

	withdraw.Cache.Add(id.String(), resp)
	withdrawal.Event(resp)
	return resp, nil
}

// RequestByID returns a withdrawal request by ID
func WithdrawRequestByID(id string) (*withdraw.Response, error) {
	v := withdraw.Cache.Get(id)
	if v != nil {
		return v.(*withdraw.Response), nil
	}
	l, err := withdrawal.EventByUUID(id)
	if err != nil {
		return nil, fmt.Errorf(ErrWithdrawRequestNotFound, id)
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
