import { debug, isPlainObject, log } from './utils.js'

const strategies = ['ABC', 'AB-C', 'A-B-C', 'AC-B']
const storages = ['multiStorage', 'cookieStorage', 'localStorage', 'sessionStorage', 'memoryStorage', 'none']

class Options {
	debug = false
	sessions = {
		autoTrack: true,
		timeout: 30 * 60000, // 30 minutes.
	}
	storage = {
		cookie: {
			domain: null,
			maxAge: 365 * 24 * 60 * 60 * 1000, // one year
			path: '/',
			sameSite: 'lax',
			secure: false,
		},
		type: 'multiStorage',
	}
	strategy = 'AB-C'
	useQueryString = true

	constructor(writeKey, endpoint, options, ready) {
		if (options != null) {
			if (options.debug != null) {
				this.debug = !!options.debug
			}
			if (options.sameDomainCookiesOnly) {
				this.storage.cookie.domain = ''
			}
			if (isSameSite(options.sameSiteCookie)) {
				this.storage.cookie.sameSite = options.sameSiteCookie.toLowerCase()
			}
			if (options.secureCookie) {
				this.storage.cookie.secure = !!options.secureCookie
			}
			if (isDomainName(options.setCookieDomain)) {
				this.storage.cookie.domain = options.setCookieDomain
			}
			if (isPlainObject(options.storage)) {
				// 'storage.cookie' overwrites 'sameDomainCookiesOnly', 'sameSiteCookie', 'secureCookie', and 'setCookieDomain' options.
				const cookie = options.storage.cookie
				if (isPlainObject(cookie)) {
					if (cookie.domain === '' || isDomainName(cookie.domain)) {
						this.storage.cookie.domain = cookie.domain
					}
					const maxAge = asPositiveFiniteNumber(cookie.maxage)
					if (maxAge != null) {
						this.storage.cookie.maxAge = maxAge
					}
					if (canBeUsedAsCookiePath(cookie.path)) {
						this.storage.cookie.path = cookie.path
					}
					if (isSameSite(cookie.samesite)) {
						this.storage.cookie.sameSite = cookie.samesite.toLowerCase()
					}
					if ('secure' in cookie) {
						this.storage.cookie.secure = !!cookie.secure
					}
				}
				if (isStorage(options.storage.type)) {
					this.storage.type = options.storage.type
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
			if (options.useQueryString != null) {
				this.useQueryString = !!options.useQueryString
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
		const url = `${endpoint}connection/${encodeURIComponent(writeKey)}/settings`
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
