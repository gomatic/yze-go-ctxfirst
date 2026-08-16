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

// TestDeclaredContextNeedsThePathAndTheDeclaration covers both halves of the
// test that resolves the interface, and the first half is a forgery case rather
// than a tidiness one. Only the standard library can hold the import path
// "context"; a package DIRECTORY named context is free, so a package clause and
// an empty `type Context interface{}` inside one would otherwise be accepted as
// the contract — and every type implements an empty interface, so every
// parameter would become a context and the rule would fall silent everywhere
// that package is reachable. Resolving by import path costs the forger the one
// thing they cannot acquire. Reading pkg.Name() instead of pkg.Path() is a
// one-word change that survives every other test in this repo.
func TestDeclaredContextNeedsThePathAndTheDeclaration(t *testing.T) {
	assert.Nil(t, declaredContext(forgedContext(t)), "only the import path \"context\" declares the contract")

	hollow := types.NewPackage("context", "context")
	hollow.MarkComplete()
	assert.Nil(t, declaredContext(hollow), "package context with no Context declaration yields nothing")
}

// forgedContext builds the package a forger can write: the directory name
// "context", a package clause to match, and an empty interface called Context
// that every type in the build satisfies.
func forgedContext(t *testing.T) *types.Package {
	t.Helper()
	pkg := types.NewPackage("example.com/mod/context", "context")
	empty := types.NewInterfaceType(nil, nil)
	empty.Complete()
	name := types.NewTypeName(token.NoPos, pkg, "Context", nil)
	types.NewNamed(name, empty, nil)
	pkg.Scope().Insert(name)
	pkg.MarkComplete()
	require.NotNil(t, pkg.Scope().Lookup("Context"))
	return pkg
}

// TestVisitedStopsSearchImportsOnAnImportCycle drives the visited guard. go/types
// will hold a cyclic import graph even though a Go build never produces one,
// and the walk without the guard does not return: deleting `seen[pkg]` makes
// this test die with a stack overflow rather than fail.
func TestVisitedStopsSearchImportsOnAnImportCycle(t *testing.T) {
	first := types.NewPackage("example.com/first", "first")
	second := types.NewPackage("example.com/second", "second")
	first.SetImports([]*types.Package{second})
	second.SetImports([]*types.Package{first})

	assert.Nil(t, searchImports(first, visited{}))
}

// TestContextInterfaceIsFoundThroughATransitiveImport pins that the search is
// transitive. A package can hold a context parameter while importing context by
// no direct path — the type comes from a dependency — so a search of the direct
// imports alone goes silent on exactly the codebases that name their own
// context type.
func TestContextInterfaceIsFoundThroughATransitiveImport(t *testing.T) {
	const src source = `package p

import "net/http"

var _ = http.DefaultClient
`
	_, pkg := check(t, src)
	assert.NotContains(t, importPaths(pkg), "context", "the fixture must not import context directly")
	assert.NotNil(t, contextInterface(pkg))
}

// importPaths names pkg's direct imports.
func importPaths(pkg *types.Package) []string {
	paths := make([]string, 0, len(pkg.Imports()))
	for _, imported := range pkg.Imports() {
		paths = append(paths, imported.Path())
	}
	return paths
}

// analyze runs the analyzer over src and returns the diagnostics it reported.
func analyze(t *testing.T, src source) (*token.FileSet, []analysis.Diagnostic) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "p.go", string(src), 0)
	require.NoError(t, err)
	info := &types.Info{Types: map[ast.Expr]types.TypeAndValue{}, Defs: map[*ast.Ident]types.Object{}}
	conf := types.Config{Importer: importer.Default()}
	pkg, err := conf.Check("example.test/p", fset, []*ast.File{file}, info)
	require.NoError(t, err)

	var diagnostics []analysis.Diagnostic
	pass := &analysis.Pass{
		Fset:      fset,
		Files:     []*ast.File{file},
		Pkg:       pkg,
		TypesInfo: info,
		ResultOf:  map[*analysis.Analyzer]any{inspect.Analyzer: inspector.New([]*ast.File{file})},
		Report:    func(d analysis.Diagnostic) { diagnostics = append(diagnostics, d) },
	}
	result, err := run(pass)
	require.NoError(t, err)
	require.Nil(t, result)
	return fset, diagnostics
}

// check type-checks src and returns its file and package.
func check(t *testing.T, src source) (*ast.File, *types.Package) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "p.go", string(src), 0)
	require.NoError(t, err)
	conf := types.Config{Importer: importer.Default()}
	pkg, err := conf.Check("example.test/p", fset, []*ast.File{file}, nil)
	require.NoError(t, err)
	require.True(t, strings.HasSuffix(pkg.Path(), "/p"))
	return file, pkg
}
