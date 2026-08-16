package ctxfirst

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// source is a Go file's text, type-checked to build a pass by hand.
type source string

// lineNumber is a 1-based line in a constructed source.
type lineNumber int

// TestMessageStatesTheRuleTheDocDeclares pins the diagnostic to the package
// comment. The message used to assert "context.Context must be the first
// parameter" — the rule as it stood before the leading-prefix widening — and
// nothing failed, because the only assertion on it was a testdata `want`
// carrying the same string the constant did. Reading the doc comment instead
// means the message cannot state one rule while the package documents another.
func TestMessageStatesTheRuleTheDocDeclares(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "ctxfirst.go", nil, parser.ParseComments)
	require.NoError(t, err)
	require.NotNil(t, file.Doc)
	assert.Contains(t, unwrapped(file.Doc.Text()), message)
}

// unwrapped joins a doc comment's wrapped lines so a sentence spanning two of
// them is one string. Only the line breaks gofmt puts there are removed; the
// words are the doc's own.
func unwrapped(text string) string {
	return strings.Join(strings.Fields(text), " ")
}

// TestReportedColumnIsTheOffendingParameterType pins the column of the
// diagnostic. analysistest matches by line and the corpus Finding is {rule,
// path, line}, so moving the report from the parameter's type to the parameter
// list left every instrument in the suite silent while every diagnostic in
// every repo moved to the opening parenthesis.
func TestReportedColumnIsTheOffendingParameterType(t *testing.T) {
	const src source = `package p

import "context"

func f(n int, ctx context.Context) { _ = n; _ = ctx }
`
	fset, diagnostics := analyze(t, src)
	require.Len(t, diagnostics, 1)
	position := fset.Position(diagnostics[0].Pos)
	assert.Equal(t, 5, position.Line)
	assert.Equal(t, 19, position.Column, "context.Context begins at column 19; the parameter list opens at 7")
	assert.Equal(t, message, diagnostics[0].Message)
}

// TestEveryOffendingParameterIsReported drives the multiplicity of the rule,
// which every fixture in this repo and every corpus case held constant at one.
// Adding a `return` after the report halves the analyzer's output on a
// signature burying two contexts, and it survived the whole suite, all twelve
// corpus cases and the 100.0% coverage gate — an author who fixed the first
// finding would have been told the signature was clean.
func TestEveryOffendingParameterIsReported(t *testing.T) {
	const src source = `package p

import "context"

func f(
	n int,
	first context.Context,
	s string,
	second context.Context,
) {
	_, _, _, _ = n, first, s, second
}
`
	fset, diagnostics := analyze(t, src)
	require.Len(t, diagnostics, 2, "one finding per offending field, not one per signature")
	assert.Equal(t, 7, fset.Position(diagnostics[0].Pos).Line)
	assert.Equal(t, 9, fset.Position(diagnostics[1].Pos).Line)
}

// reportedLine is the 1-based line a diagnostic names.
func reportedLine(fset *token.FileSet, diagnostic analysis.Diagnostic) lineNumber {
	return lineNumber(fset.Position(diagnostic.Pos).Line)
}

// analyze runs the analyzer over src and returns the diagnostics it reported.
func analyze(t *testing.T, src source) (*token.FileSet, []analysis.Diagnostic) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "p.go", string(src), 0)
	require.NoError(t, err)
	info := &types.Info{Types: map[ast.Expr]types.TypeAndValue{}, Defs: map[*ast.Ident]types.Object{}}
	conf := types.Config{Importer: importer.Default()}
	_, err = conf.Check("example.test/p", fset, []*ast.File{file}, info)
	require.NoError(t, err)

	var diagnostics []analysis.Diagnostic
	pass := &analysis.Pass{
		Fset:      fset,
		Files:     []*ast.File{file},
		TypesInfo: info,
		ResultOf:  map[*analysis.Analyzer]any{inspect.Analyzer: inspector.New([]*ast.File{file})},
		Report:    func(d analysis.Diagnostic) { diagnostics = append(diagnostics, d) },
	}
	result, err := run(pass)
	require.NoError(t, err)
	require.Nil(t, result)
	return fset, diagnostics
}
