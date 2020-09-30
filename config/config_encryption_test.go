package config

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
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
	isremoved := removeECS(ECStest)

	if string(isremoved) != "" {
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
	c := &Config{}
	c.EncryptConfig = 1
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
	passFile.WriteString("pass\npass\n")
	passFile.Close()

	// Temporarily replace Stdin with a custom input
	oldIn := os.Stdin
	defer func() { os.Stdin = oldIn }()
	os.Stdin, err = os.Open(passFile.Name())
	if err != nil {
		t.Fatalf("Problem opening temp file at %s: %s\n", passFile.Name(), err)
	}

	// Save encrypted config
	enc1 := filepath.Join(tempDir, "encrypted.dat")
	err = c.SaveConfig(enc1, false)
	if err != nil {
		t.Fatalf("Problem storing config in file %s: %s\n", enc1, err)
	}
	// Save again
	enc2 := filepath.Join(tempDir, "encrypted2.dat")
	err = c.SaveConfig(enc2, false)
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
	// legth of prefix + salt
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

	// Prepare password
	passFile, err := ioutil.TempFile(tempDir, "*.pw")
	if err != nil {
		t.Fatalf("Problem creating temp file at %s: %s\n", tempDir, err)
	}
	passFile.WriteString("pass\npass\n")
	passFile.Close()

	// Temporarily replace Stdin with a custom input
	oldIn := os.Stdin
	defer func() { os.Stdin = oldIn }()
	passFile, err = os.Open(passFile.Name())
	if err != nil {
		t.Fatalf("Problem opening temp file at %s: %s\n", passFile.Name(), err)
	}
	defer passFile.Close()
	os.Stdin = passFile

	// Save encrypted config
	enc := filepath.Join(tempDir, "encrypted.dat")
	err = c.SaveConfig(enc, false)
	if err != nil {
		t.Fatalf("Problem storing config in file %s: %s\n", enc, err)
	}

	// Prepare password input for decryption
	passFile, err = os.Open(passFile.Name())
	if err != nil {
		t.Fatalf("Problem opening temp file at %s: %s\n", passFile.Name(), err)
	}
	defer passFile.Close()
	os.Stdin = passFile

	// Clean session
	readConf := &Config{}
	// Load with no existing state, key is read from the prepared file
	err = readConf.ReadConfig(enc, true)

	// Verify
	if err != nil {
		t.Fatalf("Problem reading config in file %s: %s\n", enc, err)
	}

	if c.Name != readConf.Name || c.EncryptConfig != readConf.EncryptConfig {
		t.Error("Loaded conf not the same as original")
	}
}
