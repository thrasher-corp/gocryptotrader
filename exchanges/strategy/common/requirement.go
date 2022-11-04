package common

import (
	"context"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

const SimulationTag = "SIMULATION"

// Requirement define baseline functionality for strategy implementation to GCT
type Requirement interface {
	Run(ctx context.Context) error
	Stop() error
	GetReporter() (Reporter, error)
	GetState() (*State, error)
	OnSignal(ctx context.Context, sig interface{}) (bool, error)

	String() string

	// WaitForSignal() <-chan interface{}
	// WaitForEnd() <-chan time.Time
	// WaitForShutdown() <-chan struct{}
}

// State defines basic identification for strategy
type State struct {
	ID         uuid.UUID
	Registered time.Time
	Exchange   string
	Pair       currency.Pair
	Asset      asset.Item
	Strategy   string
	Running    bool
}

// // Deploy oversees the deployment of the current strategy adhering to policies,
// // limits, signals and timings.
// func (s *State) Deploy(ctx context.Context, strat Requirement) error {
// 	// defer func() {
// 	// 	s.wg.Done()
// 	// 	s.mtx.Lock()
// 	// 	s.running = false
// 	// 	s.mtx.Unlock()
// 	// }()

// 	report, err := strat.GetReporter()
// 	if err != nil {
// 		return err
// 	}
// 	report.OnStart(strat)

// 	go func() {
// 		for {
// 			select {
// 			case sig := <-strat.WaitForSignal():
// 				var complete bool
// 				complete, err = strat.OnSignal(ctx, sig)
// 				if err != nil {
// 					report.OnFatalError(err)
// 					return
// 				}

// 				if complete {
// 					report.OnComplete()
// 					return
// 				}
// 			case end := <-strat.WaitForEnd():
// 				report.OnTimeout(end)
// 				return
// 			case <-ctx.Done():
// 				report.OnContextDone(ctx.Err())
// 				return
// 			case <-strat.WaitForShutdown():
// 				report.OnShutdown()
// 				return
// 			}
// 		}
// 	}()
// 	return nil
// }
