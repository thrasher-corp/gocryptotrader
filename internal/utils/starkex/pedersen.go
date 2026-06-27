package starkex

// Note: currently some methods implementations here are directly copied from the github.com/yaune/starkex repository, and will be removed/update and tested

import (
	"math/big"
	"os"

	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	math_utils "github.com/thrasher-corp/gocryptotrader/internal/utils/mathutils"
)

// PedersenCfg represents a pedersen hash configuration options
type PedersenCfg struct {
	Comment        string        `json:"_comment"`
	FieldPrime     *big.Int      `json:"FIELD_PRIME"`
	FieldGen       int           `json:"FIELD_GEN"`
	EcOrder        *big.Int      `json:"EC_ORDER"`
	ALPHA          int           `json:"ALPHA"`
	BETA           *big.Int      `json:"BETA"`
	ConstantPoints [][2]*big.Int `json:"CONSTANT_POINTS"`
}

// LoadPedersenConfig loads a pedersen configuration from a json file.
func LoadPedersenConfig(path string) (*PedersenCfg, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var resp *PedersenCfg
	return resp, json.Unmarshal(file, &resp)
}

// PedersenHash hashed the
func (cfg *PedersenCfg) PedersenHash(str ...string) string {
	NElementBitsHash := cfg.FieldPrime.BitLen()
	point := cfg.ConstantPoints[0]
	for i, s := range str {
		x, ok := big.NewInt(0).SetString(s, 0)
		if !ok {
			return ""
		}
		pointList := cfg.ConstantPoints[2+i*NElementBitsHash : 2+(i+1)*NElementBitsHash]
		n := big.NewInt(0)
		for _, pt := range pointList {
			n.And(x, big.NewInt(1))
			if n.Cmp(big.NewInt(0)) > 0 {
				point = math_utils.ECCAdd(point, pt, cfg.FieldPrime)
			}
			x = x.Rsh(x, 1)
		}
	}
	return point[0].String()
}
