import { FILTER_OPERATORS } from '../lib/core/action';
import { FilterLogical, FilterOperator, Filter, FilterCondition } from '../lib/api/types/action';

const SORTED_OPERATORS = [...FILTER_OPERATORS].sort((a, b) => b.length - a.length);

const UNARY_OPERATORS: FilterOperator[] = [
	'is',
	'is not',
	'contains',
	'does not contain',
	'starts with',
	'ends with',
	'is before',
	'is after',
	'is on or before',
	'is on or after',
	'is greater than',
	'is less than',
	'is greater than or equal to',
	'is less than or equal to',
];

// isWhitespace reports whether the character is a whitespace character.
const isWhitespace = (c: string): boolean => c === ' ' || c === '\t' || c === '\n' || c === '\r';

// isPropStart reports whether the character can start a property name.
function isPropStart(c: string): boolean {
	const code = c.charCodeAt(0);
	return (code >= 65 && code <= 90) || (code >= 97 && code <= 122) || c === '_';
}

// isPropChar reports whether the character is valid inside a property name.
function isPropChar(c: string): boolean {
	const code = c.charCodeAt(0);
	return isPropStart(c) || (code >= 48 && code <= 57);
}

// skipSpaces advances the index past any whitespace characters.
function skipSpaces(s: string, i: number): number {
	while (i < s.length && isWhitespace(s[i])) i++;
	return i;
}

// parseNumber parses a numeric value.
function parseNumber(s: string, i: number): [string, number] {
	const start = i;
	while (i < s.length && /[0-9eE.+\-]/.test(s[i])) i++;
	const str = s.slice(start, i);
	const num = Number(str);
	if (!isFinite(num)) throw new Error(`Invalid number: ${str}`);
	return [str, i];
}

// parseString parses a quoted string, supporting JS-style escape sequences.
function parseString(s: string, i: number): [string, number] {
	const quote = s[i];
	if (quote !== '"' && quote !== "'") throw new Error('Invalid string delimiter');
	i++;
	let out = '';
	while (i < s.length) {
		const c = s[i];
		if (c === quote) return [out, i + 1];
		if (c === '\\') {
			i++;
			if (i >= s.length) throw new Error('Unterminated string');
			const esc = s[i];
			switch (esc) {
				case 'n':
					out += '\n';
					break;
				case 'r':
					out += '\r';
					break;
				case 't':
					out += '\t';
					break;
				case 'b':
					out += '\b';
					break;
				case 'f':
					out += '\f';
					break;
				case '\\':
					out += '\\';
					break;
				case '"':
					out += '"';
					break;
				case "'":
					out += "'";
					break;
				case 'u':
				case 'U': {
					const len = esc === 'u' ? 4 : 8;
					const hex = s.slice(i + 1, i + 1 + len);
					if (hex.length < len || /[^0-9a-fA-F]/.test(hex)) throw new Error('Invalid Unicode escape');
					const codePoint = parseInt(hex, 16);
					if (codePoint === 0 || (codePoint >= 0xd800 && codePoint < 0xe000) || codePoint > 0x10ffff) {
						throw new Error(`Invalid Unicode code point: U+${hex}`);
					}
					out += String.fromCodePoint(codePoint);
					i += len;
					break;
				}
				default:
					throw new Error('Invalid escape character');
			}
		} else {
			out += c;
		}
		i++;
	}
	throw new Error('Unterminated string');
}

// parseValue parses a single value (string, number, or boolean).
function parseValue(s: string, i: number): [string, number] {
	i = skipSpaces(s, i);
	const c = s[i];

	if (s.startsWith('true', i)) return ['true', i + 4];
	if (s.startsWith('false', i)) return ['false', i + 5];
	if (c === '"' || c === "'") return parseString(s, i);
	if ((c >= '0' && c <= '9') || c === '-' || c === '+' || c === '.') return parseNumber(s, i);

	throw new Error(`Invalid value at position ${i}`);
}

// parseValues parses one or more values, with or without parentheses for lists.
function parseValues(s: string, i: number, expectMultiple: boolean): [string[], number] {
	i = skipSpaces(s, i);
	if (s[i] === '(') {
		if (!expectMultiple) {
			throw new Error('Parentheses are not allowed when only one value is expected');
		}
		i++; // skip '('
		const values: string[] = [];
		while (true) {
			i = skipSpaces(s, i);
			const [val, ni] = parseValue(s, i);
			values.push(val);
			i = skipSpaces(s, ni);
			if (s[i] === ',') {
				i++;
				continue;
			}
			if (s[i] === ')') return [values, i + 1];
			throw new Error('Expected comma or closing parenthesis');
		}
	}
	if (expectMultiple) {
		throw new Error('Multiple values must be enclosed in parentheses');
	}
	const [val, ni] = parseValue(s, i);
	return [[val], ni];
}

// parseBracketString parses a map-style property access like ["key"].
function parseBracketString(s: string, i: number): [string, number] {
	if (s[i] !== '[') throw new Error('Expected "[" for map access');
	i++;
	const quote = s[i];
	if (quote !== '"' && quote !== "'") throw new Error('Expected quoted string in map access');
	const [str, end] = parseString(s, i);
	if (s[end] !== ']') throw new Error('Expected closing bracket ] in map access');
	return [`[${quote}${str}${quote}]`, end + 1];
}

// parseProperty parses a property path, including dot notation and brackets.
function parseProperty(s: string, i: number): [string, number] {
	let prop = '';
	let lastCharDot = false;
	while (i < s.length) {
		if (isPropStart(s[i])) {
			const begin = i;
			while (i < s.length && isPropChar(s[i])) i++;
			prop += s.slice(begin, i);
			lastCharDot = false;
			continue;
		}
		if (s[i] === '.') {
			if (lastCharDot) throw new Error('Invalid property path: consecutive dots');
			prop += '.';
			i++;
			lastCharDot = true;
			continue;
		}
		if (s[i] === '[') {
			const [bracketed, ni] = parseBracketString(s, i);
			prop += bracketed;
			i = ni;
			lastCharDot = false;
			continue;
		}
		break;
	}
	if (!prop || prop.endsWith('.')) throw new Error('Invalid property path');
	return [prop, i];
}

// parseOperator parses an operator string from the predefined list.
function parseOperator(s: string, i: number): [FilterOperator, number] {
	const len = s.length;
	for (const op of SORTED_OPERATORS) {
		const end = i + op.length;
		if (end > len) continue;
		if (s.slice(i, i + op.length) !== op) continue;
		const nextChar = s[end];
		if (end === len || isWhitespace(nextChar) || nextChar === '(' || nextChar === '"' || nextChar === "'") {
			return [op as FilterOperator, end];
		}
	}
	throw new Error('Unknown operator');
}

// parseFilter parses a user-readable filter expression into a Filter object.
//
// It supports property paths, string/number/boolean values, and a rich set of operators.
// It returns a structured Filter with conditions and a logical connector (and/or).
// Syntax errors are reported with informative messages.
export function parseFilter(s: string): Filter {
	let i = 0;
	const conditions: FilterCondition[] = [];
	let logical: FilterLogical | null = null;
	let conditionCount = 0;

	while (i < s.length) {
		i = skipSpaces(s, i);
		const [property, ni1] = parseProperty(s, i);
		i = skipSpaces(s, ni1);
		const [operator, ni2] = parseOperator(s, i);
		i = skipSpaces(s, ni2);

		let values: string[] | null = null;
		if (
			!operator.endsWith('null') &&
			!operator.endsWith('exist') &&
			operator !== 'is true' &&
			operator !== 'is false'
		) {
			const expectMultiple =
				operator === 'is one of' ||
				operator === 'is not one of' ||
				operator === 'is between' ||
				operator === 'is not between';
			const [vals, ni3] = parseValues(s, i, expectMultiple);
			values = vals;
			i = ni3;

			const n = values.length;
			if ((operator === 'is between' || operator === 'is not between') && n !== 2)
				throw new Error(`Operator '${operator}' requires exactly 2 values`);
			if ((operator === 'is one of' || operator === 'is not one of') && n < 1)
				throw new Error(`Operator '${operator}' requires at least 1 value`);
			if (UNARY_OPERATORS.includes(operator) && n !== 1)
				throw new Error(`Operator '${operator}' requires exactly 1 value`);
		}

		conditions.push({ property, operator, values });
		conditionCount++;

		i = skipSpaces(s, i);
		if (i >= s.length) return { logical: logical || 'and', conditions };

		const next = s.startsWith('and', i) ? 'and' : s.startsWith('or', i) ? 'or' : null;
		if (!next) throw new Error("Expected logical connector 'and' or 'or'");
		if (logical && logical !== next) throw new Error('Mixed logical connectors are not allowed');
		logical = next as FilterLogical;
		i += next.length;

		i = skipSpaces(s, i);
		if (i >= s.length) throw new Error('Trailing logical connector without following condition');
	}

	if (conditionCount > 1 && !logical) throw new Error('Missing logical connector (and/or) between conditions');
	return { logical: logical || 'and', conditions };
}
