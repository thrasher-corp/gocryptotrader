package math_utils

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/huandu/xstrings"
)

var zero = big.NewInt(0)
var one = big.NewInt(1)
var two = big.NewInt(2)

// ECMult Multiplies by m a point on the elliptic curve with equation y^2 = x^3 + alpha*x + beta mod p.
// Assumes the point is given in affine form (x, y) and that 0 < m < order(point).
func ECMult(m *big.Int, point [2]*big.Int, alpha int, p *big.Int) [2]*big.Int {
	if m.Cmp(one) == 0 {
		return point
	}
	//return point
	if big.NewInt(0).Mod(m, two).Cmp(zero) == 0 {
		return ECMult(big.NewInt(0).Quo(m, two), ECDouble(point, alpha, p), alpha, p)
	}
	return ECCAdd(ECMult(big.NewInt(0).Sub(m, one), point, alpha, p), point, p)
}

// ECDouble doubles a point on an elliptic curve with the equation y^2 = x^3 + alpha*x + beta mod p
func ECDouble(point [2]*big.Int, alpha int, p *big.Int) [2]*big.Int {
	// m = div_mod(3 * point[0] * point[0] + alpha, 2 * point[1], p)
	p1 := big.NewInt(3)
	p1.Mul(p1, big.NewInt(0).Mul(point[0], point[0]))
	p1.Add(p1, big.NewInt(int64(alpha)))
	p2 := big.NewInt(0)
	p2.Mul(two, point[1])
	m := DivMod(p1, p2, p)
	// x = (m * m - 2 * point[0]) % p
	x := big.NewInt(0)
	x.Sub(big.NewInt(0).Mul(m, m), big.NewInt(0).Mul(two, point[0]))
	x.Mod(x, p)
	// y = (m * (point[0] - x) - point[1]) % p
	y := big.NewInt(0)
	y.Sub(big.NewInt(0).Mul(m, big.NewInt(0).Sub(point[0], x)), point[1])
	y.Mod(y, p)
	return [2]*big.Int{x, y}
}

// Assumes the point is given in affine form (x, y) and has y != 0.

// ECCAdd Gets two points on an elliptic curve mod p and returns their sum.
// Assumes the points are given in affine form (x, y) and have different x coordinates.
func ECCAdd(point1 [2]*big.Int, point2 [2]*big.Int, p *big.Int) [2]*big.Int {
	// m = div_mod(point1[1] - point2[1], point1[0] - point2[0], p)
	d1 := big.NewInt(0).Sub(point1[1], point2[1])
	d2 := big.NewInt(0).Sub(point1[0], point2[0])
	m := DivMod(d1, d2, p)

	// x = (m * m - point1[0] - point2[0]) % p
	x := big.NewInt(0)
	x.Sub(big.NewInt(0).Mul(m, m), point1[0])
	x.Sub(x, point2[0])
	x.Mod(x, p)

	// y := (m*(point1[0]-x) - point1[1]) % p
	y := big.NewInt(0)
	y.Mul(m, big.NewInt(0).Sub(point1[0], x))
	y.Sub(y, point1[1])
	y.Mod(y, p)

	return [2]*big.Int{x, y}
}

// DivMod Finds a nonnegative integer 0 <= x < p such that (m * x) % p == n
func DivMod(n, m, p *big.Int) *big.Int {
	a, _, c := IGCdex(m, p)
	// (n * a) % p
	if c.Int64() != 1 {
		panic(fmt.Sprintf("expected c to be equal to 1, but found %d", c.Int64()))
	}
	tmp := big.NewInt(0).Mul(n, a)
	return tmp.Mod(tmp, p)
}

/*
IGCdex
Returns x, y, g such that g = x*a + y*b = gcd(a, b).

	>>> from sympy.core.numbers import IGCdex
	>>> IGCdex(2, 3)
	(-1, 1, 1)
	>>> IGCdex(10, 12)
	(-1, 1, 2)

	>>> x, y, g = IGCdex(100, 2004)
	>>> x, y, g
	(-20, 1, 4)
	>>> x*100 + y*2004
	4
*/
/**
from sympy.core.numbers import IGCdex
source code:
   if (not a) and (not b):
	   return (0, 1, 0)

   if not a:
	   return (0, b//abs(b), abs(b))
   if not b:
	   return (a//abs(a), 0, abs(a))

   if a < 0:
	   a, x_sign = -a, -1
   else:
	   x_sign = 1

   if b < 0:
	   b, y_sign = -b, -1
   else:
	   y_sign = 1

   x, y, r, s = 1, 0, 0, 1

   while b:
	   (c, q) = (a % b, a // b)
	   (a, b, r, s, x, y) = (b, c, x - q*r, y - q*s, r, s)

   return (x*x_sign, y*y_sign, a)
*/
func IGCdex(a, b *big.Int) (*big.Int, *big.Int, *big.Int) {
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
func GenerateKRfc6979(msgHash, priKey *big.Int, EC_ORDER *big.Int, seed int) *big.Int {
	msgHash = big.NewInt(0).Set(msgHash) // copy
	bitMod := msgHash.BitLen() % 8
	if bitMod <= 4 && bitMod >= 1 && msgHash.BitLen() > 248 {
		msgHash.Mul(msgHash, big.NewInt(16))
	}
	var extra []byte
	if seed > 0 {
		buf := new(bytes.Buffer)
		var data interface{}
		if seed < 256 {
			data = uint8(seed)
		} else if seed < 65536 {
			data = uint16(seed)
		} else if seed < 4294967296 {
			data = uint32(seed)
		} else {
			data = uint64(seed)
		}
		_ = binary.Write(buf, binary.BigEndian, data)
		extra = buf.Bytes()
	}
	return GenerateSecret(EC_ORDER, priKey, sha256.New, msgHash.Bytes(), extra)
	// return GenerateK(EC_ORDER, priKey, sha256.New(), msgHash.Bytes(), extra)
}

// IntToHex32 Normalize to a 32-byte hex string without 0x prefix.
func IntToHex32(x *big.Int) string {
	str := x.Text(16)
	return xstrings.RightJustify(str, 64, "0")
}
