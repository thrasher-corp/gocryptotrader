package binance

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// PayTradeHistoryResponse holds Binance Pay transactions; Code "000000" means success
type PayTradeHistoryResponse struct {
	Code    string     `json:"code"`
	Message string     `json:"message"`
	Data    []PayTrade `json:"data"`
	Success bool       `json:"success"`
}

// PayTrade is a Binance Pay transaction. Which payer and receiver fields are set depends on the order type and on
// whether the account sent or received
type PayTrade struct {
	// OrderType is PAY, PAY_REFUND, C2C, CRYPTO_BOX, CRYPTO_BOX_RF, C2C_HOLDING, C2C_HOLDING_RF, PAYOUT or REMITTANCE
	OrderType       string     `json:"orderType"`
	TransactionID   string     `json:"transactionId"`
	TransactionTime types.Time `json:"transactionTime"`
	// Amount is positive for income and negative for expenditure
	Amount   types.Number  `json:"amount"`
	Currency currency.Code `json:"currency"`
	// WalletType is the main wallet: 1 funding, 2 spot, 3 fiat, 4 or 6 card payment, 5 earn
	WalletType uint64 `json:"walletType"`
	// WalletTypes lists every wallet a combined payment used
	WalletTypes  []uint64         `json:"walletTypes"`
	FundsDetail  []PayFundsDetail `json:"fundsDetail"`
	PayerInfo    PayPayerInfo     `json:"payerInfo"`
	ReceiverInfo PayReceiverInfo  `json:"receiverInfo"`
}

// PayFundsDetail is the amount of an asset a Binance Pay transaction used
type PayFundsDetail struct {
	Currency currency.Code `json:"currency"`
	Amount   types.Number  `json:"amount"`
	// WalletAssetCost maps each wallet type, as in PayTrade.WalletType, to the amount it paid
	WalletAssetCost map[uint64]types.Number `json:"walletAssetCost"`
}

// PayPayerInfo is the payer of a Binance Pay transaction
type PayPayerInfo struct {
	Name string `json:"name"` // Nickname or merchant name
	// Type is USER or MERCHANT
	Type      string `json:"type"`
	BinanceID string `json:"binanceId"` // Binance UID
}

// PayReceiverInfo is the receiver of a Binance Pay transaction
type PayReceiverInfo struct {
	Name string `json:"name"` // Nickname or merchant name
	// Type is USER or MERCHANT
	Type        string            `json:"type"`
	Email       string            `json:"email"`
	BinanceID   string            `json:"binanceId"`   // Binance UID
	AccountID   string            `json:"accountId"`   // Binance Pay ID
	CountryCode string            `json:"countryCode"` // International dialling code
	PhoneNumber string            `json:"phoneNumber"`
	MobileCode  string            `json:"mobileCode"` // Country code
	Extend      PayReceiverExtend `json:"extend"`
}

// PayReceiverExtend holds the bank or digital wallet of a remittance receiver
type PayReceiverExtend struct {
	InstitutionName string `json:"institutionName"`
	CardNumber      string `json:"cardNumber"`
	DigitalWalletID string `json:"digitalWalletId"`
}
