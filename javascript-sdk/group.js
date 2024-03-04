class Group {
	#storage

	constructor(storage) {
		this.#storage = storage
	}

	id(id) {
		if (id === null) {
			this.#storage.setGroupId()
			return null
		}
		if ((typeof id === 'string' && id !== '') || typeof id === 'number') {
			id = String(id)
			this.#storage.setGroupId(id)
			return id
		}
		return this.#storage.groupId()
	}

	traits(traits) {
		if (traits !== undefined) {
			if (traits === null) {
				traits = {}
			}
			this.#storage.setTraits('group', traits)
		}
		return this.#storage.traits('group')
	}
}

export default Group
