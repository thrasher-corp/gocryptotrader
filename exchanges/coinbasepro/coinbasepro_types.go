package coinbasepro

import (
	"encoding/json"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/currency"
)

// Product holds product information, returned by GetAllProducts and GetProductByID
type Product struct {
	ID                        string  `json:"id"`
	BaseCurrency              string  `json:"base_currency"`
	QuoteCurrency             string  `json:"quote_currency"`
	QuoteIncrement            float64 `json:"quote_increment,string"`
	BaseIncrement             float64 `json:"base_increment,string"`
	DisplayName               string  `json:"display_name"`
	MinimumMarketFunds        float64 `json:"min_market_funds,string"`
	MarginEnabled             bool    `json:"margin_enabled"`
	PostOnly                  bool    `json:"post_only"`
	LimitOnly                 bool    `json:"limit_only"`
	CancelOnly                bool    `json:"cancel_only"`
	Status                    string  `json:"status"`
	StatusMessage             string  `json:"status_message"`
	TradingDisabled           bool    `json:"trading_disabled"`
	ForeignExchangeStableCoin bool    `json:"fx_stablecoin"`
	MaxSlippagePercentage     float64 `json:"max_slippage_percentage,string"`
	AuctionMode               bool    `json:"auction_mode"`
	HighBidLimitPercentage    string  `json:"high_bid_limit_percentage"`
}

// Ticker holds basic ticker information, returned by GetTicker
type Ticker struct {
	Ask     float64   `json:"ask,string"`
	Bid     float64   `json:"bid,string"`
	Volume  float64   `json:"volume,string"`
	TradeID int32     `json:"trade_id"`
	Price   float64   `json:"price,string"`
	Size    float64   `json:"size,string"`
	Time    time.Time `json:"time"`
}

// Trade holds executed trade information, returned by GetTrades
type Trade struct {
	TradeID int64     `json:"trade_id"`
	Price   float64   `json:"price,string"`
	Size    float64   `json:"size,string"`
	Time    time.Time `json:"time"`
	Side    string    `json:"side"`
}

// History holds historic rate information, returned by GetHistoricRates
type History struct {
	Time   time.Time
	Low    float64
	High   float64
	Open   float64
	Close  float64
	Volume float64
}

// Stats holds 30 day and 24 hr data for a currency pair, returned by GetStats
type Stats struct {
	Open        float64 `json:"open,string"`
	High        float64 `json:"high,string"`
	Low         float64 `json:"low,string"`
	Last        float64 `json:"last,string"`
	Volume      float64 `json:"volume,string"`
	Volume30Day float64 `json:"volume_30day,string"`
}

// Currency holds information on a currency, returned by GetAllCurrencies and GetCurrencyByID
type Currency struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	MinSize       float64  `json:"min_size,string"`
	Status        string   `json:"status"`
	Message       string   `json:"message"`
	MaxPrecision  float64  `json:"max_precision,string"`
	ConvertibleTo []string `json:"convertible_to"`
	Details       struct {
		Type                  string   `json:"type"`
		NetworkConfirmations  int32    `json:"network_confirmations"`
		SortOrder             int32    `json:"sort_order"`
		CryptoAddressLink     string   `json:"crypto_address_link"`
		CryptoTransactionLink string   `json:"crypto_transaction_link"`
		PushPaymentMethods    []string `json:"push_payment_methods"`
		GroupTypes            []string `json:"group_types"`
		DisplayName           string   `json:"display_name"`
		ProcessingTimeSeconds float64  `json:"processing_time_seconds"`
		MinWithdrawalAmount   float64  `json:"min_withdrawal_amount"`
		MaxWithdrawalAmount   float64  `json:"max_withdrawal_amount"`
	} `json:"details"`
	DefaultNetwork    string `json:"default_network"`
	SupportedNetworks []struct {
		ID                    string  `json:"id"`
		Name                  string  `json:"name"`
		Status                string  `json:"status"`
		ContactAddress        string  `json:"contact_address"`
		CryptoAddressLink     string  `json:"crypto_address_link"`
		CryptoTransactionLink string  `json:"crypto_transaction_link"`
		MinWithdrawalAmount   float64 `json:"min_withdrawal_amount"`
		MaxWithdrawalAmount   float64 `json:"max_withdrawal_amount"`
		NetworkConfirmations  int32   `json:"network_confirmations"`
		ProcessingTimeSeconds int32   `json:"processing_time_seconds"`
	}
}

// ServerTime holds current requested server time information
// type ServerTime struct {
// 	ISO   time.Time `json:"iso"`
// 	Epoch float64   `json:"epoch"`
// }

// AccountResponse holds details for a trading account, returned by GetAllAccounts
// and GetAccountByID
type AccountResponse struct {
	ID             string  `json:"id"`
	Currency       string  `json:"currency"`
	Balance        float64 `json:"balance,string"`
	Hold           float64 `json:"hold,string"`
	Available      float64 `json:"available,string"`
	ProfileID      string  `json:"profile_id"`
	TradingEnabled bool    `json:"trading_enabled"`
	PendingDeposit string  `json:"pending_deposit"`
}

// AccountLedgerResponse holds account history information, returned by GetAccountLedger
type AccountLedgerResponse struct {
	ID        string    `json:"id"`
	Amount    float64   `json:"amount,string"`
	CreatedAt time.Time `json:"created_at"`
	Balance   float64   `json:"balance,string"`
	Type      string    `json:"type"`
	Details   struct {
		CoinbaseAccountID       string `json:"coinbase_account_id"`
		CoinbaseTransactionID   string `json:"coinbase_transaction_id"`
		CoinbasePaymentMethodID string `json:"coinbase_payment_method_id"`
	} `json:"details"`
}

// AccountHolds contains information on a hold, returned by GetHolds
type AccountHolds struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	Amount    float64   `json:"amount,string"`
	UpdatedAt time.Time `json:"updated_at"`
	Type      string    `json:"type"`
	Reference string    `json:"ref"`
}

// GeneralizedOrderResponse contains information on an order, returned by GetAllOrders,
// PlaceOrder, and GetOrderByID
type GeneralizedOrderResponse struct {
	ID             string    `json:"id"`
	Price          float64   `json:"price,string"`
	Size           float64   `json:"size,string"`
	ProductID      string    `json:"product_id"`
	ProfileID      string    `json:"profile_id"`
	Side           string    `json:"side"`
	Funds          float64   `json:"funds,string"`
	SpecifiedFunds float64   `json:"specified_funds,string"`
	Type           string    `json:"type"`
	TimeInForce    string    `json:"time_in_force"`
	ExpireTime     time.Time `json:"expire_time"`
	PostOnly       bool      `json:"post_only"`
	CreatedAt      time.Time `json:"created_at"`
	DoneAt         time.Time `json:"done_at"`
	DoneReason     string    `json:"done_reason"`
	RejectReason   string    `json:"reject_reason"`
	FillFees       float64   `json:"fill_fees,string"`
	FilledSize     float64   `json:"filled_size,string"`
	ExecutedValue  float64   `json:"executed_value,string"`
	Status         string    `json:"status"`
	Settled        bool      `json:"settled"`
	Stop           string    `json:"stop"`
	StopPrice      float64   `json:"stop_price,string"`
	FundingAmount  float64   `json:"funding_amount,string"`
	ClientOID      string    `json:"client_oid"`
	MarketType     string    `json:"market_type"`
}

// Funding holds funding data
// type Funding struct {
// 	ID            string    `json:"id"`
// 	OrderID       string    `json:"order_id"`
// 	ProfileID     string    `json:"profile_id"`
// 	Amount        float64   `json:"amount,string"`
// 	Status        string    `json:"status"`
// 	CreatedAt     time.Time `json:"created_at"`
// 	Currency      string    `json:"currency"`
// 	RepaidAmount  float64   `json:"repaid_amount"`
// 	DefaultAmount float64   `json:"default_amount,string"`
// 	RepaidDefault bool      `json:"repaid_default"`
// }

// MarginTransfer holds margin transfer details
// type MarginTransfer struct {
// 	CreatedAt       time.Time `json:"created_at"`
// 	ID              string    `json:"id"`
// 	UserID          string    `json:"user_id"`
// 	ProfileID       string    `json:"profile_id"`
// 	MarginProfileID string    `json:"margin_profile_id"`
// 	Type            string    `json:"type"`
// 	Amount          float64   `json:"amount,string"`
// 	Currency        string    `json:"currency"`
// 	AccountID       string    `json:"account_id"`
// 	MarginAccountID string    `json:"margin_account_id"`
// 	MarginProductID string    `json:"margin_product_id"`
// 	Status          string    `json:"status"`
// 	Nonce           int       `json:"nonce"`
// }

// AccountOverview holds account information returned from position
// type AccountOverview struct {
// 	Status  string `json:"status"`
// 	Funding struct {
// 		MaxFundingValue   float64 `json:"max_funding_value,string"`
// 		FundingValue      float64 `json:"funding_value,string"`
// 		OldestOutstanding struct {
// 			ID        string    `json:"id"`
// 			OrderID   string    `json:"order_id"`
// 			CreatedAt time.Time `json:"created_at"`
// 			Currency  string    `json:"currency"`
// 			AccountID string    `json:"account_id"`
// 			Amount    float64   `json:"amount,string"`
// 		} `json:"oldest_outstanding"`
// 	} `json:"funding"`
// 	Accounts struct {
// 		LTC Account `json:"LTC"`
// 		ETH Account `json:"ETH"`
// 		USD Account `json:"USD"`
// 		BTC Account `json:"BTC"`
// 	} `json:"accounts"`
// 	MarginCall struct {
// 		Active bool    `json:"active"`
// 		Price  float64 `json:"price,string"`
// 		Side   string  `json:"side"`
// 		Size   float64 `json:"size,string"`
// 		Funds  float64 `json:"funds,string"`
// 	} `json:"margin_call"`
// 	UserID    string `json:"user_id"`
// 	ProfileID string `json:"profile_id"`
// 	Position  struct {
// 		Type       string  `json:"type"`
// 		Size       float64 `json:"size,string"`
// 		Complement float64 `json:"complement,string"`
// 		MaxSize    float64 `json:"max_size,string"`
// 	} `json:"position"`
// 	ProductID string `json:"product_id"`
// }

// Account is a sub-type for account overview
// type Account struct {
// 	ID            string  `json:"id"`
// 	Balance       float64 `json:"balance,string"`
// 	Hold          float64 `json:"hold,string"`
// 	FundedAmount  float64 `json:"funded_amount,string"`
// 	DefaultAmount float64 `json:"default_amount,string"`
// }

// PaymentMethod holds payment method information, returned by GetPayMethods
type PaymentMethod struct {
	ID           string    `json:"id"`
	Type         string    `json:"type"`
	Name         string    `json:"name"`
	Currency     string    `json:"currency"`
	PrimaryBuy   bool      `json:"primary_buy"`
	PrimarySell  bool      `json:"primary_sell"`
	InstantBuy   bool      `json:"instant_buy"`
	InstantSell  bool      `json:"instant_sell"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Resource     string    `json:"resource"`
	ResourcePath string    `json:"resource_path"`
	Verified     bool      `json:"verified"`
	Limits       struct {
		Type string `json:"type"`
		Name string `json:"name"`
	} `json:"limits"`
	AllowBuy      bool `json:"allow_buy"`
	AllowSell     bool `json:"allow_sell"`
	AllowDeposit  bool `json:"allow_deposit"`
	AllowWithdraw bool `json:"allow_withdraw"`
	FiatAccount   struct {
		ID           string `json:"id"`
		Resource     string `json:"resource"`
		ResourcePath string `json:"resource_path"`
	} `json:"fiat_account"`
	CryptoAccount struct {
		ID           string `json:"id"`
		Resource     string `json:"resource"`
		ResourcePath string `json:"resource_path"`
	} `json:"crypto_account"`
	AvailableBalance struct {
		Amount   float64 `json:"amount"`
		Currency string  `json:"currency"`
		Scale    string  `json:"scale"`
	} `json:"available_balance"`
	PickerData struct {
		Symbol                string `json:"symbol"`
		CustomerName          string `json:"customer_name"`
		AccountName           string `json:"account_name"`
		AccountNumber         string `json:"account_number"`
		AccountType           string `json:"account_type"`
		InstitutionCode       string `json:"institution_code"`
		InstitutionName       string `json:"institution_name"`
		Iban                  string `json:"iban"`
		Swift                 string `json:"swift"`
		PaypalEmail           string `json:"paypal_email"`
		PaypalOwner           string `json:"paypal_owner"`
		RoutingNumber         string `json:"routing_number"`
		InstitutionIdentifier string `json:"institution_identifier"`
		BankName              string `json:"bank_name"`
		BranchName            string `json:"branch_name"`
		IconURL               string `json:"icon_url"`
		Balance               struct {
			Amount   float64 `json:"amount,string"`
			Currency string  `json:"currency"`
		} `json:"balance"`
	} `json:"picker_data"`
	HoldBusinessDays   int64  `json:"hold_business_days"`
	HoldDays           int64  `json:"hold_days"`
	VerificationMethod string `json:"verificationMethod"`
	CDVStatus          string `json:"cdvStatus"`
}

// LimitInfo is a sub-type for payment method
// type LimitInfo struct {
// 	PeriodInDays int `json:"period_in_days"`
// 	Total        struct {
// 		Amount   float64 `json:"amount,string"`
// 		Currency string  `json:"currency"`
// 	} `json:"total"`
// }

// DepositWithdrawalInfo holds information provided when depositing or
// withdrawing from payment methods. Returned by DepositViaCoinbase,
// DepositViaPaymentMethod, WithdrawViaCoinbase, WithdrawCrypto, and
// WithdrawViaPaymentMethod
type DepositWithdrawalInfo struct {
	ID       string    `json:"id"`
	Amount   float64   `json:"amount,string"`
	Currency string    `json:"currency"`
	PayoutAt time.Time `json:"payout_at"`
	Fee      float64   `json:"fee,string"`
	Subtotal float64   `json:"subtotal,string"`
}

// CoinbaseAccounts holds coinbase account information, returned by GetCoinbaseWallets
type CoinbaseAccounts struct {
	ID                     string  `json:"id"`
	Name                   string  `json:"name"`
	Balance                float64 `json:"balance,string"`
	Currency               string  `json:"currency"`
	Type                   string  `json:"type"`
	Primary                bool    `json:"primary"`
	Active                 bool    `json:"active"`
	AvailableOnConsumer    bool    `json:"available_on_consumer"`
	Ready                  bool    `json:"ready"`
	WireDepositInformation struct {
		AccountNumber string `json:"account_number"`
		RoutingNumber string `json:"routing_number"`
		BankName      string `json:"bank_name"`
		BankAddress   string `json:"bank_address"`
		BankCountry   struct {
			Name string `json:"name"`
			Code string `json:"code"`
		} `json:"bank_country"`
		AccountName    string `json:"account_name"`
		AccountAddress string `json:"account_address"`
		Reference      string `json:"reference"`
	} `json:"wire_deposit_information"`
	SwiftDepositInformation struct {
		AccountNumber string `json:"account_number"`
		BankName      string `json:"bank_name"`
		BankAddress   string `json:"bank_address"`
		BankCountry   struct {
			Name string `json:"name"`
			Code string `json:"code"`
		} `json:"bank_country"`
		AccountName    string `json:"account_name"`
		AccountAddress string `json:"account_address"`
		Reference      string `json:"reference"`
	} `json:"swift_deposit_information"`
	SepaDepositInformation struct {
		Iban        string `json:"iban"`
		Swift       string `json:"swift"`
		BankName    string `json:"bank_name"`
		BankAddress string `json:"bank_address"`
		BankCountry struct {
			Name string `json:"name"`
			Code string `json:"code"`
		} `json:"bank_country"`
		AccountName    string `json:"account_name"`
		AccountAddress string `json:"account_address"`
		Reference      string `json:"reference"`
	} `json:"sepa_deposit_information"`
	UkDepositInformation struct {
		SortCode      string `json:"sort_code"`
		AccountNumber string `json:"account_number"`
		BankName      string `json:"bank_name"`
		AccountName   string `json:"account_name"`
		Reference     string `json:"reference"`
	} `json:"uk_deposit_information"`
	DestinationTagName  string  `json:"destination_tag_name"`
	DestinationTagRegex string  `json:"destination_tag_regex"`
	HoldBalance         float64 `json:"hold_balance,string"`
	HoldCurrency        string  `json:"hold_currency"`
}

// Report holds information on user-generated reports, returned by GetAllReports
// and GetReportByID
type Report struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	CreatedAt   time.Time `json:"created_at"`
	CompletedAt time.Time `json:"completed_at"`
	ExpiresAt   time.Time `json:"expires_at"`
	Status      string    `json:"status"`
	UserID      string    `json:"user_id"`
	FileURL     string    `json:"file_url"`
	Params      struct {
		StartDate time.Time `json:"start_date"`
		EndDate   time.Time `json:"end_date"`
		Format    string    `json:"format"`
		ProductID string    `json:"product_id"`
		AccountID string    `json:"account_id"`
		ProfileID string    `json:"profile_id"`
		Email     string    `json:"email"`
		User      struct {
			ID                      string        `json:"id"`
			CreatedAt               time.Time     `json:"created_at"`
			ActiveAt                time.Time     `json:"active_at"`
			Name                    string        `json:"name"`
			Email                   string        `json:"email"`
			Roles                   []interface{} `json:"roles"`
			IsBanned                bool          `json:"is_banned"`
			Permissions             interface{}   `json:"permissions"`
			UserType                string        `json:"user_type"`
			FulfillsNewRequirements bool          `json:"fulfills_new_requirements"`
			Flags                   interface{}   `json:"flags"`
			Details                 interface{}   `json:"details"`
			OauthClient             string        `json:"oauth_client"`
			Preferences             struct {
				PreferredMarket              string    `json:"preferred_market"`
				MarginTermsCompletedInUTC    time.Time `json:"margin_terms_completed_in_utc"`
				MarginTutorialCompletedInUTC time.Time `json:"margin_tutorial_completed_in_utc"`
			} `json:"preferences"`
			HasDefault                bool      `json:"has_default"`
			StateCode                 string    `json:"state_code"`
			CBDataFromCache           bool      `json:"cb_data_from_cache"`
			TwoFactorMethod           string    `json:"two_factor_method"`
			LegalName                 string    `json:"legal_name"`
			TermsAccepted             time.Time `json:"terms_accepted"`
			HasClawbackPaymentPending bool      `json:"has_clawback_payment_pending"`
			HasRestrictedAssets       bool      `json:"has_restricted_assets"`
		} `json:"user"`
		NewYorkState   bool      `json:"new_york_state"`
		DateTime       time.Time `json:"date_time"`
		GroupByProfile bool      `json:"group_by_profile"`
	} `json:"params"`
	FileCount uint64 `json:"file_count"`
}

// Volume type contains trailing volume information
// type Volume struct {
// 	ProductID      string  `json:"product_id"`
// 	ExchangeVolume float64 `json:"exchange_volume,string"`
// 	Volume         float64 `json:"volume,string"`
// 	RecordedAt     string  `json:"recorded_at"`
// }

// InterOrderDetail is used to make intermediary orderbook handling easier
type InterOrderDetail [][3]interface{}

// OrderbookIntermediaryResponse is used while processing the orderbook
type OrderbookIntermediaryResponse struct {
	Bids        InterOrderDetail `json:"bids"`
	Asks        InterOrderDetail `json:"asks"`
	Sequence    float64          `json:"sequence"`
	AuctionMode bool             `json:"auction_mode"`
	Auction     struct {
		OpenPrice    float64   `json:"open_price,string"`
		OpenSize     float64   `json:"open_size,string"`
		BestBidPrice float64   `json:"best_bid_price,string"`
		BestBidSize  float64   `json:"best_bid_size,string"`
		BestAskPrice float64   `json:"best_ask_price,string"`
		BestAskSize  float64   `json:"best_ask_size,string"`
		AuctionState string    `json:"auction_state"`
		CanOpen      string    `json:"can_open"`
		Time         time.Time `json:"time"`
	}
	Time time.Time `json:"time"`
}

// GenOrderDetail is a subtype used for the final state of the orderbook
type GenOrderDetail struct {
	Price     float64
	Amount    float64
	NumOrders float64
	OrderID   string
}

// OrderbookResponse is the final state of the orderbook, returned by GetOrderbook
type OrderbookFinalResponse struct {
	Bids        []GenOrderDetail `json:"bids"`
	Asks        []GenOrderDetail `json:"asks"`
	Sequence    float64          `json:"sequence"`
	AuctionMode bool             `json:"auction_mode"`
	Auction     struct {
		OpenPrice    float64   `json:"open_price,string"`
		OpenSize     float64   `json:"open_size,string"`
		BestBidPrice float64   `json:"best_bid_price,string"`
		BestBidSize  float64   `json:"best_bid_size,string"`
		BestAskPrice float64   `json:"best_ask_price,string"`
		BestAskSize  float64   `json:"best_ask_size,string"`
		AuctionState string    `json:"auction_state"`
		CanOpen      string    `json:"can_open"`
		Time         time.Time `json:"time"`
	}
	Time time.Time `json:"time"`
}

// FillResponse contains fill information, returned by GetFills
type FillResponse struct {
	TradeID         int32     `json:"trade_id"`
	ProductID       string    `json:"product_id"`
	OrderID         string    `json:"order_id"`
	UserID          string    `json:"user_id"`
	ProfileID       string    `json:"profile_id"`
	Liquidity       string    `json:"liquidity"`
	Price           float64   `json:"price,string"`
	Size            float64   `json:"size,string"`
	Fee             float64   `json:"fee,string"`
	CreatedAt       time.Time `json:"created_at"`
	Side            string    `json:"side"`
	Settled         bool      `json:"settled"`
	USDVolume       float64   `json:"usd_volume,string"`
	MarketType      string    `json:"market_type"`
	FundingCurrency string    `json:"funding_currency"`
}

// WebsocketSubscribe takes in subscription information
type WebsocketSubscribe struct {
	Type       string       `json:"type"`
	ProductIDs []string     `json:"product_ids,omitempty"`
	Channels   []WsChannels `json:"channels,omitempty"`
	Signature  string       `json:"signature,omitempty"`
	Key        string       `json:"key,omitempty"`
	Passphrase string       `json:"passphrase,omitempty"`
	Timestamp  string       `json:"timestamp,omitempty"`
}

// WsChannels defines outgoing channels for subscription purposes
type WsChannels struct {
	Name       string   `json:"name"`
	ProductIDs []string `json:"product_ids,omitempty"`
}

// wsOrderReceived holds websocket received values
type wsOrderReceived struct {
	Type          string    `json:"type"`
	OrderID       string    `json:"order_id"`
	OrderType     string    `json:"order_type"`
	Size          float64   `json:"size,string"`
	Price         float64   `json:"price,omitempty,string"`
	Funds         float64   `json:"funds,omitempty,string"`
	Side          string    `json:"side"`
	ClientOID     string    `json:"client_oid"`
	ProductID     string    `json:"product_id"`
	Sequence      int64     `json:"sequence"`
	Time          time.Time `json:"time"`
	RemainingSize float64   `json:"remaining_size,string"`
	NewSize       float64   `json:"new_size,string"`
	OldSize       float64   `json:"old_size,string"`
	Reason        string    `json:"reason"`
	Timestamp     float64   `json:"timestamp,string"`
	UserID        string    `json:"user_id"`
	ProfileID     string    `json:"profile_id"`
	StopType      string    `json:"stop_type"`
	StopPrice     float64   `json:"stop_price,string"`
	TakerFeeRate  float64   `json:"taker_fee_rate,string"`
	Private       bool      `json:"private"`
	TradeID       int64     `json:"trade_id"`
	MakerOrderID  string    `json:"maker_order_id"`
	TakerOrderID  string    `json:"taker_order_id"`
	TakerUserID   string    `json:"taker_user_id"`
}

// WebsocketHeartBeat defines JSON response for a heart beat message
type WebsocketHeartBeat struct {
	Type        string `json:"type"`
	Sequence    int64  `json:"sequence"`
	LastTradeID int64  `json:"last_trade_id"`
	ProductID   string `json:"product_id"`
	Time        string `json:"time"`
}

// WebsocketTicker defines ticker websocket response
type WebsocketTicker struct {
	Type      string        `json:"type"`
	Sequence  int64         `json:"sequence"`
	ProductID currency.Pair `json:"product_id"`
	Price     float64       `json:"price,string"`
	Open24H   float64       `json:"open_24h,string"`
	Volume24H float64       `json:"volume_24h,string"`
	Low24H    float64       `json:"low_24h,string"`
	High24H   float64       `json:"high_24h,string"`
	Volume30D float64       `json:"volume_30d,string"`
	BestBid   float64       `json:"best_bid,string"`
	BestAsk   float64       `json:"best_ask,string"`
	Side      string        `json:"side"`
	Time      time.Time     `json:"time"`
	TradeID   int64         `json:"trade_id"`
	LastSize  float64       `json:"last_size,string"`
}

// WebsocketOrderbookSnapshot defines a snapshot response
type WebsocketOrderbookSnapshot struct {
	ProductID string      `json:"product_id"`
	Type      string      `json:"type"`
	Bids      [][2]string `json:"bids"`
	Asks      [][2]string `json:"asks"`
	Time      time.Time   `json:"time"`
}

// WebsocketL2Update defines an update on the L2 orderbooks
type WebsocketL2Update struct {
	Type      string      `json:"type"`
	ProductID string      `json:"product_id"`
	Time      time.Time   `json:"time"`
	Changes   [][3]string `json:"changes"`
}

type wsMsgType struct {
	Type      string `json:"type"`
	Sequence  int64  `json:"sequence"`
	ProductID string `json:"product_id"`
}

type wsStatus struct {
	Currencies []struct {
		ConvertibleTo []string    `json:"convertible_to"`
		Details       struct{}    `json:"details"`
		ID            string      `json:"id"`
		MaxPrecision  float64     `json:"max_precision,string"`
		MinSize       float64     `json:"min_size,string"`
		Name          string      `json:"name"`
		Status        string      `json:"status"`
		StatusMessage interface{} `json:"status_message"`
	} `json:"currencies"`
	Products []struct {
		BaseCurrency   string      `json:"base_currency"`
		BaseIncrement  float64     `json:"base_increment,string"`
		BaseMaxSize    float64     `json:"base_max_size,string"`
		BaseMinSize    float64     `json:"base_min_size,string"`
		CancelOnly     bool        `json:"cancel_only"`
		DisplayName    string      `json:"display_name"`
		ID             string      `json:"id"`
		LimitOnly      bool        `json:"limit_only"`
		MaxMarketFunds float64     `json:"max_market_funds,string"`
		MinMarketFunds float64     `json:"min_market_funds,string"`
		PostOnly       bool        `json:"post_only"`
		QuoteCurrency  string      `json:"quote_currency"`
		QuoteIncrement float64     `json:"quote_increment,string"`
		Status         string      `json:"status"`
		StatusMessage  interface{} `json:"status_message"`
	} `json:"products"`
	Type string `json:"type"`
}

// RequestParamsTimeForceType Time in force
// type RequestParamsTimeForceType string

// var (
// 	// CoinbaseRequestParamsTimeGTC GTC
// 	CoinbaseRequestParamsTimeGTC = RequestParamsTimeForceType("GTC")

// 	// CoinbaseRequestParamsTimeIOC IOC
// 	CoinbaseRequestParamsTimeIOC = RequestParamsTimeForceType("IOC")
// )

// TransferResponse contains information on a transfer, returned by GetAccountTransfers,
// GetAllTransfers, and GetTransferByID
type TransferResponse struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	CreatedAt   time.Time `json:"created_at"`
	CompletedAt time.Time `json:"completed_at"`
	CanceledAt  time.Time `json:"canceled_at"`
	ProcessedAt time.Time `json:"processed_at"`
	Amount      float64   `json:"amount,string"`
	Details     struct {
		CoinbaseAccountID       string `json:"coinbase_account_id"`
		CoinbaseTransactionID   string `json:"coinbase_transaction_id"`
		CoinbasePaymentMethodID string `json:"coinbase_payment_method_id"`
	} `json:"details"`
	UserNonce int64 `json:"user_nonce"`
}

// TravelRule contains information on a travel rule, returned by GetTravelRules
// and CreateTravelRule
type TravelRule struct {
	ID            string    `json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	Address       string    `json:"address"`
	OriginName    string    `json:"originator_name"`
	OriginCountry string    `json:"originator_country"`
}

// Params is used within functions to make the setting of parameters easier
type Params struct {
	urlVals url.Values
}

// GetAddressResponse contains information on addresses, returned by GetAddressBook
type GetAddressResponse struct {
	ID                 string    `json:"id"`
	Address            string    `json:"address"`
	DestinationTag     string    `json:"destination_tag"`
	Currency           string    `json:"currency"`
	Label              string    `json:"label"`
	AddressBookAddedAt time.Time `json:"address_book_added_at"`
	LastUsed           time.Time `json:"last_used"`
	VerifiedSelfHosted bool      `json:"is_verified_self_hosted_wallet"`
	VASPID             string    `json:"vasp_id"`
}

// To is part of the struct expected by the exchange for the AddAddresses function
type To struct {
	Address        string `json:"address"`
	DestinationTag string `json:"destination_tag"`
}

// AddAddressRequest is the struct expected by the exchange for the AddAddresses function
type AddAddressRequest struct {
	Currency           string `json:"currency"`
	To                 `json:"to"`
	Label              string `json:"label"`
	VerifiedSelfHosted bool   `json:"is_verified_self_hosted_wallet"`
	VaspID             string `json:"vasp_id"`
	// TODO: It also lets us add an arbitrary amount of strings under this object,
	// but doesn't explain what they do. Investigate more later.
}

// AddAddressResponse contains information on the addresses just added, returned by
// AddAddresses
type AddAddressResponse struct {
	ID          string `json:"id"`
	Address     string `json:"address"`
	AddressInfo struct {
		Address        string `json:"address"`
		DisplayAddress string `json:"display_address"`
		DestinationTag string `json:"destination_tag"`
	} `json:"address_info"`
	Currency                     string    `json:"currency"`
	Label                        string    `json:"label"`
	DisplayAddress               string    `json:"display_address"`
	Trusted                      bool      `json:"trusted"`
	AddressBooked                bool      `json:"address_booked"`
	AddressBookAddedAt           time.Time `json:"address_book_added_at"`
	LastUsed                     time.Time `json:"last_used"`
	AddressBookEntryPendingUntil time.Time `json:"address_book_entry_pending_until"`
	VerifiedSelfHosted           bool      `json:"is_verified_self_hosted_wallet"`
	VaspID                       string    `json:"vasp_id"`
}

// CryptoAddressResponse contains information on the one-time address generated for
// depositing crypto, returned by GenerateCryptoAddress
type CryptoAddressResponse struct {
	ID          string `json:"id"`
	Address     string `json:"address"`
	AddressInfo struct {
		Address        string `json:"address"`
		DestinationTag string `json:"destination_tag"`
	} `json:"address_info"`
	Name         string    `json:"name"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Network      string    `json:"network"`
	URIScheme    string    `json:"uri_scheme"`
	Resource     string    `json:"resource"`
	ResourcePath string    `json:"resource_path"`
	Warnings     []struct {
		Title    string `json:"title"`
		Details  string `json:"details"`
		ImageURL string `json:"image_url"`
	} `json:"warnings"`
	LegacyAddress  string `json:"legacy_address"`
	DestinationTag string `json:"destination_tag"`
	DepositURI     string `json:"deposit_uri"`
	CallbackURL    string `json:"callback_url"`
}

// ConvertResponse contains information about a completed currency conversion, returned
// by ConvertCurrency and GetConversionByID
type ConvertResponse struct {
	ID            string `json:"id"`
	Amount        string `json:"amount"`
	FromAccountID string `json:"from_account_id"`
	ToAccountID   string `json:"to_account_id"`
	From          string `json:"from"`
	To            string `json:"to"`
}

// WithdrawalFeeEstimate is the exchange's estimate of the fee for a withdrawal, returned
// by GetWithdrawalFeeEstimate
type WithdrawalFeeEstimate struct {
	Fee              string `json:"fee"`
	FeeBeforeSubsidy string `json:"fee_before_subsidy"`
}

// FeeResponse contains current taker and maker fee rates, as well as 30-day trailing
// volume. Returned by GetFees
type FeeResponse struct {
	TakerFeeRate float64 `json:"taker_fee_rate,string"`
	MakerFeeRate float64 `json:"maker_fee_rate,string"`
	USDVolume    float64 `json:"usd_volume,string"`
}

// PriceMap is used to properly unmarshal the response from GetSignedPrices
type PriceMap map[string]float64

func (pm *PriceMap) UnmarshalJSON(data []byte) error {
	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	*pm = make(PriceMap)
	for k, v := range m {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return err
		}
		(*pm)[k] = f
	}
	return nil
}

// SignedPrices contains cryptographically signed prices, alongside other information
// necessary for them to be posted on-chain using Compound's Open Oracle smart contract.
// Returned by GetSignedPrices
type SignedPrices struct {
	Timestamp  string   `json:"timestamp"`
	Messages   []string `json:"messages"`
	Signatures []string `json:"signatures"`
	Prices     PriceMap `json:"prices"`
}

// Profile contains information on a profile. Returned by GetAllProfiles, CreateAProfile,
// GetProfileByID, and RenameProfile
type Profile struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Name      string    `json:"name"`
	Active    bool      `json:"active"`
	IsDefault bool      `json:"is_default"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateReportResponse contains information on a newly-created report, returned by
// CreateReport
type CreateReportResponse struct {
	ID     string `json:"id"`
	Type   string `json:"type"`
	Status string `json:"status"`
}

// ReportBalanceStruct is used internally when crafting a CreateReport request
type ReportBalanceStruct struct {
	DateTime string
}

// ReportFillsTaxStruct is used internally when crafting a CreateReport request
type ReportFillsTaxStruct struct {
	StartDate string
	EndDate   string
	ProductID string
}

// ReportAccountStruct is used internally when crafting a CreateReport request
type ReportAccountStruct struct {
	StartDate string
	EndDate   string
	AccountID string
}

// ExchangeLimits contains information on payment method transfer limits, returned
// by GetExchangeLimits
type ExchangeLimits struct {
	TransferLimits struct {
		Buy              interface{} `json:"buy"`
		Sell             interface{} `json:"sell"`
		ExchangeWithdraw interface{} `json:"exchange_withdraw"`
		Ach              []struct {
			Max          float64 `json:"max,string"`
			Remaining    float64 `json:"remaining,string"`
			PeriodInDays int32   `json:"period_in_days"`
		} `json:"ach"`
		AchNoBalance         interface{} `json:"ach_no_balance"`
		CreditDebitCard      interface{} `json:"credit_debit_card"`
		Secure3DBuy          interface{} `json:"secure3d_buy"`
		PaypalBuy            interface{} `json:"paypal_buy"`
		PaypalWithdrawal     interface{} `json:"paypal_withdrawal"`
		IdealDeposit         interface{} `json:"ideal_deposit"`
		SofortDeposit        interface{} `json:"sofort_deposit"`
		InstantAchWithdrawal interface{} `json:"instant_ach_withdrawal"`
	} `json:"transfer_limits"`
	LimitCurrency string `json:"limit_currency"`
}

// WrappedAssetResponse contains information on a wrapped asset, returned by
// GetWrappedAssetByID
type WrappedAssetResponse struct {
	ID                string                  `json:"id"`
	CirculatingSupply float64                 `json:"circulating_supply,string"`
	TotalSupply       float64                 `json:"total_supply,string"`
	ConversionRate    float64                 `json:"conversion_rate,string"`
	APY               convert.StringToFloat64 `json:"apy"`
}

// AllWrappedAssetResponse contains information on all wrapped assets, returned by
// GetAllWrappedAssets
type AllWrappedAssetResponse struct {
	WrappedAssetResponse []struct{} `json:"wrapped_assets"`
}

// StakeWrap contains information on a stake wrap, returned by GetAllStakeWraps and
// GetStakeWrapByID
type StakeWrap struct {
	ID             string    `json:"id"`
	FromAmount     float64   `json:"from_amount,string"`
	ToAmount       float64   `json:"to_amount,string"`
	FromAccountID  string    `json:"from_account_id"`
	ToAccountID    string    `json:"to_account_id"`
	FromCurrency   string    `json:"from_currency"`
	ToCurrency     string    `json:"to_currency"`
	Status         string    `json:"status"`
	ConversionRate float64   `json:"conversion_rate,string"`
	CreatedAt      time.Time `json:"created_at"`
	CompletedAt    time.Time `json:"completed_at"`
	CanceledAt     time.Time `json:"canceled_at"`
}

// WrappedAssetConversionRate contains the conversion rate for a wrapped asset, returned
// by GetWrappedAssetConversionRate
type WrappedAssetConversionRate struct {
	Amount float64 `json:"amount,string"`
}
