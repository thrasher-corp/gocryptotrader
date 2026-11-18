package kraken

import (
	"errors"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
)

var (
	errFuturesWalletEmpty = errors.New("wallet is empty")
	errFuturesUserEmpty   = errors.New("user account is empty")
)

// FuturesWalletTransferRequest holds the parameters of Initiate wallet transfer
type FuturesWalletTransferRequest struct {
	// FromWallet and ToWallet name wallets as Get wallets does, such as cash, flex or fi_ethusd
	FromWallet string
	ToWallet   string
	Currency   currency.Code
	Amount     float64
}

// FuturesSubaccountTransferRequest holds the parameters of Initiate sub account transfer
type FuturesSubaccountTransferRequest struct {
	// FromUser and ToUser are the account or one of its subaccounts
	FromUser string
	ToUser   string
	// FromWallet and ToWallet name wallets as Get wallets does, such as cash, flex or fi_ethusd
	FromWallet string
	ToWallet   string
	Currency   currency.Code
	Amount     float64
}

// FuturesWithdrawToSpotWalletRequest holds the parameters of Initiate withdrawal to Spot wallet
type FuturesWithdrawToSpotWalletRequest struct {
	Currency currency.Code
	Amount   float64
	// SourceWallet names the wallet withdrawn from as Get wallets does, such as flex; the cash wallet when it is empty
	SourceWallet string
}

// FuturesWithdrawToSpotWalletResponse holds the reference of a withdrawal to the Spot wallet
type FuturesWithdrawToSpotWalletResponse struct {
	UID        string    `json:"uid"`
	ServerTime time.Time `json:"serverTime"`
}
