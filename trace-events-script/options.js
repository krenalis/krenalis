import { isPlainObject } from './utils.js'

class Options {
	debug = false
	sameDomainCookiesOnly = false
	sameSiteCookie = 'lax'
	secureCookie = false
	sessions = {
		autoTrack: true,
		timeout: 30 * 60000, // 30 minutes.
	}
	setCookieDomain = null
	strategy = 'AB-C'

	constructor(options) {
		if (options == null) {
			return
		}
		if (options.debug != null) {
			this.debug = !!options.debug
		}
		if (options.sameDomainCookiesOnly != null) {
			this.sameDomainCookiesOnly = !!options.sameDomainCookiesOnly
		}
		if (options.sameSiteCookie != null) {
			if (!isSameSite(options.sameSiteCookie)) {
				throw new Error(`sameSiteCookie option is not valid`)
			}
			this.sameSiteCookie = options.sameSiteCookie.toLowerCase()
		}
		if (options.secureCookie != null) {
			this.secureCookie = !!options.secureCookie
		}
		if (options.setCookieDomain != null) {
			if (!isDomainName(options.setCookieDomain)) {
				throw new Error(`setCookieDomain option is not a valid domain`)
			}
			this.setCookieDomain = options.setCookieDomain
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
		if (this.setCookieDomain != null && this.sameDomainCookiesOnly) {
			throw new Error(`setCookieDomain option must be null if the sameDomainCookiesOnly option is true`)
		}
	}
}

// isDomainName reports whether s is a domain name.
function isDomainName(s) {
	return typeof s === 'string' && /^[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/.test(s) && s === encodeURIComponent(s)
}

// isSameSite reports whether s is a SameSite value.
function isSameSite(s) {
	return typeof s === 'string' && /^lax|strict|none$/i.test(s)
}

// isStrategy reports whether s is a strategy.
function isStrategy(s) {
	return typeof s === 'string' && /^ABC|AB-C|A-B-C|AC-B$/.test(s)
}

export default Options
