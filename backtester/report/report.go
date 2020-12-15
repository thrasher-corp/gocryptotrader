package report

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"

	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/statistics"
)

func GenerateReport(s statistics.Statistic) error {
	tmpl := template.Must(template.ParseFiles("tpl.gohtml"))
	file, err := os.Create(
		filepath.Join(
			"..",
			"results",
			fmt.Sprintf(
				"%v%v.html",
				s.StrategyName,
				"", /*time.Now().Format("2006-01-02-15-04-05")*/
			),
		),
	)
	if err != nil {
		return err
	}

	err = tmpl.Execute(file, s)
	if err != nil {
		return err
	}
	return nil
}

func generateCharts