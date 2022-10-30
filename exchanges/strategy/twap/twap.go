package twap

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
)

var (
	errTWAPIsNil                 = errors.New("twap is nil")
	errNoBalanceFound            = errors.New("no balance found")
	errConfigurationIsNil        = errors.New("strategy configuration is nil")
	errExceedsFreeBalance        = errors.New("amount exceeds current free balance")
	errSpreadPercentageExceeded  = errors.New("spread percentage has been exceeded")
	errImpactPercentageExceeded  = errors.New("impact percentage exceeded")
	errNominalPercentageExceeded = errors.New("nominal percentage exceeded")
	errPriceLimitExceeded        = errors.New("price limit exceeded")
	errInvalidAssetType          = errors.New("non spot trading pairs not currently supported")
)

// GetTWAP returns a TWAP struct to manage TWAP allocation or deallocation of
// position.
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

	deploymentAmount, err := c.GetDistrbutionAmount(fullDeployment, depth)
	if err != nil {
		return nil, err
	}

	return &Strategy{
		Config:           c,
		Reporter:         make(chan Report),
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

	if s.FullAmount {
		s.FullDeployment = s.Selling.GetAvailableWithoutBorrow()
	} else {
		s.FullDeployment = s.Amount
	}

	distrubution, err := s.GetDistrbutionAmount(s.FullDeployment, s.orderbook)
	if err != nil {
		return err
	}

	s.wg.Add(1)
	go s.Deploy(ctx, distrubution)
	return nil
}

func (s *Strategy) Deploy(ctx context.Context, amount float64) {
	defer s.wg.Done()
	var until time.Duration
	if s.Start.After(time.Now()) {
		until = time.Until(s.Start)
	}
	fmt.Printf("Starting twap operation in %s...\n", until)
	timer := time.NewTimer(until)
	for {
		select {
		case <-timer.C:
			timer.Reset(time.Duration(s.Interval))
			var balance *account.ProtectedBalance
			if s.Buy {
				balance = s.Buying
			} else {
				balance = s.Selling
			}

			preOrderBalance := balance.GetAvailableWithoutBorrow()
			if preOrderBalance < amount {
				amount = preOrderBalance
			}

			err := s.checkAndSubmit(ctx, amount)
			if err != nil {
				s.Reporter <- Report{Error: err, Finished: true}
				return
			}

			if s.Simulate {
				s.AmountDeployed += amount
				if s.AmountDeployed >= s.FullDeployment {
					s.Reporter <- Report{Error: err, Finished: true}
					fmt.Println("finished amount yay")
					return
				}
			} else {
				var afterOrderBalance = balance.GetAvailableWithoutBorrow()
				for x := 0; afterOrderBalance == preOrderBalance || x < 3; x++ {
					time.Sleep(time.Second)
					afterOrderBalance = balance.GetAvailableWithoutBorrow()
				}

				if afterOrderBalance == 0 {
					s.Reporter <- Report{Finished: true}
					fmt.Println("finished amount yay")
					return
				}
			}

			if !s.AllowTradingPastEndTime && time.Now().After(s.End) {
				s.Reporter <- Report{Error: err, Finished: true}
				fmt.Println("finished cute time")
				return
			}

		case <-ctx.Done():
			s.Reporter <- Report{Error: ctx.Err(), Finished: true}
			return
			// case <-s.shutdown:
			// 	s.Reporter <- Report{Finished: true}
			// 	return
		}
	}
}

type OrderExecutionInformation struct {
	Time     time.Time
	Submit   *order.Submit
	Response *order.SubmitResponse
	Error    error
}

func (s *Strategy) checkAndSubmit(ctx context.Context, amount float64) error {
	fmt.Println("AMOUNT TO DEPLOY THIS ROUND", amount)
	spread, err := s.orderbook.GetSpreadPercentage()
	if err != nil {
		return fmt.Errorf("fetching spread percentage %w", err)
	}
	if s.MaxSpreadPercentage != 0 && s.MaxSpreadPercentage < spread {
		return fmt.Errorf("book spread: %f & spread limit: %f %w",
			spread,
			s.MaxSpreadPercentage,
			errSpreadPercentageExceeded)
	}

	var details *orderbook.Movement
	if s.Buy {
		details, err = s.orderbook.LiftTheAsksFromBest(amount, false)
	} else {
		details, err = s.orderbook.HitTheBidsFromBest(amount, false)
	}
	if err != nil {
		return err
	}

	if s.MaxImpactSlippage != 0 && s.MaxImpactSlippage < details.ImpactPercentage {
		return fmt.Errorf("impact slippage: %f & slippage limit: %f %w",
			details.ImpactPercentage,
			s.MaxImpactSlippage,
			errImpactPercentageExceeded)
	}

	if s.MaxNominalSlippage != 0 && s.MaxNominalSlippage < details.NominalPercentage {
		return fmt.Errorf("nominal slippage: %f & slippage limit: %f %w",
			details.NominalPercentage,
			s.MaxNominalSlippage,
			errNominalPercentageExceeded)
	}

	if s.PriceLimit != 0 && (s.Buy && details.StartPrice > s.PriceLimit || !s.Buy && details.StartPrice < s.PriceLimit) {
		if s.Buy {
			return fmt.Errorf("ask book head price: %f price limit: %f %w",
				details.StartPrice,
				s.PriceLimit,
				errPriceLimitExceeded)
		}
		return fmt.Errorf("bid book head price: %f price limit: %f %w",
			details.StartPrice,
			s.PriceLimit,
			errPriceLimitExceeded)
	}

	submit := &order.Submit{
		Exchange:   s.Exchange.GetName(),
		Type:       order.Market,
		Pair:       s.Pair,
		AssetType:  s.Asset,
		ReduceOnly: true, // Have reduce only as default for this strategy for now
	}

	if s.Buy {
		submit.Side = order.Buy
		submit.Amount = details.Purchased // Easy way to convert to base.
	} else {
		submit.Side = order.Sell
		submit.Amount = amount // Already base.
	}

	minMax, err := s.Exchange.GetOrderExecutionLimits(s.Asset, s.Pair)
	if err != nil {
		return err
	}

	if minMax.MinAmount != 0 && minMax.MinAmount > submit.Amount {
		return fmt.Errorf("%w; %s", errUnderMinimumAmount, minimumSizeResponse)
	}

	if minMax.MaxAmount != 0 && minMax.MaxAmount < submit.Amount {
		return fmt.Errorf("%w; %s", errOverMaximumAmount, maximumSizeResponse)
	}

	conformedAmount := minMax.ConformToAmount(submit.Amount)

	fmt.Printf("conformed amount: %f iteration amount: %f changed by: %f\n",
		conformedAmount,
		submit.Amount,
		submit.Amount-conformedAmount,
	)

	submit.Amount = conformedAmount

	var resp *order.SubmitResponse
	if !s.Simulate {
		resp, err = s.Exchange.SubmitOrder(ctx, submit)
	} else {
		resp, err = submit.DeriveSubmitResponse("simulate")
	}

	info := OrderExecutionInformation{
		Time:     time.Now(),
		Submit:   submit,
		Response: resp,
		Error:    err,
	}

	s.TradeInformation = append(s.TradeInformation, info)

	s.Reporter <- Report{Information: info, Deployment: details}
	return nil
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
