import { test, expect } from '@playwright/test';
import { EditableSchema, getPropertyInsertionAnchor } from '../src/components/routes/SchemaEdit/SchemaEdit.helpers';
import { getReorderedPropertyKeys } from '../src/components/routes/SchemaEdit/usePropertyReordering';

const createMoveHistory = (...keys: string[]): ReadonlyMap<string, number> => {
	return new Map(keys.map((key, index) => [key, index + 1]));
};

const createSchema = (keys: string[], addedKeys: string[] = []): EditableSchema => {
	const addedKeySet = new Set(addedKeys);
	const objectKeys = new Set(keys.filter((key) => keys.some((candidate) => candidate.startsWith(`${key}.`))));
	const schema: EditableSchema = {};
	for (const key of keys) {
		const fragments = key.split('.');
		schema[key] = {
			indentation: fragments.length - 1,
			root: fragments[0],
			name: fragments[fragments.length - 1],
			prefilled: '',
			role: 'Both',
			type: objectKeys.has(key) ? { kind: 'object', properties: [] } : { kind: 'string' },
			readOptional: true,
			createRequired: false,
			updateRequired: false,
			nullable: false,
			description: '',
			isEditable: addedKeySet.has(key),
		};
	}
	return schema;
};

const getReorderedKeys = (initial: string[], current: string[], movedKeys: string[], addedKeys: string[] = []) => {
	return Array.from(
		getReorderedPropertyKeys(
			createSchema(current, addedKeys),
			createSchema(initial),
			createMoveHistory(...movedKeys),
		),
	).sort();
};

test(`Finds an insertion anchor within the selected branch`, () => {
	const schema = createSchema([
		'first_name',
		'address',
		'address.street',
		'address.billing',
		'address.billing.city',
		'address.country',
		'preferences',
		'preferences.language',
	]);

	expect(getPropertyInsertionAnchor(schema, '', 'first_name')).toBe('first_name');
	expect(getPropertyInsertionAnchor(schema, '', 'address')).toBe('address.country');
	expect(getPropertyInsertionAnchor(schema, '', 'address.billing.city')).toBe('address.country');
	expect(getPropertyInsertionAnchor(schema, 'address', 'address.billing.city')).toBe('address.billing.city');
	expect(getPropertyInsertionAnchor(schema, 'address', 'address')).toBeNull();
	expect(getPropertyInsertionAnchor(schema, 'address', 'preferences.language')).toBeNull();
});

test(`Attributes a move to the row that was moved instead of the rows it crossed`, () => {
	expect(getReorderedKeys(['A', 'B', 'C', 'D'], ['B', 'C', 'D', 'A'], ['A'])).toEqual(['A']);
	expect(getReorderedKeys(['A', 'B', 'C'], ['C', 'A', 'B'], ['C'])).toEqual(['C']);
	expect(getReorderedKeys(['A', 'B', 'C'], ['B', 'A', 'C'], ['A'])).toEqual(['A']);
	expect(getReorderedKeys(['A', 'B', 'C'], ['B', 'A', 'C'], ['B'])).toEqual(['B']);
	expect(getReorderedKeys(['A', 'B', 'C'], ['B', 'C', 'A'], ['B', 'C'])).toEqual(['B', 'C']);
});

test(`Uses the most recent move to resolve an ambiguous swap`, () => {
	expect(getReorderedKeys(['A', 'B', 'C'], ['B', 'A', 'C'], ['B', 'A'])).toEqual(['A']);
	expect(getReorderedKeys(['A', 'B', 'C'], ['B', 'A', 'C'], ['A', 'B'])).toEqual(['B']);
});

test(`Clears reorder-only changes when the original order is restored`, () => {
	expect(getReorderedKeys(['A', 'B', 'C'], ['A', 'B', 'C'], ['A', 'B'])).toEqual([]);
});

test(`Treats object subtrees and child groups independently`, () => {
	expect(
		getReorderedKeys(
			['address', 'address.street', 'address.city', 'preferences', 'preferences.language'],
			['preferences', 'preferences.language', 'address', 'address.street', 'address.city'],
			['preferences'],
		),
	).toEqual(['preferences']);
	expect(
		getReorderedKeys(
			['address', 'address.street', 'address.city', 'preferences'],
			['address', 'address.city', 'address.street', 'preferences'],
			['address.city'],
		),
	).toEqual(['address.city']);
	expect(
		getReorderedKeys(
			['address', 'address.billing', 'address.billing.code', 'address.other'],
			['address', 'address.other', 'address.billing', 'address.billing.code'],
			['address.billing'],
		),
	).toEqual(['address.billing']);
	expect(
		getReorderedKeys(
			[
				'address',
				'address.street',
				'address.city',
				'preferences',
				'preferences.language',
				'preferences.newsletter',
			],
			[
				'address',
				'address.city',
				'address.street',
				'preferences',
				'preferences.newsletter',
				'preferences.language',
			],
			['address.city', 'preferences.newsletter'],
		),
	).toEqual(['address.city', 'preferences.newsletter']);
});

test(`Ignores additions, removals, and stale history when comparing order`, () => {
	expect(getReorderedKeys(['A', 'B'], ['A', 'X', 'B'], ['X'], ['X'])).toEqual([]);
	expect(getReorderedKeys(['A', 'B'], ['X', 'A', 'B'], ['A'], ['X'])).toEqual([]);
	expect(getReorderedKeys(['A', 'B', 'C'], ['A', 'C'], ['B'])).toEqual([]);
	expect(getReorderedKeys(['A', 'B', 'C'], ['C', 'A'], ['C'])).toEqual(['C']);
});

test(`Ignores a newly added property that reuses the key of a removed property`, () => {
	expect(getReorderedKeys(['A', 'B', 'C'], ['B', 'A', 'C'], [], ['B'])).toEqual([]);
});

test(`Uses a deterministic fallback when move history is unavailable`, () => {
	expect(getReorderedKeys(['A', 'B', 'C'], ['B', 'A', 'C'], [])).toEqual(['B']);
});

test(`Treats property keys as data when reading move history`, () => {
	expect(getReorderedKeys(['toString', 'A'], ['A', 'toString'], [])).toEqual(['A']);
});

test(`Marks a moved row when it crosses an unmoved anchor`, () => {
	expect(getReorderedKeys(['A', 'B', 'C', 'D', 'E'], ['A', 'D', 'B', 'C', 'E'], ['D'])).toEqual(['D']);
});
