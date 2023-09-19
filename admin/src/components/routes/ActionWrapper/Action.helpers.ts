import { TransformedAction, TransformedMapping } from '../../../lib/helpers/transformedAction';

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

const updateMappingProperty = (action: TransformedAction, name: string, value: string, error: string) => {
	const getAlternativeProperties = (name: string, mapping: TransformedMapping): string[] => {
		const indentation = mapping[name].indentation;
		const parentProperties: string[] = [];
		for (const k in mapping) {
			if (mapping[k].indentation! < indentation! && name.startsWith(k)) {
				parentProperties.push(k);
			}
		}
		const childrenProperties: string[] = [];
		for (const k in mapping) {
			if (mapping[k].indentation! > indentation! && k.startsWith(name)) {
				childrenProperties.push(k);
			}
		}
		return [...parentProperties, ...childrenProperties];
	};

	const a = { ...action };

	if (a.Mapping == null) return a;

	if (a.Mapping[name].value === '' && value !== '') {
		const alternativeProperties = getAlternativeProperties(name, a.Mapping);
		// disable
		for (const k in a.Mapping) {
			if (alternativeProperties.includes(k)) {
				a.Mapping[k].disabled = true;
			}
		}
	} else if (value === '') {
		let hasFilledSiblings = false;
		const { root, indentation } = a.Mapping[name];
		for (const k in a.Mapping) {
			if (
				k !== name &&
				a.Mapping[k].root === root &&
				a.Mapping[k].indentation === indentation &&
				a.Mapping[k].value !== ''
			) {
				hasFilledSiblings = true;
			}
		}
		if (!hasFilledSiblings) {
			// enable
			const alternativeProperties = getAlternativeProperties(name, a.Mapping);
			for (const k in a.Mapping) {
				if (alternativeProperties.includes(k)) {
					a.Mapping[k].disabled = false;
				}
			}
		}
	}

	a.Mapping[name].error = error;
	a.Mapping[name].value = value;
	return a;
};

interface autocompleteExpressionReturnValue {
	autocompleted: string;
	cursorPosition: number;
}
const autocompleteExpression = (
	expression: string,
	currentCursorPosition: number,
): autocompleteExpressionReturnValue => {
	const char = expression[currentCursorPosition - 1];
	const previousChar = expression[currentCursorPosition - 2];
	const nextChar = expression[currentCursorPosition];

	let autocompleted: string = expression;
	let cursorPosition: number = currentCursorPosition;

	if (char === LEFT_PARENTHESIS) {
		autocompleted =
			autocompleted.slice(0, cursorPosition) + RIGHT_PARENTHESIS + autocompleted.slice(cursorPosition);
	}

	if (char === SINGLE_QUOTE) {
		if (isSoftChar(previousChar)) {
			autocompleted = autocompleted.slice(0, cursorPosition) + SINGLE_QUOTE + autocompleted.slice(cursorPosition);
		}
		if (previousChar === SINGLE_QUOTE && nextChar === SINGLE_QUOTE) {
			autocompleted = autocompleted.slice(0, cursorPosition - 1) + autocompleted.slice(cursorPosition);
		}
	}

	if (char === DOUBLE_QUOTE) {
		if (isSoftChar(previousChar)) {
			autocompleted = autocompleted.slice(0, cursorPosition) + DOUBLE_QUOTE + autocompleted.slice(cursorPosition);
		}
		if (previousChar === DOUBLE_QUOTE && nextChar === DOUBLE_QUOTE) {
			autocompleted = autocompleted.slice(0, cursorPosition - 1) + autocompleted.slice(cursorPosition);
		}
	}

	if (char === RIGHT_PARENTHESIS && previousChar === LEFT_PARENTHESIS && nextChar === RIGHT_PARENTHESIS) {
		autocompleted = autocompleted.slice(0, cursorPosition - 1) + autocompleted.slice(cursorPosition);
	}

	return {
		autocompleted,
		cursorPosition,
	};
};

const isSoftChar = (char: string): boolean => {
	if (SOFT_CHARS.includes(char)) {
		return true;
	}
	return false;
};

export { updateMappingProperty, autocompleteExpression };
