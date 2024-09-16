package starkex

import (
	"encoding/hex"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
)

// FactToCondition Generate the condition, signed as part of a conditional transfer.
func FactToCondition(factRegistryAddress string, fact string) *big.Int {
	data := strings.TrimPrefix(factRegistryAddress, "0x") + fact
	hexBytes, _ := hex.DecodeString(data)
	// int(Web3.keccak(data).hex(), 16) & BIT_MASK_250
	hash := crypto.Keccak256Hash(hexBytes)
	fst := hash.Big()
	fst.And(fst, BIT_MASK_250)
	return fst
}
