import { Filter, FilterCondition } from '../lib/api/types/pipeline';

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
	if (year !== date.getFullYear() || month !== date.getMonth() + 1 || day !== date.getDate()) {
		return false;
	}
	return MIN_YEAR <= year && year <= MAX_YEAR;
};

// isDateTime checks whether the string s represents a valid datetime value that
// can be used as a filter value.
const isDateTime = (s: string): boolean => {
	const m = s.match(/^(\d{4})-(\d{2})-(\d{2})T\d{2}:\d{2}:\d{2}(?:\.\d{1,9})?(?:[+-]\d{2}:\d{2}|Z)?$/);
	if (m == null) {
		return false;
	}
	const date = new Date(s);
	if (Number.isNaN(date.valueOf())) {
		return false;
	}
	let [year, month, day] = [Number(m[1]), Number(m[2]), Number(m[3])];
	if (year !== date.getFullYear() || month !== date.getMonth() + 1 || day !== date.getDate()) {
		return false;
	}
	year = date.getUTCFullYear();
	return MIN_YEAR <= year && year <= MAX_YEAR;
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
	if (bitSize == 32) {
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
		if (c < '0' || c > '9' || (i == 0 && c == '0')) {
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

// parseText parses s and returns the corresponding text value.
// If the resulting text value is empty, it throws the EmptyTextError error.
// If the resulting text is not valid, it throws the InvalidTextError error.
const parseText = (s: string): string => {
	s = s.trim();
	if (s === '') {
		throw EmptyTextError;
	}
	const quote = s[0];
	if (quote !== '"' && quote !== "'") {
		return s;
	}
	if (s.length < 3) {
		throw EmptyTextError;
	}
	if (s[s.length - 1] !== quote) {
		throw InvalidTextError;
	}
	let text = '';
	for (let i = 1; i < s.length - 1; i++) {
		switch (s[i]) {
			case '\\':
				i++;
				if (i === s.length - 1) {
					throw InvalidTextError;
				}
				break;
			case quote:
				throw InvalidTextError;
		}
		text += s[i];
	}
	return text;
};

// serializeFilter returns a string representation of the given Filter object.
// If formatted is true, each condition appears on its own line with indentation
// based on the logical connector.
const serializeFilter = (filter: Filter, formatted: boolean): string => {
	const { logical, conditions } = filter;

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
	function formatValues(values: string[] | null): string {
		if (!values || values.length === 0) {
			return '';
		}

		if (values.length === 1) {
			const v = values[0];
			if (v === 'true' || v === 'false' || (v !== '' && !isNaN(Number(v)))) {
				return v;
			}
			return escapeString(v);
		}

		const formattedList = values.map((v) => {
			if (v === 'true' || v === 'false' || (v !== '' && !isNaN(Number(v)))) {
				return v;
			}
			return escapeString(v);
		});

		return `(${formattedList.join(', ')})`;
	}

	// formatCondition builds a single condition string.
	function formatCondition(condition: FilterCondition): string {
		const { property, operator, values } = condition;

		if (!values) {
			return `${property} ${operator}`;
		}

		return `${property} ${operator} ${formatValues(values)}`;
	}

	// Build the final string from all conditions.
	const lines: string[] = [];

	for (let i = 0; i < conditions.length; i++) {
		const condStr = formatCondition(conditions[i]);

		if (!formatted) {
			lines.push(condStr);
		} else {
			if (i === 0) {
				lines.push(condStr);
			} else {
				lines.push(`${logical} ${condStr}`); // prefix subsequent lines with the connector
			}
		}
	}

	return formatted ? lines.join('\n') : lines.join(` ${logical} `);
};

export {
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
