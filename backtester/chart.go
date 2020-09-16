package backtest

import (
	"html/template"
	"os"
	"path/filepath"

	"github.com/thrasher-corp/gocryptotrader/common"
)

type ChartData struct {
	PageTitle string
	Pair      string
	EventData []eventData
	TickData  []tickData
}

type eventData struct {
	Timestamp string
}

type tickData struct {
	Timestamp string
	Value     float64
	Price     float64
	Direction string
}

func GenerateOutput(result Results) error {
	wd, _ := os.Getwd()
	outputDir := filepath.Join(wd, "output")
	_ = common.CreateDir(outputDir)
	outputFile := filepath.Join(outputDir,result.StrategyName+".html")
	tmpl := template.Must(template.ParseFiles("template.html"))
	f, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	var tData []tickData
	for x := range result.Transactions {
		tData = append(tData, tickData{
			Timestamp: result.Transactions[x].Time.Format("2006-01-02"),
			Value:     result.Transactions[x].Price,
		})
	}
	var eData []eventData
	for x := range result.Events {
		eData = append(eData, eventData{
			Timestamp: result.Events[x].Time.Format("2006-01-02"),
		})
	}

	d := ChartData{
		PageTitle: "Test",
		Pair:      result.Pair,
		TickData:  tData,
		EventData: eData,
	}

	err = tmpl.Execute(f, d)
	if err != nil {
		return err
	}
	return f.Close()
}
