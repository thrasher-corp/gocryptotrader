package exchange

var (
	allExchanges = []Details{
		{
			Name: "Alphapoint",
		},
		{
			Name: "Binance",
		},
		{
			Name: "Bitfinex",
		},
		{
			Name: "Bitflyer",
		},
		{
			Name: "Bithumb",
		},
		{
			Name: "Bitmex",
		},
		{
			Name: "Bitstamp",
		},
		{
			Name: "Bittrex",
		},
		{
			Name: "BTC Markets",
		},
		{
			Name: "BTSE",
		},
		{
			Name: "Coinbase Pro",
		},
		{
			Name: "Coinbene",
		},
		{
			Name: "Coinut",
		},
		{
			Name: "Exmo",
		},
		{
			Name: "GateIO",
		},
		{
			Name: "Gemini",
		},
		{
			Name: "HitBTC",
		},
		{
			Name: "Huobi",
		},
		{
			Name: "itBit",
		},
		{
			Name: "Kraken",
		},
		{
			Name: "lakeBTC",
		},
		{
			Name: "lBank",
		},
		{
			Name: "Local Bitcoins",
		},
		{
			Name: "OKCoin",
		},
		{
			Name: "OKEX",
		},
		{
			Name: "Poloniex",
		},
		{
			Name: "YoBit",
		},
		{
			Name: "ZB",
		},
	}
)

// Seed will import seeded data to the database
func Seed() error {
	return InsertMany(allExchanges)
}
