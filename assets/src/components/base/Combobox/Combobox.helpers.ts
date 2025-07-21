const LEFT_PARENTHESIS = `(`;
const RIGHT_PARENTHESIS = ')';
const SINGLE_QUOTE = `'`;
const DOUBLE_QUOTE = `"`;
const EMPTY_SPACE = ' ';
const COMMA = ',';
const COLON = ':';
const SEMICOLON = ';';
const PERIOD = '.';
const SOFT_CHARS = [undefined, LEFT_PARENTHESIS, RIGHT_PARENTHESIS, EMPTY_SPACE, COMMA, COLON, SEMICOLON, PERIOD];

const autocompleteExpression = (expression: string, cursorPosition: number): string | null => {
	const char = expression[cursorPosition - 1];
	const previousChar = expression[cursorPosition - 2];
	const nextChar = expression[cursorPosition];

	let autocompleted: string;

	if (char === LEFT_PARENTHESIS) {
		// Close the parenthesis.
		autocompleted = expression.slice(0, cursorPosition) + RIGHT_PARENTHESIS + expression.slice(cursorPosition);
	}

	if (char === SINGLE_QUOTE) {
		if (isSoftChar(previousChar)) {
			// Close the single quote.
			autocompleted = expression.slice(0, cursorPosition) + SINGLE_QUOTE + expression.slice(cursorPosition);
		}
		const ranges = getRanges(getPreviousExpression(expression, cursorPosition), SINGLE_QUOTE);
		const isInsideSingleQuoted =
			ranges.findIndex((range) => range[0] <= cursorPosition - 1 && cursorPosition - 1 <= range[1]) !== -1;
		if (isInsideSingleQuoted && nextChar === SINGLE_QUOTE) {
			// Move the cursor after the already inserted single quote
			// without adding another one.
			autocompleted = expression.slice(0, cursorPosition - 1) + expression.slice(cursorPosition);
		}
	}

	if (char === DOUBLE_QUOTE) {
		if (isSoftChar(previousChar)) {
			// Close the double quote.
			autocompleted = expression.slice(0, cursorPosition) + DOUBLE_QUOTE + expression.slice(cursorPosition);
		}
		const ranges = getRanges(getPreviousExpression(expression, cursorPosition), DOUBLE_QUOTE);
		const isInsideDoubleQuotes =
			ranges.findIndex((range) => range[0] <= cursorPosition - 1 && cursorPosition - 1 <= range[1]) !== -1;
		if (isInsideDoubleQuotes && nextChar === DOUBLE_QUOTE) {
			// Move the cursor after the already inserted double quote
			// without adding another one.
			autocompleted = expression.slice(0, cursorPosition - 1) + expression.slice(cursorPosition);
		}
	}

	if (char === RIGHT_PARENTHESIS && previousChar === LEFT_PARENTHESIS && nextChar === RIGHT_PARENTHESIS) {
		// Move the cursor after the already inserted right parenthesis
		// without adding another one.
		autocompleted = expression.slice(0, cursorPosition - 1) + expression.slice(cursorPosition);
	}

	return autocompleted;
};

const isSoftChar = (char: string): boolean => {
	if (SOFT_CHARS.includes(char)) {
		return true;
	}
	return false;
};

const getPreviousExpression = (expression: string, cursorPosition: number): string => {
	const split = expression.split('');
	split.splice(cursorPosition, 1);
	return split.join('');
};

const getRanges = (expression: string, symbol: string): Array<[number, number]> => {
	const split = expression.split('');
	let ranges = [];
	let currentRange = [];
	let isOpen = false;
	let index = 0;
	for (const char of split) {
		if (char === symbol) {
			if (isOpen) {
				currentRange.push(index);
				ranges.push(structuredClone(currentRange));
				isOpen = false;
				currentRange = [];
			} else {
				currentRange.push(index);
				isOpen = true;
			}
		}
		index++;
	}
	return ranges;
};

export { autocompleteExpression };
