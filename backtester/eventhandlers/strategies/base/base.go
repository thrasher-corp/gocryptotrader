package base

import (
	"errors"
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
)

// Strategy is base implementation of the Handler interface
type Strategy struct {
	useSimultaneousProcessing bool
	usingExchangeLevelFunding bool
}

// GetBaseData returns the non-interface version of the Handler
func (s *Strategy) GetBaseData(d data.Handler) (signal.Signal, error) {
	if d == nil {
		return signal.Signal{}, common.ErrNilArguments
	}
	latest := d.Latest()
	if latest == nil {
		return signal.Signal{}, common.ErrNilEvent
	}
	return signal.Signal{
		Base: event.Base{
			Offset:       latest.GetOffset(),
			Exchange:     latest.GetExchange(),
			Time:         latest.GetTime(),
			CurrencyPair: latest.Pair(),
			AssetType:    latest.GetAssetType(),
			Interval:     latest.GetInterval(),
		},
		ClosePrice: latest.ClosePrice(),
		HighPrice:  latest.HighPrice(),
		OpenPrice:  latest.OpenPrice(),
		LowPrice:   latest.LowPrice(),
	}, nil
}

// UseSimultaneousProcessing returns whether multiple currencies can be assessed in one go
func (s *Strategy) UseSimultaneousProcessing() bool {
	return s.useSimultaneousProcessing
}

// SetSimultaneousProcessing sets whether multiple currencies can be assessed in one go
func (s *Strategy) SetSimultaneousProcessing(b bool) {
	s.useSimultaneousProcessing = b
}

// UseExchangeLevelFunding returns whether funding is based on currency pairs or individual currencies at the exchange level
func (s *Strategy) UseExchangeLevelFunding() bool {
	return s.usingExchangeLevelFunding
}

// SetExchangeLevelFunding sets whether funding is based on currency pairs or individual currencies at the exchange level
func (s *Strategy) SetExchangeLevelFunding(b bool) {
	s.usingExchangeLevelFunding = b
}

var errCurrenciesMustBeTheSame = errors.New("lol")
var errExchangeCantMatchDummy = errors.New("lol")

func (s *Strategy) SendFundingToExchange(sender, receiver funding.Item, amount, fee decimal.Decimal) error {
	if sender.Item != receiver.Item {
		return errCurrenciesMustBeTheSame
	}
	if sender.Exchange == receiver.Exchange {
		return errExchangeCantMatchDummy
	}
	err := sender.Reserve(amount.Add(fee))
	if err != nil {
		return fmt.Errorf("%w", err)
	}
	err = sender.Release(amount.Add(fee), decimal.Zero)
	if err != nil {
		return fmt.Errorf("%w", err)
	}
	receiver.IncreaseAvailable(amount.Sub(fee))

	return nil
}
