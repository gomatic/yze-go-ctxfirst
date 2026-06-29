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
	URL:        "https://docs.gomatic.dev/yze/go/ctxfirst",
	Analyzer:   Analyzer,
}

// run reports each context.Context parameter that is not the first parameter.
func run(pass *analysis.Pass) (any, error) {
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	insp.Preorder([]ast.Node{(*ast.FuncDecl)(nil)}, func(n ast.Node) {
		checkParams(pass, n.(*ast.FuncDecl).Type.Params)
	})
	return nil, nil
}

// checkParams reports a context.Context field that does not start at position 0.
func checkParams(pass *analysis.Pass, params *ast.FieldList) {
	index := 0
	for _, field := range params.List {
		if index > 0 && isContext(pass, field.Type) {
			pass.Reportf(field.Type.Pos(), message)
		}
		index += positionsOf(field)
	}
}

// positionsOf returns the number of parameter positions a field occupies.
func positionsOf(field *ast.Field) int {
	if len(field.Names) == 0 {
		return 1
	}
	return len(field.Names)
}

// isContext reports whether expr names context.Context.
func isContext(pass *analysis.Pass, expr ast.Expr) bool {
	named, ok := pass.TypesInfo.TypeOf(expr).(*types.Named)
	if !ok || named.Obj().Pkg() == nil {
		return false
	}
	return named.Obj().Pkg().Path() == "context" && named.Obj().Name() == "Context"
}
