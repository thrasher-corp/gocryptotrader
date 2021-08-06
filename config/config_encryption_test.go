package config

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestPromptForConfigEncryption(t *testing.T) {
	t.Parallel()

	confirm, err := promptForConfigEncryption()
	if confirm {
		t.Error("promptForConfigEncryption return incorrect bool")
	}
	if err == nil {
		t.Error("Expected error as there is no input")
	}
}

func TestPromptForConfigKey(t *testing.T) {
	t.Parallel()

	byteyBite, err := PromptForConfigKey(true)
	if err == nil && len(byteyBite) > 1 {
		t.Errorf("PromptForConfigKey: %s", err)
	}

	_, err = PromptForConfigKey(false)
	if err == nil {
		t.Error("Expected error")
	}
}

func TestEncryptConfigFile(t *testing.T) {
	t.Parallel()
	_, err := EncryptConfigFile([]byte("test"), nil)
	if err == nil {
		t.Fatal("Expected error")
	}

	c := &Config{
		sessionDK: []byte("a"),
	}
	_, err = c.encryptConfigFile([]byte("test"))
	if err == nil {
		t.Fatal("Expected error")
	}

	sessDk, salt, err := makeNewSessionDK([]byte("asdf"))
	if err != nil {
		t.Fatal(err)
	}

	c = &Config{
		sessionDK:  sessDk,
		storedSalt: salt,
	}
	_, err = c.encryptConfigFile([]byte("test"))
	if err != nil {
		t.Fatal(err)
	}
}

func TestDecryptConfigFile(t *testing.T) {
	t.Parallel()
	result, err := EncryptConfigFile([]byte("test"), []byte("key"))
	if err != nil {
		t.Fatal(err)
	}

	_, err = DecryptConfigFile(result, nil)
	if err == nil {
		t.Fatal("Expected error")
	}

	_, err = DecryptConfigFile([]byte("test"), nil)
	if err == nil {
		t.Fatal("Expected error")
	}

	_, err = DecryptConfigFile([]byte("test"), []byte("AAAAAAAAAAAAAAAA"))
	if err == nil {
		t.Fatalf("Expected %s", errAESBlockSize)
	}

	result, err = EncryptConfigFile([]byte("test"), []byte("key"))
	if err != nil {
		t.Fatal(err)
	}

	_, err = DecryptConfigFile(result, []byte("key"))
	if err != nil {
		t.Fatal(err)
	}
}

func TestConfirmECS(t *testing.T) {
	t.Parallel()

	ECStest := []byte(EncryptConfirmString)
	if !ConfirmECS(ECStest) {
		t.Errorf("TestConfirmECS: Error finding ECS.")
	}
}

func TestRemoveECS(t *testing.T) {
	t.Parallel()

	ECStest := []byte(EncryptConfirmString)
	reader := bytes.NewReader(ECStest)
	err := skipECS(reader)
	if err != nil {
		t.Error(err)
	}

	// Attempt read
	var buf []byte
	_, err = reader.Read(buf)
	if err != io.EOF {
		t.Errorf("TestConfirmECS: Error ECS not deleted.")
	}
}

func TestMakeNewSessionDK(t *testing.T) {
	t.Parallel()

	_, _, err := makeNewSessionDK(nil)
	if err == nil {
		t.Fatal("makeNewSessionDK passed with nil key")
	}
}

func TestEncryptTwiceReusesSaltButNewCipher(t *testing.T) {
	c := &Config{
		EncryptConfig: 1,
	}
	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("Problem creating temp dir at %s: %s\n", tempDir, err)
	}
	defer os.RemoveAll(tempDir)

	// Prepare input
	passFile, err := ioutil.TempFile(tempDir, "*.pw")
	if err != nil {
		t.Fatalf("Problem creating temp file at %s: %s\n", tempDir, err)
	}
	_, err = passFile.WriteString("pass\npass\n")
	if err != nil {
		t.Error(err)
	}
	err = passFile.Close()
	if err != nil {
		t.Error(err)
	}

	// Temporarily replace Stdin with a custom input
	oldIn := os.Stdin
	defer func() { os.Stdin = oldIn }()
	os.Stdin, err = os.Open(passFile.Name())
	if err != nil {
		t.Fatalf("Problem opening temp file at %s: %s\n", passFile.Name(), err)
	}

	// Save encrypted config
	enc1 := filepath.Join(tempDir, "encrypted.dat")
	err = c.SaveConfigToFile(enc1)
	if err != nil {
		t.Fatalf("Problem storing config in file %s: %s\n", enc1, err)
	}
	// Save again
	enc2 := filepath.Join(tempDir, "encrypted2.dat")
	err = c.SaveConfigToFile(enc2)
	if err != nil {
		t.Fatalf("Problem storing config in file %s: %s\n", enc2, err)
	}
	data1, err := ioutil.ReadFile(enc1)
	if err != nil {
		t.Fatalf("Problem reading file %s: %s\n", enc1, err)
	}
	data2, err := ioutil.ReadFile(enc2)
	if err != nil {
		t.Fatalf("Problem reading file %s: %s\n", enc2, err)
	}
	// length of prefix + salt
	l := len(EncryptConfirmString+SaltPrefix) + SaltRandomLength
	// Even though prefix, including salt with the random bytes is the same
	if !bytes.Equal(data1[:l], data2[:l]) {
		t.Error("Salt is not reused.")
	}
	// the cipher text should not be
	if bytes.Equal(data1, data2) {
		t.Error("Encryption key must have been reused as cipher texts are the same")
	}
}

func TestSaveAndReopenEncryptedConfig(t *testing.T) {
	c := &Config{}
	c.Name = "myCustomName"
	c.EncryptConfig = 1
	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("Problem creating temp dir at %s: %s\n", tempDir, err)
	}
	defer os.RemoveAll(tempDir)

	// Save encrypted config
	enc := filepath.Join(tempDir, "encrypted.dat")
	err = withInteractiveResponse(t, "pass\npass\n", func() error {
		return c.SaveConfigToFile(enc)
	})
	if err != nil {
		t.Fatalf("Problem storing config in file %s: %s\n", enc, err)
	}

	readConf := &Config{}
	err = withInteractiveResponse(t, "pass\n", func() error {
		// Load with no existing state, key is read from the prepared file
		return readConf.ReadConfigFromFile(enc, true)
	})

	// Verify
	if err != nil {
		t.Fatalf("Problem reading config in file %s: %s\n", enc, err)
	}

	if c.Name != readConf.Name || c.EncryptConfig != readConf.EncryptConfig {
		t.Error("Loaded conf not the same as original")
	}
}

// setAnswersFile sets the given file as the current stdin
// returns the close function to defer for reverting the stdin
func setAnswersFile(t *testing.T, answerFile string) func() {
	oldIn := os.Stdin

	inputFile, err := os.Open(answerFile)
	if err != nil {
		t.Fatalf("Problem opening temp file at %s: %s\n", answerFile, err)
	}
	os.Stdin = inputFile
	return func() {
		inputFile.Close()
		os.Stdin = oldIn
	}
}

func TestReadConfigWithPrompt(t *testing.T) {
	// Prepare temp dir
	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("Problem creating temp dir at %s: %s\n", tempDir, err)
	}
	defer os.RemoveAll(tempDir)

	// Ensure we'll get the prompt when loading
	c := &Config{
		EncryptConfig: 0,
	}

	// Save config
	testConfigFile := filepath.Join(tempDir, "config.json")
	err = c.SaveConfigToFile(testConfigFile)
	if err != nil {
		t.Fatalf("Problem saving config file in %s: %s\n", tempDir, err)
	}

	// Run the test
	c = &Config{}
	err = withInteractiveResponse(t, "y\npass\npass\n", func() error {
		return c.ReadConfigFromFile(testConfigFile, false)
	})
	if err != nil {
		t.Fatalf("Problem reading config file at %s: %s\n", testConfigFile, err)
	}

	// Verify results
	data, err := ioutil.ReadFile(testConfigFile)
	if err != nil {
		t.Fatalf("Problem reading saved file at %s: %s\n", testConfigFile, err)
	}
	if c.EncryptConfig != fileEncryptionEnabled {
		t.Error("Config encryption flag should be set after prompts")
	}
	if !ConfirmECS(data) {
		t.Error("Config file should be encrypted after prompts")
	}
}

func TestReadEncryptedConfigFromReader(t *testing.T) {
	t.Parallel()
	keyProvider := func() ([]byte, error) { return []byte("pass"), nil }
	// Encrypted conf for: `{"name":"test"}` with key `pass`
	confBytes := []byte{84, 72, 79, 82, 83, 45, 72, 65, 77, 77, 69, 82, 126, 71, 67, 84, 126, 83, 79, 126, 83, 65, 76, 84, 89, 126, 246, 110, 128, 3, 30, 168, 172, 160, 198, 176, 136, 62, 152, 155, 253, 176, 16, 48, 52, 246, 44, 29, 151, 47, 217, 226, 178, 12, 218, 113, 248, 172, 195, 232, 136, 104, 9, 199, 20, 4, 71, 4, 253, 249}
	conf, encrypted, err := ReadConfig(bytes.NewReader(confBytes), keyProvider)
	if err != nil {
		t.Errorf("TestReadConfig %s", err)
	}
	if !encrypted {
		t.Errorf("Expected encrypted config %s", err)
	}
	if conf.Name != "test" {
		t.Errorf("Conf not properly loaded %s", err)
	}

	// Change the salt
	confBytes[20] = 0
	conf, _, err = ReadConfig(bytes.NewReader(confBytes), keyProvider)
	if err == nil {
		t.Errorf("Expected unable to decrypt, but got %+v", conf)
	}
}

// TestSaveConfigToFileWithErrorInPasswordPrompt should preserve the original file
func TestSaveConfigToFileWithErrorInPasswordPrompt(t *testing.T) {
	c := &Config{
		Name:          "test",
		EncryptConfig: fileEncryptionEnabled,
	}
	testData := []byte("testdata")
	f, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatal(err)
	}
	targetFile := f.Name()
	defer os.Remove(targetFile)

	_, err = io.Copy(f, bytes.NewReader(testData))
	if err != nil {
		t.Fatal(err)
	}
	err = f.Close()
	if err != nil {
		t.Fatal(err)
	}
	err = withInteractiveResponse(t, "\n\n", func() error {
		err = c.SaveConfigToFile(targetFile)
		if err == nil {
			t.Error("Expected error")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	data, err := ioutil.ReadFile(targetFile)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(data, testData) {
		t.Errorf("Expected contents %s, but was %s", testData, data)
	}
}

func withInteractiveResponse(t *testing.T, response string, body func() error) error {
	// Answers to the prompt
	responseFile, err := ioutil.TempFile("", "*.in")
	if err != nil {
		return fmt.Errorf("problem creating temp file: %w", err)
	}
	_, err = responseFile.WriteString(response)
	if err != nil {
		return fmt.Errorf("problem writing to temp file at %s: %w", responseFile.Name(), err)
	}
	err = responseFile.Close()
	if err != nil {
		return fmt.Errorf("problem closing temp file at %s: %w", responseFile.Name(), err)
	}
	defer os.Remove(responseFile.Name())

	// Temporarily replace Stdin with a custom input
	cleanup := setAnswersFile(t, responseFile.Name())
	defer cleanup()
	return body()
}
