interface MeergoFunction {
	name: string;
	params: string[];
	return: string;
	description: string;
}

const MEERGO_FUNCTIONS: MeergoFunction[] = [
	{
		name: 'and',
		params: ['...args: boolean'],
		return: 'boolean',
		description: 'If all arguments are true, it returns true; otherwise, false',
	},
	{
		name: 'array',
		params: ['...elements: any'],
		return: 'array(json)',
		description: 'Returns an array with the given elements',
	},
	{
		name: 'coalesce',
		params: ['...args: any'],
		return: 'json',
		description: 'Returns the first non-null argument, or null if all are null',
	},
	{
		name: 'eq',
		params: ['a: any', 'b: any'],
		return: 'boolean',
		description: 'If a and b are equal, it returns true; otherwise, false',
	},
	{
		name: 'if',
		params: ['condition: boolean', 'a: any', 'b: any'],
		return: 'json',
		description: 'If the condition is true, it returns a; otherwise, b',
	},
	{
		name: 'initcap',
		params: ['s: string'],
		return: 'string',
		description: 'Converts the first letter of each word to uppercase and the rest to lowercase',
	},
	{
		name: 'json_parse',
		params: ['s: string'],
		return: 'json',
		description: 'Parses s as JSON and returns a json value',
	},
	{
		name: 'len',
		params: ['s: any'],
		return: 'int(32)',
		description: 'Returns the length of s, depending on its type',
	},
	{
		name: 'lower',
		params: ['s: string'],
		return: 'string',
		description: 'Converts s to all lowercase letters',
	},
	{
		name: 'ltrim',
		params: ['s: string'],
		return: 'string',
		description: 'Removes all leading Unicode whitespace from s',
	},
	{
		name: 'map',
		params: ['...pairs: string,any'],
		return: 'map(json)',
		description: 'Builds a map from string keys and values',
	},
	{
		name: 'ne',
		params: ['a: any', 'b: any'],
		return: 'boolean',
		description: 'If a and b are not equal, it returns true; otherwise, false',
	},
	{
		name: 'not',
		params: ['b: boolean'],
		return: 'boolean',
		description: 'Negates b, returning false if true and true if false',
	},
	{
		name: 'or',
		params: ['...args: boolean'],
		return: 'boolean',
		description: 'Returns true if at least one argument is true; otherwise, false',
	},
	{
		name: 'rtrim',
		params: ['s: string'],
		return: 'string',
		description: 'Removes all trailing Unicode whitespace from s',
	},
	{
		name: 'substring',
		params: ['s: string', 'start: integer', 'length: integer'],
		return: 'string',
		description: 'Extracts a substring of s from position start with length',
	},
	{
		name: 'trim',
		params: ['s: string'],
		return: 'string',
		description: 'Removes all leading and trailing Unicode whitespace from s',
	},
	{
		name: 'upper',
		params: ['s: string'],
		return: 'string',
		description: 'Converts s to uppercase',
	},
];

export { MEERGO_FUNCTIONS, MeergoFunction };
