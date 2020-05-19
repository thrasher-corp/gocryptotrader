package seed

import (
	"github.com/thrasher-corp/gocryptotrader/database/repository/currency"
	gctcurrency "github.com/thrasher-corp/gocryptotrader/currency"
)

func Currency() error {
	allCurrencies := []currency.Details{
		{
			Name:      gctcurrency.BTC.String(),
			Fiat:      true,
		},
		{
			Name:      gctcurrency.LTC.String(),
			Fiat:      true,
		},
		{
			Name:      gctcurrency.ETH.String(),
			Fiat:      true,
		},
		{
			Name:      gctcurrency.XRP.String(),
			Fiat:      true,
		},
	}
	return currency.InsertMany(allCurrencies)
}