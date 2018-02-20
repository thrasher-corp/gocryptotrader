Coding Style
===============

In order to maintain a consistent style across the codebase, the following coding style has been adopted:

- Function names use PascalCase (func SomeFunc()).
- Function names using acronyms are capitilised (func SendHTTPRequest()).
- Variable names use CamelCase (var someVar()).
- Coding style uses gofmt.
- Const variables are CamelCase depending on exported items.
- In line with gofmt, for loops and if statements don't require paranthesis.

Block style example:
```go
func SendHTTPRequest(method, path string, headers map[string]string, body io.Reader) (string, error) {
	result := strings.ToUpper(method)

	if result != "POST" && result != "GET" && result != "DELETE" {
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
