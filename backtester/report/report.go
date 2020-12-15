package report

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
)

// GenerateReport sends final data from statistics to a template
// to create a lovely final report for someone to view
func (d *Data) GenerateReport() error {
	tmpl := template.Must(template.ParseFiles("tpl.gohtml"))
	file, err := os.Create(
		filepath.Join(
			"..",
			"results",
			fmt.Sprintf(
				"%v%v.html",
				d.Statistics.StrategyName,
				"", /*time.Now().Format("2006-01-02-15-04-05")*/
			),
		),
	)
	if err != nil {
		return err
	}

	err = tmpl.Execute(file, d)
	if err != nil {
		return err
	}

	return nil
}

// enhanceCandles will enhance candle data with order information allowing
// report charts to have annotations to highlight buy and sell events
func (d *Data) enhanceCandles() error {
	for i := range d.OriginalCandles {
		lookup := d.OriginalCandles[i]
		enhancedKline := DetailedKline{
			Exchange: lookup.Exchange,
			Asset:    lookup.Asset,
			Pair:     lookup.Pair,
			Interval: lookup.Interval,
		}

		statsForCandles :=
			d.Statistics.ExchangeAssetPairStatistics[lookup.Exchange][lookup.Asset][lookup.Pair]

		for j := range d.OriginalCandles[i].Candles {
			enhancedCandle := DetailedCandle{
				Time:   d.OriginalCandles[i].Candles[j].Time,
				Open:   d.OriginalCandles[i].Candles[j].Open,
				High:   d.OriginalCandles[i].Candles[j].High,
				Low:    d.OriginalCandles[i].Candles[j].Low,
				Close:  d.OriginalCandles[i].Candles[j].Close,
				Volume: d.OriginalCandles[i].Candles[j].Volume,
			}
			for k := range statsForCandles.Orders.Orders {
				if statsForCandles.Orders.Orders[k].Date.Equal(
					d.OriginalCandles[i].Candles[j].Time) {
					// an order was placed here, can enhance chart!
					enhancedCandle.MadeOrder = true
					enhancedCandle.OrderAmount = statsForCandles.Orders.Orders[k].Amount
					enhancedCandle.PurchasePrice = statsForCandles.Orders.Orders[k].Price
					enhancedCandle.OrderDirection = statsForCandles.Orders.Orders[k].Side
				}
			}
			enhancedKline.Candles = append(enhancedKline.Candles, enhancedCandle)
		}
		d.Candles = append(d.Candles, enhancedKline)
	}

	return nil
}
