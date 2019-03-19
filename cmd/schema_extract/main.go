package main

// "github.com/thrasher-/gocryptotrader/database/postgres"

import (
	"flag"
	"os"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/database/postgres"
	"github.com/thrasher-/gocryptotrader/database/sqlite3"
	log "github.com/thrasher-/gocryptotrader/logger"
)

func main() {
	sqlite := flag.Bool("lite", false, "generates sqlite3 schema")
	post := flag.Bool("post", false, "generates postgres schema")
	flag.Parse()

	if !*sqlite && !*post {
		log.Error("No flags set, please set what schema you would like to generate")
		os.Exit(1)
	}

	if *post {
		err := common.WriteFile("postgres.schema", []byte(postgres.GetSchema()))
		if err != nil {
			log.Fatal(err)
		}

		log.Info("PosgreSQL schema file generated")
	}

	if *sqlite {
		err := common.WriteFile("sqlite3.schema", []byte(sqlite3.GetSchema()))
		if err != nil {
			log.Fatal(err)
		}
		log.Info("SQLite3 schema file generated")
	}

}
