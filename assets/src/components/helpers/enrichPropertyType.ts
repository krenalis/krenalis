import Type, { ArrayType, TextType, IntType, UintType, FloatType, MapType } from '../../lib/api/types/types';

const enrichPropertyType = (type: Type) => {
	let kind: string = type.kind;
	if (kind === 'array') {
		const typ = type as ArrayType;
		kind = 'array of ' + typ.elementType?.kind;
	}
	if (kind === 'map') {
		const typ = type as MapType;
		kind = 'map of ' + typ.elementType?.kind;
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
