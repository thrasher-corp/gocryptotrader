package deposit

// Address holds a deposit address
type Address struct {
	Address string
	Tag     string // Represents either a tag or memo
	Chain   string
}
