// Command go-newliner runs the go-newliner analyzer.
package main

import (
	"github.com/carsonak/go-newliner/internal/analyzer"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(analyzer.Analyzer)
}
