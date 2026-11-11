
Snowflake is a cloud-based data warehousing platform for storing and analyzing large volumes of data. It provides a scalable architecture for real-time analytics, making it ideal for business intelligence applications.

## What can you do with this?

This connector lets you read users from a Snowflake database, store them in your data warehouse, and unify them as profiles within Krenalis.

## What does it require?

* A Snowflake database to read the users from.

## JSON number precision

Krenalis preserves the JSON representation returned by Snowflake for VARIANT,
OBJECT and ARRAY values. Integer and decimal digits are preserved. Snowflake
can round floating-point numbers in that representation, so reading them back
does not guarantee the same binary floating-point value. A number used by a
SQL comparison can therefore have more precision than the number visible in
the returned JSON.

> Snowflake is a trademark of Snowflake Inc.
> This connector is not affiliated with or endorsed by Snowflake Inc.
