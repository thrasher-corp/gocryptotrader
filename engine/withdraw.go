package engine

import (
	"errors"
	"fmt"
	"time"

	withdrawal "github.com/thrasher-corp/gocryptotrader/database/repository/withdraw"
	log "github.com/thrasher-corp/gocryptotrader/logger"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

const (
	// ErrWithdrawRequestNotFound message to display when no record is found
	ErrWithdrawRequestNotFound = "%v not found"
	// ErrRequestCannotbeNill message to display when request is nil
	ErrRequestCannotbeNill = "request cannot be nil"
)

// SubmitWithdrawal preforms validation and submits a new withdraw request to exchange
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

	resp := &withdraw.Response{
		Exchange: &withdraw.ExchangeResponse{
			Name: exchName,
		},
		RequestDetails: req,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if Bot.Settings.EnableDryRun {
		log.Warnln(log.Global, "Dry run enabled no withdrawal request will be submitted or event created")
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
		withdrawal.Event(resp)
	}
	withdraw.Cache.Add(resp.ID, resp)
	return resp, nil
}

// WithdrawEventtByID returns a withdrawal request by ID
func WithdrawEventtByID(id string) (*withdraw.Response, error) {
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

// WithdrawEventByExchange returns a withdrawal request by ID
func WithdrawEventByExchange(exchange string, limit int) ([]withdraw.Response, error) {
	l, err := withdrawal.EventByExchange(exchange, limit)
	if err != nil {
		return nil, fmt.Errorf(ErrWithdrawRequestNotFound, exchange)
	}
	return l, nil
}

// WithdrawEventByDate returns a withdrawal request by ID
func WithdrawEventByDate(start, end time.Time) ([]withdraw.Response, error) {
	return nil, nil
}

// WithdrawEventByExchangeID returns a withdrawal request by Exchange ID
func WithdrawEventByExchangeID(id string) (*withdraw.Response, error) {
	return nil, nil
}
