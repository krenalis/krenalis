# AGENTS.md

## Directory: `cmd/mcp/sqlchecker`

This directory contains a Go package responsible for validating PostgreSQL queries used against the Meergo CDP data warehouse.

### Purpose

The package exposes dialect-specific functions that analyze an input SQL query and determine whether it is **read-only**. Currently, only PostgreSQL is supported via `CheckPostgreSQL`. Additional dialects (e.g. Snowflake) will be added in the future as separate `Check<Dialect>` functions.

- If the query performs only read operations (e.g. `SELECT`, `JOIN`, `WITH`, aggregations), the function returns `nil`.
- If the query performs any write, mutation, or schema-altering operation (e.g. `INSERT`, `UPDATE`, `DELETE`, `UPSERT`, `MERGE`, `TRUNCATE`, `ALTER`, `DROP`, `CREATE`, or side-effecting functions), the function returns an error.
- If the query is designed to overload the warehouse, cause infinite loops, or never terminate, the function returns an error. Examples include: unbounded recursive CTEs (`WITH RECURSIVE` without a proper termination condition), cartesian products on large tables without join conditions, `generate_series` with extremely large ranges, `pg_sleep` or other functions that block execution, and any construct that would result in excessive resource consumption or denial of service.

The intent is to prevent accidental or malicious modifications to the underlying data warehouse, as well as queries that could degrade performance or availability.

### Context

The target database is a data warehouse backing **Meergo**, a Customer Data Platform (CDP). Queries are expected to be analytical and non-destructive. Any operation that could modify data, metadata, or execution state is treated as unsafe.

### Assumptions and Constraints

- The checker is conservative by design: if a query cannot be confidently classified as read-only, it should be rejected.
- PostgreSQL-specific syntax and semantics are assumed.
- The checker does not execute queries; it only performs static analysis.

### Expected Usage

This package is typically used as a guardrail in tooling or services that accept user-defined SQL, ensuring that only safe, read-only queries reach the Meergo data warehouse.
