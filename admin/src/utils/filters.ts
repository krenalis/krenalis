import BigNumber from 'bignumber.js';
import JSONbig from 'json-bigint';
import {
	Filter,
	FilterCondition,
	FilterRule,
	FilterOperator,
	FilterNumber,
	FilterValue,
	FilterValueKind,
	EditorFilter,
	EditorFilterRule,
	TransportFilter,
} from '../lib/api/types/pipeline';
import Type, { ObjectType, Property } from '../lib/api/types/types';

const FILTER_OPERATORS: FilterOperator[] = [
	'is',
	'is not',
	'is less than',
	'is less than or equal to',
	'is greater than',
	'is greater than or equal to',
	'is between',
	'is not between',
	'contains',
	'does not contain',
	'is one of',
	'is not one of',
	'starts with',
	'ends with',
	'is before',
	'is on or before',
	'is after',
	'is on or after',
	'is true',
	'is false',
	'is empty',
	'is not empty',
	'is null',
	'is not null',
	'exists',
	'does not exist',
];

const MAX_FILTER_DEPTH = 4;
const MAX_FILTER_RULE_COUNT = 100;

const UNARY_OPERATORS = new Set<FilterOperator>([
	'is true',
	'is false',
	'is empty',
	'is not empty',
	'is null',
	'is not null',
	'exists',
	'does not exist',
]);

const isFilterGroup = <T extends FilterRule | EditorFilterRule>(rule: T): rule is Extract<T, Filter | EditorFilter> =>
	'rules' in rule;

const isUnaryOperator = (op: FilterOperator | ''): boolean => op !== '' && UNARY_OPERATORS.has(op);

const MIN_INT = BigInt('-9223372036854775808');
const MAX_INT = BigInt('9223372036854775807');
const MAX_UNSIGNED = BigInt('18446744073709551615');
const MAX_FLOAT32 = 3.4028234663852885981170418348451692544e38;
const MIN_YEAR = 1;
const MAX_YEAR = 9999;

// formatText formats a string value as a string.
const formatString = (str: string): string => {
	if (!/^[\s"']/.test(str) && !/[\s"']$/.test(str)) {
		return str;
	}
	const quote = str.includes('"') ? "'" : '"';
	let s = quote;
	for (let i = 0; i < str.length; i++) {
		const c = str[i];
		if (c === '\\' || c === quote) {
			s += '\\';
		}
		s += c;
	}
	s += quote;
	return s;
};

// isDate checks whether the string s represents a valid date value that can be
// used as a filter value.
const isDate = (s: string): boolean => {
	const m = s.match(/^(\d{4})-(\d{2})-(\d{2})$/);
	if (m == null) {
		return false;
	}
	const date = new Date(s);
	if (Number.isNaN(date.valueOf())) {
		return false;
	}
	const [year, month, day] = [Number(m[1]), Number(m[2]), Number(m[3])];
	if (year !== date.getUTCFullYear() || month !== date.getUTCMonth() + 1 || day !== date.getUTCDate()) {
		return false;
	}
	return MIN_YEAR <= year && year <= MAX_YEAR;
};

// isDateTime checks whether the string s represents a valid datetime value that
// can be used as a filter value.
const isDateTime = (s: string): boolean => {
	const match = s.match(/^(\d{4}-\d{2}-\d{2})T(\d{2}):(\d{2}):(\d{2})(?:\.\d{1,9})?([+-]\d{2}:\d{2}|Z)?$/);
	if (match == null || !isDate(match[1]) || Number(match[2]) > 23 || Number(match[3]) > 59 || Number(match[4]) > 59)
		return false;
	const date = new Date(match[5] == null ? s + 'Z' : s);
	return Number.isFinite(date.valueOf()) && date.getUTCFullYear() >= MIN_YEAR && date.getUTCFullYear() <= MAX_YEAR;
};

// isDecimal checks whether the string s represents a valid decimal value that
// can be used as a filter value.
const isDecimal = (s: string): boolean => {
	if (s === '') {
		return false;
	}
	if (s[0] === '-' || s[0] === '+') {
		s = s.slice(1);
	}
	let i = parseDecimalDigits(s);
	if (i === 0) {
		return false;
	}
	if (i === s.length) {
		return true;
	}
	let c = s[i];
	s = s.slice(i + 1);
	if (c === '.') {
		i = parseDecimalDigits(s);
		if (i === 0) {
			return false;
		}
		if (i === s.length) {
			return true;
		}
		c = s[i];
		s = s.slice(i + 1);
	}
	if ((c !== 'e' && c !== 'E') || s === '') {
		return false;
	}
	c = s[0];
	if (c === '-' || c === '+') {
		s = s.slice(1);
	}
	i = parseDecimalDigits(s);
	return i === s.length;
};

const IPv4 = /^(?:25[0-5]|2[0-4]\d|1\d{2}|\d{1,2})(?:\.(?:25[0-5]|2[0-4]\d|1\d{2}|\d{1,2})){3}$/;
const IPv6 =
	/^((?:[0-9a-f]{1,4}:){7}(?:[0-9a-f]{1,4}|:)|(?:[0-9a-f]{1,4}:){1,7}:|(?:[0-9a-f]{1,4}:){1,6}(?::[0-9a-f]{1,4}){1,1}|(?:[0-9a-f]{1,4}:){1,5}(?::[0-9a-f]{1,4}){1,2}|(?:[0-9a-f]{1,4}:){1,4}(?::[0-9a-f]{1,4}){1,3}|(?:[0-9a-f]{1,4}:){1,3}(?::[0-9a-f]{1,4}){1,4}|(?:[0-9a-f]{1,4}:){1,2}(?::[0-9a-f]{1,4}){1,5}|[0-9a-f]?(?::(?::[0-9a-f]{1,4}){1,6})|::(?:[0-9a-f]{1,4}:){0,5}[0-9a-f]{1,4})$/i;

// isIP checks whether the string s represents a valid ip value that can be used
//  as a filter value.
const isIP = (s: string): boolean => {
	return IPv4.test(s) || IPv6.test(s);
};

// isInt checks whether the string s represents a valid int value that can be
// used as a filter value.
const isInt = (s: string): boolean => {
	let t = s;
	if ((s.length > 0 && s[0] === '-') || s[0] === '+') {
		t = s.slice(1);
	}
	if (!isUnsigned(t)) {
		return false;
	}
	const n = BigInt(s);
	return MIN_INT <= n && n <= MAX_INT;
};

// isYear checks whether the string s represents a valid year value that can be
// used as a filter value.
const isYear = (s: string): boolean => {
	if (s === '' || s.length > 4) {
		return false;
	}
	for (const c of s) {
		if (c < '0' || c > '9') {
			return false;
		}
	}
	const year = parseInt(s);
	return MIN_YEAR <= year && year <= MAX_YEAR;
};

// isFloat checks whether the string s represents a valid float value with the
// specified bit size, which can be either 32 or 64, and can be used as a filter
// value.
const isFloat = (s: string, bitSize: number): boolean => {
	if (!isDecimal(s)) {
		return false;
	}
	const n = parseFloat(s);
	if (bitSize === 32) {
		return -MAX_FLOAT32 <= n && n <= MAX_FLOAT32;
	}
	return n !== Infinity && n !== -Infinity;
};

// isTime checks whether the string s represents a valid time value that can be
// used as a filter value.
const isTime = (s: string): boolean => {
	const m = s.match(/^(\d{2}):(\d{2}):(\d{2})(?:\.\d{1,9})?$/);
	if (m == null) {
		return false;
	}
	const [hour, minute, second] = [Number(m[1]), Number(m[2]), Number(m[3])];
	return 0 <= hour && hour <= 23 && 0 <= minute && minute <= 59 && 0 <= second && second <= 59;
};

// isUUID checks whether the string s represents a valid uuid value that can be
// used as a filter value.
const isUUID = (s: string): boolean => {
	return /^[0-9a-f]{8}-[0-9a-f]{4}-[1-9][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i.test(s);
};

// isUnsigned checks whether the string s represents a valid unsigned int value
// that can be used as a filter value.
const isUnsigned = (s: string): boolean => {
	if (s.length === 0) {
		return false;
	}
	if (s === '0') {
		return true;
	}
	for (let i = 0; i < s.length; i++) {
		const c = s[i];
		if (c < '0' || c > '9' || (i === 0 && c === '0')) {
			return false;
		}
	}
	const n = BigInt(s);
	return n <= MAX_UNSIGNED;
};

// isValidPropertyPath checks whether s is a valid property path.
// A property path is formed by property names separated by periods.
const isValidPropertyPath = (s: string): boolean => {
	return /^[A-Za-z_]\w*(?:\.[A-Za-z_]\w*)*$/.test(s);
};

// parseDecimalDigits parses the string s and returns the index of the first
// character in s that is not a decimal digit (0-9).
const parseDecimalDigits = (s: string): number => {
	let i = 0;
	for (; i < s.length; i++) {
		const c = s[i];
		if (c < '0' || c > '9') {
			break;
		}
	}
	return i;
};

const EmptyTextError = new Error('text is empty');
const InvalidTextError = new Error('text is not valid');

// parseFilterText retains quoting while sharing the editor's escape syntax.
const parseFilterText = (input: string): { text: string; isQuoted: boolean } => {
	const s = input.trim();
	if (s === '') throw EmptyTextError;
	const quote = s[0];
	if (quote !== '"' && quote !== "'") return { text: s, isQuoted: false };
	if (s.length < 2 || s[s.length - 1] !== quote) throw InvalidTextError;
	let text = '';
	for (let i = 1; i < s.length - 1; i++) {
		if (s[i] === '\\') {
			i++;
			if (i === s.length - 1) throw InvalidTextError;
		} else if (s[i] === quote) {
			throw InvalidTextError;
		}
		text += s[i];
	}
	return { text, isQuoted: true };
};

const parseText = (input: string): string => {
	const { text } = parseFilterText(input);
	if (text === '') throw EmptyTextError;
	return text;
};

const JSON_NUMBER = /^-?(?:0|[1-9][0-9]*)(?:\.[0-9]+)?(?:[eE][+-]?[0-9]+)?$/;
const isJSONNumber = (text: string): boolean => JSON_NUMBER.test(text);
const isMembershipOperator = (operator: FilterOperator | ''): boolean =>
	operator === 'is one of' || operator === 'is not one of';

// isDecimalNumber checks representability without rounding or expanding powers.
const isDecimalNumber = (text: string): boolean => {
	if (!isJSONNumber(text)) return false;
	const [mantissa, exponent = '0'] = text.replace(/^-/, '').split(/[eE]/);
	const [integer, fraction = ''] = mantissa.split('.');
	const coefficient = (integer + fraction).replace(/^0+/, '');
	if (coefficient === '') return true;
	const significant = coefficient.replace(/0+$/, '');
	const exponentDigits = exponent.replace(/^[+-]?0*/, '') || '0';
	const bound = text.length + 76;
	if (exponentDigits.length > String(bound).length || Number(exponentDigits) > bound) return false;
	const e = Number(exponent) - fraction.length + coefficient.length - significant.length;
	const scale = Math.max(-e, 0);
	const precision = Math.max(significant.length + e, 0) + scale;
	return precision <= 76 && scale <= 37;
};

const filterNumber = (text: string, decimalLimits: boolean): FilterNumber => {
	if (!isJSONNumber(text)) throw new Error('Invalid JSON Number');
	if (decimalLimits && !isDecimalNumber(text)) {
		throw new Error('Number must fit precision 76 and scale 37');
	}
	// Canonicalize zero before passing an arbitrarily large exponent to BigNumber.
	const mantissa = text.split(/[eE]/)[0];
	return { kind: 'number', text: /^-?0(?:\.0+)?$/.test(mantissa) ? '0' : text };
};

const parseJSONFilterText = (input: string, operator: FilterOperator): FilterValue => {
	const { text, isQuoted } = parseFilterText(input);
	if (isQuoted) return text;
	if (isMembershipOperator(operator)) {
		if (text === 'true') return true;
		if (text === 'false') return false;
		if (text === 'null') return null;
	}
	return isJSONNumber(text) ? filterNumber(text, true) : text;
};

// parseFilterOperand converts one editor control using its schema type.
const parseFilterOperand = (input: string, type: Type, operator: FilterOperator): FilterValue => {
	const isArray = type.kind === 'array';
	if (type.kind === 'array') type = type.elementType;
	if (type.kind === 'json') return parseJSONFilterText(input, operator);
	if (type.kind === 'string') return !isArray && type.values != null ? input : parseText(input);
	let valid = false;
	switch (type.kind) {
		case 'int':
			valid = type.unsigned ? isUnsigned(input) : isInt(input);
			break;
		case 'year':
			valid = isYear(input);
			break;
		case 'float':
			valid = isFloat(input, type.bitSize);
			break;
		case 'decimal':
			valid = isDecimal(input);
			break;
		case 'datetime':
			valid = isDateTime(input);
			break;
		case 'date':
			valid = isDate(input);
			break;
		case 'time':
			valid = isTime(input);
			break;
		case 'uuid':
			valid = isUUID(input);
			break;
		case 'ip':
			valid = isIP(input);
			break;
		default:
			throw new Error(`Operands are not supported for ${type.kind}`);
	}
	if (!valid) throw new Error(`Invalid ${type.kind} filter value`);
	if (type.kind === 'int' || type.kind === 'year' || type.kind === 'float' || type.kind === 'decimal') {
		const text = input.replace(/^\+/, '').replace(/^(-?)0+(?=\d)/, '$1');
		return filterNumber(text, type.kind === 'decimal');
	}
	return input;
};

// quoteFilterText is the inverse of the editor tokenizer, not JSON quoting.
const quoteFilterText = (text: string): string => '"' + text.replace(/\\/g, '\\\\').replace(/"/g, '\\"') + '"';

const filterValueKind = (value: FilterValue): FilterValueKind =>
	value === null ? 'null' : typeof value === 'object' ? 'number' : (typeof value as 'string' | 'boolean');

const formatJSONFilterValue = (value: FilterValue): string => {
	if (typeof value === 'string') return quoteFilterText(value);
	if (value === null) return 'null';
	if (typeof value === 'boolean') return String(value);
	return value.text;
};

// filterProperty resolves ordinary object paths and stops at a JSON property.
const filterProperty = (schema: ObjectType, path: string): Property => {
	let properties = schema.properties;
	const names = path.split('.');
	for (const [i, name] of names.entries()) {
		const property = properties.find((p) => p.name === name);
		if (property == null) throw new Error(`Filter property "${path}" does not exist`);
		if (i === names.length - 1 || property.type.kind === 'json') return property;
		if (property.type.kind !== 'object') throw new Error(`Filter property "${path}" does not exist`);
		properties = property.type.properties;
	}
	throw new Error(`Invalid filter property "${path}"`);
};

const toEditorFilter = (filter: Filter, schema: ObjectType): EditorFilter => ({
	operator: filter.operator,
	rules: filter.rules.map((rule) => {
		if (isFilterGroup(rule)) return toEditorFilter(rule, schema);
		let type: Type;
		try {
			type = filterProperty(schema, rule.property).type;
		} catch {
			/* Display unresolved properties too. */
		}
		const isEnum = type?.kind === 'string' && type.values != null;
		if (type?.kind === 'array') type = type.elementType;
		const values = rule.values?.map((value) => {
			if (type?.kind === 'json' || type == null) return formatJSONFilterValue(value);
			if (typeof value !== 'string') return formatJSONFilterValue(value);
			return isEnum ? value : formatString(value);
		});
		return { ...rule, values, valueKinds: rule.values?.map(filterValueKind) };
	}),
});

// decodeFilter accepts values only from the lossless HTTP JSON decoder.
const decodeFilter = (value: unknown): Filter => {
	if (
		value == null ||
		typeof value !== 'object' ||
		!('operator' in value) ||
		!('rules' in value) ||
		(value.operator !== 'and' && value.operator !== 'or') ||
		!Array.isArray(value.rules)
	) {
		throw new Error('Invalid filter response');
	}
	return {
		operator: value.operator,
		rules: value.rules.map((rule): FilterRule => {
			if (rule == null || typeof rule !== 'object') throw new Error('Invalid filter rule response');
			if ('rules' in rule) return decodeFilter(rule);
			if (
				typeof rule.property !== 'string' ||
				!FILTER_OPERATORS.includes(rule.operator) ||
				(rule.values != null && !Array.isArray(rule.values))
			)
				throw new Error('Invalid filter condition response');
			const condition: FilterCondition = { property: rule.property, operator: rule.operator };
			if (rule.values != null)
				condition.values = rule.values.map((operand: unknown): FilterValue => {
					if (operand === null) return null;
					if (typeof operand === 'string' || typeof operand === 'boolean') return operand;
					if (BigNumber.isBigNumber(operand)) return filterNumber(operand.toString(), false);
					if (typeof operand === 'number' && Number.isFinite(operand))
						return filterNumber(String(operand), false);
					throw new Error('Invalid filter operand response');
				});
			return condition;
		}),
	};
};

const toTransportFilter = (filter: Filter): TransportFilter => ({
	operator: filter.operator,
	rules: filter.rules.map((rule) => {
		if (isFilterGroup(rule)) return toTransportFilter(rule);
		const values = rule.values?.map((value) => {
			if (value === null) return null;
			if (typeof value === 'string' || typeof value === 'boolean') return value;
			if (value.kind !== 'number' || !isJSONNumber(value.text)) throw new Error('Invalid filter Number');
			const number = new BigNumber(value.text);
			if (!number.isFinite()) throw new Error('Non-finite filter Number');
			return number;
		});
		return values == null
			? { property: rule.property, operator: rule.operator }
			: { property: rule.property, operator: rule.operator, values };
	}),
});

const encodeFilter = (filter: Filter): string => JSONbig.stringify(toTransportFilter(filter));

// serializeFilter returns a string representation of the given Filter object.
// If formatted is true, each rule appears on its own indented line.
const serializeFilter = (filter: Filter, formatted: boolean): string => {
	// escapeString returns the input string escaped and wrapped in double quotes.
	function escapeString(value: string): string {
		return `"${value
			.replace(/\\/g, '\\\\')
			.replace(/"/g, '\\"')
			.replace(/\n/g, '\\n')
			.replace(/\r/g, '\\r')
			.replace(/\t/g, '\\t')}"`;
	}

	// formatValues formats a list of values as a string.
	function formatValues(values: FilterValue[], operator: FilterOperator): string {
		const formattedValues = values.map((value) => {
			if (typeof value === 'string') return escapeString(value);
			if (value === null || typeof value === 'boolean') return String(value);
			if (!isJSONNumber(value.text)) throw new Error('Invalid filter Number');
			return value.text;
		});
		if (values.length === 0) return '';
		return values.length === 1 &&
			!isMembershipOperator(operator) &&
			operator !== 'is between' &&
			operator !== 'is not between'
			? formattedValues[0]
			: `(${formattedValues.join(', ')})`;
	}

	// formatCondition builds a single condition string.
	function formatCondition(condition: FilterCondition): string {
		const { property, operator, values } = condition;
		const formattedValues = formatValues(values ?? [], operator);
		return `${property} ${operator}${formattedValues === '' ? '' : ` ${formattedValues}`}`;
	}

	// formatRule formats a condition or nested group.
	function formatRule(rule: FilterRule, depth: number): string {
		if (!('rules' in rule)) {
			return formatCondition(rule);
		}
		if (!formatted) {
			return `(${formatGroup(rule, depth + 1)})`;
		}
		return `(\n${formatGroup(rule, depth + 1)}\n${'  '.repeat(depth)})`;
	}

	// formatGroup formats all rules in a group.
	function formatGroup(group: Filter, depth: number): string {
		if (!formatted) {
			return group.rules.map((rule) => formatRule(rule, depth)).join(` ${group.operator} `);
		}
		const indentation = '  '.repeat(depth);
		return group.rules
			.map((rule, index) => `${indentation}${index === 0 ? '' : `${group.operator} `}${formatRule(rule, depth)}`)
			.join('\n');
	}

	return formatGroup(filter, 0);
};

export {
	FILTER_OPERATORS,
	MAX_FILTER_DEPTH,
	MAX_FILTER_RULE_COUNT,
	isFilterGroup,
	isUnaryOperator,
	decodeFilter,
	toTransportFilter,
	encodeFilter,
	toEditorFilter,
	filterProperty,
	parseJSONFilterText,
	formatJSONFilterValue,
	quoteFilterText,
	filterValueKind,
	isJSONNumber,
	isDecimalNumber,
	filterNumber,
	isMembershipOperator,
	parseFilterOperand,
	formatString,
	isDate,
	isDateTime,
	isDecimal,
	isFloat,
	isIP,
	isInt,
	isTime,
	isUUID,
	isUnsigned,
	isValidPropertyPath,
	isYear,
	parseText,
	serializeFilter,
};
