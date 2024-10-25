package starkex

import (
	"encoding/hex"
	"math/big"
)

// AppendSignatures combines the r and s components of an ECDSA signature into a single signature string.
func AppendSignatures(r, s *big.Int) string {
	rBytes := r.Bytes()
	sBytes := s.Bytes()

	for i := len(rBytes); i < 32; i++ {
		rBytes = append([]byte{byte(0)}, rBytes...)
	}
	for i := len(sBytes); i < 32; i++ {
		sBytes = append([]byte{byte(0)}, sBytes...)
	}
	bytes := append(rBytes, sBytes...)
	return hex.EncodeToString(bytes)
}
