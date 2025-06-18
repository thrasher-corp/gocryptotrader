package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/file"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
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

	defaultGithubAPIPerPageLimit = 100
)

var (
	// DefaultExcludedDirectories defines the basic directory exclusion list for GCT
	DefaultExcludedDirectories = []string{
		".github",
		".git",
		"node_modules",
		".vscode",
		".idea",
		"cmd_templates",
		"common_templates",
		"communications_templates",
		"config_templates",
		"currency_templates",
		"events_templates",
		"exchanges_templates",
		"portfolio_templates",
		"root_templates",
		"sub_templates",
		"testdata_templates",
		"tools_templates",
		"web_templates",
	}

	// global flag for verbosity
	verbose bool
	// current tool directory to specify working templates
	toolDir string
	// exposes root directory if outside of document tool directory
	repoDir string
	// is a broken down version of the documentation tool dir for cross platform
	// checking
	ref          = []string{"gocryptotrader", "cmd", "documentation"}
	engineFolder = "engine"
	githubToken  = os.Getenv("GITHUB_TOKEN") // Overridden by the ghtoken flag when set
)

// Contributor defines an account associated with this code base by doing
// contributions
type Contributor struct {
	Login         string `json:"login"`
	URL           string `json:"html_url"`
	Contributions int    `json:"contributions"`
}

// ghError defines a GitHub error response
type ghError struct {
	Message string `json:"message"`
	Status  string `json:"status"`
}

// Config defines the running config to deploy documentation across a github
// repository including exclusion lists for files and directories
type Config struct {
	GithubRepo      string     `json:"githubRepo"`
	Exclusions      Exclusions `json:"exclusionList"`
	RootReadme      bool       `json:"rootReadmeActive"`
	LicenseFile     bool       `json:"licenseFileActive"`
	ContributorFile bool       `json:"contributorFileActive"`
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
	Config       *Config
}

// Attributes defines specific documentation attributes when a template is
// executed
type Attributes struct {
	Name            string
	Contributors    []Contributor
	NameURL         string
	Year            int
	CapitalName     string
	DonationAddress string
}

func main() {
	flag.BoolVar(&verbose, "v", false, "Verbose output")
	flag.StringVar(&toolDir, "tooldir", "", "Pass in the documentation tool directory if outside tool folder")
	flag.StringVar(&githubToken, "ghtoken", githubToken, "Github authentication token to use when fetching the contributors list")
	flag.Parse()

	wd, err := os.Getwd()
	if err != nil {
		fmt.Println("Documentation tool error cannot get working dir:", err)
		os.Exit(1)
	}

	if strings.Contains(wd, filepath.Join(ref...)) {
		rootDir := filepath.Dir(filepath.Dir(wd))
		repoDir = rootDir
		toolDir = wd
	} else {
		if toolDir == "" {
			fmt.Println("Please set documentation tool directory via the tooldir flag if working outside of tool directory")
			os.Exit(1)
		}
		repoDir = wd
	}

	fmt.Print(core.Banner)
	fmt.Println("This will update and regenerate documentation for the different packages in your repo.")
	fmt.Println()

	if verbose {
		fmt.Println("Fetching configuration...")
	}

	config, err := GetConfiguration()
	if err != nil {
		log.Fatalf("Documentation Generation Tool - GetConfiguration error %s",
			err)
	}

	if verbose {
		fmt.Println("Fetching project directory tree...")
	}

	dirList, err := GetProjectDirectoryTree(&config)
	if err != nil {
		log.Fatalf("Documentation Generation Tool - GetProjectDirectoryTree error %s",
			err)
	}

	var contributors []Contributor
	if config.ContributorFile {
		if verbose {
			fmt.Println("Fetching repository contributor list...")
		}
		contributors, err = GetContributorList(context.TODO(), config.GithubRepo, verbose)
		if err != nil {
			log.Fatalf("Documentation Generation Tool - GetContributorList error: %s", err)
		}

		// Github API missing/deleted user contributors
		contributors = append(contributors, []Contributor{
			// idoall's contributors were forked and merged, so his contributions
			// aren't automatically retrievable
			{
				Login:         "idoall",
				URL:           "https://github.com/idoall",
				Contributions: 1,
			},
			{
				Login:         "starit",
				URL:           "https://github.com/starit",
				Contributions: 1,
			},
		}...)

		sort.Slice(contributors, func(i, j int) bool {
			return contributors[i].Contributions > contributors[j].Contributions
		})

		if verbose {
			fmt.Println("Contributor List Fetched")
			for i := range contributors {
				fmt.Println(contributors[i].Login)
			}
		}
	} else {
		fmt.Println("Contributor list file disabled skipping fetching details")
	}

	if verbose {
		fmt.Println("Fetching template files...")
	}

	tmpl, err := GetTemplateFiles()
	if err != nil {
		log.Fatalf("Documentation Generation Tool - GetTemplateFiles error %s",
			err)
	}

	if verbose {
		fmt.Println("All core systems fetched, updating documentation...")
	}

	UpdateDocumentation(DocumentationDetails{
		dirList,
		tmpl,
		contributors,
		&config,
	})

	fmt.Println("\nDocumentation Generation Tool - Finished")
}

// GetConfiguration retrieves the documentation configuration
func GetConfiguration() (Config, error) {
	var c Config
	configFilePath := filepath.Join(toolDir, "config.json")

	if file.Exists(configFilePath) {
		config, err := os.ReadFile(configFilePath)
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

		return c, nil
	}

	fmt.Println("Creating configuration file, please check to add a different github repository path and change preferences")

	// Set default params for configuration
	c.GithubRepo = DefaultRepo
	c.ContributorFile = true
	c.LicenseFile = true
	c.RootReadme = true
	c.Exclusions.Directories = DefaultExcludedDirectories

	data, err := json.MarshalIndent(c, "", " ")
	if err != nil {
		return c, err
	}

	if err := os.WriteFile(configFilePath, data, file.DefaultPermissionOctal); err != nil {
		return c, err
	}

	return c, nil
}

// GetProjectDirectoryTree uses filepath walk functions to get each individual
// directory name and path to match templates with
func GetProjectDirectoryTree(c *Config) ([]string, error) {
	var directoryData []string
	if c.RootReadme { // Projects root README.md
		directoryData = append(directoryData, repoDir)
	}

	if c.LicenseFile { // Standard license file
		directoryData = append(directoryData, filepath.Join(repoDir, LicenseFile))
	}

	if c.ContributorFile { // Standard contributor file
		directoryData = append(directoryData, filepath.Join(repoDir, ContributorFile))
	}

	walkFn := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// Bypass what is contained in config.json directory exclusion
			if slices.Contains(c.Exclusions.Directories, info.Name()) {
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

	return directoryData, filepath.Walk(repoDir, walkFn)
}

// GetTemplateFiles parses and returns all template files in the documentation
// tree
func GetTemplateFiles() (*template.Template, error) {
	tmpl := template.New("")

	walkFn := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if path == "." || path == ".." {
				return nil
			}

			var tmplExt *template.Template
			tmplExt, err = tmpl.ParseGlob(filepath.Join(path, "*.tmpl"))
			if err != nil {
				fmt.Println(err)
				if strings.Contains(err.Error(), "pattern matches no files") {
					return nil
				}
				return err
			}
			tmpl = tmplExt
			return filepath.SkipDir
		}
		return nil
	}

	return tmpl, filepath.Walk(toolDir, walkFn)
}

// GetContributorList fetches a list of contributors from the Github API endpoint
func GetContributorList(ctx context.Context, repo string, verbose bool) ([]Contributor, error) {
	var contributors []Contributor
	vals := url.Values{}
	vals.Set("per_page", strconv.Itoa(defaultGithubAPIPerPageLimit))

	headers := make(map[string]string)
	if githubToken != "" {
		headers["Authorization"] = "Bearer " + githubToken
		fmt.Println("Using GitHub token for authentication")
	}

	for page := 1; ; page++ {
		vals.Set("page", strconv.Itoa(page))

		contents, err := common.SendHTTPRequest(ctx, http.MethodGet, common.EncodeURLValues(repo+GithubAPIEndpoint, vals), headers, nil, verbose)
		if err != nil {
			return nil, err
		}

		var g ghError
		if err := json.Unmarshal(contents, &g); err == nil && g.Message != "" {
			return nil, fmt.Errorf("GitHub error message: %q Status: %s", g.Message, g.Status)
		}

		var resp []Contributor
		if err := json.Unmarshal(contents, &resp); err != nil {
			return nil, err
		}

		contributors = append(contributors, resp...)
		if len(resp) < defaultGithubAPIPerPageLimit {
			return contributors, nil
		}
	}
}

// GetDocumentationAttributes returns specific attributes for a file template
func GetDocumentationAttributes(packageName string, contributors []Contributor) Attributes {
	return Attributes{
		Name:            GetPackageName(packageName, false),
		Contributors:    contributors,
		NameURL:         GetGoDocURL(packageName),
		Year:            time.Now().Year(),
		CapitalName:     GetPackageName(packageName, true),
		DonationAddress: core.BitcoinDonationAddress,
	}
}

// GetPackageName returns the package name after cleaning path as a string
func GetPackageName(name string, capital bool) string {
	newStrings := strings.Split(name, " ")
	var i int
	if len(newStrings) > 1 {
		// retrieve the latest spacing to define the most childish package name
		i = len(newStrings) - 1
	}
	if capital {
		return cases.Title(language.English).String(strings.ReplaceAll(newStrings[i], "_", " "))
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
func UpdateDocumentation(details DocumentationDetails) {
	for i := range details.Directories {
		cutSet := details.Directories[i][len(repoDir):]
		if cutSet != "" && cutSet[0] == os.PathSeparator {
			cutSet = cutSet[1:]
		}

		data := strings.Split(cutSet, string(os.PathSeparator))

		var temp []string
		for x := range data {
			if data[x] == ".." {
				continue
			}
			if data[x] == "" {
				break
			}
			temp = append(temp, data[x])
		}

		var name string
		if len(temp) == 0 {
			name = "root"
		} else {
			name = strings.Join(temp, " ")
		}

		if slices.Contains(details.Config.Exclusions.Files, name) {
			if verbose {
				fmt.Println("Excluding file:", name)
			}
			continue
		}
		if strings.Contains(name, engineFolder) {
			d, err := os.ReadDir(details.Directories[i])
			if err != nil {
				fmt.Println("Excluding file:", err)
			}
			for x := range d {
				nameSplit := strings.Split(d[x].Name(), ".go")
				engineTemplateName := engineFolder + " " + nameSplit[0]
				if details.Tmpl.Lookup(engineTemplateName) == nil {
					fmt.Printf("Template not found for path %s create new template with {{define \"%s\" -}} TEMPLATE HERE {{end}}\n",
						details.Directories[i],
						name)
					continue
				}
				err = runTemplate(details, filepath.Join(details.Directories[i], nameSplit[0]+".md"), engineTemplateName)
				if err != nil {
					fmt.Println(err)
				}
			}
			continue
		}
		if details.Tmpl.Lookup(name) == nil {
			fmt.Printf("Template not found for path %s create new template with {{define \"%s\" -}} TEMPLATE HERE {{end}}\n",
				details.Directories[i],
				name)
			continue
		}
		var mainPath string
		switch name {
		case LicenseFile, ContributorFile:
			mainPath = details.Directories[i]
		default:
			mainPath = filepath.Join(details.Directories[i], "README.md")
		}

		if err := runTemplate(details, mainPath, name); err != nil {
			log.Println(err)
			continue
		}
	}
}

func runTemplate(details DocumentationDetails, mainPath, name string) error {
	err := os.Remove(mainPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	f, err := os.Create(mainPath)
	if err != nil {
		return err
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			log.Printf("could not close file %s: %v", mainPath, err)
		}
	}(f)

	attr := GetDocumentationAttributes(name, details.Contributors)
	return details.Tmpl.ExecuteTemplate(f, name, attr)
}
