package main

import (
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// variables for command line overrides
var (
	orderTypeOverride          string
	outputOverride             string
	orderSideOverride          string
	currencyPairOverride       string
	assetTypeOverride          string
	orderPriceOverride         float64
	orderAmountOverride        float64
	withdrawAddressOverride    string
	authenticatedOnly          bool
	verboseOverride            bool
	exchangesToUseOverride     string
	exchangesToExcludeOverride string
	outputFileName             string
	exchangesToUseList         []string
	exchangesToExcludeList     []string
)

// Config the data structure for wrapperconfig.json to store all customisation
type Config struct {
	OrderSubmission OrderSubmission                         `json:"orderSubmission"`
	BankDetails     Bank                                    `json:"bankAccount"`
	Exchanges       map[string]*config.APICredentialsConfig `json:"exchanges"`
}

// ExchangeResponses contains all responses
// associated with an exchange
type ExchangeResponses struct {
	ID                 string
	ExchangeName       string                       `json:"exchangeName"`
	AssetPairResponses []ExchangeAssetPairResponses `json:"responses"`
	ErrorCount         int64                        `json:"errorCount"`
	APIKeysSet         bool                         `json:"apiKeysSet"`
}

// ExchangeAssetPairResponses contains all responses
// associated with an asset type and currency pair
type ExchangeAssetPairResponses struct {
	ErrorCount        int64              `json:"errorCount"`
	AssetType         asset.Item         `json:"asset"`
	Pair              currency.Pair      `json:"currency"`
	EndpointResponses []EndpointResponse `json:"responses"`
}

// EndpointResponse is the data for an individual wrapper response
type EndpointResponse struct {
	Function   string          `json:"function"`
	Error      string          `json:"error"`
	Response   any             `json:"response"`
	SentParams json.RawMessage `json:"sentParams"`
}

// Bank contains all required data for a wrapper withdrawal request
type Bank struct {
	BankAccountName               string  `json:"bankAccountName"`
	BankAccountNumber             string  `json:"bankAccountNumber"`
	BankAddress                   string  `json:"bankAddress"`
	BankCity                      string  `json:"bankCity"`
	BankCountry                   string  `json:"bankCountry"`
	BankName                      string  `json:"bankName"`
	BankPostalCode                string  `json:"bankPostalCode"`
	Iban                          string  `json:"iban"`
	IntermediaryBankAccountName   string  `json:"intermediaryBankAccountName"`
	IntermediaryBankAccountNumber float64 `json:"intermediaryBankAccountNumber"`
	IntermediaryBankAddress       string  `json:"intermediaryBankAddress"`
	IntermediaryBankCity          string  `json:"intermediaryBankCity"`
	IntermediaryBankCountry       string  `json:"intermediaryBankCountry"`
	IntermediaryBankName          string  `json:"intermediaryBankName"`
	IntermediaryBankPostalCode    string  `json:"intermediaryBankPostalCode"`
	IntermediaryIban              string  `json:"intermediaryIban"`
	IntermediaryIsExpressWire     bool    `json:"intermediaryIsExpressWire"`
	IntermediarySwiftCode         string  `json:"intermediarySwiftCode"`
	IsExpressWire                 bool    `json:"isExpressWire"`
	RequiresIntermediaryBank      bool    `json:"requiresIntermediaryBank"`
	SwiftCode                     string  `json:"swiftCode"`
	BankCode                      float64 `json:"bankCode"`
	IntermediaryBankCode          float64 `json:"intermediaryBankCode"`
}

// OrderSubmission contains all data required for a wrapper order submission
type OrderSubmission struct {
	OrderSide string     `json:"orderSide"`
	OrderType string     `json:"orderType"`
	Amount    float64    `json:"amount"`
	Price     float64    `json:"price"`
	OrderID   string     `json:"orderID"`
	AssetType asset.Item `json:"assetType"`
}
