package backtest

type Data struct {
	latest        map[string]DataEventHandler
	list          map[string][]DataEventHandler
	stream        []DataEventHandler
	streamHistory []DataEventHandler
}
