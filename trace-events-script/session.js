import { getTime } from './utils.js';

class Session {
	#autoTrack;
	#storage;
	#timeout;

	constructor(storage, autoTrack, timeout) {
		this.#autoTrack = autoTrack;
		this.#storage = storage;
		this.#timeout = timeout;
		if (autoTrack) {
			const [id, expiration] = storage.getSession();
			const now = getTime();
			if (id == null || expiration < now) {
				storage.setSession(now, now + timeout, true);
			}
		}
	}

	// end ends the current session.
	end() {
		this.#storage.setSession(null);
	}

	// getFresh returns the current session and a boolean value reporting
	// whether a new session has been started since the last call to getFresh.
	// It also extends the expiration of the current session.
	//
	// If no session exists:
	//   - if autoTrack is true, it starts a new session and then returns it.
	//   - if autoTrack is false, it returns null.
	getFresh() {
		let [id, expiration, start] = this.#storage.getSession();
		const now = getTime();
		if (this.#autoTrack) {
			if (id == null || expiration < now) {
				id = now;
				start = true;
			}
		}
		if (id != null) {
			const expiration = now + this.#timeout;
			this.#storage.setSession(id, expiration, false);
		}
		return [id, start];
	}

	// get returns the current session, or null if no session exist.
	get() {
		let [id, expiration] = this.#storage.getSession();
		if (id != null && this.#autoTrack) {
			const now = getTime();
			if (expiration < now) {
				id = null;
			}
		}
		return id;
	}

	// start starts a new session with identifier id that must be an integer. If
	// id valuates to false, start uses the time in milliseconds from the epoch
	// in UTC as identifier.
	start(id) {
		const now = getTime();
		if (id == null) {
			id = getTime();
		}
		const expiration = now + this.#timeout;
		this.#storage.setSession(id, expiration, true);
	}
}

export default Session;
