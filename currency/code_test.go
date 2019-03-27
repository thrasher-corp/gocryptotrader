package currency

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/common"
)

func TestRoleString(t *testing.T) {
	if Unset.String() != UnsetRollString {
		t.Errorf("Test Failed - Role String() error expected %s but recieved %s",
			UnsetRollString,
			Unset)
	}

	if Fiat.String() != FiatCurrencyString {
		t.Errorf("Test Failed - Role String() error expected %s but recieved %s",
			FiatCurrencyString,
			Fiat)
	}

	if Cryptocurrency.String() != CryptocurrencyString {
		t.Errorf("Test Failed - Role String() error expected %s but recieved %s",
			CryptocurrencyString,
			Cryptocurrency)
	}

	if Token.String() != TokenString {
		t.Errorf("Test Failed - Role String() error expected %s but recieved %s",
			TokenString,
			Token)
	}

	if Contract.String() != ContractString {
		t.Errorf("Test Failed - Role String() error expected %s but recieved %s",
			ContractString,
			Contract)
	}

	var random Role = 1 << 7

	if random.String() != "UNKNOWN" {
		t.Errorf("Test Failed - Role String() error expected %s but recieved %s",
			"UNKNOWN",
			random)
	}
}

func TestRoleMarshalJSON(t *testing.T) {
	d, err := common.JSONEncode(Fiat)
	if err != nil {
		t.Error("Test Failed - Role MarshalJSON() error", err)
	}

	expected := `"fiatCurrency"`
	if string(d) != expected {
		t.Errorf("Test Failed - Role MarshalJSON() error expected %s but recieved %s",
			expected,
			string(d))
	}
}

func TestRoleUnmarshalJSON(t *testing.T) {
	type AllTheRoles struct {
		RoleOne     Role `json:"RoleOne"`
		RoleTwo     Role `json:"RoleTwo"`
		RoleThree   Role `json:"RoleThree"`
		RoleFour    Role `json:"RoleFour"`
		RoleFive    Role `json:"RoleFive"`
		RoleUnknown Role `json:"RoleUnknown"`
	}

	var outgoing = AllTheRoles{
		RoleOne:   Unset,
		RoleTwo:   Cryptocurrency,
		RoleThree: Fiat,
		RoleFour:  Token,
		RoleFive:  Contract,
	}

	e, err := common.JSONEncode(1337)
	if err != nil {
		t.Fatal("Test Failed - Role UnmarshalJSON() error", err)
	}

	var incoming AllTheRoles
	err = common.JSONDecode(e, &incoming)
	if err == nil {
		t.Fatal("Test Failed - Role UnmarshalJSON() error", err)
	}

	e, err = common.JSONEncode(outgoing)
	if err != nil {
		t.Fatal("Test Failed - Role UnmarshalJSON() error", err)
	}

	err = common.JSONDecode(e, &incoming)
	if err != nil {
		t.Fatal("Test Failed - Role UnmarshalJSON() error", err)
	}

	if incoming.RoleOne != Unset {
		t.Errorf("Test Failed - Role String() error expected %s but recieved %s",
			Unset,
			incoming.RoleOne)
	}

	if incoming.RoleTwo != Cryptocurrency {
		t.Errorf("Test Failed - Role String() error expected %s but recieved %s",
			Cryptocurrency,
			incoming.RoleTwo)
	}

	if incoming.RoleThree != Fiat {
		t.Errorf("Test Failed - Role String() error expected %s but recieved %s",
			Fiat,
			incoming.RoleThree)
	}

	if incoming.RoleFour != Token {
		t.Errorf("Test Failed - Role String() error expected %s but recieved %s",
			Token,
			incoming.RoleFour)
	}

	if incoming.RoleFive != Contract {
		t.Errorf("Test Failed - Role String() error expected %s but recieved %s",
			Contract,
			incoming.RoleFive)
	}

	if incoming.RoleUnknown != Unset {
		t.Errorf("Test Failed - Role String() error expected %s but recieved %s",
			incoming.RoleFive,
			incoming.RoleUnknown)
	}
}

func TestBaseCode(t *testing.T) {
	var main BaseCodes
	if main.HasData() {
		t.Errorf("Test Failed - BaseCode HasData() error expected false but recieved %v",
			main.HasData())
	}

	catsCode := main.Register("CATS")
	if !main.HasData() {
		t.Errorf("Test Failed - BaseCode HasData() error expected true but recieved %v",
			main.HasData())
	}

	if !main.Register("CATS").Match(catsCode) {
		t.Errorf("Test Failed - BaseCode Match() error expected true but recieved %v",
			false)
	}

	if main.Register("DOGS").Match(catsCode) {
		t.Errorf("Test Failed - BaseCode Match() error expected false but recieved %v",
			true)
	}

	loadedCurrencies := main.GetCurrencies()

	if loadedCurrencies.Contains(main.Register("OWLS")) {
		t.Errorf("Test Failed - BaseCode Contains() error expected false but recieved %v",
			true)
	}

	if !loadedCurrencies.Contains(catsCode) {
		t.Errorf("Test Failed - BaseCode Contains() error expected true but recieved %v",
			false)
	}

	err := main.UpdateContract("Bitcoin Perpetual", "XBTUSD", "Bitmex")
	if err != nil {
		t.Error("Test Failed - BaseCode UpdateContract error", err)
	}

	err = main.UpdateCryptocurrency("Bitcoin", "BTC", 1337)
	if err != nil {
		t.Error("Test Failed - BaseCode UpdateContract error", err)
	}

	err = main.UpdateFiatCurrency("Australian Dollar", "AUD", 1336)
	if err != nil {
		t.Error("Test Failed - BaseCode UpdateContract error", err)
	}

	err = main.UpdateToken("Populous", "PPT", "ETH", 1335)
	if err != nil {
		t.Error("Test Failed - BaseCode UpdateContract error", err)
	}

	contract := main.Register("XBTUSD")

	if contract.IsFiatCurrency() {
		t.Errorf("Test Failed - BaseCode IsFiatCurrency() error expected false but recieved %v",
			true)
	}

	if contract.IsCryptocurrency() {
		t.Errorf("Test Failed - BaseCode IsFiatCurrency() error expected false but recieved %v",
			true)
	}

	if contract.IsDefaultFiatCurrency() {
		t.Errorf("Test Failed - BaseCode IsDefaultFiatCurrency() error expected false but recieved %v",
			true)
	}

	if contract.IsDefaultFiatCurrency() {
		t.Errorf("Test Failed - BaseCode IsFiatCurrency() error expected false but recieved %v",
			true)
	}

	err = main.LoadItem(&Item{
		ID:       0,
		FullName: "Cardano",
		Role:     Cryptocurrency,
		Symbol:   "ADA",
	})
	if err != nil {
		t.Error("Test Failed - BaseCode LoadItem() error", err)
	}

	full, err := main.GetFullCurrencyData()
	if err != nil {
		t.Error("Test Failed - BaseCode GetFullCurrencyData error", err)
	}

	if len(full.Contracts) != 1 {
		t.Errorf("Test Failed - BaseCode GetFullCurrencyData() error expected 1 but recieved %v",
			len(full.Contracts))
	}

	if len(full.Cryptocurrency) != 2 {
		t.Errorf("Test Failed - BaseCode GetFullCurrencyData() error expected 1 but recieved %v",
			len(full.Cryptocurrency))
	}

	if len(full.FiatCurrency) != 1 {
		t.Errorf("Test Failed - BaseCode GetFullCurrencyData() error expected 1 but recieved %v",
			len(full.FiatCurrency))
	}

	if len(full.Token) != 1 {
		t.Errorf("Test Failed - BaseCode GetFullCurrencyData() error expected 1 but recieved %v",
			len(full.Token))
	}

	if len(full.UnsetCurrency) != 3 {
		t.Errorf("Test Failed - BaseCode GetFullCurrencyData() error expected 3 but recieved %v",
			len(full.UnsetCurrency))
	}

	if !full.LastMainUpdate.IsZero() {
		t.Errorf("Test Failed - BaseCode GetFullCurrencyData() error expected 0 but recieved %s",
			full.LastMainUpdate)
	}
}

func TestCodeString(t *testing.T) {
	expected := "TEST"
	cc := NewCode("TEST")
	if cc.String() != expected {
		t.Errorf("Test Failed - Currency Code String() error expected %s but recieved %s",
			expected, cc)
	}
}

func TestCodeLower(t *testing.T) {
	expected := "test"
	cc := NewCode("TEST")
	if cc.Lower().String() != expected {
		t.Errorf("Test Failed - Currency Code Lower() error expected %s but recieved %s",
			expected,
			cc.Lower())
	}
}

func TestCodeUpper(t *testing.T) {
	expected := "TEST"
	cc := NewCode("test")
	if cc.Upper().String() != expected {
		t.Errorf("Test Failed - Currency Code Upper() error expected %s but recieved %s",
			expected,
			cc.Upper())
	}
}

func TestCodeUnmarshalJSON(t *testing.T) {
	var unmarshalHere Code
	expected := "BRO"
	encoded, err := common.JSONEncode(expected)
	if err != nil {
		t.Fatal("Test Failed - Currency Code UnmarshalJSON error", err)
	}

	err = common.JSONDecode(encoded, &unmarshalHere)
	if err != nil {
		t.Fatal("Test Failed - Currency Code UnmarshalJSON error", err)
	}

	err = common.JSONDecode(encoded, &unmarshalHere)
	if err != nil {
		t.Fatal("Test Failed - Currency Code UnmarshalJSON error", err)
	}

	if unmarshalHere.String() != expected {
		t.Errorf("Test Failed - Currency Code Upper() error expected %s but recieved %s",
			expected,
			unmarshalHere)
	}
}

func TestCodeMarshalJSON(t *testing.T) {
	quickstruct := struct {
		Codey Code `json:"sweetCodes"`
	}{
		Codey: NewCode("BRO"),
	}

	expectedJSON := `{"sweetCodes":"BRO"}`

	encoded, err := common.JSONEncode(quickstruct)
	if err != nil {
		t.Fatal("Test Failed - Currency Code UnmarshalJSON error", err)
	}

	if string(encoded) != expectedJSON {
		t.Errorf("Test Failed - Currency Code Upper() error expected %s but recieved %s",
			expectedJSON,
			string(encoded))
	}

	quickstruct = struct {
		Codey Code `json:"sweetCodes"`
	}{
		Codey: Code{}, // nil code
	}

	encoded, err = common.JSONEncode(quickstruct)
	if err != nil {
		t.Fatal("Test Failed - Currency Code UnmarshalJSON error", err)
	}

	newExpectedJSON := `{"sweetCodes":""}`
	if string(encoded) != newExpectedJSON {
		t.Errorf("Test Failed - Currency Code Upper() error expected %s but recieved %s",
			newExpectedJSON, string(encoded))
	}
}

func TestIsDefaultCurrency(t *testing.T) {
	if !USD.IsDefaultFiatCurrency() {
		t.Errorf("Test Failed. TestIsDefaultCurrency Cannot match currency %s.",
			USD)
	}
	if !AUD.IsDefaultFiatCurrency() {
		t.Errorf("Test Failed. TestIsDefaultCurrency Cannot match currency, %s.",
			AUD)
	}
	if LTC.IsDefaultFiatCurrency() {
		t.Errorf("Test Failed. TestIsDefaultCurrency Function return is incorrect with, %s.",
			LTC)
	}
}

func TestIsDefaultCryptocurrency(t *testing.T) {
	if !BTC.IsDefaultCryptocurrency() {
		t.Errorf("Test Failed. TestIsDefaultCryptocurrency cannot match currency, %s.",
			BTC)
	}
	if !LTC.IsDefaultCryptocurrency() {
		t.Errorf("Test Failed. TestIsDefaultCryptocurrency cannot match currency, %s.",
			LTC)
	}
	if AUD.IsDefaultCryptocurrency() {
		t.Errorf("Test Failed. TestIsDefaultCryptocurrency function return is incorrect with, %s.",
			AUD)
	}
}

func TestIsFiatCurrency(t *testing.T) {
	if !USD.IsFiatCurrency() {
		t.Errorf(
			"Test Failed. TestIsFiatCurrency cannot match currency, %s.", USD)
	}
	if !CNY.IsFiatCurrency() {
		t.Errorf(
			"Test Failed. TestIsFiatCurrency cannot match currency, %s.", CNY)
	}
	if LINO.IsFiatCurrency() {
		t.Errorf(
			"Test Failed. TestIsFiatCurrency cannot match currency, %s.", LINO,
		)
	}
}

func TestIsCryptocurrency(t *testing.T) {
	if !BTC.IsCryptocurrency() {
		t.Errorf("Test Failed. TestIsFiatCurrency cannot match currency, %s.",
			BTC)
	}
	if !LTC.IsCryptocurrency() {
		t.Errorf("Test Failed. TestIsFiatCurrency cannot match currency, %s.",
			LTC)
	}
	if AUD.IsCryptocurrency() {
		t.Errorf("Test Failed. TestIsFiatCurrency cannot match currency, %s.",
			AUD)
	}
}

func TestItemString(t *testing.T) {
	expected := "Hello,World"
	newItem := Item{
		FullName: expected,
	}

	if newItem.String() != expected {
		t.Errorf("Test Failed - Item String() error expected %s but recieved %s",
			expected,
			&newItem)
	}
}
