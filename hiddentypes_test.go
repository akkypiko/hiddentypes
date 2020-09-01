package hiddentypes_test

import (
	"testing"

	"hiddentypes"
	"golang.org/x/tools/go/analysis/analysistest"
)

// TestAnalyzer is a test for Analyzer.
func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, hiddentypes.Analyzer, "a")
}

