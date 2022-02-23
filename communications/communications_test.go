package communications

import (
	"testing"

	"github.com/thrasher-corp/gocryptotrader/communications/base"
)

func TestNewComm(t *testing.T) {
	var cfg base.CommunicationsConfig
	if _, err := NewComm(&cfg); err == nil {
		t.Error("NewComm should have failed on no enabled communication mediums")
	}

	cfg.TelegramConfig.Enabled = true
	cfg.SMSGlobalConfig.Enabled = true
	cfg.SMTPConfig.Enabled = true
	cfg.SlackConfig.Enabled = true
	communications, err := NewComm(&cfg)
	if err != nil {
		t.Error("Unexpected result")
	}

	if len(communications.IComm) != 4 {
		t.Errorf("communications NewComm, expected len 4, got len %d",
			len(communications.IComm))
	}
}
