package engine

import (
	"context"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/config"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/data/kline"
	"github.com/thrasher-corp/gocryptotrader/backtester/data/kline/live"
	"github.com/thrasher-corp/gocryptotrader/currency"
	gctexchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// RunLive is a proof of concept function that does not yet support multi currency usage
// It runs by constantly checking for new live datas and running through the list of events
// once new data is processed. It will run until application close event has been received
func (bt *BackTest) RunLive() error {
	log.Info(common.Backtester, "running backtester against live data")
	timeoutTimer := time.NewTimer(time.Minute * 5)
	// a frequent timer so that when a new candle is released by an exchange
	// that it can be processed quickly
	processEventTicker := time.NewTicker(time.Second)
	doneARun := false
	for {
		select {
		case <-bt.shutdown:
			return nil
		case <-timeoutTimer.C:
			return errLiveDataTimeout
		case <-processEventTicker.C:
			for e := bt.EventQueue.NextEvent(); ; e = bt.EventQueue.NextEvent() {
				if e == nil {
					// as live only supports singular currency, just get the proper reference manually
					var d data.Handler
					dd := bt.Datas.GetAllData()
					for k1, v1 := range dd {
						for k2, v2 := range v1 {
							for k3 := range v2 {
								d = dd[k1][k2][k3]
							}
						}
					}
					de := d.Next()
					if de == nil {
						break
					}

					bt.EventQueue.AppendEvent(de)
					doneARun = true
					continue
				}
				err := bt.handleEvent(e)
				if err != nil {
					return err
				}
			}
			if doneARun {
				timeoutTimer = time.NewTimer(time.Minute * 5)
			}
		}
	}
}

// loadLiveDataLoop is an incomplete function to continuously retrieve exchange data on a loop
// from live. Its purpose is to be able to perform strategy analysis against current data
func (bt *BackTest) loadLiveDataLoop(resp *kline.DataFromKline, cfg *config.Config, exch gctexchange.IBotExchange, fPair currency.Pair, a asset.Item, dataType int64) {
	startDate := time.Now().Add(-cfg.DataSettings.Interval.Duration() * 2)
	dates, err := gctkline.CalculateCandleDateRanges(
		startDate,
		startDate.AddDate(1, 0, 0),
		gctkline.Interval(cfg.DataSettings.Interval),
		0)
	if err != nil {
		log.Errorf(common.Backtester, "%v. Please check your GoCryptoTrader configuration", err)
		return
	}
	candles, err := live.LoadData(context.TODO(),
		exch,
		dataType,
		cfg.DataSettings.Interval.Duration(),
		fPair,
		a)
	if err != nil {
		log.Errorf(common.Backtester, "%v. Please check your GoCryptoTrader configuration", err)
		return
	}
	dates.SetHasDataFromCandles(candles.Candles)
	resp.RangeHolder = dates
	resp.Item = *candles

	loadNewDataTimer := time.NewTimer(time.Second * 5)
	for {
		select {
		case <-bt.shutdown:
			return
		case <-loadNewDataTimer.C:
			log.Infof(common.Backtester, "fetching data for %v %v %v %v", exch.GetName(), a, fPair, cfg.DataSettings.Interval)
			loadNewDataTimer.Reset(time.Second * 15)
			err = bt.loadLiveData(resp, cfg, exch, fPair, a, dataType)
			if err != nil {
				log.Error(common.Backtester, err)
				return
			}
		}
	}
}

func (bt *BackTest) loadLiveData(resp *kline.DataFromKline, cfg *config.Config, exch gctexchange.IBotExchange, fPair currency.Pair, a asset.Item, dataType int64) error {
	if resp == nil {
		return errNilData
	}
	if cfg == nil {
		return errNilConfig
	}
	if exch == nil {
		return errNilExchange
	}
	candles, err := live.LoadData(context.TODO(),
		exch,
		dataType,
		cfg.DataSettings.Interval.Duration(),
		fPair,
		a)
	if err != nil {
		return err
	}
	if len(candles.Candles) == 0 {
		return nil
	}
	resp.AppendResults(candles)
	bt.Reports.UpdateItem(&resp.Item)
	log.Info(common.Backtester, "sleeping for 30 seconds before checking for new candle data")
	return nil
}
