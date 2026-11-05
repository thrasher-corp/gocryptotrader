package bitstamp

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Instant convert address formats
const (
	AddressFormatP2SHP2WSH = "P2SHP2WSH"
	AddressFormatBECH32    = "BECH32"
)

// InstantConvertAddressResponse holds an instant convert address
type InstantConvertAddressResponse struct {
	Address string `json:"address"`
}

// InstantConvertAddressInfoResponse holds the transactions of an instant convert address
type InstantConvertAddressInfoResponse struct {
	Address      string                      `json:"address"`
	CurrencyPair currency.Pair               `json:"currency_pair"`
	Transactions []InstantConvertTransaction `json:"transactions"`
}

// InstantConvertTransaction holds a conversion order for an instant convert address
type InstantConvertTransaction struct {
	OrderID uint64                `json:"order_id"`
	Count   uint64                `json:"count"`
	Trades  []InstantConvertTrade `json:"trades"`
}

// InstantConvertTrade holds a trade executed by an instant convert order
type InstantConvertTrade struct {
	ExchangeRate types.Number `json:"exchange_rate"`
	BTCAmount    types.Number `json:"btc_amount"`
	Fees         types.Number `json:"fees"`
}
