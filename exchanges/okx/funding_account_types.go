package okx

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// FundingCurrenciesRequest holds the filters for GetFundingCurrencies
type FundingCurrenciesRequest struct {
	Currencies currency.Currencies
}

// FundingCurrencyResponse holds a currency's deposit and withdrawal details on one chain
type FundingCurrencyResponse struct {
	Currency                             currency.Code `json:"ccy"`
	Name                                 string        `json:"name"`
	LogoLink                             string        `json:"logoLink"`
	Chain                                string        `json:"chain"`
	ContractAddress                      string        `json:"ctAddr"`
	CanDeposit                           bool          `json:"canDep"`
	CanWithdraw                          bool          `json:"canWd"`
	CanInternalTransfer                  bool          `json:"canInternal"`
	DepositEstimatedOpenTime             types.Time    `json:"depEstOpenTime"`
	WithdrawalEstimatedOpenTime          types.Time    `json:"wdEstOpenTime"`
	MinimumDeposit                       types.Number  `json:"minDep"`
	MinimumWithdrawal                    types.Number  `json:"minWd"`
	MinimumInternalTransfer              types.Number  `json:"minInternal"`
	MaximumWithdrawal                    types.Number  `json:"maxWd"`
	WithdrawalTickSize                   types.Number  `json:"wdTickSz"`
	WithdrawalQuota                      types.Number  `json:"wdQuota"`
	UsedWithdrawalQuota                  types.Number  `json:"usedWdQuota"`
	Fee                                  types.Number  `json:"fee"`
	MinimumFee                           types.Number  `json:"minFee"`          // Deprecated by OKX
	MaximumFee                           types.Number  `json:"maxFee"`          // Deprecated by OKX
	MinimumFeeForContractAddress         types.Number  `json:"minFeeForCtAddr"` // Deprecated by OKX
	MaximumFeeForContractAddress         types.Number  `json:"maxFeeForCtAddr"` // Deprecated by OKX
	BurningFeeRate                       types.Number  `json:"burningFeeRate"`
	MainNet                              bool          `json:"mainNet"`
	NeedTag                              bool          `json:"needTag"`
	MinimumDepositArrivalConfirmations   types.Number  `json:"minDepArrivalConfirm"`
	MinimumWithdrawalUnlockConfirmations types.Number  `json:"minWdUnlockConfirm"`
	DepositQuotaFixed                    types.Number  `json:"depQuotaFixed"`
	UsedDepositQuotaFixed                types.Number  `json:"usedDepQuotaFixed"`
	DepositQuotaDailyLayer2              types.Number  `json:"depQuoteDailyLayer2"`
}

// FundingBalancesRequest holds the filters for GetFundingBalances
type FundingBalancesRequest struct {
	Currencies currency.Currencies
}

// FundingBalanceResponse holds a currency's funding account balance
type FundingBalanceResponse struct {
	Currency         currency.Code `json:"ccy"`
	Balance          types.Number  `json:"bal"`
	FrozenBalance    types.Number  `json:"frozenBal"`
	AvailableBalance types.Number  `json:"availBal"`
}

// NonTradableAssetsRequest holds the filters for GetNonTradableAssets
type NonTradableAssetsRequest struct {
	Currencies currency.Currencies
}

// NonTradableAssetResponse holds a non-tradable asset's balance and withdrawal details
type NonTradableAssetResponse struct {
	Currency           currency.Code `json:"ccy"`
	Name               string        `json:"name"`
	LogoLink           string        `json:"logoLink"`
	Balance            types.Number  `json:"bal"`
	CanWithdraw        bool          `json:"canWd"`
	Chain              string        `json:"chain"`
	MinimumWithdrawal  types.Number  `json:"minWd"`
	WithdrawAll        bool          `json:"wdAll"`
	Fee                types.Number  `json:"fee"`
	FeeCurrency        currency.Code `json:"feeCcy"`
	BurningFeeRate     types.Number  `json:"burningFeeRate"`
	ContractAddress    string        `json:"ctAddr"`
	WithdrawalTickSize types.Number  `json:"wdTickSz"`
	NeedTag            bool          `json:"needTag"`
}

// AccountAssetValuationRequest holds the valuation unit for GetAccountAssetValuation; OKX values in BTC by default
type AccountAssetValuationRequest struct {
	Currency currency.Code
}

// AccountAssetValuationResponse holds the account's total asset valuation
type AccountAssetValuationResponse struct {
	TotalBalance types.Number          `json:"totalBal"`
	Timestamp    types.Time            `json:"ts"`
	Details      AssetValuationDetails `json:"details"`
}

// AssetValuationDetails holds the asset valuation of each account
type AssetValuationDetails struct {
	Funding types.Number `json:"funding"`
	Trading types.Number `json:"trading"`
	Classic types.Number `json:"classic"` // Deprecated by OKX
	Earn    types.Number `json:"earn"`
}

// FundingTransferRequest holds the parameters for FundingTransfer
type FundingTransferRequest struct {
	TransferType           string        `json:"type,omitempty"`
	Currency               currency.Code `json:"ccy"`
	Amount                 float64       `json:"amt,string"`
	RemittingAccountType   string        `json:"from"`
	BeneficiaryAccountType string        `json:"to"`
	SubAccountName         string        `json:"subAcct,omitempty"`
	LoanTransfer           bool          `json:"loanTrans,omitempty"`
	OmitPositionRisk       bool          `json:"omitPosRisk,string,omitempty"`
	ClientID               string        `json:"clientId,omitempty"`
}

// FundingTransferResponse holds an accepted funds transfer
type FundingTransferResponse struct {
	TransferID             string        `json:"transId"`
	ClientID               string        `json:"clientId"`
	Currency               currency.Code `json:"ccy"`
	RemittingAccountType   string        `json:"from"`
	Amount                 types.Number  `json:"amt"`
	BeneficiaryAccountType string        `json:"to"`
}

// FundsTransferStateRequest holds the parameters for GetFundsTransferState; TransferID is used when both IDs are set
type FundsTransferStateRequest struct {
	TransferID   string
	ClientID     string
	TransferType string
}

// FundsTransferStateResponse holds a funds transfer's state
type FundsTransferStateResponse struct {
	TransferID              string        `json:"transId"`
	ClientID                string        `json:"clientId"`
	Currency                currency.Code `json:"ccy"`
	Amount                  types.Number  `json:"amt"`
	TransferType            string        `json:"type"`
	RemittingAccountType    string        `json:"from"`
	BeneficiaryAccountType  string        `json:"to"`
	SubAccountName          string        `json:"subAcct"`
	InstrumentID            string        `json:"instId"`   // Deprecated by OKX
	BeneficiaryInstrumentID string        `json:"toInstId"` // Deprecated by OKX
	State                   string        `json:"state"`
}

// AssetBillsDetailsRequest holds the filters for GetAssetBillsDetails
type AssetBillsDetailsRequest struct {
	Currency       currency.Code
	BillType       string
	ThirdPartyType string
	ClientID       string
	StartTime      time.Time
	EndTime        time.Time
	Limit          uint64
}

// AssetBillsHistoryRequest holds the filters for GetAssetBillsHistory; AfterBillID pages by bill ID in place of EndTime
type AssetBillsHistoryRequest struct {
	Currency       currency.Code
	BillType       string
	ThirdPartyType string
	ClientID       string
	StartTime      time.Time
	EndTime        time.Time
	AfterBillID    string
	Limit          uint64
}

// AssetBillResponse holds a funding account bill
type AssetBillResponse struct {
	BillID        string        `json:"billId"`
	Currency      currency.Code `json:"ccy"`
	ClientID      string        `json:"clientId"`
	BalanceChange types.Number  `json:"balChg"`
	Balance       types.Number  `json:"bal"`
	BillType      string        `json:"type"`
	Notes         string        `json:"notes"`
	Timestamp     types.Time    `json:"ts"`
}

// DepositAddressResponse holds a deposit address of a currency
type DepositAddressResponse struct {
	Address                string            `json:"addr"`
	Tag                    string            `json:"tag"`
	Memo                   string            `json:"memo"`
	PaymentID              string            `json:"pmtId"`
	AddressAttachment      map[string]string `json:"addrEx"`
	Currency               currency.Code     `json:"ccy"`
	Chain                  string            `json:"chain"`
	BeneficiaryAccountType string            `json:"to"`
	VerifiedName           string            `json:"verifiedName"`
	Selected               bool              `json:"selected"`
	ContractAddress        string            `json:"ctAddr"`
}

// DepositHistoryRequest holds the filters for GetCurrencyDepositHistory
type DepositHistoryRequest struct {
	Currency         currency.Code
	DepositID        string
	FromWithdrawalID string
	TransactionID    string
	DepositType      string
	State            string
	StartTime        time.Time
	EndTime          time.Time
	Limit            uint64
}

// DepositHistoryResponse holds a deposit record
type DepositHistoryResponse struct {
	Currency                        currency.Code `json:"ccy"`
	Chain                           string        `json:"chain"`
	Amount                          types.Number  `json:"amt"`
	From                            string        `json:"from"`
	AreaCodeFrom                    string        `json:"areaCodeFrom"`
	To                              string        `json:"to"`
	TransactionID                   string        `json:"txId"`
	Timestamp                       types.Time    `json:"ts"`
	State                           string        `json:"state"`
	DepositID                       string        `json:"depId"`
	FromWithdrawalID                string        `json:"fromWdId"`
	ActualDepositBlockConfirmations types.Number  `json:"actualDepBlkConfirm"`
	XStockMultiplier                types.Number  `json:"xstockMultiplier"`
	XStockShareAmount               types.Number  `json:"xstockShareAmount"`
	XStockShareName                 string        `json:"xstockShareName"`
}

// LockedTravelRuleDepositResponse holds a deposit locked under the Travel Rule
type LockedTravelRuleDepositResponse struct {
	DepositID     string        `json:"depId"`
	TransactionID string        `json:"txId"`
	Address       string        `json:"addr"`
	Currency      currency.Code `json:"ccy"`
	Chain         string        `json:"chain"`
	Amount        types.Number  `json:"amt"`
	Timestamp     types.Time    `json:"ts"`
}

// UnlockTravelRuleDepositRequest holds the originator information that unlocks a deposit locked under the Travel Rule
type UnlockTravelRuleDepositRequest struct {
	DepositID     string                   `json:"depId"`
	Questionnaire *TravelRuleQuestionnaire `json:"questionnaire"`
}

// TravelRuleQuestionnaire holds a deposit originator's information; the keys OKX requires depend on the user's entity
type TravelRuleQuestionnaire struct {
	PersonType                  string `json:"personType"`
	WalletType                  string `json:"walletType"`
	ExchangeID                  string `json:"exchId,omitempty"`
	ExchangeName                string `json:"exchName,omitempty"`
	OriginatorFirstName         string `json:"origFirstName,omitempty"`
	OriginatorLastName          string `json:"origLastName,omitempty"`
	OriginatorCompanyName       string `json:"origCompanyName,omitempty"`
	OriginatorCountry           string `json:"origCountry,omitempty"`
	OriginatorGeographicAddress string `json:"origGeoAddress,omitempty"`
	OriginatorDateOfBirth       string `json:"origDob,omitempty"` // A calendar date, YYYY-MM-DD
	OriginatorPlaceOfBirth      string `json:"origPob,omitempty"`
	OriginatorNationality       string `json:"origNationality,omitempty"`
	BrazilClassificationCode    string `json:"brazilClassificationCode,omitempty"`
}

// UnlockTravelRuleDepositResponse holds the outcome of a Travel Rule deposit unlock
type UnlockTravelRuleDepositResponse struct {
	Result bool `json:"result"`
}

// WithdrawalRequest holds the parameters for Withdrawal
type WithdrawalRequest struct {
	Currency             currency.Code                   `json:"ccy"`
	Amount               float64                         `json:"amt,string"`
	Destination          string                          `json:"dest"`
	ToAddress            string                          `json:"toAddr"`
	ToAddressType        string                          `json:"toAddrType,omitempty"`
	Chain                string                          `json:"chain,omitempty"`
	AreaCode             string                          `json:"areaCode,omitempty"`
	RecipientInformation *WithdrawalRecipientInformation `json:"rcvrInfo,omitempty"`
	ClientID             string                          `json:"clientId,omitempty"`
}

// WithdrawalRecipientInformation holds the recipient information some entities need for on-chain withdrawals
type WithdrawalRecipientInformation struct {
	WalletType                 string `json:"walletType"`
	ExchangeID                 string `json:"exchId,omitempty"`
	ReceiverFirstName          string `json:"rcvrFirstName,omitempty"`
	ReceiverLastName           string `json:"rcvrLastName,omitempty"`
	ReceiverCountry            string `json:"rcvrCountry,omitempty"`
	ReceiverCountrySubDivision string `json:"rcvrCountrySubDivision,omitempty"`
	ReceiverTownName           string `json:"rcvrTownName,omitempty"`
	ReceiverStreetName         string `json:"rcvrStreetName,omitempty"`
}

// WithdrawalResponse holds a submitted withdrawal
type WithdrawalResponse struct {
	Currency     currency.Code `json:"ccy"`
	Chain        string        `json:"chain"`
	Amount       types.Number  `json:"amt"`
	WithdrawalID string        `json:"wdId"`
	ClientID     string        `json:"clientId"`
}

// CancelWithdrawalRequest holds the withdrawal CancelWithdrawal cancels
type CancelWithdrawalRequest struct {
	WithdrawalID string `json:"wdId"`
}

// CancelWithdrawalResponse holds a cancelled withdrawal's ID
type CancelWithdrawalResponse struct {
	WithdrawalID string `json:"wdId"`
}

// WithdrawalHistoryRequest holds the filters for GetWithdrawalHistory
type WithdrawalHistoryRequest struct {
	Currency       currency.Code
	WithdrawalID   string
	ClientID       string
	TransactionID  string
	WithdrawalType string
	State          string
	StartTime      time.Time
	EndTime        time.Time
	Limit          uint64
}

// WithdrawalHistoryResponse holds a withdrawal record
type WithdrawalHistoryResponse struct {
	Currency          currency.Code     `json:"ccy"`
	Chain             string            `json:"chain"`
	NonTradableAsset  bool              `json:"nonTradableAsset"`
	Amount            types.Number      `json:"amt"`
	Timestamp         types.Time        `json:"ts"`
	From              string            `json:"from"`
	AreaCodeFrom      string            `json:"areaCodeFrom"`
	To                string            `json:"to"`
	AreaCodeTo        string            `json:"areaCodeTo"`
	ToAddressType     string            `json:"toAddrType"`
	Tag               string            `json:"tag"`
	PaymentID         string            `json:"pmtId"`
	Memo              string            `json:"memo"`
	AddressAttachment map[string]string `json:"addrEx"`
	TransactionID     string            `json:"txId"`
	Fee               types.Number      `json:"fee"`
	FeeCurrency       currency.Code     `json:"feeCcy"`
	State             string            `json:"state"`
	WithdrawalID      string            `json:"wdId"`
	ClientID          string            `json:"clientId"`
	Note              string            `json:"note"`
}

// DepositWithdrawalStatusRequest holds the parameters for GetDepositWithdrawalStatus: a withdrawal ID for a
// withdrawal, or a transaction ID with its currency, destination address and chain for a deposit
type DepositWithdrawalStatusRequest struct {
	WithdrawalID  string
	TransactionID string
	Currency      currency.Code
	ToAddress     string
	Chain         string
}

// DepositWithdrawalStatusResponse holds a deposit's or withdrawal's detailed status
type DepositWithdrawalStatusResponse struct {
	EstimatedCompleteTime EstimatedCompleteTime `json:"estCompleteTime"`
	State                 string                `json:"state"`
	TransactionID         string                `json:"txId"`
	WithdrawalID          string                `json:"wdId"`
}

// ExchangeResponse holds an exchange a withdrawal can name as its recipient's
type ExchangeResponse struct {
	ExchangeName string `json:"exchName"`
	ExchangeID   string `json:"exchId"`
}

// ApplyForMonthlyStatementRequest holds the month ApplyForMonthlyStatement applies for; OKX applies for last month
// when Month is zero
type ApplyForMonthlyStatementRequest struct {
	Month time.Month
}

// MarshalJSON sends the month as the three letter abbreviation OKX expects
func (r *ApplyForMonthlyStatementRequest) MarshalJSON() ([]byte, error) {
	if r.Month == 0 {
		return []byte("{}"), nil
	}
	month, err := formatStatementMonth(r.Month)
	if err != nil {
		return nil, err
	}
	return json.Marshal(map[string]string{"month": month})
}

// ApplyForMonthlyStatementResponse holds the generation time of an applied for monthly statement's download link
type ApplyForMonthlyStatementResponse struct {
	Timestamp types.Time `json:"ts"`
}

// MonthlyStatementResponse holds a monthly statement's download link
type MonthlyStatementResponse struct {
	FileLink  string     `json:"fileHref"`
	Timestamp types.Time `json:"ts"`
	State     string     `json:"state"`
}

// ConvertCurrencyResponse holds a currency available to convert
type ConvertCurrencyResponse struct {
	Currency currency.Code `json:"ccy"`
	Minimum  types.Number  `json:"min"` // Deprecated by OKX
	Maximum  types.Number  `json:"max"` // Deprecated by OKX
}

// ConvertCurrencyPairRequest holds the parameters for GetConvertCurrencyPair
type ConvertCurrencyPairRequest struct {
	FromCurrency currency.Code
	ToCurrency   currency.Code
	ConvertMode  string
}

// ConvertCurrencyPairResponse holds a convertible currency pair's limits
type ConvertCurrencyPairResponse struct {
	InstrumentID         string        `json:"instId"`
	BaseCurrency         currency.Code `json:"baseCcy"`
	BaseCurrencyMaximum  types.Number  `json:"baseCcyMax"`
	BaseCurrencyMinimum  types.Number  `json:"baseCcyMin"`
	QuoteCurrency        currency.Code `json:"quoteCcy"`
	QuoteCurrencyMaximum types.Number  `json:"quoteCcyMax"`
	QuoteCurrencyMinimum types.Number  `json:"quoteCcyMin"`
}

// EstimateQuoteRequest holds the parameters for EstimateQuote
type EstimateQuoteRequest struct {
	BaseCurrency            currency.Code `json:"baseCcy"`
	QuoteCurrency           currency.Code `json:"quoteCcy"`
	Side                    string        `json:"side"`
	RequestForQuoteSize     float64       `json:"rfqSz,string"`
	RequestForQuoteCurrency currency.Code `json:"rfqSzCcy"`
	ClientQuoteRequestID    string        `json:"clQReqId,omitempty"`
	Tag                     string        `json:"tag,omitempty"`
	ConvertMode             string        `json:"convertMode,omitempty"`
}

// EstimateQuoteResponse holds a convert quote
type EstimateQuoteResponse struct {
	QuoteTime                   types.Time    `json:"quoteTime"`
	TTLMilliseconds             types.Number  `json:"ttlMs"`
	ClientQuoteRequestID        string        `json:"clQReqId"`
	QuoteID                     string        `json:"quoteId"`
	BaseCurrency                currency.Code `json:"baseCcy"`
	QuoteCurrency               currency.Code `json:"quoteCcy"`
	Side                        order.Side    `json:"side"`
	OriginalRequestForQuoteSize types.Number  `json:"origRfqSz"`
	RequestForQuoteSize         types.Number  `json:"rfqSz"`
	RequestForQuoteCurrency     currency.Code `json:"rfqSzCcy"`
	ConvertPrice                types.Number  `json:"cnvtPx"`
	BaseSize                    types.Number  `json:"baseSz"`
	QuoteSize                   types.Number  `json:"quoteSz"`
}

// ConvertTradeRequest holds the parameters for ConvertTrade
type ConvertTradeRequest struct {
	QuoteID              string        `json:"quoteId"`
	BaseCurrency         currency.Code `json:"baseCcy"`
	QuoteCurrency        currency.Code `json:"quoteCcy"`
	Side                 string        `json:"side"`
	Size                 float64       `json:"sz,string"`
	SizeCurrency         currency.Code `json:"szCcy"`
	ClientTradeRequestID string        `json:"clTReqId,omitempty"`
	Tag                  string        `json:"tag,omitempty"`
	ConvertMode          string        `json:"convertMode,omitempty"`
}

// ConvertTradeResponse holds a convert trade
type ConvertTradeResponse struct {
	TradeID              string        `json:"tradeId"`
	QuoteID              string        `json:"quoteId"`
	ClientTradeRequestID string        `json:"clTReqId"`
	State                string        `json:"state"`
	InstrumentID         string        `json:"instId"`
	BaseCurrency         currency.Code `json:"baseCcy"`
	QuoteCurrency        currency.Code `json:"quoteCcy"`
	Side                 order.Side    `json:"side"`
	FillPrice            types.Number  `json:"fillPx"`
	FillBaseSize         types.Number  `json:"fillBaseSz"`
	FillQuoteSize        types.Number  `json:"fillQuoteSz"`
	Timestamp            types.Time    `json:"ts"`
}

// ConvertHistoryRequest holds the filters for GetConvertHistory
type ConvertHistoryRequest struct {
	ClientTradeRequestID string
	StartTime            time.Time
	EndTime              time.Time
	Limit                uint64
	Tag                  string
}

// ConvertHistoryResponse holds a convert trade from the convert history
type ConvertHistoryResponse struct {
	TradeID              string        `json:"tradeId"`
	ClientTradeRequestID string        `json:"clTReqId"`
	State                string        `json:"state"`
	InstrumentID         string        `json:"instId"`
	BaseCurrency         currency.Code `json:"baseCcy"`
	QuoteCurrency        currency.Code `json:"quoteCcy"`
	Side                 order.Side    `json:"side"`
	FillPrice            types.Number  `json:"fillPx"`
	FillBaseSize         types.Number  `json:"fillBaseSz"`
	FillQuoteSize        types.Number  `json:"fillQuoteSz"`
	Timestamp            types.Time    `json:"ts"`
}

// FiatPaymentMethodResponse holds a fiat deposit or withdrawal payment method
type FiatPaymentMethodResponse struct {
	Currency      currency.Code           `json:"ccy"`
	PaymentMethod string                  `json:"paymentMethod"`
	FeeRate       types.Number            `json:"feeRate"`
	MinimumFee    types.Number            `json:"minFee"`
	Limits        FiatPaymentMethodLimits `json:"limits"`
	Accounts      []FiatPaymentAccount    `json:"accounts"`
}

// FiatPaymentMethodLimits holds a fiat payment method's transaction limits
type FiatPaymentMethodLimits struct {
	DailyLimit            types.Number `json:"dailyLimit"`
	DailyLimitRemaining   types.Number `json:"dailyLimitRemaining"`
	WeeklyLimit           types.Number `json:"weeklyLimit"`
	WeeklyLimitRemaining  types.Number `json:"weeklyLimitRemaining"`
	MonthlyLimit          types.Number `json:"monthlyLimit"`
	MonthlyLimitRemaining types.Number `json:"monthlyLimitRemaining"`
	MaximumAmount         types.Number `json:"maxAmt"`
	MinimumAmount         types.Number `json:"minAmt"`
	LifetimeLimit         types.Number `json:"lifetimeLimit"`
}

// FiatPaymentAccount holds a payment account linked to a fiat payment method
type FiatPaymentAccount struct {
	PaymentAccountID string `json:"paymentAcctId"`
	AccountNumber    string `json:"acctNum"`
	RecipientName    string `json:"recipientName"`
	BankName         string `json:"bankName"`
	BankCode         string `json:"bankCode"`
	State            string `json:"state"`
}

// CreateFiatWithdrawalOrderRequest holds the parameters for CreateFiatWithdrawalOrder
type CreateFiatWithdrawalOrderRequest struct {
	PaymentAccountID string        `json:"paymentAcctId"`
	Currency         currency.Code `json:"ccy"`
	Amount           float64       `json:"amt,string"`
	PaymentMethod    string        `json:"paymentMethod"`
	ClientID         string        `json:"clientId"`
}

// FiatOrderResponse holds a fiat deposit or withdrawal order
type FiatOrderResponse struct {
	OrderID          string        `json:"ordId"`
	ClientID         string        `json:"clientId"`
	Amount           types.Number  `json:"amt"`
	Currency         currency.Code `json:"ccy"`
	Fee              types.Number  `json:"fee"`
	PaymentAccountID string        `json:"paymentAcctId"`
	PaymentMethod    string        `json:"paymentMethod"`
	State            string        `json:"state"`
	CreationTime     types.Time    `json:"cTime"`
	UpdateTime       types.Time    `json:"uTime"`
}

// CancelFiatWithdrawalOrderRequest holds the fiat withdrawal order CancelFiatWithdrawalOrder cancels
type CancelFiatWithdrawalOrderRequest struct {
	OrderID string `json:"ordId"`
}

// CancelFiatWithdrawalOrderResponse holds a cancelled fiat withdrawal order's state
type CancelFiatWithdrawalOrderResponse struct {
	OrderID string `json:"ordId"`
	State   string `json:"state"`
}

// FiatOrderHistoryRequest holds the filters for GetFiatWithdrawalOrderHistory and GetFiatDepositOrderHistory; both
// bounds of its time window are inclusive
type FiatOrderHistoryRequest struct {
	Currency      currency.Code
	PaymentMethod string
	State         string
	StartTime     time.Time
	EndTime       time.Time
	Limit         uint64
}

// BuySellCurrenciesResponse holds the currencies fiat buy and sell trades support
type BuySellCurrenciesResponse struct {
	FiatCurrencies   []BuySellCurrency `json:"fiatCcyList"`
	CryptoCurrencies []BuySellCurrency `json:"cryptoCcyList"`
}

// BuySellCurrency holds a currency fiat buy and sell trades support
type BuySellCurrency struct {
	Currency currency.Code `json:"ccy"`
}

// BuySellCurrencyPairRequest holds the parameters for GetBuySellCurrencyPair
type BuySellCurrencyPairRequest struct {
	FromCurrency currency.Code
	ToCurrency   currency.Code
}

// BuySellCurrencyPairResponse holds a fiat buy and sell currency pair's limits
type BuySellCurrencyPairResponse struct {
	Side                          string        `json:"side"` // Comma-separated when the pair supports both sides
	FromCurrency                  currency.Code `json:"fromCcy"`
	ToCurrency                    currency.Code `json:"toCcy"`
	SingleTradeMaximum            types.Number  `json:"singleTradeMax"`
	SingleTradeMinimum            types.Number  `json:"singleTradeMin"`
	FixedPriceDailyLimit          types.Number  `json:"fixedPxDailyLimit"`
	FixedPriceRemainingDailyQuota types.Number  `json:"fixedPxRemainingDailyQuota"`
	PaymentMethods                []string      `json:"paymentMethods"`
}

// BuySellQuoteRequest holds the parameters for GetBuySellQuote
type BuySellQuoteRequest struct {
	Side                    string        `json:"side"`
	FromCurrency            currency.Code `json:"fromCcy"`
	ToCurrency              currency.Code `json:"toCcy"`
	RequestForQuoteAmount   float64       `json:"rfqAmt,string"`
	RequestForQuoteCurrency currency.Code `json:"rfqCcy"`
}

// BuySellQuoteResponse holds a fiat buy or sell quote
type BuySellQuoteResponse struct {
	QuoteID                 string        `json:"quoteId"`
	Side                    order.Side    `json:"side"`
	FromCurrency            currency.Code `json:"fromCcy"`
	ToCurrency              currency.Code `json:"toCcy"`
	RequestForQuoteAmount   types.Number  `json:"rfqAmt"`
	RequestForQuoteCurrency currency.Code `json:"rfqCcy"`
	QuotePrice              types.Number  `json:"quotePx"`
	QuoteCurrency           currency.Code `json:"quoteCcy"`
	QuoteFromAmount         types.Number  `json:"quoteFromAmt"`
	QuoteToAmount           types.Number  `json:"quoteToAmt"`
	QuoteTime               types.Time    `json:"quoteTime"`
	TTLMilliseconds         types.Number  `json:"ttlMs"`
}

// BuySellTradeRequest holds the parameters for PlaceBuySellTrade; every field except QuoteID, PaymentMethod and
// ClientOrderID repeats the quote request's
type BuySellTradeRequest struct {
	QuoteID                 string        `json:"quoteId"`
	Side                    string        `json:"side"`
	FromCurrency            currency.Code `json:"fromCcy"`
	ToCurrency              currency.Code `json:"toCcy"`
	RequestForQuoteAmount   float64       `json:"rfqAmt,string"`
	RequestForQuoteCurrency currency.Code `json:"rfqCcy"`
	PaymentMethod           string        `json:"paymentMethod"`
	ClientOrderID           string        `json:"clOrdId"`
}

// BuySellTradeResponse holds a fiat buy or sell trade
type BuySellTradeResponse struct {
	OrderID                 string        `json:"ordId"`
	ClientOrderID           string        `json:"clOrdId"`
	QuoteID                 string        `json:"quoteId"`
	State                   string        `json:"state"`
	Side                    order.Side    `json:"side"`
	FromCurrency            currency.Code `json:"fromCcy"`
	ToCurrency              currency.Code `json:"toCcy"`
	RequestForQuoteAmount   types.Number  `json:"rfqAmt"`
	RequestForQuoteCurrency currency.Code `json:"rfqCcy"`
	FillPrice               types.Number  `json:"fillPx"`
	FillQuoteCurrency       currency.Code `json:"fillQuoteCcy"`
	FillFromAmount          types.Number  `json:"fillFromAmt"`
	FillToAmount            types.Number  `json:"fillToAmt"`
	CreationTime            types.Time    `json:"cTime"`
}

// BuySellTradeHistoryRequest holds the filters for GetBuySellTradeHistory
type BuySellTradeHistoryRequest struct {
	OrderID       string
	ClientOrderID string
	State         string
	StartTime     time.Time
	EndTime       time.Time
	Limit         uint64
}

// BuySellTradeHistoryResponse holds a fiat buy or sell trade from the trade history
type BuySellTradeHistoryResponse struct {
	OrderID                 string        `json:"ordId"`
	ClientOrderID           string        `json:"clOrdId"`
	QuoteID                 string        `json:"quoteId"`
	State                   string        `json:"state"`
	Side                    order.Side    `json:"side"`
	FromCurrency            currency.Code `json:"fromCcy"`
	ToCurrency              currency.Code `json:"toCcy"`
	RequestForQuoteAmount   types.Number  `json:"rfqAmt"`
	RequestForQuoteCurrency currency.Code `json:"rfqCcy"`
	FillPrice               types.Number  `json:"fillPx"`
	FillQuoteCurrency       currency.Code `json:"fillQuoteCcy"`
	FillFromAmount          types.Number  `json:"fillFromAmt"`
	FillToAmount            types.Number  `json:"fillToAmt"`
	CreationTime            types.Time    `json:"cTime"`
	UpdateTime              types.Time    `json:"uTime"`
}
