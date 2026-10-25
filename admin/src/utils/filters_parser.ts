import { FILTER_OPERATORS, MAX_FILTER_DEPTH, MAX_FILTER_RULE_COUNT, isFilterGroup, isUnaryOperator } from './filters';
import {
	FilterLogical,
	FilterOperator,
	EditorFilter,
	EditorFilterCondition,
	EditorFilterRule,
} from '../lib/api/types/pipeline';
import { ObjectType } from '../lib/api/types/types';
import {
	isValidPropertyPath,
	quoteFilterText,
	formatString,
	filterProperty,
	parseFilterOperand,
	filterValueKind,
} from './filters';

interface FilterToken {
	text: string;
	isQuoted: boolean;
}

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
	if (!/^[+-]?(?:\d+(?:\.\d*)?|\.\d+)(?:[eE][+-]?\d+)?$/.test(str)) throw new Error(`Invalid number: ${str}`);
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

// parseValue parses a value token while retaining its quoting.
function parseValue(s: string, i: number): [FilterToken, number] {
	i = skipSpaces(s, i);
	const c = s[i];

	if (s.startsWith('true', i)) return [{ text: 'true', isQuoted: false }, i + 4];
	if (s.startsWith('false', i)) return [{ text: 'false', isQuoted: false }, i + 5];
	if (s.startsWith('null', i)) return [{ text: 'null', isQuoted: false }, i + 4];
	if (c === '"' || c === "'") {
		const [text, end] = parseString(s, i);
		return [{ text, isQuoted: true }, end];
	}
	if ((c >= '0' && c <= '9') || c === '-' || c === '+' || c === '.') {
		const [text, end] = parseNumber(s, i);
		return [{ text, isQuoted: false }, end];
	}

	throw new Error(`Invalid value at position ${i}`);
}

// parseValues parses one or more values, with or without parentheses for lists.
function parseValues(s: string, i: number, expectMultiple: boolean): [FilterToken[], number] {
	i = skipSpaces(s, i);
	if (s[i] === '(') {
		if (!expectMultiple) {
			throw new Error('Parentheses are not allowed when only one value is expected');
		}
		i++; // skip '('
		const values: FilterToken[] = [];
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

// parseProperty parses a filter property path. Unlike transformation
// expressions, filter property paths support only dot notation; bracket
// notation for map and JSON keys is intentionally rejected.
function parseProperty(s: string, i: number): [string, number] {
	let prop = '';
	while (i < s.length) {
		if (isPropChar(s[i])) {
			const begin = i;
			while (i < s.length && isPropChar(s[i])) i++;
			prop += s.slice(begin, i);
			continue;
		}
		if (s[i] === '.') {
			prop += '.';
			i++;
			continue;
		}
		break;
	}
	if (!isValidPropertyPath(prop)) throw new Error('Invalid property path');
	if (s[i] === '[') throw new Error('Bracket notation is not supported in filter property paths');
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
		if (
			end === len ||
			isWhitespace(nextChar) ||
			nextChar === '(' ||
			nextChar === ')' ||
			nextChar === '"' ||
			nextChar === "'"
		) {
			return [op as FilterOperator, end];
		}
	}
	throw new Error('Unknown operator');
}

// parseLogical parses a logical operator at i.
const parseLogical = (s: string, i: number): FilterLogical | null => {
	for (const operator of ['and', 'or'] as FilterLogical[]) {
		const end = i + operator.length;
		if (s.slice(i, end) === operator && (end === s.length || isWhitespace(s[end]) || s[end] === '(')) {
			return operator;
		}
	}

	return null;
};

// parseFilter parses a user-readable expression into an EditorFilter object.
// Parentheses delimit nested groups with their own logical operator.
export function parseFilter(s: string, schema?: ObjectType): EditorFilter {
	let ruleCount = 0;

	const parseCondition = (start: number): [EditorFilterCondition, number] => {
		let i = start;
		const [property, ni1] = parseProperty(s, i);
		i = skipSpaces(s, ni1);
		const [operator, ni2] = parseOperator(s, i);
		i = skipSpaces(s, ni2);

		let values: FilterToken[] = [];
		if (!isUnaryOperator(operator)) {
			const expectMultiple =
				operator === 'is one of' ||
				operator === 'is not one of' ||
				operator === 'is between' ||
				operator === 'is not between';
			const [parsedValues, ni3] = parseValues(s, i, expectMultiple);
			values = parsedValues;
			i = ni3;

			if ((operator === 'is between' || operator === 'is not between') && values.length !== 2) {
				throw new Error(`Operator '${operator}' requires exactly 2 values`);
			}
			if ((operator === 'is one of' || operator === 'is not one of') && values.length < 1) {
				throw new Error(`Operator '${operator}' requires at least 1 value`);
			}
			if (UNARY_OPERATORS.includes(operator) && values.length !== 1) {
				throw new Error(`Operator '${operator}' requires exactly 1 value`);
			}
		}

		if (isUnaryOperator(operator)) return [{ property, operator }, i];
		const propertyType = schema == null ? undefined : filterProperty(schema, property).type;
		const type = propertyType?.kind === 'array' ? propertyType.elementType : propertyType;
		const controls = values.map((token) => {
			if (!token.isQuoted) return token.text;
			if (type?.kind === 'json' || type == null) return quoteFilterText(token.text);
			if (type.kind === 'int' || type.kind === 'year' || type.kind === 'float' || type.kind === 'decimal') {
				return quoteFilterText(token.text);
			}
			return propertyType.kind === 'string' && propertyType.values != null
				? token.text
				: formatString(token.text);
		});
		const valueKinds =
			propertyType == null
				? undefined
				: controls.map((text) => filterValueKind(parseFilterOperand(text, propertyType, operator)));
		return [{ property, operator, values: controls, ...(valueKinds == null ? {} : { valueKinds }) }, i];
	};

	const parseGroup = (start: number, depth: number, nested: boolean): [EditorFilter, number] => {
		if (depth > MAX_FILTER_DEPTH + 1) {
			throw new Error(`Filter exceeds maximum depth of ${MAX_FILTER_DEPTH}`);
		}

		let i = skipSpaces(s, start);
		const rules: EditorFilterRule[] = [];
		let operator: FilterLogical | null = null;
		for (;;) {
			if (nested && s[i] === ')') {
				if (rules.length === 0) throw new Error('Filter groups cannot be empty');
				return [{ operator: operator ?? 'and', rules }, i + 1];
			}
			if (i >= s.length) {
				if (nested) throw new Error('Unclosed filter group');
				if (rules.length === 0) throw new Error('Filter groups cannot be empty');
				return [{ operator: operator ?? 'and', rules }, i];
			}

			let rule: EditorFilterRule;
			if (s[i] === '(') {
				[rule, i] = parseGroup(i + 1, depth + 1, true);
			} else {
				[rule, i] = parseCondition(i);
			}
			rules.push(rule);
			ruleCount++;
			if (ruleCount > MAX_FILTER_RULE_COUNT + 1) {
				throw new Error(`Filter exceeds maximum rule count of ${MAX_FILTER_RULE_COUNT}`);
			}

			i = skipSpaces(s, i);
			if (nested && s[i] === ')') {
				return [{ operator: operator ?? 'and', rules }, i + 1];
			}
			if (i >= s.length) {
				if (nested) throw new Error('Unclosed filter group');
				return [{ operator: operator ?? 'and', rules }, i];
			}
			if (s[i] === ')') throw new Error('Unexpected closing parenthesis');

			const next = parseLogical(s, i);
			if (next == null) throw new Error("Expected logical connector 'and' or 'or'");
			if (operator != null && operator !== next) {
				throw new Error('Mixed logical connectors require parentheses');
			}
			operator = next;
			i = skipSpaces(s, i + next.length);
			if (i >= s.length || (nested && s[i] === ')')) {
				throw new Error('Trailing logical connector without following rule');
			}
		}
	};

	const [filter, end] = parseGroup(0, 1, false);
	if (skipSpaces(s, end) !== s.length) throw new Error('Unexpected content after filter');
	const result = filter.rules.length === 1 && isFilterGroup(filter.rules[0]) ? filter.rules[0] : filter;

	let actualRuleCount = 0;
	const validateGroupLimits = (group: EditorFilter, depth: number) => {
		if (depth > MAX_FILTER_DEPTH) {
			throw new Error(`Filter exceeds maximum depth of ${MAX_FILTER_DEPTH}`);
		}
		for (const rule of group.rules) {
			actualRuleCount++;
			if (actualRuleCount > MAX_FILTER_RULE_COUNT) {
				throw new Error(`Filter exceeds maximum rule count of ${MAX_FILTER_RULE_COUNT}`);
			}
			if (isFilterGroup(rule)) {
				validateGroupLimits(rule, depth + 1);
			}
		}
	};
	validateGroupLimits(result, 1);

	return result;
}
