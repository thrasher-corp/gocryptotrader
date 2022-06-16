package twap

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/strategy"
)

var (
	errParamsAreNil                   = errors.New("params are nil")
	errInvalidVolume                  = errors.New("invalid volume")
	errInvalidMaxSlippageValue        = errors.New("invalid max slippage percentage value, need to be between 0 and 100")
	errExchangeIsNil                  = errors.New("exchange is nil")
	errTWAPIsNil                      = errors.New("twap is nil")
	errNoBalanceFound                 = errors.New("no balance found")
	errVolumeToSellExceedsFreeBalance = errors.New("volume to sell exceeds free balance")
	errConfigurationIsNil             = errors.New("strategy configuration is nil")
)

// Strategy defines a TWAP strategy that handles the accumulation/de-accumulation
// of assets via a time weighted average price.
type Strategy struct {
	strategy.Base
	*Config
	holdings  map[currency.Code]*account.ProtectedBalance
	Reporter  chan Report
	Candles   kline.Item
	orderbook *orderbook.Depth
}

// GetTWAP returns a TWAP struct to manage TWAP allocation or deallocation of
// position.
func New(ctx context.Context, p *Config) (*Strategy, error) {
	if err := p.Check(ctx); err != nil {
		return nil, err
	}
	depth, err := orderbook.GetDepth(p.Exchange.GetName(), p.Pair, p.Asset)
	if err != nil {
		return nil, err
	}
	return &Strategy{Config: p, Reporter: make(chan Report), orderbook: depth}, nil
}

// Config defines the base elements required to undertake the TWAP strategy.
type Config struct {
	Exchange exchange.IBotExchange
	Pair     currency.Pair
	Asset    asset.Item

	Start time.Time
	End   time.Time

	// Interval between market orders
	Interval kline.Interval

	// Volume if accumulating refers to quotation if deaccum it will refer to
	// the base amount
	Volume float64

	// MaxSlippage needed for protection in low liqudity environments.
	// WARNING: 0 value == 100% slippage
	MaxSlippage float64
	// Accumulation if you are buying or selling value
	Accumulation bool
	// AllowTradingPastEndTime if volume has not been met exceed end time.
	AllowTradingPastEndTime bool
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

	err = cfg.Exchange.GetBase().ValidateKline(cfg.Pair, cfg.Asset, cfg.Interval)
	if err != nil {
		return err
	}

	if cfg.Volume <= 0 {
		return errInvalidVolume
	}

	if cfg.MaxSlippage < 0 || cfg.MaxSlippage > 100 {
		return fmt.Errorf("'%v' %w", cfg.MaxSlippage, errInvalidMaxSlippageValue)
	}

	_, err = cfg.Exchange.GetCredentials(ctx)
	return err
}

// Run inititates a TWAP allocation using the specified paramaters.
func (t *Strategy) Run(ctx context.Context) error {
	if t == nil {
		return errTWAPIsNil
	}

	if t.Config == nil {
		return errConfigurationIsNil
	}

	// candles, err := t.Exchange.GetHistoricCandlesExtended(ctx, t.Pair, t.Asset, time.Now(), time.Now(), kline.EightHour)
	// if err != nil {
	// 	return err
	// }

	// var count, cum float64
	// for x := range candles.Candles {
	// 	cum += candles.Candles[x].GetAveragePrice()
	// 	count++
	// }

	// twapPrice := cum / count
	// fmt.Println(twapPrice)

	// GET HISTORICAL CANDLES ON INTERVAL
	// DERIVE TWAP ON CANDLES

	balance, err := t.fetchCurrentBalance(ctx)
	if err != nil {
		return err
	}

	fmt.Println("balance", balance)

	tn := time.Now()
	start := tn.Truncate(time.Duration(t.Interval))
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

			if t.Volume > holdings.Accounts[x].Currencies[y].Free {
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

// func (t *Strategy) funky(ctx context.Context) {
// 	until := time.Until(t.Start)
// 	timer := time.NewTimer(until)
// 	for {
// 		select {
// 		case <-ctx.Done():
// 			t.Reporter <- Report{Error: ctx.Err(), Finished: true}
// 			return
// 		case <-timer.C:
// 			resp, err := t.Exchange.SubmitOrder(ctx, &order.Submit{
// 				Exchange:  t.Exchange.GetName(),
// 				Pair:      t.Pair,
// 				AssetType: t.Asset,
// 				Side:      order.Bid,
// 				Type:      order.Market,
// 				Amount:    10, // Base amount
// 			})
// 			if err != nil {
// 				fmt.Println("LAME")
// 			}
// 			t.Reporter <- Report{Order: *resp}
// 		}
// 	}
// }

// Report defines a TWAP action
type Report struct {
	Order    order.SubmitResponse
	TWAP     float64
	Slippage float64
	Error    error
	Finished bool
	Balance  map[currency.Code]float64
}
