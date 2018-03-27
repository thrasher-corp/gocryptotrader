package exmo

import "testing"

const (
	APIKey    = ""
	APISecret = ""
)

var (
	e EXMO
)

func TestDefault(t *testing.T) {
	e.SetDefaults()
}

func TestSetup(t *testing.T) {
	e.AuthenticatedAPISupport = true
	e.APIKey = APIKey
	e.APISecret = APISecret
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetTrades("BTC_USD")
	if err != nil {
		t.Errorf("Test failed. Err: %s", err)
	}
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderbook("BTC_USD")
	if err != nil {
		t.Errorf("Test failed. Err: %s", err)
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := e.GetTicker("BTC_USD")
	if err != nil {
		t.Errorf("Test failed. Err: %s", err)
	}
}

func TestGetPairSettings(t *testing.T) {
	t.Parallel()
	_, err := e.GetPairSettings()
	if err != nil {
		t.Errorf("Test failed. Err: %s", err)
	}
}

func TestGetCurrency(t *testing.T) {
	t.Parallel()
	_, err := e.GetCurrency()
	if err != nil {
		t.Errorf("Test failed. Err: %s", err)
	}
}

func TestGetUserInfo(t *testing.T) {
	t.Parallel()
	if APIKey == "" || APISecret == "" {
		t.Skip()
	}
	TestSetup(t)
	_, err := e.GetUserInfo()
	if err != nil {
		t.Errorf("Test failed. Err: %s", err)
	}
}

func TestGetRequiredAmount(t *testing.T) {
	t.Parallel()
	if APIKey == "" || APISecret == "" {
		t.Skip()
	}
	TestSetup(t)
	_, err := e.GetRequiredAmount("BTC_USD", 100)
	if err != nil {
		t.Errorf("Test failed. Err: %s", err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	if APIKey == "" || APISecret == "" {
		t.Skip()
	}
	TestSetup(t)
	_, err := e.GetDepositAddress()
	if err == nil {
		t.Errorf("Test failed. Err: %s", err)
	}
}
