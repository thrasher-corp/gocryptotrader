package portfolio

import (
	"reflect"
	"testing"
)

func TestGetEthereumBalance(t *testing.T) {
	addresses := []string{"0xb794f5ea0ba39494ce839613fffba74279579268",
		"0xe853c56864a2ebe4576a807d26fdc4a0ada51919"}
	nonsenseAddress := []string{
		"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
		"0xe853c56864a2ebe4576a807d26fdc4a0ada51919",
	}

	response, err := GetEthereumBalance(addresses)
	if err != nil {
		t.Errorf("Test Failed - Portfolio GetEthereumBalance() Error: %s", err)
	}
	if len(response.Data) != 2 {
		t.Error(
			"Test Failed - Portfolio GetEthereumBalance()  Error: Incorrect address",
		)
	}

	response, err = GetEthereumBalance(nonsenseAddress)
	if err == nil {
		t.Error("Test Failed - Portfolio GetEthereumBalance()")
	}
	if len(response.Data) != 0 {
		t.Error("Test Failed - Portfolio GetEthereumBalance() error")
	}
}

func TestGetBlockrBalanceSingle(t *testing.T) {
	litecoinAddress := "LdP8Qox1VAhCzLJNqrr74YovaWYyNBUWvL"
	bitcoinAddress := "3D2oetdNuZUqQHPJmcMDDHYoqkyNVsFk9r"
	nonsenseAddress := "DingDong"
	ltc := "LtC"
	btc := "bTc"

	response, err := GetBlockrBalanceSingle(litecoinAddress, ltc)
	if err != nil {
		t.Errorf("Test Failed - Portfolio GetBlockrBalanceSingle() Error: %s", err)
	}
	response, err = GetBlockrBalanceSingle(litecoinAddress, btc)
	if err == nil {
		t.Errorf("Test Failed - Portfolio GetBlockrBalanceSingle() Error: %s", err)
	}
	response, err = GetBlockrBalanceSingle(bitcoinAddress, btc)
	if err != nil {
		t.Errorf("Test Failed - Portfolio GetBlockrBalanceSingle() Error: %s", err)
	}
	response, err = GetBlockrBalanceSingle(bitcoinAddress, ltc)
	if err != nil {
		t.Errorf("Test Failed - Portfolio GetBlockrBalanceSingle() Error: %s", err)
	}
	response, err = GetBlockrBalanceSingle(nonsenseAddress, ltc+btc)
	if err == nil {
		t.Errorf("Test Failed - Portfolio GetBlockrBalanceSingle() Error: %s", err)
	}
	if response.Status == "success" {
		t.Error(
			"Test Failed - Portfolio GetBlockrBalanceSingle() Error: Incorrect status",
		)
	}
}

func TestGetBlockrAddressMulti(t *testing.T) {
	litecoinAddresses := []string{
		"LdP8Qox1VAhCzLJNqrr74YovaWYyNBUWvL", "LVa8wZ983PvWtdwXZ8viK6SocMENLCXkEy",
	}
	bitcoinAddresses := []string{
		"3D2oetdNuZUqQHPJmcMDDHYoqkyNVsFk9r", "3Nxwenay9Z8Lc9JBiywExpnEFiLp6Afp8v",
	}
	nonsenseAddresses := []string{"DingDong", "ningNang"}
	ltc := "LtC"
	btc := "bTc"

	_, err := GetBlockrAddressMulti(litecoinAddresses, ltc)
	if err != nil {
		t.Errorf("Test Failed - Portfolio GetBlockrAddressMulti() Error: %s", err)
	}
	_, err = GetBlockrAddressMulti(bitcoinAddresses, btc)
	if err != nil {
		t.Errorf("Test Failed - Portfolio GetBlockrAddressMulti() Error: %s", err)
	}
	_, err = GetBlockrAddressMulti(nonsenseAddresses, ltc)
	if err == nil {
		t.Errorf("Test Failed - Portfolio GetBlockrAddressMulti() Error")
	}
}

func TestGetAddressBalance(t *testing.T) {
	ltcAddress := "LdP8Qox1VAhCzLJNqrr74YovaWYyNBUWvL"
	ltc := "ltc"
	description := "Description of Wallet"
	balance := float64(1000)

	portfolio := Base{}
	portfolio.AddAddress(ltcAddress, ltc, description, balance)

	addBalance, _ := portfolio.GetAddressBalance("LdP8Qox1VAhCzLJNqrr74YovaWYyNBUWvL")
	if addBalance != balance {
		t.Error("Test Failed - Portfolio GetAddressBalance() Error: Incorrect value")
	}

	addBalance, found := portfolio.GetAddressBalance("WigWham")
	if addBalance != 0 {
		t.Error("Test Failed - Portfolio GetAddressBalance() Error: Incorrect value")
	}
	if found != false {
		t.Error("Test Failed - Portfolio GetAddressBalance() Error: Incorrect value")
	}
}

func TestExchangeExists(t *testing.T) {
	newBase := Base{}
	newBase.AddAddress("someaddress", "LTC", "LTCWALLETTEST", 0.02)
	if !newBase.ExchangeExists("someaddress") {
		t.Error("Test Failed - portfolio_test.go - AddressExists error")
	}
	if newBase.ExchangeExists("bla") {
		t.Error("Test Failed - portfolio_test.go - AddressExists error")
	}
}

func TestAddressExists(t *testing.T) {
	newbase := Base{}
	newbase.AddAddress("someaddress", "LTC", "LTCWALLETTEST", 0.02)
	if !newbase.AddressExists("someaddress") {
		t.Error("Test Failed - portfolio_test.go - AddressExists error")
	}
	if newbase.AddressExists("bla") {
		t.Error("Test Failed - portfolio_test.go - AddressExists error")
	}
}

func TestExchangeAddressExists(t *testing.T) {
	newbase := Base{}
	newbase.AddAddress("someaddress", "LTC", "LTCWALLETTEST", 0.02)
	if !newbase.ExchangeAddressExists("someaddress", "LTC") {
		t.Error("Test Failed - portfolio_test.go - ExchangeAddressExists error")
	}
	if newbase.ExchangeAddressExists("TEST", "LTC") {
		t.Error("Test Failed - portfolio_test.go - ExchangeAddressExists error")
	}

}

func TestUpdateAddressBalance(t *testing.T) {
	newbase := Base{}
	newbase.AddAddress("someaddress", "LTC", "LTCWALLETTEST", 0.02)
	newbase.UpdateAddressBalance("someaddress", 0.03)

	value := newbase.GetPortfolioSummary("LTC")
	if value["LTC"] != 0.03 {
		t.Error("Test Failed - portfolio_test.go - UpdateUpdateAddressBalance error")
	}
}

func TestUpdateExchangeAddressBalance(t *testing.T) {
	newbase := Base{}
	newbase.AddAddress("someaddress", "LTC", "LTCWALLETTEST", 0.02)
	portfolio := GetPortfolio()
	portfolio.SeedPortfolio(newbase)
	portfolio.UpdateExchangeAddressBalance("someaddress", "LTC", 0.04)

	value := portfolio.GetPortfolioSummary("LTC")
	if value["LTC"] != 0.04 {
		t.Error("Test Failed - portfolio_test.go - UpdateExchangeAddressBalance error")
	}
}

func TestAddAddress(t *testing.T) {
	newbase := Base{}
	newbase.AddAddress("someaddress", "LTC", "LTCWALLETTEST", 0.02)
	portfolio := GetPortfolio()
	portfolio.SeedPortfolio(newbase)
	if !portfolio.AddressExists("someaddress") {
		t.Error("Test Failed - portfolio_test.go - AddAddress error")
	}
}

func TestUpdatePortfolio(t *testing.T) {
	newbase := Base{}
	newbase.AddAddress("someaddress", "LTC", "LTCWALLETTEST", 0.02)
	portfolio := GetPortfolio()
	portfolio.SeedPortfolio(newbase)

	value := portfolio.UpdatePortfolio(
		[]string{"LdP8Qox1VAhCzLJNqrr74YovaWYyNBUWvL"}, "LTC",
	)
	if !value {
		t.Error("Test Failed - portfolio_test.go - UpdatePortfolio error")
	}
	value = portfolio.UpdatePortfolio([]string{"Testy"}, "LTC")
	if value {
		t.Error("Test Failed - portfolio_test.go - UpdatePortfolio error")
	}
	value = portfolio.UpdatePortfolio(
		[]string{"LdP8Qox1VAhCzLJNqrr74YovaWYyNBUWvL", "LVa8wZ983PvWtdwXZ8viK6SocMENLCXkEy"},
		"LTC",
	)
	if !value {
		t.Error("Test Failed - portfolio_test.go - UpdatePortfolio error")
	}
	value = portfolio.UpdatePortfolio(
		[]string{"LdP8Qox1VAhCzLJNqrr74YovaWYyNBUWvL", "Testy"}, "LTC",
	)
	if value {
		t.Error("Test Failed - portfolio_test.go - UpdatePortfolio error")
	}
	value = portfolio.UpdatePortfolio(
		[]string{"0xb794f5ea0ba39494ce839613fffba74279579268",
			"0xe853c56864a2ebe4576a807d26fdc4a0ada51919"}, "ETH",
	)
	if !value {
		t.Error("Test Failed - portfolio_test.go - UpdatePortfolio error")
	}
	value = portfolio.UpdatePortfolio(
		[]string{"0xb794f5ea0ba39494ce839613fffba74279579268", "TESTY"}, "ETH",
	)
	if value {
		t.Error("Test Failed - portfolio_test.go - UpdatePortfolio error")
	}
}

func TestGetExchangePortfolio(t *testing.T) {
	newbase := Base{}
	newbase.AddAddress("someaddress", "LTC", "LTCWALLETTEST", 0.02)
	portfolio := GetPortfolio()
	portfolio.SeedPortfolio(newbase)
	value := portfolio.GetExchangePortfolio()
	_, ok := value["ANX"]
	if ok {
		t.Error("Test Failed - portfolio_test.go - GetExchangePortfolio error")
	}
}

func TestGetPersonalPortfolio(t *testing.T) {
	newbase := Base{}
	newbase.AddAddress("someaddress", "LTC", "LTCWALLETTEST", 0.02)
	portfolio := GetPortfolio()
	portfolio.SeedPortfolio(newbase)
	value := portfolio.GetPersonalPortfolio()
	_, ok := value["LTC"]
	if !ok {
		t.Error("Test Failed - portfolio_test.go - GetPersonalPortfolio error")
	}
}

func TestGetPortfolioSummary(t *testing.T) {
	newbase := Base{}
	newbase.AddAddress("someaddress", "LTC", "LTCWALLETTEST", 0.02)
	portfolio := GetPortfolio()
	portfolio.SeedPortfolio(newbase)
	value := portfolio.GetPortfolioSummary("LTC")
	if value["LTC"] != 0.02 {
		t.Error("Test Failed - portfolio_test.go - GetPortfolioGroupedCoin error")
	}
}

func TestGetPortfolioGroupedCoin(t *testing.T) {
	newbase := Base{}
	newbase.AddAddress("someaddress", "LTC", "LTCWALLETTEST", 0.02)
	portfolio := GetPortfolio()
	portfolio.SeedPortfolio(newbase)
	value := portfolio.GetPortfolioGroupedCoin()
	if value["LTC"][0] != "someaddress" {
		t.Error("Test Failed - portfolio_test.go - GetPortfolioGroupedCoin error")
	}
}

func TestSeedPortfolio(t *testing.T) {
	newbase := Base{}
	newbase.AddAddress("someaddress", "LTC", "LTCWALLETTEST", 0.02)
	portfolio := GetPortfolio()
	portfolio.SeedPortfolio(newbase)

	if !portfolio.AddressExists("someaddress") {
		t.Error("Test Failed - portfolio_test.go - SeedPortfolio error")
	}
}

func TestStartPortfolioWatcher(t *testing.T) {
	//Not until testTimeoutFeature and errors
}

func TestGetPortfolio(t *testing.T) {
	ptrBASE := GetPortfolio()
	if reflect.TypeOf(ptrBASE).String() != "*portfolio.Base" {
		t.Error("Test Failed - portfolio_test.go - GetoPortfolio error")
	}
}
