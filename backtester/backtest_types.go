package backtest

const DP = 4

type BackTest struct {
	data       DataHandler
	strategy   StrategyHandler
	portfolio  PortfolioHandler
	exchange   ExecutionHandler
	statistic  StatisticHandler
	eventQueue []EventHandler
}
