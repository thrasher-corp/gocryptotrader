package hash

import "math/big"

// PedersenCfg represents a pedersen configuration options.
type PedersenCfg struct {
	Comment        string        `json:"_comment"`
	FieldPrime     *big.Int      `json:"FIELD_PRIME"`
	FieldGen       int           `json:"FIELD_GEN"`
	EcOrder        *big.Int      `json:"EC_ORDER"`
	ALPHA          int           `json:"ALPHA"`
	BETA           *big.Int      `json:"BETA"`
	ConstantPoints [][2]*big.Int `json:"CONSTANT_POINTS"`
}
