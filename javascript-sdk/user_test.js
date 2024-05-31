import { assert, assertEquals } from '@std/assert'
import * as uuid from '@std/uuid/v4'
import Storage from './storage.js'
import User from './user.js'
import Options from './options.js'

const writeKey = 'rq6JJg5ENWK28NHfxSwJZmzeIvDC8GQO'

Deno.test('User', () => {
	localStorage.clear()
	globalThis.document = {
		visibilityState: 'visible',
		addEventListener: addEventListener.bind(globalThis),
	}

	const user = new User(new Storage(writeKey, new Options()))

	assertEquals(user.id(), null)
	assert(uuid.validate(user.anonymousId()))
	assertEquals(user.traits(), {})

	assertEquals(user.id('8g1emx962iR'), '8g1emx962iR')
	assertEquals(user.id(), '8g1emx962iR')
	assertEquals(user.id('e4X9L6mcA18'), 'e4X9L6mcA18')
	assertEquals(user.id(null), null)
	assertEquals(user.id(), null)

	assertEquals(user.anonymousId('mC592p0Gn3z1Ld'), 'mC592p0Gn3z1Ld')
	assertEquals(user.anonymousId(), 'mC592p0Gn3z1Ld')

	assertEquals(user.anonymousId(null), null)
	assert(uuid.validate(user.anonymousId()))

	const rec = {}
	rec.boo = rec

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
	]

	for (let i = 0; i < changes.length; i++) {
		const change = changes[i]
		assertEquals(user.traits(change.set), change.expect)
		assertEquals(user.traits(), change.expect)
	}
})
