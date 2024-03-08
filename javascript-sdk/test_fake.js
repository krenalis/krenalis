import { assert, assertEquals, AssertionError } from 'std/assert/mod.ts'
import { DOMParser } from 'https://deno.land/x/deno_dom/deno-dom-wasm.ts'
import * as uuid from 'std/uuid/v4.ts'
import { MaxBodySize } from './sender.js'
import * as utils from './utils.js'

// Cookie represents a cookie stored by CookieDocument.
class Cookie {
	name
	value
	path
	expires
	sameSite
	secure
	domain
}

// CookieDocument implements a fake document with a 'document.cookie' property
// that accept cookie from a domain and its subdomains.
class CookieDocument {
	#location
	#domain
	#cookies = []

	// constructor returns a new CookieDocument with the provided location and
	// the domain to use for cookies.
	constructor(location, domain) {
		if (!(location instanceof URL)) {
			throw new Error('location is not an instance of URL')
		}
		this.#location = location
		this.#domain = domain
	}

	get cookie() {
		return this.#cookies.map((cookie) => `${cookie.name}=${cookie.value}`).join('; ')
	}

	set cookie(s) {
		const cookie = CookieDocument.#parse(s)
		if (cookie.domain != null && !cookie.domain.endsWith(this.#domain)) {
			return
		}
		if (cookie.path == null) {
			cookie.path = this.#location.path
		}
		for (let i = 0; i < this.#cookies.length; i++) {
			const c = this.#cookies[i]
			if (c.name === cookie.name && c.domain === cookie.domain) {
				if (cookie.expires != null && cookie.expires < new Date()) {
					this.#cookies.splice(i, 1)
				} else {
					c.value = cookie.value
					c.path = cookie.path
					c.expires = cookie.expires
				}
				return
			}
		}
		this.#cookies.push(cookie)
	}

	// getCookie returns the cookie with the provides key and domain as a Cookie
	// value. If such cookie does non exist, it returns undefined.
	getCookie(name, domain) {
		for (let i = 0; i < this.#cookies.length; i++) {
			const c = this.#cookies[i]
			if (c.name === name && c.domain === domain) {
				return Object.assign(Object.create(Object.getPrototypeOf(c)), c)
			}
		}
	}

	static #parse(s) {
		const cookie = new Cookie()
		const parts = s.split(/\s*;\s*/)
		for (let i = 0; i < parts.length; i++) {
			const pair = parts[i].split(/\s*=\s*/)
			if (i === 0) {
				cookie.name = pair[0]
				cookie.value = pair[1]
				continue
			}
			switch (pair[0]) {
				case 'path':
					cookie.path = pair[1]
					break
				case 'domain':
					cookie.domain = pair[1]
					if (cookie.domain.length > 0 && cookie.domain[0] === '.') {
						cookie.domain = cookie.domain.slice(1)
					}
					break
				case 'expires':
					cookie.expires = new Date(pair[1])
					break
				case 'samesite':
					cookie.sameSite = pair[1]
					break
				case 'secure':
					cookie.secure = true
					break
				default:
					throw new Error(`Unknown cookie attribute '${pair[0]}'`)
			}
		}
		return cookie
	}
}

// Fetch implements a fake fetch.
class Fetch {
	#installTime
	#writeKey
	#endpoint
	#keepalive
	#events = []
	#wait = null
	#error
	#fetch
	#originalFetch
	#debug

	constructor(writeKey, endpoint, keepalive, debug) {
		this.#writeKey = writeKey
		this.#endpoint = endpoint
		this.#keepalive = keepalive
		this.#fetch = async (resource, options) => {
			let events
			try {
				assertEquals(resource, endpoint)
				events = await parseRequest(this.#writeKey, this.#installTime, this.#keepalive, options)
			} catch (error) {
				if (this.#wait != null) {
					this.#wait.reject(error)
				} else {
					this.#error = error
				}
				throw error
			}
			this.#events.push(...events)
			const min = this.#wait?.min
			if (min != null && this.#events.length >= min) {
				const events = this.#events
				const resolve = this.#wait.resolve
				this.#events = []
				this.#wait = null
				this.#debug?.(`promise resolution is resolved: Fetch.events(${min})`)
				resolve(events)
			}
			const res = new Response('', {
				status: 200,
				statusText: 'OK',
				headers: new Headers({ 'content-type': 'text/plain' }),
			})
			return res
		}
		this.#debug = utils.debug(debug)
	}

	events(min) {
		if (this.#installTime == null) {
			return new Promise((_, reject) => {
				reject(new Error('Fake fetch is not installed'))
			})
		}
		if (this.#wait != null) {
			return new Promise((_, reject) => {
				reject(new Error('events already called'))
			})
		}
		return new Promise((resolve, reject) => {
			if (this.#error != null) {
				reject(this.#error)
				return
			}
			if (this.#events.length < min) {
				this.#wait = {
					min: min,
					resolve: resolve,
					reject: reject,
				}
				this.#debug?.(`promise resolution is pending: Fetch.events(${min})`)
			} else {
				const events = this.#events
				this.#events = []
				this.#wait = null
				resolve(events)
			}
		})
	}

	install() {
		if (this.#originalFetch != null) {
			throw new Error('Fake fetch is already installed')
		}
		this.#installTime = utils.getTime()
		this.#events = []
		this.#wait = null
		this.#originalFetch = globalThis.fetch
		assert(this.#originalFetch != null)
		globalThis.fetch = this.#fetch
	}

	restore() {
		if (this.#originalFetch == null) {
			throw new Error('Fake fetch is not installed')
		}
		globalThis.fetch = this.#originalFetch
		this.#originalFetch = null
		if (this.#events.length > 0) {
			throw new AssertionError(
				`Fake fetch has been restored; however, there are ${this.#events.length} unread events`,
			)
		}
	}
}

// SendBeacon implements a fake sendBeacon.
class SendBeacon {
	#installTime
	#writeKey
	#endpoint
	#events = []
	#wait = null
	#error
	#sendBeacon
	#debug

	constructor(writeKey, endpoint, debug) {
		this.#writeKey = writeKey
		this.#endpoint = endpoint
		this.#sendBeacon = (url, data) => {
			try {
				assertEquals(url, endpoint)
				assert(data instanceof Blob)
				assertEquals(data.type, 'text/plain')
				parseRequest(this.#writeKey, this.#installTime, false, {
					method: 'POST',
					headers: { 'Content-Type': data.type },
					body: data,
					redirect: 'error',
				}).then((events) => {
					this.#events.push(...events)
					const min = this.#wait?.min
					if (min != null && this.#events.length >= min) {
						const events = this.#events
						const resolve = this.#wait.resolve
						this.#events = []
						this.#wait = null
						this.#debug?.(`promise resolution is resolved: SendBeacon.events(${min})`)
						resolve(events)
					}
				})
			} catch (error) {
				if (this.#wait != null) {
					this.#wait.reject(error)
				} else {
					this.#error = error
				}
				throw error
			}
			return true
		}
		this.#debug = utils.debug(debug)
	}

	events(min) {
		if (this.#installTime == null) {
			return new Promise((_, reject) => {
				reject(new Error('Fake sendBeacon is not installed'))
			})
		}
		if (this.#wait != null) {
			return new Promise((_, reject) => {
				reject(new Error('events already called'))
			})
		}
		return new Promise((resolve, reject) => {
			if (this.#error != null) {
				reject(this.#error)
				return
			}
			if (this.#events.length < min) {
				this.#wait = {
					min: min,
					resolve: resolve,
				}
				this.#debug?.(`promise resolution is pending: SendBeacon.events(${min})`)
			} else {
				const events = this.#events
				this.#events = []
				this.#wait = null
				resolve(events)
			}
		})
	}

	install() {
		if (this.#installTime != null) {
			throw new Error('Fake sendBeacon is already installed')
		}
		this.#installTime = utils.getTime()
		this.#events = []
		this.#wait = null
		navigator.sendBeacon = this.#sendBeacon
	}

	restore() {
		if (this.#installTime == null) {
			throw new Error('Fake sendBeacon is not installed')
		}
		this.#installTime = null
		if (this.#events.length > 0) {
			throw new AssertionError(
				`Fake sendBeacon has been restored; however, there are ${this.#events.length} unread events`,
			)
		}
		delete navigator.sendBeacon
	}
}

// HTMLDocument is a fake HTMLDocument.
class HTMLDocument {
	#dom
	#visibilityState = 'visible'

	referrer = ''
	title = 'Hello from Chichi'

	get visibilityState() {
		return this.#visibilityState
	}

	set visibilityState(value) {
		if (value !== 'visible' && value !== 'hidden') {
			throw new Error(`invalid visibility state '${value}'`)
		}
		if (value === this.#visibilityState) {
			return
		}
		this.#visibilityState = value
		dispatchEvent(new Event('visibilitychange'))
	}

	constructor() {
		this.#dom = new DOMParser().parseFromString('<!DOCTYPE html>', 'text/html')
	}

	addEventListener() {
		addEventListener.bind(globalThis)(...arguments)
	}

	createElement() {
		return this.#dom.createElement(...arguments)
	}

	querySelector(selectors) {
		if (selectors !== 'link[rel="canonical"]') {
			throw new Error(`query selector '${selector}' is not supported by the fake HTMLDocument`)
		}
		const element = this.#dom.createElement('link')
		element.setAttribute('rel', 'canonical')
		element.setAttribute('href', '/path?query=123')
		element.href = 'https://example.com:8080/path?query=123'
		return element
	}
}

// Navigator is a fake Navigator.
class Navigator {
	language = 'en-US'
	userAgent =
		'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/96.0.4664.110 Safari/537.36'
	onLine = true
	#originalNavigator
	install() {
		if (this.#originalNavigator != null) {
			throw new Error('Fake Navigator is already installed')
		}
		this.#originalNavigator = navigator
		delete (globalThis.navigator)
		globalThis.navigator = this
	}
	restore() {
		if (this.#originalNavigator == null) {
			throw new Error('Fake Navigator is not installed')
		}
		delete (globalThis.navigator)
		globalThis.navigator = this.#originalNavigator
		this.#originalNavigator = null
	}
}

// Storage implements a fake storage that raises an exception at each method
// call.
//
// As a special case, getItem, setItem, and removeItem methods behave as
// expected if the key is '__test__'.
class Storage {
	#testValue = null
	length = 0

	key() {
		throw new Error('No storage available')
	}

	getItem(key) {
		if (key === '__test__') {
			return this.#testValue
		}
		throw new Error('No storage available')
	}

	setItem(key, value) {
		if (key === '__test__') {
			this.#testValue = String(value)
			return
		}
		throw new Error('Quota exceeded')
	}

	removeItem(key) {
		if (key === '__test__') {
			this.#testValue = null
			return
		}
		throw new Error('No storage available')
	}

	clear() {
		throw new Error('No storage available')
	}
}

// XMLHttpRequest is a fake XMLHttpRequest.
class XMLHttpRequest {
	static #installTime
	static #writeKey
	static #endpoint
	static #events
	static #wait
	static #error
	static #debug
	#method
	#url
	#headers = new Headers()
	onerror
	onreadystatechange
	readyState
	status
	statusText

	open(method, endpoint, async) {
		assert(endpoint, XMLHttpRequest.#endpoint)
		assert(async)
		this.#method = method.toUpperCase()
		this.#url = endpoint
	}

	setRequestHeader(name, value) {
		this.#headers.set(name.toLowerCase(), value)
	}

	send(body) {
		this.readyState = 4
		this.status = 200
		this.statusText = 'OK'
		try {
			parseRequest(XMLHttpRequest.#writeKey, XMLHttpRequest.#installTime, false, {
				method: this.#method,
				headers: this.#headers,
				body: body,
				redirect: 'error',
			}).then((events) => {
				XMLHttpRequest.#events.push(...events)
				const min = XMLHttpRequest.#wait?.min
				if (min != null && XMLHttpRequest.#events.length >= min) {
					const events = XMLHttpRequest.#events
					const resolve = XMLHttpRequest.#wait.resolve
					XMLHttpRequest.#events = []
					XMLHttpRequest.#wait = null
					XMLHttpRequest.#debug?.(`promise resolution is resolved: XMLHttpRequest.events(${min})`)
					resolve(events)
				}
			}).catch((error) => {
				console.error(error)
			})
		} catch (error) {
			if (XMLHttpRequest.#wait.reject != null) {
				XMLHttpRequest.#wait.reject(error)
			} else {
				XMLHttpRequest.#error = error
			}
			throw error
		}
		if (typeof this.onreadystatechange === 'function') {
			this.onreadystatechange()
		}
	}

	static events(min) {
		if (XMLHttpRequest.#installTime == null) {
			return new Promise((_, reject) => {
				reject(new Error('Fake XMLHttpRequest is not installed'))
			})
		}
		if (XMLHttpRequest.#wait != null) {
			return new Promise((_, reject) => {
				reject(new Error('events already called'))
			})
		}
		return new Promise((resolve, reject) => {
			if (XMLHttpRequest.#error != null) {
				reject(this.#error)
				return
			}
			if (XMLHttpRequest.#events.length < min) {
				XMLHttpRequest.#wait = {
					min: min,
					resolve: resolve,
				}
				XMLHttpRequest.#debug?.(`promise resolution is pending: XMLHttpRequest.events(${min})`)
			} else {
				const events = XMLHttpRequest.#events
				XMLHttpRequest.#events = []
				XMLHttpRequest.#wait = null
				resolve(events)
			}
		})
	}

	static install(writeKey, endpoint, debug) {
		if (XMLHttpRequest.#installTime != null) {
			throw new Error('Fake XMLHttpRequest is already installed')
		}
		XMLHttpRequest.#installTime = utils.getTime()
		XMLHttpRequest.#events = []
		XMLHttpRequest.#wait = null
		XMLHttpRequest.#writeKey = writeKey
		XMLHttpRequest.#endpoint = endpoint
		XMLHttpRequest.#debug = utils.debug(debug)
		globalThis.XMLHttpRequest = XMLHttpRequest
	}

	static restore() {
		if (XMLHttpRequest.#installTime == null) {
			throw new Error('Fake XMLHttpRequest is not installed')
		}
		XMLHttpRequest.#installTime = null
		delete (globalThis.XMLHttpRequest)
		if (this.#events.length > 0) {
			throw new AssertionError(
				`Fake fetch has been restored; however, there are ${this.#events.length} unread events`,
			)
		}
	}
}

// RandomUUID implements a fake crypto.randomUUID function.
class RandomUUID {
	#uuid
	#originalRandomUUID
	constructor(uuid) {
		this.#uuid = uuid
	}
	install() {
		if (this.#originalRandomUUID != null) {
			throw new Error('Fake crypto.randomUUID is already installed')
		}
		this.#originalRandomUUID = crypto.randomUUID.bind(crypto)
		crypto.randomUUID = () => this.#uuid
	}
	restore() {
		if (this.#originalRandomUUID == null) {
			throw new Error('Fake crypto.randomUUID is not installed')
		}
		crypto.randomUUID = this.#originalRandomUUID
		this.#originalRandomUUID = null
	}
}

// parseRequest parses a request to the fake fetch and XMLHttpRequest.send functions
async function parseRequest(writeKey, minTime, keepalive, options) {
	const now = utils.getTime()

	assertEquals(options.method, 'POST')
	let headers = options.headers
	if (!(options.headers instanceof Headers)) {
		headers = new Headers(options.headers)
	}
	assertEquals(Array.from(headers.keys()).length, 1)
	assertEquals(headers.get('content-type'), 'text/plain')
	assertEquals(options.redirect, 'error')
	assertEquals(Boolean(options.keepalive), keepalive)
	assert(options.body instanceof Blob)
	if (options.body.size > MaxBodySize) {
		throw new AssertionError(`batch body size (${options.body.size}) is greater than ${MaxBodySize}`)
	}

	const body = JSON.parse(await options.body.text())
	assertEquals(typeof body.batch, 'object')
	assert(body.batch instanceof Array)
	assert(body.batch.length > 0)
	assertEquals(typeof body.sentAt, 'string')
	const sentAt = new Date(body.sentAt)
	assert(minTime <= sentAt && sentAt <= now)
	assertEquals(body.writeKey, writeKey)

	const events = []
	for (let i = 0; i < body.batch.length; i++) {
		const event = body.batch[i]
		assertEquals(typeof event, 'object')
		assertEquals(typeof event.messageId, 'string')
		assert(uuid.validate(event.messageId))
		events.push(event)
	}

	return events
}

export { Cookie, CookieDocument, Fetch, HTMLDocument, Navigator, RandomUUID, SendBeacon, Storage, XMLHttpRequest }
