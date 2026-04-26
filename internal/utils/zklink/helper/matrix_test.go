package helper

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thrasher-corp/gocryptotrader/internal/utils/zklink/bn256/fr"
)

var zeroE = new(fr.Element).SetUint64(0)
var OneE = new(fr.Element).SetUint64(1)
var Two = new(fr.Element).SetUint64(2)
var Three = new(fr.Element).SetUint64(3)
var Four = new(fr.Element).SetUint64(4)
var Five = new(fr.Element).SetUint64(5)
var Six = new(fr.Element).SetUint64(6)
var Seven = new(fr.Element).SetUint64(7)
var Eight = new(fr.Element).SetUint64(8)
var Nine = new(fr.Element).SetUint64(9)

func TestVector(t *testing.T) {
	negTwo := new(fr.Element).Neg(Two)

	sub := []struct {
		v1, v2 Vector
		want   Vector
	}{
		{Vector{OneE, Two}, Vector{OneE, Two}, Vector{zeroE, zeroE}},
		{Vector{OneE, Two}, Vector{zeroE, zeroE}, Vector{OneE, Two}},
		{Vector{Three, Four}, Vector{OneE, Two}, Vector{Two, Two}},
		{Vector{OneE, Two}, Vector{Three, Four}, Vector{negTwo, negTwo}},
	}

	for _, cases := range sub {
		get, err := VecSub(cases.v1, cases.v2)
		assert.NoError(t, err)
		assert.Equal(t, cases.want, get)
	}

	add := []struct {
		v1, v2 Vector
		want   Vector
	}{
		{Vector{OneE, Two}, Vector{OneE, Two}, Vector{Two, Four}},
		{Vector{OneE, Two}, Vector{zeroE, zeroE}, Vector{OneE, Two}},
		{Vector{OneE, Two}, Vector{OneE, negTwo}, Vector{Two, zeroE}},
	}

	for _, cases := range add {
		get, err := VecAdd(cases.v1, cases.v2)
		assert.NoError(t, err)
		assert.Equal(t, cases.want, get)
	}

	scalarmul := []struct {
		scalar *fr.Element
		v      Vector
		want   Vector
	}{
		{zeroE, Vector{OneE, Two}, Vector{zeroE, zeroE}},
		{OneE, Vector{OneE, Two}, Vector{OneE, Two}},
		{Two, Vector{OneE, Two}, Vector{Two, Four}},
	}

	for _, cases := range scalarmul {
		get := ScalarVecMul(cases.scalar, cases.v)
		assert.Equal(t, cases.want, get)
	}

	vecmul := []struct {
		v1, v2 Vector
		want   *fr.Element
	}{
		{Vector{OneE, Two}, Vector{OneE, Two}, Five},
		{Vector{OneE, Two}, Vector{zeroE, zeroE}, zeroE},
		{Vector{OneE, Two}, Vector{negTwo, OneE}, zeroE},
	}

	for _, cases := range vecmul {
		get, err := VecMul(cases.v1, cases.v2)
		assert.NoError(t, err)
		assert.Equal(t, cases.want, get)
	}
}

func TestMatrixScalarMul(t *testing.T) {
	scalarmul := []struct {
		scalar *fr.Element
		m      Matrix
		want   Matrix
	}{
		{zeroE, Matrix{{OneE, Two}, {OneE, Two}}, Matrix{{zeroE, zeroE}, {zeroE, zeroE}}},
		{OneE, Matrix{{OneE, Two}, {OneE, Two}}, Matrix{{OneE, Two}, {OneE, Two}}},
		{Two, Matrix{{OneE, Two}, {Three, Four}}, Matrix{{Two, Four}, {Six, Eight}}},
	}

	for _, cases := range scalarmul {
		get := ScalarMul(cases.scalar, cases.m)
		assert.Equal(t, cases.want, get)
	}
}

func TestIdentity(t *testing.T) {
	get := MakeIdentity(3)
	want := Matrix{{OneE, zeroE, zeroE}, {zeroE, OneE, zeroE}, {zeroE, zeroE, OneE}}
	assert.Equal(t, want, get)
}

func TestMinor(t *testing.T) {
	m := Matrix{{OneE, Two, Three}, {Four, Five, Six}, {Seven, Eight, Nine}}

	testMatrix := []struct {
		i, j int
		want Matrix
	}{
		{0, 0, Matrix{{Five, Six}, {Eight, Nine}}},
		{0, 1, Matrix{{Four, Six}, {Seven, Nine}}},
		{0, 2, Matrix{{Four, Five}, {Seven, Eight}}},
		{1, 0, Matrix{{Two, Three}, {Eight, Nine}}},
		{1, 1, Matrix{{OneE, Three}, {Seven, Nine}}},
		{1, 2, Matrix{{OneE, Two}, {Seven, Eight}}},
		{2, 0, Matrix{{Two, Three}, {Five, Six}}},
		{2, 1, Matrix{{OneE, Three}, {Four, Six}}},
		{2, 2, Matrix{{OneE, Two}, {Four, Five}}},
	}

	for _, cases := range testMatrix {
		get, err := minor(m, cases.i, cases.j)
		assert.NoError(t, err)
		assert.Equal(t, cases.want, get)
	}
}

func TestCopyMatrix(t *testing.T) {
	m := Matrix{{OneE, Two, Three}, {Four, Five, Six}, {Seven, Eight, Nine}}

	testMatrix := []struct {
		start, end int
		want       Matrix
	}{
		{0, 1, Matrix{{OneE, Two, Three}}},
		{0, 2, Matrix{{OneE, Two, Three}, {Four, Five, Six}}},
		{0, 3, Matrix{{OneE, Two, Three}, {Four, Five, Six}, {Seven, Eight, Nine}}},
		{1, 2, Matrix{{Four, Five, Six}}},
		{1, 3, Matrix{{Four, Five, Six}, {Seven, Eight, Nine}}},
		{2, 3, Matrix{{Seven, Eight, Nine}}},
	}

	for _, cases := range testMatrix {
		get := copyMatrixRows(m, cases.start, cases.end)
		assert.Equal(t, cases.want, get)
	}
}

func TestTranspose(t *testing.T) {
	testMatrix := []struct {
		input, want Matrix
	}{
		{Matrix{{OneE, Two}, {Three, Four}}, Matrix{{OneE, Three}, {Two, Four}}},
		{Matrix{{OneE, Two, Three}, {Four, Five, Six}, {Seven, Eight, Nine}}, Matrix{{OneE, Four, Seven}, {Two, Five, Eight}, {Three, Six, Nine}}},
	}

	for _, cases := range testMatrix {
		get := transpose(cases.input)
		assert.Equal(t, cases.want, get)
	}
}

func TestUpperTriangular(t *testing.T) {
	shadow := MakeIdentity(3)
	testMatrix := []struct {
		m, s Matrix
		want bool
	}{
		{Matrix{{Two, Three, Four}, {Four, Five, Six}, {Seven, Eight, Eight}}, shadow, true},
		{Matrix{{OneE, Two, Three}, {Four, Five, Six}, {Seven, Eight, Nine}}, shadow, false},
		{Matrix{{OneE, Two, Three}, {zeroE, Three, Four}, {zeroE, zeroE, Three}}, shadow, true},
		{Matrix{{Two, Three, Four}, {zeroE, Two, Four}, {zeroE, zeroE, OneE}}, shadow, true},
	}

	for _, cases := range testMatrix {
		m, _, err := upperTriangular(cases.m, cases.s)
		assert.NoError(t, err)
		get := isUpperTriangular(m)
		assert.Equal(t, cases.want, get)
	}
}

func TestFindNonzeroE(t *testing.T) {
	vectorSet := []struct {
		k    int
		v    Vector
		want bool
	}{
		{0, Vector{zeroE, OneE, Two, Three}, false},
		{1, Vector{zeroE, OneE, Two, Three}, true},
		{2, Vector{zeroE, OneE, Two, Three}, false},
		{2, Vector{zeroE, zeroE, zeroE, OneE}, true},
		{3, Vector{zeroE, zeroE, zeroE, OneE}, true},
		{3, Vector{zeroE, OneE, Two, Three}, false},
		{4, Vector{zeroE, OneE, Two, Three}, false},
	}

	for _, cases := range vectorSet {
		get := isFirstKZero(cases.v, cases.k)
		assert.Equal(t, cases.want, get)
	}

	nonzeroESet := []struct {
		m    Matrix
		c    int
		want struct {
			e     *fr.Element
			index int
		}
	}{
		{Matrix{{Two, Three, Four}, {Four, Five, Six}, {Seven, Eight, Eight}}, 0, struct {
			e     *fr.Element
			index int
		}{Two, 0}},
		{Matrix{{Two, Three, Four}, {Four, Five, Six}, {Seven, Eight, Eight}}, 1, struct {
			e     *fr.Element
			index int
		}{Three, 0}},
		{Matrix{{Two, Three, Four}, {Four, Five, Six}, {Seven, Eight, Eight}}, 2, struct {
			e     *fr.Element
			index int
		}{Four, 0}},
		{Matrix{{OneE, zeroE, zeroE}, {Two, Three, zeroE}, {Four, Five, zeroE}}, 0, struct {
			e     *fr.Element
			index int
		}{OneE, 0}},
		{Matrix{{OneE, zeroE, zeroE}, {Two, Three, zeroE}, {Four, Five, zeroE}}, 1, struct {
			e     *fr.Element
			index int
		}{Three, 1}},
		{Matrix{{OneE, zeroE, zeroE}, {Two, Three, zeroE}, {Four, Five, zeroE}}, 2, struct {
			e     *fr.Element
			index int
		}{nil, -1}},
	}

	for _, cases := range nonzeroESet {
		gete, geti, err := findNonZero(cases.m, cases.c)
		assert.NoError(t, err)
		if gete != nil && cases.want.e != nil {
			if gete.Cmp(cases.want.e) != 0 || geti != cases.want.index {
				t.Errorf("find non zeroE failed, get element: %v, want element: %v, get index: %d, want index: %d", gete, cases.want.e, geti, cases.want.index)
				return
			}
		} else if gete == nil && cases.want.e == nil {
			if geti != cases.want.index || geti != -1 {
				t.Errorf("find non zeroE failed, get element: %v, want element: %v, get index: %d, want index: %d", gete, cases.want.e, geti, cases.want.index)
				return
			}
		} else {
			t.Errorf("find non zeroE failed, get element: %v, want element: %v, get index: %d, want index: %d", gete, cases.want.e, geti, cases.want.index)
			return
		}
	}
}

func TestMatMul(t *testing.T) {
	// [[1,2,3],[4,5,6],[7,8,9]]*[[2,3,4],[4,5,6],[7,8,8]]
	// =[[31,37,40],[70,85,95],[109,133,148]]
	m00 := new(fr.Element).SetUint64(31)
	m01 := new(fr.Element).SetUint64(37)
	m02 := new(fr.Element).SetUint64(40)
	m10 := new(fr.Element).SetUint64(70)
	m11 := new(fr.Element).SetUint64(85)
	m12 := new(fr.Element).SetUint64(94)
	m20 := new(fr.Element).SetUint64(109)
	m21 := new(fr.Element).SetUint64(133)
	m22 := new(fr.Element).SetUint64(148)

	thirteen := new(fr.Element).SetUint64(13)
	sixteen := new(fr.Element).SetUint64(16)
	eighteen := new(fr.Element).SetUint64(18)

	testMatrix := []struct {
		m1, m2 Matrix
		want   Matrix
	}{
		{Matrix{{zeroE, zeroE}, {zeroE, zeroE}}, Matrix{{OneE, Two}, {OneE, Two}}, Matrix{{zeroE, zeroE}, {zeroE, zeroE}}},
		{Matrix{{OneE, Two}, {Two, Three}}, Matrix{{OneE, Two}, {OneE, zeroE}}, Matrix{{Three, Two}, {Five, Four}}},
		{Matrix{{OneE, Two, Three}, {Four, Five, Six}, {Seven, Eight, Nine}}, Matrix{{Two, Three, Four}, {Four, Five, Six}, {Seven, Eight, Eight}}, Matrix{{m00, m01, m02}, {m10, m11, m12}, {m20, m21, m22}}},
		{Matrix{{OneE, OneE, OneE}, {OneE, OneE, OneE}, {OneE, OneE, OneE}}, Matrix{{Two, Three, Four}, {Four, Five, Six}, {Seven, Eight, Eight}}, Matrix{{thirteen, sixteen, eighteen}, {thirteen, sixteen, eighteen}, {thirteen, sixteen, eighteen}}},
		{Matrix{{zeroE, zeroE, zeroE}, {zeroE, zeroE, zeroE}, {zeroE, zeroE, zeroE}}, Matrix{{Two, Three, Four}, {Four, Five, Six}, {Seven, Eight, Eight}}, Matrix{{zeroE, zeroE, zeroE}, {zeroE, zeroE, zeroE}, {zeroE, zeroE, zeroE}}},
		{Matrix{{OneE, zeroE, zeroE}, {zeroE, OneE, zeroE}, {zeroE, zeroE, OneE}}, Matrix{{Two, Three, Four}, {Four, Five, Six}, {Seven, Eight, Eight}}, Matrix{{Two, Three, Four}, {Four, Five, Six}, {Seven, Eight, Eight}}},
	}

	for _, cases := range testMatrix {
		get, err := MatMul(cases.m1, cases.m2)
		assert.NoError(t, err)
		assert.Equal(t, cases.want, get)
	}

	// [[1,2,3],[4,5,6],[7,8,9]]*[1,1,1]
	// =[6,15,24]
	fifteen := new(fr.Element).SetUint64(15)
	twentyfour := new(fr.Element).SetUint64(24)

	testLeftMul := []struct {
		m    Matrix
		v    Vector
		want Vector
	}{
		{Matrix{{OneE, Two, Three}, {Four, Five, Six}, {Seven, Eight, Nine}}, Vector{zeroE, zeroE, zeroE}, Vector{zeroE, zeroE, zeroE}},
		{Matrix{{OneE, Two, Three}, {Four, Five, Six}, {Seven, Eight, Nine}}, Vector{OneE, zeroE, zeroE}, Vector{OneE, Four, Seven}},
		{Matrix{{OneE, Two, Three}, {Four, Five, Six}, {Seven, Eight, Nine}}, Vector{OneE, OneE, OneE}, Vector{Six, fifteen, twentyfour}},
	}

	for _, cases := range testLeftMul {
		get, err := LeftMatMul(cases.m, cases.v)
		assert.NoError(t, err)
		assert.Equal(t, cases.want, get)
	}

	// [1,1,1]*[[1,2,3],[4,5,6],[7,8,9]]
	// =[12,15,18]
	twelve := new(fr.Element).SetUint64(12)

	testRightMul := []struct {
		v    Vector
		m    Matrix
		want Vector
	}{
		{Vector{zeroE, zeroE, zeroE}, Matrix{{OneE, Two, Three}, {Four, Five, Six}, {Seven, Eight, Nine}}, Vector{zeroE, zeroE, zeroE}},
		{Vector{OneE, zeroE, zeroE}, Matrix{{OneE, Two, Three}, {Four, Five, Six}, {Seven, Eight, Nine}}, Vector{OneE, Two, Three}},
		{Vector{OneE, OneE, OneE}, Matrix{{OneE, Two, Three}, {Four, Five, Six}, {Seven, Eight, Nine}}, Vector{twelve, fifteen, eighteen}},
	}

	for _, cases := range testRightMul {
		get, err := RightMatMul(cases.v, cases.m)
		assert.NoError(t, err)
		assert.Equal(t, cases.want, get)
	}
}

func TestEliminate(t *testing.T) {
	m := Matrix{{Two, Three, Four}, {Four, Five, Six}, {Seven, Eight, Eight}}
	shadow := MakeIdentity(3)

	// result of eliminating the first column.
	// [[2,3,4],[0,-1,-2],[0,-5/2,-6]]
	negoneE := new(fr.Element).Neg(OneE)
	negtwo := new(fr.Element).Neg(Two)
	negFiveDivTwo := new(fr.Element).Neg(Five)
	negFiveDivTwo.Div(negFiveDivTwo, Two)
	negsix := new(fr.Element).Neg(Six)

	// result of eliminating the second column.
	// [[2,3,4],[2/3,0,-2/3],[5/3,0,-8/3]]
	twoDivThree := new(fr.Element).Div(Two, Three)
	negTwoDivThree := new(fr.Element).Neg(twoDivThree)
	fiveDivThree := new(fr.Element).Div(Five, Three)
	negEightDivThree := new(fr.Element).Div(Eight, Three)
	negEightDivThree.Neg(negEightDivThree)

	// result of eliminating the third column.
	// [[2,3,4],[1,1/2,0],[3,2,0]]
	oneEDivTwo := new(fr.Element).Div(OneE, Two)

	testMatrix := []struct {
		c    int
		want Matrix
	}{
		{0, Matrix{{Two, Three, Four}, {zeroE, negoneE, negtwo}, {zeroE, negFiveDivTwo, negsix}}},
		{1, Matrix{{Two, Three, Four}, {twoDivThree, zeroE, negTwoDivThree}, {fiveDivThree, zeroE, negEightDivThree}}},
		{2, Matrix{{Two, Three, Four}, {OneE, oneEDivTwo, zeroE}, {Three, Two, zeroE}}},
	}

	for _, cases := range testMatrix {
		get, _, err := eliminate(m, shadow, cases.c)
		assert.NoError(t, err)
		assert.Equal(t, cases.want, get)
	}
}

func TestReduceToIdentity(t *testing.T) {
	// m=[[1,2,3],[0,3,4],[0,0,3]]
	// m^-1=[[1,-2/3,-1/9],[0,1/3,-4/9],[0,0,1/3]]
	negTwoDivThree := new(fr.Element).Div(Two, Three)
	negTwoDivThree.Neg(negTwoDivThree)
	negoneEDivNine := new(fr.Element).Div(OneE, Nine)
	negoneEDivNine.Neg(negoneEDivNine)
	oneEDivThree := new(fr.Element).Div(OneE, Three)
	negFourDivNine := new(fr.Element).Div(Four, Nine)
	negFourDivNine.Neg(negFourDivNine)

	// m=[[2,3,4],[0,2,4],[0,0,1]]
	// m^-1=[[1/2,-3/4,1],[0,1/2,-2],[0,0,1]]
	oneEDivTwo := new(fr.Element).Div(OneE, Two)
	negThreeDivFour := new(fr.Element).Div(Three, Four)
	negThreeDivFour.Neg(negThreeDivFour)
	negtwo := new(fr.Element).Neg(Two)

	shadow := MakeIdentity(3)

	testMatrix := []struct {
		m    Matrix
		want Matrix
	}{
		{Matrix{{OneE, Two, Three}, {zeroE, Three, Four}, {zeroE, zeroE, Three}}, Matrix{{OneE, negTwoDivThree, negoneEDivNine}, {zeroE, oneEDivThree, negFourDivNine}, {zeroE, zeroE, oneEDivThree}}},
		{Matrix{{Two, Three, Four}, {zeroE, Two, Four}, {zeroE, zeroE, OneE}}, Matrix{{oneEDivTwo, negThreeDivFour, OneE}, {zeroE, oneEDivTwo, negtwo}, {zeroE, zeroE, OneE}}},
	}

	for _, cases := range testMatrix {
		_, get, err := reduceToIdentity(cases.m, shadow)
		assert.NoError(t, err)
		assert.Equal(t, cases.want, get)
	}
}

func TestIsInvertible(t *testing.T) {
	testMatrix := []struct {
		m    Matrix
		want bool
	}{
		{Matrix{{Two, Three, Four}, {Four, Five, Six}, {Seven, Eight, Eight}}, true},
		{Matrix{{OneE, Two, Three}, {zeroE, Three, Four}, {zeroE, zeroE, Three}}, true},
		{Matrix{{Two, Three, Four}, {zeroE, Two, Four}, {zeroE, zeroE, OneE}}, true},
		{Matrix{{OneE, Two, Three}, {Four, Five, Six}, {Seven, Eight, Nine}}, false},
	}

	for _, cases := range testMatrix {
		get := IsInvertible(cases.m)
		assert.Equal(t, cases.want, get)
	}
}

func TestInvert(t *testing.T) {
	// 2*2 m:
	// [1 3]
	// [2 7]
	// m^-1:
	// [7 -3]
	// [-2 1]
	negtwo := new(fr.Element).Neg(Two)
	negthree := new(fr.Element).Neg(Three)

	// 3*3 m:
	// [1 2 3]
	// [0 3 4]
	// [0 0 3]
	// m^-1:
	// [1 -2/3 -1/9]
	// [0 1/3 -4/9]
	// [0 0 1/3]
	negTwoDivThree := new(fr.Element).Div(Two, Three)
	negTwoDivThree.Neg(negTwoDivThree)
	negoneEDivNine := new(fr.Element).Div(OneE, Nine)
	negoneEDivNine.Neg(negoneEDivNine)
	oneEDivThree := new(fr.Element).Div(OneE, Three)
	negFourDivNine := new(fr.Element).Div(Four, Nine)
	negFourDivNine.Neg(negFourDivNine)

	// 3*3 m:
	// [2 3 4]
	// [4 5 6]
	// [7 8 8]
	// m^-1:
	// [-4 4 -1]
	// [5 -6 2]
	// [-3/2 5/2 -1]
	negoneE := new(fr.Element).Neg(OneE)
	negfour := new(fr.Element).Neg(Four)
	negsix := new(fr.Element).Neg(Six)
	negThreeDivTwo := new(fr.Element).Div(Three, Two)
	negThreeDivTwo.Neg(negThreeDivTwo)
	fiveDivTwo := new(fr.Element).Div(Five, Two)

	testMatrix := []struct {
		m    Matrix
		want Matrix
	}{
		{Matrix{{OneE, Three}, {Two, Seven}}, Matrix{{Seven, negthree}, {negtwo, OneE}}},
		{Matrix{{OneE, Two, Three}, {zeroE, Three, Four}, {zeroE, zeroE, Three}}, Matrix{{OneE, negTwoDivThree, negoneEDivNine}, {zeroE, oneEDivThree, negFourDivNine}, {zeroE, zeroE, oneEDivThree}}},
		{Matrix{{Two, Three, Four}, {Four, Five, Six}, {Seven, Eight, Eight}}, Matrix{{negfour, Four, negoneE}, {Five, negsix, Two}, {negThreeDivTwo, fiveDivTwo, negoneE}}},
	}

	for _, cases := range testMatrix {
		res, _ := MatMul(cases.m, cases.want)
		if !IsIdentity(res) {
			t.Error("test cases err")
		}

		get, err := Invert(cases.m)
		assert.NoError(t, err)
		assert.Equal(t, cases.want, get)
	}
}
