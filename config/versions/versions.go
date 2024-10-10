/*
versions handles config upgrades and downgrades

  - Versions must be stateful, and not rely upon type definitions in the config pkg. Instead versions must localise types to avoid issues with subsequent changes

  - Versions must upgrade to the next version. Do not retrospectively change versions to match new type changes. Create a new version

  - Versions must be registered in import.go
*/
package versions

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"

	"github.com/buger/jsonparser"
	"github.com/thrasher-corp/gocryptotrader/common"
)

var (
	errRegisteringVersion  = errors.New("error registering config version")
	errVersionIncompatible = errors.New("version does not implement ConfigVersion or ExchangeVersion")
	errVersionSequence     = errors.New("version registered out of sequence")
	errModifiyingExchange  = errors.New("error modifying exchange config")
	errNoVersions          = errors.New("error retrieving latest config version: No config versions are registered")
	errApplyingVersion     = errors.New("error applying version")
)

// DisabledVersion allows authors to rollback changes easily during development
type DisabledVersion interface {
	Disabled() bool
}

// ConfigVersion is a version that affects the general configuration
type ConfigVersion interface {
	UpgradeConfig(context.Context, []byte) ([]byte, error)
	DowngradeConfig(context.Context, []byte) ([]byte, error)
}

// ExchangeVersion is a version that affects specific exchange configurations
type ExchangeVersion interface {
	Exchanges() []string // Use `*` for all exchanges
	UpgradeExchange(context.Context, []byte) ([]byte, error)
	DowngradeExchange(context.Context, []byte) ([]byte, error)
}

// manager contains versions registerVersioned during import init
type manager struct {
	m        sync.RWMutex
	versions []any
	errors   error
}

// Manager is a public instance of the config version manager
var Manager = &manager{}

// Deploy upgrades or downgrades the config between versions
// Will immediately return any errors encountered in registerVersion calls
func (m *manager) Deploy(ctx context.Context, j []byte) ([]byte, error) {
	if m.errors != nil {
		return j, m.errors
	}

	target, err := m.latest()
	if err != nil {
		return j, err
	}
	m.m.RLock()
	defer m.m.RUnlock()

	current64, err := jsonparser.GetInt(j, "version")
	current := int(current64)
	switch {
	case err == jsonparser.KeyPathNotFoundError:
		current = -1
	case err != nil:
		return j, fmt.Errorf("%w `version`: %w", common.ErrGettingField, err)
	case target == current:
		return j, nil
	}

	for current != target {
		next := current + 1
		action := "upgrade"
		configMethod := ConfigVersion.UpgradeConfig
		exchMethod := ExchangeVersion.UpgradeExchange

		if target < current {
			next = current - 1
			action = "downgrade"
			configMethod = ConfigVersion.DowngradeConfig
			exchMethod = ExchangeVersion.DowngradeExchange
		}

		log.Printf("Running %s to config version %v\n", action, next)

		patch := m.versions[next]

		if cPatch, ok := patch.(ConfigVersion); ok {
			if j, err = configMethod(cPatch, ctx, j); err != nil {
				return j, fmt.Errorf("%w %s to %v: %w", errApplyingVersion, action, next, err)
			}
		}

		if ePatch, ok := patch.(ExchangeVersion); ok {
			if j, err = exchangeDeploy(ctx, ePatch, exchMethod, j); err != nil {
				return j, fmt.Errorf("%w %s to %v: %w", errApplyingVersion, action, next, err)
			}
		}

		current = next

		if j, err = jsonparser.Set(j, []byte(strconv.Itoa(current)), "version"); err != nil {
			return j, fmt.Errorf("%w `version` during %s to %v: %w", common.ErrSettingField, action, next, err)
		}
	}

	log.Println("Version management finished")

	return j, nil
}

func exchangeDeploy(ctx context.Context, patch ExchangeVersion, method func(ExchangeVersion, context.Context, []byte) ([]byte, error), j []byte) ([]byte, error) {
	var errs error
	wanted := patch.Exchanges()
	var i int
	eFunc := func(exchOrig []byte, _ jsonparser.ValueType, _ int, _ error) {
		defer func() { i++ }()
		name, err := jsonparser.GetString(exchOrig, "name")
		if err != nil {
			errs = common.AppendError(errs, fmt.Errorf("%w: %w `name`: %w", errModifiyingExchange, common.ErrGettingField, err))
			return
		}
		for _, want := range wanted {
			if want != "*" && want != name {
				continue
			}
			exchNew, err := method(patch, ctx, exchOrig)
			if err != nil {
				errs = common.AppendError(errs, fmt.Errorf("%w: %w", errModifiyingExchange, err))
				continue
			}
			if !bytes.Equal(exchNew, exchOrig) {
				if j, err = jsonparser.Set(j, exchNew, "exchanges", "["+strconv.Itoa(i)+"]"); err != nil {
					errs = common.AppendError(errs, fmt.Errorf("%w: %w `exchanges.[%d]`: %w", errModifiyingExchange, common.ErrSettingField, i, err))
				}
			}
		}
	}
	v, dataType, _, err := jsonparser.Get(j, "exchanges")
	switch {
	case errors.Is(err, jsonparser.KeyPathNotFoundError), dataType != jsonparser.Array:
		return j, nil
	case err != nil:
		return j, fmt.Errorf("%w: %w `exchanges`: %w", errModifiyingExchange, common.ErrGettingField, err)
	}
	if _, err := jsonparser.ArrayEach(bytes.Clone(v), eFunc); err != nil {
		return j, err
	}
	return j, errs
}

// registerVersion takes instances of config versions and adds them to the registry
// Versions should be added sequentially without gaps, in import.go init
// Any errors will also added to the registry for reporting later
func (m *manager) registerVersion(v any) {
	m.m.Lock()
	defer m.m.Unlock()
	ver, err := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(fmt.Sprintf("%T", v), "*v"), ".Version"))
	if err != nil {
		m.errors = common.AppendError(m.errors, fmt.Errorf("%w '%T': %w", errRegisteringVersion, v, err))
		return
	}
	switch v.(type) {
	case ExchangeVersion, ConfigVersion:
	default:
		m.errors = common.AppendError(m.errors, fmt.Errorf("%w: %v", errVersionIncompatible, ver))
		return
	}
	if len(m.versions) != ver {
		m.errors = common.AppendError(m.errors, fmt.Errorf("%w: %v", errVersionSequence, ver))
		return
	}
	m.versions = append(m.versions, v)
}

// latest returns the highest version number
// May return -1 if something has gone deeply wrong
func (m *manager) latest() (int, error) {
	m.m.RLock()
	defer m.m.RUnlock()
	if len(m.versions) == 0 {
		return 0, errNoVersions
	}
	return len(m.versions) - 1, nil
}
