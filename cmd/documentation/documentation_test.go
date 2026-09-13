package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunTemplateNormalizesMarkdown(t *testing.T) {
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
			assert.Equal(t, tt.expected, string(contents), "runTemplate should normalize generated Markdown")
			assert.Equal(t, tt.expected, normalizeMarkdown(tt.expected), "normalizeMarkdown should be idempotent")

			info, err := os.Stat(outputPath)
			require.NoError(t, err, "stat generated documentation must not error")
			assert.Zero(t, info.Mode().Perm()&0o111, "generated documentation should not be executable")
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
