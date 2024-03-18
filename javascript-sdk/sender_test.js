import { assert, assertEquals } from 'std/assert/mod.ts'
import { FakeTime } from 'std/testing/time.ts'
import * as fake from './test_fake.js'
import Sender from './sender.js'
import { Queue } from './queue.js'

const DEBUG = false

const writeKey = 'rq6JJg5ENWK28NHfxSwJZmzeIvDC8GQO'
const endpoint = 'https://example.com/api/v1/'

Deno.test('Sender send', async (t) => {
	// Prepare the execution environment.
	{
		localStorage.clear()
		navigator.onLine = true
		assert(navigator.onLine)
		globalThis.document = {
			visibilityState: 'visible',
			addEventListener: addEventListener.bind(globalThis),
		}
	}

	const events = [
		{ messageId: '53f6c7da-cf9c-4e8d-85e3-fa45a45b9221' },
		{ messageId: '53f6c7da-cf9c-4e8d-85e3-fa45a45b9221' },
		{ messageId: '2f825fe5-b492-4ddf-a58e-7c5567366870' },
		{ messageId: 'ba30a14a-3d9e-4985-a254-e6517c4a237c' },
	]

	await t.step('fetch', async () => {
		let time
		let fetch
		let sender
		let queue

		try {
			time = new FakeTime()
			fetch = new fake.Fetch(writeKey, endpoint + 'b', false, DEBUG)
			fetch.install()
			queue = new Queue(localStorage, 'queue', 32 * 1024)
			queue.debug(DEBUG)
			sender = new Sender(writeKey, endpoint, queue)
			sender.debug(DEBUG)
			for (let i = 0; i < events.length; i++) {
				queue.append(events[i])
			}
			time.tick(sender.timeout)
			const sentEvents = await fetch.events(events.length)
			assertEquals(sentEvents.length, events.length)
			for (let i = 0; i < events.length; i++) {
				assertEquals(sentEvents[i], events[i])
			}
		} finally {
			if (sender != null) {
				sender.close()
				queue.close()
			}
			fetch.restore()
			time.restore()
		}

		localStorage.clear()

		try {
			time = new FakeTime()
			fetch = new fake.Fetch(writeKey, endpoint + 'b', false, DEBUG)
			fetch.install()
			queue = new Queue(localStorage, 'queue', 32 * 1024)
			queue.debug(DEBUG)
			sender = new Sender(writeKey, endpoint, queue)
			sender.debug(DEBUG)
			const maxPerBatch = 9658 // This value can change if the sender's implementation change.
			// Send maxPerBatch events.
			for (let i = 0; i < maxPerBatch; i++) {
				queue.append({ messageId: crypto.randomUUID() })
			}
			queue.append({ messageId: crypto.randomUUID() })
			queue.append({ messageId: crypto.randomUUID() })
			time.tick(sender.timeout)
			let events = await fetch.events(maxPerBatch)
			assertEquals(events.length, maxPerBatch)
			await time.nextAsync()
			events = await fetch.events(2)
			assertEquals(events.length, 2)
		} finally {
			if (sender != null) {
				sender.close()
			}
			if (queue != null) {
				queue.close()
			}
			fetch.restore()
			time.restore()
		}
	})

	localStorage.clear()

	await t.step('sendBeacon', async () => {
		const time = new FakeTime()
		const sendBeacon = new fake.SendBeacon(writeKey, endpoint + 'b', DEBUG)
		sendBeacon.install()
		const fetch = new fake.Fetch(writeKey, endpoint + 'b', false, DEBUG)
		fetch.install()
		let queue
		let sender
		try {
			queue = new Queue(localStorage, 'queue', 32 * 1024)
			queue.debug(DEBUG)
			sender = new Sender(writeKey, endpoint, queue)
			sender.debug(DEBUG)
			for (let i = 0; i < events.length; i++) {
				queue.append(events[i])
			}
			// First send, with sendBeacon.
			sender.flush()
			let sentEvents = await sendBeacon.events(events.length)
			assertEquals(sentEvents.length, events.length)
			for (let i = 0; i < events.length; i++) {
				assertEquals(sentEvents[i], events[i])
			}
			assert(!queue.isEmpty())
			// Second send, with flush.
			time.tick(sender.timeout)
			await time.nextAsync()
			sentEvents = await fetch.events(events.length)
			assertEquals(sentEvents.length, events.length)
			for (let i = 0; i < events.length; i++) {
				assertEquals(sentEvents[i], events[i])
			}
			assert(queue.isEmpty())
		} finally {
			if (sender != null) {
				sender.close()
			}
			if (queue != null) {
				queue.close()
			}
			fetch.restore()
			sendBeacon.restore()
			time.restore()
		}
	})

	localStorage.clear()

	await t.step('XMLHttpRequest', async () => {
		const time = new FakeTime()
		fake.XMLHttpRequest.install(writeKey, endpoint + 'b', DEBUG)
		assertEquals(globalThis.XMLHttpRequest, XMLHttpRequest)
		const fetch = globalThis.fetch
		globalThis.fetch = undefined
		assertEquals(globalThis.fetch, undefined)
		let queue
		let sender
		try {
			queue = new Queue(localStorage, 'queue', 32 * 1024)
			queue.debug(DEBUG)
			sender = new Sender(writeKey, endpoint, queue)
			sender.debug(DEBUG)
			for (let i = 0; i < events.length; i++) {
				queue.append(events[i])
			}
			time.tick(sender.timeout)
			const sentEvents = await fake.XMLHttpRequest.events(events.length)
			assertEquals(sentEvents.length, events.length)
			for (let i = 0; i < events.length; i++) {
				assertEquals(sentEvents[i], events[i])
			}
		} finally {
			if (sender != null) {
				sender.close()
			}
			if (queue != null) {
				queue.close()
			}
			globalThis.fetch = fetch
			fake.XMLHttpRequest.restore()
			time.restore()
		}
	})
})

Deno.test('Sender postRetry', async () => {
	const exception = new Error('send exception')
	const error = new Error('send error')
	const offline = new Error('browser is offline')

	function installFetch(responses) {
		let globalResolve
		let globalReject
		let i = 0
		globalThis.fetch = () => {
			if (i === responses.length) {
				globalReject('got a request when no more requests were expected')
				return
			}
			const r = responses[i]
			i++
			// Test an exception.
			if (r === exception) {
				throw exception
			}
			// Test browser offline.
			if (r === offline) {
				navigator.onLine = false
				const addEventListener = globalThis.addEventListener
				globalThis.addEventListener = (...theArgs) => {
					const type = theArgs[0]
					if (type === 'online') {
						globalThis.addEventListener = addEventListener
						const listener = theArgs[1]
						setTimeout(() => {
							navigator.onLine = true
							listener()
						}, 10)
						return
					}
					addEventListener.apply(globalThis, theArgs)
				}
				throw offline
			}
			return new Promise(function (resolve, reject) {
				// Test a network error.
				if (r === error) {
					setTimeout(reject, 0, error)
					return
				}
				// Test an HTTP response.
				const response = new Response(null, {
					status: typeof r === 'number' ? r : r.status,
					statusText: 'Status Message',
					headers: typeof r === 'object' && 'retryAfter' in r
						? new Headers({ 'Retry-After': r.retryAfter })
						: new Headers(),
				})
				setTimeout(() => {
					resolve(response)
					if (i === responses.length) {
						setTimeout(globalResolve)
					}
				})
			})
		}
		return new Promise((resolve, reject) => {
			globalResolve = resolve
			globalReject = reject
		})
	}

	const fetch = globalThis.fetch

	const tests = [
		[exception, 200],
		[error, 200],
		[200],
		[201],
		[404],
		[500, 200],
		[{ status: 429, retryAfter: '1' }, 200],
		[{ status: 429, retryAfter: '' }, 200],
		[{ status: 429, retryAfter: null }, 200],
		[{ status: 501, retryAfter: '1' }, 200],
		[{ status: 503, retryAfter: new Date(Date.now() + 1000).toUTCString() }, 200],
		[{ status: 503, retryAfter: null }, 200],
		[{ status: 503, retryAfter: 'foo' }, 200],
		[504, 200],
		[500, 500, 500, 500, 200],
		[{ status: 503, retryAfter: new Date(Date.now() + 1000).toUTCString() }, 500, 200],
		[500, 500, 404],
		[offline, 200],
		[offline, 500, 200],
		[500, offline, 500, offline, offline, 200],
		[301],
		[400],
		[401],
	]

	navigator.onLine = true

	try {
		for (let i = 0; i < tests.length; i++) {
			const responses = tests[i]
			if (DEBUG) {
				console.debug('> expected fetch responses:', responses)
			}
			const queue = new Queue(localStorage, 'queue', 32 * 1024)
			queue.debug(DEBUG)
			const sender = new Sender(writeKey, endpoint, queue)
			sender.debug(DEBUG)
			queue.append({})
			try {
				setTimeout(sender.flush.bind(sender))
				await installFetch(responses)
				assert(queue.isEmpty())
			} finally {
				sender.close()
				queue.close()
			}
		}
	} finally {
		globalThis.fetch = fetch
		delete navigator.onLine
	}
})
