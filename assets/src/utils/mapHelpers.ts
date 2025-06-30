// parseMapExpression returns an array of arguments if expr is a sole call to
// map(...). Even indexes are keys, odd indexes are the corresponding values.
// It returns null when the input does not match.
function parseMapExpression(expr: string): string[] | null {
	const m = expr.trim().match(/^map\s*\((.*)\)$/);
	if (!m) {
		return null;
	}
	const inside = m[1].trim();
	if (inside === '') {
		return [];
	}
	const args: string[] = [];
	let current = '';
	let depth = 0;
	let quote: string | null = null;
	for (let i = 0; i < inside.length; i++) {
		const c = inside[i];
		if (quote) {
			current += c;
			if (c === '\\') {
				// Skip escaped character
				if (i + 1 < inside.length) {
					current += inside[i + 1];
					i++;
				}
				continue;
			}
			if (c === quote) {
				quote = null;
			}
			continue;
		}
		if (c === '"' || c === "'") {
			quote = c;
			current += c;
			continue;
		}
		if (c === '(') {
			depth++;
			current += c;
			continue;
		}
		if (c === ')') {
			depth--;
			current += c;
			continue;
		}
		if (c === ',' && depth === 0) {
			args.push(current.trim());
			current = '';
			continue;
		}
		current += c;
	}
	if (current.trim() !== '' || inside.endsWith(',')) {
		args.push(current.trim());
	}
	return args;
}

// buildMapExpression builds a map(...) string from an argument list.
function buildMapExpression(pairs: string[]): string {
	const serialized = pairs.map((p) => JSON.stringify(p));
	return `map(${serialized.join(',')})`;
}

// Simple self-contained tests using console.assert.
// @ts-ignore: TS6133
function runTests(): void {
	console.assert(parseMapExpression('foo') === null);
	console.assert(parseMapExpression('map() "boo"') === null);
	console.assert(JSON.stringify(parseMapExpression('map()')) === '[]');
	console.assert(
		JSON.stringify(parseMapExpression("map('a', 5, 'b', false)")!) === JSON.stringify(["'a'", '5', "'b'", 'false']),
	);
	console.assert(JSON.stringify(parseMapExpression("map('c', \"'s'\")")!) === JSON.stringify(["'c'", '"\'s\'"']));
	console.assert(
		buildMapExpression(['k1', 'foo', 'k2', "if(a,b,c) 'boo'"]) === 'map("k1","foo","k2","if(a,b,c) \'boo\'")',
	);
}

export { parseMapExpression, buildMapExpression };
