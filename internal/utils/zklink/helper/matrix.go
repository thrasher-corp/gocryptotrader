package helper

import (
	"errors"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/internal/utils/zklink/bn256/fr"
)

type Matrix [][]*fr.Element

type Vector []*fr.Element

// return the column numbers of the matrix.
func column(m Matrix) int {
	if len(m) > 0 {
		length := len(m[0])
		for i := range m {
			if len(m[i]) != length {
				panic("m is not matrix!")
			}
		}
		return length
	} else {
		return 0
	}
}

// return the row numbers of the matrix.
func row(m Matrix) int {
	return len(m)
}

// for 0 <= i < row, 0 <= j < column, compute M_ij*scalar.
func ScalarMul(scalar *fr.Element, m Matrix) Matrix {
	res := make([][]*fr.Element, len(m))
	for i := range m {
		res[i] = make([]*fr.Element, len(m[i]))
		for j := range m[i] {
			res[i][j] = NewElement().Mul(scalar, m[i][j])
		}
	}

	return res
}

// for 0 <= i < length, compute v_i*scalar.
func ScalarVecMul(scalar *fr.Element, v Vector) Vector {
	res := make([]*fr.Element, len(v))

	for i := range v {
		res[i] = NewElement().Mul(scalar, v[i])
	}

	return res
}

func VecAdd(a, b Vector) (Vector, error) {
	if len(a) != len(b) {
		return nil, errors.New("length err: cannot compute vector add")
	}

	res := make([]*fr.Element, len(a))
	for i := range a {
		res[i] = NewElement().Add(a[i], b[i])
	}

	return res, nil
}

func VecSub(a, b Vector) (Vector, error) {
	if len(a) != len(b) {
		return nil, errors.New("length err: cannot compute vector sub")
	}

	res := make([]*fr.Element, len(a))
	for i := range a {
		res[i] = NewElement().Sub(a[i], b[i])
	}

	return res, nil
}

// compute the product between two vectors.
func VecMul(a, b Vector) (*fr.Element, error) {
	res := NewElement()
	if len(a) != len(b) {
		return res, errors.New("length err: cannot compute vector mul!")
	}

	for i := range a {
		tmp := NewElement().Mul(a[i], b[i])
		res.Add(res, tmp)
	}

	return res, nil
}

func IsVecEqual(a, b Vector) bool {
	if len(a) != len(b) {
		return false
	}

	for i := 0; i < len(a); i++ {
		if a[i].Cmp(b[i]) != 0 {
			return false
		}
	}

	return true
}

// IsInvertible if delta(m)≠0, m is invertible.
// so we can transform m to the upper triangular matrix,
// and if all upper diagonal elements are not zero, then m is invertible.
func IsInvertible(m Matrix) bool {
	// need to copy m.
	tmp := copyMatrixRows(m, 0, row(m))
	if !IsSquareMatrix(tmp) {
		return false
	}

	shadow := MakeIdentity(row(tmp))
	upper, _, err := upperTriangular(tmp, shadow)
	if err != nil {
		panic(err)
	}

	for i := range row(tmp) {
		if upper[i][i].Cmp(zero()) == 0 {
			return false
		}
	}

	return true
}

// compute the product between two matrices.
func MatMul(a, b Matrix) (Matrix, error) {
	if row(a) != column(b) {
		return nil, errors.New("cannot compute the result")
	}

	transb := transpose(b)

	var err error
	res := make([][]*fr.Element, row(a))
	for i := range row(a) {
		res[i] = make([]*fr.Element, column(b))
		for j := range column(b) {
			res[i][j], err = VecMul(a[i], transb[j])
			if err != nil {
				return nil, fmt.Errorf("vec mul err: %w", err)
			}
		}
	}

	return res, nil
}

// LeftMatMul left Matrix multiplication, denote by M*V, where M is the matrix, and V is the vector.
func LeftMatMul(m Matrix, v Vector) (Vector, error) {
	if !IsSquareMatrix(m) {
		panic("matrix is not square!")
	}

	if row(m) != len(v) {
		return nil, errors.New("length err: cannot compute matrix multiplication with the vector")
	}

	res := make([]*fr.Element, len(v))
	var err error
	for i := range len(v) {
		res[i], err = VecMul(m[i], v)
		if err != nil {
			return nil, fmt.Errorf("vector mul err: %w", err)
		}
	}

	return res, nil
}

// RightMatMul right Matrix multiplication, denote by V*M, where V is the vector, and M is the matrix.
func RightMatMul(v Vector, m Matrix) (Vector, error) {
	if !IsSquareMatrix(m) {
		return nil, errors.New("matrix is not square")
	}

	if row(m) != len(v) {
		return nil, errors.New("length err: cannot compute matrix multiplication with the vector")
	}

	transm := transpose(m)
	res := make([]*fr.Element, len(v))
	var err error
	for i := range v {
		res[i], err = VecMul(transm[i], v)
		if err != nil {
			return nil, fmt.Errorf("vector mul err: %w", err)
		}
	}

	return res, nil
}

// swap rows and columns of the matrix.
func transpose(m Matrix) Matrix {
	res := make([][]*fr.Element, column(m))

	for j := range column(m) {
		res[j] = make([]*fr.Element, len(m))
		for i := range m {
			res[j][i] = m[i][j]
		}
	}

	return res
}

// IsSquareMatrix the square matrix is a t*t matrix.
func IsSquareMatrix(m Matrix) bool {
	return row(m) == column(m)
}

// MakeIdentity make t*t identity matrix.
func MakeIdentity(t int) Matrix {
	res := make([][]*fr.Element, t)

	for i := 0; i < t; i++ {
		res[i] = make([]*fr.Element, t)
		for j := range t {
			if i == j {
				res[i][j] = one()
			} else {
				res[i][j] = zero()
			}
		}
	}

	return res
}

// IsIdentity determine if a matrix is identity.
func IsIdentity(m Matrix) bool {
	for i := range row(m) {
		for j := range column(m) {
			if ((i == j) && m[i][j].Cmp(one()) != 0) || ((i != j) && (m[i][j].Cmp(zero()) != 0)) {
				return false
			}
		}
	}

	return true
}

// IsEqual compares matrixes
func IsEqual(a, b Matrix) bool {
	if row(a) != row(b) || column(a) != column(b) {
		return false
	}

	for i := range row(a) {
		for j := range column(a) {
			if a[i][j].Cmp(b[i][j]) != 0 {
				return false
			}
		}
	}
	return true
}

// remove i-th row and j-th column of the matrix.
func minor(m Matrix, rowIndex, columnIndex int) (Matrix, error) {
	if !IsSquareMatrix(m) {
		return nil, errors.New("matrix is not square!")
	}

	res := make([][]*fr.Element, row(m)-1)

	for i := range row(m) {
		if i < rowIndex {
			for j := range column(m) {
				if j != columnIndex {
					res[i] = append(res[i], m[i][j])
				}
			}
		} else if i > rowIndex {
			for j := range column(m) {
				if j != columnIndex {
					res[i-1] = append(res[i-1], m[i][j])
				}
			}
		}
	}

	return res, nil
}

// determine if the first k elements are zero.
func isFirstKZero(v Vector, k int) bool {
	if k == 0 && v[0].Cmp(zero()) == 0 {
		return false
	}

	for i := range k {
		if v[i].Cmp(zero()) != 0 {
			return false
		}
	}
	return true
}

// find the first non-zero element in the given column.
func findNonZero(m Matrix, index int) (pivot *fr.Element, pivotIndex int, err error) {
	pivotIndex = -1

	if index > column(m) {
		return NewElement(), -1, errors.New("index out of range!")
	}

	for i := range row(m) {
		if m[i][index].Cmp(zero()) != 0 {
			pivot = m[i][index]
			pivotIndex = i
			break
		}
	}
	return
}

// assume matrix is partially reduced to upper triangular.
func eliminate(m, shadow Matrix, columnIndex int) (Matrix, Matrix, error) {
	pivot, pivotIndex, err := findNonZero(m, columnIndex)
	if err != nil || pivotIndex == -1 {
		return nil, nil, fmt.Errorf("cannot find non-zero element: %w", err)
	}

	pivotInv := NewElement().Inverse(pivot)

	for i := range row(m) {
		if i == pivotIndex {
			continue
		}

		if m[i][columnIndex].Cmp(zero()) != 0 {
			factor := NewElement().Mul(m[i][columnIndex], pivotInv)

			scalarPivot := ScalarVecMul(factor, m[pivotIndex])

			m[i], err = VecSub(m[i], scalarPivot)
			if err != nil {
				return nil, nil, fmt.Errorf("matrix m eliminate failed, vec sub err: %w", err)
			}

			shadowPivot := shadow[pivotIndex]

			scalarShadowPivot := ScalarVecMul(factor, shadowPivot)

			shadow[i], err = VecSub(shadow[i], scalarShadowPivot)
			if err != nil {
				return nil, nil, fmt.Errorf("matrix shadow eliminate failed, vec sub err: %w", err)
			}
		}
	}

	return m, shadow, nil
}

// copy rows between start index and end index.
func copyMatrixRows(m Matrix, startIndex, endIndex int) Matrix {
	if startIndex >= endIndex {
		panic("start index should be less than end index!")
	}

	res := make([][]*fr.Element, endIndex-startIndex)

	for i := range endIndex - startIndex {
		res[i] = make([]*fr.Element, column(m))
		copy(res[i], m[i+startIndex])
	}

	return res
}

// reverse rows of the matrix.
func reverseRows(m Matrix) Matrix {
	res := make([][]*fr.Element, row(m))

	for i := range row(m) {
		res[i] = make([]*fr.Element, column(m))
		copy(res[i], m[row(m)-i-1])
	}

	return res
}

// determine if numbers of zero elements equals to n.
func zeroNums(v Vector, n int) bool {
	count := 0
	for i := range len(v) {
		if v[i].Cmp(zero()) != 0 {
			break
		}
		count++
	}
	return count == n
}

// determine if a matrix is upper triangular.
func isUpperTriangular(m Matrix) bool {
	for i := range row(m) {
		if !zeroNums(m[i], i) {
			return false
		}
	}

	return true
}

// transform a square matrix to upper triangular matrix.
func upperTriangular(m, shadow Matrix) (Matrix, Matrix, error) {
	if !IsSquareMatrix(m) {
		return nil, nil, errors.New("matrix is not square")
	}

	curr := copyMatrixRows(m, 0, row(m))
	currShadow := copyMatrixRows(shadow, 0, row(shadow))
	result := make([][]*fr.Element, row(m))
	shadowResult := make([][]*fr.Element, row(shadow))
	c := 0
	var err error
	for row(curr) > 1 {
		result[c] = make([]*fr.Element, column(m))
		shadowResult[c] = make([]*fr.Element, column(shadow))
		curr, currShadow, err = eliminate(curr, currShadow, c)
		if err != nil {
			return nil, nil, fmt.Errorf("matrix eliminate err: %w", err)
		}

		copy(result[c], curr[0])
		copy(shadowResult[c], currShadow[0])

		c++

		curr = copyMatrixRows(curr, 1, row(curr))
		currShadow = copyMatrixRows(currShadow, 1, row(currShadow))
	}
	result[c] = make([]*fr.Element, column(m))
	shadowResult[c] = make([]*fr.Element, column(shadow))
	copy(result[c], curr[0])
	copy(shadowResult[c], currShadow[0])

	return result, shadowResult, nil
}

// reduce a upper triangular matrix to identity matrix.
func reduceToIdentity(m, shadow Matrix) (Matrix, Matrix, error) {
	var err error
	result := make([][]*fr.Element, row(m))
	shadowResult := make([][]*fr.Element, row(shadow))
	for i := range row(m) {
		result[i] = make([]*fr.Element, column(m))
		shadowResult[i] = make([]*fr.Element, column(shadow))
		indexi := row(m) - i - 1

		factor := m[indexi][indexi]
		if factor.Cmp(zero()) == 0 {
			return nil, nil, errors.New("cannot compute the result!")
		}

		factorInv := NewElement().Inverse(factor)

		norm := ScalarVecMul(factorInv, m[indexi])

		shadowNorm := ScalarVecMul(factorInv, shadow[indexi])

		for j := range i {
			indexj := row(m) - j - 1
			val := norm[indexj]

			scalarVal := ScalarVecMul(val, result[j])
			scalarShadow := ScalarVecMul(val, shadowResult[j])

			norm, err = VecSub(norm, scalarVal)
			if err != nil {
				return nil, nil, fmt.Errorf("reduces to identity matrix failed, err: %w", err)
			}

			shadowNorm, err = VecSub(shadowNorm, scalarShadow)
			if err != nil {
				return nil, nil, fmt.Errorf("reduces to identity matrix failed, err: %w", err)
			}
		}
		copy(result[i], norm)
		copy(shadowResult[i], shadowNorm)
	}

	result = reverseRows(result)
	shadowResult = reverseRows(shadowResult)

	return result, shadowResult, nil
}

// Invert uses Gaussian elimination to invert a matrix.
// A|I -> I|A^-1.
func Invert(m Matrix) (Matrix, error) {
	if !IsInvertible(m) {
		return nil, fmt.Errorf("the matrix is not invertible")
	}

	shadow := MakeIdentity(row(m))

	up, upShadow, err := upperTriangular(m, shadow)
	if err != nil {
		return nil, fmt.Errorf("transform to upper triangular matrix failed, err: %w", err)
	}

	if !isUpperTriangular(up) {
		return nil, fmt.Errorf("the matrix should be upper triangular before reducing")
	}

	// reduce m to identity, so shadow matrix transforms to the inverse of m.
	reduce, reducedShadow, err := reduceToIdentity(up, upShadow)
	if err != nil {
		return nil, fmt.Errorf("reduce to identity failed, err: %w", err)
	}

	if !IsIdentity(reduce) {
		return nil, errors.New("reduces failed, the result is not the identity matrix")
	}

	return reducedShadow, nil
}
