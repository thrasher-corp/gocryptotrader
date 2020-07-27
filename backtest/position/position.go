package position

import "github.com/thrasher-corp/gocryptotrader/backtest/position/fill"

func (p *Position) Create(in fill.Event) {
	p.Time = in.Time()
}
