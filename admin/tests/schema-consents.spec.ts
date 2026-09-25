import { expect, test } from '@playwright/test';
import Type, { ObjectType, Property } from '../src/lib/api/types/types';
import { ConsentPurpose } from '../src/lib/api/types/workspace';
import { getConsentPurposesByPropertyPath } from '../src/components/routes/Schema/SchemaPropertyConsent.helpers';
import { formatProfileConsentLocation, validateConsentKey } from '../src/utils/consentPurposePaths';

const property = (name: string, type: Type): Property => ({
	name,
	prefilled: '',
	role: 'Both',
	type,
	createRequired: false,
	updateRequired: false,
	readOptional: true,
	nullable: false,
	displayName: '',
	description: '',
});

const purpose = (id: string, property: string, jsonKey?: string): ConsentPurpose => ({
	id,
	name: `Purpose ${id}`,
	eventConsentLocations: [],
	profileConsentLocation: property === '' ? null : { property, ...(jsonKey === undefined ? {} : { jsonKey }) },
});

test(`Associate only boolean properties and values inside JSON properties with consent purposes`, () => {
	const schema: ObjectType = {
		kind: 'object',
		properties: [
			property('boolean_consent', { kind: 'boolean' }),
			property('json_consents', { kind: 'json' }),
			property('text', { kind: 'string' }),
			property('nested', {
				kind: 'object',
				properties: [property('boolean_consent', { kind: 'boolean' })],
			}),
			property('consents', {
				kind: 'object',
				properties: [property('marketing', { kind: 'boolean' })],
			}),
		],
	};
	const purposes = [
		purpose('boolean', 'boolean_consent'),
		purpose('json-first', 'json_consents', 'first'),
		purpose('json-second', 'json_consents', 'a.b'),
		purpose('json-dotted', 'json_consents.first'),
		purpose('json-exact', 'json_consents'),
		purpose('nested', 'nested.boolean_consent'),
		purpose('text', 'text'),
		purpose('inside-boolean', 'boolean_consent.value'),
		purpose('key-on-boolean', 'boolean_consent', 'key'),
		purpose('key-on-object', 'consents', 'marketing'),
		purpose('missing', 'missing'),
		{ ...purpose('unknown', ''), eventConsentLocations: [{ purposeCode: 'marketing' }] },
	];

	const result = getConsentPurposesByPropertyPath(schema, purposes);

	expect(result.size).toBe(3);
	expect(result.get('boolean_consent')?.map((candidate) => candidate.id)).toEqual(['boolean']);
	expect(result.get('json_consents')?.map((candidate) => candidate.id)).toEqual(['json-first', 'json-second']);
	expect(result.get('nested.boolean_consent')?.map((candidate) => candidate.id)).toEqual(['nested']);
	expect(result.has('consents.marketing')).toBe(false);
	expect(result.has('text')).toBe(false);
	expect(result.has('missing')).toBe(false);
});

test(`Display profile locations without interpreting literal JSON keys`, () => {
	expect(formatProfileConsentLocation(null)).toBe('');
	for (const jsonKey of [undefined, '']) {
		expect(formatProfileConsentLocation({ property: 'preferences.newsletter', jsonKey })).toBe(
			'preferences.newsletter',
		);
	}
	for (const key of ['a.b', ' purpose code ', 'say "yes"', String.raw`\u0061`, 'a\n', '😀'.repeat(1024)]) {
		expect(formatProfileConsentLocation({ property: 'privacy.consents', jsonKey: key })).toBe(
			`privacy.consents[${JSON.stringify(key)}]`,
		);
	}
});

test(`Treat omitted and empty JSON keys as absent`, () => {
	const schema: ObjectType = {
		kind: 'object',
		properties: [property('accepted', { kind: 'boolean' }), property('consents', { kind: 'json' })],
	};
	const purposes = [
		purpose('omitted', 'accepted'),
		purpose('empty', 'accepted', ''),
		purpose('json-omitted', 'consents'),
		purpose('json-empty', 'consents', ''),
	];
	const result = getConsentPurposesByPropertyPath(schema, purposes);
	expect(result.size).toBe(1);
	expect(result.get('accepted')?.map(({ id }) => id)).toEqual(['omitted', 'empty']);
});

test(`Validate authored keys without trimming or treating dots as path separators`, () => {
	for (const key of ['marketing', 'a.b', '#CFK567', ' purpose code ', 'say "yes"', 'a\\b', '😀'.repeat(1024)]) {
		expect(() => validateConsentKey(key)).not.toThrow();
	}
	for (const key of [
		'',
		' ',
		'\u00a0',
		'a\t',
		'a\n',
		'a\u200b',
		'a\u2028',
		'a\u2029',
		'a\0',
		'\ud800',
		'😀'.repeat(1025),
	]) {
		expect(() => validateConsentKey(key)).toThrow();
	}
});
