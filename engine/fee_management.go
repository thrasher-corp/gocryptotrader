package engine

import (
	"context"
	"errors"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// StartFeeSynchronisationManager starts the fee synchronisation manager which
// will update the fee schedule for each exchange. This will also synchronise
// trading pairs for each asset type.
func (e *Engine) StartFeeSynchronisationManager(ctx context.Context) error {
	go func() {
		timer := time.NewTimer(0)
		for range timer.C {
			exchs := e.GetExchanges()
			for i := range exchs {
				if !exchs[i].IsRESTAuthenticationSupported() && !exchs[i].IsWebsocketAuthenticationSupported() {
					continue
				}

				err := exchs[i].UpdateTradablePairs(ctx, false)
				if err != nil {
					log.Errorf(log.ExchangeSys, "Failed to update tradable pairs for %s: %v\n", exchs[i].GetName(), err)
					continue
				}

				assets := exchs[i].GetAssetTypes(true)
				for x := range assets {
					err := exchs[i].SynchroniseFees(ctx, assets[x])
					if err != nil && !errors.Is(err, common.ErrNotYetImplemented) && !errors.Is(err, common.ErrFunctionNotSupported) {
						log.Errorf(log.ExchangeSys, "Fee synchronisation failed for %s %s: %v\n", exchs[i].GetName(), assets[x], err)
					}
				}
			}
			timer.Reset(time.Until(time.Now().Truncate(time.Hour).Add(time.Hour))) // Sync once per hour
		}
	}()
	return nil
}
