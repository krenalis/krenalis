class Options {
	sessions;

	constructor(options) {
		this.sessions = {
			autoTrack: true,
			timeout: 30 * 60000, // 30 minutes.
		};
		if (options != null && isPlainObject(options.sessions)) {
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

// isPlainObject reports whether obj is a plain object.
function isPlainObject(obj) {
	return typeof obj === 'object' && !Array.isArray(obj) && obj != null;
}

export default Options;
