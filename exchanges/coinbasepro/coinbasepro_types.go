package coinbasepro

import "github.com/shopspring/decimal"

// Product holds product information
type Product struct {
	ID             string          `json:"id"`
	BaseCurrency   string          `json:"base_currency"`
	QuoteCurrency  string          `json:"quote_currency"`
	BaseMinSize    decimal.Decimal `json:"base_min_size,string"`
	BaseMaxSize    int64           `json:"base_max_size,string"`
	QuoteIncrement decimal.Decimal `json:"quote_increment,string"`
	DisplayName    string          `json:"string"`
}

// Ticker holds basic ticker information
type Ticker struct {
	TradeID int64           `json:"trade_id"`
	Price   decimal.Decimal `json:"price,string"`
	Size    decimal.Decimal `json:"size,string"`
	Time    string          `json:"time"`
}

// Trade holds executed trade information
type Trade struct {
	TradeID int64           `json:"trade_id"`
	Price   decimal.Decimal `json:"price,string"`
	Size    decimal.Decimal `json:"size,string"`
	Time    string          `json:"time"`
	Side    string          `json:"side"`
}

// History holds historic rate information
type History struct {
	Time   int64           `json:"time"`
	Low    decimal.Decimal `json:"low"`
	High   decimal.Decimal `json:"high"`
	Open   decimal.Decimal `json:"open"`
	Close  decimal.Decimal `json:"close"`
	Volume decimal.Decimal `json:"volume"`
}

// Stats holds last 24 hr data for coinbasepro
type Stats struct {
	Open   decimal.Decimal `json:"open,string"`
	High   decimal.Decimal `json:"high,string"`
	Low    decimal.Decimal `json:"low,string"`
	Volume decimal.Decimal `json:"volume,string"`
}

// Currency holds singular currency product information
type Currency struct {
	ID      string
	Name    string
	MinSize decimal.Decimal `json:"min_size,string"`
}

// ServerTime holds current requested server time information
type ServerTime struct {
	ISO   string  `json:"iso"`
	Epoch float64 `json:"epoch"`
}

// AccountResponse holds the details for the trading accounts
type AccountResponse struct {
	ID            string          `json:"id"`
	Currency      string          `json:"currency"`
	Balance       decimal.Decimal `json:"balance,string"`
	Available     decimal.Decimal `json:"available,string"`
	Hold          decimal.Decimal `json:"hold,string"`
	ProfileID     string          `json:"profile_id"`
	MarginEnabled bool            `json:"margin_enabled"`
	FundedAmount  decimal.Decimal `json:"funded_amount,string"`
	DefaultAmount decimal.Decimal `json:"default_amount,string"`
}

// AccountLedgerResponse holds account history information
type AccountLedgerResponse struct {
	ID        string          `json:"id"`
	CreatedAt string          `json:"created_at"`
	Amount    decimal.Decimal `json:"amount,string"`
	Balance   decimal.Decimal `json:"balance,string"`
	Type      string          `json:"type"`
	Details   interface{}     `json:"details"`
}

// AccountHolds contains the hold information about an account
type AccountHolds struct {
	ID        string          `json:"id"`
	AccountID string          `json:"account_id"`
	CreatedAt string          `json:"created_at"`
	UpdatedAt string          `json:"updated_at"`
	Amount    decimal.Decimal `json:"amount,string"`
	Type      string          `json:"type"`
	Reference string          `json:"ref"`
}

// GeneralizedOrderResponse is the generalized return type across order
// placement and information collation
type GeneralizedOrderResponse struct {
	ID             string          `json:"id"`
	Price          decimal.Decimal `json:"price,string"`
	Size           decimal.Decimal `json:"size,string"`
	ProductID      string          `json:"product_id"`
	Side           string          `json:"side"`
	Stp            string          `json:"stp"`
	Type           string          `json:"type"`
	TimeInForce    string          `json:"time_in_force"`
	PostOnly       bool            `json:"post_only"`
	CreatedAt      string          `json:"created_at"`
	FillFees       decimal.Decimal `json:"fill_fees,string"`
	FilledSize     decimal.Decimal `json:"filled_size,string"`
	ExecutedValue  decimal.Decimal `json:"executed_value,string"`
	Status         string          `json:"status"`
	Settled        bool            `json:"settled"`
	Funds          decimal.Decimal `json:"funds,string"`
	SpecifiedFunds decimal.Decimal `json:"specified_funds,string"`
	DoneReason     string          `json:"done_reason"`
	DoneAt         string          `json:"done_at"`
}

// Funding holds funding data
type Funding struct {
	ID            string          `json:"id"`
	OrderID       string          `json:"order_id"`
	ProfileID     string          `json:"profile_id"`
	Amount        decimal.Decimal `json:"amount,string"`
	Status        string          `json:"status"`
	CreatedAt     string          `json:"created_at"`
	Currency      string          `json:"currency"`
	RepaidAmount  decimal.Decimal `json:"repaid_amount"`
	DefaultAmount decimal.Decimal `json:"default_amount,string"`
	RepaidDefault bool            `json:"repaid_default"`
}

// MarginTransfer holds margin transfer details
type MarginTransfer struct {
	CreatedAt       string          `json:"created_at"`
	ID              string          `json:"id"`
	UserID          string          `json:"user_id"`
	ProfileID       string          `json:"profile_id"`
	MarginProfileID string          `json:"margin_profile_id"`
	Type            string          `json:"type"`
	Amount          decimal.Decimal `json:"amount,string"`
	Currency        string          `json:"currency"`
	AccountID       string          `json:"account_id"`
	MarginAccountID string          `json:"margin_account_id"`
	MarginProductID string          `json:"margin_product_id"`
	Status          string          `json:"status"`
	Nonce           int             `json:"nonce"`
}

// AccountOverview holds account information returned from position
type AccountOverview struct {
	Status  string `json:"status"`
	Funding struct {
		MaxFundingValue   decimal.Decimal `json:"max_funding_value,string"`
		FundingValue      decimal.Decimal `json:"funding_value,string"`
		OldestOutstanding struct {
			ID        string          `json:"id"`
			OrderID   string          `json:"order_id"`
			CreatedAt string          `json:"created_at"`
			Currency  string          `json:"currency"`
			AccountID string          `json:"account_id"`
			Amount    decimal.Decimal `json:"amount,string"`
		} `json:"oldest_outstanding"`
	} `json:"funding"`
	Accounts struct {
		LTC Account `json:"LTC"`
		ETH Account `json:"ETH"`
		USD Account `json:"USD"`
		BTC Account `json:"BTC"`
	} `json:"accounts"`
	MarginCall struct {
		Active bool            `json:"active"`
		Price  decimal.Decimal `json:"price,string"`
		Side   string          `json:"side"`
		Size   decimal.Decimal `json:"size,string"`
		Funds  decimal.Decimal `json:"funds,string"`
	} `json:"margin_call"`
	UserID    string `json:"user_id"`
	ProfileID string `json:"profile_id"`
	Position  struct {
		Type       string          `json:"type"`
		Size       decimal.Decimal `json:"size,string"`
		Complement decimal.Decimal `json:"complement,string"`
		MaxSize    decimal.Decimal `json:"max_size,string"`
	} `json:"position"`
	ProductID string `json:"product_id"`
}

// Account is a sub-type for account overview
type Account struct {
	ID            string          `json:"id"`
	Balance       decimal.Decimal `json:"balance,string"`
	Hold          decimal.Decimal `json:"hold,string"`
	FundedAmount  decimal.Decimal `json:"funded_amount,string"`
	DefaultAmount decimal.Decimal `json:"default_amount,string"`
}

// PaymentMethod holds payment method information
type PaymentMethod struct {
	ID            string `json:"id"`
	Type          string `json:"type"`
	Name          string `json:"name"`
	Currency      string `json:"currency"`
	PrimaryBuy    bool   `json:"primary_buy"`
	PrimarySell   bool   `json:"primary_sell"`
	AllowBuy      bool   `json:"allow_buy"`
	AllowSell     bool   `json:"allow_sell"`
	AllowDeposits bool   `json:"allow_deposits"`
	AllowWithdraw bool   `json:"allow_withdraw"`
	Limits        struct {
		Buy        []LimitInfo `json:"buy"`
		InstantBuy []LimitInfo `json:"instant_buy"`
		Sell       []LimitInfo `json:"sell"`
		Deposit    []LimitInfo `json:"deposit"`
	} `json:"limits"`
}

// LimitInfo is a sub-type for payment method
type LimitInfo struct {
	PeriodInDays int `json:"period_in_days"`
	Total        struct {
		Amount   decimal.Decimal `json:"amount,string"`
		Currency string          `json:"currency"`
	} `json:"total"`
}

// DepositWithdrawalInfo holds returned deposit information
type DepositWithdrawalInfo struct {
	ID       string          `json:"id"`
	Amount   decimal.Decimal `json:"amount,string"`
	Currency string          `json:"currency"`
	PayoutAt string          `json:"payout_at"`
}

// CoinbaseAccounts holds coinbase account information
type CoinbaseAccounts struct {
	ID                     string          `json:"id"`
	Name                   string          `json:"name"`
	Balance                decimal.Decimal `json:"balance,string"`
	Currency               string          `json:"currency"`
	Type                   string          `json:"type"`
	Primary                bool            `json:"primary"`
	Active                 bool            `json:"active"`
	WireDepositInformation struct {
		AccountNumber string `json:"account_number"`
		RoutingNumber string `json:"routing_number"`
		BankName      string `json:"bank_name"`
		BankAddress   string `json:"bank_address"`
		BankCountry   struct {
			Code string `json:"code"`
			Name string `json:"name"`
		} `json:"bank_country"`
		AccountName    string `json:"account_name"`
		AccountAddress string `json:"account_address"`
		Reference      string `json:"reference"`
	} `json:"wire_deposit_information"`
	SepaDepositInformation struct {
		Iban            string `json:"iban"`
		Swift           string `json:"swift"`
		BankName        string `json:"bank_name"`
		BankAddress     string `json:"bank_address"`
		BankCountryName string `json:"bank_country_name"`
		AccountName     string `json:"account_name"`
		AccountAddress  string `json:"account_address"`
		Reference       string `json:"reference"`
	} `json:"sep_deposit_information"`
}

// Report holds historical information
type Report struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
	CompletedAt string `json:"completed_at"`
	ExpiresAt   string `json:"expires_at"`
	FileURL     string `json:"file_url"`
	Params      struct {
		StartDate string `json:"start_date"`
		EndDate   string `json:"end_date"`
	} `json:"params"`
}

// Volume type contains trailing volume information
type Volume struct {
	ProductID      string          `json:"product_id"`
	ExchangeVolume decimal.Decimal `json:"exchange_volume,string"`
	Volume         decimal.Decimal `json:"volume,string"`
	RecordedAt     string          `json:"recorded_at"`
}

// OrderL1L2 is a type used in layer conversion
type OrderL1L2 struct {
	Price     decimal.Decimal
	Amount    decimal.Decimal
	NumOrders float64
}

// OrderL3 is a type used in layer conversion
type OrderL3 struct {
	Price   decimal.Decimal
	Amount  decimal.Decimal
	OrderID string
}

// OrderbookL1L2 holds level 1 and 2 order book information
type OrderbookL1L2 struct {
	Sequence int64       `json:"sequence"`
	Bids     []OrderL1L2 `json:"bids"`
	Asks     []OrderL1L2 `json:"asks"`
}

// OrderbookL3 holds level 3 order book information
type OrderbookL3 struct {
	Sequence int64     `json:"sequence"`
	Bids     []OrderL3 `json:"bids"`
	Asks     []OrderL3 `json:"asks"`
}

// OrderbookResponse is a generalized response for order books
type OrderbookResponse struct {
	Sequence int64           `json:"sequence"`
	Bids     [][]interface{} `json:"bids"`
	Asks     [][]interface{} `json:"asks"`
}

// FillResponse contains fill information from the exchange
type FillResponse struct {
	TradeID   int             `json:"trade_id"`
	ProductID string          `json:"product_id"`
	Price     decimal.Decimal `json:"price,string"`
	Size      decimal.Decimal `json:"size,string"`
	OrderID   string          `json:"order_id"`
	CreatedAt string          `json:"created_at"`
	Liquidity string          `json:"liquidity"`
	Fee       decimal.Decimal `json:"fee,string"`
	Settled   bool            `json:"settled"`
	Side      string          `json:"side"`
}

// WebsocketSubscribe takes in subscription information
type WebsocketSubscribe struct {
	Type      string `json:"type"`
	ProductID string `json:"product_id"`
}

// WebsocketReceived holds websocket received values
type WebsocketReceived struct {
	Type     string          `json:"type"`
	Time     string          `json:"time"`
	Sequence int             `json:"sequence"`
	OrderID  string          `json:"order_id"`
	Size     decimal.Decimal `json:"size,string"`
	Price    decimal.Decimal `json:"price,string"`
	Side     string          `json:"side"`
}

// WebsocketOpen collates open orders
type WebsocketOpen struct {
	Type          string          `json:"type"`
	Time          string          `json:"time"`
	Sequence      int             `json:"sequence"`
	OrderID       string          `json:"order_id"`
	Price         decimal.Decimal `json:"price,string"`
	RemainingSize decimal.Decimal `json:"remaining_size,string"`
	Side          string          `json:"side"`
}

// WebsocketDone holds finished order information
type WebsocketDone struct {
	Type          string          `json:"type"`
	Time          string          `json:"time"`
	Sequence      int             `json:"sequence"`
	Price         decimal.Decimal `json:"price,string"`
	OrderID       string          `json:"order_id"`
	Reason        string          `json:"reason"`
	Side          string          `json:"side"`
	RemainingSize decimal.Decimal `json:"remaining_size,string"`
}

// WebsocketMatch holds match information
type WebsocketMatch struct {
	Type         string          `json:"type"`
	TradeID      int             `json:"trade_id"`
	Sequence     int             `json:"sequence"`
	MakerOrderID string          `json:"maker_order_id"`
	TakerOrderID string          `json:"taker_order_id"`
	Time         string          `json:"time"`
	Size         decimal.Decimal `json:"size,string"`
	Price        decimal.Decimal `json:"price,string"`
	Side         string          `json:"side"`
}

// WebsocketChange holds change information
type WebsocketChange struct {
	Type     string          `json:"type"`
	Time     string          `json:"time"`
	Sequence int             `json:"sequence"`
	OrderID  string          `json:"order_id"`
	NewSize  decimal.Decimal `json:"new_size,string"`
	OldSize  decimal.Decimal `json:"old_size,string"`
	Price    decimal.Decimal `json:"price,string"`
	Side     string          `json:"side"`
}
