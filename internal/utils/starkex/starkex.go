package starkex

import (
	"crypto/elliptic"
	"errors"
	"fmt"
	"math/big"
	"strings"

	path "github.com/thrasher-corp/gocryptotrader/internal/testing/utils"
	"github.com/thrasher-corp/gocryptotrader/internal/utils/hash"
	math_utils "github.com/thrasher-corp/gocryptotrader/internal/utils/math"
)

// Error declarations.
var (
	ErrInvalidPrivateKey         = errors.New("invalid private key")
	ErrInvalidPublicKey          = errors.New("invalid public key")
	ErrFailedToGenerateSignature = errors.New("failed to generate signature")
	ErrInvalidHashPayload        = errors.New("invalid hash payload")
)

// StarkConfig represents a stark configuration
type StarkConfig struct {
	*elliptic.CurveParams
	EcGenX           *big.Int
	EcGenY           *big.Int
	MinusShiftPointX *big.Int
	MinusShiftPointY *big.Int
	Max              *big.Int
	Alpha            *big.Int
	ConstantPoints   [][2]*big.Int
	PedersenHash     func(...string) string
}

var one = big.NewInt(1)

const defaultPedersenConfigsPath = "internal/utils/hash/elliptic_curve_config/"

// NewStarkExConfig returns a elliptic curve configuration given the name of the elliptic curve config
func NewStarkExConfig() (*StarkConfig, error) {
	rootPath, err := path.RootPathFromCWD()
	if err != nil {
		return nil, err
	}
	pedersenConfig, err := hash.LoadPedersenConfig(rootPath + "/" + defaultPedersenConfigsPath + strings.ToLower("starkEx") + ".json")
	if err != nil {
		return nil, err
	}
	starkCurve := &StarkConfig{
		CurveParams: &elliptic.CurveParams{
			P:       pedersenConfig.FieldPrime,
			N:       pedersenConfig.EcOrder,
			B:       pedersenConfig.BETA,
			Gx:      pedersenConfig.ConstantPoints[1][0],
			Gy:      pedersenConfig.ConstantPoints[1][1],
			BitSize: 252,
		},
		EcGenX:         pedersenConfig.ConstantPoints[1][0],
		EcGenY:         pedersenConfig.ConstantPoints[1][1],
		Alpha:          big.NewInt(int64(pedersenConfig.ALPHA)),
		ConstantPoints: pedersenConfig.ConstantPoints,
		PedersenHash:   pedersenConfig.PedersenHash,
	}
	starkCurve.MinusShiftPointX, _ = new(big.Int).SetString("2089986280348253421170679821480865132823066470938446095505822317253594081284", 10) // MINUS_SHIFT_POINT = (SHIFT_POINT[0], FIELD_PRIME - SHIFT_POINT[1])
	starkCurve.MinusShiftPointY, _ = new(big.Int).SetString("1904571459125470836673916673895659690812401348070794621786009710606664325495", 10)
	starkCurve.Max, _ = new(big.Int).SetString("3618502788666131106986593281521497120414687020801267626233049500247285301248", 10) // 2 ** 251
	return starkCurve, nil
}

// Sign generates a signature out using the users private key and signable order params.
func (sfg *StarkConfig) Sign(sgn Signable, starkPrivateKey string, starkPublicKey, starkPublicKeyYCoordinate string) (*big.Int, *big.Int, error) {
	pHash, err := sgn.GetPedersenHash(sfg.PedersenHash)
	if err != nil {
		return nil, nil, err
	}
	priKey, okay := big.NewInt(0).SetString(starkPrivateKey, 0)
	if !okay {
		return nil, nil, fmt.Errorf("%w, %v", ErrInvalidPrivateKey, starkPrivateKey)
	}
	msgHash, okay := new(big.Int).SetString(pHash, 0)
	if !okay {
		return nil, nil, ErrInvalidHashPayload
	}
	r, s, err := sfg.SignECDSA(msgHash, priKey)
	if err != nil {
		return nil, nil, err
	}
	publicKey, ok := big.NewInt(0).SetString(starkPublicKey, 0)
	if !ok {
		return nil, nil, fmt.Errorf("%w, invalid stark public key x coordinat", ErrInvalidPublicKey)
	}
	publicKeyYCoordinate, ok := big.NewInt(0).SetString(starkPublicKeyYCoordinate, 0)
	if !ok {
		publicKeyYCoordinate = sfg.GetYCoordinate(publicKey)
		if publicKeyYCoordinate.Cmp(big.NewInt(0)) == 0 {
			return nil, nil, fmt.Errorf("%w, invalid stark public key x coordinat", ErrInvalidPublicKey)
		}
	}
	ok = sfg.Verify(msgHash, r, s, [2]*big.Int{publicKey, publicKeyYCoordinate})
	if !ok {
		return nil, nil, ErrFailedToGenerateSignature
	}
	return r, s, nil
}

// GetYCoordinate generates the y-coordinate of starkEx Public key from the x coordinate
func (sc StarkConfig) GetYCoordinate(starkKeyXCoordinate *big.Int) *big.Int {
	x := starkKeyXCoordinate
	xpow3 := new(big.Int).Exp(x, big.NewInt(3), nil)
	alphaXPlusB := new(big.Int).Add(new(big.Int).Mul(sc.Alpha, x), sc.B)
	agg := new(big.Int).Mod(new(big.Int).Add(xpow3, alphaXPlusB), sc.P)
	return new(big.Int).ModSqrt(agg, sc.P)
}

// InvModCurveSize calculates the inverse modulus of a given big integer 'x' with respect to the StarkCurve 'sc'.
func (sc StarkConfig) InvModCurveSize(x *big.Int) *big.Int {
	return math_utils.DivMod(one, x, sc.N)
}

// Sign calculates the signature of a message using the StarkCurve algorithm.
func (sc StarkConfig) SignECDSA(msgHash, privKey *big.Int, seed ...*big.Int) (*big.Int, *big.Int, error) {
	if msgHash == nil {
		return nil, nil, fmt.Errorf("nil msgHash")
	}
	if privKey == nil {
		return nil, nil, fmt.Errorf("nil privKey")
	}
	if msgHash.Cmp(big.NewInt(0)) != 1 || msgHash.Cmp(sc.Max) != -1 {
		return nil, nil, fmt.Errorf("invalid bit length")
	}
	inSeed := big.NewInt(0)
	if len(seed) == 1 {
		inSeed = seed[0]
	}
	nBit := big.NewInt(0).Exp(big.NewInt(2), N_ELEMENT_BITS_ECDSA, nil)
	for {
		k := math_utils.GenerateKRfc6979(msgHash, privKey, sc.N, int(inSeed.Int64()))
		// In case r is rejected k shall be generated with new seed
		if inSeed.Int64() == 0 {
			inSeed = big.NewInt(1)
		} else {
			inSeed = inSeed.Add(inSeed, big.NewInt(1))
		}
		x := math_utils.ECMult(k, [2]*big.Int{sc.EcGenX, sc.EcGenY}, int(sc.Alpha.Int64()), sc.P)[0]
		r := big.NewInt(0).Set(x)
		// DIFF: in classic ECDSA, we take int(x) % n.
		if r.Cmp(big.NewInt(0)) != 1 || r.Cmp(sc.Max) != -1 {
			// Bad value. This fails with negligible probability.
			continue
		}
		agg := new(big.Int).Mul(r, privKey)
		agg = agg.Add(agg, msgHash)

		if new(big.Int).Mod(agg, sc.N).Cmp(big.NewInt(0)) == 0 {
			continue
		}

		w := math_utils.DivMod(k, agg, sc.N)
		// if w.Cmp(big.NewInt(0)) != 1 || w.Cmp(sc.Max) != -1 {
		if !(w.Cmp(one) > 0 && w.Cmp(nBit) < 0) {
			continue
		}

		s := sc.InvModCurveSize(w)
		return r, s, nil
	}
}

// Computes m * point + shift_point using the same steps like the AIR and throws an exception if
// and only if the AIR errors.
//
// (ref: https://github.com/starkware-libs/cairo-lang/blob/master/src/starkware/crypto/starkware/crypto/signature/signature.py)
func (sc StarkConfig) MimicEcMultAir(mout, x1, y1, x2, y2 *big.Int) (x *big.Int, y *big.Int, err error) {
	m := new(big.Int).Set(mout)
	if m.Cmp(big.NewInt(0)) != 1 || m.Cmp(sc.Max) != -1 {
		return x, y, fmt.Errorf("too many bits %v", m.BitLen())
	}

	psx := x2
	psy := y2
	for i := 0; i < 251; i++ {
		if psx == x1 {
			return x, y, fmt.Errorf("xs are the same")
		}
		if m.Bit(0) == 1 {
			point1 := math_utils.ECCAdd([2]*big.Int{psx, psy}, [2]*big.Int{x1, y1}, sc.P)
			psx, psy = point1[0], point1[1]
		}
		point := math_utils.ECDouble([2]*big.Int{x1, y1}, int(sc.Alpha.Int64()), sc.P)
		x1, y1 = point[0], point[1]
		m = m.Rsh(m, 1)
	}
	if m.Cmp(big.NewInt(0)) != 0 {
		return psx, psy, fmt.Errorf("m doesn't equal zero")
	}
	return psx, psy, nil
}

// Verifies an ECDSA signature
func (sc *StarkConfig) Verify(msgHash *big.Int, r *big.Int, s *big.Int, publicKey [2]*big.Int) bool {
	calc := func(pubX, pubY *big.Int) *big.Int {
		publicKeyPow2 := new(big.Int).Exp(pubY, big.NewInt(2), nil)
		publicKeyPow3 := new(big.Int).Exp(pubX, big.NewInt(3), nil)
		alphaPublicKey := new(big.Int).Mul(sc.Alpha, pubX)
		aggr := new(big.Int).Add(publicKeyPow3, new(big.Int).Add(alphaPublicKey, sc.B))
		sub := new(big.Int).Sub(publicKeyPow2, aggr)
		mod := new(big.Int).Mod(sub, sc.P)
		if mod.Cmp(big.NewInt(0)) != 0 {
			return nil
		}
		zGx, zGy, err := sc.MimicEcMultAir(msgHash, sc.Gx, sc.Gy, sc.MinusShiftPointX, sc.MinusShiftPointY)
		if err != nil {
			return nil
		}
		rQx, rQy, err := sc.MimicEcMultAir(r, pubX, pubY, sc.ConstantPoints[0][0], sc.ConstantPoints[0][1])
		if err != nil {
			return nil
		}
		eccAddzqp := math_utils.ECCAdd([2]*big.Int{zGx, zGy}, [2]*big.Int{rQx, rQy}, sc.P)
		w := sc.InvModCurveSize(s)
		wBx, wBy, err := sc.MimicEcMultAir(w, eccAddzqp[0], eccAddzqp[1], sc.ConstantPoints[0][0], sc.ConstantPoints[0][1])
		if err != nil {
			return nil
		}
		return math_utils.ECCAdd([2]*big.Int{wBx, wBy}, [2]*big.Int{sc.MinusShiftPointX, sc.MinusShiftPointY}, sc.P)[0]
	}

	return r.Cmp(
		calc(publicKey[0], publicKey[1])) == 0 ||
		r.Cmp(
			calc(
				publicKey[0],
				big.NewInt(0).Sub(big.NewInt(0), publicKey[1]))) == 0
}
