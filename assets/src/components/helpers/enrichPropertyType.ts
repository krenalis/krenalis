import Type, { ArrayType, TextType, IntType, UintType, FloatType } from '../../lib/api/types/types';

const enrichPropertyType = (type: Type) => {
	let kind: string = type.kind;
	if (kind === 'array') {
		const typ = type as ArrayType;
		kind = 'array(' + typ.elementType?.kind + ')';
	}
	if ('values' in type) {
		const typ = type as TextType;
		kind += ' (' + typ.values?.map((e) => '"' + e + '"').join(', ') + ')';
	}
	if (kind === 'int' || kind === 'uint' || kind === 'float') {
		const typ = type as IntType | UintType | FloatType;
		kind += `(${typ.bitSize})`;
	}
	return kind;
};

export { enrichPropertyType };
