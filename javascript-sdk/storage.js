import { decodeBase64, encodeBase64 } from './utils.js'

const storeNotSupported = new Error('store is not supported')
const warnMsg = 'Analytics: cannot stringify traits'

class Storage {
	#key
	#store
	#userStore
	#groupStore

	constructor(writeKey, options) {
		const prefix = `meergo.${writeKey.slice(0, 7)}.`
		this.#key = {
			anonymousId: prefix + 'anonymousId',
			userId: prefix + 'userId',
			groupId: prefix + 'groupId',
			traits: {
				user: prefix + 'userTraits',
				group: prefix + 'groupTraits',
			},
			session: prefix + 'session',
			suspended: prefix + 'suspended',
		}
		this.#store = this.#makeStore(options.stores, options.cookie)
		this.#userStore = options.user.storage.stores == null
			? this.#store
			: this.#makeStore(options.user.storage.stores, options.cookie)
		this.#groupStore = options.group.storage.stores == null
			? this.#store
			: this.#makeStore(options.group.storage.stores, options.cookie)
	}

	anonymousId() {
		return this.#userStore.get(this.#key.anonymousId)
	}

	groupId() {
		return this.#groupStore.get(this.#key.groupId)
	}

	removeSuspended() {
		this.#userStore.delete(this.#key.suspended)
	}

	restore() {
		let session, anonymousId, userTraits, groupId, groupTraits
		const suspended = this.#userStore.get(this.#key.suspended)
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
		this.#userStore.delete(this.#key.suspended)
	}

	session() {
		const session = this.#store.get(this.#key.session)
		if (session == null) {
			return [null, 0, false]
		}
		return JSON.parse(session)
	}

	traits(kind) {
		const store = kind === 'user' ? this.#userStore : this.#groupStore
		const traits = store.get(this.#key.traits[kind])
		if (traits == null) {
			return {}
		}
		return JSON.parse(traits)
	}

	setAnonymousId(id) {
		if (id == null) {
			this.#userStore.delete(this.#key.anonymousId)
			return
		}
		this.#userStore.set(this.#key.anonymousId, id)
	}

	setGroupId(id) {
		if (id == null) {
			this.#groupStore.delete(this.#key.groupId)
			return
		}
		this.#groupStore.set(this.#key.groupId, id)
	}

	setSession(id, expiration, start) {
		if (id == null) {
			this.#store.delete(this.#key.session)
			return
		}
		this.#store.set(this.#key.session, JSON.stringify([id, expiration, start]))
	}

	setTraits(kind, traits) {
		if (typeof kind !== 'string') {
			throw new Error('kind is ' + (typeof kind))
		}
		const store = kind === 'user' ? this.#userStore : this.#groupStore
		if (traits == null) {
			store.delete(this.#key.traits[kind])
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
		this.#store.set(this.#key.traits[kind], value)
	}

	setUserId(id) {
		if (id == null) {
			this.#userStore.delete(this.#key.userId)
		} else {
			this.#userStore.set(this.#key.userId, id)
		}
	}

	suspend() {
		const session = this.session()
		const anonymousId = this.anonymousId()
		const userTraits = this.traits('user')
		const groupId = this.groupId()
		const groupTraits = this.traits('group')
		const suspended = [session, anonymousId, userTraits, groupId, groupTraits]
		this.#userStore.set(this.#key.suspended, JSON.stringify(suspended))
	}

	userId() {
		return this.#userStore.get(this.#key.userId)
	}

	#makeStore(stores, cookie) {
		let store = null
		for (let i = 0; i < stores.length; i++) {
			try {
				let s
				switch (stores[i]) {
					case 'cookie':
						s = new cookieStore(cookie)
						break
					case 'localStorage':
						s = new webStore(localStorage)
						break
					case 'sessionStorage':
						s = new webStore(sessionStorage)
						break
					case 'memory':
						s = new memoryStore()
				}
				if (store == null) {
					store = s
				} else {
					store = new multiStore([store, s])
				}
			} catch (error) {
				if (error !== storeNotSupported) {
					throw error
				}
			}
		}
		if (store == null) {
			return new noStore()
		}
		return new base64Store(store)
	}
}

// base64Store is a store that stores the key/value pairs in another store
// encoding and decoding the values in base64.
class base64Store {
	#store
	constructor(store) {
		this.#store = store
	}
	get(key) {
		let value = this.#store.get(key)
		if (value != null) {
			try {
				value = decodeBase64(value)
			} catch {
				value = null
			}
		}
		return value
	}
	set(key, value) {
		this.#store.set(key, encodeBase64(value))
	}
	delete(key) {
		this.#store.delete(key)
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
		if (document?.cookie == null) {
			// Only in tests.
			throw storeNotSupported
		}
		this.#domain = options.domain
		this.#maxAge = options.maxAge
		this.#path = options.path
		this.#sameSite = options.sameSite
		this.#secure = options.secure
		this.#check()
	}

	get(key) {
		const s = document.cookie
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
		document.cookie = `${key}=${value}; expires=${expires}; path=${this.#path}; samesite=${this.#sameSite}` +
			`${this.#secure ? '; secure' : ''}${this.#domain === '' ? '' : `; domain=${this.#domain}`}`
	}

	delete(key) {
		document.cookie = `${key}=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=${this.#path}; samesite=${this.#sameSite}` +
			`${this.#domain === '' ? '' : `; domain=${this.#domain}`}`
	}

	// check checks whether the cookies are available, and if the domain is
	// null, it determines the domain of the cookies.
	#check() {
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
		throw storeNotSupported
	}
}

// memoryStore stores key/value pairs in memory.
class memoryStore {
	#data = {}
	get(key) {
		const value = this.#data[key]
		return value == null ? null : value
	}
	set(key, value) {
		this.#data[key] = value
	}
	delete(key) {
		delete (this.#data[key])
	}
}

// multiStore stores key/value pairs across multiple stores. The get method
// retrieves the key from the first store, the set method updates the key in
// all stores, and the delete method removes the key from all stores.
class multiStore {
	#stores
	// constructor returns a new multiStore that stores key/value pairs in
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

// noStore is a store that does not store key/value pairs.
class noStore {
	get() {
		return null
	}
	set() {}
	delete() {}
}

// webStore stores key/value pairs in a Web Storage.
class webStore {
	#storage

	// constructor returns a new webStore based on the provided Web Storage,
	// such as localStorage or sessionStorage. If the provided storage cannot be
	// used, it raises an exception with the storeNotSupported error.
	constructor(storage) {
		try {
			storage.setItem('__test__', '')
			storage.removeItem('__test__')
		} catch {
			throw storeNotSupported
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

export default Storage
export { base64Store, cookieStore, memoryStore, multiStore, noStore, webStore }
