package backtest

import (
	"fmt"
)

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
		b.Portfolio.Update(d)
		_, err := b.Execution.OnData(d, b)
		if err != nil {
			return err
		}
		_, err = b.Algo.OnData(d, b)
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
	return b.Portfolio
}

func Run(algo AlgoHandler) {
	config := algo.Init()

	app := New(config)

	data := &DataFromKlineItem{
		Item: app.config.Item,
	}

	data.Load()
	app.data = data
	app.Algo = algo

	err := app.Run()
	if err != nil {
		fmt.Printf("err: %v", err)
	}

	algo.OnEnd(app)
}

func New(config *Config) *Backtest {
	backTest := &Backtest{
		config: config,
	}

	backTest.Portfolio = &Portfolio{
		initialFunds: 10000,
	}
	backTest.Execution = &Execution{}
	backTest.Stats = &Statistic{}

	return backTest
}
