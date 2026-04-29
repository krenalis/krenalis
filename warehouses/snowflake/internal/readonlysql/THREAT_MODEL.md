# Snowflake Read-Only SQL Threat Model

This document defines the security model for the Snowflake read-only SQL
validator that backs `Snowflake.QueryReadOnly`.

The validator must be conservative. A valid Snowflake statement is not
necessarily acceptable here. The acceptable subset is limited to analytical
retrieval queries that can be classified safely with a small lexical validator.

## Goal

Allow user-provided analytical `SELECT` queries against the workspace
warehouse while rejecting statements and expressions that can mutate state,
change execution context, trigger external effects, access file stages, or
depend on user-defined executable code.

The intended initial accepted top-level forms are:

- `SELECT ...`
- `WITH ... SELECT ...`

After `WITH`, the first top-level statement body must resolve to `SELECT`.
CTEs that feed mutation, copy, session, scripting, or DDL statements are
rejected.

All other top-level statement classes are out of scope for the first
implementation, even when Snowflake would treat them as metadata reads.

## Security Layers

The validator is one layer, not the whole security boundary.

`Snowflake.QueryReadOnly` must rely on these layers together:

- The lexical validator rejects SQL outside the supported read-only subset.
- The configured Snowflake role must be read-only for Krenalis-managed data.
- The role should not have privileges to create, alter, drop, stage, execute
  procedures, call external functions, or read sensitive account metadata.
- The configured database and schema should not expose user-defined executable
  objects that can be invoked through accepted query syntax.
- The configured database and schema should not allow unqualified names on the
  function allowlist to resolve to user-defined functions or external
  functions.

Unlike PostgreSQL, this plan does not assume an equivalent runtime
`READ ONLY` transaction mode for Snowflake. Snowflake role privileges are the
primary database-enforced runtime protection.

## Assets Protected

The validator is designed to protect:

- Krenalis-managed tables, views, and schemas from mutation.
- Snowflake account, role, warehouse, database, and schema configuration.
- Session state such as current role, current schema, variables, parameters,
  and transaction state.
- Internal and external stages and files.
- External systems reachable through Snowflake features such as external
  functions, procedures, tasks, pipes, integrations, AI/model features, or
  Snowpark code.
- Query history and prior command output that might be reachable from the
  current session.

## Threats

### Data and Schema Mutation

Reject statements that can modify rows or objects, including DML, DDL, and
maintenance commands.

Examples:

- `INSERT`, `UPDATE`, `DELETE`, `MERGE`, `TRUNCATE`
- `CREATE`, `ALTER`, `DROP`, `UNDROP`, `RENAME`
- `COPY INTO <table>`
- task, stream, pipe, procedure, function, integration, stage, and warehouse
  creation or alteration

### Session and Execution Context Changes

Reject statements that can change the meaning or privilege context of later
statements.

Examples:

- `USE ROLE`, `USE DATABASE`, `USE SCHEMA`, `USE WAREHOUSE`
- `ALTER SESSION`
- `SET`, `UNSET`
- `BEGIN`, `COMMIT`, `ROLLBACK`
- Snowflake Scripting constructs such as `DECLARE`, `LET`, and `RETURN`
- `EXECUTE IMMEDIATE`

### Stage and File Access

Reject SQL that accesses or mutates stages and files. Stage reads can expose
data outside the warehouse tables, and file commands can transfer data.

Examples:

- `PUT`, `GET`, `LIST`, `LS`, `REMOVE`, `RM`
- `COPY INTO @stage`
- visible `@stage` references in query text

The validator rejects visible `@` tokens outside strings, comments, and quoted
identifiers.

### Procedure and Function Side Effects

Reject stored procedures, user-defined functions, user-defined table functions,
external functions, and unknown functions.

Snowflake external functions can call remote services. Stored procedures and
Snowpark-based functions can execute code beyond ordinary SQL expression
evaluation. Therefore, the validator should accept only non-qualified function
names on an explicit allowlist of built-in analytical functions.

Qualified calls such as `schema.func(...)` or `database.schema.func(...)` must
be rejected, even if the final function name appears on the allowlist.

### Metadata and Query-History Reads

Reject Snowflake commands and functions that expose metadata outside ordinary
workspace table reads.

Examples:

- `SHOW`, `DESCRIBE`, `DESC`
- `RESULT_SCAN(...)`
- `LAST_QUERY_ID(...)`
- `CURRENT_ROLE`, `CURRENT_USER`, `CURRENT_ACCOUNT`, and similar context
  functions unless explicitly reviewed and allowed

`RESULT_SCAN` is especially sensitive because Snowflake allows it to read the
result set of previous commands, including `SHOW`, `DESCRIBE`, metadata
queries, and stored procedure output.

`SHOW`, `DESCRIBE`, and `DESC` are rejected even though they are read-like
metadata commands.

### Multi-Statement Execution

Reject all multi-statement input. A single trailing statement terminator may be
accepted only if it is the final visible token, matching the PostgreSQL
validator's conservative behavior.

### Lexer Ambiguity

Reject syntax that the validator cannot classify with high confidence.

Potential examples for the implementation to review and either support
precisely or reject:

- dollar-quoted strings
- `::` casts
- stage references and scoped URLs
- table functions via `TABLE(...)`
- pipe operator and command-result post-processing
- functions with network, file, AI, security, account, session, or
  side-effect behavior
- Snowflake-specific literal prefixes or scripting delimiters

## Non-Goals

The validator is not intended to:

- Prove that every accepted query is computationally cheap.
- Implement the full Snowflake SQL grammar.
- Accept every valid read-only Snowflake query.
- Replace Snowflake role-based access control.
- Safely execute arbitrary UDFs, UDTFs, procedures, or external functions.
- Provide a metadata exploration API through `SHOW`, `DESCRIBE`, or
  `RESULT_SCAN`.

## Initial Acceptance Policy

The validator accepts only analytical queries that satisfy all of these
conditions:

- The top-level statement is `SELECT` or `WITH ... SELECT`.
- The query contains exactly one statement.
- All visible SQL words are outside the forbidden token set.
- Function calls are unqualified and present in the built-in allowlist.
- Quoted identifiers are allowed only as identifiers, not as function names.
- Strings, comments, and quoted identifiers are skipped as opaque regions after
  validating that they are terminated.
- Stage references, procedure calls, scripting constructs, and metadata command
  result access are rejected.
- Visible `@` tokens are rejected intentionally, even though this rejects some
  valid Snowflake SQL, to keep stage access outside the initial subset.
- `TABLE(...)` table-function syntax is rejected by default, including
  `FROM TABLE(...)`, unless a later explicit allowlist is added.

## Test Implications

The unit tests for the validator should include both accepted BI-style queries
and bypass-oriented rejected queries:

- `SELECT 1`
- `WITH a AS (SELECT 1) SELECT * FROM a`
- `CREATE TABLE x AS SELECT 1`
- `INSERT INTO x SELECT 1`
- `COPY INTO @stage FROM SELECT ...`
- `SELECT * FROM @stage`
- `PUT`, `GET`, `LIST`, `REMOVE`
- `CALL proc()`
- `EXECUTE IMMEDIATE 'SELECT 1'`
- `USE ROLE`, `USE WAREHOUSE`, `ALTER SESSION`
- `SHOW TABLES`, `DESCRIBE TABLE`
- `SELECT RESULT_SCAN(...)`
- `SELECT TABLE(...)`
- unknown and qualified function calls
- forbidden tokens inside strings, comments, and quoted identifiers

## Review Triggers

The threat model and policy must be reviewed when:

- Snowflake SQL support is expanded beyond the initial subset.
- New function names are added to the allowlist.
- Query execution starts using a different driver or session configuration.
- Snowflake introduces new SQL commands, function classes, stage syntax, or
  scripting features relevant to this validator.
- Krenalis changes how Snowflake roles, schemas, stages, procedures, or
  integrations are provisioned.

Relevant Snowflake documentation areas:

- SQL command reference:
  https://docs.snowflake.com/en/sql-reference/sql
- Stored procedures:
  https://docs.snowflake.com/en/developer-guide/stored-procedure/stored-procedures-overview
- User-defined functions and table functions:
  https://docs.snowflake.com/en/developer-guide/udf/udf-overview
- External functions:
  https://docs.snowflake.com/en/sql-reference/external-functions-introduction
- Stages and file transfer commands:
  https://docs.snowflake.com/en/user-guide/data-load-local-file-system
- `RESULT_SCAN` and persisted query results:
  https://docs.snowflake.com/en/sql-reference/functions/result_scan
