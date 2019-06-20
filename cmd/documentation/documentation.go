package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
)

const (
	// DefaultRepo is the main example repository
	DefaultRepo = "https://api.github.com/repos/thrasher-/gocryptotrader"
	// GithubAPIEndpoint allows the program to query your repository
	// contributor list
	GithubAPIEndpoint = "/contributors"

	// LicenseFile defines a license file
	LicenseFile = "LICENSE"
	// ContributorFile defines contributor file
	ContributorFile = "CONTRIBUTORS"

	welcome = `
	___________.__                        .__                           _________                      
	\__    ___/|  |______________    _____|  |__   ___________          \_   ___ \  _________________  
	  |    |   |  |  \_  __ \__  \  /  ___/  |  \_/ __ \_  __ \  ______ /    \  \/ /  _ \_  __ \____ \ 
	  |    |   |   Y  \  | \// __ \_\___ \|   Y  \  ___/|  | \/ /_____/ \     \___(  <_> )  | \/  |_> >
	  |____|   |___|  /__|  (____  /____  >___|  /\___  >__|             \______  /\____/|__|  |   __/ 
					\/           \/     \/     \/     \/                        \/             |__|    

	This will update and regenerate documentation for the different packages in your repo.`
)

// Contributor defines an account associated with this code base by doing
// contributions
type Contributor struct {
	Login         string `json:"login"`
	URL           string `json:"html_url"`
	Contributions int    `json:"contributions"`
}

// Config defines the running config to deploy documentation across a github
// repository including exclusion lists for files and directories
type Config struct {
	GithubRepo          string     `json:"githubRepo"`
	Exclusions          Exclusions `json:"exclusionList"`
	RootReadme          bool       `json:"rootReadmeActive"`
	LicenseFile         bool       `json:"licenseFileActive"`
	ContributorFile     bool       `json:"contributorFileActive"`
	ReferencePathToRepo string     `json:"referencePathToRepo"`
}

// Exclusions defines the exclusion list so documents are not generated
type Exclusions struct {
	Files       []string `json:"Files"`
	Directories []string `json:"Directories"`
}

// DocumentationDetails defines parameters to update documentation
type DocumentationDetails struct {
	Directories  []string
	Tmpl         *template.Template
	Contributors []Contributor
	Verbose      bool
	Config       *Config
}

// Attributes defines specific documentation attributes when a template is
// executed
type Attributes struct {
	Name         string
	Contributors []Contributor
	NameURL      string
	Year         int
	CapitalName  string
}

func main() {
	verbose := flag.Bool("v", false, "Verbose output")

	flag.Parse()

	fmt.Println(welcome)
	fmt.Println()

	config, err := GetConfiguration()
	if err != nil {
		log.Fatalf("Documentation Generation Tool - GetConfiguration error %s",
			err)
	}

	dirList, err := GetProjectDirectoryTree(&config)
	if err != nil {
		log.Fatalf("Documentation Generation Tool - GetProjectDirectoryTree error %s",
			err)
	}

	var contributors []Contributor
	if config.ContributorFile {
		contributors, err = GetContributorList(config.GithubRepo, *verbose)
		if err != nil {
			log.Fatalf("Documentation Generation Tool - GetContributorList error %s",
				err)
		}

		if *verbose {
			fmt.Println("Contributor List Fetched")
			for i := range contributors {
				fmt.Println(contributors[i].Login)
			}
		}
	} else {
		fmt.Println("Contributor list file disabled skipping fetching details")
	}

	tmpl, err := GetTemplateFiles()
	if err != nil {
		log.Fatalf("Documentation Generation Tool - GetTemplateFiles error %s",
			err)
	}

	if *verbose {
		fmt.Println("Templates Fetched")
	}

	err = UpdateDocumentation(DocumentationDetails{
		dirList,
		tmpl,
		contributors,
		*verbose,
		&config})
	if err != nil {
		log.Fatalf("Documentation Generation Tool - UpdateDocumentation error %s",
			err)
	}

	fmt.Println("\nDocumentation Generation Tool - Finished")
}

// GetConfiguration retrieves the documentation configuration
func GetConfiguration() (Config, error) {
	var c Config
	file, err := os.OpenFile("config.json", os.O_RDWR, os.ModePerm)
	if err != nil {
		fmt.Println("Creating configuration file, please add github repository path and preferences")

		file, err = os.Create("config.json")
		if err != nil {
			return c, err
		}

		data, err := json.MarshalIndent(c, "", " ")
		if err != nil {
			return c, err
		}

		_, err = file.WriteAt(data, 0)
		if err != nil {
			return c, err
		}
	}

	defer file.Close()

	config, err := ioutil.ReadAll(file)
	if err != nil {
		return c, err
	}

	err = json.Unmarshal(config, &c)
	if err != nil {
		return c, err
	}

	if c.GithubRepo == "" {
		return c, errors.New("repository not set")
	}

	if c.ReferencePathToRepo == "" {
		return c, errors.New("reference path not set")
	}

	return c, nil
}

// IsExcluded returns if the file path is included in the exclusion list
func IsExcluded(path string, exclusion []string) bool {
	for _, data := range exclusion {
		if strings.Contains(path, data) {
			return true
		}
	}
	return false
}

// GetProjectDirectoryTree uses filepath walk functions to get each individual
// directory name and path to match templates with
func GetProjectDirectoryTree(c *Config) ([]string, error) {
	var directoryData []string
	if c.RootReadme {
		directoryData = append(directoryData, c.ReferencePathToRepo) // Root Readme
	}

	if c.LicenseFile {
		directoryData = append(directoryData, c.ReferencePathToRepo+LicenseFile)
	}

	if c.ContributorFile {
		directoryData = append(directoryData, c.ReferencePathToRepo+ContributorFile)
	}

	walkfn := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// Bypass .git and web (front end) directories
			if IsExcluded(info.Name(), c.Exclusions.Directories) {
				fmt.Println("Excluding Directory:", info.Name())
				return filepath.SkipDir
			}
			// Don't append parent directory
			if strings.EqualFold(info.Name(), "..") {
				return nil
			}
			directoryData = append(directoryData, path)
		}
		return nil
	}

	return directoryData, filepath.Walk(c.ReferencePathToRepo, walkfn)
}

// GetTemplateFiles parses and returns all template files in the documentation
// tree
func GetTemplateFiles() (*template.Template, error) {
	tmpl := template.New("")

	walkfn := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if path == "." || path == ".." {
				return nil
			}

			var parseError error
			tmpl, parseError = tmpl.ParseGlob(filepath.Join(path, "*.tmpl"))
			if parseError != nil {
				return parseError
			}
			return filepath.SkipDir
		}
		return nil
	}

	return tmpl, filepath.Walk(".", walkfn)
}

// GetContributorList fetches a list of contributors from the github api
// endpoint
func GetContributorList(repo string, verbose bool) ([]Contributor, error) {
	var resp []Contributor
	return resp, common.SendHTTPGetRequest(repo+GithubAPIEndpoint, true, verbose, &resp)
}

// GetDocumentationAttributes returns specific attributes for a file template
func GetDocumentationAttributes(packageName string, contributors []Contributor) Attributes {
	return Attributes{
		Name:         getName(packageName, false),
		Contributors: contributors,
		NameURL:      getslashFromName(packageName),
		Year:         time.Now().Year(),
		CapitalName:  getName(packageName, true),
	}
}

func getName(name string, capital bool) string {
	newStrings := strings.Split(name, " ")
	if len(newStrings) > 1 {
		if capital {
			return getCapital(newStrings[1])
		}
		return newStrings[1]
	}
	if capital {
		return getCapital(name)
	}
	return name
}

func getCapital(name string) string {
	capLetter := strings.ToUpper(string(name[0]))
	last := name[1:]

	return capLetter + last
}

// getslashFromName returns a string for godoc package names
func getslashFromName(name string) string {
	if strings.Contains(name, " ") {
		s := strings.Split(name, " ")
		return strings.Join(s, "/")
	}
	if name == "testdata" || name == "tools" || name == ContributorFile || name == LicenseFile {
		return ""
	}
	return name
}

// UpdateDocumentation generates or updates readme/documentation files across
// the codebase
func UpdateDocumentation(details DocumentationDetails) error {
	for _, path := range details.Directories {
		data := strings.Split(path, "/")
		var temp []string
		for _, d := range data {
			if d == ".." {
				continue
			}
			if len(d) == 0 {
				break
			}

			temp = append(temp, d)
		}

		var name string
		if len(temp) == 0 {
			name = "root"
		} else {
			name = strings.Join(temp, " ")
		}

		if IsExcluded(name, details.Config.Exclusions.Files) {
			fmt.Println("Excluding file:", name)
			continue
		}

		if details.Tmpl.Lookup(name) == nil {
			fmt.Printf("Template not found for path %s create new template with {{define \"%s\" -}}\n",
				path,
				name)
			continue
		}

		var mainPath string
		if name == LicenseFile || name == ContributorFile {
			mainPath = path
		} else {
			mainPath = filepath.Join(path, "README.md")
		}

		err := os.Remove(mainPath)
		if err != nil {
			return err
		}

		file, err := os.Create(mainPath)
		if err != nil {
			return err
		}
		defer file.Close()

		attr := GetDocumentationAttributes(name, details.Contributors)

		err = details.Tmpl.ExecuteTemplate(file, name, attr)
		if err != nil {
			return err
		}
	}
	return nil
}
