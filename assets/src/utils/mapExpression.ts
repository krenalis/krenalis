type ExpressionFragment = {
	func: {
		name: string; // name of the function
		parameter: number; // zero-based index of the function parameter
	} | null;
	type: 'Property' | 'Function' | 'String' | 'Number' | null; // type of element
	pos: {
		start: number; // start position of the element in the expression
		end: number; // end position of the element in the expression
	} | null;
};

// parseMapExpression parses the provided map expression and returns an object
// describing the element at the specified index.
//
// If the index is within a function parameter, func contains the function name
// and the parameter index (starting from 0); otherwise, func is null.
//
// If the index refers to a property, function name, string, or number, type is
// either 'Property', 'Function', 'String', or 'Number' and pos provides the
// position of that element in the expression. If the index does not refer to a
// specific property, function, string, or number, type and pos are both null.
//
// It returns null if the provided expression is invalid.
const parseMapExpression = (expr: string, index: number): ExpressionFragment | null => {
	const cursor: ExpressionFragment = {
		func: null,
		type: null,
		pos: null,
	};

	let state: 'number' | 'path' | 'string' | null = null;
	let stack: Array<{ name: string; parameter: number }> = [];
	let start = 0;
	let end = 0;
	let quote: string | null = null;

	const checkFunction = (i: number) => {
		if (i === index && stack.length > 0) {
			cursor.func = { ...stack[stack.length - 1] };
		}
	};

	const parseBracketAccess = (startIndex: number): { nextIndex: number; start: number; end: number } | null => {
		let j = startIndex + 1;
		while (j < expr.length && isSpace(expr[j])) {
			checkFunction(j);
			j++;
		}
		if (j >= expr.length || !isQuote(expr[j])) {
			return null;
		}
		const bracketQuote = expr[j];
		const propertyStart = j + 1;
		j++;
		for (; j < expr.length; j++) {
			checkFunction(j);
			const ch = expr[j];
			if (isBackslash(ch)) {
				j++;
				if (j >= expr.length) {
					return null;
				}
				checkFunction(j);
				continue;
			}
			if (ch === bracketQuote) {
				const propertyEnd = j;
				j++;
				while (j < expr.length && isSpace(expr[j])) {
					checkFunction(j);
					j++;
				}
				if (j >= expr.length) {
					return null;
				}
				checkFunction(j);
				if (!isCloseBracket(expr[j])) {
					return null;
				}
				return { nextIndex: j, start: propertyStart, end: propertyEnd };
			}
		}
		return null;
	};

	let i: number;
	for (i = 0; i < expr.length; i++) {
		checkFunction(i);
		let c = expr[i];
		switch (state) {
			case null:
				if (isSpace(c)) {
					continue;
				}
				if (isAlfa(c)) {
					state = 'path';
					start = i;
					continue;
				}
				if (isQuote(c)) {
					state = 'string';
					quote = c;
					start = i;
					continue;
				}
				if (isNumber(c)) {
					state = 'number';
					start = i;
					continue;
				}
				if (isComma(c)) {
					if (stack.length === 0) {
						return null;
					}
					stack[stack.length - 1].parameter += 1;
					continue;
				}
				if (isCloseParenthesis(c)) {
					if (stack.length === 0) {
						return null;
					}
					stack.pop();
					continue;
				}
				return null;
			case 'number':
				if (isNumber(c)) {
					continue;
				}
				if (isDot(c)) {
					if (expr.slice(start, i).includes('.')) {
						return null;
					}
					continue;
				}
				end = i;
				if (start <= index && index <= end) {
					cursor.type = 'Number';
					cursor.pos = { start: start, end: end };
				}
				if (isSpace(c)) {
					state = null;
					continue;
				}
				if (isComma(c)) {
					if (stack.length === 0) {
						return null;
					}
					stack[stack.length - 1].parameter += 1;
					continue;
				}
				if (isCloseParenthesis(c)) {
					if (stack.length === 0) {
						return null;
					}
					stack.pop();
					state = null;
					continue;
				}
				if (isQuote(c)) {
					state = 'string';
					quote = c;
					start = i;
					continue;
				}
				return null;
			case 'path':
				if (isAlfaNumeric(c)) {
					continue;
				}
				if (isOpenBracket(c)) {
					if (start <= index && index <= i) {
						cursor.type = 'Property';
						cursor.pos = { start: start, end: i };
					}
					const bracket = parseBracketAccess(i);
					if (bracket == null) {
						return null;
					}
					if (bracket.start <= index && index <= bracket.end) {
						cursor.type = 'Property';
						cursor.pos = { start: bracket.start, end: bracket.end };
					}
					checkFunction(bracket.nextIndex);
					i = bracket.nextIndex;
					continue;
				}
				if (isDot(c)) {
					i++;
					if (i === expr.length || !isAlfa(expr[i])) {
						return null;
					}
					checkFunction(i);
					continue;
				}
				state = null;
				end = i;
				if (start <= index && index <= end) {
					cursor.type = 'Property';
					cursor.pos = { start: start, end: end };
				}
				while (i < expr.length && isSpace(expr[i])) {
					i++;
					checkFunction(i);
				}
				if (i === expr.length) {
					continue;
				}
				if (!isOpenParenthesis(expr[i])) {
					i--;
					continue;
				}
				const name = expr.slice(start, end);
				if (name.includes('.')) {
					return null;
				}
				if (start <= index && index <= i) {
					cursor.type = 'Function';
					cursor.pos = { start: start, end: end };
				}
				stack.push({ name: name, parameter: 0 });
				continue;
			case 'string':
				if (isBackslash(c)) {
					i++;
					checkFunction(i);
					continue;
				}
				if (c === quote) {
					end = i;
					if (start <= index && index <= end) {
						cursor.type = 'String';
						cursor.pos = { start: start, end: end };
					}
					state = null;
					quote = null;
				}
		}
	}
	if (stack.length > 0 || state === 'string') {
		return null;
	}
	if (state === 'path') {
		end = i;
		if (start <= index && index <= end) {
			if (cursor.pos == null || index < cursor.pos.start || index > cursor.pos.end) {
				cursor.type = 'Property';
				cursor.pos = { start: start, end: end };
			}
		}
	}

	return cursor;
};

function isAlfa(c: string): boolean {
	return ('a' <= c && c <= 'z') || ('A' <= c && c <= 'Z') || c === '_';
}

function isAlfaNumeric(c: string): boolean {
	return isAlfa(c) || isNumber(c);
}

function isBackslash(c: string): boolean {
	return c === '\\';
}

function isCloseBracket(c: string): boolean {
	return c === ']';
}

function isCloseParenthesis(c: string): boolean {
	return c === ')';
}

function isComma(c: string): boolean {
	return c === ',';
}

function isDot(c: string): boolean {
	return c === '.';
}

function isNumber(c: string): boolean {
	return '0' <= c && c <= '9';
}

function isOpenBracket(c: string): boolean {
	return c === '[';
}

function isOpenParenthesis(c: string): boolean {
	return c === '(';
}

function isQuote(c: string): boolean {
	return c === '"' || c === "'";
}

function isSpace(c: string): boolean {
	return c === ' ' || c === '\t' || c === '\n' || c === '\r';
}

// splitMapKey splits a key of a 'map' function, in a valid map expression, from
// the rest of the expression. For example, given `'a', 5, 'b', false`, it
// returns ["a", " 5, 'b', false"].
function splitMapKey(s: string): [string, string] {
	let key = '';
	let quote: string | null = null;
	for (let i = 0; ; i++) {
		const c = s[i];
		if (quote == null) {
			if (c === '"' || c === "'") {
				quote = c;
				continue;
			}
			if (c === ',') {
				return [key, s.slice(i + 1)];
			}
			continue;
		}
		if (c === quote) {
			quote = null;
			continue;
		}
		if (c === '\\') {
			i++;
		}
		key += s[i];
	}
}

// splitMapValue splits a value of a 'map' function, in a valid map expression,
// from the rest of the expression. For example: given ` 5, 'b', false`, it
// returns [" 5", " 'b', false"], and given ` false`, it returns [" false", ""].
function splitMapValue(s: string): [string, string] {
	let depth = 0;
	let quote: string | null = null;
	for (let i = 0; i < s.length; i++) {
		const c = s[i];
		if (quote) {
			if (c === '\\') {
				i++;
			} else if (c === quote) {
				quote = null;
			}
			continue;
		}
		if (c === '"' || c === "'") {
			quote = c;
			continue;
		}
		if (c === '(') {
			depth++;
			continue;
		}
		if (c === ')') {
			depth--;
			continue;
		}
		if (c === ',' && depth === 0) {
			return [s.slice(0, i), s.slice(i + 1)];
		}
	}
	return [s, ''];
}

// mapExpressionArguments returns an array of arguments if expr is a sole
// call to map(...). Even indexes are keys, odd indexes are the corresponding
// values. It returns null when the input does not match.
const mapExpressionArguments = (expr: string): Map<string, string> | null => {
	const m = expr.trim().match(/^map\s*\(([\s\S]*)\)$/);
	if (!m) {
		return null;
	}
	let s = m[1].trim();
	if (s === '') {
		return new Map();
	}
	let args = new Map();
	let k: string, v: string;
	while (s != '') {
		[k, s] = splitMapKey(s);
		[v, s] = splitMapValue(s);
		args.set(k, v.trim());
	}
	return args;
};

// buildMapExpression builds a map(...) string from an argument list.
const buildMapExpression = (args: Map<string, string>): string => {
	const serialized = Array.from(args.entries()).map(([k, v]) => JSON.stringify(k) + ',' + v);
	return `map(${serialized.join(',')})`;
};

// Simple self-contained tests using console.assert.
// @ts-ignore: TS6133
function runTests(): void {
	const mapsAreEqual = (map1: Map<string, string>, map2: Map<string, string>) => {
		if (map1.size !== map2.size) {
			return false;
		}
		const entries1 = [...map1];
		const entries2 = [...map2];
		return entries1.every(([k, v], i) => {
			const [k2, v2] = entries2[i];
			return k === k2 && v === v2;
		});
	};
	console.assert(mapExpressionArguments('foo') === null);
	console.assert(mapExpressionArguments('map() "boo"') === null);
	const emptyArgs = mapExpressionArguments('map()');
	console.assert(emptyArgs != null && mapsAreEqual(emptyArgs, new Map()));
	const parsedArgs = mapExpressionArguments('map("a", 5, "b", false)');
	console.assert(
		parsedArgs != null &&
			mapsAreEqual(
				parsedArgs,
				new Map([
					['a', '5'],
					['b', 'false'],
				]),
			),
	);
	const singleQuoteArgs = mapExpressionArguments("map('c', \"'s'\")");
	console.assert(singleQuoteArgs != null && mapsAreEqual(singleQuoteArgs, new Map([['c', '"\'s\'"']])));
	const m = new Map<string, string>([
		['k1', 'foo'],
		['k2', "if(a,b,c) 'boo'"],
		['k3', 'map("k", "v")'],
	]);
	const bracketDoubleQuote = parseMapExpression('value["key"]', 8);
	console.assert(bracketDoubleQuote != null);
	console.assert(bracketDoubleQuote?.type === 'Property');
	console.assert(bracketDoubleQuote?.pos?.start === 7 && bracketDoubleQuote?.pos?.end === 10);
	const bracketSingleQuote = parseMapExpression("value['name']", 9);
	console.assert(bracketSingleQuote != null);
	console.assert(bracketSingleQuote?.type === 'Property');
	console.assert(bracketSingleQuote?.pos?.start === 7 && bracketSingleQuote?.pos?.end === 11);
	console.assert(parseMapExpression('value["key"', 8) === null);
	console.assert(buildMapExpression(m) === 'map("k1",foo,"k2",if(a,b,c) \'boo\',"k3",map("k", "v"))');
}

export { parseMapExpression, mapExpressionArguments, buildMapExpression };
export type { ExpressionFragment };
