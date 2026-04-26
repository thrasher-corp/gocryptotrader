package zklink

import (
	"encoding/binary"
	"math/big"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/internal/utils/zklink/bn256/fr"
	"golang.org/x/crypto/blake2s"
	"golang.org/x/crypto/chacha20"
)

const (
	rescueStateWidth = 3
	rescueRounds     = 22
	rescueRate       = 2
	rescueCapacity   = 1
	// Total round constants: (1 + 2*rounds) * stateWidth
	rescueNumRC = (1 + 2*rescueRounds) * rescueStateWidth // 135
)

// inv5Exponent = ModInverse(5, p-1) for BN254 Fr modulus p
// Used for the inverse S-box x → x^inv5
var rescueInv5Exp = func() *big.Int {
	n, _ := new(big.Int).SetString(
		"17510594297471420177797124596205820070838691520332827474958563349260646796493", 10)
	return n
}()

var (
	rescueInitOnce sync.Once
	// Round constants, flat array of 135 fr.Element values in Montgomery form
	rescueRC [rescueNumRC]fr.Element
	// MDS matrix (3×3), row-major flat array in Montgomery form
	rescueMDS [rescueStateWidth * rescueStateWidth]fr.Element
)

func initRescue() {
	rescueInitOnce.Do(func() {
		if err := generateRescueRoundConstants(); err != nil {
			panic("zklink rescue round constants: " + err.Error())
		}
		if err := generateRescueMDS(); err != nil {
			panic("zklink rescue MDS: " + err.Error())
		}
	})
}

// generateRescueRoundConstants fills rescueRC using rejection sampling from blake2s.
// Only values strictly between 0 and the field modulus are accepted.
func generateRescueRoundConstants() error {
	tag := []byte("Rescue_f")
	modulus := fr.Modulus()
	count := 0
	nonce := uint32(0)
	nonceBytes := make([]byte, 4)

	for count < rescueNumRC {
		binary.BigEndian.PutUint32(nonceBytes, nonce)
		h, err := blake2s.New256(nil)
		if err != nil {
			return err
		}
		h.Write(tag)
		h.Write(GH_FIRST_BLOCK)
		h.Write(nonceBytes)
		hashData := h.Sum(nil)

		candidate := new(big.Int).SetBytes(hashData)
		if candidate.Sign() > 0 && candidate.Cmp(modulus) < 0 {
			rescueRC[count].SetBigInt(candidate)
			count++
		}
		nonce++
	}
	return nil
}

// generateRescueMDS fills rescueMDS using ChaCha20 seeded from "ResM0003"+GH_FIRST_BLOCK.
func generateRescueMDS() error {
	tag := []byte("ResM0003")
	h, err := blake2s.New256(nil)
	if err != nil {
		return err
	}
	h.Write(tag)
	h.Write(GH_FIRST_BLOCK)
	key := h.Sum(nil) // 32-byte key derived from the tag

	// Zero nonce, matching the Rust ChaCha20 RNG seeding convention
	nonce := make([]byte, 12)
	rng, err := chacha20.NewUnauthenticatedCipher(key, nonce)
	if err != nil {
		return err
	}

	for {
		x := make([]fr.Element, rescueStateWidth)
		y := make([]fr.Element, rescueStateWidth)
		for i := range x {
			rescueGenFrInto(&x[i], rng)
			rescueGenFrInto(&y[i], rng)
		}

		// Reject if any x[i] == x[j], y[i] == y[j], or x[i] == y[j]
		if !rescueAllUnique(x, y) {
			continue
		}

		// Build Cauchy matrix M[i][j] = 1 / (x[i] - y[j])
		for i := range rescueStateWidth {
			for j := range rescueStateWidth {
				var diff fr.Element
				diff.Sub(&x[i], &y[j])
				diff.Inverse(&diff)
				rescueMDS[i*rescueStateWidth+j] = diff
			}
		}
		return nil
	}
}

// rescueGenFrInto fills dst with a random Fr element from the ChaCha20 stream.
// Reads 8 bytes, interprets as uint64, converts to Fr (mod p via SetBigInt).
func rescueGenFrInto(dst *fr.Element, rng *chacha20.Cipher) {
	buf := make([]byte, 8)
	rng.XORKeyStream(buf, buf)
	v := binary.BigEndian.Uint64(buf)
	dst.SetBigInt(new(big.Int).SetUint64(v))
}

// rescueAllUnique returns true if all elements in x are distinct, all in y are
// distinct, and x and y share no common elements.
func rescueAllUnique(x, y []fr.Element) bool {
	for i := range x {
		for j := i + 1; j < len(x); j++ {
			if x[i].Equal(&x[j]) {
				return false
			}
		}
	}
	for i := range y {
		for j := i + 1; j < len(y); j++ {
			if y[i].Equal(&y[j]) {
				return false
			}
		}
	}
	for i := range x {
		for j := range y {
			if x[i].Equal(&y[j]) {
				return false
			}
		}
	}
	return true
}

// rescuePow5 computes x^5 mod p in Montgomery form, avoiding aliasing.
func rescuePow5(x *fr.Element) fr.Element {
	orig := *x
	var x2, x4, res fr.Element
	x2.Square(&orig)
	x4.Square(&x2)
	res.Mul(&x4, &orig)
	return res
}

// rescuePowInv5 computes x^inv5 mod p using the precomputed exponent.
func rescuePowInv5(x *fr.Element) fr.Element {
	var res fr.Element
	res.Exp(*x, rescueInv5Exp)
	return res
}

// rescueApplyMDS multiplies state by the MDS matrix in-place.
func rescueApplyMDS(state *[rescueStateWidth]fr.Element) {
	var out [rescueStateWidth]fr.Element
	for i := range rescueStateWidth {
		out[i].SetZero()
		for j := range rescueStateWidth {
			var tmp fr.Element
			tmp.Mul(&rescueMDS[i*rescueStateWidth+j], &state[j])
			out[i].Add(&out[i], &tmp)
		}
	}
	*state = out
}

// RescuePermute applies the full Rescue permutation to the 3-element state.
// All elements are treated as Montgomery-form field values.
func RescuePermute(state *[rescueStateWidth]fr.Element) {
	initRescue()

	// Initial round constant addition
	for i := range rescueStateWidth {
		state[i].Add(&state[i], &rescueRC[i])
	}

	for r := range rescueRounds {
		// Forward S-box: x → x^5
		for i := range rescueStateWidth {
			v := rescuePow5(&state[i])
			state[i] = v
		}
		rescueApplyMDS(state)
		offset := (2*r + 1) * rescueStateWidth
		for i := range rescueStateWidth {
			state[i].Add(&state[i], &rescueRC[offset+i])
		}

		// Inverse S-box: x → x^inv5
		for i := range rescueStateWidth {
			v := rescuePowInv5(&state[i])
			state[i] = v
		}
		rescueApplyMDS(state)
		offset = (2*r + 2) * rescueStateWidth
		for i := range rescueStateWidth {
			state[i].Add(&state[i], &rescueRC[offset+i])
		}
	}
}

// RescueHash computes the Rescue sponge hash over an arbitrary number of field elements.
// Inputs are absorbed in blocks of rate=2, and state[0] is returned as the digest.
// All element values are assumed to be in Montgomery form.
func RescueHash(inputs []fr.Element) *fr.Element {
	initRescue()
	var state [rescueStateWidth]fr.Element

	for i := 0; i < len(inputs); i += rescueRate {
		state[0].Add(&state[0], &inputs[i])
		if i+1 < len(inputs) {
			state[1].Add(&state[1], &inputs[i+1])
		}
		RescuePermute(&state)
	}

	// Handle empty input
	if len(inputs) == 0 {
		RescuePermute(&state)
	}

	result := new(fr.Element)
	result.Set(&state[0])
	return result
}

// RescueHashBigInt hashes a *big.Int by splitting it into 31-byte Fr field elements.
// Returns the hash as an fr.Element in Montgomery form.
func RescueHashBigInt(msg *big.Int) *fr.Element {
	initRescue()
	elems := bigIntToFrElements(msg)
	return RescueHash(elems)
}

// bigIntToFrElements splits a big.Int into 31-byte (248-bit) Fr field elements.
// The integer is zero-padded to a multiple of 31 bytes (big-endian), then chunked.
func bigIntToFrElements(n *big.Int) []fr.Element {
	if n == nil || n.Sign() == 0 {
		return []fr.Element{{}}
	}

	const chunkSize = 31
	b := n.Bytes() // big-endian, no leading zeros
	if rem := len(b) % chunkSize; rem != 0 {
		padding := make([]byte, chunkSize-rem)
		b = append(padding, b...)
	}

	elems := make([]fr.Element, len(b)/chunkSize)
	for i := range elems {
		elems[i].SetBytes(b[i*chunkSize : (i+1)*chunkSize])
	}
	return elems
}

// BatchInvert returns a new slice with every element inverted using Montgomery's trick.
func BatchInvert(a []*fr.Element) []*fr.Element {
	res := make([]*fr.Element, len(a))
	if len(a) == 0 {
		return res
	}
	zeroes := make([]bool, len(a))
	accumulator := new(fr.Element).SetOne()

	for i := range len(a) {
		if a[i].IsZero() {
			zeroes[i] = true
			continue
		}
		res[i] = new(fr.Element).Set(accumulator)
		accumulator.Mul(accumulator, a[i])
	}

	accumulator.Inverse(accumulator)

	for i := len(a) - 1; i >= 0; i-- {
		if zeroes[i] {
			res[i] = new(fr.Element)
			continue
		}
		res[i].Mul(res[i], accumulator)
		accumulator.Mul(accumulator, a[i])
	}
	return res
}

// PowerSBox represents a power S-box with precomputed inverse exponent.
type PowerSBox struct {
	Power *big.Int
	Inv   uint64
}

// QuinticSBox represents a quintic S-box marker.
type QuinticSBox struct {
	Marker *big.Int
}

// BatchInversion computes modular inverses in-place using Montgomery's trick.
func BatchInversion(v []*big.Int, modulus *big.Int) {
	if len(v) == 0 {
		return
	}
	prod := make([]*big.Int, len(v))
	tmp := big.NewInt(1)
	zero := big.NewInt(0)
	j := 0

	for _, g := range v {
		if g.Cmp(zero) != 0 {
			tmp = new(big.Int).Mod(new(big.Int).Mul(tmp, g), modulus)
			prod[j] = new(big.Int).Set(tmp)
			j++
		}
	}
	tmp.ModInverse(tmp, modulus)

	for i := j - 1; i >= 0; i-- {
		g := v[i]
		if g.Cmp(zero) != 0 {
			newTmp := new(big.Int).Mod(new(big.Int).Mul(tmp, g), modulus)
			g.Mod(new(big.Int).Mul(tmp, prod[i]), modulus)
			tmp.Set(newTmp)
		}
	}
}
