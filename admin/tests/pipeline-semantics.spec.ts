import { test, expect } from '@playwright/test';
import Type, { Property } from '../src/lib/api/types/types';
import {
	matchingSemanticsAreCompatibleWithDestination,
	propertyTypesAreEqual,
	typeSemanticsAreEqual,
	validateMatching,
} from '../src/lib/core/pipeline';

const matchingProperty = (type: Type, updateRequired = false): Property => ({
	name: 'matching',
	prefilled: '',
	role: 'Both',
	type,
	createRequired: false,
	updateRequired,
	readOptional: false,
	nullable: false,
	description: '',
});

test('Validate matching semantic compatibility', () => {
	const plain = matchingProperty({ kind: 'string' });
	const phone = matchingProperty({ kind: 'string', semantic: 'phone' });
	const otherPhone = matchingProperty({ kind: 'string', semantic: 'phone' });
	const email = matchingProperty({ kind: 'string', semantic: 'email' });
	const durationSeconds = matchingProperty({
		kind: 'int',
		semantic: 'duration',
		unit: 'second',
		bitSize: 64,
		unsigned: false,
	});
	const durationSeconds32 = matchingProperty({
		kind: 'int',
		semantic: 'duration',
		unit: 'second',
		bitSize: 32,
		unsigned: false,
	});
	const durationMinutes = matchingProperty({
		kind: 'int',
		semantic: 'duration',
		unit: 'minute',
		bitSize: 64,
		unsigned: false,
	});

	expect(propertyTypesAreEqual(phone.type, otherPhone.type)).toBe(true);
	expect(propertyTypesAreEqual(plain.type, phone.type)).toBe(false);
	expect(
		propertyTypesAreEqual(
			{ kind: 'string', semantic: 'country', format: 'alpha-2' },
			{ kind: 'string', semantic: 'country', format: 'alpha-3' },
		),
	).toBe(false);
	expect(
		typeSemanticsAreEqual(
			{ kind: 'decimal', semantic: 'money' },
			{ kind: 'decimal', semantic: 'money', currency: 'EUR' },
		),
	).toBe(false);
	expect(
		typeSemanticsAreEqual(
			{ kind: 'decimal', semantic: 'measurement', unit: 'kg' },
			{ kind: 'decimal', semantic: 'measurement', unit: 'g' },
		),
	).toBe(false);

	expect(() => validateMatching(phone, otherPhone)).not.toThrow();
	expect(() => validateMatching(durationSeconds, durationSeconds32)).not.toThrow();
	expect(() => validateMatching(plain, matchingProperty({ kind: 'uuid' }))).not.toThrow();
	expect(() => validateMatching(plain, phone)).toThrow(
		'Input and output matching properties do not have the same semantic and options',
	);
	expect(() => validateMatching(email, phone)).toThrow(
		'Input and output matching properties do not have the same semantic and options',
	);
	expect(() => validateMatching(durationSeconds, durationMinutes)).toThrow(
		'Input and output matching properties do not have the same semantic and options',
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
