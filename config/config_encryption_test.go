package config

import (
	"testing"
)

func TestPromptForConfigEncryption(t *testing.T) {
	t.Parallel()

	if Cfg.PromptForConfigEncryption("", true) {
		t.Error("PromptForConfigEncryption return incorrect bool")
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

	sessionDK = []byte("a")
	_, err = EncryptConfigFile([]byte("test"), nil)
	if err == nil {
		t.Fatal("Expected error")
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
	isremoved := RemoveECS(ECStest)

	if string(isremoved) != "" {
		t.Errorf("TestConfirmECS: Error ECS not deleted.")
	}
}

func TestMakeNewSessionDK(t *testing.T) {
	t.Parallel()

	_, err := makeNewSessionDK(nil)
	if err == nil {
		t.Fatal("makeNewSessionDK passed with nil key")
	}
}
