package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
	"golang.org/x/net/html"
)

func TestRunTemplateNormalisesMarkdown(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "prose whitespace",
			input:    "# Heading\r\n\r\n\r\nbefore\tafter  \r\n\r\n",
			expected: "# Heading\n\nbefore  after\n",
		},
		{
			name:     "fenced whitespace",
			input:    "```go\r\n\tfirst  \r\n\r\n\r\n\tsecond\r\n```\r\n",
			expected: "```go\n\tfirst  \n\n\n\tsecond\n```\n",
		},
		{
			name:     "tilde fence",
			input:    "~~~text\n\tcontent\n~~~~\n",
			expected: "~~~text\n\tcontent\n~~~~\n",
		},
		{
			name:     "fence-like content",
			input:    "````text\n```not a closing fence\n\tcontent\n````\n",
			expected: "````text\n```not a closing fence\n\tcontent\n````\n",
		},
		{
			name:     "indented fence",
			input:    "   ```text\n\tcontent  \n   ```\n",
			expected: "   ```text\n\tcontent  \n   ```\n",
		},
		{
			name:     "unmatched fence",
			input:    "```text\n\tcontent  \n\n\n",
			expected: "```text\n\tcontent  \n",
		},
		{
			name:     "code-indented fence",
			input:    "    ```text  \n\tprose  \n",
			expected: "    ```text  \n\tprose  \n",
		},
		{
			name:     "fence delimiter whitespace",
			input:    "```text   \n\tcontent  \n```   \n",
			expected: "```text\n\tcontent  \n```\n",
		},
		{
			name:     "list nested fence",
			input:    "1. Example\n\n    ```make\n    all:\n    \t@echo ok\n    ```\n",
			expected: "1. Example\n\n    ```make\n    all:\n    \t@echo ok\n    ```\n",
		},
		{
			name:     "blockquote nested fence",
			input:    "> ```make\n> all:\n> \t@echo ok\n> ```\n",
			expected: "> ```make\n> all:\n> \t@echo ok\n> ```\n",
		},
		{
			name:     "list marker inside fence",
			input:    "```text\n- ```\n\tcontent  \n```\n",
			expected: "```text\n- ```\n\tcontent  \n```\n",
		},
		{
			name:     "blockquote marker inside fence",
			input:    "```text\n> ```\n\tcontent  \n```\n",
			expected: "```text\n> ```\n\tcontent  \n```\n",
		},
		{
			name:     "ordered list marker inside fence",
			input:    "```text\n1. ```\n\tcontent  \n```\n",
			expected: "```text\n1. ```\n\tcontent  \n```\n",
		},
		{
			name:     "blockquote ends before fence",
			input:    "# Probe\n\n> ```text\n> content\n\n## Heading\n\n\n\nprose\twith tab  \n",
			expected: "# Probe\n\n> ```text\n> content\n\n## Heading\n\nprose   with tab\n",
		},
		{
			name:     "tab-indented list fence",
			input:    "- item\n\n\t```make\n\tall:\n\t\t@echo ok\n\t```\n",
			expected: "- item\n\n    ```make\n\tall:\n\t\t@echo ok\n    ```\n",
		},
		{
			name:     "empty list item converges",
			input:    "- \n    ```make\n    all:\n    \t@echo ok\n    ```\n",
			expected: "-\n    ```make\n    all:\n    \t@echo ok\n    ```\n",
		},
		{
			name:     "ordered list fence in blockquote",
			input:    "> 1. item\n>     ```make\n>     all:\n>     \t@echo ok\n>     ```\n",
			expected: "> 1. item\n>     ```make\n>     all:\n>     \t@echo ok\n>     ```\n",
		},
		{
			name:     "non-closing fence preserves whitespace",
			input:    "```text\n```not closing   \n\tcontent  \n```\n",
			expected: "```text\n```not closing   \n\tcontent  \n```\n",
		},
		{
			name:     "list item ends before fence",
			input:    "- item\n\n  ```text\n  code\n\n## Heading\n\n\n\nprose\twith tab   \n",
			expected: "- item\n\n  ```text\n  code\n\n## Heading\n\nprose   with tab\n",
		},
		{
			name:     "unindented fence after list fence",
			input:    "- ```make\n```\nall:\n\t@echo ok\n```\n",
			expected: "- ```make\n```\nall:\n\t@echo ok\n```\n",
		},
		{
			name:     "tab-indented closer after list fence",
			input:    "- ```\n```go\n\t```\n\tcode\n",
			expected: "- ```\n```go\n\t```\n\tcode\n",
		},
		{
			name:     "blockquote tab before fence",
			input:    "> \t```make\n> \tall:\n> \t\t@echo ok\n> \t```\n",
			expected: ">   ```make\n> \tall:\n> \t\t@echo ok\n>   ```\n",
		},
		{
			name:     "blockquote tab before fence character at end of input",
			input:    "> \t~",
			expected: ">   ~\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tmpl := template.Must(template.New("documentation").Parse("{{define \"documentation\"}}" + tt.input + "{{end}}"))
			outputPath := filepath.Join(t.TempDir(), "README.md")
			err := runTemplate(DocumentationDetails{Tmpl: tmpl}, outputPath, "documentation")
			require.NoError(t, err, "runTemplate must not error")

			contents, err := os.ReadFile(outputPath)
			require.NoError(t, err, "reading generated documentation must not error")
			assert.Equal(t, tt.expected, string(contents), "runTemplate should normalise generated Markdown")
			assert.Equal(t, tt.expected, normaliseMarkdown(tt.expected), "normaliseMarkdown should be idempotent")

			info, err := os.Stat(outputPath)
			require.NoError(t, err, "stat generated documentation must not error")
			assert.Zero(t, info.Mode().Perm()&0o111, "generated documentation should not be executable")
		})
	}
}

func TestRelativeRepoRoot(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{name: "root", path: filepath.Join(root, "README.md"), expected: "."},
		{name: "nested", path: filepath.Join(root, "backtester", "btcli", "README.md"), expected: "../.."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			actual, err := relativeRepoRoot(tt.path, root)
			require.NoError(t, err, "relativeRepoRoot must not error")
			assert.Equal(t, tt.expected, actual, "relativeRepoRoot should return the README-relative repository root")
		})
	}
}

func TestMarkdownDestinationsAreRepositoryRelative(t *testing.T) {
	t.Parallel()
	repositoryRoot, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err, "repository root must resolve")
	var sources []string
	if _, err := os.Stat(filepath.Join(repositoryRoot, ".git")); err == nil {
		// Read the index in a checkout, so untracked and ignored paths - editor history copies,
		// scratch notes, the package cmd/exchange_template scaffolds while this test runs - are
		// never linted.
		command := exec.CommandContext(t.Context(), "git", "-c", "safe.directory=*", "ls-files", "-z", "*.md", "*.tmpl")
		command.Dir = repositoryRoot
		output, err := command.Output()
		require.NoError(t, err, "tracked Markdown sources must be listed")
		for path := range strings.SplitSeq(string(output), "\x00") {
			if path != "" {
				sources = append(sources, filepath.Join(repositoryRoot, filepath.FromSlash(path)))
			}
		}
	} else {
		// A tree exported without repository metadata, such as the image the Docker job builds, has
		// no index to read, so walk it instead.
		require.NoError(t, filepath.WalkDir(repositoryRoot, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				switch name := d.Name(); {
				case path == repositoryRoot:
					return nil
				case name == ".git", name == "vendor", name == "node_modules":
					return fs.SkipDir
				}
				return nil
			}
			if ext := filepath.Ext(d.Name()); ext == ".md" || ext == ".tmpl" {
				sources = append(sources, path)
			}
			return nil
		}), "repository Markdown sources must be walked")
	}
	require.NotEmpty(t, sources, "repository must contain Markdown sources")

	for _, path := range sources {
		t.Run(filepath.ToSlash(strings.TrimPrefix(path, repositoryRoot+string(filepath.Separator))), func(t *testing.T) {
			t.Parallel()
			contents, err := os.ReadFile(path)
			require.NoError(t, err, "Markdown source must be readable")
			assert.Empty(t, markdownDestinationIssues(path, contents, filepath.Ext(path) != ".tmpl"),
				"Markdown destinations should be repository-relative, revision-independent and resolvable")
		})
	}
}

type markdownDestination struct {
	value   string
	isImage bool
}

func markdownDestinationIssues(sourcePath string, contents []byte, checkExists bool) []string {
	document := markdownParser.Parse(text.NewReader(contents))
	var destinations []markdownDestination
	if err := ast.Walk(document, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch node := node.(type) {
		case *ast.Link:
			destinations = append(destinations, markdownDestination{value: string(node.Destination)})
		case *ast.Image:
			destinations = append(destinations, markdownDestination{value: string(node.Destination), isImage: true})
		case *ast.RawHTML:
			htmlDestinations, err := rawHTMLDestinations(node.Segments.Value(contents))
			if err != nil {
				return ast.WalkStop, err
			}
			destinations = append(destinations, htmlDestinations...)
		case *ast.HTMLBlock:
			htmlContents := node.Lines().Value(contents)
			if node.HasClosure() {
				htmlContents = append(htmlContents, node.ClosureLine.Value(contents)...)
			}
			htmlDestinations, err := rawHTMLDestinations(htmlContents)
			if err != nil {
				return ast.WalkStop, err
			}
			destinations = append(destinations, htmlDestinations...)
		}
		return ast.WalkContinue, nil
	}); err != nil {
		return []string{"Markdown syntax tree could not be walked: " + err.Error()}
	}

	issues := make([]string, 0)
	for _, destination := range destinations {
		parsed, err := url.Parse(destination.value)
		if err != nil {
			issues = append(issues, "invalid destination "+destination.value+": "+err.Error())
			continue
		}
		external := parsed.IsAbs() || parsed.Host != ""
		if destination.isImage && external && isMasterPinnedRepositoryURL(parsed) {
			issues = append(issues, "repository image must not be pinned to master: "+destination.value)
		}
		if external || destination.value == "" || strings.HasPrefix(destination.value, "#") {
			continue
		}
		if strings.HasPrefix(parsed.Path, "/") {
			issues = append(issues, "repository destination must be relative to its source file: "+destination.value)
			continue
		}
		if !checkExists || parsed.Path == "" {
			continue
		}
		if _, err := os.Stat(filepath.Join(filepath.Dir(sourcePath), filepath.FromSlash(parsed.Path))); err != nil {
			issues = append(issues, "repository destination does not exist "+destination.value+": "+err.Error())
		}
	}
	return issues
}

func rawHTMLDestinations(contents []byte) ([]markdownDestination, error) {
	var destinations []markdownDestination
	tokenizer := html.NewTokenizer(strings.NewReader(string(contents)))
	for {
		switch tokenizer.Next() {
		case html.ErrorToken:
			if errors.Is(tokenizer.Err(), io.EOF) {
				return destinations, nil
			}
			return nil, fmt.Errorf("cannot parse raw HTML: %w", tokenizer.Err())
		case html.StartTagToken, html.SelfClosingTagToken:
			token := tokenizer.Token()
			for _, attribute := range token.Attr {
				switch strings.ToLower(attribute.Key) {
				case "href":
					destinations = append(destinations, markdownDestination{value: attribute.Val})
				case "src":
					destinations = append(destinations, markdownDestination{value: attribute.Val, isImage: true})
				}
			}
		}
	}
}

func isMasterPinnedRepositoryURL(destination *url.URL) bool {
	path := strings.ToLower(destination.EscapedPath())
	switch strings.ToLower(destination.Hostname()) {
	case "github.com":
		return strings.HasPrefix(path, "/thrasher-corp/gocryptotrader/") && strings.Contains(path, "/master/")
	case "raw.githubusercontent.com":
		return strings.HasPrefix(path, "/thrasher-corp/gocryptotrader/master/")
	default:
		return false
	}
}

func TestMarkdownDestinationIssues(t *testing.T) {
	t.Parallel()
	directory := t.TempDir()
	sourcePath := filepath.Join(directory, "README.md")
	require.NoError(t, os.WriteFile(filepath.Join(directory, "logo.png"), []byte("fixture"), 0o600), "image fixture must be written")
	require.NoError(t, os.WriteFile(filepath.Join(directory, "zz%probe.md"), []byte("fixture"), 0o600), "percent fixture must be written")

	for _, test := range []struct {
		name     string
		contents string
		issue    string
	}{
		{name: "relative raw HTML image", contents: `<img src="logo.png" alt="logo">`},
		{name: "encoded percent Markdown link", contents: `[percent](zz%25probe.md)`},
		{name: "inline raw HTML image", contents: `logo <img src="/docs/assets/logo.png" alt="logo"> here`, issue: "must be relative"},
		{name: "root-relative raw HTML image", contents: `<img src="/docs/assets/logo.png" alt="logo">`, issue: "must be relative"},
		{name: "root-relative raw HTML link", contents: `<a href="/docs/CODING_GUIDELINES.md">guidelines</a>`, issue: "must be relative"},
		{name: "master-pinned Markdown image", contents: `![logo](https://raw.githubusercontent.com/thrasher-corp/gocryptotrader/master/common/gctlogo.png)`, issue: "must not be pinned to master"},
		{name: "master-pinned raw HTML image", contents: `<img src="https://raw.githubusercontent.com/thrasher-corp/gocryptotrader/master/common/gctlogo.png" alt="logo">`, issue: "must not be pinned to master"},
		{name: "missing raw HTML image", contents: `<img src="missing.png" alt="logo">`, issue: "does not exist"},
		{name: "missing Markdown link", contents: `[missing](missing.md)`, issue: "does not exist"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			issues := markdownDestinationIssues(sourcePath, []byte(test.contents), true)
			if test.issue == "" {
				assert.Empty(t, issues, "valid destination should pass")
				return
			}
			assert.Contains(t, strings.Join(issues, "\n"), test.issue, "invalid destination should report the expected issue")
		})
	}
}

func TestRunTemplateKeepsExistingFileOnTemplateError(t *testing.T) {
	t.Parallel()
	outputPath := filepath.Join(t.TempDir(), "README.md")
	require.NoError(t, os.WriteFile(outputPath, []byte("old contents\n"), 0o644), "fixture must be written")

	tmpl := template.Must(template.New("documentation").Parse(`{{define "documentation"}}new contents {{.Missing}}{{end}}`))
	err := runTemplate(DocumentationDetails{Tmpl: tmpl}, outputPath, "documentation")
	var execErr template.ExecError
	require.ErrorAs(t, err, &execErr, "runTemplate must return the template execution error")

	contents, err := os.ReadFile(outputPath)
	require.NoError(t, err, "existing documentation must still be readable")
	assert.Equal(t, "old contents\n", string(contents), "a template error should leave existing documentation untouched")
}

func TestRunTemplateReplacesExistingFile(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not expose executable and read-only permission bits consistently")
	}

	for _, permissions := range []os.FileMode{0o444, 0o755} {
		t.Run(permissions.String(), func(t *testing.T) {
			t.Parallel()
			outputPath := filepath.Join(t.TempDir(), "README.md")
			require.NoError(t, os.WriteFile(outputPath, []byte("old contents"), permissions), "fixture must be written")
			require.NoError(t, os.Chmod(outputPath, permissions), "fixture permissions must be applied")

			tmpl := template.Must(template.New("documentation").Parse(`{{define "documentation"}}new contents{{end}}`))
			err := runTemplate(DocumentationDetails{Tmpl: tmpl}, outputPath, "documentation")
			require.NoError(t, err, "runTemplate must replace an existing file")

			contents, err := os.ReadFile(outputPath)
			require.NoError(t, err, "generated documentation must be readable")
			assert.Equal(t, "new contents\n", string(contents), "generated documentation should replace existing contents")

			info, err := os.Stat(outputPath)
			require.NoError(t, err, "generated documentation must be statable")
			assert.Zero(t, info.Mode().Perm()&0o111, "generated documentation should not be executable")
			assert.NotZero(t, info.Mode().Perm()&0o200, "generated documentation should not stay read-only")
		})
	}
}

func TestGetContributorList(t *testing.T) {
	t.Parallel()

	c, err := GetContributorList(t.Context(), DefaultRepo, true)
	require.NoError(t, err, "GetContributorList must not error")
	require.NotEmpty(t, c, "GetContributorList must not return empty list")
}
