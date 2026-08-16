// Package ctxfirst provides a go/analysis analyzer enforcing the gomatic Go
// idiom that context.Context parameters form a contiguous leading prefix of the
// positional parameter list: a context.Context parameter must not follow a
// non-context parameter.
//
// Parameters are judged by position, not by field spelling — (a, b
// context.Context) and (a context.Context, b context.Context) are identical —
// and EVERY offending parameter is reported, each at its own field's type, so a
// signature burying two contexts behind two different non-contexts draws two
// findings and fixing one of them does not silence the other.
//
// # A parameter is a context when it CARRIES context.Context's method set
//
// The rule is about the value the parameter carries, not about how its type is
// spelled, so the test is the method set: Deadline() (time.Time, bool), Done()
// <-chan struct{}, Err() error and Value(any) any, each matched by its own
// shape rather than by its name alone. context.Context itself, an alias (type C
// = context.Context), a defined type over it (type C context.Context), an
// interface embedding it, a struct embedding it, a type parameter constrained
// to it, and any type carrying those four methods by hand are all contexts and
// are all judged.
//
// Matching the spelling instead put the rule one `type` keyword away from
// silence: `type Ctx context.Context` is mutually assignable with
// context.Context, passes to every function taking one, and used to draw
// nothing. It also gave no coverage at all to a codebase that names its own
// context type, which is an ordinary framework habit, and told it nothing.
//
// # The verdict is a property of the SOURCE, deliberately
//
// Nothing here consults the analyzed package's imports, and that is a
// requirement rather than an implementation detail. An earlier revision
// resolved context.Context by walking the transitive imports for the package at
// path "context", which made the verdict a property of what the BUILD reached:
// a package whose only route to context sat behind `//go:build linux` was
// judged on linux and silent on darwin, and a hand-written method set in a
// package importing only time was silent everywhere. A verdict an author cannot
// satisfy on two machines has no remedy but a disablement, and a rule that goes
// quiet when a build tag moves is one comment away from being switched off. So
// the package path is read off the TYPE — time.Time names its own package — and
// never off the import graph.
//
// # What is not a context, and so is not reported
//
//  1. A variadic ...context.Context. The parameter's type is []context.Context,
//     a slice, whose method set is empty — and the Go spec permits ... only on
//     the final parameter, so its position is forced and no reordering remedy
//     exists. `func f(n int, ctxs ...context.Context)` is silent.
//  2. A pointer to a context (*context.Context). A pointer to an interface has
//     an empty method set and carries no context; it is its own defect and not
//     this rule's.
//  3. A type carrying the four METHOD NAMES with any other signature — a
//     Deadline returning something that is not time.Time, a send-only Done, an
//     Err returning a package's own error type, a Value taking a concrete key.
//     None of them can stand where a context is wanted, so none is judged.
//
// # A signature somebody else owns is reported, and its remedy may not exist
//
// Every function signature is inspected — declarations, methods, interface
// methods, function literals, function-typed definitions, and every other place
// a signature can be written, including a conversion target, a type-switch
// case, a composite type's element, a generic type argument and a type
// parameter's constraint term. Nothing here can tell which of those the author
// is free to reorder, and several are reported with no remedy available:
//
//   - a method whose parameter order is fixed by an interface the package
//     implements — reordering it stops satisfying the interface and the package
//     stops compiling;
//   - a function literal whose parameters a third party binds positionally by
//     reflection, such as gomega's Eventually(func(g gomega.Gomega, ctx
//     context.Context)), where reordering compiles and then fails at run time;
//   - every USE of a signature the author does not own — the conversion, the
//     type-switch case and the constraint term above — where there is no
//     declaration here to reorder at all.
//
// These are declared rather than exempted. The set of interfaces a method might
// belong to is open — any package may declare one — so an exemption keyed on
// interface satisfaction inside the analyzed module is acquired by writing an
// interface, which costs one file and none of the property the exemption exists
// for. Keying it OUTSIDE the module is the shape yze/ptrparam uses, and it is
// available here: it is not taken yet because it is a clause about module
// identity, which no corpus case can observe (analysistest leaves pass.Module
// nil) and which a `replace` directive weakens. Recorded as
// ctxfirst.foreign-signature-exempted so the choice stays open and deliberate;
// until it is made the finding stands and is answered by a scoped, recorded
// disablement rather than by a marker anybody can type.
package ctxfirst

import (
	"go/ast"

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

// run reports each context.Context parameter that follows a non-context parameter.
func run(pass *analysis.Pass) (any, error) {
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	insp.Preorder([]ast.Node{(*ast.FuncType)(nil)}, func(n ast.Node) {
		checkParams(pass, n.(*ast.FuncType).Params)
	})
	return nil, nil
}

// checkParams reports EVERY context parameter that follows a non-context
// parameter, each at its own field's type. A field is one position however many
// names it declares, so (a, b context.Context) and (a context.Context, b
// context.Context) receive the same verdict and a field is reported once.
func checkParams(pass *analysis.Pass, params *ast.FieldList) {
	sawNonContext := false
	for _, field := range params.List {
		if !isContext(pass, field.Type) {
			sawNonContext = true
			continue
		}
		if sawNonContext {
			pass.Reportf(field.Type.Pos(), message)
		}
	}
}

// isContext reports whether expr's type carries context.Context's contract.
func isContext(pass *analysis.Pass, expr ast.Expr) bool {
	return carriesContext(pass.TypesInfo.TypeOf(expr))
}
