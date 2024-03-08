package bitget

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// Bitget is the overarching type across this package
type Bitget struct {
	exchange.Base
}

const (
	bitgetAPIURL = "api.bitget.com"

	// Public endpoints

	// Authenticated endpoints

	// Errors
	errUnknownEndpointLimit = "unknown endpoint limit %v"
)

// Start implementing public and private exchange API funcs below

func (b *Bitget) SendAuthenticatedHTTPRequest(ctx context.Context, method, path string, queryparams url.Values, bodyParams map[string]interface{}) error {
	creds, err := b.GetCredentials(ctx)
	if err != nil {
		return err
	}
	payload := []byte("")
	if bodyParams != nil {
		payload, err = json.Marshal(bodyParams)
		if err != nil {
			return err
		}
	}
	query := common.EncodeURLValues("", queryparams)
	t := strconv.FormatInt(time.Now().UnixMilli(), 10)
	message := t + method + path + query + string(payload)
	// The exchange also supports user-generated RSA keys, but we haven't implemented that yet
	var hmac []byte
	hmac, err = crypto.GetHMAC(crypto.HashSHA256, []byte(message), []byte(creds.Secret))
	if err != nil {
		return err
	}
	headers := make(map[string]string)
	headers["ACCESS-KEY"] = creds.Key
	headers["ACCESS-SIGN"] = crypto.Base64Encode(hmac)
	headers["ACCESS-TIMESTAMP"] = t
	headers["ACCESS-PASSPHRASE"] = creds.ClientID
	headers["Content-Type"] = "application/json"
	headers["locale"] = "en-US"
	return nil
}
