package ctxfirst

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTheContractIsMatchedByShapeNotByName walks the boundary of every method
// in context.Context's contract. Each interface below carries all four method
// NAMES and misses the contract in exactly one way, so a rule matching names —
// or matching a method's arity without its types — reports it. `control` is the
// only context in the file and the only finding this test permits, which is
// what says the thirteen silences are the signatures' and not the run's.
func TestTheContractIsMatchedByShapeNotByName(t *testing.T) {
	const src source = `package p

import "time"

type Time struct{}

type deadlineTakesArgs interface {
	Deadline(now time.Time) (time.Time, bool)
	Done() <-chan struct{}
	Err() error
	Value(key any) any
}

type deadlineReturnsOne interface {
	Deadline() bool
	Done() <-chan struct{}
	Err() error
	Value(key any) any
}

type deadlineIsNotTime interface {
	Deadline() (Time, bool)
	Done() <-chan struct{}
	Err() error
	Value(key any) any
}

type deadlineIsUniverse interface {
	Deadline() (error, bool)
	Done() <-chan struct{}
	Err() error
	Value(key any) any
}

type Stamp = time.Time

type deadlineIsAnAliasedTime interface {
	Deadline() (Stamp, bool)
	Done() <-chan struct{}
	Err() error
	Value(key any) any
}

type deadlineIsAnotherTimeType interface {
	Deadline() (time.Month, bool)
	Done() <-chan struct{}
	Err() error
	Value(key any) any
}

type deadlineSecondIsNotBool interface {
	Deadline() (time.Time, int)
	Done() <-chan struct{}
	Err() error
	Value(key any) any
}

type doneTakesAnArgument interface {
	Deadline() (time.Time, bool)
	Done(n int) <-chan struct{}
	Err() error
	Value(key any) any
}

type errReturnsTwo interface {
	Deadline() (time.Time, bool)
	Done() <-chan struct{}
	Err() (error, bool)
	Value(key any) any
}

type doneIsNotAChannel interface {
	Deadline() (time.Time, bool)
	Done() struct{}
	Err() error
	Value(key any) any
}

type doneIsSendOnly interface {
	Deadline() (time.Time, bool)
	Done() chan<- struct{}
	Err() error
	Value(key any) any
}

type doneCarriesSomething interface {
	Deadline() (time.Time, bool)
	Done() <-chan int
	Err() error
	Value(key any) any
}

type errIsNotError interface {
	Deadline() (time.Time, bool)
	Done() <-chan struct{}
	Err() string
	Value(key any) any
}

type valueTakesAConcreteKey interface {
	Deadline() (time.Time, bool)
	Done() <-chan struct{}
	Err() error
	Value(key string) any
}

type valueTakesTwoKeys interface {
	Deadline() (time.Time, bool)
	Done() <-chan struct{}
	Err() error
	Value(key any, fallback any) any
}

type valueIsVariadic interface {
	Deadline() (time.Time, bool)
	Done() <-chan struct{}
	Err() error
	Value(key ...any) any
}

type control interface {
	Deadline() (time.Time, bool)
	Done() <-chan struct{}
	Err() error
	Value(key any) any
}

func a(n int, c deadlineTakesArgs)          { _, _ = n, c }
func b(n int, c deadlineReturnsOne)         { _, _ = n, c }
func c(n int, x deadlineIsNotTime)          { _, _ = n, x }
func d(n int, x deadlineIsUniverse)         { _, _ = n, x }
func e(n int, x deadlineIsAnotherTimeType)  { _, _ = n, x }
func f(n int, x deadlineSecondIsNotBool)    { _, _ = n, x }
func n1(n int, x doneTakesAnArgument)       { _, _ = n, x }
func n2(n int, x errReturnsTwo)             { _, _ = n, x }
func g(n int, x doneIsNotAChannel)          { _, _ = n, x }
func h(n int, x doneIsSendOnly)             { _, _ = n, x }
func i(n int, x doneCarriesSomething)       { _, _ = n, x }
func j(n int, x errIsNotError)              { _, _ = n, x }
func k(n int, x valueTakesAConcreteKey)     { _, _ = n, x }
func n4(n int, x valueTakesTwoKeys)         { _, _ = n, x }
func l(n int, x valueIsVariadic)            { _, _ = n, x }
func n3(n int, x deadlineIsAnAliasedTime)   { _, _ = n, x }
func m(n int, x control)                    { _, _ = n, x }
`
	fset, diagnostics := analyze(t, src)
	require.Len(t, diagnostics, 2, "control and the aliased-time carrier are the only contexts here")
	assert.Equal(t, lineOf(t, src, "func n3(n int, x deadlineIsAnAliasedTime)"), reportedLine(fset, diagnostics[0]),
		"an alias of time.Time IS time.Time, and the unalias is what says so")
	assert.Equal(t, lineOf(t, src, "func m(n int, x control)"), reportedLine(fset, diagnostics[1]),
		"the other finding is control's, so every near miss above was refused on its signature")
}

// lineOf resolves a line number from the fixture rather than from a run, so an
// expectation still says which declaration it is about and only the arithmetic
// is mechanical.
func lineOf(t *testing.T, src source, declaration string) lineNumber {
	t.Helper()
	for number, text := range strings.Split(string(src), "\n") {
		if strings.Contains(text, declaration) {
			return lineNumber(number + 1)
		}
	}
	t.Fatalf("no line of the fixture contains %q", declaration)
	return 0
}

// TestAHandWrittenContextIsJudgedWithoutAnImportOfContext is the case the
// earlier revision could not have: this package reaches package context by no
// path at all, and its interface is a context by every operational test — the
// compiler accepts it where a context.Context is wanted. Resolving the contract
// from the import graph left it silent, which is a rule an author turns off by
// moving one import behind a build tag.
func TestAHandWrittenContextIsJudgedWithoutAnImportOfContext(t *testing.T) {
	const src source = `package p

import "time"

type Carrier interface {
	Deadline() (time.Time, bool)
	Done() <-chan struct{}
	Err() error
	Value(key any) any
}

func f(n int, ctx Carrier) { _, _ = n, ctx }
`
	fset, diagnostics := analyze(t, src)
	require.Len(t, diagnostics, 1)
	assert.Equal(t, 12, fset.Position(diagnostics[0].Pos).Line)
}

// TestAPackageWithoutAnyContextIsSilent pins the other direction, so the test
// above is not satisfied by an implementation that reports every parameter.
func TestAPackageWithoutAnyContextIsSilent(t *testing.T) {
	const src source = `package p

type Carrier interface {
	Done() <-chan struct{}
	Err() error
}

func f(n int, c Carrier) { _, _ = n, c }
`
	_, diagnostics := analyze(t, src)
	assert.Empty(t, diagnostics)
}
