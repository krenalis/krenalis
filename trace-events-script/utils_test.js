import { assert } from 'https://deno.land/std@0.212.0/assert/mod.ts';
import * as uuid from 'https://deno.land/std@0.212.0/uuid/v4.ts';
import { _uuid_imp } from './utils.js';

Deno.test('UUID generation', () => {
	const _randomUUID = window.crypto.randomUUID.bind(window.crypto);
	window.crypto.randomUUID = undefined;

	const _getRandomValues = window.crypto.getRandomValues.bind(window.crypto);
	window.crypto.getRandomValues = undefined;

	const _URL = window.URL;
	window.URL = undefined;

	// Test crypto.randomUUID implementation.
	window.crypto.randomUUID = _randomUUID;
	assert(uuid.validate(_uuid_imp()()));
	window.crypto.randomUUID = undefined;

	// Test crypto.getRandomValues implementation.
	window.crypto.getRandomValues = _getRandomValues;
	assert(uuid.validate(_uuid_imp()()));
	window.crypto.getRandomValues = undefined;

	// Test msCrypto.getRandomValues implementation.
	window.msCrypto = { getRandomValues: _getRandomValues };
	assert(uuid.validate(_uuid_imp()()));
	delete window.msCrypto;

	// Test URL implementation.
	window.URL = _URL;
	assert(uuid.validate(_uuid_imp()()));
});
