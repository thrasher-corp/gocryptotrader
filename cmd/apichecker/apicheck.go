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
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
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
	pathCheckBoardID   = "https://api.trello.com/1/members/%s/boards?key=%s&token=%s"
	complete           = "complete"
	incomplete         = "incomplete"
)

var (
	verbose, add                                                                                                                                                                                                     bool
	apiKey, apiToken, trelloUsername, trelloBoardID, trelloListID, trelloChecklistID, trelloCardID, exchangeName, checkType, tokenData, key, val, tokenDataEnd, textTokenData, dateFormat, regExp, checkString, path string
	configData                                                                                                                                                                                                       Config
)

func main() {
	flag.StringVar(&apiKey, "apikey", "", "sets the API key for Trello board interaction")
	flag.StringVar(&apiToken, "apikoken", "", "sets the API token for Trello board interaction")
	flag.StringVar(&trelloChecklistID, "checklistid", "", "sets the checklist ID for Trello board interaction")
	flag.StringVar(&trelloCardID, "cardid", "", "sets the card ID for Trello board interaction")
	flag.StringVar(&trelloListID, "listid", "", "sets the list ID for Trello board interaction")
	flag.StringVar(&trelloBoardID, "boardid", "", "sets the board ID for Trello board interaction")
	flag.StringVar(&trelloUsername, "trellousername", "", "sets the username for Trello board interaction")
	flag.StringVar(&exchangeName, "exchangename", "", "sets the exchangeName for the new exchange")
	flag.StringVar(&checkType, "checktype", "", "sets the checkType for the new exchange")
	flag.StringVar(&tokenData, "tokendata", "", "sets the tokenData for adding a new exchange")
	flag.StringVar(&key, "key", "", "sets the key for adding a new exchange")
	flag.StringVar(&val, "val", "", "sets the val for adding a new exchange")
	flag.StringVar(&tokenDataEnd, "tokendataend", "", "sets the tokenDataEnd for adding a new exchange")
	flag.StringVar(&textTokenData, "texttokendata", "", "sets the textTokenData for adding a new exchange")
	flag.StringVar(&regExp, "regexp", "", "sets the regExp for adding a new exchange")
	flag.StringVar(&dateFormat, "dateformat", "", "sets the dateFormat for adding a new exchange")
	flag.StringVar(&path, "path", "", "sets the path for adding a new exchange")
	flag.BoolVar(&add, "add", false, "used as a trigger to add a new exchange from command line")
	flag.BoolVar(&verbose, "verbose", false, "increases logging verbosity for API Update Checker")
	flag.Parse()
	var err error
	configData, err = ReadFileData(jsonFile)
	if err != nil {
		log.Fatal(err)
	}
	err = UpdateFile(testJSONFile)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("overwriting test file with the main file data")
	if add {
		switch checkType {
		case github:
			var data GithubData
			data.Repo = path
			err = Add(exchangeName, checkType, data, false)
			if err != nil {
				log.Fatal(err)
			}
		case htmlScrape:
			var data HTMLScrapingData
			data.TokenData = tokenData
			data.Key = key
			data.Val = val
			data.TokenDataEnd = tokenDataEnd
			data.TextTokenData = textTokenData
			data.DateFormat = dateFormat
			data.RegExp = regExp
			data.Path = path
			err = Add(exchangeName, checkType, data, false)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
	if CanUpdateTrello() {
		SetAuthVars()
		err = UpdateFile(backupFile)
		if err != nil {
			log.Fatal(err)
		}
		err = CheckUpdates(jsonFile)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		log.Printf("This is a test update since API keys are not set.\n")
		err := CheckUpdates(testJSONFile)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("API update check completed successfully")
	}
}

// SetAuthVars checks if the cmdline vars are set and sets them onto config file and vice versa
func SetAuthVars() {
	if apiKey == "" {
		apiKey = configData.Key
	} else {
		configData.Key = apiKey
	}
	if apiToken == "" {
		apiToken = configData.Token
	} else {
		configData.Token = apiToken
	}
	if trelloCardID == "" {
		trelloCardID = configData.CardID
	} else {
		configData.CardID = trelloCardID
	}
	if trelloChecklistID == "" {
		trelloChecklistID = configData.ChecklistID
	} else {
		configData.ChecklistID = trelloChecklistID
	}
	if trelloListID == "" {
		trelloListID = configData.ListID
	} else {
		configData.ListID = trelloListID
	}
	if trelloBoardID == "" {
		trelloBoardID = configData.BoardID
	} else {
		configData.BoardID = trelloBoardID
	}
}

// RemoveTestAuthVars removes authenticated variables when the test file is overwritten with the main jsonfile data
func RemoveTestAuthVars() {
	configData.BoardID = ""
	configData.CardID = ""
	configData.ChecklistID = ""
	configData.ListID = ""
	configData.Key = ""
	configData.Token = ""
}

// CanUpdateTrello checks if all the data necessary for updating trello is available
func CanUpdateTrello() bool {
	return (AreAPIKeysSet() && IsTrelloBoardDataSet())
}

// AreAPIKeysSet checks if api keys and tokens are set
func AreAPIKeysSet() bool {
	return (apiKey != "" && apiToken != "") || (configData.Key != "" && configData.Token != "")
}

// IsTrelloBoardDataSet checks if data required to update trello board is set
func IsTrelloBoardDataSet() bool {
	if (trelloBoardID != "" && trelloListID != "" && trelloChecklistID != "" && trelloCardID != "") ||
		(configData.CardID != "" && configData.ChecklistID != "" && configData.BoardID != "" && configData.ListID != "") {
		return true
	}
	return false
}

// getSha gets the sha of the latest commit
func getSha(repoPath string) (ShaResponse, error) {
	var resp ShaResponse
	getPath := fmt.Sprintf(githubPath, repoPath)
	if verbose {
		log.Printf("Getting SHA of this path: %v\n", path)
	}
	return resp, SendGetReq(getPath, &resp)
}

// CheckExistingExchanges checks if the given exchange exists
func CheckExistingExchanges(exchName string) bool {
	for x := range configData.Exchanges {
		if configData.Exchanges[x].Name == exchName {
			return true
		}
	}
	return false
}

// CheckMissingExchanges checks if any supported exchanges are missing api checker functionality
func CheckMissingExchanges() []string {
	var tempArray []string
	for x := range configData.Exchanges {
		tempArray = append(tempArray, configData.Exchanges[x].Name)
	}
	supportedExchs := exchange.Exchanges
	for z := 0; z < len(supportedExchs); {
		if common.StringDataContainsInsensitive(tempArray, supportedExchs[z]) {
			supportedExchs = append(supportedExchs[:z], supportedExchs[z+1:]...)
			continue
		}
		z++
	}
	return supportedExchs
}

// ReadFileData reads the file data from the given json file
func ReadFileData(fileName string) (Config, error) {
	var c Config
	file, err := os.Open(fileName)
	if err != nil {
		return c, err
	}
	defer file.Close()
	byteValue, err := ioutil.ReadAll(file)
	if err != nil {
		return c, err
	}
	err = json.Unmarshal(byteValue, &c)
	if err != nil {
		return c, err
	}
	return c, nil
}

// CheckUpdates checks updates.json for all the existing exchanges
func CheckUpdates(fileName string) error {
	var resp []string
	errMap := make(map[string]error)
	var wg sync.WaitGroup
	var m sync.Mutex
	allExchangeData := configData.Exchanges
	for x := range allExchangeData {
		wg.Add(1)
		go func(e ExchangeInfo) {
			switch e.CheckType {
			case github:
				m.Lock()
				repoPath := e.Data.GitHubData.Repo
				m.Unlock()
				sha, err := getSha(repoPath)
				m.Lock()
				if err != nil && sha.ShaResp != "" {
					errMap[e.Name] = err
				}
				if sha.ShaResp != e.Data.GitHubData.Sha {
					resp = append(resp, e.Name)
					e.Data.GitHubData.Sha = sha.ShaResp
				}
				m.Unlock()
			case htmlScrape:
				checkStr, err := CheckChangeLog(e.Data.HTMLData)
				m.Lock()
				if err != nil {
					errMap[e.Name] = err
				}
				if checkStr != e.Data.HTMLData.CheckString {
					resp = append(resp, e.Name)
					e.Data.HTMLData.CheckString = checkStr
				}
				m.Unlock()
			}
			wg.Done()
		}(allExchangeData[x])
	}
	wg.Wait()
	file, err := json.MarshalIndent(&configData, "", " ")
	if err != nil {
		return err
	}
	var check bool
	if AreAPIKeysSet() {
		check, err = TrelloCheckBoardID(trelloUsername)
		if err != nil {
			return err
		}
		if !check {
			return errors.New("incorrect boardID or api info")
		}
		var a ChecklistItemData
		for y := range resp {
			a, err = TrelloGetChecklistItems()
			if err != nil {
				return err
			}
			for z := range a.CheckItems {
				if strings.Contains(a.CheckItems[z].Name, resp[y]) {
					err = TrelloUpdateCheckItem(a.CheckItems[z].ID, a.CheckItems[z].Name, a.CheckItems[z].State)
					if err != nil {
						return err
					}
				}
			}
		}
	}
	if !AreAPIKeysSet() {
		err = UpdateFile(testJSONFile)
		if err != nil {
			return err
		}
		fileName = testJSONFile
		if verbose {
			log.Println("Updating test file; main file & trello will not be automatically updated since API key & token are not set")
		}
	}
	log.Printf("The following exchanges need an update: %v\n", resp)
	if verbose {
		log.Printf("Errors: %v", errMap)
		unsup := CheckMissingExchanges()
		log.Printf("The following exchanges are not supported by apichecker: %v\n", unsup)
	}
	log.Printf("Saving the updates to the following file: %s\n", fileName)
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
		dataStrings, err = HTMLScrapeOk(htmlData)
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
	}
	return "", errors.New("no response found")
}

// Add appends exchange data to updates.json for future api checks
func Add(exchName, checkType string, data interface{}, update bool) error {
	var file []byte
	if !update {
		if CheckExistingExchanges(exchName) {
			log.Printf("%v exchange already exists\n", exchName)
			return nil
		}
		exchangeData, err := FillData(exchName, checkType, data)
		if err != nil {
			return err
		}
		configData.Exchanges = append(configData.Exchanges, exchangeData)
		file, err = json.MarshalIndent(&configData, "", " ")
		if err != nil {
			return err
		}
	} else {
		info, err := FillData(exchName, checkType, data)
		if err != nil {
			return err
		}
		allExchData := Update(exchName, configData.Exchanges, info)
		configData.Exchanges = allExchData
		file, err = json.MarshalIndent(&configData, "", " ")
		if err != nil {
			return err
		}
	}
	if CanUpdateTrello() {
		if !update {
			err := TrelloCreateNewCheck(fmt.Sprintf("%s 1", exchName))
			if err != nil {
				return err
			}
		}
		return ioutil.WriteFile(jsonFile, file, 0770)
	}
	return ioutil.WriteFile(testJSONFile, file, 0770)
}

// FillData fills exchange data based on the given checkType
func FillData(exchName, checkType string, data interface{}) (ExchangeInfo, error) {
	switch checkType {
	case github:
		tempData := data.(GithubData)
		tempSha, err := getSha(path)
		if err != nil {
			return ExchangeInfo{}, err
		}
		tempData.Sha = tempSha.ShaResp
		return ExchangeInfo{
			Name:      exchName,
			CheckType: checkType,
			Data: &CheckData{
				GitHubData: &tempData,
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
	r, err := regexp.Compile(htmlData.RegExp)
	if err != nil {
		return nil, err
	}
	str := r.FindAllString(string(a), -1)
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

// HTMLScrapeOk gets the check string for Okex
func HTMLScrapeOk(htmlData *HTMLScrapingData) ([]string, error) {
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
							case html.StartTagToken:
								f := tokenizer.Token()
								for _, tkz := range f.Attr {
									if tkz.Key == htmlData.Key {
										r, err := regexp.Compile(htmlData.RegExp)
										if err != nil {
											return resp, err
										}
										result := r.MatchString(tkz.Val)
										if result {
											appendStr := strings.Replace(tkz.Val, htmlData.TokenDataEnd, "", 1)
											resp = append(resp, appendStr)
										}
									}
								}
							case html.EndTagToken:
								tk := tokenizer.Token()
								if tk.Data == "ul" {
									break loop2
								}
							}
						}
					}
				}
			}
		}
	}
	resp = resp[:1]
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
	r, err := regexp.Compile(htmlData.RegExp)
	if err != nil {
		return nil, err
	}
	str := r.FindString(string(a))
	sha := crypto.GetSHA256([]byte(str))
	var resp []string
	resp = append(resp, crypto.HexEncodeToString(sha))
	return resp, nil
}

// TrelloCreateNewCard creates a new card on the list specified on trello
func TrelloCreateNewCard(fillData *CardFill) error {
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
	err := SendAuthReq(http.MethodPost,
		pathNewCard+params.Encode()+apiKey+apiToken,
		nil)
	return err
}

// TrelloCreateNewCheck creates a new checklist item within a given checklist from trello
func TrelloCreateNewCheck(newCheck string) error {
	params := url.Values{}
	params.Set("name", newCheck)
	err := SendAuthReq(http.MethodPost,
		pathChecklists+trelloChecklistID+params.Encode()+apiKey+apiToken,
		nil)
	return err
}

// TrelloCheckBoardID gets board id of the trello boards for a user which can be used for checking if a given board exists
func TrelloCheckBoardID(username string) (bool, error) {
	var data []MembersData
	err := SendAuthReq(http.MethodGet,
		fmt.Sprintf(pathCheckBoardID, username, apiKey, apiToken),
		&data)
	if err != nil {
		return false, err
	}
	if data == nil {
		return false, errors.New("no trello boards available for the given apikey and token")
	}
	for x := range data {
		if trelloBoardID == data[x].ID {
			return true, nil
		}
	}
	return false, nil
}

// TrelloGetChecklistItems get info on all the items on a given checklist from trello
func TrelloGetChecklistItems() (ChecklistItemData, error) {
	var resp ChecklistItemData
	path := fmt.Sprintf(pathChecklistItems, trelloChecklistID, configData.Key, configData.Token)
	return resp, SendGetReq(path, &resp)
}

// NameStateChanges returns the appropriate update name & state for trello (assumes single digit updates pending)
func NameStateChanges(currentName, currentState string) (string, error) {
	r, err := regexp.Compile(`[\s\S]* \d{1}$`) // nolint: gocritic
	if err != nil {
		return "", err
	}
	s, err := regexp.Compile(`[\s\S]* \d{2}$`) // nolint: gocritic
	if err != nil {
		return "", err
	}
	var tempNumber int64
	var finalNumber string
	var byteNumber, byteName []byte
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
		byteNumber = []byte(finalNumber)
		byteName = []byte(currentName)
		byteName = byteName[:len(byteName)-1]
		byteName = append(byteName, byteNumber[0])
		return string(byteName), nil
	}
	if s.MatchString(currentName) {
		stringNumTens := string(currentName[len(currentName)-2])
		stringNumZeros := string(currentName[len(currentName)-1])
		tempTens, err := strconv.ParseInt(stringNumTens, 10, 64)
		if err != nil {
			return "", err
		}
		tempZeros, err := strconv.ParseInt(stringNumZeros, 10, 64)
		if err != nil {
			return "", err
		}
		tempNumber = tempTens*10 + tempZeros + 1
		finalNumber = strconv.FormatInt(tempNumber, 10)
		byteNumber = []byte(finalNumber)
		byteName = []byte(currentName)
		byteName = byteName[:len(byteName)-2]
		byteName = append(byteName, byteNumber[0], byteNumber[1])
		return string(byteName), nil
	}
	return "", errors.New("invalid currentName or pending updates has exceeded 99")
}

// TrelloUpdateCheckItem updates a check item for trello
func TrelloUpdateCheckItem(checkItemID, name, state string) error {
	params := url.Values{}
	newName, err := NameStateChanges(name, state)
	if err != nil {
		return err
	}
	params.Set("name", newName)
	params.Set("state", incomplete)
	path := fmt.Sprintf(pathUpdateItems, trelloCardID, checkItemID, params.Encode(), configData.Key, configData.Token)
	err = SendAuthReq(http.MethodPut, path, nil)
	return err
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
func UpdateFile(name string) error {
	file, err := json.MarshalIndent(&configData, "", " ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(name, file, 0770)
}

// SendGetReq sends get req
func SendGetReq(path string, result interface{}) error {
	var requester *request.Requester
	if strings.Contains(path, "github") {
		requester = request.New("Apichecker",
			common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
			request.NewBasicRateLimit(time.Hour, 60))
	} else {
		requester = request.New("Apichecker",
			common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
			request.NewBasicRateLimit(time.Second, 100))
	}
	return requester.SendPayload(&request.Item{
		Method:  http.MethodGet,
		Path:    path,
		Result:  result,
		Verbose: verbose})
}

// SendAuthReq sends auth req
func SendAuthReq(method, path string, result interface{}) error {
	requester := request.New("Apichecker",
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.NewBasicRateLimit(time.Second*10, 100))
	return requester.SendPayload(&request.Item{
		Method:  method,
		Path:    path,
		Result:  result,
		Verbose: verbose})
}
