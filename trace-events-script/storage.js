const warnMsg = 'Analytics: cannot stringify traits';

class Storage {
	constructor() {
		try {
			localStorage.setItem('test', '');
			localStorage.removeItem('test');
		} catch (_) {
			throw new Error('local storage is not available');
		}
		this.store = globalThis.localStorage;
	}

	anonymousId() {
		return this.store.getItem('chichi_anonymous_id');
	}

	groupId() {
		return this.store.getItem('chichi_group_id');
	}

	session() {
		const session = this.store.getItem('chichi_session');
		if (session == null) {
			return [null, 0, false];
		}
		return JSON.parse(session);
	}

	traits(kind) {
		const traits = this.store.getItem(`chichi_${kind}_traits`);
		if (traits == null) {
			return {};
		}
		return JSON.parse(traits);
	}

	userId() {
		return this.store.getItem('chichi_user_id');
	}

	setAnonymousId(id) {
		if (id == null) {
			this.store.removeItem('chichi_anonymous_id');
			return;
		}
		this.store.setItem('chichi_anonymous_id', id);
	}

	setGroupId(id) {
		if (id == null) {
			this.store.removeItem('chichi_group_id');
			return;
		}
		this.store.setItem('chichi_group_id', id);
	}

	setSession(id, expiration, start) {
		if (id == null) {
			this.store.removeItem('chichi_session');
			return;
		}
		this.store.setItem('chichi_session', JSON.stringify([id, expiration, start]));
	}

	setTraits(kind, traits) {
		if (typeof kind !== 'string') {
			throw new Error('kind is ' + (typeof kind));
		}
		if (traits == null) {
			this.store.removeItem(`chichi_${kind}_traits`);
			return;
		}
		const type = typeof traits;
		if (type !== 'object') {
			console.warn(`${warnMsg}: traits is a ${type}`);
			return;
		}
		if (Array.isArray(traits)) {
			console.warn(`${warnMsg}: ${kind} traits is an array`);
			return;
		}
		let value;
		try {
			value = JSON.stringify(traits);
		} catch (error) {
			console.warn(`${warnMsg}: ${error.message}`);
			return;
		}
		this.store.setItem(`chichi_${kind}_traits`, value);
	}

	setUserId(id) {
		if (id == null) {
			this.store.removeItem('chichi_user_id');
		} else {
			this.store.setItem('chichi_user_id', id);
		}
	}
}

export default Storage;
