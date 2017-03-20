package btcc

type BTCCTicker struct {
	High       float64 `json:",string"`
	Low        float64 `json:",string"`
	Buy        float64 `json:",string"`
	Sell       float64 `json:",string"`
	Last       float64 `json:",string"`
	Vol        float64 `json:",string"`
	Date       int64
	Vwap       float64 `json:",string"`
	Prev_close float64 `json:",string"`
	Open       float64 `json:",string"`
}

type BTCCProfile struct {
	Username             string
	TradePasswordEnabled bool    `json:"trade_password_enabled,bool"`
	OTPEnabled           bool    `json:"otp_enabled,bool"`
	TradeFee             float64 `json:"trade_fee"`
	TradeFeeCNYLTC       float64 `json:"trade_fee_cnyltc"`
	TradeFeeBTCLTC       float64 `json:"trade_fee_btcltc"`
	DailyBTCLimit        float64 `json:"daily_btc_limit"`
	DailyLTCLimit        float64 `json:"daily_ltc_limit"`
	BTCDespoitAddress    string  `json:"btc_despoit_address"`
	BTCWithdrawalAddress string  `json:"btc_withdrawal_address"`
	LTCDepositAddress    string  `json:"ltc_deposit_address"`
	LTCWithdrawalAddress string  `json:"ltc_withdrawal_request"`
	APIKeyPermission     int64   `json:"api_key_permission"`
}

type BTCCCurrencyGeneric struct {
	Currency      string
	Symbol        string
	Amount        string
	AmountInt     int64   `json:"amount_integer"`
	AmountDecimal float64 `json:"amount_decimal"`
}

type BTCCOrder struct {
	ID         int64
	Type       string
	Price      float64
	Currency   string
	Amount     float64
	AmountOrig float64 `json:"amount_original"`
	Date       int64
	Status     string
	Detail     BTCCOrderDetail
}

type BTCCOrderDetail struct {
	Dateline int64
	Price    float64
	Amount   float64
}

type BTCCWithdrawal struct {
	ID          int64
	Address     string
	Currency    string
	Amount      float64
	Date        int64
	Transaction string
	Status      string
}

type BTCCDeposit struct {
	ID       int64
	Address  string
	Currency string
	Amount   float64
	Date     int64
	Status   string
}

type BTCCBidAsk struct {
	Price  float64
	Amount float64
}

type BTCCDepth struct {
	Bid []BTCCBidAsk
	Ask []BTCCBidAsk
}

type BTCCTransaction struct {
	ID        int64
	Type      string
	BTCAmount float64 `json:"btc_amount"`
	LTCAmount float64 `json:"ltc_amount"`
	CNYAmount float64 `json:"cny_amount"`
	Date      int64
}

type BTCCIcebergOrder struct {
	ID              int64
	Type            string
	Price           float64
	Market          string
	Amount          float64
	AmountOrig      float64 `json:"amount_original"`
	DisclosedAmount float64 `json:"disclosed_amount"`
	Variance        float64
	Date            int64
	Status          string
}

type BTCCStopOrder struct {
	ID          int64
	Type        string
	StopPrice   float64 `json:"stop_price"`
	TrailingAmt float64 `json:"trailing_amount"`
	TrailingPct float64 `json:"trailing_percentage"`
	Price       float64
	Market      string
	Amount      float64
	Date        int64
	Status      string
	OrderID     int64 `json:"order_id"`
}

type BTCCWebsocketOrder struct {
	Price       float64 `json:"price"`
	TotalAmount float64 `json:"totalamount"`
	Type        string  `json:"type"`
}

type BTCCWebsocketGroupOrder struct {
	Asks   []BTCCWebsocketOrder `json:"ask"`
	Bids   []BTCCWebsocketOrder `json:"bid"`
	Market string               `json:"market"`
}

type BTCCWebsocketTrade struct {
	Amount  float64 `json:"amount"`
	Date    float64 `json:"date"`
	Market  string  `json:"market"`
	Price   float64 `json:"price"`
	TradeID float64 `json:"trade_id"`
	Type    string  `json:"type"`
}

type BTCCWebsocketTicker struct {
	Buy       float64 `json:"buy"`
	Date      float64 `json:"date"`
	High      float64 `json:"high"`
	Last      float64 `json:"last"`
	Low       float64 `json:"low"`
	Market    string  `json:"market"`
	Open      float64 `json:"open"`
	PrevClose float64 `json:"prev_close"`
	Sell      float64 `json:"sell"`
	Volume    float64 `json:"vol"`
	Vwap      float64 `json:"vwap"`
}
