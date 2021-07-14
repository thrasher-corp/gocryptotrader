package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
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
	"github.com/thrasher-corp/gocryptotrader/log"
	"golang.org/x/net/html"
)

const (
	githubPath           = "https://api.github.com/repos/%s/commits/master"
	jsonFile             = "updates.json"
	testJSONFile         = "testupdates.json"
	backupFile           = "backup.json"
	github               = "GitHub Sha Check"
	htmlScrape           = "HTML String Check"
	pathBinance          = "https://binance-docs.github.io/apidocs/spot/en/#change-log"
	pathOkCoin           = "https://www.okcoin.com/docs/en/#change-change"
	pathOkex             = "https://www.okex.com/docs/en/#change-change"
	pathFTX              = "https://github.com/ftexchange/ftx"
	pathBTSE             = "https://www.btse.com/apiexplorer/spot/#btse-spot-api"
	pathBitfinex         = "https://docs.bitfinex.com/docs/changelog"
	pathBitmex           = "https://www.bitmex.com/static/md/en-US/apiChangelog"
	pathANX              = "https://anxv3.docs.apiary.io/"
	pathPoloniex         = "https://docs.poloniex.com/#changelog"
	pathIbBit            = "https://api.itbit.com/docs"
	pathBTCMarkets       = "https://api.btcmarkets.net/openapi/info/index.yaml"
	pathEXMO             = "https://exmo.com/en/api/"
	pathBitstamp         = "https://www.bitstamp.net/api/"
	pathHitBTC           = "https://api.hitbtc.com/"
	pathBitflyer         = "https://lightning.bitflyer.com/docs?lang=en"
	pathKraken           = "https://www.kraken.com/features/api"
	pathAlphaPoint       = "https://alphapoint.github.io/slate/#introduction"
	pathYobit            = "https://www.yobit.net/en/api/"
	pathLocalBitcoins    = "https://localbitcoins.com/api-docs/"
	pathGetAllLists      = "https://api.trello.com/1/boards/%s/lists?cards=none&card_fields=all&filter=open&fields=all&key=%s&token=%s"
	pathNewCard          = "https://api.trello.com/1/cards?idList=%s&name=%s&key=%s&token=%s"
	pathChecklists       = "https://api.trello.com/1/checklists/%s/checkItems?%s&key=%s&token=%s"
	pathChecklistItems   = "https://api.trello.com/1/checklists/%s?fields=name&cards=all&card_fields=name&key=%s&token=%s"
	pathUpdateItems      = "https://api.trello.com/1/cards/%s/checkItem/%s?%s&key=%s&token=%s"
	pathCheckBoardID     = "https://api.trello.com/1/members/me/boards?key=%s&token=%s"
	pathNewChecklist     = "https://api.trello.com/1/checklists?idCard=%s&name=%s&key=%s&token=%s"
	pathNewList          = "https://api.trello.com/1/lists?name=%s&idBoard=%s&key=%s&token=%s"
	pathGetCards         = "https://api.trello.com/1/lists/%s/cards?key=%s&token=%s"
	pathGetChecklists    = "https://api.trello.com/1/cards/%s/checklists?&key=%s&token=%s"
	pathDeleteCheckitems = "https://api.trello.com/1/checklists/%s/checkItems/%s?key=%s&token=%s"
	complete             = "complete"
	incomplete           = "incomplete"
	createList           = "UpdatesList"
	createCard           = "UpdatesCard"
	createChecklist      = "UpdatesChecklist"
	btcMarkets           = "BTC Markets"
	okcoin               = "OkCoin International"
)

var (
	verbose, add, create, testMode                                                                                                                                                                                    bool
	apiKey, apiToken, trelloBoardID, trelloBoardName, trelloListID, trelloChecklistID, trelloCardID, exchangeName, checkType, tokenData, key, val, tokenDataEnd, textTokenData, dateFormat, regExp, checkString, path string
	configData, testConfigData, usageData                                                                                                                                                                             Config
)

func main() {
	flag.StringVar(&apiKey, "apikey", "", "sets the API key for Trello board interaction")
	flag.StringVar(&apiToken, "apitoken", "", "sets the API token for Trello board interaction")
	flag.StringVar(&trelloChecklistID, "checklistid", "", "sets the checklist ID for Trello board interaction")
	flag.StringVar(&trelloCardID, "cardid", "", "sets the card ID for Trello board interaction")
	flag.StringVar(&trelloListID, "listid", "", "sets the list ID for Trello board interaction")
	flag.StringVar(&trelloBoardID, "boardid", "", "sets the board ID for Trello board interaction")
	flag.StringVar(&trelloBoardName, "boardname", "", "sets the board name for Trello board interaction")
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
	flag.BoolVar(&create, "create", false, "specifies whether to automatically create trello list, card and checklist in a given board")
	flag.Parse()
	var err error
	c := log.GenDefaultSettings()
	log.RWM.Lock()
	log.GlobalLogConfig = &c
	log.RWM.Unlock()
	log.SetupGlobalLogger()
	configData, err = readFileData(jsonFile)
	if err != nil {
		log.Error(log.Global, err)
		os.Exit(1)
	}
	testConfigData, err = readFileData(testJSONFile)
	if err != nil {
		log.Error(log.Global, err)
		os.Exit(1)
	}
	usageData = testConfigData
	if canUpdateTrello() || (create && areAPIKeysSet()) {
		usageData = configData
	}
	if add {
		switch checkType {
		case github:
			var data GithubData
			data.Repo = path
			err = addExch(exchangeName, checkType, data, false)
			if err != nil {
				log.Error(log.Global, err)
				os.Exit(1)
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
			err = addExch(exchangeName, checkType, data, false)
			if err != nil {
				log.Error(log.Global, err)
				os.Exit(1)
			}
		}
	}
	var a string
	if canUpdateTrello() || create {
		setAuthVars()
		if trelloBoardName != "" {
			a, err = trelloGetBoardID()
			if err != nil {
				log.Error(log.Global, err)
				os.Exit(1)
			}
			trelloBoardID = a
		}
		if create {
			err = createAndSet()
			if err != nil {
				log.Error(log.Global, err)
				os.Exit(1)
			}
		}
		err = updateFile(backupFile)
		if err != nil {
			log.Error(log.Global, err)
			os.Exit(1)
		}
		err = checkUpdates(jsonFile)
		if err != nil {
			log.Error(log.Global, err)
			os.Exit(1)
		}
	} else {
		log.Warnln(log.Global, "This is a test update since API keys are not set.")
		err := checkUpdates(testJSONFile)
		if err != nil {
			log.Error(log.Global, err)
			os.Exit(1)
		}
		log.Infoln(log.Global, "API update check completed successfully")
	}
}

// createAndSet creates and sets the necessary trello board items and sets the authenticated variables accordingly
func createAndSet() error {
	var err error
	err = trelloCreateNewList()
	if err != nil {
		return err
	}
	err = trelloCreateNewCard()
	if err != nil {
		return err
	}
	err = trelloCreateNewChecklist()
	if err != nil {
		return err
	}
	setAuthVars()
	return nil
}

// setAuthVars checks if the cmdline vars are set and sets them onto config file and vice versa
func setAuthVars() {
	if apiKey == "" {
		apiKey = configData.Key
		usageData.Key = configData.Key
	} else {
		configData.Key = apiKey
		usageData.Key = apiKey
	}
	if apiToken == "" {
		apiToken = configData.Token
		usageData.Token = configData.Token
	} else {
		configData.Token = apiToken
		usageData.Token = apiToken
	}
	if trelloCardID == "" {
		trelloCardID = configData.CardID
		usageData.CardID = configData.CardID
	} else {
		configData.CardID = trelloCardID
		usageData.CardID = trelloCardID
	}
	if trelloChecklistID == "" {
		trelloChecklistID = configData.ChecklistID
		usageData.ChecklistID = configData.ChecklistID
	} else {
		configData.ChecklistID = trelloChecklistID
		usageData.ChecklistID = trelloChecklistID
	}
	if trelloListID == "" {
		trelloListID = configData.ListID
		usageData.ListID = configData.ListID
	} else {
		configData.ListID = trelloListID
		usageData.ListID = trelloListID
	}
	if trelloBoardID == "" {
		trelloBoardID = configData.BoardID
		usageData.BoardID = configData.BoardID
	} else {
		configData.BoardID = trelloBoardID
		usageData.BoardID = trelloBoardID
	}
}

// writeAuthVars writes the new authentication variables to the updates.json file
func writeAuthVars(testMode bool) error {
	setAuthVars()
	if testMode {
		return updateFile(testJSONFile)
	}
	return updateFile(jsonFile)
}

// canUpdateTrello checks if all the data necessary for updating trello is available
func canUpdateTrello() bool {
	return areAPIKeysSet() && isTrelloBoardDataSet()
}

// areAPIKeysSet checks if api keys and tokens are set
func areAPIKeysSet() bool {
	return (apiKey != "" && apiToken != "") || (configData.Key != "" && configData.Token != "")
}

// isTrelloBoardDataSet checks if data required to update trello board is set
func isTrelloBoardDataSet() bool {
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
		log.Debugf(log.Global, "Getting SHA of this path: %v\n", getPath)
	}
	return resp, sendGetReq(getPath, &resp)
}

// checkExistingExchanges checks if the given exchange exists
func checkExistingExchanges(exchName string) bool {
	for x := range usageData.Exchanges {
		if usageData.Exchanges[x].Name == exchName {
			return true
		}
	}
	return false
}

// checkMissingExchanges checks if any supported exchanges are missing api checker functionality
func checkMissingExchanges() []string {
	var tempArray []string
	for x := range usageData.Exchanges {
		tempArray = append(tempArray, usageData.Exchanges[x].Name)
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

// readFileData reads the file data from the given json file
func readFileData(fileName string) (Config, error) {
	var c Config
	data, err := ioutil.ReadFile(fileName)
	if err != nil {
		return c, err
	}
	err = json.Unmarshal(data, &c)
	if err != nil {
		return c, err
	}
	return c, nil
}

// checkUpdates checks updates.json for all the existing exchanges
func checkUpdates(fileName string) error {
	var resp []string
	errMap := make(map[string]error)
	var wg sync.WaitGroup
	var m sync.Mutex
	allExchangeData := usageData.Exchanges
	for x := range allExchangeData {
		if allExchangeData[x].Disabled {
			continue
		}
		wg.Add(1)
		go func(e ExchangeInfo) {
			defer wg.Done()
			switch e.CheckType {
			case github:
				m.Lock()
				repoPath := e.Data.GitHubData.Repo
				m.Unlock()
				sha, err := getSha(repoPath)
				m.Lock()
				if err != nil {
					errMap[e.Name] = err
					m.Unlock()
					return
				}
				if sha.ShaResp == "" {
					errMap[e.Name] = errors.New("invalid sha")
					m.Unlock()
					return
				}
				if sha.ShaResp != e.Data.GitHubData.Sha {
					resp = append(resp, e.Name)
					e.Data.GitHubData.Sha = sha.ShaResp
				}
				m.Unlock()
			case htmlScrape:
				checkStr, err := checkChangeLog(e.Data.HTMLData)
				m.Lock()
				if err != nil {
					errMap[e.Name] = err
					m.Unlock()
					return
				}
				if checkStr != e.Data.HTMLData.CheckString {
					resp = append(resp, e.Name)
					e.Data.HTMLData.CheckString = checkStr
				}
				m.Unlock()
			}
		}(allExchangeData[x])
	}
	wg.Wait()
	file, err := json.MarshalIndent(&usageData, "", " ")
	if err != nil {
		return err
	}
	var check bool
	if areAPIKeysSet() {
		check, err = trelloCheckBoardID()
		if err != nil {
			return err
		}
		if !check {
			return errors.New("incorrect boardID or api info")
		}
		var a ChecklistItemData
		for y := range resp {
			a, err = trelloGetChecklistItems()
			if err != nil {
				return err
			}
			var contains bool
			for z := range a.CheckItems {
				if strings.Contains(a.CheckItems[z].Name, resp[y]) {
					err = trelloUpdateCheckItem(a.CheckItems[z].ID, a.CheckItems[z].Name, a.CheckItems[z].State)
					if err != nil {
						return err
					}
					contains = true
				}
			}
			if !contains {
				err = trelloCreateNewCheck(resp[y])
				if err != nil {
					return err
				}
			}
		}
		a, err = trelloGetChecklistItems()
		if err != nil {
			return err
		}
		for l := range a.CheckItems {
			if a.CheckItems[l].State == complete {
				err = trelloDeleteCheckItem(a.CheckItems[l].ID)
				if err != nil {
					return err
				}
			}
		}
	}
	if !areAPIKeysSet() {
		fileName = testJSONFile
		if verbose {
			log.Warnln(log.Global, "Updating test file; main file & trello will not be automatically updated since API key & token are not set")
		}
	}
	log.Warnf(log.Global, "The following exchanges need an update: %v\n", resp)
	for k := range errMap {
		log.Warnf(log.Global, "Error: %v\n", errMap[k])
	}
	unsup := checkMissingExchanges()
	log.Warnf(log.Global, "The following exchanges are not supported by apichecker: %v\n", unsup)
	log.Debugf(log.Global, "Saving the updates to the following file: %s\n", fileName)
	return ioutil.WriteFile(fileName, file, 0770)
}

// checkChangeLog checks the exchanges which support changelog updates.json
func checkChangeLog(htmlData *HTMLScrapingData) (string, error) {
	var dataStrings []string
	var err error
	switch htmlData.Path {
	case pathBinance:
		dataStrings, err = htmlScrapeBinance(htmlData)
	case pathBTSE:
		dataStrings, err = htmlScrapeBTSE(htmlData)
	case pathFTX:
		dataStrings, err = htmlScrapeFTX(htmlData)
	case pathBitfinex:
		dataStrings, err = htmlScrapeBitfinex(htmlData)
	case pathBitmex:
		dataStrings, err = htmlScrapeBitmex(htmlData)
	case pathANX:
		dataStrings, err = htmlScrapeANX(htmlData)
	case pathPoloniex:
		dataStrings, err = htmlScrapePoloniex(htmlData)
	case pathIbBit:
		dataStrings, err = htmlScrapeItBit(htmlData)
	case pathBTCMarkets:
		dataStrings, err = htmlScrapeBTCMarkets(htmlData)
	case pathEXMO:
		dataStrings, err = htmlScrapeExmo(htmlData)
	case pathBitstamp:
		dataStrings, err = htmlScrapeBitstamp(htmlData)
	case pathHitBTC:
		dataStrings, err = htmlScrapeHitBTC(htmlData)
	case pathBitflyer:
		dataStrings, err = htmlScrapeBitflyer(htmlData)
	case pathKraken:
		dataStrings, err = htmlScrapeKraken(htmlData)
	case pathAlphaPoint:
		dataStrings, err = htmlScrapeAlphaPoint(htmlData)
	case pathYobit:
		dataStrings, err = htmlScrapeYobit(htmlData)
	case pathLocalBitcoins:
		dataStrings, err = htmlScrapeLocalBitcoins(htmlData)
	case pathOkCoin, pathOkex:
		dataStrings, err = htmlScrapeOk(htmlData)
	default:
		dataStrings, err = htmlScrapeDefault(htmlData)
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
	return "", fmt.Errorf("no response found for the following path: %s", htmlData.Path)
}

// addExch appends exchange data to updates.json for future api checks
func addExch(exchName, checkType string, data interface{}, isUpdate bool) error {
	var file []byte
	if !isUpdate {
		if checkExistingExchanges(exchName) {
			log.Debugf(log.Global, "%v exchange already exists\n", exchName)
			return nil
		}
		exchangeData, err := fillData(exchName, checkType, data)
		if err != nil {
			return err
		}
		usageData.Exchanges = append(usageData.Exchanges, exchangeData)
		file, err = json.MarshalIndent(&usageData, "", " ")
		if err != nil {
			return err
		}
	} else {
		info, err := fillData(exchName, checkType, data)
		if err != nil {
			return err
		}
		allExchData := update(exchName, usageData.Exchanges, info)
		usageData.Exchanges = allExchData
		file, err = json.MarshalIndent(&usageData, "", " ")
		if err != nil {
			return err
		}
	}
	if canUpdateTrello() {
		if !isUpdate {
			err := trelloCreateNewCheck(fmt.Sprintf("%s 1", exchName))
			if err != nil {
				return err
			}
		}
		return ioutil.WriteFile(jsonFile, file, 0770)
	}
	return ioutil.WriteFile(testJSONFile, file, 0770)
}

// fillData fills exchange data based on the given checkType
func fillData(exchName, checkType string, data interface{}) (ExchangeInfo, error) {
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
		checkStr, err := checkChangeLog(&tempData)
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

// htmlScrapeDefault gets check string data for the default cases
func htmlScrapeDefault(htmlData *HTMLScrapingData) ([]string, error) {
	var resp []string
	temp, err := http.Get(htmlData.Path)
	if err != nil {
		return resp, err
	}
	defer temp.Body.Close()
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
										if r.MatchString(tempStr) {
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

// htmlScrapeBTSE gets the check string for BTSE exchange
func htmlScrapeBTSE(htmlData *HTMLScrapingData) ([]string, error) {
	var resp []string
	temp, err := http.Get(htmlData.Path)
	if err != nil {
		return resp, err
	}
	defer temp.Body.Close()
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

// htmlScrapeBitmex gets the check string for Bitmex exchange
func htmlScrapeBitmex(htmlData *HTMLScrapingData) ([]string, error) {
	var resp []string
	temp, err := http.Get(htmlData.Path)
	if err != nil {
		return resp, err
	}
	defer temp.Body.Close()
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

// htmlScrapeHitBTC gets the check string for HitBTC Exchange
func htmlScrapeHitBTC(htmlData *HTMLScrapingData) ([]string, error) {
	temp, err := http.Get(htmlData.Path)
	if err != nil {
		return nil, err
	}
	defer temp.Body.Close()
	a, err := ioutil.ReadAll(temp.Body)
	if err != nil {
		return nil, err
	}
	aBody := string(a)
	r, err := regexp.Compile(htmlData.RegExp)
	if err != nil {
		return nil, err
	}
	str := r.FindAllString(aBody, -1)
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

// htmlScrapeBTCMarkets gets the check string for BTCMarkets exchange
func htmlScrapeBTCMarkets(htmlData *HTMLScrapingData) ([]string, error) {
	var resp []string
	temp, err := http.Get(htmlData.Path)
	if err != nil {
		return resp, err
	}
	defer temp.Body.Close()
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

// htmlScrapeOk gets the check string for Okex
func htmlScrapeOk(htmlData *HTMLScrapingData) ([]string, error) {
	var resp []string
	temp, err := http.Get(htmlData.Path)
	if err != nil {
		return resp, err
	}
	defer temp.Body.Close()
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

// htmlScrapeANX gets the check string for BTCMarkets exchange
func htmlScrapeANX(htmlData *HTMLScrapingData) ([]string, error) {
	temp, err := http.Get(htmlData.Path)
	if err != nil {
		return nil, err
	}
	defer temp.Body.Close()
	a, err := ioutil.ReadAll(temp.Body)
	if err != nil {
		return nil, err
	}
	aBody := string(a)
	r, err := regexp.Compile(htmlData.RegExp)
	if err != nil {
		return nil, err
	}
	str := r.FindAllString(aBody, -1)
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

// htmlScrapeExmo gets the check string for Exmo Exchange
func htmlScrapeExmo(htmlData *HTMLScrapingData) ([]string, error) {
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
	defer httpResp.Body.Close()
	a, err := ioutil.ReadAll(httpResp.Body)
	if err != nil {
		return nil, err
	}
	aBody := string(a)
	r, err := regexp.Compile(htmlData.RegExp)
	if err != nil {
		return nil, err
	}
	resp := r.FindAllString(aBody, -1)
	return resp, nil
}

// htmlScrapePoloniex gets the check string for Poloniex Exchange
func htmlScrapePoloniex(htmlData *HTMLScrapingData) ([]string, error) {
	var resp []string
	temp, err := http.Get(htmlData.Path)
	if err != nil {
		return resp, err
	}
	defer temp.Body.Close()
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

// htmlScrapeItBit gets the check string for ItBit Exchange
func htmlScrapeItBit(htmlData *HTMLScrapingData) ([]string, error) {
	var resp []string
	temp, err := http.Get(htmlData.Path)
	if err != nil {
		return resp, err
	}
	defer temp.Body.Close()
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

// htmlScrapeBitstamp gets the check string for Bitstamp Exchange
func htmlScrapeBitstamp(htmlData *HTMLScrapingData) ([]string, error) {
	temp, err := http.Get(htmlData.Path)
	if err != nil {
		return nil, err
	}
	defer temp.Body.Close()
	a, err := ioutil.ReadAll(temp.Body)
	if err != nil {
		return nil, err
	}
	aBody := string(a)
	r, err := regexp.Compile(htmlData.RegExp)
	if err != nil {
		return nil, err
	}
	resp := r.FindAllString(aBody, -1)
	return resp, nil
}

// htmlScrapeAlphaPoint gets the check string for Kraken Exchange
func htmlScrapeAlphaPoint(htmlData *HTMLScrapingData) ([]string, error) {
	var resp []string
	temp, err := http.Get(htmlData.Path)
	if err != nil {
		return resp, err
	}
	defer temp.Body.Close()
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

// htmlScrapeYobit gets the check string for Yobit Exchange
func htmlScrapeYobit(htmlData *HTMLScrapingData) ([]string, error) {
	var resp []string
	temp, err := http.Get(htmlData.Path)
	if err != nil {
		return resp, err
	}
	defer temp.Body.Close()
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

// htmlScrapeLocalBitcoins gets the check string for Yobit Exchange
func htmlScrapeLocalBitcoins(htmlData *HTMLScrapingData) ([]string, error) {
	temp, err := http.Get(htmlData.Path)
	if err != nil {
		return nil, err
	}
	defer temp.Body.Close()
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

// trelloCreateNewCheck creates a new checklist item within a given checklist from trello
func trelloCreateNewCheck(newCheckName string) error {
	newName, err := nameStateChanges(newCheckName, "")
	if err != nil {
		return err
	}
	var resp interface{}
	params := url.Values{}
	params.Set("name", newName)
	return sendAuthReq(http.MethodPost,
		fmt.Sprintf(pathChecklists, trelloChecklistID, params.Encode(), apiKey, apiToken),
		&resp)
}

// trelloCheckBoardID gets board id of the trello boards for a user which can be used for checking if a given board exists
func trelloCheckBoardID() (bool, error) {
	var data []MembersData
	err := sendAuthReq(http.MethodGet,
		fmt.Sprintf(pathCheckBoardID, apiKey, apiToken),
		&data)
	if err != nil {
		return false, err
	}
	if data == nil {
		return false, errors.New("no trello boards available for the given apikey and token")
	}
	for x := range data {
		if (trelloBoardID == data[x].ShortID) || (trelloBoardID == data[x].ID) || (trelloBoardName == data[x].Name) {
			return true, nil
		}
	}
	return false, nil
}

// trelloGetChecklistItems get info on all the items on a given checklist from trello
func trelloGetChecklistItems() (ChecklistItemData, error) {
	var resp ChecklistItemData
	path := fmt.Sprintf(pathChecklistItems, trelloChecklistID, apiKey, apiToken)
	return resp, sendGetReq(path, &resp)
}

// nameStateChanges returns the appropriate update name & state for trello
func nameStateChanges(currentName, currentState string) (string, error) {
	name := currentName
	exists := false
	var num int64
	var err error
	switch currentName {
	case btcMarkets, okcoin:
		if strings.Count(currentName, " ") == 2 {
			exists = true
		}
		name = fmt.Sprintf("%s %s", strings.Split(currentName, " ")[0], strings.Split(currentName, " ")[1])
		if !exists {
			return fmt.Sprintf("%s 1", name), nil
		}
		num, err = strconv.ParseInt(strings.Split(currentName, " ")[2], 10, 64)
		if err != nil {
			return "", err
		}
	default:
		if strings.Contains(currentName, " ") {
			exists = true
			name = strings.Split(currentName, " ")[0]
			if !exists {
				return fmt.Sprintf("%s 1", name), nil
			}
			num, err = strconv.ParseInt(strings.Split(currentName, " ")[1], 10, 64)
			if err != nil {
				return "", err
			}
		}
		if !exists {
			return fmt.Sprintf("%s 1", name), nil
		}
	}

	newNum := num
	if currentState == complete {
		newNum = 1
	} else {
		newNum++
	}
	return fmt.Sprintf("%s %d", name, newNum), nil
}

// trelloUpdateCheckItem updates a check item for trello
func trelloUpdateCheckItem(checkItemID, name, state string) error {
	var resp interface{}
	params := url.Values{}
	newName, err := nameStateChanges(name, state)
	if err != nil {
		return err
	}
	params.Set("name", newName)
	params.Set("state", incomplete)
	path := fmt.Sprintf(pathUpdateItems, trelloCardID, checkItemID, params.Encode(), apiKey, apiToken)
	err = sendAuthReq(http.MethodPut, path, &resp)
	return err
}

// update updates the exchange data
func update(currentName string, info []ExchangeInfo, updatedInfo ExchangeInfo) []ExchangeInfo {
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
func updateFile(name string) error {
	file, err := json.MarshalIndent(&configData, "", " ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(name, file, 0770)
}

// SendGetReq sends get req
func sendGetReq(path string, result interface{}) error {
	var requester *request.Requester
	if strings.Contains(path, "github") {
		requester = request.New("Apichecker",
			common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
			request.WithLimiter(request.NewBasicRateLimit(time.Hour, 60)))
	} else {
		requester = request.New("Apichecker",
			common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
			request.WithLimiter(request.NewBasicRateLimit(time.Second, 100)))
	}
	return requester.SendPayload(context.Background(), &request.Item{
		Method:  http.MethodGet,
		Path:    path,
		Result:  result,
		Verbose: verbose})
}

// sendAuthReq sends auth req
func sendAuthReq(method, path string, result interface{}) error {
	requester := request.New("Apichecker",
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(request.NewBasicRateLimit(time.Second*10, 100)))
	return requester.SendPayload(context.Background(), &request.Item{
		Method:  method,
		Path:    path,
		Result:  result,
		Verbose: verbose})
}

// trelloGetBoardID gets all board ids on trello for a given user
func trelloGetBoardID() (string, error) {
	if trelloBoardName == "" {
		return "", errors.New("trelloBoardName not set, cannot find the trelloBoard")
	}
	var resp []TrelloData
	err := sendGetReq(fmt.Sprintf(pathCheckBoardID, apiKey, apiToken),
		&resp)
	if err != nil {
		return "", err
	}
	for x := range resp {
		if resp[x].Name == trelloBoardName {
			return resp[x].ID, err
		}
	}
	return "", errors.New("boardID not found")
}

// trelloGetLists gets all lists from a given board
func trelloGetLists() ([]TrelloData, error) {
	var resp []TrelloData
	return resp, sendGetReq(fmt.Sprintf(pathGetAllLists, trelloBoardID, apiKey, apiToken), &resp)
}

// trelloNewList creates a new list on a specified boards on trello
func trelloCreateNewList() error {
	if trelloBoardID == "" {
		return errors.New("trelloBoardID not set, cannot create a new list")
	}
	var resp interface{}
	listName := createList
	if configData.CreateListName != "" {
		listName = configData.CreateListName
	}
	err := sendAuthReq(http.MethodPost,
		fmt.Sprintf(pathNewList, listName, trelloBoardID, apiKey, apiToken),
		&resp)
	if err != nil {
		return err
	}
	lists, err := trelloGetLists()
	if err != nil {
		return err
	}
	for x := range lists {
		if lists[x].Name != listName {
			continue
		}
		trelloListID = lists[x].ID
		usageData.ListID = lists[x].ID
		err = writeAuthVars(testMode)
		if err != nil {
			return err
		}
		return nil
	}
	return nil
}

// trelloDeleteCheckItem deletes check item from a checklist
func trelloDeleteCheckItem(checkitemID string) error {
	if checkitemID == "" {
		return errors.New("checkitemID cannot be empty")
	}
	var resp interface{}
	return sendAuthReq(http.MethodDelete,
		fmt.Sprintf(pathDeleteCheckitems, trelloChecklistID, checkitemID, apiKey, apiToken),
		&resp)
}

// trelloGetAllCards gets all cards from a given list
func trelloGetAllCards() ([]TrelloData, error) {
	var resp []TrelloData
	return resp, sendGetReq(fmt.Sprintf(pathGetCards, trelloListID, apiKey, apiToken), &resp)
}

// trelloCreateNewCard creates a new card on the list specified on trello
func trelloCreateNewCard() error {
	if trelloListID == "" {
		return errors.New("trelloListID not set, cannot create a new checklist")
	}
	var resp interface{}
	cardName := createCard
	if configData.CreateCardName != "" {
		cardName = configData.CreateCardName
	}
	err := sendAuthReq(http.MethodPost,
		fmt.Sprintf(pathNewCard, trelloListID, cardName, apiKey, apiToken),
		&resp)
	if err != nil {
		return err
	}
	cards, err := trelloGetAllCards()
	if err != nil {
		return err
	}
	for x := range cards {
		if cards[x].Name != cardName {
			continue
		}
		trelloCardID = cards[x].ID
		usageData.CardID = cards[x].ID
		err = writeAuthVars(testMode)
		if err != nil {
			return err
		}
		return nil
	}
	return errors.New("card id and name not found, list creation failed")
}

// trelloGetAllChecklists gets all checklists from a card on trello
func trelloGetAllChecklists() ([]TrelloData, error) {
	var resp []TrelloData
	return resp, sendGetReq(fmt.Sprintf(pathGetChecklists, trelloCardID, apiKey, apiToken), &resp)
}

// trelloCreateNewChecklist creates a new checklist on a specified card on trello
func trelloCreateNewChecklist() error {
	if !areAPIKeysSet() || (trelloCardID == "") {
		return errors.New("apikeys or trelloCardID not set, cannot create a new checklist")
	}
	var resp interface{}
	checklistName := createChecklist
	if configData.CreateChecklistName != "" {
		checklistName = configData.CreateChecklistName
	}
	err := sendAuthReq(http.MethodPost,
		fmt.Sprintf(pathNewChecklist, trelloCardID, checklistName, apiKey, apiToken),
		&resp)
	if err != nil {
		return err
	}
	checklists, err := trelloGetAllChecklists()
	if err != nil {
		return err
	}
	for x := range checklists {
		if checklists[x].Name != checklistName {
			continue
		}
		trelloChecklistID = checklists[x].ID
		usageData.ChecklistID = checklists[x].ID
		err = writeAuthVars(testMode)
		if err != nil {
			return err
		}
		return nil
	}
	return errors.New("checklist id and name not found, checklist creation failed")
}

// htmlScrapeKraken gets the check string for Kraken Exchange
func htmlScrapeKraken(htmlData *HTMLScrapingData) ([]string, error) {
	var resp []string
	temp, err := http.Get(htmlData.Path)
	if err != nil {
		return resp, err
	}
	defer temp.Body.Close()
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

// htmlScrapeBitflyer gets the check string for BTCMarkets exchange
func htmlScrapeBitflyer(htmlData *HTMLScrapingData) ([]string, error) {
	var resp []string
	var tempArray []string
	temp, err := http.Get(htmlData.Path)
	if err != nil {
		return resp, err
	}
	defer temp.Body.Close()
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

// htmlScrapeFTX gets the check string for FTX exchange
func htmlScrapeFTX(htmlData *HTMLScrapingData) ([]string, error) {
	temp, err := http.Get(htmlData.Path)
	if err != nil {
		return nil, err
	}
	defer temp.Body.Close()
	a := temp.Body
	tokenizer := html.NewTokenizer(a)
	var respStr string
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
							anotherToken := tokenizer.Next()
							switch anotherToken {
							case html.StartTagToken:
								z := tokenizer.Token()
								if z.Data == "a" {
									for _, m := range z.Attr {
										if m.Key == "title" {
											switch m.Val {
											case "rest":
											loop3:
												for {
													nextToken := tokenizer.Next()
													switch nextToken {
													case html.StartTagToken:
														f := tokenizer.Token()
														if f.Data == "time-ago" {
															for _, b := range f.Attr {
																if b.Key == "datetime" {
																	respStr += b.Val
																}
															}
														}
													case html.EndTagToken:
														tk := tokenizer.Token()
														if tk.Data == htmlData.TokenDataEnd {
															break loop3
														}
													}
												}
											case "websocket":
											loop4:
												for {
													nextToken := tokenizer.Next()
													switch nextToken {
													case html.StartTagToken:
														f := tokenizer.Token()
														if f.Data == "time-ago" {
															for _, b := range f.Attr {
																if b.Key == "datetime" {
																	respStr += b.Val
																}
															}
														}
													case html.EndTagToken:
														tk := tokenizer.Token()
														if tk.Data == htmlData.TokenDataEnd {
															break loop4
														}
													}
												}
											}
										}
									}
								}
							case html.ErrorToken:
								break loop2
							}
						}
					}
				}
			}
		}
	}
	return []string{respStr}, nil
}

// htmlScrapeBitfinex gets the check string for Bitfinex exchange
func htmlScrapeBitfinex(htmlData *HTMLScrapingData) ([]string, error) {
	temp, err := http.Get(htmlData.Path)
	if err != nil {
		return nil, err
	}
	defer temp.Body.Close()
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

//  htmlScrapeBinance gets checkstring for binance exchange
func htmlScrapeBinance(htmlData *HTMLScrapingData) ([]string, error) {
	temp, err := http.Get(htmlData.Path)
	if err != nil {
		return nil, err
	}
	defer temp.Body.Close()
	tokenizer := html.NewTokenizer(temp.Body)
	var resp []string
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
								nt := tokenizer.Token()
								if nt.Data == htmlData.TokenDataEnd {
									break loop2
								}
							case html.StartTagToken:
								tk := tokenizer.Token()
								if tk.Data == htmlData.TextTokenData {
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
