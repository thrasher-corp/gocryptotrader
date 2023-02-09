package subsystem

// String implements stringer interface
func (s SynchronisationType) String() string {
	switch s {
	case Orderbook:
		return "ORDERBOOK"
	case Ticker:
		return "TICKER"
	default:
		return ""
	}
}

// String implements stringer interface
func (s ProtocolType) String() string {
	return string(s)
}
