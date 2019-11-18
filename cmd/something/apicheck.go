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
	"strings"
	"time"

	"golang.org/x/net/html"

	"github.com/thrasher-corp/gocryptotrader/common"
)

const (
	path              = "https://api.github.com/repos/%s/commits/master"
	file              = "Updates.json"
	github            = "GitHub Sha Check"
	htmlScrape        = "HTML String Check"
	pathOkCoin        = "https://www.okcoin.com/docs/en/#change-change"
	pathOkex          = "https://www.okex.com/docs/en/#change-change"
	pathBTSE          = "https://www.btse.com/apiexplorer/spot/#btse-spot-api"
	pathBitfinex      = "https://docs.bitfinex.com/docs/changelog"
	pathBitmex        = "https://www.bitmex.com/static/md/en-US/apiChangelog"
	pathANX           = "https://anxv3.docs.apiary.io/"
	pathPoloniex      = "https://docs.poloniex.com/#changelog"
	pathIbBit         = "https://api.itbit.com/docs"
	pathBTCMarkets    = "https://api.btcmarkets.net/openapi/info/index.yaml"
	pathEXMO          = "https://exmo.com/en/api/"
	pathBitstamp      = "https://www.bitstamp.net/api/"
	pathHitBTC        = "https://api.hitbtc.com/"
	pathBitflyer      = "https://lightning.bitflyer.com/docs?lang=en"
	pathLakeBTC       = "https://www.lakebtc.com/s/api_v2"
	pathKraken        = "https://www.kraken.com/features/api"
	pathGetAllLists   = "https://api.trello.com/1/boards/%s/lists?cards=none&card_fields=all&filter=open&fields=all&key=%s&token=%s"
	pathNewCard       = "https://api.trello.com/1/cards?%s&key=%s&token=%s"
	pathChecklists    = "https://api.trello.com/1/checklists/%s/checkItems?%s&key=%s&token=%s"
	apiKey            = ""
	apiToken          = ""
	updateCardID      = "5dfc54b96da13a6ac5ceca97"
	updateChecklistID = "5dfc5a5377835d0ba025787a"
)

var verbose bool

func main() {
	flag.BoolVar(&verbose, "verbose", false, "increases logging verbosity for GoCryptoTrader")
	flag.Parse()
	updates, err := CheckUpdates(file)
	if err != nil {
		log.Println(err)
	}
	log.Println(updates)
	// update - exchange name / apitype / reposiroty
}

// GetSha gets the sha of the latest commit
func GetSha(repoPath string) (ShaResponse, error) {
	var resp ShaResponse
	finalPath := fmt.Sprintf(path, repoPath)
	if verbose {
		log.Println(fmt.Sprintf(path, repoPath))
	}
	return resp, common.SendHTTPGetRequest(finalPath, true, false, &resp)
}

// CheckExistingExchanges checks if the given exchange exists
func CheckExistingExchanges(fileName, exchName string) ([]ExchangeInfo, bool, error) {
	var resp bool
	var data []ExchangeInfo
	var err error
	data, err = ReadFileData(fileName)
	if err != nil {
		return data, resp, err
	}
	for x := range data {
		if data[x].Name == exchName {
			resp = true
			break
		}
	}
	return data, resp, nil
}

// ReadFileData reads the file data for the json file
func ReadFileData(fileName string) ([]ExchangeInfo, error) {
	var data []ExchangeInfo
	jsonFile, err := os.Open(fileName)
	if err != nil {
		return data, err
	}
	defer jsonFile.Close()
	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return data, err
	}
	json.Unmarshal(byteValue, &data)
	return data, nil
}

// CheckUpdates checks Updates.json for all the existing exchanges
func CheckUpdates(fileName string) ([]string, error) {
	var resp []string
	data, err := ReadFileData(fileName)
	if err != nil {
		return resp, err
	}
	for x := range data {
		switch data[x].CheckType {
		case github:
			sha, err := GetSha(data[x].Data.GitHubData.Repo)
			if err != nil {
				return resp, err
			}
			if sha.ShaResp != data[x].Data.GitHubData.Sha {
				if verbose {
					log.Printf("%s api needs to be updated", data[x].Name)
				}
				data[x].Data.GitHubData.Sha = sha.ShaResp
				continue
			}
		case htmlScrape:
			checkStr, err := CheckChangeLog(*data[x].Data.HTMLData)
			if err != nil {
				return resp, err
			}
			if checkStr == data[x].Data.HTMLData.CheckString {
				continue
			}
			resp = append(resp, data[x].Name)
			data[x].Data.HTMLData.CheckString = checkStr
		}
	}
	file, err := json.MarshalIndent(data, "", " ")
	if err != nil {
		return resp, err
	}
	return resp, ioutil.WriteFile(fileName, []byte(file), 0644)
}

// CheckChangeLog checks the exchanges which support changelog Updates.json
func CheckChangeLog(htmlData HTMLScrapingData) (string, error) {
	var stringsss []string
	var err error
	switch htmlData.Path {
	case pathBTSE:
		stringsss, err = HTMLScrapeBTSE(htmlData)
		if err != nil {
			return "", err
		}
		return stringsss[0], nil
	case pathBitfinex:
		stringsss, err = HTMLScrapeBitfinex(htmlData)
		if err != nil {
			return "", err
		}
	case pathBitmex:
		stringsss, err = HTMLScrapeBitmex(htmlData)
		if err != nil {
			return "", err
		}
	case pathANX:
		stringsss, err = HTMLScrapeANX(htmlData)
		if err != nil {
			return "", err
		}
		return stringsss[0], nil
	case pathPoloniex:
		stringsss, err = HTMLScrapePoloniex(htmlData)
		if err != nil {
			return "", err
		}
	case pathIbBit:
		stringsss, err = HTMLScrapeItBit(htmlData)
		if err != nil {
			return "", err
		}
		return stringsss[0], nil
	case pathBTCMarkets:
		stringsss, err = HTMLScrapeBTCMarkets(htmlData)
		if err != nil {
			return "", err
		}
		return stringsss[0], nil
	case pathEXMO:
		stringsss, err = HTMLScrapeExmo(htmlData)
		if err != nil {
			return "", err
		}
		return stringsss[0], nil
	case pathBitstamp:
		stringsss, err = HTMLScrapeBitstamp(htmlData)
		if err != nil {
			return "", err
		}
		return stringsss[0], nil
	case pathHitBTC:
		stringsss, err = HTMLScrapeHitBTC(htmlData)
		if err != nil {
			return "", err
		}
		return stringsss[0], nil
	case pathBitflyer:
		stringsss, err = HTMLScrapeBitflyer(htmlData)
		if err != nil {
			return "", err
		}
		return stringsss[0], nil
	case pathLakeBTC:
		stringsss, err = HTMLScrapeLakeBTC(htmlData)
		if err != nil {
			return "", err
		}
		return stringsss[0], nil
	case pathKraken:
		stringsss, err = HTMLScrapeKraken(htmlData)
		if err != nil {
			return "", err
		}
		return stringsss[0], nil
	default:
		stringsss, err = HTMLScrapeDefault(htmlData)
		if err != nil {
			return "", err
		}
	}
	switch htmlData.Path {
	case pathOkCoin, pathOkex:
		for x := range stringsss {
			if len(stringsss[x]) != 10 {
				tempStorage := strings.Split(stringsss[x], "-")
				stringsss[x] = fmt.Sprintf("%s-0%s-%s", tempStorage[0], tempStorage[1], tempStorage[2])
			}
		}
	}

	switch {
	case len(stringsss) == 1:
		return stringsss[0], nil
	case len(stringsss) > 1:
		x, err := time.Parse(htmlData.DateFormat, stringsss[0])
		if err != nil {
			return "", err
		}
		y, err := time.Parse(htmlData.DateFormat, stringsss[len(stringsss)-1])
		if err != nil {
			return "", err
		}
		z := y.Sub(x)
		switch {
		case z > 0:
			return stringsss[len(stringsss)-1], nil
		case z < 0:
			return stringsss[0], nil
		default:
			return "", errors.New("y and x store the same value, please manually check for Updates.json")
		}
	default:
	}
	return "", errors.New("no response found")
}

// Add checks if api Updates.json are needed
func Add(exchName, checkType, path string, data interface{}) error {
	finalResp, check, err := CheckExistingExchanges(file, exchName)
	if err != nil {
		return err
	}
	if check {
		if verbose {
			log.Println("Exchange Already Exists")
		}
		return nil
	}
	exchange, err := FillData(exchName, checkType, path, data)
	if err != nil {
		return err
	}
	finalResp = append(finalResp, exchange)
	file, err := json.MarshalIndent(finalResp, "", " ")
	if err != nil {
		return err
	}
	err = ioutil.WriteFile("Updates.json", file, 0644)
	if err != nil {
		return err
	}
	return nil
}

// FillData fills exchange data based on the given checkType
func FillData(exchName, checkType, path string, data interface{}) (ExchangeInfo, error) {
	var resp ExchangeInfo
	switch checkType {
	case github:
		var gitData GithubData
		var resp ExchangeInfo
		resp.Name = exchName
		resp.CheckType = checkType
		gitData.Repo = path
		tempSha, err := GetSha(path)
		if err != nil {
			return resp, err
		}
		gitData.Sha = tempSha.ShaResp
		resp.Data.GitHubData = &gitData
		return resp, nil
	case htmlScrape:
		tempData := data.(HTMLScrapingData)
		var htmlData HTMLScrapingData
		checkStr, err := CheckChangeLog(tempData)
		if err != nil {
			return resp, err
		}
		var resp ExchangeInfo
		resp.Name = exchName
		resp.CheckType = checkType
		htmlData.CheckString = checkStr
		htmlData.DateFormat = tempData.DateFormat
		htmlData.Key = tempData.Key
		htmlData.RegExp = tempData.RegExp
		htmlData.TextTokenData = tempData.TextTokenData
		htmlData.TokenData = tempData.TokenData
		htmlData.TokenDataEnd = tempData.TokenDataEnd
		htmlData.Val = tempData.Val
		htmlData.Path = tempData.Path
		resp.Data.HTMLData = &htmlData
		return resp, nil
	default:
		return resp, errors.New("invalid checkType")
	}
}

// HTMLScrapeDefault gets check string data for the default cases
func HTMLScrapeDefault(htmlData HTMLScrapingData) ([]string, error) {
	var resp []string
	temp, err := http.Get(htmlData.Path)
	if err != nil {
		return resp, err
	}
	meow := html.NewTokenizer(temp.Body)
loop:
	for {
		next := meow.Next()
		switch next {
		case html.ErrorToken:
			break loop
		case html.StartTagToken:
			token := meow.Token()
			isBool := token.Data == htmlData.TokenData
			if isBool {
				for _, a := range token.Attr {
					if a.Key == htmlData.Key && a.Val == htmlData.Val {
					loop2:
						for {
							nextToken := meow.Next()
							switch nextToken {
							case html.EndTagToken:
								t := meow.Token()
								if t.Data == htmlData.TokenDataEnd {
									break loop2
								}
							case html.StartTagToken:
								tokz := meow.Token()
								if tokz.Data == htmlData.TextTokenData {
									inner := meow.Next()
									if inner == html.TextToken {
										tempStr := (string)(meow.Text())
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
		default:
			continue
		}
	}
	return resp, nil
}

// HTMLScrapeBTSE gets the check string for BTSE exchange
func HTMLScrapeBTSE(htmlData HTMLScrapingData) ([]string, error) {
	var resp []string
	temp, err := http.Get(htmlData.Path)
	if err != nil {
		return resp, err
	}
	meow := html.NewTokenizer(temp.Body)
loop:
	for {
		next := meow.Next()
		switch next {
		case html.ErrorToken:
			break loop
		case html.StartTagToken:
			token := meow.Token()
			isBool := token.Data == htmlData.TokenData
			if isBool {
				for _, z := range token.Attr {
					if z.Key == htmlData.Key && z.Val == htmlData.Val {
						inner := meow.Next()
						if inner == html.TextToken {
							resp = append(resp, (string)(meow.Text()))
						}
					}
				}
			}
		}
	}
	return resp, nil
}

// HTMLScrapeBitfinex gets the check string for Bitfinex exchange
func HTMLScrapeBitfinex(htmlData HTMLScrapingData) ([]string, error) {
	var resp []string
	temp, err := http.Get(htmlData.Path)
	if err != nil {
		return resp, err
	}
	meow := html.NewTokenizer(temp.Body)
loop:
	for {
		next := meow.Next()
		switch next {
		case html.ErrorToken:
			break loop
		case html.StartTagToken:
			token := meow.Token()
			isBool := token.Data == htmlData.TokenData
			if isBool {
				for _, a := range token.Attr {
					if a.Key == htmlData.Key && a.Val == htmlData.Val {
					loop2:
						for {
							nextToken := meow.Next()
							switch nextToken {
							case html.StartTagToken:
								nextToken := meow.Token()
								for _, z := range nextToken.Attr {
									if z.Key == "id" {
										r, err := regexp.Compile(htmlData.RegExp)
										if err != nil {
											return resp, err
										}
										result := r.MatchString(z.Val)
										if result {
											tempStr := strings.Replace(z.Val, "section-v-", "", 1)
											resp = append(resp, tempStr)
										}
									}
								}
							case html.EndTagToken:
								tok := meow.Token()
								if tok.Data == htmlData.TokenDataEnd {
									break loop2
								}
							}
						}
					}
				}
			}
		default:
			continue
		}
	}
	return resp, nil
}

// HTMLScrapeBitmex gets the check string for Bitmex exchange
func HTMLScrapeBitmex(htmlData HTMLScrapingData) ([]string, error) {
	var resp []string
	temp, err := http.Get(htmlData.Path)
	if err != nil {
		return resp, err
	}
	meow := html.NewTokenizer(temp.Body)
loop:
	for {
		next := meow.Next()
		switch next {
		case html.ErrorToken:
			break loop
		case html.StartTagToken:
			tokz := meow.Token()
			if tokz.Data == htmlData.TokenData {
				for _, x := range tokz.Attr {
					if x.Key == htmlData.Key {
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
		default:
			continue
		}
	}
	return resp, nil
}

// HTMLScrapeHitBTC gets the check string for HitBTC Exchange
func HTMLScrapeHitBTC(htmlData HTMLScrapingData) ([]string, error) {
	var resp []string
	temp, err := http.Get(htmlData.Path)
	if err != nil {
		return resp, err
	}
	meow := html.NewTokenizer(temp.Body)
loop:
	for {
		next := meow.Next()
		switch next {
		case html.ErrorToken:
			break loop
		case html.StartTagToken:
			token := meow.Token()
			isBool := token.Data == htmlData.TokenData
			if isBool {
				for _, z := range token.Attr {
					if z.Key == htmlData.Key && z.Val == htmlData.Val {
					loop2:
						for {
							nextToken := meow.Next()
							switch nextToken {
							case html.EndTagToken:
								t := meow.Token()
								if t.Data == htmlData.TokenDataEnd {
									break loop2
								}
							case html.StartTagToken:
								t := meow.Token()
								if t.Data == htmlData.TextTokenData {
									inner := meow.Next()
									if inner == html.TextToken {
										tempStr := ((string)(meow.Text()))
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

// HTMLScrapeBTCMarkets gets the check string for BTCMarkets exchange
func HTMLScrapeBTCMarkets(htmlData HTMLScrapingData) ([]string, error) {
	var resp []string
	temp, err := http.Get(htmlData.Path)
	if err != nil {
		return resp, err
	}
	tempData, err := ioutil.ReadAll(temp.Body)
	if err != nil {
		return resp, err
	}
	tempStr := ((string)(tempData))
	r, err := regexp.Compile(htmlData.RegExp)
	if err != nil {
		return resp, err
	}
	result := r.FindString(tempStr)
	resp = append(resp, result)
	return resp, nil
}

// HTMLScrapeBitflyer gets the check string for BTCMarkets exchange
func HTMLScrapeBitflyer(htmlData HTMLScrapingData) ([]string, error) {
	var resp []string
	var tempArray []string
	temp, err := http.Get(htmlData.Path)
	if err != nil {
		return resp, err
	}
	meow := html.NewTokenizer(temp.Body)
loop:
	for {
		next := meow.Next()
		switch next {
		case html.ErrorToken:
			break loop
		case html.StartTagToken:
			tokz := meow.Token()
			if tokz.Data == htmlData.TokenData {
				for {
					nextToken := meow.Next()
					switch nextToken {
					case html.EndTagToken:
						t := meow.Token()
						if t.Data == htmlData.TokenDataEnd {
							break loop
						}
					case html.StartTagToken:
						t := meow.Token()
						if t.Data == htmlData.TextTokenData {
							inner := meow.Next()
							if inner == html.TextToken {
								tempStr := ((string)(meow.Text()))
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
					default:
						continue
					}
				}
			}
		default:
			continue
		}
	}
	resp = append(resp, tempArray[1])
	return resp, nil
}

// HTMLScrapeANX gets the check string for BTCMarkets exchange
func HTMLScrapeANX(htmlData HTMLScrapingData) ([]string, error) {
	var resp []string
	temp, err := http.Get(htmlData.Path)
	if err != nil {
		return resp, err
	}
	meow := html.NewTokenizer(temp.Body)
loop:
	for {
		next := meow.Next()
		switch next {
		case html.ErrorToken:
			break loop
		case html.TextToken:
			tempStr := ((string)(meow.Text()))
			r, err := regexp.Compile(htmlData.RegExp)
			if err != nil {
				return resp, err
			}
			result := r.MatchString(tempStr)
			if result {
				resp = append(resp, r.FindString(tempStr))
				break loop
			}
		default:
			continue
		}
	}
	return resp, nil
}

// HTMLScrapeExmo gets the check string for Exmo Exchange
func HTMLScrapeExmo(htmlData HTMLScrapingData) ([]string, error) {
	var resp []string
	temp, err := http.NewRequest(http.MethodGet, htmlData.Path, nil)
	if err != nil {
		return resp, err
	}
	temp.Header.Set("User-Agent", "GCT")
	httpClient := &http.Client{}
	httpResp, err := httpClient.Do(temp)
	if err != nil {
		return resp, err
	}
	meow := html.NewTokenizer(httpResp.Body)

loop:
	for {
		next := meow.Next()
		switch next {
		case html.ErrorToken:
			break loop
		case html.StartTagToken:
			token := meow.Token()
			isBool := token.Data == htmlData.TokenData
			if isBool {
				for _, z := range token.Attr {
					if z.Key == htmlData.Key && z.Val == htmlData.Val {
					loop2:
						for {
							nextToken := meow.Next()
							switch nextToken {
							case html.EndTagToken:
								t := meow.Token()
								if t.Data == htmlData.TokenDataEnd {
									break loop2
								}
							case html.StartTagToken:
								t := meow.Token()
								if t.Data == htmlData.TextTokenData {
									nextToken := meow.Next()
									if nextToken == html.TextToken {
										resp = append(resp, ((string)(meow.Text())))
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

// HTMLScrapePoloniex gets the check string for Poloniex Exchange
func HTMLScrapePoloniex(htmlData HTMLScrapingData) ([]string, error) {
	var resp []string
	temp, err := http.Get(htmlData.Path)
	if err != nil {
		return resp, err
	}
	meow := html.NewTokenizer(temp.Body)
loop:
	for {
		next := meow.Next()
		switch next {
		case html.ErrorToken:
			break loop
		case html.StartTagToken:
			token := meow.Token()
			isBool := token.Data == htmlData.TokenData
			if isBool {
				for _, z := range token.Attr {
					if z.Key == htmlData.Key && z.Val == htmlData.Val {
					loop2:
						for {
							nextToken := meow.Next()
							switch nextToken {
							case html.EndTagToken:
								t := meow.Token()
								if t.Data == htmlData.TokenDataEnd {
									break loop2
								}
							case html.StartTagToken:
								t := meow.Token()
								if t.Data == htmlData.TextTokenData {
									newToken := meow.Next()
									if newToken == html.TextToken {
										tempStr := ((string)(meow.Text()))
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
func HTMLScrapeItBit(htmlData HTMLScrapingData) ([]string, error) {
	var resp []string
	temp, err := http.Get(htmlData.Path)
	if err != nil {
		return resp, err
	}
	meow := html.NewTokenizer(temp.Body)
loop:
	for {
		next := meow.Next()
		switch next {
		case html.ErrorToken:
			break loop
		case html.StartTagToken:
			token := meow.Token()
			isBool := token.Data == htmlData.TokenData
			if isBool {
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
func HTMLScrapeLakeBTC(htmlData HTMLScrapingData) ([]string, error) {
	var resp []string
	temp, err := http.Get(htmlData.Path)
	if err != nil {
		return resp, err
	}
	meow := html.NewTokenizer(temp.Body)
loop:
	for {
		next := meow.Next()
		switch next {
		case html.ErrorToken:
			break loop
		case html.StartTagToken:
			token := meow.Token()
			isBool := token.Data == htmlData.TokenData
			if isBool {
				for _, z := range token.Attr {
					if z.Key == htmlData.Key && z.Val == htmlData.Val {
					loop2:
						for {
							nextToken := meow.Next()
							switch nextToken {
							case html.EndTagToken:
								t := meow.Token()
								if t.Data == htmlData.TokenDataEnd {
									break loop2
								}
							case html.StartTagToken:
								t := meow.Token()
								if t.Data == htmlData.TextTokenData {
									inner := meow.Next()
									if inner == html.TextToken {
										tempStr := ((string)(meow.Text()))
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
		default:
			continue
		}
	}
	return resp, nil
}

// HTMLScrapeBitstamp gets the check string for Bitstamp Exchange
func HTMLScrapeBitstamp(htmlData HTMLScrapingData) ([]string, error) {
	var resp []string
	temp, err := http.Get(htmlData.Path)
	if err != nil {
		return resp, err
	}
	meow := html.NewTokenizer(temp.Body)
loop:
	for {
		next := meow.Next()
		switch next {
		case html.ErrorToken:
			break loop
		case html.StartTagToken:
			token := meow.Token()
			isBool := token.Data == htmlData.TokenData
			if isBool {
				for _, z := range token.Attr {
					if z.Key == htmlData.Key && z.Val == htmlData.Val {
					loop2:
						for {
							nextToken := meow.Next()
							switch nextToken {
							case html.EndTagToken:
								t := meow.Token()
								if t.Data == htmlData.TokenDataEnd {
									break loop2
								}
							case html.TextToken:
								tempStr := ((string)(meow.Text()))
								r, err := regexp.Compile(htmlData.RegExp)
								if err != nil {
									return resp, err
								}
								respStr := r.FindString(tempStr)
								if respStr != "" {
									resp = append(resp, respStr)
									break loop2
								}
							}
						}
					}
				}
			}
		default:
			continue
		}
	}
	return resp, nil
}

// HTMLScrapeKraken gets the check string for Kraken Exchange
func HTMLScrapeKraken(htmlData HTMLScrapingData) ([]string, error) {
	var resp []string
	temp, err := http.Get(htmlData.Path)
	if err != nil {
		return resp, err
	}
	meow := html.NewTokenizer(temp.Body)
loop:
	for {
		next := meow.Next()
		switch next {
		case html.ErrorToken:
			break loop
		case html.StartTagToken:
			token := meow.Token()
			isBool := token.Data == htmlData.TokenData
			if isBool {
				inner := meow.Next()
				if inner == html.TextToken {
					if ((string)(meow.Text())) == "Get account balance" {
					loop2:
						for {
							next := meow.Next()
							switch next {
							case html.EndTagToken:
								t := meow.Token()
								if t.Data == htmlData.TokenDataEnd {
									break loop2
								}
							case html.StartTagToken:
								t := meow.Token()
								if t.Data == htmlData.TextTokenData {
									inside := meow.Next()
									if inside == html.TextToken {
										tempStr := ((string)(meow.Text()))
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
		default:
			continue
		}
	}
	return resp, nil
}

// GetListsData gets required data for all the lists on the given board
func GetListsData(idBoard string) ([]ListData, error) {
	path := fmt.Sprintf(pathGetAllLists, idBoard, apiKey, apiToken)
	var resp []ListData
	err := SendHTTPRequest(path, &resp)
	if err != nil {
		return resp, err
	}
	log.Println(resp)
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
	log.Println(params.Encode())
	path := fmt.Sprintf(pathNewCard, params.Encode(), apiKey, apiToken)
	_, err := common.SendHTTPRequest(http.MethodPost, path, nil, nil)
	return err
}

// CreateNewCheck creates a new checklist item within a given checklist
func CreateNewCheck(newCheck string) error {
	params := url.Values{}
	params.Set("name", newCheck)
	path := fmt.Sprintf(pathChecklists, updateChecklistID, params.Encode(), apiKey, apiToken)
	_, err := common.SendHTTPRequest(http.MethodPost, path, nil, nil)
	return err
}

// SendHTTPRequest sends an unauthenticated HTTP request
func SendHTTPRequest(path string, result interface{}) error {
	return common.SendHTTPGetRequest(path, true, false, result)
}
