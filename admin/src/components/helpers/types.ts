import Type, { DurationUnit, Semantic, UnitOfMeasure } from '../../lib/api/types/types';

interface UnitOfMeasureOption {
	groupLabel?: string;
	label: string;
	value: UnitOfMeasure;
}

interface DurationUnitOption {
	label: string;
	symbol: string;
	value: DurationUnit;
}

interface ProfileSchemaTypePresentation {
	metadata?: string;
	primary: string;
}

const UNIT_OF_MEASURE_OPTIONS: UnitOfMeasureOption[] = [
	{ value: 'mm', label: 'Millimetre', groupLabel: 'Length' },
	{ value: 'cm', label: 'Centimetre' },
	{ value: 'm', label: 'Metre' },
	{ value: 'km', label: 'Kilometre' },
	{ value: 'in', label: 'Inch' },
	{ value: 'ft', label: 'Foot' },
	{ value: 'yd', label: 'Yard' },
	{ value: 'mi', label: 'Mile' },
	{ value: 'g', label: 'Gram', groupLabel: 'Weight' },
	{ value: 'kg', label: 'Kilogram' },
	{ value: 'oz', label: 'Ounce' },
	{ value: 'lb', label: 'Pound' },
	{ value: 'B', label: 'Byte', groupLabel: 'Data size' },
	{ value: 'kB', label: 'Kilobyte' },
	{ value: 'MB', label: 'Megabyte' },
	{ value: 'GB', label: 'Gigabyte' },
	{ value: '°C', label: 'Degree Celsius', groupLabel: 'Temperature' },
	{ value: '°F', label: 'Degree Fahrenheit' },
	{ value: 'mL', label: 'Millilitre', groupLabel: 'Volume' },
	{ value: 'L', label: 'Litre' },
];

const DURATION_UNIT_OPTIONS: DurationUnitOption[] = [
	{ value: 'millisecond', label: 'Milliseconds', symbol: 'ms' },
	{ value: 'second', label: 'Seconds', symbol: 's' },
	{ value: 'minute', label: 'Minutes', symbol: 'min' },
	{ value: 'hour', label: 'Hours', symbol: 'h' },
	{ value: 'day', label: 'Days', symbol: 'd' },
	{ value: 'week', label: 'Weeks', symbol: 'wk' },
];

function getPropertyValueType(type: Type | null): Type | null {
	if (type?.kind === 'array' || type?.kind === 'map') {
		return type.elementType;
	}
	return type;
}

function getProfileSchemaTypePresentation(type: Type): ProfileSchemaTypePresentation {
	const valueType = getPropertyValueType(type);
	const semantic = getTypeSemantic(valueType);
	if (semantic == null) {
		return { primary: toProfileSchemaPhysicalType(type) };
	}

	let primary = toSemanticLabel(semantic);
	switch (semantic) {
		case 'country': {
			const letters = valueType.kind === 'string' && valueType.format === 'alpha-2' ? 2 : 3;
			primary += ` — ${letters}-letter ISO code`;
			break;
		}
		case 'money':
			if (valueType.kind === 'decimal' && valueType.currency != null) {
				primary += ` — ${valueType.currency}`;
			}
			break;
		case 'measurement':
			if ('unit' in valueType) {
				primary += ` — ${valueType.unit}`;
			}
			break;
		case 'duration': {
			const unit = DURATION_UNIT_OPTIONS.find((option) => 'unit' in valueType && option.value === valueType.unit);
			if (unit != null) {
				primary += ` — ${unit.symbol}`;
			}
			break;
		}
	}
	if (type.kind === 'array' || type.kind === 'map') {
		primary = `${type.kind} of ${primary}`;
	}
	return { primary, metadata: toProfileSchemaPhysicalType(valueType) };
}

function getTypeSemantic(type: Type | null): Semantic | undefined {
	return type != null && 'semantic' in type ? type.semantic : undefined;
}

function isSuitableAsIdentifier(type: Type): boolean {
	switch (type.kind) {
		case 'string':
		case 'int':
		case 'uuid':
		case 'ip':
			return true;
		case 'decimal':
			return (type.scale ?? 0) === 0;
		default:
			return false;
	}
}

function replacePropertyValueType(type: Type, valueType: Type): Type {
	if (type.kind === 'array' || type.kind === 'map') {
		return { ...type, elementType: valueType };
	}
	return valueType;
}

function toCompactPhysicalType(type: Type): string {
	switch (type.kind) {
		case 'array':
		case 'map':
			return `${type.kind} of ${toCompactPhysicalType(type.elementType)}`;
		case 'int':
			return type.unsigned ? 'unsigned int' : 'int';
		default:
			return type.kind;
	}
}

function toProfileSchemaPhysicalType(type: Type): string {
	if (type.kind === 'array' || type.kind === 'map') {
		return `${type.kind} of ${toProfileSchemaPhysicalType(type.elementType)}`;
	}
	const normalizedType =
		type.kind === 'int' && type.unsigned && type.minimum == null ? { ...type, minimum: 0 } : type;
	return toKrenalisStringType(normalizedType, undefined, ' · ');
}

function toSemanticLabel(semantic: Semantic): string {
	switch (semantic) {
		case 'phone':
			return 'phone number';
		case 'url':
			return 'URL';
		case 'email':
		case 'country':
		case 'money':
		case 'percentage':
		case 'measurement':
		case 'duration':
			return semantic;
		default:
			throw new Error(`unknown semantic ${semantic satisfies never}`);
	}
}

function toKrenalisStringType(type: Type, nullable?: boolean, firstConstraintSeparator = ', ') {
	let t: string;
	const constraints: string[] = [];

	if (type.kind === 'string') {
		t = `${type.kind}`;
		if (type.values != null) {
			t += ' (' + type.values.map((e) => '"' + e + '"').join(', ') + ')';
		}
		if (type.pattern != null) {
			constraints.push(`must match /${type.pattern}/`);
		}
		if (type.maxBytes != null) {
			constraints.push(`max ${type.maxBytes} bytes`);
		}
		if (type.maxLength != null) {
			constraints.push(`max ${type.maxLength} chars`);
		}
	} else if (type.kind === 'int') {
		const label = type.unsigned ? 'unsigned int' : 'int';
		t = `${label}(${type.bitSize})`;
		if (type.minimum != null) {
			constraints.push(`min ${type.minimum}`);
		}
		if (type.maximum != null) {
			constraints.push(`max ${type.maximum}`);
		}
	} else if (type.kind === 'float') {
		t = `${type.kind}(${type.bitSize})`;
		if (type.real != null && type.real) {
			constraints.push('real');
		}
		if (type.minimum != null) {
			constraints.push(`min ${type.minimum}`);
		}
		if (type.maximum != null) {
			constraints.push(`max ${type.maximum}`);
		}
	} else if (type.kind === 'decimal') {
		t = `decimal(${type.precision},${type.scale || 0})`;
		if (type.minimum != null) {
			constraints.push(`min ${type.minimum}`);
		}
		if (type.maximum != null) {
			constraints.push(`max ${type.maximum}`);
		}
	} else if (type.kind === 'array') {
		t = `${type.kind} of ${toKrenalisStringType(type.elementType)}`;
		if (type.minElements != null) {
			constraints.push(`min ${type.minElements} elements`);
		}
		if (type.maxElements != null) {
			constraints.push(`max ${type.maxElements} elements`);
		}
		if (type.uniqueElements != null && type.uniqueElements) {
			constraints.push('unique elements');
		}
	} else if (type.kind === 'map') {
		t = `${type.kind} of ${toKrenalisStringType(type.elementType)}`;
	} else {
		t = type.kind;
	}

	if (constraints.length !== 0) {
		t += firstConstraintSeparator + constraints.join(', ');
	}

	if (nullable) {
		t += ' | null';
	}

	return t;
}

function toJavascriptType(type: Type, preserveJSON: boolean, nullable?: boolean) {
	let t: string;

	const kind = type.kind;
	switch (kind) {
		case 'boolean':
			t = 'boolean';
			break;
		case 'int':
			if (type.bitSize === 64) {
				t = 'bigint';
			} else {
				t = `number (${kind})`;
			}
			break;
		case 'float':
			t = 'number';
			break;
		case 'decimal':
			t = 'string';
			break;
		case 'datetime':
		case 'date':
		case 'time':
			t = 'Date';
			break;
		case 'year':
			t = 'number';
			break;
		case 'uuid':
			t = 'string';
			break;
		case 'json':
			if (preserveJSON) {
				t = 'string (JSON)';
			} else {
				t = 'any';
			}
			break;
		case 'ip':
			t = 'string';
			break;
		case 'string':
			t = 'string';
			break;
		case 'array':
			const arrayElementType = toJavascriptType(type.elementType, preserveJSON);
			t = `${arrayElementType}[]`;
			break;
		case 'object':
			t = 'object';
			break;
		case 'map':
			const mapElementType = toJavascriptType(type.elementType, preserveJSON);
			t = `object with ${mapElementType} values`;
			break;
		default:
			throw new Error(`schema contains unknown property kind ${kind}`);
	}

	if (nullable) {
		t += ' | null';
	}

	return t;
}

function toPythonType(type: Type, preserveJSON: boolean, nullable?: boolean) {
	let t: string;

	const kind = type.kind;
	switch (kind) {
		case 'boolean':
			t = 'bool';
			break;
		case 'int':
			t = 'int';
			break;
		case 'float':
			t = 'float';
			break;
		case 'decimal':
			t = 'decimal.Decimal';
			break;
		case 'datetime':
			t = 'datetime.datetime';
			break;
		case 'date':
			t = 'datetime.date';
			break;
		case 'time':
			t = 'datetime.time';
			break;
		case 'year':
			t = 'int';
			break;
		case 'uuid':
			t = 'uuid.UUID';
			break;
		case 'json':
			if (preserveJSON) {
				t = 'str (JSON)';
			} else {
				t = 'Any';
			}
			break;
		case 'ip':
			t = 'str';
			break;
		case 'string':
			t = 'str';
			break;
		case 'array':
			const arrayElementType = toPythonType(type.elementType, preserveJSON);
			t = `list[${arrayElementType}]`;
			break;
		case 'object':
			t = 'dict';
			break;
		case 'map':
			const mapElementType = toPythonType(type.elementType, preserveJSON);
			t = `dict[str, ${mapElementType}]`;
			break;
		default:
			throw new Error(`schema contains unknown property kind ${kind}`);
	}

	if (nullable) {
		t += ' | None';
	}

	return t;
}

export {
	DURATION_UNIT_OPTIONS,
	UNIT_OF_MEASURE_OPTIONS,
	getPropertyValueType,
	getProfileSchemaTypePresentation,
	getTypeSemantic,
	isSuitableAsIdentifier,
	replacePropertyValueType,
	toCompactPhysicalType,
	toProfileSchemaPhysicalType,
	toSemanticLabel,
	toKrenalisStringType,
	toJavascriptType,
	toPythonType,
};
