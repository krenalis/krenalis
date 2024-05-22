import Type, { ArrayType, TextType, IntType, UintType, FloatType } from '../../lib/api/types/types';

const enrichPropertyType = (type: Type) => {
	let typeName: string = type.name;
	if (typeName === 'Array') {
		const typ = type as ArrayType;
		typeName = 'Array(' + typ.itemType?.name + ')';
	}
	if ('values' in type) {
		const typ = type as TextType;
		typeName += ' (' + typ.values?.map((e) => '"' + e + '"').join(', ') + ')';
	}
	if (typeName === 'Int' || typeName === 'Uint' || typeName === 'Float') {
		const typ = type as IntType | UintType | FloatType;
		typeName += `(${typ.bitSize})`;
	}
	return typeName;
};

export { enrichPropertyType };
