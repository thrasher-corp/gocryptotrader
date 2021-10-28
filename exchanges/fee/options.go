package fee

import (
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/bank"
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
	// Transfer defines a map of currencies with differing withdrawal and
	// deposit fee definitions. These will commonly be fixed real values.
	Transfer map[asset.Item]map[currency.Code]Transfer
	// BankingTransfer defines a map of currencies with differing withdrawal and
	// deposit fee definitions for banking. These will commonly be fixed real
	// values.
	BankingTransfer map[bank.Transfer]map[currency.Code]Transfer
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

	for _, m1 := range o.Transfer {
		for _, v := range m1 {
			err := v.validate()
			if err != nil {
				return fmt.Errorf("transfer error: %w", err)
			}
		}
	}

	for bt, m1 := range o.BankingTransfer {
		err := bt.Validate()
		if err != nil {
			return err
		}
		for _, v := range m1 {
			err := v.validate()
			if err != nil {
				return fmt.Errorf("banking transfer error: %w", err)
			}
		}
	}
	return nil
}
