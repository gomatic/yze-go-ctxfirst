// Package nocontext reaches package context by no import path, so nothing in it
// can be a context and the analyzer returns before walking a single signature.
// The shapes below are the ones that WOULD be reported in a package that does
// reach context: a violating order, and a hand-written method set spelling
// context.Context's four methods without ever naming it.
package nocontext

// Deadline is spelled here without importing time, so this interface is not
// context.Context's method set and would be silent anywhere. It exists to say
// that a package out of context's reach is not judged at all, rather than
// judged and found clean.
type Carrier interface {
	Done() <-chan struct{}
	Err() error
}

func carrierNotFirst(n int, c Carrier) { _ = n; _ = c }

func carrierFirst(c Carrier, n int) { _ = c; _ = n }
