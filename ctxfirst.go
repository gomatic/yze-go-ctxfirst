// Package ctxfirst provides a go/analysis analyzer enforcing the gomatic Go idiom
// that a context.Context parameter is always the first parameter.
package ctxfirst

import (
	"go/ast"
	"go/types"

	goyze "github.com/gomatic/go-yze"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const message = "context.Context must be the first parameter"

// Analyzer reports context.Context parameters that are not first.
var Analyzer = &analysis.Analyzer{
	Name:     "ctxfirst",
	Doc:      "reports context.Context parameters that are not the first parameter",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// Registration declares this analyzer to the yze framework.
var Registration = goyze.Registration{
	Name:       "ctxfirst",
	Categories: []goyze.Category{"patterns"},
	URL:        "https://docs.gomatic.dev/yze/ctxfirst",
	Analyzer:   Analyzer,
}

// run reports each context.Context parameter that is not the first parameter.
// It inspects every function signature — declarations, methods, interface
// methods, function literals, and function-typed definitions — because the
// context-first idiom is a contract on any signature taking a context.Context.
func run(pass *analysis.Pass) (any, error) {
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	insp.Preorder([]ast.Node{(*ast.FuncType)(nil)}, func(n ast.Node) {
		checkParams(pass, n.(*ast.FuncType).Params)
	})
	return nil, nil
}

// checkParams reports a context.Context parameter that is not the first field.
// Only the first field's position matters: a later field is never first
// regardless of how many names earlier fields declare, so a plain field index
// suffices.
func checkParams(pass *analysis.Pass, params *ast.FieldList) {
	for i, field := range params.List {
		if i > 0 && isContext(pass, field.Type) {
			pass.Reportf(field.Type.Pos(), message)
		}
	}
}

// isContext reports whether expr names context.Context. A variadic parameter
// (...context.Context) is unwrapped to its element type, and the resolved type
// is unaliased so a context reached through a type alias (type C = context.Context)
// resolves to *types.Named rather than the *types.Alias that Go 1.23+ produces.
func isContext(pass *analysis.Pass, expr ast.Expr) bool {
	if ellipsis, ok := expr.(*ast.Ellipsis); ok {
		expr = ellipsis.Elt
	}
	named, ok := types.Unalias(pass.TypesInfo.TypeOf(expr)).(*types.Named)
	if !ok || named.Obj().Pkg() == nil {
		return false
	}
	return named.Obj().Pkg().Path() == "context" && named.Obj().Name() == "Context"
}
