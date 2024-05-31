import { assert, assertEquals } from '@std/assert'
import { FakeTime } from '@std/testing/time'
import * as fake from './test_fake.js'
import Options from './options.js'
import Storage, { base64Store, cookieStore, memoryStore, multiStore, noStore, webStore } from './storage.js'

const writeKey = 'rq6JJg5ENWK28NHfxSwJZmzeIvDC8GQO'
const oneYear = 365 * 24 * 60 * 60 * 1000

Deno.test('Storage', () => {
	localStorage.clear()

	const options = new Options()
	options.stores = ['localStorage']
	const storage = new Storage(writeKey, options)

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
	let storage = new cookieStore({ domain: null, maxAge: oneYear / 2, path: '/', sameSite: 'lax', secure: false })

	assertEquals(storage.get(''), null)
	storage.set('', '')
	assertEquals(storage.get(''), '')
	assertEquals(storage.get('boo'), null)
	storage.set('boo', 'foo')

	let cookie = globalThis.document.getCookie('boo', 'a.example.com')
	assertEquals(cookie.domain, 'a.example.com')
	assertEquals(cookie.expires, expires(oneYear / 2))
	assertEquals(cookie.path, '/')
	assertEquals(cookie.sameSite, 'lax')
	assert(!cookie.secure)

	assertEquals(storage.get('boo'), 'foo')
	storage.set('boo', '%ab')
	assertEquals(storage.get('boo'), '%ab')
	storage.set('boo', ' ;')
	assertEquals(storage.get('boo'), ' ;')
	storage.set('boo', '=')
	assertEquals(storage.get('boo'), '=')
	storage.set('a', '1')
	storage.set('b', '2')
	storage.set('ab', '3')
	assertEquals(storage.get('a'), '1')
	assertEquals(storage.get('b'), '2')
	assertEquals(storage.get('ab'), '3')
	storage.delete('c')
	storage.delete('b')
	assertEquals(storage.get('a'), '1')
	assertEquals(storage.get('b'), null)
	assertEquals(storage.get('ab'), '3')

	globalThis.document = new fake.CookieDocument(globalThis.location, 'a.example.com')
	storage = new cookieStore({ domain: '', maxAge: oneYear, path: '/store/', sameSite: 'lax', secure: true })
	storage.set('boo', 'foo')
	cookie = document.getCookie('boo', undefined)
	assertEquals(cookie.domain, undefined)
	assertEquals(cookie.expires, expires(oneYear))
	assertEquals(cookie.path, '/store/')
	assertEquals(cookie.sameSite, 'lax')
	assert(cookie.secure)

	globalThis.document = new fake.CookieDocument(globalThis.location, 'a.example.com')
	storage = new cookieStore({ domain: 'b.a.example.com', maxAge: oneYear, path: '/', sameSite: 'lax', secure: true })
	storage.set('boo', 'foo')
	cookie = globalThis.document.getCookie('boo', 'b.a.example.com')
	assertEquals(cookie.domain, 'b.a.example.com')
	assertEquals(cookie.expires, expires(oneYear))
	assertEquals(cookie.path, '/')
	assertEquals(cookie.sameSite, 'lax')
	assert(cookie.secure)

	globalThis.document = new fake.CookieDocument(globalThis.location, 'a.example.com')
	storage = new cookieStore({ domain: null, maxAge: oneYear * 2, path: '/', sameSite: 'strict', secure: true })
	storage.set('boo', 'foo')
	cookie = document.getCookie('boo', 'a.example.com')
	assertEquals(cookie.domain, 'a.example.com')
	assertEquals(cookie.expires, expires(oneYear * 2))
	assertEquals(cookie.path, '/')
	assertEquals(cookie.sameSite, 'strict')
	assert(cookie.secure)

	globalThis.location = new URL('https://172.16.254.1/')
	globalThis.document = new fake.CookieDocument(globalThis.location, '172.16.254.1')
	storage = new cookieStore({ domain: null, maxAge: oneYear, path: '/', sameSite: 'none', secure: true })
	assertEquals(storage.get('boo'), null)
	storage.set('boo', 'foo')
	assertEquals(storage.get('boo'), 'foo')

	cookie = document.getCookie('boo', '172.16.254.1')
	assertEquals(cookie.domain, '172.16.254.1')
	assertEquals(cookie.expires, expires(oneYear))
	assertEquals(cookie.path, '/')
	assertEquals(cookie.sameSite, 'none')
	assert(cookie.secure)

	globalThis.location = new URL('https://c.b.a.example.com./account/')
	globalThis.document = new fake.CookieDocument(globalThis.location, 'example.com.')
	storage = new cookieStore({ domain: null, maxAge: oneYear, path: '/', sameSite: 'strict', secure: false })
	storage.set('boo', 'foo')
	cookie = document.getCookie('boo', 'example.com.')
	assertEquals(cookie.domain, 'example.com.')

	storage.delete('boo')
	assertEquals(storage.get('boo'), null)

	time.restore()
})

Deno.test('base64Store', () => {
	const memory = new memoryStore()
	const storage = new base64Store(memory)
	assertEquals(storage.get('k'), null)
	storage.set('k', "Let's watch a 🎥. 🍔 & 🍟 for dinner!")
	assertEquals(
		memory.get('k'),
		'TGV0J3Mgd2F0Y2ggYSDwn46lLiDwn42UICYg8J+NnyBmb3IgZGlubmVyIQ',
	)
	assertEquals(storage.get('k'), "Let's watch a 🎥. 🍔 & 🍟 for dinner!")
	storage.delete('k')
	assertEquals(memory.get('k'), null)
	assertEquals(storage.get('k'), null)
})

Deno.test('memoryStore', () => {
	const storage = new memoryStore()
	assertEquals(storage.get('key'), null)
	storage.set('key', 'value')
	assertEquals(storage.get('key'), 'value')
	storage.delete('key')
	assertEquals(storage.get('key'), null)
})

Deno.test('multiStore', () => {
	localStorage.clear()

	globalThis.location = new URL('https://c.b.a.example.com/account/')
	globalThis.document = new fake.CookieDocument(globalThis.location, 'a.example.com')
	const cs = new cookieStore({ domain: null, maxAge: oneYear, path: '/', sameSite: 'lax', secure: false })
	const lss = new webStore(localStorage)
	const storage = new multiStore([cs, lss])

	assertEquals(storage.get('key'), null)
	assertEquals(cs.get('key'), null)
	assertEquals(lss.get('key'), null)

	storage.set('key', 'value')
	assertEquals(storage.get('key'), 'value')
	assertEquals(cs.get('key'), 'value')
	assertEquals(lss.get('key'), 'value')

	storage.delete('key')
	assertEquals(storage.get('key'), null)
	assertEquals(cs.get('key'), null)
	assertEquals(lss.get('key'), null)
})

Deno.test('noStore', () => {
	const storage = new noStore()
	assertEquals(storage.get('key'), null)
	storage.set('key', 'value')
	assertEquals(storage.get('key'), null)
	storage.delete('key')
	assertEquals(storage.get('key'), null)
})

Deno.test('webStore', () => {
	let storage = new webStore(localStorage)
	assertEquals(storage.get('k'), null)
	storage.set('k', 'v')
	assertEquals(storage.get('k'), 'v')
	storage.delete('k')
	assertEquals(storage.get('k'), null)

	// Exceptions are handled and not propagated.
	storage = new webStore(new fake.Storage())
	assertEquals(storage.get('k'), null)
	storage.set('k', 'v')
	assertEquals(storage.get('k'), null)
	storage.delete('k')
})
