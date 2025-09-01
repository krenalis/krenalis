type Role = 'Both' | 'Source' | 'Destination';

type TypeKind =
	| 'boolean'
	| 'int'
	| 'uint'
	| 'float'
	| 'decimal'
	| 'datetime'
	| 'date'
	| 'time'
	| 'year'
	| 'uuid'
	| 'json'
	| 'inet'
	| 'text'
	| 'array'
	| 'object'
	| 'map';

type IntBitSize = 8 | 16 | 24 | 32 | 64;

type FloatBitSize = 32 | 64;

interface Property {
	name: string;
	prefilled: string;
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
	kind: 'boolean';
}

interface IntType {
	kind: 'int';
	bitSize: IntBitSize;
	minimum?: number;
	maximum?: number;
}

interface UintType {
	kind: 'uint';
	bitSize: IntBitSize;
	minimum?: number;
	maximum?: number;
}

interface FloatType {
	kind: 'float';
	bitSize: FloatBitSize;
	real?: boolean;
	minimum?: number;
	maximum?: number;
}

interface DecimalType {
	kind: 'decimal';
	minimum?: number;
	maximum?: number;
	precision?: number;
	scale?: number;
}

interface DateTimeType {
	kind: 'datetime';
	layout?: string;
}

interface DateType {
	kind: 'date';
	layout?: string;
}

interface TimeType {
	kind: 'time';
}

interface YearType {
	kind: 'year';
}

interface UUIDType {
	kind: 'uuid';
}

interface JSONType {
	kind: 'json';
	charLen?: number;
}

interface InetType {
	kind: 'inet';
}

interface TextType {
	kind: 'text';
	byteLen?: number;
	charLen?: number;
	regexp?: string;
	values?: string[];
}

interface ArrayType {
	kind: 'array';
	minElements?: number;
	maxElements?: number;
	uniqueElements?: boolean;
	elementType?: Type;
}

interface ObjectType {
	kind: 'object';
	properties?: Property[];
}

interface MapType {
	kind: 'map';
	elementType?: Type;
}

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
