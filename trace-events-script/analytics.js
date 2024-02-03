import { campaign, isPlainObject, uuid } from './utils.js';
import Options from './options.js';
import Storage from './storage.js';
import Session from './session.js';
import { Sender } from './sender.js';
import User from './user.js';
import Group from './group.js';

const version = '0.0.0';
const none = () => {};

class Analytics {
	#options;
	#storage;
	#session;
	#sender;
	#isReady = false;
	#onReady;
	#user;
	#group;

	constructor(writeKey, endpoint, options) {
		this.#options = new Options(options);
		this.#storage = new Storage();
		this.#session = new Session(this.#storage, this.#options.sessions.autoTrack, this.#options.sessions.timeout);
		this.#sender = new Sender(writeKey, endpoint);
		this.#user = new User(this.#storage);
		this.#group = new Group(this.#storage);
		const onReady = this.#onReady;
		if (onReady) {
			for (let i = 0; i < onReady.length; i++) {
				setTimeout(onReady[i]());
			}
			this.#onReady = void 0;
		}
		this.#isReady = true;
	}

	// alias sends an alias event.
	alias() {
		return this.#send('alias', this.#setAliasArguments, arguments);
	}

	// debug toggles debug mode.
	debug(on) {
		this.#session.debug(on);
		this.#sender.debug(on);
	}

	endSession() {
		this.#session.end();
	}

	// getAnonymousId returns the current Anonymous ID. If no Anonymous ID
	// exists, it generates one and returns it.
	getAnonymousId() {
		return this.#anonymousId();
	}

	// getSessionId returns the current session ID, or null if there is no
	// session.
	getSessionId() {
		return this.#session.get();
	}

	// group sends a group event, if there is no arguments, it returns the
	// current group as a value with methods 'id', to get the Group ID, and
	// 'traits' to get the traits.
	group() {
		if (arguments.length === 0) {
			return this.#group;
		}
		return this.#send('group', this.#setGroupArguments, arguments);
	}

	// identify sends an identify event.
	identify() {
		return this.#send('identify', this.#setIdentifyArguments, arguments);
	}

	// page sends a page event.
	page() {
		return this.#send('page', this.#setPageScreenArguments, arguments);
	}

	// ready calls callback after Analytics finishes initializing.
	ready(callback) {
		if (typeof callback !== 'function') {
			return;
		}
		if (this.#isReady) {
			setTimeout(callback);
			return;
		}
		this.#onReady = this.#onReady || [];
		this.#onReady.push(callback);
	}

	// reset resets the user and group identifiers, and traits removing them
	// from the storage. It also resets the Anonymous ID by generating a new
	// one, and ends the session if one exists.
	reset() {
		this.#storage.setUserID();
		this.#storage.setGroupID();
		this.#storage.setTraits('user');
		this.#storage.setTraits('group');
		this.#storage.setAnonymousID();
		this.#session.end();
	}

	// screen sends a screen event.
	screen() {
		return this.#send('screen', this.#setPageScreenArguments, arguments);
	}

	// setAnonymousId sets the default Anonymous ID or, if id is undefined,
	// returns the default Anonymous ID.
	setAnonymousId(id) {
		return this.#user.anonymousId(id);
	}

	// startSession starts a new session.
	startSession(id) {
		if (id) {
			if (typeof id !== 'number' || id % 1 !== 0) {
				throw new Error('sessionId must be a positive integer');
			}
		} else {
			id = null;
		}
		this.#session.start(id);
	}

	// track sends a track event.
	track() {
		return this.#send('track', this.#setTrackArguments, arguments);
	}

	// user returns the current user as a value with methods 'id', to get the
	// User ID, 'traits' to get the traits, and 'anonymousId' to get the
	// Anonymous ID.
	user() {
		return this.#user;
	}

	// anonymousId returns the anonymous ID. If the anonymous ID is null, it
	// creates and stores a new generated anonymous ID, then returns it.
	#anonymousId() {
		let id = this.#storage.getAnonymousID();
		if (id == null) {
			id = uuid();
			this.#storage.setAnonymousID(id);
		}
		return id;
	}

	// getAlias returns the userId or previousId arguments of the alias calls.
	#getAlias(id) {
		if ((typeof id === 'string' && id !== '') || typeof id === 'number') {
			return String(id);
		}
		id = this.#storage.getUserID();
		if (id == null) {
			return this.#anonymousId();
		}
		return id;
	}

	// send sends an event of the given type, setting the arguments args with
	// the setArgs function.
	#send(type, setArgs, args) {
		const executor = (resolve, reject) => {
			let event;
			const data = { type };
			// ES5: "Array.from" is not available.
			args = Array.prototype.slice.call(args);
			let callback;
			if (args.length > 0 && typeof args[args.length - 1] === 'function') {
				callback = args.pop();
			}
			try {
				const options = setArgs.call(this, data, args);
				event = this.#sendEvent(data, options);
			} catch (error) {
				reject(error);
				return;
			}
			if (callback) {
				callback({ attempts: 1, event: event });
			}
			resolve({ attempts: 1, event: event });
		};
		if (typeof globalThis.Promise === 'function') {
			return new Promise(executor);
		}
		executor(none, none);
	}

	// sendEvent sends an event with the given options.
	#sendEvent(event, options) {
		if (options && 'timestamp' in options) {
			event.timestamp = options.timestamp;
		} else {
			event.timestamp = new Date();
		}

		event.messageId = uuid();
		event.anonymousId = this.#anonymousId();

		const loc = globalThis.location;

		const canonical = document.querySelector('link[rel="canonical"]');
		let pageURL = canonical ? canonical.getAttribute('href') : null;
		let path;
		if (pageURL == null || pageURL === '') {
			pageURL = loc.href;
			const p = pageURL.indexOf('#');
			if (p !== -1) {
				pageURL = pageURL.substring(0, p);
			}
			path = loc.pathname;
		} else {
			let u = pageURL;
			const p = u.indexOf('#');
			if (p !== -1) {
				u = u.substring(0, p);
			}
			path = u.substring(u.indexOf('/'));
		}

		const page = {
			path: path,
			referrer: document.referrer == null ? '' : document.referrer,
			search: loc.search,
			title: document.title,
			url: pageURL,
		};

		switch (event.type) {
			case 'page': {
				const p = isPlainObject(event.properties) ? event.properties : {};
				for (const k in page) {
					if (k in p) {
						const v = p[k];
						if (typeof v === 'string' && v !== '') {
							page[k] = p[k];
						}
					} else {
						p[k] = page[k];
					}
				}
				if ('category' in event) {
					p.category = event.category;
				}
				if ('name' in event && event.name !== '') {
					p.name = event.name;
				}
				event.properties = p;
				this.#setUserId(event);
				break;
			}
			case 'screen':
				if (!isPlainObject(event.properties)) {
					event.properties = {};
				}
				this.#setUserId(event);
				break;
			case 'track':
			case 'group':
				this.#setUserId(event);
		}

		const n = globalThis.navigator;
		event.context = {
			library: {
				name: 'chichi.js',
				version: version,
			},
			locale: n.language || n.userLanguage,
			page: page,
			screen: {
				width: globalThis.screen.width,
				height: globalThis.screen.height,
				density: globalThis.devicePixelRatio,
			},
			userAgent: n.userAgent,
		};

		const c = campaign();
		if (c) {
			event.context.campaign = c;
		}

		event.integrations = {};
		if (options && typeof options.integrations == 'object') {
			for (const n in options.integrations) {
				event.integrations[n] = options.integrations[n];
			}
		}

		for (const option in options) {
			if (option !== 'integrations' && options[option] !== void 0) {
				event.context[option] = options[option];
			}
		}

		const [sessionId, sessionStart] = this.#session.getFresh();
		if (sessionId != null) {
			event.context.sessionId = sessionId;
			if (sessionStart) {
				event.context.sessionStart = true;
			}
		}
		this.#sender.send(event);

		return event;
	}

	// setAliasArguments sets the arguments for alias calls.
	// It writes the 'userId' and 'previousId' arguments into data and
	// returns the options.
	#setAliasArguments(data, a) {
		if (a.length === 0) {
			throw new Error('User is missing');
		}
		data.userId = this.#getAlias(a.shift());
		let options;
		switch (typesOf(a)) {
			case '':
				break;
			case 'string':
				data.previousId = this.#getAlias(a[0]);
				break;
			case 'object':
				options = a[0];
				break;
			case 'string,object':
				data.previousId = this.#getAlias(a[0]);
				options = a[1];
				break;
			default:
				throw new Error('Invalid arguments');
		}
		return options;
	}

	// setIdentifyArguments sets the arguments for identify calls.
	// It writes the 'userId' and 'traits' arguments into data and
	// returns the options.
	#setIdentifyArguments(data, a) {
		let options;
		switch (typesOf(a)) {
			// ()
			case '':
				this.#setUserId(data);
				this.#setTraits(this.#user, data);
				break;
			// (userId);
			case 'string':
				this.#setUserId(data, a[0]);
				this.#setTraits(this.#user, data);
				break;
			// (traits);
			case 'object':
				this.#setUserId(data);
				this.#setTraits(this.#user, data, a[0]);
				break;
			// (userId, traits);
			case 'string,object':
				this.#setUserId(data, a[0]);
				this.#setTraits(this.#user, data, a[1]);
				break;
			// (traits, context);
			case 'object,object':
				this.#setUserId(data);
				this.#setTraits(this.#user, data, a[0]);
				options = a[1];
				break;
			// (userId, traits, context)
			case 'string,object,object':
				this.#setUserId(data, a[0]);
				this.#setTraits(this.#user, data, a[1]);
				options = a[2];
				break;
			default:
				throw new Error('Invalid arguments');
		}
		return options;
	}

	// setGroup sets the groupId with id.
	#setGroup(data, id) {
		data.groupId = this.#group.id(id !== null ? id : undefined);
	}

	// setGroupArguments sets the arguments for group calls.
	// It writes the 'groupId' and 'traits' arguments into data and
	// returns the options.
	#setGroupArguments(data, a) {
		let options;
		switch (typesOf(a)) {
			// (groupId)
			case 'string':
				this.#setGroup(data, a[0]);
				this.#setTraits(this.#group, data);
				break;
			// (traits)
			case 'object':
				this.#setTraits(this.#group, data, a[0]);
				break;
			// (groupId, traits)
			case 'string,object':
				this.#setGroup(data, a[0]);
				this.#setTraits(this.#group, data, a[1]);
				break;
			// (traits, context)
			case 'object,object':
				this.#setTraits(this.#group, data, a[0]);
				options = a[1];
				break;
			// (groupId, traits, context)
			case 'string,object,object':
				this.#setGroup(data, a[0]);
				this.#setTraits(this.#group, data, a[1]);
				options = a[2];
				break;
			default:
				throw new Error('Invalid arguments');
		}
		return options;
	}

	// setPageScreenArguments sets the arguments for page and screen calls.
	// It writes the 'category', 'name', and 'properties' arguments into data
	// and returns the options.
	#setPageScreenArguments(data, a) {
		let options;
		switch (typesOf(a)) {
			// ()
			case '':
				break;
			// (name)
			case 'string':
				data.name = a[0];
				break;
			// (properties)
			case 'object':
				data.properties = a[0];
				break;
			// (category, name)
			case 'string,string':
				data.category = a[0];
				data.name = a[1];
				break;
			// (name, properties)
			case 'string,object':
				data.name = a[0];
				data.properties = a[1];
				break;
			// (properties, context)
			case 'object,object':
				data.properties = a[0];
				options = a[1];
				break;
			// (category, name, properties)
			case 'string,string,object':
				data.category = a[0];
				data.name = a[1];
				data.properties = a[2];
				break;
			// (name, properties, context)
			case 'string,object,object':
				data.name = a[0];
				data.properties = a[1];
				options = a[2];
				break;
			// (category, name, properties, context)
			case 'string,string,object,object':
				data.category = a[0];
				data.name = a[1];
				data.properties = a[2];
				options = a[3];
				break;
			default:
				throw new Error('Invalid arguments');
		}
		return options;
	}

	// setTrackArguments sets the arguments for track calls.
	// It writes the 'event' and 'properties' arguments into data and
	// returns the options.
	#setTrackArguments(data, a) {
		if (a.length === 0 || typeof a[0] != 'string') {
			throw new Error('Event name is missing');
		}
		data.event = a.shift();
		let options;
		switch (typesOf(a)) {
			// (name)
			case '':
				break;
			// (name, properties)
			case 'object':
				data.properties = a[0];
				break;
			// (name, properties, context)
			case 'object,object':
				data.properties = a[0];
				options = a[1];
				break;
			default:
				throw new Error('Invalid arguments');
		}
		return options;
	}

	// setTraits sets the user or group traits, merging the current traits with
	// traits. k must be #user or #group.
	#setTraits(k, data, traits) {
		data.traits = k.traits();
		if (traits !== undefined) {
			for (const k in traits) {
				const v = traits[k];
				if (v === undefined) {
					delete data.traits[k];
				} else {
					data.traits[k] = v;
				}
			}
		}
		k.traits(data.traits);
		data.traits = k.traits();
	}

	// setUserId sets the userId with id.
	#setUserId(data, id) {
		data.userId = this.#user.id(id !== null ? id : undefined);
	}
}

// typesOf returns a string representing the types of the elements of the array
// arr, 'object' for non-null Object values and 'string' for all the other
// values. If arr is empty, it returns an empty string. For example, if arr is
// ['a', null, 5, {}], it returns 'string,object,string,object'.
// If arr is not an array, it throws an error.
function typesOf(arr) {
	return arr.map((v) => typeof v === 'object' && v != null ? 'object' : 'string').join(',');
}

export default Analytics;
