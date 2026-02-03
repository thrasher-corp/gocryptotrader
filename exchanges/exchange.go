package exchange

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"
	"unicode"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/dispatch"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/collateral"
	"github.com/thrasher-corp/gocryptotrader/exchanges/currencystate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/banking"
)

const (
	warningBase64DecryptSecretKeyFailed = "exchange %s unable to base64 decode secret key.. Disabling Authenticated API support" //nolint // False positive (G101: Potential hardcoded credentials)
	// DefaultHTTPTimeout is the default HTTP/HTTPS Timeout for exchange requests
	DefaultHTTPTimeout = time.Second * 15
	// DefaultWebsocketResponseCheckTimeout is the default delay in checking for an expected websocket response
	DefaultWebsocketResponseCheckTimeout = time.Millisecond * 50
	// DefaultWebsocketResponseMaxLimit is the default max wait for an expected websocket response before a timeout
	DefaultWebsocketResponseMaxLimit = time.Second * 7
	// DefaultWebsocketOrderbookBufferLimit is the maximum number of orderbook updates that get stored before being applied
	DefaultWebsocketOrderbookBufferLimit = 5
)

// Public Errors
var (
	ErrSettingProxyAddress  = errors.New("error setting proxy address")
	ErrEndpointPathNotFound = errors.New("no endpoint path found for the given key")
	ErrSymbolNotMatched     = errors.New("symbol cannot be matched")
)

var (
	errEndpointStringNotFound            = errors.New("endpoint string not found")
	errConfigPairFormatRequiresDelimiter = errors.New("config pair format requires delimiter")
	errSetDefaultsNotCalled              = errors.New("set defaults not called")
	errExchangeIsNil                     = errors.New("exchange is nil")
	errInvalidEndpointKey                = errors.New("invalid endpoint key")
)

// SetRequester sets the instance of the requester
func (b *Base) SetRequester(r *request.Requester) error {
	if r == nil {
		return fmt.Errorf("%s cannot set requester, no requester provided", b.Name)
	}

	b.Requester = r
	return nil
}

// SetClientProxyAddress sets a proxy address for REST and websocket requests
func (b *Base) SetClientProxyAddress(addr string) error {
	if addr == "" {
		return nil
	}
	proxy, err := url.Parse(addr)
	if err != nil {
		return fmt.Errorf("%w %w", ErrSettingProxyAddress, err)
	}

	err = b.Requester.SetProxy(proxy)
	if err != nil {
		return err
	}

	if b.Websocket != nil {
		err = b.Websocket.SetProxyAddress(context.TODO(), addr)
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

		b.SetSubscriptionsFromConfig()

		b.Features.Enabled.AutoPairUpdates = b.Config.Features.Enabled.AutoPairUpdates
	}
}

// SetSubscriptionsFromConfig sets the subscriptions from config
// If the subscriptions config is empty then Config will be updated from the exchange subscriptions,
// allowing e.SetDefaults to set default subscriptions for an exchange to update user's config
// Subscriptions not Enabled are skipped, meaning that e.Features.Subscriptions only contains Enabled subscriptions
func (b *Base) SetSubscriptionsFromConfig() {
	b.settingsMutex.Lock()
	defer b.settingsMutex.Unlock()
	if len(b.Config.Features.Subscriptions) == 0 {
		// Set config from the defaults, including any disabled subscriptions
		b.Config.Features.Subscriptions = b.Features.Subscriptions
	}
	b.Features.Subscriptions = b.Config.Features.Subscriptions.Enabled()
	if b.Verbose {
		names := make([]string, 0, len(b.Features.Subscriptions))
		for _, s := range b.Features.Subscriptions {
			names = append(names, s.Channel)
		}
		log.Debugf(log.ExchangeSys, "Set %v 'Subscriptions' to %v", b.Name, strings.Join(names, ", "))
	}
}

// SupportsRESTTickerBatchUpdates returns whether or not the
// exchange supports REST batch ticker fetching
func (b *Base) SupportsRESTTickerBatchUpdates() bool {
	return b.Features.Supports.RESTCapabilities.TickerBatching
}

// SupportsAutoPairUpdates returns whether or not the exchange supports
// auto currency pair updating
func (b *Base) SupportsAutoPairUpdates() bool {
	return b.Features.Supports.RESTCapabilities.AutoPairUpdates ||
		b.Features.Supports.WebsocketCapabilities.AutoPairUpdates
}

// GetLastPairsUpdateTime returns the unix timestamp of when the exchanges
// currency pairs were last updated
func (b *Base) GetLastPairsUpdateTime() int64 {
	return b.CurrencyPairs.LastUpdated
}

// GetAssetTypes returns either the enabled or available asset types for an
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
			return asset.Empty, err
		}
		if avail.Contains(c, true) {
			return assetTypes[i], nil
		}
	}
	return asset.Empty, errors.New("asset type not associated with currency pair")
}

// GetPairAndAssetTypeRequestFormatted returns the pair and the asset type
// when there is distinct differentiation between exchange request symbols asset
// types. e.g. "BTC-USD" Spot and "BTC_USD" PERP request formatted.
func (b *Base) GetPairAndAssetTypeRequestFormatted(symbol string) (currency.Pair, asset.Item, error) {
	if symbol == "" {
		return currency.EMPTYPAIR, asset.Empty, currency.ErrCurrencyPairEmpty
	}
	assetTypes := b.GetAssetTypes(true)
	for i := range assetTypes {
		pFmt, err := b.GetPairFormat(assetTypes[i], true)
		if err != nil {
			return currency.EMPTYPAIR, asset.Empty, err
		}

		enabled, err := b.GetEnabledPairs(assetTypes[i])
		if err != nil {
			return currency.EMPTYPAIR, asset.Empty, err
		}
		for j := range enabled {
			if pFmt.Format(enabled[j]) == symbol {
				return enabled[j], assetTypes[i], nil
			}
		}
	}
	return currency.EMPTYPAIR, asset.Empty, ErrSymbolNotMatched
}

// GetClientBankAccounts returns banking details associated with
// a client for withdrawal purposes
func (b *Base) GetClientBankAccounts(exchangeName, withdrawalCurrency string) (*banking.Account, error) {
	cfg := config.GetConfig()
	return cfg.GetClientBankAccounts(exchangeName, withdrawalCurrency)
}

// GetExchangeBankAccounts returns banking details associated with an exchange for funding purposes
func (b *Base) GetExchangeBankAccounts(id, depositCurrency string) (*banking.Account, error) {
	cfg := config.GetConfig()
	return cfg.GetExchangeBankAccounts(b.Name, id, depositCurrency)
}

// SetCurrencyPairFormat checks the exchange request and config currency pair
// formats and syncs it with the exchanges SetDefault settings
func (b *Base) SetCurrencyPairFormat() error {
	if b.Config.CurrencyPairs == nil {
		b.Config.CurrencyPairs = new(currency.PairsManager)
	}

	b.Config.CurrencyPairs.UseGlobalFormat = b.CurrencyPairs.UseGlobalFormat
	if b.Config.CurrencyPairs.UseGlobalFormat {
		b.Config.CurrencyPairs.RequestFormat = b.CurrencyPairs.RequestFormat
		b.Config.CurrencyPairs.ConfigFormat = b.CurrencyPairs.ConfigFormat
		return nil
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
				return err
			}
			err = b.Config.CurrencyPairs.Store(assetTypes[x], ps)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// SetConfigPairs sets the exchanges currency pairs to the pairs set in the config
func (b *Base) SetConfigPairs() error {
	assetTypes := b.Config.CurrencyPairs.GetAssetTypes(false)
	exchangeAssets := b.CurrencyPairs.GetAssetTypes(false)
	for _, a := range assetTypes {
		if !exchangeAssets.Contains(a) {
			log.Warnf(log.ExchangeSys, "%s exchange asset type %s unsupported, please manually remove from configuration", b.Name, a)
			continue
		}
		if !b.Config.CurrencyPairs.UseGlobalFormat {
			// TODO: Should be in SetCurrencyPairFormat. See #1748
			if err := b.setConfigPairFormatFromExchange(a); err != nil {
				return err
			}
		}

		cfgPS, err := b.Config.CurrencyPairs.Get(a)
		if err != nil {
			return err
		}
		if err := b.CurrencyPairs.StorePairs(a, cfgPS.Available, false); err != nil {
			return err
		}
		if err := b.CurrencyPairs.StorePairs(a, cfgPS.Enabled, true); err != nil {
			return err
		}

		var enabledAsset bool
		if b.Config.CurrencyPairs.IsAssetEnabled(a) == nil {
			enabledAsset = true
		}

		// Must happen after StorePairs, which would have enabled the asset automatically
		if err := b.CurrencyPairs.SetAssetEnabled(a, enabledAsset); err != nil {
			return err
		}
	}
	return nil
}

// setConfigPairFormatFromExchange sets the config formats from the exchange's format
// This deprecated behaviour will be removed, because config should not be backloaded from runtime
func (b *Base) setConfigPairFormatFromExchange(a asset.Item) error {
	ps, err := b.CurrencyPairs.Get(a)
	if err != nil {
		return err
	}
	if ps.ConfigFormat != nil {
		if err := b.Config.CurrencyPairs.StoreFormat(a, ps.ConfigFormat, true); err != nil {
			return err
		}
	}
	if ps.RequestFormat != nil {
		if err := b.Config.CurrencyPairs.StoreFormat(a, ps.RequestFormat, false); err != nil {
			return err
		}
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

// GetPairFormat returns the pair format based on the exchange and asset type
func (b *Base) GetPairFormat(a asset.Item, r bool) (currency.PairFormat, error) {
	return b.CurrencyPairs.GetFormat(a, r)
}

// GetEnabledPairs is a method that returns the enabled currency pairs of
// the exchange by asset type, if the asset type is disabled this will return no
// enabled pairs
func (b *Base) GetEnabledPairs(a asset.Item) (currency.Pairs, error) {
	err := b.CurrencyPairs.IsAssetEnabled(a)
	if err != nil {
		return nil, err
	}
	format, err := b.GetPairFormat(a, false)
	if err != nil {
		return nil, err
	}
	enabledPairs, err := b.CurrencyPairs.GetPairs(a, true)
	if err != nil {
		return nil, err
	}
	return enabledPairs.Format(format), nil
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
			formattedPair := pairs[j].Format(format)
			if strings.EqualFold(formattedPair.String(), p) {
				return formattedPair, assetTypes[i], nil
			}
		}
	}
	return currency.EMPTYPAIR, asset.Empty, fmt.Errorf("%s %w", p, currency.ErrPairNotFound)
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
	return pairs.Format(format), nil
}

// SupportsPair returns true or not whether a currency pair exists in the
// exchange available currencies or not
func (b *Base) SupportsPair(p currency.Pair, enabledPairs bool, assetType asset.Item) error {
	var pairs currency.Pairs
	var err error
	if enabledPairs {
		pairs, err = b.GetEnabledPairs(assetType)
	} else {
		pairs, err = b.GetAvailablePairs(assetType)
	}
	if err != nil {
		return err
	}
	if pairs.Contains(p, false) {
		return nil
	}
	return fmt.Errorf("%w %v", currency.ErrCurrencyNotSupported, p)
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
	if p.IsEmpty() {
		return currency.EMPTYPAIR, currency.ErrCurrencyPairEmpty
	}
	pairFmt, err := b.GetPairFormat(assetType, true)
	if err != nil {
		return currency.EMPTYPAIR, err
	}
	return p.Format(pairFmt), nil
}

// SetEnabled is a method that sets if the exchange is enabled
func (b *Base) SetEnabled(enabled bool) {
	b.settingsMutex.Lock()
	b.Enabled = enabled
	b.settingsMutex.Unlock()
}

// IsEnabled is a method that returns if the current exchange is enabled
func (b *Base) IsEnabled() bool {
	if b == nil {
		return false
	}
	b.settingsMutex.RLock()
	defer b.settingsMutex.RUnlock()
	return b.Enabled
}

// SetupDefaults sets the exchange settings based on the supplied config
func (b *Base) SetupDefaults(exch *config.Exchange) error {
	if err := exch.Validate(); err != nil {
		return err
	}

	b.Enabled = true
	b.LoadedByConfig = true
	b.Config = exch
	b.Verbose = exch.Verbose

	b.API.AuthenticatedSupport = exch.API.AuthenticatedSupport
	b.API.AuthenticatedWebsocketSupport = exch.API.AuthenticatedWebsocketSupport
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

	if err := b.SetHTTPClientTimeout(exch.HTTPTimeout); err != nil {
		return err
	}

	if exch.CurrencyPairs == nil {
		exch.CurrencyPairs = &b.CurrencyPairs
		a := exch.CurrencyPairs.GetAssetTypes(false)
		for i := range a {
			if err := exch.CurrencyPairs.SetAssetEnabled(a[i], true); err != nil {
				return err
			}
		}
	}

	b.HTTPDebugging = exch.HTTPDebugging
	b.BypassConfigFormatUpgrades = exch.CurrencyPairs.BypassConfigFormatUpgrades
	if err := b.SetHTTPClientUserAgent(exch.HTTPUserAgent); err != nil {
		return err
	}

	if err := b.SetCurrencyPairFormat(); err != nil {
		return err
	}

	if err := b.SetConfigPairs(); err != nil {
		return err
	}

	b.SetFeatureDefaults()

	if b.API.Endpoints == nil {
		b.API.Endpoints = b.NewEndpoints()
	}

	if err := b.SetAPIURL(); err != nil {
		return err
	}

	b.SetAPICredentialDefaults()

	if err := b.SetClientProxyAddress(exch.ProxyAddress); err != nil {
		return err
	}

	b.BaseCurrencies = exch.BaseCurrencies

	if exch.Orderbook.VerificationBypass {
		log.Warnf(log.ExchangeSys, "%s orderbook verification has been bypassed via config.", b.Name)
	}

	if b.Accounts == nil {
		var err error
		if b.Accounts, err = accounts.GetStore().GetExchangeAccounts(b); err != nil {
			return err
		}
	}

	b.ValidateOrderbook = !exch.Orderbook.VerificationBypass

	b.States = currencystate.NewCurrencyStates()

	return nil
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
	cPairs := make(currency.Pairs, len(pairs))
	copy(cPairs, pairs)
	for x := range pairs {
		cPairs[x] = pairs[x].Format(pairFmt)
	}

	err = b.CurrencyPairs.StorePairs(assetType, cPairs, enabled)
	if err != nil {
		return err
	}
	return b.Config.CurrencyPairs.StorePairs(assetType, cPairs, enabled)
}

// EnsureOnePairEnabled not all assets have pairs, eg options
// search for an asset that does and enable one if none are enabled
// error if no currency pairs found for an entire exchange
func (b *Base) EnsureOnePairEnabled() error {
	pair, item, err := b.CurrencyPairs.EnsureOnePairEnabled()
	if err != nil {
		return err
	}
	if !pair.IsEmpty() {
		log.Warnf(log.ExchangeSys, "%v had no enabled pairs, %v %v pair has been enabled", b.Name, item, pair)
	}
	return nil
}

// UpdatePairs updates the exchange currency pairs for either enabledPairs or availablePairs
func (b *Base) UpdatePairs(incoming currency.Pairs, a asset.Item, enabled bool) error {
	if len(incoming) == 0 && !enabled {
		// Exchange reports a successful response (HTTP 200) but provides no currency pairs, so we preserve the existing
		// available pairs.
		return fmt.Errorf("%w: updating available pairs", currency.ErrCurrencyPairsEmpty)
	}
	pFmt, err := b.GetPairFormat(a, false)
	if err != nil {
		return err
	}

	incoming, err = incoming.ValidateAndConform(pFmt, b.BypassConfigFormatUpgrades)
	if err != nil {
		return err
	}

	oldPairs, err := b.CurrencyPairs.GetPairs(a, enabled)
	if err != nil {
		return err
	}

	diff, err := oldPairs.FindDifferences(incoming, pFmt)
	if err != nil {
		return err
	}

	updateType := "enabled"
	if !enabled {
		updateType = "available"
	}

	if len(diff.New) > 0 {
		log.Debugf(log.ExchangeSys, "%s Updating %s pairs [%v] - Added: %s.\n", b.Name, updateType, strings.ToUpper(a.String()), diff.New)
	}
	if len(diff.Remove) > 0 {
		log.Debugf(log.ExchangeSys, "%s Updating %s pairs [%v] - Removed: %s.\n", b.Name, updateType, strings.ToUpper(a.String()), diff.Remove)
	}

	if err := common.NilGuard(b.Config, b.Config.CurrencyPairs); err != nil {
		return err
	}

	if err := b.Config.CurrencyPairs.StorePairs(a, incoming, enabled); err != nil {
		return err
	}
	if err := b.CurrencyPairs.StorePairs(a, incoming, enabled); err != nil {
		return err
	}

	if enabled {
		return nil
	}

	// This section checks for differences after an available pairs adjustment
	// which will remove currency pairs from enabled pairs that have been
	// disabled by an exchange, adjust the entire list of enabled pairs if there
	// is a required formatting change and it will also capture unintentional
	// client inputs e.g. a client can enter `linkusd` via config and loaded
	// into memory that might be unintentionally formatted too `lin-kusd` it
	// will match that against the correct available pair in memory and apply
	// correct formatting (LINK-USD) instead of being removed altogether which
	// will require a shutdown and update of the config file to enable that
	// asset.

	enabledPairs, err := b.CurrencyPairs.GetPairs(a, true)
	if err != nil &&
		!errors.Is(err, currency.ErrPairNotContainedInAvailablePairs) &&
		!errors.Is(err, currency.ErrPairDuplication) {
		return err
	}

	if err == nil && !enabledPairs.HasFormatDifference(pFmt) {
		return nil
	}

	diff, err = enabledPairs.FindDifferences(incoming, pFmt)
	if err != nil {
		return err
	}

	check := make(map[string]bool)
	var target int
	for x := range enabledPairs {
		pairNoFmt := currency.EMPTYFORMAT.Format(enabledPairs[x])
		if check[pairNoFmt] {
			diff.Remove = diff.Remove.Add(enabledPairs[x])
			continue
		}
		check[pairNoFmt] = true

		if !diff.Remove.Contains(enabledPairs[x], true) {
			enabledPairs[target] = enabledPairs[x].Format(pFmt)
		} else {
			var match currency.Pair
			match, err = incoming.DeriveFrom(pairNoFmt)
			if err != nil {
				continue
			}
			diff.Remove = diff.Remove.Remove(enabledPairs[x])
			enabledPairs[target] = match.Format(pFmt)
		}
		target++
	}

	enabledPairs = enabledPairs[:target]
	if len(enabledPairs) == 0 && len(incoming) > 0 {
		// NOTE: If enabled pairs are not populated for any reason.
		var randomPair currency.Pair
		randomPair, err = incoming.GetRandomPair()
		if err != nil {
			return err
		}
		log.Debugf(log.ExchangeSys, "%s Enabled pairs missing for %s. Added %s.\n", b.Name, strings.ToUpper(a.String()), randomPair)
		enabledPairs = currency.Pairs{randomPair}
	}

	if len(diff.Remove) > 0 {
		log.Debugf(log.ExchangeSys, "%s Checked and updated enabled pairs [%v] - Removed: %s.\n", b.Name, strings.ToUpper(a.String()), diff.Remove)
	}
	if err := b.Config.CurrencyPairs.StorePairs(a, enabledPairs, true); err != nil {
		return err
	}
	return b.CurrencyPairs.StorePairs(a, enabledPairs, true)
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
			err = b.API.Endpoints.SetRunningURL(RestSpot.String(), b.Config.API.OldEndPoints.URL)
			if err != nil {
				return err
			}
			checkInsecureEndpoint(b.Config.API.OldEndPoints.URL)
		}
		if b.Config.API.OldEndPoints.URLSecondary != "" && b.Config.API.OldEndPoints.URLSecondary != config.APIURLNonDefaultMessage {
			err = b.API.Endpoints.SetRunningURL(RestSpotSupplementary.String(), b.Config.API.OldEndPoints.URLSecondary)
			if err != nil {
				return err
			}
			checkInsecureEndpoint(b.Config.API.OldEndPoints.URLSecondary)
		}
		if b.Config.API.OldEndPoints.WebsocketURL != "" && b.Config.API.OldEndPoints.WebsocketURL != config.WebsocketURLNonDefaultMessage {
			err = b.API.Endpoints.SetRunningURL(WebsocketSpot.String(), b.Config.API.OldEndPoints.WebsocketURL)
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

			err = b.API.Endpoints.SetRunningURL(key, val)
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
	for i := range uint(32) {
		var check uint32 = 1 << i
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

// SupportsAsset whether or not the supplied asset is supported by the exchange
func (b *Base) SupportsAsset(a asset.Item) bool {
	return b.CurrencyPairs.IsAssetSupported(a)
}

// PrintEnabledPairs prints the exchanges enabled asset pairs
func (b *Base) PrintEnabledPairs() {
	for k, v := range b.CurrencyPairs.Pairs {
		log.Infof(log.ExchangeSys, "%s Asset type %v:\n\t Enabled pairs: %v", b.Name, strings.ToUpper(k.String()), v.Enabled)
	}
}

// GetBase returns the exchange base
func (b *Base) GetBase() *Base { return b }

// CheckTransientError catches transient errors and returns nil if found, used
// for validation of API credentials
func (b *Base) CheckTransientError(err error) error {
	if _, ok := err.(net.Error); ok {
		log.Warnf(log.ExchangeSys, "%s net error captured, will not disable authentication %s", b.Name, err)
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

// SetAssetPairStore initialises and stores a defined asset format
func (b *Base) SetAssetPairStore(a asset.Item, f currency.PairStore) error {
	if a.String() == "" {
		return asset.ErrInvalidAsset
	}

	if f.RequestFormat == nil || f.ConfigFormat == nil {
		return currency.ErrPairFormatIsNil
	}

	if f.ConfigFormat.Delimiter == "" {
		return errConfigPairFormatRequiresDelimiter
	}

	if b.CurrencyPairs.Pairs == nil {
		b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	}

	b.CurrencyPairs.Pairs[a] = &f

	return nil
}

// SetGlobalPairsManager sets defined asset and pairs management system with global formatting
func (b *Base) SetGlobalPairsManager(reqFmt, cfgFmt *currency.PairFormat, assets ...asset.Item) error {
	if reqFmt == nil {
		return fmt.Errorf("%s cannot set pairs manager, request pair format not provided", b.Name)
	}

	if cfgFmt == nil {
		return fmt.Errorf("%s cannot set pairs manager, config pair format not provided",
			b.Name)
	}

	if len(assets) == 0 {
		return fmt.Errorf("%s cannot set pairs manager, no assets provided",
			b.Name)
	}

	if cfgFmt.Delimiter == "" {
		return fmt.Errorf("exchange %s cannot set global pairs manager %w for assets %s",
			b.Name, errConfigPairFormatRequiresDelimiter, assets)
	}

	b.CurrencyPairs.UseGlobalFormat = true
	b.CurrencyPairs.RequestFormat = reqFmt
	b.CurrencyPairs.ConfigFormat = cfgFmt

	if b.CurrencyPairs.Pairs != nil {
		return fmt.Errorf("%s cannot set pairs manager, pairs already set",
			b.Name)
	}

	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)

	for i := range assets {
		if assets[i].String() == "" {
			b.CurrencyPairs.Pairs = nil
			return fmt.Errorf("%s cannot set pairs manager, asset is empty string", b.Name)
		}
		b.CurrencyPairs.Pairs[assets[i]] = new(currency.PairStore)
		b.CurrencyPairs.Pairs[assets[i]].AssetEnabled = true
		b.CurrencyPairs.Pairs[assets[i]].ConfigFormat = cfgFmt
		b.CurrencyPairs.Pairs[assets[i]].RequestFormat = reqFmt
	}

	return nil
}

// GetWebsocket returns a pointer to the exchange websocket
func (b *Base) GetWebsocket() (*websocket.Manager, error) {
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
	return b.Websocket.FlushChannels(context.TODO())
}

// SubscribeToWebsocketChannels appends to ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle subscribing
func (b *Base) SubscribeToWebsocketChannels(channels subscription.List) error {
	if b.Websocket == nil {
		return common.ErrFunctionNotSupported
	}
	return b.Websocket.SubscribeToChannels(context.TODO(), b.Websocket.Conn, channels)
}

// UnsubscribeToWebsocketChannels removes from ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle unsubscribing
func (b *Base) UnsubscribeToWebsocketChannels(channels subscription.List) error {
	if b.Websocket == nil {
		return common.ErrFunctionNotSupported
	}
	return b.Websocket.UnsubscribeChannels(context.TODO(), b.Websocket.Conn, channels)
}

// GetSubscriptions returns a copied list of subscriptions
func (b *Base) GetSubscriptions() (subscription.List, error) {
	if b.Websocket == nil {
		return nil, common.ErrFunctionNotSupported
	}
	return b.Websocket.GetSubscriptions(), nil
}

// GetSubscriptionTemplate returns a template for a given subscription; See exchange/subscription/README.md for more information
func (b *Base) GetSubscriptionTemplate(*subscription.Subscription) (*template.Template, error) {
	return nil, common.ErrFunctionNotSupported
}

// AuthenticateWebsocket sends an authentication message to the websocket
func (b *Base) AuthenticateWebsocket(_ context.Context) error {
	return common.ErrFunctionNotSupported
}

// CanUseAuthenticatedWebsocketEndpoints calls b.Websocket.CanUseAuthenticatedEndpoints
// Used to avoid import cycles on websocket.Manager
func (b *Base) CanUseAuthenticatedWebsocketEndpoints() bool {
	return b.Websocket != nil && b.Websocket.CanUseAuthenticatedEndpoints()
}

// klineIntervalEnabled returns if requested interval is enabled on exchange
func (b *Base) klineIntervalEnabled(in kline.Interval) bool {
	// TODO: Add in the ability to use custom klines
	return b.Features.Enabled.Kline.Intervals.ExchangeSupported(in)
}

// FormatExchangeKlineInterval returns Interval to string
// Exchanges can override this if they require custom formatting
func (b *Base) FormatExchangeKlineInterval(in kline.Interval) string {
	return strconv.FormatFloat(in.Duration().Seconds(), 'f', 0, 64)
}

// verifyKlineParameters verifies whether the pair, asset and interval are enabled on the exchange
func (b *Base) verifyKlineParameters(pair currency.Pair, a asset.Item, interval kline.Interval) error {
	if err := b.CurrencyPairs.IsAssetEnabled(a); err != nil {
		return err
	}

	if ok, err := b.IsPairEnabled(pair, a); err != nil {
		return err
	} else if !ok {
		return fmt.Errorf("%w: %v", currency.ErrPairNotEnabled, pair)
	}

	if !b.klineIntervalEnabled(interval) {
		return fmt.Errorf("%w: %v", kline.ErrInvalidInterval, interval)
	}

	return nil
}

// AddTradesToBuffer is a helper function that will only
// add trades to the buffer if it is allowed
func (b *Base) AddTradesToBuffer(trades ...trade.Data) error {
	if !b.IsSaveTradeDataEnabled() {
		return nil
	}
	return trade.AddTradesToBuffer(trades...)
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
		if err := e.SetRunningURL(k.String(), v); err != nil {
			return err
		}
	}
	return nil
}

// SetRunningURL populates running URLs map
func (e *Endpoints) SetRunningURL(endpoint, val string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := validateKey(endpoint); err != nil {
		return err
	}
	if _, err := url.ParseRequestURI(val); err != nil {
		return fmt.Errorf("parse request URI for %s=%q (exchange %s): %w", endpoint, val, e.Exchange, err)
	}
	e.defaults[endpoint] = val
	return nil
}

func validateKey(keyVal string) error {
	for x := range keyURLs {
		if keyURLs[x].String() == keyVal {
			return nil
		}
	}
	return errInvalidEndpointKey
}

// GetURL gets default url from URLs map
func (e *Endpoints) GetURL(endpoint URL) (string, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	val, ok := e.defaults[endpoint.String()]
	if !ok {
		return "", fmt.Errorf("%w %v", ErrEndpointPathNotFound, endpoint)
	}
	return val, nil
}

// GetURLMap gets all urls for either running or default map based on the bool value supplied
func (e *Endpoints) GetURLMap() map[string]string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return maps.Clone(e.defaults)
}

// GetCachedOpenInterest returns open interest data if the exchange
// supports open interest in ticker data
func (b *Base) GetCachedOpenInterest(_ context.Context, k ...key.PairAsset) ([]futures.OpenInterest, error) {
	if !b.Features.Supports.FuturesCapabilities.OpenInterest.Supported ||
		!b.Features.Supports.FuturesCapabilities.OpenInterest.SupportedViaTicker {
		return nil, common.ErrFunctionNotSupported
	}
	if len(k) == 0 {
		ticks, err := ticker.GetExchangeTickers(b.Name)
		if err != nil {
			return nil, err
		}
		resp := make([]futures.OpenInterest, 0, len(ticks))
		for i := range ticks {
			if ticks[i].OpenInterest <= 0 {
				continue
			}
			resp = append(resp, futures.OpenInterest{
				Key:          key.NewExchangeAssetPair(b.Name, ticks[i].AssetType, ticks[i].Pair),
				OpenInterest: ticks[i].OpenInterest,
			})
		}
		sort.Slice(resp, func(i, j int) bool {
			return resp[i].Key.Base.Symbol < resp[j].Key.Base.Symbol
		})
		return resp, nil
	}
	resp := make([]futures.OpenInterest, len(k))
	for i := range k {
		t, err := ticker.GetTicker(b.Name, k[i].Pair(), k[i].Asset)
		if err != nil {
			return nil, err
		}
		resp[i] = futures.OpenInterest{
			Key:          key.NewExchangeAssetPair(b.Name, t.AssetType, t.Pair),
			OpenInterest: t.OpenInterest,
		}
	}
	return resp, nil
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
	case RestFuturesSupplementary:
		return restFuturesSupplementaryURL
	case RestUSDCMargined:
		return restUSDCMarginedFuturesURL
	case RestSandbox:
		return restSandboxURL
	case RestSwap:
		return restSwapURL
	case WebsocketSpot:
		return websocketSpotURL
	case WebsocketCoinMargined:
		return websocketCoinMarginedURL
	case WebsocketUSDTMargined:
		return websocketUSDTMarginedURL
	case WebsocketUSDCMargined:
		return websocketUSDCMarginedURL
	case WebsocketOptions:
		return websocketOptionsURL
	case WebsocketTrade:
		return websocketTradeURL
	case WebsocketPrivate:
		return websocketPrivateURL
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
	case restFuturesSupplementaryURL:
		return RestFuturesSupplementary, nil
	case restUSDCMarginedFuturesURL:
		return RestUSDCMargined, nil
	case restSandboxURL:
		return RestSandbox, nil
	case restSwapURL:
		return RestSwap, nil
	case websocketSpotURL:
		return WebsocketSpot, nil
	case websocketCoinMarginedURL:
		return WebsocketCoinMargined, nil
	case websocketUSDTMarginedURL:
		return WebsocketUSDTMargined, nil
	case websocketUSDCMarginedURL:
		return WebsocketUSDCMargined, nil
	case websocketOptionsURL:
		return WebsocketOptions, nil
	case websocketTradeURL:
		return WebsocketTrade, nil
	case websocketPrivateURL:
		return WebsocketPrivate, nil
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
		return Invalid, fmt.Errorf("%w %q", errEndpointStringNotFound, ep)
	}
}

// DisableAssetWebsocketSupport disables websocket functionality for the
// supplied asset item. In the case that websocket functionality has not yet
// been implemented for that specific asset type. This is a base method to
// check availability of asset type.
func (b *Base) DisableAssetWebsocketSupport(aType asset.Item) error {
	if !b.SupportsAsset(aType) {
		return fmt.Errorf("%s %w", aType, asset.ErrNotSupported)
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
func (b *Base) UpdateCurrencyStates(_ context.Context, _ asset.Item) error {
	return common.ErrNotYetImplemented
}

// GetAvailableTransferChains returns a list of supported transfer chains based
// on the supplied cryptocurrency
func (b *Base) GetAvailableTransferChains(_ context.Context, _ currency.Code) ([]string, error) {
	return nil, common.ErrFunctionNotSupported
}

// HasAssetTypeAccountSegregation returns if the accounts are divided into asset
// types instead of just being denoted as spot holdings.
func (b *Base) HasAssetTypeAccountSegregation() bool {
	return b.Features.Supports.RESTCapabilities.HasAssetTypeAccountSegregation
}

// GetPositionSummary returns stats for a future position
func (b *Base) GetPositionSummary(context.Context, *futures.PositionSummaryRequest) (*futures.PositionSummary, error) {
	return nil, common.ErrNotYetImplemented
}

// GetKlineRequest returns a helper for the fetching of candle/kline data for
// a single request within a pre-determined time window.
func (b *Base) GetKlineRequest(pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time, fixedAPICandleLength bool) (*kline.Request, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if !a.IsValid() {
		return nil, asset.ErrNotSupported
	}
	// NOTE: This allows for checking that the required kline interval is
	// supported by the exchange and/or can be constructed from lower time frame
	// intervals.
	exchangeInterval, err := b.Features.Enabled.Kline.Intervals.Construct(interval)
	if err != nil {
		return nil, err
	}

	err = b.verifyKlineParameters(pair, a, exchangeInterval)
	if err != nil {
		return nil, err
	}

	formatted, err := b.FormatExchangeCurrency(pair, a)
	if err != nil {
		return nil, err
	}

	limit, err := b.Features.Enabled.Kline.GetIntervalResultLimit(exchangeInterval)
	if err != nil {
		return nil, err
	}

	req, err := kline.CreateKlineRequest(b.Name, pair, formatted, a, interval, exchangeInterval, start, end, limit)
	if err != nil {
		return nil, err
	}

	// NOTE: The checks below makes sure a client is notified that using this
	// functionality will result in error if the total candles cannot be
	// theoretically retrieved.
	if fixedAPICandleLength {
		origCount := kline.TotalCandlesPerInterval(req.Start, req.End, interval)
		modifiedCount := kline.TotalCandlesPerInterval(req.Start, time.Now(), exchangeInterval)
		if modifiedCount > limit {
			errMsg := fmt.Sprintf("for %v %v candles between %v-%v. ",
				origCount,
				interval,
				start.Format(common.SimpleTimeFormatWithTimezone),
				end.Format(common.SimpleTimeFormatWithTimezone))
			if interval != exchangeInterval {
				errMsg += fmt.Sprintf("Request converts to %v %v candles. ",
					modifiedCount,
					exchangeInterval)
			}
			boundary := time.Now().Add(-exchangeInterval.Duration() * time.Duration(limit)) //nolint:gosec // TODO: Ensure limit can't overflow
			return nil, fmt.Errorf("%w %v, exceeding the limit of %v %v candles up to %v. Please reduce timeframe or use GetHistoricCandlesExtended",
				kline.ErrRequestExceedsExchangeLimits,
				errMsg,
				limit,
				exchangeInterval,
				boundary.Format(common.SimpleTimeFormatWithTimezone))
		}
	} else if count := kline.TotalCandlesPerInterval(req.Start, req.End, exchangeInterval); count > limit {
		return nil, fmt.Errorf("candle count exceeded: %d. The endpoint has a set candle limit return of %d candles. Candle data will be incomplete: %w",
			count,
			limit,
			kline.ErrRequestExceedsExchangeLimits)
	}

	return req, nil
}

// GetKlineExtendedRequest returns a helper for the fetching of candle/kline
// data for a *multi* request within a pre-determined time window. This has
// extended functionality to also break down calls to fetch total history.
func (b *Base) GetKlineExtendedRequest(pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.ExtendedRequest, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if !a.IsValid() {
		return nil, asset.ErrNotSupported
	}

	exchangeInterval, err := b.Features.Enabled.Kline.Intervals.Construct(interval)
	if err != nil {
		return nil, err
	}

	err = b.verifyKlineParameters(pair, a, exchangeInterval)
	if err != nil {
		return nil, err
	}

	formatted, err := b.FormatExchangeCurrency(pair, a)
	if err != nil {
		return nil, err
	}

	limit, err := b.Features.Enabled.Kline.GetIntervalResultLimit(exchangeInterval)
	if err != nil {
		return nil, err
	}

	r, err := kline.CreateKlineRequest(b.Name, pair, formatted, a, interval, exchangeInterval, start, end, limit)
	if err != nil {
		return nil, err
	}
	r.IsExtended = true

	dates, err := r.GetRanges(limit)
	if err != nil {
		return nil, err
	}

	return &kline.ExtendedRequest{Request: r, RangeHolder: dates}, nil
}

// Shutdown closes active websocket connections if available and then cleans up
// a REST requester instance.
func (b *Base) Shutdown() error {
	if b.Websocket != nil {
		err := b.Websocket.Shutdown()
		if err != nil && !errors.Is(err, websocket.ErrNotConnected) {
			return err
		}
	}
	return b.Requester.Shutdown()
}

// GetStandardConfig returns a standard default exchange config. Set defaults
// must populate base struct with exchange specific defaults before calling
// this function.
func (b *Base) GetStandardConfig() (*config.Exchange, error) {
	if b == nil {
		return nil, errExchangeIsNil
	}

	if b.Name == "" {
		return nil, errSetDefaultsNotCalled
	}

	exchCfg := new(config.Exchange)
	exchCfg.Name = b.Name
	exchCfg.Enabled = b.Enabled
	exchCfg.HTTPTimeout = DefaultHTTPTimeout
	exchCfg.BaseCurrencies = b.BaseCurrencies

	if b.SupportsWebsocket() {
		exchCfg.WebsocketResponseCheckTimeout = config.DefaultWebsocketResponseCheckTimeout
		exchCfg.WebsocketResponseMaxLimit = config.DefaultWebsocketResponseMaxLimit
		exchCfg.WebsocketTrafficTimeout = config.DefaultWebsocketTrafficTimeout
	}

	return exchCfg, nil
}

// Futures section

// CalculatePNL is an overridable function to allow PNL to be calculated on an
// open position
// It will also determine whether the position is considered to be liquidated
// For live trading, an overriding function may wish to confirm the liquidation by
// requesting the status of the asset
func (b *Base) CalculatePNL(context.Context, *futures.PNLCalculatorRequest) (*futures.PNLResult, error) {
	return nil, common.ErrNotYetImplemented
}

// ScaleCollateral is an overridable function to determine how much
// collateral is usable in futures positions
func (b *Base) ScaleCollateral(context.Context, *futures.CollateralCalculator) (*collateral.ByCurrency, error) {
	return nil, common.ErrNotYetImplemented
}

// CalculateTotalCollateral takes in n collateral calculators to determine an overall
// standing in a singular currency
func (b *Base) CalculateTotalCollateral(_ context.Context, _ *futures.TotalCollateralCalculator) (*futures.TotalCollateralResponse, error) {
	return nil, common.ErrNotYetImplemented
}

// GetCollateralCurrencyForContract returns the collateral currency for an asset and contract pair
func (b *Base) GetCollateralCurrencyForContract(_ asset.Item, _ currency.Pair) (currency.Code, asset.Item, error) {
	return currency.Code{}, asset.Empty, common.ErrNotYetImplemented
}

// GetCurrencyForRealisedPNL returns where to put realised PNL
// example 1: Bybit universal margin PNL is paid out in USD to your spot wallet
// example 2: Binance coin margined futures pays returns using the same currency eg BTC
func (b *Base) GetCurrencyForRealisedPNL(_ asset.Item, _ currency.Pair) (currency.Code, asset.Item, error) {
	return currency.Code{}, asset.Empty, common.ErrNotYetImplemented
}

// GetMarginRatesHistory returns the margin rate history for the supplied currency
func (b *Base) GetMarginRatesHistory(context.Context, *margin.RateHistoryRequest) (*margin.RateHistoryResponse, error) {
	return nil, common.ErrNotYetImplemented
}

// GetFuturesPositionSummary returns stats for a future position
func (b *Base) GetFuturesPositionSummary(context.Context, *futures.PositionSummaryRequest) (*futures.PositionSummary, error) {
	return nil, common.ErrNotYetImplemented
}

// GetFuturesPositions returns futures positions for all currencies
func (b *Base) GetFuturesPositions(context.Context, *futures.PositionsRequest) ([]futures.PositionDetails, error) {
	return nil, common.ErrNotYetImplemented
}

// GetFuturesPositionOrders returns futures positions orders
func (b *Base) GetFuturesPositionOrders(context.Context, *futures.PositionsRequest) ([]futures.PositionResponse, error) {
	return nil, common.ErrNotYetImplemented
}

// GetHistoricalFundingRates returns historical funding rates for a future
func (b *Base) GetHistoricalFundingRates(context.Context, *fundingrate.HistoricalRatesRequest) (*fundingrate.HistoricalRates, error) {
	return nil, common.ErrNotYetImplemented
}

// IsPerpetualFutureCurrency ensures a given asset and currency is a perpetual future
// differs by exchange
func (b *Base) IsPerpetualFutureCurrency(asset.Item, currency.Pair) (bool, error) {
	return false, common.ErrNotYetImplemented
}

// SetCollateralMode sets the account's collateral mode for the asset type
func (b *Base) SetCollateralMode(_ context.Context, _ asset.Item, _ collateral.Mode) error {
	return common.ErrNotYetImplemented
}

// GetCollateralMode returns the account's collateral mode for the asset type
func (b *Base) GetCollateralMode(_ context.Context, _ asset.Item) (collateral.Mode, error) {
	return 0, common.ErrNotYetImplemented
}

// SetMarginType sets the account's margin type for the asset type
func (b *Base) SetMarginType(_ context.Context, _ asset.Item, _ currency.Pair, _ margin.Type) error {
	return common.ErrNotYetImplemented
}

// ChangePositionMargin changes the margin type for a position
func (b *Base) ChangePositionMargin(_ context.Context, _ *margin.PositionChangeRequest) (*margin.PositionChangeResponse, error) {
	return nil, common.ErrNotYetImplemented
}

// SetLeverage sets the account's initial leverage for the asset type and pair
func (b *Base) SetLeverage(_ context.Context, _ asset.Item, _ currency.Pair, _ margin.Type, _ float64, _ order.Side) error {
	return common.ErrNotYetImplemented
}

// GetLeverage gets the account's initial leverage for the asset type and pair
func (b *Base) GetLeverage(_ context.Context, _ asset.Item, _ currency.Pair, _ margin.Type, _ order.Side) (float64, error) {
	return -1, common.ErrNotYetImplemented
}

// MatchSymbolWithAvailablePairs returns a currency pair based on the supplied
// symbol and asset type. If the string is expected to have a delimiter this
// will attempt to screen it out.
func (b *Base) MatchSymbolWithAvailablePairs(symbol string, a asset.Item, hasDelimiter bool) (currency.Pair, error) {
	if hasDelimiter {
		for x := range symbol {
			if unicode.IsPunct(rune(symbol[x])) {
				symbol = symbol[:x] + symbol[x+1:]
				break
			}
		}
	}
	return b.CurrencyPairs.Match(symbol, a)
}

// MatchSymbolCheckEnabled returns a currency pair based on the supplied symbol
// and asset type against the available pairs list. If the string is expected to
// have a delimiter this will attempt to screen it out. It will also check if
// the pair is enabled.
func (b *Base) MatchSymbolCheckEnabled(symbol string, a asset.Item, hasDelimiter bool) (pair currency.Pair, enabled bool, err error) {
	pair, err = b.MatchSymbolWithAvailablePairs(symbol, a, hasDelimiter)
	if err != nil {
		return pair, false, err
	}

	enabled, err = b.IsPairEnabled(pair, a)
	return
}

// IsPairEnabled checks if a pair is enabled for an enabled asset type.
// TODO: Optimisation map for enabled pair matching, instead of linear traversal.
func (b *Base) IsPairEnabled(pair currency.Pair, a asset.Item) (bool, error) {
	return b.CurrencyPairs.IsPairEnabled(pair, a)
}

// GetOpenInterest returns the open interest rate for a given asset pair
func (b *Base) GetOpenInterest(context.Context, ...key.PairAsset) ([]futures.OpenInterest, error) {
	return nil, common.ErrFunctionNotSupported
}

// ParallelChanOp performs a single method call in parallel across streams and waits to return any errors
func (b *Base) ParallelChanOp(ctx context.Context, channels subscription.List, m func(context.Context, subscription.List) error, batchSize int) error {
	var errs common.ErrorCollector
	for _, batchedSubs := range common.Batch(channels, batchSize) {
		errs.Go(func() error { return m(ctx, batchedSubs) })
	}
	return errs.Collect()
}

// Bootstrap function allows for exchange authors to supplement or override common startup actions
// If exchange.Bootstrap returns false or error it will not perform any other actions.
// If it returns true, or is not implemented by the exchange, it will:
// * Print debug startup information
// * UpdateOrderExecutionLimits
// * UpdateTradablePairs
func Bootstrap(ctx context.Context, b IBotExchange) error {
	if continueBootstrap, err := b.Bootstrap(ctx); !continueBootstrap || err != nil {
		return err
	}

	if b.IsVerbose() {
		if b.GetSupportedFeatures().Websocket {
			wsURL := ""
			wsEnabled := false
			if w, err := b.GetWebsocket(); err == nil {
				wsURL = w.GetWebsocketURL()
				wsEnabled = w.IsEnabled()
			}
			log.Debugf(log.ExchangeSys, "%s Websocket: %s. (url: %s)", b.GetName(), common.IsEnabled(wsEnabled), wsURL)
		} else {
			log.Debugf(log.ExchangeSys, "%s Websocket: Unsupported", b.GetName())
		}
		b.PrintEnabledPairs()
	}

	if b.GetEnabledFeatures().AutoPairUpdates {
		if err := b.UpdateTradablePairs(ctx); err != nil {
			return fmt.Errorf("failed to update tradable pairs: %w", err)
		}
	}

	var errs common.ErrorCollector
	for _, a := range b.GetAssetTypes(true) {
		errs.Go(func() error {
			if err := b.UpdateOrderExecutionLimits(ctx, a); err != nil && !errors.Is(err, common.ErrNotYetImplemented) {
				return fmt.Errorf("failed to set exchange order execution limits: %w", err)
			}
			return nil
		})
	}
	return errs.Collect()
}

// Bootstrap is a fallback method for exchange startup actions
// Exchange authors should override this if they wish to customise startup actions
// Return true or an error to all default Bootstrap actions to occur afterwards
// or false to signal that no further bootstrapping should occur
func (b *Base) Bootstrap(_ context.Context) (continueBootstrap bool, err error) {
	continueBootstrap = true
	return
}

// IsVerbose returns if the exchange is set to verbose
func (b *Base) IsVerbose() bool {
	return b.Verbose
}

// GetDefaultConfig returns a default exchange config
func GetDefaultConfig(ctx context.Context, exch IBotExchange) (*config.Exchange, error) {
	if exch == nil {
		return nil, errExchangeIsNil
	}

	if exch.GetName() == "" {
		exch.SetDefaults()
	}

	b := exch.GetBase()

	exchCfg, err := b.GetStandardConfig()
	if err != nil {
		return nil, err
	}

	if err := b.SetupDefaults(exchCfg); err != nil {
		return nil, err
	}

	if b.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err = exch.UpdateTradablePairs(ctx)
		if err != nil {
			return nil, err
		}
	}

	return exchCfg, nil
}

// GetCurrencyTradeURL returns the URL to the exchange's trade page for the given asset and currency pair
func (b *Base) GetCurrencyTradeURL(context.Context, asset.Item, currency.Pair) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// GetTradingRequirements returns the exchange's trading requirements.
func (b *Base) GetTradingRequirements() protocol.TradingRequirements {
	if b == nil {
		return protocol.TradingRequirements{}
	}
	return b.Features.TradingRequirements
}

// GetCachedTicker returns the ticker for a currency pair and asset type
// NOTE: UpdateTicker (or if supported UpdateTickers) method must be called first to update the ticker map
func (b *Base) GetCachedTicker(p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	return ticker.GetTicker(b.Name, p, assetType)
}

// GetCachedOrderbook returns an orderbook snapshot for the currency pair and asset type
// NOTE: UpdateOrderbook method must be called first to update the orderbook map
func (b *Base) GetCachedOrderbook(p currency.Pair, assetType asset.Item) (*orderbook.Book, error) {
	return orderbook.Get(b.Name, p, assetType)
}

// GetCachedSubAccounts retrieves all cached SubAccounts, filtered by credentials and asset
// NOTE: Accounts.Save method should be called first to populate the local cache
func (b *Base) GetCachedSubAccounts(ctx context.Context, assetType asset.Item) (accounts.SubAccounts, error) {
	creds, err := b.GetCredentials(ctx)
	if err != nil {
		return nil, err
	}
	return b.Accounts.SubAccounts(creds, assetType)
}

// GetCachedCurrencyBalances retrieves cached balances for all SubAccounts grouped by currency
// NOTE: Accounts.Save method should be called first to populate the local cache
func (b *Base) GetCachedCurrencyBalances(ctx context.Context, assetType asset.Item) (accounts.CurrencyBalances, error) {
	creds, err := b.GetCredentials(ctx)
	if err != nil {
		return nil, err
	}
	return b.Accounts.CurrencyBalances(creds, assetType)
}

// GetOrderExecutionLimits returns a limit based on the exchange, asset and pair from storage
func (b *Base) GetOrderExecutionLimits(a asset.Item, cp currency.Pair) (limits.MinMaxLevel, error) {
	return limits.GetOrderExecutionLimits(key.NewExchangeAssetPair(b.Name, a, cp))
}

// CheckOrderExecutionLimits checks if the order execution limits are within the defined limits from storage
func (b *Base) CheckOrderExecutionLimits(a asset.Item, cp currency.Pair, amount, price float64, orderType order.Type) error {
	return limits.CheckOrderExecutionLimits(
		key.NewExchangeAssetPair(b.Name, a, cp),
		amount,
		price,
		orderType,
	)
}

// WebsocketSubmitOrder submits an order to the exchange via a websocket connection
func (*Base) WebsocketSubmitOrder(context.Context, *order.Submit) (*order.SubmitResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WebsocketSubmitOrders submits multiple orders (batch) via the websocket connection
func (*Base) WebsocketSubmitOrders(context.Context, []*order.Submit) (responses []*order.SubmitResponse, err error) {
	return nil, common.ErrFunctionNotSupported
}

// WebsocketModifyOrder modifies an order via the websocket connection
func (*Base) WebsocketModifyOrder(context.Context, *order.Modify) (*order.ModifyResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WebsocketCancelOrder cancels an order via the websocket connection
func (*Base) WebsocketCancelOrder(context.Context, *order.Cancel) error {
	return common.ErrFunctionNotSupported
}

// MessageID returns a universally unique id using UUID V7
// In the future additional params may be added to method signature to provide context for the message id for overriding exchange implementations
func (b *Base) MessageID() string {
	return uuid.Must(uuid.NewV7()).String()
}

// MessageSequence returns a sequential message sequence number from common.Counter
// It is not universally unique but should be unique and sequential within each *Base instance
func (b *Base) MessageSequence() int64 {
	return b.messageSequence.IncrementAndGet()
}

// SubscribeAccountBalances returns a pipe to stream account holding updates
func (b *Base) SubscribeAccountBalances() (dispatch.Pipe, error) {
	return b.Accounts.Subscribe()
}
