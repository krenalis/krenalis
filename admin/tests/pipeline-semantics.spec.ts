import { test, expect } from '@playwright/test';
import Type, { IntType, ObjectType, Property } from '../src/lib/api/types/types';
import TransformedConnection from '../src/lib/core/connection';
import { getIntegerTypeRange } from '../src/components/helpers/types';
import {
	FILTER_OPERATORS,
	computeDefaultTransformationFunction,
	flattenSchema,
	getCompatibleFilterOperators,
	matchingSemanticsAreCompatibleWithDestination,
	propertyTypesAreEqual,
	typeSemanticsAreEqual,
	TransformedPipelineType,
	validateMatching,
	validateAndNormalizeFilterCondition,
} from '../src/lib/core/pipeline';

const matchingProperty = (type: Type, updateRequired = false): Property => ({
	name: 'matching',
	prefilled: '',
	type,
	createRequired: false,
	updateRequired,
	readOptional: false,
	nullable: false,
	displayName: '',
	description: '',
});

test('Validate matching semantics compatibility', () => {
	const plain = matchingProperty({ kind: 'string' });
	const phone = matchingProperty({ kind: 'string', semantics: 'phone' });
	const otherPhone = matchingProperty({ kind: 'string', semantics: 'phone' });
	const email = matchingProperty({ kind: 'string', semantics: 'email' });
	const durationSeconds = matchingProperty({
		kind: 'int',
		semantics: 'duration',
		unit: 'second',
		bitSize: 64,
		unsigned: false,
	});
	const durationSeconds32 = matchingProperty({
		kind: 'int',
		semantics: 'duration',
		unit: 'second',
		bitSize: 32,
		unsigned: false,
	});
	const durationMinutes = matchingProperty({
		kind: 'int',
		semantics: 'duration',
		unit: 'minute',
		bitSize: 64,
		unsigned: false,
	});

	expect(propertyTypesAreEqual(phone.type, otherPhone.type)).toBe(true);
	expect(propertyTypesAreEqual(plain.type, phone.type)).toBe(false);
	expect(
		propertyTypesAreEqual(
			{ kind: 'string', semantics: 'country', format: 'alpha-2' },
			{ kind: 'string', semantics: 'country', format: 'alpha-3' },
		),
	).toBe(false);
	expect(
		typeSemanticsAreEqual(
			{ kind: 'decimal', semantics: 'money' },
			{ kind: 'decimal', semantics: 'money', currency: 'EUR' },
		),
	).toBe(false);
	expect(
		typeSemanticsAreEqual(
			{ kind: 'decimal', semantics: 'measurement', unit: 'kg' },
			{ kind: 'decimal', semantics: 'measurement', unit: 'g' },
		),
	).toBe(false);

	expect(() => validateMatching(phone, otherPhone)).not.toThrow();
	expect(() => validateMatching(durationSeconds, durationSeconds32)).not.toThrow();
	expect(() => validateMatching(plain, matchingProperty({ kind: 'uuid' }))).not.toThrow();
	expect(() => validateMatching(plain, phone)).not.toThrow();
	expect(() => validateMatching(phone, plain)).not.toThrow();
	expect(() => validateMatching(plain, email)).not.toThrow();
	expect(() => validateMatching(email, plain)).not.toThrow();
	expect(() => validateMatching(durationSeconds, plain)).not.toThrow();
	expect(() =>
		validateMatching(matchingProperty({ kind: 'int', bitSize: 64, unsigned: false }), durationSeconds),
	).not.toThrow();
	expect(() => validateMatching(email, phone)).toThrow(
		'Input and output matching properties do not have the same semantics and options',
	);
	expect(() => validateMatching(durationSeconds, durationMinutes)).toThrow(
		'Input and output matching properties do not have the same semantics and options',
	);

	expect(matchingSemanticsAreCompatibleWithDestination(phone.type, undefined, 'UpdateOnly')).toBe(true);
	expect(matchingSemanticsAreCompatibleWithDestination(phone.type, email, 'CreateOnly')).toBe(false);
	expect(matchingSemanticsAreCompatibleWithDestination(phone.type, email, 'CreateOrUpdate')).toBe(false);
	expect(matchingSemanticsAreCompatibleWithDestination(phone.type, email, 'UpdateOnly')).toBe(true);
	expect(
		matchingSemanticsAreCompatibleWithDestination(phone.type, matchingProperty(email.type, true), 'UpdateOnly'),
	).toBe(false);
	expect(
		matchingSemanticsAreCompatibleWithDestination(
			{ kind: 'string', maxLength: 10 },
			matchingProperty({ kind: 'string', maxLength: 20 }, true),
			'UpdateOnly',
		),
	).toBe(true);
});

test('Year filters keep numeric operators when semantics change', () => {
	const year: IntType = { kind: 'int', bitSize: 16, unsigned: false, semantics: 'year', minimum: 1, maximum: 9999 };
	const plain: IntType = { ...year, semantics: undefined };
	const duration: IntType = { ...year, semantics: 'duration', unit: 'year' };
	const plainOperators = getCompatibleFilterOperators(
		flattenSchema({ kind: 'object', properties: [matchingProperty(plain)] }).matching,
		false,
	).map((index) => FILTER_OPERATORS[index]);
	for (const type of [year, plain, duration]) {
		const property = matchingProperty(type);
		const flat = flattenSchema({ kind: 'object', properties: [property] });
		const operators = getCompatibleFilterOperators(flat.matching, false).map((index) => FILTER_OPERATORS[index]);
		if (type !== plain) {
			expect(operators).toEqual(plainOperators);
		}
		expect(operators).toContain('is less than');
		expect(operators).not.toContain('is before');
		const condition = { property: 'matching', operator: 'is less than' as const, values: ['10000'] };
		expect(validateAndNormalizeFilterCondition(condition, flat.matching, 'matching', '')).toEqual(condition);
	}
});

test('Integer transformation examples respect ranges and language representations', () => {
	interface ExampleCase {
		name: string;
		type: IntType;
		value: number | bigint;
	}
	const year: IntType = { kind: 'int', bitSize: 16, unsigned: false, semantics: 'year' };
	const cases: ExampleCase[] = [
		{ name: 'integer', type: { ...year, semantics: undefined }, value: 0 },
		{ name: '64-bit integer', type: { ...year, semantics: undefined, bitSize: 64 }, value: BigInt(0) },
		{ name: 'year', type: year, value: 2000 },
		{ name: '64-bit year', type: { ...year, bitSize: 64 }, value: BigInt(2000) },
		{ name: '8-bit year', type: { ...year, bitSize: 8 }, value: 127 },
		{ name: 'unsigned 8-bit year', type: { ...year, bitSize: 8, unsigned: true }, value: 255 },
		{ name: 'earlier years', type: { ...year, minimum: 1900, maximum: 1999 }, value: 1999 },
		{ name: 'later years', type: { ...year, minimum: 2024, maximum: 2100 }, value: 2024 },
		{
			name: 'unsigned 64-bit later years',
			type: { ...year, bitSize: 64, unsigned: true, minimum: 2024, maximum: 2100 },
			value: BigInt(2024),
		},
		{ name: 'positive integers', type: { ...year, semantics: undefined, minimum: 10, maximum: 20 }, value: 10 },
	];
	const connection = { isDestination: true } as TransformedConnection;
	const pipelineType = { target: 'Event' } as TransformedPipelineType;
	for (const example of cases) {
		const schema: ObjectType = {
			kind: 'object',
			properties: [{ ...matchingProperty(example.type), createRequired: true }],
		};
		const javascript = computeDefaultTransformationFunction(schema, 'JavaScript', connection, pipelineType);
		const result = new Function(`${javascript}; return transform({});`)();
		expect(result.matching, example.name).toBe(example.value);
		const python = computeDefaultTransformationFunction(schema, 'Python', connection, pipelineType);
		expect(python, example.name).toContain(`"matching": ${example.value},`);
	}
});

test('Year integer ranges respect calendar and bit-size bounds', () => {
	for (const bitSize of [8, 16, 64] as const) {
		for (const unsigned of [false, true]) {
			const type: IntType = { kind: 'int', bitSize, unsigned, semantics: 'year' };
			const maximum = bitSize === 8 ? (unsigned ? 255 : 127) : 9999;
			expect(getIntegerTypeRange(type)).toEqual([BigInt(1), BigInt(maximum)]);
		}
	}
});
