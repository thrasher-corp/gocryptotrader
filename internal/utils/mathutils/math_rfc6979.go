package mathutils

import (
	"bytes"
	"crypto/hmac"
	"hash"
	"math/big"
)

// rfc6979 implemented in Golang.
// copy from  https://raw.githubusercontent.com/codahale/rfc6979/master/rfc6979.go
/*
Package rfc6979 is an implementation of RFC 6979's deterministic DSA.

	Such signatures are compatible with standard Digital Signature Algorithm
	(DSA) and Elliptic Curve Digital Signature Algorithm (ECDSA) digital
	signatures and can be processed with unmodified verifiers, which need not be
	aware of the procedure described therein.  Deterministic signatures retain
	the cryptographic security features associated with digital signatures but
	can be more easily implemented in various environments, since they do not
	need access to a source of high-quality randomness.

(https://tools.ietf.org/html/rfc6979)

Provides functions similar to crypto/dsa and crypto/ecdsa.
*/

// Mac returns an HMAC of the given key and message.
func Mac(alg func() hash.Hash, k, m, buf []byte) []byte {
	h := hmac.New(alg, k)
	h.Write(m)
	return h.Sum(buf[:0])
}

// Bits2Int converts a sequence of bits to a non-negative integer as per RFC 6979 Section 2.3.2.
// See https://tools.ietf.org/html/rfc6979#section-2.3.2
func Bits2Int(in []byte, qlen int) *big.Int {
	vlen := len(in) * 8
	v := new(big.Int).SetBytes(in)
	if vlen > qlen {
		v = new(big.Int).Rsh(v, uint(vlen-qlen)) //nolint:gosec // vlen > qlen is guaranteed by the preceding if condition
	}
	return v
}

// Int2Octets converts an integer to a fixed-length sequence of octets as per RFC 6979 Section 2.3.3.
// See https://tools.ietf.org/html/rfc6979#section-2.3.3
func Int2Octets(v *big.Int, rolen int) []byte {
	out := v.Bytes()

	// pad with zeros if it's too short
	if len(out) < rolen {
		out2 := make([]byte, rolen)
		copy(out2[rolen-len(out):], out)
		return out2
	}

	// drop most significant bytes if it's too long
	if len(out) > rolen {
		out2 := make([]byte, rolen)
		copy(out2, out[len(out)-rolen:])
		return out2
	}

	return out
}

// Bits2Octets converts a sequence of bits to a sequence of octets as per RFC 6979 Section 2.3.4.
// See https://tools.ietf.org/html/rfc6979#section-2.3.4
func Bits2Octets(in []byte, q *big.Int, qlen, rolen int) []byte {
	z1 := Bits2Int(in, qlen)
	z2 := new(big.Int).Sub(z1, q)
	if z2.Sign() < 0 {
		return Int2Octets(z1, rolen)
	}
	return Int2Octets(z2, rolen)
}

// GenerateSecret generates a deterministic secret value k suitable for use in ECDSA signing,
// as per RFC 6979 Section 3.2. See https://tools.ietf.org/html/rfc6979#section-3.2
func GenerateSecret(q, x *big.Int, alg func() hash.Hash, hashInput, extraEntropy []byte) *big.Int {
	qlen := q.BitLen()
	holen := alg().Size()
	rolen := (qlen + 7) >> 3
	bx := append(Int2Octets(x, rolen), Bits2Octets(hashInput, q, qlen, rolen)...)
	// extra_entropy - extra added data in binary form as per section-3.6 of rfc6979
	if len(extraEntropy) > 0 {
		bx = append(bx, extraEntropy...)
	}

	// Step B
	v := bytes.Repeat([]byte{0x01}, holen)

	// Step C
	k := make([]byte, holen)

	// Step D
	k = Mac(alg, k, append(append(v, 0x00), bx...), k)

	// Step E
	v = Mac(alg, k, v, v)

	// Step F
	k = Mac(alg, k, append(append(v, 0x01), bx...), k)

	// Step G
	v = Mac(alg, k, v, v)

	// Step H
	for {
		// Step H1
		var t []byte

		// Step H2
		for len(t) < qlen/8 {
			v = Mac(alg, k, v, v)
			t = append(t, v...)
		}

		// Step H3
		secret := Bits2Int(t, qlen)
		if secret.Cmp(one) >= 0 && secret.Cmp(q) < 0 {
			return secret
		}
		k = Mac(alg, k, append(v, 0x00), k)
		v = Mac(alg, k, v, v)
	}
}
