class Session {
	#storage;
	#timeout = 30 * 60000; // 30 minutes.

	constructor(storage) {
		this.#storage = storage;
		this.#init();
	}

	// init initializes the current session. If no section exists, or the
	// current session is expired it starts a new session as the start method
	// does.
	#init() {
		let [id] = this.#storage.getSession();
		const timestamp = new Date().getTime();
		if (id == null || id + this.#timeout < timestamp) {
			this.#storage.setSession(timestamp, true);
		}
	}

	// end ends the current session.
	end() {
		this.#storage.setSession(null, false);
	}

	// getFresh returns the current session or, if no session exist or the
	// current session is expired, starts a new session and returns it. The
	// boolean return value reports whether a new session has been started
	// after the previous call to getFresh.
	getFresh() {
		let [id, start] = this.#storage.getSession();
		const timestamp = new Date().getTime();
		if (id == null || id + this.#timeout < timestamp) {
			id = timestamp;
			start = true;
		}
		if (start) {
			this.#storage.setSession(id, false);
		}
		return [id, start];
	}

	// get returns the current session, or null if no session exist.
	get() {
		let [id] = this.#storage.getSession();
		return id;
	}

	// start starts a new session with identifier id that must be an integer.
	// If id valuates to false, start uses the time in milliseconds from the
	// epoc in UTC as identifier.
	start(id) {
		if (id == null) {
			id = new Date().getTime();
		}
		this.#storage.setSession(id, true);
	}
}

export default Session;
