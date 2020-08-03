package position

import (
	"math"

	"github.com/thrasher-corp/gocryptotrader/backtest/event"
	"github.com/thrasher-corp/gocryptotrader/backtest/position/fill"
)

func (p *Position) Create(in fill.Handler) {
	p.Time = in.Time()
	p.Pair = in.Pair()

	p.update(in)
}

func (p *Position) Update(f fill.Handler) {
	p.Time = f.Time()
	p.update(f)
}

func (p *Position) update(f fill.Handler) {
	if f.Direction() == event.BUY {
		if p.Amount > 0 {
			p.PriceBasis += f.NetValue()
		} else {
			p.PriceBasis += math.Abs(f.Amount()) / p.Amount * p.PriceBasis
			p.RealisedPNL += f.Amount()*(p.AveragePrice-f.Price()) - f.Cost()
		}

		p.AveragePriceBought = ((p.AmountBought * p.AveragePriceBought) + (f.Amount() * f.Price())) / (p.AmountBought + f.Amount())

		p.Amount += f.Amount()
		p.AmountBought += f.Amount()

		p.Value = p.AmountBought * p.AveragePriceBought
		p.NetValueBUY += f.NetValue()
	} else if f.Direction() == event.SELL {
		if p.Amount > 0 {
			p.PriceBasis -= math.Abs(f.Amount()) / p.Amount * p.PriceBasis
			p.RealisedPNL -= f.Amount()*(p.AveragePrice-f.Price()) - f.Cost()
		} else {
			p.PriceBasis -= f.NetValue()
		}

		p.AveragePriceSold = ((p.AmountSold * p.AveragePriceSold) + (f.Amount() * f.Price())) / (p.AmountSold + f.Amount())

		p.Amount -= f.Amount()
		p.AmountSold += f.Amount()

		p.Value = p.AmountSold * p.AveragePriceSold
		p.NetValueBUY += f.NetValue()
	} else {
		// todo handle stuff
	}

	p.AveragePrice = (math.Abs(p.Amount) * p.AveragePrice) + (f.Amount()*f.Price())/(math.Abs(p.Amount)+f.Amount())
	p.AveragePriceNet = (math.Abs(p.Amount)*p.AveragePriceNet + f.NetValue()) / (math.Abs(p.Amount) + f.Amount())

}
