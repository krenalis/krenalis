class Storage {
	constructor() {
		try {
			localStorage.setItem('test', '');
			localStorage.removeItem('test');
		} catch (_) {
			throw new Error('local storage is not available');
		}
		this.store = window.localStorage;
	}

	setUserID(id) {
		this.store.setItem('chichi_user_id', id);
	}

	getUserID() {
		return this.store.getItem('chichi_user_id');
	}

	setAnonymousId(id) {
		this.store.setItem('chichi_anonymous_id', id);
	}

	getAnonymousId() {
		return this.store.getItem('chichi_anonymous_id');
	}

	setUserTraits(traits) {
		this.store.setItem('chichi_user_traits', JSON.stringify(traits));
	}
}

export default Storage;
