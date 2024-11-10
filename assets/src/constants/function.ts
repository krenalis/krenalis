interface MeergoFunction {
	name: string;
	params: string[];
	return: string;
	description: string;
}

const MEERGO_FUNCTIONS: MeergoFunction[] = [
	{
		name: 'and',
		params: ['...args: Boolean'],
		return: 'Boolean',
		description: 'If all arguments are true, it returns true; otherwise, false',
	},
	{
		name: 'array',
		params: ['...elements: Any'],
		return: 'Array(JSON)',
		description: 'Returns an array with the given elements',
	},
	{
		name: 'coalesce',
		params: ['...args: Any'],
		return: 'JSON',
		description: 'Returns the first non-null argument, or null if all are null',
	},
	{
		name: 'eq',
		params: ['a: Any', 'b: Any'],
		return: 'Boolean',
		description: 'If a and b are equal, it returns true; otherwise, false',
	},
	{
		name: 'if',
		params: ['condition: Boolean', 'a: Any', 'b: Any'],
		return: 'JSON',
		description: 'If the condition is true, it returns a; otherwise, b',
	},
	{
		name: 'initcap',
		params: ['s: Text'],
		return: 'Text',
		description: 'Converts the first letter of each word to uppercase and the rest to lowercase',
	},
	{
		name: 'json_parse',
		params: ['s: Text'],
		return: 'JSON',
		description: 'Parses s as JSON and returns a JSON value',
	},
	{
		name: 'len',
		params: ['s: Any'],
		return: 'Int(32)',
		description: 'Returns the length of s, depending on its type',
	},
	{
		name: 'lower',
		params: ['s: Text'],
		return: 'Text',
		description: 'Converts s to all lowercase letters',
	},
	{
		name: 'ltrim',
		params: ['s: Text'],
		return: 'Text',
		description: 'Removes all leading Unicode whitespace from s',
	},
	{
		name: 'ne',
		params: ['a: Any', 'b: Any'],
		return: 'Boolean',
		description: 'If a and b are not equal, it returns true; otherwise, false',
	},
	{
		name: 'not',
		params: ['b: Boolean'],
		return: 'Boolean',
		description: 'Negates b, returning false if true and true if false',
	},
	{
		name: 'or',
		params: ['...args: Boolean'],
		return: 'Boolean',
		description: 'Returns true if at least one argument is true; otherwise, false',
	},
	{
		name: 'rtrim',
		params: ['s: Text'],
		return: 'Text',
		description: 'Removes all trailing Unicode whitespace from s',
	},
	{
		name: 'substring',
		params: ['s: Text', 'start: Integer', 'length: Integer'],
		return: 'Text',
		description: 'Extracts a substring of s from position start with length',
	},
	{
		name: 'trim',
		params: ['s: Text'],
		return: 'Text',
		description: 'Removes all leading and trailing Unicode whitespace from s',
	},
	{
		name: 'upper',
		params: ['s: Text'],
		return: 'Text',
		description: 'Converts s to uppercase',
	},
];

export { MEERGO_FUNCTIONS, MeergoFunction };
