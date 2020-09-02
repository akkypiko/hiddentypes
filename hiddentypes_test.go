package hiddentypes_test

import (
	"testing"

	"hiddentypes"

	"golang.org/x/tools/go/analysis/analysistest"
)

func init() {
	hiddentypes.Analyzer.Flags.Set("type", "a.Person")
	hiddentypes.Analyzer.Flags.Set("funcs", "log.Printf log.Panicf")
}

// TestAnalyzer is a test for Analyzer.
func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	//defer hiddentypes.ExportSetFlag("a.Person", "log.Printf log.Panicf")()
	analysistest.Run(t, testdata, hiddentypes.Analyzer, "a")
}
