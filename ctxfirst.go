// Package ctxfirst provides a go/analysis analyzer enforcing the gomatic Go
// idiom that context.Context parameters form a contiguous leading prefix of the
// positional parameter list: a context.Context parameter must not follow a
// non-context parameter.
//
// Parameters are judged by position, not by field spelling — (a, b
// context.Context) and (a context.Context, b context.Context) are identical —
// and EVERY offending parameter is reported, each at its own name, so a
// signature burying two contexts behind two different non-contexts draws two
// findings and fixing one of them does not silence the other. A field declaring
// two names declares two parameters and draws two findings; reporting once per
// FIELD made the count a property of the spelling, which is the thing the first
// sentence denies, and it also weakened the promise above, because in the
// grouped spelling fixing one of the two names was answered with the same
// single finding. The report sits at the parameter's own name rather than at
// its field's type because both drivers collapse diagnostics that share a
// position, so a count honest inside the analyzer would arrive halved.
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
// One spelling still escapes that claim, and it is declared rather than left
// to be discovered. Every signature judged here is one the source WRITES, so a
// generic signature is judged at its declaration, where `Do(n Count, ctx T)` is
// correctly not a context — and the INSTANTIATION, which has no source of its
// own, is judged nowhere. `type Forged = inner[context.Context]` is the same
// type as the honest interface, satisfied by the same implementations, called
// through byte-identical call sites, and silent. Closing it means judging
// instantiated types rather than written ones, which is a second walk with its
// own false-positive surface, so it is recorded as
// ctxfirst.a-type-parameter-instantiated-with-a-context-is-judged rather than
// patched in the commit that found it. Until then this rule reaches every
// spelling above and not that one.
//
// Each of the four signatures is matched by TYPE IDENTITY and never up to
// Underlying(), so the answer is the compiler's answer: `Done() <-chan Unit`
// with `type Unit struct{}` is a different method from `Done() <-chan
// struct{}`, and go build says so. Three positions used to unwrap to
// Underlying(), which reported three shapes no compiler will accept as a
// context and prescribed a reorder their author did not owe — a false positive
// answered with a baseline, which is the population this suite exists to
// remove. An ALIAS still answers yes, because an alias IS the type it aliases:
// `type Nothing = struct{}` and `Done() <-chan Nothing` is a context.
//
// # A variadic context is reported, with the remedy that exists for it
//
// `func f(n Count, ctxs ...context.Context)` is reported. It was exempt for one
// revision on the ground that the Go spec permits ... only on the final
// parameter, so the position is forced and no reordering remedy exists. The
// first half is true and the conclusion does not follow: the position is forced
// only once the parameter is variadic, and being variadic is the author's own
// choice. Measured, the exemption inverted the rule's economics — the call site
// f(3, ctx) is byte-identical and still compiles when a declaration is changed
// from `ctx context.Context` to `ctxs ...context.Context`, while the honest
// reorder breaks every call site in the fleet, so evading cost one token and
// nothing else while complying cost an API change. It also deleted a
// compile-time guarantee: a variadic context is an OPTIONAL context, so f(3)
// compiles and panics.
//
// Forcedness is not an exemption criterion anywhere else in this rule either —
// the section below reports a method whose parameter order an interface
// dictates, where reordering genuinely does not compile — so it cannot be one
// here, and no test separates a variadic an author was forced into from one
// typed to buy silence. So the diagnostic carries its own message, naming a
// remedy the compiler accepts because the ordinary one's is not one: "a
// variadic context.Context parameter must not follow a non-context parameter:
// drop the ellipsis for a []context.Context, or lead with ctx context.Context".
//
// The remedy named first is the one that always works, because a
// []context.Context is not a context by this rule's own contract and so belongs
// among the non-contexts — which is where the variadic already is. An earlier
// wording said "a leading []context.Context" and was wrong: leading, the slice
// is the non-context that every context after it violates, so an author whose
// signature also carried a real context traded one finding for another.
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
//  1. A pointer to a context (*context.Context). A pointer to an interface has
//     an empty method set and carries no context; it is its own defect and not
//     this rule's.
//  2. A type carrying the four METHOD NAMES with any other signature — a
//     Deadline returning something that is not time.Time, a send-only Done, an
//     Err returning a package's own error type, a Value taking a concrete key.
//     None of them can stand where a context is wanted, so none is judged.
//  3. A type carrying the four method names over DEFINED types with the right
//     underlying types — Done() <-chan Unit, Deadline() (time.Time, Flag),
//     Value(Anything) Anything. The compiler refuses every one of them as a
//     context.Context, so this rule does too. `[]context.Context` is the same
//     answer reached the same way, and it is the remedy the variadic
//     diagnostic prescribes.
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

// variadicMessage states the same rule for a variadic context and names the
// remedy that exists for it, because the ordinary message's remedy does not: the
// Go spec permits ... only on the final parameter, so `func(ctxs
// ...context.Context, n Count)` is not a program.
//
// The remedy named FIRST is the one that always works, and the reason is the
// rule itself: a []context.Context is not a context, so it belongs among the
// non-contexts, exactly where the variadic already sits. Dropping the ellipsis
// is therefore silent wherever the parameter is, and it is stronger than the
// shape it replaces — a variadic context is an OPTIONAL one, so `Handle(3)`
// compiles and panics, while a slice parameter cannot be omitted.
//
// The first wording of this message said "a leading []context.Context" and was
// wrong for the same reason: leading, the slice becomes the non-context that
// every context after it violates, so an author whose signature also carried a
// real context traded one finding for another. That is the manufactured
// disablement this rule's own repair exists to remove, reintroduced by its
// diagnostic — found by an adversarial pass and pinned by
// TestVariadicMessageNamesARemedyThatSurvivesARealContext.
// TestMessageStatesTheRuleTheDocDeclares pins this to the package comment too.
const variadicMessage = "a variadic context.Context parameter must not follow a non-context parameter: " +
	"drop the ellipsis for a []context.Context, or lead with ctx context.Context"

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

// checkParams reports EVERY offending PARAMETER that follows a non-context
// parameter, once each. A field declaring several names declares that many
// parameters, so (a, b context.Context) draws the same count as (a
// context.Context, b context.Context) — reporting once per field made the count
// a property of the spelling, and weakened the promise that fixing one finding
// does not silence another, since in the grouped spelling it did.
func checkParams(pass *analysis.Pass, params *ast.FieldList) {
	sawNonContext := false
	for _, field := range params.List {
		if !isContext(pass, subjectOf(field)) {
			sawNonContext = true
			continue
		}
		if !sawNonContext {
			continue
		}
		reportField(pass, field)
	}
}

// subjectOf is the type expression whose method set decides whether a field
// carries a context. For a variadic field that is the ELEMENT type: `ctxs
// ...context.Context` has type []context.Context, whose method set is empty, so
// reading the field's own type answered no to every variadic and made `...` a
// one-token switch an author types to leave the rule behind.
func subjectOf(field *ast.Field) ast.Expr {
	if ellipsis, isVariadic := field.Type.(*ast.Ellipsis); isVariadic {
		return ellipsis.Elt
	}
	return field.Type
}

// reportField reports the field once per parameter it declares, at that
// parameter's own name — not at the field's type, where two findings on one
// field would arrive as one. go/analysis collapses diagnostics sharing a
// position, analyzer and message, and go-yze's driver collapses them again on
// {rule, path, message, line, column}, so a count honest inside the analyzer
// would be wrong in every instrument that reads it. An unnamed parameter has no
// name to point at and is reported at its type.
func reportField(pass *analysis.Pass, field *ast.Field) {
	text := message
	if _, isVariadic := field.Type.(*ast.Ellipsis); isVariadic {
		text = variadicMessage
	}
	if len(field.Names) == 0 {
		pass.Report(analysis.Diagnostic{Pos: field.Type.Pos(), Message: text})
		return
	}
	for _, name := range field.Names {
		pass.Report(analysis.Diagnostic{Pos: name.Pos(), Message: text})
	}
}

// isContext reports whether expr's type carries context.Context's contract.
func isContext(pass *analysis.Pass, expr ast.Expr) bool {
	return carriesContext(pass.TypesInfo.TypeOf(expr))
}
