import { campaign, debug, isPlainObject, isURL, onVisibilityChange, parseQueryString, uuid } from './utils.js'
import Elections from './elections.js'
import Group from './group.js'
import Options from './options.js'
import Queue, { ItemTooLargeError } from './queue.js'
import Sender from './sender.js'
import Session from './session.js'
import Storage from './storage.js'
import User from './user.js'

const version = '0.0.0'
const none = () => {}

// maxEventSize is the maximum size, in bytes, for a single JSON serialized
// event. Events exceeding this size will not be appended to the queue.
const maxEventSize = 32 * 1024

// queueKeyReg is the regexp for storage queue keys.
const queueKeyReg = /^chichi\.[a-zA-Z0-9]{7}\.[a-zA-Z0-9]{8}(?:-[a-zA-Z0-9]{4}){3}-[a-zA-Z0-9]{12}\.queue$/

class EndpointURLError extends Error {
	constructor(endpoint) {
		super(`The endpoint URL '${endpoint}' is not valid.`)
		this.name = this.constructor.name
	}
}

class Analytics {
	#id
	#writeKey
	#endpoint
	#ready
	#options
	#storage
	#session
	#keysPrefix
	#queue
	#sender
	#user
	#group
	#elections
	#isLeader = false
	#onFollowerQueue = null // is null when it is not the leader
	#debug

	// constructor returns a new Analytics instance. writeKey is the write key,
	// endpoint denotes the endpoint URL, and options is an object containing
	// the options, if any. If endpoint is not a valid URL, it raises an
	// EndpointURLError error.
	constructor(writeKey, endpoint, options) {
		if (!isURL(endpoint)) {
			throw new EndpointURLError(endpoint)
		}
		if (endpoint.slice(-1) !== '/') {
			endpoint += '/'
		}

		this.#id = uuid()
		this.#writeKey = writeKey
		this.#endpoint = endpoint
		this.#ready = new Ready()
		this.#options = new Options(writeKey, endpoint, options, (error) => {
			if (this.#ready) {
				this.#ready.emit(error)
				if (error == null && this.#options.useQueryString) {
					this.#processQuerystring()
				}
			}
		})
		this.#storage = new Storage(writeKey, this.#options.storage)
		this.#session = new Session(
			this.#storage,
			this.#options.sessions.autoTrack,
			this.#options.sessions.timeout,
			this.#options.debug,
		)
		this.#keysPrefix = `chichi.${writeKey.slice(0, 7)}`
		this.#queue = new Queue(localStorage, `${this.#keysPrefix}.*.queue`, maxEventSize)
		this.#queue.debug(this.#options.debug)
		this.#user = new User(this.#storage)
		this.#group = new Group(this.#storage)

		// Participate in leader elections.
		const electionsState = new ElectionsState(this.#keysPrefix)
		this.#elections = new Elections(this.#id, electionsState, this.#onElection.bind(this))

		onVisibilityChange((visible) => {
			if (!visible) {
				this.#queue.save()
				if (this.#isLeader) {
					this.#sender.flush()
					this.#elections.resign()
				}
			}
		})

		this.debug(this.#options.debug)
	}

	// alias sends an alias event.
	alias() {
		return this.#send('alias', this.#setAliasArguments, arguments)
	}

	// anonymize sends an anonymize event, anonymizes the user's identity by
	// removing the User ID, and updates or removes the Anonymous ID and traits
	// according to the strategy.
	anonymize() {
		const event = this.#send('anonymize', this.#setAnonymizeArguments, arguments)
		if (this.#options.strategy === 'AC-B') {
			this.#storage.setUserId()
			this.#storage.restore()
		} else {
			this.#reset(this.#options.strategy.indexOf('-C') > 0)
		}
		return event
	}

	// close closes the Analytics instance.
	// It tries to preserve the queue in the localStorage before returning.
	close() {
		this.#elections.close()
		if (this.#isLeader) {
			this.#sender.close()
		}
		this.#queue.close()
		this.#ready.close()
		this.#ready = null
		this.#debug?.('Analytics closed')
	}

	// debug toggles debug mode.
	debug(on) {
		this.#debug = debug(on)
		this.#ready.debug(on)
		this.#session.debug(on)
		this.#queue.debug(on)
		if (this.#isLeader) {
			this.#sender.debug(on)
		}
	}

	// endSession ends the session.
	// If there is no session, it does nothing.
	endSession() {
		this.#session.end()
	}

	// getAnonymousId returns the current Anonymous ID. If no Anonymous ID
	// exists, it generates one and returns it.
	getAnonymousId() {
		return this.#user.anonymousId()
	}

	// getSessionId returns the current session ID, or null if there is no
	// session.
	getSessionId() {
		return this.#session.get()
	}

	// group sends a group event, if there is no arguments, it returns the
	// current group as a value with methods 'id', to get the Group ID, and
	// 'traits' to get the traits.
	group() {
		if (arguments.length === 0) {
			return this.#group
		}
		return this.#send('group', this.#setGroupArguments, arguments)
	}

	// identify sends an identify event.
	identify() {
		return this.#send('identify', this.#setIdentifyArguments, arguments)
	}

	// page sends a page event.
	page() {
		return this.#send('page', this.#setPageScreenArguments, arguments)
	}

	// ready calls callback, if not null, after Analytics finishes initializing.
	// If promises are supported, it also returns a promise.
	ready(callback) {
		if (callback != null) {
			this.#ready.addListener(callback)
		}
		if (typeof globalThis.Promise === 'function') {
			return new Promise((resolve, reject) => {
				this.#ready.addListener(resolve, reject)
			})
		}
	}

	// reset resets the user and group identifiers, and traits removing them
	// from the storage. It also resets the Anonymous ID by generating a new
	// one, and ends the session if one exists.
	reset() {
		this.#reset(true)
	}

	// screen sends a screen event.
	screen() {
		return this.#send('screen', this.#setPageScreenArguments, arguments)
	}

	// setAnonymousId sets the default Anonymous ID or, if id is undefined,
	// returns the default Anonymous ID.
	setAnonymousId(id) {
		return this.#user.anonymousId(id)
	}

	// startSession starts a new session.
	startSession(id) {
		if (id) {
			if (typeof id !== 'number' || id % 1 !== 0) {
				throw new Error('sessionId must be a positive integer')
			}
		} else {
			id = null
		}
		this.#session.start(id)
	}

	// track sends a track event.
	track() {
		return this.#send('track', this.#setTrackArguments, arguments)
	}

	// user returns the current user as a value with methods 'id', to get the
	// User ID, 'traits' to get the traits, and 'anonymousId' to get the
	// Anonymous ID.
	user() {
		return this.#user
	}

	// getAlias returns the userId or previousId arguments of the alias calls.
	#getAlias(id) {
		if ((typeof id === 'string' && id !== '') || typeof id === 'number') {
			return String(id)
		}
		id = this.#storage.userId()
		if (id == null) {
			return this.#user.anonymousId()
		}
		return id
	}

	// onElection is called when an election occurs. isLeader reports whether it
	// is the leader.
	#onElection(isLeader) {
		if (isLeader === this.#isLeader) {
			return
		}
		this.#isLeader = isLeader
		if (!isLeader) {
			this.#debug?.(`there is another leader`)
			this.#sender.close()
			this.#sender = null
			this.#queue.setKey(`${this.#keysPrefix}.*.queue`)
			removeEventListener('storage', this.#onFollowerQueue)
			this.#onFollowerQueue = null
			return
		}
		this.#debug?.(`elected as leader`)
		this.#queue.setKey(`${this.#keysPrefix}.${this.#id}.queue`)
		// Listen to new follower queues to merge.
		const merged = new Set()
		this.#onFollowerQueue = (event) => {
			if (this.#isLeader && event.storageArea === localStorage && event.newValue != null) {
				const key = event.key
				if (!merged.has(key) && queueKeyReg.test(key)) {
					this.#queue.load(key)
					localStorage.removeItem(key)
				}
			}
		}
		addEventListener('storage', this.#onFollowerQueue)
		// Merge existing follower queues.
		for (let i = 0; i < localStorage.length; i++) {
			const key = localStorage.key(i)
			if (queueKeyReg.test(key)) {
				this.#queue.load(key)
				merged.add(key)
			}
		}
		if (merged.size > 0) {
			this.#queue.save()
			for (const key of merged) {
				localStorage.removeItem(key)
			}
		}
		this.#sender = new Sender(this.#writeKey, this.#endpoint, this.#queue)
		if (this.#debug != null) {
			this.#sender.debug(true)
		}
	}

	// processQuerystring processes the query string according to the
	// Querystring API.
	#processQuerystring() {
		const qs = parseQueryString(globalThis.location.search, 'ajs_')
		if (qs.size > 0) {
			const extract = (prefix) => {
				const props = {}
				for (const [k, v] of qs) {
					// ES5: "startsWith" is not available.
					if (k.indexOf(prefix) === 0) {
						props[k.substring(prefix.length)] = v
					}
				}
				return props
			}
			const aid = qs.get('aid')
			if (aid) {
				this.#user.anonymousId(aid)
			}
			const uid = qs.get('uid')
			if (uid) {
				void this.#send('identify', this.#setIdentifyArguments, [uid, extract('trait_')])
			}
			const event = qs.get('event')
			if (event) {
				void this.#send('track', this.#setTrackArguments, [event, extract('prop_')])
			}
		}
	}

	// reset is like the public reset method, but it differs in that it does not
	// reset the Anonymous ID and does not end the session if 'all' is false.
	#reset(all) {
		this.#storage.setUserId()
		this.#storage.setGroupId()
		this.#storage.setTraits('user')
		this.#storage.setTraits('group')
		this.#storage.removeSuspended()
		if (all) {
			this.#storage.setAnonymousId()
			this.#session.end()
		}
	}

	// send sends an event of the given type, setting the arguments args with
	// the setArgs function.
	#send(type, setArgs, args) {
		const executor = (resolve, reject) => {
			let event
			const data = { type }
			// ES5: "Array.from" is not available.
			args = Array.prototype.slice.call(args)
			let callback
			if (args.length > 0 && typeof args[args.length - 1] === 'function') {
				callback = args.pop()
			}
			try {
				const options = setArgs.call(this, data, args)
				event = this.#sendEvent(data, options)
			} catch (error) {
				reject(error)
				return
			}
			if (callback) {
				callback({ attempts: 1, event: event })
			}
			resolve({ attempts: 1, event: event })
		}
		if (typeof globalThis.Promise === 'function') {
			return new Promise(executor)
		}
		executor(none, none)
	}

	// sendEvent sends an event with the given options.
	#sendEvent(event, options) {
		if (options && 'timestamp' in options) {
			event.timestamp = options.timestamp
		} else {
			event.timestamp = new Date()
		}

		const loc = globalThis.location

		let url
		let path
		{
			const canonical = document.querySelector('link[rel="canonical"]')
			if (canonical == null) {
				url = loc.href
				path = loc.pathname
			} else {
				url = canonical.href
				// IE11 does not support URL.
				if (typeof globalThis.URL === 'function') {
					path = new URL(url).pathname
				} else {
					const a = document.createElement('a')
					a.href = url
					path = a.pathname !== '' ? a.pathname : '/'
				}
			}
			const p = url.indexOf('#')
			if (p !== -1) {
				url = url.substring(0, p)
			}
		}

		const page = {
			path: path,
			referrer: document.referrer == null ? '' : document.referrer,
			search: loc.search,
			title: document.title,
			url: url,
		}

		switch (event.type) {
			case 'page': {
				const p = isPlainObject(event.properties) ? event.properties : {}
				for (const k in page) {
					if (k in p) {
						const v = p[k]
						if (typeof v === 'string' && v !== '') {
							page[k] = v
						}
					} else {
						p[k] = page[k]
					}
				}
				if ('category' in event) {
					p.category = event.category
				}
				if ('name' in event && event.name !== '') {
					p.name = event.name
				}
				event.properties = p
				this.#setUserId(event)
				break
			}
			case 'screen':
			case 'track':
				if (!isPlainObject(event.properties)) {
					event.properties = {}
				}
				/* fallthrough */
			case 'group':
				this.#setUserId(event)
				break
			case 'identify':
				if (this.#options.strategy.indexOf('-B') > 0) {
					if (this.#options.strategy === 'AC-B') {
						this.#storage.suspend()
					} else {
						this.#storage.removeSuspended()
					}
					this.#storage.setAnonymousId()
					this.#storage.setTraits('user', event.traits)
					this.#storage.setGroupId()
					this.#storage.setTraits('group')
					this.#session.end()
				} else {
					this.#mergeTraits(this.#user, event, event.traits)
				}
				break
			case 'anonymize':
				event.userId = null
		}

		event.messageId = uuid()
		event.anonymousId = this.#user.anonymousId()

		const n = navigator
		event.context = {
			library: {
				name: 'chichi.js',
				version: version,
			},
			// According to caniuse.com, IE11 does not support the 'language' property but does support 'userLanguage'.
			// However, empirical testing suggests that IE11 supports both.
			locale: n.language || n.userLanguage,
			page: page,
			screen: {
				width: globalThis.screen.width,
				height: globalThis.screen.height,
				density: globalThis.devicePixelRatio,
			},
			userAgent: n.userAgent,
		}

		const c = campaign()
		if (c.size > 0) {
			event.context.campaign = {}
			for (const [k, v] of c) {
				if (k !== '') {
					event.context.campaign[k] = v
				}
			}
		}

		event.integrations = {}
		if (options && typeof options.integrations == 'object') {
			for (const n in options.integrations) {
				event.integrations[n] = options.integrations[n]
			}
		}

		for (const option in options) {
			if (option !== 'integrations' && options[option] !== void 0) {
				event.context[option] = options[option]
			}
		}

		const [sessionId, sessionStart] = this.#session.getFresh()
		if (sessionId != null) {
			event.context.sessionId = sessionId
			if (sessionStart) {
				event.context.sessionStart = true
			}
		}

		try {
			this.#queue.append(event)
		} catch (error) {
			if (error instanceof TypeError) {
				console.warn('cannot stringify the event to JSON:', error)
			} else if (error instanceof ItemTooLargeError) {
				console.warn('event size (' + error.size + 'bytes) is greater then 32KB')
			} else {
				throw error
			}
		}

		return event
	}

	// setAliasArguments sets the arguments for alias calls.
	// It writes the 'userId' and 'previousId' arguments into data and
	// returns the options.
	#setAliasArguments(data, a) {
		if (a.length === 0) {
			throw new Error('User is missing')
		}
		data.userId = this.#getAlias(a.shift())
		let options
		switch (typesOf(a)) {
			// (userId)
			case '':
				break
			// (userId, previousId)
			case 'string':
				data.previousId = this.#getAlias(a[0])
				break
			// (userId, options)
			case 'object':
				options = a[0]
				break
			// (userId, previousId, options)
			case 'string,object':
				data.previousId = this.#getAlias(a[0])
				options = a[1]
				break
			default:
				throw new Error('Invalid arguments')
		}
		return options
	}

	// setAnonymizeArguments sets the arguments for anonymize calls.
	#setAnonymizeArguments(_data, a) {
		if (a.length > 0) {
			throw new Error('Invalid arguments')
		}
	}

	// setIdentifyArguments sets the arguments for identify calls.
	// It writes the 'userId' and 'traits' arguments into data and
	// returns the options.
	#setIdentifyArguments(data, a) {
		let options
		switch (typesOf(a)) {
			// ()
			case '':
				this.#setUserId(data)
				break
			// (userId)
			case 'string':
				this.#setUserId(data, a[0])
				break
			// (traits)
			case 'object':
				this.#setUserId(data)
				data.traits = a[0]
				break
			// (userId, traits)
			case 'string,object':
				this.#setUserId(data, a[0])
				data.traits = a[1]
				break
			// (traits, options)
			case 'object,object':
				this.#setUserId(data)
				data.traits = a[0]
				options = a[1]
				break
			// (userId, traits, options)
			case 'string,object,object':
				this.#setUserId(data, a[0])
				data.traits = a[1]
				options = a[2]
				break
			default:
				throw new Error('Invalid arguments')
		}
		return options
	}

	// setGroup sets the groupId with id.
	#setGroup(data, id) {
		data.groupId = this.#group.id(id !== null ? id : undefined)
	}

	// setGroupArguments sets the arguments for group calls.
	// It writes the 'groupId' and 'traits' arguments into data and
	// returns the options.
	#setGroupArguments(data, a) {
		let options
		switch (typesOf(a)) {
			// (groupId)
			case 'string':
				this.#setGroup(data, a[0])
				this.#mergeTraits(this.#group, data)
				break
			// (traits)
			case 'object':
				this.#mergeTraits(this.#group, data, a[0])
				break
			// (groupId, traits)
			case 'string,object':
				this.#setGroup(data, a[0])
				this.#mergeTraits(this.#group, data, a[1])
				break
			// (traits, options)
			case 'object,object':
				this.#mergeTraits(this.#group, data, a[0])
				options = a[1]
				break
			// (groupId, traits, options)
			case 'string,object,object':
				this.#setGroup(data, a[0])
				this.#mergeTraits(this.#group, data, a[1])
				options = a[2]
				break
			default:
				throw new Error('Invalid arguments')
		}
		return options
	}

	// setPageScreenArguments sets the arguments for page and screen calls.
	// It writes the 'category', 'name', and 'properties' arguments into data
	// and returns the options.
	#setPageScreenArguments(data, a) {
		let options
		switch (typesOf(a)) {
			// ()
			case '':
				break
			// (name)
			case 'string':
				data.name = a[0]
				break
			// (properties)
			case 'object':
				data.properties = a[0]
				break
			// (category, name)
			case 'string,string':
				data.category = a[0]
				data.name = a[1]
				break
			// (name, properties)
			case 'string,object':
				data.name = a[0]
				data.properties = a[1]
				break
			// (properties, options)
			case 'object,object':
				data.properties = a[0]
				options = a[1]
				break
			// (category, name, properties)
			case 'string,string,object':
				data.category = a[0]
				data.name = a[1]
				data.properties = a[2]
				break
			// (name, properties, options)
			case 'string,object,object':
				data.name = a[0]
				data.properties = a[1]
				options = a[2]
				break
			// (category, name, properties, options)
			case 'string,string,object,object':
				data.category = a[0]
				data.name = a[1]
				data.properties = a[2]
				options = a[3]
				break
			default:
				throw new Error('Invalid arguments')
		}
		return options
	}

	// setTrackArguments sets the arguments for track calls.
	// It writes the 'event' and 'properties' arguments into data and
	// returns the options.
	#setTrackArguments(data, a) {
		if (a.length === 0 || typeof a[0] != 'string') {
			throw new Error('Event name is missing')
		}
		data.event = a.shift()
		let options
		switch (typesOf(a)) {
			// (name)
			case '':
				break
			// (name, properties)
			case 'object':
				data.properties = a[0]
				break
			// (name, properties, options)
			case 'object,object':
				data.properties = a[0]
				options = a[1]
				break
			default:
				throw new Error('Invalid arguments')
		}
		return options
	}

	// mergeTraits merges the current user or group traits with traits, store
	// them, and assign them to data.traits. k must be #user or #group.
	#mergeTraits(k, data, traits) {
		data.traits = k.traits()
		if (traits !== undefined) {
			for (const k in traits) {
				const v = traits[k]
				if (v === undefined) {
					delete data.traits[k]
				} else {
					data.traits[k] = v
				}
			}
		}
		k.traits(data.traits)
		data.traits = k.traits()
	}

	// setUserId sets the userId with id.
	#setUserId(data, id) {
		data.userId = this.#user.id(id !== null ? id : undefined)
	}
}

// Ready handles the event that is fired when Analytics finishes initializing.
class Ready {
	#emitted = false
	#listeners = []
	#error
	#debug
	addListener(resolve, reject) {
		this.#listeners.push([resolve, reject])
		if (this.#emitted) {
			this.#notify()
		}
	}
	close() {
		if (!this.#emitted) {
			this.#error = new Error('Analytics instance has been closed')
			this.#notify()
		}
	}
	debug(on) {
		this.#debug = debug(on)
	}
	emit(error) {
		this.#emitted = true
		this.#error = error
		this.#debug?.(error == null ? 'analytics is ready' : `analytics cannot be ready due to an error: ${error}`)
		this.#notify()
	}
	#notify() {
		for (let i = 0; i < this.#listeners.length; i++) {
			const [resolve, reject] = this.#listeners[i]
			if (this.#error == null || reject == null) {
				setTimeout(resolve)
			} else {
				setTimeout(reject, 0, this.#error)
			}
		}
		this.#listeners.length = 0
	}
}

// ElectionsState represents the state of the elections.
class ElectionsState {
	#keys
	constructor(prefix) {
		this.#keys = [`${prefix}.leader.beat`, `${prefix}.leader.election`]
	}
	read(_, location, cb) {
		const value = localStorage.getItem(this.#keys[location - 1])
		cb(value)
	}
	write(_, location, value, cb) {
		localStorage.setItem(this.#keys[location - 1], value)
		cb()
	}
}

// typesOf returns a string representing the types of the elements of the array
// arr, 'object' for non-null Object values and 'string' for all the other
// values. If arr is empty, it returns an empty string. For example, if arr is
// ['a', null, 5, {}], it returns 'string,object,string,object'.
// If arr is not an array, it throws an error.
function typesOf(arr) {
	return arr.map((v) => typeof v === 'object' && v != null ? 'object' : 'string').join(',')
}

export default Analytics
export { Analytics, EndpointURLError }
