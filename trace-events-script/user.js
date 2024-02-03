import { uuid } from './utils.js';

class User {
	#storage;

	constructor(storage) {
		this.#storage = storage;
	}

	id(id) {
		if (id === null) {
			this.#storage.setUserID();
			return null;
		}
		if ((typeof id === 'string' && id !== '') || typeof id === 'number') {
			id = String(id);
			const previousId = this.#storage.getUserID();
			if (id !== previousId) {
				this.#storage.setUserID(id);
				if (previousId != null) {
					this.#storage.setTraits('user');
					this.#storage.setAnonymousID(uuid());
				}
			}
			return id;
		}
		return this.#storage.getUserID();
	}

	anonymousId(id) {
		if (id === undefined) {
			id = this.#storage.getAnonymousID();
			if (id === null) {
				id = uuid();
			}
		} else if (typeof id === 'number') {
			id = String(id);
		}
		if (typeof id === 'string' && id !== '') {
			this.#storage.setAnonymousID(id);
		} else {
			this.#storage.setAnonymousID(uuid());
		}
		return id;
	}

	traits(traits) {
		if (traits !== undefined) {
			if (traits === null) {
				traits = {};
			}
			this.#storage.setTraits('user', traits);
		}
		return this.#storage.getTraits('user');
	}
}

export default User;
