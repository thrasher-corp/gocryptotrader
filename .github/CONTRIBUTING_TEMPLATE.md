<!-- use this template to generate the contributor docs with the following command: `$ lingo run docs --template CONTRIBUTING_TEMPLATE.md  --output CONTRIBUTING.md` -->
# Contributing

## Please contribute

All PR's are welcome

## Coding Style

In order to maintain a consistent style across the codebase, the following coding style has been adopted:

- Function names use PascalCase (func SomeFunc()).
- Function names using acronyms are capitilised (func SendHTTPRequest()).
- Variable names use CamelCase (var someVar()).
- Coding style uses gofmt.
- Const variables are CamelCase depending on exported items.
- In line with gofmt, for loops and if statements don't require parenthesis.

Block style example:

```go
func SendHTTPRequest(method, path string, headers map[string]string, body io.Reader) (string, error) {
    result := strings.ToUpper(method)

    if result != http.MethodPost && result != http.MethodGet && result != http.MethodDelete {
        return "", errors.New("Invalid HTTP method specified.")
    }

    req, err := http.NewRequest(method, path, body)
    if err != nil {
        return "", err
    }

    for k, v := range headers {
        req.Header.Add(k, v)
    }
    ...
}
```

## Effective Go Guidelines

[CodeLingo](https://codelingo.io) automatically checks every pull request against the following guidelines from [Effective Go](https://golang.org/doc/effective_go.html).

{{range .}}

### {{.title}}

{{.body}}
{{end}}
