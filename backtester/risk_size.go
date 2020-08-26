package backtest

func (r *Risk) EvaluateOrder(order OrderEvent, _ DataEventHandler, _ map[string]Positions) (*Order, error) {
	return order.(*Order), nil
}

func (s *Size) SizeOrder(orderevent OrderEvent, _ DataEventHandler, _ PortfolioHandler) (*Order, error) {
	return orderevent.(*Order), nil
}
