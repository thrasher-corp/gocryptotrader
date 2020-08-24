package backtest

import (
	"fmt"
)

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

	return nil
}

func (b *Backtest) setup() error {
	b.Portfolio.SetFunds(b.Portfolio.InitialFunds())

	return nil
}

func (b *Backtest) GetPortfolio() (portfolio PortfolioHandler) {
	return b.Portfolio
}

func Run(algo AlgoHandler) {
	config := algo.Init()

	backTest := &Backtest{
		config: config,
	}

	backTest.Portfolio = &Portfolio{
		initialFunds: 10000,
	}

	feeHandler := &PercentageFee{
		ExchangeFee{
			0.85,
		},
	}

	backTest.Execution = &Execution{
		ExchangeFee: feeHandler,
	}

	backTest.Stats = &Statistic{}

	data := &DataFromKlineItem{
		Item: backTest.config.Item,
	}

	data.Load()
	backTest.data = data
	backTest.Algo = algo

	err := backTest.Run()
	if err != nil {
		fmt.Printf("err: %v", err)
	}

	algo.OnEnd(backTest)
}

