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
)

const Simulation = "SIMULATION"

var (
	errTWAPIsNil           = errors.New("twap is nil")
	errNoBalanceFound      = errors.New("no balance found")
	errExceedsFreeBalance  = errors.New("amount exceeds current free balance")
	errInvalidAssetType    = errors.New("non spot trading pairs not currently supported")
	errStrategyIsNil       = errors.New("strategy is nil")
	errActivityReportIsNil = errors.New("activity report is nil")
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

	avail := selling.GetAvailableWithoutBorrow()
	if avail == 0 {
		return nil, fmt.Errorf("cannot sell %s amount %f to buy base %s %w of %f",
			deployment,
			c.Amount,
			c.Pair.Base,
			errNoBalanceFound,
			avail)
	}
	if !c.FullAmount && c.Amount > avail {
		return nil, fmt.Errorf("cannot sell %s amount %f to buy base %s %w of %f",
			deployment,
			c.Amount,
			c.Pair.Base,
			errExceedsFreeBalance,
			avail)
	}

	var fullDeployment float64
	if c.FullAmount {
		fullDeployment = selling.GetAvailableWithoutBorrow()
	} else {
		fullDeployment = c.Amount
	}

	// NOTE: For now this will not allow any amount to deplete the full
	// orderbook, just until a safe, effective and efficient system has been
	// tested and deployed.
	deploymentAmount, err := c.GetDistrbutionAmount(fullDeployment, depth)
	if err != nil {
		return nil, err
	}

	return &Strategy{
		Config:           c,
		Reporter:         make(chan *Report),
		orderbook:        depth,
		Buying:           buying,
		Selling:          selling,
		FullDeployment:   fullDeployment,
		DeploymentAmount: deploymentAmount,
	}, nil
}

// Run inititates a TWAP allocation using the specified paramaters.
func (s *Strategy) Run(ctx context.Context) error {
	if s == nil {
		return errTWAPIsNil
	}
	if s.Config == nil {
		return errConfigurationIsNil
	}

	var start time.Duration
	if s.CandleStickAligned {
		// If aligned this will need to be truncated
		var err error
		start, err = s.GetNextSchedule(s.Start)
		if err != nil {
			return err
		}
	}

	s.wg.Add(1)
	go s.Deploy(ctx, start)
	return nil
}

// Deploy oversees the deployment of the current strategy adhering to policies,
// limits, signals and timings.
func (s *Strategy) Deploy(ctx context.Context, start time.Duration) {
	defer s.wg.Done()
	fmt.Printf("Starting twap operation in %s...\n", start)
	// NOTE: Zero value start duration will execute immediately then deploy at
	// intervals.
	timer := time.NewTimer(start)
	for {
		select {
		case <-timer.C:
			err := s.SetTimer(timer)
			if err != nil {
				_ = s.SendReport(&Report{Error: err, Finished: true})
				return
			}

			preOrderBalance := s.Selling.GetAvailableWithoutBorrow()
			if preOrderBalance < s.DeploymentAmount {
				s.DeploymentAmount = preOrderBalance
			}

			err = s.checkAndSubmit(ctx, s.DeploymentAmount)
			if err != nil {
				_ = s.SendReport(&Report{Error: err, Finished: true})
				return
			}

			if s.Simulate {
				s.AmountDeployed += s.DeploymentAmount
				if s.AmountDeployed >= s.FullDeployment {
					_ = s.SendReport(&Report{Error: err, Finished: true})
					fmt.Println("finished amount yay")
					return
				}
			} else {
				var afterOrderBalance = s.Selling.GetAvailableWithoutBorrow()
				for x := 0; afterOrderBalance == preOrderBalance || x < 3; x++ {
					time.Sleep(time.Second)
					afterOrderBalance = s.Selling.GetAvailableWithoutBorrow()
				}

				if afterOrderBalance == 0 {
					_ = s.SendReport(&Report{Error: err, Finished: true})
					fmt.Println("finished amount yay")
					return
				}
			}

			if !s.AllowTradingPastEndTime && time.Now().After(s.End) {
				_ = s.SendReport(&Report{Error: err, Finished: true})
				fmt.Println("finished cute time")
				return
			}

		case <-ctx.Done():
			_ = s.SendReport(&Report{Error: ctx.Err(), Finished: true})
			return
		case <-s.shutdown:
			_ = s.SendReport(&Report{Finished: true})
			return
		case <-s.pause:
			// TODO: Pause
			_ = s.SendReport(&Report{Finished: true})
			return
		}
	}
}

type OrderExecutionInformation struct {
	Time     time.Time
	Submit   *order.Submit
	Response *order.SubmitResponse
	Error    error
}

// checkAndSubmit verifies orderbook deployability then executes an order if
// all checks pass.
func (s *Strategy) checkAndSubmit(ctx context.Context, deployment float64) error {
	if s == nil {
		return errStrategyIsNil
	}
	if deployment <= 0 {
		return errInvalidAllocatedAmount
	}

	deploymentInBase, details, err := s.VerifyBookDeployment(s.orderbook, deployment)
	if err != nil {
		return err
	}

	conformed, err := s.VerifyExecutionLimitsReturnConformed(deploymentInBase)
	if err != nil {
		return err
	}

	submit, err := s.DeriveOrder(conformed)
	if err != nil {
		return err
	}

	resp, err := s.SubmitOrder(ctx, submit)
	info := OrderExecutionInformation{
		Time:     time.Now(),
		Submit:   submit,
		Response: resp,
		Error:    err,
	}
	s.TradeInformation = append(s.TradeInformation, info)
	return s.SendReport(&Report{Information: info, Deployment: details})
}

type DeploymentSchedule struct {
	Time     time.Time
	Amount   float64
	Executed order.Detail
}

// Report defines a TWAP action
type Report struct {
	Information OrderExecutionInformation
	Deployment  *orderbook.Movement
	Error       error
	Finished    bool
}

// DeriveOrder checks amount and returns an order submission
func (s *Strategy) DeriveOrder(amountInBase float64) (*order.Submit, error) {
	if amountInBase <= 0 {
		return nil, errInvalidAllocatedAmount
	}
	side := order.Buy
	if !s.Buy {
		side = order.Sell
	}
	return &order.Submit{
		Exchange:   s.Exchange.GetName(),
		Type:       order.Market,
		Pair:       s.Pair,
		AssetType:  s.Asset,
		ReduceOnly: true, // Have reduce only as default for this strategy
		Side:       side,
		Amount:     amountInBase,
	}, nil
}

// SubmitOrder will submit and retry an order if fail.
func (s *Strategy) SubmitOrder(ctx context.Context, submit *order.Submit) (*order.SubmitResponse, error) {
	var errors common.Errors
	var err error
	var resp *order.SubmitResponse
	for attempt := 0; attempt < int(s.RetryAttempts); attempt++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		if !s.Simulate {
			resp, err = s.Exchange.SubmitOrder(ctx, submit)
		} else {
			resp, err = submit.DeriveSubmitResponse(Simulation)
		}
		if err == nil {
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

// SendReport sends a strategy activity report to a potential receiver. Will
// do nothing if there is no receiver.
func (s *Strategy) SendReport(rp *Report) error {
	if rp == nil {
		return errActivityReportIsNil
	}
	select {
	case s.Reporter <- rp:
	default:
	}
	return nil
}
