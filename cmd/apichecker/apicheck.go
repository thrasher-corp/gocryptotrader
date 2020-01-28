package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"golang.org/x/net/html"
)

const (
	githubPath         = "https://api.github.com/repos/%s/commits/master"
	jsonFile           = "updates.json"
	testJSONFile       = "testupdates.json"
	backupFile         = "backup.json"
	github             = "GitHub Sha Check"
	htmlScrape         = "HTML String Check"
	pathOkCoin         = "https://www.okcoin.com/docs/en/#change-change"
	pathOkex           = "https://www.okex.com/docs/en/#change-change"
	pathBTSE           = "https://www.btse.com/apiexplorer/spot/#btse-spot-api"
	pathBitfinex       = "https://docs.bitfinex.com/docs/changelog"
	pathBitmex         = "https://www.bitmex.com/static/md/en-US/apiChangelog"
	pathANX            = "https://anxv3.docs.apiary.io/"
	pathPoloniex       = "https://docs.poloniex.com/#changelog"
	pathIbBit          = "https://api.itbit.com/docs"
	pathBTCMarkets     = "https://api.btcmarkets.net/openapi/info/index.yaml"
	pathEXMO           = "https://exmo.com/en/api/"
	pathBitstamp       = "https://www.bitstamp.net/api/"
	pathHitBTC         = "https://api.hitbtc.com/"
	pathBitflyer       = "https://lightning.bitflyer.com/docs?lang=en"
	pathLakeBTC        = "https://www.lakebtc.com/s/api_v2"
	pathKraken         = "https://www.kraken.com/features/api"
	pathAlphaPoint     = "https://alphapoint.github.io/slate/#introduction"
	pathYobit          = "https://www.yobit.net/en/api/"
	pathLocalBitcoins  = "https://localbitcoins.com/api-docs/"
	pathGetAllLists    = "https://api.trello.com/1/boards/%s/lists?cards=none&card_fields=all&filter=open&fields=all&key=%s&token=%s"
	pathNewCard        = "https://api.trello.com/1/cards?%s&key=%s&token=%s"
	pathChecklists     = "https://api.trello.com/1/checklists/%s/checkItems?%s&key=%s&token=%s"
	pathChecklistItems = "https://api.trello.com/1/checklists/%s?fields=name&cards=all&card_fields=name&key=%s&token=%s"
	pathUpdateItems    = "https://api.trello.com/1/cards/%s/checkItem/%s?%s&key=%s&token=%s"
	trelloKey          = ""
	trelloToken        = ""
	trelloChecklistID  = "5dfc5a5377835d0ba025787a"
	trelloCardID       = "5dfc54b96da13a6ac5ceca97"
	complete           = "complete"
	incomplete         = "incomplete"
)

// Config is a format for storing update data
type Config struct {
	ConfCardID      string         `json:"CardID"`
	ConfChecklistID string         `json:"ChecklistID"`
	ConfKey         string         `json:"Key"`
	ConfToken       string         `json:"Token"`
	Exchanges       []ExchangeInfo `json:"Exchanges"`
}

var (
	verbose                                           bool
	apiKey, apiToken, updateChecklistID, updateCardID string
	configData, testConfigData                        Config
)

func main() {
	flag.StringVar(&apiKey, "key", "", "its an API Key for trello")
	flag.StringVar(&apiToken, "token", "", "its an API Token for trello")
	flag.StringVar(&updateChecklistID, "checklistid", "", "checklist id for trello")
	flag.StringVar(&updateCardID, "cardid", "", "card id for trello")
	flag.BoolVar(&verbose, "verbose", false, "Increases logging verbosity for API Update Checker")
	flag.Parse()
	var err error
	testConfigData, err = ReadFileData(testJSONFile)
	if err != nil {
		log.Fatal(err)
	}
	configData, err = ReadFileData(jsonFile)
	if err != nil {
		log.Fatal(err)
	}
	// Assumption here is that if api key n token are set, then cardid and checklistid will be set too
	if areAPIKeysSet() {
		UpdateFile(&configData, backupFile)
		configData.ConfKey = trelloKey
		configData.ConfToken = trelloToken
		configData.ConfCardID = trelloCardID
		configData.ConfChecklistID = trelloChecklistID
		err = CheckUpdates(jsonFile, &configData)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		log.Printf("This is a test update since API keys are not set.\n")
		err := CheckUpdates(testJSONFile, &testConfigData)
		if err != nil {
			log.Fatal(err)
		}
	}
}

// areAPIKeysSet checks if api keys and tokens are set
func areAPIKeysSet() bool {
	if (trelloKey != "" && trelloKey != "key") && (trelloToken != "" && trelloToken != "token") {
		return true
	}
	return false
}

// getSha gets the sha of the latest commit
func getSha(repoPath string) (ShaResponse, error) {
	var resp ShaResponse
	path := fmt.Sprintf(githubPath, repoPath)
	if verbose {
		fmt.Printf("Getting SHA of this path: %v\n", path)
	}
	return resp, common.SendHTTPGetRequest(path, true, false, &resp)
}

// CheckExistingExchanges checks if the given exchange exists
func CheckExistingExchanges(fileName, exchName string, confData *Config) bool {
	for x := range confData.Exchanges {
		if confData.Exchanges[x].Name == exchName {
			return true
		}
	}
	return false
}

// CheckMissingExchanges checks if any supported exchanges are missing api checker functionality
func CheckMissingExchanges(fileName string, confData *Config) ([]string, error) {
	var tempArray []string
	for x := range confData.Exchanges {
		tempArray = append(tempArray, confData.Exchanges[x].Name)
	}
	supportedExchs := exchange.Exchanges
	for z := 0; z < len(supportedExchs); {
		if common.StringDataContainsInsensitive(tempArray, supportedExchs[z]) {
			supportedExchs = append(supportedExchs[:z], supportedExchs[z+1:]...)
			continue
		}
		z++
	}
	return supportedExchs, nil
}

// ReadFileData reads the file data from the given json file
func ReadFileData(fileName string) (Config, error) {
	var resp Config
	file, err := os.Open(fileName)
	if err != nil {
		return resp, err
	}
	defer file.Close()
	byteValue, err := ioutil.ReadAll(file)
	if err != nil {
		return resp, err
	}
	err = json.Unmarshal(byteValue, &resp)
	if err != nil {
		return resp, err
	}
	return resp, nil
}

// CheckUpdates checks updates.json for all the existing exchanges
func CheckUpdates(fileName string, confData *Config) error {
	var resp []string
	errMap := make(map[string]error)
	var wg sync.WaitGroup
	var sha ShaResponse
	var checkStr string
	var err error
	var m sync.Mutex
	for x := range confData.Exchanges {
		wg.Add(1)
		go func(x int) {
			m.Lock()
			defer m.Unlock()
			defer wg.Done()
			switch confData.Exchanges[x].CheckType {
			case github:
				sha, err = getSha(confData.Exchanges[x].Data.GitHubData.Repo)
				if err != nil {
					errMap[confData.Exchanges[x].Name] = err
				}
				if sha.ShaResp != confData.Exchanges[x].Data.GitHubData.Sha {
					resp = append(resp, confData.Exchanges[x].Name)
					confData.Exchanges[x].Data.GitHubData.Sha = sha.ShaResp
				}
			case htmlScrape:
				checkStr, err = CheckChangeLog(confData.Exchanges[x].Data.HTMLData)
				if err != nil {
					errMap[confData.Exchanges[x].Name] = err
				}
				if checkStr != confData.Exchanges[x].Data.HTMLData.CheckString {
					resp = append(resp, confData.Exchanges[x].Name)
					confData.Exchanges[x].Data.HTMLData.CheckString = checkStr
				}
			}
		}(x)
	}
	wg.Wait()
	file, err := json.MarshalIndent(&confData, "", " ")
	if err != nil {
		return err
	}
	if areAPIKeysSet() {
		var a ChecklistItemData
		for y := range resp {
			a, err = GetChecklistItems()
			if err != nil {
				return err
			}
			for z := range a.CheckItems {
				if strings.Contains(a.CheckItems[z].Name, resp[y]) {
					err = UpdateCheckItem(a.CheckItems[z].ID, a.CheckItems[z].Name, a.CheckItems[z].State)
					if err != nil {
						return err
					}
				}
			}
		}
	}
	if verbose && !areAPIKeysSet() {
		err = UpdateFile(confData, testJSONFile)
		if err != nil {
			return err
		}
		fileName = testJSONFile
		log.Println("Updating test file, main file & trello will not be automatically updated since API key & token are not set")
	}
	if verbose {
		log.Printf("The following exchanges need an update: %v\n", resp)
		log.Printf("Errors: %v", errMap)
		unsup, err := CheckMissingExchanges(fileName, &configData)
		if err != nil {
			return err
		}
		log.Printf("Following are the exchanges that are supported by GCT but not by apichecker: %v\n", unsup)
	}
	return ioutil.WriteFile(fileName, file, 0770)
}

// CheckChangeLog checks the exchanges which support changelog updates.json
func CheckChangeLog(htmlData *HTMLScrapingData) (string, error) {
	var dataStrings []string
	var err error
	switch htmlData.Path {
	case pathBTSE:
		dataStrings, err = HTMLScrapeBTSE(htmlData)
	case pathBitfinex:
		dataStrings, err = HTMLScrapeBitfinex(htmlData)
	case pathBitmex:
		dataStrings, err = HTMLScrapeBitmex(htmlData)
	case pathANX:
		dataStrings, err = HTMLScrapeANX(htmlData)
	case pathPoloniex:
		dataStrings, err = HTMLScrapePoloniex(htmlData)
	case pathIbBit:
		dataStrings, err = HTMLScrapeItBit(htmlData)
	case pathBTCMarkets:
		dataStrings, err = HTMLScrapeBTCMarkets(htmlData)
	case pathEXMO:
		dataStrings, err = HTMLScrapeExmo(htmlData)
	case pathBitstamp:
		dataStrings, err = HTMLScrapeBitstamp(htmlData)
	case pathHitBTC:
		dataStrings, err = HTMLScrapeHitBTC(htmlData)
	case pathBitflyer:
		dataStrings, err = HTMLScrapeBitflyer(htmlData)
	case pathLakeBTC:
		dataStrings, err = HTMLScrapeLakeBTC(htmlData)
	case pathKraken:
		dataStrings, err = HTMLScrapeKraken(htmlData)
	case pathAlphaPoint:
		dataStrings, err = HTMLScrapeAlphaPoint(htmlData)
	case pathYobit:
		dataStrings, err = HTMLScrapeYobit(htmlData)
	case pathLocalBitcoins:
		dataStrings, err = HTMLScrapeLocalBitcoins(htmlData)
	case pathOkCoin, pathOkex:
		dataStrings, err = HTMLScrapeDefault(htmlData)
		for x := range dataStrings {
			if len(dataStrings[x]) != 10 {
				tempStorage := strings.Split(dataStrings[x], "-")
				dataStrings[x] = fmt.Sprintf("%s-0%s-%s", tempStorage[0], tempStorage[1], tempStorage[2])
			}
		}
	default:
		dataStrings, err = HTMLScrapeDefault(htmlData)
	}
	if err != nil {
		return "", err
	}
	switch {
	case len(dataStrings) == 1:
		return dataStrings[0], nil
	case len(dataStrings) > 1:
		x, err := time.Parse(htmlData.DateFormat, dataStrings[0])
		if err != nil {
			return "", err
		}
		y, err := time.Parse(htmlData.DateFormat, dataStrings[len(dataStrings)-1])
		if err != nil {
			return "", err
		}
		z := y.Sub(x)
		switch {
		case z > 0:
			return dataStrings[len(dataStrings)-1], nil
		case z < 0:
			return dataStrings[0], nil
		default:
			return "", errors.New("two or more updates were done on the same day, please check manually")
		}
	default:
	}
	return "", errors.New("no response found")
}

// Add appends exchange data to updates.json for future api checks
func Add(fileName, exchName, checkType, path string, data interface{}, update bool, confData *Config) error {
	check := CheckExistingExchanges(fileName, exchName, &configData)
	var file []byte
	if !update {
		if check {
			if verbose {
				log.Printf("%v exchange already exists\n", exchName)
			}
			return nil
		}
		exchange, err := FillData(exchName, checkType, path, data)
		if err != nil {
			return err
		}
		confData.Exchanges = append(confData.Exchanges, exchange)
		file, err = json.MarshalIndent(&confData, "", " ")
		if err != nil {
			return err
		}
	} else {
		info, err := FillData(exchName, checkType, path, data)
		if err != nil {
			return err
		}
		allExchData := Update(exchName, confData.Exchanges, info)
		if err != nil {
			return err
		}
		confData.Exchanges = allExchData
		file, err = json.MarshalIndent(&confData, "", " ")
		if err != nil {
			return err
		}
	}
	if areAPIKeysSet() {
		return ioutil.WriteFile(jsonFile, file, 0770)
	}
	return ioutil.WriteFile(testJSONFile, file, 0770)
}

// FillData fills exchange data based on the given checkType
func FillData(exchName, checkType, path string, data interface{}) (ExchangeInfo, error) {
	switch checkType {
	case github:
		tempSha, err := getSha(path)
		if err != nil {
			return ExchangeInfo{}, err
		}
		return ExchangeInfo{
			Name:      exchName,
			CheckType: checkType,
			Data: &CheckData{
				GitHubData: &GithubData{
					Repo: path,
					Sha:  tempSha.ShaResp},
			},
		}, nil
	case htmlScrape:
		tempData := data.(HTMLScrapingData)
		checkStr, err := CheckChangeLog(&tempData)
		if err != nil {
			return ExchangeInfo{}, err
		}
		return ExchangeInfo{
			Name:      exchName,
			CheckType: checkType,
			Data: &CheckData{
				HTMLData: &HTMLScrapingData{
					CheckString:   checkStr,
					DateFormat:    tempData.DateFormat,
					Key:           tempData.Key,
					RegExp:        tempData.RegExp,
					TextTokenData: tempData.TextTokenData,
					TokenData:     tempData.TokenData,
					TokenDataEnd:  tempData.TokenDataEnd,
					Val:           tempData.Val,
					Path:          tempData.Path},
			},
		}, nil
	default:
		return ExchangeInfo{}, errors.New("invalid checkType")
	}
}

// HTMLScrapeDefault gets check string data for the default cases
func HTMLScrapeDefault(htmlData *HTMLScrapingData) ([]string, error) {
	var resp []string
	temp, err := http.Get(htmlData.Path)
	if err != nil {
		return resp, err
	}
	tokenizer := html.NewTokenizer(temp.Body)
loop:
	for {
		next := tokenizer.Next()
		switch next {
		case html.ErrorToken:
			break loop
		case html.StartTagToken:
			token := tokenizer.Token()
			if token.Data == htmlData.TokenData {
				for _, a := range token.Attr {
					if a.Key == htmlData.Key && a.Val == htmlData.Val {
					loop2:
						for {
							nextToken := tokenizer.Next()
							switch nextToken {
							case html.EndTagToken:
								t := tokenizer.Token()
								if t.Data == htmlData.TokenDataEnd {
									break loop2
								}
							case html.StartTagToken:
								newtok := tokenizer.Token()
								if newtok.Data == htmlData.TextTokenData {
									inner := tokenizer.Next()
									if inner == html.TextToken {
										tempStr := string(tokenizer.Text())
										r, err := regexp.Compile(htmlData.RegExp)
										if err != nil {
											return resp, err
										}
										result := r.MatchString(tempStr)
										if result {
											appendStr := r.FindString(tempStr)
											resp = append(resp, appendStr)
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}
	return resp, nil
}

// HTMLScrapeBTSE gets the check string for BTSE exchange
func HTMLScrapeBTSE(htmlData *HTMLScrapingData) ([]string, error) {
	var resp []string
	temp, err := http.Get(htmlData.Path)
	if err != nil {
		return resp, err
	}
	tokenizer := html.NewTokenizer(temp.Body)
loop:
	for {
		next := tokenizer.Next()
		switch next {
		case html.ErrorToken:
			break loop
		case html.StartTagToken:
			token := tokenizer.Token()
			if token.Data == htmlData.TokenData {
				for _, z := range token.Attr {
					if z.Key == htmlData.Key && z.Val == htmlData.Val {
						inner := tokenizer.Next()
						if inner == html.TextToken {
							resp = append(resp, string(tokenizer.Text()))
						}
					}
				}
			}
		}
	}
	return resp, nil
}

// HTMLScrapeBitfinex gets the check string for Bitfinex exchange
func HTMLScrapeBitfinex(htmlData *HTMLScrapingData) ([]string, error) {
	temp, err := http.Get(htmlData.Path)
	if err != nil {
		return nil, err
	}
	a, err := ioutil.ReadAll(temp.Body)
	if err != nil {
		return nil, err
	}
	abody := string(a)
	r, err := regexp.Compile(htmlData.RegExp)
	if err != nil {
		return nil, err
	}
	str := r.FindAllString(abody, -1)
	var resp []string
	for x := range str {
		tempStr := strings.Replace(str[x], "section-v-", "", 1)
		var repeat bool
		for y := range resp {
			if tempStr == resp[y] {
				repeat = true
				break
			}
		}
		if !repeat {
			resp = append(resp, tempStr)
		}
	}
	return resp, nil
}

// HTMLScrapeBitmex gets the check string for Bitmex exchange
func HTMLScrapeBitmex(htmlData *HTMLScrapingData) ([]string, error) {
	var resp []string
	temp, err := http.Get(htmlData.Path)
	if err != nil {
		return resp, err
	}
	tokenizer := html.NewTokenizer(temp.Body)
loop:
	for {
		next := tokenizer.Next()
		switch next {
		case html.ErrorToken:
			break loop
		case html.StartTagToken:
			token := tokenizer.Token()
			if token.Data == htmlData.TokenData {
				for _, x := range token.Attr {
					if x.Key != htmlData.Key {
						continue
					}
					tempStr := x.Val
					r, err := regexp.Compile(htmlData.RegExp)
					if err != nil {
						return resp, err
					}
					result := r.MatchString(tempStr)
					if result {
						appendStr := r.FindString(tempStr)
						resp = append(resp, appendStr)
					}
				}
			}
		}
	}
	return resp, nil
}

// HTMLScrapeHitBTC gets the check string for HitBTC Exchange
func HTMLScrapeHitBTC(htmlData *HTMLScrapingData) ([]string, error) {
	temp, err := http.Get(htmlData.Path)
	if err != nil {
		return nil, err
	}
	a, err := ioutil.ReadAll(temp.Body)
	if err != nil {
		return nil, err
	}
	abody := string(a)
	r, err := regexp.Compile(htmlData.RegExp)
	if err != nil {
		return nil, err
	}
	str := r.FindAllString(abody, -1)
	var resp []string
	for x := range str {
		tempStr := strings.Replace(str[x], "section-v-", "", 1)
		var repeat bool
		for y := range resp {
			if tempStr == resp[y] {
				repeat = true
				break
			}
		}
		if !repeat {
			resp = append(resp, tempStr)
		}
	}
	return resp, nil
}

// HTMLScrapeBTCMarkets gets the check string for BTCMarkets exchange
func HTMLScrapeBTCMarkets(htmlData *HTMLScrapingData) ([]string, error) {
	var resp []string
	temp, err := http.Get(htmlData.Path)
	if err != nil {
		return resp, err
	}
	tempData, err := ioutil.ReadAll(temp.Body)
	if err != nil {
		return resp, err
	}
	r, err := regexp.Compile(htmlData.RegExp)
	if err != nil {
		return resp, err
	}
	result := r.FindString(string(tempData))
	resp = append(resp, result)
	return resp, nil
}

// HTMLScrapeBitflyer gets the check string for BTCMarkets exchange
func HTMLScrapeBitflyer(htmlData *HTMLScrapingData) ([]string, error) {
	var resp []string
	var tempArray []string
	temp, err := http.Get(htmlData.Path)
	if err != nil {
		return resp, err
	}
	tokenizer := html.NewTokenizer(temp.Body)
loop:
	for {
		next := tokenizer.Next()
		switch next {
		case html.ErrorToken:
			break loop
		case html.StartTagToken:
			token := tokenizer.Token()
			if token.Data == htmlData.TokenData {
				for {
					nextToken := tokenizer.Next()
					switch nextToken {
					case html.EndTagToken:
						t := tokenizer.Token()
						if t.Data == htmlData.TokenDataEnd {
							break loop
						}
					case html.StartTagToken:
						t := tokenizer.Token()
						if t.Data == htmlData.TextTokenData {
							inner := tokenizer.Next()
							if inner == html.TextToken {
								tempStr := string(tokenizer.Text())
								r, err := regexp.Compile(htmlData.RegExp)
								if err != nil {
									return resp, err
								}
								result := r.MatchString(tempStr)
								if result {
									appendStr := r.FindString(tempStr)
									tempArray = append(tempArray, appendStr)
								}
							}
						}
					}
				}
			}
		}
	}
	resp = append(resp, tempArray[1])
	return resp, nil
}

// HTMLScrapeANX gets the check string for BTCMarkets exchange
func HTMLScrapeANX(htmlData *HTMLScrapingData) ([]string, error) {
	temp, err := http.Get(htmlData.Path)
	if err != nil {
		return nil, err
	}
	a, err := ioutil.ReadAll(temp.Body)
	if err != nil {
		return nil, err
	}
	abody := string(a)
	r, err := regexp.Compile(htmlData.RegExp)
	if err != nil {
		return nil, err
	}
	str := r.FindAllString(abody, -1)
	var resp []string
	for x := range str {
		tempStr := strings.Replace(str[x], "section-v-", "", 1)
		var repeat bool
		for y := range resp {
			if tempStr == resp[y] {
				repeat = true
				break
			}
		}
		if !repeat {
			resp = append(resp, tempStr)
		}
	}
	return resp, nil
}

// HTMLScrapeExmo gets the check string for Exmo Exchange
func HTMLScrapeExmo(htmlData *HTMLScrapingData) ([]string, error) {
	temp, err := http.NewRequest(http.MethodGet, htmlData.Path, nil)
	if err != nil {
		return nil, err
	}
	temp.Header.Set("User-Agent", "GCT")
	httpClient := &http.Client{}
	httpResp, err := httpClient.Do(temp)
	if err != nil {
		return nil, err
	}
	a, err := ioutil.ReadAll(httpResp.Body)
	if err != nil {
		return nil, err
	}
	abody := string(a)
	r, err := regexp.Compile(htmlData.RegExp)
	if err != nil {
		return nil, err
	}
	resp := r.FindAllString(abody, -1)
	return resp, nil
}

// HTMLScrapePoloniex gets the check string for Poloniex Exchange
func HTMLScrapePoloniex(htmlData *HTMLScrapingData) ([]string, error) {
	var resp []string
	temp, err := http.Get(htmlData.Path)
	if err != nil {
		return resp, err
	}
	tokenizer := html.NewTokenizer(temp.Body)
loop:
	for {
		next := tokenizer.Next()
		switch next {
		case html.ErrorToken:
			break loop
		case html.StartTagToken:
			token := tokenizer.Token()
			if token.Data == htmlData.TokenData {
				for _, z := range token.Attr {
					if z.Key == htmlData.Key && z.Val == htmlData.Val {
					loop2:
						for {
							nextToken := tokenizer.Next()
							switch nextToken {
							case html.EndTagToken:
								t := tokenizer.Token()
								if t.Data == htmlData.TokenDataEnd {
									break loop2
								}
							case html.StartTagToken:
								t := tokenizer.Token()
								if t.Data == htmlData.TextTokenData {
									newToken := tokenizer.Next()
									if newToken == html.TextToken {
										tempStr := string(tokenizer.Text())
										r, err := regexp.Compile(htmlData.RegExp)
										if err != nil {
											return resp, err
										}
										result := r.FindString(tempStr)
										resp = append(resp, result)
									}
								}
							}
						}
					}
				}
			}
		}
	}
	return resp, nil
}

// HTMLScrapeItBit gets the check string for ItBit Exchange
func HTMLScrapeItBit(htmlData *HTMLScrapingData) ([]string, error) {
	var resp []string
	temp, err := http.Get(htmlData.Path)
	if err != nil {
		return resp, err
	}
	tokenizer := html.NewTokenizer(temp.Body)
loop:
	for {
		next := tokenizer.Next()
		switch next {
		case html.ErrorToken:
			break loop
		case html.StartTagToken:
			token := tokenizer.Token()
			if token.Data == htmlData.TokenData {
				for _, z := range token.Attr {
					if z.Key == htmlData.Key {
						r, err := regexp.Compile(htmlData.RegExp)
						if err != nil {
							return resp, err
						}
						if r.MatchString(z.Val) {
							resp = append(resp, z.Val)
						}
					}
				}
			}
		}
	}
	return resp, nil
}

// HTMLScrapeLakeBTC gets the check string for LakeBTC Exchange
func HTMLScrapeLakeBTC(htmlData *HTMLScrapingData) ([]string, error) {
	var resp []string
	temp, err := http.Get(htmlData.Path)
	if err != nil {
		return resp, err
	}
	tokenizer := html.NewTokenizer(temp.Body)
loop:
	for {
		next := tokenizer.Next()
		switch next {
		case html.ErrorToken:
			break loop
		case html.StartTagToken:
			token := tokenizer.Token()
			if token.Data == htmlData.TokenData {
				for _, z := range token.Attr {
					if z.Key == htmlData.Key && z.Val == htmlData.Val {
					loop2:
						for {
							nextToken := tokenizer.Next()
							switch nextToken {
							case html.EndTagToken:
								t := tokenizer.Token()
								if t.Data == htmlData.TokenDataEnd {
									break loop2
								}
							case html.StartTagToken:
								t := tokenizer.Token()
								if t.Data == htmlData.TextTokenData {
									inner := tokenizer.Next()
									if inner == html.TextToken {
										tempStr := string(tokenizer.Text())
										r, err := regexp.Compile(htmlData.RegExp)
										if err != nil {
											return resp, err
										}
										if r.MatchString(tempStr) {
											resp = append(resp, tempStr)
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}
	return resp, nil
}

// HTMLScrapeBitstamp gets the check string for Bitstamp Exchange
func HTMLScrapeBitstamp(htmlData *HTMLScrapingData) ([]string, error) {
	temp, err := http.Get(htmlData.Path)
	if err != nil {
		return nil, err
	}
	a, err := ioutil.ReadAll(temp.Body)
	if err != nil {
		return nil, err
	}
	abody := string(a)
	r, err := regexp.Compile(htmlData.RegExp)
	if err != nil {
		return nil, err
	}
	resp := r.FindAllString(abody, -1)
	return resp, nil
}

// HTMLScrapeKraken gets the check string for Kraken Exchange
func HTMLScrapeKraken(htmlData *HTMLScrapingData) ([]string, error) {
	var resp []string
	temp, err := http.Get(htmlData.Path)
	if err != nil {
		return resp, err
	}
	tokenizer := html.NewTokenizer(temp.Body)
loop:
	for {
		next := tokenizer.Next()
		switch next {
		case html.ErrorToken:
			break loop
		case html.StartTagToken:
			token := tokenizer.Token()
			if token.Data == htmlData.TokenData {
				inner := tokenizer.Next()
				if inner == html.TextToken {
					if string(tokenizer.Text()) == "Get account balance" {
					loop2:
						for {
							next := tokenizer.Next()
							switch next {
							case html.EndTagToken:
								t := tokenizer.Token()
								if t.Data == htmlData.TokenDataEnd {
									break loop2
								}
							case html.StartTagToken:
								t := tokenizer.Token()
								if t.Data == htmlData.TextTokenData {
									inside := tokenizer.Next()
									if inside == html.TextToken {
										tempStr := string(tokenizer.Text())
										r, err := regexp.Compile(htmlData.RegExp)
										if err != nil {
											return resp, err
										}
										result := r.MatchString(tempStr)
										if result {
											resp = append(resp, tempStr)
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}
	return resp, nil
}

// HTMLScrapeAlphaPoint gets the check string for Kraken Exchange
func HTMLScrapeAlphaPoint(htmlData *HTMLScrapingData) ([]string, error) {
	var resp []string
	temp, err := http.Get(htmlData.Path)
	if err != nil {
		return resp, err
	}
	tokenizer := html.NewTokenizer(temp.Body)
loop:
	for {
		next := tokenizer.Next()
		switch next {
		case html.ErrorToken:
			break loop
		case html.StartTagToken:
			token := tokenizer.Token()
			if token.Data == htmlData.TokenData {
				for _, x := range token.Attr {
					if x.Key == htmlData.Key && x.Val == htmlData.Val {
					loop2:
						for {
							inner := tokenizer.Next()
							switch inner {
							case html.EndTagToken:
								t := tokenizer.Token()
								if t.Data == htmlData.TokenDataEnd {
									break loop2
								}
							case html.StartTagToken:
								t := tokenizer.Token()
								if t.Data == htmlData.TextTokenData {
									for _, y := range t.Attr {
										if y.Key == htmlData.Key {
											r, err := regexp.Compile(htmlData.RegExp)
											if err != nil {
												return resp, err
											}
											result := r.MatchString(y.Val)
											if result {
												resp = append(resp, y.Val)
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}
	return resp, nil
}

// HTMLScrapeYobit gets the check string for Yobit Exchange
func HTMLScrapeYobit(htmlData *HTMLScrapingData) ([]string, error) {
	var resp []string
	temp, err := http.Get(htmlData.Path)
	if err != nil {
		return resp, err
	}
	tokenizer := html.NewTokenizer(temp.Body)
	var case1, case2, case3 bool
loop:
	for {
		next := tokenizer.Next()
		switch next {
		case html.ErrorToken:
			break loop
		case html.StartTagToken:
			token := tokenizer.Token()
			if token.Data == htmlData.TokenData {
				for _, x := range token.Attr {
					if x.Key == htmlData.Key {
						switch x.Val {
						case "n4":
							inner := tokenizer.Next()
							if inner == html.TextToken {
								if string(tokenizer.Text()) == "v2" {
									case1 = true
								}
							}
						case "n5":
							inner := tokenizer.Next()
							if inner == html.TextToken {
								tempStr := string(tokenizer.Text())
								if tempStr == "v3" {
									case2 = true
									resp = append(resp, tempStr)
								}
							}
						case "n6":
							inner := tokenizer.Next()
							if inner == html.TextToken {
								tempStr := string(tokenizer.Text())
								if tempStr != "v4" {
									case3 = true
								}
								if case1 && case2 && case3 {
									return resp, nil
								}
								resp = append(resp, tempStr)
							}
						}
					}
				}
			}
		}
	}
	return resp, nil
}

// HTMLScrapeLocalBitcoins gets the check string for Yobit Exchange
func HTMLScrapeLocalBitcoins(htmlData *HTMLScrapingData) ([]string, error) {
	temp, err := http.Get(htmlData.Path)
	if err != nil {
		return nil, err
	}
	a, err := ioutil.ReadAll(temp.Body)
	if err != nil {
		return nil, err
	}
	abody := string(a)
	r, err := regexp.Compile(htmlData.RegExp)
	if err != nil {
		return nil, err
	}
	str := r.FindString(abody)
	sha := crypto.GetSHA256([]byte(str))
	var resp []string
	resp = append(resp, crypto.HexEncodeToString(sha))
	return resp, nil
}

// GetListsData gets required data for all the lists on the given board
func GetListsData(idBoard string) ([]ListData, error) {
	var resp []ListData
	err := SendHTTPRequest(pathGetAllLists+idBoard+apiKey+apiToken, &resp)
	if err != nil {
		return resp, err
	}
	return resp, nil
}

// CreateNewCard creates a new card on the list specified
func CreateNewCard(fillData CardFill) error {
	params := url.Values{}
	params.Set("idList", fillData.ListID)
	if fillData.Name != "" {
		params.Set("name", fillData.Name)
	}
	if fillData.Desc != "" {
		params.Set("desc", fillData.Desc)
	}
	if fillData.Pos != "" {
		params.Set("pos", fillData.Pos)
	}
	if fillData.Due != "" {
		params.Set("due", fillData.Due)
	}
	if fillData.MembersID != "" {
		params.Set("idMembers", fillData.MembersID)
	}
	if fillData.LabelsID != "" {
		params.Set("idLabels", fillData.LabelsID)
	}
	_, err := common.SendHTTPRequest(http.MethodPost,
		pathNewCard+params.Encode()+apiKey+apiToken,
		nil,
		nil)
	return err
}

// CreateNewCheck creates a new checklist item within a given checklist
func CreateNewCheck(newCheck string) error {
	params := url.Values{}
	params.Set("name", newCheck)
	_, err := common.SendHTTPRequest(http.MethodPost,
		pathChecklists+updateChecklistID+params.Encode()+apiKey+apiToken,
		nil,
		nil)
	return err
}

// GetChecklistItems get info on all the items on a given checklist
func GetChecklistItems() (ChecklistItemData, error) {
	var resp ChecklistItemData
	path := fmt.Sprintf(pathChecklistItems, trelloChecklistID, configData.ConfKey, configData.ConfToken)
	return resp, common.SendHTTPGetRequest(path, true, verbose, &resp)
}

// NameStateChanges returns the appropriate update name & state for trello (assumes single digit updates pending)
func NameStateChanges(currentName, currentState string) (string, error) {
	r, err := regexp.Compile(`[\s\S]* \d{1}$`) // nolint: gocritic
	if err != nil {
		return "", err
	}
	var tempNumber int64
	var finalNumber string
	if r.MatchString(currentName) {
		stringNum := string(currentName[len(currentName)-1])
		tempNumber, err = strconv.ParseInt(stringNum, 10, 64)
		if err != nil {
			return "", err
		}
		if tempNumber != 1 || currentState != complete {
			tempNumber++
		} else {
			tempNumber = 1
		}
		finalNumber = strconv.FormatInt(tempNumber, 10)
	}
	byteNumber := []byte(finalNumber)
	byteName := []byte(currentName)
	byteName = byteName[:len(byteName)-1]
	byteName = append(byteName, byteNumber[0])
	return string(byteName), nil
}

// UpdateCheckItem updates a check item
func UpdateCheckItem(checkItemID, name, state string) error {
	params := url.Values{}
	newName, err := NameStateChanges(name, state)
	if err != nil {
		return err
	}
	params.Set("name", newName)
	params.Set("state", incomplete)
	path := fmt.Sprintf(pathUpdateItems, trelloCardID, checkItemID, params.Encode(), configData.ConfKey, configData.ConfToken)
	_, err = common.SendHTTPRequest(http.MethodPut, path, nil, nil)
	return err
}

// SendHTTPRequest sends an unauthenticated HTTP request
func SendHTTPRequest(path string, result interface{}) error {
	return common.SendHTTPGetRequest(path, true, verbose, result)
}

// Update updates the exchange data
func Update(currentName string, info []ExchangeInfo, updatedInfo ExchangeInfo) []ExchangeInfo {
	for x := range info {
		if info[x].Name == currentName {
			if info[x].CheckType == updatedInfo.CheckType {
				info[x].Data.GitHubData = updatedInfo.Data.GitHubData
				info[x].Data.HTMLData = updatedInfo.Data.HTMLData
				break
			}
		}
	}
	return info
}

// UpdateFile updates the given file to match updates.json
func UpdateFile(confData *Config, name string) error {
	file, err := json.MarshalIndent(&confData, "", " ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(name, file, 0770)
}
