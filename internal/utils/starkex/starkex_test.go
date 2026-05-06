package starkex

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	MockPublicKey  = "0x3b865a18323b8d147a12c556bfb1d502516c325b1477a23ba6c77af31f020fd"
	MockPrivateKey = "0x58c7d5a90b1776bde86ebac077e053ed85b0f7164f53b080304a531947f46e3"
)

func TestNewStarkExConfig(t *testing.T) {
	t.Parallel()
	result, err := NewStarkExConfig()
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestECDSASignature(t *testing.T) {
	t.Parallel()
	magHash, ok := big.NewInt(0).SetString("0x011049f4032190ec4b5a9420cc77006d13a260df46bfcacf60a53f447a5a925d", 0)
	require.True(t, ok)

	publicX, ok := big.NewInt(0).SetString(MockPublicKey, 0)
	require.True(t, ok)

	publicSecret, ok := big.NewInt(0).SetString(MockPrivateKey, 0)
	require.True(t, ok)

	sfg, err := NewStarkExConfig()
	require.NoError(t, err)
	require.NotNil(t, sfg)

	r, s, err := sfg.SignECDSA(magHash, publicSecret)
	require.NoError(t, err)

	publicY := sfg.GetYCoordinate(publicX)
	ok = sfg.Verify(magHash, r, s, [2]*big.Int{publicX, publicY})
	require.True(t, ok,
		ErrFailedToGenerateSignature)
}

type hashAndSignature struct {
	Hash string `json:"hash"`
	R    string `json:"r"`
	S    string `json:"s"`
}

var ref6979SignatureTestVector = &struct {
	PrivateKey string             `json:"private_key"` //nolint:gosec // Used for testing purpose
	Messages   []hashAndSignature `json:"messages"`
}{
	PrivateKey: "0x3c1e9550e66958296d11b60f8e8e7a7ad990d07fa65d5f7652c4a6c87d4e3cc",
	Messages: []hashAndSignature{
		{
			Hash: "0x1",
			R:    "3162358736122783857144396205516927012128897537504463716197279730251407200037",
			S:    "1447067116407676619871126378936374427636662490882969509559888874644844560850",
		},
		{
			Hash: "0x11",
			R:    "2282960348362869237018441985726545922711140064809058182483721438101695251648",
			S:    "2905868291002627709651322791912000820756370440695830310841564989426104902684",
		},

		{
			Hash: "0x223",
			R:    "2851492577225522862152785068304516872062840835882746625971400995051610132955",
			S:    "2227464623243182122770469099770977514100002325017609907274766387592987135410",
		},

		{
			Hash: "0x9999",
			R:    "3551214266795401081823453828727326248401688527835302880992409448142527576296",
			S:    "2580950807716503852408066180369610390914312729170066679103651110985466032285",
		},

		{
			Hash: "0x387e76d1667c4454bfb835144120583af836f8e32a516765497d23eabe16b3f",
			R:    "3518448914047769356425227827389998721396724764083236823647519654917215164512",
			S:    "3042321032945513635364267149196358883053166552342928199041742035443537684462",
		},

		{
			Hash: "0x3a7e76d1697c4455bfb835144120283af236f8e32a516765497d23eabe16b2",
			R:    "2261926635950780594216378185339927576862772034098248230433352748057295357217",
			S:    "2708700003762962638306717009307430364534544393269844487939098184375356178572",
		},

		{
			Hash: "0xfa5f0cd1ebff93c9e6474379a213ba111f9e42f2f1cb361b0327e0737203",
			R:    "3016953906936760149710218073693613509330129567629289734816320774638425763370",
			S:    "306146275372136078470081798635201810092238376869367156373203048583896337506",
		},

		{
			Hash: "0x4c1e9550e66958296d11b60f8e8e7f7ae99dd0cfa6bd5fa652c1a6c87d4e2cc",
			R:    "3562728603055564208884290243634917206833465920158600288670177317979301056463",
			S:    "1958799632261808501999574190111106370256896588537275453140683641951899459876",
		},

		{
			Hash: "0x6362b40c218fb4c8a8bd42ca482145e8513b78e00faa0de76a98ba14fc37ae8",
			R:    "3485557127492692423490706790022678621438670833185864153640824729109010175518",
			S:    "897592218067946175671768586886915961592526001156186496738437723857225288280",
		},
	},
}

func TestECDSASignatureFromFile(t *testing.T) {
	sfg, err := NewStarkExConfig()
	require.NoError(t, err)
	require.NotNil(t, sfg)

	privateKey, ok := big.NewInt(0).SetString(ref6979SignatureTestVector.PrivateKey, 0)
	require.True(t, ok)

	for a := range ref6979SignatureTestVector.Messages {
		hashMessage, ok := big.NewInt(0).SetString(ref6979SignatureTestVector.Messages[a].Hash, 0)
		require.True(t, ok)

		expR, ok := new(big.Int).SetString(ref6979SignatureTestVector.Messages[a].R, 10)
		require.True(t, ok)

		expS, ok := new(big.Int).SetString(ref6979SignatureTestVector.Messages[a].S, 10)
		require.True(t, ok)

		r, s, err := sfg.SignECDSA(hashMessage, privateKey)
		require.NoError(t, err)
		require.Equal(t, 0, r.Cmp(expR))
		require.Equal(t, 0, s.Cmp(expS))
	}
}

func TestOrderSign(t *testing.T) {
	t.Parallel()
	sfg, err := NewStarkExConfig()
	require.NoError(t, err)
	require.NotNil(t, sfg)

	syntheticAssetID, ok := big.NewInt(0).SetString("344400637343183300222065759427231744", 10)
	require.True(t, ok)

	collateralAssetID, ok := big.NewInt(0).SetString("1147032829293317481173155891309375254605214077236177772270270553197624560221", 10)
	require.True(t, ok)

	arg := &CreateOrderWithFeeParams{
		OrderType:               "LIMIT_ORDER_WITH_FEES",
		AssetIDSynthetic:        syntheticAssetID,
		AssetIDCollateral:       collateralAssetID,
		AssetIDFee:              collateralAssetID,
		QuantumAmountSynthetic:  big.NewInt(100000000),
		QuantumAmountCollateral: big.NewInt(200000000),
		QuantumAmountFee:        big.NewInt(100000),
		IsBuyingSynthetic:       false,
		PositionID:              big.NewInt(603545650545558021),
		Nonce:                   big.NewInt(3762202436),
		ExpirationEpochHours:    big.NewInt(479941),
	}
	r, s, err := sfg.Sign(arg, MockPrivateKey, MockPublicKey, "")
	require.NoError(t, err)
	assert.NotEmpty(t, r)
	assert.NotEmpty(t, s)
}

func TestGetYCoordinate(t *testing.T) {
	t.Parallel()
	sfg, err := NewStarkExConfig()
	require.NoError(t, err)
	require.NotNil(t, sfg)

	publicX, ok := big.NewInt(0).SetString(MockPublicKey, 0)
	assert.True(t, ok)

	result := sfg.GetYCoordinate(publicX)
	assert.NotNil(t, result)
}
