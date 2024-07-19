import { debug, getTime } from './utils.js'

class Session {
	#storage
	#autoTrack
	#timeout
	#debug

	constructor(storage, autoTrack, timeout, debug) {
		this.#storage = storage
		this.#autoTrack = autoTrack
		this.#timeout = timeout
		this.debug(debug)
		if (autoTrack) {
			const [id, expiration] = storage.session()
			const now = getTime()
			if (id == null || expiration < now) {
				this.#debug?.('start session', now, 'with timeout', timeout, 'ms ( there was no session )')
				storage.setSession(now, now + timeout, true)
			}
		}
	}

	// debug toggles debug mode.
	debug(on) {
		this.#debug = debug(on)
	}

	// end ends the current session.
	end() {
		if (this.#debug) {
			const [id] = this.#storage.session()
			if (id != null) {
				this.#debug('end session', id)
			}
		}
		this.#storage.setSession()
	}

	// getFresh returns the current session and a boolean value reporting
	// whether a new session has been started since the last call to getFresh.
	// It also extends the expiration of the current session.
	//
	// If no session exists:
	//   - if autoTrack is true, it starts a new session and then returns it.
	//   - if autoTrack is false, it returns null.
	getFresh() {
		let [id, expiration, start] = this.#storage.session()
		const now = getTime()
		if (this.#autoTrack) {
			if (id == null || expiration < now) {
				if (id == null) {
					this.#debug?.('start session', now, 'with timeout', this.#timeout, 'ms ( there was no session )')
				} else {
					this.#debug?.(
						'start session',
						now,
						'with timeout',
						this.#timeout,
						'ms ( previous session is expired',
						now - expiration,
						'ms ago)',
					)
				}
				id = now
				start = true
			}
		}
		if (id != null) {
			const expiration = now + this.#timeout
			this.#storage.setSession(id, expiration, false)
		}
		return [id, start]
	}

	// get returns the current session, or null if no session exist.
	get() {
		let [id, expiration] = this.#storage.session()
		if (id != null && this.#autoTrack) {
			const now = getTime()
			if (expiration < now) {
				id = null
			}
		}
		return id
	}

	// reset ends the current session and starts a new one if a session exists.
	// If there is no active session, it does nothing.
	reset() {
		const [id, _] = this.#storage.session()
		if (id != null) {
			this.start()
		}
	}

	// start starts a new session with identifier id that must be an integer. If
	// id valuates to false, start uses the time in milliseconds from the epoch
	// in UTC as identifier.
	start(id) {
		const now = getTime()
		if (id == null) {
			id = getTime()
		}
		const expiration = now + this.#timeout
		this.#debug?.('start session', id, 'with timeout', this.#timeout, 'ms ( there was no session )')
		this.#storage.setSession(id, expiration, true)
	}
}

export default Session
