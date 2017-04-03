package test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
)

func TestPromptForConfigEncryption(t *testing.T) {
	t.Parallel()

	promptForConfigEncryption := config.GetConfig()

	if promptForConfigEncryption.PromptForConfigEncryption() {
		t.Error("Test failed. PromptForConfigEncryption return incorrect bool")
	}
}

func TestPromptForConfigKey(t *testing.T) {
	t.Parallel()

	byteyBite, err := config.PromptForConfigKey()
	if err == nil && len(byteyBite) > 1 {
		t.Errorf("Test failed. PromptForConfigKey: %s", err)
	}
}

func TestEncryptDecryptConfigFile(t *testing.T) { //Dual function Test
	t.Parallel()

	testKey := []byte("12345678901234567890123456789012")
	testConfigData, err := common.ReadFile("config.dat")
	if err != nil {
		t.Errorf("Test failed. EncryptConfigFile: %s", err)
	}
	encryptedFile, err2 := config.EncryptConfigFile(testConfigData, testKey)
	if err2 != nil {
		t.Errorf("Test failed. EncryptConfigFile: %s", err2)
	}
	if reflect.TypeOf(encryptedFile).String() != "[]uint8" {
		t.Errorf("Test failed. EncryptConfigFile: Incorrect Type")
	}

	decryptedFile, err3 := config.DecryptConfigFile(encryptedFile, testKey)
	if err3 != nil {
		t.Errorf("Test failed. DecryptConfigFile: %s", err3)
	}
	if reflect.TypeOf(decryptedFile).String() != "[]uint8" {
		t.Errorf("Test failed. DecryptConfigFile: Incorrect Type")
	}
	unmarshalled := config.Config{}
	err4 := json.Unmarshal(decryptedFile, &unmarshalled)
	if err4 != nil {
		t.Errorf("Test failed. DecryptConfigFile: %s", err3)
	}
}

func TestConfirmJson(t *testing.T) {
	t.Parallel()

	var result interface{}
	testConfirmJson, err := common.ReadFile("config.dat")
	if err != nil {
		t.Errorf("Test failed. TestConfirmJson: %s", err)
	}
	err2 := config.ConfirmConfigJSON(testConfirmJson, &result)
	if err2 != nil {
		t.Errorf("Test failed. TestConfirmJson: %s", err2)
	}
	if result == nil {
		t.Errorf("Test failed. TestConfirmJson: Error Unmarshalling JSON")
	}
}

func TestConfirmECS(t *testing.T) {
	t.Parallel()

	ECStest := []byte("THORS-HAMMER")
	if !config.ConfirmECS(ECStest) {
		t.Errorf("Test failed. TestConfirmECS: Error finding ECS.")
	}
}

func TestRemoveECS(t *testing.T) {
	t.Parallel()

	ECStest := []byte("THORS-HAMMER")
	isremoved := config.RemoveECS(ECStest)

	if string(isremoved) != "" {
		t.Errorf("Test failed. TestConfirmECS: Error ECS not deleted.")
	}
}
