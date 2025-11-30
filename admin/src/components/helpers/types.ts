import Type, { StringType } from '../../lib/api/types/types';

function toMeergoStringType(type: Type, nullable?: boolean) {
	let t: string;

	if (type.kind === 'int' || type.kind === 'uint' || type.kind === 'float') {
		t = `${type.kind}(${type.bitSize})`;
	} else if (type.kind === 'decimal') {
		t = `decimal(${type.precision}, ${type.scale || 0})`;
	} else if (type.kind === 'array' || type.kind === 'map') {
		t = `${type.kind} of ${toMeergoStringType(type.elementType)}`;
	} else if ('values' in type) {
		const typ = type as StringType;
		t = type.kind + ' (' + typ.values?.map((e) => '"' + e + '"').join(', ') + ')';
	} else {
		t = type.kind;
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
		case 'uint':
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
		case 'uint':
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

export { toMeergoStringType, toJavascriptType, toPythonType };
