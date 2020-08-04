package backtest

type DataFromKlineItem struct {
	latest DataHandler
	stream []DataHandler
}

func (d *DataFromKlineItem) Reset() {
	d.latest = nil
}

func (d *DataFromKlineItem) Next() (DataEvent, bool) {
	return nil, false
}

func (d *DataFromKlineItem) Stream() []DataEvent {
	return nil
}

func (d *DataFromKlineItem) History() []DataEvent {
	return nil
}

func (d *DataFromKlineItem) Latest() DataEvent {
	return nil
}

func (d *DataFromKlineItem) Load() {

}