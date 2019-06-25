package communications

import (
	"github.com/idoall/gocryptotrader/communications/base"
	"github.com/idoall/gocryptotrader/communications/slack"
	"github.com/idoall/gocryptotrader/communications/smsglobal"
	"github.com/idoall/gocryptotrader/communications/smtpservice"
	"github.com/idoall/gocryptotrader/communications/telegram"
	"github.com/idoall/gocryptotrader/config"
)

// Communications is the overarching type across the communications packages
type Communications struct {
	base.IComm
}

// NewComm sets up and returns a pointer to a Communications object
func NewComm(cfg *config.CommunicationsConfig) *Communications {
	var comm Communications

	if cfg.TelegramConfig.Enabled {
		Telegram := new(telegram.Telegram)
		Telegram.Setup(cfg)
		comm.IComm = append(comm.IComm, Telegram)
	}

	if cfg.SMSGlobalConfig.Enabled {
		SMSGlobal := new(smsglobal.SMSGlobal)
		SMSGlobal.Setup(cfg)
		comm.IComm = append(comm.IComm, SMSGlobal)
	}

	if cfg.SMTPConfig.Enabled {
		SMTP := new(smtpservice.SMTPservice)
		SMTP.Setup(cfg)
		comm.IComm = append(comm.IComm, SMTP)
	}

	if cfg.SlackConfig.Enabled {
		Slack := new(slack.Slack)
		Slack.Setup(cfg)
		comm.IComm = append(comm.IComm, Slack)
	}

	comm.Setup()
	return &comm
}
