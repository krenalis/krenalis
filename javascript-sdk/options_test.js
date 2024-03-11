import { assert, assertEquals, AssertionError } from 'std/assert/mod.ts'
import { isStrategy, Options } from './options.js'

const days = 24 * 60 * 60 * 1000

Deno.test('Options', () => {
	localStorage.clear()

	const base = {
		autoTrack: true,
		cookie: {
			domain: null,
			maxAge: 365 * days,
			path: '/',
			sameSite: 'lax',
			secure: false,
		},
		debug: false,
		sameDomainCookiesOnly: false,
		sameSiteCookie: 'lax',
		secureCookie: false,
		setCookieDomain: null,
		storage: {
			type: 'multiStorage',
		},
		timeout: 30 * 60000,
		useQueryString: {
			aid: /\s\S/,
			uid: /\s\S/,
		},
	}

	const tests = [
		{ options: undefined, ...base },
		{ options: null, ...base },
		{ options: {}, ...base },
		{ options: [], ...base },
		{ options: '', ...base },
		{ options: { storage: null }, ...base },
		{ options: { storage: {} }, ...base },
		{ options: { storage: { cookie: null } }, ...base },
		{ options: { storage: { cookie: {} } }, ...base },
		{
			options: { cookie: { domain: null, maxage: 365, path: '/', sameSite: 'Lax', secure: false } },
			...base,
		},
		{
			options: { storage: { cookie: { domain: null, maxage: 365 * days, path: '/', samesite: 'Lax', secure: false } } },
			...base,
		},
		{
			options: { cookie: { domain: '', sameSite: 'Strict', secure: true } },
			...base,
			cookie: { ...base.cookie, domain: '', sameSite: 'strict', secure: true },
		},
		{
			options: { storage: { cookie: { domain: '', samesite: 'Strict', secure: true } } },
			...base,
			cookie: { ...base.cookie, domain: '', sameSite: 'strict', secure: true },
		},
		{
			options: { cookie: { domain: 'example.com', maxage: 30, secure: {} } },
			...base,
			cookie: { ...base.cookie, domain: 'example.com', maxAge: 30 * days, secure: true },
		},
		{
			options: { storage: { cookie: { domain: 'example.com', maxage: 30 * days, secure: {} } } },
			...base,
			cookie: { ...base.cookie, domain: 'example.com', maxAge: 30 * days, secure: true },
		},
		{
			options: { cookie: { path: '/store/' } },
			...base,
			cookie: { ...base.cookie, path: '/store/' },
		},
		{
			options: { storage: { cookie: { path: '/store/' } } },
			...base,
			cookie: { ...base.cookie, path: '/store/' },
		},
		{ options: { storage: { type: 'multiStorage' } }, ...base },
		{
			options: { storage: { type: 'cookieStorage' } },
			...base,
			storage: { ...base.storage, type: 'cookieStorage' },
		},
		{
			options: { storage: { type: 'localStorage' } },
			...base,
			storage: { ...base.storage, type: 'localStorage' },
		},
		{
			options: { storage: { type: 'sessionStorage' } },
			...base,
			storage: { ...base.storage, type: 'sessionStorage' },
		},
		{
			options: { storage: { type: 'memoryStorage' } },
			...base,
			storage: { ...base.storage, type: 'memoryStorage' },
		},
		{ options: { storage: { type: 'none' } }, ...base, storage: { ...base.storage, type: 'none' } },
		{
			options: { sameDomainCookiesOnly: true },
			...base,
			cookie: { ...base.cookie, domain: '' },
		},
		{ options: { sameDomainCookiesOnly: false }, ...base },
		{
			options: { secureCookie: true },
			...base,
			cookie: { ...base.cookie, secure: true },
		},
		{ options: { secureCookie: false }, ...base },
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
		{ options: { sessions: {} }, ...base },
		{ options: { sessions: { autoTrack: true } }, ...base },
		{ options: { sessions: { autoTrack: false } }, ...base, autoTrack: false },
		{ options: { sessions: { autoTrack: null } }, ...base },
		{ options: { sessions: { timeout: 10 * 1000 } }, ...base, timeout: 10 * 1000 },
		{ options: { sessions: { timeout: '5000' } }, ...base, timeout: 5 * 1000 },
		{ options: { sessions: { timeout: Infinity } }, ...base, timeout: Infinity },
		{ options: { sessions: { timeout: {} } }, ...base },
		{ options: { sessions: { timeout: 0 } }, ...base, autoTrack: false },
		{ options: { sessions: { timeout: -5000 } }, ...base, autoTrack: false },
		{ options: { sessions: { autoTrack: true, timeout: 20 * 1000 } }, ...base, timeout: 20 * 1000 },
		{ options: { sessions: { autoTrack: true, timeout: 0 } }, ...base, autoTrack: false },
		{ options: { setCookieDomain: null }, ...base },
		{
			options: { setCookieDomain: 'example.com' },
			...base,
			cookie: { ...base.cookie, domain: 'example.com' },
		},
		{ options: { debug: false }, ...base },
		{ options: { debug: true }, ...base, debug: true },
		{ options: { debug: 0 }, ...base },
		{ options: { debug: 1 }, ...base, debug: true },
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
	]

	for (let i = 0; i < tests.length; i++) {
		const test = tests[i]
		const options = new Options(null, null, test.options)
		assertEquals(test.debug, test.debug)
		assertEquals(options.cookie.domain, test.cookie.domain)
		assertEquals(options.cookie.maxAge, test.cookie.maxAge)
		assertEquals(options.cookie.path, test.cookie.path)
		assertEquals(options.cookie.sameSite, test.cookie.sameSite)
		assertEquals(options.cookie.secure, test.cookie.secure)
		assertEquals(options.storage.type, test.storage.type)
		assertEquals(options.sessions.autoTrack, test.autoTrack)
		assertEquals(options.sessions.timeout, test.timeout)
		assertEquals(options.useQueryString, test.useQueryString)
	}

	// Test invalid setCookieDomain values.
	let invalids = ['', {}, 'no', 'storage', 'cookiestorage', 'AsessionStorage', true]
	for (let i = 0; i < invalids.length; i++) {
		const type = invalids[i]
		const options = new Options(null, null, { storage: { type } })
		if (options.storage.type !== base.storage.type) {
			throw new AssertionError(`'${type}' is not a type for the storage.type option`)
		}
	}

	// Test invalid setCookieDomain values.
	invalids = ['', {}, '127.0.0.1', 'example.com.', '%20', '=']
	for (let i = 0; i < invalids.length; i++) {
		const setCookieDomain = invalids[i]
		const options = new Options(null, null, { setCookieDomain })
		if (options.cookie.domain !== base.cookie.domain) {
			throw new AssertionError(`'${setCookieDomain}' is not a domain name for the setCookieDomain option`)
		}
	}

	// Test invalid SameSite values.
	invalids = ['', 8, [], true, 'no', ' Lax', 'other']
	for (let i = 0; i < invalids.length; i++) {
		const sameSiteCookie = invalids[i]
		const options = new Options(null, null, { sameSiteCookie })
		const cookie = options.cookie
		if (cookie.sameSite !== base.cookie.sameSite) {
			throw new AssertionError(`'${sameSiteCookie}' is not a SameSite value, but no error has been returned`)
		}
	}
})

Deno.test('isStrategy', () => {
	assert(isStrategy('ABC'))
	assert(isStrategy('AB-C'))
	assert(isStrategy('A-B-C'))
	assert(isStrategy('AC-B'))
	assert(!isStrategy('A-BC'))
	assert(!isStrategy('ABCxy'))
	assert(!isStrategy('xyAC-B'))
	assert(!isStrategy('AB'))
	assert(!isStrategy('ABC '))
	assert(!isStrategy(''))
	assert(!isStrategy(5))
	assert(!isStrategy(null))
})
