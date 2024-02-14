import { isPlainObject } from './utils.js'

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

	constructor(options) {
		if (options == null) {
			return
		}
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
		if (options.strategy != null) {
			if (!isStrategy(options.strategy)) {
				throw new Error(`strategy option is not valid`)
			}
			this.strategy = options.strategy
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
	return typeof s === 'string' && /^Lax|Strict|None$/.test(s)
}

// isStorage reports whether s is a storage.
function isStorage(s) {
	return typeof s === 'string' && /^multiStorage|cookieStorage|localStorage|sessionStorage|memoryStorage|none$/.test(s)
}

// isStrategy reports whether s is a strategy.
function isStrategy(s) {
	return typeof s === 'string' && /^ABC|AB-C|A-B-C|AC-B$/.test(s)
}

export default Options
