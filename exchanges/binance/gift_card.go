package binance

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// CreateDualTokenGiftCard calls Create a dual-token gift card (fixed value, discount feature), paying baseTokenAmount
// of baseToken plus a fee from the spot wallet for a card that redeems to faceToken of the same value
func (e *Exchange) CreateDualTokenGiftCard(ctx context.Context, baseToken, faceToken currency.Code, baseTokenAmount float64) (*GiftCardResponse, error) {
	if baseToken.IsEmpty() {
		return nil, fmt.Errorf("%w: baseToken is required", currency.ErrCurrencyCodeEmpty)
	}
	if faceToken.IsEmpty() {
		return nil, fmt.Errorf("%w: faceToken is required", currency.ErrCurrencyCodeEmpty)
	}
	if baseTokenAmount <= 0 {
		return nil, fmt.Errorf("%w: baseTokenAmount is required", limits.ErrAmountBelowMin)
	}
	params := url.Values{}
	params.Set("baseToken", baseToken.Upper().String())
	params.Set("faceToken", faceToken.Upper().String())
	params.Set("baseTokenAmount", strconv.FormatFloat(baseTokenAmount, 'f', -1, 64))
	var resp *GiftCardResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/giftcard/buyCode", params, sapiDefaultRate, &resp)
}

// CreateSingleTokenGiftCard calls Create a single-token gift card, paying amount of token plus a fee from the spot
// wallet. Only KYB-verified entity accounts can create gift cards, the API key needs withdrawals enabled, and an
// account may create 4,200,000 USDC worth and 6,000 cards a month
func (e *Exchange) CreateSingleTokenGiftCard(ctx context.Context, token currency.Code, amount float64) (*GiftCardResponse, error) {
	if token.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	params := url.Values{}
	params.Set("token", token.Upper().String())
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	var resp *GiftCardResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/giftcard/createCode", params, sapiDefaultRate, &resp)
}

// FetchRSAPublicKey calls Fetch RSA Public Key, returning the key that encrypts a redemption code for
// RedeemBinanceGiftCard; the key is valid only for the current day
func (e *Exchange) FetchRSAPublicKey(ctx context.Context) (*RSAPublicKeyResponse, error) {
	var resp *RSAPublicKeyResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/giftcard/cryptography/rsa-public-key", nil, sapiDefaultRate, &resp)
}

// FetchTokenLimit calls Fetch Token Limit, returning the tokens a dual-token gift card paid in baseToken can redeem
// to and their limits
func (e *Exchange) FetchTokenLimit(ctx context.Context, baseToken currency.Code) (*TokenLimitResponse, error) {
	if baseToken.IsEmpty() {
		return nil, fmt.Errorf("%w: baseToken is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	params.Set("baseToken", baseToken.Upper().String())
	var resp *TokenLimitResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/giftcard/buyCode/token-limit", params, sapiDefaultRate, &resp)
}

// RedeemBinanceGiftCard calls Redeem a Binance Gift Card, depositing its tokens to the spot wallet. The code may be
// plaintext or encrypted with FetchRSAPublicKey's key; externalUID optionally identifies the redeeming user on a
// partner platform
func (e *Exchange) RedeemBinanceGiftCard(ctx context.Context, code, externalUID string) (*RedeemGiftCardResponse, error) {
	if code == "" {
		return nil, errCodeRequired
	}
	params := url.Values{}
	params.Set("code", code)
	if externalUID != "" {
		params.Set("externalUid", externalUID)
	}
	var resp *RedeemGiftCardResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/giftcard/redeemCode", params, sapiDefaultRate, &resp)
}

// VerifyBinanceGiftCardNumber calls Verify Binance Gift Card by Gift Card Number; five wrong numbers within an hour
// block verification for that hour
func (e *Exchange) VerifyBinanceGiftCardNumber(ctx context.Context, referenceNumber string) (*GiftCardVerificationResponse, error) {
	if referenceNumber == "" {
		return nil, errReferenceNumberRequired
	}
	params := url.Values{}
	params.Set("referenceNo", referenceNumber)
	var resp *GiftCardVerificationResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/giftcard/verify", params, sapiDefaultRate, &resp)
}
