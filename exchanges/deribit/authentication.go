package deribit

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
)

// Grant types of an authentication request
const (
	GrantTypeClientCredentials = "client_credentials"
	GrantTypeClientSignature   = "client_signature"
	GrantTypeRefreshToken      = "refresh_token"
)

var (
	errInvalidGrantType     = errors.New("invalid grant type")
	errClientIDRequired     = errors.New("client ID is required")
	errClientSecretRequired = errors.New("client secret is required")
	errSignatureRequired    = errors.New("signature is required")
	errRefreshTokenRequired = errors.New("refresh token is required")
	errSubjectIDRequired    = errors.New("subject ID is required")
	errSessionNameRequired  = errors.New("session name is required")
	errTimestampRequired    = errors.New("timestamp is required")
	errSignatureCredentials = errors.New("credentials are required to sign an authentication request")
	errNoAccessToken        = errors.New("no access token returned")
)

// Authenticate requests an OAuth access token for private requests with client credentials, a client signature or a
// refresh token; HTTP requests are signed per request instead, so the token suits other clients and the websocket
func (e *Exchange) Authenticate(ctx context.Context, arg *AuthenticateRequest) (*AccessTokenResponse, error) {
	return e.authenticate(ctx, arg, false)
}

func (e *Exchange) authenticate(ctx context.Context, arg *AuthenticateRequest, overWebsocket bool) (*AccessTokenResponse, error) {
	if err := arg.validate(); err != nil {
		return nil, err
	}
	params := &AuthenticateParams{AuthenticateRequest: arg, Timestamp: unixMilli(arg.Timestamp)}
	var resp *AccessTokenResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "public/auth", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil || resp.AccessToken == "" {
		return nil, errNoAccessToken
	}
	return resp, nil
}

// validate checks an authentication request carries the fields its grant type needs
func (a *AuthenticateRequest) validate() error {
	if a == nil {
		return common.ErrNilPointer
	}
	switch a.GrantType {
	case GrantTypeClientCredentials:
		if a.ClientID == "" {
			return errClientIDRequired
		}
		if a.ClientSecret == "" {
			return errClientSecretRequired
		}
	case GrantTypeClientSignature:
		if a.ClientID == "" {
			return errClientIDRequired
		}
		if a.Timestamp.IsZero() {
			return errTimestampRequired
		}
		if a.Signature == "" {
			return errSignatureRequired
		}
	case GrantTypeRefreshToken:
		if a.RefreshToken == "" {
			return errRefreshTokenRequired
		}
	default:
		return fmt.Errorf("%w: %q", errInvalidGrantType, a.GrantType)
	}
	return nil
}

// NewClientSignatureRequest returns an authentication request signing the timestamp, nonce and optional data with the
// credentials' secret, so the secret is not sent
func NewClientSignatureRequest(creds *accounts.Credentials, timestamp time.Time, nonce, data string) (*AuthenticateRequest, error) {
	if creds == nil || creds.Key == "" || creds.Secret == "" {
		return nil, errSignatureCredentials
	}
	// The timestamp signed is the one sent
	payload := strconv.FormatUint(unixMilli(timestamp), 10) + "\n" + nonce + "\n" + data
	signature, err := crypto.GetHMAC(crypto.HashSHA256, []byte(payload), []byte(creds.Secret))
	if err != nil {
		return nil, err
	}
	return &AuthenticateRequest{
		GrantType: GrantTypeClientSignature,
		ClientID:  creds.Key,
		Timestamp: timestamp,
		Signature: hex.EncodeToString(signature),
		Nonce:     nonce,
		Data:      data,
	}, nil
}

// ExchangeToken returns a token for another subject, such as a subaccount, from a refresh token
func (e *Exchange) ExchangeToken(ctx context.Context, arg *ExchangeTokenRequest) (*AccessTokenResponse, error) {
	return e.exchangeToken(ctx, arg, false)
}

func (e *Exchange) exchangeToken(ctx context.Context, arg *ExchangeTokenRequest, overWebsocket bool) (*AccessTokenResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.RefreshToken == "" {
		return nil, errRefreshTokenRequired
	}
	if arg.SubjectID == 0 {
		return nil, errSubjectIDRequired
	}
	var resp *AccessTokenResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "public/exchange_token", arg, &resp); err != nil {
		return nil, err
	}
	if resp == nil || resp.AccessToken == "" {
		return nil, errNoAccessToken
	}
	return resp, nil
}

// ForkToken returns a token for a new named session from the refresh token of a session scoped token
func (e *Exchange) ForkToken(ctx context.Context, arg *ForkTokenRequest) (*AccessTokenResponse, error) {
	return e.forkToken(ctx, arg, false)
}

func (e *Exchange) forkToken(ctx context.Context, arg *ForkTokenRequest, overWebsocket bool) (*AccessTokenResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.RefreshToken == "" {
		return nil, errRefreshTokenRequired
	}
	if arg.SessionName == "" {
		return nil, errSessionNameRequired
	}
	var resp *AccessTokenResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "public/fork_token", arg, &resp); err != nil {
		return nil, err
	}
	if resp == nil || resp.AccessToken == "" {
		return nil, errNoAccessToken
	}
	return resp, nil
}
