package account

import (
	"errors"
	"fmt"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/dispatch"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

func init() {
	service = new(Service)
	service.accounts = make(map[string]*Holdings)
	service.mux = dispatch.GetNewMux()
}

var (
	errExchangeNameUnset        = errors.New("exchange name is unset")
	errExchangeHoldingsNotFound = errors.New("exchange holdings not found")
)

// DeployHoldings associates an exchange with with the accounts system and
// returns a pointer.
func DeployHoldings(exch string, verbose bool) (*Holdings, error) {
	if exch == "" {
		return nil, errExchangeNameUnset
	}
	exch = strings.ToLower(exch)
	service.Lock()
	defer service.Unlock()
	h, ok := service.accounts[exch]
	if ok {
		return h, nil
	}
	id, err := service.mux.GetID()
	if err != nil {
		return nil, err
	}

	holdings := &Holdings{
		Exchange: exch,
		mux:      service.mux,
		id:       id,
		funds:    make(map[string]map[asset.Item]map[*currency.Item]*Holding),
		Verbose:  verbose,
	}

	// Loads the default main account for the exchange, this can be overwritten
	// later when LoadAccounts is run from setup in the exchange wrapper if
	// custom accounts needs to be specifically loaded in.
	err = holdings.LoadAccount(string(Main), true)
	if err != nil {
		return nil, err
	}

	service.accounts[exch] = holdings
	return holdings, nil
}

// SubscribeToExchangeAccount subcribes to your exchange account
func SubscribeToExchangeAccount(exch string) (dispatch.Pipe, error) {
	if exch == "" {
		return dispatch.Pipe{}, errExchangeNameUnset
	}
	exch = strings.ToLower(exch)
	service.Lock()
	defer service.Unlock()
	accountHoldings, ok := service.accounts[exch]
	if !ok {
		return dispatch.Pipe{},
			fmt.Errorf("%s %w", exch, errExchangeHoldingsNotFound)
	}
	return service.mux.Subscribe(accountHoldings.id)
}
