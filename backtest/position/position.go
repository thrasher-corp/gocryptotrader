package position

import "github.com/thrasher-corp/gocryptotrader/backtest/position/fill"

func (p *Position) Create(in fill.Handler) {
	p.Time = in.Time()
}

func (p *Position) Update(f fill.Handler) {
	p.Time = f.Time()
	p.update(f)
}

func (p *Position) update(f fill.Handler) {

}
