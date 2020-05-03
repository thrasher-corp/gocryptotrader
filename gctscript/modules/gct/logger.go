package gct

import (
	"errors"

	"github.com/d5/tengo/v2"
	objects "github.com/d5/tengo/v2"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	info = iota
	debug
	warn
	errorbro
)

var loggerModule = map[string]objects.Object{
	"infoln":  &objects.UserFunction{Name: "infoln", Value: Info},
	"debugln": &objects.UserFunction{Name: "debugln", Value: Debug},
	"warnln":  &objects.UserFunction{Name: "warnln", Value: Warn},
	"errorln": &objects.UserFunction{Name: "errorln", Value: Error},
}

// Info for linking debug output to the internal logger
func Info(args ...objects.Object) (objects.Object, error) {
	return nil, printToGCTLogger(info, args...)
}

// Debug for linking debug output to the internal logger
func Debug(args ...objects.Object) (objects.Object, error) {
	return nil, printToGCTLogger(debug, args...)
}

// Warn for linking warning output to the internal logger
func Warn(args ...objects.Object) (objects.Object, error) {
	return nil, printToGCTLogger(warn, args...)
}

// Error for linking error output to the internal logger
func Error(args ...objects.Object) (objects.Object, error) {
	return nil, printToGCTLogger(errorbro, args...)
}

func getPrintArgs(args ...tengo.Object) ([]interface{}, error) {
	var printArgs []interface{}
	l := 0
	for _, arg := range args {
		s, _ := tengo.ToString(arg)
		slen := len(s)
		// make sure length does not exceed the limit
		if l+slen > tengo.MaxStringLen {
			return nil, tengo.ErrStringLimit
		}
		l += slen
		printArgs = append(printArgs, s)
	}
	return printArgs, nil
}

// printToGCTLogger
func printToGCTLogger(options int, args ...objects.Object) error {
	printArgs, err := getPrintArgs(args...)
	if err != nil {
		return err
	}

	switch options {
	case info:
		log.Infoln(log.GCTScriptMgr, printArgs)
	case debug:
		log.Debugln(log.GCTScriptMgr, printArgs)
	case warn:
		log.Warnln(log.GCTScriptMgr, printArgs)
	case errorbro:
		log.Errorln(log.GCTScriptMgr, printArgs)
	default:
		return errors.New("unhandled logger options")
	}
	return nil
}
