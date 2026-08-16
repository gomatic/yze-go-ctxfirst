// The contract this file matches, and why it is matched by shape.
//
// context.Context is an interface, and a parameter carries a context when its
// method set holds all four of context.Context's methods with context.Context's
// own signatures. Nothing here resolves context.Context from the analyzed
// package's imports: the one type the contract needs to name, time.Time, names
// its own package, so the verdict is a property of the source and cannot turn
// on which files a build constraint selected or on what the package happened to
// import.

package ctxfirst

import "go/types"

// paramCount is how many parameters a signature takes.
type paramCount int

// resultCount is how many results a signature returns.
type resultCount int

// errorType is the universe error interface. Err's result is compared against it
// with types.Identical, so a package declaring its own type called error — or a
// method returning a wrapper — does not answer the contract.
var errorType = types.Universe.Lookup("error").Type()

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

// isDeadline matches Deadline() (time.Time, bool).
func isDeadline(sig *types.Signature) bool {
	if !shapedAs(sig, 0, 2) {
		return false
	}
	return isTimeTime(sig.Results().At(0).Type()) && isBasicKind(sig.Results().At(1).Type(), types.Bool)
}

// isDone matches Done() <-chan struct{}.
func isDone(sig *types.Signature) bool {
	if !shapedAs(sig, 0, 1) {
		return false
	}
	channel, isChannel := sig.Results().At(0).Type().Underlying().(*types.Chan)
	return isChannel && channel.Dir() == types.RecvOnly && isEmptyStruct(channel.Elem())
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
	return isEmptyInterface(sig.Params().At(0).Type()) && isEmptyInterface(sig.Results().At(0).Type())
}

// shapedAs reports whether sig takes and returns these counts.
//
// It does NOT test sig.Variadic(), and the omission is measured rather than
// overlooked. Only Value takes a parameter at all, and a variadic
// `Value(key ...any) any` has parameter type []any — a slice, which
// isEmptyInterface already refuses — so a variadic guard here is code no input
// can reach. An earlier revision carried one; deleting it changed no verdict on
// any fixture, corpus case or probe module, which is docs/s03.md's inert guard
// rather than an untested one.
func shapedAs(sig *types.Signature, params paramCount, results resultCount) bool {
	return paramCount(sig.Params().Len()) == params &&
		resultCount(sig.Results().Len()) == results
}

// isTimeTime reports whether t is time.Time. The package path is read off the
// TYPE, which names its own package, so the answer does not depend on what the
// analyzed package imported or on which files a build constraint selected.
func isTimeTime(t types.Type) bool {
	named, isNamed := types.Unalias(t).(*types.Named)
	if !isNamed || named.Obj().Pkg() == nil {
		return false
	}
	return named.Obj().Pkg().Path() == "time" && named.Obj().Name() == "Time"
}

// isEmptyInterface reports whether t is any / interface{}.
func isEmptyInterface(t types.Type) bool {
	iface, isInterface := t.Underlying().(*types.Interface)
	return isInterface && iface.Empty()
}

// isEmptyStruct reports whether t is struct{}.
func isEmptyStruct(t types.Type) bool {
	structure, isStruct := t.Underlying().(*types.Struct)
	return isStruct && structure.NumFields() == 0
}

// isBasicKind reports whether t is the named predeclared kind.
func isBasicKind(t types.Type, kind types.BasicKind) bool {
	basic, isBasic := t.Underlying().(*types.Basic)
	return isBasic && basic.Kind() == kind
}
