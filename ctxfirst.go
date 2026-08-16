// Package ctxfirst provides a go/analysis analyzer enforcing the gomatic Go
// idiom that context.Context parameters form a contiguous leading prefix of the
// positional parameter list: a context.Context parameter must not follow a
// non-context parameter.
//
// Parameters are judged by position, not by field spelling — (a, b
// context.Context) and (a context.Context, b context.Context) are identical —
// and a field is reported once, at its type.
//
// # A parameter is a context when it IMPLEMENTS context.Context
//
// The rule is about the value the parameter carries, not about how its type is
// spelled, so the test is the method set. context.Context itself, an alias
// (type C = context.Context), a defined type over it (type C context.Context),
// an interface embedding it (interface{ context.Context }), a type parameter
// constrained to it, and any concrete type carrying Deadline, Done, Err and
// Value are all contexts and are all judged.
//
// Matching the spelling instead put the rule one `type` keyword away from
// silence: `type Ctx context.Context` is mutually assignable with
// context.Context, passes to every function taking one, and used to draw
// nothing. It also gave no coverage at all to a codebase that names its own
// context type, which is an ordinary framework habit, and told it nothing.
//
// # What is not a context, and so is not reported
//
//  1. A variadic ...context.Context. The parameter's type is []context.Context
//     — a slice, which carries no context — and the Go spec permits ... only on
//     the final parameter, so its position is forced and no reordering remedy
//     exists. `func f(n int, ctxs ...context.Context)` is silent.
//  2. A pointer to a context (*context.Context). A pointer to an interface has
//     an empty method set and carries no context; it is its own defect and not
//     this rule's.
//  3. Every parameter of a package whose build never reaches package context.
//     The interface is resolved from the analyzed package's transitive imports,
//     so a package that reaches context by no path is silent — including on a
//     type whose method set is written out by hand without ever naming
//     context.Context.
//
// # A signature somebody else owns is reported, and its remedy may not exist
//
// Every function signature is inspected — declarations, methods, interface
// methods, function literals and function-typed definitions — because the
// context-first idiom is a contract on any signature taking a context.Context.
// Nothing here can tell which of those the author is free to reorder, and two
// shapes are reported with no remedy available:
//
//   - a method whose parameter order is fixed by an interface the package
//     implements — reordering it stops satisfying the interface and the package
//     stops compiling;
//   - a function literal whose parameters a third party binds positionally by
//     reflection, such as gomega's Eventually(func(g gomega.Gomega, ctx
//     context.Context)), where reordering compiles and then fails at run time.
//
// Both are declared here rather than exempted, deliberately. An
// interface-satisfaction exemption cannot be written honestly: the set of
// interfaces a method might belong to is open — any package may declare one —
// so the exemption is acquired by writing an interface, which costs one file
// and none of the property the exemption exists for. yze-go-ptrrecv declines
// the same exemption for the mirror-image question, for the same reason. So the
// finding stands, and it is answered by a scoped, recorded disablement rather
// than by a marker anybody can type.
package ctxfirst

import (
	"go/ast"
	"go/types"

	goyze "github.com/gomatic/go-yze"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// message states the rule the analyzer enforces. It is the leading prefix rule
// and not "context.Context must be the first parameter": only one parameter can
// be first, so the older wording asked the third and later contexts of
// `func(a context.Context, n int, b context.Context)` to become a position that
// is already taken. TestMessageStatesTheRuleTheDocDeclares pins it to the
// package comment above so the two cannot drift apart again.
const message = "a context.Context parameter must not follow a non-context parameter"

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

// visited records the packages an import walk has already answered for, so a
// diamond in the import graph is entered once and a cycle cannot be entered at
// all.
type visited map[*types.Package]bool

// run reports each context.Context parameter that follows a non-context
// parameter. It resolves context.Context once per package: a package the build
// never carries to package context can hold no context parameter, and judging
// it costs nothing but a walk of every signature.
func run(pass *analysis.Pass) (any, error) {
	contract := contextInterface(pass.Pkg)
	if contract == nil {
		return nil, nil
	}
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	insp.Preorder([]ast.Node{(*ast.FuncType)(nil)}, func(n ast.Node) {
		checkParams(pass, contract, n.(*ast.FuncType).Params)
	})
	return nil, nil
}

// checkParams reports each context parameter that follows a non-context
// parameter. A field is one position however many names it declares, so
// (a, b context.Context) and (a context.Context, b context.Context) receive the
// same verdict and a field is reported once, at its type.
func checkParams(pass *analysis.Pass, contract *types.Interface, params *ast.FieldList) {
	sawNonContext := false
	for _, field := range params.List {
		if !isContext(pass, contract, field.Type) {
			sawNonContext = true
			continue
		}
		if sawNonContext {
			pass.Reportf(field.Type.Pos(), message)
		}
	}
}

// isContext reports whether expr's type carries context.Context's contract. The
// question is the method set, so a defined type, an embedding interface, a
// constrained type parameter and a hand-written implementation all answer yes,
// while a slice of contexts (the variadic case) and a pointer to one answer no
// because neither carries the methods.
func isContext(pass *analysis.Pass, contract *types.Interface, expr ast.Expr) bool {
	return types.Implements(pass.TypesInfo.TypeOf(expr), contract)
}

// contextInterface returns context.Context's interface as this build declares
// it, or nil when no path out of pkg reaches package context.
func contextInterface(pkg *types.Package) *types.Interface {
	return searchImports(pkg, visited{})
}

// searchImports walks pkg and its transitive imports for package context.
func searchImports(pkg *types.Package, seen visited) *types.Interface {
	if seen[pkg] {
		return nil
	}
	seen[pkg] = true
	if contract := declaredContext(pkg); contract != nil {
		return contract
	}
	for _, imported := range pkg.Imports() {
		if contract := searchImports(imported, seen); contract != nil {
			return contract
		}
	}
	return nil
}

// declaredContext returns pkg's Context interface when pkg is package context,
// and nil for every other package. Both halves of the test are required: the
// import path alone would trust a testdata package named context, and the
// lookup alone would accept any package declaring an interface called Context.
func declaredContext(pkg *types.Package) *types.Interface {
	if pkg.Path() != "context" {
		return nil
	}
	declared := pkg.Scope().Lookup("Context")
	if declared == nil {
		return nil
	}
	contract, _ := declared.Type().Underlying().(*types.Interface)
	return contract
}
