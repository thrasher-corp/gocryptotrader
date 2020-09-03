package backtest

const DP = 8

type BackTest struct {
	data       DataHandler
	strategy   StrategyHandler
	portfolio  PortfolioHandler
	orderbook  OrderBook
	exchange   ExecutionHandler
	statistic  StatisticHandler
	eventQueue []EventHandler
}
