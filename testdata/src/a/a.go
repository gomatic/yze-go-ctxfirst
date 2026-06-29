package a

import "context"

// good has context.Context first.
func good(ctx context.Context, n int) { _ = ctx; _ = n }

// bad has context.Context not first.
func bad(n int, ctx context.Context) { _ = n; _ = ctx } // want "context.Context must be the first parameter"

// noCtx has no context parameter.
func noCtx(n int) { _ = n }

// noParams has no parameters.
func noParams() {}

// twoContexts leads with a context, so the trailing context is acceptable.
func twoContexts(a, b context.Context) { _ = a; _ = b }

// T carries a method.
type T struct{}

// method has context.Context not first.
func (T) method(n int, ctx context.Context) { _ = n; _ = ctx } // want "context.Context must be the first parameter"

// methodGood has context.Context first.
func (T) methodGood(ctx context.Context, n int) { _ = ctx; _ = n }

// unnamed has unnamed parameters with context not first.
func unnamed(int, context.Context) {} // want "context.Context must be the first parameter"

// Iface is an interface whose method signatures are subject to the convention.
type Iface interface {
	// Bad has context.Context not first.
	Bad(n int, ctx context.Context) // want "context.Context must be the first parameter"
	// Good has context.Context first.
	Good(ctx context.Context, n int)
}

// closures exercises function literals, which carry their own signatures.
func closures() {
	bad := func(n int, ctx context.Context) { _ = n; _ = ctx } // want "context.Context must be the first parameter"
	good := func(ctx context.Context, n int) { _ = ctx; _ = n }
	bad(0, context.Background())
	good(context.Background(), 0)
}

// FuncField is a function-typed signature in a type definition.
type FuncField func(n int, ctx context.Context) // want "context.Context must be the first parameter"

// Ctx aliases context.Context. Since Go 1.23 an aliased type resolves to
// *types.Alias, so the rule must unalias to still recognize it.
type Ctx = context.Context

// aliasBad takes the aliased context not first.
func aliasBad(n int, ctx Ctx) { _ = n; _ = ctx } // want "context.Context must be the first parameter"

// aliasGood leads with the aliased context.
func aliasGood(ctx Ctx, n int) { _ = ctx; _ = n }

// variadicBad takes a variadic context that is not first; each element is a
// context.Context, so the non-first position violates the rule.
func variadicBad(n int, ctxs ...context.Context) {} // want "context.Context must be the first parameter"
