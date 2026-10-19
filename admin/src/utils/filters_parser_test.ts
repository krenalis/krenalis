/**
 * filters_parser_test.ts
 *
 * Test suite for the `parseFilter` function in filters_parser.ts.
 *
 * To run:
 *
 *   1. cd into the directory containing this file
 *   2. Run the test:
 *        npx tsx filters_parser_test.ts
 */

import { parseFilter } from './filters_parser';
import { serializeFilter } from './filters';

const tests = [
	// ✅ Valid cases (single values, no parentheses)
	{
		name: 'Single simple condition',
		input: 'user.name is "Alice"',
		expectError: false,
	},
	{
		name: 'Boolean values with AND',
		input: 'active is true and verified is false',
		expectError: false,
	},
	{
		name: 'Scientific notation number',
		input: 'score is 1.23e+5',
		expectError: false,
	},
	{
		name: 'Extra spaces between tokens',
		input: 'user.age    is   42',
		expectError: false,
	},
	{
		name: 'Single-quoted value',
		input: "props.key is 'value'",
		expectError: false,
	},

	// ✅ Valid cases (multiple values, parentheses required)
	{
		name: 'List of values with parentheses (is one of)',
		input: 'color is one of ("red", "green", "blue")',
		expectError: false,
	},
	{
		name: 'Number with parentheses (is between)',
		input: 'age is between (18, 65)',
		expectError: false,
	},

	// ✅ Complex valid cases
	{
		name: 'Multiple AND conditions with correct syntax',
		input: 'name is "Alice" and age is greater than 30 and active is true',
		expectError: false,
	},
	{
		name: 'Multiple OR conditions with correct syntax',
		input: 'country is "Italy" or country is "Spain" or country is "France"',
		expectError: false,
	},
	{
		name: 'AND with nested properties and quoted values',
		input: 'properties.client_id is "abc" and metadata.referrer contains "google"',
		expectError: false,
	},
	{
		name: 'OR condition with is null and string match',
		input: 'deleted_at is null or status is "archived"',
		expectError: false,
	},
	{
		name: 'Mixed values and operators with AND',
		input: 'rating is greater than 3.5 and review_count is greater than or equal to 100 and featured is true',
		expectError: false,
	},
	{
		name: 'OR chain with boolean, string, and number',
		input: 'flag is false or user.role is "guest" or attempts is less than 3',
		expectError: false,
	},
	{
		name: 'Nested AND and OR groups',
		input: 'status is "active" and (country is "Italy" or country is "Spain")',
		expectError: false,
	},
	{
		name: 'Nested groups at maximum depth',
		input: 'a is 1 and (b is 2 or (c is 3 and (d is 4 or e is 5)))',
		expectError: false,
	},
	{
		name: 'Fully parenthesized filter at maximum depth',
		input: '((((a is 1))))',
		expectError: false,
	},
	{
		name: 'Maximum rule count',
		input: Array.from({ length: 100 }, (_, i) => `property${i} is ${i}`).join(' and '),
		expectError: false,
	},
	{
		name: 'Fully parenthesized filter at maximum rule count',
		input: `(${Array.from({ length: 100 }, (_, i) => `property${i} is ${i}`).join(' and ')})`,
		expectError: false,
	},
	{
		name: 'Unicode emoji and accents',
		input: 'comment contains "👍 café \u2764"',
		expectError: false,
	},
	// ❌ Invalid: empty filters
	{
		name: 'Empty filter',
		input: '',
		expectError: true,
	},
	{
		name: 'Whitespace-only filter',
		input: ' \n\t',
		expectError: true,
	},

	// ❌ Invalid: single value with parentheses
	{
		name: 'Single string value with parentheses',
		input: 'user.name is ("Alice")',
		expectError: true,
	},
	{
		name: 'Single numeric value with parentheses',
		input: 'score is (42)',
		expectError: true,
	},
	{
		name: 'Boolean value with parentheses',
		input: 'active is (true)',
		expectError: true,
	},

	// ❌ Invalid: multiple values without parentheses
	{
		name: 'Multiple values without parentheses (is one of)',
		input: 'color is one of "red", "green"',
		expectError: true,
	},
	{
		name: 'Multiple values without parentheses (is between)',
		input: 'age is between 18, 65',
		expectError: true,
	},

	// ❌ Invalid syntax and complex error cases
	{
		name: 'Unclosed string',
		input: 'name is "Alice',
		expectError: true,
	},
	{
		name: 'Unknown operator',
		input: 'age is not less than 30',
		expectError: true,
	},
	{
		name: 'Missing value after operator',
		input: 'score is',
		expectError: true,
	},
	{
		name: 'is between with only one value inside parentheses',
		input: 'age is between (18)',
		expectError: true,
	},
	{
		name: 'Mismatched parentheses',
		input: 'status is one of ("a", "b"',
		expectError: true,
	},
	{
		name: 'Unclosed filter group',
		input: 'a is 1 and (b is 2 or c is 3',
		expectError: true,
	},
	{
		name: 'Filter nesting exceeds maximum depth',
		input: 'a is 1 and (b is 2 or (c is 3 and (d is 4 or (e is 5 and f is 6))))',
		expectError: true,
	},
	{
		name: 'Fully parenthesized filter exceeds maximum depth',
		input: '(((((a is 1)))))',
		expectError: true,
	},
	{
		name: 'Filter exceeds maximum rule count',
		input: Array.from({ length: 101 }, (_, i) => `property${i} is ${i}`).join(' and '),
		expectError: true,
	},
	{
		name: 'Fully parenthesized filter exceeds maximum rule count',
		input: `(${Array.from({ length: 101 }, (_, i) => `property${i} is ${i}`).join(' and ')})`,
		expectError: true,
	},
	{
		name: 'Consecutive dots in property',
		input: 'user..name is "Alice"',
		expectError: true,
	},
	{
		name: 'Bracket notation with single quotes',
		input: "props['key'] is 'value'",
		expectError: true,
	},
	{
		name: 'Bracket notation with double quotes',
		input: 'metadata["key"] contains "something"',
		expectError: true,
	},
	{
		name: 'Mixed logical connectors (and/or)',
		input: 'a is 1 and b is 2 or c is 3',
		expectError: true,
	},
	{
		name: 'Trailing AND connector',
		input: 'a is 1 and',
		expectError: true,
	},
	{
		name: 'Malformed unicode escape',
		input: 'bio contains "hello \\u26"',
		expectError: true,
	},
	{
		name: 'Single operator at the beginning',
		input: 'and age is 30',
		expectError: true,
	},
	{
		name: 'Single operator at the end',
		input: 'status is "active" or',
		expectError: true,
	},
	{
		name: 'Extra comma in value list',
		input: 'color is one of ("red", "green",)',
		expectError: true,
	},
	{
		name: 'is between with three values',
		input: 'price is between (10, 20, 30)',
		expectError: true,
	},
	{
		name: 'Invalid escaped character in string',
		input: 'name is "bad\\xescape"',
		expectError: true,
	},
	{
		name: 'Unescaped quote inside string',
		input: 'title contains "He said "yes""',
		expectError: true,
	},
	{
		name: 'Empty parentheses in values',
		input: 'type is one of ()',
		expectError: true,
	},
	{
		name: 'Valid condition followed by junk',
		input: 'score is 10 abcdef',
		expectError: true,
	},
	{
		name: 'Valid condition followed by unexpected operator',
		input: 'score is 10 and or status is "ok"',
		expectError: true,
	},
];

let passed = 0;
let failed = 0;

console.log(`\n--- Running parseFilter test suite ---\n`);

for (const test of tests) {
	try {
		const result = parseFilter(test.input);
		if (test.expectError) {
			console.error(`❌ [FAIL] ${test.name} → Expected error, but got result`, result);
			failed++;
		} else {
			console.log(`✅ [PASS] ${test.name}`);
			passed++;
		}
	} catch (e: any) {
		if (test.expectError) {
			console.log(`✅ [PASS] ${test.name} (error: ${e.message})`);
			passed++;
		} else {
			console.error(`❌ [FAIL] ${test.name} → Unexpected error: ${e.message}`);
			failed++;
		}
	}
}

const roundTripExpressions = [
	{
		name: 'Nested filter',
		expression: 'status is "active" and (country is "Italy" or country is "Spain")',
	},
	...['true', 'false'].map((value) => ({
		name: `Filter with "${value}" as a string value`,
		expression: `status is "${value}"`,
	})),
	...['0', '-1', '+.5', '1.', '1.23e+5', '0x10', 'Infinity', ' 1', '1 '].map((value) => ({
		name: `Filter with ${JSON.stringify(value)} as a value`,
		expression: `code is ${JSON.stringify(value)}`,
	})),
	{
		name: 'Filter list containing numeric-looking string values',
		expression: 'code is one of ("0x10", "Infinity", " 1", "1.23e+5")',
	},
	...['is one of', 'is not one of'].map((operator) => ({
		name: `Filter using "${operator}" with one value`,
		expression: `status ${operator} ("active")`,
	})),
	...['is true', 'is false', 'is empty', 'is not empty', 'is null', 'is not null', 'exists', 'does not exist'].map(
		(operator) => ({
			name: `Nested filter ending in a condition using "${operator}"`,
			expression: `status is "active" and (property ${operator})`,
		}),
	),
];
for (const roundTrip of roundTripExpressions) {
	const filter = parseFilter(roundTrip.expression);
	for (const formatted of [false, true]) {
		const serialized = serializeFilter(filter, formatted);
		const parsed = parseFilter(serialized);
		if (JSON.stringify(parsed) === JSON.stringify(filter)) {
			console.log(`✅ [PASS] ${roundTrip.name} ${formatted ? 'formatted' : 'compact'} round trip`);
			passed++;
		} else {
			console.error(`❌ [FAIL] ${roundTrip.name} round trip → ${serialized}`);
			failed++;
		}
	}
}

console.log(`\nTotal: ${passed} passed, ${failed} failed.`);
if (failed > 0) process.exitCode = 1;
