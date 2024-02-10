import { assert, assertEquals } from 'https://deno.land/std@0.212.0/assert/mod.ts';
import * as uuid from 'https://deno.land/std@0.212.0/uuid/v4.ts';
import { _uuid_imp, campaign, onVisibilityChange } from './utils.js';

Deno.test('utils', async (t) => {
	await t.step('campaign function', () => {
		globalThis.location = { search: '' };
		assertEquals(campaign(), undefined);

		globalThis.location = { search: '?' };
		assertEquals(campaign(), undefined);

		globalThis.location = { search: '?a=b&c=d' };
		assertEquals(campaign(), undefined);

		globalThis.location = { search: '?utm_medium=social+network&utm_source=social&utm_campaign=paid' };
		assertEquals(campaign(), { 'medium': 'social network', 'source': 'social', 'name': 'paid' });
	});

	await t.step('onVisibilityChange function', () => {
		globalThis.document = { visibilityState: 'visible' };
		let isPageVisible = true;
		onVisibilityChange((visible) => {
			assertEquals(visible, !isPageVisible);
			isPageVisible = visible;
		});
		globalThis.document.visibilityState = 'hidden';
		globalThis.dispatchEvent(new Event('visibilitychange'));
		assertEquals(isPageVisible, false);
		globalThis.document.visibilityState = 'visible';
		globalThis.dispatchEvent(new Event('visibilitychange'));
		assertEquals(isPageVisible, true);
		globalThis.document.visibilityState = 'hidden';
		globalThis.dispatchEvent(new Event('pagehide'));
		assertEquals(isPageVisible, false);
		globalThis.document.visibilityState = 'visible';
		globalThis.dispatchEvent(new Event('pageshow'));
		assertEquals(isPageVisible, true);
	});

	await t.step('uuid function', () => {
		const _randomUUID = globalThis.crypto.randomUUID.bind(globalThis.crypto);
		const _getRandomValues = globalThis.crypto.getRandomValues.bind(globalThis.crypto);
		const _URL = globalThis.URL;

		// Prepare the execution environment.
		{
			globalThis.crypto.randomUUID = undefined;
			assertEquals(globalThis.crypto.randomUUID, undefined);

			globalThis.crypto.getRandomValues = undefined;
			assertEquals(globalThis.crypto.getRandomValues, undefined);

			globalThis.URL = undefined;
			assertEquals(globalThis.URL, undefined);
		}

		// Test crypto.randomUUID implementation.
		globalThis.crypto.randomUUID = _randomUUID;
		assert(uuid.validate(_uuid_imp()()));
		globalThis.crypto.randomUUID = undefined;

		// Test crypto.getRandomValues implementation.
		globalThis.crypto.getRandomValues = _getRandomValues;
		assert(uuid.validate(_uuid_imp()()));
		globalThis.crypto.getRandomValues = undefined;

		// Test msCrypto.getRandomValues implementation.
		globalThis.msCrypto = { getRandomValues: _getRandomValues };
		assert(uuid.validate(_uuid_imp()()));
		delete globalThis.msCrypto;

		// Test URL implementation.
		globalThis.URL = _URL;
		assert(uuid.validate(_uuid_imp()()));
	});
});
