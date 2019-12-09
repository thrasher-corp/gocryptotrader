package gctscript

import (
	"github.com/thrasher-corp/gocryptotrader/gctscript/gctwrapper"
	"github.com/thrasher-corp/gocryptotrader/gctscript/modules"
)

// Setup configures the wrapper interface to use
func Setup() {
	modules.SetModuleWrapper(gctwrapper.Setup())
}