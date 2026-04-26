package zklink

import (
	"crypto/sha256"
	"errors"
	"math/big"

	"github.com/thrasher-corp/gocryptotrader/internal/utils/zklink/bn256/fr"
	"github.com/thrasher-corp/gocryptotrader/internal/utils/zklink/bn256/twistededwards"
	"golang.org/x/crypto/blake2s"
)

// ZKLinkSigner holds the key material for ZKLink Schnorr signing on BN254 twisted Edwards.
type ZKLinkSigner struct {
	privateKeyBig *big.Int
	privateKey    fr.Element
	publicKey     twistededwards.Point
	pubKeyHash    [20]byte
}

// NewZKLinkSignerFromSeeds derives a ZKLinkSigner from EIP-191 seeds bytes.
//
// Algorithm:
//  1. SHA256(seeds) → 32 bytes
//  2. Reduce mod BN254 twisted Edwards curve order → privKey
//  3. privKey × G → pubKey
//  4. Blake2s(pack(pubKey))[:20] → pubKeyHash
func NewZKLinkSignerFromSeeds(seeds []byte) (*ZKLinkSigner, error) {
	if len(seeds) == 0 {
		return nil, errors.New("seeds cannot be empty")
	}

	// SHA256 of the seeds bytes
	hash := sha256.Sum256(seeds)

	// Reduce mod curve order
	curve := twistededwards.GetEdwardsCurve()
	privBig := new(big.Int).SetBytes(hash[:])
	privBig.Mod(privBig, &curve.Order)
	if privBig.Sign() == 0 {
		return nil, errors.New("derived private key is zero; try different seeds")
	}

	// Convert to fr.Element in regular (non-Montgomery) form for ScalarMul
	var privElem fr.Element
	privElem.SetBigInt(privBig)
	privElem.FromMont()

	// pubKey = privKey × G
	var pubKey twistededwards.Point
	pubKey.ScalarMul(&curve.Base, privElem)

	signer := &ZKLinkSigner{
		privateKeyBig: privBig,
		privateKey:    privElem,
		publicKey:     pubKey,
	}

	// pubKeyHash = Blake2s(packed pubkey)[:20]
	packed := signer.PublicKeyBytes()
	h, err := blake2s.New256(nil)
	if err != nil {
		return nil, err
	}
	h.Write(packed[:])
	digest := h.Sum(nil)
	copy(signer.pubKeyHash[:], digest[:20])

	return signer, nil
}

// PublicKeyBytes returns the 32-byte compressed representation of the public key.
// The Y coordinate is encoded big-endian; bit 255 (MSB of byte 0) is set if X is odd.
func (z *ZKLinkSigner) PublicKeyBytes() [32]byte {
	yBytes := z.publicKey.Y.Bytes() // 32 bytes, big-endian, regular form
	var packed [32]byte
	copy(packed[:], yBytes)

	xBytes := z.publicKey.X.Bytes()
	if xBytes[31]&1 == 1 {
		packed[0] |= 0x80
	}
	return packed
}

// PubKeyHash returns the 20-byte Blake2s hash of the packed public key.
func (z *ZKLinkSigner) PubKeyHash() [20]byte {
	return z.pubKeyHash
}

// Sign computes a Schnorr signature over msg (a raw transaction big.Int) and returns
// a 64-byte signature: pack(R)[32] || s[32].
//
// Algorithm:
//  1. msgHash = RescueHashBigInt(msg)
//  2. k = SHA256(privKey_bytes || msgHash_bytes) mod curve_order  (deterministic nonce)
//  3. R = k × G
//  4. e = RescueHash([R.X, pubKey.X, msgHash]) mod curve_order
//  5. s = (k + e × privKey) mod curve_order
func (z *ZKLinkSigner) Sign(msg *big.Int) ([64]byte, error) {
	// Step 1: hash the transaction message to a single field element
	msgHash := RescueHashBigInt(msg)

	// Step 2: deterministic nonce
	privBytes := make([]byte, 32)
	z.privateKeyBig.FillBytes(privBytes)
	msgHashBytes := msgHash.Bytes() // 32 bytes, regular form

	h := sha256.New()
	h.Write(privBytes)
	h.Write(msgHashBytes)
	nonceBytes := h.Sum(nil)

	curve := twistededwards.GetEdwardsCurve()
	k := new(big.Int).SetBytes(nonceBytes)
	k.Mod(k, &curve.Order)
	if k.Sign() == 0 {
		return [64]byte{}, errors.New("derived nonce is zero; adjust message")
	}

	// Step 3: R = k × G
	var kElem fr.Element
	kElem.SetBigInt(k)
	kElem.FromMont()

	var R twistededwards.Point
	R.ScalarMul(&curve.Base, kElem)

	// Step 4: e = RescueHash([R.X, pubKey.X, msgHash])
	// All three are fr.Element values in Montgomery form — consistent for hash arithmetic.
	eFr := RescueHash([]fr.Element{R.X, z.publicKey.X, *msgHash})

	var eBig big.Int
	eFr.ToBigIntRegular(&eBig)
	eScalar := new(big.Int).Mod(&eBig, &curve.Order)

	// Step 5: s = (k + e * privKey) mod order
	s := new(big.Int).Mul(eScalar, z.privateKeyBig)
	s.Add(s, k)
	s.Mod(s, &curve.Order)

	// Encode output: pack(R) || s, each zero-padded to 32 bytes
	var out [64]byte
	rPacked := packEdPoint(R)
	copy(out[:32], rPacked[:])
	sBytes := s.Bytes()
	copy(out[64-len(sBytes):], sBytes) // right-align in 32 bytes

	return out, nil
}

// packEdPoint encodes a twisted Edwards point in compressed form.
// Bytes 0–31 hold the Y coordinate (big-endian); bit 255 signals parity of X.
func packEdPoint(p twistededwards.Point) [32]byte {
	yBytes := p.Y.Bytes()
	var packed [32]byte
	copy(packed[:], yBytes)
	xBytes := p.X.Bytes()
	if xBytes[31]&1 == 1 {
		packed[0] |= 0x80
	}
	return packed
}
