package coinbase

// Product group types, which product lists and the transaction summary take as a product type; the public product
// list holds no product of either
const (
	ProductTypeOptionGroup = "OPTION_GROUP"
	ProductTypeFutureGroup = "FUTURE_GROUP"
)

// ProductRequest selects the product GetProduct returns
type ProductRequest struct {
	// ProductID names the product; an equity is named by its product ID, not its display ticker
	ProductID string
	// GetTradabilityStatus sets a spot product's ViewOnly when the account cannot trade it
	GetTradabilityStatus bool
}

// bestBidAskResponse holds the books GetBestBidAsk returns
type bestBidAskResponse struct {
	PriceBooks []PriceBook `json:"pricebooks"`
}
