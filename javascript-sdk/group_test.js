import { assertEquals } from 'https://deno.land/std@0.218.0/assert/mod.ts'
import Options from './options.js'
import Storage from './storage.js'
import Group from './group.js'

const writeKey = 'rq6JJg5ENWK28NHfxSwJZmzeIvDC8GQO'

Deno.test('Group', () => {
	localStorage.clear()
	globalThis.document = {
		visibilityState: 'visible',
		addEventListener: addEventListener.bind(globalThis),
	}

	const group = new Group(new Storage(writeKey, new Options().storage))

	assertEquals(group.id(), null)
	assertEquals(group.traits(), {})

	assertEquals(group.id('acme'), 'acme')
	assertEquals(group.id(), 'acme')
	assertEquals(group.id(null), null)
	assertEquals(group.id(), null)

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
		{ set: { foo: () => {}, boo: true }, expect: { boo: true } },
	]

	for (let i = 0; i < changes.length; i++) {
		const change = changes[i]
		assertEquals(group.traits(change.set), change.expect)
		assertEquals(group.traits(), change.expect)
	}
})
