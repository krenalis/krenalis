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
		description: 'Returns true only when all of its arguments are true; otherwise, it returns false',
	},
	{
		name: 'array',
		params: ['...elements: Any'],
		return: 'Array(JSON)',
		description: 'The array function returns an array with the passed arguments as elements',
	},
	{
		name: 'coalesce',
		params: ['...args: Any'],
		return: 'JSON',
		description: 'The coalesce function returns the first non-null argument, or null if all arguments are null',
	},
	{
		name: 'eq',
		params: ['arg1: Any', 'arg2: Any'],
		return: 'Boolean',
		description: 'The eq function takes two values and returns true if they are equal; otherwise, it returns false',
	},
	{
		name: 'if',
		params: ['condition: Boolean', 'return1: Any', 'return2: Any'],
		return: 'JSON',
		description:
			'The if function evaluates the first boolean argument. If it is true, the function returns the second argument. If the first argument is false or null, the function returns the third argument',
	},
	{
		name: 'initcap',
		params: ['s: Text'],
		return: 'Text',
		description:
			'The initcap function returns its argument with the first letter of each word in uppercase, all other letters in lowercase',
	},
	{
		name: 'len',
		params: ['arg: Any'],
		return: 'Int(32)',
		description: 'The len function returns the length of the given argument based on its type',
	},
	{
		name: 'lower',
		params: ['s: Text'],
		return: 'Text',
		description: 'The lower function returns its argument with all letters in lower case',
	},
	{
		name: 'ltrim',
		params: ['s: Text'],
		return: 'Text',
		description: 'The ltrim function returns its argument with all leading Unicode whitespace removed',
	},
	{
		name: 'ne',
		params: ['arg1: Any', 'arg2: Any'],
		return: 'Boolean',
		description:
			'The ne function takes two values and returns true if they are not equal; otherwise, it returns false',
	},
	{
		name: 'not',
		params: ['arg1: Boolean'],
		return: 'Boolean',
		description: 'The not function returns false if its argument is true, and true if its argument is false',
	},
	{
		name: 'or',
		params: ['...args: Boolean'],
		return: 'Boolean',
		description:
			'The or function returns true if at least one of its arguments is true; otherwise, it returns false',
	},
	{
		name: 'rtrim',
		params: ['s: Text'],
		return: 'Text',
		description: 'The rtrim function returns its argument with all trailing Unicode whitespace removed',
	},
	{
		name: 'substring',
		params: ['s: Text', 'start: Integer', 'length: Integer'],
		return: 'Text',
		description:
			'The substring function extracts a portion of a string based on a specified starting position and length. The indices are 1-based, meaning the first character of the string has an index of 1',
	},
	{
		name: 'trim',
		params: ['s: Text'],
		return: 'Text',
		description: 'The trim function returns its argument with all leading and trailing Unicode whitespace removed',
	},
	{
		name: 'upper',
		params: ['s: Text'],
		return: 'Text',
		description: 'The upper function returns its argument with all letters in upper case',
	},
];

export { MEERGO_FUNCTIONS, MeergoFunction };
