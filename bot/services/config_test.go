package services

import "testing"

var c Configuration

func TestEncryptOrDecrypt(t *testing.T) {
	reValue := c.EncryptOrDecrypt(true)
	if reValue != "encrypted" {
		t.Error(
			"Test failed - Tools/Config/Config_test.go - EncryptOrDecrypt Error",
		)
	}
	reValue = c.EncryptOrDecrypt(false)
	if reValue != "decrypted" {
		t.Error(
			"Test failed - Tools/Config/Config_test.go - EncryptOrDecrypt Error",
		)
	}
}
