package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"os"

	"github.com/thrasher-/gocryptotrader/common"
)

const (
	commonPath              = "..%s..%scommon%s"
	configPath              = "..%s..%sconfig%s"
	currencyPath            = "..%s..%scurrency%s"
	currencyPairPath        = "..%s..%scurrency%spair%s"
	currencySymbolPath      = "..%s..%scurrency%ssymbol%s"
	currencyTranslationPath = "..%s..%scurrency%stranslation%s"
	eventsPath              = "..%s..%sevents%s"
	exchangesPath           = "..%s..%sexchanges%s"
	exchangesNoncePath      = "..%s..%sexchanges%snonce%s"
	exchangesOrderbookPath  = "..%s..%sexchanges%sorderbook%s"
	exchangesStatsPath      = "..%s..%sexchanges%sstats%s"
	exchangesTickerPath     = "..%s..%sexchanges%sticker%s"
	portfolioPath           = "..%s..%sportfolio%s"
	smsglobalPath           = "..%s..%ssmsglobal%s"
	testdataPath            = "..%s..%stestdata%s"
	toolsPath               = "..%s..%stools%s"
	webPath                 = "..%s..%sweb%s"
	rootPath                = "..%s..%s"
)

var (
	verbose, replace     bool
	codebasePaths        map[string]string
	codebaseTemplatePath map[string]string
	codebaseReadme       map[string]readme
	tmpl                 *template.Template
	path                 string
)

type readme struct {
	Name         string
	Contributors string
}

func main() {

	flag.BoolVar(&verbose, "v", false, "-v Verbose flag prints more information to the std output")
	flag.BoolVar(&replace, "r", false, "-r Replace flag generates and replaces all documentation across the code base")

	flag.Parse()

	fmt.Println(`
              GoCryptoTrader: Exchange documentation tool

    This will update and regenerate documentation for the different packages
    in GoCryptoTrader.
`)

	codebasePaths = make(map[string]string)
	codebaseTemplatePath = make(map[string]string)
	codebaseReadme = make(map[string]readme)
	path = common.GetOSPathSlash()
	if err := addTemplates(); err != nil {
		log.Fatal("GoCryptoTrader: Exchange documentation tool add template error ", err)
	}

	if err := updateReadme(); err != nil {
		log.Fatal("GoCryptoTrader: Exchange documentation tool update readme error ", err)
	}

	fmt.Println("\nTool finished")
}

// Iterates through codebase paths to check for readme files and either adds
// or replaces with new readme files.
func updateReadme() error {
	addPaths()

	for packageName := range codebasePaths {
		addReadmeData(packageName)

		if !checkReadme(packageName) {
			if verbose {
				fmt.Printf("* %s Readme file FOUND.\n", packageName)
			}
			if replace {
				fmt.Println("file replacement")
				if err := replaceReadme(packageName); err != nil {
					return err
				}
				continue
			}
			continue
		}
		if verbose {
			fmt.Printf("* %s Readme file NOT FOUND.\n", packageName)
		}
		if replace {
			log.Println("file creation")
			if err := createReadme(packageName); err != nil {
				return err
			}
			continue
		}
	}
	return nil
}

// Adds paths to different potential README.md files in the codebase
func addPaths() {
	codebasePaths["common"] = fmt.Sprintf(commonPath, path, path, path)
	codebasePaths["config"] = fmt.Sprintf(configPath, path, path, path)
	codebasePaths["currency"] = fmt.Sprintf(currencyPath, path, path, path)
	codebasePaths["currency pair"] = fmt.Sprintf(currencyPairPath, path, path, path, path)
	codebasePaths["currency symbol"] = fmt.Sprintf(currencySymbolPath, path, path, path, path)
	codebasePaths["currency translation"] = fmt.Sprintf(currencyTranslationPath, path, path, path, path)
	codebasePaths["events"] = fmt.Sprintf(eventsPath, path, path, path)
	codebasePaths["exchanges"] = fmt.Sprintf(exchangesPath, path, path, path)
	codebasePaths["exchanges nonce"] = fmt.Sprintf(exchangesNoncePath, path, path, path, path)
	codebasePaths["exchanges orderbook"] = fmt.Sprintf(exchangesOrderbookPath, path, path, path, path)
	codebasePaths["exchanges stats"] = fmt.Sprintf(exchangesStatsPath, path, path, path, path)
	codebasePaths["exchanges ticker"] = fmt.Sprintf(exchangesTickerPath, path, path, path, path)
	codebasePaths["portfolio"] = fmt.Sprintf(portfolioPath, path, path, path)
	codebasePaths["smsglobal"] = fmt.Sprintf(smsglobalPath, path, path, path)
	codebasePaths["testdata"] = fmt.Sprintf(testdataPath, path, path, path)
	codebasePaths["tools"] = fmt.Sprintf(toolsPath, path, path, path)
	codebasePaths["web"] = fmt.Sprintf(webPath, path, path, path)
	codebasePaths["root"] = fmt.Sprintf(rootPath, path, path)
}

func addReadmeData(packageName string) {
	readmeInfo := readme{
		Name:         packageName,
		Contributors: "", //future implementation to track contributors
	}
	codebaseReadme[packageName] = readmeInfo
}

// adds all the template files
func addTemplates() error {
	glob, err := template.ParseGlob(fmt.Sprintf("readme_templates%s*", path))
	if err != nil {
		return err
	}
	_, err = glob.ParseGlob(fmt.Sprintf("sub_templates%s*", path))
	if err != nil {
		return err
	}

	tmpl = glob
	return nil
}

// checkReadme checks to see if the file exists
func checkReadme(packageName string) bool {
	_, err := os.Stat(codebasePaths[packageName] + "README.md")
	return os.IsNotExist(err)
}

// replaces readme file
func replaceReadme(packageName string) error {
	if err := deleteFile(codebasePaths[packageName] + "README.md"); err != nil {
		return err
	}
	return createReadme(packageName)
}

// creates new readme file and executes template
func createReadme(packageName string) error {
	file, err := os.Create(codebasePaths[packageName] + "README.md")
	defer file.Close()
	if err != nil {
		return err
	}
	fmt.Println("File done")
	return tmpl.ExecuteTemplate(file, packageName, codebaseReadme[packageName])
}

func deleteFile(path string) error {
	return os.Remove(path)
}
