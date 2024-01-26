const warnMsg = 'Analytics: cannot stringify traits';

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

	setAnonymousID(id) {
		this.store.setItem('chichi_anonymous_id', id);
	}

	setGroupID(id) {
		if (id == null) {
			this.store.removeItem('chichi_group_id');
			return;
		}
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
		const type = typeof traits;
		if (type !== 'object') {
			console.warn(`${warnMsg}: traits is a ${type}`);
			return;
		}
		if (Array.isArray(traits)) {
			console.warn(`${warnMsg}: traits is an array`);
			return;
		}
		if (traits === null) {
			traits = {};
		}
		let value;
		try {
			value = JSON.stringify(traits);
		} catch (error) {
			console.warn(`${warnMsg}: ${error.message}`);
			return;
		}
		this.store.setItem('chichi_traits', value);
	}

	setUserID(id) {
		if (id == null) {
			this.store.removeItem('chichi_user_id');
		} else {
			this.store.setItem('chichi_user_id', id);
		}
	}
}

export default Storage;
