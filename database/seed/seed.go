package seed

import (
	"fmt"
	"strings"

	exchangeDB "github.com/thrasher-corp/gocryptotrader/database/repository/exchange"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

const (
	errReturn = "failed to seed %s data: %s"
)

func seedExchanges() error {
	var allExchanges []exchangeDB.Details
	for x := range exchange.Exchanges {
		allExchanges = append(allExchanges, exchangeDB.Details{
			Name: strings.Title(exchange.Exchanges[x]),
		})
	}
	return exchangeDB.InsertMany(allExchanges)
}

func seedOhlcvFromSource(csv, sql bool) error {
	return nil
}

// Run executes all seeding methods for database
func Run(exchanges, ohlcv bool) error {
	if exchanges {
		err := seedExchanges()
		if err != nil {
			return fmt.Errorf(errReturn, "exchange", err)
		}
	}

	if ohlcv {
		err := seedOhlcvFromSource(false, false)
		if err != nil {
			return fmt.Errorf(errReturn, "ohlcv", err)
		}
	}
	return nil
}
