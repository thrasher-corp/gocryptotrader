package fee

import (
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// Options defines fee loading options and is also used as a state snapshot, in
// GetAllFees method.
type Options struct {
	// Commission defines the maker and taker rates for the indv. asset item.
	Commission map[asset.Item]Commission
	// Transfer defines a map of currencies with differing withdrawal and
	// deposit fee definitions. These will commonly be fixed real values.
	Transfer map[asset.Item]map[currency.Code]Transfer
	// BankingTransfer defines a map of currencies with differing withdrawal and
	// deposit fee definitions for banking. These will commonly be fixed real
	// values.
	BankingTransfer map[BankTransaction]map[currency.Code]Transfer
}

// validate checks for invalid values on struct, should be used prior to lock
func (o Options) validate() error {
	for _, v := range o.Commission {
		err := v.validate()
		if err != nil {
			return fmt.Errorf("commission error: %w", err)
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
