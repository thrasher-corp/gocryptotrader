package binance

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// GiftCardResponse holds a created gift card; Code "000000" means success
type GiftCardResponse struct {
	Code    string   `json:"code"`
	Message string   `json:"message"`
	Data    GiftCard `json:"data"`
	Success bool     `json:"success"`
}

// GiftCard is a created gift card: its public card number and its confidential redemption code
type GiftCard struct {
	ReferenceNumber string     `json:"referenceNo"`
	Code            string     `json:"code"`
	ExpiredTime     types.Time `json:"expiredTime"`
}

// RSAPublicKeyResponse holds the RSA public key that encrypts redemption codes; Code "000000" means success
type RSAPublicKeyResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data"`
	Success bool   `json:"success"`
}

// TokenLimitResponse holds the tokens a dual-token gift card can redeem to; Code "000000" means success
type TokenLimitResponse struct {
	Code    string               `json:"code"`
	Message string               `json:"message"`
	Data    []GiftCardTokenLimit `json:"data"`
	Success bool                 `json:"success"`
}

// GiftCardTokenLimit is a token a dual-token gift card can redeem to and the limits of the paid amount
type GiftCardTokenLimit struct {
	Coin    currency.Code `json:"coin"`
	FromMin types.Number  `json:"fromMin"`
	FromMax types.Number  `json:"fromMax"`
}

// RedeemGiftCardResponse holds a redeemed gift card; Code "000000" means success
type RedeemGiftCardResponse struct {
	Code    string           `json:"code"`
	Message string           `json:"message"`
	Data    RedeemedGiftCard `json:"data"`
	Success bool             `json:"success"`
}

// RedeemedGiftCard is a redeemed gift card and the tokens it deposited
type RedeemedGiftCard struct {
	ReferenceNumber string        `json:"referenceNo"`
	IdentityNumber  string        `json:"identityNo"`
	Token           currency.Code `json:"token"`
	Amount          types.Number  `json:"amount"`
}

// GiftCardVerificationResponse holds a gift card's validity; Code "000000" means success
type GiftCardVerificationResponse struct {
	Code    string               `json:"code"`
	Message string               `json:"message"`
	Data    GiftCardVerification `json:"data"`
	Success bool                 `json:"success"`
}

// GiftCardVerification is a gift card's validity and value
type GiftCardVerification struct {
	Valid  bool          `json:"valid"`
	Token  currency.Code `json:"token"`
	Amount types.Number  `json:"amount"`
}
