type Placeholder = string | Map<string, string> | null;

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
	flat: boolean;
}

type Type =
	| BooleanType
	| IntType
	| Int8Type
	| Int16Type
	| Int24Type
	| Int64Type
	| UIntType
	| UInt8Type
	| UInt16Type
	| UInt24Type
	| UInt64Type
	| FloatType
	| Float32Type
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
	minimum?: number;
	maximum?: number;
}

interface Int8Type {
	name: 'Int8';
	minimum?: number;
	maximum?: number;
}

interface Int16Type {
	name: 'Int16';
	minimum?: number;
	maximum?: number;
}

interface Int24Type {
	name: 'Int24';
	minimum?: number;
	maximum?: number;
}

interface Int64Type {
	name: 'Int64';
	minimum?: number;
	maximum?: number;
}

interface UIntType {
	name: 'UInt';
	minimum?: number;
	maximum?: number;
}

interface UInt8Type {
	name: 'UInt8';
	minimum?: number;
	maximum?: number;
}

interface UInt16Type {
	name: 'UInt16';
	minimum: number;
	maximum: number;
}

interface UInt24Type {
	name: 'UInt24';
	minimum?: number;
	maximum?: number;
}

interface UInt64Type {
	name: 'UInt64';
	minimum?: number;
	maximum?: number;
}

interface FloatType {
	name: 'Float';
	real?: boolean;
	minimum?: number;
	maximum?: number;
}

interface Float32Type {
	name: 'Float32';
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
	enum?: string[];
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
