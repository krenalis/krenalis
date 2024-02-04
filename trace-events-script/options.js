import { isPlainObject } from './utils.js';

class Options {
	sessions = {
		autoTrack: true,
		timeout: 30 * 60000, // 30 minutes.
	};
	strategy = 'AB-C';

	constructor(options) {
		if (options == null) {
			return;
		}
		if (options.strategy != null) {
			if (!isStrategy(options.strategy)) {
				throw new Error('strategy option is not valid');
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

// isStrategy reports whether s is a strategy.
function isStrategy(s) {
	return typeof s === 'string' && /^ABC|AB-C|A-B-C|AC-B$/.test(s);
}

export default Options;
