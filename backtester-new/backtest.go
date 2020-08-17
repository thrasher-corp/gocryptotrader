package backtest

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
		b.portfolio.Update(d)
		_, err := b.execution.OnData(d, b)
		if err != nil {
			return err
		}
		_, err = b.algo.OnData(d, b)
		if err != nil {
			return err
		}
	}

	return nil
}

func (b *Backtest) setup() error {
	return nil
}

func (b *Backtest) GetPortfolio() (portfolio PortfolioHandler) {
	return b.portfolio
}

func Run(algo AlgoHandler, data DataHandler) error {
	bt := &Backtest{}

	bt.data = data
	bt.algo = algo
	if err := bt.Run(); err != nil {
		return err
	}

	algo.OnEnd(bt)
	return nil
}
