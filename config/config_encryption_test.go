package config

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/common"
)

func TestPromptForConfigEncryption(t *testing.T) {
	t.Parallel()

	if Cfg.PromptForConfigEncryption() {
		t.Error("Test failed. PromptForConfigEncryption return incorrect bool")
	}
}

func TestPromptForConfigKey(t *testing.T) {
	t.Parallel()

	byteyBite, err := PromptForConfigKey(true)
	if err == nil && len(byteyBite) > 1 {
		t.Errorf("Test failed. PromptForConfigKey: %s", err)
	}

	_, err = PromptForConfigKey(false)
	if err == nil {
		t.Fatal(err)
	}
}

func TestEncryptConfigFile(t *testing.T) {
	_, err := EncryptConfigFile([]byte("test"), nil)
	if err == nil {
		t.Fatal("Test failed. Expected different result")
	}

	sessionDK = []byte("a")
	_, err = EncryptConfigFile([]byte("test"), nil)
	if err == nil {
		t.Fatal("Test failed. Expected different result")
	}

	sessionDK, err = makeNewSessionDK([]byte("asdf"))
	if err != nil {
		t.Fatal(err)
	}

	_, err = EncryptConfigFile([]byte("test"), []byte("key"))
	if err != nil {
		t.Fatal(err)
	}
}

func TestDecryptConfigFile(t *testing.T) {
	sessionDK = nil

	result, err := EncryptConfigFile([]byte("test"), []byte("key"))
	if err != nil {
		t.Fatal(err)
	}

	_, err = DecryptConfigFile(result, nil)
	if err == nil {
		t.Fatal("Test failed. Expected different result")
	}

	_, err = DecryptConfigFile([]byte("test"), nil)
	if err == nil {
		t.Fatal("Test failed. Expected different result")
	}

	_, err = DecryptConfigFile([]byte("test"), []byte("AAAAAAAAAAAAAAAA"))
	if err == nil {
		t.Fatalf("Test failed. Expected %s", errAESBlockSize)
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

func TestConfirmConfigJSON(t *testing.T) {
	var result interface{}
	testConfirmJSON, err := common.ReadFile(ConfigTestFile)
	if err != nil {
		t.Errorf("Test failed. testConfirmJSON: %s", err)
	}

	err = ConfirmConfigJSON(testConfirmJSON, &result)
	if err != nil || result == nil {
		t.Errorf("Test failed. testConfirmJSON: %s", err)
	}
}

func TestConfirmECS(t *testing.T) {
	t.Parallel()

	ECStest := []byte(EncryptConfirmString)
	if !ConfirmECS(ECStest) {
		t.Errorf("Test failed. TestConfirmECS: Error finding ECS.")
	}
}

func TestRemoveECS(t *testing.T) {
	t.Parallel()

	ECStest := []byte(EncryptConfirmString)
	isremoved := RemoveECS(ECStest)

	if string(isremoved) != "" {
		t.Errorf("Test failed. TestConfirmECS: Error ECS not deleted.")
	}
}

func TestMakeNewSessionDK(t *testing.T) {
	t.Parallel()

	_, err := makeNewSessionDK(nil)
	if err == nil {
		t.Fatal("Test failed. makeNewSessionDK passed with nil key")
	}
}
