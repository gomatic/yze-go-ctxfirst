# yze-ctxfirst

A [`yze`](https://github.com/gomatic/yze) analyzer (category `patterns`) enforcing the gomatic Go idiom that a `context.Context` parameter is always the **first** parameter. The convention is a contract on any signature taking a `context.Context`, so it is checked on function and method declarations, interface method signatures, function literals, and function-typed definitions alike.

- **Rule:** `yze/ctxfirst`
- **Library:** exports `Analyzer` and `Registration` for the [`yze`](https://github.com/gomatic/yze) aggregator and [`stickler`](https://github.com/gomatic/stickler) runner.
- **Binary:** `cmd/yze-ctxfirst` runs it standalone (`text`/`-json`, and as a `go vet -vettool`).

Built on the [`go-yze`](https://github.com/gomatic/go-yze) framework.
