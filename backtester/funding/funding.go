package funding

import (
	"errors"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

type FullFunder interface {
	SetFunds(float64)
	GetFunds() float64
}

type ReadFunder interface {
	GetFunds() float64
}

func (f *Funderoo) GetFundingForEvent(e common.EventHandler) *Funderino {
	exch := e.GetExchange()
	a := e.GetAssetType()
	p := e.Pair()
	for i := range f.Fundos {
		if f.Fundos[i].Exchange == exch &&
			f.Fundos[i].Asset == a {
			if f.ExchangeLevelFunding && f.Fundos[i].Quote == p.Quote {
				return &f.Fundos[i]
			} else if !f.ExchangeLevelFunding && f.Fundos[i].Base == p.Base && f.Fundos[i].Quote == p.Quote {
				return &f.Fundos[i]
			}
		}
	}
	return nil
}

var errNotFound = errors.New("funding not found")

type Funderoo struct {
	ExchangeLevelFunding bool
	Fundos               []Funderino
}

type Funderino struct {
	Exchange string
	Asset    asset.Item
	Base     currency.Code
	Quote    currency.Code
	Funding  float64
}

// perhaps funding should also include sizing? This would allow sizing to easliy occur across portfolio and exchange and stay within size
// but hold off, because scope is really hard here
