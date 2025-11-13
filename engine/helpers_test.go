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
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/common/file"
	"github.com/thrasher-corp/gocryptotrader/communications"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/dispatch"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/deposit"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stats"
	"github.com/thrasher-corp/gocryptotrader/gctscript/vm"
	"github.com/thrasher-corp/gocryptotrader/log"
)

var testExchange = "Bitstamp"

func CreateTestBot(tb testing.TB) *Engine {
	tb.Helper()
	cFormat := &currency.PairFormat{Uppercase: true}
	cp1 := currency.NewBTCUSD()
	cp2 := currency.NewBTCUSDT()

	pairs1 := map[asset.Item]*currency.PairStore{
		asset.Spot: {
			AssetEnabled: true,
			Available:    currency.Pairs{cp1},
			Enabled:      currency.Pairs{cp1},
		},
	}
	pairs2 := map[asset.Item]*currency.PairStore{
		asset.Spot: {
			AssetEnabled: true,
			Available:    currency.Pairs{cp2},
			Enabled:      currency.Pairs{cp2},
		},
	}
	bot := &Engine{
		ExchangeManager: NewExchangeManager(),
		Config: &config.Config{Exchanges: []config.Exchange{
			{
				Name:                    testExchange,
				Enabled:                 true,
				WebsocketTrafficTimeout: time.Second,
				API: config.APIConfig{
					Credentials: config.APICredentialsConfig{},
				},
				CurrencyPairs: &currency.PairsManager{
					RequestFormat:   cFormat,
					ConfigFormat:    cFormat,
					UseGlobalFormat: true,
					Pairs:           pairs1,
				},
			},
			{
				Name:                    "binance",
				Enabled:                 true,
				WebsocketTrafficTimeout: time.Second,
				API: config.APIConfig{
					Credentials: config.APICredentialsConfig{},
				},
				CurrencyPairs: &currency.PairsManager{
					RequestFormat:   cFormat,
					ConfigFormat:    cFormat,
					UseGlobalFormat: true,
					Pairs:           pairs2,
				},
			},
		}},
	}
	err := bot.LoadExchange(testExchange)
	assert.NoError(tb, err, "LoadExchange should not error")

	return bot
}

func TestGetSubsystemsStatus(t *testing.T) {
	assert.Len(t, (&Engine{}).GetSubsystemsStatus(), 13, "GetSubsystemStatus should return the correct number of subsystems")
}

func TestGetRPCEndpoints(t *testing.T) {
	_, err := (&Engine{}).GetRPCEndpoints()
	require.ErrorIs(t, err, errNilConfig)

	m, err := (&Engine{Config: &config.Config{}}).GetRPCEndpoints()
	require.NoError(t, err)
	assert.Len(t, m, 2, "GetRPCEndpoints should return the correct number of RPC endpoints")
}

func TestSetSubsystem(t *testing.T) { //nolint // TO-DO: Fix race t.Parallel() usage
	testCases := []struct {
		Subsystem    string
		Engine       *Engine
		EnableError  error
		DisableError error
	}{
		{Subsystem: "sillyBilly", EnableError: errNilBot, DisableError: errNilBot},
		{Subsystem: "sillyBilly", Engine: &Engine{}, EnableError: errNilConfig, DisableError: errNilConfig},
		{Subsystem: "sillyBilly", Engine: &Engine{Config: &config.Config{}}, EnableError: errSubsystemNotFound, DisableError: errSubsystemNotFound},
		{
			Subsystem:    CommunicationsManagerName,
			Engine:       &Engine{Config: &config.Config{}},
			EnableError:  communications.ErrNoRelayersEnabled,
			DisableError: ErrNilSubsystem,
		},
		{
			Subsystem:    ConnectionManagerName,
			Engine:       &Engine{Config: &config.Config{}},
			EnableError:  nil,
			DisableError: nil,
		},
		{
			Subsystem:    OrderManagerName,
			Engine:       &Engine{Config: &config.Config{}},
			EnableError:  nil,
			DisableError: nil,
		},
		{
			Subsystem:    PortfolioManagerName,
			Engine:       &Engine{Config: &config.Config{}},
			EnableError:  errNilExchangeManager,
			DisableError: ErrNilSubsystem,
		},
		{
			Subsystem:    NTPManagerName,
			Engine:       &Engine{Config: &config.Config{Logging: log.Config{Enabled: convert.BoolPtr(false)}}},
			EnableError:  errNilNTPConfigValues,
			DisableError: ErrNilSubsystem,
		},
		{
			Subsystem:    DatabaseConnectionManagerName,
			Engine:       &Engine{Config: &config.Config{}},
			EnableError:  database.ErrDatabaseSupportDisabled,
			DisableError: ErrSubSystemNotStarted,
		},
		{
			Subsystem:    SyncManagerName,
			Engine:       &Engine{Config: &config.Config{}},
			EnableError:  errNoSyncItemsEnabled,
			DisableError: ErrNilSubsystem,
		},
		{
			Subsystem:    dispatch.Name,
			Engine:       &Engine{Config: &config.Config{}},
			EnableError:  nil,
			DisableError: nil,
		},
		{
			Subsystem:    grpcName,
			Engine:       &Engine{Config: &config.Config{}},
			EnableError:  errGRPCManagementFault,
			DisableError: errGRPCManagementFault,
		},
		{
			Subsystem:    grpcProxyName,
			Engine:       &Engine{Config: &config.Config{}},
			EnableError:  errGRPCManagementFault,
			DisableError: errGRPCManagementFault,
		},
		{
			Subsystem:    dataHistoryManagerName,
			Engine:       &Engine{Config: &config.Config{}},
			EnableError:  database.ErrNilInstance,
			DisableError: ErrNilSubsystem,
		},
		{
			Subsystem:    vm.Name,
			Engine:       &Engine{Config: &config.Config{}},
			EnableError:  nil,
			DisableError: nil,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.Subsystem, func(t *testing.T) {
			t.Parallel()
			err := tt.Engine.SetSubsystem(tt.Subsystem, true)
			require.ErrorIs(t, err, tt.EnableError)

			err = tt.Engine.SetSubsystem(tt.Subsystem, false)
			require.ErrorIs(t, err, tt.DisableError)
		})
	}
}

func TestGetExchangeOTPs(t *testing.T) {
	t.Parallel()
	bot := CreateTestBot(t)
	_, err := bot.GetExchangeOTPs()
	if err == nil {
		t.Fatal("Expected err with no exchange OTP secrets set")
	}

	bnCfg, err := bot.Config.GetExchangeConfig("binance")
	if err != nil {
		t.Fatal(err)
	}
	bCfg, err := bot.Config.GetExchangeConfig(testExchange)
	if err != nil {
		t.Fatal(err)
	}

	bnCfg.API.Credentials.OTPSecret = "JBSWY3DPEHPK3PXP"
	bCfg.API.Credentials.OTPSecret = "JBSWY3DPEHPK3PXP"
	result, err := bot.GetExchangeOTPs()
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 2 {
		t.Fatal("Expected 2 OTP results")
	}

	bnCfg.API.Credentials.OTPSecret = "°"
	result, err = bot.GetExchangeOTPs()
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 {
		t.Fatal("Expected 1 OTP code with invalid OTP Secret")
	}

	// Flush settings
	bnCfg.API.Credentials.OTPSecret = ""
	bCfg.API.Credentials.OTPSecret = ""
}

func TestGetExchangeoOTPByName(t *testing.T) {
	t.Parallel()
	bot := CreateTestBot(t)
	_, err := bot.GetExchangeOTPByName(testExchange)
	if err == nil {
		t.Fatal("Expected err with no exchange OTP secrets set")
	}

	bCfg, err := bot.Config.GetExchangeConfig(testExchange)
	if err != nil {
		t.Fatal(err)
	}

	bCfg.API.Credentials.OTPSecret = "JBSWY3DPEHPK3PXP"
	result, err := bot.GetExchangeOTPByName(testExchange)
	if err != nil {
		t.Fatal(err)
	}
	if result == "" {
		t.Fatal("Expected valid OTP code")
	}

	// Flush setting
	bCfg.API.Credentials.OTPSecret = ""
}

func TestGetAuthAPISupportedExchanges(t *testing.T) {
	t.Parallel()
	e := CreateTestBot(t)
	if result := e.GetAuthAPISupportedExchanges(); len(result) != 0 {
		t.Fatal("Unexpected result", result)
	}

	exch, err := e.ExchangeManager.GetExchangeByName(testExchange)
	if err != nil {
		t.Fatal(err)
	}

	b := exch.GetBase()
	b.API.AuthenticatedWebsocketSupport = true
	b.SetCredentials("test", "test", "", "", "", "")
	if result := e.GetAuthAPISupportedExchanges(); len(result) != 1 {
		t.Fatal("Unexpected result", result)
	}
}

func TestIsOnline(t *testing.T) {
	t.Parallel()
	e := CreateTestBot(t)
	var err error
	e.connectionManager, err = setupConnectionManager(&e.Config.ConnectionMonitor)
	if err != nil {
		t.Fatal(err)
	}
	if r := e.IsOnline(); r {
		t.Fatal("Unexpected result")
	}

	if err = e.connectionManager.Start(); err != nil {
		t.Fatal(err)
	}

	tick := time.NewTicker(time.Second * 5)
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			t.Fatal("Test timeout")
		default:
			if e.IsOnline() {
				if err := e.connectionManager.Stop(); err != nil {
					t.Fatal("unable to shutdown connection manager")
				}
				return
			}
		}
	}
}

func TestGetSpecificAvailablePairs(t *testing.T) {
	t.Parallel()
	e := CreateTestBot(t)
	c := currency.Code{
		Item: &currency.Item{
			Role:   currency.Cryptocurrency,
			Symbol: "usdt",
		},
	}
	e.Config = &config.Config{
		Exchanges: []config.Exchange{
			{
				Enabled: true,
				Name:    testExchange,
				CurrencyPairs: &currency.PairsManager{Pairs: map[asset.Item]*currency.PairStore{
					asset.Spot: {
						AssetEnabled: true,
						Enabled:      currency.Pairs{currency.NewBTCUSD(), currency.NewPair(currency.BTC, c)},
						Available:    currency.Pairs{currency.NewBTCUSD(), currency.NewPair(currency.BTC, c)},
						ConfigFormat: &currency.PairFormat{
							Uppercase: true,
						},
					},
				}},
			},
		},
	}
	assetType := asset.Spot

	result := e.GetSpecificAvailablePairs(true, true, true, true, assetType)
	btcUSD := currency.NewBTCUSD()
	if !result.Contains(btcUSD, true) {
		t.Error("Unexpected result")
	}

	btcUSDT := currency.NewPair(currency.BTC, c)
	if !result.Contains(btcUSDT, false) {
		t.Error("Unexpected result")
	}

	result = e.GetSpecificAvailablePairs(true, true, false, false, assetType)

	if result.Contains(btcUSDT, false) {
		t.Error("Unexpected result")
	}

	ltcBTC := currency.NewPair(currency.LTC, currency.BTC)
	result = e.GetSpecificAvailablePairs(true, false, false, true, assetType)
	if result.Contains(ltcBTC, false) {
		t.Error("Unexpected result")
	}
}

func TestIsRelatablePairs(t *testing.T) {
	t.Parallel()
	CreateTestBot(t)

	btcusd := currency.NewBTCUSD()
	xbtusd := currency.NewPair(currency.XBT, currency.USD)
	xbtusdt := currency.NewPair(currency.XBT, currency.USDT)

	// Test relational pairs with similar names
	result := IsRelatablePairs(xbtusd, btcusd, false)
	if !result {
		t.Fatal("Unexpected result")
	}

	// Test relational pairs with similar names reversed
	result = IsRelatablePairs(btcusd, xbtusd, false)
	if !result {
		t.Fatal("Unexpected result")
	}

	// Test relational pairs with similar names but with Tether support disabled
	result = IsRelatablePairs(xbtusd, currency.NewBTCUSDT(), false)
	if result {
		t.Fatal("Unexpected result")
	}

	// Test relational pairs with similar names but with Tether support enabled
	result = IsRelatablePairs(xbtusdt, btcusd, true)
	if !result {
		t.Fatal("Unexpected result")
	}

	aeusdt, err := currency.NewPairFromStrings("AE", "USDT")
	if err != nil {
		t.Fatal(err)
	}

	usdtae, err := currency.NewPairDelimiter("USDT-AE", "-")
	if err != nil {
		t.Fatal(err)
	}

	// Test relational pairs with different ordering, a delimiter and with
	// Tether support enabled
	result = IsRelatablePairs(aeusdt, usdtae, true)
	if !result {
		t.Fatal("Unexpected result")
	}

	// Test relational pairs with different ordering, a delimiter and with
	// Tether support disabled
	result = IsRelatablePairs(aeusdt, usdtae, false)
	if !result {
		t.Fatal("Unexpected result")
	}

	xbteur, err := currency.NewPairFromStrings("XBT", "EUR")
	if err != nil {
		t.Fatal(err)
	}

	btcaud, err := currency.NewPairFromStrings("BTC", "AUD")
	if err != nil {
		t.Fatal(err)
	}

	// Test relationl pairs with similar names and different fiat currencies
	result = IsRelatablePairs(xbteur, btcaud, false)
	if !result {
		t.Fatal("Unexpected result")
	}

	usdbtc, err := currency.NewPairFromStrings("USD", "BTC")
	if err != nil {
		t.Fatal(err)
	}

	btceur, err := currency.NewPairFromStrings("BTC", "EUR")
	if err != nil {
		t.Fatal(err)
	}

	// Test relationl pairs with similar names, different fiat currencies and
	// with different ordering
	result = IsRelatablePairs(usdbtc, btceur, false)
	if !result { // Is this really expected result???
		t.Fatal("Unexpected result")
	}

	// Test relationl pairs with similar names, different fiat currencies and
	// with Tether enabled
	result = IsRelatablePairs(usdbtc, currency.NewBTCUSDT(), true)
	if !result {
		t.Fatal("Unexpected result")
	}

	ltcbtc, err := currency.NewPairFromStrings("LTC", "BTC")
	if err != nil {
		t.Fatal(err)
	}

	btcltc, err := currency.NewPairFromStrings("BTC", "LTC")
	if err != nil {
		t.Fatal(err)
	}

	// Test relationl crypto pairs with similar names
	result = IsRelatablePairs(ltcbtc, btcltc, false)
	if !result {
		t.Fatal("Unexpected result")
	}

	ltceth, err := currency.NewPairFromStrings("LTC", "ETH")
	if err != nil {
		t.Fatal(err)
	}

	btceth, err := currency.NewPairFromStrings("BTC", "ETH")
	if err != nil {
		t.Fatal(err)
	}

	// Test relationl crypto pairs with similar different pairs
	result = IsRelatablePairs(ltceth, btceth, false)
	if result {
		t.Fatal("Unexpected result")
	}

	// Test relationl crypto pairs with similar different pairs and with USDT
	// enabled
	usdtusd, err := currency.NewPairFromStrings("USDT", "USD")
	if err != nil {
		t.Fatal(err)
	}

	result = IsRelatablePairs(usdtusd, btcusd, true)
	if result {
		t.Fatal("Unexpected result")
	}

	xbtltc, err := currency.NewPairFromStrings("XBT", "LTC")
	if err != nil {
		t.Fatal(err)
	}

	// Test relationl crypto pairs with similar names
	result = IsRelatablePairs(xbtltc, btcltc, false)
	if !result {
		t.Fatal("Unexpected result")
	}

	ltcxbt, err := currency.NewPairFromStrings("LTC", "XBT")
	if err != nil {
		t.Fatal(err)
	}

	// Test relationl crypto pairs with different ordering and similar names
	result = IsRelatablePairs(ltcxbt, btcltc, false)
	if !result {
		t.Fatal("Unexpected result")
	}

	// Test edge case between two pairs when currency translations were causing
	// non-relational pairs to be relatable
	eurusd, err := currency.NewPairFromStrings("EUR", "USD")
	if err != nil {
		t.Fatal(err)
	}

	result = IsRelatablePairs(eurusd, btcusd, false)
	if result {
		t.Fatal("Unexpected result")
	}
}

func TestGetRelatableCryptocurrencies(t *testing.T) {
	t.Parallel()
	CreateTestBot(t)
	btcltc, err := currency.NewPairFromStrings("BTC", "LTC")
	if err != nil {
		t.Fatal(err)
	}

	btcbtc, err := currency.NewPairFromStrings("BTC", "BTC")
	if err != nil {
		t.Fatal(err)
	}

	ltcltc, err := currency.NewPairFromStrings("LTC", "LTC")
	if err != nil {
		t.Fatal(err)
	}

	btceth, err := currency.NewPairFromStrings("BTC", "ETH")
	if err != nil {
		t.Fatal(err)
	}

	p := GetRelatableCryptocurrencies(btcltc)
	if p.Contains(btcltc, true) {
		t.Error("Unexpected result")
	}
	if p.Contains(btcbtc, true) {
		t.Error("Unexpected result")
	}
	if p.Contains(ltcltc, true) {
		t.Error("Unexpected result")
	}
	if !p.Contains(btceth, true) {
		t.Error("Unexpected result")
	}

	p = GetRelatableCryptocurrencies(btcltc)
	if p.Contains(btcltc, true) {
		t.Error("Unexpected result")
	}
	if p.Contains(btcbtc, true) {
		t.Error("Unexpected result")
	}
	if p.Contains(ltcltc, true) {
		t.Error("Unexpected result")
	}
	if !p.Contains(btceth, true) {
		t.Error("Unexpected result")
	}
}

func TestGetRelatableFiatCurrencies(t *testing.T) {
	t.Parallel()
	btcUSD, err := currency.NewPairFromStrings("BTC", "USD")
	if err != nil {
		t.Fatal(err)
	}

	btcEUR, err := currency.NewPairFromStrings("BTC", "EUR")
	if err != nil {
		t.Fatal(err)
	}

	p := GetRelatableFiatCurrencies(btcUSD)
	if !p.Contains(btcEUR, true) {
		t.Error("Unexpected result")
	}

	if p.Contains(currency.NewPair(currency.DOGE, currency.XRP), true) {
		t.Error("Unexpected result")
	}
}

func TestMapCurrenciesByExchange(t *testing.T) {
	t.Parallel()
	e := CreateTestBot(t)

	pairs := []currency.Pair{
		currency.NewBTCUSD(),
		currency.NewPair(currency.BTC, currency.EUR),
	}

	result := e.MapCurrenciesByExchange(pairs, true, asset.Spot)
	pairs, ok := result[testExchange]
	if !ok {
		t.Fatal("Unexpected result")
	}

	if len(pairs) != 2 {
		t.Fatal("Unexpected result")
	}
}

func TestGetExchangeNamesByCurrency(t *testing.T) {
	t.Parallel()
	btsusd, err := currency.NewPairFromStrings("BTC", "USD")
	if err != nil {
		t.Fatal(err)
	}

	btcjpy, err := currency.NewPairFromStrings("BTC", "JPY")
	if err != nil {
		t.Fatal(err)
	}

	blahjpy, err := currency.NewPairFromStrings("blah", "JPY")
	if err != nil {
		t.Fatal(err)
	}

	e := CreateTestBot(t)
	bf := "Bitflyer"
	e.Config.Exchanges = append(e.Config.Exchanges, config.Exchange{
		Enabled: true,
		Name:    bf,
		CurrencyPairs: &currency.PairsManager{Pairs: map[asset.Item]*currency.PairStore{
			asset.Spot: {
				AssetEnabled: true,
				Enabled:      currency.Pairs{btcjpy},
				Available:    currency.Pairs{btcjpy},
				ConfigFormat: &currency.PairFormat{
					Uppercase: true,
				},
			},
		}},
	})
	assetType := asset.Spot

	result := e.GetExchangeNamesByCurrency(btsusd,
		true,
		assetType)
	if !slices.Contains(result, testExchange) {
		t.Fatal("Unexpected result")
	}

	result = e.GetExchangeNamesByCurrency(btcjpy,
		true,
		assetType)
	if !slices.Contains(result, bf) {
		t.Fatal("Unexpected result")
	}

	result = e.GetExchangeNamesByCurrency(blahjpy,
		true,
		assetType)
	if len(result) > 0 {
		t.Fatal("Unexpected result")
	}
}

func TestGetExchangeHighestPriceByCurrencyPair(t *testing.T) {
	t.Parallel()
	CreateTestBot(t)

	p, err := currency.NewPairFromStrings("BTC", "USD")
	if err != nil {
		t.Fatal(err)
	}

	err = stats.Add("Bitfinex", p, asset.Spot, 1000, 10000)
	if err != nil {
		t.Error(err)
	}
	err = stats.Add(testExchange, p, asset.Spot, 1337, 10000)
	if err != nil {
		t.Error(err)
	}
	exchangeName, err := GetExchangeHighestPriceByCurrencyPair(p, asset.Spot)
	if err != nil {
		t.Error(err)
	}

	if exchangeName != testExchange {
		t.Error("Unexpected result")
	}

	btcaud, err := currency.NewPairFromStrings("BTC", "AUD")
	if err != nil {
		t.Fatal(err)
	}

	_, err = GetExchangeHighestPriceByCurrencyPair(btcaud, asset.Spot)
	if err == nil {
		t.Error("Unexpected result")
	}
}

func TestGetExchangeLowestPriceByCurrencyPair(t *testing.T) {
	t.Parallel()
	CreateTestBot(t)

	p, err := currency.NewPairFromStrings("BTC", "USD")
	if err != nil {
		t.Fatal(err)
	}

	err = stats.Add("Bitfinex", p, asset.Spot, 1000, 10000)
	if err != nil {
		t.Error(err)
	}
	err = stats.Add(testExchange, p, asset.Spot, 1337, 10000)
	if err != nil {
		t.Error(err)
	}
	exchangeName, err := GetExchangeLowestPriceByCurrencyPair(p, asset.Spot)
	if err != nil {
		t.Error(err)
	}

	if exchangeName != "Bitfinex" {
		t.Error("Unexpected result")
	}

	btcaud, err := currency.NewPairFromStrings("BTC", "AUD")
	if err != nil {
		t.Fatal(err)
	}

	_, err = GetExchangeLowestPriceByCurrencyPair(btcaud, asset.Spot)
	if err == nil {
		t.Error("Unexpected result")
	}
}

func TestGetCryptocurrenciesByExchange(t *testing.T) {
	t.Parallel()
	e := CreateTestBot(t)
	_, err := e.GetCryptocurrenciesByExchange("Bitfinex", false, false, asset.Spot)
	if err != nil {
		t.Fatalf("Err %s", err)
	}
}

type fakeDepositExchangeOpts struct {
	SupportsAuth             bool
	SupportsMultiChain       bool
	RequiresChainSet         bool
	ReturnMultipleChains     bool
	ThrowPairError           bool
	ThrowTransferChainError  bool
	ThrowDepositAddressError bool
}

type fakeDepositExchange struct {
	exchange.IBotExchange
	*fakeDepositExchangeOpts
}

func (f fakeDepositExchange) GetName() string {
	return "fake"
}

func (f fakeDepositExchange) GetBase() *exchange.Base {
	return &exchange.Base{
		Features: exchange.Features{Supports: exchange.FeaturesSupported{
			RESTCapabilities: protocol.Features{
				MultiChainDeposits:                f.SupportsMultiChain,
				MultiChainDepositRequiresChainSet: f.RequiresChainSet,
			},
		}},
	}
}

func (f fakeDepositExchange) IsRESTAuthenticationSupported() bool {
	return f.SupportsAuth
}

func (f fakeDepositExchange) GetAvailableTransferChains(_ context.Context, c currency.Code) ([]string, error) {
	if f.ThrowTransferChainError {
		return nil, errors.New("unable to get available transfer chains")
	}
	if c.Equal(currency.XRP) {
		return nil, nil
	}
	if c.Equal(currency.USDT) {
		return []string{"sol", "btc", "usdt", ""}, nil
	}
	return []string{"BITCOIN"}, nil
}

func (f fakeDepositExchange) GetDepositAddress(_ context.Context, _ currency.Code, _, _ string) (*deposit.Address, error) {
	if f.ThrowDepositAddressError {
		return nil, errors.New("unable to get deposit address")
	}
	return &deposit.Address{Address: "fakeaddr"}, nil
}

func createDepositEngine(opts *fakeDepositExchangeOpts) *Engine {
	ps := currency.PairStore{
		AssetEnabled: true,
		Enabled: currency.Pairs{
			currency.NewBTCUSDT(),
			currency.NewPair(currency.XRP, currency.USDT),
		},
		Available: currency.Pairs{
			currency.NewBTCUSDT(),
			currency.NewPair(currency.XRP, currency.USDT),
		},
	}
	if opts.ThrowPairError {
		ps.Available = nil
	}
	return &Engine{
		Settings: Settings{CoreSettings: CoreSettings{Verbose: true}},
		Config: &config.Config{
			Exchanges: []config.Exchange{
				{
					Name:    "fake",
					Enabled: true,
					CurrencyPairs: &currency.PairsManager{
						UseGlobalFormat: true,
						ConfigFormat:    &currency.EMPTYFORMAT,
						Pairs: map[asset.Item]*currency.PairStore{
							asset.Spot: &ps,
						},
					},
				},
			},
		},
		ExchangeManager: &ExchangeManager{
			exchanges: map[string]exchange.IBotExchange{
				"fake": fakeDepositExchange{
					fakeDepositExchangeOpts: opts,
				},
			},
		},
	}
}

func TestGetCryptocurrencyDepositAddressesByExchange(t *testing.T) {
	t.Parallel()
	const exchName = "fake"
	e := createDepositEngine(&fakeDepositExchangeOpts{SupportsAuth: true, SupportsMultiChain: true})
	_, err := e.GetCryptocurrencyDepositAddressesByExchange(exchName)
	assert.NoError(t, err, "GetCryptocurrencyDepositAddressesByExchange should not error")
	_, err = e.GetCryptocurrencyDepositAddressesByExchange("non-existent")
	assert.ErrorIs(t, err, ErrExchangeNotFound)

	e.DepositAddressManager = SetupDepositAddressManager()
	_, err = e.GetCryptocurrencyDepositAddressesByExchange(exchName)
	if err == nil {
		t.Error("expected error")
	}
	if err = e.DepositAddressManager.Sync(e.GetAllExchangeCryptocurrencyDepositAddresses()); err != nil {
		t.Fatal(err)
	}
	_, err = e.GetCryptocurrencyDepositAddressesByExchange(exchName)
	if err != nil {
		t.Error(err)
	}
}

func TestGetExchangeCryptocurrencyDepositAddress(t *testing.T) {
	t.Parallel()
	e := createDepositEngine(&fakeDepositExchangeOpts{SupportsAuth: true, SupportsMultiChain: true})
	_, err := e.GetExchangeCryptocurrencyDepositAddress(t.Context(), "non-existent", "", "", currency.BTC, false)
	assert.ErrorIs(t, err, ErrExchangeNotFound)

	const exchName = "fake"
	r, err := e.GetExchangeCryptocurrencyDepositAddress(t.Context(), exchName, "", "", currency.BTC, false)
	require.NoError(t, err, "GetExchangeCryptocurrencyDepositAddress must not error")
	assert.Equal(t, "fakeaddr", r.Address, "Should return the correct r.Address")
	e.DepositAddressManager = SetupDepositAddressManager()
	err = e.DepositAddressManager.Sync(e.GetAllExchangeCryptocurrencyDepositAddresses())
	assert.NoError(t, err, "Sync should not error")
	_, err = e.GetExchangeCryptocurrencyDepositAddress(t.Context(), "meow", "", "", currency.BTC, false)
	assert.ErrorIs(t, err, ErrExchangeNotFound)
	_, err = e.GetExchangeCryptocurrencyDepositAddress(t.Context(), exchName, "", "", currency.BTC, false)
	assert.NoError(t, err, "GetExchangeCryptocurrencyDepositAddress should not error")
}

func TestGetAllExchangeCryptocurrencyDepositAddresses(t *testing.T) {
	t.Parallel()
	e := createDepositEngine(&fakeDepositExchangeOpts{})
	if r := e.GetAllExchangeCryptocurrencyDepositAddresses(); len(r) > 0 {
		t.Error("should have no addresses returned for an unauthenticated exchange")
	}
	e = createDepositEngine(&fakeDepositExchangeOpts{SupportsAuth: true, ThrowPairError: true})
	if r := e.GetAllExchangeCryptocurrencyDepositAddresses(); len(r) > 0 {
		t.Error("should have no cryptos returned for no enabled pairs")
	}
	e = createDepositEngine(&fakeDepositExchangeOpts{SupportsAuth: true, SupportsMultiChain: true, ThrowTransferChainError: true})
	if r := e.GetAllExchangeCryptocurrencyDepositAddresses(); len(r["fake"]) != 0 {
		t.Error("should have returned no deposit addresses for a fake exchange with transfer error")
	}
	e = createDepositEngine(&fakeDepositExchangeOpts{SupportsAuth: true, SupportsMultiChain: true, ThrowDepositAddressError: true})
	if r := e.GetAllExchangeCryptocurrencyDepositAddresses(); len(r["fake"]["btc"]) != 0 {
		t.Error("should have returned no deposit addresses for fake exchange with deposit error, with multichain support enabled")
	}
	e = createDepositEngine(&fakeDepositExchangeOpts{SupportsAuth: true, SupportsMultiChain: true, RequiresChainSet: true})
	if r := e.GetAllExchangeCryptocurrencyDepositAddresses(); len(r["fake"]["btc"]) == 0 {
		t.Error("should of returned a BTC address")
	}
	e = createDepositEngine(&fakeDepositExchangeOpts{SupportsAuth: true, SupportsMultiChain: true})
	if r := e.GetAllExchangeCryptocurrencyDepositAddresses(); len(r["fake"]["btc"]) == 0 {
		t.Error("should of returned a BTC address")
	}
	e = createDepositEngine(&fakeDepositExchangeOpts{SupportsAuth: true})
	if r := e.GetAllExchangeCryptocurrencyDepositAddresses(); len(r["fake"]["xrp"]) == 0 {
		t.Error("should have returned a XRP address")
	}
}

func TestGetExchangeNames(t *testing.T) {
	t.Parallel()
	bot := CreateTestBot(t)
	if e := bot.GetExchangeNames(true); len(e) == 0 {
		t.Error("exchange names should be populated")
	}
	if err := bot.UnloadExchange(testExchange); err != nil {
		t.Fatal(err)
	}
	if e := bot.GetExchangeNames(true); slices.Contains(e, testExchange) {
		t.Error("Bitstamp should be missing")
	}
	if e := bot.GetExchangeNames(false); len(e) != 0 {
		t.Errorf("Expected %v Received %v", len(e), 0)
	}

	for i := range bot.Config.Exchanges {
		exch, err := bot.ExchangeManager.NewExchangeByName(bot.Config.Exchanges[i].Name)
		require.Truef(t, err == nil || errors.Is(err, ErrExchangeAlreadyLoaded),
			"%s NewExchangeByName must not error: %s", bot.Config.Exchanges[i].Name, err)
		if exch != nil {
			exch.SetDefaults()
			err = bot.ExchangeManager.Add(exch)
			require.NoError(t, err)
		}
	}
	if e := bot.GetExchangeNames(false); len(e) != len(bot.Config.Exchanges) {
		t.Errorf("Expected %v Received %v", len(bot.Config.Exchanges), len(e))
	}
}

func mockCert(derType string, notAfter time.Time) ([]byte, error) {
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, err
	}

	host, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	dnsNames := []string{host}
	if host != "localhost" {
		dnsNames = append(dnsNames, "localhost")
	}

	if notAfter.IsZero() {
		notAfter = time.Now().Add(time.Hour * 24 * 365)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"gocryptotrader"},
			CommonName:   host,
		},
		NotBefore:             time.Now(),
		NotAfter:              notAfter,
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
		return nil, err
	}

	if derType == "" {
		derType = "CERTIFICATE"
	}

	certData := pem.EncodeToMemory(&pem.Block{Type: derType, Bytes: derBytes})
	if certData == nil {
		return nil, err
	}

	return certData, nil
}

func TestVerifyCert(t *testing.T) {
	t.Parallel()

	tester := []struct {
		PEMType       string
		CreateBypass  bool
		NotAfter      time.Time
		ErrorExpected error
	}{
		{
			ErrorExpected: nil,
		},
		{
			CreateBypass:  true,
			ErrorExpected: errCertDataIsNil,
		},
		{
			PEMType:       "MEOW",
			ErrorExpected: errCertTypeInvalid,
		},
		{
			NotAfter:      time.Now().Add(-time.Hour),
			ErrorExpected: errCertExpired,
		},
	}

	for x := range tester {
		var cert []byte
		var err error
		if !tester[x].CreateBypass {
			cert, err = mockCert(tester[x].PEMType, tester[x].NotAfter)
			if err != nil {
				t.Errorf("test %d unexpected error: %s", x, err)
				continue
			}
		}
		err = verifyCert(cert)
		if err != tester[x].ErrorExpected {
			t.Fatalf("test %d expected %v, got %v", x, tester[x].ErrorExpected, err)
		}
	}
}

func TestCheckAndGenCerts(t *testing.T) {
	t.Parallel()

	tempDir := filepath.Join(os.TempDir(), "gct-temp-tls")
	cleanup := func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Errorf("unable to remove temp dir %s, manual deletion required", tempDir)
		}
	}

	if err := genCert(tempDir); err != nil {
		cleanup()
		t.Fatal(err)
	}

	defer cleanup()
	if err := CheckCerts(tempDir); err != nil {
		t.Fatal(err)
	}

	// Now delete cert.pem and test regeneration of cert/key files
	certFile := filepath.Join(tempDir, "cert.pem")
	if err := os.Remove(certFile); err != nil {
		t.Fatal(err)
	}
	if err := CheckCerts(tempDir); err != nil {
		t.Fatal(err)
	}

	// Now call CheckCerts to test an expired cert
	certData, err := mockCert("", time.Now().Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	err = file.Write(certFile, certData)
	if err != nil {
		t.Fatal(err)
	}
	if err = CheckCerts(tempDir); err != nil {
		t.Fatal(err)
	}
}

func TestNewSupportedExchangeByName(t *testing.T) {
	t.Parallel()

	for x := range exchange.Exchanges {
		exch, err := NewSupportedExchangeByName(exchange.Exchanges[x])
		if err != nil {
			t.Fatal(err)
		}

		if exch == nil {
			t.Fatalf("received nil exchange")
		}
	}

	_, err := NewSupportedExchangeByName("")
	assert.ErrorIs(t, err, ErrExchangeNotFound)
}

func TestNewExchangeByNameWithDefaults(t *testing.T) {
	t.Parallel()

	_, err := NewExchangeByNameWithDefaults(t.Context(), "moarunlikelymeow")
	assert.ErrorIs(t, err, ErrExchangeNotFound, "Invalid exchange name should error")
	for x := range exchange.Exchanges {
		name := exchange.Exchanges[x]
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if isCITest() && slices.Contains(blockedCIExchanges, name) {
				t.Skipf("skipping %s due to CI test restrictions", name)
			}
			if slices.Contains(unsupportedDefaultConfigExchanges, name) {
				t.Skipf("skipping %s unsupported", name)
			}
			exch, err := NewExchangeByNameWithDefaults(t.Context(), name)
			if assert.NoError(t, err, "NewExchangeByNameWithDefaults should not error") {
				assert.Equal(t, name, strings.ToLower(exch.GetName()), "Should get correct exchange name")
			}
		})
	}
}

func TestStartPPROF(t *testing.T) {
	t.Parallel()
	assert.NoError(t, StartPPROF(t.Context(), &config.Profiler{Enabled: false}), "StartPPROF with a disabled config should not error")
	pprofConfig := &config.Profiler{
		Enabled:              true,
		ListenAddress:        "",
		MutexProfileFraction: 1,
		BlockProfileRate:     1,
	}
	require.NoError(t, StartPPROF(t.Context(), pprofConfig), "StartPPROF with a valid config must not error")
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://localhost:8085/debug/pprof/mutex", http.NoBody)
	require.NoError(t, err, "NewRequestWithContext must not error")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err, "Do must not error")
	require.Equal(t, http.StatusOK, resp.StatusCode, "Get response status must be OK")
	resp.Body.Close()
	assert.Error(t, StartPPROF(t.Context(), pprofConfig), "StartPPROF with a valid config on already used port should error")
}
