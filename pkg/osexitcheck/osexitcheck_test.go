package osexitcheck

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestOsExitCheckAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), OsExitCheckAnalyzer, "./...")
}
