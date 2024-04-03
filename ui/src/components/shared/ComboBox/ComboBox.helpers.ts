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
		autocompleted = expression.slice(0, cursorPosition) + RIGHT_PARENTHESIS + expression.slice(cursorPosition);
	}

	if (char === SINGLE_QUOTE) {
		if (isSoftChar(previousChar)) {
			autocompleted = expression.slice(0, cursorPosition) + SINGLE_QUOTE + expression.slice(cursorPosition);
		}
		if (previousChar === SINGLE_QUOTE && nextChar === SINGLE_QUOTE) {
			autocompleted = expression.slice(0, cursorPosition - 1) + expression.slice(cursorPosition);
		}
	}

	if (char === DOUBLE_QUOTE) {
		if (isSoftChar(previousChar)) {
			autocompleted = expression.slice(0, cursorPosition) + DOUBLE_QUOTE + expression.slice(cursorPosition);
		}
		if (previousChar === DOUBLE_QUOTE && nextChar === DOUBLE_QUOTE) {
			autocompleted = expression.slice(0, cursorPosition - 1) + expression.slice(cursorPosition);
		}
	}

	if (char === RIGHT_PARENTHESIS && previousChar === LEFT_PARENTHESIS && nextChar === RIGHT_PARENTHESIS) {
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

export { autocompleteExpression };
