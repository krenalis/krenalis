import { debug, isPlainObject, log } from './utils.js'

const matchAll = /\s\S/
const strategies = ['Fusion', 'Conversion', 'Isolation', 'Preservation']
const storages = ['multiStorage', 'cookieStorage', 'localStorage', 'sessionStorage', 'memoryStorage', 'none']
const stores = ['cookie', 'localStorage', 'sessionStorage', 'memory']

class Options {
	cookie = {
		domain: null,
		maxAge: 365 * 24 * 60 * 60 * 1000, // one year
		path: '/',
		sameSite: 'lax',
		secure: false,
	}
	debug = false
	group = {
		storage: {
			stores: null,
		},
	}
	sessions = {
		autoTrack: true,
		timeout: 30 * 60000, // 30 minutes.
	}
	stores = ['localStorage', 'cookie', 'memory']
	strategy = 'Conversion'
	useQueryString = {
		aid: matchAll,
		uid: matchAll,
	}
	user = {
		storage: {
			stores: null,
		},
	}

	constructor(writeKey, endpoint, options, ready) {
		const setCookie = (cookie, rs) => {
			if (isPlainObject(cookie)) {
				if (cookie.domain === '' || isDomainName(cookie.domain)) {
					this.cookie.domain = cookie.domain
				}
				const maxAge = asPositiveFiniteNumber(cookie.maxage)
				if (maxAge != null) {
					this.cookie.maxAge = maxAge * (rs ? 1 : 24 * 60 * 60 * 1000)
				}
				if (canBeUsedAsCookiePath(cookie.path)) {
					this.cookie.path = cookie.path
				}
				const sameSite = rs ? cookie.samesite : cookie.sameSite
				if (isSameSite(sameSite)) {
					this.cookie.sameSite = sameSite.toLowerCase()
				}
				if ('secure' in cookie) {
					this.cookie.secure = !!cookie.secure
				}
			}
		}
		if (typeof options === 'object' && options != null) {
			if (options.debug != null) {
				this.debug = !!options.debug
			}
			if ('cookie' in options) {
				// 'options.cookie' overwrites 'storage.cookie', 'sameDomainCookiesOnly', 'sameSiteCookie', 'secureCookie', and 'setCookieDomain' options.
				setCookie(options.cookie, false)
			} else {
				if (options.sameDomainCookiesOnly) {
					this.cookie.domain = ''
				}
				if (isSameSite(options.sameSiteCookie)) {
					this.cookie.sameSite = options.sameSiteCookie.toLowerCase()
				}
				if (options.secureCookie) {
					this.cookie.secure = !!options.secureCookie
				}
				if (isDomainName(options.setCookieDomain)) {
					this.cookie.domain = options.setCookieDomain
				}
				if (isPlainObject(options.storage)) {
					// 'storage.cookie' overwrites 'sameDomainCookiesOnly', 'sameSiteCookie', 'secureCookie', and 'setCookieDomain' options.
					setCookie(options.storage.cookie, true)
				}
			}
			if (isPlainObject(options.sessions)) {
				const s = options.sessions
				if (s.autoTrack === false) {
					this.sessions.autoTrack = false
				}
				const timeout = Number(s.timeout)
				if (!isNaN(timeout)) {
					if (s.timeout > 0) {
						this.sessions.timeout = timeout
					} else {
						this.sessions.autoTrack = false
					}
				}
			}
			if (isPlainObject(options.storage)) {
				const stores = parseStores(options.storage.stores)
				if (stores != null) {
					// 'options.storage.stores' overwrites 'options.storage.type'.
					this.stores = stores
				} else if (isStorage(options.storage.type)) {
					let stores
					switch (options.storage.type) {
						case 'cookieStorage':
							stores = ['cookie', 'localStorage', 'sessionStorage', 'memory']
							break
						case 'localStorage':
							stores = ['localStorage', 'memory']
							break
						case 'sessionStorage':
							stores = ['sessionStorage', 'memory']
							break
						case 'memoryStorage':
							stores = ['memory']
							break
						case 'none':
							stores = []
					}
					this.stores = stores
				}
			}
			if (isPlainObject(options.user) && isPlainObject(options.user.storage)) {
				// 'options.storage.user.stores' overwrites 'options.storage.stores' and 'options.storage.type'.
				const stores = parseStores(options.user.storage.stores)
				if (stores != null) {
					this.user.storage.stores = stores
				}
			}
			if (isPlainObject(options.group) && isPlainObject(options.group.storage)) {
				// 'options.storage.group.stores' overwrites 'options.storage.stores' and 'options.storage.type'.
				const stores = parseStores(options.group.storage.stores)
				if (stores != null) {
					this.group.storage.stores = stores
				}
			}
			const qs = options.useQueryString
			if (qs === false) {
				this.useQueryString = null
			} else if (typeof qs === 'object' && qs != null) {
				if (qs.aid instanceof RegExp) {
					this.useQueryString.aid = qs.aid
				}
				if (qs.uid instanceof RegExp) {
					this.useQueryString.uid = qs.uid
				}
			}
		}
		// Asynchronously load settings from the endpoint.
		if (writeKey != null) {
			this.#load(writeKey, endpoint, true, ready)
		}
	}

	// load loads the settings from the provided URL, and then calls the
	// callback when the settings have been loaded. If an error occurs and
	// canRetry is true, it retries one more time after a random delay between
	// 10 and 100 milliseconds.
	#load(writeKey, endpoint, canRetry, callback) {
		const retry = (error, status) => {
			if (canRetry) {
				const delay = Math.floor(Math.random() * 91) + 10
				setTimeout(() => this.#load(writeKey, endpoint, false, callback), delay)
			} else if (error != null) {
				log(`An error occurred while loading the endpoint URL ${endpoint}: ${error.message}`)
			} else {
				log(`The request to ${endpoint} encountered an unexpected HTTP status code: ${status}.`)
			}
		}
		const receive = (error, status, body) => {
			debug(this.debug)?.('received endpoint response: error =', error, ', status =', status + ', body =', body)
			if (error != null || (status !== 200 && status !== 404)) {
				retry(error, status)
				return
			}
			if (status === 404) {
				const msg = body === 'error: invalid write key'
					? `The specified write key '${writeKey}' is invalid.`
					: `The specified endpoint URL '${endpoint}' does not exist.`
				log(msg)
				return
			}
			try {
				const s = JSON.parse(body)
				if (typeof s !== 'object' || s == null || !isStrategy(s.strategy)) {
					throw null
				}
				this.strategy = s.strategy
			} catch {
				log(`The response body from the endpoint URL '${endpoint}' is invalid.`)
				return
			}
			if (callback != null) {
				callback()
			}
		}
		const url = `${endpoint}projects/${encodeURIComponent(writeKey)}/settings`
		if (globalThis.fetch && typeof globalThis.fetch === 'function') {
			fetch(url)
				.then((response) => {
					// Read the body if the status is 200 or 400.
					if (response.status === 200 || response.status === 404) {
						response.text()
							.then((body) => receive(null, response.status, body))
							.catch((error) => receive(error))
					} else {
						receive(null, response.status)
					}
				})
				.catch((error) => receive(error))
		} else {
			const xhr = new XMLHttpRequest()
			xhr.open('GET', url, true)
			xhr.onerror = (error) => receive(error)
			xhr.onreadystatechange = () => {
				if (xhr.readyState === 4) {
					receive(null, xhr.status, xhr.responseText)
				}
			}
			xhr.send()
		}
	}
}

// asPositiveFiniteNumber returns n if it is a positive finite Number, otherwise
// return undefined. If n is a String, it converts it to a Number.
function asPositiveFiniteNumber(n) {
	if (typeof n === 'string') {
		n = Number(n)
	}
	if (typeof n === 'number' && isFinite(n) && n > 0) {
		return n
	}
}

// canBeUsedAsCookiePath reports whether s can be used as a cookie path.
function canBeUsedAsCookiePath(s) {
	return s === '/' || (typeof s === 'string' && /^[ -~]+$/.test(s) && s.indexOf(';') === -1)
}

// isDomainName reports whether s is a domain name.
function isDomainName(s) {
	return typeof s === 'string' && /^[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/.test(s) && s === encodeURIComponent(s)
}

// isSameSite reports whether s is a SameSite value.
function isSameSite(s) {
	return typeof s === 'string' && ['Lax', 'Strict', 'None'].indexOf(s) >= 0
}

// isStore reports whether s is a store.
function isStore(s) {
	for (let i = 0; i < stores.length; i++) {
		if (s === stores[i]) {
			return true
		}
	}
	return false
}

// parseStores parses stores as an array of stores and returns a copy of the
// array with unique values. If stores it is not valid, it returns null.
function parseStores(stores) {
	if (!Array.isArray(stores)) {
		return null
	}
	const s = []
	for (let i = 0; i < stores.length; i++) {
		if (!isStore(stores[i])) {
			return false
		}
		if (s.indexOf(stores[i]) === -1) {
			s.push(stores[i])
		}
	}
	return s
}

// isStorage reports whether s is a storage.
function isStorage(s) {
	return typeof s === 'string' && storages.indexOf(s) >= 0
}

// isStrategy reports whether s is a strategy.
function isStrategy(s) {
	return typeof s === 'string' && strategies.indexOf(s) >= 0
}

export default Options
export { isStrategy, Options }
