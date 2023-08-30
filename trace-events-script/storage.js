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

	getSession() {
		let id = this.store.getItem('chichi_session_id');
		if (id != null) {
			id = Number(id);
		}
		const start = this.store.getItem('chichi_session_start') === 'true';
		return [id, start];
	}

	getTraits() {
		const traits = this.store.getItem('chichi_traits');
		if (traits == null) {
			return {};
		}
		return JSON.parse(traits);
	}

	getUserID() {
		return this.store.getItem('chichi_user_id');
	}

	reset() {
		this.store.removeItem('chichi_group_id');
		this.store.removeItem('chichi_traits');
		this.store.removeItem('chichi_user_id');
	}

	setAnonymousID(id) {
		this.store.setItem('chichi_anonymous_id', id);
	}

	setGroupID(id) {
		this.store.setItem('chichi_group_id', id);
	}

	setSession(id, start) {
		if (id == null) {
			this.store.removeItem('chichi_session_id');
			this.store.removeItem('chichi_session_start');
			return;
		}
		this.store.setItem('chichi_session_id', id);
		this.store.setItem('chichi_session_start', start);
	}

	setTraits(traits) {
		try {
			this.store.setItem('chichi_traits', JSON.stringify(traits));
		} catch (_) {}
	}

	setUserID(id) {
		this.store.setItem('chichi_user_id', id);
	}
}

export default Storage;
