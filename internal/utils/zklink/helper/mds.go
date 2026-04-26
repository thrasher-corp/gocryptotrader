package helper

import (
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/internal/utils/zklink/bn256/fr"
)

// mdsMatrices is matrices for improving the efficiency of Poseidon hash.
// see more details in the paper https://eprint.iacr.org/2019/458.pdf page 20.
type mdsMatrices struct {
	// the input mds matrix.
	m Matrix
	// mInv is the inverse of the mds matrix.
	mInv Matrix
	// mHat is the matrix by eliminating the first row and column of the matrix.
	mHat Matrix
	// mHatInv is the inverse of the mHat matrix.
	mHatInv Matrix
	// mPrime is the matrix m' in the paper, and it holds m = m'*m''.
	// mPrime consists of:
	// 1  |  0
	// 0  |  mHat
	mPrime Matrix
	// mDoublePrime is the matrix m'' in the paper, and it holds m = m'*m''.
	// mDoublePrime consists of:
	// m_00  |  v
	// w_hat |  I
	// where M_00 is the first element of the mds matrix,
	// w_hat and v are t-1 length vectors,
	// I is the (t-1)*(t-1) identity matrix.
	mDoublePrime Matrix
}

// SparseMatrix is specifically one of the form of m”.
// This means its first row and column are each dense, and the interior matrix
// (minor to the element in both the row and column) is the identity.
// For simplicity, we omit the identity matrix in m”.
type SparseMatrix struct {
	// WHat is the first column of the M'' matrix, this is a little different with the WHat in the paper because
	// we add M_00 to the beginning of the WHat.
	WHat Vector
	// V contains all but the first element, because it is already included in WHat.
	V Vector
}

func GenMDS(t int) Matrix {
	xVec := make([]*fr.Element, t)
	yVec := make([]*fr.Element, t)

regen:
	// generate x and y value where x[i] != y[i] to allow the values to be inverted, and
	// there are no duplicates in the x vector or y vector, so that
	// the determinant is always non-zero.
	for i := 0; i < t; i++ {
		xVec[i] = NewElement().SetUint64(uint64(i))
		yVec[i] = NewElement().SetUint64(uint64(i + t))
	}

	m := make([][]*fr.Element, t)
	for i := 0; i < t; i++ {
		m[i] = make([]*fr.Element, t)
		for j := 0; j < t; j++ {
			m[i][j] = NewElement().Add(xVec[i], yVec[j])
			m[i][j].Inverse(m[i][j])
		}
	}

	// m must be invertible.
	if !IsInvertible(m) {
		t++
		goto regen
	}

	// m must be symmetric.
	transm := transpose(m)
	if !IsEqual(transm, m) {
		panic("m is not symmetric!")
	}

	return m
}

// derive the mds matrices from m.
func deriveMatrices(m Matrix) (*mdsMatrices, error) {
	mInv, err := Invert(m)
	if err != nil {
		return nil, fmt.Errorf("gen mInv failed, err: %w", err)
	}

	mHat, err := minor(m, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("gen mHat failed, err: %w", err)
	}

	mHatInv, err := Invert(mHat)
	if err != nil {
		return nil, fmt.Errorf("gen mHatInv failed, err: %w", err)
	}

	mPrime := genPrime(m)

	mDoublePrime, err := genDoublePrime(m, mHatInv)
	if err != nil {
		return nil, fmt.Errorf("gen double prime m failed, err: %w", err)
	}

	return &mdsMatrices{m, mInv, mHat, mHatInv, mPrime, mDoublePrime}, nil
}

// generate the matrix m', where m = m'*m”.
func genPrime(m Matrix) Matrix {
	prime := make([][]*fr.Element, row(m))
	prime[0] = append(prime[0], one())
	for i := 1; i < column(m); i++ {
		prime[0] = append(prime[0], zero())
	}

	for i := 1; i < row(m); i++ {
		prime[i] = make([]*fr.Element, column(m))
		prime[i][0] = zero()
		for j := 1; j < column(m); j++ {
			prime[i][j] = m[i][j]
		}
	}
	return prime
}

// generate the matrix m”, where m = m'*m”.
func genDoublePrime(m, mHatInv Matrix) (Matrix, error) {
	w, v := genPreVectors(m)

	wHat, err := LeftMatMul(mHatInv, w)
	if err != nil {
		return nil, fmt.Errorf("compute WHat failed, err: %w", err)
	}

	doublePrime := make([][]*fr.Element, row(m))
	doublePrime[0] = append([]*fr.Element{m[0][0]}, v...)
	for i := 1; i < row(m); i++ {
		doublePrime[i] = make([]*fr.Element, column(m))
		doublePrime[i][0] = wHat[i-1]
		for j := 1; j < column(m); j++ {
			if j == i {
				doublePrime[i][j] = one()
			} else {
				doublePrime[i][j] = zero()
			}
		}
	}

	return doublePrime, nil
}

// generate pre-computed vectors used in the sparse matrix.
func genPreVectors(m Matrix) (Vector, Vector) {
	v := make([]*fr.Element, column(m)-1)
	copy(v, m[0][1:])

	w := make([]*fr.Element, row(m)-1)
	for i := 1; i < row(m); i++ {
		w[i-1] = m[i][0]
	}

	return w, v
}
