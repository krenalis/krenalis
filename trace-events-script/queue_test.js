import { assert, assertEquals, assertThrows } from 'https://deno.land/std@0.218.0/assert/mod.ts'
import { FakeTime } from 'https://deno.land/std@0.218.0/testing/time.ts'
import * as fake from './test_fake.js'
import { getTime } from './utils.js'
import { ItemTooLargeError, Queue } from './queue.js'

const DEBUG = false

Deno.test('Queue', async (t) => {
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
		assertEquals(q.size(), 0)
		const items = q.read(1000)
		assert(Array.isArray(items))
		assertEquals(items.length, 0)
	}

	const maxItemBytes = 20
	let q = new Queue(localStorage, 'queue', maxItemBytes)
	q.debug(DEBUG)

	// The queue is empty.
	assertEmpty()

	// The append, read, and remove methods work correctly.
	q.append({ foo: true })
	assert(q.size() === 1)
	time.tick(100)
	assertRead([`{"foo":true}`])
	time.tick(100)
	const age = getTime()
	q.append({ boo: '😁' })
	assert(q.size() === 2)
	assertRead([`{"foo":true}`, `{"boo":"😁"}`])
	q.remove([`{"foo":true}`])
	time.tick(100)
	assert(q.size() === 1)
	assertRead([`{"boo":"😁"}`])
	q.append({ a: { b: 23.4 } })
	q.append({ c: null })
	time.tick(100)
	assertRead([`{"boo":"😁"}`, `{"a":{"b":23.4}}`, `{"c":null}`])
	assertEquals(q.age(), age)
	assertEquals(q.isEmpty(), false)
	assertEquals(q.size(), 3)
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
	localStorage.setItem('queue', `{"boo":"😁"}\n${age - 1000}\n14`)
	q.load('queue')
	q.remove([`{"boo":"😁"}`])
	assertEquals(q.age(), age)
	assertEquals(q.isEmpty(), false)
	assertEquals(q.size(), 3)

	q.close()
	time.tick(100)

	// The queue has been saved.
	q = new Queue(localStorage, 'queue', maxItemBytes)
	q.debug(DEBUG)
	q.load('queue')
	assertRead([`{"boo":"😁"}`, `{"a":{"b":23.4}}`, `{"c":null}`])
	assertEquals(q.age(), age)
	assertEquals(q.isEmpty(), false)
	assertEquals(q.size(), 3)
	time.tick(100)

	// A too long item is not appended.
	assertThrows(
		() => q.append({ a_very_long_item_key: 'a very long item value' }),
		ItemTooLargeError,
		'The item is too large',
	)
	assertEquals(q.age(), age)
	assertEquals(q.isEmpty(), false)
	assertEquals(q.size(), 3)

	// Clear the queue.
	q.clear()
	assertRead([])

	q.close()
	time.tick(100)

	// Load an empty queue into an empty queue.
	q = new Queue(localStorage, 'queue', maxItemBytes)
	q.debug(DEBUG)
	localStorage.setItem('queue', '')
	q.load()
	assertEquals(q.size(), 0)

	// Load a non-empty queue.
	localStorage.setItem('queue', '{"i":251}\n{"i":609}\n{"i":731}\n251 609 731\n9 9 9')
	q.load('queue')
	assertEquals(q.size(), 3)
	assertEquals(q.read(), ['{"i":251}', '{"i":609}', '{"i":731}'])

	// Load an empty queue in a non-empty queue.
	localStorage.setItem('queue', '')
	q.load()
	assertEquals(q.size(), 3)
	assertEquals(q.read(), ['{"i":251}', '{"i":609}', '{"i":731}'])

	// Load non-empty queues into a non-empty queue.
	localStorage.setItem('queue', '{"i":144}\n{"i":609}\n{"i":671}\n{"i":1072}\n144 609 671 1072\n9 9 9 10')
	q.load('queue')
	assertEquals(q.size(), 7)
	assertEquals(q.read(), ['{"i":144}', '{"i":251}', '{"i":609}', '{"i":609}', '{"i":671}', '{"i":731}', '{"i":1072}'])
	localStorage.setItem('queue', '{"i":855}\n{"i":982}\n855 982\n9 9')
	q.load('queue')
	assertEquals(q.size(), 9)
	assertEquals(q.read(), [
		'{"i":144}',
		'{"i":251}',
		'{"i":609}',
		'{"i":609}',
		'{"i":671}',
		'{"i":731}',
		'{"i":855}',
		'{"i":982}',
		'{"i":1072}',
	])

	q.close()
	time.tick(100)

	// A corrupted saved queue does not break Queue.
	localStorage.setItem('queue', '....')
	q = new Queue(localStorage, 'queue', maxItemBytes)
	q.debug(DEBUG)
	assertEmpty()
	q.close()
	localStorage.setItem('queue', '{}\n[}\n{}\n123\n123\n123\n2\n')
	q = new Queue(localStorage, 'queue', maxItemBytes)
	q.debug(DEBUG)
	assertEmpty()
	q.close()

	// The queue also works without localStorage.
	q = new Queue(new fake.Storage(), 'queue', maxItemBytes)
	q.debug(DEBUG)
	assertEmpty()
	time.tick(100)
	q.append({ foo: true })
	time.tick(100)
	assertRead([`{"foo":true}`])

	// An empty queue is correctly saved.
	q.clear()
	time.tick(100)
	q.close()
	q = new Queue(localStorage, 'queue', maxItemBytes)
	q.debug(DEBUG)
	assertEmpty()
	q.close()

	// Two queues with two different keys are completely separated.
	localStorage.clear()
	let q1 = new Queue(localStorage, 'queue1', maxItemBytes)
	q1.debug(DEBUG)
	let q2 = new Queue(localStorage, 'queue2', maxItemBytes)
	q2.debug(DEBUG)
	q1.append({ q: 1 })
	time.tick(200)
	assertEquals(q2.size(), 0)
	q2.append({ q: 2 })
	time.tick(200)
	q1.close()
	q2.close()
	time.tick(200)
	q1 = new Queue(localStorage, 'queue1', maxItemBytes)
	q1.debug(DEBUG)
	q1.load('queue1')
	time.tick(200)
	assertEquals(q1.read(), ['{"q":1}'])
	q2 = new Queue(localStorage, 'queue2', maxItemBytes)
	q2.debug(DEBUG)
	q2.load('queue2')
	time.tick(200)
	assertEquals(q2.read(), ['{"q":2}'])
	q1.close()
	q2.close()

	time.restore()

	await t.step('Save the queue', () => {
		const time = new FakeTime(1709192826078)
		// Named queue.
		localStorage.clear()
		let q = new Queue(localStorage, 'queue', maxItemBytes)
		q.append({ foo: 'boo' })
		q.save()
		assertEquals(localStorage.getItem('queue'), '{"foo":"boo"}\n1709192826078\n13')
		assert(!q.isEmpty())
		q.close()

		// Unnamed queue.
		localStorage.clear()
		q = new Queue(localStorage, '*.queue', maxItemBytes)
		q.append({ foo: 'boo' })
		q.save()
		assertEquals(localStorage.length, 1)
		assertEquals(localStorage.getItem(localStorage.key(0)), '{"foo":"boo"}\n1709192826078\n13')
		assert(q.isEmpty())
		q.close()
		time.restore()
	})
})
