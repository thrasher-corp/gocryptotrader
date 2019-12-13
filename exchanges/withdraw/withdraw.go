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
	return false, nil
}

func ValidateCrypto(request *CryptoRequest) (bool, error) {
	return false, nil
}