# Krenalis

Go monorepo. Dependencies are vendored in `vendor/`. This file holds the conventions that every change must follow.

# Changes

Keep the diff as small as the change allows. Touch only the lines the change needs: no drive-by reformatting, renaming, reordering, or rewrapping of a comment you happened to read. When the same fix can be written in two ways, pick the one that leaves more of the surrounding lines untouched. Reread your own diff before reporting it: every hunk should be one you can justify.

# Review process

When assessing a change against existing practices, use the code as it existed before the changes under review as the source of precedent. Do not treat code introduced or altered by the current change as evidence of an established convention.

Apply review rules to code introduced or modified by the change under review. Do not alter untouched pre-existing code merely to retrofit a newly adopted convention; handle that cleanup separately unless the user explicitly expands the scope or the in-scope change cannot be implemented correctly without it.

# Go conventions

## Line length

Allow Go code lines to be up to 120 characters long without wrapping. A line may exceed 120 characters when a more specific convention requires it, notably the rule that keeps exported signatures on one line.

## File organization

Keep declarations in a predictable order. Place exported type groups before unexported type groups. Complete an exported type group, including all of its exported and unexported methods, before declaring an unexported type. For each type, place its declarations as a group in this sequence:

1. The type itself.
2. Its constructor functions, when present.
3. Its exported methods, sorted alphabetically.
4. Its unexported methods, sorted alphabetically.

After the type groups, place package-level functions in alphabetical order.

Place package-level constants and variables at the beginning of the file when they have broad relevance. When one is specific to a single function or method, or is otherwise of secondary importance, declare it immediately before the function or method that uses it.

## Layout and naming

An exported function or method never calls another exported function or method of the same package. Extract a shared unexported one and call that from both.

Declare a variable as close as possible to where it is used, as long as the code stays readable.

A field guarded by a mutex carries a `// protected by mu` comment, so the guard shows up in the editor when reading the field's documentation.

Keep the receiver name the type already uses. The facade types in `core/` (`Connection`, `Connector`, `Organization`, `Pipeline`, `Workspace`) use `this`; leave it as it is.

## Imports

Outside the implementation and compatibility tests of `tools/errors`, always import `github.com/krenalis/krenalis/tools/errors` instead of the standard-library `errors` package. The repository package exposes every standard `errors` name in addition to repository-specific functionality.

Use an imported package's default name unless Go requires disambiguation because of an actual identifier conflict. Do not alias an import merely to avoid reusing the same name for a variable, field, or selector when the language permits it.

When a real conflict exists, decide whether to alias the import or rename the conflicting identifier by weighing the context and importance of both names. Preserve whichever name is more obvious and expected for readers; do not treat the import name as the automatic choice to change.

## Maps

When initializing an empty writable map without a capacity hint, prefer a map literal such as `m := map[K]V{}`. Use `make(map[K]V, capacity)` when providing a meaningful capacity hint.

## Composite values

When allocating the zero value of a composite type, prefer `&T{}` over `new(T)` when both forms are valid.

## Signatures

Keep the complete parameter list of every function or method on one source line, regardless of line length. A list too long to read comfortably on one line is a signal to evaluate a lower-cardinality signature or another API design, not a reason to wrap the parameters.

During review, do not change an exported signature without prior agreement. Until a redesign is approved, preserve the signature and place its existing parameters on one line.

## Contexts

Functions and methods whose execution time cannot be bounded in advance must accept a `context.Context` as their first parameter. Omit the context only when there is a compelling reason, and make that exception explicit and justified.

Write a function or method that accepts a `context.Context` under the assumption that its caller checked immediately before the call that the context had not already expired or been canceled, whether or not the caller actually did so. Do not begin the callee with an eager `ctx.Err()` or `ctx.Done()` check that merely repeats this assumed precondition.

## Error handling

Handle an error returned by a function or method call with a direct `if err != nil { ... }` branch, whether the error is assigned before the branch or in its initializer. Use a different structure only when there is a concrete reason that makes the direct nil check unsuitable.

When a function or method call has side effects, never place it in the initializer of an error check. Assign the returned error first, then check it. Prefer:

```go
err := f()
if err != nil {
    return err
}
```

over:

```go
if err := f(); err != nil {
    return err
}
```

Perform any classification of the returned error, including `errors.Is` checks for sentinel errors and `errors.AsType` checks for typed errors, inside that non-nil branch rather than using classification as a substitute for the `err != nil` check. In tests that expect a particular error, enter the `err != nil` branch, verify the error there, and fail after the branch when the call returned nil.

Never attach an `else` block to an `if err != nil { ... }` error check. Restructure the control flow using a more readable alternative; this may be a `switch`, but need not be one.

Prefer `errors.AsType` to `errors.As` when matching a typed error.

A direct return is a valid exception when handling the error would consist only of returning it, possibly together with the other values returned by the same call. When the results are assigned first, they may be returned unchanged without an `if err != nil` branch. Prefer:

```go
err = f()

return err
```

over:

```go
err = f()
if err != nil {
    return err
}

return nil
```

When the call is short and no blank line separates it from the return, simplify it further:

```go
return f()
```

This applies equally when the call returns additional values. Prefer:

```go
return f()
```

over:

```go
n, err := f()
if err != nil {
    return 0, err
}
return n, nil
```

## Tests

In tests, use `t.Context()` for operations whose lifetime follows the test. Pass `t.Context()` directly instead of first assigning it to a local variable when its scope of use is small; name it only when it spans a broader portion of the test or must be used to derive another context. When a test helper already accepts `*testing.T`, obtain that context inside the helper instead of also passing a `context.Context`. Accept a separate context only when callers intentionally need to supply a context with different values, deadline, cancellation state, or lifetime.

Use the standard-library `testing/synctest` package when testing concurrent or asynchronous Go code whose behavior depends on timers, deadlines, or goroutine quiescence and can run entirely inside a synctest bubble. Prefer its virtual time and `synctest.Wait` to real sleeps or polling. Do not use it around real network I/O, system calls, or external processes unless those dependencies are replaced with fakes that operate entirely within the bubble.

When running the complete Admin test suite, you may use `go run ./test/commit -just-test-admin` to receive each individual test result without waiting for the complete suite to finish. This command is optional.

## Behavioral contracts

A function or method must return a non-nil error whenever it does not perform the behavior promised primarily by its name and secondarily by its declaration comment. Do not report success after silently skipping, failing, or only partially completing a promised operation.

Permit an exception only when it is narrowly scoped and supported by a compelling reason. Make the exceptional success semantics explicit in the function's name or declaration comment so callers do not mistake them for completion of the usual contract.

## Function extraction

Do not extract part of a function or method solely to unit-test it independently. Keep small or operation-specific implementation steps inline; extract a helper only when it represents a meaningful production abstraction independently of its testability.

For parsing, validation, and conversion between representations, extract a helper only when it captures non-trivial, reusable rules inherent to the nature or representation of a value. Identical code in multiple operations is not sufficient by itself. Prefer focused parsers, validators, and converters for individual value concepts that callers can compose, rather than one shared function that processes an operation's complete set of parameters or results.

Before introducing a parsing, validation, or conversion function, inspect how comparable cases were handled in the same context before the current change. Keep the code inline when similar work is inline; when extraction is justified, match the granularity and abstraction level of existing parsers, validators, or converters. If the local precedent does not make the choice clear, ask for approval before introducing the function.

When a parsing or validation sequence is dense, covers multiple distinct values or coherent groups, and would otherwise be difficult to scan, precede each group with a short comment that names the operation and its subject, such as `// Validate start and end.` Use one comment for a coherent group rather than commenting every condition. Add a blank line before such a comment only when it makes the group boundary clearer. Do not apply this pattern to a single check, a short straightforward sequence, or code for which the comments would merely restate the conditions.

Exported functions and methods are responsible for validating their own arguments.

Exported methods in `core/internal/metrics` are an exception: they must assume that callers provide valid arguments and must not repeat argument validation. This exception applies only to arguments; values read from a data warehouse remain untrusted and must still be validated according to the data warehouse trust boundary.

## Block spacing

If a brace-delimited code block contains any blank line, make all of its boundaries visible: leave the first line after the opening brace and the last line before the closing brace blank, and separate the complete construct that owns the block from the surrounding code with blank lines. Apply this rule to function and method bodies as well.

When a function or method body ends with a `return`, place the final separating blank line immediately before the `return` instead of between the `return` and the closing brace.

When appropriate, code that initializes a variable may be confined to its own block. Declare the variable that receives the result outside the block and place the block immediately after that declaration, without an intervening blank line:

```go
var x int
{
    // compute the value of x
    x = compute()
}
```

## Declaration comments

Every exported package-level type, function, variable, and constant, as well as every exported method, must have a declaration comment written in the style of the Go standard library, except for the grouped declarations and self-explanatory test fixture constants described below. An unexported declaration does not need one, but add it when the declaration is long, takes or returns several values, or its behavior is not obvious from the code.

Keep comments compact: one precise sentence beats three loose ones, and do not restate what the code already says.

When a variable or constant belongs to a parenthesized `var` or `const` declaration whose other members do not have individual declaration comments, do not add an individual comment only to that member. Preserve the established comment style consistently throughout the group.

An unexported package-level constant or constant group in a `_test.go` file may omit its declaration comment when its identifiers and surrounding context already make both its fixture role and meaning clear. Do not add a comment that merely labels such constants as test fixtures or states that tests use them. Keep the comment when it conveys non-obvious semantics, constraints, relationships, or reasons for particular values.

When a function or method can return an error, document all and only the errors that callers can identify and handle without inspecting the error message. Do not document errors whose only distinguishing information is their message.

A short comment for a package-level variable or constant may appear on the same line as its declaration. In that case, start the comment with a lowercase letter or a number, do not use a period, and use a semicolon to separate phrases that would otherwise be separate sentences.

## Error messages

Never begin an error message with the article `the`.

## Correctness

Anything arriving from outside — request bodies, settings, API values, connector responses — is validated and bounded before use, unless there is a stated reason not to.

# Reuse

Before writing something new, look for the same thing already done elsewhere in the repo, and reuse it or follow it.

Prefer the standard library, or an existing helper under `tools/`, over a hand-rolled version.

When deleting something, look for what it leaves behind: callers, columns, fixtures, documentation, constants that are now dead.

# Dependencies

Do not state what a vendored library does from its API surface or from memory. Read it in `vendor/` and follow the call chain up to the caller that matters: an option that looks dropped along one path often arrives by another. Cite file and line when reporting such behavior.

# Database constraints

Use database constraints when an invariant cannot be validated easily,
efficiently, and atomically by the application. Primary keys, foreign keys,
uniqueness constraints, and constraints that protect concurrent cross-row
state are generally appropriate because PostgreSQL is the natural enforcement
boundary for them.

Do not add a database constraint merely to duplicate a deterministic,
row-local validation that the application can perform before writing. Keep
such validation in the application unless a concrete risk or access pattern
justifies moving it to the database.

When a write can legitimately violate a database constraint, make the
constraint identifiable and handle its violation explicitly in the
application. Do not rely on matching database error messages.

In `CREATE TABLE` definitions, prefer declaring a single-column constraint
inline with its column when there is no specific reason to use a table-level
declaration. Use table-level syntax for multi-column constraints and preserve
an established constraint-specific convention, such as the schema's
table-level primary keys.

Apply this policy to code introduced or modified by the change under review.
Do not retrofit constraints onto unrelated existing tables, or remove their
existing constraints, without a separately agreed scope.

# Data warehouse trust boundary

Treat every value returned by a user-controlled data warehouse as untrusted,
including values read from tables managed by Krenalis. This is a safety boundary
for Krenalis, its resources, and isolation between workspaces; it is not an
integrity boundary for data belonging to the workspace that controls the
warehouse. Do not infer from an invalid value whether the cause is user
modification, a warehouse defect, deliberate manipulation, or a Krenalis
defect.

Validate every property needed to use warehouse data safely in application
state, metrics, resource sizing, control flow, or user-visible output. Fail
closed when a value could harm Krenalis: bound payload sizes and row counts at
the query source when possible and always before allocation or decoding, reject
values outside representable or safety-relevant semantic limits, prevent
arithmetic overflow, and do not copy unbounded or unsanitized values into
errors or logs. Require structural consistency only when its violation could
harm Krenalis, consume excessive resources, corrupt Krenalis-owned state, or
break isolation between workspaces.

A compromised warehouse can return false or mutually inconsistent values that
still satisfy every safety and representation constraint. Do not add validation
whose sole purpose is detecting or correcting such values, or protecting the
owning workspace from their consequences. Accept them when they are safe for
Krenalis. Establishing truthfulness or authenticity requires an independent
integrity mechanism.

# `core` and `cmd` conventions

## API errors and validation

Every error returned by an exported method in `core` must fall into exactly one of two categories:

- An error that callers are expected to identify and handle must have a type or identity defined by `tools/errors`.
- An error that represents an internal failure of the method must be opaque to callers: its concrete type must not be exported, and its error chain must not expose an exported type or sentinel that would invite callers to identify or handle the underlying cause with `errors.As` or `errors.Is`.

As a narrow exception, an exported `core` method may propagate an error returned by the database driver while accessing Krenalis's own database. Such an error remains an internal failure even when the driver gives it an exported type or identity; callers in `cmd` must treat it as an internal error rather than identify or handle its driver-specific cause. This exception does not apply to errors from a data warehouse.

Do not assign a public error classification to an internal failure merely to satisfy this rule. Apply the same distinction in `cmd` to functions and methods that handle API endpoints.

Only functions and methods in `core` or `cmd` that serve an API request may return errors implementing `tools/errors.ResponseWriterTo`. Do not expose HTTP response errors from code outside that boundary.

When a `core` method serving an API delegates parameter parsing or validation to a helper, prefer the helper to return a generic, non-`ResponseWriterTo` error if the method does not need to branch on the error type. Convert that error explicitly at the call site to the final API error, for example with `return errors.BadRequest("%s", err)`, so the response semantics remain visible in the method. Let the helper return a `ResponseWriterTo` error only when its error classification is part of the helper's reusable contract or its callers genuinely need to distinguish or preserve that classification.

Use `tools/errors.NotFoundError` only when the value concerned was supplied through the endpoint path or query string, because this error indicates that the requested path does not exist. When a resource does not exist and its identifier came from the request body, return a `tools/errors.UnprocessableError` instead.

API endpoint handlers in `cmd` parse values from paths, query strings, and request bodies into the argument types accepted by `core`. If parsing fails, return a `tools/errors` error compatible with the invalid-argument error that `core` would expose instead of passing a plainly malformed value. Once parsing succeeds, do not duplicate semantic validation in `cmd`: let the exported `core` method validate the argument and propagate its API-ready error directly.

Methods in `core` that serve an endpoint must select their error according to where that endpoint receives each argument, even though this couples them to the endpoint shape. Return the final API-ready error from `core` so that `cmd` can propagate it without translation.

A method of `core` that can return an `errors.UnprocessableError` documents it and lists the codes. Handlers in `cmd/api.go` never document unprocessable errors.

## Core entry guards

Every exported method in `core` that is called by `cmd` must execute `<receiver>.core.mustBeOpen()` as its first statement. If the method body contains any blank line, leave a blank line immediately after the opening brace and another immediately after the guard statement.

# Before finishing

Run `go build ./...`, `go vet ./...`, and `gofmt -l` over what you touched. Add tests where the package already has them. Report plainly what passed, what failed, and what you did not run.
