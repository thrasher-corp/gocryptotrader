package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/pquerna/otp/totp"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/core"
)

const defaultSleepTime = time.Second * 30

func containsOTP(cfg *config.Config) bool {
	for x := range cfg.Exchanges {
		if cfg.Exchanges[x].API.Credentials.OTPSecret != "" {
			return true
		}
	}
	return false
}

func main() {
	var cfgFile, code string
	var single bool
	var err error

	flag.StringVar(&cfgFile, "config", config.DefaultFilePath(), "The config input file to process.")
	flag.BoolVar(&single, "single", false, "prompt for single use OTP code gen")
	flag.Parse()

	log.Println("GoCryptoTrader: OTP code generator tool.")
	log.Println(core.Copyright)

	// Handle single use OTP code gen
	if single {
		var input string
		for {
			log.Println("Please enter in your OTP secret:")
			if _, err = fmt.Scanln(&input); err != nil {
				log.Printf("Failed to read input. Err: %s\n", err)
				continue
			}
			if input != "" {
				break
			}
		}

		for {
			code, err = totp.GenerateCode(input, time.Now())
			if err != nil {
				log.Fatalf("Unable to generate OTP code. Err: %s", err)
			}
			log.Printf("OTP code: %s\n", code)
			time.Sleep(defaultSleepTime)
		}
	}

	// Otherwise default to loading the config file and generating OTP codes from it
	var cfg config.Config
	err = cfg.LoadConfig(cfgFile, true)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Loaded config file.")

	if !containsOTP(&cfg) {
		log.Fatal("No exchanges with OTP code stored. Exiting.")
	}

	for {
		for x := range cfg.Exchanges {
			if cfg.Exchanges[x].API.Credentials.OTPSecret != "" {
				code, err = totp.GenerateCode(cfg.Exchanges[x].API.Credentials.OTPSecret, time.Now())
				if err != nil {
					log.Printf("Exchange %s: Failed to generate OTP code. Err: %s\n", cfg.Exchanges[x].Name, err)
					continue
				}
				log.Printf("%s: %s\n", cfg.Exchanges[x].Name, code)
			}
		}
		time.Sleep(defaultSleepTime)
	}
}
