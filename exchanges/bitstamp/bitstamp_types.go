package bitstamp

type BitstampTicker struct {
	Last      float64 `json:"last,string"`
	High      float64 `json:"high,string"`
	Low       float64 `json:"low,string"`
	Vwap      float64 `json:"vwap,string"`
	Volume    float64 `json:"volume,string"`
	Bid       float64 `json:"bid,string"`
	Ask       float64 `json:"ask,string"`
	Timestamp int64   `json:"timestamp,string"`
	Open      float64 `json:"open,string"`
}

type BitstampBalances struct {
	BTCReserved  float64 `json:"btc_reserved,string"`
	BTCEURFee    float64 `json:"btceur_fee,string"`
	BTCAvailable float64 `json:"btc_available,string"`
	XRPAvailable float64 `json:"xrp_available,string"`
	EURAvailable float64 `json:"eur_available,string"`
	USDReserved  float64 `json:"usd_reserved,string"`
	EURReserved  float64 `json:"eur_reserved,string"`
	XRPEURFee    float64 `json:"xrpeur_fee,string"`
	XRPReserved  float64 `json:"xrp_reserved,string"`
	XRPBalance   float64 `json:"xrp_balance,string"`
	XRPUSDFee    float64 `json:"xrpusd_fee,string"`
	EURBalance   float64 `json:"eur_balance,string"`
	BTCBalance   float64 `json:"btc_balance,string"`
	BTCUSDFee    float64 `json:"btcusd_fee,string"`
	USDBalance   float64 `json:"usd_balance,string"`
	USDAvailable float64 `json:"usd_available,string"`
	EURUSDFee    float64 `json:"eurusd_fee,string"`
}

type BitstampOrderbookBase struct {
	Price  float64
	Amount float64
}

type BitstampOrderbook struct {
	Timestamp int64 `json:"timestamp,string"`
	Bids      []BitstampOrderbookBase
	Asks      []BitstampOrderbookBase
}

type BitstampTransactions struct {
	Date    int64   `json:"date,string"`
	TradeID int64   `json:"tid,string"`
	Price   float64 `json:"price,string"`
	Type    int     `json:"type,string"`
	Amount  float64 `json:"amount,string"`
}

type BitstampEURUSDConversionRate struct {
	Buy  float64 `json:"buy,string"`
	Sell float64 `json:"sell,string"`
}

type BitstampUserTransactions struct {
	Date    string  `json:"datetime"`
	TransID int64   `json:"id"`
	Type    int     `json:"type,string"`
	USD     float64 `json:"usd"`
	EUR     float64 `json:"eur"`
	BTC     float64 `json:"btc"`
	XRP     float64 `json:"xrp"`
	BTCUSD  float64 `json:"btc_usd"`
	Fee     float64 `json:"fee,string"`
	OrderID int64   `json:"order_id"`
}

type BitstampOrder struct {
	ID     int64   `json:"id"`
	Date   string  `json:"datetime"`
	Type   int     `json:"type"`
	Price  float64 `json:"price"`
	Amount float64 `json:"amount"`
}

type BitstampOrderStatus struct {
	Status       string
	Transactions []struct {
		TradeID int64   `json:"tid"`
		USD     float64 `json:"usd,string"`
		Price   float64 `json:"price,string"`
		Fee     float64 `json:"fee,string"`
		BTC     float64 `json:"btc,string"`
	}
}

type BitstampWithdrawalRequests struct {
	OrderID int64   `json:"id"`
	Date    string  `json:"datetime"`
	Type    int     `json:"type"`
	Amount  float64 `json:"amount,string"`
	Status  int     `json:"status"`
	Data    interface{}
}

type BitstampUnconfirmedBTCTransactions struct {
	Amount        float64 `json:"amount,string"`
	Address       string  `json:"address"`
	Confirmations int     `json:"confirmations"`
}

type BitstampXRPDepositResponse struct {
	Address        string `json:"address"`
	DestinationTag int64  `json:"destination_tag"`
}
