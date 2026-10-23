package coinbase

// APIKeyPermissionsResponse is what a CDP API key may do and the portfolio it is permissioned for
type APIKeyPermissionsResponse struct {
	CanView  bool `json:"can_view"`
	CanTrade bool `json:"can_trade"`
	// CanTransfer is set when the key may deposit and withdraw
	CanTransfer   bool   `json:"can_transfer"`
	PortfolioUUID string `json:"portfolio_uuid"`
	// PortfolioType is DEFAULT, CONSUMER or INTX
	PortfolioType string `json:"portfolio_type"`
}
