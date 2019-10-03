package core

import (
	"github.com/d5/tengo/script"
)

type VM struct {
	Script   *script.Script
	Compiled *script.Compiled
}
