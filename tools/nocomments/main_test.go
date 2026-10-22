package main

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestOnlyDirectivesSurvive(t *testing.T) {
	t.Parallel()
	analysistest.Run(t, analysistest.TestData(), analyzer, "sample")
}
