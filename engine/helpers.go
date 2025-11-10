package engine

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/pquerna/otp/totp"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/file"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/dispatch"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/binance"
	"github.com/thrasher-corp/gocryptotrader/exchanges/binanceus"
	"github.com/thrasher-corp/gocryptotrader/exchanges/bitfinex"
	"github.com/thrasher-corp/gocryptotrader/exchanges/bitflyer"
	"github.com/thrasher-corp/gocryptotrader/exchanges/bithumb"
	"github.com/thrasher-corp/gocryptotrader/exchanges/bitmex"
	"github.com/thrasher-corp/gocryptotrader/exchanges/bitstamp"
	"github.com/thrasher-corp/gocryptotrader/exchanges/btcmarkets"
	"github.com/thrasher-corp/gocryptotrader/exchanges/btse"
	"github.com/thrasher-corp/gocryptotrader/exchanges/bybit"
	"github.com/thrasher-corp/gocryptotrader/exchanges/coinbase"
	"github.com/thrasher-corp/gocryptotrader/exchanges/coinut"
	"github.com/thrasher-corp/gocryptotrader/exchanges/deposit"
	"github.com/thrasher-corp/gocryptotrader/exchanges/deribit"
	"github.com/thrasher-corp/gocryptotrader/exchanges/exmo"
	"github.com/thrasher-corp/gocryptotrader/exchanges/gateio"
	"github.com/thrasher-corp/gocryptotrader/exchanges/gemini"
	"github.com/thrasher-corp/gocryptotrader/exchanges/hitbtc"
	"github.com/thrasher-corp/gocryptotrader/exchanges/huobi"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kraken"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kucoin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/lbank"
	"github.com/thrasher-corp/gocryptotrader/exchanges/okx"
	"github.com/thrasher-corp/gocryptotrader/exchanges/poloniex"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stats"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/yobit"
	"github.com/thrasher-corp/gocryptotrader/gctscript/vm"
	"github.com/thrasher-corp/gocryptotrader/log"
)

var (
	errCertExpired         = errors.New("gRPC TLS certificate has expired")
	errCertDataIsNil       = errors.New("gRPC TLS certificate PEM data is nil")
	errCertTypeInvalid     = errors.New("gRPC TLS certificate type is invalid")
	errSubsystemNotFound   = errors.New("subsystem not found")
	errGRPCManagementFault = errors.New("cannot manage GRPC subsystem via GRPC. Please manually change your config")
	errNilBot              = errors.New("received nil engine bot")
)

const (
	defaultPPROFListenAddress = "localhost:8085"
)

// GetSubsystemsStatus returns the status of various subsystems
func (bot *Engine) GetSubsystemsStatus() map[string]bool {
	return map[string]bool{
		CommunicationsManagerName:     bot.CommunicationsManager.IsRunning(),
		ConnectionManagerName:         bot.connectionManager.IsRunning(),
		OrderManagerName:              bot.OrderManager.IsRunning(),
		PortfolioManagerName:          bot.portfolioManager.IsRunning(),
		NTPManagerName:                bot.ntpManager.IsRunning(),
		DatabaseConnectionManagerName: bot.DatabaseManager.IsRunning(),
		SyncManagerName:               bot.Settings.EnableExchangeSyncManager,
		grpcName:                      bot.Settings.EnableGRPC,
		grpcProxyName:                 bot.Settings.EnableGRPCProxy,
		vm.Name:                       bot.gctScriptManager.IsRunning(),
		dispatch.Name:                 dispatch.IsRunning(),
		dataHistoryManagerName:        bot.dataHistoryManager.IsRunning(),
		CurrencyStateManagementName:   bot.currencyStateManager.IsRunning(),
	}
}

// RPCEndpoint stores an RPC endpoint status and addr
type RPCEndpoint struct {
	Started    bool
	ListenAddr string
}

// GetRPCEndpoints returns a list of RPC endpoints and their listen addrs
func (bot *Engine) GetRPCEndpoints() (map[string]RPCEndpoint, error) {
	if bot.Config == nil {
		return nil, errNilConfig
	}
	return map[string]RPCEndpoint{
		grpcName: {
			Started:    bot.Settings.EnableGRPC,
			ListenAddr: "grpc://" + bot.Config.RemoteControl.GRPC.ListenAddress,
		},
		grpcProxyName: {
			Started:    bot.Settings.EnableGRPCProxy,
			ListenAddr: "https://" + bot.Config.RemoteControl.GRPC.GRPCProxyListenAddress,
		},
	}, nil
}

// SetSubsystem enables or disables an engine subsystem
func (bot *Engine) SetSubsystem(subSystemName string, enable bool) error {
	if bot == nil {
		return errNilBot
	}

	if bot.Config == nil {
		return errNilConfig
	}

	var err error
	switch strings.ToLower(subSystemName) {
	case CommunicationsManagerName:
		if enable {
			if bot.CommunicationsManager == nil {
				communicationsConfig := bot.Config.GetCommunicationsConfig()
				bot.CommunicationsManager, err = SetupCommunicationManager(&communicationsConfig)
				if err != nil {
					return err
				}
			}
			return bot.CommunicationsManager.Start()
		}
		return bot.CommunicationsManager.Stop()
	case ConnectionManagerName:
		if enable {
			if bot.connectionManager == nil {
				bot.connectionManager, err = setupConnectionManager(&bot.Config.ConnectionMonitor)
				if err != nil {
					return err
				}
			}
			return bot.connectionManager.Start()
		}
		return bot.connectionManager.Stop()
	case OrderManagerName:
		if enable {
			if bot.OrderManager == nil {
				bot.OrderManager, err = SetupOrderManager(bot.ExchangeManager, bot.CommunicationsManager, &bot.ServicesWG, &bot.Config.OrderManager)
				if err != nil {
					return err
				}
			}
			return bot.OrderManager.Start()
		}
		return bot.OrderManager.Stop()
	case PortfolioManagerName:
		if enable {
			if bot.portfolioManager == nil {
				bot.portfolioManager, err = setupPortfolioManager(bot.ExchangeManager, bot.Settings.PortfolioManagerDelay, bot.Config.Portfolio)
				if err != nil {
					return err
				}
			}
			return bot.portfolioManager.Start(&bot.ServicesWG)
		}
		return bot.portfolioManager.Stop()
	case NTPManagerName:
		if enable {
			if bot.ntpManager == nil {
				bot.ntpManager, err = setupNTPManager(
					&bot.Config.NTPClient,
					*bot.Config.Logging.Enabled)
				if err != nil {
					return err
				}
			}
			return bot.ntpManager.Start()
		}
		return bot.ntpManager.Stop()
	case DatabaseConnectionManagerName:
		if enable {
			if bot.DatabaseManager == nil {
				bot.DatabaseManager, err = SetupDatabaseConnectionManager(&bot.Config.Database)
				if err != nil {
					return err
				}
			}
			return bot.DatabaseManager.Start(&bot.ServicesWG)
		}
		return bot.DatabaseManager.Stop()
	case SyncManagerName:
		if enable {
			if bot.currencyPairSyncer == nil {
				cfg := bot.Config.SyncManagerConfig
				cfg.SynchronizeTicker = bot.Settings.EnableTickerSyncing
				cfg.SynchronizeOrderbook = bot.Settings.EnableOrderbookSyncing
				cfg.SynchronizeContinuously = bot.Settings.SyncContinuously
				cfg.SynchronizeTrades = bot.Settings.EnableTradeSyncing
				cfg.Verbose = bot.Settings.Verbose || cfg.Verbose

				if cfg.TimeoutREST != bot.Settings.SyncTimeoutREST &&
					bot.Settings.SyncTimeoutREST != config.DefaultSyncerTimeoutREST {
					cfg.TimeoutREST = bot.Settings.SyncTimeoutREST
				}
				if cfg.TimeoutWebsocket != bot.Settings.SyncTimeoutWebsocket &&
					bot.Settings.SyncTimeoutWebsocket != config.DefaultSyncerTimeoutWebsocket {
					cfg.TimeoutWebsocket = bot.Settings.SyncTimeoutWebsocket
				}
				if cfg.NumWorkers != bot.Settings.SyncWorkersCount &&
					bot.Settings.SyncWorkersCount != config.DefaultSyncerWorkers {
					cfg.NumWorkers = bot.Settings.SyncWorkersCount
				}
				bot.currencyPairSyncer, err = SetupSyncManager(
					&cfg,
					bot.ExchangeManager,
					&bot.Config.RemoteControl,
					bot.Settings.EnableWebsocketRoutine)
				if err != nil {
					return err
				}
			}
			return bot.currencyPairSyncer.Start()
		}
		return bot.currencyPairSyncer.Stop()
	case dispatch.Name:
		if enable {
			return dispatch.Start(bot.Settings.DispatchMaxWorkerAmount, bot.Settings.DispatchJobsLimit)
		}
		return dispatch.Stop()
	case grpcName, grpcProxyName:
		return errGRPCManagementFault
	case dataHistoryManagerName:
		if enable {
			if bot.dataHistoryManager == nil {
				bot.dataHistoryManager, err = SetupDataHistoryManager(bot.ExchangeManager, bot.DatabaseManager, &bot.Config.DataHistoryManager)
				if err != nil {
					return err
				}
			}
			return bot.dataHistoryManager.Start()
		}
		return bot.dataHistoryManager.Stop()
	case vm.Name:
		if enable {
			if bot.gctScriptManager == nil {
				bot.gctScriptManager, err = vm.NewManager(&bot.Config.GCTScript)
				if err != nil {
					return err
				}
			}
			return bot.gctScriptManager.Start(&bot.ServicesWG)
		}
		return bot.gctScriptManager.Stop()
	case strings.ToLower(CurrencyStateManagementName):
		if enable {
			if bot.currencyStateManager == nil {
				bot.currencyStateManager, err = SetupCurrencyStateManager(
					bot.Config.CurrencyStateManager.Delay,
					bot.ExchangeManager)
				if err != nil {
					return err
				}
			}
			return bot.currencyStateManager.Start()
		}
		return bot.currencyStateManager.Stop()
	}
	return fmt.Errorf("%s: %w", subSystemName, errSubsystemNotFound)
}

// GetExchangeOTPs returns OTP codes for all exchanges which have a otpsecret
// stored
func (bot *Engine) GetExchangeOTPs() (map[string]string, error) {
	otpCodes := make(map[string]string)
	for x := range bot.Config.Exchanges {
		if otpSecret := bot.Config.Exchanges[x].API.Credentials.OTPSecret; otpSecret != "" {
			exchName := bot.Config.Exchanges[x].Name
			o, err := totp.GenerateCode(otpSecret, time.Now())
			if err != nil {
				log.Errorf(log.Global, "Unable to generate OTP code for exchange %s. Err: %s\n",
					exchName, err)
				continue
			}
			otpCodes[exchName] = o
		}
	}

	if len(otpCodes) == 0 {
		return nil, errors.New("no exchanges found which have a OTP secret stored")
	}

	return otpCodes, nil
}

// GetExchangeOTPByName returns a OTP code for the desired exchange
// if it exists
func (bot *Engine) GetExchangeOTPByName(exchName string) (string, error) {
	for x := range bot.Config.Exchanges {
		if !strings.EqualFold(bot.Config.Exchanges[x].Name, exchName) {
			continue
		}

		if otpSecret := bot.Config.Exchanges[x].API.Credentials.OTPSecret; otpSecret != "" {
			return totp.GenerateCode(otpSecret, time.Now())
		}
	}
	return "", errors.New("exchange does not have a OTP secret stored")
}

// GetAuthAPISupportedExchanges returns a list of auth api enabled exchanges
func (bot *Engine) GetAuthAPISupportedExchanges() []string {
	exchanges := bot.GetExchanges()
	exchangeNames := make([]string, 0, len(exchanges))
	for x := range exchanges {
		if !exchanges[x].IsRESTAuthenticationSupported() &&
			!exchanges[x].IsWebsocketAuthenticationSupported() {
			continue
		}
		exchangeNames = append(exchangeNames, exchanges[x].GetName())
	}
	return exchangeNames
}

// IsOnline returns whether or not the engine has Internet connectivity
func (bot *Engine) IsOnline() bool {
	return bot.connectionManager.IsOnline()
}

// GetAllAvailablePairs returns a list of all available pairs on either enabled
// or disabled exchanges
func (bot *Engine) GetAllAvailablePairs(enabledExchangesOnly bool, assetType asset.Item) currency.Pairs {
	var pairList currency.Pairs
	for x := range bot.Config.Exchanges {
		if enabledExchangesOnly && !bot.Config.Exchanges[x].Enabled {
			continue
		}

		exchName := bot.Config.Exchanges[x].Name
		pairs, err := bot.Config.GetAvailablePairs(exchName, assetType)
		if err != nil {
			continue
		}

		for y := range pairs {
			if pairList.Contains(pairs[y], false) {
				continue
			}
			pairList = append(pairList, pairs[y])
		}
	}
	return pairList
}

// GetSpecificAvailablePairs returns a list of supported pairs based on specific
// parameters
func (bot *Engine) GetSpecificAvailablePairs(enabledExchangesOnly, fiatPairs, includeUSDT, cryptoPairs bool, assetType asset.Item) currency.Pairs {
	var pairList currency.Pairs
	supportedPairs := bot.GetAllAvailablePairs(enabledExchangesOnly, assetType)

	for x := range supportedPairs {
		if fiatPairs {
			if supportedPairs[x].IsCryptoFiatPair() &&
				!supportedPairs[x].Contains(currency.USDT) ||
				(includeUSDT &&
					supportedPairs[x].Contains(currency.USDT) &&
					supportedPairs[x].IsCryptoPair()) {
				if pairList.Contains(supportedPairs[x], false) {
					continue
				}
				pairList = append(pairList, supportedPairs[x])
			}
		}
		if cryptoPairs {
			if supportedPairs[x].IsCryptoPair() {
				if pairList.Contains(supportedPairs[x], false) {
					continue
				}
				pairList = append(pairList, supportedPairs[x])
			}
		}
	}
	return pairList
}

// IsRelatablePairs checks to see if the two pairs are relatable
func IsRelatablePairs(p1, p2 currency.Pair, includeUSDT bool) bool {
	if p1.EqualIncludeReciprocal(p2) {
		return true
	}

	relatablePairs := GetRelatableCurrencies(p1, true, includeUSDT)
	if p1.IsCryptoFiatPair() {
		for x := range relatablePairs {
			relatablePairs = append(relatablePairs,
				GetRelatableFiatCurrencies(relatablePairs[x])...)
		}
	}
	return relatablePairs.Contains(p2, false)
}

// MapCurrenciesByExchange returns a list of currency pairs mapped to an
// exchange
func (bot *Engine) MapCurrenciesByExchange(p currency.Pairs, enabledExchangesOnly bool, assetType asset.Item) map[string]currency.Pairs {
	currencyExchange := make(map[string]currency.Pairs)
	for x := range p {
		for y := range bot.Config.Exchanges {
			if enabledExchangesOnly && !bot.Config.Exchanges[y].Enabled {
				continue
			}
			exchName := bot.Config.Exchanges[y].Name
			if !bot.Config.SupportsPair(exchName, p[x], assetType) {
				continue
			}

			result := currencyExchange[exchName]
			if result.Contains(p[x], false) {
				continue
			}
			result = append(result, p[x])
			currencyExchange[exchName] = result
		}
	}
	return currencyExchange
}

// GetExchangeNamesByCurrency returns a list of exchanges supporting
// a currency pair based on whether the exchange is enabled or not
func (bot *Engine) GetExchangeNamesByCurrency(p currency.Pair, enabled bool, assetType asset.Item) []string {
	exchanges := make([]string, 0, len(bot.Config.Exchanges))
	for x := range bot.Config.Exchanges {
		if enabled != bot.Config.Exchanges[x].Enabled {
			continue
		}

		exchName := bot.Config.Exchanges[x].Name
		if !bot.Config.SupportsPair(exchName, p, assetType) {
			continue
		}
		exchanges = append(exchanges, exchName)
	}
	return exchanges
}

// GetRelatableCryptocurrencies returns a list of currency pairs if it can find
// any relatable currencies (e.g ETHBTC -> ETHLTC -> ETHUSDT -> ETHREP)
func GetRelatableCryptocurrencies(p currency.Pair) currency.Pairs {
	var pairs currency.Pairs
	cryptocurrencies := currency.GetCryptocurrencies()
	for x := range cryptocurrencies {
		newPair := currency.NewPair(p.Base, cryptocurrencies[x])
		if newPair.IsInvalid() {
			continue
		}
		if newPair.Equal(p) {
			continue
		}
		if pairs.Contains(newPair, false) {
			continue
		}
		pairs = append(pairs, newPair)
	}
	return pairs
}

// GetRelatableFiatCurrencies returns a list of currency pairs if it can find
// any relatable currencies (e.g ETHUSD -> ETHAUD -> ETHGBP -> ETHJPY)
func GetRelatableFiatCurrencies(p currency.Pair) currency.Pairs {
	var pairs currency.Pairs
	fiatCurrencies := currency.GetFiatCurrencies()

	for x := range fiatCurrencies {
		newPair := currency.NewPair(p.Base, fiatCurrencies[x])
		if newPair.Base.Equal(newPair.Quote) {
			continue
		}

		if newPair.Equal(p) {
			continue
		}

		if pairs.Contains(newPair, false) {
			continue
		}
		pairs = append(pairs, newPair)
	}
	return pairs
}

// GetRelatableCurrencies returns a list of currency pairs if it can find
// any relatable currencies (e.g BTCUSD -> BTC USDT -> XBT USDT -> XBT USD)
// incOrig includes the supplied pair if desired
func GetRelatableCurrencies(p currency.Pair, incOrig, incUSDT bool) currency.Pairs {
	var pairs currency.Pairs

	addPair := func(p currency.Pair) {
		if pairs.Contains(p, true) {
			return
		}
		pairs = append(pairs, p)
	}

	buildPairs := func(p currency.Pair, incOrig bool) {
		if incOrig {
			addPair(p)
		}

		first := currency.GetTranslation(p.Base)
		if !first.Equal(p.Base) {
			addPair(currency.NewPair(first, p.Quote))

			second := currency.GetTranslation(p.Quote)
			if !second.Equal(p.Quote) {
				addPair(currency.NewPair(first, second))
			}
		}

		second := currency.GetTranslation(p.Quote)
		if !second.Equal(p.Quote) {
			addPair(currency.NewPair(p.Base, second))
		}
	}

	buildPairs(p, incOrig)
	buildPairs(p.Swap(), incOrig)

	if !incUSDT {
		pairs = pairs.RemovePairsByFilter(currency.USDT)
	}

	return pairs
}

// GetExchangeHighestPriceByCurrencyPair returns the exchange with the highest
// price for a given currency pair and asset type
func GetExchangeHighestPriceByCurrencyPair(p currency.Pair, a asset.Item) (string, error) {
	result := stats.SortExchangesByPrice(p, a, true)
	if len(result) == 0 {
		return "", errors.New("no stats for supplied currency pair and asset type")
	}

	return result[0].Exchange, nil
}

// GetExchangeLowestPriceByCurrencyPair returns the exchange with the lowest
// price for a given currency pair and asset type
func GetExchangeLowestPriceByCurrencyPair(p currency.Pair, assetType asset.Item) (string, error) {
	result := stats.SortExchangesByPrice(p, assetType, false)
	if len(result) == 0 {
		return "", errors.New("no stats for supplied currency pair and asset type")
	}

	return result[0].Exchange, nil
}

// GetCryptocurrenciesByExchange returns a list of cryptocurrencies the exchange supports
func (bot *Engine) GetCryptocurrenciesByExchange(exchangeName string, enabledExchangesOnly, enabledPairs bool, assetType asset.Item) ([]string, error) {
	var cryptocurrencies []string
	for x := range bot.Config.Exchanges {
		if !strings.EqualFold(bot.Config.Exchanges[x].Name, exchangeName) {
			continue
		}
		if enabledExchangesOnly && !bot.Config.Exchanges[x].Enabled {
			continue
		}

		var err error
		var pairs currency.Pairs
		if enabledPairs {
			pairs, err = bot.Config.GetEnabledPairs(exchangeName, assetType)
		} else {
			pairs, err = bot.Config.GetAvailablePairs(exchangeName, assetType)
		}
		if err != nil {
			return nil, err
		}
		cryptocurrencies = pairs.GetCrypto().Strings()
		break
	}
	return cryptocurrencies, nil
}

// GetCryptocurrencyDepositAddressesByExchange returns the cryptocurrency deposit addresses for a particular exchange
func (bot *Engine) GetCryptocurrencyDepositAddressesByExchange(exchName string) (map[string][]deposit.Address, error) {
	if bot.DepositAddressManager != nil {
		if bot.DepositAddressManager.IsSynced() {
			return bot.DepositAddressManager.GetDepositAddressesByExchange(exchName)
		}
		return nil, errors.New("deposit address manager has not yet synced all exchange deposit addresses")
	}

	result := bot.GetAllExchangeCryptocurrencyDepositAddresses()
	r, ok := result[exchName]
	if !ok {
		return nil, fmt.Errorf("%s %w", exchName, ErrExchangeNotFound)
	}
	return r, nil
}

// GetExchangeCryptocurrencyDepositAddress returns the cryptocurrency deposit address for a particular
// exchange
func (bot *Engine) GetExchangeCryptocurrencyDepositAddress(ctx context.Context, exchName, accountID, chain string, item currency.Code, bypassCache bool) (*deposit.Address, error) {
	if bot.DepositAddressManager != nil &&
		bot.DepositAddressManager.IsSynced() &&
		!bypassCache {
		resp, err := bot.DepositAddressManager.GetDepositAddressByExchangeAndCurrency(exchName, chain, item)
		return &resp, err
	}
	exch, err := bot.GetExchangeByName(exchName)
	if err != nil {
		return nil, err
	}
	return exch.GetDepositAddress(ctx, item, accountID, chain)
}

// GetAllExchangeCryptocurrencyDepositAddresses obtains an exchanges deposit cryptocurrency list
func (bot *Engine) GetAllExchangeCryptocurrencyDepositAddresses() map[string]ExchangeDepositAddresses {
	result := make(map[string]ExchangeDepositAddresses)
	exchanges := bot.GetExchanges()
	var depositSyncer sync.WaitGroup
	depositSyncer.Add(len(exchanges))
	var m sync.Mutex
	for x := range exchanges {
		go func(exch exchange.IBotExchange) {
			defer depositSyncer.Done()
			exchName := exch.GetName()
			if !exch.IsRESTAuthenticationSupported() {
				if bot.Settings.Verbose {
					log.Debugf(log.ExchangeSys, "GetAllExchangeCryptocurrencyDepositAddresses: Skippping %s due to disabled authenticated API support.\n", exchName)
				}
				return
			}

			cryptoCurrencies, err := bot.GetCryptocurrenciesByExchange(exchName, true, true, asset.Spot)
			if err != nil {
				log.Errorf(log.ExchangeSys, "%s failed to get cryptocurrency deposit addresses. Err: %s\n", exchName, err)
				return
			}
			supportsMultiChain := exch.GetBase().Features.Supports.RESTCapabilities.MultiChainDeposits
			requiresChainSet := exch.GetBase().Features.Supports.RESTCapabilities.MultiChainDepositRequiresChainSet
			cryptoAddr := make(map[string][]deposit.Address)
			for y := range cryptoCurrencies {
				cryptocurrency := cryptoCurrencies[y]
				isSingular := false
				var depositAddrs []deposit.Address
				if supportsMultiChain {
					availChains, err := exch.GetAvailableTransferChains(context.TODO(), currency.NewCode(cryptocurrency))
					if err != nil {
						log.Errorf(log.Global, "%s failed to get cryptocurrency available transfer chains. Err: %s\n", exchName, err)
						continue
					}
					if len(availChains) > 0 {
						// store the default non-chain specified address for a specified crypto
						chainContainsItself := common.StringSliceCompareInsensitive(availChains, cryptocurrency)
						if !chainContainsItself && !requiresChainSet {
							depositAddr, err := exch.GetDepositAddress(context.TODO(), currency.NewCode(cryptocurrency), "", "")
							if err != nil {
								log.Errorf(log.Global, "%s failed to get cryptocurrency deposit address for %s. Err: %s\n",
									exchName,
									cryptocurrency,
									err)
								continue
							}
							depositAddr.Chain = cryptocurrency
							depositAddrs = append(depositAddrs, *depositAddr)
						}
						for z := range availChains {
							if availChains[z] == "" {
								log.Warnf(log.Global, "%s %s available transfer chain is populated with an empty string\n",
									exchName,
									cryptocurrency)
								continue
							}

							depositAddr, err := exch.GetDepositAddress(context.TODO(), currency.NewCode(cryptocurrency), "", availChains[z])
							if err != nil {
								log.Errorf(log.Global, "%s failed to get cryptocurrency deposit address for %s [chain %s]. Err: %s\n",
									exchName,
									cryptocurrency,
									availChains[z],
									err)
								continue
							}
							depositAddr.Chain = availChains[z]
							depositAddrs = append(depositAddrs, *depositAddr)
						}
					} else {
						// cryptocurrency doesn't support multichain transfers
						isSingular = true
					}
				}

				if !supportsMultiChain || isSingular {
					depositAddr, err := exch.GetDepositAddress(context.TODO(), currency.NewCode(cryptocurrency), "", "")
					if err != nil {
						log.Errorf(log.Global, "%s failed to get cryptocurrency deposit address for %s. Err: %s\n",
							exchName,
							cryptocurrency,
							err)
						continue
					}
					depositAddrs = append(depositAddrs, *depositAddr)
				}
				cryptoAddr[cryptocurrency] = depositAddrs
			}
			m.Lock()
			result[exchName] = cryptoAddr
			m.Unlock()
		}(exchanges[x])
	}
	depositSyncer.Wait()
	if len(result) > 0 {
		log.Infoln(log.Global, "Deposit addresses synced")
	}
	return result
}

// GetExchangeNames returns a list of enabled or disabled exchanges
func (bot *Engine) GetExchangeNames(enabledOnly bool) []string {
	exchanges := bot.GetExchanges()
	var response []string
	for i := range exchanges {
		if !enabledOnly || (enabledOnly && exchanges[i].IsEnabled()) {
			response = append(response, exchanges[i].GetName())
		}
	}
	return response
}

// AllEnabledExchangeCurrencies holds the enabled exchange currencies
type AllEnabledExchangeCurrencies struct {
	Data []EnabledExchangeCurrencies `json:"data"`
}

// EnabledExchangeCurrencies is a sub type for singular exchanges and respective
// currencies
type EnabledExchangeCurrencies struct {
	ExchangeName   string          `json:"exchangeName"`
	ExchangeValues []*ticker.Price `json:"exchangeValues"`
}

// GetAllActiveTickers returns all enabled exchange tickers
func (bot *Engine) GetAllActiveTickers() []EnabledExchangeCurrencies {
	var tickerData []EnabledExchangeCurrencies
	exchanges := bot.GetExchanges()
	for x := range exchanges {
		assets := exchanges[x].GetAssetTypes(true)
		exchName := exchanges[x].GetName()
		var exchangeTicker EnabledExchangeCurrencies
		exchangeTicker.ExchangeName = exchName

		for y := range assets {
			currencies, err := exchanges[x].GetEnabledPairs(assets[y])
			if err != nil {
				log.Errorf(log.ExchangeSys, "Exchange %s could not retrieve enabled currencies. Err: %s\n", exchName, err)
				continue
			}
			for z := range currencies {
				tp, err := exchanges[x].GetCachedTicker(currencies[z], assets[y])
				if err != nil {
					log.Errorf(log.ExchangeSys, "Exchange %s failed to retrieve %s ticker. Err: %s\n", exchName, currencies[z].String(), err)
					continue
				}
				exchangeTicker.ExchangeValues = append(exchangeTicker.ExchangeValues, tp)
			}
			tickerData = append(tickerData, exchangeTicker)
		}
	}
	return tickerData
}

func verifyCert(pemData []byte) error {
	var pemBlock *pem.Block
	pemBlock, _ = pem.Decode(pemData)
	if pemBlock == nil {
		return errCertDataIsNil
	}

	if pemBlock.Type != "CERTIFICATE" {
		return errCertTypeInvalid
	}

	cert, err := x509.ParseCertificate(pemBlock.Bytes)
	if err != nil {
		return err
	}

	if time.Now().After(cert.NotAfter) {
		return errCertExpired
	}
	return nil
}

// CheckCerts checks and verifies RPC server certificates
func CheckCerts(certDir string) error {
	certFile := filepath.Join(certDir, "cert.pem")
	keyFile := filepath.Join(certDir, "key.pem")

	if !file.Exists(certFile) || !file.Exists(keyFile) {
		log.Warnln(log.Global, "gRPC certificate/key file missing, recreating...")
		return genCert(certDir)
	}

	pemData, err := os.ReadFile(certFile)
	if err != nil {
		return fmt.Errorf("unable to open TLS cert file: %s", err)
	}

	if err = verifyCert(pemData); err != nil {
		if err != errCertExpired {
			return err
		}
		log.Warnln(log.Global, "gRPC certificate has expired, regenerating...")
		return genCert(certDir)
	}

	log.Infoln(log.Global, "gRPC TLS certificate and key files exist, will use them.")
	return nil
}

func genCert(targetDir string) error {
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to generate ecdsa private key: %s", err)
	}

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return fmt.Errorf("failed to generate serial number: %s", err)
	}

	host, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("failed to get hostname: %s", err)
	}

	dnsNames := []string{host}
	if host != "localhost" {
		dnsNames = append(dnsNames, "localhost")
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"gocryptotrader"},
			CommonName:   host,
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour * 24 * 365),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses: []net.IP{
			net.ParseIP("127.0.0.1"),
			net.ParseIP("::1"),
		},
		DNSNames: dnsNames,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privKey.PublicKey, privKey)
	if err != nil {
		return fmt.Errorf("failed to create certificate: %s", err)
	}

	certData := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	if certData == nil {
		return errors.New("cert data is nil")
	}

	b, err := x509.MarshalECPrivateKey(privKey)
	if err != nil {
		return fmt.Errorf("failed to marshal ECDSA private key: %s", err)
	}

	keyData := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: b})
	if keyData == nil {
		return errors.New("key pem data is nil")
	}

	err = file.Write(filepath.Join(targetDir, "key.pem"), keyData)
	if err != nil {
		return fmt.Errorf("failed to write key.pem file %s", err)
	}

	err = file.Write(filepath.Join(targetDir, "cert.pem"), certData)
	if err != nil {
		return fmt.Errorf("failed to write cert.pem file %s", err)
	}

	log.Infof(log.Global, "gRPC TLS key.pem and cert.pem files written to %s\n", targetDir)
	return nil
}

// NewSupportedExchangeByName helps create a new exchange to be loaded that is
// supported by GCT. This function will return an error if the exchange is not
// supported.
func NewSupportedExchangeByName(name string) (exchange.IBotExchange, error) {
	switch strings.ToLower(name) {
	case "binanceus":
		return new(binanceus.Exchange), nil
	case "binance":
		return new(binance.Exchange), nil
	case "bitfinex":
		return new(bitfinex.Exchange), nil
	case "bitflyer":
		return new(bitflyer.Exchange), nil
	case "bithumb":
		return new(bithumb.Exchange), nil
	case "bitmex":
		return new(bitmex.Exchange), nil
	case "bitstamp":
		return new(bitstamp.Exchange), nil
	case "btc markets":
		return new(btcmarkets.Exchange), nil
	case "btse":
		return new(btse.Exchange), nil
	case "bybit":
		return new(bybit.Exchange), nil
	case "coinut":
		return new(coinut.Exchange), nil
	case "deribit":
		return new(deribit.Exchange), nil
	case "exmo":
		return new(exmo.Exchange), nil
	case "coinbase":
		return new(coinbase.Exchange), nil
	case "gateio":
		return new(gateio.Exchange), nil
	case "gemini":
		return new(gemini.Exchange), nil
	case "hitbtc":
		return new(hitbtc.Exchange), nil
	case "huobi":
		return new(huobi.Exchange), nil
	case "kraken":
		return new(kraken.Exchange), nil
	case "kucoin":
		return new(kucoin.Exchange), nil
	case "lbank":
		return new(lbank.Exchange), nil
	case "okx":
		return new(okx.Exchange), nil
	case "poloniex":
		return new(poloniex.Exchange), nil
	case "yobit":
		return new(yobit.Exchange), nil
	default:
		return nil, fmt.Errorf("%q, %w", name, ErrExchangeNotFound)
	}
}

// NewExchangeByNameWithDefaults returns a defaulted exchange by its name if it
// exists. This will allocate a new exchange and setup the default config for it.
// This will automatically fetch available pairs.
func NewExchangeByNameWithDefaults(ctx context.Context, name string) (exchange.IBotExchange, error) {
	exch, err := NewSupportedExchangeByName(name)
	if err != nil {
		return nil, err
	}
	defaultConfig, err := exchange.GetDefaultConfig(ctx, exch)
	if err != nil {
		return nil, err
	}
	err = exch.Setup(defaultConfig)
	if err != nil {
		return nil, err
	}
	return exch, nil
}

// StartPPROF starts a pprof profiler if enabled
func StartPPROF(ctx context.Context, cfg *config.Profiler) error {
	if !cfg.Enabled {
		return nil
	}

	runtime.SetMutexProfileFraction(cfg.MutexProfileFraction)
	runtime.SetBlockProfileRate(cfg.BlockProfileRate)

	listenAddr := cfg.ListenAddress
	if listenAddr == "" {
		listenAddr = defaultPPROFListenAddress
	}

	lc := net.ListenConfig{}
	ln, err := lc.Listen(ctx, "tcp", listenAddr)
	if err != nil {
		return fmt.Errorf("pprof listen error: %w", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	srv := &http.Server{
		Addr:         listenAddr,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		Handler:      mux,
	}

	log.Infof(log.Global, "PPROF profiler listening on http://%s/debug/pprof/", listenAddr)

	go func() {
		if err := srv.Serve(ln); err != nil {
			log.Errorf(log.Global, "PPROF serve error: %s", err)
		}
	}()

	return nil
}
