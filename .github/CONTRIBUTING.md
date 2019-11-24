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

### Comment First Word as Subject

Doc comments work best as complete sentences, which allow a wide variety of automated presentations.
The first sentence should be a one-sentence summary that starts with the name being declared.

### Good Package Name

It's helpful if everyone using the package can use the same name
to refer to its contents, which implies that the package name should
be good: short, concise, evocative. By convention, packages are
given lower case, single-word names; there should be no need for
underscores or mixedCaps. Err on the side of brevity, since everyone
using your package will be typing that name. And don't worry about
collisions a priori. The package name is only the default name for
imports; it need not be unique across all source code, and in the
rare case of a collision the importing package can choose a different
name to use locally. In any case, confusion is rare because the file
name in the import determines just which package is being used.

### Package Comment

Every package should have a package comment, a block comment preceding the package clause.
For multi-file packages, the package comment only needs to be present in one file, and any one will do.
The package comment should introduce the package and provide information relevant to the package as a
whole. It will appear first on the godoc page and should set up the detailed documentation that follows.

### Single Method Interface Name

By convention, one-method interfaces are named by the method name plus an -er suffix
or similar modification to construct an agent noun: Reader, Writer, Formatter, CloseNotifier etc.

There are a number of such names and it's productive to honor them and the function names they capture.
Read, Write, Close, Flush, String and so on have canonical signatures and meanings. To avoid confusion,
don't give your method one of those names unless it has the same signature and meaning. Conversely,
if your type implements a method with the same meaning as a method on a well-known type, give it the
same name and signature; call your string-converter method String not ToString.

### Avoid Annotations in Comments

Comments do not need extra formatting such as banners of stars. The generated output
may not even be presented in a fixed-width font, so don't depend on spacing for alignment—godoc,
like gofmt, takes care of that. The comments are uninterpreted plain text, so HTML and other
annotations such as _this_ will reproduce verbatim and should not be used. One adjustment godoc
does do is to display indented text in a fixed-width font, suitable for program snippets.
The package comment for the fmt package uses this to good effect.
