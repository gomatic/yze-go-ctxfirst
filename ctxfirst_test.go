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

func TestRegistrationIsWellFormed(t *testing.T) {
	assert.NoError(t, ctxfirst.Registration.Validate())
	assert.Equal(t, "yze/ctxfirst", ctxfirst.Registration.RuleID())
	assert.Same(t, ctxfirst.Analyzer, ctxfirst.Registration.Analyzer)
}
