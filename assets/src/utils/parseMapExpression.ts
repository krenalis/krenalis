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
			cursor.type = 'Property';
			cursor.pos = { start: start, end: end };
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

function isOpenParenthesis(c: string): boolean {
	return c === '(';
}

function isQuote(c: string): boolean {
	return c === '"' || c === "'";
}

function isSpace(c: string): boolean {
	return c === ' ' || c === '\t' || c === '\n' || c === '\r';
}

export { parseMapExpression, ExpressionFragment };
