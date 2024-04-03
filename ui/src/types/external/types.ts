// TODO: Placholder can even contain a map in the case of MapTypes. Currently
// this is not handled.
type Placeholder = string | null;

type Role = 'Both' | 'Source' | 'Destination';

interface Property {
	name: string;
	label: string;
	description: string;
	placeholder: Placeholder;
	role: Role;
	type: Type;
	required: boolean;
	nullable: boolean;
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
	bitSize: 8 | 16 | 24 | 32 | 64;
	minimum?: number;
	maximum?: number;
}

interface UintType {
	name: 'Uint';
	bitSize: 8 | 16 | 24 | 32 | 64;
	minimum?: number;
	maximum?: number;
}

interface FloatType {
	name: 'Float';
	bitSize: 32 | 64;
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
	minItems?: number;
	maxItems?: number;
	uniqueItems?: boolean;
	itemType?: Type;
}

interface ObjectType {
	name: 'Object';
	properties?: Property[];
}

interface MapType {
	name: 'Map';
	valueType?: Type;
}

export default Type;
export type { Property, ArrayType, TextType, ObjectType, IntType, UintType, FloatType };
