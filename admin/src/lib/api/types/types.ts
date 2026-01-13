type Role = 'Both' | 'Source' | 'Destination';

type TypeKind =
	| 'boolean'
	| 'int'
	| 'float'
	| 'decimal'
	| 'datetime'
	| 'date'
	| 'time'
	| 'year'
	| 'uuid'
	| 'json'
	| 'ip'
	| 'string'
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
	| StringType
	| BooleanType
	| IntType
	| FloatType
	| DecimalType
	| DateTimeType
	| DateType
	| TimeType
	| YearType
	| UUIDType
	| JSONType
	| IPType
	| ArrayType
	| ObjectType
	| MapType;

interface StringType {
	kind: 'string';
	maxBytes?: number;
	maxLength?: number;
	pattern?: string;
	values?: string[];
}

interface BooleanType {
	kind: 'boolean';
}

interface IntType {
	kind: 'int';
	bitSize: IntBitSize;
	unsigned: boolean;
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
}

interface DateType {
	kind: 'date';
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
}

interface IPType {
	kind: 'ip';
}

interface ArrayType {
	kind: 'array';
	minElements?: number;
	maxElements?: number;
	uniqueElements?: boolean;
	elementType: Type;
}

interface ObjectType {
	kind: 'object';
	properties?: Property[];
}

interface MapType {
	kind: 'map';
	elementType: Type;
}

export default Type;
export type {
	Property,
	ArrayType,
	StringType,
	ObjectType,
	IntType,
	DecimalType,
	FloatType,
	Role,
	TypeKind,
	IntBitSize,
	FloatBitSize,
	MapType,
};
