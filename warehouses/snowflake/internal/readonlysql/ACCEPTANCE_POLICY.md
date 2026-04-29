# Snowflake Read-Only SQL Acceptance Policy

This document defines the initial SQL subset accepted by the Snowflake
read-only validator. It is intentionally narrower than Snowflake SQL and should
be treated as the implementation contract for the first version of
`readonlysql.ValidateReadOnly`.

The policy is conservative: valid Snowflake SQL is rejected unless it is part
of the explicit analytical subset below.

## Accepted Top-Level Statements

The validator accepts only one top-level statement in one of these forms:

- `SELECT ...`
- `WITH ... SELECT ...`

For `WITH`, the first top-level statement after the CTE definitions must be
`SELECT`. The validator must reject `WITH` forms that feed `INSERT`, `CREATE`,
`COPY`, scripting, session, or other non-SELECT statements.

A single trailing `;` may be accepted only when it is the final visible token.
Any other visible semicolon is a multi-statement rejection.

## Required Rejections

The validator must reject a query when any of these conditions are observed in
visible SQL text outside strings, comments, and quoted identifiers.

### Non-Analytical Statements

Reject DML, DDL, administrative, and maintenance statement tokens, including:

- `ALTER`
- `CALL`
- `COMMENT`
- `COPY`
- `CREATE`
- `DELETE`
- `DROP`
- `GRANT`
- `INSERT`
- `MERGE`
- `RENAME`
- `REVOKE`
- `TRUNCATE`
- `UNDROP`
- `UPDATE`

### Session, Transaction, and Scripting Tokens

Reject tokens that can change execution context or invoke Snowflake Scripting:

- `BEGIN`
- `COMMIT`
- `DECLARE`
- `EXECUTE`
- `IMMEDIATE`
- `LET`
- `RETURN`
- `ROLLBACK`
- `SET`
- `UNSET`
- `USE`

`BEGIN` is rejected whether Snowflake would interpret it as a transaction
token or a scripting block token.

`END` is not rejected as a global token because ordinary `CASE ... END`
expressions are accepted. Snowflake Scripting blocks are rejected through their
entry and control tokens such as `BEGIN`, `DECLARE`, `LET`, `RETURN`, and
`EXECUTE`.

### Metadata Commands and Query-History Access

Reject metadata command tokens and functions that expose prior command output
or session/query state:

- `DESC`
- `DESCRIBE`
- `EXPLAIN`
- `SHOW`
- `RESULT_SCAN(...)`
- `LAST_QUERY_ID(...)`

`SHOW`, `DESCRIBE`, and top-level `DESC` are rejected even though they are
read-like. They are not part of the analytical table-query subset and can feed
`RESULT_SCAN`. `ORDER BY ... DESC` remains ordinary query syntax.

### Stage and File Access

Reject stage and file transfer syntax:

- `GET`
- `LIST`
- `LS`
- `PUT`
- `REMOVE`
- `RM`
- visible `@` tokens

The visible `@` rule intentionally rejects some valid Snowflake SQL. This keeps
stage reads, stage writes, and staged-file references out of the first version.

`GET` is also the name of a Snowflake semi-structured data function. The file
transfer command form is rejected; `get(...)` may be accepted only when the
lexer classifies it as an unqualified function call and the function allowlist
contains `get`.

### Table Functions

Reject `TABLE(...)` by default, including `FROM TABLE(...)`.

Snowflake table-function syntax can invoke built-in table functions, UDTFs, or
procedural/external surfaces. It may be allowed later only through an explicit
table-function allowlist.

### Function Calls

Reject every function call unless all of these are true:

- the function name is unquoted
- the function name is unqualified
- the lowercased function name is in the scalar/aggregate allowlist below

Reject calls such as:

- `"LOWER"('x')`
- `schema.lower('x')`
- `database.schema.lower('x')`
- `unknown_function(...)`

This policy assumes deployment prevents unqualified allowlisted names from
resolving to user-defined functions or external functions in the active
database and schema.

### Ambiguous Lexical Forms

Reject syntax that the first validator cannot classify safely:

- dollar-quoted strings
- visible `::` casts
- pipe operator command-result post-processing
- Snowflake Scripting delimiters or blocks
- literal or identifier forms not intentionally modeled by the lexer

Standard single-quoted strings, double-quoted identifiers, line comments, and
block comments may be skipped as opaque regions after termination is verified.

## Initial Function Allowlist

The initial allowlist is intentionally small and BI-oriented. Names are
case-insensitive and must be unqualified.

### Aggregate and Window-Oriented Functions

- `avg`
- `count`
- `count_if`
- `listagg`
- `max`
- `min`
- `sum`

Window syntax such as `OVER (...)` is allowed only as ordinary SQL syntax
around an allowlisted function call. Unknown window functions remain rejected.

### Numeric Functions

- `abs`
- `ceil`
- `ceiling`
- `floor`
- `round`

### String Functions

- `coalesce`
- `concat`
- `concat_ws`
- `length`
- `len`
- `lower`
- `ltrim`
- `nullif`
- `replace`
- `rtrim`
- `split_part`
- `substring`
- `trim`
- `upper`

### Date and Time Functions

- `current_date`
- `current_time`
- `current_timestamp`
- `date_part`
- `date_trunc`
- `extract`

The `current_*` entries may be implemented as recognized Snowflake special
forms when they appear without parentheses, or as ordinary allowlisted function
calls when Snowflake accepts parentheses. Context/session disclosure functions
remain rejected unless explicitly reviewed. Examples include
`current_account`, `current_role`, `current_user`, `current_warehouse`,
`current_database`, and `current_schema`.

### Semi-Structured Data Functions

- `array_construct`
- `array_size`
- `get`
- `get_path`
- `json_extract_path_text`
- `object_construct`
- `typeof`

This list may be expanded later for real BI needs. Functions that parse
external formats, access stages/files, call external services, invoke AI/model
features, or expose account/session/security metadata are excluded.

## Non-Function SQL Constructs

These ordinary query constructs are in scope when they appear inside an
accepted `SELECT` or `WITH ... SELECT` statement and do not contain forbidden
tokens:

- projection expressions
- `FROM` table/view references
- joins and subqueries
- `WHERE`
- `GROUP BY`
- `HAVING`
- `QUALIFY`
- `ORDER BY`
- `LIMIT`
- `OFFSET`
- `UNION`, `UNION ALL`, `INTERSECT`, `EXCEPT`
- `CASE`
- `CAST(... AS ...)`

`CAST(... AS ...)` is allowed as a SQL construct because it is not a generic
function call in this policy. The `::` cast syntax remains rejected initially
because it is easier to misclassify lexically.

## Not Accepted Initially

These are not accepted in the first version, even if Snowflake can execute
them as read-only operations:

- `VALUES` as a standalone top-level query form
- `SHOW`, `DESCRIBE`, `DESC`
- `SELECT * FROM @stage`
- `SELECT * FROM TABLE(...)`
- `RESULT_SCAN(...)`
- `TRY_CAST(...)`
- user-defined functions
- user-defined table functions
- external functions
- stored procedures
- Snowflake Scripting blocks
- stage file listing, upload, download, or removal

## Minimum Test Matrix

The validator tests should include these classes:

- accepted simple `SELECT`
- accepted `WITH ... SELECT`
- accepted BI-style aggregates, grouping, filtering, ordering, and limits
- accepted allowlisted scalar functions
- accepted forbidden words inside strings, comments, and quoted identifiers
- rejected multi-statement input
- rejected DML, DDL, session, transaction, and scripting tokens
- rejected metadata commands
- rejected stage/file syntax and visible `@`
- rejected `TABLE(...)`
- rejected `RESULT_SCAN(...)` and `LAST_QUERY_ID(...)`
- rejected unknown, quoted, and qualified function calls
- rejected top-level `VALUES`
- rejected `::` casts
- rejected unterminated strings, identifiers, and block comments

## Sources Used for the Initial Function Set

- Snowflake aggregate functions:
  https://docs.snowflake.com/en/sql-reference/functions-aggregation
- Snowflake string and binary functions:
  https://docs.snowflake.com/en/sql-reference/functions-string
- `DATE_TRUNC`:
  https://docs.snowflake.com/en/sql-reference/functions/date_trunc
- `DATE_PART`:
  https://docs.snowflake.com/en/sql-reference/functions/date_part
- `EXTRACT`:
  https://docs.snowflake.com/en/sql-reference/functions/extract
- Snowflake semi-structured and structured data functions:
  https://docs.snowflake.com/en/sql-reference/functions-semistructured
- `RESULT_SCAN`:
  https://docs.snowflake.com/en/sql-reference/functions/result_scan
