// Package ctxfirst provides a go/analysis analyzer enforcing the gomatic Go idiom
// that context.Context parameters form a contiguous leading prefix of the
// positional parameter list. Parameters are judged by position, not by field
// spelling — (a, b context.Context) and (a context.Context, b context.Context)
// are identical — and any context.Context parameter that follows a non-context
// parameter is reported.
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

// Analyzer reports context.Context parameters that follow a non-context parameter.
var Analyzer = &analysis.Analyzer{
	Name:     "ctxfirst",
	Doc:      "reports context.Context parameters that do not form a leading prefix of the parameter list",
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

// run reports each context.Context parameter that follows a non-context
// parameter. It inspects every function signature — declarations, methods, interface
// methods, function literals, and function-typed definitions — because the
// context-first idiom is a contract on any signature taking a context.Context.
func run(pass *analysis.Pass) (any, error) {
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	insp.Preorder([]ast.Node{(*ast.FuncType)(nil)}, func(n ast.Node) {
		checkParams(pass, n.(*ast.FuncType).Params)
	})
	return nil, nil
}

// checkParams reports each context.Context parameter that follows a
// non-context parameter. Parameters are judged by position, not by field
// index, so (a, b context.Context) and (a context.Context, b context.Context)
// receive the same verdict. A field whose names span several offending
// positions is reported once.
func checkParams(pass *analysis.Pass, params *ast.FieldList) {
	var reported ast.Expr
	sawNonContext := false
	for _, typ := range flattenTypes(params.List) {
		if !isContext(pass, typ) {
			sawNonContext = true
			continue
		}
		if sawNonContext && typ != reported {
			reported = typ
			pass.Reportf(typ.Pos(), message)
		}
	}
}

// flattenTypes expands fields into one type expression per parameter position.
func flattenTypes(fields []*ast.Field) []ast.Expr {
	var positions []ast.Expr
	for _, field := range fields {
		count := len(field.Names)
		if count == 0 {
			count = 1
		}
		for range count {
			positions = append(positions, field.Type)
		}
	}
	return positions
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
