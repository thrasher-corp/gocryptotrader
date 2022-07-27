package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/config"
	"github.com/thrasher-corp/gocryptotrader/backtester/data/kline"
	"github.com/thrasher-corp/gocryptotrader/backtester/data/kline/live"
	"github.com/thrasher-corp/gocryptotrader/currency"
	gctexchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// RunLive is a proof of concept function that does not yet support multi currency usage
// It runs by constantly checking for new live datas and running through the list of events
// once new data is processed. It will run until application close event has been received
func (bt *BackTest) RunLive() error {
	log.Info(common.Livetester, "running backtester against live data")
	timer := bt.EventQueue.GetRunTimer()
	timeout := bt.EventQueue.GetNewEventTimeout()
	timeoutTimer := time.NewTimer(timeout)
	// a frequent timer so that when a new candle is released by an exchange
	// that it can be processed quickly
	processEventTicker := time.NewTimer(0)
	for {
		select {
		case <-bt.shutdown:
			return nil
		case <-timeoutTimer.C:
			return fmt.Errorf("%w %v", errLiveDataTimeout, timeout)
		case <-processEventTicker.C:
			bt.Run()
			processEventTicker.Reset(timer)
			timeoutTimer.Reset(timeout)
		}
	}
}

// loadLiveDataLoop is an incomplete function to continuously retrieve exchange data on a loop
// from live. Its purpose is to be able to perform strategy analysis against current data
func (bt *BackTest) loadLiveDataLoop(resp *kline.DataFromKline, cfg *config.Config, exch gctexchange.IBotExchange, fPair currency.Pair, a asset.Item, dataType int64) {
	loadNewDataTimer := time.NewTimer(0)
	dataCheckInterval := bt.EventQueue.GetDataCheckTimer()
	var err error
	for {
		select {
		case <-bt.shutdown:
			return
		case <-loadNewDataTimer.C:
			if cfg.DataSettings.LiveData.VerboseDataCheck {
				log.Infof(common.Livetester, "%v has passed, fetching data for %v %v %v ", dataCheckInterval, exch.GetName(), a, fPair)
			}
			err = bt.loadLiveData(resp, cfg, exch, fPair, a, dataType)
			if err != nil {
				log.Error(common.Livetester, err)
				return
			}
			loadNewDataTimer.Reset(dataCheckInterval)
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
	if !resp.IsLive() {
		resp.SetLive(true)
	}
	err = bt.Reports.UpdateItem(&resp.Item)
	if err != nil {
		return err
	}
	return nil
}
