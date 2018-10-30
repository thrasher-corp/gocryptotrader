package main

import (
	"flag"
	"log"
	"time"

	"github.com/pquerna/otp/totp"
	"github.com/thrasher-/gocryptotrader/config"
)

func containsOTP(cfg *config.Config) bool {
	for x := range cfg.Exchanges {
		if cfg.Exchanges[x].API.Credentials.OTPSecret != "" {
			return true
		}
	}
	return false
}

func main() {
	var inFile string
	defaultCfg, err := config.GetFilePath("")
	if err != nil {
		log.Fatal(err)
	}

	flag.StringVar(&inFile, "infile", defaultCfg, "The config input file to process.")
	flag.Parse()

	log.Println("GoCryptoTrader: OTP code generator tool.")

	var cfg config.Config
	err = cfg.LoadConfig(inFile)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Loaded config file.")

	if !containsOTP(&cfg) {
		log.Println("No exchanges with OTP code stored. Exiting.")
	}

	for {
		for x := range cfg.Exchanges {
			if cfg.Exchanges[x].API.Credentials.OTPSecret != "" {
				code, err := totp.GenerateCode(cfg.Exchanges[x].API.Credentials.OTPSecret, time.Now())
				if err != nil {
					log.Printf("Exchange %s: Failed to generate OTP code. Err: %s", cfg.Exchanges[x].Name, err)
					continue
				}
				log.Printf("%s: %s", cfg.Exchanges[x].Name, code)
			}
			time.Sleep(time.Second)
		}
	}
}
