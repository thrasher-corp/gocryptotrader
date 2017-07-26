package main

import "testing"

func TestEncryptOrDecrypt(t *testing.T) {
	reValue := EncryptOrDecrypt(true)
	if reValue != "encrypted" {
		t.Error(
			"Test failed - Tools/Config/Config_test.go - EncryptOrDecrypt Error",
		)
	}
	reValue = EncryptOrDecrypt(false)
	if reValue != "decrypted" {
		t.Error(
			"Test failed - Tools/Config/Config_test.go - EncryptOrDecrypt Error",
		)
	}
}
