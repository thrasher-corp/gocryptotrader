package migrations

import (
	log "github.com/thrasher-corp/gocryptotrader/logger"
)

type MLogger struct{}

func (t MLogger) Printf(format string, v ...interface{}) {
	log.Infof(log.DatabaseMgr, format, v...)
}

func (t MLogger) Println(v ...interface{}) {
	log.Infoln(log.DatabaseMgr, v...)
}

func (t MLogger) Errorf(format string, v ...interface{}) {
	log.Errorf(log.DatabaseMgr, format, v...)
}
