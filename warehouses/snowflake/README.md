# JSON number transport

The Snowflake warehouse and connector preserve numeric tokens in the JSON text
returned by the server. This applies to scalar VARIANT values and to numbers
inside VARIANT, OBJECT and ARRAY, including nested documents.
The distinction concerns Snowflake's physical numeric values, not a type
inferred from the event catalog.

- INTEGER/DECIMAL digits remain exact. JSON decoding must not route all numbers
  through `float64` or otherwise round the received tokens.
- Finite FLOAT/DOUBLE values retain the server's representation, which can
  have fewer digits than the stored binary64 value needs for a round trip.
  No binary64 identity guarantee is made for JSON output. The reader cannot
  identify every token that has lost precision or recover the missing digits.
- This limit does not change native FLOAT columns, existing conversions of
  typed arrays/maps, SQL NULL versus JSON null, or the treatment of invalid
  JSON and non-finite values. Errors do not become values or semantic UNKNOWN.

Previous remote probes observed the DOUBLE after 100 as
`1.000000000000000e+02` in JSON, although its canonical decimal reading is
`100.00000000000001`. Sixteen significant digits were observed, not established
as a universal server formatting guarantee. The reader preserves whatever
valid numeric text is actually received, including more precise text and
numbers outside the binary64 range.

Record JSON comparisons keep their approved canonical DOUBLE reading,
`strconv.FormatFloat(d, 'g', -1, 64)`, with exact decimal operands. Governed
monetary readings retain their separate binary-exact Decimal(18,4) contract.
A profile can consequently match `value > 100` while the returned JSON shows
100. Clients must not use displayed JSON to recompute warehouse membership.

Both query paths use this output contract. There are no corrective queries,
result dictionaries, query wrappers or modifications to the user's SQL.
Rows, ordering and columns retain their existing behavior.

New JSON writes separately expand scientific number tokens whose exact values
fit NUMBER precision 38 and scale 37. Other document bytes and values outside
that domain retain their existing loading behavior. This reduces avoidable
DOUBLE storage in new data; it does not recover historical precision or
constrain results produced by arbitrary SQL expressions.

Database import cursors are temporal. The current validation accepts datetime,
date and formatted strings; JSON cursors must contain a string, not a number.
Accepting server JSON text does not introduce FLOAT cursors or alter cursor
ordering, parsing or persistence.

Local regressions cover numeric text preservation and the connector's single,
unchanged query. They simulate driver results and do not certify server
serialization or physical storage. Remaining remote checks include the two
real Merge/read paths, NUMBER borders, nested containers, native FLOAT
extraction versus JSON display, Record decisions and relevant error/resource
cases. The Unified Snowflake gate remains closed until its complete campaign
passes; this accepted JSON output limit does not open it. DECFLOAT and other
additional native types remain outside this increment.
