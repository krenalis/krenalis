import { decodeBase64, encodeBase64 } from './utils.js'

const storageNotSupported = new Error('storage is not supported')
const warnMsg = 'Analytics: cannot stringify traits'

class Storage {
	#key
	#storage

	constructor(writeKey, options) {
		const prefix = `chichi.${writeKey.slice(0, 7)}.`
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
		let storage
		const tryStorage = (newStorage) => {
			try {
				const s = newStorage()
				if (storage == null) {
					storage = s
				} else {
					storage = new multiStorage([storage, s])
				}
			} catch (error) {
				if (error !== storageNotSupported) {
					throw error
				}
			}
		}
		switch (options.storage.type) {
			case 'multiStorage':
			case 'cookieStorage':
				tryStorage(() => new cookieStorage(options.cookie))
				if (options.storage.type === 'multiStorage' || !storage) {
					tryStorage(() => new webStorage(localStorage))
					if (!storage) {
						tryStorage(() => new webStorage(sessionStorage))
					}
				}
				break
			case 'localStorage':
				tryStorage(() => new webStorage(localStorage))
				break
			case 'sessionStorage':
				tryStorage(() => new webStorage(sessionStorage))
				break
			case 'none':
				storage = new noStorage()
		}
		if (storage) {
			if (options.storage.type !== 'none') {
				storage = new base64Storage(storage)
			}
		} else {
			storage = new memoryStorage()
		}
		this.#storage = storage
	}

	anonymousId() {
		return this.#storage.get(this.#key.anonymousId)
	}

	groupId() {
		return this.#storage.get(this.#key.groupId)
	}

	removeSuspended() {
		this.#storage.delete(this.#key.suspended)
	}

	restore() {
		let session, anonymousId, userTraits, groupId, groupTraits
		const suspended = this.#storage.get(this.#key.suspended)
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
		this.#storage.delete(this.#key.suspended)
	}

	session() {
		const session = this.#storage.get(this.#key.session)
		if (session == null) {
			return [null, 0, false]
		}
		return JSON.parse(session)
	}

	traits(kind) {
		const traits = this.#storage.get(this.#key.traits[kind])
		if (traits == null) {
			return {}
		}
		return JSON.parse(traits)
	}

	setAnonymousId(id) {
		if (id == null) {
			this.#storage.delete(this.#key.anonymousId)
			return
		}
		this.#storage.set(this.#key.anonymousId, id)
	}

	setGroupId(id) {
		if (id == null) {
			this.#storage.delete(this.#key.groupId)
			return
		}
		this.#storage.set(this.#key.groupId, id)
	}

	setSession(id, expiration, start) {
		if (id == null) {
			this.#storage.delete(this.#key.session)
			return
		}
		this.#storage.set(this.#key.session, JSON.stringify([id, expiration, start]))
	}

	setTraits(kind, traits) {
		if (typeof kind !== 'string') {
			throw new Error('kind is ' + (typeof kind))
		}
		if (traits == null) {
			this.#storage.delete(this.#key.traits[kind])
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
		this.#storage.set(this.#key.traits[kind], value)
	}

	setUserId(id) {
		if (id == null) {
			this.#storage.delete(this.#key.userId)
		} else {
			this.#storage.set(this.#key.userId, id)
		}
	}

	suspend() {
		const session = this.session()
		const anonymousId = this.anonymousId()
		const userTraits = this.traits('user')
		const groupId = this.groupId()
		const groupTraits = this.traits('group')
		const suspended = [session, anonymousId, userTraits, groupId, groupTraits]
		this.#storage.set(this.#key.suspended, JSON.stringify(suspended))
	}

	userId() {
		return this.#storage.get(this.#key.userId)
	}
}

// base64Storage is a storage that stores the key/value pairs in another storage
// encoding and decoding the values in base64.
class base64Storage {
	#storage
	constructor(storage) {
		this.#storage = storage
	}
	get(key) {
		let value = this.#storage.get(key)
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
		this.#storage.set(key, encodeBase64(value))
	}
	delete(key) {
		this.#storage.delete(key)
	}
}

// cookieStorage stores key/value pairs in cookies.
class cookieStorage {
	#domain
	#maxAge
	#path
	#sameSite
	#secure

	// constructor returns a new cookieStorage given the following options:
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
			throw storageNotSupported
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
		throw storageNotSupported
	}
}

// memoryStorage stores key/value pairs in memory.
class memoryStorage {
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

// multiStorage stores key/value pairs across multiple storages. The get
// method retrieves the key from the first storage, the set method updates the
// key in all storages, and the delete method removes the key from all storages.
class multiStorage {
	#storages
	// constructor returns a new multiStorage that stores key/value pairs in
	// the provided storages.
	constructor(storages) {
		this.#storages = storages
	}
	get(key) {
		let value = null
		for (let i = 0; i < this.#storages.length; i++) {
			value = this.#storages[i].get(key)
			if (value != null) {
				break
			}
		}
		return value
	}
	set(key, value) {
		for (let i = 0; i < this.#storages.length; i++) {
			this.#storages[i].set(key, value)
		}
	}
	delete(key) {
		for (let i = 0; i < this.#storages.length; i++) {
			this.#storages[i].delete(key)
		}
	}
}

// noStorage is a storage that does not store key/value pairs.
class noStorage {
	get() {
		return null
	}
	set() {}
	delete() {}
}

// webStorage stores key/value pairs in a Web Storage.
class webStorage {
	#storage

	// constructor returns a new webStorage based on the provided Web Storage,
	// such as localStorage or sessionStorage. If the provided storage cannot be
	// used, it raises an exception with the storeNotSupported error.
	constructor(storage) {
		try {
			storage.setItem('__test__', '')
			storage.removeItem('__test__')
		} catch {
			throw storageNotSupported
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
export { base64Storage, cookieStorage, memoryStorage, multiStorage, noStorage, webStorage }
