package gctscript

import (
	"github.com/thrasher-corp/gocryptotrader/gctscript/gctwrapper"
	"github.com/thrasher-corp/gocryptotrader/gctscript/modules"
)

func Setup() {
	modules.SetModuleWrapper(gctwrapper.Setup())
}
