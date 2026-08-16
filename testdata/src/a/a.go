package a

import (
	"context"
	"time"
)

// good has context.Context first.
func good(ctx context.Context, n int) { _ = ctx; _ = n }

// bad has context.Context not first.
func bad(n int, ctx context.Context) { _ = n; _ = ctx } // want "a context.Context parameter must not follow a non-context parameter"

// noCtx has no context parameter.
func noCtx(n int) { _ = n }

// noParams has no parameters.
func noParams() {}

// twoContexts leads with two contexts spelled as one field; a contiguous
// leading prefix of contexts is acceptable.
func twoContexts(a, b context.Context) { _ = a; _ = b }

// twoContextsSplit spells the same two leading contexts as separate fields;
// the positional parameter list is identical to twoContexts, so the verdict
// must be too.
func twoContextsSplit(a context.Context, b context.Context) { _ = a; _ = b }

// interleaved leads with a context, but a later context follows a non-context
// parameter, breaking the leading prefix. A context.Context already IS the
// first parameter here, which is why the message states the prefix rule rather
// than asking b to become a position that is taken.
func interleaved(a context.Context, n int, b context.Context) { // want "a context.Context parameter must not follow a non-context parameter"
	_, _, _ = a, n, b
}

// trailingGrouped declares two trailing contexts in ONE field after a
// non-context parameter. A field declaring two names declares two parameters,
// so it draws two findings — the same count as trailingSplit below, whose
// positional list is identical. Reporting once per field made the count a
// property of the spelling, which the package comment denies twice.
func trailingGrouped(n int, a, b context.Context) { // want "a context.Context parameter must not follow a non-context parameter" "a context.Context parameter must not follow a non-context parameter"
	_, _, _ = n, a, b
}

// trailingSplit spells trailingGrouped's positional list as separate fields.
// The two are paired here in the VIOLATING direction, which is what the corpus
// and this file only ever did in the conforming direction, where both are
// silent and 0 == 0 discriminates nothing.
func trailingSplit(n int, a context.Context, b context.Context) { // want "a context.Context parameter must not follow a non-context parameter" "a context.Context parameter must not follow a non-context parameter"
	_, _, _ = n, a, b
}

// twoBuried buries two contexts behind two DIFFERENT non-context parameters, so
// the two offences are separate fields and both must be reported. Every other
// fixture in this repo and every corpus case holds the count at one, which let a
// `return` after the report halve the analyzer's output while the whole suite,
// the corpus and the coverage gate stayed green — and an author who fixed the
// first finding would have been told the signature was clean.
func twoBuried(
	n int,
	first context.Context, // want "a context.Context parameter must not follow a non-context parameter"
	s string,
	second context.Context, // want "a context.Context parameter must not follow a non-context parameter"
) {
	_, _, _, _ = n, first, s, second
}

// T carries a method.
type T struct{}

// method has context.Context not first.
func (T) method(n int, ctx context.Context) { _ = n; _ = ctx } // want "a context.Context parameter must not follow a non-context parameter"

// methodGood has context.Context first.
func (T) methodGood(ctx context.Context, n int) { _ = ctx; _ = n }

// unnamed has unnamed parameters with context not first.
func unnamed(int, context.Context) {} // want "a context.Context parameter must not follow a non-context parameter"

// Iface is an interface whose method signatures are subject to the convention.
type Iface interface {
	// Bad has context.Context not first.
	Bad(n int, ctx context.Context) // want "a context.Context parameter must not follow a non-context parameter"
	// Good has context.Context first.
	Good(ctx context.Context, n int)
}

// closures exercises function literals, which carry their own signatures.
func closures() {
	bad := func(n int, ctx context.Context) { _ = n; _ = ctx } // want "a context.Context parameter must not follow a non-context parameter"
	good := func(ctx context.Context, n int) { _ = ctx; _ = n }
	bad(0, context.Background())
	good(context.Background(), 0)
}

// FuncField is a function-typed signature in a type definition.
type FuncField func(n int, ctx context.Context) // want "a context.Context parameter must not follow a non-context parameter"

// Ctx aliases context.Context. Since Go 1.23 an aliased type resolves to
// *types.Alias, so a spelling test would have to unalias to still recognize it.
type Ctx = context.Context

// aliasBad takes the aliased context not first.
func aliasBad(n int, ctx Ctx) { _ = n; _ = ctx } // want "a context.Context parameter must not follow a non-context parameter"

// aliasGood leads with the aliased context.
func aliasGood(ctx Ctx, n int) { _ = ctx; _ = n }

// Defined is a type written over context.Context. It is mutually assignable
// with context.Context — consumeDefined passes one straight to useContext with
// no conversion — so it is a context and the rule applies to it.
type Defined context.Context

// definedBad takes a defined context not first.
func definedBad(n int, ctx Defined) { _ = n; _ = ctx } // want "a context.Context parameter must not follow a non-context parameter"

// definedGood leads with the defined context.
func definedGood(ctx Defined, n int) { _ = ctx; _ = n }

// Embedder is an interface embedding context.Context, so every value of it is
// a context.
type Embedder interface{ context.Context }

// embedderBad takes an embedding interface not first.
func embedderBad(n int, ctx Embedder) { _ = n; _ = ctx } // want "a context.Context parameter must not follow a non-context parameter"

// embedderGood leads with the embedding interface.
func embedderGood(ctx Embedder, n int) { _ = ctx; _ = n }

// paramBad takes a type parameter constrained to context.Context not first.
func paramBad[C context.Context](n int, ctx C) { _ = n; _ = ctx } // want "a context.Context parameter must not follow a non-context parameter"

// paramGood leads with the constrained type parameter.
func paramGood[C context.Context](ctx C, n int) { _ = ctx; _ = n }

// useContext proves the shapes above really are contexts: each is passed to it
// with no conversion, so a rule that reported only the context.Context spelling
// was silent on values it had already agreed were contexts.
func useContext(ctx context.Context) { _ = ctx }

func consumeDefined(ctx Defined)                   { useContext(ctx) }
func consumeEmbedder(ctx Embedder)                 { useContext(ctx) }
func produceEmbedder(ctx context.Context) Embedder { return ctx }
func consumeParam[C context.Context](ctx C)        { useContext(ctx) }

// Lookalike spells context.Context's four method names and is NOT a context:
// its Deadline returns a bool pair rather than (time.Time, bool), so no value
// of it can be used where a context is wanted. It sits at the boundary of the
// method-set test and must stay silent.
type Lookalike interface {
	Deadline() (deadline bool, ok bool)
	Done() <-chan struct{}
	Err() error
	Value(key any) any
}

// lookalikeSilent holds a Lookalike after a non-context parameter and reports
// nothing, because a Lookalike does not implement context.Context.
func lookalikeSilent(n int, ctx Lookalike) { _ = n; _ = ctx }

// Unit is an empty struct under another name. It is a DEFINED type, so it is
// not struct{} — `var _ context.Context = DefinedElem(nil)` does not compile.
type Unit struct{}

// DefinedElem deviates from context.Context in exactly one position, by naming
// a defined type whose UNDERLYING type is the right one. A contract compared up
// to Underlying() answers yes here and the compiler answers no.
type DefinedElem interface {
	Deadline() (time.Time, bool)
	Done() <-chan Unit
	Err() error
	Value(key any) any
}

// definedElemSilent holds it after a non-context parameter and reports nothing,
// because no value of it can stand where a context is wanted.
func definedElemSilent(n int, c DefinedElem) { _ = n; _ = c }

// Nothing is an ALIAS of the empty struct, which IS the empty struct.
type Nothing = struct{}

// AliasedElem is the other side of that boundary: the compiler accepts it as a
// context.Context, so a fix narrowing to the spelling `struct{}` goes silent on
// a genuine context here.
type AliasedElem interface {
	Deadline() (time.Time, bool)
	Done() <-chan Nothing
	Err() error
	Value(key any) any
}

// aliasedElemBad holds the aliased-element context after a non-context.
func aliasedElemBad(n int, c AliasedElem) { _ = n; _ = c } // want "a context.Context parameter must not follow a non-context parameter"

// Implementor carries context.Context's whole method set on a concrete type, so
// it IS a context however it is spelled.
type Implementor struct{}

func (Implementor) Deadline() (time.Time, bool) { return time.Time{}, false }
func (Implementor) Done() <-chan struct{}       { return nil }
func (Implementor) Err() error                  { return nil }
func (Implementor) Value(key any) any           { _ = key; return nil }

// implementorBad takes a hand-written context implementation not first.
func implementorBad(n int, ctx Implementor) { _ = n; _ = ctx } // want "a context.Context parameter must not follow a non-context parameter"

// variadicBad takes a variadic context that is not first and IS reported, with
// the message that names a remedy the compiler accepts. Exempting this shape
// made the rule switchable off by one token at no cost to anybody: the call
// site `variadicBad(3, ctx)` is byte-identical under the non-variadic spelling
// and compiles under both, while the honest reorder breaks every call site. The
// exemption's reason was that ... is forced to the final position, but the
// author who writes ... chose that position freely — and the genuinely forced
// case, a method whose order an interface dictates, is reported here too.
func variadicBad(n int, ctxs ...context.Context) { // want "a variadic context.Context parameter must not follow a non-context parameter: drop the ellipsis for a \\[\\]context.Context, or lead with ctx context.Context"
	_, _ = n, ctxs
}

// variadicFirst puts the same variadic first, where it breaks no prefix and is
// silent, so the finding above is about the position and not about the ellipsis.
func variadicFirst(ctxs ...context.Context) { _ = ctxs }

// variadicRemedy is the remedy variadicBad's message prescribes: the ellipsis
// dropped, the slice left where it was. Compiled here so the instruction is
// known to be takeable rather than assumed to be, and stronger than the shape
// it replaces, since a variadic context is an optional one — `variadicBad(3)`
// compiles and panics, and this cannot be called without a slice.
func variadicRemedy(n int, ctxs []context.Context) { _, _ = n, ctxs }

// variadicRemedyBesideAContext is the shape that condemned the FIRST wording of
// that message, which said "a leading []context.Context". A slice is not a
// context, so leading it becomes the non-context every following context
// violates: the author who took the old instruction verbatim here traded one
// finding for another. Dropping the ellipsis leaves the slice among the
// non-contexts, where it belongs, and is silent.
func variadicRemedyBesideAContext(ctx context.Context, n int, ctxs []context.Context) {
	_, _, _ = ctx, n, ctxs
}

// variadicRemedyLeadingIsReported is that old instruction taken literally, and
// it is a finding. The pair is the case: a remedy is only proven by the shape
// where it can fail.
func variadicRemedyLeadingIsReported(ctxs []context.Context, ctx context.Context, n int) { // want "a context.Context parameter must not follow a non-context parameter"
	_, _, _ = ctxs, ctx, n
}

// pointerSilent takes a pointer to a context after a non-context parameter. A
// pointer to an interface has an empty method set, so it is not a context and
// is not this rule's finding.
func pointerSilent(n int, ctx *context.Context) { _ = n; _ = ctx }

// universeNamed holds a universe-scope named type after a non-context
// parameter. error has no package, so a rule resolving a type's package to
// compare its path dereferences nil here and dies; the method-set test never
// asks. Silent, and its sibling below pins that the silence is about error and
// not about position.
func universeNamed(n int, err error) { _ = n; _ = err }

// universeNamedFirst is universeNamed with the same types in the other order,
// so a rule that had simply stopped reporting would leave both silent for the
// same wrong reason.
func universeNamedFirst(err error, n int) { _ = err; _ = n }

// cancelFuncSilent holds another named type FROM package context after a
// non-context parameter. context.CancelFunc is not a context, so a rule keyed
// on the package path alone reports it and this case says it must not.
func cancelFuncSilent(n int, cancel context.CancelFunc) { _ = n; _ = cancel }

// Context is a locally-declared type named Context, the shape gin.Context and
// cli.Context take in the wild.
type Context struct{ deadline time.Time }

// localContextSilent holds it after a non-context parameter. It carries none of
// context.Context's methods, so a rule keyed on the type name alone reports it
// and this case says it must not.
func localContextSilent(n int, c Context) { _ = n; _ = c }
