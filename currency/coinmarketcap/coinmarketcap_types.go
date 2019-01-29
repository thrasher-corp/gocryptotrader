package coinmarketcap

import "time"

// Settings defines the current settings from configuration file
type Settings struct {
	Name        string `json:"name"`
	Enabled     bool   `json:"enabled"`
	Verbose     bool   `json:"verbose"`
	APIkey      string `json:"apiKey"`
	AccountPlan string `json:"accountPlan"`
}

// Status defines a response status JSON struct that is received with every
// HTTP request
type Status struct {
	Timestamp    string `json:"timestamp"`
	ErrorCode    int64  `json:"error_code"`
	ErrorMessage string `json:"error_message"`
	Elapsed      int64  `json:"elapsed"`
	CreditCount  int64  `json:"credit_count"`
}

// CryptoCurrencyInfo defines cryptocurrency information
type CryptoCurrencyInfo map[string]struct {
	ID       int         `json:"id"`
	Name     string      `json:"name"`
	Symbol   string      `json:"symbol"`
	Category string      `json:"category"`
	Slug     string      `json:"slug"`
	Logo     string      `json:"logo"`
	Tags     []string    `json:"tags"`
	Platform interface{} `json:"platform"`
	Urls     struct {
		Website      []string      `json:"website"`
		Explorer     []string      `json:"explorer"`
		SourceCode   []string      `json:"source_code"`
		MessageBoard []string      `json:"message_board"`
		Chat         []interface{} `json:"chat"`
		Announcement []interface{} `json:"announcement"`
		Reddit       []string      `json:"reddit"`
		Twitter      []string      `json:"twitter"`
	} `json:"urls"`
}

// CryptoCurrencyMap defines a cryptocurrency struct
type CryptoCurrencyMap struct {
	ID                  int         `json:"id"`
	Name                string      `json:"name"`
	Symbol              string      `json:"symbol"`
	Slug                string      `json:"slug"`
	IsActive            int         `json:"is_active"`
	FirstHistoricalData time.Time   `json:"first_historical_data"`
	LastHistoricalData  time.Time   `json:"last_historical_data"`
	Platform            interface{} `json:"platform"`
}

// CryptocurrencyHistoricalListings defines a historical listing data
type CryptocurrencyHistoricalListings struct {
	ID                int       `json:"id"`
	Name              string    `json:"name"`
	Symbol            string    `json:"symbol"`
	Slug              string    `json:"slug"`
	CmcRank           int       `json:"cmc_rank"`
	NumMarketPairs    int       `json:"num_market_pairs"`
	CirculatingSupply float64   `json:"circulating_supply"`
	TotalSupply       float64   `json:"total_supply"`
	MaxSupply         float64   `json:"max_supply"`
	LastUpdated       time.Time `json:"last_updated"`
	Quote             struct {
		USD struct {
			Price            float64 `json:"price"`
			Volume24H        int64   `json:"volume_24h"`
			PercentChange1H  float64 `json:"percent_change_1h"`
			PercentChange24H float64 `json:"percent_change_24h"`
			PercentChange7D  float64 `json:"percent_change_7d"`
			MarketCap        int64   `json:"market_cap"`
		} `json:"USD"`
		BTC struct {
			Price            int `json:"price"`
			Volume24H        int `json:"volume_24h"`
			PercentChange1H  int `json:"percent_change_1h"`
			PercentChange24H int `json:"percent_change_24h"`
			PercentChange7D  int `json:"percent_change_7d"`
			MarketCap        int `json:"market_cap"`
		} `json:"BTC"`
	} `json:"quote"`
}

// CryptocurrencyLatestListings defines latest cryptocurrency listing data
type CryptocurrencyLatestListings struct {
	ID                int         `json:"id"`
	Name              string      `json:"name"`
	Symbol            string      `json:"symbol"`
	Slug              string      `json:"slug"`
	CmcRank           int         `json:"cmc_rank"`
	NumMarketPairs    int         `json:"num_market_pairs"`
	CirculatingSupply float64     `json:"circulating_supply"`
	TotalSupply       float64     `json:"total_supply"`
	MaxSupply         float64     `json:"max_supply"`
	LastUpdated       time.Time   `json:"last_updated"`
	DateAdded         time.Time   `json:"date_added"`
	Tags              []string    `json:"tags"`
	Platform          interface{} `json:"platform"`
	Quote             struct {
		USD struct {
			Price            float64   `json:"price"`
			Volume24H        float64   `json:"volume_24h"`
			PercentChange1H  float64   `json:"percent_change_1h"`
			PercentChange24H float64   `json:"percent_change_24h"`
			PercentChange7D  float64   `json:"percent_change_7d"`
			MarketCap        float64   `json:"market_cap"`
			LastUpdated      time.Time `json:"last_updated"`
		} `json:"USD"`
		BTC struct {
			Price            float64   `json:"price"`
			Volume24H        float64   `json:"volume_24h"`
			PercentChange1H  float64   `json:"percent_change_1h"`
			PercentChange24H float64   `json:"percent_change_24h"`
			PercentChange7D  float64   `json:"percent_change_7d"`
			MarketCap        float64   `json:"market_cap"`
			LastUpdated      time.Time `json:"last_updated"`
		} `json:"BTC"`
	} `json:"quote"`
}

// CryptocurrencyLatestMarketPairs defines the latest cryptocurrency pairs
type CryptocurrencyLatestMarketPairs struct {
	ID             int    `json:"id"`
	Name           string `json:"name"`
	Symbol         string `json:"symbol"`
	NumMarketPairs int    `json:"num_market_pairs"`
	MarketPairs    []struct {
		Exchange struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
			Slug string `json:"slug"`
		} `json:"exchange"`
		MarketPair     string `json:"market_pair"`
		MarketPairBase struct {
			CurrencyID     int    `json:"currency_id"`
			CurrencySymbol string `json:"currency_symbol"`
			CurrencyType   string `json:"currency_type"`
		} `json:"market_pair_base"`
		MarketPairQuote struct {
			CurrencyID     int    `json:"currency_id"`
			CurrencySymbol string `json:"currency_symbol"`
			CurrencyType   string `json:"currency_type"`
		} `json:"market_pair_quote"`
		Quote struct {
			ExchangeReported struct {
				Price          float64   `json:"price"`
				Volume24HBase  float64   `json:"volume_24h_base"`
				Volume24HQuote float64   `json:"volume_24h_quote"`
				LastUpdated    time.Time `json:"last_updated"`
			} `json:"exchange_reported"`
			USD struct {
				Price       float64   `json:"price"`
				Volume24H   float64   `json:"volume_24h"`
				LastUpdated time.Time `json:"last_updated"`
			} `json:"USD"`
		} `json:"quote"`
	} `json:"market_pairs"`
}

// CryptocurrencyOHLCHistorical defines open high low close historical data
type CryptocurrencyOHLCHistorical struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Symbol string `json:"symbol"`
	Quotes []struct {
		TimeOpen  time.Time `json:"time_open"`
		TimeClose time.Time `json:"time_close"`
		Quote     struct {
			USD struct {
				Open      float64   `json:"open"`
				High      float64   `json:"high"`
				Low       float64   `json:"low"`
				Close     float64   `json:"close"`
				Volume    int64     `json:"volume"`
				Timestamp time.Time `json:"timestamp"`
			} `json:"USD"`
		} `json:"quote"`
	} `json:"quotes"`
}

// CryptocurrencyOHLCLatest defines open high low close latest data
type CryptocurrencyOHLCLatest map[string]struct {
	ID          int         `json:"id"`
	Name        string      `json:"name"`
	Symbol      string      `json:"symbol"`
	LastUpdated time.Time   `json:"last_updated"`
	TimeOpen    time.Time   `json:"time_open"`
	TimeClose   interface{} `json:"time_close"`
	Quote       struct {
		USD struct {
			Open        float64   `json:"open"`
			High        float64   `json:"high"`
			Low         float64   `json:"low"`
			Close       float64   `json:"close"`
			Volume      int64     `json:"volume"`
			LastUpdated time.Time `json:"last_updated"`
		} `json:"USD"`
	} `json:"quote"`
}

// CryptocurrencyLatestQuotes defines latest cryptocurrency quotation data
type CryptocurrencyLatestQuotes map[string]struct {
	ID                int         `json:"id"`
	Name              string      `json:"name"`
	Symbol            string      `json:"symbol"`
	Slug              string      `json:"slug"`
	CirculatingSupply int         `json:"circulating_supply"`
	TotalSupply       int         `json:"total_supply"`
	MaxSupply         int         `json:"max_supply"`
	DateAdded         time.Time   `json:"date_added"`
	NumMarketPairs    int         `json:"num_market_pairs"`
	CmcRank           int         `json:"cmc_rank"`
	LastUpdated       time.Time   `json:"last_updated"`
	Tags              []string    `json:"tags"`
	Platform          interface{} `json:"platform"`
	Quote             struct {
		USD struct {
			Price            float64   `json:"price"`
			Volume24H        float64   `json:"volume_24h"`
			PercentChange1H  float64   `json:"percent_change_1h"`
			PercentChange24H float64   `json:"percent_change_24h"`
			PercentChange7D  float64   `json:"percent_change_7d"`
			MarketCap        float64   `json:"market_cap"`
			LastUpdated      time.Time `json:"last_updated"`
		} `json:"USD"`
	} `json:"quote"`
}

// CryptocurrencyHistoricalQuotes defines historical cryptocurrency quotation
// data
type CryptocurrencyHistoricalQuotes struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Symbol string `json:"symbol"`
	Quotes []struct {
		Timestamp time.Time `json:"timestamp"`
		Quote     struct {
			USD struct {
				Price       float64   `json:"price"`
				Volume24H   int64     `json:"volume_24h"`
				MarketCap   float64   `json:"market_cap"`
				LastUpdated time.Time `json:"last_updated"`
			} `json:"USD"`
		} `json:"quote"`
	} `json:"quotes"`
}

// ExchangeInfo defines exchange information
type ExchangeInfo map[string]struct {
	Urls struct {
		Website []string      `json:"website"`
		Twitter []string      `json:"twitter"`
		Blog    []interface{} `json:"blog"`
		Chat    []string      `json:"chat"`
		Fee     []string      `json:"fee"`
	} `json:"urls"`
	Logo string `json:"logo"`
	ID   int    `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// ExchangeMap defines a data for an exchange
type ExchangeMap struct {
	ID                  int       `json:"id"`
	Name                string    `json:"name"`
	Slug                string    `json:"slug"`
	IsActive            int       `json:"is_active"`
	FirstHistoricalData time.Time `json:"first_historical_data"`
	LastHistoricalData  time.Time `json:"last_historical_data"`
}

// ExchangeHistoricalListings defines historical exchange listings
type ExchangeHistoricalListings struct {
	ID             int       `json:"id"`
	Name           string    `json:"name"`
	Slug           string    `json:"slug"`
	CmcRank        int       `json:"cmc_rank"`
	NumMarketPairs int       `json:"num_market_pairs"`
	Timestamp      time.Time `json:"timestamp"`
	Quote          struct {
		USD struct {
			Timestamp              time.Time `json:"timestamp"`
			Volume24H              int       `json:"volume_24h"`
			Volume7D               int       `json:"volume_7d"`
			Volume30D              int       `json:"volume_30d"`
			PercentChangeVolume24H float64   `json:"percent_change_volume_24h"`
			PercentChangeVolume7D  float64   `json:"percent_change_volume_7d"`
			PercentChangeVolume30D float64   `json:"percent_change_volume_30d"`
		} `json:"USD"`
	} `json:"quote"`
}

// ExchangeLatestListings defines latest exchange listings
type ExchangeLatestListings struct {
	ID             int       `json:"id"`
	Name           string    `json:"name"`
	Slug           string    `json:"slug"`
	NumMarketPairs int       `json:"num_market_pairs"`
	LastUpdated    time.Time `json:"last_updated"`
	Quote          struct {
		USD struct {
			Volume24H              float64 `json:"volume_24h"`
			Volume24HAdjusted      float64 `json:"volume_24h_adjusted"`
			Volume7D               int64   `json:"volume_7d"`
			Volume30D              int64   `json:"volume_30d"`
			PercentChangeVolume24H float64 `json:"percent_change_volume_24h"`
			PercentChangeVolume7D  float64 `json:"percent_change_volume_7d"`
			PercentChangeVolume30D float64 `json:"percent_change_volume_30d"`
		} `json:"USD"`
	} `json:"quote"`
}

// ExchangeLatestMarketPairs defines latest market pairs
type ExchangeLatestMarketPairs struct {
	ID             int    `json:"id"`
	Name           string `json:"name"`
	Slug           string `json:"slug"`
	NumMarketPairs int    `json:"num_market_pairs"`
	MarketPairs    []struct {
		MarketPair     string `json:"market_pair"`
		MarketPairBase struct {
			CurrencyID     int    `json:"currency_id"`
			CurrencySymbol string `json:"currency_symbol"`
			CurrencyType   string `json:"currency_type"`
		} `json:"market_pair_base"`
		MarketPairQuote struct {
			CurrencyID     int    `json:"currency_id"`
			CurrencySymbol string `json:"currency_symbol"`
			CurrencyType   string `json:"currency_type"`
		} `json:"market_pair_quote"`
		Quote struct {
			ExchangeReported struct {
				Price          float64   `json:"price"`
				Volume24HBase  float64   `json:"volume_24h_base"`
				Volume24HQuote float64   `json:"volume_24h_quote"`
				LastUpdated    time.Time `json:"last_updated"`
			} `json:"exchange_reported"`
			USD struct {
				Price       float64   `json:"price"`
				Volume24H   float64   `json:"volume_24h"`
				LastUpdated time.Time `json:"last_updated"`
			} `json:"USD"`
		} `json:"quote"`
	} `json:"market_pairs"`
}

// ExchangeLatestQuotes defines latest exchange quotations
type ExchangeLatestQuotes struct {
	Binance struct {
		ID             int       `json:"id"`
		Name           string    `json:"name"`
		Slug           string    `json:"slug"`
		NumMarketPairs int       `json:"num_market_pairs"`
		LastUpdated    time.Time `json:"last_updated"`
		Quote          struct {
			USD struct {
				Volume24H              float64 `json:"volume_24h"`
				Volume24HAdjusted      float64 `json:"volume_24h_adjusted"`
				Volume7D               int64   `json:"volume_7d"`
				Volume30D              int64   `json:"volume_30d"`
				PercentChangeVolume24H float64 `json:"percent_change_volume_24h"`
				PercentChangeVolume7D  float64 `json:"percent_change_volume_7d"`
				PercentChangeVolume30D float64 `json:"percent_change_volume_30d"`
			} `json:"USD"`
		} `json:"quote"`
	} `json:"binance"`
}

// ExchangeHistoricalQuotes defines historical exchange quotations
type ExchangeHistoricalQuotes struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Slug   string `json:"slug"`
	Quotes []struct {
		Timestamp time.Time `json:"timestamp"`
		Quote     struct {
			USD struct {
				Volume24H int       `json:"volume_24h"`
				Timestamp time.Time `json:"timestamp"`
			} `json:"USD"`
		} `json:"quote"`
		NumMarketPairs int `json:"num_market_pairs"`
	} `json:"quotes"`
}

// GlobalMeticLatestQuotes defines latest global metric quotations
type GlobalMeticLatestQuotes struct {
	BtcDominance           float64   `json:"btc_dominance"`
	EthDominance           float64   `json:"eth_dominance"`
	ActiveCryptocurrencies int       `json:"active_cryptocurrencies"`
	ActiveMarketPairs      int       `json:"active_market_pairs"`
	ActiveExchanges        int       `json:"active_exchanges"`
	LastUpdated            time.Time `json:"last_updated"`
	Quote                  struct {
		USD struct {
			TotalMarketCap float64   `json:"total_market_cap"`
			TotalVolume24H float64   `json:"total_volume_24h"`
			LastUpdated    time.Time `json:"last_updated"`
		} `json:"USD"`
	} `json:"quote"`
}

// GlobalMeticHistoricalQuotes defines historical global metric quotations
type GlobalMeticHistoricalQuotes struct {
	Quotes []struct {
		Timestamp    time.Time `json:"timestamp"`
		BtcDominance float64   `json:"btc_dominance"`
		Quote        struct {
			USD struct {
				TotalMarketCap float64   `json:"total_market_cap"`
				TotalVolume24H float64   `json:"total_volume_24h"`
				Timestamp      time.Time `json:"timestamp"`
			} `json:"USD"`
		} `json:"quote"`
	} `json:"quotes"`
}

// PriceConversion defines price conversion data
type PriceConversion struct {
	Symbol      string    `json:"symbol"`
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Amount      int       `json:"amount"`
	LastUpdated time.Time `json:"last_updated"`
	Quote       struct {
		GBP struct {
			Price       float64   `json:"price"`
			LastUpdated time.Time `json:"last_updated"`
		} `json:"GBP"`
		LTC struct {
			Price       float64   `json:"price"`
			LastUpdated time.Time `json:"last_updated"`
		} `json:"LTC"`
		USD struct {
			Price       int       `json:"price"`
			LastUpdated time.Time `json:"last_updated"`
		} `json:"USD"`
	} `json:"quote"`
}
