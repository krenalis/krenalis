type Role = 'Both' | 'Source' | 'Destination';

type TypeKind =
	| 'Boolean'
	| 'Int'
	| 'Uint'
	| 'Float'
	| 'Decimal'
	| 'DateTime'
	| 'Date'
	| 'Time'
	| 'Year'
	| 'UUID'
	| 'JSON'
	| 'Inet'
	| 'Text'
	| 'Array'
	| 'Object'
	| 'Map';

type IntBitSize = 8 | 16 | 24 | 32 | 64;

type FloatBitSize = 32 | 64;

interface Property {
	name: string;
	placeholder: string;
	role: Role;
	type: Type;
	createRequired: boolean;
	updateRequired: boolean;
	readOptional: boolean;
	nullable: boolean;
	description: string;
}

type Type =
	| BooleanType
	| IntType
	| UintType
	| FloatType
	| DecimalType
	| DateTimeType
	| DateType
	| TimeType
	| YearType
	| UUIDType
	| JSONType
	| InetType
	| TextType
	| ArrayType
	| ObjectType
	| MapType;

interface BooleanType {
	kind: 'Boolean';
}

interface IntType {
	kind: 'Int';
	bitSize: IntBitSize;
	minimum?: number;
	maximum?: number;
}

interface UintType {
	kind: 'Uint';
	bitSize: IntBitSize;
	minimum?: number;
	maximum?: number;
}

interface FloatType {
	kind: 'Float';
	bitSize: FloatBitSize;
	real?: boolean;
	minimum?: number;
	maximum?: number;
}

interface DecimalType {
	kind: 'Decimal';
	minimum?: number;
	maximum?: number;
	precision?: number;
	scale?: number;
}

interface DateTimeType {
	kind: 'DateTime';
	layout?: string;
}

interface DateType {
	kind: 'Date';
	layout?: string;
}

interface TimeType {
	kind: 'Time';
}

interface YearType {
	kind: 'Year';
}

interface UUIDType {
	kind: 'UUID';
}

interface JSONType {
	kind: 'JSON';
	charLen?: number;
}

interface InetType {
	kind: 'Inet';
}

interface TextType {
	kind: 'Text';
	byteLen?: number;
	charLen?: number;
	regexp?: string;
	values?: string[];
}

interface ArrayType {
	kind: 'Array';
	minElements?: number;
	maxElements?: number;
	uniqueElements?: boolean;
	elementType?: Type;
}

interface ObjectType {
	kind: 'Object';
	properties?: Property[];
}

interface MapType {
	kind: 'Map';
	elementType?: Type;
}

const typeKindToIconName: Record<TypeKind, string> = {
	Boolean: 'type-bold',
	Int: '123',
	Uint: '123',
	Float: '123',
	Decimal: '123',
	DateTime: 'calendar-date',
	Date: 'calendar-date',
	Time: 'clock',
	Year: '123',
	UUID: 'fonts',
	JSON: 'filetype-json',
	Inet: 'fonts',
	Text: 'fonts',
	Array: 'input-cursor',
	Object: 'braces',
	Map: 'braces-asterisk',
};

export { typeKindToIconName };
export default Type;
export type {
	Property,
	ArrayType,
	TextType,
	ObjectType,
	IntType,
	UintType,
	DecimalType,
	FloatType,
	Role,
	TypeKind,
	IntBitSize,
	FloatBitSize,
	MapType,
};
