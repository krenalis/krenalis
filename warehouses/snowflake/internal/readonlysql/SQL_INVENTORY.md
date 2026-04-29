# Snowflake SQL Inventory for Read-Only Validation

This inventory maps Snowflake SQL surfaces to the initial read-only validator
policy. It is not a complete Snowflake reference. It lists the dialect areas
that matter for deciding whether `readonlysql.ValidateReadOnly` should accept,
reject, or defer support for a construct.

Classification:

- `Accept`: in scope for the first validator.
- `Reject`: explicitly out of scope and should produce a rejection.
- `Review`: not accepted initially; requires a later security review before
  support is added.

## Query Syntax

| Surface | Classification | Validator impact |
| --- | --- | --- |
| Top-level `SELECT` | Accept | Primary accepted statement form. |
| `WITH ... SELECT` | Accept | Accept only when the top-level body after CTE definitions is `SELECT`. |
| Subqueries | Accept | Allowed inside accepted `SELECT` statements if they do not contain rejected visible tokens. |
| Joins | Accept | Ordinary analytical query syntax. |
| `WHERE`, `GROUP BY`, `HAVING`, `QUALIFY`, `ORDER BY`, `LIMIT`, `OFFSET` | Accept | Ordinary analytical query syntax. |
| `UNION`, `UNION ALL`, `INTERSECT`, `EXCEPT` | Accept | Set operations remain within query syntax. |
| Top-level `VALUES` | Reject | Not part of the initial top-level subset. |
| `TABLE(...)` in `FROM` | Reject | Can invoke table functions, UDTFs, or table literals that resolve object names from strings or variables. |
| Pipe command-result processing (`->>`) | Reject | Snowflake documents command-result post-processing as an alternative to `RESULT_SCAN`; keep it out of scope. |

## DML and Data Movement

| Surface | Classification | Validator impact |
| --- | --- | --- |
| `INSERT` | Reject | Data mutation. |
| `UPDATE` | Reject | Data mutation. |
| `DELETE` | Reject | Data mutation. |
| `MERGE` | Reject | Data mutation. |
| `TRUNCATE` | Reject | Data mutation. |
| `COPY INTO <table>` | Reject | Data loading. |
| `COPY INTO <location>` | Reject | Data unloading to stages/files. |
| `PUT` | Reject | Uploads local files to an internal stage. |
| `GET` command | Reject | Downloads files from an internal stage. `get(...)` as a function is handled separately. |
| `LIST` / `LS` | Reject | Lists staged files. |
| `REMOVE` / `RM` | Reject | Removes staged files. |
| Visible `@stage` syntax | Reject | Stage access is outside the initial analytical subset. |

## DDL and Object Management

| Surface | Classification | Validator impact |
| --- | --- | --- |
| `CREATE` | Reject | Object creation. |
| `ALTER` | Reject | Object/session/account mutation. |
| `DROP` | Reject | Object deletion. |
| `UNDROP` | Reject | Object restoration. |
| `RENAME` | Reject | Object mutation. |
| `COMMENT` | Reject | Metadata mutation. |
| `GRANT` | Reject | Privilege mutation. |
| `REVOKE` | Reject | Privilege mutation. |
| Object commands for users, roles, warehouses, databases, schemas, stages, integrations, tasks, streams, pipes, procedures, functions, and policies | Reject | Covered by command tokens above; add dedicated tests when a token is easy to miss. |

## Session, Transaction, and Scripting

| Surface | Classification | Validator impact |
| --- | --- | --- |
| `USE ROLE`, `USE DATABASE`, `USE SCHEMA`, `USE WAREHOUSE` | Reject | Changes execution context. |
| `ALTER SESSION` | Reject | Changes session behavior. |
| `SET` / `UNSET` | Reject | Changes session variables or parameters. |
| `BEGIN`, `COMMIT`, `ROLLBACK` | Reject | Transaction control. `BEGIN` also starts Snowflake Scripting blocks. |
| `DECLARE`, `LET`, `RETURN` | Reject | Snowflake Scripting. `END` is not globally rejected because `CASE ... END` is accepted query syntax. |
| `EXECUTE IMMEDIATE` | Reject | Dynamic SQL or scripting execution. |
| Anonymous Snowflake Scripting blocks | Reject | Can contain arbitrary SQL statements and control flow. |

## Metadata and History

| Surface | Classification | Validator impact |
| --- | --- | --- |
| `SHOW` | Reject | Read-like metadata command, not analytical table query syntax. |
| `DESCRIBE` / top-level `DESC` | Reject | Read-like metadata command, not analytical table query syntax. `ORDER BY ... DESC` remains accepted query syntax. |
| `EXPLAIN` | Reject | Not an analytical result query. |
| `RESULT_SCAN(...)` | Reject | Can read prior command output, including `SHOW`, `DESCRIBE`, metadata queries, and procedure output. |
| `LAST_QUERY_ID(...)` | Reject | Enables prior query-result access patterns through `RESULT_SCAN`. |
| Account usage / information schema table functions | Reject | Usually reached via `TABLE(...)`; not part of the initial analytical subset. |

## Functions

| Surface | Classification | Validator impact |
| --- | --- | --- |
| Unqualified built-in scalar or aggregate functions on the allowlist | Accept | Compare case-insensitively against the policy allowlist. |
| Unknown scalar functions | Reject | Could be UDFs, external functions, or future functions with side effects/sensitive access. |
| `CALL` | Reject | Invokes stored procedures. |
| Qualified function calls | Reject | Avoid explicit UDF/external/built-in resolution through database/schema qualification. |
| Quoted function names | Reject | Avoid case-sensitive or user-defined function resolution. |
| UDFs and external functions | Reject | Snowflake supports user-defined and external executable functions. |
| UDTFs and built-in table functions | Reject | Reached through `TABLE(...)`; require explicit later review. |
| Table literals through `TABLE(<string_or_variable>)` | Reject | Snowflake treats `TABLE(...)` table literals as a way to resolve table names from strings, session variables, or bind variables. |
| Context/session functions such as `CURRENT_ROLE`, `CURRENT_USER`, `CURRENT_ACCOUNT`, `CURRENT_DATABASE`, `CURRENT_SCHEMA`, `CURRENT_WAREHOUSE` | Reject | Metadata/session disclosure, except date/time forms explicitly allowed by policy. |
| File, network, AI/model, security, account, or administration functions | Reject | Excluded unless explicitly reviewed and allowlisted later. |

## Identifiers and Lexical Forms

| Surface | Classification | Validator impact |
| --- | --- | --- |
| Unquoted identifiers | Accept | Snowflake allows ASCII letters or `_` as the first character, followed by ASCII letters, `_`, decimal digits, or `$`. The lexer treats `$` as part of an identifier only after the first character and rejects visible non-ASCII characters outside opaque regions. |
| Double-quoted identifiers | Accept | Snowflake allows quoted identifiers to start with and contain numbers, spaces, punctuation, periods, `$`, `@`, extended ASCII, and non-ASCII characters. Skip them as opaque identifiers after validating termination. Do not allow quoted identifiers as function names. |
| Escaped quotes in double-quoted identifiers | Accept | Snowflake represents an embedded double quote as `""`; the lexer skips the doubled quote as part of the identifier. |
| Qualified object names with quoted components | Accept | Quoted components may contain periods, and periods outside quotes separate object-name parts. The lexer must not treat periods inside quotes as qualification separators. |
| Identifier length up to 255 characters | Review | Snowflake documents a 255-character identifier limit. The validator relies on Snowflake to enforce this limit instead of duplicating character-counting rules for quoted Unicode identifiers and escaped quotes. |
| `IDENTIFIER(...)` syntax | Reject | Can resolve object names from strings or variables; not part of the initial function allowlist. |
| Session variable references such as `$name` | Reject | Can feed `IDENTIFIER(...)`, `TABLE(...)`, and dynamic SQL patterns. Dollar signs remain allowed only inside unquoted identifiers after the first character. |
| String literals | Accept | Skip as opaque strings after validating termination. |
| Dollar-quoted string constants (`$$...$$`) | Reject | Snowflake supports them as string constants; reject initially because they are common in executable object definitions and require separate lexer handling. |
| Visible `::` casts | Reject | Cast syntax is valid Snowflake SQL but rejected initially to simplify classification. |
| `CAST(... AS ...)` | Accept | Treat as a non-function SQL construct. |
| `TRY_CAST(...)` | Reject | Not in the initial function allowlist. |

## Tests Derived From This Inventory

The validator should include rejection tests for each `Reject` row that is not
already covered by a broader token-level test. The most important bypass tests
are:

- `CREATE TABLE x AS SELECT 1`
- `INSERT INTO x SELECT 1`
- `COPY INTO @stage FROM (SELECT 1)`
- `SELECT * FROM @stage`
- `SELECT * FROM TABLE(RESULT_SCAN(LAST_QUERY_ID()))`
- `SELECT * FROM TABLE('DB.SCHEMA.TABLE')`
- `SELECT RESULT_SCAN(-1)`
- `CALL proc()`
- `SHOW TABLES`
- `DESCRIBE TABLE x`
- `USE ROLE analyst`
- `ALTER SESSION SET QUOTED_IDENTIFIERS_IGNORE_CASE = TRUE`
- `EXECUTE IMMEDIATE 'SELECT 1'`
- `DECLARE x INT`
- `SELECT IDENTIFIER('T') FROM t`
- `SELECT schema.lower('x')`
- `SELECT "LOWER"('x')`
- `SELECT unknown_function(x) FROM t`
- `SELECT x::NUMBER FROM t`
- `SELECT $$DELETE FROM t$$`
- `SELECT $session_table FROM t`

## Official Documentation References

- SQL command reference:
  https://docs.snowflake.com/en/sql-reference-commands
- Query syntax:
  https://docs.snowflake.com/en/sql-reference/constructs
- File transfer command `GET`:
  https://docs.snowflake.com/en/sql-reference/sql/get
- Table functions:
  https://docs.snowflake.com/en/sql-reference/functions-table
- Table literals:
  https://docs.snowflake.com/en/sql-reference/literals-table
- String constants:
  https://docs.snowflake.com/en/sql-reference/data-types-text
- Snowflake Scripting `DECLARE`:
  https://docs.snowflake.com/en/sql-reference/snowflake-scripting/declare
- Snowflake Scripting `BEGIN ... END`:
  https://docs.snowflake.com/en/sql-reference/snowflake-scripting/begin
- `EXECUTE IMMEDIATE`:
  https://docs.snowflake.com/en/sql-reference/sql/execute-immediate
- Object identifiers:
  https://docs.snowflake.com/en/sql-reference/identifiers-syntax
- `IDENTIFIER(...)` syntax:
  https://docs.snowflake.com/en/sql-reference/identifier-literal
- `RESULT_SCAN`:
  https://docs.snowflake.com/en/sql-reference/functions/result_scan
