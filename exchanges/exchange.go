package exchange

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
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
			&http.Client{Transport: new(http.Transport)})
	}
}

// SetHTTPClientTimeout sets the timeout value for the exchanges HTTP Client and
// also the underlying transports idle connection timeout
func (e *Base) SetHTTPClientTimeout(t time.Duration) error {
	e.checkAndInitRequester()
	e.Requester.HTTPClient.Timeout = t
	tr, ok := e.Requester.HTTPClient.Transport.(*http.Transport)
	if !ok {
		return errors.New("transport not set, cannot set timeout")
	}
	tr.IdleConnTimeout = t
	return nil
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
	if addr == "" {
		return nil
	}
	proxy, err := url.Parse(addr)
	if err != nil {
		return fmt.Errorf("exchange.go - setting proxy address error %s",
			err)
	}

	err = e.Requester.SetProxy(proxy)
	if err != nil {
		return err
	}

	if e.Websocket != nil {
		err = e.Websocket.SetProxyAddress(addr)
		if err != nil {
			return err
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

		if e.IsSaveTradeDataEnabled() != e.Config.Features.Enabled.SaveTradeData {
			e.SetSaveTradeDataStatus(e.Config.Features.Enabled.SaveTradeData)
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

// GetAssetTypes returns the available asset types for an individual exchange
func (e *Base) GetAssetTypes() asset.Items {
	return e.CurrencyPairs.GetAssetTypes()
}

// GetPairAssetType returns the associated asset type for the currency pair
// This method is only useful for exchanges that have pair names with multiple delimiters (BTC-USD-0626)
// Helpful if the exchange has only a single asset type but in that case the asset type can be hard coded
func (e *Base) GetPairAssetType(c currency.Pair) (asset.Item, error) {
	assetTypes := e.GetAssetTypes()
	for i := range assetTypes {
		avail, err := e.GetAvailablePairs(assetTypes[i])
		if err != nil {
			return "", err
		}
		if avail.Contains(c, true) {
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
		if _, err := e.Config.CurrencyPairs.Get(assetTypes[x]); err != nil {
			ps, err := e.CurrencyPairs.Get(assetTypes[x])
			if err != nil {
				continue
			}
			e.Config.CurrencyPairs.Store(assetTypes[x], *ps)
		}
	}
}

// SetConfigPairs sets the exchanges currency pairs to the pairs set in the config
func (e *Base) SetConfigPairs() error {
	assetTypes := e.Config.CurrencyPairs.GetAssetTypes()
	exchangeAssets := e.CurrencyPairs.GetAssetTypes()
	for x := range assetTypes {
		if !exchangeAssets.Contains(assetTypes[x]) {
			log.Warnf(log.ExchangeSys,
				"%s exchange asset type %s unsupported, please manually remove from configuration",
				e.Name,
				assetTypes[x])
		}
		cfgPS, err := e.Config.CurrencyPairs.Get(assetTypes[x])
		if err != nil {
			return err
		}

		var enabledAsset bool
		if e.Config.CurrencyPairs.IsAssetEnabled(assetTypes[x]) == nil {
			enabledAsset = true
		}
		e.CurrencyPairs.SetAssetEnabled(assetTypes[x], enabledAsset)

		if e.Config.CurrencyPairs.UseGlobalFormat {
			e.CurrencyPairs.StorePairs(assetTypes[x], cfgPS.Available, false)
			e.CurrencyPairs.StorePairs(assetTypes[x], cfgPS.Enabled, true)
			continue
		}
		exchPS, err := e.CurrencyPairs.Get(assetTypes[x])
		if err != nil {
			return err
		}
		cfgPS.ConfigFormat = exchPS.ConfigFormat
		cfgPS.RequestFormat = exchPS.RequestFormat
		e.CurrencyPairs.StorePairs(assetTypes[x], cfgPS.Available, false)
		e.CurrencyPairs.StorePairs(assetTypes[x], cfgPS.Enabled, true)
	}
	return nil
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
func (e *Base) GetPairFormat(assetType asset.Item, requestFormat bool) (currency.PairFormat, error) {
	if e.CurrencyPairs.UseGlobalFormat {
		if requestFormat {
			if e.CurrencyPairs.RequestFormat == nil {
				return currency.PairFormat{},
					errors.New("global request format is nil")
			}
			return *e.CurrencyPairs.RequestFormat, nil
		}

		if e.CurrencyPairs.ConfigFormat == nil {
			return currency.PairFormat{},
				errors.New("global config format is nil")
		}
		return *e.CurrencyPairs.ConfigFormat, nil
	}

	ps, err := e.CurrencyPairs.Get(assetType)
	if err != nil {
		return currency.PairFormat{}, err
	}

	if requestFormat {
		if ps.RequestFormat == nil {
			return currency.PairFormat{},
				errors.New("asset type request format is nil")
		}
		return *ps.RequestFormat, nil
	}

	if ps.ConfigFormat == nil {
		return currency.PairFormat{},
			errors.New("asset type config format is nil")
	}
	return *ps.ConfigFormat, nil
}

// GetEnabledPairs is a method that returns the enabled currency pairs of
// the exchange by asset type, if the asset type is disabled this will return no
// enabled pairs
func (e *Base) GetEnabledPairs(a asset.Item) (currency.Pairs, error) {
	err := e.CurrencyPairs.IsAssetEnabled(a)
	if err != nil {
		return nil, nil
	}
	format, err := e.GetPairFormat(a, false)
	if err != nil {
		return nil, err
	}
	enabledpairs, err := e.CurrencyPairs.GetPairs(a, true)
	if err != nil {
		return nil, err
	}
	return enabledpairs.Format(format.Delimiter,
			format.Index,
			format.Uppercase),
		nil
}

// GetRequestFormattedPairAndAssetType is a method that returns the enabled currency pair of
// along with its asset type. Only use when there is no chance of the same name crossing over
func (e *Base) GetRequestFormattedPairAndAssetType(p string) (currency.Pair, asset.Item, error) {
	assetTypes := e.GetAssetTypes()
	var response currency.Pair
	for i := range assetTypes {
		format, err := e.GetPairFormat(assetTypes[i], true)
		if err != nil {
			return response, assetTypes[i], err
		}

		pairs, err := e.CurrencyPairs.GetPairs(assetTypes[i], true)
		if err != nil {
			return response, assetTypes[i], err
		}

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
func (e *Base) GetAvailablePairs(assetType asset.Item) (currency.Pairs, error) {
	format, err := e.GetPairFormat(assetType, false)
	if err != nil {
		return nil, err
	}
	pairs, err := e.CurrencyPairs.GetPairs(assetType, false)
	if err != nil {
		return nil, err
	}
	return pairs.Format(format.Delimiter, format.Index, format.Uppercase), nil
}

// SupportsPair returns true or not whether a currency pair exists in the
// exchange available currencies or not
func (e *Base) SupportsPair(p currency.Pair, enabledPairs bool, assetType asset.Item) error {
	if enabledPairs {
		pairs, err := e.GetEnabledPairs(assetType)
		if err != nil {
			return err
		}
		if pairs.Contains(p, false) {
			return nil
		}
		return errors.New("pair not supported")
	}

	avail, err := e.GetAvailablePairs(assetType)
	if err != nil {
		return err
	}
	if avail.Contains(p, false) {
		return nil
	}
	return errors.New("pair not supported")
}

// FormatExchangeCurrencies returns a string containing
// the exchanges formatted currency pairs
func (e *Base) FormatExchangeCurrencies(pairs []currency.Pair, assetType asset.Item) (string, error) {
	var currencyItems strings.Builder
	pairFmt, err := e.GetPairFormat(assetType, true)
	if err != nil {
		return "", err
	}

	for x := range pairs {
		format, err := e.FormatExchangeCurrency(pairs[x], assetType)
		if err != nil {
			return "", err
		}
		currencyItems.WriteString(format.String())
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
func (e *Base) FormatExchangeCurrency(p currency.Pair, assetType asset.Item) (currency.Pair, error) {
	pairFmt, err := e.GetPairFormat(assetType, true)
	if err != nil {
		return currency.Pair{}, err
	}
	return p.Format(pairFmt.Delimiter, pairFmt.Uppercase), nil
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
	}

	err := e.SetHTTPClientTimeout(exch.HTTPTimeout)
	if err != nil {
		return err
	}

	if exch.CurrencyPairs == nil {
		exch.CurrencyPairs = new(currency.PairsManager)
	}

	e.HTTPDebugging = exch.HTTPDebugging
	e.SetHTTPClientUserAgent(exch.HTTPUserAgent)
	e.SetCurrencyPairFormat()

	err = e.SetConfigPairs()
	if err != nil {
		return err
	}

	e.SetFeatureDefaults()

	if e.API.Endpoints == nil {
		e.API.Endpoints = e.NewEndpoints()
	}

	err = e.SetAPIURL()
	if err != nil {
		return err
	}

	e.SetAPICredentialDefaults()

	err = e.SetClientProxyAddress(exch.ProxyAddress)
	if err != nil {
		return err
	}
	e.BaseCurrencies = exch.BaseCurrencies
	e.OrderbookVerificationBypass = exch.OrderbookConfig.VerificationBypass
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

	pairFmt, err := e.GetPairFormat(assetType, false)
	if err != nil {
		return err
	}

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
	exchangeProducts = exchangeProducts.Upper()
	var products currency.Pairs
	for x := range exchangeProducts {
		if exchangeProducts[x].String() == "" {
			continue
		}
		products = append(products, exchangeProducts[x])
	}

	var updateType string
	targetPairs, err := e.CurrencyPairs.GetPairs(assetType, enabled)
	if err != nil {
		return err
	}

	if enabled {
		updateType = "enabled"
	} else {
		updateType = "available"
	}

	newPairs, removedPairs := targetPairs.FindDifferences(products)
	if force || len(newPairs) > 0 || len(removedPairs) > 0 {
		if force {
			log.Debugf(log.ExchangeSys,
				"%s forced update of %s [%v] pairs.",
				e.Name,
				updateType,
				strings.ToUpper(assetType.String()))
		} else {
			if len(newPairs) > 0 {
				log.Debugf(log.ExchangeSys,
					"%s Updating %s pairs [%v] - Added: %s.\n",
					e.Name,
					updateType,
					strings.ToUpper(assetType.String()),
					newPairs)
			}
			if len(removedPairs) > 0 {
				log.Debugf(log.ExchangeSys,
					"%s Updating %s pairs [%v] - Removed: %s.\n",
					e.Name,
					updateType,
					strings.ToUpper(assetType.String()),
					removedPairs)
			}
		}

		e.Config.CurrencyPairs.StorePairs(assetType, products, enabled)
		e.CurrencyPairs.StorePairs(assetType, products, enabled)

		if !enabled {
			// If available pairs are changed we will remove currency pair items
			// that are still included in the enabled pairs list.
			enabledPairs, err := e.CurrencyPairs.GetPairs(assetType, true)
			if err == nil {
				return nil
			}
			_, remove := enabledPairs.FindDifferences(products)
			for i := range remove {
				enabledPairs = enabledPairs.Remove(remove[i])
			}

			if len(remove) > 0 {
				log.Debugf(log.ExchangeSys,
					"%s Checked and updated enabled pairs [%v] - Removed: %s.\n",
					e.Name,
					strings.ToUpper(assetType.String()),
					remove)
				e.Config.CurrencyPairs.StorePairs(assetType, enabledPairs, true)
				e.CurrencyPairs.StorePairs(assetType, enabledPairs, true)
			}
		}
	}
	return nil
}

// SetAPIURL sets configuration API URL for an exchange
func (e *Base) SetAPIURL() error {
	checkInsecureEndpoint := func(endpoint string) {
		if strings.Contains(endpoint, "https") || strings.Contains(endpoint, "wss") {
			return
		}
		log.Warnf(log.ExchangeSys,
			"%s is using HTTP instead of HTTPS or WS instead of WSS [%s] for API functionality, an"+
				" attacker could eavesdrop on this connection. Use at your"+
				" own risk.",
			e.Name, endpoint)
	}
	var err error
	if e.Config.API.OldEndPoints != nil {
		if e.Config.API.OldEndPoints.URL != "" && e.Config.API.OldEndPoints.URL != config.APIURLNonDefaultMessage {
			err = e.API.Endpoints.SetRunning(RestSpot.String(), e.Config.API.OldEndPoints.URL)
			if err != nil {
				return err
			}
			checkInsecureEndpoint(e.Config.API.OldEndPoints.URL)
		}
		if e.Config.API.OldEndPoints.URLSecondary != "" && e.Config.API.OldEndPoints.URLSecondary != config.APIURLNonDefaultMessage {
			err = e.API.Endpoints.SetRunning(RestSpotSupplementary.String(), e.Config.API.OldEndPoints.URLSecondary)
			if err != nil {
				return err
			}
			checkInsecureEndpoint(e.Config.API.OldEndPoints.URLSecondary)
		}
		if e.Config.API.OldEndPoints.WebsocketURL != "" && e.Config.API.OldEndPoints.WebsocketURL != config.WebsocketURLNonDefaultMessage {
			err = e.API.Endpoints.SetRunning(WebsocketSpot.String(), e.Config.API.OldEndPoints.WebsocketURL)
			if err != nil {
				return err
			}
			checkInsecureEndpoint(e.Config.API.OldEndPoints.WebsocketURL)
		}
		e.Config.API.OldEndPoints = nil
	} else if e.Config.API.Endpoints != nil {
		for key, val := range e.Config.API.Endpoints {
			if val == "" ||
				val == config.APIURLNonDefaultMessage ||
				val == config.WebsocketURLNonDefaultMessage {
				continue
			}
			checkInsecureEndpoint(val)
			err = e.API.Endpoints.SetRunning(key, val)
			if err != nil {
				return err
			}
		}
	}
	runningMap := e.API.Endpoints.GetURLMap()
	e.Config.API.Endpoints = runningMap
	return nil
}

// SupportsREST returns whether or not the exchange supports
// REST
func (e *Base) SupportsREST() bool {
	return e.Features.Supports.REST
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
	_, ok := e.CurrencyPairs.Pairs[a]
	return ok
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

// StoreAssetPairFormat initialises and stores a defined asset format
func (e *Base) StoreAssetPairFormat(a asset.Item, f currency.PairStore) error {
	if a.String() == "" {
		return fmt.Errorf("%s cannot add to pairs manager, no asset provided",
			e.Name)
	}

	if f.AssetEnabled == nil {
		f.AssetEnabled = convert.BoolPtr(true)
	}

	if f.RequestFormat == nil {
		return fmt.Errorf("%s cannot add to pairs manager, request pair format not provided",
			e.Name)
	}

	if f.ConfigFormat == nil {
		return fmt.Errorf("%s cannot add to pairs manager, config pair format not provided",
			e.Name)
	}

	if e.CurrencyPairs.Pairs == nil {
		e.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	}

	e.CurrencyPairs.Pairs[a] = &f
	return nil
}

// SetGlobalPairsManager sets defined asset and pairs management system with
// with global formatting
func (e *Base) SetGlobalPairsManager(request, config *currency.PairFormat, assets ...asset.Item) error {
	if request == nil {
		return fmt.Errorf("%s cannot set pairs manager, request pair format not provided",
			e.Name)
	}

	if config == nil {
		return fmt.Errorf("%s cannot set pairs manager, config pair format not provided",
			e.Name)
	}

	if len(assets) == 0 {
		return fmt.Errorf("%s cannot set pairs manager, no assets provided",
			e.Name)
	}

	e.CurrencyPairs.UseGlobalFormat = true
	e.CurrencyPairs.RequestFormat = request
	e.CurrencyPairs.ConfigFormat = config

	if e.CurrencyPairs.Pairs != nil {
		return fmt.Errorf("%s cannot set pairs manager, pairs already set",
			e.Name)
	}

	e.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)

	for i := range assets {
		if assets[i].String() == "" {
			e.CurrencyPairs.Pairs = nil
			return fmt.Errorf("%s cannot set pairs manager, asset is empty string",
				e.Name)
		}
		e.CurrencyPairs.Pairs[assets[i]] = new(currency.PairStore)
		e.CurrencyPairs.Pairs[assets[i]].ConfigFormat = config
		e.CurrencyPairs.Pairs[assets[i]].RequestFormat = request
	}

	return nil
}

// GetWebsocket returns a pointer to the exchange websocket
func (e *Base) GetWebsocket() (*stream.Websocket, error) {
	if e.Websocket == nil {
		return nil, common.ErrFunctionNotSupported
	}
	return e.Websocket, nil
}

// SupportsWebsocket returns whether or not the exchange supports
// websocket
func (e *Base) SupportsWebsocket() bool {
	return e.Features.Supports.Websocket
}

// IsWebsocketEnabled returns whether or not the exchange has its
// websocket client enabled
func (e *Base) IsWebsocketEnabled() bool {
	if e.Websocket == nil {
		return false
	}
	return e.Websocket.IsEnabled()
}

// FlushWebsocketChannels refreshes websocket channel subscriptions based on
// websocket features. Used in the event of a pair/asset or subscription change.
func (e *Base) FlushWebsocketChannels() error {
	if e.Websocket == nil {
		return nil
	}
	return e.Websocket.FlushChannels()
}

// SubscribeToWebsocketChannels appends to ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle subscribing
func (e *Base) SubscribeToWebsocketChannels(channels []stream.ChannelSubscription) error {
	if e.Websocket == nil {
		return common.ErrFunctionNotSupported
	}
	return e.Websocket.SubscribeToChannels(channels)
}

// UnsubscribeToWebsocketChannels removes from ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle unsubscribing
func (e *Base) UnsubscribeToWebsocketChannels(channels []stream.ChannelSubscription) error {
	if e.Websocket == nil {
		return common.ErrFunctionNotSupported
	}
	return e.Websocket.UnsubscribeChannels(channels)
}

// GetSubscriptions returns a copied list of subscriptions
func (e *Base) GetSubscriptions() ([]stream.ChannelSubscription, error) {
	if e.Websocket == nil {
		return nil, common.ErrFunctionNotSupported
	}
	return e.Websocket.GetSubscriptions(), nil
}

// AuthenticateWebsocket sends an authentication message to the websocket
func (e *Base) AuthenticateWebsocket() error {
	return common.ErrFunctionNotSupported
}

// KlineIntervalEnabled returns if requested interval is enabled on exchange
func (e *Base) klineIntervalEnabled(in kline.Interval) bool {
	return e.Features.Enabled.Kline.Intervals[in.Word()]
}

// FormatExchangeKlineInterval returns Interval to string
// Exchanges can override this if they require custom formatting
func (e *Base) FormatExchangeKlineInterval(in kline.Interval) string {
	return strconv.FormatFloat(in.Duration().Seconds(), 'f', 0, 64)
}

// ValidateKline confirms that the requested pair, asset & interval are supported and/or enabled by the requested exchange
func (e *Base) ValidateKline(pair currency.Pair, a asset.Item, interval kline.Interval) error {
	var errorList []string
	var err kline.ErrorKline
	if e.CurrencyPairs.IsAssetEnabled(a) != nil {
		err.Asset = a
		errorList = append(errorList, "asset not enabled")
	} else if !e.CurrencyPairs.Pairs[a].Enabled.Contains(pair, true) {
		err.Pair = pair
		errorList = append(errorList, "pair not enabled")
	}

	if !e.klineIntervalEnabled(interval) {
		err.Interval = interval
		errorList = append(errorList, "interval not supported")
	}

	if len(errorList) > 0 {
		err.Err = errors.New(strings.Join(errorList, ","))
		return &err
	}

	return nil
}

// AddTradesToBuffer is a helper function that will only
// add trades to the buffer if it is allowed
func (e *Base) AddTradesToBuffer(trades ...trade.Data) error {
	if !e.IsSaveTradeDataEnabled() {
		return nil
	}

	return trade.AddTradesToBuffer(e.Name, trades...)
}

// IsSaveTradeDataEnabled checks the state of
// SaveTradeData in a concurrent-friendly manner
func (e *Base) IsSaveTradeDataEnabled() bool {
	e.settingsMutex.RLock()
	isEnabled := e.Features.Enabled.SaveTradeData
	e.settingsMutex.RUnlock()
	return isEnabled
}

// SetSaveTradeDataStatus locks and sets the status of
// the config and the exchange's setting for SaveTradeData
func (e *Base) SetSaveTradeDataStatus(enabled bool) {
	e.settingsMutex.Lock()
	defer e.settingsMutex.Unlock()
	e.Features.Enabled.SaveTradeData = enabled
	e.Config.Features.Enabled.SaveTradeData = enabled
	if e.Verbose {
		log.Debugf(log.Trade, "Set %v 'SaveTradeData' to %v", e.Name, enabled)
	}
}

// NewEndpoints declares default and running URLs maps
func (e *Base) NewEndpoints() *Endpoints {
	return &Endpoints{
		Exchange: e.Name,
		defaults: make(map[string]string),
	}
}

// SetDefaultEndpoints declares and sets the default URLs map
func (e *Endpoints) SetDefaultEndpoints(m map[URL]string) error {
	for k, v := range m {
		err := e.SetRunning(k.String(), v)
		if err != nil {
			return err
		}
	}
	return nil
}

// SetRunning populates running URLs map
func (e *Endpoints) SetRunning(key, val string) error {
	e.Lock()
	defer e.Unlock()
	err := validateKey(key)
	if err != nil {
		return err
	}
	_, err = url.ParseRequestURI(val)
	if err != nil {
		log.Warnf(log.ExchangeSys, "Could not set custom URL for %s to %s for exchange %s. invalid URI for request.", key, val, e.Exchange)
		return nil
	}
	e.defaults[key] = val
	return nil
}

func validateKey(keyVal string) error {
	for x := range keyURLs {
		if keyURLs[x].String() == keyVal {
			return nil
		}
	}
	return errors.New("keyVal invalid")
}

// GetURL gets default url from URLs map
func (e *Endpoints) GetURL(key URL) (string, error) {
	e.RLock()
	defer e.RUnlock()
	val, ok := e.defaults[key.String()]
	if !ok {
		return "", fmt.Errorf("no endpoint path found for the given key: %v", key)
	}
	return val, nil
}

// GetURLMap gets all urls for either running or default map based on the bool value supplied
func (e *Endpoints) GetURLMap() map[string]string {
	e.RLock()
	var urlMap = make(map[string]string)
	for k, v := range e.defaults {
		urlMap[k] = v
	}
	e.RUnlock()
	return urlMap
}

// FormatSymbol formats the given pair to a string suitable for exchange API requests
func (e *Base) FormatSymbol(pair currency.Pair, assetType asset.Item) (string, error) {
	pairFmt, err := e.GetPairFormat(assetType, true)
	if err != nil {
		return pair.String(), err
	}
	return pairFmt.Format(pair), nil
}

func (u URL) String() string {
	switch u {
	case RestSpot:
		return "RestSpotURL"
	case RestSpotSupplementary:
		return "RestSpotSupplementaryURL"
	case RestUSDTMargined:
		return "RestUSDTMarginedFuturesURL"
	case RestCoinMargined:
		return "RestCoinMarginedFuturesURL"
	case RestFutures:
		return "RestFuturesURL"
	case RestSandbox:
		return "RestSandboxURL"
	case RestSwap:
		return "RestSwapURL"
	case WebsocketSpot:
		return "WebsocketSpotURL"
	case WebsocketSpotSupplementary:
		return "WebsocketSpotSupplementaryURL"
	case ChainAnalysis:
		return "ChainAnalysisURL"
	case EdgeCase1:
		return "EdgeCase1URL"
	case EdgeCase2:
		return "EdgeCase2URL"
	case EdgeCase3:
		return "EdgeCase3URL"
	default:
		return ""
	}
}

// UpdateOrderExecutionLimits updates order execution limits this is overridable
func (e *Base) UpdateOrderExecutionLimits(a asset.Item) error {
	return common.ErrNotYetImplemented
}
