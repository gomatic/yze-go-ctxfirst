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

// TestAHandWrittenContextIsJudged runs the analyzer over a package that reaches
// package context by no import path and declares a context anyway. Its method
// set IS context.Context's, so it stands wherever a context is wanted and the
// rule applies to it; the sibling carrying two of the four methods does not.
func TestAHandWrittenContextIsJudged(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), ctxfirst.Analyzer, "handwritten")
}

func TestRegistrationIsWellFormed(t *testing.T) {
	assert.NoError(t, ctxfirst.Registration.Validate())
	assert.Equal(t, "yze/ctxfirst", ctxfirst.Registration.RuleID())
	assert.Same(t, ctxfirst.Analyzer, ctxfirst.Registration.Analyzer)
}
