package fee

import (
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// Options defines fee loading options and is also used as a state snapshot, in
// GetAllFees method.
type Options struct {
	// GlobalCommissions defines the maker and taker rates for the indv. asset
	// item.
	GlobalCommissions map[asset.Item]Commission
	// PairCommissions defines the maker and taker rates for the individual
	// trading pair item.
	PairCommissions map[asset.Item]map[currency.Pair]Commission
	// ChainTransfer defines deposit and withdrawal fees between cryptocurrency
	// wallets and exchanges. These will commonly be fixed values.
	ChainTransfer []Transfer
	// BankTransfer defines a map of currencies with differing withdrawal and
	// deposit fee definitions for banking. These will commonly be fixed real
	// values.
	BankTransfer []Transfer
}

// validate checks for invalid values on struct, should be used prior to lock
func (o Options) validate() error {
	for a, v := range o.GlobalCommissions {
		err := v.validate()
		if err != nil {
			return fmt.Errorf("global commission for %s error: %w", a, err)
		}
	}

	for a, m1 := range o.PairCommissions {
		for pair, v := range m1 {
			err := v.validate()
			if err != nil {
				return fmt.Errorf("%s %s commission error: %w", pair, a, err)
			}
		}
	}

	for x := range o.ChainTransfer {
		err := o.ChainTransfer[x].validate()
		if err != nil {
			return fmt.Errorf("chain transfer error for: %w", err)
		}
	}

	for x := range o.BankTransfer {
		err := o.BankTransfer[x].validate()
		if err != nil {
			return fmt.Errorf("bank transfer error: %w", err)
		}
	}
	return nil
}
