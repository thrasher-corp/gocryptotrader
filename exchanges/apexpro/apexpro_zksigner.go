package apexpro

// ZKLinkSigner represents a ZK link signing information
type ZKLinkSigner struct{}

type ZKKeyInfo struct {
	Seeds         []byte
	L2Key         string
	PublicKeyHash []byte
}
