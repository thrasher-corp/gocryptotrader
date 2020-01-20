package vm

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/hex"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/d5/tengo/v2"
	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	scriptevent "github.com/thrasher-corp/gocryptotrader/database/repository/script"
	"github.com/thrasher-corp/gocryptotrader/gctscript/modules/loader"
	"github.com/thrasher-corp/gocryptotrader/gctscript/wrappers/validator"
	log "github.com/thrasher-corp/gocryptotrader/logger"
	"github.com/volatiletech/null"
)

// NewVM attempts to create a new Virtual Machine firstly from pool
func NewVM() (vm *VM) {
	newUUID, err := uuid.NewV4()
	if err != nil {
		log.Error(log.GCTScriptMgr, Error{
			Action: "New: UUID",
			Cause:  err,
		})
		return nil
	}

	if GCTScriptConfig.Verbose {
		log.Debugln(log.GCTScriptMgr, "New GCTScript VM created")
	}

	vm = &VM{
		ID:     newUUID,
		Script: pool.Get().(*tengo.Script),
	}
	vm.event(StatusSuccess, TypeCreate, false)
	return
}

// Load parses and creates a new instance of tengo script vm
func (vm *VM) Load(file string) error {
	if vm == nil {
		return ErrNoVMLoaded
	}

	if !GCTScriptConfig.Enabled {
		return &Error{
			Action: "Load",
			Cause:  ErrScriptingDisabled,
		}
	}

	if filepath.Ext(file) != ".gct" {
		file += ".gct"
	}

	if GCTScriptConfig.Verbose {
		log.Debugf(log.GCTScriptMgr, "Loading script: %s ID: %v", vm.ShortName(), vm.ID)
	}

	f, err := os.Open(file)
	if err != nil {
		return &Error{
			Action: "Load: Open",
			Script: file,
			Cause:  err,
		}
	}

	defer f.Close()
	code, err := ioutil.ReadAll(f)
	if err != nil {
		return &Error{
			Action: "Load: Read",
			Script: file,
			Cause:  err,
		}
	}

	vm.File = f.Name()
	vm.Path = filepath.Dir(file)
	vm.Script = tengo.NewScript(code)
	vm.Script.SetImports(loader.GetModuleMap())

	if GCTScriptConfig.AllowImports {
		if GCTScriptConfig.Verbose {
			log.Debugf(log.GCTScriptMgr, "File imports enabled for vm: %v", vm.ID)
		}
		vm.Script.EnableFileImport(true)
	}
	vm.event(StatusSuccess, TypeLoad, true)
	return nil
}

// Compile compiles to byte code loaded copy of vm script
func (vm *VM) Compile() (err error) {
	vm.Compiled = new(tengo.Compiled)
	vm.Compiled, err = vm.Script.Compile()
	return
}

// Run runs byte code
func (vm *VM) Run() (err error) {
	if GCTScriptConfig.Verbose {
		log.Debugf(log.GCTScriptMgr, "Running script: %s ID: %v", vm.ShortName(), vm.ID)
	}

	err = vm.Compiled.Run()
	if err != nil {
		vm.event(StatusFailure, TypeExecute, true)
		return Error{
			Action: "Run",
			Cause:  err,
		}
	}
	vm.event(StatusSuccess, TypeExecute, true)
	return
}

// RunCtx runs compiled byte code with context.Context support.
func (vm *VM) RunCtx() (err error) {
	if vm.ctx == nil {
		vm.ctx = context.Background()
	}

	ct, cancel := context.WithTimeout(vm.ctx, GCTScriptConfig.ScriptTimeout)
	defer cancel()

	if GCTScriptConfig.Verbose {
		log.Debugf(log.GCTScriptMgr, "Running script: %s ID: %v", vm.ShortName(), vm.ID)
	}

	err = vm.Compiled.RunContext(ct)
	if err != nil {
		vm.event(StatusFailure, TypeExecute, true)
		return Error{
			Action: "RunCtx",
			Cause:  err,
		}
	}
	vm.event(StatusSuccess, TypeExecute, true)
	return
}

// CompileAndRun Compile and Run script with support for task running
func (vm *VM) CompileAndRun() {
	if vm == nil {
		return
	}
	err := vm.Compile()
	if err != nil {
		log.Error(log.GCTScriptMgr, err)
		err = RemoveVM(vm.ID)
		if err != nil {
			log.Error(log.GCTScriptMgr, err)
		}
		return
	}

	err = vm.RunCtx()
	if err != nil {
		log.Error(log.GCTScriptMgr, err)
		err = RemoveVM(vm.ID)
		if err != nil {
			log.Error(log.GCTScriptMgr, err)
		}
		return
	}
	if vm.Compiled.Get("timer").String() != "" {
		vm.T, err = time.ParseDuration(vm.Compiled.Get("timer").String())
		if err != nil {
			log.Error(log.GCTScriptMgr, err)
			err = vm.Shutdown()
			if err != nil {
				log.Error(log.GCTScriptMgr, err)
			}
			return
		}
		if vm.T < time.Nanosecond {
			log.Error(log.GCTScriptMgr, "Repeat timer cannot be under 1 nano second")
			err = vm.Shutdown()
			if err != nil {
				log.Errorln(log.GCTScriptMgr, err)
			}
			return
		}
		vm.runner()
	} else {
		err = vm.Shutdown()
		if err != nil {
			log.Error(log.GCTScriptMgr, err)
		}
		return
	}
}

// Shutdown shuts down current VMP
func (vm *VM) Shutdown() error {
	if vm.S != nil {
		close(vm.S)
	}
	if GCTScriptConfig.Verbose {
		log.Debugf(log.GCTScriptMgr, "Shutting down script: %s ID: %v", vm.ShortName(), vm.ID)
	}
	vm.Script = nil
	pool.Put(vm.Script)
	vm.event(StatusSuccess, TypeStop, true)
	return RemoveVM(vm.ID)
}

func (vm *VM) Read() ([]byte, error) {
	vm.event(StatusSuccess, TypeRead, true)
	return vm.read()
}

// Read contents of script back
func (vm *VM) read() ([]byte, error) {
	if GCTScriptConfig.Verbose {
		log.Debugf(log.GCTScriptMgr, "Read script: %s ID: %v", vm.ShortName(), vm.ID)
	}
	return ioutil.ReadFile(vm.File)
}

// ShortName returns short (just filename.extension) of running script
func (vm *VM) ShortName() string {
	return filepath.Base(vm.File)
}

func (vm *VM) event(status, executionType string, includeScriptHash bool) {
	if validator.IsTestExecution {
		return
	}

	var hash null.String
	var data null.Bytes
	if includeScriptHash {
		hash.SetValid(vm.getHash(false))
	}
	if executionType != TypeCreate {
		scriptData, err := vm.scriptData()
		if err != nil {
			log.Errorf(log.GCTScriptMgr, "Failed to retrieve scriptData: %v", err)
		}
		data.SetValid(scriptData)
	}
	scriptevent.Event(vm.getHash(true), vm.ShortName(), vm.Path, hash, data, executionType, status, time.Now())
}

func (vm *VM) scriptData() ([]byte, error) {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	f, err := w.Create(vm.ShortName())
	if err != nil {
		return []byte{}, err
	}
	contents, err := vm.read()
	if err != nil {
		return []byte{}, err
	}
	_, err = f.Write(contents)
	if err != nil {
		return []byte{}, err
	}
	err = w.Close()
	if err != nil {
		return []byte{}, err
	}
	return buf.Bytes(), nil
}

func (vm *VM) getHash(includeFileName bool) string {
	contents, err := vm.read()
	if err != nil {
		log.Errorln(log.GCTScriptMgr, err)
	}
	if includeFileName {
		contents = append(contents, vm.ShortName()...)
	}
	return hex.EncodeToString(crypto.GetSHA256(contents))
}
