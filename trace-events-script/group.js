class Group {
	#storage;

	constructor(storage) {
		this.#storage = storage;
	}

	id(id) {
		if (id === null) {
			this.#storage.setGroupID();
			return null;
		}
		if ((typeof id === 'string' && id !== '') || typeof id === 'number') {
			id = String(id);
			this.#storage.setGroupID(id);
			return id;
		}
		return this.#storage.getGroupID();
	}

	traits(traits) {
		if (traits !== undefined) {
			if (traits === null) {
				traits = {};
			}
			this.#storage.setTraits('group', traits);
		}
		return this.#storage.getTraits('group');
	}
}

export default Group;
