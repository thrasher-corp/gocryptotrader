// The assertioncheck command checks GCT's Testify message conventions.
package main

import "golang.org/x/tools/go/analysis/unitchecker"

func main() {
	unitchecker.Main(newAnalyzer())
}
