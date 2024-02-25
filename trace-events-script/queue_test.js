import { assert, assertEquals } from 'https://deno.land/std@0.212.0/assert/mod.ts'
import { FakeTime } from 'https://deno.land/std@0.212.0/testing/time.ts'
import * as fake from './test_fake.js'
import { getTime } from './utils.js'
import Queue from './queue.js'

const DEBUG = false

Deno.test('Queue', () => {
	localStorage.clear()
	globalThis.document = {
		visibilityState: 'visible',
		addEventListener: addEventListener.bind(globalThis),
	}
	const time = new FakeTime()

	function assertRead(items, maxBytes, separatorBytes) {
		const gotItems = q.read(maxBytes, separatorBytes)
		assert(Array.isArray(gotItems))
		assertEquals(gotItems.length, items.length)
		for (let i = 0; i < items.length; i++) {
			assertEquals(gotItems[i], items[i])
		}
	}

	function assertEmpty() {
		assertEquals(q.age(), null)
		assertEquals(q.isEmpty(), true)
		assertEquals(q.length(), 0)
		const items = q.read(1000)
		assert(Array.isArray(items))
		assertEquals(items.length, 0)
	}

	const maxItemBytes = 20
	let q = new Queue(localStorage, 'queue', maxItemBytes, DEBUG)

	// The queue is empty.
	assertEmpty()

	// The append, read, and remove methods work correctly.
	assertEquals(q.append({ foo: true }), 12)
	time.tick(100)
	assertRead([`{"foo":true}`])
	time.tick(100)
	const age = getTime()
	assertEquals(q.append({ boo: '😁' }), 14)
	assertRead([`{"foo":true}`, `{"boo":"😁"}`])
	q.remove(1)
	time.tick(100)
	assertRead([`{"boo":"😁"}`])
	assertEquals(q.append({ a: { b: 23.4 } }), 16)
	assertEquals(q.append({ c: null }), 10)
	time.tick(100)
	assertRead([`{"boo":"😁"}`, `{"a":{"b":23.4}}`, `{"c":null}`])
	assertEquals(q.age(), age)
	assertEquals(q.isEmpty(), false)
	assertEquals(q.length(), 3)
	assertRead([], 13, 0)
	assertRead([`{"boo":"😁"}`], 14, 0)
	assertRead([`{"boo":"😁"}`], 14, 1)
	assertRead([`{"boo":"😁"}`], 15, 0)
	assertRead([`{"boo":"😁"}`], 29, 0)
	assertRead([`{"boo":"😁"}`, `{"a":{"b":23.4}}`], 30, 0)
	assertRead([`{"boo":"😁"}`], 30, 1)
	assertRead([`{"boo":"😁"}`, `{"a":{"b":23.4}}`], 31, 1)
	assertRead([`{"boo":"😁"}`], 31, 2)
	assertRead([`{"boo":"😁"}`, `{"a":{"b":23.4}}`], 32, 2)

	q.close()
	time.tick(100)

	// The queue has been make persistent.
	q = new Queue(localStorage, 'queue', maxItemBytes, DEBUG)
	assertRead([`{"boo":"😁"}`, `{"a":{"b":23.4}}`, `{"c":null}`])
	assertEquals(q.age(), age)
	assertEquals(q.isEmpty(), false)
	assertEquals(q.length(), 3)
	time.tick(100)

	// A too long item is not appended.
	assertEquals(q.append({ a_very_long_item_key: 'a very long item value' }), 49)
	assertEquals(q.age(), age)
	assertEquals(q.isEmpty(), false)
	assertEquals(q.length(), 3)

	// remove with a null or undefined argument removes all items.
	q.remove()
	assertRead([])
	assertEquals(q.append({ foo: [1, 3] }), 13)
	q.remove(null)
	assertRead([])

	q.close()
	time.tick(100)

	// A corrupted persisted queue does not break Queue.
	localStorage.setItem('queue', '....')
	q = new Queue(localStorage, 'queue', maxItemBytes, DEBUG)
	assertEmpty()
	q.close()
	localStorage.setItem('queue', '{}\n[}\n{}\n123\n123\n123\n2\n')
	q = new Queue(localStorage, 'queue', maxItemBytes, DEBUG)
	assertEmpty()
	q.close()

	// The queue also works without localStorage.
	q = new Queue(new fake.Storage(), 'queue', maxItemBytes, DEBUG)
	assertEmpty()
	time.tick(100)
	assertEquals(q.append({ foo: true }), 12)
	time.tick(100)
	assertRead([`{"foo":true}`])

	// An empty queue is correctly made persistent.
	q.remove()
	time.tick(100)
	q.close()
	q = new Queue(localStorage, 'queue', maxItemBytes, DEBUG)
	assertEmpty()
	q.close()

	// Two queues with two different keys are completely separated.
	localStorage.clear()
	let q1 = new Queue(localStorage, 'queue1', maxItemBytes, DEBUG)
	let q2 = new Queue(localStorage, 'queue2', maxItemBytes, DEBUG)
	q1.append({ q: 1 })
	time.tick(200)
	assertEquals(q2.length(), 0)
	q2.append({ q: 2 })
	time.tick(200)
	q1.close()
	q2.close()
	time.tick(200)
	q1 = new Queue(localStorage, 'queue1', maxItemBytes, DEBUG)
	time.tick(200)
	assertEquals(q1.read(), ['{"q":1}'])
	q2 = new Queue(localStorage, 'queue2', maxItemBytes, DEBUG)
	time.tick(200)
	assertEquals(q2.read(), ['{"q":2}'])
	q1.close()
	q2.close()
})
