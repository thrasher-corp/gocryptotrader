package portfolio

import (
	"testing"
)

func TestGetEthereumBalance(t *testing.T) {
	addresses := []string{"0xb794f5ea0ba39494ce839613fffba74279579268",
		"0xe853c56864a2ebe4576a807d26fdc4a0ada51919"}
	nonsenseAddress := []string{"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA, 0xe853c56864a2ebe4576a807d26fdc4a0ada51919"}

	response, err := GetEthereumBalance(addresses)
	if err != nil {
		t.Errorf("Test Failed - Portfolio GetEthereumBalance() Error: %s", err)
	}
	if len(response.Data) != 2 {
		t.Error("Test Failed - Portfolio GetEthereumBalance()  Error: Incorrect address")
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
		t.Error("Test Failed - Portfolio GetBlockrBalanceSingle() Error: Incorrect status")
	}
}

func TestGetBlockrAddressMulti(t *testing.T) {
	litecoinAddresses := []string{"LdP8Qox1VAhCzLJNqrr74YovaWYyNBUWvL", "LVa8wZ983PvWtdwXZ8viK6SocMENLCXkEy"}
	bitcoinAddresses := []string{"3D2oetdNuZUqQHPJmcMDDHYoqkyNVsFk9r", "3Nxwenay9Z8Lc9JBiywExpnEFiLp6Afp8v"}
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

	portfolio := PortfolioBase{}
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

// func TestExchangeExists(t *testing.T) {
// 	portfolio := PortfolioBase{}
// 	portfolio.SeedPortfolio(port)
// }

func TestAddressExists(t *testing.T) {

}

func TestExchangeAddressExists(t *testing.T) {

}

func TestUpdateAddressBalance(t *testing.T) {

}

func TestUpdateExchangeAddressBalance(t *testing.T) {

}

func TestAddAddress(t *testing.T) {

}

func TestUpdatePortfolio(t *testing.T) {

}

func TestGetExchangePortfolio(t *testing.T) {

}

func TestGetPersonalPortfolio(t *testing.T) {

}

func TestGetPortfolioSummary(t *testing.T) {

}

func TestGetPortfolioGroupedCoin(t *testing.T) {

}

func TestSeedPortfolio(t *testing.T) {

}

func TestStartPortfolioWatcher(t *testing.T) {

}

func TestGetPortfolio(t *testing.T) {

}
