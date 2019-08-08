package migrations

import (
	"html/template"
	"path/filepath"
	"strings"
)

func (m *Migrator) LoadMigrations(path string) error {

	path = strings.TrimRight(path, string(filepath.Separator))

	mainTmpl := template.New("main")

	return nil
}
