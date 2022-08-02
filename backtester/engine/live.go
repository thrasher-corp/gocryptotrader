package engine

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// RunLive is a proof of concept function that does not yet support multi currency usage
// It runs by constantly checking for new live datas and running through the list of events
// once new data is processed. It will run until application close event has been received
func (bt *BackTest) RunLive() error {
	log.Info(common.Livetester, "running backtester against live data")
	for {
		select {
		case <-bt.shutdown:
			return nil
		case <-bt.LiveDataHandler.Updated():
			bt.Run()
		}
	}
}
