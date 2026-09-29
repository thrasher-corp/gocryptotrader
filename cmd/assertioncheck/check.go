package main

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/types"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/types/typeutil"
)

var messageWord = regexp.MustCompile(`(?i)\b(should|must)\b`)

func newAnalyzer() *analysis.Analyzer {
	var sourceFiles string
	analyzer := &analysis.Analyzer{
		Name: "assertionmessages",
		Doc:  "check Testify message wording and formatted assertion variants",
		Run: func(pass *analysis.Pass) (any, error) {
			return checkAssertions(pass, sourceFiles)
		},
	}
	analyzer.Flags.StringVar(&sourceFiles, "files", "", "NUL-separated list of absolute source paths to check")
	return analyzer
}

func checkAssertions(pass *analysis.Pass, sourceFiles string) (any, error) {
	var selected map[string]bool
	if sourceFiles != "" {
		data, err := os.ReadFile(sourceFiles)
		if err != nil {
			return nil, fmt.Errorf("reading tracked source paths: %w", err)
		}
		selected = make(map[string]bool)
		for path := range strings.SplitSeq(string(data), "\x00") {
			if path != "" {
				selected[filepath.Clean(path)] = true
			}
		}
	}
	for _, file := range pass.Files {
		if ast.IsGenerated(file) || selected != nil && !selected[filepath.Clean(pass.Fset.Position(file.Pos()).Filename)] {
			continue
		}
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			checkAssertion(pass, call)
			return true
		})
	}
	return nil, nil
}

func checkAssertion(pass *analysis.Pass, call *ast.CallExpr) {
	// Resolve identity and message parameters, so aliases, assertion methods and multiline calls do not depend on source spelling.
	// testifylint resolves calls same way: https://github.com/antonboom/testifylint/blob/720065d36f434c616017441aa8f2069e621b5e44/internal/checkers/call_meta.go
	fn := typeutil.StaticCallee(pass.TypesInfo, call)
	if fn == nil || fn.Pkg() == nil {
		return
	}
	var wrongWord, requiredWord string
	switch fn.Pkg().Path() {
	case "github.com/stretchr/testify/assert":
		wrongWord, requiredWord = "must", "should"
	case "github.com/stretchr/testify/require":
		wrongWord, requiredWord = "should", "must"
	default:
		return
	}

	sig, ok := pass.TypesInfo.TypeOf(call.Fun).(*types.Signature)
	if !ok {
		return
	}
	messageIndex := messagePosition(sig)
	if messageIndex < 0 || messageIndex >= len(call.Args) {
		return
	}
	value := pass.TypesInfo.Types[call.Args[messageIndex]].Value
	if value == nil || value.Kind() != constant.String {
		// Dynamic messages and expanded msgAndArgs cannot be checked without
		// evaluating the program. Do not guess their wording or formatting.
		return
	}
	message := constant.StringVal(value)
	for _, word := range messageWord.FindAllString(message, -1) {
		if strings.EqualFold(word, wrongWord) {
			pass.Reportf(call.Args[messageIndex].Pos(), "%s message uses %q; use %q", fn.FullName(), wrongWord, requiredWord)
			break
		}
	}

	formatted := strings.HasSuffix(fn.Name(), "f")
	alternative := fn.Name() + "f"
	if formatted {
		alternative = strings.TrimSuffix(fn.Name(), "f")
	}
	if !hasAlternative(fn, alternative) {
		return
	}
	if !strings.Contains(message, "%") {
		if formatted {
			pass.Reportf(call.Pos(), "%s has a plain message; use %s", fn.FullName(), alternative)
		}
		return
	}
	if !formatted && formatsMessage(message) {
		pass.Reportf(call.Pos(), "%s has a format message; use %s", fn.FullName(), alternative)
	}
}

// formatsMessage reports whether message formats anything, such as verb or "%%".
// Formatting it with no arguments marks every verb and resolves every "%%", so fmt judges without copying Go's printf parser.
// Lone trailing "%" prints unchanged, so its %!(NOVERB) marker maps back to "%".
// Malformed formats remain testifylint's job.
func formatsMessage(message string) bool {
	// Empty argument list stops vet reporting non-constant format string.
	return strings.ReplaceAll(fmt.Sprintf(message, []any{}...), "%!(NOVERB)", "%") != message
}

func messagePosition(sig *types.Signature) int {
	// Adapted from testifylint's signature-based lookup: https://github.com/antonboom/testifylint/blob/720065d36f434c616017441aa8f2069e621b5e44/internal/checkers/formatter.go
	// Copyright (c) 2022 Anton Telyshev. See THIRD_PARTY_NOTICES at the repository root for its MIT licence.
	// Lookup also covers dot imports and method expressions.
	// Use call signature so receiver arguments align.
	for i := range sig.Params().Len() {
		param := sig.Params().At(i)
		switch param.Name() {
		case "msgAndArgs":
			if _, ok := param.Type().(*types.Slice); ok {
				return i
			}
		case "msg", "format":
			if basic, ok := param.Type().(*types.Basic); ok && basic.Kind() == types.String {
				return i
			}
		}
	}
	return -1
}

func hasAlternative(fn *types.Func, name string) bool {
	sig, ok := fn.Type().(*types.Signature)
	if !ok {
		return false
	}
	if recv := sig.Recv(); recv != nil {
		alternative, _, _ := types.LookupFieldOrMethod(recv.Type(), false, fn.Pkg(), name)
		_, ok = alternative.(*types.Func)
		return ok
	}
	_, ok = fn.Pkg().Scope().Lookup(name).(*types.Func)
	return ok
}
