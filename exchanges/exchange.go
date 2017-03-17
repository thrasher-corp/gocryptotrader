package exchange

import (
	"log"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
)

const (
	WarningBase64DecryptSecretKeyFailed = "WARNING -- Exchange %s unable to base64 decode secret key.. Disabling Authenticated API support."
)

type ExchangeBase struct {
	Name                        string
	Enabled                     bool
	Verbose                     bool
	Websocket                   bool
	RESTPollingDelay            time.Duration
	AuthenticatedAPISupport     bool
	APISecret, APIKey, ClientID string
	TakerFee, MakerFee, Fee     float64
	BaseCurrencies              []string
	AvailablePairs              []string
	EnabledPairs                []string
	WebsocketURL                string
	APIUrl                      string
}

func (e *ExchangeBase) GetName() string {
	return e.Name
}
func (e *ExchangeBase) GetEnabledCurrencies() []string {
	return e.EnabledPairs
}
func (e *ExchangeBase) SetEnabled(enabled bool) {
	e.Enabled = enabled
}

func (e *ExchangeBase) IsEnabled() bool {
	return e.Enabled
}

func (e *ExchangeBase) SetAPIKeys(APIKey, APISecret, ClientID string, b64Decode bool) {
	e.APIKey = APIKey
	e.ClientID = ClientID

	if b64Decode {
		result, err := common.Base64Decode(APISecret)
		if err != nil {
			e.AuthenticatedAPISupport = false
			log.Printf(WarningBase64DecryptSecretKeyFailed, e.Name)
		}
		e.APISecret = string(result)
	} else {
		e.APISecret = APISecret
	}
}
