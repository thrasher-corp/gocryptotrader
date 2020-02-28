package engine

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/file"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stats"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
)

var (
	helperTestLoaded = false
)

func SetupTestHelpers(t *testing.T) {
	if !helperTestLoaded {
		if Bot == nil {
			Bot = new(Engine)
		}
		Bot.Config = &config.Cfg
		err := Bot.Config.LoadConfig(config.TestFile, true)
		if err != nil {
			t.Fatalf("SetupTest: Failed to load config: %s", err)
		}
		err = Bot.Config.RetrieveConfigCurrencyPairs(true, asset.Spot)
		if err != nil {
			t.Fatalf("Failed to retrieve config currency pairs. %s", err)
		}
		helperTestLoaded = true
	}

	if GetExchangeByName(testExchange) == nil {
		err := LoadExchange(testExchange, false, nil)
		if err != nil {
			t.Fatalf("SetupTest: Failed to load exchange: %s", err)
		}
	}
	if GetExchangeByName(fakePassExchange) == nil {
		err := addPassingFakeExchange(testExchange)
		if err != nil {
			t.Fatalf("SetupTest: Failed to load exchange: %s", err)
		}
	}
}

func TestGetExchangeOTPs(t *testing.T) {
	SetupTestHelpers(t)
	_, err := GetExchangeOTPs()
	if err == nil {
		t.Fatal("Expected err with no exchange OTP secrets set")
	}

	bfxCfg, err := Bot.Config.GetExchangeConfig("Bitfinex")
	if err != nil {
		t.Fatal(err)
	}
	bCfg, err := Bot.Config.GetExchangeConfig("Bitstamp")
	if err != nil {
		t.Fatal(err)
	}

	bfxCfg.API.Credentials.OTPSecret = "JBSWY3DPEHPK3PXP"
	bCfg.API.Credentials.OTPSecret = "JBSWY3DPEHPK3PXP"
	result, err := GetExchangeOTPs()
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 2 {
		t.Fatal("Expected 2 OTP results")
	}

	bfxCfg.API.Credentials.OTPSecret = "°"
	result, err = GetExchangeOTPs()
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 {
		t.Fatal("Expected 1 OTP code with invalid OTP Secret")
	}

	// Flush settings
	bfxCfg.API.Credentials.OTPSecret = ""
	bCfg.API.Credentials.OTPSecret = ""
}

func TestGetExchangeoOTPByName(t *testing.T) {
	SetupTestHelpers(t)
	_, err := GetExchangeoOTPByName("Bitstamp")
	if err == nil {
		t.Fatal("Expected err with no exchange OTP secrets set")
	}

	bCfg, err := Bot.Config.GetExchangeConfig("Bitstamp")
	if err != nil {
		t.Fatal(err)
	}

	bCfg.API.Credentials.OTPSecret = "JBSWY3DPEHPK3PXP"
	result, err := GetExchangeoOTPByName("Bitstamp")
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
	SetupTestHelpers(t)
	if result := GetAuthAPISupportedExchanges(); len(result) != 1 {
		t.Fatal("Unexpected result", result)
	}
}

func TestIsOnline(t *testing.T) {
	SetupTestHelpers(t)
	if r := IsOnline(); r {
		t.Fatal("Unexpected result")
	}

	if err := Bot.ConnectionManager.Start(); err != nil {
		t.Fatal(err)
	}

	tick := time.NewTicker(time.Second * 5)
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			t.Fatal("Test timeout")
		default:
			if IsOnline() {
				if err := Bot.ConnectionManager.Stop(); err != nil {
					t.Fatal("unable to shutdown connection manager")
				}
				return
			}
		}
	}
}

func TestGetAvailableExchanges(t *testing.T) {
	SetupTestHelpers(t)
	if r := len(GetAvailableExchanges()); r == 0 {
		t.Error("Expected len > 0")
	}
}

func TestGetSpecificAvailablePairs(t *testing.T) {
	SetupTestHelpers(t)
	assetType := asset.Spot
	result := GetSpecificAvailablePairs(true, true, true, false, assetType)

	if !result.Contains(currency.NewPairFromStrings("BTC", "USD"), true) {
		t.Fatal("Unexpected result")
	}

	if !result.Contains(currency.NewPairFromStrings("BTC", "USDT"), false) {
		t.Fatal("Unexpected result")
	}

	result = GetSpecificAvailablePairs(true, true, false, false, assetType)

	if result.Contains(currency.NewPairFromStrings("BTC", "USDT"), false) {
		t.Fatal("Unexpected result")
	}

	result = GetSpecificAvailablePairs(true, false, false, true, assetType)
	if !result.Contains(currency.NewPairFromStrings("LTC", "BTC"), false) {
		t.Fatal("Unexpected result")
	}
}

func TestIsRelatablePairs(t *testing.T) {
	SetupTestHelpers(t)

	// Test relational pairs with similar names
	result := IsRelatablePairs(currency.NewPairFromStrings("XBT", "USD"),
		currency.NewPairFromStrings("BTC", "USD"), false)
	if !result {
		t.Fatal("Unexpected result")
	}

	// Test relational pairs with similar names reversed
	result = IsRelatablePairs(currency.NewPairFromStrings("BTC", "USD"),
		currency.NewPairFromStrings("XBT", "USD"), false)
	if !result {
		t.Fatal("Unexpected result")
	}

	// Test relational pairs with similar names but with Tether support disabled
	result = IsRelatablePairs(currency.NewPairFromStrings("XBT", "USD"),
		currency.NewPairFromStrings("BTC", "USDT"), false)
	if result {
		t.Fatal("Unexpected result")
	}

	// Test relational pairs with similar names but with Tether support enabled
	result = IsRelatablePairs(currency.NewPairFromStrings("XBT", "USDT"),
		currency.NewPairFromStrings("BTC", "USD"), true)
	if !result {
		t.Fatal("Unexpected result")
	}

	// Test relational pairs with different ordering, a delimiter and with
	// Tether support enabled
	result = IsRelatablePairs(currency.NewPairFromStrings("AE", "USDT"),
		currency.NewPairDelimiter("USDT-AE", "-"), true)
	if !result {
		t.Fatal("Unexpected result")
	}

	// Test relational pairs with different ordering, a delimiter and with
	// Tether support disabled
	result = IsRelatablePairs(currency.NewPairFromStrings("AE", "USDT"),
		currency.NewPairDelimiter("USDT-AE", "-"), false)
	if !result {
		t.Fatal("Unexpected result")
	}

	// Test relationl pairs with similar names and different fiat currencies
	result = IsRelatablePairs(currency.NewPairFromStrings("XBT", "EUR"),
		currency.NewPairFromStrings("BTC", "AUD"), false)
	if !result {
		t.Fatal("Unexpected result")
	}

	// Test relationl pairs with similar names, different fiat currencies and
	// with different ordering
	result = IsRelatablePairs(currency.NewPairFromStrings("USD", "BTC"),
		currency.NewPairFromStrings("BTC", "EUR"), false)
	if !result { // Is this really expected result???
		t.Fatal("Unexpected result")
	}

	// Test relationl pairs with similar names, different fiat currencies and
	// with Tether enabled
	result = IsRelatablePairs(currency.NewPairFromStrings("USD", "BTC"),
		currency.NewPairFromStrings("BTC", "USDT"), true)
	if !result {
		t.Fatal("Unexpected result")
	}

	// Test relationl crypto pairs with similar names
	result = IsRelatablePairs(currency.NewPairFromStrings("LTC", "BTC"),
		currency.NewPairFromStrings("BTC", "LTC"), false)
	if !result {
		t.Fatal("Unexpected result")
	}

	// Test relationl crypto pairs with similar different pairs
	result = IsRelatablePairs(currency.NewPairFromStrings("LTC", "ETH"),
		currency.NewPairFromStrings("BTC", "ETH"), false)
	if result {
		t.Fatal("Unexpected result")
	}

	// Test relationl crypto pairs with similar different pairs and with USDT
	// enabled
	result = IsRelatablePairs(currency.NewPairFromStrings("USDT", "USD"),
		currency.NewPairFromStrings("BTC", "USD"), true)
	if result {
		t.Fatal("Unexpected result")
	}

	// Test relationl crypto pairs with with similar names
	result = IsRelatablePairs(currency.NewPairFromStrings("XBT", "LTC"),
		currency.NewPairFromStrings("BTC", "LTC"), false)
	if !result {
		t.Fatal("Unexpected result")
	}

	// Test relationl crypto pairs with different ordering and similar names
	result = IsRelatablePairs(currency.NewPairFromStrings("LTC", "XBT"),
		currency.NewPairFromStrings("BTC", "LTC"), false)
	if !result {
		t.Fatal("Unexpected result")
	}

	// Test edge case between two pairs when currency translations were causing
	// non-relational pairs to be relatable
	result = IsRelatablePairs(currency.NewPairFromStrings("EUR", "USD"),
		currency.NewPairFromStrings("BTC", "USD"), false)
	if result {
		t.Fatal("Unexpected result")
	}
}

func TestGetRelatableCryptocurrencies(t *testing.T) {
	SetupTestHelpers(t)
	p := GetRelatableCryptocurrencies(currency.NewPairFromStrings("BTC", "LTC"))
	if p.Contains(currency.NewPairFromStrings("BTC", "LTC"), true) {
		t.Fatal("Unexpected result")
	}
	if p.Contains(currency.NewPairFromStrings("BTC", "BTC"), true) {
		t.Fatal("Unexpected result")
	}
	if p.Contains(currency.NewPairFromStrings("LTC", "LTC"), true) {
		t.Fatal("Unexpected result")
	}
	if !p.Contains(currency.NewPairFromStrings("BTC", "ETH"), true) {
		t.Fatal("Unexpected result")
	}

	p = GetRelatableCryptocurrencies(currency.NewPairFromStrings("BTC", "LTC"))
	if p.Contains(currency.NewPairFromStrings("BTC", "LTC"), true) {
		t.Fatal("Unexpected result")
	}
	if p.Contains(currency.NewPairFromStrings("BTC", "BTC"), true) {
		t.Fatal("Unexpected result")
	}
	if p.Contains(currency.NewPairFromStrings("LTC", "LTC"), true) {
		t.Fatal("Unexpected result")
	}
	if !p.Contains(currency.NewPairFromStrings("BTC", "ETH"), true) {
		t.Fatal("Unexpected result")
	}
}

func TestGetRelatableFiatCurrencies(t *testing.T) {
	SetupTestHelpers(t)
	p := GetRelatableFiatCurrencies(currency.NewPairFromStrings("BTC", "USD"))
	if !p.Contains(currency.NewPairFromStrings("BTC", "EUR"), true) {
		t.Fatal("Unexpected result")
	}

	p = GetRelatableFiatCurrencies(currency.NewPairFromStrings("BTC", "USD"))
	if !p.Contains(currency.NewPairFromStrings("BTC", "ZAR"), true) {
		t.Fatal("Unexpected result")
	}
}

func TestMapCurrenciesByExchange(t *testing.T) {
	SetupTestHelpers(t)

	var pairs = []currency.Pair{
		currency.NewPair(currency.BTC, currency.USD),
		currency.NewPair(currency.BTC, currency.EUR),
	}

	result := MapCurrenciesByExchange(pairs, true, asset.Spot)
	pairs, ok := result["Bitstamp"]
	if !ok {
		t.Fatal("Unexpected result")
	}

	if len(pairs) != 2 {
		t.Fatal("Unexpected result")
	}
}

func TestGetExchangeNamesByCurrency(t *testing.T) {
	SetupTestHelpers(t)
	assetType := asset.Spot

	result := GetExchangeNamesByCurrency(currency.NewPairFromStrings("BTC", "USD"),
		true,
		assetType)
	if !common.StringDataCompare(result, "Bitstamp") {
		t.Fatal("Unexpected result")
	}

	result = GetExchangeNamesByCurrency(currency.NewPairFromStrings("BTC", "JPY"),
		true,
		assetType)
	if !common.StringDataCompare(result, "Bitflyer") {
		t.Fatal("Unexpected result")
	}

	result = GetExchangeNamesByCurrency(currency.NewPairFromStrings("blah", "JPY"),
		true,
		assetType)
	if len(result) > 0 {
		t.Fatal("Unexpected result")
	}
}

func TestGetSpecificOrderbook(t *testing.T) {
	SetupTestHelpers(t)

	LoadExchange("Bitstamp", false, nil)

	var bids []orderbook.Item
	bids = append(bids, orderbook.Item{Price: 1000, Amount: 1})

	base := orderbook.Base{
		Pair:         currency.NewPair(currency.BTC, currency.USD),
		Bids:         bids,
		ExchangeName: "Bitstamp",
		AssetType:    asset.Spot,
	}

	err := base.Process()
	if err != nil {
		t.Fatal("Unexpected result", err)
	}

	ob, err := GetSpecificOrderbook(currency.NewPairFromString("BTCUSD"),
		"Bitstamp",
		asset.Spot)
	if err != nil {
		t.Fatal(err)
	}

	if ob.Bids[0].Price != 1000 {
		t.Fatal("Unexpected result")
	}

	_, err = GetSpecificOrderbook(currency.NewPairFromStrings("ETH", "LTC"),
		"Bitstamp",
		asset.Spot)
	if err == nil {
		t.Fatal("Unexpected result")
	}

	UnloadExchange("Bitstamp")
}

func TestGetSpecificTicker(t *testing.T) {
	SetupTestHelpers(t)

	LoadExchange("Bitstamp", false, nil)
	p := currency.NewPairFromStrings("BTC", "USD")
	err := ticker.ProcessTicker("Bitstamp",
		&ticker.Price{Pair: p, Last: 1000},
		asset.Spot)
	if err != nil {
		t.Fatal("ProcessTicker error", err)
	}

	tick, err := GetSpecificTicker(currency.NewPairFromStrings("BTC", "USD"), "Bitstamp",
		asset.Spot)
	if err != nil {
		t.Fatal(err)
	}

	if tick.Last != 1000 {
		t.Fatal("Unexpected result")
	}

	_, err = GetSpecificTicker(currency.NewPairFromStrings("ETH", "LTC"), "Bitstamp",
		asset.Spot)
	if err == nil {
		t.Fatal("Unexpected result")
	}

	UnloadExchange("Bitstamp")
}

func TestGetCollatedExchangeAccountInfoByCoin(t *testing.T) {
	SetupTestHelpers(t)

	var exchangeInfo []account.Holdings

	var bitfinexHoldings account.Holdings
	bitfinexHoldings.Exchange = "Bitfinex"
	bitfinexHoldings.Accounts = append(bitfinexHoldings.Accounts,
		account.SubAccount{
			Currencies: []account.Balance{
				{
					CurrencyName: currency.BTC,
					TotalValue:   100,
					Hold:         0,
				},
			},
		})

	exchangeInfo = append(exchangeInfo, bitfinexHoldings)

	var bitstampHoldings account.Holdings
	bitstampHoldings.Exchange = "Bitstamp"
	bitstampHoldings.Accounts = append(bitstampHoldings.Accounts,
		account.SubAccount{
			Currencies: []account.Balance{
				{
					CurrencyName: currency.LTC,
					TotalValue:   100,
					Hold:         0,
				},
				{
					CurrencyName: currency.BTC,
					TotalValue:   100,
					Hold:         0,
				},
			},
		})

	exchangeInfo = append(exchangeInfo, bitstampHoldings)

	result := GetCollatedExchangeAccountInfoByCoin(exchangeInfo)
	if len(result) == 0 {
		t.Fatal("Unexpected result")
	}

	amount, ok := result[currency.BTC]
	if !ok {
		t.Fatal("Expected currency was not found in result map")
	}

	if amount.TotalValue != 200 {
		t.Fatal("Unexpected result")
	}

	_, ok = result[currency.ETH]
	if ok {
		t.Fatal("Unexpected result")
	}
}

func TestGetExchangeHighestPriceByCurrencyPair(t *testing.T) {
	SetupTestHelpers(t)

	p := currency.NewPairFromStrings("BTC", "USD")
	stats.Add("Bitfinex", p, asset.Spot, 1000, 10000)
	stats.Add("Bitstamp", p, asset.Spot, 1337, 10000)
	exchangeName, err := GetExchangeHighestPriceByCurrencyPair(p, asset.Spot)
	if err != nil {
		t.Error(err)
	}

	if exchangeName != "Bitstamp" {
		t.Error("Unexpected result")
	}

	_, err = GetExchangeHighestPriceByCurrencyPair(currency.NewPairFromStrings("BTC", "AUD"),
		asset.Spot)
	if err == nil {
		t.Error("Unexpected result")
	}
}

func TestGetExchangeLowestPriceByCurrencyPair(t *testing.T) {
	SetupTestHelpers(t)

	p := currency.NewPairFromStrings("BTC", "USD")
	stats.Add("Bitfinex", p, asset.Spot, 1000, 10000)
	stats.Add("Bitstamp", p, asset.Spot, 1337, 10000)
	exchangeName, err := GetExchangeLowestPriceByCurrencyPair(p, asset.Spot)
	if err != nil {
		t.Error(err)
	}

	if exchangeName != "Bitfinex" {
		t.Error("Unexpected result")
	}

	_, err = GetExchangeLowestPriceByCurrencyPair(currency.NewPairFromStrings("BTC", "AUD"),
		asset.Spot)
	if err == nil {
		t.Error("Unexpected reuslt")
	}
}

func TestGetCryptocurrenciesByExchange(t *testing.T) {
	SetupTestHelpers(t)

	_, err := GetCryptocurrenciesByExchange("Bitfinex", false, false, asset.Spot)
	if err != nil {
		t.Fatalf("Err %s", err)
	}
}

func TestGetExchangeNames(t *testing.T) {
	SetupTestHelpers(t)
	if e := GetExchangeNames(true); len(e) == 0 {
		t.Error("exchange names should be populated")
	}
	if err := UnloadExchange(testExchange); err != nil {
		t.Fatal(err)
	}
	if e := GetExchangeNames(true); common.StringDataCompare(e, testExchange) {
		t.Error("Bitstamp should be missing")
	}
	if e := GetExchangeNames(false); len(e) != len(Bot.Config.Exchanges) {
		t.Errorf("Expected %v Received %v", len(e), len(Bot.Config.Exchanges))
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
	if err := checkCerts(tempDir); err != nil {
		t.Fatal(err)
	}

	// Now delete cert.pem and test regeneration of cert/key files
	certFile := filepath.Join(tempDir, "cert.pem")
	if err := os.Remove(certFile); err != nil {
		t.Fatal(err)
	}
	if err := checkCerts(tempDir); err != nil {
		t.Fatal(err)
	}

	// Now call checkCerts to test an expired cert
	certData, err := mockCert("", time.Now().Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	err = file.Write(certFile, certData)
	if err != nil {
		t.Fatal(err)
	}
	if err = checkCerts(tempDir); err != nil {
		t.Fatal(err)
	}
}
