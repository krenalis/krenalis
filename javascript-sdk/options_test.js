import { assert, assertEquals, AssertionError } from 'https://deno.land/std@0.218.0/assert/mod.ts'
import { isStrategy, Options } from './options.js'

const oneYear = 365 * 24 * 60 * 60 * 1000

Deno.test('Options', () => {
	localStorage.clear()

	const base = {
		autoTrack: true,
		debug: false,
		sameDomainCookiesOnly: false,
		sameSiteCookie: 'lax',
		secureCookie: false,
		setCookieDomain: null,
		storage: {
			cookie: {
				domain: null,
				maxAge: oneYear,
				path: '/',
				sameSite: 'lax',
				secure: false,
			},
			type: 'multiStorage',
		},
		timeout: 30 * 60000,
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
			options: { storage: { cookie: { domain: null, maxage: oneYear, path: '/', samesite: 'Lax', secure: false } } },
			...base,
		},
		{
			options: { storage: { cookie: { domain: '', samesite: 'Strict', secure: true } } },
			...base,
			storage: { ...base.storage, cookie: { ...base.storage.cookie, domain: '', sameSite: 'strict', secure: true } },
		},
		{
			options: { storage: { cookie: { domain: 'example.com', maxage: 10000000, secure: {} } } },
			...base,
			storage: {
				...base.storage,
				cookie: { ...base.storage.cookie, domain: 'example.com', maxAge: 10000000, secure: true },
			},
		},
		{
			options: { storage: { cookie: { path: '/store/' } } },
			...base,
			storage: { ...base.storage, cookie: { ...base.storage.cookie, path: '/store/' } },
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
			storage: { ...base.storage, cookie: { ...base.storage.cookie, domain: '' } },
		},
		{ options: { sameDomainCookiesOnly: false }, ...base },
		{
			options: { secureCookie: true },
			...base,
			storage: { ...base.storage, cookie: { ...base.storage.cookie, secure: true } },
		},
		{ options: { secureCookie: false }, ...base },
		{ options: { sameSiteCookie: 'Lax' }, ...base },
		{
			options: { sameSiteCookie: 'Strict' },
			...base,
			storage: { ...base.storage, cookie: { ...base.storage.cookie, sameSite: 'strict' } },
		},
		{
			options: { sameSiteCookie: 'None' },
			...base,
			storage: { ...base.storage, cookie: { ...base.storage.cookie, sameSite: 'none' } },
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
			storage: { ...base.storage, cookie: { ...base.storage.cookie, domain: 'example.com' } },
		},
		{ options: { debug: false }, ...base },
		{ options: { debug: true }, ...base, debug: true },
		{ options: { debug: 0 }, ...base },
		{ options: { debug: 1 }, ...base, debug: true },
	]

	for (let i = 0; i < tests.length; i++) {
		const test = tests[i]
		const options = new Options(null, null, test.options)
		assertEquals(test.debug, test.debug)
		assertEquals(options.storage.cookie.domain, test.storage.cookie.domain)
		assertEquals(options.storage.cookie.maxAge, test.storage.cookie.maxAge)
		assertEquals(options.storage.cookie.path, test.storage.cookie.path)
		assertEquals(options.storage.cookie.sameSite, test.storage.cookie.sameSite)
		assertEquals(options.storage.cookie.secure, test.storage.cookie.secure)
		assertEquals(options.storage.type, test.storage.type)
		assertEquals(options.sessions.autoTrack, test.autoTrack)
		assertEquals(options.sessions.timeout, test.timeout)
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
		if (options.storage.cookie.domain !== base.storage.cookie.domain) {
			throw new AssertionError(`'${setCookieDomain}' is not a domain name for the setCookieDomain option`)
		}
	}

	// Test invalid SameSite values.
	invalids = ['', 8, [], true, 'no', ' Lax', 'other']
	for (let i = 0; i < invalids.length; i++) {
		const sameSiteCookie = invalids[i]
		const options = new Options(null, null, { sameSiteCookie })
		const cookie = options.storage.cookie
		if (cookie.sameSite !== base.storage.cookie.sameSite) {
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
