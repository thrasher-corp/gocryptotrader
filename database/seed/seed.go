package seed

import (
	exchangeDB "github.com/thrasher-corp/gocryptotrader/database/repository/exchange"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)


func Exchange() error {
	var allExchanges []exchangeDB.Details
	for x := range exchange.Exchanges {
		allExchanges = append(allExchanges, exchangeDB.Details{
			Name: exchange.Exchanges[x],
		})
	}
	return exchangeDB.InsertMany(allExchanges)
}

// Run executes all seeding methods for database
func Run() error {
	return Exchange()
}
