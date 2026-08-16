// The contract this file matches, and why it is matched by shape.
//
// context.Context is an interface, and a parameter carries a context when its
// method set holds all four of context.Context's methods with context.Context's
// own signatures. Nothing here resolves context.Context from the analyzed
// package's imports: the one type the contract needs to name, time.Time, names
// its own package, so the verdict is a property of the source and cannot turn
// on which files a build constraint selected or on what the package happened to
// import.
//
// This file decides WHICH methods a context has and how a type is measured
// against them. What each method's signature must BE — and how each position in
// it is compared — is identity.go.

package ctxfirst

import "go/types"

// paramCount is how many parameters a signature takes.
type paramCount int

// resultCount is how many results a signature returns.
type resultCount int

// contract is context.Context's method set: each name paired with the shape its
// signature must have. Matching the name alone would take any type carrying a
// Deadline, a Done, an Err and a Value, none of which could stand where a
// context is wanted.
var contract = map[string]func(*types.Signature) bool{
	"Deadline": isDeadline,
	"Done":     isDone,
	"Err":      isErr,
	"Value":    isValue,
}

// carriesContext reports whether t's method set holds all four of
// context.Context's methods, each with context.Context's own signature. A
// slice, a pointer to an interface and a bare primitive have no method set at
// all and answer no without a special case for any of them.
func carriesContext(t types.Type) bool {
	set := types.NewMethodSet(t)
	held := 0
	for i := range set.Len() {
		method := set.At(i).Obj()
		shaped, isContractName := contract[method.Name()]
		if isContractName && shaped(method.Type().(*types.Signature)) {
			held++
		}
	}
	return held == len(contract)
}

// shapedAs reports whether sig takes and returns these counts.
//
// It does NOT test sig.Variadic(), and the omission is measured rather than
// overlooked. Only Value takes a parameter at all, and a variadic
// `Value(key ...any) any` has parameter type []any — a slice, which the
// identity comparison against any already refuses — so a variadic guard here is
// code no input can reach. An earlier revision carried one; deleting it changed
// no verdict on any fixture, corpus case or probe module, which is
// docs/s03.md's inert guard rather than an untested one.
func shapedAs(sig *types.Signature, params paramCount, results resultCount) bool {
	return paramCount(sig.Params().Len()) == params &&
		resultCount(sig.Results().Len()) == results
}
