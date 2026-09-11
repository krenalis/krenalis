import { expect, test } from '@playwright/test';
import Type, { ObjectType, Property } from '../src/lib/api/types/types';
import { ConsentPurpose } from '../src/lib/api/types/workspace';
import {
	getConsentPurposeEventPath,
	getConsentPurposeProfilePath,
	getConsentPurposesByPropertyPath,
} from '../src/components/routes/Schema/SchemaPropertyConsent.helpers';

const property = (name: string, type: Type): Property => ({
	name,
	prefilled: '',
	role: 'Both',
	type,
	createRequired: false,
	updateRequired: false,
	readOptional: true,
	nullable: false,
	description: '',
});

const purpose = (id: string, code: string, profilePath: string): ConsentPurpose => ({
	id,
	code,
	name: `Purpose ${id}`,
	aliases: [],
	eventPath: '',
	profilePath,
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
		purpose('boolean', 'boolean', 'boolean_consent'),
		purpose('json-first', 'json_first', 'json_consents.first'),
		purpose('json-second', 'json_second', 'json_consents.second'),
		purpose('json-exact', 'json_exact', 'json_consents'),
		purpose('nested', 'nested', 'nested.boolean_consent'),
		purpose('text', 'text', 'text'),
		purpose('inside-boolean', 'inside_boolean', 'boolean_consent.value'),
		purpose('missing', 'missing', 'missing'),
		purpose('default', 'marketing', ''),
	];

	const result = getConsentPurposesByPropertyPath(schema, purposes);

	expect(result.get('boolean_consent')?.map((candidate) => candidate.id)).toEqual(['boolean']);
	expect(result.get('json_consents')?.map((candidate) => candidate.id)).toEqual(['json-first', 'json-second']);
	expect(result.get('nested.boolean_consent')?.map((candidate) => candidate.id)).toEqual(['nested']);
	expect(result.get('consents.marketing')?.map((candidate) => candidate.id)).toEqual(['default']);
	expect(result.has('text')).toBe(false);
	expect(result.has('missing')).toBe(false);
});

test(`Resolve default consent purpose paths`, () => {
	const defaultPurpose = purpose('default', 'marketing', '');
	const customPurpose = {
		...purpose('custom', 'custom', 'privacy.marketing'),
		eventPath: 'privacy.events.marketing',
	};

	expect(getConsentPurposeEventPath(defaultPurpose)).toBe('context.consents.marketing');
	expect(getConsentPurposeProfilePath(defaultPurpose)).toBe('consents.marketing');
	expect(getConsentPurposeEventPath(customPurpose)).toBe('privacy.events.marketing');
	expect(getConsentPurposeProfilePath(customPurpose)).toBe('privacy.marketing');
});
