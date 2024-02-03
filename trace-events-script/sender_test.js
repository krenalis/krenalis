import { assert, assertEquals } from 'https://deno.land/std@0.212.0/assert/mod.ts';
import { FakeTime } from 'https://deno.land/std@0.212.0/testing/time.ts';
import * as fake from './test_fake.js';
import { Sender } from './sender.js';

const DEBUG = false;

const writeKey = 'rq6JJg5ENWK28NHfxSwJZmzeIvDC8GQO';
const endpoint = 'https://example.com/api/v1/batch';

Deno.test('Sender send', async (t) => {
	// Prepare the execution environment.
	{
		localStorage.clear();

		globalThis.navigator.onLine = true;
		assert(globalThis.navigator.onLine);
	}

	const events = [
		{ messageId: '53f6c7da-cf9c-4e8d-85e3-fa45a45b9221' },
		{ messageId: '53f6c7da-cf9c-4e8d-85e3-fa45a45b9221' },
		{ messageId: '2f825fe5-b492-4ddf-a58e-7c5567366870' },
		{ messageId: 'ba30a14a-3d9e-4985-a254-e6517c4a237c' },
	];

	await t.step('fetch', async () => {
		let time;
		const fetch = new fake.Fetch(writeKey, endpoint, DEBUG);

		try {
			time = new FakeTime();
			fetch.install();
			const sender = new Sender(writeKey, endpoint);
			sender.debug(DEBUG);
			for (let i = 0; i < events.length; i++) {
				sender.send(events[i]);
			}
			time.tick(sender.timeout);
			const sentEvents = await fetch.events(events.length);
			assertEquals(sentEvents.length, events.length);
			for (let i = 0; i < events.length; i++) {
				assertEquals(sentEvents[i], events[i]);
			}
		} finally {
			fetch.restore();
			time.restore();
		}

		try {
			time = new FakeTime();
			fetch.install();
			const sender = new Sender(writeKey, endpoint);
			sender.debug(DEBUG);
			const maxPerBatch = 9658; // This value can change if the sender's implementation change.
			// Send maxPerBatch events.
			for (let i = 0; i < maxPerBatch; i++) {
				sender.send({ messageId: crypto.randomUUID() });
			}
			time.tick(100);
			sender.send({ messageId: crypto.randomUUID() });
			sender.send({ messageId: crypto.randomUUID() });
			time.tick(sender.timeout - 100);
			time.tick(10);
			let events = await fetch.events(maxPerBatch);
			assertEquals(events.length, maxPerBatch);
			time.tick(150);
			events = await fetch.events(2);
			assertEquals(events.length, 2);
		} finally {
			fetch.restore();
			time.restore();
		}
	});

	await t.step('sendBeacon', async () => {
		const time = new FakeTime();
		const sendBeacon = new fake.SendBeacon(writeKey, endpoint, DEBUG);
		sendBeacon.install();
		try {
			const sender = new Sender(writeKey, endpoint);
			sender.debug(DEBUG);
			for (let i = 0; i < events.length; i++) {
				sender.send(events[i]);
			}
			globalThis.dispatchEvent(new Event('pagehide'));
			const sentEvents = await sendBeacon.events(events.length);
			assertEquals(sentEvents.length, events.length);
			for (let i = 0; i < events.length; i++) {
				assertEquals(sentEvents[i], events[i]);
			}
		} finally {
			sendBeacon.restore();
			time.restore();
		}
	});

	await t.step('XMLHttpRequest', async () => {
		const time = new FakeTime();
		fake.XMLHttpRequest.install(writeKey, endpoint, DEBUG);
		assertEquals(globalThis.XMLHttpRequest, XMLHttpRequest);
		const fetch = globalThis.fetch;
		globalThis.fetch = undefined;
		assertEquals(globalThis.fetch, undefined);
		try {
			const sender = new Sender(writeKey, endpoint);
			sender.debug(DEBUG);
			for (let i = 0; i < events.length; i++) {
				sender.send(events[i]);
			}
			time.tick(sender.timeout);
			const sentEvents = await fake.XMLHttpRequest.events(events.length);
			assertEquals(sentEvents.length, events.length);
			for (let i = 0; i < events.length; i++) {
				assertEquals(sentEvents[i], events[i]);
			}
		} finally {
			globalThis.fetch = fetch;
			fake.XMLHttpRequest.restore();
			time.restore();
		}
	});
});
