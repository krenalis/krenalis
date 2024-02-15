import { assert, assertEquals } from 'https://deno.land/std@0.212.0/assert/mod.ts'
import * as uuid from 'https://deno.land/std@0.212.0/uuid/v4.ts'
import { _uuid_imp, campaign, decodeBase64, encodeBase64, onVisibilityChange } from './utils.js'

Deno.test('utils', async (t) => {
	await t.step('campaign function', () => {
		globalThis.location = { search: '' }
		assertEquals(campaign(), undefined)

		globalThis.location = { search: '?' }
		assertEquals(campaign(), undefined)

		globalThis.location = { search: '?a=b&c=d' }
		assertEquals(campaign(), undefined)

		globalThis.location = { search: '?utm_medium=social+network&utm_source=social&utm_campaign=paid' }
		assertEquals(campaign(), { 'medium': 'social network', 'source': 'social', 'name': 'paid' })
	})

	await t.step('decodeBase64', () => {
		// With TextDecoder.
		assertEquals(decodeBase64(''), '')
		assertEquals(decodeBase64('YQ'), 'a')
		assertEquals(
			decodeBase64('SGVsbG8hIPCfmIogVGhpcyBpcyBhIHRlc3QuIOS9oOWlvSDwn4yN'),
			'Hello! 😊 This is a test. 你好 🌍',
		)
		assertEquals(
			decodeBase64(
				'VGhlIHN1biBzZXRzIGJlaGluZCB0aGUgbW91bnRhaW5zLiDwn4yEIExldCdzIGdvIGZvciBhIPCfmpcgcmlkZSE',
			),
			"The sun sets behind the mountains. 🌄 Let's go for a 🚗 ride!",
		)
		// Without TextDecoder or TextEncoder.
		const fns = ['TextDecoder', 'TextEncoder']
		for (const i in fns) {
			const fn = globalThis[fns[i]]
			globalThis[fns[i]] = null
			try {
				assertEquals(decodeBase64(''), '')
				assertEquals(decodeBase64('_YQA'), 'a')
				assertEquals(
					decodeBase64('_SABlAGwAbABvACEAIAA92AreIABUAGgAaQBzACAAaQBzACAAYQAgAHQAZQBzAHQALgAgAGBPfVkgADzYDd8'),
					'Hello! 😊 This is a test. 你好 🌍',
				)
				assertEquals(
					decodeBase64(
						'_VABoAGUAIABzAHUAbgAgAHMAZQB0AHMAIABiAGUAaABpAG4AZAAgAHQAaABlACAAbQBvAHUAbgB0AGEAaQBuAHMALgAgADzYBN8gAEwAZQB0ACcAcwAgAGcAbwAgAGYAbwByACAAYQAgAD3Yl94gAHIAaQBkAGUAIQA',
					),
					"The sun sets behind the mountains. 🌄 Let's go for a 🚗 ride!",
				)
			} finally {
				globalThis[fns[i]] = fn
			}
		}
	})

	await t.step('encodeBase64', () => {
		// With TextDecoder.
		assertEquals(encodeBase64(''), '')
		assertEquals(encodeBase64('a'), 'YQ')
		assertEquals(
			encodeBase64('Hello! 😊 This is a test. 你好 🌍'),
			'SGVsbG8hIPCfmIogVGhpcyBpcyBhIHRlc3QuIOS9oOWlvSDwn4yN',
		)
		assertEquals(
			encodeBase64("The sun sets behind the mountains. 🌄 Let's go for a 🚗 ride!"),
			'VGhlIHN1biBzZXRzIGJlaGluZCB0aGUgbW91bnRhaW5zLiDwn4yEIExldCdzIGdvIGZvciBhIPCfmpcgcmlkZSE',
		)
		// Without TextDecoder or TextEncoder.
		const fns = ['TextDecoder', 'TextEncoder']
		for (const i in fns) {
			const fn = globalThis[fns[i]]
			globalThis[fns[i]] = null
			try {
				assertEquals(encodeBase64(''), '')
				assertEquals(encodeBase64('a'), '_YQA')
				assertEquals(
					encodeBase64('Hello! 😊 This is a test. 你好 🌍'),
					'_SABlAGwAbABvACEAIAA92AreIABUAGgAaQBzACAAaQBzACAAYQAgAHQAZQBzAHQALgAgAGBPfVkgADzYDd8',
				)
				assertEquals(
					encodeBase64("The sun sets behind the mountains. 🌄 Let's go for a 🚗 ride!"),
					'_VABoAGUAIABzAHUAbgAgAHMAZQB0AHMAIABiAGUAaABpAG4AZAAgAHQAaABlACAAbQBvAHUAbgB0AGEAaQBuAHMALgAgADzYBN8gAEwAZQB0ACcAcwAgAGcAbwAgAGYAbwByACAAYQAgAD3Yl94gAHIAaQBkAGUAIQA',
				)
			} finally {
				globalThis[fns[i]] = fn
			}
		}
	})

	await t.step('onVisibilityChange function', () => {
		globalThis.document = {
			visibilityState: 'visible',
			addEventListener: globalThis.addEventListener.bind(globalThis),
		}
		let isPageVisible = true
		onVisibilityChange((visible) => {
			assertEquals(visible, !isPageVisible)
			isPageVisible = visible
		})
		globalThis.document.visibilityState = 'hidden'
		globalThis.dispatchEvent(new Event('visibilitychange'))
		assertEquals(isPageVisible, false)
		globalThis.document.visibilityState = 'visible'
		globalThis.dispatchEvent(new Event('visibilitychange'))
		assertEquals(isPageVisible, true)
		globalThis.document.visibilityState = 'hidden'
		globalThis.dispatchEvent(new Event('pagehide'))
		assertEquals(isPageVisible, false)
		globalThis.document.visibilityState = 'visible'
		globalThis.dispatchEvent(new Event('pageshow'))
		assertEquals(isPageVisible, true)
	})

	await t.step('uuid function', () => {
		const _randomUUID = globalThis.crypto.randomUUID.bind(globalThis.crypto)
		const _getRandomValues = globalThis.crypto.getRandomValues.bind(globalThis.crypto)
		const _URL = globalThis.URL

		// Prepare the execution environment.
		{
			globalThis.crypto.randomUUID = undefined
			assertEquals(globalThis.crypto.randomUUID, undefined)

			globalThis.crypto.getRandomValues = undefined
			assertEquals(globalThis.crypto.getRandomValues, undefined)

			globalThis.URL = undefined
			assertEquals(globalThis.URL, undefined)
		}

		// Test crypto.randomUUID implementation.
		globalThis.crypto.randomUUID = _randomUUID
		assert(uuid.validate(_uuid_imp()()))
		globalThis.crypto.randomUUID = undefined

		// Test crypto.getRandomValues implementation.
		globalThis.crypto.getRandomValues = _getRandomValues
		assert(uuid.validate(_uuid_imp()()))
		globalThis.crypto.getRandomValues = undefined

		// Test msCrypto.getRandomValues implementation.
		globalThis.msCrypto = { getRandomValues: _getRandomValues }
		assert(uuid.validate(_uuid_imp()()))
		delete globalThis.msCrypto

		// Test URL implementation.
		globalThis.URL = _URL
		assert(uuid.validate(_uuid_imp()()))
	})
})
