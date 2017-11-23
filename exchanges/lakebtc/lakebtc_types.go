package lakebtc

// Ticker holds ticker information
type Ticker struct {
	Last   float64
	Bid    float64
	Ask    float64
	High   float64
	Low    float64
	Volume float64
}

// OrderbookStructure stores price and amount for order books
type OrderbookStructure struct {
	Price  float64
	Amount float64
}

// Orderbook contains arrays of orderbook information
type Orderbook struct {
	Bids []OrderbookStructure `json:"bids"`
	Asks []OrderbookStructure `json:"asks"`
}

// TickerResponse stores temp response
// Silly hack due to API returning null instead of strings
type TickerResponse struct {
	Last   interface{}
	Bid    interface{}
	Ask    interface{}
	High   interface{}
	Low    interface{}
	Volume interface{}
}

// TradeHistory holds trade history data
type TradeHistory struct {
	Date   int64   `json:"data"`
	Price  float64 `json:"price,string"`
	Amount float64 `json:"amount,string"`
	TID    int64   `json:"tid"`
}

// AccountInfo contains account information
type AccountInfo struct {
	Balance map[string]string `json:"balance"`
	Locked  map[string]string `json:"locked"`
	Profile struct {
		Email             string `json:"email"`
		UID               string `json:"uid"`
		BTCDepositAddress string `json:"btc_deposit_addres"`
	} `json:"profile"`
}

// Trade holds trade information
type Trade struct {
	ID     int64  `json:"id"`
	Result string `json:"result"`
}

// OpenOrders stores full information on your open orders
type OpenOrders struct {
	ID     int64   `json:"id"`
	Amount float64 `json:"amount,string"`
	Price  float64 `json:"price,string"`
	Symbol string  `json:"symbol"`
	Type   string  `json:"type"`
	At     int64   `json:"at"`
}

// Orders holds current order information
type Orders struct {
	ID             int64   `json:"id"`
	OriginalAmount float64 `json:"original_amount,string"`
	Amount         float64 `json:"amount,string"`
	Price          float64 `json:"price,string"`
	Symbol         string  `json:"symbol"`
	Type           string  `json:"type"`
	State          string  `json:"state"`
	At             int64   `json:"at"`
}

// AuthenticatedTradeHistory is a store of personalised auth trade history
type AuthenticatedTradeHistory struct {
	Type   string  `json:"type"`
	Symbol string  `json:"symbol"`
	Amount float64 `json:"amount,string"`
	Total  float64 `json:"total,string"`
	At     int64   `json:"at"`
}

// ExternalAccounts holds external account information
type ExternalAccounts struct {
	ID         int64       `json:"id,string"`
	Type       string      `json:"type"`
	Address    string      `json:"address"`
	Alias      interface{} `json:"alias"`
	Currencies string      `json:"currencies"`
	State      string      `json:"state"`
	UpdatedAt  int64       `json:"updated_at,string"`
}

// Withdraw holds withdrawal information
type Withdraw struct {
	ID                int64   `json:"id,string"`
	Amount            float64 `json:"amount,string"`
	Currency          string  `json:"currency"`
	Fee               float64 `json:"fee,string"`
	State             string  `json:"state"`
	Source            string  `json:"source"`
	ExternalAccountID int64   `json:"external_account_id,string"`
	At                int64   `json:"at"`
}
