package migrations

import (
	log "github.com/thrasher-corp/gocryptotrader/logger"
)

type MLogger struct{}

// Printf implantation of migration Logger interface
// Passes off to log.Infof
func (t MLogger) Printf(format string, v ...interface{}) {
	log.Infof(log.DatabaseMgr, format, v...)
}

// Println implantation of migration Logger interface
// Passes off to log.Infoln
func (t MLogger) Println(v ...interface{}) {
	log.Infoln(log.DatabaseMgr, v...)
}

// Errorf implantation of migration Logger interface
// Passes off to log.Errorf
func (t MLogger) Errorf(format string, v ...interface{}) {
	log.Errorf(log.DatabaseMgr, format, v...)
}
