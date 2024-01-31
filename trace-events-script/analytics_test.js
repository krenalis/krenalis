import { assert, assertEquals, AssertionError } from 'https://deno.land/std@0.212.0/assert/mod.ts';
import { FakeTime } from 'https://deno.land/std@0.212.0/testing/time.ts';
import { DOMParser, HTMLDocument } from 'https://deno.land/x/deno_dom/deno-dom-wasm.ts';
import * as uuid from 'https://deno.land/std@0.212.0/uuid/v4.ts';
import * as fake from './test_fake.js';
import { steps } from './analytics_test_steps.js';
import { getTime } from './utils.js';
import Analytics from './analytics.js';

const DEBUG = false;

const writeKey = 'rq6JJg5ENWK28NHfxSwJZmzeIvDC8GQO';
const endpoint = 'https://example.com/api/v1/batch';

Deno.test('Analytics', async (t) => {
	// Prepare the execution environment.
	{
		globalThis.navigator.onLine = true;
		assert(globalThis.navigator.onLine);

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
		);
		globalThis.document = doc;
		assert(document instanceof HTMLDocument);

		// Mock location.
		const url = new URL('https://example.com:8080/path?query=123#fragment');
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
		};

		// Mock screen.
		globalThis.screen = { width: 2560, height: 1440 };

		// Mock devicePixelRatio.
		globalThis.devicePixelRatio = 1.25;
	}

	const minute = 60 * 1000;
	const thirtyMinutes = 30 * minute;
	const fiveMinutes = 5 * minute;

	function newAnalytics(options) {
		localStorage.clear();
		const analytics = new Analytics(writeKey, endpoint, options);
		analytics.debug(DEBUG);
		return analytics;
	}

	await t.step('ready callbacks are invoked', async () => {
		let ready1, ready2;
		const p1 = new Promise((resolve) => {
			ready1 = resolve;
		});
		const p2 = new Promise((resolve) => {
			ready2 = resolve;
		});
		const a = newAnalytics();
		a.ready(() => {
			ready1();
		});
		a.ready(() => {
			ready2();
		});
		await p1;
		await p2;
	});

	await t.step('getAnonymousId function', () => {
		const a = newAnalytics();
		assert(uuid.validate(a.getAnonymousId()));
		a.setAnonymousId('f5d354ed');
		assertEquals(a.getAnonymousId(), 'f5d354ed');
		a.setAnonymousId(903726473);
		assertEquals(a.getAnonymousId(), '903726473');
		a.setAnonymousId('');
		assert(uuid.validate(a.getAnonymousId()));
		a.setAnonymousId({});
		assert(uuid.validate(a.getAnonymousId()));
	});

	await t.step('setAnonymousId function', () => {
		const a = newAnalytics();
		assert(uuid.validate(a.setAnonymousId()));
		const anonymousId = 'f5d354ed';
		assertEquals(a.setAnonymousId(anonymousId), anonymousId);
		assertEquals(a.setAnonymousId(), anonymousId);
		assertEquals(a.setAnonymousId(903726473), '903726473');
		assertEquals(a.setAnonymousId(), '903726473');
		assertEquals(a.setAnonymousId(''), '');
		assert(uuid.validate(a.setAnonymousId()));
		assertEquals(a.setAnonymousId({}), {});
		assert(uuid.validate(a.setAnonymousId()));
	});

	await t.step('user function', () => {
		const a = newAnalytics();

		assertEquals(a.user().id(), null);
		assert(uuid.validate(a.user().anonymousId()));
		assertEquals(a.user().traits(), {});

		assertEquals(a.user().id('8g1emx962iR'), '8g1emx962iR');
		assertEquals(a.user().id(), '8g1emx962iR');
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
			{ set: { foo: () => {}, boo: true }, expect: { boo: true } },
		];

		for (let i = 0; i < changes.length; i++) {
			const change = changes[i];
			assertEquals(a.user().traits(change.set), change.expect);
			assertEquals(a.user().traits(), change.expect);
		}
	});

	await t.step('sessions with auto tracking', () => {
		const time = new FakeTime();
		const fetch = new fake.Fetch(writeKey, endpoint);
		fetch.install();
		try {
			for (const option of [null, { sessions: { autoTrack: true } }]) {
				const a = newAnalytics(option);
				let sessionId = getTime();
				assertEquals(a.getSessionId(), sessionId);
				time.tick(fiveMinutes);
				assertEquals(a.getSessionId(), sessionId);
				time.tick(thirtyMinutes);
				assertEquals(a.getSessionId(), null);
				void a.track('click');
				sessionId = getTime();
				assertEquals(a.getSessionId(), sessionId);
				a.reset();
				assertEquals(a.getSessionId(), null);
			}
		} finally {
			fetch.restore();
		}
	});

	await t.step('sessions without auto tracking', async () => {
		const time = new FakeTime();

		const fetch = new fake.Fetch(writeKey, endpoint);
		fetch.install();

		const a = newAnalytics({ sessions: { autoTrack: false } });

		try {
			assertEquals(a.getSessionId(), null);
			time.tick(fiveMinutes);
			assertEquals(a.getSessionId(), null);
			time.tick(thirtyMinutes);
			assertEquals(a.getSessionId(), null);
			void a.track('click');
			time.tick(100);
			assertEquals(a.getSessionId(), null);
			time.tick(300);
			let events = await fetch.events(1);
			assertEquals(events.length, 1);

			a.startSession(728472643);
			assertEquals(a.getSessionId(), 728472643);
			time.tick(2 * thirtyMinutes);
			assertEquals(a.getSessionId(), 728472643);
			a.endSession();
			assertEquals(a.getSessionId(), null);
			void a.track('click');
			time.tick(100);
			assertEquals(a.getSessionId(), null);
			time.tick(300);
			a.startSession(728819037);
			assertEquals(a.getSessionId(), 728819037);
			a.reset();
			assertEquals(a.getSessionId(), null);
			events = await fetch.events(1);
			assertEquals(events.length, 1);
		} finally {
			fetch.restore();
		}
	});

	// Execute the steps in the 'analytics_test_steps.js' module.
	const fetch = new fake.Fetch(writeKey, endpoint);
	const randomUUID = new fake.RandomUUID('9587b6d1-ae92-4d3c-a8d9-87c3e9ce7ae3');
	const navigator = new fake.Navigator();
	const now = new Date('2024-01-01T00:00:00Z');
	for (let i = 0; i < steps.length; i++) {
		const step = steps[i];
		await t.step(step.name, async () => {
			localStorage.clear();
			const time = new FakeTime(now);
			fetch.install();
			randomUUID.install();
			navigator.install();
			try {
				const analytics = new Analytics(writeKey, endpoint, step.options);
				analytics.setAnonymousId('1b82c7e4-00b7-45d1-bbe2-6375fa9f8fa7');
				if (step.options?.sessions?.autoTrack !== false) {
					// Start a session and sent an event to mark it as not just started.
					analytics.startSession(1704070861000);
					void analytics.page('Home');
					time.tick(1000);
					await fetch.events(1);
				} else {
					time.tick(1000);
				}
				try {
					await step.call(analytics);
				} catch (error) {
					if (step.error) {
						assertEquals(Object.getPrototypeOf(error), Object.getPrototypeOf(step.error));
						assertEquals(error.message, step.error.message);
						return;
					}
					throw new AssertionError(`unexpected error from step '${step.name}': ${error}`);
				}
				time.tick(300);
				const events = await fetch.events(1);
				assertEquals(events.length, 1);
				assertEquals(events[0], step.event);
			} finally {
				time.restore();
				navigator.restore();
				randomUUID.restore();
				fetch.restore();
			}
		});
	}
});
