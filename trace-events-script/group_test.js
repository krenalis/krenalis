import { assertEquals } from 'https://deno.land/std@0.212.0/assert/mod.ts';
import Analytics from './analytics.js';

const DEBUG = false;

const writeKey = 'rq6JJg5ENWK28NHfxSwJZmzeIvDC8GQO';
const endpoint = 'https://example.com/api/v1/batch';

Deno.test('Group', () => {
	localStorage.clear();
	globalThis.document = { visibilityState: 'visible' };

	const a = new Analytics(writeKey, endpoint);
	a.debug(DEBUG);

	assertEquals(a.group().id(), null);
	assertEquals(a.group().traits(), {});

	assertEquals(a.group().id('acme'), 'acme');
	assertEquals(a.group().id(), 'acme');
	assertEquals(a.group().id(null), null);
	assertEquals(a.group().id(), null);

	const rec = {};
	rec.boo = rec;

	// Apply the following changes to traits consecutively and test the results of each step.
	const changes = [
		{ set: { foo: true }, expect: { foo: true } },
		{ set: undefined, expect: { foo: true } },
		{ set: null, expect: {} },
		{ set: { foo: false }, expect: { foo: false } },
		{ set: 'foo', expect: { foo: false } },
		{ set: { foo: {} }, expect: { foo: {} } },
		{ set: { foo: 5n, boo: true }, expect: { foo: {} } },
		{ set: { foo: undefined, boo: 5 }, expect: { boo: 5 } },
		{ set: { foo: rec }, expect: { boo: 5 } },
		{ set: { foo: () => {}, boo: true }, expect: { boo: true } },
	];

	for (let i = 0; i < changes.length; i++) {
		const change = changes[i];
		assertEquals(a.group().traits(change.set), change.expect);
		assertEquals(a.group().traits(), change.expect);
	}
});
