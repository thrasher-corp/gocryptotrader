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
	errParamsAreNil                   = errors.New("params are nil")
	errInvalidVolume                  = errors.New("invalid volume")
	errInvalidMaxSlippageValue        = errors.New("invalid max slippage percentage value")
	errExchangeIsNil                  = errors.New("exchange is nil")
	errTWAPIsNil                      = errors.New("twap is nil")
	errNoBalanceFound                 = errors.New("no balance found")
	errVolumeToSellExceedsFreeBalance = errors.New("volume to sell exceeds free balance")
	errConfigurationIsNil             = errors.New("strategy configuration is nil")
	errInvalidPriceLimit              = errors.New("invalid price limit")
	errInvalidMaxSpreadPercentage     = errors.New("invalid spread percentage")
	errExceedsFreeBalance             = errors.New("amount exceeds current free balance")
	errCannotSetAmount                = errors.New("specific amount cannot be set, full amount bool set")
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

	if p.Buy {
		freeQuote := quoteAmount.GetFree()
		if freeQuote == 0 {
			fmt.Errorf("cannot sell quote %s amount %f to buy base %s %w of %f",
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
	} else {
		freeBase := baseAmount.GetFree()
		if freeBase == 0 {
			fmt.Errorf("cannot sell quote %s amount %f to buy base %s %w of %f",
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
	}

	monAmounts := map[currency.Code]*account.ProtectedBalance{
		p.Pair.Base:  baseAmount,
		p.Pair.Quote: quoteAmount,
	}

	return &Strategy{
		Config:    p,
		Reporter:  make(chan Report),
		orderbook: depth,
		holdings:  monAmounts,
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

	balance, err := s.fetchCurrentBalance(ctx)
	if err != nil {
		return err
	}

	fmt.Println("balance", balance)

	tn := time.Now()
	start := tn.Truncate(time.Duration(s.Interval))
	fmt.Println(kline.ThirtyMin, start)
	return nil
}

// fetchCurrentBalance checks current available balance to undertake full
// strategy.
func (t *Strategy) fetchCurrentBalance(ctx context.Context) (float64, error) {
	holdings, err := t.Exchange.UpdateAccountInfo(ctx, t.Asset)
	if err != nil {
		return 0, err
	}

	var selling currency.Code
	if t.Accumulation {
		selling = t.Pair.Quote
	} else {
		selling = t.Pair.Base
	}

	for x := range holdings.Accounts {
		if holdings.Accounts[x].AssetType != t.Asset /*&& holdings.Accounts[x].ID != t.creds.SubAccount*/ {
			continue
		}

		for y := range holdings.Accounts[x].Currencies {
			if !holdings.Accounts[x].Currencies[y].CurrencyName.Equal(selling) {
				continue
			}

			if t.Amount > holdings.Accounts[x].Currencies[y].Free {
				return 0, fmt.Errorf("%s %w %v",
					selling,
					errVolumeToSellExceedsFreeBalance,
					holdings.Accounts[x].Currencies[y].Free)
			}

			return holdings.Accounts[x].Currencies[y].Free, nil
		}
		break
	}
	return 0, fmt.Errorf("selling currency %s %s %w",
		selling, t.Asset, errNoBalanceFound)
}

func (t *Strategy) funky(ctx context.Context) {
	until := time.Until(t.Start)
	timer := time.NewTimer(until)
	for {
		select {
		case <-ctx.Done():
			t.Reporter <- Report{Error: ctx.Err(), Finished: true}
			return
		case <-timer.C:
			resp, err := t.Exchange.SubmitOrder(ctx, &order.Submit{
				Exchange:  t.Exchange.GetName(),
				Pair:      t.Pair,
				AssetType: t.Asset,
				Side:      order.Bid,
				Type:      order.Market,
				Amount:    10, // Base amount
			})
			if err != nil {
				fmt.Println("LAME")
			}
			t.Reporter <- Report{Order: *resp}
		}
	}
}

// Report defines a TWAP action
type Report struct {
	Order    order.SubmitResponse
	TWAP     float64
	Slippage float64
	Error    error
	Finished bool
	Balance  map[currency.Code]float64
}
