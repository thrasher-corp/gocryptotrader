package backtest

func New() *BackTest {
	return &BackTest{}
}

func (t *BackTest) Reset() {
	t.eventQueue = nil
	t.data.Reset()
	t.portfolio.Reset()
	t.statistic.Reset()
}

func (t *BackTest) Stats() StatisticHandler {
	return t.statistic
}

func (t *BackTest) Run() error {
	t.portfolio.SetFunds(t.portfolio.InitialFunds())
	for event, ok := t.nextEvent(); true; event, ok = t.nextEvent() {
		if !ok {
			data, ok := t.data.Next()
			if !ok {
				break
			}
			t.eventQueue = append(t.eventQueue, data)
			continue
		}

		err := t.eventLoop(event)
		if err != nil {
			return err
		}
		t.statistic.TrackEvent(event)
	}

	return nil
}

func (t *BackTest) nextEvent() (e EventHandler, ok bool) {
	if len(t.eventQueue) == 0 {
		return e, false
	}

	e = t.eventQueue[0]
	t.eventQueue = t.eventQueue[1:]

	return e, true
}

func (t *BackTest) eventLoop(e EventHandler) error {
	switch event := e.(type) {
	case DataEventHandler:
		t.portfolio.Update(event)
		t.statistic.Update(event, t.portfolio)

		signal, err := t.strategy.OnSignal(t.data, t.portfolio)
		if err != nil {
			break
		}
		t.eventQueue = append(t.eventQueue, signal)

	case SignalEvent:
		order, err := t.portfolio.OnSignal(event, t.data)
		if err != nil {
			break
		}
		t.eventQueue = append(t.eventQueue, order)

	case OrderEvent:
		fill, err := t.exchange.ExecuteOrder(event, t.data)
		if err != nil {
			break
		}
		t.eventQueue = append(t.eventQueue, fill)
	case FillEvent:
		transaction, err := t.portfolio.OnFill(event, t.data)
		if err != nil {
			break
		}
		t.statistic.TrackTransaction(transaction)
	}

	return nil
}
