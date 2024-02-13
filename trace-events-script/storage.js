const noStorageSupported = new Error('no storage supported')
const warnMsg = 'Analytics: cannot stringify traits'

class Storage {
	#store

	constructor(options) {
		const stores = []
		if (globalThis.document?.cookie != null) {
			try {
				stores.push(new cookieStore(options.cookie))
			} catch (error) {
				if (error !== noStorageSupported) {
					throw error
				}
			}
		}
		try {
			stores.push(new storageStore(globalThis.localStorage))
		} catch (error) {
			if (error !== noStorageSupported) {
				throw error
			}
		}
		if (stores.length === 0) {
			throw noStorageSupported
		}
		this.#store = new multipleStore(stores)
	}

	anonymousId() {
		return this.#store.get('chichi_anonymous_id')
	}

	groupId() {
		return this.#store.get('chichi_group_id')
	}

	removeSuspended() {
		this.#store.delete('chichi_suspended')
	}

	restore() {
		let session, anonymousId, userTraits, groupId, groupTraits
		const suspended = this.#store.get('chichi_suspended')
		if (suspended != null) {
			;[session, anonymousId, userTraits, groupId, groupTraits] = JSON.parse(suspended)
		}
		if (session == null) {
			session = [null, 0, false]
		}
		this.setSession(...session)
		this.setAnonymousId(anonymousId)
		this.setTraits('user', userTraits)
		this.setGroupId(groupId)
		this.setTraits('group', groupTraits)
		this.#store.delete('chichi_suspended')
	}

	session() {
		const session = this.#store.get('chichi_session')
		if (session == null) {
			return [null, 0, false]
		}
		return JSON.parse(session)
	}

	traits(kind) {
		const traits = this.#store.get(`chichi_${kind}_traits`)
		if (traits == null) {
			return {}
		}
		return JSON.parse(traits)
	}

	setAnonymousId(id) {
		if (id == null) {
			this.#store.delete('chichi_anonymous_id')
			return
		}
		this.#store.set('chichi_anonymous_id', id)
	}

	setGroupId(id) {
		if (id == null) {
			this.#store.delete('chichi_group_id')
			return
		}
		this.#store.set('chichi_group_id', id)
	}

	setSession(id, expiration, start) {
		if (id == null) {
			this.#store.delete('chichi_session')
			return
		}
		this.#store.set('chichi_session', JSON.stringify([id, expiration, start]))
	}

	setTraits(kind, traits) {
		if (typeof kind !== 'string') {
			throw new Error('kind is ' + (typeof kind))
		}
		if (traits == null) {
			this.#store.delete(`chichi_${kind}_traits`)
			return
		}
		const type = typeof traits
		if (type !== 'object') {
			console.warn(`${warnMsg}: traits is a ${type}`)
			return
		}
		if (Array.isArray(traits)) {
			console.warn(`${warnMsg}: ${kind} traits is an array`)
			return
		}
		let value
		try {
			value = JSON.stringify(traits)
		} catch (error) {
			console.warn(`${warnMsg}: ${error.message}`)
			return
		}
		this.#store.set(`chichi_${kind}_traits`, value)
	}

	setUserId(id) {
		if (id == null) {
			this.#store.delete('chichi_user_id')
		} else {
			this.#store.set('chichi_user_id', id)
		}
	}

	suspend() {
		const session = this.session()
		const anonymousId = this.anonymousId()
		const userTraits = this.traits('user')
		const groupId = this.groupId()
		const groupTraits = this.traits('group')
		const suspended = [session, anonymousId, userTraits, groupId, groupTraits]
		this.#store.set('chichi_suspended', JSON.stringify(suspended))
	}

	userId() {
		return this.#store.get('chichi_user_id')
	}
}

// cookieStore stores key/value pairs in cookies.
class cookieStore {
	#domain
	#maxAge
	#path
	#sameSite
	#secure

	// constructor returns a new cookieStore given the following options:
	//
	// * domain, if not null or empty, specifies the domain to use for cookies.
	//   If it is empty, cookies are restricted to the exact domain where they
	//   were created. If not empty, the cookies' domain will be set to the
	//   smallest subdomain of the page's domain, or possibly the page's domain
	//   itself, where cookie setting is supported.
	//
	// * maxAge is the value in milliseconds used for the 'expires' attribute.
	//
	// * path is the value used in the 'path' attribute.
	//
	// * sameSite determines the value for the 'SameSite' attribute, which can
	//   be set to 'lax', 'strict', or 'none'.
	//
	// * secure, if it is set to true, will add the 'secure' attribute.
	//
	// If cookies are not supported, it raises an exception with the error
	// storeNotSupported.
	constructor(options) {
		this.#domain = options.domain
		this.#maxAge = options.maxAge
		this.#path = options.path
		this.#sameSite = options.sameSite
		this.#secure = options.secure
		this.#setDomain()
	}

	get(key) {
		const s = globalThis.document.cookie
		const cookies = s.length > 0 ? s.split('; ') : []
		for (let i = 0; i < cookies.length; i++) {
			const cookie = cookies[i]
			const p = cookie.indexOf('=')
			if (p === key.length && cookie.substring(0, p) === key) {
				let value = null
				try {
					value = globalThis.decodeURIComponent(cookie.substring(p + 1))
				} catch {
					// value contains an invalid escape sequence.
				}
				return value
			}
		}
		return null
	}

	set(key, value) {
		try {
			value = globalThis.encodeURIComponent(value)
		} catch {
			// value contains a lone surrogate.
			return null
		}
		const expires = new Date(Date.now() + this.#maxAge).toUTCString()
		globalThis.document.cookie = `${key}=${value}; expires=${expires}; path=${this.#path}; samesite=${this.#sameSite}` +
			`${this.#secure ? '; secure' : ''}${this.#domain === '' ? '' : `; domain=${this.#domain}`}`
	}

	delete(key) {
		document.cookie = `${key}=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=${this.#path}${
			this.#domain === '' ? '' : `; domain=${this.#domain}`
		}`
	}

	// setDomain sets the domain to use for cookies. It is the smallest
	// subdomain of the page's domain, or possibly the page's domain itself,
	// for which cookie setting is supported. If cookie setting is not
	// supported, it raises an exception with the error storeNotSupported.
	#setDomain() {
		const hostnames = () => {
			if (this.#domain != null) {
				return [this.#domain]
			}
			const hostname = globalThis.location.hostname
			const components = hostname.split('.')
			// Note that if the domain ends with a dot, it should be left as is because some browsers,
			// such as Chrome and Firefox, treat domains with and without dots as distinct.
			if (components.length < 3) {
				return [hostname] // top-level, second-level domain, or IPv6
			}
			const c = components[0][0]
			if ('0' <= c && c <= '9') {
				return [hostname] // IPv4
			}
			const names = []
			for (let i = 2; i < components.length + 1; i++) {
				names.push(components.slice(-i).join('.'))
			}
			return names
		}
		const domains = hostnames()
		const key = '__test__'
		const value = String(Math.floor(Math.random() * 100000000))
		for (let i = 0; i < domains.length; i++) {
			this.#domain = domains[i]
			this.set(key, value)
			if (this.get(key) === value) {
				this.delete(key)
				return
			}
		}
		throw noStorageSupported
	}
}

// storageStore stores key/value pairs in a Storage.
class storageStore {
	#storage

	// constructor returns a new storageStore based on the provided Storage,
	// such as localStorage or sessionStorage. If the provided Storage cannot be
	// used, it raises an exception with the storeNotSupported error.
	constructor(storage) {
		try {
			storage.setItem('__test__', '')
			storage.removeItem('__test__')
		} catch {
			throw noStorageSupported
		}
		this.#storage = storage
	}
	get(key) {
		try {
			return this.#storage.getItem(key)
		} catch {
			return null
		}
	}
	set(key, value) {
		try {
			this.#storage.setItem(key, value)
		} catch {
			// Nothing to do.
		}
	}
	delete(key) {
		try {
			this.#storage.removeItem(key)
		} catch {
			// Nothing to do.
		}
	}
}

// multipleStore stores key/value pairs across multiple stores. The get method
// retrieves the key from the first store, the set method updates the key in all
// stores, and the delete method removes the key from all stores.
class multipleStore {
	#stores
	// constructor returns a new multipleStore that stores key/value pairs in
	// the provided stores.
	constructor(stores) {
		this.#stores = stores
	}
	get(key) {
		let value = null
		for (let i = 0; i < this.#stores.length; i++) {
			value = this.#stores[i].get(key)
			if (value != null) {
				break
			}
		}
		return value
	}
	set(key, value) {
		for (let i = 0; i < this.#stores.length; i++) {
			this.#stores[i].set(key, value)
		}
	}
	delete(key) {
		for (let i = 0; i < this.#stores.length; i++) {
			this.#stores[i].delete(key)
		}
	}
}

export default Storage
export { cookieStore, multipleStore, storageStore }
