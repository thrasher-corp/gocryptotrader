package backtest

import (
	portfolio "github.com/thrasher-corp/gocryptotrader/backtester/datahandler"
	"github.com/thrasher-corp/gocryptotrader/backtester/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/orderbook"
	"github.com/thrasher-corp/gocryptotrader/backtester/signal"
)

// New returns a new BackTest instance
func New() *BackTest {
	return &BackTest{}
}

// Reset BackTest values to default
func (t *BackTest) Reset() {
	t.EventQueue = nil
	t.Data.Reset()
	t.Portfolio.Reset()
	t.Statistic.Reset()
}

// Run executes Backtest
func (t *BackTest) Run() error {
	t.Portfolio.SetFunds(t.Portfolio.GetInitialFunds())
	for event, ok := t.nextEvent(); true; event, ok = t.nextEvent() {
		if !ok {
			data, ok := t.Data.Next()
			if !ok {
				break
			}
			t.EventQueue = append(t.EventQueue, data)
			continue
		}

		err := t.eventLoop(event)
		if err != nil {
			return err
		}
		t.Statistic.TrackEvent(event)
	}

	return nil
}

func (t *BackTest) nextEvent() (e portfolio.EventHandler, ok bool) {
	if len(t.EventQueue) == 0 {
		return e, false
	}

	e = t.EventQueue[0]
	t.EventQueue = t.EventQueue[1:]

	return e, true
}

func (t *BackTest) eventLoop(e portfolio.EventHandler) error {
	switch event := e.(type) {
	case portfolio.DataEventHandler:
		t.Portfolio.Update(event)
		t.Statistic.Update(event, t.Portfolio)

		signal, err := t.Strategy.OnSignal(t.Data, t.Portfolio)
		if err != nil {
			break
		}
		t.EventQueue = append(t.EventQueue, signal)

	case signal.SignalEvent:
		order, err := t.Portfolio.OnSignal(event, t.Data)
		if err != nil {
			break
		}
		t.EventQueue = append(t.EventQueue, order)

	case orderbook.OrderEvent:
		fill, err := t.Exchange.ExecuteOrder(event, t.Data)
		if err != nil {
			break
		}
		t.Orderbook.Add(event)
		t.EventQueue = append(t.EventQueue, fill)
	case fill.FillEvent:
		transaction, err := t.Portfolio.OnFill(event, t.Data)
		if err != nil {
			break
		}
		t.Statistic.TrackTransaction(transaction)
	}

	return nil
}
