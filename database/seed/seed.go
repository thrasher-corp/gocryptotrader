package seed

import "github.com/thrasher-corp/gocryptotrader/database/repository/exchange"

func setupEngine() {

}

// Run executes all seeding methods for database
func Run() error {
	return exchange.Seed()
}
