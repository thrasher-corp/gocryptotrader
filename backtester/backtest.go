package backtest

import "fmt"

func (b *Backtest) Reset() error {
	b.data.Reset()
	return nil
}

func (b *Backtest) Run() error {
	err := b.setup()
	if err != nil {
		return err
	}

	for d, ok := b.data.Next(); ok; d, ok = b.data.Next() {
		fmt.Println(d)
	}

	return nil
}

func (b *Backtest) setup() error {
	return nil
}

func (b *Backtest) GetPortfolio() (portfolio PortfolioHandler) {
	return b.portfolio
}

func Run(algo AlgoHandler) error {
	bt := &Backtest{}

	klineData := DataFromKlineItem{}
	bt.data = &klineData
	bt.algo = algo
	if err := bt.Run(); err != nil {
		return err
	}

	algo.OnEnd(bt)
	return nil
}