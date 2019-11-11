package crypto

import (
	"bytes"
	"testing"
)

func TestHexEncodeToString(t *testing.T) {
	t.Parallel()
	originalInput := []byte("string")
	expectedOutput := "737472696e67"
	actualResult := HexEncodeToString(originalInput)
	if actualResult != expectedOutput {
		t.Errorf("Expected '%s'. Actual '%s'",
			expectedOutput, actualResult)
	}
}

func TestBase64Decode(t *testing.T) {
	t.Parallel()
	originalInput := "aGVsbG8="
	expectedOutput := []byte("hello")
	actualResult, err := Base64Decode(originalInput)
	if !bytes.Equal(actualResult, expectedOutput) {
		t.Errorf("Expected '%s'. Actual '%s'. Error: %s",
			expectedOutput, actualResult, err)
	}

	_, err = Base64Decode("-")
	if err == nil {
		t.Error("Bad base64 string failed returned nil error")
	}
}

func TestBase64Encode(t *testing.T) {
	t.Parallel()
	originalInput := []byte("hello")
	expectedOutput := "aGVsbG8="
	actualResult := Base64Encode(originalInput)
	if actualResult != expectedOutput {
		t.Errorf("Expected '%s'. Actual '%s'",
			expectedOutput, actualResult)
	}
}

func TestGetRandomSalt(t *testing.T) {
	t.Parallel()

	_, err := GetRandomSalt(nil, -1)
	if err == nil {
		t.Fatal("Expected err on negative salt length")
	}

	salt, err := GetRandomSalt(nil, 10)
	if err != nil {
		t.Fatal(err)
	}

	if len(salt) != 10 {
		t.Fatal("Expected salt of len=10")
	}

	salt, err = GetRandomSalt([]byte("RAWR"), 12)
	if err != nil {
		t.Fatal(err)
	}

	if len(salt) != 16 {
		t.Fatal("Expected salt of len=16")
	}
}

func TestGetMD5(t *testing.T) {
	t.Parallel()
	var originalString = []byte("I am testing the MD5 function in common!")
	var expectedOutput = []byte("18fddf4a41ba90a7352765e62e7a8744")
	actualOutput := GetMD5(originalString)
	actualStr := HexEncodeToString(actualOutput)
	if !bytes.Equal(expectedOutput, []byte(actualStr)) {
		t.Errorf("Expected '%s'. Actual '%s'",
			expectedOutput, []byte(actualStr))
	}
}

func TestGetSHA512(t *testing.T) {
	t.Parallel()
	var originalString = []byte("I am testing the GetSHA512 function in common!")
	var expectedOutput = []byte(
		`a2273f492ea73fddc4f25c267b34b3b74998bd8a6301149e1e1c835678e3c0b90859fce22e4e7af33bde1711cbb924809aedf5d759d648d61774b7185c5dc02b`,
	)
	actualOutput := GetSHA512(originalString)
	actualStr := HexEncodeToString(actualOutput)
	if !bytes.Equal(expectedOutput, []byte(actualStr)) {
		t.Errorf("Expected '%x'. Actual '%x'",
			expectedOutput, []byte(actualStr))
	}
}

func TestGetSHA256(t *testing.T) {
	t.Parallel()
	var originalString = []byte("I am testing the GetSHA256 function in common!")
	var expectedOutput = []byte(
		"0962813d7a9f739cdcb7f0c0be0c2a13bd630167e6e54468266e4af6b1ad9303",
	)
	actualOutput := GetSHA256(originalString)
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

	sha1 := GetHMAC(HashSHA1, []byte("Hello,World"), []byte("1234"))
	if string(sha1) != string(expectedSha1) {
		t.Errorf("Common GetHMAC error: Expected '%x'. Actual '%x'",
			expectedSha1, sha1,
		)
	}
	sha256 := GetHMAC(HashSHA256, []byte("Hello,World"), []byte("1234"))
	if string(sha256) != string(expectedsha256) {
		t.Errorf("Common GetHMAC error: Expected '%x'. Actual '%x'",
			expectedsha256, sha256,
		)
	}
	sha512 := GetHMAC(HashSHA512, []byte("Hello,World"), []byte("1234"))
	if string(sha512) != string(expectedsha512) {
		t.Errorf("Common GetHMAC error: Expected '%x'. Actual '%x'",
			expectedsha512, sha512,
		)
	}
	sha512384 := GetHMAC(HashSHA512_384, []byte("Hello,World"), []byte("1234"))
	if string(sha512384) != string(expectedsha512384) {
		t.Errorf("Common GetHMAC error: Expected '%x'. Actual '%x'",
			expectedsha512384, sha512384,
		)
	}
	md5 := GetHMAC(HashMD5, []byte("Hello World"), []byte("1234"))
	if string(md5) != string(expectedmd5) {
		t.Errorf("Common GetHMAC error: Expected '%x'. Actual '%x'",
			expectedmd5, md5,
		)
	}
}

func TestSha1Tohex(t *testing.T) {
	t.Parallel()
	expectedResult := "fcfbfcd7d31d994ef660f6972399ab5d7a890149"
	actualResult := Sha1ToHex("Testing Sha1ToHex")
	if actualResult != expectedResult {
		t.Errorf("Expected '%s'. Actual '%s'",
			expectedResult, actualResult)
	}
}
