package twap

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	strategy "github.com/thrasher-corp/gocryptotrader/exchanges/strategy/common"
)

const (
	endTimeLapse = "Stategy has lapsed end time"
)

var (
	errNoBalanceFound     = errors.New("no balance found")
	errExceedsFreeBalance = errors.New("amount exceeds current free balance")
	errInvalidAssetType   = errors.New("non spot trading pairs not currently supported")
	errStrategyIsNil      = errors.New("strategy is nil")
)

// GetTWAP returns a TWAP struct to manage allocation or deallocation of
// position(s).
func New(ctx context.Context, c *Config) (*Strategy, error) {
	err := c.Check(ctx)
	if err != nil {
		return nil, err
	}

	if c.Asset != asset.Spot {
		return nil, errInvalidAssetType
	}

	depth, err := orderbook.GetDepth(c.Exchange.GetName(), c.Pair, c.Asset)
	if err != nil {
		return nil, err
	}

	creds, err := c.Exchange.GetCredentials(ctx)
	if err != nil {
		return nil, err
	}

	buying, err := account.GetBalance(c.Exchange.GetName(),
		creds.SubAccount, creds, c.Asset, c.Pair.Base)
	if err != nil {
		return nil, err
	}

	deployment := c.Pair.Quote
	selling, err := account.GetBalance(c.Exchange.GetName(),
		creds.SubAccount, creds, c.Asset, c.Pair.Quote)
	if err != nil {
		return nil, err
	}

	if !c.Buy {
		buying, selling = selling, buying
		deployment = c.Pair.Base
	}

	fullDeployment := selling.GetAvailableWithoutBorrow()
	if fullDeployment == 0 {
		return nil, fmt.Errorf("cannot sell %s amount %f to buy base %s %w of %f",
			deployment,
			c.Amount,
			c.Pair.Base,
			errNoBalanceFound,
			fullDeployment)
	}

	if !c.FullAmount {
		if c.Amount > fullDeployment {
			return nil, fmt.Errorf("cannot sell %s amount %f to buy base %s %w of %f",
				deployment,
				c.Amount,
				c.Pair.Base,
				errExceedsFreeBalance,
				fullDeployment)
		}
		fullDeployment = c.Amount
	}

	// NOTE: For now this will not allow any amount to deplete the full
	// orderbook, just until a safe, effective and efficient system has been
	// tested and deployed. TODO: Bypass error
	// errBookSmallerThanDeploymentAmount.
	allocation, err := c.GetDistrbutionAmount(fullDeployment, depth)
	if err != nil {
		return nil, err
	}

	return &Strategy{
		Config:     c,
		orderbook:  depth,
		Buying:     buying,
		Selling:    selling,
		allocation: allocation,
	}, nil
}

// deploy oversees the deployment of the current strategy adhering to policies,
// limits, signals and timings.
func (s *Strategy) deploy(ctx context.Context, start time.Duration) {
	defer func() {
		s.wg.Done()
		s.mtx.Lock()
		s.running = false
		s.mtx.Unlock()
	}()
	wow := fmt.Sprintf("Starting twap operation in %s...\n", start)
	s.reporter.Send(&strategy.Report{Reason: wow})
	// NOTE: Zero value start duration will execute immediately then deploy at
	// intervals.
	timer := time.NewTimer(start)
	finished := time.NewTimer(time.Until(s.End))
	for {
		select {
		case <-timer.C:
			if !s.AllowTradingPastEndTime && time.Now().After(s.End) {
				s.reporter.Send(&strategy.Report{Reason: endTimeLapse, Finished: true})
				return
			}

			err := s.SetTimer(timer)
			if err != nil {
				s.reporter.Send(&strategy.Report{Error: err, Finished: true})
				return
			}

			err = s.checkAndSubmit(ctx)
			if err != nil {
				s.reporter.Send(&strategy.Report{Error: err, Finished: true})
				return
			}

			if s.allocation.Deployed == s.allocation.Total {
				s.reporter.Send(&strategy.Report{Reason: "SIMULATION COMPLETED", Finished: true})
				return
			}

			if !s.AllowTradingPastEndTime && time.Now().After(s.End) {
				s.reporter.Send(&strategy.Report{Reason: endTimeLapse, Finished: true})
				return
			}
		case <-finished.C:
			s.reporter.Send(&strategy.Report{Reason: endTimeLapse, Finished: true})
			return
		case <-ctx.Done():
			s.reporter.Send(&strategy.Report{Error: ctx.Err(), Finished: true})
			return
		case <-s.shutdown:
			s.reporter.Send(&strategy.Report{Reason: "Shutdown called on strategy", Finished: true})
			return
		}
	}
}

// checkAndSubmit verifies orderbook deployability then executes an order if
// all checks pass.
func (s *Strategy) checkAndSubmit(ctx context.Context) error {
	if s == nil {
		return errStrategyIsNil
	}

	deploymentInBase, details, err := s.VerifyBookDeployment(s.orderbook, s.allocation.Deployment)
	if err != nil {
		return err
	}

	conformed, err := s.VerifyExecutionLimitsReturnConformed(deploymentInBase)
	if err != nil {
		return err
	}

	submit, err := s.deriveOrder(conformed)
	if err != nil {
		return err
	}

	resp, err := s.submitOrder(ctx, submit)
	if err != nil {
		return err
	}

	// Note: For first iteration of strategy this is just easy reconciliation.
	// TODO: Reconcile to adjusted amount.
	s.allocation.Deployed += s.allocation.Deployment
	s.allocation.Deployments++

	s.reporter.Send(&strategy.Report{
		Submit:     submit,
		Response:   resp,
		Deployment: details,
	})

	return nil
}

// deriveOrder checks amount and returns an order submission. TODO: Abstract
// futher.
func (s *Strategy) deriveOrder(amountInBase float64) (*order.Submit, error) {
	if amountInBase <= 0 {
		return nil, errInvalidAllocatedAmount
	}
	side := order.Buy
	if !s.Buy {
		side = order.Sell
	}
	return &order.Submit{
		Exchange:  s.Exchange.GetName(),
		Type:      order.Market,
		Pair:      s.Pair,
		AssetType: s.Asset,
		// ReduceOnly: true, // Have reduce only as default for this strategy.
		Side:   side,
		Amount: amountInBase,
	}, nil
}

// submitOrder will submit and retry an order if fail. TODO: Abstract futher
func (s *Strategy) submitOrder(ctx context.Context, submit *order.Submit) (*order.SubmitResponse, error) {
	if submit == nil {
		return nil, errors.New("submit order is invalid")
	}
	var errors common.Errors
	var resp *order.SubmitResponse
	for attempt := 0; attempt < int(s.RetryAttempts); attempt++ {
		// Check context here so we can immediately bypass the retry attempt and
		// release resources.
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		var err error
		if !s.Simulate {
			resp, err = s.Exchange.SubmitOrder(ctx, submit)
		} else {
			resp, err = submit.DeriveSubmitResponse(strategy.SimulationTag)
		}
		if err == nil {
			errors = nil // These errors prior we don't need to worry about.
			break
		}
		errors = append(errors, err)
		time.Sleep(time.Second)
	}
	var errReturn error
	if errors != nil {
		errReturn = errors
	}
	return resp, errReturn
}

// if !s.Simulate {
// wait, cancel, err := s.Selling.Wait(0)
// if err != nil {
// 	s.reporter.Send(&strategy.Report{Error: err, Finished: true})
// 	return
// }

// var timedOut bool
// select {
// case timedOut = <-wait:
// case <-ctx.Done():
// 	select {
// 	case cancel <- struct{}{}:
// 	default:
// 	}
// 	s.reporter.Send(&strategy.Report{Error: ctx.Err(), Finished: true})
// 	return
// }

// if timedOut {
// 	// TODO: Logger output
// 	continue
// }

// afterOrderBalance := s.Selling.GetAvailableWithoutBorrow()

// if afterOrderBalance == 0 {
// 	s.reporter.Send(&strategy.Report{Reason: "Balance depleted", Finished: true})
// 	return
// }

// change := fmt.Sprintf("change received prev: %f, now: %f and change: %f\n",
// 	preOrderBalance,
// 	afterOrderBalance,
// 	preOrderBalance-afterOrderBalance,
// )

// s.reporter.Send(&strategy.Report{Reason: change})
// }
