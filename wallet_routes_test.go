package main

import (
	"testing"
)

func TestGetCollatedExchangeAccountInfoByCoin(t *testing.T) {
	GetCollatedExchangeAccountInfoByCoin(GetAllEnabledExchangeAccountInfo().Data)
}

func TestGetAccountCurrencyInfoByExchangeName(t *testing.T) {
	_, err := GetAccountCurrencyInfoByExchangeName(
		GetAllEnabledExchangeAccountInfo().Data, "ANX",
	)
	if err == nil {
		t.Error(
			"Test Failed - Wallet_Routes_Test.go - GetAccountCurrencyInfoByExchangeName",
		)
	}
}

func TestGetAllEnabledExchangeAccountInfo(t *testing.T) {
	if value := GetAllEnabledExchangeAccountInfo(); len(value.Data) != 0 {
		t.Error(
			"Test Failed - Wallet_Routes_Test.go - GetAllEnabledExchangeAccountInfo",
		)
	}
}
