package config

import (
	"bytes"
	"crypto/aes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPromptForConfigEncryption(t *testing.T) {
	t.Parallel()

	confirm, err := promptForConfigEncryption()
	require.ErrorIs(t, err, io.EOF)
	require.False(t, confirm)
}

func TestPromptForConfigKey(t *testing.T) {
	t.Parallel()

	withInteractiveResponse(t, "\n\n", func() {
		_, err := PromptForConfigKey(false)
		require.ErrorIs(t, err, io.EOF)
	})

	withInteractiveResponse(t, "pass\n", func() {
		k, err := PromptForConfigKey(false)
		require.NoError(t, err)
		assert.Equal(t, "pass", string(k))
	})

	withInteractiveResponse(t, "what\nwhat\n", func() {
		k, err := PromptForConfigKey(true)
		require.NoError(t, err)
		assert.Equal(t, "what", string(k))
	})

	withInteractiveResponse(t, "what\nno\n", func() {
		_, err := PromptForConfigKey(true)
		require.ErrorIs(t, err, io.EOF, "PromptForConfigKey must EOF when asking for another input but none is given")
	})

	withInteractiveResponse(t, "what\nno\nwhat\nno\nwhat\nno\n", func() {
		_, err := PromptForConfigKey(true)
		require.ErrorIs(t, err, io.EOF, "PromptForConfigKey must EOF when asking for another input but none is given")
	})

	withInteractiveResponse(t, "what\nno\nwhat\nno\nwhat\nwhat\n", func() {
		k, err := PromptForConfigKey(true)
		require.NoError(t, err, "PromptForConfigKey must not error if the user eventually answers consistently")
		assert.Equal(t, "what", string(k))
	})
}

func TestEncryptConfigFile(t *testing.T) {
	t.Parallel()
	_, err := EncryptConfigData([]byte("test"), nil)
	require.ErrorIs(t, err, errKeyIsEmpty)

	c := &Config{
		sessionDK: []byte("a"),
	}
	_, err = c.encryptConfigData([]byte(`test`))
	require.ErrorIs(t, err, ErrSettingEncryptConfig)

	_, err = c.encryptConfigData([]byte(`{"test":1}`))
	require.Error(t, err)
	require.IsType(t, aes.KeySizeError(1), err)

	sessDk, salt, err := makeNewSessionDK([]byte("asdf"))
	require.NoError(t, err, "makeNewSessionDK must not error")

	c = &Config{
		sessionDK:  sessDk,
		storedSalt: salt,
	}
	_, err = c.encryptConfigData([]byte(`{"test":1}`))
	require.NoError(t, err)
}

func TestDecryptConfigFile(t *testing.T) {
	t.Parallel()
	e, err := EncryptConfigData([]byte(`{"test":1}`), []byte("key"))
	require.NoError(t, err)

	d, err := DecryptConfigData(e, []byte("key"))
	require.NoError(t, err)
	assert.Equal(t, `{"test":1,"encryptConfig":1}`, string(d), "encryptConfig should be set to 1 after first encryption")

	_, err = DecryptConfigData(e, nil)
	require.ErrorIs(t, err, errKeyIsEmpty)

	_, err = DecryptConfigData([]byte("test"), nil)
	require.ErrorIs(t, err, errNoPrefix)

	_, err = DecryptConfigData(encryptionPrefix, []byte("AAAAAAAAAAAAAAAA"))
	require.ErrorIs(t, err, errAESBlockSize)
}

func TestIsEncrypted(t *testing.T) {
	t.Parallel()
	assert.True(t, IsEncrypted(encryptionPrefix))
	assert.False(t, IsEncrypted([]byte("mhmmm. Donuts.")))
}

func TestMakeNewSessionDK(t *testing.T) {
	t.Parallel()

	if _, _, err := makeNewSessionDK(nil); err == nil {
		t.Fatal("makeNewSessionDK passed with nil key")
	}
}

func TestEncryptTwiceReusesSaltButNewCipher(t *testing.T) {
	c := &Config{
		EncryptConfig: 1,
	}
	tempDir := t.TempDir()

	// Prepare input
	passFile, err := os.CreateTemp(tempDir, "*.pw")
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
	t.Cleanup(func() { os.Stdin = oldIn })
	os.Stdin, err = os.Open(passFile.Name())
	if err != nil {
		t.Fatalf("Problem opening temp file at %s: %s\n", passFile.Name(), err)
	}
	t.Cleanup(func() {
		err = os.Stdin.Close()
		if err != nil {
			t.Error(err)
		}
	})

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
	data1, err := os.ReadFile(enc1)
	if err != nil {
		t.Fatalf("Problem reading file %s: %s\n", enc1, err)
	}
	data2, err := os.ReadFile(enc2)
	if err != nil {
		t.Fatalf("Problem reading file %s: %s\n", enc2, err)
	}
	// length of prefix + salt
	l := len(encryptionPrefix) + len(saltPrefix) + saltRandomLength
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
	tempDir := t.TempDir()

	// Save encrypted config
	enc := filepath.Join(tempDir, "encrypted.dat")
	withInteractiveResponse(t, "pass\npass\n", func() {
		err := c.SaveConfigToFile(enc)
		require.NoError(t, err, "SaveConfigToFile must not error")
	})

	readConf := &Config{}
	withInteractiveResponse(t, "pass\n", func() {
		// Load with no existing state, key is read from the prepared file
		err := readConf.ReadConfigFromFile(enc, true)
		require.NoError(t, err, "ReadConfigFromFile must not error")
	})

	assert.Equal(t, "myCustomName", readConf.Name, "Name should be correct")
	assert.Equal(t, 1, readConf.EncryptConfig, "EncryptConfig should be set correctly")
}

func TestReadConfigWithPrompt(t *testing.T) {
	// Prepare temp dir
	tempDir := t.TempDir()

	// Ensure we'll get the prompt when loading
	c := &Config{
		EncryptConfig: 0,
	}

	// Save config
	testConfigFile := filepath.Join(tempDir, "config.json")
	err := c.SaveConfigToFile(testConfigFile)
	require.NoError(t, err, "SaveConfigToFile must not error")

	// Run the test
	c = &Config{}
	withInteractiveResponse(t, "y\npass\npass\n", func() {
		err = c.ReadConfigFromFile(testConfigFile, false)
		require.NoError(t, err, "ReadConfigFromFile must not error")
	})
	if err != nil {
		t.Fatalf("Problem reading config file at %s: %s\n", testConfigFile, err)
	}

	// Verify results
	data, err := os.ReadFile(testConfigFile)
	if err != nil {
		t.Fatalf("Problem reading saved file at %s: %s\n", testConfigFile, err)
	}
	if c.EncryptConfig != fileEncryptionEnabled {
		t.Error("Config encryption flag should be set after prompts")
	}
	assert.True(t, IsEncrypted(data), "data should be encrypted after prompts")
}

func TestReadEncryptedConfigFromReader(t *testing.T) {
	t.Parallel()
	c := &Config{
		EncryptionKeyProvider: func(_ bool) ([]byte, error) { return []byte("pass"), nil },
	}
	// Encrypted conf for: `{"name":"test"}` with key `pass`
	confBytes := []byte{84, 72, 79, 82, 83, 45, 72, 65, 77, 77, 69, 82, 126, 71, 67, 84, 126, 83, 79, 126, 83, 65, 76, 84, 89, 126, 246, 110, 128, 3, 30, 168, 172, 160, 198, 176, 136, 62, 152, 155, 253, 176, 16, 48, 52, 246, 44, 29, 151, 47, 217, 226, 178, 12, 218, 113, 248, 172, 195, 232, 136, 104, 9, 199, 20, 4, 71, 4, 253, 249}
	err := c.readConfig(bytes.NewReader(confBytes))
	require.NoError(t, err)
	assert.Equal(t, "test", c.Name)

	// Change the salt
	confBytes[20] = 0
	err = c.readConfig(bytes.NewReader(confBytes))
	require.ErrorIs(t, err, errDecryptFailed)
}

// TestSaveConfigToFileWithErrorInPasswordPrompt should preserve the original file
func TestSaveConfigToFileWithErrorInPasswordPrompt(t *testing.T) {
	c := &Config{
		Name:          "test",
		EncryptConfig: fileEncryptionEnabled,
	}
	testData := []byte("testdata")
	f, err := os.CreateTemp("", "")
	require.NoError(t, err, "CreateTemp must not error")

	targetFile := f.Name()
	defer os.Remove(targetFile)

	_, err = io.Copy(f, bytes.NewReader(testData))
	require.NoError(t, err, "io.Copy must not error")
	require.NoError(t, f.Close(), "file Close must not error")

	withInteractiveResponse(t, "\n\n", func() {
		err = c.SaveConfigToFile(targetFile)
		require.ErrorIs(t, err, io.EOF, "SaveConfigToFile must not error")
	})

	data, err := os.ReadFile(targetFile)
	require.NoError(t, err, "ReadFile must not error")
	assert.Equal(t, testData, data)
}

func withInteractiveResponse(tb testing.TB, response string, fn func()) {
	tb.Helper()
	f, err := os.CreateTemp("", "*.in")
	require.NoError(tb, err, "CreateTemp must not error")
	defer f.Close()
	defer os.Remove(f.Name())
	_, err = f.WriteString(response)
	require.NoError(tb, err, "WriteString must not error")
	_, err = f.Seek(0, 0)
	require.NoError(tb, err, "Seek must not error")
	defer func(orig *os.File) { os.Stdin = orig }(os.Stdin)
	os.Stdin = f
	fn()
}
