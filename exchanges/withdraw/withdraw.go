package withdraw

func Valid(request interface{})(bool, error) {
	switch request.(type) {
	case *FiatRequest:
		return ValidateFiat(request.(*FiatRequest))
	case *CryptoRequest:
		return ValidateCrypto(request.(*CryptoRequest))
	default:
		return false, nil
	}
}

func ValidateFiat(request *FiatRequest) (bool, error) {
	if request == nil {
		return false, nil
	}
	return false, nil
}

func ValidateCrypto(request *CryptoRequest) (bool, error) {
	if request == nil {
		return false, nil
	}

	return false, nil
}


/*

type FiatRequest struct {
	GenericInfo
	// FIAT related information
	BankAccountName   string
	BankAccountNumber string
	BankName          string
	BankAddress       string
	BankCity          string
	BankCountry       string
	BankPostalCode    string
	BSB               string
	SwiftCode         string
	IBAN              string
	BankCode          float64
	IsExpressWire     bool
	// Intermediary bank information
	RequiresIntermediaryBank      bool
	IntermediaryBankAccountNumber float64
	IntermediaryBankName          string
	IntermediaryBankAddress       string
	IntermediaryBankCity          string
	IntermediaryBankCountry       string
	IntermediaryBankPostalCode    string
	IntermediarySwiftCode         string
	IntermediaryBankCode          float64
	IntermediaryIBAN              string
	WireCurrency                  string
}

 */