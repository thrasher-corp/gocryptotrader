package backtest

import "fmt"

func (b *Backtest) Reset() error {
	b.dataProvider.Reset()
	return nil
}

func (b *Backtest) Run() error {
	err := b.setup()
	if err != nil {
		return err
	}

	for d, ok := b.dataProvider.Next(); ok; d, ok = b.dataProvider.Next() {
		fmt.Println(d)
	}

	return nil
}

func (b *Backtest) setup() error {
	return nil
}

