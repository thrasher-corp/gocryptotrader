package exchange

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/banking"
)

const (
	warningBase64DecryptSecretKeyFailed = "exchange %s unable to base64 decode secret key.. Disabling Authenticated API support" // nolint // False positive (G101: Potential hardcoded credentials)
	// WarningAuthenticatedRequestWithoutCredentialsSet error message for authenticated request without credentials set
	WarningAuthenticatedRequestWithoutCredentialsSet = "exchange %s authenticated HTTP request called but not supported due to unset/default API keys"
	// DefaultHTTPTimeout is the default HTTP/HTTPS Timeout for exchange requests
	DefaultHTTPTimeout = time.Second * 15
	// DefaultWebsocketResponseCheckTimeout is the default delay in checking for an expected websocket response
	DefaultWebsocketResponseCheckTimeout = time.Millisecond * 50
	// DefaultWebsocketResponseMaxLimit is the default max wait for an expected websocket response before a timeout
	DefaultWebsocketResponseMaxLimit = time.Second * 7
	// DefaultWebsocketOrderbookBufferLimit is the maximum number of orderbook updates that get stored before being applied
	DefaultWebsocketOrderbookBufferLimit = 5
)

func (e *Base) checkAndInitRequester() {
	if e.Requester == nil {
		e.Requester = request.New(e.Name,
			new(http.Client),
			nil)
	}
}

// SetHTTPClientTimeout sets the timeout value for the exchanges
// HTTP Client
func (e *Base) SetHTTPClientTimeout(t time.Duration) {
	e.checkAndInitRequester()
	e.Requester.HTTPClient.Timeout = t
}

// SetHTTPClient sets exchanges HTTP client
func (e *Base) SetHTTPClient(h *http.Client) {
	e.checkAndInitRequester()
	e.Requester.HTTPClient = h
}

// GetHTTPClient gets the exchanges HTTP client
func (e *Base) GetHTTPClient() *http.Client {
	e.checkAndInitRequester()
	return e.Requester.HTTPClient
}

// SetHTTPClientUserAgent sets the exchanges HTTP user agent
func (e *Base) SetHTTPClientUserAgent(ua string) {
	e.checkAndInitRequester()
	e.Requester.UserAgent = ua
	e.HTTPUserAgent = ua
}

// GetHTTPClientUserAgent gets the exchanges HTTP user agent
func (e *Base) GetHTTPClientUserAgent() string {
	return e.HTTPUserAgent
}

// SetClientProxyAddress sets a proxy address for REST and websocket requests
func (e *Base) SetClientProxyAddress(addr string) error {
	if addr != "" {
		proxy, err := url.Parse(addr)
		if err != nil {
			return fmt.Errorf("exchange.go - setting proxy address error %s",
				err)
		}

		// No needs to check err here as the only err condition is an empty
		// string which is already checked above
		_ = e.Requester.SetProxy(proxy)

		if e.Websocket != nil {
			err = e.Websocket.SetProxyAddress(addr)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// SetFeatureDefaults sets the exchanges default feature
// support set
func (e *Base) SetFeatureDefaults() {
	if e.Config.Features == nil {
		s := &config.FeaturesConfig{
			Supports: config.FeaturesSupportedConfig{
				Websocket: e.Features.Supports.Websocket,
				REST:      e.Features.Supports.REST,
				RESTCapabilities: protocol.Features{
					AutoPairUpdates: e.Features.Supports.RESTCapabilities.AutoPairUpdates,
				},
			},
		}

		if e.Config.SupportsAutoPairUpdates != nil {
			s.Supports.RESTCapabilities.AutoPairUpdates = *e.Config.SupportsAutoPairUpdates
			s.Enabled.AutoPairUpdates = *e.Config.SupportsAutoPairUpdates
		} else {
			s.Supports.RESTCapabilities.AutoPairUpdates = e.Features.Supports.RESTCapabilities.AutoPairUpdates
			s.Enabled.AutoPairUpdates = e.Features.Supports.RESTCapabilities.AutoPairUpdates
			if !s.Supports.RESTCapabilities.AutoPairUpdates {
				e.Config.CurrencyPairs.LastUpdated = time.Now().Unix()
				e.CurrencyPairs.LastUpdated = e.Config.CurrencyPairs.LastUpdated
			}
		}
		e.Config.Features = s
		e.Config.SupportsAutoPairUpdates = nil
	} else {
		if e.Features.Supports.RESTCapabilities.AutoPairUpdates != e.Config.Features.Supports.RESTCapabilities.AutoPairUpdates {
			e.Config.Features.Supports.RESTCapabilities.AutoPairUpdates = e.Features.Supports.RESTCapabilities.AutoPairUpdates

			if !e.Config.Features.Supports.RESTCapabilities.AutoPairUpdates {
				e.Config.CurrencyPairs.LastUpdated = time.Now().Unix()
			}
		}

		if e.Features.Supports.REST != e.Config.Features.Supports.REST {
			e.Config.Features.Supports.REST = e.Features.Supports.REST
		}

		if e.Features.Supports.RESTCapabilities.TickerBatching != e.Config.Features.Supports.RESTCapabilities.TickerBatching {
			e.Config.Features.Supports.RESTCapabilities.TickerBatching = e.Features.Supports.RESTCapabilities.TickerBatching
		}

		if e.Features.Supports.Websocket != e.Config.Features.Supports.Websocket {
			e.Config.Features.Supports.Websocket = e.Features.Supports.Websocket
		}

		e.Features.Enabled.AutoPairUpdates = e.Config.Features.Enabled.AutoPairUpdates
	}
}

// SetAPICredentialDefaults sets the API Credential validator defaults
func (e *Base) SetAPICredentialDefaults() {
	// Exchange hardcoded settings take precedence and overwrite the config settings
	if e.Config.API.CredentialsValidator == nil {
		e.Config.API.CredentialsValidator = new(config.APICredentialsValidatorConfig)
	}
	if e.Config.API.CredentialsValidator.RequiresKey != e.API.CredentialsValidator.RequiresKey {
		e.Config.API.CredentialsValidator.RequiresKey = e.API.CredentialsValidator.RequiresKey
	}

	if e.Config.API.CredentialsValidator.RequiresSecret != e.API.CredentialsValidator.RequiresSecret {
		e.Config.API.CredentialsValidator.RequiresSecret = e.API.CredentialsValidator.RequiresSecret
	}

	if e.Config.API.CredentialsValidator.RequiresBase64DecodeSecret != e.API.CredentialsValidator.RequiresBase64DecodeSecret {
		e.Config.API.CredentialsValidator.RequiresBase64DecodeSecret = e.API.CredentialsValidator.RequiresBase64DecodeSecret
	}

	if e.Config.API.CredentialsValidator.RequiresClientID != e.API.CredentialsValidator.RequiresClientID {
		e.Config.API.CredentialsValidator.RequiresClientID = e.API.CredentialsValidator.RequiresClientID
	}

	if e.Config.API.CredentialsValidator.RequiresPEM != e.API.CredentialsValidator.RequiresPEM {
		e.Config.API.CredentialsValidator.RequiresPEM = e.API.CredentialsValidator.RequiresPEM
	}
}

// SupportsRESTTickerBatchUpdates returns whether or not the
// exhange supports REST batch ticker fetching
func (e *Base) SupportsRESTTickerBatchUpdates() bool {
	return e.Features.Supports.RESTCapabilities.TickerBatching
}

// SupportsAutoPairUpdates returns whether or not the exchange supports
// auto currency pair updating
func (e *Base) SupportsAutoPairUpdates() bool {
	if e.Features.Supports.RESTCapabilities.AutoPairUpdates ||
		e.Features.Supports.WebsocketCapabilities.AutoPairUpdates {
		return true
	}
	return false
}

// GetLastPairsUpdateTime returns the unix timestamp of when the exchanges
// currency pairs were last updated
func (e *Base) GetLastPairsUpdateTime() int64 {
	return e.CurrencyPairs.LastUpdated
}

// SetAssetTypes checks the exchange asset types (whether it supports SPOT,
// Binary or Futures) and sets it to a default setting if it doesn't exist
func (e *Base) SetAssetTypes() {
	if e.Config.CurrencyPairs.AssetTypes.JoinToString(",") == "" {
		e.Config.CurrencyPairs.AssetTypes = e.CurrencyPairs.AssetTypes
	} else if e.Config.CurrencyPairs.AssetTypes.JoinToString(",") != e.CurrencyPairs.AssetTypes.JoinToString(",") {
		e.Config.CurrencyPairs.AssetTypes = e.CurrencyPairs.AssetTypes
	}
}

// GetAssetTypes returns the available asset types for an individual exchange
func (e *Base) GetAssetTypes() asset.Items {
	return e.CurrencyPairs.AssetTypes
}

// GetPairAssetType returns the associated asset type for the currency pair
func (e *Base) GetPairAssetType(c currency.Pair) (asset.Item, error) {
	assetTypes := e.GetAssetTypes()
	for i := range assetTypes {
		if e.GetEnabledPairs(assetTypes[i]).Contains(c, true) {
			return assetTypes[i], nil
		}
	}
	return "", errors.New("asset type not associated with currency pair")
}

// GetClientBankAccounts returns banking details associated with
// a client for withdrawal purposes
func (e *Base) GetClientBankAccounts(exchangeName, withdrawalCurrency string) (*banking.Account, error) {
	cfg := config.GetConfig()
	return cfg.GetClientBankAccounts(exchangeName, withdrawalCurrency)
}

// GetExchangeBankAccounts returns banking details associated with an
// exchange for funding purposes
func (e *Base) GetExchangeBankAccounts(id, depositCurrency string) (*banking.Account, error) {
	cfg := config.GetConfig()
	return cfg.GetExchangeBankAccounts(e.Name, id, depositCurrency)
}

// SetCurrencyPairFormat checks the exchange request and config currency pair
// formats and syncs it with the exchanges SetDefault settings
func (e *Base) SetCurrencyPairFormat() {
	if e.Config.CurrencyPairs == nil {
		e.Config.CurrencyPairs = new(currency.PairsManager)
	}

	e.Config.CurrencyPairs.UseGlobalFormat = e.CurrencyPairs.UseGlobalFormat
	if e.Config.CurrencyPairs.UseGlobalFormat {
		e.Config.CurrencyPairs.RequestFormat = e.CurrencyPairs.RequestFormat
		e.Config.CurrencyPairs.ConfigFormat = e.CurrencyPairs.ConfigFormat
		return
	}

	if e.Config.CurrencyPairs.ConfigFormat != nil {
		e.Config.CurrencyPairs.ConfigFormat = nil
	}
	if e.Config.CurrencyPairs.RequestFormat != nil {
		e.Config.CurrencyPairs.RequestFormat = nil
	}

	assetTypes := e.GetAssetTypes()
	for x := range assetTypes {
		if e.Config.CurrencyPairs.Get(assetTypes[x]) == nil {
			r := e.CurrencyPairs.Get(assetTypes[x])
			if r == nil {
				continue
			}
			e.Config.CurrencyPairs.Store(assetTypes[x], *e.CurrencyPairs.Get(assetTypes[x]))
		}
	}
}

// SetConfigPairs sets the exchanges currency pairs to the pairs set in the config
func (e *Base) SetConfigPairs() {
	assetTypes := e.GetAssetTypes()
	for x := range assetTypes {
		cfgPS := e.Config.CurrencyPairs.Get(assetTypes[x])
		if cfgPS == nil {
			continue
		}
		if e.Config.CurrencyPairs.UseGlobalFormat {
			e.CurrencyPairs.StorePairs(assetTypes[x], cfgPS.Available, false)
			e.CurrencyPairs.StorePairs(assetTypes[x], cfgPS.Enabled, true)
			continue
		}
		exchPS := e.CurrencyPairs.Get(assetTypes[x])
		cfgPS.ConfigFormat = exchPS.ConfigFormat
		cfgPS.RequestFormat = exchPS.RequestFormat
		e.CurrencyPairs.StorePairs(assetTypes[x], cfgPS.Available, false)
		e.CurrencyPairs.StorePairs(assetTypes[x], cfgPS.Enabled, true)
	}
}

// GetAuthenticatedAPISupport returns whether the exchange supports
// authenticated API requests
func (e *Base) GetAuthenticatedAPISupport(endpoint uint8) bool {
	switch endpoint {
	case RestAuthentication:
		return e.API.AuthenticatedSupport
	case WebsocketAuthentication:
		return e.API.AuthenticatedWebsocketSupport
	}
	return false
}

// GetName is a method that returns the name of the exchange base
func (e *Base) GetName() string {
	return e.Name
}

// GetEnabledFeatures returns the exchanges enabled features
func (e *Base) GetEnabledFeatures() FeaturesEnabled {
	return e.Features.Enabled
}

// GetSupportedFeatures returns the exchanges supported features
func (e *Base) GetSupportedFeatures() FeaturesSupported {
	return e.Features.Supports
}

// GetPairFormat returns the pair format based on the exchange and
// asset type
func (e *Base) GetPairFormat(assetType asset.Item, requestFormat bool) currency.PairFormat {
	if e.CurrencyPairs.UseGlobalFormat {
		if requestFormat {
			return *e.CurrencyPairs.RequestFormat
		}
		return *e.CurrencyPairs.ConfigFormat
	}

	if requestFormat {
		return *e.CurrencyPairs.Get(assetType).RequestFormat
	}
	return *e.CurrencyPairs.Get(assetType).ConfigFormat
}

// GetEnabledPairs is a method that returns the enabled currency pairs of
// the exchange by asset type
func (e *Base) GetEnabledPairs(assetType asset.Item) currency.Pairs {
	format := e.GetPairFormat(assetType, false)
	pairs := e.CurrencyPairs.GetPairs(assetType, true)
	return pairs.Format(format.Delimiter, format.Index, format.Uppercase)
}

// GetRequestFormattedPairAndAssetType is a method that returns the enabled currency pair of
// along with its asset type. Only use when there is no chance of the same name crossing over
func (e *Base) GetRequestFormattedPairAndAssetType(p string) (currency.Pair, asset.Item, error) {
	assetTypes := e.GetAssetTypes()
	var response currency.Pair
	for i := range assetTypes {
		format := e.GetPairFormat(assetTypes[i], true)
		pairs := e.CurrencyPairs.GetPairs(assetTypes[i], true)
		for j := range pairs {
			formattedPair := pairs[j].Format(format.Delimiter, format.Uppercase)
			if strings.EqualFold(formattedPair.String(), p) {
				return formattedPair, assetTypes[i], nil
			}
		}
	}
	return response, "", errors.New("pair not found: " + p)
}

// GetAvailablePairs is a method that returns the available currency pairs
// of the exchange by asset type
func (e *Base) GetAvailablePairs(assetType asset.Item) currency.Pairs {
	format := e.GetPairFormat(assetType, false)
	pairs := e.CurrencyPairs.GetPairs(assetType, false)
	return pairs.Format(format.Delimiter, format.Index, format.Uppercase)
}

// SupportsPair returns true or not whether a currency pair exists in the
// exchange available currencies or not
func (e *Base) SupportsPair(p currency.Pair, enabledPairs bool, assetType asset.Item) bool {
	if enabledPairs {
		return e.GetEnabledPairs(assetType).Contains(p, false)
	}
	return e.GetAvailablePairs(assetType).Contains(p, false)
}

// FormatExchangeCurrencies returns a string containing
// the exchanges formatted currency pairs
func (e *Base) FormatExchangeCurrencies(pairs []currency.Pair, assetType asset.Item) (string, error) {
	var currencyItems strings.Builder
	pairFmt := e.GetPairFormat(assetType, true)

	for x := range pairs {
		currencyItems.WriteString(e.FormatExchangeCurrency(pairs[x], assetType).String())
		if x == len(pairs)-1 {
			continue
		}
		currencyItems.WriteString(pairFmt.Separator)
	}

	if currencyItems.Len() == 0 {
		return "", errors.New("returned empty string")
	}
	return currencyItems.String(), nil
}

// FormatExchangeCurrency is a method that formats and returns a currency pair
// based on the user currency display preferences
func (e *Base) FormatExchangeCurrency(p currency.Pair, assetType asset.Item) currency.Pair {
	pairFmt := e.GetPairFormat(assetType, true)
	return p.Format(pairFmt.Delimiter, pairFmt.Uppercase)
}

// SetEnabled is a method that sets if the exchange is enabled
func (e *Base) SetEnabled(enabled bool) {
	e.Enabled = enabled
}

// IsEnabled is a method that returns if the current exchange is enabled
func (e *Base) IsEnabled() bool {
	return e.Enabled
}

// SetAPIKeys is a method that sets the current API keys for the exchange
func (e *Base) SetAPIKeys(apiKey, apiSecret, clientID string) {
	e.API.Credentials.Key = apiKey
	e.API.Credentials.ClientID = clientID

	if e.API.CredentialsValidator.RequiresBase64DecodeSecret {
		result, err := crypto.Base64Decode(apiSecret)
		if err != nil {
			e.API.AuthenticatedSupport = false
			e.API.AuthenticatedWebsocketSupport = false
			log.Warnf(log.ExchangeSys,
				warningBase64DecryptSecretKeyFailed,
				e.Name)
			return
		}
		e.API.Credentials.Secret = string(result)
	} else {
		e.API.Credentials.Secret = apiSecret
	}
}

// SetupDefaults sets the exchange settings based on the supplied config
func (e *Base) SetupDefaults(exch *config.ExchangeConfig) error {
	e.Enabled = true
	e.LoadedByConfig = true
	e.Config = exch
	e.Verbose = exch.Verbose

	e.API.AuthenticatedSupport = exch.API.AuthenticatedSupport
	e.API.AuthenticatedWebsocketSupport = exch.API.AuthenticatedWebsocketSupport
	if e.API.AuthenticatedSupport || e.API.AuthenticatedWebsocketSupport {
		e.SetAPIKeys(exch.API.Credentials.Key,
			exch.API.Credentials.Secret,
			exch.API.Credentials.ClientID)
	}

	if exch.HTTPTimeout <= time.Duration(0) {
		exch.HTTPTimeout = DefaultHTTPTimeout
	} else {
		e.SetHTTPClientTimeout(exch.HTTPTimeout)
	}

	if exch.CurrencyPairs == nil {
		exch.CurrencyPairs = new(currency.PairsManager)
	}

	e.HTTPDebugging = exch.HTTPDebugging
	e.SetHTTPClientUserAgent(exch.HTTPUserAgent)
	e.SetAssetTypes()
	e.SetCurrencyPairFormat()
	e.SetConfigPairs()
	e.SetFeatureDefaults()
	e.SetAPIURL()
	e.SetAPICredentialDefaults()
	e.SetClientProxyAddress(exch.ProxyAddress)
	e.BaseCurrencies = exch.BaseCurrencies

	if e.Features.Supports.Websocket {
		return e.Websocket.Initialise()
	}
	return nil
}

// AllowAuthenticatedRequest checks to see if the required fields have been set
// before sending an authenticated API request
func (e *Base) AllowAuthenticatedRequest() bool {
	if e.SkipAuthCheck {
		return true
	}

	// Individual package usage, allow request if API credentials are valid a
	// and without needing to set AuthenticatedSupport to true
	if !e.LoadedByConfig {
		return e.ValidateAPICredentials()
	}

	// Bot usage, AuthenticatedSupport can be disabled by user if desired, so
	// don't allow authenticated requests.
	if !e.API.AuthenticatedSupport && !e.API.AuthenticatedWebsocketSupport {
		return false
	}

	// Check to see if the user has enabled AuthenticatedSupport, but has
	// invalid API credentials set and loaded by config
	return e.ValidateAPICredentials()
}

// ValidateAPICredentials validates the exchanges API credentials
func (e *Base) ValidateAPICredentials() bool {
	if e.API.CredentialsValidator.RequiresKey {
		if e.API.Credentials.Key == "" ||
			e.API.Credentials.Key == config.DefaultAPIKey {
			log.Warnf(log.ExchangeSys,
				"exchange %s requires API key but default/empty one set",
				e.Name)
			return false
		}
	}

	if e.API.CredentialsValidator.RequiresSecret {
		if e.API.Credentials.Secret == "" ||
			e.API.Credentials.Secret == config.DefaultAPISecret {
			log.Warnf(log.ExchangeSys,
				"exchange %s requires API secret but default/empty one set",
				e.Name)
			return false
		}
	}

	if e.API.CredentialsValidator.RequiresPEM {
		if e.API.Credentials.PEMKey == "" ||
			strings.Contains(e.API.Credentials.PEMKey, "JUSTADUMMY") {
			log.Warnf(log.ExchangeSys,
				"exchange %s requires API PEM key but default/empty one set",
				e.Name)
			return false
		}
	}

	if e.API.CredentialsValidator.RequiresClientID {
		if e.API.Credentials.ClientID == "" ||
			e.API.Credentials.ClientID == config.DefaultAPIClientID {
			log.Warnf(log.ExchangeSys,
				"exchange %s requires API ClientID but default/empty one set",
				e.Name)
			return false
		}
	}

	if e.API.CredentialsValidator.RequiresBase64DecodeSecret && !e.LoadedByConfig {
		_, err := crypto.Base64Decode(e.API.Credentials.Secret)
		if err != nil {
			log.Warnf(log.ExchangeSys,
				"exchange %s API secret base64 decode failed: %s",
				e.Name, err)
			return false
		}
	}
	return true
}

// SetPairs sets the exchange currency pairs for either enabledPairs or
// availablePairs
func (e *Base) SetPairs(pairs currency.Pairs, assetType asset.Item, enabled bool) error {
	if len(pairs) == 0 {
		return fmt.Errorf("%s SetPairs error - pairs is empty", e.Name)
	}

	pairFmt := e.GetPairFormat(assetType, false)
	var newPairs currency.Pairs
	for x := range pairs {
		newPairs = append(newPairs, pairs[x].Format(pairFmt.Delimiter,
			pairFmt.Uppercase))
	}

	e.CurrencyPairs.StorePairs(assetType, newPairs, enabled)
	e.Config.CurrencyPairs.StorePairs(assetType, newPairs, enabled)
	return nil
}

// UpdatePairs updates the exchange currency pairs for either enabledPairs or
// availablePairs
func (e *Base) UpdatePairs(exchangeProducts currency.Pairs, assetType asset.Item, enabled, force bool) error {
	if len(exchangeProducts) == 0 {
		return fmt.Errorf("%s UpdatePairs error - exchangeProducts is empty", e.Name)
	}

	exchangeProducts = exchangeProducts.Upper()
	var products currency.Pairs
	for x := range exchangeProducts {
		if exchangeProducts[x].String() == "" {
			continue
		}
		products = append(products, exchangeProducts[x])
	}

	var newPairs, removedPairs currency.Pairs
	var updateType string
	targetPairs := e.CurrencyPairs.GetPairs(assetType, enabled)

	if enabled {
		newPairs, removedPairs = targetPairs.FindDifferences(products)
		updateType = "enabled"
	} else {
		newPairs, removedPairs = targetPairs.FindDifferences(products)
		updateType = "available"
	}

	if force || len(newPairs) > 0 || len(removedPairs) > 0 {
		if force {
			log.Debugf(log.ExchangeSys,
				"%s forced update of %s [%v] pairs.", e.Name, updateType,
				strings.ToUpper(assetType.String()))
		} else {
			if len(newPairs) > 0 {
				log.Debugf(log.ExchangeSys,
					"%s Updating pairs [%v] - New: %s.\n", e.Name,
					strings.ToUpper(assetType.String()), newPairs)
			}
			if len(removedPairs) > 0 {
				log.Debugf(log.ExchangeSys,
					"%s Updating pairs [%v] - Removed: %s.\n", e.Name,
					strings.ToUpper(assetType.String()), removedPairs)
			}
		}
		e.Config.CurrencyPairs.StorePairs(assetType, products, enabled)
		e.CurrencyPairs.StorePairs(assetType, products, enabled)
	}
	return nil
}

// SetAPIURL sets configuration API URL for an exchange
func (e *Base) SetAPIURL() error {
	if e.Config.API.Endpoints.URL == "" || e.Config.API.Endpoints.URLSecondary == "" {
		return fmt.Errorf("exchange %s: SetAPIURL error. URL vals are empty", e.Name)
	}

	checkInsecureEndpoint := func(endpoint string) {
		if strings.Contains(endpoint, "https") {
			return
		}
		log.Warnf(log.ExchangeSys,
			"%s is using HTTP instead of HTTPS [%s] for API functionality, an"+
				" attacker could eavesdrop on this connection. Use at your"+
				" own risk.",
			e.Name, endpoint)
	}

	if e.Config.API.Endpoints.URL != config.APIURLNonDefaultMessage {
		e.API.Endpoints.URL = e.Config.API.Endpoints.URL
		checkInsecureEndpoint(e.API.Endpoints.URL)
	}
	if e.Config.API.Endpoints.URLSecondary != config.APIURLNonDefaultMessage {
		e.API.Endpoints.URLSecondary = e.Config.API.Endpoints.URLSecondary
		checkInsecureEndpoint(e.API.Endpoints.URLSecondary)
	}
	return nil
}

// GetAPIURL returns the set API URL
func (e *Base) GetAPIURL() string {
	return e.API.Endpoints.URL
}

// GetSecondaryAPIURL returns the set Secondary API URL
func (e *Base) GetSecondaryAPIURL() string {
	return e.API.Endpoints.URLSecondary
}

// GetAPIURLDefault returns exchange default URL
func (e *Base) GetAPIURLDefault() string {
	return e.API.Endpoints.URLDefault
}

// GetAPIURLSecondaryDefault returns exchange default secondary URL
func (e *Base) GetAPIURLSecondaryDefault() string {
	return e.API.Endpoints.URLSecondaryDefault
}

// SupportsWebsocket returns whether or not the exchange supports
// websocket
func (e *Base) SupportsWebsocket() bool {
	return e.Features.Supports.Websocket
}

// SupportsREST returns whether or not the exchange supports
// REST
func (e *Base) SupportsREST() bool {
	return e.Features.Supports.REST
}

// IsWebsocketEnabled returns whether or not the exchange has its
// websocket client enabled
func (e *Base) IsWebsocketEnabled() bool {
	if e.Websocket != nil {
		return e.Websocket.IsEnabled()
	}
	return false
}

// GetWithdrawPermissions passes through the exchange's withdraw permissions
func (e *Base) GetWithdrawPermissions() uint32 {
	return e.Features.Supports.WithdrawPermissions
}

// SupportsWithdrawPermissions compares the supplied permissions with the exchange's to verify they're supported
func (e *Base) SupportsWithdrawPermissions(permissions uint32) bool {
	exchangePermissions := e.GetWithdrawPermissions()
	return permissions&exchangePermissions == permissions
}

// FormatWithdrawPermissions will return each of the exchange's compatible withdrawal methods in readable form
func (e *Base) FormatWithdrawPermissions() string {
	var services []string
	for i := 0; i < 32; i++ {
		var check uint32 = 1 << uint32(i)
		if e.GetWithdrawPermissions()&check != 0 {
			switch check {
			case AutoWithdrawCrypto:
				services = append(services, AutoWithdrawCryptoText)
			case AutoWithdrawCryptoWithAPIPermission:
				services = append(services, AutoWithdrawCryptoWithAPIPermissionText)
			case AutoWithdrawCryptoWithSetup:
				services = append(services, AutoWithdrawCryptoWithSetupText)
			case WithdrawCryptoWith2FA:
				services = append(services, WithdrawCryptoWith2FAText)
			case WithdrawCryptoWithSMS:
				services = append(services, WithdrawCryptoWithSMSText)
			case WithdrawCryptoWithEmail:
				services = append(services, WithdrawCryptoWithEmailText)
			case WithdrawCryptoWithWebsiteApproval:
				services = append(services, WithdrawCryptoWithWebsiteApprovalText)
			case WithdrawCryptoWithAPIPermission:
				services = append(services, WithdrawCryptoWithAPIPermissionText)
			case AutoWithdrawFiat:
				services = append(services, AutoWithdrawFiatText)
			case AutoWithdrawFiatWithAPIPermission:
				services = append(services, AutoWithdrawFiatWithAPIPermissionText)
			case AutoWithdrawFiatWithSetup:
				services = append(services, AutoWithdrawFiatWithSetupText)
			case WithdrawFiatWith2FA:
				services = append(services, WithdrawFiatWith2FAText)
			case WithdrawFiatWithSMS:
				services = append(services, WithdrawFiatWithSMSText)
			case WithdrawFiatWithEmail:
				services = append(services, WithdrawFiatWithEmailText)
			case WithdrawFiatWithWebsiteApproval:
				services = append(services, WithdrawFiatWithWebsiteApprovalText)
			case WithdrawFiatWithAPIPermission:
				services = append(services, WithdrawFiatWithAPIPermissionText)
			case WithdrawCryptoViaWebsiteOnly:
				services = append(services, WithdrawCryptoViaWebsiteOnlyText)
			case WithdrawFiatViaWebsiteOnly:
				services = append(services, WithdrawFiatViaWebsiteOnlyText)
			case NoFiatWithdrawals:
				services = append(services, NoFiatWithdrawalsText)
			default:
				services = append(services, fmt.Sprintf("%s[1<<%v]", UnknownWithdrawalTypeText, i))
			}
		}
	}
	if len(services) > 0 {
		return strings.Join(services, " & ")
	}

	return NoAPIWithdrawalMethodsText
}

// SupportsAsset whether or not the supplied asset is supported
// by the exchange
func (e *Base) SupportsAsset(a asset.Item) bool {
	return e.CurrencyPairs.AssetTypes.Contains(a)
}

// PrintEnabledPairs prints the exchanges enabled asset pairs
func (e *Base) PrintEnabledPairs() {
	for k, v := range e.CurrencyPairs.Pairs {
		log.Infof(log.ExchangeSys, "%s Asset type %v:\n\t Enabled pairs: %v",
			e.Name, strings.ToUpper(k.String()), v.Enabled)
	}
}

// GetBase returns the exchange base
func (e *Base) GetBase() *Base { return e }

// CheckTransientError catches transient errors and returns nil if found, used
// for validation of API credentials
func (e *Base) CheckTransientError(err error) error {
	if _, ok := err.(net.Error); ok {
		log.Warnf(log.ExchangeSys,
			"%s net error captured, will not disable authentication %s",
			e.Name,
			err)
		return nil
	}
	return err
}

// DisableRateLimiter disables the rate limiting system for the exchange
func (e *Base) DisableRateLimiter() error {
	return e.Requester.DisableRateLimiter()
}

// EnableRateLimiter enables the rate limiting system for the exchange
func (e *Base) EnableRateLimiter() error {
	return e.Requester.EnableRateLimiter()
}
