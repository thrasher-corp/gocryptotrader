package seed

import "github.com/thrasher-corp/gocryptotrader/database/repository/exchange"

func Run() error {
	return exchange.Seed()
}