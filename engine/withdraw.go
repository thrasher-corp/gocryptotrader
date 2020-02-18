package engine

import (
	"errors"
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	withdrawDataStore "github.com/thrasher-corp/gocryptotrader/database/repository/withdraw"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

const (
	// ErrWithdrawRequestNotFound message to display when no record is found
	ErrWithdrawRequestNotFound = "%v not found"
	// ErrRequestCannotbeNil message to display when request is nil
	ErrRequestCannotbeNil = "request cannot be nil"
)

// SubmitWithdrawal preforms validation and submits a new withdraw request to exchange
func SubmitWithdrawal(exchName string, req *withdraw.Request) (*withdraw.Response, error) {
	if req == nil {
		return nil, errors.New(ErrRequestCannotbeNil)
	}

	if req.Exchange == "" {
		req.Exchange = exchName
	}

	err := withdraw.Validate(req)
	if err != nil {
		return nil, err
	}

	exch := GetExchangeByName(exchName)
	if exch == nil {
		return nil, ErrExchangeNotFound
	}

	resp := &withdraw.Response{
		Exchange: &withdraw.ExchangeResponse{
			Name: exchName,
		},
		RequestDetails: req,
	}

	if Bot.Settings.EnableDryRun {
		log.Warnln(log.Global, "Dry run enabled, no withdrawal request will be submitted or have an event created")
		resp.ID = withdraw.DryRunID
		resp.Exchange.Status = "dryrun"
		resp.Exchange.ID = withdraw.DryRunID.String()
	} else {
		if req.Type == withdraw.Fiat {
			v, errFiat := exch.WithdrawFiatFunds(req)
			if errFiat != nil {
				return nil, errFiat
			}
			resp.Exchange.Status = v.Status
			resp.Exchange.ID = v.ID
		} else if req.Type == withdraw.Crypto {
			v, err := exch.WithdrawCryptocurrencyFunds(req)
			if err != nil {
				return nil, err
			}
			resp.Exchange.Status = v.Status
			resp.Exchange.ID = v.ID
		}
		withdrawDataStore.Event(resp)
	}

	withdraw.Cache.Add(resp.ID, resp)
	return resp, nil
}

// WithdrawEventByID returns a withdrawal request by ID
func WithdrawEventByID(id string) (*withdraw.Response, error) {
	v := withdraw.Cache.Get(id)
	if v != nil {
		return v.(*withdraw.Response), nil
	}

	l, err := withdrawDataStore.GetEventByUUID(id)
	if err != nil {
		return nil, fmt.Errorf(ErrWithdrawRequestNotFound, id)
	}
	withdraw.Cache.Add(id, l)
	return l, nil
}

// WithdrawEventByExchange returns a withdrawal request by ID
func WithdrawEventByExchange(exchange string, limit int) ([]*withdraw.Response, error) {
	l, err := withdrawDataStore.GetEventsByExchange(exchange, limit)
	if err != nil {
		return nil, fmt.Errorf(ErrWithdrawRequestNotFound, exchange)
	}
	return l, nil
}

// WithdrawEventByDate returns a withdrawal request by ID
// TODO: impelment method
func WithdrawEventByDate(start, end time.Time) ([]*withdraw.Response, error) {
	return nil, common.ErrNotYetImplemented
}

// WithdrawEventByExchangeID returns a withdrawal request by Exchange ID
func WithdrawEventByExchangeID(exchange, id string) ([]*withdraw.Response, error) {
	l, err := withdrawDataStore.GetEventByExchangeID(exchange, id, 1)
	if err != nil {
		return nil, fmt.Errorf(ErrWithdrawRequestNotFound, exchange)
	}
	return l, nil
}
