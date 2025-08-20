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
		name: 'Single quote string',
		input: "props['key'] is 'value'",
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
		name: 'AND with escaped keys and quoted values',
		input: 'properties["client id"] is "abc" and metadata["referrer"] contains "google"',
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
		name: 'Unicode emoji and accents',
		input: 'comment contains "👍 café \u2764"',
		expectError: false,
	},
	{
		name: 'Property key with escaped quotes',
		input: 'metadata["key with \\"quotes\\""] contains "something"',
		expectError: false,
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
		name: 'Consecutive dots in property',
		input: 'user..name is "Alice"',
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
		name: 'Brackets not closed in property key',
		input: 'meta["client id" is "abc"',
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

console.log(`\nTotal: ${passed} passed, ${failed} failed.`);
