package crypto

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHexEncodeToString(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "737472696e67", HexEncodeToString([]byte("string")))
}

func TestBase64Decode(t *testing.T) {
	t.Parallel()

	r, err := Base64Decode("aGVsbG8=")
	require.NoError(t, err, "Base64Decode must not error")
	assert.Equal(t, []byte("hello"), r, "Base64Decode should return the correct byte slice")

	_, err = Base64Decode("-")
	assert.Error(t, err, "Base64Decode should error on invalid input")
}

func TestBase64Encode(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "aGVsbG8=", Base64Encode([]byte("hello")),
		"Base64Encode should return the correct base64 string")
}

func TestGetRandomSalt(t *testing.T) {
	t.Parallel()

	_, err := GetRandomSalt(nil, -1)
	assert.ErrorContains(t, err, "salt length is too small", "Expected error on negative salt length")

	salt, err := GetRandomSalt(nil, 10)
	require.NoError(t, err, "GetRandomSalt must not error")
	assert.Len(t, salt, 10, "GetRandomSalt should return a salt of the specified length")

	salt, err = GetRandomSalt([]byte("RAWR"), 12)
	require.NoError(t, err, "GetRandomSalt must not error")
	assert.Len(t, salt, 16, "GetRandomSalt should return a salt of the specified length plus input length")
}

func TestGetMD5(t *testing.T) {
	t.Parallel()
	r, err := GetMD5([]byte("I am testing the MD5 function in common!"))
	require.NoError(t, err, "GetMD5 must not error")
	assert.Equal(t, "18fddf4a41ba90a7352765e62e7a8744", hex.EncodeToString(r), "GetMD5 result should match the expected output")
}

func TestGetSHA512(t *testing.T) {
	t.Parallel()
	originalString := []byte("I am testing the GetSHA512 function in common!")
	expectedOutput := []byte(
		`a2273f492ea73fddc4f25c267b34b3b74998bd8a6301149e1e1c835678e3c0b90859fce22e4e7af33bde1711cbb924809aedf5d759d648d61774b7185c5dc02b`,
	)
	actualOutput, err := GetSHA512(originalString)
	if err != nil {
		t.Fatal(err)
	}
	actualStr := HexEncodeToString(actualOutput)
	if !bytes.Equal(expectedOutput, []byte(actualStr)) {
		t.Errorf("Expected '%x'. Actual '%x'",
			expectedOutput, []byte(actualStr))
	}
}

func TestGetSHA256(t *testing.T) {
	t.Parallel()
	originalString := []byte("I am testing the GetSHA256 function in common!")
	expectedOutput := []byte(
		"0962813d7a9f739cdcb7f0c0be0c2a13bd630167e6e54468266e4af6b1ad9303",
	)
	actualOutput, err := GetSHA256(originalString)
	if err != nil {
		t.Fatal(err)
	}

	actualStr := HexEncodeToString(actualOutput)
	if !bytes.Equal(expectedOutput, []byte(actualStr)) {
		t.Errorf("Expected '%x'. Actual '%x'", expectedOutput,
			[]byte(actualStr))
	}
}

func TestGetHMAC(t *testing.T) {
	t.Parallel()
	expectedSha1 := []byte{
		74, 253, 245, 154, 87, 168, 110, 182, 172, 101, 177, 49, 142, 2, 253, 165,
		100, 66, 86, 246,
	}
	expectedsha256 := []byte{
		54, 68, 6, 12, 32, 158, 80, 22, 142, 8, 131, 111, 248, 145, 17, 202, 224,
		59, 135, 206, 11, 170, 154, 197, 183, 28, 150, 79, 168, 105, 62, 102,
	}
	expectedsha512 := []byte{
		249, 212, 31, 38, 23, 3, 93, 220, 81, 209, 214, 112, 92, 75, 126, 40, 109,
		95, 247, 182, 210, 54, 217, 224, 199, 252, 129, 226, 97, 201, 245, 220, 37,
		201, 240, 15, 137, 236, 75, 6, 97, 12, 190, 31, 53, 153, 223, 17, 214, 11,
		153, 203, 49, 29, 158, 217, 204, 93, 179, 109, 140, 216, 202, 71,
	}
	expectedsha512384 := []byte{
		121, 203, 109, 105, 178, 68, 179, 57, 21, 217, 76, 82, 94, 100, 210, 1, 55,
		201, 8, 232, 194, 168, 165, 58, 192, 26, 193, 167, 254, 183, 172, 4, 189,
		158, 158, 150, 173, 33, 119, 125, 94, 13, 125, 89, 241, 184, 166, 128,
	}
	expectedmd5 := []byte{
		113, 64, 132, 129, 213, 68, 231, 99, 252, 15, 175, 109, 198, 132, 139, 39,
	}

	sha1, err := GetHMAC(HashSHA1, []byte("Hello,World"), []byte("1234"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(sha1, expectedSha1) {
		t.Errorf("Common GetHMAC error: Expected '%x'. Actual '%x'",
			expectedSha1, sha1,
		)
	}
	sha256, err := GetHMAC(HashSHA256, []byte("Hello,World"), []byte("1234"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(sha256, expectedsha256) {
		t.Errorf("Common GetHMAC error: Expected '%x'. Actual '%x'",
			expectedsha256, sha256,
		)
	}
	sha512, err := GetHMAC(HashSHA512, []byte("Hello,World"), []byte("1234"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(sha512, expectedsha512) {
		t.Errorf("Common GetHMAC error: Expected '%x'. Actual '%x'",
			expectedsha512, sha512,
		)
	}
	sha512384, err := GetHMAC(HashSHA512_384, []byte("Hello,World"), []byte("1234"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(sha512384, expectedsha512384) {
		t.Errorf("Common GetHMAC error: Expected '%x'. Actual '%x'",
			expectedsha512384, sha512384,
		)
	}
	md5, err := GetHMAC(HashMD5, []byte("Hello World"), []byte("1234"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(md5, expectedmd5) {
		t.Errorf("Common GetHMAC error: Expected '%x'. Actual '%x'",
			expectedmd5, md5,
		)
	}
}
