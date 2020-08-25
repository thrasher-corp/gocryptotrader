package backtest

func (b *Backtest) Reset() error {
	b.data.Reset()
	_ = b.Portfolio.Reset()
	b.Stats.Reset()
	return nil
}

func (b *Backtest) Run() error {
	err := b.setup()
	if err != nil {
		return err
	}

	for d, ok := b.data.Next(); ok; d, ok = b.data.Next() {
		b.Portfolio.Update(d)
		b.Stats.Update(d, b.Portfolio)
		_, err := b.Execution.OnData(d, b)
		if err != nil {
			return err
		}

		b.Stats.TrackEvent(d)
		_, err = b.Algo.OnData(d, b)
		if err != nil {
			return err
		}
	}

	b.Algo.OnEnd(b)

	return nil
}

func (b *Backtest) setup() error {
	b.Portfolio.SetFunds(b.Portfolio.InitialFunds())

	return nil
}

func (b *Backtest) GetPortfolio() (portfolio PortfolioHandler) {
	return b.Portfolio
}

func New(algo AlgoHandler) (*Backtest, error) {
	cfg := algo.Init()

	itemData := &DataFromKlineItem{
		Item: cfg.Item,
	}
	err := itemData.Load()
	if err != nil {
		return nil, err
	}

	backTest := &Backtest{
		config: cfg,
		Portfolio: &Portfolio{
			initialFunds: cfg.InitialFunds,
		},
		Execution: &Execution{
			ExchangeFee: &PercentageFee{
				ExchangeFee{
					Fee: cfg.Fee,
				},
			},
		},
		Stats: &Statistic{},
		Algo:  algo,
		data:  itemData,
	}

	return backTest, nil
}
