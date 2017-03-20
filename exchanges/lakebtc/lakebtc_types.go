package lakebtc

type LakeBTCTicker struct {
	Last   float64
	Bid    float64
	Ask    float64
	High   float64
	Low    float64
	Volume float64
}

type LakeBTCOrderbookStructure struct {
	Price  float64
	Amount float64
}

type LakeBTCOrderbook struct {
	Bids []LakeBTCOrderbookStructure `json:"bids"`
	Asks []LakeBTCOrderbookStructure `json:"asks"`
}

/* Silly hack due to API returning null instead of strings */
type LakeBTCTickerResponse struct {
	Last   interface{}
	Bid    interface{}
	Ask    interface{}
	High   interface{}
	Low    interface{}
	Volume interface{}
}

type LakeBTCTradeHistory struct {
	Date   int64   `json:"data"`
	Price  float64 `json:"price,string"`
	Amount float64 `json:"amount,string"`
	TID    int64   `json:"tid"`
}

type LakeBTCAccountInfo struct {
	Balance map[string]string `json:"balance"`
	Locked  map[string]string `json:"locked"`
	Profile struct {
		Email             string `json:"email"`
		UID               string `json:"uid"`
		BTCDepositAddress string `json:"btc_deposit_addres"`
	} `json:"profile"`
}

type LakeBTCTrade struct {
	ID     int64  `json:"id"`
	Result string `json:"result"`
}

type LakeBTCOpenOrders struct {
	ID     int64   `json:"id"`
	Amount float64 `json:"amount,string"`
	Price  float64 `json:"price,string"`
	Symbol string  `json:"symbol"`
	Type   string  `json:"type"`
	At     int64   `json:"at"`
}

type LakeBTCOrders struct {
	ID             int64   `json:"id"`
	OriginalAmount float64 `json:"original_amount,string"`
	Amount         float64 `json:"amount,string"`
	Price          float64 `json:"price,string"`
	Symbol         string  `json:"symbol"`
	Type           string  `json:"type"`
	State          string  `json:"state"`
	At             int64   `json:"at"`
}

type LakeBTCAuthenticaltedTradeHistory struct {
	Type   string  `json:"type"`
	Symbol string  `json:"symbol"`
	Amount float64 `json:"amount,string"`
	Total  float64 `json:"total,string"`
	At     int64   `json:"at"`
}
type LakeBTCExternalAccounts struct {
	ID         int64       `json:"id,string"`
	Type       string      `json:"type"`
	Address    string      `json:"address"`
	Alias      interface{} `json:"alias"`
	Currencies string      `json:"currencies"`
	State      string      `json:"state"`
	UpdatedAt  int64       `json:"updated_at,string"`
}

type LakeBTCWithdraw struct {
	ID                int64   `json:"id,string"`
	Amount            float64 `json:"amount,string"`
	Currency          string  `json:"currency"`
	Fee               float64 `json:"fee,string"`
	State             string  `json:"state"`
	Source            string  `json:"source"`
	ExternalAccountID int64   `json:"external_account_id,string"`
	At                int64   `json:"at"`
}
