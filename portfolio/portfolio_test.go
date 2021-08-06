package portfolio

import (
	"errors"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
)

const (
	testBTCAddress = "0x1D01TH0R53"
)

func TestGetEthereumBalance(t *testing.T) {
	t.Parallel()
	b := Base{}
	address := "0xb794f5ea0ba39494ce839613fffba74279579268"
	nonsenseAddress := "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"

	response, err := b.GetEthereumBalance(address)
	if err != nil {
		t.Errorf("Portfolio GetEthereumBalance() Error: %s", err)
	}

	if response.Address != "0xb794f5ea0ba39494ce839613fffba74279579268" {
		t.Error("Portfolio GetEthereumBalance() address invalid")
	}

	response, err = b.GetEthereumBalance(nonsenseAddress)
	if !errors.Is(err, errNotEthAddress) {
		t.Errorf("received '%v', expected '%v'", err, errNotEthAddress)
	}
}

func TestGetCryptoIDBalance(t *testing.T) {
	t.Parallel()
	b := Base{}
	ltcAddress := "LX2LMYXtuv5tiYEMztSSoEZcafFPYJFRK1"
	_, err := b.GetCryptoIDAddress(ltcAddress, currency.LTC)
	if err != nil {
		t.Fatalf("TestGetCryptoIDBalance error: %s", err)
	}
}

func TestGetAddressBalance(t *testing.T) {
	t.Parallel()
	ltcAddress := "LdP8Qox1VAhCzLJNqrr74YovaWYyNBUWvL"
	ltc := currency.LTC
	description := "Description of Wallet"
	balance := float64(1000)

	b := Base{}
	err := b.AddAddress(ltcAddress, description, ltc, balance)
	if err != nil {
		t.Error(err)
	}

	addBalance, _ := b.GetAddressBalance("LdP8Qox1VAhCzLJNqrr74YovaWYyNBUWvL",
		description,
		ltc)

	if addBalance != balance {
		t.Error("Incorrect value")
	}

	addBalance, found := b.GetAddressBalance("WigWham",
		description,
		ltc)

	if addBalance != 0 {
		t.Error("Incorrect value")
	}
	if found {
		t.Error("Incorrect value")
	}
}

func TestGetRippleBalance(t *testing.T) {
	t.Parallel()
	b := Base{}
	nonsenseAddress := "Wigwham"
	_, err := b.GetRippleBalance(nonsenseAddress)
	if err == nil {
		t.Error("error cannot be nil on a bad address")
	}

	rippleAddress := "r962iS5subzbVeXZN8MTzyEuuaQKo5qksh"
	_, err = b.GetRippleBalance(rippleAddress)
	if err != nil {
		t.Error(err)
	}
}

func TestExchangeExists(t *testing.T) {
	t.Parallel()
	newBase := Base{}
	err := newBase.AddAddress("someaddress",
		currency.LTC.String(),
		currency.NewCode("LTCWALLETTEST"),
		0.02)
	if err != nil {
		t.Error(err)
	}

	if !newBase.ExchangeExists("someaddress") {
		t.Error("expected exchange to exist")
	}
	if newBase.ExchangeExists("bla") {
		t.Error("expected exchange to not exist")
	}
}

func TestAddressExists(t *testing.T) {
	t.Parallel()
	newBase := Base{}
	err := newBase.AddAddress("someaddress",
		currency.LTC.String(),
		currency.NewCode("LTCWALLETTEST"),
		0.02)
	if err != nil {
		t.Error(err)
	}

	if !newBase.AddressExists("someaddress") {
		t.Error("expected address to exist")
	}
	if newBase.AddressExists("bla") {
		t.Error("expected address to not exist")
	}
}

func TestExchangeAddressExists(t *testing.T) {
	t.Parallel()
	newBase := Base{}
	err := newBase.AddAddress("someaddress",
		currency.LTC.String(),
		currency.LTC,
		0.02)
	if err != nil {
		t.Error(err)
	}

	if !newBase.ExchangeAddressExists("someaddress", currency.LTC) {
		t.Error("expected exchange address to exist")
	}
	if newBase.ExchangeAddressExists("TEST", currency.LTC) {
		t.Error("expected exchange address to not exist")
	}
}

func TestAddExchangeAddress(t *testing.T) {
	t.Parallel()
	newBase := Base{}
	newBase.AddExchangeAddress("OKEX", currency.BTC, 100)
	newBase.AddExchangeAddress("OKEX", currency.BTC, 200)

	if !newBase.ExchangeAddressExists("OKEX", currency.BTC) {
		t.Error("address doesn't exist")
	}
}

func TestUpdateAddressBalance(t *testing.T) {
	t.Parallel()
	newBase := Base{}
	err := newBase.AddAddress("someaddress",
		currency.LTC.String(),
		currency.NewCode("LTCWALLETTEST"),
		0.02)
	if err != nil {
		t.Error(err)
	}

	newBase.UpdateAddressBalance("someaddress", 0.03)

	value := newBase.GetPortfolioSummary()
	if value.Totals[0].Coin != currency.LTC &&
		value.Totals[0].Balance != 0.03 {
		t.Error("UpdateUpdateAddressBalance error")
	}
}

func TestRemoveAddress(t *testing.T) {
	t.Parallel()
	var newBase Base
	if err := newBase.RemoveAddress("", "MEOW", currency.LTC); err == nil {
		t.Error("invalid address should throw an error")
	}

	if err := newBase.RemoveAddress("Gibson", "", currency.NewCode("")); err == nil {
		t.Error("invalid coin type should throw an error")
	}

	if err := newBase.RemoveAddress("HIDDENERINO", "MEOW", currency.LTC); err == nil {
		t.Error("non-existent address should throw an error")
	}

	err := newBase.AddAddress("someaddr",
		currency.LTC.String(),
		currency.NewCode("LTCWALLETTEST"),
		420)
	if err != nil {
		t.Error(err)
	}

	if !newBase.AddressExists("someaddr") {
		t.Error("address does not exist")
	}

	err = newBase.RemoveAddress("someaddr",
		currency.LTC.String(),
		currency.NewCode("LTCWALLETTEST"))
	if err != nil {
		t.Error(err)
	}
	if newBase.AddressExists("someaddr") {
		t.Error("address should not exist")
	}
}

func TestRemoveExchangeAddress(t *testing.T) {
	t.Parallel()
	newBase := Base{}
	exchangeName := "BallerExchange"
	coinType := currency.LTC

	newBase.AddExchangeAddress(exchangeName, coinType, 420)

	if !newBase.ExchangeAddressExists(exchangeName, coinType) {
		t.Error("address does not exist")
	}

	newBase.RemoveExchangeAddress(exchangeName, coinType)
	if newBase.ExchangeAddressExists(exchangeName, coinType) {
		t.Error("address should not exist")
	}
}

func TestUpdateExchangeAddressBalance(t *testing.T) {
	t.Parallel()
	newBase := Base{}
	newBase.AddExchangeAddress("someaddress", currency.LTC, 0.02)
	b := Base{}
	b.Seed(newBase)
	b.UpdateExchangeAddressBalance("someaddress", currency.LTC, 0.04)

	value := b.GetPortfolioSummary()
	if value.Totals[0].Coin != currency.LTC && value.Totals[0].Balance != 0.04 {
		t.Error("incorrect portfolio balance")
	}
}

func TestAddAddress(t *testing.T) {
	t.Parallel()
	var newBase Base
	if err := newBase.AddAddress("", "MEOW", currency.LTC, 1); err == nil {
		t.Error("invalid address should throw an error")
	}

	if err := newBase.AddAddress("Gibson", "", currency.NewCode(""), 1); err == nil {
		t.Error("invalid coin type should throw an error")
	}

	// test adding an exchange address
	err := newBase.AddAddress("COINUT", ExchangeAddress, currency.LTC, 0)
	if err != nil {
		t.Errorf("failed to add address: %v", err)
	}

	// add a test portfolio address and amount
	err = newBase.AddAddress("Gibson",
		currency.LTC.String(),
		currency.NewCode("LTCWALLETTEST"),
		0.02)
	if err != nil {
		t.Error(err)
	}

	// test updating the balance and make sure it's reflected
	err = newBase.AddAddress("Gibson", currency.LTC.String(),
		currency.NewCode("LTCWALLETTEST"), 0.05)
	if err != nil {
		t.Error(err)
	}
	b, _ := newBase.GetAddressBalance("Gibson", "LTC",
		currency.NewCode("LTCWALLETTEST"))
	if b != 0.05 {
		t.Error("invalid portfolio amount")
	}

	nb := Base{}
	nb.Seed(newBase)
	if !nb.AddressExists("Gibson") {
		t.Error("AddAddress error")
	}

	// Test updating balance to <= 0, expected result is to remove the address.
	// Fail if address still exists.
	err = newBase.AddAddress("Gibson",
		currency.LTC.String(),
		currency.NewCode("LTCWALLETTEST"),
		-1)
	if err != nil {
		t.Error(err)
	}

	if newBase.AddressExists("Gibson") {
		t.Error("AddAddress error")
	}
}

func TestUpdatePortfolio(t *testing.T) {
	t.Parallel()
	newBase := Base{}
	err := newBase.UpdatePortfolio([]string{"Testy"}, currency.LTC)
	if err == nil {
		t.Error("UpdatePortfolio error cannot be nil")
	}
	err = newBase.UpdatePortfolio([]string{
		"LdP8Qox1VAhCzLJNqrr74YovaWYyNBUWvL",
		"LVa8wZ983PvWtdwXZ8viK6SocMENLCXkEy"},
		currency.LTC,
	)
	if err != nil {
		t.Error("UpdatePortfolio error", err)
	}
	err = newBase.UpdatePortfolio(
		[]string{"Testy"}, currency.LTC,
	)
	if err == nil {
		t.Error("UpdatePortfolio error cannot be nil")
	}

	err = newBase.UpdatePortfolio([]string{
		"0xb794f5ea0ba39494ce839613fffba74279579268"},
		currency.ETH)
	if err != nil {
		t.Error(err)
	}
	err = newBase.UpdatePortfolio([]string{
		"TESTY"},
		currency.ETH)
	if err == nil {
		t.Error("UpdatePortfolio error cannot be nil")
	}

	err = newBase.UpdatePortfolio([]string{ExchangeAddress,
		PersonalAddress},
		currency.LTC)
	if err != nil {
		t.Error(err)
	}

	err = newBase.UpdatePortfolio([]string{
		"r962iS5subzbVeXZN8MTzyEuuaQKo5qksh"},
		currency.XRP)
	if err != nil {
		t.Error(err)
	}

	err = newBase.UpdatePortfolio([]string{
		"TESTY"},
		currency.XRP)
	if err == nil {
		t.Error("error cannot be nil")
	}
}

func TestGetPortfolioByExchange(t *testing.T) {
	t.Parallel()
	newBase := Base{}
	newBase.AddExchangeAddress("OKEX", currency.LTC, 0.07)
	newBase.AddExchangeAddress("Bitfinex", currency.LTC, 0.05)
	err := newBase.AddAddress("someaddress", "LTC", currency.NewCode(PersonalAddress), 0.03)
	if err != nil {
		t.Fatal(err)
	}
	value := newBase.GetPortfolioByExchange("OKEX")
	result, ok := value[currency.LTC]
	if !ok {
		t.Error("missing portfolio entry")
	}

	if result != 0.07 {
		t.Error("incorrect result")
	}

	value = newBase.GetPortfolioByExchange("Bitfinex")
	result, ok = value[currency.LTC]
	if !ok {
		t.Error("missing portfolio entry")
	}

	if result != 0.05 {
		t.Error("incorrect result")
	}
}

func TestGetExchangePortfolio(t *testing.T) {
	t.Parallel()
	newBase := Base{}
	err := newBase.AddAddress("OKEX", ExchangeAddress, currency.LTC, 0.03)
	if err != nil {
		t.Fatal(err)
	}
	err = newBase.AddAddress("Bitfinex", ExchangeAddress, currency.LTC, 0.05)
	if err != nil {
		t.Fatal(err)
	}
	err = newBase.AddAddress("someaddress", PersonalAddress, currency.LTC, 0.03)
	if err != nil {
		t.Fatal(err)
	}

	value := newBase.GetExchangePortfolio()

	result, ok := value[currency.LTC]
	if !ok {
		t.Error("missing portfolio entry")
	}

	if result != 0.08 {
		t.Error("result != 0.08")
	}
}

func TestGetPersonalPortfolio(t *testing.T) {
	t.Parallel()
	newBase := Base{}
	err := newBase.AddAddress("someaddress", PersonalAddress, currency.N2O, 0.02)
	if err != nil {
		t.Fatal(err)
	}
	err = newBase.AddAddress("anotheraddress", PersonalAddress, currency.N2O, 0.03)
	if err != nil {
		t.Fatal(err)
	}
	err = newBase.AddAddress("Exchange", ExchangeAddress, currency.N2O, 0.01)
	if err != nil {
		t.Fatal(err)
	}

	value := newBase.GetPersonalPortfolio()
	result, ok := value[currency.N2O]
	if !ok {
		t.Error("GetPersonalPortfolio error")
	}

	if result != 0.05 {
		t.Error("GetPersonalPortfolio result != 0.05")
	}
}

func TestGetPortfolioSummary(t *testing.T) {
	t.Parallel()
	newBase := Base{}
	// Personal holdings
	err := newBase.AddAddress("someaddress", PersonalAddress, currency.LTC, 1)
	if err != nil {
		t.Fatal(err)
	}
	err = newBase.AddAddress("someaddress2", PersonalAddress, currency.LTC, 2)
	if err != nil {
		t.Fatal(err)
	}
	err = newBase.AddAddress("someaddress3", PersonalAddress, currency.BTC, 100)
	if err != nil {
		t.Fatal(err)
	}
	err = newBase.AddAddress("0xde0b295669a9fd93d5f28d9ec85e40f4cb697bae",
		PersonalAddress, currency.ETH, 865346880000000000)
	if err != nil {
		t.Fatal(err)
	}
	err = newBase.AddAddress("0x9edc81c813b26165f607a8d1b8db87a02f34307f",
		PersonalAddress, currency.ETH, 165346880000000000)
	if err != nil {
		t.Fatal(err)
	}

	// Exchange holdings
	newBase.AddExchangeAddress("Bitfinex", currency.LTC, 20)
	newBase.AddExchangeAddress("Bitfinex", currency.BTC, 100)
	newBase.AddExchangeAddress("OKEX", currency.ETH, 42)

	value := newBase.GetPortfolioSummary()

	getTotalsVal := func(c currency.Code) Coin {
		for x := range value.Totals {
			if value.Totals[x].Coin == c {
				return value.Totals[x]
			}
		}
		return Coin{}
	}

	if getTotalsVal(currency.LTC).Coin != currency.LTC {
		t.Error("mismatched currency")
	}

	if getTotalsVal(currency.ETH).Coin == currency.LTC {
		t.Error("mismatched currency")
	}

	if getTotalsVal(currency.LTC).Balance != 23 {
		t.Error("incorrect balance")
	}

	if getTotalsVal(currency.BTC).Balance != 200 {
		t.Error("incorrect balance")
	}
}

func TestGetPortfolioGroupedCoin(t *testing.T) {
	t.Parallel()
	newBase := Base{}
	err := newBase.AddAddress("someaddress", currency.LTC.String(), currency.LTC, 0.02)
	if err != nil {
		t.Fatal(err)
	}
	err = newBase.AddAddress("Exchange", ExchangeAddress, currency.LTC, 0.05)
	if err != nil {
		t.Fatal(err)
	}

	value := newBase.GetPortfolioGroupedCoin()
	if value[currency.LTC][0] != "someaddress" && len(value[currency.LTC][0]) != 1 {
		t.Error("incorrect balance")
	}
}

func TestSeed(t *testing.T) {
	t.Parallel()
	newBase := Base{}
	err := newBase.AddAddress("someaddress", currency.LTC.String(), currency.LTC, 0.02)
	if err != nil {
		t.Fatal(err)
	}
	if !newBase.AddressExists("someaddress") {
		t.Error("Seed error")
	}
}

func TestIsExchangeSupported(t *testing.T) {
	t.Parallel()
	newBase := seedPortFolioForTest(t)
	ret := newBase.IsExchangeSupported("BTC Markets", core.BitcoinDonationAddress)
	if !ret {
		t.Fatal("expected IsExchangeSupported() to return true")
	}
	ret = newBase.IsExchangeSupported("Kraken", core.BitcoinDonationAddress)
	if ret {
		t.Fatal("expected IsExchangeSupported() to return false")
	}
}

func TestIsColdStorage(t *testing.T) {
	t.Parallel()
	newBase := seedPortFolioForTest(t)
	ret := newBase.IsColdStorage(core.BitcoinDonationAddress)
	if !ret {
		t.Fatal("expected IsColdStorage() to return true")
	}
	ret = newBase.IsColdStorage(testBTCAddress)
	if ret {
		t.Fatal("expected IsColdStorage() to return false")
	}
	ret = newBase.IsColdStorage("hello")
	if ret {
		t.Fatal("expected IsColdStorage() to return false")
	}
}

func TestIsWhiteListed(t *testing.T) {
	t.Parallel()
	b := seedPortFolioForTest(t)
	ret := b.IsWhiteListed(core.BitcoinDonationAddress)
	if !ret {
		t.Fatal("expected IsWhiteListed() to return true")
	}
	ret = b.IsWhiteListed(testBTCAddress)
	if ret {
		t.Fatal("expected IsWhiteListed() to return false")
	}
	ret = b.IsWhiteListed("hello")
	if ret {
		t.Fatal("expected IsWhiteListed() to return false")
	}
}

func TestStartPortfolioWatcher(t *testing.T) {
	t.Parallel()
	newBase := Base{}
	err := newBase.AddAddress("LX2LMYXtuv5tiYEMztSSoEZcafFPYJFRK1",
		currency.LTC.String(),
		currency.NewCode(PersonalAddress),
		0.02)
	if err != nil {
		t.Error(err)
	}

	err = newBase.AddAddress("Testy",
		currency.LTC.String(),
		currency.NewCode(PersonalAddress),
		0.02)
	if err != nil {
		t.Error(err)
	}

	if !newBase.AddressExists("LX2LMYXtuv5tiYEMztSSoEZcafFPYJFRK1") {
		t.Error("address does not exist")
	}

	go newBase.StartPortfolioWatcher()
}

func seedPortFolioForTest(t *testing.T) *Base {
	t.Helper()
	newBase := Base{}

	err := newBase.AddAddress(core.BitcoinDonationAddress, "test", currency.BTC, 1500)
	if err != nil {
		t.Fatalf("failed to add portfolio address with reason: %v, unable to continue tests", err)
	}
	newBase.Addresses[0].WhiteListed = true
	newBase.Addresses[0].ColdStorage = true
	newBase.Addresses[0].SupportedExchanges = "BTC Markets,Binance"

	err = newBase.AddAddress(testBTCAddress, "test", currency.BTC, 1500)
	if err != nil {
		t.Fatalf("failed to add portfolio address with reason: %v, unable to continue tests", err)
	}
	newBase.Addresses[1].SupportedExchanges = "BTC Markets,Binance"
	b := Base{}
	b.Seed(newBase)
	if len(b.Addresses) == 0 {
		t.Error("failed to seed")
	}
	return &b
}
