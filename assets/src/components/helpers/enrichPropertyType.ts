import Type, { ArrayType, TextType, IntType, UintType, FloatType } from '../../lib/api/types/types';

const enrichPropertyType = (type: Type) => {
	let kind: string = type.kind;
	if (kind === 'Array') {
		const typ = type as ArrayType;
		kind = 'Array(' + typ.elementType?.kind + ')';
	}
	if ('values' in type) {
		const typ = type as TextType;
		kind += ' (' + typ.values?.map((e) => '"' + e + '"').join(', ') + ')';
	}
	if (kind === 'Int' || kind === 'Uint' || kind === 'Float') {
		const typ = type as IntType | UintType | FloatType;
		kind += `(${typ.bitSize})`;
	}
	return kind;
};

export { enrichPropertyType };
