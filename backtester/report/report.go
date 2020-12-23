package report

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// GenerateReport sends final data from statistics to a template
// to create a lovely final report for someone to view
func (d *Data) GenerateReport() error {
	err := d.enhanceCandles()
	if err != nil {
		return err
	}

	for i := range d.Candles {
		sort.Slice(d.Candles[i].Candles, func(x, y int) bool {
			return d.Candles[i].Candles[x].Time < d.Candles[i].Candles[y].Time
		})
		if len(d.Candles[i].Candles) >= maxChartLimit {
			d.Candles[i].IsOverLimit = true
			//		d.Candles[i].Candles = d.Candles[i].Candles[:maxChartLimit]
		}
	}

	curr, _ := os.Getwd()
	tmpl := template.Must(
		template.ParseFiles(
			filepath.Join(curr, "report", "tpl.gohtml"),
		),
	)
	file, err := os.Create(
		filepath.Join(
			curr,
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

func (d *Data) AddCandles(k *kline.Item) {
	d.OriginalCandles = append(d.OriginalCandles, k)
}

// enhanceCandles will enhance candle data with order information allowing
// report charts to have annotations to highlight buy and sell events
func (d *Data) enhanceCandles() error {
	for i := range d.OriginalCandles {
		lookup := d.OriginalCandles[i]
		enhancedKline := DetailedKline{
			Exchange:  lookup.Exchange,
			Asset:     lookup.Asset,
			Pair:      lookup.Pair,
			Interval:  lookup.Interval,
			Watermark: fmt.Sprintf("%v - %v - %v", strings.Title(lookup.Exchange), lookup.Asset.String(), strings.ToUpper(lookup.Pair.String())),
		}

		statsForCandles :=
			d.Statistics.ExchangeAssetPairStatistics[lookup.Exchange][lookup.Asset][lookup.Pair]
		if statsForCandles == nil {
			continue
		}
		for j := range d.OriginalCandles[i].Candles {
			enhancedCandle := DetailedCandle{
				Time:         d.OriginalCandles[i].Candles[j].Time.Unix(),
				Open:         d.OriginalCandles[i].Candles[j].Open,
				High:         d.OriginalCandles[i].Candles[j].High,
				Low:          d.OriginalCandles[i].Candles[j].Low,
				Close:        d.OriginalCandles[i].Candles[j].Close,
				Volume:       d.OriginalCandles[i].Candles[j].Volume,
				VolumeColour: "rgba(47, 194, 27, 0.8)",
			}
			if j != 0 {
				if d.OriginalCandles[i].Candles[j].Close < d.OriginalCandles[i].Candles[j-1].Close {
					enhancedCandle.VolumeColour = "rgba(252, 3, 3, 0.8)"
				}
			}
			for k := range statsForCandles.Orders.Orders {
				if statsForCandles.Orders.Orders[k].Date.Equal(
					d.OriginalCandles[i].Candles[j].Time) {
					// an order was placed here, can enhance chart!
					enhancedCandle.MadeOrder = true
					enhancedCandle.OrderAmount = statsForCandles.Orders.Orders[k].Amount
					enhancedCandle.PurchasePrice = statsForCandles.Orders.Orders[k].Price
					enhancedCandle.OrderDirection = statsForCandles.Orders.Orders[k].Side
					if enhancedCandle.OrderDirection == order.Buy {
						enhancedCandle.Colour = "green"
						enhancedCandle.Position = "aboveBar"
						enhancedCandle.Shape = "arrowDown"
					} else if enhancedCandle.OrderDirection == order.Sell {
						enhancedCandle.Colour = "red"
						enhancedCandle.Position = "belowBar"
						enhancedCandle.Shape = "arrowUp"
					}
					enhancedCandle.Text = fmt.Sprintf("%v", enhancedCandle.OrderDirection)
					break
				}
			}
			enhancedKline.Candles = append(enhancedKline.Candles, enhancedCandle)
		}
		d.Candles = append(d.Candles, enhancedKline)
	}

	return nil
}
