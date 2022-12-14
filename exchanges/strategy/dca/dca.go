package dca

import (
	"context"
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	strategy "github.com/thrasher-corp/gocryptotrader/exchanges/strategy/common"
)

// New returns a struct that implements the Requirements interface to
// manage allocation or deallocation of position(s) using the dollar cost
// average (DCA) strategy.
func New(ctx context.Context, c *Config) (strategy.Requirements, error) {
	err := c.Check(ctx)
	if err != nil {
		return nil, err
	}

	// NOTE: Other asset types currently not supported by this strategy
	// TODO: Add support.
	if c.Asset != asset.Spot {
		return nil, strategy.ErrInvalidAssetType
	}

	depth, err := orderbook.GetDepth(c.Exchange.GetName(), c.Pair, c.Asset)
	if err != nil {
		return nil, err
	}

	var selling *account.ProtectedBalance
	var balance float64
	if !c.Simulate {
		creds, err := c.Exchange.GetCredentials(ctx)
		if err != nil {
			return nil, err
		}

		buying, err := account.GetBalance(c.Exchange.GetName(),
			creds.SubAccount, creds, c.Asset, c.Pair.Base)
		if err != nil {
			return nil, err
		}

		selling, err := account.GetBalance(c.Exchange.GetName(),
			creds.SubAccount, creds, c.Asset, c.Pair.Quote)
		if err != nil {
			return nil, err
		}

		deployment, acquiring := c.Pair.Quote, c.Pair.Base
		if !c.Buy {
			selling = buying
			deployment = c.Pair.Base
			acquiring = c.Pair.Quote
		}

		balance = selling.GetFree()
		if balance == 0 {
			return nil, fmt.Errorf("cannot sell %v %s to acquire %s this %w of %v %s",
				c.Amount,
				deployment,
				acquiring,
				strategy.ErrNoBalance,
				balance,
				deployment)
		}

		if !c.FullAmount {
			if c.Amount > balance {
				return nil, fmt.Errorf("cannot sell %v %s to acquire %s this %w of %v %s",
					c.Amount,
					deployment,
					acquiring,
					strategy.ErrExceedsFreeBalance,
					balance,
					deployment)
			}
			balance = c.Amount
		}
	} else {
		if c.FullAmount {
			return nil, strategy.ErrFullAmountSimulation
		}
		if c.Amount <= 0 {
			return nil, fmt.Errorf("%w %v for simulation",
				strategy.ErrInvalidAmount,
				c.Amount)
		}
		balance = c.Amount
	}

	// NOTE: For now this will not allow any amount to deplete the full
	// orderbook, just until a safe, effective and efficient system has been
	// tested and deployed for public use.
	// TODO: Bypass error strategy.ErrExceedsFreeBalance.
	allocation, err := c.GetDistrbutionAmount(balance, depth)
	if err != nil {
		return nil, err
	}

	schedule, err := strategy.NewScheduler(c.Start, c.End, c.CandleStickAligned, c.Interval)
	if err != nil {
		return nil, err
	}

	activities, err := strategy.NewActivities("DOLLAR COST AVERAGE (DCA)", c.Simulate)
	if err != nil {
		return nil, err
	}

	return &Strategy{
		Config:     c,
		orderbook:  depth,
		Selling:    selling,
		allocation: allocation,
		Scheduler:  schedule,
		Requirement: strategy.Requirement{
			Activities:       *activities,
			OperateBeyondEnd: c.AllowTradingPastEndTime,
		},
	}, nil
}

// checkAndSubmit verifies orderbook deployability then executes an order if
// all checks pass.
func (s *Strategy) checkAndSubmit(ctx context.Context) error {
	if s == nil {
		return strategy.ErrIsNil
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

	s.ReportAcceptedSignal(nil)

	resp, err := s.submitOrder(ctx, submit)
	if err != nil {
		return err
	}

	// Note: For first iteration of strategy this is just easy reconciliation.
	// TODO: Reconcile to adjusted amount.
	s.allocation.Deployed += s.allocation.Deployment
	s.allocation.Deployments++

	s.ReportOrder(strategy.OrderAction{Submit: submit, Response: resp, Orderbook: details})
	return nil
}

// deriveOrder checks amount and returns an order submission. TODO: Abstract
// futher.
func (s *Strategy) deriveOrder(amountInBase float64) (*order.Submit, error) {
	if amountInBase <= 0 {
		return nil, fmt.Errorf("amount in base: %w", strategy.ErrInvalidAmount)
	}
	side := order.Buy
	if !s.Buy {
		side = order.Sell
	}
	return &order.Submit{
		Exchange:  s.Config.Exchange.GetName(),
		Type:      order.Market,
		Pair:      s.Config.Pair,
		AssetType: s.Config.Asset,
		Side:      side,
		Amount:    amountInBase,
	}, nil
}

// submitOrder will submit and retry an order if fail. TODO: Abstract futher
func (s *Strategy) submitOrder(ctx context.Context, submit *order.Submit) (*order.SubmitResponse, error) {
	if submit == nil {
		return nil, strategy.ErrSubmitOrderIsNil
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
			resp, err = s.Config.Exchange.SubmitOrder(ctx, submit)
		} else {
			resp, err = submit.DeriveSubmitResponse(strategy.Simulation)
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
