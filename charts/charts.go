package charts

import (
	"html/template"
	"io/ioutil"
	"os"
	"path/filepath"
)

func newBasic() Chart {
	return Chart{
		template: "basic.tmpl",
	}
}

func New(template string) Chart {
	switch template {
	case "basic":
		return newBasic()
	}
	return Chart{}
}

func (c *Chart) Generate() error {
	list := []string{
		filepath.Join("templates","base.tmpl"),
		filepath.Join("templates", c.template),
	}

	tmpl, err := template.ParseFiles(list...)
	if err != nil {
		return err
	}

	if c.writeFile {
		c.w, err = os.Create(filepath.Join("output", c.output))
	}
	err = tmpl.Execute(c.w , c.Data)
	if err != nil {
		return err
	}
	return nil
}

func (c *Chart) Result() ([]byte, error) {
	return ioutil.ReadAll(c.w)
}