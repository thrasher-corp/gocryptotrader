package v0

// Exchange contains a sub-section of exchange config
type Exchange struct {
	AvailablePairs            string      `json:"availablePairs,omitempty"`
	EnabledPairs              string      `json:"enabledPairs,omitempty"`
	PairsLastUpdated          int64       `json:"pairsLastUpdated,omitempty"`
	ConfigCurrencyPairFormat  *PairFormat `json:"configCurrencyPairFormat,omitempty"`
	RequestCurrencyPairFormat *PairFormat `json:"requestCurrencyPairFormat,omitempty"`
}

// PairFormat contains pair formatting config
type PairFormat struct {
	Uppercase bool   `json:"uppercase"`
	Delimiter string `json:"delimiter,omitempty"`
	Separator string `json:"separator,omitempty"`
}
