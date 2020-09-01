package main

import (
	"hiddentypes"
	"golang.org/x/tools/go/analysis/unitchecker"
)

func main() { unitchecker.Main(hiddentypes.Analyzer) }

