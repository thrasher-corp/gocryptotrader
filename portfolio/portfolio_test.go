package portfolio

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
)

const (
	testInvalidBTCAddress = "0x1D01TH0R53"
	testLTCAddress        = "LX2LMYXtuv5tiYEMztSSoEZcafFPYJFRK1"
	testBTCAddress        = "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa"
	testETHAddress        = "0xb794f5ea0ba39494ce839613fffba74279579268"
	testXRPAddress        = "rs8ZPbYqgecRcDzQpJYAMhSxSi5htsjnza"
	cryptoIDAPIKey        = ""
)

func TestGetEthereumAddressBalance(t *testing.T) {
	t.Parallel()
	b := Base{}

	_, err := b.GetEthereumAddressBalance(t.Context(), testBTCAddress)
	assert.ErrorIs(t, err, common.ErrAddressIsEmptyOrInvalid)

	_, err = b.GetEthereumAddressBalance(t.Context(), testETHAddress)
	assert.NoError(t, err, "GetEthereumAddressBalance should not error")
}

func TestGetCryptoIDAddressBalance(t *testing.T) {
	t.Parallel()
	b := Base{}

	_, err := b.GetCryptoIDAddressBalance(t.Context(), testInvalidBTCAddress, currency.BTC)
	assert.ErrorIs(t, err, common.ErrAddressIsEmptyOrInvalid)

	_, err = b.GetCryptoIDAddressBalance(t.Context(), testLTCAddress, currency.LTC)
	assert.ErrorIs(t, err, errProviderNotFound)

	b.Providers = providers{{Name: "CryptoID"}}
	_, err = b.GetCryptoIDAddressBalance(t.Context(), testLTCAddress, currency.LTC)
	assert.ErrorIs(t, err, errProviderAPIKeyNotSet)

	b.Providers[0].APIKey = "bob"
	ctx, cancel := context.WithDeadline(t.Context(), time.Now().Add(time.Nanosecond))
	defer cancel()
	_, err = b.GetCryptoIDAddressBalance(ctx, testLTCAddress, currency.LTC)
	assert.True(t, errors.Is(err, context.DeadlineExceeded) || strings.Contains(err.Error(), "rate limiter wait error"),
		"GetCryptoIDAddressBalance should return DeadlineExceeded or rate limiter wait error")

	if cryptoIDAPIKey == "" {
		t.Skip("Skipping test as CryptoID API key is not set")
	}

	b.Providers[0].APIKey = cryptoIDAPIKey
	_, err = b.GetCryptoIDAddressBalance(t.Context(), testLTCAddress, currency.LTC)
	assert.NoError(t, err, "GetCryptoIDAddressBalance should not error")
}

func TestGetRippleAddressBalance(t *testing.T) {
	t.Parallel()
	b := Base{}

	_, err := b.GetRippleAddressBalance(t.Context(), testXRPAddress)
	assert.NoError(t, err, "GetRippleAddressBalance should not error")
}

func TestGetAddressBalance(t *testing.T) {
	t.Parallel()

	const (
		description = "Description of Wallet"
		balance     = 1000.0
	)

	b := Base{}
	assert.NoError(t, b.AddAddress(testLTCAddress, description, currency.LTC, balance))

	r, ok := b.GetAddressBalance("meow", description, currency.LTC)
	assert.False(t, ok, "GetAddressBalance should return false for non-existent address")
	assert.Zero(t, r, "GetAddressBalance should return 0 for non-existent address")

	r, ok = b.GetAddressBalance(testLTCAddress, description, currency.LTC)
	assert.True(t, ok, "GetAddressBalance should return true for existing address")
	assert.Equal(t, balance, r, "GetAddressBalance should return the correct balance")
}

func TestExchangeExists(t *testing.T) {
	t.Parallel()
	b := Base{}
	assert.False(t, b.ExchangeExists("someaddress"))
	b.AddExchangeAddress("someaddress", currency.LTC, 0.02)
	assert.True(t, b.ExchangeExists("someaddress"))
}

func TestAddressExists(t *testing.T) {
	t.Parallel()
	b := Base{}
	assert.False(t, b.AddressExists("meow"))
	assert.NoError(t, b.AddAddress("someaddress", "desc", currency.NewCode("LTCWALLETTEST"), 0.02))
	assert.True(t, b.AddressExists("someaddress"))
}

func TestExchangeAddressCoinExists(t *testing.T) {
	t.Parallel()
	b := Base{}
	assert.False(t, b.ExchangeAddressCoinExists("someaddress", currency.LTC))
	b.AddExchangeAddress("someaddress", currency.LTC, 0.02)
	assert.True(t, b.ExchangeAddressCoinExists("someaddress", currency.LTC))
	assert.False(t, b.ExchangeAddressCoinExists("someaddress", currency.BTC))
}

func TestAddExchangeAddress(t *testing.T) {
	t.Parallel()
	b := Base{}
	b.AddExchangeAddress("someaddress", currency.LTC, 69)
	bal, ok := b.GetAddressBalance("someaddress", ExchangeAddress, currency.LTC)
	assert.True(t, ok, "GetAddressBalance should return true for existing address")
	assert.Equal(t, 69.0, bal, "GetAddressBalance should return the correct balance")
	b.AddExchangeAddress("someaddress", currency.LTC, 420)
	bal, ok = b.GetAddressBalance("someaddress", ExchangeAddress, currency.LTC)
	assert.True(t, ok, "GetAddressBalance should return true for existing address")
	assert.Equal(t, 420.0, bal, "GetAddressBalance should return the correct balance")
}

func TestUpdateAddressBalance(t *testing.T) {
	t.Parallel()
	b := Base{}
	assert.NoError(t, b.AddAddress("someaddress", "desc", currency.LTC, 0.02))
	b.UpdateAddressBalance("someaddress", 0.03)
	bal, ok := b.GetAddressBalance("someaddress", "desc", currency.LTC)
	assert.True(t, ok, "GetAddressBalance should return true for existing address")
	assert.Equal(t, 0.03, bal, "GetAddressBalance should return the correct balance")
}

func TestRemoveExchangeAddress(t *testing.T) {
	t.Parallel()
	b := Base{}
	b.AddExchangeAddress("BallerExchange", currency.LTC, 420)
	bal, ok := b.GetAddressBalance("BallerExchange", ExchangeAddress, currency.LTC)
	assert.True(t, ok, "GetAddressBalance should return true for existing address")
	assert.Equal(t, 420.0, bal, "GetAddressBalance should return the correct balance")
	b.RemoveExchangeAddress("BallerExchange", currency.LTC)
	bal, ok = b.GetAddressBalance("BallerExchange", ExchangeAddress, currency.LTC)
	assert.False(t, ok, "GetAddressBalance should return false for non-existent address")
	assert.Zero(t, bal, "GetAddressBalance should return 0 for non-existent address")
}

func TestUpdateExchangeAddressBalance(t *testing.T) {
	t.Parallel()
	b := Base{}
	b.AddExchangeAddress("someaddress", currency.LTC, 0.02)
	b.UpdateExchangeAddressBalance("someaddress", currency.LTC, 0.04)
	bal, ok := b.GetAddressBalance("someaddress", ExchangeAddress, currency.LTC)
	assert.True(t, ok, "GetAddressBalance should return true for existing address")
	assert.Equal(t, 0.04, bal, "GetAddressBalance should return the correct balance")
}

func TestAddAddress(t *testing.T) {
	t.Parallel()
	b := Base{}
	assert.ErrorIs(t, b.AddAddress("", "desc", currency.LTC, 0.02), common.ErrAddressIsEmptyOrInvalid)
	assert.ErrorIs(t, b.AddAddress("someaddress", "", currency.EMPTYCODE, 0.02), currency.ErrCurrencyCodeEmpty)
	assert.NoError(t, b.AddAddress("okx", ExchangeAddress, currency.LTC, 0.02))
	assert.True(t, b.ExchangeAddressCoinExists("okx", currency.LTC), "ExchangeAddressCoinExists should return true for an existing address and coin")
	assert.NoError(t, b.AddAddress("someaddress", PersonalAddress, currency.LTC, 0.03))
	assert.True(t, b.AddressExists("someaddress"), "AddressExists should return true for an existing address")
	assert.NoError(t, b.AddAddress("someaddress", PersonalAddress, currency.LTC, 69))
	bal, ok := b.GetAddressBalance("someaddress", PersonalAddress, currency.LTC)
	assert.True(t, ok, "GetAddressBalance should return true for existing address")
	assert.Equal(t, 69.0, bal, "GetAddressBalance should return the correct balance")
}

func TestRemoveAddress(t *testing.T) {
	t.Parallel()
	b := Base{}
	assert.ErrorIs(t, b.RemoveAddress("", "desc", currency.LTC), common.ErrAddressIsEmptyOrInvalid)
	assert.ErrorIs(t, b.RemoveAddress("someaddress", "", currency.EMPTYCODE), currency.ErrCurrencyCodeEmpty)
	assert.ErrorIs(t, b.RemoveAddress("someaddress", "desc", currency.LTC), errPortfolioItemNotFound)
	assert.NoError(t, b.AddAddress("someaddress", "desc", currency.LTC, 0.02))
	assert.NoError(t, b.RemoveAddress("someaddress", "desc", currency.LTC))
	assert.False(t, b.AddressExists("someaddress"), "AddressExists should return false for non-existent address")
}

func TestUpdatePortfolio(t *testing.T) {
	t.Parallel()
	b := Base{
		Providers: providers{
			{
				Name:    "XRPScan",
				Enabled: true,
			},
			{
				Name:    "Ethplorer",
				Enabled: true,
			},
		},
	}

	assert.NoError(t, b.UpdatePortfolio(t.Context(), []string{PersonalAddress, ExchangeAddress}, currency.LTC))
	assert.NoError(t, b.UpdatePortfolio(t.Context(), []string{testETHAddress}, currency.ETH))
	assert.NoError(t, b.UpdatePortfolio(t.Context(), []string{testXRPAddress}, currency.XRP))
	assert.ErrorIs(t, b.UpdatePortfolio(t.Context(), []string{testETHAddress}, currency.ADA), currency.ErrCurrencyNotSupported)
	assert.ErrorIs(t, b.UpdatePortfolio(t.Context(), []string{testBTCAddress}, currency.BTC), errProviderNotFound)

	b.Providers = append(b.Providers, provider{
		Name: "CryptoID",
	})

	assert.ErrorIs(t, b.UpdatePortfolio(t.Context(), []string{testLTCAddress}, currency.LTC), errProviderNotEnabled)
	b.Providers[2].Enabled = true
	assert.ErrorIs(t, b.UpdatePortfolio(t.Context(), []string{testLTCAddress}, currency.LTC), errProviderAPIKeyNotSet)

	if cryptoIDAPIKey == "" {
		t.Skip("Skipping test as CryptoID API key is not set")
	}
	b.Providers[2].APIKey = cryptoIDAPIKey
	assert.NoError(t, b.UpdatePortfolio(t.Context(), []string{testLTCAddress}, currency.LTC))
	assert.NoError(t, b.UpdatePortfolio(t.Context(), []string{testBTCAddress}, currency.BTC))
}

func TestGetPortfolioByExchange(t *testing.T) {
	t.Parallel()
	b := Base{}
	b.AddExchangeAddress("Okx", currency.LTC, 0.07)
	b.AddExchangeAddress("Bitfinex", currency.LTC, 0.05)
	assert.NoError(t, b.AddAddress("someaddress", "LTC", currency.NewCode(PersonalAddress), 0.03))
	assert.Equal(t, 0.07, b.GetPortfolioByExchange("Okx")[currency.LTC], "GetPortfolioByExchange should return the correct balance")
	assert.Equal(t, 0.05, b.GetPortfolioByExchange("Bitfinex")[currency.LTC], "GetPortfolioByExchange should return the correct balance")
}

func TestGetExchangePortfolio(t *testing.T) {
	t.Parallel()
	b := Base{}
	assert.NoError(t, b.AddAddress("Okx", ExchangeAddress, currency.LTC, 0.03))
	assert.NoError(t, b.AddAddress("Bitfinex", ExchangeAddress, currency.LTC, 0.05))
	assert.NoError(t, b.AddAddress("someaddress", PersonalAddress, currency.LTC, 0.03))
	assert.Equal(t, 0.08, b.GetExchangePortfolio()[currency.LTC], "GetExchangePortfolio should return the correct balance")
}

func TestGetPersonalPortfolio(t *testing.T) {
	t.Parallel()
	b := Base{}
	assert.NoError(t, b.AddAddress("someaddress", PersonalAddress, currency.WIF, 0.02))
	assert.NoError(t, b.AddAddress("anotheraddress", PersonalAddress, currency.WIF, 0.03))
	assert.NoError(t, b.AddAddress("Exchange", ExchangeAddress, currency.WIF, 0.01))
	assert.Equal(t, 0.05, b.GetPersonalPortfolio()[currency.WIF], "GetPersonalPortfolio should return the correct balance")
}

func TestGetPortfolioSummary(t *testing.T) {
	t.Parallel()
	b := Base{}
	// Personal holdings
	assert.NoError(t, b.AddAddress("someaddress", PersonalAddress, currency.LTC, 1))
	assert.NoError(t, b.AddAddress("someaddress2", PersonalAddress, currency.LTC, 2))
	assert.NoError(t, b.AddAddress("someaddress3", PersonalAddress, currency.BTC, 100))
	assert.NoError(t, b.AddAddress("0xde0b295669a9fd93d5f28d9ec85e40f4cb697bae", PersonalAddress, currency.ETH, 69))
	assert.NoError(t, b.AddAddress("0x9edc81c813b26165f607a8d1b8db87a02f34307f", PersonalAddress, currency.ETH, 420))

	// Exchange holdings
	b.AddExchangeAddress("Bitfinex", currency.LTC, 20)
	b.AddExchangeAddress("Bitfinex", currency.BTC, 100)
	b.AddExchangeAddress("Okx", currency.ETH, 42)

	value := b.GetPortfolioSummary()

	getTotalsVal := func(c currency.Code) Coin {
		for x := range value.Totals {
			if value.Totals[x].Coin.Equal(c) {
				return value.Totals[x]
			}
		}
		return Coin{}
	}

	assert.Equal(t, currency.LTC, getTotalsVal(currency.LTC).Coin, "Coin should be LTC")
	assert.Equal(t, 23.0, getTotalsVal(currency.LTC).Balance, "LTC balance should be correct")
	assert.Equal(t, 200.0, getTotalsVal(currency.BTC).Balance, "BTC balance should be correct")
	assert.Equal(t, 69.0+420.0+42, getTotalsVal(currency.ETH).Balance, "ETH balance should be correct")
}

func TestGetPortfolioAddressesGroupedByCoin(t *testing.T) {
	t.Parallel()
	b := Base{}
	assert.NoError(t, b.AddAddress(testLTCAddress, PersonalAddress, currency.LTC, 0.02))
	assert.NoError(t, b.AddAddress("Exchange", ExchangeAddress, currency.LTC, 0.03))
	assert.Len(t, b.GetPortfolioAddressesGroupedByCoin(), 1, "GetPortfolioAddressesGroupedByCoin should return the correct number of addresses")
	assert.Equal(t, testLTCAddress, b.GetPortfolioAddressesGroupedByCoin()[currency.LTC][0], "GetPortfolioAddressesGroupedByCoin should return the correct address")
}

func TestIsExchangeSupported(t *testing.T) {
	t.Parallel()
	b := Base{
		Addresses: []Address{
			{
				Address:            core.BitcoinDonationAddress,
				SupportedExchanges: "Binance, BTC Markets",
			},
		},
	}
	assert.True(t, b.IsExchangeSupported("Binance", core.BitcoinDonationAddress), "IsExchangeSupported should return true for supported exchange")
	assert.False(t, b.IsExchangeSupported("Coinbase", core.BitcoinDonationAddress), "IsExchangeSupported should return false for unsupported exchange")
	assert.False(t, b.IsExchangeSupported("Binance", testBTCAddress), "IsExchangeSupported should return false for non-existent address")
}

func TestIsColdStorage(t *testing.T) {
	t.Parallel()
	b := Base{
		Addresses: []Address{
			{
				Address:     core.BitcoinDonationAddress,
				ColdStorage: true,
			},
			{
				Address: testBTCAddress,
			},
		},
	}
	assert.True(t, b.IsColdStorage(core.BitcoinDonationAddress), "IsColdStorage should return true for cold storage address")
	assert.False(t, b.IsColdStorage(testBTCAddress), "IsColdStorage should return false for non-cold storage address")
}

func TestIsWhiteListed(t *testing.T) {
	t.Parallel()
	b := Base{
		Addresses: []Address{
			{
				Address:     core.BitcoinDonationAddress,
				WhiteListed: true,
			},
			{
				Address: testBTCAddress,
			},
		},
	}
	assert.True(t, b.IsWhiteListed(core.BitcoinDonationAddress), "IsWhiteListed should return true for whitelisted address")
	assert.False(t, b.IsWhiteListed(testBTCAddress), "IsWhiteListed should return false for non-whitelisted address")
}

func TestStartPortfolioWatcher(t *testing.T) {
	t.Parallel()
	b := Base{}
	assert.ErrorIs(t, b.StartPortfolioWatcher(t.Context(), time.Second), errNoPortfolioItemsToWatch)

	assert.NoError(t, b.AddAddress(testXRPAddress, PersonalAddress, currency.XRP, 0.02))

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	assert.ErrorIs(t, b.StartPortfolioWatcher(ctx, 0), context.Canceled, "StartPortfolioWatcher should return context.Canceled")

	b.Providers = append(b.Providers, provider{
		Name:    "XRPScan",
		Enabled: true,
	})

	ctx2, cancel2 := context.WithCancel(t.Context())

	doneCh := make(chan error)
	go func() {
		doneCh <- b.StartPortfolioWatcher(ctx2, time.Second)
	}()

	assert.Eventually(t, func() bool {
		portfolio := b.GetPersonalPortfolio()
		xrpBalance, ok := portfolio[currency.XRP]
		return ok && xrpBalance > 0.02
	}, 10*time.Second, time.Second, "GetPersonalPortfolio should return a balance greater than 0.02")

	cancel2()
	assert.ErrorIs(t, <-doneCh, context.Canceled, "StartPortfolioWatcher should return a context canceled error")
}

func TestGetProvider(t *testing.T) {
	t.Parallel()
	b := Base{
		Providers: providers{
			{
				Name:    "XRPScan",
				Enabled: true,
			},
		},
	}
	p, ok := b.Providers.GetProvider("XrPSCaN")
	assert.True(t, ok, "GetProvider should return true for existing provider")
	assert.Equal(t, "XRPScan", p.Name, "GetProvider should return the correct provider name")
	assert.True(t, p.Enabled, "GetProvider should return the correct provider enabled status")
	p, ok = b.Providers.GetProvider("NonExistent")
	assert.False(t, ok, "GetProvider should return false for non-existent provider")
	assert.Equal(t, provider{}, p, "GetProvider should return an empty provider for non-existent provider")
}
