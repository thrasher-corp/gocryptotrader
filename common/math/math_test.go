package math

import (
	"math"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCalculateFee(t *testing.T) {
	t.Parallel()
	originalInput := float64(1)
	fee := float64(1)
	if expectedOutput, actualResult := 0.01, CalculateFee(originalInput, fee); expectedOutput != actualResult {
		t.Errorf(
			"Expected '%f'. Actual '%f'.", expectedOutput, actualResult)
	}
}

func TestCalculateAmountWithFee(t *testing.T) {
	t.Parallel()
	originalInput := float64(1)
	fee := float64(1)
	if actualResult, expectedOutput := CalculateAmountWithFee(originalInput, fee), 1.01; expectedOutput != actualResult {
		t.Errorf(
			"Expected '%f'. Actual '%f'.", expectedOutput, actualResult)
	}
}

func TestPercentageChange(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 3.3333333333333335, PercentageChange(9000, 9300))
	assert.Equal(t, -3.225806451612903, PercentageChange(9300, 9000))
	assert.True(t, math.IsNaN(PercentageChange(0, 0)))
	assert.Equal(t, 0.0, PercentageChange(1, 1))
	assert.Equal(t, 0.0, PercentageChange(-1, -1))
	assert.True(t, math.IsInf(PercentageChange(0, 1), 1))
	assert.Equal(t, -100., PercentageChange(1, 0))
}

func TestPercentageDifference(t *testing.T) {
	t.Parallel()
	require.Equal(t, 196.03960396039605, PercentageDifference(1, 100))
	require.Equal(t, 196.03960396039605, PercentageDifference(100, 1))
	require.Equal(t, 0.13605442176870758, PercentageDifference(1.469, 1.471))
	require.Equal(t, 0.13605442176870758, PercentageDifference(1.471, 1.469))
	require.Equal(t, 0.0, PercentageDifference(1.0, 1.0))
	require.True(t, math.IsNaN(PercentageDifference(0.0, 0.0)))
}

// 1000000000	         0.2215 ns/op	       0 B/op	       0 allocs/op
func BenchmarkPercentageDifference(b *testing.B) {
	for b.Loop() {
		PercentageDifference(1.469, 1.471)
	}
}

func TestPercentageDifferenceDecimal(t *testing.T) {
	t.Parallel()
	require.Equal(t, "196.03960396039604", PercentageDifferenceDecimal(decimal.NewFromFloat(1), decimal.NewFromFloat(100)).String())
	require.Equal(t, "196.03960396039604", PercentageDifferenceDecimal(decimal.NewFromFloat(100), decimal.NewFromFloat(1)).String())
	require.Equal(t, "0.13605442176871", PercentageDifferenceDecimal(decimal.NewFromFloat(1.469), decimal.NewFromFloat(1.471)).String())
	require.Equal(t, "0.13605442176871", PercentageDifferenceDecimal(decimal.NewFromFloat(1.471), decimal.NewFromFloat(1.469)).String())
	require.Equal(t, "0", PercentageDifferenceDecimal(decimal.NewFromFloat(1.0), decimal.NewFromFloat(1.0)).String())
	require.Equal(t, "0", PercentageDifferenceDecimal(decimal.Zero, decimal.Zero).String())
}

// 1585596	       751.8 ns/op	     792 B/op	      27 allocs/op
func BenchmarkDecimalPercentageDifference(b *testing.B) {
	d1, d2 := decimal.NewFromFloat(1.469), decimal.NewFromFloat(1.471)
	for b.Loop() {
		PercentageDifferenceDecimal(d1, d2)
	}
}

func TestCalculateNetProfit(t *testing.T) {
	t.Parallel()
	amount := float64(5)
	priceThen := float64(1)
	priceNow := float64(10)
	costs := float64(1)
	actualResult := CalculateNetProfit(amount, priceThen, priceNow, costs)
	if expectedOutput := float64(44); expectedOutput != actualResult {
		t.Errorf(
			"Expected '%f'. Actual '%f'.", expectedOutput, actualResult)
	}
}

func TestRoundFloat(t *testing.T) {
	t.Parallel()
	// mapping of input vs expected result : map[precision]map[testedValue]expectedOutput
	testTableValues := map[int]map[float64]float64{
		0: {
			2.23456789:  2,
			-2.23456789: -2,
		},
		1: {
			2.23456789:  2.2,
			-2.23456789: -2.2,
		},
		2: {
			2.23456789:  2.23,
			-2.23456789: -2.23,
		},
		4: {
			2.23456789:  2.2346,
			-2.23456789: -2.2346,
		},
		8: {
			2.23456781:  2.23456781,
			-2.23456781: -2.23456781,
		},
	}

	for precision, values := range testTableValues {
		for testInput, expectedOutput := range values {
			actualOutput := RoundFloat(testInput, precision)
			if actualOutput != expectedOutput {
				t.Errorf("RoundFloat Expected '%v'. Actual '%v' on precision %d",
					expectedOutput, actualOutput, precision)
			}
		}
	}
}

func TestSortinoRatio(t *testing.T) {
	t.Parallel()
	rfr := 0.001
	figures := []float64{0.10, 0.04, 0.15, -0.05, 0.20, -0.02, 0.08, -0.06, 0.13, 0.23}
	avg, err := ArithmeticMean(figures)
	if err != nil {
		t.Error(err)
	}
	_, err = SortinoRatio(nil, rfr, avg)
	assert.ErrorIs(t, err, errZeroValue)

	var r float64
	r, err = SortinoRatio(figures, rfr, avg)
	if err != nil {
		t.Error(err)
	}
	if r != 3.0377875479459906 {
		t.Errorf("expected 3.0377875479459906, received %v", r)
	}
	avg, err = FinancialGeometricMean(figures)
	if err != nil {
		t.Error(err)
	}
	r, err = SortinoRatio(figures, rfr, avg)
	if err != nil {
		t.Error(err)
	}
	if r != 2.8712802265603243 {
		t.Errorf("expected 2.525203164136098, received %v", r)
	}

	// this follows and matches the example calculation from
	// https://www.wallstreetmojo.com/sortino-ratio/
	example := []float64{
		0.1,
		0.12,
		0.07,
		-0.03,
		0.08,
		-0.04,
		0.15,
		0.2,
		0.12,
		0.06,
		-0.03,
		0.02,
	}
	avg, err = ArithmeticMean(example)
	if err != nil {
		t.Error(err)
	}
	r, err = SortinoRatio(example, 0.06, avg)
	if err != nil {
		t.Error(err)
	}
	if rr := math.Round(r*10) / 10; rr != 0.2 {
		t.Errorf("expected 0.2, received %v", rr)
	}
}

func TestInformationRatio(t *testing.T) {
	t.Parallel()
	figures := []float64{0.0665, 0.0283, 0.0911, 0.0008, -0.0203, -0.0978, 0.0164, -0.0537, 0.078, 0.0032, 0.0249, 0}
	comparisonFigures := []float64{0.0216, 0.0048, 0.036, 0.0303, 0.0043, -0.0694, 0.0179, -0.0918, 0.0787, 0.0297, 0.003, 0}
	avg, err := ArithmeticMean(figures)
	if err != nil {
		t.Error(err)
	}
	if avg != 0.01145 {
		t.Error(avg)
	}
	var avgComparison float64
	avgComparison, err = ArithmeticMean(comparisonFigures)
	if err != nil {
		t.Error(err)
	}
	if avgComparison != 0.005425 {
		t.Error(avgComparison)
	}

	eachDiff := make([]float64, len(figures))
	for i := range figures {
		eachDiff[i] = figures[i] - comparisonFigures[i]
	}
	stdDev, err := PopulationStandardDeviation(eachDiff)
	if err != nil {
		t.Error(err)
	}
	if stdDev != 0.028992588851865803 {
		t.Error(stdDev)
	}
	information := (avg - avgComparison) / stdDev
	if information != 0.20781172839666107 {
		t.Errorf("expected %v received %v", 0.20781172839666107, information)
	}
	var information2 float64
	information2, err = InformationRatio(figures, comparisonFigures, avg, avgComparison)
	if err != nil {
		t.Error(err)
	}
	if information != information2 {
		t.Error(information2)
	}

	_, err = InformationRatio(figures, []float64{1}, avg, avgComparison)
	assert.ErrorIs(t, err, errInformationBadLength)
}

func TestCalmarRatio(t *testing.T) {
	t.Parallel()
	_, err := CalmarRatio(0, 0, 0, 0)
	assert.ErrorIs(t, err, errCalmarHighest)

	var ratio float64
	ratio, err = CalmarRatio(50000, 15000, 0.2, 0.1)
	if err != nil {
		t.Error(err)
	}
	if ratio != 0.14285714285714288 {
		t.Error(ratio)
	}
}

func TestCAGR(t *testing.T) {
	t.Parallel()
	_, err := CompoundAnnualGrowthRate(
		0,
		0,
		0,
		0)
	assert.ErrorIs(t, err, errCAGRNoIntervals)

	_, err = CompoundAnnualGrowthRate(
		0,
		0,
		0,
		1)
	assert.ErrorIs(t, err, errCAGRZeroOpenValue)

	var cagr float64
	cagr, err = CompoundAnnualGrowthRate(
		100,
		147,
		1,
		1)
	if err != nil {
		t.Error(err)
	}
	if cagr != 47 {
		t.Error("expected 47%")
	}
	cagr, err = CompoundAnnualGrowthRate(
		100,
		147,
		365,
		365)
	if err != nil {
		t.Error(err)
	}
	if cagr != 47 {
		t.Error("expected 47%")
	}

	cagr, err = CompoundAnnualGrowthRate(
		100,
		200,
		1,
		20)
	if err != nil {
		t.Error(err)
	}
	if cagr != 3.5264923841377582 {
		t.Error("expected 3.53%")
	}
}

func TestCalculateSharpeRatio(t *testing.T) {
	t.Parallel()
	result, err := SharpeRatio(nil, 0, 0)
	assert.ErrorIs(t, err, errZeroValue)

	if result != 0 {
		t.Error("expected 0")
	}

	result, err = SharpeRatio([]float64{0.026}, 0.017, 0.026)
	if err != nil {
		t.Error(err)
	}
	if result != 0 {
		t.Error("expected 0")
	}

	// this follows and matches the example calculation (without rounding) from
	// https://www.educba.com/sharpe-ratio-formula/
	returns := []float64{
		-0.0005,
		-0.0065,
		-0.0113,
		0.0031,
		-0.0112,
		0.0056,
		0.0156,
		0.0048,
		0.0012,
		0.0038,
		-0.0008,
		0.0032,
		0,
		-0.0128,
		-0.0058,
		0.003,
		0.0042,
		0.0055,
		0.0009,
	}
	var avg float64
	avg, err = ArithmeticMean(returns)
	if err != nil {
		t.Error(err)
	}
	result, err = SharpeRatio(returns, -0.0017, avg)
	if err != nil {
		t.Error(err)
	}
	result = math.Round(result*100) / 100
	if result != 0.26 {
		t.Errorf("expected 0.26, received %v", result)
	}
}

func TestStandardDeviation2(t *testing.T) {
	t.Parallel()
	r := []float64{9, 2, 5, 4, 12, 7}
	mean, err := ArithmeticMean(r)
	if err != nil {
		t.Error(err)
	}
	superMean := []float64{}
	for i := range r {
		result := math.Pow(r[i]-mean, 2)
		superMean = append(superMean, result)
	}
	superMeany := (superMean[0] + superMean[1] + superMean[2] + superMean[3] + superMean[4] + superMean[5]) / 5
	manualCalculation := math.Sqrt(superMeany)
	var codeCalcu float64
	codeCalcu, err = SampleStandardDeviation(r)
	if err != nil {
		t.Error(err)
	}
	if manualCalculation != codeCalcu && codeCalcu != 3.619 {
		t.Error("expected 3.619")
	}
}

func TestGeometricAverage(t *testing.T) {
	t.Parallel()
	values := []float64{1, 2, 3, 4, 5, 6, 7, 8}
	_, err := GeometricMean(nil)
	assert.ErrorIs(t, err, errZeroValue)

	var mean float64
	mean, err = GeometricMean(values)
	if err != nil {
		t.Error(err)
	}
	if mean != 3.764350599503129 {
		t.Errorf("expected %v, received %v", 3.95, mean)
	}

	values = []float64{15, 12, 13, 19, 10}
	mean, err = GeometricMean(values)
	if err != nil {
		t.Error(err)
	}
	if mean != 13.477020583645698 {
		t.Errorf("expected %v, received %v", 13.50, mean)
	}

	values = []float64{-1, 12, 13, 19, 10}
	mean, err = GeometricMean(values)
	assert.ErrorIs(t, err, errGeometricNegative)

	if mean != 0 {
		t.Errorf("expected %v, received %v", 0, mean)
	}
}

func TestFinancialGeometricAverage(t *testing.T) {
	t.Parallel()
	values := []float64{1, 2, 3, 4, 5, 6, 7, 8}
	_, err := FinancialGeometricMean(nil)
	assert.ErrorIs(t, err, errZeroValue)

	var mean float64
	mean, err = FinancialGeometricMean(values)
	if err != nil {
		t.Error(err)
	}
	if mean != 3.9541639996482028 {
		t.Errorf("expected %v, received %v", 3.95, mean)
	}

	values = []float64{15, 12, 13, 19, 10}
	mean, err = FinancialGeometricMean(values)
	if err != nil {
		t.Error(err)
	}
	if mean != 13.49849123325646 {
		t.Errorf("expected %v, received %v", 13.50, mean)
	}

	values = []float64{-1, 12, 13, 19, 10}
	mean, err = FinancialGeometricMean(values)
	if err != nil {
		t.Error(err)
	}
	if mean != 0 {
		t.Errorf("expected %v, received %v", 0, mean)
	}

	values = []float64{-2, 12, 13, 19, 10}
	_, err = FinancialGeometricMean(values)
	assert.ErrorIs(t, err, errNegativeValueOutOfRange)
}

func TestArithmeticAverage(t *testing.T) {
	t.Parallel()
	values := []float64{1, 2, 3, 4, 5, 6, 7, 8}
	_, err := ArithmeticMean(nil)
	assert.ErrorIs(t, err, errZeroValue)

	var avg float64
	avg, err = ArithmeticMean(values)
	if err != nil {
		t.Error(err)
	}
	if avg != 4.5 {
		t.Error("expected 4.5")
	}
}

func TestDecimalSortinoRatio(t *testing.T) {
	t.Parallel()
	rfr := decimal.NewFromFloat(0.001)
	figures := []decimal.Decimal{
		decimal.NewFromFloat(0.10),
		decimal.NewFromFloat(0.04),
		decimal.NewFromFloat(0.15),
		decimal.NewFromFloat(-0.05),
		decimal.NewFromFloat(0.20),
		decimal.NewFromFloat(-0.02),
		decimal.NewFromFloat(0.08),
		decimal.NewFromFloat(-0.06),
		decimal.NewFromFloat(0.13),
		decimal.NewFromFloat(0.23),
	}
	avg, err := DecimalArithmeticMean(figures)
	require.NoError(t, err)
	_, err = DecimalSortinoRatio(nil, rfr, avg)
	assert.ErrorIs(t, err, errZeroValue)

	r, err := DecimalSortinoRatio(figures, rfr, avg)
	assert.ErrorIs(t, err, ErrInexactConversion)
	rf, exact := r.Float64()
	assert.False(t, exact)
	assert.Equal(t, 3.0377875479459906, rf)

	avg, err = DecimalFinancialGeometricMean(figures)
	require.NoError(t, err)
	r, err = DecimalSortinoRatio(figures, rfr, avg)
	assert.ErrorIs(t, err, ErrInexactConversion)
	assert.True(t, r.Equal(decimal.NewFromFloat(2.8712802265603243)))

	// this follows and matches the example calculation from
	// https://www.wallstreetmojo.com/sortino-ratio/
	example := []decimal.Decimal{
		decimal.NewFromFloat(0.1),
		decimal.NewFromFloat(0.12),
		decimal.NewFromFloat(0.07),
		decimal.NewFromFloat(-0.03),
		decimal.NewFromFloat(0.08),
		decimal.NewFromFloat(-0.04),
		decimal.NewFromFloat(0.15),
		decimal.NewFromFloat(0.2),
		decimal.NewFromFloat(0.12),
		decimal.NewFromFloat(0.06),
		decimal.NewFromFloat(-0.03),
		decimal.NewFromFloat(0.02),
	}
	avg, err = DecimalArithmeticMean(example)
	require.NoError(t, err)
	r, err = DecimalSortinoRatio(example, decimal.NewFromFloat(0.06), avg)
	assert.ErrorIs(t, err, ErrInexactConversion)
	assert.True(t, r.Round(1).Equal(decimal.NewFromFloat(0.2)))
}

func TestDecimalInformationRatio(t *testing.T) {
	t.Parallel()
	figures := []decimal.Decimal{
		decimal.NewFromFloat(0.0665),
		decimal.NewFromFloat(0.0283),
		decimal.NewFromFloat(0.0911),
		decimal.NewFromFloat(0.0008),
		decimal.NewFromFloat(-0.0203),
		decimal.NewFromFloat(-0.0978),
		decimal.NewFromFloat(0.0164),
		decimal.NewFromFloat(-0.0537),
		decimal.NewFromFloat(0.078),
		decimal.NewFromFloat(0.0032),
		decimal.NewFromFloat(0.0249),
		decimal.Zero,
	}
	comparisonFigures := []decimal.Decimal{
		decimal.NewFromFloat(0.0216),
		decimal.NewFromFloat(0.0048),
		decimal.NewFromFloat(0.036),
		decimal.NewFromFloat(0.0303),
		decimal.NewFromFloat(0.0043),
		decimal.NewFromFloat(-0.0694),
		decimal.NewFromFloat(0.0179),
		decimal.NewFromFloat(-0.0918),
		decimal.NewFromFloat(0.0787),
		decimal.NewFromFloat(0.0297),
		decimal.NewFromFloat(0.003),
		decimal.Zero,
	}
	avg, err := DecimalArithmeticMean(figures)
	require.NoError(t, err)
	assert.True(t, decimal.NewFromFloat(0.01145).Equal(avg))

	avgComparison, err := DecimalArithmeticMean(comparisonFigures)
	require.NoError(t, err)
	assert.True(t, decimal.NewFromFloat(0.005425).Equal(avgComparison))

	eachDiff := make([]decimal.Decimal, len(figures))
	for i := range figures {
		eachDiff[i] = figures[i].Sub(comparisonFigures[i])
	}
	stdDev, err := DecimalPopulationStandardDeviation(eachDiff)
	require.ErrorIs(t, err, ErrInexactConversion)
	assert.Equal(t, decimal.NewFromFloat(0.028992588851865227), stdDev)

	information := avg.Sub(avgComparison).Div(stdDev)
	assert.Equal(t, decimal.NewFromFloat(0.2078117283966652), information)

	information2, err := DecimalInformationRatio(figures, comparisonFigures, avg, avgComparison)
	require.NoError(t, err)
	assert.Equal(t, information, information2)

	_, err = DecimalInformationRatio(figures, []decimal.Decimal{decimal.NewFromInt(1)}, avg, avgComparison)
	assert.ErrorIs(t, err, errInformationBadLength)
}

func TestDecimalCalmarRatio(t *testing.T) {
	t.Parallel()
	_, err := DecimalCalmarRatio(decimal.Zero, decimal.Zero, decimal.Zero, decimal.Zero)
	assert.ErrorIs(t, err, errCalmarHighest)

	var ratio decimal.Decimal
	ratio, err = DecimalCalmarRatio(
		decimal.NewFromInt(50000),
		decimal.NewFromInt(15000),
		decimal.NewFromFloat(0.2),
		decimal.NewFromFloat(0.1))
	if err != nil {
		t.Error(err)
	}
	if !ratio.Equal(decimal.NewFromFloat(0.1428571428571429)) {
		t.Error(ratio)
	}
}

func TestDecimalCalculateSharpeRatio(t *testing.T) {
	t.Parallel()
	result, err := DecimalSharpeRatio(nil, decimal.Zero, decimal.Zero)
	assert.ErrorIs(t, err, errZeroValue)

	if !result.IsZero() {
		t.Error("expected 0")
	}

	result, err = DecimalSharpeRatio([]decimal.Decimal{decimal.NewFromFloat(0.026)}, decimal.NewFromFloat(0.017), decimal.NewFromFloat(0.026))
	if err != nil {
		t.Error(err)
	}
	if !result.IsZero() {
		t.Error("expected 0")
	}

	// this follows and matches the example calculation (without rounding) from
	// https://www.educba.com/sharpe-ratio-formula/
	returns := []decimal.Decimal{
		decimal.NewFromFloat(-0.0005),
		decimal.NewFromFloat(-0.0065),
		decimal.NewFromFloat(-0.0113),
		decimal.NewFromFloat(0.0031),
		decimal.NewFromFloat(-0.0112),
		decimal.NewFromFloat(0.0056),
		decimal.NewFromFloat(0.0156),
		decimal.NewFromFloat(0.0048),
		decimal.NewFromFloat(0.0012),
		decimal.NewFromFloat(0.0038),
		decimal.NewFromFloat(-0.0008),
		decimal.NewFromFloat(0.0032),
		decimal.Zero,
		decimal.NewFromFloat(-0.0128),
		decimal.NewFromFloat(-0.0058),
		decimal.NewFromFloat(0.003),
		decimal.NewFromFloat(0.0042),
		decimal.NewFromFloat(0.0055),
		decimal.NewFromFloat(0.0009),
	}
	var avg decimal.Decimal
	avg, err = DecimalArithmeticMean(returns)
	if err != nil {
		t.Error(err)
	}
	result, err = DecimalSharpeRatio(returns, decimal.NewFromFloat(-0.0017), avg)
	if err != nil {
		t.Error(err)
	}
	result = result.Round(2)
	if !result.Equal(decimal.NewFromFloat(0.26)) {
		t.Errorf("expected 0.26, received %v", result)
	}
}

func TestDecimalStandardDeviation2(t *testing.T) {
	t.Parallel()
	r := []decimal.Decimal{
		decimal.NewFromInt(9),
		decimal.NewFromInt(2),
		decimal.NewFromInt(5),
		decimal.NewFromInt(4),
		decimal.NewFromInt(12),
		decimal.NewFromInt(7),
	}
	mean, err := DecimalArithmeticMean(r)
	if err != nil {
		t.Error(err)
	}
	superMean := make([]decimal.Decimal, len(r))
	for i := range r {
		result := r[i].Sub(mean).Pow(decimal.NewFromInt(2))
		superMean[i] = result
	}
	superMeany := superMean[0].Add(superMean[1].Add(superMean[2].Add(superMean[3].Add(superMean[4].Add(superMean[5]))))).Div(decimal.NewFromInt(5))
	manualCalculation := decimal.NewFromFloat(math.Sqrt(superMeany.InexactFloat64()))
	var codeCalcu decimal.Decimal
	codeCalcu, err = DecimalSampleStandardDeviation(r)
	if err != nil {
		t.Error(err)
	}
	if !manualCalculation.Equal(codeCalcu) && codeCalcu.Equal(decimal.NewFromFloat(3.619)) {
		t.Error("expected 3.619")
	}
}

func TestDecimalGeometricAverage(t *testing.T) {
	t.Parallel()
	values := []decimal.Decimal{
		decimal.NewFromInt(1),
		decimal.NewFromInt(2),
		decimal.NewFromInt(3),
		decimal.NewFromInt(4),
		decimal.NewFromInt(5),
		decimal.NewFromInt(6),
		decimal.NewFromInt(7),
		decimal.NewFromInt(8),
	}
	_, err := DecimalGeometricMean(nil)
	assert.ErrorIs(t, err, errZeroValue)

	var mean decimal.Decimal
	mean, err = DecimalGeometricMean(values)
	if err != nil {
		t.Error(err)
	}
	if !mean.Equal(decimal.NewFromFloat(3.764350599503129)) {
		t.Errorf("expected %v, received %v", 3.95, mean)
	}

	values = []decimal.Decimal{
		decimal.NewFromInt(15),
		decimal.NewFromInt(12),
		decimal.NewFromInt(13),
		decimal.NewFromInt(19),
		decimal.NewFromInt(10),
	}
	mean, err = DecimalGeometricMean(values)
	if err != nil {
		t.Error(err)
	}
	if !mean.Equal(decimal.NewFromFloat(13.477020583645698)) {
		t.Errorf("expected %v, received %v", 13.50, mean)
	}

	values = []decimal.Decimal{
		decimal.NewFromInt(-1),
		decimal.NewFromInt(12),
		decimal.NewFromInt(13),
		decimal.NewFromInt(19),
		decimal.NewFromInt(10),
	}
	mean, err = DecimalGeometricMean(values)
	assert.ErrorIs(t, err, errGeometricNegative)

	if !mean.IsZero() {
		t.Errorf("expected %v, received %v", 0, mean)
	}
}

func TestDecimalFinancialGeometricAverage(t *testing.T) {
	t.Parallel()
	values := []decimal.Decimal{
		decimal.NewFromInt(1),
		decimal.NewFromInt(2),
		decimal.NewFromInt(3),
		decimal.NewFromInt(4),
		decimal.NewFromInt(5),
		decimal.NewFromInt(6),
		decimal.NewFromInt(7),
		decimal.NewFromInt(8),
	}
	_, err := DecimalFinancialGeometricMean(nil)
	assert.ErrorIs(t, err, errZeroValue)

	var mean decimal.Decimal
	mean, err = DecimalFinancialGeometricMean(values)
	if err != nil {
		t.Error(err)
	}
	if !mean.Equal(decimal.NewFromFloat(3.9541639996482028)) {
		t.Errorf("expected %v, received %v", 3.95, mean)
	}

	values = []decimal.Decimal{
		decimal.NewFromInt(15),
		decimal.NewFromInt(12),
		decimal.NewFromInt(13),
		decimal.NewFromInt(19),
		decimal.NewFromInt(10),
	}
	mean, err = DecimalFinancialGeometricMean(values)
	if err != nil {
		t.Error(err)
	}
	if !mean.Equal(decimal.NewFromFloat(13.49849123325646)) {
		t.Errorf("expected %v, received %v", 13.50, mean)
	}

	values = []decimal.Decimal{
		decimal.NewFromInt(-1),
		decimal.NewFromInt(12),
		decimal.NewFromInt(13),
		decimal.NewFromInt(19),
		decimal.NewFromInt(10),
	}
	mean, err = DecimalFinancialGeometricMean(values)
	if err != nil {
		t.Error(err)
	}
	if !mean.IsZero() {
		t.Errorf("expected %v, received %v", 0, mean)
	}

	values = []decimal.Decimal{
		decimal.NewFromInt(-2),
		decimal.NewFromInt(12),
		decimal.NewFromInt(13),
		decimal.NewFromInt(19),
		decimal.NewFromInt(10),
	}
	_, err = DecimalFinancialGeometricMean(values)
	assert.ErrorIs(t, err, errNegativeValueOutOfRange)
}

func TestDecimalArithmeticAverage(t *testing.T) {
	t.Parallel()
	values := []decimal.Decimal{
		decimal.NewFromInt(1),
		decimal.NewFromInt(2),
		decimal.NewFromInt(3),
		decimal.NewFromInt(4),
		decimal.NewFromInt(5),
		decimal.NewFromInt(6),
		decimal.NewFromInt(7),
		decimal.NewFromInt(8),
	}
	_, err := DecimalArithmeticMean(nil)
	assert.ErrorIs(t, err, errZeroValue)

	var avg decimal.Decimal
	avg, err = DecimalArithmeticMean(values)
	if err != nil {
		t.Error(err)
	}
	if !avg.Equal(decimal.NewFromFloat(4.5)) {
		t.Error("expected 4.5")
	}
}

func TestDecimalPow(t *testing.T) {
	t.Parallel()
	pow := DecimalPow(decimal.NewFromInt(2), decimal.NewFromInt(2))
	if !pow.Equal(decimal.NewFromInt(4)) {
		t.Errorf("received '%v' expected '%v'", pow, 4)
	}

	// zero
	pow = DecimalPow(decimal.Zero, decimal.NewFromInt(1))
	if !pow.Equal(decimal.Zero) {
		t.Errorf("received '%v' expected '%v'", pow, 0)
	}

	// inf
	pow = DecimalPow(decimal.Zero, decimal.NewFromInt(-3))
	if !pow.Equal(decimal.Zero) {
		t.Errorf("received '%v' expected '%v'", pow, 0)
	}

	// nan
	pow = DecimalPow(decimal.NewFromInt(-1), decimal.NewFromFloat(0.1111))
	if !pow.Equal(decimal.Zero) {
		t.Errorf("received '%v' expected '%v'", pow, 0)
	}
}
