package backtest

import (
	"html/template"
	"os"
)

type ChartData struct {
	PageTitle string
	ChartJSON []byte
}

func GenerateOutput(j []byte) error {
	tmpl := template.Must(template.ParseFiles("template.html"))
	f, err := os.Create("output.html")
	if err != nil {
		return err
	}
	d := ChartData{
		PageTitle:  "Test",
		ChartJSON: j,
	}
	err = tmpl.Execute(f, d)
	if err != nil {
		return err
	}
	return f.Close()
}