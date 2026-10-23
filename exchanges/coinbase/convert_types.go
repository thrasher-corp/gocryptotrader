package coinbase

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Convert trade statuses
const (
	ConvertTradeStatusCreated   = "TRADE_STATUS_CREATED"
	ConvertTradeStatusStarted   = "TRADE_STATUS_STARTED"
	ConvertTradeStatusCompleted = "TRADE_STATUS_COMPLETED"
	ConvertTradeStatusCancelled = "TRADE_STATUS_CANCELED"
)

// Convert waived fee sources, why part of a convert trade's fee is waived
const (
	ConvertWaivedFeeSourceTradeIncentive    = "WAIVED_FEE_SOURCE_TRADE_INCENTIVE"
	ConvertWaivedFeeSourceCoinbaseOne       = "WAIVED_FEE_SOURCE_COINBASE_ONE"
	ConvertWaivedFeeSourceFeeReductionBadge = "WAIVED_FEE_SOURCE_FEE_REDUCTION_BADGE"
	ConvertWaivedFeeSourceTradePromotion    = "WAIVED_FEE_SOURCE_TRADE_PROMOTION"
)

// Convert payment method types, which name the detail field a ConvertPaymentMethod sets
const (
	ConvertPaymentMethodTypeBlockchainAddress           = "BLOCKCHAIN_ADDRESS"
	ConvertPaymentMethodTypeCoinbaseAccount             = "COINBASE_ACCOUNT"
	ConvertPaymentMethodTypeBlockchainTransaction       = "BLOCKCHAIN_TRANSACTION"
	ConvertPaymentMethodTypeFedwire                     = "FEDWIRE"
	ConvertPaymentMethodTypeSWIFT                       = "SWIFT"
	ConvertPaymentMethodTypeCard                        = "CARD"
	ConvertPaymentMethodTypeZengin                      = "ZENGIN"
	ConvertPaymentMethodTypeUK                          = "UK"
	ConvertPaymentMethodTypeSEPA                        = "SEPA"
	ConvertPaymentMethodTypePayPal                      = "PAYPAL"
	ConvertPaymentMethodTypeLedgerAccount               = "LEDGER_ACCOUNT"
	ConvertPaymentMethodTypeExternalPaymentMethod       = "EXTERNAL_PAYMENT_METHOD"
	ConvertPaymentMethodTypeProAccount                  = "PRO_ACCOUNT"
	ConvertPaymentMethodTypeRTP                         = "RTP"
	ConvertPaymentMethodTypeCardRepresentment           = "CARD_REPRESENTMENT"
	ConvertPaymentMethodTypeVenue                       = "VENUE"
	ConvertPaymentMethodTypeLedgerNamedAccount          = "LEDGER_NAMED_ACCOUNT"
	ConvertPaymentMethodTypeCustodialPool               = "CUSTODIAL_POOL"
	ConvertPaymentMethodTypeApplePay                    = "APPLE_PAY"
	ConvertPaymentMethodTypeDefaultAccount              = "DEFAULT_ACCOUNT"
	ConvertPaymentMethodTypeRemitly                     = "REMITLY"
	ConvertPaymentMethodTypeProInternalAccount          = "PRO_INTERNAL_ACCOUNT"
	ConvertPaymentMethodTypeDAppWalletAccount           = "DAPP_WALLET_ACCOUNT" //nolint:gosec // A payment method type, which G101 mistakes for a password as DAppWallet holds pw
	ConvertPaymentMethodTypeGooglePay                   = "GOOGLE_PAY"
	ConvertPaymentMethodTypeDAppWalletBlockchainAddress = "DAPP_WALLET_BLOCKCHAIN_ADDRESS" //nolint:gosec // A payment method type, which G101 mistakes for a password as DAppWallet holds pw
	ConvertPaymentMethodTypeZaakpayMobikwik             = "ZAAKPAY_MOBIKWIK"
	ConvertPaymentMethodTypeDenebUPI                    = "DENEB_UPI"
	ConvertPaymentMethodTypeBankAccount                 = "BANK_ACCOUNT"
	ConvertPaymentMethodTypeIdentityContractCall        = "IDENTITY_CONTRACT_CALL"
	ConvertPaymentMethodTypeDenebIMPS                   = "DENEB_IMPS"
	ConvertPaymentMethodTypeAllocation                  = "ALLOCATION"
	ConvertPaymentMethodTypeLiquidityPool               = "LIQUIDITY_POOL"
	ConvertPaymentMethodTypeZenginV2                    = "ZENGIN_V2"
	ConvertPaymentMethodTypeDirectDeposit               = "DIRECT_DEPOSIT"
	ConvertPaymentMethodTypeSEPAV2                      = "SEPA_V2"
	ConvertPaymentMethodTypeZepto                       = "ZEPTO"
	ConvertPaymentMethodTypePixEbanx                    = "PIX_EBANX"
	ConvertPaymentMethodTypeSignet                      = "SIGNET"
	ConvertPaymentMethodTypeDerivativeSettlement        = "DERIVATIVE_SETTLEMENT"
	ConvertPaymentMethodTypeUser                        = "USER"
	ConvertPaymentMethodTypeSingaporeFAST               = "SG_FAST"
	ConvertPaymentMethodTypeInterac                     = "INTERAC"
	ConvertPaymentMethodTypeIntraBank                   = "INTRA_BANK"
	ConvertPaymentMethodTypeCBIT                        = "CBIT"
	ConvertPaymentMethodTypeIDEAL                       = "IDEAL"
	ConvertPaymentMethodTypeSofort                      = "SOFORT"
	ConvertPaymentMethodTypeSingaporePayNow             = "SG_PAYNOW"
	ConvertPaymentMethodTypeCheckoutPaymentLink         = "CHECKOUT_PAYMENT_LINK"
	ConvertPaymentMethodTypeEmailAddress                = "EMAIL_ADDRESS"
	ConvertPaymentMethodTypePhoneNumber                 = "PHONE_NUMBER"
	ConvertPaymentMethodTypeVendorPayment               = "VENDOR_PAYMENT"
	ConvertPaymentMethodTypeCTN                         = "CTN"
	ConvertPaymentMethodTypeBancomatPay                 = "BANCOMAT_PAY"
	ConvertPaymentMethodTypeHotWallet                   = "HOT_WALLET"
	ConvertPaymentMethodTypeNovaAccount                 = "NOVA_ACCOUNT"
	ConvertPaymentMethodTypeMagicSpendBlockchainAddress = "MAGIC_SPEND_BLOCKCHAIN_ADDRESS"
	ConvertPaymentMethodTypeTransferPointer             = "TRANSFER_POINTER"
	ConvertPaymentMethodTypeEFT                         = "EFT"
	ConvertPaymentMethodTypeWallaceAccount              = "WALLACE_ACCOUNT"
	ConvertPaymentMethodTypeManualDeposit               = "MANUAL"
	ConvertPaymentMethodTypeArgentineBankAccount        = "ARGENTINE_BANK_ACCOUNT"
	ConvertPaymentMethodTypeRepresentment               = "REPRESENTMENT"
	ConvertPaymentMethodTypeBankingCircleNow            = "BANKING_CIRCLE_NOW"
	ConvertPaymentMethodTypeTrustly                     = "TRUSTLY"
	ConvertPaymentMethodTypeBLIK                        = "BLIK"
	ConvertPaymentMethodTypeMBWay                       = "MB_WAY"
	ConvertPaymentMethodTypePix                         = "PIX"
	ConvertPaymentMethodTypeMagicWallet                 = "MAGIC_WALLET"
	ConvertPaymentMethodTypeSamsungPay                  = "SAMSUNG_PAY"
	ConvertPaymentMethodTypeCADWire                     = "CAD_WIRE"
	ConvertPaymentMethodTypeWABIntraBank                = "WAB_INTRA_BANK"
	ConvertPaymentMethodTypeIndiaBankAccount            = "INDIA_BANK_ACCOUNT"
	ConvertPaymentMethodTypeOpenBanking                 = "OPEN_BANKING"
	ConvertPaymentMethodTypeSPEI                        = "SPEI"
	ConvertPaymentMethodTypeJPMorganKinexys             = "JPMORGAN_KINEXYS"
	ConvertPaymentMethodTypeSIC                         = "SIC"
	ConvertPaymentMethodTypePixDeposit                  = "PIX_DEPOSIT"
	ConvertPaymentMethodTypePixWithdrawal               = "PIX_WITHDRAWAL"
	ConvertPaymentMethodTypeFakeACH                     = "FAKE_ACH"
)

// CreateConvertQuoteRequest quotes converting an amount from one currency's account to another's
type CreateConvertQuoteRequest struct {
	// FromAccount and ToAccount are the accounts converted from and to, which Coinbase documents by their currency,
	// such as USD and USDC
	FromAccount string
	ToAccount   string
	// Amount is in FromAccount's currency
	Amount float64
	// UserIncentiveID and CodeValue, a promo code, apply a trade incentive that waives the trade's fees
	UserIncentiveID string
	CodeValue       string
}

// CreateConvertQuoteWireRequest is CreateConvertQuote's body
type CreateConvertQuoteWireRequest struct {
	FromAccount            string                         `json:"from_account"`
	ToAccount              string                         `json:"to_account"`
	Amount                 string                         `json:"amount"`
	TradeIncentiveMetadata *ConvertTradeIncentiveMetadata `json:"trade_incentive_metadata,omitempty"`
}

// ConvertTradeIncentiveMetadata is the trade incentive a convert quote applies
type ConvertTradeIncentiveMetadata struct {
	UserIncentiveID string `json:"user_incentive_id,omitempty"`
	CodeValue       string `json:"code_val,omitempty"`
}

// ConvertTradeRequest selects a convert trade, by the trade ID its quote returned and the accounts it converts
// between
type ConvertTradeRequest struct {
	TradeID string
	// FromAccount and ToAccount are the accounts the trade's quote converts from and to
	FromAccount string
	ToAccount   string
}

// ConvertAccountsWireRequest is CommitConvertTrade's body, the accounts a trade converts between
type ConvertAccountsWireRequest struct {
	FromAccount string `json:"from_account"`
	ToAccount   string `json:"to_account"`
}

// ConvertTradeResponse holds a convert trade
type ConvertTradeResponse struct {
	Trade *ConvertTrade `json:"trade"`
}

// ConvertTrade is a convert quote, or the trade that commits it, between two currencies
type ConvertTrade struct {
	// ID is the trade ID CommitConvertTrade and GetConvertTrade take
	ID string `json:"id"`
	// Status is one of the ConvertTradeStatus values
	Status            string `json:"status"`
	UserEnteredAmount Amount `json:"user_entered_amount"`
	Amount            Amount `json:"amount"`
	Subtotal          Amount `json:"subtotal"`
	Total             Amount `json:"total"`
	// Fees are the fees charged; TotalFee is their total and TotalFeeWithoutTax that total less tax
	Fees     []ConvertFee `json:"fees"`
	TotalFee *ConvertFee  `json:"total_fee"`
	// Source and Target are where the funds come from and go to
	Source         *ConvertPaymentMethod `json:"source"`
	Target         *ConvertPaymentMethod `json:"target"`
	UnitPrice      *ConvertUnitPrice     `json:"unit_price"`
	UserWarnings   []ConvertUserWarning  `json:"user_warnings"`
	UserReference  string                `json:"user_reference"`
	SourceCurrency currency.Code         `json:"source_currency"`
	TargetCurrency currency.Code         `json:"target_currency"`
	// CancellationReason is set when the trade was cancelled
	CancellationReason *ConvertCancellationReason `json:"cancellation_reason"`
	SourceID           string                     `json:"source_id"`
	TargetID           string                     `json:"target_id"`
	SubscriptionInfo   *ConvertSubscriptionInfo   `json:"subscription_info"`
	ExchangeRate       Amount                     `json:"exchange_rate"`
	TaxDetails         []ConvertTax               `json:"tax_details"`
	TradeIncentiveInfo *ConvertTradeIncentive     `json:"trade_incentive_info"`
	TotalFeeWithoutTax *ConvertFee                `json:"total_fee_without_tax"`
	FiatDenotedTotal   Amount                     `json:"fiat_denoted_total"`
}

// ConvertFee is a fee a convert trade charges
type ConvertFee struct {
	Title         string                `json:"title"`
	Description   string                `json:"description"`
	Amount        Amount                `json:"amount"`
	Label         string                `json:"label"`
	Disclosure    *ConvertFeeDisclosure `json:"disclosure"`
	WaivedDetails *ConvertWaivedFee     `json:"waived_details"`
	Metadata      *ConvertFeeMetadata   `json:"metadata"`
}

// ConvertFeeDisclosure is a fee's localised disclosure
type ConvertFeeDisclosure struct {
	Title       string       `json:"title"`
	Description string       `json:"description"`
	Link        *ConvertLink `json:"link"`
}

// ConvertLink is an in-app link, usually to a support page
type ConvertLink struct {
	Text string `json:"text"`
	URL  string `json:"url"`
}

// ConvertWaivedFee is the part of a fee that is waived, and why
type ConvertWaivedFee struct {
	Amount Amount `json:"amount"`
	// Source is one of the ConvertWaivedFeeSource values
	Source string `json:"source"`
}

// ConvertFeeMetadata holds a fee's calculation details
type ConvertFeeMetadata struct {
	NetBurn *ConvertNetBurnFee `json:"netburn"`
}

// ConvertNetBurnFee is a net burn fee's threshold and value
type ConvertNetBurnFee struct {
	Threshold    Amount `json:"threshold"`
	NetBurnValue Amount `json:"netburn_value"`
}

// ConvertPaymentMethod is where a convert trade's funds come from or go to. Type names it and the one detail field set
// for it: LEDGER_ACCOUNT sets LedgerAccount, BLOCKCHAIN_ADDRESS sets BlockchainAddress and so on
type ConvertPaymentMethod struct {
	// Type is one of the ConvertPaymentMethodType values
	Type                        string                              `json:"type"`
	Network                     string                              `json:"network"`
	PaymentMethodID             string                              `json:"payment_method_id"`
	PaymentMethodUUID           string                              `json:"payment_method_uuid"`
	PaymentMethodTypeString     string                              `json:"payment_method_type_string"`
	BlockchainAddress           *ConvertBlockchainAddress           `json:"blockchain_address"`
	CoinbaseAccount             *ConvertCoinbaseAccount             `json:"coinbase_account"`
	BlockchainTransaction       *ConvertBlockchainTransaction       `json:"blockchain_transaction"`
	Fedwire                     *ConvertFedwire                     `json:"fedwire"`
	SWIFT                       *ConvertSWIFT                       `json:"swift"`
	Card                        *ConvertCard                        `json:"card"`
	Zengin                      *ConvertZengin                      `json:"zengin"`
	UK                          *ConvertUK                          `json:"uk"`
	SEPA                        *ConvertSEPA                        `json:"sepa"`
	PayPal                      *ConvertPayPal                      `json:"paypal"`
	LedgerAccount               *ConvertLedgerAccount               `json:"ledger_account"`
	ExternalPaymentMethod       *ConvertExternalPaymentMethod       `json:"external_payment_method"`
	ProAccount                  *ConvertProAccount                  `json:"pro_account"`
	RTP                         *ConvertRTP                         `json:"rtp"`
	Venue                       *ConvertNamedEntity                 `json:"venue"`
	LedgerNamedAccount          *ConvertLedgerNamedAccount          `json:"ledger_named_account"`
	CustodialPool               *ConvertCustodialPool               `json:"custodial_pool"`
	ApplePay                    *ConvertApplePay                    `json:"apple_pay"`
	DefaultAccount              *ConvertDefaultAccount              `json:"default_account"`
	Remitly                     *ConvertRemitly                     `json:"remitly"`
	ProInternalAccount          *ConvertProInternalAccount          `json:"pro_internal_account"`
	DAppWalletAccount           *ConvertDAppWalletAccount           `json:"dapp_wallet_account"`
	GooglePay                   *ConvertGooglePay                   `json:"google_pay"`
	DAppWalletBlockchainAddress *ConvertDAppWalletBlockchainAddress `json:"dapp_wallet_blockchain_address"`
	ZaakpayMobikwik             *ConvertPhoneAccount                `json:"zaakpay_mobikwik"`
	DenebUPI                    *ConvertDenebUPI                    `json:"deneb_upi"`
	BankAccount                 *ConvertBankAccount                 `json:"bank_account"`
	IdentityContractCall        *ConvertNetworkAddress              `json:"identity_contract_call"`
	DenebIMPS                   *ConvertDenebIMPS                   `json:"deneb_imps"`
	Allocation                  *ConvertAllocation                  `json:"allocation"`
	LiquidityPool               *ConvertLiquidityPool               `json:"liquidity_pool"`
	ZenginV2                    *ConvertZengin                      `json:"zengin_v2"`
	DirectDeposit               *ConvertDirectDeposit               `json:"direct_deposit"`
	SEPAV2                      *ConvertSEPAV2                      `json:"sepa_v2"`
	Zepto                       *ConvertZepto                       `json:"zepto"`
	PixEbanx                    *ConvertPixEbanx                    `json:"pix_ebanx"`
	Signet                      *ConvertSignet                      `json:"signet"`
	DerivativeSettlement        *ConvertDerivativeSettlement        `json:"derivative_settlement"`
	User                        *ConvertUser                        `json:"user"`
	SingaporeFAST               *ConvertSingaporeFAST               `json:"sg_fast"`
	Interac                     *ConvertInterac                     `json:"interac"`
	IntraBank                   *ConvertIntraBank                   `json:"intra_bank"`
	CBIT                        *ConvertCBIT                        `json:"cbit"`
	IDEAL                       *ConvertBankRedirect                `json:"ideal"`
	Sofort                      *ConvertBankRedirect                `json:"sofort"`
	SingaporePayNow             *ConvertSingaporePayNow             `json:"sg_paynow"`
	CheckoutPaymentLink         *ConvertCheckoutPaymentLink         `json:"checkout_payment_link"`
	EmailAddress                *ConvertContact                     `json:"email_address"`
	PhoneNumber                 *ConvertContact                     `json:"phone_number"`
	VendorPayment               *ConvertVendorPayment               `json:"vendor_payment"`
	CTN                         *ConvertCTN                         `json:"ctn"`
	BancomatPay                 *ConvertBancomatPay                 `json:"bancomat_pay"`
	HotWallet                   *ConvertNetworkAddress              `json:"hot_wallet"`
	NovaAccount                 *ConvertNovaAccount                 `json:"nova_account"`
	MagicSpendBlockchainAddress *ConvertBlockchainAddress           `json:"magic_spend_blockchain_address"`
	TransferPointer             *ConvertTransferPointer             `json:"transfer_pointer"`
	EFT                         *ConvertEFT                         `json:"eft"`
	WallaceAccount              *ConvertWallaceAccount              `json:"wallace_account"`
	ManualDeposit               *ConvertManualDeposit               `json:"manual"`
	ArgentineBankAccount        *ConvertArgentineBankAccount        `json:"argentine_bank_account"`
	Representment               *ConvertRepresentment               `json:"representment"`
	BankingCircleNow            *ConvertBankingCircleNow            `json:"banking_circle_now"`
	Trustly                     *ConvertTrustly                     `json:"trustly"`
	BLIK                        *ConvertBLIK                        `json:"blik"`
	MBWay                       *ConvertMBWay                       `json:"mb_way"`
	Pix                         *ConvertPix                         `json:"pix"`
	MagicWallet                 *ConvertMagicWallet                 `json:"magic_wallet"`
	SamsungPay                  *ConvertSamsungPay                  `json:"samsung_pay"`
	CADWire                     *ConvertCADWire                     `json:"cad_wire"`
	WABIntraBank                *ConvertWABIntraBank                `json:"wab_intra_bank"`
	IndiaBankAccount            *ConvertIndiaBankAccount            `json:"india_bank_account"`
	JPMorganKinexys             *ConvertJPMorganKinexys             `json:"jpmorgan_kinexys"`
	SIC                         *ConvertSIC                         `json:"sic"`
	SPEI                        *ConvertAccountHolder               `json:"spei"`
}

// ConvertBlockchainAddress is a blockchain address, with the destination tag some networks route by
type ConvertBlockchainAddress struct {
	Address        string `json:"address"`
	DestinationTag string `json:"destination_tag"`
}

// ConvertCoinbaseAccount is a Coinbase account
type ConvertCoinbaseAccount struct {
	AccountID string `json:"account_id"`
}

// ConvertBlockchainTransaction is a blockchain transaction
type ConvertBlockchainTransaction struct {
	Hash   string `json:"hsh"`
	Height uint64 `json:"height"`
	// NormalisedHash is set for some assets' deposits and withdrawals
	NormalisedHash string `json:"normalized_hash"`
}

// ConvertFedwire is a US bank account paid by Fedwire
type ConvertFedwire struct {
	RoutingNumber    string                     `json:"routing_number"`
	AccountHolder    *ConvertFedwireAccount     `json:"account_holder"`
	Bank             *ConvertFedwireInstitution `json:"bank"`
	IntermediaryBank *ConvertFedwireInstitution `json:"intermediary_bank"`
	Currency         currency.Code              `json:"currency"`
	// FiatID is only used by Coinbase Prime
	FiatID string `json:"fiat_id"`
	// RejectionCount is how often the payment method has been rejected
	RejectionCount uint64 `json:"rejection_count_v2"`
}

// ConvertFedwireAccount is a Fedwire bank account and its holder
type ConvertFedwireAccount struct {
	LegalName     string                 `json:"legal_name"`
	AccountNumber string                 `json:"account_number"`
	Address       *ConvertFedwireAddress `json:"address"`
}

// ConvertFedwireAddress is an address on a Fedwire payment
type ConvertFedwireAddress struct {
	// Lines are the address lines, at least 2; Fedwire takes only the first 3
	Lines []string `json:"lines"`
	// CountryCode is an ISO 3166-1 alpha-2 code, such as US
	CountryCode string `json:"country_code"`
	City        string `json:"city"`
	State       string `json:"state"`
	PostalCode  string `json:"postal_code"`
}

// ConvertFedwireInstitution is a bank, credit union or other financial institution Fedwire pays
type ConvertFedwireInstitution struct {
	Name    string                 `json:"name"`
	Address *ConvertFedwireAddress `json:"address"`
	// Identifier is the institution's identifier of the kind IdentifierCode names: a 9 digit ABA routing number for F,
	// a SWIFT code for B, or an account number for further credit through an intermediary bank for D
	Identifier string `json:"identifier"`
	// Type is the kind of identifier, such as FEDWIRE, SWIFT_BIC or CHIPS_IDENTIFIER
	Type string `json:"type"`
	// IdentifierCode is B for a SWIFT BIC, C for a CHIPS participant, D for a demand deposit account number, F for a
	// Fed routing number, T for a SWIFT BIC or BEI and account number, or U for a CHIPS identifier
	IdentifierCode string `json:"identifier_code"`
}

// ConvertSWIFT is a bank account paid by SWIFT
type ConvertSWIFT struct {
	AccountHolder *ConvertSWIFTAccount     `json:"account_holder"`
	Institution   *ConvertSWIFTInstitution `json:"institution"`
	Intermediary  *ConvertSWIFTInstitution `json:"intermediary"`
	Currency      currency.Code            `json:"currency"`
	// RejectionCount is how often the payment method has been rejected
	RejectionCount uint64 `json:"rejection_count"`
}

// ConvertSWIFTAccount is a SWIFT bank account and its holder
type ConvertSWIFTAccount struct {
	LegalName                  string `json:"legal_name"`
	IBAN                       string `json:"iban"`
	BBAN                       string `json:"bban"`
	DomesticAccountID          string `json:"domestic_account_id"`
	CustomerPaymentAddress1    string `json:"customer_payment_address1"`
	CustomerPaymentAddress2    string `json:"customer_payment_address2"`
	CustomerPaymentAddress3    string `json:"customer_payment_address3"`
	CustomerPaymentCountryCode string `json:"customer_payment_country_code"`
	// IBANGenerated is set when Coinbase generated the IBAN
	IBANGenerated bool `json:"iban_generated"`
}

// ConvertSWIFTInstitution is a bank SWIFT pays
type ConvertSWIFTInstitution struct {
	BIC                 string `json:"bic"`
	Name                string `json:"name"`
	BankAddress1        string `json:"bank_address1"`
	BankAddress2        string `json:"bank_address2"`
	BankAddress3        string `json:"bank_address3"`
	BankCountryCode     string `json:"bank_country_code"`
	DomesticBankID      string `json:"domestic_bank_id"`
	InternationalBankID string `json:"international_bank_id"`
}

// ConvertCard is a debit or credit card
type ConvertCard struct {
	FirstDataToken *ConvertFirstDataToken `json:"first_data_token"`
	Merchant       *ConvertCardMerchant   `json:"merchant"`
	VaultToken     *ConvertVaultToken     `json:"vault_token"`
	WorldpayParams *ConvertWorldpayParams `json:"worldpay_params"`
	CheckoutToken  *ConvertCheckoutToken  `json:"checkout_token"`
	StripeParams   *ConvertStripeParams   `json:"stripe_params"`
	SixDigitBIN    string                 `json:"six_digit_bin"`
	LastFour       string                 `json:"last_four"`
	EightDigitBIN  string                 `json:"eight_digit_bin"`
	CardSubBrand   string                 `json:"card_sub_brand"`
	CardBrand      string                 `json:"card_brand"`
	ObfuscatedPAN  string                 `json:"obfuscated_pan"`
	BINInfo        *ConvertCardBINInfo    `json:"bin_info"`
	// Scheme is UNKNOWN, VISA or MASTERCARD
	Scheme            string               `json:"scheme"`
	Issuer            string               `json:"issuer"`
	IssuerCountryCode string               `json:"issuer_country_code"`
	Address           *ConvertAddress      `json:"address"`
	PhoneNumber       string               `json:"phone_number"`
	UserID            string               `json:"user_id"`
	PostalCode        string               `json:"postal_code"`
	CustomerInfo      *ConvertCustomerInfo `json:"customer_info"`
	ExpiryDate        *ConvertExpiryDate   `json:"expiry_date"`
	// CDVMethod, CDVAmount1 and CDVAmount2 are the card debit verification, which confirms a card by two small charges
	CDVMethod                   string                    `json:"cdv_method"`
	CDVAmount1                  ConvertMoney              `json:"cdv_amount_1"`
	CDVAmount2                  ConvertMoney              `json:"cdv_amount_2"`
	WorldpayCardCDVStatus       *ConvertWorldpayCDVStatus `json:"worldpay_card_cdv_status"`
	PreviousSchemeTransactionID string                    `json:"previous_scheme_tx_id"`
	InitialSchemeTransactionID  string                    `json:"initial_scheme_tx_id"`
	Rejected                    bool                      `json:"rejected"`
	RejectedAt                  time.Time                 `json:"rejected_at"`
	// RejectionCount is how often the card has been rejected
	RejectionCount          uint64    `json:"rejection_count"`
	RejectedViaPurchase     bool      `json:"rejected_via_purchase"`
	AccountUpdaterUpdatedAt time.Time `json:"account_updater_updated_at"`
	AccountUpdaterReason    string    `json:"account_updater_reason"`
	AccountUpdaterStatus    string    `json:"account_updater_status"`
	AlmostExpiredNotified   bool      `json:"almost_expired_notified"`
	IsMigratedCard          bool      `json:"is_migrated_card"`
	MigratedCreatedAt       time.Time `json:"migrated_created_at"`
	// CardSource is how the card was added: SOURCE_UNKNOWN, CARD_IO_SCAN, MANUAL_ENTRY, DYNETI_SCAN,
	// DYNETI_SCAN_AND_MANUAL or AUTO_ASSIGNED
	CardSource               string                      `json:"card_source"`
	AddedViaBuyWidget        bool                        `json:"added_via_buy_widget"`
	EncryptedData            string                      `json:"encrypted_data"`
	VerificationMethod       string                      `json:"verification_method"`
	CoinbaseCreditCardParams *ConvertCoinbaseCreditCard  `json:"coinbase_credit_card_params"`
	AddressVerificationInfo  *ConvertAddressVerification `json:"address_verification_info"`
	AccountNameInquiry       *ConvertAccountNameInquiry  `json:"account_name_inquiry"`
	// CustomerName, CustomerFirstName, CustomerLastName and CustomerDateOfBirth are deprecated
	CustomerName        string              `json:"customer_name"`
	CustomerFirstName   string              `json:"customer_first_name"`
	CustomerLastName    string              `json:"customer_last_name"`
	CustomerDateOfBirth *ConvertDateOfBirth `json:"customer_dob"`
}

// ConvertFirstDataToken is a card's First Data token
type ConvertFirstDataToken struct {
	Value   string `json:"value"`
	StoreID string `json:"store_id"`
	ID      string `json:"id"`
}

// ConvertCardMerchant is the merchant a card is charged through
type ConvertCardMerchant struct {
	MerchantID string `json:"mid"`
}

// ConvertVaultToken is a card's vault token
type ConvertVaultToken struct {
	Value         string    `json:"value"`
	VaultID       string    `json:"vault_id"`
	AttemptedAt   time.Time `json:"attempted_at"`
	EncryptedData string    `json:"encrypted_data"`
	EncryptedCVV  string    `json:"encrypted_cvv"`
}

// ConvertWorldpayParams are a card's Worldpay parameters
type ConvertWorldpayParams struct {
	TokenValue        string `json:"token_value"`
	TokenEvent        string `json:"token_event"`
	UsesMerchantToken bool   `json:"uses_merchant_token"`
	AcceptHeader      string `json:"accept_header"`
	UserAgentHeader   string `json:"user_agent_header"`
	ShopperIP         string `json:"shopper_ip"`
	ShopperSessionID  string `json:"shopper_session_id"`
}

// ConvertCheckoutToken is a card's Checkout.com payment source
type ConvertCheckoutToken struct {
	SourceID string `json:"source_id"`
}

// ConvertStripeParams are a card's Stripe payment method and the checks Stripe made
type ConvertStripeParams struct {
	StripeID       string `json:"stripe_id"`
	RadarSessionID string `json:"radar_session_id"`
	// AddressLine1Check, AddressPostalCodeCheck and CVCCheck are STRIPE_CARD_CHECK_STATUS_ values: UNKNOWN, PASS, FAIL,
	// UNAVAILABLE or UNCHECKED
	AddressLine1Check      string `json:"address_line1_check"`
	AddressPostalCodeCheck string `json:"address_postal_code_check"`
	CVCCheck               string `json:"cvc_check"`
}

// ConvertCardBINInfo is what a card's bank identification number says of it; it is empty unless MatchFound
type ConvertCardBINInfo struct {
	MatchFound bool `json:"match_found"`
	// Type is UNKNOWN, CREDIT or DEBIT
	Type              string                 `json:"type"`
	Issuer            string                 `json:"issuer"`
	IssuerCountryCode string                 `json:"issuer_country_code"`
	Prepaid           bool                   `json:"prepaid"`
	Corporate         bool                   `json:"corporate"`
	Consumer          bool                   `json:"consumer"`
	Visa              *ConvertVisaInfo       `json:"visa"`
	Mastercard        *ConvertMastercardInfo `json:"mastercard"`
	// Use3DS is set for a card that uses 3D Secure
	Use3DS bool `json:"use_3ds"`
}

// ConvertVisaInfo is what a Visa card supports
type ConvertVisaInfo struct {
	// OCTSupported is whether the card takes original credit transactions, payments to the card itself
	OCTSupported bool `json:"oct_supported"`
	// FastFundsSupported is whether the card holder receives funds almost at once
	FastFundsSupported bool `json:"fast_funds_supported"`
}

// ConvertMastercardInfo is what a Mastercard supports
type ConvertMastercardInfo struct {
	// FastFundsSupported is whether the card holder receives funds almost at once
	FastFundsSupported bool `json:"fast_funds_supported"`
}

// ConvertAddress is a postal address
type ConvertAddress struct {
	Address1   string `json:"address1"`
	Address2   string `json:"address2"`
	City       string `json:"city"`
	State      string `json:"state"`
	PostalCode string `json:"postal_code"`
	Country    string `json:"country"`
	UUID       string `json:"uuid"`
	ID         string `json:"id"`
}

// ConvertCustomerInfo is a card holder's details
type ConvertCustomerInfo struct {
	FullName    string              `json:"full_name"`
	FirstName   string              `json:"first_name"`
	LastName    string              `json:"last_name"`
	DateOfBirth *ConvertDateOfBirth `json:"dob"`
	// Nationality is the card holder's ISO 3166-1 alpha-2 country code
	Nationality string `json:"nationality"`
	Email       string `json:"email"`
}

// ConvertDateOfBirth is a date of birth
type ConvertDateOfBirth struct {
	Month string `json:"month"`
	Day   string `json:"day"`
	Year  string `json:"year"`
}

// ConvertExpiryDate is a card's expiry month and year
type ConvertExpiryDate struct {
	Month string `json:"month"`
	Year  string `json:"year"`
}

// ConvertMoney is an amount of a currency; a fiat amount is in the currency's smallest unit, so 1 is 0.01 USD or 1 JPY
type ConvertMoney struct {
	Amount   types.Number  `json:"amount"`
	Currency currency.Code `json:"currency"`
}

// ConvertWorldpayCDVStatus is a card's Worldpay card debit verification status
type ConvertWorldpayCDVStatus struct {
	CDVRequired bool `json:"cdv_required"`
	CDVFailed   bool `json:"cdv_failed"`
}

// ConvertCoinbaseCreditCard is a Coinbase credit card's account, which Coinbase One fees are charged to
type ConvertCoinbaseCreditCard struct {
	CreditCardAccountID string `json:"credit_card_account_id"`
}

// ConvertAddressVerification is a card's address verification result
type ConvertAddressVerification struct {
	Source string `json:"source"`
	// AVSStreetAddress and AVSPostalCode are ADDRESS_VERIFICATION_STATUS_MATCH or ADDRESS_VERIFICATION_STATUS_NO_MATCH
	AVSStreetAddress string `json:"avs_street_address"`
	AVSPostalCode    string `json:"avs_postal_code"`
}

// ConvertAccountNameInquiry is a Visa account name inquiry's result, which matches the card holder's name against the
// issuing bank's
type ConvertAccountNameInquiry struct {
	// FirstNameMatch, MiddleNameMatch and LastNameMatch are MATCH_RESULT_MATCH, MATCH_RESULT_NO_MATCH or
	// MATCH_RESULT_PARTIAL_MATCH
	FirstNameMatch  string `json:"first_name_match"`
	MiddleNameMatch string `json:"middle_name_match"`
	LastNameMatch   string `json:"last_name_match"`
	// NameMatchIndicator is NAME_MATCH_INDICATOR_PERFORMED, NAME_MATCH_INDICATOR_NOT_SUPPORTED or
	// NAME_MATCH_INDICATOR_NOT_PERFORMED
	NameMatchIndicator string `json:"name_match_indicator"`
}

// ConvertZengin is a Japanese bank account paid through Zengin
type ConvertZengin struct {
	AccountHolder *ConvertZenginAccount     `json:"account_holder"`
	Institution   *ConvertZenginInstitution `json:"institution"`
}

// ConvertZenginAccount is a Zengin bank account and its holder
type ConvertZenginAccount struct {
	LegalName  string `json:"legal_name"`
	Identifier string `json:"identifier"`
	Type       string `json:"type"`
}

// ConvertZenginInstitution is a Zengin bank and branch
type ConvertZenginInstitution struct {
	BankCode   string `json:"bank_code"`
	BranchCode string `json:"branch_code"`
}

// ConvertUK is a UK bank account
type ConvertUK struct {
	AccountHolder     *ConvertUKAccount   `json:"account_holder"`
	Institution       *ConvertNamedEntity `json:"institution"`
	CustomerFirstName string              `json:"customer_first_name"`
	CustomerLastName  string              `json:"customer_last_name"`
	Email             string              `json:"email"`
	PhoneNumber       string              `json:"phone_number"`
	Currency          currency.Code       `json:"currency"`
	OpenBanking       *ConvertOpenBanking `json:"open_banking_details"`
}

// ConvertUKAccount is a UK bank account and its holder
type ConvertUKAccount struct {
	LegalName     string          `json:"legal_name"`
	BBAN          string          `json:"bban"`
	SortCode      string          `json:"sort_code"`
	AccountNumber string          `json:"account_number"`
	Address       *ConvertAddress `json:"address"`
	// AccountType is UK_ACCOUNT_TYPE_SAVINGS for a savings account
	AccountType string `json:"account_type"`
}

// ConvertNamedEntity is a UK bank or a venue, known by its name
type ConvertNamedEntity struct {
	Name string `json:"name"`
}

// ConvertOpenBanking identifies a bank to Open Banking
type ConvertOpenBanking struct {
	// TrueLayerBankProviderID is TrueLayer's ID of the bank
	TrueLayerBankProviderID string `json:"true_layer_bank_provider_id"`
	// OpenBankingProviderID is the bank's universal Open Banking ID
	OpenBankingProviderID string `json:"open_banking_provider_id"`
}

// ConvertSEPA is a bank account paid by SEPA transfer
type ConvertSEPA struct {
	AccountHolder     *ConvertSEPAAccount     `json:"account_holder"`
	Institution       *ConvertSEPAInstitution `json:"institution"`
	CustomerFirstName string                  `json:"customer_first_name"`
	CustomerLastName  string                  `json:"customer_last_name"`
	Email             string                  `json:"email"`
	PhoneNumber       string                  `json:"phone_number"`
}

// ConvertSEPAAccount is a SEPA bank account and its holder
type ConvertSEPAAccount struct {
	LegalName string `json:"legal_name"`
	IBAN      string `json:"iban"`
	BBAN      string `json:"bban"`
}

// ConvertSEPAInstitution is a bank SEPA transfers pay
type ConvertSEPAInstitution struct {
	// BIC is the bank's 8 to 11 character ISO 9362 SWIFT code
	BIC             string `json:"bic"`
	Name            string `json:"name"`
	BankCountryCode string `json:"bank_country_code"`
	BankCountryName string `json:"bank_country_name"`
	BankAddress     string `json:"bank_address"`
}

// ConvertPayPal is a PayPal account
type ConvertPayPal struct {
	AccountHolder *ConvertPayPalAccount  `json:"account_holder"`
	Merchant      *ConvertPayPalMerchant `json:"merchant"`
	Metadata      *ConvertPayPalMetadata `json:"metadata"`
	Email         string                 `json:"email"`
	OwnerName     string                 `json:"owner_name"`
	// BirthDate is kept as sent, as Coinbase leaves its format undocumented
	BirthDate           string          `json:"birth_date"`
	PhoneNumber         string          `json:"phone_number"`
	PhoneType           string          `json:"phone_type"`
	BillingAddress      *ConvertAddress `json:"billing_address"`
	BraintreeCustomerID string          `json:"braintree_customer_id"`
	PaymentMethodNonce  string          `json:"payment_method_nonce"`
	PayerID             string          `json:"payer_id"`
}

// ConvertPayPalAccount is a customer's PayPal account
type ConvertPayPalAccount struct {
	PayPalID              string `json:"paypal_id"`
	PayPalPaymentMethodID string `json:"paypal_pm_id"`
	PayPalVerified        bool   `json:"paypal_verified"`
	AccountType           string `json:"account_type"`
	AccountAge            string `json:"account_age"`
}

// ConvertPayPalMerchant is Coinbase's PayPal merchant account, which payouts are paid from
type ConvertPayPalMerchant struct {
	MerchantAccountID string `json:"merchant_account_id"`
}

// ConvertPayPalMetadata is a PayPal payment's correlation ID and authorisation code
type ConvertPayPalMetadata struct {
	PayPalCorrelationID string `json:"paypal_correlation_id"`
	AuthorisationCode   string `json:"authorization_code"`
}

// ConvertLedgerAccount is an account in Coinbase's ledger, such as a wallet
type ConvertLedgerAccount struct {
	AccountID   string                     `json:"account_id"`
	Currency    currency.Code              `json:"currency"`
	Owner       *ConvertLedgerAccountOwner `json:"owner"`
	AccountUUID string                     `json:"account_uuid"`
	AccountName string                     `json:"account_name"`
}

// ConvertLedgerAccountOwner is whoever owns a ledger account
type ConvertLedgerAccountOwner struct {
	ID       string `json:"id"`
	UUID     string `json:"uuid"`
	UserUUID string `json:"user_uuid"`
	// Type is UNKNOWN, INVESTMENT_VEHICLE, RETAIL, VENUE or PORTFOLIO
	Type string `json:"type"`
}

// ConvertExternalPaymentMethod is a payment method Coinbase's payment method service holds
type ConvertExternalPaymentMethod struct {
	PaymentMethodID string `json:"payment_method_id"`
}

// ConvertProAccount is an account on Pro, Coinbase's exchange
type ConvertProAccount struct {
	AccountID         string        `json:"account_id"`
	CoinbaseAccountID string        `json:"coinbase_account_id"`
	UserID            string        `json:"user_id"`
	Currency          currency.Code `json:"currency"`
	PortfolioID       string        `json:"portfolio_id"`
}

// ConvertRTP is a US bank account paid over the RTP network
type ConvertRTP struct {
	AccountHolder *ConvertRTPAccount     `json:"account_holder"`
	Institution   *ConvertRTPInstitution `json:"institution"`
	Currency      currency.Code          `json:"currency"`
}

// ConvertRTPAccount is an RTP bank account and its holder
type ConvertRTPAccount struct {
	LegalName     string `json:"legal_name"`
	Identifier    string `json:"identifier"`
	BankAccountID string `json:"bank_account_id"`
}

// ConvertRTPInstitution is a bank the RTP network pays
type ConvertRTPInstitution struct {
	RoutingNumber string `json:"routing_number"`
}

// ConvertLedgerNamedAccount is a named account in Coinbase's ledger
type ConvertLedgerNamedAccount struct {
	Name           string        `json:"name"`
	Currency       currency.Code `json:"currency"`
	ForeignNetwork string        `json:"foreign_network"`
}

// ConvertCustodialPool is a custodial pool of funds
type ConvertCustodialPool struct {
	Name    string `json:"name"`
	Network string `json:"network"`
	FiatID  string `json:"fiat_id"`
}

// ConvertApplePay is a card paid through Apple Pay
type ConvertApplePay struct {
	Braintree *ConvertBraintree     `json:"braintree"`
	ApplePay  *ConvertDigitalWallet `json:"apple_pay"`
	UserID    string                `json:"user_id"`
	// PostalCode and the fields after it are set once an order is submitted
	PostalCode     string          `json:"postal_code"`
	CustomerName   string          `json:"customer_name"`
	Address        *ConvertAddress `json:"address"`
	SixDigitBIN    string          `json:"six_digit_bin"`
	LastFour       string          `json:"last_four"`
	IssuingCountry string          `json:"issuing_country"`
	IssuingBank    string          `json:"issuing_bank"`
	ProductID      string          `json:"product_id"`
	// CustomerDateOfBirth is in YYYY-MM-DD form
	CustomerDateOfBirth string        `json:"customer_dob"`
	BillingAddressID    string        `json:"billing_address_id"`
	Currency            currency.Code `json:"currency"`
	// Scheme is UNKNOWN, VISA or MASTERCARD, and Prepaid and Debit UNCERTAIN, TRUE or FALSE
	Scheme  string `json:"scheme"`
	Prepaid string `json:"prepaid"`
	Debit   string `json:"debit"`
}

// ConvertBraintree is a Braintree payment's nonce and correlation ID
type ConvertBraintree struct {
	Nonce         string `json:"nonce"`
	CorrelationID string `json:"correlation_id"`
}

// ConvertDigitalWallet is a digital wallet payment's nonce, correlation ID and 3DS version
type ConvertDigitalWallet struct {
	Nonce         string `json:"nonce"`
	CorrelationID string `json:"correlation_id"`
	// Version is the payment's 3DS version, such as 100
	Version string `json:"version"`
}

// ConvertDefaultAccount is a user's default account for a currency
type ConvertDefaultAccount struct {
	UserUUID string        `json:"user_uuid"`
	Currency currency.Code `json:"currency"`
}

// ConvertRemitly is a Remitly recipient
type ConvertRemitly struct {
	AccountHolder *ConvertRemitlyAccount `json:"account_holder"`
}

// ConvertRemitlyAccount is a Remitly recipient and how it is paid out
type ConvertRemitlyAccount struct {
	RecipientID      string        `json:"recipient_id"`
	PayoutMethodType string        `json:"payout_method_type"`
	RecipientVersion string        `json:"recipient_version"`
	Currency         currency.Code `json:"currency"`
}

// ConvertProInternalAccount is an internal account on Pro
type ConvertProInternalAccount struct {
	UserID   string        `json:"user_id"`
	Currency currency.Code `json:"currency"`
}

// ConvertDAppWalletAccount is a user's dapp wallet account
type ConvertDAppWalletAccount struct {
	UserUUID       string        `json:"user_uuid"`
	Network        string        `json:"network"`
	CohortID       string        `json:"cohort_id"`
	SigningBackend string        `json:"signing_backend"`
	Currency       currency.Code `json:"currency"`
}

// ConvertGooglePay is a card paid through Google Pay
type ConvertGooglePay struct {
	Braintree *ConvertBraintree     `json:"braintree"`
	GooglePay *ConvertDigitalWallet `json:"google_pay"`
	UserID    string                `json:"user_id"`
	// PostalCode and the fields after it are set once an order is submitted
	PostalCode     string          `json:"postal_code"`
	CustomerName   string          `json:"customer_name"`
	Address        *ConvertAddress `json:"address"`
	SixDigitBIN    string          `json:"six_digit_bin"`
	LastFour       string          `json:"last_four"`
	IssuingCountry string          `json:"issuing_country"`
	IssuingBank    string          `json:"issuing_bank"`
	ProductID      string          `json:"product_id"`
	// CustomerDateOfBirth is in YYYY-MM-DD form
	CustomerDateOfBirth string        `json:"customer_dob"`
	BillingAddressID    string        `json:"billing_address_id"`
	Currency            currency.Code `json:"currency"`
	// Scheme is UNKNOWN, VISA or MASTERCARD, and Prepaid and Debit UNCERTAIN, TRUE or FALSE
	Scheme  string `json:"scheme"`
	Prepaid string `json:"prepaid"`
	Debit   string `json:"debit"`
}

// ConvertDAppWalletBlockchainAddress is a dapp wallet's blockchain address
type ConvertDAppWalletBlockchainAddress struct {
	Network  string `json:"network"`
	Address  string `json:"address"`
	CohortID string `json:"cohort_id"`
	UserUUID string `json:"user_uuid"`
	Pool     string `json:"pool"`
}

// ConvertPhoneAccount is an account known by its phone number, a MobiKwik wallet or a Bancomat Pay account
type ConvertPhoneAccount struct {
	PhoneNumber string `json:"phone_number"`
}

// ConvertDenebUPI is an Indian UPI account
type ConvertDenebUPI struct {
	VPAID             string          `json:"vpa_id"`
	CustomerFirstName string          `json:"customer_first_name"`
	CustomerLastName  string          `json:"customer_last_name"`
	Email             string          `json:"email"`
	PhoneNumber       string          `json:"phone_number"`
	PAN               string          `json:"pan"`
	Address           *ConvertAddress `json:"address"`
}

// ConvertBankAccount is a US bank account paid by ACH
type ConvertBankAccount struct {
	CustomerAccountType   string `json:"customer_account_type"`
	CustomerAccountNumber string `json:"customer_account_number"`
	CustomerRoutingNumber string `json:"customer_routing_number"`
	CustomerName          string `json:"customer_name"`
	// Currency is USD unless set
	Currency      currency.Code `json:"currency"`
	BankName      string        `json:"bank_name"`
	ACHEntryClass string        `json:"ach_entry_class"`
	// VerificationMethod is cdv unless set
	VerificationMethod       string                             `json:"verification_method"`
	CDVAmount1               ConvertMoney                       `json:"cdv_amount_1"`
	CDVAmount2               ConvertMoney                       `json:"cdv_amount_2"`
	ACHInvalidAccount        bool                               `json:"ach_invalid_account"`
	ACHBadAccount            bool                               `json:"ach_bad_account"`
	ReceivedNonNSFReturn     bool                               `json:"received_non_nsf_return"`
	GIACTInfo                *ConvertGIACTInfo                  `json:"giactInfo"`
	VendorVerificationStatus *ConvertVendorVerification         `json:"vendor_verification_status"`
	PlaidVerificationInfo    *ConvertPlaidInfo                  `json:"plaid_verification_info"`
	StripeVerificationInfo   *ConvertStripeFinancialConnections `json:"stripe_verification_info"`
}

// ConvertGIACTInfo is a bank account's GIACT verification
type ConvertGIACTInfo struct {
	GIACTItemID                 string    `json:"giact_item_id"`
	GIACTBankResult             string    `json:"giact_bank_result"`
	GIACTCustomerResult         string    `json:"giact_customer_result"`
	GIACTStatus                 string    `json:"giact_status"`
	GIACTLastUpdatedAt          time.Time `json:"giact_last_updated_at"`
	GIACTAccountAddedDate       time.Time `json:"giact_account_added_date"`
	GIACTAccountLastUpdatedDate time.Time `json:"giact_account_last_updated_date"`
	// GIACTAccountAddedRange and GIACTAccountLastUpdatedRange are how recently GIACT added and updated the account,
	// such as "Less than 7 days as of Mar 18, 2026 UTC"
	GIACTAccountAddedRange       string `json:"giact_account_added_range"`
	GIACTAccountLastUpdatedRange string `json:"giact_account_last_updated_range"`
}

// ConvertVendorVerification is a bank account's verification with its vendor
type ConvertVendorVerification struct {
	InstantAccountVerificationStatus string `json:"iav_status"`
	MicroDepositStatus               string `json:"md_status"`
	EmailVerificationStatus          string `json:"email_verification_status"`
}

// ConvertPlaidInfo is a bank account linked through Plaid
type ConvertPlaidInfo struct {
	PlaidBankName       string       `json:"plaid_bank_name"`
	PlaidAccessToken    string       `json:"plaid_access_token"`
	PlaidItemID         string       `json:"plaid_item_id"`
	PlaidOldAccessToken string       `json:"plaid_old_access_token"`
	PlaidResponse       string       `json:"plaid_response"`
	PlaidInfo           string       `json:"plaid_info"`
	PlaidAccountID      string       `json:"plaid_account_id"`
	PlaidTestAccount    bool         `json:"plaid_test_account"`
	PlaidLinkAccount    bool         `json:"plaid_link_account"`
	PlaidLinkBankName   string       `json:"plaid_link_bank_name"`
	Balance             ConvertMoney `json:"balance"`
	BalanceUpdatedAt    time.Time    `json:"balance_updated_at"`
	PlaidNameCheck      bool         `json:"plaid_name_check"`
	PlaidPublicToken    string       `json:"plaid_public_token"`
	AccountNumberMask   string       `json:"account_number_mask"`
	// PlaidPersistentAccountID identifies the account across re-links, at banks that tokenise account numbers
	PlaidPersistentAccountID string `json:"plaid_persistent_account_id"`
	// IsTokenisedAccountNumber is set for a tokenised account number, which only PlaidPersistentAccountID matches
	IsTokenisedAccountNumber bool `json:"is_tokenized_account_number"`
	// IsPlaidSupported is whether Plaid supports the account's routing number
	IsPlaidSupported bool `json:"is_plaid_supported"`
}

// ConvertStripeFinancialConnections is a bank account linked through Stripe Financial Connections
type ConvertStripeFinancialConnections struct {
	StripeResponse               string       `json:"stripe_response"`
	StripeInfo                   string       `json:"stripe_info"`
	StripeFinancialConnectionsID string       `json:"stripe_financial_connections_id"`
	StripePaymentMethodID        string       `json:"stripe_payment_method_id"`
	StripeTestAccount            bool         `json:"stripe_test_account"`
	StripeLinkAccount            bool         `json:"stripe_link_account"`
	StripeLinkBankName           string       `json:"stripe_link_bank_name"`
	StripeInstitutionID          string       `json:"stripe_institution_id"`
	Balance                      ConvertMoney `json:"balance"`
	BalanceUpdatedAt             time.Time    `json:"balance_updated_at"`
	StripeNameCheck              bool         `json:"stripe_name_check"`
}

// ConvertNetworkAddress is an address on a network, a hot wallet or the contract an identity contract call goes to
type ConvertNetworkAddress struct {
	Network string `json:"network"`
	Address string `json:"address"`
}

// ConvertDenebIMPS is an Indian bank account paid by IMPS
type ConvertDenebIMPS struct {
	IFSCCode          string          `json:"ifsc_code"`
	AccountNumber     string          `json:"account_number"`
	CustomerFirstName string          `json:"customer_first_name"`
	CustomerLastName  string          `json:"customer_last_name"`
	Email             string          `json:"email"`
	PhoneNumber       string          `json:"phone_number"`
	PAN               string          `json:"pan"`
	Address           *ConvertAddress `json:"address"`
}

// ConvertAllocation is an allocation of funds, in legs
type ConvertAllocation struct {
	ID       string                 `json:"id"`
	Legs     []ConvertAllocationLeg `json:"legs"`
	IsNetted bool                   `json:"is_netted"`
}

// ConvertAllocationLeg is an allocation's leg, in fund movements
type ConvertAllocationLeg struct {
	ID        string                `json:"id"`
	Movements []ConvertFundMovement `json:"movements"`
	IsNetted  bool                  `json:"is_netted"`
}

// ConvertFundMovement is a movement of funds between ledger accounts
type ConvertFundMovement struct {
	ID                 string                `json:"id"`
	SourceAccount      *ConvertLedgerAccount `json:"source_account"`
	DestinationAccount *ConvertLedgerAccount `json:"destination_account"`
	SourceCurrency     currency.Code         `json:"source_currency"`
	TargetCurrency     currency.Code         `json:"target_currency"`
	Amount             ConvertMoney          `json:"amount"`
	Fee                ConvertMoney          `json:"fee"`
}

// ConvertLiquidityPool is a liquidity pool, which manages Coinbase's liquidity
type ConvertLiquidityPool struct {
	Network     string        `json:"network"`
	Pool        string        `json:"pool"`
	Currency    currency.Code `json:"currency"`
	AccountID   string        `json:"account_id"`
	FromAddress string        `json:"from_address"`
}

// ConvertDirectDeposit is a direct deposit account
type ConvertDirectDeposit struct {
	DirectDepositAccount string `json:"direct_deposit_account"`
}

// ConvertSEPAV2 is a bank account paid by SEPA transfer or direct debit, in the SEPA_V2 form
type ConvertSEPAV2 struct {
	Account           *ConvertSEPAAccountDetail `json:"account"`
	CustomerFirstName string                    `json:"customer_first_name"`
	CustomerLastName  string                    `json:"customer_last_name"`
	Email             string                    `json:"email"`
	PhoneNumber       string                    `json:"phone_number"`
	// CustomerCountry is the customer's ISO 3166-1 alpha-2 country code
	CustomerCountry string `json:"customer_country"`
	// Address is deprecated
	Address             *ConvertAddress `json:"address"`
	SupportsOpenBanking bool            `json:"supports_open_banking"`
	// BankingCircleToken is the mandate SEPA direct debits are paid under
	BankingCircleToken *ConvertBankingCircleToken `json:"banking_circle_token"`
	Institution        *ConvertSEPAInstitution    `json:"institution"`
	Currency           currency.Code              `json:"currency"`
	// FiatID is only used by Coinbase Prime
	FiatID string `json:"fiat_id"`
	// DirectDebitIneligible is set when the bank account cannot pay by direct debit
	DirectDebitIneligible bool                `json:"direct_debit_ineligible"`
	OpenBanking           *ConvertOpenBanking `json:"open_banking_details"`
}

// ConvertSEPAAccountDetail is a SEPA bank account and its holder
type ConvertSEPAAccountDetail struct {
	LegalName      string `json:"legal_name"`
	IBAN           string `json:"iban"`
	BBAN           string `json:"bban"`
	AccountAddress string `json:"account_address"`
}

// ConvertBankingCircleToken is a Banking Circle mandate, which SEPA direct debits are paid under
type ConvertBankingCircleToken struct {
	// MandateID is Banking Circle's 32 character ID of the mandate
	MandateID             string `json:"mandate_id"`
	Active                bool   `json:"active"`
	BalanceCheckSupported bool   `json:"balance_check_supported"`
	// HostedPageID is the hosted page the mandate was created on
	HostedPageID string `json:"hosted_page_id"`
}

// ConvertZepto is an Australian bank account paid through Zepto
type ConvertZepto struct {
	Account     *ConvertZeptoAccount     `json:"account"`
	Institution *ConvertZeptoInstitution `json:"institution"`
}

// ConvertZeptoAccount is a Zepto contact's bank account
type ConvertZeptoAccount struct {
	// ContactID and BankAccountID are Zepto's IDs of the contact and its bank account
	ContactID           string `json:"contact_id"`
	BankAccountID       string `json:"bank_account_id"`
	AccountNumber       string `json:"account_number"`
	CustomerPaymentName string `json:"customer_payment_name"`
}

// ConvertZeptoInstitution is a Zepto bank account's branch
type ConvertZeptoInstitution struct {
	BranchCode string `json:"branch_code"`
}

// ConvertPixEbanx is a Brazilian Pix account paid through EBANX
type ConvertPixEbanx struct {
	PaymentMethodID string                     `json:"payment_method_id"`
	UserUUID        string                     `json:"user_uuid"`
	Deposit         *ConvertPixEbanxDeposit    `json:"deposit"`
	Withdrawal      *ConvertPixEbanxWithdrawal `json:"withdrawal"`
}

// ConvertPixEbanxDeposit is the account a Pix deposit comes from
type ConvertPixEbanxDeposit struct {
	TransactionID       string `json:"transaction_id"`
	AccountID           string `json:"account_id"`
	AccountNumber       string `json:"account_number"`
	AccountType         string `json:"account_type"`
	BankCode            string `json:"bank_code"`
	BankName            string `json:"bank_name"`
	BranchNumber        string `json:"branch_number"`
	CustomerPaymentName string `json:"customer_payment_name"`
	// SenderDocument is the sender's tax ID, a CPF for a person or a CNPJ for a business
	SenderDocument string `json:"sender_document"`
}

// ConvertPixEbanxWithdrawal is the account a Pix withdrawal goes to
type ConvertPixEbanxWithdrawal struct {
	AccountNumber       string `json:"account_number"`
	AccountType         string `json:"account_type"`
	BankCode            string `json:"bank_code"`
	BranchNumber        string `json:"branch_number"`
	PixKey              string `json:"pix_key"`
	CustomerPaymentName string `json:"customer_payment_name"`
	// SenderDocument is the sender's tax ID, a CPF for a person or a CNPJ for a business
	SenderDocument string `json:"sender_document"`
	BankName       string `json:"bank_name"`
}

// ConvertSignet is a Signet wallet
type ConvertSignet struct {
	SignetWalletID string `json:"signet_wallet_id"`
}

// ConvertDerivativeSettlement is a US derivatives end of day settlement's fund movements
type ConvertDerivativeSettlement struct {
	AccountSettlements []ConvertDerivativeAccountSettlement `json:"account_settlements"`
	EquityReset        *ConvertDerivativeEquityReset        `json:"equity_reset"`
}

// ConvertDerivativeAccountSettlement is a settlement's movement of funds between ledger accounts, which can replace a
// hold with a new one
type ConvertDerivativeAccountSettlement struct {
	Amount                   ConvertMoney               `json:"amount"`
	SourceLedgerAccount      *ConvertLedgerAccount      `json:"source_ledger_account"`
	SourceLedgerNamedAccount *ConvertLedgerNamedAccount `json:"source_ledger_named_account"`
	TargetLedgerAccount      *ConvertLedgerAccount      `json:"target_ledger_account"`
	TargetLedgerNamedAccount *ConvertLedgerNamedAccount `json:"target_ledger_named_account"`
	// HoldIDToReplace is deprecated; ExistingHoldID replaces it
	HoldIDToReplace string `json:"hold_id_to_replace"`
	// NewHoldID and NewHoldAmount are the hold that replaces the existing one
	NewHoldID      string       `json:"new_hold_id"`
	NewHoldAmount  ConvertMoney `json:"new_hold_amount"`
	ExistingHoldID string       `json:"existing_hold_id"`
}

// ConvertDerivativeEquityReset resets a US derivatives equity account
type ConvertDerivativeEquityReset struct {
	Amount        ConvertMoney          `json:"amount"`
	EquityAccount *ConvertLedgerAccount `json:"equity_account"`
}

// ConvertUser is a Coinbase user
type ConvertUser struct {
	UserUUID string `json:"user_uuid"`
}

// ConvertSingaporeFAST is a Singapore bank account paid by FAST transfer
type ConvertSingaporeFAST struct {
	Account     *ConvertAccountHolder            `json:"account"`
	Institution *ConvertSingaporeFASTInstitution `json:"institution"`
	// DirectDebitAuthorisationState is STATE_PENDING, STATE_AUTHORIZED, STATE_REJECTED or STATE_EXPIRED
	DirectDebitAuthorisationState string `json:"direct_debit_auth_state"`
	DirectDebitToken              string `json:"direct_debit_token"`
}

// ConvertAccountHolder is a bank account's number and its holder's name, of a Singapore FAST or Mexican SPEI account
type ConvertAccountHolder struct {
	CustomerName  string `json:"customer_name"`
	AccountNumber string `json:"account_number"`
}

// ConvertSingaporeFASTInstitution is a bank FAST transfers pay
type ConvertSingaporeFASTInstitution struct {
	// BankCode is the bank's SWIFT code, such as DBSSGSG
	BankCode string `json:"bank_code"`
}

// ConvertInterac is a Canadian bank account paid by Interac
type ConvertInterac struct {
	Account                *ConvertInteracAccount `json:"account"`
	PaymentMethodServiceID string                 `json:"pmsvc_id"`
	WithdrawalOnly         bool                   `json:"withdrawal_only"`
	CoinbaseEmail          string                 `json:"coinbase_email"`
}

// ConvertInteracAccount is an Interac bank account
type ConvertInteracAccount struct {
	AccountName string `json:"account_name"`
	// InstitutionNumber is the bank's 3 digit number, TransitNumber the branch's 5 digit number and AccountNumber the
	// account's 7 digit number
	InstitutionNumber string `json:"institution_number"`
	TransitNumber     string `json:"transit_number"`
	AccountNumber     string `json:"account_number"`
}

// ConvertIntraBank is a bank account paid by a transfer within its bank
type ConvertIntraBank struct {
	Currency      currency.Code `json:"currency"`
	AccountNumber string        `json:"account_number"`
	RoutingNumber string        `json:"routing_number"`
	CustomerName  string        `json:"customer_name"`
	FiatID        string        `json:"fiat_id"`
}

// ConvertCBIT is a bank account paid over CBIT
type ConvertCBIT struct {
	CBITWalletAddress      string        `json:"cbit_wallet_address"`
	CustomersBankAccountID string        `json:"customers_bank_account_id"`
	Currency               currency.Code `json:"currency"`
}

// ConvertBankRedirect is a European bank account paid by iDEAL or Sofort
type ConvertBankRedirect struct {
	Currency            currency.Code `json:"currency"`
	IBAN                string        `json:"iban"`
	BIC                 string        `json:"bic"`
	BankName            string        `json:"bank_name"`
	CustomerPaymentName string        `json:"customer_payment_name"`
	CustomerCountryCode string        `json:"customer_country_code"`
}

// ConvertSingaporePayNow is a Singapore PayNow account
type ConvertSingaporePayNow struct {
	// IdentifierType is TYPE_NRIC or TYPE_PHONE
	IdentifierType string `json:"identifier_type"`
	Identifier     string `json:"identifier"`
	CustomerName   string `json:"customer_name"`
}

// ConvertCheckoutPaymentLink is a Checkout.com payment link
type ConvertCheckoutPaymentLink struct {
	PaymentLinkID string `json:"payment_link_id"`
}

// ConvertContact is an email address or phone number funds are sent to
type ConvertContact struct {
	Value string `json:"value"`
}

// ConvertVendorPayment is a vendor's payment, which only Coinbase Pay guest checkout makes
type ConvertVendorPayment struct {
	VendorName      string `json:"vendor_name"`
	VendorPaymentID string `json:"vendor_payment_id"`
}

// ConvertCTN is a CTN payment method
type ConvertCTN struct {
	ID string `json:"id"`
}

// ConvertBancomatPay is an Italian Bancomat Pay account
type ConvertBancomatPay struct {
	CustomerName string               `json:"customer_name"`
	Account      *ConvertPhoneAccount `json:"account"`
}

// ConvertNovaAccount is a Nova account in a pool
type ConvertNovaAccount struct {
	Network               string `json:"network"`
	NovaAccountID         string `json:"nova_account_id"`
	PoolName              string `json:"pool_name"`
	AccountIdempotencyKey string `json:"account_idempotency_key"`
}

// ConvertTransferPointer refers to a transfer by its idempotency key
type ConvertTransferPointer struct {
	IdempotencyKey string `json:"idem"`
}

// ConvertEFT is a Canadian bank account paid by EFT
type ConvertEFT struct {
	Account *ConvertEFTAccount `json:"account"`
}

// ConvertEFTAccount is an EFT bank account
type ConvertEFTAccount struct {
	AccountName        string `json:"account_name"`
	AccountPhoneNumber string `json:"account_phone_number"`
	AccountEmail       string `json:"account_email"`
	// InstitutionNumber is the bank's 3 digit number, TransitNumber the branch's 5 digit number and AccountNumber the
	// account's 7 digit number
	InstitutionNumber string `json:"institution_number"`
	TransitNumber     string `json:"transit_number"`
	AccountNumber     string `json:"account_number"`
	ProviderID        string `json:"provider_id"`
	ProviderName      string `json:"provider_name"`
	// TrustlyAccountTokenStatus is TRUSTLY_TOKEN_STATUS_ACTIVE or TRUSTLY_TOKEN_STATUS_RELINK_NEEDED
	TrustlyAccountTokenStatus string `json:"trustly_account_token_status"`
	AccountType               string `json:"account_type"`
}

// ConvertWallaceAccount is a Wallace account in a pool
type ConvertWallaceAccount struct {
	WallaceAccountID string `json:"wallace_account_id"`
	PoolName         string `json:"pool_name"`
}

// ConvertManualDeposit is a manual fiat deposit's settlement account
type ConvertManualDeposit struct {
	SettlementBankName      string `json:"settlement_bank_name"`
	SettlementAccountNumber string `json:"settlement_account_number"`
	Reference               string `json:"reference"`
}

// ConvertArgentineBankAccount is an Argentine bank account
type ConvertArgentineBankAccount struct {
	TaxID string `json:"tax_id"`
	CBU   string `json:"cbu"`
}

// ConvertRepresentment is the debit of a won card chargeback representment
type ConvertRepresentment struct {
	Currency currency.Code `json:"currency"`
}

// ConvertBankingCircleNow is a bank account paid through Banking Circle Now
type ConvertBankingCircleNow struct {
	IBAN                string        `json:"iban"`
	Currency            currency.Code `json:"currency"`
	CustomerPaymentName string        `json:"customer_payment_name"`
}

// ConvertTrustly is a bank account paid through Trustly
type ConvertTrustly struct {
	Country              string `json:"country"`
	IBAN                 string `json:"iban"`
	AccountHolder        string `json:"account_holder"`
	BankCode             string `json:"bank_code"`
	AccountNumber        string `json:"account_number"`
	PartialAccountNumber string `json:"partial_account_number"`
	BankName             string `json:"bank_name"`
	Email                string `json:"email"`
}

// ConvertBLIK is a Polish BLIK account
type ConvertBLIK struct {
	Email         string `json:"email"`
	Country       string `json:"country"`
	AccountHolder string `json:"account_holder"`
}

// ConvertMBWay is a Portuguese MB Way account
type ConvertMBWay struct {
	AccountHolder string `json:"account_holder"`
	Country       string `json:"country"`
	// PhoneNumber is the country code, a # and the rest of the number
	PhoneNumber string `json:"phone_number"`
}

// ConvertPix is a Brazilian Pix account
type ConvertPix struct {
	AccountNumber string `json:"account_number"`
	AccountType   string `json:"account_type"`
	BankCode      string `json:"bank_code"`
	BankName      string `json:"bank_name"`
	// BranchNumber is deprecated; BranchCode replaces it
	BranchNumber        string `json:"branch_number"`
	CustomerPaymentName string `json:"customer_payment_name"`
	// SenderDocument is the sender's tax ID, a CPF for a person or a CNPJ for a business
	SenderDocument string `json:"sender_document"`
	// PixKey is only set for a withdrawal
	PixKey string `json:"pix_key"`
	// PixAccountType is ACCOUNT_TYPE_CHECKING, ACCOUNT_TYPE_SALARY, ACCOUNT_TYPE_SAVINGS or ACCOUNT_TYPE_TRANSACTION
	PixAccountType string `json:"pix_account_type"`
	BranchCode     string `json:"branch_code"`
}

// ConvertMagicWallet is a user's named wallet on a network
type ConvertMagicWallet struct {
	UserUUID string `json:"user_uuid"`
	// NetworkName is the blockchain and network, such as base-mainnet
	NetworkName string `json:"network_name"`
	WalletName  string `json:"wallet_name"`
}

// ConvertSamsungPay is a card paid through Samsung Pay
type ConvertSamsungPay struct {
	SamsungPay *ConvertDigitalWallet `json:"samsung_pay"`
	UserID     string                `json:"user_id"`
	// PostalCode and the fields after it are set once an order is submitted
	PostalCode     string          `json:"postal_code"`
	CustomerName   string          `json:"customer_name"`
	Address        *ConvertAddress `json:"address"`
	SixDigitBIN    string          `json:"six_digit_bin"`
	LastFour       string          `json:"last_four"`
	IssuingCountry string          `json:"issuing_country"`
	IssuingBank    string          `json:"issuing_bank"`
	ProductID      string          `json:"product_id"`
	// CustomerDateOfBirth is in YYYY-MM-DD form
	CustomerDateOfBirth string        `json:"customer_dob"`
	Currency            currency.Code `json:"currency"`
	BillingAddressID    string        `json:"billing_address_id"`
	// Scheme is UNKNOWN, VISA or MASTERCARD, and Prepaid and Debit UNCERTAIN, TRUE or FALSE
	Scheme  string `json:"scheme"`
	Prepaid string `json:"prepaid"`
	Debit   string `json:"debit"`
}

// ConvertCADWire is a Canadian bank account paid by wire
type ConvertCADWire struct {
	AccountNumber       string `json:"account_number"`
	CustomerPaymentName string `json:"customer_payment_name"`
	AccountAddress      string `json:"account_address"`
	// InstitutionNumber is the bank's 3 digit number and TransitNumber the branch's 5 digit number
	InstitutionNumber string `json:"institution_number"`
	TransitNumber     string `json:"transit_number"`
	BankAddress1      string `json:"bank_address1"`
	InstitutionName   string `json:"institution_name"`
}

// ConvertWABIntraBank is a WAB account paid by a transfer within WAB
type ConvertWABIntraBank struct {
	AccountID    string `json:"account_id"`
	CustomerName string `json:"customer_name"`
}

// ConvertIndiaBankAccount is an Indian bank account
type ConvertIndiaBankAccount struct {
	// AccountNumber is 9 to 18 characters
	AccountNumber string `json:"account_number"`
	// IFSCCode identifies the bank branch
	IFSCCode          string `json:"ifsc_code"`
	BankName          string `json:"bank_name"`
	CustomerFirstName string `json:"customer_first_name"`
	CustomerLastName  string `json:"customer_last_name"`
	Email             string `json:"email"`
	// PhoneNumber is the 10 digits after +91
	PhoneNumber string `json:"phone_number"`
	// PANNumber is the holder's 10 character Permanent Account Number
	PANNumber           string `json:"pan_number"`
	CustomerPaymentName string `json:"customer_payment_name"`
}

// ConvertJPMorganKinexys is a JPMorgan blockchain deposit account paid over Kinexys
type ConvertJPMorganKinexys struct {
	AccountNumber string `json:"account_number"`
	// BIC is the JPMorgan branch that serves the account
	BIC string `json:"bic"`
}

// ConvertSIC is a Swiss bank account paid over SIC
type ConvertSIC struct {
	LegalName               string `json:"legal_name"`
	IBAN                    string `json:"iban"`
	SWIFT                   string `json:"swift"`
	CustomerPaymentAddress1 string `json:"customer_payment_address1"`
	CustomerPaymentAddress2 string `json:"customer_payment_address2"`
	CustomerPaymentAddress3 string `json:"customer_payment_address3"`
	BankName                string `json:"bank_name"`
	BankAddress1            string `json:"bank_address1"`
	BankAddress2            string `json:"bank_address2"`
	BankAddress3            string `json:"bank_address3"`
	BankCountryCode         string `json:"bank_country_code"`
}

// ConvertUnitPrice is a convert trade's prices; a fiat trade's TargetToFiat may equal its TargetToSource, and only a
// crypto to crypto trade sets TargetToSource and SourceToFiat
type ConvertUnitPrice struct {
	TargetToFiat   *ConvertScaledAmount `json:"target_to_fiat"`
	TargetToSource *ConvertScaledAmount `json:"target_to_source"`
	SourceToFiat   *ConvertScaledAmount `json:"source_to_fiat"`
}

// ConvertScaledAmount is a price and its scale
type ConvertScaledAmount struct {
	Amount Amount `json:"amount"`
	Scale  uint64 `json:"scale"`
}

// ConvertUserWarning is a warning about a convert trade for its user
type ConvertUserWarning struct {
	ID      string                     `json:"id"`
	Link    *ConvertLink               `json:"link"`
	Context *ConvertUserWarningContext `json:"context"`
	Code    string                     `json:"code"`
	Message string                     `json:"message"`
}

// ConvertUserWarningContext is a warning's details, which some payment warnings send
type ConvertUserWarningContext struct {
	Details  []string `json:"details"`
	Title    string   `json:"title"`
	LinkText string   `json:"link_text"`
}

// ConvertCancellationReason is why Coinbase could not make a convert trade, such as a blocked card
type ConvertCancellationReason struct {
	Message string `json:"message"`
	Code    string `json:"code"`
	// ErrorCode is one of the ERROR_CODES values, such as ERROR_CODES_INSUFFICIENT_BALANCE
	ErrorCode string `json:"error_code"`
	// CallToAction is the ERROR_CTA value naming what the user can do about it, such as ERROR_CTA_SWITCH_PAYMENT
	CallToAction  string                       `json:"error_cta"`
	ErrorMetadata *ConvertCancellationMetadata `json:"error_metadata"`
	Title         string                       `json:"title"`
}

// ConvertCancellationMetadata is a cancellation's details, such as the limit an amount exceeded and a payment method to
// use instead
type ConvertCancellationMetadata struct {
	LimitAmount                          Amount `json:"limit_amount"`
	SuggestedPaymentMethodUUID           string `json:"suggested_pm_uuid"`
	SuggestedPaymentMethodObfuscatedName string `json:"suggested_pm_obfuscated_name"`
	// SuggestedPaymentMethodCategory is ASSET, ON_PLATFORM_FIAT or OFF_PLATFORM_FIAT
	SuggestedPaymentMethodCategory string `json:"suggested_pm_category"`
}

// ConvertSubscriptionInfo is a Coinbase One subscription's effect on a convert trade
type ConvertSubscriptionInfo struct {
	FreeTradingResetDate                       time.Time `json:"free_trading_reset_date"`
	UsedZeroFeeTrading                         Amount    `json:"used_zero_fee_trading"`
	RemainingFreeTradingVolume                 Amount    `json:"remaining_free_trading_volume"`
	MaximumFreeTradingVolume                   Amount    `json:"max_free_trading_volume"`
	HasBenefitCap                              bool      `json:"has_benefit_cap"`
	AppliedSubscriptionBenefit                 bool      `json:"applied_subscription_benefit"`
	FeeWithoutSubscriptionBenefit              Amount    `json:"fee_without_subscription_benefit"`
	PaymentMethodFeeWithoutSubscriptionBenefit Amount    `json:"payment_method_fee_without_subscription_benefit"`
}

// ConvertTax is a tax a convert trade pays
type ConvertTax struct {
	Name   string `json:"name"`
	Amount Amount `json:"amount"`
}

// ConvertTradeIncentive is a trade incentive's effect on a convert trade
type ConvertTradeIncentive struct {
	AppliedIncentive bool   `json:"applied_incentive"`
	UserIncentiveID  string `json:"user_incentive_id"`
	// CodeValue is the promo code applied
	CodeValue           string    `json:"code_val"`
	EndsAt              time.Time `json:"ends_at"`
	FeeWithoutIncentive Amount    `json:"fee_without_incentive"`
	Redeemed            bool      `json:"redeemed"`
}
