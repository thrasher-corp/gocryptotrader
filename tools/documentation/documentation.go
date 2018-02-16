package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"os"
	"os/exec"
	"strings"

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
)

type readme struct {
	Name         string
	Godoc        godoc
	Contributors string
}

type godoc struct {
	Constants string
	Functions string
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
	s := common.GetOSPathSlash()
	codebasePaths["common"] = fmt.Sprintf(commonPath, s, s, s)
	codebasePaths["config"] = fmt.Sprintf(configPath, s, s, s)
	codebasePaths["currency"] = fmt.Sprintf(currencyPath, s, s, s)
	codebasePaths["currency pair"] = fmt.Sprintf(currencyPairPath, s, s, s, s)
	codebasePaths["currency symbol"] = fmt.Sprintf(currencySymbolPath, s, s, s, s)
	codebasePaths["currency translation"] = fmt.Sprintf(currencyTranslationPath, s, s, s, s)
	codebasePaths["events"] = fmt.Sprintf(eventsPath, s, s, s)
	codebasePaths["exchanges"] = fmt.Sprintf(exchangesPath, s, s, s)
	codebasePaths["exchanges nonce"] = fmt.Sprintf(exchangesNoncePath, s, s, s, s)
	codebasePaths["exchanges orderbook"] = fmt.Sprintf(exchangesOrderbookPath, s, s, s, s)
	codebasePaths["exchanges stats"] = fmt.Sprintf(exchangesStatsPath, s, s, s, s)
	codebasePaths["exchanges ticker"] = fmt.Sprintf(exchangesTickerPath, s, s, s, s)
	codebasePaths["portfolio"] = fmt.Sprintf(portfolioPath, s, s, s)
	codebasePaths["smsglobal"] = fmt.Sprintf(smsglobalPath, s, s, s)
	codebasePaths["testdata"] = fmt.Sprintf(testdataPath, s, s, s)
	codebasePaths["tools"] = fmt.Sprintf(toolsPath, s, s, s)
	codebasePaths["web"] = fmt.Sprintf(webPath, s, s, s)
	codebasePaths["root"] = fmt.Sprintf(rootPath, s, s)
}

func addReadmeData(packageName string) {
	readmeInfo := readme{
		Name:         packageName,
		Godoc:        findGodoc(packageName),
		Contributors: "",
	}
	codebaseReadme[packageName] = readmeInfo
}

func findGodoc(packageName string) godoc {
	var doc godoc

	godoc := exec.Command("godoc", codebasePaths[packageName])
	op, err := godoc.Output()
	if err != nil {
		log.Println(err)
	}

	sop := string(op[:])
	if strings.Index(sop, "CONSTANTS") == -1 {
		return doc
	}

	sopTrim := sop[strings.Index(sop, "CONSTANTS"):]
	constantsList := sopTrim[strings.Index(sopTrim, "const"):strings.Index(sopTrim, ")")]
	doc.Constants = constantsList
	functions := sopTrim[strings.Index(sopTrim, "func"):]
	doc.Functions = functions
	return doc
}

// adds all the template files
func addTemplates() error {
	glob, err := template.ParseGlob("*.tmpl")
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
	log.Println("File done")
	return tmpl.ExecuteTemplate(file, packageName, codebaseReadme[packageName])
}

func deleteFile(path string) error {
	return os.Remove(path)
}
