package report

import (
	"errors"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// GenerateReport sends final data from statistics to a template
// to create a lovely final report for someone to view
func (d *Data) GenerateReport() error {
	err := d.enhanceCandles()
	if err != nil {
		return err
	}

	for i := range d.EnhancedCandles {
		cands := d.EnhancedCandles[i].Candles
		sort.Slice(cands, func(x, y int) bool {
			return cands[x].Time < cands[y].Time
		})
		if len(d.EnhancedCandles[i].Candles) >= maxChartLimit {
			d.EnhancedCandles[i].IsOverLimit = true
			d.EnhancedCandles[i].Candles = d.EnhancedCandles[i].Candles[:maxChartLimit]
		}
	}

	tmpl := template.Must(
		template.ParseFiles(
			filepath.Join(d.TemplatePath),
		),
	)
	var nickName string
	if d.Config.Nickname != "" {
		nickName = d.Config.Nickname + "-"
	}
	var f *os.File
	f, err = os.Create(
		filepath.Join(d.OutputPath,
			fmt.Sprintf(
				"%v%v-%v.html",
				nickName,
				d.Statistics.StrategyName,
				time.Now().Format("2006-01-02-15-04-05"),
			),
		),
	)
	if err != nil {
		return err
	}
	defer func() {
		err = f.Close()
		if err != nil {
			log.Error(log.BackTester, err)
		}
	}()

	err = tmpl.Execute(f, d)
	if err != nil {
		return err
	}

	return nil
}

// AddKlineItem appends a SET of candles for the report to enhance upon
// generation
func (d *Data) AddKlineItem(k *kline.Item) {
	d.OriginalCandles = append(d.OriginalCandles, k)
}

// enhanceCandles will enhance candle data with order information allowing
// report charts to have annotations to highlight buy and sell events
func (d *Data) enhanceCandles() error {
	if len(d.OriginalCandles) == 0 {
		return errors.New("no candles to enhance")
	}
	if d.Statistics == nil {
		return errors.New("unable to proceed with unset Statistics property")
	}
	d.Statistics.RiskFreeRate *= 100

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
			for k := range statsForCandles.Events {
				if statsForCandles.Events[k].SignalEvent != nil {
					if statsForCandles.Events[k].SignalEvent.GetTime().Equal(d.OriginalCandles[i].Candles[j].Time) {
						if statsForCandles.Events[k].SignalEvent.GetDirection() == common.MissingData {
							// if the data is missing, ensure that all values just continue the previous candle's close price visually
							enhancedCandle.Open = enhancedKline.Candles[len(enhancedKline.Candles)-1].Close
							enhancedCandle.High = enhancedKline.Candles[len(enhancedKline.Candles)-1].Close
							enhancedCandle.Low = enhancedKline.Candles[len(enhancedKline.Candles)-1].Close
							enhancedCandle.Close = enhancedKline.Candles[len(enhancedKline.Candles)-1].Close

							enhancedCandle.Colour = "white"
							enhancedCandle.Position = "aboveBar"
							enhancedCandle.Shape = "arrowDown"
							enhancedCandle.Text = common.MissingData.String()
						}
					}
				}
			}
			for k := range statsForCandles.FinalOrders.Orders {
				if statsForCandles.FinalOrders.Orders[k].Detail == nil {
					continue
				}
				if statsForCandles.FinalOrders.Orders[k].Date.Equal(
					d.OriginalCandles[i].Candles[j].Time) {
					// an order was placed here, can enhance chart!
					enhancedCandle.MadeOrder = true
					enhancedCandle.OrderAmount = statsForCandles.FinalOrders.Orders[k].Amount
					enhancedCandle.PurchasePrice = statsForCandles.FinalOrders.Orders[k].Price
					enhancedCandle.OrderDirection = statsForCandles.FinalOrders.Orders[k].Side
					if enhancedCandle.OrderDirection == order.Buy {
						enhancedCandle.Colour = "green"
						enhancedCandle.Position = "aboveBar"
						enhancedCandle.Shape = "arrowDown"
					} else if enhancedCandle.OrderDirection == order.Sell {
						enhancedCandle.Colour = "red"
						enhancedCandle.Position = "belowBar"
						enhancedCandle.Shape = "arrowUp"
					}
					enhancedCandle.Text = enhancedCandle.OrderDirection.String()
					break
				}
			}
			enhancedKline.Candles = append(enhancedKline.Candles, enhancedCandle)
		}
		d.EnhancedCandles = append(d.EnhancedCandles, enhancedKline)
	}

	return nil
}
