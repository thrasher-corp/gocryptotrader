package solsha3

import (
	"errors"

	"golang.org/x/crypto/sha3"
)

var errInvalidDataType = errors.New("invalid data types")

// solsha3 solidity sha3
func solsha3(types []string, values ...any) []byte {
	b := [][]byte{}
	for i, typ := range types {
		data, err := pack(typ, values[i], false)
		if err != nil {
			return nil
		}
		b = append(b, data)
	}

	hash := sha3.NewLegacyKeccak256()
	var bs []byte
	for _, bi := range b {
		bs = append(bs, bi...)
	}
	hash.Write(bs)
	return hash.Sum(nil)
}

// SoliditySHA3 computes the KECCAK-256 hash of the given input.
func SoliditySHA3(data ...any) ([]byte, error) {
	types, ok := data[0].([]string)
	if !ok {
		return nil, errInvalidDataType
	}
	rest := data[1:]
	if len(rest) == len(types) {
		return solsha3(types, data[1:]...), nil
	}
	iface, ok := data[1].([]any)
	if ok {
		return solsha3(types, iface...), nil
	}
	return nil, errInvalidDataType
}
