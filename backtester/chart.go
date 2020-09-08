package backtest

import (
	"html/template"
	"os"
	"time"
)

type ChartData struct {
	PageTitle string

}

type timeData struct {
	Timestamp time.Time
	Value float64
	Price float64
	Direction string
}

func GenerateOutput() error {
	tmpl := template.Must(template.ParseFiles("template.html"))
	f, err := os.Create("output.html")
	if err != nil {
		return err
	}
	d := ChartData{
		PageTitle:  "Test",
	}
	err = tmpl.Execute(f, d)
	if err != nil {
		return err
	}
	return f.Close()
}