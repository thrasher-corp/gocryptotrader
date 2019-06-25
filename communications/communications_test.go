package communications

import (
	"testing"

	"github.com/idoall/gocryptotrader/config"
)

func TestNewComm(t *testing.T) {
	var cfg config.CommunicationsConfig
	communications := NewComm(&cfg)

	if len(communications.IComm) != 0 {
		t.Errorf("Test failed, communications NewComm, expected len 0, got len %d",
			len(communications.IComm))
	}

	cfg.TelegramConfig.Enabled = true
	cfg.SMSGlobalConfig.Enabled = true
	cfg.SMTPConfig.Enabled = true
	cfg.SlackConfig.Enabled = true
	communications = NewComm(&cfg)

	if len(communications.IComm) != 4 {
		t.Errorf("Test failed, communications NewComm, expected len 4, got len %d",
			len(communications.IComm))
	}
}
