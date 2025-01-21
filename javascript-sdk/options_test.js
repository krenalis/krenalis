import { assert, assertEquals } from '@std/assert'
import { isStrategy, Options } from './options.js'

const days = 24 * 60 * 60 * 1000

Deno.test('Options', () => {
	localStorage.clear()

	const base = {
		cookie: {
			domain: null,
			maxAge: 365 * days,
			path: '/',
			sameSite: 'lax',
			secure: false,
		},
		debug: false,
		group: {
			storage: {
				stores: null,
			},
		},
		sessions: {
			autoTrack: true,
			timeout: 30 * 60000, // 30 minutes.
		},
		stores: ['localStorage', 'cookie', 'memory'],
		strategy: 'Conversion',
		useQueryString: {
			aid: /\s\S/,
			uid: /\s\S/,
		},
		user: {
			storage: {
				stores: null,
			},
		},
	}

	const tests = [
		// Invalid options.
		{ options: undefined, ...base },
		{ options: null, ...base },
		{ options: {}, ...base },
		{ options: [], ...base },
		{ options: '', ...base },

		// storage.
		{ options: { storage: null }, ...base },
		{ options: { storage: {} }, ...base },
		{ options: { storage: { cookie: null } }, ...base },
		{ options: { storage: { cookie: {} } }, ...base },

		// storage.cookie.
		{
			options: {
				storage: {
					cookie: {
						domain: null,
						maxage: 365 * days,
						path: '/',
						samesite: 'Lax',
						secure: false,
					},
				},
			},
			...base,
		},
		{
			options: {
				storage: { cookie: { domain: '', samesite: 'Strict', secure: true } },
			},
			...base,
			cookie: { ...base.cookie, domain: '', sameSite: 'strict', secure: true },
		},
		{
			options: {
				storage: {
					cookie: { domain: 'example.com', maxage: 30 * days, secure: {} },
				},
			},
			...base,
			cookie: {
				...base.cookie,
				domain: 'example.com',
				maxAge: 30 * days,
				secure: true,
			},
		},
		{
			options: { storage: { cookie: { path: '/store/' } } },
			...base,
			cookie: { ...base.cookie, path: '/store/' },
		},

		// storage.stores.
		{ options: { storage: { stores: null } }, ...base },
		{ options: { storage: { stores: [] } }, ...base, stores: [] },
		{
			options: {
				storage: {
					stores: [
						'cookie',
						'memory',
						'memory',
						'localStorage',
						'sessionStorage',
					],
				},
			},
			...base,
			stores: ['cookie', 'memory', 'localStorage', 'sessionStorage'],
		},

		// storage.type.
		{
			options: { storage: { type: 'cookieStorage' } },
			...base,
			stores: ['cookie', 'localStorage', 'sessionStorage', 'memory'],
		},
		{
			options: { storage: { type: 'localStorage' } },
			...base,
			stores: ['localStorage', 'memory'],
		},
		{
			options: { storage: { type: 'sessionStorage' } },
			...base,
			stores: ['sessionStorage', 'memory'],
		},
		{
			options: { storage: { type: 'memoryStorage' } },
			...base,
			stores: ['memory'],
		},
		{
			options: { storage: { type: 'none' } },
			...base,
			stores: [],
		},

		// cookie.
		{
			options: {
				cookie: {
					domain: null,
					maxage: 365,
					path: '/',
					sameSite: 'Lax',
					secure: false,
				},
			},
			...base,
		},
		{
			options: { cookie: { domain: '', sameSite: 'Strict', secure: true } },
			...base,
			cookie: { ...base.cookie, domain: '', sameSite: 'strict', secure: true },
		},
		{
			options: { cookie: { domain: 'example.com', maxage: 30, secure: {} } },
			...base,
			cookie: {
				...base.cookie,
				domain: 'example.com',
				maxAge: 30 * days,
				secure: true,
			},
		},
		{
			options: { cookie: { path: '/store/' } },
			...base,
			cookie: { ...base.cookie, path: '/store/' },
		},

		// debug.
		{ options: { debug: false }, ...base },
		{ options: { debug: true }, ...base, debug: true },
		{ options: { debug: 0 }, ...base },
		{ options: { debug: 1 }, ...base, debug: true },

		// group.storage.stores.
		{ options: { group: { storage: { stores: null } } }, ...base },
		{
			options: { group: { storage: { stores: [] } } },
			...base,
			group: { storage: { stores: [] } },
		},
		{
			options: { group: { storage: { stores: ['cookie', 'memory'] } } },
			...base,
			group: { storage: { stores: ['cookie', 'memory'] } },
		},

		// sameDomainCookiesOnly.
		{
			options: { sameDomainCookiesOnly: true },
			...base,
			cookie: { ...base.cookie, domain: '' },
		},
		{ options: { sameDomainCookiesOnly: false }, ...base },

		// secureCookie.
		{
			options: { secureCookie: true },
			...base,
			cookie: { ...base.cookie, secure: true },
		},
		{ options: { secureCookie: false }, ...base },

		// sameSiteCookie.
		{ options: { sameSiteCookie: 'Lax' }, ...base },
		{
			options: { sameSiteCookie: 'Strict' },
			...base,
			cookie: { ...base.cookie, sameSite: 'strict' },
		},
		{
			options: { sameSiteCookie: 'None' },
			...base,
			cookie: { ...base.cookie, sameSite: 'none' },
		},
		{ options: { sameSiteCookie: 'lax' }, ...base },
		{ options: { sameSiteCookie: 'strict' }, ...base },
		{ options: { sameSiteCookie: 'none' }, ...base },

		// sessions.
		{ options: { sessions: {} }, ...base },
		{ options: { sessions: { autoTrack: true } }, ...base },
		{
			options: { sessions: { autoTrack: false } },
			...base,
			sessions: { autoTrack: false, timeout: 30 * 60000 },
		},
		{ options: { sessions: { autoTrack: null } }, ...base },
		{
			options: { sessions: { timeout: 10 * 1000 } },
			...base,
			sessions: { autoTrack: true, timeout: 10 * 1000 },
		},
		{
			options: { sessions: { timeout: '5000' } },
			...base,
			sessions: { autoTrack: true, timeout: 5 * 1000 },
		},
		{
			options: { sessions: { timeout: Infinity } },
			...base,
			sessions: { autoTrack: true, timeout: Infinity },
		},
		{ options: { sessions: { timeout: {} } }, ...base },
		{
			options: { sessions: { timeout: 0 } },
			...base,
			sessions: { autoTrack: false, timeout: 1800000 },
		},
		{
			options: { sessions: { timeout: -5000 } },
			...base,
			sessions: { autoTrack: false, timeout: 1800000 },
		},
		{
			options: { sessions: { autoTrack: true, timeout: 20 * 1000 } },
			...base,
			sessions: { autoTrack: true, timeout: 20 * 1000 },
		},
		{
			options: { sessions: { autoTrack: true, timeout: 0 } },
			...base,
			sessions: { autoTrack: false, timeout: 1800000 },
		},

		// setCookieDomain.
		{ options: { setCookieDomain: null }, ...base },
		{
			options: { setCookieDomain: 'example.com' },
			...base,
			cookie: { ...base.cookie, domain: 'example.com' },
		},

		// useQueryString.
		{ options: { useQueryString: true }, ...base },
		{ options: { useQueryString: false }, ...base, useQueryString: null },
		{ options: { useQueryString: 0 }, ...base },
		{ options: { useQueryString: { aid: null } }, ...base },
		{ options: { useQueryString: { uid: 'a' } }, ...base },
		{
			options: { useQueryString: { uid: /^[a-z]+$/ } },
			...base,
			useQueryString: {
				aid: /\s\S/,
				uid: /^[a-z]+$/,
			},
		},
		{
			options: { useQueryString: { aid: /\S+/ } },
			...base,
			useQueryString: {
				aid: /\S+/,
				uid: /\s\S/,
			},
		},

		// user.storage.stores.
		{ options: { user: { storage: { stores: null } } }, ...base },
		{
			options: { user: { storage: { stores: [] } } },
			...base,
			user: { storage: { stores: [] } },
		},
		{
			options: { user: { storage: { stores: ['cookie', 'memory'] } } },
			...base,
			user: { storage: { stores: ['cookie', 'memory'] } },
		},
	]

	for (let i = 0; i < tests.length; i++) {
		const test = tests[i]
		const options = new Options(null, null, test.options)
		delete test.options
		assertEquals({ ...options }, test)
	}

	// Test invalid setCookieDomain values.
	let invalids = [
		'',
		{},
		'no',
		'storage',
		'cookiestorage',
		'AsessionStorage',
		true,
	]
	for (let i = 0; i < invalids.length; i++) {
		const type = invalids[i]
		const options = new Options(null, null, { storage: { type } })
		assertEquals(options.user.storage.stores, base.user.storage.stores)
		assertEquals(options.group.storage.stores, base.group.storage.stores)
	}

	// Test invalid setCookieDomain values.
	invalids = ['', {}, '127.0.0.1', 'example.com.', '%20', '=']
	for (let i = 0; i < invalids.length; i++) {
		const setCookieDomain = invalids[i]
		const options = new Options(null, null, { setCookieDomain })
		assertEquals(options.cookie.domain, base.cookie.domain)
	}

	// Test invalid SameSite values.
	invalids = ['', 8, [], true, 'no', ' Lax', 'other']
	for (let i = 0; i < invalids.length; i++) {
		const sameSiteCookie = invalids[i]
		const options = new Options(null, null, { sameSiteCookie })
		assertEquals(options.cookie.sameSite, base.cookie.sameSite)
	}
})

Deno.test('isStrategy', () => {
	assert(isStrategy('Fusion'))
	assert(isStrategy('Conversion'))
	assert(isStrategy('Isolation'))
	assert(isStrategy('Preservation'))
	assert(!isStrategy('Unification'))
	assert(!isStrategy('fusion'))
	assert(!isStrategy(' Preservation'))
	assert(!isStrategy('isolation'))
	assert(!isStrategy('Conversion '))
	assert(!isStrategy(''))
	assert(!isStrategy(5))
	assert(!isStrategy(null))
})
