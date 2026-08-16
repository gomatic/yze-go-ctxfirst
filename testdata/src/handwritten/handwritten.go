// Package handwritten reaches package context by no import path at all, and
// declares a context anyway: Carrier's method set IS context.Context's, so a
// Carrier stands wherever a context.Context is wanted and the rule applies to
// it. An earlier revision resolved the contract by walking this package's
// imports and was silent here — which made the verdict a property of the build
// rather than of the source, and made one import behind a build tag enough to
// turn the rule off for a whole package.
package handwritten

import "time"

// Carrier carries context.Context's whole method set without naming it.
type Carrier interface {
	Deadline() (time.Time, bool)
	Done() <-chan struct{}
	Err() error
	Value(key any) any
}

// carrierSecond takes the hand-written context after a non-context parameter.
func carrierSecond(n int, ctx Carrier) { _ = n; _ = ctx } // want "a context.Context parameter must not follow a non-context parameter"

// carrierFirst leads with it, so the silence below is about position and not
// about the type going unrecognized.
func carrierFirst(ctx Carrier, n int) { _ = ctx; _ = n }

// Partial carries two of the four methods and is not a context.
type Partial interface {
	Done() <-chan struct{}
	Err() error
}

// partialSecond holds it after a non-context parameter and is silent.
func partialSecond(n int, p Partial) { _ = n; _ = p }
