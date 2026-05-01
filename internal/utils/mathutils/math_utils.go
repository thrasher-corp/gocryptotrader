package mathutils

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math/big"
)

var (
	zero = big.NewInt(0)
	one  = big.NewInt(1)
	two  = big.NewInt(2)
)

// ECMult Multiplies by m a point on the elliptic curve with equation y^2 = x^3 + alpha*x + beta mod p.
// Assumes the point is given in affine form (x, y) and that 0 < m < order(point).
func ECMult(m *big.Int, point [2]*big.Int, alpha int, p *big.Int) [2]*big.Int {
	if m.Cmp(one) == 0 {
		return point
	}
	if big.NewInt(0).Mod(m, two).Cmp(zero) == 0 {
		return ECMult(big.NewInt(0).Quo(m, two), ECDouble(point, alpha, p), alpha, p)
	}
	return ECCAdd(ECMult(big.NewInt(0).Sub(m, one), point, alpha, p), point, p)
}

// ECDouble doubles a point on an elliptic curve with the equation y^2 = x^3 + alpha*x + beta mod p
func ECDouble(point [2]*big.Int, alpha int, p *big.Int) [2]*big.Int {
	// computes m = div_mod(3 * point[0] * point[0] + alpha, 2 * point[1], p)
	p1 := big.NewInt(3)
	p1.Mul(p1, big.NewInt(0).Mul(point[0], point[0]))
	p1.Add(p1, big.NewInt(int64(alpha)))
	p2 := big.NewInt(0)
	p2.Mul(two, point[1])
	m := DivMod(p1, p2, p)
	// computes x = (m * m - 2 * point[0]) % p
	x := big.NewInt(0)
	x.Sub(big.NewInt(0).Mul(m, m), big.NewInt(0).Mul(two, point[0]))
	x.Mod(x, p)
	// computes y = (m * (point[0] - x) - point[1]) % p
	y := big.NewInt(0)
	y.Sub(big.NewInt(0).Mul(m, big.NewInt(0).Sub(point[0], x)), point[1])
	y.Mod(y, p)
	return [2]*big.Int{x, y}
}

// Assumes the point is given in affine form (x, y) and has y != 0.

// ECCAdd gets two points on an elliptic curve mod p and returns their sum.
// Assumes the points are given in affine form (x, y) and have different x coordinates.
func ECCAdd(point1, point2 [2]*big.Int, p *big.Int) [2]*big.Int {
	// computes m = div_mod(point1[1] - point2[1], point1[0] - point2[0], p)
	d1 := big.NewInt(0).Sub(point1[1], point2[1])
	d2 := big.NewInt(0).Sub(point1[0], point2[0])
	m := DivMod(d1, d2, p)

	// computes x = (m * m - point1[0] - point2[0]) % p
	x := big.NewInt(0)
	x.Sub(big.NewInt(0).Mul(m, m), point1[0])
	x.Sub(x, point2[0])
	x.Mod(x, p)

	// computes y := (m*(point1[0]-x) - point1[1]) % p
	y := big.NewInt(0)
	y.Mul(m, big.NewInt(0).Sub(point1[0], x))
	y.Sub(y, point1[1])
	y.Mod(y, p)

	return [2]*big.Int{x, y}
}

// DivMod finds a nonnegative integer 0 <= x < p such that (m * x) % p == n
func DivMod(n, m, p *big.Int) *big.Int {
	a, _, c := IGCdex(m, p)
	// (n * a) % p
	if c.Int64() != 1 {
		panic(fmt.Sprintf("expected c to be equal to 1, but found %d", c.Int64()))
	}
	tmp := big.NewInt(0).Mul(n, a)
	return tmp.Mod(tmp, p)
}

func IGCdex(a, b *big.Int) (x, y, g *big.Int) {
	if a.Cmp(zero) == 0 && b.Cmp(zero) == 0 {
		return big.NewInt(0), big.NewInt(1), big.NewInt(0)
	}
	if a.Cmp(zero) == 0 {
		return big.NewInt(0), big.NewInt(0).Quo(b, big.NewInt(0).Abs(b)), big.NewInt(0).Abs(b)
	}
	if b.Cmp(zero) == 0 {
		return big.NewInt(0).Quo(a, big.NewInt(0).Abs(a)), big.NewInt(0), big.NewInt(0).Abs(a)
	}
	xSign := big.NewInt(1)
	ySign := big.NewInt(1)
	if a.Cmp(zero) == -1 {
		a, xSign = a.Neg(a), big.NewInt(-1)
	}
	if b.Cmp(zero) == -1 {
		b, ySign = b.Neg(b), big.NewInt(-1)
	}
	x, y, r, s := big.NewInt(1), big.NewInt(0), big.NewInt(0), big.NewInt(1)
	for b.Cmp(zero) > 0 {
		c, q := big.NewInt(0).Mod(a, b), big.NewInt(0).Quo(a, b)
		a, b, r, s, x, y = b, c, big.NewInt(0).Sub(x, big.NewInt(0).Mul(q, r)), big.NewInt(0).Sub(y, big.NewInt(0).Mul(big.NewInt(0).Neg(q), s)), r, s
	}
	return x.Mul(x, xSign), y.Mul(y, ySign), a
}

// GenerateKRfc6979 generates a deterministic 'k' value as specified in RFC 6979 for use in ECDSA signatures.
// This method ensures that the same private key and message will always produce the same 'k', providing
// a defense against certain attacks that exploit poor randomness in 'k' values.
func GenerateKRfc6979(msgHash, priKey, ecOrder *big.Int, seed int) *big.Int {
	msgHash = big.NewInt(0).Set(msgHash) // copy
	bitMod := msgHash.BitLen() % 8
	if bitMod <= 4 && bitMod >= 1 && msgHash.BitLen() > 248 {
		msgHash.Mul(msgHash, big.NewInt(16))
	}
	var extra []byte
	if seed > 0 {
		buf := new(bytes.Buffer)
		var data interface{}
		switch {
		case seed < 256:
			data = uint8(seed)
		case seed < 65536:
			data = uint16(seed)
		case seed < 4294967296:
			data = uint32(seed)
		default:
			data = uint64(seed)
		}
		_ = binary.Write(buf, binary.BigEndian, data)
		extra = buf.Bytes()
	}
	return GenerateSecret(ecOrder, priKey, sha256.New, msgHash.Bytes(), extra)
}
