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
