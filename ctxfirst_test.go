package ctxfirst_test

import (
	"testing"

	ctxfirst "github.com/gomatic/yze-go-ctxfirst"
	"github.com/stretchr/testify/assert"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestContextMustBeFirst(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), ctxfirst.Analyzer, "a")
}

func TestRegistrationIsWellFormed(t *testing.T) {
	assert.NoError(t, ctxfirst.Registration.Validate())
	assert.Equal(t, "yze/go/ctxfirst", ctxfirst.Registration.RuleID())
	assert.Same(t, ctxfirst.Analyzer, ctxfirst.Registration.Analyzer)
}
