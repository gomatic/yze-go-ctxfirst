package ctxfirst_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/tools/go/analysis/analysistest"

	ctxfirst "github.com/gomatic/yze-go-ctxfirst"
)

func TestContextMustBeFirst(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), ctxfirst.Analyzer, "a")
}

// TestRunJudgesNoPackageOutOfContextsReach runs the analyzer over a package the
// build carries to package context by no path. It has no context to find, so it
// walks no signature — the shapes it declares would be reported in a package
// that does reach context.
func TestRunJudgesNoPackageOutOfContextsReach(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), ctxfirst.Analyzer, "nocontext")
}

func TestRegistrationIsWellFormed(t *testing.T) {
	assert.NoError(t, ctxfirst.Registration.Validate())
	assert.Equal(t, "yze/ctxfirst", ctxfirst.Registration.RuleID())
	assert.Same(t, ctxfirst.Analyzer, ctxfirst.Registration.Analyzer)
}
