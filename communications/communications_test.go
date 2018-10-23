package communications

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/config"
)

func TestNewComm(t *testing.T) {
	var config config.CommunicationsConfig
	communications := NewComm(config)

	if len(communications.IComm) != 0 {
		t.Errorf("Test failed, communications NewComm, expected len 0, got len %d",
			len(communications.IComm))
	}

	config.TelegramConfig.Enabled = true
	config.SMSGlobalConfig.Enabled = true
	config.SMTPConfig.Enabled = true
	config.SlackConfig.Enabled = true
	communications = NewComm(config)

	if len(communications.IComm) != 4 {
		t.Errorf("Test failed, communications NewComm, expected len 4, got len %d",
			len(communications.IComm))
	}
}
