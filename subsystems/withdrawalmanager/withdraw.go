package withdrawalmanager

import (
	"errors"
	"fmt"
	"time"

	dbwithdraw "github.com/thrasher-corp/gocryptotrader/database/repository/withdraw"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
	"github.com/thrasher-corp/gocryptotrader/subsystems/exchangemanager"
)

var (
	// ErrWithdrawRequestNotFound message to display when no record is found
	ErrWithdrawRequestNotFound = errors.New("request not found")
)

type Manager struct {
	exchangeManager iExchangeManager
	isDryRun        bool
}

type iExchangeManager interface {
	GetExchangeByName(string) exchange.IBotExchange
}

func Setup(manager iExchangeManager, isDryRun bool) (*Manager, error) {
	if manager == nil {
		return nil, errors.New("nil manager")
	}
	return &Manager{
		exchangeManager: manager,
		isDryRun:        isDryRun,
	}, nil
}

// SubmitWithdrawal performs validation and submits a new withdraw request to
// exchange
func (m *Manager) SubmitWithdrawal(req *withdraw.Request) (*withdraw.Response, error) {
	if req == nil {
		return nil, withdraw.ErrRequestCannotBeNil
	}

	exch := m.exchangeManager.GetExchangeByName(req.Exchange)
	if exch == nil {
		return nil, exchangemanager.ErrExchangeNotFound
	}

	resp := &withdraw.Response{
		Exchange: withdraw.ExchangeResponse{
			Name: req.Exchange,
		},
		RequestDetails: *req,
	}

	var err error
	if m.isDryRun {
		log.Warnln(log.Global, "Dry run enabled, no withdrawal request will be submitted or have an event created")
		resp.ID = withdraw.DryRunID
		resp.Exchange.Status = "dryrun"
		resp.Exchange.ID = withdraw.DryRunID.String()
	} else {
		var ret *withdraw.ExchangeResponse
		if req.Type == withdraw.Fiat {
			ret, err = exch.WithdrawFiatFunds(req)
			if err != nil {
				resp.Exchange.Status = err.Error()
			} else {
				resp.Exchange.Status = ret.Status
				resp.Exchange.ID = ret.ID
			}
		} else if req.Type == withdraw.Crypto {
			ret, err = exch.WithdrawCryptocurrencyFunds(req)
			if err != nil {
				resp.Exchange.Status = err.Error()
			} else {
				resp.Exchange.Status = ret.Status
				resp.Exchange.ID = ret.ID
			}
		}
		// withdrawDataStore.Event(resp)
	}
	if err == nil {
		withdraw.Cache.Add(resp.ID, resp)
	}
	return resp, nil
}

// WithdrawalEventByID returns a withdrawal request by ID
func (m *Manager) WithdrawalEventByID(id string) (*withdraw.Response, error) {
	v := withdraw.Cache.Get(id)
	if v != nil {
		return v.(*withdraw.Response), nil
	}

	l, err := dbwithdraw.GetEventByUUID(id)
	if err != nil {
		return nil, fmt.Errorf("%w %v", ErrWithdrawRequestNotFound, id)
	}
	withdraw.Cache.Add(id, l)
	return l, nil
}

// WithdrawalEventByExchange returns a withdrawal request by ID
func (m *Manager) WithdrawalEventByExchange(exchange string, limit int) ([]*withdraw.Response, error) {
	exch := m.exchangeManager.GetExchangeByName(exchange)
	if exch == nil {
		return nil, exchangemanager.ErrExchangeNotFound
	}

	return dbwithdraw.GetEventsByExchange(exchange, limit)
}

// WithdrawEventByDate returns a withdrawal request by ID
func (m *Manager) WithdrawEventByDate(exchange string, start, end time.Time, limit int) ([]*withdraw.Response, error) {
	exch := m.exchangeManager.GetExchangeByName(exchange)
	if exch == nil {
		return nil, exchangemanager.ErrExchangeNotFound
	}

	return dbwithdraw.GetEventsByDate(exchange, start, end, limit)
}

// WithdrawalEventByExchangeID returns a withdrawal request by Exchange ID
func (m *Manager) WithdrawalEventByExchangeID(exchange, id string) (*withdraw.Response, error) {
	exch := m.exchangeManager.GetExchangeByName(exchange)
	if exch == nil {
		return nil, exchangemanager.ErrExchangeNotFound
	}

	return dbwithdraw.GetEventByExchangeID(exchange, id)
}
