package exchange

// Seed will import seeded data to the database
func Seed() error {
	var allExchanges []Details
	// for x := range exchange.Exchanges {
	// 	allExchanges = append(allExchanges, Details{
	// 		Name: exchange.Exchanges[x],
	// 	})
	// }
	return InsertMany(allExchanges)
}
