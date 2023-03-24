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

	getAnonymousID() {
		return this.store.getItem('chichi_anonymous_id');
	}

	getGroupID() {
		return this.store.getItem('chichi_group_id');
	}

	getUserID() {
		return this.store.getItem('chichi_user_id');
	}

	reset() {
		this.store.getItem('chichi_anonymous_id');
		this.store.removeItem('chichi_group_id');
		this.store.removeItem('chichi_user_id');
	}

	setAnonymousID(id) {
		this.store.setItem('chichi_anonymous_id', id);
	}

	setGroupID(id) {
		this.store.setItem('chichi_group_id', id);
	}

	setUserID(id) {
		this.store.setItem('chichi_user_id', id);
	}
}

export default Storage;
