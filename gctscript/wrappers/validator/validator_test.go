package validator

import (
	"testing"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

var (
	currencyPair = currency.NewPairFromString("BTCAUD")
	testWrapper  = Wrapper{}
)

func TestWrapper_IsEnabled(t *testing.T) {
	f := testWrapper.IsEnabled("hello")
	if !f {
		t.Fatal("expected IsEnabled to return true for enabled exchange")
	}

	f = testWrapper.IsEnabled(exchError.String())
	if f {
		t.Fatal("expected IsEnabled to return false for disabled exchange")
	}
}

func TestWrapper_AccountInformation(t *testing.T) {
	_, err := testWrapper.AccountInformation(exchError.String())
	if err == nil {
		t.Fatal("expected AccountInformation to return error on invalid name")
	}
}

func TestWrapper_CancelOrder(t *testing.T) {
	_, err := testWrapper.CancelOrder(exchError.String(), "")
	if err == nil {
		t.Fatal("expected CancelOrder to return error on invalid name")
	}
}

func TestWrapper_DepositAddress(t *testing.T) {
	_, err := testWrapper.DepositAddress(exchError.String(), currency.NewCode("BTC"))
	if err == nil {
		t.Fatal("expected DepositAddress to return error on invalid name")
	}
}

func TestWrapper_Exchanges(t *testing.T) {

}

func TestWrapper_Orderbook(t *testing.T) {
	_, err := testWrapper.Orderbook(exchError.String(), currencyPair, asset.Spot)
	if err == nil {
		t.Fatal("expected Orderbook to return error with invalid name")
	}
}

func TestWrapper_Pairs(t *testing.T) {
	_, err := testWrapper.Pairs(exchError.String(), false, asset.Spot)
	if err == nil {
		t.Fatal("expected Pairs to return error on invalid name")
	}
}

func TestWrapper_QueryOrder(t *testing.T) {
	_, err := testWrapper.QueryOrder(exchError.String(), "")
	if err == nil {
		t.Fatal("expected QueryOrder to return error on invalid name")
	}
}

func TestWrapper_SubmitOrder(t *testing.T) {
	_, err := testWrapper.SubmitOrder(exchError.String(), nil)
	if err == nil {
		t.Fatal("expected SubmitOrder to return error with invalid name")
	}
}

func TestWrapper_Ticker(t *testing.T) {
	_, err := testWrapper.Ticker(exchError.String(), currencyPair, asset.Spot)
	if err == nil {
		t.Fatal("expected Ticker to return error with invalid name")
	}
}

func TestWrapper_WithdrawalCryptoFunds(t *testing.T) {
	_, err := testWrapper.WithdrawalCryptoFunds(exchError.String(), nil)
	if err == nil {
		t.Fatal("expected WithdrawalCryptoFunds to return error with invalid name")
	}
}

func TestWrapper_WithdrawalFiatFunds(t *testing.T) {
	_, err := testWrapper.WithdrawalFiatFunds(exchError.String(), "", nil)
	if err == nil {
		t.Fatal("expected WithdrawalFiatFunds to return error with invalid name")
	}
}
