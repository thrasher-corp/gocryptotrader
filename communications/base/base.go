package base

import (
	"time"
)

// Base enforces standard variables across communication packages
type Base struct {
	Name           string
	Enabled        bool
	Verbose        bool
	Connected      bool
	ServiceStarted time.Time
}

// Event is a generalise event type
type Event struct {
	Type    string
	Message string
}

// CommsStatus stores the status of a comms relayer
type CommsStatus struct {
	Enabled   bool `json:"enabled"`
	Connected bool `json:"connected"`
}

// IsEnabled returns if the comms package has been enabled in the configuration
func (b *Base) IsEnabled() bool {
	return b.Enabled
}

// IsConnected returns if the package is connected to a server and/or ready to
// send
func (b *Base) IsConnected() bool {
	return b.Connected
}

// GetName returns a package name
func (b *Base) GetName() string {
	return b.Name
}

// GetStatus returns status data
func (b *Base) GetStatus() string {
	return `
	GoCryptoTrader Service: Online
	Service Started: ` + b.ServiceStarted.UTC().String()
}

// SetServiceStarted sets the time the service started
func (b *Base) SetServiceStarted(t time.Time) {
	b.ServiceStarted = t
}

// CommunicationsConfig holds all the information needed for each
// enabled communication package
type CommunicationsConfig struct {
	SlackConfig     SlackConfig     `json:"slack"`
	SMSGlobalConfig SMSGlobalConfig `json:"smsGlobal"`
	SMTPConfig      SMTPConfig      `json:"smtp"`
	TelegramConfig  TelegramConfig  `json:"telegram"`
}

// IsAnyEnabled returns whether any comms relayers
// are enabled
func (c *CommunicationsConfig) IsAnyEnabled() bool {
	if c.SMSGlobalConfig.Enabled ||
		c.SMTPConfig.Enabled ||
		c.SlackConfig.Enabled ||
		c.TelegramConfig.Enabled {
		return true
	}
	return false
}

// SlackConfig holds all variables to start and run the Slack package
type SlackConfig struct {
	Name              string `json:"name"`
	Enabled           bool   `json:"enabled"`
	Verbose           bool   `json:"verbose"`
	TargetChannel     string `json:"targetChannel"`
	VerificationToken string `json:"verificationToken"`
}

// SMSContact stores the SMS contact info
type SMSContact struct {
	Name    string `json:"name"`
	Number  string `json:"number"`
	Enabled bool   `json:"enabled"`
}

// SMSGlobalConfig structure holds all the variables you need for instant
// messaging and broadcast used by SMSGlobal
type SMSGlobalConfig struct {
	Name     string       `json:"name"`
	From     string       `json:"from"`
	Enabled  bool         `json:"enabled"`
	Verbose  bool         `json:"verbose"`
	Username string       `json:"username"`
	Password string       `json:"password"`
	Contacts []SMSContact `json:"contacts"`
}

// SMTPConfig holds all variables to start and run the SMTP package
type SMTPConfig struct {
	Name            string `json:"name"`
	Enabled         bool   `json:"enabled"`
	Verbose         bool   `json:"verbose"`
	Host            string `json:"host"`
	Port            string `json:"port"`
	AccountName     string `json:"accountName"`
	AccountPassword string `json:"accountPassword"`
	From            string `json:"from"`
	RecipientList   string `json:"recipientList"`
}

// TelegramConfig holds all variables to start and run the Telegram package
type TelegramConfig struct {
	Name              string           `json:"name"`
	Enabled           bool             `json:"enabled"`
	Verbose           bool             `json:"verbose"`
	VerificationToken string           `json:"verificationToken"`
	AuthorisedClients map[string]int64 `json:"authorisedClients"`
}
