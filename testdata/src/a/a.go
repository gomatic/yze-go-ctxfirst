package a

import "context"

// good has context.Context first.
func good(ctx context.Context, n int) { _ = ctx; _ = n }

// bad has context.Context not first.
func bad(n int, ctx context.Context) { _ = n; _ = ctx } // want `first parameter`

// noCtx has no context parameter.
func noCtx(n int) { _ = n }

// T carries a method.
type T struct{}

// method has context.Context not first.
func (T) method(n int, ctx context.Context) { _ = n; _ = ctx } // want `first parameter`

// unnamed has unnamed parameters with context not first.
func unnamed(int, context.Context) {} // want `first parameter`
