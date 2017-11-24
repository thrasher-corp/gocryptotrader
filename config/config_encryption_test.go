package config

import (
	"reflect"
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

	byteyBite, err := PromptForConfigKey()
	if err == nil && len(byteyBite) > 1 {
		t.Errorf("Test failed. PromptForConfigKey: %s", err)
	}
}

func TestEncryptDecryptConfigFile(t *testing.T) { //Dual function Test
	testKey := []byte("12345678901234567890123456789012")

	testConfigData, err := common.ReadFile(ConfigTestFile)
	if err != nil {
		t.Errorf("Test failed. EncryptConfigFile: %s", err)
	}
	encryptedFile, err2 := EncryptConfigFile(testConfigData, testKey)
	if err2 != nil {
		t.Errorf("Test failed. EncryptConfigFile: %s", err2)
	}
	if reflect.TypeOf(encryptedFile).String() != "[]uint8" {
		t.Errorf("Test failed. EncryptConfigFile: Incorrect Type")
	}

	decryptedFile, err3 := DecryptConfigFile(encryptedFile, testKey)
	if err3 != nil {
		t.Errorf("Test failed. DecryptConfigFile: %s", err3)
	}
	if reflect.TypeOf(decryptedFile).String() != "[]uint8" {
		t.Errorf("Test failed. DecryptConfigFile: Incorrect Type")
	}
	// unmarshalled := Config{} // racecondition
	// err4 := json.Unmarshal(decryptedFile, &unmarshalled)
	// if err4 != nil {
	// 	t.Errorf("Test failed. DecryptConfigFile: %s", err3)
	// }
}

func TestConfirmConfigJSON(t *testing.T) {
	var result interface{}
	testConfirmJSON, err := common.ReadFile(ConfigTestFile)
	if err != nil {
		t.Errorf("Test failed. testConfirmJSON: %s", err)
	}

	err2 := ConfirmConfigJSON(testConfirmJSON, &result)
	if err2 != nil {
		t.Errorf("Test failed. testConfirmJSON: %s", err2)
	}
	if result == nil {
		t.Errorf("Test failed. testConfirmJSON: Error Unmarshalling JSON")
	}
	err3 := ConfirmConfigJSON(testConfirmJSON, result)
	if err3 == nil {
		t.Errorf("Test failed. testConfirmJSON: %s", err3)
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
