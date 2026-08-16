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
	doc := unwrapped(file.Doc.Text())
	assert.Contains(t, doc, message)
	assert.Contains(t, doc, variadicMessage,
		"the variadic diagnostic is a second message and states a second remedy, so the doc owes it the same")
}

// unwrapped joins a doc comment's wrapped lines so a sentence spanning two of
// them is one string. Only the line breaks gofmt puts there are removed; the
// words are the doc's own.
func unwrapped(text string) string {
	return strings.Join(strings.Fields(text), " ")
}

// TestReportedColumnIsTheOffendingParameter pins the column of the diagnostic.
// analysistest matches by line and the corpus Finding is {rule, path, line}, so
// moving the report from the parameter to the parameter list left every
// instrument in the suite silent while every diagnostic in every repo moved to
// the opening parenthesis. The column is the offending PARAMETER's name, which
// is what makes one finding per parameter distinguishable from another in a
// grouped field — the field's type is one position for however many parameters
// it declares, and both drivers collapse two diagnostics that share one.
func TestReportedColumnIsTheOffendingParameter(t *testing.T) {
	const src source = `package p

import "context"

func f(n int, ctx context.Context) { _ = n; _ = ctx }
`
	fset, diagnostics := analyze(t, src)
	require.Len(t, diagnostics, 1)
	position := fset.Position(diagnostics[0].Pos)
	assert.Equal(t, 5, position.Line)
	assert.Equal(t, 15, position.Column, "the parameter ctx begins at column 15; the parameter list opens at 7")
	assert.Equal(t, message, diagnostics[0].Message)
}

// TestAnUnnamedParameterIsReportedAtItsType is the other half of that rule: a
// parameter with no name has nothing to point at but its type, and one unnamed
// field declares exactly one parameter, so the count is unaffected.
func TestAnUnnamedParameterIsReportedAtItsType(t *testing.T) {
	const src source = `package p

import "context"

func f(int, context.Context) {}
`
	fset, diagnostics := analyze(t, src)
	require.Len(t, diagnostics, 1)
	assert.Equal(t, 13, fset.Position(diagnostics[0].Pos).Column,
		"context.Context begins at column 13; the parameter list opens at 7")
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

// TestAVariadicContextAfterANonContextIsReported closes the escape a fix in
// this session opened. Exempting `...context.Context` made the rule switchable
// off by one token, and the token costs the author nothing: the call site
// `Handle(3, ctx)` is byte-identical under both spellings and compiles under
// both, while the honest remedy — reordering — breaks every call site. The
// exemption's stated reason was that the Go spec forces a variadic's position,
// but an author who writes `...` chose that position freely, and the analyzer
// already reports the genuinely-forced case (a method whose order an interface
// dictates) without exempting it.
func TestAVariadicContextAfterANonContextIsReported(t *testing.T) {
	const src source = `package p

import "context"

func f(n int, ctxs ...context.Context) { _, _ = n, ctxs }
`
	fset, diagnostics := analyze(t, src)
	require.Len(t, diagnostics, 1)
	assert.Equal(t, 5, fset.Position(diagnostics[0].Pos).Line)
	assert.Equal(t, variadicMessage, diagnostics[0].Message,
		"the ordinary message prescribes a reorder the Go spec forbids for a variadic")
}

// TestALeadingVariadicContextIsSilent is the other side of that boundary: a
// variadic context with no non-context before it breaks no prefix, so the
// finding above is about the position and not about the ellipsis.
func TestALeadingVariadicContextIsSilent(t *testing.T) {
	const src source = `package p

import "context"

func f(ctxs ...context.Context) { _ = ctxs }
`
	_, diagnostics := analyze(t, src)
	assert.Empty(t, diagnostics)
}

// TestTheVariadicRemedyCompiles runs the remedy the diagnostic prescribes.
// s04 item 8 asks whether the author can do anything except silence the rule,
// and the answer has to be a shape the compiler accepts: a leading
// []context.Context is silent, and unlike the variadic it cannot be omitted at
// the call site.
func TestTheVariadicRemedyCompiles(t *testing.T) {
	const src source = `package p

import "context"

func f(ctxs []context.Context, n int) { _, _ = ctxs, n }
`
	_, diagnostics := analyze(t, src)
	assert.Empty(t, diagnostics, "a leading slice of contexts is the remedy the message names")
}

// TestFieldSpellingDoesNotChangeTheCount pins the promise the package comment
// makes twice. `(a, b context.Context)` and `(a context.Context, b
// context.Context)` are the same positional list, so they draw the same number
// of findings; reporting once per FIELD made the count a property of the
// spelling, which also weakened C4 — in the grouped spelling, fixing one of the
// two names was answered with the same single finding.
func TestFieldSpellingDoesNotChangeTheCount(t *testing.T) {
	const grouped source = `package p

import "context"

func f(n int, a, b context.Context) { _, _, _ = n, a, b }
`
	const split source = `package p

import "context"

func f(n int, a context.Context, b context.Context) { _, _, _ = n, a, b }
`
	_, groupedDiagnostics := analyze(t, grouped)
	_, splitDiagnostics := analyze(t, split)
	assert.Len(t, groupedDiagnostics, 2, "the grouped field declares two offending parameters")
	assert.Len(t, splitDiagnostics, len(groupedDiagnostics), "the positional lists are identical")
}

// TestAGroupedFieldIsReportedAtEachName keeps the two findings above
// distinguishable. Both drivers collapse duplicate diagnostics at one position —
// go/analysis on {pos, end, analyzer, message} and go-yze on {rule, path,
// message, line, col} — so two findings reported at the field's TYPE arrive as
// one, and the count would be honest in the analyzer and wrong everywhere else.
func TestAGroupedFieldIsReportedAtEachName(t *testing.T) {
	const src source = `package p

import "context"

func f(n int, a, b context.Context) { _, _, _ = n, a, b }
`
	fset, diagnostics := analyze(t, src)
	require.Len(t, diagnostics, 2)
	assert.Equal(t, 15, fset.Position(diagnostics[0].Pos).Column, "a begins at column 15")
	assert.Equal(t, 18, fset.Position(diagnostics[1].Pos).Column, "b begins at column 18")
}

// TestVariadicMessageNamesARemedyThatSurvivesARealContext is the shape the first
// wording of variadicMessage got wrong, and the reason it was wrong is worth
// keeping: a []context.Context is NOT a context by this rule's own contract, so
// putting it FIRST makes it the non-context that every following context
// violates. The message said "a leading []context.Context", and an author who
// took that instruction verbatim on a signature that also carried a real
// context traded one finding for another — the same manufactured-disablement
// shape the variadic fix exists to remove. The remedy that always works is to
// drop the ellipsis and leave the slice where it was.
func TestVariadicMessageNamesARemedyThatSurvivesARealContext(t *testing.T) {
	const dropped source = `package p

import "context"

func f(ctx context.Context, n int, ctxs []context.Context) { _, _, _ = ctx, n, ctxs }
`
	const leading source = `package p

import "context"

func f(ctxs []context.Context, ctx context.Context, n int) { _, _, _ = ctxs, ctx, n }
`
	_, droppedDiagnostics := analyze(t, dropped)
	assert.Empty(t, droppedDiagnostics,
		"dropping the ellipsis leaves the slice among the non-contexts, where a slice belongs")

	_, leadingDiagnostics := analyze(t, leading)
	assert.Len(t, leadingDiagnostics, 1,
		"a LEADING slice is the non-context every following context violates, "+
			"which is why the message must not name it")
	assert.NotContains(t, variadicMessage, "leading []context.Context",
		"the message must not prescribe the shape the assertion above proves is reported")
}
