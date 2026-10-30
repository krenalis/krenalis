# Krenalis

Go monorepo with dependencies vendored in `vendor/`. Every change must follow these conventions.

# Changes

Keep the diff minimal: touch only the lines the change needs, with no drive-by reformatting, renaming, reordering, or comment rewrapping. Between equivalent fixes, pick the one that leaves more surrounding lines untouched. Reread your diff before reporting it: every hunk must be justifiable.

# Review process

Judge a change against the code as it was before the change; code introduced or altered by the change is not precedent.

Apply review rules only to code the change introduces or modifies. Do not retrofit a new convention onto untouched code unless the user explicitly expands the scope or the change cannot be done correctly without it.

# Go conventions

## Line length

Lines may be up to 120 characters, counting a tab as 4. Longer lines are allowed when a more specific rule requires them, such as one-line signatures, and in the case tables of table-driven tests; the rest of a test file follows the limit.

## File organization

Place exported type groups before unexported ones, completing each exported type, with all its methods, before any unexported type. Order each type group as:

1. The type.
2. Its constructors.
3. Its exported methods, alphabetically.
4. Its unexported methods, alphabetically.

A type without methods that appears in the declaration of exactly one other defined type may follow that type directly, before its constructors; several such types follow the order of first use.

Package-level functions follow the type groups, alphabetically.

Place broadly relevant package-level constants and variables at the top of the file; place one specific to a single function, or of secondary importance, immediately before the function that uses it.

Place a new `init` function right after the imports. Do not flag or move an existing `init`, even in a change under review: its placement may be deliberate.

## Layout and naming

An exported function or method may call another exported function or method of the same package only when it is solely a simpler wrapper for a common case; otherwise both call a shared unexported function or method.

Declare variables as close to their use as readability allows.

Mark a mutex-guarded field with `// protected by mu`.

Keep the receiver name the type already uses; the `core/` facade types (`Connection`, `Connector`, `Organization`, `Pipeline`, `Workspace`) use `this`.

## Imports

In `cmd`, `core`, and their subpackages, always import `github.com/krenalis/krenalis/tools/errors`, which exposes every standard `errors` name, instead of the standard `errors`. Elsewhere, prefer the standard `errors` and import `tools/errors` only for its repository-specific functionality; `tools/errors` itself and its compatibility tests use the standard package to avoid an import cycle.

Use an import's default name unless an actual identifier conflict requires disambiguation; reusing the name for a variable, field, or selector is not a conflict when the language permits it. When a real conflict exists, weigh both names and keep the one more obvious to readers, rather than renaming the import by default.

## Maps and composite values

Initialize an empty writable map with `map[K]V{}`, or with `make(map[K]V, capacity)` when there is a meaningful capacity hint. Prefer `&T{}` to `new(T)`.

## Switch statements

Every `switch`, including type switches and switches without an expression, has a `default` clause that makes explicit what happens to unmatched values:

- If they cannot occur, treat them as an unexpected situation (see Errors versus panics), naming the value or type in the message.
- If they need no action, leave `default` empty except for a short comment saying why.

Omit `default` only when:

- the switch expression, or type switch guard, is a local variable assigned just above, visibly limited to the cases covered; or
- every case ends in a terminating statement and unmatched values are expected ones that continue after the switch.

## Signatures

Keep every parameter list on one line, whatever its length; a list too long to read is a signal to redesign the signature, not to wrap it. During review, do not change an exported signature without prior agreement; meanwhile, keep it and put its parameters on one line.

## Contexts

Functions whose execution time cannot be bounded in advance take a `context.Context` as first parameter; any exception must be compelling, explicit, and justified. Assume the caller has just checked that the context is still live: do not start the callee with a `ctx.Err()` or `ctx.Done()` check.

## Error handling

Handle errors with a direct `if err != nil { ... }` branch, with the error assigned before it or in its initializer, unless a concrete reason makes that unsuitable.

Never put a call with side effects in the initializer of an error check: write `err := f()` followed by `if err != nil`, not `if err := f(); err != nil`. Exception: `(*sql.Rows).Scan` goes in the initializer, following Go convention; `(*sql.Row).Scan` does not.

Classify errors (`errors.Is`, `errors.AsType`) inside the non-nil branch, never as a substitute for it. A test expecting a specific error checks it inside the `err != nil` branch and fails after the branch if the call returned nil.

Never attach an `else` to an `if err != nil` check; restructure the flow instead.

Prefer `errors.AsType` to `errors.As`.

When handling an error would only return it, possibly with the call's other results, skip the `if err != nil` branch: write `err = f()` then `return err` rather than checking and returning `nil`, and `return f()` when the call is short and directly precedes the return, including calls with several results.

## Errors versus panics

Return an error when a function is expected to fail, because it performs fallible operations or returns errors callers can usefully handle; otherwise the error may be omitted.

An unexpected situation is one that callers or invariants rule out, such as an invalid argument or an impossible value, and reveals a bug. Return an error for it if the signature has one, so one failing operation does not bring down the process; panic if the signature has none, or if internal state is corrupted beyond safe recovery.

## Tests

Failure messages take the form `expected ..., got ...`.

Use `t.Context()` for operations whose lifetime follows the test, passing it directly unless it spans a broad part of the test or derives another context. A helper that takes `*testing.T` obtains the context itself; it accepts a separate context only when callers need one with different values, deadline, cancellation, or lifetime.

Use `testing/synctest` for concurrent or asynchronous code that depends on timers, deadlines, or goroutine quiescence and can run entirely in a bubble, preferring virtual time and `synctest.Wait` to sleeps or polling. Do not use it around real network I/O, system calls, or external processes unless they are replaced by fakes inside the bubble.

To see individual Admin test results without waiting for the full suite, you may run `go run ./test/commit -just-test-admin`.

When a test reads a mapped port of a testcontainers container, its wait strategy must include `wait.ForListeningPort` for that port, or `MappedPort` can fail, notably on Docker Desktop. For PostgreSQL, `postgres.BasicWaitStrategies()` includes it.

## Behavioral contracts

Return a non-nil error whenever a function does not perform what its name, and secondarily its declaration comment, promises; never report success after skipping, failing, or partially completing the operation. An exception must be narrow, compelling, and stated in the name or declaration comment.

## Function extraction

Do not extract code solely to unit-test it. Keep small or operation-specific steps inline; extract a helper only when it is a meaningful production abstraction.

Extract a parser, validator, or converter only when it captures non-trivial, reusable rules inherent to a value or its representation; duplicated code alone is not enough. Prefer focused helpers for individual value concepts that callers compose, rather than one function handling an operation's whole set of parameters or results. First check how comparable cases were handled in the same context before the change: stay inline if they are inline, otherwise match their granularity; if precedent is unclear, ask before introducing the function.

When a parsing or validation sequence is dense and covers several distinct values or groups, precede each group with a short comment naming the operation and subject, such as `// Validate start and end.`, one per group rather than per condition. Add a blank line before it only when it clarifies the boundary. Skip this for single checks, short sequences, or comments that would restate the conditions.

Exported functions validate their own arguments. Exception: exported methods in `core/internal/metrics` assume valid arguments and do not revalidate them; values read from a data warehouse must still be validated per the trust boundary.

## Block spacing

Decide the spacing of every brace-delimited block on its own, from the inside out:

1. Internal spacing: ignoring blank lines next to the braces, add a blank line only between phases that do clearly different things, where it genuinely helps. Judge by structure, not length: steps toward one result (a run of guards, building a value, a loop and the operation consuming its result) stay together even past 20 lines, while unrelated phases, especially each with its own comment, may be separated even in a short block. A single-statement block never has blank lines, and a guard or `return` does not earn one by itself. Use length only for doubtful cases: below about 20 non-blank lines, nested blocks included, separation rarely pays; above it, more often.
2. Padding: if step 1 left any internal blank line, add one after the opening brace and one before the closing brace; otherwise add none.
3. Final `return`: in a padded block ending with `return`, the closing pad goes before the `return`, which is followed directly by the brace. A `return` in a nested block is not the enclosing block's final statement.

So a block is either compact, with no blank lines at all, or separated, with internal blank lines and both pads, the closing one above a final `return`.

Each nested block is decided independently, in both directions; a function literal (subtest, goroutine, HTTP handler) is decided like a function body. A control-flow block may be separated inside a compact block that is only a thin frame around it, such as declaring a result, filling it in a loop, and returning it.

Separate the construct owning a block from surrounding code with blank lines, keeping declaration comments attached. Exception: a variable may be initialized in its own block placed directly after its declaration:

```go
var x int
{
    x = compute()
}
```

When reviewing in-scope code, apply step 1 ignoring boundary padding, including the pad before a final `return`, and remove the padding if no internal blank line remains.

A function whose body is only a `synctest.Test` call stays compact even if the function passed to it is padded.

Short bodies stay compact even when they perform distinct steps:

```go
func sortedKeys(entries map[string]int) []string {
    keys := make([]string, 0, len(entries))
    for key := range entries {
        keys = append(keys, key)
    }
    slices.Sort(keys)
    return keys
}
```

The separated examples below show only padding placement; real separated blocks are longer:

```go
func topKeys(entries map[string]int, n int) []string {

    keys := slices.Collect(maps.Keys(entries))

    slices.SortFunc(keys, func(a, b string) int {
        return cmp.Compare(entries[b], entries[a])
    })

    return keys[:min(n, len(keys))]
}

func (c *cache) reset() {

    clear(c.entries)
    clear(c.pending)

    c.hits = 0
    c.misses = 0

}
```

## Declaration comments

Every exported package-level declaration and exported method has a declaration comment in the style of the Go standard library, except as noted below. An unexported declaration needs one when it is long, takes or returns several values, or is not obvious.

Keep comments compact and do not restate the code. Declaration comment lines starting in column 1 are at most 80 characters, comment markers included; end-of-line comments are exempt.

In a parenthesized `var` or `const` group whose members lack individual comments, do not comment just one member.

An unexported package-level constant or group in a `_test.go` file may omit its comment when names and context make its fixture role and meaning clear. Never add a comment merely saying tests use it, but keep one that conveys non-obvious semantics, constraints, relationships, or reasons for values.

Document all and only the errors callers can identify and handle without inspecting the message. Exception: a `core` method does not document the `errors.BadRequestError` it returns for invalid arguments, which callers propagate; state non-obvious argument requirements instead, and document it only when callers must distinguish it from other errors.

A short same-line comment on a package-level variable or constant starts lowercase or with a number, has no period, and uses semicolons between phrases.

## Error messages

Never begin an error message with `the`.

Prefix every panic message, and every error reporting an unexpected situation, with the producing package's import path relative to the module path, followed by `: `, as in `core/internal/transformers/lambda: invalid function name`. Do not repeat the prefix when wrapping an error from the same package; re-panic with a `recover` value unchanged; when passing on another package's error, as in `panic(err)`, add the prefix only if its message does not show where the problem occurred.

## Correctness

Validate and bound anything arriving from outside (request bodies, settings, API values, connector responses) before use, unless there is a stated reason not to.

# Admin conventions

For the TypeScript and React code under `admin/`:

- Use boolean expressions in boolean contexts, not truthiness: `name !== ''`, `items.length > 0`, `value != null`. In JSX, idiomatic patterns such as `{condition && <Component />}` are fine when clearer.
- Prefix boolean names with `is`, `has`, `can`, or `should` when it clarifies them.
- Compare with `===` and `!==`, except `value == null` or `value != null` to cover both `null` and `undefined`.
- Use `interface` for object shapes and `type` for unions, tuples, and other aliases.
- Reuse the domain and API types in `admin/src/lib/` instead of redefining their shapes.
- Use PascalCase for components and types, camelCase for functions, variables, and ordinary consts, and UPPER_SNAKE_CASE for module-level constants holding fixed keys, limits, delays, and similar configuration.
- Write components and custom hooks as arrow functions assigned to `const`.
- Type props with an interface `<ComponentName>Props` immediately before the component, unless shared from another module; destructure and type them in the parameter list, without `React.FC`.
- Put component styles in a `.css` file next to the component and import it there, with BEM-style names such as `schema-grid`, `schema-grid__search`, and `schema-grid--loading`.
- For a Shoelace icon, list its file name in `admin/src/shoelace-icons.txt` alphabetically and run `npm run makevendor` from `admin/`; unlisted icons are not embedded and fail to load.

# Reuse

Before writing something new, look for it elsewhere in the repo and reuse or follow it. Prefer the standard library or a `tools/` helper to a hand-rolled version. When deleting something, look for what it leaves behind: callers, columns, fixtures, documentation, dead constants.

# Dependencies

Do not describe a vendored library's behavior from its API or from memory: read it in `vendor/`, following the call chain to the caller that matters, since an option dropped on one path often arrives by another. Cite file and line.

# Database constraints

Use a database constraint when the application cannot validate an invariant easily, efficiently, and atomically: primary keys, foreign keys, uniqueness, and constraints protecting concurrent cross-row state usually qualify. Do not duplicate a deterministic, row-local validation the application can perform before writing, unless a concrete risk or access pattern justifies it.

When a write can legitimately violate a constraint, make the constraint identifiable and handle the violation explicitly, without matching database error messages.

In `CREATE TABLE`, declare single-column constraints inline unless there is a reason not to; use table-level syntax for multi-column constraints and keep established conventions such as table-level primary keys.

Apply this only to code introduced or modified by the change; do not add or remove constraints on unrelated tables without an agreed scope.

# Data warehouse trust boundary

Treat every value from a user-controlled data warehouse as untrusted, including values from tables managed by Krenalis. This boundary protects Krenalis, its resources, and isolation between workspaces, not the integrity of the owning workspace's data. Do not infer from an invalid value whether the user, the warehouse, an attacker, or Krenalis caused it.

Validate every property needed to use warehouse data safely in application state, metrics, resource sizing, control flow, or user-visible output. Fail closed on values that could harm Krenalis: bound payload sizes and row counts at the query source when possible and always before allocation or decoding, reject values outside representable or safety-relevant limits, prevent arithmetic overflow, and do not copy unbounded or unsanitized values into errors or logs. Require structural consistency only when its violation could harm Krenalis, exhaust resources, corrupt Krenalis-owned state, or break workspace isolation.

A compromised warehouse can return false or inconsistent values that are still safe: accept them, and do not add validation solely to detect or correct them or to protect the owning workspace, since truthfulness requires an independent integrity mechanism. Exception: validation may reject values that violate a relationship their meaning guarantees, such as a subset count exceeding the count of its set, so callers can rely on it.

# Uniformity across data warehouses

Each data warehouse has its own package under `warehouses/` implementing the same interfaces. Keep the packages as similar as the platforms allow, so that knowing one means knowing all and a change to one is easy to apply to the others; none is the reference implementation, including any added later.

Mirror file names per responsibility, declaration order, type, function, and variable names, signatures, error messages, and code shape within functions. Add a feature, fix, or validation to every warehouse in the same form unless a platform makes it inapplicable. Tests follow suit: same files, test names, table-driven cases, and comparable fixtures; write a case common to all platforms the same way everywhere, and keep a platform-specific case recognizable as an exception.

Similarity is a means, not an end. Express genuine differences in SQL dialect, type system, quoting, driver behavior, and features naturally where they occur, without pointless abstraction, dead code kept for symmetry, tests that assert nothing on a platform, or renaming of concepts a platform names differently for good reason. Keep necessary divergences local, leaving the surrounding code identical.

# `core` and `cmd` conventions

## API errors and validation

Every error returned by an exported `core` method is exactly one of:

- an error callers are expected to identify and handle, with a type or identity defined by `tools/errors`;
- an internal failure, opaque to callers: its concrete type is unexported, and its chain exposes no exported type or sentinel inviting `errors.As` or `errors.Is`.

Narrow exception: an error from the database driver while accessing Krenalis's own database may be propagated; it remains an internal failure, which `cmd` must not classify by its driver-specific cause. This does not apply to data warehouse errors.

Do not give an internal failure a public classification just to satisfy this rule. The same distinction applies in `cmd` to API endpoint handlers.

Only `core` and `cmd` code serving an API request may return errors implementing `tools/errors.ResponseWriterTo`.

When a `core` API method delegates parsing or validation to a helper and does not branch on the error type, the helper returns a generic error that the method converts at the call site, as in `return errors.BadRequest("%s", err)`, keeping the response semantics visible. The helper returns a `ResponseWriterTo` error only when that classification is part of its reusable contract or its callers need to distinguish or preserve it.

Use `tools/errors.NotFoundError` only for values supplied in the endpoint path or query string, since it means the path does not exist; for a missing resource whose identifier came from the request body, return `tools/errors.UnprocessableError`.

`core` methods serving an endpoint choose errors according to where the endpoint receives each argument, and return the final API-ready error so that `cmd` propagates it untranslated. Handlers in `cmd` parse paths, query strings, and bodies into `core` argument types; on parse failure they return a `tools/errors` error compatible with the invalid-argument error `core` would expose, and otherwise leave semantic validation to `core`.

A `core` method that can return `errors.UnprocessableError` documents it and lists its codes; handlers in `cmd/api.go` never document unprocessable errors.

## Core entry guards

Every exported `core` method called by `cmd` starts with `<receiver>.core.mustBeOpen()`. If the body contains any blank line, put one after the opening brace and one after the guard.

## Pipeline specifications

`core/Pipelines.md` documents, as part of the code, the pipeline rules for each combination of connection role, connector type, and target; update it in the same change whenever `core` or the Admin alters them.

## Database upgrades

Write, restore, or remove the database updater only during the release procedure. When a change requires upgrading an existing Krenalis PostgreSQL database, including a stored-data format change that leaves the schema unchanged, record it in `core/internal/initdb/RELEASE-UPGRADE.md` in the same change, following `.agents/skills/release-upgrade/SKILL.md`.
