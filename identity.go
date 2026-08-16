// Each of context.Context's four signatures, and why every position in them is
// compared by TYPE IDENTITY rather than up to Underlying().
//
// The rule's question is what the COMPILER will do — will this value stand
// where a context.Context is wanted — so the comparison has to be the
// compiler's. A DEFINED type is a new type: `type Unit struct{}` has struct{}
// underneath and is NOT struct{}, so `Done() <-chan Unit` does not implement
// context.Context, and go build says so in as many words ("have Done() <-chan
// Unit, want Done() <-chan struct{}"). Three positions here used to unwrap to
// Underlying() while the other two compared by identity, so the analyzer was
// inconsistent across its own four methods and reported three shapes no
// compiler accepts — a false positive whose prescribed reorder its author does
// not owe, and which leaves a baseline as the only move.
//
// types.Identical unaliases, so an ALIAS still answers yes: `type Nothing =
// struct{}` IS struct{}, and `Done() <-chan Nothing` is a context. That is the
// other side of the same boundary and both sides are cased.

package ctxfirst

import "go/types"

// errorType is the universe error interface. Err's result is compared against it
// with types.Identical, so a package declaring its own type called error — or a
// method returning a wrapper — does not answer the contract.
var errorType = types.Universe.Lookup("error").Type()

// doneResultType is context.Context's Done result, <-chan struct{}.
var doneResultType = types.NewChan(types.RecvOnly, types.NewStruct(nil, nil))

// anyType is the universe any, the type context.Context's Value takes and
// returns.
var anyType = types.Universe.Lookup("any").Type()

// boolType is the universe bool, Deadline's second result.
var boolType = types.Typ[types.Bool]

// isDeadline matches Deadline() (time.Time, bool).
func isDeadline(sig *types.Signature) bool {
	if !shapedAs(sig, 0, 2) {
		return false
	}
	return isTimeTime(sig.Results().At(0).Type()) && types.Identical(sig.Results().At(1).Type(), boolType)
}

// isDone matches Done() <-chan struct{}.
func isDone(sig *types.Signature) bool {
	if !shapedAs(sig, 0, 1) {
		return false
	}
	return types.Identical(sig.Results().At(0).Type(), doneResultType)
}

// isErr matches Err() error.
func isErr(sig *types.Signature) bool {
	if !shapedAs(sig, 0, 1) {
		return false
	}
	return types.Identical(sig.Results().At(0).Type(), errorType)
}

// isValue matches Value(any) any.
func isValue(sig *types.Signature) bool {
	if !shapedAs(sig, 1, 1) {
		return false
	}
	return types.Identical(sig.Params().At(0).Type(), anyType) &&
		types.Identical(sig.Results().At(0).Type(), anyType)
}

// isTimeTime reports whether t is time.Time. It is the one position compared by
// name rather than against a constructed type, because time.Time is the one
// type in the contract with no universe spelling — and the package path is read
// off the TYPE, which names its own package, so the answer does not depend on
// what the analyzed package imported or on which files a build constraint
// selected.
func isTimeTime(t types.Type) bool {
	named, isNamed := types.Unalias(t).(*types.Named)
	if !isNamed || named.Obj().Pkg() == nil {
		return false
	}
	return named.Obj().Pkg().Path() == "time" && named.Obj().Name() == "Time"
}
