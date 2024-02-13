import { assert, assertEquals } from 'https://deno.land/std@0.212.0/assert/mod.ts'
import { FakeTime } from 'https://deno.land/std@0.212.0/testing/time.ts'
import * as fake from './test_fake.js'
import Storage, { cookieStore, multipleStore, storageStore } from './storage.js'

const oneYear = 365 * 24 * 60 * 60 * 1000

Deno.test('Storage', () => {
	localStorage.clear()

	const storage = new Storage()

	function expectAnonymousId(id) {
		assertEquals(storage.anonymousId(), id)
	}

	function expectGroupId(id) {
		assertEquals(storage.groupId(), id)
	}

	function expectSession(id, expiration, start) {
		const [actualId, actualExpiration, actualStart] = storage.session()
		assertEquals(actualId, id)
		assertEquals(actualExpiration, expiration)
		assertEquals(actualStart, start)
	}

	function expectTraits(kind, traits) {
		assertEquals(storage.traits(kind), traits)
	}

	function expectUserId(id) {
		assertEquals(storage.userId(), id)
	}

	function expectEmptySuspended() {
		expectSession(null, 0, false)
		expectAnonymousId(null)
		expectTraits('user', {})
		expectGroupId(null)
		expectTraits('group', {})
	}

	expectAnonymousId(null)
	expectGroupId(null)
	expectSession(null, 0, false)
	expectTraits('user', {})
	expectTraits('group', {})
	expectUserId(null)

	storage.setAnonymousId('703a1h3b830')
	expectAnonymousId('703a1h3b830')

	storage.setGroupId('72047285')
	expectGroupId('72047285')
	storage.setGroupId()
	expectGroupId(null)

	storage.setSession()
	expectSession(null, 0, false)

	storage.setSession(1706175160340, 1706176628710, false)
	expectSession(1706175160340, 1706176628710, false)

	storage.setSession(1706178514540, 1706178239698, true)
	expectSession(1706178514540, 1706178239698, true)

	storage.setTraits('user', { name: 'John' })
	expectTraits('user', { name: 'John' })
	storage.setTraits('user', { name: 0n })
	expectTraits('user', { name: 'John' })
	storage.setTraits('user', {})
	expectTraits('user', {})
	storage.setTraits('user', { name: 'John' })
	storage.setTraits('user')
	expectTraits('user', {})

	storage.setTraits('group', { name: 'Acme' })
	expectTraits('group', { name: 'Acme' })
	storage.setTraits('group', { name: 0n })
	expectTraits('group', { name: 'Acme' })
	storage.setTraits('group', {})
	expectTraits('group', {})
	storage.setTraits('group', { name: 'Acme' })
	storage.setTraits('group')
	expectTraits('group', {})

	storage.setUserId('86103517')
	expectUserId('86103517')
	storage.setUserId()
	expectUserId(null)

	storage.setSession()
	expectSession(null, 0, false)

	// Test suspend and restore.

	localStorage.clear()

	storage.suspend()
	expectEmptySuspended()
	storage.restore()
	expectEmptySuspended()

	localStorage.clear()

	storage.restore()
	expectEmptySuspended()

	localStorage.clear()

	storage.setSession(1706175160340, 1706176628710, false)
	storage.setAnonymousId('703a1h3b830')
	storage.setTraits('user', { name: 'John' })
	storage.setGroupId('acme')
	storage.setTraits('group', { name: 'Acme' })
	storage.suspend()

	expectSession(1706175160340, 1706176628710, false)
	expectAnonymousId('703a1h3b830')
	expectTraits('user', { name: 'John' })
	expectGroupId('acme')
	expectTraits('group', { name: 'Acme' })

	storage.setSession(1706178514540, 1706178239698, true)
	storage.setAnonymousId('t67w1mvz4t2i')
	storage.setTraits('user', { name: 'Susan' })
	storage.setGroupId('inc')
	storage.setTraits('group', { name: 'Inc' })

	storage.restore()
	expectSession(1706175160340, 1706176628710, false)
	expectAnonymousId('703a1h3b830')
	expectTraits('user', { name: 'John' })
	expectGroupId('acme')
	expectTraits('group', { name: 'Acme' })

	// Test removeSuspended.

	localStorage.clear()

	storage.setSession(1706175160340, 1706176628710, false)
	storage.setAnonymousId('703a1h3b830')
	storage.setTraits('user', { name: 'John' })
	storage.setGroupId('acme')
	storage.setTraits('group', { name: 'Acme' })
	storage.suspend()

	storage.setSession(1706178514540, 1706178239698, true)
	storage.setAnonymousId('t67w1mvz4t2i')
	storage.setTraits('user', { name: 'Susan' })
	storage.setGroupId('inc')
	storage.setTraits('group', { name: 'Inc' })

	storage.removeSuspended()
	storage.restore()
	expectEmptySuspended()
})

Deno.test('cookieStore', () => {
	const time = new FakeTime()

	function expires(maxAge) {
		const expires = new Date(Date.now() + maxAge)
		expires.setMilliseconds(0)
		return expires
	}

	globalThis.location = new URL('https://c.b.a.example.com/account/')

	globalThis.document = new fake.CookieDocument(globalThis.location, 'a.example.com')
	let store = new cookieStore({ domain: null, maxAge: oneYear / 2, path: '/', sameSite: 'lax', secure: false })

	assertEquals(store.get(''), null)
	store.set('', '')
	assertEquals(store.get(''), '')
	assertEquals(store.get('boo'), null)
	store.set('boo', 'foo')

	let cookie = globalThis.document.getCookie('boo', 'a.example.com')
	assertEquals(cookie.domain, 'a.example.com')
	assertEquals(cookie.expires, expires(oneYear / 2))
	assertEquals(cookie.path, '/')
	assertEquals(cookie.sameSite, 'lax')
	assert(!cookie.secure)

	assertEquals(store.get('boo'), 'foo')
	store.set('boo', '%ab')
	assertEquals(store.get('boo'), '%ab')
	store.set('boo', ' ;')
	assertEquals(store.get('boo'), ' ;')
	store.set('boo', '=')
	assertEquals(store.get('boo'), '=')
	store.set('a', '1')
	store.set('b', '2')
	store.set('ab', '3')
	assertEquals(store.get('a'), '1')
	assertEquals(store.get('b'), '2')
	assertEquals(store.get('ab'), '3')
	store.delete('c')
	store.delete('b')
	assertEquals(store.get('a'), '1')
	assertEquals(store.get('b'), null)
	assertEquals(store.get('ab'), '3')

	globalThis.document = new fake.CookieDocument(globalThis.location, 'a.example.com')
	store = new cookieStore({ domain: '', maxAge: oneYear, path: '/store/', sameSite: 'lax', secure: true })
	store.set('boo', 'foo')
	cookie = globalThis.document.getCookie('boo', undefined)
	assertEquals(cookie.domain, undefined)
	assertEquals(cookie.expires, expires(oneYear))
	assertEquals(cookie.path, '/store/')
	assertEquals(cookie.sameSite, 'lax')
	assert(cookie.secure)

	globalThis.document = new fake.CookieDocument(globalThis.location, 'a.example.com')
	store = new cookieStore({ domain: 'b.a.example.com', maxAge: oneYear, path: '/', sameSite: 'lax', secure: true })
	store.set('boo', 'foo')
	cookie = globalThis.document.getCookie('boo', 'b.a.example.com')
	assertEquals(cookie.domain, 'b.a.example.com')
	assertEquals(cookie.expires, expires(oneYear))
	assertEquals(cookie.path, '/')
	assertEquals(cookie.sameSite, 'lax')
	assert(cookie.secure)

	globalThis.document = new fake.CookieDocument(globalThis.location, 'a.example.com')
	store = new cookieStore({ domain: null, maxAge: oneYear * 2, path: '/', sameSite: 'strict', secure: true })
	store.set('boo', 'foo')
	cookie = globalThis.document.getCookie('boo', 'a.example.com')
	assertEquals(cookie.domain, 'a.example.com')
	assertEquals(cookie.expires, expires(oneYear * 2))
	assertEquals(cookie.path, '/')
	assertEquals(cookie.sameSite, 'strict')
	assert(cookie.secure)

	globalThis.location = new URL('https://172.16.254.1/')
	globalThis.document = new fake.CookieDocument(globalThis.location, '172.16.254.1')
	store = new cookieStore({ domain: null, maxAge: oneYear, path: '/', sameSite: 'none', secure: true })
	assertEquals(store.get('boo'), null)
	store.set('boo', 'foo')
	assertEquals(store.get('boo'), 'foo')

	cookie = globalThis.document.getCookie('boo', '172.16.254.1')
	assertEquals(cookie.domain, '172.16.254.1')
	assertEquals(cookie.expires, expires(oneYear))
	assertEquals(cookie.path, '/')
	assertEquals(cookie.sameSite, 'none')
	assert(cookie.secure)

	globalThis.location = new URL('https://c.b.a.example.com./account/')
	globalThis.document = new fake.CookieDocument(globalThis.location, 'example.com.')
	store = new cookieStore({ domain: null, maxAge: oneYear, path: '/', sameSite: 'strict', secure: false })
	store.set('boo', 'foo')
	cookie = globalThis.document.getCookie('boo', 'example.com.')
	assertEquals(cookie.domain, 'example.com.')

	store.delete('boo')
	assertEquals(store.get('boo'), null)

	time.restore()
})

Deno.test('storageStore', () => {
	let store = new storageStore(globalThis.localStorage)
	assertEquals(store.get('k'), null)
	store.set('k', 'v')
	assertEquals(store.get('k'), 'v')
	store.delete('k')
	assertEquals(store.get('k'), null)

	// Exceptions are handled and not propagated.
	store = new storageStore(new fake.Storage())
	assertEquals(store.get('k'), null)
	store.set('k', 'v')
	assertEquals(store.get('k'), null)
	store.delete('k')
})

Deno.test('multipleStore', () => {
	localStorage.clear()

	globalThis.location = new URL('https://c.b.a.example.com/account/')
	globalThis.document = new fake.CookieDocument(globalThis.location, 'a.example.com')
	const cs = new cookieStore({ domain: null, maxAge: oneYear, path: '/', sameSite: 'lax', secure: false })
	const lss = new storageStore(globalThis.localStorage)
	const store = new multipleStore([cs, lss])

	assertEquals(store.get('key'), null)
	assertEquals(cs.get('key'), null)
	assertEquals(lss.get('key'), null)

	store.set('key', 'value')
	assertEquals(store.get('key'), 'value')
	assertEquals(cs.get('key'), 'value')
	assertEquals(lss.get('key'), 'value')

	store.delete('key')
	assertEquals(store.get('key'), null)
	assertEquals(cs.get('key'), null)
	assertEquals(lss.get('key'), null)
})
