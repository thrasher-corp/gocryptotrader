package coinbasepro

import (
	"net/url"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// CoinbasePro is the overarching type across the coinbasepro package
type CoinbasePro struct {
	exchange.Base
}

// Version is used for the niche cases where the Version of the API must be specified and passed
// around for proper functionality
type Version bool

// FiatTransferType is used so that we don't need to duplicate the four fiat transfer-related
// endpoints under version 2 of the API
type FiatTransferType bool

// ValCur is a sub-struct used in the type Account
type ValCur struct {
	Value    float64 `json:"value,string"`
	Currency string  `json:"currency"`
}

// Account holds details for a trading account, returned by GetAccountByID and used as
// a sub-struct in the type AllAccountsResponse
type Account struct {
	UUID             string    `json:"uuid"`
	Name             string    `json:"name"`
	Currency         string    `json:"currency"`
	AvailableBalance ValCur    `json:"available_balance"`
	Default          bool      `json:"default"`
	Active           bool      `json:"active"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	DeletedAt        time.Time `json:"deleted_at"`
	Type             string    `json:"type"`
	Ready            bool      `json:"ready"`
	Hold             ValCur    `json:"hold"`
}

// AllAccountsResponse holds many Account structs, as well as pagination information,
// returned by GetAllAccounts
type AllAccountsResponse struct {
	Accounts []Account `json:"accounts"`
	HasNext  bool      `json:"has_next"`
	Cursor   string    `json:"cursor"`
	Size     uint8     `json:"size"`
}

// OneAccountResponse is a temporary struct used for unmarshalling in GetAccountByID
type OneAccountResponse struct {
	Account Account `json:"account"`
}

// PriSiz is a sub-struct used in the type BestBidAsk
type PriSiz struct {
	Price float64 `json:"price,string"`
	Size  float64 `json:"size,string"`
}

// ProductBook holds bid and ask prices for a particular product, returned by GetProductBook
// and used as a sub-struct in the type BestBidAsk
type ProductBook struct {
	ProductID string    `json:"product_id"`
	Bids      []PriSiz  `json:"bids"`
	Asks      []PriSiz  `json:"asks"`
	Time      time.Time `json:"time"`
}

// BestBidAsk holds the best bid and ask prices for a variety of products, returned by
// GetBestBidAsk
type BestBidAsk struct {
	Pricebooks []ProductBook `json:"pricebooks"`
}

// ProductBook holds bids and asks for a particular product, returned by GetProductBook
type ProductBookResponse struct {
	Pricebook ProductBook `json:"pricebook"`
}

// Product holds product information, returned by GetAllProducts and GetProductByID
type Product struct {
	ID                        string                  `json:"product_id"`
	Price                     convert.StringToFloat64 `json:"price"`
	PricePercentageChange24H  convert.StringToFloat64 `json:"price_percentage_change_24h"`
	Volume24H                 convert.StringToFloat64 `json:"volume_24h"`
	VolumePercentageChange24H convert.StringToFloat64 `json:"volume_percentage_change_24h"`
	BaseIncrement             convert.StringToFloat64 `json:"base_increment"`
	QuoteIncrement            convert.StringToFloat64 `json:"quote_increment"`
	QuoteMinSize              convert.StringToFloat64 `json:"quote_min_size"`
	QuoteMaxSize              convert.StringToFloat64 `json:"quote_max_size"`
	BaseMinSize               convert.StringToFloat64 `json:"base_min_size"`
	BaseMaxSize               convert.StringToFloat64 `json:"base_max_size"`
	BaseName                  string                  `json:"base_name"`
	QuoteName                 string                  `json:"quote_name"`
	Watched                   bool                    `json:"watched"`
	IsDisabled                bool                    `json:"is_disabled"`
	New                       bool                    `json:"new"`
	Status                    string                  `json:"status"`
	CancelOnly                bool                    `json:"cancel_only"`
	LimitOnly                 bool                    `json:"limit_only"`
	PostOnly                  bool                    `json:"post_only"`
	TradingDisabled           bool                    `json:"trading_disabled"`
	AuctionMode               bool                    `json:"auction_mode"`
	ProductType               string                  `json:"product_type"`
	QuoteCurrencyID           string                  `json:"quote_currency_id"`
	BaseCurrencyID            string                  `json:"base_currency_id"`
	FCMTradingSessionDetails  struct {
		IsSessionOpen bool      `json:"is_session_open"`
		OpenTime      time.Time `json:"open_time"`
		CloseTime     time.Time `json:"close_time"`
	} `json:"fcm_trading_session_details"`
	MidMarketPrice       convert.StringToFloat64 `json:"mid_market_price"`
	Alias                string                  `json:"alias"`
	AliasTo              []string                `json:"alias_to"`
	BaseDisplaySymbol    string                  `json:"base_display_symbol"`
	QuoteDisplaySymbol   string                  `json:"quote_display_symbol"`
	ViewOnly             bool                    `json:"view_only"`
	PriceIncrement       convert.StringToFloat64 `json:"price_increment"`
	FutureProductDetails struct {
		Venue                  string                  `json:"venue"`
		ContractCode           string                  `json:"contract_code"`
		ContractExpiry         string                  `json:"contract_expiry"`
		ContractSize           convert.StringToFloat64 `json:"contract_size"`
		ContractRootUnit       string                  `json:"contract_root_unit"`
		GroupDescription       string                  `json:"group_description"`
		ContractExpiryTimezone string                  `json:"contract_expiry_timezone"`
		GroupShortDescription  string                  `json:"group_short_description"`
		RiskManagedBy          string                  `json:"risk_managed_by"`
		ContractExpiryType     string                  `json:"contract_expiry_type"`
		PerpetualDetails       struct {
			OpenInterest convert.StringToFloat64 `json:"open_interest"`
			FundingRate  convert.StringToFloat64 `json:"funding_rate"`
			FundingTime  time.Time               `json:"funding_time"`
		} `json:"perpetual_details"`
		ContractDisplayName string `json:"contract_display_name"`
	} `json:"future_product_details"`
}

// AllProducts holds information on a lot of available currency pairs, returned by
// GetAllProducts
type AllProducts struct {
	Products    []Product `json:"products"`
	NumProducts int32     `json:"num_products"`
}

// UnixTimestamp is a type used to unmarshal unix timestamps returned from
// the exchange
type UnixTimestamp time.Time

// History holds historic rate information, returned by GetHistoricRates
type History struct {
	Candles []struct {
		Start  UnixTimestamp `json:"start"`
		Low    float64       `json:"low,string"`
		High   float64       `json:"high,string"`
		Open   float64       `json:"open,string"`
		Close  float64       `json:"close,string"`
		Volume float64       `json:"volume,string"`
	} `json:"candles"`
}

// Ticker holds basic ticker information, returned by GetTicker
type Ticker struct {
	Trades []struct {
		TradeID   string                  `json:"trade_id"`
		ProductID string                  `json:"product_id"`
		Price     float64                 `json:"price,string"`
		Size      float64                 `json:"size,string"`
		Time      time.Time               `json:"time"`
		Side      string                  `json:"side"`
		Bid       convert.StringToFloat64 `json:"bid"`
		Ask       convert.StringToFloat64 `json:"ask"`
	} `json:"trades"`
	BestBid convert.StringToFloat64 `json:"best_bid"`
	BestAsk convert.StringToFloat64 `json:"best_ask"`
}

// MarketMarketIOC is a sub-struct used in the type OrderConfiguration
type MarketMarketIOC struct {
	QuoteSize string `json:"quote_size,omitempty"`
	BaseSize  string `json:"base_size,omitempty"`
}

// LimitLimitGTC is a sub-struct used in the type OrderConfiguration
type LimitLimitGTC struct {
	BaseSize   string `json:"base_size"`
	LimitPrice string `json:"limit_price"`
	PostOnly   bool   `json:"post_only"`
}

// LimitLimitGTD is a sub-struct used in the type OrderConfiguration
type LimitLimitGTD struct {
	BaseSize   string    `json:"base_size"`
	LimitPrice string    `json:"limit_price"`
	EndTime    time.Time `json:"end_time"`
	PostOnly   bool      `json:"post_only"`
}

// StopLimitStopLimitGTC is a sub-struct used in the type OrderConfiguration
type StopLimitStopLimitGTC struct {
	BaseSize      string `json:"base_size"`
	LimitPrice    string `json:"limit_price"`
	StopPrice     string `json:"stop_price"`
	StopDirection string `json:"stop_direction"`
}

// StopLimitStopLimitGTD is a sub-struct used in the type OrderConfiguration
type StopLimitStopLimitGTD struct {
	BaseSize      string    `json:"base_size"`
	LimitPrice    string    `json:"limit_price"`
	StopPrice     string    `json:"stop_price"`
	EndTime       time.Time `json:"end_time"`
	StopDirection string    `json:"stop_direction"`
}

// OrderConfiguration is a struct used in the formation of requests for PlaceOrder
type OrderConfiguration struct {
	MarketMarketIOC       *MarketMarketIOC       `json:"market_market_ioc,omitempty"`
	LimitLimitGTC         *LimitLimitGTC         `json:"limit_limit_gtc,omitempty"`
	LimitLimitGTD         *LimitLimitGTD         `json:"limit_limit_gtd,omitempty"`
	StopLimitStopLimitGTC *StopLimitStopLimitGTC `json:"stop_limit_stop_limit_gtc,omitempty"`
	StopLimitStopLimitGTD *StopLimitStopLimitGTD `json:"stop_limit_stop_limit_gtd,omitempty"`
}

// PlaceOrderResp contains information on an order, returned by PlaceOrder
type PlaceOrderResp struct {
	Success         bool   `json:"success"`
	FailureReason   string `json:"failure_reason"`
	OrderID         string `json:"order_id"`
	SuccessResponse struct {
		OrderID       string `json:"order_id"`
		ProductID     string `json:"product_id"`
		Side          string `json:"side"`
		ClientOrderID string `json:"client_oid"`
	} `json:"success_response"`
	OrderConfiguration OrderConfiguration `json:"order_configuration"`
}

// CancelOrderResp contains information on attempted order cancellations, returned by
// CancelOrders
type CancelOrderResp struct {
	Results []struct {
		Success       bool   `json:"success"`
		FailureReason string `json:"failure_reason"`
		OrderID       string `json:"order_id"`
	} `json:"results"`
}

// EditOrderPreviewResp contains information on the effects of editing an order,
// returned by EditOrderPreview
type EditOrderPreviewResp struct {
	Slippage           float64 `json:"slippage,string"`
	OrderTotal         float64 `json:"order_total,string"`
	CommissionTotal    float64 `json:"commission_total,string"`
	QuoteSize          float64 `json:"quote_size,string"`
	BaseSize           float64 `json:"base_size,string"`
	BestBid            float64 `json:"best_bid,string"`
	BestAsk            float64 `json:"best_ask,string"`
	AverageFilledPrice float64 `json:"average_filled_price,string"`
}

// GetOrderResponse contains information on an order, returned by GetOrderByID
// and used in GetAllOrdersResp
type GetOrderResponse struct {
	OrderID               string             `json:"order_id"`
	ProductID             string             `json:"product_id"`
	UserID                string             `json:"user_id"`
	OrderConfiguration    OrderConfiguration `json:"order_configuration"`
	Side                  string             `json:"side"`
	ClientOID             string             `json:"client_order_id"`
	Status                string             `json:"status"`
	TimeInForce           string             `json:"time_in_force"`
	CreatedTime           time.Time          `json:"created_time"`
	CompletionPercentage  float64            `json:"completion_percentage,string"`
	FilledSize            float64            `json:"filled_size,string"`
	AverageFilledPrice    float64            `json:"average_filled_price,string"`
	Fee                   float64            `json:"fee,string"`
	NumberOfFills         int64              `json:"num_fills,string"`
	FilledValue           float64            `json:"filled_value,string"`
	PendingCancel         bool               `json:"pending_cancel"`
	SizeInQuote           bool               `json:"size_in_quote"`
	TotalFees             float64            `json:"total_fees,string"`
	SizeInclusiveOfFees   bool               `json:"size_inclusive_of_fees"`
	TotalValueAfterFees   float64            `json:"total_value_after_fees,string"`
	TriggerStatus         string             `json:"trigger_status"`
	OrderType             string             `json:"order_type"`
	RejectReason          string             `json:"reject_reason"`
	Settled               bool               `json:"settled"`
	ProductType           string             `json:"product_type"`
	RejectMessage         string             `json:"reject_message"`
	CancelMessage         string             `json:"cancel_message"`
	OrderPlacementSource  string             `json:"order_placement_source"`
	OutstandingHoldAmount float64            `json:"outstanding_hold_amount,string"`
	IsLiquidation         bool               `json:"is_liquidation"`
	LastFillTime          time.Time          `json:"last_fill_time"`
	EditHistory           []struct {
		Price                  float64   `json:"price,string"`
		Size                   float64   `json:"size,string"`
		ReplaceAcceptTimestamp time.Time `json:"replace_accept_timestamp"`
	} `json:"edit_history"`
}

// FillResponse contains fill information, returned by GetFills
type FillResponse struct {
	Fills []struct {
		EntryID            string    `json:"entry_id"`
		TradeID            string    `json:"trade_id"`
		OrderID            string    `json:"order_id"`
		TradeTime          time.Time `json:"trade_time"`
		TradeType          string    `json:"trade_type"`
		Price              float64   `json:"price,string"`
		Size               float64   `json:"size,string"`
		Commission         float64   `json:"commission,string"`
		ProductID          string    `json:"product_id"`
		SequenceTimestamp  time.Time `json:"sequence_timestamp"`
		LiquidityIndicator string    `json:"liquidity_indicator"`
		SizeInQuote        bool      `json:"size_in_quote"`
		UserID             string    `json:"user_id"`
		Side               string    `json:"side"`
	} `json:"fills"`
	Cursor string `json:"cursor"`
}

// TransactionSummary contains a summary of transaction fees, volume, and the like. Returned
// by GetTransactionSummary
type TransactionSummary struct {
	TotalVolume float64 `json:"total_volume"`
	TotalFees   float64 `json:"total_fees"`
	FeeTier     struct {
		PricingTier  float64 `json:"pricing_tier,string"`
		USDFrom      float64 `json:"usd_from,string"`
		USDTo        float64 `json:"usd_to,string"`
		TakerFeeRate float64 `json:"taker_fee_rate,string"`
		MakerFeeRate float64 `json:"maker_fee_rate,string"`
	}
	MarginRate struct {
		Value float64 `json:"value,string"`
	}
	GoodsAndServicesTax struct {
		Rate float64 `json:"rate,string"`
		Type string  `json:"type"`
	}
	AdvancedTradeOnlyVolume float64 `json:"advanced_trade_only_volume"`
	AdvancedTradeOnlyFees   float64 `json:"advanced_trade_only_fees"`
	CoinbaseProVolume       float64 `json:"coinbase_pro_volume"`
	CoinbaseProFees         float64 `json:"coinbase_pro_fees"`
}

// GetAllOrdersResp contains information on a lot of orders, returned by GetAllOrders
type GetAllOrdersResp struct {
	Orders   []GetOrderResponse `json:"orders"`
	Sequence int64              `json:"sequence,string"`
	HasNext  bool               `json:"has_next"`
	Cursor   string             `json:"cursor"`
}

// FeeStruct is a sub-struct storing information on fees, used in ConvertResponse
type FeeStruct struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Amount      ValCur `json:"amount"`
	Label       string `json:"label"`
	Disclosure  struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Link        struct {
			Text string `json:"text"`
			URL  string `json:"url"`
		} `json:"link"`
	} `json:"disclosure"`
}

// AccountStruct is a sub-struct storing information on accounts, used in ConvertResponse
type AccountStruct struct {
	Type          string `json:"type"`
	Network       string `json:"network"`
	LedgerAccount struct {
		AccountID string `json:"account_id"`
		Currency  string `json:"currency"`
		Owner     struct {
			ID       string `json:"id"`
			UUID     string `json:"uuid"`
			UserUUID string `json:"user_uuid"`
			Type     string `json:"type"`
		} `json:"owner"`
	} `json:"ledger_account"`
}

// AmScale is a sub-struct storing information on amounts and scales, used in ConvertResponse
type AmScale struct {
	Amount ValCur `json:"amount"`
	Scale  int32  `json:"scale"`
}

// ConvertResponse contains information on a convert trade, returned by CreateConvertQuote,
// CommitConvertTrade, and GetConvertTrade
type ConvertResponse struct {
	Trade struct {
		ID                string        `json:"id"`
		Status            string        `json:"status"`
		UserEnteredAmount ValCur        `json:"user_entered_amount"`
		Amount            ValCur        `json:"amount"`
		Subtotal          ValCur        `json:"subtotal"`
		Total             ValCur        `json:"total"`
		Fees              []FeeStruct   `json:"fees"`
		TotalFee          FeeStruct     `json:"total_fee"`
		Source            AccountStruct `json:"source"`
		Target            AccountStruct `json:"target"`
		UnitPrice         struct {
			TargetToFiat   AmScale `json:"target_to_fiat"`
			TargetToSource AmScale `json:"target_to_source"`
			SourceToFiat   AmScale `json:"source_to_fiat"`
		} `json:"unit_price"`
		UserWarnings []struct {
			ID   string `json:"id"`
			Link struct {
				Text string `json:"text"`
				URL  string `json:"url"`
			} `json:"link"`
			Context struct {
				Details  []string `json:"details"`
				Title    string   `json:"title"`
				LinkText string   `json:"link_text"`
			} `json:"context"`
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"user_warnings"`
		UserReference      string `json:"user_reference"`
		SourceCurrency     string `json:"source_currency"`
		TargetCurrency     string `json:"target_currency"`
		CancellationReason struct {
			Message   string `json:"message"`
			Code      string `json:"code"`
			ErrorCode string `json:"error_code"`
			ErrorCTA  string `json:"error_cta"`
		} `json:"cancellation_reason"`
		SourceID     string `json:"source_id"`
		TargetID     string `json:"target_id"`
		ExchangeRate ValCur `json:"exchange_rate"`
		TaxDetails   []struct {
			Name   string `json:"name"`
			Amount ValCur `json:"amount"`
		} `json:"tax_details"`
		TradeIncentiveInfo struct {
			AppliedIncentive    bool      `json:"applied_incentive"`
			UserIncentiveID     string    `json:"user_incentive_id"`
			CodeVal             string    `json:"code_val"`
			EndsAt              time.Time `json:"ends_at"`
			FeeWithoutIncentive ValCur    `json:"fee_without_incentive"`
			Redeemed            bool      `json:"redeemed"`
		} `json:"trade_incentive_info"`
		TotalFeeWithoutTax FeeStruct `json:"total_fee_without_tax"`
		FiatDenotedTotal   ValCur    `json:"fiat_denoted_total"`
	} `json:"trade"`
}

type ServerTimeV3 struct {
	Iso               time.Time `json:"iso"`
	EpochSeconds      int64     `json:"epochSeconds,string"`
	EpochMilliseconds int64     `json:"epochMillis,string"`
}

// IDResource holds an ID, resource type, and associated data, used in ListNotificationsResponse,
// TransactionData
type IDResource struct {
	ID           string `json:"id"`
	Resource     string `json:"resource"`
	ResourcePath string `json:"resource_path"`
	Email        string `json:"email"`
}

// PaginationResp holds pagination information, used in ListNotificationsResponse,
// GetAllWalletsResponse,
type PaginationResp struct {
	EndingBefore         string `json:"ending_before"`
	StartingAfter        string `json:"starting_after"`
	PreviousEndingBefore string `json:"previous_ending_before"`
	NextStartingAfter    string `json:"next_starting_after"`
	Limit                uint8  `json:"limit"`
	Order                string `json:"order"`
	PreviousURI          string `json:"previous_uri"`
	NextURI              string `json:"next_uri"`
}

// PaginationInp holds information needed to engage in pagination with Sign in With
// Coinbase. Used in ListNotifications, GetAllWallets, GetAllAddresses,
type PaginationInp struct {
	Limit         uint8
	OrderAscend   bool
	StartingAfter string
	EndingBefore  string
}

// ListNotificationsResponse holds information on notifications that the user is subscribed
// to. Returned by ListNotifications
type ListNotificationsResponse struct {
	Pagination PaginationResp `json:"pagination"`
	Data       []struct {
		ID   string `json:"id"`
		Type string `json:"type"`
		Data struct {
			ID            string     `json:"id"`
			Address       string     `json:"address"`
			Name          string     `json:"name"`
			Status        string     `json:"status"`
			PaymentMethod IDResource `json:"payment_method"`
			Transaction   IDResource `json:"transaction"`
			Amount        AmCur      `json:"amount"`
			Total         AmCur      `json:"total"`
			Subtotal      AmCur      `json:"subtotal"`
			CreatedAt     time.Time  `json:"created_at"`
			UpdatedAt     time.Time  `json:"updated_at"`
			Resource      string     `json:"resource"`
			ResourcePath  string     `json:"resource_path"`
			Committed     bool       `json:"committed"`
			Instant       bool       `json:"instant"`
			Fee           AmCur      `json:"fee"`
			Fees          []struct {
				Type   string `json:"type"`
				Amount AmCur  `json:"amount"`
			} `json:"fees"`
			PayoutAt time.Time `json:"payout_at"`
		} `json:"data"`
		AdditionalData struct {
			Hash   string `json:"hash"`
			Amount AmCur  `json:"amount"`
		} `json:"additional_data"`
		User             IDResource `json:"user"`
		Account          IDResource `json:"account"`
		DeliveryAttempts int32      `json:"delivery_attempts"`
		CreatedAt        time.Time  `json:"created_at"`
		Resource         string     `json:"resource"`
		ResourcePath     string     `json:"resource_path"`
		Transaction      IDResource `json:"transaction"`
	} `json:"data"`
}

// UserResponse holds information on a user, returned by GetUseByID and GetCurrentUser
type UserResponse struct {
	Data struct {
		ID              string `json:"id"`
		Name            string `json:"name"`
		Username        string `json:"username"`
		ProfileLocation string `json:"profile_location"`
		ProfileBio      string `json:"profile_bio"`
		ProfileURL      string `json:"profile_url"`
		AvatarURL       string `json:"avatar_url"`
		Resource        string `json:"resource"`
		ResourcePath    string `json:"resource_path"`
		LegacyID        string `json:"legacy_id"`
		TimeZone        string `json:"time_zone"`
		NativeCurrency  string `json:"native_currency"`
		BitcoinUnit     string `json:"bitcoin_unit"`
		State           string `json:"state"`
		Country         struct {
			Code       string `json:"code"`
			Name       string `json:"name"`
			IsInEurope bool   `json:"is_in_europe"`
		} `json:"country"`
		Nationality struct {
			Code string `json:"code"`
			Name string `json:"name"`
		} `json:"nationality"`
		RegionSupportsFiatTransfers           bool      `json:"region_supports_fiat_transfers"`
		RegionSupportsCryptoToCryptoTransfers bool      `json:"region_supports_crypto_to_crypto_transfers"`
		CreatedAt                             time.Time `json:"created_at"`
		SupportsRewards                       bool      `json:"supports_rewards"`
		Tiers                                 struct {
			CompletedDescription string `json:"completed_description"`
			UpgradeButtonText    string `json:"upgrade_button_text"`
			Header               string `json:"header"`
			Body                 string `json:"body"`
		} `json:"tiers"`
		ReferralMoney struct {
			Amount            float64 `json:"amount,string"`
			Currency          string  `json:"currency"`
			CurrencySymbol    string  `json:"currency_symbol"`
			ReferralThreshold float64 `json:"referral_threshold,string"`
		} `json:"referral_money"`
		HasBlockingBuyRestrictions            bool   `json:"has_blocking_buy_restrictions"`
		HasMadeAPurchase                      bool   `json:"has_made_a_purchase"`
		HasBuyDepositPaymentMethods           bool   `json:"has_buy_deposit_payment_methods"`
		HasUnverifiedBuyDepositPaymentMethods bool   `json:"has_unverified_buy_deposit_payment_methods"`
		NeedsKYCRemediation                   bool   `json:"needs_kyc_remediation"`
		ShowInstantAchUx                      bool   `json:"show_instant_ach_ux"`
		UserType                              string `json:"user_type"`
		Email                                 string `json:"email"`
		SendsDisabled                         bool   `json:"sends_disabled"`
	} `json:"data"`
}

// AuthResponse holds authentication information, returned by GetAuthInfo
type AuthResponse struct {
	Data struct {
		Method    string   `json:"method"`
		Scopes    []string `json:"scopes"`
		OAuthMeta interface{}
	} `json:"data"`
}

// WalletData holds wallet information, returned by in GenWalletResponse and GetAllWalletsResponse
type WalletData struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Primary  bool   `json:"primary"`
	Type     string `json:"type"`
	Currency struct {
		Code                string `json:"code"`
		Name                string `json:"name"`
		Color               string `json:"color"`
		SortIndex           int32  `json:"sort_index"`
		Exponent            int32  `json:"exponent"`
		Type                string `json:"type"`
		AddressRegex        string `json:"address_regex"`
		AssetID             string `json:"asset_id"`
		DestinationTagName  string `json:"destination_tag_name"`
		DestinationTagRegex string `json:"destination_tag_regex"`
		Slug                string `json:"slug"`
	} `json:"currency"`
	Balance struct {
		Amount   float64 `json:"amount,string"`
		Currency string  `json:"currency"`
	} `json:"balance"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	Resource         string    `json:"resource"`
	ResourcePath     string    `json:"resource_path"`
	AllowDeposits    bool      `json:"allow_deposits"`
	AllowWithdrawals bool      `json:"allow_withdrawals"`
}

// GenWalletResponse holds information on a single wallet, returned by CreateWallet,
// GetWalletByID, and UpdateWalletName
type GenWalletResponse struct {
	Data WalletData `json:"data"`
}

// GetAllWalletsResponse holds information on many wallets, returned by GetAllWallets
type GetAllWalletsResponse struct {
	Pagination PaginationResp `json:"pagination"`
	Data       []WalletData   `json:"data"`
}

// AddressData holds address information, used in GenAddrResponse and GetAllAddrResponse
type AddressData struct {
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
		Type     string `json:"type"`
		Title    string `json:"title"`
		Details  string `json:"details"`
		ImageURL string `json:"image_url"`
		Options  []struct {
			Text  string `json:"text"`
			Style string `json:"style"`
			ID    string `json:"id"`
		} `json:"options"`
	} `json:"warnings"`
	QRCodeImageURL   string `json:"qr_code_image_url"`
	AddressLabel     string `json:"address_label"`
	DefaultReceive   bool   `json:"default_receive"`
	DestinationTag   string `json:"destination_tag"`
	DepositURI       string `json:"deposit_uri"`
	CallbackURL      string `json:"callback_url"`
	ShareAddressCopy struct {
		Line1 string `json:"line1"`
		Line2 string `json:"line2"`
	} `json:"share_address_copy"`
	ReceiveSubtitle string `json:"receive_subtitle"`
	InlineWarning   struct {
		Text    string `json:"text"`
		Tooltip struct {
			Title    string `json:"title"`
			Subtitle string `json:"subtitle"`
		} `json:"tooltip"`
	} `json:"inline_warning"`
}

// GenAddrResponse holds information on a generated address, returned by CreateAddress and
// GetAddressByID. Used in GetAllAddrResponse
type GenAddrResponse struct {
	Data AddressData `json:"data"`
}

// GetAllAddrResponse holds information on many addresses, returned by GetAllAddresses
type GetAllAddrResponse struct {
	Pagination PaginationResp `json:"pagination"`
	Data       []AddressData  `json:"data"`
}

// AmCur is a sub-type used in TransactionData, LimitStruct
type AmCur struct {
	Amount   float64 `json:"amount,string"`
	Currency string  `json:"currency"`
}

// TransactionData is a sub-type that holds information on a transaction. Used in
// ManyTransactionsResp
type TransactionData struct {
	ID                string     `json:"id"`
	Type              string     `json:"type"`
	Status            string     `json:"status"`
	Amount            AmCur      `json:"amount"`
	NativeAmount      AmCur      `json:"native_amount"`
	Description       string     `json:"description"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	Resource          string     `json:"resource"`
	ResourcePath      string     `json:"resource_path"`
	InstantExchange   bool       `json:"instant_exchange"`
	Buy               IDResource `json:"buy"`
	AdvancedTradeFill struct {
		FillPrice  float64 `json:"fill_price,string"`
		ProductID  string  `json:"product_id"`
		OrderID    string  `json:"order_id"`
		Commission float64 `json:"commission,string"`
		OrderSide  string  `json:"order_side"`
	} `json:"advanced_trade_fill"`
	Details struct {
		Title    string `json:"title"`
		Subtitle string `json:"subtitle"`
	} `json:"details"`
	Network struct {
		Status string `json:"status"`
		Hash   string `json:"hash"`
		Name   string `json:"name"`
	} `json:"network"`
	To      IDResource `json:"to"`
	From    IDResource `json:"from"`
	Address struct {
	} `json:"address"`
	Application struct {
	} `json:"application"`
}

// ManyTransactionsResp holds information on many transactions. Returned by
// GetAddressTransactions, ListTransactions
type ManyTransactionsResp struct {
	Pagination PaginationResp    `json:"pagination"`
	Data       []TransactionData `json:"data"`
}

// GenTransactionResp holds information on one transaction. Returned by SendMoney,
type GenTransactionResp struct {
	Data TransactionData `json:"data"`
}

// DeposWithdrData is a sub-type that holds information on a deposit/withdrawal. Used in
// GenDeposWithdrResp and ManyDeposWithdrResp
type DeposWithdrData struct {
	ID            string     `json:"id"`
	Status        string     `json:"status"`
	PaymentMethod IDResource `json:"payment_method"`
	Transaction   IDResource `json:"transaction"`
	Amount        AmCur      `json:"amount"`
	Subtotal      AmCur      `json:"subtotal"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	Resource      string     `json:"resource"`
	ResourcePath  string     `json:"resource_path"`
	Committed     bool       `json:"committed"`
	Fee           AmCur      `json:"fee"`
	PayoutAt      time.Time  `json:"payout_at"`
	TransferType  FiatTransferType
}

// GenDeposWithdrResp holds information on a deposit. Returned by DepositFunds, CommitDeposit,
// and GetDepositByID
type GenDeposWithdrResp struct {
	Data DeposWithdrData `json:"data"`
}

// ManyDeposWithdrResp holds information on many deposits. Returned by GetAllDeposits
type ManyDeposWithdrResp struct {
	Pagination PaginationResp    `json:"pagination"`
	Data       []DeposWithdrData `json:"data"`
}

// PaymentMethodData is a sub-type that holds information on a payment method. Used in
// GenPaymentMethodResp and GetAllPaymentMethodsResp
type PaymentMethodData struct {
	ID            string    `json:"id"`
	Type          string    `json:"type"`
	Name          string    `json:"name"`
	Currency      string    `json:"currency"`
	PrimaryBuy    bool      `json:"primary_buy"`
	PrimarySell   bool      `json:"primary_sell"`
	AllowBuy      bool      `json:"allow_buy"`
	AllowSell     bool      `json:"allow_sell"`
	AllowDeposit  bool      `json:"allow_deposit"`
	AllowWithdraw bool      `json:"allow_withdraw"`
	InstantBuy    bool      `json:"instant_buy"`
	InstantSell   bool      `json:"instant_sell"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	Resource      string    `json:"resource"`
	ResourcePath  string    `json:"resource_path"`
	Limits        struct {
		Type string `json:"type"`
		Name string `json:"name"`
	} `json:"limits"`
	FiatAccount           IDResource `json:"fiat_account"`
	Verified              bool       `json:"verified"`
	MinimumPurchaseAmount AmCur      `json:"minimum_purchase_amount"`
}

// GetAllPaymentMethodsResp holds information on many payment methods. Returned by
// GetAllPaymentMethods
type GetAllPaymentMethodsResp struct {
	Pagination PaginationResp      `json:"pagination"`
	Data       []PaymentMethodData `json:"data"`
}

// GenPaymentMethodResp holds information on a payment method. Returned by
// GetPaymentMethodByID
type GenPaymentMethodResp struct {
	Data PaymentMethodData `json:"data"`
}

// GetFiatCurrenciesResp holds information on fiat currencies. Returned by
// GetFiatCurrencies
type GetFiatCurrenciesResp struct {
	Data []struct {
		ID      string  `json:"id"`
		Name    string  `json:"name"`
		MinSize float64 `json:"min_size,string"`
	}
}

// GetCryptocurrenciesResp holds information on cryptocurrencies. Returned by
// GetCryptocurrencies
type GetCryptocurrenciesResp struct {
	Data []struct {
		Code         string `json:"code"`
		Name         string `json:"name"`
		Color        string `json:"color"`
		SortIndex    uint16 `json:"sort_index"`
		Exponent     uint8  `json:"exponent"`
		Type         string `json:"type"`
		AddressRegex string `json:"address_regex"`
		AssetID      string `json:"asset_id"`
	}
}

// GetExchangeRatesResp holds information on exchange rates. Returned by GetExchangeRates
type GetExchangeRatesResp struct {
	Data struct {
		Currency string                             `json:"currency"`
		Rates    map[string]convert.StringToFloat64 `json:"rates"`
	} `json:"data"`
}

// GetPriceResp holds information on a price. Returned by GetPrice
type GetPriceResp struct {
	Data struct {
		Amount   float64 `json:"amount,string"`
		Base     string  `json:"base"`
		Currency string  `json:"currency"`
	} `json:"data"`
}

// ServerTimeV2 holds current requested server time information, returned from V2 of the API
type ServerTimeV2 struct {
	Data struct {
		ISO   time.Time `json:"iso"`
		Epoch uint64    `json:"epoch"`
	} `json:"data"`
}

// Trade holds executed trade information, returned by GetTrades
type Trade struct {
	TradeID int64     `json:"trade_id"`
	Price   float64   `json:"price,string"`
	Size    float64   `json:"size,string"`
	Time    time.Time `json:"time"`
	Side    string    `json:"side"`
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
		Symbol                string   `json:"symbol"`
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
	} `json:"supported_networks"`
}

// AccountLedgerResponse holds account history information, returned by GetAccountLedger
type AccountLedgerResponse struct {
	ID        string    `json:"id"`
	Amount    float64   `json:"amount,string"`
	CreatedAt time.Time `json:"created_at"`
	Balance   float64   `json:"balance,string"`
	Type      string    `json:"type"`
	Details   struct {
		OrderID      string `json:"order_id"`
		ProductID    string `json:"product_id"`
		TradeID      string `json:"trade_id"`
		TransferID   string `json:"transfer_id"`
		TransferType string `json:"transfer_type"`
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

// LimitStruct is a sub-type used in PaymentMethod
type LimitStruct struct {
	PeriodInDays int   `json:"period_in_days"`
	Total        AmCur `json:"total"`
	Remaining    AmCur `json:"remaining"`
}

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
		Type       string        `json:"type"`
		Name       string        `json:"name"`
		Buy        []LimitStruct `json:"buy"`
		InstantBuy []LimitStruct `json:"instant_buy"`
		Sell       []LimitStruct `json:"sell"`
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
	VerificationMethod string `json:"verification_method"`
	CDVStatus          string `json:"cdv_status"`
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
		Iban            string `json:"iban"`
		Swift           string `json:"swift"`
		BankName        string `json:"bank_name"`
		BankAddress     string `json:"bank_address"`
		BankCountryName string `json:"bank_country_name"`
		AccountName     string `json:"account_name"`
		AccountAddress  string `json:"account_address"`
		Reference       string `json:"reference"`
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
			DefaultProfileID        string        `json:"default_profile_id"`
			OauthClient             string        `json:"oauth_client"`
			Preferences             struct {
				PreferredMarket              string    `json:"preferred_market"`
				MarginTermsCompletedInUTC    time.Time `json:"margin_terms_completed_in_utc"`
				MarginTutorialCompletedInUTC time.Time `json:"margin_tutorial_completed_in_utc"`
			} `json:"preferences"`
			HasDefault                bool        `json:"has_default"`
			OrgID                     interface{} `json:"org_id"`
			IsBrokerage               bool        `json:"is_brokerage"`
			TaxDomain                 string      `json:"tax_domain"`
			ProfileLimit              uint16      `json:"profile_limit"`
			APIKeyLimit               uint16      `json:"api_key_limit"`
			ConnectionLimit           uint16      `json:"connection_limit"`
			RateLimit                 uint16      `json:"rate_limit"`
			GlobalConnectionLimit     uint16      `json:"global_connection_limit"`
			SettlementPreference      interface{} `json:"settlement_preference"`
			PrimeLendingEntityID      interface{} `json:"prime_lending_entity_id"`
			StateCode                 string      `json:"state_code"`
			CBDataFromCache           bool        `json:"cb_data_from_cache"`
			TwoFactorMethod           string      `json:"two_factor_method"`
			LegalName                 string      `json:"legal_name"`
			TermsAccepted             time.Time   `json:"terms_accepted"`
			HasClawbackPaymentPending bool        `json:"has_clawback_payment_pending"`
			HasRestrictedAssets       bool        `json:"has_restricted_assets"`
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
	ID          string   `json:"id"`
	Type        string   `json:"type"`
	CreatedAt   ExchTime `json:"created_at"`
	CompletedAt ExchTime `json:"completed_at"`
	CanceledAt  ExchTime `json:"canceled_at"`
	ProcessedAt ExchTime `json:"processed_at"`
	AccountID   string   `json:"account_id"`
	UserID      string   `json:"user_id"`
	Amount      float64  `json:"amount,string"`
	Details     struct {
		CoinbasePayoutAt          time.Time `json:"coinbase_payout_at"`
		CoinbaseAccountID         string    `json:"coinbase_account_id"`
		CoinbaseTransactionID     string    `json:"coinbase_transaction_id"`
		CoinbaseDepositID         string    `json:"coinbase_deposit_id"`
		CoinbasePaymentMethodID   string    `json:"coinbase_payment_method_id"`
		CoinbasePaymentMethodType string    `json:"coinbase_payment_method_type"`
	} `json:"details"`
	UserNonce int64  `json:"user_nonce"`
	ProfileID string `json:"profile_id"`
	Currency  string `json:"currency"`
	Idem      string `json:"idem"`
}

type ExchTime time.Time

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
	Name                   string    `json:"name"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
	Network                string    `json:"network"`
	URIScheme              string    `json:"uri_scheme"`
	Resource               string    `json:"resource"`
	ResourcePath           string    `json:"resource_path"`
	ExchangeDepositAddress bool      `json:"exchange_deposit_address"`
	Warnings               []struct {
		Title    string `json:"title"`
		Details  string `json:"details"`
		ImageURL string `json:"image_url"`
	} `json:"warnings"`
	LegacyAddress  string `json:"legacy_address"`
	DestinationTag string `json:"destination_tag"`
	DepositURI     string `json:"deposit_uri"`
	CallbackURL    string `json:"callback_url"`
}

// WithdrawalFeeEstimate is the exchange's estimate of the fee for a withdrawal, returned
// by GetWithdrawalFeeEstimate
type WithdrawalFeeEstimate struct {
	Fee              float64 `json:"fee"`
	FeeBeforeSubsidy float64 `json:"fee_before_subsidy"`
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
	DateTime string `json:"datetime"`
}

// ReportFillsTaxStruct is used internally when crafting a CreateReport request
type ReportFillsTaxStruct struct {
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	ProductID string `json:"product_id"`
}

// ReportAccountStruct is used internally when crafting a CreateReport request
type ReportAccountStruct struct {
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	AccountID string `json:"account_id"`
}

// MaxRemSubStruct is a sub-type used in CurListSubStruct, which is itself used in ExchangeLimits
type MaxRemSubStruct struct {
	Max       float64 `json:"max"`
	Remaining float64 `json:"remaining"`
}

// CurListSubStruct is a sub-type used in ExchangeLimits
type CurListSubStruct struct {
	USD MaxRemSubStruct `json:"usd"`
	EUR MaxRemSubStruct `json:"eur"`
	GBP MaxRemSubStruct `json:"gbp"`
	BTC MaxRemSubStruct `json:"btc"`
	ETH MaxRemSubStruct `json:"eth"`
}

// ExchangeLimits contains information on payment method transfer limits, returned
// by GetExchangeLimits
type ExchangeLimits struct {
	TransferLimits struct {
		Buy                  CurListSubStruct `json:"buy"`
		Sell                 CurListSubStruct `json:"sell"`
		ExchangeWithdraw     CurListSubStruct `json:"exchange_withdraw"`
		Ach                  CurListSubStruct `json:"ach"`
		InstantBuy           CurListSubStruct `json:"instant_buy"`
		AchNoBalance         CurListSubStruct `json:"ach_no_balance"`
		CreditDebitCard      CurListSubStruct `json:"credit_debit_card"`
		Secure3DBuy          CurListSubStruct `json:"secure3d_buy"`
		PaypalBuy            CurListSubStruct `json:"paypal_buy"`
		PaypalWithdrawal     CurListSubStruct `json:"paypal_withdrawal"`
		IdealDeposit         CurListSubStruct `json:"ideal_deposit"`
		SofortDeposit        CurListSubStruct `json:"sofort_deposit"`
		InstantAchWithdrawal CurListSubStruct `json:"instant_ach_withdrawal"`
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
	WrappedAssetResponse []WrappedAssetResponse `json:"wrapped_assets"`
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
