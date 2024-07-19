import { assert, assertEquals, AssertionError, assertNotEquals, assertThrows } from '@std/assert'
import { FakeTime } from '@std/testing/time'
import * as uuid from '@std/uuid/v4'
import * as fake from './test_fake.js'
import { steps } from './analytics_test_steps.js'
import { getTime } from './utils.js'
import Analytics from './analytics.js'

const DEBUG = false

const writeKey = 'rq6JJg5ENWK28NHfxSwJZmzeIvDC8GQO'
const endpoint = 'https://example.com/api/v1/'

Deno.test('Analytics', async (t) => {
	// Prepare the execution environment.
	{
		globalThis.navigator.onLine = true
		assert(globalThis.navigator.onLine)

		// Mock document.
		globalThis.document = new fake.HTMLDocument()

		// Mock location.
		const url = new URL('https://example.com:8080/path?query=123#fragment')
		globalThis.location = {
			href: url.toString(),
			protocol: url.protocol,
			host: url.host,
			hostname: url.hostname,
			port: url.port,
			pathname: url.pathname,
			search: url.search,
			hash: url.hash,
			origin: url.origin,
		}

		// Mock screen.
		globalThis.screen = { width: 2560, height: 1440 }

		// Mock devicePixelRatio.
		globalThis.devicePixelRatio = 1.25
	}

	const _Promise = globalThis.Promise

	const minute = 60 * 1000
	const thirtyMinutes = 30 * minute
	const fiveMinutes = 5 * minute

	// newAnalytics returns a new instance of Analytics. However, the returned
	// instance is not immediately ready; it becomes ready after a delay of
	// 'latency' milliseconds.
	//
	// To wait until it's ready, call the 'ready' method and await the returned
	// promise. If using fake time, you can advance the time by calling
	// time.tick(latency), or alternatively, call time.next() to avoid advancing
	// the fake time.
	function newAnalytics(options, strategy, latency) {
		const fetch = globalThis.fetch
		globalThis.fetch = function () {
			return new _Promise(function (resolve) {
				const body = JSON.stringify({ strategy: strategy || 'AB-C' })
				const response = new Response(body, {
					status: 200,
					statusText: 'OK',
					headers: new Headers({ 'content-type': 'text/javascript' }),
				})
				setTimeout(resolve, latency || 0, response)
			})
		}
		localStorage.clear()
		if (DEBUG) {
			if (options == null) {
				options = {}
			}
			options.debug = true
		}
		try {
			return new Analytics(writeKey, endpoint, options)
		} finally {
			globalThis.fetch = fetch
		}
	}

	await t.step('ready, when Promise is not supported', async () => {
		globalThis.Promise = null
		const a = newAnalytics()
		try {
			// Before Analytics is ready.
			let cb
			let promise = new _Promise((resolve) => {
				cb = resolve
			})
			void a.ready(cb)
			let cb2
			const promise2 = new _Promise((resolve) => {
				cb2 = resolve
			})
			void a.ready(cb2)
			await promise
			await promise2
			// After Analytics is ready.
			promise = new _Promise((resolve) => {
				cb = resolve
			})
			void a.ready(cb)
			await promise
		} finally {
			globalThis.Promise = _Promise
		}
		a.close()
	})

	await t.step('ready, when Promise is supported', async () => {
		const a = newAnalytics()
		// Before Analytics is ready.
		await a.ready()
		// After Analytics is ready.
		await a.ready()
		// With a callback.
		let callback
		const promise = new Promise((resolve) => {
			callback = resolve
		})
		void a.ready(callback)
		await promise
		a.close()
	})

	await t.step('no key is created in the localStorage', () => {
		const a = newAnalytics({ sessions: { autoTrack: false } })
		assertEquals(localStorage.length, 0)
		a.close()
	})

	await t.step('reset function', async () => {
		const fetch = new fake.Fetch(writeKey, endpoint + 'b', false, DEBUG)
		const a = newAnalytics({ sessions: { autoTrack: false } }, 'AC-B')
		await a.ready()
		a.startSession(137206)
		a.setAnonymousId('53c5986a-7fa4-493c-9a61-75c483aaf3d7')
		const time = new FakeTime()
		fetch.install()
		try {
			void a.identify('17258645', { name: 'John' })
			void a.group('2649247', { name: 'Acme' })
			time.tick(1000)
			await fetch.events(2)
		} finally {
			fetch.restore()
			time.restore()
		}
		a.reset()
		// localStorage only contains the keys corresponding to the election and the queue items.
		let hasQueueKey = false
		for (let i = 0; i < localStorage.length; i++) {
			if (localStorage.key(i).endsWith('.queue')) {
				hasQueueKey = true
				break
			}
		}
		assert(hasQueueKey)
		assert(localStorage.getItem(`meergo.rq6JJg5.leader.beat`) != null)
		assert(localStorage.getItem(`meergo.rq6JJg5.leader.election`) != null)
		a.close()
	})

	await t.step('startSession argument validation', async () => {
		const a = newAnalytics({ sessions: { autoTrack: false } }, 'AC-B')
		await a.ready()
		// Check valid startSession arguments.
		let ids = [null, undefined, 1, 300, Number.MAX_SAFE_INTEGER]
		for (let i = 0; i < ids.length; i++) {
			a.startSession(ids[i])
		}
		// Check invalid startSession arguments.
		ids = ['a', {}, -100, -10.56, -1, 0, -0, 0.1, 23.904, Number.MAX_SAFE_INTEGER + 1]
		for (let i = 0; i < ids.length; i++) {
			assertThrows(
				() => {
					a.startSession(ids[i])
				},
				Error,
				'sessionId must be a positive integer',
			)
		}
		a.close()
	})

	await t.step('getAnonymousId function', () => {
		const a = newAnalytics()
		assert(uuid.validate(a.getAnonymousId()))
		a.setAnonymousId('f5d354ed')
		assertEquals(a.getAnonymousId(), 'f5d354ed')
		a.setAnonymousId(903726473)
		assertEquals(a.getAnonymousId(), '903726473')
		a.setAnonymousId('')
		assert(uuid.validate(a.getAnonymousId()))
		a.setAnonymousId({})
		assert(uuid.validate(a.getAnonymousId()))
		a.close()
	})

	await t.step('setAnonymousId function', () => {
		const a = newAnalytics()
		assert(uuid.validate(a.setAnonymousId()))
		const anonymousId = 'f5d354ed'
		assertEquals(a.setAnonymousId(anonymousId), anonymousId)
		assertEquals(a.setAnonymousId(), anonymousId)
		assertEquals(a.setAnonymousId(903726473), '903726473')
		assertEquals(a.setAnonymousId(), '903726473')
		assertEquals(a.setAnonymousId(''), '')
		assert(uuid.validate(a.setAnonymousId()))
		assertEquals(a.setAnonymousId({}), {})
		assert(uuid.validate(a.setAnonymousId()))
		a.close()
	})

	await t.step('sessions with auto tracking', () => {
		const time = new FakeTime()
		const fetch = new fake.Fetch(writeKey, endpoint + 'b', false, DEBUG)
		fetch.install()
		let a
		try {
			a = newAnalytics()
			let sessionId = getTime()
			assertEquals(a.getSessionId(), sessionId)
			time.tick(fiveMinutes)
			assertEquals(a.getSessionId(), sessionId)
			time.tick(thirtyMinutes)
			assertEquals(a.getSessionId(), null)
			void a.track('click')
			sessionId = getTime()
			assertEquals(a.getSessionId(), sessionId)
			time.tick(100)
			a.reset()
			assertNotEquals(a.getSessionId(), sessionId)
		} finally {
			if (a != null) {
				a.close()
			}
			fetch.restore()
			time.restore()
		}
	})

	await t.step('Querystring API', async () => {
		const location = globalThis.location
		const query =
			'?ajs_uid=u179362&ajs_event=click&ajs_aid=90261537&ajs_prop_a=foo&ajs_prop_b=&ajs_prop_c=boo&ajs_trait_name=John&ajs_trait_age=32'
		const url = new URL(`https://example.com/${query}`)
		globalThis.location = {
			href: url.toString(),
			protocol: url.protocol,
			host: url.host,
			hostname: url.hostname,
			port: url.port,
			pathname: url.pathname,
			search: url.search,
			hash: url.hash,
			origin: url.origin,
		}
		const fetch = new fake.Fetch(writeKey, endpoint + 'b', false, DEBUG)
		fetch.install()
		let a
		try {
			a = newAnalytics({ useQueryString: { aid: /^\d+$/, uid: /^u\d+$/ } })
			await a.ready()
			assertEquals(a.user().anonymousId(), '90261537')
			let events = await fetch.events(2)
			assertEquals(events.length, 2)
			assertEquals(events[0].type, 'identify')
			assertEquals(events[0].anonymousId, '90261537')
			assertEquals(events[0].userId, 'u179362')
			assertEquals(events[0].traits, { name: 'John', age: '32' })
			assertEquals(events[1].type, 'track')
			assertEquals(events[1].event, 'click')
			assertEquals(events[1].anonymousId, '90261537')
			assertEquals(events[1].userId, 'u179362')
			assertEquals(events[1].properties, { a: 'foo', b: '', c: 'boo' })
			a.close()

			a = newAnalytics({ useQueryString: { aid: /^[a-z]+$/, uid: /^[A-Za-z\-]+$/ } })
			await a.ready()
			assertNotEquals(a.user().anonymousId(), '90261537')
			events = await fetch.events(1)
			assertEquals(events.length, 1)
			assertEquals(events[0].type, 'track')
			assertEquals(events[0].event, 'click')
			assertNotEquals(events[0].anonymousId, '90261537')
			assertEquals(events[0].userId, null)
			assertEquals(events[0].properties, { a: 'foo', b: '', c: 'boo' })
		} finally {
			globalThis.location = location
			if (a != null) {
				a.close()
			}
			fetch.restore()
		}
	})

	await t.step('sessions without auto tracking', async () => {
		const time = new FakeTime()
		const a = await newAnalytics({ sessions: { autoTrack: false } }, null, 10)
		time.tick(10)
		const fetch = new fake.Fetch(writeKey, endpoint + 'b', false, DEBUG)
		fetch.install()
		try {
			assertEquals(a.getSessionId(), null)
			time.tick(fiveMinutes)
			assertEquals(a.getSessionId(), null)
			time.tick(thirtyMinutes)
			assertEquals(a.getSessionId(), null)
			void a.track('click')
			time.tick(100)
			assertEquals(a.getSessionId(), null)
			time.tick(300)
			let events = await fetch.events(1)
			assertEquals(events.length, 1)
			a.startSession(728472643)
			assertEquals(a.getSessionId(), 728472643)
			time.tick(2 * thirtyMinutes)
			assertEquals(a.getSessionId(), 728472643)
			a.endSession()
			assertEquals(a.getSessionId(), null)
			void a.track('click')
			time.tick(100)
			assertEquals(a.getSessionId(), null)
			time.tick(300)
			a.startSession(728819037)
			assertEquals(a.getSessionId(), 728819037)
			time.tick(100)
			a.reset()
			assertNotEquals(a.getSessionId(), 728819037)
			await time.nextAsync()
			events = await fetch.events(1)
			assertEquals(events.length, 1)
		} finally {
			fetch.restore()
			time.restore()
		}
		a.close()
	})

	// Test identify and reset with each strategy, both with and without sessions.
	for (const strategy of ['ABC', 'AB-C', 'A-B-C', 'AC-B']) {
		for (const autoTrack of [true, false]) {
			await t.step(`strategy ${strategy} with${autoTrack ? '' : 'out'} sessions`, async () => {
				const a = newAnalytics({ sessions: { autoTrack } }, strategy)
				await a.ready()

				const time = new FakeTime()
				const fetch = new fake.Fetch(writeKey, endpoint + 'b', false, DEBUG)
				fetch.install()

				try {
					let sessionId = a.getSessionId()
					time.tick(1000)
					let anonymousId = a.getAnonymousId()
					const userTraits = { score: 729 }
					a.user().traits(userTraits)
					const groupId = 'acme'
					a.group().id(groupId)
					const groupTraits = { name: 'Acme' }
					a.group().traits(groupTraits)

					const original = { sessionId, anonymousId, userTraits, groupId, groupTraits }

					// identity.
					void a.identify('5F20MB18', { name: 'Susan' })
					time.tick(1000)
					let events = await fetch.events(1)
					let event = events[0]

					assertEquals(event.userId, '5F20MB18')
					if (!autoTrack) {
						assert(!('sessionId' in event.context))
						assert(!('sessionStart' in event.context))
						assertEquals(a.getSessionId(), null)
					}
					if (strategy.includes('-B')) {
						if (autoTrack) {
							assertNotEquals(event.context.sessionId, sessionId)
						}
						assertNotEquals(event.anonymousId, anonymousId)
						assertEquals(event.traits, { name: 'Susan' })
						assertEquals(a.group().id(), null)
						assertEquals(a.group().traits(), {})
					} else {
						if (autoTrack) {
							assertEquals(event.context.sessionId, sessionId)
						}
						assertEquals(event.anonymousId, anonymousId)
						assertEquals(event.traits, { name: 'Susan', score: 729 })
						assertEquals(a.group().id(), groupId)
						assertEquals(a.group().traits(), groupTraits)
					}
					assertEquals(a.getAnonymousId(), event.anonymousId)
					assertEquals(a.user().id(), event.userId)
					assertEquals(a.user().traits(), event.traits)

					sessionId = a.getSessionId()
					anonymousId = a.getAnonymousId()

					a.reset()

					if (!autoTrack) {
						assertEquals(a.getSessionId(), null)
					}
					if (strategy === 'AC-B') {
						if (autoTrack) {
							assertEquals(a.getSessionId(), original.sessionId)
						}
						assertEquals(a.getAnonymousId(), original.anonymousId)
						assertEquals(a.user().traits(), original.userTraits)
						assertEquals(a.group().id(), original.groupId)
						assertEquals(a.group().traits(), original.groupTraits)
					} else if (strategy.includes('-C')) {
						if (autoTrack) {
							assertNotEquals(a.getSessionId(), original.sessionId)
							assertNotEquals(a.getSessionId(), sessionId)
						}
						assertNotEquals(a.getAnonymousId(), original.anonymousId)
						assertNotEquals(a.getAnonymousId(), anonymousId)
						assertEquals(a.user().traits(), {})
						assertEquals(a.group().id(), null)
						assertEquals(a.group().traits(), {})
					} else {
						if (autoTrack) {
							assertEquals(a.getSessionId(), sessionId)
						}
						assertEquals(a.getAnonymousId(), anonymousId)
						assertEquals(a.user().traits(), {})
						assertEquals(a.group().id(), null)
						assertEquals(a.group().traits(), {})
					}

					// test reset with "all".
					a.startSession(215271912)
					anonymousId = a.getAnonymousId()
					time.tick(100)
					a.reset(true)
					assertNotEquals(anonymousId, a.getAnonymousId())
					assertNotEquals(a.getSessionId(), 215271912)

					a.endSession()
					a.reset(true)
					assertEquals(a.getSessionId(), null)
				} finally {
					fetch.restore()
					time.restore()
				}

				a.close()
			})
		}
	}

	await t.step('changing User ID, resets traits and Anonymous ID', async () => {
		const time = new FakeTime()
		const fetch = new fake.Fetch(writeKey, endpoint + 'b', false, DEBUG)
		fetch.install()
		const a = newAnalytics({ sessions: { autoTrack: false } })
		try {
			a.user().id('274084295')
			a.user().traits({ first_name: 'Susan' })
			const anonymousId = a.getAnonymousId()
			void a.identify('920577314')
			time.tick(1000)
			const events = await fetch.events(1)
			assertEquals(a.user().traits(), {})
			const newAnonymousId = a.getAnonymousId()
			assertNotEquals(newAnonymousId, anonymousId)
			assertEquals(events[0].traits, {})
			assertEquals(events[0].anonymousId, newAnonymousId)
		} finally {
			fetch.restore()
			time.restore()
		}
		a.close()
	})

	await t.step('after hiding the page, the queue is immediately persisted in the localStorage', () => {
		localStorage.clear()
		const time = new FakeTime()
		const sendBeacon = new fake.SendBeacon(writeKey, endpoint + 'b', DEBUG)
		sendBeacon.install()
		const a = newAnalytics()
		try {
			time.tick(200)
			void a.track('click')
			assert(!Object.keys(localStorage).some((key) => key.endsWith('.queue')))
			document.visibilityState = 'hidden'
			let isPersisted = false
			for (let i = 0; i < localStorage.length; i++) {
				const key = localStorage.key(i)
				if (key.endsWith('.queue')) {
					isPersisted = localStorage.getItem(key).length > 0
					break
				}
			}
			assert(isPersisted)
			document.visibilityState = 'visible'
			dispatchEvent(new Event('visibilitychange'))
		} finally {
			a.close()
			sendBeacon.restore()
			time.restore()
		}
	})

	await t.step('after hiding the page, the queue is immediately flushed', async () => {
		localStorage.clear()
		const time = new FakeTime()
		const fetch = new fake.Fetch(writeKey, endpoint + 'b', true, DEBUG)
		fetch.install()
		const a = newAnalytics()
		try {
			time.tick(200)
			void a.track('click')
			document.visibilityState = 'hidden'
			const events = await fetch.events(1)
			assertEquals(events.length, 1)
		} finally {
			a.close()
			fetch.restore()
			time.restore()
		}
	})

	// Execute the steps in the 'analytics_test_steps.js' module.
	const fetch = new fake.Fetch(writeKey, endpoint + 'b', false, DEBUG)
	const randomUUID = new fake.RandomUUID('9587b6d1-ae92-4d3c-a8d9-87c3e9ce7ae3')
	const navigator = new fake.Navigator()
	const now = new Date('2024-01-01T00:00:00Z')
	for (let i = 0; i < steps.length; i++) {
		const step = steps[i]
		await t.step(step.name, async () => {
			localStorage.clear()
			const time = new FakeTime(now)
			fetch.install()
			randomUUID.install()
			navigator.install()
			const a = await newAnalytics(step.options, null, 0)
			try {
				time.next()
				a.setAnonymousId('1b82c7e4-00b7-45d1-bbe2-6375fa9f8fa7')
				if (step.options?.sessions?.autoTrack !== false) {
					// Start a session and sent an event to mark it as not just started.
					a.startSession(1704070861000)
					void a.page('Home')
					time.tick(1000)
					await fetch.events(1)
				} else {
					time.tick(1000)
				}
				try {
					await step.call(a)
				} catch (error) {
					time.tick(1000)
					if (step.error) {
						assertEquals(Object.getPrototypeOf(error), Object.getPrototypeOf(step.error))
						assertEquals(error.message, step.error.message)
						return
					}
					throw new AssertionError(`unexpected error from step '${step.name}': ${error}`)
				}
				time.tick(1000)
				if (step.error) {
					throw new AssertionError(`expected error '${step.error}' from step '${step.name}', got no errors`)
				}
				const events = await fetch.events(1)
				assertEquals(events.length, 1)
				assertEquals(events[0], step.event)
			} finally {
				time.restore()
				navigator.restore()
				randomUUID.restore()
				fetch.restore()
			}
			a.close()
		})
	}
})
