import { isPlainObject } from './utils.js';

class Options {
	debug = false;
	sameSiteCookie = 'lax';
	secureCookie = false;
	sessions = {
		autoTrack: true,
		timeout: 30 * 60000, // 30 minutes.
	};
	strategy = 'AB-C';

	constructor(options) {
		if (options == null) {
			return;
		}
		if (options.debug != null) {
			this.debug = !!options.debug;
		}
		if (options.sameSiteCookie != null) {
			if (!isSameSite(options.sameSiteCookie)) {
				throw new Error(`sameSiteCookie option '${options.sameSiteCookie}' is not valid`);
			}
			this.sameSiteCookie = options.sameSiteCookie.toLowerCase();
		}
		if (options.secureCookie != null) {
			this.secureCookie = !!options.secureCookie;
		}
		if (options.strategy != null) {
			if (!isStrategy(options.strategy)) {
				throw new Error(`strategy option '${options.strategy}' is not valid`);
			}
			this.strategy = options.strategy;
		}
		if (isPlainObject(options.sessions)) {
			const s = options.sessions;
			if (s.autoTrack === false) {
				this.sessions.autoTrack = false;
			}
			const timeout = Number(s.timeout);
			if (!isNaN(timeout)) {
				if (s.timeout > 0) {
					this.sessions.timeout = timeout;
				} else {
					this.sessions.autoTrack = false;
				}
			}
		}
	}
}

// isSameSite reports whether s is a SameSite value.
function isSameSite(s) {
	return typeof s === 'string' && /^lax|strict|none$/i.test(s);
}

// isStrategy reports whether s is a strategy.
function isStrategy(s) {
	return typeof s === 'string' && /^ABC|AB-C|A-B-C|AC-B$/.test(s);
}

export default Options;
