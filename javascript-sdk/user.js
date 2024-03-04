import { uuid } from './utils.js'

class User {
	#storage

	constructor(storage) {
		this.#storage = storage
	}

	id(id) {
		if (id === null) {
			this.#storage.setUserId()
			return null
		}
		if ((typeof id === 'string' && id !== '') || typeof id === 'number') {
			id = String(id)
			const previousId = this.#storage.userId()
			if (id !== previousId) {
				this.#storage.setUserId(id)
				if (previousId != null) {
					this.#storage.setTraits('user')
					this.#storage.setAnonymousId(uuid())
				}
			}
			return id
		}
		return this.#storage.userId()
	}

	anonymousId(id) {
		if (id === undefined) {
			id = this.#storage.anonymousId()
			if (id === null) {
				id = uuid()
			}
		} else if (typeof id === 'number') {
			id = String(id)
		}
		if (typeof id === 'string' && id !== '') {
			this.#storage.setAnonymousId(id)
		} else {
			this.#storage.setAnonymousId(uuid())
		}
		return id
	}

	traits(traits) {
		if (traits !== undefined) {
			if (traits === null) {
				traits = {}
			}
			this.#storage.setTraits('user', traits)
		}
		return this.#storage.traits('user')
	}
}

export default User
