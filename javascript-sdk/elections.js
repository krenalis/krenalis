import { getTime } from './utils.js'

// Elections maintain leader elections. Each candidate should have its own
// instance.
class Elections {
	#id
	#state
	#onElection
	#timeoutID

	// constructor returns an instance of Elections for a candidate identified
	// by id. state implements methods for reading and writing the election
	// state. During each election, the onElection function is invoked with a
	// boolean argument indicating whether the candidate emerged as the leader.
	constructor(id, state, onElection) {
		this.#id = id
		this.#state = state
		this.#onElection = onElection
		this.#timeoutID = setTimeout(this.#keep.bind(this))
	}

	// close closes elections.
	close() {
		clearTimeout(this.#timeoutID)
		this.#timeoutID = null
	}

	// resign resigns as leader.
	resign() {
		this.#getOwner(1, (id) => {
			if (id === this.#id) {
				this.#state.write(this.#id, 1, '', () => this.#onElection(false))
			}
		})
	}

	// keep keeps the elections. It is initially invoked by the constructor and
	// then recursively calls itself.
	#keep() {
		// While debugging tests in Deno, there have been instances where #keep
		// is invoked even after the timeout has been canceled.
		if (this.#timeoutID == null) {
			console.warn('elections.#keep called after closure')
			return
		}
		this.#timeoutID = null
		this.#tryElection((isLeader, expiration) => {
			const interval = expiration - getTime()
			// The next election occurs 1.2 seconds later. The leader retries after 1 second
			// while followers retry between 1.2 and 1.4 seconds.
			const delay = isLeader ? interval - 200 : interval + Math.floor(Math.random() * 200) + 1
			this.#timeoutID = setTimeout(this.#keep.bind(this), delay > 0 ? delay : 0)
			this.#onElection(isLeader)
		})
	}

	// tryElection attempts to elect the current node as the leader. It returns
	// a boolean indicating whether the election was successful and the
	// expiration time, in milliseconds from the epoch, of the latest election.
	// If the storage is out of quota, it raises a QuotaExceededError exception.
	#tryElection(cb) {
		// The implementation utilizes callbacks to facilitate code testing.
		this.#getOwner(1, (id, expiration) => {
			if (id != null && id !== this.#id) {
				cb(false, expiration)
				return
			}
			this.#setOwner(1, this.#id, (newExpiration) => {
				if (id === this.#id) {
					const now = getTime()
					if (now < expiration) {
						cb(true, newExpiration)
						return
					}
				}
				this.#getOwner(2, (id, expiration) => {
					if (id != null && id !== this.#id) {
						cb(false, expiration)
						return
					}
					this.#setOwner(2, this.#id, () => {
						this.#getOwner(1, (id, expiration) => {
							cb(id === this.#id, expiration)
						})
					})
				})
			})
		})
	}

	// getOwner calls the callback with two arguments: the identifier of the
	// owner of the provided location, and the expiration time of the ownership
	// in milliseconds. If the location has no owner, both arguments passed to
	// the callback are undefined.
	#getOwner(location, cb) {
		this.#state.read(this.#id, location, (owner) => {
			if (owner != null) {
				const [id, expiration] = owner.split(' ')
				const now = getTime()
				if (Number(expiration) > now) {
					cb(id, expiration)
					return
				}
			}
			cb()
		})
	}

	// setOwner assigns the candidate with identifier id as the owner of the
	// provided location, and then proceeds to call the callback.
	#setOwner(location, id, cb) {
		const expiration = getTime() + 1200
		this.#state.write(this.#id, location, `${id} ${expiration}`, () => cb(expiration))
	}
}

export default Elections
