package account

import (
	"errors"
	"fmt"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/dispatch"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

func init() {
	service = new(Service)
	service.accounts = make(map[string]*Account)
	service.mux = dispatch.GetNewMux()
}

// SubscribeToExchangeAccount subcribes to your exchange account
func SubscribeToExchangeAccount(exchange string) (dispatch.Pipe, error) {
	exchange = strings.ToLower(exchange)
	service.Lock()

	acc, ok := service.accounts[exchange]
	if !ok {
		service.Unlock()
		return dispatch.Pipe{},
			fmt.Errorf("%s exchange account holdings not found", exchange)
	}

	defer service.Unlock()
	return service.mux.Subscribe(acc.ID)
}

// Process processes new account holdings updates
func Process(h *Holdings) error {
	if h == nil {
		return errors.New("cannot be nil")
	}

	if h.Exchange == "" {
		return errors.New("exchange name unset")
	}

	return service.Update(h)
}

// GetHoldings returns full holdings for an exchange
func GetHoldings(exch string, assetType asset.Item) (Holdings, error) {
	if exch == "" {
		return Holdings{}, errors.New("exchange name unset")
	}

	exch = strings.ToLower(exch)

	if !assetType.IsValid() {
		return Holdings{}, fmt.Errorf("assetType %v is invalid", assetType)
	}

	service.Lock()
	defer service.Unlock()
	h, ok := service.accounts[exch]
	if !ok {
		return Holdings{}, errors.New("exchange account holdings not found")
	}
	for y := range h.h.Accounts {
		if h.h.Accounts[y].AssetType == assetType {
			return *h.h, nil
		}
	}
	return Holdings{}, fmt.Errorf("%v holdings data not found for %s", assetType, exch)
}

// Update updates holdings with new account info
func (s *Service) Update(a *Holdings) error {
	exch := strings.ToLower(a.Exchange)
	s.Lock()
	acc, ok := s.accounts[exch]
	if !ok {
		id, err := s.mux.GetID()
		if err != nil {
			s.Unlock()
			return err
		}

		s.accounts[exch] = &Account{h: a, ID: id}
		s.Unlock()
		return nil
	}

	acc.h.Accounts = a.Accounts
	defer s.Unlock()

	return s.mux.Publish([]uuid.UUID{acc.ID}, acc.h)
}
