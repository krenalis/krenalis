type Role = 'Both' | 'Source' | 'Destination';

type TypeName =
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
	label: string;
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
	name: 'Boolean';
}

interface IntType {
	name: 'Int';
	bitSize: IntBitSize;
	minimum?: number;
	maximum?: number;
}

interface UintType {
	name: 'Uint';
	bitSize: IntBitSize;
	minimum?: number;
	maximum?: number;
}

interface FloatType {
	name: 'Float';
	bitSize: FloatBitSize;
	real?: boolean;
	minimum?: number;
	maximum?: number;
}

interface DecimalType {
	name: 'Decimal';
	minimum?: number;
	maximum?: number;
	precision?: number;
	scale?: number;
}

interface DateTimeType {
	name: 'DateTime';
	layout?: string;
}

interface DateType {
	name: 'Date';
	layout?: string;
}

interface TimeType {
	name: 'Time';
}

interface YearType {
	name: 'Year';
}

interface UUIDType {
	name: 'UUID';
}

interface JSONType {
	name: 'JSON';
	charLen?: number;
}

interface InetType {
	name: 'Inet';
}

interface TextType {
	name: 'Text';
	byteLen?: number;
	charLen?: number;
	regexp?: string;
	values?: string[];
}

interface ArrayType {
	name: 'Array';
	minElements?: number;
	maxElements?: number;
	uniqueElements?: boolean;
	elementType?: Type;
}

interface ObjectType {
	name: 'Object';
	properties?: Property[];
}

interface MapType {
	name: 'Map';
	elementType?: Type;
}

const typeNameToIconName: Record<TypeName, string> = {
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

export { typeNameToIconName };
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
	TypeName,
	IntBitSize,
	FloatBitSize,
	MapType,
};
