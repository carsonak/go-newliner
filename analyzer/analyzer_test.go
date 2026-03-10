package analyzer_test

import (
	"testing"

	"github.com/carsonak/go-newliner/analyzer"
	analysisTest "golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	testdata := analysisTest.TestData()
	analysisTest.RunWithSuggestedFixes(t, testdata, analyzer.Analyzer, "a")
}
