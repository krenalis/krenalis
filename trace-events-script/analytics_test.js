import { assert, assertEquals, AssertionError, assertNotEquals } from 'https://deno.land/std@0.212.0/assert/mod.ts'
import { FakeTime } from 'https://deno.land/std@0.212.0/testing/time.ts'
import { DOMParser, HTMLDocument } from 'https://deno.land/x/deno_dom/deno-dom-wasm.ts'
import * as uuid from 'https://deno.land/std@0.212.0/uuid/v4.ts'
import * as fake from './test_fake.js'
import { steps } from './analytics_test_steps.js'
import { getTime } from './utils.js'
import Analytics from './analytics.js'

const DEBUG = false

const writeKey = 'rq6JJg5ENWK28NHfxSwJZmzeIvDC8GQO'
const endpoint = 'https://example.com/api/v1/batch'

Deno.test('Analytics', async (t) => {
	// Prepare the execution environment.
	{
		globalThis.navigator.onLine = true
		assert(globalThis.navigator.onLine)

		// Mock document.
		const doc = new DOMParser().parseFromString(
			`<!DOCTYPE html>
						<html lang="en">
					<head>
						<title>Hello from Chichi</title>
					</head>
					<body>
						<h1>Hello from Deno</h1>
						<form>
							<input name="user">
							<button>Submit</button>
						</form>
					</body>
				</html>`,
			'text/html',
		)
		globalThis.document = doc
		assert(document instanceof HTMLDocument)

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

	const minute = 60 * 1000
	const thirtyMinutes = 30 * minute
	const fiveMinutes = 5 * minute

	function newAnalytics(options) {
		localStorage.clear()
		const analytics = new Analytics(writeKey, endpoint, options)
		analytics.debug(DEBUG)
		return analytics
	}

	await t.step('ready callbacks are invoked', async () => {
		let ready1, ready2
		const p1 = new Promise((resolve) => {
			ready1 = resolve
		})
		const p2 = new Promise((resolve) => {
			ready2 = resolve
		})
		const a = newAnalytics()
		a.ready(() => {
			ready1()
		})
		a.ready(() => {
			ready2()
		})
		await p1
		await p2
	})

	await t.step('no key is created in the localStorage', () => {
		newAnalytics({ sessions: { autoTrack: false } })
		assertEquals(localStorage.length, 0)
	})

	await t.step('reset function', async () => {
		const fetch = new fake.Fetch(writeKey, endpoint, false, DEBUG)
		const a = newAnalytics({ strategy: 'AC-B', sessions: { autoTrack: false } })
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
		localStorage.removeItem('chichi_queue')
		assertEquals(localStorage.length, 0)
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
	})

	await t.step('sessions with auto tracking', () => {
		const time = new FakeTime()
		const fetch = new fake.Fetch(writeKey, endpoint, false, DEBUG)
		fetch.install()
		try {
			for (const option of [null, { sessions: { autoTrack: true } }]) {
				const a = newAnalytics(option)
				let sessionId = getTime()
				assertEquals(a.getSessionId(), sessionId)
				time.tick(fiveMinutes)
				assertEquals(a.getSessionId(), sessionId)
				time.tick(thirtyMinutes)
				assertEquals(a.getSessionId(), null)
				void a.track('click')
				sessionId = getTime()
				assertEquals(a.getSessionId(), sessionId)
				a.reset()
				assertEquals(a.getSessionId(), null)
			}
		} finally {
			fetch.restore()
			time.restore()
		}
	})

	await t.step('sessions without auto tracking', async () => {
		const time = new FakeTime()

		const fetch = new fake.Fetch(writeKey, endpoint, false, DEBUG)
		fetch.install()

		const a = newAnalytics({ sessions: { autoTrack: false } })

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
			a.reset()
			assertEquals(a.getSessionId(), null)
			events = await fetch.events(1)
			assertEquals(events.length, 1)
		} finally {
			fetch.restore()
			time.restore()
		}
	})

	// Test identify and anonymize with each strategy, both with and without sessions.
	for (const strategy of ['ABC', 'AB-C', 'A-B-C', 'AC-B']) {
		for (const autoTrack of [true, false]) {
			await t.step(`strategy ${strategy} with${autoTrack ? '' : 'out'} sessions`, async () => {
				const time = new FakeTime()

				const fetch = new fake.Fetch(writeKey, endpoint, false, DEBUG)
				fetch.install()

				const a = newAnalytics({ strategy, sessions: { autoTrack } })

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

					// anonymize.
					void a.anonymize()
					time.tick(1000)
					events = await fetch.events(1)
					event = events[0]

					assertEquals(event.userId, null)
					assertEquals(event.traits, undefined)
					if (autoTrack) {
						assertEquals(event.context.sessionId, sessionId)
					} else {
						assert(!('sessionId' in event.context))
						assert(!('sessionStart' in event.context))
						assertEquals(a.getSessionId(), null)
					}
					assertEquals(event.anonymousId, anonymousId)
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
				} finally {
					fetch.restore()
					time.restore()
				}
			})
		}
	}

	await t.step('changing User ID, resets traits and Anonymous ID', async () => {
		const time = new FakeTime()

		const fetch = new fake.Fetch(writeKey, endpoint, false, DEBUG)
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
	})

	// Execute the steps in the 'analytics_test_steps.js' module.
	const fetch = new fake.Fetch(writeKey, endpoint, false, DEBUG)
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
			try {
				const analytics = new Analytics(writeKey, endpoint, step.options)
				analytics.debug(DEBUG)
				analytics.setAnonymousId('1b82c7e4-00b7-45d1-bbe2-6375fa9f8fa7')
				if (step.options?.sessions?.autoTrack !== false) {
					// Start a session and sent an event to mark it as not just started.
					analytics.startSession(1704070861000)
					void analytics.page('Home')
					time.tick(1000)
					await fetch.events(1)
				} else {
					time.tick(1000)
				}
				try {
					await step.call(analytics)
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
		})
	}
})
