package backtest

type Portfolio struct {
	initialCash  float64
	cash         float64
	holdings     map[string]Positions
	transactions []FillEvent
	sizeManager  SizeHandler
	riskManager  RiskHandler
}
