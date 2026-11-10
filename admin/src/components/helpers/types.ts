import Type, { Semantics, Unit } from '../../lib/api/types/types';

interface UnitOption {
	groupLabel?: string;
	isDurationUnit?: boolean;
	label: string;
	symbol: string;
	value: Unit;
}

interface ProfileSchemaTypePresentation {
	metadata?: string;
	primary: string;
}

const UNIT_OPTIONS: UnitOption[] = [
	{ value: 'mm', label: 'Millimetre', symbol: 'mm', groupLabel: 'Length' },
	{ value: 'cm', label: 'Centimetre', symbol: 'cm' },
	{ value: 'm', label: 'Metre', symbol: 'm' },
	{ value: 'km', label: 'Kilometre', symbol: 'km' },
	{ value: 'in', label: 'Inch', symbol: 'in' },
	{ value: 'ft', label: 'Foot', symbol: 'ft' },
	{ value: 'yd', label: 'Yard', symbol: 'yd' },
	{ value: 'mi', label: 'Mile', symbol: 'mi' },
	{ value: 'g', label: 'Gram', symbol: 'g', groupLabel: 'Weight' },
	{ value: 'kg', label: 'Kilogram', symbol: 'kg' },
	{ value: 'oz', label: 'Ounce', symbol: 'oz' },
	{ value: 'lb', label: 'Pound', symbol: 'lb' },
	{ value: 'B', label: 'Byte', symbol: 'B', groupLabel: 'Data size' },
	{ value: 'kB', label: 'Kilobyte', symbol: 'kB' },
	{ value: 'MB', label: 'Megabyte', symbol: 'MB' },
	{ value: 'GB', label: 'Gigabyte', symbol: 'GB' },
	{ value: '°C', label: 'Degree Celsius', symbol: '°C', groupLabel: 'Temperature' },
	{ value: '°F', label: 'Degree Fahrenheit', symbol: '°F' },
	{ value: 'mL', label: 'Millilitre', symbol: 'mL', groupLabel: 'Volume' },
	{ value: 'L', label: 'Litre', symbol: 'L' },
	{ value: 'millisecond', label: 'Millisecond', symbol: 'ms', groupLabel: 'Time', isDurationUnit: true },
	{ value: 'second', label: 'Second', symbol: 's', isDurationUnit: true },
	{ value: 'minute', label: 'Minute', symbol: 'min', isDurationUnit: true },
	{ value: 'hour', label: 'Hour', symbol: 'h', isDurationUnit: true },
	{ value: 'day', label: 'Day', symbol: 'd', isDurationUnit: true },
	{ value: 'week', label: 'Week', symbol: 'wk', isDurationUnit: true },
	{ value: 'month', label: 'Month', symbol: 'mo', isDurationUnit: true },
	{ value: 'year', label: 'Year', symbol: 'yr', isDurationUnit: true },
];

const DURATION_UNIT_OPTIONS = UNIT_OPTIONS.filter((option) => option.isDurationUnit === true);

function getPropertyValueType(type: Type | null): Type | null {
	if (type?.kind === 'array' || type?.kind === 'map') {
		return type.elementType;
	}
	return type;
}

function getProfileSchemaTypePresentation(type: Type): ProfileSchemaTypePresentation {
	const valueType = getPropertyValueType(type);
	const semantics = getTypeSemantics(valueType);
	if (semantics == null) {
		return { primary: toProfileSchemaPhysicalType(type) };
	}

	let primary = toSemanticsLabel(semantics);
	switch (semantics) {
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
		case 'duration': {
			const unit = UNIT_OPTIONS.find((option) => 'unit' in valueType && option.value === valueType.unit);
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

function getTypeSemantics(type: Type | null): Semantics | undefined {
	return type != null && 'semantics' in type ? type.semantics : undefined;
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

function toSemanticsLabel(semantics: Semantics): string {
	switch (semantics) {
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
		case 'year':
			return semantics;
		default:
			throw new Error(`unknown property semantics ${semantics satisfies never}`);
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
	UNIT_OPTIONS,
	getPropertyValueType,
	getProfileSchemaTypePresentation,
	getTypeSemantics,
	isSuitableAsIdentifier,
	replacePropertyValueType,
	toCompactPhysicalType,
	toProfileSchemaPhysicalType,
	toSemanticsLabel,
	toKrenalisStringType,
	toJavascriptType,
	toPythonType,
};
