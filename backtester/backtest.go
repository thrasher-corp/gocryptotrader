package backtest

const DP = 4 // DP

type Reseter interface {
	Reset() error
}

type Backtest struct {
	dataProvider DataHandler
	portfolio    PortfolioHandler
	exchange     ExecutionHandler
	statistic    StatisticHandler

	algos AlgoHandler

	config *AppConfig
}

var backTest *Backtest

func GetBackTest(config *AppConfig) *Backtest {
	if backTest != nil {
		return backTest
	}
	backTest = &Backtest{
		config: config,
	}

	backTest.portfolio = NewPortfolio()
	backTest.exchange = NewExchange()
	backTest.statistic = &Statistic{}

	return backTest
}

func (b *Backtest) SetAlgo(algo AlgoHandler) *Backtest {
	b.algos = algo
	return b
}

func (b *Backtest) SetDataProvider(data DataHandler) {
	b.dataProvider = data
}

func (b *Backtest) GetDataProvider() (DataHandler, bool) {
	if b.dataProvider == nil {
		return nil, false
	}

	return b.dataProvider, true
}

func (b *Backtest) SetPortfolio(portfolio PortfolioHandler) {
	b.portfolio = portfolio
}

func (b *Backtest) GetPortfolio() (portfolio PortfolioHandler) {
	return b.portfolio
}

func (b *Backtest) SetExchange(exchange ExecutionHandler) {
	b.exchange = exchange
}

func (b *Backtest) SetStatistic(statistic StatisticHandler) {
	b.statistic = statistic
}

func (b *Backtest) GetStats() StatisticHandler {
	return b.statistic
}

func (b *Backtest) Reset() error {
	b.dataProvider.Reset()
	b.portfolio.Reset()
	b.statistic.Reset()
	return nil
}

func (b *Backtest) Run() error {
	err := b.setup()
	if err != nil {
		return err
	}

	for ticker, ok := b.dataProvider.Next(); ok; ticker, ok = b.dataProvider.Next() {

		b.portfolio.Update(ticker)
		b.statistic.Update(ticker, b.portfolio)

		b.exchange.OnData(ticker, b)

		b.statistic.TrackEvent(ticker)

		b.algos.OnData(ticker, b)

	}

	err = b.teardown()
	if err != nil {
		return err
	}

	return nil
}

func (b *Backtest) setup() error {
	b.portfolio.SetCash(b.portfolio.InitialCash())

	return nil
}

func (b *Backtest) teardown() error {
	return nil
}

func (b *Backtest) getConfig() *AppConfig {
	return b.config
}
