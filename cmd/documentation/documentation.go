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

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/core"
)

const (
	// DefaultRepo is the main example repository
	DefaultRepo = "https://api.github.com/repos/thrasher-corp/gocryptotrader"

	// GithubAPIEndpoint allows the program to query your repository
	// contributor list
	GithubAPIEndpoint = "/contributors"

	// LicenseFile defines a license file
	LicenseFile = "LICENSE"

	// ContributorFile defines contributor file
	ContributorFile = "CONTRIBUTORS"
)

// DefaultExcludedDirectories defines the basic directory exclusion list for GCT
var DefaultExcludedDirectories = []string{".github",
	".git",
	"node_modules",
	".vscode"}

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

	fmt.Println(core.Banner)
	fmt.Println("This will update and regenerate documentation for the different packages in your repo.")
	fmt.Println()

	if *verbose {
		fmt.Println("Fetching configuration...")
	}

	config, err := GetConfiguration()
	if err != nil {
		log.Fatalf("Documentation Generation Tool - GetConfiguration error %s",
			err)
	}

	if *verbose {
		fmt.Println("Fetching project directory tree...")
	}

	dirList, err := GetProjectDirectoryTree(&config, *verbose)
	if err != nil {
		log.Fatalf("Documentation Generation Tool - GetProjectDirectoryTree error %s",
			err)
	}

	var contributors []Contributor
	if config.ContributorFile {
		if *verbose {
			fmt.Println("Fetching repository contributor list...")
		}
		contributors, err = GetContributorList(config.GithubRepo)
		if err != nil {
			log.Fatalf("Documentation Generation Tool - GetContributorList error %s",
				err)
		}

		// idoall's contributors were forked and merged, so his contributions
		// aren't automatically retrievable
		contributors = append(contributors, Contributor{
			Login:         "idoall",
			URL:           "https://github.com/idoall",
			Contributions: 1,
		})

		// Github API missing contributors
		missingAPIContributors := []Contributor{
			{
				Login:         "mattkanwisher",
				URL:           "https://github.com/mattkanwisher",
				Contributions: 1,
			},
			{
				Login:         "mKurrels",
				URL:           "https://github.com/mKurrels",
				Contributions: 1,
			},
			{
				Login:         "m1kola",
				URL:           "https://github.com/m1kola",
				Contributions: 1,
			},
			{
				Login:         "cavapoo2",
				URL:           "https://github.com/cavapoo2",
				Contributions: 1,
			},
			{
				Login:         "zeldrinn",
				URL:           "https://github.com/zeldrinn",
				Contributions: 1,
			},
		}
		contributors = append(contributors, missingAPIContributors...)

		if *verbose {
			fmt.Println("Contributor List Fetched")
			for i := range contributors {
				fmt.Println(contributors[i].Login)
			}
		}
	} else {
		fmt.Println("Contributor list file disabled skipping fetching details")
	}

	if *verbose {
		fmt.Println("Fetching template files...")
	}

	tmpl, err := GetTemplateFiles()
	if err != nil {
		log.Fatalf("Documentation Generation Tool - GetTemplateFiles error %s",
			err)
	}

	if *verbose {
		fmt.Println("All core systems fetched, updating documentation...")
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
		fmt.Println("Creating configuration file, please check to add a different github repository path and change preferences")

		file, err = os.Create("config.json")
		if err != nil {
			return c, err
		}

		// Set default params for configuration
		c.GithubRepo = DefaultRepo
		c.ContributorFile = true
		c.LicenseFile = true
		c.RootReadme = true
		c.ReferencePathToRepo = "../../"
		c.Exclusions.Directories = DefaultExcludedDirectories

		data, mErr := json.MarshalIndent(c, "", " ")
		if mErr != nil {
			return c, mErr
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
		return c, errors.New("repository not set in config.json file, please change")
	}

	if c.ReferencePathToRepo == "" {
		return c, errors.New("reference path not set in the config.json file, please set")
	}

	return c, nil
}

// IsExcluded returns if the file path is included in the exclusion list
func IsExcluded(path string, exclusion []string) bool {
	for i := range exclusion {
		if path == exclusion[i] {
			return true
		}
	}
	return false
}

// GetProjectDirectoryTree uses filepath walk functions to get each individual
// directory name and path to match templates with
func GetProjectDirectoryTree(c *Config, verbose bool) ([]string, error) {
	var directoryData []string
	if c.RootReadme { // Projects root README.md
		directoryData = append(directoryData, c.ReferencePathToRepo)
	}

	if c.LicenseFile { // Standard license file
		directoryData = append(directoryData, c.ReferencePathToRepo+LicenseFile)
	}

	if c.ContributorFile { // Standard contributor file
		directoryData = append(directoryData, c.ReferencePathToRepo+ContributorFile)
	}

	walkfn := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// Bypass what is contained in config.json directory exclusion
			if IsExcluded(info.Name(), c.Exclusions.Directories) {
				if verbose {
					fmt.Println("Excluding Directory:", info.Name())
				}
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
func GetContributorList(repo string) ([]Contributor, error) {
	var resp []Contributor
	return resp, common.SendHTTPGetRequest(repo+GithubAPIEndpoint, true, false, &resp)
}

// GetDocumentationAttributes returns specific attributes for a file template
func GetDocumentationAttributes(packageName string, contributors []Contributor) Attributes {
	return Attributes{
		Name:         GetPackageName(packageName, false),
		Contributors: contributors,
		NameURL:      GetGoDocURL(packageName),
		Year:         time.Now().Year(),
		CapitalName:  GetPackageName(packageName, true),
	}
}

// GetPackageName returns the package name after cleaning path as a string
func GetPackageName(name string, capital bool) string {
	newStrings := strings.Split(name, " ")
	var i int
	if len(newStrings) > 1 {
		i = 1
	}
	if capital {
		return strings.Title(newStrings[i])
	}
	return newStrings[i]
}

// GetGoDocURL returns a string for godoc package names
func GetGoDocURL(name string) string {
	if strings.Contains(name, " ") {
		return strings.Join(strings.Split(name, " "), "/")
	}
	if name == "testdata" ||
		name == "tools" ||
		name == ContributorFile ||
		name == LicenseFile {
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
			if d == "" {
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
			if details.Verbose {
				fmt.Println("Excluding file:", name)
			}
			continue
		}

		if details.Tmpl.Lookup(name) == nil {
			fmt.Printf("Template not found for path %s create new template with {{define \"%s\" -}} TEMPLATE HERE {{end}}\n",
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
		if err != nil && !strings.Contains(err.Error(), "no such file or directory") {
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
