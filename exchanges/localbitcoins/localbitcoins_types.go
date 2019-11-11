package localbitcoins

import (
	"time"
)

// GeneralError is an error capture type
type GeneralError struct {
	Error struct {
		Message   string `json:"message"`
		ErrorCode int    `json:"error_code"`
	} `json:"error"`
}

// AccountInfo holds public user information
type AccountInfo struct {
	Username              string `json:"username"`
	FeedbackScore         int    `json:"feedback_score"`
	FeedbackCount         int    `json:"feedback_count"`
	RealNameVeriTrusted   int    `json:"real_name_verifications_trusted"`
	TradingPartners       int    `json:"trading_partners_count"`
	URL                   string `json:"url"`
	RealNameVeriUntrusted int    `json:"real_name_verifications_untrusted"`
	HasFeedback           bool   `json:"has_feedback"`
	IdentityVerifiedAt    string `json:"identify_verified_at"`
	TrustedCount          int    `json:"trusted_count"`
	FeedbacksUnconfirmed  int    `json:"feedbacks_unconfirmed_count"`
	BlockedCount          int    `json:"blocked_count"`
	TradeVolumeText       string `json:"trade_volume_text"`
	HasCommonTrades       bool   `json:"has_common_trades"`
	RealNameVeriRejected  int    `json:"real_name_verifications_rejected"`
	AgeText               string `json:"age_text"`
	ConfirmedTradesText   string `json:"confirmed_trade_count_text"`
	CreatedAt             string `json:"created_at"`
}

// AdData references the full possible return of ad data
type AdData struct {
	AdList []struct {
		Data struct {
			Visible                    bool        `json:"visible"`
			HiddenByOpeningHours       bool        `json:"hidden_by_opening_hours"`
			Location                   string      `json:"location_string"`
			CountryCode                string      `json:"countrycode"`
			City                       string      `json:"city"`
			TradeType                  string      `json:"trade_type"`
			OnlineProvider             string      `json:"online_provider"`
			FirstTimeLimitBTC          string      `json:"first_time_limit_btc"`
			VolumeCoefficientBTC       string      `json:"volume_coefficient_btc"`
			SMSVerficationRequired     bool        `json:"sms_verification_required"`
			ReferenceType              string      `json:"reference_type"`
			DisplayReference           bool        `json:"display_reference"`
			Currency                   string      `json:"currency"`
			Lat                        float64     `json:"lat"`
			Lon                        float64     `json:"lon"`
			MinAmount                  string      `json:"min_amount"`
			MaxAmount                  string      `json:"max_amount"`
			MaXAmountAvailable         string      `json:"max_amount_available"`
			LimitToFiatAmounts         string      `json:"limit_to_fiat_amounts"`
			AdID                       int64       `json:"ad_id"`
			TempPriceUSD               string      `json:"temp_price_usd"`
			Floating                   bool        `json:"floating"`
			Profile                    interface{} `json:"profile"`
			RequireFeedBackScore       int         `json:"require_feedback_score"`
			RequireTradeVolume         float64     `json:"require_trade_volume"`
			RequireTrustedByAdvertiser bool        `json:"require_trusted_by_advertiser"`
			PaymentWindowMinutes       int         `json:"payment_window_minutes"`
			BankName                   string      `json:"bank_name"`
			TrackMaxAmount             bool        `json:"track_max_amount"`
			ATMModel                   string      `json:"atm_model"`
			PriceEquation              string      `json:"price_equation"`
			OpeningHours               interface{} `json:"opening_hours"`
			AccountInfo                string      `json:"account_info"`
			AccountDetails             interface{} `json:"account_details"`
		} `json:"data"`
		Actions struct {
			PublicView  string `json:"public_view"`
			HTMLEdit    string `json:"html_edit"`
			ChangeForm  string `json:"change_form"`
			ContactForm string `json:"contact_form"`
		} `json:"actions"`
	} `json:"ad_list"`
	AdCount int `json:"ad_count"`
}

// AdEdit references an outgoing paramater type for EditAd() method
type AdEdit struct {
	// Required Arguments
	PriceEquation              string `json:"price_equation"`
	Latitude                   int    `json:"lat"`
	Longitude                  int    `json:"lon"`
	City                       string `json:"city"`
	Location                   string `json:"location_string"`
	CountryCode                string `json:"countrycode"`
	Currency                   string `json:"currency"`
	AccountInfo                string `json:"account_info"`
	BankName                   string `json:"bank_name"`
	MSG                        string `json:"msg"`
	SMSVerficationRequired     bool   `json:"sms_verification_required"`
	TrackMaxAmount             bool   `json:"track_max_amount"`
	RequireTrustedByAdvertiser bool   `json:"require_trusted_by_advertiser"`
	RequireIdentification      bool   `json:"require_identification"`

	// Optional Arguments
	MinAmount          int      `json:"min_amount"`
	MaxAmount          int      `json:"max_amount"`
	OpeningHours       []string `json:"opening_hours"`
	LimitToFiatAmounts string   `json:"limit_to_fiat_amounts"`
	Visible            bool     `json:"visible"`

	// Optional Arguments ONLINE_SELL ads
	RequireTradeVolume   int    `json:"require_trade_volume"`
	RequireFeedBackScore int    `json:"require_feedback_score"`
	FirstTimeLimitBTC    int    `json:"first_time_limit_btc"`
	VolumeCoefficientBTC int    `json:"volume_coefficient_btc"`
	ReferenceType        string `json:"reference_type"`
	DisplayReference     bool   `json:"display_reference"`

	// Optional Arguments ONLINE_BUY
	PaymentWindowMinutes int `json:"payment_window_minutes"`

	// Optional Arguments LOCAL_SELL
	Floating bool `json:"floating"`
}

// AdCreate references an outgoing paramater type for CreateAd() method
type AdCreate struct {
	// Required Arguments
	PriceEquation              string `json:"price_equation"`
	Latitude                   int    `json:"lat"`
	Longitude                  int    `json:"lon"`
	City                       string `json:"city"`
	Location                   string `json:"location_string"`
	CountryCode                string `json:"countrycode"`
	Currency                   string `json:"currency"`
	AccountInfo                string `json:"account_info"`
	BankName                   string `json:"bank_name"`
	MSG                        string `json:"msg"`
	SMSVerficationRequired     bool   `json:"sms_verification_required"`
	TrackMaxAmount             bool   `json:"track_max_amount"`
	RequireTrustedByAdvertiser bool   `json:"require_trusted_by_advertiser"`
	RequireIdentification      bool   `json:"require_identification"`
	OnlineProvider             string `json:"online_provider"`
	TradeType                  string `json:"trade_type"`

	// Optional Arguments
	MinAmount          int      `json:"min_amount"`
	MaxAmount          int      `json:"max_amount"`
	OpeningHours       []string `json:"opening_hours"`
	LimitToFiatAmounts string   `json:"limit_to_fiat_amounts"`
	Visible            bool     `json:"visible"`

	// Optional Arguments ONLINE_SELL ads
	RequireTradeVolume   int    `json:"require_trade_volume"`
	RequireFeedBackScore int    `json:"require_feedback_score"`
	FirstTimeLimitBTC    int    `json:"first_time_limit_btc"`
	VolumeCoefficientBTC int    `json:"volume_coefficient_btc"`
	ReferenceType        string `json:"reference_type"`
	DisplayReference     bool   `json:"display_reference"`

	// Optional Arguments ONLINE_BUY
	PaymentWindowMinutes int `json:"payment_window_minutes"`

	// Optional Arguments LOCAL_SELL
	Floating bool `json:"floating"`
}

// Message holds the returned message data from a contact
type Message struct {
	MSG    string `json:"msg"`
	Sender struct {
		ID         int64  `json:"id"`
		Name       string `json:"name"`
		Username   string `json:"username"`
		TradeCount int64  `json:"trafe_count"`
		LastOnline string `json:"last_online"`
	} `json:"sender"`
	CreatedAt      string `json:"created_at"`
	IsAdmin        bool   `json:"is_admin"`
	AttachmentName string `json:"attachment_name"`
	AttachmentType string `json:"attachment_type"`
	AttachmentURL  string `json:"attachment_url"`
}

// DashBoardInfo holds the full range of metadata for a dashboard image
type DashBoardInfo struct {
	Data struct {
		CreatedAt string `json:"created_at"`
		Buyer     struct {
			Username                 string `json:"username"`
			TradeCount               string `json:"trade_count"`
			FeedbackScore            int    `json:"feedback_score"`
			Name                     string `json:"name"`
			LastOnline               string `json:"last_online"`
			RealName                 string `json:"real_name"`
			CompanyName              string `json:"company_name"`
			CountryCodeByIP          string `json:"countrycode_by_ip"`
			CountryCodeByPhoneNUmber string `json:"countrycode_by_phone_number"`
		} `json:"buyer"`
		Seller struct {
			Username      string `json:"username"`
			TradeCount    string `json:"trade_count"`
			FeedbackScore int    `json:"feedback_score"`
			Name          string `json:"name"`
			LastOnline    string `json:"last_online"`
		} `json:"seller"`
		ReferenceCode         string  `json:"reference_code"`
		Currency              string  `json:"currency"`
		Amount                float64 `json:"amount,string"`
		AmountBTC             float64 `json:"amount_btc,string"`
		FeeBTC                float64 `json:"fee_btc,string"`
		ExchangeRateUpdatedAt string  `json:"exchange_rate_updated_at"`
		Advertisement         struct {
			ID         int    `json:"id"`
			TradeType  string `json:"trade_type"`
			Advertiser struct {
				Username      string `json:"username"`
				TradeCount    string `json:"trade_count"`
				FeedbackScore int    `json:"feedback_score"`
				Name          string `json:"name"`
				LastOnline    string `json:"last_online"`
			} `json:"advertiser"`
		} `json:"advertisement"`
		ContactID          int         `json:"contact_id"`
		CanceledAt         string      `json:"canceled_at"`
		EscrowedAt         string      `json:"escrowed_at"`
		FundedAt           string      `json:"funded_at"`
		PaymentCompletedAt string      `json:"payment_completed_at"`
		DisputedAt         string      `json:"disputed_at"`
		ClosedAt           string      `json:"closed_at"`
		ReleasedAt         string      `json:"released_at"`
		IsBuying           bool        `json:"is_buying"`
		IsSelling          bool        `json:"is_selling"`
		AccountDetails     interface{} `json:"account_details"`
		AccountInfo        string      `json:"account_info"`
		Floating           bool        `json:"floating"`
	} `json:"data"`
	Actions struct {
		MarkAsPaidURL           string `json:"mark_as_paid_url"`
		AdvertisementPublicView string `json:"advertisement_public_view"`
		MessageURL              string `json:"message_url"`
		MessagePostURL          string `json:"message_post_url"`
	} `json:"actions"`
}

// Invoice contains invoice data
type Invoice struct {
	Invoice struct {
		Description     string  `json:"description"`
		Created         string  `json:"created"`
		URL             string  `json:"url"`
		Amount          float64 `json:"amount,string"`
		Internal        bool    `json:"internal"`
		Currency        string  `json:"currency"`
		State           string  `json:"state"`
		ID              string  `json:"id"`
		BTCAmount       string  `json:"btc_amount"`
		BTCAddress      string  `json:"btc_address"`
		DeletingAllowed bool    `json:"deleting_allowed"`
	} `json:"invoice"`
}

// NotificationInfo holds Notification data
type NotificationInfo struct {
	URL       string `json:"url"`
	CreatedAt string `json:"created_at"`
	ContactID int64  `json:"contact_id"`
	Read      bool   `json:"read"`
	MSG       string `json:"msg"`
	ID        string `json:"id"`
}

// WalletInfo holds full wallet information data
type WalletInfo struct {
	Message                 string              `json:"message"`
	Total                   Balance             `json:"total"`
	SentTransactions30d     []WalletTransaction `json:"sent_transactions_30d"`
	ReceivedTransactions30d []WalletTransaction `json:"received_transactions_30d"`
	ReceivingAddressCount   int                 `json:"receiving_address_count"`
	ReceivingAddressList    []WalletAddressList `json:"receiving_address_list"`
}

// Balance is a sub-type for WalletInfo & WalletBalanceInfo
type Balance struct {
	Balance  float64 `json:"balance,string"`
	Sendable float64 `json:"Sendable,string"`
}

// WalletTransaction is a sub-type for WalletInfo
type WalletTransaction struct {
	TXID        string    `json:"txid"`
	Amount      float64   `json:"amount,string"`
	Description string    `json:"description"`
	TXType      int       `json:"tx_type"`
	CreatedAt   time.Time `json:"created_at"`
}

// WalletAddressList is a sub-type for WalletInfo & WalletBalanceInfo
type WalletAddressList struct {
	Address  string  `json:"address"`
	Received float64 `json:"received,string"`
}

// WalletBalanceInfo standard wallet balance information
type WalletBalanceInfo struct {
	Message               string              `json:"message"`
	Total                 Balance             `json:"total"`
	ReceivingAddressCount int                 `json:"receiving_address_count"` // always 1
	ReceivingAddressList  []WalletAddressList `json:"receiving_address_list"`
}

// Ticker contains ticker information
type Ticker struct {
	Avg12h float64 `json:"avg_12h,string"`
	Avg1h  float64 `json:"avg_1h,string,omitempty"`
	Avg6h  float64 `json:"avg_6h,string,omitempty"`
	Avg24h float64 `json:"avg_24h,string"`
	Rates  struct {
		Last float64 `json:"last,string"`
	} `json:"rates"`
	VolumeBTC float64 `json:"volume_btc,string"`
}

// Trade holds closed trade information
type Trade struct {
	TID    int64   `json:"tid"`
	Date   int64   `json:"date"`
	Amount float64 `json:"amount,string"`
	Price  float64 `json:"price,string"`
}

// Orderbook is a full range of bid and asks for localbitcoins
type Orderbook struct {
	Bids []Price `json:"bids"`
	Asks []Price `json:"asks"`
}

// Price is a sub-type for orderbook
type Price struct {
	Price  float64
	Amount float64
}
