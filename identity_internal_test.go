package ctxfirst

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestATypeTheCompilerRefusesIsNotAContext pins the contract to type IDENTITY
// rather than to Underlying(). Each interface below carries all four contract
// names and deviates from context.Context in exactly ONE position, by naming a
// DEFINED type whose underlying type is the right one — and `go build` refuses
// every one of them as a context.Context ("have Done() <-chan Unit, want Done()
// <-chan struct{}"). A rule comparing up to Underlying() answers yes to a type
// the compiler answers no to, and then prints a reorder its own subject does not
// owe. `control` is the only context here.
func TestATypeTheCompilerRefusesIsNotAContext(t *testing.T) {
	const src source = `package p

import "time"

type Unit struct{}

type Signal <-chan struct{}

type Flag bool

type Anything interface{}

type doneElementIsDefined interface {
	Deadline() (time.Time, bool)
	Done() <-chan Unit
	Err() error
	Value(key any) any
}

type doneResultIsDefined interface {
	Deadline() (time.Time, bool)
	Done() Signal
	Err() error
	Value(key any) any
}

type deadlineSecondIsDefined interface {
	Deadline() (time.Time, Flag)
	Done() <-chan struct{}
	Err() error
	Value(key any) any
}

type valueKeyIsDefined interface {
	Deadline() (time.Time, bool)
	Done() <-chan struct{}
	Err() error
	Value(key Anything) any
}

type valueResultIsDefined interface {
	Deadline() (time.Time, bool)
	Done() <-chan struct{}
	Err() error
	Value(key any) Anything
}

type control interface {
	Deadline() (time.Time, bool)
	Done() <-chan struct{}
	Err() error
	Value(key any) any
}

func a(n int, x doneElementIsDefined)    { _, _ = n, x }
func b(n int, x doneResultIsDefined)     { _, _ = n, x }
func c(n int, x deadlineSecondIsDefined) { _, _ = n, x }
func d(n int, x valueKeyIsDefined)       { _, _ = n, x }
func e(n int, x valueResultIsDefined)    { _, _ = n, x }
func z(n int, x control)                 { _, _ = n, x }
`
	fset, diagnostics := analyze(t, src)
	require.Len(t, diagnostics, 1, "only control implements context.Context; the compiler refuses the other five")
	assert.Equal(t, lineOf(t, src, "func z(n int, x control)"), reportedLine(fset, diagnostics[0]))
}

// TestAnAliasOfAContractTypeIsStillTheContractType is the other side of that
// boundary, and it is why the fix above is identity rather than a name test. An
// ALIAS is the type it aliases — the compiler accepts each interface below where
// a context.Context is wanted — so all four must be reported. A fix that
// narrowed to "the type is spelled struct{} / bool / any" goes silent here.
func TestAnAliasOfAContractTypeIsStillTheContractType(t *testing.T) {
	const src source = `package p

import "time"

type Nothing = struct{}

type Wire = <-chan struct{}

type Truth = bool

type Whatever = any

type doneElementIsAliased interface {
	Deadline() (time.Time, bool)
	Done() <-chan Nothing
	Err() error
	Value(key any) any
}

type doneResultIsAliased interface {
	Deadline() (time.Time, bool)
	Done() Wire
	Err() error
	Value(key any) any
}

type deadlineSecondIsAliased interface {
	Deadline() (time.Time, Truth)
	Done() <-chan struct{}
	Err() error
	Value(key any) any
}

type valueIsAliased interface {
	Deadline() (time.Time, bool)
	Done() <-chan struct{}
	Err() error
	Value(key Whatever) Whatever
}

func a(n int, x doneElementIsAliased)    { _, _ = n, x }
func b(n int, x doneResultIsAliased)     { _, _ = n, x }
func c(n int, x deadlineSecondIsAliased) { _, _ = n, x }
func d(n int, x valueIsAliased)          { _, _ = n, x }
`
	_, diagnostics := analyze(t, src)
	assert.Len(t, diagnostics, 4, "an alias is the type it aliases, so all four are contexts")
}
