package lbank

import (
	"bytes"
	"context"
	"crypto"
	"crypto/hmac"
	"crypto/md5" //nolint:gosec // LBank's signature scheme signs an MD5 digest of the request parameters
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	restURL             = "https://api.lbank.info"
	contractRESTURL     = "https://lbkperp.lbank.com"
	websocketURL        = "wss://api.lbank.info/ws/V2/"
	tradeBaseURL        = "https://www.lbank.com/trade/"
	futuresTradeBaseURL = "https://www.lbank.com/futures/"

	signatureMethodRSA  = "RSA"
	signatureMethodHMAC = "HmacSHA256"

	// echostrLength is the length of the random string LBank requires on every signed request, which must be 30 to 40
	// letters and digits
	echostrLength = 35
)

var (
	errRequestFailed           = errors.New("request failed")
	errPEMBlockIsNil           = errors.New("pem block is nil")
	errUnableToParsePrivateKey = errors.New("unable to parse private key")
	errOrderIDRequired         = errors.New("order ID or client order ID is required")
	errInvalidOrderType        = errors.New("invalid order type")
	errAddressRequired         = errors.New("address is required")
	errNetworkRequired         = errors.New("network name is required")
	errSubscribeKeyRequired    = errors.New("subscribe key is required")
	errProductGroupRequired    = errors.New("product group is required")
	errLimitOutOfRange         = errors.New("limit is out of range")
	errMalformedKline          = errors.New("malformed kline")
	errTimeRangeTooLong        = errors.New("time range is too long")
	errHistoryPageFull         = errors.New("a full page of trades shares one timestamp")
	errHistoryPageLimit        = errors.New("history page limit reached")
)

// Exchange implements exchange.IBotExchange and contains additional specific api methods for interacting with LBank
type Exchange struct {
	exchange.Base

	// rsaKeys holds the private key parsed from each RSA secret, so a signed request needn't parse its key again
	rsaKeys sync.Map
	// subscribeKeys holds the subscribe key each account websocket connection receives its events with
	subscribeKeys subscribeKeyStore
}

// SendHTTPRequest sends an unauthenticated request and decodes the data its reply carries into result
func (e *Exchange) SendHTTPRequest(ctx context.Context, ep exchange.URL, epl request.EndpointLimit, method, path string, params url.Values, result any) error {
	endpoint, err := e.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	var resp json.RawMessage
	item := &request.Item{
		Method:                 method,
		Path:                   common.EncodeURLValues(endpoint+path, params),
		Result:                 &resp,
		Verbose:                e.Verbose,
		HTTPDebugging:          e.HTTPDebugging,
		HTTPRecording:          e.HTTPRecording,
		HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
	}
	if err := e.SendPayload(ctx, epl, func() (*request.Item, error) {
		return item, nil
	}, request.UnauthenticatedRequest); err != nil {
		return err
	}
	if err := decodeResponse(resp, result); err != nil {
		return fmt.Errorf("%s: %w", e.Name, err)
	}
	return nil
}

// SendAuthHTTPRequest sends a signed request to the spot API and decodes the data its reply carries into result. LBank
// signs the request's parameters together with the timestamp, signature method and echostr it also sends as headers
func (e *Exchange) SendAuthHTTPRequest(ctx context.Context, epl request.EndpointLimit, path string, params url.Values, result any) error {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}
	signer, err := e.getSigner(creds.Secret)
	if err != nil {
		return err
	}
	endpoint, err := e.API.Endpoints.GetURL(exchange.RestSpot)
	if err != nil {
		return err
	}
	var resp json.RawMessage
	if err := e.SendPayload(ctx, epl, func() (*request.Item, error) {
		// The request is rebuilt for each attempt, since LBank refuses a timestamp more than 5 seconds old
		body, headers, err := signRequest(signer, creds, params, time.Now())
		if err != nil {
			return nil, err
		}
		return &request.Item{
			Method:                 http.MethodPost,
			Path:                   endpoint + path,
			Headers:                headers,
			Body:                   strings.NewReader(body),
			Result:                 &resp,
			Verbose:                e.Verbose,
			HTTPDebugging:          e.HTTPDebugging,
			HTTPRecording:          e.HTTPRecording,
			HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
		}, nil
	}, request.AuthenticatedRequest); err != nil {
		return err
	}
	if err := decodeResponse(resp, result); err != nil {
		if errors.Is(err, errRequestFailed) {
			// LBank reports refused credentials, signatures and permissions in the reply rather than with an HTTP status
			return fmt.Errorf("%s: %w: %w", e.Name, request.ErrAuthRequestFailed, err)
		}
		return fmt.Errorf("%s: %w", e.Name, err)
	}
	return nil
}

// signRequest returns the form body and headers of a signed request
func signRequest(signer *requestSigner, creds *accounts.Credentials, params url.Values, now time.Time) (body string, headers map[string]string, err error) {
	echostr, err := common.GenerateRandomString(echostrLength, common.SmallLetters, common.CapitalLetters, common.NumberCharacters)
	if err != nil {
		return "", nil, err
	}
	timestamp := strconv.FormatInt(now.UnixMilli(), 10)
	signed := make(url.Values, len(params)+4)
	for k, v := range params {
		signed[k] = slices.Clone(v)
	}
	signed.Set("api_key", creds.Key)
	signed.Set("echostr", echostr)
	signed.Set("signature_method", signer.method)
	signed.Set("timestamp", timestamp)
	sign, err := signer.sign(parameterString(signed))
	if err != nil {
		return "", nil, err
	}
	// LBank's connectors send the timestamp, signature method and echostr as headers only, and sign them with the
	// parameters
	signed.Del("echostr")
	signed.Del("signature_method")
	signed.Del("timestamp")
	signed.Set("sign", sign)
	return signed.Encode(), map[string]string{
		"Content-Type":     "application/x-www-form-urlencoded",
		"timestamp":        timestamp,
		"signature_method": signer.method,
		"echostr":          echostr,
	}, nil
}

// parameterString returns the parameters sorted by name and joined with their raw values, which is the string LBank
// signs; url.Values.Encode would escape the values and change the signature
func parameterString(params url.Values) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	var sb strings.Builder
	for i, k := range keys {
		if i > 0 {
			sb.WriteByte('&')
		}
		sb.WriteString(k)
		sb.WriteByte('=')
		sb.WriteString(params.Get(k))
	}
	return sb.String()
}

// requestSigner signs requests with an RSA private key or an HmacSHA256 secret, the two signature methods LBank offers
type requestSigner struct {
	method string
	rsaKey *rsa.PrivateKey
	secret []byte
}

// getSigner returns the signer for a secret. LBank issues RSA keys as a PKCS #8 private key and HmacSHA256 keys as a
// hexadecimal secret, so a secret is taken as an RSA key when it is PEM encoded or is base64 DER, whose encoding
// starts with "MI"
func (e *Exchange) getSigner(secret string) (*requestSigner, error) {
	trimmed := strings.TrimSpace(secret)
	if !strings.Contains(trimmed, "-----BEGIN") && !strings.HasPrefix(trimmed, "MI") {
		return &requestSigner{method: signatureMethodHMAC, secret: []byte(secret)}, nil
	}
	if key, ok := e.rsaKeys.Load(secret); ok {
		if rsaKey, ok := key.(*rsa.PrivateKey); ok {
			return &requestSigner{method: signatureMethodRSA, rsaKey: rsaKey}, nil
		}
	}
	rsaKey, err := parseRSAPrivateKey(trimmed)
	if err != nil {
		return nil, err
	}
	e.rsaKeys.Store(secret, rsaKey)
	return &requestSigner{method: signatureMethodRSA, rsaKey: rsaKey}, nil
}

// parseRSAPrivateKey parses a PKCS #8 RSA private key, PEM encoded or as the bare base64 LBank shows it
func parseRSAPrivateKey(secret string) (*rsa.PrivateKey, error) {
	if !strings.Contains(secret, "-----BEGIN") {
		secret = "-----BEGIN PRIVATE KEY-----\n" + secret + "\n-----END PRIVATE KEY-----"
	}
	block, _ := pem.Decode([]byte(secret))
	if block == nil {
		return nil, errPEMBlockIsNil
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errUnableToParsePrivateKey, err)
	}
	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, common.GetTypeAssertError("*rsa.PrivateKey", key)
	}
	return rsaKey, nil
}

// sign returns the signature of a parameter string. Both methods sign the upper case hexadecimal MD5 digest of it: RSA
// as a base64 SHA256withRSA signature and HmacSHA256 as a lower case hexadecimal HMAC
func (s *requestSigner) sign(parameters string) (string, error) {
	digest := md5.Sum([]byte(parameters)) //nolint:gosec // LBank's signature scheme signs an MD5 digest of the request parameters
	prepared := []byte(strings.ToUpper(hex.EncodeToString(digest[:])))
	if s.method == signatureMethodHMAC {
		mac := hmac.New(sha256.New, s.secret)
		mac.Write(prepared)
		return hex.EncodeToString(mac.Sum(nil)), nil
	}
	hashed := sha256.Sum256(prepared)
	signature, err := rsa.SignPKCS1v15(rand.Reader, s.rsaKey, crypto.SHA256, hashed[:])
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(signature), nil
}

// decodeResponse decodes the data an LBank reply carries into result, or returns the failure the reply reports
func decodeResponse(raw json.RawMessage, result any) error {
	payload, err := unwrapV2Response(raw)
	if err != nil {
		return err
	}
	if result == nil {
		return nil
	}
	return json.Unmarshal(payload, result)
}

// unwrapV2Response returns the payload held in an LBank v2 response envelope. A payload that is not a JSON object is
// returned unchanged; a JSON object that does not decode as the envelope is reported as an error, so that a change of
// shape in the envelope cannot silently disable the failed request check. Several documented replies have no envelope,
// so an object without data is returned whole. The errors returned here are bare; the senders add the exchange name
func unwrapV2Response(payload json.RawMessage) (json.RawMessage, error) {
	var resp v2Response
	if err := json.Unmarshal(payload, &resp); err != nil {
		if bytes.HasPrefix(bytes.TrimLeft(payload, " \t\r\n"), []byte("{")) {
			return nil, fmt.Errorf("decoding response envelope: %w", err)
		}
		return payload, nil
	}
	if resp.Result != nil && !resp.Result.Bool() {
		return nil, &requestError{code: resp.ErrorCode.Int64(), message: resp.Message}
	}
	if resp.Data != nil {
		return resp.Data, nil
	}
	return payload, nil
}
