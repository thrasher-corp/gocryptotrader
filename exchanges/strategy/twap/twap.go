package twap

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
)

var (
	errParamsAreNil            = errors.New("params are nil")
	errInvalidVolume           = errors.New("invalid volume")
	errInvalidMaxSlippageValue = errors.New("invalid max slippage percentage value")
	errExchangeIsNil           = errors.New("exchange is nil")
	errTWAPIsNil               = errors.New("twap is nil")
	errNoBalanceFound          = errors.New("no balance found")
	// errVolumeToSellExceedsFreeBalance = errors.New("volume to sell exceeds free balance")
	errConfigurationIsNil         = errors.New("strategy configuration is nil")
	errInvalidPriceLimit          = errors.New("invalid price limit")
	errInvalidMaxSpreadPercentage = errors.New("invalid spread percentage")
	errExceedsFreeBalance         = errors.New("amount exceeds current free balance")
	errCannotSetAmount            = errors.New("specific amount cannot be set, full amount bool set")

	errSpreadPercentageExceeded = errors.New("spread percentage has been exceeded")
)

// GetTWAP returns a TWAP struct to manage TWAP allocation or deallocation of
// position.
func New(ctx context.Context, p *Config) (*Strategy, error) {
	err := p.Check(ctx)
	if err != nil {
		return nil, err
	}

	depth, err := orderbook.GetDepth(p.Exchange.GetName(), p.Pair, p.Asset)
	if err != nil {
		return nil, err
	}

	creds, err := p.Exchange.GetCredentials(ctx)
	if err != nil {
		return nil, err
	}

	baseAmount, err := account.GetBalance(p.Exchange.GetName(),
		creds.SubAccount, creds, p.Asset, p.Pair.Base)
	if err != nil {
		return nil, err
	}

	quoteAmount, err := account.GetBalance(p.Exchange.GetName(),
		creds.SubAccount, creds, p.Asset, p.Pair.Quote)
	if err != nil {
		return nil, err
	}

	var buying, selling Holding
	if p.Buy {
		freeQuote := quoteAmount.GetFree()
		if freeQuote == 0 {
			return nil, fmt.Errorf("cannot sell quote %s amount %f to buy base %s %w of %f",
				p.Pair.Quote,
				p.Amount,
				p.Pair.Base,
				errNoBalanceFound,
				freeQuote)
		}
		if p.FullAmount && p.Amount > freeQuote {
			return nil, fmt.Errorf("cannot sell quote %s amount %f to buy base %s %w of %f",
				p.Pair.Quote,
				p.Amount,
				p.Pair.Base,
				errExceedsFreeBalance,
				freeQuote)
		}
		buying = Holding{Currency: p.Pair.Base, Amount: baseAmount}
		selling = Holding{Currency: p.Pair.Quote, Amount: quoteAmount}
	} else {
		freeBase := baseAmount.GetFree()
		if freeBase == 0 {
			return nil, fmt.Errorf("cannot sell quote %s amount %f to buy base %s %w of %f",
				p.Pair.Quote,
				p.Amount,
				p.Pair.Base,
				errNoBalanceFound,
				freeBase)
		}
		if p.FullAmount && p.Amount > freeBase {
			return nil, fmt.Errorf("cannot sell base %s amount %f to buy quote %s %w of %f",
				p.Pair.Base,
				p.Amount,
				p.Pair.Quote,
				errExceedsFreeBalance,
				freeBase)
		}
		selling = Holding{Currency: p.Pair.Base, Amount: baseAmount}
		buying = Holding{Currency: p.Pair.Quote, Amount: quoteAmount}
	}

	return &Strategy{
		Config:    p,
		Reporter:  make(chan Report),
		orderbook: depth,
		Buying:    buying,
		Selling:   selling,
	}, nil
}

// Check validates all parameter fields before undertaking specfic strategy
func (cfg *Config) Check(ctx context.Context) error {
	if cfg == nil {
		return errParamsAreNil
	}

	if cfg.Exchange == nil {
		return errExchangeIsNil
	}

	if cfg.Pair.IsEmpty() {
		return currency.ErrPairIsEmpty
	}

	if !cfg.Asset.IsValid() {
		return fmt.Errorf("'%v' %w", cfg.Asset, asset.ErrNotSupported)
	}

	err := common.StartEndTimeCheck(cfg.Start, cfg.End)
	if err != nil {
		return err
	}

	if cfg.Interval == 0 {
		return kline.ErrUnsetInterval
	}

	if cfg.FullAmount && cfg.Amount != 0 {
		return errCannotSetAmount
	}
	if !cfg.FullAmount && cfg.Amount <= 0 {
		return errInvalidVolume
	}

	if cfg.MaxImpactSlippage < 0 || !cfg.Buy && cfg.MaxImpactSlippage > 100 {
		return fmt.Errorf("impact '%v' %w", cfg.MaxImpactSlippage, errInvalidMaxSlippageValue)
	}

	if cfg.MaxNominalSlippage < 0 || !cfg.Buy && cfg.MaxNominalSlippage > 100 {
		return fmt.Errorf("nominal '%v' %w", cfg.MaxNominalSlippage, errInvalidMaxSlippageValue)
	}

	if cfg.PriceLimit < 0 {
		return fmt.Errorf("price '%v' %w", cfg.PriceLimit, errInvalidPriceLimit)
	}

	if cfg.MaxSpreadpercentage < 0 {
		return fmt.Errorf("max spread '%v' %w", cfg.MaxSpreadpercentage, errInvalidMaxSpreadPercentage)
	}

	return nil
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
		s.AmountRequired = s.Selling.Amount.GetFree()
	} else {
		s.AmountRequired = s.Amount
	}

	distrubution, err := s.GetDistrbutionAmount(s.AmountRequired)
	if err != nil {
		return err
	}

	s.wg.Add(1)
	go s.Deploy(ctx, distrubution)
	return nil
}

func (s *Strategy) Deploy(ctx context.Context, amount float64) {
	defer s.wg.Done()
	timer := time.NewTimer(0)
	for {
		select {
		case <-timer.C:
			err := s.checkAndSubmit(ctx, amount)
			if err != nil {
				s.Reporter <- Report{Error: err, Finished: true}
				return
			}

			s.AmountDeployed += amount
			if s.AmountDeployed >= s.AmountRequired {
				s.Reporter <- Report{Error: err, Finished: true}
				fmt.Println("finished amount yay")
				return
			}

			if !s.AllowTradingPastEndTime && time.Now().After(s.End) {
				s.Reporter <- Report{Error: err, Finished: true}
				fmt.Println("finished cute time")
				return
			}

			timer.Reset(time.Duration(s.Interval))
		case <-ctx.Done():
			s.Reporter <- Report{Error: ctx.Err(), Finished: true}
			return
		case <-s.shutdown:
			s.Reporter <- Report{Finished: true}
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

var errImpactPercentageExceeded = errors.New("impact percentage exceeded")
var errNominalPercentageExceeded = errors.New("nominal percentage exceeded")
var errPriceLimitExceeded = errors.New("price limit exceeded")

func (s *Strategy) checkAndSubmit(ctx context.Context, amount float64) error {
	fmt.Println("AMOUNT TO DEPLOY THIS ROUND", amount)
	spread, err := s.orderbook.GetSpreadPercentage()
	if err != nil {
		return fmt.Errorf("fetching spread percentage %w", err)
	}
	if s.MaxSpreadpercentage != 0 && s.MaxSpreadpercentage < spread {
		return fmt.Errorf("book spread: %f & spread limit: %f %w",
			spread, s.MaxSpreadpercentage, errSpreadPercentageExceeded)
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
		ReduceOnly: s.ReduceOnly,
	}

	if s.Buy {
		submit.Side = order.Buy
		submit.Amount = details.Purchased // Easy way to convert to base.
	} else {
		submit.Side = order.Sell
		submit.Amount = amount // Already base.
	}

	var resp *order.SubmitResponse
	if !s.Simulate {
		// resp, err = s.Exchange.SubmitOrder(ctx, submit)
		panic("smelly")
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

// GetDeploymentAmount will truncate and equally distribute amounts across time.
func (c *Config) GetDistrbutionAmount(amount float64) (float64, error) {
	window := c.End.Sub(c.Start)
	if int64(window) <= int64(c.Interval) {
		return 0, errors.New("start end time window is equal to or less than interval")
	}
	segment := int64(window) / int64(c.Interval)
	return amount / float64(segment), nil
}

// Report defines a TWAP action
type Report struct {
	Information OrderExecutionInformation
	Deployment  *orderbook.Movement
	Error       error
	Finished    bool
}
