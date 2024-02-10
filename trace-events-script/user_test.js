import { assert, assertEquals, assertNotEquals } from 'https://deno.land/std@0.212.0/assert/mod.ts';
import * as uuid from 'https://deno.land/std@0.212.0/uuid/v4.ts';
import Analytics from './analytics.js';

const DEBUG = false;

const writeKey = 'rq6JJg5ENWK28NHfxSwJZmzeIvDC8GQO';
const endpoint = 'https://example.com/api/v1/batch';

Deno.test('User', () => {
	localStorage.clear();
	globalThis.document = { visibilityState: 'visible' };

	const a = new Analytics(writeKey, endpoint);
	a.debug(DEBUG);

	assertEquals(a.user().id(), null);
	assert(uuid.validate(a.user().anonymousId()));
	assertEquals(a.user().traits(), {});

	assertEquals(a.user().id('8g1emx962iR'), '8g1emx962iR');
	assertEquals(a.user().id(), '8g1emx962iR');
	const anonymousId = a.getAnonymousId();
	assertEquals(a.user().id('e4X9L6mcA18'), 'e4X9L6mcA18');
	assertNotEquals(a.getAnonymousId(), anonymousId);
	assertEquals(a.user().id(null), null);
	assertEquals(a.user().id(), null);

	assertEquals(a.user().anonymousId('mC592p0Gn3z1Ld'), 'mC592p0Gn3z1Ld');
	assertEquals(a.user().anonymousId(), 'mC592p0Gn3z1Ld');

	assertEquals(a.user().anonymousId(null), null);
	assert(uuid.validate(a.user().anonymousId()));

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
		{
			set: {
				foo: () => {
				},
				boo: true,
			},
			expect: { boo: true },
		},
	];

	for (let i = 0; i < changes.length; i++) {
		const change = changes[i];
		assertEquals(a.user().traits(change.set), change.expect);
		assertEquals(a.user().traits(), change.expect);
	}
});
