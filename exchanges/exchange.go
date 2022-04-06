package exchange

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/currencystate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/banking"
)

const (
	warningBase64DecryptSecretKeyFailed = "exchange %s unable to base64 decode secret key.. Disabling Authenticated API support" // nolint // False positive (G101: Potential hardcoded credentials)
	// DefaultHTTPTimeout is the default HTTP/HTTPS Timeout for exchange requests
	DefaultHTTPTimeout = time.Second * 15
	// DefaultWebsocketResponseCheckTimeout is the default delay in checking for an expected websocket response
	DefaultWebsocketResponseCheckTimeout = time.Millisecond * 50
	// DefaultWebsocketResponseMaxLimit is the default max wait for an expected websocket response before a timeout
	DefaultWebsocketResponseMaxLimit = time.Second * 7
	// DefaultWebsocketOrderbookBufferLimit is the maximum number of orderbook updates that get stored before being applied
	DefaultWebsocketOrderbookBufferLimit = 5
	// ResetConfigPairsWarningMessage is displayed when a currency pair format in the config needs to be updated
	ResetConfigPairsWarningMessage = "%s Enabled and available pairs for %s reset due to config upgrade, please enable the ones you would like to use again. Defaulting to %v"
)

var errEndpointStringNotFound = errors.New("endpoint string not found")

// SetClientProxyAddress sets a proxy address for REST and websocket requests
func (b *Base) SetClientProxyAddress(addr string) error {
	if addr == "" {
		return nil
	}
	proxy, err := url.Parse(addr)
	if err != nil {
		return fmt.Errorf("setting proxy address error %s",
			err)
	}

	err = b.Requester.SetProxy(proxy)
	if err != nil {
		return err
	}

	if b.Websocket != nil {
		err = b.Websocket.SetProxyAddress(addr)
		if err != nil {
			return err
		}
	}
	return nil
}

// SetFeatureDefaults sets the exchanges default feature support set
func (b *Base) SetFeatureDefaults() {
	if b.Config.Features == nil {
		s := &config.FeaturesConfig{
			Supports: config.FeaturesSupportedConfig{
				Websocket: b.Features.Supports.Websocket,
				REST:      b.Features.Supports.REST,
				RESTCapabilities: protocol.Features{
					AutoPairUpdates: b.Features.Supports.RESTCapabilities.AutoPairUpdates,
				},
			},
		}

		if b.Config.SupportsAutoPairUpdates != nil {
			s.Supports.RESTCapabilities.AutoPairUpdates = *b.Config.SupportsAutoPairUpdates
			s.Enabled.AutoPairUpdates = *b.Config.SupportsAutoPairUpdates
		} else {
			s.Supports.RESTCapabilities.AutoPairUpdates = b.Features.Supports.RESTCapabilities.AutoPairUpdates
			s.Enabled.AutoPairUpdates = b.Features.Supports.RESTCapabilities.AutoPairUpdates
			if !s.Supports.RESTCapabilities.AutoPairUpdates {
				b.Config.CurrencyPairs.LastUpdated = time.Now().Unix()
				b.CurrencyPairs.LastUpdated = b.Config.CurrencyPairs.LastUpdated
			}
		}
		b.Config.Features = s
		b.Config.SupportsAutoPairUpdates = nil
	} else {
		if b.Features.Supports.RESTCapabilities.AutoPairUpdates != b.Config.Features.Supports.RESTCapabilities.AutoPairUpdates {
			b.Config.Features.Supports.RESTCapabilities.AutoPairUpdates = b.Features.Supports.RESTCapabilities.AutoPairUpdates

			if !b.Config.Features.Supports.RESTCapabilities.AutoPairUpdates {
				b.Config.CurrencyPairs.LastUpdated = time.Now().Unix()
			}
		}

		if b.Features.Supports.REST != b.Config.Features.Supports.REST {
			b.Config.Features.Supports.REST = b.Features.Supports.REST
		}

		if b.Features.Supports.RESTCapabilities.TickerBatching != b.Config.Features.Supports.RESTCapabilities.TickerBatching {
			b.Config.Features.Supports.RESTCapabilities.TickerBatching = b.Features.Supports.RESTCapabilities.TickerBatching
		}

		if b.Features.Supports.Websocket != b.Config.Features.Supports.Websocket {
			b.Config.Features.Supports.Websocket = b.Features.Supports.Websocket
		}

		if b.IsSaveTradeDataEnabled() != b.Config.Features.Enabled.SaveTradeData {
			b.SetSaveTradeDataStatus(b.Config.Features.Enabled.SaveTradeData)
		}

		if b.IsTradeFeedEnabled() != b.Config.Features.Enabled.TradeFeed {
			b.SetTradeFeedStatus(b.Config.Features.Enabled.TradeFeed)
		}

		if b.IsFillsFeedEnabled() != b.Config.Features.Enabled.FillsFeed {
			b.SetFillsFeedStatus(b.Config.Features.Enabled.FillsFeed)
		}

		b.Features.Enabled.AutoPairUpdates = b.Config.Features.Enabled.AutoPairUpdates
	}
}

// SupportsRESTTickerBatchUpdates returns whether or not the
// exhange supports REST batch ticker fetching
func (b *Base) SupportsRESTTickerBatchUpdates() bool {
	return b.Features.Supports.RESTCapabilities.TickerBatching
}

// SupportsAutoPairUpdates returns whether or not the exchange supports
// auto currency pair updating
func (b *Base) SupportsAutoPairUpdates() bool {
	if b.Features.Supports.RESTCapabilities.AutoPairUpdates ||
		b.Features.Supports.WebsocketCapabilities.AutoPairUpdates {
		return true
	}
	return false
}

// GetLastPairsUpdateTime returns the unix timestamp of when the exchanges
// currency pairs were last updated
func (b *Base) GetLastPairsUpdateTime() int64 {
	return b.CurrencyPairs.LastUpdated
}

// GetAssetTypes returns the either the enabled or available asset types for an
// individual exchange
func (b *Base) GetAssetTypes(enabled bool) asset.Items {
	return b.CurrencyPairs.GetAssetTypes(enabled)
}

// GetPairAssetType returns the associated asset type for the currency pair
// This method is only useful for exchanges that have pair names with multiple delimiters (BTC-USD-0626)
// Helpful if the exchange has only a single asset type but in that case the asset type can be hard coded
func (b *Base) GetPairAssetType(c currency.Pair) (asset.Item, error) {
	assetTypes := b.GetAssetTypes(false)
	for i := range assetTypes {
		avail, err := b.GetAvailablePairs(assetTypes[i])
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
func (b *Base) GetClientBankAccounts(exchangeName, withdrawalCurrency string) (*banking.Account, error) {
	cfg := config.GetConfig()
	return cfg.GetClientBankAccounts(exchangeName, withdrawalCurrency)
}

// GetExchangeBankAccounts returns banking details associated with an
// exchange for funding purposes
func (b *Base) GetExchangeBankAccounts(id, depositCurrency string) (*banking.Account, error) {
	cfg := config.GetConfig()
	return cfg.GetExchangeBankAccounts(b.Name, id, depositCurrency)
}

// SetCurrencyPairFormat checks the exchange request and config currency pair
// formats and syncs it with the exchanges SetDefault settings
func (b *Base) SetCurrencyPairFormat() {
	if b.Config.CurrencyPairs == nil {
		b.Config.CurrencyPairs = new(currency.PairsManager)
	}

	b.Config.CurrencyPairs.UseGlobalFormat = b.CurrencyPairs.UseGlobalFormat
	if b.Config.CurrencyPairs.UseGlobalFormat {
		b.Config.CurrencyPairs.RequestFormat = b.CurrencyPairs.RequestFormat
		b.Config.CurrencyPairs.ConfigFormat = b.CurrencyPairs.ConfigFormat
		return
	}

	if b.Config.CurrencyPairs.ConfigFormat != nil {
		b.Config.CurrencyPairs.ConfigFormat = nil
	}
	if b.Config.CurrencyPairs.RequestFormat != nil {
		b.Config.CurrencyPairs.RequestFormat = nil
	}

	assetTypes := b.GetAssetTypes(false)
	for x := range assetTypes {
		if _, err := b.Config.CurrencyPairs.Get(assetTypes[x]); err != nil {
			ps, err := b.CurrencyPairs.Get(assetTypes[x])
			if err != nil {
				continue
			}
			b.Config.CurrencyPairs.Store(assetTypes[x], *ps)
		}
	}
}

// SetConfigPairs sets the exchanges currency pairs to the pairs set in the config
func (b *Base) SetConfigPairs() error {
	assetTypes := b.Config.CurrencyPairs.GetAssetTypes(false)
	exchangeAssets := b.CurrencyPairs.GetAssetTypes(false)
	for x := range assetTypes {
		if !exchangeAssets.Contains(assetTypes[x]) {
			log.Warnf(log.ExchangeSys,
				"%s exchange asset type %s unsupported, please manually remove from configuration",
				b.Name,
				assetTypes[x])
			continue // If there are unsupported assets contained in config, skip.
		}
		cfgPS, err := b.Config.CurrencyPairs.Get(assetTypes[x])
		if err != nil {
			return err
		}

		var enabledAsset bool
		if b.Config.CurrencyPairs.IsAssetEnabled(assetTypes[x]) == nil {
			enabledAsset = true
		}

		err = b.CurrencyPairs.SetAssetEnabled(assetTypes[x], enabledAsset)
		// Suppress error when assets are enabled by default and they are being
		// enabled by config. A check for the inverse
		// e.g. currency.ErrAssetAlreadyDisabled is not needed.
		if err != nil && err != currency.ErrAssetAlreadyEnabled {
			return err
		}

		if b.Config.CurrencyPairs.UseGlobalFormat {
			b.CurrencyPairs.StorePairs(assetTypes[x], cfgPS.Available, false)
			b.CurrencyPairs.StorePairs(assetTypes[x], cfgPS.Enabled, true)
			continue
		}
		exchPS, err := b.CurrencyPairs.Get(assetTypes[x])
		if err != nil {
			return err
		}
		cfgPS.ConfigFormat = exchPS.ConfigFormat
		cfgPS.RequestFormat = exchPS.RequestFormat
		b.CurrencyPairs.StorePairs(assetTypes[x], cfgPS.Available, false)
		b.CurrencyPairs.StorePairs(assetTypes[x], cfgPS.Enabled, true)
	}
	return nil
}

// GetName is a method that returns the name of the exchange base
func (b *Base) GetName() string {
	return b.Name
}

// GetEnabledFeatures returns the exchanges enabled features
func (b *Base) GetEnabledFeatures() FeaturesEnabled {
	return b.Features.Enabled
}

// GetSupportedFeatures returns the exchanges supported features
func (b *Base) GetSupportedFeatures() FeaturesSupported {
	return b.Features.Supports
}

// GetPairFormat returns the pair format based on the exchange and
// asset type
func (b *Base) GetPairFormat(assetType asset.Item, requestFormat bool) (currency.PairFormat, error) {
	if b.CurrencyPairs.UseGlobalFormat {
		if requestFormat {
			if b.CurrencyPairs.RequestFormat == nil {
				return currency.PairFormat{},
					errors.New("global request format is nil")
			}
			return *b.CurrencyPairs.RequestFormat, nil
		}

		if b.CurrencyPairs.ConfigFormat == nil {
			return currency.PairFormat{},
				errors.New("global config format is nil")
		}
		return *b.CurrencyPairs.ConfigFormat, nil
	}

	ps, err := b.CurrencyPairs.Get(assetType)
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
func (b *Base) GetEnabledPairs(a asset.Item) (currency.Pairs, error) {
	err := b.CurrencyPairs.IsAssetEnabled(a)
	if err != nil {
		return nil, nil // nolint:nilerr // non-fatal error
	}
	format, err := b.GetPairFormat(a, false)
	if err != nil {
		return nil, err
	}
	enabledpairs, err := b.CurrencyPairs.GetPairs(a, true)
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
func (b *Base) GetRequestFormattedPairAndAssetType(p string) (currency.Pair, asset.Item, error) {
	assetTypes := b.GetAssetTypes(true)
	for i := range assetTypes {
		format, err := b.GetPairFormat(assetTypes[i], true)
		if err != nil {
			return currency.EMPTYPAIR, assetTypes[i], err
		}

		pairs, err := b.CurrencyPairs.GetPairs(assetTypes[i], true)
		if err != nil {
			return currency.EMPTYPAIR, assetTypes[i], err
		}

		for j := range pairs {
			formattedPair := pairs[j].Format(format.Delimiter, format.Uppercase)
			if strings.EqualFold(formattedPair.String(), p) {
				return formattedPair, assetTypes[i], nil
			}
		}
	}
	return currency.EMPTYPAIR, "", fmt.Errorf("%s %w", p, currency.ErrPairNotFound)
}

// GetAvailablePairs is a method that returns the available currency pairs
// of the exchange by asset type
func (b *Base) GetAvailablePairs(assetType asset.Item) (currency.Pairs, error) {
	format, err := b.GetPairFormat(assetType, false)
	if err != nil {
		return nil, err
	}
	pairs, err := b.CurrencyPairs.GetPairs(assetType, false)
	if err != nil {
		return nil, err
	}
	return pairs.Format(format.Delimiter, format.Index, format.Uppercase), nil
}

// SupportsPair returns true or not whether a currency pair exists in the
// exchange available currencies or not
func (b *Base) SupportsPair(p currency.Pair, enabledPairs bool, assetType asset.Item) error {
	if enabledPairs {
		pairs, err := b.GetEnabledPairs(assetType)
		if err != nil {
			return err
		}
		if pairs.Contains(p, false) {
			return nil
		}
		return errors.New("pair not supported")
	}

	avail, err := b.GetAvailablePairs(assetType)
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
func (b *Base) FormatExchangeCurrencies(pairs []currency.Pair, assetType asset.Item) (string, error) {
	var currencyItems strings.Builder
	pairFmt, err := b.GetPairFormat(assetType, true)
	if err != nil {
		return "", err
	}

	for x := range pairs {
		format, err := b.FormatExchangeCurrency(pairs[x], assetType)
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
func (b *Base) FormatExchangeCurrency(p currency.Pair, assetType asset.Item) (currency.Pair, error) {
	pairFmt, err := b.GetPairFormat(assetType, true)
	if err != nil {
		return currency.EMPTYPAIR, err
	}
	return p.Format(pairFmt.Delimiter, pairFmt.Uppercase), nil
}

// SetEnabled is a method that sets if the exchange is enabled
func (b *Base) SetEnabled(enabled bool) {
	b.Enabled = enabled
}

// IsEnabled is a method that returns if the current exchange is enabled
func (b *Base) IsEnabled() bool {
	return b.Enabled
}

// SetupDefaults sets the exchange settings based on the supplied config
func (b *Base) SetupDefaults(exch *config.Exchange) error {
	err := exch.Validate()
	if err != nil {
		return err
	}

	b.Enabled = true
	b.LoadedByConfig = true
	b.Config = exch
	b.Verbose = exch.Verbose

	b.API.AuthenticatedSupport = exch.API.AuthenticatedSupport
	b.API.AuthenticatedWebsocketSupport = exch.API.AuthenticatedWebsocketSupport
	if b.API.credentials == nil {
		b.API.credentials = &Credentials{}
	}
	b.API.credentials.SubAccount = exch.API.Credentials.Subaccount
	if b.API.AuthenticatedSupport || b.API.AuthenticatedWebsocketSupport {
		b.SetCredentials(exch.API.Credentials.Key,
			exch.API.Credentials.Secret,
			exch.API.Credentials.ClientID,
			exch.API.Credentials.Subaccount,
			exch.API.Credentials.PEMKey,
			exch.API.Credentials.OTPSecret,
		)
	}

	if exch.HTTPTimeout <= time.Duration(0) {
		exch.HTTPTimeout = DefaultHTTPTimeout
	}

	err = b.SetHTTPClientTimeout(exch.HTTPTimeout)
	if err != nil {
		return err
	}

	if exch.CurrencyPairs == nil {
		exch.CurrencyPairs = new(currency.PairsManager)
	}

	b.HTTPDebugging = exch.HTTPDebugging
	b.BypassConfigFormatUpgrades = exch.CurrencyPairs.BypassConfigFormatUpgrades
	err = b.SetHTTPClientUserAgent(exch.HTTPUserAgent)
	if err != nil {
		return err
	}
	b.SetCurrencyPairFormat()

	err = b.SetConfigPairs()
	if err != nil {
		return err
	}

	b.SetFeatureDefaults()

	if b.API.Endpoints == nil {
		b.API.Endpoints = b.NewEndpoints()
	}

	err = b.SetAPIURL()
	if err != nil {
		return err
	}

	b.SetAPICredentialDefaults()

	err = b.SetClientProxyAddress(exch.ProxyAddress)
	if err != nil {
		return err
	}
	b.BaseCurrencies = exch.BaseCurrencies

	if exch.Orderbook.VerificationBypass {
		log.Warnf(log.ExchangeSys,
			"%s orderbook verification has been bypassed via config.",
			b.Name)
	}
	b.CanVerifyOrderbook = !exch.Orderbook.VerificationBypass
	b.States = currencystate.NewCurrencyStates()
	return err
}

// SetPairs sets the exchange currency pairs for either enabledPairs or
// availablePairs
func (b *Base) SetPairs(pairs currency.Pairs, assetType asset.Item, enabled bool) error {
	if len(pairs) == 0 {
		return fmt.Errorf("%s SetPairs error - pairs is empty", b.Name)
	}

	pairFmt, err := b.GetPairFormat(assetType, false)
	if err != nil {
		return err
	}

	var newPairs currency.Pairs
	for x := range pairs {
		newPairs = append(newPairs, pairs[x].Format(pairFmt.Delimiter,
			pairFmt.Uppercase))
	}

	b.CurrencyPairs.StorePairs(assetType, newPairs, enabled)
	b.Config.CurrencyPairs.StorePairs(assetType, newPairs, enabled)
	return nil
}

// UpdatePairs updates the exchange currency pairs for either enabledPairs or
// availablePairs
func (b *Base) UpdatePairs(exchangeProducts currency.Pairs, assetType asset.Item, enabled, force bool) error {
	exchangeProducts = exchangeProducts.Upper()
	var products currency.Pairs
	for x := range exchangeProducts {
		if exchangeProducts[x].String() == "" {
			continue
		}
		products = append(products, exchangeProducts[x])
	}

	var updateType string
	targetPairs, err := b.CurrencyPairs.GetPairs(assetType, enabled)
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
				b.Name,
				updateType,
				strings.ToUpper(assetType.String()))
		} else {
			if len(newPairs) > 0 {
				log.Debugf(log.ExchangeSys,
					"%s Updating %s pairs [%v] - Added: %s.\n",
					b.Name,
					updateType,
					strings.ToUpper(assetType.String()),
					newPairs)
			}
			if len(removedPairs) > 0 {
				log.Debugf(log.ExchangeSys,
					"%s Updating %s pairs [%v] - Removed: %s.\n",
					b.Name,
					updateType,
					strings.ToUpper(assetType.String()),
					removedPairs)
			}
		}

		b.Config.CurrencyPairs.StorePairs(assetType, products, enabled)
		b.CurrencyPairs.StorePairs(assetType, products, enabled)

		if !enabled {
			// If available pairs are changed we will remove currency pair items
			// that are still included in the enabled pairs list.
			enabledPairs, err := b.CurrencyPairs.GetPairs(assetType, true)
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
					b.Name,
					strings.ToUpper(assetType.String()),
					remove)
				b.Config.CurrencyPairs.StorePairs(assetType, enabledPairs, true)
				b.CurrencyPairs.StorePairs(assetType, enabledPairs, true)
			}
		}
	}
	return nil
}

// SetAPIURL sets configuration API URL for an exchange
func (b *Base) SetAPIURL() error {
	checkInsecureEndpoint := func(endpoint string) {
		if strings.Contains(endpoint, "https") || strings.Contains(endpoint, "wss") {
			return
		}
		log.Warnf(log.ExchangeSys,
			"%s is using HTTP instead of HTTPS or WS instead of WSS [%s] for API functionality, an"+
				" attacker could eavesdrop on this connection. Use at your"+
				" own risk.",
			b.Name, endpoint)
	}
	var err error
	if b.Config.API.OldEndPoints != nil {
		if b.Config.API.OldEndPoints.URL != "" && b.Config.API.OldEndPoints.URL != config.APIURLNonDefaultMessage {
			err = b.API.Endpoints.SetRunning(RestSpot.String(), b.Config.API.OldEndPoints.URL)
			if err != nil {
				return err
			}
			checkInsecureEndpoint(b.Config.API.OldEndPoints.URL)
		}
		if b.Config.API.OldEndPoints.URLSecondary != "" && b.Config.API.OldEndPoints.URLSecondary != config.APIURLNonDefaultMessage {
			err = b.API.Endpoints.SetRunning(RestSpotSupplementary.String(), b.Config.API.OldEndPoints.URLSecondary)
			if err != nil {
				return err
			}
			checkInsecureEndpoint(b.Config.API.OldEndPoints.URLSecondary)
		}
		if b.Config.API.OldEndPoints.WebsocketURL != "" && b.Config.API.OldEndPoints.WebsocketURL != config.WebsocketURLNonDefaultMessage {
			err = b.API.Endpoints.SetRunning(WebsocketSpot.String(), b.Config.API.OldEndPoints.WebsocketURL)
			if err != nil {
				return err
			}
			checkInsecureEndpoint(b.Config.API.OldEndPoints.WebsocketURL)
		}
		b.Config.API.OldEndPoints = nil
	} else if b.Config.API.Endpoints != nil {
		for key, val := range b.Config.API.Endpoints {
			if val == "" ||
				val == config.APIURLNonDefaultMessage ||
				val == config.WebsocketURLNonDefaultMessage {
				continue
			}

			var u URL
			u, err = getURLTypeFromString(key)
			if err != nil {
				return err
			}

			var defaultURL string
			defaultURL, err = b.API.Endpoints.GetURL(u)
			if err != nil {
				log.Warnf(
					log.ExchangeSys,
					"%s: Config cannot match with default endpoint URL: [%s] with key: [%s], please remove or update core support endpoints.",
					b.Name,
					val,
					u)
				continue
			}

			if defaultURL == val {
				continue
			}

			log.Warnf(
				log.ExchangeSys,
				"%s: Config is overwriting default endpoint URL values from: [%s] to: [%s] for: [%s]",
				b.Name,
				defaultURL,
				val,
				u)

			checkInsecureEndpoint(val)

			err = b.API.Endpoints.SetRunning(key, val)
			if err != nil {
				return err
			}
		}
	}
	runningMap := b.API.Endpoints.GetURLMap()
	b.Config.API.Endpoints = runningMap
	return nil
}

// SupportsREST returns whether or not the exchange supports
// REST
func (b *Base) SupportsREST() bool {
	return b.Features.Supports.REST
}

// GetWithdrawPermissions passes through the exchange's withdraw permissions
func (b *Base) GetWithdrawPermissions() uint32 {
	return b.Features.Supports.WithdrawPermissions
}

// SupportsWithdrawPermissions compares the supplied permissions with the exchange's to verify they're supported
func (b *Base) SupportsWithdrawPermissions(permissions uint32) bool {
	exchangePermissions := b.GetWithdrawPermissions()
	return permissions&exchangePermissions == permissions
}

// FormatWithdrawPermissions will return each of the exchange's compatible withdrawal methods in readable form
func (b *Base) FormatWithdrawPermissions() string {
	var services []string
	for i := 0; i < 32; i++ {
		var check uint32 = 1 << uint32(i)
		if b.GetWithdrawPermissions()&check != 0 {
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
func (b *Base) SupportsAsset(a asset.Item) bool {
	_, ok := b.CurrencyPairs.Pairs[a]
	return ok
}

// PrintEnabledPairs prints the exchanges enabled asset pairs
func (b *Base) PrintEnabledPairs() {
	for k, v := range b.CurrencyPairs.Pairs {
		log.Infof(log.ExchangeSys, "%s Asset type %v:\n\t Enabled pairs: %v",
			b.Name, strings.ToUpper(k.String()), v.Enabled)
	}
}

// GetBase returns the exchange base
func (b *Base) GetBase() *Base { return b }

// CheckTransientError catches transient errors and returns nil if found, used
// for validation of API credentials
func (b *Base) CheckTransientError(err error) error {
	if _, ok := err.(net.Error); ok {
		log.Warnf(log.ExchangeSys,
			"%s net error captured, will not disable authentication %s",
			b.Name,
			err)
		return nil
	}
	return err
}

// DisableRateLimiter disables the rate limiting system for the exchange
func (b *Base) DisableRateLimiter() error {
	return b.Requester.DisableRateLimiter()
}

// EnableRateLimiter enables the rate limiting system for the exchange
func (b *Base) EnableRateLimiter() error {
	return b.Requester.EnableRateLimiter()
}

// StoreAssetPairFormat initialises and stores a defined asset format
func (b *Base) StoreAssetPairFormat(a asset.Item, f currency.PairStore) error {
	if a.String() == "" {
		return fmt.Errorf("%s cannot add to pairs manager, no asset provided",
			b.Name)
	}

	if f.AssetEnabled == nil {
		f.AssetEnabled = convert.BoolPtr(true)
	}

	if f.RequestFormat == nil {
		return fmt.Errorf("%s cannot add to pairs manager, request pair format not provided",
			b.Name)
	}

	if f.ConfigFormat == nil {
		return fmt.Errorf("%s cannot add to pairs manager, config pair format not provided",
			b.Name)
	}

	if b.CurrencyPairs.Pairs == nil {
		b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	}

	b.CurrencyPairs.Pairs[a] = &f
	return nil
}

// SetGlobalPairsManager sets defined asset and pairs management system with
// with global formatting
func (b *Base) SetGlobalPairsManager(request, config *currency.PairFormat, assets ...asset.Item) error {
	if request == nil {
		return fmt.Errorf("%s cannot set pairs manager, request pair format not provided",
			b.Name)
	}

	if config == nil {
		return fmt.Errorf("%s cannot set pairs manager, config pair format not provided",
			b.Name)
	}

	if len(assets) == 0 {
		return fmt.Errorf("%s cannot set pairs manager, no assets provided",
			b.Name)
	}

	b.CurrencyPairs.UseGlobalFormat = true
	b.CurrencyPairs.RequestFormat = request
	b.CurrencyPairs.ConfigFormat = config

	if b.CurrencyPairs.Pairs != nil {
		return fmt.Errorf("%s cannot set pairs manager, pairs already set",
			b.Name)
	}

	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)

	for i := range assets {
		if assets[i].String() == "" {
			b.CurrencyPairs.Pairs = nil
			return fmt.Errorf("%s cannot set pairs manager, asset is empty string",
				b.Name)
		}
		b.CurrencyPairs.Pairs[assets[i]] = new(currency.PairStore)
		b.CurrencyPairs.Pairs[assets[i]].ConfigFormat = config
		b.CurrencyPairs.Pairs[assets[i]].RequestFormat = request
	}

	return nil
}

// GetWebsocket returns a pointer to the exchange websocket
func (b *Base) GetWebsocket() (*stream.Websocket, error) {
	if b.Websocket == nil {
		return nil, common.ErrFunctionNotSupported
	}
	return b.Websocket, nil
}

// SupportsWebsocket returns whether or not the exchange supports
// websocket
func (b *Base) SupportsWebsocket() bool {
	return b.Features.Supports.Websocket
}

// IsWebsocketEnabled returns whether or not the exchange has its
// websocket client enabled
func (b *Base) IsWebsocketEnabled() bool {
	if b.Websocket == nil {
		return false
	}
	return b.Websocket.IsEnabled()
}

// FlushWebsocketChannels refreshes websocket channel subscriptions based on
// websocket features. Used in the event of a pair/asset or subscription change.
func (b *Base) FlushWebsocketChannels() error {
	if b.Websocket == nil {
		return nil
	}
	return b.Websocket.FlushChannels()
}

// SubscribeToWebsocketChannels appends to ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle subscribing
func (b *Base) SubscribeToWebsocketChannels(channels []stream.ChannelSubscription) error {
	if b.Websocket == nil {
		return common.ErrFunctionNotSupported
	}
	return b.Websocket.SubscribeToChannels(channels)
}

// UnsubscribeToWebsocketChannels removes from ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle unsubscribing
func (b *Base) UnsubscribeToWebsocketChannels(channels []stream.ChannelSubscription) error {
	if b.Websocket == nil {
		return common.ErrFunctionNotSupported
	}
	return b.Websocket.UnsubscribeChannels(channels)
}

// GetSubscriptions returns a copied list of subscriptions
func (b *Base) GetSubscriptions() ([]stream.ChannelSubscription, error) {
	if b.Websocket == nil {
		return nil, common.ErrFunctionNotSupported
	}
	return b.Websocket.GetSubscriptions(), nil
}

// AuthenticateWebsocket sends an authentication message to the websocket
func (b *Base) AuthenticateWebsocket(_ context.Context) error {
	return common.ErrFunctionNotSupported
}

// KlineIntervalEnabled returns if requested interval is enabled on exchange
func (b *Base) klineIntervalEnabled(in kline.Interval) bool {
	return b.Features.Enabled.Kline.Intervals[in.Word()]
}

// FormatExchangeKlineInterval returns Interval to string
// Exchanges can override this if they require custom formatting
func (b *Base) FormatExchangeKlineInterval(in kline.Interval) string {
	return strconv.FormatFloat(in.Duration().Seconds(), 'f', 0, 64)
}

// ValidateKline confirms that the requested pair, asset & interval are supported and/or enabled by the requested exchange
func (b *Base) ValidateKline(pair currency.Pair, a asset.Item, interval kline.Interval) error {
	var errorList []string
	var err kline.Error
	if b.CurrencyPairs.IsAssetEnabled(a) != nil {
		err.Asset = a
		errorList = append(errorList, "asset not enabled")
	} else if !b.CurrencyPairs.Pairs[a].Enabled.Contains(pair, true) {
		err.Pair = pair
		errorList = append(errorList, "pair not enabled")
	}

	if !b.klineIntervalEnabled(interval) {
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
func (b *Base) AddTradesToBuffer(trades ...trade.Data) error {
	if !b.IsSaveTradeDataEnabled() {
		return nil
	}
	return trade.AddTradesToBuffer(b.Name, trades...)
}

// IsSaveTradeDataEnabled checks the state of
// SaveTradeData in a concurrent-friendly manner
func (b *Base) IsSaveTradeDataEnabled() bool {
	b.settingsMutex.RLock()
	isEnabled := b.Features.Enabled.SaveTradeData
	b.settingsMutex.RUnlock()
	return isEnabled
}

// SetSaveTradeDataStatus locks and sets the status of
// the config and the exchange's setting for SaveTradeData
func (b *Base) SetSaveTradeDataStatus(enabled bool) {
	b.settingsMutex.Lock()
	defer b.settingsMutex.Unlock()
	b.Features.Enabled.SaveTradeData = enabled
	b.Config.Features.Enabled.SaveTradeData = enabled
	if b.Verbose {
		log.Debugf(log.Trade, "Set %v 'SaveTradeData' to %v", b.Name, enabled)
	}
}

// IsTradeFeedEnabled checks the state of
// TradeFeed in a concurrent-friendly manner
func (b *Base) IsTradeFeedEnabled() bool {
	b.settingsMutex.RLock()
	isEnabled := b.Features.Enabled.TradeFeed
	b.settingsMutex.RUnlock()
	return isEnabled
}

// SetTradeFeedStatus locks and sets the status of
// the config and the exchange's setting for TradeFeed
func (b *Base) SetTradeFeedStatus(enabled bool) {
	b.settingsMutex.Lock()
	defer b.settingsMutex.Unlock()
	b.Features.Enabled.TradeFeed = enabled
	b.Config.Features.Enabled.TradeFeed = enabled
	if b.Verbose {
		log.Debugf(log.Trade, "Set %v 'TradeFeed' to %v", b.Name, enabled)
	}
}

// IsFillsFeedEnabled checks the state of
// FillsFeed in a concurrent-friendly manner
func (b *Base) IsFillsFeedEnabled() bool {
	b.settingsMutex.RLock()
	isEnabled := b.Features.Enabled.FillsFeed
	b.settingsMutex.RUnlock()
	return isEnabled
}

// SetFillsFeedStatus locks and sets the status of
// the config and the exchange's setting for FillsFeed
func (b *Base) SetFillsFeedStatus(enabled bool) {
	b.settingsMutex.Lock()
	defer b.settingsMutex.Unlock()
	b.Features.Enabled.FillsFeed = enabled
	b.Config.Features.Enabled.FillsFeed = enabled
	if b.Verbose {
		log.Debugf(log.Trade, "Set %v 'FillsFeed' to %v", b.Name, enabled)
	}
}

// NewEndpoints declares default and running URLs maps
func (b *Base) NewEndpoints() *Endpoints {
	return &Endpoints{
		Exchange: b.Name,
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
		log.Warnf(log.ExchangeSys,
			"Could not set custom URL for %s to %s for exchange %s. invalid URI for request.",
			key,
			val,
			e.Exchange)
		return nil // nolint:nilerr // non-fatal error as we won't update the running URL
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
func (b *Base) FormatSymbol(pair currency.Pair, assetType asset.Item) (string, error) {
	pairFmt, err := b.GetPairFormat(assetType, true)
	if err != nil {
		return pair.String(), err
	}
	return pairFmt.Format(pair), nil
}

func (u URL) String() string {
	switch u {
	case RestSpot:
		return restSpotURL
	case RestSpotSupplementary:
		return restSpotSupplementaryURL
	case RestUSDTMargined:
		return restUSDTMarginedFuturesURL
	case RestCoinMargined:
		return restCoinMarginedFuturesURL
	case RestFutures:
		return restFuturesURL
	case RestSandbox:
		return restSandboxURL
	case RestSwap:
		return restSwapURL
	case WebsocketSpot:
		return websocketSpotURL
	case WebsocketSpotSupplementary:
		return websocketSpotSupplementaryURL
	case ChainAnalysis:
		return chainAnalysisURL
	case EdgeCase1:
		return edgeCase1URL
	case EdgeCase2:
		return edgeCase2URL
	case EdgeCase3:
		return edgeCase3URL
	default:
		return ""
	}
}

// getURLTypeFromString returns URL type from the endpoint string association
func getURLTypeFromString(ep string) (URL, error) {
	switch ep {
	case restSpotURL:
		return RestSpot, nil
	case restSpotSupplementaryURL:
		return RestSpotSupplementary, nil
	case restUSDTMarginedFuturesURL:
		return RestUSDTMargined, nil
	case restCoinMarginedFuturesURL:
		return RestCoinMargined, nil
	case restFuturesURL:
		return RestFutures, nil
	case restSandboxURL:
		return RestSandbox, nil
	case restSwapURL:
		return RestSwap, nil
	case websocketSpotURL:
		return WebsocketSpot, nil
	case websocketSpotSupplementaryURL:
		return WebsocketSpotSupplementary, nil
	case chainAnalysisURL:
		return ChainAnalysis, nil
	case edgeCase1URL:
		return EdgeCase1, nil
	case edgeCase2URL:
		return EdgeCase2, nil
	case edgeCase3URL:
		return EdgeCase3, nil
	default:
		return Invalid, fmt.Errorf("%w for %s", errEndpointStringNotFound, ep)
	}
}

// UpdateOrderExecutionLimits updates order execution limits this is overridable
func (b *Base) UpdateOrderExecutionLimits(_ context.Context, _ asset.Item) error {
	return common.ErrNotYetImplemented
}

// DisableAssetWebsocketSupport disables websocket functionality for the
// supplied asset item. In the case that websocket functionality has not yet
// been implemented for that specific asset type. This is a base method to
// check availability of asset type.
func (b *Base) DisableAssetWebsocketSupport(aType asset.Item) error {
	if !b.SupportsAsset(aType) {
		return fmt.Errorf("%s %w",
			aType,
			asset.ErrNotSupported)
	}
	b.AssetWebsocketSupport.m.Lock()
	if b.AssetWebsocketSupport.unsupported == nil {
		b.AssetWebsocketSupport.unsupported = make(map[asset.Item]bool)
	}
	b.AssetWebsocketSupport.unsupported[aType] = true
	b.AssetWebsocketSupport.m.Unlock()
	return nil
}

// IsAssetWebsocketSupported checks to see if the supplied asset type is
// supported by websocket.
func (a *AssetWebsocketSupport) IsAssetWebsocketSupported(aType asset.Item) bool {
	a.m.RLock()
	defer a.m.RUnlock()
	return a.unsupported == nil || !a.unsupported[aType]
}

// UpdateCurrencyStates updates currency states
func (b *Base) UpdateCurrencyStates(ctx context.Context, a asset.Item) error {
	return common.ErrNotYetImplemented
}

// GetAvailableTransferChains returns a list of supported transfer chains based
// on the supplied cryptocurrency
func (b *Base) GetAvailableTransferChains(_ context.Context, _ currency.Code) ([]string, error) {
	return nil, common.ErrFunctionNotSupported
}

// CalculatePNL is an overridable function to allow PNL to be calculated on an
// open position
// It will also determine whether the position is considered to be liquidated
// For live trading, an overrided function may wish to confirm the liquidation by
// requesting the status of the asset
func (b *Base) CalculatePNL(context.Context, *order.PNLCalculatorRequest) (*order.PNLResult, error) {
	return nil, common.ErrNotYetImplemented
}

// ScaleCollateral is an overridable function to determine how much
// collateral is usable in futures positions
func (b *Base) ScaleCollateral(context.Context, *order.CollateralCalculator) (*order.CollateralByCurrency, error) {
	return nil, common.ErrNotYetImplemented
}

// CalculateTotalCollateral takes in n collateral calculators to determine an overall
// standing in a singular currency. See FTX's implementation
func (b *Base) CalculateTotalCollateral(ctx context.Context, calculator *order.TotalCollateralCalculator) (*order.TotalCollateralResponse, error) {
	return nil, common.ErrNotYetImplemented
}

// GetFuturesPositions returns futures positions according to the provided parameters
func (b *Base) GetFuturesPositions(context.Context, asset.Item, currency.Pair, time.Time, time.Time) ([]order.Detail, error) {
	return nil, common.ErrNotYetImplemented
}
